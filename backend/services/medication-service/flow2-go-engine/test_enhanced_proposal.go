package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// Enhanced Proposal Generation Test
// This demonstrates the comprehensive clinical recommendation structure

func main() {
	fmt.Println("=== Enhanced Proposal Generation Demo ===")
	
	// Create a sample enhanced proposal based on the design
	proposal := createSampleEnhancedProposal()
	
	// Convert to JSON for display
	jsonData, err := json.MarshalIndent(proposal, "", "  ")
	if err != nil {
		log.Fatal("Error marshaling proposal:", err)
	}
	
	fmt.Println("\n=== COMPREHENSIVE CLINICAL RECOMMENDATION ===")
	fmt.Println(string(jsonData))
	
	// Demonstrate key features
	demonstrateKeyFeatures(proposal)
}

// EnhancedProposedOrder represents the comprehensive clinical recommendation structure
type EnhancedProposedOrder struct {
	ProposalID      string                    `json:"proposalId"`
	ProposalVersion string                    `json:"proposalVersion"`
	Timestamp       time.Time                 `json:"timestamp"`
	ExpiresAt       time.Time                 `json:"expiresAt"`
	Metadata        ProposalMetadata          `json:"metadata"`
	CalculatedOrder CalculatedOrder           `json:"calculatedOrder"`
	MonitoringPlan  EnhancedMonitoringPlan    `json:"monitoringPlan"`
	TherapeuticAlternatives TherapeuticAlternatives `json:"therapeuticAlternatives"`
	ClinicalRationale ClinicalRationale       `json:"clinicalRationale"`
	ProposalMetadata ProposalMetadataSection  `json:"proposalMetadata"`
}

type ProposalMetadata struct {
	PatientID           string  `json:"patientId"`
	EncounterID         string  `json:"encounterId"`
	PrescriberID        string  `json:"prescriberId"`
	Status              string  `json:"status"`
	Urgency             string  `json:"urgency"`
	ProposalType        string  `json:"proposalType"`
	RecipeUsed          string  `json:"recipeUsed"`
	ContextCompleteness float64 `json:"contextCompleteness"`
	ConfidenceScore     float64 `json:"confidenceScore"`
}

type CalculatedOrder struct {
	Medication        MedicationDetail        `json:"medication"`
	Dosing            DosingDetail            `json:"dosing"`
	CalculationDetails CalculationDetails     `json:"calculationDetails"`
	Formulation       FormulationDetail       `json:"formulation"`
}

type MedicationDetail struct {
	PrimaryIdentifier    Identifier   `json:"primaryIdentifier"`
	AlternateIdentifiers []Identifier `json:"alternateIdentifiers"`
	BrandName            *string      `json:"brandName"`
	GenericName          string       `json:"genericName"`
	TherapeuticClass     string       `json:"therapeuticClass"`
	IsHighAlert          bool         `json:"isHighAlert"`
	IsControlled         bool         `json:"isControlled"`
}

type Identifier struct {
	System  string `json:"system"`
	Code    string `json:"code"`
	Display string `json:"display"`
}

type DosingDetail struct {
	Dose         DoseInfo         `json:"dose"`
	Route        RouteInfo        `json:"route"`
	Frequency    FrequencyInfo    `json:"frequency"`
	Duration     DurationInfo     `json:"duration"`
	Instructions InstructionInfo  `json:"instructions"`
}

type DoseInfo struct {
	Value   float64 `json:"value"`
	Unit    string  `json:"unit"`
	PerDose bool    `json:"perDose"`
}

type RouteInfo struct {
	Code    string `json:"code"`
	Display string `json:"display"`
}

type FrequencyInfo struct {
	Code          string   `json:"code"`
	Display       string   `json:"display"`
	TimesPerDay   int      `json:"timesPerDay"`
	SpecificTimes []string `json:"specificTimes"`
}

type DurationInfo struct {
	Value   int    `json:"value"`
	Unit    string `json:"unit"`
	Refills int    `json:"refills"`
}

type InstructionInfo struct {
	PatientInstructions     string   `json:"patientInstructions"`
	PharmacyInstructions    string   `json:"pharmacyInstructions"`
	AdditionalInstructions  []string `json:"additionalInstructions"`
}

