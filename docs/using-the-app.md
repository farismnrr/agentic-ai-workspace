# Using the app

## Pages

The app is a single-page Svelte app served by the Go binary: `/` is the landing page, `/login` and `/register` are the sign-in and create-account screens, and `/app` is the workspace: home, an inbox of things that need you, projects with their tasks, a task page (conversation plus plan, proof, and changes), and settings (models, persona and rules, connectors, usage, account). Press Ctrl/Cmd+K anywhere in the workspace to search and jump. Everything here is real except notifications and data export, which are marked as coming soon.

## Working with the AI team

1. **Add a model.** Under Settings, AI models, add Anthropic, OpenAI, Google Gemini (key from [Google AI Studio](https://aistudio.google.com/apikey)), or an OpenAI-compatible gateway such as [9router](http://localhost:20128/v1) (use its address and key). Hosted providers use their fixed official address, and a gateway must run on the same computer as this app (`localhost`); the app refuses to connect to any other address, so nobody can point the server at your internal network. The key is stored encrypted and checked right away.
2. **Create a project.** Each project gets its own folder under `DATA_DIR/projects`; the AI can only touch files inside it.
3. **Describe what you need** on Home. The AI makes a plan, works step by step, and shows every step. Deleting or moving files and running checks need your OK first; choose "Ask me before every change" to be asked about edits too.
4. **Review.** When it finishes you see what changed and what was checked. The proof says "Proven to work" only when the app itself ran a check that passed after the last change, and every change can be undone.

To let the app verify the AI's work, add **checks** on the project's Checks tab: named commands such as `go test ./...`. The AI can run the checks you list and, when you allow it, any command in the sandbox (Settings, Spaces & tools, Run a command). It asks before every run, and "Proven to work" appears only when a check passed after its last change. Checks run on this computer with the app's rights, so add only commands you trust.

## Your data

Under Settings, Account you can change your password (other devices are signed out), download everything as a zip (settings, projects with their files, tasks with every step, usage; saved AI keys are never included), delete all tasks, or delete the account with its projects and files after confirming your password. Settings, Notifications can alert you in this browser, while the tab is in the background, when a task needs you or fails.
