package application

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// workOnPlan lets a teammate do their job with the person's own plan: one turn of the vendor's program with the
// role's brief, held to the tools the role may use. What it reports back is the teammate's report.
func (r *Runner) workOnPlan(ctx context.Context, teammate domain.Task, role AgentRole, d Delegation) outcome {
	program, _ := domain.PlanRuntime(d.ProviderID)
	name := programNames[program]

	if r.deps.CLI == nil {
		return failed("This server cannot use your own plan yet.", "")
	}

	if signedIn, err := r.deps.CLI.Ready(ctx, teammate.UserID, program); err != nil || !signedIn {
		return failed("The "+d.RoleName+" is set to use your "+name+" plan, which is not signed in.", "")
	}

	worker := r.child(role, d)

	allowed, err := worker.allowedTools(ctx, teammate.UserID)
	if err != nil {
		return failed("I could not check what the "+d.RoleName+" may use.", "")
	}

	sink := &planSink{runner: worker, task: teammate, quiet: true}

	result, err := r.deps.CLI.Run(ctx, CLIRequest{
		UserID: teammate.UserID, ProjectID: teammate.ProjectID, Program: program,
		Model: d.Model, Effort: string(d.Effort), Tools: allowed, Limited: true,
		Prompt: worker.rolePrompt(ctx, teammate, role, d),
	}, func(event CLIEvent) { sink.emit(ctx, event) })

	if stopped, over := r.limitReached(ctx); over {
		return stopped
	}

	if err != nil {
		return failed(d.RoleName+" could not run "+name+" for this job.", "")
	}

	closing := sink.pending
	if closing == "" {
		closing = sink.last
	}

	return reportOf(role, result, closing)
}

// reportOf is how a teammate's turn ended: the closing words are the report, and the document of a role that
// writes one.
func reportOf(role AgentRole, result CLIResult, closing string) outcome {
	if result.Reason != "" {
		return failed(result.Reason, "")
	}

	text := strings.TrimSpace(closing)
	if text == "" {
		text = strings.TrimSpace(result.Summary)
	}

	done := outcome{stop: stopFinished, text: text}
	if role.WritesPRD || role.WritesResearch {
		done.document = text
	}

	return done
}

// allowedTools are the project tools this role may use, after the person's own switches.
func (r *Runner) allowedTools(ctx context.Context, userID string) ([]string, error) {
	enabled, err := r.deps.Gate.Enabled(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("read tool switches: %w", err)
	}

	names := make([]string, 0, len(enabled))

	for name, on := range enabled {
		if on {
			names = append(names, name)
		}
	}

	sort.Strings(names)

	return names, nil
}

// rolePrompt is what the program is told: who the teammate is, the job, and how to report.
func (r *Runner) rolePrompt(ctx context.Context, task domain.Task, role AgentRole, d Delegation) string {
	project, err := r.deps.Projects.Name(ctx, task.UserID, task.ProjectID)
	if err != nil {
		project = "your project"
	}

	var out strings.Builder

	fmt.Fprintf(&out, "You are the %s on a small software team working for a person who is not a developer. ", role.Name)
	fmt.Fprintf(&out, "The orchestrator gave you one job inside the project %q: ", project)
	out.WriteString("its files are in your current folder.\n")

	if role.Purpose != "" {
		out.WriteString("Your part on the team: " + role.Purpose + "\n")
	}

	if role.Prompt != "" {
		out.WriteString("\n" + role.Prompt + "\n")
	}

	appendSkills(&out, role)
	out.WriteString("\nYou cannot talk to the person: when something is unclear, take the safest reasonable assumption ")
	out.WriteString("and say so. When you are done, reply with a short plain summary: what you found or did, what you ")
	out.WriteString("ran and how it ended, and anything left open. Never claim a check you did not run.\n")
	out.WriteString("\nYour job:\n" + r.brief(ctx, task, d))

	return out.String()
}

// brief is the job as the orchestrator gave it, with the team's documents.
func (r *Runner) brief(ctx context.Context, task domain.Task, d Delegation) string {
	messages, err := r.deps.Transcript.LoadTranscript(ctx, task.ID)
	if err == nil && len(messages) > 0 && len(messages[0].Blocks) > 0 {
		return messages[0].Blocks[0].Text
	}

	return d.Brief
}
