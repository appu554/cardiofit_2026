// Package fhir provides a Google Cloud Healthcare API FHIR client for KB-9.
package fhir

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// ============================================================================
// Client Configuration
// ============================================================================

// ClientConfig holds configuration for the FHIR client.
type ClientConfig struct {
	ProjectID       string
	Location        string
	DatasetID       string
	FHIRStoreID     string
	CredentialsPath string
	Timeout         time.Duration
}

// ============================================================================
// Google Healthcare FHIR Client
// ============================================================================

// Client provides access to Google Cloud Healthcare API FHIR store.
type Client struct {
	config     ClientConfig
	baseURL    string
	httpClient *http.Client
	tokenSrc   oauth2.TokenSource
	logger     *zap.Logger
	mu         sync.RWMutex
	initialized bool
}

// NewClient creates a new Google Healthcare FHIR client.
func NewClient(cfg ClientConfig, logger *zap.Logger) *Client {
	baseURL := fmt.Sprintf(
		"https://healthcare.googleapis.com/v1/projects/%s/locations/%s/datasets/%s/fhirStores/%s/fhir",
		cfg.ProjectID,
		cfg.Location,
		cfg.DatasetID,
		cfg.FHIRStoreID,
	)

	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	return &Client{
		config:  cfg,
		baseURL: baseURL,
		logger:  logger,
	}
}

// Initialize sets up the client with proper authentication.
func (c *Client) Initialize(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.initialized {
		return nil
	}

	c.logger.Info("Initializing Google Healthcare FHIR client",
		zap.String("project", c.config.ProjectID),
		zap.String("location", c.config.Location),
		zap.String("dataset", c.config.DatasetID),
		zap.String("fhirStore", c.config.FHIRStoreID),
	)

	// Load credentials
	var creds *google.Credentials
	var err error

	if c.config.CredentialsPath != "" {
		// Load from file path
		data, err := os.ReadFile(c.config.CredentialsPath)
		if err != nil {
			return fmt.Errorf("failed to read credentials file: %w", err)
		}

		creds, err = google.CredentialsFromJSON(ctx, data, "https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			return fmt.Errorf("failed to parse credentials: %w", err)
		}
		c.logger.Info("Loaded credentials from file", zap.String("path", c.config.CredentialsPath))
	} else if credJSON := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_JSON"); credJSON != "" {
		// Load from environment variable JSON
		creds, err = google.CredentialsFromJSON(ctx, []byte(credJSON), "https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			return fmt.Errorf("failed to parse credentials from env: %w", err)
		}
		c.logger.Info("Loaded credentials from GOOGLE_APPLICATION_CREDENTIALS_JSON")
	} else {
		// Use default credentials (ADC)
		creds, err = google.FindDefaultCredentials(ctx, "https://www.googleapis.com/auth/cloud-platform")
		if err != nil {
			return fmt.Errorf("failed to find default credentials: %w", err)
		}
		c.logger.Info("Using Application Default Credentials (ADC)")
	}

	c.tokenSrc = creds.TokenSource
	c.httpClient = &http.Client{
		Timeout: c.config.Timeout,
	}
	c.initialized = true

	c.logger.Info("Google Healthcare FHIR client initialized successfully",
		zap.String("baseURL", c.baseURL),
	)

	return nil
}

// IsInitialized returns whether the client has been initialized.
func (c *Client) IsInitialized() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.initialized
}

// ============================================================================
// HTTP Request Helpers
// ============================================================================

