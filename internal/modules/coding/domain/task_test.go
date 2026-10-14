package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/coding/domain"
)

func TestTasksSurviveAWriteAndReadCycle(t *testing.T) {
	t.Parallel()

	want := []domain.Task{
		{Name: "lint", Description: "Style", Command: []string{"npm", "run", "lint"}},
		{Name: "test", Command: []string{"go", "test", "./..."}, TimeoutSeconds: 300},
	}

	data, err := domain.EncodeTasks(want)
	if err != nil {
		t.Fatal(err)
	}

	got, err := domain.ParseTasks(data)
	if err != nil || len(got) != 2 || got[0].Name != "lint" || got[1].TimeoutSeconds != 300 {
		t.Fatalf("round trip lost data: %+v %v", got, err)
	}
}

func TestBadTasksAreRefused(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("x", domain.MaxArgLength+1)
	cases := map[string][]domain.Task{
		"uppercase name":  {{Name: "Test", Command: []string{"go"}}},
		"no command":      {{Name: "test"}},
		"blank program":   {{Name: "test", Command: []string{""}}},
		"too long a word": {{Name: "test", Command: []string{long}}},
		"negative time":   {{Name: "test", Command: []string{"go"}, TimeoutSeconds: -1}},
		"too much time":   {{Name: "test", Command: []string{"go"}, TimeoutSeconds: 601}},
		"duplicate":       {{Name: "a", Command: []string{"go"}}, {Name: "a", Command: []string{"go"}}},
	}

	for name, tasks := range cases {
		if err := domain.ValidateTasks(tasks); !errors.Is(err, domain.ErrBadTasks) {
			t.Errorf("%s must be refused, got %v", name, err)
		}
	}
}
