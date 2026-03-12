package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"kb-21-behavioral-intelligence/internal/models"

	"go.uber.org/zap"
)

// SafetyClient provides direct fast-path communication with KB-23 for safety alerts.
//
// G-01: When OutcomeCorrelation classifies BEHAVIORAL_GAP or DISCORDANT, KB-21 calls
// KB-23 directly to generate a DecisionCard with the appropriate gate:
//   - BEHAVIORAL_GAP  → MODIFY gate, dose_adjustment_notes = "BEHAVIORAL_GAP"
//   - DISCORDANT      → SAFE gate, recommendation = MEDICATION_REVIEW
//
// G-03: When HYPO_RISK_ELEVATED fires, KB-21 calls KB-23 directly instead of routing
// through KB-19. KB-23 produces PAUSE (not HALT) for behavioral sources because
// behavioral risk is probabilistic, not confirmed hypoglycaemia.
// KB-19 is notified as a SECONDARY event after KB-23 generates the card.
//
// This is a two-hop path (KB-21 → KB-23) vs the previous four-hop path (KB-21 → KB-19 → KB-4).
// It also ensures a SafetyTrace record exists for regulatory audit.
type SafetyClient struct {
	kb23URL    string
	httpClient *http.Client
	logger     *zap.Logger
}

func NewSafetyClient(kb23URL string, logger *zap.Logger) *SafetyClient {
	return &SafetyClient{
		kb23URL: kb23URL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		logger: logger,
	}
}

// AlertBehavioralGap sends a BEHAVIORAL_GAP or DISCORDANT alert to KB-23 (G-01).
// Called by CorrelationService when treatment_response_class changes.
//
// BEHAVIORAL_GAP: adherence improving but HbA1c not responding — do NOT escalate dose.
//   KB-23 generates MODIFY-gate DecisionCard with BEHAVIORAL_GAP dose_adjustment_notes token.
//
// DISCORDANT: high adherence but no clinical improvement — pharmacological failure.
//   KB-23 generates SAFE-gate DecisionCard with MEDICATION_REVIEW recommendation.
func (c *SafetyClient) AlertBehavioralGap(corr models.OutcomeCorrelation) error {
	if corr.TreatmentResponseClass != models.ResponseBehavioral &&
		corr.TreatmentResponseClass != models.ResponseDiscordant {
		return nil // Only alert on these two classes
	}

	gateType := "MODIFY"
	notes := "BEHAVIORAL_GAP: Do not intensify medication. Adherence is the primary problem."
	severity := "HIGH"

	if corr.TreatmentResponseClass == models.ResponseDiscordant {
		gateType = "SAFE"
		notes = "MEDICATION_REVIEW: High adherence with no clinical improvement. Consider medication class change."
		severity = "MODERATE"
	}

	req := models.SafetyAlertRequest{
		PatientID:              corr.PatientID,
		Source:                 "KB21_BEHAVIORAL",
		AlertType:              string(corr.TreatmentResponseClass),
		GateType:               gateType,
		Severity:               severity,
		Timestamp:              time.Now().UTC(),
		TreatmentResponseClass: corr.TreatmentResponseClass,
		MeanAdherenceScore:     corr.MeanAdherenceScore,
		HbA1cDelta:             corr.HbA1cDelta,
		DoseAdjustmentNotes:    notes,
	}

	return c.postSafetyAlert("/safety/behavioral-gap-alert", req)
}

// AlertHypoRisk sends a HYPO_RISK_ELEVATED alert to KB-23 fast-path (G-03).
// KB-23 produces PAUSE (not HALT) because behavioral risk is probabilistic.
// KB-19 is notified as a secondary event after KB-23 generates the DecisionCard.
func (c *SafetyClient) AlertHypoRisk(event models.HypoRiskEvent) error {
	req := models.SafetyAlertRequest{
		PatientID:           event.PatientID,
		Source:              "KB21_BEHAVIORAL",
		AlertType:           "HYPO_RISK_BEHAVIORAL",
		GateType:            "PAUSE", // PAUSE not HALT — behavioral risk is probabilistic
		Severity:            string(event.RiskLevel),
		Timestamp:           event.Timestamp,
		RiskFactors:         event.RiskFactors,
		RiskLevel:           event.RiskLevel,
		AffectedMedications: event.AffectedMedications,
	}

	return c.postSafetyAlert("/safety/hypoglycaemia-alert", req)
}

// postSafetyAlert sends a safety alert to KB-23.
func (c *SafetyClient) postSafetyAlert(path string, req models.SafetyAlertRequest) error {
	if c.kb23URL == "" {
		c.logger.Warn("KB-23 safety URL not configured — alert logged but not delivered",
			zap.String("alert_type", req.AlertType),
			zap.String("patient_id", req.PatientID),
		)
		return nil
	}

	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal safety alert: %w", err)
	}

	url := c.kb23URL + path
	resp, err := c.httpClient.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		c.logger.Error("KB-23 safety alert delivery failed — will retry via event bus",
			zap.String("url", url),
			zap.String("alert_type", req.AlertType),
			zap.Error(err),
		)
		return fmt.Errorf("KB-23 unreachable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		c.logger.Error("KB-23 rejected safety alert",
			zap.String("alert_type", req.AlertType),
			zap.Int("status_code", resp.StatusCode),
		)
		return fmt.Errorf("KB-23 returned status %d", resp.StatusCode)
	}

	c.logger.Info("Safety alert delivered to KB-23",
		zap.String("alert_type", req.AlertType),
		zap.String("gate_type", req.GateType),
		zap.String("patient_id", req.PatientID),
	)

	return nil
}
