package patient_reported

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/canonical"
	"github.com/cardiofit/ingestion-service/internal/coding"
)

// WhatsAppNLUPayload represents the parsed output from the Tier-1 NLU service.
// The NLU service extracts intent and entities from Hindi/regional language
// free text and sends structured JSON to the ingestion service.
type WhatsAppNLUPayload struct {
	PatientID  uuid.UUID        `json:"patient_id"`
	TenantID   uuid.UUID        `json:"tenant_id"`
	MessageID  string           `json:"message_id"`
	Timestamp  time.Time        `json:"timestamp"`
	Intent     string           `json:"intent"`     // e.g., "report_glucose", "report_bp", "report_symptom"
	Entities   []WhatsAppEntity `json:"entities"`
	Confidence float64          `json:"confidence"` // NLU confidence 0.0-1.0
	RawText    string           `json:"raw_text"`   // Original message text
}

// WhatsAppEntity is an extracted entity from NLU parsing.
type WhatsAppEntity struct {
	Type  string  `json:"type"`  // e.g., "glucose_value", "systolic_bp", "medication_name"
	Value float64 `json:"value,omitempty"`
	Text  string  `json:"text,omitempty"`
	Unit  string  `json:"unit,omitempty"`
}

// intentToAnalyte maps WhatsApp NLU intents to analyte names.
var intentToAnalyte = map[string]string{
	"report_glucose":    "glucose",
	"report_fasting":    "fasting_glucose",
	"report_bp":         "systolic_bp",
	"report_weight":     "weight",
	"report_symptom":    "",
	"report_hba1c":      "hba1c",
	"report_heart_rate": "heart_rate",
}

// WhatsAppAdapter converts NLU-parsed WhatsApp messages into CanonicalObservations.
type WhatsAppAdapter struct {
	logger *zap.Logger
}

// NewWhatsAppAdapter creates a new WhatsAppAdapter.
func NewWhatsAppAdapter(logger *zap.Logger) *WhatsAppAdapter {
	return &WhatsAppAdapter{logger: logger}
}

// Parse converts a WhatsAppNLUPayload into CanonicalObservations.
func (a *WhatsAppAdapter) Parse(payload WhatsAppNLUPayload) ([]canonical.CanonicalObservation, error) {
	if payload.PatientID == uuid.Nil {
		return nil, fmt.Errorf("whatsapp message missing patient_id")
	}
	if len(payload.Entities) == 0 {
		return nil, fmt.Errorf("whatsapp NLU extracted no entities")
	}

	timestamp := payload.Timestamp
	if timestamp.IsZero() {
		timestamp = time.Now().UTC()
	}

	observations := make([]canonical.CanonicalObservation, 0, len(payload.Entities))

	for _, entity := range payload.Entities {
		obs := canonical.CanonicalObservation{
			ID:              uuid.New(),
			PatientID:       payload.PatientID,
			TenantID:        payload.TenantID,
			SourceType:      canonical.SourcePatientReported,
			SourceID:        "whatsapp",
			ObservationType: canonical.ObsPatientReported,
			Value:           entity.Value,
			Unit:            entity.Unit,
			ValueString:     entity.Type,
			Timestamp:       timestamp,
			Flags:           []canonical.Flag{canonical.FlagManualEntry},
			RawPayload:      []byte(payload.RawText),
		}

		// Low NLU confidence adds LOW_QUALITY flag
		if payload.Confidence < 0.70 {
			obs.Flags = append(obs.Flags, canonical.FlagLowQuality)
		}

		// Categorize vitals
		if vitalsAnalytes[entity.Type] {
			obs.ObservationType = canonical.ObsVitals
		}

		// Resolve LOINC code
		analyte := entity.Type
		if mapped, ok := intentToAnalyte[payload.Intent]; ok && mapped != "" && analyte == "" {
			analyte = mapped
		}
		if analyte != "" {
			if loincCode, ok := coding.LookupLOINCByAnalyte(analyte); ok {
				obs.LOINCCode = loincCode
			}
		}

		observations = append(observations, obs)
	}

	a.logger.Info("parsed whatsapp message",
		zap.String("patient_id", payload.PatientID.String()),
		zap.String("intent", payload.Intent),
		zap.Float64("confidence", payload.Confidence),
		zap.Int("entity_count", len(observations)),
	)

	return observations, nil
}
