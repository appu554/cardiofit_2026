// Package registry contains pre-configured registry definitions
package registry

import (
	"kb-17-population-registry/internal/models"
)

// GetAllRegistryDefinitions returns all pre-configured registry definitions
func GetAllRegistryDefinitions() []models.Registry {
	return []models.Registry{
		GetDiabetesRegistry(),
		GetHypertensionRegistry(),
		GetHeartFailureRegistry(),
		GetCKDRegistry(),
		GetCOPDRegistry(),
		GetPregnancyRegistry(),
		GetOpioidUseRegistry(),
		GetAnticoagulationRegistry(),
	}
}

// GetRegistryDefinition returns a specific registry definition by code
func GetRegistryDefinition(code models.RegistryCode) *models.Registry {
	switch code {
	case models.RegistryDiabetes:
		r := GetDiabetesRegistry()
		return &r
	case models.RegistryHypertension:
		r := GetHypertensionRegistry()
		return &r
	case models.RegistryHeartFailure:
		r := GetHeartFailureRegistry()
		return &r
	case models.RegistryCKD:
		r := GetCKDRegistry()
		return &r
	case models.RegistryCOPD:
		r := GetCOPDRegistry()
		return &r
	case models.RegistryPregnancy:
		r := GetPregnancyRegistry()
		return &r
	case models.RegistryOpioidUse:
		r := GetOpioidUseRegistry()
		return &r
	case models.RegistryAnticoagulation:
		r := GetAnticoagulationRegistry()
		return &r
	default:
		return nil
	}
}

// GetDiabetesRegistry returns the Diabetes registry definition
// ICD-10: E10.* (Type 1), E11.* (Type 2), E13.* (Other specified)
// Key Labs: HbA1c, Fasting Plasma Glucose
// Risk Stratification: HbA1c thresholds
func GetDiabetesRegistry() models.Registry {
	return models.Registry{
		Code:        models.RegistryDiabetes,
		Name:        "Diabetes Mellitus Registry",
		Description: "Registry for patients with Type 1, Type 2, or other specified diabetes mellitus",
		Category:    models.CategoryChronic,
		AutoEnroll:  true,
		Active:      true,
		InclusionCriteria: []models.CriteriaGroup{
			{
				ID:       "dm-diag-1",
				Operator: models.LogicalOr,
				Criteria: []models.Criterion{
					{
						ID:         "dm-e10",
						Type:       models.CriteriaTypeDiagnosis,
						Field:      "code",
						Operator:   models.OperatorStartsWith,
						Value:      "E10",
						CodeSystem: models.CodeSystemICD10,
						Description: "Type 1 Diabetes Mellitus",
					},
					{
						ID:         "dm-e11",
						Type:       models.CriteriaTypeDiagnosis,
						Field:      "code",
						Operator:   models.OperatorStartsWith,
						Value:      "E11",
						CodeSystem: models.CodeSystemICD10,
						Description: "Type 2 Diabetes Mellitus",
					},
					{
						ID:         "dm-e13",
						Type:       models.CriteriaTypeDiagnosis,
						Field:      "code",
						Operator:   models.OperatorStartsWith,
						Value:      "E13",
						CodeSystem: models.CodeSystemICD10,
						Description: "Other Specified Diabetes Mellitus",
					},
				},
				Description: "Diabetes diagnosis codes",
			},
		},
		RiskStratification: &models.RiskStratificationConfig{
			Method: models.RiskMethodRules,
			Thresholds: map[string]interface{}{
				"hba1c_critical": 10.0,
				"hba1c_high":     8.0,
				"hba1c_moderate": 7.0,
			},
			Rules: []models.RiskRule{
				{
					Tier:     models.RiskTierCritical,
					Priority: 1,
					Criteria: []models.CriteriaGroup{
						{
							ID:       "dm-risk-critical",
							Operator: models.LogicalOr,
							Criteria: []models.Criterion{
								{
									Type:     models.CriteriaTypeLabResult,
									Field:    "value",
									Operator: models.OperatorGreaterOrEqual,
									Value:    10.0,
									CodeSystem: models.CodeSystemLOINC,
									Description: "HbA1c >= 10%",
								},
							},
						},
					},
				},
				{
					Tier:     models.RiskTierHigh,
					Priority: 2,
					Criteria: []models.CriteriaGroup{
						{
							ID:       "dm-risk-high",
							Operator: models.LogicalOr,
							Criteria: []models.Criterion{
								{
									Type:     models.CriteriaTypeLabResult,
									Field:    "value",
									Operator: models.OperatorBetween,
									Values:   []interface{}{8.0, 10.0},
									Description: "HbA1c 8-10%",
								},
							},
						},
					},
				},
				{
					Tier:     models.RiskTierModerate,
					Priority: 3,
					Criteria: []models.CriteriaGroup{
						{
							ID:       "dm-risk-moderate",
							Operator: models.LogicalOr,
							Criteria: []models.Criterion{
								{
									Type:     models.CriteriaTypeLabResult,
									Field:    "value",
									Operator: models.OperatorBetween,
									Values:   []interface{}{7.0, 8.0},
									Description: "HbA1c 7-8%",
								},
							},
						},
					},
				},
			},
		},
		CareGapMeasures: []string{"CMS122", "NQF0059", "HEDIS-CDC"},
	}
}

