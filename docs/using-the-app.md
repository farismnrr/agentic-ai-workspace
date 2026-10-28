# Using the app

Add a model, create a project, and describe what you need. The AI shows every step and asks before it changes anything.

## Pages

| Address | What it is |
| --- | --- |
| `/` | Landing page |
| `/login`, `/register` | Sign in, create an account. If you are already signed in, they take you straight to the dashboard (or to the page in `?next=`) |
| `/app` | Your workspace: home, inbox, projects, tasks and settings |

Press Ctrl/Cmd+K anywhere in the workspace to search.

On a first open or reload, the app shows a brief “Memuat...” loading screen.
Projects and task lists show placeholders while their data loads.

Pin a task from its sidebar row to keep it in **Pinned** above Recent.
Pinned tasks stay there across projects and reloads. Choose the pin button
again to remove it; the task then returns to Recent.

## Archive and delete tasks

Use **Archive** in Tasks or a project's Tasks tab to hide
work you no longer need in active lists. Archived tasks leave Recent, Pinned,
Tasks and needs-you counts, and appear in **Settings → Workspace → Archive**.

Archiving a running task stops it. Its task page becomes read-only. Restore it
from the Archive list to return it to active lists; its pin is kept.

Select several tasks to archive, restore or delete them together. **Select all
shown** applies to the rows currently displayed. In Archive, restore tasks to
use them again or choose **Delete permanently** to remove them for good.

Permanent deletion asks you to confirm the number of tasks. Running tasks are
stopped first, and the confirmation warns you when that will happen. Deletion
cannot be undone. It removes task records, screenshots and task documents;
project-shared files in the Files tab are kept.

## Attach files to a task

Use the paper-clip, drag files onto the message box, or paste an image. Uploads
start as soon as files are added. Send waits until each upload is ready. If one
fails, choose Retry or remove it. Photos show a thumbnail after upload; select
it to open a larger preview. Other files show their name and size.

You can attach up to 10 files to one message. Remove a file with its button
before sending. After sending, photos appear in your message bubble and other
files show as chips; select one to open it.

Photos (PNG, JPEG, WebP, GIF, SVG), PDF, ZIP and text files (TXT, Markdown,
CSV, JSON, LOG) can be up to 25 MB each. Videos (MP4, WebM, MOV) can be up
to 100 MB. The app checks file contents as well as the filename.

ZIP files are stored without extraction. Unsafe paths, links, encrypted archives,
nested archives and decompression bombs are rejected. A ZIP may contain up to
256 entries and 100 MB of uncompressed data. Treat archive contents as untrusted.

Files are saved in the project's `uploads/` folder and appear in the Files tab.
Videos can play and seek there, but the AI cannot watch or understand them.
The AI can inspect PNG photos and read PDFs and text files from the project.

## Your first task

