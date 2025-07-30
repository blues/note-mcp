package lib

import (
	"context"
	"embed"
	"fmt"
	"os"

	"note-mcp/utils"

	"github.com/mark3labs/mcp-go/mcp"
)

//go:embed docs/arduino/power_management.md
var powerManagementFS embed.FS

func HandleArduinoNotePowerManagementTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	content, err := powerManagementFS.ReadFile("docs/arduino/power_management.md")
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(string(content)), nil
}

//go:embed docs/arduino/best_practices.md
var bestPracticesFS embed.FS

func HandleArduinoNoteBestPracticesTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	content, err := bestPracticesFS.ReadFile("docs/arduino/best_practices.md")
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(string(content)), nil
}

//go:embed docs/arduino/templates.md
var templatesFS embed.FS

// HandleArduinoNoteTemplatesTool handles the arduino note templates tool
func HandleArduinoNoteTemplatesTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	content, err := templatesFS.ReadFile("docs/arduino/templates.md")
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(string(content)), nil
}

//go:embed docs/arduino/sensors.md
var sensorsFS embed.FS

func HandleArduinoSensorsTool(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	content, err := sensorsFS.ReadFile("docs/arduino/sensors.md")
	if err != nil {
		return nil, err
	}
	return mcp.NewToolResultText(string(content)), nil
}

func HandleArduinoCLICompileTool(logger *utils.MCPLogger) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		board := request.GetString("board", "Swan")
		ino, err := request.RequireString("ino")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid Arduino sketch file: %v", err)), nil
		}
		outputDir, err := request.RequireString("output-dir")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid output directory: %v", err)), nil
		}

		// Check if the board is valid
		switch board {
		case "Swan":
			board = "STMicroelectronics:stm32:Blues"
		case "Cygnet":
			board = "STMicroelectronics:stm32:Blues:pnum=CYGNET"
		default:
			return mcp.NewToolResultError(fmt.Sprintf("Invalid board: %s. Valid options are: Swan, Cygnet", board)), nil
		}

		// Check if the ino file exists
		if _, err := os.Stat(ino); os.IsNotExist(err) {
			return mcp.NewToolResultError(fmt.Sprintf("File not found: %s", ino)), nil
		}

		// Check if the output directory, if it doesn't exist, create it
		if _, err := os.Stat(outputDir); os.IsNotExist(err) {
			err = os.MkdirAll(outputDir, 0755)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to create output directory: %v", err)), nil
			}
		}

		// Set write permissions to the output directory
		err = os.Chmod(outputDir, 0755)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to set write permissions to the output directory: %v", err)), nil
		}

		logger.Info("Compiling Arduino project...")
		command := fmt.Sprintf("Command: arduino-cli compile --fqbn %s %s --output-dir %s --verbose", board, ino, outputDir)
		logger.Info(command)

		output, err := utils.ExecuteArduinoCLICommandWithLogger([]string{"compile", "--fqbn", board, ino, "--output-dir", outputDir, "--no-color"}, logger)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to build Arduino project: %v", err)), nil
		}
		return mcp.NewToolResultText(output), nil
	}
}

func HandleArduinoCLIUploadTool(logger *utils.MCPLogger) func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		board, err := request.RequireString("board")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid board: %v", err)), nil
		}
		ino, err := request.RequireString("ino")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Invalid Arduino sketch file: %v", err)), nil
		}
		port := request.GetString("port", "")

		switch board {
		case "Swan":
			board = "STMicroelectronics:stm32:Blues"
		case "Cygnet":
			board = "STMicroelectronics:stm32:Blues:pnum=CYGNET"
		default:
			return mcp.NewToolResultError(fmt.Sprintf("Invalid board type: %s. Valid options are: Swan, Cygnet", board)), nil
		}

		// Check if the ino file exists
		if _, err := os.Stat(ino); os.IsNotExist(err) {
			return mcp.NewToolResultError(fmt.Sprintf("File not found: %s", ino)), nil
		}

		if port != "" {
			command := fmt.Sprintf("Command: arduino-cli upload --port %s --fqbn %s %s", port, board, ino)
			logger.Info(command)
			output, err := utils.ExecuteArduinoCLICommandWithLogger([]string{"upload", "--port", port, "--fqbn", board, ino, "--no-color"}, logger)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to upload Arduino project: %v", err)), nil
			}
			return mcp.NewToolResultText(output), nil
		} else {
			command := fmt.Sprintf("Command: arduino-cli upload --fqbn %s %s", board, ino)
			logger.Info(command)
			output, err := utils.ExecuteArduinoCLICommandWithLogger([]string{"upload", "--fqbn", board, ino, "--no-color"}, logger)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to upload Arduino project: %v", err)), nil
			}
			return mcp.NewToolResultText(output), nil
		}
	}
}
