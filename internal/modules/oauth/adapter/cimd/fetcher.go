package cimd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/oauth/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egress"
)

const (
	fetchTimeout = 5 * time.Second
	maxBody      = 5 << 10
)

var (
	errRedirected  = errors.New("client metadata document must not redirect")
	errTooLarge    = errors.New("client metadata document is too large")
	errMismatch    = errors.New("client_id in the document does not match its URL")
	errNoRedirects = errors.New("client metadata document has no valid redirect_uris")
	errStatus      = errors.New("client metadata document request failed")
)

// Fetcher downloads Client ID Metadata Documents. The authorization server
// fetches attacker-chosen URLs, so connections are only allowed to public
// addresses, checked after DNS resolution so rebinding cannot bypass it.
type Fetcher struct {
	client *http.Client
}

// NewFetcher builds a Fetcher with SSRF protections.
func NewFetcher() *Fetcher {
	return &Fetcher{client: &http.Client{
		Timeout:       fetchTimeout,
		Transport:     egress.Transport(egress.Policy{Rule: egress.PublicOnly}),
		CheckRedirect: func(*http.Request, []*http.Request) error { return errRedirected },
	}}
}

// Fetch loads, size-limits and validates the document at clientID.
func (f *Fetcher) Fetch(ctx context.Context, clientID string) (domain.Client, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, clientID, http.NoBody)
	if err != nil {
		return domain.Client{}, fmt.Errorf("build request: %w", err)
	}

	request.Header.Set("Accept", "application/json")

	response, err := f.client.Do(request)
	if err != nil {
		return domain.Client{}, fmt.Errorf("fetch client metadata: %w", err)
	}

	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return domain.Client{}, fmt.Errorf("%w: status %d", errStatus, response.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, maxBody+1))
	if err != nil {
		return domain.Client{}, fmt.Errorf("read client metadata: %w", err)
	}

	if len(body) > maxBody {
		return domain.Client{}, errTooLarge
	}

	return parseDocument(clientID, body)
}

type document struct {
	ClientID     string   `json:"client_id"`
	ClientName   string   `json:"client_name"`
	RedirectURIs []string `json:"redirect_uris"`
}

func parseDocument(clientID string, body []byte) (domain.Client, error) {
	var doc document
	if err := json.Unmarshal(body, &doc); err != nil {
		return domain.Client{}, fmt.Errorf("decode client metadata: %w", err)
	}

	if doc.ClientID != clientID {
		return domain.Client{}, errMismatch
	}

	if len(doc.RedirectURIs) == 0 {
		return domain.Client{}, errNoRedirects
	}

	for _, uri := range doc.RedirectURIs {
		if err := domain.ValidateRedirectURI(uri); err != nil {
			return domain.Client{}, fmt.Errorf("redirect_uri %q: %w", uri, err)
		}
	}

	name := doc.ClientName
	if name == "" {
		name = domain.RedirectHost(clientID)
	}

	return domain.Client{ID: clientID, Name: name, RedirectURIs: doc.RedirectURIs, Source: domain.SourceCIMD}, nil
}
