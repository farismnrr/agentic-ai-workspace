package http

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

// A sandboxed page has an opaque origin, so the browser can neither keep nor send the app's cookies.
// The jar keeps them here instead, per site and visitor, and the proxy puts them back on the next
// request. The browser never sees them, so one app's login cannot reach another app or the core app.
const (
	jarIdle       = 4 * time.Hour // as long as a ticket lives
	jarMaxJars    = 1000
	jarMaxCookies = 50
	jarMaxBytes   = 4096 // name and value together, as browsers allow
	jarMaxAge     = 400 * 24 * time.Hour
)

type storedCookie struct {
	expires           time.Time // zero for a cookie that lasts as long as the jar
	name, value, path string
	quoted            bool
}

// pair is the cookie as a request carries it: name=value.
func (c storedCookie) pair() string {
	if c.quoted {
		return c.name + `="` + c.value + `"`
	}

	return c.name + "=" + c.value
}

type jarEntry struct {
	cookies map[string]storedCookie
	touched time.Time
}

type cookieJar struct {
	now  func() time.Time
	jars map[string]*jarEntry
	mu   sync.Mutex
}

func newCookieJar() *cookieJar {
	return &cookieJar{now: time.Now, jars: map[string]*jarEntry{}}
}

// jarSession is one visitor's jar for one site. A nil session keeps nothing, as for the AI's preview.
type jarSession struct {
	jar *cookieJar
	key string
}

// session returns the jar of one site address token and visitor.
func (j *cookieJar) session(token, holder string) *jarSession {
	return &jarSession{jar: j, key: token + "|" + holder}
}

// header returns the Cookie header for a request the app sees at path, or "" when there is none.
func (s *jarSession) header(path string) string {
	if s == nil {
		return ""
	}

	s.jar.mu.Lock()
	defer s.jar.mu.Unlock()

	entry, now := s.jar.jars[s.key], s.jar.now()
	if entry == nil {
		return ""
	}

	entry.touched = now
	parts := []string{}

	for _, c := range entry.cookies {
		if live(c, now) && pathMatches(c.path, path) {
			parts = append(parts, c.pair())
		}
	}

	return strings.Join(parts, "; ")
}

// record keeps the cookies the app set in answer to a request at path, and forgets those it ended.
func (s *jarSession) record(path string, setCookies []string) {
	if s == nil || len(setCookies) == 0 {
		return
	}

	s.jar.mu.Lock()
	defer s.jar.mu.Unlock()

	now := s.jar.now()
	entry := s.jar.entry(s.key, now)

	for _, line := range setCookies {
		parsed, err := http.ParseSetCookie(line)
		if err != nil || len(parsed.Name)+len(parsed.Value) > jarMaxBytes {
			continue
		}

		c := storedCookie{
			name: parsed.Name, value: parsed.Value, quoted: parsed.Quoted, path: cookiePath(parsed.Path, path),
		}
		id := c.name + "\x00" + c.path

		expires, keep := lifetime(parsed, now)
		if !keep {
			delete(entry.cookies, id)
			continue
		}

		c.expires = expires
		if _, known := entry.cookies[id]; known || len(entry.cookies) < jarMaxCookies {
			entry.cookies[id] = c
		}
	}
}

// entry returns the jar for key, making room first: idle jars go, and the oldest when still full.
func (j *cookieJar) entry(key string, now time.Time) *jarEntry {
	if entry := j.jars[key]; entry != nil {
		entry.touched = now
		return entry
	}

	oldest, oldestKey := now, ""

	for k, e := range j.jars {
		if now.Sub(e.touched) > jarIdle {
			delete(j.jars, k)
		} else if e.touched.Before(oldest) {
			oldest, oldestKey = e.touched, k
		}
	}

	if len(j.jars) >= jarMaxJars {
		delete(j.jars, oldestKey)
	}

	entry := &jarEntry{cookies: map[string]storedCookie{}, touched: now}
	j.jars[key] = entry

	return entry
}

// lifetime says when a cookie ends and whether to keep it at all: Max-Age wins over Expires.
func lifetime(c *http.Cookie, now time.Time) (time.Time, bool) {
	switch {
	case c.MaxAge < 0:
		return time.Time{}, false
	case c.MaxAge > 0:
		return now.Add(min(time.Duration(c.MaxAge)*time.Second, jarMaxAge)), true
	case !c.Expires.IsZero():
		return c.Expires, c.Expires.After(now)
	}

	return time.Time{}, true
}

func live(c storedCookie, now time.Time) bool { return c.expires.IsZero() || c.expires.After(now) }

// cookiePath is the cookie's Path, or the folder of the request that set it (RFC 6265 default-path).
func cookiePath(attr, requestPath string) string {
	if strings.HasPrefix(attr, "/") {
		return attr
	}

	if i := strings.LastIndex(requestPath, "/"); i > 0 {
		return requestPath[:i]
	}

	return "/"
}

func pathMatches(cookie, request string) bool {
	if cookie == "/" || cookie == request {
		return true
	}

	rest, found := strings.CutPrefix(request, cookie)

	return found && (strings.HasSuffix(cookie, "/") || strings.HasPrefix(rest, "/"))
}
