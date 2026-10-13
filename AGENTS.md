# Agent Rules and Repository Memory

This file is the source of truth for AI agents working in this repository. It contains binding engineering rules, durable repository memory, and the self-improvement loop.

## Documentation audiences

1. `README.md` and files under `docs/` are written for humans. Keep them clear, practical, and easy to understand without agent-specific instructions.
2. Agent-only instructions, operating rules, self-improvement behavior, and durable AI memory belong in `AGENTS.md`.
3. Do not move agent prompts, hidden workflow instructions, or agent memory into human documentation.
4. When a durable project rule changes, update `AGENTS.md` in the same task so future agents inherit the current truth.
5. Keep human docs synchronized with user-visible behavior, but explain the behavior rather than instructing an agent how to think.

## Read and research before coding

1. Read `AGENTS.md`, `README.md`, and relevant human docs before editing.
2. Before the first code edit or project-code execution of every task, search the internet for current official documentation and best practices for the affected stack.
3. Prefer primary sources: official docs, specifications, release notes, security guidance, and upstream repositories.
4. Confirm current compatibility, deprecations, breaking changes, and security guidance. Do not rely only on model memory for mutable technology facts.
5. Apply research findings before modifying or executing project code.
6. Research again if the scope expands to a new framework, dependency, tool, or materially different technical area.

## Runtime contract

1. Production is one Go binary, one process, and one listening port.
2. Go Fiber v3 owns API routing and serves the compiled Svelte SPA.
3. Node.js, Vite, Tailwind CSS, and daisyUI are build-time dependencies only.
4. Do not add a second runtime server, frontend port, or reverse proxy without explicit approval.
5. Do not add hot reload or a dev-server workflow. Supported execution is `build -> run binary`.
6. `make run` must verify and build before running the binary (unlike a build-if-missing flow, it always rebuilds so the binary never goes stale).
7. `backend-build` always uses `CGO_ENABLED=0`, so local and release binaries are the same static, single-file artifact.

## Clean Architecture dependency rule

Dependencies point inward toward policy. Frameworks and I/O are outer details.

```text
internal/modules/<feature>/
  domain/
  application/
  adapter/
```

Process-wide technical details live in `internal/platform/`, dependency wiring lives in `internal/bootstrap/`, and `cmd/` stays thin.

1. `domain` must not import Fiber, platform, bootstrap, databases, environment access, or transport DTOs.
2. `application` may depend on domain and abstractions it owns, but not Fiber or platform details.
3. `adapter` may depend inward and may use the framework or SDK required by that adapter.
4. `platform` contains technical process-wide details, never business rules.
5. `bootstrap` is the composition root for concrete dependencies.
6. Cross-feature interaction goes through explicit application contracts, never another feature's adapter.
7. Interfaces belong with the consumer and must represent a real boundary or substitution need.

## SOLID

1. Single Responsibility: each package, file, function, component, and service has one cohesive reason to change.
2. Open/Closed: prefer composition and focused adapters over central switches or god objects.
3. Liskov Substitution: implementations must preserve the contract expected by the consumer.
4. Interface Segregation: keep interfaces small and consumer-specific.
5. Dependency Inversion: inner policy owns abstractions; outer details implement them.
6. SOLID is not permission to over-abstract. Prefer simple concrete code until a real boundary exists.

## Backend rules

1. Keep `cmd/` to process startup only.
2. Organize business behavior by feature, then by Clean Architecture layer.
3. Avoid catch-all `helpers`, `utils`, `common`, or `misc` packages.
4. Extract shared code only after real reuse exists and give it one concrete responsibility.
5. Runtime environment access belongs in `internal/platform/config`.
6. Register API routes before the SPA fallback.
7. Prefer explicit constructor/function injection over package globals.

## Frontend rules

