package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/interfaces"
	"github.com/cardiofit/shared/v2_substrate/models"
)

func TestKB20Client_CreateCapacityAssessment_ImpairedMedicalReturnsEvent(t *testing.T) {
	rid := uuid.New()
	roleRef := uuid.New()
	var seenPath, seenMethod string
	var captured CreateCapacityAssessmentRequest

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		seenMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&captured)

		out := interfaces.CapacityAssessmentResult{
			Assessment: &models.CapacityAssessment{
				ID:              uuid.New(),
				ResidentRef:     rid,
				AssessedAt:      captured.AssessedAt,
				AssessorRoleRef: captured.AssessorRoleRef,
				Domain:          captured.Domain,
				Outcome:         captured.Outcome,
				Duration:        captured.Duration,
			},
			Event: &models.Event{
				ID:            uuid.New(),
				EventType:     models.EventTypeCapacityChange,
				ResidentID:    rid,
				ReportedByRef: captured.AssessorRoleRef,
			},
			EvidenceTraceNodeRef: uuid.New(),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}))
	defer ts.Close()

	c := NewKB20Client(ts.URL)
	req := CreateCapacityAssessmentRequest{
		AssessedAt:      time.Now().UTC().Truncate(time.Second),
		AssessorRoleRef: roleRef,
		Domain:          models.CapacityDomainMedical,
		Outcome:         models.CapacityOutcomeImpaired,
		Duration:        models.CapacityDurationPermanent,
	}
	out, err := c.CreateCapacityAssessment(context.Background(), rid, req)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if seenMethod != http.MethodPost {
		t.Errorf("method drift: %s", seenMethod)
	}
	if seenPath != "/v2/residents/"+rid.String()+"/capacity" {
		t.Errorf("path drift: %s", seenPath)
	}
	if out.Assessment == nil || out.Assessment.Domain != models.CapacityDomainMedical {
		t.Errorf("Assessment drift")
	}
	if out.Event == nil || out.Event.EventType != models.EventTypeCapacityChange {
		t.Errorf("expected capacity_change Event")
	}
	if out.EvidenceTraceNodeRef == uuid.Nil {
		t.Errorf("expected EvidenceTraceNodeRef to be set")
	}
}

func TestKB20Client_GetCurrentCapacityForDomain(t *testing.T) {
	rid := uuid.New()
	var seenPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(models.CapacityAssessment{
			ID:              uuid.New(),
			ResidentRef:     rid,
			AssessedAt:      time.Now().UTC(),
			AssessorRoleRef: uuid.New(),
			Domain:          models.CapacityDomainFinancial,
			Outcome:         models.CapacityOutcomeImpaired,
			Duration:        models.CapacityDurationPermanent,
		})
	}))
	defer ts.Close()

	c := NewKB20Client(ts.URL)
	got, err := c.GetCurrentCapacityForDomain(context.Background(), rid, models.CapacityDomainFinancial)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if seenPath != "/v2/residents/"+rid.String()+"/capacity/current/"+models.CapacityDomainFinancial {
		t.Errorf("path drift: %s", seenPath)
	}
	if got.Domain != models.CapacityDomainFinancial {
		t.Errorf("Domain drift: %s", got.Domain)
	}
}

func TestKB20Client_ListCurrentCapacityByResident(t *testing.T) {
	rid := uuid.New()
	var seenPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]models.CapacityAssessment{
			{ID: uuid.New(), ResidentRef: rid, Domain: models.CapacityDomainMedical, Outcome: models.CapacityOutcomeIntact, Duration: models.CapacityDurationPermanent, AssessedAt: time.Now().UTC()},
			{ID: uuid.New(), ResidentRef: rid, Domain: models.CapacityDomainFinancial, Outcome: models.CapacityOutcomeImpaired, Duration: models.CapacityDurationPermanent, AssessedAt: time.Now().UTC()},
		})
	}))
	defer ts.Close()

	c := NewKB20Client(ts.URL)
	got, err := c.ListCurrentCapacityByResident(context.Background(), rid)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if seenPath != "/v2/residents/"+rid.String()+"/capacity/current" {
		t.Errorf("path drift: %s", seenPath)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 rows; got %d", len(got))
	}
}

func TestKB20Client_ListCapacityHistoryForDomain(t *testing.T) {
	rid := uuid.New()
	var seenPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]models.CapacityAssessment{
			{ID: uuid.New(), ResidentRef: rid, Domain: models.CapacityDomainMedical, Outcome: models.CapacityOutcomeImpaired, Duration: models.CapacityDurationPermanent, AssessedAt: time.Now().UTC()},
			{ID: uuid.New(), ResidentRef: rid, Domain: models.CapacityDomainMedical, Outcome: models.CapacityOutcomeIntact, Duration: models.CapacityDurationPermanent, AssessedAt: time.Now().UTC().Add(-24 * time.Hour)},
		})
	}))
	defer ts.Close()

	c := NewKB20Client(ts.URL)
	got, err := c.ListCapacityHistoryForDomain(context.Background(), rid, models.CapacityDomainMedical)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if seenPath != "/v2/residents/"+rid.String()+"/capacity/history/"+models.CapacityDomainMedical {
		t.Errorf("path drift: %s", seenPath)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 rows; got %d", len(got))
	}
}
