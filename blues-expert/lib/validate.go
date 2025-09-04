package lib

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/santhosh-tekuri/jsonschema/v5"
	_ "github.com/santhosh-tekuri/jsonschema/v5/httploader" // Enable HTTP/HTTPS loading
)

// schema is a cached, compiled JSON schema
var (
	schema      *jsonschema.Schema
	schemaOnce  sync.Once
	schemaErr   error
	schemaMutex sync.RWMutex
)

// cacheDir is the directory where schemas are stored
const cacheDir = "/tmp/notecard-schema/"

// Default Notecard API schema URL
const defaultSchemaURL = "https://github.com/blues/notecard-schema/releases/latest/download/notecard.api.json"

// Cache expiration duration (24 hours)
const cacheExpirationDuration = 24 * time.Hour

// CacheMetadata represents metadata for cached schema files
type CacheMetadata struct {
	FetchTime time.Time `json:"fetch_time"`
	URL       string    `json:"url"`
}

// resetSchemaWithLock safely resets the schema state for re-initialization
// This function must be called with schemaMutex write lock held
func resetSchemaWithLock() {
	schemaOnce = sync.Once{}
	schema = nil
	schemaErr = nil
}

// extractRefs recursively extracts $ref URLs from a schema
func extractRefs(schemaMap map[string]interface{}, baseURL string) []string {
	var refs []string
	if ref, ok := schemaMap["$ref"].(string); ok && strings.HasPrefix(ref, "http") {
		refs = append(refs, ref)
	}
	for _, v := range schemaMap {
		switch v := v.(type) {
		case map[string]interface{}:
			refs = append(refs, extractRefs(v, baseURL)...)
		case []interface{}:
			for _, item := range v {
				if m, ok := item.(map[string]interface{}); ok {
					refs = append(refs, extractRefs(m, baseURL)...)
				}
			}
		}
	}
	return refs
}

// fetchAndCacheSchema fetches a schema from the URL and caches it
func fetchAndCacheSchema(ctx context.Context, request *mcp.CallToolRequest, url string) (io.Reader, error) {
	// Log that we're fetching the schema
	if request != nil && request.Session != nil {
		request.Session.Log(ctx, &mcp.LoggingMessageParams{
			Level: "info",
			Data:  fmt.Sprintf("Fetching Notecard API schema from %s...", url),
		})
	}
	log.Printf("Fetching Notecard API schema from %s...", url)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch schema %s: %v", url, err)
	}
	defer resp.Body.Close()

	// Log schema download progress
	if request != nil && request.Session != nil {
		request.Session.Log(ctx, &mcp.LoggingMessageParams{
			Level: "info",
			Data:  "Schema download in progress, please wait...",
		})
	}
	log.Println("Schema download in progress, please wait...")

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch schema %s: status %d", url, resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read schema %s: %v", url, err)
	}

	// Log processing status
	if request != nil && request.Session != nil {
		request.Session.Log(ctx, &mcp.LoggingMessageParams{
			Level: "info",
			Data:  "Processing and validating schema...",
		})
	}
	log.Println("Processing and validating schema...")

	// Verify it's valid JSON before caching
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("invalid JSON schema %s: %v", url, err)
	}

	// Log caching status
	if request != nil && request.Session != nil {
		request.Session.Log(ctx, &mcp.LoggingMessageParams{
			Level: "info",
			Data:  "Caching schema for future use...",
		})
	}
	log.Println("Caching schema for future use...")

	// Save to cache
	cachePath := getCachePath(url)
	fetchTime := time.Now()
	if err := os.WriteFile(cachePath, data, 0600); err != nil {
		// Log error but continue - don't fail if we can't cache
		fmt.Fprintf(os.Stderr, "warning: failed to cache schema %s: %v\n", url, err)
	} else {
		// Save cache metadata
		if err := saveCacheMetadata(url, fetchTime); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to save cache metadata for %s: %v\n", url, err)
		}
	}

	// Log completion
	if request != nil && request.Session != nil {
		request.Session.Log(ctx, &mcp.LoggingMessageParams{
			Level: "info",
			Data:  "Schema fetch and cache completed successfully",
		})
	}
	log.Println("Schema fetch and cache completed successfully")

	return bytes.NewReader(data), nil
}

