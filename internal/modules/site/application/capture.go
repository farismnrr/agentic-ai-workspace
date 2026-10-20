package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

// marker starts the line the capture command prints its result on.
const marker = "CAPTURE_JSON:"

// CaptureRequest is what the runtime needs to photograph a page.
type CaptureRequest struct {
	ProjectID string
	// ViewSocket and Prefix say how to reach the app as the person does; without a viewer the browser reaches the
	// app's own port directly.
	ViewSocket string
	Prefix     string
	Spec       domain.CaptureSpec
	Port       int
}

// Capture opens one page of the project's running app in a browser inside the sandbox and returns a picture of
// it and what the page says about itself.
func (s *Service) Capture(ctx context.Context, projectID string, spec domain.CaptureSpec) (domain.Capture, error) {
	spec, err := spec.Normalize()
	if err != nil {
		return domain.Capture{}, fmt.Errorf("check the page: %w", err)
	}

	site, err := s.deps.Store.ByProject(ctx, projectID)
	if err != nil || site.Status != domain.Running || !s.deps.Runtime.Alive(ctx, site.ContainerID) {
		return domain.Capture{}, domain.ErrNotRunning
	}

	request := CaptureRequest{ProjectID: projectID, Port: site.Port, Spec: spec}

	if s.deps.View != nil {
		prefix := site.BasePath()
		if site.Visibility == domain.Private {
			prefix += "/~preview" // a private site's address carries a ticket; the preview needs none
		}

		view, closeView, openErr := s.deps.View.Open(ctx, s.deps.Runtime.Socket(projectID), site.BasePath(), prefix)
		if openErr != nil {
			return domain.Capture{}, fmt.Errorf("capture: %w", openErr)
		}

		defer closeView()

		request.ViewSocket, request.Prefix = view, prefix
	}

	output, err := s.deps.Runtime.Capture(ctx, request)
	if err != nil {
		return domain.Capture{}, fmt.Errorf("capture: %w", err)
	}

	return ParseCapture(output)
}

// ParseCapture reads the result line the capture command printed among the browser's own output.
func ParseCapture(output string) (domain.Capture, error) {
	for _, line := range strings.Split(output, "\n") {
		text, found := strings.CutPrefix(strings.TrimSpace(line), marker)
		if !found {
			continue
		}

		var raw struct {
			Error   string       `json:"error"`
			PNG     string       `json:"png"`
			Console []string     `json:"console"`
			Failed  []string     `json:"failed"`
			Metrics domain.Page  `json:"metrics"`
			Shot    domain.Frame `json:"shot"`
			OK      bool         `json:"ok"`
		}
		if err := json.Unmarshal([]byte(text), &raw); err != nil {
			return domain.Capture{}, fmt.Errorf("%w: unreadable result", domain.ErrNoPicture)
		}

		if !raw.OK {
			return domain.Capture{}, fmt.Errorf("%w: %s", domain.ErrNoPicture, raw.Error)
		}

		png, err := base64.StdEncoding.DecodeString(raw.PNG)
		if err != nil || len(png) == 0 {
			return domain.Capture{}, fmt.Errorf("%w: empty picture", domain.ErrNoPicture)
		}

		return domain.Capture{Page: raw.Metrics, Console: raw.Console, Failed: raw.Failed, PNG: png, Frame: raw.Shot}, nil
	}

	return domain.Capture{}, fmt.Errorf("%w: no answer from the browser", domain.ErrNoPicture)
}
