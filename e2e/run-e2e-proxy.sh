#!/usr/bin/env bash
# Third e2e server, for auth: proxy (proxy-auth.spec.js).
#
# The browser connects from loopback, and loopback is the trusted proxy here,
# so the browser stands in for "the request came through the authenticating
# proxy"; the spec sends the Remote-User header that proxy would add. There is
# no LAN bypass and a user is configured, to prove neither matters in this mode.
set -euo pipefail

E2E="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$E2E/.." && pwd)"
STATE="$E2E/state-proxy"

rm -rf "$STATE"
mkdir -p "$STATE/conversations" "$STATE/profiles"

cat > "$E2E/config-proxy.yaml" <<YAML
port: 8796
auth: proxy
trusted_proxies:
  - "127.0.0.0/8"
  - "::1/128"
agent_socket: ""
agent_secret: ""
lan_subnets: []
users:
  - username: e2e
    password_hash: "\$2a\$10\$H/SC9MUlyPBtcbU1Y8/EMu1vClnnTOUf8gK3jQ7WDv.8.5pwwTQ4W"
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

# Own binary, for the same reason as run-e2e-auth.sh: builds started together
# must not write the same file.
(cd "$ROOT" && CGO_ENABLED=0 go build -o "$E2E/ccsm-e2e-proxy" ./cmd/ccsm)

export HOME="$STATE"

exec "$E2E/ccsm-e2e-proxy" --config "$E2E/config-proxy.yaml"
