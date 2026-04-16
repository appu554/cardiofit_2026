package fhir

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"kb-patient-profile/internal/config"
)

// FHIRClient wraps the Google Cloud Healthcare FHIR REST API.
// Authenticates via service account JSON and auto-refreshes OAuth2 tokens.
type FHIRClient struct {
	httpClient *http.Client
	baseURL    string
	logger     *zap.Logger
}

// NewFHIRClient creates a FHIR client authenticated with the service account
// credentials at cfg.CredentialsPath.
func NewFHIRClient(cfg config.GoogleFHIRConfig, logger *zap.Logger) (*FHIRClient, error) {
	ctx := context.Background()

	credBytes, err := readCredentials(cfg.CredentialsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials: %w", err)
	}

	creds, err := google.CredentialsFromJSON(ctx, credBytes,
		"https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	httpClient := oauth2.NewClient(ctx, creds.TokenSource)

	return &FHIRClient{
		httpClient: httpClient,
		baseURL:    cfg.BaseURL(),
		logger:     logger,
	}, nil
}

// GetPatient retrieves a FHIR Patient resource by ID.
func (c *FHIRClient) GetPatient(fhirID string) (map[string]interface{}, error) {
	return c.getResource("Patient", fhirID)
}

// GetObservation retrieves a FHIR Observation resource by ID.
func (c *FHIRClient) GetObservation(fhirID string) (map[string]interface{}, error) {
	return c.getResource("Observation", fhirID)
}

// SearchObservations searches for Observations by patient and LOINC code.
// Follows FHIR Bundle pagination to return all matching results.
func (c *FHIRClient) SearchObservations(patientID, loincCode string) ([]map[string]interface{}, error) {
	params := url.Values{
		"patient": {patientID},
		"code":    {"http://loinc.org|" + loincCode},
		"_sort":   {"-date"},
		"_count":  {"100"},
	}

	reqURL := c.baseURL + "/Observation?" + params.Encode()
	return c.fetchAllPages(reqURL, "Observation")
}

// GetMedicationRequest retrieves a FHIR MedicationRequest resource by ID.
func (c *FHIRClient) GetMedicationRequest(fhirID string) (map[string]interface{}, error) {
	return c.getResource("MedicationRequest", fhirID)
}

// UpsertCondition creates or updates a FHIR Condition resource (e.g., CKD diagnosis).
func (c *FHIRClient) UpsertCondition(condition map[string]interface{}) error {
	return c.upsertResource("Condition", condition)
}

// UpsertDetectedIssue creates or updates a FHIR DetectedIssue resource (e.g., safety alerts).
func (c *FHIRClient) UpsertDetectedIssue(issue map[string]interface{}) error {
	return c.upsertResource("DetectedIssue", issue)
}

// UpsertResource creates or updates a FHIR resource of any type.
// Phase 10 Gap 9: generic method for CommunicationRequest and future
// resource types that don't need their own typed wrapper.
func (c *FHIRClient) UpsertResource(resourceType string, resource map[string]interface{}) error {
	return c.upsertResource(resourceType, resource)
}

// SearchPatients queries FHIR Store for patients updated since the given time.
func (c *FHIRClient) SearchPatients(since time.Time) ([]map[string]interface{}, error) {
	return c.searchResourcesSince("Patient", since)
}

// SearchObservationsSince queries FHIR Store for observations updated since the given time.
func (c *FHIRClient) SearchObservationsSince(since time.Time) ([]map[string]interface{}, error) {
	return c.searchResourcesSince("Observation", since)
}

// SearchMedicationRequestsSince queries FHIR Store for medication requests updated since the given time.
func (c *FHIRClient) SearchMedicationRequestsSince(since time.Time) ([]map[string]interface{}, error) {
	return c.searchResourcesSince("MedicationRequest", since)
}

// --- internal helpers ---

func (c *FHIRClient) getResource(resourceType, fhirID string) (map[string]interface{}, error) {
	reqURL := c.baseURL + "/" + resourceType + "/" + fhirID
	body, err := c.doRequestWithRetry("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse FHIR %s: %w", resourceType, err)
	}
	return result, nil
}

