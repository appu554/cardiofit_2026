package delta

import (
	"context"
	"errors"
	"math"
	"sort"

	"github.com/google/uuid"
)

// BaselineConfidence enumerates the confidence tiers a persisted baseline can
// carry. The semantics follow Layer 2 doc §2.2: the tier is derived from
// sample size (n) and the inter-quartile range (IQR) relative to the median.
//
//   high             n >= 7  AND  IQR < 0.25 * median
//   medium           n >= 4  AND  IQR < 0.50 * median
//   low              n >= 3                      (otherwise)
//   insufficient_data n < 3                      (no reliable baseline)
//
// Stored verbatim in baseline_state.confidence; the Postgres CHECK constraint
// enforces this set as the source of truth.
type BaselineConfidence string

const (
	BaselineConfidenceHigh             BaselineConfidence = "high"
	BaselineConfidenceMedium           BaselineConfidence = "medium"
	BaselineConfidenceLow              BaselineConfidence = "low"
	BaselineConfidenceInsufficientData BaselineConfidence = "insufficient_data"
)

// MinSamplesForBaseline is the minimum n below which a baseline is treated as
// insufficient_data. Three observations is the floor at which a median + IQR
// can be computed at all (per spec). Callers below this threshold persist a
// row with baseline_value=NULL and confidence=insufficient_data so downstream
// reads see ErrNoBaseline rather than a misleading point estimate.
const MinSamplesForBaseline = 3

// DefaultBaselineLookbackDays is the rolling window used to bound which
// observations contribute to the running baseline. Layer 2 doc §2.2 calls
// for a 14-day window for vitals; callers may override per-vital via the
// RecomputeAndUpsert lookbackDays parameter.
const DefaultBaselineLookbackDays = 14

// BaselineStateStore is the persistence contract that backs
// PersistentBaselineProvider. Implementations live close to the database
// (kb-20's internal/storage.BaselineStore today; eventually a kb-26 client
// once the AcuteRepository goes live). The delta package is persistence-
// agnostic: it owns the algorithm and the Provider façade but never speaks
// SQL itself.
//
// Get returns ErrNoBaseline when no row exists for (residentID, vitalTypeKey)
// or when the persisted row carries confidence=insufficient_data. Other
// errors (decode/network) are propagated as-is.
//
// Upsert writes the supplied Baseline as the new running state. It is a pure
// "replace by primary key" operation; callers that need to recompute from
// observation history should use RecomputeAndUpsert instead.
//
// RecomputeAndUpsert recomputes the median + IQR + confidence tier from the
// underlying observations table over the supplied lookback window and
// persists the result. The returned Baseline reflects the freshly-persisted
// row. Implementations that participate in transactional observation writes
// also expose a Tx-variant; PersistentBaselineProvider does not depend on
// that variant directly because the transactional path is mediated by the
// kb-20 V2SubstrateStore.
type BaselineStateStore interface {
	Get(ctx context.Context, residentID uuid.UUID, vitalTypeKey string) (*Baseline, error)
	Upsert(ctx context.Context, residentID uuid.UUID, vitalTypeKey string, b Baseline) error
	RecomputeAndUpsert(ctx context.Context, residentID uuid.UUID, vitalTypeKey string, lookbackDays int) (*Baseline, error)
}

// PersistentBaselineProvider is the production-shaped delta.BaselineProvider:
// it serves baselines from a persistent BaselineStateStore (Postgres-backed
// in kb-20) instead of an in-memory map. State survives process restart, and
// every Observation insert refreshes the row inside the same transaction so
// the baseline a future ComputeDelta sees is exactly the running median over
// the persisted observation history.
//
// The provider deliberately does not own the recompute path; that lives on
// V2SubstrateStore.UpsertObservation so the recompute participates in the
// same transaction as the observation write. This provider is the read-side
// of that loop.
type PersistentBaselineProvider struct {
	store     BaselineStateStore
	cfgStore  BaselineConfigStore // optional; consulted by the recompute path when wired
}

// NewPersistentBaselineProvider wires a BaselineStateStore into the
// delta.BaselineProvider façade. The store is captured by reference; callers
// own its lifecycle.
func NewPersistentBaselineProvider(store BaselineStateStore) *PersistentBaselineProvider {
	return &PersistentBaselineProvider{store: store}
}

// WithConfigStore attaches a BaselineConfigStore to the provider. When
// set, callers (notably the kb-20 BaselineStore.RecomputeAndUpsertTx
// path) can resolve per-observation-type parameters via
// ResolveConfig. The read path (FetchBaseline) is unchanged: it still
// returns the persisted Baseline as-is, because the config governs
// writes/recomputes, not reads.
//
// Returns the receiver for fluent wiring.
func (p *PersistentBaselineProvider) WithConfigStore(cfg BaselineConfigStore) *PersistentBaselineProvider {
	p.cfgStore = cfg
	return p
}

