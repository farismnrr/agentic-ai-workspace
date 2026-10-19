package domain

import (
	"net/url"
	"path"
	"regexp"
	"strings"
)

// A page's links are only right when they stay under the site's address: the proxy removes the
// address before the app sees a request, so a link that starts with "/" leaves the app and lands
// on the platform. References finds the links of a page and Problem says what is wrong with one.

var (
	attrPattern = regexp.MustCompile(`(?i)\b(?:href|src|action|poster)\s*=\s*(?:"([^"]*)"|'([^']*)')`)
	urlPattern  = regexp.MustCompile(`(?i)url\(\s*['"]?([^'")\s]+)`)
	schemeLike  = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9+.-]*:`)
)

// Reference is a link of a page, resolved against the address the page is opened at.
type Reference struct {
	// Raw is the link as written.
	Raw string
	// Path is where the browser would request it, starting with the address prefix when it stays inside.
	Path string
}

// References returns the links in a page's text, resolved against the page at base+pagePath. Links to
// other sites, anchors and data or script links are left out: they do not depend on the address.
func References(base, pagePath, text string) []Reference {
	page, err := url.Parse(base + pagePath)
	if err != nil {
		return nil
	}

	seen := map[string]bool{}
	out := []Reference{}

	for _, raw := range rawLinks(text) {
		raw = strings.TrimSpace(raw)
		if !local(raw) || seen[raw] {
			continue
		}

		seen[raw] = true

		if target, parseErr := page.Parse(raw); parseErr == nil {
			out = append(out, Reference{Raw: raw, Path: target.Path})
		}
	}

	return out
}

func rawLinks(text string) []string {
	var links []string

	for _, m := range attrPattern.FindAllStringSubmatch(text, -1) {
		links = append(links, m[1]+m[2])
	}

	for _, m := range urlPattern.FindAllStringSubmatch(text, -1) {
		links = append(links, m[1])
	}

	return links
}

// local reports whether a link points into the app: not empty, not an anchor, not another site, not a scheme.
func local(raw string) bool {
	return raw != "" && !strings.HasPrefix(raw, "#") && !strings.HasPrefix(raw, "//") && !schemeLike.MatchString(raw)
}

// Inside reports whether a resolved path is under the site's address, and gives the path the app receives.
func Inside(base, resolved string) (string, bool) {
	if resolved == base {
		return "/", true
	}

	rest, found := strings.CutPrefix(resolved, base+"/")

	return "/" + rest, found
}

// IsPage guesses whether a link opens a page rather than a file: it ends in a slash or has no extension.
func IsPage(resolved string) bool {
	return strings.HasSuffix(resolved, "/") || path.Ext(resolved) == ""
}

// Problem is one link that would not work when the person opens the app.
type Problem struct {
	Link   string
	OnPage string
	Reason string
}

// Fix words a problem for the agent that wrote the page.
func (p Problem) Fix() string {
	return "`" + p.Link + "` on `" + p.OnPage + "` " + p.Reason
}

// Outside is the problem of a link that leaves the app's address.
func Outside(link, onPage string) Problem {
	return Problem{
		Link: link, OnPage: onPage,
		Reason: "points outside the app's address, so the browser asks the platform for it; start it with ./ " +
			"(../ for each folder up from a page in a folder), never with a bare /",
	}
}
