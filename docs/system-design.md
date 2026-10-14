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

Tasks run inside `platform/sandbox`, which starts one throwaway Docker container per task through the Docker Engine API (the `moby/moby/client` module, so the binary stays static and needs no Docker CLI). The image is `SANDBOX_IMAGE` (default `node:24-bookworm`, pulled on first use). The container has no network, a read-only root filesystem, a private `/tmp` tmpfs, all capabilities dropped, `no-new-privileges`, Docker's default seccomp profile, an init process, and `memory` (swap off), `cpu`, `pids` and file-size limits. The person's workspace is mounted at `/workspace`: a bind mount when the server runs on the host, or the data volume narrowed with a volume `Subpath` (Engine API 1.45+) when the server itself runs in a container, so a task can never see another person's folder. The task runs as the server's own uid and gid, so files stay editable. Containers carry a label with an owner id derived from the data folder; at startup the server removes only the leftovers that carry its own owner id. Docker is opened lazily and warmed in the background: if the daemon is unreachable, checks fail with a plain reason and nothing else breaks. A shared pool (`MAX_SANDBOXES`) bounds how many containers run at once, and a watcher stops a task whose workspace grows past 2 GiB. `SANDBOX_RUNTIME` can select a hardened runtime such as gVisor. Not covered: disk quotas (the watcher polls once a second, so use filesystem quotas for a hard limit), the internet (tasks have none, so dependencies cannot be downloaded), and the host kernel and Docker daemon, which a container always shares with the host. Because the server needs the Docker socket, whoever controls it controls the host; Docker rootless mode is the recommended mitigation.

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

- **Fast** runs on every push to `dev`. It verifies the code, builds the binary and the Docker image, smoke-tests both, and publishes the `dev` image to GHCR.
- **Full** is started manually from `dev`. It repeats every check from a clean checkout, then merges `dev` into `main` and publishes the verified image to GHCR as `latest`.

## Deployment

For normal use, deployment is the Docker image (`ghcr.io/farismnrr/agentic-ai-workspace`) run with `compose.latest.yaml` (or `compose.dev.yaml`), which mounts the Docker socket so checks can run in containers, plus any environment variables the application needs.

There is no Node.js runtime, frontend server, or additional application port to deploy.
