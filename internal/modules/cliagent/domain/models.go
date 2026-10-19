package domain

import "slices"

// DefaultModel is the choice that leaves the model to the program and the person's plan.
const DefaultModel = "default"

// Model is a choice the program accepts for its model, and the effort levels it takes with it.
type Model struct {
	ID      string
	Name    string
	Efforts []string
}

// The vendors list no models a program can reach, so these are the program's own names and levels, from its
// documentation: Claude Code's model aliases (each one follows the newest model of its family) and its effort
// levels. Codex is left to choose its own model; the levels are the ones every Codex model takes.
var (
	claudeEfforts = []string{"low", "medium", "high", "xhigh", "max"}
	codexEfforts  = []string{"low", "medium", "high"}
)

// Models are the choices a person has with a program, the program's own choice first.
func Models(rt Runtime) []Model {
	if rt == ChatGPT {
		return []Model{{ID: DefaultModel, Name: "Default", Efforts: codexEfforts}}
	}

	return []Model{
		{ID: DefaultModel, Name: "Default", Efforts: claudeEfforts},
		{ID: "fable", Name: "Fable", Efforts: claudeEfforts},
		{ID: "opus", Name: "Opus", Efforts: claudeEfforts},
		{ID: "sonnet", Name: "Sonnet", Efforts: claudeEfforts},
		{ID: "haiku", Name: "Haiku", Efforts: claudeEfforts},
	}
}

// Choice checks a model and an effort a person asked for against what the program accepts. It returns what to
// pass to the program: an empty model or effort means the program's own.
func Choice(rt Runtime, model, effort string) (string, string) {
	for _, known := range Models(rt) {
		if known.ID != model {
			continue
		}

		if model == DefaultModel {
			model = ""
		}

		if slices.Contains(known.Efforts, effort) {
			return model, effort
		}

		return model, ""
	}

	return "", ""
}
