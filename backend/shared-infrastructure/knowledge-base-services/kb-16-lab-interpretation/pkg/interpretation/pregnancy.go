// Package interpretation provides clinical interpretation algorithms
// pregnancy.go implements pregnancy-specific reference ranges (ACOG/ATA)
package interpretation

// =============================================================================
// PREGNANCY-SPECIFIC REFERENCE RANGES (ACOG 2023, ATA 2017)
// =============================================================================

// PregnancyContext represents the pregnancy state for lab interpretation
type PregnancyContext struct {
	IsPregnant       bool     `json:"isPregnant"`
	Trimester        int      `json:"trimester"`        // 1, 2, 3
	GestationalWeeks int      `json:"gestationalWeeks"` // 0-42
	Conditions       []string `json:"conditions"`       // preeclampsia, gestational_diabetes, etc.
	HighRisk         bool     `json:"highRisk"`
}

// PregnancyRange represents trimester-specific reference ranges
type PregnancyRange struct {
	TestCode     string            `json:"testCode"`
	TestName     string            `json:"testName"`
	Unit         string            `json:"unit"`
	Trimester1   *RangeValues      `json:"trimester1,omitempty"`
	Trimester2   *RangeValues      `json:"trimester2,omitempty"`
	Trimester3   *RangeValues      `json:"trimester3,omitempty"`
	NonPregnant  *RangeValues      `json:"nonPregnant,omitempty"`
	Governance   PregnancyGov      `json:"governance"`
}

// RangeValues holds low and high values for a range
type RangeValues struct {
	Low  float64 `json:"low"`
	High float64 `json:"high"`
}

// PregnancyGov tracks governance for pregnancy-specific ranges
type PregnancyGov struct {
	GuidelineSource string `json:"guidelineSource"`
	GuidelineRef    string `json:"guidelineRef"`
	EvidenceLevel   string `json:"evidenceLevel"`
	LastReviewed    string `json:"lastReviewed"`
}

// PregnancyLabResult represents an interpreted lab result with pregnancy adjustment
type PregnancyLabResult struct {
	TestCode         string          `json:"testCode"`
	TestName         string          `json:"testName"`
	Value            float64         `json:"value"`
	Unit             string          `json:"unit"`
	Trimester        int             `json:"trimester"`
	ReferenceRange   *RangeValues    `json:"referenceRange"`
	Interpretation   string          `json:"interpretation"`   // NORMAL, LOW, HIGH, CRITICAL
	ClinicalContext  string          `json:"clinicalContext"`  // Pregnancy-specific interpretation
	AlertLevel       string          `json:"alertLevel"`       // ROUTINE, URGENT, EMERGENT
	Recommendations  []string        `json:"recommendations"`
	Governance       PregnancyGov    `json:"governance"`
}

// =============================================================================
// PREGNANCY REFERENCE RANGES DATABASE (ACOG 2023, ATA 2017)
// =============================================================================

