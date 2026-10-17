// Package fetch reads web pages and pictures the model asks for. The addresses come from strangers' pages and
// search results, so every connection goes through the egress guard (public internet only, checked on the address
// each connection really uses, redirects included), and everything read is capped and checked.
package fetch

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/web/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/egress"
)

const (
	timeout   = 25 * time.Second
	sniffSize = 512
	userAgent = "agentic-ai-workspace/1 (an AI assistant reading a page for its user)"
)

// Fetcher reads addresses over a client the caller has already restricted.
type Fetcher struct {
	http *http.Client
}

// New wraps a client. Use NewClient for the one that guards connections.
func New(client *http.Client) *Fetcher { return &Fetcher{http: client} }

// NewClient is a client that may only reach the public internet and follows a few redirects.
func NewClient(policy egress.Policy) *http.Client {
	// HTTP/1.1 only: Flickr's CDN answers every Go HTTP/2 request with 429 "CIDR range blocked" and the same
	// request over HTTP/1.1 with the picture, and most pictures from Openverse live there.
	transport := egress.Transport(policy)
	transport.ForceAttemptHTTP2 = false
	transport.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) > domain.MaxRedirects {
				return domain.ErrUnavailable
			}

			if _, err := domain.ValidURL(req.URL.String()); err != nil {
				return err //nolint:wrapcheck // the domain error is what the model is told
			}

			return nil
		},
	}
}

func (f *Fetcher) get(ctx context.Context, address string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, http.NoBody)
	if err != nil {
		return nil, domain.ErrBadURL
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := f.http.Do(req)
	if err != nil {
		if errors.Is(err, egress.ErrBlocked) {
			return nil, domain.ErrBlocked
		}

		return nil, fmt.Errorf("%w: %w", domain.ErrUnavailable, errors.Unwrap(err))
	}

	return resp, nil
}

// Page returns the text of an HTML or plain-text page.
func (f *Fetcher) Page(ctx context.Context, address string) (domain.Page, error) {
	resp, err := f.get(ctx, address)
	if err != nil {
		return domain.Page{}, err
	}

	defer func() { _ = resp.Body.Close() }()

	kind, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if resp.StatusCode != http.StatusOK || !readable(kind) {
		return domain.Page{}, domain.ErrNotAPage
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, domain.MaxPageBytes))
	if err != nil {
		return domain.Page{}, domain.ErrUnavailable
	}

	page := domain.Page{URL: resp.Request.URL.String()}
	if kind == "text/plain" {
		page.Text = tidy(string(body))
	} else {
		page.Title, page.Text = extract(strings.NewReader(string(body)))
	}

	if runes := []rune(page.Text); len(runes) > domain.MaxPageText {
		page.Text, page.Cut = string(runes[:domain.MaxPageText]), true
	}

	return page, nil
}

func readable(kind string) bool {
	return kind == "text/html" || kind == "application/xhtml+xml" || kind == "text/plain"
}

// Image returns a picture's bytes if they really are a JPEG, PNG, GIF or WebP: the type is read from the bytes,
// not from what the server says.
func (f *Fetcher) Image(ctx context.Context, address string, limit int64) ([]byte, string, error) {
	resp, err := f.get(ctx, address)
	if err != nil {
		return nil, "", err
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, "", domain.ErrNotAnImage
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, "", domain.ErrUnavailable
	}

	if int64(len(data)) > limit {
		return nil, "", domain.ErrTooLarge
	}

	kind := http.DetectContentType(data[:min(len(data), sniffSize)])
	switch kind {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
		return data, kind, nil
	}

	return nil, "", domain.ErrNotAnImage
}
