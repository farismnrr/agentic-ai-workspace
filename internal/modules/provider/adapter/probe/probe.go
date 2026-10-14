package probe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egress"
)

const (
	timeout      = 15 * time.Second
	maxBody      = 2 << 20
	anthropicAPI = "2023-06-01"
)

// nonChat marks Gemini catalogue entries that are not chat models.
var nonChat = []string{"embedding", "tts", "image", "live", "audio", "robotics", "computer-use"}

var (
	errStatus     = errors.New("the provider rejected the request")
	errNoRedirect = errors.New("the provider address redirects elsewhere")
	errNoModels   = errors.New("the provider returned no models")
)

// Probe lists the models of a provider over plain HTTP; a successful call
// proves the address and key work.
type Probe struct {
	hosted  *http.Client
	gateway *http.Client
}

// New builds a Probe that connects to hosted providers and to custom gateways under
// the given policies. Redirects are refused so a key is never sent to another host.
func New(hosted, gateway egress.Policy) *Probe {
	return &Probe{hosted: guarded(hosted), gateway: guarded(gateway)}
}

func guarded(policy egress.Policy) *http.Client {
	return &http.Client{
		Timeout:       timeout,
		Transport:     egress.Transport(policy),
		CheckRedirect: func(*http.Request, []*http.Request) error { return errNoRedirect },
	}
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

	client := p.hosted
	if credential.Kind.CustomAddress() {
		client = p.gateway
	}

	models, err := fetch(client, request)
	if err != nil || credential.Kind != domain.KindGemini {
		return models, err
	}

	return geminiChatModels(models)
}

// geminiChatModels keeps the models that can chat: Google lists embedding, speech,
// image and live models in the same catalogue, and names them "models/<id>".
func geminiChatModels(listed []string) ([]string, error) {
	chat := make([]string, 0, len(listed))

	for _, id := range listed {
		id = strings.TrimPrefix(id, "models/")
		if strings.HasPrefix(id, "gemini-") && !slices.ContainsFunc(nonChat, func(word string) bool {
			return strings.Contains(id, word)
		}) {
			chat = append(chat, id)
		}
	}

	if len(chat) == 0 {
		return nil, errNoModels
	}

	return chat, nil
}

func fetch(client *http.Client, request *http.Request) ([]string, error) {
	response, err := client.Do(request)
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
