package main

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"note-mcp/utils"

	"github.com/blues/note-go/notecard"
	"github.com/mark3labs/mcp-go/mcp"
)

var (
	notecardContext *notecard.Context
	notecardPort    *string
	notecardBaud    *int
)

func initializeNotecard() error {
	// If the notecard is already initialized, return nil
	if notecardContext != nil {
		return nil
	}

	moduleInterface, port, portConfig := notecard.Defaults()

	// If the port is provided, use it
	if notecardPort != nil {
		port = *notecardPort
	}

	// If the baud is provided, use it
	if notecardBaud != nil {
		portConfig = *notecardBaud
	}

	ctx, err := notecard.Open(moduleInterface, port, portConfig)
	if err != nil {
		return fmt.Errorf("failed to open notecard: %w", err)
	}

	// Set up cleanup that runs on any error
	var validationSuccess bool
	defer func() {
		if !validationSuccess {
			ctx.Close()
			notecardContext = nil
			notecardPort = nil
			notecardBaud = nil
		}
	}()

	// Validate that a Notecard device is actually present by sending a card.version request
	testReq := map[string]interface{}{
		"req": "card.version",
	}

	response, err := ctx.Transaction(testReq)
	if err != nil {
		return fmt.Errorf("notecard device not found or not responding on port %s: %w", port, err)
	}

	// Check if the response indicates an error
	if errMsg, exists := response["err"]; exists && errMsg != nil {
		return fmt.Errorf("notecard device validation failed: %v", errMsg)
	}

	// Verify we got a valid version response
	if _, exists := response["version"]; !exists {
		return fmt.Errorf("notecard device validation failed: invalid response format")
	}

	// Mark validation as successful and set the global context
	validationSuccess = true
	notecardContext = ctx
	return nil
}

// closeNotecard closes the connection to the Notecard and cleans up resources
func closeNotecard() {
	if notecardContext != nil {
		notecardContext.Close()
		notecardContext = nil
		notecardPort = nil
		notecardBaud = nil
	}
}

// GetNotecardContext returns the current notecard context for use by other tools
// Returns nil if the notecard is not initialized
func GetNotecardContext() *notecard.Context {
	return notecardContext
}

// IsNotecardInitialized returns true if the notecard context is initialized
func IsNotecardInitialized() bool {
	return notecardContext != nil
}

// S3ListBucketResult represents the XML structure returned by S3 list bucket API
type S3ListBucketResult struct {
	XMLName  xml.Name `xml:"ListBucketResult"`
	Contents []struct {
		Key string `xml:"Key"`
	} `xml:"Contents"`
}

// compareVersions compares two semantic version strings
// Returns: 1 if v1 > v2, -1 if v1 < v2, 0 if v1 == v2
func compareVersions(v1, v2 string) int {
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var p1, p2 int
		if i < len(parts1) {
			p1, _ = strconv.Atoi(parts1[i])
		}
		if i < len(parts2) {
			p2, _ = strconv.Atoi(parts2[i])
		}

		if p1 > p2 {
			return 1
		}
		if p1 < p2 {
			return -1
		}
	}
	return 0
}

// extractKeysFromXml extracts firmware keys from S3 XML response
func extractKeysFromXml(xmlData []byte) ([]string, error) {
	var result S3ListBucketResult
	if err := xml.Unmarshal(xmlData, &result); err != nil {
		return nil, fmt.Errorf("failed to parse XML: %w", err)
	}

	keys := make([]string, len(result.Contents))
	for i, content := range result.Contents {
		keys[i] = content.Key
	}
	return keys, nil
}

// extractAvailableVersions extracts unique version numbers from firmware keys
func extractAvailableVersions(relevantKeys []string) []string {
	versionRegex := regexp.MustCompile(`-(\d+\.\d+\.\d+\.\d+)\.(?:bin|dfu)`)
	versionSet := make(map[string]bool)

	for _, key := range relevantKeys {
		matches := versionRegex.FindStringSubmatch(key)
		if len(matches) > 1 {
			versionSet[matches[1]] = true
		}
	}

	versions := make([]string, 0, len(versionSet))
	for version := range versionSet {
		versions = append(versions, version)
	}
	return versions
}

