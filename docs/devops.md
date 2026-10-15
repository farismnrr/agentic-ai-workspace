# Build, hotfix deploy and Jenkins CI/CD

How the image is built, how a fixed binary is swapped into the running container without a rebuild, and how Jenkins chains it all together. Tools: Git, Docker 29, Go **1.27.2**, Jenkins 2.580.1 (local, in Docker with a `docker:dind` daemon).

## Part I: Build

The [`Dockerfile`](../Dockerfile) has three stages:

1. `node:24` compiles the Svelte frontend.
2. `golang:1.27.2` embeds it and builds one binary with `CGO_ENABLED=0`, so it is statically linked and needs no libc or other shared library from the builder.
3. `gcr.io/distroless/static-debian12:nonroot` holds that binary and nothing else.

**Why distroless static.** It is about 2 MB and has only CA certificates, timezone data, `/etc/passwd` and a non-root user (65532). There is no shell or package manager, so there is little to attack or patch. `scratch` would be smaller but lacks CA certificates and a non-root user, which this app needs for outbound HTTPS. `alpine` adds a shell and musl that the static binary does not use. The container health check is the binary itself (`agentic-ai-workspace healthcheck`), so no curl is needed inside.

**Version injection.** `cmd/workspace/main.go` declares `var version = "dev"`. The build sets it with `-ldflags "-X main.version=<version>"`: through `ARG VERSION` in the Dockerfile, or `VERSION=` with `make backend-build`. The running build names its version on **every** response, including `GET /`, in the `X-App-Version` header, and in the body of `GET /api/health`. (`/` itself serves the web app's HTML, so the header is how it reports the version.)

```bash
docker build --build-arg VERSION=1.0.0 -t ghcr.io/farismnrr/agentic-ai-workspace:1.0.0 .
docker run --rm -d -p 8080:8081 ghcr.io/farismnrr/agentic-ai-workspace:1.0.0
curl -si http://localhost:8080/ | grep -i x-app-version     # X-App-Version: 1.0.0
curl -s  http://localhost:8080/api/health                   # {"status":"ok","version":"1.0.0"}
```

The image runs anywhere Docker runs; no Go toolchain is needed on the host.

**Image size** (`docker images`, from the Jenkins run):

```text
IMAGE                                                ID             DISK USAGE   CONTENT SIZE
ghcr.io/farismnrr/agentic-ai-workspace:sha-b6e2744   9aa0201ba09c         41MB         11.5MB
```

About 41 MB on disk (11.5 MB compressed). The distroless base is about 2 MB. Almost everything else is the one binary layer (23.2 MB in `docker history`): the Go runtime, Fiber, SQLite (pure Go), the Docker client and the whole web app embedded in it. `-s -w` strips the debug symbols. There is no OS userland to shrink further.

## Part II: Deploy and swap the binary

**Approach: the binary lives in a mounted folder, not in the image.** The container is created once from the image, but its entrypoint is `/opt/app/agentic-ai-workspace`, a host folder mounted read-only. A hotfix copies the new binary into that folder under a temporary name, renames it over the old one (atomic), and restarts the container. The folder is mounted rather than the single file because a single-file bind mount keeps the old inode, so a renamed file would not be seen inside. [`scripts/deploy-binary.sh`](../scripts/deploy-binary.sh) does it all.

```bash
# first run: create the container (port 8080, restart on crash)
docker run -d --name agentic-ai-workspace-prod --restart unless-stopped \
  -p 8080:8080 -e PORT=8080 \
  -v /var/jenkins_home/deploy/agentic-ai-workspace:/opt/app:ro \
  -v agentic-ai-workspace-prod-data:/data \
  --entrypoint /opt/app/agentic-ai-workspace ghcr.io/farismnrr/agentic-ai-workspace:sha-b6e2744

# hotfix: new binary in, keep the old one, restart (no docker build, container stays)
make backend-build VERSION=<new>
cp bin/agentic-ai-workspace "$DIR/agentic-ai-workspace.new" && chmod 755 "$DIR/agentic-ai-workspace.new"
cp "$DIR/agentic-ai-workspace" "$DIR/agentic-ai-workspace.prev"
mv -f "$DIR/agentic-ai-workspace.new" "$DIR/agentic-ai-workspace"
docker restart -t 5 agentic-ai-workspace-prod
```

Proof from Jenkins build #15 (the deploy stage, which runs the script):

```text
before swap:
X-App-Version: b6e2744-b14
{"status":"ok","version":"b6e2744-b14"}
after swap (back in 2s, no image rebuild):
X-App-Version: b6e2744-b15
{"status":"ok","version":"b6e2744-b15"}
```

**Why this approach.** For a production hotfix the swap must be fast, repeatable and easy to undo. Swapping a file and restarting takes about 1–2 seconds of downtime and keeps the same container, network, ports, volumes and restart policy, with no rebuild. Unlike `docker cp` into the container, the binary is never written inside a running container's filesystem, so it survives a container recreate and avoids "text file busy" on an executing file. Keeping the previous binary next to it makes rollback a rename plus a restart. The trade-off is that the running binary no longer matches the image tag, so each binary reports its own version and the image is rebuilt on the next normal release.

![Deploy stage of build #15 showing the version before and after the swap](./images/jenkins-deploy-swap.png)

## Part III: CI/CD with Jenkins

The [`Jenkinsfile`](../Jenkinsfile) runs these stages in order:

| Stage | What it does |
| --- | --- |
| Checkout | Clones the repository at the `BRANCH` parameter (default `main`). Version = `<short commit>-b<build number>`. |
| Test | `make frontend-verify`, then `make guardrails backend-verify` (includes `go test ./cmd/... ./internal/...` and lint), in `node` and `golang` containers. Any failure stops the pipeline: nothing is built, pushed or deployed. |
| Build binary | `make backend-build` with the version in `-ldflags`, then a smoke test of the binary. |
| Build image | `docker build --build-arg VERSION=...` tagged `sha-<commit>`, a smoke test of the image, then the sandbox image. |
| Push | Logs in to GHCR with the `ghcr-credentials` Jenkins credential (credentials binding, never in the file) and pushes `sha-<commit>`; only `main` also moves `latest`. |
| Deploy | Runs `scripts/deploy-binary.sh`: the Part II binary swap, via the Docker daemon the agent already uses. |

The Jenkins container only has Git, curl and Docker. Node.js and Go run in short-lived containers that mount the workspace at the same path, which works because the `docker:dind` daemon shares `/var/jenkins_home` with Jenkins. Port 8080 is published on that daemon's host.

![All stages green in Jenkins build #15](./images/jenkins-pipeline-run.png)

**Rollback when the deploy fails midway.** Before swapping, the script copies the running binary to `agentic-ai-workspace.prev`. After the restart it waits up to 30 seconds for `/api/health` to report the new version. If that does not happen (the binary crashes, does not start, or reports the wrong version), it prints the container logs, renames `.prev` back, restarts, waits for the old version, and exits non-zero, so the Jenkins build is red. The container itself is never removed, so a failed deploy leaves the last good binary running. Earlier builds stay available as `sha-<commit>` images in GHCR, and re-running the job on an older commit redeploys that version. A failure before Deploy (tests, build, push) never touches the running app.
