# PRD mini — MoLab from chat

**Status:** Implemented in code; live E2E not yet tested.

User supplies the active MoLab HTTPS URL and access code together in an ordinary chat. The internal agent calls marimo_execute with Python code; the app finds both values in the same latest matching user message. No dedicated MoLab task, form, memory pairing store or manual EGRESS_GRANTS approval.

Only the exact sb-*.sb.molab.run HTTPS host is allowed for the isolated one-shot sandbox via the existing public-only proxy. The access code is passed to the script as a transient environment value, never a command argument or Project Env. Chat messages themselves remain in task history. MoLab owns token expiry and revocation; connection errors tell the user to provide fresh session details.

Tests: parser security checks, pinned marimo-pair mock HTTP smoke, Go build, frontend verify, GHCR app/sandbox publish. Live notebook toast + termination and invalid-token behavior remain to be verified on a real notebook.
