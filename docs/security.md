# Security Model

## Principle

CCSM follows a defense-in-depth model. Compromising any single layer should not grant an attacker control over Claude Code sessions.

## Layers

### 1. Network

- CCSM listens on the container's internal port
- Publish only to `127.0.0.1` and reverse proxy with TLS (Caddy/Nginx)
- Never expose CCSM directly to the internet without TLS + auth

### 2. Authentication

- **Login form**: username + bcrypt-hashed password
- **Session cookie**: HMAC-SHA256(token, secret), HttpOnly, SameSite=Lax, 24h TTL
- **LAN bypass**: requests from configured CIDRs skip login. This relies on network security (your LAN is trusted)
- **2FA-ready**: the session architecture supports adding TOTP verification before cookie issuance — no structural changes needed
- **User management**: users are managed through the API (`GET`/`POST /api/config/users`, `DELETE /api/config/users/{username}`, `POST /api/config/users/{username}/password`). Passwords are hashed with bcrypt (`DefaultCost = 10`) and must be at least 8 characters — enforced client-side **and** server-side. The API returns usernames only: password hashes and passwords never leave the server, are never logged, and are never serialized. Deleting the last remaining user returns `400`, which prevents lockout
- **Config surface**: the settings/config endpoints (`PATCH /api/config`, `GET /api/settings`, and the user-management routes above) sit behind the same `auth` middleware as every other route — a valid login cookie or a LAN-bypass request

### 3. Agent Authentication

- **Shared secret**: a base64-encoded random string (32 bytes) passed in every request body
- **Constant-time comparison**: prevents timing attacks on the secret
- **Unix socket permissions** (`0600`): the socket is owner-only on the host, and the container mounts it as a volume — only the host owner and the container process (usually root) can connect
- **Secrets never served**: `PATCH /api/config` does not accept `session_secret` or `agent_secret` — both require a restart and are excluded by design — and the config endpoints never return them. `GET /api/config` exposes only non-secret deployment info; `GET /api/settings` returns the currently applied `settings.json` (a profile's settings) under the same auth

### 4. Command Validation

The agent validates every argument against closed regex patterns BEFORE execution:

| Argument type | Pattern | Blocks |
|---|---|---|
| Session name | `^[A-Za-z0-9_-]{1,32}$` | Shell metacharacters, path traversal |
| Conversation ID | `^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$` | Non-UUID strings, injections |
| Profile name | `^[a-z0-9][a-z0-9_-]{0,31}$` | `/`, `..`, absolute paths |
| Claude title | `^[\p{L}\p{N}\p{P} ]{1,80}$` (letters, numbers, punctuation incl. `!`, spaces) | Control characters, newlines, titles over 80 chars |
| Username | `^[a-z0-9][a-z0-9_-]{0,31}$` | `/`, `..`, absolute paths (user management) |

The same whitelists are enforced **again in the web server** before anything reaches the
agent, so a stray misconfigured agent cannot be the last line of defense.

tmux operations run via `exec.Command()` with direct argv — never a shell. `claude`
sessions are launched as `bash -lc 'claude …'` (the shell is what keeps the tmux pane
alive), but every interpolated value is validated against the closed patterns above **and**
single-quote-escaped (`'` → `'\''`), so the composed string cannot break out of its quoting.

### 5. Filesystem

- The agent writes to only one file: `settings.json` (profile application)
- Writes use `os.WriteFile(..., 0600)`. If `settings.json` is a symlink (Syncthing-shared config), the write goes through to the target — the link itself is never replaced
- Every profile is validated as JSON **before** the write, so a broken profile can never leave `settings.json` unparseable (Claude Code would then fail to start)
- The write is not atomic (direct truncate + write); the JSON pre-validation is what guarantees the file is never left half-written
- **Config write-back is atomic**: `PATCH /api/config` and the user-management endpoints persist `config.yaml` (server-side) via a temp file + `rename`, so a crash can never leave a half-written config. This is separate from the profile write above, which stays non-atomic (truncate + write) and relies on JSON pre-validation

### 6. Host permissions required

Whatever process runs the commands — `ccsm-agent` (container mode) or the `ccsm` binary
itself (package mode) — needs exactly this access on the host. Least-privilege checklist:

| Path / resource | Access | Why |
|---|---|---|
| `/run/ccsm/` directory | `rwx` for the service user (created with `RuntimeDirectory=ccsm`, so `0755` root → adjust to `0770`) | the agent binds its socket here |
| `/run/ccsm/agent.sock` | `0600` (socket created by the agent itself) | only the owner may connect |
| `~/.claude/projects/<hash>/` (conversations) | `r` on files, `rx` on dirs | list + read `.jsonl` for the browser |
| `claude-shared/claude-perfiles/` | `r` | list and validate profiles |
| `~/.claude/settings.json` | `rw` | apply a profile (the only file the agent writes) |
| `tmux` binary + tmux server socket | execute + connect as the service user | list/create/kill sessions, read pane status for RC |
| `claude` binary | execute as the service user | launch sessions |

The agent runs **as your user** (`User=admin` in the systemd unit), not root. It should
never be given root: it validates every argument against closed regexes, but the tmux
session it spawns is a real Claude process with real credentials — run it as the user
that owns the Claude config.

In package mode the same permissions apply to the user running `ccsm`.

## Threats Not Mitigated (current)

| Threat | Risk | Mitigation in future version |
|--------|------|------------------------------|
| Brute force on login | Low (LAN-only) | Rate limiting (v0.3.0) |
| CSRF on login form | Low (no state-changing GETs, SameSite cookies) | CSRF tokens (v0.3.0) |
| Container escape | Medium (Docker isolation) | Read-only rootfs, non-root user, seccomp profile |
| Agent secret leak via env | Low (container only) | Secrets file with restricted permissions |
| Session hijacking via cookie theft | Medium (if host compromised) | Token binding to IP / user-agent (v0.3.0) |

## API Key Handling

⚠️ **CRITICAL**: Never put API keys in Claude Code profiles or `settings.json`.

Claude Code supports `apiKeyHelper` — a path to an executable that prints the API key to stdout. This is the ONLY safe way to configure API keys with CCSM:

```json
{
    "apiKeyHelper": "$HOME/.local/bin/claude-apikey"
}
```

The helper script (mode `700`, never synced, never committed):

```sh
#!/bin/sh
printf '%s' 'sk-your-api-key'
```

Reasons:
1. Profiles are files — they get synced (Syncthing), backed up, committed by accident
2. CCSM reads profile contents to detect the active profile — it must never see API keys
3. `apiKeyHelper` runs at Claude Code startup, the key never hits disk in plaintext in sync paths

CCSM does not read, store, or transmit API keys. Profile management only copies settings files — the security of those files is your responsibility.
