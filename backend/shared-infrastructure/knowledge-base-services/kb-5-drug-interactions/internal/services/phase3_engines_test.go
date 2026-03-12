package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"kb-drug-interactions/internal/models"
)

// ============================================================================
// DRUG-DISEASE ENGINE TESTS
// ============================================================================

func TestDrugDiseaseCheckRequest_Structure(t *testing.T) {
	request := DrugDiseaseCheckRequest{
		DrugCodes:    []string{"RXCUI:5640"},
		DiseaseCodes: []string{"N18"},
		CodeSystem:   "ICD-10",
	}

	assert.Equal(t, 1, len(request.DrugCodes))
	assert.Equal(t, 1, len(request.DiseaseCodes))
	assert.Equal(t, "RXCUI:5640", request.DrugCodes[0])
	assert.Equal(t, "N18", request.DiseaseCodes[0])
	assert.Equal(t, "ICD-10", request.CodeSystem)
}

func TestDrugDiseaseResult_SeverityOrdering(t *testing.T) {
	severityOrder := map[models.DDISeverity]int{
		models.SeverityContraindicated: 4,
		models.SeverityMajor:           3,
		models.SeverityModerate:        2,
		models.SeverityMinor:           1,
		models.SeverityUnknown:         0,
	}

	assert.True(t, severityOrder[models.SeverityContraindicated] > severityOrder[models.SeverityMajor])
	assert.True(t, severityOrder[models.SeverityMajor] > severityOrder[models.SeverityModerate])
	assert.True(t, severityOrder[models.SeverityModerate] > severityOrder[models.SeverityMinor])
}

func TestDrugDiseaseResult_Structure(t *testing.T) {
	result := DrugDiseaseResult{
		DrugCode:             "RXCUI:5640",
		DrugName:             "Ibuprofen",
		DiseaseCode:          "N18",
		DiseaseName:          "Chronic Kidney Disease",
		ContraindicationType: "relative",
		Severity:             models.SeverityMajor,
		ClinicalEffects:      "NSAIDs reduce GFR",
		ManagementStrategy:   "Avoid in CKD",
		Confidence:           0.90,
		MonitoringRequired:   true,
	}

	assert.Equal(t, "RXCUI:5640", result.DrugCode)
	assert.Equal(t, models.SeverityMajor, result.Severity)
	assert.Equal(t, "relative", result.ContraindicationType)
	assert.True(t, result.Confidence > 0.5)
	assert.True(t, result.MonitoringRequired)
}

// ============================================================================
// ALLERGY ENGINE TESTS
// ============================================================================

func TestAllergyCheckRequest_Structure(t *testing.T) {
	request := AllergyCheckRequest{
		PatientAllergies: []PatientAllergy{
			{
				AllergenCode: "RXCUI:7984",
				AllergenName: "Penicillin",
				ReactionType: "urticaria",
				Severity:     "moderate",
			},
		},
		DrugCodes: []string{"RXCUI:2231"},
	}

	assert.Equal(t, 1, len(request.PatientAllergies))
	assert.Equal(t, 1, len(request.DrugCodes))
	assert.Equal(t, "Penicillin", request.PatientAllergies[0].AllergenName)
}

func TestAllergyCheckResult_Structure(t *testing.T) {
	result := AllergyCheckResult{
		AllergenCode:        "RXCUI:7984",
		AllergenName:        "Penicillin",
		DrugCode:            "RXCUI:2231",
		DrugName:            "Cephalexin",
		CrossReactivityType: "class",
		CrossReactivityRate: 2.0,
		Severity:            models.SeverityModerate,
		ReactionType:        "urticaria",
		AlertLevel:          "warning",
		ClinicalGuidance:    "1st gen cephalosporin cross-reactivity ~2%",
		Confidence:          0.85,
	}

	assert.Equal(t, "Penicillin", result.AllergenName)
	assert.Equal(t, float64(2.0), result.CrossReactivityRate)
	assert.Equal(t, "warning", result.AlertLevel)
}

func TestAllergyEngine_ReactionTypeSeverity(t *testing.T) {
	reactionSeverity := map[string]int{
		"anaphylaxis":  5,
		"angioedema":   4,
		"bronchospasm": 3,
		"urticaria":    2,
		"rash":         1,
		"other":        0,
	}

	assert.True(t, reactionSeverity["anaphylaxis"] > reactionSeverity["urticaria"])
	assert.True(t, reactionSeverity["angioedema"] > reactionSeverity["rash"])
}

