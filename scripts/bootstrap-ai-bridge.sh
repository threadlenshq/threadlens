#!/usr/bin/env bash
# bootstrap-ai-bridge.sh
# Sets up the Scout AI bridge configuration for local development and Docker environments.
# Safe to run multiple times (idempotent).
set -uo pipefail

# ---------------------------------------------------------------------------
# Configuration defaults
# ---------------------------------------------------------------------------
BRIDGE_PORT="${SCOUT_BRIDGE_PORT:-4761}"
HOST_URL="http://127.0.0.1:${BRIDGE_PORT}"
DOCKER_URL="http://host.docker.internal:${BRIDGE_PORT}"
TOKEN_FILE_IN_CONTAINER="/run/secrets/scout-ai-bridge-token"

# Root dir for the open-core project (override for tests)
ROOT_DIR="${SCOUT_BRIDGE_ROOT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"

# XDG config directory for the host token file
XDG_CONFIG="${XDG_CONFIG_HOME:-${HOME}/.config}"
TOKEN_DIR="${XDG_CONFIG}/scout"
TOKEN_FILE="${TOKEN_DIR}/ai-bridge.token"
CONFIG_FILE="${TOKEN_DIR}/ai-bridge.json"

# XDG state directory for PID tracking
XDG_STATE="${XDG_STATE_HOME:-${HOME}/.local/state}"
STATE_DIR="${XDG_STATE}/scout"
PID_FILE="${STATE_DIR}/ai-bridge.pid"

ENV_FILE="${ROOT_DIR}/.env"
ENV_EXAMPLE="${ROOT_DIR}/.env.example"

# Allow overriding uname for tests
UNAME_OVERRIDE="${SCOUT_BRIDGE_UNAME:-}"

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------
log() { printf '[bootstrap-ai-bridge] %s\n' "$*" >&2; }

remove_env_var() {
  local key="$1"
  local file="$2"

  [[ -f "$file" ]] || return 0

  local tmp
  tmp="$(mktemp)"
  awk -v key="$key" 'index($0, key "=") != 1 { print }' "$file" > "$tmp"
  mv "$tmp" "$file"
}

clear_bridge_env_vars() {
  remove_env_var "SCOUT_AI_BRIDGE_MODE" "$ENV_FILE"
  remove_env_var "SCOUT_AI_BRIDGE_URL" "$ENV_FILE"
  remove_env_var "SCOUT_AI_BRIDGE_TOKEN_FILE" "$ENV_FILE"
  remove_env_var "SCOUT_AI_BRIDGE_HOST_TOKEN_FILE" "$ENV_FILE"
}

is_supported_os() {
  local uname_val
  uname_val="${UNAME_OVERRIDE:-$(uname -s 2>/dev/null || echo 'Unknown')}"
  case "$uname_val" in
    Linux* | Darwin*) return 0 ;;
    *) return 1 ;;
  esac
}

ensure_env_file() {
  if [[ ! -f "$ENV_FILE" ]]; then
    if [[ -f "$ENV_EXAMPLE" ]]; then
      cp "$ENV_EXAMPLE" "$ENV_FILE"
      log "Created $ENV_FILE from $ENV_EXAMPLE"
    else
      touch "$ENV_FILE"
      log "Created empty $ENV_FILE (no .env.example found)"
    fi
  fi
}

generate_token() {
  # Try openssl first, then python3, then /dev/urandom fallback
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 32
  elif command -v python3 >/dev/null 2>&1; then
    python3 -c "import secrets; print(secrets.token_hex(32))"
  else
    # /dev/urandom fallback — works on Linux and macOS
    LC_ALL=C tr -dc 'a-f0-9' </dev/urandom 2>/dev/null | head -c 64
    echo
  fi
}

