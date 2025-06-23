package lib

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

// MakeNotehubAPIRequest makes an authenticated request to the Notehub API
func MakeNotehubAPIRequest(method, endpoint string, body []byte) (string, error) {
	baseURL := "https://api.notefile.net"
	url := baseURL + endpoint

	var req *http.Request
	var err error

	if body != nil {
		req, err = http.NewRequest(method, url, bytes.NewBuffer(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}

	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-SESSION-TOKEN", SessionToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Sprintf("Request failed with status %d: %s", resp.StatusCode, string(responseBody)), nil
	}

	return string(responseBody), nil
}
