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

# ?= sets a default that can be overridden: make lint GOLANGCI_LINT=/path/to/golangci-lint
GOLANGCI_LINT ?= golangci-lint

# These targets are task names, not files, so make always runs them.
.PHONY: all build test lint clean

# Default target (plain `make`): lint, test, then build.
all: lint test build

# Compile the CLI into bin/dockopt.
build:
	go build -o $(BIN_DIR)/$(BINARY) $(CMD)

# Run all tests with the race detector and print coverage.
test:
	go test -race -cover $(PKGS)

# Static checks: go vet (built in) + golangci-lint (needs to be installed).
lint:
	go vet $(PKGS)
	$(GOLANGCI_LINT) run $(PKGS)

# Remove build output.
clean:
	rm -rf $(BIN_DIR)