// PregnancyRanges contains all pregnancy-adjusted reference ranges
var PregnancyRanges = map[string]PregnancyRange{
	// Hemoglobin (ACOG 2023)
	"718-7": {
		TestCode: "718-7",
		TestName: "Hemoglobin",
		Unit:     "g/dL",
		Trimester1: &RangeValues{Low: 11.0, High: 14.5},
		Trimester2: &RangeValues{Low: 10.5, High: 14.0},
		Trimester3: &RangeValues{Low: 10.5, High: 14.0},
		NonPregnant: &RangeValues{Low: 12.0, High: 16.0},
		Governance: PregnancyGov{
			GuidelineSource: "ACOG.Anemia",
			GuidelineRef:    "ACOG Practice Bulletin No. 233: Anemia in Pregnancy",
			EvidenceLevel:   "STRONG (1A)",
			LastReviewed:    "2023-08-01",
		},
	},
	// Hematocrit (ACOG 2023)
	"4544-3": {
		TestCode: "4544-3",
		TestName: "Hematocrit",
		Unit:     "%",
		Trimester1: &RangeValues{Low: 33.0, High: 44.0},
		Trimester2: &RangeValues{Low: 32.0, High: 42.0},
		Trimester3: &RangeValues{Low: 32.0, High: 42.0},
		NonPregnant: &RangeValues{Low: 36.0, High: 48.0},
		Governance: PregnancyGov{
			GuidelineSource: "ACOG.Anemia",
			GuidelineRef:    "ACOG Practice Bulletin No. 233",
			EvidenceLevel:   "STRONG (1A)",
			LastReviewed:    "2023-08-01",
		},
	},
	// Creatinine (ACOG 2020 - Renal changes)
	"2160-0": {
		TestCode: "2160-0",
		TestName: "Creatinine",
		Unit:     "mg/dL",
		Trimester1: &RangeValues{Low: 0.4, High: 0.8},
		Trimester2: &RangeValues{Low: 0.4, High: 0.8},
		Trimester3: &RangeValues{Low: 0.4, High: 0.9},
		NonPregnant: &RangeValues{Low: 0.6, High: 1.1},
		Governance: PregnancyGov{
			GuidelineSource: "ACOG.Renal",
			GuidelineRef:    "ACOG Obstetric Care Consensus: Renal Disease",
			EvidenceLevel:   "MODERATE (2B)",
			LastReviewed:    "2020-01-01",
		},
	},
	// TSH - Thyroid (ATA 2017)
	"3016-3": {
		TestCode: "3016-3",
		TestName: "TSH",
		Unit:     "mIU/L",
		Trimester1: &RangeValues{Low: 0.1, High: 2.5},
		Trimester2: &RangeValues{Low: 0.2, High: 3.0},
		Trimester3: &RangeValues{Low: 0.3, High: 3.0},
		NonPregnant: &RangeValues{Low: 0.4, High: 4.0},
		Governance: PregnancyGov{
			GuidelineSource: "ATA.Thyroid",
			GuidelineRef:    "ATA 2017 Guidelines for Thyroid Disease in Pregnancy",
			EvidenceLevel:   "STRONG (1B)",
			LastReviewed:    "2017-01-01",
		},
	},
	// Free T4 (ATA 2017)
	"3024-7": {
		TestCode: "3024-7",
		TestName: "Free T4",
		Unit:     "ng/dL",
		Trimester1: &RangeValues{Low: 0.8, High: 1.5},
		Trimester2: &RangeValues{Low: 0.6, High: 1.2},
		Trimester3: &RangeValues{Low: 0.5, High: 1.0},
		NonPregnant: &RangeValues{Low: 0.8, High: 1.8},
		Governance: PregnancyGov{
			GuidelineSource: "ATA.Thyroid",
			GuidelineRef:    "ATA 2017 Guidelines for Thyroid Disease in Pregnancy",
			EvidenceLevel:   "MODERATE (2B)",
			LastReviewed:    "2017-01-01",
		},
	},
	// Platelets (ACOG 2022 - Thrombocytopenia)
	"777-3": {
		TestCode: "777-3",
		TestName: "Platelets",
		Unit:     "x10^9/L",
		Trimester1: &RangeValues{Low: 150.0, High: 400.0},
		Trimester2: &RangeValues{Low: 130.0, High: 400.0},
		Trimester3: &RangeValues{Low: 100.0, High: 400.0}, // Gestational thrombocytopenia acceptable >100
		NonPregnant: &RangeValues{Low: 150.0, High: 400.0},
		Governance: PregnancyGov{
			GuidelineSource: "ACOG.Thrombocytopenia",
			GuidelineRef:    "ACOG Practice Bulletin No. 207: Thrombocytopenia in Pregnancy",
			EvidenceLevel:   "MODERATE (2B)",
			LastReviewed:    "2022-03-01",
		},
	},
	// Uric Acid (Preeclampsia marker)
	"3084-1": {
		TestCode: "3084-1",
		TestName: "Uric Acid",
		Unit:     "mg/dL",
		Trimester1: &RangeValues{Low: 2.0, High: 4.5},
		Trimester2: &RangeValues{Low: 2.0, High: 4.5},
		Trimester3: &RangeValues{Low: 2.5, High: 5.5}, // >5.5 concerning for preeclampsia
		NonPregnant: &RangeValues{Low: 2.5, High: 7.0},
		Governance: PregnancyGov{
			GuidelineSource: "ACOG.Hypertension",
			GuidelineRef:    "ACOG Practice Bulletin No. 222: Gestational Hypertension and Preeclampsia",
			EvidenceLevel:   "MODERATE (2B)",
			LastReviewed:    "2020-06-01",
		},
	},
	// ALT (Liver function)
	"1742-6": {
		TestCode: "1742-6",
		TestName: "ALT",
		Unit:     "U/L",
		Trimester1: &RangeValues{Low: 6.0, High: 32.0},
		Trimester2: &RangeValues{Low: 6.0, High: 32.0},
		Trimester3: &RangeValues{Low: 6.0, High: 32.0},
		NonPregnant: &RangeValues{Low: 7.0, High: 56.0},
		Governance: PregnancyGov{
			GuidelineSource: "ACOG.Liver",
			GuidelineRef:    "ACOG Obstetric Care Consensus: Liver Disease",
			EvidenceLevel:   "MODERATE (2B)",
			LastReviewed:    "2021-01-01",
		},
	},
	// AST (Liver function)
	"1920-8": {
		TestCode: "1920-8",
		TestName: "AST",
		Unit:     "U/L",
		Trimester1: &RangeValues{Low: 10.0, High: 28.0},
		Trimester2: &RangeValues{Low: 10.0, High: 28.0},
		Trimester3: &RangeValues{Low: 10.0, High: 28.0},
		NonPregnant: &RangeValues{Low: 10.0, High: 40.0},
		Governance: PregnancyGov{
			GuidelineSource: "ACOG.Liver",
			GuidelineRef:    "ACOG Obstetric Care Consensus: Liver Disease",
			EvidenceLevel:   "MODERATE (2B)",
			LastReviewed:    "2021-01-01",
		},
	},
	// LDH (HELLP marker)
	"2532-0": {
		TestCode: "2532-0",
		TestName: "LDH",
		Unit:     "U/L",
		Trimester1: &RangeValues{Low: 82.0, High: 524.0},
		Trimester2: &RangeValues{Low: 103.0, High: 227.0},
		Trimester3: &RangeValues{Low: 115.0, High: 221.0},
		NonPregnant: &RangeValues{Low: 140.0, High: 280.0},
		Governance: PregnancyGov{
			GuidelineSource: "ACOG.HELLP",
			GuidelineRef:    "ACOG Practice Bulletin: HELLP Syndrome",
			EvidenceLevel:   "MODERATE (2B)",
			LastReviewed:    "2022-01-01",
		},
	},
	// Total Bilirubin
	"1975-2": {
		TestCode: "1975-2",
		TestName: "Total Bilirubin",
		Unit:     "mg/dL",
		Trimester1: &RangeValues{Low: 0.1, High: 1.0},
		Trimester2: &RangeValues{Low: 0.1, High: 1.0},
		Trimester3: &RangeValues{Low: 0.1, High: 1.0},
		NonPregnant: &RangeValues{Low: 0.1, High: 1.2},
		Governance: PregnancyGov{
			GuidelineSource: "ACOG.HELLP",
			GuidelineRef:    "ACOG Practice Bulletin: HELLP Syndrome",
			EvidenceLevel:   "MODERATE (2B)",
			LastReviewed:    "2022-01-01",
		},
	},
	// Fibrinogen (increases in pregnancy)
	"3255-7": {
		TestCode: "3255-7",
		TestName: "Fibrinogen",
		Unit:     "mg/dL",
		Trimester1: &RangeValues{Low: 200.0, High: 400.0},
		Trimester2: &RangeValues{Low: 300.0, High: 500.0},
		Trimester3: &RangeValues{Low: 400.0, High: 650.0},
		NonPregnant: &RangeValues{Low: 200.0, High: 400.0},
		Governance: PregnancyGov{
			GuidelineSource: "ACOG.Coagulation",
			GuidelineRef:    "ACOG Practice Bulletin: Coagulation Changes in Pregnancy",
			EvidenceLevel:   "MODERATE (2B)",
			LastReviewed:    "2021-01-01",
		},
	},
}

