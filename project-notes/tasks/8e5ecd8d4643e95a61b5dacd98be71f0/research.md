# # Refactor Testing Strategy — Black-box First, Minimal White…

Task: 8e5ecd8d4643e95a61b5dacd98be71f0
Status: done
Updated: 2026-10-28T02:16:53Z

Reference material; confirm applicability before reusing.

# Research report: testing refactor

**The repository has 365 Go test files with 993 top-level test, benchmark, or fuzz functions; the test bodies were not run.**  
**The strongest existing black-box foundations are the app-level HTTP/MCP tests and three Playwright flows; CI does not currently run those browser flows.**

## Inventory

Counts below are Go test files / top-level `Test*`, `Benchmark*`, or `Fuzz*` functions, grouped by module and package family. They come from source-file and declaration counts; the number of subtests inside functions is not counted.

| Module / package family | Files / functions | Style and criticality |
|---|---:|---|
| `internal/bootstrap` | 62 / 144 | Mixed; app-level HTTP/MCP tests with temporary app state, alongside fakes. Critical: auth, isolation, egress, approvals. |
| `agentteam` domain; `cliagent` application/domain | 6 / 29 | Mostly white-box; non-critical to listed security boundaries. |
| `coding` adapter/application/domain | 44 / 117 | Mixed: MCP integration and container-backed sandbox checks plus package tests. Critical: tool permissions, workspace isolation, sandbox limits. |
| `dbconn` adapter/application/domain | 27 / 82 | Mixed: HTTP and SQLite tests plus fakes. Critical: credentials, project database scope, persistence. |
| `egressgrant` adapter/application/domain | 18 / 63 | Mostly SQLite/service tests; bootstrap adds app-flow coverage. Critical: approval scope, proxy grants, races and revocation. |
| `env` adapter/application/domain; `extension` adapter/domain; `gateway` application/domain | 7 / 23 | Mostly white-box. Env secrets have security relevance. |
| `github` adapter/application/domain | 8 / 24 | Mostly adapter/service tests. Critical: user and project credential isolation. |
| `oauth` adapter/domain | 5 / 16 | Protocol and package tests. Critical: token binding, replay and revocation. |
| `project` application/domain; `provider` adapter/application/domain | 11 / 29 | Mostly package-level tests; project ownership is critical. |
| `site` adapter/application/domain | 27 / 88 | Mixed proxy HTTP tests and package tests. Critical: private-site access, cookie isolation and browser sandboxing. |
| `task` adapter/application/domain | 111 / 262 | Mostly package and SQLite tests. Critical: transactions, ownership, concurrency, task/handoff state and data loss. |
| `web` adapter/application/domain | 6 / 17 | Mostly package tests; generally non-critical here. |
| `platform/config`, `database`, `egress`, `egressproxy`, `healthcheck`, `httpserver`, `instancelock`, `mcpbridge`, `sandbox`, `secretbox` | 32 / 92 | Mostly direct package tests. Critical subsets include migration integrity, SSRF/egress, sandbox privilege boundaries and encryption. |

The frontend has **24 first-party Vitest files and 77 `it`/`test` calls**, all under `web/src/`; these exercise frontend logic directly. I excluded dependency tests under `web/node_modules/`. The three Playwright scripts are [`scripts/e2e-projects.ts`](/workspace/scripts/e2e-projects.ts), [`scripts/e2e-sites.ts`](/workspace/scripts/e2e-sites.ts), and [`scripts/e2e-thread.ts`](/workspace/scripts/e2e-thread.ts).

## Reusable black-box infrastructure

- [`internal/bootstrap/harness_test.go`](/workspace/internal/bootstrap/harness_test.go) builds the Fiber app with a temporary data directory. [`internal/bootstrap/request_harness_test.go`](/workspace/internal/bootstrap/request_harness_test.go) and [`internal/bootstrap/e2e_test.go`](/workspace/internal/bootstrap/e2e_test.go) issue HTTP requests against it.
- Bootstrap tests include MCP calls over the app route; `e2e_test.go` defines MCP headers and checks token scope and tool access. This is app-level protocol coverage, but it runs in-process rather than launching the built executable.
- The browser scripts use Go test functions as temporary app servers. `TestServeThreadForABrowser` and `TestServeSiteForABrowser` write connection details to `info.json`; shell scripts start the test process, wait for that file, then invoke Playwright. See [`internal/bootstrap/thread_browser_test.go`](/workspace/internal/bootstrap/thread_browser_test.go), [`internal/bootstrap/sites_browser_test.go`](/workspace/internal/bootstrap/sites_browser_test.go), and `scripts/e2e-*.sh`.
- [`scripts/smoke-test.sh`](/workspace/scripts/smoke-test.sh) covers the built binary; separate image smoke scripts exist. These are lifecycle/smoke checks, not a general disposable-state API/MCP harness.
- Gaps for a small standalone harness: launch the compiled app as a child process with isolated disposable state; expose its HTTP and MCP endpoints to black-box clients; provide controlled fixture/cleanup and readiness; and make Playwright setup/run/reporting a repeatable CI target. Current browser scripts rely on globally installed Playwright and Docker, and the app is hosted by a Go test rather than the executable.

