#!/usr/bin/env bash

grant_http() {
  local method=$1 jar=$2 path=$3 payload=${4:-} csrf=${5:-}
  HTTP_BODY_FILE="$APP_TMP_DIR/grant-body"
  HTTP_HEADERS_FILE="$APP_TMP_DIR/grant-headers"
  local args=(-sS --max-time 20 -X "$method" "$APP_BASE$path"
    -H 'Content-Type: application/json' -D "$HTTP_HEADERS_FILE" -o "$HTTP_BODY_FILE"
    -w '%{http_code}')
  [[ -z $jar ]] || args+=(-b "$jar" -c "$jar")
  [[ -z $csrf ]] || args+=(-H "X-Egress-CSRF: $csrf")
  [[ -z $payload ]] || args+=(--data "$payload")
  # shellcheck disable=SC2034 # Callers inspect the shared HTTP status and body.
  HTTP_STATUS=$(curl "${args[@]}")
}

grant_require_status() {
  assert_status "$HTTP_STATUS" "$1" "$2"
}

grant_create_task() {
  local jar=$1 project_id=$2
  http_request POST "$jar" /api/tasks \
    "$(jq -cn --arg project "$project_id" \
      '{projectId:$project,prompt:"Disposable grant check",providerId:"plan:claude",model:"default",team:false}')"
  assert_status "$HTTP_STATUS" 201 'task fixture created through the API'
  GRANT_TASK_ID=$(jq -r '.id // empty' "$HTTP_BODY_FILE")
  [[ -n $GRANT_TASK_ID ]]
}

grant_setup() {
  local jar=$1 state_file=$2 project_id csrf grant_a grant_b
  local password='volume-isolation-test-only-password'
  http_register "$jar" 'Grant Check' 'grant-check@example.invalid' "$password"
  assert_status "$HTTP_STATUS" 201 'test account registered'
  http_project_create "$jar" 'Volume isolation grant check'
  assert_status "$HTTP_STATUS" 201 'test project created'
  project_id=$(jq -r '.id // empty' "$HTTP_BODY_FILE")
  [[ -n $project_id ]]

  local grant_path="/api/projects/$project_id/egress-grants"

  grant_http GET '' "$grant_path/"
  grant_require_status 401 'anonymous grant access denied'
  grant_http GET "$jar" "$grant_path/csrf"
  grant_require_status 200 'grant CSRF token issued'
  csrf=$(jq -r '.token // empty' "$HTTP_BODY_FILE")
  [[ -n $csrf ]]

  grant_create_task "$jar" "$project_id"
  local task_a=$GRANT_TASK_ID
  grant_http POST "$jar" "$grant_path/" \
    "$(jq -cn --arg task "$task_a" \
      '{taskId:$task,host:"example.com",reason:"disposable isolation check"}')" "$csrf"
  grant_require_status 201 'first task grant requested'
  assert_json '.status == "pending"' 'new grant starts pending'
  grant_a=$(jq -r '.id // empty' "$HTTP_BODY_FILE")
  [[ -n $grant_a ]]
  grant_http POST "$jar" "$grant_path/$grant_a/approve" '{"ttlHours":1}' "$csrf"
  grant_require_status 200 'first task grant approved'
  assert_json '.status == "approved"' 'approval is persisted'
  grant_http POST "$jar" "$grant_path/$grant_a/revoke" '{}' "$csrf"
  grant_require_status 200 'first task grant revoked'
  assert_json '.status == "revoked"' 'revocation is persisted'
  grant_http POST "$jar" "$grant_path/$grant_a/approve" '{}' "$csrf"
  grant_require_status 409 'revoked grant cannot be replayed'

  grant_create_task "$jar" "$project_id"
  local task_b=$GRANT_TASK_ID
  grant_http POST "$jar" "$grant_path/" \
    "$(jq -cn --arg task "$task_b" \
      '{taskId:$task,host:"example.com",reason:"restart restoration check"}')" "$csrf"
  grant_require_status 201 'second task grant requested'
  grant_b=$(jq -r '.id // empty' "$HTTP_BODY_FILE")
  [[ -n $grant_b ]]
  grant_http POST "$jar" "$grant_path/$grant_b/approve" '{"ttlHours":1}' "$csrf"
  grant_require_status 200 'second task grant approved'
  assert_json '.status == "approved"' 'approved grant is persisted for restart check'
  jq -cn --arg project "$project_id" --arg grant "$grant_b" \
    '{project:$project,grant:$grant}' > "$state_file"
}

grant_verify_restart() {
  local jar=$1 state_file=$2 project_id grant_id grant_path csrf
  project_id=$(jq -r '.project' "$state_file")
  grant_id=$(jq -r '.grant' "$state_file")
  http_request POST "$jar" /api/auth/login \
    '{"email":"grant-check@example.invalid","password":"volume-isolation-test-only-password"}'
  assert_status "$HTTP_STATUS" 200 'test account signs in after restart'
  grant_path="/api/projects/$project_id/egress-grants"
  grant_http GET "$jar" "$grant_path/"
  grant_require_status 200 'grant list available after restart'
  if jq -e --arg id "$grant_id" \
    '.[] | select(.id == $id and .status == "approved")' "$HTTP_BODY_FILE" >/dev/null; then
    printf 'PASS approved grant restored after app restart\n'
    ((ASSERTIONS_PASSED += 1))
  else
    printf 'FAIL approved grant restored after app restart\n'
    ((ASSERTIONS_FAILED += 1))
  fi
  grant_http GET "$jar" "$grant_path/csrf"
  grant_require_status 200 'post-restart CSRF token issued'
  csrf=$(jq -r '.token // empty' "$HTTP_BODY_FILE")
  grant_http POST "$jar" "$grant_path/$grant_id/revoke" '{}' "$csrf"
  grant_require_status 200 'restored grant can be revoked'
  grant_http POST "$jar" "$grant_path/$grant_id/approve" '{}' "$csrf"
  grant_require_status 409 'restored revoked grant cannot be replayed'
}
