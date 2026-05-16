#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${SCOUT_ENV_FILE:-${ROOT_DIR}/.env}"
API_URL="${THREADLENS_API_URL:-http://localhost:4749}"
WEB_URL="${THREADLENS_WEB_URL:-http://localhost:4748}"
DRY_RUN="${THREADLENS_SMOKE_DRY_RUN:-0}"
# How many times to retry an endpoint check before giving up (each attempt is ~2 s apart)
RETRY_MAX="${THREADLENS_SMOKE_RETRIES:-3}"

print_step() {
  printf '\n==> %s\n' "$1"
}

require_command() {
  if ! command -v "$1" >/dev/null 2>&1; then
    printf 'Missing required command: %s\n' "$1" >&2
    return 1
  fi
  printf 'Found %s\n' "$1"
}

# check_url LABEL URL
# Retries up to RETRY_MAX times with a 2-second pause between attempts.
# In dry-run mode the network call is skipped and the check always succeeds.
check_url() {
  local label="$1"
  local url="$2"
  if [ "$DRY_RUN" = "1" ]; then
    printf 'DRY RUN: would check %s at %s\n' "$label" "$url"
    return 0
  fi
  local attempt=0
  while [ "$attempt" -lt "$RETRY_MAX" ]; do
    if curl -fsS --max-time 5 "$url" >/dev/null 2>&1; then
      printf '%s reachable: %s\n' "$label" "$url"
      return 0
    fi
    attempt=$(( attempt + 1 ))
    if [ "$attempt" -lt "$RETRY_MAX" ]; then
      printf '%s not reachable yet (%d/%d), retrying in 2 s...\n' "$label" "$attempt" "$RETRY_MAX" >&2
      sleep 2
    fi
  done
  printf '%s not reachable yet after %d attempt(s): %s\n' "$label" "$RETRY_MAX" "$url" >&2
  return 1
}

print_step "ThreadLens self-host smoke check"
printf 'Open-core directory: %s\n' "$ROOT_DIR"
printf 'Env file: %s\n' "$ENV_FILE"
printf 'API URL: %s\n' "$API_URL"
printf 'Web URL: %s\n' "$WEB_URL"

print_step "Prerequisites"
require_command curl

print_step "Environment file"
if [ -f "$ENV_FILE" ]; then
  printf 'Env file exists.\n'
else
  printf 'ERROR: Env file missing at %s — create it with: cp .env.example .env\n' "$ENV_FILE" >&2
  exit 1
fi

print_step "Running services"
check_url "API health" "${API_URL}/api/health"
check_url "Onboarding status" "${API_URL}/api/onboarding/status"
check_url "Web app" "$WEB_URL"

print_step "Next activation step"
printf 'Open %s and complete the ThreadLens setup wizard. Then create one research project, add one narrow Reddit query, and run one manual scout.\n' "$WEB_URL"