// ConfigStore returns the wired BaselineConfigStore (or nil). Callers
// that need to read/list configs should go through this accessor rather
// than touching the field directly.
func (p *PersistentBaselineProvider) ConfigStore() BaselineConfigStore {
	return p.cfgStore
}

// ResolveConfig returns the BaselineConfig for observationType. When no
// ConfigStore is wired, or when the store has no row for the type, the
// fallback DefaultConfig(observationType) is returned. Errors other
// than ErrBaselineConfigNotFound are propagated.
//
// This is the single canonical lookup path; callers MUST NOT reach into
// the store directly when they want fallback semantics.
func ResolveConfig(ctx context.Context, store BaselineConfigStore, observationType string) (BaselineConfig, error) {
	if store == nil {
		return DefaultConfig(observationType), nil
	}
	c, err := store.Get(ctx, observationType)
	if err != nil {
		if errors.Is(err, ErrBaselineConfigNotFound) {
			return DefaultConfig(observationType), nil
		}
		return BaselineConfig{}, err
	}
	return *c, nil
}

// FetchBaseline implements BaselineProvider. It reads the cached
// baseline_state row for (residentID, vitalType) and returns it. Returns
// ErrNoBaseline when the underlying store has no row, when the row's
// SampleSize is below MinSamplesForBaseline (insufficient_data), or when
// StdDev is zero (degenerate baseline that ComputeDelta would have to flag
// as no_baseline anyway — surface the sentinel here so callers don't
// allocate a zero Delta they'll immediately discard).
func (p *PersistentBaselineProvider) FetchBaseline(ctx context.Context, residentID uuid.UUID, vitalType string) (*Baseline, error) {
	bl, err := p.store.Get(ctx, residentID, vitalType)
	if err != nil {
		return nil, err
	}
	if bl == nil || bl.SampleSize < MinSamplesForBaseline || bl.StdDev == 0 {
		return nil, ErrNoBaseline
	}
	return bl, nil
}

// ClassifyBaselineConfidence maps a sample-size + IQR + median triple to the
// canonical BaselineConfidence tier per Layer 2 doc §2.2. Pure function;
// shared between the persistent recompute path and unit tests so the
// classifier semantics live in exactly one place.
//
// median is taken as the absolute value to handle vitals that can be
// negative-valued in principle (none today, but the divisor must not flip
// sign of the threshold check). Zero or near-zero median falls back to the
// low/insufficient tier because the relative IQR test is undefined.
func ClassifyBaselineConfidence(n int, iqr, median float64) BaselineConfidence {
	if n < MinSamplesForBaseline {
		return BaselineConfidenceInsufficientData
	}
	absMedian := math.Abs(median)
	switch {
	case n >= 7 && absMedian > 0 && iqr < 0.25*absMedian:
		return BaselineConfidenceHigh
	case n >= 4 && absMedian > 0 && iqr < 0.50*absMedian:
		return BaselineConfidenceMedium
	default:
		return BaselineConfidenceLow
	}
}

// Percentiles returns the requested percentiles of values using standard
// linear interpolation between order statistics (NIST/Excel "exclusive"
// definition: position = p * (n - 1) for p in [0, 1]). Values are sorted
// in-place; callers that need to preserve input order should pass a copy.
//
// Returns NaN entries for percentiles requested when len(values) == 0.
//
// Why a hand-rolled helper rather than a stats library: the persistent
// baseline path runs inside the observation-insert critical section. A
// dependency on a third-party stats package would pull a heavier import
// graph than the (well-tested, single-purpose) sort + interp pair below.
//
// Reference: for [1,2,3,4,5,6,7,8,9,10] and ps=[0.25,0.5,0.75] this returns
// [3.25, 5.5, 7.75], matching the textbook linear-interpolation result.
func Percentiles(values []float64, ps ...float64) []float64 {
	out := make([]float64, len(ps))
	if len(values) == 0 {
		for i := range out {
			out[i] = math.NaN()
		}
		return out
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)
	n := len(sorted)
	for i, p := range ps {
		if p <= 0 {
			out[i] = sorted[0]
			continue
		}
		if p >= 1 {
			out[i] = sorted[n-1]
			continue
		}
		pos := p * float64(n-1)
		lo := int(math.Floor(pos))
		hi := int(math.Ceil(pos))
		if lo == hi {
			out[i] = sorted[lo]
			continue
		}
		frac := pos - float64(lo)
		out[i] = sorted[lo] + frac*(sorted[hi]-sorted[lo])
	}
	return out
}

// Compile-time interface assertion.
var _ BaselineProvider = (*PersistentBaselineProvider)(nil)
