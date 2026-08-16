# Contributing to CCSM

Thanks for wanting to help. CCSM is a Go web app that manages Claude Code
sessions on a host with `tmux`. The layout and architecture are in `docs/`
— if you change behavior, update the relevant doc.

## Getting started

Prerequisites: Go 1.24+ and a host with `tmux` and `claude` for actually
running sessions.

```bash
make build        # bin/ccsm (web server)
make agent        # bin/ccsm-agent (host agent for container mode)
make test         # go test ./...
make lint         # go vet ./...
```

Dev run against a `config.yaml` (copy `config.example.yaml`):

```bash
make run
./ccsm --generate-secret
./ccsm --hash-password "yourpassword"
```

Docker: `make docker-build`, `make docker-run`.

## Where things live

- `internal/` — Go. `host/` is the single source of command logic;
  `direct/` (package mode), `agent/` (container mode over Unix socket),
  `server/`, `auth/`, `config/`, `handlers/`.
- `static/` — HTML/CSS/JS (Alpine.js + Tailwind).
- `docs/` — architecture and security notes.

## Conventions

- Go: idiomatic, `gofmt`-clean, flat packages under `internal/`.
- Code and docs in English; the UI is i18n es/en, add new strings to both.
- API errors are JSON `{"error": "..."}`, shown as a red toast in the UI.
- Commits: conventional commits (`feat:`, `fix:`, `docs:`, `chore:`), one
  logical change per commit.

## Testing

`make test` runs unit + integration tests. There's also a Playwright e2e
suite for the UI in `e2e/`:

```bash
cd e2e
npm ci
npx playwright install --with-deps chromium
npx playwright test
```

CI runs both on every push (see `.github/workflows/e2e.yml`).

## Issues and discussions

- Bugs → use the bug report template.
- Feature ideas → the feature request template.
- Questions → GitHub Discussions.
- Security problems → report privately via **Security → Report a
  vulnerability** (see [SECURITY.md](SECURITY.md)). Please don't open a public
  issue.