// GetHypertensionRegistry returns the Hypertension registry definition
// ICD-10: I10 (Essential), I11.* (Hypertensive heart disease), I12.* (Hypertensive CKD), I13.* (Hypertensive heart and CKD)
// Key Labs: Blood Pressure
// Risk Stratification: BP thresholds
func GetHypertensionRegistry() models.Registry {
	return models.Registry{
		Code:        models.RegistryHypertension,
		Name:        "Hypertension Registry",
		Description: "Registry for patients with essential hypertension and hypertensive disease",
		Category:    models.CategoryChronic,
		AutoEnroll:  true,
		Active:      true,
		InclusionCriteria: []models.CriteriaGroup{
			{
				ID:       "htn-diag-1",
				Operator: models.LogicalOr,
				Criteria: []models.Criterion{
					{
						ID:         "htn-i10",
						Type:       models.CriteriaTypeDiagnosis,
						Field:      "code",
						Operator:   models.OperatorEquals,
						Value:      "I10",
						CodeSystem: models.CodeSystemICD10,
						Description: "Essential (primary) hypertension",
					},
					{
						ID:         "htn-i11",
						Type:       models.CriteriaTypeDiagnosis,
						Field:      "code",
						Operator:   models.OperatorStartsWith,
						Value:      "I11",
						CodeSystem: models.CodeSystemICD10,
						Description: "Hypertensive heart disease",
					},
					{
						ID:         "htn-i12",
						Type:       models.CriteriaTypeDiagnosis,
						Field:      "code",
						Operator:   models.OperatorStartsWith,
						Value:      "I12",
						CodeSystem: models.CodeSystemICD10,
						Description: "Hypertensive chronic kidney disease",
					},
					{
						ID:         "htn-i13",
						Type:       models.CriteriaTypeDiagnosis,
						Field:      "code",
						Operator:   models.OperatorStartsWith,
						Value:      "I13",
						CodeSystem: models.CodeSystemICD10,
						Description: "Hypertensive heart and chronic kidney disease",
					},
				},
				Description: "Hypertension diagnosis codes",
			},
		},
		RiskStratification: &models.RiskStratificationConfig{
			Method: models.RiskMethodRules,
			Thresholds: map[string]interface{}{
				"systolic_critical":  180,
				"systolic_high":      160,
				"systolic_moderate":  140,
				"diastolic_critical": 120,
				"diastolic_high":     100,
				"diastolic_moderate": 90,
			},
			Rules: []models.RiskRule{
				{
					Tier:     models.RiskTierCritical,
					Priority: 1,
					Criteria: []models.CriteriaGroup{
						{
							ID:       "htn-risk-critical",
							Operator: models.LogicalOr,
							Criteria: []models.Criterion{
								{
									Type:     models.CriteriaTypeVitalSign,
									Field:    "systolic",
									Operator: models.OperatorGreaterOrEqual,
									Value:    180,
									Description: "Systolic BP >= 180 mmHg",
								},
								{
									Type:     models.CriteriaTypeVitalSign,
									Field:    "diastolic",
									Operator: models.OperatorGreaterOrEqual,
									Value:    120,
									Description: "Diastolic BP >= 120 mmHg",
								},
							},
						},
					},
				},
				{
					Tier:     models.RiskTierHigh,
					Priority: 2,
					Criteria: []models.CriteriaGroup{
						{
							ID:       "htn-risk-high",
							Operator: models.LogicalOr,
							Criteria: []models.Criterion{
								{
									Type:     models.CriteriaTypeVitalSign,
									Field:    "systolic",
									Operator: models.OperatorBetween,
									Values:   []interface{}{160, 180},
									Description: "Systolic BP 160-180 mmHg",
								},
								{
									Type:     models.CriteriaTypeVitalSign,
									Field:    "diastolic",
									Operator: models.OperatorBetween,
									Values:   []interface{}{100, 120},
									Description: "Diastolic BP 100-120 mmHg",
								},
							},
						},
					},
				},
			},
		},
		CareGapMeasures: []string{"CMS165", "NQF0018", "HEDIS-CBP"},
	}
}