// fetchAndCacheSchemaBackground fetches a schema without MCP logging (for background operations)
func fetchAndCacheSchemaBackground(url string) (io.Reader, error) {
	return fetchAndCacheSchema(context.Background(), nil, url)
}

// formatErrorMessage formats jsonschema validation errors into user-friendly messages
func formatErrorMessage(reqType string, errUnformatted error) (err error) {
	if errUnformatted == nil {
		return nil
	}

	// Convert the error to a string
	errMsg := errUnformatted.Error()

	// Define constants
	const prefix = "jsonschema: '"
	const mid1 = "' does not validate with "
	const mid2 = ": "

	// Check if message starts with prefix
	if !strings.HasPrefix(errMsg, prefix) {
		return fmt.Errorf("invalid error message format")
	}

	// Remove prefix and split on mid1
	rest := strings.TrimPrefix(errMsg, prefix)
	parts := strings.SplitN(rest, mid1, 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid error message format")
	}

	// Extract property and remaining part
	property := parts[0]
	if len(property) > 0 {
		// As of jsonschema v5.3.1, a forward-slash is prefixed to the
		// property name. Remove it to improve readability.
		// Workaround for issue:
		// https://github.com/santhosh-tekuri/jsonschema/issues/220
		property = parts[0][1:]
	}
	remaining := parts[1]

	// Split remaining part on mid2
	finalParts := strings.SplitN(remaining, mid2, 2)
	if len(finalParts) != 2 {
		return fmt.Errorf("invalid error message format")
	}

	// Extract schema rule and error message
	// schemaRule := finalParts[0] // Not used in output, but available if needed
	errorMessage := finalParts[1]

	if len(property) > 0 {
		err = fmt.Errorf("'%s' is not valid for %s: %s", property, reqType, errorMessage)
	} else {
		err = fmt.Errorf("for '%s' %s", reqType, errorMessage)
	}

	// Return the formatted error
	return err
}

// getCachePath converts a URL to a safe file path in the cache directory
func getCachePath(url string) string {
	// Use the URL path as the filename, replacing invalid characters
	filename := strings.ReplaceAll(filepath.Base(url), string(os.PathSeparator), "_")
	return filepath.Join(cacheDir, filename)
}

// getCacheMetadataPath returns the metadata file path for a cached schema
func getCacheMetadataPath(url string) string {
	cachePath := getCachePath(url)
	return cachePath + ".meta"
}

// saveCacheMetadata saves metadata for a cached schema file
func saveCacheMetadata(url string, fetchTime time.Time) error {
	metadata := CacheMetadata{
		FetchTime: fetchTime,
		URL:       url,
	}

	metadataPath := getCacheMetadataPath(url)
	data, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal cache metadata: %v", err)
	}

	if err := os.WriteFile(metadataPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write cache metadata: %v", err)
	}

	return nil
}

// loadCacheMetadata loads metadata for a cached schema file
func loadCacheMetadata(url string) (*CacheMetadata, error) {
	metadataPath := getCacheMetadataPath(url)

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache metadata: %v", err)
	}

	var metadata CacheMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache metadata: %v", err)
	}

	return &metadata, nil
}

// isCacheExpired checks if a cached schema has expired
func isCacheExpired(url string) bool {
	metadata, err := loadCacheMetadata(url)
	if err != nil {
		// If we can't load metadata, consider it expired to force refresh
		return true
	}

	return time.Since(metadata.FetchTime) > cacheExpirationDuration
}

