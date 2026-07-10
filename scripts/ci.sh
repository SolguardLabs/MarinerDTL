#!/usr/bin/env bash
set -euo pipefail

if [ -x "$HOME/.cache/codex-node/node/bin/node" ]; then
  export PATH="$HOME/.cache/codex-node/node/bin:$PATH"
fi

GO_CMD="go"
GOFMT_CMD="gofmt"
if ! command -v go >/dev/null 2>&1; then
  if [ -x "$HOME/.cache/codex-go/go/bin/go" ] || [ -x "$HOME/.cache/codex-go/go/bin/go.exe" ]; then
    export PATH="$HOME/.cache/codex-go/go/bin:$PATH"
    if [ -f "$HOME/.cache/codex-go/go/bin/go.exe" ]; then
      GO_CMD="go.exe"
      GOFMT_CMD="gofmt.exe"
    fi
  elif [ -n "${USERPROFILE:-}" ] && command -v cygpath >/dev/null 2>&1; then
    user_profile="$(cygpath -u "$USERPROFILE")"
    if [ -f "$user_profile/.cache/codex-go/go/bin/go.exe" ]; then
      export PATH="$user_profile/.cache/codex-go/go/bin:$PATH"
      GO_CMD="go.exe"
      GOFMT_CMD="gofmt.exe"
    fi
  elif [[ "$(pwd -P)" == /mnt/*/Users/*/Desktop/* ]]; then
    user_profile="$(pwd -P)"
    user_profile="${user_profile%%/Desktop/*}"
    if [ -f "$user_profile/.cache/codex-go/go/bin/go.exe" ]; then
      export PATH="$user_profile/.cache/codex-go/go/bin:$PATH"
      GO_CMD="go.exe"
      GOFMT_CMD="gofmt.exe"
    fi
  fi
fi

"$GOFMT_CMD" -w src
"$GO_CMD" test ./...
"$GO_CMD" vet ./...
npm run build
npm run check
npm test
npm run loc
