package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/config"
	"kb-22-hpi-engine/internal/models"
)

// MedicationSafetyProvider queries KB-5 for medication safety context when a
// safety flag fires (N-02). This enriches safety flags with contraindication
// and dose adjustment information from the patient's active medication list.
//
// All KB-5 calls are non-blocking enrichments: if KB-5 is unavailable or slow,
// the safety flag is still raised without medication context. The 30ms timeout
// ensures this never blocks the answer processing path.
type MedicationSafetyProvider struct {
	config *config.Config
	log    *zap.Logger
	client *http.Client
}

// MedicationSafetyResponse holds the KB-5 medication safety check result.
type MedicationSafetyResponse struct {
	Contraindicated []ContraindicationEntry `json:"contraindicated"`
	DoseAdjustments []DoseAdjustmentEntry   `json:"dose_adjustments"`
}

// ContraindicationEntry represents a single contraindication from KB-5.
type ContraindicationEntry struct {
	DrugClass string `json:"drug_class"`
	Reason    string `json:"reason"`
	Severity  string `json:"severity"`
}

// DoseAdjustmentEntry represents a single dose adjustment recommendation from KB-5.
type DoseAdjustmentEntry struct {
	DrugClass string `json:"drug_class"`
	Reason    string `json:"reason"`
	Action    string `json:"action"`
}

// NewMedicationSafetyProvider creates a new MedicationSafetyProvider.
func NewMedicationSafetyProvider(cfg *config.Config, log *zap.Logger) *MedicationSafetyProvider {
	return &MedicationSafetyProvider{
		config: cfg,
		log:    log,
		client: &http.Client{
			Timeout: cfg.KB5Timeout(),
		},
	}
}

// CheckContraindications queries KB-5 for medication safety information
// relevant to the given safety flag and the patient's active medications.
//
// Endpoint: GET KB-5 /api/v1/patients/{patient_id}/medication-safety?flag_id={flag_id}&medications={csv}
//
// Returns nil (not an error) on any failure. This is intentional: KB-5
// medication safety enrichment is a non-blocking enhancement that must never
// prevent a safety flag from being raised or delay the answer processing path.
//
// Timeout: KB5TimeoutMS (default 30ms).
func (p *MedicationSafetyProvider) CheckContraindications(
	ctx context.Context,
	patientID uuid.UUID,
	flagID string,
	activeMedications []string,
) (*MedicationSafetyResponse, error) {
	if p.config.KB5URL == "" {
		p.log.Debug("KB-5 URL not configured, skipping medication safety check")
		return nil, nil
	}

	if len(activeMedications) == 0 {
		p.log.Debug("no active medications, skipping medication safety check",
			zap.String("patient_id", patientID.String()),
			zap.String("flag_id", flagID),
		)
		return nil, nil
	}

	medicationsCSV := strings.Join(activeMedications, ",")
	url := fmt.Sprintf("%s/api/v1/patients/%s/medication-safety?flag_id=%s&medications=%s",
		p.config.KB5URL, patientID.String(), flagID, medicationsCSV)

	reqCtx, cancel := context.WithTimeout(ctx, p.config.KB5Timeout())
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		p.log.Warn("failed to create KB-5 request, skipping medication safety",
			zap.String("patient_id", patientID.String()),
			zap.String("flag_id", flagID),
			zap.Error(err),
		)
		return nil, nil
	}
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		p.log.Warn("KB-5 request failed, skipping medication safety enrichment",
			zap.String("patient_id", patientID.String()),
			zap.String("flag_id", flagID),
			zap.Error(err),
		)
		return nil, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		p.log.Warn("KB-5 returned non-OK status, skipping medication safety",
			zap.String("patient_id", patientID.String()),
			zap.String("flag_id", flagID),
			zap.Int("status", resp.StatusCode),
			zap.String("body", truncateBody(string(body), 200)),
		)
		return nil, nil
	}

	var result MedicationSafetyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		p.log.Warn("failed to decode KB-5 response, skipping medication safety",
			zap.String("patient_id", patientID.String()),
			zap.String("flag_id", flagID),
			zap.Error(err),
		)
		return nil, nil
	}

	p.log.Debug("KB-5 medication safety check complete",
		zap.String("patient_id", patientID.String()),
		zap.String("flag_id", flagID),
		zap.Int("contraindications", len(result.Contraindicated)),
		zap.Int("dose_adjustments", len(result.DoseAdjustments)),
	)

	return &result, nil
}

// truncateBody limits a response body string to maxLen characters for logging.
func truncateBody(body string, maxLen int) string {
	if len(body) <= maxLen {
		return body
	}
	return body[:maxLen] + "..."
}

// EnrichSafetyFlag attaches KB-5 medication safety context to a safety flag's
// MedicationSafetyContext field. This is a convenience method that combines
// CheckContraindications with JSON serialisation.
//
// Returns the enriched flag. If KB-5 is unavailable, the flag is returned
// unchanged with nil MedicationSafetyContext.
func (p *MedicationSafetyProvider) EnrichSafetyFlag(
	ctx context.Context,
	flag *models.SafetyFlag,
	patientID uuid.UUID,
	activeMedications []string,
) {
	if !flag.IsUrgentOrImmediate() {
		return
	}

	medSafety, _ := p.CheckContraindications(ctx, patientID, flag.FlagID, activeMedications)
	if medSafety == nil {
		return
	}

	data, err := json.Marshal(medSafety)
	if err != nil {
		p.log.Warn("failed to marshal medication safety context",
			zap.String("flag_id", flag.FlagID),
			zap.Error(err),
		)
		return
	}

	flag.MedicationSafetyContext = data
	p.log.Debug("enriched safety flag with KB-5 medication context",
		zap.String("flag_id", flag.FlagID),
		zap.Int("contraindications", len(medSafety.Contraindicated)),
		zap.Int("dose_adjustments", len(medSafety.DoseAdjustments)),
	)
}