1. Human-authored code and tooling under `web/` must use TypeScript where applicable. `.js`, `.jsx`, `.mjs`, and `.cjs` are forbidden outside dependencies.
2. Svelte components with scripts must use `<script lang="ts">`.
3. TypeScript strictness must not be weakened to hide errors.
4. Use Svelte and daisyUI as the default UI layer.
5. Before creating frontend UI or logic, inspect existing `web/src/shared/ui/`, focused shared logic under `web/src/shared/`, feature-local reusable code, and the current daisyUI component set.
6. Reuse order is mandatory: existing local reusable component or logic -> daisyUI component or pattern -> Tailwind utilities -> custom CSS only when necessary.
7. Do not recreate generic controls that daisyUI already provides adequately, including buttons, inputs, selects, textareas, cards, badges, alerts, tabs, dropdowns, modals, tooltips, loading states, and similar primitives. Adapt them with semantic classes, composition, props, and the existing theme instead.
8. Every frontend change must use the existing design-system theme and semantic roles. Do not introduce a parallel palette, one-off theme, or hard-coded brand styling inside a feature.
9. Reusable UI lives in `web/src/shared/ui/`; focused shared logic lives under `web/src/shared/`.
10. Keep feature-specific UI and logic local until real reuse exists. When the same project-specific pattern is reused across features or screens, extract one reusable component or focused shared module instead of copying it.
11. Feature code belongs in `web/src/features/<feature>/`; app composition belongs in `web/src/app/`.
12. Prefer props, slots/snippets, composition, and existing helpers over copied markup or duplicated behavior.
13. Generated SPA output is not source and must never be manually edited.

## Design system rules

1. `docs/design-system.md` is the source of truth for user-visible visual semantics and theme behavior.
2. Preserve Masih Awam branding through semantic theme roles, not by copying LMS implementation details or framework choices.
3. Use daisyUI semantic roles before literal product colors: `primary`, `secondary`, `accent`, `neutral`, `info`, `success`, `warning`, `error`, and `base-*`.
4. Product terminology calls daisyUI `accent` the tertiary role. Disabled, loading, hover, active, focus, and selected are interaction states, not extra brand colors.
5. The brand trio is split-complementary and anchored on the Masih Awam olive/lime primary: secondary is blue and tertiary is violet. Keep status colors semantic rather than forcing them into the brand harmony.
6. The app ships one theme, `masihawam-light`. Do not add a dark theme, theme toggle, or `dark:` variants unless the user explicitly reintroduces them.
7. Keep brand values in the central theme source. Do not scatter hard-coded Masih Awam colors across feature components.
8. Prefer configuring and composing daisyUI with the central theme over introducing custom component styling. Add custom tokens or CSS only for capabilities that daisyUI and Tailwind do not already cover cleanly.
9. Preserve visible focus, sufficient contrast, non-color status cues, and reduced-motion behavior.
10. Before reporting landing or marketing UI as complete, compare the final implementation against current public design guidance and explicitly check for generic AI-default patterns such as repeated eyebrow/headline/body stacks, interchangeable hype copy, decorative gradients or glows, repeated equal card grids, excessive mono/uppercase decoration, and unnecessary nested surfaces. Fix unjustified patterns instead of merely noting them.
11. Product positioning is proof-first AI work for people who are not developers: tell it what you need, see every step, and accept only what is proven (the receipt, the "Proven to work" stamp). Use plain language, not agent or orchestration jargon. Avoid generic chatbot or abstract AI imagery when the real workflow can tell the story.
12. Landing visuals are the hand-drawn illustrations in `web/src/features/landing/assets/` (imported through `illustrations.ts` so Vite hashes them and dimensions stay in one place). Section references live in `assets/landing/reference/` and are not embedded. Overlay UI (checklist, toast, form) is real HTML, not baked into images.

## Size and complexity limits

1. Every human-authored source, config, script, markup, and documentation file is limited to 200 lines.
2. Generated output and dependency lockfiles are exempt.
3. Go functions are limited by lint to 60 lines / 40 statements and strict complexity thresholds.
4. Frontend functions are limited by ESLint to 60 lines and complexity 10.
5. Split code by responsibility, never into meaningless `part1` / `part2` files just to pass the limit.

