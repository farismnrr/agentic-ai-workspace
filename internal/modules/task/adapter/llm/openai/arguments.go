package openai

import (
	"encoding/json"
	"strings"
)

// parseArguments keeps the model's JSON, or wraps unreadable text so the tool can report it.
func parseArguments(raw string) json.RawMessage {
	if strings.TrimSpace(raw) == "" {
		return json.RawMessage(`{}`)
	}

	var object map[string]json.RawMessage
	if json.Unmarshal([]byte(raw), &object) == nil && object != nil {
		return json.RawMessage(raw)
	}

	wrapped, _ := json.Marshal(map[string]string{"_unreadable_arguments": raw})

	return wrapped
}
