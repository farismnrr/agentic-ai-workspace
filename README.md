# agentic-ai-workspace

An AI workspace for people who are not developers. You describe what you need, watch every step, and accept only work that a check has proven.

It ships as one Go binary with the Svelte web app built in: one process, one port.

## Quick start

You need [Docker](https://docs.docker.com/get-docker/). Then run:

```bash
curl -fsSL https://raw.githubusercontent.com/farismnrr/agentic-ai-workspace/main/scripts/install.sh | sh
```

The script asks for a few settings, starts the app, and prints its address (`http://localhost:8081` by default).

To install by hand, see [deployment.md](./docs/deployment.md).

## Documentation

| Doc | Read it to |
| --- | --- |
| [using-the-app.md](./docs/using-the-app.md) | Add a model, create projects, run tasks |
| [sites.md](./docs/sites.md) | Open the web apps the AI builds |
| [extensions.md](./docs/extensions.md) | Build and install browser extensions |
| [connector.md](./docs/connector.md) | Connect ChatGPT and other AI apps |
| [deployment.md](./docs/deployment.md) | Install, update, back up and configure the app |
| [devops.md](./docs/devops.md) | Build the image, hot-swap the binary, run the Jenkins pipeline |
| [system-design.md](./docs/system-design.md) | Understand how the code is organized |
| [design-system.md](./docs/design-system.md) | Follow the colors, theme and component rules |

## Stack

- Go 1.27.2 and Fiber v3
- Svelte 5 with strict TypeScript
- Tailwind CSS and daisyUI

## Project layout

```text
cmd/workspace/        starts the app
internal/bootstrap/   wires everything together
internal/modules/     features, one folder each
internal/platform/    HTTP server, config, embedded web app
web/                  Svelte source
docs/                 documentation for people
agents/               notes for AI agents (see AGENTS.md)
```
