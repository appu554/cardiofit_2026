// Package interpretation provides clinical interpretation algorithms
// hellp.go implements HELLP syndrome screening (Tennessee + Mississippi criteria)
package interpretation

// =============================================================================
// HELLP SYNDROME SCREENING (Tennessee 1990 + Mississippi 2006)
// =============================================================================

// HELLPScreening represents HELLP syndrome assessment results
type HELLPScreening struct {
	// Component Assessment
	Hemolysis     HELLPHemolysis   `json:"hemolysis"`
	ElevatedLiver HELLPLiver       `json:"elevatedLiver"`
	LowPlatelets  HELLPPlatelets   `json:"lowPlatelets"`

	// Classification
	Classification     string `json:"classification"`     // COMPLETE_HELLP, PARTIAL_HELLP, NO_HELLP
	MississippiClass   int    `json:"mississippiClass"`   // 1, 2, 3 (1 = most severe)
	TennesseeCriteria  bool   `json:"tennesseeCriteria"`  // Meets Tennessee criteria

	// Risk Assessment
	RiskScore       int      `json:"riskScore"`       // Composite severity score
	Urgency         string   `json:"urgency"`         // EMERGENT, URGENT, ROUTINE
	Recommendations []string `json:"recommendations"`

	// Governance
	Governance HELLPGovernance `json:"governance"`
}

// HELLPHemolysis represents hemolysis component assessment
type HELLPHemolysis struct {
	LDH           float64 `json:"ldh"`           // U/L
	TotalBili     float64 `json:"totalBili"`     // mg/dL
	Haptoglobin   float64 `json:"haptoglobin"`   // mg/dL (if available)
	Schistocytes  bool    `json:"schistocytes"`  // Peripheral smear finding
	Present       bool    `json:"present"`
	Criteria      string  `json:"criteria"`      // Description of findings
}

// HELLPLiver represents liver component assessment
type HELLPLiver struct {
	AST       float64 `json:"ast"`       // U/L
	ALT       float64 `json:"alt"`       // U/L
	Present   bool    `json:"present"`
	Criteria  string  `json:"criteria"`
}

// HELLPPlatelets represents platelet component assessment
type HELLPPlatelets struct {
	Count            float64 `json:"count"`            // x10^9/L
	MississippiClass int     `json:"mississippiClass"` // 1, 2, 3
	Present          bool    `json:"present"`
	Criteria         string  `json:"criteria"`
}

// HELLPGovernance tracks clinical authority for HELLP screening
type HELLPGovernance struct {
	TennesseeRef   string `json:"tennesseeRef"`
	MississippiRef string `json:"mississippiRef"`
	ACOGRef        string `json:"acogRef"`
	EvidenceLevel  string `json:"evidenceLevel"`
}

// =============================================================================
// HELLP DIAGNOSTIC THRESHOLDS
// =============================================================================

// Tennessee Criteria (Sibai 1990)
const (
	// Hemolysis
	TennesseeLDHThreshold      = 600.0  // U/L (or >2x ULN)
	TennesseeBiliThreshold     = 1.2    // mg/dL

	// Elevated Liver Enzymes
	TennesseeASTThreshold      = 70.0   // U/L (≥2x ULN)
	TennesseeALTThreshold      = 70.0   // U/L (≥2x ULN)

	// Low Platelets
	TennesseePlateletThreshold = 100.0  // x10^9/L
)

// Mississippi Classification (Martin 2006)
const (
	// Class 1: Most Severe
	MississippiClass1Platelets = 50.0   // <50 x10^9/L

	// Class 2: Moderate
	MississippiClass2PlateletsLow  = 50.0   // 50-100 x10^9/L
	MississippiClass2PlateletsHigh = 100.0

	// Class 3: Least Severe
	MississippiClass3PlateletsLow  = 100.0  // 100-150 x10^9/L
	MississippiClass3PlateletsHigh = 150.0

	// Additional Mississippi thresholds
	MississippiLDHThreshold = 600.0  // U/L
	MississippiASTThreshold = 40.0   // U/L (Mississippi uses lower threshold)
)

// =============================================================================
// HELLP SCREENING FUNCTION
// =============================================================================

