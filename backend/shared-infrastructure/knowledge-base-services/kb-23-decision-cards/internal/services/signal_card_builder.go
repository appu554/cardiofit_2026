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

	rationale := fmt.Sprintf("signal event %s from node %s", event.EventID, event.NodeID)
	if event.SignalType == "MRI_DETERIORATION" {
		rationale = fmt.Sprintf("MRI score %.1f (%s) — %s; gate per MRI spec §7 table 8",
			event.MRIScore, event.MRICategory, event.MRISeverity)
	}

	card := &models.DecisionCard{
		CardID:                   uuid.New(),
		PatientID:                patientID,
		TemplateID:               templateID,
		NodeID:                   event.NodeID,
		CardSource:               models.SourceClinicalSignal,
		DiagnosticConfidenceTier: confidenceTier,
		MCUGate:                  gate,
		MCUGateRationale:         rationale,
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
			category == "improving" || category == "good" || category == "adequate" ||
			category == "negative-screen" {
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

	if event.SignalType == "MRI_DETERIORATION" {
		// MRI_DETERIORATION events from KB-26 always use the fixed template ID.
		// OPTIMAL / MILD_DYSREGULATION don't generate cards (no worsening boundary crossed).
		if event.MRICategory == "OPTIMAL" || event.MRICategory == "MILD_DYSREGULATION" || event.MRICategory == "" {
			return ""
		}
		return "MRI_DETERIORATION_01"
	}

	return ""
}

// deriveConfidenceTier maps data sufficiency + severity -> confidence tier.
func (b *SignalCardBuilder) deriveConfidenceTier(event *models.ClinicalSignalEvent) models.ConfidenceTier {
	// MRI_DETERIORATION: score-derived — always FIRM when a card is generated
	// (PublishDeteriorationEvent only fires on category boundary crossings).
	if event.SignalType == "MRI_DETERIORATION" {
		return models.TierFirm
	}

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
	// MRI_DETERIORATION events carry MCUGateSuggestion as a top-level string (not pointer).
	if event.SignalType == "MRI_DETERIORATION" {
		if event.MCUGateSuggestion != nil {
			return parseMCUGate(*event.MCUGateSuggestion)
		}
		// Fallback: derive from severity field sent by KB-26.
		switch event.MRISeverity {
		case "IMMEDIATE":
			return models.GateModify
		default:
			return models.GateSafe
		}
	}
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
	// MRI_DETERIORATION: safety tier from the MRISeverity field.
	if event.SignalType == "MRI_DETERIORATION" {
		switch event.MRISeverity {
		case "IMMEDIATE":
			return models.SafetyImmediate
		case "URGENT":
			return models.SafetyUrgent
		default:
			return models.SafetyRoutine
		}
	}
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
