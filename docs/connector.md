# Connecting AI apps

The server can act as a connector for AI apps such as ChatGPT. Each connector is a "space" with its own address, for example `https://your-domain.com/api/coding`. Give ChatGPT that address and it finds everything else on its own: how to sign in, which permissions exist, and how to register itself.

## What the coding space can do

Every person gets their own coding folder (`DATA_DIR/workspaces/<account>`); nobody can read or change anyone else's files, and the folder is deleted with the account. Every check runs in its own throwaway Docker container, removed when the check ends. It sees only your coding folder (as `/workspace`, the one place it can write), a private `/tmp`, a private download cache, and the programs in the sandbox image, which every person shares (each tool is stored once on disk). It can reach package registries only, cannot see your other files or processes, runs without privileges and with Docker's default system-call filter, and is held to a memory, CPU, process, file-size and workspace-size limit.

Each tool can be switched on or off under Settings, Spaces & tools. Looking around is on from the start; changing files and running tasks stay off until you turn them on.

| Group | Tools |
| --- | --- |
| Look around (on) | list files, find files by name, read a file or some lines of it, search code, list tasks |
| Change files (off) | edit, write, delete and move files |
| Run tasks (off) | run a named task |

Every step an AI app takes is listed under Settings, Activity. Before a file is edited, written over, deleted or moved, its old content is saved, and the Undo button there puts it back. Undo is refused if the file changed again afterwards, so it never overwrites newer work.

Tasks are commands you approve in advance, in `.workspace/tasks.json` inside the workspace folder. The AI app can only ask for a task by name; it cannot see or change this file, and it cannot run anything else.

```json
{
  "tasks": {
    "test": { "description": "Run the tests", "command": ["go", "test", "./..."], "timeout_seconds": 300 },
    "lint": { "description": "Check the code style", "command": ["npm", "run", "lint"] }
  }
}
```

A task runs with the same rights as the app itself, one at a time, with a time limit and only the last 64 KB of its output kept. Because an AI app that can change files could also change a script that a task runs, changing files and running tasks are separate permissions. Turn on only what you need.

### Trying it from your computer

To try it, run the app, expose it with a tunnel (for example `cloudflared tunnel --url http://localhost:8081`), and start the app with `CONNECTOR_URL` set to the tunnel address. In ChatGPT, turn on developer mode, add a connector with `<CONNECTOR_URL>/api/coding`, and sign in when asked. You can see and remove connected apps under Settings, Connected apps. Each tool has a switch under Settings, Spaces & tools; looking at files is on from the start and changing files or running checks stay off until you turn them on. The coding space never shows `.env`, `.git`, or key files.
