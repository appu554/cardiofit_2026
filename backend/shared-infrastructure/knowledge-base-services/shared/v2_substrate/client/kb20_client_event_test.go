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

func TestKB20Client_UpsertGetEvent(t *testing.T) {
	captured := models.Event{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v2/events" {
			_ = json.NewDecoder(r.Body).Decode(&captured)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(captured)
			return
		}
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/v2/events/") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(captured)
			return
		}
		http.Error(w, "unexpected route "+r.Method+" "+r.URL.Path, http.StatusNotFound)
	}))
	defer ts.Close()

	c := NewKB20Client(ts.URL)
	in := models.Event{
		ID:            uuid.New(),
		EventType:     models.EventTypeGPVisit,
		OccurredAt:    time.Now().UTC().Truncate(time.Second),
		ResidentID:    uuid.New(),
		ReportedByRef: uuid.New(),
	}
	out, err := c.UpsertEvent(context.Background(), in)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if out.ID != in.ID {
		t.Errorf("Upsert ID drift: got %v want %v", out.ID, in.ID)
	}
	got, err := c.GetEvent(context.Background(), in.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.EventType != models.EventTypeGPVisit {
		t.Errorf("Get EventType drift: got %q", got.EventType)
	}
}

func TestKB20Client_ListEventsByResident_BuildsURL(t *testing.T) {
	var seenPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenPath = r.URL.Path + "?" + r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer ts.Close()

	c := NewKB20Client(ts.URL)
	rid := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	_, err := c.ListEventsByResident(context.Background(), rid, 25, 5)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	expected := "/v2/residents/11111111-2222-3333-4444-555555555555/events?limit=25&offset=5"
	if seenPath != expected {
		t.Errorf("URL mismatch: got %q want %q", seenPath, expected)
	}
}

func TestKB20Client_ListEventsByType_BuildsURL(t *testing.T) {
	var seenURL string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenURL = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer ts.Close()

	c := NewKB20Client(ts.URL)
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	_, err := c.ListEventsByType(context.Background(), models.EventTypeFall, from, to, 50, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, want := range []string{"type=fall", "from=2026-01-01", "to=2026-06-01", "limit=50", "offset=0"} {
		if !strings.Contains(seenURL, want) {
			t.Errorf("URL missing %q; got %q", want, seenURL)
		}
	}
}

func TestKB20Client_ListEventsByType_OmitsZeroBounds(t *testing.T) {
	var seenURL string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenURL = r.URL.String()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer ts.Close()

	c := NewKB20Client(ts.URL)
	_, err := c.ListEventsByType(context.Background(), models.EventTypeRuleFire, time.Time{}, time.Time{}, 10, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if strings.Contains(seenURL, "from=") || strings.Contains(seenURL, "to=") {
		t.Errorf("expected from/to omitted when zero; got %q", seenURL)
	}
}
