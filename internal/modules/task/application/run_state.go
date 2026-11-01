package application

import (
	"sync"
)

// runState is what one run remembers between model turns.
type runState struct {
	// budget is the model turns left in the whole run; a teammate's loop shares the run's budget.
	budget *budget
	// roster is the team the orchestrator leads on this run; nil for a solo task and for teammates.
	roster *Roster
	holds
	// size is the tokens the last model call carried; zero while unknown.
	size int64
	// window is the model's context window in tokens, as of the last turn.
	window int64
	// failed counts compactions that produced no summary, so a stuck one is not retried forever.
	failed int
	// segment is the model turns this loop may still take; a teammate's has a share of its own.
	segment int
	// misses is how many of a teammate's steps in a row failed.
	misses int
	// noImages is set once the model refused a request with screenshots; it then gets their text reports only.
	noImages bool
	mu       sync.Mutex
}
