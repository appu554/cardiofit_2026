package google_fhir

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"google.golang.org/api/transport"
)

// GoogleFHIRClient provides access to Google Cloud Healthcare API FHIR store
type GoogleFHIRClient struct {
	projectID       string
	location        string
	datasetID       string
	fhirStoreID     string
	credentialsPath string
	baseURL         string
	httpClient      *http.Client
	initialized     bool
}

// Config holds configuration for Google FHIR client
type Config struct {
	ProjectID       string
	Location        string
	DatasetID       string
	FHIRStoreID     string
	CredentialsPath string
}

// NewGoogleFHIRClient creates a new Google FHIR client
func NewGoogleFHIRClient(cfg *Config) *GoogleFHIRClient {
	baseURL := fmt.Sprintf(
		"https://healthcare.googleapis.com/v1/projects/%s/locations/%s/datasets/%s/fhirStores/%s/fhir",
		cfg.ProjectID,
		cfg.Location,
		cfg.DatasetID,
		cfg.FHIRStoreID,
	)

	return &GoogleFHIRClient{
		projectID:       cfg.ProjectID,
		location:        cfg.Location,
		datasetID:       cfg.DatasetID,
		fhirStoreID:     cfg.FHIRStoreID,
		credentialsPath: cfg.CredentialsPath,
		baseURL:         baseURL,
		initialized:     false,
	}
}

// Initialize sets up the Google Healthcare API client
func (c *GoogleFHIRClient) Initialize(ctx context.Context) error {
	if c.initialized {
		return nil
	}

	log.Printf("Initializing Google Healthcare API client with:")
	log.Printf("  Project ID: %s", c.projectID)
	log.Printf("  Location: %s", c.location)
	log.Printf("  Dataset ID: %s", c.datasetID)
	log.Printf("  FHIR Store ID: %s", c.fhirStoreID)
	log.Printf("  Base URL: %s", c.baseURL)

	// Set up authentication with explicit Healthcare API scopes
	scopes := []string{
		"https://www.googleapis.com/auth/cloud-platform",
		"https://www.googleapis.com/auth/cloud-healthcare",
	}

	var opts []option.ClientOption

	if c.credentialsPath != "" && fileExists(c.credentialsPath) {
		log.Printf("Loading credentials from file: %s", c.credentialsPath)
		opts = append(opts, option.WithCredentialsFile(c.credentialsPath))
		opts = append(opts, option.WithScopes(scopes...))
	} else if envCreds := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); envCreds != "" && fileExists(envCreds) {
		log.Printf("Loading credentials from GOOGLE_APPLICATION_CREDENTIALS: %s", envCreds)
		opts = append(opts, option.WithCredentialsFile(envCreds))
		opts = append(opts, option.WithScopes(scopes...))
	} else {
		log.Printf("No credentials file found, attempting to use default credentials with scopes: %v", scopes)
		// Try default credentials with explicit scopes
		creds, err := google.FindDefaultCredentials(ctx, scopes...)
		if err != nil {
			return fmt.Errorf("failed to find default credentials with Healthcare API scopes: %w", err)
		}
		opts = append(opts, option.WithCredentials(creds))
	}

	// Create HTTP client with authentication
	httpClient, _, err := transport.NewHTTPClient(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to create HTTP client: %w", err)
	}

	c.httpClient = httpClient
	c.initialized = true
	log.Printf("Successfully initialized Google Cloud Healthcare API client")
	return nil
}

