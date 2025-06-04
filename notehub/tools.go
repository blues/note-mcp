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
//   - route_list: List routes in a project
//   - route_detail: Get detailed route information
//   - monitor_list: List monitors in a project
//   - monitor_detail: Get detailed monitor information
//   - send_note: Send a note to a device
//   - send_encrypted_note: Send an encrypted note using device public key
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
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"strings"

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

// CreateProjectCreateTool creates a tool for creating a new Notehub project
func CreateProjectCreateTool() mcp.Tool {
	return mcp.NewTool("project_create",
		mcp.WithDescription("Create a new Notehub project"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("The name of the project to create"),
		),
		mcp.WithString("description",
			mcp.Description("Optional description for the project"),
		),
	)
}

// HandleProjectCreateTool handles creating a new Notehub project
func HandleProjectCreateTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if sessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	projectName, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid name parameter: %v", err)), nil
	}

	projectDescription := request.GetString("description", "")

	// Create the project payload
	projectPayload := map[string]interface{}{
		"name": projectName,
	}

	// Add description if provided
	if projectDescription != "" {
		projectPayload["description"] = projectDescription
	}

	payloadBytes, err := json.Marshal(projectPayload)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to marshal project payload: %v", err)), nil
	}

	response, err := makeNotehubAPIRequest("POST", "/v1/projects", payloadBytes)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to create project: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}

// CreateDeviceListTool creates a tool for listing devices in a Notehub project.
//
// This tool provides access to the Notehub API endpoint:
// GET /v1/projects/{projectUID}/devices[?tag=tag1&tag=tag2...]
//
// Parameters:
//   - project_uid (required): The UID of the project to list devices for
//   - tags (optional): Array of tags to filter devices by
//
// Example usage:
//   - List all devices: {"project_uid": "app:123..."}
//   - Filter by tags: {"project_uid": "app:123...", "tags": ["production", "sensor"]}
//
// Returns:
//
//	JSON array of device objects with their metadata and status information.
func CreateDeviceListTool() mcp.Tool {
	return mcp.NewTool("device_list",
		mcp.WithDescription("List all devices in a specific Notehub project, optionally filtered by tags"),
		mcp.WithString("project_uid",
			mcp.Required(),
			mcp.Description("The UID of the project to list devices for (format: app:xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)"),
		),
		mcp.WithArray("tags",
			mcp.Description("Optional array of tags to filter devices by. Example: ['production', 'sensor', 'outdoor']"),
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

	// Extract tags from arguments (optional)
	tags := request.GetStringSlice("tags", []string{})

	// Build query parameters
	queryParams := ""
	if len(tags) > 0 {
		// Build query parameters directly from the string slice
		for i, tag := range tags {
			if i == 0 {
				queryParams = fmt.Sprintf("?tag=%s", strings.TrimSpace(tag))
			} else {
				queryParams += fmt.Sprintf("&tag=%s", strings.TrimSpace(tag))
			}
		}
	}

	// Make the API request to list devices for the project
	endpoint := fmt.Sprintf("/v1/projects/%s/devices%s", projectUID, queryParams)
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

// HandleRouteListTool handles listing routes in a Notehub project
func HandleRouteListTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if sessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	// Extract project_uid from arguments
	projectUID, err := request.RequireString("project_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid project_uid parameter: %v", err)), nil
	}

	// Make the API request to list routes for the project
	endpoint := fmt.Sprintf("/v1/projects/%s/routes", projectUID)
	response, err := makeNotehubAPIRequest("GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list routes: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
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

// HandleRouteDetailTool handles getting detailed information about a specific route
func HandleRouteDetailTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if sessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	// Extract project_uid from arguments
	projectUID, err := request.RequireString("project_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid project_uid parameter: %v", err)), nil
	}

	// Extract route_uid from arguments
	routeUID, err := request.RequireString("route_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid route_uid parameter: %v", err)), nil
	}

	// Make the API request to get detailed route information
	endpoint := fmt.Sprintf("/v1/projects/%s/routes/%s", projectUID, routeUID)
	response, err := makeNotehubAPIRequest("GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get route details: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
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

// HandleDeviceHealthLogTool handles getting device health log information
func HandleDeviceHealthLogTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if sessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	// Extract project_uid from arguments
	projectUID, err := request.RequireString("project_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid project_uid parameter: %v", err)), nil
	}

	// Extract device_uid from arguments
	deviceUID, err := request.RequireString("device_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid device_uid parameter: %v", err)), nil
	}

	// Make the API request to get device health log information
	endpoint := fmt.Sprintf("/v1/projects/%s/devices/%s/health-log", projectUID, deviceUID)
	response, err := makeNotehubAPIRequest("GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get device health log: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
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

// HandleMonitorListTool handles listing monitors in a Notehub project
func HandleMonitorListTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if sessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	// Extract project_uid from arguments
	projectUID, err := request.RequireString("project_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid project_uid parameter: %v", err)), nil
	}

	// Make the API request to list monitors for the project
	endpoint := fmt.Sprintf("/v1/projects/%s/monitors", projectUID)
	response, err := makeNotehubAPIRequest("GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to list monitors: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
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

// HandleMonitorDetailTool handles getting detailed information about a specific monitor
func HandleMonitorDetailTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if sessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	// Extract project_uid from arguments
	projectUID, err := request.RequireString("project_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid project_uid parameter: %v", err)), nil
	}

	// Extract monitor_uid from arguments
	monitorUID, err := request.RequireString("monitor_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid monitor_uid parameter: %v", err)), nil
	}

	// Make the API request to get detailed monitor information
	endpoint := fmt.Sprintf("/v1/projects/%s/monitors/%s", projectUID, monitorUID)
	response, err := makeNotehubAPIRequest("GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get monitor details: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
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

// HandleDevicePublicKeyTool handles getting device public key information
func HandleDevicePublicKeyTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if sessionToken == "" {
		return mcp.NewToolResultError("No session token available. Please refresh token first."), nil
	}

	// Extract project_uid from arguments
	projectUID, err := request.RequireString("project_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid project_uid parameter: %v", err)), nil
	}

	// Extract device_uid from arguments
	deviceUID, err := request.RequireString("device_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid device_uid parameter: %v", err)), nil
	}

	// Make the API request to get device public key information
	endpoint := fmt.Sprintf("/v1/projects/%s/devices/%s/public-key", projectUID, deviceUID)
	response, err := makeNotehubAPIRequest("GET", endpoint, nil)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get device public key: %v", err)), nil
	}

	return mcp.NewToolResultText(response), nil
}

