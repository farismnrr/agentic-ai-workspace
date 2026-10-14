# System Design

This project is designed to be easy to run and easy to grow without turning into one large, tangled codebase.

## The simple version

At runtime there is only one program:

```text
Browser
  |
  | HTTP
  v
Go binary
  |
  +-- Fiber handles API requests
  |
  +-- Fiber serves the built Svelte app
```

The Svelte frontend is compiled during the build and embedded inside the Go executable. Node.js is not needed on the server after the binary has been built.

That gives us a few practical benefits:

- one executable to deploy;
- one process to monitor;
- one public port to configure;
- no frontend server to manage separately.

## How a build works

```text
Svelte + TypeScript
  -> type checking and linting
  -> Vite production build
  -> frontend files
  -> embedded into Go
  -> Go tests and linting
  -> final executable
```

The normal command is:

```bash
make build
```

The result is:

```text
bin/agentic-ai-workspace
```

## Backend layout

```text
cmd/workspace/
  main.go

internal/bootstrap/
  app.go

internal/modules/<feature>/
  domain/
  application/
  adapter/

internal/platform/
  config/
  httpserver/
  webassets/
```

### `cmd/workspace`

This is only the program entry point. It starts the application and should stay small.

### `internal/bootstrap`

This is where the real pieces are connected together. For example, a use case can be connected to its database or HTTP adapter here.

### `internal/modules`

Business features live here. Each feature can have up to three layers:

- `domain`: business rules and important business types;
- `application`: use cases and the interfaces they need;
- `adapter`: HTTP, database, or external-service implementations.

A feature should only create the layers it actually needs. Empty folders do not make the architecture better.

### `internal/platform`

This contains technical details shared by the process, such as configuration, the Fiber server, and the embedded frontend.

Business rules do not belong here.

## Accounts, sign-in, and connectors

The binary also hosts its own sign-in and an OAuth 2.1 authorization server, so AI apps can connect to it without any extra service.

- **Accounts** (`modules/account`): email and password (argon2id) with a cookie session. Data is kept in SQLite under `DATA_DIR`, using a pure-Go driver so the binary stays static.
- **Spaces** (`modules/space`, `modules/coding`): each space is an MCP endpoint at `/api/<name>`. The first, `coding`, offers tools to list, read, search, change files in, and run tasks in the caller's own workspace folder (`DATA_DIR/workspaces/<user id>`, created on first use and removed with the account). Each call resolves the folder from the verified token, so one person can never name another's files. Paths are confined with Go's `os.Root`, and secret files are never shown.
- **Authorization server** (`modules/oauth`): issues opaque access tokens (stored hashed) tied to one space, plus rotating refresh tokens. A replayed code or refresh token revokes the whole connection.

What an AI app discovers from `CONNECTOR_URL/api/coding` (for example `https://domain.com/api/coding`):

1. A call without a token gets `401` and a pointer to `/.well-known/oauth-protected-resource/api/coding`.
2. That document names the authorization server and the permissions (`coding:read`).
3. `/.well-known/oauth-authorization-server` (and the OpenID Connect twin) lists the endpoints, PKCE `S256`, and the two ways to register: Client ID Metadata Documents and dynamic registration.
4. The app sends the person to `/oauth/authorize`, which opens the consent page in the app. After approval the app exchanges the code at `/oauth/token`.

Each tool has a permission scope (`coding:read`, `coding:write`, `coding:exec`) and a switch the person controls in Settings. A call must pass the switch first, then the token's scope, and is always written to an activity log. Edits save the old content first (under `DATA_DIR/backups`), which is what the Undo button restores. Tasks are named commands from `.workspace/tasks.json`, run without a shell and one at a time; the tools can never read or write that folder.

Tasks run inside `platform/sandbox`: the binary re-executes itself into new user, mount, PID, network, IPC and UTS namespaces and `pivot_root`s into the **toolbox**, a read-only Alpine Linux root with Git, Node.js, Python, Go, Rust and a C compiler that the owner built with `sandbox install` (a checksum-verified minirootfs plus `apk add`, run unprivileged in its own namespace). Nothing of the host's programs or accounts is visible. On top go a private `/tmp`, a minimal `/dev` and `/proc`, and the person's workspace at `/workspace`. The task is started inside its own cgroup v2 group (`CLONE_INTO_CGROUP`) with `memory.max`, `memory.swap.max=0`, `memory.oom.group`, `pids.max` and `cpu.max`; the server gets the delegation from systemd (`Delegate=yes`), moves itself into a `supervisor` child and enables the cpu, memory and pids controllers. Without that delegation tasks are refused unless the owner sets `SANDBOX_LIMITS=relaxed`, which falls back to rlimits (private memory, file size, a process backstop). Either way the command then runs with all capabilities dropped, no-new-privs, and a seccomp filter, a deny list modelled on Docker's default profile: no mounting, no new namespaces (`clone` with namespace flags, `unshare`, `setns`), no BPF, io_uring, userfaultfd, keyring, tracing, module, clock or swap calls, and no foreign-ABI calls; `clone3` is answered as unknown so libraries fall back to `clone`. A shared pool (`MAX_SANDBOXES`) bounds how many sandboxes run at once across all people, and a watcher stops a task whose workspace grows past 2 GiB. Not covered: disk quotas (the watcher polls once a second, so use filesystem quotas for a hard limit), the internet (tasks have none, so dependencies cannot be downloaded), and the kernel itself, which a sandbox always shares with the host.

