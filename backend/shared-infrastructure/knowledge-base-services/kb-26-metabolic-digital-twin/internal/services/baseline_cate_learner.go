package services

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sort"

	"kb-26-metabolic-digital-twin/internal/models"
)

// Baseline learner hyperparameters. Sprint 1 scope — Sprint 2's Python service
// replaces the entire learner behind the same CATEEstimate contract.
const (
	bootstrapResamples  = 500  // number of bootstrap samples for CI
	ciLowerPct          = 5.0  // 90% CI lower percentile
	ciUpperPct          = 95.0 // 90% CI upper percentile
	minTrainingN        = 40   // below this, short-circuit to OverlapInsufficientData
	minBucketArmN       = 5    // below this per arm in the bucket, short-circuit
	topKFeatureContribs = 3    // top-K features to report in contributions
	bucketHalfFraction  = 2    // take len(rows)/bucketHalfFraction nearest neighbours
)

// TrainingRow is one labelled example in the cohort used to fit the baseline CATE
// learner. Populated from Gap 21's ConsolidatedAlertRecord + OutcomeRecord join
// by Task 7's HTTP handler. Exported so the handler (in the `api` package) can
// build this shape without duplicating it.
type TrainingRow struct {
	PatientID       string
	Features        map[string]float64
	Treated         bool
	OutcomeOccurred bool
}

// EstimateFromCohort is the Sprint 1 CATE kernel. It runs:
//  1. Insufficient-data check (combined N < minTrainingN → OverlapInsufficientData).
//  2. Propensity fit on the cohort, patient propensity → overlap diagnostic.
//  3. Nearest-neighbour bucket around the patient on the shared feature set.
//  4. Bootstrap CI on (mean outcome | control) − (mean outcome | treated) inside the bucket.
//  5. Top-K feature contributions = signed bucket-mean minus cohort-mean deltas.
//
// The returned CATEEstimate is the final on-the-wire shape. Caller persists and
// appends the CATE_ESTIMATE ledger entry (Task 7).
//
// Sign convention: positive PointEstimate means treatment reduces the outcome
// probability (i.e. the treated arm has lower outcome-occurrence rate), matching
// Gap 21's AttributionVerdict.RiskDifference sign.
//
// Feature normalization: raw feature values are z-score normalized using cohort
// statistics before propensity fitting. This prevents large-magnitude features
// (e.g. raw age in [60,85]) from driving the gradient descent to degenerate weights
// that push sigmoid to 0 or 1. Patient features are normalized with the same
// cohort statistics so the propensity prediction is consistent.
//
// Missing-feature handling: if patientFeatures omits a key present in the
// training rows, it defaults to 0.0, which after cohort-z-score normalization
// equals the cohort mean for that feature — the patient is treated as "average
// on this axis" rather than as an error. Extra keys in patientFeatures not
// present in training rows are silently ignored. The Task 7 HTTP handler is
// responsible for logging when the feature vector it passes is incomplete
// relative to the intervention's feature_signature.
func EstimateFromCohort(rows []TrainingRow, patientID string, patientFeatures map[string]float64, band models.OverlapBand) (models.CATEEstimate, error) {
	est := models.CATEEstimate{
		PatientID:   patientID,
		LearnerType: models.LearnerBaselineDiffMeans,
	}

	// Insufficient-data short-circuit.
	if len(rows) < minTrainingN {
		est.OverlapStatus = models.OverlapInsufficientData
		est.TrainingN = len(rows)
		return est, nil
	}

	// Fit cohort propensity on the union of features present in the training rows.
	// Features are z-score normalized using cohort statistics so the logistic
	// regression gradient descent converges to a meaningful intercept (~logit(0.5))
	// rather than drifting to extreme weights when features have large magnitude.
	featureKeys := extractFeatureKeys(rows)
	means, stds := computeFeatureStats(rows, featureKeys)
	X, y := buildNormalizedPropensityMatrix(rows, featureKeys, means, stds)
	prop, err := FitPropensity(X, y, featureKeys)
	if err != nil {
		return est, fmt.Errorf("baseline CATE propensity fit (patientID=%s): %w", patientID, err)
	}
	normalizedPatient := normalizeFeatures(patientFeatures, featureKeys, means, stds)
	p := prop.Predict(normalizedPatient)
	est.Propensity = p
	est.TrainingN = len(rows)
	est.OverlapStatus = EvaluateOverlap(p, band)
	if est.OverlapStatus != models.OverlapPass {
		return est, nil
	}

	// Nearest-neighbour bucket: take the top-half of rows by L1 distance on featureKeys.
	bucketSize := len(rows) / bucketHalfFraction
	if bucketSize < minTrainingN {
		bucketSize = minTrainingN
	}
	bucket := nearestBucket(rows, patientFeatures, featureKeys, bucketSize)

	var treatedOut, controlOut []int
	for _, r := range bucket {
		o := 0
		if r.OutcomeOccurred {
			o = 1
		}
		if r.Treated {
			treatedOut = append(treatedOut, o)
		} else {
			controlOut = append(controlOut, o)
		}
	}
	est.CohortTreatedN = len(treatedOut)
	est.CohortControlN = len(controlOut)

	// Arm-imbalance guard: even if the patient passed the propensity overlap
	// check, the nearest-neighbour bucket may be skewed (e.g., the patient's
	// feature neighbourhood happens to be mostly treated or mostly control).
	// Returning OverlapInsufficientData here is safer than a point estimate
	// backed by 1-2 observations in the minority arm.
	if len(treatedOut) < minBucketArmN || len(controlOut) < minBucketArmN {
		est.OverlapStatus = models.OverlapInsufficientData
		return est, nil
	}

	// Point estimate: control − treated (positive means treatment reduces risk).
	point := meanInt(controlOut) - meanInt(treatedOut)
	lower, upper := bootstrapDiffCI(treatedOut, controlOut, bootstrapResamples, ciLowerPct, ciUpperPct, 42)
	est.PointEstimate = point
	est.CILower = lower
	est.CIUpper = upper

	// Feature contributions: bucket mean − cohort mean per feature, top-K by |delta|.
	est.FeatureContributionKeys, est.FeatureContributionsJSON = computeFeatureContributions(rows, bucket, patientFeatures, featureKeys, topKFeatureContribs)
	return est, nil
}

