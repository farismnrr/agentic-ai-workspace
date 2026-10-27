# Tests

Run the built-binary API and MCP scenarios with `make blackbox`.
The harness creates disposable data and removes it when each scenario ends.
The retained Go white-box inventory is in `whitebox/registry.md`.

The black-box suite needs `curl`, `jq`, `python3`, and `unzip`.
Run `make smoke` for the built-binary check. Image and sandbox checks need Docker.
Run `make e2e FLOW=projects` for the projects browser flow against the built
binary; it needs Chromium.

Run `make e2e FLOW=sites` or `make e2e FLOW=thread` for those browser flows.
They start retained Go fixture hosts because they need a scripted model and
Docker with the configured sandbox image. The registry checker runs with
`scripts/check-guardrails.sh`.
