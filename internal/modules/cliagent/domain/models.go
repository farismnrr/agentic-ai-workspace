package domain

import "slices"

// DefaultModel is the choice that leaves the model to the program and the person's plan.
const DefaultModel = "default"

// Model is a choice the program accepts for its model, and the effort levels it takes with it.
type Model struct {
	ID      string
	Name    string
	Efforts []string
	// ContextWindow is how many tokens the model holds at once, when the program says; zero when it does not.
	ContextWindow int64
}

// Claude Code lists no models a person can reach, so these are its own names, from its documentation: the
// model aliases (each one follows the newest model of its family) and its effort levels.
var claudeEfforts = []string{"low", "medium", "high", "xhigh", "max"}

// ClaudeModels are the choices with Claude Code, its own choice first.
func ClaudeModels() []Model {
	return []Model{
		{ID: DefaultModel, Name: "Default", Efforts: claudeEfforts},
		{ID: "fable", Name: "Fable", Efforts: claudeEfforts},
		{ID: "opus", Name: "Opus", Efforts: claudeEfforts},
		{ID: "sonnet", Name: "Sonnet", Efforts: claudeEfforts},
		{ID: "haiku", Name: "Haiku", Efforts: claudeEfforts},
	}
}

// OwnChoice is all there is to pick from a program whose models cannot be listed: its own.
func OwnChoice() []Model {
	return []Model{{ID: DefaultModel, Name: "Default", Efforts: []string{"low", "medium", "high"}}}
}

// Choice checks a model and an effort a person asked for against the models the program lists. It returns
// what to pass to the program: an empty model or effort means the program's own, which is also what a model
// the list does not name gets.
func Choice(models []Model, model, effort string) (string, string) {
	for _, known := range models {
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
