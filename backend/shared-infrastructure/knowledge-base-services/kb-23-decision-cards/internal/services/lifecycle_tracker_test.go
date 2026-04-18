package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"kb-23-decision-cards/internal/models"
)

func newLifecycleTracker() *LifecycleTracker {
	return NewLifecycleTracker(nil, nil)
}

func TestTracker_RecordT0_CreatesLifecycle(t *testing.T) {
	tracker := newLifecycleTracker()

	lc := tracker.RecordT0("WEIGHT_GAIN", "RAPID", "patient-001", "URGENT", "kb-26", nil, nil)

	require.NotNil(t, lc)
	assert.Equal(t, string(models.LifecyclePendingNotification), lc.CurrentState)
	assert.Equal(t, "WEIGHT_GAIN", lc.DetectionType)
	assert.Equal(t, "RAPID", lc.DetectionSubtype)
	assert.Equal(t, "patient-001", lc.PatientID)
	assert.Equal(t, "URGENT", lc.TierAtDetection)
	assert.Equal(t, "kb-26", lc.SourceService)
	assert.False(t, lc.DetectedAt.IsZero())
	assert.Nil(t, lc.DeliveredAt)
	assert.Nil(t, lc.AcknowledgedAt)
	assert.Nil(t, lc.ActionedAt)
	assert.Nil(t, lc.ResolvedAt)
	assert.Nil(t, lc.DeliveryLatencyMs)
	assert.Nil(t, lc.AcknowledgmentLatencyMs)
	assert.Nil(t, lc.ActionLatencyMs)
	assert.Nil(t, lc.OutcomeLatencyMs)
	assert.Nil(t, lc.TotalLatencyMs)
}

func TestTracker_RecordT1_SetsDelivered(t *testing.T) {
	tracker := newLifecycleTracker()

	t0 := time.Now().UTC().Add(-10 * time.Minute)
	lc := &models.DetectionLifecycle{
		DetectedAt:   t0,
		CurrentState: string(models.LifecyclePendingNotification),
	}

	t1 := time.Now().UTC()
	tracker.RecordT1(lc, t1)

	require.NotNil(t, lc.DeliveredAt)
	assert.Equal(t, t1, *lc.DeliveredAt)
	assert.Equal(t, string(models.LifecycleNotified), lc.CurrentState)
	require.NotNil(t, lc.DeliveryLatencyMs)
	// ~600000ms (10 min) — allow 1s tolerance
	assert.InDelta(t, 600000, *lc.DeliveryLatencyMs, 1000)
}

func TestTracker_RecordT2_SetsAcknowledged(t *testing.T) {
	tracker := newLifecycleTracker()

	t1 := time.Now().UTC().Add(-5 * time.Minute)
	lc := &models.DetectionLifecycle{
		DetectedAt:   time.Now().UTC().Add(-15 * time.Minute),
		DeliveredAt:  &t1,
		CurrentState: string(models.LifecycleNotified),
	}

	t2 := time.Now().UTC()
	tracker.RecordT2(lc, "dr-smith", t2)

	require.NotNil(t, lc.AcknowledgedAt)
	assert.Equal(t, t2, *lc.AcknowledgedAt)
	assert.Equal(t, string(models.LifecycleAcknowledged), lc.CurrentState)
	assert.Equal(t, "dr-smith", lc.AssignedClinicianID)
	require.NotNil(t, lc.AcknowledgmentLatencyMs)
	// ~300000ms (5 min) — allow 1s tolerance
	assert.InDelta(t, 300000, *lc.AcknowledgmentLatencyMs, 1000)
}

func TestTracker_RecordT3_SetsActioned(t *testing.T) {
	tracker := newLifecycleTracker()

	t2 := time.Now().UTC().Add(-2 * time.Hour)
	lc := &models.DetectionLifecycle{
		DetectedAt:     time.Now().UTC().Add(-3 * time.Hour),
		AcknowledgedAt: &t2,
		CurrentState:   string(models.LifecycleAcknowledged),
	}

	t3 := time.Now().UTC()
	tracker.RecordT3(lc, "CALL_PATIENT", "Called about weight gain", t3)

	require.NotNil(t, lc.ActionedAt)
	assert.Equal(t, t3, *lc.ActionedAt)
	assert.Equal(t, string(models.LifecycleActioned), lc.CurrentState)
	assert.Equal(t, "CALL_PATIENT", lc.ActionType)
	assert.Equal(t, "Called about weight gain", lc.ActionDetail)
	require.NotNil(t, lc.ActionLatencyMs)
	// ~7200000ms (2h) — allow 1s tolerance
	assert.InDelta(t, 7200000, *lc.ActionLatencyMs, 1000)
}

