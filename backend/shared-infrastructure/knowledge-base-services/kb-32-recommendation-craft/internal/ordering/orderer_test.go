package ordering

import (
	"testing"

	"github.com/cardiofit/kb32/internal/generator"
	"github.com/google/uuid"
)

// makePacket is a test helper that constructs a minimal *generator.Packet with
// the given Type. The RecommendationID is set so packets are distinguishable.
func makePacket(packetType string) *generator.Packet {
	return &generator.Packet{
		RecommendationID: uuid.New(),
		Type:             packetType,
	}
}

// ---------------------------------------------------------------------------
// Basic ordering
// ---------------------------------------------------------------------------

func TestOrder_StopFirst(t *testing.T) {
	add := makePacket("ADD")
	monitor := makePacket("MONITOR")
	stop := makePacket("STOP")

	out := Order([]*generator.Packet{add, monitor, stop})

	if len(out) != 3 {
		t.Fatalf("len = %d, want 3", len(out))
	}
	if out[0].Type != "STOP" {
		t.Errorf("out[0].Type = %q, want STOP", out[0].Type)
	}
	if out[1].Type != "MONITOR" {
		t.Errorf("out[1].Type = %q, want MONITOR", out[1].Type)
	}
	if out[2].Type != "ADD" {
		t.Errorf("out[2].Type = %q, want ADD", out[2].Type)
	}
}

// ---------------------------------------------------------------------------
// Stable sort within same type
// ---------------------------------------------------------------------------

func TestOrder_StableSortWithinType(t *testing.T) {
	stop1 := makePacket("STOP")
	stop2 := makePacket("STOP")
	add := makePacket("ADD")

	// Record original pointer identity for the two STOP packets.
	out := Order([]*generator.Packet{stop1, stop2, add})

	if len(out) != 3 {
		t.Fatalf("len = %d, want 3", len(out))
	}
	// The two STOP packets must retain their relative input order (stable).
	if out[0] != stop1 {
		t.Errorf("out[0] is not stop1 — stable sort violated for STOP type")
	}
	if out[1] != stop2 {
		t.Errorf("out[1] is not stop2 — stable sort violated for STOP type")
	}
}

// ---------------------------------------------------------------------------
// Anti-suppression invariant
// ---------------------------------------------------------------------------

func TestOrder_AntiSuppressionInvariant(t *testing.T) {
	packets := []*generator.Packet{
		makePacket("ADD"),
		makePacket("STOP"),
		makePacket("MONITOR"),
		makePacket("DOSE_CHANGE"),
		makePacket("ADD"),
	}
	out := Order(packets)

	if len(out) != len(packets) {
		t.Errorf("len(out) = %d, want %d — anti-suppression invariant violated", len(out), len(packets))
	}
}

// ---------------------------------------------------------------------------
// Empty input
// ---------------------------------------------------------------------------

func TestOrder_EmptyInput(t *testing.T) {
	out := Order([]*generator.Packet{})

	if out == nil {
		t.Fatal("Order(empty) returned nil; want non-nil empty slice")
	}
	if len(out) != 0 {
		t.Errorf("len = %d, want 0", len(out))
	}
	// Confirm sliceability.
	_ = out[0:0]
}

// ---------------------------------------------------------------------------
// Unknown type sorts to end
// ---------------------------------------------------------------------------

func TestOrder_UnknownType(t *testing.T) {
	// "WAFFLE" is not in typeRank. It must receive math.MaxInt and sort after
	// all recognised types. This is forward-compatibility behaviour: new types
	// introduced in future pipeline versions pass through without panicking.
	waffle := makePacket("WAFFLE")
	stop := makePacket("STOP")
	add := makePacket("ADD")

	out := Order([]*generator.Packet{waffle, stop, add})

	if len(out) != 3 {
		t.Fatalf("len = %d, want 3", len(out))
	}
	if out[0].Type != "STOP" {
		t.Errorf("out[0].Type = %q, want STOP", out[0].Type)
	}
	if out[1].Type != "ADD" {
		t.Errorf("out[1].Type = %q, want ADD", out[1].Type)
	}
	if out[2].Type != "WAFFLE" {
		t.Errorf("out[2].Type = %q, want WAFFLE (unknown type must sort to end)", out[2].Type)
	}
}

// ---------------------------------------------------------------------------
// Input not mutated
// ---------------------------------------------------------------------------

func TestOrder_DoesNotMutateInput(t *testing.T) {
	p1 := makePacket("ADD")
	p2 := makePacket("STOP")
	input := []*generator.Packet{p1, p2}

	_ = Order(input)

	// Original slice order must be preserved.
	if input[0] != p1 || input[1] != p2 {
		t.Error("Order mutated the input slice")
	}
}
