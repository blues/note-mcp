package main

import (
	"flag"
	"fmt"
	"log"

	"note-mcp/utils"

	"github.com/mark3labs/mcp-go/server"
)

var (
	envFilePath  string
	credentials  NotehubCredentials
	sessionToken string
)

func init() {
	flag.StringVar(&envFilePath, "env", ".env", "Path to .env file")
}

func main() {
	flag.Parse()

	// Load credentials from .env file
	var err error
	credentials, err = GetNotehubCredentials(envFilePath)
	if err != nil {
		log.Printf("Warning: Failed to load Notehub credentials: %v", err)
	} else {
		// Create session token on startup
		sessionToken, err = CreateSessionToken(credentials.Username, credentials.Password)
		if err != nil {
			log.Printf("Warning: Failed to create session token: %v", err)
		}
	}

	s := server.NewMCPServer(
		"Notehub MCP",
		utils.Commit,
		server.WithToolCapabilities(true),
		server.WithPromptCapabilities(true),
		// server.WithResourceCapabilities(true, true),
		server.WithRecovery(),
	)

	// Create and register Notehub tools
	projectListTool := CreateProjectListTool()
	deviceListTool := CreateDeviceListTool()
	projectEventsTool := CreateProjectEventsTool()
	sendNoteTool := CreateSendNoteTool()
	routeListTool := CreateRouteListTool()
	routeDetailTool := CreateRouteDetailTool()
	deviceHealthLogTool := CreateDeviceHealthLogTool()
	monitorListTool := CreateMonitorListTool()
	monitorDetailTool := CreateMonitorDetailTool()

	// Add tool handlers
	s.AddTool(projectListTool, HandleProjectListTool)
	s.AddTool(deviceListTool, HandleDeviceListTool)
	s.AddTool(projectEventsTool, HandleProjectEventsTool)
	s.AddTool(sendNoteTool, HandleSendNoteTool)
	s.AddTool(routeListTool, HandleRouteListTool)
	s.AddTool(routeDetailTool, HandleRouteDetailTool)
	s.AddTool(deviceHealthLogTool, HandleDeviceHealthLogTool)
	s.AddTool(monitorListTool, HandleMonitorListTool)
	s.AddTool(monitorDetailTool, HandleMonitorDetailTool)

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
