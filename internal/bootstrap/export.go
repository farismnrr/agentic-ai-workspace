package bootstrap

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/gofiber/fiber/v3"

	accountapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/account/application"
	mydatahttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/mydata/adapter/http"
	mydataapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/mydata/application"
	prefapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/preference/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/project/adapter/folder"
	projectapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/application"
	projectdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/domain"
	providerapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/provider/application"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
)

// newExport wires GET /api/account/export: profile and settings, every
// project with its files, tasks with their threads, and usage.
func newExport(
	account *accountapp.Service, models modelsWiring, projects *projectapp.Service, tasks *taskapp.Service,
	folders *folder.Folders, auth fiber.Handler,
) httpserver.Mount {
	exporter := mydataapp.NewExporter(
		settingsExport{account: account, prefs: models.prefs, providers: models.providers},
		projectsExport{projects: projects, tasks: tasks, folders: folders},
		usageExport{tasks: tasks},
	)

	return mydatahttp.NewHandler(exporter, time.Now).Mount(auth)
}

func writeJSON(out mydataapp.Archive, path string, value any) error {
	file, err := out.Create(path)
	if err != nil {
		return fmt.Errorf("start %s: %w", path, err)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(value); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}

// settingsExport is the profile, preferences and providers. API keys are never exported.
type settingsExport struct {
	account   *accountapp.Service
	prefs     *prefapp.Service
	providers *providerapp.Service
}

func (s settingsExport) Write(ctx context.Context, userID string, out mydataapp.Archive) error {
	user, err := s.account.Profile(ctx, userID)
	if err != nil {
		return fmt.Errorf("profile: %w", err)
	}

	prefs, err := s.prefs.Get(ctx, userID)
	if err != nil {
		return fmt.Errorf("preferences: %w", err)
	}

	providers, err := s.providers.List(ctx, userID)
	if err != nil {
		return fmt.Errorf("providers: %w", err)
	}

	return writeJSON(out, "account.json", map[string]any{
		"name": user.Name, "email": user.Email, "createdAt": user.CreatedAt,
		"preferences": prefs, "providers": providers,
	})
}

// usageExport is token usage per model.
type usageExport struct{ tasks *taskapp.Service }

func (u usageExport) Write(ctx context.Context, userID string, out mydataapp.Archive) error {
	rows, err := u.tasks.Usage(ctx, userID, time.Time{})
	if err != nil {
		return fmt.Errorf("usage: %w", err)
	}

	return writeJSON(out, "usage.json", rows)
}

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// folderName names a project's folder in the archive, readable and unique.
func folderName(name, id string) string {
	return unsafeName.ReplaceAllString(name, "-") + "-" + id
}

// projectsExport is every project: details, files, and its tasks with their threads.
type projectsExport struct {
	projects *projectapp.Service
	tasks    *taskapp.Service
	folders  *folder.Folders
}

func (p projectsExport) Write(ctx context.Context, userID string, out mydataapp.Archive) error {
	projects, err := p.projects.Everything(ctx, userID)
	if err != nil {
		return fmt.Errorf("projects: %w", err)
	}

	tasks, err := p.tasks.Everything(ctx, userID)
	if err != nil {
		return fmt.Errorf("tasks: %w", err)
	}

	for _, project := range projects {
		if err := p.writeProject(ctx, userID, project, tasks, out); err != nil {
			return err
		}
	}

	return nil
}

func (p projectsExport) writeProject(
	ctx context.Context, userID string, project projectdomain.Project, tasks []taskdomain.Task, out mydataapp.Archive,
) error {
	base := "projects/" + folderName(project.Name, project.ID)
	if err := writeJSON(out, base+"/project.json", project); err != nil {
		return err
	}

	if err := p.writeFiles(project.ID, base+"/files", out); err != nil {
		return err
	}

	for _, task := range tasks {
		if task.ProjectID != project.ID {
			continue
		}

		if err := p.writeTask(ctx, userID, base, out, task); err != nil {
			return err
		}
	}

	return nil
}

func (p projectsExport) writeTask(
	ctx context.Context, userID, base string, out mydataapp.Archive, task taskdomain.Task,
) error {
	events, err := p.tasks.Events(ctx, userID, task.ID, 0)
	if err != nil {
		return fmt.Errorf("events: %w", err)
	}

	return writeJSON(out, base+"/tasks/"+task.ID+".json", map[string]any{"task": task, "events": events})
}
