package lib

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var globalSessionManager *SessionManager

// RequestLog holds information about a specific request
type RequestLog struct {
	Timestamp time.Time   `json:"timestamp"`
	ToolName  string      `json:"tool_name"`
	Arguments interface{} `json:"arguments"`
}

// SessionData holds session-specific data and state
type SessionData struct {
	ID           string            `json:"id"`
	CreatedAt    time.Time         `json:"created_at"`
	LastAccessed time.Time         `json:"last_accessed"`
	RequestCount int64             `json:"request_count"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	RequestLog   []RequestLog      `json:"request_log,omitempty"`
}

// SessionManager manages client sessions for the MCP server
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*SessionData
}

// NewSessionManager creates a new session manager
func NewSessionManager() *SessionManager {
	sm := &SessionManager{
		sessions: make(map[string]*SessionData),
	}

	// Set global reference
	globalSessionManager = sm

	// Start cleanup goroutine for expired sessions
	go sm.cleanupExpiredSessions()

	return sm
}

// GetSessionManager returns the global session manager
func GetSessionManager() *SessionManager {
	return globalSessionManager
}

// GetOrCreateSession retrieves an existing session or creates a new one
func (sm *SessionManager) GetOrCreateSession(sessionID string) *SessionData {
	if sessionID == "" {
		// Handle stateless sessions by returning a temporary session
		return &SessionData{
			ID:           "stateless",
			CreatedAt:    time.Now(),
			LastAccessed: time.Now(),
			RequestCount: 1, // This is the current request
			Metadata:     make(map[string]string),
			RequestLog:   make([]RequestLog, 0),
		}
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		session = &SessionData{
			ID:           sessionID,
			CreatedAt:    time.Now(),
			LastAccessed: time.Now(),
			RequestCount: 0,
			Metadata:     make(map[string]string),
			RequestLog:   make([]RequestLog, 0),
		}
		sm.sessions[sessionID] = session
		log.Printf("Session %s created", sessionID)
	} else {
		session.LastAccessed = time.Now()
	}

	session.RequestCount++
	return session
}

// GetSession retrieves an existing session
func (sm *SessionManager) GetSession(sessionID string) (*SessionData, bool) {
	if sessionID == "" {
		return nil, false
	}

	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	if exists {
		// Update last accessed time (we need to lock for write)
		sm.mu.RUnlock()
		sm.mu.Lock()
		session.LastAccessed = time.Now()
		sm.mu.Unlock()
		sm.mu.RLock()
	}
	return session, exists
}

// RemoveSession removes a session from the manager
func (sm *SessionManager) RemoveSession(sessionID string) {
	if sessionID == "" {
		return
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	if session, exists := sm.sessions[sessionID]; exists {
		// Log session exit with summary statistics
		log.Printf("Session %s exited after %d requests (duration: %v)",
			sessionID, session.RequestCount, time.Since(session.CreatedAt).Truncate(time.Second))
		delete(sm.sessions, sessionID)
	}
}

// ListSessions returns all active sessions (for debugging/monitoring)
func (sm *SessionManager) ListSessions() map[string]*SessionData {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]*SessionData, len(sm.sessions))
	for id, session := range sm.sessions {
		// Create a copy of the session data
		sessionCopy := *session
		result[id] = &sessionCopy
	}
	return result
}

// GetSessionCount returns the number of active sessions
func (sm *SessionManager) GetSessionCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.sessions)
}

// cleanupExpiredSessions periodically removes sessions that haven't been accessed recently
func (sm *SessionManager) cleanupExpiredSessions() {
	ticker := time.NewTicker(10 * time.Minute) // Cleanup every 10 minutes
	defer ticker.Stop()

	for range ticker.C {
		sm.mu.Lock()
		now := time.Now()
		expiredSessions := make([]string, 0)

		// Find sessions that haven't been accessed in the last hour
		for sessionID, session := range sm.sessions {
			if now.Sub(session.LastAccessed) > time.Hour {
				expiredSessions = append(expiredSessions, sessionID)
			}
		}

		// Remove expired sessions
		for _, sessionID := range expiredSessions {
			if session, exists := sm.sessions[sessionID]; exists {
				log.Printf("Session %s expired after %d requests (duration: %v, idle: %v)",
					sessionID, session.RequestCount,
					time.Since(session.CreatedAt).Truncate(time.Second),
					time.Since(session.LastAccessed).Truncate(time.Second))
				delete(sm.sessions, sessionID)
			}
		}

		sm.mu.Unlock()

		if len(expiredSessions) > 0 {
			log.Printf("Cleaned up %d expired sessions", len(expiredSessions))
		}
	}
}

// GetSessionIDFromRequest extracts the session ID from an MCP request
func GetSessionIDFromRequest(request *mcp.CallToolRequest) string {
	if request == nil || request.Session == nil {
		return ""
	}
	return request.Session.ID()
}

// LogSessionActivity logs session activity for monitoring
func LogSessionActivity(sessionID, toolName string, sessionData *SessionData) {
	if sessionID == "" || sessionID == "stateless" {
		log.Printf("Tool %s called (stateless session)", toolName)
	} else {
		log.Printf("Tool %s called by session %s (requests: %d)",
			toolName, sessionID, sessionData.RequestCount)
	}
}

// LogSessionActivityWithArgs logs session activity including request arguments
func LogSessionActivityWithArgs(sessionID, toolName string, sessionData *SessionData, arguments interface{}) {
	var argsStr string
	if arguments != nil {
		if argsBytes, err := json.Marshal(arguments); err == nil {
			argsStr = string(argsBytes)
		} else {
			argsStr = "<failed to marshal arguments>"
		}
	} else {
		argsStr = "<no arguments>"
	}

	if sessionID == "" || sessionID == "stateless" {
		log.Printf("Tool %s called (stateless session) with args: %s", toolName, argsStr)
	} else {
		historyCount := len(sessionData.RequestLog)
		totalRequests := sessionData.RequestCount

		// Show if we've truncated history
		if totalRequests > int64(historyCount) && historyCount == 50 {
			log.Printf("Tool %s called by session %s (total: %d requests, recent: %d stored) with args: %s",
				toolName, sessionID, totalRequests, historyCount, argsStr)
		} else {
			log.Printf("Tool %s called by session %s (requests: %d) with args: %s",
				toolName, sessionID, totalRequests, argsStr)
		}
	}
}

// AddRequestToLog adds a request to the session's request log
func (sm *SessionManager) AddRequestToLog(sessionData *SessionData, toolName string, arguments interface{}) {
	if sessionData.ID == "stateless" {
		// Don't store logs for stateless sessions
		return
	}

	// Limit the number of logged requests per session (keep last 50)
	const maxLogEntries = 50

	requestLog := RequestLog{
		Timestamp: time.Now(),
		ToolName:  toolName,
		Arguments: arguments,
	}

	// We need to lock the session manager since we're modifying session data
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Find the session in our map (sessionData might be a copy)
	if session, exists := sm.sessions[sessionData.ID]; exists {
		session.RequestLog = append(session.RequestLog, requestLog)

		// Keep only the last maxLogEntries
		if len(session.RequestLog) > maxLogEntries {
			session.RequestLog = session.RequestLog[len(session.RequestLog)-maxLogEntries:]
		}
	}
}

// TrackSession handles session tracking for a tool handler and returns the session data
func TrackSession(request *mcp.CallToolRequest, toolName string) *SessionData {
	sessionID := GetSessionIDFromRequest(request)
	sessionData := GetSessionManager().GetOrCreateSession(sessionID)

	// Capture the request arguments if available
	var arguments interface{}
	if request != nil && request.Params != nil {
		arguments = request.Params.Arguments
		GetSessionManager().AddRequestToLog(sessionData, toolName, arguments)
	}

	// Log with arguments included
	LogSessionActivityWithArgs(sessionID, toolName, sessionData, arguments)

	return sessionData
}

// GetSessionRequestHistory returns the recent request history for a session
func (sm *SessionManager) GetSessionRequestHistory(sessionID string, limit int) []RequestLog {
	if sessionID == "" || sessionID == "stateless" {
		return []RequestLog{}
	}

	sm.mu.RLock()
	defer sm.mu.RUnlock()

	session, exists := sm.sessions[sessionID]
	if !exists {
		return []RequestLog{}
	}

	// Return the last 'limit' entries
	logLen := len(session.RequestLog)
	if limit <= 0 || limit > logLen {
		limit = logLen
	}

	if limit == 0 {
		return []RequestLog{}
	}

	// Return a copy to avoid race conditions
	result := make([]RequestLog, limit)
	copy(result, session.RequestLog[logLen-limit:])
	return result
}
