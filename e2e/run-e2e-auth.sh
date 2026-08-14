#!/usr/bin/env bash
# Second e2e server, for the login flow only.
#
# run-e2e.sh puts 127.0.0.0/8 in lan_subnets, so every other spec runs under
# the LAN bypass and never sees the login form. This one does the opposite —
# no bypass and one user with a second factor — so the password + TOTP steps
# can actually be driven in a browser.
set -euo pipefail

E2E="$(cd "$(dirname "$0")" && pwd)"
ROOT="$(cd "$E2E/.." && pwd)"
STATE="$E2E/state-auth"

rm -rf "$STATE"
mkdir -p "$STATE/conversations" "$STATE/profiles"

# bcrypt hash of "test123" — the same fixture the Go auth tests use.
# TOTP secret: the RFC 6238 test key, so the spec can compute codes for it.
cat > "$E2E/config-auth.yaml" <<YAML
port: 8798
session_secret: e2e-session-secret-no-uso
agent_socket: ""
agent_secret: ""
lan_subnets: []
users:
  - username: e2e
    password_hash: "\$2a\$10\$H/SC9MUlyPBtcbU1Y8/EMu1vClnnTOUf8gK3jQ7WDv.8.5pwwTQ4W"
    totp_secret: GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ
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

# Its own binary: Playwright starts both webServers at once, and two builds
# racing on the same output file would clobber each other. The Go build cache
# makes the second one nearly free.
(cd "$ROOT" && CGO_ENABLED=0 go build -o "$E2E/ccsm-e2e-auth" ./cmd/ccsm)

export HOME="$STATE"

exec "$E2E/ccsm-e2e-auth" --config "$E2E/config-auth.yaml"
