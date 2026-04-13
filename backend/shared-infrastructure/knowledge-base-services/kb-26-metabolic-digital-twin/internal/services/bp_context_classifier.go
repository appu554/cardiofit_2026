package services

import (
	"fmt"
	"math"
	"time"

	"kb-26-metabolic-digital-twin/internal/config"
	"kb-26-metabolic-digital-twin/internal/models"
)

// morningSurgeCompoundLimit is from clinical literature (ESH 2023) and is not
// market-tunable, so it stays as a package-level constant rather than moving
// into BPContextThresholds.
const morningSurgeCompoundLimit = 20.0

// defaultBPContextThresholds returns the ESH 2023 / ISH 2020 reference values
// used when no market config is loaded. Tests pass nil and get these via
// the same code path production uses.
func defaultBPContextThresholds() *config.BPContextThresholds {
	return &config.BPContextThresholds{
		ClinicSBPElevated:        140.0,
		ClinicDBPElevated:        90.0,
		ClinicSBPElevatedDM:      130.0,
		ClinicDBPElevatedDM:      80.0,
		HomeSBPElevated:          135.0,
		HomeDBPElevated:          85.0,
		MinClinicReadings:        2,
		MinHomeReadings:          12,
		MinHomeDays:              4,
		WCEClinicallySignificant: 15.0,
		MinHomeForConfidence:     20,
		FlagIfReadingsBelow:      12,
	}
}

