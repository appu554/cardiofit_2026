// Package reference provides reference data for lab interpretation
// icmr_ranges.go implements ICMR (Indian Council of Medical Research) reference ranges
package reference

// =============================================================================
// ICMR REFERENCE RANGES - India-Specific Population Adjustments
// =============================================================================

// ICMRReferenceRange represents India-specific reference ranges
type ICMRReferenceRange struct {
	TestCode        string          `json:"testCode"`
	TestName        string          `json:"testName"`
	Unit            string          `json:"unit"`
	GlobalRange     RangeWithDemog  `json:"globalRange"`
	IndiaRange      RangeWithDemog  `json:"indiaRange"`
	RegionalRanges  []RegionalRange `json:"regionalRanges,omitempty"`
	Rationale       string          `json:"rationale"`
	Governance      ICMRGovernance  `json:"governance"`
}

// RangeWithDemog holds range values with demographic context
type RangeWithDemog struct {
	Low          float64 `json:"low"`
	High         float64 `json:"high"`
	Sex          string  `json:"sex,omitempty"`          // M, F, or empty for both
	AgeMinYears  int     `json:"ageMinYears,omitempty"`
	AgeMaxYears  int     `json:"ageMaxYears,omitempty"`
}

// RegionalRange provides region-specific ranges within India
type RegionalRange struct {
	Region string  `json:"region"` // NORTH, SOUTH, EAST, WEST, NORTHEAST
	Low    float64 `json:"low"`
	High   float64 `json:"high"`
	Notes  string  `json:"notes,omitempty"`
}

// ICMRGovernance tracks authority for ICMR ranges
type ICMRGovernance struct {
	Source            string `json:"source"`
	Publication       string `json:"publication"`
	StudyPopulation   string `json:"studyPopulation"`
	SampleSize        int    `json:"sampleSize,omitempty"`
	YearPublished     int    `json:"yearPublished"`
	EvidenceLevel     string `json:"evidenceLevel"`
	LocalValidation   bool   `json:"localValidation"`
}

// =============================================================================
// ICMR REFERENCE RANGES DATABASE
// =============================================================================

