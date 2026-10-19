package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/domain"
)

type namedStore struct {
	application.Store
	names []domain.Named
}

func (s *namedStore) ActiveNames(context.Context) ([]domain.Named, error) { return s.names, nil }

type oneID struct{}

func (oneID) ID() (string, error) { return "new", nil }

type noFolders struct{}

func (noFolders) Create(string) error { return nil }
func (noFolders) Remove(string) error { return nil }

func TestAProjectCannotTakeTheNameOfAnother(t *testing.T) {
	t.Parallel()

	data := &namedStore{names: []domain.Named{{ID: "p1", Name: "Bakery"}}}
	service := application.NewService(data, noFolders{}, nil, oneID{}, time.Now)

	if _, err := service.Create(context.Background(), "me", "bakery", ""); !errors.Is(err, domain.ErrNameTaken) {
		t.Fatalf("a taken name must be refused: %v", err)
	}

	free, suggestion, err := service.Available(context.Background(), "Bakery", "")
	if err != nil || free || suggestion != "Bakery 2" {
		t.Fatalf("free=%v suggestion=%q err=%v", free, suggestion, err)
	}

	if free, _, _ = service.Available(context.Background(), "Bakery", "p1"); !free {
		t.Fatal("a project may keep its own name")
	}
}
