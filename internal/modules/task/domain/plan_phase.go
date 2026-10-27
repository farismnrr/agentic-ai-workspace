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

// CanMoveTo reports whether this slice's lifecycle operations allow a transition.
func (p PlanPhase) CanMoveTo(next PlanPhase) bool {
	switch p {
	case PlanDraft:
		return next == PlanReadyForApproval || next == PlanAwaitingClarification
	case PlanReadyForApproval:
		return next == PlanDraft || next == PlanAwaitingClarification
	case PlanAwaitingClarification:
		return next == PlanDraft || next == PlanReadyForApproval
	case PlanApproved:
		return next == PlanHandedOff
	case PlanHandedOff:
		return false
	default:
		return false
	}
}
