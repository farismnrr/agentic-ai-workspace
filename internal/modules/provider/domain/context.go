package domain

import (
	"errors"
	"strings"
)

// Context window sizes, in tokens.
const (
	DefaultContextWindow = 200_000
	LongContextWindow    = 1_000_000
	MinContextWindow     = 8_000
	MaxContextWindow     = 2_000_000
	longContextSuffix    = "[1m]"
)

// ErrInvalidContext rejects a context window that is not a sensible size.
var ErrInvalidContext = errors.New("choose a context window between 8,000 and 2,000,000 tokens")

// ContextWindow is how much a model can hold at once: the size a person chose, else the size its provider
// reported, else 1M for a model named like "...[1m]", else the default.
func ContextWindow(model string, chosen map[string]int64, catalog map[string]ModelInfo) int64 {
	if tokens, ok := chosen[model]; ok && tokens > 0 {
		return tokens
	}

	if info, ok := catalog[model]; ok && info.ContextWindow > 0 {
		return info.ContextWindow
	}

	if strings.HasSuffix(strings.ToLower(model), longContextSuffix) {
		return LongContextWindow
	}

	return DefaultContextWindow
}

// ValidContextWindow reports whether tokens is a size a person may choose.
func ValidContextWindow(tokens int64) bool {
	return tokens >= MinContextWindow && tokens <= MaxContextWindow
}
