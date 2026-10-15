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
- **Spaces** (`modules/space`, `modules/coding`): each space is an MCP endpoint at `/api/<name>`. The first, `coding`, offers tools to list, read, search, change files in, run tasks and commands in, and start the web app of the caller's project for that app. Each call resolves the project from the verified token (user id plus the app's name from its grant): `projectapp.ForApp` finds or makes the live project whose `source` is that name, so one person can never name another's files and two apps never share a folder. The tool catalogue (`coding/domain.Tools()`) and the switches (`tool_permissions`, `approval_settings`) are the same ones the agent obeys. Paths are confined with Go's `os.Root`, and secret files are never shown.
- **Authorization server** (`modules/oauth`): issues opaque access tokens (stored hashed) tied to one space, plus rotating refresh tokens. A replayed code or refresh token revokes the whole connection.

What an AI app discovers from `CONNECTOR_URL/api/coding` (for example `https://domain.com/api/coding`):

1. A call without a token gets `401` and a pointer to `/.well-known/oauth-protected-resource/api/coding`.
2. That document names the authorization server and the permissions (`coding:read`).
3. `/.well-known/oauth-authorization-server` (and the OpenID Connect twin) lists the endpoints, PKCE `S256`, and the two ways to register: Client ID Metadata Documents and dynamic registration.
4. The app sends the person to `/oauth/authorize`, which opens the consent page in the app. After approval the app exchanges the code at `/oauth/token`.

Each tool has a permission scope (`coding:read`, `coding:write`, `coding:exec`) and a switch the person controls in Settings. A call must pass the switch first, then the token's scope, and is always written to an activity log. Edits save the old content first (under `DATA_DIR/backups`), which is what the Undo button restores. Tasks are named commands from `.workspace/tasks.json`, run without a shell and one at a time; the tools can never read or write that folder.

Tasks and commands run inside `platform/sandbox`, which starts one throwaway Docker container per run through the Docker Engine API (the `moby/moby/client` module, so the binary stays static and needs no Docker CLI). The image is `SANDBOX_IMAGE` (default `ghcr.io/<owner>/agentic-ai-workspace-sandbox`, built from `sandbox/Dockerfile`: Python, Node.js, Go, Rust, Java, Maven and Bash, pulled on first use). Toolchains live in the image, so every person's container shares the same read-only layers and nothing is downloaded per person; the image advertises them in the `io.masihawam.toolchains` label, which the agent prompt reads. The container has no network interface, a read-only root filesystem, a private `/tmp` tmpfs, all capabilities dropped, `no-new-privileges`, Docker's default seccomp profile, an init process, and `memory` (swap off), `cpu`, `pids` and file-size limits. The person's workspace is mounted at `/workspace`, their package cache at `/cache` (`DATA_DIR/caches/<id>`, emptied above 5 GiB) and the proxy socket folder read-only at `/run/egress`: bind mounts when the server runs on the host, or the data volume narrowed with a volume `Subpath` (Engine API 1.45+) when the server itself runs in a container, so a run can never see another person's folder. Registry access goes through `platform/egressproxy`: an HTTPS `CONNECT` proxy served on one Unix socket per workspace (so the server still listens on one TCP port), allowing only `SANDBOX_EGRESS_HOSTS` on port 443, with every resolved address checked by `platform/egress` at dial time. Inside the container `sandbox-entry` bridges `127.0.0.1:3128` to that socket and the usual `HTTPS_PROXY` variables point there. `run_command` (exec scope, off by default, asks every time) runs `bash -c` in this sandbox; named tasks run their argument vector directly. Runs execute as the server's own uid and gid, so files stay editable. Containers carry a label with an owner id derived from the data folder; at startup the server removes only the leftovers that carry its own owner id. Docker is opened lazily and warmed in the background: if the daemon is unreachable, checks fail with a plain reason and nothing else breaks. A shared pool (`MAX_SANDBOXES`) bounds how many containers run at once, and a watcher stops a run whose workspace grows past 2 GiB. `SANDBOX_RUNTIME` can select a hardened runtime such as gVisor. Not covered: disk quotas (the watcher polls once a second, so use filesystem quotas for a hard limit), data smuggled out inside requests to an allowed registry (it has no credentials, but a request can still carry bytes), and the host kernel and Docker daemon, which a container always shares with the host. Because the server needs the Docker socket, whoever controls it controls the host; Docker rootless mode is the recommended mitigation. Web apps the AI starts (`modules/site`) use the same sandbox with a long-lived container and are reached through a Unix socket by a `/run/<name>/<uuid>/` proxy on the same port; see [sites.md](./sites.md) for the isolation layers.

### The agent

There is no separate chat: every message starts a thread in a project (`modules/task`), and the agent decides whether to answer directly or to plan and work. Reading is automatic; anything that changes files or runs a check waits for the person's yes. A run that changed nothing ends as an answer (status Done, no receipt) and the thread stays open for the next message. The agent loop sends the conversation to the chosen model, runs tools inside the project folder, and records plain-language steps and a provider-neutral transcript. Because the transcript is stored, a task can pause for approval and resume later.

- **Models:** Anthropic Messages and OpenAI Chat Completions over plain HTTP. Gemini is reached through Google's OpenAI-compatible endpoint (`/v1beta/openai`) with the same client, and a gateway works too if it runs on the same computer. Every outgoing model call goes through `platform/egress`, which checks the real IP when the connection opens: hosted providers may reach only public addresses, gateways only loopback and never this server's own port, with no redirects and no proxy variables. API keys are sealed with AES-256-GCM (`platform/secretbox`) using `DATA_DIR/secret.key`.
- **GitHub** (`modules/github`): each person connects their own account through a GitHub App (web flow with PKCE; tokens sealed, renewed on demand). Commands in a project get the owner's token and, on a socket of their own, `github.com`, `api.github.com` and `codeload.github.com`; sites never do. Setup is in the deployment guide.
- **Switches and approvals:** a tool the person switched off in Spaces & tools is neither offered to the model nor run (checked again right before each call). Reads run on their own; edits, deletes, moves, commands and starting a web app wait for a yes unless the person turned on Skip approvals for the space, which never overrides an off switch.
- **Proof:** built only from checks the server ran after the last change. Claims the AI makes without running anything are listed as claimed.
- **Checks:** a project's checks live in `.workspace/tasks.json` inside its folder, which the AI's tools can never read or write; only the Checks tab changes them.
- **Limits and context:** 40 model turns and 15 minutes per run. Each model has a context window (default 200K tokens, 1M when its name ends in `[1m]`, changeable per model in Settings, AI models). Before a turn, when the conversation nears the window (window minus 33K), the model writes a summary of the task so far; the summary replaces the earlier messages in what is sent (the full history stays on record), and the thread shows "I shortened my notes to keep going". If no summary comes back the task goes on, and after three misses in a run it stops trying. There is no total token budget per task; a project runs one task at a time; interrupted tasks are marked and can be continued.

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
