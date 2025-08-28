package lib

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	// BluesDocsAPIBaseURL is the base URL for the Blues documentation search API
	BluesDocsAPIBaseURL = "https://ragpi.blues.tools/sources/blues-docs/search"
)

// getAPIKeyFromAWS retrieves the API key from AWS Secrets Manager
func getAPIKeyFromAWS(ctx context.Context) (string, error) {
	secretName := "blues_expert_mcp_rag_pi_key"
	region := "us-east-1"

	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create Secrets Manager client
	svc := secretsmanager.NewFromConfig(cfg)

	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretName),
		VersionStage: aws.String("AWSCURRENT"), // VersionStage defaults to AWSCURRENT if unspecified
	}

	result, err := svc.GetSecretValue(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to get secret value: %w", err)
	}

	// Decrypts secret using the associated KMS key
	if result.SecretString == nil {
		return "", fmt.Errorf("secret string is nil")
	}

	// Parse JSON to extract the API key
	var secretData map[string]string
	if err := json.Unmarshal([]byte(*result.SecretString), &secretData); err != nil {
		return "", fmt.Errorf("failed to parse secret JSON: %w", err)
	}

	apiKey, exists := secretData["BLUES_DOCS_API_KEY"]
	if !exists {
		return "", fmt.Errorf("BLUES_DOCS_API_KEY not found in secret")
	}

	return apiKey, nil
}

// SearchResult represents a single search result from the API
type SearchResult struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	URL       string `json:"url"`
	CreatedAt string `json:"created_at"`
}

// SearchNotecardDocs performs a search against the Blues documentation API
func SearchNotecardDocs(ctx context.Context, request *mcp.CallToolRequest, query string) (*mcp.CallToolResult, error) {

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Build the search URL
	searchURL, err := url.Parse(BluesDocsAPIBaseURL)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Failed to parse search URL: %v", err)},
			},
			IsError: true,
		}, nil
	}

	// Add query parameter
	params := searchURL.Query()
	params.Add("query", query)
	searchURL.RawQuery = params.Encode()

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", searchURL.String(), nil)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Failed to create request: %v", err)},
			},
			IsError: true,
		}, nil
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json")

	// Get API key from AWS Secrets Manager or use Environment Variable
	if os.Getenv("BLUES_DOCS_API_KEY") != "" {
		req.Header.Set("x-api-key", os.Getenv("BLUES_DOCS_API_KEY"))
	} else {
		// Log that we're requesting permission to access the Blues documentation API
		if request != nil && request.Session != nil {
			request.Session.Log(ctx, &mcp.LoggingMessageParams{
				Level: "info",
				Data:  "Requesting access to the blues.dev documentation...",
			})
		}

		apiKey, err := getAPIKeyFromAWS(ctx)
		if err != nil {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					&mcp.TextContent{Text: fmt.Sprintf("Failed to access the blues.dev documentation API: %v", err)},
				},
				IsError: true,
			}, nil
		}
		req.Header.Set("x-api-key", apiKey)
	}

	// Log that we're making the search request
	if request != nil && request.Session != nil {
		request.Session.Log(ctx, &mcp.LoggingMessageParams{
			Level: "info",
			Data:  fmt.Sprintf("Searching the blues.dev documentation for: %s", query),
		})
	}

	// Make the request
	resp, err := client.Do(req)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Failed to make search request: %v", err)},
			},
			IsError: true,
		}, nil
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Search API returned status %d", resp.StatusCode)},
			},
			IsError: true,
		}, nil
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Failed to read response body: %v", err)},
			},
			IsError: true,
		}, nil
	}

	// Parse JSON response as array of SearchResult
	var searchResults []SearchResult
	if err := json.Unmarshal(body, &searchResults); err != nil {
		// If JSON parsing fails, return raw response
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Search Results for '%s':\n\n%s", query, string(body))},
			},
		}, nil
	}

	// Format the response
	if len(searchResults) == 0 {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("No results found for query: '%s'", query)},
			},
		}, nil
	}

	// Build formatted response
	result := fmt.Sprintf("# Search Results for '%s'\n\nFound %d result(s):\n\n", query, len(searchResults))
	for i, item := range searchResults {
		result += fmt.Sprintf("## %d. %s\n\n", i+1, item.Title)
		if item.URL != "" {
			result += fmt.Sprintf("**Source:** [%s](%s)\n\n", item.URL, item.URL)
		}

		// Clean up and format the content
		content := cleanContent(item.Content)
		result += fmt.Sprintf("**Content:**\n%s\n\n", content)

		if i < len(searchResults)-1 {
			result += "---\n\n"
		}

		result += "\n\nCheck that the query is related to the response of the search. If not, the answer may not be available in the documentation. Suggest to the user to post a question on the Blues Discourse forum, https://discuss.blues.com/."

	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: result},
		},
	}, nil
}

// cleanContent cleans up and formats the content from search results
func cleanContent(content string) string {
	// Remove excessive whitespace and normalize line breaks
	content = regexp.MustCompile(`\s+`).ReplaceAllString(content, " ")
	content = strings.TrimSpace(content)

	// Split into sentences for better readability
	sentences := regexp.MustCompile(`[.!?]\s+`).Split(content, -1)

	// Limit to first few sentences if content is very long
	if len(sentences) > 8 {
		sentences = sentences[:8]
		content = strings.Join(sentences, ". ") + "..."
	} else {
		content = strings.Join(sentences, ". ")
	}

	// Clean up any remaining formatting issues
	content = strings.ReplaceAll(content, "  ", " ")
	content = strings.ReplaceAll(content, " .", ".")

	// Ensure content doesn't end abruptly
	if !strings.HasSuffix(content, ".") && !strings.HasSuffix(content, "...") {
		content += "."
	}

	return content
}
