package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/models"
)

// CacheClient is a minimal interface for the DataResolver's cache access.
// Implemented by the real Redis-backed cache and by in-memory mocks in tests.
type CacheClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
}

// DataResolver is the central data-fetching abstraction used by both the
// MonitoringNodeEngine (PM-*) and the DeteriorationNodeEngine (MD-*).
//
// It groups required inputs by source (KB-20, KB-26, DEVICE, TIER1_CHECKIN),
// fetches them, computes statistical aggregates over time-series data, and
// returns a ResolvedData struct with DataSufficiency judgment.
type DataResolver interface {
	Resolve(ctx context.Context, patientID string, inputs []models.RequiredInput, aggInputs []models.AggregatedInputDef) (*models.ResolvedData, error)
}

// DataResolverImpl is the production DataResolver.
type DataResolverImpl struct {
	kb20BaseURL        string
	kb20Client         *http.Client
	kb26Client         *KB26Client
	cache              CacheClient
	stalenessThreshold time.Duration
	log                *zap.Logger
}

// NewDataResolver constructs a DataResolverImpl.
//
//   - kb20BaseURL: base URL of KB-20 Patient Profile service.
//   - kb26Client:  pre-constructed KB26Client.
//   - cache:       any CacheClient (Redis or in-memory mock for tests).
//   - stalenessThreshold: if KB-26 twin state is older than this, SUFFICIENT → PARTIAL.
//   - log:         structured logger.
func NewDataResolver(
	kb20BaseURL string,
	kb26Client *KB26Client,
	cache CacheClient,
	stalenessThreshold time.Duration,
	log *zap.Logger,
) *DataResolverImpl {
	return &DataResolverImpl{
		kb20BaseURL: kb20BaseURL,
		kb20Client: &http.Client{
			Timeout: 10 * time.Second,
		},
		kb26Client:         kb26Client,
		cache:              cache,
		stalenessThreshold: stalenessThreshold,
		log:                log.With(zap.String("component", "data-resolver")),
	}
}

