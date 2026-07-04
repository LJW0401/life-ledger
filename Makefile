# Project build commands. Go itself cannot be configured to change the default
# output path for a bare `go build`, so this file defines the repository build contract.

BIN_DIR := bin
BIN := $(BIN_DIR)/life-ledger

.PHONY: build frontend backend init-local-config clean

build: frontend backend

frontend:
	npm run build

backend:
	mkdir -p $(BIN_DIR)
	go build -o $(BIN) ./cmd/server

init-local-config: build
	$(BIN) init-config --config $(BIN_DIR)/config.toml --cookie-secure=false

clean:
	rm -rf $(BIN_DIR)
