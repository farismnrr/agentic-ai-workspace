package domain

import (
	"reflect"
	"strings"
	"testing"
)

func TestReconcilePlanKeepsStableIDsAndState(t *testing.T) {
	current := []PlanStep{
		{ID: "s1", Title: " Read input ", State: PlanDone},
		{ID: "custom", Title: "Write output", State: PlanActive},
		{ID: "s8", Title: "Remove me", State: PlanPending},
	}
	incoming := []PlanStep{{Title: "read   input"}, {ID: "custom", Title: "Write better output"}, {Title: "Check"}}

	got, diff := ReconcilePlan(current, incoming)
	want := []PlanStep{
		{ID: "s1", Title: "read   input", State: PlanDone},
		{ID: "custom", Title: "Write better output", State: PlanActive},
		{ID: "s9", Title: "Check", State: PlanPending},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ReconcilePlan() = %#v, want %#v", got, want)
	}
	if !reflect.DeepEqual(diff.Added, []string{"s9"}) || !reflect.DeepEqual(diff.Removed, []string{"s8"}) {
		t.Fatalf("unexpected diff: %#v", diff)
	}
}

func TestReconcilePlanIdenticalIsNoop(t *testing.T) {
	current := []PlanStep{{ID: "s1", Title: "Check", State: PlanDone}}
	got, diff := ReconcilePlan(current, []PlanStep{{Title: "Check", State: PlanDone}})
	if !reflect.DeepEqual(got, current) || diff.Changed {
		t.Fatalf("expected unchanged plan; got %#v, diff %#v", got, diff)
	}
}

func TestRedactCredentials(t *testing.T) {
	input := "Authorization: Bearer abc password=hunter2 sk-12345 ghu_12345 " +
		"github_pat_12345 AKIA12345678 xoxb-12345 " +
		"eyJhbGci.eyJzdWI.signature -----BEGIN PRIVATE KEY-----secret-----END PRIVATE KEY----- " +
		"https://user:pass@example.test"
	clean, count := Redact(input)
	if count != 10 {
		t.Fatalf("redaction count = %d, want 10: %s", count, clean)
	}
	secrets := []string{
		"abc", "hunter2", "sk-12345", "ghu_12345", "github_pat_12345",
		"AKIA12345678", "xoxb-12345", "eyJhbGci", "user:pass",
	}
	for _, secret := range secrets {
		if strings.Contains(clean, secret) {
			t.Errorf("secret %q remains in %q", secret, clean)
		}
	}
}
