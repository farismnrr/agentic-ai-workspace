# Retained white-box test registry

Every retained non-bootstrap Go test has one row. Bootstrap tests remain an
in-process app integration group and are tracked together here. Support files
are listed after the table; they contain fixtures used by registered tests.

After this slice, 81 Go test files contain 157 top-level test functions:
62 bootstrap files contain 142 functions, 9 registered white-box files
contain 15 functions, and 10 support files contain no test functions.

| Test path::Name | Category | Why black-box is insufficient | Bash counterpart |
| --- | --- | --- | --- |
| `internal/bootstrap/*_test.go::bootstrap integration group` | app-level integration | These tests exercise the assembled app over HTTP and MCP in-process; the existing group is retained without a broad rewrite. | `test/blackbox/` scenarios |
| `internal/bootstrap/sites_browser_test.go::TestServeSiteForABrowser` | app-level integration | scripted model needed; no API to create deterministic task events | `test/e2e/02-sites.spec.ts` |
| `internal/bootstrap/thread_browser_test.go::TestServeThreadForABrowser` | app-level integration | scripted model needed; no API to create deterministic task events | `test/e2e/03-thread.spec.ts` |
| `internal/bootstrap/foundations_test.go::TestProviderKeyIsSealedVerifiedAndNeverReturned` | crypto/credentials | The Bash scenario stores a key but does not verify it or prove the credential remains usable. | `test/blackbox/04-secrets-never-returned.test.sh` |
| `internal/bootstrap/isolation_test.go::TestPeopleNeverShareAWorkspace` | cross-user isolation | The Bash scenario checks project access but does not check each person's coding workspace. | `test/blackbox/02-cross-user-isolation.test.sh` |
| `internal/bootstrap/task_access_test.go::TestTaskNeedsAModelAndStaysPrivate` | auth/authz | The Bash scenario does not check task creation without a model or an unknown project. | `test/blackbox/02-cross-user-isolation.test.sh` |
| `internal/bootstrap/task_access_test.go::TestAnotherPersonCannotSeeOrControlATask` | cross-user isolation | The Bash scenario checks projects but does not attempt to read or control another person's task. | `test/blackbox/02-cross-user-isolation.test.sh` |
| `internal/modules/coding/adapter/runner/sandbox_test.go::TestOnlyTheWorkspaceIsWritable` | sandbox/privilege | Directly checks write confinement inside the task container, including workspace mount edges. | `test/smoke/sandbox.test.sh` |
| `internal/modules/coding/adapter/runner/sandbox_test.go::TestTaskHasNoNetworkAndNoPrivileges` | sandbox/egress/privilege | Inspects actual container network and capability settings that app responses cannot prove. | `test/smoke/sandbox.test.sh` |
| `internal/modules/task/adapter/sqlite/work_store_test.go::TestWorkStoreCommitRollsBackAllChangesOnEventFailure` | DB integrity/transactions | Injects a failure between writes to prove transaction rollback deterministically. | none; invariant remains white-box |
| `internal/modules/task/adapter/sqlite/work_store_test.go::TestWorkStoreCreateRollsBackTaskWorkAndRequestWhenRoleEventFails` | DB integrity/transactions | Injects a role-event persistence failure and proves related rows roll back atomically. | none; invariant remains white-box |
| `internal/modules/task/adapter/sqlite/approval_store_concurrent_test.go::TestConcurrentApprovalCreatesOneDecisionAndHandoff` | critical concurrency/data loss | Synchronizes concurrent approvals at the store boundary and verifies one decision and outbox row. | none; public timing race is not deterministic |
| `internal/platform/secretbox/cipher_test.go::TestRoundTripAndKeyPersistsAcrossRestarts` | crypto/credentials | Checks ciphertext and key behavior across adapter reopen, outside one credential API flow. | `test/blackbox/05-restart-keeps-data.test.sh` |
| `internal/platform/secretbox/cipher_test.go::TestSealedValuesDifferAndTamperingFails` | crypto/credentials | Corrupts ciphertext directly to prove authenticated decryption rejects tampering; no public API permits mutation. | `test/blackbox/04-secrets-never-returned.test.sh` |
| `internal/platform/egress/egress_test.go::TestPrivateAllowedDialControlRejectsResolvedMetadataAddress` | sandbox/egress/SSRF | Controls address resolution between validation and dial to verify metadata IP rejection. | none; deterministic resolver and dial control are needed |
| `internal/platform/egress/egress_test.go::TestTransportReallyRefusesToConnect` | sandbox/egress/SSRF | Verifies custom transport enforcement at the socket boundary with a controlled dialer. | none; controlled dial is needed |
| `internal/platform/egressproxy/grantpath_test.go::TestADNSAnswerThatChangesAfterTheCheckChangesNothing` | sandbox/egress/SSRF | Forces a DNS answer change between check and connect to cover rebinding deterministically. | none; controlled resolver is needed |
| `internal/platform/egressproxy/grants_race_test.go::TestRevokeRacingNewTunnelsLeavesNoneOpen` | critical concurrency/data loss | Synchronizes revocation with tunnel creation and proves no tunnel escapes revocation. | none; controlled concurrency is needed |
| `internal/platform/database/migrate_test.go::TestJournalRecordsChecksums` | DB integrity/migrations | Inspects stored migration checksums directly after migration. | none; journal inspection is required |
| `internal/platform/database/migrate_test.go::TestEditedMigrationIsRejected` | DB integrity/migrations | Mutates migration history and verifies startup rejects changed applied migrations. | none; direct journal mutation is required |
| `internal/platform/database/migrate_test.go::TestRemovedMigrationIsRejected` | DB integrity/migrations | Mutates migration history and verifies startup rejects removed applied migrations. | none; direct journal mutation is required |
| `internal/platform/sandbox/subpath_integration_test.go::TestStage7ActualVolumeSubpathsAreIsolated` | sandbox/privilege | Uses a Docker volume to prove owner subpaths cannot read or write one another. | `test/smoke/sandbox.test.sh` |

## Support files

- `internal/modules/task/adapter/sqlite/work_store_fixtures_test.go` provides fixtures used by retained work-store checks.
- `internal/modules/task/adapter/sqlite/approval_fixture_test.go` provides the plan submission fixture used by the retained concurrent approval check.
- `internal/modules/task/adapter/sqlite/linked_tasks_parallel_test.go` provides the SQLite store fixture used by the retained concurrent approval check.
- `internal/modules/task/adapter/sqlite/work_test_schema_test.go` provides the SQLite schema fixture used by retained work-store checks.
- `internal/modules/coding/adapter/runner/runner_test.go` provides the Docker runner fixture used by retained sandbox checks.
- `internal/platform/egressproxy/client_test.go` provides the tunnel client used by the retained grant-path check.
- `internal/platform/egressproxy/clienthello_test.go` provides a TLS ClientHello fixture used by the retained grant-path check.
- `internal/platform/egressproxy/extra_test.go` provides the fake dial adapter used by the retained grant-path check.
- `internal/platform/egressproxy/grants_test.go` provides the open-tunnel counter used by the retained revoke race check.
- `internal/platform/egressproxy/rig_test.go` provides the rig used by retained grant-path checks.
