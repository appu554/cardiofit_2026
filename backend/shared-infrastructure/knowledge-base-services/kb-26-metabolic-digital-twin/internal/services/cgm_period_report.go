// PROVISIONAL (Phase 7 P7-E Decision 2): this Go implementation of the
// CGM period-report compute is a duplicate of the canonical Java
// implementation at
// backend/shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/Module3_CGMAnalytics.java
//
// The Java version is the source of record: it runs inside
// Module3_CGMStreamJob as the 14-day sliding window compute and its
// output (CGMAnalyticsEvent) flows through clinical.cgm-analytics.v1
// to KB-26's cgm_analytics_consumer, which persists it as a
// CGMPeriodReport row — which is what downstream callers actually read.
//
// This Go file remains for two reasons: (1) it preserves the Phase 6
// P6-4 unit test surface for the pure compute function, useful for
// regression checks when the Java version changes; (2) the
// cgm_daily_batch.go heartbeat still imports types from this file.
// Do NOT extend this file with new clinical logic — add to the Java
// side. A future Phase 8 consolidation may delete this file once the
// Flink pipeline is running in production and the daily batch migrates
// to a pure read-path.
package services

import (
	"math"
	"time"

	"kb-26-metabolic-digital-twin/internal/models"
)

// GlucoseReading is a single CGM measurement. Phase 6 P6-4 — the batch
// accepts a slice of these and produces a full CGMPeriodReport.
type GlucoseReading struct {
	Timestamp time.Time
	ValueMgDL float64
}

// ADA 2023 consensus ranges (mg/dL):
const (
	// Time in Range target band
	tirLowMgDL  = 70.0
	tirHighMgDL = 180.0
	// Level-1 hypoglycaemia (clinically significant)
	tbrL1Threshold = 70.0
	// Level-2 hypoglycaemia (severe)
	tbrL2Threshold = 54.0
	// Level-1 hyperglycaemia
	tarL1Threshold = 180.0
	// Level-2 hyperglycaemia (severe)
	tarL2Threshold = 250.0
	// Minimum data capture for a reliable period report (ADA: ≥70% of
	// the window must have readings — a 14-day window needs ≥9.8 days
	// of data).
	minCoveragePct = 70.0
)

// ComputePeriodReport produces a CGMPeriodReport from raw glucose readings
// within [periodStart, periodEnd]. Phase 6 P6-4.
//
// The computation is defensive: an empty reading slice produces a zero
// report with SufficientData=false; insufficient coverage (<70% of the
// period window) also yields SufficientData=false with a LOW confidence
// level so downstream consumers know not to treat the metrics as
// clinically actionable.
//
// Metrics per ADA 2023 consensus + International Consensus on Time in
// Range (Battelino 2019):
//   - TIR  = % readings in [70, 180] mg/dL
//   - TBR_L1 = % readings in [54, 70) mg/dL
//   - TBR_L2 = % readings < 54 mg/dL
//   - TAR_L1 = % readings in (180, 250] mg/dL
//   - TAR_L2 = % readings > 250 mg/dL
//   - CV   = (stdev / mean) * 100
//   - GMI  = 3.31 + 0.02392 * mean_glucose_mg_dL
//
// Full AGP percentile overlays are deferred to Phase 7 per the Phase 6
// plan Locked Decision 5.
func ComputePeriodReport(readings []GlucoseReading, periodStart, periodEnd time.Time) models.CGMPeriodReport {
	report := models.CGMPeriodReport{
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		CreatedAt:   time.Now().UTC(),
	}

	// Filter to the reporting window.
	inWindow := make([]GlucoseReading, 0, len(readings))
	for _, r := range readings {
		if r.Timestamp.Before(periodStart) || r.Timestamp.After(periodEnd) {
			continue
		}
		inWindow = append(inWindow, r)
	}

	if len(inWindow) == 0 {
		report.SufficientData = false
		report.ConfidenceLevel = "LOW"
		return report
	}

	// Coverage: expected reading count for FreeStyle Libre is every
	// ~15 minutes (96 per day); other sensors every 5 minutes (288 per
	// day). Use the pessimistic 96/day so coverage is only high when
	// data is genuinely dense.
	windowDays := periodEnd.Sub(periodStart).Hours() / 24
	if windowDays <= 0 {
		windowDays = 1
	}
	expected := windowDays * 96
	report.CoveragePct = math.Min(100.0, (float64(len(inWindow))/expected)*100)
	report.SufficientData = report.CoveragePct >= minCoveragePct

	switch {
	case report.CoveragePct >= 90:
		report.ConfidenceLevel = "HIGH"
	case report.CoveragePct >= minCoveragePct:
		report.ConfidenceLevel = "MEDIUM"
	default:
		report.ConfidenceLevel = "LOW"
	}

	// Aggregate metrics.
	var sum, sumSq float64
	var tir, tbrL1, tbrL2, tarL1, tarL2 int
	for _, r := range inWindow {
		v := r.ValueMgDL
		sum += v
		sumSq += v * v
		switch {
		case v < tbrL2Threshold:
			tbrL2++
		case v < tbrL1Threshold:
			tbrL1++
		case v > tarL2Threshold:
			tarL2++
		case v > tarL1Threshold:
			tarL1++
		default:
			if v >= tirLowMgDL && v <= tirHighMgDL {
				tir++
			}
		}
	}

	n := float64(len(inWindow))
	report.MeanGlucose = sum / n
	// Sample standard deviation using the unbiased estimator.
	variance := (sumSq - (sum*sum)/n) / n
	if variance < 0 {
		variance = 0
	}
	report.SDGlucose = math.Sqrt(variance)
	if report.MeanGlucose > 0 {
		report.CVPct = (report.SDGlucose / report.MeanGlucose) * 100
	}
	report.GlucoseStable = report.CVPct < 36.0 // ADA consensus stability cutoff

	report.TIRPct = (float64(tir) / n) * 100
	report.TBRL1Pct = (float64(tbrL1) / n) * 100
	report.TBRL2Pct = (float64(tbrL2) / n) * 100
	report.TARL1Pct = (float64(tarL1) / n) * 100
	report.TARL2Pct = (float64(tarL2) / n) * 100

	// GMI (Glucose Management Indicator) per Bergenstal 2018 (Diabetes Care).
	report.GMI = 3.31 + 0.02392*report.MeanGlucose

	// Simplified GRI (Glycemia Risk Index) per Klonoff 2022 (J Diab Sci
	// Tech). Formal formula: GRI = (3.0 × VLow) + (2.4 × Low) + (1.6 × VHigh) + (0.8 × High)
	// where VLow = TBRL2, Low = TBRL1, VHigh = TARL2, High = TARL1.
	report.GRI = 3.0*report.TBRL2Pct + 2.4*report.TBRL1Pct +
		1.6*report.TARL2Pct + 0.8*report.TARL1Pct

	switch {
	case report.GRI < 20:
		report.GRIZone = "A"
	case report.GRI < 40:
		report.GRIZone = "B"
	case report.GRI < 60:
		report.GRIZone = "C"
	case report.GRI < 80:
		report.GRIZone = "D"
	default:
		report.GRIZone = "E"
	}

	return report
}
