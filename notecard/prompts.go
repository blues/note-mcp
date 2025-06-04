package main

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
)

// CreateDecryptNotePrompt creates a prompt for decrypting notes using the Notecard
func CreateDecryptNotePrompt() mcp.Prompt {
	return mcp.NewPrompt("decrypt_note",
		mcp.WithPromptDescription("How to decrypt a note using the Notecard. First, get any pending notefiles with a request to the Notecard {\"req\":\"hub.sync\"}. Then, decrypt the note with the request {\"req\":\"note.get\",\"file\":\"mysecrets.qi\",\"decrypt\":true,\"delete\":true}. The note_file is the name of the note file to decrypt, e.g. 'mysecrets.qi'"),
		mcp.WithArgument("note_file",
			mcp.ArgumentDescription("The note file to decrypt"),
		),
	)
}

// HandleDecryptNotePrompt handles the decrypt note prompt
func HandleDecryptNotePrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	noteFile := request.Params.Arguments["note_file"]
	if noteFile == "" {
		noteFile = "mysecrets.qi" // Default value
	}

	return mcp.NewGetPromptResult(
		"Decrypt Note Instructions",
		[]mcp.PromptMessage{
			mcp.NewPromptMessage(
				mcp.RoleUser,
				mcp.NewTextContent(fmt.Sprintf("To decrypt the note file '%s', follow these steps:\n\n1. First, sync with the hub to get any pending notefiles:\n   {\"req\":\"hub.sync\"}\n\n2. Then decrypt and retrieve the note:\n   {\"req\":\"note.get\",\"file\": \"%s\",\"decrypt\":true,\"delete\":true}", noteFile, noteFile)),
			),
		},
	), nil
}

// CreateRequestValidatorPrompt creates a prompt for validating Notecard API requests
func CreateRequestValidatorPrompt() mcp.Prompt {
	return mcp.NewPrompt("request_validator",
		mcp.WithPromptDescription("Validate a request to the Notecard. The request is a JSON string of the request to send to the Notecard, e.g. '{\"req\":\"card.version\"}' or '{\"req\":\"card.temp\",\"minutes\":60}'. All requests are documented in the Notecard API documentation, which is provided by the MCP as a resource."),
		mcp.WithArgument("request",
			mcp.ArgumentDescription("The request to validate"),
			mcp.RequiredArgument(),
		),
	)
}

// HandleRequestValidatorPrompt handles the request validator prompt
func HandleRequestValidatorPrompt(ctx context.Context, request mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	requestToValidate := request.Params.Arguments["request"]
	if requestToValidate == "" {
		return nil, fmt.Errorf("request is required")
	}

	return mcp.NewGetPromptResult(
		"Notecard Request Validation",
		[]mcp.PromptMessage{
			mcp.NewPromptMessage(
				mcp.RoleUser,
				mcp.NewTextContent(fmt.Sprintf("Please validate this Notecard API request: %s\n\nPlease check if:\n1. The request format is valid JSON\n2. The 'req' field contains a valid API endpoint\n3. All required parameters are present\n4. Parameter types and values are correct\n5. Provide suggestions for corrections if needed", requestToValidate)),
			),
			mcp.NewPromptMessage(
				mcp.RoleAssistant,
				mcp.NewEmbeddedResource(mcp.TextResourceContents{
					URI:      "docs://api/overview",
					MIMEType: "text/markdown",
				}),
			),
		},
	), nil
}
