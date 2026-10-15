# Deployment

How to run, configure and release the app. For what the app does, see [using-the-app.md](./using-the-app.md); for how it is built, see [system-design.md](./system-design.md).

## Pick how you want to run it

| I want to... | Choose | Needs |
| --- | --- | --- |
| Just use it, as fast as possible | **A. Pull the ready-made image** | Docker |
| Run my own changes, still in Docker | **B. Build the image yourself** | Docker, Git |
| Run it without Docker for the app itself | **C. Build the program and run it** | Go, Node.js, Docker |

Every option needs Docker, because the commands the AI runs on your project (checks) run in short-lived Docker containers. If Docker is not reachable, the rest of the app still works and checks say so plainly.

The app opens on `http://localhost:8081` in all three. Go slowly: each option below is a few small steps, and you can stop after any step.

### Before you start (all options)

1. Check Docker works: `docker version` should print a Client and a Server section. If it says "permission denied", add yourself to the `docker` group or use rootless Docker.
2. Check Compose works: `docker compose version`.
3. Check port 8081 is free (`ss -ltn | grep 8081` prints nothing). If it is taken, pick another one later with `PORT=9000`.

### A. Pull the ready-made image (easiest)

**Guided (recommended):** one script asks for each setting one by one (with an example and a default; press Enter to keep it), saves them in `.env`, and only then downloads the compose file and starts the app:

```bash
curl -fsSL https://raw.githubusercontent.com/farismnrr/agentic-ai-workspace/main/scripts/install.sh | sh
```

It creates the folder `~/agentic-ai-workspace/` (your home folder, whatever directory you run it from) waits until the app answers (up to 60 seconds, showing the logs if it does not), lists both containers, and tells you how to read logs and stop the app. The `sandbox` container showing `Exited (0)` is normal: it only downloads the toolbox image and stops. Use `CHANNEL=dev` before `sh` for the dev image, or `INSTALL_DIR=/opt/workspace` to choose the folder. Running it again is safe: it asks before replacing an existing `.env` (the old one is kept as `.env.bak`).

**By hand:** the same thing in four steps.

```bash
# 1. Get the compose file
curl -LO https://raw.githubusercontent.com/farismnrr/agentic-ai-workspace/main/compose.latest.yaml

# 2. Tell the app which group owns the Docker socket (it needs this to start checks)
export DOCKER_GID=$(stat -c %g /var/run/docker.sock)

# 3. Download the images and start the app in the background
docker compose -f compose.latest.yaml up -d

# 4. Look at it
docker compose -f compose.latest.yaml ps
docker compose -f compose.latest.yaml logs -f app
```

