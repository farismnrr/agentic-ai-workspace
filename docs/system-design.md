# System design

How the app is put together, in short. The goal: easy to run, easy to grow, never one tangled codebase.

## At a glance

| Part | What it is |
| --- | --- |
| Program | One Go binary, one process, one port |
| Web app | Svelte, built once and embedded in the binary |
| Data | SQLite under `DATA_DIR` |
| AI work | Short-lived Docker containers, started through the Docker socket |
| AI models | Called over HTTPS with the person's own keys |

## How it runs

```mermaid
flowchart LR
  B[Browser] -->|HTTP| G["Go binary (Fiber)"]
  G --> A[Handles API requests]
  G --> S[Serves the built Svelte app]
```

- **One executable** to deploy and one process to monitor.
- **One public port** to configure.
- **No Node.js** on the server: the frontend is compiled during the build.

## How a build works

```mermaid
flowchart LR
  A["Svelte + TypeScript"] --> B[Type check and lint] --> C[Vite build]
  C --> D[Files embedded into Go] --> E[Go tests and lint] --> F[bin/agentic-ai-workspace]
```

Run `make build`. It verifies first, then builds the binary.

## Code layout

```text
cmd/workspace/            program entry point, stays small
internal/bootstrap/       connects the pieces
internal/modules/<feature>/
  domain/                 business rules and types
  application/            use cases and the interfaces they need
  adapter/                HTTP, database and other implementations
internal/platform/        config, HTTP server, embedded web app
```

- **Dependencies point inward:** `adapter -> application -> domain`.
- **A domain rule never knows** which HTTP framework or database is used. That keeps it easy to test and to replace.
- **Platform code** holds technical details only, never business rules.
- **A feature creates only the layers it needs.** Empty folders do not help.

### Frontend

```text
web/src/
  app/                    shell, router
  features/<feature>/     one feature's screens and logic
  shared/                 UI and logic used by two or more features
```

- Strict TypeScript everywhere.
- daisyUI is the component layer. Tailwind covers layout and small tweaks.
- Features never import other features. Shared code goes in `shared/`.

## Accounts and connectors

The binary hosts its own sign-in and an OAuth 2.1 server, so AI apps can connect without any extra service.

- **Accounts:** email and password with a cookie session. Data lives in SQLite, using a pure-Go driver so the binary stays static.
- **Spaces:** each space is an endpoint an AI app can call. The first, `coding`, lets it read, search, change and run things in that app's own project.
- **Isolation:** every call resolves to one person's project. Nobody can reach another person's files, and two apps never share a folder.
- **Tokens:** stored hashed and tied to one space. A replayed code or refresh token revokes the whole connection.

Each tool has a switch the person controls in Settings. A call must pass the switch, then the token's scope, and is always logged. Edits save the old content first, which is what Undo restores.

## The sandbox

Tasks, commands and web apps run in one throwaway Docker container per run.

| Limit | How |
| --- | --- |
| Files | Only that person's project folder is visible |
| Network | No network. Package registries only, through a proxy |
| System | Read-only root, no extra privileges, memory, CPU and process limits |
| Capacity | `MAX_SANDBOXES` runs at once. A run stops if its folder passes 2 GiB |

- **Toolchains** (Python, Node.js, Go, Rust, Java and more) live in the image, so nothing downloads per person.
- **Browser:** a headless Chromium in the image takes screenshots of a running site. It runs in its own container with no network.
- **Docker down?** Checks fail with a plain reason and nothing else breaks.
- **Not covered:** hard disk quotas, and the host kernel and Docker daemon that every container shares.
- **Docker socket:** whoever controls it controls the host. Docker rootless mode is the recommended mitigation.

Web apps the AI starts use the same sandbox. See [sites.md](./sites.md).

## The agent

- **One thread per message.** There is no separate chat. The AI decides whether to answer or to plan and work.
- **Approvals.** Reading is automatic. Changing files or running commands waits for your yes, unless you turned on Skip approvals.
- **Switches win.** A tool that is off is never offered to the model and never run.
- **Pause and resume.** The conversation is stored, so a task can wait for you and continue later.
- **Project documents.** Task plans, requirements and research are exported into `project-notes/`. New tasks get an index reference; documents are loaded through existing file tools.
- **Long input.** Messages over 12,000 characters become an exact text document, a manifest and readable parts. The thread holds a preview and download link. Request-body limits still apply.
- **Proof.** Built only from checks the app ran itself after the last change. Claims the AI makes alone are marked "claimed".
- **Checks.** A project's checks live in `.workspace/tasks.json`. The AI's tools can never read or write that file.
- **GitHub.** Each person connects their own account. Commands get only their token. Web apps never do.

## Your own AI plan

- **How:** the provider's own program (Claude Code or Codex) runs in the sandbox image, once per turn, in the project's folder. The app reads what it prints and records the steps.
- **Sign-in:** the program signs in on the provider's page. The app never reads or stores the token. It only keeps the program's folder per person and removes it with the account.
- **Network:** the program reaches only its provider's hosts, through the same per-person proxy socket.
- **Proof:** commands the program ran count like the app's own, so "Proven to work" still comes from the app.

## Models

- **Providers:** Anthropic, OpenAI, Gemini, or an OpenAI-compatible gateway on the same computer, such as 9router.
- **Keys:** stored encrypted. Hosted providers can reach only public addresses. Gateways only this computer.
- **Gateways:** the model list is re-read on its own, at most every 30 seconds, so a provider added behind it appears without a click.
- **Model and effort:** app tasks resolve current orchestrator settings at each run. Assigned teammate jobs retain their selection on resume. Explicit model or effort choices from the API stay pinned.
- **Provider changes:** a different vendor program starts a fresh session with task context. A gateway can say which effort levels each model takes.
- **Context window:** the size you chose, else the size the provider reports, else a default. Near the limit the AI summarizes its notes and keeps going.
- **Limits:** 40 model turns and 15 minutes per run. One task at a time per project. Interrupted tasks can be continued.

## Data and updates

- **Export and delete:** your data downloads as one zip. Deleting a task list or the account removes everything tied to it.
- **Live updates:** pages follow changes over server-sent events and fall back to a slow poll.
- **Rate limits:** sign-in, sign-up and OAuth endpoints are limited per visitor. Wrong guesses never lock you out.
- **Database changes:** only through numbered migration files. Applied files are checksummed and never edited.

## Quality and release

`make verify` runs every check:

1. Repository guardrails.
2. Svelte and TypeScript checks.
3. Frontend lint and production build.
4. Go module check and tests.
5. Strict Go lint and architecture rules.

`make e2e-sites` also tests a private site in a real browser. It needs Docker and Playwright and is not part of `make verify`.

Rules in the code: source files stay under 200 lines (documentation is exempt), and functions have size and complexity limits.

Work happens on `dev`. Two GitHub Actions workflows run the same checks:

| Workflow | When | What it does |
| --- | --- | --- |
| **Fast** | Every push to `dev` | Verifies, builds the binary and image, smoke-tests both, publishes the `dev` image |
| **Full** | Started by hand from `dev` | Repeats everything from a clean checkout, merges `dev` into `main`, publishes `latest`, then deploys it to production |

For the Jenkins pipeline and hotfix deploys, see [devops.md](./devops.md).

## Deployment

- Run the image `ghcr.io/farismnrr/agentic-ai-workspace` with `compose.latest.yaml` (or `compose.dev.yaml`).
- The Docker socket is mounted so checks can run in containers.
- There is no Node.js runtime, frontend server or extra port to deploy.

Steps and settings: [deployment.md](./deployment.md).
