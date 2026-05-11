#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT="$ROOT_DIR/scripts/bootstrap-ai-bridge.sh"
TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT

fail() { printf 'FAIL: %s\n' "$1" >&2; exit 1; }
assert_file() { [[ -f "$1" ]] || fail "missing file $1"; }
assert_contains() { grep -Fq "$2" "$1" || fail "expected $1 to contain $2"; }
assert_not_contains() { ! grep -Fq "$2" "$1" || fail "expected $1 not to contain $2"; }

run_bootstrap() {
  local name="$1"
  shift
  local dir="$TMP/$name"
  mkdir -p "$dir/open-core" "$dir/bin"
  cp "$ROOT_DIR/.env.example" "$dir/open-core/.env.example"
  PATH="$dir/bin:$PATH" \
  HOME="$dir/home" \
  XDG_CONFIG_HOME="$dir/xdg" \
  XDG_STATE_HOME="$dir/state" \
  SCOUT_BRIDGE_ROOT_DIR="$dir/open-core" \
  SCOUT_BRIDGE_NO_LAUNCH="1" \
  env "$@" "$SCRIPT"
  printf '%s\n' "$dir"
}

test_creates_env_config_and_token() {
  local dir
  dir="$(run_bootstrap create)"
  assert_file "$dir/open-core/.env"
  assert_file "$dir/xdg/scout/ai-bridge.token"
  assert_file "$dir/xdg/scout/ai-bridge.json"
  assert_contains "$dir/open-core/.env" 'SCOUT_AI_BRIDGE_URL=http://host.docker.internal:4761'
  assert_contains "$dir/open-core/.env" 'SCOUT_AI_BRIDGE_TOKEN_FILE=/run/secrets/scout-ai-bridge-token'
  assert_contains "$dir/open-core/.env" "SCOUT_AI_BRIDGE_HOST_TOKEN_FILE=$dir/xdg/scout/ai-bridge.token"
}

test_preserves_valid_token() {
  local dir token_before token_after
  dir="$(run_bootstrap preserve)"
  token_before="$(cat "$dir/xdg/scout/ai-bridge.token")"
  SCOUT_BRIDGE_ROOT_DIR="$dir/open-core" HOME="$dir/home" XDG_CONFIG_HOME="$dir/xdg" XDG_STATE_HOME="$dir/state" SCOUT_BRIDGE_NO_LAUNCH="1" "$SCRIPT"
  token_after="$(cat "$dir/xdg/scout/ai-bridge.token")"
  [[ "$token_before" == "$token_after" ]] || fail 'valid token rotated unexpectedly'
}

test_repairs_short_token() {
  local dir token_after
  dir="$(run_bootstrap repair)"
  printf 'short\n' > "$dir/xdg/scout/ai-bridge.token"
  SCOUT_BRIDGE_ROOT_DIR="$dir/open-core" HOME="$dir/home" XDG_CONFIG_HOME="$dir/xdg" XDG_STATE_HOME="$dir/state" SCOUT_BRIDGE_NO_LAUNCH="1" "$SCRIPT"
  token_after="$(cat "$dir/xdg/scout/ai-bridge.token")"
  [[ ${#token_after} -ge 32 ]] || fail 'short token was not repaired'
}

test_disable_skips_bridge_variables() {
  local dir
  dir="$(run_bootstrap disabled SCOUT_AI_BRIDGE_DISABLE=1)"
  assert_file "$dir/open-core/.env"
  assert_not_contains "$dir/open-core/.env" 'SCOUT_AI_BRIDGE_URL=http://host.docker.internal:4761'
}

test_unsupported_os_returns_success() {
  local dir="$TMP/unsupported"
  mkdir -p "$dir/open-core"
  cp "$ROOT_DIR/.env.example" "$dir/open-core/.env.example"
  SCOUT_BRIDGE_ROOT_DIR="$dir/open-core" SCOUT_BRIDGE_UNAME="MINGW64_NT" "$SCRIPT" >/tmp/scout-bridge-test-unsupported.log 2>&1 || fail 'unsupported OS should not fail'
  grep -Fq 'Skipping host CLI bridge bootstrap' /tmp/scout-bridge-test-unsupported.log || fail 'missing unsupported OS skip message'
}

test_creates_env_config_and_token
test_preserves_valid_token
test_repairs_short_token
test_disable_skips_bridge_variables
test_unsupported_os_returns_success
printf 'bootstrap-ai-bridge tests passed\n'
