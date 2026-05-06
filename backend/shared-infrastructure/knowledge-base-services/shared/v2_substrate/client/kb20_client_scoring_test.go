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

func TestKB20Client_CreateCFSScore_HighScoreReturnsHint(t *testing.T) {
	rid := uuid.New()
	roleRef := uuid.New()
	var seenPath, seenMethod string
	var captured CreateCFSScoreRequest

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		seenMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&captured)
		scoreID := uuid.New()
		out := interfaces.ScoringResult{
			CFSScore: &models.CFSScore{
				ID:                scoreID,
				ResidentRef:       rid,
				AssessedAt:        captured.AssessedAt,
				AssessorRoleRef:   captured.AssessorRoleRef,
				InstrumentVersion: captured.InstrumentVersion,
				Score:             captured.Score,
			},
			CareIntensityHint: &interfaces.CareIntensityReviewHint{
				Instrument: "CFS",
				Score:      captured.Score,
				ScoreRef:   scoreID,
				Reason:     "CFS>=7 — consider care intensity review",
			},
			EvidenceTraceNodeRef: uuid.New(),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}))
	defer ts.Close()

	c := NewKB20Client(ts.URL)
	req := CreateCFSScoreRequest{
		AssessedAt:        time.Now().UTC().Truncate(time.Second),
		AssessorRoleRef:   roleRef,
		InstrumentVersion: "v2.0",
		Score:             7,
	}
	out, err := c.CreateCFSScore(context.Background(), rid, req)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if seenMethod != http.MethodPost {
		t.Errorf("method drift: %s", seenMethod)
	}
	if seenPath != "/v2/residents/"+rid.String()+"/cfs" {
		t.Errorf("path drift: %s", seenPath)
	}
	if out.CFSScore == nil || out.CFSScore.Score != 7 {
		t.Errorf("CFSScore drift: %+v", out.CFSScore)
	}
	if out.CareIntensityHint == nil {
		t.Errorf("expected hint for CFS=7")
	}
	if out.EvidenceTraceNodeRef == uuid.Nil {
		t.Errorf("expected EvidenceTraceNodeRef")
	}
}

func TestKB20Client_CreateAKPSScore(t *testing.T) {
	rid := uuid.New()
	var captured CreateAKPSScoreRequest
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&captured)
		out := interfaces.ScoringResult{
			AKPSScore: &models.AKPSScore{
				ID:                uuid.New(),
				ResidentRef:       rid,
				AssessedAt:        captured.AssessedAt,
				AssessorRoleRef:   captured.AssessorRoleRef,
				InstrumentVersion: captured.InstrumentVersion,
				Score:             captured.Score,
			},
			EvidenceTraceNodeRef: uuid.New(),
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}))
	defer ts.Close()
	c := NewKB20Client(ts.URL)
	req := CreateAKPSScoreRequest{
		AssessedAt:        time.Now().UTC().Truncate(time.Second),
		AssessorRoleRef:   uuid.New(),
		InstrumentVersion: "abernethy_2005",
		Score:             50,
	}
	out, err := c.CreateAKPSScore(context.Background(), rid, req)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if out.AKPSScore == nil || out.AKPSScore.Score != 50 {
		t.Errorf("AKPSScore drift")
	}
	if out.CareIntensityHint != nil {
		t.Errorf("expected no hint for AKPS=50")
	}
}

func TestKB20Client_GetCurrentScoresAggregatesAllFour(t *testing.T) {
	rid := uuid.New()
	cfsID := uuid.New()
	akpsID := uuid.New()
	dbiID := uuid.New()
	acbID := uuid.New()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out := interfaces.CurrentScores{
			CFS:  &models.CFSScore{ID: cfsID, ResidentRef: rid, Score: 6},
			AKPS: &models.AKPSScore{ID: akpsID, ResidentRef: rid, Score: 60},
			DBI:  &models.DBIScore{ID: dbiID, ResidentRef: rid, Score: 1.5},
			ACB:  &models.ACBScore{ID: acbID, ResidentRef: rid, Score: 4},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}))
	defer ts.Close()

	c := NewKB20Client(ts.URL)
	out, err := c.GetCurrentScores(context.Background(), rid)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if out.CFS == nil || out.CFS.ID != cfsID {
		t.Errorf("CFS drift")
	}
	if out.AKPS == nil || out.AKPS.ID != akpsID {
		t.Errorf("AKPS drift")
	}
	if out.DBI == nil || out.DBI.ID != dbiID {
		t.Errorf("DBI drift")
	}
	if out.ACB == nil || out.ACB.ID != acbID {
		t.Errorf("ACB drift")
	}
}

func TestKB20Client_ListCFSHistory(t *testing.T) {
	rid := uuid.New()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out := []models.CFSScore{
			{ID: uuid.New(), ResidentRef: rid, Score: 5, AssessedAt: time.Now()},
			{ID: uuid.New(), ResidentRef: rid, Score: 4, AssessedAt: time.Now().Add(-24 * time.Hour)},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}))
	defer ts.Close()
	c := NewKB20Client(ts.URL)
	out, err := c.ListCFSHistory(context.Background(), rid)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(out) != 2 {
		t.Errorf("expected 2 rows; got %d", len(out))
	}
}