// GetHeartFailureRegistry returns the Heart Failure registry definition
// ICD-10: I50.* (Heart failure), I42.* (Cardiomyopathy)
// Key Labs: BNP, NT-proBNP
// Risk Stratification: BNP/diagnosis-based
func GetHeartFailureRegistry() models.Registry {
	return models.Registry{
		Code:        models.RegistryHeartFailure,
		Name:        "Heart Failure Registry",
		Description: "Registry for patients with heart failure and cardiomyopathy",
		Category:    models.CategoryChronic,
		AutoEnroll:  true,
		Active:      true,
		InclusionCriteria: []models.CriteriaGroup{
			{
				ID:       "hf-diag-1",
				Operator: models.LogicalOr,
				Criteria: []models.Criterion{
					{
						ID:         "hf-i50",
						Type:       models.CriteriaTypeDiagnosis,
						Field:      "code",
						Operator:   models.OperatorStartsWith,
						Value:      "I50",
						CodeSystem: models.CodeSystemICD10,
						Description: "Heart failure",
					},
					{
						ID:         "hf-i42",
						Type:       models.CriteriaTypeDiagnosis,
						Field:      "code",
						Operator:   models.OperatorStartsWith,
						Value:      "I42",
						CodeSystem: models.CodeSystemICD10,
						Description: "Cardiomyopathy",
					},
				},
				Description: "Heart failure diagnosis codes",
			},
		},
		RiskStratification: &models.RiskStratificationConfig{
			Method: models.RiskMethodRules,
			Thresholds: map[string]interface{}{
				"bnp_critical":     1000,
				"bnp_high":         400,
				"bnp_moderate":     100,
				"ntprobnp_critical": 5000,
				"ntprobnp_high":    900,
				"ntprobnp_moderate": 300,
			},
			Rules: []models.RiskRule{
				{
					Tier:     models.RiskTierCritical,
					Priority: 1,
					Criteria: []models.CriteriaGroup{
						{
							ID:       "hf-risk-critical",
							Operator: models.LogicalOr,
							Criteria: []models.Criterion{
								{
									Type:     models.CriteriaTypeLabResult,
									Field:    "value",
									Operator: models.OperatorGreaterOrEqual,
									Value:    1000,
									Description: "BNP >= 1000 pg/mL",
								},
								{
									Type:     models.CriteriaTypeDiagnosis,
									Field:    "code",
									Operator: models.OperatorIn,
									Values:   []interface{}{"I50.21", "I50.31", "I50.41"},
									CodeSystem: models.CodeSystemICD10,
									Description: "Acute systolic/diastolic HF",
								},
							},
						},
					},
				},
				{
					Tier:     models.RiskTierHigh,
					Priority: 2,
					Criteria: []models.CriteriaGroup{
						{
							ID:       "hf-risk-high",
							Operator: models.LogicalOr,
							Criteria: []models.Criterion{
								{
									Type:     models.CriteriaTypeLabResult,
									Field:    "value",
									Operator: models.OperatorBetween,
									Values:   []interface{}{400, 1000},
									Description: "BNP 400-1000 pg/mL",
								},
							},
						},
					},
				},
			},
		},
		CareGapMeasures: []string{"CMS144", "CMS145", "HEDIS-PCE"},
	}
}

