#!/bin/sh
# Start script for the agent's commands and sites. The container has no network of its own.
#  - Registry access: a Unix socket to the server's registry-only proxy is made reachable as
#    127.0.0.1:3128 (the address HTTPS_PROXY points to).
#  - Sites: when SITE_PORT is set, the app's local port is published as a Unix socket in the folder
#    the server mounted at /run/site, which is the only way the server can reach the app.
#  - Screenshots: when CAPTURE_PORT is set, that socket is made reachable as a local port for the browser.
set -eu

if [ -S /run/egress/p.sock ]; then
  socat TCP-LISTEN:3128,bind=127.0.0.1,reuseaddr,fork UNIX-CONNECT:/run/egress/p.sock &
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
