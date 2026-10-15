# agentic-ai-workspace

An AI workspace for people who are not developers. You describe what you need, watch every step, and accept only work that a check has proven.

It ships as one Go binary with the Svelte web app built in. One process, one port.

## Quick start

You need [Docker](https://docs.docker.com/get-docker/).

```bash
curl -fsSL https://raw.githubusercontent.com/farismnrr/agentic-ai-workspace/main/scripts/install.sh | sh
```

The script asks for a few settings, starts the app and prints its address (`http://localhost:8081` by default). To install by hand, see [docs/deployment.md](./docs/deployment.md).

## Build and deploy

```bash
# Build the image; the version shows in /api/health and the X-App-Version header
docker build --build-arg VERSION=1.0.0 -t agentic-ai-workspace .

# Build and run the binary without Docker
make run
```

- `scripts/deploy-binary.sh` swaps a new binary into the running container without a rebuild, and rolls back if it fails to start.
- The `Jenkinsfile` runs checkout, test, build, push and deploy.

Details, commands and proof: [docs/devops.md](./docs/devops.md).

## Stack

Go 1.27.2, Fiber v3, Svelte 5 with strict TypeScript, Tailwind CSS and daisyUI.

## Documentation

| Doc | Read it to |
| --- | --- |
| [deployment.md](./docs/deployment.md) | Install, update, back up and configure the app |
| [devops.md](./docs/devops.md) | Build the image, hot-swap the binary, run the Jenkins pipeline |
| [using-the-app.md](./docs/using-the-app.md) | Add a model, create projects, run tasks |
| [sites.md](./docs/sites.md) | Open the web apps the AI builds |
| [connector.md](./docs/connector.md) | Connect ChatGPT and other AI apps |
| [system-design.md](./docs/system-design.md) | Understand how the code is organized |
| [design-system.md](./docs/design-system.md) | Follow the colors, theme and component rules |

## Project layout

```text
cmd/workspace/        starts the app
internal/bootstrap/   wires everything together
internal/modules/     features (domain -> application <- adapter)
internal/platform/    HTTP server, config, embedded web app
web/                  Svelte source
docs/                 documentation
agents/               notes for AI agents (see AGENTS.md)
```
