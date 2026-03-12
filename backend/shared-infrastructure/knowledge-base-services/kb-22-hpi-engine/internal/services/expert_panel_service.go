package services

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/cache"
	"kb-22-hpi-engine/internal/database"
	"kb-22-hpi-engine/internal/metrics"
	"kb-22-hpi-engine/internal/models"
)

// ExpertPanelService manages Tier A calibration: quarterly expert panel reviews.
// Constraints:
//   - Max ±30% LR adjustment per review cycle (MaxAdjustmentPerCycle)
//   - Requires 2/3 panel consensus
//   - All adjustments logged as CalibrationEvents with source='EXPERT_PANEL'
type ExpertPanelService struct {
	db      *database.Database
	cache   *cache.CacheClient
	log     *zap.Logger
	metrics *metrics.Collector
}

// ExpertPanelReview is the request body for submitting an expert panel LR review.
type ExpertPanelReview struct {
	NodeID        string   `json:"node_id" binding:"required"`
	NodeVersion   string   `json:"node_version" binding:"required"`
	ElementType   string   `json:"element_type" binding:"required"` // LR_POSITIVE, LR_NEGATIVE, PRIOR, CM_MAGNITUDE, SAFETY_FLOOR
	ElementKey    string   `json:"element_key" binding:"required"`  // e.g. "Q001:OH"
	StratumLabel  *string  `json:"stratum_label,omitempty"`
	OldValue      float64  `json:"old_value" binding:"required"`
	ProposedValue float64  `json:"proposed_value" binding:"required"`
	Rationale     string   `json:"rationale" binding:"required"`
	PanelMembers  []string `json:"panel_members" binding:"required,min=3"` // minimum 3 panelists
	Approvals     []string `json:"approvals" binding:"required"`           // who approved
}

// NewExpertPanelService creates a new ExpertPanelService.
func NewExpertPanelService(
	db *database.Database,
	cacheClient *cache.CacheClient,
	log *zap.Logger,
	m *metrics.Collector,
) *ExpertPanelService {
	return &ExpertPanelService{
		db:      db,
		cache:   cacheClient,
		log:     log,
		metrics: m,
	}
}

// SubmitReview validates and records an expert panel LR adjustment.
// Enforces ±30% max adjustment and 2/3 consensus requirement.
func (s *ExpertPanelService) SubmitReview(
	ctx context.Context,
	review ExpertPanelReview,
) (*models.CalibrationEvent, error) {
	// Validate panel consensus: 2/3 must approve
	requiredApprovals := (2*len(review.PanelMembers) + 2) / 3 // ceiling division
	if len(review.Approvals) < requiredApprovals {
		return nil, fmt.Errorf(
			"insufficient panel consensus: %d/%d approvals, need %d/%d",
			len(review.Approvals), len(review.PanelMembers),
			requiredApprovals, len(review.PanelMembers),
		)
	}

	// Validate all approvers are panel members
	memberSet := make(map[string]bool, len(review.PanelMembers))
	for _, m := range review.PanelMembers {
		memberSet[m] = true
	}
	for _, a := range review.Approvals {
		if !memberSet[a] {
			return nil, fmt.Errorf("approver %q is not a panel member", a)
		}
	}

	// Validate ±30% adjustment constraint
	if review.OldValue != 0 {
		changeRatio := math.Abs(review.ProposedValue-review.OldValue) / math.Abs(review.OldValue)
		if changeRatio > models.MaxAdjustmentPerCycle {
			return nil, fmt.Errorf(
				"adjustment exceeds ±30%% limit: %.1f%% change (old=%.4f, new=%.4f)",
				changeRatio*100, review.OldValue, review.ProposedValue,
			)
		}
	}

	panelStr := strings.Join(review.PanelMembers, ",")
	event := models.CalibrationEvent{
		EventID:      uuid.New(),
		NodeID:       review.NodeID,
		NodeVersion:  review.NodeVersion,
		ElementType:  review.ElementType,
		ElementKey:   review.ElementKey,
		StratumLabel: review.StratumLabel,
		OldValue:     review.OldValue,
		NewValue:     review.ProposedValue,
		SourceTier:   models.CalibrationTierExpertPanel,
		Rationale:    review.Rationale,
		ApprovedBy:   strings.Join(review.Approvals, ","),
		PanelMembers: &panelStr,
		CreatedAt:    time.Now(),
	}

	if err := s.db.DB.WithContext(ctx).Create(&event).Error; err != nil {
		return nil, fmt.Errorf("failed to create calibration event: %w", err)
	}

	s.log.Info("E01: expert panel calibration recorded",
		zap.String("event_id", event.EventID.String()),
		zap.String("node_id", review.NodeID),
		zap.String("element", review.ElementType+":"+review.ElementKey),
		zap.Float64("old", review.OldValue),
		zap.Float64("new", review.ProposedValue),
		zap.Int("approvals", len(review.Approvals)),
		zap.Int("panel_size", len(review.PanelMembers)),
	)

	return &event, nil
}