// =============================================================================
// INTERPRETATION FUNCTIONS
// =============================================================================

// GetPregnancyAdjustedRange returns trimester-specific reference range
func GetPregnancyAdjustedRange(testCode string, trimester int) *RangeValues {
	pregRange, exists := PregnancyRanges[testCode]
	if !exists {
		return nil
	}

	switch trimester {
	case 1:
		return pregRange.Trimester1
	case 2:
		return pregRange.Trimester2
	case 3:
		return pregRange.Trimester3
	default:
		return pregRange.NonPregnant
	}
}

// InterpretPregnancyLab interprets a lab result with pregnancy context
func InterpretPregnancyLab(testCode string, value float64, ctx *PregnancyContext) *PregnancyLabResult {
	pregRange, exists := PregnancyRanges[testCode]
	if !exists {
		return nil
	}

	result := &PregnancyLabResult{
		TestCode:   testCode,
		TestName:   pregRange.TestName,
		Value:      value,
		Unit:       pregRange.Unit,
		Trimester:  ctx.Trimester,
		Governance: pregRange.Governance,
	}

	// Get appropriate range
	var refRange *RangeValues
	if ctx.IsPregnant {
		refRange = GetPregnancyAdjustedRange(testCode, ctx.Trimester)
	} else {
		refRange = pregRange.NonPregnant
	}
	result.ReferenceRange = refRange

	if refRange == nil {
		result.Interpretation = "INDETERMINATE"
		result.ClinicalContext = "Reference range not available"
		return result
	}

	// Interpret result
	result.interpretValue(ctx)
	result.addRecommendations(ctx)

	return result
}

