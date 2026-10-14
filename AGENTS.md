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
6. `make run` always verifies and rebuilds before running, so the binary never goes stale.
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
2. Organize business behavior by feature, then layer (see Folder structure).
3. Avoid catch-all `helpers`, `utils`, `common`, or `misc` packages.
4. Extract shared code only after real reuse exists and give it one concrete responsibility.
5. Runtime environment access belongs in `internal/platform/config`.
6. Register API routes before the SPA fallback.
7. Prefer explicit constructor/function injection over package globals.

## Frontend rules

1. Human-authored code and tooling under `web/` must use TypeScript where applicable. `.js`, `.jsx`, `.mjs`, and `.cjs` are forbidden outside dependencies.
2. Svelte components with scripts use `<script lang="ts">`; never weaken TypeScript strictness to hide errors.
3. Use Svelte and daisyUI as the default UI layer.
4. Before creating frontend UI or logic, inspect existing `web/src/shared/ui/`, focused shared logic under `web/src/shared/`, feature-local reusable code, and the current daisyUI component set.
5. Reuse order is mandatory: existing local reusable component or logic -> daisyUI component or pattern -> Tailwind utilities -> custom CSS only when necessary.
6. Do not recreate generic controls that daisyUI already provides adequately, including buttons, inputs, selects, textareas, cards, badges, alerts, tabs, dropdowns, modals, tooltips, loading states, and similar primitives. Adapt them with semantic classes, composition, props, and the existing theme instead.
7. Every frontend change must use the existing design-system theme and semantic roles. Do not introduce a parallel palette, one-off theme, or hard-coded brand styling inside a feature.
8. Keep feature-specific UI and logic local until real reuse exists. When the same project-specific pattern is reused across features or screens, extract one reusable component or focused shared module instead of copying it.
9. Prefer props, slots/snippets, composition, and existing helpers over copied markup or duplicated behavior.
10. Generated SPA output is not source and must never be manually edited.

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
13. Auth screens follow the same rule: illustrations live in `web/src/features/auth/assets/` (via `illustrations.ts`), and the page mockups live in `assets/login-register/reference/` and are not embedded.

## Size and complexity limits

1. Every human-authored source, config, script, markup, and documentation file is limited to 200 lines.
2. Generated output and dependency lockfiles are exempt.
3. Go functions are limited by lint to 60 lines / 40 statements and strict complexity thresholds.
4. Frontend functions are limited by ESLint to 60 lines and complexity 10.
5. Split code by responsibility, never into meaningless `part1` / `part2` files just to pass the limit.

## Folder structure

Folders express responsibility. Each folder has one reason to change; add a subfolder when a folder holds more than about 8 files or mixes two responsibilities.

```text
cmd/<binary>/main.go                      process startup only
internal/bootstrap/                       composition root
internal/platform/<concern>/              config, httpserver, webassets, ...
internal/modules/<feature>/
  domain/                                 entities, value objects, rules, errors
  application/                            use cases + consumer-owned ports
  adapter/<technology>/                   http, persistence, ... one folder per technology
web/src/app/                              shell, router, composition
web/src/features/<feature>/               components/, content/, assets/, logic/ by role
web/src/shared/<concern>/                 ui, motion, theme, ... reused by 2+ features
```

1. Import direction is `app -> features -> shared` on the frontend and `adapter -> application -> domain` on the backend. Features never import other features, and `shared` never imports a feature; promote the code to `shared` once two features need it. The only inward exception is the route store in `app/router`. `scripts/check-guardrails.sh` enforces this.
2. Go packages are lowercase nouns named after their folder; Svelte components are `PascalCase.svelte`; TypeScript modules are `kebab-case.ts`; tests sit beside their code.
3. Name folders by responsibility (`thread`, `settings`, `httpserver`), never by kind-of-file buckets such as `helpers`, `utils`, `misc`, or `types`.
4. A new feature gets its layer/role folders from the start; do not grow a flat folder and split it later.
5. When code moves between folders, update this section, `docs/system-design.md`, and the guardrails in the same change.

