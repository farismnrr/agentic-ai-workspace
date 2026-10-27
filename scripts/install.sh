#!/bin/sh
# Guided install (option A in docs/deployment.md): asks for each setting one by one, writes .env,
# then downloads the compose file and starts the app. Settings first, compose second.
#
#   curl -fsSL https://raw.githubusercontent.com/farismnrr/agentic-ai-workspace/main/scripts/install.sh | sh
#
# INSTALL_DIR   where .env and the compose file go (default ~/agentic-ai-workspace)
# CHANNEL       latest (stable, default) or dev (newest)
# INSTALL_TTY   where answers are read from (default /dev/tty, so piping into sh still works)
set -eu

REPO_RAW="https://raw.githubusercontent.com/farismnrr/agentic-ai-workspace"
CHANNEL="${CHANNEL:-latest}"
INSTALL_DIR="${INSTALL_DIR:-$HOME/agentic-ai-workspace}"
TTY="${INSTALL_TTY:-/dev/tty}"

case "$CHANNEL" in
  latest) BRANCH=main ;;
  dev) BRANCH=dev ;;
  *) echo "CHANNEL must be latest or dev" >&2; exit 1 ;;
esac
COMPOSE_FILE="compose.$CHANNEL.yaml"

exec 3<"$TTY" || { echo "Error: no terminal to read answers from" >&2; exit 1; }

say() { printf '%s\n' "$*" >&2; }
die() { say "Error: $*"; exit 1; }

# ask VAR "what it is" "example" "default" [check]
# Shows the question, reads one line, Enter keeps the default, and asks again if the check fails.
ask() {
  var="$1"; what="$2"; example="$3"; default="$4"; check="${5:-any}"
  say ""
  say "$var"
  say "  $what"
  [ -n "$example" ] && say "  Example: $example"
  while :; do
    if [ -n "$default" ]; then printf '  Value [%s]: ' "$default" >&2; else printf '  Value (Enter to skip): ' >&2; fi
    IFS= read -r answer <&3 || die "no terminal to read answers from"
    answer="${answer:-$default}"
    if valid "$check" "$answer"; then break; fi
  done
  eval "$var=\$answer"
}

valid() {
  case "$1:$2" in
    any:*) return 0 ;;
    port:*[!0-9]*|port:) say "  Please enter a number, like 8081."; return 1 ;;
    port:*) [ "$2" -ge 1 ] && [ "$2" -le 65535 ] && return 0; say "  A port is between 1 and 65535."; return 1 ;;
    url:) return 0 ;;
    url:http://*|url:https://*) return 0 ;;
    url:*) say "  It must start with http:// or https://"; return 1 ;;
    number:*[!0-9]*|number:) say "  Please enter a whole number."; return 1 ;;
    number:*) return 0 ;;
  esac
}

confirm() {
  printf '%s [y/N]: ' "$1" >&2
  IFS= read -r reply <&3 || die "no terminal to read answers from"
  case "$reply" in y|Y|yes|YES) return 0 ;; *) return 1 ;; esac
}

check_tools() {
  command -v docker >/dev/null 2>&1 || die "Docker is not installed. See https://docs.docker.com/get-docker/"
  docker compose version >/dev/null 2>&1 || die "Docker Compose is missing. See https://docs.docker.com/compose/install/"
  docker info >/dev/null 2>&1 || die "Docker is installed but not reachable. Is it running, and are you allowed to use it?"
  [ -S /var/run/docker.sock ] || die "/var/run/docker.sock not found; the app needs it to start checks."
}

collect_settings() {
  say "Answer each question, or press Enter to keep the value in [brackets]."
  ask PORT "The port the app listens on." "8081" "8081" port
  ask PUBLIC_URL "The address you open the app on. Decides which sites may sign in." \
    "https://workspace.example.com" "http://localhost:$PORT" url
  ask CONNECTOR_URL "Public https address for AI apps like ChatGPT. Leave empty if you do not connect any." \
    "https://workspace.example.com" "" url
  ask ALLOWED_ORIGINS "Extra addresses allowed to sign in, comma-separated. Leave empty for local use." \
    "https://workspace.example.com,http://192.168.1.20:8081" ""
  ask CLIENT_IP_HEADER "Header carrying the real visitor address behind a tunnel or proxy. Leave empty if none." \
    "CF-Connecting-IP" ""
  if confirm "Change advanced limits (memory, CPU, parallel checks)?"; then
    ask SANDBOX_MEMORY_MB "Memory one check may use, in MB." "2048" "2048" number
    ask SANDBOX_CPUS "CPU cores one check may use." "1" "1" number
    ask MAX_SANDBOXES "How many checks may run at once across all people." "4" "4" number
  fi
}

write_env() {
  env_file="$INSTALL_DIR/.env"
  if [ -f "$env_file" ]; then
    say ""
    confirm "$env_file already exists. Replace it (the old one is saved as .env.bak)?" || die "kept your existing .env; nothing changed."
    cp "$env_file" "$env_file.bak"
  fi
  {
    echo "# Written by scripts/install.sh. Empty settings are left out and use the defaults."
    echo "DOCKER_GID=$(stat -c %g /var/run/docker.sock)"
    for var in PORT PUBLIC_URL CONNECTOR_URL ALLOWED_ORIGINS CLIENT_IP_HEADER SANDBOX_MEMORY_MB SANDBOX_CPUS MAX_SANDBOXES; do
      eval "value=\${$var:-}"
      if [ -n "$value" ]; then echo "$var=$value"; fi
    done
  } >"$env_file"
  say ""
  say "Saved $env_file"
}

fetch_compose() {
  say "Downloading $COMPOSE_FILE ..."
  curl -fsSL "$REPO_RAW/$BRANCH/$COMPOSE_FILE" -o "$INSTALL_DIR/$COMPOSE_FILE" || die "could not download $COMPOSE_FILE"
}

# Polls /api/health for up to 60 seconds; on failure shows the last log lines.
wait_healthy() {
  say "Waiting for the app to answer on port $PORT ..."
  for _ in $(seq 1 60); do
    if curl -fsS "http://localhost:$PORT/api/health" >/dev/null 2>&1; then
      say "The app is healthy."
      return 0
    fi
    sleep 1
  done
  say "The app did not answer within 60 seconds. Last log lines:"
  (cd "$INSTALL_DIR" && docker compose -f "$COMPOSE_FILE" logs --tail 20 app) >&2 || true
  return 1
}

# Shows the running app container; the app manages its shared agentic-sandbox container on startup.
show_containers() {
  say ""
  (cd "$INSTALL_DIR" && docker compose -f "$COMPOSE_FILE" ps -a) >&2
}

start_app() {
  say "Starting the app (the first time downloads two images, so it can take a few minutes) ..."
  (cd "$INSTALL_DIR" && docker compose -f "$COMPOSE_FILE" up -d)
  healthy=0
  wait_healthy && healthy=1
  show_containers
  [ "$healthy" = 1 ] || die "the app is not healthy yet; see the logs above."
  say ""
  say "Done. Open ${PUBLIC_URL} and create your account."
  say "Logs: cd $INSTALL_DIR && docker compose -f $COMPOSE_FILE logs -f app"
  say "Stop: cd $INSTALL_DIR && docker compose -f $COMPOSE_FILE down"
}

check_tools
if [ -e "$INSTALL_DIR" ] && [ ! -d "$INSTALL_DIR" ]; then
  die "$INSTALL_DIR exists but is a file, not a folder. Pick another place: INSTALL_DIR=/some/folder"
fi
mkdir -p "$INSTALL_DIR"
collect_settings
write_env
fetch_compose
start_app
