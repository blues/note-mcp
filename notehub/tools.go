// Package main provides MCP (Model Context Protocol) tools for interacting with the Notehub API.
//
// This package implements a comprehensive set of tools for managing Blues Wireless Notehub
// projects, devices, routes, monitors, and encrypted communications.
//
// Available Tools:
//   - project_list: List all Notehub projects
//   - project_create: Create a new Notehub project
//   - device_list: List devices in a project (with optional tag filtering)
//   - device_health_log: Get device health information
//   - device_public_key: Get device public key for encryption
//   - project_events: List events in a project
//   - check_notefiles: Check for Notefiles sent to Notehub by filtering project events
//   - route_list: List routes in a project
//   - route_detail: Get detailed route information
//   - monitor_list: List monitors in a project
//   - monitor_detail: Get detailed monitor information
//   - send_note: Send a note to a device
//   - send_encrypted_note: Send an encrypted note using device public key
//   - billing_account_list: List all billing accounts
//   - product_create: Create a new product in a project
//   - product_list: List all products in a project
//   - environment_variables_set: Set environment variables at device, fleet, or project scope
//   - fleet_list: List all fleets in a project
//   - fleet_get: Get detailed information about a specific fleet
//   - device_dfu_history: Get device DFU history for host or notecard firmware
//   - firmware_host_upload: Upload host firmware binary to a project
//
// Authentication:
//
//	Uses session token authentication via NOTEHUB_USER and NOTEHUB_PASS environment variables.
//	Tokens are automatically refreshed as needed.
//
// API Base URL: https://api.notefile.net
//
// Example .env file:
//
//	NOTEHUB_USER=your@email.com
//	NOTEHUB_PASS=your_password
package main

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// CreateProjectListTool creates a tool for listing Notehub projects
func CreateProjectListTool() mcp.Tool {
	return mcp.NewTool("project_list",
		mcp.WithDescription("List all Notehub projects belonging to the authenticated user"),
	)
}

// CreateProjectDetailTool creates a tool for getting detailed information about a specific project
func CreateProjectDetailTool() mcp.Tool {
	return mcp.NewTool("project_detail",
		mcp.WithDescription("Get detailed information about a specific project in a Notehub project"),
		mcp.WithString("project_uid",
			mcp.Required(),
			mcp.Description("The UID of the project to get details for"),
		),
	)
}

// CreateProjectCreateTool creates a tool for creating a new Notehub project
func CreateProjectCreateTool() mcp.Tool {
	return mcp.NewTool("project_create",
		mcp.WithDescription("Create a new Notehub project. You will need to have a billing account to create a project. You should typically follow up by issuing a 'product_create' request to create a new product in the project, as a Product UID is needed to provision a Notecard."),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("The name of the project to create"),
		),
		mcp.WithString("billing_account_uid",
			mcp.Required(),
			mcp.Description("The UID of the billing account to associate with the project"),
		),
		mcp.WithString("description",
			mcp.Description("Optional description for the project"),
		),
	)
}

// CreateDeviceListTool creates a tool for listing devices in a Notehub project.
//
// This tool provides access to the Notehub API endpoint:
// GET /v1/projects/{projectUID}/devices
//
// Parameters:
//   - project_uid (required): The UID of the project to list devices for
//   - pageSize (optional): Number of devices to return per page (default 50)
//   - pageNum (optional): Page number of results (must be >= 1, default 1)
//   - deviceUID (optional): Array of specific device UIDs to filter by
//   - tags (optional): Array of tags to filter devices by
//   - serialNumber (optional): Array of serial numbers to filter devices by
//
// Example usage:
//   - List all devices: {"project_uid": "app:123..."}
//   - Filter by tags: {"project_uid": "app:123...", "tags": ["production", "sensor"]}
//   - Filter by device UIDs: {"project_uid": "app:123...", "deviceUID": ["dev:123...", "dev:456..."]}
//   - Paginated results: {"project_uid": "app:123...", "pageSize": 25, "pageNum": 2}
//
// Returns:
//
//	JSON array of device objects with their metadata and status information.
func CreateDeviceListTool() mcp.Tool {
	return mcp.NewTool("device_list",
		mcp.WithDescription("List all devices in a specific Notehub project with optional filtering and pagination"),
		mcp.WithString("project_uid",
			mcp.Required(),
			mcp.Description("The UID of the project to list devices for (format: app:xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)"),
		),
		mcp.WithNumber("pageSize",
			mcp.Description("Optional number of devices to return per page (defaults to 50, try larger values if you have a lot of devices)"),
		),
		mcp.WithNumber("pageNum",
			mcp.Description("Optional page number of results (must be >= 1, defaults to 1)"),
		),
		mcp.WithArray("deviceUID",
			mcp.Description("Optional array of specific device UIDs to filter by. Example: ['dev:864475012345678', 'dev:864622087654321']"),
			mcp.Items(map[string]any{
				"type": "string",
			}),
		),
		mcp.WithArray("tags",
			mcp.Description("Optional array of tags to filter devices by. Example: ['production', 'sensor', 'outdoor']"),
			mcp.Items(map[string]any{
				"type": "string",
			}),
		),
		mcp.WithArray("serialNumber",
			mcp.Description("Optional array of device serial numbers to filter by. Example: ['SN001', 'SN002']"),
			mcp.Items(map[string]any{
				"type": "string",
			}),
		),
	)
}