// initSchema compiles the schema, using cached files if available
func initSchema(url string) error {
	schemaMutex.RLock()
	currentSchema := schema
	currentErr := schemaErr
	schemaMutex.RUnlock()

	// If schema is already initialized and no error, return early
	if currentSchema != nil && currentErr == nil {
		return nil
	}

	// Use sync.Once for initialization, but protect the state with mutex
	schemaOnce.Do(func() {
		schemaMutex.Lock()
		defer schemaMutex.Unlock()

		compiler := jsonschema.NewCompiler()
		compiler.Draft = jsonschema.Draft2020

		// Ensure cache directory exists
		if err := os.MkdirAll(cacheDir, 0700); err != nil {
			schemaErr = fmt.Errorf("failed to create cache directory %s: %v", cacheDir, err)
			return
		}

		mainSchemaReader, err := loadOrFetchSchema(url)
		if err != nil {
			schemaErr = fmt.Errorf("failed to load main schema %s: %v", url, err)
			return
		}
		// Read main schema to extract $ref URLs
		mainSchemaData, err := io.ReadAll(mainSchemaReader)
		if err != nil {
			schemaErr = fmt.Errorf("failed to read main schema %s: %v", url, err)
			return
		}
		var mainSchema map[string]interface{}
		if err := json.Unmarshal(mainSchemaData, &mainSchema); err != nil {
			schemaErr = fmt.Errorf("failed to parse main schema %s: %v", url, err)
			return
		}
		// Add main schema resource
		if err := compiler.AddResource(url, bytes.NewReader(mainSchemaData)); err != nil {
			schemaErr = fmt.Errorf("failed to add main schema resource %s: %v", url, err)
			return
		}
		// Extract and cache referenced schemas
		refs := extractRefs(mainSchema, url)
		if len(refs) > 0 {
			log.Printf("Processing %d referenced schema files...", len(refs))
		}
		for i, refURL := range refs {
			log.Printf("Loading referenced schema %d/%d: %s", i+1, len(refs), filepath.Base(refURL))
			refReader, err := loadOrFetchSchema(refURL)
			if err != nil {
				schemaErr = fmt.Errorf("failed to load referenced schema %s: %v", refURL, err)
				return
			}
			if err := compiler.AddResource(refURL, refReader); err != nil {
				schemaErr = fmt.Errorf("failed to add referenced schema resource %s: %v", refURL, err)
				return
			}
		}

		schema, err = compiler.Compile(url)
		if err != nil {
			schemaErr = fmt.Errorf("failed to compile schema %s: %v", url, err)
			return
		}
	})

	schemaMutex.RLock()
	defer schemaMutex.RUnlock()
	return schemaErr
}

// loadOrFetchSchema loads a schema from cache or fetches it from the URL, caching the result
func loadOrFetchSchema(url string) (io.Reader, error) {
	cachePath := getCachePath(url)

	// Check if cache exists and is not expired
	if file, err := os.Open(cachePath); err == nil {
		defer file.Close()

		// Check if cache has expired
		if isCacheExpired(url) {
			// Cache expired: fetch fresh copy
			return fetchAndCacheSchemaBackground(url)
		}

		data, err := io.ReadAll(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read cached schema %s: %v", cachePath, err)
		}
		// Verify it's valid JSON
		var v interface{}
		if err := json.Unmarshal(data, &v); err != nil {
			// Invalid cache: proceed to fetch
			return fetchAndCacheSchemaBackground(url)
		}
		return bytes.NewReader(data), nil
	}
	// Cache miss: fetch from URL
	return fetchAndCacheSchemaBackground(url)
}

// resolveSchemaError attempts to validate against specific request schemas for better error messages
func resolveSchemaError(reqMap map[string]interface{}) (err error) {
	reqType := reqMap["req"]
	if reqType == nil {
		reqType = reqMap["cmd"]
	}
	reqTypeStr, ok := reqType.(string)
	if !ok {
		err = fmt.Errorf("request type not a string")
	} else if reqTypeStr == "" {
		err = fmt.Errorf("no request type specified")
	} else {
		// Validate against the specific request schema
		schemaPath := filepath.Join(cacheDir, reqTypeStr+".req.notecard.api.json")
		if _, err = os.Stat(schemaPath); os.IsNotExist(err) {
			err = fmt.Errorf("unknown request type: %s", reqTypeStr)
		} else if err == nil {
			var reqSchema *jsonschema.Schema
			reqSchema, err = jsonschema.Compile(schemaPath)
			if err == nil {
				err = reqSchema.Validate(reqMap)
				if err != nil {
					err = formatErrorMessage(reqTypeStr, err)
				}
			}
		}
	}

	return err
}

