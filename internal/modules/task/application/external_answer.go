package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// QuestionAnswerStore resolves external question cards with owner and project binding.
type QuestionAnswerStore interface {
	FindQuestion(context.Context, string, string, string, string, string) (WorkQuestion, domain.Event, error)
	AnswerExternalQuestion(context.Context, string, string, string, string, string, [][]string, string) (bool, error)
}

func (s *Service) answerExternal(ctx context.Context, userID string, task domain.Task, eventID string,
	answer Answer,
) error {
	if s.externalQuestions == nil {
		return domain.ErrNotWaiting
	}
	_, event, err := s.externalQuestions.FindQuestion(ctx, userID, task.ProjectID, task.ID, task.WorkID, eventID)
	if err != nil {
		return domain.ErrNotFound
	}
	summary, err := answerQuestion(event, answer)
	if err != nil {
		return err
	}
	answers := answer.Answers
	if len(answers) == 0 {
		answers = [][]string{{answer.Choice}}
	}
	answers = trimAnswerValues(answers)
	_, err = s.externalQuestions.AnswerExternalQuestion(ctx, userID, task.ProjectID, task.ID,
		task.WorkID, eventID, answers, summary)
	if err == nil && s.deps.Notify != nil {
		s.deps.Notify.Changed(userID, task.ID)
	}
	if err != nil {
		return fmt.Errorf("save external question answer: %w", err)
	}
	return nil
}

func trimAnswerValues(answers [][]string) [][]string {
	cleaned := make([][]string, len(answers))
	for question, values := range answers {
		for _, value := range values {
			if value = strings.TrimSpace(value); value != "" {
				cleaned[question] = append(cleaned[question], value)
			}
		}
	}
	return cleaned
}