// interpretValue sets interpretation and clinical context
func (r *PregnancyLabResult) interpretValue(ctx *PregnancyContext) {
	if r.ReferenceRange == nil {
		return
	}

	// Basic interpretation
	if r.Value < r.ReferenceRange.Low {
		r.Interpretation = "LOW"
	} else if r.Value > r.ReferenceRange.High {
		r.Interpretation = "HIGH"
	} else {
		r.Interpretation = "NORMAL"
	}

	// Test-specific clinical context
	switch r.TestCode {
	case "718-7": // Hemoglobin
		r.interpretHemoglobin(ctx)
	case "3016-3": // TSH
		r.interpretTSH(ctx)
	case "2160-0": // Creatinine
		r.interpretCreatinine(ctx)
	case "777-3": // Platelets
		r.interpretPlatelets(ctx)
	case "3084-1": // Uric Acid
		r.interpretUricAcid(ctx)
	case "1742-6", "1920-8": // ALT, AST
		r.interpretLiverEnzymes(ctx)
	default:
		r.setDefaultContext(ctx)
	}
}

func (r *PregnancyLabResult) interpretHemoglobin(ctx *PregnancyContext) {
	if r.Value < 7.0 {
		r.AlertLevel = "EMERGENT"
		r.ClinicalContext = "Severe anemia - immediate transfusion may be required"
	} else if r.Value < 10.0 {
		r.AlertLevel = "URGENT"
		r.ClinicalContext = "Moderate anemia - iron studies and evaluation needed"
	} else if r.Value < r.ReferenceRange.Low {
		r.AlertLevel = "ROUTINE"
		r.ClinicalContext = "Mild anemia - physiologic hemodilution vs iron deficiency"
	} else {
		r.AlertLevel = "ROUTINE"
		r.ClinicalContext = "Hemoglobin within pregnancy-adjusted normal range"
	}
}

func (r *PregnancyLabResult) interpretTSH(ctx *PregnancyContext) {
	// TSH has critical thresholds in pregnancy (ATA 2017)
	if r.Value < 0.01 {
		r.AlertLevel = "URGENT"
		r.ClinicalContext = "Suppressed TSH - evaluate for hyperthyroidism vs gestational thyrotoxicosis"
		r.Interpretation = "CRITICAL_LOW"
	} else if r.Value > 4.0 {
		r.AlertLevel = "URGENT"
		r.ClinicalContext = "Elevated TSH - subclinical/overt hypothyroidism requires treatment"
		r.Interpretation = "CRITICAL_HIGH"
	} else if r.Value > r.ReferenceRange.High {
		r.AlertLevel = "ROUTINE"
		r.ClinicalContext = "Mildly elevated TSH - consider levothyroxine, check TPO antibodies"
	} else if r.Value < r.ReferenceRange.Low {
		r.AlertLevel = "ROUTINE"
		r.ClinicalContext = "Low TSH - may be physiologic in early pregnancy (hCG effect)"
	} else {
		r.AlertLevel = "ROUTINE"
		r.ClinicalContext = "TSH within pregnancy-adjusted trimester-specific range"
	}
}

