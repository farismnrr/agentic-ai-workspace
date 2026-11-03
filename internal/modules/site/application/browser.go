package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

const (
	maxBrowserSteps = 12
	maxBrowserText  = 512
	maxBrowserInput = 24 << 10
)

// BrowserAction is one supported operation in a QA browser run.
type BrowserAction struct {
	Action      string `json:"action"`
	Path        string `json:"path,omitempty"`
	LocatorType string `json:"locator_type,omitempty"`
	Locator     string `json:"locator,omitempty"`
	Value       string `json:"value,omitempty"`
	Check       string `json:"check,omitempty"`
}

// BrowserStep is the outcome of one requested browser action.
type BrowserStep struct {
	Index  int    `json:"index"`
	Action string `json:"action"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

// BrowserReport contains bounded diagnostics and the final screenshot.
type BrowserReport struct {
	Steps []BrowserStep `json:"steps"`
	PNG   []byte        `json:"-"`
}

// BrowserRuntime runs declarative actions in the isolated Site capture container.
type BrowserRuntime interface {
	Browser(context.Context, CaptureRequest, []BrowserAction) (string, error)
}

// BrowserForAgent uses only the current task's cookie jar and running project Site.
func (s *Service) BrowserForAgent(
	ctx context.Context, userID, projectID, taskID string, actions []BrowserAction,
) (BrowserReport, error) {
	if err := validateBrowserActions(actions); err != nil {
		return BrowserReport{}, err
	}
	site, err := s.runningSite(ctx, projectID)
	if err != nil || site.UserID != userID {
		return BrowserReport{}, domain.ErrNotRunning
	}
	runner, ok := s.deps.Runtime.(BrowserRuntime)
	if !ok {
		return BrowserReport{}, domain.ErrNoPicture
	}
	var report BrowserReport
	err = s.withAgentSession(ctx, site, userID, projectID, taskID, func(opCtx context.Context, jar http.CookieJar) error {
		request := CaptureRequest{ProjectID: projectID, Port: site.Port, Prefix: site.BasePath()}
		if s.deps.View != nil {
			view, closeView, openErr := s.deps.View.Open(
				opCtx, s.deps.Runtime.Socket(projectID), site.BasePath(), site.BasePath(), jar,
			)
			if openErr != nil {
				return fmt.Errorf("open Site viewer: %w", openErr)
			}
			defer closeView()
			request.ViewSocket = view
		}
		output, runErr := runner.Browser(opCtx, request, actions)
		if runErr != nil {
			return runErr
		}
		report, runErr = parseBrowserReport(output)
		return runErr
	})
	if err != nil {
		return BrowserReport{}, fmt.Errorf("run browser test: %w", err)
	}
	return report, nil
}

func validateBrowserActions(actions []BrowserAction) error {
	if len(actions) == 0 || len(actions) > maxBrowserSteps {
		return domain.ErrBadBrowserInput
	}
	size := 0
	for _, action := range actions {
		size += len(action.Path) + len(action.LocatorType) + len(action.Locator) + len(action.Value) + len(action.Check)
		if len(action.Path) > maxBrowserText || len(action.Locator) > maxBrowserText ||
			len(action.Value) > maxBrowserText || size > maxBrowserInput || !utf8.ValidString(action.Value) {
			return domain.ErrBadBrowserInput
		}
		switch action.Action {
		case "goto":
			if !validRelativeSitePath(action.Path) || action.LocatorType != "" || action.Locator != "" ||
				action.Value != "" || action.Check != "" {
				return domain.ErrBadBrowserPath
			}
		case "click", "fill":
			if !validLocator(action.LocatorType, action.Locator) || action.Path != "" || action.Check != "" ||
				(action.Action == "fill" && action.Value == "") ||
				(action.Action == "click" && action.Value != "") {
				return domain.ErrBadBrowserInput
			}
		case "assert":
			if !validLocator(action.LocatorType, action.Locator) ||
				(action.Check != "visible" && action.Check != "text") ||
				(action.Check == "text" && action.Value == "") ||
				(action.Check == "visible" && action.Value != "") || action.Path != "" {
				return domain.ErrBadBrowserInput
			}
		case "screenshot":
			if action.Path != "" || action.Locator != "" || action.Value != "" || action.Check != "" {
				return domain.ErrBadBrowserInput
			}
		default:
			return domain.ErrBadBrowserInput
		}
	}
	return nil
}

func validRelativeSitePath(path string) bool {
	if strings.HasPrefix(path, "/") || strings.ContainsAny(path, "\\?#%") || strings.Contains(path, ":") {
		return false
	}
	for _, part := range strings.Split(path, "/") {
		if part == ".." || part == "." {
			return false
		}
	}
	return true
}

func validLocator(kind, value string) bool {
	if value == "" || kind == "" {
		return false
	}
	switch kind {
	case "css", "text", "label", "placeholder", "role", "testid":
		return true
	default:
		return false
	}
}

func parseBrowserReport(output string) (BrowserReport, error) {
	line := "BROWSER_JSON:"
	for _, row := range strings.Split(output, "\n") {
		if text, found := strings.CutPrefix(strings.TrimSpace(row), line); found {
			var raw struct {
				Steps []BrowserStep `json:"steps"`
				PNG   string        `json:"png"`
			}
			if err := json.Unmarshal([]byte(text), &raw); err != nil {
				return BrowserReport{}, domain.ErrNoPicture
			}
			png, err := base64.StdEncoding.DecodeString(raw.PNG)
			if err != nil || len(png) == 0 || len(png) > 5<<20 {
				return BrowserReport{}, domain.ErrNoPicture
			}
			return BrowserReport{Steps: raw.Steps, PNG: png}, nil
		}
	}
	return BrowserReport{}, domain.ErrNoPicture
}
