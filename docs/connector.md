# Connecting AI apps

The server can be a connector for AI apps like ChatGPT. Each connector has its own address, such as `https://your-domain.com/api/coding`. Give that address to ChatGPT. It finds out how to sign in, what it may do, and registers itself.

## Try it from your computer

1. Run the app and expose it with a tunnel, for example `cloudflared tunnel --url http://localhost:8081`.
2. Start the app with `CONNECTOR_URL` set to the tunnel address.
3. In ChatGPT, turn on developer mode and add a connector with `<CONNECTOR_URL>/api/coding`.
4. Sign in when asked.

You can see and remove connected apps under Settings, Connected apps.

## Where the AI app works

Each connected app works in its own project, named after it (ChatGPT works in "ChatGPT"). It's a normal project: you can open its files and run tasks in it. No one can see anyone else's projects, and projects are deleted with the account.

Every check runs in its own throwaway container, which:

- sees only that project's folder, as `/workspace`, plus a private `/tmp` and download cache
- reaches only package registries (and GitHub, if you connected it)
- can't see your other files or processes
- runs without privileges, with limits on memory, CPU, processes, file size and workspace size

The coding space never shows `.env`, `.git` or key files.

## Tools

Turn each tool on or off under Settings, Spaces & tools. The same switches apply to tasks in the app. "Skip approvals" lets the AI use every tool that's on without asking.

| Group | Tools | Default |
| --- | --- | --- |
| Look around | List, find and read files, search code, list tasks | On |
| Change files | Edit, write, delete and move files | Off |
| Run things | Run a named check or a command, start the web app it builds | Off |

Settings, Activity lists every step an AI app takes. Before a file is changed, deleted or moved, the old content is saved, and Undo puts it back. Undo is refused if the file changed again since, so it never overwrites newer work.

## Tasks

Tasks are commands you approve in advance, in `.workspace/tasks.json` in the project folder. The AI app can run a task by name. It can't see or change this file, and it can't run anything else.

```json
{
  "tasks": {
    "test": { "description": "Run the tests", "command": ["go", "test", "./..."], "timeout_seconds": 300 },
    "lint": { "description": "Check the code style", "command": ["npm", "run", "lint"] }
  }
}
```

Tasks run one at a time, with the app's rights and a time limit. Only the last 64 KB of output is kept.

An app that can change files could also change a script a task runs. That's why changing files and running tasks are separate permissions. Turn on only what you need.
