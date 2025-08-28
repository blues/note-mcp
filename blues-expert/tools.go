package main

import (
	"github.com/google/jsonschema-go/jsonschema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Firmware Tools
func CreateFirmwareEntrypointTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "firmware_entrypoint",
		Description: "Get a starting point for a firmware project. This tool will return information about developing firmware for the Notecard using a specific SDK.",
		InputSchema: &jsonschema.Schema{
			Type: "object",
			Properties: map[string]*jsonschema.Schema{
				"sdk": {
					Type:        "string",
					Description: "The sdk to use for the firmware project. Must be one of: Arduino, C, Zephyr, Python",
					Enum: []any{
						"Arduino",
						"C",
						"Zephyr",
						"Python",
					},
				},
			},
			Required: []string{"sdk"},
		},
	}
}

// Notecard API Tools
func CreateAPIValidateTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "api_validate",
		Description: "Validate a Notecard API request against the Notecard API Schema. This should be used to ensure that Notecard requests/commands are valid for use in firmware projects.",
	}
}

func CreateAPIDocsTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "api_docs",
		Description: "Get detailed documentation for a specific Notecard API. Returns comprehensive API information including parameters, descriptions, types, and usage examples. APIs may be called using 'req' or 'cmd' properties, where 'req' returns a response and 'cmd' does not. If no API is provided, returns a list of all available APIs and their descriptions. When reading descriptions, if a markdown link is provided, append 'https://dev.blues.io' to the start of the link in order to follow it.",
	}
}

// Blues Documentation Tools
func CreateDocsSearchTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "docs_search",
		Description: "Search the blues.dev documentation for answers to questions about the Notecard, cellular connectivity, GPS, power management, Notehub, and other Notecard-specific topics.",
	}
}

func CreateDocsSearchExpertTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "docs_search_expert",
		Description: "Search the blues.dev documentation and get expert analysis from an LLM Notecard specialist with deep knowledge of IoT product development and embedded systems design. This tool provides more comprehensive, contextual answers than basic search. This tool REQUIRES the client to support Sampling, use `notecard_search` if Sampling is not supported.",
	}
}
