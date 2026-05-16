#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="${SCOUT_BRIDGE_ROOT_DIR:-$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)}"
BRIDGE_PORT="${SCOUT_BRIDGE_PORT:-4761}"
BRIDGE_BIND="${SCOUT_AI_BRIDGE_BIND:-127.0.0.1:${BRIDGE_PORT}}"
HOST_URL="http://127.0.0.1:${BRIDGE_PORT}"
XDG_CONFIG="${XDG_CONFIG_HOME:-${HOME}/.config}"
XDG_STATE="${XDG_STATE_HOME:-${HOME}/.local/state}"
TOKEN_DIR="${XDG_CONFIG}/scout"
STATE_DIR="${XDG_STATE}/scout"
TOKEN_FILE="${TOKEN_DIR}/ai-bridge.token"
CONFIG_FILE="${TOKEN_DIR}/ai-bridge.json"
PID_FILE="${STATE_DIR}/ai-bridge.pid"
LOG_FILE="${STATE_DIR}/ai-bridge.log"
BINARY="${ROOT_DIR}/bin/scout-ai-bridge"
COMMAND="${1:-status}"

log() { printf '[ai-bridge] %s\n' "$*" >&2; }

generate_token() {
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 32
  elif command -v python3 >/dev/null 2>&1; then
    python3 -c "import secrets; print(secrets.token_hex(32))"
  else
    LC_ALL=C tr -dc 'a-f0-9' </dev/urandom 2>/dev/null | head -c 64
    printf '\n'
  fi
}

ensure_token() {
  mkdir -p "$TOKEN_DIR"
  chmod 700 "$TOKEN_DIR"

  local existing=""
  if [[ -f "$TOKEN_FILE" ]]; then
    existing="$(tr -d '[:space:]' < "$TOKEN_FILE")"
  fi

  if [[ ${#existing} -lt 32 ]]; then
    generate_token > "$TOKEN_FILE"
    chmod 600 "$TOKEN_FILE"
    log "Generated token at $TOKEN_FILE"
  fi
}

write_config() {
  cat > "$CONFIG_FILE" <<EOF
{
  "type": "http-localhost",
  "url": "${HOST_URL}",
  "tokenFile": "${TOKEN_FILE}",
  "runtimes": ["copilot", "claude-cli"]
}
EOF
}

build_binary() {
  mkdir -p "${ROOT_DIR}/bin"
  (
    cd "${ROOT_DIR}/apps/api" &&
    go build -o "$BINARY" ./cmd/scout-ai-bridge
  )
}

ensure_binary() {
  if [[ -x "$BINARY" ]]; then
    return 0
  fi

  if ! command -v go >/dev/null 2>&1; then
    log "Go is required to build $BINARY"
    exit 1
  fi

  build_binary
}

auth_header() {
  printf 'Authorization: Bearer %s' "$(tr -d '[:space:]' < "$TOKEN_FILE")"
}

health_raw() {
  [[ -f "$TOKEN_FILE" ]] || return 1
  command -v curl >/dev/null 2>&1 || return 1
  curl -sf -H "$(auth_header)" "$HOST_URL/v1/health"
}

health_ok() {
  health_raw >/dev/null
}

read_pid() {
  [[ -f "$PID_FILE" ]] || return 1
  tr -d '[:space:]' < "$PID_FILE"
}

running_pid() {
  local pid
  pid="$(read_pid)" || return 1
  kill -0 "$pid" 2>/dev/null
}

ensure_runtime_state() {
  mkdir -p "$STATE_DIR"
}

bootstrap() {
  ensure_token
  write_config
  ensure_runtime_state
  ensure_binary
  log "Bridge bootstrap complete"
  log "Config: $CONFIG_FILE"
  log "Token: $TOKEN_FILE"
}

start_bridge() {
  bootstrap

  if health_ok; then
    log "Bridge already healthy at $HOST_URL"
    return 0
  fi

  if running_pid; then
    local pid
    pid="$(read_pid)"
    log "Bridge process $pid is running but health check failed"
    return 1
  fi

  nohup env \
    SCOUT_AI_BRIDGE_BIND="$BRIDGE_BIND" \
    SCOUT_AI_BRIDGE_TOKEN_FILE="$TOKEN_FILE" \
    "$BINARY" >> "$LOG_FILE" 2>&1 &
  local pid=$!
  printf '%s\n' "$pid" > "$PID_FILE"

  local i
  for i in 1 2 3 4 5; do
    sleep 1
    if health_ok; then
      log "Bridge started (pid $pid)"
      return 0
    fi
  done

  log "Bridge failed health check after start. See $LOG_FILE"
  return 1
}

stop_bridge() {
  if running_pid; then
    local pid
    pid="$(read_pid)"
    kill "$pid"
    rm -f "$PID_FILE"
    log "Stopped bridge process $pid"
    return 0
  fi

  rm -f "$PID_FILE"
  log "No managed bridge process found"
}

status_bridge() {
  if running_pid; then
    local pid
    pid="$(read_pid)"
    log "Bridge process running with pid $pid"
  elif health_ok; then
    log "Bridge is healthy, but not managed by this helper"
  else
    log "Bridge process not running"
  fi

  if health_ok; then
    log "Bridge health OK"
    health_raw
    return 0
  fi

  log "Bridge health unavailable"
  return 1
}

case "$COMMAND" in
  bootstrap)
    bootstrap
    ;;
  start)
    start_bridge
    ;;
  stop)
    stop_bridge
    ;;
  status)
    status_bridge
    ;;
  health)
    health_raw
    ;;
  *)
    printf 'Usage: %s {bootstrap|start|stop|status|health}\n' "$(basename "$0")" >&2
    exit 1
    ;;
esac
