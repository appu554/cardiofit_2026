package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

func TestKB20Client_CreateActiveConcern(t *testing.T) {
	rid := uuid.New()
	startedBy := uuid.New()

	var seenPath, seenMethod string
	var captured CreateActiveConcernRequest

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		seenMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&captured)
		out := models.ActiveConcern{
			ID:                   uuid.New(),
			ResidentID:           rid,
			ConcernType:          captured.ConcernType,
			StartedAt:            captured.StartedAt,
			StartedByEventRef:    captured.StartedByEventRef,
			ExpectedResolutionAt: captured.ExpectedResolutionAt,
			ResolutionStatus:     models.ResolutionStatusOpen,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}))
	defer ts.Close()

	c := NewKB20Client(ts.URL)
	started := time.Now().UTC().Truncate(time.Second)
	req := CreateActiveConcernRequest{
		ConcernType:          models.ActiveConcernPostFall72h,
		StartedAt:            started,
		StartedByEventRef:    &startedBy,
		ExpectedResolutionAt: started.Add(72 * time.Hour),
	}
	out, err := c.CreateActiveConcern(context.Background(), rid, req)
	if err != nil {
		t.Fatalf("CreateActiveConcern: %v", err)
	}
	if seenMethod != http.MethodPost {
		t.Errorf("method: got %s want POST", seenMethod)
	}
	if seenPath != "/v2/residents/"+rid.String()+"/active-concerns" {
		t.Errorf("path: got %s", seenPath)
	}
	if captured.ConcernType != models.ActiveConcernPostFall72h {
		t.Errorf("captured concern_type drift")
	}
	if out.ResidentID != rid {
		t.Errorf("ResidentID drift")
	}
	if out.ResolutionStatus != models.ResolutionStatusOpen {
		t.Errorf("ResolutionStatus drift: got %s", out.ResolutionStatus)
	}
}

func TestKB20Client_ListActiveConcernsByResident_BuildsURL(t *testing.T) {
	rid := uuid.New()
	var seenPath, seenQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		seenQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]models.ActiveConcern{})
	}))
	defer ts.Close()

	c := NewKB20Client(ts.URL)
	if _, err := c.ListActiveConcernsByResident(context.Background(), rid, models.ResolutionStatusOpen); err != nil {
		t.Fatalf("List: %v", err)
	}
	if seenPath != "/v2/residents/"+rid.String()+"/active-concerns" {
		t.Errorf("path: got %s", seenPath)
	}
	if !strings.Contains(seenQuery, "status=open") {
		t.Errorf("query: expected status=open, got %s", seenQuery)
	}
}

func TestKB20Client_PatchActiveConcernResolution(t *testing.T) {
	id := uuid.New()
	var seenPath, seenMethod string
	var captured PatchActiveConcernResolutionRequest
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		seenMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&captured)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(models.ActiveConcern{
			ID: id, ResolutionStatus: captured.ResolutionStatus, ResolvedAt: &captured.ResolvedAt,
		})
	}))
	defer ts.Close()

	c := NewKB20Client(ts.URL)
	now := time.Now().UTC().Truncate(time.Second)
	out, err := c.PatchActiveConcernResolution(context.Background(), id, PatchActiveConcernResolutionRequest{
		ResolutionStatus: models.ResolutionStatusEscalated,
		ResolvedAt:       now,
	})
	if err != nil {
		t.Fatalf("Patch: %v", err)
	}
	if seenMethod != http.MethodPatch {
		t.Errorf("method: got %s want PATCH", seenMethod)
	}
	if seenPath != "/v2/active-concerns/"+id.String() {
		t.Errorf("path: got %s", seenPath)
	}
	if captured.ResolutionStatus != models.ResolutionStatusEscalated {
		t.Errorf("captured resolution_status drift")
	}
	if out.ResolutionStatus != models.ResolutionStatusEscalated {
		t.Errorf("out resolution_status drift: got %s", out.ResolutionStatus)
	}
}

func TestKB20Client_ListExpiringActiveConcerns(t *testing.T) {
	var seenQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]models.ActiveConcern{})
	}))
	defer ts.Close()

	c := NewKB20Client(ts.URL)
	if _, err := c.ListExpiringActiveConcerns(context.Background(), 6*time.Hour); err != nil {
		t.Fatalf("ListExpiring: %v", err)
	}
	if !strings.Contains(seenQuery, "within=6h") {
		t.Errorf("query: expected within=6h, got %s", seenQuery)
	}
}
