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
	input := "Authorization: Bearer abc password=hunter2 DATABASE_PASSWORD=db-secret sk-12345 ghu_12345 " +
		"github_pat_12345 AKIA12345678 xoxb-12345 " +
		"eyJhbGci.eyJzdWI.signature -----BEGIN PRIVATE KEY-----secret-----END PRIVATE KEY----- " +
		"https://user:pass@example.test"
	clean, count := Redact(input)
	if count != 11 {
		t.Fatalf("redaction count = %d, want 11: %s", count, clean)
	}
	secrets := []string{
		"abc", "hunter2", "db-secret", "sk-12345", "ghu_12345", "github_pat_12345",
		"AKIA12345678", "xoxb-12345", "eyJhbGci", "user:pass",
	}
	for _, secret := range secrets {
		if strings.Contains(clean, secret) {
			t.Errorf("secret %q remains in %q", secret, clean)
		}
	}
}

func TestRedactStructuredAndAdditionalCredentials(t *testing.T) {
	//nolint:gosec // synthetic value verifies the Basic authorization redaction pattern.
	basicCredential := "dXNlcjpwYXNzd29yZA=="
	cases := []struct {
		name   string
		input  string
		secret string
	}{
		{name: "json password", input: `{"password":"hunter2hunter2"}`, secret: "hunter2hunter2"},
		{name: "quoted api key", input: `"api_key":"api-secret-value"`, secret: "api-secret-value"},
		{name: "spaced password", input: `password: "my long passphrase"`, secret: "my long passphrase"},
		{name: "basic authorization", input: "Authorization: Basic " + basicCredential, secret: basicCredential},
		{name: "live key", input: "sk_live_abcdefghijk", secret: "sk_live_abcdefghijk"},
		{name: "google key", input: "AIza0123456789abcdefgh", secret: "AIza0123456789abcdefgh"},
		{name: "npm token", input: "npm_abcdefghijklmno", secret: "npm_abcdefghijklmno"},
		{name: "gitlab token", input: "glpat-abcdefghijklmno", secret: "glpat-abcdefghijklmno"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			clean, count := Redact(tc.input)
			if count == 0 || strings.Contains(clean, tc.secret) {
				t.Fatalf("Redact(%q) = %q, %d; secret remains", tc.input, clean, count)
			}
		})
	}
}

func TestRedactLeavesNonSecretLookalikes(t *testing.T) {
	input := `keynote=opening passwordless=true token_count=3 api_key_name=public`
	clean, count := Redact(input)
	if count != 0 || clean != input {
		t.Fatalf("Redact(%q) = %q, %d; want unchanged", input, clean, count)
	}
}
