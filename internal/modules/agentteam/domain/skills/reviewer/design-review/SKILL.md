---
name: design-review
description: Judge an interface in five dimensions with evidence, and list what to keep, fix and quickly win. Use only when the job is to review a design or a UI, not code.
---

# Design review

Use this playbook when independent review is required by deep-review or
uncertain impact, or when the brief specifically asks for a design review.
Ordinary Swift changes do not automatically require a Reviewer. Review only
the interface and flows affected by the change.

When a review includes test changes, apply `AGENTS.md`'s no non-critical
white-box rule and `agents/testing.md`. Check that critical retained Go tests
have registry entries and that removed guarantees have named replacements.

For task verification, review the latest required Security assessment and its
recorded checks when the impact calls for them. A previous PASS does not cover
later mutations or failed attempts. A successful write is not a check, and do
not describe work as proven without a genuine passing check after its latest
mutation. Begin the report with exactly `Verdict: PASS` or `Verdict: FIX FIRST`.

You review; you do not change files. Read the interface and its `DESIGN.md` (files, CSS, markup). Then look at it
with `screenshot`, if the app is running: desktop, then `device: mobile`, and `y` or `overview` for the rest of a
long page. Judge what you see, not only the source, and use the facts that come with each picture (sideways
scroll, low contrast, broken images, console errors). If the app is not running, say so and judge from the source;
the DevOps teammate must start it.

Score each dimension 0 to 10 with the band (0-4 broken, 5-6 works, 7-8 strong, 9-10 exceptional) and one paragraph
of evidence that names files, selectors or lines. A dimension is independent: do not average away a failure.

0. **Is it a product?** Before the five dimensions, check a web product has what a product has: a public landing page,
   register and login, an app shell, a dashboard with key numbers, charts and recent activity, empty states, settings
   and a 404 page. A missing piece, or a page that looks like a tutorial (a heading and a table), goes first in the
   Fix list.
0b. **One component system.** In a Nuxt app every control comes from Nuxt UI and the pages share one shell. A hand-made
   button, input or card beside a Nuxt UI one, or pages with different layouts, is a finding.
1. **Direction.** One clear direction held through every decision (type, colour, spacing, copy voice), or three
   styles in a trench coat? Does it follow the project's `DESIGN.md` and tokens, with no raw colours?
2. **Hierarchy.** One obvious entry point, a readable order, a type scale used with intent, one accent used
   sparingly. Check `craft/typography` and `craft/color` with `read_reference`.
3. **Detail.** Layout against the "Layout" section of `DESIGN.md` on a phone and a wide screen (container, columns,
   gutters, equal gaps, edges that line up, nothing cramped, floating or cut off), spacing rhythm, alignment, states of every control (hover, focus, disabled, loading, empty,
   error), real content instead of filler. Check `craft/state-coverage`.
4. **Function and access.** Keyboard path, visible focus, labels, contrast, targets, responsive from phone to
   wide, reduced motion, forms that explain errors. Check `craft/accessibility-baseline`.
5. **Not machine-made.** Run `craft/anti-ai-slop`: the cardinal sins first (default indigo, two-stop hero
   gradient, emoji as icons, left-border cards, invented metrics, filler copy), then the soft tells. Is there at
   least one choice that only fits this product?

Finish with three lists, each item one line with where:
- **Keep**: what works, do not touch.
- **Fix**: P0 and P1 problems, most expensive first, each with the change to make.
- **Quick wins**: five-minute changes with large effect.

Start the report summary with exactly `Verdict: PASS` or `Verdict: FIX FIRST`. The orchestrator sends the Fix list
to the Designer. Do not use the document of your report.
