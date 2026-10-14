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

The app ships as one Docker image holding the single Go binary. Checks (the commands the AI runs on your project) run in short-lived containers that the app starts through the Docker socket, so the compose file mounts it:

```bash
curl -LO https://raw.githubusercontent.com/farismnrr/agentic-ai-workspace/main/compose.yaml
DOCKER_GID=$(stat -c %g /var/run/docker.sock) docker compose up -d
```

By default the app listens on port `8081`; change it with `PORT=9000 docker compose up -d`. Images are published to `ghcr.io/farismnrr/agentic-ai-workspace`: `latest` is the last release from `main`, `dev` follows the `dev` branch, and `sha-<commit>` pins an exact build.

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

`make run` needs a local Docker daemon for checks and listens on port 8081 by default. Use `make run PORT=9000` to change it. `make help` lists the available commands.

The build produces one static Linux binary with no shared-library dependencies, the same file that goes into the image. `make image` builds the Docker image locally under the same name compose pulls, so it is used instead of the published one.

There is intentionally no development server or hot reload. The supported local flow is always build first, then run the resulting binary.

## Quality checks

Run everything with:

```bash
make verify
```

This checks the frontend, Go code, architecture guardrails, lint rules, tests, and production frontend build.

Development happens on `dev`. A fast GitHub Actions workflow checks and builds every push to `dev`. A full workflow, started manually, repeats all checks, merges `dev` into `main`, and publishes the image to GHCR as `latest`. Pushes to `dev` publish the `dev` image once all checks and an image smoke test pass.

## Connecting ChatGPT

The server can act as a connector for AI apps such as ChatGPT. Each connector is a "space" with its own address, for example `https://your-domain.com/api/coding`. Give ChatGPT that address and it finds everything else on its own: how to sign in, which permissions exist, and how to register itself.

Settings (environment variables):

| Variable | Meaning |
| --- | --- |
| `PUBLIC_URL` | The address you open the app on, such as `http://localhost:8081`. Defaults to `http://localhost:<PORT>`. It decides which browser origins may sign in and whether cookies are marked Secure. |
| `CONNECTOR_URL` | The public `https` address AI apps such as ChatGPT use to reach the server, for example `https://workspace.example.com`. This is the address shown under Settings, Spaces & tools, the issuer of the sign-in they are sent through, and the audience of their tokens. Defaults to `PUBLIC_URL`. Point your tunnel or reverse proxy for this address at the server. |
| `ALLOWED_ORIGINS` | Comma-separated browser origins that may sign in and change data (CSRF protection). Defaults to `http://localhost:<PORT>,http://127.0.0.1:<PORT>`. `PUBLIC_URL` is always allowed too. Add a domain or IP here, for example `https://workspace.example.com,http://192.168.1.20:8081`. |
| `CORS_ORIGINS` | Comma-separated origins browser-based AI tools may call the sign-in endpoints from. Defaults to `ALLOWED_ORIGINS`; `*` allows any. |
| `SANDBOX_IMAGE` | The Docker image checks run in (default `node:24-bookworm`, which has Git, Node.js, Python, make and a C compiler). Use any image with the tools your projects need; it is downloaded on first use. |
| `SANDBOX_RUNTIME` | Optional hardened container runtime, such as `runsc` (gVisor), for extra isolation. |
| `SANDBOX_DATA_VOLUME` | Set automatically by `compose.yaml`: the Docker volume that holds `DATA_DIR`, so check containers can mount a person's folder. Leave empty when the app runs directly on the host. |
| `SANDBOX_MEMORY_MB` / `SANDBOX_CPUS` | Memory (default 2048) and CPU cores (default 1) one check may use. |
| `MAX_SANDBOXES` | How many checks may run at the same time across all people (default 4). Extra ones wait up to 30 seconds, then are told the server is busy. |
| `AGENT_WORKERS` | How many tasks the AI runs at the same time (default 2; one project runs one task at a time). |
| `CLIENT_IP_HEADER` | The header that carries the real visitor address when the app sits behind a tunnel or proxy, for example `CF-Connecting-IP`. Used by the sign-in rate limits; without it every visitor behind a tunnel looks like one address. |
| `DATA_DIR` | Where accounts and connections are stored (default `data`). Each person's private coding folder lives inside it, under `workspaces/`. |

### Docker and checks

Checks need access to a Docker daemon (`DOCKER_HOST`, or the default socket). If Docker is not reachable, checks say so plainly and the rest of the app keeps working. Tools that need the internet (`npm install`, `cargo fetch`, `go mod download`) cannot run, because check containers have no network: a project has to bring its dependencies with it.

