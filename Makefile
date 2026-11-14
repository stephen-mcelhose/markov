MODULE  ?= github.com/stephen-mcelhose/markov
GO      ?= go
GOTEST  ?= $(GO) test
GOFLAGS ?=

.DEFAULT_GOAL := check

.PHONY: help check check-all test test-race spec spec-all surface surface-ast surface-emit docs-gen cover vet fmt fmt-check lint tidy generate clean

help: ## List available targets
	@grep -E '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'

check: fmt-check vet surface surface-ast test ## Fast pre-commit checks (default)

check-all: fmt-check vet surface surface-ast lint test-race spec ## Full CI-style suite

test: ## Run unit tests
	$(GOTEST) $(GOFLAGS) -count=1 ./...

spec: ## Run tests and display as RFC 2119 spec (specable sub-tests only)
	$(GOTEST) $(GOFLAGS) -v -count=1 ./... 2>&1 | $(GO) run ./tools/testspec

spec-all: ## Run tests and display all sub-tests (specable + non-specable)
	$(GOTEST) $(GOFLAGS) -v -count=1 ./... 2>&1 | $(GO) run ./tools/testspec --all

surface: ## Validate JSON-LD surface graph against SHACL shapes and architecture.md
	$(GO) run ./tools/surface validate

surface-ast: ## Validate implemented Go packages against surface.jsonld and AGENTS.md policies
	$(GO) run ./tools/surface ast

surface-emit: ## Regenerate docs/surface/surface.jsonld from docs/architecture.md and validate
	$(GO) run ./tools/surface emit

docs-gen: ## Render docs/architecture.md from docs/surface/surface.jsonld and validate
	$(GO) run ./tools/surface render

test-race: ## Run tests with the race detector
	$(GOTEST) $(GOFLAGS) -race -count=1 ./...

cover: ## Write coverage.out and coverage.html
	$(GOTEST) $(GOFLAGS) -covermode=atomic -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

vet: ## Run go vet
	$(GO) vet $(GOFLAGS) ./...

fmt: ## Format all Go sources with gofmt
	@gofmt -w $$(find . -name '*.go' -not -path './.git/*')

fmt-check: ## Fail if any Go file needs gofmt
	@out=$$(gofmt -l $$(find . -name '*.go' -not -path './.git/*')); \
	if [ -n "$$out" ]; then echo "$$out"; exit 1; fi

lint: ## Run golangci-lint (requires local install)
	golangci-lint run ./...

tidy: ## Tidy go.mod / go.sum
	$(GO) mod tidy

generate: ## Run go generate
	$(GO) generate $(GOFLAGS) ./...

clean: ## Remove coverage artifacts
	rm -f coverage.out coverage.html
