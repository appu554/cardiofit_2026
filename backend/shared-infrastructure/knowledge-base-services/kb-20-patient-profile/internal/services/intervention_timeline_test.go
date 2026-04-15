package services

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupInterventionTimelineTestDB creates a minimal in-memory sqlite DB
// with the medication_states schema needed by BuildTimeline.
func setupInterventionTimelineTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	err = db.Exec(`
		CREATE TABLE medication_states (
			id TEXT PRIMARY KEY,
			patient_id TEXT NOT NULL,
			drug_name TEXT NOT NULL,
			drug_class TEXT NOT NULL,
			dose_mg TEXT,
			fhir_medication_request_id TEXT,
			atc_code TEXT,
			is_active INTEGER DEFAULT 1,
			start_date DATETIME,
			route TEXT,
			updated_at DATETIME,
			created_at DATETIME
		)
	`).Error
	if err != nil {
		t.Fatalf("create medication_states: %v", err)
	}
	return db
}

func seedMedState(t *testing.T, db *gorm.DB, patientID, drugClass, drugName string, dose float64, updatedAt time.Time) {
	t.Helper()
	row := map[string]interface{}{
		"id":         uuid.New().String(),
		"patient_id": patientID,
		"drug_name":  drugName,
		"drug_class": drugClass,
		"dose_mg":    decimal.NewFromFloat(dose).String(),
		"is_active":  1,
		"start_date": updatedAt,
		"updated_at": updatedAt,
		"created_at": updatedAt,
		"route":      "ORAL",
	}
	if err := db.Table("medication_states").Create(row).Error; err != nil {
		t.Fatalf("seed medication_state: %v", err)
	}
}