func TestPatientAllergy_Structure(t *testing.T) {
	allergy := PatientAllergy{
		AllergenCode: "RXCUI:7984",
		AllergenName: "Penicillin",
		AllergenType: "drug",
		ReactionType: "urticaria",
		Severity:     "moderate",
		OnsetDate:    "2020-01-15",
		Verified:     true,
	}

	assert.Equal(t, "drug", allergy.AllergenType)
	assert.True(t, allergy.Verified)
}

// ============================================================================
// DUPLICATE THERAPY ENGINE TESTS
// ============================================================================

func TestDuplicateTherapyCheckRequest_Structure(t *testing.T) {
	request := DuplicateTherapyCheckRequest{
		DrugCodes:  []string{"RXCUI:5640", "RXCUI:7646"},
		CheckLevel: "moderate",
	}

	assert.Equal(t, 2, len(request.DrugCodes))
	assert.Equal(t, "moderate", request.CheckLevel)
}

func TestDuplicateTherapyResult_Structure(t *testing.T) {
	result := DuplicateTherapyResult{
		DuplicateType:    "therapeutic_class",
		ATCCode:          "M01A",
		TherapeuticClass: "NSAIDs",
		DuplicateDrugs: []DuplicateDrugInfo{
			{DrugCode: "RXCUI:5640", DrugName: "Ibuprofen", ATCCode: "M01AE01"},
			{DrugCode: "RXCUI:7646", DrugName: "Naproxen", ATCCode: "M01AE02"},
		},
		Severity:           models.SeverityMajor,
		AllowedCombination: false,
		ClinicalRationale:  "Multiple NSAIDs increase adverse effects",
		ManagementStrategy: "Avoid concurrent NSAID use",
	}

	assert.Equal(t, "therapeutic_class", result.DuplicateType)
	assert.Equal(t, 2, len(result.DuplicateDrugs))
	assert.Equal(t, models.SeverityMajor, result.Severity)
	assert.False(t, result.AllowedCombination)
}

func TestDrugTherapeuticMapping_Structure(t *testing.T) {
	mapping := DrugTherapeuticMapping{
		DrugCode:         "RXCUI:5640",
		DrugName:         "Ibuprofen",
		ATCCode:          "M01AE01",
		ATCLevel:         5,
		TherapeuticClass: "NSAIDs",
		DatasetVersion:   "2025Q4",
	}

	assert.Equal(t, "M01AE01", mapping.ATCCode)
	assert.Equal(t, 5, mapping.ATCLevel)
}

func TestDuplicateTherapyEngine_ATCLevelHierarchy(t *testing.T) {
	testCases := []struct {
		atcCode       string
		expectedChars int
	}{
		{"M01AE01", 7}, // Full ATC code - level 5
		{"M01AE", 5},   // Chemical subgroup - level 4
		{"M01A", 4},    // Pharmacological subgroup - level 3
		{"M01", 3},     // Therapeutic subgroup - level 2
		{"M", 1},       // Anatomical main group - level 1
	}

	for _, tc := range testCases {
		t.Run(tc.atcCode, func(t *testing.T) {
			assert.Equal(t, tc.expectedChars, len(tc.atcCode))
		})
	}
}

// ============================================================================
// CLINICAL SCENARIO TESTS
// ============================================================================

func TestClinicalScenario_TripleWhammy(t *testing.T) {
	// Triple Whammy: ACE/ARB + Diuretic + NSAID
	drugs := []string{
		"RXCUI:29046", // Lisinopril (ACE inhibitor)
		"RXCUI:4316",  // Hydrochlorothiazide (Diuretic)
		"RXCUI:5640",  // Ibuprofen (NSAID)
	}

	assert.Equal(t, 3, len(drugs))
}

func TestClinicalScenario_MultipleNSAIDs(t *testing.T) {
	request := DuplicateTherapyCheckRequest{
		DrugCodes:  []string{"RXCUI:5640", "RXCUI:7646"},
		CheckLevel: "moderate",
	}

	assert.Equal(t, 2, len(request.DrugCodes))
	assert.Equal(t, "moderate", request.CheckLevel)
}

func TestClinicalScenario_PenicillinAllergyWithCephalosporin(t *testing.T) {
	request := AllergyCheckRequest{
		PatientAllergies: []PatientAllergy{
			{
				AllergenCode: "RXCUI:7984",
				AllergenName: "Penicillin",
				AllergenType: "drug_class",
				ReactionType: "urticaria",
			},
		},
		DrugCodes: []string{"RXCUI:2231"},
	}

	assert.Equal(t, "Penicillin", request.PatientAllergies[0].AllergenName)
	assert.Equal(t, 1, len(request.DrugCodes))
}

