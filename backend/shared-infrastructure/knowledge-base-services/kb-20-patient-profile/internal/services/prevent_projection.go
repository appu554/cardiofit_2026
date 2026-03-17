package services

import (
	"kb-patient-profile/internal/models"
)

// assemblePREVENTInput constructs a PREVENTInput from KB-20 data sources.
// All data is already in KB-20 — no external service calls needed.
func assemblePREVENTInput(
	age float64, sex Sex,
	totalChol, hdlChol, sbp float64,
	onBPTreatment, diabetes, smoking bool,
	egfr, bmi float64,
	hba1c, uacr *float64,
	southAsianCalibration bool,
	bmiCalibrationOffset float64,
) PREVENTInput {
	effectiveBMI := bmi
	if southAsianCalibration && bmiCalibrationOffset > 0 {
		effectiveBMI = ApplySouthAsianBMICalibration(bmi, bmiCalibrationOffset)
	}

	input := PREVENTInput{
		Age:              age,
		Sex:              sex,
		TotalCholesterol: totalChol,
		HDLCholesterol:   hdlChol,
		SystolicBP:       sbp,
		OnBPTreatment:    onBPTreatment,
		DiabetesStatus:   diabetes,
		CurrentSmoking:   smoking,
		EGFR:             egfr,
		BMI:              effectiveBMI,
		HbA1c:            hba1c,
		UACR:             uacr,
	}

	input.ModelVariant = SelectPREVENTModel(hba1c, uacr)
	return input
}

// ComputePREVENTProjection computes the PREVENT score and populates
// the PREVENT fields on ChannelCProjection. Called inside buildChannelCProjection().
func (s *ProjectionService) ComputePREVENTProjection(
	patientID string,
	profile models.PatientProfile,
	activeMeds []models.MedicationState,
	proj *models.ChannelCProjection,
) {
	// 1. Gather inputs from labs (via existing latestLabValue helpers)
	totalChol, _ := s.latestLabValue(patientID, models.LabTypeTotalCholesterol)
	hdlChol, _ := s.latestLabValue(patientID, models.LabTypeHDL)
	sbp, _ := s.latestLabValue(patientID, models.LabTypeSBP)
	hba1c, _ := s.latestLabValue(patientID, models.LabTypeHbA1c)
	egfr, _ := s.latestLabValue(patientID, models.LabTypeEGFR)

	// UACR from ACR lab type
	uacr, _ := s.latestLabValue(patientID, models.LabTypeACR)

	// BMI: derive from weight + height, or use profile
	var bmi float64
	if profile.WeightKg > 0 && profile.HeightCm > 0 {
		heightM := profile.HeightCm / 100.0
		bmi = profile.WeightKg / (heightM * heightM)
	}

	// 2. Check minimum required inputs (age, sex, SBP, TC, HDL, eGFR, BMI)
	if profile.Age < 30 || profile.Age > 79 {
		return // PREVENT valid for age 30-79 only
	}
	if totalChol == nil || hdlChol == nil || sbp == nil || egfr == nil || bmi == 0 {
		return // insufficient data — cannot compute PREVENT
	}

	// 3. Medication booleans
	onBPTreatment := hasDrugClass(activeMeds, models.DrugClassACEInhibitor) ||
		hasDrugClass(activeMeds, models.DrugClassARB) ||
		hasDrugClass(activeMeds, models.DrugClassCCB) ||
		hasDrugClass(activeMeds, models.DrugClassBetaBlocker) ||
		hasDrugClass(activeMeds, models.DrugClassDiuretic)
	onStatin := hasDrugClass(activeMeds, models.DrugClassStatin)

	// 4. Sex mapping
	sex := SexMale
	if profile.Sex == "F" || profile.Sex == "FEMALE" {
		sex = SexFemale
	}

	// 5. Diabetes and smoking from PatientProfile fields
	hasDiabetes := profile.DMType == "T1DM" || profile.DMType == "T2DM"
	isSmoker := profile.SmokingStatus == "current"

	// 6a. Assemble input
	// South Asian BMI calibration is deferred to a follow-up task that loads
	// prevent_config.yaml via Viper. For now, calibration is disabled (false, 0).
	input := assemblePREVENTInput(
		float64(profile.Age), sex,
		*totalChol, *hdlChol, *sbp,
		onBPTreatment, hasDiabetes, isSmoker,
		*egfr, bmi,
		hba1c, uacr,
		false, 0,
	)

	// 6b. Compute
	result := ComputePREVENT(input)

	// 7. Override SBP target using config-loaded threshold.
	intensiveThreshold := 0.075 // TODO: load from Viper: cfg.GetFloat64("intensive_target_threshold")
	acr := 0.0
	if uacr != nil {
		acr = *uacr
	}
	result.SBPTarget = DetermineSBPTarget(result.RiskTier, result.TenYearTotalCVD, *egfr, acr, intensiveThreshold)

	// 8. Populate projection
	proj.PREVENTRiskTier = string(result.RiskTier)
	proj.PREVENTSBPTarget = result.SBPTarget
	proj.PREVENT10yrCVD = result.TenYearTotalCVD
	proj.PREVENT10yrASCVD = result.TenYearASCVD
	proj.PREVENT10yrHF = result.TenYearHF
	proj.PREVENTModelUsed = string(result.ModelUsed)
	proj.OnStatin = onStatin
}
