package utils

import (
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/server"
)

// LogLevel represents different logging levels
type LogLevel string

const (
	LogLevelDebug   LogLevel = "debug"
	LogLevelInfo    LogLevel = "info"
	LogLevelWarning LogLevel = "warning"
	LogLevelError   LogLevel = "error"
)

// MCPLogger provides logging capabilities for MCP servers
type MCPLogger struct {
	server     *server.MCPServer
	loggerName string
}

// NewMCPLogger creates a new MCP logger instance
func NewMCPLogger(server *server.MCPServer, loggerName string) *MCPLogger {
	if loggerName == "" {
		loggerName = "mcp-server"
	}

	return &MCPLogger{
		server:     server,
		loggerName: loggerName,
	}
}

// SendLogNotification sends a log notification to all connected MCP clients
func (l *MCPLogger) SendLogNotification(level LogLevel, message string) error {
	if l.server == nil {
		return fmt.Errorf("MCP server not initialized")
	}

	// Create MCP-compliant log notification
	notification := map[string]interface{}{
		"level":     string(level),
		"logger":    l.loggerName,
		"data":      message,
		"timestamp": time.Now().Unix(),
	}

	// Send to all connected clients
	l.server.SendNotificationToAllClients(
		"notifications/message",
		notification,
	)
	return nil
}

// SendProgressNotification sends progress updates to MCP clients
func (l *MCPLogger) SendProgressNotification(progressToken string, progress, total float64, message string) error {
	if l.server == nil {
		return fmt.Errorf("MCP server not initialized")
	}

	notification := map[string]interface{}{
		"progressToken": progressToken,
		"progress":      progress,
		"total":         total,
		"message":       message,
		"timestamp":     time.Now().Unix(),
	}

	l.server.SendNotificationToAllClients(
		"notifications/progress",
		notification,
	)
	return nil
}

// SendCustomNotification sends a custom notification to MCP clients
func (l *MCPLogger) SendCustomNotification(notificationType string, data map[string]interface{}) error {
	if l.server == nil {
		return fmt.Errorf("MCP server not initialized")
	}

	// Add timestamp if not already present
	if _, exists := data["timestamp"]; !exists {
		data["timestamp"] = time.Now().Unix()
	}

	l.server.SendNotificationToAllClients(notificationType, data)
	return nil
}

// Debug logs a debug message
func (l *MCPLogger) Debug(message string) error {
	return l.SendLogNotification(LogLevelDebug, message)
}

// Debugf logs a formatted debug message
func (l *MCPLogger) Debugf(format string, args ...interface{}) error {
	return l.Debug(fmt.Sprintf(format, args...))
}

// Info logs an info message
func (l *MCPLogger) Info(message string) error {
	return l.SendLogNotification(LogLevelInfo, message)
}

// Infof logs a formatted info message
func (l *MCPLogger) Infof(format string, args ...interface{}) error {
	return l.Info(fmt.Sprintf(format, args...))
}

// Warning logs a warning message
func (l *MCPLogger) Warning(message string) error {
	return l.SendLogNotification(LogLevelWarning, message)
}

// Warningf logs a formatted warning message
func (l *MCPLogger) Warningf(format string, args ...interface{}) error {
	return l.Warning(fmt.Sprintf(format, args...))
}

// Error logs an error message
func (l *MCPLogger) Error(message string) error {
	return l.SendLogNotification(LogLevelError, message)
}

// Errorf logs a formatted error message
func (l *MCPLogger) Errorf(format string, args ...interface{}) error {
	return l.Error(fmt.Sprintf(format, args...))
}

// LogWithContext logs a message with additional context information
func (l *MCPLogger) LogWithContext(level LogLevel, message string, context map[string]interface{}) error {
	if l.server == nil {
		return fmt.Errorf("MCP server not initialized")
	}

	// Create enhanced notification with context
	notification := map[string]interface{}{
		"level":     string(level),
		"logger":    l.loggerName,
		"data":      message,
		"context":   context,
		"timestamp": time.Now().Unix(),
	}

	l.server.SendNotificationToAllClients(
		"notifications/message",
		notification,
	)
	return nil
}