**Access to the Docker socket is access to the whole machine.** Whoever controls the app can start containers as root on the host. Run Docker in [rootless mode](https://docs.docker.com/engine/security/rootless/) where you can, put the data folder on a filesystem with a size quota (the workspace limit is checked once per second), and keep sign-ups to people you trust. For stronger isolation set `SANDBOX_RUNTIME=runsc` after installing gVisor.

### What the coding space can do

Every person gets their own coding folder (`DATA_DIR/workspaces/<account>`); nobody can read or change anyone else's files, and the folder is deleted with the account. Every check runs in its own throwaway Docker container, removed when the check ends. It sees only your coding folder (as `/workspace`, the one place it can write), a private `/tmp`, and the programs in the sandbox image. It has no internet, cannot see your other files or processes, runs without privileges and with Docker's default system-call filter, and is held to a memory, CPU, process, file-size and workspace-size limit.

Each tool can be switched on or off under Settings, Spaces & tools. Looking around is on from the start; changing files and running tasks stay off until you turn them on.

| Group | Tools |
| --- | --- |
| Look around (on) | list files, find files by name, read a file or some lines of it, search code, list tasks |
| Change files (off) | edit, write, delete and move files |
| Run tasks (off) | run a named task |

Every step an AI app takes is listed under Settings, Activity. Before a file is edited, written over, deleted or moved, its old content is saved, and the Undo button there puts it back. Undo is refused if the file changed again afterwards, so it never overwrites newer work.

Tasks are commands you approve in advance, in `.workspace/tasks.json` inside the workspace folder. The AI app can only ask for a task by name; it cannot see or change this file, and it cannot run anything else.

```json
{
  "tasks": {
    "test": { "description": "Run the tests", "command": ["go", "test", "./..."], "timeout_seconds": 300 },
    "lint": { "description": "Check the code style", "command": ["npm", "run", "lint"] }
  }
}
```

A task runs with the same rights as the app itself, one at a time, with a time limit and only the last 64 KB of its output kept. Because an AI app that can change files could also change a script that a task runs, changing files and running tasks are separate permissions. Turn on only what you need.

### Trying it from your computer

To try it, run the app, expose it with a tunnel (for example `cloudflared tunnel --url http://localhost:8081`), and start the app with `CONNECTOR_URL` set to the tunnel address. In ChatGPT, turn on developer mode, add a connector with `<CONNECTOR_URL>/api/coding`, and sign in when asked. You can see and remove connected apps under Settings, Connected apps. Each tool has a switch under Settings, Spaces & tools; looking at files is on from the start and changing files or running checks stay off until you turn them on. The coding space never shows `.env`, `.git`, or key files.

## Design system

The UI uses the Masih Awam visual language through semantic daisyUI themes instead of copying the LMS implementation. The app ships a single light theme that defines the roles for actions, surfaces, and statuses.

See [docs/design-system.md](./docs/design-system.md) for the color roles, interaction states, and component rules.

## Pages

The app is a single-page Svelte app served by the Go binary: `/` is the landing page, `/login` and `/register` are the sign-in and create-account screens, and `/app` is the workspace: home, an inbox of things that need you, projects with their tasks, a task page (conversation plus plan, proof, and changes), and settings (models, persona and rules, connectors, usage, account). Press Ctrl/Cmd+K anywhere in the workspace to search and jump. Everything here is real except notifications and data export, which are marked as coming soon.

## Working with the AI team

1. **Add a model.** Under Settings, AI models, add Anthropic, OpenAI, Google Gemini (key from [Google AI Studio](https://aistudio.google.com/apikey)), or an OpenAI-compatible gateway such as [9router](http://localhost:20128/v1) (use its address and key). Hosted providers use their fixed official address, and a gateway must run on the same computer as this app (`localhost`); the app refuses to connect to any other address, so nobody can point the server at your internal network. The key is stored encrypted and checked right away.
2. **Create a project.** Each project gets its own folder under `DATA_DIR/projects`; the AI can only touch files inside it.
3. **Describe what you need** on Home. The AI makes a plan, works step by step, and shows every step. Deleting or moving files and running checks need your OK first; choose "Ask me before every change" to be asked about edits too.
4. **Review.** When it finishes you see what changed and what was checked. The proof says "Proven to work" only when the app itself ran a check that passed after the last change, and every change can be undone.

To let the app verify the AI's work, add **checks** on the project's Checks tab: named commands such as `go test ./...`. The AI can only run the checks you list, asks before running one, and "Proven to work" appears only when a check passed after its last change. Checks run on this computer with the app's rights, so add only commands you trust.

## Your data

Under Settings, Account you can change your password (other devices are signed out), download everything as a zip (settings, projects with their files, tasks with every step, usage; saved AI keys are never included), delete all tasks, or delete the account with its projects and files after confirming your password. Settings, Notifications can alert you in this browser, while the tab is in the background, when a task needs you or fails.

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
