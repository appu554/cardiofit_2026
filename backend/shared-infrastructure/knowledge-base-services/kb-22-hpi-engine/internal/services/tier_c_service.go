package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/cache"
	"kb-22-hpi-engine/internal/database"
	"kb-22-hpi-engine/internal/metrics"
	"kb-22-hpi-engine/internal/models"
)

// TierCService implements E03: Data-Driven Calibration for Month 18+.
//
// When a node/stratum accumulates ≥200 adjudicated cases, this service
// computes empirical likelihood ratios using logistic regression with
// hierarchical shrinkage priors for rare differentials.
//
// The calibration flow:
//  1. Query adjudicated CalibrationRecords for a node/stratum
//  2. Compute observed LR+ and LR- per question×differential pair
//  3. Apply hierarchical shrinkage: rare differentials (n<20) are pulled
//     toward the population mean to prevent extreme LR estimates
//  4. Create CalibrationEvent records for governance audit
//  5. Require governance committee approval before YAML update
//
// Governance: Quarterly review by calibration committee.
// Each adjustment creates an immutable CalibrationEvent with
// source_tier=DATA_DRIVEN, deviation, sample_size, and approval.
type TierCService struct {
	db      *database.Database
	cache   *cache.CacheClient
	log     *zap.Logger
	metrics *metrics.Collector
}

// TierCThreshold is the minimum adjudicated cases for Tier C activation.
const TierCThreshold = 200

// RareDifferentialThreshold is the minimum cases for a differential to
// avoid shrinkage toward the population mean.
const RareDifferentialThreshold = 20

// MinShrinkageWeight prevents complete shrinkage for rare differentials.
const MinShrinkageWeight = 0.30

// NewTierCService creates the data-driven calibration service.
func NewTierCService(
	db *database.Database,
	cacheClient *cache.CacheClient,
	log *zap.Logger,
	m *metrics.Collector,
) *TierCService {
	return &TierCService{db: db, cache: cacheClient, log: log, metrics: m}
}

// EmpiricalLR holds computed likelihood ratios for a question×differential pair.
type EmpiricalLR struct {
	QuestionID     string  `json:"question_id"`
	DifferentialID string  `json:"differential_id"`
	LRPositive     float64 `json:"lr_positive"`
	LRNegative     float64 `json:"lr_negative"`
	SampleSize     int     `json:"sample_size"`
	ShrinkageW     float64 `json:"shrinkage_w"` // 1.0 = no shrinkage, 0.3 = max shrinkage
}

// TierCProposal is a governance-gated calibration proposal.
type TierCProposal struct {
	ProposalID   uuid.UUID     `json:"proposal_id"`
	NodeID       string        `json:"node_id"`
	StratumLabel string        `json:"stratum_label"`
	TotalCases   int           `json:"total_cases"`
	EmpiricalLRs []EmpiricalLR `json:"empirical_lrs"`
	ComputedAt   time.Time     `json:"computed_at"`
	Status       string        `json:"status"` // PENDING_REVIEW, APPROVED, REJECTED
	ReviewedBy   string        `json:"reviewed_by,omitempty"`
	ReviewedAt   *time.Time    `json:"reviewed_at,omitempty"`
	Rationale    string        `json:"rationale,omitempty"`
}

