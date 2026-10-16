package application

import "time"

const (
	defaultWallTime  = 15 * time.Minute
	defaultMaxSteps  = 40
	defaultRetries   = 2
	defaultMaxResult = 24_000

	// A team task has several agents working inside one run, so it gets more room.
	defaultTeamWallTime  = 30 * time.Minute
	defaultTeamSteps     = 100
	defaultDelegateSteps = 25
)

// Limits keep one task from running away with time or steps.
type Limits struct {
	Backoff  func(attempt int) time.Duration
	WallTime time.Duration
	// TeamWallTime and TeamSteps replace WallTime and MaxSteps for a team task; the steps are shared by the
	// orchestrator and every teammate. DelegateSteps caps one teammate's turns per delegation.
	TeamWallTime  time.Duration
	MaxSteps      int
	TeamSteps     int
	DelegateSteps int
	Retries       int
	MaxResult     int
}

// DefaultLimits are the production limits.
func DefaultLimits() Limits {
	return Limits{
		Backoff:  func(attempt int) time.Duration { return time.Duration(attempt+1) * 3 * time.Second },
		WallTime: defaultWallTime, MaxSteps: defaultMaxSteps,
		TeamWallTime: defaultTeamWallTime, TeamSteps: defaultTeamSteps, DelegateSteps: defaultDelegateSteps,
		Retries: defaultRetries, MaxResult: defaultMaxResult,
	}
}

func (l Limits) wallTime(team bool) time.Duration {
	if team && l.TeamWallTime > 0 {
		return l.TeamWallTime
	}

	return l.WallTime
}

func (l Limits) steps(team bool) int {
	if team && l.TeamSteps > 0 {
		return l.TeamSteps
	}

	return l.MaxSteps
}

// budget counts the model turns a run has left. The orchestrator and the teammates it delegates to
// draw from the same one; a run is a single goroutine, so it needs no lock.
type budget struct {
	left int
}

func (l Limits) delegateSteps() int {
	if l.DelegateSteps > 0 {
		return l.DelegateSteps
	}

	return l.MaxSteps
}
