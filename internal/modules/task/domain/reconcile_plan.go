package domain

import (
	"fmt"
	"strconv"
	"strings"
)

// PlanDiff describes the stable IDs affected by a plan replacement.
type PlanDiff struct {
	Changed bool
	Added   []string
	Removed []string
	Updated []string
	Moved   []string
}

// ReconcilePlan keeps IDs and existing status when incoming steps omit them.
// Omitted existing steps are removed; an identical plan is returned unchanged.
func ReconcilePlan(current, incoming []PlanStep) ([]PlanStep, PlanDiff) {
	byID, byTitle := indexPlan(current)
	used := map[string]bool{}
	steps := make([]PlanStep, 0, len(incoming))
	next := nextPlanID(current, incoming)

	for _, step := range incoming {
		previous, found := byID[step.ID]
		if step.ID == "" {
			previous, found = unmatchedTitle(byTitle[normalizeTitle(step.Title)], used)
			if found {
				step.ID = previous.ID
			} else {
				step.ID = fmt.Sprintf("s%d", next)
				next++
			}
		} else if found && step.State == "" {
			step.State = previous.State
		}
		if found {
			used[previous.ID] = true
			if step.State == "" {
				step.State = previous.State
			}
		} else if step.State == "" {
			step.State = PlanPending
		}
		steps = append(steps, step)
	}

	diff := comparePlans(current, steps)
	return steps, diff
}

func indexPlan(plan []PlanStep) (map[string]PlanStep, map[string][]PlanStep) {
	byID, byTitle := make(map[string]PlanStep, len(plan)), map[string][]PlanStep{}
	for _, step := range plan {
		byID[step.ID] = step
		key := normalizeTitle(step.Title)
		byTitle[key] = append(byTitle[key], step)
	}
	return byID, byTitle
}

func unmatchedTitle(steps []PlanStep, used map[string]bool) (PlanStep, bool) {
	for _, step := range steps {
		if !used[step.ID] {
			return step, true
		}
	}
	return PlanStep{}, false
}

func normalizeTitle(title string) string {
	return strings.ToLower(strings.Join(strings.Fields(title), " "))
}

func nextPlanID(current, incoming []PlanStep) int {
	next := 1
	for _, plan := range [][]PlanStep{current, incoming} {
		for _, step := range plan {
			value, ok := strings.CutPrefix(step.ID, "s")
			if number, err := strconv.Atoi(value); ok && err == nil && number >= next {
				next = number + 1
			}
		}
	}
	return next
}

func comparePlans(current, updated []PlanStep) PlanDiff {
	diff := PlanDiff{Changed: len(current) != len(updated)}
	positions := make(map[string]int, len(current))
	for i, step := range current {
		positions[step.ID] = i
	}
	seen := make(map[string]bool, len(updated))
	for i, step := range updated {
		old, exists := positions[step.ID]
		if !exists {
			diff.Added = append(diff.Added, step.ID)
			diff.Changed = true
		} else {
			if current[old] != step {
				diff.Updated = append(diff.Updated, step.ID)
				diff.Changed = true
			}
			if old != i {
				diff.Moved = append(diff.Moved, step.ID)
				diff.Changed = true
			}
		}
		seen[step.ID] = true
	}
	for _, step := range current {
		if !seen[step.ID] {
			diff.Removed = append(diff.Removed, step.ID)
			diff.Changed = true
		}
	}
	return diff
}
