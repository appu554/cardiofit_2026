package services

import (
	"fmt"
	"time"

	"kb-patient-profile/internal/models"
)

// CKMClassifierInput holds all data needed for CKM staging.
type CKMClassifierInput struct {
	Age         int
	BMI         float64
	WaistCm     float64
	Sex         string
	HasDiabetes    bool
	HasPrediabetes bool
	HasHTN         bool
	HasDyslipidemia bool
	HasMetSyndrome  bool
	HbA1c          *float64
	EGFR           float64
	ACR            *float64
	ASCVD10YearRisk *float64
	PREVENTScore    *float64
	CACScore       *float64
	CIMTPercentile *int
	HasLVH         bool
	NTproBNP       *float64
	HasSubclinicalAtherosclerosis bool
	ASCVDEvents    []models.ASCVDEvent
	HasHeartFailure bool
	LVEF           *float64
	NYHAClass      string
	HFEtiology     string
	AsianBMICutoffs bool
	RheumaticEtiology bool
}

// ClassifyCKMStage computes the full CKM stage with substage.
// Implements Ndumele et al. 2023 (Circulation) staging algorithm.
// Hierarchy: 4c > 4b > 4a > 3 > 2 > 1 > 0.
func ClassifyCKMStage(input CKMClassifierInput) models.CKMStageResult {
	result := models.CKMStageResult{
		Metadata: models.SubstageMetadata{
			StagingDate:   time.Now(),
			StagingSource: "ALGORITHM",
		},
	}

	// Stage 4c: Heart failure — highest, check first
	if input.HasHeartFailure {
		result.Stage = models.CKMStageV2_4c
		result.StagingRationale = "Heart failure in CKM context"

		if input.LVEF != nil {
			hfType := models.ClassifyHFType(*input.LVEF)
			result.Metadata.HFClassification = hfType
			result.Metadata.LVEFPercent = input.LVEF
			result.StagingRationale += " — " + string(hfType)
		}
		result.Metadata.NYHAClass = input.NYHAClass
		result.Metadata.NTproBNP = input.NTproBNP
		result.Metadata.HFEtiology = input.HFEtiology
		result.Metadata.RheumaticEtiology = input.RheumaticEtiology
		return result
	}

	// Stage 4b: Clinical ASCVD
	if len(input.ASCVDEvents) > 0 {
		result.Stage = models.CKMStageV2_4b
		result.Metadata.ASCVDEvents = input.ASCVDEvents

		mostRecent := input.ASCVDEvents[0].EventDate
		for _, e := range input.ASCVDEvents[1:] {
			if e.EventDate.After(mostRecent) {
				mostRecent = e.EventDate
			}
		}
		result.Metadata.MostRecentEventDate = &mostRecent
		result.StagingRationale = "Clinical ASCVD history: " + input.ASCVDEvents[0].EventType
		return result
	}

	// Stage 4a: Subclinical CVD
	hasSubclinical := false
	var markers []models.SubclinicalMarker

	if input.CACScore != nil && *input.CACScore > 0 {
		hasSubclinical = true
		result.Metadata.CACScore = input.CACScore
		markers = append(markers, models.SubclinicalMarker{
			MarkerType: "CAC", Value: fmt.Sprintf("%.0f", *input.CACScore), Date: time.Now(),
		})
	}
	if input.CIMTPercentile != nil && *input.CIMTPercentile > 75 {
		hasSubclinical = true
		markers = append(markers, models.SubclinicalMarker{
			MarkerType: "CIMT", Value: fmt.Sprintf(">p%d", *input.CIMTPercentile), Date: time.Now(),
		})
	}
	if input.HasLVH {
		hasSubclinical = true
		result.Metadata.HasLVH = true
		markers = append(markers, models.SubclinicalMarker{
			MarkerType: "LVH", Value: "present", Date: time.Now(),
		})
	}
	if input.NTproBNP != nil && *input.NTproBNP > 125 {
		hasSubclinical = true
		result.Metadata.NTproBNP = input.NTproBNP
		markers = append(markers, models.SubclinicalMarker{
			MarkerType: "NT_PROBNP_ELEVATED",
			Value:      fmt.Sprintf("%.0f pg/mL", *input.NTproBNP),
			Date:       time.Now(),
		})
	}
	if input.HasSubclinicalAtherosclerosis {
		hasSubclinical = true
		markers = append(markers, models.SubclinicalMarker{
			MarkerType: "SUBCLINICAL_ATHEROSCLEROSIS", Value: "present", Date: time.Now(),
		})
	}

	if hasSubclinical {
		result.Stage = models.CKMStageV2_4a
		result.Metadata.SubclinicalMarkers = markers
		result.StagingRationale = fmt.Sprintf("Subclinical CVD: %d marker(s) detected", len(markers))
		return result
	}

	// Stage 3: High predicted risk
	highRisk := false
	if input.PREVENTScore != nil && *input.PREVENTScore >= 10.0 {
		highRisk = true
	} else if input.ASCVD10YearRisk != nil && *input.ASCVD10YearRisk >= 7.5 {
		highRisk = true
	}
	if input.EGFR < 30 || (input.ACR != nil && *input.ACR >= 300) {
		highRisk = true
	}

	if highRisk && (input.HasDiabetes || input.HasHTN || input.HasDyslipidemia) {
		result.Stage = models.CKMStageV2_3
		result.StagingRationale = "High predicted ASCVD risk with metabolic risk factors"
		return result
	}

	// Stage 2: Metabolic risk factors or CKD
	hasMetabolic := input.HasDiabetes || input.HasHTN || input.HasDyslipidemia || input.HasMetSyndrome
	hasCKD := input.EGFR < 60 || (input.ACR != nil && *input.ACR >= 30)

	if hasMetabolic || hasCKD {
		result.Stage = models.CKMStageV2_2
		result.StagingRationale = "Metabolic risk factors and/or CKD present"
		return result
	}

	// Stage 1: Excess adiposity
	bmiOverweight := 25.0
	if input.AsianBMICutoffs {
		bmiOverweight = 23.0
	}
	if input.BMI >= bmiOverweight {
		result.Stage = models.CKMStageV2_1
		result.StagingRationale = "Excess adiposity without metabolic derangement"
		return result
	}

	// Stage 0: No CKM risk factors
	result.Stage = models.CKMStageV2_0
	result.StagingRationale = "No CKM risk factors identified"
	return result
}
