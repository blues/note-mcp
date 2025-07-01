package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// HandleProjectListTool handles listing Notehub projects
func HandleProjectListTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if SessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	response, err := MakeNotehubAPIRequest("GET", "/v1/projects", nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list projects: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}

// HandleProjectCreateTool handles creating a new Notehub project
func HandleProjectCreateTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if SessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	projectName, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid name parameter: %v", err)), nil
	}

	billingAccountUID, err := request.RequireString("billing_account_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid billing_account_uid parameter: %v", err)), nil
	}

	projectDescription := request.GetString("description", "")

	projectPayload := map[string]interface{}{
		"label":               projectName,
		"billing_account_uid": billingAccountUID,
	}

	if projectDescription != "" {
		projectPayload["description"] = projectDescription
	}

	payloadBytes, err := json.Marshal(projectPayload)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal project payload: %v", err)), nil
	}

	response, err := MakeNotehubAPIRequest("POST", "/v1/projects", payloadBytes)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create project: %v", err)), nil
	}

	var responseData map[string]interface{}
	if err := json.Unmarshal([]byte(response), &responseData); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse project creation response: %v", err)), nil
	}

	if uid, exists := responseData["uid"]; exists {
		projectURL := fmt.Sprintf("https://notehub.io/project/%s", uid)
		responseData["project_url"] = projectURL
	} else {
		return mcp.NewToolResultError("Failed to extract project UID from response"), nil
	}

	modifiedResponse, err := json.Marshal(responseData)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal modified response: %v", err)), nil
	}

	return mcp.NewToolResultText(string(modifiedResponse)), nil
}

