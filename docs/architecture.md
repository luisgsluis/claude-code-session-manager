# Architecture

## Communication Flow

```
Browser ──HTTP──► CCSM Container ──Unix Socket──► ccsm-agent (host) ──exec──► tmux / claude
```

### Why a host agent?

The CCSM container runs Alpine Linux (musl libc). Host binaries (`tmux`, `claude`, `bash`) are compiled against glibc. These ABIs are incompatible — mounting host binaries into the container doesn't work.

The agent (`ccsm-agent`) is a minimal Go binary that runs directly on the host. It listens on a Unix socket and executes commands on behalf of the container. This architecture:

- Avoids SSH overhead for local IPC
- Keeps the container image small (~15 MB Alpine)
- Is portable: the agent is distributed alongside the container image
- Has two security layers: Unix socket permissions + shared secret authentication

## Deployment Modes

The web server can run in two modes, selected by `agent_socket` in `config.yaml`:

| Mode | `agent_socket` | How commands run |
|---|---|---|
| **Container** | `/run/ccsm/agent.sock` | Server talks HTTP-over-Unix-socket to `ccsm-agent` (host daemon) |
| **Package (direct)** | `""` (empty) | Server runs `tmux`/`claude` in-process, no agent |

Both share the same command logic (`internal/host`), which returns typed errors with an
HTTP status. The two execution backends mirror each other:

- **`internal/host`** — single source of truth for every command (`Host.Exec(cmd, args)`).
  Returns `*host.Error{Status, Msg}` where Status is 400 (invalid input), 404 (missing
  resource) or 500 (execution failure).
- **`internal/direct`** — package mode. Wraps `host.Host` in-process; host errors flow
  straight through.
- **`cmd/ccsm-agent`** — container mode. Exposes the same `Host.Exec` over a Unix socket
  with a shared secret; maps `host.Error` to the matching HTTP status.
- **`internal/agent`** — HTTP client over the socket. Maps non-2xx responses to
  `*agent.ClientError{Status, Msg}`.

The web handlers accept either backend through the `handlers.Agent` interface and map any
of these errors to the HTTP status it carries — callers see a meaningful code and a clean
message instead of a blanket 502.

### Agent singleton control

`ccsm-agent` refuses to run twice. At startup it dials the socket: if a peer answers it
exits with `ErrAlreadyRunning` (another agent holds the socket); if the dial fails it
removes the stale socket (left by a crash) and takes over. This guarantees a single agent
deployment per host even if systemd restarts race.

### Request Lifecycle

1. User clicks "New session" in the browser
2. Browser sends `POST /api/sessions/new` to CCSM
3. Auth middleware checks cookie or LAN bypass
4. Handler serializes the request and sends it to the executor (direct in-process, or agent over the Unix socket)
5. Agent (if any) validates the shared secret
6. The executor validates arguments against closed regex patterns
7. It executes `tmux new-session -d ... "claude --remote-control"`
8. It returns session name + status (or a typed error → HTTP 400/404/500)
9. Handler enriches the response with attach command
10. Browser shows success toast and refreshes session list

## API Design

REST JSON API. All endpoints live under `/api`; every one is auth-protected
(cookie or LAN bypass) except `health`, `login`, `logout` and `status`.

### Endpoints

```
GET  /api/health                             public; used by the Docker healthcheck
POST /api/auth/login                         login (password, or LAN bypass)
POST /api/auth/logout
GET  /api/auth/status
GET  /api/sessions                           list tmux sessions (each may carry `project`)
POST /api/sessions/new                       start a Claude session; body may include
                                             {"project":"<name from /api/projects>"} to
                                             launch with that dir as cwd (its CLAUDE.md
                                             applies) and tag the session
GET  /api/projects                           launch targets: "principal" (home) + dirs
                                             under home with a CLAUDE.md (≤3 levels)
POST /api/sessions/resume                    resume a conversation (UUID)
DELETE /api/sessions/{name}                  kill a tmux session
POST /api/sessions/{name}/rename             rename the tmux session; body {"new_name":"..."}
POST /api/sessions/{name}/claude-name        set the Claude conversation title; body {"title":"..."}
GET  /api/profiles                           list profiles
GET  /api/profiles/{name}                    profile content viewer
POST /api/profiles/apply                     apply a profile to settings.json
GET  /api/conversations                      search conversations
GET  /api/conversations/{id}                 single-conversation preview
GET  /api/config                             non-secret deployment info (☰ menu)
PATCH /api/config                            hot-reload settings; atomic write to config.yaml
GET  /api/config/users                       list usernames
POST /api/config/users                       add a user; body {"username","password"}
DELETE /api/config/users/{username}          delete a user
POST /api/config/users/{username}/password   change password; body {"password"}
GET  /api/settings                           content of the CURRENTLY APPLIED settings.json
```