// CreateProjectEventsTool creates a tool for listing events in a Notehub project
func CreateProjectEventsTool() mcp.Tool {
	return mcp.NewTool("project_events",
		mcp.WithDescription("List all events in a specific Notehub project"),
		mcp.WithString("project_uid",
			mcp.Required(),
			mcp.Description("The UID of the project to list events for"),
		),
	)
}

// CreateCheckNotefilesTool creates a tool for checking Notefiles sent to Notehub
func CreateCheckNotefilesTool() mcp.Tool {
	return mcp.NewTool("check_notefiles",
		mcp.WithDescription("Check for Notefiles that have been sent to Notehub by filtering project events to show only Notefile-related data"),
		mcp.WithString("project_uid",
			mcp.Required(),
			mcp.Description("The UID of the project to check for Notefiles"),
		),
		mcp.WithString("device_uid",
			mcp.Description("Optional: The specific device UID to filter for (e.g., 'dev:123456789'). If not provided, will show Notefiles from all devices in the project."),
		),
	)
}

// CreateSendNoteTool creates a tool for sending a note to a device
func CreateSendNoteTool() mcp.Tool {
	return mcp.NewTool("send_note",
		mcp.WithDescription("Send a note to a specific device in a Notehub project"),
		mcp.WithString("project_uid",
			mcp.Required(),
			mcp.Description("The UID of the project containing the target device"),
		),
		mcp.WithString("device_uid",
			mcp.Required(),
			mcp.Description("The UID of the device to send the note to"),
		),
		mcp.WithString("note_file",
			mcp.Required(),
			mcp.Description("The note file name, it should always be of *.qi or *.db as it is an inbound note (e.g., 'data.qi' for an incoming queue or 'data.db' for a bidirectionally synchronized database). If the user wishes enabled TLS encryption, the note file name should be of *.qis or *.dbs."),
		),
		mcp.WithString("note_body",
			mcp.Required(),
			mcp.Description("The JSON body content of the note"),
		),
	)
}

// CreateRouteListTool creates a tool for listing routes in a Notehub project
func CreateRouteListTool() mcp.Tool {
	return mcp.NewTool("route_list",
		mcp.WithDescription("List all routes in a specific Notehub project"),
		mcp.WithString("project_uid",
			mcp.Required(),
			mcp.Description("The UID of the project to list routes for"),
		),
	)
}

// CreateRouteDetailTool creates a tool for getting detailed information about a specific route
func CreateRouteDetailTool() mcp.Tool {
	return mcp.NewTool("route_detail",
		mcp.WithDescription("Get detailed information about a specific route in a Notehub project"),
		mcp.WithString("project_uid",
			mcp.Required(),
			mcp.Description("The UID of the project containing the route"),
		),
		mcp.WithString("route_uid",
			mcp.Required(),
			mcp.Description("The UID of the route to get details for"),
		),
	)
}

