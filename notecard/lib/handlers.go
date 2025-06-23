package lib

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/user"
	"regexp"
	"strconv"
	"strings"
	"time"

	"note-mcp/utils"

	"github.com/mark3labs/mcp-go/mcp"
)

// ConfigDir returns the config directory
func getConfigDir() string {
	usr, err := user.Current()
	if err != nil {
		return "."
	}
	path := usr.HomeDir + "/note"
	return path
}

// Get the pathname of config settings
func getConfigSettingsPath() string {
	return getConfigDir() + "/config.json"
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

	return ""
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

	allKeys, err := extractKeysFromXml(xmlData)
	if err != nil {
		return nil, fmt.Errorf("failed to extract keys from XML: %w", err)
	}

	if len(allKeys) == 0 {
		return []string{}, nil
	}

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

	// Extract and return unique versions
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

// HandleNotecardInitializeTool handles the notecard initialization
func HandleNotecardInitializeTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	reset := request.GetBool("reset", false)
	port := request.GetString("port", "")
	baud := request.GetInt("baud", 0)

	if reset {
		// check if config exists and delete it
		if _, err := os.Stat(getConfigSettingsPath()); err == nil {
			os.Remove(getConfigSettingsPath())
		}
	}

	if port != "" {
		_, err := utils.ExecuteNotecardCommand([]string{"-port", port})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to set port: %v", err)), nil
		}
	}

	if baud != 0 {
		_, err := utils.ExecuteNotecardCommand([]string{"-portconfig", strconv.Itoa(baud)})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to set baud rate: %v", err)), nil
		}
	}

	output, err := utils.ExecuteNotecardCommand([]string{"-req", "{\"req\":\"card.version\"}"})

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to initialize notecard: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Notecard initialized successfully. Output: %s", output)), nil
}

// HandleNotecardRequestTool handles sending requests to the notecard
func HandleNotecardRequestTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {

	reqType, err := request.RequireString("request")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid request parameter: %v", err)), nil
	}

	var output string
	if _, ok := os.LookupEnv("BLUES"); !ok {
		output, err = utils.ExecuteNotecardCommand([]string{"-req", reqType})
	} else {
		output, err = utils.ExecuteNotecardCommand([]string{"-req", reqType, "-verbose"})
	}
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to send request to notecard: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Response:\n%s", output)), nil
}

// HandleNotecardListFirmwareVersionsTool handles listing firmware versions
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

