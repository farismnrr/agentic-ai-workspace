package http

import (
	"net/http"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/site/domain"
)

// sandboxPolicy puts every page a site serves in a sandbox without allow-same-origin: the
// browser gives it an opaque origin, so it cannot read the person's cookies or storage,
// cannot register a service worker, and its requests to the app carry no login. This header
// is what keeps a site from touching the core app; the other rules here back it up.
const sandboxPolicy = "sandbox allow-scripts allow-forms allow-popups allow-modals allow-downloads; " +
	"frame-ancestors 'self'"

// hopByHop headers describe one connection, never the message; a proxy must not forward them.
var hopByHop = []string{
	"Connection", "Proxy-Connection", "Keep-Alive", "Proxy-Authenticate", "Proxy-Authorization",
	"Te", "Trailer", "Transfer-Encoding", "Upgrade",
}

// requestDrop never reaches the app: the person's login, anything that names the client,
// and the header browsers send when fetching a service worker script.
var requestDrop = []string{
	"Cookie", "Authorization", "Service-Worker", "Forwarded", "X-Forwarded-For", "X-Forwarded-Host",
	"X-Forwarded-Proto", "X-Forwarded-Prefix", "X-Real-Ip", "Expect", "Accept-Encoding",
}

// responseDrop never reaches the browser: cookies for the shared origin, instructions that
// change how the browser treats the whole origin, and the app's own security policy.
var responseDrop = []string{
	"Set-Cookie", "Set-Cookie2", "Clear-Site-Data", "Service-Worker-Allowed", "Strict-Transport-Security",
	"Content-Security-Policy", "Content-Security-Policy-Report-Only", "Public-Key-Pins", "Alt-Svc",
	"Server", "Via", "Content-Length",
}

// stripHopByHop removes the standard hop-by-hop headers and every header the sender
// nominated in Connection. It runs before this proxy adds anything, so a client cannot use
// Connection to make the proxy drop a header it set itself.
func stripHopByHop(h http.Header) {
	for _, value := range h.Values("Connection") {
		for _, name := range strings.Split(value, ",") {
			if name = strings.TrimSpace(name); name != "" {
				h.Del(name)
			}
		}
	}

	for _, name := range hopByHop {
		h.Del(name)
	}
}

// forRequest returns the headers the app may see.
func forRequest(in http.Header, prefix, clientIP string) http.Header {
	out := in.Clone()
	stripHopByHop(out)

	for _, name := range requestDrop {
		out.Del(name)
	}

	for name := range out {
		if strings.HasPrefix(strings.ToLower(name), "proxy-") {
			delete(out, name)
		}
	}

	out.Set("X-Forwarded-Prefix", prefix)
	out.Set("X-Forwarded-For", clientIP)

	return out
}

// forResponse returns the headers the browser may see, with the security headers added.
func forResponse(in http.Header, base, prefix string) http.Header {
	out := in.Clone()
	stripHopByHop(out)

	for _, name := range responseDrop {
		out.Del(name)
	}

	for name := range out {
		if strings.HasPrefix(strings.ToLower(name), "access-control-") {
			delete(out, name)
		}
	}

	if loc := out.Get("Location"); loc != "" {
		out.Set("Location", domain.Link(loc, base, prefix))
	}

	secure(out)
	out.Set("Access-Control-Allow-Origin", "*")

	return out
}

// secure adds the headers every answer under /run carries, errors and preflights included.
func secure(h http.Header) {
	h.Set("Content-Security-Policy", sandboxPolicy)
	h.Set("X-Content-Type-Options", "nosniff")
	h.Set("Referrer-Policy", "no-referrer")
	h.Set("X-Robots-Tag", "noindex, nofollow")
	h.Set("Cache-Control", "no-store")
}

// A dedicated hostname is a separate browser origin. Only app-specific cookies
// may cross the proxy; the Workspace session cookie is never forwarded.
func forSiteRequest(in http.Header, prefix, clientIP string, preview bool) http.Header {
	out := forRequest(in, prefix, clientIP)
	if !preview {
		return out
	}
	cookies := []string{}
	for _, part := range strings.Split(in.Get("Cookie"), ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		name, _, ok := strings.Cut(part, "=")
		if !ok || strings.EqualFold(name, "workspace_session") || strings.EqualFold(name, "session") ||
			strings.EqualFold(name, previewAccessCookie) {
			continue
		}
		cookies = append(cookies, part)
	}
	if len(cookies) > 0 {
		out.Set("Cookie", strings.Join(cookies, "; "))
	}
	return out
}

func forSiteResponse(in http.Header, base, prefix string, preview bool) http.Header {
	out := forResponse(in, base, prefix)
	if !preview {
		return out
	}
	out.Del("Content-Security-Policy")
	// Even on its dedicated host the app cannot embed the Workspace UI.
	if policy := in.Get("Content-Security-Policy"); policy != "" {
		out.Set("Content-Security-Policy", policy+"; frame-ancestors 'none'")
	} else {
		out.Set("Content-Security-Policy", "frame-ancestors 'none'; object-src 'none'")
	}
	out.Del("Access-Control-Allow-Origin")
	for _, cookie := range in.Values("Set-Cookie") {
		lower := strings.ToLower(cookie)
		if strings.Contains(lower, "domain=") || strings.HasPrefix(lower, "workspace_session=") ||
			strings.HasPrefix(lower, "session=") || strings.HasPrefix(lower, strings.ToLower(previewAccessCookie)+"=") {
			continue
		}
		out.Add("Set-Cookie", cookie)
	}
	return out
}
