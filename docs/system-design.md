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
- **Spaces** (`modules/space`, `modules/coding`): each space is an MCP endpoint at `/api/<name>`. The first, `coding`, offers read-only tools to list, read, and search a folder you choose. Paths are confined with Go's `os.Root`, and secret files are never shown.
- **Authorization server** (`modules/oauth`): issues opaque access tokens (stored hashed) tied to one space, plus rotating refresh tokens. A replayed code or refresh token revokes the whole connection.

What an AI app discovers from `https://domain.com/api/coding`:

1. A call without a token gets `401` and a pointer to `/.well-known/oauth-protected-resource/api/coding`.
2. That document names the authorization server and the permissions (`coding:read`).
3. `/.well-known/oauth-authorization-server` (and the OpenID Connect twin) lists the endpoints, PKCE `S256`, and the two ways to register: Client ID Metadata Documents and dynamic registration.
4. The app sends the person to `/oauth/authorize`, which opens the consent page in the app. After approval the app exchanges the code at `/oauth/token`.

Each tool has a permission scope (`coding:read`, `coding:write`, `coding:exec`) and a switch the person controls in Settings. A call must pass the switch first, then the token's scope, and is always written to an activity log. Edits save the old content first (under `DATA_DIR/backups`), which is what the Undo button restores. Tasks are named commands from `.workspace/tasks.json`, run without a shell and one at a time; the tools can never read or write that folder.

### The agent

A task is one request in a project. The agent loop (`modules/task`) sends the conversation to the chosen model, runs the tools it asks for inside the project's folder, and records everything twice: as plain-language steps the person reads (the thread) and as a provider-neutral transcript used to rebuild the conversation. Because the transcript is stored, a task can stop to ask for approval or an answer and later continue, even after a restart.

- **Models:** Anthropic Messages and OpenAI Chat Completions over plain HTTP; any compatible gateway works. API keys are sealed with AES-256-GCM using `DATA_DIR/secret.key`.
- **Approvals:** reads, searches and edits run on their own (edits are undoable); deleting, moving and running checks wait for a yes. A task can also ask before every change.
- **Proof:** built only from checks the server ran after the last change. Claims the AI makes without running anything are listed as claimed.
- **Limits:** 40 model turns, 15 minutes and a token budget per run; a project runs one task at a time; interrupted tasks are marked and can be continued.

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
