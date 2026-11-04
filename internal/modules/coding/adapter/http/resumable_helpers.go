package http

import (
	"encoding/base64"
	"errors"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v3"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

const tusVersion = "1.0.0"

func (h *Handler) zipLock(project string) func() {
	actual, _ := h.zipLocks.LoadOrStore(project, &sync.Mutex{})
	lock := actual.(*sync.Mutex)
	lock.Lock()
	return lock.Unlock
}

func zipHeaders(c fiber.Ctx) {
	c.Set("Tus-Resumable", tusVersion)
	c.Set("Cache-Control", "no-store")
}

func zipFailure(c fiber.Ctx, err error) error {
	zipHeaders(c)
	switch {
	case errors.Is(err, application.ErrZIPConflict), errors.Is(err, application.ErrZIPState):
		return c.Status(409).JSON(httpserver.Failure("upload_conflict"))
	case errors.Is(err, application.ErrZIPExpired):
		return c.Status(410).JSON(httpserver.Failure("upload_expired"))
	case errors.Is(err, application.ErrZIPQuota):
		return c.Status(507).JSON(httpserver.Failure("upload_quota"))
	default:
		return fail(c, err)
	}
}

func tusUploadName(header string) (string, bool) {
	for _, field := range strings.Split(header, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(field), " ")
		if !ok || key != "filename" {
			continue
		}
		raw, err := base64.StdEncoding.DecodeString(value)
		if err == nil && len(raw) > 0 && len(raw) <= 255 {
			return string(raw), true
		}
	}
	return "", false
}