// getNotecardTypeFromModel determines Notecard type from model string
func getNotecardTypeFromModel(notecardModel string) string {
	if notecardModel == "" {
		return ""
	}

	// Mapping from firmware type prefix to substrings found in model names
	notecardTypeMap := map[string][]string{
		"":   {"500"},            // e.g., "NOTE-NBGL-500" maps to ""
		"u5": {"NB", "MB", "WB"}, // e.g., "NOTE-WBNA", "NOTE-NBGL", "NOTE-MBNA" map to "u5"
		"wl": {"LW"},             // e.g., "NOTE-LWL" maps to "wl"
		"s3": {"ESP"},            // e.g., "NOTE-ESP32" maps to "s3"
	}

	for notecardType, modelSubstrings := range notecardTypeMap {
		for _, substring := range modelSubstrings {
			if strings.Contains(notecardModel, substring) {
				return notecardType
			}
		}
	}

	return "" // Return empty string if no match found
}

// listAvailableFirmwareVersions fetches available firmware versions for a given type
func listAvailableFirmwareVersions(updateChannel, notecardType string) ([]string, error) {
	firmwareIndexUrl := fmt.Sprintf("https://s3.us-east-1.amazonaws.com/notecard-firmware?prefix=%s", updateChannel)

	resp, err := http.Get(firmwareIndexUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch firmware index: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch firmware index: %s", resp.Status)
	}

	xmlData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 2. Extract all keys
	allKeys, err := extractKeysFromXml(xmlData)
	if err != nil {
		return nil, fmt.Errorf("failed to extract keys from XML: %w", err)
	}

	if len(allKeys) == 0 {
		return []string{}, nil
	}

	// 3. Filter keys relevant to notecardType
	var relevantKeys []string
	searchPattern := fmt.Sprintf("-%s-", notecardType)
	for _, key := range allKeys {
		if strings.Contains(key, searchPattern) {
			relevantKeys = append(relevantKeys, key)
		}
	}

	if len(relevantKeys) == 0 {
		return []string{}, nil
	}

	// 4. Extract and return unique versions
	availableVersions := extractAvailableVersions(relevantKeys)
	return availableVersions, nil
}

// getLatestFirmwareVersion returns the latest firmware version for a given update channel and notecard type
func getLatestFirmwareVersion(updateChannel, notecardType string) (string, error) {
	versions, err := listAvailableFirmwareVersions(updateChannel, notecardType)
	if err != nil {
		return "", err
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("no firmware versions found for %s/%s", updateChannel, notecardType)
	}

	// Find the latest version by comparing all versions
	latestVersion := versions[0]
	for _, version := range versions[1:] {
		if compareVersions(version, latestVersion) > 0 {
			latestVersion = version
		}
	}

	return latestVersion, nil
}

func CreateNotecardInitializeTool() mcp.Tool {
	return mcp.NewTool("initialize",
		mcp.WithDescription("Initialize a connection to the Notecard for communication. This creates a notecard object that can be used for subsequent operations."),
		mcp.WithString("port",
			mcp.Description("The port to connect to the Notecard. If not provided, the default port will be used."),
		),
		mcp.WithNumber("baud",
			mcp.Description("The baud rate to connect to the Notecard. If not provided, the default baud rate will be used."),
		),
	)
}

func CreateNotecardCloseTool() mcp.Tool {
	return mcp.NewTool("close",
		mcp.WithDescription("Close the connection to the Notecard and clean up resources. Only use this tool if you are done with the Notecard and want to free up resources."),
	)
}

func CreateNotecardRequestTool() mcp.Tool {
	return mcp.NewTool("request",
		mcp.WithDescription("Send a request to the Notecard and return the response. The notecard must be initialized first and you should verify that the request is valid before sending it (e.g. by using the 'docs://api/overview' resource). The request type is the name of the request to send to the Notecard, e.g. 'card.version'. The arguments are optional and are a JSON object that contains the arguments for the request. All requests are documented in the Notecard API documentation, which is provided by the MCP as a resource."),
		mcp.WithString("request",
			mcp.Required(),
			mcp.Description("The request type to send to the Notecard (e.g., 'card.version', 'card.status', 'hub.status', 'card.temp', 'card.voltage')"),
		),
		mcp.WithString("arguments",
			mcp.Description("Optional JSON arguments for the request as a string (for requests that require additional parameters). For example, '{\"minutes\": 60}' will instruct the Notecard to take a temperature reading every 60 minutes, when used in conjunction with the 'card.temp' request."),
		),
	)
}

