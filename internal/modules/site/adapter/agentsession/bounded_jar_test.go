package agentsession

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestCookieJarBoundsAllCookieFieldsAndTotalBytes(t *testing.T) {
	jar, target := newTestBoundedJar(t), sessionURL(t)
	cookies := []*http.Cookie{
		{
			Name: strings.Repeat("n", maxCookieBytes), Value: "v",
			Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode,
		},
		{
			Name: "path", Value: "v", Path: strings.Repeat("p", maxCookieBytes),
			Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode,
		},
		{
			Name: "domain", Value: "v", Domain: strings.Repeat("d", maxCookieBytes),
			Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode,
		},
	}
	jar.SetCookies(target, cookies)
	if got := len(jar.Cookies(target)); got != 0 {
		t.Fatalf("oversized cookie fields accepted: %d cookies", got)
	}
	for i := range 9 {
		jar.SetCookies(target, []*http.Cookie{
			secureTestCookie(fmt.Sprintf("large%d", i), strings.Repeat("v", maxCookieBytes-16)),
		})
	}
	if got := jar.bytes; got > maxJarBytes {
		t.Fatalf("jar bytes = %d, limit %d", got, maxJarBytes)
	}
}

func TestExpiredCookieNamesReleaseBoundedJarAccounting(t *testing.T) {
	jar, target := newTestBoundedJar(t), sessionURL(t)
	jar.SetCookies(target, []*http.Cookie{{
		Name: "old", Value: "value", Expires: time.Now().Add(-time.Second),
		Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode,
	}})
	if len(jar.keys) != 0 || jar.bytes != 0 {
		t.Fatalf("expired cookie accounting remains: %d names, %d bytes", len(jar.keys), jar.bytes)
	}
	jar.SetCookies(target, []*http.Cookie{{
		Name: "fresh", Value: "value", Expires: time.Now().Add(time.Hour),
		Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode,
	}})
	if len(jar.Cookies(target)) != 1 || len(jar.keys) != 1 {
		t.Fatalf("fresh cookie was not accounted: %d names", len(jar.keys))
	}
}

func TestDefaultCookiePathsHaveSeparateBoundedEntries(t *testing.T) {
	jar := newTestBoundedJar(t)
	first, second := sessionURL(t), sessionURL(t)
	first.Path, second.Path = "/one/page", "/two/page"
	jar.SetCookies(first, []*http.Cookie{secureTestCookie("session", "one")})
	jar.SetCookies(second, []*http.Cookie{secureTestCookie("session", "two")})
	if len(jar.keys) != 2 || jar.bytes == 0 {
		t.Fatalf("path-scoped cookies accounting = %d names, %d bytes", len(jar.keys), jar.bytes)
	}
}

func secureTestCookie(name, value string) *http.Cookie {
	return &http.Cookie{
		Name: name, Value: value, Secure: true, HttpOnly: true, SameSite: http.SameSiteStrictMode,
	}
}
