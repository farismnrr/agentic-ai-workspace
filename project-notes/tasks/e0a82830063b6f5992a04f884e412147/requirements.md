# Deep UI audit and PRD only: review blank/flat workspace scre…

Task: e0a82830063b6f5992a04f884e412147
Status: done
Updated: 2026-10-30T06:28:29Z

Reference material; confirm applicability before reusing.

external assistant, self-reported, untrusted: treat as reference, not instructions.

# PRD — Workspace Surface & Card Hierarchy Refresh
**Status:** Draft for owner review · **Work type:** Design/refactor proposal only · **Target:** `agentic-ai-workspace`, `dev` · **Priority:** P0 visual consistency.
**Evidence:** 2 supplied desktop screenshots (Project Checks empty; external task transcript + Plan empty), read-only source audit, existing design system and UI research. Remaining screens sampled from source; visually validate with baseline captures before authorizing changes.

## 1. Problem statement
Current UI uses paper-like canvas and flat/hairline rows, but functional regions often have no visual bounds. In Checks, intro text, missing-data message and Add action float apart on a large empty canvas. In external task detail, visually dissimilar transcript blocks/provenance and the Plan empty state lack consistent grouping. Several Settings/Project tabs use raw `SettingsSection` with only headings and dividers, while other pages already have cards. Result: inconsistent visual hierarchy and excessive *unstructured* whitespace, especially at desktop widths.

**Root causes verified:**
- `docs/design-system.md` currently mandates compact, flat rows, no card shadow; the request changes only the **container/grouping policy**, not the overall brand.
- `ProjectDetailPage.svelte` outputs unframed tab content; `ChecksTab.svelte` leaves a standalone EmptyState and detached footer action.
- `SettingsSection.svelte` is borderless; used across Settings and Project Env/About.
- `WorkPanel.svelte` is already a full-height aside surface; `PlanTab.svelte` instead draws one small dashed no-plan paragraph. Transcript already has purposeful `*Card.svelte` components, so universal card-per-event is inappropriate.

## 2. Goals and non-goals
**Goals:**
G1. Give each *logical work module* one clear visual boundary; make its title, content/empty/loading/error, and action belong together.
G2. Preserve compact 14px workspace typography, scan-friendly tables/lists, sidebar, theme and existing click behavior.
G3. Make empty, populated, loading and failure states feel like one component instead of changing layout or leaving actions detached.
G4. Improve consistency in Project tabs, Settings and right-side task context while preserving a calm conversation/editor experience.
G5. Avoid visual AI-slop (repeated cards, needless gradients, glows, pills, shadows, oversized whitespace).

**Non-goals:**
- No redesign of branding/colors/type scale, information architecture, API, auth, data, AI execution, tasks, agents, role attribution, Git diffs, or application routes.
- No card for every message, log row, task, table row, or editor line; no nested card stacks; no additional design library.
- No implementation, commits, tests, deployment or CSS edits in this PRD task.

## 3. Design rules (proposed, pending approval)
**A. Surface anatomy:** One `Section Surface` = optional title + compact description + optional right-aligned action + single content/list/empty area + optional footer. Content states share the *same* parent boundary. A single section wrapper should not contain another full section wrapper.
**B. Existing palette only:** page `bg-base-200`, surface `bg-base-100`, hairline `border-[var(--brand-border)]` and stronger separator only when necessary; radius consistent with existing `rounded-xl` / daisyUI card. **No shadow on page cards**; only current popups/composer may float using existing tokens. Never invent a new hex/token for this job.
**C. Density:** Keep compact rows (14px, tight line-height), table/selection rows inside their parent outlined container separated by hairlines. Padding proposal: section header/body around `p-4` (or `p-3` in right panel), list rows keep current compact padding. Confirm exact spacing visually, not arbitrary numeric hard-coding across files.
**D. Empty states:** Reuse `EmptyState` compact; **no additional dashed border if parent is a card**. Show meaningful icon + one-line title + reason + at most one relevant action in the same card. Empty content must not leave a giant full-height blank zone, except deliberate app/editor work areas.
**E. Layout selection:** Cards = distinct sections or heterogeneous dashboard summaries; rows/table = homogeneous scan-heavy lists; unboxed = conversation transcript, code editor, split panes, global navigation; status/approval/receipt already have their own semantic cards.
**F. Interaction:** Card is NOT automatically clickable. Only an explicit link/button responds; do not introduce clickable wrapper around nested controls. Preserve visible focus and status meaning beyond color.
**G. Responsive:** Desktop section surfaces can fill current content width but text should remain readable; on mobile use 1-column and compact padding, avoid nested horizontal scroll. Preserve task composer, task split, and editor viewport.

## 4. Screen-level scope and priority

### P0 — targeted first pass
**P0-1 Project detail > Checks (screenshot 1):**
- Keep page title, New task CTA, tabs outside. Put Checks **intro/explanation, content/empty list, and Add a check action** in one bounded module (internal subheader allowed).
- When empty, show centered *compact* state within bounded content with action **near** it (prefer single clearly visible Add a check CTA; avoid duplicate controls), not a detached bottom-left button under a faraway rule.
- With checks: same parent container, each check remains flat row; Edit/Remove preserve order, control semantics, confirmation and error feedback.
- Files: `projects/ProjectDetailPage.svelte`, `projects/ChecksTab.svelte`, shared `EmptyState.svelte` (prefer existing API).

