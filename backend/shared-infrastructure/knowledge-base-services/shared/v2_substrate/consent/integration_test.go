package consent

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
	"github.com/cardiofit/shared/v2_substrate/recommendation"
)

// TestIntegration_PsychotropicRecBlockedWithoutConsent is the executable
// definition of the v2 §3 line 140 consent gate: drafted → submitted is
// rejected with ErrConsentRequired when ConsentRequired=true and no
// matching active consent exists.
//
// Two scenarios in one test:
//
//  1. With NO active consent: recommendation.Lifecycle.Transition(submitted)
//     returns recommendation.ErrConsentRequired; state stays drafted.
//  2. After seeding an active psychotropic consent: same transition
//     succeeds; one EvidenceTraceNode emitted by recommendation.EvidenceTraceAdapter.
func TestIntegration_PsychotropicRecBlockedWithoutConsent(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	ctx := context.Background()

	// Real consent infrastructure
	consentStore := NewPostgresStore(db)
	mapper := func(rt string) (string, bool) {
		// Production wires a richer mapping table; this small mapper covers
		// the test recommendation type.
		if rt == "stop" { // any "stop" rec is treated as psychotropic-gated for this test
			return models.ConsentClassPsychotropic, true
		}
		return "", false
	}
	checker := NewPostgresConsentChecker(consentStore, mapper)

	// Real recommendation infrastructure with the real consent checker
	recStore := recommendation.NewPostgresStore(db)
	nodeWriter := newCapturingNodeWriter()
	edges := recommendation.NewEvidenceTraceAdapter(nodeWriter)
	recLC := recommendation.NewLifecycle(recStore, edges, checker)

	// Seed: a drafted psychotropic recommendation with ConsentRequired=true
	resident := uuid.New()
	rec := models.Recommendation{
		ID:              uuid.New(),
		ResidentID:      resident,
		AuthorID:        uuid.New(),
		State:           models.RecommendationStateDrafted,
		Type:            "stop",
		Urgency:         models.RecommendationUrgencyAmber,
		Title:           "Cease risperidone",
		ConsentRequired: true,
		ClinicalContent: models.ClinicalContent{
			Issue:        "BPSD reassessment",
			Rationale:    "12-week trial completed",
			ProposedPlan: "cease",
		},
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := recStore.Create(ctx, &rec); err != nil {
		t.Fatalf("create rec: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM recommendations WHERE id = $1", rec.ID)
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM consents WHERE resident_id = $1", resident)
	})

	// SCENARIO 1: no active consent → ErrConsentRequired
	err := recLC.Transition(ctx, recommendation.TransitionRequest{
		RecommendationID: rec.ID,
		ToState:          models.RecommendationStateSubmitted,
		ActorID:          uuid.New(),
		ActorClass:       recommendation.ActorClassHuman,
	})
	if !errors.Is(err, recommendation.ErrConsentRequired) {
		t.Fatalf("expected ErrConsentRequired; got %v", err)
	}

	// State must NOT have advanced.
	got, err := recStore.Get(ctx, rec.ID)
	if err != nil {
		t.Fatalf("get rec: %v", err)
	}
	if got.State != models.RecommendationStateDrafted {
		t.Errorf("rec state = %q, want drafted (consent gate must not advance state)", got.State)
	}
	if got.SubmittedAt != nil {
		t.Errorf("submitted_at must be nil after rejected transition; got %v", *got.SubmittedAt)
	}

	// No evidence nodes should have been emitted.
	if n := nodeWriter.count(); n != 0 {
		t.Errorf("rejected transition must not emit evidence nodes; got %d", n)
	}

	// SCENARIO 2: seed an active psychotropic consent and retry
	consent := models.Consent{
		ID:            uuid.New(),
		ResidentID:    resident,
		Class:         models.ConsentClassPsychotropic,
		State:         models.ConsentStateActive,
		GrantedByID:   uuid.New(),
		GrantedByRole: "substitute_decision_maker",
		ValidFrom:     time.Now().UTC(),
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	if err := consentStore.Create(ctx, &consent); err != nil {
		t.Fatalf("create consent: %v", err)
	}

	err = recLC.Transition(ctx, recommendation.TransitionRequest{
		RecommendationID: rec.ID,
		ToState:          models.RecommendationStateSubmitted,
		ActorID:          uuid.New(),
		ActorClass:       recommendation.ActorClassHuman,
		ReasoningSummary: "consent granted by SDM",
	})
	if err != nil {
		t.Fatalf("expected transition to succeed once active consent exists; got %v", err)
	}

	got, _ = recStore.Get(ctx, rec.ID)
	if got.State != models.RecommendationStateSubmitted {
		t.Errorf("rec state = %q, want submitted", got.State)
	}
	if got.SubmittedAt == nil {
		t.Errorf("submitted_at must be populated after successful transition")
	}

	if n := nodeWriter.count(); n != 1 {
		t.Errorf("expected 1 evidence node after successful transition; got %d", n)
	}
}

// capturingNodeWriter is a NodeWriter that records every upserted node.
// Lives in this test file so the recommendation package's fakeNodeWriter
// (defined in edge_adapter_test.go) doesn't need to be exported.
type capturingNodeWriter struct {
	nodes []models.EvidenceTraceNode
}

func newCapturingNodeWriter() *capturingNodeWriter {
	return &capturingNodeWriter{}
}

func (c *capturingNodeWriter) UpsertEvidenceTraceNode(_ context.Context,
	n models.EvidenceTraceNode) (*models.EvidenceTraceNode, error) {
	c.nodes = append(c.nodes, n)
	return &n, nil
}

func (c *capturingNodeWriter) count() int {
	return len(c.nodes)
}
