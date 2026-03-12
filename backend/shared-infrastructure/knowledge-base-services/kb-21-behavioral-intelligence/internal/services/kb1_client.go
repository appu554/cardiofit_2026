package services

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// =============================================================================
// KB-1 CLIENT - FDC Component Lookup
// =============================================================================
// Provides FDC decomposition for adherence tracking. When a patient takes
// a fixed-dose combination (e.g., telmisartan 40mg + amlodipine 5mg),
// that single pill counts as adherence to TWO drug classes (ARB + CCB).

// KB1FDCComponent represents one component of a fixed-dose combination.
// Mirrors kb-1-drug-rules/internal/models.FDCComponent.
type KB1FDCComponent struct {
	DrugName  string  `json:"drug_name"`
	DrugClass string  `json:"drug_class"`
	DoseMg    float64 `json:"dose_mg"`
}

// KB1FDCMapping maps an FDC product to its constituent components.
// Mirrors kb-1-drug-rules/internal/models.FDCMapping.
type KB1FDCMapping struct {
	FDCName    string            `json:"fdc_name"`
	Components []KB1FDCComponent `json:"components"`
	IsHTN      bool              `json:"is_htn"`
}

// KB1Client defines the interface for FDC lookup from KB-1 Drug Rules Service.
// Uses interface-based dependency injection so the adherence service can be
// tested without a running KB-1 instance.
type KB1Client interface {
	// GetFDCComponents returns the constituent drug classes for a fixed-dose combination.
	// Returns nil, nil if the drug is not an FDC (404 from KB-1).
	// Returns nil, error on communication failure.
	GetFDCComponents(drugName string) (*KB1FDCMapping, error)
}

// ExpandedDrugClass represents a single drug class extracted from a medication,
// with metadata about whether it came from an FDC decomposition.
type ExpandedDrugClass struct {
	DrugClass    string `json:"drug_class"`
	DrugName     string `json:"drug_name"`
	IsFDC        bool   `json:"is_fdc"`
	FDCParent    string `json:"fdc_parent,omitempty"` // original FDC name if decomposed
	MedicationID string `json:"medication_id,omitempty"`
}

// HTTPKB1Client implements KB1Client using HTTP calls to KB-1 Drug Rules Service.
type HTTPKB1Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewKB1Client creates a new HTTP-based KB-1 client.
// baseURL should be the root URL of KB-1, e.g. "http://localhost:8085".
func NewKB1Client(baseURL string, logger *zap.Logger) *HTTPKB1Client {
	return &HTTPKB1Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
		},
		logger: logger,
	}
}

// GetFDCComponents calls KB-1 to decompose an FDC into its constituent drug classes.
func (c *HTTPKB1Client) GetFDCComponents(drugName string) (*KB1FDCMapping, error) {
	url := fmt.Sprintf("%s/v1/fdc/%s/components", c.baseURL, drugName)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		c.logger.Warn("KB-1 FDC lookup failed",
			zap.String("drug_name", drugName),
			zap.Error(err),
		)
		return nil, fmt.Errorf("KB-1 FDC lookup failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// Not an FDC — this is normal, not an error
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-1 FDC lookup returned status %d", resp.StatusCode)
	}

	var mapping KB1FDCMapping
	if err := json.NewDecoder(resp.Body).Decode(&mapping); err != nil {
		return nil, fmt.Errorf("failed to decode KB-1 FDC response: %w", err)
	}

	return &mapping, nil
}

// NoOpKB1Client is a fallback client that always returns nil (no FDC decomposition).
// Used when KB-1 integration is disabled or unavailable.
type NoOpKB1Client struct{}

func (c *NoOpKB1Client) GetFDCComponents(drugName string) (*KB1FDCMapping, error) {
	return nil, nil
}
