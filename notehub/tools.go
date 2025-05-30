package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/mcp"
)

type NotehubCredentials struct {
	Username string
	Password string
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	SessionToken string `json:"session_token"`
}

// GetNotehubCredentials loads credentials from .env file
func GetNotehubCredentials(envFilePath string) (NotehubCredentials, error) {
	envFile, err := godotenv.Read(envFilePath)
	if err != nil {
		return NotehubCredentials{}, fmt.Errorf("failed to read .env file: %w", err)
	}

	envFileUsername := envFile["NOTEHUB_USER"]
	envFilePassword := envFile["NOTEHUB_PASS"]

	if envFileUsername == "" {
		return NotehubCredentials{}, fmt.Errorf("NOTEHUB_USER not found in .env file")
	}

	if envFilePassword == "" {
		return NotehubCredentials{}, fmt.Errorf("NOTEHUB_PASS not found in .env file")
	}

	return NotehubCredentials{
		Username: envFileUsername,
		Password: envFilePassword,
	}, nil
}

// CreateSessionToken creates a session token using username and password
func CreateSessionToken(username, password string) (string, error) {
	loginReq := LoginRequest{
		Username: username,
		Password: password,
	}

	jsonData, err := json.Marshal(loginReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal login request: %w", err)
	}

	resp, err := http.Post("https://api.notefile.net/auth/login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to make login request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var loginResp LoginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return loginResp.SessionToken, nil
}

// makeNotehubAPIRequest makes an authenticated request to the Notehub API
func makeNotehubAPIRequest(method, endpoint string, body []byte) (string, error) {
	baseURL := "https://api.notefile.net"
	url := baseURL + endpoint

	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add authentication header
	req.Header.Set("X-SESSION-TOKEN", sessionToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Sprintf("Request failed with status %d: %s", resp.StatusCode, string(responseBody)), nil
	}

	return string(responseBody), nil
}

// CreateProjectListTool creates a tool for listing Notehub projects
func CreateProjectListTool() mcp.Tool {
	return mcp.NewTool("project_list",
		mcp.WithDescription("List all Notehub projects belonging to the authenticated user"),
	)
}

// HandleProjectListTool handles listing Notehub projects
func HandleProjectListTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if sessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	// Make the API request to list projects
	response, err := makeNotehubAPIRequest("GET", "/v1/projects", nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list projects: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}

// CreateDeviceListTool creates a tool for listing devices in a Notehub project
func CreateDeviceListTool() mcp.Tool {
	return mcp.NewTool("device_list",
		mcp.WithDescription("List all devices in a specific Notehub project"),
		mcp.WithString("project_uid",
			mcp.Required(),
			mcp.Description("The UID of the project to list devices for"),
		),
	)
}

// HandleDeviceListTool handles listing devices in a Notehub project
func HandleDeviceListTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if sessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	// Extract project_uid from arguments
	projectUID, err := request.RequireString("project_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid project_uid parameter: %v", err)), nil
	}

	// Make the API request to list devices for the project
	endpoint := fmt.Sprintf("/v1/projects/%s/devices", projectUID)
	response, err := makeNotehubAPIRequest("GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list devices: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
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

// HandleProjectEventsTool handles listing events in a Notehub project
func HandleProjectEventsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if sessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	// Extract project_uid from arguments
	projectUID, err := request.RequireString("project_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid project_uid parameter: %v", err)), nil
	}

	// Make the API request to list events for the project
	endpoint := fmt.Sprintf("/v1/projects/%s/events", projectUID)
	response, err := makeNotehubAPIRequest("GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list events: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
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
			mcp.Description("The note file name (e.g., 'data.qi', 'config.db')"),
		),
		mcp.WithString("note_body",
			mcp.Required(),
			mcp.Description("The JSON body content of the note"),
		),
	)
}

// HandleSendNoteTool handles sending a note to a device
func HandleSendNoteTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if sessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	// Extract parameters from arguments
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

	// Construct the note payload
	notePayload := map[string]interface{}{
		"body": json.RawMessage(noteBody),
	}

	payloadBytes, err := json.Marshal(notePayload)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal note payload: %v", err)), nil
	}

	// Make the API request to send the note
	endpoint := fmt.Sprintf("/v1/projects/%s/devices/%s/notes/%s", projectUID, deviceUID, noteFile)
	response, err := makeNotehubAPIRequest("POST", endpoint, payloadBytes)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send note: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}