### The agent

There is no separate chat: every message starts a thread in a project (`modules/task`), and the agent decides whether to answer directly or to plan and work. Reading is automatic; anything that changes files or runs a check waits for the person's yes. A run that changed nothing ends as an answer (status Done, no receipt) and the thread stays open for the next message. The agent loop sends the conversation to the chosen model, runs tools inside the project folder, and records plain-language steps and a provider-neutral transcript. Because the transcript is stored, a task can pause for approval and resume later.

- **Models:** Anthropic Messages and OpenAI Chat Completions over plain HTTP. Gemini is reached through Google's OpenAI-compatible endpoint (`/v1beta/openai`) with the same client, and a gateway works too if it runs on the same computer. Every outgoing model call goes through `platform/egress`, which checks the real IP when the connection opens: hosted providers may reach only public addresses, gateways only loopback and never this server's own port, with no redirects and no proxy variables. API keys are sealed with AES-256-GCM using `DATA_DIR/secret.key`.
- **Approvals:** reads, searches and edits run on their own (edits are undoable); deleting, moving and running checks wait for a yes. A task can also ask before every change.
- **Proof:** built only from checks the server ran after the last change. Claims the AI makes without running anything are listed as claimed.
- **Checks:** a project's checks live in `.workspace/tasks.json` inside its folder, which the AI's tools can never read or write; only the Checks tab changes them.
- **Limits:** 40 model turns, 15 minutes and a token budget per run; a project runs one task at a time; interrupted tasks are marked and can be continued.

Your data (`modules/mydata`): `GET /api/account/export` streams a zip built from sections that bootstrap assembles from the other features (`bootstrap/export.go`). `DELETE /api/tasks` clears a person's tasks, cancelling running ones. `POST /api/auth/delete` checks the password, then runs the erasers other features registered (tasks, then projects and their folders) before deleting the user row; remaining rows go with it through foreign-key cascades.

Task pages and the workspace follow changes live over server-sent events (`GET /api/tasks/:id/stream` for steps and streamed text, `GET /api/stream` for list changes); a client that reconnects resumes from `Last-Event-ID`, and the workspace falls back to a slow poll.

Sign-in, sign-up and the open OAuth endpoints are rate limited per visitor address (failed sign-ins also per account and address, so someone else's wrong guesses never lock you out). Expired sessions, codes, tokens and unused clients are swept away every hour.

The database schema changes only through numbered files in `internal/platform/database/migrations/`. Applied files are checksummed and can never be edited, so every change stays on record.

## Why dependencies point inward

The main rule is that business code should not depend on infrastructure details.

```text
adapter -> application -> domain
```

For example, a domain rule should not need to know which HTTP framework or database is being used.

This makes business behavior easier to test and makes infrastructure easier to replace later.

## Frontend layout

```text
web/src/
  main.ts
  app/
  features/<feature>/
    domain/
    application/
    ui/
  shared/
    ui/
    config/
```

The frontend uses strict TypeScript.

Reusable UI belongs in `shared/ui`. Feature-specific UI stays with the feature that owns it. daisyUI is the default component layer, with Tailwind utilities used for layout and small adjustments.

## Keeping files small

Human-written source, configuration, scripts, markup, and documentation are limited to 200 lines per file.

The goal is not to split files randomly. The goal is to notice when a file has accumulated too many responsibilities and separate those responsibilities cleanly.

Functions also have size and complexity limits enforced by linting.

## Quality checks

`make verify` checks:

1. repository guardrails;
2. Svelte and TypeScript correctness;
3. frontend linting;
4. production frontend build;
5. Go module consistency;
6. Go tests;
7. strict Go linting and architecture rules.

Work happens on the `dev` branch. Two GitHub Actions workflows run the same checks:

- **Fast** runs on every push to `dev`. It verifies the code and builds the Linux binary quickly, so problems show up early.
- **Full** is started manually from `dev`. It repeats every check from a clean checkout, then merges `dev` into `main` and publishes the verified binary as the rolling **Latest** GitHub release.

## Deployment

For normal use, deployment only needs the Linux binary and any environment variables required by the application.

There is no Node.js runtime, frontend server, or additional application port to deploy.
