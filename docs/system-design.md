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
