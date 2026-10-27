package application

import (
	"context"
	"encoding/json"
	"fmt"
)

type answeredPlanQuestion struct {
	Question string   `json:"question"`
	Answer   []string `json:"answer"`
}

// qaSnapshot freezes answered questions into the submitted plan revision.
func (w *Works) qaSnapshot(ctx context.Context, workID string) (string, error) {
	if w.questions == nil {
		return "", nil
	}
	questions, err := w.questions.Questions(ctx, workID)
	if err != nil {
		return "", fmt.Errorf("read answers for plan snapshot: %w", err)
	}
	answered := make([]answeredPlanQuestion, 0, len(questions))
	for _, question := range questions {
		if question.State != "answered" {
			continue
		}
		for index, item := range question.Form {
			if index >= len(question.Answers) {
				continue
			}
			answered = append(answered, answeredPlanQuestion{
				Question: item.Text, Answer: append([]string{}, question.Answers[index]...),
			})
		}
	}
	if len(answered) == 0 {
		return "", nil
	}
	encoded, err := json.Marshal(answered)
	if err != nil {
		return "", fmt.Errorf("encode plan question snapshot: %w", err)
	}
	return string(encoded), nil
}
