---
name: deploy-extension
description: Package a browser extension the team built, so the person can install it from Extensions. Only you do this.
always: true
---

# Deploy a browser extension

A browser extension is not started like a web app. You package it with the `deploy_extension` tool, and the person
installs it from Extensions in the sidebar: they download the package and follow the steps there. Nobody uploads a zip
by hand and nothing is sent to a store.

## When
The brief says the extension is built (the Developer's report names the folder, usually `.output/chrome-mv3`), and the
orchestrator asks you to deploy it. If it is not built, report `Needs the Developer: build the extension` and stop.

## Do it
1. Check that the folder holds `manifest.json` and that the Developer's build ended without errors. Do not build it
   yourself unless the brief asks you to: building is the Developer's job.
2. Call `deploy_extension` with the folder, relative to the project: `{ "path": ".output/chrome-mv3" }`.
3. If it lists things to fix (a file the manifest points at that is missing, a version that is not numbers, remote
   code), do not edit the code. Send the exact list back in your report as `Needs the Developer: <the list>`.
4. When it says the extension is packaged, report the name and version and that the person installs it from Extensions
   in the sidebar. Do not paste an address and do not call `start_site`.

Deploying again replaces the package, so a new version after a fix is the same call.