// meanInt returns arithmetic mean of an int slice; empty → 0.
func meanInt(xs []int) float64 {
	if len(xs) == 0 {
		return 0
	}
	var s int
	for _, x := range xs {
		s += x
	}
	return float64(s) / float64(len(xs))
}

// bootstrapDiffCI resamples treated and control with replacement B times and returns
// the (lowerPct, upperPct) percentiles of the (control-mean − treated-mean) distribution.
func bootstrapDiffCI(treated, control []int, B int, lowerPct, upperPct float64, seed int64) (float64, float64) {
	r := rand.New(rand.NewSource(seed))
	samples := make([]float64, B)
	for b := 0; b < B; b++ {
		t := resampleInt(treated, r)
		c := resampleInt(control, r)
		samples[b] = meanInt(c) - meanInt(t)
	}
	sort.Float64s(samples)
	// Uses "exclusive-lower" percentile convention: for B=500 and lowerPct=5.0,
	// lower = samples[25] (the 26th sorted value). This differs by one index from
	// numpy default but keeps the interval width identical. Sprint 3 explanation
	// layer should prefer `samples[int(lowerPct/100 * (B-1))]` if interval endpoints
	// are compared against probability thresholds.
	lower := samples[int(math.Floor(lowerPct/100*float64(B)))]
	upper := samples[int(math.Floor(upperPct/100*float64(B)))]
	return lower, upper
}

func resampleInt(xs []int, r *rand.Rand) []int {
	out := make([]int, len(xs))
	for i := range out {
		out[i] = xs[r.Intn(len(xs))]
	}
	return out
}

// extractFeatureKeys returns the sorted union of all feature keys across rows.
// Sorting guarantees deterministic propensity model column ordering.
func extractFeatureKeys(rows []TrainingRow) []string {
	seen := map[string]struct{}{}
	var keys []string
	for _, r := range rows {
		for k := range r.Features {
			if _, ok := seen[k]; !ok {
				seen[k] = struct{}{}
				keys = append(keys, k)
			}
		}
	}
	sort.Strings(keys)
	return keys
}

