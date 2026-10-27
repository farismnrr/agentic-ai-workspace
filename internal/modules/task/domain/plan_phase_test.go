package domain

import "testing"

func TestPlanPhaseTransitions(t *testing.T) {
	valid := [][2]PlanPhase{
		{PlanDraft, PlanReadyForApproval},
		{PlanDraft, PlanAwaitingClarification},
		{PlanReadyForApproval, PlanDraft},
		{PlanReadyForApproval, PlanAwaitingClarification},
		{PlanReadyForApproval, PlanApproved},
		{PlanAwaitingClarification, PlanDraft},
		{PlanAwaitingClarification, PlanReadyForApproval},
		{PlanApproved, PlanHandedOff},
	}
	for _, transition := range valid {
		if !transition[0].CanMoveTo(transition[1]) {
			t.Errorf("%s cannot move to %s", transition[0], transition[1])
		}
	}
	invalid := [][2]PlanPhase{
		{PlanDraft, PlanApproved},
		{PlanReadyForApproval, PlanHandedOff},
		{PlanHandedOff, PlanDraft},
		{PlanAwaitingClarification, PlanApproved},
	}
	for _, transition := range invalid {
		if transition[0].CanMoveTo(transition[1]) {
			t.Errorf("%s unexpectedly moved to %s", transition[0], transition[1])
		}
	}
}
