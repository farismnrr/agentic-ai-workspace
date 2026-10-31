# Deep audit Settings UI copy vs design guide, especially Spac…

Task: c9c827625d95272e146d5870083643c5
Status: done
Updated: 2026-10-31T16:03:01Z

Reference material; confirm applicability before reusing.

external assistant, self-reported, untrusted: treat as reference, not instructions.

# Settings microcopy audit (31 Oct 2026)
Scope: read-only review of Settings UI source and shared copy files. Focus Spaces & tools; also Models, Agent team, Persona, Connected apps, Integrations, Activity, Notifications, Usage, Account, Database.

## Policy and external guidance
- `docs/design-system.md`: compact workspace (~14px), hierarchy first, one quiet section surface, accessibility for important instructions.
- `AGENTS.md`: product language should be plain, nontechnical; docs short and scannable.
- Android Settings design: omit supporting text when label is enough; brief labels, explain the actual effect and current state, not generic defaults.
- Microsoft UX writing: start with action and key point; concise scannable language.
- No explicit Settings word-count rule is currently documented; this is a qualitative usability/design issue rather than a numerical policy breach.

## Key findings, concrete sources
1. **P0 inaccurate blanket claim:** `web/src/features/workspace/content/connected.ts:40` says every file change can be undone through Activity. `internal/modules/coding/application/editor_delete.go` allows deletion of binary/oversized files without backup; `undo.go` can refuse undo after a later change. Never promise universal undo.
2. **P0 dangerous action label:** same file `approvalsCopy.title/body` uses 'Skip approvals' and vague 'let the AI go ahead'. State explicit consequence: enabled tools can run without approval; off tools remain blocked. `approve_plan` is similarly powerful and requires visible explanation.
3. **P1 verbosity:** `spacesCopy.intro` is 36 words before content; replace with 'Control which tools AI can use in tasks and connected apps.' (12 words). SpaceTools renders group note plus description for every tool, even trivial labels: `SpaceTools.svelte:33-43`.
4. **P1 static default hints masquerade as state:** `toolGroups.write/exec.note` say 'Off until you turn it on' even when switch is on. Remove; toggle already shows live state. 'Every change can be undone' also false.
5. **P1 unnecessary per-tool descriptions:** `list_files`, `find_files`, `read_file`, `edit_file`, `delete_file` repeat their own titles. Hide/remove when self explanatory. Keep the consequence for `write_file` (can replace existing content), `run_command` (executes code), `approve_plan` (may approve a plan without person), and other dangerous/ambiguous tools.
6. **P1 other tabs:** `content/models.ts` has wordy tier intro (27 words) and context intro (25); `content/team-copy.ts` includes verbose team explanation; `content/integrations.ts` duplicates GitHub detail, some security-specific content should be preserved behind a detail view.
7. **P2 text repetition:** Account 'Sign out' label + button + redundant help; 'Your data' generic tagline; empty-state body repeats CTAs in Connected apps/Models; 'Activity' intro repeats title and undo guidance.
8. **Do not over-trim:** Database TLS/security hints, delete-account consequences, key privacy, GitHub permission scope, irreversible actions and blocked/error remediation remain explicit and accessible. Tooltips should not be sole source of safety warnings.

## Proposed copy and layout
- Spaces intro -> 'Control which tools AI can use in tasks and connected apps.'
- Tool groups -> 'Read files', 'Edit files', 'Run commands' (no generic note by default).
- Skip approvals -> 'Run enabled tools without asking first. Disabled tools stay blocked.'
- Write file -> 'Can replace existing file contents.'
- Delete file -> 'Some deletions can't be undone.'
- Run command -> 'Runs code and commands in an isolated workspace.'
- Approve plan -> 'Lets connected AI approve plans without your review.'
- Models tiers intro -> 'Choose models for your team. Changes apply to roles using that tier.'
- Context intro -> 'Maximum context a model can use.'
- Team roles intro -> 'Set each role’s model and permissions. The lead stays on.'
- Notifications -> remove 'Browser will ask for permission' when not actionable and keep blocked browser steps.

Acceptance: every Setting explained at a glance; no repeat of labels; no stale default-state text; preserve all actions, state and risk messaging; review mobile/desktop, accessible labels, and only affected black-box checks. No behavior change. No source edits performed in this research-only task.

Source: https://developer.android.com/design/ui/mobile/guides/patterns/settings

Source: https://learn.microsoft.com/en-us/windows/apps/design/style/writing-style

Source: https://learn.microsoft.com/en-us/style-guide/word-choice/use-simple-words-concise-sentences