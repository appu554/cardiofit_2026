package fixtures

// ============================================================================
// CDSS Integration Test Fixtures
// ============================================================================
// Comprehensive clinical test data for validating the KB-7 CDSS evaluation
// pipeline including FHIR parsing, fact building, THREE-CHECK pipeline,
// rule engine, and alert generation.
// ============================================================================

import "time"

// ============================================================================
// Clinical Patient Scenarios
// ============================================================================

// SepsisPatientBundle - Test 7.1: Sepsis Patient with Elevated Lactate
// Full FHIR Bundle for a sepsis patient with multiple clinical indicators
var SepsisPatientBundle = map[string]interface{}{
	"resourceType": "Bundle",
	"type":         "collection",
	"entry": []map[string]interface{}{
		{
			"resource": map[string]interface{}{
				"resourceType": "Patient",
				"id":           "sepsis-patient-001",
				"birthDate":    "1960-05-20",
				"gender":       "female",
			},
		},
		{
			"resource": map[string]interface{}{
				"resourceType": "Condition",
				"id":           "sepsis-condition-001",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{
							"system":  "http://snomed.info/sct",
							"code":    "91302008",
							"display": "Sepsis",
						},
					},
				},
				"clinicalStatus": map[string]interface{}{
					"coding": []map[string]interface{}{
						{
							"system": "http://terminology.hl7.org/CodeSystem/condition-clinical",
							"code":   "active",
						},
					},
				},
			},
		},
		{
			"resource": map[string]interface{}{
				"resourceType": "Condition",
				"id":           "pneumonia-condition-001",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{
							"system":  "http://snomed.info/sct",
							"code":    "233604007",
							"display": "Pneumonia",
						},
					},
				},
				"clinicalStatus": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"code": "active"},
					},
				},
			},
		},
		{
			"resource": map[string]interface{}{
				"resourceType": "Observation",
				"id":           "lactate-obs-001",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{
							"system":  "http://loinc.org",
							"code":    "2524-7",
							"display": "Lactate",
						},
					},
				},
				"valueQuantity": map[string]interface{}{
					"value":  4.5,
					"unit":   "mmol/L",
					"system": "http://unitsofmeasure.org",
					"code":   "mmol/L",
				},
				"status": "final",
			},
		},
		{
			"resource": map[string]interface{}{
				"resourceType": "Observation",
				"id":           "bp-obs-001",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{
							"system":  "http://loinc.org",
							"code":    "8480-6",
							"display": "Systolic blood pressure",
						},
					},
				},
				"valueQuantity": map[string]interface{}{
					"value":  85,
					"unit":   "mmHg",
					"system": "http://unitsofmeasure.org",
					"code":   "mm[Hg]",
				},
				"status": "final",
			},
		},
		{
			"resource": map[string]interface{}{
				"resourceType": "MedicationRequest",
				"id":           "noradrenaline-001",
				"medicationCodeableConcept": map[string]interface{}{
					"coding": []map[string]interface{}{
						{
							"system":  "http://snomed.info/sct",
							"code":    "21611011000036109",
							"display": "Noradrenaline",
						},
					},
				},
				"status": "active",
			},
		},
	},
}