// HandleProjectDetailTool handles getting detailed information about a specific project
func HandleProjectDetailTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if SessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	projectUID, err := request.RequireString("project_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid project_uid parameter: %v", err)), nil
	}

	endpoint := fmt.Sprintf("/v1/projects/%s", projectUID)
	response, err := MakeNotehubAPIRequest("GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get project details: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}

// HandleDeviceListTool handles listing devices in a Notehub project with optional filtering and pagination
func HandleDeviceListTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if SessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	projectUID, err := request.RequireString("project_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid project_uid parameter: %v", err)), nil
	}

	// Extract optional parameters
	pageSize := request.GetInt("pageSize", 0)
	pageNum := request.GetInt("pageNum", 0)
	deviceUIDs := request.GetStringSlice("deviceUID", []string{})
	tags := request.GetStringSlice("tags", []string{})
	serialNumbers := request.GetStringSlice("serialNumber", []string{})
	fleetUIDs := request.GetStringSlice("fleetUID", []string{})
	notecardFirmware := request.GetString("notecardFirmware", "")
	location := request.GetString("location", "")
	hostFirmware := request.GetString("hostFirmware", "")
	productUIDs := request.GetStringSlice("productUID", []string{})
	skus := request.GetStringSlice("sku", []string{})

	params := url.Values{}

	if pageSize > 0 {
		params.Add("pageSize", strconv.Itoa(pageSize))
	}
	if pageNum > 0 {
		params.Add("pageNum", strconv.Itoa(pageNum))
	}

	// Add device UID filters
	for _, deviceUID := range deviceUIDs {
		if deviceUID := strings.TrimSpace(deviceUID); deviceUID != "" {
			params.Add("deviceUID", deviceUID)
		}
	}

	// Add tag filters
	for _, tag := range tags {
		if tag := strings.TrimSpace(tag); tag != "" {
			params.Add("tag", tag)
		}
	}

	// Add serial number filters
	for _, serialNumber := range serialNumbers {
		if serialNumber := strings.TrimSpace(serialNumber); serialNumber != "" {
			params.Add("serialNumber", serialNumber)
		}
	}

	// Add fleet UID filters
	for _, fleetUID := range fleetUIDs {
		if fleetUID := strings.TrimSpace(fleetUID); fleetUID != "" {
			params.Add("fleetUID", fleetUID)
		}
	}

	// Add additional filters
	if notecardFirmware != "" {
		params.Add("notecardFirmware", notecardFirmware)
	}
	if location != "" {
		params.Add("location", location)
	}
	if hostFirmware != "" {
		params.Add("hostFirmware", hostFirmware)
	}

	// Add product UID filters
	for _, productUID := range productUIDs {
		if productUID := strings.TrimSpace(productUID); productUID != "" {
			params.Add("productUID", productUID)
		}
	}

	// Add SKU filters
	for _, sku := range skus {
		if sku := strings.TrimSpace(sku); sku != "" {
			params.Add("sku", sku)
		}
	}

	queryString := ""
	if len(params) > 0 {
		queryString = "?" + params.Encode()
	}

	endpoint := fmt.Sprintf("/v1/projects/%s/devices%s", projectUID, queryString)
	response, err := MakeNotehubAPIRequest("GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list devices: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}

// HandleProjectEventsTool handles listing events in a Notehub project
func HandleProjectEventsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if SessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	projectUID, err := request.RequireString("project_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid project_uid parameter: %v", err)), nil
	}

	// Extract optional parameters
	pageSize := request.GetInt("pageSize", 0)
	pageNum := request.GetInt("pageNum", 0)
	deviceUIDs := request.GetStringSlice("deviceUID", []string{})
	sortBy := request.GetString("sortBy", "")
	sortOrder := request.GetString("sortOrder", "")
	startDate := request.GetString("startDate", "")
	endDate := request.GetString("endDate", "")
	dateType := request.GetString("dateType", "")
	systemFilesOnly := request.GetString("systemFilesOnly", "")
	files := request.GetString("files", "")
	format := request.GetString("format", "")
	serialNumbers := request.GetStringSlice("serialNumber", []string{})
	fleetUIDs := request.GetStringSlice("fleetUID", []string{})
	sessionUIDs := request.GetStringSlice("sessionUID", []string{})
	eventUIDs := request.GetStringSlice("eventUID", []string{})
	selectFields := request.GetString("selectFields", "")

	params := url.Values{}

	if pageSize > 0 {
		params.Add("pageSize", strconv.Itoa(pageSize))
	}
	if pageNum > 0 {
		params.Add("pageNum", strconv.Itoa(pageNum))
	}

	// Add device UID filters
	for _, deviceUID := range deviceUIDs {
		if deviceUID := strings.TrimSpace(deviceUID); deviceUID != "" {
			params.Add("deviceUID", deviceUID)
		}
	}

	if sortBy != "" {
		params.Add("sortBy", sortBy)
	}
	if sortOrder != "" {
		params.Add("sortOrder", sortOrder)
	}
	if startDate != "" {
		params.Add("startDate", startDate)
	}
	if endDate != "" {
		params.Add("endDate", endDate)
	}
	if dateType != "" {
		params.Add("dateType", dateType)
	}
	if systemFilesOnly != "" {
		params.Add("systemFilesOnly", systemFilesOnly)
	}
	if files != "" {
		params.Add("files", files)
	}
	if format != "" {
		params.Add("format", format)
	}

	// Add serial number filters
	for _, serialNumber := range serialNumbers {
		if serialNumber := strings.TrimSpace(serialNumber); serialNumber != "" {
			params.Add("serialNumber", serialNumber)
		}
	}

	// Add fleet UID filters
	for _, fleetUID := range fleetUIDs {
		if fleetUID := strings.TrimSpace(fleetUID); fleetUID != "" {
			params.Add("fleetUID", fleetUID)
		}
	}

	// Add session UID filters
	for _, sessionUID := range sessionUIDs {
		if sessionUID := strings.TrimSpace(sessionUID); sessionUID != "" {
			params.Add("sessionUID", sessionUID)
		}
	}

	// Add event UID filters
	for _, eventUID := range eventUIDs {
		if eventUID := strings.TrimSpace(eventUID); eventUID != "" {
			params.Add("eventUID", eventUID)
		}
	}

	if selectFields != "" {
		params.Add("selectFields", selectFields)
	}

	queryString := ""
	if len(params) > 0 {
		queryString = "?" + params.Encode()
	}

	endpoint := fmt.Sprintf("/v1/projects/%s/events%s", projectUID, queryString)
	response, err := MakeNotehubAPIRequest("GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list events: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}

// HandleSendNoteTool handles sending a note to a device
func HandleSendNoteTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if SessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	projectUID, err := request.RequireString("project_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid project_uid parameter: %v", err)), nil
	}

	deviceUID, err := request.RequireString("device_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid device_uid parameter: %v", err)), nil
	}

	noteFile, err := request.RequireString("note_file")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid note_file parameter: %v", err)), nil
	}

	noteBody, err := request.RequireString("note_body")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid note_body parameter: %v", err)), nil
	}

	notePayload := map[string]interface{}{
		"body": json.RawMessage(noteBody),
	}

	payloadBytes, err := json.Marshal(notePayload)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal note payload: %v", err)), nil
	}

	endpoint := fmt.Sprintf("/v1/projects/%s/devices/%s/notes/%s", projectUID, deviceUID, noteFile)
	response, err := MakeNotehubAPIRequest("POST", endpoint, payloadBytes)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send note: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}

