# Using the app

## Pages

The app is a single-page Svelte app served by the Go binary: `/` is the landing page, `/login` and `/register` are the sign-in and create-account screens, and `/app` is the workspace: home, an inbox of things that need you, projects with their tasks, a task page (conversation plus plan, proof, and changes), and settings (models, persona and rules, connectors, usage, account). Press Ctrl/Cmd+K anywhere in the workspace to search and jump. Everything here is real except notifications and data export, which are marked as coming soon.

## Working with the AI team

1. **Add a model.** Under Settings, AI models, add Anthropic, OpenAI, Google Gemini (key from [Google AI Studio](https://aistudio.google.com/apikey)), or an OpenAI-compatible gateway such as [9router](http://localhost:20128/v1) (use its address and key). Hosted providers use their fixed official address, and a gateway must run on the same computer as this app (`localhost`); the app refuses to connect to any other address, so nobody can point the server at your internal network. The key is stored encrypted and checked right away.
2. **Create a project.** Each project gets its own folder under `DATA_DIR/projects`; the AI can only touch files inside it.
3. **Describe what you need** on Home. The AI makes a plan, works step by step, and shows every step. Editing, deleting or moving files, running commands and starting a web app need your OK first, unless you turn on **Skip approvals** (see below).
4. **Review.** When it finishes you see what changed and what was checked. The proof says "Proven to work" only when the app itself ran a check that passed after the last change, and every change can be undone.

To let the app verify the AI's work, add **checks** on the project's Checks tab: named commands such as `go test ./...`. The AI can run the checks you list and, when you allow it, any command in the sandbox once you switch **Run a command** on. It asks before every run, and "Proven to work" appears only when a check passed after its last change. Checks run on this computer with the app's rights, so add only commands you trust.

## One set of switches

Settings, **Spaces & tools** is the single list of tools, and it applies to the tasks in this app and to the AI apps you connect (ChatGPT, Gemini and others) alike. A tool that is off cannot be used by anyone: the AI in a task is not even told it exists. A tool that is on is allowed, and the AI still asks you before risky steps in a task.

**Skip approvals** (at the top of each space) lets the AI use every tool that is on in that space without asking you first. It never turns a tool on: a tool that is off stays off. Looking around is on from the start; changing files, running commands and starting a web app stay off until you turn them on, so a new account has to switch them on before tasks can change files.

Each AI app you connect works in a project of its own named after it, so ChatGPT's work is in a project called ChatGPT and Gemini's in one called Gemini. They are ordinary projects: you can open them, see their files and run tasks in them, and they are marked "Connected app" in the list. Settings, Activity shows which app did what.

## Sites

When the AI builds a web app and runs it, whether in a task here or from a connected app such as ChatGPT, it appears under **Sites** (and in the sidebar while it runs). Open it from there or from the link in the task, make it public if you want to share it, and press Stop when you are done. See [sites.md](./sites.md).

## Your data

Under Settings, Account you can change your password (other devices are signed out), download everything as a zip (settings, projects with their files, tasks with every step, usage; saved AI keys are never included), delete all tasks, or delete the account with its projects and files after confirming your password. Settings, Notifications can alert you in this browser, while the tab is in the background, when a task needs you or fails.
