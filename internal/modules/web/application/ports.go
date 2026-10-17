// Package application holds what an agent can do on the web: search, read a page, find usable pictures and fetch one.
package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/web/domain"
)

// Searcher finds web pages. It is optional: a server without one cannot search the web.
type Searcher interface {
	Search(ctx context.Context, query string, limit int) ([]domain.Result, error)
}

// ImageSource finds pictures that carry a licence, such as a stock or a free-media library.
type ImageSource interface {
	Name() string
	Search(ctx context.Context, query string, limit int) ([]domain.Image, error)
}

// Fetcher reads addresses a stranger may have chosen. It must refuse internal addresses and cap what it reads.
type Fetcher interface {
	Page(ctx context.Context, address string) (domain.Page, error)
	// Image returns a picture's bytes and its detected type (image/jpeg, image/png, image/webp or image/gif).
	Image(ctx context.Context, address string, limit int64) (data []byte, mime string, err error)
}
