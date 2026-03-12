package phase2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"flow2-go-engine/internal/config"
	"flow2-go-engine/internal/models"
)

// KnowledgeBrokerClient interface defines operations for Knowledge Broker integration
type KnowledgeBrokerClient interface {
	// Version set operations
	GetActiveVersionSet(ctx context.Context, environment string) (*models.ActiveVersionSet, error)
	ValidateKBVersions(ctx context.Context, versions map[string]string) error
	GetVersionSetHistory(ctx context.Context, environment string, limit int) ([]*models.ActiveVersionSet, error)
	
	// Health and status
	HealthCheck(ctx context.Context) error
	GetServiceStatus(ctx context.Context) (*KnowledgeBrokerStatus, error)
	
	// System methods
	Close() error
}

// knowledgeBrokerClient implements the KnowledgeBrokerClient interface
type knowledgeBrokerClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *logrus.Logger
	config     config.KnowledgeBrokerConfig
}

// KnowledgeBrokerStatus represents the Knowledge Broker service status
type KnowledgeBrokerStatus struct {
	Status           string                    `json:"status"`
	Version          string                    `json:"version"`
	Environment      string                    `json:"environment"`
	ActiveVersionSet string                    `json:"active_version_set"`
	KnowledgeBases   map[string]KBStatus       `json:"knowledge_bases"`
	LastUpdated      time.Time                 `json:"last_updated"`
}

// KBStatus represents the status of a specific knowledge base
type KBStatus struct {
	Version     string    `json:"version"`
	Status      string    `json:"status"`
	LastUpdated time.Time `json:"last_updated"`
}

// VersionValidationResult represents the result of version validation
type VersionValidationResult struct {
	Valid           bool              `json:"valid"`
	InvalidVersions map[string]string `json:"invalid_versions,omitempty"`
	Warnings        []string          `json:"warnings,omitempty"`
}

// NewKnowledgeBrokerClient creates a new Knowledge Broker HTTP client
func NewKnowledgeBrokerClient(cfg config.KnowledgeBrokerConfig) (KnowledgeBrokerClient, error) {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	if cfg.URL == "" {
		return nil, fmt.Errorf("knowledge broker URL is required")
	}

	logger.WithField("url", cfg.URL).Info("Initializing Knowledge Broker client")

	// Configure HTTP client with timeouts and retry logic
	httpClient := &http.Client{
		Timeout: cfg.Timeout,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 10,
			IdleConnTimeout:     30 * time.Second,
		},
	}

	client := &knowledgeBrokerClient{
		baseURL:    cfg.URL,
		httpClient: httpClient,
		logger:     logger,
		config:     cfg,
	}

	// Test connection immediately
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.HealthCheck(ctx); err != nil {
		return nil, fmt.Errorf("knowledge broker health check failed: %w", err)
	}

	logger.Info("Successfully connected to Knowledge Broker")
	return client, nil
}

// GetActiveVersionSet retrieves the currently active version set for an environment
func (kbc *knowledgeBrokerClient) GetActiveVersionSet(ctx context.Context, environment string) (*models.ActiveVersionSet, error) {
	kbc.logger.WithField("environment", environment).Info("Fetching active version set")

	url := fmt.Sprintf("%s/api/v1/version-sets/active", kbc.baseURL)
	
	// Create request with query parameter
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add environment as query parameter
	q := req.URL.Query()
	q.Add("environment", environment)
	req.URL.RawQuery = q.Encode()

	resp, err := kbc.makeHTTPRequest(req)
	if err != nil {
		return nil, fmt.Errorf("active version set request failed: %w", err)
	}

	var versionSet models.ActiveVersionSet
	if err := json.Unmarshal(resp, &versionSet); err != nil {
		return nil, fmt.Errorf("failed to unmarshal version set: %w", err)
	}

	kbc.logger.WithFields(logrus.Fields{
		"version_set": versionSet.Name,
		"kb_count":    len(versionSet.KBVersions),
		"activated_at": versionSet.ActivatedAt,
	}).Info("✅ Active version set retrieved")

	return &versionSet, nil
}

