#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${SCOUT_ENV_FILE:-${ROOT_DIR}/.env}"
API_URL="${THREADLENS_API_URL:-http://localhost:4749}"
WEB_URL="${THREADLENS_WEB_URL:-http://localhost:4748}"
DRY_RUN="${THREADLENS_SMOKE_DRY_RUN:-0}"

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
# In dry-run mode the network call is skipped and the check always succeeds.
check_url() {
  local label="$1"
  local url="$2"
  if [ "$DRY_RUN" = "1" ]; then
    printf 'DRY RUN: would check %s at %s\n' "$label" "$url"
    return 0
  fi
  if curl -fsS --max-time 5 "$url" >/dev/null 2>&1; then
    printf '%s reachable: %s\n' "$label" "$url"
    return 0
  fi
  printf '%s not reachable yet: %s\n' "$label" "$url" >&2
  return 1
}

print_step "ThreadLens self-host smoke check"
printf 'Open-core directory: %s\n' "$ROOT_DIR"
printf 'Env file: %s\n' "$ENV_FILE"
printf 'API URL: %s\n' "$API_URL"
printf 'Web URL: %s\n' "$WEB_URL"

print_step "Prerequisites"
require_command pnpm
require_command docker
require_command curl

print_step "Environment file"
if [ -f "$ENV_FILE" ]; then
  printf 'Env file exists.\n'
else
  printf 'Env file missing. Create it with: cp .env.example .env\n'
fi

print_step "Running services"
check_url "API health" "${API_URL}/api/health"
check_url "Onboarding status" "${API_URL}/api/onboarding/status"
check_url "Web app" "$WEB_URL"

print_step "Next activation step"
printf 'Open %s and complete the ThreadLens setup wizard. Then create one research project, add one narrow Reddit query, and run one manual scout.\n' "$WEB_URL"
