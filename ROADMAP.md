# Roadmap

Prioritized list of features. The "Backlog" holds features deliberately deferred.

## Next

- **Configurable mode wheel**: `modeWheel` in `host.go` is the wheel order; if an account
  enables `bypassPermissions` or lacks `auto`, the order changes. Make it configurable.

## Shipped (unreleased)

- **Modo terminal**: a full-screen tiled view of every active session's
  raw terminal pane at once (top action bar, next to the profile switcher). Each tile
  reuses the existing `/stream` SSE and `/send` mechanisms — no new backend commands other
  than an opt-in `color=1` query param on `/stream` (keeps `-e` + skips the server-side ANSI
  strip only when asked) and one `ctrl-o` entry in the key whitelist, so a tile's "fold
  output" button just sends Claude Code's own collapse/expand keystroke to the pane. Tiles
  can be minimized (pulled into a compact rail so the rest grow to fill the space), zoomed
  fullscreen (tmux `Ctrl+b z` style — the other tiles stay mounted and streaming, hidden
  only via CSS, so returning is instant), and are interactive (a per-tile terminal-style
  prompt, plus the existing Approve/Stop/choice-picker controls). Renders real ANSI colour,
  including 256-colour and truecolor SGR sequences. This closes the "Full terminal view"
  backlog item.

## Shipped (1.0.0)

- **Live chat robustness**: the `EventSource` now lets the browser's native retry reconnect
  it when the connection drops, instead of closing it and staying dead; a status message
  surfaces the drop and clears on reconnect. Sending a chat message refreshes the view right
  away instead of waiting for the next 1s poll.

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

## Release/CI

The current release batch (conversations with title/alive state, live session view via SSE,
metrics by model with filter and download, audit log, web push notifications, settings
editing, user management) is shipped. Automation lives in `.github/workflows/`:

- `e2e.yml` — `go vet` + `go build` + `go test -race` (unit + integration), then Playwright
  e2e against the host stubs; e2e only runs once the Go tests pass.
- `docker-publish.yml` — multiarch image (amd64, arm64, armv7) to GHCR on `v*` tags, plus
  `ccsm` and `ccsm-agent` binaries attached to the release.
