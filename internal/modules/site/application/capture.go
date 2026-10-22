package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
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
	Session    *SessionAction
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

	site, err := s.runningSite(ctx, projectID)
	if err != nil {
		return domain.Capture{}, domain.ErrNotRunning
	}

	return s.captureSite(ctx, site, spec, nil)
}

// CaptureForAgent uses the task's private cookie jar for one capture. All identifiers come from the task runner.
func (s *Service) CaptureForAgent(
	ctx context.Context, userID, projectID, taskID string, spec domain.CaptureSpec,
) (domain.Capture, error) {
	spec, err := spec.Normalize()
	if err != nil {
		return domain.Capture{}, fmt.Errorf("check the page: %w", err)
	}

	site, err := s.runningSite(ctx, projectID)
	if err != nil || site.UserID != userID {
		return domain.Capture{}, domain.ErrNotRunning
	}
	var shot domain.Capture
	err = s.withAgentSession(ctx, site, userID, projectID, taskID, func(opCtx context.Context, jar http.CookieJar) error {
		var captureErr error
		shot, captureErr = s.captureSite(opCtx, site, spec, jar)
		return captureErr
	})
	if err != nil {
		return domain.Capture{}, fmt.Errorf("capture: %w", err)
	}

	return shot, nil
}

func (s *Service) runningSite(ctx context.Context, projectID string) (domain.Site, error) {
	site, err := s.deps.Store.ByProject(ctx, projectID)
	if err != nil || site.Status != domain.Running || !s.deps.Runtime.Alive(ctx, site.ContainerID) {
		return domain.Site{}, domain.ErrNotRunning
	}
	return site, nil
}

func (s *Service) captureSite(
	ctx context.Context, site domain.Site, spec domain.CaptureSpec, jar http.CookieJar,
) (domain.Capture, error) {
	projectID := site.ProjectID
	request := CaptureRequest{ProjectID: projectID, Port: site.Port, Spec: spec}

	if s.deps.View != nil {
		prefix := site.BasePath()

		view, closeView, openErr := s.deps.View.Open(
			ctx, s.deps.Runtime.Socket(projectID), site.BasePath(), prefix, jar,
		)
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
		return parseCaptureLine(text)
	}

	return domain.Capture{}, fmt.Errorf("%w: no answer from the browser", domain.ErrNoPicture)
}

func parseCaptureLine(text string) (domain.Capture, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		return domain.Capture{}, fmt.Errorf("%w: unreadable result", domain.ErrNoPicture)
	}
	var ok bool
	if json.Unmarshal(raw["ok"], &ok) != nil || !ok {
		return domain.Capture{}, domain.ErrNoPicture
	}
	status, frame, page, console, failed, err := captureDetails(raw)
	if err != nil {
		return domain.Capture{}, fmt.Errorf("%w: unreadable result", domain.ErrNoPicture)
	}
	png, err := capturePNG(raw)
	if err != nil {
		return domain.Capture{}, err
	}
	return domain.Capture{
		Status: status, Page: page, Console: console, Failed: failed, PNG: png, Frame: frame,
	}, nil
}

func captureDetails(
	raw map[string]json.RawMessage,
) (int, domain.Frame, domain.Page, []string, []string, error) {
	var status int
	var frame domain.Frame
	var page domain.Page
	var console, failed []string
	for _, field := range []struct {
		target any
		name   string
	}{
		{&status, "status"},
		{&frame, "shot"},
		{&page, "metrics"},
		{&console, "console"},
		{&failed, "failed"},
	} {
		if err := decodeCapture(raw, field.name, field.target); err != nil {
			return 0, domain.Frame{}, domain.Page{}, nil, nil, err
		}
	}
	return status, frame, page, console, failed, nil
}

func capturePNG(raw map[string]json.RawMessage) ([]byte, error) {
	var text string
	if err := decodeCapture(raw, "png", &text); err != nil {
		return nil, fmt.Errorf("%w: unreadable image", domain.ErrNoPicture)
	}
	png, err := base64.StdEncoding.DecodeString(text)
	if err != nil || len(png) == 0 {
		return nil, fmt.Errorf("%w: empty picture", domain.ErrNoPicture)
	}
	return png, nil
}

func decodeCapture(values map[string]json.RawMessage, name string, target any) error {
	if _, ok := values[name]; !ok {
		return nil
	}
	if err := json.Unmarshal(values[name], target); err != nil {
		return fmt.Errorf("decode capture %s: %w", name, err)
	}
	return nil
}
