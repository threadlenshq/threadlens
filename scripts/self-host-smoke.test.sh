#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SMOKE_SCRIPT="${ROOT_DIR}/scripts/self-host-smoke.sh"

# ── helpers ──────────────────────────────────────────────────────────────────
pass() { printf 'PASS: %s\n' "$1"; }
fail() { printf 'FAIL: %s\n' "$1" >&2; exit 1; }

assert_contains() {
  local label="$1" needle="$2" haystack="$3"
  case "$haystack" in
    *"$needle"*) pass "$label" ;;
    *) fail "$label — expected: $needle"; ;;
  esac
}

assert_not_contains() {
  local label="$1" needle="$2" haystack="$3"
  case "$haystack" in
    *"$needle"*) fail "$label — unexpected text found: $needle" ;;
    *) pass "$label" ;;
  esac
}

# ── shared temp dir ──────────────────────────────────────────────────────────
TMPDIR_TEST="$(mktemp -d)"
trap 'rm -rf "$TMPDIR_TEST"' EXIT

# ─────────────────────────────────────────────────────────────────────────────
# TEST 1 – existing dry-run assertions (regression: must keep passing)
# ─────────────────────────────────────────────────────────────────────────────
printf '\n=== TEST 1: dry-run happy path ===\n'

TMPENV="$(mktemp)"
OUTPUT="$(THREADLENS_SMOKE_DRY_RUN=1 SCOUT_ENV_FILE="$TMPENV" bash "$SMOKE_SCRIPT")"
rm -f "$TMPENV"

assert_contains "dry-run: smoke heading"        "ThreadLens self-host smoke check"                              "$OUTPUT"
assert_contains "dry-run: API dry-run line"     "DRY RUN: would check API health at http://localhost:4749/api/health" "$OUTPUT"
assert_contains "dry-run: activation next step" "complete the ThreadLens setup wizard"                          "$OUTPUT"

# ─────────────────────────────────────────────────────────────────────────────
# TEST 2 – missing env file causes the script to fail with the expected error
# ─────────────────────────────────────────────────────────────────────────────
printf '\n=== TEST 2: missing env file produces expected error ===\n'

MISSING_ENV="${TMPDIR_TEST}/does_not_exist.env"
# Run without dry-run flag so the env-file gate is reached first.
# Capture stderr too; the error is printed to stderr.
MISSING_OUTPUT="$(SCOUT_ENV_FILE="$MISSING_ENV" bash "$SMOKE_SCRIPT" 2>&1)" && {
  fail "missing-env: script should have exited non-zero"
} || true  # non-zero exit is expected — we just want to inspect the output

assert_contains "missing-env: error message"         "ERROR: Env file missing at"     "$MISSING_OUTPUT"
assert_contains "missing-env: cp suggestion"          "cp .env.example .env"           "$MISSING_OUTPUT"
assert_not_contains "missing-env: no wizard step shown" "setup wizard"                 "$MISSING_OUTPUT"

# ─────────────────────────────────────────────────────────────────────────────
# TEST 3 – unreachable endpoint emits retry messaging, then fails
# ─────────────────────────────────────────────────────────────────────────────
printf '\n=== TEST 3: unreachable endpoint emits retry messaging then fails ===\n'

# Create a fake curl that always fails, and a no-op sleep so the test is fast.
FAKE_BIN="${TMPDIR_TEST}/bin"
mkdir -p "$FAKE_BIN"

cat > "${FAKE_BIN}/curl" <<'EOF'
#!/usr/bin/env bash
exit 1
EOF
chmod +x "${FAKE_BIN}/curl"

cat > "${FAKE_BIN}/sleep" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
chmod +x "${FAKE_BIN}/sleep"

# Give it a real env file so the env-file gate passes.
TMPENV2="$(mktemp)"

RETRY_OUTPUT="$(
  PATH="${FAKE_BIN}:${PATH}" \
  SCOUT_ENV_FILE="$TMPENV2" \
  THREADLENS_SMOKE_RETRIES=3 \
  bash "$SMOKE_SCRIPT" 2>&1
)" && {
  fail "retry: script should have exited non-zero when endpoint is unreachable"
} || true  # non-zero exit expected

rm -f "$TMPENV2"

assert_contains "retry: retry messaging present"    "retrying in 2 s"        "$RETRY_OUTPUT"
assert_contains "retry: final failure message"      "not reachable yet after" "$RETRY_OUTPUT"
assert_not_contains "retry: should not reach wizard" "setup wizard"          "$RETRY_OUTPUT"

# ─────────────────────────────────────────────────────────────────────────────
printf '\n==> All self-host smoke tests passed.\n'