// ValidateNotecardRequest validates a Notecard API request against the schema
func ValidateNotecardRequest(reqMap map[string]interface{}, schemaURL string) error {
	if schemaURL == "" {
		schemaURL = defaultSchemaURL
	}

	if err := initSchema(schemaURL); err != nil {
		return fmt.Errorf("failed to initialize schema: %v", err)
	}

	// Use read lock to safely access schema for validation
	schemaMutex.RLock()
	currentSchema := schema
	schemaMutex.RUnlock()

	if currentSchema == nil {
		return fmt.Errorf("schema not initialized")
	}

	if err := currentSchema.Validate(reqMap); err != nil {
		return resolveSchemaError(reqMap)
	}

	return nil
}

// APICategory represents a category of Notecard APIs
type APICategory struct {
	Name        string     `json:"name"`
	Description string     `json:"description"`
	APIs        []APIEntry `json:"apis"`
}

// APIEntry represents a single API endpoint
type APIEntry struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Annotation  string                 `json:"annotation,omitempty"`
	Properties  map[string]APIProperty `json:"properties,omitempty"`
	Required    []string               `json:"required,omitempty"`
	Examples    []string               `json:"examples,omitempty"`
	Samples     []APISample            `json:"samples,omitempty"`
	SKUs        []string               `json:"skus,omitempty"`
	Version     string                 `json:"version,omitempty"`
	APIVersion  string                 `json:"api_version,omitempty"`
}

// APISample represents a sample usage from the schema
type APISample struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	JSON        string `json:"json"`
}

// APIProperty represents a property of an API
type APIProperty struct {
	Type            string                   `json:"type"`
	Description     string                   `json:"description"`
	Default         interface{}              `json:"default,omitempty"`
	Enum            []string                 `json:"enum,omitempty"`
	Minimum         *float64                 `json:"minimum,omitempty"`
	Maximum         *float64                 `json:"maximum,omitempty"`
	SKUs            []string                 `json:"skus,omitempty"`
	SubDescriptions []PropertySubDescription `json:"sub_descriptions,omitempty"`
}

// PropertySubDescription represents detailed descriptions for specific property values
type PropertySubDescription struct {
	Const       string   `json:"const"`
	Description string   `json:"description"`
	SKUs        []string `json:"skus,omitempty"`
}

