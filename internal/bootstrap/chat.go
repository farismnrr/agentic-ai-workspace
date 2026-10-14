package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	chathttp "github.com/farismnrr/agentic-ai-workspace/internal/modules/chat/adapter/http"
	chatsqlite "github.com/farismnrr/agentic-ai-workspace/internal/modules/chat/adapter/sqlite"
	chatapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/chat/application"
	chatdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/chat/domain"
	projectapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/project/application"
	taskapp "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/application"
	taskdomain "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/httpserver"
	"github.com/farismnrr/agentic-ai-workspace/internal/platform/opaque"
)

type chatModelBridge struct {
	model taskapp.Model
}

func (b chatModelBridge) Complete(ctx context.Context, req chatapp.ModelRequest) (string, error) {
	messages := make([]taskdomain.Message, 0, len(req.Messages))
	for _, m := range req.Messages {
		role := taskdomain.RoleUser
		if m.Role == chatdomain.RoleAssistant {
			role = taskdomain.RoleAssistant
		}
		messages = append(messages, taskdomain.Message{
			Role:   role,
			Blocks: []taskdomain.Block{{Kind: taskdomain.BlockText, Text: m.Content}},
		})
	}

	reply, err := b.model.Complete(ctx, taskapp.Request{
		Model:    req.Model,
		System:   req.System,
		Messages: messages,
		OnText:   req.OnText,
	})
	if err != nil {
		return "", fmt.Errorf("model complete: %w", err)
	}

	var texts []string
	for _, blk := range reply.Blocks {
		if blk.Kind == taskdomain.BlockText {
			texts = append(texts, blk.Text)
		}
	}

	return strings.Join(texts, "\n"), nil
}

type chatModelsBridge struct {
	models agentModels
}

func (b chatModelsBridge) Client(ctx context.Context, userID, providerID string) (chatapp.Model, error) {
	client, err := b.models.Client(ctx, userID, providerID)
	if err != nil {
		return nil, err
	}
	return chatModelBridge{model: client}, nil
}

func (b chatModelsBridge) Default(ctx context.Context, userID string) (string, string, error) {
	return b.models.Default(ctx, userID)
}

type chatPersonaBridge struct {
	persona agentPersona
}

func (b chatPersonaBridge) Persona(ctx context.Context, userID string) (chatapp.Persona, error) {
	p, err := b.persona.Persona(ctx, userID)
	if err != nil {
		return chatapp.Persona{}, err
	}
	return chatapp.Persona{
		Name: p.Name, TalkStyle: p.TalkStyle, Language: p.Language, About: p.About, Rules: p.Rules,
	}, nil
}

type chatProjectsBridge struct {
	projects agentProjects
}

func (b chatProjectsBridge) Name(ctx context.Context, userID, projectID string) (string, error) {
	return b.projects.Name(ctx, userID, projectID)
}

type chatTaskCreatorBridge struct {
	tasks *taskapp.Service
}

func (b chatTaskCreatorBridge) CreateTask(ctx context.Context, userID string, in chatapp.TaskInput) (string, error) {
	task, err := b.tasks.Create(ctx, userID, taskapp.CreateInput{
		ProjectID: in.ProjectID,
		Prompt:    in.Prompt,
		Mode:      taskdomain.Mode(in.Mode),
	})
	if err != nil {
		return "", fmt.Errorf("create task from chat: %w", err)
	}
	return task.ID, nil
}

func newChat(
	db *sql.DB, models modelsWiring, projects *projectapp.Service, tasks *taskapp.Service,
	auth, sameOrigin fiber.Handler,
) (*chatapp.Service, httpserver.Mount) {
	store := chatsqlite.NewStore(db, opaque.Generator{})
	service := chatapp.NewService(chatapp.Deps{
		Store:       store,
		Models:      chatModelsBridge{models: agentModels{providers: models.providers, prefs: models.prefs}},
		Persona:     chatPersonaBridge{persona: agentPersona{prefs: models.prefs}},
		Projects:    chatProjectsBridge{projects: agentProjects{projects: projects}},
		TaskCreator: chatTaskCreatorBridge{tasks: tasks},
		IDs:         opaque.Generator{},
		Now:         time.Now,
	})

	handler := chathttp.NewHandler(service)
	return service, handler.Mount(auth, sameOrigin)
}
