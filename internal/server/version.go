package server

// Version is the CCSM release version, reported by /api/health and the
// no-static fallback page. Not bumped by hand: it's overridden at build time
// via -ldflags "-X .../internal/server.Version=vX.Y.Z" — `make build`/`make
// agent` inject the local `git describe --tags`, and docker-publish.yml
// injects the pushed tag (see VERSION build-arg in Dockerfile). "dev" is
// what's left when a binary is built neither way (e.g. `go build` directly).
var Version = "dev"
