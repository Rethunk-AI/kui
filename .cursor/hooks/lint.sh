#!/bin/bash
# Advisory lint on changed Go files after each file edit.
# Always exits 0 so it never blocks the agent.
# Cursor expects valid JSON on stdout; capture lint output and wrap in response.
input=$(cat)
file_path=$(echo "$input" | jq -r '.file_path // empty')
root="${CURSOR_PROJECT_DIR:-$(pwd)}"

run_lint() {
  local out rel pkg
  if [[ "$file_path" == *.go ]]; then
    rel="$file_path"
    [[ "$file_path" == "$root"/* ]] && rel="${file_path#$root/}"
    pkg=$(dirname "$rel")
    if command -v golangci-lint &>/dev/null; then
      out=$(cd "$root" && golangci-lint run "./$pkg/..." 2>&1)
    else
      out=$(cd "$root" && go vet "./$pkg/..." 2>&1)
    fi
  else
    out="No linter configured for ${file_path:-<unknown>}"
  fi
  printf '%s' "$out" | jq -Rs '{ok: true, lint_output: .}'
}

run_lint
exit 0
