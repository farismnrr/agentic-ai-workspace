# Testing rules and workflow

## Choose the test boundary

1. Start with the rule in `AGENTS.md`: do not add non-critical white-box unit tests.
2. For features, refactors, and fixes, prefer Bash scenarios against the built binary, real HTTP/MCP endpoints, Docker, or browser flows.
3. Reuse `test/lib/`, fixtures, and existing scenarios before adding helpers.
4. Test a meaningful outcome and a failure path. Do not duplicate an existing guarantee or add tests to raise coverage.
5. Keep a Go white-box test only when the behavior is super-critical and cannot be triggered deterministically at a public boundary. Add its exact `path::TestName`, category, reason, and Bash counterpart to `test/whitebox/registry.md` in the same change.
6. Never remove security or data-integrity verification without an equal or stronger named replacement.

## Add a Bash scenario

- Put runnable scenarios in `test/blackbox/`, `test/smoke/`, or `test/e2e/` by purpose.
- Name black-box scenarios lowercase kebab-case with a two-digit run prefix
  and `.test.sh`; browser flows use `.spec.ts`.
- Smoke scripts are named by subject because suites select them by role;
  numbered prefixes are reserved for ordered scenarios.
- Portable smoke scripts may use `#!/bin/sh` with `set -eu`; Bash scenarios
  use `#!/usr/bin/env bash` with `set -euo pipefail`.
- Source shared helpers from `test/lib/*.sh`; those files are sourced, not run directly.
- Use the strict mode for the script's shell, disposable data, and cleanup
  traps. Never use the live database or `.env`.
- Run one scenario directly while developing, then invoke it through `test/run.sh` or its Make target.
- Keep Playwright TypeScript flows under `test/e2e/` and start them via `test/e2e/run.sh`.

## Registry enforcement

`test/whitebox/check-registry.sh` defaults to enforcement and validates every
explicit registry row, including named bootstrap tests. Enforcement checks
every non-bootstrap `*_test.go` file and every `Test*` function. Test files
need a named test row or an entry in the support-file list. Bootstrap remains
the registered integration group.

Follow-up: replace the site and thread browser-host Go fixtures with an
API-backed model fixture.
