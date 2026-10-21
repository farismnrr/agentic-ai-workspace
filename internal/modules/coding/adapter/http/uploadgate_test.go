package http

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

func TestUploadConcurrencyRejectsRequestsAboveCap(t *testing.T) {
	app := fiber.New()
	entered := make(chan struct{}, 1)
	release := make(chan struct{})
	app.Post("/upload", uploadConcurrency(1), func(c fiber.Ctx) error {
		entered <- struct{}{}
		<-release
		return c.SendStatus(fiber.StatusOK)
	})
	firstDone := make(chan error, 1)
	go func() {
		request := httptest.NewRequestWithContext(context.Background(), "POST", "http://localhost/upload", nil)
		response, err := app.Test(request)
		if err != nil {
			firstDone <- err
			return
		}
		_ = response.Body.Close()
		firstDone <- nil
	}()
	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("first upload did not enter handler")
	}
	request := httptest.NewRequestWithContext(context.Background(), "POST", "http://localhost/upload", nil)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != fiber.StatusTooManyRequests {
		t.Fatalf("second request status = %d, want 429", response.StatusCode)
	}
	close(release)
	if err := <-firstDone; err != nil {
		t.Fatalf("first request failed: %v", err)
	}
}
