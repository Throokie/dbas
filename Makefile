VERSION := $(shell git describe --tags --always 2>/dev/null || echo dev)
GOEXE := $(shell go env GOEXE)
BINARY := dbas$(GOEXE)
ROOT := ./dbas

.PHONY: build test vet run demo clean lint coverage docker-run

build:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BINARY) $(ROOT)/cmd/dbas

build-mcp-example:
	CGO_ENABLED=0 go build -ldflags="-s -w" -o mcp-example$(GOEXE) $(ROOT)/examples/mcp-server

test:
	go test $(ROOT)/... -count=1 -timeout 120s -v

test-short:
	go test $(ROOT)/... -count=1 -timeout 120s

vet:
	go vet $(ROOT)/...

run:
	./$(BINARY) -config dbas.toml

demo:
	./$(BINARY) --demo --port 8080

lint:
	go vet $(ROOT)/...

clean:
	rm -f $(BINARY)
	rm -f mcp-example$(GOEXE)
	rm -rf dbas/dbas-data/

coverage:
	go test $(ROOT)/... -count=1 -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

docker-build:
	docker build -t dbas:latest -f Dockerfile .

docker-run:
	docker run -p 8080:8080 -v $(PWD)/dbas.toml:/app/dbas.toml dbas:latest

# Quick cross-compilation for release
release: build
	@mkdir -p dist
	@for p in darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64; do \
		os=$${p%/*}; arch=$${p#*/}; ext=; [ $$os = windows ] && ext=.exe; \
		echo "  building $$os/$$arch"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -ldflags="-s -w" -o dist/dbas-$$os-$$arch$$ext $(ROOT)/cmd/dbas; \
	done
	@echo "Release binaries in dist/"

.PHONY: all
all: clean vet test build
	@echo "All checks passed."
