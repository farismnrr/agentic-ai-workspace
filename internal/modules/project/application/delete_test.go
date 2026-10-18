package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/domain"
)

var errStuck = errors.New("tasks would not stop")

type store struct {
	application.Store
	owner   string
	deleted bool
}

func (s *store) Find(_ context.Context, userID, id string) (domain.Project, error) {
	if userID != s.owner || id != "p1" {
		return domain.Project{}, domain.ErrNotFound
	}

	return domain.Project{ID: id, UserID: userID}, nil
}

func (s *store) Delete(context.Context, string, string) error {
	s.deleted = true

	return nil
}

type folders struct {
	application.Folders
	removed []string
}

func (f *folders) Remove(id string) error {
	f.removed = append(f.removed, id)

	return nil
}

func newService(owner string) (*application.Service, *store, *folders) {
	data, files := &store{owner: owner}, &folders{}

	return application.NewService(data, files, nil, nil, time.Now), data, files
}

func TestDeletingAProjectCleansUpThenRemovesItAndItsFolder(t *testing.T) {
	t.Parallel()

	service, data, files := newService("me")
	ran := false

	service.BeforeDelete(func(_ context.Context, userID, id string) error {
		ran = userID == "me" && id == "p1" && !data.deleted

		return nil
	})

	if err := service.Delete(context.Background(), "me", "p1"); err != nil {
		t.Fatal(err)
	}

	if !ran || !data.deleted || len(files.removed) != 1 || files.removed[0] != "p1" {
		t.Fatalf("ran=%v deleted=%v removed=%v", ran, data.deleted, files.removed)
	}
}

func TestAProjectThatIsNotTheirsOrWhoseCleanupFailsIsKept(t *testing.T) {
	t.Parallel()

	service, data, files := newService("me")
	service.BeforeDelete(func(context.Context, string, string) error { return errStuck })

	if err := service.Delete(context.Background(), "someone", "p1"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("another person's project: %v", err)
	}

	if err := service.Delete(context.Background(), "me", "p1"); err == nil {
		t.Fatal("a failed cleanup must stop the delete")
	}

	if data.deleted || len(files.removed) != 0 {
		t.Fatalf("deleted=%v removed=%v", data.deleted, files.removed)
	}
}
