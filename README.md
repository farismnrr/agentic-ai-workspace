# agentic-ai-workspace

Single-binary foundation for an AI workspace.

## Stack

- Backend: Go 1.26 + Fiber v3
- Frontend: Svelte 5 SPA built with Vite
- UI: Tailwind CSS + daisyUI
- Runtime: one Go process, one port, one executable

Node.js, Vite, Tailwind CSS, and daisyUI are build-time dependencies only. The compiled frontend is embedded into the Go executable and served by Fiber.

## Build and run

```bash
make build
./bin/agentic-ai-workspace
```

Or build and immediately run:

```bash
make run
```

The default address is `:8080`. Override it with `PORT`:

```bash
PORT=9000 ./bin/agentic-ai-workspace
```

There is intentionally no dev server or hot-reload workflow. Every supported local run follows `build -> binary`.

## Project layout

```text
cmd/workspace/              Go executable entrypoint
internal/config/            Runtime configuration
internal/server/            Fiber app composition and routes
internal/web/               Embedded SPA filesystem
web/                        Svelte source and build-only tooling
web/src/lib/components/     Reusable Svelte UI components
docs/                       Architecture and AI engineering guidance
```

Read [AGENTS.md](./AGENTS.md) before making changes. Architecture decisions are documented in [docs/system-design.md](./docs/system-design.md), and the evidence-driven AI improvement loop is documented in [docs/ai-self-improvement.md](./docs/ai-self-improvement.md).