type CalculationDetails struct {
	Method           string                 `json:"method"`
	Factors          CalculationFactors     `json:"factors"`
	Adjustments      []string               `json:"adjustments"`
	RoundingApplied  bool                   `json:"roundingApplied"`
	MaximumDoseCheck MaximumDoseCheck       `json:"maximumDoseCheck"`
}

type CalculationFactors struct {
	PatientWeight  float64      `json:"patientWeight"`
	PatientAge     int          `json:"patientAge"`
	RenalFunction  RenalFunction `json:"renalFunction"`
}

type RenalFunction struct {
	EGFR     float64 `json:"eGFR"`
	Category string  `json:"category"`
}

type MaximumDoseCheck struct {
	Daily        float64 `json:"daily"`
	Maximum      float64 `json:"maximum"`
	WithinLimits bool    `json:"withinLimits"`
}

type FormulationDetail struct {
	SelectedForm             string                    `json:"selectedForm"`
	AvailableStrengths       []float64                 `json:"availableStrengths"`
	Splittable               bool                      `json:"splittable"`
	Crushable                bool                      `json:"crushable"`
	AlternativeFormulations  []AlternativeFormulation `json:"alternativeFormulations"`
}

type AlternativeFormulation struct {
	Form         string    `json:"form"`
	Strengths    []float64 `json:"strengths"`
	ClinicalNote string    `json:"clinicalNote"`
}

type EnhancedMonitoringPlan struct {
	RiskStratification RiskStratification    `json:"riskStratification"`
	Baseline          []BaselineMonitoring  `json:"baseline"`
	Ongoing           []OngoingMonitoring   `json:"ongoing"`
	SymptomMonitoring []SymptomMonitoring   `json:"symptomMonitoring"`
}

type RiskStratification struct {
	OverallRisk string       `json:"overallRisk"`
	Factors     []RiskFactor `json:"factors"`
}

type RiskFactor struct {
	Factor  string `json:"factor"`
	Present bool   `json:"present"`
	Impact  string `json:"impact"`
}

type BaselineMonitoring struct {
	Parameter      string         `json:"parameter"`
	LOINC          string         `json:"loinc"`
	Timing         string         `json:"timing"`
	Priority       string         `json:"priority"`
	Rationale      string         `json:"rationale"`
	CriticalValues CriticalValues `json:"criticalValues"`
}

type CriticalValues struct {
	Contraindicated string `json:"contraindicated"`
	CautionRequired string `json:"cautionRequired"`
	Normal          string `json:"normal"`
}

type OngoingMonitoring struct {
	Parameter        string             `json:"parameter"`
	Frequency        MonitoringFrequency `json:"frequency"`
	Rationale        string             `json:"rationale"`
	ActionThresholds []ActionThreshold  `json:"actionThresholds"`
	TargetRange      *TargetRange       `json:"targetRange,omitempty"`
}

type MonitoringFrequency struct {
	Interval   int                      `json:"interval"`
	Unit       string                   `json:"unit"`
	Conditions []FrequencyCondition     `json:"conditions"`
}

type FrequencyCondition struct {
	Condition         string              `json:"condition"`
	ModifiedFrequency MonitoringFrequency `json:"modifiedFrequency"`
}

type ActionThreshold struct {
	Value   string `json:"value"`
	Action  string `json:"action"`
	Urgency string `json:"urgency"`
}

type TargetRange struct {
	Min  float64 `json:"min"`
	Max  float64 `json:"max"`
	Unit string  `json:"unit"`
}

type SymptomMonitoring struct {
	Symptom           string   `json:"symptom"`
	Frequency         string   `json:"frequency"`
	EducationProvided string   `json:"educationProvided"`
	RedFlags          []string `json:"redFlags,omitempty"`
}

type TherapeuticAlternatives struct {
	PrimaryReason        string                    `json:"primaryReason"`
	Alternatives         []TherapeuticAlternative  `json:"alternatives"`
	NonPharmAlternatives []NonPharmAlternative     `json:"nonPharmAlternatives"`
}

