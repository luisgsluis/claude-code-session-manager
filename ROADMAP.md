# Roadmap

Prioritized list of features. The "Backlog" holds features deliberately deferred.

## Next

- **Live chat robustness**: reconnect the `EventSource` when it drops (today it shows
  "closed" and stays dead) and refresh the chat right after a send, without waiting for the
  1s poll.
- **Configurable mode wheel**: `modeWheel` in `host.go` is the wheel order; if an account
  enables `bypassPermissions` or lacks `auto`, the order changes. Make it configurable.

## Shipped (0.1.6)

- Real mode selector (`/mode` does not exist in Claude Code 2.1.227): `/plan` as the anchor +
  the Shift+Tab wheel verified empirically (`manual → accept-edits → plan → auto`).
- Permission-approval dialog detection (`waiting` in the chat payload) with **Approve**
  (Enter) and **Stop** (Escape, bottom bar) buttons.
- Project-subfolder conversations (`convFileFor`/`convFiles`), which makes terminal/mobile
  messages visible in sessions opened in another folder.
- Correct Enter/Shift+Enter (Alpine `.exact` gotcha), ANSI stripping of the pane, consistent
  scroll arrows (↑ page, ↓ end).

## Backlog

- **Security: 2FA via TOTP**: the session-cookie architecture allows adding TOTP verification
  without structural changes. After validating user+password, a TOTP code is required before
  issuing the session cookie; the auth middleware checks a `totp_verified` flag.
- **Mobile app**: native or PWA mobile version
- **Full terminal view**

## Release/CI

The current release batch (conversations with title/alive state, live session view via SSE,
metrics by model with filter and download, audit log, web push notifications, settings
editing, user management) is shipped. Automation lives in `.github/workflows/`:

- `e2e.yml` — Playwright e2e in CI (smoke test against the host stubs).
- `docker-publish.yml` — multiarch image (amd64, arm64, armv7) to GHCR on `v*` tags, plus
  `ccsm` and `ccsm-agent` binaries attached to the release.
