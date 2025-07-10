# Makefile for note-mcp project

.PHONY: all clean build test notecard notehub inspect inspect-notecard inspect-notehub help docs docs-notehub docs-notecard fmt vet dev

# Go build flags
GO=go
GOFLAGS=-v
LDFLAGS=-w -s

# Project directories
NOTECARD_DIR=./notecard
NOTEHUB_DIR=./notehub
DEV_DIR=./dev

# Default target
all: vet build

# Help target
help: ## Show this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2}'

# Build targets
build: notecard notehub dev ## Build all MCP servers
	@echo "✓ Build complete"

# Build targets for each MCP server
notecard: ## Build the Notecard MCP server
	@echo "Building notecard..."
	cd $(NOTECARD_DIR) && $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o notecard

notehub: ## Build the Notehub MCP server
	@echo "Building notehub..."
	cd $(NOTEHUB_DIR) && $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o notehub

dev: ## Build the Dev MCP server
	@echo "Building dev..."
	cd $(DEV_DIR) && $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o dev

# Build targets for each MCP server with DXT
dxt: dxt-notecard dxt-notehub dxt-dev
	@echo "✓ DXT build complete"

# Generate DXT files
dxt-notecard: notecard ## Generate DXT file for notecard
	@echo "Generating DXT file for notecard..."
	cp ./assets/icon.png $(NOTECARD_DIR)/icon.png
	cd $(NOTECARD_DIR) && dxt pack
	rm $(NOTECARD_DIR)/icon.png

dxt-notehub: notehub ## Generate DXT file for notehub
	@echo "Generating DXT file for notehub..."
	cp ./assets/icon.png $(NOTEHUB_DIR)/icon.png
	cd $(NOTEHUB_DIR) && dxt pack
	rm $(NOTEHUB_DIR)/icon.png

dxt-dev: dev ## Generate DXT file for dev
	@echo "Generating DXT file for dev..."
	cp ./assets/icon.png $(DEV_DIR)/icon.png
	cd $(DEV_DIR) && dxt pack
	rm $(DEV_DIR)/icon.png

inspect-notecard: notecard ## Run MCP inspector for notecard
	@echo "Starting MCP inspector for notecard..."
	npx @modelcontextprotocol/inspector --config mcp.json --server notecard

inspect-notehub: notehub ## Run MCP inspector for notehub
	@echo "Starting MCP inspector for notehub..."
	npx @modelcontextprotocol/inspector --config mcp.json --server notehub

inspect-dev: dev ## Run MCP inspector for dev
	@echo "Starting MCP inspector for dev..."
	npx @modelcontextprotocol/inspector --config mcp.json --server dev

# Treat the argument as a target to prevent "No rule to make target" errors
%:
	@:

# Documentation targets
docs: docs-notehub docs-notecard docs-dev ## Generate documentation for all packages

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

docs-dev: ## Generate Go documentation for dev package
	@echo "Generating Dev API documentation..."
	@mkdir -p dev/docs
	@go doc -all ./dev > ./dev/docs/API_DOCS.md
	@echo "✓ Dev documentation generated: dev/docs/API_DOCS.md"

# Development targets
test: ## Run tests for all packages
	@echo "Running tests..."
	$(GO) test ./... -v

clean: ## Clean build artifacts and generated docs
	@echo "Cleaning build artifacts..."
	rm -f $(NOTECARD_DIR)/notecard
	rm -f $(NOTEHUB_DIR)/notehub
	rm -f $(DEV_DIR)/dev
	find . -name "*.dxt" -type f -delete
	rm -rf ./dist ./dxt
	@rm -rf notehub/docs notecard/docs dev/docs
	@echo "✓ Clean complete"

# Utility targets
fmt: ## Format Go code
	@echo "Formatting Go code..."
	@go fmt ./notehub/... ./notecard/... ./dev/...
	@echo "✓ Code formatted"

vet: ## Run go vet on all packages
	@go vet ./notehub/... ./notecard/... ./dev/...
	@echo "✓ Vet complete"

develop: fmt vet test docs build ## Run full development workflow (format, vet, test, docs, build)
