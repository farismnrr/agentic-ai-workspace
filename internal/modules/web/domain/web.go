// Package domain holds the rules of what an agent may look up on the web: what a search result and a page are,
// which photo licences may be used, and why a request is refused.
package domain

import (
	"errors"
	"net/url"
	"strings"
)

// Errors a web lookup can fail with; their text is safe to show to the model.
var (
	ErrNotConfigured = errors.New("web search is not set up on this server; ask the person to set SEARXNG_URL")
	ErrBadQuery      = errors.New("give a search of 2 to 200 characters")
	ErrBadURL        = errors.New("give a full http or https address")
	ErrTooLarge      = errors.New("that file is too large")
	ErrNotAnImage    = errors.New("that is not a JPEG, PNG, WebP or GIF picture")
	ErrNotAPage      = errors.New("that address is not a web page that can be read as text")
	ErrBlocked       = errors.New("that address cannot be reached from here; use a public website")
	ErrNotListed     = errors.New("that picture was not in the latest image search; search again and pick from it")
	ErrBusy          = errors.New("the picture service is busy; try again in a minute or use other words")
	ErrUnavailable   = errors.New("the web lookup did not answer; try again or use other words")
)

// Limits keep every lookup small and cheap.
const (
	MinQuery       = 2
	MaxQuery       = 200
	MaxResults     = 8
	MaxPageBytes   = 2 << 20
	MaxPageText    = 20_000
	MaxImageBytes  = 10 << 20
	MaxRedirects   = 5
	MaxImageSearch = 8
)

// Result is one hit of a web search.
type Result struct {
	Title   string
	URL     string
	Snippet string
}

// Page is the readable text of a web page. Its content comes from a stranger, so whoever reads it must treat it as
// data, never as instructions.
type Page struct {
	URL   string
	Title string
	Text  string
	// Cut is true when the text was shortened to the limit.
	Cut bool
}

// Image is a picture that may be used, with what its licence asks of whoever uses it.
type Image struct {
	// Title and Creator describe the work; Source names where it was found (openverse, commons, flickr...).
	Title   string
	Creator string
	Source  string
	// URL is the picture itself; Page is where it is shown, for the credit.
	URL  string
	Page string
	// License is a short name such as "CC BY 4.0" or "CC0"; LicenseURL says what it means.
	License    string
	LicenseURL string
	Width      int
	Height     int
}

// Credit is the line to put in the project's credits file: what to show and where it came from.
func (i Image) Credit() string {
	creator := i.Creator
	if creator == "" {
		creator = "unknown author"
	}

	line := "\"" + i.Title + "\" by " + creator + ", " + i.License
	if i.LicenseURL != "" {
		line += " (" + i.LicenseURL + ")"
	}

	return line + ", from " + i.Page
}

// ValidQuery trims a search and checks its length.
func ValidQuery(query string) (string, error) {
	query = strings.Join(strings.Fields(query), " ")
	if n := len([]rune(query)); n < MinQuery || n > MaxQuery {
		return "", ErrBadQuery
	}

	return query, nil
}

// ValidURL checks an address is a plain http or https URL with a host.
func ValidURL(raw string) (*url.URL, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil {
		return nil, ErrBadURL
	}

	return parsed, nil
}
