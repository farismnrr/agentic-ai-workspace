package application

import (
	"encoding/json"
	"strconv"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// Names of the tools the agent loop handles itself.
const (
	toolUpdatePlan = "update_plan"
	toolAskUser    = "ask_user"
	toolFinish     = "finish"
)

const (
	maxPlanSteps = 12
	maxText      = 600
	maxOptions   = 4
	maxChecks    = 8
)

func agentTools() []domain.ToolDef {
	return []domain.ToolDef{
		{
			Name:        toolUpdatePlan,
			Description: "Set or update the plan shown to the person: 3 to 6 short steps in plain words, each with a state.",
			Schema: json.RawMessage(`{"type":"object","properties":{"steps":{"type":"array","items":{"type":"object",` +
				`"properties":{"title":{"type":"string"},"detail":{"type":"string"},` +
				`"state":{"type":"string","enum":["pending","active","done","failed"]}},"required":["title","state"]}}},` +
				`"required":["steps"]}`),
		},
		{
			Name:        toolAskUser,
			Description: "Ask the person a question and wait for the answer. Offer 2 to 4 short options when possible.",
			Schema: json.RawMessage(`{"type":"object","properties":{"question":{"type":"string"},` +
				`"options":{"type":"array","items":{"type":"string"}}},"required":["question"]}`),
		},
		{
			Name:        toolFinish,
			Description: "Call when the work is done. Summarise in plain words and list the checks you actually ran.",
			Schema: json.RawMessage(`{"type":"object","properties":{"summary":{"type":"string"},` +
				`"checks":{"type":"array","items":{"type":"string"}}},"required":["summary"]}`),
		},
	}
}

type planArgs struct {
	Steps []struct {
		Title  string `json:"title"`
		Detail string `json:"detail"`
		State  string `json:"state"`
	} `json:"steps"`
}

// parsePlan turns update_plan input into plan steps with stable ids.
func parsePlan(input json.RawMessage) ([]domain.PlanStep, bool) {
	var args planArgs
	if err := json.Unmarshal(input, &args); err != nil || len(args.Steps) == 0 || len(args.Steps) > maxPlanSteps {
		return nil, false
	}

	steps := make([]domain.PlanStep, 0, len(args.Steps))

	for index, step := range args.Steps {
		state := domain.PlanState(step.State)
		if state != domain.PlanActive && state != domain.PlanDone && state != domain.PlanFailed {
			state = domain.PlanPending
		}

		steps = append(steps, domain.PlanStep{
			ID: "p" + strconv.Itoa(index+1), Title: clip(step.Title), Detail: clip(step.Detail), State: state,
		})
	}

	return steps, true
}

type askArgs struct {
	Question string   `json:"question"`
	Options  []string `json:"options"`
}

func parseAsk(input json.RawMessage) (askArgs, bool) {
	var args askArgs
	if err := json.Unmarshal(input, &args); err != nil || args.Question == "" {
		return askArgs{}, false
	}

	args.Question = clip(args.Question)
	if len(args.Options) > maxOptions {
		args.Options = args.Options[:maxOptions]
	}

	for i := range args.Options {
		args.Options[i] = clip(args.Options[i])
	}

	return args, true
}

type finishArgs struct {
	Summary string   `json:"summary"`
	Checks  []string `json:"checks"`
}

func parseFinish(input json.RawMessage) finishArgs {
	var args finishArgs
	if err := json.Unmarshal(input, &args); err != nil {
		return finishArgs{}
	}

	args.Summary = clip(args.Summary)
	if len(args.Checks) > maxChecks {
		args.Checks = args.Checks[:maxChecks]
	}

	return args
}

func clip(text string) string {
	runes := []rune(text)
	if len(runes) <= maxText {
		return text
	}

	return string(runes[:maxText]) + "…"
}
