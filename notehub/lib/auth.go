package lib

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/joho/godotenv"
)

// SessionToken holds the current session token for API requests
var SessionToken string

// NotehubCredentials holds the username and password for Notehub authentication
type NotehubCredentials struct {
	Username string
	Password string
}

// LoginRequest represents the login request payload for Notehub authentication
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents the response from Notehub login containing the session token
type LoginResponse struct {
	SessionToken string `json:"session_token"`
}

// GetNotehubCredentials loads credentials from .env file
func GetNotehubCredentials(envFilePath string) (NotehubCredentials, error) {
	envFile, err := godotenv.Read(envFilePath)
	if err != nil {
		return NotehubCredentials{}, fmt.Errorf("failed to read .env file: %w", err)
	}

	envFileUsername := envFile["NOTEHUB_USER"]
	envFilePassword := envFile["NOTEHUB_PASS"]

	if envFileUsername == "" {
		return NotehubCredentials{}, fmt.Errorf("NOTEHUB_USER not found in .env file")
	}

	if envFilePassword == "" {
		return NotehubCredentials{}, fmt.Errorf("NOTEHUB_PASS not found in .env file")
	}

	return NotehubCredentials{
		Username: envFileUsername,
		Password: envFilePassword,
	}, nil
}

// CreateSessionToken creates a session token using username and password
func CreateSessionToken(username, password string) (string, error) {
	loginReq := LoginRequest{
		Username: username,
		Password: password,
	}

	jsonData, err := json.Marshal(loginReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal login request: %w", err)
	}

	resp, err := http.Post("https://api.notefile.net/auth/login", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to make login request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var loginResp LoginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return loginResp.SessionToken, nil
}
