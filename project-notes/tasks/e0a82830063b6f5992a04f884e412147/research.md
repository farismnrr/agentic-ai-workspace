# Deep UI audit and PRD only: review blank/flat workspace scre…

Task: e0a82830063b6f5992a04f884e412147
Status: done
Updated: 2026-10-30T06:28:29Z

Reference material; confirm applicability before reusing.

external assistant, self-reported, untrusted: treat as reference, not instructions.

# Deep visual UI audit — cards and surface hierarchy (research only)

## Scope and evidence
- Inspected the two owner screenshots: desktop Project > Checks (empty) and external task detail > Plan (no plan; transcript with mixed message/event blocks); this is direct visual evidence. Additional page evaluations are code inspection only, **not** rendered screenshot confirmations.
- Branch `dev` at `e461846`, clean before research. No UI code edits or app tests performed.
- Read `docs/design-system.md`, `agents/memory-frontend.md`, `web/src/app.css`, `web/src/features/workspace/shell/AppShell.svelte`, project tabs, Checks, SettingsFrame/SettingsSection, task split panel/PlanTab/TaskThread/EventItem/ChatBubble, empty states, Home, Sites, Inbox, Projects, Tasks and Files.
- Workspace has ~135 Svelte files; roughly 16 consumers of `SettingsSection`; ~20 Svelte files consume `EmptyState`; some existing proper cards at Sites list, Projects loading state, Home modules, ResultCard, etc. This is a sampled structural audit, not a claim that all 135 screens were visually verified.

## Confirmed structural causes
1. `docs/design-system.md` Workspace explicitly mandates flat rows, hairlines, no card shadows; `agents/memory-frontend.md` repeats. This is contrary to proposed redesign, so policy must be updated *after owner approves PRD*. Brand priorities remain paper-like surfaces, olive semantic primary, base-200 canvas, base-100 panels, borders before shadows, compact 14px scale.
2. Project detail `ProjectDetailPage.svelte` header + tabs sit at top; tab body is an unframed `mt-4` region. `ChecksTab.svelte` emits two loose intro paragraphs, a bare list whose empty slot is `EmptyState compact`, then a loose full-width divider and 'Add a check' button. Owner screenshot shows contextless grouping and huge featureless field.
3. Task detail `TaskScreen.svelte` already separates transcript and right aside with a border + bg-base-100, so wrapping the entire conversation and panel in cards would be redundant; `TaskThread.svelte` intentionally has loose agent messages and collapsed action blocks, while `EventItem` and `ResultCard` use cards for meaningful events. Screenshot shows inconsistent vertical grouping of provenance, tool steps, receipts and assistant summary. `PlanTab.svelte` has standalone dashed paragraph for empty plan, `WorkPanel.svelte` has surface with no semantic section hierarchy.
4. `SettingsFrame.svelte` has nav and raw tab content; `SettingsSection.svelte` outputs just headings/dividers, no surface; 16 usages across account, models, notifications, activity and project Env/About. Scope to reusable section panel with opt-in variant to prevent nested panels; avoid changing controls.
5. Project `TasksTab.svelte`, global `TasksPage.svelte`, `TasksSelection.svelte` currently put filter/selection and flat rows on background. Add **one** container around an entire list/empty state; retain flat internal rows.
6. Other already-framed regions should not be re-carded: Home `NeedsYou` / `RunningNow`, Sites populated list, Projects loading, receipt/approval cards, user chat bubble, sidebar/topbar, FileBrowser editor grid, Inbox split. Inconsistent empty-vs-populated states in Sites/Projects are candidates for surface normalization.

## Best-practice cross-check
- Nielsen Norman Group: cards group related content; good for dashboard collections/heterogeneous summaries, bad for long homogeneous scan-heavy lists, which become less scannable and need more space: https://www.nngroup.com/articles/cards-component/
- Material: use cards only when grouping requires more separation than whitespace/dividers; avoid excessive regions: https://m1.material.io/layout/structure.html
- Carbon: a tile is a foundational grouped surface, not necessarily elevated; cards must not nest in tables / dense content: https://www.carbondesignsystem.com/building-blocks/core/components/tile/guidelines
- Queensland Health Design System: keep one concept per card, actions uncluttered, short information: https://www.design-system.health.qld.gov.au/components/card

## Recommended decision
Adopt a **bounded-section / quiet-card** language: page canvas base-200, one base-100 outlined rounded section panel per coherent function, no card shadow; internal flat/hairline rows, semantic action footer, compact EmptyState *inside same parent card* with no extra dashed border. Distinguish 3 display categories: (A) bounded section card for functional form/list/empty, (B) list rows within bounded section, (C) unboxed transcript/editor/split for high-density reading. Avoid card-per-event/card-in-card.
Suggested P0: Project Checks empty/filled, reuse SettingsSection with controlled surface, Project Tasks/Global Tasks lists; task WorkPanel/Plan empty+docs. P1: Settings affected tabs, Project Env, Projects/Sites empty+filled consistency, Inbox empty; P2: review Home, other editor/file and timeline variants only if screenshots justify.

## Risks and guardrails
No fake data, no API/auth/agent lifecycle changes, no new design library/tokens unless approved; preserve textarea composer, live streaming, async empty/loading/error, keyboard focus, file/editor scroll, role/provenance evidence distinctions. Screenshots for desktop 1440 and widths 390/820/1024, content states empty/filled/loading/error; verify no horizontal overflow, no excessive double outlines or nested cards; compare before/after and use targeted browser black-box smoke.

Source: https://www.nngroup.com/articles/cards-component/

Source: https://m1.material.io/layout/structure.html

Source: https://www.carbondesignsystem.com/building-blocks/core/components/tile/guidelines

Source: https://www.design-system.health.qld.gov.au/components/card