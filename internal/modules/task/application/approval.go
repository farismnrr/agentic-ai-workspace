package application

import (
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// changingTools are the tools that make or run something. A role with one of them builds; the others only look.
var changingTools = []string{
	"edit_file", "write_file", "delete_file", "move_file", "run_task", "run_command", "start_site", "download_asset",
}

// startBuilding is the answer that lets a plan-mode task go on once its PRD is written.
const startBuilding = "Start building"

func buildsThings(role AgentRole) bool {
	for _, tool := range role.Tools {
		for _, changing := range changingTools {
			if tool == changing {
				return true
			}
		}
	}

	return false
}

// awaitingApproval returns what the lead must do first, or "" when the role may start. In plan mode a role that
// builds waits until the person has said yes to the PRD: the server checks it because a prompt alone would let a
// model start building. Work with no PRD (a small change handed straight to the Developer) is not held.
func awaitingApproval(task domain.Task, lead string, role AgentRole, events []domain.Event) string {
	if !task.PlanFirst || !buildsThings(role) {
		return ""
	}

	var prd int64

	for _, event := range events {
		if event.Kind == domain.EventPRD {
			prd = event.Seq
		}
	}

	if prd == 0 || approvedAfter(events, lead, prd) {
		return ""
	}

	return "The PRD is written, and in plan mode nobody builds until the person says yes. Ask them once with ask_user, " +
		"offering \"" + startBuilding + " (Recommended)\" and \"Change something\". If they want a change, send it to " +
		"the Planner; when they say \"" + startBuilding + "\", hand the " + role.Name + " the job again."
}

func approvedAfter(events []domain.Event, lead string, prd int64) bool {
	for _, event := range events {
		if event.Seq > prd && event.Kind == domain.EventQuestion && event.Agent == lead &&
			strings.HasPrefix(strings.ToLower(strings.TrimSpace(event.Answered)), strings.ToLower(startBuilding)) {
			return true
		}
	}

	return false
}