// HandleRouteListTool handles listing routes in a Notehub project
func HandleRouteListTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if SessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	projectUID, err := request.RequireString("project_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid project_uid parameter: %v", err)), nil
	}

	endpoint := fmt.Sprintf("/v1/projects/%s/routes", projectUID)
	response, err := MakeNotehubAPIRequest("GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list routes: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}

// HandleRouteDetailTool handles getting detailed information about a specific route
func HandleRouteDetailTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if SessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	projectUID, err := request.RequireString("project_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid project_uid parameter: %v", err)), nil
	}

	routeUID, err := request.RequireString("route_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid route_uid parameter: %v", err)), nil
	}

	endpoint := fmt.Sprintf("/v1/projects/%s/routes/%s", projectUID, routeUID)
	response, err := MakeNotehubAPIRequest("GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get route details: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}

// HandleDeviceHealthLogTool handles getting device health log information
func HandleDeviceHealthLogTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if SessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	projectUID, err := request.RequireString("project_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid project_uid parameter: %v", err)), nil
	}

	deviceUID, err := request.RequireString("device_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid device_uid parameter: %v", err)), nil
	}

	endpoint := fmt.Sprintf("/v1/projects/%s/devices/%s/health-log", projectUID, deviceUID)
	response, err := MakeNotehubAPIRequest("GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get device health log: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}

// HandleMonitorListTool handles listing monitors in a Notehub project
func HandleMonitorListTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if SessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	projectUID, err := request.RequireString("project_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid project_uid parameter: %v", err)), nil
	}

	endpoint := fmt.Sprintf("/v1/projects/%s/monitors", projectUID)
	response, err := MakeNotehubAPIRequest("GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list monitors: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}

// HandleMonitorDetailTool handles getting detailed information about a specific monitor
func HandleMonitorDetailTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if SessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	projectUID, err := request.RequireString("project_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid project_uid parameter: %v", err)), nil
	}

	monitorUID, err := request.RequireString("monitor_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid monitor_uid parameter: %v", err)), nil
	}

	endpoint := fmt.Sprintf("/v1/projects/%s/monitors/%s", projectUID, monitorUID)
	response, err := MakeNotehubAPIRequest("GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get monitor details: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}

// HandleDevicePublicKeyTool handles getting device public key information
func HandleDevicePublicKeyTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if SessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	projectUID, err := request.RequireString("project_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid project_uid parameter: %v", err)), nil
	}

	deviceUID, err := request.RequireString("device_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid device_uid parameter: %v", err)), nil
	}

	endpoint := fmt.Sprintf("/v1/projects/%s/devices/%s/public-key", projectUID, deviceUID)
	response, err := MakeNotehubAPIRequest("GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get device public key: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}

// HandleSendEncryptedNoteTool handles sending encrypted notes to devices
func HandleSendEncryptedNoteTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if SessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	projectUID, err := request.RequireString("project_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid project_uid parameter: %v", err)), nil
	}

	deviceUID, err := request.RequireString("device_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid device_uid parameter: %v", err)), nil
	}

	noteFile, err := request.RequireString("note_file")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid note_file parameter: %v", err)), nil
	}

	plaintextMessage, err := request.RequireString("plaintext_message")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid plaintext_message parameter: %v", err)), nil
	}

	// Get the device's public key
	publicKeyEndpoint := fmt.Sprintf("/v1/projects/%s/devices/%s/public-key", projectUID, deviceUID)
	publicKeyResponse, err := MakeNotehubAPIRequest("GET", publicKeyEndpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get device public key: %v", err)), nil
	}

	// Parse the public key response
	var keyData struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal([]byte(publicKeyResponse), &keyData); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse public key response: %v", err)), nil
	}

	// Encrypt the message
	encryptedNote, err := EncryptMessage(keyData.Key, []byte(plaintextMessage))
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to encrypt message: %v", err)), nil
	}

	// Create the encrypted note payload
	notePayload := map[string]interface{}{
		"body": encryptedNote,
	}

	payloadBytes, err := json.Marshal(notePayload)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal encrypted note payload: %v", err)), nil
	}

	sendEndpoint := fmt.Sprintf("/v1/projects/%s/devices/%s/notes/%s", projectUID, deviceUID, noteFile)
	response, err := MakeNotehubAPIRequest("POST", sendEndpoint, payloadBytes)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send encrypted note: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Encrypted note sent successfully. Response: %s", response)), nil
}