// ComputeEmpiricalLRs generates empirical LRs from adjudicated data.
//
// Algorithm:
//  1. Load all CalibrationRecords for node/stratum with N ≥ TierCThreshold
//  2. For each question, count answer distributions per confirmed diagnosis
//  3. Compute sensitivity = P(answer=yes | diagnosis=D)
//  4. Compute specificity = P(answer=no | diagnosis≠D)
//  5. LR+ = sensitivity / (1 - specificity)
//  6. LR- = (1 - sensitivity) / specificity
//  7. Apply hierarchical shrinkage for differentials with n < 20
//
// Returns a TierCProposal (not yet applied — requires governance approval).
func (s *TierCService) ComputeEmpiricalLRs(
	ctx context.Context,
	nodeID string,
	stratum string,
) (*TierCProposal, error) {
	// Step 1: load adjudicated records
	var records []models.CalibrationRecord
	query := s.db.DB.WithContext(ctx).
		Where("node_id = ? AND stratum_label = ?", nodeID, stratum).
		Order("adjudicated_at ASC")

	if err := query.Find(&records).Error; err != nil {
		return nil, fmt.Errorf("failed to load calibration records: %w", err)
	}

	totalCases := len(records)
	if totalCases < TierCThreshold {
		return nil, fmt.Errorf(
			"insufficient cases for Tier C: %d < %d threshold",
			totalCases, TierCThreshold,
		)
	}

	// Step 2: build answer×diagnosis contingency tables
	type ContingencyKey struct {
		QuestionID     string
		DifferentialID string
	}

	// Counts: [question][differential] → {yesWithD, noWithD, yesWithoutD, noWithoutD}
	type Cell struct {
		yesWithD      int
		noWithD       int
		yesWithoutD   int
		noWithoutD    int
		totalWithD    int
		totalWithoutD int
	}
	tables := make(map[ContingencyKey]*Cell)

	// Collect all question×answer pairs and confirmed diagnoses
	for _, rec := range records {
		var qa map[string]string
		if err := json.Unmarshal(rec.QuestionAnswers, &qa); err != nil {
			continue
		}

		confirmedDx := rec.ConfirmedDiagnosis

		// For each question answered in this session
		for qID, answer := range qa {
			isYes := answer == "yes" || answer == "true" || answer == "1"

			// Update contingency for every known differential
			for _, diffID := range s.getKnownDifferentials(records) {
				key := ContingencyKey{QuestionID: qID, DifferentialID: diffID}
				cell, ok := tables[key]
				if !ok {
					cell = &Cell{}
					tables[key] = cell
				}

				if confirmedDx == diffID {
					cell.totalWithD++
					if isYes {
						cell.yesWithD++
					} else {
						cell.noWithD++
					}
				} else {
					cell.totalWithoutD++
					if isYes {
						cell.yesWithoutD++
					} else {
						cell.noWithoutD++
					}
				}
			}
		}
	}

	// Step 3-6: compute LR+ and LR- from contingency tables
	// First compute population-mean LRs for shrinkage target
	popLRPos, popLRNeg := populationMeanLR()

	empiricalLRs := make([]EmpiricalLR, 0, len(tables))
	for key, cell := range tables {
		if cell.totalWithD == 0 || cell.totalWithoutD == 0 {
			continue
		}

		// Sensitivity = P(yes | D)
		sensitivity := float64(cell.yesWithD) / float64(cell.totalWithD)
		// False positive rate = P(yes | ¬D)
		falsePositiveRate := float64(cell.yesWithoutD) / float64(cell.totalWithoutD)
		// Specificity = 1 - FPR
		specificity := 1.0 - falsePositiveRate

		// LR+ = sensitivity / (1 - specificity) = sensitivity / FPR
		// LR- = (1 - sensitivity) / specificity
		var rawLRPos, rawLRNeg float64
		if falsePositiveRate > 0.001 {
			rawLRPos = sensitivity / falsePositiveRate
		} else {
			rawLRPos = sensitivity / 0.001 // cap to avoid infinity
		}
		if specificity > 0.001 {
			rawLRNeg = (1.0 - sensitivity) / specificity
		} else {
			rawLRNeg = (1.0 - sensitivity) / 0.001
		}

		// Step 7: hierarchical shrinkage for rare differentials
		shrinkageW := computeShrinkageWeight(cell.totalWithD)
		lrPos := shrinkageW*rawLRPos + (1.0-shrinkageW)*popLRPos
		lrNeg := shrinkageW*rawLRNeg + (1.0-shrinkageW)*popLRNeg

		// Clamp LRs to clinically reasonable range [0.01, 100]
		lrPos = clampLR(lrPos)
		lrNeg = clampLR(lrNeg)

		empiricalLRs = append(empiricalLRs, EmpiricalLR{
			QuestionID:     key.QuestionID,
			DifferentialID: key.DifferentialID,
			LRPositive:     math.Round(lrPos*1000) / 1000,
			LRNegative:     math.Round(lrNeg*1000) / 1000,
			SampleSize:     cell.totalWithD,
			ShrinkageW:     math.Round(shrinkageW*100) / 100,
		})
	}

	proposal := &TierCProposal{
		ProposalID:   uuid.New(),
		NodeID:       nodeID,
		StratumLabel: stratum,
		TotalCases:   totalCases,
		EmpiricalLRs: empiricalLRs,
		ComputedAt:   time.Now(),
		Status:       "PENDING_REVIEW",
	}

	s.log.Info("E03: Tier C empirical LRs computed",
		zap.String("node_id", nodeID),
		zap.String("stratum", stratum),
		zap.Int("total_cases", totalCases),
		zap.Int("lr_pairs", len(empiricalLRs)),
	)

	return proposal, nil
}