`claude-name` is implemented by `tmux send-keys` of the literal `/rename <title>` +
Enter into the pane. `GET /api/settings` reflects the **active profile** — it may
differ from the saved profile files, so it is separate from `GET /api/profiles/{name}`.

### Rename gotcha

The `=` target prefix (`-t =name`) is valid only for tmux **session** targets —
`rename-session`, `has-session` and `kill-session` all use it. It is NOT valid for
**pane** targets: `tmux send-keys -t =name` fails with "can't find pane". It is also
NOT valid for `set-option`'s target: `tmux set-option -t =name` looks for a session
literally named "=name" (visto 2026-08 al etiquetar sesiones con `@ccsm_project`) —
use the bare session name there. So the tmux-session rename targets the session with
`=name`, the Claude-title rename (`/claude-name`) sends keys to the pane with the bare
name, and project tagging (`set-option`) uses the bare name too.

### Deployment info for the UI

`GET /api/config` (protected) feeds the settings ☰ menu. It returns the **non-secret**
subset of the running config: mode (`direct`/`agent`), port, agent socket, host attach
address, LAN subnets, usernames, the resolved paths and the RC bootstrap values.

This endpoint only reports the deployment subset. The rest of the ☰ panel is backed
by the other config endpoints above: `GET /api/settings` (active settings.json),
`GET`/`POST`/`DELETE /api/config/users` (user management) and `PATCH /api/config`
(hot-reload).

```json
{
  "port": 8080,
  "mode": "agent",
  "agent_socket": "/run/ccsm/agent.sock",
  "host_attach_addr": "admin@192.168.1.175",
  "lan_subnets": ["192.168.1.0/24"],
  "users": ["admin"],
  "paths": { "conversations": "…", "profiles": "…", "settings": "…", "claude_binary": "…", "tmux_binary": "…", "bash_binary": "…" },
  "rc": { "bootstrap_profile": "estandar", "wait_seconds": 25, "poll_seconds": 2 }
}
```

Secret material is deliberately excluded: `session_secret`, `agent_secret` and
`password_hash` are never serialized. The UI shows "secrets never exposed" and renders
copy-to-clipboard buttons for the visible values.

### Hot-reload config (PATCH /api/config)

`PATCH /api/config` updates a subset of settings **in memory** (no restart) and
persists them atomically to `config.yaml` via a temp file + rename.

Hot-reloadable:
- `lan_subnets` — each value CIDR-validated
- `host_attach_addr`
- `rc.{bootstrap_profile, wait_seconds, poll_seconds}` — affects only NEW sessions

Not accepted (require a restart):
- `port`, `agent_socket`, `agent_secret`, `session_secret`, `paths`, and the
  claude/tmux/bash binaries

Success returns `{"ok": true, "updated": [...], "restart_needed": false}`. An invalid
value returns `400 {"error": "..."}` and nothing is written.

### User management

Users live in `config.yaml` as bcrypt hashes (cost 10). The API only ever returns
usernames — hashes never leave the server. Passwords must be at least 8 characters.
`POST /api/config/users` creates a user, `DELETE /api/config/users/{username}`
removes one (the last user cannot be deleted), and
`POST /api/config/users/{username}/password` changes a password.

### Validation

Every argument is checked against a closed regex, first in `internal/host` (the
single source of truth) and again in the web handlers, so invalid input is rejected
with a clean 400 even if the agent were bypassed:

| Field | Pattern |
|---|---|
| Session name | `^[A-Za-z0-9_-]{1,32}$` |
| Profile name | `^[a-z0-9][a-z0-9_-]{0,31}$` |
| Conversation ID | UUID |
| Claude title | `^[\p{L}\p{N}\p{P} ]{1,80}$` (letters, numbers, punctuation incl. `!`, spaces) |
| Username | `^[a-z0-9][a-z0-9_-]{0,31}$` |
| `host_attach_addr` | `^[a-zA-Z0-9@._-]{1,120}$` |

### Session States

```json
{
    "name": "3",
    "created": "",
    "task": "analizando logs del NAS",
    "attach_cmd": "ssh admin@rb.lan -t tmux a -t 3",
    "status": "rc_pending"
}
```

