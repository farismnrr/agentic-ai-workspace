# Lnjut dah kita mau implement 2 fitur

Task: 20c2ef530617785d0cefb7b799dc5729
Status: working
Updated: 2026-10-21T01:00:12Z

Reference material; confirm applicability before reusing.

## Research report: project Env and external database settings

The project currently has no per-project Env settings or external database connector.  
SearXNG is configured at server startup; sandbox commands receive a scrubbed environment and cannot reach databases under the current network rules.

### What I found in the project

**Settings tabs.** The settings route and allowed tab names are in [router.ts](/workspace/web/src/app/router.ts); the tab navigation is defined in [settings-nav.ts](/workspace/web/src/features/workspace/settings/settings-nav.ts), and [SettingsPage.svelte](/workspace/web/src/features/workspace/settings/SettingsPage.svelte) selects which tab to show. The current tabs are AI models, Persona & rules, Agent team, Spaces & tools, Connected apps, Integrations, Activity, Notifications, Usage, and Account. There is no Env tab today.

**SearXNG.** It is configured with the process-level `SEARXNG_URL`, read by [config.go](/workspace/internal/platform/config/config.go). [bootstrap/web.go](/workspace/internal/bootstrap/web.go) creates a SearXNG client when that value is present; otherwise web search is off. [client.go](/workspace/internal/modules/web/adapter/searxng/client.go) calls `/search` with JSON format and a 20-second timeout. The agent receives the `web_search` tool only when search is configured ([agenttools_web_defs.go](/workspace/internal/bootstrap/agenttools_web_defs.go)).

SearXNG has **no project settings UI, database storage, or test-connection API** in the code I inspected. The deployment guide says to run it separately, enable JSON responses in SearXNG, and test it with `curl`; a `403` indicates JSON output is not enabled ([deployment.md](/workspace/docs/deployment.md)). The requested per-project address and test button would therefore be a new capability, not a reuse of existing settings behavior.

**Secrets today.** Provider API keys are encrypted with AES-256-GCM by [secretbox/cipher.go](/workspace/internal/platform/secretbox/cipher.go); the key is held in `DATA_DIR/secret.key`, while encrypted provider keys are stored in SQLite in `providers.key_sealed` ([migration 005](/workspace/internal/platform/database/migrations/005_providers.sql), [provider store](/workspace/internal/modules/provider/adapter/sqlite/store.go)). The UI says provider keys are stored encrypted and not shown in full ([AddProviderModal.svelte](/workspace/web/src/features/workspace/settings/AddProviderModal.svelte)). This is evidence about provider keys; I found no existing per-project secret-variable storage.

**Sandbox environment and network.** Commands run in short-lived Docker containers. [runner.go](/workspace/internal/modules/coding/adapter/runner/runner.go) constructs each run’s environment; [sidecars.go](/workspace/internal/modules/coding/adapter/runner/sidecars.go) fixes the base environment and only adds proxy settings when a proxy socket is mounted. [sandbox.go](/workspace/internal/platform/sandbox/sandbox.go) defines the run specification, and [container.go](/workspace/internal/platform/sandbox/container.go) sets `NetworkDisabled: true`.

The egress proxy allows HTTPS `CONNECT` on port 443 to configured hostnames, with DNS-resolved addresses checked to reject non-public IPs ([proxy.go](/workspace/internal/platform/egressproxy/proxy.go)). **Implication:** adding a connection string to a command’s environment would not make a database reachable. Native database traffic on ports such as PostgreSQL 5432 or MySQL 3306 has no path through this proxy today. Private or local database addresses also conflict with its public-address checks. Supporting database access requires an explicit network design decision.

The existing provider “verify” endpoint is a useful reference for API shape, but it checks provider credentials and retrieves models; it is not a database connection tester ([handler.go](/workspace/internal/modules/provider/adapter/http/handler.go), [service.go](/workspace/internal/modules/provider/application/service.go)).

### Technical references and security guidance

- The [OWASP Secrets Management Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Secrets_Management_Cheat_Sheet.html) recommends limiting who can read secrets, supporting rotation and revocation, and preventing secrets from appearing in logs. The [Cryptographic Storage Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Cryptographic_Storage_Cheat_Sheet.html) discusses encryption and keeping encryption keys separate from the data they protect.
- A server-side connection test accepts a user-supplied destination, so it creates an SSRF boundary. The [OWASP SSRF Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html) covers destination validation and redirect risks. In this project, [egress.go](/workspace/internal/platform/egress/egress.go) already checks resolved IPs at dial time, but its available policies are public-only or loopback-only—not arbitrary private database access.
- Go’s [database connection guide](https://go.dev/doc/database/open-handle) documents checking connectivity with `Ping` and advises against keeping credentials in source code. The [connection management guide](https://go.dev/doc/database/manage-connections) describes pooling and connection limits. These are relevant if a test is implemented in Go; the project does not currently have an external database driver path for these requested types.
- PostgreSQL warns that `PGPASSWORD` may be visible to other local users on some systems and recommends a password file instead ([PostgreSQL environment variables](https://www.postgresql.org/docs/current/libpq-envars.html)). That warning is a reason to make the sandbox secret-delivery method a deliberate security decision, even if the requested integration ultimately uses environment variables.
- For database-side TLS, see the [MySQL 8.4 encrypted connections guide](https://dev.mysql.com/doc/refman/8.4/en/using-encrypted-connections.html) and the [PostgreSQL connection documentation](https://www.postgresql.org/docs/current/libpq-connect.html).
- For settings UI acceptance, [WCAG 2.2](https://www.w3.org/TR/WCAG22/) includes keyboard access, visible focus, labels, and minimum contrast. A masked secret field should still have a clear label and status text; success or failure should not be communicated by color alone.

### Quality checks to make checkable

- Env values are scoped to the selected project; one project cannot list, replace, or delete another project’s variables.
- After saving, secret values are not returned by list/read APIs or shown in the UI; replacing a value does not require retrieving the old value.
- Names are validated, duplicate names have a clear outcome, and add/replace/delete behavior is understandable.
- A command in the intended sandbox run receives only that project’s configured variables; another project’s run does not.
- Database test results distinguish success from failure without echoing credentials or full connection strings. Test timeouts and malformed addresses are handled.
- Test public, private, loopback, and metadata-service destinations, DNS resolution changes, redirects if HTTP is used, and unsupported DB ports. Confirm the chosen network path still enforces the project’s isolation rules.
- Check keyboard operation, visible focus, accessible labels, status announcements, mobile layout, and WCAG 2.2 AA contrast.

### Not found

- No Env tab, project-scoped variable storage, external database settings, database connection test, or matching migration/API route.
- No sandbox network path for native PostgreSQL, MySQL/MariaDB, or MongoDB connections.
- No exact requirement for database hostname scope, allowed network locations, whether values are secret by default, or whether all project commands receive them.

### Suggestions

- **Decision needed for the plan:** whether project databases may be private/local or must be publicly addressable. Current sandbox egress cannot reach private databases or native DB ports.
- Treat connection testing as a separate security boundary from storing Env values. A backend test that accepts user-controlled hosts needs destination controls; broadening it to private addresses changes the SSRF threat boundary.
- Keep the app’s own SQLite database separate from project-provided external database variables, as the request specifies.
- **No code was changed and no tests or project commands were run.**

---

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