// GetCKDRegistry returns the Chronic Kidney Disease registry definition
// ICD-10: N18.* (Chronic kidney disease)
// Key Labs: eGFR, UACR, Creatinine
// Risk Stratification: eGFR staging (CKD stages 1-5)
func GetCKDRegistry() models.Registry {
	return models.Registry{
		Code:        models.RegistryCKD,
		Name:        "Chronic Kidney Disease Registry",
		Description: "Registry for patients with chronic kidney disease (CKD stages 1-5)",
		Category:    models.CategoryChronic,
		AutoEnroll:  true,
		Active:      true,
		InclusionCriteria: []models.CriteriaGroup{
			{
				ID:       "ckd-diag-1",
				Operator: models.LogicalOr,
				Criteria: []models.Criterion{
					{
						ID:         "ckd-n18",
						Type:       models.CriteriaTypeDiagnosis,
						Field:      "code",
						Operator:   models.OperatorStartsWith,
						Value:      "N18",
						CodeSystem: models.CodeSystemICD10,
						Description: "Chronic kidney disease",
					},
				},
				Description: "CKD diagnosis codes",
			},
			{
				ID:       "ckd-lab-1",
				Operator: models.LogicalOr,
				Criteria: []models.Criterion{
					{
						ID:       "ckd-egfr-low",
						Type:     models.CriteriaTypeLabResult,
						Field:    "value",
						Operator: models.OperatorLessThan,
						Value:    60,
						Unit:     "mL/min/1.73m2",
						Description: "eGFR < 60 mL/min/1.73m2",
					},
				},
				Description: "CKD lab criteria",
			},
		},
		RiskStratification: &models.RiskStratificationConfig{
			Method: models.RiskMethodRules,
			Thresholds: map[string]interface{}{
				"egfr_stage5": 15,
				"egfr_stage4": 30,
				"egfr_stage3b": 45,
				"egfr_stage3a": 60,
			},
			Rules: []models.RiskRule{
				{
					Tier:     models.RiskTierCritical,
					Priority: 1,
					Criteria: []models.CriteriaGroup{
						{
							ID:       "ckd-risk-critical",
							Operator: models.LogicalOr,
							Criteria: []models.Criterion{
								{
									Type:     models.CriteriaTypeLabResult,
									Field:    "value",
									Operator: models.OperatorLessThan,
									Value:    15,
									Description: "eGFR < 15 (Stage 5/ESRD)",
								},
								{
									Type:       models.CriteriaTypeDiagnosis,
									Field:      "code",
									Operator:   models.OperatorEquals,
									Value:      "N18.6",
									CodeSystem: models.CodeSystemICD10,
									Description: "End stage renal disease",
								},
							},
						},
					},
				},
				{
					Tier:     models.RiskTierHigh,
					Priority: 2,
					Criteria: []models.CriteriaGroup{
						{
							ID:       "ckd-risk-high",
							Operator: models.LogicalOr,
							Criteria: []models.Criterion{
								{
									Type:     models.CriteriaTypeLabResult,
									Field:    "value",
									Operator: models.OperatorBetween,
									Values:   []interface{}{15, 30},
									Description: "eGFR 15-30 (Stage 4)",
								},
							},
						},
					},
				},
				{
					Tier:     models.RiskTierModerate,
					Priority: 3,
					Criteria: []models.CriteriaGroup{
						{
							ID:       "ckd-risk-moderate",
							Operator: models.LogicalOr,
							Criteria: []models.Criterion{
								{
									Type:     models.CriteriaTypeLabResult,
									Field:    "value",
									Operator: models.OperatorBetween,
									Values:   []interface{}{30, 60},
									Description: "eGFR 30-60 (Stage 3)",
								},
							},
						},
					},
				},
			},
		},
		CareGapMeasures: []string{"NQF2372", "HEDIS-KED"},
	}
}

