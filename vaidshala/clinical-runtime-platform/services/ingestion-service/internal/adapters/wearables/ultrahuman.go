package wearables

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

// CGM glucose thresholds in mg/dL for time-in-range classification.
const (
	CGMLowThreshold  = 70.0
	CGMHighThreshold = 180.0
	CGMCVTarget      = 36.0
)

// UltrahumanCGMPayload is the top-level envelope for a batch of CGM
// readings from an Ultrahuman M1/Ring device.
type UltrahumanCGMPayload struct {
	PatientID   uuid.UUID    `json:"patient_id"`
	TenantID    uuid.UUID    `json:"tenant_id"`
	DeviceID    string       `json:"device_id"`
	SensorID    string       `json:"sensor_id"`
	Readings    []CGMReading `json:"readings"`
	PeriodStart time.Time    `json:"period_start"`
	PeriodEnd   time.Time    `json:"period_end"`
}

// CGMReading is a single interstitial glucose measurement.
type CGMReading struct {
	Timestamp   time.Time `json:"timestamp"`
	GlucoseMgDL float64   `json:"glucose_mg_dl"`
}

// CGMAggregation holds computed CGM metrics over a reporting period.
type CGMAggregation struct {
	TIR          float64 `json:"tir"`
	TAR          float64 `json:"tar"`
	TBR          float64 `json:"tbr"`
	CV           float64 `json:"cv"`
	MAG          float64 `json:"mag"`
	MeanGlucose  float64 `json:"mean_glucose"`
	GMI          float64 `json:"gmi"`
	ReadingCount int     `json:"reading_count"`
}

// LOINC codes for CGM-derived observations.
var cgmLOINCMap = map[string]string{
	"TIR":         "97507-8",
	"TAR":         "97506-0",
	"TBR":         "97505-2",
	"CV":          "97504-5",
	"MAG":         "97503-7",
	"MeanGlucose": "2339-0",
	"GMI":         "97502-9",
}

// UltrahumanAdapter converts Ultrahuman CGM payloads into canonical
// observations.
type UltrahumanAdapter struct{}

// Convert transforms an UltrahumanCGMPayload into a slice of 7
// canonical observations representing the aggregated CGM metrics.
// A minimum of 12 readings is required.
func (a *UltrahumanAdapter) Convert(payload UltrahumanCGMPayload) ([]canonical.CanonicalObservation, error) {
	if len(payload.Readings) < 12 {
		return nil, fmt.Errorf("insufficient CGM readings: got %d, need at least 12", len(payload.Readings))
	}

	agg := AggregateCGM(payload.Readings)
	quality := qualityFromReadingCount(agg.ReadingCount)

	type metric struct {
		name  string
		value float64
		unit  string
	}

	metrics := []metric{
		{"TIR", agg.TIR, "%"},
		{"TAR", agg.TAR, "%"},
		{"TBR", agg.TBR, "%"},
		{"CV", agg.CV, "%"},
		{"MAG", agg.MAG, "mg/dL/h"},
		{"MeanGlucose", agg.MeanGlucose, "mg/dL"},
		{"GMI", agg.GMI, "%"},
	}

	observations := make([]canonical.CanonicalObservation, 0, len(metrics))

	for _, m := range metrics {
		var flags []canonical.Flag

		// Critical flag when TBR > 4% (hypoglycemia risk).
		if m.name == "TBR" && m.value > 4.0 {
			flags = append(flags, canonical.FlagCriticalValue)
		}

		// Low quality flag when CV > 36% (high glycemic variability).
		if m.name == "CV" && m.value > CGMCVTarget {
			flags = append(flags, canonical.FlagLowQuality)
		}

		obs := canonical.CanonicalObservation{
			ID:              uuid.New(),
			PatientID:       payload.PatientID,
			TenantID:        payload.TenantID,
			SourceType:      canonical.SourceWearable,
			SourceID:        fmt.Sprintf("ultrahuman:%s:%s:%s", payload.DeviceID, payload.SensorID, m.name),
			ObservationType: canonical.ObsDeviceData,
			LOINCCode:       cgmLOINCMap[m.name],
			Value:           m.value,
			Unit:            m.unit,
			Timestamp:       payload.PeriodEnd,
			QualityScore:    quality,
			Flags:           flags,
			DeviceContext: &canonical.DeviceContext{
				DeviceID:     payload.DeviceID,
				DeviceType:   "cgm",
				Manufacturer: "Ultrahuman",
				Model:        "M1",
			},
		}

		observations = append(observations, obs)
	}

	return observations, nil
}

// AggregateCGM computes summary CGM metrics from a slice of readings.
// Readings are sorted by timestamp before computation.
func AggregateCGM(readings []CGMReading) CGMAggregation {
	sort.Slice(readings, func(i, j int) bool {
		return readings[i].Timestamp.Before(readings[j].Timestamp)
	})

	n := len(readings)
	var sum float64
	var inRange, aboveRange, belowRange int

	for _, r := range readings {
		sum += r.GlucoseMgDL
		switch {
		case r.GlucoseMgDL < CGMLowThreshold:
			belowRange++
		case r.GlucoseMgDL > CGMHighThreshold:
			aboveRange++
		default:
			inRange++
		}
	}

	mean := sum / float64(n)

	// Coefficient of variation: CV = (stddev / mean) * 100
	var sumSqDiff float64
	for _, r := range readings {
		diff := r.GlucoseMgDL - mean
		sumSqDiff += diff * diff
	}
	stddev := math.Sqrt(sumSqDiff / float64(n))
	cv := (stddev / mean) * 100.0

	// Mean Amplitude of Glycemic Excursions: sum(|delta|) / total hours
	var sumAbsChange float64
	for i := 1; i < n; i++ {
		sumAbsChange += math.Abs(readings[i].GlucoseMgDL - readings[i-1].GlucoseMgDL)
	}
	totalHours := readings[n-1].Timestamp.Sub(readings[0].Timestamp).Hours()
	var mag float64
	if totalHours > 0 {
		mag = sumAbsChange / totalHours
	}

	// Glucose Management Indicator: GMI = 3.31 + 0.02392 * mean
	gmi := 3.31 + 0.02392*mean

	return CGMAggregation{
		TIR:          roundTo2(float64(inRange) / float64(n) * 100.0),
		TAR:          roundTo2(float64(aboveRange) / float64(n) * 100.0),
		TBR:          roundTo2(float64(belowRange) / float64(n) * 100.0),
		CV:           roundTo2(cv),
		MAG:          roundTo2(mag),
		MeanGlucose:  roundTo2(mean),
		GMI:          roundTo2(gmi),
		ReadingCount: n,
	}
}

// roundTo2 rounds a float64 to two decimal places.
func roundTo2(v float64) float64 {
	return math.Round(v*100) / 100
}

// qualityFromReadingCount maps the number of CGM readings to a quality
// score. 288 readings (5-minute intervals over 24h) is the gold standard.
func qualityFromReadingCount(count int) float64 {
	switch {
	case count >= 288:
		return 0.95
	case count >= 200:
		return 0.85
	case count >= 100:
		return 0.75
	default:
		return 0.60
	}
}
