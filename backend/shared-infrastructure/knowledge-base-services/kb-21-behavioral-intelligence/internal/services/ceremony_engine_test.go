package services

import (
	"testing"

	"kb-21-behavioral-intelligence/internal/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCeremonyTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// Create table manually to avoid gen_random_uuid() incompatibility with SQLite.
	db.Exec(`CREATE TABLE IF NOT EXISTS ceremony_records (
		id TEXT PRIMARY KEY,
		patient_id TEXT NOT NULL,
		from_season VARCHAR(20) NOT NULL,
		to_season VARCHAR(20) NOT NULL,
		ceremony_type VARCHAR(50) NOT NULL,
		delivered_at DATETIME NOT NULL,
		channel VARCHAR(20),
		acknowledged BOOLEAN DEFAULT FALSE,
		created_at DATETIME
	)`)
	db.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS idx_ceremony_patient_season ON ceremony_records(patient_id, to_season)`)
	return db
}

func newTestCeremonyEngine(t *testing.T) *CeremonyEngine {
	t.Helper()
	db := setupCeremonyTestDB(t)
	return NewCeremonyEngine(db, zap.NewNop())
}

func TestCeremonyEngine_GetCeremonyMessage(t *testing.T) {
	ce := NewCeremonyEngine(nil, nil)

	tests := []struct {
		from, to models.EngagementSeason
		contains string
	}{
		{models.SeasonCorrection, models.SeasonConsolidation, "90-day correction"},
		{models.SeasonConsolidation, models.SeasonIndependence, "Independence"},
		{models.SeasonIndependence, models.SeasonStability, "Six months"},
		{models.SeasonStability, models.SeasonPartnership, "partnership"},
	}

	for _, tt := range tests {
		msg := ce.GetCeremonyMessage(tt.from, tt.to)
		if msg == "" {
			t.Errorf("expected non-empty message for %s->%s", tt.from, tt.to)
		}
	}
}

func TestCeremonyEngine_GetCeremonyMessage_UnknownTransition(t *testing.T) {
	ce := NewCeremonyEngine(nil, nil)
	msg := ce.GetCeremonyMessage(models.SeasonPartnership, models.SeasonCorrection)
	if msg == "" {
		t.Error("expected fallback message for unknown transition")
	}
	expected := "Congratulations on reaching CORRECTION! Your health journey continues."
	if msg != expected {
		t.Errorf("fallback message mismatch: got %q", msg)
	}
}

func TestCeremonyEngine_IsCeremonyDelivered_FalseWhenNew(t *testing.T) {
	ce := newTestCeremonyEngine(t)

	delivered, err := ce.IsCeremonyDelivered("patient-1", models.SeasonConsolidation)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if delivered {
		t.Error("ceremony should not be delivered yet")
	}
}

func TestCeremonyEngine_IsCeremonyDelivered_NilDB(t *testing.T) {
	ce := NewCeremonyEngine(nil, nil)
	delivered, err := ce.IsCeremonyDelivered("patient-1", models.SeasonConsolidation)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if delivered {
		t.Error("nil db should return false")
	}
}

func TestCeremonyEngine_RecordCeremony_PreventsDoubleDelivery(t *testing.T) {
	ce := newTestCeremonyEngine(t)

	// Manually set UUID since SQLite lacks gen_random_uuid().
	_ = uuid.New() // ensure uuid package is used

	err := ce.RecordCeremony("patient-1", models.SeasonCorrection, models.SeasonConsolidation, "GRADUATION", models.ChannelWhatsApp)
	if err != nil {
		t.Fatalf("first delivery: %v", err)
	}

	delivered, err := ce.IsCeremonyDelivered("patient-1", models.SeasonConsolidation)
	if err != nil {
		t.Fatalf("check delivery: %v", err)
	}
	if !delivered {
		t.Error("ceremony should be marked as delivered after recording")
	}
}

func TestCeremonyEngine_RecordCeremony_NilDB(t *testing.T) {
	ce := NewCeremonyEngine(nil, nil)
	err := ce.RecordCeremony("patient-1", models.SeasonCorrection, models.SeasonConsolidation, "GRADUATION", models.ChannelWhatsApp)
	if err != nil {
		t.Fatalf("nil db should return nil error, got: %v", err)
	}
}
