package application_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

type cleanupNotes struct{ removed []string }

func (n *cleanupNotes) Write(context.Context, string, []application.NoteFile) error { return nil }
func (n *cleanupNotes) RemoveTask(_ context.Context, _, taskID string) error {
	n.removed = append(n.removed, taskID)
	return nil
}

type cleanupShots struct{ removed []string }

func (s *cleanupShots) Save(context.Context, string, []byte) (string, error) { return "", nil }
func (s *cleanupShots) Load(context.Context, string, string) ([]byte, error) { return nil, nil }
func (s *cleanupShots) Remove(_ context.Context, taskID string) error {
	s.removed = append(s.removed, taskID)
	return nil
}

func TestArchiveRestoreKeepsPin(t *testing.T) {
	r := newRig(nil)
	task := r.start(t, "Archive me")
	if _, err := r.service.Pin(t.Context(), "u1", task.ID); err != nil {
		t.Fatal(err)
	}
	results := r.service.ArchiveTasks(t.Context(), "u1", []string{task.ID, task.ID, "missing"})
	if len(results) != 2 || !results[0].OK || results[1].OK {
		t.Fatalf("archive should de-duplicate ids and report unknown ids: %+v", results)
	}
	if results := r.service.RestoreTasks(t.Context(), "u1", []string{task.ID}); !results[0].OK {
		t.Fatalf("restore failed: %+v", results)
	}
	got, err := r.service.Get(t.Context(), "u1", task.ID)
	if err != nil || got.ArchivedAt != nil || got.PinnedAt == nil {
		t.Fatalf("restore should clear archive and keep pin: %+v, %v", got, err)
	}
}

func TestArchivedTaskIsReadOnlyAndListedInArchive(t *testing.T) {
	r := newRig(nil)
	task := r.start(t, "Archive me")
	results := r.service.ArchiveTasks(t.Context(), "u1", []string{task.ID})
	if len(results) != 1 || !results[0].OK {
		t.Fatalf("archive failed: %+v", results)
	}
	if got, _ := r.service.List(t.Context(), "u1", ""); len(got) != 0 {
		t.Fatalf("archived task remained in active list: %+v", got)
	}
	archived, err := r.service.Archived(t.Context(), "u1")
	if err != nil || len(archived) != 1 || archived[0].ArchivedAt == nil {
		t.Fatalf("archived task missing: %+v, %v", archived, err)
	}
	err = r.service.FollowUp(t.Context(), "u1", task.ID, "Continue")
	if !errors.Is(err, domain.ErrWrongState) {
		t.Fatalf("archived task accepted input: %v", err)
	}
	if _, err = r.service.Get(t.Context(), "u1", task.ID); err != nil {
		t.Fatalf("archived task should remain readable: %v", err)
	}
}

func TestArchiveStopsRunningTaskBeforeHidingIt(t *testing.T) {
	r := newRig(nil)
	task := r.start(t, "Running task")
	if err := r.mem.SetStatus(context.Background(), task.ID, domain.StatusWorking, time.Now()); err != nil {
		t.Fatal(err)
	}
	r.disp.cancel = 1 // CancelRunning
	result := r.service.ArchiveTasks(t.Context(), "u1", []string{task.ID})
	if len(result) != 1 || !result[0].OK || len(r.disp.canceled) != 1 || r.disp.canceled[0] != task.ID {
		t.Fatalf("archive failed: %+v", result)
	}
}

func TestDeleteAllIncludesMoreThanTheNormalListLimitAndArchivedTasks(t *testing.T) {
	r := newRig(nil)
	for i := 0; i < 250; i++ {
		task := domain.Task{
			ID: fmt.Sprintf("task-%d", i), UserID: "u1", ProjectID: "proj", Title: "Task",
			Status: domain.StatusDone, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		}
		if err := r.mem.Create(t.Context(), task); err != nil {
			t.Fatal(err)
		}
		if i%2 == 0 {
			if err := r.mem.Archive(t.Context(), "u1", task.ID, time.Now()); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := r.service.DeleteAll(t.Context(), "u1"); err != nil {
		t.Fatal(err)
	}
	remaining, err := r.mem.All(t.Context(), "u1", "", -1)
	if err != nil || len(remaining) != 0 {
		t.Fatalf("DeleteAll left %d tasks: %v", len(remaining), err)
	}
}

func TestDeleteStopsRunningTaskAndHandlesBatchesBeyond500(t *testing.T) {
	r := newRig(nil)
	ids := make([]string, 1001)
	for i := range ids {
		ids[i] = fmt.Sprintf("task-%d", i)
		err := r.mem.Create(t.Context(), domain.Task{
			ID: ids[i], UserID: "u1", ProjectID: "proj", Title: "Task",
			Status: domain.StatusDone, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := r.mem.SetStatus(t.Context(), ids[0], domain.StatusWorking, time.Now()); err != nil {
		t.Fatal(err)
	}
	r.disp.cancel = 1 // CancelRunning
	results := r.service.DeleteTasks(t.Context(), "u1", ids)
	if len(results) != len(ids) {
		t.Fatalf("got %d results for %d ids", len(results), len(ids))
	}
	for _, result := range results {
		if !result.OK {
			t.Fatalf("batch item failed: %+v", result)
		}
	}
	if len(r.disp.canceled) != 1 || r.disp.canceled[0] != ids[0] {
		t.Fatalf("running task was not stopped first: %v", r.disp.canceled)
	}
	if tasks, _ := r.mem.All(t.Context(), "u1", "", -1); len(tasks) != 0 {
		t.Fatalf("batch left %d tasks", len(tasks))
	}
}

func TestPermanentDeleteCleansTaskFiles(t *testing.T) {
	notes, shots := &cleanupNotes{}, &cleanupShots{}
	r := newRigWith(nil, func(deps *application.Deps, _ *application.Limits) {
		deps.Notes, deps.Shots = notes, shots
	})
	task := domain.Task{
		ID: "task-file-id", UserID: "u1", ProjectID: "proj", Title: "Task",
		Status: domain.StatusDone, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	if err := r.mem.Create(t.Context(), task); err != nil {
		t.Fatal(err)
	}
	results := r.service.DeleteTasks(t.Context(), "u1", []string{task.ID})
	if len(results) != 1 || !results[0].OK || len(notes.removed) != 1 || len(shots.removed) != 1 {
		t.Fatalf("permanent delete did not clean task files: %+v, notes=%v, shots=%v", results, notes.removed, shots.removed)
	}
}