// GetCOPDRegistry returns the COPD registry definition
// ICD-10: J44.* (COPD), J43.9 (Emphysema)
// Key Labs: FEV1
// Risk Stratification: GOLD staging
func GetCOPDRegistry() models.Registry {
	return models.Registry{
		Code:        models.RegistryCOPD,
		Name:        "COPD Registry",
		Description: "Registry for patients with chronic obstructive pulmonary disease",
		Category:    models.CategoryChronic,
		AutoEnroll:  true,
		Active:      true,
		InclusionCriteria: []models.CriteriaGroup{
			{
				ID:       "copd-diag-1",
				Operator: models.LogicalOr,
				Criteria: []models.Criterion{
					{
						ID:         "copd-j44",
						Type:       models.CriteriaTypeDiagnosis,
						Field:      "code",
						Operator:   models.OperatorStartsWith,
						Value:      "J44",
						CodeSystem: models.CodeSystemICD10,
						Description: "Chronic obstructive pulmonary disease",
					},
					{
						ID:         "copd-j43",
						Type:       models.CriteriaTypeDiagnosis,
						Field:      "code",
						Operator:   models.OperatorEquals,
						Value:      "J43.9",
						CodeSystem: models.CodeSystemICD10,
						Description: "Emphysema, unspecified",
					},
				},
				Description: "COPD diagnosis codes",
			},
		},
		RiskStratification: &models.RiskStratificationConfig{
			Method: models.RiskMethodRules,
			Thresholds: map[string]interface{}{
				"fev1_gold4": 30,  // < 30% predicted
				"fev1_gold3": 50,  // 30-50% predicted
				"fev1_gold2": 80,  // 50-80% predicted
			},
			Rules: []models.RiskRule{
				{
					Tier:     models.RiskTierCritical,
					Priority: 1,
					Criteria: []models.CriteriaGroup{
						{
							ID:       "copd-risk-critical",
							Operator: models.LogicalOr,
							Criteria: []models.Criterion{
								{
									Type:     models.CriteriaTypeLabResult,
									Field:    "value",
									Operator: models.OperatorLessThan,
									Value:    30,
									Description: "FEV1 < 30% predicted (GOLD 4)",
								},
							},
						},
					},
				},
				{
					Tier:     models.RiskTierHigh,
					Priority: 2,
					Criteria: []models.CriteriaGroup{
						{
							ID:       "copd-risk-high",
							Operator: models.LogicalOr,
							Criteria: []models.Criterion{
								{
									Type:     models.CriteriaTypeLabResult,
									Field:    "value",
									Operator: models.OperatorBetween,
									Values:   []interface{}{30, 50},
									Description: "FEV1 30-50% predicted (GOLD 3)",
								},
							},
						},
					},
				},
			},
		},
		CareGapMeasures: []string{"CMS165", "HEDIS-PCE"},
	}
}

// GetPregnancyRegistry returns the Pregnancy registry definition
// ICD-10: Z34.* (Supervision of normal pregnancy), O* (Pregnancy complications)
// Key Labs: HCG, GCT
// Risk Stratification: Age, complications
func GetPregnancyRegistry() models.Registry {
	return models.Registry{
		Code:        models.RegistryPregnancy,
		Name:        "Pregnancy Registry",
		Description: "Registry for pregnant patients requiring prenatal care coordination",
		Category:    models.CategoryPreventive,
		AutoEnroll:  true,
		Active:      true,
		InclusionCriteria: []models.CriteriaGroup{
			{
				ID:       "preg-diag-1",
				Operator: models.LogicalOr,
				Criteria: []models.Criterion{
					{
						ID:         "preg-z34",
						Type:       models.CriteriaTypeDiagnosis,
						Field:      "code",
						Operator:   models.OperatorStartsWith,
						Value:      "Z34",
						CodeSystem: models.CodeSystemICD10,
						Description: "Supervision of normal pregnancy",
					},
					{
						ID:         "preg-o",
						Type:       models.CriteriaTypeDiagnosis,
						Field:      "code",
						Operator:   models.OperatorStartsWith,
						Value:      "O",
						CodeSystem: models.CodeSystemICD10,
						Description: "Pregnancy, childbirth and puerperium",
					},
				},
				Description: "Pregnancy diagnosis codes",
			},
		},
		ExclusionCriteria: []models.CriteriaGroup{
			{
				ID:       "preg-excl-1",
				Operator: models.LogicalOr,
				Criteria: []models.Criterion{
					{
						ID:         "preg-excl-delivered",
						Type:       models.CriteriaTypeDiagnosis,
						Field:      "code",
						Operator:   models.OperatorStartsWith,
						Value:      "O80",
						CodeSystem: models.CodeSystemICD10,
						Description: "Delivered - exclude post-delivery",
					},
				},
				Description: "Post-delivery exclusion",
			},
		},
		RiskStratification: &models.RiskStratificationConfig{
			Method: models.RiskMethodRules,
			Thresholds: map[string]interface{}{
				"age_high_risk_lower": 35,
				"age_high_risk_upper": 18,
			},
			Rules: []models.RiskRule{
				{
					Tier:     models.RiskTierCritical,
					Priority: 1,
					Criteria: []models.CriteriaGroup{
						{
							ID:       "preg-risk-critical",
							Operator: models.LogicalOr,
							Criteria: []models.Criterion{
								{
									Type:       models.CriteriaTypeDiagnosis,
									Field:      "code",
									Operator:   models.OperatorIn,
									Values:     []interface{}{"O14", "O15", "O44", "O45", "O46"},
									CodeSystem: models.CodeSystemICD10,
									Description: "Pre-eclampsia, eclampsia, placenta previa, abruption",
								},
							},
						},
					},
				},
				{
					Tier:     models.RiskTierHigh,
					Priority: 2,
					Criteria: []models.CriteriaGroup{
						{
							ID:       "preg-risk-high",
							Operator: models.LogicalOr,
							Criteria: []models.Criterion{
								{
									Type:     models.CriteriaTypeAge,
									Field:    "age",
									Operator: models.OperatorGreaterOrEqual,
									Value:    35,
									Description: "Advanced maternal age (>= 35)",
								},
								{
									Type:     models.CriteriaTypeAge,
									Field:    "age",
									Operator: models.OperatorLessThan,
									Value:    18,
									Description: "Teen pregnancy (< 18)",
								},
								{
									Type:       models.CriteriaTypeDiagnosis,
									Field:      "code",
									Operator:   models.OperatorStartsWith,
									Value:      "O24",
									CodeSystem: models.CodeSystemICD10,
									Description: "Gestational diabetes",
								},
							},
						},
					},
				},
			},
		},
		CareGapMeasures: []string{"CMS153", "HEDIS-PPC"},
	}
}

