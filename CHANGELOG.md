# Changelog

## [1.1.1] — 2026-08-13

### Fixed
- e2e: the terminal-grid spec located tiles via `.tgrid > div`, which matched
  a `.tgrid-row` (possibly holding several tiles) after the row-packing
  layout change, not a single tile — CI caught this as a strict-mode
  violation. Each tile now carries its own `.tgrid-tile` class for the spec
  to target. No runtime behaviour change.

## [1.1.0] — 2026-08-13

### Added
- **Terminal mode** ("Modo terminal" in the UI): full-screen grid tiling every active
  session's raw terminal pane at once, with minimize/restore, tmux-style zoom, per-tile
  interactive input, and rendered ANSI colour (16/256/truecolor). Backend surface: an
  opt-in `color=1` query param on `GET /api/sessions/{name}/stream`, and a `ctrl-o` entry
  in the session-send key whitelist.
- Each grid tile now has its own compact mode/model selectors in the header row
  (no extra line), mirroring the single-session view's controls.
- Grid tiles strip Claude Code's own input box (the rule/❯/rule chrome) from
  the rendered pane: each tile already has its own input row below, so the
  native box was redundant.

### Changed
- Grid tile header row dropped the "open in single-session view" (👁️) button —
  the tile is already the interactive view, so it was a redundant hop.
