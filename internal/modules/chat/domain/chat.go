package domain

import (
	"strings"
	"time"
	"unicode/utf8"
)

// Size limits on chat messages and titles.
const (
	MaxPrompt = 4000
	MaxTitle  = 60
)

// Role of a chat message sender.
type Role string

// Message roles.
const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// Message is one message in a conversation.
type Message struct {
	CreatedAt time.Time `json:"createdAt"`
	ID        string    `json:"id"`
	ChatID    string    `json:"chatId"`
	Role      Role      `json:"role"`
	Content   string    `json:"content"`
}

// Chat is a lightweight conversation inside a project.
type Chat struct {
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	ID        string    `json:"id"`
	ProjectID string    `json:"projectId"`
	UserID    string    `json:"userId"`
	Title     string    `json:"title"`
}

// TitleOf creates a short label for a chat from its first message.
func TitleOf(prompt string) string {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return "Chat"
	}

	lines := strings.Split(prompt, "\n")
	line := strings.TrimSpace(lines[0])
	runes := []rune(line)
	if len(runes) > MaxTitle {
		return string(runes[:MaxTitle-1]) + "…"
	}

	return line
}

// ValidateMessage checks text length and non-emptiness.
func ValidateMessage(text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" || utf8.RuneCountInString(text) > MaxPrompt {
		return "", ErrBadRequest
	}

	return text, nil
}
