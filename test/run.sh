#!/usr/bin/env bash
set -eu

REPO_ROOT=$(cd "$(dirname "$0")/.." && pwd)
suite=${1:-}

case $suite in
  blackbox)
    for dependency in curl jq python3 unzip; do
      if ! command -v "$dependency" >/dev/null 2>&1; then
        printf 'Required black-box dependency is missing: %s\n' "$dependency" >&2
        exit 1
      fi
    done
    if [[ -z ${APP_BIN:-} ]]; then
      make -C "$REPO_ROOT" backend-build
      APP_BIN="$REPO_ROOT/bin/agentic-ai-workspace"
    fi
    export APP_BIN
    if [[ -n ${2:-} ]]; then
      if [[ $2 == *[!a-z0-9-]* ]]; then
        printf 'Invalid black-box scenario: %s\n' "$2" >&2
        exit 2
      fi
      scenario="$REPO_ROOT/test/blackbox/${2}.test.sh"
      if [[ ! -f $scenario ]]; then
        printf 'Unknown black-box scenario: %s\n' "$2" >&2
        exit 2
      fi
      printf '\n==> %s\n' "$(basename "$scenario")"
      bash "$scenario"
    else
      for scenario in "$REPO_ROOT"/test/blackbox/*.test.sh; do
        printf '\n==> %s\n' "$(basename "$scenario")"
        bash "$scenario"
      done
    fi
    ;;
  smoke)
    if [[ -z ${APP_BIN:-} ]]; then
      make -C "$REPO_ROOT" backend-build
      APP_BIN="$REPO_ROOT/bin/agentic-ai-workspace"
    fi
    export APP_BIN
    bash "$REPO_ROOT/test/smoke/binary.test.sh" "$APP_BIN"
    ;;
  e2e)
    bash "$REPO_ROOT/test/e2e/run.sh" "${2:-}"
    ;;
  registry-check)
    bash "$REPO_ROOT/test/go/check-registry.sh"
    ;;
  *)
    printf 'Usage: %s {blackbox [scenario]|smoke|e2e|registry-check}\n' "$0" >&2
    exit 2
    ;;
esac