- Reconnect status text shortened to "Reconnecting…" (was "Connection lost,
  reconnecting…").

### Fixed
- **Mode badge/switch reliability**: `paneMode` only checked the pane's single
  last non-blank line; on a long footer that wraps, the trailing `/rc` hint
  lands on its own line and pushed the real mode badge out of view, making
  both the mode switch and the chat view's persistent mode indicator
  unreliable. It now checks the last few non-blank lines. `sessionMode` also
  no longer sends `/plan` as a slash command under any circumstance (even
  when the badge is unreadable): it reaches every mode, including plan,
  purely by cycling the real Shift+Tab wheel, re-checking the badge after
  each press.

## [1.0.0] — 2026-08-12

First stable release.

### Added
- `ccsm --generate-agent-secret`: generates a base64 agent shared secret from the CLI,
  no external `openssl` dependency needed.

### Fixed
- **Live chat robustness**: the live chat/terminal `EventSource` no longer closes itself on
  a drop (which killed the browser's automatic retry) — it now reconnects on its own, with a
  status message that clears once the stream is back. Sending a chat message refreshes the
  view immediately instead of waiting for the next 1s poll.
- **Agent secret comparison was not constant-time**: `ccsm-agent`'s shared-secret check used
  `!=` instead of `crypto/subtle.ConstantTimeCompare`, contradicting the documented timing-attack
  mitigation.
- **Session store crash risk**: `Store.GetSession` deleted expired sessions while holding only
  a read lock; two concurrent requests hitting the same expired token could both call `delete()`
  on the map at once, which Go's runtime treats as a fatal error, not a panic — the whole process
  would crash.
- **Config mutation was unsynchronized**: `PATCH /api/config` and the user-management endpoints
  mutated shared config fields without a lock while other requests read them concurrently, a
  data race that could also let two simultaneous "add user" calls both pass the duplicate check
  and corrupt `config.yaml`. Guarded by a `sync.RWMutex`.
- **Internal error detail no longer leaks to API clients**: backend/config-write failures used
  to include the raw Go error (paths, socket errors) in the JSON response; they're now logged
  server-side and returned as a generic message.
- Malformed JSON bodies on session endpoints (`new`, `resume`, `rename`, `claude-name`, `send`)
  were silently ignored instead of returning `400`.
- Request bodies decoded as JSON are now capped at 1 MiB, closing an unbounded-memory-allocation
  path on every JSON-accepting endpoint.
- `Content-Disposition` on conversation export now escapes the filename instead of interpolating
  it raw into the header.
- Session cookie now sets `Secure` when the request reached CCSM over TLS (directly or via a
  reverse proxy's `X-Forwarded-Proto`).

## [0.1.6] — 2026-08-11

### Fixed
- **Enter did not send the message**: the Alpine 3.14.9 bundle does not implement the
  `.exact` modifier, which blocked the whole listener: Enter inserted a newline without
  sending and Shift+Enter (the expected newline) did nothing. Handling is now manual
  (`onChatKeydown`): Enter sends, Shift+Enter inserts a newline.
- **Mode change broken**: `/mode` does not exist in Claude Code 2.1.227 (only `/plan` and
  `/model`). The mode selector now sends structured `{mode}` and the host resolves it
  (`sessionMode`): `plan` via `/plan` (deterministic anchor), the rest via the Shift+Tab
  wheel (raw `\e[Z`), verified empirically: `manual → accept-edits → plan → auto`.
  `paneMode` normalizes the "accept edits" badge (with a space) to `accept-edits`.
- **Model with ANSI tags**: the pane output arrived with `\x1b[1m`/`\x1b[22m` codes.
  `sessionPane` strips them server-side (CSI + OSC); a literal `[1m]` in a model id
  (e.g. `deepseek-v4-pro[1m]`) is left alone.
- **Session blocked on an invisible dialog**: permission approval never reaches the
  transcript, so the chat looked like a silent session. The chat payload now carries
  `waiting` when the pane shows "Do you want to proceed?" and the footer shows
  "awaiting approval" with **Approve** (Enter, dialog option 1) and **Stop** (Escape)
  buttons.
- **Stop that did not work in dialogs**: the ⏹ button sent Ctrl-C. It now lives in the
  bottom bar, to the right of re-register, and sends Escape, which always works
  (interrupts generation and cancels dialogs/permissions).
- **Terminal/mobile messages invisible in project sessions**: the transcript of a session
  opened in another folder (e.g. inside a project) is stored in
  `~/.claude/projects/<cwd-slug>/<uuid>.jsonl`, not in `-home-admin`. All conversation
  access (live chat, browser, export, metrics, alive state and the id fallbacks) now
  searches every `projects/` subfolder (`convFileFor`/`convFiles`), so a message sent from
  the terminal or mobile in a project session shows up in the chat.
- **Inconsistent scroll arrows**: ↑ scrolled up a page in the chat but jumped to the top in
  the terminal. Both now scroll up a full page and ↓ goes to the very end, with double-line
  icons (⏶/⏬).
- **Session chats mixed**: the no-pinned-id fallback (`lifetimeConv`/`newestActiveConv`) picked
  the globally most-recent transcript across every project, so a session without `--session-id`
  in its argv could show another session's chat. The fallback now (1) restricts the search to the
  transcript folder matching the session's cwd (tmux `pane_current_path` → the `<cwd-slug>`
  subfolder), and (2) drops transcripts born before the session started (coreutils `stat %W`),
  so an older conversation that another running session keeps touching can no longer hijack the
  chat.
- **Mid-turn messages invisible in the chat**: a message typed while Claude is busy is recorded
  by Claude Code as a `queue-operation`, never as a `user` line, so the chat view and the mobile
  app never showed it. `chatRoleAndText` now surfaces the `enqueue` of those as a user message
  (filtering the `remove` drain and control commands like `/remote-control`) in the live chat,
  the conversations browser and the txt export.
- **Mid-turn messages duplicated after the fix above**: once a queued message drains, Claude
  Code keeps the `queue-operation` (enqueue) **and** writes the real `user` line
  (`promptSource: "queued"`) with the same text, so each mid-turn message rendered twice. The
  parsers now fold lines through `chatDedup`: an enqueue whose text later arrives as a real user
  line is dropped (the real line stands for it; enqueues that never drain — the message reaches
  the turn as a `queued_command` attachment instead — still show, they are its only trace), and
  assistant lines sharing a `message.id` (streaming snapshots / split content blocks) collapse
  into a single turn keeping the last text. Verified against the real transcripts: the 7 paired
  duplicates in a working session all resolve to single messages.
- **RC status read from the session file (Claude Code 2.1.228)**: 2.1.228 removed the `/rc`
  badge and the `/remote-control is active` footer flag from the status bar, so the pane no
  longer indicates the bridge state. `rcStatusLive` now reads `bridgeSessionId` from the
  per-process session file `~/.claude/sessions/<pid>.json` (the pane pid is the claude pid,
  since `newSession` runs claude directly under `bash -lc`), falling back to the badge/scrollback
  checks for older versions. The homelab watchdog `claude-rc-watch` uses the same file:
  it identifies Claude-with-RC sessions by `--remote-control` in the pane's argv and reconnects
  when `bridgeSessionId` is absent.
- **RC re-registration goes through staging on non-Anthropic profiles**: `/remote-control`
  only exists under the Anthropic endpoint — on a `perfilSinRC` session (deepseek, custom base
  URL) it returns "Unknown command". `sessionRc` (the `/rc` button) and `claude-rc-watch` now
  stage it like a fresh launch when the active profile disables RC: apply the bootstrap profile,
  send `/remote-control`, wait for the real bridge, restore the target profile.
- **Current model in the live status bar**: `session-chat` exposes `model` (the last assistant
  message's `message.model`) alongside `mode`, and the chat footer renders it ("model: …") in
  both languages.
- **File-edit approval dialog detected in the chat**: `paneWaitingReason` now also flags the
  edit-permission dialog ("Do you want to make this edit to <file>?", 2.1.228 wording) and the
  shared choice footer ("Esc to cancel · Tab to amend"), so a session blocked on an edit prompt
  shows the "waiting" banner and the Approve button in the chat view, not only in the terminal.
- **Choice dialogs (AskUserQuestion) detected and selectable in the chat**: `paneWaitingReason`
  distinguishes `choice` (option picker, footer "Enter to select · ↑/↓ to navigate") from
  `approval`. The chat shows ⚠ "choosing option" with ↑/↓ navigation and a Select (Enter) button
  instead of a plain Approve, so an interactive choice is answered from the chat, not only in the
  terminal.
- **Choice question and options shown in the chat**: `paneChoice` parses the AskUserQuestion
  dialog from the pane (the transcript does not record `input.question`), exposing
  `choice: {question, options, selected}` in the chat payload. The chat renders the question and
  each option (current one marked with ❯); ↑/↓ move the cursor, Enter or a click on an option
  selects it.

## [0.1.5] — 2026-08-11

### Fixed
- **Chat no longer empty on idle sessions**: the conversation-id fallback only looked at
  transcripts written in the last 2 minutes, so an idle session (last message >2 min ago,
  started before the deterministic mapping) showed an empty chat until the next message
  touched the file. New `lifetimeConv` fallback picks the most recently modified transcript
  written during this session's lifetime (`session_created` from tmux, −2 min slack),
  degrading to the freshness window when creation time is unavailable.
- **Terminal history**: `session-pane` now captures the full scrollback (`capture-pane -S -`).
  Note: Claude Code renders in the tmux alternate screen, which keeps no scrollback, so for
  these sessions the terminal shows the current screen only — the full history lives in the
  Chat tab.
- **Up/down scroll controls in the live view**: ↑ scrolls up half a screen, ↓ jumps to the
  end; auto-scroll on new content only follows when you're already at the bottom (terminal
  and chat). Ctrl-C button moved to a shared control bar for both tabs.
- **Mobile layout**: live modal height uses `min(80vh, 80dvh)` so it no longer overflows the
  visible viewport on mobile browsers.
- **Blank lines trimmed**: leading/trailing blank lines removed from each chat message.
- **Chat bubble borders**: the `whitespace-pre-wrap` bubble preserved the newlines/indentation
  of the HTML template between the bubble and the text span, rendering a blank line above and
  below every message. The span now sits flush inside the bubble, so only real content newlines
  show.
- **Model dropdown empty**: `x-for` on a bare `<option>` never renders inside `<select>` (Alpine
  leaves it as an unexpanded template). Wrapped the options in `<template x-for>` — the 5 models
  (opus/sonnet/haiku + the ANTHROPIC_DEFAULT_* values from the applied settings.json) now show.
- **Stale UI cache**: only `/static/*` was served with `Cache-Control: no-cache`; the SPA root
  (`index.html`) was heuristically cached by browsers, keeping old bubbles and dropdowns alive
  after releases. The root now sends `no-cache` too.
- **Uncaught error on fresh modal**: the status line read `live.meta.mode` in `x-text` even when
  `live.meta` was still null, throwing on open. Guarded.
- **Mode selector**: added `manual` to auto/plan/accept-edits; mode/model selectors relabeled to
  `Modo`/`Modelo` (es) and `Mode`/`Model` (en) — no "live" prefix.
- **Mobile zoom on input**: the browser auto-zoomed into the chat textarea (font 14px) every time
  it was tapped. Form fields (input/textarea/select) now force 16px on touch devices only
  (`@media (pointer: coarse)`), which iOS/Android treat as "no readability zoom needed"; desktop
  is untouched and pinch-zoom still works.
- **Config save from the UI**: `PATCH /api/config` failed with "device or resource busy" because
  config.yaml is bind-mounted into the container — temp+rename can't replace a mounted inode and
  the mount was `:ro`. `writeConfig` now falls back to an in-place truncate+write when rename
  fails with EBUSY, and the compose mount is `:rw`.
- **Sent messages invisible in long sessions**: `session-chat` returned the whole transcript
  (this session: 1279 messages, ~750 KB), re-emitted on every SSE frame while the session works —
  on mobile that flooded the connection and new messages never rendered. The live chat now caps
  to the most recent 200 messages (`maxChatMsgs`); the full history remains in the conversations
  browser.
- **Mode/model change gave no feedback**: the selectors do send `/mode x`/`/model x` into the
  session (same send path as a normal message), but the select snapped back to the placeholder
  and the footer never reflected it (`paneMode` only knew insert/edit/budget/normal). Changing
  now shows a success toast ("Modo cambiado a …", "Modelo cambiado a …") and the footer detects
  `plan`/`manual`/`auto`/`accept-edits` too.
- Version string reported by `/api/health` bumped to match the release (was stale at 0.1.3).

### Config
- `lan_subnets` on `rb.lan` includes the inbound WireGuard VPN range `192.168.9.0/24`
  (`vpn_in` interface), so VPN clients get the same LAN treatment as 192.168.1.0/24.

## [0.1.4] — 2026-08-11

### Added
- **Clean chat view for live sessions** (Nivel 2): the live modal now has a **Chat** tab
  alongside Terminal. It shows the FULL conversation history (every user/assistant turn, no
  truncation; assistant tool blocks skipped), a message input (Enter to send, Shift+Enter for
  a new line) and special-key buttons (⏹ Ctrl-C, ↑ Up). New endpoints:
  - `GET /api/sessions/{name}/chat` — one-shot full conversation + status metadata.
  - `GET /api/sessions/{name}/chat/stream` — SSE live updates (deduplicated by fingerprint).
  - `POST /api/sessions/{name}/send` — send a text message (`{"text": ...}`) or a whitelisted
    special key (`{"keys": "ctrl-c"}`) into a live session; text is capped at 2000 chars.
- **Status as window chrome**: the modal footer is a real UI status area (alive/off, RC
  status, mode when detectable, elapsed time), not terminal text.
- **Deterministic session↔conversation mapping**: fresh sessions launch `claude` with a
  pinned `--session-id <uuid>`; the id is recovered from the pane's process argv, with a
  fallback to the most recently written transcript for sessions started outside CCSM.

### Fixed
- Conversation-id lookup in argv now matches the uuid only when it directly follows
  `--session-id`/`--resume` (NUL-separated). Previously a bare uuid anywhere in a process's
  argv — e.g. tool-call shell text that merely mentions one — could be picked as the
  conversation id, mixing up sessions.
- tmux pane commands (`capture-pane`, `send-keys`, `list-panes`) now address a session by its
  pane id (`%N`) resolved from the full pane table matched on session **name**. Without a
  client context (`$TMUX` unset, as in the agent), `tmux -t 0` is parsed as window 0 of the
  most recent session, so a session literally named "0" was addressed as another session.

## [0.1.3] — 2026-08-11

### Added
- **Session rename**: `POST /api/sessions/{name}/rename` renames the tmux session, and
  `POST /api/sessions/{name}/claude-name` sets the Claude conversation title by typing
  `/rename <title>` into the pane; titles may include punctuation such as "!".
- **Editable settings from the ☰ menu**: `PATCH /api/config` edits hot-reloadable fields
  (`lan_subnets`, `host_attach_addr`, `rc.*`) without a restart, persisted atomically to
  config.yaml (temp file + rename).
- **User management from the UI**: add / delete users and change passwords via the
  `/api/config/users` endpoints; passwords require a minimum of 8 characters, are stored
  bcrypt-hashed, hashes never leave the server, and the last user cannot be deleted.
- **Active settings.json viewer**: `GET /api/settings` shows the currently applied
  settings.json (which may differ from the saved profiles), with syntax highlighting.
- **Profile viewer**: `GET /api/profiles/{name}` with syntax highlighting.
- Session name "0" is no longer special (reserved-name guard removed).
- **Conversations: Claude title and alive state**: the current Claude title
  (`ai-title`/`custom-title`) is shown in the list, cards, search results and detail view; an
  archived conversation whose session is still alive is no longer hidden from the list.
- **Advanced conversation search**: by text, by source machine (`origin`), by date range
  (`from`/`to`), and restricted to live sessions (`alive`).
- **Export conversation**: download a conversation as .jsonl or plain text.
- **Tags/notes on conversations**: organize conversations with tags and notes, persisted in a
  per-conversation sidecar file.
- **Audit log**: every action (session created/killed, profile change, login) is recorded in
  audit.jsonl and browsable in the UI.
- **Web push notifications**: desktop notifications when a session changes state (RC
  connected, task completed, session dead).
- **Metrics by model with filter and download**: `token_usage_by_model`
  (input/output/cache/messages per model), a `?model=` filter, and JSON/CSV export buttons in
  the metrics modal.

### Fixed
- Session rename failing with "can't find pane" from tmux send-keys: the `=` target
  prefix is only valid for session targets (rename-session/has-session/kill-session), not
  for pane targets like send-keys; claude-name now sends to the bare session name.
- **Live session view (SSE)**: the stream returned 500 "streaming unsupported" in production —
  the logging middleware wrapped the ResponseWriter in a type that did not implement
  `http.Flusher`. The wrapper now promotes `Flush()` and `Unwrap()` to the real writer.

### Tests
- **E2E in CI** (`e2e.yml`): Playwright smoke test (LAN auto-login → create session → close
  it) against the host stubs (`claude`/`tmux`).
- **Multiarch release workflow** (`docker-publish.yml`): image for amd64/arm64/armv7 pushed to
  GHCR on `v*` tags, with `ccsm` and `ccsm-agent` binaries attached to the release.

## [0.1.2] — 2026-08-10

### Added
- **Dual deployment modes**: package mode (single `ccsm` binary, no agent, commands run
  in-process via `internal/direct` + shared `internal/host`) and container mode (existing
  Unix-socket agent). Chosen by `agent_socket` (`""` = direct). Docs updated (README,
  architecture.md, config.example.yaml).
- **Top action bar** in the UI: "Nueva sesión", "Nueva con perfil ▾" (accent-outlined,
  with a "pick a profile for the new session" hint) and "Cambiar perfil ▾", placed above
  the session list.
- **i18n es/en** selectable per user via a 🌐 dropdown, persisted in localStorage; all UI
  strings (toasts, confirms, settings, login) go through `t(key, [vars])` with an es
  fallback for unknown locales.
- **Settings menu (☰)**: modal fed by the new auth-protected `GET /api/config` endpoint —
  shows deployment mode, port, paths, users, LAN subnets and RC values grouped with
  copy-to-clipboard; secrets (`session_secret`, `agent_secret`, `password_hash`) are never
  serialized or shown.
- **Docker healthcheck** (`wget` against `/api/health`, in the Dockerfile, inherited by
  compose) so orchestrators and the homelab updater see the container as healthy.
- **Agent singleton guard**: `ccsm-agent` refuses to start if a peer already answers on the
  socket and cleans up stale sockets left by crashes.
- **`ccsm-agent --secret-file`**: read the shared secret from a 0600 file instead of the
  command line, so it never appears in `ps aux` or the systemd unit.

### Fixed
- **RC status regression**: clean-profile sessions (RC bridge connected) were reported as
  `rc_pending` because `rcState` didn't pass `rc_connected`/`rc_failed` through. Sessions
  created with a clean profile now correctly show `rc_connected`.

### Tests
- Coverage raised: config 100%, agent 96%, server 96.5%; remaining gaps are defensive
  branches on impossible inputs (e.g. `json.Marshal` of a `Host` result). Full suite green
  (`go build`, `go vet`, `go test ./...`).
- Healthcheck verified end-to-end against a real container (`docker inspect` → `healthy`).

## [0.1.1] — 2026-08-10

### Fixed
- Remote Control bootstrap now actually used: profiles that disable RC (apiKeyHelper,
  env API key, non-Anthropic base URL) start with a clean bootstrap profile and hot-apply
  the target once the RC bridge is up (`lanzar_con_staging`, ported from olivetin-cmd).
- "New with profile" no longer passes an invalid `--profile` flag: RC-clean profiles start
  with `claude --settings <profile> --remote-control` (global settings.json untouched);
  RC-disabling ones go through the bootstrap.
- "Resume" now returns 404 for missing conversation IDs instead of starting a dead session.
- Session liveness (`is_alive`) detection: fixes sessions started without `--resume` being
  reported dead (mtime-based complement), and hides conversations whose session isn't alive.
- Dockerfile: `--config /config/config.yaml` now the entrypoint default so config is loaded
  when mounted at `/config` (was silently reading `config.yaml` from the container root).
- Profile application validates JSON before writing and never replaces a `settings.json`
  symlink (Syncthing-shared config stays a link).
- Server-side re-validation of session/profile/conversation names matches the agent's closed
  patterns; docs corrected (security.md, architecture.md) to match actual behavior.

### Added
- Unit tests for the agent (RC bootstrap, profile application, sin-RC detection).
- E2E coverage for invalid names (kill/new/resume/apply) and parse-error paths.
- Coverage: agent 95%, auth 98%, config 100%, handlers 99%, server 96%.

## [0.1.0] — 2026-08-10

### Added
- CCSM web server (Go, static binary) with a REST API
- Host agent (`ccsm-agent`) that runs tmux/claude over a Unix socket
- Authentication with a login form + LAN bypass
- tmux session management: list, create, kill
- Claude profile management: list, apply
- Conversation browser with text search and a dual view (list/cards)
- Responsive UI with Tailwind CSS + Alpine.js
- Multi-stage Dockerfile (Alpine, ~15 MB final)
- YAML configuration + environment variables
