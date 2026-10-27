package http

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/task/domain"
)

func TestTaskPayloadIncludesExternalWorkSummary(t *testing.T) {
	task := domain.Task{
		ID: "task-1", Source: "ChatGPT", WorkID: "work-1",
		LinkedTaskID: "linked-1", OriginTaskID: "origin-1",
		Work: &domain.WorkSummary{
			ID: "work-1", State: domain.WorkBlocked, Role: "qa", RoleName: "QA",
			Actor: "ChatGPT", PlanRev: 3, Stale: true, UpdatedAt: time.Unix(100, 0).UTC(),
		},
	}
	encoded, err := json.Marshal(toPayload(task))
	if err != nil {
		t.Fatal(err)
	}
	body := string(encoded)
	for _, field := range []string{
		`"source":"ChatGPT"`, `"workId":"work-1"`, `"state":"blocked"`,
		`"linkedTaskId":"linked-1"`,
		`"originTaskId":"origin-1"`,
		`"role":"qa"`, `"roleName":"QA"`, `"actor":"ChatGPT"`,
		`"planRev":3`, `"stale":true`, `"updatedAt":`,
	} {
		if !strings.Contains(body, field) {
			t.Errorf("task payload is missing %s: %s", field, body)
		}
	}
}

func TestTaskEventWorkUsesDomainJSONNames(t *testing.T) {
	event := domain.Event{
		ID: "event-1", Kind: domain.EventRole, Text: "Developer",
		Work: &domain.WorkRef{ID: "work-1", Actor: "ChatGPT", Role: "developer", RoleName: "Developer", From: "qa"},
	}
	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	body := string(encoded)
	for _, field := range []string{
		`"work":`, `"id":"work-1"`, `"actor":"ChatGPT"`,
		`"role":"developer"`, `"roleName":"Developer"`, `"from":"qa"`,
	} {
		if !strings.Contains(body, field) {
			t.Errorf("task event is missing %s: %s", field, body)
		}
	}
}
