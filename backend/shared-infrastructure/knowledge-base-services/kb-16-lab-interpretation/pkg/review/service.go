// Package review provides lab result review workflow management
package review

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"kb-16-lab-interpretation/pkg/integration"
	"kb-16-lab-interpretation/pkg/types"
)

// Service manages the review workflow for lab results
type Service struct {
	db         *gorm.DB
	kb14Client *integration.KB14Client
	log        *logrus.Entry
}

// NewService creates a new review service
func NewService(db *gorm.DB, kb14Client *integration.KB14Client, log *logrus.Entry) *Service {
	return &Service{
		db:         db,
		kb14Client: kb14Client,
		log:        log.WithField("component", "review_service"),
	}
}

// CreateReview creates a review record for a result
func (s *Service) CreateReview(ctx context.Context, resultID uuid.UUID, isCritical bool) (*types.ResultReview, error) {
	review := &types.ResultReview{
		ID:        uuid.New(),
		ResultID:  resultID,
		Status:    types.ReviewStatusPending,
		CreatedAt: time.Now(),
	}

	if isCritical {
		review.Status = types.ReviewStatusCritical
	}

	if err := s.db.WithContext(ctx).Create(review).Error; err != nil {
		return nil, fmt.Errorf("failed to create review: %w", err)
	}

	return review, nil
}