// computeFeatureStats computes per-feature mean and standard deviation across
// all training rows. std is clamped to ≥1e-9 to prevent divide-by-zero for
// constant features.
func computeFeatureStats(rows []TrainingRow, keys []string) (means, stds map[string]float64) {
	means = make(map[string]float64, len(keys))
	stds = make(map[string]float64, len(keys))
	n := float64(len(rows))
	for _, k := range keys {
		var sum float64
		for _, r := range rows {
			sum += r.Features[k]
		}
		means[k] = sum / n
	}
	for _, k := range keys {
		var variance float64
		for _, r := range rows {
			d := r.Features[k] - means[k]
			variance += d * d
		}
		stds[k] = math.Sqrt(variance/n) + 1e-9
	}
	return means, stds
}

// normalizeFeatures returns a copy of features with each key z-scored using
// the provided cohort means and stds. Missing keys default to 0 (cohort mean).
func normalizeFeatures(features map[string]float64, keys []string, means, stds map[string]float64) map[string]float64 {
	out := make(map[string]float64, len(keys))
	for _, k := range keys {
		out[k] = (features[k] - means[k]) / stds[k]
	}
	return out
}

// buildNormalizedPropensityMatrix projects rows onto featureKeys after z-score
// normalization using the provided cohort means and stds.
func buildNormalizedPropensityMatrix(rows []TrainingRow, keys []string, means, stds map[string]float64) ([][]float64, []bool) {
	X := make([][]float64, len(rows))
	y := make([]bool, len(rows))
	for i, r := range rows {
		X[i] = make([]float64, len(keys))
		for j, k := range keys {
			X[i][j] = (r.Features[k] - means[k]) / stds[k]
		}
		y[i] = r.Treated
	}
	return X, y
}

// nearestBucket returns the k rows nearest (L1) to the target feature vector.
func nearestBucket(rows []TrainingRow, target map[string]float64, keys []string, k int) []TrainingRow {
	type scored struct {
		row  TrainingRow
		dist float64
	}
	out := make([]scored, len(rows))
	for i, r := range rows {
		var d float64
		for _, key := range keys {
			d += math.Abs(r.Features[key] - target[key])
		}
		out[i] = scored{r, d}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].dist < out[j].dist })
	if k > len(out) {
		k = len(out)
	}
	bucket := make([]TrainingRow, k)
	for i := 0; i < k; i++ {
		bucket[i] = out[i].row
	}
	return bucket
}

// computeFeatureContributions returns the top-K signed deltas of (bucket_mean −
// cohort_mean) per feature, plus the JSON-serialised full FeatureContribution
// list (used by Task 7 to persist in the CATEEstimate row).
func computeFeatureContributions(all, bucket []TrainingRow, patient map[string]float64, keys []string, topK int) ([]string, string) {
	type kd struct {
		key   string
		delta float64
		pv    float64
		cm    float64
	}
	deltas := make([]kd, 0, len(keys))
	for _, k := range keys {
		var cohortSum, bucketSum float64
		for _, r := range all {
			cohortSum += r.Features[k]
		}
		for _, r := range bucket {
			bucketSum += r.Features[k]
		}
		cohortMean := cohortSum / float64(len(all))
		bucketMean := bucketSum / float64(len(bucket))
		deltas = append(deltas, kd{k, bucketMean - cohortMean, patient[k], cohortMean})
	}
	sort.Slice(deltas, func(i, j int) bool { return math.Abs(deltas[i].delta) > math.Abs(deltas[j].delta) })
	if topK > len(deltas) {
		topK = len(deltas)
	}
	keysOut := make([]string, topK)
	contribs := make([]models.FeatureContribution, topK)
	for i := 0; i < topK; i++ {
		keysOut[i] = deltas[i].key
		contribs[i] = models.FeatureContribution{
			FeatureKey:   deltas[i].key,
			Contribution: deltas[i].delta,
			PatientValue: deltas[i].pv,
			CohortMean:   deltas[i].cm,
		}
	}
	// FeatureContribution fields are all string/float64; json.Marshal cannot fail.
	payload, _ := json.Marshal(contribs)
	return keysOut, string(payload)
}
