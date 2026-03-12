// Package manifest provides HTTP handlers for authority capability exposure.
//
// This addresses Gap 3: OHDSI Optionality Guardrail
// Leadership and operators need visibility into what level of clinical safety is active.
//
// Endpoints:
//   GET /health/authorities     - Full capability status
//   GET /health/authorities/ddi - DDI-specific coverage
//   GET /health/authorities/validate - Trigger validation
package manifest

import (
	"encoding/json"
	"net/http"
	"time"
)

// Handler provides HTTP endpoints for authority capability exposure
type Handler struct {
	validator *Validator
}

// NewHandler creates a new HTTP handler for manifest operations
func NewHandler(validator *Validator) *Handler {
	return &Handler{validator: validator}
}

// RegisterRoutes registers the manifest HTTP routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health/authorities", h.handleCapabilities)
	mux.HandleFunc("/health/authorities/ddi", h.handleDDICoverage)
	mux.HandleFunc("/health/authorities/validate", h.handleValidate)
	mux.HandleFunc("/health/authorities/facts", h.handleFactTypeCoverage)
}

// =============================================================================
// CAPABILITY ENDPOINT (Gap 3 Fix)
// =============================================================================

// CapabilitiesResponse is the JSON response for the capabilities endpoint
type CapabilitiesResponse struct {
	Status          string                     `json:"status"`
	CoverageLevel   string                     `json:"coverage_level"`
	CoverageWarning string                     `json:"coverage_warning,omitempty"`
	Authorities     map[string]AuthorityStatus `json:"authorities"`
	FactTypes       map[string]bool            `json:"fact_types"`
	Timestamp       time.Time                  `json:"timestamp"`
}

func (h *Handler) handleCapabilities(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	caps := h.validator.GetCapabilities()

	response := CapabilitiesResponse{
		Status:          "ok",
		CoverageLevel:   caps.CoverageLevel,
		CoverageWarning: caps.CoverageWarning,
		Authorities:     caps.Authorities,
		FactTypes:       caps.FactTypeCoverage,
		Timestamp:       caps.LastUpdated,
	}

	w.Header().Set("Content-Type", "application/json")

	// Set status based on coverage level
	switch caps.CoverageLevel {
	case "FULL":
		w.WriteHeader(http.StatusOK)
	case "PARTIAL":
		w.WriteHeader(http.StatusOK) // Still OK, just with warnings
	case "MINIMAL":
		w.WriteHeader(http.StatusOK) // Degraded but functional
	default:
		w.WriteHeader(http.StatusServiceUnavailable) // No authorities = not safe
	}

	json.NewEncoder(w).Encode(response)
}

// =============================================================================
// DDI COVERAGE ENDPOINT (Specific Gap 3 Requirement)
// =============================================================================
// The review specifically called out: "ONC DDI covers ~1,200 pairs, OHDSI covers 200k+"
// Operators need to know what DDI coverage level is active.

// DDICoverageResponse shows DDI-specific coverage
type DDICoverageResponse struct {
	Status      string `json:"status"`
	Coverage    DDICoverage `json:"ddi_coverage"`
	Timestamp   time.Time `json:"timestamp"`
}

// DDICoverage details DDI source availability
type DDICoverage struct {
	ONCHighPriority     bool   `json:"onc_high_priority"`      // ~1,200 pairs
	OHDSIAthena         bool   `json:"ohdsi_athena"`           // ~200K pairs
	DrugBank            bool   `json:"drugbank"`               // PK + DDI
	CredibleMeds        bool   `json:"crediblemeds"`           // QT DDI
	CoverageLevel       string `json:"coverage_level"`         // "FULL", "PARTIAL", "BASIC"
	EstimatedPairCount  string `json:"estimated_pair_count"`
	Warning             string `json:"warning,omitempty"`
}