// DiabeticPatientBundle - Test 7.2: Diabetic Patient with AKI Risk
var DiabeticPatientBundle = map[string]interface{}{
	"resourceType": "Bundle",
	"type":         "collection",
	"entry": []map[string]interface{}{
		{
			"resource": map[string]interface{}{
				"resourceType": "Patient",
				"id":           "diabetic-patient-001",
				"birthDate":    "1952-08-10",
				"gender":       "male",
			},
		},
		{
			"resource": map[string]interface{}{
				"resourceType": "Condition",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{
							"system":  "http://snomed.info/sct",
							"code":    "44054006",
							"display": "Type 2 diabetes mellitus",
						},
					},
				},
				"clinicalStatus": map[string]interface{}{
					"coding": []map[string]interface{}{{"code": "active"}},
				},
			},
		},
		{
			"resource": map[string]interface{}{
				"resourceType": "Condition",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{
							"system":  "http://snomed.info/sct",
							"code":    "431856006",
							"display": "Chronic kidney disease stage 2",
						},
					},
				},
				"clinicalStatus": map[string]interface{}{
					"coding": []map[string]interface{}{{"code": "active"}},
				},
			},
		},
		{
			"resource": map[string]interface{}{
				"resourceType": "Observation",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{
							"system":  "http://loinc.org",
							"code":    "2160-0",
							"display": "Creatinine",
						},
					},
				},
				"valueQuantity": map[string]interface{}{
					"value": 1.8,
					"unit":  "mg/dL",
				},
				"status": "final",
			},
		},
		{
			"resource": map[string]interface{}{
				"resourceType": "Observation",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{
							"system":  "http://loinc.org",
							"code":    "4548-4",
							"display": "Hemoglobin A1c",
						},
					},
				},
				"valueQuantity": map[string]interface{}{
					"value": 9.2,
					"unit":  "%",
				},
				"status": "final",
			},
		},
		{
			"resource": map[string]interface{}{
				"resourceType": "MedicationRequest",
				"medicationCodeableConcept": map[string]interface{}{
					"coding": []map[string]interface{}{
						{
							"system":  "http://snomed.info/sct",
							"code":    "21411011000036105",
							"display": "Gentamicin 80 mg/2 mL injection",
						},
					},
				},
				"status": "active",
			},
		},
		{
			"resource": map[string]interface{}{
				"resourceType": "MedicationRequest",
				"medicationCodeableConcept": map[string]interface{}{
					"coding": []map[string]interface{}{
						{
							"system":  "http://snomed.info/sct",
							"code":    "21490011000036101",
							"display": "Ibuprofen 400 mg tablet",
						},
					},
				},
				"status": "active",
			},
		},
	},
}

// CardiacPatientBundle - Test 7.3: Cardiac Patient with Anticoagulation Decision
var CardiacPatientBundle = map[string]interface{}{
	"resourceType": "Bundle",
	"type":         "collection",
	"entry": []map[string]interface{}{
		{
			"resource": map[string]interface{}{
				"resourceType": "Patient",
				"id":           "cardiac-patient-001",
				"birthDate":    "1948-03-25",
				"gender":       "female",
			},
		},
		{
			"resource": map[string]interface{}{
				"resourceType": "Condition",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{
							"system":  "http://snomed.info/sct",
							"code":    "49436004",
							"display": "Atrial fibrillation",
						},
					},
				},
				"clinicalStatus": map[string]interface{}{
					"coding": []map[string]interface{}{{"code": "active"}},
				},
			},
		},
		{
			"resource": map[string]interface{}{
				"resourceType": "Condition",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{
							"system":  "http://snomed.info/sct",
							"code":    "59621000",
							"display": "Essential hypertension",
						},
					},
				},
				"clinicalStatus": map[string]interface{}{
					"coding": []map[string]interface{}{{"code": "active"}},
				},
			},
		},
		{
			"resource": map[string]interface{}{
				"resourceType": "Condition",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{
							"system":  "http://snomed.info/sct",
							"code":    "44054006",
							"display": "Type 2 diabetes mellitus",
						},
					},
				},
				"clinicalStatus": map[string]interface{}{
					"coding": []map[string]interface{}{{"code": "active"}},
				},
			},
		},
		{
			"resource": map[string]interface{}{
				"resourceType": "Observation",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{
							"system":  "http://loinc.org",
							"code":    "6301-6",
							"display": "INR",
						},
					},
				},
				"valueQuantity": map[string]interface{}{
					"value": 1.1,
					"unit":  "ratio",
				},
				"status": "final",
			},
		},
	},
}