// HandleBillingAccountListTool handles listing billing accounts
func HandleBillingAccountListTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if SessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	response, err := MakeNotehubAPIRequest("GET", "/v1/billing-accounts", nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list billing accounts: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}

// HandleProductCreateTool handles creating a new product in a Notehub project
func HandleProductCreateTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if SessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	projectUID, err := request.RequireString("project_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid project_uid parameter: %v", err)), nil
	}

	productUID, err := request.RequireString("product_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid product_uid parameter: %v", err)), nil
	}

	label, err := request.RequireString("label")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid label parameter: %v", err)), nil
	}

	autoProvisionFleets := request.GetString("auto_provision_fleets", "")
	disableDevicesByDefault := request.GetString("disable_devices_by_default", "")

	productPayload := map[string]interface{}{
		"product_uid": productUID,
		"label":       label,
	}

	if autoProvisionFleets != "" {
		productPayload["auto_provision_fleets"] = autoProvisionFleets
	}

	if disableDevicesByDefault != "" {
		productPayload["disable_devices_by_default"] = disableDevicesByDefault
	}

	payloadBytes, err := json.Marshal(productPayload)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal product payload: %v", err)), nil
	}

	endpoint := fmt.Sprintf("/v1/projects/%s/products", projectUID)
	response, err := MakeNotehubAPIRequest("POST", endpoint, payloadBytes)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create product: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}

// HandleProductListTool handles listing products in a Notehub project
func HandleProductListTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if SessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	projectUID, err := request.RequireString("project_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid project_uid parameter: %v", err)), nil
	}

	endpoint := fmt.Sprintf("/v1/projects/%s/products", projectUID)
	response, err := MakeNotehubAPIRequest("GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list products: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}

// HandleEnvironmentVariablesSetTool handles setting environment variables at device, fleet, or project scope
func HandleEnvironmentVariablesSetTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if SessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	projectUID, err := request.RequireString("project_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid project_uid parameter: %v", err)), nil
	}

	scope, err := request.RequireString("scope")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid scope parameter: %v", err)), nil
	}

	environmentVariables, err := request.RequireString("environment_variables")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid environment_variables parameter: %v", err)), nil
	}

	var endpoint string

	switch scope {
	case "device":
		uid, err := request.RequireString("uid")
		if err != nil {
			return mcp.NewToolResultError("uid is required when scope is 'device'"), nil
		}
		endpoint = fmt.Sprintf("/v1/projects/%s/devices/%s/environment_variables", projectUID, uid)
	case "fleet":
		uid, err := request.RequireString("uid")
		if err != nil {
			return mcp.NewToolResultError("uid is required when scope is 'fleet'"), nil
		}
		endpoint = fmt.Sprintf("/v1/projects/%s/fleets/%s/environment_variables", projectUID, uid)
	case "project":
		endpoint = fmt.Sprintf("/v1/projects/%s/environment_variables", projectUID)
	default:
		return mcp.NewToolResultError("Invalid scope. Must be 'device', 'fleet', or 'project'"), nil
	}

	var envVars map[string]interface{}
	if err := json.Unmarshal([]byte(environmentVariables), &envVars); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid environment_variables JSON: %v", err)), nil
	}

	payload := map[string]interface{}{
		"environment_variables": envVars,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal environment variables payload: %v", err)), nil
	}

	response, err := MakeNotehubAPIRequest("PUT", endpoint, payloadBytes)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to set environment variables: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}