func CreateNotecardListFirmwareVersionsTool() mcp.Tool {
	return mcp.NewTool("list-firmware-versions",
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

func HandleNotecardInitializeTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	port := request.GetString("port", "")
	baud := request.GetInt("baud", 0)

	if port != "" {
		notecardPort = &port
	}

	if baud != 0 {
		notecardBaud = &baud
	}

	err := initializeNotecard()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to initialize notecard: %v", err)), nil
	}

	protocol, port, portConfig := notecardContext.Identify()

	return mcp.NewToolResultText(fmt.Sprintf("Notecard initialized successfully. Protocol: %s, Port: %s, Config: %d", protocol, port, portConfig)), nil
}

func HandleNotecardCloseTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if notecardContext == nil {
		return mcp.NewToolResultText("Notecard is not currently connected"), nil
	}

	closeNotecard()
	return mcp.NewToolResultText("Notecard connection closed successfully"), nil
}

func HandleNotecardRequestTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	if notecardContext == nil {
		return mcp.NewToolResultError("Notecard is not initialized. Please run 'initialize-notecard' first."), nil
	}

	reqType, err := request.RequireString("request")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid request parameter: %v", err)), nil
	}

	req := map[string]interface{}{
		"req": reqType,
	}

	if argsStr := request.GetString("arguments", ""); argsStr != "" {
		var args map[string]interface{}
		if err := json.Unmarshal([]byte(argsStr), &args); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid arguments JSON: %v", err)), nil
		}
		req["args"] = args
	}

	response, err := notecardContext.Transaction(req)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send request to notecard: %v", err)), nil
	}

	responseJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		responseStr := fmt.Sprintf("%+v", response)
		return mcp.NewToolResultText(fmt.Sprintf("Request: %s\nResponse: %s", reqType, responseStr)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Request: %s\nResponse:\n%s", reqType, string(responseJSON))), nil
}

func HandleNotecardListFirmwareVersionsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	updateChannel, err := request.RequireString("updateChannel")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid updateChannel parameter: %v", err)), nil
	}

	notecardModel, err := request.RequireString("notecardModel")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid notecardModel parameter: %v", err)), nil
	}

	notecardType := getNotecardTypeFromModel(notecardModel)
	if notecardType == "" && !strings.Contains(notecardModel, "500") {
		return mcp.NewToolResultError(fmt.Sprintf("Could not determine Notecard type for model '%s'. Check the provided model.", notecardModel)), nil
	}

	versions, err := listAvailableFirmwareVersions(updateChannel, notecardType)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Could not list firmware versions: %v", err)), nil
	}

	if len(versions) == 0 {
		return mcp.NewToolResultText(fmt.Sprintf("No firmware versions found for %s/%s", updateChannel, notecardModel)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Available firmware versions for %s: %s", updateChannel, strings.Join(versions, ", "))), nil
}

// downloadFirmware downloads a firmware binary from the given URL
func downloadFirmware(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download firmware: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download firmware: %s at %s", resp.Status, url)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read firmware data: %w", err)
	}

	return data, nil
}

// constructFirmwareURL builds the firmware download URL based on parameters
func constructFirmwareURL(updateChannel, notecardType, version string) string {
	var filename string
	directory := fmt.Sprintf("notecard-%s", version)

	if updateChannel == "DevRel" {
		updateChannel = fmt.Sprintf("DevRel/%s", strings.Join(strings.Split(version, ".")[:3], "."))
	}

	if notecardType == "" {
		// For NOTE-NBGL-500 and similar
		filename = fmt.Sprintf("notecard-%s.bin", version)
	} else {
		// For other notecard types
		filename = fmt.Sprintf("notecard-%s-%s.bin", notecardType, version)
	}

	return fmt.Sprintf("https://s3.us-east-1.amazonaws.com/notecard-firmware/%s/%s/%s", updateChannel, directory, filename)
}

