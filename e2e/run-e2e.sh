#!/usr/bin/env bash
# Starts the ccsm binary against the stubs for Playwright's e2e tests.
# Playwright launches this script as its webServer; `exec` makes the shell's
# PID the server's own, so shutdown is clean.
set -euo pipefail

E2E="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$E2E/.." && pwd)"
STATE="$E2E/state"

rm -rf "$STATE"
mkdir -p "$STATE/tmux" "$STATE/conversations" "$STATE/profiles"
# "Projects" for the dropdown: discovery scans $HOME looking for CLAUDE.md,
# so we point HOME at $STATE and create two of them here (to be able to
# check the dropdown's alphabetical order).
mkdir -p "$STATE/projects/demo" "$STATE/projects/alpha"
printf '# demo\n' > "$STATE/projects/demo/CLAUDE.md"
printf '# alpha\n' > "$STATE/projects/alpha/CLAUDE.md"

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
voice:
  enabled: true
  language: es
  prompts_path: $STATE/prompts
  stt:
    mode: whisper
    provider: stub
    model: stub-whisper
    vocabulary: "sonarr, radarr, tmux, systemd, macvlan"
  rewrite:
    enabled: true
    provider: stub
    model: stub-chat
    default_role: auto
  providers:
    stub:
      base_url: http://127.0.0.1:8797/v1
      api_key: e2e-stub-key
      stt_models: [stub-whisper]
      chat_models: [stub-chat]
    chatonly:
      base_url: http://127.0.0.1:8797/v1
      api_key: e2e-stub-key
      chat_models: [stub-chat, stub-chat-2]
YAML

if [ ! -x "$E2E/ccsm-e2e" ]; then
  (cd "$ROOT" && CGO_ENABLED=0 go build -o "$E2E/ccsm-e2e" ./cmd/ccsm)
fi

# HOME pins the host's home (h.home), which is the root of project discovery
# and the default cwd for new sessions.
export HOME="$STATE"

exec "$E2E/ccsm-e2e" --config "$E2E/config.yaml"