// HandleFleetListTool handles listing fleets in a Notehub project
func HandleFleetListTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if SessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	projectUID, err := request.RequireString("project_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid project_uid parameter: %v", err)), nil
	}

	endpoint := fmt.Sprintf("/v1/projects/%s/fleets", projectUID)
	response, err := MakeNotehubAPIRequest("GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list fleets: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}

// HandleFleetGetTool handles getting detailed information about a specific fleet
func HandleFleetGetTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if SessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	projectUID, err := request.RequireString("project_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid project_uid parameter: %v", err)), nil
	}

	fleetUID, err := request.RequireString("fleet_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid fleet_uid parameter: %v", err)), nil
	}

	endpoint := fmt.Sprintf("/v1/projects/%s/fleets/%s", projectUID, fleetUID)
	response, err := MakeNotehubAPIRequest("GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get fleet details: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}

// HandleDeviceDfuHistoryTool handles getting device DFU history
func HandleDeviceDfuHistoryTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if SessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	projectUID, err := request.RequireString("project_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid project_uid parameter: %v", err)), nil
	}

	deviceUID, err := request.RequireString("device_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid device_uid parameter: %v", err)), nil
	}

	firmwareType, err := request.RequireString("firmware_type")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid firmware_type parameter: %v", err)), nil
	}

	if firmwareType != "host" && firmwareType != "notecard" {
		return mcp.NewToolResultError("Invalid firmware_type. Must be 'host' or 'notecard'"), nil
	}

	endpoint := fmt.Sprintf("/v1/projects/%s/devices/%s/dfu/%s/history", projectUID, deviceUID, firmwareType)
	response, err := MakeNotehubAPIRequest("GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get device DFU history: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}

// HandleFirmwareHostUploadTool handles uploading host firmware binary to Notehub
func HandleFirmwareHostUploadTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if SessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	projectUID, err := request.RequireString("project_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid project_uid parameter: %v", err)), nil
	}

	filePath, err := request.RequireString("file_path")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid file_path parameter: %v", err)), nil
	}

	// Extract optional parameters
	org := request.GetString("org", "")
	product := request.GetString("product", "")
	firmware := request.GetString("firmware", "")
	version := request.GetString("version", "")
	target := request.GetString("target", "")
	versionString := request.GetString("version_string", "")
	built := request.GetString("built", "")
	builder := request.GetString("builder", "")

	// Parse version string into components if provided
	var verMajor, verMinor, verPatch int32
	if versionString != "" {
		parts := strings.Split(versionString, ".")
		if len(parts) != 3 {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid version_string format '%s'. Expected format: 'major.minor.patch' (e.g., '1.2.3')", versionString)), nil
		}

		versions := []*int32{&verMajor, &verMinor, &verPatch}
		versionNames := []string{"major", "minor", "patch"}

		for i, part := range parts {
			val, err := strconv.Atoi(part)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Invalid %s version '%s' in version_string '%s'. Must be a number", versionNames[i], part, versionString)), nil
			}
			*versions[i] = int32(val)
		}
	}

	// Read the file from the filesystem
	binaryBytes, err := os.ReadFile(filePath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to read file from path '%s': %v", filePath, err)), nil
	}

	filename := filepath.Base(filePath)

	endpoint := fmt.Sprintf("/v1/projects/%s/firmware/host/%s", projectUID, filename)

	// Add query parameters if provided
	queryParams := []string{}

	if org != "" {
		queryParams = append(queryParams, fmt.Sprintf("org=%s", url.QueryEscape(org)))
	}
	if product != "" {
		queryParams = append(queryParams, fmt.Sprintf("product=%s", url.QueryEscape(product)))
	}
	if firmware != "" {
		queryParams = append(queryParams, fmt.Sprintf("firmware=%s", url.QueryEscape(firmware)))
	}
	if version != "" {
		queryParams = append(queryParams, fmt.Sprintf("version=%s", url.QueryEscape(version)))
	}
	if target != "" {
		queryParams = append(queryParams, fmt.Sprintf("target=%s", url.QueryEscape(target)))
	}
	if versionString != "" {
		queryParams = append(queryParams, fmt.Sprintf("ver_major=%d", verMajor))
		queryParams = append(queryParams, fmt.Sprintf("ver_minor=%d", verMinor))
		queryParams = append(queryParams, fmt.Sprintf("ver_patch=%d", verPatch))
	}
	if built != "" {
		queryParams = append(queryParams, fmt.Sprintf("built=%s", url.QueryEscape(built)))
	}
	if builder != "" {
		queryParams = append(queryParams, fmt.Sprintf("builder=%s", url.QueryEscape(builder)))
	}

	// Append query parameters to endpoint if any exist
	if len(queryParams) > 0 {
		endpoint += "?" + strings.Join(queryParams, "&")
	}

	response, err := MakeNotehubAPIRequest("PUT", endpoint, binaryBytes)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to upload host firmware: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}

