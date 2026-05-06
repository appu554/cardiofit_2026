package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/identity"
	"github.com/cardiofit/shared/v2_substrate/interfaces"
)

func TestKB20Client_MatchIdentity(t *testing.T) {
	residentRef := uuid.New()
	nodeRef := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method: got %s want POST", r.Method)
		}
		if r.URL.Path != "/v2/identity/match" {
			t.Fatalf("path: got %s want /v2/identity/match", r.URL.Path)
		}
		var got identity.IncomingIdentifier
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatal(err)
		}
		if got.IHI != "8003608000099111" {
			t.Errorf("IHI mismatch: %q", got.IHI)
		}
		_ = json.NewEncoder(w).Encode(IdentityMatchResponse{
			Match: identity.MatchResult{
				ResidentRef: &residentRef,
				Confidence:  identity.ConfidenceHigh,
				Path:        identity.MatchPathIHI,
			},
			EvidenceTraceNodeRef: nodeRef,
		})
	}))
	defer srv.Close()

	c := NewKB20Client(srv.URL)
	out, err := c.MatchIdentity(context.Background(), identity.IncomingIdentifier{
		IHI:    "8003608000099111",
		Source: "client-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.Match.Confidence != identity.ConfidenceHigh {
		t.Errorf("Confidence: got %s want HIGH", out.Match.Confidence)
	}
	if out.EvidenceTraceNodeRef != nodeRef {
		t.Errorf("EvidenceTraceNodeRef mismatch")
	}
}

func TestKB20Client_ListIdentityReviewQueue(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/v2/identity/review-queue") {
			t.Fatalf("path: got %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("status") != "pending" {
			t.Errorf("status: got %q want pending", q.Get("status"))
		}
		if q.Get("limit") != "25" {
			t.Errorf("limit: got %q want 25", q.Get("limit"))
		}
		if q.Get("offset") != "10" {
			t.Errorf("offset: got %q want 10", q.Get("offset"))
		}
		_ = json.NewEncoder(w).Encode([]interfaces.IdentityReviewQueueEntry{{ID: uuid.New(), Status: "pending"}})
	}))
	defer srv.Close()

	c := NewKB20Client(srv.URL)
	got, err := c.ListIdentityReviewQueue(context.Background(), "pending", 25, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Errorf("len: got %d want 1", len(got))
	}
}

func TestKB20Client_ListIdentityReviewQueue_NoStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Empty status MUST be omitted from the query string.
		if r.URL.Query().Has("status") {
			t.Errorf("status query param should be omitted when empty")
		}
		_ = json.NewEncoder(w).Encode([]interfaces.IdentityReviewQueueEntry{})
	}))
	defer srv.Close()

	c := NewKB20Client(srv.URL)
	if _, err := c.ListIdentityReviewQueue(context.Background(), "", 100, 0); err != nil {
		t.Fatal(err)
	}
}

func TestKB20Client_ResolveIdentityReview(t *testing.T) {
	queueID := uuid.New()
	resolvedRef := uuid.New()
	resolvedBy := uuid.New()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method: got %s want POST", r.Method)
		}
		expected := "/v2/identity/review/" + queueID.String() + "/resolve"
		if r.URL.Path != expected {
			t.Fatalf("path: got %s want %s", r.URL.Path, expected)
		}
		var body struct {
			ResolvedResidentRef uuid.UUID `json:"resolved_resident_ref"`
			ResolvedBy          uuid.UUID `json:"resolved_by"`
			ResolutionNote      string    `json:"resolution_note,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.ResolvedResidentRef != resolvedRef {
			t.Errorf("ResolvedResidentRef mismatch")
		}
		if body.ResolutionNote != "verified" {
			t.Errorf("ResolutionNote: got %q want verified", body.ResolutionNote)
		}
		_ = json.NewEncoder(w).Encode(IdentityResolveResponse{
			Entry:    &interfaces.IdentityReviewQueueEntry{ID: queueID, Status: "resolved"},
			Rerouted: 3,
		})
	}))
	defer srv.Close()

	c := NewKB20Client(srv.URL)
	out, err := c.ResolveIdentityReview(context.Background(), queueID, resolvedRef, resolvedBy, "verified")
	if err != nil {
		t.Fatal(err)
	}
	if out.Rerouted != 3 {
		t.Errorf("Rerouted: got %d want 3", out.Rerouted)
	}
	if out.Entry == nil || out.Entry.Status != "resolved" {
		t.Errorf("Entry status: %v", out.Entry)
	}
}
