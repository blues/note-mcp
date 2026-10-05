package main

import (
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"os"
	"time"

	"note-mcp/blues-expert/lib"

	"github.com/joho/godotenv"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/zerolog/log"
)

// mcpEndpointPath is the canonical Streamable HTTP endpoint for the MCP server.
const mcpEndpointPath = "/expert/mcp"

var (
	envFilePath    string
	logLevel       string
	sessionManager *lib.SessionManager
)

func init() {
	flag.StringVar(&envFilePath, "env", "", "Path to .env file to load environment variables")
	flag.StringVar(&logLevel, "log-level", "info", "Log level (trace, debug, info, warn, error, fatal, panic)")
}

// panicRecoveryMiddleware wraps an HTTP handler with panic recovery
func panicRecoveryMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Error().
					Interface("error", err).
					Str("path", r.URL.Path).
					Str("method", r.Method).
					Msg("Panic recovered in HTTP handler")

				// Return a 500 error to the client
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				errorResponse := map[string]any{
					"error": map[string]any{
						"code":    -32603,
						"message": "Internal server error: panic recovered",
					},
				}
				json.NewEncoder(w).Encode(errorResponse)
			}
		}()
		next(w, r)
	}
}

// servePublicSSEStream answers a session-less GET to the MCP endpoint with an
// open (keep-alive only) SSE stream, returning 200 instead of the go-sdk's
// default 405 "GET requires an active session".
//
// Why: some MCP clients — notably the claude.ai custom-connector flow — probe
// the endpoint with a session-less GET and treat the SDK's 405 as "this server
// requires authentication", then attempt an OAuth registration that fails
// because this server is public. Public reference servers (e.g. DeepWiki)
// answer a session-less GET with an open SSE stream, which clients accept as a
// public endpoint. This shim mirrors that behaviour.
//
// It only affects a session-less GET on the MCP endpoint. POST, DELETE, and
// session-scoped GET (the transport paths real clients and Claude Code use) are
// untouched and still handled by the SDK. No server->client messages are sent
// here — there is no session to attach them to — so this stream only carries
// keep-alive comments until the client disconnects.
func servePublicSSEStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			if _, err := io.WriteString(w, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func main() {
	flag.Parse()

	// Initialize logger with specified log level
	lib.InitLogger(logLevel)

	// Load environment variables from .env file if specified
	if envFilePath != "" {
		log.Info().Str("path", envFilePath).Msg("Loading environment variables")
		err := godotenv.Load(envFilePath)
		if err != nil {
			log.Warn().Err(err).Str("path", envFilePath).Msg("Failed to load .env file")
		}
	}

	// Initialize session manager
	sessionManager = lib.NewSessionManager()

	// Create a new MCP server
	impl := &mcp.Implementation{Name: "Blues Expert MCP", Version: serverVersion()}
	opts := &mcp.ServerOptions{
		Instructions: "This MCP server provides expert guidance on using the Blues Notecard & Notehub. When using this tool for developing firmware, use the 'firmware_entrypoint' tool to get started. Otherwise, use the 'docs_search' tool to search the Blues documentation.",
	}
	s := mcp.NewServer(impl, opts)

	// Send initial startup log
	log.Info().Str("version", serverVersion()).Msg("Blues Expert MCP server starting...")

	// Add tools
	firmwareEntrypointTool := CreateFirmwareEntrypointTool()
	firmwareBestPracticesTool := CreateFirmwareBestPracticesTool()
	apiValidateTool := CreateAPIValidateTool()
	apiDocsTool := CreateAPIDocsTool()
	docsSearchTool := CreateDocsSearchTool()

	// Add tool handlers
	mcp.AddTool(s, firmwareEntrypointTool, lib.HandleFirmwareEntrypointTool)
	mcp.AddTool(s, firmwareBestPracticesTool, lib.HandleFirmwareBestPracticesTool)
	mcp.AddTool(s, apiValidateTool, lib.HandleAPIValidateTool)
	mcp.AddTool(s, apiDocsTool, lib.HandleAPIDocsTool)
	mcp.AddTool(s, docsSearchTool, lib.HandleDocsSearchTool)

	// Get port from environment variable (AppRunner provides this)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default fallback for local development
	}

	// Create a custom HTTP multiplexer to handle both MCP and additional endpoints
	mux := http.NewServeMux()

	// Health check endpoint (AWS)
	mux.HandleFunc("/expert/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// The SDK's DNS rebinding protection rejects requests that arrive over
	// loopback with a non-localhost Host header (e.g. via a local tunnel).
	// MCP_DISABLE_LOCALHOST_PROTECTION=1 turns it off without a code change.
	disableLocalhostProtection := os.Getenv("MCP_DISABLE_LOCALHOST_PROTECTION") == "1"
	if disableLocalhostProtection {
		log.Warn().Msg("DNS rebinding (localhost) protection is disabled")
	}

	// Create StreamableHTTPHandler for MCP requests
	httpHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return s
	}, &mcp.StreamableHTTPOptions{
		DisableLocalhostProtection: disableLocalhostProtection,
	})

	// Route MCP server requests to /expert/ path with panic recovery.
	//
	// Public-server shim: answer a session-less GET to the MCP endpoint with an
	// open SSE stream rather than letting the SDK return 405 (see
	// servePublicSSEStream for why). POST, DELETE, and session-scoped GET — the
	// transport paths real MCP clients and Claude Code use — are untouched and
	// delegated to the SDK.
	mux.HandleFunc("/expert/", panicRecoveryMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == mcpEndpointPath && r.Header.Get("Mcp-Session-Id") == "" {
			servePublicSSEStream(w, r)
			return
		}
		httpHandler.ServeHTTP(w, r)
	}))

	log.Info().Str("port", port).Msg("Starting HTTP server")
	log.Info().Msg("MCP server available at /expert/")
	log.Info().Msg("Health check at /expert/health")

	// Start HTTP server with our custom multiplexer
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal().Err(err).Msg("Failed to start HTTP server")
	}
}