func (r *PregnancyLabResult) interpretCreatinine(ctx *PregnancyContext) {
	// Creatinine should be LOW in pregnancy due to increased GFR
	if r.Value > 0.9 {
		r.AlertLevel = "URGENT"
		r.ClinicalContext = "Elevated creatinine in pregnancy - evaluate for preeclampsia, renal disease"
		r.Interpretation = "HIGH"
	} else if r.Value > r.ReferenceRange.High {
		r.AlertLevel = "ROUTINE"
		r.ClinicalContext = "Creatinine above expected pregnancy range - monitor closely"
	} else {
		r.AlertLevel = "ROUTINE"
		r.ClinicalContext = "Creatinine within expected pregnancy range (physiologic hyperfiltration)"
	}

	// Check for preeclampsia conditions
	for _, cond := range ctx.Conditions {
		if cond == "preeclampsia" && r.Value > 1.1 {
			r.AlertLevel = "EMERGENT"
			r.ClinicalContext = "Elevated creatinine with preeclampsia - severe feature"
		}
	}
}

func (r *PregnancyLabResult) interpretPlatelets(ctx *PregnancyContext) {
	if r.Value < 50 {
		r.AlertLevel = "EMERGENT"
		r.ClinicalContext = "Severe thrombocytopenia - risk of bleeding, evaluate for HELLP"
		r.Interpretation = "CRITICAL_LOW"
	} else if r.Value < 100 {
		r.AlertLevel = "URGENT"
		r.ClinicalContext = "Moderate thrombocytopenia - evaluate for HELLP, ITP, preeclampsia"
	} else if r.Value < 150 && ctx.Trimester == 3 {
		r.AlertLevel = "ROUTINE"
		r.ClinicalContext = "Gestational thrombocytopenia - common in third trimester, usually benign"
	} else if r.Value < r.ReferenceRange.Low {
		r.AlertLevel = "ROUTINE"
		r.ClinicalContext = "Mild thrombocytopenia - monitor trend"
	} else {
		r.AlertLevel = "ROUTINE"
		r.ClinicalContext = "Platelets within normal pregnancy range"
	}
}

func (r *PregnancyLabResult) interpretUricAcid(ctx *PregnancyContext) {
	// Uric acid >5.5 mg/dL is concerning for preeclampsia
	if r.Value > 7.0 {
		r.AlertLevel = "URGENT"
		r.ClinicalContext = "Significantly elevated uric acid - strongly associated with preeclampsia severity"
	} else if r.Value > 5.5 {
		r.AlertLevel = "ROUTINE"
		r.ClinicalContext = "Elevated uric acid - monitor for preeclampsia, check BP and proteinuria"
	} else {
		r.AlertLevel = "ROUTINE"
		r.ClinicalContext = "Uric acid within acceptable pregnancy range"
	}
}

func (r *PregnancyLabResult) interpretLiverEnzymes(ctx *PregnancyContext) {
	// AST/ALT >70 U/L is HELLP criterion
	if r.Value >= 70 {
		r.AlertLevel = "URGENT"
		r.ClinicalContext = "Elevated liver enzymes ≥70 U/L - meets HELLP criterion, evaluate immediately"
		r.Interpretation = "CRITICAL_HIGH"
	} else if r.Value > r.ReferenceRange.High {
		r.AlertLevel = "ROUTINE"
		r.ClinicalContext = "Mildly elevated liver enzymes - monitor for HELLP, check LDH and platelets"
	} else {
		r.AlertLevel = "ROUTINE"
		r.ClinicalContext = "Liver enzymes within normal pregnancy range"
	}
}