// GetNotecardAPIs returns API documentation for a specific API or lists available APIs
func GetNotecardAPIs(ctx context.Context, request *mcp.CallToolRequest, apiName string) (*APICategory, error) {
	// Ensure schema is initialized
	if err := initSchema(defaultSchemaURL); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %v", err)
	}

	// Load cached schema files to extract API documentation
	cacheFiles, err := filepath.Glob(filepath.Join(cacheDir, "*.req.notecard.api.json"))
	if err != nil {
		return nil, fmt.Errorf("failed to find cached schema files: %v", err)
	}

	// If no cache files found, force schema initialization to populate cache
	if len(cacheFiles) == 0 {
		// Log to client that we're fetching fresh schema
		if request != nil && request.Session != nil {
			request.Session.Log(ctx, &mcp.LoggingMessageParams{
				Level: "info",
				Data:  "No cached API schema found, fetching fresh schema from remote...",
			})
		}
		log.Println("No cached API schema found, fetching fresh schema from remote...")

		// Force a fresh fetch by safely resetting the schema cache
		schemaMutex.Lock()
		resetSchemaWithLock()
		schemaMutex.Unlock()

		// Re-initialize schema which will populate the cache
		if err := initSchema(defaultSchemaURL); err != nil {
			return nil, fmt.Errorf("failed to fetch and initialize schema: %v", err)
		}

		// Try to find cache files again after initialization
		cacheFiles, err = filepath.Glob(filepath.Join(cacheDir, "*.req.notecard.api.json"))
		if err != nil {
			return nil, fmt.Errorf("failed to find cached schema files after initialization: %v", err)
		}

		if len(cacheFiles) == 0 {
			return nil, fmt.Errorf("no API documentation found even after fetching fresh schema")
		}
	}

	// If specific API requested, find and return just that API
	if apiName != "" {
		schemaFile := filepath.Join(cacheDir, apiName+".req.notecard.api.json")

		// Check if the specific API schema file exists
		if _, err := os.Stat(schemaFile); os.IsNotExist(err) {
			// Log to client that we're refreshing schema
			if request != nil && request.Session != nil {
				request.Session.Log(ctx, &mcp.LoggingMessageParams{
					Level: "info",
					Data:  fmt.Sprintf("API '%s' not found in cache, refreshing schema...", apiName),
				})
			}
			log.Printf("API '%s' not found in cache, refreshing schema...", apiName)

			// Try to refresh the cache in case the API was recently added
			schemaMutex.Lock()
			resetSchemaWithLock()
			schemaMutex.Unlock()

			if err := initSchema(defaultSchemaURL); err != nil {
				return nil, fmt.Errorf("failed to refresh schema for API '%s': %v", apiName, err)
			}

			// Check again after refresh
			if _, err := os.Stat(schemaFile); os.IsNotExist(err) {
				return nil, fmt.Errorf("API '%s' not found. Available APIs can be listed by calling this tool without the 'api' parameter", apiName)
			}
		}

		// Load and parse the specific schema file
		data, err := os.ReadFile(schemaFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read schema for API '%s': %v", apiName, err)
		}

		var schemaData map[string]interface{}
		if err := json.Unmarshal(data, &schemaData); err != nil {
			return nil, fmt.Errorf("failed to parse schema for API '%s': %v", apiName, err)
		}

		// Extract API documentation from schema
		apiEntry := extractAPIFromSchema(apiName, schemaData)
		if apiEntry == nil {
			return nil, fmt.Errorf("failed to extract documentation for API '%s'", apiName)
		}

		// Return the API entry directly, not wrapped in a category
		return &APICategory{
			Name:        apiEntry.Name,
			Description: apiEntry.Description,
			APIs:        []APIEntry{*apiEntry},
		}, nil
	}

	// No specific API requested - return list of available APIs
	var allAPIs []APIEntry

	for _, file := range cacheFiles {
		// Extract API name from filename (e.g., "card.version.req.notecard.api.json" -> "card.version")
		filename := filepath.Base(file)
		extractedAPIName := strings.TrimSuffix(filename, ".req.notecard.api.json")

		// Load and parse the schema file to get the real description
		data, err := os.ReadFile(file)
		if err != nil {
			// If we can't read the file, use a fallback description
			allAPIs = append(allAPIs, APIEntry{
				Name:        extractedAPIName,
				Description: fmt.Sprintf("Use this tool with api='%s' to get detailed documentation", extractedAPIName),
			})
			continue
		}

		var schemaData map[string]interface{}
		if err := json.Unmarshal(data, &schemaData); err != nil {
			// If we can't parse the JSON, use a fallback description
			allAPIs = append(allAPIs, APIEntry{
				Name:        extractedAPIName,
				Description: fmt.Sprintf("Use this tool with api='%s' to get detailed documentation", extractedAPIName),
			})
			continue
		}

		// Extract the real description from the schema
		description := fmt.Sprintf("Use this tool with api='%s' to get detailed documentation", extractedAPIName)
		if desc, ok := schemaData["description"].(string); ok && desc != "" {
			description = desc
		}

		allAPIs = append(allAPIs, APIEntry{
			Name:        extractedAPIName,
			Description: description,
		})
	}

	if len(allAPIs) == 0 {
		return nil, fmt.Errorf("no API documentation found in cache")
	}

	return &APICategory{
		Name:        "available_apis",
		Description: fmt.Sprintf("Found %d available Notecard APIs. Use the 'api' parameter with any of these names to get detailed documentation.", len(allAPIs)),
		APIs:        allAPIs,
	}, nil
}

