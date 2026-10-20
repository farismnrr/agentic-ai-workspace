---
name: browser-extension
description: How to build a browser extension (Chrome and Edge, manifest version 3) in TypeScript with WXT, with the least permissions, ready for DevOps to package.
---

# Browser extension (Chrome and Edge)

Use this when the request is a browser extension: a button in the toolbar, a popup, something that changes the pages
people visit, or a small tool that lives in the browser. The person installs it from Extensions in the sidebar, not
from Sites; DevOps packages it with `deploy_extension` after you build it. You never start a server for it.

## Stack
TypeScript with WXT (`wxt`), manifest version 3, Bun. Nothing is downloaded from GitHub in the sandbox, so do not run
`wxt init` or `bun create wxt`: write the files yourself, they are few. Keep the files readable (see `delivery-rules`).

```
package.json            "type": "module", scripts build = "wxt build", typecheck = "tsc --noEmit"; dependencies: wxt, typescript
wxt.config.ts           import { defineConfig } from 'wxt'; export default defineConfig({ manifest: { name, description, permissions } })
tsconfig.json           { "extends": "./.wxt/tsconfig.json" }   (made by `bunx wxt prepare`, which `bun install` runs)
entrypoints/popup/index.html + main.ts      the popup behind the toolbar button
entrypoints/background.ts                   export default defineBackground(() => { ... })  (only if it needs one)
entrypoints/content.ts                      export default defineContentScript({ matches: ['https://example.com/*'], main() { ... } })
public/icon/16.png 32.png 48.png 96.png 128.png   WXT finds icons here by name
```
`bun install`, then `bun --bun run build`. The build is written to `.output/chrome-mv3/`; that folder, the one holding
`manifest.json`, is what DevOps packages. Build in production mode (`wxt build`, never a dev build), because a dev build
adds permissions. The version comes from `package.json` (`"version": "1.0.0"`); raise it for every change.

## Icons
Make real icons, not empty files. Draw a simple mark in the colours of `DESIGN.md` with ImageMagick and write all five
sizes, for example `convert -size 128x128 xc:'#2563eb' -fill white -gravity center -pointsize 72 -annotate 0 'T'
public/icon/128.png`, then `convert public/icon/128.png -resize 48x48 public/icon/48.png`, and the same for 16, 32, 96.

## Permissions: as few as possible
The person reads what the extension may do before installing it, in plain words, and a long list scares people off.
- Prefer `activeTab` (works on the page they click it on) over `tabs`; prefer `storage` for settings.
- Name the sites it needs in `host_permissions` or the content script's `matches`. Never `<all_urls>` or `*://*/*`
  unless the idea cannot work otherwise, and then say why in your report.
- Never ask for `cookies`, `history`, `webRequest`, `debugger`, `management` or `nativeMessaging` unless the request is
  about exactly that.
- No remote code: no script loaded from the web, no `eval`, no inline `<script>` in the popup. The code is in the package.
- Send nothing to anyone's server unless the request needs it; if it does, name the address in your report.

## Look and feel
The popup is small: about 320 to 400 pixels wide and no taller than 600. Follow `DESIGN.md`; fonts come from npm like
everywhere else (see `delivery-rules`). `chrome.*` does not exist when the popup page is opened as a normal web page, so
guard calls (`typeof chrome !== 'undefined'`) if the Designer is to look at `.output/chrome-mv3/popup.html` served by
DevOps.

## Before you report
`bun --bun run typecheck`, the tests of what you wrote, and `bun --bun run build` must pass. Report the build folder
(`.output/chrome-mv3`), the version, and every permission with the reason for it, in one line each. DevOps packages it.