## Super-critical white-box tests to consider retaining

These are candidates for a *minimal* retained set. Some could later be replaced if the new harness can reliably induce the same failure modes.

| Candidate | Why black-box coverage alone may miss it |
|---|---|
| [`internal/platform/secretbox/cipher_test.go`](/workspace/internal/platform/secretbox/cipher_test.go): `TestSealedValuesDifferAndTamperingFails`, `TestRoundTripAndKeyPersistsAcrossRestarts` | Tests ciphertext tampering and key persistence at the encryption boundary; a normal application flow may only observe successful secret use. |
| [`internal/platform/egress/egress_test.go`](/workspace/internal/platform/egress/egress_test.go): `TestPrivateAllowedDialControlRejectsResolvedMetadataAddress`, `TestTransportReallyRefusesToConnect` | Exercises DNS resolution/dial-time SSRF defenses, which are difficult to trigger deterministically through ordinary app requests. |
| [`internal/platform/egressproxy/grantpath_test.go`](/workspace/internal/platform/egressproxy/grantpath_test.go): `TestADNSAnswerThatChangesAfterTheCheckChangesNothing`; [`grants_race_test.go`](/workspace/internal/platform/egressproxy/grants_race_test.go): `TestRevokeRacingNewTunnelsLeavesNoneOpen` | Covers DNS rebinding and a revoke/open-tunnel race at the proxy boundary; retain unless the harness can control DNS and tunnel timing. |
| [`internal/modules/task/adapter/sqlite/work_store_test.go`](/workspace/internal/modules/task/adapter/sqlite/work_store_test.go): rollback-on-event-failure and owner-isolation cases; [`approval_store_concurrent_test.go`](/workspace/internal/modules/task/adapter/sqlite/approval_store_concurrent_test.go): `TestConcurrentApprovalCreatesOneDecisionAndHandoff` | Forced persistence failures and concurrent commits verify atomicity/idempotency that a successful end-to-end path may not expose. |
| [`internal/modules/coding/adapter/runner/sandbox_test.go`](/workspace/internal/modules/coding/adapter/runner/sandbox_test.go): `TestTaskHasNoNetworkAndNoPrivileges`, `TestOnlyTheWorkspaceIsWritable` | Directly verifies operating-system/container restrictions; application-level outputs alone do not prove those restrictions held. |

**Ambiguous:** OAuth has useful app-level protocol coverage in [`internal/bootstrap/e2e_test.go`](/workspace/internal/bootstrap/e2e_test.go) and [`internal/modules/oauth/adapter/sqlite/flow_cases_test.go`](/workspace/internal/modules/oauth/adapter/sqlite/flow_cases_test.go). Decide based on whether the replacement harness exercises authorization, resource-bound access, refresh reuse, code replay and revocation. Site proxy tests also check security-sensitive headers at the adapter boundary; the existing browser test covers a private-site flow, but not necessarily every hostile-header and cookie case.

## CI, targets and current timing evidence

