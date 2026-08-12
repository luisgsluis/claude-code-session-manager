# Claude Code Session Manager (CCSM)

Web app to manage [Claude Code](https://claude.ai/code) sessions from any device on your LAN. Create, resume, rename, and monitor Claude sessions — with profile switching, editable settings, user management, conversation search, and Remote Control support.

**Stack**: Go + Alpine.js + Tailwind CSS | **Image**: ~15 MB (Alpine) | **RAM**: ~10-15 MB | **Version**: 0.1.5

![License](https://img.shields.io/badge/license-MIT-blue)

## Features

- 🚀 **One-click sessions**: new Claude session, with profile, or resume a past conversation (top action bar)
- ✏️ **Rename sessions**: rename the tmux session AND set the Claude conversation title — the title is typed as `/rename <title>` into the pane, so punctuation like `!` is fine
- 🔍 **Search conversations**: full-text search across all your `.jsonl` history, filter by machine origin
- 👁 **Preview conversations** before resuming
- ⚙️ **Profile switching**: apply any Claude Code profile from your catalog
- 📄 **Profile viewer**: inspect a profile's JSON with syntax highlighting before applying it
- ⚙️ **Editable settings**: the ☰ menu edits hot-reloadable settings in place (no restart), persisted atomically to `config.yaml`; restart-only settings are left untouched
- 👥 **User management**: add / delete users and change passwords from the UI — min 8-char passwords, bcrypt-hashed, hashes never exposed
- 🔧 **Active settings viewer**: see the CURRENTLY APPLIED `settings.json` (distinct from the saved profile files) with syntax highlighting
- 🌐 **Multi-language UI** (ES/EN via a 🌐 dropdown), selectable per user and remembered in localStorage
- 📱 **Responsive UI**: works on desktop and mobile, with a future mobile-native skin planned
- 🔒 **Security in depth**: login form + LAN bypass + Unix socket agent + restricted tmux/claude execution
- 🐳 **Two deployment modes**: container (Alpine image + host agent over Unix socket) **or** plain binary on the host (no container, no agent, in-process execution)
- 🩺 **Healthcheck**: `/api/health` wired into the image so orchestrators and the homelab updater can verify liveness

## Requirements

- **Host**: Linux with `tmux` and `claude` (Claude Code CLI) installed
- **Container mode**: Docker or Podman
- **Package mode**: nothing but the `ccsm` binary (runs in-process, no agent, no container)
- **Architecture**: `amd64` or `arm64` (ARMv7 on request)

## How it works

CCSM manages Claude Code sessions **on a host that has `tmux` and `claude` installed** — the
host where the sessions actually live. On top of that requirement there are two ways to run
the web app:

- **Package mode**: the `ccsm` binary runs directly on that host and executes tmux/claude
  in-process (no agent, no container).
- **Container mode**: the `ccsm` web server runs in a container (which has neither tmux nor
  claude) and talks over a Unix socket to the **`ccsm-agent`**, which **is installed on that
  same host** and is the one that launches tmux/claude.

Either way the non-negotiable requirement is the host with `tmux` + `claude`; the agent only
exists in container mode.

## Two deployment modes

| | **Package** | **Container** |
|---|---|---|
| Artifacts | `ccsm` binary only | Docker image + `ccsm-agent` binary on the host |
| Extra process | none | `ccsm-agent` (Unix socket daemon) |
| `agent_socket` in config | `""` (empty) | `/run/ccsm/agent.sock` |
| Config paths used | yes (or `auto` → `$HOME`) | hints only; agent uses its own flags |
| When to choose | host has the binaries, don't want Docker | isolated, single image, multi-host |

Both modes share `internal/host` as the single source of command logic — the
container mode wraps it in the Unix-socket agent, package mode in a direct
in-process executor, so the API, UI and behaviour are identical.

## Quick Start

### 1. Install the host agent

Download the `ccsm-agent` binary for your architecture from the [releases page](https://github.com/luisgsluis/claude-code-session-manager/releases).

```bash
sudo cp ccsm-agent-linux-arm64 /usr/local/bin/ccsm-agent
sudo chmod +x /usr/local/bin/ccsm-agent
```

Generate a shared secret:

```bash
openssl rand -base64 32
```

Run the agent (systemd unit recommended for production, see below):

```bash
ccsm-agent --secret "YOUR_BASE64_SECRET" --socket /run/ccsm/agent.sock &
```

### 2. Configure CCSM

```bash
mkdir ccsm-config
cd ccsm-config
curl -O https://raw.githubusercontent.com/luisgsluis/claude-code-session-manager/main/config.example.yaml
mv config.example.yaml config.yaml
```

Edit `config.yaml`:

```yaml
port: 8080
session_secret: ""                            # generate: ccsm --generate-secret
lan_subnets:
  - "192.168.1.0/24"
  - "10.0.0.0/8"

users:
  - username: "admin"
    password_hash: ""                         # generate: ccsm --hash-password

agent_socket: "/run/ccsm/agent.sock"
agent_secret: "YOUR_BASE64_SECRET"            # same secret given to ccsm-agent
host_attach_addr: "admin@myhost.local"        # shown in the UI for tmux attach
```

### 3. Run the container

```yaml
# docker-compose.yml
services:
  ccsm:
    image: ghcr.io/luisgsluis/ccsm:latest
    container_name: claude-sessions
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./config.yaml:/config/config.yaml:ro
      - /run/ccsm/agent.sock:/run/ccsm/agent.sock
    environment:
      - TZ=Europe/Madrid
```

```bash
docker compose up -d
```

Open `http://<host-ip>:8080`. From your LAN, login is bypassed automatically. From outside, you'll see the login form.

### 4. Host agent as a systemd service (recommended)

Put the shared secret in a file (readable only by the agent user) and pass
`--secret-file` — this keeps the secret out of `ps aux` and out of the unit
itself. The directory `/run/ccsm` is created at boot by a tmpfiles rule.

```
# /etc/tmpfiles.d/ccsm.conf
d /run/ccsm 0700 admin admin -
```

```bash
printf '%s' 'YOUR_BASE64_SECRET' | sudo tee /etc/ccsm/agent-secret
sudo chmod 640 /etc/ccsm/agent-secret   # readable by the agent user (group)
```

```
# /etc/systemd/system/ccsm-agent.service
[Unit]
Description=CCSM Host Agent
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/ccsm-agent \
  --secret-file /etc/ccsm/agent-secret \
  --socket /run/ccsm/agent.sock \
  --profiles /home/admin/claude-shared/claude-perfiles \
  --settings /home/admin/.claude/settings.json \
  --conversations /home/admin/.claude/projects/-home-admin
User=admin
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

The agent has a **singleton guard**: if another instance already answers on the
socket it refuses to start; if the socket is stale (crash leftover) it removes it
and takes over. `ccsm --secret-file` and `--secret` are equivalent; the file form
is preferred for services.

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now ccsm-agent
```

### Alternative: package mode (no container, no agent)

If the host already has `tmux` and `claude` and you don't want Docker, run the `ccsm`
binary directly. It executes every command **in-process** — no agent, no Unix socket.

```bash
# Download the ccsm binary for your architecture from the releases page
sudo cp ccsm-linux-arm64 /usr/local/bin/ccsm
sudo chmod +x /usr/local/bin/ccsm

# config.yaml: set agent_socket to "" and point the paths at your setup
cat > /etc/ccsm/config.yaml <<'EOF'
port: 8080
session_secret: "<ccsm --generate-secret>"
lan_subnets: ["192.168.1.0/24"]
users:
  - username: "admin"
    password_hash: "<ccsm --hash-password>"
agent_socket: ""
host_attach_addr: "admin@myhost.local"
conversations_path: "/home/admin/.claude/projects/-home-admin"
profiles_path: "/home/admin/claude-shared/claude-perfiles"
settings_path: "/home/admin/.claude/settings.json"
EOF

ccsm --config /etc/ccsm/config.yaml
```

Systemd unit example:

```
# /etc/systemd/system/ccsm.service
[Unit]
Description=CCSM (Claude Code Session Manager)
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/ccsm --config /etc/ccsm/config.yaml
User=admin
Restart=on-failure

[Install]
WantedBy=multi-user.target
```

The `ccsm` user needs the same host access the agent would: read the conversations
directory, read the profiles directory, write `settings.json`, and run `tmux` + `claude`
as that user. See [docs/security.md](docs/security.md) → "Host permissions required".

## Configuration Reference

All settings can be set in `config.yaml` or overridden via environment variables.

| YAML key | Env var | Default | Description |
|----------|---------|---------|-------------|
| `port` | `CCSM_PORT` | `8080` | HTTP listen port |
| `session_secret` | `CCSM_SESSION_SECRET` | — | HMAC secret for session cookies |
| `lan_subnets` | `CCSM_LAN_SUBNETS` | — | CIDRs that bypass login (comma-separated in env) |
| `users[].username` | — | — | Login username |
| `users[].password_hash` | — | — | bcrypt hash (`ccsm --hash-password`) |
| `agent_socket` | `CCSM_AGENT_SOCKET` | `/run/ccsm/agent.sock` | Unix socket to ccsm-agent; `""` = direct/package mode |
| `agent_secret` | `CCSM_AGENT_SECRET` | — | Shared secret for agent auth (container mode) |
| `host_attach_addr` | `CCSM_HOST_ATTACH_ADDR` | `localhost` | Shown in UI as `ssh <addr> -t tmux a -t N` |
| `conversations_path` | `CCSM_CONVERSATIONS_PATH` | `auto` | Conversation `.jsonl` dir (used in direct mode; hint for agent) |
| `profiles_path` | `CCSM_PROFILES_PATH` | `auto` | Profile `.json` dir (used in direct mode; hint for agent) |
| `settings_path` | `CCSM_SETTINGS_PATH` | `auto` | `settings.json` path (used in direct mode; hint for agent) |
| `claude_binary` | `CCSM_CLAUDE_BINARY` | `$HOME/.local/bin/claude` | Claude CLI path (direct mode only) |
| `tmux_binary` | `CCSM_TMUX_BINARY` | `/usr/bin/tmux` | tmux path (direct mode only) |
| `bash_binary` | `CCSM_BASH_BINARY` | `/usr/bin/bash` | bash path (direct mode only) |
| `rc.bootstrap_profile` | `CCSM_RC_BOOTSTRAP_PROFILE` | `estandar` | Profile used for RC bootstrap |
| `rc.wait_seconds` | `CCSM_RC_WAIT_SECONDS` | `25` | Max seconds to wait for RC bridge |
| `rc.poll_seconds` | `CCSM_RC_POLL_SECONDS` | `2` | Seconds between RC status polls |

Some of these are **hot-reloadable** from the UI's ☰ menu (`lan_subnets`,
`host_attach_addr`, and the `rc.*` block) — see [API](#patch-apiconfig-semantics)
below. The rest (`port`, `agent_socket`, `agent_secret`, `session_secret`, paths,
and the `claude`/`tmux`/`bash` binaries) require a service restart.

### CLI Tools

```bash
# Generate a random session secret
ccsm --generate-secret

# Hash a password for config.yaml
ccsm --hash-password "my-password"
```

## API

All endpoints live under `/api` and are protected by authentication — a session
cookie or the LAN-subnet bypass — except `GET /api/health`, which is public and
used by the container healthcheck.

**Sessions**
- `GET /api/sessions` — list sessions
- `POST /api/sessions/new` — start a new session (optionally with a profile)
- `POST /api/sessions/resume` — resume a past conversation
- `DELETE /api/sessions/{name}` — kill a session
- `POST /api/sessions/{name}/rename` — rename the tmux session; body `{"new_name": "..."}`
- `POST /api/sessions/{name}/claude-name` — set the Claude conversation title; body `{"title": "..."}`. Implemented by typing `/rename <title>` + Enter into the pane, so the title may include punctuation (including `!`).

**Profiles & conversations**
- `GET /api/profiles` — list profiles
- `GET /api/profiles/{name}` — profile content viewer
- `POST /api/profiles/apply` — apply a profile
- `GET /api/conversations` — list conversations
- `GET /api/conversations/{id}` — get one conversation

**Config & users**
- `GET /api/config` — non-secret deployment info for the ☰ menu
- `PATCH /api/config` — edit hot-reloadable settings, persisted atomically
- `GET /api/config/users` — list usernames
- `POST /api/config/users` — add a user
- `DELETE /api/config/users/{username}` — delete a user
- `POST /api/config/users/{username}/password` — change a password
- `GET /api/settings` — content of the CURRENTLY APPLIED `settings.json` (the active profile; may differ from the profile files)

**Auth**
- `POST /api/auth/login`, `POST /api/auth/logout`, `GET /api/auth/status`

### PATCH /api/config semantics

- **Hot-reloads in memory, no restart**: `lan_subnets` (each CIDR-validated), `host_attach_addr`, and `rc.{bootstrap_profile, wait_seconds, poll_seconds}`. RC changes affect only NEW sessions.
- **Not accepted** by this endpoint (need a service restart): `port`, `agent_socket`, `agent_secret`, `session_secret`, paths, and the `claude`/`tmux`/`bash` binaries.
- The write is **atomic**: temp file + rename.
- Response: `{"ok": true, "updated": [...], "restart_needed": false}`; `400` with `{"error": "..."}` on an invalid value or when no field is valid.

### User management

- Passwords are hashed with **bcrypt (cost 10)**; the API returns usernames only — hashes and passwords never leave the server.
- Password minimum is **8 characters**, enforced on both client and server.
- You **cannot delete the last remaining user** (`400`).

### Validation patterns

Enforced in `internal/host` on every entry point, and again in the web handlers:

| Field | Pattern |
|-------|---------|
| Session name | `^[A-Za-z0-9_-]{1,32}$` |
| Profile name | `^[a-z0-9][a-z0-9_-]{0,31}$` |
| Conversation ID | UUID (`8-4-4-4-12` hex) |
| Claude title | `^[\p{L}\p{N}\p{P} ]{1,80}$` — letters, numbers, punctuation (incl. `!`), spaces |
| Username | `^[a-z0-9][a-z0-9_-]{0,31}$` |
| `host_attach_addr` | `^[a-zA-Z0-9@._-]{1,120}$` |

Session names are free-form within the pattern; `0` is a valid session name (no longer reserved).

## Architecture

**Container mode** (agent_socket set):

```
 Browser                     Container (Alpine)              Host (Linux)
┌──────────┐    HTTP    ┌──────────────────────────┐    Unix socket   ┌──────────────┐
│  Mobile   │◄─────────►│  CCSM (Go binary)         │◄──────────────►│  ccsm-agent  │
│  Desktop  │           │  • Serves UI + API        │  shared secret  │  • tmux      │
│  Tablet   │           │  • Auth (bcrypt + LAN)    │                 │  • claude    │
└──────────┘           │  • Agent client           │                 │  • profiles  │
                       └──────────────────────────┘                 │  • .jsonl    │
                                                                     └──────────────┘
```

1. **Browser** → HTTP to CCSM container
2. **CCSM** → Unix socket to `ccsm-agent` on the host
3. **ccsm-agent** → executes `tmux`, `claude`, reads profiles and conversation files on the host

The container runs Alpine (musl libc), so it **cannot** directly execute host binaries (glibc). The Unix socket agent bridges this gap cleanly.

**Package mode** (agent_socket `""`): the same `ccsm` binary runs on the host and executes
`tmux`/`claude` **in-process**. The command logic is a shared `internal/host` package; the
container mode wraps it in the Unix-socket agent, package mode in a direct in-process
executor. Both answer the same API, so the UI, auth and endpoints are identical.

```
 Browser ──HTTP──►  ccsm (host, in-process) ──exec──►  tmux / claude
```

The `ccsm-agent` binary has a **singleton guard**: it refuses to start if a socket is
already answering, and cleans up a stale socket left by a crash. See
[docs/architecture.md](docs/architecture.md).

## Security Model

### Defense in Depth

| Layer | Mechanism |
|-------|-----------|
| **Network** | Publish only on LAN (`127.0.0.1:8080:8080` or use reverse proxy with TLS) |
| **Authentication** | Login form with bcrypt password hashing (cost 10). LAN IPs bypass login automatically |
| **User management** | Passwords min 8 chars (client + server); API exposes usernames only — hashes never leave the server; cannot delete the last user |
| **Session** | Encrypted cookie (HMAC-SHA256), `HttpOnly`, `SameSite=Lax`, 24h TTL |
| **Agent auth** | Shared secret sent in every request to ccsm-agent |
| **Agent execution** | Argument validation with closed regex patterns, no shell interpolation |
| **Config persistence** | Settings edits are written atomically (temp file + rename), never corrupting `config.yaml` |
| **Unix socket** | File permissions (`0600`) restrict access to socket owner only |

### HTTP Security Headers

```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Content-Security-Policy: default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self'
```

### ⚠️ IMPORTANT: Never put API keys in profiles or settings

Claude Code profiles and `settings.json` should **never contain API keys directly**.
Use `apiKeyHelper` instead — a script or binary that prints the key to stdout at runtime.
This keeps secrets out of Syncthing, out of git, and out of any sync mechanism.

```json
{
    "apiKeyHelper": "$HOME/.local/bin/claude-apikey"
}
```

The helper script (mode `700`, never synced) simply prints the key:

```sh
#!/bin/sh
printf '%s' 'sk-your-api-key'
```

CCSM does not read or transmit API keys. The profiles it manages are standard Claude Code
`settings.json` files. The same security rules apply.

## Troubleshooting

### Sessions don't appear after creation
The session may have aborted at startup (invalid profile, missing `--resume` id, trust dialog).
Check: `tmux list-sessions` on the host. If the session died, the Claude error will be in
the tmux pane scrollback.

### "agent error" in all responses
Check that `ccsm-agent` is running on the host and the socket path matches:
```bash
ls -la /run/ccsm/agent.sock
systemctl status ccsm-agent
```

### Login page loops / can't authenticate
- Verify `session_secret` is set (not empty)
- Check that `users[].password_hash` was generated with `ccsm --hash-password`, not raw text
- From LAN: check that your subnet is in `lan_subnets`

### Renaming the Claude title fails with "can't find pane"
This is the tmux `send-keys` target issue. tmux accepts the `=` prefix only for
SESSION targets (`rename-session`, `has-session`, `kill-session`); it is INVALID
for PANE targets — `tmux send-keys -t =<name>` fails with "can't find pane".
CCSM therefore targets the **bare session name** for the Claude-title rename; if
you ever see this error it means a `=` prefix leaked into the send-keys target.

### Settings edits "don't take effect"
Only hot-reloadable fields can be edited from the ☰ menu without a restart:
`lan_subnets`, `host_attach_addr`, and `rc.{bootstrap_profile, wait_seconds, poll_seconds}`.
`port`, `agent_socket`, `agent_secret`, `session_secret`, paths, and the
`claude`/`tmux`/`bash` binaries require a service restart — the API rejects them
with a `400` so they never silently apply.

### Profile changes don't take effect
- The `ccsm-agent` needs the correct `--profiles` and `--settings` paths
- If `settings.json` is a symlink, the agent writes through it correctly (`os.WriteFile` — the link is followed, never replaced)
- Model/endpoint changes hot-reload. Remote Control enablement is decided at session startup

### Remote Control not working after profile switch
Non-Anthropic profiles (with `apiKeyHelper` or custom `ANTHROPIC_BASE_URL`) disable
Remote Control at startup. CCSM works around this with a two-phase bootstrap:
1. Temporarily apply the `rc.bootstrap_profile` (must be a clean, RC-friendly profile)
2. Start the session with `--remote-control`
3. Wait for the RC bridge to connect (up to `rc.wait_seconds`)
4. Hot-apply the target profile (RC survives the switch)

If RC still fails: check that the bootstrap profile exists in your profiles directory
and has `remoteControlAtStartup: true` without `apiKeyHelper` or non-Anthropic URLs.

Profiles that do NOT disable RC skip the bootstrap entirely and start with
`claude --settings <profile> --remote-control` — the global `settings.json` is never
touched for those sessions.

## Development

```bash
go build ./...            # build the web server and agent
go test ./...             # unit tests
cd e2e && npx playwright test   # end-to-end tests (stubbed tmux/claude)
```

## License

MIT. See [LICENSE](LICENSE).
