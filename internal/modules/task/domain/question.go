package domain

import "strings"

// QuestionType is how the person answers one question of a form.
type QuestionType string

// Question types: one option (radio), several options (checkbox) or typed text.
const (
	QuestionSingle QuestionType = "single"
	QuestionMulti  QuestionType = "multi"
	QuestionText   QuestionType = "text"
)

// Valid reports whether the type is one the form knows how to show.
func (t QuestionType) Valid() bool {
	return t == QuestionSingle || t == QuestionMulti || t == QuestionText
}

// Option is one choice of a question; Description says what it means.
type Option struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	// Preview names a picture the form shows with the option, such as a style family; the page knows the names.
	Preview string `json:"preview,omitempty"`
}

// Question is one decision put to the person. A form holds one to four of them. Besides the options the
// person can always type their own answer.
type Question struct {
	Header  string       `json:"header,omitempty"`
	Text    string       `json:"text"`
	Type    QuestionType `json:"type"`
	Options []Option     `json:"options,omitempty"`
}

// Summary writes an answer as one readable line for the transcript. A form with several questions names each
// one by its header; a single question is just the answer.
func Summary(questions []Question, answers [][]string) string {
	parts := make([]string, 0, len(questions))

	for index, question := range questions {
		value := strings.Join(answers[index], ", ")
		if len(questions) == 1 {
			return value
		}

		name := question.Header
		if name == "" {
			name = question.Text
		}

		parts = append(parts, name+": "+value)
	}

	return strings.Join(parts, "; ")
}