- [`Makefile`](/workspace/Makefile): `make verify` runs guardrails, frontend check/lint/Vitest/build, `go mod tidy -diff`, `go test ./cmd/... ./internal/...`, and Go lint. `make build` verifies first. `make e2e-sites`, `make e2e-thread`, and `make e2e-projects` are separate targets and are not part of `verify`.
- [Fast workflow](/workspace/.github/workflows/fast.yml) runs guardrails, frontend, Go tests and Go lint in parallel; then builds and smoke-tests the static binary, followed by app and conditional sandbox image builds/smoke tests. It publishes on pushes to `dev`; pull requests do not publish. Browser E2E is absent.
- [Full workflow](/workspace/.github/workflows/full.yml) runs `make verify`, builds and smoke-tests the binary, then merges `dev` to `main`, builds and smoke-tests app and sandbox images, deploys, and checks health.
- **No elapsed CI timings were found.** Workflow `timeout-minutes` values are ceilings, not measured durations: Fast jobs specify 5, 10 or 15 minutes; Full jobs specify 15 or 20 minutes.
- Go tests use no visible `-race`, `-short`, or build-tag flags in the Makefile/workflows. Go setup enables module/build caching and keys dependencies from `go.sum` in [`.github/actions/setup-go/action.yml`](/workspace/.github/actions/setup-go/action.yml).

## Official guidance checked