func (h *Handler) handleDDICoverage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	caps := h.validator.GetCapabilities()

	// Check DDI-related authorities
	drugbankAvail := caps.Authorities["drugbank"].Available
	crediblemedsAvail := caps.Authorities["crediblemeds"].Available

	// Determine DDI coverage level
	var coverageLevel string
	var estimatedPairs string
	var warning string

	// Note: ONC and OHDSI are tracked in Phase 1 MANIFEST, not Phase 3b
	// This endpoint shows Phase 3b DDI-related authority availability
	switch {
	case drugbankAvail && crediblemedsAvail:
		coverageLevel = "FULL"
		estimatedPairs = "DrugBank DDI + CredibleMeds QT"
	case drugbankAvail || crediblemedsAvail:
		coverageLevel = "PARTIAL"
		estimatedPairs = "Limited DDI coverage"
		warning = "Not all DDI authorities are available. Some interactions may not be detected."
	default:
		coverageLevel = "BASIC"
		estimatedPairs = "Phase 1 ONC only (check Phase 1 manifest)"
		warning = "Phase 3b DDI authorities unavailable. Relying on Phase 1 ONC data only."
	}

	coverage := DDICoverage{
		ONCHighPriority:    true,  // Phase 1 - assumed available
		OHDSIAthena:        false, // Phase 1 - check Phase 1 manifest
		DrugBank:           drugbankAvail,
		CredibleMeds:       crediblemedsAvail,
		CoverageLevel:      coverageLevel,
		EstimatedPairCount: estimatedPairs,
		Warning:            warning,
	}

	response := DDICoverageResponse{
		Status:    "ok",
		Coverage:  coverage,
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// =============================================================================
// VALIDATION ENDPOINT
// =============================================================================

func (h *Handler) handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	result := h.validator.Validate(r.Context())

	w.Header().Set("Content-Type", "application/json")

	if result.Valid {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusUnprocessableEntity)
	}

	json.NewEncoder(w).Encode(result)
}

// =============================================================================
// FACT TYPE COVERAGE ENDPOINT
// =============================================================================

// FactTypeCoverageResponse shows coverage by fact type
type FactTypeCoverageResponse struct {
	Status       string                     `json:"status"`
	FactTypes    map[string]FactTypeDetail  `json:"fact_types"`
	Timestamp    time.Time                  `json:"timestamp"`
}

// FactTypeDetail shows details for a single fact type
type FactTypeDetail struct {
	Available       bool     `json:"available"`
	ProvidedBy      []string `json:"provided_by"`
	LLMPolicy       string   `json:"llm_policy"`
	AuthorityLevel  string   `json:"authority_level"`
}

func (h *Handler) handleFactTypeCoverage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	caps := h.validator.GetCapabilities()

	// Build fact type details
	factTypes := make(map[string]FactTypeDetail)

	// Map fact types to their providing authorities
	factTypeProviders := map[string]struct {
		providers      []string
		llmPolicy      string
		authorityLevel string
	}{
		"LACTATION_SAFETY": {
			providers:      []string{"lactmed"},
			llmPolicy:      "NEVER",
			authorityLevel: "DEFINITIVE",
		},
		"PHARMACOGENOMICS": {
			providers:      []string{"cpic"},
			llmPolicy:      "GAP_FILL_ONLY",
			authorityLevel: "DEFINITIVE",
		},
		"QT_PROLONGATION": {
			providers:      []string{"crediblemeds"},
			llmPolicy:      "NEVER",
			authorityLevel: "DEFINITIVE",
		},
		"HEPATOTOXICITY": {
			providers:      []string{"livertox"},
			llmPolicy:      "NEVER",
			authorityLevel: "DEFINITIVE",
		},
		"GERIATRIC_PIM": {
			providers:      []string{"ohdsi_beers"},
			llmPolicy:      "NEVER",
			authorityLevel: "DEFINITIVE",
		},
		"PK_PARAMETERS": {
			providers:      []string{"drugbank"},
			llmPolicy:      "GAP_FILL_ONLY",
			authorityLevel: "PRIMARY",
		},
		"PROTEIN_BINDING": {
			providers:      []string{"drugbank"},
			llmPolicy:      "GAP_FILL_ONLY",
			authorityLevel: "PRIMARY",
		},
		"DRUG_INTERACTION": {
			providers:      []string{"drugbank"},
			llmPolicy:      "GAP_FILL_ONLY",
			authorityLevel: "PRIMARY",
		},
		"CYP_INTERACTION": {
			providers:      []string{"drugbank"},
			llmPolicy:      "GAP_FILL_ONLY",
			authorityLevel: "PRIMARY",
		},
		"TRANSPORTER_INTERACTION": {
			providers:      []string{"drugbank"},
			llmPolicy:      "GAP_FILL_ONLY",
			authorityLevel: "PRIMARY",
		},
	}

	for factType, info := range factTypeProviders {
		available := false
		activeProviders := []string{}

		for _, provider := range info.providers {
			if status, ok := caps.Authorities[provider]; ok && status.Available {
				available = true
				activeProviders = append(activeProviders, provider)
			}
		}

		factTypes[factType] = FactTypeDetail{
			Available:      available,
			ProvidedBy:     activeProviders,
			LLMPolicy:      info.llmPolicy,
			AuthorityLevel: info.authorityLevel,
		}
	}

	response := FactTypeCoverageResponse{
		Status:    "ok",
		FactTypes: factTypes,
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
