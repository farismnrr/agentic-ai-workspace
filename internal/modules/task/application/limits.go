package application

import "time"

const (
	defaultWallTime  = 15 * time.Minute
	defaultMaxSteps  = 40
	defaultRetries   = 2
	defaultMaxResult = 24_000

	// A team task has several agents working inside one run and can build something big, so it gets a lot of
	// room. Running out is a pause, not a loss: Continue picks up where the work stopped, with a fresh budget.
	defaultTeamWallTime  = 2 * time.Hour
	defaultTeamSteps     = 800
	defaultDelegateSteps = 60
	// A team does not stop for how many jobs it hands out, only when jobs stop getting anywhere: this many in a
	// row that changed nothing, ran no check and wrote no document means it is stuck.
	defaultMaxIdleJobs = 6
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
	// MaxIdleJobs is how many jobs in a row may hand nothing back, before the orchestrator is told it is stuck.
	MaxIdleJobs int
	// RequirePlan holds the first job of a request until the plan shows it, so the todo list is always current.
	RequirePlan bool
	// AskFirst holds the first job of a request until the orchestrator has asked the person or chosen not to.
	AskFirst  bool
	Retries   int
	MaxResult int
}

// DefaultLimits are the production limits.
func DefaultLimits() Limits {
	return Limits{
		Backoff:  func(attempt int) time.Duration { return time.Duration(attempt+1) * 3 * time.Second },
		WallTime: defaultWallTime, MaxSteps: defaultMaxSteps,
		TeamWallTime: defaultTeamWallTime, TeamSteps: defaultTeamSteps, DelegateSteps: defaultDelegateSteps,
		MaxIdleJobs: defaultMaxIdleJobs, RequirePlan: true, AskFirst: true,
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

func (l Limits) idleJobs() int {
	if l.MaxIdleJobs > 0 {
		return l.MaxIdleJobs
	}

	return defaultMaxIdleJobs
}
