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

	"github.com/cardiofit/shared/v2_substrate/evidence_trace"
	"github.com/cardiofit/shared/v2_substrate/models"
)

func newRoundTripNode() models.EvidenceTraceNode {
	now := time.Now().UTC().Truncate(time.Second)
	return models.EvidenceTraceNode{
		ID:              uuid.New(),
		StateMachine:    models.EvidenceTraceStateMachineRecommendation,
		StateChangeType: "draft -> submitted",
		RecordedAt:      now,
		OccurredAt:      now,
		CreatedAt:       now,
	}
}

func TestKB20Client_UpsertEvidenceTraceNode(t *testing.T) {
	want := newRoundTripNode()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method: got %s, want POST", r.Method)
		}
		if r.URL.Path != "/v2/evidence-trace/nodes" {
			t.Fatalf("path: got %s", r.URL.Path)
		}
		var got models.EvidenceTraceNode
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if got.ID != want.ID {
			t.Errorf("id mismatch")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(got)
	}))
	defer srv.Close()

	c := NewKB20Client(srv.URL)
	out, err := c.UpsertEvidenceTraceNode(context.Background(), want)
	if err != nil {
		t.Fatal(err)
	}
	if out.ID != want.ID {
		t.Errorf("returned ID mismatch")
	}
}

func TestKB20Client_GetEvidenceTraceNode(t *testing.T) {
	want := newRoundTripNode()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := "/v2/evidence-trace/nodes/" + want.ID.String()
		if r.URL.Path != expected {
			t.Fatalf("path: got %s, want %s", r.URL.Path, expected)
		}
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer srv.Close()

	c := NewKB20Client(srv.URL)
	got, err := c.GetEvidenceTraceNode(context.Background(), want.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != want.ID {
		t.Errorf("ID mismatch")
	}
}

func TestKB20Client_InsertEvidenceTraceEdge(t *testing.T) {
	edge := evidence_trace.Edge{
		From: uuid.New(), To: uuid.New(), Kind: evidence_trace.EdgeKindLedTo,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method: got %s, want POST", r.Method)
		}
		if r.URL.Path != "/v2/evidence-trace/edges" {
			t.Fatalf("path: got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	c := NewKB20Client(srv.URL)
	if err := c.InsertEvidenceTraceEdge(context.Background(), edge); err != nil {
		t.Fatal(err)
	}
}

func TestKB20Client_TraceEvidenceForward(t *testing.T) {
	startID := uuid.New()
	nodes := []models.EvidenceTraceNode{newRoundTripNode(), newRoundTripNode()}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := "/v2/evidence-trace/" + startID.String() + "/forward"
		if r.URL.Path != expected {
			t.Fatalf("path: got %s", r.URL.Path)
		}
		if r.URL.Query().Get("depth") != "5" {
			t.Errorf("depth query: got %q, want 5", r.URL.Query().Get("depth"))
		}
		_ = json.NewEncoder(w).Encode(nodes)
	}))
	defer srv.Close()

	c := NewKB20Client(srv.URL)
	got, err := c.TraceEvidenceForward(context.Background(), startID, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Errorf("len: got %d, want 2", len(got))
	}
}

func TestKB20Client_TraceEvidenceBackward(t *testing.T) {
	startID := uuid.New()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/backward") {
			t.Errorf("path should end /backward; got %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode([]models.EvidenceTraceNode{})
	}))
	defer srv.Close()

	c := NewKB20Client(srv.URL)
	got, err := c.TraceEvidenceBackward(context.Background(), startID, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("len: got %d, want 0", len(got))
	}
}
