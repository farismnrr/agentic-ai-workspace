package runtime

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/application"
)

const (
	getTimeout = 10 * time.Second
	maxPage    = 1 << 20
)

// Get requests a path from the running app as the proxy would: through its socket, without the address
// prefix, and with the prefix named in X-Forwarded-Prefix.
func (r *Runtime) Get(ctx context.Context, projectID, prefix, path string) (application.Page, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://site"+path, http.NoBody)
	if err != nil {
		return application.Page{}, fmt.Errorf("build request: %w", err)
	}

	req.Host = "site"
	req.Header.Set("X-Forwarded-Prefix", prefix)

	resp, err := Client(r.Socket(projectID), getTimeout).Do(req)
	if err != nil {
		return application.Page{}, fmt.Errorf("get %s: %w", path, err)
	}

	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxPage))
	if err != nil {
		return application.Page{}, fmt.Errorf("read %s: %w", path, err)
	}

	return application.Page{Status: resp.StatusCode, Type: resp.Header.Get("Content-Type"), Body: string(body)}, nil
}
