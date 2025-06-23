package utils

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ExecuteArduinoCLICommand executes a arduino-cli command
func ExecuteArduinoCLICommand(args []string) (string, error) {
	cmd := exec.Command("arduino-cli", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute command: %v\nOutput: %s", err, string(output))
	}
	return string(output), nil
}

// ExecuteArduinoCLICommandWithLogger executes a arduino-cli command with logging support
// Provides enhanced logging and real-time output capture for any arduino-cli command
func ExecuteArduinoCLICommandWithLogger(args []string, logger *MCPLogger) (string, error) {
	if logger != nil {
		logger.Infof("Executing arduino-cli command: %v", args)
	}

	cmd := exec.Command("arduino-cli", args...)
	cmd.Env = os.Environ()

	// Create a user-writable temporary directory for arduino-cli operations
	// This ensures arduino-cli has a safe place to write temporary files
	var tempDir string
	var shouldCleanup bool

	homeDir, err := os.UserHomeDir()
	if err != nil {
		if logger != nil {
			logger.Warningf("Could not get user home directory: %v", err)
		}
	} else {
		// Create a temporary directory in the user's home directory
		tempDir = filepath.Join(homeDir, ".arduino-cli-temp")
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			if logger != nil {
				logger.Warningf("Could not create arduino-cli temp directory: %v", err)
			}
		} else {
			// Set TMPDIR environment variable to point to our writable directory
			cmd.Env = append(cmd.Env, "TMPDIR="+tempDir)
			shouldCleanup = true
			if logger != nil {
				logger.Debugf("Set TMPDIR to: %s", tempDir)
			}
		}
	}

	// Ensure cleanup of temporary directory after command completion
	defer func() {
		if shouldCleanup && tempDir != "" {
			if err := os.RemoveAll(tempDir); err != nil {
				if logger != nil {
					logger.Warningf("Failed to clean up temporary directory %s: %v", tempDir, err)
				}
			} else {
				if logger != nil {
					logger.Debugf("Cleaned up temporary directory: %s", tempDir)
				}
			}
		}
	}()

	// Use streaming output to capture real-time progress for all commands
	return executeWithStreamingLogging(cmd, logger, args)
}

// ExecuteNotecardCommand executes a notecard command
func ExecuteNotecardCommand(args []string) (string, error) {
	cmd := exec.Command("notecard", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute command: %v\nOutput: %s", err, string(output))
	}
	return string(output), nil
}

// ExecuteNotecardCommandWithEnv executes a notecard command with custom environment variables
func ExecuteNotecardCommandWithEnv(args []string, env []string) (string, error) {
	cmd := exec.Command("notecard", args...)
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute command: %v\nOutput: %s", err, string(output))
	}
	return string(output), nil
}

// ExecuteNotecardCommandWithLogger executes a notecard command with logging support
// Provides enhanced logging and real-time output capture for any notecard command
func ExecuteNotecardCommandWithLogger(args []string, logger *MCPLogger) (string, error) {
	if logger != nil {
		logger.Infof("Executing notecard command: %v", args)
	}

	cmd := exec.Command("notecard", args...)

	// Use streaming output to capture real-time progress for all commands
	return executeWithStreamingLogging(cmd, logger, args)
}

// executeWithStreamingLogging handles any notecard command with real-time logging
func executeWithStreamingLogging(cmd *exec.Cmd, logger *MCPLogger, args []string) (string, error) {
	// Determine command name for logging
	commandName := "notecard"
	if len(args) > 0 {
		commandName = fmt.Sprintf("notecard %s", args[0])
	}

	if logger != nil {
		logger.Info(fmt.Sprintf("Starting %s operation...", commandName))
	}

	// Get stdout and stderr pipes for real-time output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %v", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start command: %v", err)
	}

	// Channel to collect all output
	outputChan := make(chan string, 100)
	errorChan := make(chan error, 2)

	// Start goroutines to read stdout and stderr
	go readAndLogCommandOutput(stdout, "stdout", logger, outputChan, errorChan)
	go readAndLogCommandOutput(stderr, "stderr", logger, outputChan, errorChan)

	// Collect all output
	var allOutput strings.Builder
	outputDone := make(chan bool)

	go func() {
		for output := range outputChan {
			allOutput.WriteString(output)
			allOutput.WriteString("\n")
		}
		outputDone <- true
	}()

	// Wait for command to complete
	cmdErr := cmd.Wait()

	// Close output channel and wait for collection to finish
	close(outputChan)
	<-outputDone

	// Check for read errors
	select {
	case readErr := <-errorChan:
		if readErr != nil && logger != nil {
			logger.Errorf("Error reading command output: %v", readErr)
		}
	default:
	}

	if cmdErr != nil {
		if logger != nil {
			logger.Errorf("%s command failed: %v", commandName, cmdErr)
		}
		return allOutput.String(), fmt.Errorf("failed to execute command: %v\nOutput: %s", cmdErr, allOutput.String())
	}

	if logger != nil {
		logger.Info(fmt.Sprintf("%s operation completed successfully", commandName))
	}

	return allOutput.String(), nil
}

// readAndLogCommandOutput reads from a pipe and logs output for any command
func readAndLogCommandOutput(pipe interface{}, streamType string, logger *MCPLogger, outputChan chan<- string, errorChan chan<- error) {
	scanner := bufio.NewScanner(pipe.(interface{ Read([]byte) (int, error) }))

	for scanner.Scan() {
		line := scanner.Text()
		outputChan <- line

		if logger != nil {
			// Log the raw output for debugging
			logger.Debugf("%s: %s", streamType, line)

			// Log significant events based on content
			if isSignificantCommandEvent(line) {
				logger.Infof("Command output: %s", line)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		errorChan <- err
	}
}

// isSignificantCommandEvent determines if a log line represents a significant event for any command
func isSignificantCommandEvent(line string) bool {
	line = strings.ToLower(strings.TrimSpace(line))

	// Check for empty or very short lines
	if len(line) < 3 {
		return false
	}

	significantKeywords := []string{
		"error", "failed", "success", "completed", "finished",
		"starting", "progress", "warning", "info",
		"connecting", "connected", "disconnected",
		"timeout", "retry", "retrying",
		"firmware", "version", "update", "download",
		"transfer", "chunk", "bytes", "binary",
		"waiting", "restarting", "ready",
		"notecard", "response", "request", "side-loading",
	}

	for _, keyword := range significantKeywords {
		if strings.Contains(line, keyword) {
			return true
		}
	}

	// Also consider lines with JSON-like content as significant
	if (strings.Contains(line, "{") && strings.Contains(line, "}")) ||
		(strings.Contains(line, "[") && strings.Contains(line, "]")) {
		return true
	}

	// Consider lines with percentage or numeric progress
	if strings.Contains(line, "%") ||
		(strings.Contains(line, "/") && (strings.Contains(line, "mb") || strings.Contains(line, "kb") || strings.Contains(line, "bytes"))) {
		return true
	}

	return false
}
