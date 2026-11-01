package application

import "github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"

func orderBlocks(messages []domain.Message) []domain.Message {
	ordered := make([]domain.Message, 0, len(messages))

	for _, message := range messages {
		results := make([]domain.Block, 0, len(message.Blocks))
		rest := make([]domain.Block, 0, len(message.Blocks))

		for _, block := range message.Blocks {
			if block.Kind == domain.BlockToolResult {
				results = append(results, block)
			} else {
				rest = append(rest, block)
			}
		}

		ordered = append(ordered, domain.Message{Role: message.Role, Blocks: append(results, rest...)})
	}

	return ordered
}
