package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"note-mcp/notecard/lib"
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
	decryptNoteTool := CreateDecryptNoteTool()
	provisionNotecardTool := CreateProvisionNotecardTool()
	troubleshootConnectionTool := CreateTroubleshootConnectionTool()
	sendNoteTool := CreateSendNoteTool()
	notecardGetAPIsTool := CreateNotecardGetAPIsTool()

	// Add Docs API resources with their handlers
	for _, resource := range APIResources {
		s.AddResource(resource, HandleAPIResource)
	}

	// Add tool handlers
	s.AddTool(notecardInitializeTool, lib.HandleNotecardInitializeTool)
	s.AddTool(notecardRequestTool, lib.HandleNotecardRequestTool)
	s.AddTool(notecardListFirmwareVersionsTool, lib.HandleNotecardListFirmwareVersionsTool)
	s.AddTool(notecardUpdateFirmwareTool, lib.HandleNotecardUpdateFirmwareTool(logger))
	s.AddTool(decryptNoteTool, lib.HandleDecryptNoteTool)
	s.AddTool(provisionNotecardTool, lib.HandleProvisionNotecardTool)
	s.AddTool(troubleshootConnectionTool, lib.HandleTroubleshootConnectionTool)
	s.AddTool(sendNoteTool, lib.HandleSendNoteTool)
	s.AddTool(notecardGetAPIsTool, lib.HandleNotecardGetAPIsTool)

	// Add tools hidden behind BLUES environment variable
	if _, ok := os.LookupEnv("BLUES"); ok {
		notecardValidateRequestTool := CreateNotecardValidateRequestTool()
		s.AddTool(notecardValidateRequestTool, lib.HandleNotecardValidateRequestTool)
	}

	logger.Info("Notecard MCP server ready with logging capabilities")

	if err := server.ServeStdio(s); err != nil {
		logger.Errorf("Server error: %v", err)
		fmt.Printf("Server error: %v\n", err)
	}
}