// HandleNotecardUpdateFirmwareTool handles firmware updates
func HandleNotecardUpdateFirmwareTool(logger *utils.MCPLogger) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
			sendProgress(15, 100, fmt.Sprintf("Using latest version: %s", version))
		}

		// Get current firmware version for comparison
		if !force {
			logger.Info("Checking current firmware version...")
			sendProgress(20, 100, "Checking current firmware version...")

			response, err := utils.ExecuteNotecardCommand([]string{"-req", "{\"req\":\"card.version\"}"})
			if err != nil {
				logger.Errorf("Failed to get current firmware version: %v", err)
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get current firmware version: %v", err)), nil
			}

			if strings.Contains(response, version) {
				logger.Warningf("Notecard is already running version %s, skipping update", version)
				sendProgress(100, 100, fmt.Sprintf("Already running version %s", version))
				return mcp.NewToolResultText(fmt.Sprintf("Notecard is already running version %s. Use force=true to update anyway.", version)), nil
			}
		} else {
			logger.Info("Force update enabled, skipping version check")
			sendProgress(20, 100, "Force update enabled, skipping version check")
		}

		firmwareURL := constructFirmwareURL(updateChannel, notecardType, version)
		logger.Infof("Firmware download URL: %s", firmwareURL)
		sendProgress(25, 100, "Preparing firmware download...")

		logger.Info("Starting firmware download...")
		sendProgress(30, 100, "Downloading firmware...")
		firmwareData, err := downloadFirmware(firmwareURL)
		if err != nil {
			logger.Errorf("Failed to download firmware: %v", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to download firmware: %v", err)), nil
		}

		// write firmwareData to a file
		err = os.WriteFile("/tmp/firmware.bin", firmwareData, 0644)
		if err != nil {
			logger.Errorf("Failed to write firmware to file: %v", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to write firmware to file: %v", err)), nil
		}

		logger.Infof("Successfully downloaded firmware: %d bytes", len(firmwareData))
		sendProgress(40, 100, fmt.Sprintf("Downloaded firmware: %d bytes", len(firmwareData)))

		logger.Info("Starting firmware sideload to Notecard...")
		sendProgress(45, 100, "Starting firmware sideload...")

		_, err = utils.ExecuteNotecardCommandWithLogger([]string{"-sideload", "/tmp/firmware.bin", "-fast"}, logger)
		if err != nil {
			logger.Errorf("Failed to sideload firmware: %v", err)
			return mcp.NewToolResultError(fmt.Sprintf("Failed to sideload firmware: %v", err)), nil
		}

		logger.Info("Firmware sideload completed successfully")
		sendProgress(100, 100, "Firmware sideload completed.")

		response, err := utils.ExecuteNotecardCommand([]string{"-req", "{\"req\":\"card.version\"}"})
		if err != nil {
			logger.Warningf("Could not verify new firmware version: %v", err)
			return mcp.NewToolResultText(fmt.Sprintf("Firmware update completed to version %s, but could not verify the new version. The Notecard connection has been reestablished.", version)), nil
		}

		var newVersion string
		if strings.Contains(response, version) {
			newVersion = version
		}

		if newVersion != "" {
			logger.Infof("Firmware update completed successfully. New version: %s", newVersion)

			// Log completion with context
			completionContext := map[string]interface{}{
				"previousVersion": "unknown", // We could store this from earlier
				"newVersion":      newVersion,
				"updateChannel":   updateChannel,
				"notecardModel":   notecardModel,
			}
			logger.LogWithContext(utils.LogLevelInfo, "Firmware update completed successfully", completionContext)

			return mcp.NewToolResultText(fmt.Sprintf("Successfully updated Notecard firmware to %s %s. Connection reestablished and verified.", updateChannel, newVersion)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully updated Notecard firmware to version %s. The Notecard has restarted and connection has been reestablished.", version)), nil
	}
}

// HandleNotecardValidateRequestTool handles request validation
func HandleNotecardValidateRequestTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	reqString, err := request.RequireString("request")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid request parameter: %v", err)), nil
	}

	schemaURL := request.GetString("schema_url", "")
	// set the NOTE_JSON_SCHEMA_URL environment variable if a schema URL is provided, otherwise unset it
	if schemaURL != "" {
		os.Setenv("NOTE_JSON_SCHEMA_URL", schemaURL)
	} else {
		os.Unsetenv("NOTE_JSON_SCHEMA_URL")
	}

	output, err := utils.ExecuteNotecardCommandWithEnv([]string{"-req", reqString, "-dry", "-verbose"}, os.Environ())
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Attempt to validate request failed: %v", err)), nil
	}

	output = strings.ReplaceAll(output, "\n", "")

	// If the output is not equal to the request, return the reason why it failed
	if output != reqString {
		return mcp.NewToolResultText(fmt.Sprintf("Validation for %s failed: %s", reqString, output)), nil
	}

	// if the BLUES environment variable is not set, return a warning
	if _, ok := os.LookupEnv("BLUES"); !ok {
		return mcp.NewToolResultText(fmt.Sprintf("✓ Request validation successful: The request '%s' is valid JSON.", reqString)), nil
	} else {
		return mcp.NewToolResultText(fmt.Sprintf("✓ Request validation successful: The request '%s' is valid according to the Notecard API schema.", reqString)), nil
	}
}

// HandleDecryptNoteTool handles the decrypt note tool
func HandleDecryptNoteTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	noteFile := request.GetString("note_file", "mysecrets.qi")

	instructions := fmt.Sprintf(`To decrypt the note file '%s', follow these steps:

1. First, sync with the hub to get any pending notefiles:
   {"req":"hub.sync"}

2. Then decrypt and retrieve the note:
   {"req":"note.get","file":"%s","decrypt":true,"delete":true}

Key points:
- The "decrypt":true flag tells the Notecard to decrypt the note contents
- The "delete":true flag removes the note from the queue after retrieval
- Make sure the Notecard has the proper encryption keys configured
- If decryption fails, check that the note was encrypted with the correct public key`, noteFile, noteFile)

	return mcp.NewToolResultText(instructions), nil
}

