package main

import (
	"fmt"

	"note-mcp/utils"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
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

	// Add API resources with their handlers
	for _, resource := range APIResources {
		s.AddResource(resource, HandleAPIResource)
	}

	// Add tool handlers
	s.AddTool(notecardInitializeTool, HandleNotecardInitializeTool)
	s.AddTool(notecardRequestTool, HandleNotecardRequestTool)
	s.AddTool(notecardListFirmwareVersionsTool, HandleNotecardListFirmwareVersionsTool)
	s.AddTool(notecardUpdateFirmwareTool, HandleNotecardUpdateFirmwareTool(logger))

	// Log that server is ready
	logger.Info("Notecard MCP server ready with logging capabilities")

	// Start the server
	if err := server.ServeStdio(s); err != nil {
		logger.Errorf("Server error: %v", err)
		fmt.Printf("Server error: %v\n", err)
	}
}
