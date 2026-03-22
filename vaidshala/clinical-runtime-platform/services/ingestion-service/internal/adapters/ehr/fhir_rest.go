package ehr

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cardiofit/ingestion-service/internal/canonical"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// FHIRBundle represents a FHIR R4 Bundle resource.
type FHIRBundle struct {
	ResourceType string        `json:"resourceType"`
	Type         string        `json:"type"`
	Entry        []BundleEntry `json:"entry"`
}

// BundleEntry is a single entry within a FHIR Bundle.
type BundleEntry struct {
	FullURL  string          `json:"fullUrl,omitempty"`
	Resource json.RawMessage `json:"resource"`
	Request  *BundleRequest  `json:"request,omitempty"`
}

// BundleRequest carries the HTTP-verb metadata for transaction/batch entries.
type BundleRequest struct {
	Method string `json:"method"`
	URL    string `json:"url"`
}

// ResourceHeader is used to peek at the resourceType field before full
// deserialization of a FHIR resource.
type ResourceHeader struct {
	ResourceType string `json:"resourceType"`
}

// FHIRObservation is a minimal representation of a FHIR R4 Observation
// resource, capturing only the fields needed for canonical conversion.
type FHIRObservation struct {
	ResourceType    string `json:"resourceType"`
	ID              string `json:"id"`
	Status          string `json:"status"`
	Code            *struct {
		Coding []struct {
			System  string `json:"system"`
			Code    string `json:"code"`
			Display string `json:"display"`
		} `json:"coding"`
	} `json:"code"`
	Subject *struct {
		Reference string `json:"reference"`
	} `json:"subject"`
	EffectiveDateTime string `json:"effectiveDateTime,omitempty"`
	ValueQuantity     *struct {
		Value  float64 `json:"value"`
		Unit   string  `json:"unit"`
		System string  `json:"system"`
		Code   string  `json:"code"`
	} `json:"valueQuantity,omitempty"`
}

// FHIRRestAdapter validates and converts FHIR R4 Bundles into canonical
// observations for the ingestion pipeline.
type FHIRRestAdapter struct {
	logger *zap.Logger
}

// NewFHIRRestAdapter creates a new FHIRRestAdapter.
func NewFHIRRestAdapter(logger *zap.Logger) *FHIRRestAdapter {
	return &FHIRRestAdapter{logger: logger}
}

// vitalsLOINC maps LOINC codes that are classified as vitals observations.
var vitalsLOINC = map[string]bool{
	"8480-6":  true, // Systolic blood pressure
	"8462-4":  true, // Diastolic blood pressure
	"8867-4":  true, // Heart rate
	"8310-5":  true, // Body temperature
	"9279-1":  true, // Respiratory rate
	"2708-6":  true, // SpO2
	"29463-7": true, // Body weight
	"8302-2":  true, // Body height
}

// classifyByLOINC returns ObsVitals for known vital-sign LOINC codes and
// ObsLabs for everything else.
func classifyByLOINC(loincCode string) canonical.ObservationType {
	if vitalsLOINC[loincCode] {
		return canonical.ObsVitals
	}
	return canonical.ObsLabs
}

// ParseBundle validates a raw JSON payload as a FHIR R4 Bundle and converts
// any contained Observation resources into canonical observations.
func (a *FHIRRestAdapter) ParseBundle(ctx context.Context, raw []byte) (*FHIRBundle, []canonical.CanonicalObservation, error) {
	var bundle FHIRBundle
	if err := json.Unmarshal(raw, &bundle); err != nil {
		return nil, nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Validate resourceType.
	if bundle.ResourceType != "Bundle" {
		return nil, nil, fmt.Errorf("expected resourceType 'Bundle', got '%s'", bundle.ResourceType)
	}

	// Validate bundle type.
	switch bundle.Type {
	case "transaction", "batch", "collection":
		// OK
	default:
		return nil, nil, fmt.Errorf("unsupported bundle type '%s': must be transaction, batch, or collection", bundle.Type)
	}

	// Validate non-empty entries.
	if len(bundle.Entry) == 0 {
		return nil, nil, fmt.Errorf("bundle contains no entries")
	}

	var observations []canonical.CanonicalObservation

	for i, entry := range bundle.Entry {
		// Peek at resourceType.
		var header ResourceHeader
		if err := json.Unmarshal(entry.Resource, &header); err != nil {
			a.logger.Warn("skipping entry: cannot read resourceType",
				zap.Int("index", i),
				zap.Error(err),
			)
			continue
		}

		if header.ResourceType != "Observation" {
			continue
		}

		var obs FHIRObservation
		if err := json.Unmarshal(entry.Resource, &obs); err != nil {
			a.logger.Warn("skipping malformed Observation",
				zap.Int("index", i),
				zap.Error(err),
			)
			continue
		}

		canonical, err := a.toCanonical(obs, entry.Resource)
		if err != nil {
			a.logger.Warn("skipping unconvertible Observation",
				zap.Int("index", i),
				zap.Error(err),
			)
			continue
		}

		observations = append(observations, canonical)
	}

	a.logger.Info("FHIR bundle parsed",
		zap.String("bundle_type", bundle.Type),
		zap.Int("total_entries", len(bundle.Entry)),
		zap.Int("observations_extracted", len(observations)),
	)

	return &bundle, observations, nil
}

// toCanonical converts a single FHIRObservation to a CanonicalObservation.
func (a *FHIRRestAdapter) toCanonical(obs FHIRObservation, raw json.RawMessage) (canonical.CanonicalObservation, error) {
	// Extract LOINC code.
	var loincCode, display string
	if obs.Code != nil {
		for _, coding := range obs.Code.Coding {
			if coding.System == "http://loinc.org" || coding.Code != "" {
				loincCode = coding.Code
				display = coding.Display
				break
			}
		}
	}

	// Extract value.
	var value float64
	var unit string
	if obs.ValueQuantity != nil {
		value = obs.ValueQuantity.Value
		unit = obs.ValueQuantity.Unit
	}

	// Parse effective date.
	var ts time.Time
	if obs.EffectiveDateTime != "" {
		parsed, err := time.Parse(time.RFC3339, obs.EffectiveDateTime)
		if err != nil {
			// Try date-only format.
			parsed, err = time.Parse("2006-01-02", obs.EffectiveDateTime)
			if err != nil {
				return canonical.CanonicalObservation{}, fmt.Errorf("invalid effectiveDateTime '%s': %w", obs.EffectiveDateTime, err)
			}
		}
		ts = parsed
	} else {
		ts = time.Now().UTC()
	}

	_ = display // used for logging only

	return canonical.CanonicalObservation{
		ID:              uuid.New(),
		SourceType:      canonical.SourceEHR,
		SourceID:        "fhir_rest",
		ObservationType: classifyByLOINC(loincCode),
		LOINCCode:       loincCode,
		Value:           value,
		Unit:            unit,
		Timestamp:       ts,
		QualityScore:    0.90,
		RawPayload:      raw,
	}, nil
}