// HandleProvisionNotecardTool handles the provision notecard tool
func HandleProvisionNotecardTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	productUID, err := request.RequireString("product_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid product_uid parameter: %v", err)), nil
	}

	_, err = request.RequireString("project_uid")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid project_uid parameter: %v", err)), nil
	}

	ssid := request.GetString("ssid", "")
	password := request.GetString("password", "")
	var wifiInstructions string
	if ssid != "" && password != "" {
		wifiInstructions = fmt.Sprintf(`
		2a. Configure the Notecard to connect to your WiFi network: {"req":"card.wifi","ssid":"%s","password":"%s"}`, ssid, password)
	} else {
		wifiInstructions = ""
	}

	instructions := fmt.Sprintf(`To provision your Notecard for the Notehub with product '%s', follow these steps (IMPORTANT: Use the 'notecard_request' tool to send the following requests to the Notecard):

1. Restore the Notecard to factory defaults:
   {"req":"card.restore","delete":true,"connected":true}

2. Configure the Notecard to connect to your project:
   {"req":"hub.set","product":"%s","mode":"continuous"}
%s
3. Sync with the hub to complete provisioning:
   {"req":"hub.sync"}

4. The Notecard should now appear in your Notehub dashboard. You can verify successful connection to Notehub by checking the {"req":"hub.status"} response for "connected":true.

5. You should follow up by using the 'project_events' in the Notehub MCP tool to verify that the Notecard has uploaded the Note file to Notehub. This can sometimes take a short while to propagate, check again after a few seconds if no events are shown.

6. Provide a link to the 'project_url', to help the user verify the provisioning.

Important Notes:
- Ensure you have the correct Product UID from your Notehub project
- For WiFi Notecards, WiFi credentials are required for successful provisioning
- The Notecard will restart during the provisioning process
- Provisioning may take a few minutes to complete fully`, productUID, productUID, wifiInstructions)

	return mcp.NewToolResultText(instructions), nil
}

// HandleTroubleshootConnectionTool handles the troubleshoot connection tool
func HandleTroubleshootConnectionTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	errorMessage, err := request.RequireString("error_message")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid error_message parameter: %v", err)), nil
	}

	notecardType, err := request.RequireString("notecard_type")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid notecard_type parameter: %v", err)), nil
	}

	var typeSpecificSteps string
	if notecardType == "WiFi" {
		typeSpecificSteps = `
MODEL-SPECIFIC STEPS (WiFi Notecard):
- Verify WiFi credentials: {"req":"card.wifi"}
- Check WiFi signal strength in the response
- Try connecting to a different WiFi network if signal is weak
`
	} else if notecardType == "Cellular" {
		typeSpecificSteps = `
MODEL-SPECIFIC STEPS (Cellular Notecard):
- Check SIM card status: {"req":"card.wireless"}
- Verify cellular signal strength (look for "rssi" and "bars" in response)
- Ensure SIM card is properly inserted and activated
- Try moving to a location with better cellular coverage
`
	}

	instructions := fmt.Sprintf(`TROUBLESHOOTING NOTECARD CONNECTION TO NOTEHUB

Error context: %s

Follow these diagnostic steps in order:

1. CHECK CURRENT STATUS:
   {"req":"hub.status"}
   - Look for "connected":true in the response
   - Note any error messages in the "status" field

2. VERIFY BASIC CONNECTIVITY:
   {"req":"card.status"}
   - Check if the Notecard is responsive
   - Verify basic health indicators

3. CHECK PRODUCT CONFIGURATION:
   {"req":"hub.get"}
   - Verify the product UID is correct
   - Ensure mode is set appropriately (usually "continuous")

4. NETWORK CONNECTIVITY CHECK:
%s
5. VERIFY TIME SYNCHRONIZATION:
   {"req":"card.time"}
   - Ensure the Notecard has correct time (required for secure connections)

6. FORCE SYNC ATTEMPT:
   {"req":"hub.sync"}
   - Manually trigger a sync and observe any error messages

7. CHECK FOR PENDING CONFIGURATION:
   {"req":"hub.sync","allow":true}
   - This allows downloading any pending configuration from Notehub

8. RESET AND REPROVISION (if above steps fail):
   - Factory reset: {"req":"card.restore","delete":true,"connected":true}
   - Reconfigure: {"req":"hub.set","product":"YOUR_PRODUCT_UID","mode":"continuous"}
   - For WiFi: {"req":"card.wifi","ssid":"YOUR_SSID","password":"YOUR_PASSWORD"}
   - Sync: {"req":"hub.sync"}

9. FIRMWARE UPDATE (last resort):
   - Check current version: {"req":"card.version"}
   - Consider updating to latest firmware if version is outdated

COMMON ISSUES:
- Incorrect product UID
- WiFi credentials wrong or network blocked
- Poor signal strength
- Outdated firmware
- Time synchronization issues

If issues persist, check the Notehub dashboard for device status and any error logs.`, errorMessage, typeSpecificSteps)

	return mcp.NewToolResultText(instructions), nil
}

