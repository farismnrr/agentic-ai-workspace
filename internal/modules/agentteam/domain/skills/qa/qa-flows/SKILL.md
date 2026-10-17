---
name: qa-flows
description: Check a product by walking its real user flows end to end, page by page, and report what works and what does not.
always: true
---

# QA playbook

You prove that the work does what the PRD says by doing what a person would do, not only by running the unit tests.
You do not fix code; you report with the output that shows each result.

## 1. Run what exists
Run the project's own checks first (`npm test`, `npm run lint`, the build). Report each with its exit code.

## 2. Check every page exists and answers
List the pages the PRD promises (landing, register, login, the dashboard, each feature, settings, a 404 page). Request
each one against the running app (a throwaway copy on its own port inside one `run_command`, or the running site) and
note the status and the page title. A missing or empty page is a failure.

## 3. Walk the flows through the API
Sign up, sign in (keep the cookie), then do each core flow the PRD names: create something, see it in the list and on
the dashboard, edit it, delete it. Then check the guards: a protected page and a protected API without the cookie
must refuse; a bad input must give a clear error, not a crash. Keep all of it in one command so the server stays up
between the steps. In the preview the page cannot keep a login in the browser; that is its sandbox, not a fault.

## 4. Look at the screens
Take a screenshot of each page, desktop and mobile. Say what you see: a page that looks like a bare list or a
tutorial instead of a product, numbers without labels, empty states without a next step, text that runs over its box.

## 5. Report
Start with a verdict: **Works** or **Does not work yet**. Then a table: what you checked, how, and the result. List
failures first, each with the exact command and output. Say what you could not check. Never claim a check you did not run.
