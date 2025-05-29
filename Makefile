.PHONY: all clean build test notecard notehub inspect

# Go build flags
GO=go
GOFLAGS=-v
LDFLAGS=-w -s

# Project directories
NOTECARD_DIR=./notecard
NOTEHUB_DIR=./notehub

all: build

build: notecard notehub

inspect:
	@if [ "$(filter-out $@,$(MAKECMDGOALS))" = "notecard" ]; then \
		$(MAKE) notecard && npx @modelcontextprotocol/inspector --config mcp.json --server notecard; \
	elif [ "$(filter-out $@,$(MAKECMDGOALS))" = "notehub" ]; then \
		$(MAKE) notehub && npx @modelcontextprotocol/inspector --config mcp.json --server notehub; \
	else \
		echo "Usage: make inspect <notecard|notehub>"; \
		exit 1; \
	fi

# Treat the argument as a target to prevent "No rule to make target" errors
%:
	@:

notecard:
	@echo "Building notecard..."
	cd $(NOTECARD_DIR) && $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o notecard

notehub:
	@echo "Building notehub..."
	cd $(NOTEHUB_DIR) && $(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o notehub

test:
	@echo "Running tests..."
	$(GO) test ./... -v

clean:
	@echo "Cleaning build artifacts..."
	rm -f $(NOTECARD_DIR)/notecard
	rm -f $(NOTEHUB_DIR)/notehub
