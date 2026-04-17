package services

import (
	"strings"
	"time"

	"kb-patient-profile/internal/models"
)

// ─── types ───────────────────────────────────────────────────────────────────

// PatientClinicalEvent represents a discrete clinical event in the patient timeline.
type PatientClinicalEvent struct {
	Type     string    `json:"type"`                    // MEDICATION_START, MEDICATION_STOP, HOSPITALIZATION, LAB_RESULT
	DrugName string    `json:"drug_name,omitempty"`
	LabType  string    `json:"lab_type,omitempty"`
	Value    float64   `json:"value,omitempty"`
	Date     time.Time `json:"date"`
	Duration int       `json:"duration_days,omitempty"`
}

// ConfounderWeights holds configurable weights and washout periods for each
// clinical event confounder type.
type ConfounderWeights struct {
	SteroidWeight        float64
	SteroidWashoutDays   int
	HospWeight           float64
	HospWashoutDays      int
	InfectionWeight      float64
	InfectionWashoutDays int
	AKIWeight            float64
	AKIWashoutDays       int
	SurgeryWeight        float64
	SurgeryWashoutDays   int
}

// ClinicalEventDetector scans patient clinical events and returns confounder
// factors that overlap a given outcome window.
type ClinicalEventDetector struct {
	weights *ConfounderWeights
}

// ─── drug pattern lists ──────────────────────────────────────────────────────

var steroidPatterns = []string{
	"prednisolone", "prednisone", "dexamethasone", "methylprednisolone",
	"hydrocortisone", "cortisone", "betamethasone", "triamcinolone",
}

var antibioticPatterns = []string{
	"amoxicillin", "azithromycin", "ciprofloxacin", "levofloxacin",
	"doxycycline", "cephalexin", "trimethoprim", "nitrofurantoin",
	"metronidazole", "clindamycin", "augmentin", "cefuroxime",
}

// ─── constructor ─────────────────────────────────────────────────────────────

// NewClinicalEventDetector creates a ClinicalEventDetector with the given weights.
func NewClinicalEventDetector(w *ConfounderWeights) *ClinicalEventDetector {
	return &ClinicalEventDetector{weights: w}
}

// ─── main detection ──────────────────────────────────────────────────────────

// DetectConfounders scans events and returns confounder factors that overlap
// [windowStart, windowEnd].
func (d *ClinicalEventDetector) DetectConfounders(
	events []PatientClinicalEvent,
	windowStart, windowEnd time.Time,
) []models.ConfounderFactor {
	if len(events) == 0 {
		return nil
	}

	var factors []models.ConfounderFactor

	// 1. Steroid courses
	if f, ok := d.detectSteroids(events, windowStart, windowEnd); ok {
		factors = append(factors, f)
	}

	// 2. Hospitalization
	if f, ok := d.detectHospitalization(events, windowStart, windowEnd); ok {
		factors = append(factors, f)
	}

	// 3. Acute infection (antibiotic proxy)
	if f, ok := d.detectInfection(events, windowStart, windowEnd); ok {
		factors = append(factors, f)
	}

	// 4. AKI (creatinine spike)
	if f, ok := d.detectAKI(events, windowStart, windowEnd); ok {
		factors = append(factors, f)
	}

	return factors
}

// ─── steroid detection ───────────────────────────────────────────────────────

