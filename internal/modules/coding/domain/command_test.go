package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

func TestCommandTaskRunsThroughBash(t *testing.T) {
	task, err := domain.CommandTask("  go test ./... && echo ok ", 30)
	if err != nil {
		t.Fatal(err)
	}

	if task.Command[0] != "bash" || task.Command[2] != "go test ./... && echo ok" || task.Timeout() != 30*time.Second {
		t.Fatalf("unexpected task %+v", task)
	}
}

func TestCommandTaskRejectsEmptyAndHugeCommands(t *testing.T) {
	for _, command := range []string{"", "   ", strings.Repeat("x", domain.MaxCommandLength+1)} {
		if _, err := domain.CommandTask(command, 0); !errors.Is(err, domain.ErrBadCommand) {
			t.Errorf("command of %d bytes: got %v", len(command), err)
		}
	}
}

func TestCommandTaskCapsTheTimeout(t *testing.T) {
	task, _ := domain.CommandTask("sleep 1", 100000)
	if task.Timeout() != domain.MaxTaskTimeout {
		t.Fatalf("timeout %v", task.Timeout())
	}
}
