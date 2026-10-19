package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/domain"
)

type slugStore struct {
	application.Store
	taken   map[string]bool
	created []domain.Project
}

func (s *slugStore) SlugTaken(_ context.Context, slug string) (bool, error) {
	return s.taken[slug], nil
}

func (s *slugStore) Create(_ context.Context, p domain.Project) error {
	s.created = append(s.created, p)

	return nil
}

type oneID struct{}

func (oneID) ID() (string, error) { return "new", nil }

type noFolders struct{}

func (noFolders) Create(string) error { return nil }
func (noFolders) Remove(string) error { return nil }

func newSlugService(taken ...string) (*application.Service, *slugStore) {
	data := &slugStore{taken: map[string]bool{}}
	for _, slug := range taken {
		data.taken[slug] = true
	}

	return application.NewService(data, noFolders{}, nil, oneID{}, time.Now), data
}

func TestTwoProjectsMayShareANameButNeverAnID(t *testing.T) {
	t.Parallel()

	service, data := newSlugService("bakery")

	made, err := service.Create(context.Background(), "me", "Bakery", "", "")
	if err != nil || made.Slug != "bakery-2" || made.Name != "Bakery" {
		t.Fatalf("the name is kept and the ID gets a number: %+v %v", made, err)
	}

	if _, err = service.Create(context.Background(), "me", "Bakery", "", "bakery"); !errors.Is(err, domain.ErrSlugTaken) {
		t.Fatalf("a chosen ID that is taken must be refused: %v", err)
	}

	_, err = service.Create(context.Background(), "me", "Bakery", "", "No Good")
	if !errors.Is(err, domain.ErrInvalidSlug) {
		t.Fatalf("a chosen ID that is not valid must be refused: %v", err)
	}

	if len(data.created) != 1 {
		t.Errorf("only the first was saved: %d", len(data.created))
	}
}

func TestCheckSlugAnswersForTheIDOrForTheName(t *testing.T) {
	t.Parallel()

	service, _ := newSlugService("bakery", "bakery-2")

	cases := map[string]struct {
		id, name string
		want     application.SlugStatus
	}{
		"made from the name": {"", "Cafe", application.SlugStatus{Suggestion: "cafe-app", Valid: true, Available: true}},
		"a taken ID":         {"bakery", "", application.SlugStatus{Suggestion: "bakery-3", Valid: true}},
		"an invalid ID":      {"Bad ID", "", application.SlugStatus{Suggestion: "bad-id"}},
	}

	for label, c := range cases {
		got, err := service.CheckSlug(context.Background(), c.id, c.name)
		if err != nil || got != c.want {
			t.Errorf("%s: got %+v (%v), want %+v", label, got, err, c.want)
		}
	}
}
