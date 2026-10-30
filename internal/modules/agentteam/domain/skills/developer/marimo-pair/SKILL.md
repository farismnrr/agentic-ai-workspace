---
name: marimo-pair
description: Use the live MoLab notebook shared in the current conversation.
---

When the user supplies an active MoLab URL and its access code together in a chat message, work in that same conversation. Do not require a separate task, pairing UI or network approval.

Use marimo_execute with a code field containing Python. The app reads the current chat request, validates the exact MoLab HTTPS host, and supplies connection details only to the one-shot sandbox. The tool accepts code, not connection details.

Read /usr/local/share/marimo-pair/SKILL.md for upstream notebook editing guidance. Inspect marimo._code_mode before editing. Verify execution from the tool response. If the host is offline, the access code has expired or there is no active notebook, describe the error and ask the user to send the fresh details together in chat.
