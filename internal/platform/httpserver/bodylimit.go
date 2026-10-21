package httpserver

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/valyala/fasthttp"
)

const uploadSessionLookupTimeout = 2 * time.Second

var validProjectIDPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{4,28}[a-z0-9]$`)

// BodyRule describes the request headers that may receive a larger body limit.
type BodyRule struct {
	Method      string
	Path        string
	ContentType string
	MaxBodySize int
	ReadTimeout time.Duration
}

// RaiseBodyLimit chains a request-specific limit onto Fiber's existing hook.
func RaiseBodyLimit(app *fiber.App, rule BodyRule, auth Authenticator) {
	server := app.Server()
	previous := server.HeaderReceived
	server.HeaderReceived = func(header *fasthttp.RequestHeader) fasthttp.RequestConfig {
		var config fasthttp.RequestConfig
		if previous != nil {
			config = previous(header)
		}
		if matchesBodyRule(header, rule) && validUploadSession(header, auth) {
			config.MaxRequestBodySize = rule.MaxBodySize
			config.ReadTimeout = rule.ReadTimeout
		}
		return config
	}
}

func validUploadSession(header *fasthttp.RequestHeader, auth Authenticator) (valid bool) {
	if auth == nil {
		return false
	}
	defer func() {
		if recover() != nil {
			valid = false
		}
	}()
	token := string(header.Cookie(SessionCookieName))
	if token == "" {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), uploadSessionLookupTimeout)
	defer cancel()
	_, err := auth.UserID(ctx, token)
	return err == nil
}

func matchesBodyRule(header *fasthttp.RequestHeader, rule BodyRule) bool {
	if string(header.Method()) != rule.Method || header.ContentLength() <= 0 {
		return false
	}
	requestPath := string(header.RequestURI())
	if query := strings.IndexByte(requestPath, '?'); query >= 0 {
		requestPath = requestPath[:query]
	}
	if !matchesProjectPath(requestPath, rule.Path) {
		return false
	}
	contentType := strings.ToLower(string(header.Peek("Content-Type")))
	expectedType := strings.ToLower(rule.ContentType)
	if contentType != expectedType && !strings.HasPrefix(contentType, expectedType+";") {
		return false
	}
	return true
}

func matchesProjectPath(got, suffix string) bool {
	if !strings.HasSuffix(got, suffix) {
		return false
	}
	prefix := strings.TrimSuffix(got, suffix)
	id := strings.TrimPrefix(prefix, "/api/projects/")
	return strings.HasPrefix(prefix, "/api/projects/") && validProjectID(id)
}

func validProjectID(id string) bool {
	return validProjectIDPattern.MatchString(id)
}