// ComplexMultimorbidPatientBundle - Test 7.4: Complex Multi-Morbid Patient
var ComplexMultimorbidPatientBundle = map[string]interface{}{
	"resourceType": "Bundle",
	"type":         "collection",
	"entry": []map[string]interface{}{
		{
			"resource": map[string]interface{}{
				"resourceType": "Patient",
				"id":           "complex-patient-001",
				"birthDate":    "1945-11-12",
				"gender":       "male",
			},
		},
		// Multiple conditions
		{
			"resource": map[string]interface{}{
				"resourceType": "Condition",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://snomed.info/sct", "code": "44054006", "display": "Type 2 diabetes"},
					},
				},
				"clinicalStatus": map[string]interface{}{"coding": []map[string]interface{}{{"code": "active"}}},
			},
		},
		{
			"resource": map[string]interface{}{
				"resourceType": "Condition",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://snomed.info/sct", "code": "84114007", "display": "Heart failure"},
					},
				},
				"clinicalStatus": map[string]interface{}{"coding": []map[string]interface{}{{"code": "active"}}},
			},
		},
		{
			"resource": map[string]interface{}{
				"resourceType": "Condition",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://snomed.info/sct", "code": "433146000", "display": "CKD Stage 5"},
					},
				},
				"clinicalStatus": map[string]interface{}{"coding": []map[string]interface{}{{"code": "active"}}},
			},
		},
		{
			"resource": map[string]interface{}{
				"resourceType": "Condition",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://snomed.info/sct", "code": "49436004", "display": "Atrial fibrillation"},
					},
				},
				"clinicalStatus": map[string]interface{}{"coding": []map[string]interface{}{{"code": "active"}}},
			},
		},
		// Labs
		{
			"resource": map[string]interface{}{
				"resourceType": "Observation",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://loinc.org", "code": "2160-0", "display": "Creatinine"},
					},
				},
				"valueQuantity": map[string]interface{}{"value": 5.2, "unit": "mg/dL"},
				"status":        "final",
			},
		},
		{
			"resource": map[string]interface{}{
				"resourceType": "Observation",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://loinc.org", "code": "33762-6", "display": "NT-proBNP"},
					},
				},
				"valueQuantity": map[string]interface{}{"value": 8500, "unit": "pg/mL"},
				"status":        "final",
			},
		},
		{
			"resource": map[string]interface{}{
				"resourceType": "Observation",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://loinc.org", "code": "2339-0", "display": "Glucose"},
					},
				},
				"valueQuantity": map[string]interface{}{"value": 45, "unit": "mg/dL"},
				"status":        "final",
			},
		},
		// Medications
		{
			"resource": map[string]interface{}{
				"resourceType": "MedicationRequest",
				"medicationCodeableConcept": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://snomed.info/sct", "code": "21542011000036106", "display": "Lisinopril 10mg"},
					},
				},
				"status": "active",
			},
		},
		{
			"resource": map[string]interface{}{
				"resourceType": "MedicationRequest",
				"medicationCodeableConcept": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://snomed.info/sct", "code": "21122011000036101", "display": "Apixaban 5mg"},
					},
				},
				"status": "active",
			},
		},
	},
}

// ============================================================================
// CDSS Evaluate Request Structures
// ============================================================================

// SimpleCDSSRequest - Minimal CDSS evaluate request
type SimpleCDSSRequest struct {
	PatientID   string                   `json:"patient_id"`
	EncounterID string                   `json:"encounter_id,omitempty"`
	Conditions  []map[string]interface{} `json:"conditions,omitempty"`
	Observations []map[string]interface{} `json:"observations,omitempty"`
	Medications []map[string]interface{} `json:"medications,omitempty"`
	Procedures  []map[string]interface{} `json:"procedures,omitempty"`
	Allergies   []map[string]interface{} `json:"allergies,omitempty"`
	Options     map[string]interface{}   `json:"options,omitempty"`
}

// SepsisCDSSRequest - Pre-built request for sepsis patient test
var SepsisCDSSRequest = SimpleCDSSRequest{
	PatientID:   "sepsis-patient-001",
	EncounterID: "encounter-456",
	Conditions: []map[string]interface{}{
		{
			"resourceType": "Condition",
			"code": map[string]interface{}{
				"coding": []map[string]interface{}{
					{"system": "http://snomed.info/sct", "code": "91302008", "display": "Sepsis"},
				},
			},
			"clinicalStatus": map[string]interface{}{
				"coding": []map[string]interface{}{{"code": "active"}},
			},
		},
	},
	Observations: []map[string]interface{}{
		{
			"resourceType": "Observation",
			"code": map[string]interface{}{
				"coding": []map[string]interface{}{
					{"system": "http://loinc.org", "code": "2524-7", "display": "Lactate"},
				},
			},
			"valueQuantity": map[string]interface{}{
				"value": 4.5,
				"unit":  "mmol/L",
			},
			"status": "final",
		},
	},
	Options: map[string]interface{}{
		"enable_subsumption": true,
		"generate_alerts":    true,
		"evaluate_rules":     true,
	},
}

// ============================================================================
// THREE-CHECK Pipeline Test Cases
// ============================================================================

// ThreeCheckTestCase represents a test case for the THREE-CHECK pipeline
type ThreeCheckTestCase struct {
	Description     string
	Code            string
	System          string
	ValueSet        string
	ExpectedValid   bool
	ExpectedStep    int    // 2 = Exact Match, 3 = Subsumption
	ExpectedType    string // "exact", "subsumption", "none"
	ExpectedAncestor string // For subsumption matches
}