// HELLPInput contains lab values for HELLP screening
type HELLPInput struct {
	LDH          float64 `json:"ldh"`          // U/L
	TotalBili    float64 `json:"totalBili"`    // mg/dL
	AST          float64 `json:"ast"`          // U/L
	ALT          float64 `json:"alt"`          // U/L
	Platelets    float64 `json:"platelets"`    // x10^9/L
	Haptoglobin  float64 `json:"haptoglobin"`  // mg/dL (optional, 0 if not available)
	Schistocytes bool    `json:"schistocytes"` // From peripheral smear
}

// ScreenHELLP performs HELLP syndrome screening using Tennessee + Mississippi criteria
func ScreenHELLP(input *HELLPInput) *HELLPScreening {
	result := &HELLPScreening{
		Governance: HELLPGovernance{
			TennesseeRef:   "Sibai BM et al. Am J Obstet Gynecol 1990;162:311-316",
			MississippiRef: "Martin JN et al. Am J Obstet Gynecol 2006;195:S59",
			ACOGRef:        "ACOG Practice Bulletin No. 222: Gestational Hypertension and Preeclampsia",
			EvidenceLevel:  "STRONG (1B)",
		},
	}

	// Assess each component
	result.assessHemolysis(input)
	result.assessLiver(input)
	result.assessPlatelets(input)

	// Determine classification
	result.classify()

	// Generate recommendations
	result.generateRecommendations()

	return result
}

// assessHemolysis evaluates hemolysis component
func (h *HELLPScreening) assessHemolysis(input *HELLPInput) {
	h.Hemolysis = HELLPHemolysis{
		LDH:          input.LDH,
		TotalBili:    input.TotalBili,
		Haptoglobin:  input.Haptoglobin,
		Schistocytes: input.Schistocytes,
	}

	// Tennessee criteria for hemolysis
	ldh_elevated := input.LDH >= TennesseeLDHThreshold
	bili_elevated := input.TotalBili >= TennesseeBiliThreshold
	haptoglobin_low := input.Haptoglobin > 0 && input.Haptoglobin < 25.0 // Low haptoglobin

	// Hemolysis present if LDH elevated AND (bili elevated OR schistocytes OR low haptoglobin)
	h.Hemolysis.Present = ldh_elevated && (bili_elevated || input.Schistocytes || haptoglobin_low)

	// Build criteria string
	criteria := []string{}
	if ldh_elevated {
		criteria = append(criteria, "LDH ≥600 U/L")
	}
	if bili_elevated {
		criteria = append(criteria, "Bilirubin ≥1.2 mg/dL")
	}
	if input.Schistocytes {
		criteria = append(criteria, "Schistocytes on smear")
	}
	if haptoglobin_low {
		criteria = append(criteria, "Low haptoglobin")
	}

	if h.Hemolysis.Present {
		h.Hemolysis.Criteria = "HEMOLYSIS PRESENT: " + joinCriteria(criteria)
	} else if len(criteria) > 0 {
		h.Hemolysis.Criteria = "Partial findings: " + joinCriteria(criteria)
	} else {
		h.Hemolysis.Criteria = "No hemolysis markers detected"
	}
}

// assessLiver evaluates liver enzyme component
func (h *HELLPScreening) assessLiver(input *HELLPInput) {
	h.ElevatedLiver = HELLPLiver{
		AST: input.AST,
		ALT: input.ALT,
	}

	// Tennessee criteria: AST or ALT ≥70 U/L (≥2x ULN)
	ast_elevated := input.AST >= TennesseeASTThreshold
	alt_elevated := input.ALT >= TennesseeALTThreshold

	h.ElevatedLiver.Present = ast_elevated || alt_elevated

	if h.ElevatedLiver.Present {
		if ast_elevated && alt_elevated {
			h.ElevatedLiver.Criteria = "ELEVATED LIVER: Both AST and ALT ≥70 U/L"
		} else if ast_elevated {
			h.ElevatedLiver.Criteria = "ELEVATED LIVER: AST ≥70 U/L"
		} else {
			h.ElevatedLiver.Criteria = "ELEVATED LIVER: ALT ≥70 U/L"
		}
	} else if input.AST >= MississippiASTThreshold || input.ALT >= MississippiASTThreshold {
		h.ElevatedLiver.Criteria = "Mildly elevated (Mississippi criteria): AST/ALT ≥40 U/L"
	} else {
		h.ElevatedLiver.Criteria = "Liver enzymes within normal limits"
	}
}