func (r *PregnancyLabResult) setDefaultContext(ctx *PregnancyContext) {
	if r.Interpretation == "NORMAL" {
		r.AlertLevel = "ROUTINE"
		r.ClinicalContext = "Result within pregnancy-adjusted reference range"
	} else {
		r.AlertLevel = "ROUTINE"
		r.ClinicalContext = "Result outside pregnancy-adjusted reference range - clinical correlation required"
	}
}

// addRecommendations adds test-specific recommendations
func (r *PregnancyLabResult) addRecommendations(ctx *PregnancyContext) {
	r.Recommendations = []string{}

	switch r.AlertLevel {
	case "EMERGENT":
		r.Recommendations = append(r.Recommendations,
			"Immediate clinical evaluation required",
			"Consider admission for monitoring",
			"Notify obstetric team urgently")
	case "URGENT":
		r.Recommendations = append(r.Recommendations,
			"Clinical evaluation within 24 hours",
			"Review in context of other labs and clinical findings")
	case "ROUTINE":
		r.Recommendations = append(r.Recommendations,
			"Routine follow-up as clinically indicated")
	}

	// Test-specific recommendations
	switch r.TestCode {
	case "718-7": // Hemoglobin
		if r.Interpretation == "LOW" {
			r.Recommendations = append(r.Recommendations,
				"Check iron studies (ferritin, TIBC, serum iron)",
				"Consider B12 and folate if MCV elevated",
				"Dietary counseling and iron supplementation")
		}
	case "3016-3": // TSH
		if r.Interpretation == "CRITICAL_HIGH" || r.Interpretation == "HIGH" {
			r.Recommendations = append(r.Recommendations,
				"Check TPO antibodies",
				"Endocrinology referral if TSH >4.0",
				"Consider levothyroxine therapy")
		}
	case "777-3": // Platelets
		if r.Value < 100 {
			r.Recommendations = append(r.Recommendations,
				"Check peripheral smear for schistocytes",
				"Evaluate for HELLP syndrome",
				"Monitor for bleeding symptoms")
		}
	}
}

// =============================================================================
// PREECLAMPSIA SCREENING SUPPORT
// =============================================================================

// PreeclampsiaRiskFactors represents risk assessment inputs
type PreeclampsiaRiskFactors struct {
	SBP              int      `json:"sbp"`              // mmHg
	DBP              int      `json:"dbp"`              // mmHg
	Proteinuria      bool     `json:"proteinuria"`
	UricAcid         float64  `json:"uricAcid"`         // mg/dL
	Creatinine       float64  `json:"creatinine"`       // mg/dL
	Platelets        float64  `json:"platelets"`        // x10^9/L
	AST              float64  `json:"ast"`              // U/L
	ALT              float64  `json:"alt"`              // U/L
	LDH              float64  `json:"ldh"`              // U/L
	Headache         bool     `json:"headache"`
	VisualChanges    bool     `json:"visualChanges"`
	EpigastricPain   bool     `json:"epigastricPain"`
}

// PreeclampsiaAssessment contains risk assessment results
type PreeclampsiaAssessment struct {
	Classification   string   `json:"classification"` // NORMAL, GESTATIONAL_HTN, PREECLAMPSIA, SEVERE_PREECLAMPSIA
	SevereFeatures   []string `json:"severeFeatures"`
	RiskScore        int      `json:"riskScore"`
	Recommendations  []string `json:"recommendations"`
	UrgencyLevel     string   `json:"urgencyLevel"`
}