// GetOpioidUseRegistry returns the Opioid Use registry definition
// ICD-10: F11.* (Opioid-related disorders)
// Key Labs: UDS (Urine Drug Screen)
// Risk Stratification: ORT score
func GetOpioidUseRegistry() models.Registry {
	return models.Registry{
		Code:        models.RegistryOpioidUse,
		Name:        "Opioid Use Disorder Registry",
		Description: "Registry for patients with opioid use disorder requiring treatment coordination",
		Category:    models.CategorySpecialty,
		AutoEnroll:  true,
		Active:      true,
		InclusionCriteria: []models.CriteriaGroup{
			{
				ID:       "oud-diag-1",
				Operator: models.LogicalOr,
				Criteria: []models.Criterion{
					{
						ID:         "oud-f11",
						Type:       models.CriteriaTypeDiagnosis,
						Field:      "code",
						Operator:   models.OperatorStartsWith,
						Value:      "F11",
						CodeSystem: models.CodeSystemICD10,
						Description: "Opioid related disorders",
					},
				},
				Description: "Opioid use disorder diagnosis codes",
			},
		},
		RiskStratification: &models.RiskStratificationConfig{
			Method:    models.RiskMethodScore,
			ScoreType: "ORT",
			Thresholds: map[string]interface{}{
				"ort_high":     8,
				"ort_moderate": 4,
			},
			Rules: []models.RiskRule{
				{
					Tier:     models.RiskTierCritical,
					Priority: 1,
					Criteria: []models.CriteriaGroup{
						{
							ID:       "oud-risk-critical",
							Operator: models.LogicalOr,
							Criteria: []models.Criterion{
								{
									Type:       models.CriteriaTypeDiagnosis,
									Field:      "code",
									Operator:   models.OperatorIn,
									Values:     []interface{}{"F11.20", "F11.21", "F11.23", "F11.24"},
									CodeSystem: models.CodeSystemICD10,
									Description: "Opioid dependence with complications",
								},
							},
						},
					},
				},
				{
					Tier:     models.RiskTierHigh,
					Priority: 2,
					Criteria: []models.CriteriaGroup{
						{
							ID:       "oud-risk-high",
							Operator: models.LogicalOr,
							Criteria: []models.Criterion{
								{
									Type:     models.CriteriaTypeRiskScore,
									Field:    "ORT",
									Operator: models.OperatorGreaterOrEqual,
									Value:    8,
									Description: "ORT score >= 8 (high risk)",
								},
							},
						},
					},
				},
			},
		},
		CareGapMeasures: []string{"CMS460", "HEDIS-IET"},
	}
}

