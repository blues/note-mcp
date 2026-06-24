# Makefile for note-mcp project

.PHONY: all clean build test blues-expert inspect-blues-expert help docs docs-blues-expert fmt vet develop

# Go build flags
GO=go
GOFLAGS=-v
LDFLAGS=-w -s

# Project directories
BLUES_EXPERT_DIR=./blues-expert

# Default target
all: vet build

# Help target
help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Build targets
build: blues-expert ## Build all MCP servers
	@echo "✓ Build complete"

# Build targets for each MCP server
blues-expert: ## Build the Blues Expert MCP server
	@echo "Building blues-expert..."
	cd $(BLUES_EXPERT_DIR) && $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o blues-expert

inspect-blues-expert: blues-expert ## Run MCP inspector for blues-expert
	@echo "Starting MCP inspector for blues-expert..."
	npx @modelcontextprotocol/inspector --config mcp.json --server blues-expert

# Treat the argument as a target to prevent "No rule to make target" errors
%:
	@:

# Documentation targets
docs: docs-blues-expert ## Generate documentation for all packages

docs-blues-expert: ## Generate Go documentation for blues-expert package
	@echo "Generating Blues Expert API documentation..."
	@mkdir -p blues-expert/docs
	@go doc -all ./blues-expert > ./blues-expert/docs/API_DOCS.md
	@echo "✓ Blues Expert documentation generated: blues-expert/docs/API_DOCS.md"

# Development targets
test: ## Run tests for all packages
	@echo "Running tests..."
	$(GO) test ./... -v

clean: ## Clean build artifacts and generated docs
	@echo "Cleaning build artifacts..."
	rm -f $(BLUES_EXPERT_DIR)/blues-expert
	@rm -rf blues-expert/docs
	@echo "✓ Clean complete"

# Utility targets
fmt: ## Format Go code
	@echo "Formatting Go code..."
	@go fmt ./blues-expert/...
	@echo "✓ Code formatted"

vet: ## Run go vet on all packages
	@go vet ./blues-expert/...
	@echo "✓ Vet complete"

develop: fmt vet test docs build ## Run full development workflow (format, vet, test, docs, build)
