#!/usr/bin/env bash
# CI guard: ensures every go.mod, Dockerfile (FROM golang / go.dev tarball),
# and hardcoded go-version: in .github workflows matches .go-version

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
WANT="$(tr -d '[:space:]' < "$ROOT/.go-version")"
FAIL=0

echo "Expected Go version: $WANT"

check() {
  local file="$1" version="$2"
  if [[ -z "$version" ]]; then
    echo "  FAIL: $file — unpinned or missing version"
    return 1
  elif [[ "$version" != "$WANT" ]]; then
    echo "  FAIL: $file — found $version, expected $WANT"
    return 1
  else
    echo "  OK:   $file"
  fi
}

echo "Checking go.mod files..."
while IFS= read -r file; do
  check "$file" "$(awk '/^go [0-9]/{print $2; exit}' "$file")" || FAIL=1
done < <(find "$ROOT" -name "go.mod" -not -path "*/vendor/*")

echo "Checking Dockerfiles..."
while IFS= read -r file; do
  while IFS= read -r line; do
    version=$(echo "$line" | sed -n 's/.*golang:\([0-9]*\.[0-9]*\(\.[0-9]*\)\?\).*/\1/p')
    check "$file" "$version" || FAIL=1
  done < <(grep -E '^FROM golang:' "$file" || true)
  while IFS= read -r line; do
    version=$(echo "$line" | sed -n 's#.*go\.dev/dl/go\([0-9]*\.[0-9]*\(\.[0-9]*\)\?\)\..*#\1#p')
    check "$file (go.dev/dl tarball)" "$version" || FAIL=1
  done < <(grep -E 'go\.dev/dl/go[0-9]' "$file" || true)
done < <(find "$ROOT" -name "Dockerfile" -not -path "*/vendor/*")

echo "Checking hardcoded go-version in workflows and actions..."
while IFS= read -r file; do
  while IFS= read -r line; do
    version=$(echo "$line" | sed -n "s/.*go-version:[[:space:]]*['\"]\?\([0-9][0-9.]*\)['\"]\?.*/\1/p")
    check "$file" "$version" || FAIL=1
  done < <(grep -E "^[[:space:]]*go-version:[[:space:]]*['\"]?[0-9]" "$file" || true)
done < <(find "$ROOT/.github" \( -name "*.yml" -o -name "*.yaml" \) -not -path "*/node_modules/*")

if [[ $FAIL -ne 0 ]]; then
  echo "FAILED: version mismatch found — update files to match .go-version ($WANT)"
  exit 1
fi
echo "All Go versions match $WANT"
