package domain

import (
	"bytes"
	"regexp"
	"strings"
)

// The proxy removes the site's address before the app sees a request, so a link that starts with "/" leaves the
// app and lands on the platform: the page shows without its stylesheet. Whatever stack the AI picked, Localize
// puts the address back on every such link in the pages and stylesheets the app answers with.

var (
	// head, the opening quote and the slash that starts a link; a second slash makes it a scheme-relative link.
	attrLink   = regexp.MustCompile(`(?i)(\b(?:href|src|action|poster|formaction)\s*=\s*["'])/(/?)`)
	cssLink    = regexp.MustCompile(`(?i)(url\(\s*["']?)/(/?)`)
	importLink = regexp.MustCompile(`(?i)(@import\s+["'])/(/?)`)
	srcsetAttr = regexp.MustCompile(`(?i)(\bsrcset\s*=\s*")([^"]*)(")`)
	srcsetLink = regexp.MustCompile(`((?:^|,)\s*)/(/?)`)
	headTag    = regexp.MustCompile(`(?i)<head(?:\s[^>]*)?>`)
	htmlTag    = regexp.MustCompile(`(?i)<html(?:\s[^>]*)?>`)
)

// Localize returns the text answer with its root links under prefix, the address the page was opened at (the
// site's base path, plus the ticket segment of a private site). Anything but a page or a stylesheet, or an answer
// the app compressed, comes back unchanged. A link that already starts with prefix, or with base (the address
// without a ticket), is kept or moved to prefix, so an app that knows its address is not rewritten twice.
func Localize(contentType, encoding, base, prefix string, body []byte) []byte {
	if encoding != "" && !strings.EqualFold(encoding, "identity") {
		return body
	}

	kind := strings.ToLower(contentType)

	switch {
	case strings.Contains(kind, "text/html") || strings.Contains(kind, "application/xhtml"):
		body = rewrite(body, attrLink, base, prefix)
		body = rewriteSrcset(body, base, prefix)
		body = rewrite(body, cssLink, base, prefix)

		return withShim(body, prefix)
	case strings.Contains(kind, "text/css"):
		return rewrite(rewrite(body, cssLink, base, prefix), importLink, base, prefix)
	default:
		return body
	}
}

// rewrite puts prefix in front of every link the pattern finds, unless the link already carries the address.
func rewrite(body []byte, pattern *regexp.Regexp, base, prefix string) []byte {
	var out bytes.Buffer

	last := 0

	for _, m := range pattern.FindAllSubmatchIndex(body, -1) {
		slash := m[3] // the byte after the head is the link's first slash

		if m[5] > m[4] { // "//host/...": not ours
			continue
		}

		out.Write(body[last:slash])
		last = slash

		rest := body[slash:]
		switch {
		case hasAddress(rest, prefix):
		case base != prefix && hasAddress(rest, base):
			out.WriteString(prefix)

			last = slash + len(base)
		default:
			out.WriteString(prefix)
		}
	}

	out.Write(body[last:])

	return out.Bytes()
}

func rewriteSrcset(body []byte, base, prefix string) []byte {
	return srcsetAttr.ReplaceAllFunc(body, func(attr []byte) []byte {
		m := srcsetAttr.FindSubmatch(attr)
		value := rewrite(m[2], srcsetLink, base, prefix)

		return append(append(append([]byte{}, m[1]...), value...), m[3]...)
	})
}

// hasAddress reports whether a link starts with the address as a whole path segment.
func hasAddress(link []byte, address string) bool {
	if !bytes.HasPrefix(link, []byte(address)) {
		return false
	}

	if len(link) == len(address) {
		return true
	}

	return strings.ContainsRune(`/?#"' )`, rune(link[len(address)]))
}

// withShim adds the script that does the same for links a page builds while it runs.
func withShim(body []byte, prefix string) []byte {
	if bytes.Contains(body, []byte(shimMarker)) {
		return body
	}

	tag := shimTag(prefix)

	for _, pattern := range []*regexp.Regexp{headTag, htmlTag} {
		if loc := pattern.FindIndex(body); loc != nil {
			return append(append(append([]byte{}, body[:loc[1]]...), tag...), body[loc[1]:]...)
		}
	}

	return append([]byte(tag), body...)
}

// Link returns one link, such as a redirect's Location, under prefix: a root link gets the address, a link that
// already carries it is kept, and one with the base but no ticket is moved to prefix. Anything else is unchanged.
func Link(link, base, prefix string) string {
	if !strings.HasPrefix(link, "/") || strings.HasPrefix(link, "//") {
		return link
	}

	switch raw := []byte(link); {
	case hasAddress(raw, prefix):
		return link
	case base != prefix && hasAddress(raw, base):
		return prefix + link[len(base):]
	default:
		return prefix + link
	}
}
