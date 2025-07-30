package lib

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Counter for total tool calls by tool name and status
	toolCallsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mcp_tool_calls_total",
			Help: "Total number of MCP tool calls by tool name and status",
		},
		[]string{"tool_name", "status"},
	)

	// Histogram for tool execution duration
	toolDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mcp_tool_duration_seconds",
			Help:    "Duration of MCP tool calls in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"tool_name"},
	)

	// Gauge for active tool calls
	activeToolCalls = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "mcp_tool_active_calls",
			Help: "Number of currently active MCP tool calls",
		},
		[]string{"tool_name"},
	)

	// Histogram for argument count
	toolArgumentCount = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mcp_tool_argument_count",
			Help:    "Number of arguments passed to MCP tools",
			Buckets: []float64{0, 1, 2, 5, 10, 20, 50},
		},
		[]string{"tool_name"},
	)

	// Histogram for argument size in bytes
	toolArgumentSize = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mcp_tool_argument_size_bytes",
			Help:    "Size of arguments passed to MCP tools in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 2, 10), // 100B to ~100KB
		},
		[]string{"tool_name"},
	)

	// Counter for parameter types (safe cardinality)
	toolParameterTypes = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mcp_tool_parameter_types_total",
			Help: "Count of parameter types used in MCP tool calls",
		},
		[]string{"tool_name", "param_type"},
	)
)

