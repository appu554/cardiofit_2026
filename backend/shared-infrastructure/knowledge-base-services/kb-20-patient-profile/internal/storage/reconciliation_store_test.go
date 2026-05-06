package storage

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/interfaces"
	"github.com/cardiofit/shared/v2_substrate/models"
)

// openTestReconciliationStore opens a DB-gated test store. Skipped when
// KB20_TEST_DATABASE_URL is not set so unit-only CI runs stay fast.
func openTestReconciliationStore(t *testing.T) (*ReconciliationStore, *DischargeDocumentStore, *V2SubstrateStore, *sql.DB) {
	t.Helper()
	dsn := os.Getenv("KB20_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KB20_TEST_DATABASE_URL not set; skipping DB-gated reconciliation store test")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
	v2 := NewV2SubstrateStoreWithDB(db)
	docs := NewDischargeDocumentStore(db)
	return NewReconciliationStore(db, v2, docs), docs, v2, db
}

func TestReconciliationStore_StartDecideFinalise(t *testing.T) {
	s, docs, v2, db := openTestReconciliationStore(t)
	defer db.Close()

	residentRef := uuid.New()
	roleRef := uuid.New()

	// Stage a pre-admission MedicineUse so the diff sees a ceased entry.
	pre := models.MedicineUse{
		ID:          uuid.New(),
		ResidentID:  residentRef,
		AMTCode:     "AMT-WAR-2",
		DisplayName: "warfarin",
		Dose:        "2mg",
		Frequency:   "QD",
		Route:       "oral",
		StartedAt:   time.Now().Add(-90 * 24 * time.Hour),
		Status:      models.MedicineUseStatusActive,
		Intent:      models.Intent{Category: models.IntentTherapeutic, Indication: "AF"},
	}
	if _, err := v2.UpsertMedicineUse(context.Background(), pre); err != nil {
		t.Fatalf("seed pre-admission medicine: %v", err)
	}

	// Persist a discharge document with two lines: apixaban (new acute) and a metformin missing.
	doc := interfaces.DischargeDocument{
		ResidentRef:             residentRef,
		Source:                  "manual",
		DocumentID:              "test-" + uuid.NewString(),
		DischargeDate:           time.Now(),
		DischargingFacilityName: "Test Hospital",
		MedicationLines: []interfaces.DischargeMedicationLine{
			{
				LineNumber:        1,
				MedicationNameRaw: "apixaban",
				AMTCode:           "AMT-APX-5",
				DoseRaw:           "5mg",
				FrequencyRaw:      "BID",
				RouteRaw:          "oral",
				IndicationText:    "atrial fibrillation post-op",
			},
		},
	}
	persistedDoc, err := docs.CreateDischargeDocument(context.Background(), doc)
	if err != nil {
		t.Fatalf("create discharge doc: %v", err)
	}

	// Start worklist.
	res, err := s.StartWorklist(context.Background(), interfaces.ReconciliationStartInputs{
		DischargeDocumentRef: persistedDoc.ID,
		AssignedRoleRef:      &roleRef,
	})
	if err != nil {
		t.Fatalf("start worklist: %v", err)
	}
	if len(res.Decisions) < 2 {
		t.Fatalf("expected >=2 decisions (ceased warfarin + new apixaban), got %d", len(res.Decisions))
	}

	// Decide each row.
	for _, d := range res.Decisions {
		decision := "accept"
		if _, err := s.DecideReconciliation(context.Background(), interfaces.DecideReconciliationInputs{
			WorklistRef:  res.Worklist.ID,
			DecisionRef:  d.ID,
			ACOPDecision: decision,
			ACOPRoleRef:  roleRef,
		}); err != nil {
			t.Fatalf("decide: %v", err)
		}
	}

	// Finalise.
	final, err := s.FinaliseWorklist(context.Background(), res.Worklist.ID, roleRef)
	if err != nil {
		t.Fatalf("finalise: %v", err)
	}
	if final.Worklist.Status != "completed" {
		t.Fatalf("expected completed status, got %s", final.Worklist.Status)
	}
	if len(final.ResultingMedicineUseRefs) == 0 {
		t.Fatalf("expected resulting MedicineUse refs from accept decisions")
	}
	if final.CompletionEventID == nil {
		t.Fatalf("expected reconciliation_completed event id")
	}
}
