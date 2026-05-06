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

func TestKB20Client_UpsertGetObservation(t *testing.T) {
	val := 132.0
	captured := models.Observation{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v2/observations" {
			_ = json.NewDecoder(r.Body).Decode(&captured)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(captured)
			return
		}
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/observations/") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(captured)
			return
		}
		http.Error(w, "unexpected route "+r.Method+" "+r.URL.Path, http.StatusNotFound)
	}))
	defer ts.Close()

	c := NewKB20Client(ts.URL)
	in := models.Observation{
		ID: uuid.New(), ResidentID: uuid.New(),
		Kind: models.ObservationKindVital, LOINCCode: "8480-6",
		Value: &val, Unit: "mmHg", ObservedAt: time.Now().UTC(),
	}
	out, err := c.UpsertObservation(context.Background(), in)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if out.ID != in.ID {
		t.Errorf("Upsert ID drift: got %v want %v", out.ID, in.ID)
	}
	got, err := c.GetObservation(context.Background(), in.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Kind != models.ObservationKindVital {
		t.Errorf("Get Kind drift: got %q", got.Kind)
	}
}

func TestKB20Client_ListObservationsByResidentAndKind_BuildsURL(t *testing.T) {
	var seenPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path + "?" + r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer ts.Close()

	c := NewKB20Client(ts.URL)
	rid := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	_, err := c.ListObservationsByResidentAndKind(context.Background(), rid, models.ObservationKindWeight, 50, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	expected := "/v2/residents/11111111-2222-3333-4444-555555555555/observations/weight?limit=50&offset=10"
	if seenPath != expected {
		t.Errorf("URL mismatch: got %q want %q", seenPath, expected)
	}
}
