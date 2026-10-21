package httpserver

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

const testUploadBodyLimit = 30 << 20

var errInvalidTestSession = errors.New("invalid session")

func TestUploadBodyRuleRequiresKnownProjectMultipartRequest(t *testing.T) {
	t.Parallel()
	rule := BodyRule{
		Method: "POST", Path: "/uploads", ContentType: "multipart/form-data",
		MaxBodySize: 101 << 20, ReadTimeout: 10 * time.Minute,
	}
	valid := uploadHeader(
		"POST", "/api/projects/project-alpha/uploads?x=1", "multipart/form-data; boundary=abc", "session=x", 12<<20,
	)
	if !matchesBodyRule(valid, rule) {
		t.Fatal("valid upload request did not match")
	}
	for _, header := range []*fasthttp.RequestHeader{
		uploadHeader("GET", "/api/projects/project-alpha/uploads", "multipart/form-data", "session=x", 100),
		uploadHeader("POST", "/api/projects/project-alpha/uploads/extra", "multipart/form-data", "session=x", 100),
		uploadHeader("POST", "/api/projects/project-alpha/uploads", "application/json", "session=x", 100),
		uploadHeader("POST", "/api/projects/short/uploads", "multipart/form-data", "session=x", 100),
		uploadHeader("POST", "/api/projects/project-alpha/uploads", "multipart/form-data", "session=x", 0),
	} {
		if matchesBodyRule(header, rule) {
			t.Errorf("unexpected match for request: %s %s", header.Method(), header.RequestURI())
		}
	}
}

func TestRaisedUploadLimitRequiresValidSession(t *testing.T) {
	t.Parallel()
	auth := sessionAuthenticator(func(token string) error {
		if token != "valid" {
			return errInvalidTestSession
		}
		return nil
	})
	rule := BodyRule{
		Method: "POST", Path: "/uploads", ContentType: "multipart/form-data",
		MaxBodySize: 101 << 20, ReadTimeout: 10 * time.Minute,
	}
	app := fiber.New()
	RaiseBodyLimit(app, rule, auth)
	for _, tc := range []struct {
		cookie string
		want   int
	}{{"", 0}, {"session=garbage", 0}, {"session=valid", 101 << 20}} {
		header := uploadHeader(
			"POST", "/api/projects/project-alpha/uploads", "multipart/form-data; boundary=x", tc.cookie, 12<<20,
		)
		config := app.Server().HeaderReceived(header)
		if config.MaxRequestBodySize != tc.want {
			t.Errorf("cookie %q: limit = %d, want %d", tc.cookie, config.MaxRequestBodySize, tc.want)
		}
	}
}

func TestRaisedUploadLimitFailsClosedWhenSessionLookupPanics(t *testing.T) {
	t.Parallel()
	auth := sessionAuthenticator(func(string) error { panic("session lookup failed") })
	app := fiber.New()
	RaiseBodyLimit(app, BodyRule{
		Method: "POST", Path: "/uploads", ContentType: "multipart/form-data",
		MaxBodySize: 101 << 20,
	}, auth)
	header := uploadHeader(
		"POST", "/api/projects/project-alpha/uploads", "multipart/form-data; boundary=x", "session=valid", 12<<20,
	)
	if config := app.Server().HeaderReceived(header); config.MaxRequestBodySize != 0 {
		t.Fatalf("panic raised request limit to %d", config.MaxRequestBodySize)
	}
}

func TestRaiseBodyLimitChainsExistingHeaderHook(t *testing.T) {
	t.Parallel()
	app := fiber.New()
	called := false
	app.Server().HeaderReceived = func(*fasthttp.RequestHeader) fasthttp.RequestConfig {
		called = true
		return fasthttp.RequestConfig{WriteTimeout: time.Second}
	}
	RaiseBodyLimit(app, BodyRule{
		Method: "POST", Path: "/uploads", ContentType: "multipart/form-data", MaxBodySize: testUploadBodyLimit,
	}, nil)
	config := app.Server().HeaderReceived(uploadHeader("POST", "/api/projects/p/uploads", "multipart/form-data", "", 10))
	if !called || config.WriteTimeout != time.Second || config.MaxRequestBodySize != 0 {
		t.Fatalf("prior hook was not preserved: called=%v config=%+v", called, config)
	}
}

type sessionAuthenticator func(string) error

func (f sessionAuthenticator) UserID(_ context.Context, token string) (string, error) {
	return "user", f(token)
}

func uploadHeader(method, requestPath, contentType, cookie string, length int) *fasthttp.RequestHeader {
	header := &fasthttp.RequestHeader{}
	header.SetMethod(method)
	header.SetRequestURI(requestPath)
	header.Set("Content-Type", contentType)
	header.Set("Cookie", cookie)
	header.SetContentLength(length)
	return header
}
