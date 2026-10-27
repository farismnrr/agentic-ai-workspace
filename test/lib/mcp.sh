#!/usr/bin/env bash

mcp_call() {
  local token=$1 payload=$2
  HTTP_BODY_FILE="$APP_TMP_DIR/http-body"
  HTTP_HEADERS_FILE="$APP_TMP_DIR/http-headers"
  HTTP_STATUS=$(curl -sS -X POST "$APP_BASE/api/coding" \
    -H 'Content-Type: application/json' \
    -H 'Accept: application/json, text/event-stream' \
    -H 'MCP-Protocol-Version: 2025-11-25' \
    -H "Authorization: Bearer $token" --data "$payload" \
    -D "$HTTP_HEADERS_FILE" -o "$HTTP_BODY_FILE" -w '%{http_code}')
}

mcp_list_tools() {
  mcp_call "$1" '{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}'
}

mcp_call_tool() {
  local token=$1 name=$2 arguments=$3
  mcp_call "$token" "$(jq -cn --arg name "$name" --argjson arguments "$arguments" \
    '{jsonrpc:"2.0",id:1,method:"tools/call",params:{name:$name,arguments:$arguments}}')"
}

oauth_code_token() {
  local jar=$1 scope=$2 resource=$3 exchange_resource=${4:-$3}
  local exchange_verifier=${5:-dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk}
  local verifier challenge redirect request_id redirect_uri query
  verifier='dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk'
  challenge='E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM'
  redirect_uri='https://chatgpt.com/connector_platform_oauth_redirect'
  http_request POST "$jar" /oauth/register \
    "$(jq -cn --arg redirect "$redirect_uri" \
      '{client_name:"Blackbox harness",redirect_uris:[$redirect],token_endpoint_auth_method:"none"}')"
  assert_status "$HTTP_STATUS" 201 'OAuth client registration'
  OAUTH_CLIENT_ID=$(jq -r '.client_id // empty' "$HTTP_BODY_FILE")

  query="response_type=code&client_id=$(http_urlencode "$OAUTH_CLIENT_ID")"
  query+="&redirect_uri=$(http_urlencode "$redirect_uri")&scope=$(http_urlencode "$scope")"
  query+="&state=s1&code_challenge=$challenge&code_challenge_method=S256"
  query+="&resource=$(http_urlencode "$resource")"
  http_request GET "$jar" "/oauth/authorize?$query"
  assert_status "$HTTP_STATUS" 302 'OAuth authorize redirects to consent'
  request_id=$(sed -n 's/.*[?&]request=\([^&]*\).*/\1/p' "$HTTP_HEADERS_FILE" | tr -d '\r')

  http_request POST "$jar" "/api/oauth/requests/$request_id/decision" \
    '{"approve":true}'
  assert_status "$HTTP_STATUS" 200 'OAuth consent accepted'
  OAUTH_CODE=$(jq -r '.redirect // empty' "$HTTP_BODY_FILE" | sed -n 's/.*[?&]code=\([^&]*\).*/\1/p')

  http_form /oauth/token \
    --data-urlencode 'grant_type=authorization_code' \
    --data-urlencode "code=$OAUTH_CODE" --data-urlencode "client_id=$OAUTH_CLIENT_ID" \
    --data-urlencode "redirect_uri=$redirect_uri" \
    --data-urlencode "code_verifier=$exchange_verifier" \
    --data-urlencode "resource=$exchange_resource"
}
