package evidence_trace

import "testing"

func TestIsValidEdgeKind(t *testing.T) {
	for _, k := range []EdgeKind{
		EdgeKindLedTo,
		EdgeKindDerivedFrom,
		EdgeKindEvidenceFor,
		EdgeKindSuppressed,
	} {
		if !IsValidEdgeKind(string(k)) {
			t.Errorf("expected %q valid", k)
		}
	}
	for _, s := range []string{"", "child_of", "LED_TO", "led-to"} {
		if IsValidEdgeKind(s) {
			t.Errorf("expected %q invalid", s)
		}
	}
}
