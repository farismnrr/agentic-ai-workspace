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

You need [Docker](https://docs.docker.com/get-docker/). Then run one command:

```bash
curl -fsSL https://raw.githubusercontent.com/farismnrr/agentic-ai-workspace/main/scripts/install.sh | sh
```

It asks for each setting one by one, saves them, starts the app, and tells you where to open it (`http://localhost:8081` by default).

Want to deploy by hand (pull the compose file, build the image yourself, or run the binary)? Follow [docs/deployment.md](./docs/deployment.md).

## Build, deploy and CI/CD

- **Build:** `docker build --build-arg VERSION=1.0.0 -t agentic-ai-workspace .` makes a static Go binary in a distroless image; the version shows in the `X-App-Version` header and `/api/health`.
- **Hotfix deploy:** `scripts/deploy-binary.sh` swaps the binary in the running container without a rebuild, and rolls back if the new one does not come up.
- **Jenkins:** the `Jenkinsfile` runs checkout, test, build, push and deploy.

The write-ups, commands and proof are in [docs/devops.md](./docs/devops.md).

## Documentation

| Read | For |
| --- | --- |
| [docs/devops.md](./docs/devops.md) | Image build, hotfix binary swap, and the Jenkins pipeline |
| [docs/deployment.md](./docs/deployment.md) | Step-by-step install (pull or build), updating, backups, settings, and Docker security notes |
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
