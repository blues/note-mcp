# note-mcp

MCP servers for Notecard, Notehub, and Development.

> [!WARNING]
> These MCP servers are experimental and subject to change. Please wait until a versioned release is available before relying on them.

## Build

```bash
make build
```

## Install

Add the following to your `mcp.json` file, where `mcp.json` is the file that determines where the MCP servers are located (e.g. for Claude Desktop, this is `claude_desktop_config.json`):

```json
{
    "mcpServers" : {
        "notecard": {
            "command": "/absolute/path/to/notecard/notecard",
            "args": [
                "--env",
                "/absolute/path/to/.env"
            ]
        },
        "notehub": {
            "command": "/absolute/path/to/notehub/notehub",
            "args": [
                "--env",
                "/absolute/path/to/.env"
            ]
        },
        "dev": {
            "command": "/absolute/path/to/dev/dev",
            "args": [
                "--env",
                "/absolute/path/to/.env"
            ]
        }
    }
}
```

The `.env` file should contain the following variables:

```bash
NOTEHUB_USER="your_notehub_username"
NOTEHUB_PASS="your_notehub_password"
```

Additional variables will be added.

## Development

To run the MCP inspector, you'll need nodejs installed (atleast v18).

For Notecard MCP:

```bash
make inspect notecard
```

For Notehub MCP:

```bash
make inspect notehub
```

For dev MCP:

```bash
make inspect dev
```
