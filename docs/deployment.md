# Deployment

How to run, configure and release the app. For what the app does, see [using-the-app.md](./using-the-app.md); for how it is built, see [system-design.md](./system-design.md).

## Run it with Docker Compose

The app ships as one Docker image holding the single Go binary. Checks (the commands the AI runs on your project) run in short-lived containers that the app starts through the Docker socket, so the compose file mounts it:

```bash
curl -LO https://raw.githubusercontent.com/farismnrr/agentic-ai-workspace/main/compose.latest.yaml
DOCKER_GID=$(stat -c %g /var/run/docker.sock) docker compose -f compose.latest.yaml up -d
```

Two compose files exist: `compose.latest.yaml` runs the image published from `main`, and `compose.dev.yaml` runs the one published from `dev`. Use one at a time, since they share a port and a data volume.

By default the app listens on port `8081`; change it with `PORT=9000 docker compose -f compose.latest.yaml up -d`. Images are published to `ghcr.io/farismnrr/agentic-ai-workspace`: `latest` is the last release from `main`, `dev` follows the `dev` branch, and `sha-<commit>` pins an exact build.

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

The build produces one static Linux binary with no shared-library dependencies, the same file that goes into the image. `make image` builds the Docker image locally under the names the compose files pull (`latest` and `dev`), so it is used instead of the published one.

There is intentionally no development server or hot reload. The supported local flow is always build first, then run the resulting binary.

## Configuration

Settings are environment variables, set in the shell or in a `.env` file next to the compose file:

| Variable | Meaning |
| --- | --- |
| `PUBLIC_URL` | The address you open the app on, such as `http://localhost:8081`. Defaults to `http://localhost:<PORT>`. It decides which browser origins may sign in and whether cookies are marked Secure. |
| `CONNECTOR_URL` | The public `https` address AI apps such as ChatGPT use to reach the server, for example `https://workspace.example.com`. This is the address shown under Settings, Spaces & tools, the issuer of the sign-in they are sent through, and the audience of their tokens. Defaults to `PUBLIC_URL`. Point your tunnel or reverse proxy for this address at the server. |
| `ALLOWED_ORIGINS` | Comma-separated browser origins that may sign in and change data (CSRF protection). Defaults to `http://localhost:<PORT>,http://127.0.0.1:<PORT>`. `PUBLIC_URL` is always allowed too. Add a domain or IP here, for example `https://workspace.example.com,http://192.168.1.20:8081`. |
| `CORS_ORIGINS` | Comma-separated origins browser-based AI tools may call the sign-in endpoints from. Defaults to `ALLOWED_ORIGINS`; `*` allows any. |
| `SANDBOX_IMAGE` | The Docker image checks and commands run in (default `ghcr.io/<owner>/agentic-ai-workspace-sandbox`, which has Python, Node.js, Go, Rust, Java, Maven, Git, Bash and a C compiler). `make sandbox-image` builds it locally. Any image works, but only the default one gets the registry proxy and the toolchain list. The compose files pull it together with the app (a `sandbox` service that exits at once), so it is ready before the app starts. |
| `SANDBOX_EGRESS_HOSTS` | Comma-separated hostnames commands may reach over HTTPS (default: PyPI, npm, the Go module proxy, crates.io and Maven Central). Everything else, including your computer and network, stays unreachable. |
| `SANDBOX_RUNTIME` | Optional hardened container runtime, such as `runsc` (gVisor), for extra isolation. |
| `SANDBOX_DATA_VOLUME` | Set automatically by the compose files: the Docker volume that holds `DATA_DIR`, so check containers can mount a person's folder. Leave empty when the app runs directly on the host. |
| `SANDBOX_MEMORY_MB` / `SANDBOX_CPUS` | Memory (default 2048) and CPU cores (default 1) one check may use. |
| `SITE_MAX_PER_USER` / `SITE_MAX_TOTAL` | How many web apps (see [sites.md](./sites.md)) one person (default 2) and the whole server (default 6) may keep running. Apps run until stopped. |
| `MAX_SANDBOXES` | How many checks may run at the same time across all people (default 4). Extra ones wait up to 30 seconds, then are told the server is busy. |
| `AGENT_WORKERS` | How many tasks the AI runs at the same time (default 2; one project runs one task at a time). |
| `CLIENT_IP_HEADER` | The header that carries the real visitor address when the app sits behind a tunnel or proxy, for example `CF-Connecting-IP`. Used by the sign-in rate limits and to tie private site tickets to the visitor's address; without it every visitor behind a tunnel looks like one address. |
| `DATA_DIR` | Where accounts and connections are stored (default `data`). Projects, including the ones connected AI apps work in, live inside it, under `projects/`. |

## Docker and the sandbox

Checks need access to a Docker daemon (`DOCKER_HOST`, or the default socket). If Docker is not reachable, checks say so plainly and the rest of the app keeps working. Commands have no network of their own. They reach only the package registries in `SANDBOX_EGRESS_HOSTS`, through a proxy inside the app, so `npm install`, `pip install` (or `uv pip install`), `cargo fetch`, `go mod download` and `mvn` work while other websites do not. Downloads are cached per person (up to 5 GB, emptied when over) in `DATA_DIR/caches`.

**Access to the Docker socket is access to the whole machine.** Whoever controls the app can start containers as root on the host. Run Docker in [rootless mode](https://docs.docker.com/engine/security/rootless/) where you can, put the data folder on a filesystem with a size quota (the workspace limit is checked once per second), and keep sign-ups to people you trust. For stronger isolation set `SANDBOX_RUNTIME=runsc` after installing gVisor.

## Quality checks and releases

Run everything with:

```bash
make verify
```

This checks the frontend, Go code, architecture guardrails, lint rules, tests, and production frontend build.

Development happens on `dev`. A fast GitHub Actions workflow checks and builds every push to `dev`. A full workflow, started manually, repeats all checks, merges `dev` into `main`, and publishes the image to GHCR as `latest`. Pushes to `dev` publish the `dev` image once all checks and an image smoke test pass.

`make e2e-sites` drives a private site with a real signed-out browser (it needs Docker and Playwright: `npm i -g playwright && npx playwright install chromium`). It is not part of `make verify`.
