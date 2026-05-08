package metrics

import (
	"bytes"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/prometheus/common/expfmt"
)

// formatHistogram renders the EvaluationLatency histogram in Prometheus
// text exposition format so tests can grep for label values.
func formatHistogram(t *testing.T) string {
	t.Helper()
	reg := prometheus.NewRegistry()
	if err := reg.Register(EvaluationLatency); err != nil {
		// Already registered globally via promauto — gather from the
		// default gatherer instead.
		mfs, err := prometheus.DefaultGatherer.Gather()
		if err != nil {
			t.Fatalf("gather error: %v", err)
		}
		var buf bytes.Buffer
		enc := expfmt.NewEncoder(&buf, expfmt.NewFormat(expfmt.TypeTextPlain))
		for _, mf := range mfs {
			if mf.GetName() == "kb30_authorise_evaluation_latency_seconds" {
				if err := enc.Encode(mf); err != nil {
					t.Fatalf("encode error: %v", err)
				}
			}
		}
		return buf.String()
	}
	mfs, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather error: %v", err)
	}
	var buf bytes.Buffer
	enc := expfmt.NewEncoder(&buf, expfmt.NewFormat(expfmt.TypeTextPlain))
	for _, mf := range mfs {
		if err := enc.Encode(mf); err != nil {
			t.Fatalf("encode error: %v", err)
		}
	}
	return buf.String()
}

func TestObserveEvaluation_AllowOutcome(t *testing.T) {
	EvaluationLatency.Reset()

	ObserveEvaluation(OutcomeAllow, 0.012)
	ObserveEvaluation(OutcomeAllow, 0.087)

	count := testutil.CollectAndCount(EvaluationLatency)
	if count == 0 {
		t.Errorf("expected histogram to have observations; got 0")
	}

	got := formatHistogram(t)
	if !strings.Contains(got, `outcome="allow"`) {
		t.Errorf(`expected outcome="allow" label in output; got %q`, got)
	}
}

func TestObserveEvaluation_AllOutcomeLabels(t *testing.T) {
	EvaluationLatency.Reset()

	ObserveEvaluation(OutcomeAllow, 0.005)
	ObserveEvaluation(OutcomeDeny, 0.015)
	ObserveEvaluation(OutcomeError, 0.030)

	got := formatHistogram(t)
	for _, label := range []string{`outcome="allow"`, `outcome="deny"`, `outcome="error"`} {
		if !strings.Contains(got, label) {
			t.Errorf("expected label %s in output; got %q", label, got)
		}
	}
}