Status values (detected from the per-process session file, with the pane as fallback):
- `rc_connected` — Remote Control bridge registered: `bridgeSessionId` present in `~/.claude/sessions/<pid>.json` (the pane's pid IS the claude pid, since `newSession` runs claude directly under `bash -lc`, which execs into it). Older versions fall back to the pane's status bar/scrollback (`/rc` or `/remote-control is active`).
- `rc_failed` — RC bridge failed (older versions: `/rc failed` in the status bar; no equivalent signal exists in 2.1.228+)
- `rc_pending` — no bridge registered yet: booting, RC disabled by the profile, or RC unavailable

> **2.1.228 removed the status-bar RC flag.** The `/rc` badge and the persistent
> `/remote-control is active` footer hint no longer appear in the TUI, so the pane is not a
> reliable indicator on current versions. The session file's `bridgeSessionId` is the source of
> truth — it is written exactly when the mobile-app bridge registers. The badge/scrollback
> checks remain only as a fallback for older Claude Code versions.

The name `0` carries no special meaning — the old reserved-name guard for session 0
was removed.

### Conversation Search

`GET /api/conversations?q=bug&origin=pi&page=1&per_page=20`

The search is full-text across the first human message of each conversation. The agent reads each `.jsonl` file, extracts the first `type=="user"` message that isn't `isMeta`, and matches against the query.

`GET /api/conversations/{id}` returns a preview of a single conversation — the last
N messages (default 50, up to 200 with `?lines=`).

Origin is inferred from the `cwd` of the first message:
- `/home/admin*` → `pi`
- `/home/luis*` → `pc`
- anything else → `?`

Only files with bare UUID filenames are returned (Syncthing `.sync-conflict-*.jsonl` files are filtered out).

## Configuration

### Layered config

```
config.yaml (file)  →  env vars (override)  →  defaults (fallback)
```

Every config key has a corresponding `CCSM_` env var. Nested keys use underscores: `rc.bootstrap_profile` → `CCSM_RC_BOOTSTRAP_PROFILE`.

### Agent configuration

The agent accepts its own flags (paths to profiles, settings, conversations). These can differ between deployments but typically map to the same Claude directories that CCSM references.

## Remote Control Bootstrap

Claude Code ≥ 2.1.195 disables Remote Control at startup when `settings.json` contains:
- `apiKeyHelper`
- `ANTHROPIC_API_KEY` (in `env`)
- `ANTHROPIC_BASE_URL` pointing to a non-Anthropic host

The same values as **shell environment variables** do NOT block RC. CCSM exploits this with a two-phase bootstrap.

The agent applies it automatically (`perfil_sin_rc`) whenever a session would start with an RC-disabling profile: for "new session" / "resume" it checks the profile currently applied to `settings.json`; for "new with profile" it checks the requested one. RC-clean explicit profiles skip the bootstrap and start with `claude --settings <profile.json> --remote-control`, leaving the global `settings.json` untouched.

The two phases:

1. **Phase 1**: Temporarily apply a clean profile (`rc_bootstrap_profile`, e.g., `estandar`) — no `apiKeyHelper`, no custom URL → RC enables at startup
2. **Phase 2**: Start the session with `--remote-control`
3. **Phase 3**: Poll for the bridge via `rcStatusLive` (up to `rc.wait_seconds`), requiring 2 consecutive "ok" polls
4. **Phase 4**: Hot-apply the target profile — RC survives the switch, and subsequent API calls go to the target provider

**Staging is the general rule, not just at launch.** The `/remote-control` command only exists
under the Anthropic endpoint; on a `perfilSinRC` session it returns "Unknown command" and
registers nothing. Any later RC action that re-issues `/remote-control` on such a session must
repeat the same staging — apply the bootstrap profile → send the command → wait for the real
bridge → restore the target. This is what `sessionRc` (the `/rc` endpoint) and the homelab
`claude-rc-watch` do. The endpoint hot-reloads in a live session, so switching `settings.json`
takes effect on the running process.

### Requirements

- The bootstrap profile MUST exist in the profiles directory
- The bootstrap profile MUST have `remoteControlAtStartup: true`
- The bootstrap profile MUST NOT have `apiKeyHelper` or non-Anthropic URLs
- The target profile can have any configuration RC-compatible or not

### Polling mechanism

`rcStatusLive` checks, in order:
1. The per-process session file `~/.claude/sessions/<pid>.json` for `bridgeSessionId` — present → "ok" poll (authoritative since 2.1.228)
2. The tmux pane's status bar (last non-empty line) via `capture-pane`: `/rc` → "ok", `/rc failed` → "fail" (older versions)
3. Scrollback for `/remote-control is active` (older versions)

2 consecutive "ok" polls → confirmed → proceed to phase 4.

## Security Boundaries

```
Internet ──┬──► Caddy/Nginx (TLS) ──► CCSM:8080
           │                            │
           │                       [auth middleware]
           │                            │
           │                       [handler logic]
           │                            │
LAN ───────┘                       [agent client]
                                        │
                                   Unix socket (0600)
                                        │
                                   [ccsm-agent]
                                        │
                                   [regex validation]
                                        │
                                   tmux / claude / files
```

Each arrow is a security boundary with its own enforcement.