// assessPlatelets evaluates platelet component with Mississippi classification
func (h *HELLPScreening) assessPlatelets(input *HELLPInput) {
	h.LowPlatelets = HELLPPlatelets{
		Count: input.Platelets,
	}

	// Tennessee criteria: Platelets <100
	h.LowPlatelets.Present = input.Platelets < TennesseePlateletThreshold

	// Mississippi Classification
	if input.Platelets < MississippiClass1Platelets {
		h.LowPlatelets.MississippiClass = 1
		h.LowPlatelets.Criteria = "SEVERE: Mississippi Class 1 (Platelets <50)"
	} else if input.Platelets < MississippiClass2PlateletsHigh {
		h.LowPlatelets.MississippiClass = 2
		h.LowPlatelets.Criteria = "MODERATE: Mississippi Class 2 (Platelets 50-100)"
	} else if input.Platelets < MississippiClass3PlateletsHigh {
		h.LowPlatelets.MississippiClass = 3
		h.LowPlatelets.Criteria = "MILD: Mississippi Class 3 (Platelets 100-150)"
	} else {
		h.LowPlatelets.MississippiClass = 0
		h.LowPlatelets.Criteria = "Normal platelet count (>150)"
	}

	h.MississippiClass = h.LowPlatelets.MississippiClass
}

// classify determines overall HELLP classification
func (h *HELLPScreening) classify() {
	componentCount := 0
	if h.Hemolysis.Present {
		componentCount++
	}
	if h.ElevatedLiver.Present {
		componentCount++
	}
	if h.LowPlatelets.Present {
		componentCount++
	}

	// Calculate risk score
	h.RiskScore = componentCount * 3
	if h.MississippiClass == 1 {
		h.RiskScore += 4
	} else if h.MississippiClass == 2 {
		h.RiskScore += 2
	}

	// Tennessee criteria: All 3 components must be present for complete HELLP
	h.TennesseeCriteria = componentCount == 3

	// Classification
	if componentCount == 3 {
		h.Classification = "COMPLETE_HELLP"
		h.Urgency = "EMERGENT"
	} else if componentCount >= 1 {
		h.Classification = "PARTIAL_HELLP"
		if h.MississippiClass == 1 || componentCount == 2 {
			h.Urgency = "URGENT"
		} else {
			h.Urgency = "ROUTINE"
		}
	} else {
		h.Classification = "NO_HELLP"
		h.Urgency = "ROUTINE"
	}
}

// generateRecommendations creates clinical recommendations based on classification
func (h *HELLPScreening) generateRecommendations() {
	h.Recommendations = []string{}

	switch h.Classification {
	case "COMPLETE_HELLP":
		h.Recommendations = []string{
			"IMMEDIATE HOSPITALIZATION REQUIRED",
			"Initiate magnesium sulfate for seizure prophylaxis",
			"Aggressive blood pressure control if SBP ≥160 or DBP ≥110",
			"Type and screen - prepare for possible transfusion",
			"Corticosteroids if <34 weeks for fetal lung maturity",
			"Plan delivery: Complete HELLP is indication for delivery regardless of gestational age",
			"Serial labs q6h: CBC, CMP, LDH, coagulation studies",
			"Consider ICU admission for close monitoring",
			"Notify blood bank, anesthesia, and NICU",
		}

		// Additional recommendations based on Mississippi class
		if h.MississippiClass == 1 {
			h.Recommendations = append(h.Recommendations,
				"CRITICAL: Platelets <50 - high bleeding risk",
				"Consider platelet transfusion before delivery or procedures",
				"Avoid regional anesthesia until platelets >75")
		}

	case "PARTIAL_HELLP":
		h.Recommendations = []string{
			"Hospital admission for observation and monitoring",
			"Serial labs q6-12h to monitor for progression",
			"Blood pressure monitoring q4h",
			"Fetal surveillance (NST, BPP)",
			"Consider corticosteroids if <34 weeks",
			"Plan for delivery at 34-37 weeks depending on stability",
			"Monitor for development of complete HELLP",
		}

		// Specific component recommendations
		if h.Hemolysis.Present && !h.ElevatedLiver.Present {
			h.Recommendations = append(h.Recommendations,
				"Evaluate for other causes of hemolysis (TTP, HUS, aHUS)")
		}
		if h.ElevatedLiver.Present && !h.Hemolysis.Present {
			h.Recommendations = append(h.Recommendations,
				"Evaluate for acute fatty liver of pregnancy (AFLP)",
				"Check ammonia, glucose, fibrinogen")
		}

	case "NO_HELLP":
		h.Recommendations = []string{
			"HELLP syndrome not diagnosed at this time",
			"If clinical suspicion persists, repeat labs in 6-12 hours",
			"Continue monitoring for preeclampsia features",
			"Patient education on warning symptoms",
		}
	}

	// Universal recommendations
	h.Recommendations = append(h.Recommendations,
		"Document discussion with patient about diagnosis and plan",
		"Notify obstetric attending and consulting services as needed")
}

