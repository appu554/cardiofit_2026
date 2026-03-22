package asha

import (
	"fmt"
	"time"

	"go.uber.org/zap"
)

type SyncService struct {
	queue  *OfflineQueue
	logger *zap.Logger
}

func NewSyncService(queue *OfflineQueue, logger *zap.Logger) *SyncService {
	return &SyncService{queue: queue, logger: logger}
}

func (s *SyncService) ReconcileOfflineBatch(sub TabletSubmission) ([]SubmissionResult, error) {
	results := make([]SubmissionResult, 0, len(sub.Slots))

	lastSeq, err := s.queue.GetLastProcessedSeq(sub.DeviceID)
	if err != nil {
		return nil, fmt.Errorf("get last processed seq: %w", err)
	}

	if sub.SyncSeqNo <= lastSeq {
		s.logger.Info("duplicate sync batch, returning cached results",
			zap.String("device_id", sub.DeviceID),
			zap.Int64("seq", sub.SyncSeqNo),
		)
		for _, slot := range sub.Slots {
			results = append(results, SubmissionResult{
				SlotName: slot.SlotName,
				Status:   "ACCEPTED",
				Message:  "already processed",
			})
		}
		return results, nil
	}

	for _, slot := range sub.Slots {
		result := s.reconcileSlot(sub, slot)
		results = append(results, result)
	}

	if err := s.queue.RecordProcessedSeq(sub.DeviceID, sub.SyncSeqNo); err != nil {
		s.logger.Error("failed to record processed seq", zap.Error(err))
	}

	return results, nil
}

func (s *SyncService) reconcileSlot(sub TabletSubmission, slot SlotEntry) SubmissionResult {
	existing, err := s.queue.GetExistingSlotValue(sub.PatientID.String(), slot.SlotName)
	if err != nil {
		s.logger.Error("conflict check failed", zap.Error(err))
		return SubmissionResult{
			SlotName: slot.SlotName,
			Status:   "ERROR",
			Message:  "conflict check failed",
		}
	}

	if existing != nil {
		if existing.CollectedAt.After(sub.CollectedAt) && existing.Source != "ASHA" {
			s.logger.Info("offline ASHA data takes priority over self-reported",
				zap.String("slot", slot.SlotName),
			)
		} else if existing.CollectedAt.After(sub.CollectedAt) && existing.Source == "ASHA" {
			return SubmissionResult{
				SlotName: slot.SlotName,
				Status:   "CONFLICT",
				Message:  fmt.Sprintf("newer ASHA value exists from %s", existing.CollectedAt.Format(time.RFC3339)),
			}
		}
	}

	return SubmissionResult{
		SlotName: slot.SlotName,
		Status:   "ACCEPTED",
	}
}