func (d *ClinicalEventDetector) detectSteroids(
	events []PatientClinicalEvent,
	windowStart, windowEnd time.Time,
) (models.ConfounderFactor, bool) {
	var startDate, stopDate time.Time
	foundStart := false

	for _, e := range events {
		if !matchesAny(e.DrugName, steroidPatterns) {
			continue
		}
		if e.Type == "MEDICATION_START" {
			startDate = e.Date
			foundStart = true
		}
		if e.Type == "MEDICATION_STOP" {
			stopDate = e.Date
		}
	}

	if !foundStart {
		return models.ConfounderFactor{}, false
	}

	// If no stop recorded, assume still ongoing — use windowEnd as effective stop.
	if stopDate.IsZero() {
		stopDate = windowEnd
	}

	// Effective end includes washout.
	effectiveEnd := stopDate.AddDate(0, 0, d.weights.SteroidWashoutDays)

	if !overlaps(startDate, effectiveEnd, windowStart, windowEnd) {
		return models.ConfounderFactor{}, false
	}

	overlapDays := computeOverlapDays(startDate, effectiveEnd, windowStart, windowEnd)

	return models.ConfounderFactor{
		Category:          models.ConfounderIatrogenic,
		Name:              "STEROID_COURSE",
		Weight:            d.weights.SteroidWeight,
		AffectedOutcomes:  []string{"DELTA_HBA1C", "DELTA_FPG"},
		ExpectedDirection: "INCREASE",
		ExpectedMagnitude: "MODERATE",
		WindowStart:       startDate,
		WindowEnd:         effectiveEnd,
		OverlapDays:       overlapDays,
		OverlapPct:        overlapPct(overlapDays, windowStart, windowEnd),
		Source:            "CLINICAL_EVENT",
		Confidence:        "HIGH",
	}, true
}

// ─── hospitalization detection ───────────────────────────────────────────────

func (d *ClinicalEventDetector) detectHospitalization(
	events []PatientClinicalEvent,
	windowStart, windowEnd time.Time,
) (models.ConfounderFactor, bool) {
	for _, e := range events {
		if e.Type != "HOSPITALIZATION" {
			continue
		}

		dur := e.Duration
		if dur <= 0 {
			dur = 7 // default assumption
		}
		hospEnd := e.Date.AddDate(0, 0, dur)
		effectiveEnd := hospEnd.AddDate(0, 0, d.weights.HospWashoutDays)

		if !overlaps(e.Date, effectiveEnd, windowStart, windowEnd) {
			continue
		}

		overlapDays := computeOverlapDays(e.Date, effectiveEnd, windowStart, windowEnd)

		return models.ConfounderFactor{
			Category:          models.ConfounderAcuteIllness,
			Name:              "HOSPITALIZATION",
			Weight:            d.weights.HospWeight,
			AffectedOutcomes:  []string{"DELTA_HBA1C", "DELTA_EGFR", "DELTA_BP"},
			ExpectedDirection: "INCREASE",
			ExpectedMagnitude: "HIGH",
			WindowStart:       e.Date,
			WindowEnd:         effectiveEnd,
			OverlapDays:       overlapDays,
			OverlapPct:        overlapPct(overlapDays, windowStart, windowEnd),
			Source:            "CLINICAL_EVENT",
			Confidence:        "HIGH",
		}, true
	}
	return models.ConfounderFactor{}, false
}

// ─── infection detection (antibiotic proxy) ──────────────────────────────────

const assumedAntibioticCourseDays = 14

func (d *ClinicalEventDetector) detectInfection(
	events []PatientClinicalEvent,
	windowStart, windowEnd time.Time,
) (models.ConfounderFactor, bool) {
	for _, e := range events {
		if e.Type != "MEDICATION_START" {
			continue
		}
		if !matchesAny(e.DrugName, antibioticPatterns) {
			continue
		}
		// Already matched as steroid? Skip (shouldn't happen but guard).
		if matchesAny(e.DrugName, steroidPatterns) {
			continue
		}

		courseEnd := e.Date.AddDate(0, 0, assumedAntibioticCourseDays)
		effectiveEnd := courseEnd.AddDate(0, 0, d.weights.InfectionWashoutDays)

		if !overlaps(e.Date, effectiveEnd, windowStart, windowEnd) {
			continue
		}

		overlapDays := computeOverlapDays(e.Date, effectiveEnd, windowStart, windowEnd)

		return models.ConfounderFactor{
			Category:          models.ConfounderAcuteIllness,
			Name:              "ACUTE_INFECTION",
			Weight:            d.weights.InfectionWeight,
			AffectedOutcomes:  []string{"DELTA_HBA1C", "DELTA_FPG", "DELTA_WBC"},
			ExpectedDirection: "INCREASE",
			ExpectedMagnitude: "MODERATE",
			WindowStart:       e.Date,
			WindowEnd:         effectiveEnd,
			OverlapDays:       overlapDays,
			OverlapPct:        overlapPct(overlapDays, windowStart, windowEnd),
			Source:            "CLINICAL_EVENT",
			Confidence:        "HIGH",
		}, true
	}
	return models.ConfounderFactor{}, false
}

