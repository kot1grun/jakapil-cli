# Makefile for jakapil-cli
# GNU Make. Default target: build.

BINARY   := jakapil-cli
BIN_DIR  := bin
GO       ?= go
GOPROXY  ?= $(shell $(GO) env GOPROXY)

# Path to golangci-lint, if installed (used by the lint target below).
GOLANGCI_LINT := $(shell command -v golangci-lint 2>/dev/null)

.PHONY: all build clean test lint vet

# `all` is the default target when `make` is invoked without arguments.
all: build

## build: compile the CLI binary into $(BIN_DIR)/$(BINARY).
build:
	$(GO) build -o $(BIN_DIR)/$(BINARY) .

## clean: remove the build output directory.
clean:
	rm -rf $(BIN_DIR)

## test: run the full test suite.
test:
	$(GO) test ./...

## vet: run go vet (static analysis built into the Go toolchain).
vet:
	$(GO) vet ./...

## lint: run go vet, plus golangci-lint when it is installed.
lint: vet
	@if [ -n "$(GOLANGCI_LINT)" ]; then \
		$(GOLANGCI_LINT) run ./...; \
	else \
		echo "golangci-lint is not installed; running go vet only (make lint-check to require golangci-lint)"; \
	fi

## lint-check: fail unless golangci-lint is installed and passes.
lint-check:
	@if [ -z "$(GOLANGCI_LINT)" ]; then \
		echo "golangci-lint is required but was not found in PATH"; exit 1; \
	fi
	$(GOLANGCI_LINT) run ./...

# Print help for targets when running `make help`.
help:
	@echo "Available targets:"
	@echo "  all        build the $(BINARY) binary (default)"
	@echo "  build      compile into $(BIN_DIR)/$(BINARY)"
	@echo "  clean      remove $(BIN_DIR)"
	@echo "  test       run the test suite (go test ./...)"
	@echo "  lint       go vet (+ golangci-lint when installed)"
	@echo "  lint-check require golangci-lint and run it"
	@echo "  vet        run go vet"