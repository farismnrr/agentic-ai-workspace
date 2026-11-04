package application

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// workOnPlan lets a teammate do their job with the person's own plan: one turn of the vendor's program with the
// role's brief, held to the tools the role may use. What it reports back is the teammate's report.
func (r *Runner) workOnPlan(
	ctx context.Context, worker *Runner, teammate domain.Task, role AgentRole, d Delegation,
) outcome {
	program, _ := domain.PlanRuntime(d.ProviderID)
	name := programNames[program]

	if r.deps.CLI == nil {
		return failed("This server cannot use your own plan yet.", "")
	}

	if signedIn, err := r.deps.CLI.Ready(ctx, teammate.UserID, program); err != nil || !signedIn {
		return failed("The "+d.RoleName+" is set to use your "+name+" plan, which is not signed in.", "")
	}

	allowed, err := worker.allowedTools(ctx, teammate.UserID, role)
	if err != nil {
		return failed("I could not check what the "+d.RoleName+" may use.", "")
	}

	// The model prompt, CLI built-ins and team MCP bridge MUST see the
	// same owner-switch-filtered tool inventory. Never advertise a disabled
	// server tool or imply that a missing tool has a shell fallback.
	role.Tools = allowed
	sink := &planSink{runner: worker, task: teammate, quiet: true}
	site := worker.siteBridge(teammate, role, d.ID)

	result, err := r.deps.CLI.Run(ctx, CLIRequest{
		UserID: teammate.UserID, ProjectID: teammate.ProjectID, TaskID: teammate.ID, Program: program,
		Model: d.Model, Effort: string(d.Effort), Tools: allowed, Limited: true,
		RoleKey: role.Key, Bridge: site.bridge(),
		Prompt: worker.rolePrompt(ctx, teammate, role, d),
	}, func(event CLIEvent) { sink.emit(ctx, event) })

	if stopped, over := r.limitReached(ctx); over {
		return stopped
	}

	if waiting, ok := site.stoppedOutcome(); ok {
		return waiting
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
func (r *Runner) allowedTools(ctx context.Context, userID string, role AgentRole) ([]string, error) {
	enabled, err := r.deps.Gate.Enabled(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("read tool switches: %w", err)
	}

	// Never pass globally enabled tools the teammate's server-owned role
	// does not have. In particular, Developer must not inherit QA Bash/test
	// or DevOps workflow execution just because the person enabled them.
	names := make([]string, 0, len(role.Tools))
	for _, name := range role.Tools {
		if enabled[name] && !slices.Contains(names, name) {
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
	out.WriteString(preActionToolSOP(role.Key, role.Tools))

	if role.Prompt != "" {
		out.WriteString("\n" + role.Prompt + "\n")
	}

	appendSkills(&out, role)

	if slices.Contains(role.Tools, toolStartSite) {
		out.WriteString("\nYou start and stop the web app with the start_site and stop_site tools of the team server, " +
			"never with a shell command.\n")
	}
	out.WriteString(bridgeVCSInstructions(role.Tools))
	out.WriteString(reviewModePrompt(task.ReviewMode))

	out.WriteString("\n" + strings.TrimSpace(laneRule) + "\n")
	out.WriteString("\n" + stuckRule + "\n")
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

// BridgeVCSInstructions prevents restricted vendor CLI workers from attempting GitHub
// operations through their own shell. That container deliberately has neither
// GitHub credentials nor GitHub egress; the server-mediated git/gh tools alone
// enforce the project's owner credentials and the teammate's role permissions.
func BridgeVCSInstructions(tools []string) string {
	gitAllowed := slices.Contains(tools, "git")
	ghAllowed := slices.Contains(tools, "gh")
	if !gitAllowed && !ghAllowed {
		return ""
	}

	var out strings.Builder
	out.WriteString("\nGitHub access for this restricted teammate is ONLY through the team's server tools. " +
		"Your shell/native Git subprocess has no GitHub token or github.com network access by design. " +
		"Do NOT use Bash, run_command, shell git/gh, or bypass HTTPS_PROXY for GitHub operations. ")
	if gitAllowed {
		out.WriteString("Use the server tool named git with an args array for status, diff, fetch, commit and push. " +
			"For example, a normal dev push is git with args [\"push\", \"origin\", \"dev\"]. ")
	}
	if ghAllowed {
		out.WriteString("Use the server tool named gh with an args array for authorized GitHub CLI operations. ")
	}
	out.WriteString("If the server tool is disabled or missing, report that blocker instead of falling back to shell. " +
		"If it says CONNECT_GITHUB_REQUIRED, tell the orchestrator the project owner must connect GitHub " +
		"in Masih Awam Settings. An HTTP proxy 403 from a shell is a restricted-container policy denial; " +
		"retry with the appropriate server tool, not with curl, an alternate proxy, or direct DNS. " +
		"Keep the repository's dirty-worktree and non-force-push safety rules.\n")
	return out.String()
}

func bridgeVCSInstructions(tools []string) string {
	return BridgeVCSInstructions(tools)
}
