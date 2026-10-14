package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/chat/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/chat/domain"
)

type memoryChatStore struct {
	chats    map[string]domain.Chat
	messages map[string][]domain.Message
}

func newMemoryChatStore() *memoryChatStore {
	return &memoryChatStore{
		chats:    make(map[string]domain.Chat),
		messages: make(map[string][]domain.Message),
	}
}

func (m *memoryChatStore) FindOrCreate(_ context.Context, userID, projectID string) (domain.Chat, error) {
	key := userID + ":" + projectID
	if c, ok := m.chats[key]; ok {
		return c, nil
	}
	c := domain.Chat{
		ID:        "chat-1",
		ProjectID: projectID,
		UserID:    userID,
		Title:     "Project Chat",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.chats[key] = c
	return c, nil
}

func (m *memoryChatStore) Get(_ context.Context, userID, chatID string) (domain.Chat, error) {
	for _, c := range m.chats {
		if c.ID == chatID && c.UserID == userID {
			return c, nil
		}
	}
	return domain.Chat{}, domain.ErrNotFound
}

func (m *memoryChatStore) Messages(_ context.Context, chatID string) ([]domain.Message, error) {
	return m.messages[chatID], nil
}

func (m *memoryChatStore) Append(_ context.Context, msg domain.Message) error {
	m.messages[msg.ChatID] = append(m.messages[msg.ChatID], msg)
	return nil
}

func (m *memoryChatStore) Clear(_ context.Context, chatID string) error {
	m.messages[chatID] = nil
	return nil
}

func (m *memoryChatStore) Touch(_ context.Context, chatID string, at time.Time) error {
	for k, c := range m.chats {
		if c.ID == chatID {
			c.UpdatedAt = at
			m.chats[k] = c
		}
	}
	return nil
}

type fakeModel struct {
	reply string
}

func (f fakeModel) Complete(_ context.Context, req application.ModelRequest) (string, error) {
	if req.OnText != nil {
		req.OnText(f.reply)
	}
	return f.reply, nil
}

type fakeModels struct {
	model fakeModel
}

func (f fakeModels) Client(_ context.Context, _, _ string) (application.Model, error) {
	return f.model, nil
}

func (f fakeModels) Default(_ context.Context, _ string) (string, string, error) {
	return "provider-1", "model-1", nil
}

type fakeProjects struct{}

func (fakeProjects) Name(_ context.Context, _, _ string) (string, error) {
	return "Test Project", nil
}

type fakePersona struct{}

func (fakePersona) Persona(_ context.Context, _ string) (application.Persona, error) {
	return application.Persona{Name: "Awam"}, nil
}

type fakeTaskCreator struct {
	createdInput application.TaskInput
}

func (f *fakeTaskCreator) CreateTask(_ context.Context, _ string, in application.TaskInput) (string, error) {
	f.createdInput = in
	return "task-123", nil
}

type fixedIDs struct{}

func (fixedIDs) ID() (string, error) { return "msg-id", nil }

func newTestService(taskCreator application.TaskCreator) *application.Service {
	return application.NewService(application.Deps{
		Store:       newMemoryChatStore(),
		Models:      fakeModels{model: fakeModel{reply: "Hello from Awam!"}},
		Projects:    fakeProjects{},
		Persona:     fakePersona{},
		TaskCreator: taskCreator,
		IDs:         fixedIDs{},
		Now:         time.Now,
	})
}

func TestConversationAndSend(t *testing.T) {
	service := newTestService(&fakeTaskCreator{})
	ctx := context.Background()

	chat, msgs, err := service.Conversation(ctx, "user-1", "proj-1")
	if err != nil || chat.ID != "chat-1" || len(msgs) != 0 {
		t.Fatalf("expected empty conversation: %v, %+v", err, chat)
	}

	var streamed string
	userMsg, asstMsg, err := service.Send(ctx, "user-1", "proj-1", "Hello!", func(delta string) {
		streamed += delta
	})
	if err != nil || userMsg.Content != "Hello!" || asstMsg.Content != "Hello from Awam!" {
		t.Fatalf("send failed: %v", err)
	}
	if streamed != "Hello from Awam!" {
		t.Errorf("streamed text mismatch: %s", streamed)
	}
}

func TestEscalateToTask(t *testing.T) {
	tc := &fakeTaskCreator{}
	service := newTestService(tc)
	ctx := context.Background()

	taskID, err := service.Escalate(ctx, "user-1", application.EscalateInput{
		ProjectID: "proj-1",
		Prompt:    "Refactor login",
		Mode:      "always",
	})
	if err != nil || taskID != "task-123" {
		t.Fatalf("escalate failed: %v, id=%s", err, taskID)
	}
	if tc.createdInput.Prompt != "Refactor login" {
		t.Errorf("expected custom prompt, got %s", tc.createdInput.Prompt)
	}
}

func TestClear(t *testing.T) {
	service := newTestService(&fakeTaskCreator{})
	ctx := context.Background()

	_, _, err := service.Send(ctx, "user-1", "proj-1", "Hello!", nil)
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}

	if err = service.Clear(ctx, "user-1", "proj-1"); err != nil {
		t.Fatalf("clear failed: %v", err)
	}

	_, msgs, err := service.Conversation(ctx, "user-1", "proj-1")
	if err != nil || len(msgs) != 0 {
		t.Fatalf("expected cleared messages: %v, msgs=%v", err, msgs)
	}
}
