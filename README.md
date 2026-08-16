# Claude Code Session Manager (CCSM)

**Run [Claude Code](https://claude.ai/code) on your own server, drive it from anywhere, even if not using claudeai account.**

**You no longer need to be at your server — or hold an SSH session open — to start a Claude Code session in `tmux`.** You start it from a browser, it runs on the host, and it stays alive when you close the tab.

CCSM is a lightweight web app that turns any Linux box you control — a homelab server, a VPS, your workstation — into a personal Claude Code hub. Sessions run in `tmux` on that machine; you create, resume, and chat with them from a browser on your phone, tablet, or laptop — or attach the official Claude mobile app or claude.ai/code to that same session via Remote Control — even though the session itself runs on your own API key instead of your claude.ai subscription. Funnily enough, just two days after releasing this, Claude Code introduced a remote control mode, enabling users to start sessions on their own server remotely.

![License](https://img.shields.io/badge/license-MIT-blue)

## Why CCSM

- 🖥️ **Remote, effortless session management via `tmux`** — every session is a real `tmux` session on your own server. Close the laptop, lose the connection, come back tomorrow: the session is still there, exactly where you left it, manageable from any device.
- 📱 **Claude Code Web & App, without using your claude.ai subscription** — Remote Control lets the official Claude mobile app and claude.ai/code attach live to sessions that run on **your own API key** (Anthropic or an Anthropic-compatible provider) instead of a logged-in personal subscription. CCSM's two-phase bootstrap keeps that mobile bridge alive even across profile switches.
- ⚙️ **Advanced profile management** — catalog, preview, and hot-swap full `settings.json` profiles (different models, providers, keys) per session, applied atomically with JSON pre-validation so a bad profile can never brick `settings.json`.
- ♻️ **Session recovery on the server** — every conversation is a real Claude Code transcript stored on the host; browse, search, and resume any past session from any device, even after a crash or a reboot.
- 💬🖥️ **Chat mode and Terminal mode** — a clean chat view for quick back-and-forth with live SSE updates, or drop into the raw terminal pane when you need the full TUI — approvals, dialogs, everything.
- 🔍 **Advanced session search** — full-text search across your entire `.jsonl` conversation history, filterable by machine origin, date range, live/archived state, tags and notes.

## Features

- 🚀 **One-click sessions**: new Claude session, with profile, or resume a past conversation (top action bar)
- 🖥️ **Terminal grid**: tile every active session's raw terminal pane at once, in colour — minimize, zoom (tmux-style), and send text/keys per tile without leaving the view. Below 1024px, where a mosaic wouldn't fit, it becomes a one-session-at-a-time view instead: every tile starts minimized as a header chip and tapping one opens it full-screen, replacing whichever was open before
- ✏️ **Rename sessions**: rename the tmux session AND set the Claude conversation title — the title is typed as `/rename <title>` into the pane, so punctuation like `!` is fine
- 🔍 **Search conversations**: full-text search across all your `.jsonl` history, filter by machine origin
- 👁 **Preview conversations** before resuming
- ⚙️ **Profile switching**: apply any Claude Code profile from your catalog
- 📄 **Profile viewer**: inspect a profile's JSON with syntax highlighting before applying it
- ⚙️ **Editable settings**: the ☰ menu edits hot-reloadable settings in place (no restart), persisted atomically to `config.yaml`; restart-only settings are left untouched
- 👥 **User management**: add / delete users and change passwords from the UI — min 8-char passwords, bcrypt-hashed, hashes never exposed
- 🔐 **Two-factor authentication (TOTP)**: opt-in per user, enrolled by scanning a QR with Google Authenticator (Aegis, 1Password, any RFC 6238 app). LAN clients keep bypassing login; the second factor guards internet access
- 🔧 **Active settings viewer**: see the CURRENTLY APPLIED `settings.json` (distinct from the saved profile files) with syntax highlighting
- 🎨 **Selectable skins** (Light, Dark, Ocean, Contrast) via a header dropdown, remembered
  in localStorage — see [`docs/skins.md`](docs/skins.md) to add your own
- 🌐 **Multi-language UI** (ES/EN via a 🌐 dropdown), selectable per user and remembered in localStorage
- 📱 **Responsive UI**: works on desktop, tablet and mobile, with a future mobile-native skin planned
- 🔒 **Security in depth**: login form (optional TOTP) + per-IP rate limiting + LAN bypass + Unix socket agent + restricted tmux/claude execution
- 🐳 **Two deployment modes**: container (Alpine image + host agent over Unix socket) **or** plain binary on the host (no container, no agent, in-process execution)
- 🩺 **Healthcheck**: `/api/health` wired into the image so orchestrators and the homelab updater can verify liveness

## Requirements

**Stack**: Go + Alpine.js + Tailwind CSS | **Image**: ~15 MB (Alpine) | **RAM**: ~10-15 MB | **Version**: 1.5.2

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

## Profiles & Remote Control

Sessions are launched from a **profile**: a complete Claude Code `settings.json`
stored in the profiles catalog (`profiles_path`, e.g. `claude-shared/claude-perfiles`).
Applying a profile copies it over the global `settings.json` — CCSM validates the
JSON first, so a broken profile can never leave it unparseable. A profile decides
more than the model and endpoint: it decides **whether the session can be driven
remotely from the Claude mobile app / claude.ai/code**.

### The profile decides if the remote bridge registers

Claude Code's Remote Control registers a live session into your claude.ai account
only when it starts under a **clean Anthropic profile**:

- Profiles with `apiKeyHelper`, a static API key/token, or a non-Anthropic
  `ANTHROPIC_BASE_URL` disable Remote Control at startup (`perfilSinRC`). A
  session started under one never appears in the mobile app on its own.
- Clean profiles (no alternate credentials) register the bridge directly: CCSM
  launches them with `--remote-control` and the session shows up in the app.

For profiles that disable RC, CCSM runs a **two-phase bootstrap** so the mobile
bridge still registers:

1. Temporarily apply `rc.bootstrap_profile` — a clean, RC-friendly profile
   (`estandar` by default)
2. Start the session with `--remote-control`
3. Wait for the bridge to connect (up to `rc.wait_seconds`)
4. Hot-apply the target profile — RC survives the switch

### If the bootstrap profile is missing

If `rc.bootstrap_profile` names a profile that doesn't exist, CCSM does **not**
fail: it skips the staging, launches the session directly (no `--remote-control`)
and assumes there will be no bridge. The session stays fully usable from the
CCSM UI, it just won't appear in the mobile app, and `/rc` answers
`{"presses":0,"status":"fail"}` instead of attempting a recovery.

### Managing profiles for remote sessions

- Keep a clean bootstrap profile (`estandar`): Anthropic endpoint, no
  `apiKeyHelper`, no alternate base URL.
- **In container mode the staging profile is set on the agent**, via
  `--rc-profile` in the `ccsm-agent` unit — editing `rc.bootstrap_profile` in
  the server's `config.yaml` does not reach the agent.
- When a session loses its bridge, use **"RC: re-register"**
  (`POST /api/sessions/{name}/rc`): it re-runs the bootstrap and, if the bridge
  still doesn't come back, kills and resumes the conversation, whose two-phase
  launch re-registers it.
- Sessions you want to drive from the mobile app should use a clean profile (or
  let CCSM stage them); if you only need the CCSM web UI, any profile works.

## Quick Start

### 1. Install the host agent

Download the `ccsm-agent` binary for your architecture from the [releases page](https://github.com/luisgsluis/claude-code-session-manager/releases).

```bash
sudo cp ccsm-agent-linux-arm64 /usr/local/bin/ccsm-agent
sudo chmod +x /usr/local/bin/ccsm-agent
```

Generate a shared secret:

```bash
ccsm --generate-agent-secret
# or: openssl rand -base64 32
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
# KillMode=process (default would be control-group): the agent launches
# `tmux new-session -d` for every Claude session; those sessions live in the
# user's tmux server, independent of the agent, but they're forked inside its
# cgroup. Without this, `systemctl restart ccsm-agent` (e.g. after a redeploy)
# kills everything in that cgroup — your live tmux/claude sessions included.
KillMode=process
Restart=on-failure
RestartSec=3
# The agent is polled every few seconds per open session (tmux capture-pane,
# a process-tree scan, a tail read of the conversation file). Go's runtime
# doesn't return freed heap to the OS promptly without GOMEMLIMIT, so under
# sustained polling RSS climbs for hours before leveling off — harmless on a
# workstation, but enough to push a memory-constrained host (e.g. a
# Raspberry Pi) into swap. GOMEMLIMIT makes the runtime give memory back
# proactively; MemoryMax is a hard backstop (systemd kills and
# Restart=on-failure brings it back) if it's ever exceeded anyway. Tune both
# to the host's headroom — these values fit a ~4GB Pi.
Environment=GOMEMLIMIT=256MiB
MemoryMax=400M

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
# Same reason as ccsm-agent.service above: in package mode `ccsm` itself runs
# tmux in-process, so a plain restart would otherwise kill live sessions too.
KillMode=process
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
| `trusted_proxies` | — | — | CIDRs of reverse proxies whose `X-Forwarded-For` is believed. Required behind a proxy, or every client looks like the proxy — see [Security Model](#security-model) |
| `users[].username` | — | — | Login username |
| `users[].password_hash` | — | — | bcrypt hash (`ccsm --hash-password`) |
| `users[].totp_secret` | — | — | base32 TOTP secret; empty/absent = no 2FA for that user. Written by the enrollment flow, not by hand |
| `audit_path` | `CCSM_AUDIT_PATH` | `auto` | JSONL audit log; `auto` → `$HOME/.ccsm/audit.jsonl` |
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
| `rc.settle_seconds` | `CCSM_RC_SETTLE_SECONDS` | `1` | Confirmation margin after a resumed session is idle+bridged before restoring the target profile |

Some of these are **hot-reloadable** from the UI's ☰ menu (`lan_subnets`,
`host_attach_addr`, and the `rc.*` block) — see [API](#patch-apiconfig-semantics)
below. The rest (`port`, `agent_socket`, `agent_secret`, `session_secret`, paths,
and the `claude`/`tmux`/`bash` binaries) require a service restart.

### CLI Tools

```bash
# Generate a random session secret
ccsm --generate-secret

# Generate a random agent shared secret (container mode)
ccsm --generate-agent-secret

# Hash a password for config.yaml
ccsm --hash-password "my-password"
```

## API

All endpoints live under `/api` and are protected by authentication — a session
cookie or the LAN-subnet bypass — except `GET /api/health`, which is public and
used by the container healthcheck.

**Sessions**
- `GET /api/sessions` — list sessions
- `GET /api/projects` — launch targets for a new session ("principal" = home, plus any dir
  under home with a `CLAUDE.md`)
- `POST /api/sessions/new` — start a new session (optionally with a profile or a project)
- `POST /api/sessions/resume` — resume a past conversation
- `DELETE /api/sessions/{name}` — kill a session
- `POST /api/sessions/{name}/rename` — rename the tmux session; body `{"new_name": "..."}`
- `POST /api/sessions/{name}/claude-name` — set the Claude conversation title; body `{"title": "..."}`. Implemented by typing `/rename <title>` + Enter into the pane, so the title may include punctuation (including `!`).
- `GET /api/sessions/{name}/chat` — recent conversation turns plus live status (mode, model, waiting/approval state)
- `GET /api/sessions/{name}/chat/stream` — SSE stream of chat updates
- `GET /api/sessions/{name}/stream` — SSE stream of the raw terminal pane (what the Terminal
  tab and the terminal grid render); `?color=1` preserves ANSI colour instead of the default
  plain text
- `POST /api/sessions/{name}/send` — send a chat message (`{"text": "..."}`), a special key
  (`{"keys": "..."}`, e.g. `enter`/`escape`/`ctrl-o`), or a mode switch (`{"mode": "..."}`,
  driven by cycling Claude Code's real Shift+Tab wheel)
- `POST /api/sessions/{name}/rc` — force a fresh Remote Control bridge re-registration

**Profiles & conversations**
- `GET /api/profiles` — list profiles
- `GET /api/profiles/{name}` — profile content viewer
- `POST /api/profiles/apply` — apply a profile
- `GET /api/conversations` — list conversations
- `GET /api/conversations/{id}` — get one conversation

**Config & users**
- `GET /api/config` — non-secret deployment info for the ☰ menu
- `PATCH /api/config` — edit hot-reloadable settings, persisted atomically
- `GET /api/config/users` — list users as `[{"username": "...", "totp": bool}]` (never the hash or the TOTP secret)
- `POST /api/config/users` — add a user
- `DELETE /api/config/users/{username}` — delete a user
- `POST /api/config/users/{username}/password` — change a password
- `POST /api/config/users/{username}/totp` — start 2FA enrollment: returns `{"secret", "uri"}` **once**, held in memory and not yet persisted
- `PUT /api/config/users/{username}/totp` — confirm enrollment with `{"code": "123456"}`; only now is the secret written to `config.yaml`
- `DELETE /api/config/users/{username}/totp` — disable 2FA for that user
- `GET /api/settings` — content of the CURRENTLY APPLIED `settings.json` (the active profile; may differ from the profile files)

**Auth**
- `POST /api/auth/login`, `POST /api/auth/logout`, `GET /api/auth/status`
- `POST /api/auth/totp` — second step of a 2FA login, `{"code": "123456"}`. Public like
  `/login`: the short-lived cookie issued by the first step is what identifies the caller.
- When the user has 2FA enrolled, `POST /api/auth/login` answers `{"ok": false,
  "totp_required": true}` and sets a **pending** cookie that opens nothing — every protected
  route returns `401 {"error": "totp required"}` until `/api/auth/totp` succeeds.
  `GET /api/auth/status` reports the same state, so a page reload resumes at the code step.
- Failed attempts are rate-limited **per client IP**, shared between the password and the
  TOTP step: 5 failures in 15 minutes → `429` for 15 minutes.

### PATCH /api/config semantics

- **Hot-reloads in memory, no restart**: `lan_subnets` (each CIDR-validated), `host_attach_addr`, and `rc.{bootstrap_profile, wait_seconds, poll_seconds}`. RC changes affect only NEW sessions.
- **Not accepted** by this endpoint (need a service restart): `port`, `agent_socket`, `agent_secret`, `session_secret`, paths, and the `claude`/`tmux`/`bash` binaries.
- The write is **atomic**: temp file + rename.
- Response: `{"ok": true, "updated": [...], "restart_needed": false}`; `400` with `{"error": "..."}` on an invalid value or when no field is valid.

### User management

- Passwords are hashed with **bcrypt (cost 10)**; the API returns usernames only — hashes and passwords never leave the server.
- Password minimum is **8 characters**, enforced on both client and server.
- You **cannot delete the last remaining user** (`400`).
- **2FA is opt-in per user.** Enroll from the ☰ menu: the panel shows a QR (and the secret in
  text, for manual entry) and asks for a code from the app. The secret is only written to
  `config.yaml` once that code verifies — a scan that silently failed can therefore never
  lock anyone out. It is returned exactly once, at enrollment, and never again.
- **There are no recovery codes.** Two ways back in if the phone is lost: log in from a
  `lan_subnets` address (the bypass never asks for a code), or delete that user's
  `totp_secret` line from `config.yaml` on the host and restart.

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
| **Two-factor (TOTP)** | Opt-in per user (RFC 6238, 6 digits, 30 s, ±1 step for clock drift). Verified **before** the session cookie is issued; a code cannot be replayed inside its own window. LAN bypass never reaches it |
| **Rate limiting** | 5 failed attempts per client IP in 15 min → `429` for 15 min, shared between the password and the TOTP step. The real IP (see `trusted_proxies`) is recorded in the audit log, so an external blocker can act on it |
| **User management** | Passwords min 8 chars (client + server); API exposes usernames only — hashes and TOTP secrets never leave the server; cannot delete the last user |
| **Session** | Encrypted cookie (HMAC-SHA256), `HttpOnly`, `SameSite=Lax`, 24h TTL |
| **Agent auth** | Shared secret sent in every request to ccsm-agent |
| **Agent execution** | Argument validation with closed regex patterns, no shell interpolation |
| **Config persistence** | Settings edits are written atomically (temp file + rename), never corrupting `config.yaml` |
| **Unix socket** | File permissions (`0600`) restrict access to socket owner only |

### HTTP Security Headers

```
X-Content-Type-Options: nosniff
X-Frame-Options: DENY
Referrer-Policy: no-referrer
Permissions-Policy: camera=(), microphone=(), geolocation=(), interest-cohort=()
Content-Security-Policy: default-src 'self'; style-src 'self' 'unsafe-inline'; script-src 'self' 'unsafe-eval'
```

`'unsafe-eval'` is required by Alpine.js, which compiles `x-data` expressions with
`new Function`. Everything else is `'self'`: no inline scripts (an inline `<script>` **will**
be blocked — that is how the skin-persistence script silently broke in 1.2.0) and no external
origins, which is why the QR encoder is vendored in `static/js/` rather than loaded from a CDN.

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
- Behind a reverse proxy: check `trusted_proxies`. Every proxied request arrives from the same
  peer, so without it the real client IP is never read and both the LAN bypass and the rate
  limiter see one single client
- `429 too many failed attempts`: the per-IP limiter tripped. It clears itself after 15 minutes,
  or immediately on a restart (the counters are in memory)

### The 2FA code is always rejected
- **Check the server's clock.** TOTP is a function of time; CCSM accepts ±1 step (±30 s) and
  nothing more, so a drifting host rejects every code from a correct app. `timedatectl` /
  `chronyc tracking` on the host.
- Codes cannot be reused: entering the same one twice within its 30-second window fails the
  second time by design.
- Locked out for good? Delete that user's `totp_secret` from `config.yaml` and restart, or log
  in from a `lan_subnets` address, which never asks for a code.

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

The e2e suite starts **two** servers: `run-e2e.sh` (:8799) with `127.0.0.0/8` in
`lan_subnets`, so every spec runs under the LAN bypass and never sees the login form; and
`run-e2e-auth.sh` (:8798) with the bypass off and a 2FA user, which is what `auth.spec.js`
drives. Both use the stub `tmux`/`claude` in `e2e/stubs/`.

Adding a UI skin: [`docs/skins.md`](docs/skins.md).

## Looking for Contributors

CCSM is a side project I maintain solo, and I'm looking for people to bring into it as
contributors — not a queue for feature requests. If you want to actually work on something
(pick an item from `ROADMAP.md`'s backlog, or propose your own), open an issue to introduce
yourself and what you'd want to tackle.

## License

MIT. See [LICENSE](LICENSE).
