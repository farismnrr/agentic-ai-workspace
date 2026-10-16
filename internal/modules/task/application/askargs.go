package application

import (
	"encoding/json"
	"fmt"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

const (
	maxQuestions = 4
	maxHeader    = 24
)

// askSchema is the ask_user input: a form of one to four questions.
const askSchema = `{"type":"object","properties":{"questions":{"type":"array","minItems":1,"maxItems":4,` +
	`"items":{"type":"object","properties":{"header":{"type":"string","description":"One or two words naming it"},` +
	`"question":{"type":"string"},"type":{"type":"string","enum":["single","multi","text"],` +
	`"description":"single = pick one, multi = pick several, text = typed answer"},` +
	`"options":{"type":"array","maxItems":4,"items":{"type":"object","properties":{"label":{"type":"string"},` +
	`"description":{"type":"string"}},"required":["label"]}}},"required":["question","type"]}}},"required":["questions"]}`

// optionArg reads an option written as a plain string or as {label, description}.
type optionArg struct{ domain.Option }

func (o *optionArg) UnmarshalJSON(data []byte) error {
	var label string
	if json.Unmarshal(data, &label) == nil {
		o.Label = label

		return nil
	}

	if err := json.Unmarshal(data, &o.Option); err != nil {
		return fmt.Errorf("read option: %w", err)
	}

	return nil
}

type questionArg struct {
	Header   string      `json:"header"`
	Question string      `json:"question"`
	Type     string      `json:"type"`
	Options  []optionArg `json:"options"`
}

// askArgs accepts the form, and the one-question shape {question, options[]} older prompts still produce.
type askArgs struct {
	questionArg
	Questions []questionArg `json:"questions"`
}

// parseAsk turns ask_user input into a form of one to four questions.
func parseAsk(input json.RawMessage) ([]domain.Question, bool) {
	var args askArgs
	if err := json.Unmarshal(input, &args); err != nil {
		return nil, false
	}

	list := args.Questions
	if len(list) == 0 {
		list = []questionArg{args.questionArg}
	}

	if len(list) > maxQuestions {
		list = list[:maxQuestions]
	}

	questions := make([]domain.Question, 0, len(list))

	for _, item := range list {
		question, ok := cleanQuestion(item)
		if !ok {
			return nil, false
		}

		questions = append(questions, question)
	}

	return questions, true
}

func cleanQuestion(item questionArg) (domain.Question, bool) {
	if item.Question == "" {
		return domain.Question{}, false
	}

	kind := domain.QuestionType(item.Type)
	if !kind.Valid() {
		kind = domain.QuestionSingle
	}

	question := domain.Question{Header: clipTo(item.Header, maxHeader), Text: clip(item.Question), Type: kind}

	if kind == domain.QuestionText {
		return question, true
	}

	for _, option := range item.Options {
		if option.Label == "" || len(question.Options) == maxOptions {
			continue
		}

		question.Options = append(question.Options, domain.Option{
			Label: clipTo(option.Label, maxText), Description: clip(option.Description),
		})
	}

	if len(question.Options) == 0 {
		question.Type = domain.QuestionText
	}

	return question, true
}

// optionLabels lists the labels of a question's options, the shape questions had before forms.
func optionLabels(question domain.Question) []string {
	labels := make([]string, 0, len(question.Options))
	for _, option := range question.Options {
		labels = append(labels, option.Label)
	}

	return labels
}
