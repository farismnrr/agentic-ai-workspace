package application

import (
	"context"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// planSink turns what the program reports into the task's thread.
type planSink struct {
	didStep    bool
	meterKnown bool
	runner     *Runner
	pending    string
	last       string
	task       domain.Task
	// quiet keeps the program's typing out of the task's own reply: a teammate's words are shown as steps.
	quiet bool
}

func (p *planSink) emit(ctx context.Context, event CLIEvent) {
	switch event.Kind {
	case CLIActivity:
		p.didStep = true
	case CLIUsage:
		if p.runner.role == nil && event.Used > 0 && event.Window > 0 {
			p.runner.saveMeter(ctx, p.task, CLIResult{Used: event.Used, Window: event.Window})
			p.meterKnown = true
		}
	case CLIDelta:
		if !p.quiet {
			p.runner.deps.Notify.Delta(p.task.ID, event.Text)
		}
	case CLIText:
		p.flush(ctx)
		p.pending = strings.TrimSpace(event.Text)
	case CLIChange:
		p.flush(ctx)
		p.step(ctx, domain.Event{Text: event.Text, Technical: event.Technical, Files: event.Files})
		p.runner.deps.Notify.FilesChanged(p.task.ProjectID)
	case CLICommand:
		p.flush(ctx)
		p.step(ctx, domain.Event{
			Text: event.Text, Technical: event.Technical,
			Run: &domain.RunInfo{
				Name: checkName(event.Technical), Command: event.Technical,
				// Read-only checks do not have file effects. Unknown commands
				// remain conservative, even when no paths were reported.
				EffectPaths: event.Files, EffectsKnown: len(event.Files) > 0 || knownNonMutatingCommand(event.Technical),
				ExitCode: event.ExitCode, TimedOut: event.TimedOut,
			},
		})
	}
}

func (p *planSink) flush(ctx context.Context) {
	if p.pending != "" {
		p.last = p.pending
	}

	p.runner.say(ctx, p.task.ID, p.pending)
	p.pending = ""
}

func (p *planSink) step(ctx context.Context, event domain.Event) {
	p.didStep = true
	event.Kind, event.Time = domain.EventAction, p.runner.deps.Now()
	_, _ = p.runner.deps.Events.AppendEvent(ctx, p.task.ID, event)
}

// finish ends the task like the agent's own loop does: work gets a receipt, a plain answer does not.
// finish distinguishes a terminal result from a soft completion hold. A soft
// hold returns the server's exact review instruction for the SAME CLI session.
func (p *planSink) finish(ctx context.Context, result CLIResult, lead *leadSession) (outcome, string) {
	if result.Reason != "" {
		p.flush(ctx)
		if isCLIContextOverflow(result.Reason) {
			return failed("AI context is full and automatic session recovery did not succeed.",
				"Press Continue to retry or shorten the input."), ""
		}
		return failed(result.Reason, "Press Continue to try again."), ""
	}

	summary := p.pending
	if summary == "" {
		summary = strings.TrimSpace(result.Summary)
	}

	events, err := p.runner.deps.Events.EventsAfter(ctx, p.task.ID, 0)
	if err != nil {
		return failed("I lost track of this task's history.", "Press Continue in a moment."), ""
	}

	if workDone(events) {
		state := &runState{roster: p.runner.roster(ctx, p.task)}
		if lead != nil {
			state = lead.state
		}
		gateID := ""
		if lead != nil {
			gateID = "cli-completion-gate"
		}
		// The CLI bridge owns a captured-result channel, not the model's
		// normal transcript. Review feedback must be captured there so it can
		// be sent as the next turn's prompt instead of silently disappearing.
		gateRunner := p.runner
		if lead != nil {
			gateRunner = lead.inline
		}
		blocked, allowed := gateRunner.completionGate(ctx, p.task, state, Call{ID: gateID})
		if !allowed {
			if blocked.stop != 0 {
				return blocked, ""
			}
			if lead == nil {
				return failed("Independent verification is required but no team lead can delegate it.",
					"Enable the applicable reviewer role and continue with a team."), ""
			}
			feedback := strings.TrimSpace(lead.results.take(gateID).text)
			if feedback == "" {
				return failed("The required review could not be scheduled.",
					"Check the team's review roles and restart verification."), ""
			}
			return outcome{}, feedback
		}
		return p.runner.complete(ctx, p.task, summary, buildProof(events, nil)), ""
	}
	return p.runner.answer(ctx, p.task, summary), ""
}

const maxCheckName = 60

// checkName is how a program's command is listed among the checks that ran: its first line, kept short.
func checkName(command string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(command), "\n")
	runes := []rune(line)

	if len(runes) == 0 {
		return "command"
	}

	if len(runes) > maxCheckName {
		return string(runes[:maxCheckName]) + "…"
	}

	return line
}