## Branch workflow

1. `dev` is the default working branch for features, fixes, refactors, docs, and CI changes. Start work from `dev` and push to `dev`.
2. `main` is the production branch. It is updated only by the full release workflow merging a `dev` -> `main` pull request. Never commit, push, or force-push to `main`.
3. Never delete `dev` or `main`; the release merge must not delete the head branch.

## Verification and CI

1. `make verify` is the mandatory local quality gate; `make build` verifies before producing the binary.
2. Go uses strict golangci-lint; the frontend uses `svelte-check --fail-on-warnings` and type-aware strict ESLint.
3. Warnings fail verification. Fix diagnostics instead of silencing them. `//nolint` must name the linter and explain the exception.
4. Keep exactly two workflow files in `.github/workflows/`: `fast.yml` and `full.yml`. Neither gets extra files or full-release auto-triggers without approval. Shared setup lives in `.github/actions/`.
5. **Fast** runs on pushes to `dev` (and pull requests targeting `dev`, without publishing). It runs `make verify` equivalents, builds the single `linux/amd64` static binary, and smoke-tests it (starts, serves `/` and an API route). Checks run in parallel and the build is not repeated. Documentation-only changes skip it.
6. **Full** is manual (`workflow_dispatch`) and runs only from `dev`. It repeats every fast check from a clean checkout, builds and smoke-tests the release binary, and only then merges `dev` -> `main` and publishes the rolling release.
7. Fast stays quicker than full by skipping clean-room rebuilds, cross-checks, merging, and publishing. It must never skip a lint, type, test, or guardrail gate.
8. CI uses committed lockfiles, deterministic installs, and official caches. Never cache `node_modules`. CI verifies dependency drift instead of fixing it.
9. Third-party Actions are pinned to full commit SHAs with the version documented beside them.
10. Permissions default to read-only; write permission belongs only to the full release job.
11. A change is incomplete while CI is failing.

## Rolling GitHub release

1. Only a successful full workflow publishes a release. Pushes to `dev` and pull requests never do.
2. Keep exactly one rolling release identity: tag `latest`, title `Latest`. No semantic, dated, or per-commit versions unless the user changes this rule.
3. The asset is the verified binary only, named `agentic-ai-workspace-linux-amd64`. Publish the binary built in that run; never rebuild different source.
4. A newer release moves `latest` to the merged `main` commit and replaces the asset. If immutable releases are enabled, fail clearly instead of creating versioned releases.

## Backend platform lessons

1. Migrations in `internal/platform/database/migrations/` are an append-only journal with checksums: never edit or delete an applied file; every schema or query change is a new numbered file. Startup fails if history was altered.
2. The server is its own OAuth 2.1 authorization server (`modules/oauth`) and hosts MCP spaces (`modules/space`; first is `modules/coding` at `/api/coding`). Tokens are opaque and stored hashed; a space accepts only tokens whose `resource` is its own URL. Features meet only in `internal/bootstrap/bridges.go`. Open endpoints (sign-in, sign-up, OAuth) stay rate limited via `bootstrap/limits.go`.
3. Discovery contract: protected resource metadata at `/.well-known/oauth-protected-resource/api/<slug>`, server metadata at `/.well-known/oauth-authorization-server` and `openid-configuration`, CIMD (SSRF-guarded, public IPs only), DCR, PKCE S256, RFC 9207 `iss`. Keep `internal/bootstrap/e2e_test.go` passing when touching any of it.
4. A new space is a `space.Space` plus a bootstrap mount, never an `oauth` special case; the coding space is off unless `CODING_WORKSPACE_ROOT` is set (keep `DATA_DIR` outside it). `coding/domain.Tools()` is the single tool catalogue; calls pass permission, then scope, then run, are always logged, and no tool may reach `.workspace/` (task definitions). Write and run stay separate scopes, Off by default.
5. Adapters answer errors with `c.Status(n).JSON(httpserver.Failure(code))`; returning a wrapped error after writing a response makes Fiber overwrite it with a 500. Direct Q&A uses `modules/chat` with SSE and one-click task escalation to `modules/task`.

