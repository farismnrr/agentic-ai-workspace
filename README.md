# agentic-ai-workspace

A small web-native AI team that turns plain-language requests into visible, verified work and ships as a single Linux executable.

The frontend is built with Svelte, then embedded into the Go binary. In production you only run one file: no Node.js server, no second port, and no separate frontend process.

## Stack

- Go 1.27.2
- Fiber v3
- Svelte 5 with strict TypeScript
- Tailwind CSS + daisyUI
- One process and one HTTP port at runtime

## Run it

```bash
curl -LO https://raw.githubusercontent.com/farismnrr/agentic-ai-workspace/main/compose.latest.yaml
DOCKER_GID=$(stat -c %g /var/run/docker.sock) docker compose -f compose.latest.yaml up -d
```

Then open `http://localhost:8081`. To build from source, run `make run` (needs Go, Node.js and Docker). There is intentionally no dev server or hot reload: build first, then run the binary.

## Documentation

| Read | For |
| --- | --- |
| [docs/deployment.md](./docs/deployment.md) | Running with Docker, settings, releases, and Docker security notes |
| [docs/using-the-app.md](./docs/using-the-app.md) | Adding a model, projects, checks, and your data |
| [docs/sites.md](./docs/sites.md) | Web apps the AI starts and how they are kept apart from the app |
| [docs/connector.md](./docs/connector.md) | Connecting ChatGPT and other AI apps |
| [docs/system-design.md](./docs/system-design.md) | How the code is organized |
| [docs/design-system.md](./docs/design-system.md) | Colors, theme and component rules |

## Project structure

```text
cmd/workspace/              starts the application
internal/bootstrap/         wires the application together
internal/modules/           business features
internal/platform/          HTTP server, config, embedded frontend
web/                        Svelte frontend source
docs/                       documentation for people
agents/                     notes and memory for AI agents (see AGENTS.md)
```

Business features follow Clean Architecture inside `internal/modules/<feature>/`:

```text
domain -> application <- adapter
```

The important idea is simple: business rules stay independent from Fiber, databases, and other infrastructure details.

For a more detailed explanation, read [docs/system-design.md](./docs/system-design.md).
