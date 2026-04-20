package services

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"kb-26-metabolic-digital-twin/internal/models"
)

func setupInterventionTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqlDB, _ := db.DB()
	stmt := `CREATE TABLE intervention_definitions (
		id                      TEXT NOT NULL,
		cohort_id               TEXT NOT NULL,
		category                TEXT NOT NULL,
		name                    TEXT NOT NULL,
		clinician_language      TEXT,
		cool_down_hours         INTEGER NOT NULL DEFAULT 0,
		resource_cost           REAL NOT NULL DEFAULT 0,
		feature_signature       TEXT,
		eligibility_json        TEXT,
		contraindications_json  TEXT,
		version                 TEXT NOT NULL DEFAULT '1.0.0',
		source_yaml_path        TEXT,
		loaded_at               DATETIME,
		ledger_entry_id         TEXT,
		PRIMARY KEY (id, cohort_id)
	)`
	if _, err := sqlDB.Exec(stmt); err != nil {
		t.Fatalf("DDL: %v", err)
	}
	return db
}

// Features used across tests — kept as a helper for readability.
func baseHCFFeatures() map[string]float64 {
	return map[string]float64{
		"days_since_discharge":      3,
		"polypharmacy_score":        6,
		"phone_contact_opt_out":     0,
		"prior_cardiology_30d":      0,
		"device_eligible_flag":      1,
		"fluid_overload_flag":       0,
		"cognitive_impairment_flag": 0,
		"palliative_care_flag":      0,
	}
}

func TestInterventionRegistry_LoadFromYAML_PersistsDefinitions(t *testing.T) {
	db := setupInterventionTestDB(t)
	reg := NewInterventionRegistry(db)
	if err := reg.LoadFromYAML("../../../../market-configs/shared/intervention_taxonomy_hcf_chf.yaml"); err != nil {
		t.Fatalf("load: %v", err)
	}
	var count int64
	db.Model(&models.InterventionDefinition{}).Where("cohort_id = ?", "hcf_catalyst_chf").Count(&count)
	if count != 6 {
		t.Fatalf("want 6 HCF CHF interventions, got %d", count)
	}
}

func TestInterventionRegistry_ListEligible_FiltersByCohort(t *testing.T) {
	db := setupInterventionTestDB(t)
	reg := NewInterventionRegistry(db)
	if err := reg.LoadFromYAML("../../../../market-configs/shared/intervention_taxonomy_hcf_chf.yaml"); err != nil {
		t.Fatalf("load hcf: %v", err)
	}
	if err := reg.LoadFromYAML("../../../../market-configs/shared/intervention_taxonomy_aged_care_au.yaml"); err != nil {
		t.Fatalf("load aged: %v", err)
	}

	got, err := reg.ListEligible("hcf_catalyst_chf", baseHCFFeatures())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, d := range got {
		if d.CohortID != "hcf_catalyst_chf" {
			t.Fatalf("leaked cross-cohort: %s", d.CohortID)
		}
	}
	if len(got) == 0 {
		t.Fatal("expected at least one eligible intervention for this patient")
	}
}

func TestInterventionRegistry_ListEligible_ExcludesContraindicated(t *testing.T) {
	db := setupInterventionTestDB(t)
	reg := NewInterventionRegistry(db)
	if err := reg.LoadFromYAML("../../../../market-configs/shared/intervention_taxonomy_hcf_chf.yaml"); err != nil {
		t.Fatalf("load: %v", err)
	}

	features := baseHCFFeatures()
	features["phone_contact_opt_out"] = 1 // triggers nurse_phone_48h contraindication
	got, err := reg.ListEligible("hcf_catalyst_chf", features)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, d := range got {
		if d.ID == "nurse_phone_48h" {
			t.Fatal("phone-opt-out patient was offered nurse_phone_48h")
		}
	}
}

func TestInterventionRegistry_ListEligible_EnforcesEligibilityPredicate(t *testing.T) {
	db := setupInterventionTestDB(t)
	reg := NewInterventionRegistry(db)
	if err := reg.LoadFromYAML("../../../../market-configs/shared/intervention_taxonomy_hcf_chf.yaml"); err != nil {
		t.Fatalf("load: %v", err)
	}

	features := baseHCFFeatures()
	features["polypharmacy_score"] = 2 // below the gte:5 eligibility for pharmacist review
	got, err := reg.ListEligible("hcf_catalyst_chf", features)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	for _, d := range got {
		if d.ID == "pharmacist_medication_review" {
			t.Fatal("low-polypharmacy patient offered pharmacist review")
		}
	}
}