func (c *FHIRClient) upsertResource(resourceType string, resource map[string]interface{}) error {
	data, err := json.Marshal(resource)
	if err != nil {
		return fmt.Errorf("failed to marshal FHIR %s: %w", resourceType, err)
	}

	var reqURL string
	method := "POST"

	if id, ok := resource["id"].(string); ok && id != "" {
		reqURL = c.baseURL + "/" + resourceType + "/" + id
		method = "PUT"
	} else {
		reqURL = c.baseURL + "/" + resourceType
	}

	_, err = c.doRequestWithRetry(method, reqURL, data)
	return err
}

func (c *FHIRClient) searchResourcesSince(resourceType string, since time.Time) ([]map[string]interface{}, error) {
	params := url.Values{
		"_lastUpdated": {"gt" + since.Format(time.RFC3339)},
		"_count":       {"100"},
	}

	reqURL := c.baseURL + "/" + resourceType + "?" + params.Encode()
	return c.fetchAllPages(reqURL, resourceType)
}

// fetchAllPages follows FHIR Bundle pagination (link.next) to collect all resources.
// Safety cap of 50 pages (5000 resources at _count=100) prevents runaway loops.
func (c *FHIRClient) fetchAllPages(initialURL, resourceType string) ([]map[string]interface{}, error) {
	const maxPages = 50
	var allResults []map[string]interface{}
	nextURL := initialURL

	for page := 0; page < maxPages && nextURL != ""; page++ {
		body, err := c.doRequestWithRetry("GET", nextURL, nil)
		if err != nil {
			return allResults, err
		}

		var bundle map[string]interface{}
		if err := json.Unmarshal(body, &bundle); err != nil {
			return allResults, fmt.Errorf("failed to parse FHIR Bundle: %w", err)
		}

		entries, _ := bundle["entry"].([]interface{})
		for _, entry := range entries {
			entryMap, ok := entry.(map[string]interface{})
			if !ok {
				continue
			}
			resource, ok := entryMap["resource"].(map[string]interface{})
			if ok {
				allResults = append(allResults, resource)
			}
		}

		// Follow pagination: look for link with relation "next"
		nextURL = extractNextLink(bundle)

		if nextURL != "" {
			c.logger.Debug("FHIR pagination: following next page",
				zap.String("resourceType", resourceType),
				zap.Int("page", page+1),
				zap.Int("totalSoFar", len(allResults)))
		}
	}

	c.logger.Info("FHIR search completed",
		zap.String("resourceType", resourceType),
		zap.Int("totalResources", len(allResults)))

	return allResults, nil
}

// extractNextLink finds the "next" pagination URL from a FHIR Bundle's link array.
func extractNextLink(bundle map[string]interface{}) string {
	links, ok := bundle["link"].([]interface{})
	if !ok {
		return ""
	}
	for _, link := range links {
		linkMap, ok := link.(map[string]interface{})
		if !ok {
			continue
		}
		if rel, _ := linkMap["relation"].(string); rel == "next" {
			if url, _ := linkMap["url"].(string); url != "" {
				return url
			}
		}
	}
	return ""
}

// doRequestWithRetry executes an HTTP request with exponential backoff (1s, 2s, 4s)
// on 429/5xx responses, up to 3 attempts.
func (c *FHIRClient) doRequestWithRetry(method, reqURL string, body []byte) ([]byte, error) {
	var lastErr error
	backoff := 1 * time.Second

	for attempt := 0; attempt < 3; attempt++ {
		var reqBody io.Reader
		if body != nil {
			reqBody = strings.NewReader(string(body))
		}

		req, err := http.NewRequest(method, reqURL, reqBody)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/fhir+json")
		req.Header.Set("Accept", "application/fhir+json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			c.logger.Warn("FHIR request failed, retrying",
				zap.String("method", method),
				zap.Int("attempt", attempt+1),
				zap.Error(err))
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return respBody, nil
		}

		if resp.StatusCode == 429 || resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("FHIR %s %s returned %d: %s", method, reqURL, resp.StatusCode, string(respBody))
			c.logger.Warn("FHIR retryable error",
				zap.String("method", method),
				zap.Int("status", resp.StatusCode),
				zap.Int("attempt", attempt+1))
			time.Sleep(backoff)
			backoff *= 2
			continue
		}

		return nil, fmt.Errorf("FHIR %s %s returned %d: %s", method, reqURL, resp.StatusCode, string(respBody))
	}

	return nil, fmt.Errorf("FHIR request failed after 3 attempts: %w", lastErr)
}

// readCredentials reads the service account JSON file.
func readCredentials(path string) ([]byte, error) {
	// Use os.ReadFile for the credentials file
	return readFile(path)
}
