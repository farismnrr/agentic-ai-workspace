#!/usr/bin/env bash

app_init() {
  if [[ -z ${APP_BIN:-} ]]; then
    make -C "$REPO_ROOT" backend-build
    APP_BIN="$REPO_ROOT/bin/agentic-ai-workspace"
  fi
  if [[ ! -x $APP_BIN ]]; then
    printf 'App binary is not executable: %s\n' "$APP_BIN" >&2
    return 1
  fi
  APP_TMP_DIR=$(mktemp -d)
  APP_DATA_DIR="$APP_TMP_DIR"
  APP_LOG="$APP_TMP_DIR/app.log"
  trap app_cleanup EXIT
}

app_free_port() {
  python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1", 0)); print(s.getsockname()[1]); s.close()'
}

app_start() {
  local attempt port ready
  for attempt in 1 2 3 4 5; do
    port=$(app_free_port)
    APP_PORT=$port
    APP_BASE="http://127.0.0.1:$APP_PORT"
    PORT="$APP_PORT" DATA_DIR="$APP_DATA_DIR" PUBLIC_URL="$APP_BASE" \
      "$APP_BIN" >>"$APP_LOG" 2>&1 &
    APP_PID=$!
    ready=0
    for _ in $(seq 1 50); do
      if ! kill -0 "$APP_PID" 2>/dev/null; then
        break
      fi
      if curl -fsS "$APP_BASE/api/health" >/dev/null 2>&1; then
        if kill -0 "$APP_PID" 2>/dev/null; then
          ready=1
        fi
        break
      elif ! kill -0 "$APP_PID" 2>/dev/null; then
        break
      fi
      sleep 0.2
    done
    if ((ready == 1)); then
      return 0
    fi
    if kill -0 "$APP_PID" 2>/dev/null; then
      kill "$APP_PID" 2>/dev/null || true
      wait "$APP_PID" 2>/dev/null || true
    fi
    if ((attempt < 5)); then
      printf 'App did not bind port %s; retrying with another port.\n' "$port" >&2
    fi
  done
  printf 'App failed to become healthy; log follows:\n' >&2
  cat "$APP_LOG" >&2
  return 1
}

app_stop() {
  if [[ -n ${APP_PID:-} ]] && kill -0 "$APP_PID" 2>/dev/null; then
    kill "$APP_PID" 2>/dev/null || true
    wait "$APP_PID" 2>/dev/null || true
  fi
  APP_PID=
}

app_restart() {
  app_stop
  app_start
}

app_cleanup() {
  app_stop
  if [[ -n ${APP_TMP_DIR:-} && -d $APP_TMP_DIR ]]; then
    rm -rf -- "$APP_TMP_DIR"
  fi
}