type TherapeuticAlternative struct {
	Medication              AlternativeMedicationDetail `json:"medication"`
	Category                string                      `json:"category"`
	FormularyStatus         FormularyStatus             `json:"formularyStatus"`
	CostComparison          CostComparison              `json:"costComparison"`
	ClinicalConsiderations  ClinicalConsiderations      `json:"clinicalConsiderations"`
	SwitchingInstructions   string                      `json:"switchingInstructions"`
	Evidence                *AlternativeEvidence        `json:"evidence,omitempty"`
}

type AlternativeMedicationDetail struct {
	Name     string  `json:"name"`
	Code     string  `json:"code"`
	Strength float64 `json:"strength"`
	Unit     string  `json:"unit"`
}

type FormularyStatus struct {
	Tier              int     `json:"tier"`
	PriorAuthRequired bool    `json:"priorAuthRequired"`
	QuantityLimits    *string `json:"quantityLimits"`
}

type CostComparison struct {
	RelativeCost         string  `json:"relativeCost"`
	EstimatedMonthlyCost float64 `json:"estimatedMonthlyCost"`
	PatientCopay         float64 `json:"patientCopay"`
}

type ClinicalConsiderations struct {
	Advantages        []string `json:"advantages"`
	Disadvantages     []string `json:"disadvantages"`
	Contraindications []string `json:"contraindications,omitempty"`
}

type AlternativeEvidence struct {
	ComparativeEffectiveness string   `json:"comparativeEffectiveness"`
	GuidelinePosition        string   `json:"guidelinePosition"`
	References               []string `json:"references"`
}

type NonPharmAlternative struct {
	Intervention   string   `json:"intervention"`
	Components     []string `json:"components"`
	Effectiveness  string   `json:"effectiveness"`
	Recommendation string   `json:"recommendation"`
}

type ClinicalRationale struct {
	Summary               RationaleSummary        `json:"summary"`
	IndicationAssessment  IndicationAssessment    `json:"indicationAssessment"`
	DosingRationale       DosingRationale         `json:"dosingRationale"`
	FormularyRationale    FormularyRationale      `json:"formularyRationale"`
	PatientFactors        PatientFactors          `json:"patientFactors"`
	QualityMeasures       QualityMeasures         `json:"qualityMeasures"`
}

type RationaleSummary struct {
	Decision   string `json:"decision"`
	Confidence string `json:"confidence"`
	Complexity string `json:"complexity"`
}

type IndicationAssessment struct {
	PrimaryIndication string             `json:"primaryIndication"`
	ICDCode           string             `json:"icdCode"`
	ClinicalCriteria  []ClinicalCriterion `json:"clinicalCriteria"`
	Appropriateness   string             `json:"appropriateness"`
}

type ClinicalCriterion struct {
	Criterion string  `json:"criterion"`
	Met       bool    `json:"met"`
	Value     *string `json:"value,omitempty"`
}

type DosingRationale struct {
	Strategy     string         `json:"strategy"`
	Explanation  string         `json:"explanation"`
	TitrationPlan TitrationPlan  `json:"titrationPlan"`
	EvidenceBase EvidenceBase   `json:"evidenceBase"`
}

type TitrationPlan struct {
	Week2   string `json:"week2"`
	Week4   string `json:"week4"`
	MaxDose string `json:"maxDose"`
}

type EvidenceBase struct {
	Source                 string `json:"source"`
	RecommendationStrength string `json:"recommendationStrength"`
	EvidenceQuality        string `json:"evidenceQuality"`
}

type FormularyRationale struct {
	FormularyDecision   string            `json:"formularyDecision"`
	CostEffectiveness   string            `json:"costEffectiveness"`
	InsuranceCoverage   InsuranceCoverage `json:"insuranceCoverage"`
}

type InsuranceCoverage struct {
	Covered           bool    `json:"covered"`
	Tier              int     `json:"tier"`
	Copay             float64 `json:"copay"`
	DeductibleApplies bool    `json:"deductibleApplies"`
}

type PatientFactors struct {
	PositiveFactors      []string              `json:"positiveFactors"`
	Considerations       []string              `json:"considerations"`
	SharedDecisionMaking SharedDecisionMaking  `json:"sharedDecisionMaking"`
}

