package application

import "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"

// researchFirst says what the lead must do before this role may start, or "" when the way is clear. It is
// enforced here because rules in the lead's prompt alone were skipped on a real run. Two rounds are required
// when the team has a Researcher:
//   - before the Planner writes the PRD: a first round on the request itself;
//   - after the PRD, before anyone else works: one round on the PRD's stack, so each teammate builds from
//     design references, official documentation and quality checks instead of memory.
//
// Small work has no Planner and so no PRD, and needs neither round.
func researchFirst(roster *Roster, role AgentRole, events []domain.Event) string {
	scout, found := researcherOf(roster)
	if !found || role.WritesResearch {
		return ""
	}

	if role.WritesPRD {
		if askedAfter(events, scout, -1) {
			return ""
		}

		return "Before the " + role.Name + " starts, hand the " + scout +
			" a job: it looks things up on the web and its notes go into the " + role.Name + "'s brief."
	}

	prd, written := firstPRD(events)
	if !written || askedAfter(events, scout, prd) {
		return ""
	}

	return "The PRD is ready. Before the " + role.Name + " starts, hand the " + scout + " one round on the PRD's " +
		"stack and kind of work: design references, official docs and current versions, pictures, and quality " +
		"checks, as they apply. Then put the section that is theirs into the " + role.Name + "'s brief."
}

// askedAfter reports whether the Researcher has been given a job, or has delivered notes, after an event.
func askedAfter(events []domain.Event, scout string, after int64) bool {
	for _, event := range events {
		isRound := event.Kind == domain.EventResearch || (event.Kind == domain.EventHandoff && event.Agent == scout)
		if event.Seq > after && isRound {
			return true
		}
	}

	return false
}

func firstPRD(events []domain.Event) (int64, bool) {
	for _, event := range events {
		if event.Kind == domain.EventPRD {
			return event.Seq, true
		}
	}

	return 0, false
}

func researcherOf(roster *Roster) (string, bool) {
	for _, member := range roster.Members {
		if member.WritesResearch {
			return member.Name, true
		}
	}

	return "", false
}
