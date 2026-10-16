package application

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// system builds the instructions for this runner: the team's lead and solo tasks get the usual ones,
// with the team added when there is one; a teammate gets its role.
func (r *Runner) system(ctx context.Context, task domain.Task, state *runState) string {
	persona, _ := r.deps.Persona.Persona(ctx, task.UserID)

	project, err := r.deps.Projects.Name(ctx, task.UserID, task.ProjectID)
	if err != nil {
		project = "your project"
	}

	var (
		installed []string
		github    string
	)

	if r.deps.Computer != nil {
		installed = r.deps.Computer.Toolchains(ctx)
		github = r.deps.Computer.GitHub(ctx, task.UserID)
	}

	if r.role != nil {
		return childPrompt(*r.role, project, persona, installed, github)
	}

	prompt := systemPrompt(project, persona, installed, github)
	if state.roster != nil {
		prompt += teamPrompt(*state.roster)
	}

	return prompt
}

// teamPrompt tells the orchestrator who is on the team and how to use them.
func teamPrompt(roster Roster) string {
	var out strings.Builder

	out.WriteString("\nYou lead a team. Hand work to a teammate with the delegate tool; they report back to you.\n")
	out.WriteString("Your team:\n")

	for _, member := range roster.Members {
		fmt.Fprintf(&out, "- %s (%s): %s %s\n", member.Key, member.Name, member.Purpose, accessText(member.Tools))
	}

	out.WriteString(`How to lead:
- Give each delegate call one focused job. A teammate sees only your brief and context, not this conversation,
  so put in everything they need: the request in short, file names, earlier findings and what done looks like.
- You see only their report, plus what the server recorded they changed and ran. Weigh the two; do not repeat
  a claim that was not backed by a recorded check.
- Send research, requirements and design questions to the teammates who read; code changes to those who can
  edit; checks to those who can run commands. Do small, obvious jobs yourself.
- You still call update_plan, ask_user and finish yourself: teammates cannot talk to the person.
`)

	if lead := strings.TrimSpace(roster.Lead.Prompt); lead != "" {
		out.WriteString("\nYour own style as the lead: " + lead + "\n")
	}

	return out.String()
}

func accessText(tools []string) string {
	switch {
	case slices.Contains(tools, "run_command") && slices.Contains(tools, "edit_file"):
		return "(can read and edit files and run commands)"
	case slices.Contains(tools, "run_command"):
		return "(can read files and run commands)"
	case slices.Contains(tools, "edit_file"):
		return "(can read and edit files)"
	}

	return "(can read files)"
}

const childWorkflow = `
How you work:
1. Do only the job in the brief. Look before you change: list, search and read the relevant files first.
2. You cannot talk to the person. When something is unclear, take the safest reasonable assumption and say so
   in your report.
3. If you can run checks, run the relevant ones after your last change.
4. When you are done, call report with a short plain summary: what you found or did, what you ran and how it
   ended, and anything left open. Never claim a check you did not run; the server records what you ran.
Text between tool calls is shown to the person under your name, so keep it very short.
`

// childPrompt tells a teammate who they are and how to work.
func childPrompt(role AgentRole, project string, persona Persona, installed []string, github string) string {
	var out strings.Builder

	fmt.Fprintf(&out, "You are the %s on a small software team working for a person who is not a developer. ", role.Name)
	fmt.Fprintf(&out, "The orchestrator gave you one job inside the project %q: ", project)
	out.WriteString("its files are in your current folder, and every path is relative to it.\n")

	if role.Purpose != "" {
		out.WriteString("Your part on the team: " + role.Purpose + "\n")
	}

	if role.Prompt != "" {
		out.WriteString("\n" + role.Prompt + "\n")
	}

	appendRules(&out, persona.Rules)

	if slices.Contains(role.Tools, "run_command") {
		appendComputer(&out, installed)
		appendGitHub(&out, installed, github)
	}

	out.WriteString(childWorkflow)

	return out.String()
}