// HandleCheckNotefilesTool handles checking for Notefiles sent to Notehub
func HandleCheckNotefilesTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if SessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	projectUID, err := request.RequireString("project_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid project_uid parameter: %v", err)), nil
	}

	deviceUID := request.GetString("device_uid", "")

	// Get project events
	endpoint := fmt.Sprintf("/v1/projects/%s/events", projectUID)
	response, err := MakeNotehubAPIRequest("GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get project events: %v", err)), nil
	}

	// Parse the response to filter for Notefiles
	var eventsData struct {
		Events []map[string]interface{} `json:"events"`
	}
	if err := json.Unmarshal([]byte(response), &eventsData); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to parse events response: %v", err)), nil
	}

	// Filter events to show only Notefile-related events
	var notefileEvents []map[string]interface{}
	for _, event := range eventsData.Events {
		// Check if this event has a file field (indicates a Notefile)
		if file, exists := event["file"]; exists && file != nil {
			fileStr, ok := file.(string)
			if !ok {
				continue
			}

			// Filter by device if specified
			if deviceUID != "" {
				if eventDeviceUID, exists := event["device_uid"]; exists {
					if eventDeviceUID != deviceUID {
						continue
					}
				}
			}

			// Check if it's a Notefile pattern (.qo, .dbo, etc.)
			if strings.HasSuffix(fileStr, ".qo") || strings.HasSuffix(fileStr, ".dbo") ||
				strings.HasSuffix(fileStr, ".qi") || strings.HasSuffix(fileStr, ".dbi") ||
				strings.Contains(fileStr, "_session") || strings.Contains(fileStr, "_health") {
				notefileEvents = append(notefileEvents, event)
			}
		}
	}

	// Create a summary response
	var result struct {
		Summary         string                   `json:"summary"`
		TotalEvents     int                      `json:"total_events"`
		NotefileEvents  int                      `json:"notefile_events"`
		ProjectUID      string                   `json:"project_uid"`
		DeviceUID       string                   `json:"device_uid,omitempty"`
		Events          []map[string]interface{} `json:"events"`
		CommonFiles     map[string]int           `json:"common_files"`
		Troubleshooting []string                 `json:"troubleshooting"`
	}

	result.ProjectUID = projectUID
	if deviceUID != "" {
		result.DeviceUID = deviceUID
	}
	result.TotalEvents = len(eventsData.Events)
	result.NotefileEvents = len(notefileEvents)
	result.Events = notefileEvents

	// Count common file types
	result.CommonFiles = make(map[string]int)
	for _, event := range notefileEvents {
		if file, exists := event["file"]; exists {
			if fileStr, ok := file.(string); ok {
				result.CommonFiles[fileStr]++
			}
		}
	}

	// Create summary
	if len(notefileEvents) == 0 {
		result.Summary = "No Notefiles found in recent events. This could mean your Notecard hasn't sent data yet, or the data was sent earlier than the current event window."
		result.Troubleshooting = []string{
			"Check if your Notecard is properly provisioned and connected",
			"Verify the Notecard has synced with Notehub (check hub.status)",
			"Ensure your Notecard application is sending data to Notefiles",
			"Check device health logs for connection issues",
			"Verify your Notecard has the correct project UID configured",
		}
	} else {
		deviceText := ""
		if deviceUID != "" {
			deviceText = fmt.Sprintf(" from device %s", deviceUID)
		}
		result.Summary = fmt.Sprintf("Found %d Notefile events%s out of %d total events in project %s",
			len(notefileEvents), deviceText, len(eventsData.Events), projectUID)
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal result: %v", err)), nil
	}

	return mcp.NewToolResultText(string(resultBytes)), nil
}
