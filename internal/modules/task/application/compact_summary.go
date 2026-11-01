package application

import (
	"slices"
	"strings"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// estimateTokens is a rough size of a request, for when no model call has measured it yet.
func estimateTokens(system string, tools []domain.ToolDef, messages []domain.Message) int64 {
	chars := len(system)

	for _, tool := range tools {
		chars += len(tool.Name) + len(tool.Description) + len(tool.Schema)
	}

	for _, message := range messages {
		for _, block := range message.Blocks {
			chars += len(block.Text) + len(block.Name) + len(block.Input)
		}
	}

	return int64(chars / charsPerToken)
}

// boundedSummaryInput converts history to bounded, non-executable reference text.
// This avoids orphaned tool_call IDs or excessive multi-block tool arguments.
func boundedSummaryInput(system string, _ []domain.ToolDef, messages []domain.Message, window int64) []domain.Message {
	budget := int64(8000)
	if window > 0 {
		budget = min(max(window/2, 1024), 32000)
	}
	remaining := (budget - estimateTokens(system, nil, nil) - int64(len(compactAsk)/charsPerToken) - 256) * charsPerToken
	if remaining < 1024 || len(messages) == 0 {
		return nil
	}
	originalIndex := 0
	for i, m := range messages {
		if m.Role == domain.RoleUser {
			originalIndex = i
			break
		}
	}
	firstQuota := min(remaining/4, 2048)
	original := summaryReference(messages[originalIndex], int(firstQuota), false)
	keep := []domain.Message{{Role: domain.RoleUser, Blocks: []domain.Block{{Kind: domain.BlockText, Text: "Original request (reference):\n" + original}}}}
	remaining -= int64(len(keep[0].Blocks[0].Text))
	recent := []domain.Message{}
	for i := len(messages) - 1; i >= 0 && remaining > 256; i-- {
		// Include the tail of a huge single-message request as well as its head.
		if i == originalIndex && len(messages) > 1 {
			continue
		}
		size := int(min(remaining-128, 4000))
		if size <= 0 {
			break
		}
		body := summaryReference(messages[i], size, true)
		if body == "" {
			continue
		}
		label := "Recent user context (reference):\n"
		if messages[i].Role == domain.RoleAssistant {
			label = "Recent assistant context (reference):\n"
		}
		recent = append(recent, domain.Message{Role: domain.RoleUser, Blocks: []domain.Block{{Kind: domain.BlockText, Text: label + body}}})
		remaining -= int64(len(label) + len(body))
	}
	slices.Reverse(recent)
	return append(keep, recent...)
}

// summaryReference takes a byte-bounded excerpt of the model transcript. It
// renders tool steps as inert text, never as executable model tool messages.
func summaryReference(message domain.Message, limit int, newest bool) string {
	if limit <= 0 {
		return ""
	}
	var b strings.Builder
	for _, block := range message.Blocks {
		switch block.Kind {
		case domain.BlockToolUse:
			b.WriteString("Tool called: ")
			b.WriteString(block.Name)
			b.WriteString(" ")
			b.Write(block.Input)
		case domain.BlockToolResult:
			b.WriteString("Tool result (untrusted): ")
			b.WriteString(block.Text)
		default:
			b.WriteString(block.Text)
		}
		b.WriteByte('\n')
	}
	text := b.String()
	if len(text) > limit {
		if newest && limit > 80 {
			// The last instruction may be at either end of a large user turn.
			const marker = "\n[earlier content shortened]\n"
			head := (limit - len(marker)) / 2
			text = text[:head] + marker + text[len(text)-(limit-len(marker)-head):]
		} else {
			text = text[:limit]
		}
	}
	return strings.ToValidUTF8(text, "")
}