// ThreeCheckTestCases - Test cases for THREE-CHECK pipeline validation
var ThreeCheckTestCases = []ThreeCheckTestCase{
	{
		Description:   "Test 3.1: Exact Match - Sepsis code directly in value set",
		Code:          "91302008",
		System:        "http://snomed.info/sct",
		ValueSet:      "AUSepsisConditions",
		ExpectedValid: true,
		ExpectedStep:  2,
		ExpectedType:  "exact",
	},
	{
		Description:   "Test 3.2: Exact Match - Heart failure code",
		Code:          "84114007",
		System:        "http://snomed.info/sct",
		ValueSet:      "HeartFailure",
		ExpectedValid: true,
		ExpectedStep:  2,
		ExpectedType:  "exact",
	},
	{
		Description:   "Test 3.3: Subsumption Match - Streptococcal sepsis (child of Sepsis)",
		Code:          "448417001",
		System:        "http://snomed.info/sct",
		ValueSet:      "SepsisDiagnosis",
		ExpectedValid: true,
		ExpectedStep:  3,
		ExpectedType:  "subsumption",
		ExpectedAncestor: "91302008",
	},
	{
		Description:   "Test 3.4: No Match - Diabetes not in Sepsis value set",
		Code:          "73211009",
		System:        "http://snomed.info/sct",
		ValueSet:      "AUSepsisConditions",
		ExpectedValid: false,
		ExpectedStep:  3,
		ExpectedType:  "none",
	},
	{
		Description:   "Test 3.5: Cross-System Match - ICD-10 Sepsis",
		Code:          "A41.9",
		System:        "http://hl7.org/fhir/sid/icd-10-au",
		ValueSet:      "SepsisDiagnosis",
		ExpectedValid: true,
		ExpectedStep:  2,
		ExpectedType:  "exact",
	},
	{
		Description:   "Test 3.6: Exact Match - ACE Inhibitor medication",
		Code:          "386873009",
		System:        "http://snomed.info/sct",
		ValueSet:      "ACEInhibitors",
		ExpectedValid: true,
		ExpectedStep:  2,
		ExpectedType:  "exact",
	},
	{
		Description:   "Test 3.7: Exact Match - Lactate LOINC code",
		Code:          "2524-7",
		System:        "http://loinc.org",
		ValueSet:      "LabLactate",
		ExpectedValid: true,
		ExpectedStep:  2,
		ExpectedType:  "exact",
	},
}

// ============================================================================
// Rule Engine Test Cases
// ============================================================================

// RuleEngineTestCase represents a test case for rule engine evaluation
type RuleEngineTestCase struct {
	Description      string
	RuleID           string
	InputConditions  []map[string]interface{}
	InputLabs        []map[string]interface{}
	InputMedications []map[string]interface{}
	ExpectedFired    bool
	ExpectedSeverity string
}

// RuleEngineTestCases - Test cases for rule engine validation
var RuleEngineTestCases = []RuleEngineTestCase{
	{
		Description: "Test 5.1: Simple VALUE_SET Condition - Sepsis Detected",
		RuleID:      "sepsis-detected",
		InputConditions: []map[string]interface{}{
			{"code": "91302008", "system": "http://snomed.info/sct", "display": "Sepsis"},
		},
		ExpectedFired:    true,
		ExpectedSeverity: "critical",
	},
	{
		Description: "Test 5.2: THRESHOLD Condition - Elevated Lactate",
		RuleID:      "elevated-lactate",
		InputLabs: []map[string]interface{}{
			{"code": "2524-7", "system": "http://loinc.org", "value": 4.2, "unit": "mmol/L"},
		},
		ExpectedFired:    true,
		ExpectedSeverity: "warning",
	},
	{
		Description: "Test 5.3: COMPOUND Condition (AND) - Sepsis with Elevated Lactate",
		RuleID:      "sepsis-lactate-elevated",
		InputConditions: []map[string]interface{}{
			{"code": "91302008", "system": "http://snomed.info/sct", "display": "Sepsis"},
		},
		InputLabs: []map[string]interface{}{
			{"code": "2524-7", "system": "http://loinc.org", "value": 4.2, "unit": "mmol/L"},
		},
		ExpectedFired:    true,
		ExpectedSeverity: "critical",
	},
	{
		Description: "Test 5.4: COMPOUND Condition (AND) - Partial Match (should NOT fire)",
		RuleID:      "sepsis-lactate-elevated",
		InputConditions: []map[string]interface{}{
			{"code": "91302008", "system": "http://snomed.info/sct", "display": "Sepsis"},
		},
		InputLabs: []map[string]interface{}{
			{"code": "2524-7", "system": "http://loinc.org", "value": 1.5, "unit": "mmol/L"},
		},
		ExpectedFired: false,
	},
	{
		Description: "Test 5.5: Nephrotoxic Drug Detection",
		RuleID:      "nephrotoxic-drug",
		InputMedications: []map[string]interface{}{
			{"code": "21411011000036105", "system": "http://snomed.info/sct", "display": "Gentamicin"},
		},
		ExpectedFired:    true,
		ExpectedSeverity: "warning",
	},
}

