package application

import "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"

// researchFirst reports whether the lead must send the Researcher out before this role starts: the role writes
// the PRD, the team has a Researcher, and no Researcher has been asked yet on this task. It is enforced here
// because a rule in the lead's prompt alone was skipped on a real run.
func researchFirst(roster *Roster, role AgentRole, events []domain.Event) (string, bool) {
	if !role.WritesPRD {
		return "", false
	}

	for _, member := range roster.Members {
		if !member.WritesResearch {
			continue
		}

		for _, event := range events {
			if event.Kind == domain.EventResearch || (event.Kind == domain.EventHandoff && event.Agent == member.Name) {
				return "", false
			}
		}

		return member.Name, true
	}

	return "", false
}
