package provision

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestApplyRunsOrderedTransactionAndQuotesIdentifiers(t *testing.T) {
	fake := &fakeSession{tx: &fakeTx{passwordLiteral: "'quoted-secret'"}}
	projectID := `x"; DROP SCHEMA public; --`
	name := projectName(projectID)
	if strings.Contains(name, projectID) || len(name) != 26 {
		t.Fatalf("unexpected generated name %q", name)
	}
	if err := apply(context.Background(), fake, "workspace", name, "secret"); err != nil {
		t.Fatal(err)
	}
	events := fake.tx.events
	if len(events) != 13 {
		t.Fatalf("got %d events: %#v", len(events), events)
	}
	want := []string{
		"BEGIN",
		"QUERY " + literalQuery,
		"QUERY " + roleExistsSQL,
		`EXEC CREATE ROLE "` + name + `" WITH LOGIN NOSUPERUSER NOCREATEDB ` +
			`NOCREATEROLE NOINHERIT CONNECTION LIMIT 10 PASSWORD 'quoted-secret'`,
		"QUERY " + schemaExistsSQL,
		`EXEC CREATE SCHEMA "` + name + `" AUTHORIZATION "` + name + `"`,
		"EXEC REVOKE ALL ON SCHEMA \"public\" FROM PUBLIC",
		`EXEC REVOKE ALL ON DATABASE "workspace" FROM "` + name + `"`,
		`EXEC GRANT CONNECT ON DATABASE "workspace" TO "` + name + `"`,
		`EXEC ALTER ROLE "` + name + `" SET search_path = "` + name + `"`,
		"QUERY " + publicAccessSQL,
		"COMMIT",
		"ROLLBACK",
	}
	if strings.Join(events, "\n") != strings.Join(want, "\n") {
		t.Fatalf("unexpected SQL order:\n%s", strings.Join(events, "\n"))
	}
	for _, statement := range events {
		if strings.Contains(strings.ToUpper(statement), "DROP ") {
			t.Fatalf("unexpected drop statement: %s", statement)
		}
	}
}

func TestApplyReconcilesExistingRoleAndSchema(t *testing.T) {
	fake := &fakeSession{tx: &fakeTx{roleExists: true, schemaExists: true, schemaOwner: "same"}}
	if err := apply(context.Background(), fake, "db", "same", "next-password"); err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(fake.tx.events, "\n")
	if !strings.Contains(joined, `ALTER ROLE "same" WITH LOGIN`) || strings.Contains(joined, "CREATE ROLE") {
		t.Fatalf("existing role was not reconciled: %s", joined)
	}
	if strings.Contains(joined, "CREATE SCHEMA") || !strings.Contains(joined, "QUERY "+schemaOwnerSQL) {
		t.Fatalf("existing schema was not owner-checked: %s", joined)
	}
}

func TestApplyRollsBackEachFailedStep(t *testing.T) {
	for failAt := 1; failAt <= 11; failAt++ {
		t.Run(fmt.Sprintf("call_%d", failAt), func(t *testing.T) {
			fake := &fakeSession{tx: &fakeTx{passwordLiteral: "'safe'", failAt: failAt}}
			err := apply(context.Background(), fake, "db", "name", "password")
			if err == nil {
				t.Fatal("expected injected failure")
			}
			commitBeforeFailure := failAt < 11 && containsEvent(fake.tx.events, "COMMIT")
			if !containsEvent(fake.tx.events, "ROLLBACK") || commitBeforeFailure {
				t.Fatalf("failure did not roll back: %#v", fake.tx.events)
			}
		})
	}
}

func TestApplyRejectsWrongExistingSchemaOwner(t *testing.T) {
	fake := &fakeSession{tx: &fakeTx{roleExists: true, schemaExists: true, schemaOwner: "other"}}
	err := apply(context.Background(), fake, "db", "expected", "password")
	if err == nil || !containsEvent(fake.tx.events, "ROLLBACK") || containsEvent(fake.tx.events, "COMMIT") {
		t.Fatalf("owner mismatch was not rolled back: err=%v events=%#v", err, fake.tx.events)
	}
}

func TestApplyVerifiesPublicSchemaPrivilegesAreClosed(t *testing.T) {
	fake := &fakeSession{tx: &fakeTx{passwordLiteral: "'safe'", publicUsage: true}}
	err := apply(context.Background(), fake, "db", "name", "password")
	if !errors.Is(err, errPublicPrivilegesOpen) || !containsEvent(fake.tx.events, "ROLLBACK") ||
		containsEvent(fake.tx.events, "COMMIT") {
		t.Fatalf("open PUBLIC privilege was not rejected: err=%v events=%#v", err, fake.tx.events)
	}
}

func TestPasswordLiteralQueryCastsParameterToText(t *testing.T) {
	if literalQuery != `SELECT format('%L', $1::text)` {
		t.Fatalf("password literal query = %q", literalQuery)
	}
}

func containsEvent(events []string, prefix string) bool {
	for _, event := range events {
		if strings.HasPrefix(event, prefix) {
			return true
		}
	}
	return false
}

var (
	errFakeFailure     = errors.New("fake failure")
	errUnexpectedQuery = errors.New("unexpected query")
)

type fakeSession struct{ tx *fakeTx }

func (f *fakeSession) Begin(context.Context) (transaction, error) {
	f.tx.events = append(f.tx.events, "BEGIN")
	return f.tx, nil
}

func (f *fakeSession) Close(context.Context) error {
	f.tx.events = append(f.tx.events, "CLOSE")
	return nil
}

type fakeTx struct {
	passwordLiteral string
	schemaOwner     string
	events          []string
	failAt          int
	calls           int
	roleExists      bool
	schemaExists    bool
	publicCreate    bool
	publicUsage     bool
}

func (f *fakeTx) Exec(_ context.Context, statement string, _ ...any) (pgconn.CommandTag, error) {
	f.events = append(f.events, "EXEC "+statement)
	return pgconn.CommandTag{}, f.failure()
}

func (f *fakeTx) QueryRow(_ context.Context, query string, _ ...any) row {
	f.events = append(f.events, "QUERY "+query)
	return fakeRow{tx: f, query: query}
}

func (f *fakeTx) Commit(context.Context) error {
	f.events = append(f.events, "COMMIT")
	return f.failure()
}

func (f *fakeTx) Rollback(context.Context) error {
	f.events = append(f.events, "ROLLBACK")
	return nil
}

func (f *fakeTx) failure() error {
	f.calls++
	if f.calls == f.failAt {
		return errFakeFailure
	}
	return nil
}

type fakeRow struct {
	tx    *fakeTx
	query string
}

func (r fakeRow) Scan(dest ...any) error {
	if err := r.tx.failure(); err != nil {
		return err
	}
	switch r.query {
	case literalQuery:
		*dest[0].(*string) = r.tx.passwordLiteral
	case roleExistsSQL:
		*dest[0].(*bool) = r.tx.roleExists
	case schemaExistsSQL:
		*dest[0].(*bool) = r.tx.schemaExists
	case schemaOwnerSQL:
		*dest[0].(*string) = r.tx.schemaOwner
	case publicAccessSQL:
		*dest[0].(*bool) = r.tx.publicCreate
		*dest[1].(*bool) = r.tx.publicUsage
	default:
		return errUnexpectedQuery
	}
	return nil
}