// Resolve fetches all required data for a set of inputs and aggregated inputs,
// applies aggregations, evaluates sufficiency, and returns a ResolvedData.
//
// Algorithm:
//  1. Group inputs by source.
//  2. Fetch KB-20 scalars (GET /api/v1/patient/{id}/labs?type={}&days={}).
//  3. Fetch KB-26 twin state once (lazily), check staleness.
//  4. Map each RequiredInput to a resolved field or MissingFields.
//  5. For each AggregatedInputDef, fetch the history series and compute aggregate.
//  6. Compute final DataSufficiency.
func (r *DataResolverImpl) Resolve(
	ctx context.Context,
	patientID string,
	inputs []models.RequiredInput,
	aggInputs []models.AggregatedInputDef,
) (*models.ResolvedData, error) {

	resolved := &models.ResolvedData{
		Fields:          make(map[string]float64),
		TimeSeries:      make(map[string][]models.TimeSeriesPoint),
		FieldTimestamps: make(map[string]time.Time),
		Sources:         make(map[string]string),
		Sufficiency:     models.DataSufficient,
	}

	// Determine which sources are needed so we avoid unnecessary fetches.
	needsKB26 := false
	for _, inp := range inputs {
		if inp.Source == "KB-26" {
			needsKB26 = true
		}
	}
	for _, agg := range aggInputs {
		if agg.Source == "KB-26" {
			needsKB26 = true
		}
	}

	// Fetch KB-26 twin state once (if needed).
	var twinState *models.TwinStateView
	twinStale := false
	if needsKB26 {
		var err error
		twinState, err = r.kb26Client.GetTwinState(ctx, patientID)
		if err != nil {
			r.log.Warn("kb26 twin state fetch failed", zap.String("patient_id", patientID), zap.Error(err))
			// We'll mark required KB-26 fields as missing below
		} else {
			// Check staleness
			age := time.Since(twinState.LastUpdated)
			if age > r.stalenessThreshold {
				twinStale = true
				r.log.Info("kb26 twin state stale",
					zap.String("patient_id", patientID),
					zap.Duration("age", age),
					zap.Duration("threshold", r.stalenessThreshold),
				)
			}
		}
	}

	// -----------------------------------------------------------------------
	// Process RequiredInputs
	// -----------------------------------------------------------------------
	hasMissingRequired := false
	hasMissingOptional := false

	for _, inp := range inputs {
		switch inp.Source {
		case "KB-20":
			val, ts, err := r.fetchKB20Lab(ctx, patientID, inp.Field, inp.LookbackDays)
			if err != nil {
				r.log.Debug("kb20 field not found",
					zap.String("patient_id", patientID),
					zap.String("field", inp.Field),
					zap.Error(err),
				)
				resolved.MissingFields = append(resolved.MissingFields, inp.Field)
				if inp.Optional {
					hasMissingOptional = true
				} else {
					hasMissingRequired = true
				}
				continue
			}
			resolved.Fields[inp.Field] = val
			resolved.FieldTimestamps[inp.Field] = ts
			resolved.Sources[inp.Field] = "KB-20"

		case "KB-26":
			if twinState == nil {
				resolved.MissingFields = append(resolved.MissingFields, inp.Field)
				if inp.Optional {
					hasMissingOptional = true
				} else {
					hasMissingRequired = true
				}
				continue
			}
			val, ok := extractTwinStateField(twinState, inp.Field)
			if !ok {
				resolved.MissingFields = append(resolved.MissingFields, inp.Field)
				if inp.Optional {
					hasMissingOptional = true
				} else {
					hasMissingRequired = true
				}
				continue
			}
			resolved.Fields[inp.Field] = val
			resolved.FieldTimestamps[inp.Field] = twinState.LastUpdated
			resolved.Sources[inp.Field] = "KB-26"

		case "TIER1_CHECKIN":
			// TIER1_CHECKIN values are patient self-reports delivered by the
			// check-in subsystem. At resolution time they may not be available.
			// When absent, they are treated as optional.
			resolved.MissingFields = append(resolved.MissingFields, inp.Field)
			if inp.Optional {
				hasMissingOptional = true
			} else {
				// Required checkin not provided → missing optional semantics
				// (cannot block clinical evaluation on patient self-report)
				hasMissingOptional = true
			}

		case "DEVICE":
			// DEVICE data is fetched from KB-20 using the same lab endpoint
			// (device readings are stored as observations in KB-20).
			val, ts, err := r.fetchKB20Lab(ctx, patientID, inp.Field, inp.LookbackDays)
			if err != nil {
				resolved.MissingFields = append(resolved.MissingFields, inp.Field)
				if inp.Optional {
					hasMissingOptional = true
				} else {
					hasMissingRequired = true
				}
				continue
			}
			resolved.Fields[inp.Field] = val
			resolved.FieldTimestamps[inp.Field] = ts
			resolved.Sources[inp.Field] = "DEVICE"

		default:
			r.log.Warn("unknown input source",
				zap.String("field", inp.Field),
				zap.String("source", inp.Source),
			)
			resolved.MissingFields = append(resolved.MissingFields, inp.Field)
			if inp.Optional {
				hasMissingOptional = true
			} else {
				hasMissingRequired = true
			}
		}
	}

	// -----------------------------------------------------------------------
	// Process AggregatedInputDefs
	// -----------------------------------------------------------------------
	for _, agg := range aggInputs {
		series, err := r.fetchTimeSeries(ctx, patientID, agg.Source, agg.Field, agg.LookbackDays, twinState)
		if err != nil || len(series) == 0 {
			r.log.Debug("aggregated input: series unavailable",
				zap.String("patient_id", patientID),
				zap.String("field", agg.Field),
				zap.Error(err),
			)
			resolved.MissingFields = append(resolved.MissingFields, agg.Field)
			if agg.Optional {
				hasMissingOptional = true
			} else {
				hasMissingOptional = true // aggregated inputs: treat as optional
			}
			continue
		}

		// Extract raw float values for aggregation
		vals := make([]float64, len(series))
		for i, pt := range series {
			vals[i] = pt.Value
		}

		switch strings.ToUpper(agg.Aggregation) {
		case "MEAN":
			resolved.Fields[agg.Field] = computeMean(vals)
			resolved.Sources[agg.Field] = agg.Source

		case "STDEV":
			resolved.Fields[agg.Field] = computeStdev(vals)
			resolved.Sources[agg.Field] = agg.Source

		case "CV":
			mean := computeMean(vals)
			stdev := computeStdev(vals)
			if mean == 0 {
				resolved.Fields[agg.Field] = 0
			} else {
				resolved.Fields[agg.Field] = (stdev / mean) * 100.0
			}
			resolved.Sources[agg.Field] = agg.Source

		case "COUNT":
			resolved.Fields[agg.Field] = float64(len(vals))
			resolved.Sources[agg.Field] = agg.Source

		case "MAX":
			resolved.Fields[agg.Field] = computeMax(vals)
			resolved.Sources[agg.Field] = agg.Source

		case "MIN":
			resolved.Fields[agg.Field] = computeMin(vals)
			resolved.Sources[agg.Field] = agg.Source

		case "RAW":
			// Store entire series in TimeSeries for TrajectoryComputer
			resolved.TimeSeries[agg.Field] = series
			resolved.Sources[agg.Field] = agg.Source

		default:
			r.log.Warn("unknown aggregation type",
				zap.String("field", agg.Field),
				zap.String("aggregation", agg.Aggregation),
			)
		}
	}

	// -----------------------------------------------------------------------
	// Compute DataSufficiency
	// -----------------------------------------------------------------------
	switch {
	case hasMissingRequired:
		resolved.Sufficiency = models.DataInsufficient
	case twinStale:
		// Stale twin state always downgrades from SUFFICIENT → PARTIAL
		resolved.Sufficiency = models.DataPartial
	case hasMissingOptional:
		resolved.Sufficiency = models.DataPartial
	default:
		resolved.Sufficiency = models.DataSufficient
	}

	r.log.Debug("resolve complete",
		zap.String("patient_id", patientID),
		zap.String("sufficiency", string(resolved.Sufficiency)),
		zap.Int("fields_resolved", len(resolved.Fields)),
		zap.Int("missing_fields", len(resolved.MissingFields)),
	)

	return resolved, nil
}