// ─── AKI detection (creatinine spike) ────────────────────────────────────────

const kdigoAKIRatio = 1.5

func (d *ClinicalEventDetector) detectAKI(
	events []PatientClinicalEvent,
	windowStart, windowEnd time.Time,
) (models.ConfounderFactor, bool) {
	// Collect creatinine labs sorted chronologically.
	var creatLabs []PatientClinicalEvent
	for _, e := range events {
		if e.Type == "LAB_RESULT" && strings.EqualFold(e.LabType, "CREATININE") {
			creatLabs = append(creatLabs, e)
		}
	}

	if len(creatLabs) < 2 {
		return models.ConfounderFactor{}, false
	}

	// Find baseline (earliest) and peak (latest with highest value).
	baseline := creatLabs[0]
	for _, lab := range creatLabs[1:] {
		if lab.Date.Before(baseline.Date) {
			baseline = lab
		}
	}

	var peak PatientClinicalEvent
	peakFound := false
	for _, lab := range creatLabs {
		if lab.Date.After(baseline.Date) && lab.Value > baseline.Value {
			if !peakFound || lab.Value > peak.Value {
				peak = lab
				peakFound = true
			}
		}
	}

	if !peakFound {
		return models.ConfounderFactor{}, false
	}

	// KDIGO Stage 1: >=1.5x baseline within 7 days (we relax to any timeframe for confounder detection).
	if baseline.Value <= 0 || peak.Value/baseline.Value < kdigoAKIRatio {
		return models.ConfounderFactor{}, false
	}

	// AKI event date is the peak date; washout from that point.
	effectiveEnd := peak.Date.AddDate(0, 0, d.weights.AKIWashoutDays)

	if !overlaps(peak.Date, effectiveEnd, windowStart, windowEnd) {
		return models.ConfounderFactor{}, false
	}

	overlapDays := computeOverlapDays(peak.Date, effectiveEnd, windowStart, windowEnd)

	return models.ConfounderFactor{
		Category:          models.ConfounderAcuteIllness,
		Name:              "ACUTE_KIDNEY_INJURY",
		Weight:            d.weights.AKIWeight,
		AffectedOutcomes:  []string{"DELTA_EGFR", "DELTA_CREATININE", "DELTA_HBA1C"},
		ExpectedDirection: "INCREASE",
		ExpectedMagnitude: "HIGH",
		WindowStart:       peak.Date,
		WindowEnd:         effectiveEnd,
		OverlapDays:       overlapDays,
		OverlapPct:        overlapPct(overlapDays, windowStart, windowEnd),
		Source:            "CLINICAL_EVENT",
		Confidence:        "HIGH",
	}, true
}

// ─── helpers ─────────────────────────────────────────────────────────────────

// matchesAny returns true if s case-insensitively matches any of the patterns.
func matchesAny(s string, patterns []string) bool {
	lower := strings.ToLower(s)
	for _, p := range patterns {
		if strings.EqualFold(lower, p) {
			return true
		}
	}
	return false
}

// overlaps returns true if intervals [aStart, aEnd) and [bStart, bEnd) overlap.
func overlaps(aStart, aEnd, bStart, bEnd time.Time) bool {
	return aStart.Before(bEnd) && bStart.Before(aEnd)
}

// computeOverlapDays returns the number of days the two intervals overlap.
func computeOverlapDays(aStart, aEnd, bStart, bEnd time.Time) int {
	start := maxTime(aStart, bStart)
	end := minTime(aEnd, bEnd)
	if !start.Before(end) {
		return 0
	}
	days := int(end.Sub(start).Hours() / 24)
	if days < 1 {
		days = 1
	}
	return days
}

// overlapPct computes the percentage of the window covered by overlapDays.
func overlapPct(overlapDays int, windowStart, windowEnd time.Time) float64 {
	windowDays := int(windowEnd.Sub(windowStart).Hours() / 24)
	if windowDays <= 0 {
		windowDays = 1
	}
	return float64(overlapDays) / float64(windowDays) * 100.0
}