// ICMRRanges contains all India-specific reference ranges
var ICMRRanges = map[string]ICMRReferenceRange{
	// Hemoglobin - Female (ICMR adjusted for Indian population)
	"718-7_F": {
		TestCode: "718-7",
		TestName: "Hemoglobin",
		Unit:     "g/dL",
		GlobalRange: RangeWithDemog{
			Low:  12.0,
			High: 16.0,
			Sex:  "F",
		},
		IndiaRange: RangeWithDemog{
			Low:  11.0,
			High: 14.5,
			Sex:  "F",
		},
		Rationale: "Indian women have lower average Hb due to dietary factors, vegetarianism prevalence, and endemic conditions. WHO/ICMR defines anemia in Indian women as <11 g/dL",
		Governance: ICMRGovernance{
			Source:          "ICMR.Anemia",
			Publication:     "ICMR-NIN Expert Committee on Anemia 2020",
			StudyPopulation: "Indian adult females 18-60 years",
			SampleSize:      12500,
			YearPublished:   2020,
			EvidenceLevel:   "MODERATE (2B)",
			LocalValidation: true,
		},
	},
	// Hemoglobin - Male
	"718-7_M": {
		TestCode: "718-7",
		TestName: "Hemoglobin",
		Unit:     "g/dL",
		GlobalRange: RangeWithDemog{
			Low:  13.5,
			High: 17.5,
			Sex:  "M",
		},
		IndiaRange: RangeWithDemog{
			Low:  12.5,
			High: 16.5,
			Sex:  "M",
		},
		Rationale: "Indian adult males show lower Hb levels compared to Western populations. ICMR defines anemia in Indian men as <12.5 g/dL",
		Governance: ICMRGovernance{
			Source:          "ICMR.Anemia",
			Publication:     "ICMR-NIN Expert Committee on Anemia 2020",
			StudyPopulation: "Indian adult males 18-60 years",
			SampleSize:      11800,
			YearPublished:   2020,
			EvidenceLevel:   "MODERATE (2B)",
			LocalValidation: true,
		},
	},
	// Vitamin B12
	"2132-9": {
		TestCode: "2132-9",
		TestName: "Vitamin B12",
		Unit:     "pg/mL",
		GlobalRange: RangeWithDemog{
			Low:  200.0,
			High: 900.0,
		},
		IndiaRange: RangeWithDemog{
			Low:  150.0, // Lower threshold due to high vegetarian prevalence
			High: 900.0,
		},
		RegionalRanges: []RegionalRange{
			{Region: "SOUTH", Low: 180.0, High: 900.0, Notes: "Higher vegetarian population"},
			{Region: "WEST_GUJARAT", Low: 160.0, High: 900.0, Notes: "Jain community - strict vegetarian"},
			{Region: "NORTHEAST", Low: 200.0, High: 900.0, Notes: "More non-vegetarian diet"},
		},
		Rationale: "High prevalence of vegetarianism in India (30-40% population) leads to lower B12 levels. Subclinical B12 deficiency common even at levels 200-350 pg/mL",
		Governance: ICMRGovernance{
			Source:          "ICMR.Micronutrients",
			Publication:     "ICMR Dietary Guidelines for Indians 2020",
			StudyPopulation: "Pan-India adults",
			SampleSize:      8500,
			YearPublished:   2020,
			EvidenceLevel:   "MODERATE (2B)",
			LocalValidation: true,
		},
	},
	// Vitamin D
	"1989-3": {
		TestCode: "1989-3",
		TestName: "Vitamin D (25-OH)",
		Unit:     "ng/mL",
		GlobalRange: RangeWithDemog{
			Low:  30.0,
			High: 100.0,
		},
		IndiaRange: RangeWithDemog{
			Low:  20.0, // Adjusted due to endemic deficiency
			High: 100.0,
		},
		Rationale: "Despite tropical climate, 70-90% of Indians are Vitamin D deficient due to limited sun exposure (cultural, urban lifestyle), darker skin pigmentation. Endemic deficiency makes 30 ng/mL threshold unrealistic - ICMR recommends >20 ng/mL as sufficient for Indian population",
		Governance: ICMRGovernance{
			Source:          "ICMR.VitaminD",
			Publication:     "ICMR Consensus Statement on Vitamin D 2018",
			StudyPopulation: "Multi-center Indian population",
			SampleSize:      15000,
			YearPublished:   2018,
			EvidenceLevel:   "MODERATE (2B)",
			LocalValidation: true,
		},
	},
	// Ferritin - Female
	"2276-4_F": {
		TestCode: "2276-4",
		TestName: "Ferritin",
		Unit:     "ng/mL",
		GlobalRange: RangeWithDemog{
			Low:  12.0,
			High: 150.0,
			Sex:  "F",
		},
		IndiaRange: RangeWithDemog{
			Low:  10.0,
			High: 120.0,
			Sex:  "F",
		},
		Rationale: "Lower iron stores in Indian women due to vegetarian diets, lower meat consumption. Consider iron deficiency if ferritin <15 ng/mL with Indian population",
		Governance: ICMRGovernance{
			Source:          "ICMR.IronStatus",
			Publication:     "ICMR Guidelines on Iron Deficiency 2017",
			StudyPopulation: "Indian adult females",
			SampleSize:      6200,
			YearPublished:   2017,
			EvidenceLevel:   "MODERATE (2B)",
			LocalValidation: true,
		},
	},
	// Fasting Glucose (adjusted for higher diabetes risk)
	"1558-6": {
		TestCode: "1558-6",
		TestName: "Fasting Glucose",
		Unit:     "mg/dL",
		GlobalRange: RangeWithDemog{
			Low:  70.0,
			High: 100.0,
		},
		IndiaRange: RangeWithDemog{
			Low:  70.0,
			High: 100.0, // Same range but different risk interpretation
		},
		Rationale: "While reference range is similar, Indians develop T2DM at lower BMI and younger age (Asian Indian phenotype). Values 90-100 mg/dL warrant closer monitoring than in Western populations",
		Governance: ICMRGovernance{
			Source:          "ICMR.Diabetes",
			Publication:     "ICMR Guidelines for Management of T2DM 2018",
			StudyPopulation: "Indian adults without diabetes",
			SampleSize:      10000,
			YearPublished:   2018,
			EvidenceLevel:   "HIGH (1A)",
			LocalValidation: true,
		},
	},
	// HbA1c (different diagnostic thresholds for Indians)
	"4548-4": {
		TestCode: "4548-4",
		TestName: "HbA1c",
		Unit:     "%",
		GlobalRange: RangeWithDemog{
			Low:  4.0,
			High: 5.6, // Global: <5.7% normal
		},
		IndiaRange: RangeWithDemog{
			Low:  4.0,
			High: 5.5, // India: <5.6% recommended as normal
		},
		Rationale: "ICMR recommends stricter HbA1c targets for Indians due to higher metabolic risk at lower glycemic values. Consider prediabetes at 5.6-6.4% vs global 5.7-6.4%",
		Governance: ICMRGovernance{
			Source:          "ICMR.Diabetes",
			Publication:     "RSSDI-ICMR Consensus Guidelines 2020",
			StudyPopulation: "Indian population diabetes screening studies",
			SampleSize:      25000,
			YearPublished:   2020,
			EvidenceLevel:   "HIGH (1A)",
			LocalValidation: true,
		},
	},
	// Total Cholesterol
	"2093-3": {
		TestCode: "2093-3",
		TestName: "Total Cholesterol",
		Unit:     "mg/dL",
		GlobalRange: RangeWithDemog{
			Low:  125.0,
			High: 200.0,
		},
		IndiaRange: RangeWithDemog{
			Low:  130.0,
			High: 200.0, // Similar upper limit but different risk interpretation
		},
		Rationale: "Indians have higher cardiovascular risk at same cholesterol levels compared to Western populations. ICMR recommends stricter lipid targets for primary prevention",
		Governance: ICMRGovernance{
			Source:          "ICMR.CVD",
			Publication:     "ICMR-INDIAB Study Lipid Guidelines 2019",
			StudyPopulation: "Indian adults 25-65 years",
			SampleSize:      18000,
			YearPublished:   2019,
			EvidenceLevel:   "HIGH (1A)",
			LocalValidation: true,
		},
	},
	// LDL Cholesterol
	"2089-1": {
		TestCode: "2089-1",
		TestName: "LDL Cholesterol",
		Unit:     "mg/dL",
		GlobalRange: RangeWithDemog{
			Low:  0.0,
			High: 100.0, // Optimal
		},
		IndiaRange: RangeWithDemog{
			Low:  0.0,
			High: 100.0, // Same but earlier intervention recommended
		},
		Rationale: "Indians have predominance of small dense LDL particles increasing atherogenic potential. Consider intervention at LDL >100 even in moderate risk Indians",
		Governance: ICMRGovernance{
			Source:          "ICMR.Lipids",
			Publication:     "Lipid Association of India Guidelines 2020",
			StudyPopulation: "Indian population lipid studies",
			SampleSize:      12000,
			YearPublished:   2020,
			EvidenceLevel:   "HIGH (1A)",
			LocalValidation: true,
		},
	},
	// Triglycerides
	"2571-8": {
		TestCode: "2571-8",
		TestName: "Triglycerides",
		Unit:     "mg/dL",
		GlobalRange: RangeWithDemog{
			Low:  0.0,
			High: 150.0,
		},
		IndiaRange: RangeWithDemog{
			Low:  0.0,
			High: 150.0, // Similar but hypertriglyceridemia more common
		},
		Rationale: "Hypertriglyceridemia highly prevalent in Indians (30-40%) due to carbohydrate-rich diets. Values 150-200 mg/dL require closer attention than in Western populations",
		Governance: ICMRGovernance{
			Source:          "ICMR.CVD",
			Publication:     "ICMR-INDIAB Study 2019",
			StudyPopulation: "Indian adults",
			SampleSize:      16000,
			YearPublished:   2019,
			EvidenceLevel:   "HIGH (1A)",
			LocalValidation: true,
		},
	},
	// Creatinine - Male (adjusted for lower muscle mass)
	"2160-0_M": {
		TestCode: "2160-0",
		TestName: "Creatinine",
		Unit:     "mg/dL",
		GlobalRange: RangeWithDemog{
			Low:  0.7,
			High: 1.3,
			Sex:  "M",
		},
		IndiaRange: RangeWithDemog{
			Low:  0.6,
			High: 1.2,
			Sex:  "M",
		},
		Rationale: "Indians have lower average muscle mass compared to Western populations, leading to lower baseline creatinine. Important for accurate eGFR calculation",
		Governance: ICMRGovernance{
			Source:          "ISN.India",
			Publication:     "Indian Society of Nephrology Guidelines 2018",
			StudyPopulation: "Indian adults without CKD",
			SampleSize:      8000,
			YearPublished:   2018,
			EvidenceLevel:   "MODERATE (2B)",
			LocalValidation: true,
		},
	},
	// Creatinine - Female
	"2160-0_F": {
		TestCode: "2160-0",
		TestName: "Creatinine",
		Unit:     "mg/dL",
		GlobalRange: RangeWithDemog{
			Low:  0.6,
			High: 1.1,
			Sex:  "F",
		},
		IndiaRange: RangeWithDemog{
			Low:  0.5,
			High: 1.0,
			Sex:  "F",
		},
		Rationale: "Lower reference range for Indian females due to lower muscle mass. Important for accurate eGFR calculation",
		Governance: ICMRGovernance{
			Source:          "ISN.India",
			Publication:     "Indian Society of Nephrology Guidelines 2018",
			StudyPopulation: "Indian adult females",
			SampleSize:      7500,
			YearPublished:   2018,
			EvidenceLevel:   "MODERATE (2B)",
			LocalValidation: true,
		},
	},
	// Uric Acid - Male
	"3084-1_M": {
		TestCode: "3084-1",
		TestName: "Uric Acid",
		Unit:     "mg/dL",
		GlobalRange: RangeWithDemog{
			Low:  3.5,
			High: 7.2,
			Sex:  "M",
		},
		IndiaRange: RangeWithDemog{
			Low:  3.0,
			High: 7.0,
			Sex:  "M",
		},
		Rationale: "Rising hyperuricemia prevalence in urban India. ICMR recommends lower threshold for lifestyle intervention",
		Governance: ICMRGovernance{
			Source:          "ICMR.Metabolic",
			Publication:     "ICMR Metabolic Syndrome Guidelines 2020",
			StudyPopulation: "Indian urban adults",
			SampleSize:      5500,
			YearPublished:   2020,
			EvidenceLevel:   "MODERATE (2B)",
			LocalValidation: true,
		},
	},
	// TSH (different normal range considerations)
	"3016-3": {
		TestCode: "3016-3",
		TestName: "TSH",
		Unit:     "mIU/L",
		GlobalRange: RangeWithDemog{
			Low:  0.4,
			High: 4.0,
		},
		IndiaRange: RangeWithDemog{
			Low:  0.5,
			High: 4.5, // Slightly higher upper limit accepted
		},
		Rationale: "High iodine deficiency areas in India may have higher TSH levels. Some studies suggest TSH up to 4.5 mIU/L may be normal in Indian population, especially elderly",
		Governance: ICMRGovernance{
			Source:          "ICMR.Thyroid",
			Publication:     "Indian Thyroid Society Consensus 2019",
			StudyPopulation: "Pan-India thyroid studies",
			SampleSize:      9500,
			YearPublished:   2019,
			EvidenceLevel:   "MODERATE (2B)",
			LocalValidation: true,
		},
	},
}

