// Package clients provides HTTP clients for external service integration.
//
// CRITICAL: KB-11 is READ-ONLY for patient data.
// All clients in this package MUST only perform read operations.
// Patient data is NOT owned by KB-11 - we only CONSUME from upstream.
package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/cardiofit/kb-11-population-health/internal/models"
)

// FHIRClient provides read-only access to FHIR Store.
// IMPORTANT: This client is intentionally READ-ONLY.
// KB-11 does NOT write to FHIR Store - data flows only inward.
type FHIRClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *logrus.Entry
}

// FHIRPatient represents a FHIR Patient resource (R4).
type FHIRPatient struct {
	ResourceType string           `json:"resourceType"`
	ID           string           `json:"id"`
	Identifier   []FHIRIdentifier `json:"identifier,omitempty"`
	Name         []FHIRHumanName  `json:"name,omitempty"`
	Gender       string           `json:"gender,omitempty"`
	BirthDate    string           `json:"birthDate,omitempty"`
}

// FHIRIdentifier represents a FHIR Identifier.
type FHIRIdentifier struct {
	System string `json:"system,omitempty"`
	Value  string `json:"value,omitempty"`
}

// FHIRHumanName represents a FHIR HumanName.
type FHIRHumanName struct {
	Use    string   `json:"use,omitempty"`
	Family string   `json:"family,omitempty"`
	Given  []string `json:"given,omitempty"`
}

// FHIRBundle represents a FHIR Bundle resource.
type FHIRBundle struct {
	ResourceType string            `json:"resourceType"`
	Type         string            `json:"type"`
	Total        int               `json:"total,omitempty"`
	Link         []FHIRBundleLink  `json:"link,omitempty"`
	Entry        []FHIRBundleEntry `json:"entry,omitempty"`
}

// FHIRBundleLink represents pagination links in a FHIR Bundle.
type FHIRBundleLink struct {
	Relation string `json:"relation"`
	URL      string `json:"url"`
}

// FHIRBundleEntry represents an entry in a FHIR Bundle.
type FHIRBundleEntry struct {
	FullURL  string      `json:"fullUrl,omitempty"`
	Resource interface{} `json:"resource,omitempty"`
}

// NewFHIRClient creates a new FHIR client.
func NewFHIRClient(baseURL string, logger *logrus.Entry) *FHIRClient {
	return &FHIRClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger.WithField("client", "fhir"),
	}
}

// GetPatient retrieves a single patient by FHIR ID (READ-ONLY).
func (c *FHIRClient) GetPatient(ctx context.Context, fhirID string) (*FHIRPatient, error) {
	url := fmt.Sprintf("%s/Patient/%s", c.baseURL, fhirID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/fhir+json")

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
		return nil, fmt.Errorf("FHIR API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var patient FHIRPatient
	if err := json.NewDecoder(resp.Body).Decode(&patient); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &patient, nil
}

// SearchPatients retrieves patients with pagination (READ-ONLY).
func (c *FHIRClient) SearchPatients(ctx context.Context, count int, pageToken string) (*FHIRBundle, error) {
	url := fmt.Sprintf("%s/Patient?_count=%d", c.baseURL, count)
	if pageToken != "" {
		url = fmt.Sprintf("%s&_page_token=%s", url, pageToken)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/fhir+json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("FHIR API error: status=%d, body=%s", resp.StatusCode, string(body))
	}

	var bundle FHIRBundle
	if err := json.NewDecoder(resp.Body).Decode(&bundle); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &bundle, nil
}

// ToPatientProjection converts a FHIR Patient to a PatientProjection.
func (p *FHIRPatient) ToPatientProjection() *models.PatientProjection {
	proj := models.NewPatientProjection(p.ID, models.SyncSourceFHIR)

	// Extract MRN from identifiers
	for _, id := range p.Identifier {
		if id.System == "http://hospital.example/mrn" || id.System == "urn:oid:2.16.840.1.113883.4.6" {
			proj.MRN = &id.Value
			break
		}
	}

	// Extract name
	for _, name := range p.Name {
		if name.Use == "official" || name.Use == "" {
			proj.LastName = &name.Family
			if len(name.Given) > 0 {
				proj.FirstName = &name.Given[0]
			}
			break
		}
	}

	// Gender
	if p.Gender != "" {
		g := models.Gender(p.Gender)
		proj.Gender = &g
	}

	// Birth date
	if p.BirthDate != "" {
		if t, err := time.Parse("2006-01-02", p.BirthDate); err == nil {
			proj.DateOfBirth = &t
		}
	}

	return proj
}

// GetNextPageToken extracts the next page token from a bundle.
func (b *FHIRBundle) GetNextPageToken() string {
	for _, link := range b.Link {
		if link.Relation == "next" {
			return link.URL
		}
	}
	return ""
}

// Health checks if the FHIR Store is accessible.
func (c *FHIRClient) Health(ctx context.Context) error {
	url := fmt.Sprintf("%s/metadata", c.baseURL)

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
		return fmt.Errorf("FHIR Store unhealthy: status=%d", resp.StatusCode)
	}

	return nil
}
