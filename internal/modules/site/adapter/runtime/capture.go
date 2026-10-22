package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/sandbox"
)

const (
	captureTimeout = 90 * time.Second
	// captureScript is where the sandbox image keeps the script that drives the browser.
	captureScript = "/usr/local/share/capture/capture.ts"
	// maxCaptureOutput holds a base64 screenshot of a tall page with room to spare.
	maxCaptureOutput = 16 << 20
)

// Capture photographs a page of the running app. It runs in a container of its own that reaches the app through
// the same socket as the server, because no container has a network; only the project's own app is reachable.
func (r *Runtime) Capture(ctx context.Context, req application.CaptureRequest) (string, error) {
	return r.runCapture(ctx, req, nil)
}

// Session performs a bounded form action. Credentials are serialized only to the container's stdin.
func (r *Runtime) Session(ctx context.Context, req application.CaptureRequest) (string, error) {
	if req.Session == nil {
		return "", domain.ErrNoPicture
	}
	input, err := json.Marshal(sessionPayload(req.Session))
	if err != nil {
		return "", domain.ErrNoPicture
	}
	return r.runCapture(ctx, req, input)
}

func sessionPayload(action *application.SessionAction) map[string]string {
	return map[string]string{
		"action": action.Action, "path": action.Path,
		"emailSelector": action.EmailSelector, "passwordSelector": action.PasswordSelector,
		"submitSelector": action.SubmitSelector, "readySelector": action.ReadySelector,
		"errorSelector": action.ErrorSelector, "email": action.Email, "password": action.Password,
	}
}

func (r *Runtime) runCapture(ctx context.Context, req application.CaptureRequest, input []byte) (string, error) {
	docker, err := r.engine.Docker(ctx)
	if err != nil {
		return "", fmt.Errorf("%w: %w", domain.ErrNoSandboxes, err)
	}

	width, height := req.Spec.Window()

	task, err := json.Marshal(map[string]any{
		"url":   "http://127.0.0.1:" + strconv.Itoa(req.Port) + req.Prefix + req.Spec.Path,
		"width": width, "height": height, "y": req.Spec.Y, "overview": req.Spec.Overview, "mobile": req.Spec.Mobile,
	})
	if err != nil {
		return "", fmt.Errorf("describe capture: %w", err)
	}

	env := []string{
		"LANG=C.UTF-8", "PATH=" + sandbox.Path, "HOME=/tmp", "TMPDIR=/tmp", "CAPTURE_PORT=" + strconv.Itoa(req.Port),
	}
	if req.ViewSocket != "" {
		env = append(env, "CAPTURE_SOCKET=/run/view/"+filepath.Base(req.ViewSocket))
	}

	argv := []string{"node", captureScript, string(task)}
	var stdin io.Reader
	if input != nil {
		argv = []string{"node", captureScript, "--session", string(task)}
		stdin = bytes.NewReader(input)
	}
	spec := sandbox.Spec{Site: r.dir(req.ProjectID), Argv: argv, Env: env, Stdin: stdin}
	if req.ViewSocket != "" {
		spec.View = filepath.Dir(req.ViewSocket)
	}

	ctx, cancel := context.WithTimeout(ctx, captureTimeout)
	defer cancel()

	out := &cappedBuffer{limit: maxCaptureOutput}
	if _, err = r.engine.RunOnce(ctx, docker, spec, out); err != nil {
		return "", fmt.Errorf("run the browser: %w", err)
	}

	return out.String(), nil
}

// cappedBuffer keeps what a command printed up to a limit and drops the rest.
type cappedBuffer struct {
	bytes.Buffer
	limit int
}

func (b *cappedBuffer) Write(p []byte) (int, error) {
	if room := b.limit - b.Len(); room > 0 {
		_, _ = b.Buffer.Write(p[:min(room, len(p))])
	}

	return len(p), nil
}
