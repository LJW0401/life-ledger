# Project build commands. GitHub Actions call these targets so local and CI
# gates stay aligned.

BIN_DIR := bin
BIN_NAME := life-ledger
BIN ?= $(BIN_DIR)/$(BIN_NAME)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo 0.1.0-dev)
LDFLAGS := -s -w

.PHONY: verify ci fmt-check check-workflows vet test test-race check-release-notes web-install web-typecheck web-lint web-build frontend go-build backend build e2e-smoke build-cross init-local-config clean

verify: fmt-check check-workflows web-build vet test web-typecheck web-lint go-build
	@echo "verify ok"

ci: fmt-check check-workflows web-build vet test-race web-typecheck web-lint e2e-smoke go-build
	@echo "ci ok"

fmt-check:
	@test -z "$$(gofmt -l ./cmd ./internal ./web/*.go)" || (echo "gofmt diff detected; run gofmt -w ./cmd ./internal ./web/*.go" && gofmt -d ./cmd ./internal ./web/*.go && exit 1)

check-workflows:
	sh scripts/check-workflows.sh

vet:
	go vet ./...

test:
	go test ./...

test-race:
	go test -race ./...

check-release-notes:
	go run ./scripts/check-release-notes -file release.md -tag "$(VERSION)"

web-install:
	npm ci --prefix web --no-audit --no-fund

web-typecheck: web-install
	npm run typecheck

web-lint: web-install
	npm run lint

web-build: web-install
	npm run build

frontend: web-build

go-build:
	mkdir -p $(dir $(BIN))
	@test -f web/dist/index.html || (echo "web/dist missing — run make web-build" && exit 1)
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/server

backend: go-build

build: web-build go-build

e2e-smoke: web-install
	CI=1 npm run test:e2e

build-cross: web-build
	mkdir -p $(BIN_DIR)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(MAKE) go-build BIN=$(BIN_DIR)/$(BIN_NAME)-linux-amd64
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 $(MAKE) go-build BIN=$(BIN_DIR)/$(BIN_NAME)-linux-arm64

init-local-config: build
	$(BIN) init-config --config $(BIN_DIR)/config.toml --cookie-secure=false

clean:
	rm -rf $(BIN_DIR) release-dist web/dist/assets web/dist/index.html
