package application

import "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"

// pendingCalls returns the tool calls of the last assistant message that have no result yet.
func pendingCalls(transcript []domain.Message) []Call {
	last := lastAssistant(transcript)
	if last < 0 {
		return nil
	}

	answered := map[string]bool{}

	for _, message := range transcript[last+1:] {
		for _, block := range message.Blocks {
			if block.Kind == domain.BlockToolResult {
				answered[block.ID] = true
			}
		}
	}

	var calls []Call

	for _, block := range transcript[last].Blocks {
		if block.Kind == domain.BlockToolUse && !answered[block.ID] {
			calls = append(calls, Call{ID: block.ID, Name: block.Name, Input: block.Input})
		}
	}

	return calls
}

func lastAssistant(transcript []domain.Message) int {
	for i := len(transcript) - 1; i >= 0; i-- {
		if transcript[i].Role == domain.RoleAssistant {
			return i
		}
	}

	return -1
}
