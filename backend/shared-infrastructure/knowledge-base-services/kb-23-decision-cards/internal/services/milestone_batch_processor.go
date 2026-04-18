package services

import (
	"context"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// milestoneDueRow mirrors KB-20's TransitionMilestone for DB queries.
// Option α coupling — manual sync, caught by integration tests.
type milestoneDueRow struct {
	ID               string     `gorm:"column:id"`
	TransitionID     string     `gorm:"column:transition_id"`
	MilestoneType    string     `gorm:"column:milestone_type"`
	ScheduledFor     time.Time  `gorm:"column:scheduled_for"`
	CompletedAt      *time.Time `gorm:"column:completed_at"`
	CompletionStatus string     `gorm:"column:completion_status"`
}

func (milestoneDueRow) TableName() string { return "transition_milestones" }

// MilestoneBatchProcessor polls for due milestones and triggers them.
// Runs as a background goroutine, checking every 15 minutes for
// milestones where ScheduledFor is in the past and CompletionStatus
// is SCHEDULED.
type MilestoneBatchProcessor struct {
	db  *gorm.DB
	log *zap.Logger
}

// NewMilestoneBatchProcessor creates a batch processor.
func NewMilestoneBatchProcessor(db *gorm.DB, log *zap.Logger) *MilestoneBatchProcessor {
	if log == nil {
		log = zap.NewNop()
	}
	return &MilestoneBatchProcessor{db: db, log: log}
}

// Start begins the background polling loop. Cancellable via context.
func (p *MilestoneBatchProcessor) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()

		p.processDueMilestones()

		for {
			select {
			case <-ctx.Done():
				p.log.Info("milestone batch processor shutting down")
				return
			case <-ticker.C:
				p.processDueMilestones()
			}
		}
	}()
}

// processDueMilestones finds all SCHEDULED milestones past their due time
// and marks them as TRIGGERED. A grace period of 48 hours is allowed —
// milestones past their ScheduledFor + 48h are marked MISSED.
func (p *MilestoneBatchProcessor) processDueMilestones() {
	if p.db == nil {
		return
	}

	now := time.Now().UTC()
	gracePeriod := 48 * time.Hour

	var dueMilestones []milestoneDueRow
	if err := p.db.Where("completion_status = ? AND scheduled_for < ?",
		"SCHEDULED", now).Find(&dueMilestones).Error; err != nil {
		p.log.Error("milestone batch: failed to query", zap.Error(err))
		return
	}

	for i := range dueMilestones {
		m := &dueMilestones[i]

		if now.After(m.ScheduledFor.Add(gracePeriod)) {
			m.CompletionStatus = "MISSED"
			completedAt := now
			m.CompletedAt = &completedAt
			if err := p.db.Save(m).Error; err != nil {
				p.log.Error("milestone batch: failed to mark MISSED",
					zap.String("id", m.ID), zap.Error(err))
			} else {
				p.log.Warn("milestone MISSED",
					zap.String("type", m.MilestoneType),
					zap.String("transition", m.TransitionID))
			}
			continue
		}

		m.CompletionStatus = "TRIGGERED"
		if err := p.db.Save(m).Error; err != nil {
			p.log.Error("milestone batch: failed to mark TRIGGERED",
				zap.String("id", m.ID), zap.Error(err))
		} else {
			p.log.Info("milestone TRIGGERED",
				zap.String("type", m.MilestoneType),
				zap.String("transition", m.TransitionID))
		}
	}

	if len(dueMilestones) > 0 {
		p.log.Info("milestone batch completed", zap.Int("processed", len(dueMilestones)))
	}
}
