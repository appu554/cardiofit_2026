package fixtures

import (
	"time"
)

// AllValueSetNames contains all 50 value set names for validation testing
var AllValueSetNames = []string{
	// FHIR R4 Administrative (18)
	"AdministrativeGender",
	"AddressType",
	"AddressUse",
	"ContactPointSystem",
	"ContactPointUse",
	"IdentifierUse",
	"NameUse",
	"PublicationStatus",
	"NarrativeStatus",
	"QuantityComparator",
	"ResourceTypes",
	"Languages",
	"MaritalStatus",
	"ContactRelationship",
	"AllergyIntoleranceCategory",
	"AllergyIntoleranceCriticality",
	"AllergyIntoleranceSeverity",
	"AllergyIntoleranceType",

	// AU-Specific Clinical (6)
	"AUAKIConditions",
	"AURenalLabs",
	"AURenalMedications",
	"AUSepsisAntibiotics",
	"AUSepsisConditions",
	"AUSepsisLabs",

	// KB7 Clinical Conditions (8)
	"InfectionSource",
	"Hypertension",
	"AtrialFibrillation",
	"IschemicStroke",
	"ActiveBleeding",
	"RespiratoryFailure",
	"DiabetesMellitus",
	"HeartFailure",

	// KB7 Medications (6)
	"BroadSpectrumAntibiotics",
	"Anticoagulants",
	"ACEInhibitors",
	"NSAIDs",
	"BetaBlockers",
	"Statins",

	// KB7 Labs (7)
	"LabTroponin",
	"LabINR",
	"LabEGFR",
	"LabBloodCulture",
	"LabBNP",
	"LabCreatinine",
	"LabLactate",

	// KB7 Procedures (2)
	"ProcDialysis",
	"ProcCTContrast",

	// KB7 Extended (2)
	"RenalConditions",
	"CardiacConditions",

	// CHA₂DS₂-VASc Scoring (1)
	"VascularDisease",
}

// ValueSetCategoryMap categorizes each value set by its clinical domain
var ValueSetCategoryMap = map[string]string{
	// FHIR Administrative
	"AdministrativeGender":           "administrative",
	"AddressType":                    "administrative",
	"AddressUse":                     "administrative",
	"ContactPointSystem":             "administrative",
	"ContactPointUse":                "administrative",
	"IdentifierUse":                  "administrative",
	"NameUse":                        "administrative",
	"PublicationStatus":              "administrative",
	"NarrativeStatus":                "administrative",
	"QuantityComparator":             "administrative",
	"ResourceTypes":                  "administrative",
	"Languages":                      "administrative",
	"MaritalStatus":                  "administrative",
	"ContactRelationship":            "administrative",
	"AllergyIntoleranceCategory":     "administrative",
	"AllergyIntoleranceCriticality":  "administrative",
	"AllergyIntoleranceSeverity":     "administrative",
	"AllergyIntoleranceType":         "administrative",

	// AU Clinical
	"AUAKIConditions":      "au-clinical",
	"AURenalLabs":          "au-clinical",
	"AURenalMedications":   "au-clinical",
	"AUSepsisAntibiotics":  "au-clinical",
	"AUSepsisConditions":   "au-clinical",
	"AUSepsisLabs":         "au-clinical",

	// Conditions
	"InfectionSource":      "conditions",
	"Hypertension":         "conditions",
	"AtrialFibrillation":   "conditions",
	"IschemicStroke":       "conditions",
	"ActiveBleeding":       "conditions",
	"RespiratoryFailure":   "conditions",
	"DiabetesMellitus":     "conditions",
	"HeartFailure":         "conditions",
	"RenalConditions":      "conditions",
	"CardiacConditions":    "conditions",

	// Medications
	"BroadSpectrumAntibiotics": "medications",
	"Anticoagulants":           "medications",
	"ACEInhibitors":            "medications",
	"NSAIDs":                   "medications",
	"BetaBlockers":             "medications",
	"Statins":                  "medications",

	// Labs
	"LabTroponin":     "labs",
	"LabINR":          "labs",
	"LabEGFR":         "labs",
	"LabBloodCulture": "labs",
	"LabBNP":          "labs",
	"LabCreatinine":   "labs",
	"LabLactate":      "labs",

	// Procedures
	"ProcDialysis":   "procedures",
	"ProcCTContrast": "procedures",

	// CHA₂DS₂-VASc
	"VascularDisease": "conditions",
}

// AUSpecificValueSets contains AU-specific value set names
var AUSpecificValueSets = []string{
	"AUAKIConditions",
	"AURenalLabs",
	"AURenalMedications",
	"AUSepsisAntibiotics",
	"AUSepsisConditions",
	"AUSepsisLabs",
}