// BPReading represents a single blood pressure measurement.
type BPReading struct {
	SBP         float64   `json:"sbp"`
	DBP         float64   `json:"dbp"`
	Source      string    `json:"source"`
	TimeContext string    `json:"time_context,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// BPContextInput is the input to the clinic-home BP classifier.
type BPContextInput struct {
	PatientID           string
	ClinicReadings      []BPReading
	HomeReadings        []BPReading
	OnAntihypertensives bool
	IsDiabetic          bool
	HasCKD              bool
	EngagementPhenotype string
	MorningSurge7dAvg   float64
}

// ClassifyBPContext performs the clinic-home BP discordance analysis.
// Pass nil for thresholds to use ESH 2023 / ISH 2020 reference defaults.
func ClassifyBPContext(input BPContextInput, thresholds *config.BPContextThresholds) models.BPContextClassification {
	if thresholds == nil {
		thresholds = defaultBPContextThresholds()
	}

	result := models.BPContextClassification{
		PatientID:           input.PatientID,
		ComputedAt:          time.Now(),
		OnAntihypertensives: input.OnAntihypertensives,
		IsDiabetic:          input.IsDiabetic,
		HasCKD:              input.HasCKD,
	}

	// Set thresholds (diabetic patients use stricter clinic thresholds per ISH 2020)
	clinicSBPThresh := thresholds.ClinicSBPElevated
	clinicDBPThresh := thresholds.ClinicDBPElevated
	if input.IsDiabetic {
		clinicSBPThresh = thresholds.ClinicSBPElevatedDM
		clinicDBPThresh = thresholds.ClinicDBPElevatedDM
	}
	result.ClinicSBPThreshold = clinicSBPThresh
	result.ClinicDBPThreshold = clinicDBPThresh
	result.HomeSBPThreshold = thresholds.HomeSBPElevated
	result.HomeDBPThreshold = thresholds.HomeDBPElevated

	// Check data sufficiency
	result.SufficientClinic = len(input.ClinicReadings) >= thresholds.MinClinicReadings
	homeDistinctDays := countDistinctDays(input.HomeReadings)
	result.SufficientHome = len(input.HomeReadings) >= thresholds.MinHomeReadings && homeDistinctDays >= thresholds.MinHomeDays
	result.ClinicReadingCount = len(input.ClinicReadings)
	result.HomeReadingCount = len(input.HomeReadings)
	result.HomeDaysWithData = homeDistinctDays

	if !result.SufficientClinic || !result.SufficientHome {
		result.Phenotype = models.PhenotypeInsufficientData
		result.Confidence = "LOW"
		if len(input.ClinicReadings) > 0 {
			result.ClinicSBPMean, result.ClinicDBPMean = computeBPMeans(input.ClinicReadings)
		}
		if len(input.HomeReadings) > 0 {
			result.HomeSBPMean, result.HomeDBPMean = computeBPMeans(input.HomeReadings)
		}
		return result
	}

	// Compute means
	result.ClinicSBPMean, result.ClinicDBPMean = computeBPMeans(input.ClinicReadings)
	result.HomeSBPMean, result.HomeDBPMean = computeBPMeans(input.HomeReadings)

	// Compute discordance: positive = clinic higher, negative = home higher
	result.ClinicHomeGapSBP = math.Round((result.ClinicSBPMean-result.HomeSBPMean)*10) / 10
	result.ClinicHomeGapDBP = math.Round((result.ClinicDBPMean-result.HomeDBPMean)*10) / 10
	result.WhiteCoatEffect = math.Max(0, result.ClinicHomeGapSBP)

	// Classify against thresholds
	result.ClinicAboveThreshold = result.ClinicSBPMean >= clinicSBPThresh || result.ClinicDBPMean >= clinicDBPThresh
	result.HomeAboveThreshold = result.HomeSBPMean >= thresholds.HomeSBPElevated || result.HomeDBPMean >= thresholds.HomeDBPElevated

	switch {
	case result.ClinicAboveThreshold && result.HomeAboveThreshold:
		result.Phenotype = models.PhenotypeSustainedHTN
	case result.ClinicAboveThreshold && !result.HomeAboveThreshold:
		if input.OnAntihypertensives {
			result.Phenotype = models.PhenotypeWhiteCoatUncontrolled
		} else {
			result.Phenotype = models.PhenotypeWhiteCoatHTN
		}
	case !result.ClinicAboveThreshold && result.HomeAboveThreshold:
		if input.OnAntihypertensives {
			result.Phenotype = models.PhenotypeMaskedUncontrolled
		} else {
			result.Phenotype = models.PhenotypeMaskedHTN
		}
	default:
		result.Phenotype = models.PhenotypeSustainedNormotension
	}

	// Cross-domain amplification — applies to masked AND sustained HTN
	isMasked := result.Phenotype == models.PhenotypeMaskedHTN ||
		result.Phenotype == models.PhenotypeMaskedUncontrolled
	isElevated := isMasked || result.Phenotype == models.PhenotypeSustainedHTN

	if isElevated && input.IsDiabetic {
		result.DiabetesAmplification = true
	}
	if isElevated && input.HasCKD {
		result.CKDAmplification = true
	}
	if isElevated && input.MorningSurge7dAvg > morningSurgeCompoundLimit {
		result.MorningSurgeCompound = true
	}

	// Selection bias detection
	biasRiskPhenotypes := map[string]bool{
		"MEASUREMENT_AVOIDANT": true,
		"CRISIS_ONLY_MEASURER": true,
	}
	if biasRiskPhenotypes[input.EngagementPhenotype] && len(input.HomeReadings) < thresholds.MinHomeForConfidence {
		result.SelectionBiasRisk = true
	}

	// Confidence assessment
	result.Confidence = assessBPConfidence(result, input, thresholds)

	// Medication timing hypothesis
	if input.OnAntihypertensives && isElevated {
		result.MedicationTimingHypothesis = detectMedicationTimingPattern(input.HomeReadings)
	}

	result.EngagementPhenotype = input.EngagementPhenotype
	return result
}

func computeBPMeans(readings []BPReading) (float64, float64) {
	if len(readings) == 0 {
		return 0, 0
	}
	var sumSBP, sumDBP float64
	for _, r := range readings {
		sumSBP += r.SBP
		sumDBP += r.DBP
	}
	n := float64(len(readings))
	return math.Round(sumSBP/n*10) / 10, math.Round(sumDBP/n*10) / 10
}

func countDistinctDays(readings []BPReading) int {
	days := make(map[string]bool)
	for _, r := range readings {
		days[r.Timestamp.Format("2006-01-02")] = true
	}
	return len(days)
}

func assessBPConfidence(result models.BPContextClassification, input BPContextInput, thresholds *config.BPContextThresholds) string {
	if result.SelectionBiasRisk {
		return "LOW"
	}
	if len(input.ClinicReadings) >= 3 && len(input.HomeReadings) >= thresholds.MinHomeForConfidence &&
		countDistinctDays(input.HomeReadings) >= 7 {
		return "HIGH"
	}
	if result.SufficientClinic && result.SufficientHome {
		return "MODERATE"
	}
	return "LOW"
}

func detectMedicationTimingPattern(homeReadings []BPReading) string {
	var morningSBPs, eveningSBPs []float64
	for _, r := range homeReadings {
		switch r.TimeContext {
		case "MORNING":
			morningSBPs = append(morningSBPs, r.SBP)
		case "EVENING":
			eveningSBPs = append(eveningSBPs, r.SBP)
		}
	}

	if len(morningSBPs) < 3 || len(eveningSBPs) < 3 {
		return ""
	}

	morningMean := meanFloat(morningSBPs)
	eveningMean := meanFloat(eveningSBPs)

	if morningMean-eveningMean > 15 {
		return fmt.Sprintf("Morning BP (mean %.0f) significantly higher than evening (mean %.0f) — "+
			"suggests medication wearing off overnight. Consider evening dosing or longer-acting formulation.",
			morningMean, eveningMean)
	}
	if eveningMean-morningMean > 15 {
		return fmt.Sprintf("Evening BP (mean %.0f) higher than morning (mean %.0f) — "+
			"investigate afternoon/evening BP triggers: dietary sodium, stress, medication timing.",
			eveningMean, morningMean)
	}
	return ""
}

func meanFloat(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	s := 0.0
	for _, v := range vals {
		s += v
	}
	return s / float64(len(vals))
}