1. Add a model under Settings, AI models (see [Models](#models)).
2. Create a project with Projects, New project. It gets its own folder under `DATA_DIR/projects`, and the AI can only touch files in it.
   - Step 1: choose "From a template" or "From a GitHub repository". Without a GitHub App on the server there is only the first, so this step is skipped.
   - "From a template" opens a gallery. Each template shows a screenshot: landing page, product website, dashboard, documentation, AI chat, portfolio, release notes, text editor, calendar. Its look is copied into the project; the login and data are still built for your app.
   - The gallery also has "Without a template": an empty project where the AI designs the look from your request, and you refine it by hand.
   - After a template (or none), choose "New project" (files stay in this workspace) or "New repository" (also creates a repository on your GitHub account, with a README; you pick public or private).
   - The name is a label. Type anything; it can be changed later and other projects may share it.
   - The dice button suggests a two-word name when you do not want to think of one.
   - The Project ID fills in from the name. Use Edit to choose your own: 6 to 30 lowercase letters, digits or dashes.
   - The ID is the address of your app (`/run/<id>/`), so it is unique and cannot be changed later. If it is taken, you see it right away, with a nearby ID to use in one click.
   - Then give it a name, and it opens.
   - "From a GitHub repository" starts from one of your repositories, with its history, and has no template. Type to search; the list scrolls.
   - Without GitHub connected, the GitHub choices stay visible but unavailable, with a link to connect GitHub.
   - If GitHub does not let the app create a repository, you are told. Create it on GitHub, share it with the app, then use "From a GitHub repository".
3. Describe what you need on Home. The model and effort come from Settings: each teammate has its own, and the default covers the rest.
   Long messages become project documents automatically. The complete text stays available to the AI and can be downloaded from the thread.
4. Expect a few questions first. The AI asks what you want before it starts, with a recommended answer for each, and skips questions for a small, clear request. Answer in your own words or pick an option.
5. If you asked for a browser extension, open **Settings → Workspace → Extensions** to install it. See [extensions.md](./extensions.md).
6. Before the AI tells you it is done, it saves the work in git on the branch `dev` and pushes it. Without a GitHub remote it only saves it here, and says so.
7. Watch the plan and the steps. The AI asks before it edits, deletes or moves files, runs commands, or starts a web app.
5. Review the result, then accept it or ask for changes.

## Reading a task

The task page has the conversation on one side and a panel on the other. On a phone or tablet, the panel is a row of tabs.

External tasks keep their progress in the external task thread. Messages sent there go to a separate internal
orchestrator conversation; the external assistant does not reply to them. After sending, use **Open conversation**
to see the reply. That conversation links back to the external task for context. Archived external tasks are read-only.
If no AI model is ready, choose a provider and model in **Settings → AI models**, then send again.

| Tab | Shows |
| --- | --- |
| Plan | The team's todo list and each step's owner and status |
| Documents | Plans, requirements and research for this task |
| Team | Jobs, who did them, and their reports (shown on every task) |
| Changes | Current Git changes, with file diffs and task history |
| Files | The whole project: open any file, edit it by hand and save. PDFs and pictures are shown in place, and any file can be downloaded |

About Changes:

- **Current Git changes** include edits made by you, the AI or shell commands.
  Staged and unstaged changes are shown separately against the current commit.
- Open a file's diff to compare it with `HEAD`. Deleted paths stay in the list.
- The Files editor marks added and removed lines against `HEAD`. Unsaved edits
  remain separate from the saved file status.
- When Git is clean, active changes and line marks clear. Task steps, evidence
  and Undo remain under **Task history**.
- Projects without Git tracking say that comparisons are unavailable.

About Files:

- **The project page** has the same Files tab, so you can read the code without starting a task.
- **Archive or delete a project** on its About tab. Archive hides it and keeps its files and tasks. Delete asks you to confirm, then removes the project with its files, tasks and history for good. It cannot be undone.
- **Live:** when the AI or a command changes a file, the tree and the open file update at once.
- **Your edits are safe.** If the AI changes a file you are editing, you choose: keep yours or see the new one.
- **Delete a folder:** hover a folder and press the bin. It removes everything inside, and it cannot be undone.
- **Hidden:** secrets such as `.env` files and keys never show.
- **Save** with the button or Ctrl/Cmd+S.

About the result:

- **"Proven to work"** appears only when a check the app ran itself passed after the last change.
- **"Verified"** marks checks the app ran. **"Claimed"** marks things the AI only said.
- The result summarizes the checks in plain language.
- **A question that needs no changes** is answered in the conversation and ends as Done, with no review step.
- **The header** shows which model and effort the task used, and how full the AI's memory is.

## Project environment

Open a project and choose its **Env** tab to manage environment variables.
Each project has its own list; variables are added only to runs in that project.
Other projects do not receive them.

- Names use uppercase letters, numbers and underscores. Values are limited to
  4 KB, with up to 50 variables per project.
- New variables are secret by default. Their values stay hidden after saving.
  A non-secret value can be shown again and edited.
- Saving an existing name replaces its value after confirmation. Deleting a
  variable removes it from later runs.
- The app encrypts secret values in its data folder. The agent receives them as
  environment variables for commands, checks and project sites. Names appear
  in this list. The app keeps values out of notes and model prompts, but a
  command such as `env` or `printenv`, or app output that echoes a value, can
  expose it in the task log. Avoid printing secret values.

### External database

The Env tab's **Database** section starts with **SQLite (default)**. It uses
the project's own SQLite file. Choose PostgreSQL, MySQL or MongoDB to create
an isolated connection from details saved under **Settings → Database**.
PostgreSQL and MySQL use `DATABASE_URL`; MongoDB uses `MONGODB_URL`.
The AI's commands and your running app can reach that database, and only that one.
The **Model gateway** section lets the project call an OpenAI-compatible API on this computer using your own API key. Account-sharing relays such as 9router are not supported.
The code then gets `OPENAI_BASE_URL` and, if the provider has a key, `OPENAI_API_KEY`.
When configured, the public CA is provided as `DATABASE_SSL_CA`.

Project apps can read `DATABASE_SSL_CA` as PEM text and pass it to their
database driver's TLS root certificate setting. The app does not write a
certificate file. It never sends the administrator's client certificate or
private key to project apps. Apps needing their own client certificate must
set their own Env values.

Kinds without saved settings are unavailable in the dropdown. Ask an
administrator to configure them in Settings. Older PostgreSQL, MySQL/MariaDB
and MongoDB connections remain visible as read-only legacy details. Choose
SQLite to detach one. No password is shown, and the external database and its
data are left untouched. New custom connections are unavailable.

Switching from a shared or legacy connection to SQLite asks for confirmation.
The app removes this project's connection and managed database Env values;
the data stays in its database. Schemas, databases and users stay on servers.

Choose one of these three options:

- **No TLS** uses no TLS.
- **CA certificate only**
  verifies the server hostname. Servers with self-signed certificates need
  that CA uploaded. Older Require and Prefer settings are migrated to CA only.
- **Full (CA + client certificate)**
  also sends a client certificate.

Choose files instead of pasting PEM text.
Each file can be up to 16 KiB.

Certificate values are encrypted at rest.
Settings never returns them; it reports which
values are saved. The public CA is shared with
project apps. TLS checks the server chain and
hostname.

Use **Test connection** to check the credentials from the app server. Tests
allow public, private-network and loopback database addresses. They block
link-local addresses, including `169.254.169.254`, and the app's own port.
Tests are limited to five per minute per signed-in person.

When the app runs in Docker, `localhost` means the app container. For a database
on the Docker host, try `host.docker.internal`; for another computer, use its
LAN IP and make sure the database accepts connections from the app host.

Only administrators can open **Settings → Database**
or change PostgreSQL, MySQL and MongoDB host, port,
database, user, password, TLS choice and certificates.
Password and certificate contents are write-only.
Other Settings pages configure each signed-in
person's own account. The project database API
reports which kinds are available. PostgreSQL can use
`POSTGRES_ADMIN_URL` when no settings are saved.

An administrator saves each database kind's host,
port, database, user, password, TLS choice and
optional certificates under **Settings → Database**.
Secret contents are write-only.
If no saved settings exist, the server can use `POSTGRES_ADMIN_URL` instead.
Use a dedicated PostgreSQL database because provisioning revokes `PUBLIC`
access to the `public` schema for that whole database. The admin account must
be allowed to create roles and schemas; some services limit them. MySQL uses
a separate database and grants project users access only to it. MongoDB uses
a separate database and a user with `readWrite` access only to it.

Choosing a kind provisions a per-project schema or database and login.
Reconnecting resets that project's generated password. Switching kinds
replaces the connection and managed Env values; it never drops the previous
schema, database, user or data.

The agent receives the connection string as an environment variable, but its
sandbox cannot connect directly to the database. The test checks access from
the app server only. The app does not run queries, inspect schemas or manage
database migrations or backups.

## Models

You can add Anthropic, OpenAI, Google Gemini (key from [Google AI Studio](https://aistudio.google.com/apikey)), or a legitimate local OpenAI-compatible API using your own API key. Account relays are unsupported.

- A gateway must run on the same computer as the app (`localhost`). The app refuses any other address, so nobody can point it at your internal network.
- Keys are stored encrypted and checked right away.
- Models are grouped by provider.
- With a gateway, each provider it serves is its own group. Codex, Claude Code and OpenCode have names. Others show the gateway's short name, such as `kr`.
- A provider added in the gateway appears within about 30 seconds, or at once when you press Refresh.
- Legacy 9router providers cannot make requests or be enabled. They remain listed as disabled so you can delete them yourself, without losing unrelated data.

### Three model tiers

Pick a model and an effort once for each tier, under **Settings → Models**.

| Tier | Use it for |
|---|---|
| Thinking | Hard problems such as design, security and review. A strong model at high effort. |
| Default | New tasks, and teammates on the default tier. |
| Fast | Quick, simple jobs. A small model at low effort. |

- Under **Settings → Agent team**, give each teammate a tier, or a model of its own (Custom).
- Change a tier's model once, and every teammate on it follows.
- The orchestrator follows current Settings when a task starts or continues, including an old task. A run already working keeps its model until its next run.
- The thread records model changes. Jobs already assigned to teammates keep their chosen model when resumed.
- A tier you leave empty uses the Default model.
- The task thread shows the model and the effort each teammate was given, such as "Haiku · medium".

Two settings shape how a model works:

| Setting | What it does |
| --- | --- |
| Effort | Controls how much thought the AI uses. Higher levels take longer and cost more. **Auto** lets the lead choose for each teammate. Your explicit default or role choice wins. Without a team, Auto uses the model default. |
| Context window | How much the model keeps in mind at once. Near the limit, the AI summarizes its notes and keeps going. |

The context window comes from what the provider reports, else 200K, or 1M for names ending in `[1m]`.
If a model still says a request is too long, the AI summarizes once and tries again.
With Claude or ChatGPT on your own plan, the program keeps its notes short itself, by the size you set for the model here; the meter shows what it reports.

## Your own AI plan

Sign in with a Claude (Pro or Max) or ChatGPT (Plus or Pro) plan you already pay for, instead of adding an API key.

1. Open **Settings → Models**, press **Add**, and choose **Anthropic** (or **OpenAI**).
2. Switch from **API key** to **Your plan**, press **Sign in with Claude** (or ChatGPT), then **Open Claude** (or **Open ChatGPT**).
3. Claude shows a code: paste it into the window. ChatGPT shows a code in the window: type it on the ChatGPT page.
4. The plan now shows under **Connected providers**, like any other provider, with its own on/off switch. Its models appear in every model list, with an **Effort** level per model. Claude lists Default, Fable, Opus, Sonnet and Haiku. ChatGPT lists the models Codex offers.

- **Your sign-in stays private.** You sign in on the provider's own page. The app never sees your password or token.
- **Each person signs in on their own.** Plans are never shared between people.
- **Not signed in?** A task stops with a note telling you to sign in, and Continue works afterwards.
- **Today** a task on your plan works in one project, one step at a time. There is no Undo and no per-step approval yet.
- **Teams:** the lead and teammates can use your plan, set under **Settings → Agent team**. The lead only looks at files. The server assigns jobs, asks questions and keeps the plan, so a plan lead works like any other model.

## Agent team

An agent team works like a small software company. A lead takes your request and hands each part to the right teammate, who reports back.

- Every task has the team. There is no switch to turn it off.
- A question, or a small change, never needs the whole team: the lead answers it or hands it straight to the Developer.
- The lead decides who to call and when. Only the lead talks to you.
- The lead never changes anything itself, even a one-line fix. It reads, plans, asks you and hands every change to a teammate.
- A teammate you did not need is simply never called.
- Teammates do not chat with each other. The lead passes on what each one needs, so nobody works from a different picture.
- Once the shared app shell is in place, the lead can assign independent feature slices to multiple Developers in parallel, each with its own folder scope.
- Read and verification checks (Researcher, QA Tester, Reviewer, Security, Architect) can also run in parallel when their work is independent.

The lead sizes your request first and tells you:

| Size | What happens |
| --- | --- |
| Question | The lead answers. No plan, no team. |
| Small | A quick change to something that exists, such as a typo or a title. The lead hands it straight to the Developer, with no Planner and no plan list. |
| Medium | Anything new, such as a page or a script, or three or more files. The Planner asks you a few questions and writes a short PRD, then the Developer builds and QA checks. |
| Large | The Planner plans the same way. Research, requirements and design are added only if needed. A Designer shapes anything you will see. A Reviewer looks it over. |

Research comes first on medium or large tasks, then again after the PRD for the stack and visual design. The Researcher checks real-world facts about a business, market, style or technology, then writes short notes with sources.
When a teammate gets stuck, it says so instead of struggling alone. The Orchestrator asks the Researcher how it is done properly, then sends the job again with the answer.
Only the DevOps teammate starts your app, and always in Sites, so you can open it from any computer. Nobody gives you a 127.0.0.1 or localhost address.

You can read them under the Plan tab, in Research notes. Later teammates build from them.

Under **Plan → Project documents**, open plans, requirements, research rounds and long messages from any task in the same project.
The app saves these files automatically under `project-notes/`; a new task can read them without inheriting another task's conversation.
Old plans and research are exported when the project is used or its document list is opened. Check their dates and status before reusing them.
Earlier approval does not approve a new task.

A team task always plans first. The team plans, then stops and waits for your yes:

- It researches, turns your request into requirements, asks you what it still needs, and writes the PRD.
- Then it asks "Start building?". Nothing is built, run or started until you say yes. Say what to change and it revises the PRD.
- After your yes the team only executes: build, security check, tests and a check of the final code.
- A small change never needs a plan. The lead sizes the request first and hands a small one straight to the Developer.

A web app the AI builds uses Nuxt, Nuxt UI (one consistent set of components) and SQLite, written in TypeScript and run on Bun, unless you ask for another stack. It is not a single page. Unless you say otherwise it has:

- A public landing page, with what it is, the benefits, how it works, pricing or a call to action, and a short FAQ.
- Register and login with its own forms, and pages behind the login.
- An app with a dashboard: key numbers, charts of your real data, recent activity and a clear first step when it is empty.
- Each feature as a full flow, plus settings and a friendly "not found" page.

If you started the project from a template, the Designer adapts that one to fit. In an empty project it designs the look itself and copies no template. Only the look comes from a template; the login, data and server are built for your app.

Pictures come from you first:

- The Designer lists each picture with its size, format and a prompt you can paste into an image generator.
- The team asks once: upload them, or find free ones for you.
- Attach pictures with the paperclip in the message box. They are saved in the project's `assets/` folder.
- If you ask for free ones, only pictures you may use are chosen, and the credits stay in a file, never on the page.

Before a task is finished, a Reviewer looks at every screen on the running app, and the Designer looks at its own
screens first. The team builds one piece at a time, then styles every page.

For a medium or large task:

1. The Planner asks only what blocks the scope, such as who it is for or what to leave out.
   - It shows one short form. Pick one option, tick several, or type your own answer, then send.
   - For the look of a page, each choice comes with a picture. The one it suggests comes first.
   - At most three questions for medium, five for large.
2. It writes a short PRD: overview, goals, what is not included, acceptance criteria, requirements and a todo. It shows in the thread and under the Plan tab.
3. The lead asks once whether to start building. The team waits for your yes. Say what to change and the Planner revises it.
4. If you will see a screen or page, the Designer sets the look and builds it, and writes it down in `DESIGN.md` for the Developer.
5. Every teammate is shown the PRD with its job.
6. A screen ends with a design review by the Reviewer. Its list of fixes goes back to the Designer, at most two rounds.
7. The Designer and the Reviewer look at your running app through screenshots, on a desktop and a phone screen. You see each screenshot in the thread.
8. Work that includes a web page is never reported until its app is running. You get a link to open it.

When unsure, the lead picks the smaller size. There is no cap on how many jobs a team may hand out, as long as they keep getting somewhere.

| Role | What it does |
| --- | --- |
| Orchestrator (lead) | Plans, hands out work, answers you |
| Planner | Asks you what it needs in a short form, then writes the PRD. It follows a built-in playbook (its skill) that ships with the app |
| Designer | Sets the visual direction, writes `DESIGN.md` and builds the interface. It has built-in skills (design brief, frontend design, UI polish) and a library of craft rules, style families and page templates, and picks what fits the job |
| Researcher | Reads the project and looks things up on the web. It writes notes with sources before the PRD, the design or the build. It cannot change files |
| Media | Finds pictures you may use and edits them: resize, crop, convert, remove a background. Every picture it saves has its licence and credit written in `assets/CREDITS.md`, never on the page. It lists pictures that need no credit first |
| Documents | Creates PowerPoint, Word and PDF files, then previews each page. The result opens under Sites with a Download button. Files stay in `documents/`; the Files tab can also preview PDFs. |
| Business Analyst | Turns your request, and what the Researcher found, into requirements before the PRD |
| Architect | Designs the change, and checks the final code against the design, SOLID and Clean Architecture before the task is done. It cannot change files |
| Developer | Writes the code. A new app is built with Nuxt, SQLite and the latest stable versions, with its own register and login forms when it needs accounts |
| Security | Checks an app for known vulnerabilities, risky code and missing protections. An app with code is not started until it has reported. It cannot change files |
| QA Tester | Runs the checks |
| Reviewer | Reads the finished work |
| DevOps | Builds, scripts and running the app |

Under Settings, Agent team you can change each role:

- Pick its own model and effort. Leave it empty to use your default.
- Write its personality and way of working.
- For a role you add, choose what it may do: look at files, change files, run commands. Built-in roles keep the access that fits them.
- Turn a role off, or add your own. Built-in roles cannot be removed, and what you add is yours alone.
- Reset to built-in restores the default team, including roles added in later versions.

Good to know:

- Every step shows who did it, including the lead's own, with a card when work changes hands.
- The Team tab lists each job: who has it, which model, what was asked, how many steps, files and checks, and the report.
- The result summarizes checks, distinguishing verified runs from claims.
- A teammate still asks before changing anything, with the same Allow button.
- Checks any teammate ran after the last change count toward the result.
- A team task may run 400 model turns and 2 hours, shared by everyone on it. A teammate may take 60 turns per job.
- If a limit is reached, the task pauses with Continue. Nothing is lost: the work stays and the team carries on.
- If six jobs in a row change nothing, run no check and write nothing, the lead is told the team is stuck and must change its approach or tell you what blocks it.

## Checks

A check is a command a project defines, such as `go test ./...`. The AI runs checks to prove its work.

- Add them on a project's Checks tab.
- They run on this computer with the app's rights, so add only commands you trust.
- The AI can also run any command in the sandbox once you turn on "Run a command". It asks before every run.

## Per-task internet access (not yet active)

When this feature is approved for activation, a project's **Internet access** tab shows requests from its active tasks. Until then, the tab stays hidden and internet access works as before.

- An agent may request access to a public HTTPS hostname, but cannot approve it.
- Project owners and admins may approve, deny or revoke requests. You can also request access manually for a solo task.
- Approval covers one hostname and one task. The default is 24 hours; the maximum is seven days. Access ends on expiry, revocation or task completion.
- Requests show their creation time and remaining lifetime; expired requests stop offering decisions. The panel also identifies who decided a request.
- Successful requests and decisions appear in the task thread once; retries do not add duplicate notices. The activity log retains requester, decision actor, owner and task scope without the request reason. A pending request grants no access.
- Only HTTPS connections on port 443 are considered. The proxy verifies the TLS server name but does not decrypt traffic. Approving a host means trusting that destination.
- Existing static sandbox hosts and their access rules are unchanged.

## Tools and approvals

Settings, Spaces & tools has one list of tools. It applies to tasks in this app and to the AI apps you connect.

- A tool that's off can't be used. The AI isn't even told it exists.
- A tool that's on is allowed, but the AI still asks before risky steps.
- "Skip approvals" lets the AI use every tool that's on without asking. It never turns a tool on.
- On a new account, only looking around is on. Changing files, running commands and starting a web app stay off until you turn them on.

Connecting ChatGPT and other AI apps is covered in [connector.md](./connector.md).

## External work review

- A connected assistant chooses a role to start work. It can research, write a
  plan and save questions for you. Answer questions in the task thread.
- **Working** means the assistant has sent recent updates. The app shows a
  working indicator while it considers the session active.
- **Reported, not confirmed** means the assistant saved its summary but did not
  confirm the session with `complete_work`. Read the report before deciding.
- **Idle** means there have been no updates for over 30 minutes. The assistant
  may still be working; inactivity never means the work is finished or proven.
- **Ready for your review** means a reported session passed the configured
  cleanup window. Review the result and accept the task, or continue it.
- **Lost contact** means an unreported session passed that window and was
  closed as failed. Continue the task to pick it up again.
- **Mark complete** confirms reported or idle work and changes the task to Done.
  It records your confirmation, but does not add a “Proven to work” stamp.
- **Stop** ends an open work session and marks its task failed. It does not
  confirm that the requested result is complete.
- The flow is researcher → planner → your answer → ready for approval →
  approved → handed off. Small tasks can skip research when the assistant marks
  them as small.
- Review the submitted plan revision in its card. Approve it to hand its saved
  snapshot to the Orchestrator, or request changes for a new revision.
- Status icons include a text label, so status remains readable without color.
  Motion stops when your device requests reduced motion.
- Settings stays available on Archive and Extensions pages, with the current
  page selected in its sidebar.

## GitHub

Under Settings, Integrations, press Connect GitHub, approve, and choose which repositories the app may reach.

After that, the AI's commands in your projects act as you:

- They can clone those repos, commit under your name and push.
- The AI pushes only when you ask.
- No one else's projects can use your connection, and web apps never can.
- Disconnect removes it at once.

If you see "not set up", the server's owner has to add a GitHub App first (see [deployment.md](./deployment.md#github)).

## Sites

When the AI builds and runs a web app, it appears under Sites and in the sidebar.

- Starting one needs "Run a command" to be on.
- You can open it, make it public, or stop it.
- Details: [sites.md](./sites.md).

## Your data

Under Settings, Account you can:

- Change your password. This signs out other devices.
- Download everything as a zip. Saved AI keys are never included.
- Delete all tasks.
- Delete your account, with its projects and files.

Settings, Notifications can alert you in the browser when a task needs you or fails.