ensure_token() {
  mkdir -p "$TOKEN_DIR"
  chmod 700 "$TOKEN_DIR"

  local existing=""
  if [[ -f "$TOKEN_FILE" ]]; then
    existing="$(cat "$TOKEN_FILE")"
    # Strip whitespace
    existing="${existing//[$'\t\r\n ']}"
  fi

  if [[ ${#existing} -ge 32 ]]; then
    log "Host token already valid, preserving."
  else
    local token
    token="$(generate_token)"
    printf '%s\n' "$token" > "$TOKEN_FILE"
    chmod 600 "$TOKEN_FILE"
    log "Generated new host token at $TOKEN_FILE"
  fi
}

write_config() {
  cat > "$CONFIG_FILE" <<EOF
{
  "type": "http-localhost",
  "url": "${HOST_URL}",
  "tokenFile": "${TOKEN_FILE}",
  "runtimes": ["copilot", "claude-cli", "opencode"]
}
EOF
  log "Wrote $CONFIG_FILE"
}

set_env_var() {
  local key="$1"
  local value="$2"
  local file="$3"

  if grep -q "^${key}=" "$file" 2>/dev/null; then
    # Replace existing line (portable sed -i for both Linux and macOS)
    local tmp
    tmp="$(mktemp)"
    sed "s|^${key}=.*|${key}=${value}|" "$file" > "$tmp" && mv "$tmp" "$file"
  else
    printf '%s=%s\n' "$key" "$value" >> "$file"
  fi
}

write_env_vars() {
  set_env_var "SCOUT_AI_BRIDGE_MODE" "local" "$ENV_FILE"
  set_env_var "SCOUT_AI_BRIDGE_URL" "$DOCKER_URL" "$ENV_FILE"
  set_env_var "SCOUT_AI_BRIDGE_TOKEN_FILE" "$TOKEN_FILE_IN_CONTAINER" "$ENV_FILE"
  set_env_var "SCOUT_AI_BRIDGE_HOST_TOKEN_FILE" "$TOKEN_FILE" "$ENV_FILE"
  log "Updated $ENV_FILE with bridge env vars"
}

health_response() {
  local token
  token="$(cat "$TOKEN_FILE" 2>/dev/null || echo '')"
  if command -v curl >/dev/null 2>&1; then
    curl -sf -H "Authorization: Bearer ${token}" "${HOST_URL}/v1/health" 2>/dev/null
    return $?
  fi

  return 1
}

health_ok() {
  health_response >/dev/null
}

bridge_has_available_runtime() {
  local response
  response="$(health_response)" || return 1

  [[ "$response" == *'"available":true'* ]] && return 0
  [[ "$response" =~ \"runtimes\"[[:space:]]*:[[:space:]]*\[[[:space:]]*\"[^\"]+ ]] && return 0

  return 1
}

write_pid() { printf '%s\n' "$1" > "$PID_FILE"; }
read_pid() { [[ -f "$PID_FILE" ]] && cat "$PID_FILE" || echo ''; }
pid_alive() { local p; p="$(read_pid)"; [[ -n "$p" ]] && kill -0 "$p" 2>/dev/null; }

kill_bridge_on_port() {
  local port_pid
  if command -v lsof >/dev/null 2>&1; then
    port_pid="$(lsof -ti ":${BRIDGE_PORT}" 2>/dev/null || true)"
  elif command -v fuser >/dev/null 2>&1; then
    port_pid="$(fuser "${BRIDGE_PORT}/tcp" 2>/dev/null || true)"
  fi
  if [[ -n "$port_pid" ]]; then
    kill $port_pid 2>/dev/null || true
    sleep 1
  fi
}

build_binary() {
  local bin_dir="${ROOT_DIR}/bin"
  mkdir -p "$bin_dir"
  local module_dir="${ROOT_DIR}/apps/api"
  if command -v go >/dev/null 2>&1 && [[ -f "${module_dir}/cmd/scout-ai-bridge/main.go" ]]; then
    (
      cd "$module_dir" &&
      go build -o "${bin_dir}/scout-ai-bridge" ./cmd/scout-ai-bridge
    ) >/dev/null 2>&1 && \
      log "Built scout-ai-bridge binary" || \
      log "Build failed; bridge binary must be compiled manually"
  fi
}

start_daemon() {
  local binary="${ROOT_DIR}/bin/scout-ai-bridge"
  local token
  token="$(cat "$TOKEN_FILE" 2>/dev/null || echo '')"

  # Bridge is fully healthy: responding AND has at least one working runtime.
  if bridge_has_available_runtime; then
    log "AI bridge already running at ${HOST_URL} with available runtimes"
    return 0
  fi

  # Bridge responds to health but has no working runtimes — process is alive but
  # broken (e.g. CLI auth expired after a restart).  Kill and restart it.
  if health_ok; then
    log "Bridge running but no available runtimes; restarting..."
    local managed_pid
    managed_pid="$(read_pid)"
    if [[ -n "$managed_pid" ]] && kill -0 "$managed_pid" 2>/dev/null; then
      kill "$managed_pid" 2>/dev/null || true
      sleep 1
    fi
    if health_ok; then
      kill_bridge_on_port
    fi
  fi

  # Stale PID file (process dead but port might be used by something else).
  if pid_alive && ! health_ok; then
    log "Stale bridge PID found; cleaning up..."
    local old_pid
    old_pid="$(read_pid)"
    kill "$old_pid" 2>/dev/null || true
    sleep 1
  fi

  # Try to build if binary not present
  if [[ ! -x "$binary" ]]; then
    build_binary || true
  fi

  if [[ ! -x "$binary" ]]; then
    log "scout-ai-bridge binary not found; skipping daemon start"
    return 0
  fi

  mkdir -p "$STATE_DIR"

  SCOUT_AI_BRIDGE_BIND="127.0.0.1:${BRIDGE_PORT}" \
  SCOUT_AI_BRIDGE_TOKEN="$token" \
    "$binary" &>/dev/null &
  local daemon_pid=$!
  disown $daemon_pid 2>/dev/null || true
  write_pid "$daemon_pid"
  log "Started scout-ai-bridge (pid $daemon_pid)"

  # Best-effort health check (up to 5 seconds)
  local i=0
  while (( i < 5 )); do
    sleep 1
    if health_ok; then
      log "AI bridge health check passed"
      return 0
    fi
    (( i++ )) || true
  done
  log "Warning: AI bridge health check did not pass within 5 seconds (non-fatal)"
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
  if ! is_supported_os; then
    log "Skipping host CLI bridge bootstrap (unsupported OS: ${UNAME_OVERRIDE:-$(uname -s 2>/dev/null || echo Unknown)})"
    return 0
  fi

  ensure_env_file

  # Allow disabling bridge variable injection entirely
  if [[ "${SCOUT_AI_BRIDGE_DISABLE:-}" == "1" ]]; then
    clear_bridge_env_vars
    log "SCOUT_AI_BRIDGE_DISABLE=1: skipping bridge env var injection"
    return 0
  fi

  ensure_token
  write_config

  if [[ "${SCOUT_BRIDGE_NO_LAUNCH:-}" == "1" ]]; then
    log "SCOUT_BRIDGE_NO_LAUNCH=1: skipping daemon start"
  else
    start_daemon
  fi

  if bridge_has_available_runtime; then
    write_env_vars
    return 0
  fi

  clear_bridge_env_vars
  log "Bridge not usable yet; leaving Docker bridge env vars disabled"
}

main "$@"
