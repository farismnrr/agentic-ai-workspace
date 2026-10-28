package projects_test

import (
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/test/go/apptest"
)

func requireStatus(t *testing.T, response apptest.Reply, want int, action string) {
	t.Helper()
	if response.Status != want {
		t.Fatalf("%s returned %d, want %d: %s", action, response.Status, want, response.Body)
	}
}
