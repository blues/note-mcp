package main

import (
	"github.com/mark3labs/mcp-go/mcp"
)

func CreateNotecardInitializeTool() mcp.Tool {
	return mcp.NewTool("notecard_initialize",
		mcp.WithDescription("Initialize a connection to the Notecard for communication. This creates a notecard object that can be used for subsequent operations."),
		mcp.WithString("port",
			mcp.Description("The port to connect to the Notecard. If not provided, the default port will be used."),
		),
		mcp.WithNumber("baud",
			mcp.Description("The baud rate to connect to the Notecard. If not provided, the default baud rate will be used."),
		),
		mcp.WithBoolean("reset",
			mcp.Description("Reset the Notecard connection and initialize a new connection. Only use this if instructed to do so by the user as it will delete the config file. If provided, the default connection will be used."),
		),
	)
}

func CreateNotecardRequestTool() mcp.Tool {
	return mcp.NewTool("notecard_request",
		mcp.WithDescription("Send a request to the Notecard and return the response. The notecard must be initialized first and you should check you have up-to-date API documentation before sending it (by using the 'notecard_get_apis' tool). The request type is the JSON string of the request to send to the Notecard, e.g. '{\"req\":\"card.version\"}' or '{\"req\":\"card.temp\",\"minutes\":60}'. If a request is unknown, you can use the 'notecard_request_validate' tool to validate it. If the request is valid, you can use this tool to send it to the Notecard."),
		mcp.WithString("request",
			mcp.Required(),
			mcp.Description("The request type to send to the Notecard (e.g., '{\"req\":\"card.version\"}', '{\"req\":\"card.status\"}', '{\"req\":\"hub.status\"}', '{\"req\":\"card.temp\",\"minutes\":60}', '{\"req\":\"card.voltage\"}')"),
		),
	)
}

func CreateNotecardListFirmwareVersionsTool() mcp.Tool {
	return mcp.NewTool("notecard_firmware_versions",
		mcp.WithDescription("Lists all available firmware versions for a given update channel and Notecard model."),
		mcp.WithString("updateChannel",
			mcp.Required(),
			mcp.Description("The type of update to list versions for (e.g., 'LTS', 'DevRel', 'nightly')"),
		),
		mcp.WithString("notecardModel",
			mcp.Required(),
			mcp.Description("The model of Notecard to list versions for (e.g., 'NOTE-WBEXW', 'NOTE-NBGL-500', etc.)"),
		),
	)
}

func CreateNotecardUpdateFirmwareTool() mcp.Tool {
	return mcp.NewTool("notecard_firmware_update",
		mcp.WithDescription("Downloads and flashes firmware to the Notecard. This tool will download the specified firmware version and perform a sideload update to the connected Notecard. Warn the user that this may take some time and that the Notecard will restart. The Notecard must be initialized first."),
		mcp.WithString("updateChannel",
			mcp.Description("The firmware update channel (e.g., 'LTS', 'DevRel', 'nightly')"),
		),
		mcp.WithString("notecardModel",
			mcp.Required(),
			mcp.Description("The model of Notecard to update (e.g., 'NOTE-WBEXW', 'NOTE-NBGL-500', etc.)"),
		),
		mcp.WithString("version",
			mcp.Description("The specific firmware version to install (e.g., '8.1.4.17149$20250319220838'). If not provided, will use the latest available version."),
		),
		mcp.WithBoolean("force",
			mcp.Description("Force the update even if the version is the same or older (default: false)"),
		),
	)
}

func CreateNotecardValidateRequestTool() mcp.Tool {
	return mcp.NewTool("notecard_request_validate",
		mcp.WithDescription("Validate a Notecard API request against the Notecard API Schema. This helps ensure your request is valid before sending it to the Notecard. To use this tool, you must have the environment variable BLUES exported, otherwise it will just check for JSON validity."),
		mcp.WithString("request",
			mcp.Required(),
			mcp.Description("The JSON string of the request to validate (e.g., '{\"req\":\"card.version\"}', '{\"req\":\"card.temp\",\"minutes\":60}')"),
		),
		mcp.WithString("schema_url",
			mcp.Description("The schema URL to validate against. If not provided, uses the default Notecard API schema."),
		),
	)
}

