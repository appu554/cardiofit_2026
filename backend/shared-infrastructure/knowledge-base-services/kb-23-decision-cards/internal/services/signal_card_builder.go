package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-23-decision-cards/internal/models"
)

// SignalCardBuilder builds DecisionCards from KB-22 ClinicalSignalEvents.
type SignalCardBuilder struct {
	log *zap.Logger
}

// NewSignalCardBuilder creates a new SignalCardBuilder.
func NewSignalCardBuilder(log *zap.Logger) *SignalCardBuilder {
	return &SignalCardBuilder{log: log}
}

// Build creates a DecisionCard from a ClinicalSignalEvent.
// Returns nil if no card template matches (caller should return 204).
func (b *SignalCardBuilder) Build(ctx context.Context, event *models.ClinicalSignalEvent) (*models.DecisionCard, error) {
	templateID := b.resolveTemplate(event)
	if templateID == "" {
		b.log.Debug("no card template for signal event",
			zap.String("node_id", event.NodeID),
			zap.String("event_id", event.EventID))
		return nil, nil
	}

	patientID, err := uuid.Parse(event.PatientID)
	if err != nil {
		return nil, fmt.Errorf("invalid patient_id: %w", err)
	}

	confidenceTier := b.deriveConfidenceTier(event)
	gate := b.evaluateGate(event)
	safetyTier := b.deriveSafetyTier(event)

	card := &models.DecisionCard{
		CardID:                   uuid.New(),
		PatientID:                patientID,
		TemplateID:               templateID,
		NodeID:                   event.NodeID,
		CardSource:               models.SourceClinicalSignal,
		DiagnosticConfidenceTier: confidenceTier,
		MCUGate:                  gate,
		MCUGateRationale:         fmt.Sprintf("signal event %s from node %s", event.EventID, event.NodeID),
		SafetyTier:               safetyTier,
		Status:                   models.StatusActive,
		CreatedAt:                time.Now().UTC(),
		UpdatedAt:                time.Now().UTC(),
	}

	// Set HALT-specific fields
	if gate == models.GateHalt {
		card.PendingReaffirmation = true
	}

	b.log.Info("built decision card from signal",
		zap.String("card_id", card.CardID.String()),
		zap.String("template_id", templateID),
		zap.String("confidence", string(confidenceTier)),
		zap.String("gate", string(gate)))

	return card, nil
}

// resolveTemplate maps event -> card template ID.
// Returns "" if no template (normal/safe events don't generate cards).
func (b *SignalCardBuilder) resolveTemplate(event *models.ClinicalSignalEvent) string {
	if event.SignalType == "MONITORING_CLASSIFICATION" && event.Classification != nil {
		// For PM nodes, template from classification category.
		// Pattern: dc-pmNN-category-v1 (lowercased, hyphens)
		category := strings.ToLower(strings.ReplaceAll(event.Classification.Category, "_", "-"))
		if category == "" || category == "normal-dipper" || category == "at-target" ||
			category == "asymptomatic" || category == "stable" || category == "normal-excursion" ||
			category == "improving" || category == "good" || category == "adequate" {
			return "" // Normal/safe classifications don't need cards
		}
		nodeNum := strings.TrimPrefix(strings.ToLower(event.NodeID), "pm-")
		return fmt.Sprintf("dc-pm%s-%s-v1", nodeNum, category)
	}

	if event.SignalType == "DETERIORATION_SIGNAL" && event.DeteriorationSignal != nil {
		// For MD nodes, template from signal name.
		signal := strings.ToLower(strings.ReplaceAll(event.DeteriorationSignal.Signal, "_", "-"))
		if strings.HasSuffix(signal, "-stable") || strings.HasSuffix(signal, "-improving") ||
			signal == "vr-stable" || signal == "vr-improving" ||
			signal == "rr-stable" || signal == "autonomic-normal" ||
			signal == "glycemic-controlled" || signal == "cv-risk-low" {
			return "" // Stable/improving signals don't need cards
		}
		nodeNum := strings.TrimPrefix(strings.ToLower(event.NodeID), "md-")
		return fmt.Sprintf("dc-md%s-%s-v1", nodeNum, signal)
	}

	return ""
}

// deriveConfidenceTier maps data sufficiency + severity -> confidence tier.
func (b *SignalCardBuilder) deriveConfidenceTier(event *models.ClinicalSignalEvent) models.ConfidenceTier {
	sufficiency := ""
	severity := ""

	if event.Classification != nil {
		sufficiency = event.Classification.DataSufficiency
	}
	if event.DeteriorationSignal != nil {
		severity = event.DeteriorationSignal.Severity
	}

	// Extract severity from MCUGateSuggestion as a proxy when direct severity not available
	if severity == "" && event.MCUGateSuggestion != nil {
		switch *event.MCUGateSuggestion {
		case "HALT", "PAUSE":
			severity = "CRITICAL"
		case "MODIFY":
			severity = "MODERATE"
		default:
			severity = "NONE"
		}
	}

	// Confidence tiers: FIRM, PROBABLE, POSSIBLE
	if sufficiency == "SUFFICIENT" && (severity == "CRITICAL" || severity == "MODERATE") {
		return models.TierFirm
	}
	if sufficiency == "SUFFICIENT" {
		return models.TierProbable
	}
	return models.TierPossible
}

// evaluateGate extracts the MCU gate suggestion from the event.
func (b *SignalCardBuilder) evaluateGate(event *models.ClinicalSignalEvent) models.MCUGate {
	if event.MCUGateSuggestion != nil {
		return parseMCUGate(*event.MCUGateSuggestion)
	}
	if event.DeteriorationSignal != nil {
		return parseMCUGate(event.DeteriorationSignal.MCUGateSuggestion)
	}
	return models.GateSafe
}

// deriveSafetyTier determines the safety tier from the event's safety flags.
func (b *SignalCardBuilder) deriveSafetyTier(event *models.ClinicalSignalEvent) models.SafetyTier {
	for _, flag := range event.SafetyFlags {
		if flag.Severity == "IMMEDIATE" {
			return models.SafetyImmediate
		}
	}
	for _, flag := range event.SafetyFlags {
		if flag.Severity == "URGENT" {
			return models.SafetyUrgent
		}
	}
	return models.SafetyRoutine
}

// parseMCUGate converts a string gate value to a typed MCUGate.
func parseMCUGate(gate string) models.MCUGate {
	switch gate {
	case "HALT":
		return models.GateHalt
	case "PAUSE":
		return models.GatePause
	case "MODIFY":
		return models.GateModify
	default:
		return models.GateSafe
	}
}
