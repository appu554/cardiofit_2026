package services

import (
	"fmt"
	"math"
	"time"

	"kb-26-metabolic-digital-twin/internal/models"
)

// Default thresholds — overridden by market config at runtime.
const (
	defaultClinicSBP          = 140.0
	defaultClinicDBP          = 90.0
	defaultClinicSBP_DM       = 130.0
	defaultClinicDBP_DM       = 80.0
	defaultHomeSBP            = 135.0
	defaultHomeDBP            = 85.0
	minClinicReadings         = 2
	minHomeReadings           = 12
	minHomeDays               = 4
	significantWCE            = 15.0
	morningSurgeCompoundLimit = 20.0
	minHomeForConfidence      = 20
)

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
	ClinicReadings      []BPReading
	HomeReadings        []BPReading
	OnAntihypertensives bool
	IsDiabetic          bool
	HasCKD              bool
	EGFR                float64
	EngagementPhenotype string
	MorningSurge7dAvg   float64
}

// ClassifyBPContext performs the clinic-home BP discordance analysis.
func ClassifyBPContext(input BPContextInput) models.BPContextClassification {
	result := models.BPContextClassification{
		ComputedAt:          time.Now(),
		OnAntihypertensives: input.OnAntihypertensives,
		IsDiabetic:          input.IsDiabetic,
		HasCKD:              input.HasCKD,
	}

	// Set thresholds (diabetic patients use stricter clinic thresholds per ISH 2020)
	clinicSBPThresh := defaultClinicSBP
	clinicDBPThresh := defaultClinicDBP
	if input.IsDiabetic {
		clinicSBPThresh = defaultClinicSBP_DM
		clinicDBPThresh = defaultClinicDBP_DM
	}
	result.ClinicSBPThreshold = clinicSBPThresh
	result.ClinicDBPThreshold = clinicDBPThresh
	result.HomeSBPThreshold = defaultHomeSBP
	result.HomeDBPThreshold = defaultHomeDBP

	// Check data sufficiency
	result.SufficientClinic = len(input.ClinicReadings) >= minClinicReadings
	homeDistinctDays := countDistinctDays(input.HomeReadings)
	result.SufficientHome = len(input.HomeReadings) >= minHomeReadings && homeDistinctDays >= minHomeDays
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
	result.HomeAboveThreshold = result.HomeSBPMean >= defaultHomeSBP || result.HomeDBPMean >= defaultHomeDBP

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
	if biasRiskPhenotypes[input.EngagementPhenotype] && len(input.HomeReadings) < minHomeForConfidence {
		result.SelectionBiasRisk = true
	}

	// Confidence assessment
	result.Confidence = assessBPConfidence(result, input)

	// Medication timing hypothesis
	if input.OnAntihypertensives && isElevated {
		result.MedicationTimingHypothesis = detectMedicationTimingPattern(input.HomeReadings)
	}

	result.EngagementPhenotype = input.EngagementPhenotype
	return result
}

func computeBPMeans(readings []BPReading) (float64, float64) {
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

func assessBPConfidence(result models.BPContextClassification, input BPContextInput) string {
	if result.SelectionBiasRisk {
		return "LOW"
	}
	if len(input.ClinicReadings) >= 3 && len(input.HomeReadings) >= minHomeForConfidence &&
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
	s := 0.0
	for _, v := range vals {
		s += v
	}
	return s / float64(len(vals))
}