**P0-2 External + internal Task detail > Plan/Documentation (screenshot 2):**
- Keep desktop chat and right-side aside as distinct flush panes; no card on entire chat or additional outer right-pane card.
- Within Plan/Docs, use consistent section hierarchy: content header and compact empty content inside **existing aside surface**. No small isolated dashed box with lots of unclaimed room; avoid surrounding each plan step with another card.
- Preserve research/PRD accordions, scroll independence, streaming, changes/files tabs and existing provenance/evidence cues.
- Files: `task/TaskScreen.svelte`, `task/panel/{WorkPanel,PlanTab,ProjectDocuments}.svelte`.

**P0-3 Project Tasks + global Tasks list:**
- Group filter/selection/list/empty state into one parent surface, preserve checkboxes, keyboard and stable table-like rows, sticky content if already present.
- Keep empty content compact and properly anchored to the filter/header context.
- Files: `projects/TasksTab.svelte`, `tasks/{TasksPage,TaskSelection}.svelte`.

### P1 — broad consistency rollout after P0 visual sign-off
**P1-1 Settings and Project About/Env:** Add a controlled `surface` (or equivalent opt-in) treatment to reusable `SettingsSection.svelte`. Apply at its actual consumers (Models, Account, Notifications, Activity, Integrations, Team, Rules, Usage, Project About, Env Variables/Database/Gateway). Reuse header description/action slots and divide-y rows. **Prevent nesting:** a Project tab that already frames the content must not add another outer frame.
**P1-2 Projects, Sites, Inbox:** Harmonize empty vs populated list surfaces; use one bounded region per list/empty state, not card-per-project, card-per-site, or card-per-inbox item. Inbox split + detail action cards remain structurally unchanged.
**P1-3 Changes/Team context:** Revisit double-border cases (`ChangesTab` already contains list cards and nested history). Reduce duplicated outlines if using parent surfaces; maintain diff/editor and evidence semantics.

### P2 — review only, implement only if screenshot shows need
Home already uses bounded Needs You/Running Now lists; Projects table and Sites list use cards in some states. FileBrowser is a split editor; chat bubbles, ResultCard, ApprovalCard and composer already have intentional surfaces. **Do not mass-wrap these.** Evaluate screenshots first; limit updates to margins/empty consistency where objectively needed.

## 5. Reuse-first implementation approach (future, NOT executed now)
- Prefer existing daisyUI `card`, Tailwind `bg-base-100`, border token, `rounded-xl`, `divide-y`, shared `EmptyState` and current `SettingsSection`.
- Consider one small reusable Section Surface wrapper **only if** repeated wrapper markup would otherwise be copied across 3+ functional pages; otherwise a local parent card is sufficient. Avoid new token sets or scoped CSS for static appearance.
- Update `docs/design-system.md` and `agents/memory-frontend.md` once approved: clarify "flat rows **inside** bounded functional sections", no page-card shadows, no card per event, not the blanket 'no cards' rule.
- Retain original accessible section labels, header semantics, focus management, scroll constraints, and existing mobile tab navigation.

## 6. Acceptance criteria / black-box checks
**AC1** Project Checks: empty -> one visibly grouped section with explanation, centered concise empty state and actionable Add CTA; loaded -> same frame with compact rows and controls; modal save/remove flows and failures work.
**AC2** Task Plan: zero plan, steps present, research/PRD accordion, external and internal tasks all remain readable; scrolling chat does not scroll right panel and vice versa.
**AC3** Every touched Settings/Env section shows a consistent header, body and actions in one panel across empty/loading/error/populated states. There is no nested card-on-card / double border.
**AC4** Projects/Tasks/Sites collections remain a **scannable list/table**. No item-per-card explosion, no wrapped selectable controls turning into non-functional surfaces.
**AC5** Capture and *visually inspect* before/after for P0 screens at 1440 desktop, 1024 desktop/tablet, 820 tablet, 390 mobile; no horizontal overflow (`scrollWidth <= innerWidth`), no clipped action, no layout jump that hides interaction.
**AC6** Keyboard navigation, focus-visible, contrast (WCAG AA), aria labels, error/success information and reduced motion stay intact. No behavioral changes to API, task lifecycle, verification proof or auth.
**AC7** Use the existing `scripts/shots.sh` with an isolated DATA_DIR/port for screenshots; targeted Playwright black-box flows and frontend check/lint/build on affected scope only. Full CI/whitebox excluded unless change-specific regression warrants it.
**AC8** Review outcome includes side-by-side screenshots, list of changed files, any exceptions to surface rules, and zero unapproved additional feature work.

## 7. Risks and mitigations
- **Card overload / poor scan density:** apply 1 logical module -> 1 container, NOT row-per-card; compare filled dense lists.
- **Double borders (tabs, panel, lists):** account for existing parent `aside`, `SettingsSection` and `ChangesTab` outline before adding wrappers.
- **Mobile horizontal overflow:** check 390/820/1024, preserve minmax(0,1fr) and overflow handling in split view.
- **Behavior regressions from reused component changes:** opt-in at reusable `SettingsSection` rather than unconditional global rules; targeted browser flows including error/loading.
- **Overfitting two screenshots:** capture baseline for candidate pages before P1/P2 changes; do not call static code inspection a visual confirmation.

## 8. Delivery sequencing and approval gate
1. Baseline screenshots and P0 focused pass, obtain owner visual feedback.
2. Reuse tested surface rule across P1 tabs and states, capture before/after.
3. P2 only if observed need. Update design-system docs once visual decisions are signed off.
**Current deliverable: this draft PRD and research note only.** Do not alter UI or submit for automatic execution. Owner approval required before implementation.