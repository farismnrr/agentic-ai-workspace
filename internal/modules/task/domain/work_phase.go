package domain

import "time"

// WorkPhase describes the current, derived state shown for an external work session.
type WorkPhase string

const (
	WorkPhaseCompleted      WorkPhase = "completed"
	WorkPhaseFailed         WorkPhase = "failed"
	WorkPhaseStopped        WorkPhase = "stopped"
	WorkPhaseAbandoned      WorkPhase = "abandoned"
	WorkPhaseAwaitingPerson WorkPhase = "awaiting_person"
	WorkPhaseBlocked        WorkPhase = "blocked"
	WorkPhaseReported       WorkPhase = "reported"
	WorkPhaseIdle           WorkPhase = "idle"
	WorkPhaseRunning        WorkPhase = "running"
)

// WorkPhaseThresholds holds the durations that affect a derived work phase.
type WorkPhaseThresholds struct {
	IdleAfter    time.Duration
	AbandonAfter time.Duration
}

// DeriveWorkPhase derives presentation state without persisting a phase.
func DeriveWorkPhase(work ExternalWork, now time.Time, thresholds WorkPhaseThresholds) WorkPhase {
	switch work.State {
	case WorkCompleted:
		return WorkPhaseCompleted
	case WorkFailed:
		return WorkPhaseFailed
	case WorkStopped:
		return WorkPhaseStopped
	case WorkAbandoned:
		return WorkPhaseAbandoned
	}
	if waitingForPerson(work.PlanState) {
		return WorkPhaseAwaitingPerson
	}
	if work.State == WorkBlocked {
		return WorkPhaseBlocked
	}
	if work.ReportedAt != nil {
		return WorkPhaseReported
	}
	if work.State == WorkOpen && now.Sub(work.UpdatedAt) >= thresholds.IdleAfter {
		return WorkPhaseIdle
	}
	return WorkPhaseRunning
}

func waitingForPerson(phase PlanPhase) bool {
	return phase == PlanReadyForApproval || phase == PlanAwaitingClarification
}
