# Repository Guardrails

These rules apply to every human or AI change in this repository.

## Read first

Before editing code, read this file, `README.md`, and the relevant document under `docs/`. Preserve existing architectural constraints unless the user explicitly changes them.

## Runtime architecture

1. Production runtime is exactly one Go binary and one listening port.
2. Svelte is a static SPA. Fiber serves the compiled and embedded SPA assets.
3. Node.js, Vite, Tailwind CSS, and daisyUI are build-time dependencies only.
4. Do not add a separate frontend runtime server, reverse proxy requirement, or second application port without explicit approval.
5. Do not add dev-server or hot-reload workflows. Supported execution is `build -> run binary`.
6. `make run` must build first and then execute the produced binary. Do not replace it with `go run`, `vite dev`, or similar shortcuts.

## Frontend rules

1. Use Svelte for UI and daisyUI as the default component styling layer.
2. UI decision order: reuse an existing local component -> compose daisyUI semantic classes -> use Tailwind utilities for layout/detail -> write custom CSS only when necessary.
3. Repeated UI belongs in `web/src/lib/components/`. Do not duplicate page-specific copies of the same pattern.
4. Repeated frontend logic belongs under `web/src/lib/` in a focused module. Avoid generic dumping grounds such as `utils.js`.
5. Keep components small and responsibility-focused. Prefer props and composition over copy-paste variants.
6. Keep app-wide constants/configuration reusable instead of hard-coding the same value in multiple components.
7. Generated frontend output under `internal/web/dist/` is not source code and must not be manually edited.

## Backend rules

1. Use Go Fiber v3.
2. Keep `cmd/` thin: wiring and process startup only.
3. Put application behavior in focused packages under `internal/`.
4. Organize new backend features by module/domain. Shared code is extracted only when at least two modules genuinely reuse it.
5. Avoid catch-all `helpers`, `common`, or `utils` packages. Shared packages must have a clear responsibility.
6. API routes must be registered before the SPA fallback route.
7. Runtime configuration comes from environment/config packages, not scattered `os.Getenv` calls.

## Reuse and change discipline

1. Search for an existing component, function, module, or abstraction before creating a new one.
2. Prefer the smallest coherent change that solves the requirement.
3. Do not introduce an abstraction for hypothetical future reuse; extract after a concrete repeated need exists.
4. Keep dependency additions deliberate. Prefer standard library or existing dependencies when they fit.
5. Never silently weaken architecture, security, test, or repository guardrails.

## Build and verification

1. `make build` is the canonical build path.
2. The frontend must build before the Go binary so the latest SPA is embedded.
3. A completed change should leave the project able to build into `bin/agentic-ai-workspace`.
4. Do not commit `web/node_modules/`, generated SPA assets, binaries, local env files, or editor artifacts.
5. Commit dependency lockfiles when they are generated and intentionally updated.

## AI self-improvement rule

AI improvement in this repository means improving engineering behavior through evidence, not autonomously rewriting its own authority or safety rules. Follow `docs/ai-self-improvement.md`: observe -> diagnose -> make a minimal change -> verify -> reflect -> codify a reusable lesson. Changes to this guardrail require explicit user intent.
