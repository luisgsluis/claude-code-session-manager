#!/usr/bin/env bash
# Arranca el binario ccsm contra los stubs para las pruebas e2e de Playwright.
# Playwright lanza este script como webServer; `exec` hace que el PID del shell
# sea el del servidor para que el cierre sea limpio.
set -euo pipefail

E2E="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$E2E/.." && pwd)"
STATE="$E2E/state"

rm -rf "$STATE"
mkdir -p "$STATE/tmux" "$STATE/conversations" "$STATE/profiles"

cat > "$E2E/config.yaml" <<YAML
port: 8799
session_secret: e2e-session-secret-no-uso
agent_socket: ""
agent_secret: ""
lan_subnets:
  - "127.0.0.0/8"
  - "::1/128"
users: []
host_attach_addr: localhost
conversations_path: $STATE/conversations
profiles_path: $STATE/profiles
settings_path: $STATE/settings.json
audit_path: $STATE/audit.jsonl
claude_binary: $E2E/stubs/claude
tmux_binary: $E2E/stubs/tmux
bash_binary: /bin/bash
rc:
  bootstrap_profile: ""
  wait_seconds: 5
  poll_seconds: 1
YAML

if [ ! -x "$E2E/ccsm-e2e" ]; then
  (cd "$ROOT" && CGO_ENABLED=0 go build -o "$E2E/ccsm-e2e" ./cmd/ccsm)
fi

exec "$E2E/ccsm-e2e" --config "$E2E/config.yaml"