// =============================================================================
// LOOKUP AND INTERPRETATION FUNCTIONS
// =============================================================================

// GetICMRRange returns India-specific range for a test
func GetICMRRange(testCode string, sex string) *ICMRReferenceRange {
	// Try sex-specific code first
	if sex != "" {
		sexSpecificCode := testCode + "_" + sex
		if icmrRange, exists := ICMRRanges[sexSpecificCode]; exists {
			return &icmrRange
		}
	}

	// Try generic code
	if icmrRange, exists := ICMRRanges[testCode]; exists {
		return &icmrRange
	}

	return nil
}

// GetJurisdictionRange returns appropriate range based on jurisdiction
func GetJurisdictionRange(testCode string, jurisdiction string, sex string) *RangeWithDemog {
	icmrRange := GetICMRRange(testCode, sex)
	if icmrRange == nil {
		return nil
	}

	if jurisdiction == "IN" || jurisdiction == "IND" || jurisdiction == "INDIA" {
		return &icmrRange.IndiaRange
	}

	return &icmrRange.GlobalRange
}

// ICMRLabResult represents an interpreted lab result with ICMR context
type ICMRLabResult struct {
	TestCode             string          `json:"testCode"`
	TestName             string          `json:"testName"`
	Value                float64         `json:"value"`
	Unit                 string          `json:"unit"`
	GlobalInterpretation string          `json:"globalInterpretation"`
	IndiaInterpretation  string          `json:"indiaInterpretation"`
	RangeUsed            string          `json:"rangeUsed"`       // GLOBAL or INDIA
	ClinicalNotes        string          `json:"clinicalNotes"`
	Governance           ICMRGovernance  `json:"governance"`
}

