Package main provides MCP (Model Context Protocol) tools for interacting with
the Notehub API.

This package implements a comprehensive set of tools for managing Blues Wireless
Notehub projects, devices, routes, monitors, and encrypted communications.

Available Tools:
  - project_list: List all Notehub projects
  - project_create: Create a new Notehub project
  - device_list: List devices in a project (with optional tag filtering)
  - device_health_log: Get device health information
  - device_public_key: Get device public key for encryption
  - project_events: List events in a project
  - check_notefiles: Check for Notefiles sent to Notehub by filtering project
    events
  - route_list: List routes in a project
  - route_detail: Get detailed route information
  - monitor_list: List monitors in a project
  - monitor_detail: Get detailed monitor information
  - send_note: Send a note to a device
  - send_encrypted_note: Send an encrypted note using device public key
  - billing_account_list: List all billing accounts
  - product_create: Create a new product in a project
  - product_list: List all products in a project
  - environment_variables_set: Set environment variables at device, fleet,
    or project scope
  - fleet_list: List all fleets in a project
  - fleet_get: Get detailed information about a specific fleet
  - device_dfu_history: Get device DFU history for host or notecard firmware
  - firmware_host_upload: Upload host firmware binary to a project

Authentication:

    Uses session token authentication via NOTEHUB_USER and NOTEHUB_PASS environment variables.
    Tokens are automatically refreshed as needed.

API Base URL: https://api.notefile.net

Example .env file:

    NOTEHUB_USER=your@email.com
    NOTEHUB_PASS=your_password

FUNCTIONS

func CreateBillingAccountListTool() mcp.Tool
    CreateBillingAccountListTool creates a tool for listing billing accounts

func CreateCheckNotefilesTool() mcp.Tool
    CreateCheckNotefilesTool creates a tool for checking Notefiles sent to
    Notehub

func CreateDeviceDfuHistoryTool() mcp.Tool
    CreateDeviceDfuHistoryTool creates a tool for getting device DFU history

func CreateDeviceHealthLogTool() mcp.Tool
    CreateDeviceHealthLogTool creates a tool for getting device health log
    information

func CreateDeviceListTool() mcp.Tool
    CreateDeviceListTool creates a tool for listing devices in a Notehub
    project.

    This tool provides access to the Notehub API endpoint: GET
    /v1/projects/{projectUID}/devices

    Parameters:
      - project_uid (required): The UID of the project to list devices for
      - pageSize (optional): Number of devices to return per page (default 50)
      - pageNum (optional): Page number of results (must be >= 1, default 1)
      - deviceUID (optional): Array of specific device UIDs to filter by
      - tags (optional): Array of tags to filter devices by
      - serialNumber (optional): Array of serial numbers to filter devices by

    Example usage:
      - List all devices: {"project_uid": "app:123..."}
      - Filter by tags: {"project_uid": "app:123...", "tags": ["production",
        "sensor"]}
      - Filter by device UIDs: {"project_uid": "app:123...", "deviceUID":
        ["dev:123...", "dev:456..."]}
      - Paginated results: {"project_uid": "app:123...", "pageSize": 25,
        "pageNum": 2}

    Returns:

        JSON array of device objects with their metadata and status information.

func CreateDevicePublicKeyTool() mcp.Tool
    CreateDevicePublicKeyTool creates a tool for getting device public key
    information

func CreateEnvironmentVariablesSetTool() mcp.Tool
    CreateEnvironmentVariablesSetTool creates a tool for setting environment
    variables at device, fleet, or project scope

func CreateFirmwareHostUploadTool() mcp.Tool
    CreateFirmwareHostUploadTool creates a tool for uploading host firmware
    binary to Notehub

func CreateFleetGetTool() mcp.Tool
    CreateFleetGetTool creates a tool for getting detailed information about a
    specific fleet

func CreateFleetListTool() mcp.Tool
    CreateFleetListTool creates a tool for listing fleets in a Notehub project

func CreateMonitorDetailTool() mcp.Tool
    CreateMonitorDetailTool creates a tool for getting detailed information
    about a specific monitor

func CreateMonitorListTool() mcp.Tool
    CreateMonitorListTool creates a tool for listing monitors in a Notehub
    project

func CreateProductCreateTool() mcp.Tool
    CreateProductCreateTool creates a tool for creating a new product in a
    Notehub project

func CreateProductListTool() mcp.Tool
    CreateProductListTool creates a tool for listing products in a Notehub
    project

func CreateProjectCreateTool() mcp.Tool
    CreateProjectCreateTool creates a tool for creating a new Notehub project

func CreateProjectDetailTool() mcp.Tool
    CreateProjectDetailTool creates a tool for getting detailed information
    about a specific project

func CreateProjectEventsTool() mcp.Tool
    CreateProjectEventsTool creates a tool for listing events in a Notehub
    project

func CreateProjectListTool() mcp.Tool
    CreateProjectListTool creates a tool for listing Notehub projects

func CreateRouteDetailTool() mcp.Tool
    CreateRouteDetailTool creates a tool for getting detailed information about
    a specific route

func CreateRouteListTool() mcp.Tool
    CreateRouteListTool creates a tool for listing routes in a Notehub project

func CreateSendEncryptedNoteTool() mcp.Tool
    CreateSendEncryptedNoteTool creates a tool for sending encrypted notes to
    devices

func CreateSendNoteTool() mcp.Tool
    CreateSendNoteTool creates a tool for sending a note to a device

