# Tests

Run the built-binary API and MCP scenarios with `make blackbox`.
The harness creates disposable data and removes it when each scenario ends.
The retained Go test inventory is in `go/registry.md`.

The black-box suite needs `curl`, `jq`, `python3`, and `unzip`.
Run `make smoke` for the built-binary check. Image and sandbox checks need Docker.
Run `make volume-isolation` for the Docker volume, socket, vendor and grant checks.
Run `make e2e FLOW=projects` for the projects browser flow against the built
binary; it needs Chromium.

Run `make e2e FLOW=sites` or `make e2e FLOW=thread` for those browser flows.
They start retained Go fixture hosts because they need a scripted model and
Docker with the configured sandbox image. Run `test/go/check-registry.sh` to
check the registered Go tests; `scripts/check-guardrails.sh` runs it too.

Run `make e2e FLOW=lifecycle` for the external work lifecycle in Chromium.
It seeds work through MCP and uses the built binary without Docker. The default
`make e2e` run includes this flow with the other browser scenarios.
