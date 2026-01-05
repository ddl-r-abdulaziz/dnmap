# dnmap Makefile

# Go parameters
GO_FILES := $(shell find . -type f -name '*.go')
BINARY_NAME := dnmap
BUILD_DIR := bin

# Tool versions
GOLANGCI_LINT_VERSION ?= v1.61.0

.PHONY: all
all: clean lint test build ## Run clean, lint, test, and build

##@ Development

.PHONY: build
build: $(BUILD_DIR)/$(BINARY_NAME) ## Build the binary

$(BUILD_DIR)/$(BINARY_NAME): $(GO_FILES)
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/dnmap

.PHONY: run
run: build ## Build and run the CLI
	./$(BUILD_DIR)/$(BINARY_NAME)

.PHONY: clean
clean: ## Remove build artifacts
	rm -rf $(BUILD_DIR)
	rm -f coverage.out

##@ Testing

.PHONY: test
test: ## Run unit tests
	go test -v -race ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

##@ Code Quality

.PHONY: lint
lint: golangci-lint ## Run linters
	$(GOLANGCI_LINT) run ./...

.PHONY: fmt
fmt: ## Format Go source files
	go fmt ./...
	goimports -w .

.PHONY: vet
vet: ## Run go vet
	go vet ./...

##@ Dependencies

.PHONY: deps
deps: ## Download dependencies
	go mod download

.PHONY: tidy
tidy: ## Tidy go modules
	go mod tidy

.PHONY: verify
verify: tidy ## Verify dependencies
	git diff --exit-code go.mod go.sum

##@ Tools

GOLANGCI_LINT = $(shell pwd)/bin/golangci-lint
.PHONY: golangci-lint
golangci-lint: ## Download golangci-lint locally if necessary
	@if [ ! -f $(GOLANGCI_LINT) ]; then \
		echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(shell pwd)/bin $(GOLANGCI_LINT_VERSION); \
	fi

##@ Help

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help