// SNOMEDSubsumptionTestCases provides test cases for SNOMED subsumption testing
var SNOMEDSubsumptionTestCases = []struct {
	Name           string
	SubCode        string // Child/descendant concept
	SuperCode      string // Parent/ancestor concept
	ExpectedResult bool   // true = subsumes, false = does not subsume
	Description    string
}{
	// Clinical Finding hierarchy tests
	{
		Name:           "ClinicalFinding_SubsumesDisease",
		SubCode:        "64572001",  // Disease (disorder)
		SuperCode:      "404684003", // Clinical finding
		ExpectedResult: true,
		Description:    "Disease is-a Clinical finding",
	},
	{
		Name:           "SepsisSubsumption",
		SubCode:        "91302008",  // Sepsis (disorder)
		SuperCode:      "64572001",  // Disease (disorder)
		ExpectedResult: true,
		Description:    "Sepsis is-a Disease",
	},
	{
		Name:           "AKISubsumption",
		SubCode:        "14669001",  // Acute kidney injury (disorder)
		SuperCode:      "90708001",  // Kidney disease (disorder)
		ExpectedResult: true,
		Description:    "AKI is-a Kidney disease",
	},

	// Negative test cases
	{
		Name:           "NotSubsumed_DiseaseNotProcedure",
		SubCode:        "64572001",  // Disease (disorder)
		SuperCode:      "71388002",  // Procedure
		ExpectedResult: false,
		Description:    "Disease is NOT a Procedure",
	},
	{
		Name:           "NotSubsumed_FindingNotMedication",
		SubCode:        "404684003", // Clinical finding
		SuperCode:      "410942007", // Drug or medicament
		ExpectedResult: false,
		Description:    "Clinical finding is NOT a Drug",
	},

	// Pharmaceutical hierarchy tests
	{
		Name:           "Paracetamol_IsAnalgesic",
		SubCode:        "387517004", // Paracetamol (substance)
		SuperCode:      "373265006", // Analgesic (substance)
		ExpectedResult: true,
		Description:    "Paracetamol is-a Analgesic",
	},
	{
		Name:           "VancomycinIsAntibiotic",
		SubCode:        "372735009", // Vancomycin (substance)
		SuperCode:      "373297006", // Anti-infective agent (substance)
		ExpectedResult: true,
		Description:    "Vancomycin is-a Anti-infective",
	},

	// Laboratory finding tests
	{
		Name:           "HighCreatinine_IsLabFinding",
		SubCode:        "166717003", // Serum creatinine raised
		SuperCode:      "365636006", // Finding of creatinine level
		ExpectedResult: true,
		Description:    "High creatinine is-a creatinine finding",
	},
}

// ValueSetMembershipTestCases provides test cases for value set membership testing
var ValueSetMembershipTestCases = []struct {
	ValueSetName   string
	Code           string
	System         string
	ExpectedMember bool
	Description    string
}{
	// AU Sepsis Conditions
	{
		ValueSetName:   "AUSepsisConditions",
		Code:           "91302008",
		System:         "http://snomed.info/sct",
		ExpectedMember: true,
		Description:    "Sepsis should be in AUSepsisConditions",
	},
	{
		ValueSetName:   "AUSepsisConditions",
		Code:           "76571007",
		System:         "http://snomed.info/sct",
		ExpectedMember: true,
		Description:    "Septic shock should be in AUSepsisConditions",
	},

	// AU AKI Conditions
	{
		ValueSetName:   "AUAKIConditions",
		Code:           "14669001",
		System:         "http://snomed.info/sct",
		ExpectedMember: true,
		Description:    "Acute kidney injury should be in AUAKIConditions",
	},

	// AU Renal Labs
	{
		ValueSetName:   "AURenalLabs",
		Code:           "113075003",
		System:         "http://snomed.info/sct",
		ExpectedMember: true,
		Description:    "Creatinine measurement should be in AURenalLabs",
	},

	// Administrative Gender
	{
		ValueSetName:   "AdministrativeGender",
		Code:           "male",
		System:         "http://hl7.org/fhir/administrative-gender",
		ExpectedMember: true,
		Description:    "male should be in AdministrativeGender",
	},
	{
		ValueSetName:   "AdministrativeGender",
		Code:           "female",
		System:         "http://hl7.org/fhir/administrative-gender",
		ExpectedMember: true,
		Description:    "female should be in AdministrativeGender",
	},

	// Negative test cases
	{
		ValueSetName:   "AUSepsisConditions",
		Code:           "73211009",
		System:         "http://snomed.info/sct",
		ExpectedMember: false,
		Description:    "Diabetes should NOT be in AUSepsisConditions",
	},
	{
		ValueSetName:   "AURenalLabs",
		Code:           "387517004",
		System:         "http://snomed.info/sct",
		ExpectedMember: false,
		Description:    "Paracetamol should NOT be in AURenalLabs",
	},
}