// ---------------------------------------------------------------------------
// KB-20 fetch helpers
// ---------------------------------------------------------------------------

// kb20LabResult is the JSON body returned by KB-20's GET /labs endpoint.
type kb20LabResult struct {
	Value     float64 `json:"value"`
	Timestamp string  `json:"timestamp"`
	Unit      string  `json:"unit"`
}

// fetchKB20Lab performs GET {kb20BaseURL}/api/v1/patient/{id}/labs?type={labType}&days={days}
// and returns the scalar value and its timestamp.
func (r *DataResolverImpl) fetchKB20Lab(ctx context.Context, patientID, labType string, lookbackDays int) (float64, time.Time, error) {
	if lookbackDays <= 0 {
		lookbackDays = 90
	}
	url := fmt.Sprintf("%s/api/v1/patient/%s/labs?type=%s&days=%d",
		r.kb20BaseURL, patientID, labType, lookbackDays)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("kb20 build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := r.kb20Client.Do(req)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("kb20 fetch %s: %w", labType, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, time.Time{}, fmt.Errorf("kb20 fetch %s: status %d: %s", labType, resp.StatusCode, truncateBody(string(body), 128))
	}

	var result kb20LabResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, time.Time{}, fmt.Errorf("kb20 decode %s: %w", labType, err)
	}

	ts := time.Now()
	if result.Timestamp != "" {
		if parsed, err := time.Parse(time.RFC3339, result.Timestamp); err == nil {
			ts = parsed
		}
	}

	return result.Value, ts, nil
}

// kb20HistoryPoint is one point from the KB-20 history endpoint.
type kb20HistoryPoint struct {
	Value     float64 `json:"value"`
	Timestamp string  `json:"timestamp"`
}

