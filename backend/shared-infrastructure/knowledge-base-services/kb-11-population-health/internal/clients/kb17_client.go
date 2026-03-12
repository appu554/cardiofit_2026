// Package clients provides HTTP clients for external service integration.
package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/cardiofit/kb-11-population-health/internal/models"
)

// KB17Client provides read-only access to KB-17 Population Registry.
// IMPORTANT: This client is intentionally READ-ONLY.
// KB-11 does NOT write to KB-17 - data flows only inward.
type KB17Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *logrus.Entry
}

// KB17Patient represents a patient from KB-17 Registry.
type KB17Patient struct {
	ID            uuid.UUID  `json:"id"`
	FHIRID        string     `json:"fhir_id"`
	MRN           *string    `json:"mrn,omitempty"`
	FirstName     *string    `json:"first_name,omitempty"`
	LastName      *string    `json:"last_name,omitempty"`
	DateOfBirth   *time.Time `json:"date_of_birth,omitempty"`
	Gender        *string    `json:"gender,omitempty"`
	PrimaryCareProvider *string `json:"primary_care_provider,omitempty"`
	Practice      *string    `json:"practice,omitempty"`
	Status        string     `json:"status"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// KB17PatientList represents a paginated list of patients from KB-17.
type KB17PatientList struct {
	Patients   []KB17Patient `json:"patients"`
	Total      int           `json:"total"`
	Limit      int           `json:"limit"`
	Offset     int           `json:"offset"`
	HasMore    bool          `json:"has_more"`
}

// NewKB17Client creates a new KB-17 client.
func NewKB17Client(baseURL string, logger *logrus.Entry) *KB17Client {
	return &KB17Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger.WithField("client", "kb17"),
	}
}

// GetPatient retrieves a single patient by KB-17 ID (READ-ONLY).
func (c *KB17Client) GetPatient(ctx context.Context, patientID uuid.UUID) (*KB17Patient, error) {
	url := fmt.Sprintf("%s/v1/patients/%s", c.baseURL, patientID.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-17 API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var patient KB17Patient
	if err := json.NewDecoder(resp.Body).Decode(&patient); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &patient, nil
}

// GetPatientByFHIRID retrieves a patient by their FHIR ID (READ-ONLY).
func (c *KB17Client) GetPatientByFHIRID(ctx context.Context, fhirID string) (*KB17Patient, error) {
	url := fmt.Sprintf("%s/v1/patients?fhir_id=%s", c.baseURL, fhirID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-17 API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var list KB17PatientList
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(list.Patients) == 0 {
		return nil, nil
	}

	return &list.Patients[0], nil
}

// ListPatients retrieves patients with pagination (READ-ONLY).
func (c *KB17Client) ListPatients(ctx context.Context, limit, offset int) (*KB17PatientList, error) {
	url := fmt.Sprintf("%s/v1/patients?limit=%d&offset=%d", c.baseURL, limit, offset)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-17 API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var list KB17PatientList
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &list, nil
}

// ToPatientProjection converts a KB-17 Patient to a PatientProjection.
func (p *KB17Patient) ToPatientProjection() *models.PatientProjection {
	proj := models.NewPatientProjection(p.FHIRID, models.SyncSourceKB17)

	proj.KB17PatientID = &p.ID
	proj.MRN = p.MRN
	proj.FirstName = p.FirstName
	proj.LastName = p.LastName
	proj.DateOfBirth = p.DateOfBirth

	if p.Gender != nil {
		g := models.Gender(*p.Gender)
		proj.Gender = &g
	}

	// Attribution from KB-17
	proj.AttributedPCP = p.PrimaryCareProvider
	proj.AttributedPractice = p.Practice
	if p.PrimaryCareProvider != nil || p.Practice != nil {
		now := time.Now()
		proj.AttributionDate = &now
	}

	return proj
}

// Health checks if KB-17 is accessible.
func (c *KB17Client) Health(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-17 unhealthy: status=%d", resp.StatusCode)
	}

	return nil
}
