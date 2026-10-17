package application

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/web/domain"
)

// Service is what an agent uses to look things up: web search, a page's text, usable pictures and picture files.
type Service struct {
	searcher   Searcher
	fetcher    Fetcher
	candidates *candidates
	images     []ImageSource
}

// NewService wires the lookups. searcher may be nil.
func NewService(searcher Searcher, images []ImageSource, fetcher Fetcher, now func() time.Time) *Service {
	return &Service{searcher: searcher, images: images, fetcher: fetcher, candidates: newCandidates(now)}
}

// CanSearch reports whether this server can search the web at all.
func (s *Service) CanSearch() bool { return s != nil && s.searcher != nil }

// Search looks the web up and returns up to eight hits.
func (s *Service) Search(ctx context.Context, query string) ([]domain.Result, error) {
	if !s.CanSearch() {
		return nil, domain.ErrNotConfigured
	}

	query, err := domain.ValidQuery(query)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	results, err := s.searcher.Search(ctx, query, domain.MaxResults)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}

	return results, nil
}

// Read returns the text of a web page.
func (s *Service) Read(ctx context.Context, address string) (domain.Page, error) {
	parsed, err := domain.ValidURL(address)
	if err != nil {
		return domain.Page{}, fmt.Errorf("read page: %w", err)
	}

	page, err := s.fetcher.Page(ctx, parsed.String())
	if err != nil {
		return domain.Page{}, fmt.Errorf("read page: %w", err)
	}

	return page, nil
}

// SearchImages asks every picture source at once and keeps only pictures whose licence allows editing and
// commercial use, alternating between sources so one does not crowd out the other. What it returns is what a
// later Asset call may fetch.
func (s *Service) SearchImages(ctx context.Context, projectID, query string) ([]domain.Image, error) {
	query, err := domain.ValidQuery(query)
	if err != nil {
		return nil, fmt.Errorf("search pictures: %w", err)
	}

	lists := make([][]domain.Image, len(s.images))
	failures := make([]error, len(s.images))

	var group sync.WaitGroup

	for i, source := range s.images {
		group.Add(1)

		go func() {
			defer group.Done()

			found, err := source.Search(ctx, query, domain.MaxImageSearch)
			lists[i], failures[i] = usable(found), err
		}()
	}

	group.Wait()

	merged := interleave(lists, domain.MaxImageSearch)
	if len(merged) == 0 && slices.ContainsFunc(failures, func(err error) bool { return err != nil }) {
		return nil, fmt.Errorf("search pictures: %w", firstFailure(failures))
	}

	s.candidates.remember(projectID, merged)

	return merged, nil
}

func usable(images []domain.Image) []domain.Image {
	kept := make([]domain.Image, 0, len(images))

	for _, image := range images {
		if domain.UsableLicense(image.License) && strings.HasPrefix(image.URL, "http") {
			kept = append(kept, image)
		}
	}

	return kept
}

// interleave takes one picture from each list in turn, skipping a picture whose address is already taken.
func interleave(lists [][]domain.Image, limit int) []domain.Image {
	var out []domain.Image

	seen := map[string]bool{}

	for round := 0; len(out) < limit; round++ {
		more := false

		for _, list := range lists {
			if round >= len(list) {
				continue
			}

			more = true

			if image := list[round]; !seen[image.URL] && len(out) < limit {
				seen[image.URL] = true
				out = append(out, image)
			}
		}

		if !more {
			break
		}
	}

	return out
}

func firstFailure(failures []error) error {
	for _, err := range failures {
		if err != nil {
			return err
		}
	}

	return domain.ErrUnavailable
}

// Asset is a picture ready to be saved, with the line to put in the credits.
type Asset struct {
	Mime  string
	Data  []byte
	Image domain.Image
}

// Ext is the file extension that matches the picture's type.
func (a Asset) Ext() string {
	switch a.Mime {
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	}

	return ".jpg"
}

// Fetch downloads one of the pictures the project's latest image search returned.
func (s *Service) Fetch(ctx context.Context, projectID, address string) (Asset, error) {
	image, ok := s.candidates.find(projectID, strings.TrimSpace(address))
	if !ok {
		return Asset{}, domain.ErrNotListed
	}

	data, mime, err := s.fetcher.Image(ctx, image.URL, domain.MaxImageBytes)
	if err != nil {
		return Asset{}, fmt.Errorf("fetch picture: %w", err)
	}

	return Asset{Image: image, Mime: mime, Data: data}, nil
}

// StatusOf maps an HTTP status from a source to the error the model sees.
func StatusOf(status int) error {
	if status == http.StatusTooManyRequests {
		return domain.ErrBusy
	}

	return domain.ErrUnavailable
}
