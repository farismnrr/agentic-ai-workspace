package application

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestParseAskReadsAFormAndTheOldShape(t *testing.T) {
	form := json.RawMessage(`{"questions":[{"header":"Kind","question":"What web?","type":"single",` +
		`"options":[{"label":"Landing","description":"One page"},"Shop"]},` +
		`{"question":"Which parts?","type":"multi","options":[{"label":"Login"}]},{"question":"Name?","type":"text"}]}`)

	questions, ok := parseAsk(form)
	if !ok || len(questions) != 3 || questions[0].Options[0].Description != "One page" ||
		questions[0].Options[1].Label != "Shop" {
		t.Fatalf("form = %+v ok=%v", questions, ok)
	}

	old, ok := parseAsk(json.RawMessage(`{"question":"Start?","options":["Yes","No"]}`))
	if !ok || len(old) != 1 || old[0].Type != domain.QuestionSingle || len(optionLabels(old[0])) != 2 {
		t.Fatalf("old shape = %+v ok=%v", old, ok)
	}

	if _, ok := parseAsk(json.RawMessage(`{"questions":[{"type":"single"}]}`)); ok {
		t.Fatal("a question without text must be refused")
	}
}

func TestAnswerQuestionChecksTheFormAndSummarises(t *testing.T) {
	event := domain.Event{Kind: domain.EventQuestion, Questions: []domain.Question{
		{Header: "Kind", Text: "What?", Type: domain.QuestionSingle},
		{Header: "Parts", Text: "Which?", Type: domain.QuestionMulti},
	}}

	got, err := answerQuestion(event, Answer{Answers: [][]string{{"Shop"}, {"Login", " Cart "}}})
	if err != nil || got != "Kind: Shop; Parts: Login, Cart" {
		t.Fatalf("summary = %q err=%v", got, err)
	}

	for _, bad := range [][][]string{{{"Shop"}}, {{"a", "b"}, {"x"}}, {{"Shop"}, {}}} {
		if _, err := answerQuestion(event, Answer{Answers: bad}); !errors.Is(err, domain.ErrBadRequest) {
			t.Fatalf("%v should be refused, err=%v", bad, err)
		}
	}

	if got, err := answerQuestion(domain.Event{}, Answer{Choice: " Yes "}); err != nil || got != "Yes" {
		t.Fatalf("old question = %q err=%v", got, err)
	}
}

func TestATeammatePromptCarriesItsPlaybook(t *testing.T) {
	playbook := []Skill{{Name: "prd", Body: "# Playbook text"}}
	role := AgentRole{Name: "Planner", Prompt: "persona", Skills: playbook, Tools: []string{toolAskUser}}

	prompt := childPrompt(role, "demo", Persona{}, nil, "")
	if !strings.Contains(prompt, "# Playbook text") || !strings.Contains(prompt, "a form of one to four questions") {
		t.Fatalf("prompt = %q", prompt)
	}
}
