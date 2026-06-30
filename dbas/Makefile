.PHONY: all build test clean run demo docker help

BINARY ?= dbas
GO ?= go
GOFLAGS ?= -ldflags="-s -w"
OUTDIR ?= build

all: build

build:
	mkdir -p $(OUTDIR)
	$(GO) build $(GOFLAGS) -o $(OUTDIR)/$(BINARY) ./dbas/cmd/dbas

install:
	$(GO) install ./dbas/cmd/dbas

test:
	$(GO) test ./dbas/... -v -count=1 -timeout 120s

test-short:
	$(GO) test ./dbas/... -count=1 -timeout 60s

test-race:
	$(GO) test -race ./dbas/... -count=1 -timeout 300s

coverage:
	$(GO) test -coverprofile=coverage.out -covermode=atomic ./dbas/...
	$(GO) tool cover -html=coverage.out -o coverage.html

vet:
	$(GO) vet ./dbas/...

lint:
	golangci-lint run ./dbas/... --timeout=5m

fmt:
	$(GO) fmt ./dbas/...

clean:
	rm -rf $(OUTDIR) coverage.out coverage.html dbas-data

run: build
	$(OUTDIR)/$(BINARY)

run-demo: build
	$(OUTDIR)/$(BINARY) --demo --port 8080

docker:
	docker build -t dbas:latest -f dbas/Dockerfile .

docker-run: docker
	docker run -p 8080:8080 dbas:latest --demo

help:
	@echo 'DBAS Makefile'
	@echo ''
	@echo '  build        Build the DBAS binary'
	@echo '  install      Install to GOPATH/bin'
	@echo '  test         Run all tests'
	@echo '  test-race    Run tests with race detection'
	@echo '  coverage     Generate coverage report'
	@echo '  run          Build and run DBAS'
	@echo '  run-demo     Build and run in demo mode'
	@echo '  docker       Build Docker image'
	@echo '  docker-run   Run in demo mode via Docker'
	@echo '  clean        Remove build artifacts'
	@echo '  lint         Run golangci-lint'
