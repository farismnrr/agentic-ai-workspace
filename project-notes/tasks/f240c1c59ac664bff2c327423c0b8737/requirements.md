# Critical: @deep-review team completes developer plan but req…

Task: f240c1c59ac664bff2c327423c0b8737
Status: done
Updated: 2026-10-29T03:46:45Z

Reference material; confirm applicability before reusing.

external assistant, self-reported, untrusted: treat as reference, not instructions.

# PRD: Fix deep-review CLI orchestrator repeated Needs Attention after 3/3 plan done

Observed: user screenshot shows @deep-review task, all steps Done, Developer reported, orchestrator read `rg --files -g 'AGENTS.md' -g '*_test.go'`, then three identical blocked cards 'This work is blocked until its required review is recorded.' Task status Failed/Needs attention, user must press Continue repeatedly.

Required: one existing task, automatic continuation of vendor CLI session if completion gate requests independent reviewer/security/architect/commit review; feed exact gate feedback to CLI and require actual role report; do not silently bypass evidence; count attempts; disabled reviewer or exhausted review rounds returns ONE specific blocker; read-only inspection should not be mislabeled unknown file effects. Existing blocked task can be continued after deployment without duplicate generic error. Swift unaffected. No unrelated change or excessive unit testing.

Acceptance: deterministic CLI fake session simulates changed CSS, Reviewer reports PASS -> same task and resumable vendor session complete; reviewer absent/nonresponsive -> bounded specific failure; historical error cards + continuation -> no newly duplicated generic blocked cards; read-only `rg --files` produces known no-effects while unknown shell remains conservative; full Go + frontend + browser lifecycle/context pass; Git clean and dev push.