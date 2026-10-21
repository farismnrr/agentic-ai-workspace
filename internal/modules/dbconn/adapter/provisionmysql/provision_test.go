package provisionmysql

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	mysqldriver "github.com/go-sql-driver/mysql"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/application"
	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/domain"
)

var errDriverFixture = errors.New("driver failure fixture")

func TestStatementsQuoteProjectIdentityAndPassword(t *testing.T) {
	got, err := statements("p_123", "safe'\\pass")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"CREATE DATABASE IF NOT EXISTS `p_123`",
		"CREATE USER IF NOT EXISTS 'p_123'@'%' IDENTIFIED BY 'safe''\\\\pass'",
		"ALTER USER 'p_123'@'%' IDENTIFIED BY 'safe''\\\\pass'",
		"GRANT ALL PRIVILEGES ON `p\\_123`.* TO 'p_123'@'%'",
	}
	for index := range want {
		if got[index] != want[index] {
			t.Errorf("statement %d = %q, want %q", index, got[index], want[index])
		}
	}
	if _, err := statements("x; DROP DATABASE prod", "password"); err == nil {
		t.Fatal("hostile database identifier was accepted")
	}
}

func TestProvisionIsIdempotentUsesTLSAndReturnsConnection(t *testing.T) {
	settings := domain.SharedSettings{
		Kind: "mysql", Host: "db.example", Port: 3306,
		Database: "admin", User: "admin", Password: "admin-secret", TLS: "ca",
	}
	fake := &fakeExecutor{}
	adapter := New(settings)
	adapter.connect = func(_ context.Context, config *mysqldriver.Config) (executor, error) {
		if config.User != "admin" || config.DBName != "admin" || config.Addr != "db.example:3306" ||
			config.TLS == nil || config.TLS.ServerName != "db.example" {
			t.Fatal("connector did not receive saved admin settings and TLS")
		}
		return fake, nil
	}
	first, err := adapter.Provision(context.Background(), "project-a")
	if err != nil {
		t.Fatal(err)
	}
	second, err := adapter.Provision(context.Background(), "project-a")
	if err != nil {
		t.Fatal(err)
	}
	assertMySQLReconciled(t, fake, first, second, settings)
}

func assertMySQLReconciled(
	t *testing.T, fake *fakeExecutor, first, second application.ProvisionResult, settings domain.SharedSettings,
) {
	t.Helper()
	if len(fake.statements) != 8 || first.Database != second.Database || first.Role != first.Database ||
		first.Host != settings.Host || first.Port != settings.Port || first.Password == second.Password || !first.TLS {
		t.Fatalf("provision results or statements differ: %#v %#v %#v", first, second, fake.statements)
	}
}

func TestProvisionMapsErrorsWithoutLeakingThem(t *testing.T) {
	secret := strings.Join([]string{"mysql://admin:", "admin-secret", "@db.example/admin"}, "")
	for _, test := range []struct {
		err  error
		want application.ProvisionFailure
	}{
		{err: &mysqldriver.MySQLError{Number: mysqlAccessDenied, Message: secret}, want: application.AdminRejected},
		{
			err:  &mysqldriver.MySQLError{Number: mysqlDatabaseDenied, Message: secret},
			want: application.InsufficientPrivileges,
		},
		{err: errDriverFixture, want: application.ProvisionFailed},
	} {
		got := classifyError(test.err).Error()
		if got != string(test.want) || strings.Contains(got, "admin-secret") {
			t.Fatalf("unsafe error = %q, want %q", got, test.want)
		}
	}
}

type fakeExecutor struct{ statements []string }

func (f *fakeExecutor) ExecContext(_ context.Context, statement string, _ ...any) (sql.Result, error) {
	f.statements = append(f.statements, statement)
	return fakeResult{}, nil
}

func (*fakeExecutor) Close() error { return nil }

type fakeResult struct{}

func (fakeResult) LastInsertId() (int64, error) { return 0, nil }
func (fakeResult) RowsAffected() (int64, error) { return 0, nil }