// PerformanceBenchmarkConfig provides configuration for performance testing
type PerformanceBenchmarkConfig struct {
	ConcurrentUsers   int
	RequestsPerSecond int
	DurationSeconds   int
	WarmupSeconds     int
}

// DefaultPerformanceBenchmarkConfig returns default benchmark configuration
func DefaultPerformanceBenchmarkConfig() PerformanceBenchmarkConfig {
	return PerformanceBenchmarkConfig{
		ConcurrentUsers:   10,
		RequestsPerSecond: 100,
		DurationSeconds:   30,
		WarmupSeconds:     5,
	}
}

// LoadTestConfig provides configuration for k6 load tests
type LoadTestConfig struct {
	BaseURL     string
	VUs         int
	Duration    string
	Thresholds  LoadTestThresholds
}

// LoadTestThresholds defines performance thresholds
type LoadTestThresholds struct {
	HTTPReqDuration95thPercentile float64 // p95 latency in ms
	HTTPReqFailed                 float64 // failure rate percentage
	IterationDuration             float64 // max iteration duration in ms
}

// DefaultLoadTestConfig returns default load test configuration
func DefaultLoadTestConfig() LoadTestConfig {
	return LoadTestConfig{
		BaseURL:  "http://localhost:8087",
		VUs:      10,
		Duration: "30s",
		Thresholds: LoadTestThresholds{
			HTTPReqDuration95thPercentile: 500.0,  // 500ms p95
			HTTPReqFailed:                 1.0,    // <1% failure rate
			IterationDuration:             2000.0, // 2s max iteration
		},
	}
}

// StressTestConfig provides configuration for k6 stress tests
type StressTestConfig struct {
	BaseURL    string
	Stages     []StressTestStage
	Thresholds LoadTestThresholds
}

// StressTestStage defines a stage in stress testing
type StressTestStage struct {
	Duration string
	Target   int
}

// DefaultStressTestConfig returns default stress test configuration
func DefaultStressTestConfig() StressTestConfig {
	return StressTestConfig{
		BaseURL: "http://localhost:8087",
		Stages: []StressTestStage{
			{Duration: "30s", Target: 10},   // Ramp up
			{Duration: "1m", Target: 50},    // Peak load
			{Duration: "30s", Target: 100},  // Stress
			{Duration: "30s", Target: 0},    // Ramp down
		},
		Thresholds: LoadTestThresholds{
			HTTPReqDuration95thPercentile: 1000.0, // 1s p95 under stress
			HTTPReqFailed:                 5.0,    // <5% failure rate
			IterationDuration:             5000.0, // 5s max iteration
		},
	}
}

// TestServiceEndpoints defines all API endpoints for testing
var TestServiceEndpoints = map[string]string{
	"health":              "/health",
	"valuesets_list":      "/v1/rules/valuesets",
	"valueset_get":        "/v1/rules/valuesets/{id}",
	"valueset_contains":   "/v1/rules/valuesets/{id}/contains",
	"subsumption_check":   "/v1/subsumption/check",
	"subsumption_config":  "/v1/subsumption/config",
	"terminology_lookup":  "/v1/terminology/lookup",
	"seed":                "/v1/rules/seed",
	"builtin_count":       "/v1/valuesets/builtin/count",
}

// GetAUTestTimeout returns the timeout for AU-specific tests
func GetAUTestTimeout() time.Duration {
	return 60 * time.Second
}

// GetNeo4JAUURL returns the Neo4j AU connection URL for testing
func GetNeo4JAUURL() string {
	return "bolt://localhost:7688"
}

// GetNeo4JAUCredentials returns Neo4j AU test credentials
func GetNeo4JAUCredentials() (username, password string) {
	return "neo4j", "kb7aupassword"
}

// GetGraphDBURL returns the GraphDB URL for testing
func GetGraphDBURL() string {
	return "http://localhost:7200"
}

// GetGraphDBRepository returns the GraphDB repository name for testing
func GetGraphDBRepository() string {
	return "kb7-terminology"
}

// GetKafkaBrokers returns Kafka broker addresses for testing
func GetKafkaBrokers() string {
	return "localhost:9093"
}

// GetKafkaTopic returns the CDC Kafka topic name
func GetKafkaTopic() string {
	return "kb7.graphdb.changes"
}

// ExpectedValueSetCounts provides expected counts for verification
var ExpectedValueSetCounts = struct {
	Total          int
	FHIRAdmin      int
	AUSpecific     int
	Conditions     int
	Medications    int
	Labs           int
	Procedures     int
}{
	Total:          50,
	FHIRAdmin:      18,
	AUSpecific:     6,
	Conditions:     11, // Added VascularDisease for CHA₂DS₂-VASc
	Medications:    6,
	Labs:           7,
	Procedures:     2,
}
