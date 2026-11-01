package application

import (
	"context"

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
				"Check QA availability and the recorded review before trying again.")
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


