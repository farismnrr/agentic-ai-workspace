# Agent Rules

This file is the entry point for AI agents working in this repository: binding engineering rules, plus pointers to the agent-only knowledge in `agents/`.

## Documentation audiences

Three audiences, three places. Never mix them.

1. `README.md` is general: what the project is, the stack, the shortest way to run it, and links to `docs/`. No settings tables, no deep how-tos, no agent instructions.
2. `docs/` is for people: deployment, configuration, releases, using the app, architecture, and the design system. Explain behavior; never instruct an agent how to think.
3. `agents/` (and this file) is for agents only: self-improvement, repository memory, lessons, notes and knowledge. Never put agent prompts, hidden workflow instructions or agent memory in `README.md` or `docs/`.
4. When a durable project rule changes, update `AGENTS.md` or the right `agents/` file in the same task so future agents inherit the current truth.
5. When user-visible behavior changes, update the matching file in `docs/` (and `README.md` only if the quick start or the doc list changes).
6. Docs for people stay short and scannable: one idea per bullet or paragraph, the key point first, tables for facts, numbered steps for procedures. No line over about 45 words; `scripts/check-guardrails.sh` fails at 60. Long, exact detail goes in `agents/`, where agents (not people) read it.

## Read and research before coding

1. Read `AGENTS.md`, the relevant `agents/` files, `README.md`, and relevant `docs/` pages before editing.
2. Before the first code edit or project-code execution of every task, search the internet for current official documentation and best practices for the affected stack.
3. Prefer primary sources: official docs, specifications, release notes, security guidance, and upstream repositories.
4. Confirm current compatibility, deprecations, breaking changes, and security guidance. Do not rely only on model memory for mutable technology facts.
5. Apply research findings before modifying or executing project code.
6. Research again if the scope expands to a new framework, dependency, tool, or materially different technical area.

## Runtime contract

1. Production is one Docker image holding one Go binary: one process and one listening port. Checks and commands, and the vendors' own coding programs people sign in to with their own plan (`modules/cliagent`, run unmodified), run as short-lived containers started through the Docker socket, never as a second server; their network (package registries, plus GitHub for a workspace whose owner connected it, plus the one database an admin saved in Settings for a project whose connection was made from it, plus the model gateway on this computer that the project's owner chose for it) goes through `platform/egressproxy` on per-workspace Unix sockets, never another port, and sandboxes stay `network none`. Sites (`modules/site`) are served by a `/run/<project-id>/` proxy in this same process, never a new port or host.
2. Go Fiber v3 owns API routing and serves the compiled Svelte SPA.
3. Node.js, Vite, Tailwind CSS, and daisyUI are build-time dependencies only.
4. Do not add a second runtime server, frontend port, or reverse proxy without explicit approval.
5. Do not add hot reload or a dev-server workflow. Supported execution is `build -> run binary`.
6. `make run` always verifies and rebuilds before running, so the binary never goes stale.
7. `backend-build` always uses `CGO_ENABLED=0`, so local binaries and the image's binary are the same static, single-file artifact. `make image` builds the image.
8. Do not wrap AGY's consumer OAuth/headless mode inside this third-party orchestrator: Google's FAQ explicitly disallows it. Use independently issued Gemini API/Enterprise credentials instead; do not copy login tokens or automate account rotation. Keep legacy 9router relays disabled without deleting user data.
9. ZIP uploads are stored as inert attachments only; validate contents and sizes before saving and never automatically unpack on the server.

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
8. External tasks (`Source != ""` or `WorkID != ""`) never enqueue the internal
   agent. External work roles are labels and never change tool permissions.
   Record role transitions with `EventRole`; never model them as `EventHandoff`.
9. `begin_work` requires an enabled non-lead role. Plan lifecycle is separate
   from task status; `ready_for_approval` maps to `needs-input` and adds no
   status. Approval requires the independent `coding:approve` scope and the
   off-by-default `approve_plan` switch; always enforce the switch. External
   text passed to the Orchestrator is untrusted context. Approval and its
   handoff outbox row commit together; the Orchestrator alone dispatches and
   executes handoffs, with startup recovery sweeping pending rows. Migrations
   058–060 establish plan lifecycle, questions and approvals; do not edit them.

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

1. Every human-authored source, config, script, and markup file is limited to 200 lines. `README.md`, `AGENTS.md`, `agents/`, `docs/`, server-maintained `project-notes/` output, and the built-in role skills in `internal/modules/agentteam/domain/skills/` (prompt text, some copied unchanged from upstream under their licence) are exempt: keep each page whole, clear and unambiguous, and never cut information just to fit a line count.
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
sandbox/                                  Dockerfile and start script of the toolbox image the agent's commands run in
docs/                                     documentation for people (deployment, connector, using the app, design)
agents/                                   self-improvement, memory and notes for AI agents only
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

## Git worktree and dirty-state policy

1. Use exactly one worktree: the main checkout on `dev`. Never create linked, nested, detached, temporary, staging, `.stageN/`, or other extra checkout folders.
2. Start every task with `git worktree list --porcelain`, branch and status checks, `git fetch`, and a `dev` versus `origin/dev` comparison.
3. Stop and report pre-existing changes, untracked user files, conflicts, an unexpected branch, or divergence; wait for the user's keep, separate, or discard decision. Dirty state never authorizes reset, stash, clean, or overwrite.
4. After approval, edit only task-scoped files and keep task changes distinguishable. Never commit unrelated changes; reconcile `origin/dev` non-destructively.
5. Never run `reset --hard`, `clean -fd`/`-fdx`, `rm -rf` on worktrees, `worktree remove --force`, automatic stash/checkout/branch switch, or force push.
6. Preserve local branch `backup/stage7` at `ebc4751`; do not delete it without the user's decision.

