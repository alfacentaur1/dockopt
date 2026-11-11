# Variables. Use them as $(NAME); change a value here and every target picks it up.
# (Comments go on their own line: anything after a value, including the spaces before #, becomes part of the value.)

# Name of the compiled program.
BINARY  := dockopt
# Package containing main().
CMD     := ./cmd/$(BINARY)
# Where build output goes.
BIN_DIR := bin
# All packages in the module.
PKGS    := ./...
# Version baked into the binary; defaults to the latest tag or commit.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
# Where the web demo lives.
WEB_DIR := web

# ?= sets a default that can be overridden: make lint GOLANGCI_LINT=/path/to/golangci-lint
GOLANGCI_LINT ?= golangci-lint

# These targets are task names, not files, so make always runs them.
.PHONY: all build install test lint web serve clean

# Default target (plain `make`): lint, test, then build.
all: lint test build

# Compile the CLI into bin/dockopt.
build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) $(CMD)

# Install the CLI into $GOPATH/bin.
install:
	go install -trimpath -ldflags "$(LDFLAGS)" $(CMD)

# Run all tests with the race detector and print coverage.
test:
	go test -race -cover $(PKGS)

# Static checks: go vet (built in) + golangci-lint (needs to be installed).
lint:
	go vet $(PKGS)
	$(GOLANGCI_LINT) run $(PKGS)

# Compile the core to WebAssembly for the web demo and copy Go's JS glue next to it.
web:
	GOOS=js GOARCH=wasm go build -trimpath -ldflags "-s -w" -o $(WEB_DIR)/dockopt.wasm ./cmd/wasm
	cp "$$(go env GOROOT)/lib/wasm/wasm_exec.js" $(WEB_DIR)/

# Serve the web demo on http://localhost:8080 (wasm needs http, file:// won't do).
serve: web
	python3 -m http.server --directory $(WEB_DIR) 8080

# Remove build output.
clean:
	rm -rf $(BIN_DIR) $(WEB_DIR)/dockopt.wasm $(WEB_DIR)/wasm_exec.js
