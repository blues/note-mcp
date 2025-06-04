# Makefile for note-mcp project

.PHONY: all clean build test notecard notehub inspect inspect-notecard inspect-notehub help docs docs-notehub docs-notecard fmt vet dev

# Go build flags
GO=go
GOFLAGS=-v
LDFLAGS=-w -s

# Project directories
NOTECARD_DIR=./notecard
NOTEHUB_DIR=./notehub

# Default target
all: build

# Help target
help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Build targets
build: notecard notehub ## Build all MCP servers

notecard: ## Build the Notecard MCP server
	@echo "Building notecard..."
	cd $(NOTECARD_DIR) && $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o notecard

notehub: ## Build the Notehub MCP server
	@echo "Building notehub..."
	cd $(NOTEHUB_DIR) && $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o notehub

inspect-notecard: notecard ## Run MCP inspector for notecard
	@echo "Starting MCP inspector for notecard..."
	npx @modelcontextprotocol/inspector --config mcp.json --server notecard

inspect-notehub: notehub ## Run MCP inspector for notehub
	@echo "Starting MCP inspector for notehub..."
	npx @modelcontextprotocol/inspector --config mcp.json --server notehub

# Treat the argument as a target to prevent "No rule to make target" errors
%:
	@:

# Documentation targets
docs: docs-notehub docs-notecard ## Generate documentation for all packages

docs-notehub: ## Generate Go documentation for notehub package
	@echo "Generating Notehub API documentation..."
	@mkdir -p notehub/docs
	@go doc -all ./notehub > ./notehub/docs/API_DOCS.md
	@echo "✓ Notehub documentation generated: notehub/docs/API_DOCS.md"

docs-notecard: ## Generate Go documentation for notecard package
	@echo "Generating Notecard API documentation..."
	@mkdir -p notecard/docs
	@go doc -all ./notecard > ./notecard/docs/API_DOCS.md
	@echo "✓ Notecard documentation generated: notecard/docs/API_DOCS.md"

# Development targets
test: ## Run tests for all packages
	@echo "Running tests..."
	$(GO) test ./... -v

clean: ## Clean build artifacts and generated docs
	@echo "Cleaning build artifacts..."
	rm -f $(NOTECARD_DIR)/notecard
	rm -f $(NOTEHUB_DIR)/notehub
	@rm -rf notehub/docs notecard/docs
	@echo "✓ Clean complete"

# Utility targets
fmt: ## Format Go code
	@echo "Formatting Go code..."
	@go fmt ./notehub/... ./notecard/...
	@echo "✓ Code formatted"

vet: ## Run go vet on all packages
	@echo "Running go vet..."
	@go vet ./notehub/... ./notecard/...
	@echo "✓ Vet complete"

dev: fmt vet test docs build ## Run full development workflow (format, vet, test, docs, build)