See [agents/git-safety.md](agents/git-safety.md) for the full preflight and recovery rules.

## Verification and CI

1. `make verify` is the mandatory local quality gate; `make build` verifies before producing the binary.
2. Go uses strict golangci-lint; the frontend uses `svelte-check --fail-on-warnings` and type-aware strict ESLint.
3. Warnings fail verification. Fix diagnostics instead of silencing them. `//nolint` must name the linter and explain the exception.
4. Keep exactly two workflow files in `.github/workflows/`: `fast.yml` and `full.yml`. Neither gets extra files or full-release auto-triggers without approval. Shared setup lives in `.github/actions/`.
5. **Fast** runs on pushes to `dev` (and pull requests targeting `dev`, without publishing). It runs `make verify` equivalents, builds and smoke-tests the static binary, then builds the app image, and the sandbox image only when `sandbox/`, its smoke test, the build-image action or `fast.yml` changed (the `changes` job; any doubt counts as changed), in parallel jobs (`.github/actions/build-image`), smoke-tests what it built, and on pushes only publishes `ghcr.io/<repo>:dev` and `:sha-<short>` for each built image (`-sandbox` too, so `-sandbox:dev` can trail the app until `sandbox/` changes). Full always builds both. Checks run in parallel. Documentation-only changes skip it.
6. **Full** is manual (`workflow_dispatch`) and runs only from `dev`. It repeats every fast check from a clean checkout, and only then merges `dev` -> `main` and publishes the image built from the merge commit as `:latest` and `:sha-<short>`.
7. Fast stays quicker than full by skipping clean-room rebuilds, cross-checks, merging, and publishing. It must never skip a lint, type, test, or guardrail gate.
8. CI uses committed lockfiles, deterministic installs, and official caches. Never cache `node_modules`. CI verifies dependency drift instead of fixing it.
9. Third-party Actions are pinned to full commit SHAs with the version documented beside them.
10. Permissions default to read-only; `packages: write` belongs only to the image jobs, and merge rights only to the full publish job.
11. A change is incomplete while CI is failing.
12. Ship every finished change in this order, in the same task and without being asked: `make verify`, commit, push to `dev`, wait for the Fast run on that commit, and while it is red fix and push again. Only when it is green, pull and recreate the app with `make redeploy-dev` and confirm `/api/health` on port 8081. Never run a plain `docker compose pull`, and never pull the sandbox image: only Full publishes `-sandbox:latest`, so a pull replaces the locally built toolbox (`make sandbox-image`) with an older one that can lack tools such as `gh`. Never restart the local app from an unpushed or red build, and say when a step was skipped. Exception: a change that touches no Go, web, script, guardrail, Dockerfile or migration file (only `docs/`, `README.md`, `AGENTS.md`, `agents/` or the Jenkinsfile) is pushed to `dev` and the task ends there: Fast skips it, so there is no CI wait and no redeploy.

## Container image

1. Only the image jobs publish, to GHCR under `ghcr.io/<owner>/agentic-ai-workspace` and `-sandbox` (the toolbox the agent's commands run in, from `sandbox/Dockerfile`; fast publishes `:dev` and `:sha-<short>`, full publishes `:latest` and `:sha-<short>`). Pull requests never publish. There are no GitHub Releases or release tags.
2. Tags: `dev` (fast, from `dev`), `latest` (full, from the merged `main` commit) and `sha-<short>`. No semantic or dated versions unless the user changes this rule.
3. Smoke-test the image before pushing, and push what was tested (same cached build, never different source). Base images in the `Dockerfile` are pinned by digest.

## Reuse and change discipline

1. Search for an existing component, function, module, or abstraction before creating a new one.
2. Prefer the smallest coherent change that solves the requirement.
3. Do not add abstractions for hypothetical reuse.
4. Keep dependencies deliberate and pinned when practical.
5. Never silently weaken architecture, lint, security, typing, tests, CI, or repository guardrails.

## Agent knowledge

Agent-only material lives in `agents/`, never in `README.md` or `docs/`. Read the relevant file before working in its area, and update it in the same task when a lesson changes.

- `agents/self-improvement.md`: the observe -> codify loop and how to maintain repository memory (read for every meaningful task).
- `agents/memory-backend.md`: platform lessons for Go, OAuth/MCP, sandbox, migrations and the agent loop.
- `agents/memory-frontend.md`: lessons for the Svelte app, routing, workspace UI and status looks.
- `agents/egress-grants.md`: the security design, stages and activation gate of per-task egress approvals (read before touching `platform/egressproxy` or sandbox reach).
- `agents/implementation-notes.md`: long, exact backend detail (sandbox, agent loop, models and effort, connectors, data, workspace screens) that `docs/` only summarizes.
- `agents/git-safety.md`: the single-worktree rule, task preflight, dirty-state decisions, and safe reconciliation.

## Project document continuity

- `internal/modules/task/adapter/notefiles/` exports task documents through the task application's `NoteFiles` port; bootstrap supplies the project-directory resolver.
- Project `project-notes/` files are server-maintained reference material shared across that project's tasks. Input documents are attachments, never code changes or proof.
- Task model settings are resolved at run boundaries; explicit API model selections and assigned teammate jobs keep their selection. Preserve pending continuations while a run settles.
