package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"kb-patient-profile/internal/models"
)

// LOINCMapping holds a single lab-type-to-LOINC mapping, validated against KB-7.
type LOINCMapping struct {
	LabType   string `json:"lab_type"`   // KB-20 lab type constant (e.g., "CREATININE")
	LOINCCode string `json:"loinc_code"` // Canonical LOINC code (e.g., "2160-0")
	Display   string `json:"display"`    // KB-7-resolved display name
	Verified  bool   `json:"verified"`   // true if KB-7 confirmed the code exists
}

// KB7ConceptLookup is the interface the LOINC registry needs from KB-7.
// Satisfied by fhir.KB7Client — avoids import cycle between services ↔ fhir.
type KB7ConceptLookup interface {
	LookupConcept(loincCode string) (concept *KB7ConceptResult, err error)
}

// KB7ConceptResult is a minimal struct for concept lookup results.
// Decouples the services package from fhir package types.
type KB7ConceptResult struct {
	Code    string
	Display string
}

// LOINCRegistry maps KB-20 lab types to their canonical LOINC codes.
// All codes are resolved and validated via KB-7 Terminology Service at startup.
// Thread-safe for concurrent reads.
type LOINCRegistry struct {
	kb7    KB7ConceptLookup
	logger *zap.Logger

	mu       sync.RWMutex
	mappings map[string]*LOINCMapping // keyed by KB-20 lab type
	ready    bool
}

// canonicalLOINCCodes defines the 13 LOINC codes required by FactStore projections.
// These are the FHIR standard codes for the lab types KB-20 tracks.
var canonicalLOINCCodes = map[string]string{
	models.LabTypeCreatinine: "2160-0",  // Creatinine [Mass/volume] in Serum or Plasma
	models.LabTypeEGFR:      "33914-3", // eGFR (CKD-EPI)
	models.LabTypeFBG:       "1558-6",  // Fasting glucose [Mass/volume] in Serum or Plasma
	models.LabTypeHbA1c:     "4548-4",  // Hemoglobin A1c/Hemoglobin.total in Blood
	models.LabTypeSBP:       "8480-6",  // Systolic blood pressure
	models.LabTypeDBP:       "8462-4",  // Diastolic blood pressure
	models.LabTypePotassium: "6298-4",  // Potassium [Moles/volume] in Serum or Plasma
	models.LabTypeSodium:    "2951-2",  // Sodium [Moles/volume] in Serum or Plasma
	"HEART_RATE":            "8867-4",  // Heart rate
	"WEIGHT":                "29463-7", // Body weight
	models.LabTypeACR:       "9318-7",  // Albumin/Creatinine [Mass Ratio] in Urine
	models.LabTypeTotalCholesterol: "2093-3",  // Cholesterol [Mass/volume] in Serum or Plasma
	models.LabTypeHDL:              "2085-9",  // Cholesterol in HDL [Mass/volume] in Serum or Plasma
}

// NewLOINCRegistry creates a registry with KB-7 concept lookup for validation.
func NewLOINCRegistry(kb7 KB7ConceptLookup, logger *zap.Logger) *LOINCRegistry {
	return &LOINCRegistry{
		kb7:      kb7,
		logger:   logger,
		mappings: make(map[string]*LOINCMapping),
	}
}

// Initialize validates all canonical LOINC codes against KB-7 and populates the registry.
// Should be called at service startup. Non-blocking — logs warnings for unverified codes
// but does not fail the service (graceful degradation).
func (r *LOINCRegistry) Initialize(ctx context.Context) {
	r.mu.Lock()
	defer r.mu.Unlock()

	verified := 0
	total := len(canonicalLOINCCodes)

	for labType, loincCode := range canonicalLOINCCodes {
		mapping := &LOINCMapping{
			LabType:   labType,
			LOINCCode: loincCode,
		}

		// Validate against KB-7 with a per-code timeout
		_, cancel := context.WithTimeout(ctx, 5*time.Second)
		concept, err := r.kb7.LookupConcept(loincCode)
		cancel()

		if err != nil {
			r.logger.Warn("KB-7 LOINC verification failed — using unverified code",
				zap.String("lab_type", labType),
				zap.String("loinc_code", loincCode),
				zap.Error(err))
			mapping.Verified = false
		} else if concept != nil {
			mapping.Display = concept.Display
			mapping.Verified = true
			verified++
		}

		r.mappings[labType] = mapping
	}

	r.ready = true
	r.logger.Info("LOINC registry initialized",
		zap.Int("verified", verified),
		zap.Int("total", total))
}

// LOINCForLabType returns the canonical LOINC code for a KB-20 lab type.
// Returns empty string if the lab type is not in the registry.
func (r *LOINCRegistry) LOINCForLabType(labType string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if m, ok := r.mappings[labType]; ok {
		return m.LOINCCode
	}
	return ""
}

// LabTypeForLOINC returns the KB-20 lab type for a LOINC code.
// Returns empty string if the code is not in the registry.
func (r *LOINCRegistry) LabTypeForLOINC(loincCode string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, m := range r.mappings {
		if m.LOINCCode == loincCode {
			return m.LabType
		}
	}
	return ""
}

// GetMapping returns the full mapping for a lab type, or nil if not found.
func (r *LOINCRegistry) GetMapping(labType string) *LOINCMapping {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.mappings[labType]
}

// AllMappings returns all registered LOINC mappings.
func (r *LOINCRegistry) AllMappings() []*LOINCMapping {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*LOINCMapping, 0, len(r.mappings))
	for _, m := range r.mappings {
		result = append(result, m)
	}
	return result
}

// IsReady returns true if the registry has been initialized.
func (r *LOINCRegistry) IsReady() bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.ready
}

// VerificationSummary returns a summary string for health check / startup logging.
func (r *LOINCRegistry) VerificationSummary() string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	verified, total := 0, len(r.mappings)
	for _, m := range r.mappings {
		if m.Verified {
			verified++
		}
	}
	return fmt.Sprintf("%d/%d LOINC codes verified via KB-7", verified, total)
}