// AssessPreeclampsiaRisk evaluates preeclampsia risk per ACOG criteria
func AssessPreeclampsiaRisk(factors *PreeclampsiaRiskFactors, ctx *PregnancyContext) *PreeclampsiaAssessment {
	assessment := &PreeclampsiaAssessment{
		SevereFeatures:  []string{},
		Recommendations: []string{},
	}

	// Check for hypertension
	hasHypertension := factors.SBP >= 140 || factors.DBP >= 90
	hasSevereHypertension := factors.SBP >= 160 || factors.DBP >= 110

	// Check for severe features (ACOG 2020)
	if hasSevereHypertension {
		assessment.SevereFeatures = append(assessment.SevereFeatures, "Severe hypertension (≥160/110)")
	}
	if factors.Platelets < 100 {
		assessment.SevereFeatures = append(assessment.SevereFeatures, "Thrombocytopenia (<100,000)")
	}
	if factors.Creatinine > 1.1 {
		assessment.SevereFeatures = append(assessment.SevereFeatures, "Renal insufficiency (Cr >1.1)")
	}
	if factors.AST >= 70 || factors.ALT >= 70 {
		assessment.SevereFeatures = append(assessment.SevereFeatures, "Elevated liver enzymes (≥2x ULN)")
	}
	if factors.Headache {
		assessment.SevereFeatures = append(assessment.SevereFeatures, "Severe persistent headache")
	}
	if factors.VisualChanges {
		assessment.SevereFeatures = append(assessment.SevereFeatures, "Visual disturbances")
	}
	if factors.EpigastricPain {
		assessment.SevereFeatures = append(assessment.SevereFeatures, "Epigastric/RUQ pain")
	}

	// Classification
	if !hasHypertension {
		assessment.Classification = "NORMAL"
		assessment.UrgencyLevel = "ROUTINE"
		assessment.RiskScore = 0
	} else if !factors.Proteinuria && len(assessment.SevereFeatures) == 0 {
		assessment.Classification = "GESTATIONAL_HTN"
		assessment.UrgencyLevel = "ROUTINE"
		assessment.RiskScore = 2
	} else if len(assessment.SevereFeatures) > 0 {
		assessment.Classification = "SEVERE_PREECLAMPSIA"
		assessment.UrgencyLevel = "EMERGENT"
		assessment.RiskScore = 8 + len(assessment.SevereFeatures)
	} else {
		assessment.Classification = "PREECLAMPSIA"
		assessment.UrgencyLevel = "URGENT"
		assessment.RiskScore = 5
	}

	// Add recommendations
	assessment.addPreeclampsiaRecommendations()

	return assessment
}

func (a *PreeclampsiaAssessment) addPreeclampsiaRecommendations() {
	switch a.Classification {
	case "SEVERE_PREECLAMPSIA":
		a.Recommendations = []string{
			"Immediate hospitalization required",
			"Initiate magnesium sulfate for seizure prophylaxis",
			"Antihypertensive therapy if BP ≥160/110",
			"Evaluate for delivery - consider timing based on gestational age",
			"Continuous fetal monitoring",
			"Serial labs q6-12h (CBC, CMP, LDH)",
		}
	case "PREECLAMPSIA":
		a.Recommendations = []string{
			"Hospital admission for evaluation",
			"Serial BP monitoring",
			"24-hour urine protein or protein/creatinine ratio",
			"Fetal assessment (NST, BPP)",
			"Plan for delivery at 37 weeks if stable",
		}
	case "GESTATIONAL_HTN":
		a.Recommendations = []string{
			"Weekly BP monitoring",
			"Serial labs (weekly CBC, CMP)",
			"Patient education on warning signs",
			"Fetal growth assessment",
		}
	case "NORMAL":
		a.Recommendations = []string{
			"Continue routine prenatal care",
			"Monitor for development of hypertension",
		}
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// IsHighRiskPregnancy determines if pregnancy is high-risk based on conditions
func IsHighRiskPregnancy(ctx *PregnancyContext) bool {
	highRiskConditions := map[string]bool{
		"preeclampsia":          true,
		"gestational_diabetes":  true,
		"hellp_syndrome":        true,
		"placenta_previa":       true,
		"multiple_gestation":    true,
		"preterm_labor":         true,
		"iugr":                  true,
		"chronic_hypertension":  true,
	}

	for _, cond := range ctx.Conditions {
		if highRiskConditions[cond] {
			return true
		}
	}
	return false
}

// GetAvailablePregnancyTests returns list of tests with pregnancy-specific ranges
func GetAvailablePregnancyTests() []string {
	tests := make([]string, 0, len(PregnancyRanges))
	for code := range PregnancyRanges {
		tests = append(tests, code)
	}
	return tests
}
