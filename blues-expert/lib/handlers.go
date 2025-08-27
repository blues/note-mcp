package lib

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
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

// HandleNotecardSearchTool handles the notecard documentation search tool
func HandleNotecardSearchTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := request.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid query parameter: %v", err)), nil
	}

	// Call the search implementation from query.go
	return SearchNotecardDocs(ctx, query)
}

// HandleNotecardSearchExpertTool handles the expert notecard search tool with AI sampling
func HandleNotecardSearchExpertTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query, err := request.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid query parameter: %v", err)), nil
	}

	userContext := request.GetString("context", "")

	// First, get the raw search results
	searchResult, err := SearchNotecardDocs(ctx, query)
	if err != nil {
		return searchResult, err
	}

	// Check if search returned an error
	if searchResult.IsError {
		return searchResult, nil
	}

	// Extract the search results text
	var searchText string
	if len(searchResult.Content) > 0 {
		if textContent, ok := searchResult.Content[0].(mcp.TextContent); ok {
			searchText = textContent.Text
		} else {
			searchText = fmt.Sprintf("%v", searchResult.Content[0])
		}
	}

	// Build the expert system prompt
	systemPrompt := `You are a Notecard Expert with specialist knowledge in IoT product development and embedded systems design. You have deep expertise in:

- Blues Notecard hardware and firmware capabilities
- Cellular IoT connectivity and optimization
- GPS/GNSS positioning systems
- Power management for battery-operated devices
- Embedded systems architecture and best practices
- IoT product development lifecycle
- Troubleshooting connectivity and hardware issues
- Integration with microcontrollers and development boards

Your role is to analyze search results from Blues documentation and provide expert, actionable advice. Focus on:
1. Practical implementation details
2. Best practices and common pitfalls
3. Performance optimization recommendations
4. Troubleshooting guidance
5. Real-world deployment considerations

Be concise but comprehensive. Provide specific code examples, configuration recommendations, or step-by-step guidance when appropriate.`

	// Build the user prompt with search results and context
	var userPrompt string
	if userContext != "" {
		userPrompt = fmt.Sprintf(`Question: %s

Additional Context: %s

Based on the following search results from Blues documentation, provide expert analysis and recommendations:

%s

Please provide a comprehensive, expert-level response that goes beyond just summarizing the search results. Include practical advice, best practices, and any additional considerations for successful implementation.`, query, userContext, searchText)
	} else {
		userPrompt = fmt.Sprintf(`Question: %s

Based on the following search results from Blues documentation, provide expert analysis and recommendations:

%s

Please provide a comprehensive, expert-level response that goes beyond just summarizing the search results. Include practical advice, best practices, and any additional considerations for successful implementation.`, query, searchText)
	}

	// Send progress notification to client
	serverFromCtx := server.ServerFromContext(ctx)
	if serverFromCtx != nil {
		// Send a progress notification to let the client know we're processing
		total := 100.0
		message := "Analyzing search results and preparing expert consultation..."
		progressNotification := mcp.NewProgressNotification("expert_search_progress", 50.0, &total, &message)

		// Send as a notification to the client
		err := serverFromCtx.SendNotificationToClient(ctx, "notifications/progress", map[string]any{
			"progressToken": progressNotification.Params.ProgressToken,
			"progress":      progressNotification.Params.Progress,
			"total":         progressNotification.Params.Total,
			"message":       progressNotification.Params.Message,
		})
		if err != nil {
			// Log error but don't fail the operation
			fmt.Printf("Failed to send progress notification: %v\n", err)
		}
	}

	// Create sampling request

	samplingRequest := mcp.CreateMessageRequest{
		CreateMessageParams: mcp.CreateMessageParams{
			Messages: []mcp.SamplingMessage{
				{
					Role: mcp.RoleUser,
					Content: mcp.TextContent{
						Type: "text",
						Text: userPrompt,
					},
				},
			},
			SystemPrompt: systemPrompt,
			MaxTokens:    20000,
			Temperature:  0.3, // Lower temperature for more focused, technical responses
		},
	}

	// Request sampling from the client
	samplingCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	result, err := serverFromCtx.RequestSampling(samplingCtx, samplingRequest)
	if err != nil {
		// Check if the error is due to sampling not being supported
		if isUnsupportedSamplingError(err) {
			// Fallback: Return enhanced search results with expert guidance
			return createExpertFallbackResponse(query, userContext, searchText), nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				mcp.TextContent{
					Type: "text",
					Text: fmt.Sprintf("Error requesting sampling: %v", err),
				},
			},
			IsError: true,
		}, nil
	}

	// Send completion progress notification
	if serverFromCtx != nil {
		total := 100.0
		completeMessage := "Expert analysis complete!"
		completionNotification := mcp.NewProgressNotification("expert_search_progress", 100.0, &total, &completeMessage)
		serverFromCtx.SendNotificationToClient(ctx, "notifications/progress", map[string]any{
			"progressToken": completionNotification.Params.ProgressToken,
			"progress":      completionNotification.Params.Progress,
			"total":         completionNotification.Params.Total,
			"message":       completionNotification.Params.Message,
		})
	}

	// Extract the expert response
	expertResponse := getTextFromContent(result.Content)

	// Return the expert analysis
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("# Notecard Expert Analysis\n\n**Query:** %s\n\n**Expert Response:**\n%s\n\n---\n*Analysis provided by AI model: %s*", query, expertResponse, result.Model),
			},
		},
	}, nil
}

// Helper function to extract text from content safely
func getTextFromContent(content any) string {
	// Handle the most common case first
	if textContent, ok := content.(mcp.TextContent); ok {
		return textContent.Text
	}

	// Handle map structures (JSON unmarshaled content)
	if contentMap, ok := content.(map[string]any); ok {
		if text, ok := contentMap["text"].(string); ok {
			return text
		}
	}

	// Handle string directly
	if str, ok := content.(string); ok {
		return str
	}

	// Fallback to string representation
	return fmt.Sprintf("%v", content)
}

// isUnsupportedSamplingError checks if the error indicates sampling is not supported
func isUnsupportedSamplingError(err error) bool {
	errorMsg := err.Error()
	return errorMsg == "session does not support sampling" ||
		errorMsg == "sampling not supported" ||
		errorMsg == "client does not support sampling"
}

// createExpertFallbackResponse creates an enhanced response when sampling is not available
func createExpertFallbackResponse(query, userContext, searchText string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: searchText,
			},
		},
	}
}
