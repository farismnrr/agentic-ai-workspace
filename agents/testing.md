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

## Registry enforcement

`test/go/check-registry.sh` defaults to enforcement and validates every
explicit registry row. It checks every Go test file and every `Test*`
function under `test/go/`. Test files need a named test row or an entry in the
support-file list. App-level integration tests are listed too.

Follow-up: replace the site and thread browser-host Go fixtures with an
API-backed model fixture.
