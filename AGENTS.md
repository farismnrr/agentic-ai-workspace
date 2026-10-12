# Repository Guardrails

These rules are binding for every human or AI change in this repository.

## Read first

Before editing code, read this file, `README.md`, and the relevant files under `docs/`.
Preserve these constraints unless the user explicitly changes them.

## Non-negotiable runtime contract

1. Production is one Go binary, one process, and one listening port.
2. Go Fiber v3 owns API routing and serves the compiled Svelte SPA.
3. Node.js, Vite, Tailwind CSS, and daisyUI are build-time dependencies only.
4. Do not add a second runtime server, frontend port, or reverse-proxy requirement without explicit approval.
5. Do not add hot reload or dev-server workflows. Supported execution is `build -> run binary`.
6. `make run` must build and verify before it executes the binary.

## Clean Architecture dependency rule

Source dependencies point inward toward policy. Frameworks and I/O are details.

Backend feature layout:

```text
internal/modules/<feature>/
  domain/        pure business rules and value types
  application/   use cases and consumer-owned ports/interfaces
  adapter/       HTTP, persistence, external-service implementations
```

Cross-cutting outer details live under `internal/platform/`. Process composition lives only under
`internal/bootstrap/`. `cmd/` stays thin.

Rules:

1. `domain` must not import Fiber, platform, bootstrap, databases, environment access, or transport DTOs.
2. `application` may depend on its domain and abstractions it owns; it must not import Fiber or platform details.
3. `adapter` may depend inward on application/domain and may use frameworks required by that adapter.
4. `platform` contains technical process-wide details, not business rules.
5. `bootstrap` is the composition root and is the only place expected to wire concrete outer dependencies.
6. Do not import another feature's adapter implementation. Cross-feature interaction goes through explicit application contracts.
7. Interfaces belong with the consumer that needs the abstraction. Do not create interfaces only to mirror concrete structs.

`golangci-lint` `depguard` rules enforce the most important inner-layer dependency boundaries.

## SOLID rules

1. Single Responsibility: packages, files, functions, components, and services have one cohesive reason to change.
2. Open/Closed: prefer composition, small ports, and new adapters over growing switch statements or central god objects.
3. Liskov Substitution: every implementation of a port must preserve the contract expected by its consumer.
4. Interface Segregation: keep interfaces small and consumer-specific. Avoid broad service/repository interfaces.
5. Dependency Inversion: business policy depends on abstractions; outer adapters implement those abstractions.

SOLID is not permission to over-abstract. A stable domain boundary or real substitution need must justify an abstraction.

## Backend code rules

1. Keep `cmd/` to process startup only.
2. Organize business behavior by feature module, then by Clean Architecture layer inside the module.
3. Avoid catch-all `helpers`, `utils`, `common`, or `misc` packages.
4. Shared code is extracted only after real reuse exists and must have a concrete responsibility.
5. Runtime environment access belongs in `internal/platform/config`.
6. API routes are registered before the SPA fallback.
7. Prefer explicit dependency injection through constructors/functions over package globals.

## Frontend rules

1. All human-authored files under `web/` that can be TypeScript must be TypeScript. `.js`, `.jsx`, `.mjs`, and `.cjs` are forbidden outside dependencies.
2. Every Svelte component containing a script block must use `<script lang="ts">`.
3. TypeScript runs in strict mode with additional strictness flags; do not weaken them to make errors disappear.
4. Use Svelte for UI and daisyUI as the default component layer.
5. UI decision order: reuse local component -> compose daisyUI -> Tailwind utilities -> custom CSS only when necessary.
6. Reusable UI lives under `web/src/shared/ui/`; reusable focused logic belongs under `web/src/shared/`.
7. Feature-specific code belongs under `web/src/features/<feature>/` and should keep domain/application logic separate from UI.
8. `web/src/app/` is application composition, not a dumping ground for feature behavior.
9. Prefer props and composition over copied markup or page-specific variants.
10. Generated SPA output is not source code and must never be manually edited.

## Mandatory size and complexity limits

1. Every human-authored source, config, script, markup, and documentation file is limited to 200 lines.
2. Generated output and dependency lockfiles are exempt from the file-size gate.
3. Go functions are limited by lint to 60 lines / 40 statements and strict complexity thresholds.
4. Frontend functions are limited by ESLint to 60 lines and complexity 10.
5. Split by responsibility when a limit is approached. Do not create meaningless `part1`/`part2` files to bypass the rule.

The limits are enforced by `scripts/check-guardrails.sh`, ESLint, and golangci-lint.

## Lint and verification

1. `make verify` is the mandatory quality gate.
2. `make build` runs verification before producing the binary.
3. Go uses golangci-lint v2 with strict correctness, security, complexity, formatting, and architecture checks.
4. Frontend uses `svelte-check --fail-on-warnings` plus type-aware strict ESLint rules.
5. Warnings are failures. Do not silence a diagnostic without fixing it or documenting a narrow justified suppression.
6. `//nolint` directives must name the linter and explain the exception.
7. A completed change must keep `make verify` and `make build` green.

## Reuse and change discipline

1. Search for an existing component, function, module, or abstraction before creating a new one.
2. Prefer the smallest coherent change that solves the requirement.
3. Do not add abstractions for hypothetical reuse.
4. Keep dependencies deliberate and pinned when practical.
5. Never silently weaken architecture, lint, security, typing, tests, or repository guardrails.

## AI self-improvement

Follow `docs/ai-self-improvement.md`: observe -> diagnose -> plan -> implement -> verify -> reflect -> codify.
Self-improvement means improving engineering behavior through evidence. It never authorizes an agent to weaken these rules
or expand its own authority. Changes to guardrails require explicit user intent.
