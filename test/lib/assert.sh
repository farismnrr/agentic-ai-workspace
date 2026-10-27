#!/usr/bin/env bash

ASSERTIONS_PASSED=0
ASSERTIONS_FAILED=0

assert_status() {
  local actual=$1 expected=$2 label=$3
  if [[ $actual == "$expected" ]]; then
    printf 'PASS %s (HTTP %s)\n' "$label" "$actual"
    ((ASSERTIONS_PASSED += 1))
  else
    printf 'FAIL %s: expected HTTP %s, got %s\n' "$label" "$expected" "$actual"
    ((ASSERTIONS_FAILED += 1))
  fi
}

assert_json() {
  local expression=$1 label=$2
  shift 2
  if jq -e "$@" "$expression" "$HTTP_BODY_FILE" >/dev/null; then
    printf 'PASS %s\n' "$label"
    ((ASSERTIONS_PASSED += 1))
  else
    printf 'FAIL %s: JSON assertion failed\n' "$label"
    cat "$HTTP_BODY_FILE"
    printf '\n'
    ((ASSERTIONS_FAILED += 1))
  fi
}

assert_contains() {
  local file=$1 expected=$2 label=$3
  if grep -Fq -- "$expected" "$file"; then
    printf 'PASS %s\n' "$label"
    ((ASSERTIONS_PASSED += 1))
  else
    printf 'FAIL %s: expected text was absent\n' "$label"
    ((ASSERTIONS_FAILED += 1))
  fi
}

assert_not_contains() {
  local file=$1 unexpected=$2 label=$3
  if grep -Fq -- "$unexpected" "$file"; then
    printf 'FAIL %s: secret text was present\n' "$label"
    ((ASSERTIONS_FAILED += 1))
  else
    printf 'PASS %s\n' "$label"
    ((ASSERTIONS_PASSED += 1))
  fi
}

assert_file_exists() {
  if [[ -e $1 ]]; then
    printf 'PASS %s\n' "$2"
    ((ASSERTIONS_PASSED += 1))
  else
    printf 'FAIL %s\n' "$2"
    ((ASSERTIONS_FAILED += 1))
  fi
}

assertion_summary() {
  printf 'Assertions: %s passed, %s failed\n' "$ASSERTIONS_PASSED" "$ASSERTIONS_FAILED"
  ((ASSERTIONS_FAILED == 0))
}
