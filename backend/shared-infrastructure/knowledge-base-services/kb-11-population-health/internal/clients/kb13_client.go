// Package clients provides HTTP clients for external services.
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// KB13Client provides integration with KB-13 Care Gap Identification Service.
// IMPORTANT: KB-11 CONSUMES care gap data from KB-13 (READ-ONLY).
// We only fetch aggregated care gap counts, not individual gap details.
type KB13Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *logrus.Entry
}

// NewKB13Client creates a new KB-13 client.
func NewKB13Client(baseURL string, logger *logrus.Entry) *KB13Client {
	return &KB13Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger.WithField("client", "kb-13"),
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Care Gap Data Models
// ──────────────────────────────────────────────────────────────────────────────

// PatientCareGapSummary contains aggregated care gap info for a patient.
// NOTE: KB-11 only needs counts, NOT detailed gap information.
type PatientCareGapSummary struct {
	PatientFHIRID   string    `json:"patient_fhir_id"`
	TotalGapCount   int       `json:"total_gap_count"`
	OpenGapCount    int       `json:"open_gap_count"`
	ClosedGapCount  int       `json:"closed_gap_count"`
	OverdueGapCount int       `json:"overdue_gap_count"`
	CriticalGaps    int       `json:"critical_gaps"`
	LastUpdated     time.Time `json:"last_updated"`
}

// PopulationCareGapMetrics contains population-level care gap metrics.
type PopulationCareGapMetrics struct {
	TotalPatients        int                `json:"total_patients"`
	PatientsWithGaps     int                `json:"patients_with_gaps"`
	TotalOpenGaps        int                `json:"total_open_gaps"`
	AverageGapsPerPatient float64           `json:"average_gaps_per_patient"`
	GapsByCategory       map[string]int     `json:"gaps_by_category"`
	GapsByPriority       map[string]int     `json:"gaps_by_priority"`
	TopGapTypes          []GapTypeSummary   `json:"top_gap_types"`
	CalculatedAt         time.Time          `json:"calculated_at"`
}

// GapTypeSummary summarizes a specific gap type.
type GapTypeSummary struct {
	GapType     string `json:"gap_type"`
	GapName     string `json:"gap_name"`
	TotalCount  int    `json:"total_count"`
	OpenCount   int    `json:"open_count"`
	ClosedCount int    `json:"closed_count"`
}

// CareGapTrend represents care gap trends over time.
type CareGapTrend struct {
	Period       string `json:"period"`
	OpenGaps     int    `json:"open_gaps"`
	ClosedGaps   int    `json:"closed_gaps"`
	NewGaps      int    `json:"new_gaps"`
	ClosureRate  float64 `json:"closure_rate"`
}

// ──────────────────────────────────────────────────────────────────────────────
// API Methods
// ──────────────────────────────────────────────────────────────────────────────

// GetPatientCareGapSummary retrieves care gap summary for a patient.
func (c *KB13Client) GetPatientCareGapSummary(ctx context.Context, patientFHIRID string) (*PatientCareGapSummary, error) {
	url := fmt.Sprintf("%s/v1/patients/%s/care-gaps/summary", c.baseURL, patientFHIRID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Source", "KB-11-Population-Health")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WithError(err).WithField("patient", patientFHIRID).Warn("Failed to fetch care gap summary")
		return nil, fmt.Errorf("failed to fetch care gap summary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return &PatientCareGapSummary{
			PatientFHIRID: patientFHIRID,
			TotalGapCount: 0,
			LastUpdated:   time.Now(),
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-13 returned status %d", resp.StatusCode)
	}

	var summary PatientCareGapSummary
	if err := json.NewDecoder(resp.Body).Decode(&summary); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &summary, nil
}

// GetBatchPatientCareGapSummaries retrieves care gap summaries for multiple patients.
func (c *KB13Client) GetBatchPatientCareGapSummaries(ctx context.Context, patientFHIRIDs []string) (map[string]*PatientCareGapSummary, error) {
	if len(patientFHIRIDs) == 0 {
		return map[string]*PatientCareGapSummary{}, nil
	}

	url := fmt.Sprintf("%s/v1/care-gaps/batch/summary", c.baseURL)

	requestBody := struct {
		PatientFHIRIDs []string `json:"patient_fhir_ids"`
	}{
		PatientFHIRIDs: patientFHIRIDs,
	}

	bodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Source", "KB-11-Population-Health")

	// Use a new request with body
	req, err = http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Source", "KB-11-Population-Health")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WithError(err).Warn("Failed to fetch batch care gap summaries")
		return nil, fmt.Errorf("failed to fetch batch summaries: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-13 returned status %d", resp.StatusCode)
	}

	var response struct {
		Summaries map[string]*PatientCareGapSummary `json:"summaries"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Summaries, nil
}

// GetPopulationCareGapMetrics retrieves population-level care gap metrics.
func (c *KB13Client) GetPopulationCareGapMetrics(ctx context.Context, filter *CareGapFilter) (*PopulationCareGapMetrics, error) {
	url := fmt.Sprintf("%s/v1/care-gaps/population/metrics", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Add query parameters for filtering
	if filter != nil {
		q := req.URL.Query()
		if filter.Practice != "" {
			q.Set("practice", filter.Practice)
		}
		if filter.PCP != "" {
			q.Set("pcp", filter.PCP)
		}
		if filter.Category != "" {
			q.Set("category", filter.Category)
		}
		if filter.Priority != "" {
			q.Set("priority", filter.Priority)
		}
		req.URL.RawQuery = q.Encode()
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Source", "KB-11-Population-Health")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WithError(err).Warn("Failed to fetch population care gap metrics")
		return nil, fmt.Errorf("failed to fetch metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-13 returned status %d", resp.StatusCode)
	}

	var metrics PopulationCareGapMetrics
	if err := json.NewDecoder(resp.Body).Decode(&metrics); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &metrics, nil
}

// GetCareGapTrends retrieves care gap trends over time.
func (c *KB13Client) GetCareGapTrends(ctx context.Context, periods int, granularity string) ([]CareGapTrend, error) {
	url := fmt.Sprintf("%s/v1/care-gaps/trends", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Set("periods", fmt.Sprintf("%d", periods))
	q.Set("granularity", granularity) // "daily", "weekly", "monthly"
	req.URL.RawQuery = q.Encode()

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Source", "KB-11-Population-Health")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WithError(err).Warn("Failed to fetch care gap trends")
		return nil, fmt.Errorf("failed to fetch trends: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-13 returned status %d", resp.StatusCode)
	}

	var response struct {
		Trends []CareGapTrend `json:"trends"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Trends, nil
}

// GetTopCareGaps retrieves the most common care gap types.
func (c *KB13Client) GetTopCareGaps(ctx context.Context, limit int) ([]GapTypeSummary, error) {
	url := fmt.Sprintf("%s/v1/care-gaps/top?limit=%d", c.baseURL, limit)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Source", "KB-11-Population-Health")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.WithError(err).Warn("Failed to fetch top care gaps")
		return nil, fmt.Errorf("failed to fetch top care gaps: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-13 returned status %d", resp.StatusCode)
	}

	var response struct {
		GapTypes []GapTypeSummary `json:"gap_types"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.GapTypes, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Health Check
// ──────────────────────────────────────────────────────────────────────────────

// Health checks if KB-13 is available.
func (c *KB13Client) Health(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("KB-13 health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-13 unhealthy: status %d", resp.StatusCode)
	}

	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Filter Types
// ──────────────────────────────────────────────────────────────────────────────

// CareGapFilter provides filtering options for care gap queries.
type CareGapFilter struct {
	Practice string
	PCP      string
	Category string
	Priority string
}
