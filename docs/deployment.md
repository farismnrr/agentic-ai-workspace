# Deployment

Install, update, back up and configure the app. To learn what it does, read [using-the-app.md](./using-the-app.md).

## Choose a way to run it

| You want to | Use | You need |
| --- | --- | --- |
| Start fast | A. Pull the image | Docker |
| Run your own changes in Docker | B. Build the image | Docker, Git |
| Run the binary directly | C. Build and run | Go, Node.js, Docker |

All three need Docker, because the AI's commands run in short-lived containers. Without Docker the app still works, but checks can't run. The app opens on `http://localhost:8081`.

Before you start:

```bash
docker version          # shows Client and Server
docker compose version
ss -ltn | grep 8081     # prints nothing if the port is free
```

If `docker version` says "permission denied", add yourself to the `docker` group or use rootless Docker.

### A. Pull the image

The installer asks for each setting, saves them in `.env`, and starts the app:

```bash
curl -fsSL https://raw.githubusercontent.com/farismnrr/agentic-ai-workspace/main/scripts/install.sh | sh
```

- It installs to `~/agentic-ai-workspace/`. Set `INSTALL_DIR=/opt/workspace` to change that.
- Set `CHANNEL=dev` to get the dev image.
- It's safe to run again. Your old `.env` is kept as `.env.bak`.

To do the same by hand:

```bash
curl -LO https://raw.githubusercontent.com/farismnrr/agentic-ai-workspace/main/compose.latest.yaml
export DOCKER_GID=$(stat -c %g /var/run/docker.sock)   # lets the app start checks
docker compose -f compose.latest.yaml up -d
docker compose -f compose.latest.yaml logs -f app
```

The first start downloads two images, so it can take a few minutes. The `sandbox` container shows `Exited (0)`. That is normal: it only downloads the toolbox and stops.

| Compose file | Image | Use for |
| --- | --- | --- |
| `compose.latest.yaml` | `latest`, from `main` | Stable |
| `compose.dev.yaml` | `dev`, from `dev` | Newest, less tested |
| `compose.prod.yaml` | `sha-<commit>`, from the Full workflow | Masih Awam production (port 5004) |

Run only one of `latest` and `dev` at a time, because they share a port and a data volume. `compose.prod.yaml` has its own port, volume, and project name. You can pin an exact build with `sha-<commit>`.

### B. Build the image

```bash
git clone https://github.com/farismnrr/agentic-ai-workspace.git
cd agentic-ai-workspace
make image            # the app
make sandbox-image    # the toolbox the AI's commands run in
export DOCKER_GID=$(stat -c %g /var/run/docker.sock)
docker compose -f compose.latest.yaml up -d --pull never
```

`--pull never` keeps your local build. Don't run `docker compose pull` here, or it replaces your build with the published one.

### C. Build and run the binary

```bash
make run              # verify, build, start on port 8081
make run PORT=9000    # another port
```

You can also run `make build` and then `./bin/agentic-ai-workspace`. There is no dev server or hot reload, so always build first. `make help` lists every command.

## Everyday commands

The examples use `compose.latest.yaml`. On the dev channel, use `compose.dev.yaml` instead.

### Check and read

```bash
docker compose -f compose.latest.yaml ps          # healthy after about 10 seconds
docker compose -f compose.latest.yaml logs -f app
```

### Stop and start

```bash
docker compose -f compose.latest.yaml down        # keeps your data
docker compose -f compose.latest.yaml up -d
```

### Update

If you pulled the image (A):

```bash
docker compose -f compose.latest.yaml pull
docker compose -f compose.latest.yaml up -d
```

If you built it yourself (B):

```bash
make image
docker compose -f compose.latest.yaml up -d --pull never --force-recreate
```

### Change the port

```bash
PORT=9000 docker compose -f compose.latest.yaml up -d
```

### Delete everything

This cannot be undone.

```bash
docker compose -f compose.latest.yaml down -v
```

### Back up your data

Your data lives in the Docker volume `agentic-data`. Only `down -v` or `docker volume rm` deletes it. Back it up while the app is stopped:

```bash
docker run --rm -v agentic-data:/data -v "$PWD":/backup alpine tar czf /backup/agentic-data.tar.gz -C /data .
```

## Troubleshooting

| Problem | Fix |
| --- | --- |
| `DOCKER_GID` is required | `export DOCKER_GID=$(stat -c %g /var/run/docker.sock)`, or add it to `.env` |
| Port already in use | Set another `PORT`, and update `PUBLIC_URL` to match |
| Docker is not reachable | Make sure `docker ps` works and the socket is at `/var/run/docker.sock` |
| Sign-in refused from a domain or IP | Add it to `ALLOWED_ORIGINS` or set `PUBLIC_URL`, then recreate the app |
| Image not found with `--pull never` | Run `make image` and `make sandbox-image` first |
| Something else | Read `docker compose -f compose.latest.yaml logs app` |

## Settings

Set these as environment variables or in `.env` next to the compose file. A local try-out needs none of them. After a change, run `up -d` again.

### Address and sign-in

| Variable | What it does | Default |
| --- | --- | --- |
| `PUBLIC_URL` | The address you open the app on. Controls which origins can sign in and whether cookies are Secure. | `http://localhost:<PORT>` |
| `CONNECTOR_URL` | Public `https` address AI apps like ChatGPT use. Point your tunnel or proxy here. | `PUBLIC_URL` |
| `ALLOWED_ORIGINS` | Comma-separated origins that can sign in and change data (CSRF). `PUBLIC_URL` is always allowed. | localhost and 127.0.0.1 on `PORT` |
| `CORS_ORIGINS` | Origins browser-based AI tools can call the sign-in endpoints from. `*` allows any. | `ALLOWED_ORIGINS` |
| `CLIENT_IP_HEADER` | Header with the real visitor IP behind a proxy, such as `CF-Connecting-IP`. | none |

### Sandbox

| Variable | What it does | Default |
| --- | --- | --- |
| `SANDBOX_IMAGE` | Image the AI's commands run in. | `ghcr.io/<owner>/agentic-ai-workspace-sandbox` |
| `SANDBOX_EGRESS_HOSTS` | HTTPS host allowlist for commands; other hosts are blocked. | PyPI, npm, Go proxy and its storage host, crates.io, Maven Central |
| `CLAUDE_EGRESS_HOSTS` / `CHATGPT_EGRESS_HOSTS` | Hosts the Claude and ChatGPT programs can reach while people sign in and work on their plan. | the providers' own hosts |
| `SANDBOX_RUNTIME` | Hardened runtime, such as `runsc` (gVisor). | none |
| `SANDBOX_MEMORY_MB` / `SANDBOX_CPUS` | Limits for one check. | 2048 / 1 |
| `SANDBOX_DATA_VOLUME` | Volume holding `DATA_DIR`. Compose sets it. Leave it empty when you run the binary directly. | set by compose |

Go module ZIP downloads may redirect from `proxy.golang.org` to `storage.googleapis.com`. Keep both hosts in a custom `SANDBOX_EGRESS_HOSTS` list. This allows HTTPS to that storage hostname, not just Go module paths. The checksum database remains enabled.

The default image has Python, Node.js, Go, Rust, Java, Maven, Git, `gh`, Bash and a C compiler. It also has Bun, the runtime for apps the AI builds, and picture tools: libvips, ImageMagick, `cwebp`, `avifenc` and background removal (`removebg`, models included, works offline).

For documents it has python-pptx, python-docx, reportlab, pypdf and openpyxl (run with `docpython`). It also has LibreOffice (`docconvert file.docx pdf`), poppler (`pdftoppm`, `pdftotext`) and `docsite`, which turns a folder of documents into a website. All work offline. `make sandbox-image` builds it locally.

### Capacity

| Variable | What it does | Default |
| --- | --- | --- |
| `MAX_SANDBOXES` | Checks running at once. Extra ones wait 30 seconds, then get "busy". | 4 |
| `AGENT_WORKERS` | Tasks the AI runs at once (one per project). | 2 |
| `SITE_MAX_PER_USER` / `SITE_MAX_TOTAL` | Running [sites](./sites.md) per person and per server. 0 means no limit. | 0 / 20 |

