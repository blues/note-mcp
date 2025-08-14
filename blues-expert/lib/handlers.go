package lib

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

//go:embed docs/arduino/power_management.md
var powerManagementFS embed.FS

func HandleArduinoNotePowerManagementTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	content, err := powerManagementFS.ReadFile("docs/arduino/power_management.md")
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(string(content)), nil
}

//go:embed docs/arduino/best_practices.md
var bestPracticesFS embed.FS

func HandleArduinoNoteBestPracticesTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	content, err := bestPracticesFS.ReadFile("docs/arduino/best_practices.md")
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(string(content)), nil
}

//go:embed docs/arduino/templates.md
var templatesFS embed.FS

// HandleArduinoNoteTemplatesTool handles the arduino note templates tool
func HandleArduinoNoteTemplatesTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	content, err := templatesFS.ReadFile("docs/arduino/templates.md")
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(string(content)), nil
}

//go:embed docs/arduino/sensors.md
var sensorsFS embed.FS

func HandleArduinoSensorsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	content, err := sensorsFS.ReadFile("docs/arduino/sensors.md")
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(string(content)), nil
}

// HandleNotecardRequestValidateTool handles the notecard request validation tool
func HandleNotecardRequestValidateTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	requestJSON, err := request.RequireString("request")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid request parameter: %v", err)), nil
	}

	// Parse the JSON request
	var reqMap map[string]interface{}
	if err := json.Unmarshal([]byte(requestJSON), &reqMap); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid JSON request: %v", err)), nil
	}

	// Validate the request using the validate.go function with default schema
	if err := ValidateNotecardRequest(reqMap, ""); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Validation failed: %v", err)), nil
	}

	return mcp.NewToolResultText("Request validation successful: The JSON request is valid according to the Notecard API schema."), nil
}

// HandleNotecardGetAPIsTool handles the notecard API documentation tool
func HandleNotecardGetAPIsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	apiName := request.GetString("api", "")

	// Get API documentation
	apiCategory, err := GetNotecardAPIs(apiName)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to get API documentation: %v", err)), nil
	}

	var response []byte
	// If specific API requested, return just the API object
	if apiName != "" && len(apiCategory.APIs) > 0 {
		response, err = json.MarshalIndent(apiCategory.APIs[0], "", "  ")
	} else {
		// Otherwise return the full category structure for listing
		response, err = json.MarshalIndent(apiCategory, "", "  ")
	}

	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Failed to format API documentation: %v", err)), nil
	}

	return mcp.NewToolResultText(string(response)), nil
}
