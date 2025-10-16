# Makefile for jumphost development workflow
#
# Common targets:
#   make fmt          Run go fmt on all packages
#   make vet          Run go vet
#   make lint         Run golangci-lint (auto-installs pinned version locally into ./bin)
#   make test         Run tests with race detector and coverage
#   make coverage     Show coverage summary (enforces minimum)
#   make build        Build binary into ./bin/
#   make run          Run the application (go run)
#   make install      Install built binary into GOPATH/bin by default (e.g. ~/go/bin)
#   make uninstall    Remove previously installed binary from install dir
#   make tidy         Ensure go.mod/go.sum tidy
#   make mocks        Generate mocks using mockery
#   make ci           Run fmt + vet + lint + test
#   make clean        Remove build artifacts

SHELL := /bin/bash

PROJECT        := jumphost
MODULE         := github.com/n2jsoft/jumphost
BIN_DIR        := bin
DIST_DIR       := dist
GOLANGCI_LINT_VERSION ?= 2.5.0
GOLANGCI_LINT := $(BIN_DIR)/golangci-lint-$(GOLANGCI_LINT_VERSION)
MOCKERY_VERSION ?= 3.5.5
MOCKERY       := $(BIN_DIR)/mockery-$(MOCKERY_VERSION)
GO             ?= go
GOOS           := $(shell $(GO) env GOOS)
PKGS           := $(shell $(GO) list ./...)
GIT_VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
VERSION        ?= $(GIT_VERSION)
COMMIT         := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE           := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
# Inject variables defined in cmd/root.go
LDFLAGS        := -s -w -X $(MODULE)/cmd.version=$(VERSION) -X $(MODULE)/cmd.commit=$(COMMIT) -X $(MODULE)/cmd.date=$(DATE)
COVER_PROFILE  := coverage.out
COVER_MODE     := atomic
COVERAGE_MIN   ?= 60
GOPATH_FULL    := $(shell $(GO) env GOPATH)
GOPATH_FIRST   := $(firstword $(subst :, ,$(GOPATH_FULL)))
PREFIX         ?= $(GOPATH_FIRST)
INSTALL_BIN_DIR := $(PREFIX)/bin
BINARY_NAME    := $(PROJECT)$(if $(filter windows,$(GOOS)),.exe,)

.PHONY: all help fmt vet lint test coverage build run install uninstall tidy mocks ci clean deps release snapshot version

all: build

help:
	@grep -E '^#|^[a-zA-Z_-]+:' Makefile | sed -e 's/:.*//' -e 's/^# //' | awk 'BEGIN{print "Available targets:"} /^[^#]/ {print "  " $$0}'

$(BIN_DIR):
	@mkdir -p $(BIN_DIR)

$(GOLANGCI_LINT): | $(BIN_DIR)
	@echo "Installing golangci-lint v$(GOLANGCI_LINT_VERSION)...";
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(BIN_DIR) v$(GOLANGCI_LINT_VERSION) && mv $(BIN_DIR)/golangci-lint $(GOLANGCI_LINT)
	@$(GOLANGCI_LINT) version

$(MOCKERY): | $(BIN_DIR)
	@echo "Installing mockery v$(MOCKERY_VERSION)...";
	@GOBIN=$(shell pwd)/$(BIN_DIR) $(GO) install github.com/vektra/mockery/v3@v$(MOCKERY_VERSION) && mv $(BIN_DIR)/mockery $(MOCKERY)
	@$(MOCKERY) version

fmt:
	@echo "Running go fmt...";
	@$(GO) fmt ./...

vet:
	@echo "Running go vet...";
	@$(GO) vet ./...

lint: $(GOLANGCI_LINT)
	@echo "Running golangci-lint...";
	@$(GOLANGCI_LINT) run --timeout=5m

# Run tests with race + coverage
test: tidy
	@echo "Running tests...";
	@$(GO) test -race -coverprofile=$(COVER_PROFILE) -covermode=$(COVER_MODE) ./...

coverage: test
	@echo "Coverage summary:";
	@$(GO) tool cover -func=$(COVER_PROFILE) | tail -n 1
	@PCT=$$( $(GO) tool cover -func=$(COVER_PROFILE) | grep total: | awk '{print $$3}' | tr -d '%' ); \
	if awk -v c="$$PCT" -v m="$(COVERAGE_MIN)" 'BEGIN{exit (c+0 >= m+0)?0:1}'; then \
	  echo "Coverage OK ($$PCT% >= $(COVERAGE_MIN)%)"; \
	else \
	  echo "Coverage below threshold: $$PCT% < $(COVERAGE_MIN)%" >&2; exit 1; fi

# Build binary
build: tidy | $(BIN_DIR)
	@echo "Building $(PROJECT) for $(GOOS)...";
	@CGO_ENABLED=0 $(GO) build -trimpath -ldflags='$(LDFLAGS)' -o $(BIN_DIR)/$(BINARY_NAME) .
	@echo "Built $(BIN_DIR)/$(BINARY_NAME)"

run: tidy
	@echo "Running $(PROJECT) (go run) ...";
	@$(GO) run -ldflags='$(LDFLAGS)' .

# Install binary into GOPATH/bin (default) or PREFIX/bin
install: build
	@echo "Installing $(BINARY_NAME) to $(INSTALL_BIN_DIR)...";
	@mkdir -p $(INSTALL_BIN_DIR)
ifeq ($(GOOS),windows)
	@cp $(BIN_DIR)/$(BINARY_NAME) $(INSTALL_BIN_DIR)/$(BINARY_NAME)
else
	@install -m 0755 $(BIN_DIR)/$(BINARY_NAME) $(INSTALL_BIN_DIR)/$(BINARY_NAME)
endif
	@echo "Installed $(INSTALL_BIN_DIR)/$(BINARY_NAME)"

# Remove installed binary
uninstall:
	@echo "Uninstalling $(BINARY_NAME) from $(INSTALL_BIN_DIR)...";
	@if [ -f "$(INSTALL_BIN_DIR)/$(BINARY_NAME)" ]; then rm -f "$(INSTALL_BIN_DIR)/$(BINARY_NAME)" && echo "Removed"; else echo "Not found"; fi

# GoReleaser snapshot (no publish)
snapshot: tidy
	@command -v goreleaser >/dev/null || (echo "goreleaser not installed" >&2; exit 1)
	@goreleaser release --snapshot --skip=publish --skip=announce --clean

# GoReleaser full release (expects a tag & GITHUB_TOKEN)
release: tidy
	@command -v goreleaser >/dev/null || (echo "goreleaser not installed" >&2; exit 1)
	@goreleaser release --clean

# Ensure go.mod/sum are tidy
tidy:
	@$(GO) mod tidy
	@git diff --quiet go.mod go.sum || (echo 'go.mod/go.sum not tidy (run make tidy and commit changes)' >&2)

# Generate mocks using mockery
mocks: $(MOCKERY)
	@echo "Generating mocks...";
	@$(MOCKERY)

# Aggregate dev workflow
ci: fmt vet lint test

clean:
	@rm -rf $(BIN_DIR) $(DIST_DIR) $(COVER_PROFILE)
	@echo "Cleaned build artifacts."

deps:
	@$(GO) mod download

version:
	@echo "Version:    $(VERSION)"
	@echo "Commit:     $(COMMIT)"
	@echo "Date:       $(DATE)"
	@echo "Module:     $(MODULE)"
	@echo "Binary:     $(BINARY_NAME)"
	@echo "Go version: $(shell $(GO) version)"


# End of Makefile
