package lib

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog/log"
)

// FirmwareEntrypointArgs defines the arguments for the firmware entrypoint tool
type FirmwareEntrypointArgs struct {
	Sdk string `json:"sdk" jsonschema:"The sdk to use for the firmware project. Must be one of: Arduino, C, Zephyr, Python"`
}

// FirmwareBestPracticesArgs defines the arguments for the firmware best practices tool
type FirmwareBestPracticesArgs struct {
	Sdk          string `json:"sdk" jsonschema:"The sdk to use for the firmware project. Must be one of: arduino, c, zephyr, python"`
	DocumentType string `json:"document_type" jsonschema:"The type of documentation to retrieve (e.g., 'best_practices', 'templates', 'debugging', 'connectivity', 'sensors', 'power_management')"`
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
				&mcp.TextContent{Text: "Error: document_type parameter is required and cannot be empty. Examples: 'best_practices', 'templates', 'debugging', 'connectivity', 'sensors', 'power_management'"},
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

	// Get schema version for metadata
	schemaVersion := GetSchemaVersion("")

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: "Request validation successful: The JSON request is valid according to the Notecard API schema.",
				Meta: mcp.Meta{
					"schema_version": schemaVersion,
				},
			},
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

	// Get schema version for metadata
	schemaVersion := GetSchemaVersion("")

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
			&mcp.TextContent{
				Text: string(response),
				Meta: mcp.Meta{
					"schema_version": schemaVersion,
				},
			},
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

	// Log the response for server debugging
	if result != nil && !result.IsError && len(result.Content) > 0 {
		if textContent, ok := result.Content[0].(*mcp.TextContent); ok {
			log.Debug().
				Str("tool", "docs_search").
				Str("response", textContent.Text).
				Msg("Response sent to client")
		}
	}

	return result, nil, nil
}