// InterpretWithICMR provides dual interpretation using global and ICMR ranges
func InterpretWithICMR(testCode string, value float64, sex string) *ICMRLabResult {
	icmrRange := GetICMRRange(testCode, sex)
	if icmrRange == nil {
		return nil
	}

	result := &ICMRLabResult{
		TestCode:   testCode,
		TestName:   icmrRange.TestName,
		Value:      value,
		Unit:       icmrRange.Unit,
		Governance: icmrRange.Governance,
	}

	// Global interpretation
	if value < icmrRange.GlobalRange.Low {
		result.GlobalInterpretation = "LOW (Global)"
	} else if value > icmrRange.GlobalRange.High {
		result.GlobalInterpretation = "HIGH (Global)"
	} else {
		result.GlobalInterpretation = "NORMAL (Global)"
	}

	// India interpretation
	if value < icmrRange.IndiaRange.Low {
		result.IndiaInterpretation = "LOW (India)"
	} else if value > icmrRange.IndiaRange.High {
		result.IndiaInterpretation = "HIGH (India)"
	} else {
		result.IndiaInterpretation = "NORMAL (India)"
	}

	// Add clinical notes if interpretations differ
	if result.GlobalInterpretation != result.IndiaInterpretation {
		result.ClinicalNotes = "Note: Global and India ranges differ. " + icmrRange.Rationale
		result.RangeUsed = "INDIA (Recommended for Indian patients)"
	} else {
		result.RangeUsed = "GLOBAL"
	}

	return result
}

