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

func TestKB20Client_CreateCareIntensity(t *testing.T) {
	rid := uuid.New()
	roleRef := uuid.New()

	var seenPath, seenMethod string
	var captured CreateCareIntensityRequest

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		seenMethod = r.Method
		_ = json.NewDecoder(r.Body).Decode(&captured)

		out := interfaces.CareIntensityTransitionResult{
			CareIntensity: &models.CareIntensity{
				ID:                  uuid.New(),
				ResidentRef:         rid,
				Tag:                 captured.Tag,
				EffectiveDate:       captured.EffectiveDate,
				DocumentedByRoleRef: captured.DocumentedByRoleRef,
			},
			Event: &models.Event{
				ID:            uuid.New(),
				EventType:     models.EventTypeCareIntensityTransition,
				ResidentID:    rid,
				ReportedByRef: captured.DocumentedByRoleRef,
				Severity:      models.EventSeverityModerate,
			},
			Cascades: []interfaces.CareIntensityCascadeHint{
				{Kind: "review_preventive_medications", Reason: "test"},
				{Kind: "revisit_monitoring_plan", Reason: "test"},
				{Kind: "consent_refresh_needed", Reason: "test"},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}))
	defer ts.Close()

	c := NewKB20Client(ts.URL)
	req := CreateCareIntensityRequest{
		Tag:                 models.CareIntensityTagPalliative,
		EffectiveDate:       time.Now().UTC().Truncate(time.Second),
		DocumentedByRoleRef: roleRef,
	}
	out, err := c.CreateCareIntensity(context.Background(), rid, req)
	if err != nil {
		t.Fatalf("CreateCareIntensity: %v", err)
	}
	if seenMethod != http.MethodPost {
		t.Errorf("method: got %s want POST", seenMethod)
	}
	if seenPath != "/v2/residents/"+rid.String()+"/care-intensity" {
		t.Errorf("path: got %s", seenPath)
	}
	if captured.Tag != models.CareIntensityTagPalliative {
		t.Errorf("captured Tag drift: %s", captured.Tag)
	}
	if out.CareIntensity == nil || out.CareIntensity.ResidentRef != rid {
		t.Errorf("CareIntensity result drift")
	}
	if out.Event == nil || out.Event.EventType != models.EventTypeCareIntensityTransition {
		t.Errorf("Event result drift")
	}
	if len(out.Cascades) != 3 {
		t.Errorf("expected 3 cascades; got %d", len(out.Cascades))
	}
}

func TestKB20Client_GetCurrentCareIntensity(t *testing.T) {
	rid := uuid.New()
	var seenPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(models.CareIntensity{
			ID:                  uuid.New(),
			ResidentRef:         rid,
			Tag:                 models.CareIntensityTagActiveTreatment,
			EffectiveDate:       time.Now().UTC(),
			DocumentedByRoleRef: uuid.New(),
		})
	}))
	defer ts.Close()

	c := NewKB20Client(ts.URL)
	got, err := c.GetCurrentCareIntensity(context.Background(), rid)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if seenPath != "/v2/residents/"+rid.String()+"/care-intensity/current" {
		t.Errorf("path: got %s", seenPath)
	}
	if got.Tag != models.CareIntensityTagActiveTreatment {
		t.Errorf("tag drift: %s", got.Tag)
	}
}

func TestKB20Client_ListCareIntensityHistory(t *testing.T) {
	rid := uuid.New()
	var seenPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]models.CareIntensity{
			{ID: uuid.New(), ResidentRef: rid, Tag: models.CareIntensityTagPalliative, EffectiveDate: time.Now().UTC()},
			{ID: uuid.New(), ResidentRef: rid, Tag: models.CareIntensityTagComfortFocused, EffectiveDate: time.Now().UTC().Add(-24 * time.Hour)},
		})
	}))
	defer ts.Close()

	c := NewKB20Client(ts.URL)
	got, err := c.ListCareIntensityHistory(context.Background(), rid)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if seenPath != "/v2/residents/"+rid.String()+"/care-intensity/history" {
		t.Errorf("path: got %s", seenPath)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 rows; got %d", len(got))
	}
}
