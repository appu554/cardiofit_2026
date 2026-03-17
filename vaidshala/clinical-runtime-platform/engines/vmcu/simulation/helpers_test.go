package simulation

import (
	"time"

	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_b"
)

// freshRawLabs returns a *channel_b.RawPatientData with all timestamp fields
// set to time.Now() and sensible clinical defaults. This prevents DA-06/DA-07/DA-08
// staleness rules from firing during tests.
func freshRawLabs() *channel_b.RawPatientData {
	now := time.Now()
	return &channel_b.RawPatientData{
		GlucoseCurrent:    f64(7.0),
		GlucoseTimestamp:  now.Add(-30 * time.Minute),
		CreatinineCurrent: f64(85.0),
		PotassiumCurrent:  f64(4.2),
		SBPCurrent:        f64(130.0),
		WeightKgCurrent:   f64(80.0),
		EGFRCurrent:       f64(70.0),
		HbA1cCurrent:      f64(7.5),

		Creatinine48hAgo: f64(83.0),
		EGFRPrior48h:     f64(70.0),
		HbA1cPrior30d:    f64(7.5),
		Weight72hAgo:     f64(80.0),

		EGFRLastMeasuredAt:       timePtr(now.Add(-12 * time.Hour)),
		HbA1cLastMeasuredAt:      timePtr(now.Add(-24 * time.Hour)),
		CreatinineLastMeasuredAt: timePtr(now.Add(-6 * time.Hour)),
	}
}

// timePtr returns a pointer to the given time.Time value.
func timePtr(t time.Time) *time.Time { return &t }

// freshLabsWithOverrides returns freshRawLabs() with caller-specified overrides
// applied via a mutation function.
func freshLabsWithOverrides(mutate func(d *channel_b.RawPatientData)) *channel_b.RawPatientData {
	d := freshRawLabs()
	mutate(d)
	return d
}
