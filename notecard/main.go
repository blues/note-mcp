package main

import (
	"flag"
	"fmt"
	"log"

	"note-mcp/utils"

	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/server"
)

var (
	envFilePath string
)

func init() {
	flag.StringVar(&envFilePath, "env", "", "Path to .env file to load environment variables")
}

func main() {
	flag.Parse()

	// Load environment variables from .env file if specified
	if envFilePath != "" {
		err := godotenv.Load(envFilePath)
		if err != nil {
			log.Printf("Warning: Failed to load .env file '%s': %v", envFilePath, err)
		}
	}

	// Create a new MCP server
	s := server.NewMCPServer(
		"Notecard MCP",
		utils.Commit,
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithRecovery(),
	)

	// Create MCP logger
	logger := utils.NewMCPLogger(s, "notecard-mcp")

	// Send initial startup log
	logger.Info("Notecard MCP server starting up...")

	// Create resources using functions from resources.go
	APIResources := CreateAPIResources()

	// Add tools
	notecardInitializeTool := CreateNotecardInitializeTool()
	notecardRequestTool := CreateNotecardRequestTool()
	notecardListFirmwareVersionsTool := CreateNotecardListFirmwareVersionsTool()
	notecardUpdateFirmwareTool := CreateNotecardUpdateFirmwareTool()
	notecardValidateRequestTool := CreateNotecardValidateRequestTool()

	// Add prompts
	decryptNotePrompt := CreateDecryptNotePrompt()
	requestValidatorPrompt := CreateRequestValidatorPrompt()

	// Add API resources with their handlers
	for _, resource := range APIResources {
		s.AddResource(resource, HandleAPIResource)
	}

	// Add tool handlers
	s.AddTool(notecardInitializeTool, HandleNotecardInitializeTool)
	s.AddTool(notecardRequestTool, HandleNotecardRequestTool)
	s.AddTool(notecardListFirmwareVersionsTool, HandleNotecardListFirmwareVersionsTool)
	s.AddTool(notecardUpdateFirmwareTool, HandleNotecardUpdateFirmwareTool(logger))
	s.AddTool(notecardValidateRequestTool, HandleNotecardValidateRequestTool)

	// Add prompt handlers
	s.AddPrompt(decryptNotePrompt, HandleDecryptNotePrompt)
	s.AddPrompt(requestValidatorPrompt, HandleRequestValidatorPrompt)

	// Log that server is ready
	logger.Info("Notecard MCP server ready with logging capabilities")

	// Start the server
	if err := server.ServeStdio(s); err != nil {
		logger.Errorf("Server error: %v", err)
		fmt.Printf("Server error: %v\n", err)
	}
}