type SharedDecisionMaking struct {
	Discussed         []string `json:"discussed"`
	PatientPreference string   `json:"patientPreference"`
}

type QualityMeasures struct {
	AlignedMeasures []QualityMeasure `json:"alignedMeasures"`
}

type QualityMeasure struct {
	Measure   string `json:"measure"`
	NQFNumber string `json:"nqfNumber"`
	Impact    string `json:"impact"`
}

type ProposalMetadataSection struct {
	ClinicalReferences []ClinicalReference `json:"clinicalReferences"`
	AuditTrail         AuditTrail          `json:"auditTrail"`
	NextSteps          []NextStep          `json:"nextSteps"`
}

type ClinicalReference struct {
	Type     string `json:"type"`
	Citation string `json:"citation"`
	URL      string `json:"url"`
}

type AuditTrail struct {
	CalculationTime     int64            `json:"calculationTime"`
	ContextFetchTime    int64            `json:"contextFetchTime"`
	TotalProcessingTime int64            `json:"totalProcessingTime"`
	CacheUtilization    CacheUtilization `json:"cacheUtilization"`
}

type CacheUtilization struct {
	FormularyCache       string `json:"formularyCache"`
	DoseCalculationCache string `json:"doseCalculationCache"`
	MonitoringCache      string `json:"monitoringCache"`
}

type NextStep struct {
	Step           string   `json:"step"`
	Service        string   `json:"service"`
	Optional       bool     `json:"optional"`
	Reason         string   `json:"reason"`
	RequiredChecks []string `json:"requiredChecks,omitempty"`
}

