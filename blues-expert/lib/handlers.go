package lib

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// FirmwareEntrypointArgs defines the arguments for the firmware entrypoint tool
type FirmwareEntrypointArgs struct {
	Sdk string `json:"sdk" jsonschema:"The sdk to use for the firmware project. Must be one of: Arduino, C, Zephyr, Python"`
}

// FirmwareBestPracticesArgs defines the arguments for the firmware best practices tool
type FirmwareBestPracticesArgs struct {
	Sdk          string `json:"sdk" jsonschema:"The sdk to use for the firmware project. Must be one of: arduino, c, zephyr, python"`
	DocumentType string `json:"document_type" jsonschema:"The type of documentation to retrieve (e.g., 'power_management', 'best_practices', 'sensors', 'templates')"`
}

// RequestValidateArgs defines the arguments for the notecard request validation tool
type RequestValidateArgs struct {
	Request string `json:"request" jsonschema:"The JSON string of the request to validate (e.g., '{\"req\":\"card.version\"}', '{\"req\":\"card.temp\",\"minutes\":60}')"`
}

// GetAPIsArgs defines the arguments for the notecard API documentation tool
type GetAPIsArgs struct {
	API string `json:"api,omitempty" jsonschema:"The specific Notecard API to get documentation for (e.g., 'card.attn', 'card.version', 'hub.status', 'note.add')"`
}

// SearchArgs defines the arguments for the notecard search tool
type SearchArgs struct {
	Query string `json:"query" jsonschema:"The search query or question to find relevant documentation (e.g., 'How can I use cellular and gps at the same time?', 'Notecard power consumption', 'Troubleshooting connectivity issues')"`
}

// SearchExpertArgs defines the arguments for the expert notecard search tool
type SearchExpertArgs struct {
	Query   string `json:"query" jsonschema:"The search query or technical question about Notecard, IoT development, or embedded systems (e.g., 'How can I optimize power consumption for a solar-powered sensor?', 'Best practices for cellular connectivity in remote locations')"`
	Context string `json:"context,omitempty" jsonschema:"Optional additional context about your specific use case, hardware setup, or constraints to help the expert provide more targeted advice"`
}

//go:embed docs
var docs embed.FS

// Firmware Tools
func HandleFirmwareEntrypointTool(ctx context.Context, request *mcp.CallToolRequest, args FirmwareEntrypointArgs) (*mcp.CallToolResult, any, error) {
	TrackSession(request, "firmware_entrypoint")

	if args.Sdk == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "Error: SDK parameter is required and cannot be empty. Valid values are: Arduino, C, Zephyr, Python"},
			},
			IsError: true,
		}, nil, nil
	}

	// Get the SDK & Index file - convert to lowercase for directory name
	sdk := strings.ToLower(args.Sdk)
	indexFile := fmt.Sprintf("docs/%s/index.md", sdk)

	// Get the docs
	docContent, err := docs.ReadFile(indexFile)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Error reading docs: %v", err)},
			},
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(docContent)},
		},
	}, nil, nil
}

func HandleFirmwareBestPracticesTool(ctx context.Context, request *mcp.CallToolRequest, args FirmwareBestPracticesArgs) (*mcp.CallToolResult, any, error) {
	TrackSession(request, "firmware_best_practices")

	if args.Sdk == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "Error: SDK parameter is required and cannot be empty. Valid values are: arduino, c, zephyr, python"},
			},
			IsError: true,
		}, nil, nil
	}

	if args.DocumentType == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: "Error: document_type parameter is required and cannot be empty. Examples: 'power_management', 'best_practices', 'sensors', 'templates'"},
			},
			IsError: true,
		}, nil, nil
	}

	// Convert SDK to lowercase for directory name
	sdk := strings.ToLower(args.Sdk)
	docFile := fmt.Sprintf("docs/%s/%s.md", sdk, args.DocumentType)

	// Get the docs
	docContent, err := docs.ReadFile(docFile)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Error reading documentation: %v. Make sure the SDK ('%s') and document_type ('%s') are valid.", err, sdk, args.DocumentType)},
			},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(docContent)},
		},
	}, nil, nil
}

// Notecard API Tools
func HandleAPIValidateTool(ctx context.Context, request *mcp.CallToolRequest, args RequestValidateArgs) (*mcp.CallToolResult, any, error) {
	TrackSession(request, "api_validate")

	var reqMap map[string]interface{}
	if err := json.Unmarshal([]byte(args.Request), &reqMap); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Invalid JSON request: %v", err)},
			},
			IsError: true,
		}, nil, nil
	}

	if err := ValidateNotecardRequest(reqMap, ""); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Validation failed: %v", err)},
			},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Request validation successful: The JSON request is valid according to the Notecard API schema."},
		},
	}, nil, nil
}

