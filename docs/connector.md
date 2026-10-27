# Connecting AI apps

Let AI apps like ChatGPT work on your projects through this server.

- Each connector has its own address, such as `https://your-domain.com/api/coding`.
- Give that address to the AI app.
- It finds out how to sign in and what it may do, then registers itself.

## Try it from your computer

1. Start a tunnel to the app's port, for example `cloudflared tunnel --url http://localhost:8081`. It prints a public address.
2. Start the app with `CONNECTOR_URL` set to that address.
3. In ChatGPT, turn on developer mode and add a connector with `<CONNECTOR_URL>/api/coding`.
4. Sign in when asked.

See and remove connected apps under Settings, Connected apps.

## Where the AI app works

Each connected app works in its own project, named after it (ChatGPT works in "ChatGPT").

- It is a normal project: you can open its files and run checks in it.
- No one can see anyone else's projects.
- Projects are deleted with the account.

## Following the AI app's work

Each connected-app conversation shows as a task beside the work done in the app.

- For substantial work, the AI app can start a separate work session and choose
  the roles it uses. These names describe its work; no other AI agent is started.
- Plan shows its steps. Documents separates research from requirements.
  Team shows only roles it declared or used.
- Changes shows edited files and offers Undo when the server saved a reversible
  change. Files shows the project's current files.
- The timeline shows conversation messages when the AI app saves them, plus
  work steps and results. An app that never calls the work tools still gets the
  simple timeline used before.
- In a work session, saved user and assistant messages remain chat bubbles.
  `saved` is returned only after the thread event commits.
- You can message an external task to ask the workspace orchestrator for help.
  It opens one linked internal chat in the same project; the external task's
  timeline and lifecycle remain separate. The chat includes the external task's
  request, plan, research, requirements and reported progress, with the external
  assistant's claims labelled separately from server-recorded evidence. The
  context is bounded and clearly separated from your message. If its linked chat
  is archived, the next message opens a fresh conversation.
- Running work steps update in place while the thread is open, including after
  a reconnect. New work sessions appear in the task list as soon as they start.
- Evidence labels what the server can support: the AI app's own progress report,
  file changes the server saw, and commands the server ran with their exit codes.
- The server cannot see activity outside its tools or the AI app's private
  thinking. It asks the app to save your request and its answer, but the app may
  skip either message.
- Stop ends a work session, but cannot interrupt a command already running.
- “Proven to work” means a command exited with code 0 after the last change. It
  does not mean the work itself is correct.
- Settings, Activity lists every tool call, even when the app did not save a
  conversation message.

## Working in your other projects

Tell the AI app which project you mean, such as "work in sensio". It lists your projects and uses the Project ID.

- Every active project of yours is open to your connected apps, by its Project ID.
- Without a project, the app stays in its own project ("ChatGPT").
- It can read, edit, run checks and start the web app of the project you named.
- The same tool switches and approvals apply as in its own project.
- The web app still opens from Sites and stays private.

Every check runs in its own throwaway container:

- It sees only that project's folder, as `/workspace`, plus a private `/tmp` and download cache.
- It reaches only package registries, and GitHub if you connected it.
- It also reaches your project's database, if an admin saved one in Settings and the project uses it. Nothing else on your network.
- It can't see your other files or processes.
- It runs without privileges, with limits on memory, CPU, processes, file size and workspace size.

The coding space never shows `.env`, `.git` or key files.

## What it can do

Turn each tool on or off under Settings, Spaces & tools. The switches are the same ones your own tasks use. See [using-the-app.md](./using-the-app.md#tools-and-approvals).

| Group | Tools | Default |
| --- | --- | --- |
| Look around | List, find and read files, search code, list checks | On |
| Change files | Edit, write, delete and move files | Off |
| Run things | Run a named check or a command, start the web app it builds | Off |

### Undo

Settings, Activity lists every step an AI app takes.

- Before a file is changed, deleted or moved, the old content is saved.
- Undo puts it back.
- Undo is refused if the file changed again since, so it never overwrites newer work.

## Named checks

A named check is a command you approve in advance. It lives in `.workspace/tasks.json` in the project folder (the file calls it a task).

- The AI app can run a check by name.
- It can't see or change this file, and it can't run anything else.
- Checks run one at a time, with the app's rights and a time limit.
- Only the last 64 KB of output is kept.

```json
{
  "tasks": {
    "test": { "description": "Run the tests", "command": ["go", "test", "./..."], "timeout_seconds": 300 },
    "lint": { "description": "Check the code style", "command": ["npm", "run", "lint"] }
  }
}
```

An app that can change files could also change a script a check runs. That is why changing files and running checks are separate permissions. Turn on only what you need.
