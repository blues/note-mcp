package main

import (
	"flag"
	"log"

	"note-mcp/blues-expert/lib"
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
		"Blues Expert MCP",
		utils.Commit,
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, true),
		server.WithRecovery(),
	)

	// Create MCP logger
	logger := utils.NewMCPLogger(s, "blues-expert-mcp")

	// Send initial startup log
	log.Println("Blues Expert MCP server starting up...")
	logger.Info("Blues Expert MCP server starting up...")

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

	log.Println("Blues Expert MCP server ready with logging capabilities")

	log.Println("Starting StreamableHTTP server on :8080/mcp")
	httpServer := server.NewStreamableHTTPServer(s)
	if err := httpServer.Start(":8080"); err != nil {
		log.Fatal(err)
	}
}