// doRequest performs an authenticated HTTP request.
func (c *Client) doRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	if !c.IsInitialized() {
		if err := c.Initialize(ctx); err != nil {
			return nil, err
		}
	}

	// Get OAuth token
	token, err := c.tokenSrc.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to get auth token: %w", err)
	}

	// Build request URL
	reqURL := c.baseURL + path

	req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token.AccessToken)
	req.Header.Set("Accept", "application/fhir+json")
	if body != nil {
		req.Header.Set("Content-Type", "application/fhir+json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// parseResponse reads and parses a JSON response.
func (c *Client) parseResponse(resp *http.Response, target interface{}) error {
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("FHIR API error (status %d): %s", resp.StatusCode, string(bodyBytes))
	}

	if target == nil {
		return nil
	}

	return json.NewDecoder(resp.Body).Decode(target)
}

// ============================================================================
// FHIR Resource Operations
// ============================================================================

// GetResource retrieves a single FHIR resource by type and ID.
func (c *Client) GetResource(ctx context.Context, resourceType, resourceID string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/%s/%s", resourceType, resourceID)

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return nil, nil // Resource not found
	}

	var result map[string]interface{}
	if err := c.parseResponse(resp, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// SearchResources searches for FHIR resources with query parameters.
func (c *Client) SearchResources(ctx context.Context, resourceType string, params map[string]string) (*Bundle, error) {
	// Build query string
	query := url.Values{}
	for k, v := range params {
		query.Set(k, v)
	}

	path := fmt.Sprintf("/%s", resourceType)
	if len(query) > 0 {
		path += "?" + query.Encode()
	}

	resp, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var bundle Bundle
	if err := c.parseResponse(resp, &bundle); err != nil {
		return nil, err
	}

	return &bundle, nil
}

// ============================================================================
// Patient Data Aggregation
// ============================================================================

// GetPatientData retrieves all relevant clinical data for a patient.
func (c *Client) GetPatientData(ctx context.Context, patientID string, period *Period) (*PatientData, error) {
	c.logger.Info("Fetching patient data",
		zap.String("patientID", patientID),
	)

	data := &PatientData{
		FetchedAt:         time.Now(),
		MeasurementPeriod: period,
	}

	// Build date filter for observations and procedures
	dateFilter := ""
	if period != nil {
		if period.Start != "" {
			dateFilter = fmt.Sprintf("ge%s", period.Start)
		}
	}

	// Fetch patient demographics
	patientRaw, err := c.GetResource(ctx, "Patient", patientID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch patient: %w", err)
	}
	if patientRaw != nil {
		patient := parsePatient(patientRaw)
		data.Patient = patient
	}

	// Fetch conditions
	condParams := map[string]string{
		"patient": patientID,
		"clinical-status": "active,recurrence,relapse",
	}
	condBundle, err := c.SearchResources(ctx, "Condition", condParams)
	if err != nil {
		c.logger.Warn("Failed to fetch conditions", zap.Error(err))
	} else {
		data.Conditions = parseConditions(condBundle)
	}

	// Fetch observations (labs, vitals)
	obsParams := map[string]string{
		"patient": patientID,
		"_count":  "100",
	}
	if dateFilter != "" {
		obsParams["date"] = dateFilter
	}
	obsBundle, err := c.SearchResources(ctx, "Observation", obsParams)
	if err != nil {
		c.logger.Warn("Failed to fetch observations", zap.Error(err))
	} else {
		data.Observations = parseObservations(obsBundle)
	}

	// Fetch procedures
	procParams := map[string]string{
		"patient": patientID,
		"_count":  "50",
	}
	if dateFilter != "" {
		procParams["date"] = dateFilter
	}
	procBundle, err := c.SearchResources(ctx, "Procedure", procParams)
	if err != nil {
		c.logger.Warn("Failed to fetch procedures", zap.Error(err))
	} else {
		data.Procedures = parseProcedures(procBundle)
	}

	// Fetch medication requests
	medParams := map[string]string{
		"patient": patientID,
		"status":  "active,completed",
		"_count":  "50",
	}
	medBundle, err := c.SearchResources(ctx, "MedicationRequest", medParams)
	if err != nil {
		c.logger.Warn("Failed to fetch medications", zap.Error(err))
	} else {
		data.MedicationRequests = parseMedicationRequests(medBundle)
	}

	// Fetch immunizations
	immParams := map[string]string{
		"patient": patientID,
		"_count":  "50",
	}
	immBundle, err := c.SearchResources(ctx, "Immunization", immParams)
	if err != nil {
		c.logger.Warn("Failed to fetch immunizations", zap.Error(err))
	} else {
		data.Immunizations = parseImmunizations(immBundle)
	}

	// Fetch encounters
	encParams := map[string]string{
		"patient": patientID,
		"_count":  "20",
	}
	encBundle, err := c.SearchResources(ctx, "Encounter", encParams)
	if err != nil {
		c.logger.Warn("Failed to fetch encounters", zap.Error(err))
	} else {
		data.Encounters = parseEncounters(encBundle)
	}

	c.logger.Info("Patient data fetched successfully",
		zap.String("patientID", patientID),
		zap.Int("conditions", len(data.Conditions)),
		zap.Int("observations", len(data.Observations)),
		zap.Int("procedures", len(data.Procedures)),
		zap.Int("medications", len(data.MedicationRequests)),
		zap.Int("immunizations", len(data.Immunizations)),
		zap.Int("encounters", len(data.Encounters)),
	)

	return data, nil
}

// ============================================================================
// Health Check
// ============================================================================

// HealthCheck verifies connectivity to the FHIR store.
func (c *Client) HealthCheck(ctx context.Context) error {
	if !c.IsInitialized() {
		if err := c.Initialize(ctx); err != nil {
			return err
		}
	}

	// Try to search for a non-existent resource (quick connectivity check)
	params := map[string]string{
		"_count": "1",
	}
	_, err := c.SearchResources(ctx, "Patient", params)
	if err != nil {
		return fmt.Errorf("FHIR health check failed: %w", err)
	}

	return nil
}

// GetBaseURL returns the base URL of the FHIR store.
func (c *Client) GetBaseURL() string {
	return c.baseURL
}