### Web lookup

The Researcher and Media roles look things up on the web from the app itself, not from the sandbox.

| Variable | What it does | Default |
| --- | --- | --- |
| `SEARXNG_URL` | Address of your own [SearXNG](https://docs.searxng.org/) server. Empty turns web search off. | empty |

- Pictures come from Openverse and Wikimedia Commons. They need no key and no setting. Only pictures under a licence that allows edited commercial use are offered.
- Run SearXNG yourself, outside this compose file. In its `settings.yml`, add `json` to `search.formats`.
- Check it with `curl "$SEARXNG_URL/search?q=test&format=json"`. A `403` means JSON is not enabled.
- Keep SearXNG private. Its limiter may block the app when many searches arrive; relax it for the app's address.
- The app refuses internal addresses on every web request, so a page cannot point it at your network. Your own `SEARXNG_URL` is the one address it is asked to reach.

### Other

| Variable | What it does | Default |
| --- | --- | --- |
| `DATA_DIR` | Where accounts, settings and projects are stored. | `data` |
| `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`, `GITHUB_APP_SLUG` | The GitHub App (see [GitHub](#github)). Leave all three empty to turn GitHub off. | empty |

## GitHub

Each person can connect their own GitHub account so the AI can clone, commit and push for them. For that, the server needs one GitHub App (not an OAuth App). Create it on GitHub under Settings, Developer settings, GitHub Apps:

1. Set the callback URL to `<PUBLIC_URL>/api/integrations/github/callback`. Turn on "Request user authorization (OAuth) during installation". Keep "Expire user authorization tokens" on.
2. Turn the webhook off.
3. Under repository permissions, set Contents and Pull requests to read and write, and Metadata to read-only. Add Workflows only if the AI should edit `.github/workflows`.
4. Allow installs on any account if several people use the server.
5. Generate a client secret. Set the three `GITHUB_*` variables and recreate the app.

What you should know about the token:

- A person's token reaches only the repos they picked, and only what both the app and the person are allowed to do.
- It is stored encrypted with `DATA_DIR/secret.key` and renews every 8 hours.
- Only commands in that person's own projects get it, and they can use it only with GitHub. Sites never get it.
- Any program a command runs can read it during that run.
- Disconnecting revokes it at GitHub.

## Docker and security

Commands have no network of their own. They reach only the hosts in `SANDBOX_EGRESS_HOSTS` (plus GitHub, if connected), through a proxy in the app. Downloads are cached per person in `DATA_DIR/caches`, up to 5 GB.

The Claude and ChatGPT programs run in the same image, each person with a folder in `DATA_DIR/cli`. It holds their sign-in and is removed with their account.

Access to the Docker socket is access to the whole machine. To limit the risk:

- Use [rootless Docker](https://docs.docker.com/engine/security/rootless/) if you can.
- Put `DATA_DIR` on a filesystem with a size quota.
- Let only people you trust sign up.
- For stronger isolation, install gVisor and set `SANDBOX_RUNTIME=runsc`.

## Releases

- **What each workflow does:** see [system-design.md](./system-design.md#quality-and-release).
- **Jenkins and hotfix deploys:** see [devops.md](./devops.md).

## Masih Awam production

The Full workflow deploys production by itself. After it publishes the image, the `deploy` job runs on the on-premise runner.

| Part | Detail |
| --- | --- |
| File | `compose.prod.yaml`, project `masihawam` |
| App | The exact `sha-<commit>` image, on port 5004 |
| Tunnel | A `cloudflared` container that publishes the app |
| Restart | `always` for both containers |
| Data | The named volume `masihawam-data` |
| Settings | `~/masihawam/.env` on the runner host, never committed |

The settings file holds `PUBLIC_URL` and `TUNNEL_TOKEN`. In Cloudflare, route the public hostname to `http://localhost:5004`.

The job checks `/api/health` after the deploy and fails the run if the app does not answer. The runner is a user service, `actions.runner.farismnrr-agentic-ai-workspace.arch-masihawam`, with the label `masihawam-prod`.