## Frontend and routing lessons

1. Headings get `font-display` from the base layer in `web/src/app.css`. Serif is for section headlines only; navbar and footer wordmarks use Inter 600; the logo is the inline stroke SVG in `LogoMark.svelte`.
2. Scroll reveals use the `reveal` action in `web/src/shared/motion/`; the hidden state exists only under `prefers-reduced-motion: no-preference` with the `js-reveal` root class. Below-the-fold sections use `LazySection` with the section `id` on the placeholder.
3. Favicon, touch icon, and manifest are static files in `web/public/`; Go serves hashed `/assets/*` as immutable and `index.html` as `no-cache`.
4. The SPA has no router library: `web/src/app/router.ts` is a History API router over a store of parsed routes. Add static paths to `staticRoutes` and `/app/*` patterns to `appPatterns`. The Go fallback must force status 200; missing `/assets/*` stay 404.
5. Every workspace screen talks to the real API (`logic/*-api.ts`, plain-TS stores in `logic/workspace-store.ts` kept fresh by `AppShell`, `logic/thread.ts` for one task); `/app/*` requires a session. `content/` holds copy only, never sample data. Never fake success; a page without a backend says so plainly. Notifications are per device (`logic/notify.ts`, localStorage + Notification API). The Inbox and counts are derived from tasks in `logic/derive.ts`.
6. The workspace uses the compact `app-ui` scale from `shell/AppShell.svelte` (14px text, hairline-divided flat rows, no card shadows, `btn-sm`, `h-10` inputs). Pages declare `spacious`, `dense`, or `split` width; split pages own their height and scroll inside panes. Home's greeting is the only serif.
7. Task status looks come only from `content/status.ts`; the thread is the typed `TaskEvent` union from the API rendered by `task/thread/EventItem.svelte`; consecutive actions collapse (`task/thread/blocks.ts`); the composer is one component (`components/ComposerBox.svelte`).
8. The agent loop (`modules/task`) rebuilds from the stored transcript, so a task can pause for an answer and resume. Proof is derived only from tasks the server ran (`application/proof.go`), never from the agent's claims. API keys are read only through `provider.Service.Credential`.
9. Settings use `settings/parts/` with tabs driven by `settings/settings-nav.ts`; create-style forms open in a daisyUI `modal`.
10. Icon names live in `shared/ui/icon-names.ts`; `Icon.svelte` holds only the paths.
11. Required bindable props are declared `value?: string | undefined` with `$bindable('')`. `.svelte.ts` rune modules are not parsed by ESLint; keep shared state in plain `.ts` stores.

## Reuse and change discipline

1. Search for an existing component, function, module, or abstraction before creating a new one.
2. Prefer the smallest coherent change that solves the requirement.
3. Do not add abstractions for hypothetical reuse.
4. Keep dependencies deliberate and pinned when practical.
5. Never silently weaken architecture, lint, security, typing, tests, CI, or repository guardrails.

## Self-improvement and repository memory

For every meaningful task, follow: **observe -> diagnose -> plan -> implement -> verify -> reflect -> codify**. Read the repository, tests, logs, CI, and current docs; separate evidence from assumptions; make the smallest coherent change in the correct boundary; verify behavior and the canonical quality path; codify durable lessons as a test, reusable abstraction, guardrail, or concise memory here.

Before finishing a meaningful task, review this file as repository memory:

1. Add a lesson only when it is likely to matter again.
2. Update stale facts instead of appending contradictory rules.
3. Remove obsolete agent memory when the architecture or user intent replaces it.
4. Keep memory concise and durable; do not turn this file into a chronological work log.
5. Never claim a verification step that was not actually performed.
6. Self-improvement never grants authority to weaken these rules or override explicit user intent.
