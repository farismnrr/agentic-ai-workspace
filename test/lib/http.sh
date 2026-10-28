#!/usr/bin/env bash

http_request() {
  local method=$1 jar=$2 path=$3 payload=${4:-} type=${5:-application/json}
  local -a cookie_opts=()
  if [[ -n "$jar" ]]; then
    cookie_opts=(-b "$jar" -c "$jar")
  fi
  HTTP_BODY_FILE="$APP_TMP_DIR/http-body"
  HTTP_HEADERS_FILE="$APP_TMP_DIR/http-headers"
  if [[ -n $payload ]]; then
    # shellcheck disable=SC2034 # Callers inspect this shared response status.
    HTTP_STATUS=$(curl -sS -X "$method" "$APP_BASE$path" \
      "${cookie_opts[@]}" -H "Content-Type: $type" \
      --data "$payload" -D "$HTTP_HEADERS_FILE" -o "$HTTP_BODY_FILE" -w '%{http_code}')
  else
    # shellcheck disable=SC2034 # Callers inspect this shared response status.
    HTTP_STATUS=$(curl -sS -X "$method" "$APP_BASE$path" \
      "${cookie_opts[@]}" -D "$HTTP_HEADERS_FILE" \
      -o "$HTTP_BODY_FILE" -w '%{http_code}')
  fi
}

http_unauthenticated() {
  local method=$1 path=$2
  HTTP_BODY_FILE="$APP_TMP_DIR/http-body"
  HTTP_HEADERS_FILE="$APP_TMP_DIR/http-headers"
  # shellcheck disable=SC2034 # Callers inspect this shared response status.
  HTTP_STATUS=$(curl -sS -X "$method" "$APP_BASE$path" \
    -D "$HTTP_HEADERS_FILE" -o "$HTTP_BODY_FILE" -w '%{http_code}')
}

http_form() {
  local path=$1
  shift
  HTTP_BODY_FILE="$APP_TMP_DIR/http-body"
  HTTP_HEADERS_FILE="$APP_TMP_DIR/http-headers"
  # shellcheck disable=SC2034 # Callers inspect this shared response status.
  HTTP_STATUS=$(curl -sS -X POST "$APP_BASE$path" \
    -H 'Content-Type: application/x-www-form-urlencoded' \
    "$@" -D "$HTTP_HEADERS_FILE" -o "$HTTP_BODY_FILE" -w '%{http_code}')
}

http_urlencode() {
  jq -rn --arg value "$1" '$value | @uri'
}

http_register() {
  local jar=$1 name=$2 email=$3 password=$4
  http_request POST "$jar" /api/auth/register \
    "$(jq -cn --arg name "$name" --arg email "$email" --arg password "$password" \
      '{name:$name,email:$email,password:$password}')"
}

http_project_create() {
  local jar=$1 name=$2
  http_request POST "$jar" /api/projects \
    "$(jq -cn --arg name "$name" '{name:$name}')"
}
