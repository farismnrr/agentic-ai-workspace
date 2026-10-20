package application

import (
	"context"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// lastInput is the latest thing the person gave the program: what they asked for, or what they answered.
func (r *Runner) lastInput(ctx context.Context, taskID, leadName string) string {
	events, err := r.deps.Events.EventsAfter(ctx, taskID, 0)
	if err != nil {
		return ""
	}

	for i := len(events) - 1; i >= 0; i-- {
		switch event := events[i]; {
		case event.Kind == domain.EventRequest:
			if len(event.Documents) > 0 && event.Technical != "" {
				return event.Technical
			}
			return event.Text
		case event.Kind == domain.EventQuestion && event.Answered != "":
			return answeredText(event, leadName)
		case event.Kind == domain.EventApproval && event.Resolved != "":
			return "The person " + event.Resolved + " a step of the " + event.Agent + ": " + event.Text +
				"\nCall delegate again with the same role and brief to continue."
		}
	}

	return ""
}

// answeredText is an answer in words, for the lead to act on: a teammate's question is theirs to be asked again.
func answeredText(event domain.Event, leadName string) string {
	if event.Agent == "" || event.Agent == leadName {
		return "The person answered: " + event.Answered
	}

	return "The person answered the " + event.Agent + ": " + event.Answered +
		"\nCall delegate again with the same role and brief to continue."
}
