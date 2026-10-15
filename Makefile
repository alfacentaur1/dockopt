BINARY  := dockopt
CMD     := ./cmd/$(BINARY)
BIN_DIR := bin
PKGS    := ./...

GOLANGCI_LINT ?= golangci-lint

.PHONY: all build test lint clean

all: lint test build

build:
	go build -o $(BIN_DIR)/$(BINARY) $(CMD)

test:
	go test -race -cover $(PKGS)

lint:
	go vet $(PKGS)
	$(GOLANGCI_LINT) run $(PKGS)

clean:
	rm -rf $(BIN_DIR)
