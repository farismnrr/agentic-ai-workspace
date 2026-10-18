// Package commons finds pictures on Wikimedia Commons, with the licence and author the file page states.
package commons

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/web/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/web/domain"
)

const (
	// DefaultBase is the public API.
	DefaultBase = "https://commons.wikimedia.org"
	maxBody     = 2 << 20
	// thumbWidth asks for a copy no wider than this, which is big enough for a page and far smaller than an original.
	thumbWidth = "1600"
	userAgent  = "agentic-ai-workspace/1 (finds licensed pictures for a person's own page)"
)

var tags = regexp.MustCompile(`<[^>]*>`)

// Client reads the Commons API.
type Client struct {
	http *http.Client
	base string
}

// New builds a client; the HTTP client decides where connections may go.
func New(base string, client *http.Client) *Client {
	return &Client{base: strings.TrimRight(base, "/"), http: client}
}

// Name is how the source is listed.
func (c *Client) Name() string { return "commons" }

type field struct {
	Value string `json:"value"`
}

type response struct {
	Query struct {
		Pages []struct {
			Title     string `json:"title"`
			ImageInfo []struct {
				Meta     map[string]field `json:"extmetadata"`
				URL      string           `json:"url"`
				ThumbURL string           `json:"thumburl"`
				Page     string           `json:"descriptionurl"`
				Mime     string           `json:"mime"`
				Width    int              `json:"thumbwidth"`
				Height   int              `json:"thumbheight"`
			} `json:"imageinfo"`
			Index int `json:"index"`
		} `json:"pages"`
	} `json:"query"`
}

// Search looks for photographs (not drawings or documents) matching the words.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]domain.Image, error) {
	address := c.base + "/w/api.php?" + url.Values{
		"action": {"query"}, "generator": {"search"}, "gsrsearch": {query + " filetype:bitmap"}, "gsrnamespace": {"6"},
		"gsrlimit": {"20"}, "prop": {"imageinfo"}, "iiprop": {"url|extmetadata|size|mime"}, "iiurlwidth": {thumbWidth},
		"format": {"json"}, "formatversion": {"2"},
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

	return images(body, limit), nil
}

func images(body response, limit int) []domain.Image {
	pages := body.Query.Pages
	sort.Slice(pages, func(i, j int) bool { return pages[i].Index < pages[j].Index })

	out := make([]domain.Image, 0, limit)

	for _, page := range pages {
		if len(out) == limit || len(page.ImageInfo) == 0 {
			continue
		}

		info := page.ImageInfo[0]
		if !strings.HasPrefix(info.Mime, "image/") || info.Meta["Restrictions"].Value != "" {
			continue
		}

		address := info.ThumbURL
		if address == "" {
			address = info.URL
		}

		out = append(out, domain.Image{
			Title:   strings.TrimSuffix(strings.TrimPrefix(page.Title, "File:"), filepathExt(page.Title)),
			Creator: plain(info.Meta["Artist"].Value),
			Source:  "commons", URL: address, Page: info.Page, License: plain(info.Meta["LicenseShortName"].Value),
			LicenseURL: plain(info.Meta["LicenseUrl"].Value), Width: info.Width, Height: info.Height,
		})
	}

	return out
}

func filepathExt(title string) string {
	if dot := strings.LastIndex(title, "."); dot > 0 && len(title)-dot <= 5 {
		return title[dot:]
	}

	return ""
}

// plain turns the markup Commons keeps in its fields into the text a person reads.
func plain(value string) string {
	return strings.Join(strings.Fields(html.UnescapeString(tags.ReplaceAllString(value, " "))), " ")
}
