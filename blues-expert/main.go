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

func withBasicAuth(handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username := os.Getenv("LOGS_AUTH_USER")
		password := os.Getenv("LOGS_AUTH_PASS")

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

	// Set the global logger for schema operations
	lib.SetGlobalLogger(logger)

	// Send initial startup log
	log.Println("Blues Expert MCP server starting up...")
	logger.Info("Blues Expert MCP server starting up...")

	// Create resources using functions from resources.go
	// APIResources := CreateAPIResources()

	// Add tools
	arduinoNotePowerManagementTool := CreateArduinoNotePowerManagementTool()
	arduinoNoteBestPracticesTool := CreateArduinoNoteBestPracticesTool()
	arduinoNoteTemplatesTool := CreateArduinoNoteTemplatesTool()
	arduinoSensorsTool := CreateArduinoSensorsTool()
	notecardRequestValidateTool := CreateNotecardRequestValidateTool()
	notecardGetAPIsTool := CreateNotecardGetAPIsTool()

	// Add Docs API resources with their handlers
	// for _, resource := range APIResources {
	// 	s.AddResource(resource, HandleAPIResource)
	// }

	// Add tool handlers with metrics instrumentation
	s.AddTool(arduinoNotePowerManagementTool, lib.InstrumentToolHandler("arduino_note_power_management", lib.HandleArduinoNotePowerManagementTool))
	s.AddTool(arduinoNoteBestPracticesTool, lib.InstrumentToolHandler("arduino_note_best_practices", lib.HandleArduinoNoteBestPracticesTool))
	s.AddTool(arduinoNoteTemplatesTool, lib.InstrumentToolHandler("arduino_note_templates", lib.HandleArduinoNoteTemplatesTool))
	s.AddTool(arduinoSensorsTool, lib.InstrumentToolHandler("arduino_sensors", lib.HandleArduinoSensorsTool))
	s.AddTool(notecardRequestValidateTool, lib.InstrumentToolHandler("notecard_request_validate", lib.HandleNotecardRequestValidateTool))
	s.AddTool(notecardGetAPIsTool, lib.InstrumentToolHandler("notecard_get_apis", lib.HandleNotecardGetAPIsTool))

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
	mux.HandleFunc("/expert/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Metrics endpoint
	mux.Handle("/expert/metrics", promhttp.Handler())

	// Logging endpoints - enabled if authentication credentials are provided
	logsAuthUser := os.Getenv("LOGS_AUTH_USER")
	logsAuthPass := os.Getenv("LOGS_AUTH_PASS")
	loggingEnabled := logsAuthUser != "" && logsAuthPass != ""
	if loggingEnabled {
		mux.HandleFunc("/expert/logs", withBasicAuth(lib.LogsHandler))
		mux.HandleFunc("/expert/logs/stream", withBasicAuth(lib.LogsStreamHandler))
		mux.HandleFunc("/expert/logs/stats", withBasicAuth(lib.LogsStatsHandler))
	}

	// Route all other /expert requests to the MCP server
	mux.HandleFunc("/expert/", func(w http.ResponseWriter, r *http.Request) {
		// Strip the /expert prefix before passing to the MCP server
		r.URL.Path = r.URL.Path[7:] // Remove "/expert" (7 characters)
		if r.URL.Path == "" {
			r.URL.Path = "/"
		}
		httpServer.ServeHTTP(w, r)
	})

	log.Printf("Starting HTTP server on port %s", port)
	log.Printf("MCP server available at /expert/")
	log.Printf("Health check at /expert/health")
	log.Printf("Metrics available at /expert/metrics")

	if loggingEnabled {
		log.Printf("Logs available at /expert/logs (requires basic auth)")
		log.Printf("Logs streaming (Loki) at /expert/logs/stream (requires basic auth)")
		log.Printf("Logs buffer stats at /expert/logs/stats (requires basic auth)")
	} else {
		log.Printf("Logging endpoints disabled (set credentials to enable)")
	}

	// Start HTTP server with our custom multiplexer
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
