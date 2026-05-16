#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Provide a temporary .env so the env-file check does not block the dry run.
TMPENV="$(mktemp)"
trap 'rm -f "$TMPENV"' EXIT

OUTPUT="$(THREADLENS_SMOKE_DRY_RUN=1 SCOUT_ENV_FILE="$TMPENV" bash "${ROOT_DIR}/scripts/self-host-smoke.sh")"

case "$OUTPUT" in
  *"ThreadLens self-host smoke check"*) ;;
  *) printf 'missing smoke check heading\n' >&2; exit 1 ;;
esac

case "$OUTPUT" in
  *"DRY RUN: would check API health at http://localhost:4749/api/health"*) ;;
  *) printf 'missing API dry-run line\n' >&2; exit 1 ;;
esac

case "$OUTPUT" in
  *"complete the ThreadLens setup wizard"*) ;;
  *) printf 'missing activation next step\n' >&2; exit 1 ;;
esac

printf 'self-host smoke dry-run test passed\n'
