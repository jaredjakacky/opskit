SHELL := /bin/sh

GO ?= go
GOFMT ?= gofmt
PKGS ?= ./...
GOFILES := $(filter-out $(shell git ls-files --deleted -- '*.go'),$(shell git ls-files -- '*.go'))
EXAMPLE_GOFILES := $(shell find examples -name '*.go' -print 2>/dev/null)
GOVULNCHECK_VERSION ?= v1.6.0
RELEASE_CHECK_DIR := tools/releasecheck

# Keep build cache inside the repo so local runs are reproducible and do not
# depend on a writable global cache path.
export GOCACHE ?= $(CURDIR)/.cache/go-build
export GOMODCACHE ?= $(CURDIR)/.cache/go-mod

.DEFAULT_GOAL := help

.PHONY: \
	help \
	build-examples \
	dependency-boundary \
	fmt \
	fmt-check \
	vet \
	test \
	test-race \
	coverage \
	tidy \
	tidy-check \
	govulncheck \
	verify \
	clean

help: ## Show available targets.
	@printf "Available targets:\n"
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_.-]+:.*##/ {printf "  %-14s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build-examples: ## Compile the runnable example programs.
	@echo "==> build examples"
	@if [ -n "$(EXAMPLE_GOFILES)" ]; then \
		$(GO) build ./examples/...; \
	else \
		echo "no example packages"; \
	fi

dependency-boundary: ## Ensure the root module has no external module dependencies.
	@echo "==> checking dependency boundary"
	@deps="$$(GOWORK=off $(GO) list -mod=readonly -m -f '{{if not .Main}}{{.Path}}{{end}}' all)" || exit $$?; \
	if [ -n "$$deps" ]; then \
		echo "Opskit's root module must not depend on external modules:"; \
		echo "$$deps"; \
		exit 1; \
	fi

fmt: ## Format tracked Go source files.
	@echo "==> formatting"
	@$(GOFMT) -w $(GOFILES)

fmt-check: ## Verify tracked Go source files are formatted.
	@echo "==> checking formatting"
	@out="$$($(GOFMT) -l $(GOFILES))"; \
	if [ -n "$$out" ]; then \
		echo "The following files are not formatted:"; \
		echo "$$out"; \
		exit 1; \
	fi

vet: ## Run go vet on all verified modules.
	@echo "==> vet"
	@$(GO) vet $(PKGS)
	@GOWORK=off $(GO) -C $(RELEASE_CHECK_DIR) vet ./...

test: ## Run tests for all verified modules.
	@echo "==> test"
	@$(GO) test $(PKGS)
	@GOWORK=off $(GO) -C $(RELEASE_CHECK_DIR) test ./...

test-race: ## Run tests for all verified modules with the race detector enabled.
	@echo "==> test (race)"
	@$(GO) test -race $(PKGS)
	@GOWORK=off $(GO) -C $(RELEASE_CHECK_DIR) test -race ./...

coverage: ## Run tests with coverage output written to coverage.out.
	@echo "==> coverage"
	@$(GO) test -coverprofile=coverage.out $(PKGS)

tidy: ## Synchronize go.mod and go.sum with the source tree.
	@echo "==> tidy"
	@GOWORK=off $(GO) mod tidy
	@GOWORK=off $(GO) -C $(RELEASE_CHECK_DIR) mod tidy

tidy-check: ## Verify go.mod/go.sum are already tidy.
	@echo "==> checking tidy"
	@GOWORK=off $(GO) mod tidy -diff
	@GOWORK=off $(GO) -C $(RELEASE_CHECK_DIR) mod tidy -diff

govulncheck: ## Run the pinned govulncheck tool against all verified modules.
	@echo "==> govulncheck"
	@$(GO) run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) $(PKGS)
	@GOWORK=off $(GO) -C $(RELEASE_CHECK_DIR) run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

verify: fmt-check dependency-boundary vet test build-examples tidy-check ## Run the local verification suite.
	@echo "==> verification passed"

clean: ## Remove local build outputs and caches.
	@echo "==> clean"
	@rm -rf .cache coverage.out .bin
