package domain

import "strings"

// planPrefix marks a provider id that stands for a person's own plan with a vendor's coding program
// (for example "plan:claude"), instead of one of the providers they added.
const planPrefix = "plan:"

// PlanModel is the model name a plan task carries: the program chooses its own model.
const PlanModel = "default"

// plans are the programs a task can run on.
var plans = map[string]bool{"claude": true, "chatgpt": true}

// PlanProvider is the provider id for a person's plan with a program.
func PlanProvider(program string) string { return planPrefix + program }

// PlanRuntime says which program a provider id stands for; false when it is an ordinary provider.
func PlanRuntime(providerID string) (string, bool) {
	program, ok := strings.CutPrefix(providerID, planPrefix)

	return program, ok && plans[program]
}