// GetAnticoagulationRegistry returns the Anticoagulation registry definition
// Medication-based enrollment (Warfarin, DOACs)
// Key Labs: INR, eGFR
// Risk Stratification: HAS-BLED score
func GetAnticoagulationRegistry() models.Registry {
	return models.Registry{
		Code:        models.RegistryAnticoagulation,
		Name:        "Anticoagulation Management Registry",
		Description: "Registry for patients on anticoagulation therapy requiring close monitoring",
		Category:    models.CategoryMedication,
		AutoEnroll:  true,
		Active:      true,
		InclusionCriteria: []models.CriteriaGroup{
			{
				ID:       "anticoag-med-1",
				Operator: models.LogicalOr,
				Criteria: []models.Criterion{
					{
						ID:         "anticoag-warfarin",
						Type:       models.CriteriaTypeMedication,
						Field:      "code",
						Operator:   models.OperatorIn,
						Values:     []interface{}{"11289", "855288", "855290", "855292", "855296", "855298", "855302", "855306", "855308", "855312", "855314", "855318"},
						CodeSystem: models.CodeSystemRxNorm,
						Description: "Warfarin products",
					},
					{
						ID:         "anticoag-apixaban",
						Type:       models.CriteriaTypeMedication,
						Field:      "code",
						Operator:   models.OperatorIn,
						Values:     []interface{}{"1364430", "1364435"},
						CodeSystem: models.CodeSystemRxNorm,
						Description: "Apixaban (Eliquis)",
					},
					{
						ID:         "anticoag-rivaroxaban",
						Type:       models.CriteriaTypeMedication,
						Field:      "code",
						Operator:   models.OperatorIn,
						Values:     []interface{}{"1114195", "1114198", "1114202"},
						CodeSystem: models.CodeSystemRxNorm,
						Description: "Rivaroxaban (Xarelto)",
					},
					{
						ID:         "anticoag-dabigatran",
						Type:       models.CriteriaTypeMedication,
						Field:      "code",
						Operator:   models.OperatorIn,
						Values:     []interface{}{"1037042", "1037045"},
						CodeSystem: models.CodeSystemRxNorm,
						Description: "Dabigatran (Pradaxa)",
					},
					{
						ID:         "anticoag-edoxaban",
						Type:       models.CriteriaTypeMedication,
						Field:      "code",
						Operator:   models.OperatorIn,
						Values:     []interface{}{"1599538", "1599543"},
						CodeSystem: models.CodeSystemRxNorm,
						Description: "Edoxaban (Savaysa)",
					},
				},
				Description: "Anticoagulant medications",
			},
		},
		RiskStratification: &models.RiskStratificationConfig{
			Method:    models.RiskMethodScore,
			ScoreType: "HAS-BLED",
			Thresholds: map[string]interface{}{
				"hasbled_high":     3,
				"hasbled_moderate": 2,
				"inr_critical_high": 5.0,
				"inr_critical_low":  1.5,
			},
			Rules: []models.RiskRule{
				{
					Tier:     models.RiskTierCritical,
					Priority: 1,
					Criteria: []models.CriteriaGroup{
						{
							ID:       "anticoag-risk-critical",
							Operator: models.LogicalOr,
							Criteria: []models.Criterion{
								{
									Type:     models.CriteriaTypeLabResult,
									Field:    "value",
									Operator: models.OperatorGreaterOrEqual,
									Value:    5.0,
									Description: "INR >= 5.0 (critical high)",
								},
								{
									Type:     models.CriteriaTypeRiskScore,
									Field:    "HAS-BLED",
									Operator: models.OperatorGreaterOrEqual,
									Value:    4,
									Description: "HAS-BLED >= 4 (very high bleeding risk)",
								},
							},
						},
					},
				},
				{
					Tier:     models.RiskTierHigh,
					Priority: 2,
					Criteria: []models.CriteriaGroup{
						{
							ID:       "anticoag-risk-high",
							Operator: models.LogicalOr,
							Criteria: []models.Criterion{
								{
									Type:     models.CriteriaTypeLabResult,
									Field:    "value",
									Operator: models.OperatorGreaterOrEqual,
									Value:    4.0,
									Description: "INR >= 4.0",
								},
								{
									Type:     models.CriteriaTypeRiskScore,
									Field:    "HAS-BLED",
									Operator: models.OperatorGreaterOrEqual,
									Value:    3,
									Description: "HAS-BLED >= 3 (high bleeding risk)",
								},
							},
						},
					},
				},
			},
		},
		CareGapMeasures: []string{"NQF0555", "HEDIS-ART"},
	}
}
