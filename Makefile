# harness — developer Makefile.
# Run `make` (or `make help`) to list targets.

BINARY      := harness
INSTALL_DIR ?= /usr/local/bin
DIST        := dist

# Locate the Go toolchain even when it is not on PATH.
GO          ?= $(shell command -v go 2>/dev/null || echo /usr/local/go/bin/go)
GOFMT       ?= $(shell command -v gofmt 2>/dev/null || echo /usr/local/go/bin/gofmt)

# Optional, stricter tooling installed via `make tools` (see go-code-standards rule).
GOBIN         := $(shell $(GO) env GOPATH)/bin
GOFUMPT       ?= $(shell command -v gofumpt 2>/dev/null)
GOLANGCI_LINT ?= $(shell command -v golangci-lint 2>/dev/null)
# Prefer gofumpt (a strict gofmt superset) when present, else fall back to gofmt.
FMT           := $(if $(GOFUMPT),$(GOFUMPT),$(GOFMT))

# Stamp the binary with the git version (tag/commit), falling back to "dev".
VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS     := -s -w -X main.version=$(VERSION)

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help
	@echo "harness — make targets:"
	@awk 'BEGIN{FS=":.*##"} /^[a-zA-Z0-9_-]+:.*##/ {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build: ## Compile the binary into ./dist
	@mkdir -p $(DIST)
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(DIST)/$(BINARY) .
	@echo "built $(DIST)/$(BINARY) ($(VERSION))"

.PHONY: run
run: ## Run the selection TUI from source
	$(GO) run .

.PHONY: install
install: build ## Build from source and install to $(INSTALL_DIR) (sudo if needed)
	@if [ -d "$(INSTALL_DIR)" ] && [ -w "$(INSTALL_DIR)" ]; then \
		install -m 0755 $(DIST)/$(BINARY) $(INSTALL_DIR)/$(BINARY); \
	else \
		echo "Elevated permissions required to write to $(INSTALL_DIR) (using sudo)."; \
		sudo install -d -m 0755 $(INSTALL_DIR); \
		sudo install -m 0755 $(DIST)/$(BINARY) $(INSTALL_DIR)/$(BINARY); \
	fi
	@echo "installed $(INSTALL_DIR)/$(BINARY) ($(VERSION))"

.PHONY: uninstall
uninstall: ## Remove the installed binary
	INSTALL_DIR=$(INSTALL_DIR) ./uninstall.sh

.PHONY: test
test: ## Run all tests
	$(GO) test ./...

.PHONY: fmt
fmt: ## Format the code in place (gofumpt if installed, else gofmt)
	$(FMT) -w .

.PHONY: vet
vet: ## Run go vet
	$(GO) vet ./...

.PHONY: lint
lint: ## Run golangci-lint (install with `make tools`)
	@if [ -z "$(GOLANGCI_LINT)" ]; then \
		echo "golangci-lint not found — run 'make tools'"; exit 1; \
	fi
	$(GOLANGCI_LINT) run ./...

.PHONY: tools
tools: ## Install dev tools (gofumpt, golangci-lint) into $(GOBIN)
	$(GO) install mvdan.cc/gofumpt@latest
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $(GOBIN)
	@echo "installed gofumpt and golangci-lint into $(GOBIN)"

.PHONY: tidy
tidy: ## Tidy go.mod / go.sum
	$(GO) mod tidy

.PHONY: check
check: ## CI gate: format check + vet + lint + test
	@unformatted="$$($(FMT) -l .)"; \
	if [ -n "$$unformatted" ]; then \
		echo "format needed for:"; echo "$$unformatted"; exit 1; \
	fi
	$(GO) vet ./...
	@if [ -n "$(GOLANGCI_LINT)" ]; then \
		$(GOLANGCI_LINT) run ./...; \
	else \
		echo "warning: golangci-lint not installed (run 'make tools') — skipping lint"; \
	fi
	$(GO) test ./...

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(DIST)
