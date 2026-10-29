package application

import (
	"context"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

var programNames = map[string]string{"claude": "Claude", "chatgpt": "ChatGPT"}

const maxCLIReviewContinuations = 9

// runPlan does a task on the person's own plan: the vendor's coding program does the work in the project's
// folder and this records what it reports, so the thread, the files and the receipt look like any other task.
func (r *Runner) runPlan(ctx context.Context, task domain.Task) outcome {
	program, _ := domain.PlanRuntime(task.ProviderID)
	name := programNames[program]

	if r.deps.CLI == nil || r.deps.Sessions == nil {
		return failed("This server cannot use your own plan yet.", "")
	}

	signedIn, err := r.deps.CLI.Ready(ctx, task.UserID, program)
	if err != nil {
		return failed("I could not check your "+name+" sign-in.", "Press Continue in a moment.")
	}

	if !signedIn {
		return failed("You are not signed in to "+name+".", "Sign in under Settings, Models, then press Continue.")
	}

	if err = r.deps.Tasks.SetStatus(ctx, task.ID, domain.StatusWorking, r.deps.Now()); err != nil {
		return failed("I could not start this task.", "")
	}

	return r.turnOnPlan(ctx, task, program)
}

func (r *Runner) turnOnPlan(ctx context.Context, task domain.Task, program string) outcome {
	session, _ := r.deps.Sessions.Session(ctx, task.ID)
	runner, lead, leadName := r, (*leadSession)(nil), ""
	if task.Team {
		if lead = r.leadOf(ctx, task); lead != nil {
			runner, leadName = lead.lead, lead.state.roster.Lead.Name
		}
	}

	req := CLIRequest{
		UserID: task.UserID, ProjectID: task.ProjectID, TaskID: task.ID, Program: program, Session: session,
		Model: task.Model, Effort: string(task.Effort), Prompt: r.lastInput(ctx, task.ID, leadName),
	}
	if session == "" {
		req.Prompt = r.continuation(ctx, task.ID) + "\n\nCurrent request:\n" + req.Prompt
	}
	if lead != nil {
		req = lead.shape(ctx, req)
	} else {
		req.Prompt = reviewModePrompt(task.ReviewMode) + r.notesPrompt() + req.Prompt
	}

	// A completed CLI turn may still require an independent role's report.
	// Resume the SAME task/session with the server's review feedback instead
	// of turning each soft hold into another user-facing failure card.
	for attempt := range maxCLIReviewContinuations + 1 {
		sink := &planSink{runner: runner, task: task}
		result, settled, fresh, runErr := r.runCLIWithRecovery(ctx, task, req, sink)
		if stopped, over := r.limitReached(ctx); over {
			return stopped
		}
		if runErr != nil {
			return failed("I could not run "+programNames[program]+" for this task.", "Press Continue to try again.")
		}
		if result.Reason == "" {
			if result.Session != "" {
				_ = r.deps.Sessions.SaveSession(ctx, task.ID, result.Session)
				req.Session = result.Session
			} else if fresh {
				_ = r.deps.Sessions.SaveSession(ctx, task.ID, "")
				req.Session = ""
			}
		}
		if lead != nil && lead.stopped != nil {
			settled.flush(ctx)
			return *lead.stopped
		}
		completed, feedback := settled.finish(ctx, result, lead)
		if feedback == "" {
			return completed
		}
		if attempt == maxCLIReviewContinuations {
			return failed("The required review is still missing after automatic continuation.",
				"Check the Reviewer's availability and recorded report before trying again.")
		}
		if req.Session == "" {
			return failed("Independent review is required, but the CLI did not provide a resumable session.",
				"Press Continue after the vendor CLI can resume its session.")
		}
		req.Prompt = "The server has NOT approved completion of this task. " + feedback +
			"\nContinue this SAME task and delegate the named verification role. " +
			"If the plan says all steps are done, add the required review step with update_plan first. " +
			"Do not repeat completed edits. Only report completion after the independent review is recorded " +
			"and its findings are resolved."
	}
	return failed("The required review could not be completed.", "Review the task's evidence before continuing.")
}

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