// =============================================================================
// SPECIAL POPULATION ADJUSTMENTS
// =============================================================================

// IndianDiabetesRiskAssessment provides diabetes risk assessment for Indians
type IndianDiabetesRiskAssessment struct {
	FastingGlucose   float64  `json:"fastingGlucose"`
	HbA1c            float64  `json:"hba1c"`
	PostprandialGlucose float64 `json:"postprandialGlucose,omitempty"`
	BMI              float64  `json:"bmi"`
	WaistCircum      float64  `json:"waistCircum,omitempty"` // cm
	FamilyHistory    bool     `json:"familyHistory"`

	RiskCategory     string   `json:"riskCategory"`   // LOW, MODERATE, HIGH, DIABETES
	IndianPhenotype  bool     `json:"indianPhenotype"` // Asian Indian phenotype
	Recommendations  []string `json:"recommendations"`
}

// AssessIndianDiabetesRisk performs India-specific diabetes risk assessment
func AssessIndianDiabetesRisk(fastingGlucose, hba1c, bmi float64, familyHistory bool) *IndianDiabetesRiskAssessment {
	assessment := &IndianDiabetesRiskAssessment{
		FastingGlucose: fastingGlucose,
		HbA1c:          hba1c,
		BMI:            bmi,
		FamilyHistory:  familyHistory,
		Recommendations: []string{},
	}

	// Check for Asian Indian phenotype (higher risk at lower BMI)
	// ICMR defines obesity for Indians at BMI ≥25 (vs 30 globally)
	// Overweight at BMI ≥23 (vs 25 globally)
	assessment.IndianPhenotype = bmi >= 23

	// Diabetes diagnosis (ICMR criteria)
	if fastingGlucose >= 126 || hba1c >= 6.5 {
		assessment.RiskCategory = "DIABETES"
		assessment.Recommendations = []string{
			"Diabetes diagnosed - initiate management per ICMR-RSSDI guidelines",
			"Complete diabetes workup (lipids, renal function, eye exam)",
			"Lifestyle modification + pharmacotherapy per guidelines",
			"Screen for complications",
		}
		return assessment
	}

	// Prediabetes (ICMR uses stricter thresholds)
	isPrediabeticFG := fastingGlucose >= 100 && fastingGlucose < 126
	isPrediabeticA1c := hba1c >= 5.6 && hba1c < 6.5

	if isPrediabeticFG || isPrediabeticA1c {
		assessment.RiskCategory = "HIGH"
		assessment.Recommendations = []string{
			"Prediabetes - high risk for progression to T2DM",
			"Intensive lifestyle modification (diet, exercise)",
			"Consider metformin if BMI ≥25 with additional risk factors",
			"Recheck FG and HbA1c in 3-6 months",
			"Cardiovascular risk assessment recommended",
		}
	} else if (fastingGlucose >= 90 && fastingGlucose < 100) || (hba1c >= 5.5 && hba1c < 5.6) {
		// High-normal values warrant closer monitoring in Indians
		assessment.RiskCategory = "MODERATE"
		assessment.Recommendations = []string{
			"Values in high-normal range - increased monitoring recommended for Indians",
			"Lifestyle counseling (diet, exercise)",
			"Annual screening recommended",
		}
		if familyHistory || assessment.IndianPhenotype {
			assessment.Recommendations = append(assessment.Recommendations,
				"Family history/BMI increases risk - consider 6-monthly screening")
		}
	} else {
		assessment.RiskCategory = "LOW"
		assessment.Recommendations = []string{
			"Current values normal",
			"Annual screening recommended for all Indians ≥30 years",
		}
	}

	// Add Indian-phenotype specific recommendations
	if assessment.IndianPhenotype && assessment.RiskCategory != "DIABETES" {
		assessment.Recommendations = append(assessment.Recommendations,
			"Note: BMI ≥23 indicates overweight by Indian standards",
			"Target BMI <23 for optimal metabolic health")
	}

	return assessment
}