// ValidateKBVersions validates that the provided KB versions exist and are compatible
func (kbc *knowledgeBrokerClient) ValidateKBVersions(ctx context.Context, versions map[string]string) error {
	kbc.logger.WithField("version_count", len(versions)).Info("Validating KB versions")

	url := fmt.Sprintf("%s/api/v1/version-sets/validate", kbc.baseURL)
	
	payload, err := json.Marshal(map[string]interface{}{
		"kb_versions": versions,
	})
	if err != nil {
		return fmt.Errorf("failed to marshal validation request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("failed to create validation request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := kbc.makeHTTPRequest(req)
	if err != nil {
		return fmt.Errorf("version validation request failed: %w", err)
	}

	var result VersionValidationResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return fmt.Errorf("failed to unmarshal validation result: %w", err)
	}

	if !result.Valid {
		kbc.logger.WithFields(logrus.Fields{
			"invalid_versions": result.InvalidVersions,
			"warnings":         result.Warnings,
		}).Error("❌ KB version validation failed")
		
		return fmt.Errorf("KB version validation failed: %v", result.InvalidVersions)
	}

	if len(result.Warnings) > 0 {
		kbc.logger.WithField("warnings", result.Warnings).Warn("⚠️ KB version validation warnings")
	}

	kbc.logger.Info("✅ KB versions validated successfully")
	return nil
}

// GetVersionSetHistory retrieves the history of version sets for an environment
func (kbc *knowledgeBrokerClient) GetVersionSetHistory(ctx context.Context, environment string, limit int) ([]*models.ActiveVersionSet, error) {
	kbc.logger.WithFields(logrus.Fields{
		"environment": environment,
		"limit":       limit,
	}).Info("Fetching version set history")

	url := fmt.Sprintf("%s/api/v1/version-sets/history", kbc.baseURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add query parameters
	q := req.URL.Query()
	q.Add("environment", environment)
	if limit > 0 {
		q.Add("limit", fmt.Sprintf("%d", limit))
	}
	req.URL.RawQuery = q.Encode()

	resp, err := kbc.makeHTTPRequest(req)
	if err != nil {
		return nil, fmt.Errorf("version set history request failed: %w", err)
	}

	var history []*models.ActiveVersionSet
	if err := json.Unmarshal(resp, &history); err != nil {
		return nil, fmt.Errorf("failed to unmarshal version set history: %w", err)
	}

	kbc.logger.WithField("history_count", len(history)).Info("✅ Version set history retrieved")
	return history, nil
}

// GetServiceStatus retrieves Knowledge Broker service status
func (kbc *knowledgeBrokerClient) GetServiceStatus(ctx context.Context) (*KnowledgeBrokerStatus, error) {
	kbc.logger.Info("Retrieving Knowledge Broker service status")

	url := fmt.Sprintf("%s/api/v1/status", kbc.baseURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := kbc.makeHTTPRequest(req)
	if err != nil {
		return nil, fmt.Errorf("service status request failed: %w", err)
	}

	var status KnowledgeBrokerStatus
	if err := json.Unmarshal(resp, &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal service status: %w", err)
	}

	kbc.logger.WithFields(logrus.Fields{
		"status":            status.Status,
		"active_version_set": status.ActiveVersionSet,
	}).Info("✅ Knowledge Broker service status retrieved")

	return &status, nil
}

// HealthCheck performs a health check against the Knowledge Broker
func (kbc *knowledgeBrokerClient) HealthCheck(ctx context.Context) error {
	kbc.logger.Debug("Performing Knowledge Broker health check")

	url := fmt.Sprintf("%s/health", kbc.baseURL)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := kbc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status %d", resp.StatusCode)
	}

	kbc.logger.Debug("✅ Knowledge Broker health check passed")
	return nil
}

// Close closes the HTTP client connections
func (kbc *knowledgeBrokerClient) Close() error {
	kbc.logger.Info("Closing Knowledge Broker client")
	
	// Close idle connections
	if transport, ok := kbc.httpClient.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}

	kbc.logger.Info("✅ Knowledge Broker client closed")
	return nil
}

// makeHTTPRequest is a helper method for making HTTP requests with error handling
func (kbc *knowledgeBrokerClient) makeHTTPRequest(req *http.Request) ([]byte, error) {
	// Set standard headers
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Flow2-Go-Engine/1.0")

	// Make request with retry logic
	var lastErr error
	maxRetries := 3
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		resp, err := kbc.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < maxRetries {
				kbc.logger.WithFields(logrus.Fields{
					"attempt": attempt + 1,
					"error":   err.Error(),
				}).Warn("Request failed, retrying...")
				
				// Exponential backoff
				time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
				continue
			}
			break
		}
		defer resp.Body.Close()

		// Read response body
		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = err
			if attempt < maxRetries {
				continue
			}
			break
		}

		// Check status code
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return responseBody, nil
		}

		// Handle error status codes
		var errorResp map[string]interface{}
		if json.Unmarshal(responseBody, &errorResp) == nil {
			if detail, ok := errorResp["detail"].(string); ok {
				lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, detail)
			} else {
				lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(responseBody))
			}
		} else {
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(responseBody))
		}

		// Don't retry for client errors (4xx)
		if resp.StatusCode >= 400 && resp.StatusCode < 500 {
			break
		}

		if attempt < maxRetries {
			kbc.logger.WithFields(logrus.Fields{
				"attempt":     attempt + 1,
				"status_code": resp.StatusCode,
			}).Warn("Request failed, retrying...")
			
			time.Sleep(time.Duration(attempt+1) * 100 * time.Millisecond)
		}
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", maxRetries+1, lastErr)
}