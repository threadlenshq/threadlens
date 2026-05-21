#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SMOKE_SCRIPT="${ROOT_DIR}/scripts/self-host-smoke.sh"

OUTPUT="$(THREADLENS_SMOKE_DRY_RUN=1 bash "${SMOKE_SCRIPT}")"

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
