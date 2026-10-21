# Lnjut dah kita mau implement 2 fitur

Task: 20c2ef530617785d0cefb7b799dc5729
Status: done
Updated: 2026-10-21T02:32:27Z

Reference material; confirm applicability before reusing.

**Summary:** Current official documentation supports the proposed server-side connection checks and dial-time destination policy.  
Key review risks are SSRF enforcement at the actual dial, credential leakage through errors or logs, and the MongoDB driver’s recent security fixes.

## Technical references

- **Versions and compatibility.** Project `go.mod` declares Go 1.27.2. Current upstream releases found: pgx **v5.11.0**, MySQL driver **v1.10.1**, and MongoDB Go driver **v2.9.2**. The MongoDB driver requires Go 1.25+; MySQL maintainers support MySQL 8.0+ and MariaDB 10.11+. [pgx releases](https://github.com/jackc/pgx/releases), [MySQL releases and changelog](https://github.com/go-sql-driver/mysql/blob/master/CHANGELOG.md), [MySQL requirements](https://github.com/go-sql-driver/mysql), [MongoDB release notes](https://www.mongodb.com/docs/drivers/go/current/reference/release-notes/), [MongoDB driver requirements](https://github.com/mongodb/mongo-go-driver).  
  MongoDB v2.9.2 fixes **CVE-2026-81521**; v2.9.1 fixed **CVE-2026-88031**. Use a current patched v2 release. [MongoDB release notes](https://www.mongodb.com/docs/drivers/go/current/reference/release-notes/)

- **Connection checks.** With pgx, `pgxpool.NewWithConfig` returns before establishing a connection; call `Ping(ctx)` to verify reachability. MongoDB `Connect` is also non-blocking for discovery; use `client.Ping(ctx, readpref.Primary())`, then disconnect. MongoDB v2 supports a custom context dialer through `options.Client().SetDialer(...)`. [pgxpool](https://pkg.go.dev/github.com/jackc/pgx/v5/pgxpool), [MongoDB driver usage and Ping](https://github.com/mongodb/mongo-go-driver), [MongoDB `SetDialer`](https://pkg.go.dev/go.mongodb.org/mongo-driver/v2/mongo/options)

- **MySQL and MariaDB.** Use the driver through `database/sql`; its current maintained support includes MySQL 8.0+ and MariaDB 10.11+. For a policy-enforcing dialer, the current API is `RegisterDialContext`; `RegisterDial` is deprecated. Because registration is package-global and keyed by network name, avoid mutable per-request global registration in concurrent tests; use an isolated name/connector strategy or another way to supply the guarded dial function. `Config.FormatDSN()` builds a DSN, and database names are path-escaped. [Driver README](https://github.com/go-sql-driver/mysql), [dial registration source](https://github.com/go-sql-driver/mysql/blob/master/driver.go), [DSN guidance](https://github.com/go-sql-driver/mysql/blob/master/README.md)

- **Dial-time address policy.** Go’s `net.Dialer.Control` runs after the network connection is created but before it is established; its callback receives the resolved `network` and `address`. The repository already implements `Control` and parses the supplied host/port as an IP in [egress.go](/workspace/internal/platform/egress/egress.go). For the planned `PrivateAllowed` behavior, retain the check at the actual dial and classify IPv4 and IPv6 there; checking DNS separately before connecting does not prevent rebinding. [Go `net.Dialer`](https://pkg.go.dev/net#Dialer), [OWASP SSRF Prevention](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html)

- **Secret storage and connection strings.** Existing `secretbox` uses AES-256-GCM with a random nonce and stores the key at `DATA_DIR/secret.key` ([cipher.go](/workspace/internal/platform/secretbox/cipher.go)). Go’s AEAD contract requires a unique nonce per key and authenticates ciphertext on `Open`. Build database URLs with `net/url` fields and escaped user/password values; do not concatenate raw credentials. For MySQL, prefer its `Config.FormatDSN()` to hand-built DSNs. [Go AES-GCM API](https://pkg.go.dev/crypto/cipher), [OWASP Secrets Management](https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html), [MySQL DSN guidance](https://github.com/go-sql-driver/mysql/blob/master/README.md)

- **Container environment injection.** The project passes `sandbox.Spec.Env` to Docker’s container configuration ([container.go](/workspace/internal/platform/sandbox/container.go)). Docker supports environment variables on container creation, but its docs also show that attached stdout/stderr are retained in container logs. Keep values out of command arguments, generated commands, task/event text, error strings, and stdout/stderr; pass them only through the container environment and redact at logging boundaries. [Docker run environment and logs](https://docs.docker.com/reference/cli/docker/container/run/), [OWASP Logging](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html)

## Quality checks

- **SSRF and network policy:** test public, RFC1918, loopback, IPv4/IPv6 link-local (including `169.254.169.254`), unspecified, multicast, special-use ranges, IPv4-mapped IPv6, and the app’s own listening port. Test a hostname resolving to both allowed and blocked addresses, plus DNS changes between resolution and dialing. Confirm the driver’s custom dialer is used for every connection and that the sandbox remains network-disabled.

- **Connection behavior:** test each supported driver for a reachable database, rejected login, closed port, bad TLS, malformed host/port, and timeout under the overall 10-second deadline. MongoDB discovery can retry until its server-selection timeout, so ensure that timeout fits inside the route deadline.

- **Secrets and errors:** assert API responses and logs do not contain passwords, full connection URLs/DSNs, or decrypted values. Include credentials with URL-reserved characters and control characters in tests. Confirm saved secret values are absent from list/read responses and do not cross project boundaries.

- **Project and container scope:** test owner authorization on the connection-test endpoint, per-user rate limiting, project A receiving only A’s variables, project B receiving none of A’s values, and rejected reserved names. Check that logs do not record the injected environment.

- **Accessibility:** verify keyboard-only operation, visible focus, programmatic labels, masked secret fields, announced test results (`aria-live`), text-plus-icon status, and WCAG 2.2 AA contrast at desktop and mobile widths. [WCAG 2.2](https://www.w3.org/TR/WCAG22/)

## Not found

- No existing external-database driver or connection-test implementation in the project. The current `go.mod` does not yet pin these three drivers.
- I did not verify advisories for pgx or MySQL beyond the upstream release/changelog pages linked above.

## Suggestions

- Treat `Control`/custom dialer coverage as a release-critical security check; a preflight DNS check alone is insufficient.
- Pin patched versions and review transitive advisories when adding the drivers. Keep the failure response categorized but generic, since driver errors may contain target or credential details.
- **What ran:** read repository notes/code and checked official upstream documentation. No tests or project commands were run; no files were changed.

---

## Report: running the built binary with `start_site`

**The reported “No such file” error is explained by the site workspace mount:** `start_site` sees the project directory at `/workspace`, and `bin/` is excluded from Docker’s site image build context but is not excluded from that mount. The repo’s own `.gitignore` also excludes `bin/`.

**Build the executable into a path inside the project workspace that isn’t excluded, then start that path.** The app can run as a site process, but using Sites to verify the main application is a poor fit; run it directly with `PORT=8081` for that check.

### What the code does

- The site runtime resolves the project directory and mounts that whole directory as the sandbox workspace. It does not copy selected files or apply `.gitignore` or `.dockerignore` when mounting it. [`runtime.go`](/workspace/internal/modules/site/adapter/runtime/runtime.go:85), [`mount.go`](/workspace/internal/platform/sandbox/mount.go:33)
- The site uses the configured sandbox image, runs `bash -c <command>`, and sets `SITE_PORT`, `PORT`, `HOST=127.0.0.1`, and `BASE_PATH`. It waits up to 60 seconds for an HTTP response. [`runtime.go`](/workspace/internal/modules/site/adapter/runtime/runtime.go:101), [`runtime.go`](/workspace/internal/modules/site/adapter/runtime/runtime.go:124)
- The sandbox image is the toolbox defined by [`sandbox/Dockerfile`](/workspace/sandbox/Dockerfile). It is Debian Bookworm based and has Go installed. Containers run as UID/GID from the server process. [`container.go`](/workspace/internal/platform/sandbox/container.go:118), [`sandbox/Dockerfile`](/workspace/sandbox/Dockerfile)
- Each container has a 2 GiB default memory limit and 512 MiB per-file write limit; project command runs also have a 2 GiB workspace quota. These are not reported as a site-specific binary-size limit. [`limits.go`](/workspace/internal/platform/sandbox/limits.go:16), [`quota.go`](/workspace/internal/modules/coding/adapter/runner/quota.go:12)
- The sandbox network is disabled. For sites, the only route back to the server is the Unix socket proxy; `sandbox-entry.sh` bridges the chosen site port to that socket. [`container.go`](/workspace/internal/platform/sandbox/container.go:120), [`sandbox-entry.sh`](/workspace/sandbox/sandbox-entry.sh:5)
- `.dockerignore` excludes `bin`, but this controls Docker build context, not the site’s workspace mount. `.gitignore` also has `bin/`. [` .dockerignore`](/workspace/.dockerignore:1), [`.gitignore`](/workspace/.gitignore:6)

### Recommended way to try it

Build into a non-excluded workspace path, such as `./local-build/agentic-ai-workspace`, rather than `./bin/agentic-ai-workspace`. For example:

```sh
CGO_ENABLED=0 go build -trimpath -o ./local-build/agentic-ai-workspace ./cmd/workspace
```

Then call `start_site` with command `./local-build/agentic-ai-workspace` and port `8081`. The `backend-build` Make target uses `CGO_ENABLED=0` and writes to `bin/`, which is the path at issue. [`Makefile`](/workspace/Makefile:28), [`.gitignore`](/workspace/.gitignore:6)

**Inference:** a `CGO_ENABLED=0` Linux binary should run in the Debian-based sandbox without glibc; this project’s normal backend build already produces that static artifact. The repository does not document a separate “Go/glibc in Sites” compatibility guarantee.

### Is Sites a sensible way to run this app?

Sites is designed to host a project’s web app at `/run/<project-id>/` through the main server, and expects the site process to answer HTTP on the selected port. [`docs/sites.md`](/workspace/docs/sites.md:3), [`docs/sites.md`](/workspace/docs/sites.md:26)

The main application is also an HTTP server: its address is `:<PORT>`, with port 8081 as the default. [`config.go`](/workspace/internal/platform/config/config.go:17), [`config.go`](/workspace/internal/platform/config/config.go:134) So it could answer the site readiness check if its runtime dependencies and data directory are suitable. But that would launch a separate app instance inside a project sandbox, not verify the normal local server or its data. The direct run specified in the task, `PORT=8081 ./bin/agentic-ai-workspace`, is the relevant way to start the built server; `make run` additionally invokes verification and rebuilds first. [`Makefile`](/workspace/Makefile:22)

### Not found

- I found no rule applying `.gitignore` or `.dockerignore` exclusions to the mounted site workspace.
- I found no site-specific overall workspace size limit; the cited 2 GiB quota is for project command runs.
- I did not run the binary or `start_site`.

### Suggestions

- For a site-based experiment, put the binary in the mounted project directory under a path such as `local-build/`, then use that path in the start command and choose the same port.
- For starting the main app, use the direct binary command from the task. The `make run` OOM remains a separate verification issue.

**What I did:** read the relevant project documents and site/sandbox implementation, then searched for mount rules, exclusions, resource limits, image details, and start-command conventions. No files were changed and no project commands or tests were run.

---

## Research report: per-project PostgreSQL schema provisioning

**The current `dbconn` feature saves one external connection per project, encrypts its password, and tests connectivity; it does not provision schemas or roles.**  
**Official PostgreSQL and pgx docs support schema-based separation with project-specific login roles, but this adds remote lifecycle and privilege-management work.**

### What the project does today

- [`domain/connection.go`](/workspace/internal/modules/dbconn/domain/connection.go) defines PostgreSQL, MySQL, and MongoDB connection details: host, port, database, user, password, TLS, and environment-variable name. `URL()` builds the string using `net/url`, escaping credentials and the database path; PostgreSQL gets `sslmode=disable` or `require`.
- [`application/service.go`](/workspace/internal/modules/dbconn/application/service.go) checks project ownership, seals the password, and saves the connection. It also writes the generated URL into that project’s secret Env variable. **Save does not connect to the server or provision anything.**
- [`adapter/sqlite/store.go`](/workspace/internal/modules/dbconn/adapter/sqlite/store.go) stores one row per project. Migration [`044_project_db_connections.sql`](/workspace/internal/platform/database/migrations/044_project_db_connections.sql) defines that row and cascades its deletion when the project row is removed. This cascade only deletes local SQLite data.
- [`adapter/probe/postgres.go`](/workspace/internal/modules/dbconn/adapter/probe/postgres.go) parses the URL with pgx, installs the project’s `PrivateAllowed` dial policy, and calls `Ping`. [`adapter/probe/probe.go`](/workspace/internal/modules/dbconn/adapter/probe/probe.go) converts driver failures to safe categories.
- **Likely fit (inference):** add a provisioning port to `dbconn/application` and a PostgreSQL adapter wired through [`bootstrap/dbconn.go`](/workspace/internal/bootstrap/dbconn.go). `Service.Save` is the natural application boundary for provisioning before saving the generated agent connection. The current record has no separate administrator credentials, schema/role identifiers, or generated tenant password, so those inputs and their sealed storage would need a design decision.

### PostgreSQL and pgx guidance

- PostgreSQL schemas are namespaces inside one database, but are not rigidly isolated: access depends on privileges. `CREATE SCHEMA` requires database-level `CREATE`; `GRANT USAGE` permits access to objects in a schema, and `GRANT CREATE` permits creating objects there. A project role needs the appropriate schema privileges to use and create its own objects. [Schemas and privileges](https://www.postgresql.org/docs/current/ddl-schemas.html), [CREATE SCHEMA](https://www.postgresql.org/docs/current/sql-createschema.html)
- A dedicated role with `LOGIN`, a generated high-entropy password, and no elevated attributes is a reasonable connection identity. PostgreSQL supports `CONNECTION LIMIT`, but documents it as approximate. Neon’s role guide also recommends strong passwords and gives a minimum entropy threshold. [CREATE ROLE](https://www.postgresql.org/docs/current/sql-createrole.html), [Neon role management](https://neon.com/docs/manage/roles)
- Set that role’s `search_path` to its project schema so unqualified object names land there. Avoid putting writable schemas in a path for untrusted users: PostgreSQL warns that this can change how other users’ queries resolve names. [Schema search path security](https://www.postgresql.org/docs/current/ddl-schemas.html)
- `PUBLIC` is shared by all roles. PostgreSQL 15 changed the default so `PUBLIC` no longer has `CREATE` on `public`, but upgraded databases can retain the older grant. Revoking `CREATE` (and, if the intended policy requires it, `USAGE`) from `PUBLIC` affects everyone in that database, not just this project. PostgreSQL also grants `PUBLIC` `EXECUTE` on new functions by default; per-schema default-privilege revokes cannot undo a global default grant. [PostgreSQL 15 changes](https://www.postgresql.org/docs/15/release-15.html), [ALTER DEFAULT PRIVILEGES](https://www.postgresql.org/docs/current/sql-alterdefaultprivileges.html)
- Identifiers cannot be passed as ordinary SQL value parameters. pgx provides `pgx.Identifier(...).Sanitize()` for safe SQL interpolation. Do not insert a project ID directly into DDL; derive a stable, restricted schema/role name or safely quote it. [pgx `Identifier.Sanitize`](https://pkg.go.dev/github.com/jackc/pgx/v5#Identifier.Sanitize)
- `IF NOT EXISTS` makes `CREATE SCHEMA` tolerate an existing name, but does not establish that the existing schema has the expected owner or grants. Treat retries as reconciliation: verify state before changing it. On cleanup, `DROP SCHEMA ... CASCADE` removes contained objects; `DROP OWNED` revokes privileges and removes objects owned by a role in the current database, and may need to run in each database before `DROP ROLE`. [CREATE SCHEMA](https://www.postgresql.org/docs/current/sql-createschema.html), [DROP OWNED](https://www.postgresql.org/docs/18/sql-drop-owned.html), [DROP ROLE](https://www.postgresql.org/docs/current/sql-droprole.html)
- Managed-service permissions vary. RDS does not provide a true PostgreSQL superuser; its `rds_superuser` member can create roles. Supabase documents that some superuser operations are unavailable. The feature should verify the actual account’s capabilities and report a clear provisioning failure. [RDS role permissions](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Appendix.PostgreSQL.CommonDBATasks.Roles.rds_superuser.html), [Supabase superuser limits](https://supabase.com/docs/guides/database/postgres/roles-superuser)

### Other database types

- **MySQL/MariaDB:** a database per project plus a login per project is a sensible shared-server analogue. MySQL treats `SCHEMA` as a synonym for `DATABASE`; database-level grants can scope access. Creating accounts requires administrative privileges, so managed-service rights still matter. [MySQL CREATE DATABASE](https://dev.mysql.com/doc/refman/8.4/en/create-database.html), [MySQL GRANT](https://dev.mysql.com/doc/refman/8.4/en/grant.html), [MariaDB GRANT](https://mariadb.com/docs/server/reference/sql-statements/account-management-sql-statements/grant)
- **MongoDB:** a database per project with a database-scoped user/role is also supported. MongoDB roles apply to database resources; Atlas support for user-management commands varies by cluster tier. [MongoDB roles](https://www.mongodb.com/docs/manual/tutorial/manage-users-and-roles/), [Atlas command support](https://www.mongodb.com/docs/manual/reference/command/)

### Risks and QA checks

- **Privilege escalation:** keep administrator credentials server-side and separate from the project login. Verify the project role cannot create roles, access another schema, or use `public` contrary to the policy.
- **Injection and collisions:** test unusual, long, Unicode, and SQL-shaped project IDs; verify generated identifiers are stable, valid, and safely quoted.
- **Partial failure and cleanup:** test failures after role creation, schema creation, grants, and local persistence. Project deletion currently cascades only local rows; remote schema/role cleanup needs an explicit lifecycle hook and retry strategy.
- **Secret handling:** seal generated passwords using the project’s existing secret mechanism; keep passwords and full connection strings out of logs, errors, and API responses. Test password rotation and failed-save recovery.
- **Unit tests with fakes:** cover deterministic identifier generation, safe SQL construction, call ordering, retry/state checks, compensation after failures, and no-secret logging.
- **PostgreSQL integration tests:** a real PostgreSQL server (for example, a test container) is needed to verify actual role/schema privileges, `search_path`, `PUBLIC` behavior, denied cross-project access, connection limits, repeat provisioning, and cleanup. **The sandbox has no real PostgreSQL server available.** Managed-service compatibility also needs testing against each supported provider/account type.

### Not found

- No existing provisioning adapter, administrator-credential model, generated-role password storage, or remote cleanup hook.
- No settled answer on whether generated tenant credentials are stored separately from the admin connection, or how they rotate.
- I did not verify exact role-creation permissions for every Supabase, Neon, or Atlas plan.

### Suggestions

- Ask the Planner to specify the admin-credential source, generated credential storage/rotation, schema and role naming, and behavior when provisioning succeeds but local persistence fails.
- Treat cleanup as a required lifecycle design: deleting a project’s SQLite connection row does not remove its remote schema or role.
- **What I did:** read the project guidance, notes, `dbconn` implementation, and migration; checked official PostgreSQL, pgx, MySQL, MariaDB, MongoDB, and managed-service documentation. No files were changed and no tests or project commands were run.

---

## Research report: shared PostgreSQL schema provisioning

The repository’s existing PostgreSQL probe uses pgx and the project’s `PrivateAllowed` dial policy; `dbconn` has no provisioning adapter yet. Sources: [postgres.go](/workspace/internal/modules/dbconn/adapter/probe/postgres.go), [service.go](/workspace/internal/modules/dbconn/application/service.go), and the approved PRD at `project-notes/tasks/20c2ef530617785d0cefb7b799dc5729/prd/750.md`.

The documented pgx v5 APIs support a custom dialer and transaction-scoped DDL. Key implementation risks are safely quoting the generated password and accounting for privileges inherited through `PUBLIC` or role membership.

### Technical references

**pgx v5 connection and transaction**

The current tagged pgx release found is **v5.11.0**, published September 7, 2026. Its API provides `pgx.ParseConfig`, `pgx.ConnectConfig`, `(*Conn).Begin`, `pgx.Tx.Exec`, `Commit`, and `Rollback`. `pgconn.DialFunc` has signature `func(ctx context.Context, network, addr string) (net.Conn, error)`. Sources: [pgx v5.11.0 API](https://pkg.go.dev/github.com/jackc/pgx/v5@v5.11.0), [pgconn v5.11.0 API](https://pkg.go.dev/github.com/jackc/pgx/v5/pgconn).

Expected connection pattern:

```go
cfg, err := pgx.ParseConfig(adminURL)
if err != nil { /* return safe category */ }
cfg.DialFunc = egress.Dialer(egress.Policy{
    Rule: egress.PrivateAllowed,
    OwnPort: ownPort,
}).DialContext

conn, err := pgx.ConnectConfig(ctx, cfg)
if err != nil { /* return safe category */ }
defer conn.Close(ctx)

tx, err := conn.Begin(ctx)
if err != nil { /* return safe category */ }
defer tx.Rollback(ctx)

// tx.Exec(ctx, sql, args...) for statements
// Commit only after every check and statement succeeds.
```

pgx says `Begin`’s context only applies to the begin command; cancellation does not auto-rollback. The deferred rollback is therefore useful on failures. `pgx.Identifier{...}.Sanitize()` quotes identifiers for SQL interpolation. Ordinary values use `$1` parameters, but that does not substitute for identifiers. Sources: [pgx `Begin` and `Tx`](https://pkg.go.dev/github.com/jackc/pgx/v5#Conn.Begin), [pgx `Identifier.Sanitize`](https://pkg.go.dev/github.com/jackc/pgx/v5#Identifier.Sanitize).

**Transaction-safe SQL and safe password literal**

The reviewed PostgreSQL `CREATE ROLE`, `ALTER ROLE`, `CREATE SCHEMA`, `GRANT`, and `REVOKE` command references do not mark these statements as forbidden inside a transaction block. This allows role, schema, and privilege reconciliation in one transaction. `CREATE ROLE` requires `CREATEROLE` or superuser; managed-service admin accounts may not have that capability. Sources: [CREATE ROLE](https://www.postgresql.org/docs/18/sql-createrole.html), [ALTER ROLE](https://www.postgresql.org/docs/18/sql-alterrole.html), [CREATE SCHEMA](https://www.postgresql.org/docs/18/sql-createschema.html), [GRANT](https://www.postgresql.org/docs/18/sql-grant.html), [REVOKE](https://www.postgresql.org/docs/18/sql-revoke.html).

PostgreSQL role password syntax takes a literal rather than a bind parameter. One PL/pgSQL-free option is to ask PostgreSQL to quote the value, then execute the returned literal as a separate statement on the same connection:

```sql
SELECT format('%L', $1)
```

Scan that result, then use it only in the `PASSWORD <literal>` position of `CREATE ROLE` or `ALTER ROLE`. PostgreSQL documents `%L` as literal quoting equivalent to `quote_nullable`; for a non-null generated password it produces a quoted SQL literal. Do not log the returned literal or statement text containing it. Source: [PostgreSQL `format`, `%L`, and `quote_literal`](https://www.postgresql.org/docs/18/functions-string.html).

pgx’s internal `sanitize` package is not a suitable app-level shortcut: it is internal, and its docs say it is safe only when `standard_conforming_strings` is on. Use server-side quoting or a separately reviewed, tested literal-quoting method. Source: [pgx internal sanitize docs](https://pkg.go.dev/github.com/jackc/pgx/v5/internal/sanitize).

**Provisioning SQL and existence checks**

Use quoted `pgx.Identifier` names for role/schema/database identifiers. A straightforward idempotent flow is:

```sql
SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = $1);
SELECT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = $1);
```

`pg_roles` exposes role attributes without exposing password hashes; `pg_namespace` contains `nspname` and `nspowner`. If the role exists, reconcile its password and required flags with `ALTER ROLE`; otherwise create it. If the schema exists, verify `pg_get_userbyid(nspowner)` matches the expected role rather than trusting `IF NOT EXISTS`. Otherwise use:

```sql
CREATE SCHEMA <quoted_schema> AUTHORIZATION <quoted_role>;
```

Then use statements of the form:

```sql
REVOKE ALL ON SCHEMA public FROM PUBLIC;
REVOKE ALL ON DATABASE <quoted_database> FROM <quoted_role>;
GRANT CONNECT ON DATABASE <quoted_database> TO <quoted_role>;
ALTER ROLE <quoted_role> CONNECTION LIMIT 10;
ALTER ROLE <quoted_role> SET search_path = <quoted_schema>;
```

The role should explicitly be `LOGIN NOSUPERUSER NOCREATEDB NOCREATEROLE NOINHERIT`. Note: PostgreSQL’s `REVOKE ALL ON DATABASE` removes `CONNECT`, `CREATE`, and `TEMPORARY`; a subsequent `GRANT CONNECT` expresses “CONNECT only” for direct database ACLs. Effective privileges can still come from `PUBLIC` or inherited role membership, so verify the actual policy or ensure the role has no memberships. Sources: [pg_roles](https://www.postgresql.org/docs/18/view-pg-roles.html), [pg_namespace](https://www.postgresql.org/docs/18/catalog-pg-namespace.html), [database REVOKE syntax](https://www.postgresql.org/docs/18/sql-revoke.html), [CREATE SCHEMA ownership](https://www.postgresql.org/docs/18/sql-createschema.html), [ALTER ROLE settings and limits](https://www.postgresql.org/docs/18/sql-alterrole.html).

**PostgreSQL 15+ and default privileges**

PostgreSQL 15 removed `PUBLIC`’s default `CREATE` privilege on `public` for new clusters and newly created databases. Upgrades and restored databases preserve existing grants, so older permissions can remain. The PRD’s `REVOKE ALL ON SCHEMA public FROM PUBLIC` is broader than that default change: it also revokes `USAGE` for everyone and affects the whole database. This supports the PRD’s dedicated-database requirement. Sources: [PostgreSQL 15 release notes](https://www.postgresql.org/docs/15/release-15.html), [schema docs](https://www.postgresql.org/docs/18/ddl-schemas.html).

Default privileges need separate consideration: per-schema `REVOKE` entries cannot remove privileges granted globally by default. PostgreSQL also grants `PUBLIC` execute on new functions by default. A project role that owns and creates objects in its schema can expose those objects under default ACL behavior unless the owning role’s defaults are deliberately managed. Source: [ALTER DEFAULT PRIVILEGES](https://www.postgresql.org/docs/18/sql-alterdefaultprivileges.html).

### Quality checks

**Fake executor tests:** assert SQL sequence and transaction boundaries; role creation versus reconciliation; schema creation versus existing-schema owner verification; password rotation; every identifier passed through safe quoting; password values never interpolated except as a server-quoted literal; error categories do not expose URLs/passwords; rollback occurs on each failure; commit occurs only on success. Re-run with “already exists” responses to prove idempotence. Test SQL-shaped and unusual project IDs through the deterministic name derivation.

**Optional integration test (`PG_ADMIN_TEST_URL`):** skip when unset. With a disposable dedicated database and sufficiently privileged admin, assert repeat provisioning succeeds; the role exists with login and expected restrictive flags/connection limit; schema owner is correct; role search path resolves to that schema; `CONNECT` succeeds while direct database `CREATE`/`TEMPORARY` are absent; project role can create/use objects in its schema and cannot access another project’s schema. Check `public` behavior carefully because the PRD revokes its privileges globally.

### Not found

- No live PostgreSQL server or `PG_ADMIN_TEST_URL` was available or exercised.
- The PRD does not state whether `TEMPORARY` should be denied explicitly beyond its “CONNECT only” database-grant policy.
- Managed-service-specific role privileges were not checked for every provider.

### Suggestions

- Keep transaction-scoped DDL and the custom pgx dialer in the new adapter, with a short overall context deadline.
- Make the global effect of revoking `PUBLIC` privileges clear in deployment docs and require a dedicated database.
- Treat inherited grants and default privileges as part of the integration test’s effective-access checks, not just direct ACL assertions.

**What I did:** read the project notes, PRD, backend memory, and existing `dbconn` code; checked official pgx and PostgreSQL references. No files were changed; no tests or project commands were run.