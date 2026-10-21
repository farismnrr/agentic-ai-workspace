package provision

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/farismnrr/agentic-ai-workspace/internal/modules/dbconn/application"
)

func TestPostgresProvisioningIntegration(t *testing.T) {
	adminURL := os.Getenv("PG_ADMIN_TEST_URL")
	if adminURL == "" {
		t.Skip("PG_ADMIN_TEST_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	adapter := New(adminURL, "")
	project := fmt.Sprintf("integration-%d", time.Now().UnixNano())
	first, err := adapter.Provision(ctx, project)
	if err != nil {
		t.Fatal(err)
	}
	second, err := adapter.Provision(ctx, project)
	if err != nil {
		t.Fatal(err)
	}
	if first.Password == second.Password {
		t.Fatal("repeat provisioning did not rotate the password")
	}
	other, err := adapter.Provision(ctx, project+"-other")
	if err != nil {
		t.Fatal(err)
	}

	adminCfg, err := pgx.ParseConfig(adminURL)
	if err != nil {
		t.Fatal("invalid PG_ADMIN_TEST_URL")
	}
	admin, err := pgx.ConnectConfig(ctx, adminCfg)
	if err != nil {
		t.Fatal("could not connect to PG_ADMIN_TEST_URL")
	}
	closeOnCleanup(t, context.WithoutCancel(ctx), admin)
	assertRoleState(t, ctx, admin, first)
	assertSchemaOwner(t, ctx, admin, first)
	otherSchema := pgx.Identifier{other.Schema}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE TABLE "+otherSchema+".secret_table (value integer)"); err != nil {
		t.Fatal("could not create integration isolation fixture")
	}
	assertProjectIdentity(t, ctx, adminCfg, second)
	assertCannotReadOtherSchema(t, ctx, adminCfg, second, other.Schema)
}

func assertRoleState(t *testing.T, ctx context.Context, admin *pgx.Conn, result application.ProvisionResult) {
	t.Helper()
	var login, superuser, createDB, createRole, inherit bool
	var limit int32
	err := admin.QueryRow(ctx, `SELECT rolcanlogin, rolsuper, rolcreatedb, rolcreaterole, rolinherit, rolconnlimit
		FROM pg_roles WHERE rolname = $1`, result.Role).Scan(&login, &superuser, &createDB, &createRole, &inherit, &limit)
	if err != nil || !login || superuser || createDB || createRole || inherit || limit != 10 {
		t.Fatalf("unexpected project role attributes: login=%v super=%v createdb=%v createrole=%v inherit=%v limit=%d err=%v",
			login, superuser, createDB, createRole, inherit, limit, err)
	}
}

func assertSchemaOwner(t *testing.T, ctx context.Context, admin *pgx.Conn, result application.ProvisionResult) {
	t.Helper()
	var owner string
	err := admin.QueryRow(ctx,
		`SELECT pg_get_userbyid(nspowner) FROM pg_namespace WHERE nspname = $1`, result.Schema,
	).Scan(&owner)
	if err != nil || owner != result.Role {
		t.Fatalf("schema owner=%q want=%q err=%v", owner, result.Role, err)
	}
}

func assertProjectIdentity(
	t *testing.T,
	ctx context.Context,
	adminCfg *pgx.ConnConfig,
	result application.ProvisionResult,
) {
	t.Helper()
	projectCfg := *adminCfg
	projectCfg.User = result.Role
	projectCfg.Password = result.Password
	project, err := pgx.ConnectConfig(ctx, &projectCfg)
	if err != nil {
		t.Fatal("project role could not connect")
	}
	closeOnCleanup(t, context.WithoutCancel(ctx), project)
	var role, schema, searchPath string
	err = project.QueryRow(ctx, `SELECT current_user, current_schema(), current_setting('search_path')`).
		Scan(&role, &schema, &searchPath)
	if err != nil || role != result.Role || schema != result.Schema || searchPath != result.Schema {
		t.Fatalf("unexpected project identity role=%q schema=%q search_path=%q err=%v",
			role, schema, searchPath, err)
	}
}

func assertCannotReadOtherSchema(
	t *testing.T, ctx context.Context, adminCfg *pgx.ConnConfig, result application.ProvisionResult, schema string,
) {
	t.Helper()
	projectCfg := *adminCfg
	projectCfg.User = result.Role
	projectCfg.Password = result.Password
	project, err := pgx.ConnectConfig(ctx, &projectCfg)
	if err != nil {
		t.Fatal("project role could not connect")
	}
	closeOnCleanup(t, context.WithoutCancel(ctx), project)
	query := "SELECT * FROM " + pgx.Identifier{schema}.Sanitize() + ".secret_table"
	_, err = project.Exec(ctx, query)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "42501" {
		t.Fatalf("cross-schema read was not denied with insufficient privilege: %v", err)
	}
}

func closeOnCleanup(t *testing.T, ctx context.Context, conn *pgx.Conn) {
	t.Helper()
	t.Cleanup(func() {
		if closeErr := conn.Close(ctx); closeErr != nil {
			t.Errorf("close integration connection: %v", closeErr)
		}
	})
}
