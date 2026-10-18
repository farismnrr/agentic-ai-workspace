package application

import (
	"strings"
	"unicode/utf8"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// maxReplies is how many options one multi-select question can carry back.
const maxReplies = 8

// answerQuestion checks a reply against the form the agent asked and writes it as one line. A question saved
// before forms has no form to check, so its single reply is taken as it is.
func answerQuestion(event domain.Event, answer Answer) (string, error) {
	answers := answer.Answers
	if len(answers) == 0 && strings.TrimSpace(answer.Choice) != "" {
		answers = [][]string{{answer.Choice}}
	}

	questions := event.Questions
	if len(questions) == 0 {
		questions = []domain.Question{{Type: domain.QuestionText}}
	}

	if len(answers) != len(questions) {
		return "", domain.ErrBadRequest
	}

	cleaned := make([][]string, len(questions))

	for index, question := range questions {
		replies, ok := cleanReplies(answers[index])
		if !ok || (question.Type != domain.QuestionMulti && len(replies) != 1) || len(replies) > maxReplies {
			return "", domain.ErrBadRequest
		}

		cleaned[index] = replies
	}

	return domain.Summary(questions, cleaned), nil
}

func cleanReplies(replies []string) ([]string, bool) {
	out := make([]string, 0, len(replies))

	for _, reply := range replies {
		reply = strings.TrimSpace(reply)
		if reply == "" {
			continue
		}

		if utf8.RuneCountInString(reply) > maxText {
			return nil, false
		}

		out = append(out, reply)
	}

	return out, len(out) > 0
}