// =============================================================================
// DIFFERENTIAL DIAGNOSIS SUPPORT
// =============================================================================

// HELLPDifferential contains differential diagnosis information
type HELLPDifferential struct {
	Diagnosis       string   `json:"diagnosis"`
	Likelihood      string   `json:"likelihood"`  // HIGH, MODERATE, LOW
	KeyFeatures     []string `json:"keyFeatures"`
	DistinguishFrom string   `json:"distinguishFrom"` // How to distinguish from HELLP
}

// GetHELLPDifferentials returns differential diagnoses to consider
func GetHELLPDifferentials(screening *HELLPScreening) []HELLPDifferential {
	differentials := []HELLPDifferential{
		{
			Diagnosis:       "Acute Fatty Liver of Pregnancy (AFLP)",
			Likelihood:      "MODERATE",
			KeyFeatures:     []string{"Hypoglycemia", "Hyperammonemia", "Coagulopathy (low fibrinogen)", "Encephalopathy"},
			DistinguishFrom: "Check glucose, ammonia, fibrinogen, PT/INR. AFLP has more pronounced synthetic dysfunction",
		},
		{
			Diagnosis:       "Thrombotic Thrombocytopenic Purpura (TTP)",
			Likelihood:      "LOW",
			KeyFeatures:     []string{"Fever", "Neurologic changes", "Severe thrombocytopenia (<20)", "ADAMTS13 <10%"},
			DistinguishFrom: "TTP has more severe thrombocytopenia, less liver involvement. Check ADAMTS13 activity",
		},
		{
			Diagnosis:       "Hemolytic Uremic Syndrome (HUS)",
			Likelihood:      "LOW",
			KeyFeatures:     []string{"Renal failure predominant", "Diarrheal prodrome (typical)", "Complement abnormalities (atypical)"},
			DistinguishFrom: "HUS has predominant renal involvement. Check complement levels if suspected aHUS",
		},
		{
			Diagnosis:       "Systemic Lupus Erythematosus (SLE) Flare",
			Likelihood:      "LOW",
			KeyFeatures:     []string{"History of SLE", "Low C3/C4", "Positive dsDNA", "Multi-organ involvement"},
			DistinguishFrom: "Check ANA, dsDNA, complement levels. History crucial",
		},
		{
			Diagnosis:       "Severe Preeclampsia Without HELLP",
			Likelihood:      "HIGH",
			KeyFeatures:     []string{"Hypertension", "Proteinuria", "May have isolated component abnormalities"},
			DistinguishFrom: "HELLP requires all 3 components. Severe PE may have 1-2 abnormalities",
		},
	}

	// Adjust likelihood based on screening results
	if screening.ElevatedLiver.Present && !screening.Hemolysis.Present {
		for i := range differentials {
			if differentials[i].Diagnosis == "Acute Fatty Liver of Pregnancy (AFLP)" {
				differentials[i].Likelihood = "HIGH"
			}
		}
	}

	if screening.LowPlatelets.MississippiClass == 1 && !screening.ElevatedLiver.Present {
		for i := range differentials {
			if differentials[i].Diagnosis == "Thrombotic Thrombocytopenic Purpura (TTP)" {
				differentials[i].Likelihood = "MODERATE"
			}
		}
	}

	return differentials
}