func CreateNotecardUpdateFirmwareTool() mcp.Tool {
	return mcp.NewTool("update-firmware",
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

func HandleNotecardUpdateFirmwareTool(logger *utils.MCPLogger) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if notecardContext == nil {
			logger.Error("Firmware update failed: Notecard is not initialized")
			return mcp.NewToolResultError("Notecard is not initialized. Please run 'initialize' first."), nil
		}

		updateChannel := request.GetString("updateChannel", "LTS")
		notecardModel, err := request.RequireString("notecardModel")
		if err != nil {
			logger.Errorf("Firmware update failed: Invalid notecardModel parameter: %v", err)
			return mcp.NewToolResultError(fmt.Sprintf("Invalid notecardModel parameter: %v", err)), nil
		}

		version := request.GetString("version", "")
		force := request.GetBool("force", false)

		// Create a unique progress token for this operation
		progressToken := fmt.Sprintf("firmware_update_%d", time.Now().Unix())

		// Helper function to send MCP-compliant progress notifications
		sendProgress := func(progress, total float64, message string) {
			logger.SendProgressNotification(progressToken, progress, total, message)
		}

		// Log the start of firmware update with context
		updateContext := map[string]interface{}{
			"updateChannel": updateChannel,
			"notecardModel": notecardModel,
			"version":       version,
			"force":         force,
		}
		logger.LogWithContext(utils.LogLevelInfo, "Starting firmware update process", updateContext)
		sendProgress(0, 100, "Starting firmware update process...")

		// Determine notecard type from model
		notecardType := getNotecardTypeFromModel(notecardModel)
		if notecardType == "" && !strings.Contains(notecardModel, "500") {
			logger.Errorf("Could not determine Notecard type for model '%s'", notecardModel)
			return mcp.NewToolResultError(fmt.Sprintf("Could not determine Notecard type for model '%s'. Check the provided model.", notecardModel)), nil
		}

		logger.Infof("Determined Notecard type: '%s' for model: '%s'", notecardType, notecardModel)
		sendProgress(5, 100, fmt.Sprintf("Determined Notecard type: %s", notecardType))

		// If no version specified, get the latest version
		if version == "" {
			logger.Info("No version specified, fetching latest version...")
			sendProgress(10, 100, "Fetching latest firmware version...")

			latestVersion, err := getLatestFirmwareVersion(updateChannel, notecardType)
			if err != nil {
				logger.Errorf("Failed to get latest firmware version: %v", err)
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get latest firmware version: %v", err)), nil
			}
			version = latestVersion
			logger.Infof("Using latest version: %s", version)
			fmt.Fprintf(os.Stderr, "No version specified, using latest version: %s\n", version)
			sendProgress(15, 100, fmt.Sprintf("Using latest version: %s", version))
		}

		// Get current firmware version for comparison
		if !force {
			logger.Info("Checking current firmware version...")
			sendProgress(20, 100, "Checking current firmware version...")

			versionReq := map[string]interface{}{
				"req": "card.version",
			}

			response, err := notecardContext.Transaction(versionReq)
			if err != nil {
				logger.Errorf("Failed to get current firmware version: %v", err)
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get current firmware version: %v", err)), nil
			}

			if currentVersion, exists := response["version"]; exists {
				if currentVersionStr, ok := currentVersion.(string); ok {
					logger.Infof("Current firmware version: %s", currentVersionStr)
					if strings.Contains(currentVersionStr, version) {
						logger.Warningf("Notecard is already running version %s, skipping update", currentVersionStr)
						sendProgress(100, 100, fmt.Sprintf("Already running version %s", currentVersionStr))
						return mcp.NewToolResultText(fmt.Sprintf("Notecard is already running version %s. Use force=true to update anyway.", currentVersionStr)), nil
					}
				}
			}
		} else {
			logger.Info("Force update enabled, skipping version check")
			sendProgress(20, 100, "Force update enabled, skipping version check")
		}

		// Construct firmware URL
		firmwareURL := constructFirmwareURL(updateChannel, notecardType, version)
		logger.Infof("Firmware download URL: %s", firmwareURL)
		sendProgress(25, 100, "Preparing firmware download...")

		// Download firmware
		logger.Info("Starting firmware download...")
		sendProgress(30, 100, "Downloading firmware...")
		fmt.Fprintf(os.Stderr, "Downloading firmware from: %s\n", firmwareURL)

		firmwareData, err := downloadFirmware(firmwareURL)
		if err != nil {
			logger.Errorf("Failed to download firmware: %v", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to download firmware: %v", err)), nil
		}

		logger.Infof("Successfully downloaded firmware: %d bytes", len(firmwareData))
		fmt.Fprintf(os.Stderr, "Downloaded firmware: %d bytes\n", len(firmwareData))
		sendProgress(40, 100, fmt.Sprintf("Downloaded firmware: %d bytes", len(firmwareData)))

		// Perform sideload with progress tracking
		logger.Info("Starting firmware sideload to Notecard...")
		sendProgress(45, 100, "Starting firmware sideload...")
		fmt.Fprintf(os.Stderr, "Starting firmware update...\n")

		// Create a progress callback for the sideload operation
		sideloadProgressCallback := func(currentProgress, totalProgress float64, message string) {
			// Map sideload progress (0-100) to our overall progress (45-85)
			mappedProgress := 45 + (currentProgress/totalProgress)*40
			sendProgress(mappedProgress, 100, message)
		}

		err = utils.SideloadFirmwareWithProgressAndLogger(notecardContext, firmwareData, logger, sideloadProgressCallback)
		if err != nil {
			logger.Errorf("Failed to sideload firmware: %v", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to sideload firmware: %v", err)), nil
		}

		logger.Info("Firmware sideload completed successfully")
		sendProgress(85, 100, "Firmware sideload completed, Notecard restarting...")

		// Close the current connection since the Notecard has restarted
		closeNotecard()
		logger.Info("Closed connection to allow Notecard restart")

		// Wait a moment for the Notecard to fully restart
		logger.Info("Waiting for Notecard to restart...")
		sendProgress(87, 100, "Waiting for Notecard to restart...")
		fmt.Fprintf(os.Stderr, "Waiting for Notecard to restart...\n")
		time.Sleep(3 * time.Second)

		// Attempt to reinitialize the connection with retries
		logger.Info("Attempting to reconnect to Notecard...")
		sendProgress(90, 100, "Attempting to reconnect...")
		fmt.Fprintf(os.Stderr, "Attempting to reconnect...\n")

		var reconnectErr error
		for retry := 0; retry < 5; retry++ {
			logger.Debugf("Reconnection attempt %d/5", retry+1)
			sendProgress(90+float64(retry), 100, fmt.Sprintf("Reconnection attempt %d/5", retry+1))

			reconnectErr = initializeNotecard()
			if reconnectErr == nil {
				logger.Info("Successfully reconnected to Notecard")
				break
			}
			if retry < 4 {
				logger.Debugf("Reconnection attempt %d failed, retrying in 2 seconds...", retry+1)
				time.Sleep(2 * time.Second)
			}
		}

		if reconnectErr != nil {
			logger.Warningf("Failed to reconnect after firmware update: %v", reconnectErr)
			sendProgress(95, 100, "Firmware update completed but reconnection failed")
			return mcp.NewToolResultText(fmt.Sprintf("Firmware update completed successfully to version %s, but failed to reconnect. Please run 'initialize' to reconnect to the Notecard.", version)), nil
		}

		// Verify the new firmware version
		logger.Info("Verifying new firmware version...")
		sendProgress(97, 100, "Verifying new firmware version...")

		versionReq := map[string]interface{}{
			"req": "card.version",
		}
		response, err := notecardContext.Transaction(versionReq)
		if err != nil {
			logger.Warningf("Could not verify new firmware version: %v", err)
			sendProgress(100, 100, "Firmware update completed (verification failed)")
			return mcp.NewToolResultText(fmt.Sprintf("Firmware update completed to version %s, but could not verify the new version. The Notecard connection has been reestablished.", version)), nil
		}

		var newVersion string
		if ver, exists := response["version"]; exists {
			if verStr, ok := ver.(string); ok {
				newVersion = verStr
			}
		}

		if newVersion != "" {
			logger.Infof("Firmware update completed successfully. New version: %s", newVersion)
			sendProgress(100, 100, fmt.Sprintf("Firmware update completed successfully to %s", newVersion))

			// Log completion with context
			completionContext := map[string]interface{}{
				"previousVersion": "unknown", // We could store this from earlier
				"newVersion":      newVersion,
				"updateChannel":   updateChannel,
				"notecardModel":   notecardModel,
			}
			logger.LogWithContext(utils.LogLevelInfo, "Firmware update completed successfully", completionContext)

			return mcp.NewToolResultText(fmt.Sprintf("Successfully updated Notecard firmware to %s. Connection reestablished and verified.", newVersion)), nil
		}

		logger.Infof("Firmware update completed to version %s", version)
		sendProgress(100, 100, "Firmware update completed")
		return mcp.NewToolResultText(fmt.Sprintf("Successfully updated Notecard firmware to version %s. The Notecard has restarted and connection has been reestablished.", version)), nil
	}
}