// ApproveProposal applies an approved Tier C proposal by creating
// CalibrationEvent records for each LR adjustment.
func (s *TierCService) ApproveProposal(
	ctx context.Context,
	proposal *TierCProposal,
	approvedBy string,
	rationale string,
) ([]models.CalibrationEvent, error) {
	if proposal.Status != "PENDING_REVIEW" {
		return nil, fmt.Errorf("proposal %s is not pending review (status=%s)",
			proposal.ProposalID, proposal.Status)
	}

	now := time.Now()
	proposal.Status = "APPROVED"
	proposal.ReviewedBy = approvedBy
	proposal.ReviewedAt = &now
	proposal.Rationale = rationale

	var events []models.CalibrationEvent
	for _, lr := range proposal.EmpiricalLRs {
		// Create event for LR+ adjustment
		sampleSize := lr.SampleSize
		events = append(events, models.CalibrationEvent{
			EventID:      uuid.New(),
			NodeID:       proposal.NodeID,
			NodeVersion:  "tier-c-auto",
			ElementType:  "LR_POSITIVE",
			ElementKey:   fmt.Sprintf("%s:%s", lr.QuestionID, lr.DifferentialID),
			StratumLabel: &proposal.StratumLabel,
			OldValue:     0, // populated by caller from current YAML
			NewValue:     lr.LRPositive,
			SourceTier:   models.CalibrationTierDataDriven,
			SampleSize:   &sampleSize,
			Deviation:    &lr.ShrinkageW,
			Rationale:    rationale,
			ApprovedBy:   approvedBy,
			CreatedAt:    now,
		})

		// Create event for LR- adjustment
		events = append(events, models.CalibrationEvent{
			EventID:      uuid.New(),
			NodeID:       proposal.NodeID,
			NodeVersion:  "tier-c-auto",
			ElementType:  "LR_NEGATIVE",
			ElementKey:   fmt.Sprintf("%s:%s", lr.QuestionID, lr.DifferentialID),
			StratumLabel: &proposal.StratumLabel,
			OldValue:     0,
			NewValue:     lr.LRNegative,
			SourceTier:   models.CalibrationTierDataDriven,
			SampleSize:   &sampleSize,
			Deviation:    &lr.ShrinkageW,
			Rationale:    rationale,
			ApprovedBy:   approvedBy,
			CreatedAt:    now,
		})
	}

	// Persist all events in a transaction
	tx := s.db.DB.WithContext(ctx).Begin()
	for i := range events {
		if err := tx.Create(&events[i]).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("failed to create calibration event: %w", err)
		}
	}
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit calibration events: %w", err)
	}

	s.log.Info("E03: Tier C proposal approved",
		zap.String("proposal_id", proposal.ProposalID.String()),
		zap.String("approved_by", approvedBy),
		zap.Int("events_created", len(events)),
	)

	return events, nil
}

// getKnownDifferentials extracts the set of unique confirmed diagnoses.
func (s *TierCService) getKnownDifferentials(records []models.CalibrationRecord) []string {
	seen := make(map[string]bool)
	var result []string
	for _, r := range records {
		if !seen[r.ConfirmedDiagnosis] {
			seen[r.ConfirmedDiagnosis] = true
			result = append(result, r.ConfirmedDiagnosis)
		}
	}
	return result
}

// populationMeanLR returns the shrinkage target for hierarchical shrinkage.
// Uses uninformative priors: LR+=1.0, LR-=1.0 (no discriminative value).
// As the system matures with more data, this could be replaced with
// empirical population means computed across all nodes.
func populationMeanLR() (float64, float64) {
	return 1.0, 1.0
}

// computeShrinkageWeight determines how much to trust the empirical LR
// versus the population mean. Follows the same formula as Tier B but
// with higher threshold: w = max(0.3, 1 - sqrt(RareDifferentialThreshold / n))
//
// n ≥ 20: w approaches 1.0 (trust empirical data)
// n < 20: w pulled toward 0.3 (strong shrinkage toward population mean)
func computeShrinkageWeight(nCases int) float64 {
	if nCases <= 0 {
		return MinShrinkageWeight
	}
	w := 1.0 - math.Sqrt(float64(RareDifferentialThreshold)/float64(nCases))
	if w < MinShrinkageWeight {
		return MinShrinkageWeight
	}
	return w
}

// clampLR constrains likelihood ratios to the clinically reasonable range [0.01, 100].
func clampLR(lr float64) float64 {
	if lr < 0.01 {
		return 0.01
	}
	if lr > 100.0 {
		return 100.0
	}
	return lr
}
