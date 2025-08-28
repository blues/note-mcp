# Blues Expert MCP

## Setup

### Claude Code

```bash
claude mcp add blues-expert https://mcp.blues.io/expert/mcp --transport http
```

### Cursor

[![Install MCP Server](https://cursor.com/deeplink/mcp-install-dark.svg)](https://cursor.com/en/install-mcp?name=blues%20expert&config=eyJ1cmwiOiJodHRwczovL21jcC5ibHVlcy5pby9leHBlcnQvbWNwIn0%3D)

or

```json
{
    "mcpServers": {
        "blues expert": {
        "url": "https://mcp.blues.io/expert/mcp"
        }
    }
}
```

## Deployment

### Local

For local deployment, you can use the following command:

```bash
cd .. # go to the root of the project
docker build -f blues-expert/Dockerfile -t blues-expert .
docker run -d --name blues-expert blues-expert
```

### AWS

For AWS deployment, AppRunner is used.
