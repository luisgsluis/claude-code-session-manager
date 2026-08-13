# Roadmap

Forward-looking only: what's next and what's deliberately deferred. Shipped work lives in
`CHANGELOG.md`, not here.

**Looking for contributors** to bring into the project — not requests, actual hands on
something. Pick anything from Backlog below, or open an issue to propose something else and
introduce yourself.

## Next

Nothing prioritized right now — see Backlog below.

## Backlog

Deliberately deferred; pick up if/when there's a concrete need.

- **Security: 2FA via TOTP** — the session-cookie architecture allows adding TOTP
  verification without structural changes. After validating user+password, a TOTP code
  would be required before issuing the session cookie; the auth middleware would check a
  `totp_verified` flag.
- **Mobile app**: native or PWA mobile version.

## Release process

Tags matching `v*` trigger `.github/workflows/docker-publish.yml`: builds a multiarch image
(amd64, arm64, armv7) to GHCR and attaches the `ccsm`/`ccsm-agent` binaries to the GitHub
release. `.github/workflows/e2e.yml` runs on every push: `go vet` + `go build` + `go test
-race`, then the Playwright e2e suite against the host stubs (only once the Go tests pass).
