package bootstrap

import (
	"strings"
	"testing"
)

func TestLogoutApprovalShowsTargetAndSelectorsWithoutSecretReferences(t *testing.T) {
	described := describeSessionAction(toolArgs{
		Action: "logout", Path: "/account/logout?token=private-token",
		SubmitSelector: `button[name="logout"]`, ReadySelector: `[data-state="signed-out"]`,
		EmailSecret: "TEST_EMAIL", PasswordSecret: "TEST_PASSWORD",
	})
	for _, expected := range []string{
		"log out", "/account/logout", "submit=button[name=\"logout\"]", "ready=[data-state=\"signed-out\"]",
	} {
		if !strings.Contains(described.Ask, expected) {
			t.Errorf("approval %q does not include %q", described.Ask, expected)
		}
	}
	for _, secret := range []string{"private-token", "TEST_EMAIL", "TEST_PASSWORD"} {
		if strings.Contains(described.Ask, secret) || strings.Contains(described.Technical, secret) {
			t.Errorf("approval exposed secret reference or value %q", secret)
		}
	}
}

func TestApprovalRedactsCredentialLikeSelectorValues(t *testing.T) {
	described := describeSessionAction(toolArgs{
		Action: "login", Path: "/login", EmailSelector: `input[value="person@example.test"]`,
		SubmitSelector: `button[type="submit"]`, PasswordSecret: "PASSWORD_REF",
	})
	if strings.Contains(described.Ask, "person@example.test") || strings.Contains(described.Ask, "PASSWORD_REF") {
		t.Fatalf("approval contains a credential value or reference: %q", described.Ask)
	}
	if !strings.Contains(described.Ask, "email=input[value=[redacted]]") {
		t.Fatalf("approval omitted sanitized selector: %q", described.Ask)
	}
}
