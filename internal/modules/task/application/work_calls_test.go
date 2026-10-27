package application

import (
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestCleanWorkEventRedactsCommandAndRunSummary(t *testing.T) {
	event := domain.Event{
		Text:      "command token=secret-value",
		Technical: "DATABASE_PASSWORD=hunter2",
		Run:       &domain.RunInfo{Name: "go test --token=secret-value", ExitCode: 1},
	}
	cleanWorkEvent(&event)
	for _, value := range []string{event.Text, event.Technical, event.Run.Name} {
		if strings.Contains(value, "secret-value") || strings.Contains(value, "hunter2") {
			t.Errorf("secret remained in stored work event: %q", value)
		}
	}
}
