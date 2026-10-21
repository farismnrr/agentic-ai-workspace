package provision

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

var errPublicPrivilegesOpen = errors.New("public schema privileges remain open")

var errSchemaOwnerMismatch = errors.New("schema owner mismatch")

const (
	roleExistsSQL   = `SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1)`
	schemaExistsSQL = `SELECT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = $1)`
	schemaOwnerSQL  = `SELECT pg_get_userbyid(nspowner) FROM pg_namespace WHERE nspname = $1`
	literalQuery    = `SELECT format('%L', $1::text)`
	publicAccessSQL = `SELECT has_schema_privilege('public', 'public', 'CREATE'), ` +
		`has_schema_privilege('public', 'public', 'USAGE')`
)

func apply(ctx context.Context, s session, database, name, password string) error {
	tx, err := s.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin provisioning transaction: %w", err)
	}
	defer func(ctx context.Context) {
		rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Second)
		defer cancel()
		_ = tx.Rollback(rollbackCtx)
	}(ctx)

	if err := applyRole(ctx, tx, name, password); err != nil {
		return err
	}
	if err := applySchema(ctx, tx, name); err != nil {
		return err
	}
	if err := applyPrivileges(ctx, tx, database, name); err != nil {
		return err
	}
	if err := verifyPublicPrivileges(ctx, tx); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit provisioning transaction: %w", err)
	}
	return nil
}

func applyRole(ctx context.Context, tx transaction, name, password string) error {
	roleID := pgx.Identifier{name}.Sanitize()
	literal, err := quotePassword(ctx, tx, password)
	if err != nil {
		return err
	}
	roleExists, err := exists(ctx, tx, roleExistsSQL, name)
	if err != nil {
		return err
	}
	verb := "CREATE ROLE"
	if roleExists {
		verb = "ALTER ROLE"
	}
	statement := fmt.Sprintf(
		"%s %s WITH LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT CONNECTION LIMIT 10 PASSWORD %s",
		verb, roleID, literal,
	)
	if _, err := tx.Exec(ctx, statement); err != nil {
		return fmt.Errorf("reconcile project role: %w", err)
	}
	return nil
}

func applySchema(ctx context.Context, tx transaction, name string) error {
	roleID := pgx.Identifier{name}.Sanitize()
	schemaExists, err := exists(ctx, tx, schemaExistsSQL, name)
	if err != nil {
		return err
	}
	schemaID := pgx.Identifier{name}.Sanitize()
	if !schemaExists {
		if _, err := tx.Exec(ctx, fmt.Sprintf("CREATE SCHEMA %s AUTHORIZATION %s", schemaID, roleID)); err != nil {
			return fmt.Errorf("create project schema: %w", err)
		}
		return nil
	}
	return verifySchemaOwner(ctx, tx, name)
}

func applyPrivileges(ctx context.Context, tx transaction, database, name string) error {
	roleID := pgx.Identifier{name}.Sanitize()
	schemaID := pgx.Identifier{name}.Sanitize()
	databaseID := pgx.Identifier{database}.Sanitize()
	publicSchema := pgx.Identifier{"public"}.Sanitize()
	statements := []string{
		fmt.Sprintf("REVOKE ALL ON SCHEMA %s FROM PUBLIC", publicSchema),
		fmt.Sprintf("REVOKE ALL ON DATABASE %s FROM %s", databaseID, roleID),
		fmt.Sprintf("GRANT CONNECT ON DATABASE %s TO %s", databaseID, roleID),
		fmt.Sprintf("ALTER ROLE %s SET search_path = %s", roleID, schemaID),
	}
	for _, statement := range statements {
		if _, execErr := tx.Exec(ctx, statement); execErr != nil {
			return fmt.Errorf("apply project database privileges: %w", execErr)
		}
	}
	return nil
}

func verifyPublicPrivileges(ctx context.Context, tx transaction) error {
	var create, usage bool
	if err := tx.QueryRow(ctx, publicAccessSQL).Scan(&create, &usage); err != nil {
		return fmt.Errorf("verify public schema privileges: %w", err)
	}
	if create || usage {
		return fmt.Errorf("public schema privileges remain open: %w", errPublicPrivilegesOpen)
	}
	return nil
}

func quotePassword(ctx context.Context, tx transaction, password string) (string, error) {
	var literal string
	if err := tx.QueryRow(ctx, literalQuery, password).Scan(&literal); err != nil {
		return "", fmt.Errorf("quote project password: %w", err)
	}
	return literal, nil
}

func exists(ctx context.Context, tx transaction, query, name string) (bool, error) {
	var found bool
	if err := tx.QueryRow(ctx, query, name).Scan(&found); err != nil {
		return false, fmt.Errorf("check database object: %w", err)
	}
	return found, nil
}

func verifySchemaOwner(ctx context.Context, tx transaction, name string) error {
	var owner string
	if err := tx.QueryRow(ctx, schemaOwnerSQL, name).Scan(&owner); err != nil {
		return fmt.Errorf("read project schema owner: %w", err)
	}
	if owner != name {
		return errSchemaOwnerMismatch
	}
	return nil
}
