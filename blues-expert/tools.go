package main

import "github.com/mark3labs/mcp-go/mcp"

// Arduino Tools

func CreateArduinoNotePowerManagementTool() mcp.Tool {
	return mcp.NewTool("arduino_note_power_management",
		mcp.WithDescription("Explain how to manage power for an Arduino project that uses the Notecard. This REQUIRES a Notecarrier-F or equivalent-wired carrier board, as the host MCU's power rails are controlled by the carrier board. If you cannot confirm this, please ask the user to confirm that they are using a Notecarrier-F or equivalent-wired carrier board."),
	)
}

func CreateArduinoNoteBestPracticesTool() mcp.Tool {
	return mcp.NewTool("arduino_note_best_practices",
		mcp.WithDescription("Explain how to write an Arduino project that uses the Notecard, using the best practices for the Notecard."),
	)
}

func CreateArduinoNoteTemplatesTool() mcp.Tool {
	return mcp.NewTool("arduino_note_templates",
		mcp.WithDescription("Explain how to format a templated note for an Arduino project."),
	)
}

func CreateArduinoSensorsTool() mcp.Tool {
	return mcp.NewTool("arduino_sensors",
		mcp.WithDescription("Suggest sensors to use in an Arduino project."),
	)
}

// Notecard Tools

func CreateNotecardRequestValidateTool() mcp.Tool {
	return mcp.NewTool("notecard_request_validate",
		mcp.WithDescription("Validate a Notecard API request against the Notecard API Schema. This should be used to ensure that Notecard requests/commands are valid for use in firmware projects."),
		mcp.WithString("request",
			mcp.Required(),
			mcp.Description("The JSON string of the request to validate (e.g., '{\"req\":\"card.version\"}', '{\"req\":\"card.temp\",\"minutes\":60}')"),
		),
	)
}

func CreateNotecardGetAPIsTool() mcp.Tool {
	return mcp.NewTool("notecard_get_apis",
		mcp.WithDescription("Get detailed documentation for a specific Notecard API. Returns comprehensive API information including parameters, descriptions, types, and usage examples. APIs may be called using 'req' or 'cmd' properties, where 'req' returns a response and 'cmd' does not. If no API is provided, returns a list of available APIs."),
		mcp.WithString("api",
			mcp.Description("The specific Notecard API to get documentation for (e.g., 'card.attn', 'card.version', 'hub.status', 'note.add'). "),
		),
	)
}

func CreateNotecardSearchTool() mcp.Tool {
	return mcp.NewTool("notecard_search",
		mcp.WithDescription("Search the Blues documentation for answers to questions about the Notecard, cellular connectivity, GPS, power management, and other Blues-specific topics."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("The search query or question to find relevant documentation (e.g., 'How can I use cellular and gps at the same time?', 'Notecard power consumption', 'Troubleshooting connectivity issues')"),
		),
	)
}

func CreateNotecardSearchExpertTool() mcp.Tool {
	return mcp.NewTool("notecard_search_expert",
		mcp.WithDescription("Search the Blues documentation and get expert analysis from an AI Notecard specialist with deep knowledge of IoT product development and embedded systems design. This tool provides more comprehensive, contextual answers than basic search. This tool REQUIRES the client to support Sampling, use `notecard_search` if Sampling is not supported."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("The search query or technical question about Notecard, IoT development, or embedded systems (e.g., 'How can I optimize power consumption for a solar-powered sensor?', 'Best practices for cellular connectivity in remote locations')"),
		),
		mcp.WithString("context",
			mcp.Description("Optional additional context about your specific use case, hardware setup, or constraints to help the expert provide more targeted advice"),
		),
	)
}