// CreateDeviceHealthLogTool creates a tool for getting device health log information
func CreateDeviceHealthLogTool() mcp.Tool {
	return mcp.NewTool("device_health_log",
		mcp.WithDescription("Get device health log information for a specific device in a Notehub project"),
		mcp.WithString("project_uid",
			mcp.Required(),
			mcp.Description("The UID of the project containing the device"),
		),
		mcp.WithString("device_uid",
			mcp.Required(),
			mcp.Description("The UID of the device to get health log for"),
		),
	)
}

// CreateMonitorListTool creates a tool for listing monitors in a Notehub project
func CreateMonitorListTool() mcp.Tool {
	return mcp.NewTool("monitor_list",
		mcp.WithDescription("List all monitors in a specific Notehub project"),
		mcp.WithString("project_uid",
			mcp.Required(),
			mcp.Description("The UID of the project to list monitors for"),
		),
	)
}

// CreateMonitorDetailTool creates a tool for getting detailed information about a specific monitor
func CreateMonitorDetailTool() mcp.Tool {
	return mcp.NewTool("monitor_detail",
		mcp.WithDescription("Get detailed information about a specific monitor in a Notehub project"),
		mcp.WithString("project_uid",
			mcp.Required(),
			mcp.Description("The UID of the project containing the monitor"),
		),
		mcp.WithString("monitor_uid",
			mcp.Required(),
			mcp.Description("The UID of the monitor to get details for"),
		),
	)
}

// CreateDevicePublicKeyTool creates a tool for getting device public key information
func CreateDevicePublicKeyTool() mcp.Tool {
	return mcp.NewTool("device_public_key",
		mcp.WithDescription("Get device public key information for a specific device in a Notehub project"),
		mcp.WithString("project_uid",
			mcp.Required(),
			mcp.Description("The UID of the project containing the device"),
		),
		mcp.WithString("device_uid",
			mcp.Required(),
			mcp.Description("The UID of the device to get public key for"),
		),
	)
}

// CreateSendEncryptedNoteTool creates a tool for sending encrypted notes to devices
func CreateSendEncryptedNoteTool() mcp.Tool {
	return mcp.NewTool("send_encrypted_note",
		mcp.WithDescription("Send an encrypted note to a specific device using its public key"),
		mcp.WithString("project_uid",
			mcp.Required(),
			mcp.Description("The UID of the project containing the target device"),
		),
		mcp.WithString("device_uid",
			mcp.Required(),
			mcp.Description("The UID of the device to send the encrypted note to"),
		),
		mcp.WithString("note_file",
			mcp.Required(),
			mcp.Description("The note file name, it should always be of *.qi or *.db as it is an inbound note (e.g., 'data.qi' for an incoming queue or 'data.db' for a bidirectionally synchronized database). If the user wishes enabled TLS encryption (in addition to encrypting the payload), the note file name should be of *.qis or *.dbs."),
		),
		mcp.WithString("plaintext_message",
			mcp.Required(),
			mcp.Description("The plaintext message to encrypt and send"),
		),
	)
}

// CreateBillingAccountListTool creates a tool for listing billing accounts
func CreateBillingAccountListTool() mcp.Tool {
	return mcp.NewTool("billing_account_list",
		mcp.WithDescription("List all billing accounts belonging to the authenticated user"),
	)
}

// CreateProductCreateTool creates a tool for creating a new product in a Notehub project
func CreateProductCreateTool() mcp.Tool {
	return mcp.NewTool("product_create",
		mcp.WithDescription("Create a new product in a specific Notehub project"),
		mcp.WithString("project_uid",
			mcp.Required(),
			mcp.Description("The UID of the project to create the product in"),
		),
		mcp.WithString("product_uid",
			mcp.Required(),
			mcp.Description("The UID for the new product, (without the reverse URL prefix notation). For example, if the desired product UID is 'com.company.username:productname', the value should be 'productname'."),
		),
		mcp.WithString("label",
			mcp.Required(),
			mcp.Description("The label/name/description of the new product"),
		),
		mcp.WithString("auto_provision_fleets",
			mcp.Description("Optional fleet UIDs for auto-provisioning (comma-separated)"),
		),
		mcp.WithString("disable_devices_by_default",
			mcp.Description("Optional: whether to disable devices by default (true/false)"),
		),
	)
}

