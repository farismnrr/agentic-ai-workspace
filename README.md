# agentic-ai-workspace

Single-binary foundation for an AI workspace with strict architecture and quality gates.

## Stack

- Backend: Go 1.27.2 + Fiber v3
- Frontend: Svelte 5 SPA + strict TypeScript
- UI: Tailwind CSS + daisyUI
- Runtime: one Go process, one port, one executable
- Go quality gate: golangci-lint v2.14.0
- Frontend quality gate: svelte-check + type-aware strict ESLint

Node.js and the frontend toolchain are build-time dependencies only. The compiled SPA is embedded into the Go executable.

## Canonical workflow

Before coding or executing project code for a task, review current official documentation and best practices for the affected stack as required by `AGENTS.md`.

```bash
make verify
make build
./bin/agentic-ai-workspace
```

Or build, verify, and immediately run:

```bash
make run
```

The default address is `:8080`. Override it at runtime:

```bash
PORT=9000 ./bin/agentic-ai-workspace
```

There is intentionally no dev server or hot-reload command. Local execution always follows `build -> binary`.

## Continuous integration

GitHub Actions runs on every push and pull request. Guardrails, frontend checks, Go tests, and Go lint run independently where possible; `Quality / quality-gate` starts only after all verification jobs pass and then compiles the final binary with the verified SPA artifact.

CI uses the committed npm lockfile with `npm ci`, the official npm and Go caches keyed by `web/package-lock.json` and `go.sum`, and the official cached golangci-lint action. Third-party actions are pinned to full commit SHAs. CI remains read-only during normal quality runs.

A change is not complete while any quality job is failing.

## Architecture

Backend business features use feature-first Clean Architecture:

```text
internal/modules/<feature>/
  domain/
  application/
  adapter/
```

Technical outer details live in `internal/platform/`, composition lives in `internal/bootstrap/`, and `cmd/` remains thin.
The dependency rule is enforced by lint where it can be checked statically.

Frontend composition lives under `web/src/app/`. Future feature code belongs under `web/src/features/`, while reusable
components and focused logic belong under `web/src/shared/`.

See [docs/system-design.md](./docs/system-design.md) for the full dependency rules and [AGENTS.md](./AGENTS.md) for binding
guardrails.

## Hard quality rules

- human-authored files: maximum 200 lines;
- Go functions: maximum 60 lines / 40 statements plus complexity limits;
- frontend source: TypeScript only, with strict compiler flags;
- Svelte script blocks: TypeScript only;
- warnings fail verification;
- daisyUI and existing reusable components are preferred before custom UI;
- no second runtime port or frontend server.

Run the complete gate with:

```bash
make verify
```
