# Blues Expert MCP

## Deployment

### Local

For local deployment, you can use the following command:

```bash
docker compose up -d
```

### AWS

For AWS deployment, AppRunner is used. The following command can be used to deploy the service:

```bash
aws apprunner create-service --cli-input-json file://apprunner-service.json
```

The `apprunner-service.json` file is a JSON file that contains the configuration for the AppRunner service.