// =============================================================================
// MONITORING AND FOLLOW-UP
// =============================================================================

// HELLPMonitoringProtocol provides monitoring recommendations
type HELLPMonitoringProtocol struct {
	LabFrequency       string   `json:"labFrequency"`
	VitalFrequency     string   `json:"vitalFrequency"`
	FetalMonitoring    string   `json:"fetalMonitoring"`
	EscalationCriteria []string `json:"escalationCriteria"`
	DeliveryTiming     string   `json:"deliveryTiming"`
}

// GetMonitoringProtocol returns appropriate monitoring protocol
func GetMonitoringProtocol(screening *HELLPScreening) *HELLPMonitoringProtocol {
	protocol := &HELLPMonitoringProtocol{}

	switch screening.Classification {
	case "COMPLETE_HELLP":
		protocol.LabFrequency = "CBC, CMP, LDH every 6 hours"
		protocol.VitalFrequency = "Continuous BP monitoring, neuro checks every 2 hours"
		protocol.FetalMonitoring = "Continuous fetal heart rate monitoring"
		protocol.EscalationCriteria = []string{
			"Platelets falling >50% in 24h",
			"LDH increasing despite treatment",
			"Development of DIC (falling fibrinogen, rising D-dimer)",
			"Neurologic symptoms (headache, visual changes, seizure)",
			"Renal function decline (Cr >1.5)",
			"Pulmonary edema or respiratory distress",
		}
		protocol.DeliveryTiming = "Delivery indicated once maternal stabilization achieved (usually within 24-48 hours)"

	case "PARTIAL_HELLP":
		protocol.LabFrequency = "CBC, CMP, LDH every 6-12 hours"
		protocol.VitalFrequency = "BP every 4 hours, daily weights, strict I/O"
		protocol.FetalMonitoring = "NST every 8-12 hours, daily BPP"
		protocol.EscalationCriteria = []string{
			"Development of third HELLP component",
			"Worsening of existing abnormalities",
			"Severe hypertension (≥160/110)",
			"Symptoms of severe preeclampsia",
			"Nonreassuring fetal status",
		}
		protocol.DeliveryTiming = "Delivery at 34-37 weeks depending on disease trajectory and fetal status"

	case "NO_HELLP":
		protocol.LabFrequency = "Repeat labs in 6-12 hours if clinical suspicion, otherwise per preeclampsia protocol"
		protocol.VitalFrequency = "BP every 4-6 hours"
		protocol.FetalMonitoring = "Per standard preeclampsia protocol"
		protocol.EscalationCriteria = []string{
			"Development of any HELLP component",
			"Clinical deterioration",
		}
		protocol.DeliveryTiming = "Per underlying diagnosis (preeclampsia, gestational HTN)"
	}

	return protocol
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func joinCriteria(criteria []string) string {
	if len(criteria) == 0 {
		return ""
	}
	result := criteria[0]
	for i := 1; i < len(criteria); i++ {
		result += ", " + criteria[i]
	}
	return result
}

// IsHELLPEmergency returns true if immediate intervention required
func IsHELLPEmergency(screening *HELLPScreening) bool {
	return screening.Classification == "COMPLETE_HELLP" || screening.MississippiClass == 1
}

// CalculateHELLPSeverityScore provides a numeric severity score
func CalculateHELLPSeverityScore(screening *HELLPScreening) int {
	score := 0

	// Hemolysis severity
	if screening.Hemolysis.Present {
		score += 2
		if screening.Hemolysis.LDH >= 1000 {
			score += 2 // Very high LDH
		}
	}

	// Liver severity
	if screening.ElevatedLiver.Present {
		score += 2
		if screening.ElevatedLiver.AST >= 200 || screening.ElevatedLiver.ALT >= 200 {
			score += 2 // Very high transaminases
		}
	}

	// Platelet severity (Mississippi-based)
	switch screening.MississippiClass {
	case 1:
		score += 4
	case 2:
		score += 2
	case 3:
		score += 1
	}

	return score
}
