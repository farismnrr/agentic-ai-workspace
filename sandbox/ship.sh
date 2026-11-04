#!/bin/sh
# Saves the project's work in git and pushes it, the one way the DevOps teammate does it before the team reports.
# Usage: ship "<commit message>"   (the first line is the title; more lines are the body)
#  - the work always goes on the branch dev: made when it does not exist, never main, never a force-push;
#  - secrets (.env, keys) and dependency folders are kept out of git;
#  - no repository, no remote or no GitHub connection is not a failure: the work is committed here, and the
#    last lines say whether it was pushed and why not.
set -u

message="${1:-}"
[ -n "$message" ] || { echo "usage: ship \"<commit message>\"" >&2; exit 2; }

cd "${PROJECT_DIR:-/workspace}" || exit 2

git rev-parse --is-inside-work-tree > /dev/null 2>&1 || git init -q -b dev || { echo "could not start a repository" >&2; exit 2; }
if ! git var GIT_AUTHOR_IDENT > /dev/null 2>&1 || ! git var GIT_COMMITTER_IDENT > /dev/null 2>&1; then
  echo "error: Git author or committer identity is not configured." >&2
  echo "detail: Connect GitHub in Settings or configure user.name and user.email before shipping." >&2
  exit 2
fi

# Keep secrets and generated folders out, whatever the project already ignores.
touch .gitignore
for pattern in .env '.env.*' '*.pem' '*.key' 'id_rsa*' node_modules .next .nuxt .output dist target __pycache__ .venv; do
  grep -qxF -- "$pattern" .gitignore || echo "$pattern" >> .gitignore
done

# Work on dev: the branch is made when it is missing (an empty repository just points at it).
if [ "$(git symbolic-ref --short -q HEAD || true)" != dev ]; then
  if ! git rev-parse --verify -q HEAD > /dev/null 2>&1; then
    git symbolic-ref HEAD refs/heads/dev
  elif git show-ref --verify -q refs/heads/dev; then
    git switch -q dev || { echo "could not switch to dev: finish or move the changes first" >&2; exit 3; }
  else
    git switch -q -c dev || { echo "could not make the branch dev" >&2; exit 3; }
  fi
fi

git add -A
# A secret that slipped past .gitignore is taken out again, and said.
git diff --cached --name-only | grep -E '(^|/)\.env($|\.)|\.pem$|\.key$|(^|/)id_rsa' | while read -r secret; do
  git reset -q -- "$secret" && echo "left out (looks like a secret): $secret"
done

if git diff --cached --quiet; then
  echo "commit: nothing new to save"
else
  git commit -q -m "$message" || { echo "commit failed" >&2; exit 4; }
  echo "commit: $(git rev-parse --short HEAD) on dev"
fi

# Report the exact repository and commit used by this invocation. Missing upstream is
# not missing origin; imported repositories commonly have only origin/main.
head="$(git rev-parse HEAD 2>/dev/null || true)"
printf 'repository: %s\n' "$(git rev-parse --show-toplevel)"
printf 'head: %s\n' "$head"
printf 'branch: %s\n' "$(git symbolic-ref --short HEAD)"

if ! git remote get-url origin > /dev/null 2>&1; then
  if [ -n "$(git remote)" ]; then
    echo "pushed: no (origin is missing; existing remotes: $(git remote | tr '\n' ' '))"
  else
    echo "pushed: no (this repository has no Git remotes; attach its GitHub repository before pushing)"
  fi
  exit 0
fi

# A successful push is verified against the remote's actual branch tip, not
# merely against the exit code or a locally configured tracking branch.
refspec='+refs/heads/dev:refs/remotes/origin/dev'
if ! git config --get-all remote.origin.fetch 2>/dev/null | grep -qxF "$refspec"; then
  git config --add remote.origin.fetch "$refspec"
fi

# Fetch origin dev before pushing and set upstream tracking.
if git fetch -q origin refs/heads/dev:refs/remotes/origin/dev 2>/dev/null; then
  git branch --set-upstream-to=origin/dev dev > /dev/null 2>&1 || true
fi

# When remote dev is ahead, safely synchronize without force-pushing.
if git rev-parse --verify -q origin/dev > /dev/null 2>&1; then
  behind="$(git rev-list --count HEAD..origin/dev 2>/dev/null || echo 0)"
  ahead="$(git rev-list --count origin/dev..HEAD 2>/dev/null || echo 0)"
  if [ "$behind" -gt 0 ]; then
    if [ "$ahead" -eq 0 ]; then
      git merge --ff-only origin/dev > /dev/null 2>&1 || true
    else
      if ! git rebase origin/dev > /dev/null 2>&1; then
        git rebase --abort > /dev/null 2>&1 || true
        echo "warning: upstream dev has diverged and could not be automatically rebased" >&2
      fi
    fi
    head="$(git rev-parse HEAD 2>/dev/null || true)"
  fi
fi

if output="$(git push --porcelain -u origin HEAD:refs/heads/dev 2>&1)"; then
  remote_head="$(git ls-remote --heads origin refs/heads/dev 2>/dev/null | awk 'NR == 1 {print $1}')"
  if [ -n "$head" ] && [ "$remote_head" = "$head" ]; then
    git branch --set-upstream-to=origin/dev dev > /dev/null 2>&1 || true
    echo "pushed: yes (origin/dev at $head)"
  else
    echo "pushed: no (push returned success but remote dev could not be verified against local HEAD)"
    echo "local head: $head"
    echo "remote head: ${remote_head:-unavailable}"
  fi
else
  reason="$(printf '%s\n' "$output" | grep -m1 -E 'rejected|fatal|error|denied' || printf '%s\n' "$output" | tail -n 1)"
  echo "pushed: no ($reason)"
fi
exit 0
