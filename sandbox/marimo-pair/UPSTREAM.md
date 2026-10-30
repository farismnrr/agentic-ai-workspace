# Bundled marimo-pair skill

Source: https://github.com/marimo-team/marimo-pair
Pinned upstream commit: `2f1d7fffa50607dc9d8660cb931f4470707f8697`.
License: Apache-2.0, copied as `LICENSE`.

AI Code changes: baked-in absolute script location; forbid the CLI `--token` argument; pass bearer credentials to curl through a private temporary mode-0600 header file rather than process arguments. The upstream shell execution behavior otherwise remains pinned.

Note: this vendored code does not implement task-scoped credential storage. The caller must supply `MARIMO_TOKEN` securely and must not use project-wide variables as a replacement for a scoped secret.