// CreateProductListTool creates a tool for listing products in a Notehub project
func CreateProductListTool() mcp.Tool {
	return mcp.NewTool("product_list",
		mcp.WithDescription("List all products in a specific Notehub project"),
		mcp.WithString("project_uid",
			mcp.Required(),
			mcp.Description("The UID of the project to list products for"),
		),
	)
}

// CreateEnvironmentVariablesSetTool creates a tool for setting environment variables at device, fleet, or project scope
func CreateEnvironmentVariablesSetTool() mcp.Tool {
	return mcp.NewTool("environment_variables_set",
		mcp.WithDescription("Set environment variables at device, fleet, or project scope in a Notehub project"),
		mcp.WithString("project_uid",
			mcp.Required(),
			mcp.Description("The UID of the project"),
		),
		mcp.WithString("scope",
			mcp.Required(),
			mcp.Enum("device", "fleet", "project"),
			mcp.Description("The scope for setting environment variables: 'device', 'fleet', or 'project'"),
		),
		mcp.WithString("uid",
			mcp.Description("The UID of the specified scope (e.g. device or fleet)"),
		),
		mcp.WithString("environment_variables",
			mcp.Required(),
			mcp.Description("JSON string of environment variables to set (e.g., '{\"VAR1\":\"value1\",\"VAR2\":\"value2\"}')"),
		),
	)
}

// CreateFleetListTool creates a tool for listing fleets in a Notehub project
func CreateFleetListTool() mcp.Tool {
	return mcp.NewTool("fleet_list",
		mcp.WithDescription("List all fleets in a specific Notehub project"),
		mcp.WithString("project_uid",
			mcp.Required(),
			mcp.Description("The UID of the project to list fleets for"),
		),
	)
}

// CreateFleetGetTool creates a tool for getting detailed information about a specific fleet
func CreateFleetGetTool() mcp.Tool {
	return mcp.NewTool("fleet_get",
		mcp.WithDescription("Get detailed information about a specific fleet in a Notehub project"),
		mcp.WithString("project_uid",
			mcp.Required(),
			mcp.Description("The UID of the project containing the fleet"),
		),
		mcp.WithString("fleet_uid",
			mcp.Required(),
			mcp.Description("The UID of the fleet to get details for"),
		),
	)
}

// CreateDeviceDfuHistoryTool creates a tool for getting device DFU history
func CreateDeviceDfuHistoryTool() mcp.Tool {
	return mcp.NewTool("device_dfu_history",
		mcp.WithDescription("Get device DFU (Device Firmware Update) history for a specific device in a Notehub project"),
		mcp.WithString("project_uid",
			mcp.Required(),
			mcp.Description("The UID of the project containing the device"),
		),
		mcp.WithString("device_uid",
			mcp.Required(),
			mcp.Description("The UID of the device to get DFU history for"),
		),
		mcp.WithString("firmware_type",
			mcp.Required(),
			mcp.Enum("host", "notecard"),
			mcp.Description("The type of firmware to get history for: 'host' or 'notecard'"),
		),
	)
}

// CreateFirmwareHostUploadTool creates a tool for uploading host firmware binary to Notehub
func CreateFirmwareHostUploadTool() mcp.Tool {
	return mcp.NewTool("firmware_host_upload",
		mcp.WithDescription("Upload a host firmware binary file to a Notehub project"),
		mcp.WithString("project_uid",
			mcp.Required(),
			mcp.Description("The UID of the project to upload firmware to"),
		),
		mcp.WithString("file_path",
			mcp.Required(),
			mcp.Description("The local file system path to the firmware binary file (e.g., '/path/to/app-v1.0.0.bin')"),
		),
		mcp.WithString("org",
			mcp.Description("Optional organization identifier"),
		),
		mcp.WithString("product",
			mcp.Description("Optional product identifier"),
		),
		mcp.WithString("firmware",
			mcp.Description("Optional firmware identifier"),
		),
		mcp.WithString("version",
			mcp.Description("Optional version string"),
		),
		mcp.WithString("target",
			mcp.Description("Optional target identifier"),
		),
		mcp.WithString("version_string",
			mcp.Description("Optional version string (e.g., '1.2.3'). Will be parsed into major.minor.patch components"),
		),
		mcp.WithString("built",
			mcp.Description("Optional build timestamp"),
		),
		mcp.WithString("builder",
			mcp.Description("Optional builder identifier"),
		),
	)
}
