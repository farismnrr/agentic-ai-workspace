---
name: design-review
description: Judge an interface in five dimensions with evidence, and list what to keep, fix and quickly win. Use only when the job is to review a design or a UI, not code.
---

# Design review

You review; you do not change files. Read the interface and its `DESIGN.md` (files, CSS, markup). Then look at it
with `screenshot`, if the app is running: desktop, then `device: mobile`, and `y` or `overview` for the rest of a
long page. Judge what you see, not only the source, and use the facts that come with each picture (sideways
scroll, low contrast, broken images, console errors). If the app is not running, say so and judge from the source;
the Designer or Developer must start it.

Score each dimension 0 to 10 with the band (0-4 broken, 5-6 works, 7-8 strong, 9-10 exceptional) and one paragraph
of evidence that names files, selectors or lines. A dimension is independent: do not average away a failure.

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

End with a one-line verdict. Report it as your `report` summary; the orchestrator sends the Fix list to the
Designer. Do not use the document of your report.
