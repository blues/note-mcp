package main

import (
	"flag"
	"fmt"
	"log"

	"note-mcp/dev/lib"
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
		"Dev MCP",
		utils.Commit,
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithRecovery(),
	)

	// Create MCP logger
	logger := utils.NewMCPLogger(s, "dev-mcp")

	// Send initial startup log
	logger.Info("Dev MCP server starting up...")

	// Create resources using functions from resources.go
	// APIResources := CreateAPIResources()

	// Add tools
	arduinoNotePowerManagementTool := CreateArduinoNotePowerManagementTool()
	arduinoNoteBestPracticesTool := CreateArduinoNoteBestPracticesTool()
	arduinoNoteTemplatesTool := CreateArduinoNoteTemplatesTool()
	arduinoCLICompileTool := CreateArduinoCLICompileTool()
	arduinoCLIUploadTool := CreateArduinoCLIUploadTool()
	arduinoSensorsTool := CreateArduinoSensorsTool()

	// Add Docs API resources with their handlers
	// for _, resource := range APIResources {
	// 	s.AddResource(resource, HandleAPIResource)
	// }

	// Add tool handlers
	s.AddTool(arduinoNotePowerManagementTool, lib.HandleArduinoNotePowerManagementTool)
	s.AddTool(arduinoNoteBestPracticesTool, lib.HandleArduinoNoteBestPracticesTool)
	s.AddTool(arduinoNoteTemplatesTool, lib.HandleArduinoNoteTemplatesTool)
	s.AddTool(arduinoCLICompileTool, lib.HandleArduinoCLICompileTool(logger))
	s.AddTool(arduinoCLIUploadTool, lib.HandleArduinoCLIUploadTool(logger))
	s.AddTool(arduinoSensorsTool, lib.HandleArduinoSensorsTool)

	logger.Info("Dev MCP server ready with logging capabilities")

	if err := server.ServeStdio(s); err != nil {
		logger.Errorf("Server error: %v", err)
		fmt.Printf("Server error: %v\n", err)
	}
}
