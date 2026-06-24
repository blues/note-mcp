# note-mcp

Blues Expert MCP server for Notecard & Notehub development.

> [!WARNING]
> This MCP server is experimental and subject to change. Please wait until a versioned release is available before relying on it.

## About

The Blues Expert MCP server is a remote tool designed to help you develop Notecard projects.
When used with an LLM, it provides guidance on best practices for writing firmware and leveraging Notecard's capabilities.
It provides correct and accurate information about Notecard, reducing hallucinations and errors when building Notecard projects.

## Build

Requirements:

- Go (at least v1.23)
- Make
- [Docker](https://www.docker.com/products/docker-desktop/)

```bash
make build
```

## Install

Add the following to your `mcp.json` file (e.g. for Claude Desktop, this is `claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "blues-expert": {
      "type": "http",
      "url": "http://localhost:8080/expert/mcp"
    }
  }
}
```

## Development

To run the MCP inspector, you'll need Node.js installed (at least v18).

```bash
make inspect-blues-expert
```
