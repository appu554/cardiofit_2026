package services

import (
	"math"
	"math/rand"
	"strconv"
	"testing"

	"kb-26-metabolic-digital-twin/internal/models"
)

// generateSyntheticCohort creates a cohort where treatment causes a uniform
// outcome-probability reduction of `trueEffect`. Used to test whether the
// baseline learner recovers a known effect.
func generateSyntheticCohort(n int, trueEffect float64, seed int64) []TrainingRow {
	r := rand.New(rand.NewSource(seed))
	out := make([]TrainingRow, n)
	for i := 0; i < n; i++ {
		age := 60 + r.Float64()*25
		treated := r.Float64() > 0.5
		baseRisk := 0.3 + 0.01*(age-70)
		p := baseRisk
		if treated {
			p -= trueEffect
		}
		if p < 0 {
			p = 0
		}
		if p > 1 {
			p = 1
		}
		out[i] = TrainingRow{
			PatientID:       "T" + string(rune('A'+(i%26))),
			Features:        map[string]float64{"age": age},
			Treated:         treated,
			OutcomeOccurred: r.Float64() < p,
		}
	}
	return out
}

func mustEstimate(t *testing.T, rows []TrainingRow) models.CATEEstimate {
	t.Helper()
	est, err := EstimateFromCohort(rows, "P_test", map[string]float64{"age": 72}, models.OverlapBand{Floor: 0.05, Ceiling: 0.95})
	if err != nil {
		t.Fatalf("estimate: %v", err)
	}
	return est
}

func TestBaselineCATELearner_RecoversKnownEffect(t *testing.T) {
	training := generateSyntheticCohort(500, 0.20, 42)
	est, err := EstimateFromCohort(training, "P_target", map[string]float64{"age": 70}, models.OverlapBand{Floor: 0.05, Ceiling: 0.95})
	if err != nil {
		t.Fatalf("estimate: %v", err)
	}
	if math.Abs(est.PointEstimate-0.20) > 0.08 {
		t.Fatalf("want CATE ≈ 0.20 (±0.08), got %.3f", est.PointEstimate)
	}
	if est.OverlapStatus != models.OverlapPass {
		t.Fatalf("want OVERLAP_PASS, got %s", est.OverlapStatus)
	}
}

func TestBaselineCATELearner_CIWidthShrinksWithN(t *testing.T) {
	narrow := mustEstimate(t, generateSyntheticCohort(1000, 0.15, 7))
	wide := mustEstimate(t, generateSyntheticCohort(80, 0.15, 7))
	narrowW := narrow.CIUpper - narrow.CILower
	wideW := wide.CIUpper - wide.CILower
	if narrowW >= wideW {
		t.Fatalf("expected larger N → narrower CI. narrow(N=1000) width=%.3f, wide(N=80) width=%.3f", narrowW, wideW)
	}
}

func TestBaselineCATELearner_InsufficientDataReturnsStatus(t *testing.T) {
	training := generateSyntheticCohort(5, 0.10, 1) // below minTrainingN=40
	est, err := EstimateFromCohort(training, "P1", map[string]float64{"age": 70}, models.OverlapBand{Floor: 0.05, Ceiling: 0.95})
	if err != nil {
		t.Fatalf("estimate: %v", err)
	}
	if est.OverlapStatus != models.OverlapInsufficientData {
		t.Fatalf("want OVERLAP_INSUFFICIENT_DATA, got %s", est.OverlapStatus)
	}
}

func TestBaselineCATELearner_FeatureContributionsPresent(t *testing.T) {
	training := generateSyntheticCohort(400, 0.12, 3)
	est := mustEstimate(t, training)
	if len(est.FeatureContributionKeys) == 0 {
		t.Fatal("expected at least one feature contribution")
	}
	if est.FeatureContributionsJSON == "" {
		t.Fatal("expected feature contributions JSON payload populated")
	}
}

func TestBaselineCATELearner_CohortCountsPopulated(t *testing.T) {
	training := generateSyntheticCohort(200, 0.15, 11)
	est := mustEstimate(t, training)
	if est.CohortTreatedN == 0 || est.CohortControlN == 0 {
		t.Fatalf("cohort counts should be populated, got treated=%d control=%d", est.CohortTreatedN, est.CohortControlN)
	}
	if est.TrainingN != 200 {
		t.Fatalf("want TrainingN=200, got %d", est.TrainingN)
	}
}

func TestBaselineCATELearner_OverlapFailReturnsStatus(t *testing.T) {
	// Cohort with 100% treated → propensity for any patient ~ 1.0 → OverlapAboveCeiling.
	// This exercises the spec §6.1 hard-guard path end-to-end through EstimateFromCohort.
	rows := make([]TrainingRow, 60)
	for i := range rows {
		rows[i] = TrainingRow{
			PatientID:       "T" + strconv.Itoa(i),
			Features:        map[string]float64{"age": 70 + float64(i%5)},
			Treated:         true, // all treated
			OutcomeOccurred: i%3 == 0,
		}
	}
	est, err := EstimateFromCohort(rows, "P_test", map[string]float64{"age": 72}, models.OverlapBand{Floor: 0.05, Ceiling: 0.95})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if est.OverlapStatus == models.OverlapPass {
		t.Fatalf("expected non-pass overlap status for all-treated cohort, got %s", est.OverlapStatus)
	}
	if est.PointEstimate != 0 {
		t.Fatalf("expected point=0 when overlap fails, got %.4f", est.PointEstimate)
	}
	if est.LearnerType != models.LearnerBaselineDiffMeans {
		t.Fatalf("LearnerType must be set even on overlap-fail, got %q", est.LearnerType)
	}
}
