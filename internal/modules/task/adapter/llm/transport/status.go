package transport

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

// statusError turns an HTTP failure into a ModelError; rate limits and outages may be retried.
func statusError(status int, data []byte) error {
	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	_ = json.Unmarshal(data, &body)

	message := body.Error.Message
	if message == "" {
		message = fmt.Sprintf("the provider answered HTTP %d", status)
	}

	retryable := status == http.StatusTooManyRequests || status >= http.StatusInternalServerError

	return &domain.ModelError{Message: shorten(message), Status: status, Retryable: retryable}
}

func shorten(text string) string {
	runes := []rune(text)
	if len(runes) <= maxMessage {
		return text
	}

	return string(runes[:maxMessage]) + "…"
}