The first start downloads two images (the app and the toolbox the AI's commands run in), so it can take a few minutes. Then open `http://localhost:8081` and create your account.

Two compose files exist: `compose.latest.yaml` runs the image published from `main` (the stable one), and `compose.dev.yaml` runs the one published from `dev` (newest, less tested). Use one at a time, since they share a port and a data volume. Images live at `ghcr.io/farismnrr/agentic-ai-workspace`: `latest` is the last release, `dev` follows the `dev` branch, and `sha-<commit>` pins an exact build.

### B. Build the image yourself

Use this when you changed the code, or want to run exactly what is in your checkout.

```bash
# 1. Get the code
git clone https://github.com/farismnrr/agentic-ai-workspace.git
cd agentic-ai-workspace

# 2. Build the app image (tagged latest and dev, the names the compose files use)
make image

# 3. Build the toolbox image the AI's commands run in
make sandbox-image

# 4. Start, using only the images you just built (never download)
export DOCKER_GID=$(stat -c %g /var/run/docker.sock)
docker compose -f compose.latest.yaml up -d --pull never
```

`--pull never` is what keeps your local build in use. Do not run `docker compose pull` here: it would replace your build with the published image.

### C. Build the program and run it

No Docker image for the app: one static binary runs on your computer. Docker is still needed for checks.

```bash
make run                 # verifies, builds, then starts on port 8081
make run PORT=9000       # same, on another port
```

Or in two steps: `make build`, then `./bin/agentic-ai-workspace`. `make help` lists every command. There is intentionally no development server or hot reload: always build first, then run.

## Everyday tasks (options A and B)

Replace the file name with `compose.dev.yaml` if you use the dev channel.

| I want to... | Command |
| --- | --- |
| See if it is running and healthy | `docker compose -f compose.latest.yaml ps` (the app shows `Up ... (healthy)` about 10 seconds after it starts; `sandbox` shows `Exited (0)`, which is normal) |
| Read the logs | `docker compose -f compose.latest.yaml logs -f app` |
| Stop it (data stays) | `docker compose -f compose.latest.yaml down` |
| Start it again | `docker compose -f compose.latest.yaml up -d` |
| Update to the newest published image (A) | `docker compose -f compose.latest.yaml pull` then `docker compose -f compose.latest.yaml up -d` |
| Update after rebuilding yourself (B) | `make image` then `docker compose -f compose.latest.yaml up -d --pull never --force-recreate` |
| Use another port | `PORT=9000 docker compose -f compose.latest.yaml up -d` |
| Delete everything, including accounts and projects | `docker compose -f compose.latest.yaml down -v` (cannot be undone) |

Your accounts, projects and settings live in the Docker volume `agentic-data`. Stopping, updating and recreating the app keeps it; only `down -v` or `docker volume rm agentic-data` removes it. To back it up while the app is stopped:

```bash
docker run --rm -v agentic-data:/data -v "$PWD":/backup alpine tar czf /backup/agentic-data.tar.gz -C /data .
```

## If something goes wrong

| What you see | What to do |
| --- | --- |
| `DOCKER_GID` is required | Run `export DOCKER_GID=$(stat -c %g /var/run/docker.sock)` in the same terminal, or put `DOCKER_GID=...` in `.env`. |
| Port already in use | Set another `PORT` (and `PUBLIC_URL` to match). |
| The app says Docker is not reachable | Check `docker ps` works and the socket path is `/var/run/docker.sock`. |
| Sign-in is refused from a domain or IP | Add it to `ALLOWED_ORIGINS` or set `PUBLIC_URL` (see below), then recreate the app. |
| Image not found with `--pull never` | Run `make image` and `make sandbox-image` first. |
| Still stuck | Read `docker compose -f compose.latest.yaml logs app`. |

## Configuration

Settings are environment variables, set in the shell or in a `.env` file next to the compose file (for example `PUBLIC_URL=https://workspace.example.com`). You usually need none of them for a local try-out. After changing one, run the `up -d` command again so the app restarts with it. Lines starting with `#` in `.env` are ignored.

| Variable | Meaning |
| --- | --- |
| `PUBLIC_URL` | The address you open the app on, such as `http://localhost:8081`. Defaults to `http://localhost:<PORT>`. It decides which browser origins may sign in and whether cookies are marked Secure. |
| `CONNECTOR_URL` | The public `https` address AI apps such as ChatGPT use to reach the server, for example `https://workspace.example.com`. This is the address shown under Settings, Spaces & tools, the issuer of the sign-in they are sent through, and the audience of their tokens. Defaults to `PUBLIC_URL`. Point your tunnel or reverse proxy for this address at the server. |
| `ALLOWED_ORIGINS` | Comma-separated browser origins that may sign in and change data (CSRF protection). Defaults to `http://localhost:<PORT>,http://127.0.0.1:<PORT>`. `PUBLIC_URL` is always allowed too. Add a domain or IP here, for example `https://workspace.example.com,http://192.168.1.20:8081`. |
| `CORS_ORIGINS` | Comma-separated origins browser-based AI tools may call the sign-in endpoints from. Defaults to `ALLOWED_ORIGINS`; `*` allows any. |
| `SANDBOX_IMAGE` | The Docker image checks and commands run in (default `ghcr.io/<owner>/agentic-ai-workspace-sandbox`, which has Python, Node.js, Go, Rust, Java, Maven, Git, the GitHub CLI (`gh`), Bash and a C compiler). `make sandbox-image` builds it locally. Any image works, but only the default one gets the registry proxy and the toolchain list. The compose files pull it together with the app (a `sandbox` service that exits at once), so it is ready before the app starts. |
| `SANDBOX_EGRESS_HOSTS` | Comma-separated hostnames commands may reach over HTTPS (default: PyPI, npm, the Go module proxy, crates.io and Maven Central). Everything else, including your computer and network, stays unreachable. |
| `GITHUB_CLIENT_ID` / `GITHUB_CLIENT_SECRET` / `GITHUB_APP_SLUG` | The GitHub App people sign in with under Settings, Integrations (see [GitHub](#github) below). All three empty means the feature is off. |
| `SANDBOX_RUNTIME` | Optional hardened container runtime, such as `runsc` (gVisor), for extra isolation. |
| `SANDBOX_DATA_VOLUME` | Set automatically by the compose files: the Docker volume that holds `DATA_DIR`, so check containers can mount a person's folder. Leave empty when the app runs directly on the host. |
| `SANDBOX_MEMORY_MB` / `SANDBOX_CPUS` | Memory (default 2048) and CPU cores (default 1) one check may use. |
| `SITE_MAX_PER_USER` / `SITE_MAX_TOTAL` | How many web apps (see [sites.md](./sites.md)) one person (default 2) and the whole server (default 6) may keep running. Apps run until stopped. |
| `MAX_SANDBOXES` | How many checks may run at the same time across all people (default 4). Extra ones wait up to 30 seconds, then are told the server is busy. |
| `AGENT_WORKERS` | How many tasks the AI runs at the same time (default 2; one project runs one task at a time). |
| `CLIENT_IP_HEADER` | The header that carries the real visitor address when the app sits behind a tunnel or proxy, for example `CF-Connecting-IP`. Used by the sign-in rate limits and to tie private site tickets to the visitor's address; without it every visitor behind a tunnel looks like one address. |
| `DATA_DIR` | Where accounts and connections are stored (default `data`). Projects, including the ones connected AI apps work in, live inside it, under `projects/`. |

## GitHub

Settings, Integrations lets each person connect their own GitHub account so the AI can clone, commit and push for them. The server needs one **GitHub App** (not an OAuth App) to do that. Create it once, at GitHub under Settings, Developer settings, GitHub Apps, New GitHub App:

1. **Callback URL:** `<PUBLIC_URL>/api/integrations/github/callback`. Turn on **Request user authorization (OAuth) during installation**, so installing and signing in is one step. Leave **Expire user authorization tokens** on.
2. **Webhook:** switch it off.
3. **Repository permissions:** Contents and Pull requests set to read and write, Metadata read-only. Add Workflows only if the AI should be able to change `.github/workflows` files. Ask for nothing else.
4. **Where can this app be installed:** any account if several people use the server.
5. After creating it, generate a **client secret**. Set `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET` and `GITHUB_APP_SLUG` (the last part of the app's page address) and recreate the app.

Each person then presses **Connect GitHub**, approves, and chooses on GitHub which repositories the app may reach. What a person's commands can do on GitHub is the overlap of the app's permissions, their own rights and the repositories they chose. Their token is stored encrypted with `DATA_DIR/secret.key` and renewed automatically (it lasts 8 hours). It is given only to commands (`run_command` and tasks) in that person's own projects, together with access to `github.com`, `api.github.com` and `codeload.github.com` from those projects alone; web apps a project serves never get it. Be aware that any program a command runs, such as an install script, can read the token while the command runs; it can only be sent to GitHub, and it works only for the repositories the person chose. Disconnecting revokes the grant at GitHub.

## Docker and the sandbox

Checks need access to a Docker daemon (`DOCKER_HOST`, or the default socket). If Docker is not reachable, checks say so plainly and the rest of the app keeps working. Commands have no network of their own. They reach only the package registries in `SANDBOX_EGRESS_HOSTS` (plus GitHub, for people who connected it), through a proxy inside the app, so `npm install`, `pip install` (or `uv pip install`), `cargo fetch`, `go mod download` and `mvn` work while other websites do not. Downloads are cached per person (up to 5 GB, emptied when over) in `DATA_DIR/caches`.

**Access to the Docker socket is access to the whole machine.** Whoever controls the app can start containers as root on the host. Run Docker in [rootless mode](https://docs.docker.com/engine/security/rootless/) where you can, put the data folder on a filesystem with a size quota (the workspace limit is checked once per second), and keep sign-ups to people you trust. For stronger isolation set `SANDBOX_RUNTIME=runsc` after installing gVisor.

## Quality checks and releases

Run everything with:

```bash
make verify
```

This checks the frontend, Go code, architecture guardrails, lint rules, tests, and production frontend build.

Development happens on `dev`. A fast GitHub Actions workflow checks and builds every push to `dev`. A full workflow, started manually, repeats all checks, merges `dev` into `main`, and publishes the image to GHCR as `latest`. Pushes to `dev` publish the `dev` image once all checks and an image smoke test pass.

The `Jenkinsfile` is a Jenkins alternative to the full workflow. It builds `main` as it is (no `dev` to `main` merge), runs the same checks and smoke tests, and pushes the app and sandbox images to GHCR as `latest` and `sha-<commit>`. The agent needs only Git and Docker (Node.js and Go run in containers that mount the workspace, so the daemon must share `/var/jenkins_home`), plus a `ghcr-credentials` username/password credential whose password is a token with `write:packages`.

`make e2e-sites` drives a private site with a real signed-out browser (it needs Docker and Playwright: `npm i -g playwright && npx playwright install chromium`). It is not part of `make verify`.
