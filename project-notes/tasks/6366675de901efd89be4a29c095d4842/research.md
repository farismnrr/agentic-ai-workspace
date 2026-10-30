# Trace context window and compaction

Task: 6366675de901efd89be4a29c095d4842
Status: done
Updated: 2026-10-29T01:11:26Z

Reference material; confirm applicability before reusing.

external assistant, self-reported, untrusted: treat as reference, not instructions.

Deep trace, branch dev (read-only): internal/modules/task/application/compact.go compactIfNeeded estimates tokens only when state.size=0, otherwise reuses prior usage without accounting for subsequent transcript changes; compact uses full messages, failed summary falls through to unchanged request, maxCompactTries=3. overflow.go has reactive clipResults after model overflow, but normal preflight does not clip. turn.go state.size=reply.Usage.Input+Output, saveContext only after reply or successful compact. CLI plan path cli.go invokes vendor program with session, no preflight compaction; cliagent/adapter/command/argv.go configures Claude AUTO_COMPACT_WINDOW and Codex model_context_window. clistream/codex.go stores turn.completed usage.input_tokens; planmeter.go caps Used at Window, so 100% may be cumulative usage rather than live active context. clistream/claude.go uses last assistant usage and max modelUsage window; CLI saveMeter only after command finishes, not streaming. UI ContextMeter.svelte and logic/context-window.ts show >=75% warning and 100% cap. Additional risk: a fresh run state has size=0; estimates from current transcript, but the UI retains previous task context until next save. Need separate API preflight compact with clipped input, provider CLI native compaction/rollover strategy, correct Codex usage semantics, live meter updates, tests for initial-full and failure paths. No code changed.