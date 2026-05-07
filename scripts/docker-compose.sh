#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
DEFAULT_ENV_FILE="$ROOT_DIR/.env"
ENV_FILE="${SCOUT_ENV_FILE:-$DEFAULT_ENV_FILE}"
ENV_EXAMPLE_FILE="$ROOT_DIR/.env.example"
COMPOSE_FILE="$ROOT_DIR/infra/docker/compose.yml"
COMMAND="${1:-}"

ensure_env_file() {
  if [[ -f "$ENV_FILE" ]]; then
    return 0
  fi

  if [[ "$ENV_FILE" != "$DEFAULT_ENV_FILE" ]]; then
    printf 'Missing env file: %s\n' "$ENV_FILE" >&2
    printf 'Create it from %s and any SaaS-only overlay before running Docker.\n' "$ENV_EXAMPLE_FILE" >&2
    exit 1
  fi

  if [[ ! -f "$ENV_EXAMPLE_FILE" ]]; then
    printf 'Missing %s\n' "$ENV_EXAMPLE_FILE" >&2
    exit 1
  fi

  cp "$ENV_EXAMPLE_FILE" "$ENV_FILE"
  printf 'Created %s from %s\n' "$ENV_FILE" "$ENV_EXAMPLE_FILE"
  printf 'Fill in provider keys in open-core/.env for Google or Bluesky features.\n'
}

cd "$ROOT_DIR"

case "$COMMAND" in
  dev)
    ensure_env_file
    exec docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" --profile dev up --build -d
    ;;
  prod)
    ensure_env_file
    exec docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" --profile prod up --build -d
    ;;
  down)
    if [[ -f "$ENV_FILE" ]]; then
      docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" --profile dev down
      exec docker compose --env-file "$ENV_FILE" -f "$COMPOSE_FILE" --profile prod down
    fi
    docker compose -f "$COMPOSE_FILE" --profile dev down
    exec docker compose -f "$COMPOSE_FILE" --profile prod down
    ;;
  *)
    printf 'Usage: %s {dev|prod|down}\n' "$(basename "$0")" >&2
    exit 1
    ;;
esac
