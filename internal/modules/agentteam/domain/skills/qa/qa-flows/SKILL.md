---
name: qa-flows
description: Check a product by walking its real user flows end to end, page by page, and report what works and what does not.
always: true
---

# QA playbook

You prove that the work does what the PRD says by exercising public behavior. Follow `agents/testing.md`: black-box scenarios are the default; retain a white-box check only for a registered super-critical behavior that cannot be tested at a public boundary.
You do not fix code; you report with the output that shows each result.

## 1. Choose the checks
Use saved mode and changed behavior, dependencies, contracts and
configuration. Ordinary Swift work uses affected public-boundary scenarios
and relevant static checks. Sensitive changes retain their named critical
scenarios. Deep-review adds broader applicable regression and failure-path
checks. Unknown impact broadens conservatively. Do not add tests to raise
coverage or duplicate an existing guarantee. Report each command and result.

## 2. Check affected pages and routes
For a complete product build, check the pages the PRD promises. For a focused
change, check the pages and routes the changed behavior can affect. Use the
running app or an approved test harness; do not start the app yourself. Note
status and visible result. A missing or empty in-scope page is a failure.

## 3. Walk the flows through the API
Walk the core flows in scope, including relevant failure paths and ownership
guards. For a complete product build, include sign-in and each named flow. Keep
all steps in one command when the server must stay up. In the preview the page
cannot keep a login in the browser; that is its sandbox, not a fault.

## 4. Look at the screens
When visual review is in scope, check affected screens at desktop and mobile
sizes. Report concrete issues such as missing labels, empty states without a
next step, overflow or broken styling.

## 5. Report
Start with a verdict: **Works** or **Does not work yet**. Then a table: what you checked, how, and the result. List
failures first, each with the exact command and output. Say what you could not check. Never claim a check you did not run.
