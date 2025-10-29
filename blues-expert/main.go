package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"note-mcp/blues-expert/lib"
	"note-mcp/utils"

	"github.com/joho/godotenv"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	envFilePath    string
	sessionManager *lib.SessionManager
)

func init() {
	flag.StringVar(&envFilePath, "env", "", "Path to .env file to load environment variables")
}

func main() {
	flag.Parse()

	// Load environment variables from .env file if specified
	if envFilePath != "" {
		log.Printf("Loading environment variables from %s", envFilePath)
		err := godotenv.Load(envFilePath)
		if err != nil {
			log.Printf("Warning: Failed to load .env file '%s': %v", envFilePath, err)
		}
	}

	// Initialize session manager
	sessionManager = lib.NewSessionManager()

	// Create a new MCP server
	impl := &mcp.Implementation{Name: "Blues Expert MCP", Version: utils.Commit}
	opts := &mcp.ServerOptions{
		Instructions: "This MCP server provides expert guidance on using the Blues Notecard & Notehub. When using this tool for developing firmware, use the 'firmware_entrypoint' tool to get started. Otherwise, use the 'docs_search' or 'docs_search_expert' tool to search the Blues documentation.",
		HasTools:     true,
	}
	s := mcp.NewServer(impl, opts)

	// Send initial startup log
	log.Println("Blues Expert MCP server starting...")

	// Add tools
	firmwareEntrypointTool := CreateFirmwareEntrypointTool()
	firmwareBestPracticesTool := CreateFirmwareBestPracticesTool()
	apiValidateTool := CreateAPIValidateTool()
	apiDocsTool := CreateAPIDocsTool()
	docsSearchTool := CreateDocsSearchTool()
	docsSearchExpertTool := CreateDocsSearchExpertTool()

	// Add tool handlers
	mcp.AddTool(s, firmwareEntrypointTool, lib.HandleFirmwareEntrypointTool)
	mcp.AddTool(s, firmwareBestPracticesTool, lib.HandleFirmwareBestPracticesTool)
	mcp.AddTool(s, apiValidateTool, lib.HandleAPIValidateTool)
	mcp.AddTool(s, apiDocsTool, lib.HandleAPIDocsTool)
	mcp.AddTool(s, docsSearchTool, lib.HandleDocsSearchTool)
	mcp.AddTool(s, docsSearchExpertTool, lib.HandleDocsSearchExpertTool)

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

	// Create StreamableHTTPHandler for MCP requests
	httpHandler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return s
	}, nil)

	// Route MCP server requests to /expert/ path
	mux.HandleFunc("/expert/", func(w http.ResponseWriter, r *http.Request) {
		httpHandler.ServeHTTP(w, r)
	})

	log.Printf("Starting HTTP server on port %s", port)
	log.Printf("MCP server available at /expert/")
	log.Printf("Health check at /expert/health")

	// Start HTTP server with our custom multiplexer
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}
