package domain

// PlanPhase is the lifecycle of an external assistant's plan.
type PlanPhase string

const (
	PlanDraft                 PlanPhase = "draft"
	PlanAwaitingClarification PlanPhase = "awaiting_clarification"
	PlanReadyForApproval      PlanPhase = "ready_for_approval"
	PlanApproved              PlanPhase = "approved"
	PlanHandedOff             PlanPhase = "handed_off"
)

var planPhaseTransitions = map[PlanPhase]map[PlanPhase]struct{}{
	PlanDraft: {
		PlanReadyForApproval:      {},
		PlanAwaitingClarification: {},
	},
	PlanReadyForApproval: {
		PlanDraft:                 {},
		PlanAwaitingClarification: {},
		PlanApproved:              {},
	},
	PlanAwaitingClarification: {
		PlanDraft:            {},
		PlanReadyForApproval: {},
	},
	PlanApproved: {
		PlanHandedOff: {},
	},
}

// CanMoveTo reports whether this slice's lifecycle operations allow a transition.
func (p PlanPhase) CanMoveTo(next PlanPhase) bool {
	_, allowed := planPhaseTransitions[p][next]
	return allowed
}