func CreateDecryptNoteTool() mcp.Tool {
	return mcp.NewTool("notecard_decrypt_note",
		mcp.WithDescription("Get instructions for decrypting a note using the Notecard. Returns step-by-step instructions to sync with the hub and decrypt the note."),
		mcp.WithString("note_file",
			mcp.Description("The note file to decrypt (e.g., 'mysecrets.qi'). Defaults to 'mysecrets.qi' if not provided."),
		),
	)
}

func CreateProvisionNotecardTool() mcp.Tool {
	return mcp.NewTool("notecard_provision",
		mcp.WithDescription("Get instructions for provisioning a new Notecard to a Notehub project, creating a new project if necessary. Returns comprehensive provisioning steps including WiFi configuration if applicable."),
		mcp.WithString("product_uid",
			mcp.Required(),
			mcp.Description("Product UID within the project, typically in the format 'com.company.username:productname'. Do not assume you know the Product UID. You should first find the Product UID using the 'product_list' tool via the Notehub MCP tool."),
		),
		mcp.WithString("project_uid",
			mcp.Required(),
			mcp.Description("Project UID, typically in the format 'app:xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx'."),
		),
		mcp.WithString("ssid",
			mcp.Description("Optional SSID for WiFi provisioning. Do not assume you know the SSID. If the Notecard supports WiFi, stop and ask the user for the SSID."),
		),
		mcp.WithString("password",
			mcp.Description("Optional password for WiFi provisioning. Do not assume you know the password. If the Notecard supports WiFi, stop and ask the user for the password."),
		),
	)
}

func CreateTroubleshootConnectionTool() mcp.Tool {
	return mcp.NewTool("notecard_troubleshoot_connection",
		mcp.WithDescription("Get troubleshooting instructions for when a Notecard will not connect to Notehub. Provides step-by-step diagnostic and resolution suggestions."),
		mcp.WithString("error_message",
			mcp.Required(),
			mcp.Description("Error message or symptom description from the Notecard"),
		),
		mcp.WithString("notecard_type",
			mcp.Required(),
			mcp.Description("Notecard type (e.g., 'WiFi', 'Cellular', 'LoRa') to provide type-specific troubleshooting"),
		),
	)
}

func CreateSendNoteTool() mcp.Tool {
	return mcp.NewTool("notecard_send_note",
		mcp.WithDescription("Get instructions for sending notes from a Notecard to Notehub. Returns detailed steps for adding data to notefiles and syncing to the cloud."),
		mcp.WithString("note_file",
			mcp.Description("The name of the notefile to send data to (e.g., 'sensors.qo', 'data.qo'). Use '.qo' extension for outbound queue files or '.dbo' for database files. To use TLS encryption, use .qos and .dbos. Defaults to 'data.qo' if not provided."),
		),
		mcp.WithString("note_data",
			mcp.Description("JSON string of the data to include in the note body, e.g. '{\"temperature\":23.5,\"humidity\":65,\"timestamp\":\"2024-01-15T10:30:00Z\"}'"),
		),
		mcp.WithString("note_sync",
			mcp.Description("Sync the note immediately. Defaults to false."),
		),
	)
}

func CreateNotecardGetAPIsTool() mcp.Tool {
	return mcp.NewTool("notecard_get_apis",
		mcp.WithDescription("Get comprehensive documentation for all Notecard APIs or a specific API category. Returns detailed API information organized by category including endpoints, parameters, and usage examples."),
		mcp.WithString("category",
			mcp.Description("Optional API category to get specific documentation for. Valid categories: 'card', 'hub', 'note', 'env', 'file', 'web', 'var', 'ntn', 'dfu'. If not provided, returns overview of all APIs."),
		),
	)
}