// extractAPIFromSchema extracts API documentation from a JSON schema
func extractAPIFromSchema(apiName string, schemaData map[string]interface{}) *APIEntry {
	entry := &APIEntry{
		Name:       apiName,
		Properties: make(map[string]APIProperty),
	}

	// Extract description
	if desc, ok := schemaData["description"].(string); ok {
		entry.Description = desc
	}

	// Extract annotation from annotations array
	if annotations, ok := schemaData["annotations"].([]interface{}); ok {
		if len(annotations) > 0 {
			if annotationObj, ok := annotations[0].(map[string]interface{}); ok {
				if desc, ok := annotationObj["description"].(string); ok {
					entry.Annotation = desc
				}
			}
		}
	}

	// Extract root-level SKUs
	if skus, ok := schemaData["skus"].([]interface{}); ok {
		for _, sku := range skus {
			if s, ok := sku.(string); ok {
				entry.SKUs = append(entry.SKUs, s)
			}
		}
	}

	// Extract version information
	if version, ok := schemaData["version"].(string); ok {
		entry.Version = version
	}
	if apiVersion, ok := schemaData["apiVersion"].(string); ok {
		entry.APIVersion = apiVersion
	}

	// Extract properties (excluding implicit req/cmd properties)
	if props, ok := schemaData["properties"].(map[string]interface{}); ok {
		for propName, propData := range props {
			// Skip implicit req/cmd properties as they're always required and match the API name
			if propName == "req" || propName == "cmd" {
				continue
			}

			if propMap, ok := propData.(map[string]interface{}); ok {
				property := APIProperty{}

				if propType, ok := propMap["type"].(string); ok {
					property.Type = propType
				}
				if propDesc, ok := propMap["description"].(string); ok {
					property.Description = propDesc
				}
				if propDefault, ok := propMap["default"]; ok {
					property.Default = propDefault
				}
				if propEnum, ok := propMap["enum"].([]interface{}); ok {
					for _, e := range propEnum {
						if s, ok := e.(string); ok {
							property.Enum = append(property.Enum, s)
						}
					}
				}
				if propMin, ok := propMap["minimum"].(float64); ok {
					property.Minimum = &propMin
				}
				if propMax, ok := propMap["maximum"].(float64); ok {
					property.Maximum = &propMax
				}
				// Extract property-level SKUs
				if propSKUs, ok := propMap["skus"].([]interface{}); ok {
					for _, sku := range propSKUs {
						if s, ok := sku.(string); ok {
							property.SKUs = append(property.SKUs, s)
						}
					}
				}
				if subDescs, ok := propMap["sub-descriptions"].([]interface{}); ok {
					for _, subDescInterface := range subDescs {
						if subDescMap, ok := subDescInterface.(map[string]interface{}); ok {
							subDesc := PropertySubDescription{}

							if constVal, ok := subDescMap["const"].(string); ok {
								subDesc.Const = constVal
							}
							if desc, ok := subDescMap["description"].(string); ok {
								subDesc.Description = desc
							}
							if skusInterface, ok := subDescMap["skus"].([]interface{}); ok {
								for _, skuInterface := range skusInterface {
									if sku, ok := skuInterface.(string); ok {
										subDesc.SKUs = append(subDesc.SKUs, sku)
									}
								}
							}

							property.SubDescriptions = append(property.SubDescriptions, subDesc)
						}
					}
				}

				entry.Properties[propName] = property
			}
		}
	}

	// Extract required fields (excluding implicit req/cmd properties)
	if required, ok := schemaData["required"].([]interface{}); ok {
		for _, req := range required {
			if s, ok := req.(string); ok {
				// Skip implicit req/cmd properties as they're always required
				if s == "req" || s == "cmd" {
					continue
				}
				entry.Required = append(entry.Required, s)
			}
		}
	}

	// Extract samples if available
	if samples, ok := schemaData["samples"].([]interface{}); ok {
		for _, sampleInterface := range samples {
			if sampleMap, ok := sampleInterface.(map[string]interface{}); ok {
				sample := APISample{}

				if title, ok := sampleMap["title"].(string); ok {
					sample.Title = title
				}
				if desc, ok := sampleMap["description"].(string); ok {
					sample.Description = desc
				}
				if jsonStr, ok := sampleMap["json"].(string); ok {
					sample.JSON = jsonStr
				}

				entry.Samples = append(entry.Samples, sample)
			}
		}
	}

	// Add basic example usage if no samples were found
	if len(entry.Samples) == 0 {
		entry.Examples = []string{
			fmt.Sprintf(`{"req":"%s"}`, apiName),
		}
	}

	return entry
}