// EncryptedNote represents the encrypted note data in Blues format
type EncryptedNote struct {
	Alg  string `json:"alg"`
	Data string `json:"data"`
	Key  string `json:"key"`
}

// encryptMessage encrypts a message using ECDH key exchange and AES-256-CBC encryption
func encryptMessage(publicKeyPEM string, message []byte) (*EncryptedNote, error) {
	// Parse the PEM-encoded public key
	pemBlock, _ := pem.Decode([]byte(publicKeyPEM))
	if pemBlock == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	// Parse the public key
	publicKeyInterface, err := x509.ParsePKIXPublicKey(pemBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	// Convert to ECDSA public key
	ecdsaPublicKey, ok := publicKeyInterface.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not an ECDSA key")
	}

	// Convert to ECDH public key (P-384 curve)
	curve := ecdh.P384()

	// Generate ephemeral key pair
	ephemeralPrivateKey, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ephemeral key: %w", err)
	}
	ephemeralPublicKey := ephemeralPrivateKey.PublicKey()

	// Convert the recipient's ECDSA public key to ECDH format
	// Concatenate X and Y coordinates with 0x04 prefix (uncompressed point format)
	pubKeyBytes := append([]byte{0x04}, append(ecdsaPublicKey.X.Bytes(), ecdsaPublicKey.Y.Bytes()...)...)
	recipientPublicKey, err := curve.NewPublicKey(pubKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create ECDH public key: %w", err)
	}

	// Perform ECDH key exchange
	sharedSecret, err := ephemeralPrivateKey.ECDH(recipientPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to perform ECDH: %w", err)
	}

	// Derive AES key from shared secret using SHA256
	hash := sha256.Sum256(sharedSecret)
	aesKey := hash[:] // Use full 32-byte hash for AES-256

	// Use zero IV (to match your decryption)
	iv := make([]byte, aes.BlockSize)

	// Apply PKCS#7 padding
	paddingLength := aes.BlockSize - (len(message) % aes.BlockSize)
	padding := bytes.Repeat([]byte{byte(paddingLength)}, paddingLength)
	paddedMessage := append(message, padding...)

	// Encrypt with AES-256-CBC
	cipherBlock, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	ciphertext := make([]byte, len(paddedMessage))
	mode := cipher.NewCBCEncrypter(cipherBlock, iv)
	mode.CryptBlocks(ciphertext, paddedMessage)

	// Convert ephemeral public key to DER format and base64 encode
	ephemeralKeyDER, err := x509.MarshalPKIXPublicKey(ephemeralPublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal ephemeral public key: %w", err)
	}

	return &EncryptedNote{
		Alg:  "secp384r1-aes256cbc",
		Data: base64.StdEncoding.EncodeToString(ciphertext), // Only ciphertext, no IV
		Key:  base64.StdEncoding.EncodeToString(ephemeralKeyDER),
	}, nil
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
			mcp.Description("The note file name, it should always be of *.qi as it is an inbound note (e.g., 'data.qi', 'secret.qi')"),
		),
		mcp.WithString("plaintext_message",
			mcp.Required(),
			mcp.Description("The plaintext message to encrypt and send"),
		),
	)
}

// HandleSendEncryptedNoteTool handles sending encrypted notes to devices
func HandleSendEncryptedNoteTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

	plaintextMessage, err := request.RequireString("plaintext_message")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid plaintext_message parameter: %v", err)), nil
	}

	// First, get the device's public key
	publicKeyEndpoint := fmt.Sprintf("/v1/projects/%s/devices/%s/public-key", projectUID, deviceUID)
	publicKeyResponse, err := makeNotehubAPIRequest("GET", publicKeyEndpoint, nil)
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
	encryptedNote, err := encryptMessage(keyData.Key, []byte(plaintextMessage))
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

	// Send the encrypted note
	sendEndpoint := fmt.Sprintf("/v1/projects/%s/devices/%s/notes/%s", projectUID, deviceUID, noteFile)
	response, err := makeNotehubAPIRequest("POST", sendEndpoint, payloadBytes)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send encrypted note: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Encrypted note sent successfully. Response: %s", response)), nil
}
