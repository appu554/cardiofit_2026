package storage

import (
	"testing"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// parseUUIDArray is the only piece of evidence_trace_views.go reachable
// without a live PostgreSQL connection. The DB-bound query methods are
// covered by the kb-20 integration test pack which runs only when
// KB20_TEST_DATABASE_URL is set (matching Wave 1R/Wave 2 precedent).

func TestParseUUIDArray_Empty(t *testing.T) {
	if got := parseUUIDArray(nil); got != nil {
		t.Fatalf("nil input → nil expected, got %v", got)
	}
	if got := parseUUIDArray(pq.StringArray{}); got != nil {
		t.Fatalf("empty input → nil expected, got %v", got)
	}
}

func TestParseUUIDArray_HappyPath(t *testing.T) {
	a := uuid.New()
	b := uuid.New()
	in := pq.StringArray{a.String(), b.String()}
	out := parseUUIDArray(in)
	if len(out) != 2 {
		t.Fatalf("want 2 elements, got %d", len(out))
	}
	if out[0] != a || out[1] != b {
		t.Fatalf("element drift: %v", out)
	}
}

func TestParseUUIDArray_MixedValid(t *testing.T) {
	a := uuid.New()
	in := pq.StringArray{a.String(), "", "not-a-uuid", uuid.Nil.String()}
	out := parseUUIDArray(in)
	// Nil UUID is technically parseable; we keep it. The malformed entry
	// and the empty entry are dropped.
	if len(out) != 2 {
		t.Fatalf("want 2 elements (1 real + 1 nil-uuid), got %d (%v)", len(out), out)
	}
	if out[0] != a {
		t.Fatalf("first element should be the real UUID, got %v", out[0])
	}
}

func TestParseUUIDArray_AllInvalid(t *testing.T) {
	in := pq.StringArray{"x", "y", ""}
	if got := parseUUIDArray(in); got != nil {
		t.Fatalf("all-invalid input → nil expected, got %v", got)
	}
}