// =============================================================================
// ANEMIA CLASSIFICATION (ICMR/WHO India-specific)
// =============================================================================

// IndianAnemiaClassification provides ICMR-specific anemia grading
type IndianAnemiaClassification struct {
	Hemoglobin      float64 `json:"hemoglobin"`
	Sex             string  `json:"sex"`
	AgeYears        int     `json:"ageYears"`
	IsPregnant      bool    `json:"isPregnant"`
	Classification  string  `json:"classification"` // NONE, MILD, MODERATE, SEVERE
	ICMRThreshold   float64 `json:"icmrThreshold"`
	Recommendations []string `json:"recommendations"`
}

// ClassifyIndianAnemia classifies anemia per ICMR/WHO India guidelines
func ClassifyIndianAnemia(hb float64, sex string, ageYears int, isPregnant bool) *IndianAnemiaClassification {
	result := &IndianAnemiaClassification{
		Hemoglobin:      hb,
		Sex:             sex,
		AgeYears:        ageYears,
		IsPregnant:      isPregnant,
		Recommendations: []string{},
	}

	// Determine threshold based on demographics (ICMR/WHO for India)
	var threshold float64
	if isPregnant {
		threshold = 11.0
	} else if sex == "F" && ageYears >= 15 {
		threshold = 12.0 // Non-pregnant adult female
	} else if sex == "M" && ageYears >= 15 {
		threshold = 13.0 // Adult male
	} else if ageYears >= 12 && ageYears < 15 {
		threshold = 12.0 // Adolescent
	} else if ageYears >= 5 && ageYears < 12 {
		threshold = 11.5 // Child 5-11 years
	} else {
		threshold = 11.0 // Child 6 months to 5 years
	}
	result.ICMRThreshold = threshold

	// Classify severity
	if hb >= threshold {
		result.Classification = "NONE"
		result.Recommendations = []string{"Hemoglobin within normal range for Indian standards"}
	} else if hb >= threshold-3.0 {
		result.Classification = "MILD"
		result.Recommendations = []string{
			"Mild anemia detected",
			"Dietary counseling: iron-rich foods, vitamin C with meals",
			"Consider iron supplementation if diet inadequate",
			"Recheck in 3 months",
		}
	} else if hb >= threshold-5.0 {
		result.Classification = "MODERATE"
		result.Recommendations = []string{
			"Moderate anemia - requires investigation and treatment",
			"Complete iron studies (ferritin, TIBC, serum iron)",
			"Check for B12/folate if macrocytic",
			"Oral iron supplementation recommended",
			"Evaluate for underlying cause (chronic disease, blood loss)",
			"Follow-up in 4-6 weeks",
		}
	} else {
		result.Classification = "SEVERE"
		result.Recommendations = []string{
			"SEVERE ANEMIA - urgent evaluation and treatment required",
			"Evaluate for need of blood transfusion",
			"Complete workup: iron studies, reticulocyte count, peripheral smear",
			"Identify and treat underlying cause urgently",
			"Consider hematology referral",
		}
	}

	return result
}

// GetAllICMRTests returns list of all tests with ICMR-specific ranges
func GetAllICMRTests() []string {
	tests := make([]string, 0, len(ICMRRanges))
	seen := make(map[string]bool)
	for _, icmr := range ICMRRanges {
		if !seen[icmr.TestCode] {
			tests = append(tests, icmr.TestCode)
			seen[icmr.TestCode] = true
		}
	}
	return tests
}