func HandleAPIDocsTool(ctx context.Context, request *mcp.CallToolRequest, args GetAPIsArgs) (*mcp.CallToolResult, any, error) {
	TrackSession(request, "api_docs")

	// Get API documentation
	apiCategory, err := GetNotecardAPIs(ctx, request, args.API)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Failed to get API documentation: %v", err)},
			},
			IsError: true,
		}, nil, nil
	}

	var response []byte
	// If specific API requested, return just the API object
	if args.API != "" && len(apiCategory.APIs) > 0 {
		response, err = json.MarshalIndent(apiCategory.APIs[0], "", "  ")
	} else {
		// Otherwise return the full category structure for listing
		response, err = json.MarshalIndent(apiCategory, "", "  ")
	}

	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Failed to format API documentation: %v", err)},
			},
			IsError: true,
		}, nil, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(response)},
		},
	}, nil, nil
}

// Blues Documentation Tools
func HandleDocsSearchTool(ctx context.Context, request *mcp.CallToolRequest, args SearchArgs) (*mcp.CallToolResult, any, error) {
	TrackSession(request, "docs_search")

	// Call the search implementation from query.go
	result, err := SearchNotecardDocs(ctx, request, args.Query)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Search error: %v", err)},
			},
			IsError: true,
		}, nil, nil
	}

	return result, nil, nil
}

func HandleDocsSearchExpertTool(ctx context.Context, request *mcp.CallToolRequest, args SearchExpertArgs) (*mcp.CallToolResult, any, error) {
	TrackSession(request, "docs_search_expert")

	// First, get the raw search results
	searchResult, err := SearchNotecardDocs(ctx, request, args.Query)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Search error: %v", err)},
			},
			IsError: true,
		}, nil, nil
	}

	// Check if search returned an error
	if searchResult.IsError {
		return searchResult, nil, nil
	}

	// Extract the search results text
	var searchText string
	if len(searchResult.Content) > 0 {
		if textContent, ok := searchResult.Content[0].(*mcp.TextContent); ok {
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
	if args.Context != "" {
		userPrompt = fmt.Sprintf(`Question: %s

Additional Context: %s

Based on the following search results from Blues documentation, provide expert analysis and recommendations:

%s

Please provide a comprehensive, expert-level response that goes beyond just summarizing the search results. Include practical advice, best practices, and any additional considerations for successful implementation.`, args.Query, args.Context, searchText)
	} else {
		userPrompt = fmt.Sprintf(`Question: %s

Based on the following search results from Blues documentation, provide expert analysis and recommendations:

%s

Please provide a comprehensive, expert-level response that goes beyond just summarizing the search results. Include practical advice, best practices, and any additional considerations for successful implementation.`, args.Query, searchText)
	}

	// Create sampling request using the new SDK
	samplingParams := &mcp.CreateMessageParams{
		Messages: []*mcp.SamplingMessage{
			{
				Role:    "user",
				Content: &mcp.TextContent{Text: userPrompt},
			},
		},
		SystemPrompt: systemPrompt,
		MaxTokens:    4000,
		Temperature:  0.3, // Lower temperature for more focused, technical responses
	}

	// Request sampling from the client
	result, err := request.Session.CreateMessage(ctx, samplingParams)
	if err != nil {
		// Check if the error is due to sampling not being supported
		if isUnsupportedSamplingError(err) {
			// Fallback: Return enhanced search results with expert guidance
			return createExpertFallbackResponse(args.Query, args.Context, searchText), nil, nil
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Error requesting sampling: %v", err)},
			},
			IsError: true,
		}, nil, nil
	}

	// Extract the expert response
	expertResponse := getTextFromContent(result.Content)

	// Return the expert analysis
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fmt.Sprintf("# Notecard Expert Analysis\n\n**Query:** %s\n\n**Expert Response:**\n%s\n\n---\n*Analysis provided by AI model: %s*", args.Query, expertResponse, result.Model)},
		},
	}, nil, nil
}

// Helper function to extract text from content safely
func getTextFromContent(content mcp.Content) string {
	// Handle the most common case first
	if textContent, ok := content.(*mcp.TextContent); ok {
		return textContent.Text
	}

	// Fallback to string representation
	return fmt.Sprintf("%v", content)
}

// isUnsupportedSamplingError checks if the error indicates sampling is not supported
func isUnsupportedSamplingError(err error) bool {
	errorMsg := err.Error()
	return errorMsg == "session does not support sampling" ||
		errorMsg == "sampling not supported" ||
		errorMsg == "client does not support sampling" ||
		errorMsg == "client does not support CreateMessage"
}

// createExpertFallbackResponse creates an enhanced response when sampling is not available
func createExpertFallbackResponse(query, userContext, searchText string) *mcp.CallToolResult {
	contextMsg := ""
	if userContext != "" {
		contextMsg = fmt.Sprintf("\n\n**Additional Context:** %s", userContext)
	}

	fallbackText := fmt.Sprintf(`# Notecard Documentation Search Results

**Query:** %s%s

**Note:** Expert AI analysis is not available because sampling is not supported by the client. Below are the raw search results from Blues documentation:

%s

---
*For expert AI-enhanced analysis, please use a client that supports sampling functionality.*`, query, contextMsg, searchText)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: fallbackText},
		},
	}
}