func TestTracker_RecordT4_SetsResolved(t *testing.T) {
	tracker := newLifecycleTracker()

	t0 := time.Now().UTC().Add(-48 * time.Hour)
	t3 := time.Now().UTC().Add(-24 * time.Hour)
	lc := &models.DetectionLifecycle{
		DetectedAt:   t0,
		ActionedAt:   &t3,
		CurrentState: string(models.LifecycleActioned),
	}

	t4 := time.Now().UTC()
	tracker.RecordT4(lc, "Weight returned to baseline", t4)

	require.NotNil(t, lc.ResolvedAt)
	assert.Equal(t, t4, *lc.ResolvedAt)
	assert.Equal(t, string(models.LifecycleResolved), lc.CurrentState)
	assert.Equal(t, "Weight returned to baseline", lc.OutcomeDescription)
	require.NotNil(t, lc.OutcomeLatencyMs)
	// ~86400000ms (24h) — allow 1s tolerance
	assert.InDelta(t, 86400000, *lc.OutcomeLatencyMs, 1000)
	require.NotNil(t, lc.TotalLatencyMs)
	// ~172800000ms (48h) — allow 1s tolerance
	assert.InDelta(t, 172800000, *lc.TotalLatencyMs, 1000)
}

func TestTracker_FullLifecycle_AllLatencies(t *testing.T) {
	tracker := newLifecycleTracker()

	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(5 * time.Minute)
	t2 := t0.Add(30 * time.Minute)
	t3 := t0.Add(2 * time.Hour)
	t4 := t0.Add(48 * time.Hour)

	// Create lifecycle manually with fixed T0
	lc := &models.DetectionLifecycle{
		DetectedAt:   t0,
		CurrentState: string(models.LifecyclePendingNotification),
	}

	tracker.RecordT1(lc, t1)
	tracker.RecordT2(lc, "dr-jones", t2)
	tracker.RecordT3(lc, "ADJUST_MEDICATION", "Reduced diuretic dose", t3)
	tracker.RecordT4(lc, "BP stabilized", t4)

	// Delivery latency: T1 - T0 = 5 min = 300000ms
	require.NotNil(t, lc.DeliveryLatencyMs)
	assert.Equal(t, int64(300000), *lc.DeliveryLatencyMs)

	// Acknowledgment latency: T2 - T1 = 25 min = 1500000ms
	require.NotNil(t, lc.AcknowledgmentLatencyMs)
	assert.Equal(t, int64(1500000), *lc.AcknowledgmentLatencyMs)

	// Action latency: T3 - T2 = 1h30m = 5400000ms
	require.NotNil(t, lc.ActionLatencyMs)
	assert.Equal(t, int64(5400000), *lc.ActionLatencyMs)

	// Outcome latency: T4 - T3 = 46h = 165600000ms
	require.NotNil(t, lc.OutcomeLatencyMs)
	assert.Equal(t, int64(165600000), *lc.OutcomeLatencyMs)

	// Total latency: T4 - T0 = 48h = 172800000ms
	require.NotNil(t, lc.TotalLatencyMs)
	assert.Equal(t, int64(172800000), *lc.TotalLatencyMs)

	assert.Equal(t, string(models.LifecycleResolved), lc.CurrentState)
}

func TestTracker_OutOfOrder_T2BeforeT1(t *testing.T) {
	tracker := newLifecycleTracker()

	t0 := time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(5 * time.Minute)
	t2 := t0.Add(30 * time.Minute)

	lc := &models.DetectionLifecycle{
		DetectedAt:   t0,
		CurrentState: string(models.LifecyclePendingNotification),
	}

	// Record T2 before T1 (out-of-order)
	tracker.RecordT2(lc, "dr-jones", t2)

	require.NotNil(t, lc.AcknowledgedAt)
	assert.Equal(t, t2, *lc.AcknowledgedAt)
	assert.Equal(t, "dr-jones", lc.AssignedClinicianID)
	// AcknowledgmentLatencyMs should be nil because T1 is not yet known
	assert.Nil(t, lc.AcknowledgmentLatencyMs)

	// Now record T1 — should backfill AcknowledgmentLatencyMs
	tracker.RecordT1(lc, t1)

	require.NotNil(t, lc.DeliveredAt)
	require.NotNil(t, lc.DeliveryLatencyMs)
	assert.Equal(t, int64(300000), *lc.DeliveryLatencyMs) // T1 - T0 = 5 min

	// AcknowledgmentLatencyMs should now be computed: T2 - T1 = 25 min
	require.NotNil(t, lc.AcknowledgmentLatencyMs)
	assert.Equal(t, int64(1500000), *lc.AcknowledgmentLatencyMs)
}
