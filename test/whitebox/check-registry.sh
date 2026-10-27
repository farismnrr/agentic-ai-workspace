#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
registry="$repo_root/test/whitebox/registry.md"
enforce=${REGISTRY_ENFORCE:-1}

if [[ ! -f $registry ]]; then
  printf 'White-box registry is missing: %s\n' "$registry" >&2
  exit 1
fi

mapfile -t entries < <(
  sed -nE 's/^\| `([^`]+::Test[^`]+)` \|.*/\1/p' "$registry"
)
mapfile -t support_files < <(
  sed -nE 's/^- `([^`]+_test\.go)` .*/\1/p' "$registry"
)

for entry in "${entries[@]}"; do
  path=${entry%%::*}
  name=${entry##*::}
  file="$repo_root/$path"
  if [[ ! -f $file ]] || ! grep -Eq "^[[:space:]]*func[[:space:]]+$name\\(" "$file"; then
    printf 'Registry entry does not name an existing test: %s\n' "$entry" >&2
    exit 1
  fi
  [[ $path == internal/bootstrap/* ]] && continue
done

if [[ $enforce != 1 ]]; then
  exit 0
fi

missing=0
while IFS= read -r -d '' file; do
  relative=${file#"$repo_root/"}
  [[ $relative == internal/bootstrap/* ]] && continue
  registered_file=0
  for entry in "${entries[@]}"; do
    if [[ ${entry%%::*} == "$relative" ]]; then
      registered_file=1
      break
    fi
  done
  if [[ $registered_file == 0 ]]; then
    for support in "${support_files[@]}"; do
      if [[ $support == "$relative" ]]; then
        registered_file=1
        break
      fi
    done
  fi
  if [[ $registered_file == 0 ]]; then
    printf 'Unregistered white-box test file: %s\n' "$relative" >&2
    missing=1
  fi
  while IFS= read -r name; do
    [[ -z $name ]] && continue
    found=0
    for entry in "${entries[@]}"; do
      if [[ $entry == "$relative::$name" ]]; then
        found=1
        break
      fi
    done
    if [[ $found == 0 ]]; then
      printf 'Unregistered white-box test: %s::%s\n' "$relative" "$name" >&2
      missing=1
    fi
  done < <(sed -nE 's/^[[:space:]]*func[[:space:]]+(Test[^ (]+)\(.*/\1/p' "$file")
done < <(find "$repo_root/internal" -type f -name '*_test.go' -print0)

exit "$missing"
