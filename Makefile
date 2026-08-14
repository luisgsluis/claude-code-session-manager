.PHONY: build agent deploy-agent run test lint clean docker-build docker-run

BINARY := bin/ccsm
AGENT  := bin/ccsm-agent
IMAGE  := ccsm

build:
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BINARY) ./cmd/ccsm

agent:
	mkdir -p bin
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(AGENT) ./cmd/ccsm-agent

# Reconstruye e instala ccsm-agent en el host y reinicia el servicio en un
# solo paso atómico (ver scripts/deploy-agent.sh). Solo tiene sentido en rb,
# donde vive la unit systemd.
deploy-agent:
	./scripts/deploy-agent.sh

run: build
	./$(BINARY) --config config.yaml

test:
	go test ./...

lint:
	go vet ./...

clean:
	rm -rf bin

docker-build:
	DOCKER_BUILDKIT=1 docker build -t $(IMAGE):latest .

docker-run:
	docker run --rm -it -p 8080:8080 -v ./config:/config $(IMAGE):latest
