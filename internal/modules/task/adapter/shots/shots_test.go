package shots_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/adapter/shots"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestAScreenshotIsSavedReadAndRemovedWithItsTask(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := shots.New(t.TempDir())

	id, err := store.Save(ctx, "task1", []byte("png"))
	if err != nil {
		t.Fatal(err)
	}

	if got, err := store.Load(ctx, "task1", id); err != nil || !bytes.Equal(got, []byte("png")) {
		t.Fatalf("load = %q, %v", got, err)
	}

	if _, err := store.Load(ctx, "task2", id); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("another task must not read it: %v", err)
	}

	if err := store.Remove(ctx, "task1"); err != nil {
		t.Fatal(err)
	}

	if _, err := store.Load(ctx, "task1", id); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("removed with its task: %v", err)
	}
}

func TestNamesThatLeaveTheFolderAreRefused(t *testing.T) {
	t.Parallel()

	store := shots.New(t.TempDir())

	if _, err := store.Load(context.Background(), "task1", "../../etc/passwd"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("a name that leaves the folder must be refused: %v", err)
	}

	if _, err := store.Save(context.Background(), "../x", []byte("png")); err == nil {
		t.Error("a task id that leaves the folder must be refused")
	}
}
