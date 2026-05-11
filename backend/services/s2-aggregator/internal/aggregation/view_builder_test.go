package aggregation

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestS2ViewBuilder_InterfaceConformance is a compile-time assertion that
// defaultViewBuilder satisfies S2ViewBuilder. If the interface evolves
// without the concrete type keeping up, this test file fails to compile.
func TestS2ViewBuilder_InterfaceConformance(t *testing.T) {
	var _ S2ViewBuilder = (*defaultViewBuilder)(nil)
	var _ S2ViewBuilder = NewDefaultViewBuilder()
}

func newTestReq() WorkspaceRequest {
	return WorkspaceRequest{
		ResidentID:   uuid.New(),
		EntryPath:    EntryPathWorklist,
		PharmacistID: uuid.New(),
		SessionID:    uuid.New(),
		AsOf:         time.Now().UTC(),
	}
}

func TestBuildLayer1Baseline_ReturnsEmptyView_NoError(t *testing.T) {
	b := NewDefaultViewBuilder()
	view, err := b.BuildLayer1Baseline(context.Background(), newTestReq())
	if err != nil {
		t.Fatalf("expected nil error from Layer 1 stub, got %v", err)
	}
	if view.Layer() != 1 {
		t.Fatalf("Layer1View.Layer() = %d, want 1", view.Layer())
	}
	// Zero-value confirmation: the struct is empty in Task 1; Tasks 3–7
	// populate fields per S2 v1.0 Parts 4–13.
	if view != (Layer1View{}) {
		t.Fatalf("expected zero-value Layer1View in Task 1 scaffold")
	}
}

// sentinelCheck asserts that the error message names the layer and cites
// the Addendum. Both substrings are load-bearing: callers (frontend
// error-classifying middleware in particular) pivot on the "Addendum"
// citation to distinguish "deferred by architectural discipline" from
// runtime failures.
func sentinelCheck(t *testing.T, layer int, err error) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected sentinel error for layer %d, got nil", layer)
	}
	msg := err.Error()
	wantLayerPhrase := "layer " + itoa(layer) + " not yet implemented"
	if !strings.Contains(msg, wantLayerPhrase) {
		t.Errorf("error %q missing %q", msg, wantLayerPhrase)
	}
	if !strings.Contains(msg, "Addendum") {
		t.Errorf("error %q does not cite the Addendum (required so callers can classify content-deferred vs runtime errors)", msg)
	}
}

// itoa avoids importing strconv just for one call site.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func TestBuildLayer2_ReturnsNotImplementedSentinel(t *testing.T) {
	b := NewDefaultViewBuilder()
	_, err := b.BuildLayer2Escalated(context.Background(), newTestReq())
	sentinelCheck(t, 2, err)
}

func TestBuildLayer3_ReturnsNotImplementedSentinel(t *testing.T) {
	b := NewDefaultViewBuilder()
	_, err := b.BuildLayer3Complex(context.Background(), newTestReq())
	sentinelCheck(t, 3, err)
}

func TestBuildLayer4_ReturnsNotImplementedSentinel(t *testing.T) {
	b := NewDefaultViewBuilder()
	_, err := b.BuildLayer4SituationBoard(context.Background(), newTestReq())
	sentinelCheck(t, 4, err)
}

func TestBuildLayer5_ReturnsNotImplementedSentinel(t *testing.T) {
	b := NewDefaultViewBuilder()
	_, err := b.BuildLayer5Investigation(context.Background(), newTestReq())
	sentinelCheck(t, 5, err)
}

func TestLogEscalation_WritesEvent(t *testing.T) {
	var buf bytes.Buffer
	b := NewDefaultViewBuilderWithLogger(&buf)
	ev := EscalationEvent{
		PharmacistID: uuid.New(),
		ResidentID:   uuid.New(),
		SessionID:    uuid.New(),
		FromLayer:    1,
		ToLayer:      2,
		TriggeredBy:  TriggerPharmacistInitiated,
		Timestamp:    time.Date(2026, 5, 11, 9, 30, 0, 0, time.UTC),
		AuditTraceID: uuid.New(),
	}
	if err := b.LogEscalation(context.Background(), ev); err != nil {
		t.Fatalf("LogEscalation returned error: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"escalation_event",
		"from_layer=1",
		"to_layer=2",
		"triggered_by=pharmacist_initiated",
		ev.ResidentID.String(),
		ev.PharmacistID.String(),
		ev.SessionID.String(),
		ev.AuditTraceID.String(),
	} {
		if !strings.Contains(out, want) {
			t.Errorf("escalation log %q missing %q", out, want)
		}
	}
}