// TestInterventionTimeline_LatestPerDomain verifies the core contract:
// multiple medication_state rows across 3 domains are collapsed to the
// latest action per domain. Maps to P7-D.1 step 2 in the plan.
func TestInterventionTimeline_LatestPerDomain(t *testing.T) {
	db := setupInterventionTimelineTestDB(t)
	patientID := "p-inertia"

	now := time.Now().UTC()
	// GLYCAEMIC: two rows, latest should win
	seedMedState(t, db, patientID, "METFORMIN", "metformin", 500, now.AddDate(0, 0, -60))
	seedMedState(t, db, patientID, "METFORMIN", "metformin", 1000, now.AddDate(0, 0, -10))
	// HEMODYNAMIC: single row
	seedMedState(t, db, patientID, "ACEi", "lisinopril", 10, now.AddDate(0, 0, -30))
	// RENAL: single row
	seedMedState(t, db, patientID, "FINERENONE", "finerenone", 20, now.AddDate(0, 0, -20))
	// LIPID: single row
	seedMedState(t, db, patientID, "STATIN", "atorvastatin", 40, now.AddDate(0, 0, -45))

	svc := NewInterventionTimelineService(db, zap.NewNop())
	result, err := svc.BuildTimeline(patientID)
	if err != nil {
		t.Fatalf("BuildTimeline: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// All four domains should be present.
	for _, domain := range []string{"GLYCAEMIC", "HEMODYNAMIC", "RENAL", "LIPID"} {
		if _, ok := result.ByDomain[domain]; !ok {
			t.Errorf("expected domain %s in timeline, got %+v", domain, result.ByDomain)
		}
	}

	// Glycaemic latest should be the 1000 mg dose (10 days old, not 60).
	glyc := result.ByDomain["GLYCAEMIC"]
	if glyc.DoseMg != 1000 {
		t.Errorf("GLYCAEMIC latest dose = %.0f, want 1000 (most recent)", glyc.DoseMg)
	}
	if glyc.DaysSince > 11 {
		t.Errorf("GLYCAEMIC days_since = %d, want ~10", glyc.DaysSince)
	}

	// AnyChangeInLast12Weeks should be true (multiple actions within 84 days).
	if !result.AnyChangeInLast12Weeks {
		t.Error("AnyChangeInLast12Weeks expected true with fresh medication changes")
	}
}

// TestInterventionTimeline_EmptyWhenNoRows confirms a patient with zero
// medication_state rows returns an empty, non-nil result.
func TestInterventionTimeline_EmptyWhenNoRows(t *testing.T) {
	db := setupInterventionTimelineTestDB(t)
	svc := NewInterventionTimelineService(db, zap.NewNop())

	result, err := svc.BuildTimeline("p-none")
	if err != nil {
		t.Fatalf("BuildTimeline: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result even for empty patient")
	}
	if len(result.ByDomain) != 0 {
		t.Errorf("expected empty ByDomain, got %+v", result.ByDomain)
	}
	if result.AnyChangeInLast12Weeks {
		t.Error("expected AnyChangeInLast12Weeks=false for empty patient")
	}
}

// TestInterventionTimeline_SkipsOldRows confirms rows older than 90 days
// are excluded from the scan.
func TestInterventionTimeline_SkipsOldRows(t *testing.T) {
	db := setupInterventionTimelineTestDB(t)
	patientID := "p-stale"

	seedMedState(t, db, patientID, "METFORMIN", "metformin", 500,
		time.Now().UTC().AddDate(0, 0, -120)) // 120 days old — excluded

	svc := NewInterventionTimelineService(db, zap.NewNop())
	result, err := svc.BuildTimeline(patientID)
	if err != nil {
		t.Fatalf("BuildTimeline: %v", err)
	}
	if len(result.ByDomain) != 0 {
		t.Errorf("expected no domains for all-stale patient, got %+v", result.ByDomain)
	}
}

func TestMapDrugClassToDomain(t *testing.T) {
	tests := []struct {
		drugClass string
		want      string
	}{
		// Glycaemic domain (9 classes)
		{"METFORMIN", "GLYCAEMIC"},
		{"SULFONYLUREA", "GLYCAEMIC"},
		{"DPP4i", "GLYCAEMIC"},
		{"SGLT2i", "GLYCAEMIC"},
		{"GLP1_RA", "GLYCAEMIC"},
		{"INSULIN", "GLYCAEMIC"},
		{"BASAL_INSULIN", "GLYCAEMIC"},
		{"PIOGLITAZONE", "GLYCAEMIC"},
		{"EXENATIDE", "GLYCAEMIC"},

		// Hemodynamic domain
		{"ACEi", "HEMODYNAMIC"},
		{"ARB", "HEMODYNAMIC"},
		{"BETA_BLOCKER", "HEMODYNAMIC"},

		// Lipid domain
		{"STATIN", "LIPID"},
		{"EZETIMIBE", "LIPID"},

		// Renal domain
		{"FINERENONE", "RENAL"},

		// Unknown → OTHER
		{"ASPIRIN", "OTHER"},
		{"", "OTHER"},
	}

	for _, tt := range tests {
		t.Run(tt.drugClass, func(t *testing.T) {
			got := MapDrugClassToDomain(tt.drugClass)
			if got != tt.want {
				t.Errorf("MapDrugClassToDomain(%q) = %q, want %q", tt.drugClass, got, tt.want)
			}
		})
	}
}

func TestMapDrugClassToAllDomains(t *testing.T) {
	tests := []struct {
		drugClass   string
		wantDomains []string
	}{
		// SGLT2i: glycaemic (primary) + renal + hemodynamic (secondary)
		{"SGLT2i", []string{"GLYCAEMIC", "RENAL", "HEMODYNAMIC"}},
		// GLP1_RA: glycaemic + hemodynamic
		{"GLP1_RA", []string{"GLYCAEMIC", "HEMODYNAMIC"}},
		// ACEi: hemodynamic + renal
		{"ACEi", []string{"HEMODYNAMIC", "RENAL"}},
		// ARB: hemodynamic + renal
		{"ARB", []string{"HEMODYNAMIC", "RENAL"}},
		// FINERENONE: renal + hemodynamic
		{"FINERENONE", []string{"RENAL", "HEMODYNAMIC"}},
		// Plain glycaemic drug — no secondary domains
		{"METFORMIN", []string{"GLYCAEMIC"}},
		// Plain hemodynamic drug — no secondary
		{"AMLODIPINE", []string{"HEMODYNAMIC"}},
		// Unknown
		{"ASPIRIN", nil},
	}

	for _, tt := range tests {
		t.Run(tt.drugClass, func(t *testing.T) {
			got := MapDrugClassToAllDomains(tt.drugClass)
			if tt.wantDomains == nil {
				if got != nil {
					t.Errorf("MapDrugClassToAllDomains(%q) = %v, want nil", tt.drugClass, got)
				}
				return
			}
			if len(got) != len(tt.wantDomains) {
				t.Fatalf("MapDrugClassToAllDomains(%q) returned %d domains, want %d: got %v",
					tt.drugClass, len(got), len(tt.wantDomains), got)
			}
			for i, want := range tt.wantDomains {
				if got[i] != want {
					t.Errorf("MapDrugClassToAllDomains(%q)[%d] = %q, want %q", tt.drugClass, i, got[i], want)
				}
			}
		})
	}
}
