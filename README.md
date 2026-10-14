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

The GitHub release is a rolling build named **Latest**. When a full release workflow succeeds, the Linux binary in that release is replaced.

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

`make run` listens on port 8080 by default. Use `make run PORT=9000` to change it. `make help` lists the available commands.

The build produces one static Linux binary with no shared-library dependencies, the same file that is published to the release.

There is intentionally no development server or hot reload. The supported local flow is always build first, then run the resulting binary.

## Quality checks

Run everything with:

```bash
make verify
```

This checks the frontend, Go code, architecture guardrails, lint rules, tests, and production frontend build.

Development happens on `dev`. A fast GitHub Actions workflow checks and builds every push to `dev`. A full workflow, started manually, repeats all checks, merges `dev` into `main`, and publishes the Linux release.

## Connecting ChatGPT

The server can act as a connector for AI apps such as ChatGPT. Each connector is a "space" with its own address, for example `https://your-domain.com/api/coding`. Give ChatGPT that address and it finds everything else on its own: how to sign in, which permissions exist, and how to register itself.

Settings (environment variables):

| Variable | Meaning |
| --- | --- |
| `PUBLIC_URL` | The public address of the app, such as `https://your-domain.com`. Defaults to `http://localhost:<PORT>`. ChatGPT needs a public `https` address. |
| `ALLOWED_ORIGINS` | Comma-separated browser origins that may sign in and change data (CSRF protection). Defaults to `http://localhost:<PORT>,http://127.0.0.1:<PORT>`. `PUBLIC_URL` is always allowed too. Add a domain or IP here, for example `https://workspace.example.com,http://192.168.1.20:8081`. |
| `CORS_ORIGINS` | Comma-separated origins browser-based AI tools may call the sign-in endpoints from. Defaults to `ALLOWED_ORIGINS`; `*` allows any. |
| `AGENT_WORKERS` | How many tasks the AI runs at the same time (default 2; one project runs one task at a time). |
| `CLIENT_IP_HEADER` | The header that carries the real visitor address when the app sits behind a tunnel or proxy, for example `CF-Connecting-IP`. Used by the sign-in rate limits; without it every visitor behind a tunnel looks like one address. |
| `SMTP_ADDR`, `SMTP_USER`, `SMTP_PASSWORD`, `SMTP_FROM` | A mail server (`host:port`) used to send password reset links. Without `SMTP_ADDR` the link is written to the server log instead, so the owner can pass it on. |
| `DATA_DIR` | Where accounts and connections are stored (default `data`). Keep it outside the coding folder. |
| `CODING_WORKSPACE_ROOT` | A folder the coding space may read. The coding space is off when this is empty. |

### What the coding space can do

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

To try it, run the app, expose it with a tunnel (for example `cloudflared tunnel --url http://localhost:8081`), and start the app with `PUBLIC_URL` set to the tunnel address. In ChatGPT, turn on developer mode, add a connector with `<PUBLIC_URL>/api/coding`, and sign in when asked. You can see and remove connected apps under Settings, Connected apps. The coding space can only read files; it never changes them and never shows `.env`, `.git`, or key files.

## Design system

The UI uses the Masih Awam visual language through semantic daisyUI themes instead of copying the LMS implementation. The app ships a single light theme that defines the roles for actions, surfaces, and statuses.

See [docs/design-system.md](./docs/design-system.md) for the color roles, interaction states, and component rules.

## Pages

The app is a single-page Svelte app served by the Go binary: `/` is the landing page, `/login` and `/register` are the sign-in and create-account screens, and `/app` is the workspace: home, an inbox of things that need you, projects with their tasks, a task page (conversation plus plan, proof, and changes), and settings (models, persona and rules, connectors, usage, account). Press Ctrl/Cmd+K anywhere in the workspace to search and jump. Everything here is real except notifications and data export, which are marked as coming soon.

## Working with the AI team

1. **Add a model.** Under Settings, AI models, add Anthropic, OpenAI, or any OpenAI-compatible gateway such as [9router](http://localhost:20128/v1) (use its address and key). The key is stored encrypted and checked right away.
2. **Create a project.** Each project gets its own folder under `DATA_DIR/projects`; the AI can only touch files inside it.
3. **Describe what you need** on Home. The AI makes a plan, works step by step, and shows every step. Deleting or moving files and running checks need your OK first; choose "Ask me before every change" to be asked about edits too.
4. **Review.** When it finishes you see what changed and what was checked. The proof says "Proven to work" only when the app itself ran a check that passed after the last change, and every change can be undone.

To let the app verify the AI's work, add **checks** on the project's Checks tab: named commands such as `go test ./...`. The AI can only run the checks you list, asks before running one, and "Proven to work" appears only when a check passed after its last change. Checks run on this computer with the app's rights, so add only commands you trust.

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
