# agentic-ai-workspace

A small web-native AI team that turns plain-language requests into visible, verified work and ships as a single Linux executable.

The frontend is built with Svelte, then embedded into the Go binary. In production you only run one file: no Node.js server, no second port, and no separate frontend process.

## Stack

- Go 1.27.2
- Fiber v3
- Svelte 5 with strict TypeScript
- Tailwind CSS + daisyUI
- One process and one HTTP port at runtime

## Fastest way to run it

Download the latest Linux build:

```bash
curl -L \
  -o agentic-ai-workspace \
  https://github.com/farismnrr/agentic-ai-workspace/releases/latest/download/agentic-ai-workspace-linux-amd64
chmod +x agentic-ai-workspace
./agentic-ai-workspace
```

By default the app listens on port `8080`.

Use a different port with:

```bash
PORT=9000 ./agentic-ai-workspace
```

The GitHub release is a rolling build named **Latest**. After a new change reaches `main` and passes all checks, the Linux binary in that release is replaced automatically.

## Building from source

You need Go and Node.js only when building the project yourself.

```bash
make build
./bin/agentic-ai-workspace
```

Or build and run in one command:

```bash
make run
```

There is intentionally no development server or hot reload. The supported local flow is always build first, then run the resulting binary.

## Quality checks

Run everything with:

```bash
make verify
```

This checks the frontend, Go code, architecture guardrails, lint rules, tests, and production frontend build.

GitHub Actions runs the same quality checks on every push and pull request. A Linux release is published only after all required checks pass.

## Design system

The UI uses the Masih Awam visual language through semantic daisyUI themes instead of copying the LMS implementation. The app ships a single light theme that defines the roles for actions, surfaces, and statuses.

See [docs/design-system.md](./docs/design-system.md) for the color roles, interaction states, and component rules.

## Project structure

```text
cmd/workspace/              starts the application
internal/bootstrap/         wires the application together
internal/modules/           business features
internal/platform/          HTTP server, config, embedded frontend
web/                        Svelte frontend source
docs/                       human-readable project documentation
```

Business features follow Clean Architecture inside `internal/modules/<feature>/`:

```text
domain -> application <- adapter
```

The important idea is simple: business rules stay independent from Fiber, databases, and other infrastructure details.

For a more detailed explanation, read [docs/system-design.md](./docs/system-design.md).