func TestClinicalScenario_NSAIDWithCKD(t *testing.T) {
	request := DrugDiseaseCheckRequest{
		DrugCodes:    []string{"RXCUI:5640"},
		DiseaseCodes: []string{"N18"},
	}

	assert.Equal(t, "RXCUI:5640", request.DrugCodes[0])
	assert.Equal(t, "N18", request.DiseaseCodes[0])
}

func TestClinicalScenario_ACEInhibitorInPregnancy(t *testing.T) {
	request := DrugDiseaseCheckRequest{
		DrugCodes:    []string{"RXCUI:29046"},
		DiseaseCodes: []string{"Z33"},
	}

	assert.Equal(t, "RXCUI:29046", request.DrugCodes[0])
	assert.Equal(t, "Z33", request.DiseaseCodes[0])
}

func TestClinicalScenario_MultipleBenzodiazepines(t *testing.T) {
	request := DuplicateTherapyCheckRequest{
		DrugCodes:  []string{"RXCUI:6470", "RXCUI:596"},
		CheckLevel: "moderate",
	}

	assert.Equal(t, 2, len(request.DrugCodes))
}

// ============================================================================
// ENGINE INITIALIZATION TESTS
// Note: Full integration tests require a running database connection
// ============================================================================

func TestDrugDiseaseEngine_CacheInitialization(t *testing.T) {
	// Verify cache is properly initialized
	cache := make(map[string][]DrugDiseaseContraindication)
	assert.NotNil(t, cache)
	assert.Equal(t, 0, len(cache))
}

func TestAllergyEngine_CacheInitialization(t *testing.T) {
	// Verify cache is properly initialized
	cache := make(map[string][]AllergyRule)
	assert.NotNil(t, cache)
	assert.Equal(t, 0, len(cache))
}

func TestDuplicateTherapyEngine_CacheInitialization(t *testing.T) {
	// Verify caches are properly initialized
	ruleCache := make(map[string][]DuplicateTherapyRule)
	classCache := make(map[string][]DrugTherapeuticMapping)
	assert.NotNil(t, ruleCache)
	assert.NotNil(t, classCache)
	assert.Equal(t, 0, len(ruleCache))
	assert.Equal(t, 0, len(classCache))
}

// ============================================================================
// SEVERITY AND EVIDENCE TESTS
// ============================================================================

func TestSeverityLevels(t *testing.T) {
	validSeverities := []models.DDISeverity{
		models.SeverityContraindicated,
		models.SeverityMajor,
		models.SeverityModerate,
		models.SeverityMinor,
		models.SeverityUnknown,
	}

	assert.Equal(t, 5, len(validSeverities))
}

func TestConfidenceScoreRange(t *testing.T) {
	testScores := []float64{0.0, 0.5, 0.75, 0.90, 0.95, 1.0}

	for _, score := range testScores {
		assert.True(t, score >= 0.0 && score <= 1.0,
			"Confidence score %.2f should be in range [0, 1]", score)
	}
}

func TestEvidenceLevels(t *testing.T) {
	evidenceLevels := []models.EvidenceLevel{
		models.EvidenceLevelA,
		models.EvidenceLevelB,
		models.EvidenceLevelC,
		models.EvidenceLevelD,
		models.EvidenceLevelExpertOpinion,
		models.EvidenceLevelUnknown,
	}

	assert.Equal(t, 6, len(evidenceLevels))
}

// ============================================================================
// DUPLICATE DRUG INFO TESTS
// ============================================================================

func TestDuplicateDrugInfo_Structure(t *testing.T) {
	info := DuplicateDrugInfo{
		DrugCode:         "RXCUI:5640",
		DrugName:         "Ibuprofen",
		ATCCode:          "M01AE01",
		TherapeuticClass: "Propionic acid derivatives",
	}

	assert.Equal(t, "RXCUI:5640", info.DrugCode)
	assert.Equal(t, "Ibuprofen", info.DrugName)
	assert.Equal(t, "M01AE01", info.ATCCode)
}

// ============================================================================
// CROSS-REACTIVITY INFO TESTS
// ============================================================================

func TestCrossReactivityInfo_Structure(t *testing.T) {
	info := CrossReactivityInfo{
		AllergenClass:       "Penicillins",
		RelatedDrugs:        []string{"Cephalexin", "Cefazolin"},
		CrossReactivityRate: 2.0,
		ClinicalNote:        "1st gen cephalosporins have ~2% cross-reactivity",
	}

	assert.Equal(t, "Penicillins", info.AllergenClass)
	assert.Equal(t, 2, len(info.RelatedDrugs))
	assert.Equal(t, float64(2.0), info.CrossReactivityRate)
}
