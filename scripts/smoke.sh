#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${SMOKE_BASE_URL:-http://localhost:8080}"
TIMEOUT="${SMOKE_TIMEOUT:-60}"
TARGET_URL="https://example.com/smoke"

fail() {
  echo "smoke: $*" >&2
  exit 1
}

wait_for() {
  local url=$1
  local deadline=$((SECONDS + TIMEOUT))

  until curl -fsS -o /dev/null "$url"; do
    if ((SECONDS >= deadline)); then
      fail "timed out waiting for $url"
    fi
    sleep 1
  done
}

echo "smoke: waiting for frontend and backend"
wait_for "$BASE_URL/"
wait_for "$BASE_URL/api/links"

echo "smoke: creating link"
payload=$(curl -fsS -X POST "$BASE_URL/api/links" \
  -H 'Content-Type: application/json' \
  -d "{\"url\":\"$TARGET_URL\"}")

code=$(printf '%s' "$payload" | sed -n 's/.*"code":"\([A-Za-z0-9]*\)".*/\1/p')
[[ -n "$code" ]] || fail "could not parse code from response: $payload"

short_url=$(printf '%s' "$payload" | sed -n 's/.*"short_url":"\([^"]*\)".*/\1/p')
case "$short_url" in
  */r/"$code") ;;
  *) fail "unexpected short_url: $short_url" ;;
esac

echo "smoke: following redirect for $code"
headers=$(curl -sS -D - -o /dev/null "$BASE_URL/r/$code")
status=$(printf '%s\n' "$headers" | awk 'NR==1 {print $2}')
location=$(printf '%s\n' "$headers" | awk 'tolower($1) == "location:" {print $2}' | tr -d '\r')

[[ "$status" == "302" ]] || fail "redirect status = $status, want 302"
[[ "$location" == "$TARGET_URL" ]] || fail "redirect location = $location, want $TARGET_URL"

echo "smoke: waiting for click aggregation"
deadline=$((SECONDS + TIMEOUT))
while :; do
  stats=$(curl -fsS "$BASE_URL/api/links/$code/stats")
  total=$(printf '%s' "$stats" | sed -n 's/.*"total":\([0-9][0-9]*\).*/\1/p')

  if [[ "$total" == "1" ]]; then
    break
  fi
  if ((SECONDS >= deadline)); then
    fail "stats total = ${total:-unknown}, want 1 (response: $stats)"
  fi
  sleep 1
done

echo "smoke: ok (code=$code, total=1)"