// GetPendingReviews retrieves reviews awaiting action
func (s *Service) GetPendingReviews(ctx context.Context, filters types.PendingReviewFilters) ([]types.ResultReview, int, error) {
	var reviews []types.ResultReview
	var total int64

	query := s.db.WithContext(ctx).Model(&types.ResultReview{}).
		Where("status IN ?", []string{string(types.ReviewStatusPending), string(types.ReviewStatusCritical)})

	if filters.Priority != "" {
		if filters.Priority == "critical" {
			query = query.Where("status = ?", types.ReviewStatusCritical)
		}
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (filters.Page - 1) * filters.Limit
	if err := query.
		Order("CASE WHEN status = 'critical' THEN 0 ELSE 1 END, created_at DESC").
		Offset(offset).
		Limit(filters.Limit).
		Find(&reviews).Error; err != nil {
		return nil, 0, err
	}

	return reviews, int(total), nil
}

// GetCriticalQueue retrieves critical results requiring immediate attention
func (s *Service) GetCriticalQueue(ctx context.Context) ([]types.CriticalResult, error) {
	var reviews []types.ResultReview

	err := s.db.WithContext(ctx).
		Where("status = ?", types.ReviewStatusCritical).
		Order("created_at ASC").
		Find(&reviews).Error

	if err != nil {
		return nil, err
	}

	// Convert to CriticalResult format
	critical := make([]types.CriticalResult, 0, len(reviews))
	for _, r := range reviews {
		// Calculate wait time
		waitMinutes := int(time.Since(r.CreatedAt).Minutes())
		slaBreached := waitMinutes > 30 // 30-minute SLA for critical values

		critical = append(critical, types.CriticalResult{
			ReviewID:     r.ID,
			ResultID:     r.ResultID,
			CreatedAt:    r.CreatedAt,
			WaitMinutes:  waitMinutes,
			SLABreached:  slaBreached,
			KB14TaskID:   r.KB14TaskID,
		})
	}

	return critical, nil
}

// Acknowledge marks a result as acknowledged
func (s *Service) Acknowledge(ctx context.Context, req types.AcknowledgeRequest) error {
	result := s.db.WithContext(ctx).Model(&types.ResultReview{}).
		Where("result_id = ?", req.ResultID).
		Updates(map[string]interface{}{
			"acknowledged_by": req.AcknowledgedBy,
			"acknowledged_at": time.Now(),
			"status":          types.ReviewStatusAcknowledged,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to acknowledge: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("no review found for result ID: %s", req.ResultID)
	}

	s.log.WithFields(logrus.Fields{
		"result_id":       req.ResultID,
		"acknowledged_by": req.AcknowledgedBy,
	}).Info("Result acknowledged")

	return nil
}

// CompleteReview marks a review as completed with action taken
func (s *Service) CompleteReview(ctx context.Context, req types.CompleteReviewRequest) error {
	result := s.db.WithContext(ctx).Model(&types.ResultReview{}).
		Where("result_id = ?", req.ResultID).
		Updates(map[string]interface{}{
			"reviewed_by":  req.ReviewedBy,
			"reviewed_at":  time.Now(),
			"review_notes": req.ReviewNotes,
			"action_taken": req.ActionTaken,
			"status":       types.ReviewStatusCompleted,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to complete review: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("no review found for result ID: %s", req.ResultID)
	}

	s.log.WithFields(logrus.Fields{
		"result_id":    req.ResultID,
		"reviewed_by":  req.ReviewedBy,
		"action_taken": req.ActionTaken,
	}).Info("Review completed")

	return nil
}

// GetReviewStats returns review statistics
func (s *Service) GetReviewStats(ctx context.Context) (*types.ReviewStats, error) {
	stats := &types.ReviewStats{}

	// Count by status
	var statusCounts []struct {
		Status string
		Count  int64
	}

	err := s.db.WithContext(ctx).Model(&types.ResultReview{}).
		Select("status, count(*) as count").
		Group("status").
		Find(&statusCounts).Error

	if err != nil {
		return nil, err
	}

	for _, sc := range statusCounts {
		switch types.ReviewStatus(sc.Status) {
		case types.ReviewStatusPending:
			stats.Pending = int(sc.Count)
		case types.ReviewStatusCritical:
			stats.Critical = int(sc.Count)
		case types.ReviewStatusAcknowledged:
			stats.Acknowledged = int(sc.Count)
		case types.ReviewStatusCompleted:
			stats.Completed = int(sc.Count)
		}
	}

	stats.Total = stats.Pending + stats.Critical + stats.Acknowledged + stats.Completed

	// Average acknowledgment time for completed reviews
	var avgAckTime struct {
		AvgMinutes float64
	}
	err = s.db.WithContext(ctx).Model(&types.ResultReview{}).
		Select("AVG(EXTRACT(EPOCH FROM (acknowledged_at - created_at))/60) as avg_minutes").
		Where("acknowledged_at IS NOT NULL").
		Scan(&avgAckTime).Error

	if err == nil {
		stats.AvgAcknowledgmentMinutes = avgAckTime.AvgMinutes
	}

	// Count SLA breaches (>30 min for critical)
	var breachCount int64
	err = s.db.WithContext(ctx).Model(&types.ResultReview{}).
		Where("status = ? AND EXTRACT(EPOCH FROM (NOW() - created_at))/60 > 30", types.ReviewStatusCritical).
		Count(&breachCount).Error

	if err == nil {
		stats.SLABreaches = int(breachCount)
	}

	return stats, nil
}

// CreateCriticalTask creates a KB-14 task for a critical result
func (s *Service) CreateCriticalTask(ctx context.Context, interpreted *types.InterpretedResult) (string, error) {
	if s.kb14Client == nil {
		s.log.Warn("KB-14 client not configured, skipping task creation")
		return "", nil
	}

	taskID, err := s.kb14Client.CreateCriticalLabTask(ctx, interpreted)
	if err != nil {
		s.log.WithError(err).Error("Failed to create KB-14 task")
		return "", err
	}

	// Update review with task ID
	s.db.WithContext(ctx).Model(&types.ResultReview{}).
		Where("result_id = ?", interpreted.Result.ID).
		Update("kb14_task_id", taskID)

	s.log.WithFields(logrus.Fields{
		"result_id":   interpreted.Result.ID,
		"kb14_task_id": taskID,
	}).Info("Created KB-14 task for critical result")

	return taskID, nil
}

// GetReviewByResultID retrieves a review by result ID
func (s *Service) GetReviewByResultID(ctx context.Context, resultID string) (*types.ResultReview, error) {
	var review types.ResultReview
	err := s.db.WithContext(ctx).
		Where("result_id = ?", resultID).
		First(&review).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &review, nil
}

// UpdateReviewWithTaskID updates a review with KB-14 task ID
func (s *Service) UpdateReviewWithTaskID(ctx context.Context, resultID uuid.UUID, taskID string) error {
	return s.db.WithContext(ctx).Model(&types.ResultReview{}).
		Where("result_id = ?", resultID).
		Update("kb14_task_id", taskID).Error
}