- Go’s official [`go` command test, cache and flag documentation](https://pkg.go.dev/cmd/go#hdr-Test_packages) describes test result caching, `-race`, `-short`, build tags and package parallelism. The [`race detector guide`](https://go.dev/doc/articles/race_detector) explains its runtime instrumentation and supported-platform constraints. GitHub’s official [`setup-go` action documentation](https://github.com/actions/setup-go) documents caching Go modules/build outputs and dependency-file cache keys.
- Playwright’s [CI guide](https://playwright.dev/docs/ci) recommends one worker in CI for stability, a global test timeout, and sharding across jobs for wider parallelism. It says browser-binary caching is generally not recommended because restore time can approach download time.

## Not found

- Actual CI elapsed times or a recorded timing baseline.
- A standalone executable-based disposable-state HTTP/MCP test harness.
- A Playwright job in either GitHub Actions workflow.

## Suggestions

- Use bootstrap app tests as the contract map for the new harness. Separate black-box scenarios from package tests deliberately; module-level counts do not identify every test’s true boundary.
- Initially retain the five candidate groups above, then remove a white-box case only when a repeatable harness scenario covers its failure mode.
- Treat the egress, OAuth, task transaction and site groups as planner review points because their tests mix externally visible behavior with invariants that are hard to force through normal requests.

**Run record:** Read-only file inspection, counting scripts and official documentation fetches only. No Go, frontend or Playwright test suite was run. The attempted `git fetch` failed because `.git/FETCH_HEAD` was read-only; the pre-existing deleted `project-notes/` files and untracked task directory were left untouched.

---

# Research report: testing refactor references

The repo is on Go **1.27.2**; the official release history lists **Go 1.27.3**, released 2026-10-19, as current on 2026-10-27.  
The current workflow runs independent quality jobs before the binary build; Playwright is not a declared web dependency today.

## Technical references

- **Go child-process harness:** Use `exec.CommandContext` to start the built app, wait for `/api/health` with a deadline, and register shutdown and temp-directory removal with `t.Cleanup`. Go’s own test support uses subprocesses and cleanup to detect leaks. For `TestMain`, build once before `m.Run`; call `os.Exit` with its result. Go documents `TestMain` setup/teardown, but does not prescribe this specific build-once harness. A port obtained by listening on `127.0.0.1:0` and then closing the listener has a race before the app binds it. [Go testing source](https://go.dev/src/testing/testing.go), [Go subprocess test helper](https://go.dev/src/internal/testenv/exec.go). In this repo, [`scripts/smoke-test.sh`](/workspace/scripts/smoke-test.sh) already passes `PORT` and `DATA_DIR` and polls `/api/health`; [`internal/bootstrap/harness_test.go`](/workspace/internal/bootstrap/harness_test.go) is an existing in-process harness.

- **Go version and test flags:** [`go.mod`](/workspace/go.mod) requires Go 1.27.2. Official release history lists Go 1.27.3 on 2026-10-19, including a `crypto/x509` security fix; consider updating the pinned toolchain separately. Official `go test` docs: `-count=1` disables cached test results; `-race` enables race detection on supported platforms; `-shuffle=on` or a numeric seed randomizes test order; `-p` limits package-level parallelism; `-parallel` limits parallel tests inside each test binary. [Go release history](https://go.dev/doc/devel/release), [Go test flags](https://pkg.go.dev/cmd/go#hdr-Testing_flags), [race detector](https://go.dev/doc/articles/race_detector).

- **Lint and Actions:** [`fast.yml`](/workspace/.github/workflows/fast.yml) already runs guardrails, frontend, Go tests, and Go lint as separate jobs; the binary build depends on them. [`setup-go/action.yml`](/workspace/.github/actions/setup-go/action.yml) enables Go module/build caching keyed from `go.sum`; the lint job uses the golangci-lint action, which caches its analysis separately. GitHub says jobs run in parallel unless linked by `needs`; use artifacts to pass built outputs between jobs and cache reusable dependencies. Treat restored caches as untrusted and do not cache secrets. [golangci-lint action cache guidance](https://github.com/golangci/golangci-lint-action), [Actions workflow syntax](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax), [Actions caching guidance](https://docs.github.com/en/actions/concepts/workflows-and-actions/dependency-caching).

- **Playwright:** Official CI guidance installs dependencies with `npm ci` and browsers with `npx playwright install --with-deps chromium`; install only Chromium when that is the only browser needed. Playwright recommends one worker for CI stability, with sharding across jobs when needed. Its release notes currently show v1.63; pin an exact package version in the manifest and lockfile so the matching browser revision is installed. The repo currently has no Playwright package pin and its E2E scripts expect a globally installed package. [Playwright CI guide](https://playwright.dev/docs/ci), [browser installation](https://playwright.dev/docs/browsers), [release notes](https://playwright.dev/docs/release-notes).

- **Removing Vitest:** [`web/package.json`](/workspace/web/package.json) has a `test` script, a `verify` script that invokes it, and Vitest 5.0.5 as a dev dependency. Remove or update both scripts and remove the dependency; npm’s uninstall command updates `package.json` and `package-lock.json`. Then run the repository’s locked install/check workflow. [npm uninstall docs](https://docs.npmjs.com/cli/v11/commands/npm-uninstall/). The current `Makefile` calls `npm test` through `frontend-verify`, so that target also needs to stop invoking the removed script.

- **Fast timing:** `gh run list --branch dev --workflow Fast --status success --json databaseId,headSha,startedAt,updatedAt` can identify recent successful runs. `gh run view <id> --json jobs` provides job data; calculate each job’s elapsed time from its start and completion timestamps. The CLI docs list the run JSON fields; GitHub’s workflow-jobs API documents `started_at` and `completed_at`. [gh run list](https://cli.github.com/manual/gh_run_list), [gh run view](https://cli.github.com/manual/gh_run_view), [workflow-jobs API](https://docs.github.com/en/rest/actions/workflow-jobs).

## Quality checks

- Compare the approved keep/delete inventory against the actual test declarations after edits. Confirm every retained white-box test has a registry row, and that the guardrail rejects an unregistered white-box test. This is a proposed QA check based on the PRD’s registry and guardrail requirements.
- For each security or integrity deletion, verify its named replacement exercises the same failure condition. In particular, check cross-user access denial, token scope denial, secret redaction, restart persistence, and failure/rollback behavior; a successful happy path alone does not establish these.
- Do mutation-style spot checks on representative boundaries: temporarily weaken or bypass an authorization, isolation, or secret-filtering condition and confirm its replacement test fails. Restore the change before review. This is a suggested check, not something performed here.
- Run the approved suite with `-count=1`; use `-race` for the retained concurrency checks and a fixed `-shuffle` seed when diagnosing order-dependent failures. These flags complement coverage review; they do not prove deleted checks were replaced.

## Not found

- An official Go recipe specifying the exact combination of free-port selection, binary build-once `TestMain`, readiness polling, and cleanup. The pattern above combines documented Go primitives with the project’s existing smoke-test behavior.
- A Playwright version currently pinned for this repo.
- Measured Fast job durations. The workflow’s `timeout-minutes` values are ceilings, not elapsed times.

## Suggestions

Use the current smoke script as a lifecycle reference, but account for the free-port handoff race. Pin Playwright exactly before browser installation. Make the test inventory-to-registry comparison and representative mutation checks review gates before deleting security-sensitive tests.

**Run record:** Read project notes, repository files, and official documentation. No files were changed; no tests or project commands were run.

---

**Report**

The repo pins golangci-lint **v2.14.0**; `.golangci.yml` enables `unused` and sets `run.tests: true`.  
For the deleted-test cleanup, remove helpers that have no remaining callers and format the touched Go files.

- **Observed in the project:** `.golangci-lint-version` contains `v2.14.0`; `.golangci.yml` enables `unused` and includes test files. The linter checks for unused constants, variables, functions and types. Its `run.tests` setting controls whether test files are included; the documented default is `true`. [Linter docs](https://golangci-lint.run/docs/linters/), [configuration docs](https://golangci-lint.run/docs/configuration/file/)

- **Version notes:** v2.14.0 was released 2026-09-24. Its changelog lists changes to several linters, but no `unused` change or deprecation. The current changelog also shows v2.15.0 released 2026-10-16; the repo remains pinned to v2.14.0. [Changelog](https://golangci-lint.run/docs/product/changelog/)

- **Recommended resolution:** When deleting tests leaves test-only helpers unused, delete the dead helpers and imports. Avoid `nolint` for code that no longer serves a purpose. Go’s guidance says unused imports make a package fail to compile; `go test` discovers test functions in `_test.go` files. `go vet` checks suspicious constructs, while formatting is handled by `gofmt`; neither substitutes for the `unused` linter. [Effective Go](https://go.dev/doc/effective_go), [Go test FAQ](https://go.dev/doc/faq), [vet docs](https://pkg.go.dev/cmd/vet)

**Not found:** An official `unused`-specific rule explaining treatment of helpers in `_test.go` beyond its general unused-symbol description.

**Suggestions:** Keep `tests: true` so the linter continues analyzing test files. Remove helpers and imports made obsolete by test deletion, then rely on the pinned lint run and Go compilation to identify remaining issues.

**Run record:** Read project instructions and config, searched and opened official documentation, and attempted the mandated `git fetch`; it failed because `.git/FETCH_HEAD` is read-only. No files were changed and no tests or lint commands were run. The worktree had extensive existing modifications and deletions, which I left untouched.

---

# Research report: script and history-name inventory

**The repo already has `test/` scenarios and a registered Docker Subpath test, but the manual Docker runner still checks additional behaviors.**  
**I found “Stage 7” references in the runner, its Go test, build wiring, workflow, docs, and registry; the protected `backup/stage7` branch is explicitly out of scope.**

## `scripts/` inventory

| File | What it does and callers | Verdict |
| --- | --- | --- |
| `scripts/deploy-binary.sh` | Hot-swaps the binary in a running container and rolls back on failure. Called by `Jenkinsfile`; also documented in `docs/devops.md` and `agents/memory-backend.md`. | **KEEP** — deployment tooling. |
| `scripts/install.sh` | Prompts for settings, writes `.env`, fetches Compose, starts the app, and checks health. Linked by `README.md` and `docs/deployment.md`; it also references itself in its generated `.env` comment. | **KEEP** — installer. |
| `scripts/check-guardrails.sh` | Checks source limits, frontend import rules, and the white-box registry. Called by the Makefile’s guardrails target and linked from `test/README.md`. | **KEEP** — guardrail tooling. |
| `scripts/shots.sh` | Starts a throwaway built app, registers a demo user/project, and captures screenshots. Calls `scripts/shots/capture-auth.ts`; `agents/memory-frontend.md` documents its use. | **KEEP** — screenshot tooling, not a test suite. |
| `scripts/shots/capture-auth.ts` | Uses Playwright/CDP to capture authenticated pages. Called by `scripts/shots.sh`. | **KEEP** — screenshot support. |
| `scripts/style-preview.html` | Static page template used to render style previews. Read by `scripts/style-previews.ts`. | **KEEP** — generator input. |
| `scripts/style-previews.ts` | Generates images for style families. Its usage is in its header and `agents/memory-frontend.md`; `web/src/features/workspace/assets/style-previews.ts` loads the resulting images. | **KEEP** — asset generation tooling. |
| `scripts/stage7-docker.sh` | Manual Docker gate for volume Subpath isolation, egress socket isolation, vendor containers, race/migration checks, container health and restart, and HTTP grant approval/revocation. It calls `scripts/stage7-http.py`. The Makefile’s `volume-isolation` target overlaps only the Subpath test; `test/smoke/sandbox.test.sh` checks general toolbox capabilities, not this grant/socket behavior. | **MOVE** to `test/smoke/volume-isolation.test.sh` (or a similarly subject-named smoke script). It contains tests and has unique assertions; do not delete it as redundant based on the existing checks. |
| `scripts/stage7-http.py` | Python HTTP scenario for anonymous denial, request/approve/revoke/replay, and restoring an approved grant after restart. It also writes task rows and inspects the SQLite migration journal directly. Called only by `scripts/stage7-docker.sh`. | **DELETE after replacement** with Bash in `test/`. Its checks are not fully redundant: `test/blackbox/05-restart-keeps-data.test.sh` tests app data persistence, but not restoration of an approved egress grant. The task asks for Bash tests, so preserve the behavior in a Bash scenario before removing the Python file. |

The three existing `test/blackbox/` scenarios most relevant to the HTTP script are `02-cross-user-isolation.test.sh`, `03-mcp-token-scope.test.sh`, and `05-restart-keeps-data.test.sh`. None is named as covering grant lifecycle or post-restart grant restoration. The registry’s `sandbox.test.sh` counterpart for the Subpath Go test is not the same as the manual runner’s egress socket, vendor, or grant lifecycle assertions.

## History-style names found

The repository-wide scan (excluding `project-notes/`, `test-approval-list.md`, Git metadata, and dependencies) found these Stage 7 references:

- **Runner and evidence:** `scripts/stage7-docker.sh` uses `stage7-evidence/`, `aw-stage7-*` volume/container names, `STAGE7_VOLUME`, `STAGE7_IMAGE`, and the `TestStage7...` test name. `scripts/stage7-http.py` uses `stage7-task-*`, `STAGE7_PROJECT`, `STAGE7_USER`, and Stage 7 dummy account/project labels.
- **Test and wiring:** `internal/platform/sandbox/subpath_integration_test.go` contains `TestStage7ActualVolumeSubpathsAreIsolated` and reads `STAGE7_VOLUME` and `STAGE7_IMAGE`. `Makefile` target `volume-isolation` uses `stage7_image`, `agentic-stage7-*`, those environment variables, and that test name. `.github/workflows/full.yml` sets `STAGE7_IMAGE`. `test/whitebox/registry.md` records the test name.
- **Documentation:** `docs/deployment.md` says the Stage 7 Docker checks passed. `agents/egress-grants.md` describes implementation stages 1–6 and the Stage 7 verification gate.
- **Other filename candidates:** `internal/bootstrap/v2_test.go` and `internal/modules/task/domain/plan_phase.go` appear in the filename scan. `PlanPhase` is a current plan lifecycle concept, not process history. `v2_test.go` is a history-style filename; I did not find a Stage-number reference associated with it.
- **Ordinary words excluded:** occurrences such as Go’s `new`, HTTP payload `old_text`/`new_text`, and prose “final” or “next step” are ordinary language/code, not process-history names.

**Do not rename without the person’s say:**

- `backup/stage7` at `ebc4751` is protected by `AGENTS.md` and `agents/git-safety.md`; leave the branch name untouched.
- “Stages” in `agents/egress-grants.md` mostly label real implementation and verification milestones, including an activation gate and deferred compatibility work. They are not simple test filename suffixes. Recommend leaving those design references intact unless the person wants a broader terminology cleanup; update the Stage 7 runner/test terminology independently.

## Naming consistency

`agents/testing.md` says runnable scenarios belong under `test/`, shell scenarios use lowercase kebab-case with `.test.sh`, Playwright flows use `.spec.ts`, and smoke scripts are subject-named. The current `test/` files follow those forms. The two `scripts/stage7-*` files are the clear mismatch: they are test-oriented names outside `test/`, one is Python despite the requested Bash convention, and both carry history names. The style-preview pair is intentionally a template and generator pair; its names are consistent with their roles.

## `go test -run` strings and registry references

The stage runner and Makefile contain these `-run` expressions:

- `^TestStage7ActualVolumeSubpathsAreIsolated$` — `scripts/stage7-docker.sh` and `Makefile`.
- `^Test(AGrantIsUsableOnlyFromItsOwnTasksContainer|TheContainerSeesOnlyItsOwnSocketAndCannotChangeIt|Stage7RealPublicHTTPSNeedsTaskGrant|AVendorProgramContainerGetsOnlyItsOwnTasksSocket|AVendorProgramForATaskThatIsOverDoesNotStartAndFreesItsPlace|WithoutAScopeAndForSitesTheWorkspaceSocketIsUsedAndCarriesNoGrants)$` — `scripts/stage7-docker.sh`.
- `^Test(RevokeRacingNewTunnelsLeavesNoneOpen|ApprovalRevocationAndSweepsRacingLeaveTheRegistryEqualToTheDatabase|ConcurrentSweepsAndTaskEndNeverResurrectAGrant|ADatabaseFailureRefusesConnectionsAndTheSweepEmptiesTheRegistry|AnExistingDatabaseTakes052WithoutTouchingTheJournal|ExistingAuditMigrationGetsAppendOnlyActorAttribution)$` — `scripts/stage7-docker.sh`.
- Dynamic `go test ... -run "$host_test"` in `test/e2e/run.sh` selects `TestServeSiteForABrowser` or `TestServeThreadForABrowser`; these are not Stage 7 names.
- Registry row: `internal/platform/sandbox/subpath_integration_test.go::TestStage7ActualVolumeSubpathsAreIsolated`.

**Not found:** I did not find another `stageN`/`STAGEN` identifier family or numbered phase/step filename family outside the candidates above. The supplied exclusions were not searched.

**Suggestions:** If renaming the Stage 7 check, rename its runner, environment variables, Docker names, evidence directory, Go identifier, Makefile/workflow wiring, docs, and registry row together. Keep the socket-isolation, vendor-container, race/migration, grant lifecycle, and restart-restoration checks accounted for by name. If the inventory is intended to cover *all* legacy vocabulary such as `v2`, `final`, `new`, and `old`, define which occurrences count as history labels before a broad rename.

**Run record:** Read project instructions, indexed task notes, testing guidance, and the named files; searched repository references. `git fetch` failed because `.git/FETCH_HEAD` was read-only. No tests or scripts were run and no files were changed. The existing deleted `project-notes/` files and untracked task files were left untouched.

---

### Summary

**Observed:** `make shellcheck` runs `shellcheck -x` on scripts from the repository root. The smoke script’s `source=` annotations point to `../lib/...`, but its helpers are under `test/lib/`. `APP_TMP_DIR` is set in the smoke script and read by `test/lib/grants.sh`.

**Recommendation:** Make the source annotations resolve the actual `test/lib` helpers, then suppress SC2034 on the assignment with a comment explaining that the sourced helper consumes it.

### Findings

- ShellCheck’s SC1091 guidance says `-x` allows following source files not listed as command-line inputs, and a `source=` directive identifies the file to analyze. `-x` does not fix an incorrect path. [SC1091 wiki](https://github.com/koalaman/shellcheck/wiki/SC1091)
- ShellCheck’s `source=` paths are resolved using its source search paths. The special `SCRIPTDIR` means the checked script’s directory; `source-path=SCRIPTDIR/../lib` would resolve to `test/lib` for this smoke script. `--source-path` (or `-P`) adds search directories and takes precedence over source-path directives. [ShellCheck manual: directives and source paths](https://github.com/koalaman/shellcheck/blob/master/shellcheck.1.md)
- Here, the directive paths `../lib/assert.sh`, `../lib/http.sh`, and `../lib/grants.sh` are wrong: from `test/smoke/`, they point to `test/lib/` only if normalized as one directory up—actually `test/smoke/../lib/` does resolve to `test/lib`. The reported missing path therefore appears tied to how the invocation resolves directive paths from the working directory. The Makefile invokes ShellCheck at the repository root, where those relative paths point to `../lib/`, outside the repo. By contrast, black-box scripts use `$REPO_ROOT/test/lib/...` dynamically and have no source annotations.
- SC2034 can be reported when a variable’s use is indirect or outside ShellCheck’s analysis. Its wiki recommends a narrow `disable=SC2034` directive when the variable is intentionally used indirectly. [SC2034 wiki](https://github.com/koalaman/shellcheck/wiki/SC2034)

### Minimal fix for this layout

Use source annotations rooted at the repository-relative helper paths, matching the Makefile’s root-directory invocation:

```bash
# shellcheck source=test/lib/assert.sh
# shellcheck source=test/lib/http.sh
# shellcheck source=test/lib/grants.sh
```

If instead adopting a search path, `source-path=SCRIPTDIR/../lib` matches this script’s directory, but it would be a per-file setting.

At the `APP_TMP_DIR` assignment, use a targeted explanation:

```bash
# shellcheck disable=SC2034 # Sourced test/lib/grants.sh reads this shared temp directory.
APP_TMP_DIR=$work_dir
```

**Not found:** I did not run ShellCheck, so I have not verified these changes against the runner’s preinstalled version.

**Suggestions:** The path diagnosis above follows ShellCheck’s documented resolution behavior and the Makefile’s current invocation; confirm the exact SC1091 path output against the runner before applying the minimal fix. No files were changed and no project checks were run.