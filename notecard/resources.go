package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// API categories for chunked resources
var apiCategories = []string{
	"card", "hub", "note", "env", "file", "web", "var", "ntn", "dfu",
}

// CreateAPIResources creates multiple Notecard API documentation resources, one for each category
func CreateAPIResources() []mcp.Resource {
	var resources []mcp.Resource

	// Create a resource for each API category
	for _, category := range apiCategories {
		resources = append(resources, mcp.NewResource(
			fmt.Sprintf("docs://api/%s", category),
			fmt.Sprintf("Notecard %s API", strings.Title(category)),
			mcp.WithResourceDescription(fmt.Sprintf("The Notecard %s API Documentation", strings.Title(category))),
			mcp.WithMIMEType("text/markdown"),
		))
	}

	// Also create a general overview resource
	resources = append(resources, mcp.NewResource(
		"docs://api/overview",
		"Notecard API Overview",
		mcp.WithResourceDescription("Overview of the Notecard API with general information"),
		mcp.WithMIMEType("text/markdown"),
	))

	return resources
}

// fetchAPIDocumentation fetches the full API documentation from the Blues website
func fetchAPIDocumentation() (string, error) {
	resp, err := http.Get("https://blues.github.io/notecard-schema/index.md")
	if err != nil {
		return "", fmt.Errorf("failed to fetch API documentation: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch API documentation: HTTP %d", resp.StatusCode)
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read API documentation: %w", err)
	}

	return string(content), nil
}

// parseAPIDocumentation splits the full API documentation into category-specific chunks
func parseAPIDocumentation(content string) map[string]string {
	chunks := make(map[string]string)

	// Split content by API endpoints (### `endpoint`)
	endpointRegex := regexp.MustCompile(`(?m)^### ` + "`" + `([^` + "`" + `]+)` + "`")
	matches := endpointRegex.FindAllStringSubmatchIndex(content, -1)

	// Extract the header (everything before the first endpoint)
	var header string
	if len(matches) > 0 {
		header = content[:matches[0][0]]
	} else {
		header = content
	}

	// Collect all endpoints for the overview
	var allEndpoints []EndpointInfo

	// Process each endpoint
	for i, match := range matches {
		// Extract endpoint name
		endpointName := content[match[2]:match[3]]

		// Determine the category (first part before the dot)
		category := strings.Split(endpointName, ".")[0]

		// Find the content for this endpoint (from this match to the next one or end)
		var endpointContent string
		if i < len(matches)-1 {
			endpointContent = content[match[0]:matches[i+1][0]]
		} else {
			endpointContent = content[match[0]:]
		}

		// Extract description from the endpoint content
		description := extractEndpointDescription(endpointContent)
		allEndpoints = append(allEndpoints, EndpointInfo{
			Name:        endpointName,
			Category:    category,
			Description: description,
		})

		// Add to the appropriate category chunk
		if chunks[category] == "" {
			chunks[category] = header + "\n\n"
		}
		chunks[category] += endpointContent
	}

	// Create comprehensive overview with all endpoints
	chunks["overview"] = createOverview(header, allEndpoints)

	return chunks
}

// EndpointInfo holds information about an API endpoint
type EndpointInfo struct {
	Name        string
	Category    string
	Description string
}

// extractEndpointDescription extracts the description from an endpoint's content
func extractEndpointDescription(content string) string {
	lines := strings.Split(content, "\n")

	// Look for the first non-empty line after "#### Request"
	inRequest := false
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "#### Request" {
			inRequest = true
			continue
		}
		if inRequest && line != "" && !strings.HasPrefix(line, "**Parameters:**") && !strings.HasPrefix(line, "|") {
			// Clean up the description
			description := strings.TrimSpace(line)
			// Remove any trailing periods for consistency
			description = strings.TrimSuffix(description, ".")
			return description
		}
	}

	return "No description available"
}

// createOverview creates a comprehensive overview with all API endpoints
func createOverview(header string, endpoints []EndpointInfo) string {
	var overview strings.Builder

	// Add the original header
	overview.WriteString(header)
	overview.WriteString("\n\n## Available API Endpoints\n\n")
	overview.WriteString("The following API endpoints are available, organized by category:\n\n")

	// Group endpoints by category
	categoryMap := make(map[string][]EndpointInfo)
	for _, endpoint := range endpoints {
		categoryMap[endpoint.Category] = append(categoryMap[endpoint.Category], endpoint)
	}

	// Define category order and descriptions
	categoryDescriptions := map[string]string{
		"card": "Core Notecard functionality and configuration",
		"hub":  "Notehub connectivity and synchronization",
		"note": "Note management and operations",
		"env":  "Environment variable management",
		"file": "File operations and management",
		"web":  "HTTP/HTTPS web requests",
		"var":  "Variable storage and retrieval",
		"ntn":  "NTN (satellite) connectivity",
		"dfu":  "Device firmware update operations",
	}

	// Output endpoints by category in a logical order
	categoryOrder := []string{"card", "hub", "note", "env", "file", "web", "var", "ntn", "dfu"}

	for _, category := range categoryOrder {
		if endpoints, exists := categoryMap[category]; exists {
			overview.WriteString(fmt.Sprintf("### %s APIs\n", strings.Title(category)))
			if desc, hasDesc := categoryDescriptions[category]; hasDesc {
				overview.WriteString(fmt.Sprintf("*%s*\n\n", desc))
			}

			for _, endpoint := range endpoints {
				overview.WriteString(fmt.Sprintf("- **`%s`** - %s\n", endpoint.Name, endpoint.Description))
			}
			overview.WriteString("\n")
		}
	}

	overview.WriteString("## Usage\n\n")
	overview.WriteString("To access detailed documentation for any category, use the corresponding resource:\n")
	overview.WriteString("- `docs://api/card` - Card API documentation\n")
	overview.WriteString("- `docs://api/hub` - Hub API documentation\n")
	overview.WriteString("- `docs://api/note` - Note API documentation\n")
	overview.WriteString("- `docs://api/env` - Environment API documentation\n")
	overview.WriteString("- `docs://api/file` - File API documentation\n")
	overview.WriteString("- `docs://api/web` - Web API documentation\n")
	overview.WriteString("- `docs://api/var` - Variable API documentation\n")
	overview.WriteString("- `docs://api/ntn` - NTN API documentation\n")
	overview.WriteString("- `docs://api/dfu` - DFU API documentation\n")

	return overview.String()
}

// HandleAPIResource handles requests for category-specific API documentation resources
func HandleAPIResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Extract category from URI (docs://api/{category})
	uriParts := strings.Split(request.Params.URI, "/")
	if len(uriParts) < 3 {
		return nil, fmt.Errorf("invalid resource URI: %s", request.Params.URI)
	}
	category := uriParts[len(uriParts)-1]

	// Fetch the full API documentation
	fullContent, err := fetchAPIDocumentation()
	if err != nil {
		return nil, err
	}

	// Parse into chunks
	chunks := parseAPIDocumentation(fullContent)

	// Get the requested chunk
	content, exists := chunks[category]
	if !exists {
		return nil, fmt.Errorf("API category '%s' not found", category)
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "text/markdown",
			Text:     content,
		},
	}, nil
}
