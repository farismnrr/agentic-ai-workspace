# Egress grants (per-task outbound HTTPS approvals)

Design and status of `internal/modules/egressgrant`. Agent-only. Read before touching `platform/egressproxy`, `bootstrap/reach.go`, or the grant tool/endpoints.

**Status: stages 1-3 done (domain, migration 052, use cases, SQLite store, proxy enforcement in `platform/egressproxy`, tests). Nothing is wired: no bootstrap, no config flag yet, nothing calls `Hub.TaskDir`, `Hub.WithGrants` or `Grants.Allow`, no sandbox change, no route. The feature stays off (`EGRESS_GRANTS` unset; the flag itself arrives with stage 4) in every environment until every item in "Activation gate" is done.**

## Decisions (user, 2026-10-26)

- Approvers: the project owner and admins, through the UI only. An agent may request, never approve, deny or revoke.
- Default lifetime 24 hours, hard cap 7 days; expiry is automatic.
- Scope is one task in one project (`project_id` + `task_id`), never global and never the whole workspace.
- The static `SANDBOX_EGRESS_HOSTS` list is unchanged and always applies. Grants only add to it.

## Security requirements and how each is met

1. **Internal targets and external grants never share a path.** `Hub.Dir(ctx, id, extra...)` takes strings, and a `host:port` string there becomes an exact target (private addresses, any port). Grants must NOT travel through it. They live in a separate typed registry (`egressproxy.Grants`) consulted by the proxy, and the proxy dials them only through `p.dial` (PublicOnly, port 443). A grant host is never added to `targets`, and `Granted.Targets` never receives one.
2. **Identity is bound server-side.** A sandbox only ever sees the socket folder the server mounts for it. The grant scope (`project_id`, `task_id`) is captured in the listener's closure when the server creates the socket, never read from the request (`Host` header, headers, env, path). One socket per (project, task) that holds grants; two concurrent tasks never share a socket, so one task's grant cannot be used from another's. The model/agent cannot create sockets or choose a scope. Tests: two tasks of one project, same host, only the approved one connects.
3. **Revocation and expiry are enforced in the proxy, not only on read.**
   - The registry holds `grantID -> {host, expiresAt}` in memory, set by the application when a grant is approved and removed when it is revoked or expires (timer at `expiresAt`, plus a check of `now < expiresAt` on every `CONNECT`).
   - Each live tunnel is registered under its grant id; revoke/expiry closes both ends. As a backstop, both connections get a deadline at `expiresAt`.
   - The database is the source of truth. The registry is rebuilt from approved, unexpired rows at startup; a lost registry fails closed (no access).
4. **Hostname and DNS.**
   - One normalizer (`domain.NormalizeHost`) is used when a grant is created and again by the proxy on every `CONNECT`, and the proxy compares the normalized form exactly: lower case, one trailing dot removed, IDNA (Lookup profile) to ASCII, no wildcard, no userinfo/port/path, no IP literal in any notation (dotted, decimal, hex, octal, IPv6, brackets, zone), no single-label or internal-looking names (`localhost`, `.local`, `.internal`, `.lan`, `.home.arpa`, `.localdomain`).
   - At connect time the proxy resolves the name itself, refuses it when ANY answer is non-public (rebinding with mixed answers), and dials the validated IP literal, so the name is not resolved a second time. The existing `egress.Check` `Control` callback stays as defence in depth. IPv4-mapped IPv6 is unmapped before the check (already in `egress.Check`).
   - A CNAME is followed by the resolver; only the final addresses matter and the grant stays for the name that was asked.
5. **No privilege escalation.** Approve/deny/revoke are methods that take a `Decider` built only by the HTTP adapter from the authenticated session. The agent tool path builds `Actor{Kind: Agent}` and can call `Request` only. The sandbox has no network except the proxy, and the proxy refuses the server's own port. No approval credential exists in env, files or tool results. A test fails if any tool/bridge handler can reach an approve/revoke method.
6. **Off by default.** `EGRESS_GRANTS` (read in `platform/config`) gates everything: tool not registered, endpoints 404, registry nil, `reach` returns no hosts. Enabling it before the gate is complete is a bug.
7. **Replay.** A grant id is bound to its task; approving a second time, approving a denied/revoked/expired request, or re-using a stale id after revoke returns an error. An approval for host A never covers host B.
8. **HTTPS-only is not TLS.** The proxy enforces `CONNECT` to port 443 and nothing more: after `200 Connection Established` the bytes are opaque, so a client could speak plaintext, or TLS with another SNI to the same IP (fronting). For grants the proxy therefore reads the first record, requires a TLS ClientHello and requires its SNI to equal the granted host, then forwards the bytes unchanged. It is not a TLS-terminating proxy and never sees content. A client using ECH is refused (fail closed). The static registry list is not changed by this (regression risk), and the docs must say so.

## Stages