// HandleSendNoteTool handles the send note tool
func HandleSendNoteTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	noteFile := request.GetString("note_file", "data.qo")
	noteData := request.GetString("note_data", "")

	var dataExample string
	if noteData != "" {
		dataExample = fmt.Sprintf(`{"req":"note.add","file":"%s","body":%s}`, noteFile, noteData)
	} else {
		dataExample = fmt.Sprintf(`{"req":"note.add","file":"%s","body":{"temperature":23.5,"humidity":65,"timestamp":"2024-01-15T10:30:00Z"}}`, noteFile)
	}

	instructions := fmt.Sprintf(`To send notes from your Notecard to Notehub, follow these steps:

1. ADD A NOTE TO THE NOTEFILE:
   %s

   Key points about the note.add request:
   - "file": The notefile name (use .qo for outbound queues, .dbo for databases)
   - "body": JSON object containing your data
   - Optional "sync": Set to true to immediately sync this note

2. SYNC THE NOTE TO NOTEHUB:
   {"req":"hub.sync"}

   This uploads all pending notes to Notehub. The Notecard will automatically sync based on your sync settings, but you can force an immediate sync.

3. VERIFY THE SYNC STATUS:
   {"req":"hub.status"}

   Check the response for:
   - "connected": true (confirms connection to Notehub)
   - "count": Shows number of notes pending sync
   - "completed": Shows total completed syncs

4. CHECK FOR SYNC ERRORS:
   {"req":"hub.log"}

   This shows recent sync activity and any error messages.

NOTEFILE NAMING CONVENTIONS:
- .qo: Outbound queue (one-way from device to cloud)
- .qi: Inbound queue (one-way from cloud to device)
- .dbo: Outbound database (bidirectional sync, device to cloud)
- .dbi: Inbound database (bidirectional sync, cloud to device)
- .db: Bidirectional database (both directions)

TIPS:
- Use descriptive filenames like "sensors.qo" or "alerts.qo"
- Keep note bodies reasonably small (under 8KB recommended)
- The Notecard will automatically batch multiple small notes for efficient transmission
- Use "sync":true in note.add for time-critical data that needs immediate delivery`, dataExample)

	return mcp.NewToolResultText(instructions), nil
}

// HandleNotecardGetAPIsTool handles the get APIs tool
func HandleNotecardGetAPIsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	category := request.GetString("category", "")

	if category == "" {
		// Return overview of all APIs
		overview, err := GetAPIOverview()
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch API overview: %v", err)), nil
		}
		return mcp.NewToolResultText(overview), nil
	} else {
		// Return specific category documentation
		categoryDoc, err := GetAPICategoryDocumentation(category)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to fetch API category documentation: %v", err)), nil
		}
		return mcp.NewToolResultText(categoryDoc), nil
	}
}
