package services

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"kb-26-metabolic-digital-twin/internal/clients"
	"kb-26-metabolic-digital-twin/internal/models"
)

// trackingClassifier wraps a real classifier (orchestrator) but counts
// how many patients are processed and lets tests force errors on specific IDs.
type trackingClassifier struct {
	inner    BPContextClassifier
	count    atomic.Int32
	errOn    map[string]bool
	errValue error
}

func (t *trackingClassifier) Classify(ctx context.Context, patientID string) (*models.BPContextClassification, error) {
	t.count.Add(1)
	if t.errOn[patientID] {
		return nil, t.errValue
	}
	return t.inner.Classify(ctx, patientID)
}

func setupBatchJobTest(t *testing.T, kb20Profile *clients.KB20PatientProfile) (*BPContextDailyBatch, *BPContextRepository, *trackingClassifier) {
	t.Helper()

	db := setupBPContextTestDB(t)
	// Create the SQLite-compatible twin_states table using Task 3's exported helper.
	setupTwinStateTable(t, db)

	repo := NewBPContextRepository(db)
	kb20 := &stubKB20Client{profile: kb20Profile}
	kb21 := &stubKB21Client{}
	thresholds := defaultBPContextThresholds()
	inner := NewBPContextOrchestrator(kb20, kb21, repo, thresholds, zap.NewNop(), nil, nil)

	tracker := &trackingClassifier{inner: inner}
	job := NewBPContextDailyBatch(repo, tracker, 30*24*time.Hour, 4, zap.NewNop(), nil)
	return job, repo, tracker
}

// seedTwinStateRow inserts one twin_states row using the SQLite-compatible
// table created by setupTwinStateTable (Task 3). Column types match the DDL
// in bp_context_repository_test.go exactly.
func seedTwinStateRow(t *testing.T, repo *BPContextRepository, patientID string, daysAgo int) {
	t.Helper()
	type twinRow struct {
		ID           string    `gorm:"column:id;primaryKey"`
		PatientID    string    `gorm:"column:patient_id"`
		StateVersion int       `gorm:"column:state_version"`
		UpdateSource string    `gorm:"column:update_source"`
		UpdatedAt    time.Time `gorm:"column:updated_at"`
	}
	row := twinRow{
		ID:           patientID + "-row",
		PatientID:    patientID,
		StateVersion: 1,
		UpdateSource: "TEST",
		UpdatedAt:    time.Now().UTC().AddDate(0, 0, -daysAgo),
	}
	if err := repo.DB().Table("twin_states").Create(&row).Error; err != nil {
		t.Fatalf("seed twin_states row for %s: %v", patientID, err)
	}
}

func TestBPContextDailyBatch_ProcessesAllActivePatients(t *testing.T) {
	job, repo, tracker := setupBatchJobTest(t, &clients.KB20PatientProfile{
		PatientID:        "shared",
		SBP14dMean:       ptrFloat(120),
		DBP14dMean:       ptrFloat(75),
		ClinicSBPMean:    ptrFloat(118),
		ClinicDBPMean:    ptrFloat(74),
		ClinicReadings:   2,
		HomeReadings:     14,
		HomeDaysWithData: 7,
	})

	// Seed 5 active patients with distinct IDs.
	for i := 0; i < 5; i++ {
		seedTwinStateRow(t, repo, "patient-"+string(rune('a'+i)), i)
	}

	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := tracker.count.Load(); got != 5 {
		t.Errorf("expected 5 patients processed, got %d", got)
	}
}

func TestBPContextDailyBatch_OneClassificationErrors_OthersStillRun(t *testing.T) {
	job, repo, tracker := setupBatchJobTest(t, &clients.KB20PatientProfile{
		PatientID:        "shared",
		SBP14dMean:       ptrFloat(120),
		DBP14dMean:       ptrFloat(75),
		ClinicSBPMean:    ptrFloat(118),
		ClinicDBPMean:    ptrFloat(74),
		ClinicReadings:   2,
		HomeReadings:     14,
		HomeDaysWithData: 7,
	})

	patientIDs := []string{"patient-a", "patient-b", "patient-c"}
	for i, pid := range patientIDs {
		seedTwinStateRow(t, repo, pid, i)
	}

	// Make patient-b's classification fail.
	tracker.errOn = map[string]bool{"patient-b": true}
	tracker.errValue = errSimulated()

	if err := job.Run(context.Background()); err != nil {
		t.Fatalf("Run should not fail when individual patients error: %v", err)
	}
	if got := tracker.count.Load(); got != 3 {
		t.Errorf("all 3 patients should be attempted, got %d", got)
	}
}

func TestBPContextDailyBatch_RespectsContextCancel(t *testing.T) {
	job, repo, tracker := setupBatchJobTest(t, &clients.KB20PatientProfile{
		PatientID:        "shared",
		SBP14dMean:       ptrFloat(120),
		DBP14dMean:       ptrFloat(75),
		ClinicSBPMean:    ptrFloat(118),
		ClinicDBPMean:    ptrFloat(74),
		ClinicReadings:   2,
		HomeReadings:     14,
		HomeDaysWithData: 7,
	})

	// Seed 100 active patients so the batch takes measurable time.
	for i := 0; i < 100; i++ {
		pid := "patient-" + string(rune('a'+i%26)) + "-" + string(rune('a'+i/26))
		seedTwinStateRow(t, repo, pid, i%30)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before run starts

	err := job.Run(ctx)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	// At most a few patients may have been picked up before cancel propagated;
	// the strict expectation is "much less than 100".
	if tracker.count.Load() >= 100 {
		t.Errorf("batch should have aborted on cancel, processed %d/100", tracker.count.Load())
	}
}

// Ensure trackingClassifier satisfies BPContextClassifier at compile time.
var _ BPContextClassifier = (*trackingClassifier)(nil)

// Silence the "gorm imported but not used" error — gorm is needed for the
// seedTwinStateRow anonymous struct tags even though we only call via repo.DB().
var _ *gorm.DB
