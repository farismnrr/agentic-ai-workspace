package bootstrap

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// streamStep answers the way a streaming model does: the same reply as the
// scripted one, but cut into small server-sent chunks.
func streamStep(w http.ResponseWriter, step string) {
	var parsed struct {
		Usage   map[string]int `json:"usage"`
		Choices []struct {
			Message struct {
				Content   *string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}

	_ = json.Unmarshal([]byte(step), &parsed)

	w.Header().Set("Content-Type", "text/event-stream")

	send := func(value any) {
		data, _ := json.Marshal(value)
		_, _ = fmt.Fprintf(w, "data: %s\n\n", data)

		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
	}
	delta := func(value map[string]any) { send(map[string]any{"choices": []any{map[string]any{"delta": value}}}) }

	message := parsed.Choices[0].Message
	if message.Content != nil {
		half := len(*message.Content) / 2
		delta(map[string]any{"content": (*message.Content)[:half]})
		delta(map[string]any{"content": (*message.Content)[half:]})
	}

	for index, call := range message.ToolCalls {
		half := len(call.Function.Arguments) / 2
		delta(map[string]any{"tool_calls": []any{map[string]any{
			"index": index, "id": call.ID, "type": "function",
			"function": map[string]string{"name": call.Function.Name, "arguments": call.Function.Arguments[:half]},
		}}})
		delta(map[string]any{"tool_calls": []any{map[string]any{
			"index": index, "function": map[string]string{"arguments": call.Function.Arguments[half:]},
		}}})
	}

	send(map[string]any{"choices": []any{}, "usage": parsed.Usage})
	_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
}

// textThenCall is a reply with some words followed by a tool call.
func textThenCall(text, id, name string, args any) string {
	var reply map[string]any

	_ = json.Unmarshal([]byte(toolCall(id, name, args)), &reply)

	choices, _ := reply["choices"].([]any)
	choice, _ := choices[0].(map[string]any)
	message, _ := choice["message"].(map[string]any)
	message["content"] = text

	encoded, _ := json.Marshal(reply)

	return string(encoded)
}
