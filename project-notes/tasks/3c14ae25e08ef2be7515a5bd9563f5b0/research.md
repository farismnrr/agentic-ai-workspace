# Settings UX writing cleanup: PRD and implementation, same ta…

Task: 3c14ae25e08ef2be7515a5bd9563f5b0
Status: done
Updated: 2026-10-31T16:28:49Z

Reference material; confirm applicability before reusing.

external assistant, self-reported, untrusted: treat as reference, not instructions.

## Research for the Settings microcopy PRD

Current source verified on branch dev at 803dfbb. `web/src/features/workspace/content/connected.ts` has a 36-word spaces introduction and group hints that describe a default off state irrespective of live switch state. The sentence 'Every change can be undone under Activity' is contradicted by `internal/modules/coding/application/editor_delete.go`: binary and oversized files can be deleted without backups, and `internal/modules/coding/application/undo.go` refuses to reverse changes if later file state differs. `SpaceTools.svelte` renders each tool's explanatory text for all tools by default. `models.ts`, `team-copy.ts`, `integrations.ts`, `subscriptions.ts`, and individual tabs contain similar redundancies.

Authoritative outside UX guidelines confirm the treatment:
- Microsoft Visual Studio UX says don't explain the obvious; make helper text brief and eliminate duplicated wording.
- Microsoft Windows writing guide advises placing key information first, concise verbs, and scannable labels while preserving necessary explanation.
- Android Settings guidelines recommend brief meaningful titles and showing actual status/value rather than merely rephrasing labels.
- Fluent 2's field guidance treats helper text as format/requirement clarification, not filler.

Keep security warnings, permission scope and irreversible consequences visible. Design-system docs already specify compact, quiet Settings surfaces and accessible text, so the changes should mostly be copy and conditional suppression of obvious secondary rows. PRD is saved in this task, and full previous audit research remains in prior issue notes.

Source: https://learn.microsoft.com/en-us/visualstudio/extensibility/ux-guidelines/ui-text-and-help-for-visual-studio

Source: https://learn.microsoft.com/en-us/windows/apps/design/style/writing-style

Source: https://source.android.com/docs/core/settings/settings-guidelines

Source: https://fluent2.microsoft.design/components/web/react/core/field/usage