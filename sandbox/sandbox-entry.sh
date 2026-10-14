#!/bin/sh
# Start script for the agent's commands. The container has no network of its own, only a
# Unix socket to the server's registry-only proxy; this makes it reachable as 127.0.0.1:3128
# (the address HTTPS_PROXY points to), then runs the command.
set -eu

if [ -S /run/egress/p.sock ]; then
  socat TCP-LISTEN:3128,bind=127.0.0.1,reuseaddr,fork UNIX-CONNECT:/run/egress/p.sock &
fi

exec "$@"
