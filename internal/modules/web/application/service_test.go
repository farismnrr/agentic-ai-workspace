package application_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/web/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/web/domain"
)

type source struct {
	err    error
	name   string
	images []domain.Image
}

func (s source) Name() string { return s.name }

func (s source) Search(context.Context, string, int) ([]domain.Image, error) { return s.images, s.err }

type fetcher struct{ asked []string }

func (f *fetcher) Page(context.Context, string) (domain.Page, error) {
	return domain.Page{Text: "x"}, nil
}

func (f *fetcher) Image(_ context.Context, address string, _ int64) ([]byte, string, error) {
	f.asked = append(f.asked, address)

	return []byte("PNG"), "image/png", nil
}

func photo(url, license string) domain.Image {
	return domain.Image{Title: url, URL: url, Page: url + "/page", License: license, Creator: "someone"}
}

func addresses(images []domain.Image) []string {
	list := make([]string, 0, len(images))
	for _, image := range images {
		list = append(list, image.URL)
	}

	return list
}

func TestOnlyUsablePicturesAreOfferedAndOnlyTheyCanBeFetched(t *testing.T) {
	t.Parallel()

	fetch := &fetcher{}
	now := time.Now()
	svc := application.NewService(nil, []application.ImageSource{
		source{name: "a", images: []domain.Image{
			photo("https://a/1.jpg", "CC BY 4.0"), photo("https://a/2.jpg", "CC BY-NC 4.0"),
		}},
		source{name: "b", images: []domain.Image{photo("https://b/1.jpg", "CC0"), photo("https://a/1.jpg", "CC BY 4.0")}},
	}, fetch, func() time.Time { return now })

	images, err := svc.SearchImages(context.Background(), "p1", "coffee shop")
	if err != nil || !slices.Equal(addresses(images), []string{"https://a/1.jpg", "https://b/1.jpg"}) {
		t.Fatalf("NonCommercial and duplicates are dropped, sources take turns: %+v, %v", images, err)
	}

	for _, refused := range []struct{ project, url, why string }{
		{"p1", "https://a/2.jpg", "a picture the search did not offer cannot be fetched"},
		{"p2", "https://a/1.jpg", "another project's search does not count"},
	} {
		if _, err = svc.Fetch(context.Background(), refused.project, refused.url); !errors.Is(err, domain.ErrNotListed) {
			t.Fatalf("%s: %v", refused.why, err)
		}
	}

	asset, err := svc.Fetch(context.Background(), "p1", "https://b/1.jpg")
	if err != nil || asset.Ext() != ".png" || asset.Image.License != "CC0" || len(fetch.asked) != 1 {
		t.Fatalf("an offered picture is fetched with its licence: %+v, %v", asset, err)
	}
}

func TestAnOldImageSearchIsForgotten(t *testing.T) {
	t.Parallel()

	now := time.Now()
	svc := application.NewService(nil, []application.ImageSource{
		source{name: "a", images: []domain.Image{photo("https://a/1.jpg", "CC0")}},
	}, &fetcher{}, func() time.Time { return now })

	if _, err := svc.SearchImages(context.Background(), "p1", "coffee"); err != nil {
		t.Fatal(err)
	}

	now = now.Add(2 * time.Hour)

	if _, err := svc.Fetch(context.Background(), "p1", "https://a/1.jpg"); !errors.Is(err, domain.ErrNotListed) {
		t.Fatalf("an old search is forgotten: %v", err)
	}
}

func TestWebSearchNeedsAServerAndAGoodQuery(t *testing.T) {
	t.Parallel()

	none := application.NewService(nil, nil, &fetcher{}, time.Now)
	if _, err := none.Search(context.Background(), "kopi bandung"); !errors.Is(err, domain.ErrNotConfigured) {
		t.Fatalf("without SearXNG: %v", err)
	}

	if none.CanSearch() {
		t.Fatal("no search server, no search")
	}

	failing := []application.ImageSource{source{name: "a", err: domain.ErrBusy}}
	broken := application.NewService(nil, failing, &fetcher{}, time.Now)
	if _, err := broken.SearchImages(context.Background(), "p1", "kopi"); !errors.Is(err, domain.ErrBusy) {
		t.Fatalf("when every source fails the reason is told: %v", err)
	}

	if _, err := none.Read(context.Background(), "file:///etc/passwd"); !errors.Is(err, domain.ErrBadURL) {
		t.Fatalf("only http and https are read: %v", err)
	}
}
