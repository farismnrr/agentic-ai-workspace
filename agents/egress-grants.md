# Egress grants (per-task outbound HTTPS approvals)

Design and status of `internal/modules/egressgrant`. Agent-only. Read before touching `platform/egressproxy`, `bootstrap/reach.go`, or the grant tool/endpoints.

**Status: stage 1 only (domain, migration 052, unit tests). Nothing is wired. The feature stays off (`EGRESS_GRANTS` unset) in every environment until every item in "Activation gate" is done.**

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
2. Application use cases and ports (`Request`, `Approve`, `Deny`, `Revoke`, `Active`), sqlite adapter, tests.
3. `egressproxy.Grants`: registry, normalized match, resolve-then-dial, SNI check, tunnel tracking, revoke/expiry. Security tests below.
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
