// Package openverse finds Creative Commons and public-domain pictures through the Openverse API. It needs no key.
package openverse

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/web/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/web/domain"
)

const (
	// DefaultBase is the public API.
	DefaultBase = "https://api.openverse.org"
	maxBody     = 1 << 20
	userAgent   = "agentic-ai-workspace/1 (finds licensed pictures for a person's own page)"
)

// Client reads the Openverse API.
type Client struct {
	http *http.Client
	base string
}

// New builds a client; the HTTP client decides where connections may go.
func New(base string, client *http.Client) *Client {
	return &Client{base: strings.TrimRight(base, "/"), http: client}
}

// Name is how the source is listed.
func (c *Client) Name() string { return "openverse" }

type response struct {
	Results []struct {
		Title      string `json:"title"`
		URL        string `json:"url"`
		Page       string `json:"foreign_landing_url"`
		Creator    string `json:"creator"`
		License    string `json:"license"`
		Version    string `json:"license_version"`
		LicenseURL string `json:"license_url"`
		Source     string `json:"source"`
		Width      int    `json:"width"`
		Height     int    `json:"height"`
	} `json:"results"`
}

// Search asks for pictures whose licence allows commercial use and changes.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]domain.Image, error) {
	address := c.base + "/v1/images/?" + url.Values{
		"q": {query}, "license_type": {"commercial,modification"}, "page_size": {"20"}, "mature": {"false"},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, http.NoBody)
	if err != nil {
		return nil, domain.ErrUnavailable
	}

	req.Header.Set("User-Agent", userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, domain.ErrUnavailable
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: %w", resp.Status, application.StatusOf(resp.StatusCode))
	}

	var body response
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxBody)).Decode(&body); err != nil {
		return nil, domain.ErrUnavailable
	}

	images := make([]domain.Image, 0, limit)

	for _, hit := range body.Results {
		if len(images) == limit {
			break
		}

		images = append(images, domain.Image{
			Title: hit.Title, Creator: hit.Creator, Source: "openverse/" + hit.Source, URL: hit.URL, Page: hit.Page,
			License: licenseName(hit.License, hit.Version), LicenseURL: hit.LicenseURL, Width: hit.Width, Height: hit.Height,
		})
	}

	return images, nil
}

// licenseName writes Openverse's short code ("by-sa", "2.0") the way people read it ("CC BY-SA 2.0").
func licenseName(code, version string) string {
	switch strings.ToLower(code) {
	case "cc0":
		return "CC0"
	case "pdm":
		return "Public domain"
	case "":
		return ""
	}

	return strings.TrimSpace("CC " + strings.ToUpper(code) + " " + version)
}
