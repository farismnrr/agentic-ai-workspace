package application

import (
	"context"
	"fmt"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/chat/domain"
)

// Deps holds the collaborators of the chat service.
type Deps struct {
	Store       ChatStore
	Models      Models
	Projects    Projects
	Persona     PersonaSource
	TaskCreator TaskCreator
	IDs         IDs
	Now         func() time.Time
}

// Service provides chat use cases.
type Service struct {
	deps Deps
}

// NewService constructs a chat application service.
func NewService(deps Deps) *Service {
	return &Service{deps: deps}
}

// Conversation returns the active chat and its messages for a project.
func (s *Service) Conversation(ctx context.Context, userID, projectID string) (domain.Chat, []domain.Message, error) {
	if _, err := s.deps.Projects.Name(ctx, userID, projectID); err != nil {
		return domain.Chat{}, nil, domain.ErrNoProject
	}

	chat, err := s.deps.Store.FindOrCreate(ctx, userID, projectID)
	if err != nil {
		return domain.Chat{}, nil, fmt.Errorf("find or create chat: %w", err)
	}

	msgs, err := s.deps.Store.Messages(ctx, chat.ID)
	if err != nil {
		return domain.Chat{}, nil, fmt.Errorf("load messages: %w", err)
	}

	return chat, msgs, nil
}

// Send records a user message and streams the assistant's reply.
func (s *Service) Send(
	ctx context.Context, userID, projectID, text string, onText func(delta string),
) (domain.Message, domain.Message, error) {
	validText, err := domain.ValidateMessage(text)
	if err != nil {
		return domain.Message{}, domain.Message{}, fmt.Errorf("validate message: %w", err)
	}

	chat, msgs, err := s.Conversation(ctx, userID, projectID)
	if err != nil {
		return domain.Message{}, domain.Message{}, err
	}

	userMsg, err := s.saveMessage(ctx, chat.ID, domain.RoleUser, validText)
	if err != nil {
		return domain.Message{}, domain.Message{}, err
	}

	replyText, err := s.generateReply(ctx, userID, projectID, append(msgs, userMsg), onText)
	if err != nil {
		return userMsg, domain.Message{}, err
	}

	assistantMsg, err := s.saveMessage(ctx, chat.ID, domain.RoleAssistant, replyText)
	if err != nil {
		return userMsg, domain.Message{}, err
	}

	_ = s.deps.Store.Touch(ctx, chat.ID, s.deps.Now())

	return userMsg, assistantMsg, nil
}

func (s *Service) saveMessage(
	ctx context.Context, chatID string, role domain.Role, text string,
) (domain.Message, error) {
	id, err := s.deps.IDs.ID()
	if err != nil {
		return domain.Message{}, fmt.Errorf("generate message id: %w", err)
	}

	msg := domain.Message{
		ID:        id,
		ChatID:    chatID,
		Role:      role,
		Content:   text,
		CreatedAt: s.deps.Now(),
	}

	if err := s.deps.Store.Append(ctx, msg); err != nil {
		return domain.Message{}, fmt.Errorf("save message: %w", err)
	}

	return msg, nil
}

func (s *Service) generateReply(
	ctx context.Context, userID, projectID string, history []domain.Message, onText func(string),
) (string, error) {
	providerID, model, err := s.deps.Models.Default(ctx, userID)
	if err != nil || providerID == "" || model == "" {
		return "", domain.ErrNoModel
	}

	client, err := s.deps.Models.Client(ctx, userID, providerID)
	if err != nil {
		return "", domain.ErrNoModel
	}

	projectName, _ := s.deps.Projects.Name(ctx, userID, projectID)
	persona, _ := s.deps.Persona.Persona(ctx, userID)
	sys := chatSystemPrompt(projectName, persona)

	modelMsgs := make([]ModelMessage, 0, len(history))
	for _, m := range history {
		modelMsgs = append(modelMsgs, ModelMessage{Role: m.Role, Content: m.Content})
	}

	reply, err := client.Complete(ctx, ModelRequest{
		Model:    model,
		System:   sys,
		Messages: modelMsgs,
		OnText:   onText,
	})
	if err != nil {
		return "", fmt.Errorf("complete chat: %w", err)
	}

	return reply, nil
}

// Clear removes all messages from a project's chat.
func (s *Service) Clear(ctx context.Context, userID, projectID string) error {
	chat, _, err := s.Conversation(ctx, userID, projectID)
	if err != nil {
		return err
	}

	if err := s.deps.Store.Clear(ctx, chat.ID); err != nil {
		return fmt.Errorf("clear chat: %w", err)
	}

	return nil
}
