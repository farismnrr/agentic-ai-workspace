#!/bin/sh
# Start script for the agent's commands and sites. The container has no network of its own.
#  - Registry access: a Unix socket to the server's registry-only proxy is made reachable as
#    127.0.0.1:3128 (the address HTTPS_PROXY points to).
#  - Sites: when SITE_PORT is set, the app's local port is published as a Unix socket in the folder
#    the server mounted at /run/site, which is the only way the server can reach the app.
#  - Screenshots: when CAPTURE_PORT is set, that socket is made reachable as a local port for the browser.
set -eu

# The image ships the pinned marimo-pair skill. Vendor CLIs discover it from
# their per-container HOME; never install skills from the internet at runtime.
# HOME=/tmp is a private tmpfs for each disposable task container.
if [ "${HOME:-}" = /tmp ] && [ -d /usr/local/share/marimo-pair ]; then
  mkdir -p /tmp/.agents/skills /tmp/.claude/skills
  [ -e /tmp/.agents/skills/marimo-pair ] || ln -s /usr/local/share/marimo-pair /tmp/.agents/skills/marimo-pair
  [ -e /tmp/.claude/skills/marimo-pair ] || ln -s /usr/local/share/marimo-pair /tmp/.claude/skills/marimo-pair
fi

if [ -S /run/egress/p.sock ]; then
  socat TCP-LISTEN:3128,bind=127.0.0.1,reuseaddr,fork UNIX-CONNECT:/run/egress/p.sock &
fi

# Exact targets the server grants this project, such as its database and model gateway (TUNNEL_TARGETS, host:port
# separated by spaces): each is opened on the same loopback port and passed through the proxy socket; its own name
# resolves here (/etc/hosts), so the saved address works as it is.
if [ -n "${TUNNEL_TARGETS:-}" ] && [ -S /run/egress/p.sock ]; then
  for target in $TUNNEL_TARGETS; do
    socat "TCP-LISTEN:${target##*:},bind=127.0.0.1,reuseaddr,fork" "PROXY:127.0.0.1:${target},proxyport=3128" &
  done
fi

if [ -n "${SITE_PORT:-}" ] && [ -d /run/site ]; then
  socat UNIX-LISTEN:/run/site/s.sock,fork,mode=0660 "TCP:127.0.0.1:${SITE_PORT}" &
fi

# Screenshots: the server's capture command runs in its own container and reaches the app through the same
# socket, as a local port, because the container has no network.
CAPTURE_SOCKET="${CAPTURE_SOCKET:-/run/site/s.sock}"
if [ -n "${CAPTURE_PORT:-}" ] && [ -S "$CAPTURE_SOCKET" ]; then
  socat "TCP-LISTEN:${CAPTURE_PORT},bind=127.0.0.1,reuseaddr,fork" UNIX-CONNECT:"$CAPTURE_SOCKET" &
fi

exec "$@"
