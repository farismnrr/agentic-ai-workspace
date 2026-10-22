package agentsession

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	maxCookiesPerSession = 128
	maxCookieBytes       = 8 * 1024
	maxJarBytes          = 64 * 1024
)

type cookieRecord struct {
	bytes       int
	expiresUnix int64
}

type boundedJar struct {
	jar   http.CookieJar
	keys  map[string]cookieRecord
	bytes int
	mu    sync.Mutex
}

func (j *boundedJar) Cookies(target *url.URL) []*http.Cookie {
	j.mu.Lock()
	j.prune(time.Now())
	j.mu.Unlock()
	return j.jar.Cookies(target)
}

func (j *boundedJar) SetCookies(target *url.URL, cookies []*http.Cookie) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.prune(time.Now())
	accepted := make([]*http.Cookie, 0, len(cookies))
	for _, cookie := range cookies {
		if j.accept(target, cookie) {
			accepted = append(accepted, cookie)
		}
	}
	j.jar.SetCookies(target, accepted)
}

func (j *boundedJar) accept(target *url.URL, cookie *http.Cookie) bool {
	if cookie == nil {
		return false
	}
	key, size := cookieKey(target, cookie), cookieSize(target, cookie)
	if size > maxCookieBytes {
		return false
	}
	if expired(cookie) {
		j.remove(key)
		return true
	}
	previous, exists := j.keys[key]
	count, bytes := len(j.keys), j.bytes
	if exists {
		count--
		bytes -= previous.bytes
	}
	if count >= maxCookiesPerSession || bytes+size > maxJarBytes {
		return false
	}
	var expiresUnix int64
	if expires := cookieExpiration(cookie); !expires.IsZero() {
		expiresUnix = expires.UnixNano()
	}
	j.keys[key] = cookieRecord{bytes: size, expiresUnix: expiresUnix}
	j.bytes = bytes + size
	return true
}

func cookieSize(target *url.URL, cookie *http.Cookie) int {
	domain := cookie.Domain
	if domain == "" {
		domain = target.Hostname()
	}
	path := cookie.Path
	if path == "" {
		path = cookiePath(target, path)
	}
	return len(cookie.Name) + len(cookie.Value) + len(domain) + len(path)
}

func cookieExpiration(cookie *http.Cookie) time.Time {
	if cookie.MaxAge > 0 {
		return time.Now().Add(time.Duration(cookie.MaxAge) * time.Second)
	}
	return cookie.Expires
}

func (j *boundedJar) prune(now time.Time) {
	for key, record := range j.keys {
		if record.expiresUnix != 0 && record.expiresUnix <= now.UnixNano() {
			j.remove(key)
		}
	}
}

func (j *boundedJar) remove(key string) {
	if record, ok := j.keys[key]; ok {
		j.bytes -= record.bytes
		delete(j.keys, key)
	}
}

func cookieKey(target *url.URL, cookie *http.Cookie) string {
	host := target.Hostname()
	domain := strings.TrimPrefix(strings.ToLower(cookie.Domain), ".")
	if domain == "" {
		domain = strings.ToLower(host)
	}
	return fmt.Sprintf("%s\n%s\n%s", domain, cookiePath(target, cookie.Path), cookie.Name)
}

func cookiePath(target *url.URL, path string) string {
	if strings.HasPrefix(path, "/") {
		return path
	}
	requestPath := target.Path
	if requestPath == "" || !strings.HasPrefix(requestPath, "/") {
		return "/"
	}
	if index := strings.LastIndex(requestPath, "/"); index > 0 {
		return requestPath[:index]
	}
	return "/"
}

func expired(cookie *http.Cookie) bool {
	return cookie.MaxAge < 0 || (!cookie.Expires.IsZero() && !cookie.Expires.After(time.Now()))
}