## Verification and CI

1. `make verify` is the mandatory local quality gate.
2. `make build` verifies before producing the binary.
3. Go uses strict golangci-lint checks for correctness, security, architecture, formatting, and complexity.
4. Frontend uses `svelte-check --fail-on-warnings` and type-aware strict ESLint.
5. Warnings fail verification. Fix diagnostics instead of silencing them unless a narrow suppression is justified.
6. `//nolint` must name the linter and explain the exception.
7. `.github/workflows/quality.yml` runs on every push and pull request and must preserve the semantic equivalent of `make verify` plus final binary compilation.
8. Independent checks should run in parallel when safe.
9. CI uses committed lockfiles, deterministic installs, and official caches. Never cache `node_modules`.
10. Third-party GitHub Actions are pinned to full commit SHAs with the intended version documented beside them.
11. CI defaults to least-privilege read-only permissions. Grant write permission only to the job that truly requires it.
12. CI verifies dependency metadata drift instead of silently fixing it.
13. A change is incomplete while quality CI is failing.

## Rolling GitHub release

1. Every successful push to `main` must publish the verified Linux binary to GitHub Releases.
2. Keep exactly one rolling release identity: tag `latest`, release title `Latest`.
3. Do not create semantic, dated, or per-commit release versions unless the user explicitly changes this rule.
4. The release asset is the built binary only, named `agentic-ai-workspace-linux-amd64`.
5. Build once in the quality gate and publish that verified binary; do not rebuild different source for release.
6. A newer successful `main` push moves `latest` to that commit and replaces the existing asset.
7. Release publication must depend on all quality checks passing and must never run for pull requests.
8. Keep release write permission isolated to the release job.
9. Rolling replacement requires mutable releases. If repository immutable releases are enabled, fail clearly rather than creating parallel versioned releases.

## Frontend lessons

1. Headings get `font-display` from the base layer in `web/src/app.css`; do not repeat it per component. Serif is for section headlines only; the navbar and footer wordmark use Inter 600, and the logo is the inline stroke SVG in `LogoMark.svelte`.
2. Scroll reveals use the `reveal` action in `web/src/shared/motion/`, and the hidden state exists only under `prefers-reduced-motion: no-preference` with the `js-reveal` root class, so content never stays invisible.
3. Below-the-fold sections load through `LazySection`. Give the placeholder the section `id` so anchor links keep working.
4. Favicon, touch icon, and manifest are static files in `web/public/`. Vite copies them into the embedded `dist/`, and Go serves hashed `/assets/*` as immutable with `index.html` set to `no-cache`.

## Reuse and change discipline

1. Search for an existing component, function, module, or abstraction before creating a new one.
2. Prefer the smallest coherent change that solves the requirement.
3. Do not add abstractions for hypothetical reuse.
4. Keep dependencies deliberate and pinned when practical.
5. Never silently weaken architecture, lint, security, typing, tests, CI, or repository guardrails.

## Self-improvement and repository memory

For every meaningful task, follow: **observe -> diagnose -> plan -> implement -> verify -> reflect -> codify**.

- Observe the repository, tests, logs, CI, and current official documentation.
- Diagnose concrete evidence separately from assumptions.
- Plan the smallest coherent change that preserves architecture and reuse rules.
- Implement in the correct boundary.
- Verify the actual behavior and the canonical quality path.
- Reflect on whether the task exposed a repeatable engineering lesson.
- Codify durable lessons as a test, reusable abstraction, guardrail, or concise memory in this file.

Before finishing a meaningful task, review this file as repository memory:

1. Add a lesson only when it is likely to matter again.
2. Update stale facts instead of appending contradictory rules.
3. Remove obsolete agent memory when the architecture or user intent replaces it.
4. Keep memory concise and durable; do not turn this file into a chronological work log.
5. Never claim a verification step that was not actually performed.
6. Self-improvement never grants authority to weaken these rules or override explicit user intent.
