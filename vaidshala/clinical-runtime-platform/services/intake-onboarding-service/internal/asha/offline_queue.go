package asha

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type OfflineQueue struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

func NewOfflineQueue(db *pgxpool.Pool, logger *zap.Logger) *OfflineQueue {
	return &OfflineQueue{db: db, logger: logger}
}

type DeviceSyncStatus struct {
	DeviceID      string    `json:"device_id"`
	LastSyncSeqNo int64     `json:"last_sync_seq_no"`
	LastSyncAt    time.Time `json:"last_sync_at"`
	PendingCount  int       `json:"pending_count"`
	ConflictCount int       `json:"conflict_count"`
}

type ExistingSlotValue struct {
	SlotName    string
	Value       interface{}
	Source      string
	CollectedAt time.Time
}

func (q *OfflineQueue) GetDeviceSyncStatus(deviceID string) (*DeviceSyncStatus, error) {
	ctx := context.Background()
	var status DeviceSyncStatus
	status.DeviceID = deviceID

	err := q.db.QueryRow(ctx,
		`SELECT COALESCE(last_sync_seq_no, 0), COALESCE(last_sync_at, now())
		 FROM asha_device_sync WHERE device_id = $1`,
		deviceID,
	).Scan(&status.LastSyncSeqNo, &status.LastSyncAt)

	if err != nil {
		status.LastSyncSeqNo = 0
		status.LastSyncAt = time.Time{}
	}

	return &status, nil
}

func (q *OfflineQueue) GetLastProcessedSeq(deviceID string) (int64, error) {
	ctx := context.Background()
	var seq int64
	err := q.db.QueryRow(ctx,
		`SELECT COALESCE(last_sync_seq_no, 0) FROM asha_device_sync WHERE device_id = $1`,
		deviceID,
	).Scan(&seq)
	if err != nil {
		return 0, nil
	}
	return seq, nil
}

func (q *OfflineQueue) RecordProcessedSeq(deviceID string, seqNo int64) error {
	ctx := context.Background()
	_, err := q.db.Exec(ctx,
		`INSERT INTO asha_device_sync (device_id, last_sync_seq_no, last_sync_at)
		 VALUES ($1, $2, now())
		 ON CONFLICT (device_id)
		 DO UPDATE SET last_sync_seq_no = $2, last_sync_at = now()`,
		deviceID, seqNo,
	)
	return err
}

func (q *OfflineQueue) GetExistingSlotValue(patientID, slotName string) (*ExistingSlotValue, error) {
	ctx := context.Background()
	var val ExistingSlotValue

	err := q.db.QueryRow(ctx,
		`SELECT slot_name, source_channel, created_at
		 FROM slot_events
		 WHERE patient_id = $1 AND slot_name = $2
		 ORDER BY created_at DESC LIMIT 1`,
		patientID, slotName,
	).Scan(&val.SlotName, &val.Source, &val.CollectedAt)

	if err != nil {
		return nil, nil
	}
	return &val, nil
}
