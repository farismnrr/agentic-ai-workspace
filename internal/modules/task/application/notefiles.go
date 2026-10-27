package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// NoteFile is one server-maintained project document, readable by every task in that project.
type NoteFile struct{ Path, Content string }

// NoteFiles writes complete documents inside one confined project folder.
type NoteFiles interface {
	Write(ctx context.Context, projectID string, files []NoteFile) error
	RemoveTask(ctx context.Context, projectID, taskID string) error
}

// ProjectDocument identifies a saved document without copying another task's conversation.
type ProjectDocument struct {
	UpdatedAt time.Time     `json:"updatedAt"`
	Path      string        `json:"path"`
	Title     string        `json:"title"`
	TaskID    string        `json:"taskId"`
	Status    domain.Status `json:"status"`
}

const notesFolder = "project-notes/"

// Documents repairs exports from durable task data, including tasks created before file export existed.
func (s *Service) Documents(ctx context.Context, userID, taskID string) ([]ProjectDocument, error) {
	task, err := s.owned(ctx, userID, taskID)
	if err != nil {
		return nil, err
	}
	return s.projectDocuments(ctx, task)
}

func (s *Service) projectDocuments(ctx context.Context, task domain.Task) ([]ProjectDocument, error) {
	docs := []ProjectDocument{}
	if s.deps.Notes == nil {
		return docs, nil
	}
	tasks, err := s.deps.Tasks.All(ctx, task.UserID, task.ProjectID, -1)
	if err != nil {
		return nil, fmt.Errorf("list project documents: %w", err)
	}
	var index strings.Builder
	index.WriteString("# Project documents\n\nReference material from tasks in this project. " +
		"Check dates and status; prior approval does not approve new work.\n")
	for _, previous := range tasks {
		if previous.ProjectID != task.ProjectID {
			continue
		}
		files, syncErr := syncNotes(ctx, s.deps, previous)
		if syncErr != nil {
			return nil, syncErr
		}
		for _, file := range files {
			docs = append(docs, ProjectDocument{
				Path: file.Path, Title: previous.Title, TaskID: previous.ID,
				Status: previous.Status, UpdatedAt: previous.UpdatedAt,
			})
			fmt.Fprintf(&index, "\n- [%s](%s) — %s; %s; %s", previous.Title, strings.TrimPrefix(file.Path, notesFolder),
				file.Path, previous.Status, previous.UpdatedAt.UTC().Format(time.RFC3339))
		}
	}
	err = s.deps.Notes.Write(ctx, task.ProjectID, []NoteFile{{Path: notesFolder + "index.md", Content: index.String()}})
	if err != nil {
		return nil, fmt.Errorf("write project index: %w", err)
	}
	return docs, nil
}

func syncNotes(ctx context.Context, deps Deps, task domain.Task) ([]NoteFile, error) {
	if deps.Notes == nil {
		return nil, nil
	}
	events, err := deps.Events.EventsAfter(ctx, task.ID, 0)
	if err != nil {
		return nil, fmt.Errorf("read document history: %w", err)
	}
	files := taskNoteFiles(task, events)
	if err = deps.Notes.Write(ctx, task.ProjectID, files); err != nil {
		return nil, fmt.Errorf("export task documents: %w", err)
	}
	return files, nil
}

func taskNoteFiles(task domain.Task, events []domain.Event) []NoteFile {
	base := notesFolder + "tasks/" + task.ID + "/"
	header := fmt.Sprintf("# %s\n\nTask: %s\nStatus: %s\nUpdated: %s\n\n"+
		"Reference material; confirm applicability before reusing.\n\n",
		task.Title, task.ID, task.Status, task.UpdatedAt.UTC().Format(time.RFC3339))
	files := []NoteFile{}
	if len(task.Plan) > 0 {
		var plan strings.Builder
		for _, step := range task.Plan {
			fmt.Fprintf(&plan, "- [%s] %s\n  %s\n", step.State, step.Title, step.Detail)
		}
		files = append(files, NoteFile{Path: base + "plan.md", Content: header + plan.String()})
	}
	if task.PRD != "" {
		files = append(files, NoteFile{Path: base + "requirements.md", Content: header + task.PRD})
	}
	if task.Research != "" {
		files = append(files, NoteFile{Path: base + "research.md", Content: header + task.Research})
	}
	for _, event := range events {
		files = append(files, eventNoteFiles(base, header, event)...)
	}
	return files
}

func eventNoteFiles(base, header string, event domain.Event) []NoteFile {
	if event.Kind == domain.EventResearch || event.Kind == domain.EventPRD {
		return documentEventNoteFiles(base, header, event)
	}
	if event.Kind != domain.EventRequest {
		return nil
	}
	return inputEventNoteFiles(base, event)
}

func documentEventNoteFiles(base, header string, event domain.Event) []NoteFile {
	metadata := fmt.Sprintf("Author: %s\nRecorded: %s\n\n", documentAuthor(event), event.Time.UTC().Format(time.RFC3339))
	return []NoteFile{{
		Path:    fmt.Sprintf("%s%s/%d.md", base, event.Kind, event.Seq),
		Content: header + metadata + event.Technical,
	}}
}

func documentAuthor(event domain.Event) string {
	if event.Agent != "" || event.Work == nil {
		return event.Agent
	}
	author := "external assistant, self-reported"
	if event.Work.Actor == "" && event.Work.RoleName == "" {
		return author
	}
	return author + " (" + strings.Join(nonEmpty(event.Work.Actor, event.Work.RoleName), "; ") + ")"
}

func inputEventNoteFiles(base string, event domain.Event) []NoteFile {
	files := []NoteFile{}
	for _, name := range event.Documents {
		if strings.HasPrefix(name, base+"input/") {
			files = append(files, NoteFile{Path: name})
		}
	}
	return files
}

func nonEmpty(values ...string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func (r *Runner) notesPrompt() string {
	if r.deps.Notes == nil {
		return ""
	}
	return "\nProject documents are indexed in project-notes/index.md. Read the index and relevant plans, research, " +
		"requirements and input documents before planning or delegating. " +
		"Archived documents are reference material. The document explicitly attached to the current user message " +
		"contains that user request. " +
		"Do not overwrite server-maintained project-notes files. Earlier approval never approves new work.\n"
}

func (r *Runner) saveNotes(ctx context.Context, task domain.Task) error {
	current, err := r.deps.Tasks.Find(ctx, "", task.ID)
	if err != nil {
		return fmt.Errorf("read task for document export: %w", err)
	}
	_, err = NewService(r.deps, nil).projectDocuments(ctx, current)
	if err == nil && r.deps.Notify != nil {
		r.deps.Notify.FilesChanged(task.ProjectID)
	}
	return err
}
