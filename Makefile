# Makefile for note-mcp project

.PHONY: all clean build test notecard notehub blues-expert inspect inspect-notecard inspect-notehub inspect-blues-expert help docs docs-notehub docs-notecard docs-blues-expert fmt vet develop

# Go build flags
GO=go
GOFLAGS=-v
LDFLAGS=-w -s

# Project directories
NOTECARD_DIR=./notecard
NOTEHUB_DIR=./notehub
BLUES_EXPERT_DIR=./blues-expert

# Default target
all: vet build

# Help target
help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Build targets
build: notecard notehub blues-expert ## Build all MCP servers
	@echo "✓ Build complete"

# Build targets for each MCP server
notecard: ## Build the Notecard MCP server
	@echo "Building notecard..."
	cd $(NOTECARD_DIR) && $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o notecard

notehub: ## Build the Notehub MCP server
	@echo "Building notehub..."
	cd $(NOTEHUB_DIR) && $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o notehub

blues-expert: ## Build the Blues Expert MCP server
	@echo "Building blues-expert..."
	cd $(BLUES_EXPERT_DIR) && $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o blues-expert

inspect-notecard: notecard ## Run MCP inspector for notecard
	@echo "Starting MCP inspector for notecard..."
	npx @modelcontextprotocol/inspector --config mcp.json --server notecard

inspect-notehub: notehub ## Run MCP inspector for notehub
	@echo "Starting MCP inspector for notehub..."
	npx @modelcontextprotocol/inspector --config mcp.json --server notehub

inspect-blues-expert: blues-expert ## Run MCP inspector for blues-expert
	@echo "Starting MCP inspector for blues-expert..."
	npx @modelcontextprotocol/inspector --config mcp.json --server blues-expert

# Treat the argument as a target to prevent "No rule to make target" errors
%:
	@:

# Documentation targets
docs: docs-notehub docs-notecard docs-blues-expert ## Generate documentation for all packages

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
	rm -f $(NOTECARD_DIR)/notecard
	rm -f $(NOTEHUB_DIR)/notehub
	rm -f $(BLUES_EXPERT_DIR)/blues-expert
	@rm -rf notehub/docs notecard/docs blues-expert/docs
	@echo "✓ Clean complete"

# Utility targets
fmt: ## Format Go code
	@echo "Formatting Go code..."
	@go fmt ./notehub/... ./notecard/... ./blues-expert/...
	@echo "✓ Code formatted"

vet: ## Run go vet on all packages
	@go vet ./notehub/... ./notecard/... ./blues-expert/...
	@echo "✓ Vet complete"

develop: fmt vet test docs build ## Run full development workflow (format, vet, test, docs, build)