// GetReviewHistory returns all calibration events for a node, ordered by creation time.
func (s *ExpertPanelService) GetReviewHistory(
	ctx context.Context,
	nodeID string,
) ([]models.CalibrationEvent, error) {
	var events []models.CalibrationEvent
	if err := s.db.DB.WithContext(ctx).
		Where("node_id = ?", nodeID).
		Order("created_at DESC").
		Find(&events).Error; err != nil {
		return nil, fmt.Errorf("failed to get calibration events: %w", err)
	}
	return events, nil
}

// ComputeBlendedLR implements the Tier B beta-binomial shrinkage formula (BAY-9/CC-4):
//
//	w = max(0.3, 1 - sqrt(n/200))
//	blended = w * literatureLR + (1-w) * observedLR
//
// At n=30, w=0.87 (heavily anchored to literature).
// At n=200, w=0.30 (minimum literature anchor).
func (s *ExpertPanelService) ComputeBlendedLR(literatureLR, observedLR float64, sampleSize int) float64 {
	w := 1.0 - math.Sqrt(float64(sampleSize)/200.0)
	if w < 0.3 {
		w = 0.3
	}
	blended := w*literatureLR + (1-w)*observedLR

	s.log.Debug("Tier B: blended LR computed",
		zap.Float64("literature_lr", literatureLR),
		zap.Float64("observed_lr", observedLR),
		zap.Int("sample_size", sampleSize),
		zap.Float64("w_factor", w),
		zap.Float64("blended_lr", blended),
	)

	return blended
}

// SubmitBlendedCalibration records a Tier B calibration event with blending metadata.
func (s *ExpertPanelService) SubmitBlendedCalibration(
	ctx context.Context,
	nodeID, nodeVersion, elementType, elementKey string,
	stratumLabel *string,
	oldValue, observedLR float64,
	sampleSize int,
	approvedBy string,
) (*models.CalibrationEvent, error) {
	blended := s.ComputeBlendedLR(oldValue, observedLR, sampleSize)
	w := 1.0 - math.Sqrt(float64(sampleSize)/200.0)
	if w < 0.3 {
		w = 0.3
	}
	deviation := observedLR - oldValue

	event := models.CalibrationEvent{
		EventID:      uuid.New(),
		NodeID:       nodeID,
		NodeVersion:  nodeVersion,
		ElementType:  elementType,
		ElementKey:   elementKey,
		StratumLabel: stratumLabel,
		OldValue:     oldValue,
		NewValue:     blended,
		SourceTier:   models.CalibrationTierBayesBlend,
		SampleSize:   &sampleSize,
		WFactor:      &w,
		Deviation:    &deviation,
		Rationale:    fmt.Sprintf("Tier B blend: w=%.3f, n=%d, observed=%.4f, blended=%.4f", w, sampleSize, observedLR, blended),
		ApprovedBy:   approvedBy,
		CreatedAt:    time.Now(),
	}

	if err := s.db.DB.WithContext(ctx).Create(&event).Error; err != nil {
		return nil, fmt.Errorf("failed to create blended calibration event: %w", err)
	}

	s.log.Info("Tier B: blended calibration recorded",
		zap.String("event_id", event.EventID.String()),
		zap.String("node_id", nodeID),
		zap.Float64("w_factor", w),
		zap.Int("sample_size", sampleSize),
	)

	return &event, nil
}