// CreateResource creates a new FHIR resource
func (c *GoogleFHIRClient) CreateResource(ctx context.Context, resourceType string, resource interface{}) (map[string]interface{}, error) {
	if !c.initialized {
		if err := c.Initialize(ctx); err != nil {
			return nil, fmt.Errorf("failed to initialize client: %w", err)
		}
	}

	// Ensure resource has correct resourceType
	var resourceMap map[string]interface{}
	if m, ok := resource.(map[string]interface{}); ok {
		resourceMap = m
	} else {
		// Convert struct to map
		data, err := json.Marshal(resource)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal resource: %w", err)
		}
		if err := json.Unmarshal(data, &resourceMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal resource to map: %w", err)
		}
	}
	resourceMap["resourceType"] = resourceType

	// Create request
	url := fmt.Sprintf("%s/%s", c.baseURL, resourceType)
	body, err := json.Marshal(resourceMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/fhir+json; charset=utf-8")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// GetResource retrieves a FHIR resource by ID
func (c *GoogleFHIRClient) GetResource(ctx context.Context, resourceType string, resourceID string) (map[string]interface{}, error) {
	if !c.initialized {
		if err := c.Initialize(ctx); err != nil {
			return nil, fmt.Errorf("failed to initialize client: %w", err)
		}
	}

	// Create request
	url := fmt.Sprintf("%s/%s/%s", c.baseURL, resourceType, resourceID)
	log.Printf("Fetching %s resource from: %s", resourceType, url)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/fhir+json; charset=utf-8")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check if not found
	if resp.StatusCode == 404 {
		log.Printf("%s resource with ID %s not found", resourceType, resourceID)
		return nil, nil
	}

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors with detailed logging
	if resp.StatusCode >= 400 {
		log.Printf("Google Healthcare API error %d for %s/%s: %s", resp.StatusCode, resourceType, resourceID, string(respBody))
		if resp.StatusCode == 403 {
			return nil, fmt.Errorf("permission denied: check Healthcare API permissions and OAuth scopes. Status: %d, Response: %s", resp.StatusCode, string(respBody))
		}
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	log.Printf("Successfully retrieved %s resource with ID %s", resourceType, resourceID)
	return result, nil
}

// UpdateResource updates an existing FHIR resource
func (c *GoogleFHIRClient) UpdateResource(ctx context.Context, resourceType string, resourceID string, resource interface{}) (map[string]interface{}, error) {
	if !c.initialized {
		if err := c.Initialize(ctx); err != nil {
			return nil, fmt.Errorf("failed to initialize client: %w", err)
		}
	}

	// Ensure resource has correct resourceType and ID
	var resourceMap map[string]interface{}
	if m, ok := resource.(map[string]interface{}); ok {
		resourceMap = m
	} else {
		// Convert struct to map
		data, err := json.Marshal(resource)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal resource: %w", err)
		}
		if err := json.Unmarshal(data, &resourceMap); err != nil {
			return nil, fmt.Errorf("failed to unmarshal resource to map: %w", err)
		}
	}
	resourceMap["resourceType"] = resourceType
	resourceMap["id"] = resourceID

	// Create request
	url := fmt.Sprintf("%s/%s/%s", c.baseURL, resourceType, resourceID)
	body, err := json.Marshal(resourceMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/fhir+json; charset=utf-8")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// DeleteResource deletes a FHIR resource
func (c *GoogleFHIRClient) DeleteResource(ctx context.Context, resourceType string, resourceID string) error {
	if !c.initialized {
		if err := c.Initialize(ctx); err != nil {
			return fmt.Errorf("failed to initialize client: %w", err)
		}
	}

	// Create request
	url := fmt.Sprintf("%s/%s/%s", c.baseURL, resourceType, resourceID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check if not found (not an error for delete)
	if resp.StatusCode == 404 {
		return nil
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// SearchResources searches for FHIR resources
func (c *GoogleFHIRClient) SearchResources(ctx context.Context, resourceType string, params map[string]string) ([]map[string]interface{}, error) {
	if !c.initialized {
		if err := c.Initialize(ctx); err != nil {
			return nil, fmt.Errorf("failed to initialize client: %w", err)
		}
	}

	// Build URL with query parameters
	u, err := url.Parse(fmt.Sprintf("%s/%s", c.baseURL, resourceType))
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	q := u.Query()
	for key, value := range params {
		if value != "" {
			q.Set(key, value)
		}
	}
	u.RawQuery = q.Encode()

	// Create request
	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/fhir+json; charset=utf-8")

	// Execute request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response as FHIR Bundle
	var bundle map[string]interface{}
	if err := json.Unmarshal(respBody, &bundle); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract resources from bundle entries
	var resources []map[string]interface{}
	if entries, ok := bundle["entry"].([]interface{}); ok {
		for _, entry := range entries {
			if entryMap, ok := entry.(map[string]interface{}); ok {
				if resource, ok := entryMap["resource"].(map[string]interface{}); ok {
					resources = append(resources, resource)
				}
			}
		}
	}

	return resources, nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}