package httpserver

import (
	"testing"

	"github.com/gofiber/fiber/v3"
)

func TestErrorScreensKeepTheirStatusAndOtherClientRoutesAreOK(t *testing.T) {
	cases := map[string]int{
		"/not-found":  fiber.StatusNotFound,
		"/forbidden":  fiber.StatusForbidden,
		"/forbidden/": fiber.StatusForbidden,
		"/":           fiber.StatusOK,
		"/app/inbox":  fiber.StatusOK,
	}

	for path, want := range cases {
		if got := pageStatus(path); got != want {
			t.Errorf("%s: status %d, want %d", path, got, want)
		}
	}
}
