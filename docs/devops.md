# Build, publish, and deploy

AI Code distributes **two** OCI images via GitHub Container Registry (GHCR):

| Image | Purpose |
| --- | --- |
| `ghcr.io/farismnrr/agentic-ai-workspace` | Trusted Go app + compiled UI |
| `ghcr.io/farismnrr/agentic-ai-workspace-sandbox` | Isolated command runner with developer tools and skills |

## GitHub Actions is the publisher

- `.github/workflows/fast.yml` runs on every push to `dev`. It builds and pushes both images under immutable `sha-<full-SHA>` tags, then updates both moving `:dev` tags.
- `.github/workflows/full.yml` is manually dispatched on `dev` for releases. It builds both immutable images, then promotes matching `:latest` and `:release` tags.
- Each image has the same `org.opencontainers.image.revision` label. Actions use the scoped `GITHUB_TOKEN` with `packages: write`. No registry password is stored in the source.
- A failed build must not promote a new channel. Promotions are sequential (not an atomic registry transaction), so host pulls verify the labels again before any app restart.

Review the Actions run result before deploying. Do not assume pushing a Git commit means the corresponding images have already been published.

## Deployment hosts only pull

For the `dev` channel, run from the repository checkout:

```bash
git pull --ff-only origin dev
make redeploy-dev
```

For the stable `latest` channel:

```bash
make redeploy-latest
```

Both commands pull the app and sandbox from GHCR using `scripts/pull-ghcr-pair.sh`. If the image labels disagree, the command exits without recreating the running app.

To pull without restarting anything, use `bash scripts/pull-ghcr-pair.sh dev`. If a GHCR package is private, configure read-only `read:packages` login on the host.

Only use locally built `agentic-ai-workspace-local` images for developer experiments. They must not be substituted into production Compose configurations. Do not use Docker-in-Docker or give the sandbox the host Docker socket.

## Verify the running deployment

```bash
bash scripts/check-docker-host.sh dev
docker compose -f compose.dev.yaml ps
docker compose -f compose.dev.yaml logs --tail 50 app
```

Check `/api/health` at the configured `PUBLIC_URL`. The app uses the host Docker daemon to launch isolated sandbox containers through its trusted SDK. The sandbox itself has no host socket, capabilities, or network interface.

## Rollback

Every Actions publication retains matching immutable `sha-<full-SHA>` image tags. For a controlled rollback, select the *same* known-good revision for both app and sandbox and verify the image labels before restarting. Preserve the data volume and never use `down -v`.

See [deployment.md](./deployment.md) for setup, environment settings, logs and host access. MoLab network grants remain disabled until their independent live security gate passes.