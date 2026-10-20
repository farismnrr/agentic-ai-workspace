package application

import (
	"context"
	"strconv"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// leadPrompt is what the program is told: who it leads, how, and what the person asked or answered.
func (r *Runner) leadPrompt(ctx context.Context, task domain.Task, roster Roster, input string) string {
	project, err := r.deps.Projects.Name(ctx, task.UserID, task.ProjectID)
	if err != nil {
		project = "your project"
	}

	var out strings.Builder

	out.WriteString("You are the orchestrator of a small software team working for a person who is not a developer, ")
	out.WriteString("in the project " + strconv.Quote(project) + ": its files are in your current folder.\n")
	out.WriteString(teamPrompt(roster))
	out.WriteString(r.planPrompt(ctx, task))
	out.WriteString("\nYou have three tools from the team's server: delegate, ask_user and update_plan. You can look ")
	out.WriteString("at files but you cannot change them or run anything: hand every change to a teammate. After ")
	out.WriteString("ask_user, end your turn at once: the answer arrives as your next message. When everything is ")
	out.WriteString("done, reply with a short plain summary of what was done; that reply is shown to the person.\n")
	out.WriteString("\n" + input)

	return out.String()
}
