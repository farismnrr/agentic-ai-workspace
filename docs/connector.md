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

Each conversation of a connected app shows up as a task, next to the ones the AI team ran.

- Every step is written down: reading, searching, editing and running commands.
- A command that takes a while shows as "Running: …" while it runs, and the task shows as working until it ends.
- Your request and the app's answer appear only if the app saves them. The server asks it to, but ChatGPT and Gemini may skip it.
- Settings, Activity lists every call, whether or not the app saved the conversation.

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
