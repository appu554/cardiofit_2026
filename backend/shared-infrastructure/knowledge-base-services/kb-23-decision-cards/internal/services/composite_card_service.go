package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-23-decision-cards/internal/models"
)

// compositeWindow is the rolling window within which active cards are
// aggregated into a single composite signal for the clinician dashboard.
const compositeWindow = 72 * time.Hour

// Synthesize aggregates all ACTIVE decision cards for a patient created within
// the last 72 hours into a single CompositeCardSignal. If no active cards
// exist the method returns (nil, nil). The resulting composite is persisted to
// the database before it is returned.
func (s *CompositeCardService) Synthesize(ctx context.Context, patientID uuid.UUID) (*models.CompositeCardSignal, error) {
	windowStart := time.Now().Add(-compositeWindow)

	var cards []models.DecisionCard
	result := s.db.DB.WithContext(ctx).
		Where("patient_id = ? AND status = ? AND created_at >= ?",
			patientID, models.StatusActive, windowStart).
		Order("created_at ASC").
		Find(&cards)

	if result.Error != nil {
		return nil, fmt.Errorf("query active cards for composite: %w", result.Error)
	}

	if len(cards) == 0 {
		return nil, nil
	}

	composite := s.buildComposite(patientID, cards)

	if err := s.db.DB.WithContext(ctx).Create(composite).Error; err != nil {
		return nil, fmt.Errorf("save composite card signal: %w", err)
	}

	s.metrics.CompositeCardsCreated.Inc()

	s.log.Info("composite card synthesised",
		zap.String("composite_id", composite.CompositeID.String()),
		zap.String("patient_id", patientID.String()),
		zap.Int("card_count", len(cards)),
		zap.String("most_restrictive_gate", string(composite.MostRestrictiveGate)),
		zap.Bool("urgency_upgraded", composite.UrgencyUpgraded),
	)

	return composite, nil
}

// GetLatestComposite returns the most recent composite card signal for the
// given patient, or (nil, nil) if none exists.
func (s *CompositeCardService) GetLatestComposite(ctx context.Context, patientID uuid.UUID) (*models.CompositeCardSignal, error) {
	var composite models.CompositeCardSignal
	result := s.db.DB.WithContext(ctx).
		Where("patient_id = ?", patientID).
		Order("created_at DESC").
		First(&composite)

	if result.Error != nil {
		if result.Error.Error() == "record not found" {
			return nil, nil
		}
		return nil, fmt.Errorf("query latest composite: %w", result.Error)
	}

	return &composite, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// buildComposite creates a CompositeCardSignal from one or more active cards.
func (s *CompositeCardService) buildComposite(patientID uuid.UUID, cards []models.DecisionCard) *models.CompositeCardSignal {
	// Collect card IDs and determine aggregate fields in a single pass.
	cardIDs := make([]string, 0, len(cards))
	gate := cards[0].MCUGate
	urgencyUpgraded := false
	clinicianParts := make([]string, 0, len(cards))
	hindiParts := make([]string, 0, len(cards))

	for _, c := range cards {
		cardIDs = append(cardIDs, c.CardID.String())
		gate = models.MostRestrictive(gate, c.MCUGate)

		if c.SafetyTier == models.SafetyImmediate {
			urgencyUpgraded = true
		}

		if c.ClinicianSummary != "" {
			clinicianParts = append(clinicianParts, c.ClinicianSummary)
		}
		if c.PatientSummaryHi != "" {
			hindiParts = append(hindiParts, c.PatientSummaryHi)
		}
	}

	// Marshal card IDs into a JSONB array.
	cardIDsJSON, err := json.Marshal(cardIDs)
	if err != nil {
		// Marshalling a []string cannot realistically fail, but guard anyway.
		s.log.Error("failed to marshal card IDs", zap.Error(err))
		cardIDsJSON = []byte("[]")
	}

	// cards are ordered ASC by created_at so the first is oldest, last is newest.
	windowStart := cards[0].CreatedAt
	windowEnd := cards[len(cards)-1].CreatedAt

	return &models.CompositeCardSignal{
		PatientID:           patientID,
		CardIDs:             models.JSONB(cardIDsJSON),
		MostRestrictiveGate: gate,
		RecurrenceCount:     len(cards),
		UrgencyUpgraded:     urgencyUpgraded,
		SynthesisSummaryEn:  strings.Join(clinicianParts, " | "),
		SynthesisSummaryHi:  strings.Join(hindiParts, " | "),
		WindowStart:         windowStart,
		WindowEnd:           windowEnd,
		CreatedAt:           time.Now(),
	}
}