// ============================================================================
// Alert Generation Test Cases
// ============================================================================

// AlertTestCase represents a test case for alert generation
type AlertTestCase struct {
	Description          string
	PatientID            string
	Conditions           []map[string]interface{}
	Labs                 []map[string]interface{}
	Medications          []map[string]interface{}
	ExpectedAlertCount   int
	ExpectedSeverities   []string
	ExpectedDomains      []string
	ExpectedContainsText []string
}

// AlertTestCases - Test cases for alert generation validation
var AlertTestCases = []AlertTestCase{
	{
		Description: "Test 6.1: Alert from Fired Rule - Sepsis",
		PatientID:   "test-patient-001",
		Conditions: []map[string]interface{}{
			{
				"resourceType": "Condition",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://snomed.info/sct", "code": "91302008", "display": "Sepsis"},
					},
				},
				"clinicalStatus": map[string]interface{}{
					"coding": []map[string]interface{}{{"code": "active"}},
				},
			},
		},
		Labs: []map[string]interface{}{
			{
				"resourceType": "Observation",
				"code": map[string]interface{}{
					"coding": []map[string]interface{}{
						{"system": "http://loinc.org", "code": "2524-7", "display": "Lactate"},
					},
				},
				"valueQuantity": map[string]interface{}{"value": 4.5, "unit": "mmol/L"},
				"status":        "final",
			},
		},
		ExpectedAlertCount:   1,
		ExpectedSeverities:   []string{"critical"},
		ExpectedDomains:      []string{"sepsis"},
		ExpectedContainsText: []string{"Sepsis", "Lactate", "4.5"},
	},
}

// ============================================================================
// Performance Test Configurations
// ============================================================================

// PerformanceTestConfig holds configuration for performance tests
type PerformanceTestConfig struct {
	SingleRequestP50Target time.Duration
	SingleRequestP95Target time.Duration
	SingleRequestP99Target time.Duration
	ConcurrentRequests     int
	MinThroughput          float64 // requests per second
	MaxErrorRate           float64 // percentage
	MaxLatencyDegradation  float64 // multiplier
}

// DefaultPerformanceConfig - Default performance test configuration
var DefaultPerformanceConfig = PerformanceTestConfig{
	SingleRequestP50Target: 50 * time.Millisecond,
	SingleRequestP95Target: 100 * time.Millisecond,
	SingleRequestP99Target: 200 * time.Millisecond,
	ConcurrentRequests:     50,
	MinThroughput:          100,
	MaxErrorRate:           1.0,
	MaxLatencyDegradation:  2.0,
}

// ============================================================================
// Expected CDSS Response Structures
// ============================================================================

// ExpectedSepsisResponse - Expected response for sepsis patient evaluation
var ExpectedSepsisResponse = map[string]interface{}{
	"success":              true,
	"min_facts_extracted":  2,
	"min_alerts_generated": 1,
	"pipeline_used":        "THREE-CHECK",
	"expected_domains":     []string{"sepsis"},
	"critical_alert_expected": true,
}

// ExpectedDiabeticResponse - Expected response for diabetic patient with AKI risk
var ExpectedDiabeticResponse = map[string]interface{}{
	"success":              true,
	"min_facts_extracted":  4,
	"min_alerts_generated": 2,
	"pipeline_used":        "THREE-CHECK",
	"expected_domains":     []string{"renal", "nephrotoxic"},
}

// ============================================================================
// Helper Functions
// ============================================================================

// GetCDSSTestTimeout returns the timeout for CDSS integration tests
// E2E tests with complex patient bundles may take longer due to:
// - FHIR bundle parsing with multiple resources
// - Multiple THREE-CHECK pipeline evaluations
// - Neo4j subsumption queries
func GetCDSSTestTimeout() time.Duration {
	return 2 * time.Minute
}

// GetPerformanceTestTimeout returns the timeout for performance tests
func GetPerformanceTestTimeout() time.Duration {
	return 5 * time.Minute
}