// createSampleEnhancedProposal creates a comprehensive sample proposal
func createSampleEnhancedProposal() *EnhancedProposedOrder {
	now := time.Now()
	hba1cValue := "7.8%"

	return &EnhancedProposedOrder{
		ProposalID:      "prop-uuid-112233",
		ProposalVersion: "1.0",
		Timestamp:       now,
		ExpiresAt:       now.Add(24 * time.Hour),

		Metadata: ProposalMetadata{
			PatientID:           "PriyaSharma-456",
			EncounterID:         "enc-789",
			PrescriberID:        "dr-smith-123",
			Status:              "PROPOSED",
			Urgency:             "ROUTINE",
			ProposalType:        "NEW_PRESCRIPTION",
			RecipeUsed:          "business-standard-dose-calc-v1.0",
			ContextCompleteness: 0.92,
			ConfidenceScore:     0.95,
		},

		CalculatedOrder: CalculatedOrder{
			Medication: MedicationDetail{
				PrimaryIdentifier: Identifier{
					System:  "RxNorm",
					Code:    "860975",
					Display: "Metformin 500 MG Oral Tablet",
				},
				AlternateIdentifiers: []Identifier{
					{
						System: "NDC",
						Code:   "00378-6100-01",
					},
				},
				BrandName:        nil,
				GenericName:      "Metformin",
				TherapeuticClass: "Biguanides",
				IsHighAlert:      false,
				IsControlled:     false,
			},
			Dosing: DosingDetail{
				Dose: DoseInfo{
					Value:   500,
					Unit:    "mg",
					PerDose: true,
				},
				Route: RouteInfo{
					Code:    "PO",
					Display: "Oral",
				},
				Frequency: FrequencyInfo{
					Code:          "DAILY",
					Display:       "Once daily",
					TimesPerDay:   1,
					SpecificTimes: []string{"08:00"},
				},
				Duration: DurationInfo{
					Value:   90,
					Unit:    "days",
					Refills: 3,
				},
				Instructions: InstructionInfo{
					PatientInstructions:  "Take 1 tablet by mouth once daily with breakfast",
					PharmacyInstructions: "Dispense 90 tablets",
					AdditionalInstructions: []string{
						"Take with food to minimize GI upset",
						"If a dose is missed, take as soon as remembered unless it's almost time for the next dose",
					},
				},
			},
			CalculationDetails: CalculationDetails{
				Method: "STANDARD_DOSING",
				Factors: CalculationFactors{
					PatientWeight: 65,
					PatientAge:    45,
					RenalFunction: RenalFunction{
						EGFR:     85,
						Category: "G2",
					},
				},
				Adjustments:     []string{},
				RoundingApplied: false,
				MaximumDoseCheck: MaximumDoseCheck{
					Daily:        500,
					Maximum:      2000,
					WithinLimits: true,
				},
			},
			Formulation: FormulationDetail{
				SelectedForm:       "IMMEDIATE_RELEASE_TABLET",
				AvailableStrengths: []float64{500, 850, 1000},
				Splittable:         true,
				Crushable:          false,
				AlternativeFormulations: []AlternativeFormulation{
					{
						Form:         "EXTENDED_RELEASE_TABLET",
						Strengths:    []float64{500, 750, 1000},
						ClinicalNote: "Consider for improved adherence with once-daily dosing",
					},
				},
			},
		},

		MonitoringPlan: EnhancedMonitoringPlan{
			RiskStratification: RiskStratification{
				OverallRisk: "MODERATE",
				Factors: []RiskFactor{
					{
						Factor:  "Age > 65",
						Present: false,
						Impact:  "NONE",
					},
					{
						Factor:  "Renal impairment",
						Present: false,
						Impact:  "NONE",
					},
				},
			},
			Baseline: []BaselineMonitoring{
				{
					Parameter: "eGFR",
					LOINC:     "48642-3",
					Timing:    "BEFORE_INITIATION",
					Priority:  "REQUIRED",
					Rationale: "To establish baseline renal function before starting Metformin",
					CriticalValues: CriticalValues{
						Contraindicated: "< 30",
						CautionRequired: "30-45",
						Normal:          "> 45",
					},
				},
				{
					Parameter: "Vitamin B12",
					LOINC:     "2132-9",
					Timing:    "WITHIN_3_MONTHS",
					Priority:  "RECOMMENDED",
					Rationale: "To establish baseline due to risk of deficiency with long-term use",
					CriticalValues: CriticalValues{
						Contraindicated: "",
						CautionRequired: "",
						Normal:          "",
					},
				},
				{
					Parameter: "HbA1c",
					LOINC:     "4548-4",
					Timing:    "WITHIN_3_MONTHS",
					Priority:  "REQUIRED",
					Rationale: "To establish baseline glycemic control",
					CriticalValues: CriticalValues{
						Contraindicated: "",
						CautionRequired: "",
						Normal:          "",
					},
				},
			},
			Ongoing: []OngoingMonitoring{
				{
					Parameter: "eGFR",
					Frequency: MonitoringFrequency{
						Interval: 12,
						Unit:     "months",
						Conditions: []FrequencyCondition{
							{
								Condition: "eGFR < 60",
								ModifiedFrequency: MonitoringFrequency{
									Interval: 6,
									Unit:     "months",
								},
							},
						},
					},
					Rationale: "Monitor for changes in renal function",
					ActionThresholds: []ActionThreshold{
						{
							Value:   "< 45",
							Action:  "DOSE_REDUCTION",
							Urgency: "ROUTINE",
						},
						{
							Value:   "< 30",
							Action:  "DISCONTINUE",
							Urgency: "URGENT",
						},
					},
				},
				{
					Parameter: "HbA1c",
					Frequency: MonitoringFrequency{
						Interval: 3,
						Unit:     "months",
					},
					Rationale: "Monitor glycemic control",
					TargetRange: &TargetRange{
						Min:  6.5,
						Max:  7.0,
						Unit: "%",
					},
				},
			},
			SymptomMonitoring: []SymptomMonitoring{
				{
					Symptom:           "GI upset",
					Frequency:         "At each visit",
					EducationProvided: "Common initially, usually improves with time",
				},
				{
					Symptom:   "Signs of lactic acidosis",
					Frequency: "Patient education",
					RedFlags:  []string{"Unusual muscle pain", "Difficulty breathing", "Severe fatigue"},
				},
			},
		},

		TherapeuticAlternatives: TherapeuticAlternatives{
			PrimaryReason: "FORMULARY_OPTIMIZATION",
			Alternatives: []TherapeuticAlternative{
				{
					Medication: AlternativeMedicationDetail{
						Name:     "Metformin Extended Release",
						Code:     "RxNorm:861787",
						Strength: 500,
						Unit:     "mg",
					},
					Category: "SAME_DRUG_DIFFERENT_FORMULATION",
					FormularyStatus: FormularyStatus{
						Tier:              1,
						PriorAuthRequired: false,
						QuantityLimits:    nil,
					},
					CostComparison: CostComparison{
						RelativeCost:         "SIMILAR",
						EstimatedMonthlyCost: 10.00,
						PatientCopay:         5.00,
					},
					ClinicalConsiderations: ClinicalConsiderations{
						Advantages: []string{
							"Once daily dosing may improve adherence",
							"Potentially fewer GI side effects",
						},
						Disadvantages: []string{
							"Cannot be crushed or split",
							"Slightly higher cost",
						},
					},
					SwitchingInstructions: "Direct substitution at same daily dose",
				},
				{
					Medication: AlternativeMedicationDetail{
						Name:     "Gliclazide",
						Code:     "RxNorm:4821",
						Strength: 80,
						Unit:     "mg",
					},
					Category: "THERAPEUTIC_ALTERNATIVE",
					FormularyStatus: FormularyStatus{
						Tier:              2,
						PriorAuthRequired: false,
					},
					CostComparison: CostComparison{
						RelativeCost:         "HIGHER",
						EstimatedMonthlyCost: 25.00,
					},
					ClinicalConsiderations: ClinicalConsiderations{
						Advantages: []string{
							"Option if Metformin contraindicated",
							"No renal adjustment needed",
						},
						Disadvantages: []string{
							"Higher hypoglycemia risk",
							"Weight gain potential",
							"Requires glucose monitoring",
						},
						Contraindications: []string{"Sulfa allergy"},
					},
					Evidence: &AlternativeEvidence{
						ComparativeEffectiveness: "SIMILAR_A1C_REDUCTION",
						GuidelinePosition:        "SECOND_LINE",
						References:               []string{"ADA Standards of Care 2024"},
					},
				},
			},
			NonPharmAlternatives: []NonPharmAlternative{
				{
					Intervention:   "Lifestyle modification",
					Components:     []string{"Diet", "Exercise", "Weight loss"},
					Effectiveness:  "Can reduce A1c by 0.5-1.0%",
					Recommendation: "Should be continued regardless of medication",
				},
			},
		},

		ClinicalRationale: ClinicalRationale{
			Summary: RationaleSummary{
				Decision:   "Initiate Metformin 500mg daily for newly diagnosed Type 2 Diabetes",
				Confidence: "HIGH",
				Complexity: "LOW",
			},
			IndicationAssessment: IndicationAssessment{
				PrimaryIndication: "Type 2 Diabetes Mellitus",
				ICDCode:           "E11.9",
				ClinicalCriteria: []ClinicalCriterion{
					{
						Criterion: "HbA1c > 6.5%",
						Met:       true,
						Value:     &hba1cValue,
					},
					{
						Criterion: "Symptomatic hyperglycemia",
						Met:       false,
					},
				},
				Appropriateness: "FIRST_LINE_INDICATED",
			},
			DosingRationale: DosingRationale{
				Strategy:    "CONSERVATIVE_INITIATION",
				Explanation: "Starting at 500mg daily to minimize GI side effects",
				TitrationPlan: TitrationPlan{
					Week2:   "Increase to 500mg twice daily if tolerated",
					Week4:   "Increase to 1000mg twice daily if needed for glycemic control",
					MaxDose: "2000mg daily",
				},
				EvidenceBase: EvidenceBase{
					Source:                 "ADA/EASD Consensus Statement 2023",
					RecommendationStrength: "STRONG",
					EvidenceQuality:        "HIGH",
				},
			},
			FormularyRationale: FormularyRationale{
				FormularyDecision: "PREFERRED_DRUG_SELECTED",
				CostEffectiveness: "Metformin is the most cost-effective first-line agent",
				InsuranceCoverage: InsuranceCoverage{
					Covered:           true,
					Tier:              1,
					Copay:             5.00,
					DeductibleApplies: false,
				},
			},
			PatientFactors: PatientFactors{
				PositiveFactors: []string{
					"Normal renal function (eGFR 85)",
					"No contraindications",
					"Motivated for medication therapy",
				},
				Considerations: []string{
					"Patient prefers once-daily dosing",
					"Previous mild nausea with medications",
				},
				SharedDecisionMaking: SharedDecisionMaking{
					Discussed:         []string{"Benefits vs risks", "Alternative options", "Monitoring requirements"},
					PatientPreference: "Willing to try medication",
				},
			},
			QualityMeasures: QualityMeasures{
				AlignedMeasures: []QualityMeasure{
					{
						Measure:   "Diabetes: HbA1c Poor Control",
						NQFNumber: "0059",
						Impact:    "IMPROVES_PERFORMANCE",
					},
				},
			},
		},

		ProposalMetadata: ProposalMetadataSection{
			ClinicalReferences: []ClinicalReference{
				{
					Type:     "GUIDELINE",
					Citation: "American Diabetes Association. Standards of Medical Care in Diabetes—2024",
					URL:      "https://doi.org/10.2337/dc24-S009",
				},
			},
			AuditTrail: AuditTrail{
				CalculationTime:     45,
				ContextFetchTime:    120,
				TotalProcessingTime: 200,
				CacheUtilization: CacheUtilization{
					FormularyCache:       "HIT",
					DoseCalculationCache: "MISS",
					MonitoringCache:      "HIT",
				},
			},
			NextSteps: []NextStep{
				{
					Step:           "SAFETY_VALIDATION",
					Service:        "Safety Gateway",
					RequiredChecks: []string{"Drug interactions", "Allergies", "Contraindications"},
				},
				{
					Step:     "PROVIDER_REVIEW",
					Optional: false,
					Reason:   "New medication initiation",
				},
			},
		},
	}
}

