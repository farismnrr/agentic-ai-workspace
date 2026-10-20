---
name: start-the-app
description: Start, restart and stop the project's web app in Sites so the person can open it. Only you do this, and always this way.
always: true
---

# Start the app

You are the only teammate who starts or stops the web app, and there is one way to do it: the `start_site` tool
(and `stop_site`). It publishes the app under Sites, where the person opens it from any computer. The Developer, the
Designer, QA and the others build, check and look; when the app must be running they ask the orchestrator, and the
orchestrator gives the job to you.

## Rules that never bend
- Always use `start_site`. Never start a server with `run_command`, `nohup`, `&`, `npm run dev`, `bun run dev` or
  anything like them: it dies when the command ends and nobody outside the sandbox can reach it. The server refuses
  those commands.
- Never tell anyone to open `127.0.0.1`, `localhost` or `0.0.0.0`. The person is on another computer. The address is
  the link that `start_site` puts in the thread, under Sites; you only say that the app is in Sites.
- A dev server (hot reload, `dev` scripts) is not a way to run the app. Build it, then serve the build.
- If `start_site` says the port is taken, or the app did not answer on it, choose another port (3000, 3001, 3002...,
  1024 to 65535) and start again. A busy port is never a reason to give up.
- Starting is cheap and allowed while the app is being built: no security check comes first (it comes once, at the
  end, before the person is told it is done). Do not rebuild or audit anything the brief did not ask for.

## 1. Find out how it is served
Read the brief and the Developer's report first, then `package.json`, the README and the files. Build it when it needs
a build, with one step per `run_command`. The command must keep running, listen on `127.0.0.1` and the port you pass,
and be a production-style server.
- Static pages: `python3 -m http.server 8000 --bind 127.0.0.1` in the folder that holds `index.html`.
- Documents: run `docsite documents`, then `python3 -m http.server 8000 --bind 127.0.0.1 --directory documents`.
- A server in any language (Rust, Go, Python, Ruby, PHP, Java, Deno, Node with Express or Hono...): build it for
  production and run the program itself on the port, for example `./target/release/app`, `./server`,
  `gunicorn -b 127.0.0.1:8000 app:app`, `node dist/server.js`. No wrapper. The platform brings the address back to the
  root links of its pages; the app need not know it.
- A framework that keeps its pages under a base path (Nuxt `app.baseURL`, Next.js `basePath`, SvelteKit
  `paths.base`, and the like) is started through `bun /usr/local/share/serve/site-serve.ts <its production start
  command>` on the port you pass. `start_site` sets `BASE_PATH` to the app's address and the wrapper hands it to the
  server and carries a private app's ticket; the Developer built the app for that base. Nuxt: build, then
  `bun /usr/local/share/serve/site-serve.ts bun .output/server/index.mjs`, port 3000; do not run `.output` directly.
  Never put a server through `site-serve` that was not built for a base path: it asks for `<BASE_PATH>/...` and
  would answer 404 to everything.
- Anything else: the command the project documents.
Those commands are for `start_site`'s `command` field. Starting again replaces the running app.

## 2. Check it
- `start_site` loads the home page and the pages it links to, and lists every link that would fail. A list that is
  not empty is not ready: put the exact links in your report so the Developer can fix them.
  It also opens the home page in a real browser at the app's address, the way the person's browser does (no login
  needed), and reports a page with no styling and any file the app answered badly. It also reports a stylesheet that is empty, answered with a web page, or never built (it still holds `@tailwind`
  or `@apply`), and files named inside a stylesheet. Send those to the Developer too: they are why a page shows as plain HTML.
- The app runs in its own container, and a shell command (`curl 127.0.0.1`, a port scan) cannot reach it. Do not
  check it that way and do not call it failed because a shell command cannot see it. `start_site` is the check: it
  starts the app, loads the home page and the pages it links to, and says the app is running when they answer.
  A "running" answer with an empty list of broken links is a pass; say so.
- In the preview there are no cookies, so a login cannot stay signed in there. Say that, and do not call it a fault.
- Never weaken a security setting to make the preview work.

## 3. When it will not start
After two honest tries (another port, the build redone, the log read), stop. Report
`Stuck: <what you tried, the exact error, the question to look up>`. The orchestrator has the Researcher look up the
proper way and sends you the job again with what it found. Do not try variations forever.

## 4. Report
Say whether the app is running in Sites, on which port, the command you used, what you built or ran, and every link
or page that failed. Never say it works without having seen it answer, and never give a local address.
