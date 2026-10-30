---
name: marimo-pair
description: Pair with an approved running marimo/MoLab notebook from this task's isolated sandbox.
---

Use this skill only when the person explicitly asks to work with a marimo notebook. It grants neither network permission nor credentials.

1. Read the pinned upstream protocol with `run_command`: `cat /usr/local/share/marimo-pair/SKILL.md`. The official scripts and references are under `/usr/local/share/marimo-pair/`, baked into the sandbox image.
2. If the exact notebook host is blocked, call `request_egress` with only that hostname and a clear reason, when this tool is available. Wait for owner/admin approval. Never bypass the task egress proxy or widen the static allowlist.
3. A token-authenticated notebook requires `MARIMO_TOKEN` set securely in this task's execution environment. Never ask for tokens in chat, echo them, place them in code, put them in command arguments, or pass `--token`. If no scoped secret is available, report a blocker rather than impersonating a successful pairing.
4. After authorization, use `run_command` to execute `bash /usr/local/share/marimo-pair/scripts/execute-code.sh --url <the-user-provided-HTTPS-url> -c 'import marimo._code_mode as cm; help(cm)'`. This read-only inspection must succeed before other notebook code.
5. Follow the upstream marimo instructions; for durable changes, use `marimo._code_mode` and do not edit live notebook files directly. Never claim a toast, cell edit or successful connection without its returned result.

Only task roles with command execution may use this workflow. No vendor/agent may approve its own egress request.