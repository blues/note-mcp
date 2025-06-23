package main

import (
	"flag"
	"fmt"
	"log"

	"note-mcp/notehub/lib"
	"note-mcp/utils"

	"github.com/mark3labs/mcp-go/server"
)

var (
	envFilePath string
)

func init() {
	flag.StringVar(&envFilePath, "env", ".env", "Path to .env file")
}

func main() {
	flag.Parse()

	// Load credentials from .env file
	credentials, err := lib.GetNotehubCredentials(envFilePath)
	if err != nil {
		log.Printf("Warning: Failed to load Notehub credentials: %v", err)
	} else {
		sessionToken, err := lib.CreateSessionToken(credentials.Username, credentials.Password)
		if err != nil {
			log.Printf("Warning: Failed to create session token: %v", err)
		} else {
			lib.SessionToken = sessionToken
		}
	}

	s := server.NewMCPServer(
		"Notehub MCP",
		utils.Commit,
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)

	// Create and register Notehub tools
	projectListTool := CreateProjectListTool()
	projectCreateTool := CreateProjectCreateTool()
	projectDetailTool := CreateProjectDetailTool()
	deviceListTool := CreateDeviceListTool()
	projectEventsTool := CreateProjectEventsTool()
	checkNotefilesTool := CreateCheckNotefilesTool()
	sendNoteTool := CreateSendNoteTool()
	sendEncryptedNoteTool := CreateSendEncryptedNoteTool()
	routeListTool := CreateRouteListTool()
	routeDetailTool := CreateRouteDetailTool()
	deviceHealthLogTool := CreateDeviceHealthLogTool()
	monitorListTool := CreateMonitorListTool()
	monitorDetailTool := CreateMonitorDetailTool()
	devicePublicKeyTool := CreateDevicePublicKeyTool()
	billingAccountListTool := CreateBillingAccountListTool()
	productCreateTool := CreateProductCreateTool()
	productListTool := CreateProductListTool()
	environmentVariablesSetTool := CreateEnvironmentVariablesSetTool()
	fleetListTool := CreateFleetListTool()
	fleetGetTool := CreateFleetGetTool()
	deviceDfuHistoryTool := CreateDeviceDfuHistoryTool()
	firmwareHostUploadTool := CreateFirmwareHostUploadTool()

	// Add tool handlers from lib package
	s.AddTool(projectListTool, lib.HandleProjectListTool)
	s.AddTool(projectCreateTool, lib.HandleProjectCreateTool)
	s.AddTool(projectDetailTool, lib.HandleProjectDetailTool)
	s.AddTool(deviceListTool, lib.HandleDeviceListTool)
	s.AddTool(projectEventsTool, lib.HandleProjectEventsTool)
	s.AddTool(checkNotefilesTool, lib.HandleCheckNotefilesTool)
	s.AddTool(sendNoteTool, lib.HandleSendNoteTool)
	s.AddTool(sendEncryptedNoteTool, lib.HandleSendEncryptedNoteTool)
	s.AddTool(routeListTool, lib.HandleRouteListTool)
	s.AddTool(routeDetailTool, lib.HandleRouteDetailTool)
	s.AddTool(deviceHealthLogTool, lib.HandleDeviceHealthLogTool)
	s.AddTool(monitorListTool, lib.HandleMonitorListTool)
	s.AddTool(monitorDetailTool, lib.HandleMonitorDetailTool)
	s.AddTool(devicePublicKeyTool, lib.HandleDevicePublicKeyTool)
	s.AddTool(billingAccountListTool, lib.HandleBillingAccountListTool)
	s.AddTool(productCreateTool, lib.HandleProductCreateTool)
	s.AddTool(productListTool, lib.HandleProductListTool)
	s.AddTool(environmentVariablesSetTool, lib.HandleEnvironmentVariablesSetTool)
	s.AddTool(fleetListTool, lib.HandleFleetListTool)
	s.AddTool(fleetGetTool, lib.HandleFleetGetTool)
	s.AddTool(deviceDfuHistoryTool, lib.HandleDeviceDfuHistoryTool)
	s.AddTool(firmwareHostUploadTool, lib.HandleFirmwareHostUploadTool)

	if err := server.ServeStdio(s); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}
