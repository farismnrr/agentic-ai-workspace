package http

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/gofiber/fiber/v3"
)

// CSRF is a per-process session-bound synchronizer token MAC.
type CSRF struct{ key [32]byte }

func NewCSRF() (*CSRF, error) {
	c := &CSRF{}
	if _, err := rand.Read(c.key[:]); err != nil {
		return nil, fmt.Errorf("initialize csrf: %w", err)
	}
	return c, nil
}

func (a *CSRF) Token(c fiber.Ctx) string {
	mac := hmac.New(sha256.New, a.key[:])
	_, _ = mac.Write([]byte(httpserver.UserID(c)))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(c.Cookies(httpserver.SessionCookieName)))
	return hex.EncodeToString(mac.Sum(nil))
}

func (a *CSRF) Validate(c fiber.Ctx) error {
	token, err := hex.DecodeString(c.Get("X-Egress-CSRF"))
	expected, _ := hex.DecodeString(a.Token(c))
	if err != nil || len(token) != len(expected) || !hmac.Equal(token, expected) {
		return errorJSON(c, fiber.StatusForbidden, "csrf_failed")
	}
	return c.Next()
}