// MCPRequestLog represents a structured log entry for MCP requests
type MCPRequestLog struct {
	Timestamp time.Time              `json:"timestamp"`
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments"`
	Duration  float64                `json:"duration_seconds"`
	Status    string                 `json:"status"`
	Error     string                 `json:"error,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	UserAgent string                 `json:"user_agent,omitempty"`
}

// InMemoryLogBuffer provides thread-safe in-memory log storage with circular buffer
type InMemoryLogBuffer struct {
	logs    []MCPRequestLog
	size    int
	index   int
	count   int
	mutex   sync.RWMutex
	fileLog *log.Logger // Optional file backup
}

var (
	// Global in-memory log buffer
	logBuffer *InMemoryLogBuffer
)

func init() {
	logBuffer = initializeLogBuffer()
}

func initializeLogBuffer() *InMemoryLogBuffer {
	bufferSize := getBufferSizeFromEnv()
	logBuffer := NewInMemoryLogBuffer(bufferSize)
	setupFileLogging(logBuffer, bufferSize)
	return logBuffer
}

func getBufferSizeFromEnv() int {
	bufferSize := 1000
	if envSize := os.Getenv("MCP_LOG_BUFFER_SIZE"); envSize != "" {
		if size, err := strconv.Atoi(envSize); err == nil && size > 0 {
			bufferSize = size
		}
	}
	return bufferSize
}

func setupFileLogging(lb *InMemoryLogBuffer, bufferSize int) {
	logFileName := os.Getenv("MCP_LOG_FILE")
	if logFileName == "" {
		logFileName = "mcp_requests.jsonl"
	}

	if logFileName != "none" {
		if logFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err == nil {
			lb.fileLog = log.New(logFile, "", 0)
			log.Printf("MCP logs will be written to both memory buffer (size: %d) and file: %s", bufferSize, logFileName)
		} else {
			log.Printf("Warning: Could not open MCP log file %s: %v. Using memory-only logging.", logFileName, err)
		}
	} else {
		log.Printf("MCP logs will be stored in memory only (buffer size: %d)", bufferSize)
	}
}

func NewInMemoryLogBuffer(size int) *InMemoryLogBuffer {
	return &InMemoryLogBuffer{
		logs: make([]MCPRequestLog, size),
		size: size,
	}
}

// Add stores a log entry in the circular buffer
func (lb *InMemoryLogBuffer) Add(entry MCPRequestLog) {
	lb.mutex.Lock()
	defer lb.mutex.Unlock()

	lb.logs[lb.index] = entry
	lb.index = (lb.index + 1) % lb.size

	if lb.count < lb.size {
		lb.count++
	}

	// Optional file backup
	if lb.fileLog != nil {
		if logData, err := json.Marshal(entry); err == nil {
			lb.fileLog.Println(string(logData))
		}
	}
}

// GetRecent returns recent log entries (up to limit, filtered by since time)
func (lb *InMemoryLogBuffer) GetRecent(limit int, since time.Time) []MCPRequestLog {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()

	var result []MCPRequestLog

	// Calculate starting position (oldest entry in our buffer)
	start := lb.index
	if lb.count < lb.size {
		start = 0
	}

	// Collect logs in chronological order
	for i := 0; i < lb.count; i++ {
		pos := (start + i) % lb.size
		entry := lb.logs[pos]

		// Filter by time
		if !since.IsZero() && entry.Timestamp.Before(since) {
			continue
		}

		result = append(result, entry)
	}

	// Apply limit (return most recent entries)
	if limit > 0 && limit < len(result) {
		result = result[len(result)-limit:]
	}

	return result
}

// GetStats returns buffer statistics
func (lb *InMemoryLogBuffer) GetStats() map[string]interface{} {
	lb.mutex.RLock()
	defer lb.mutex.RUnlock()

	return map[string]interface{}{
		"buffer_size":    lb.size,
		"entries_stored": lb.count,
		"current_index":  lb.index,
		"file_backup":    lb.fileLog != nil,
	}
}

func InstrumentToolHandler(toolName string, handler server.ToolHandlerFunc) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		start := time.Now()
		activeToolCalls.WithLabelValues(toolName).Inc()
		defer activeToolCalls.WithLabelValues(toolName).Dec()

		arguments := extractArgumentsFromRequest(request)
		recordArgumentMetrics(toolName, arguments)

		result, err := handler(ctx, request)

		duration := time.Since(start).Seconds()
		status, errorMsg := getStatusAndError(err)

		recordCallMetrics(toolName, duration, status)
		logToolCall(start, toolName, arguments, duration, status, errorMsg)

		return result, err
	}
}

func recordArgumentMetrics(toolName string, arguments map[string]interface{}) {
	argCount := len(arguments)
	if argCount > 0 {
		toolArgumentCount.WithLabelValues(toolName).Observe(float64(argCount))

		if argSize := calculateArgumentsSize(arguments); argSize > 0 {
			toolArgumentSize.WithLabelValues(toolName).Observe(float64(argSize))
		}

		for _, value := range arguments {
			paramType := getParameterType(value)
			toolParameterTypes.WithLabelValues(toolName, paramType).Inc()
		}
	}
}

func getStatusAndError(err error) (string, string) {
	if err != nil {
		return "error", err.Error()
	}
	return "success", ""
}

func recordCallMetrics(toolName string, duration float64, status string) {
	toolDuration.WithLabelValues(toolName).Observe(duration)
	toolCallsTotal.WithLabelValues(toolName, status).Inc()
}

func logToolCall(start time.Time, toolName string, arguments map[string]interface{}, duration float64, status, errorMsg string) {
	logEntry := MCPRequestLog{
		Timestamp: start,
		ToolName:  toolName,
		Arguments: arguments,
		Duration:  duration,
		Status:    status,
		Error:     errorMsg,
	}
	logBuffer.Add(logEntry)
}

func calculateArgumentsSize(args map[string]interface{}) int {
	if data, err := json.Marshal(args); err == nil {
		return len(data)
	}
	return 0
}

func getParameterType(value interface{}) string {
	switch value.(type) {
	case string:
		return "string"
	case float64, int, int64, int32:
		return "number"
	case bool:
		return "boolean"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	case nil:
		return "null"
	default:
		return "unknown"
	}
}

func extractArgumentsFromRequest(request mcp.CallToolRequest) map[string]interface{} {
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return make(map[string]interface{})
	}

	var requestMap map[string]interface{}
	if err := json.Unmarshal(requestBytes, &requestMap); err != nil {
		return make(map[string]interface{})
	}

	if params, ok := requestMap["params"].(map[string]interface{}); ok {
		if args, ok := params["arguments"].(map[string]interface{}); ok {
			return args
		}
	}

	return make(map[string]interface{})
}

// LogsResponse represents the structure for the logs endpoint response
type LogsResponse struct {
	Status string          `json:"status"`
	Count  int             `json:"count"`
	Logs   []MCPRequestLog `json:"logs"`
}

// GetRecentLogs retrieves recent MCP request logs from the in-memory buffer
func GetRecentLogs(limit int, since time.Time) ([]MCPRequestLog, error) {
	return logBuffer.GetRecent(limit, since), nil
}

type LogsParams struct {
	Limit        int
	Since        time.Time
	ToolFilter   string
	StatusFilter string
}

func parseLogsParams(r *http.Request) LogsParams {
	limitStr := r.URL.Query().Get("limit")
	sinceStr := r.URL.Query().Get("since")

	limit := 100
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}

	var since time.Time
	if sinceStr != "" {
		if parsedSince, err := time.Parse(time.RFC3339, sinceStr); err == nil {
			since = parsedSince
		}
	}

	return LogsParams{
		Limit:        limit,
		Since:        since,
		ToolFilter:   r.URL.Query().Get("tool"),
		StatusFilter: r.URL.Query().Get("status"),
	}
}

func filterLogs(logs []MCPRequestLog, params LogsParams) []MCPRequestLog {
	var filtered []MCPRequestLog
	for _, logEntry := range logs {
		if params.ToolFilter != "" && logEntry.ToolName != params.ToolFilter {
			continue
		}
		if params.StatusFilter != "" && logEntry.Status != params.StatusFilter {
			continue
		}
		filtered = append(filtered, logEntry)
	}
	return filtered
}

func LogsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	params := parseLogsParams(r)
	logs, err := GetRecentLogs(params.Limit, params.Since)
	if err != nil {
		log.Printf("Error reading logs: %v", err)
		http.Error(w, `{"status":"error","message":"Failed to read logs"}`, http.StatusInternalServerError)
		return
	}

	filteredLogs := filterLogs(logs, params)
	response := LogsResponse{
		Status: "success",
		Count:  len(filteredLogs),
		Logs:   filteredLogs,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding logs response: %v", err)
		http.Error(w, `{"status":"error","message":"Failed to encode response"}`, http.StatusInternalServerError)
	}
}

func LogsStreamHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/x-ndjson")

	params := parseLogsParams(r)
	logs, err := GetRecentLogs(params.Limit, params.Since)
	if err != nil {
		log.Printf("Error reading logs: %v", err)
		http.Error(w, `{"status":"error","message":"Failed to read logs"}`, http.StatusInternalServerError)
		return
	}

	for _, logEntry := range logs {
		lokiEntry := map[string]interface{}{
			"timestamp": logEntry.Timestamp.Format(time.RFC3339Nano),
			"level":     "info",
			"message":   "MCP tool call",
			"labels": map[string]string{
				"tool_name": logEntry.ToolName,
				"status":    logEntry.Status,
				"service":   "mcp-server",
			},
			"fields": map[string]interface{}{
				"tool_name":        logEntry.ToolName,
				"arguments":        logEntry.Arguments,
				"duration_seconds": logEntry.Duration,
				"status":           logEntry.Status,
				"error":            logEntry.Error,
			},
		}

		if err := json.NewEncoder(w).Encode(lokiEntry); err != nil {
			log.Printf("Error encoding log entry: %v", err)
			break
		}
	}
}

func LogsStatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	stats := logBuffer.GetStats()
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		log.Printf("Error encoding stats response: %v", err)
		http.Error(w, `{"status":"error","message":"Failed to encode response"}`, http.StatusInternalServerError)
	}
}
