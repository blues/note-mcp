package main

import (
	"crypto/subtle"
	"flag"
	"log"
	"net/http"
	"os"

	"note-mcp/blues-expert/lib"
	"note-mcp/utils"

	"github.com/joho/godotenv"
	"github.com/mark3labs/mcp-go/server"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	envFilePath string
)

func init() {
	flag.StringVar(&envFilePath, "env", "", "Path to .env file to load environment variables")
}

// withBasicAuth wraps an HTTP handler with basic authentication
func withBasicAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := os.Getenv("LOGS_AUTH_USER")
		password := os.Getenv("LOGS_AUTH_PASS")

		// Skip auth if credentials are not configured
		if username == "" || password == "" {
			log.Printf("Warning: LOGS_AUTH_USER or LOGS_AUTH_PASS not set, logging endpoints are unprotected")
			handler(w, r)
			return
		}

		user, pass, ok := r.BasicAuth()
		if !ok {
			w.Header().Set("WWW-Authenticate", `Basic realm="Logging Endpoints"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Use constant-time comparison to prevent timing attacks
		if subtle.ConstantTimeCompare([]byte(user), []byte(username)) != 1 ||
			subtle.ConstantTimeCompare([]byte(pass), []byte(password)) != 1 {
			w.Header().Set("WWW-Authenticate", `Basic realm="Logging Endpoints"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		handler(w, r)
	}
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

	// Add tool handlers with metrics instrumentation
	s.AddTool(arduinoNotePowerManagementTool, lib.InstrumentToolHandler("arduino_note_power_management", lib.HandleArduinoNotePowerManagementTool))
	s.AddTool(arduinoNoteBestPracticesTool, lib.InstrumentToolHandler("arduino_note_best_practices", lib.HandleArduinoNoteBestPracticesTool))
	s.AddTool(arduinoNoteTemplatesTool, lib.InstrumentToolHandler("arduino_note_templates", lib.HandleArduinoNoteTemplatesTool))
	s.AddTool(arduinoCLICompileTool, lib.InstrumentToolHandler("arduino_compile", lib.HandleArduinoCLICompileTool(logger)))
	s.AddTool(arduinoCLIUploadTool, lib.InstrumentToolHandler("arduino_upload", lib.HandleArduinoCLIUploadTool(logger)))
	s.AddTool(arduinoSensorsTool, lib.InstrumentToolHandler("arduino_sensors", lib.HandleArduinoSensorsTool))

	log.Println("Blues Expert MCP server ready with logging capabilities")

	// Get port from environment variable (AppRunner provides this)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default fallback for local development
	}

	// Create StreamableHTTPServer
	httpServer := server.NewStreamableHTTPServer(s)

	// Create a custom HTTP multiplexer to handle both MCP and additional endpoints
	mux := http.NewServeMux()

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Metrics endpoint
	mux.Handle("/metrics", promhttp.Handler())

	// Logging endpoints
	loggingEnabled := os.Getenv("ENABLE_LOGGING_ENDPOINTS") != ""
	if loggingEnabled {
		mux.HandleFunc("/logs", withBasicAuth(lib.LogsHandler))
		mux.HandleFunc("/logs/stream", withBasicAuth(lib.LogsStreamHandler))
		mux.HandleFunc("/logs/stats", withBasicAuth(lib.LogsStatsHandler))
	}

	// Route all other requests to the MCP server
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" || r.URL.Path == "/metrics" {
			return
		}

		if loggingEnabled && (r.URL.Path == "/logs" || r.URL.Path == "/logs/stream" || r.URL.Path == "/logs/stats") {
			return
		}

		httpServer.ServeHTTP(w, r)
	})

	log.Printf("Starting HTTP server on port %s", port)
	log.Printf("MCP server available at /mcp")
	log.Printf("Health check at /health")
	log.Printf("Metrics available at /metrics")

	if loggingEnabled {
		log.Printf("Logs available at /logs (requires basic auth)")
		log.Printf("Logs streaming (Loki) at /logs/stream (requires basic auth)")
		log.Printf("Logs buffer stats at /logs/stats (requires basic auth)")
		log.Printf("Set LOGS_AUTH_USER and LOGS_AUTH_PASS environment variables for authentication")
	} else {
		log.Printf("Logging endpoints disabled (set ENABLE_LOGGING_ENDPOINTS to enable)")
	}

	// Start HTTP server with our custom multiplexer
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
