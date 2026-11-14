MODULE  ?= github.com/stephen-mcelhose/markov
CMD     ?= ./cmd/markov
BIN_DIR ?= bin
BINARY  ?= markov
GO      ?= go
GOTEST  ?= $(GO) test
GOFLAGS ?=

.DEFAULT_GOAL := check

.PHONY: help check check-all build run test test-race cover vet fmt fmt-check lint tidy generate clean

help: ## List available targets
	@grep -E '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'

check: fmt-check vet test ## Fast pre-commit checks (default)

check-all: fmt-check vet lint test-race ## Full CI-style suite

build: ## Build cmd/markov into bin/
	@mkdir -p $(BIN_DIR)
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) -trimpath -o $(BIN_DIR)/$(BINARY) $(CMD)

run: ## Run cmd/markov
	$(GO) run $(GOFLAGS) $(CMD)

test: ## Run unit tests
	$(GOTEST) $(GOFLAGS) -count=1 ./...

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

clean: ## Remove build and coverage artifacts
	rm -rf $(BIN_DIR) coverage.out coverage.html
