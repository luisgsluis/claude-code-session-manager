# Stage 1: build CCSM
FROM golang:1.24-alpine AS builder
RUN apk add --no-cache git ca-certificates
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /ccsm ./cmd/ccsm

# Stage 2: runtime
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /ccsm /usr/local/bin/ccsm
COPY static/ /app/static/
EXPOSE 8080
# The compose file mounts config at /config/config.yaml; pass it explicitly so
# the binary (which defaults to "config.yaml" in CWD) loads the mounted file.
ENTRYPOINT ["/usr/local/bin/ccsm", "--config", "/config/config.yaml"]

# Healthcheck (busybox wget, ya presente en Alpine): /api/health no requiere auth
# ni agente, es la señal de que el servidor HTTP responde. El update del homelab
# verifica este estado y marca el contenedor como unhealthy si no pasa.
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/api/health >/dev/null 2>&1 || exit 1
