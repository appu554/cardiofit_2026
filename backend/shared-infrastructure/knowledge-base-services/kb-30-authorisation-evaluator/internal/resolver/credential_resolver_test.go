package resolver

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"kb-authorisation-evaluator/internal/dsl"
	"kb-authorisation-evaluator/internal/evaluator"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("KB30_TEST_DATABASE_URL")
	if dsn == "" {
		dsn = os.Getenv("KB30_DATABASE_URL")
	}
	if dsn == "" {
		t.Skip("no KB30_TEST_DATABASE_URL or KB30_DATABASE_URL set; skipping DB-backed resolver tests")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
	return db
}

func cleanupCredentials(t *testing.T, db *sql.DB, person uuid.UUID) {
	t.Helper()
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM credentials WHERE person_id = $1", person)
	})
}

func cleanupAgreements(t *testing.T, db *sql.DB, prescriber uuid.UUID) {
	t.Helper()
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM prescribing_agreements WHERE prescriber_id = $1", prescriber)
	})
}

// --- Credential checks ---

func TestCredentialResolver_ActiveAPCTrainingPasses(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	r := NewCredentialResolver(db)
	ctx := context.Background()
	person := uuid.New()
	cleanupCredentials(t, db, person)

	_, err := db.ExecContext(ctx, `
INSERT INTO credentials (id, person_id, type, identifier, valid_from, valid_to)
VALUES ($1, $2, 'apc_training', 'APC-001', $3, $4)`,
		uuid.New(), person,
		time.Now().Add(-30*24*time.Hour), time.Now().Add(365*24*time.Hour))
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	res, err := r.Resolve(ctx,
		evaluator.Query{ActorRef: person, Role: "acop_pharmacist", ActionDate: time.Now()},
		dsl.Condition{Check: "Credential.kind='apc_training' AND Credential.valid_at_action_time"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if !res.Passed {
		t.Errorf("expected Passed=true; got %+v", res)
	}
}

func TestCredentialResolver_ExpiredAPCTrainingFails(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	r := NewCredentialResolver(db)
	ctx := context.Background()
	person := uuid.New()
	cleanupCredentials(t, db, person)

	_, err := db.ExecContext(ctx, `
INSERT INTO credentials (id, person_id, type, identifier, valid_from, valid_to)
VALUES ($1, $2, 'apc_training', 'APC-EXPIRED', $3, $4)`,
		uuid.New(), person,
		time.Now().Add(-365*24*time.Hour), time.Now().Add(-1*24*time.Hour))
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	res, err := r.Resolve(ctx,
		evaluator.Query{ActorRef: person, ActionDate: time.Now()},
		dsl.Condition{Check: "Credential.kind='apc_training' AND Credential.valid_at_action_time"})
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if res.Passed {
		t.Errorf("expected Passed=false for expired credential; got %+v", res)
	}
}

func TestCredentialResolver_RevokedCredentialFails(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	r := NewCredentialResolver(db)
	ctx := context.Background()
	person := uuid.New()
	cleanupCredentials(t, db, person)

	_, err := db.ExecContext(ctx, `
INSERT INTO credentials (id, person_id, type, identifier, valid_from, valid_to, revoked_at, revocation_reason)
VALUES ($1, $2, 'apc_training', 'APC-REV', $3, $4, NOW(), 'misconduct')`,
		uuid.New(), person,
		time.Now().Add(-30*24*time.Hour), time.Now().Add(365*24*time.Hour))
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	res, _ := r.Resolve(ctx,
		evaluator.Query{ActorRef: person, ActionDate: time.Now()},
		dsl.Condition{Check: "Credential.kind='apc_training' AND Credential.valid_at_action_time"})
	if res.Passed {
		t.Errorf("expected Passed=false for revoked credential; got %+v", res)
	}
}

func TestCredentialResolver_NoCredentialFails(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	r := NewCredentialResolver(db)
	res, _ := r.Resolve(context.Background(),
		evaluator.Query{ActorRef: uuid.New(), ActionDate: time.Now()},
		dsl.Condition{Check: "Credential.kind='apc_training' AND Credential.valid_at_action_time"})
	if res.Passed {
		t.Errorf("no credential seeded; expected Passed=false; got %+v", res)
	}
}

func TestCredentialResolver_AHPRARegistration(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	r := NewCredentialResolver(db)
	ctx := context.Background()
	person := uuid.New()
	cleanupCredentials(t, db, person)

	_, err := db.ExecContext(ctx, `
INSERT INTO credentials (id, person_id, type, identifier, valid_from, valid_to)
VALUES ($1, $2, 'ahpra_pharmacist_registration', 'PHA0001', $3, $4)`,
		uuid.New(), person,
		time.Now().Add(-30*24*time.Hour), time.Now().Add(365*24*time.Hour))
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	res, _ := r.Resolve(ctx,
		evaluator.Query{ActorRef: person, ActionDate: time.Now()},
		dsl.Condition{Check: "Credential.kind='ahpra_pharmacist_registration' AND Credential.valid_at_action_time"})
	if !res.Passed {
		t.Errorf("expected Passed=true for active AHPRA registration; got %+v", res)
	}
}

func TestCredentialResolver_DRNPEndorsementValid(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	r := NewCredentialResolver(db)
	ctx := context.Background()
	person := uuid.New()
	cleanupCredentials(t, db, person)

	_, err := db.ExecContext(ctx, `
INSERT INTO credentials (id, person_id, type, identifier, valid_from, valid_to)
VALUES ($1, $2, 'NMBA_DRNP_endorsement', 'DRNP-001', $3, $4)`,
		uuid.New(), person,
		time.Now().Add(-30*24*time.Hour), time.Now().Add(365*24*time.Hour))
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	res, _ := r.Resolve(ctx,
		evaluator.Query{ActorRef: person, ActionDate: time.Now()},
		dsl.Condition{Check: "Credential.endorsement_valid_at_action_time"})
	if !res.Passed {
		t.Errorf("expected Passed=true for active DRNP endorsement; got %+v", res)
	}
}

// --- Prescribing-agreement checks ---

func TestCredentialResolver_ActiveAgreementPasses(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	r := NewCredentialResolver(db)
	ctx := context.Background()
	prescriber := uuid.New()
	resident := uuid.New()
	cleanupAgreements(t, db, prescriber)

	_, err := db.ExecContext(ctx, `
INSERT INTO prescribing_agreements (id, prescriber_id, authoriser_id,
    medication_classes, resident_scope, valid_from, valid_to,
    mentorship_status)
VALUES ($1, $2, $3, $4, 'all', $5, $6, 'complete')`,
		uuid.New(), prescriber, uuid.New(),
		"{antihypertensives}",
		time.Now().Add(-30*24*time.Hour),
		time.Now().Add(365*24*time.Hour))
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	res, _ := r.Resolve(ctx,
		evaluator.Query{
			ActorRef:        prescriber,
			ResidentRef:     resident,
			MedicationClass: "antihypertensives",
			ActionDate:      time.Now(),
		},
		dsl.Condition{Check: "PrescribingAgreement.exists_for_person_AND_resident_AND_medication_class"})
	if !res.Passed {
		t.Errorf("expected Passed=true for active agreement; got %+v", res)
	}
}

func TestCredentialResolver_AgreementScopeMissesClassFails(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	r := NewCredentialResolver(db)
	ctx := context.Background()
	prescriber := uuid.New()
	cleanupAgreements(t, db, prescriber)

	_, err := db.ExecContext(ctx, `
INSERT INTO prescribing_agreements (id, prescriber_id, authoriser_id,
    medication_classes, resident_scope, valid_from, valid_to,
    mentorship_status)
VALUES ($1, $2, $3, $4, 'all', $5, $6, 'complete')`,
		uuid.New(), prescriber, uuid.New(),
		"{antihypertensives}",
		time.Now().Add(-30*24*time.Hour),
		time.Now().Add(365*24*time.Hour))
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	res, _ := r.Resolve(ctx,
		evaluator.Query{
			ActorRef:        prescriber,
			ResidentRef:     uuid.New(),
			MedicationClass: "opioid_analgesics", // outside scope
			ActionDate:      time.Now(),
		},
		dsl.Condition{Check: "PrescribingAgreement.scope_includes(medication_class)"})
	if res.Passed {
		t.Errorf("expected Passed=false for class outside scope; got %+v", res)
	}
}

func TestCredentialResolver_AgreementScopeIncludesClassPasses(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	r := NewCredentialResolver(db)
	ctx := context.Background()
	prescriber := uuid.New()
	cleanupAgreements(t, db, prescriber)

	_, err := db.ExecContext(ctx, `
INSERT INTO prescribing_agreements (id, prescriber_id, authoriser_id,
    medication_classes, resident_scope, valid_from, valid_to,
    mentorship_status)
VALUES ($1, $2, $3, $4, 'all', $5, $6, 'complete')`,
		uuid.New(), prescriber, uuid.New(),
		"{antihypertensives,diabetics}",
		time.Now().Add(-30*24*time.Hour),
		time.Now().Add(365*24*time.Hour))
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	res, _ := r.Resolve(ctx,
		evaluator.Query{
			ActorRef:        prescriber,
			MedicationClass: "diabetics",
			ActionDate:      time.Now(),
		},
		dsl.Condition{Check: "PrescribingAgreement.scope_includes(medication_class)"})
	if !res.Passed {
		t.Errorf("expected Passed=true for class within scope; got %+v", res)
	}
}

func TestCredentialResolver_NamedResidentScopeBlocksOthers(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	r := NewCredentialResolver(db)
	ctx := context.Background()
	prescriber := uuid.New()
	namedResident := uuid.New()
	otherResident := uuid.New()
	cleanupAgreements(t, db, prescriber)

	_, err := db.ExecContext(ctx, `
INSERT INTO prescribing_agreements (id, prescriber_id, authoriser_id,
    medication_classes, resident_scope, named_residents, valid_from, valid_to,
    mentorship_status)
VALUES ($1, $2, $3, $4, 'named', $5, $6, $7, 'complete')`,
		uuid.New(), prescriber, uuid.New(),
		"{antihypertensives}",
		"{"+namedResident.String()+"}",
		time.Now().Add(-30*24*time.Hour),
		time.Now().Add(365*24*time.Hour))
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Check: agreement should NOT cover an unnamed resident.
	res, _ := r.Resolve(ctx,
		evaluator.Query{
			ActorRef:        prescriber,
			ResidentRef:     otherResident,
			MedicationClass: "antihypertensives",
			ActionDate:      time.Now(),
		},
		dsl.Condition{Check: "PrescribingAgreement.exists_for_person_AND_resident_AND_medication_class"})
	if res.Passed {
		t.Errorf("expected Passed=false for unnamed resident; got %+v", res)
	}

	// Check: agreement should cover the named resident.
	res2, _ := r.Resolve(ctx,
		evaluator.Query{
			ActorRef:        prescriber,
			ResidentRef:     namedResident,
			MedicationClass: "antihypertensives",
			ActionDate:      time.Now(),
		},
		dsl.Condition{Check: "PrescribingAgreement.exists_for_person_AND_resident_AND_medication_class"})
	if !res2.Passed {
		t.Errorf("expected Passed=true for named resident; got %+v", res2)
	}
}

func TestCredentialResolver_BreachedMentorshipFails(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	r := NewCredentialResolver(db)
	ctx := context.Background()
	prescriber := uuid.New()
	cleanupAgreements(t, db, prescriber)

	_, err := db.ExecContext(ctx, `
INSERT INTO prescribing_agreements (id, prescriber_id, authoriser_id,
    medication_classes, resident_scope, valid_from, valid_to,
    mentorship_status)
VALUES ($1, $2, $3, $4, 'all', $5, $6, 'breached')`,
		uuid.New(), prescriber, uuid.New(),
		"{antihypertensives}",
		time.Now().Add(-30*24*time.Hour),
		time.Now().Add(365*24*time.Hour))
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	res, _ := r.Resolve(ctx,
		evaluator.Query{ActorRef: prescriber},
		dsl.Condition{Check: "MentorshipStatus IN ['active', 'complete']"})
	if res.Passed {
		t.Errorf("expected Passed=false for breached mentorship; got %+v", res)
	}
}

func TestCredentialResolver_CompleteMentorshipPasses(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	r := NewCredentialResolver(db)
	ctx := context.Background()
	prescriber := uuid.New()
	cleanupAgreements(t, db, prescriber)

	_, err := db.ExecContext(ctx, `
INSERT INTO prescribing_agreements (id, prescriber_id, authoriser_id,
    medication_classes, resident_scope, valid_from, valid_to,
    mentorship_status, mentorship_completed_at)
VALUES ($1, $2, $3, $4, 'all', $5, $6, 'complete', $7)`,
		uuid.New(), prescriber, uuid.New(),
		"{antihypertensives}",
		time.Now().Add(-30*24*time.Hour),
		time.Now().Add(365*24*time.Hour),
		time.Now().Add(-7*24*time.Hour))
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	res, _ := r.Resolve(ctx,
		evaluator.Query{ActorRef: prescriber},
		dsl.Condition{Check: "MentorshipStatus IN ['active', 'complete']"})
	if !res.Passed {
		t.Errorf("expected Passed=true for complete mentorship; got %+v", res)
	}
}

// --- Safety: unknown checks must deny, not pass ---

func TestCredentialResolver_UnknownCheckSafelyDenies(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	r := NewCredentialResolver(db)
	res, err := r.Resolve(context.Background(),
		evaluator.Query{ActorRef: uuid.New()},
		dsl.Condition{Check: "completely.unknown.check"})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if res.Passed {
		t.Errorf("unknown check must safely deny; got Passed=true")
	}
	if res.Detail == "" {
		t.Errorf("unknown check must populate Detail explaining the deny")
	}
}
