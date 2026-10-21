package provision

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/application"
)

func TestProvisionErrorsAreCategorizedWithoutSecrets(t *testing.T) {
	secretURL := "postgres://admin:admin-secret@host/private" //nolint:gosec // Fake URL used to verify redaction.
	adminFailure := &pgconn.PgError{Code: "28P01", Message: "bad admin-secret in " + secretURL}
	privilegeFailure := &pgconn.PgError{Code: "42501", Message: "permission denied; password=project-secret"}
	for _, test := range []struct {
		err  error
		want string
	}{
		{err: adminFailure, want: "admin_rejected"},
		{err: privilegeFailure, want: "insufficient_privileges"},
		{err: errDialFailed, want: "provision_failed"},
	} {
		got := classifyError(test.err).Error()
		if got != test.want || strings.Contains(got, "admin-secret") || strings.Contains(got, "project-secret") {
			t.Fatalf("got unsafe error %q, want %q", got, test.want)
		}
	}
}

func TestGeneratedPasswordIsBase64URLFor32RandomBytes(t *testing.T) {
	password, err := generatedPassword()
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.RawURLEncoding.DecodeString(password)
	if err != nil || len(decoded) != 32 {
		t.Fatalf("password did not encode 32 bytes: len=%d err=%v", len(decoded), err)
	}
}

func TestProvisionReturnsAdminAddressAndProjectIdentity(t *testing.T) {
	adminURL := "postgres://admin:admin-secret@db.example:5544/tenantdb?sslmode=require" //nolint:gosec // Fake test URL.
	name := projectName("project-7")
	fake := &fakeSession{tx: &fakeTx{passwordLiteral: "'generated'", schemaOwner: name}}
	adapter := New(adminURL, "8081")
	adapter.connect = func(_ context.Context, gotURL, ownPort string) (session, error) {
		if gotURL != adminURL || ownPort != "8081" {
			t.Fatalf("connector received unexpected configuration")
		}
		return fake, nil
	}
	result, err := adapter.Provision(context.Background(), "project-7")
	if err != nil {
		t.Fatal(err)
	}
	assertProvisionAddress(t, result)
	assertProvisionIdentity(t, result, name)
}

func assertProvisionAddress(t *testing.T, result application.ProvisionResult) {
	t.Helper()
	if result.Host != "db.example" || result.Port != 5544 || result.Database != "tenantdb" {
		t.Fatalf("unexpected admin database address: %#v", result)
	}
}

func assertProvisionIdentity(t *testing.T, result application.ProvisionResult, name string) {
	t.Helper()
	if result.Schema != name || result.Role != name || result.Password == "" || !result.TLS {
		t.Fatalf("unexpected provision result: %#v", result)
	}
}

var errDialFailed = errors.New("dial failed for postgres://admin:admin-secret@host/private")

func TestAdminTLSUsesExplicitSSLMode(t *testing.T) {
	for _, test := range []struct {
		url  string
		want bool
	}{
		{url: "postgres://admin@db/work?sslmode=require", want: true},
		{url: "postgres://admin@db/work?sslmode=disable", want: false},
		{url: "postgres://admin@db/work", want: false},
	} {
		if got := adminTLS(test.url); got != test.want {
			t.Errorf("adminTLS(%q) = %t, want %t", test.url, got, test.want)
		}
	}
}

func TestProvisionRequiresAdminURLDatabaseName(t *testing.T) {
	for _, rawURL := range []string{"postgres://admin@host", "not a url"} {
		adapter := New(rawURL, "8081")
		if _, err := adapter.Provision(context.Background(), "project-7"); err == nil || err.Error() != "admin_url_invalid" {
			t.Errorf("Provision(%q) error = %v, want safe admin_url_invalid", rawURL, err)
		}
	}
}