// fetchKB20History performs GET {kb20BaseURL}/api/v1/patient/{id}/labs/history?type={}&days={}
// and returns a time-series slice.
func (r *DataResolverImpl) fetchKB20History(ctx context.Context, patientID, labType string, lookbackDays int) ([]models.TimeSeriesPoint, error) {
	if lookbackDays <= 0 {
		lookbackDays = 90
	}
	url := fmt.Sprintf("%s/api/v1/patient/%s/labs/history?type=%s&days=%d",
		r.kb20BaseURL, patientID, labType, lookbackDays)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("kb20 history build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := r.kb20Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("kb20 history fetch %s: %w", labType, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("kb20 history fetch %s: status %d: %s", labType, resp.StatusCode, truncateBody(string(body), 128))
	}

	var points []kb20HistoryPoint
	if err := json.NewDecoder(resp.Body).Decode(&points); err != nil {
		return nil, fmt.Errorf("kb20 history decode %s: %w", labType, err)
	}

	series := make([]models.TimeSeriesPoint, 0, len(points))
	for _, p := range points {
		ts := time.Now()
		if p.Timestamp != "" {
			if parsed, err := time.Parse(time.RFC3339, p.Timestamp); err == nil {
				ts = parsed
			}
		}
		series = append(series, models.TimeSeriesPoint{
			Timestamp: ts,
			Value:     p.Value,
		})
	}
	return series, nil
}

// fetchTimeSeries fetches a time series from the appropriate source for aggregation.
func (r *DataResolverImpl) fetchTimeSeries(
	ctx context.Context,
	patientID, source, field string, lookbackDays int,
	twin *models.TwinStateView,
) ([]models.TimeSeriesPoint, error) {
	switch source {
	case "KB-20", "DEVICE":
		return r.fetchKB20History(ctx, patientID, field, lookbackDays)

	case "KB-26":
		if twin == nil {
			return nil, fmt.Errorf("kb26 twin state unavailable for history of %s", field)
		}
		return r.kb26Client.GetVariableHistory(ctx, patientID, field, lookbackDays)

	default:
		return nil, fmt.Errorf("unsupported source for aggregation: %s", source)
	}
}

// ---------------------------------------------------------------------------
// KB-26 field extraction helper
// ---------------------------------------------------------------------------

// extractTwinStateField maps a field name to the corresponding TwinStateView value.
// Returns (value, true) when the field exists and is non-zero, (0, false) otherwise.
func extractTwinStateField(ts *models.TwinStateView, field string) (float64, bool) {
	switch field {
	// Tier 3 estimated variables
	case "IS":
		return ts.IS.Value, ts.IS.Value != 0 || ts.IS.Confidence != 0
	case "HGO":
		return ts.HGO.Value, ts.HGO.Value != 0 || ts.HGO.Confidence != 0
	case "MM":
		return ts.MM.Value, ts.MM.Value != 0 || ts.MM.Confidence != 0

	// Tier 2
	case "VF":
		return ts.VF, true // VF=0 is a valid value
	case "VR":
		return ts.VR.Value, ts.VR.Value != 0 || ts.VR.Confidence != 0
	case "RR":
		return ts.RR.Value, ts.RR.Value != 0 || ts.RR.Confidence != 0

	case "renal_slope", "RenalSlope":
		return ts.RenalSlope, true

	case "glycemic_var", "GlycemicVar":
		return ts.GlycemicVar, true

	// Tier 1
	case "eGFR", "EGFR", "egfr":
		if ts.EGFR != nil {
			return *ts.EGFR, true
		}
		return 0, false

	case "daily_steps", "DailySteps":
		if ts.DailySteps != nil {
			return *ts.DailySteps, true
		}
		return 0, false

	case "resting_hr", "RestingHR":
		if ts.RestingHR != nil {
			return *ts.RestingHR, true
		}
		return 0, false
	}

	return 0, false
}

// ---------------------------------------------------------------------------
// Statistical aggregation helpers
// ---------------------------------------------------------------------------

// computeMean returns the arithmetic mean of vals.
// Returns 0 for empty slices.
func computeMean(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum / float64(len(vals))
}

// computeStdev returns the sample standard deviation of vals.
// Returns 0 for slices with fewer than 2 elements.
func computeStdev(vals []float64) float64 {
	n := float64(len(vals))
	if n < 2 {
		return 0
	}
	mean := computeMean(vals)
	variance := 0.0
	for _, v := range vals {
		d := v - mean
		variance += d * d
	}
	variance /= (n - 1) // sample variance
	return math.Sqrt(variance)
}

// computeMax returns the maximum value from vals.
// Returns 0 for empty slices.
func computeMax(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	max := vals[0]
	for _, v := range vals[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

// computeMin returns the minimum value from vals.
// Returns 0 for empty slices.
func computeMin(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	min := vals[0]
	for _, v := range vals[1:] {
		if v < min {
			min = v
		}
	}
	return min
}