// demonstrateKeyFeatures shows the key benefits of the enhanced proposal
func demonstrateKeyFeatures(proposal *EnhancedProposedOrder) {
	fmt.Println("\n=== KEY FEATURES DEMONSTRATION ===")

	fmt.Printf("✅ Clinical Completeness: Confidence Score %.2f\n", proposal.Metadata.ConfidenceScore)
	fmt.Printf("✅ Risk-Stratified Monitoring: %s risk with %d baseline checks\n",
		proposal.MonitoringPlan.RiskStratification.OverallRisk,
		len(proposal.MonitoringPlan.Baseline))
	fmt.Printf("✅ Therapeutic Alternatives: %d alternatives provided\n",
		len(proposal.TherapeuticAlternatives.Alternatives))
	fmt.Printf("✅ Evidence-Based Rationale: %s confidence, %s complexity\n",
		proposal.ClinicalRationale.Summary.Confidence,
		proposal.ClinicalRationale.Summary.Complexity)
	fmt.Printf("✅ Quality Measures: %d measures aligned\n",
		len(proposal.ClinicalRationale.QualityMeasures.AlignedMeasures))
	fmt.Printf("✅ Processing Efficiency: %dms total processing time\n",
		proposal.ProposalMetadata.AuditTrail.TotalProcessingTime)

	fmt.Println("\n=== CLINICAL DECISION SUPPORT HIGHLIGHTS ===")
	fmt.Printf("🎯 Primary Recommendation: %s\n", proposal.ClinicalRationale.Summary.Decision)
	fmt.Printf("📊 Dose Calculation: %s with %s strategy\n",
		proposal.CalculatedOrder.CalculationDetails.Method,
		proposal.ClinicalRationale.DosingRationale.Strategy)
	fmt.Printf("🔍 Monitoring Plan: %d ongoing parameters\n", len(proposal.MonitoringPlan.Ongoing))
	fmt.Printf("💊 Alternative Options: %s primary reason\n", proposal.TherapeuticAlternatives.PrimaryReason)
	fmt.Printf("📋 Next Steps: %d workflow steps defined\n", len(proposal.ProposalMetadata.NextSteps))

	fmt.Println("\n=== COMPREHENSIVE CLINICAL INTELLIGENCE ACHIEVED ===")
	fmt.Println("This enhanced proposal transforms simple medication requests into")
	fmt.Println("comprehensive clinical recommendations - truly embodying the")
	fmt.Println("'Clinical Pharmacist's Digital Twin' concept!")
}
