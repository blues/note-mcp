package main

import (
	"context"
	"fmt"
	"strings"

	"note-mcp/notecard/lib"

	"github.com/mark3labs/mcp-go/mcp"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// titleCaser is used to convert strings to title case
var titleCaser = cases.Title(language.English)

// CreateAPIResources creates multiple Notecard API documentation resources, one for each category
func CreateAPIResources() []mcp.Resource {
	var resources []mcp.Resource

	// Create a resource for each API category
	for _, category := range lib.APICategories {
		resources = append(resources, mcp.NewResource(
			fmt.Sprintf("docs://api/%s", category),
			fmt.Sprintf("Notecard %s API", titleCaser.String(category)),
			mcp.WithResourceDescription(fmt.Sprintf("The Notecard %s API Documentation", titleCaser.String(category))),
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

// HandleAPIResource handles requests for category-specific API documentation resources
func HandleAPIResource(ctx context.Context, request mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	// Extract category from URI (docs://api/{category})
	uriParts := strings.Split(request.Params.URI, "/")
	if len(uriParts) < 3 {
		return nil, fmt.Errorf("invalid resource URI: %s", request.Params.URI)
	}
	category := uriParts[len(uriParts)-1]

	var content string
	var err error

	if category == "overview" {
		content, err = lib.GetAPIOverview()
	} else {
		content, err = lib.GetAPICategoryDocumentation(category)
	}

	if err != nil {
		return nil, err
	}

	return []mcp.ResourceContents{
		mcp.TextResourceContents{
			URI:      request.Params.URI,
			MIMEType: "text/markdown",
			Text:     content,
		},
	}, nil
}