1. Domain model, migration `052_egress_grants.sql`, unit tests. (done)
2. Application use cases and ports (`Request`, `Approve`, `Deny`, `Revoke`, `Active`, `List`), sqlite adapter, tests. (done)
3. `egressproxy.Grants`: registry, normalized match, resolve-then-dial, SNI check, tunnel tracking, revoke/expiry. Security tests below. (done)
4. Wiring behind `EGRESS_GRANTS`: per-task socket, `reach`, startup rebuild.
5. `request_egress` tool (bridge included) with `Actor{Agent}`.
6. HTTP endpoints, owner/admin authorization, Svelte panel, thread notice.
7. Docs for people (`docs/`), final gate review.

## Activation gate (all required)

- Authorization (owner/admin only) and the approval UI done.
- Proxy enforcement, revocation, expiry done.
- Integration tests: concurrent sandbox isolation, unauthorized approval (agent, non-owner, other project's owner), permission replay, DNS rebinding (single and mixed answers), active tunnel revocation, TTL expiry, SNI mismatch, and the regression suite for database/gateway tunnels, `SANDBOX_EGRESS_HOSTS`, GitHub hosts and the own-port refusal.
- Black-box run on the real app, as in `blackbox-testing`.
- `docs/` and this file updated.

## Notes

- Two migrations already share number 051; the next free number is 052. Migrations are append-only.
- Pending requests also expire after 24 hours and cannot be approved afterwards.

## Stage 2 facts (what the code guarantees)

- **Deciding rights.** `domain.Decider{ID, ProjectID, Owner, Admin}`: `Owner` counts only for `ProjectID == grant.ProjectID`; `Admin` spans projects. The HTTP adapter (stage 6) must build it from the session and from a server-side project-owner lookup, never from request fields.
- **Atomicity.** `Create` is one `INSERT ... SELECT` that checks in the same statement that the task belongs to the project and that the task has no live (pending or unexpired approved) grant for the host. `Decide` runs in one transaction whose first statement is a no-op `UPDATE` that takes SQLite's write lock before anything is read (a read-then-write transaction fails with a stale snapshot when another decision commits in between, which showed up as non-domain errors in CI); then it reads, applies the domain rule and the `UPDATE` names the old status, end, project and task, so a decision made from a stale state changes nothing; it is retried (up to 5 times) and the retry meets the domain rule (`ErrNotPending`/`ErrNotApproved`). Only lock conflicts are retried; any other database error surfaces as itself. Tested with 20-30 goroutines under `-race`.
- **Scope.** `Active` joins `tasks` on both ids, filters `status = 'approved' AND expires_at > now`, and the service re-applies `Grant.Covers` to what the store returned.
- **Not done on purpose:** no cap on pending requests per task (add one if agents flood the UI), no event/log write on decisions (stage 6), no registry or proxy hook (stage 3).
- **Rollback caveat.** Once the app has run with migration 052, an older binary refuses to start (`ErrMigrationMissing`). Roll back by deploying a build that still contains `052_egress_grants.sql`; never delete the file.
- **Two 051 files.** `051_project_gateway.sql` and `051_task_archived_at.sql` are both applied on real databases. The runner journals by full file name and sorts names, so it is safe; `migrate_order_test.go` pins both names and their order. Verified on a copy of the live database (both applied, 052 added, no earlier journal row changed, no foreign-key violations). Never rename or renumber either file.

## Static `SANDBOX_EGRESS_HOSTS` and the marimo host

On this computer `.env` currently lists `sb-f6c7943aa05fb36c.sb.molab.run` in `SANDBOX_EGRESS_HOSTS` (added 2026-10-26), so every sandbox of every person can reach it. It is a stop-gap. After dynamic grants are enabled and proven: approve that host through a grant for the task that needs it, then remove the host from `.env` and recreate only the app container. Keep the registry defaults (`pypi.org`, `files.pythonhosted.org`, `registry.npmjs.org`, `proxy.golang.org`, `sum.golang.org`, `storage.googleapis.com`, `crates.io`, `index.crates.io`, `static.crates.io`, `static.rust-lang.org`, `repo.maven.apache.org`, `repo1.maven.org`). Setting the variable replaces the defaults, so removing the host means either deleting the whole line (defaults return) or listing the defaults again.

## Stage 3 must also prove (proxy gate)

A valid ClientHello with the right SNI does not prove that everything inside the tunnel is TLS: after the first record the proxy still forwards opaque bytes, and a client can finish a handshake and then send anything, or put plaintext after a ClientHello. The proxy cannot see inside without terminating TLS, which this design does not do. So stage 3 states the limit in its docs, and tests at least: plaintext first bytes refused, a non-ClientHello record refused, SNI different from the granted host refused, no SNI refused, ECH-style outer SNI refused, a truncated or oversized first record refused with a timeout, and a ClientHello followed by plaintext still reaches only the approved public address (the limit: the destination, not the content, is what is enforced). These tests must exist before the activation gate.

## Stage 3 facts (proxy enforcement, in `platform/egressproxy`)

- **Three separate paths, one switch.** `Proxy.ServeHTTP` tries the older rules first, unchanged: registry list and exact targets (`tunnelTo`). Only when none allows the request and the proxy belongs to a task socket (`task.grants != nil`) does it call `serveGrant` (`grantpath.go`). A proxy from `Hub.Dir` has no grants, so its behaviour is byte-for-byte the old one (the old tests pass untouched). A grant host is never written to `Proxy.hosts` or `Proxy.targets` (tested), and `Hub.TaskDir`'s `extra` hosts keep going through `WithHosts` exactly as in `Dir`.
- **Identity.** `Hub.TaskDir(ctx, workspaceID, Scope{Project, Task}, extra...)` makes one socket and folder per scope. The scope is captured in the listener's proxy when the server creates it (`proxy.task.scope`); nothing in a request (`Host`, headers, path) can change it, and a sandbox is only mounted its own folder. Zero or half scopes are refused (`ErrBadGrant`) and `Lease` also refuses them. `Hub.Remove(workspaceID)` stops task sockets too; `RemoveTask(scope)` stops one task's and revokes its grants.
- **Registry.** `Grants` (`grants.go`, `lease.go`, `limits.go`): `Allow(id, scope, host, ends)` (host canonicalized by `egress.CanonicalHost`, a timer revokes at `ends`), `Revoke(id)`, `Clear(scope)`, `Close()`. `Lease(scope, host)` checks the grant, the end time, and the per-task/total limits and registers the tunnel in ONE locked step, so a revoke either sees the tunnel and closes it or the tunnel never opens (tested with 16 goroutines racing a revoke). Every connection of a tunnel (client, destination) is added to its `Lease`; `Revoke` closes them all before it returns. Closing a lease takes the registry lock again, so never close leases while holding it (`Allow` once deadlocked on that).
- **Fail closed.** Unknown, expired, revoked, over-limit, zero scope, non-canonical host, port other than 443, a failing `Verifier`, a name that does not resolve, a name with ANY non-public answer: all refuse (403, or 502 when nothing resolves, 503 over the limit). The proxy holds grants only in memory: a restart or a lost registry means no access. `Verifier` (stage 4 will back it with the database) is asked on every CONNECT after the local match; an error refuses the connection. After a database error stage 4 should also call `Grants.Close`/`Clear`.
- **DNS and destination pinning.** The name is resolved once (`lookupIP`), every answer must pass `egress.Check` PublicOnly (IPv4-mapped IPv6 unmapped; reserved, NAT64, 6to4, link-local, private and loopback refused), and the connection goes to the checked IP literal, so a second lookup can never lead elsewhere. The dialer's `Control` check stays as defence in depth. A CNAME is followed by the resolver; only final addresses matter.
- **TLS check, and its limit.** After `200 Connection Established` the proxy reads exactly one TLS record (`clienthello.go`): handshake record, version 3.1-3.3, at most 16384 bytes, one unfragmented ClientHello with exact lengths, no repeated extension, no `encrypted_client_hello`, exactly one `host_name` whose canonical form equals the granted host. Anything else closes the tunnel before the destination sees a byte. Then bytes are forwarded unchanged. **This proves the destination name, not that the rest of the tunnel is TLS**: after the hello the client can send anything to that approved host (the proxy never terminates TLS and never sees content). The docs for people must say this. Tests: plaintext, SSH, garbage, alert and application-data records, bad versions and lengths, truncation, trailing bytes, fragmentation, duplicate/ECH/multiple/non-host names, IP and wildcard names, no SNI, mismatched SNI, a fuzz target.
- **Resource limits** (`Limits`, defaults 16 per task, 128 total): the hello must arrive within `Handshake` (10 s) of the tunnel opening, `Idle` (5 min) closes a quiet tunnel (deadline refreshed on each read and write), `Dial` (10 s) bounds resolution plus connect, and every deadline is capped at the grant's end.
- **Host spelling.** `egress.CanonicalHost` (platform) and `domain.NormalizeHost` (module) cannot import each other, so `bootstrap/egress_host_test.go` asserts that for every name the domain accepts the proxy yields the same spelling. Keep that test when changing either.
- **Write lock (stage 2 follow-up).** `_txlock=immediate` on the driver is connection-wide (every transaction of every module), so the targeted no-op `UPDATE` stays. `lock_test.go` pins the driver behaviour: a decision blocks while another writer holds the lock, then sees what it committed.

## Residual risks (state them in the review)

- Nothing is wired, so none of this runs in the app. Stage 4 must build the scope only from server data, rebuild the registry from approved rows at start, call `Revoke` synchronously on revoke/expiry, and `RemoveTask` when a task ends.
- Content to an approved host cannot be inspected; approving a host trusts its operator and its DNS (an attacker who controls the approved name's DNS chooses which public address answers).
- ECH and a ClientHello split over several records are refused, so some clients will not work through a grant.
- Expiry and revoke are enforced in-process; a crash drops all grants (fail closed) but also all tunnels.
- No per-destination rate limit and no bandwidth cap beyond the connection limits.
- The static `SANDBOX_EGRESS_HOSTS` list still contains the marimo host for everyone until grants are enabled (see above).
