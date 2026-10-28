# Testing rules and workflow

## Choose the test boundary

1. Start with the rule in `AGENTS.md`: do not add non-critical white-box unit tests.
2. Put every test under `test/`; never put a Go test beside program code. The untouched `sandbox/starter/` template is the sole exception.
3. For features, refactors, and fixes, prefer Bash scenarios against the built binary, real HTTP/MCP endpoints, Docker, or browser flows.
4. Reuse `test/lib/`, fixtures, and existing scenarios before adding helpers.
5. Test a meaningful outcome and a failure path. Do not duplicate an existing guarantee or add tests to raise coverage.
6. Keep a Go white-box test only when the behavior is super-critical and cannot be triggered deterministically at a public boundary. Put it under `test/go/` and add its exact `path::TestName`, category, reason, and Bash counterpart to `test/go/registry.md` in the same change.
7. Never remove security or data-integrity verification without an equal or stronger named replacement.

## CI lanes

- Fast runs `make guardrails shellcheck` and `make backend-test` before image builds.
- Full runs `make verify-full`, which adds frontend checks, Go lint, build and smoke, black-box, browser, Docker isolation, and race checks.

## Impact-based task verification

- Assess impact from changed behavior, dependencies, shared contracts and configuration; extensions alone are not enough. Ordinary Swift work runs affected public-boundary scenarios and relevant static checks; do not require every suite or review role by default.
- Sensitive changes need focused Security review and their named critical scenarios in either mode.
- Deep-review adds an independent Reviewer assessment and broader applicable regression, static, dependency and security checks. Add Architect when contracts or module boundaries are affected.
- Unknown paths, ambiguous behavior or unclear dependencies broaden conservatively and must be explained in the lead's report.
- Every task completion path, including a direct model answer, CLI result and finish tool, must enforce the same gate. Require the latest applicable review and every named critical check; a PASS label without evidence is insufficient.
- A Security PASS must include an `Assessment:` naming every affected path and explaining how the scenarios cover it, plus `Checks:` with one exact full command per required scenario. Each distinct command needs its own latest matching passing server-recorded run after the latest change. Listing, collection, compile-only and no-test commands are not meaningful checks. Evidence runs require known effects and no changed paths.
- A missing mandatory role, missing or failed evidence, or exhausted review limit blocks completion as unverified. Broad Make targets remain available when requested or indicated by impact.
- The Full release merge must match the verified `GITHUB_SHA`. Build from its returned merge SHA and publish a tag from that SHA for deployment; if `dev` moved, fail and verify again.
- `scripts/check-guardrails.sh` exempts `test/` and `scripts/` from its file-size and content rules. It still enforces test placement, `testdata/` placement, and the ban on program imports from `test/`.
- ShellCheck, golangci-lint, and ESLint remain separate checks and still analyze test and script code in their configured scopes.

## Add a Bash scenario

- Put runnable scenarios in `test/blackbox/`, `test/smoke/`, or `test/e2e/` by purpose.
- Name black-box scenarios lowercase kebab-case with a two-digit run prefix
  and `.test.sh`; browser flows use `.spec.ts`.
- Smoke scripts are named by subject because suites select them by role;
  numbered prefixes are reserved for ordered scenarios.
- Use subject names for tests and files; do not use numbered process-history
  labels.
- Portable smoke scripts may use `#!/bin/sh` with `set -eu`; Bash scenarios
  use `#!/usr/bin/env bash` with `set -euo pipefail`.
- Source shared helpers from `test/lib/*.sh`; those files are sourced, not run directly.
- Use the strict mode for the script's shell, disposable data, and cleanup
  traps. Never use the live database or `.env`.
- Run one scenario directly while developing, then invoke it through `test/run.sh` or its Make target.
- Keep Playwright TypeScript flows under `test/e2e/` and start them via `test/e2e/run.sh`.

The external work lifecycle has black-box coverage in
`test/blackbox/07-external-work-lifecycle.test.sh` and a Chromium flow in
`test/e2e/04-work-lifecycle.spec.ts`. The flow seeds work over MCP, checks the
running, reported, idle and confirmed states, and needs no Docker. Run it with
`make e2e FLOW=lifecycle`; the default browser suite includes it.

## Registry enforcement

`test/go/check-registry.sh` defaults to enforcement and validates every
explicit registry row. It checks every Go test file and every `Test*`
function under `test/go/`. Test files need a named test row or an entry in the
support-file list. App-level integration tests are listed too.

Follow-up: replace the site and thread browser-host Go fixtures with an
API-backed model fixture.
