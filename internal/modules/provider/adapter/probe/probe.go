package probe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/domain"
)

const (
	timeout      = 15 * time.Second
	maxBody      = 2 << 20
	anthropicAPI = "2023-06-01"
)

var (
	errStatus     = errors.New("the provider rejected the request")
	errNoRedirect = errors.New("the provider address redirects elsewhere")
	errNoModels   = errors.New("the provider returned no models")
)

// Probe lists the models of a provider over plain HTTP; a successful call
// proves the address and key work.
type Probe struct {
	client *http.Client
}

// New builds a Probe. Redirects are refused so a key is never sent to another host.
func New() *Probe {
	return &Probe{client: &http.Client{
		Timeout:       timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return errNoRedirect },
	}}
}

// Models returns the model ids a credential can use.
func (p *Probe) Models(ctx context.Context, credential domain.Credential) ([]string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, credential.BaseURL+"/models", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	request.Header.Set("Accept", "application/json")

	if credential.Kind == domain.KindAnthropic {
		request.URL.Path = "/v1/models"
		request.Header.Set("x-api-key", credential.APIKey)
		request.Header.Set("anthropic-version", anthropicAPI)
	} else {
		request.Header.Set("Authorization", "Bearer "+credential.APIKey)
	}

	return p.fetch(request)
}

func (p *Probe) fetch(request *http.Request) ([]string, error) {
	response, err := p.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("call provider: %w", err)
	}

	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP %d", errStatus, response.StatusCode)
	}

	var body struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}

	if err := json.NewDecoder(io.LimitReader(response.Body, maxBody)).Decode(&body); err != nil {
		return nil, fmt.Errorf("read model list: %w", err)
	}

	models := make([]string, 0, len(body.Data))

	for _, model := range body.Data {
		if model.ID != "" {
			models = append(models, model.ID)
		}
	}

	if len(models) == 0 {
		return nil, errNoModels
	}

	return models, nil
}
