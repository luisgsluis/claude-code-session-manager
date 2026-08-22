# Stage 1: build CCSM
FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# VERSION is the release this image reports at /api/health — passed by whoever
# builds the image (docker-publish.yml passes the pushed tag; `make
# docker-build` passes the local `git describe`). .dockerignore excludes .git
# from the build context, so it can't be derived in here.
ARG VERSION=dev
RUN CGO_ENABLED=0 go build -ldflags="-s -w -X github.com/luisgsluis/claude-code-session-manager/internal/server.Version=${VERSION}" -o /ccsm ./cmd/ccsm

# Stage 2: runtime
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /ccsm /usr/local/bin/ccsm
COPY static/ /app/static/
EXPOSE 8080
# The compose file mounts config at /config/config.yaml; pass it explicitly so
# the binary (which defaults to "config.yaml" in CWD) loads the mounted file.
ENTRYPOINT ["/usr/local/bin/ccsm", "--config", "/config/config.yaml"]

# Healthcheck (busybox wget, already present on Alpine): /api/health requires
# no auth and no agent — it's the signal that the HTTP server is responding.
# Orchestrators poll this to mark the container unhealthy if it fails.
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/api/health >/dev/null 2>&1 || exit 1
