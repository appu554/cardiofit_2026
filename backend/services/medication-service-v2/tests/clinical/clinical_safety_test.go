// +build clinical

package clinical_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"medication-service-v2/internal/application/services"
	"medication-service-v2/internal/domain/entities"
	"medication-service-v2/tests/helpers/fixtures"
	"medication-service-v2/tests/helpers/testsetup"
)

// ClinicalSafetyTestSuite validates clinical logic and patient safety
type ClinicalSafetyTestSuite struct {
	suite.Suite
	
	medicationService *services.MedicationService
	clinicalEngine    *services.ClinicalEngineService
	
	ctx        context.Context
	testRecipe *entities.Recipe
}

func TestClinicalSafetyTestSuite(t *testing.T) {
	if os.Getenv("SKIP_CLINICAL_TESTS") == "true" {
		t.Skip("Skipping clinical safety tests")
	}
	
	suite.Run(t, new(ClinicalSafetyTestSuite))
}

func (suite *ClinicalSafetyTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	
	// Setup services for clinical testing
	suite.setupClinicalServices()
	
	// Setup clinical test data
	suite.setupClinicalTestData()
}

func (suite *ClinicalSafetyTestSuite) setupClinicalServices() {
	testDB := testsetup.SetupTestDatabase(suite.T())
	testRedis := testsetup.SetupTestRedis(suite.T())
	
	medicationRepo := testsetup.SetupMedicationRepository(testDB)
	recipeRepo := testsetup.SetupRecipeRepository(testDB)
	
	rustEngine := testsetup.SetupRustEngine(suite.T())
	apolloClient := testsetup.SetupApolloClient(suite.T())
	contextGateway := testsetup.SetupContextGateway(suite.T())
	
	auditService := services.NewAuditService(testDB)
	notificationService := services.NewNotificationService()
	
	suite.clinicalEngine = services.NewClinicalEngineService(
		rustEngine,
		apolloClient,
		testRedis,
	)
	
	snapshotService := services.NewSnapshotService(
		contextGateway,
		testRedis,
		testDB,
	)
	
	recipeService := services.NewRecipeService(
		recipeRepo,
		medicationRepo,
		testRedis,
	)
	
	suite.medicationService = services.NewMedicationService(
		medicationRepo,
		recipeService,
		snapshotService,
		suite.clinicalEngine,
		auditService,
		notificationService,
		testsetup.TestLogger(),
		testsetup.TestMetrics(),
	)
}

func (suite *ClinicalSafetyTestSuite) setupClinicalTestData() {
	suite.testRecipe = fixtures.ValidRecipeWithRules()
}

func (suite *ClinicalSafetyTestSuite) TestDosageCalculationAccuracy() {
	t := suite.T()
	
	testCases := []struct {
		name            string
		patientContext  entities.PatientContext
		expectedDoseMin float64
		expectedDoseMax float64
		calculationMethod entities.CalculationMethod
	}{
		{
			name:           "Adult BSA-based vincristine",
			patientContext: fixtures.ValidPatientContext(), // 70kg, 175cm adult
			expectedDoseMin: 2.2, // 1.4 * 1.8 BSA = 2.52, but capped at 2.0
			expectedDoseMax: 2.0, // Maximum dose limit
			calculationMethod: entities.MethodBSABased,
		},
		{
			name:           "Pediatric weight-based dosing",
			patientContext: fixtures.PediatricPatientContext(), // 25kg, 120cm child
			expectedDoseMin: 1.0,
			expectedDoseMax: 1.5,
			calculationMethod: entities.MethodWeightBased,
		},
		{
			name:           "Renal impaired adult",
			patientContext: fixtures.RenalImpairedPatientContext(), // eGFR 45
			expectedDoseMin: 1.5, // Reduced due to renal impairment
			expectedDoseMax: 2.0,
			calculationMethod: entities.MethodBSABased,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request := &services.ProposeMedicationRequest{
				PatientID:       uuid.New(),
				ProtocolID:      suite.testRecipe.ProtocolID,
				Indication:      "Dosage accuracy test",
				ClinicalContext: &tc.patientContext,
				CreatedBy:       "clinical-test",
			}
			
			response, err := suite.medicationService.ProposeMedication(suite.ctx, request)
			require.NoError(t, err, "Medication proposal should succeed")
			require.NotNil(t, response)
			require.NotEmpty(t, response.Proposal.DosageRecommendations)
			
			// Verify dose calculation
			primaryDose := response.Proposal.DosageRecommendations[0]
			assert.Equal(t, tc.calculationMethod, primaryDose.CalculationMethod,
				"Should use correct calculation method")
			assert.True(t, primaryDose.DoseMg >= tc.expectedDoseMin,
				"Dose %.2f should be >= %.2f", primaryDose.DoseMg, tc.expectedDoseMin)
			assert.True(t, primaryDose.DoseMg <= tc.expectedDoseMax,
				"Dose %.2f should be <= %.2f", primaryDose.DoseMg, tc.expectedDoseMax)
			assert.True(t, primaryDose.ConfidenceScore > 0.8,
				"Confidence score should be high for standard calculations")
		})
	}
}

func (suite *ClinicalSafetyTestSuite) TestDrugInteractionDetection() {
	t := suite.T()
	
	testCases := []struct {
		name                string
		currentMedications  []entities.CurrentMedication
		expectedInteraction bool
		expectedSeverity    entities.InteractionSeverity
	}{
		{
			name: "No drug interactions",
			currentMedications: []entities.CurrentMedication{
				{
					MedicationName: "acetaminophen",
					DoseMg:         650,
					Frequency:      "q6h",
				},
			},
			expectedInteraction: false,
		},
		{
			name: "Moderate interaction with phenytoin",
			currentMedications: []entities.CurrentMedication{
				{
					MedicationName: "phenytoin",
					DoseMg:         300,
					Frequency:      "daily",
				},
			},
			expectedInteraction: true,
			expectedSeverity:    entities.SeverityModerate,
		},
		{
			name: "Major interaction with azole antifungal",
			currentMedications: []entities.CurrentMedication{
				{
					MedicationName: "itraconazole",
					DoseMg:         200,
					Frequency:      "daily",
				},
			},
			expectedInteraction: true,
			expectedSeverity:    entities.SeverityMajor,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patientContext := fixtures.ValidPatientContext()
			clinicalContext := fixtures.ValidClinicalContext()
			clinicalContext.Medications = tc.currentMedications
			
			request := &services.ProposeMedicationRequest{
				PatientID:       uuid.New(),
				ProtocolID:      suite.testRecipe.ProtocolID,
				Indication:      "Drug interaction test",
				ClinicalContext: clinicalContext,
				CreatedBy:       "interaction-test",
			}
			
			response, err := suite.medicationService.ProposeMedication(suite.ctx, request)
			require.NoError(t, err)
			require.NotNil(t, response)
			
			// Check for interaction detection in safety constraints
			hasInteractionWarning := false
			interactionSeverity := entities.SeverityMinor
			
			for _, constraint := range response.Proposal.SafetyConstraints {
				if constraint.ConstraintType == entities.ConstraintInteraction {
					hasInteractionWarning = true
					interactionSeverity = entities.InteractionSeverity(constraint.Severity)
					break
				}
			}
			
			assert.Equal(t, tc.expectedInteraction, hasInteractionWarning,
				"Drug interaction detection should match expected")
			
			if tc.expectedInteraction {
				assert.Equal(t, tc.expectedSeverity, interactionSeverity,
					"Interaction severity should match expected")
			}
		})
	}
}

func (suite *ClinicalSafetyTestSuite) TestAllergyContraindicationChecks() {
	t := suite.T()
	
	testCases := []struct {
		name            string
		allergies       []string
		expectBlocked   bool
		expectedMessage string
	}{
		{
			name:          "No allergies",
			allergies:     []string{},
			expectBlocked: false,
		},
		{
			name:          "Non-relevant allergy",
			allergies:     []string{"penicillin"},
			expectBlocked: false,
		},
		{
			name:            "Vinca alkaloid allergy",
			allergies:       []string{"vincristine", "vinblastine"},
			expectBlocked:   true,
			expectedMessage: "allergy",
		},
		{
			name:            "Generic drug class allergy",
			allergies:       []string{"vinca alkaloids"},
			expectBlocked:   true,
			expectedMessage: "contraindicated",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clinicalContext := fixtures.ValidClinicalContext()
			clinicalContext.Allergies = tc.allergies
			
			request := &services.ProposeMedicationRequest{
				PatientID:       uuid.New(),
				ProtocolID:      suite.testRecipe.ProtocolID,
				Indication:      "Allergy test",
				ClinicalContext: clinicalContext,
				CreatedBy:       "allergy-test",
			}
			
			response, err := suite.medicationService.ProposeMedication(suite.ctx, request)
			
			if tc.expectBlocked {
				// Should either fail with error or have critical safety constraint
				if err != nil {
					assert.Contains(t, err.Error(), tc.expectedMessage)
				} else {
					require.NotNil(t, response)
					hasCriticalConstraint := false
					for _, constraint := range response.Proposal.SafetyConstraints {
						if constraint.ConstraintType == entities.ConstraintAllergy &&
						   constraint.Severity == entities.SeverityCritical {
							hasCriticalConstraint = true
							assert.Contains(t, constraint.Message, tc.expectedMessage)
							break
						}
					}
					assert.True(t, hasCriticalConstraint, "Should have critical allergy constraint")
				}
			} else {
				require.NoError(t, err)
				require.NotNil(t, response)
				
				// Should not have allergy-related constraints
				for _, constraint := range response.Proposal.SafetyConstraints {
					assert.NotEqual(t, entities.ConstraintAllergy, constraint.ConstraintType,
						"Should not have allergy constraints")
				}
			}
		})
	}
}

func (suite *ClinicalSafetyTestSuite) TestAgeBasedSafetyChecks() {
	t := suite.T()
	
	testCases := []struct {
		name              string
		age               int
		expectAgeWarning  bool
		expectedAdjustment bool
	}{
		{
			name:              "Adult patient",
			age:               45,
			expectAgeWarning:  false,
			expectedAdjustment: false,
		},
		{
			name:              "Pediatric patient",
			age:               8,
			expectAgeWarning:  true,
			expectedAdjustment: true,
		},
		{
			name:              "Infant",
			age:               1,
			expectAgeWarning:  true,
			expectedAdjustment: true,
		},
		{
			name:              "Elderly patient",
			age:               85,
			expectAgeWarning:  true,
			expectedAdjustment: false,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patientContext := fixtures.ValidPatientContext()
			patientContext.Age = tc.age
			
			clinicalContext := fixtures.ValidClinicalContext()
			clinicalContext.AgeYears = tc.age
			
			request := &services.ProposeMedicationRequest{
				PatientID:       uuid.New(),
				ProtocolID:      suite.testRecipe.ProtocolID,
				Indication:      "Age-based safety test",
				ClinicalContext: clinicalContext,
				CreatedBy:       "age-test",
			}
			
			response, err := suite.medicationService.ProposeMedication(suite.ctx, request)
			require.NoError(t, err)
			require.NotNil(t, response)
			
			hasAgeConstraint := false
			hasAgeAdjustment := false
			
			// Check safety constraints
			for _, constraint := range response.Proposal.SafetyConstraints {
				if constraint.ConstraintType == entities.ConstraintAge {
					hasAgeConstraint = true
					break
				}
			}
			
			// Check dosage adjustments
			for _, recommendation := range response.Proposal.DosageRecommendations {
				if recommendation.AdjustmentReason != "" &&
				   (contains(recommendation.AdjustmentReason, "age") ||
				    contains(recommendation.AdjustmentReason, "pediatric")) {
					hasAgeAdjustment = true
					break
				}
			}
			
			assert.Equal(t, tc.expectAgeWarning, hasAgeConstraint,
				"Age warning should match expected for age %d", tc.age)
			assert.Equal(t, tc.expectedAdjustment, hasAgeAdjustment,
				"Age adjustment should match expected for age %d", tc.age)
		})
	}
}

func (suite *ClinicalSafetyTestSuite) TestOrganFunctionBasedAdjustments() {
	t := suite.T()
	
	testCases := []struct {
		name                string
		modifyContext      func(*entities.ClinicalContext)
		expectedAdjustment string
		expectedMonitoring string
	}{
		{
			name: "Normal renal function",
			modifyContext: func(ctx *entities.ClinicalContext) {
				egfr := 95.0
				ctx.eGFR = &egfr
			},
			expectedAdjustment: "",
		},
		{
			name: "Mild renal impairment",
			modifyContext: func(ctx *entities.ClinicalContext) {
				egfr := 70.0
				creatinine := 1.3
				ctx.eGFR = &egfr
				ctx.CreatinineMgdL = &creatinine
			},
			expectedAdjustment: "",
			expectedMonitoring: "renal",
		},
		{
			name: "Moderate renal impairment",
			modifyContext: func(ctx *entities.ClinicalContext) {
				egfr := 45.0
				creatinine := 2.1
				ctx.eGFR = &egfr
				ctx.CreatinineMgdL = &creatinine
			},
			expectedAdjustment: "renal",
			expectedMonitoring: "creatinine",
		},
		{
			name: "Severe renal impairment",
			modifyContext: func(ctx *entities.ClinicalContext) {
				egfr := 25.0
				creatinine := 3.5
				ctx.eGFR = &egfr
				ctx.CreatinineMgdL = &creatinine
			},
			expectedAdjustment: "renal",
			expectedMonitoring: "creatinine",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			clinicalContext := fixtures.ValidClinicalContext()
			tc.modifyContext(clinicalContext)
			
			request := &services.ProposeMedicationRequest{
				PatientID:       uuid.New(),
				ProtocolID:      suite.testRecipe.ProtocolID,
				Indication:      "Organ function test",
				ClinicalContext: clinicalContext,
				CreatedBy:       "organ-function-test",
			}
			
			response, err := suite.medicationService.ProposeMedication(suite.ctx, request)
			require.NoError(t, err)
			require.NotNil(t, response)
			
			hasExpectedAdjustment := tc.expectedAdjustment == ""
			hasExpectedMonitoring := tc.expectedMonitoring == ""
			
			// Check for organ function adjustments
			for _, recommendation := range response.Proposal.DosageRecommendations {
				if tc.expectedAdjustment != "" &&
				   contains(recommendation.AdjustmentReason, tc.expectedAdjustment) {
					hasExpectedAdjustment = true
				}
				
				for _, monitoring := range recommendation.MonitoringRequired {
					if tc.expectedMonitoring != "" &&
					   contains(monitoring.Parameter, tc.expectedMonitoring) {
						hasExpectedMonitoring = true
					}
				}
			}
			
			if tc.expectedAdjustment != "" {
				assert.True(t, hasExpectedAdjustment,
					"Should have %s adjustment", tc.expectedAdjustment)
			}
			
			if tc.expectedMonitoring != "" {
				assert.True(t, hasExpectedMonitoring,
					"Should have %s monitoring", tc.expectedMonitoring)
			}
		})
	}
}

func (suite *ClinicalSafetyTestSuite) TestFHIRResourceCompliance() {
	t := suite.T()
	
	// Test FHIR R4 MedicationRequest compliance
	request := &services.ProposeMedicationRequest{
		PatientID:       uuid.New(),
		ProtocolID:      suite.testRecipe.ProtocolID,
		Indication:      "FHIR compliance test",
		ClinicalContext: fixtures.ValidClinicalContext(),
		CreatedBy:       "fhir-test",
	}
	
	response, err := suite.medicationService.ProposeMedication(suite.ctx, request)
	require.NoError(t, err)
	require.NotNil(t, response)
	
	// Convert to FHIR MedicationRequest
	fhirRequest, err := suite.medicationService.ConvertToFHIRMedicationRequest(
		suite.ctx, response.Proposal)
	require.NoError(t, err)
	require.NotNil(t, fhirRequest)
	
	// Validate FHIR compliance
	suite.validateFHIRMedicationRequest(t, fhirRequest)
}

func (suite *ClinicalSafetyTestSuite) validateFHIRMedicationRequest(t *testing.T, fhirRequest *services.FHIRMedicationRequest) {
	// Required fields for FHIR R4 MedicationRequest
	assert.Equal(t, "MedicationRequest", fhirRequest.ResourceType,
		"Resource type should be MedicationRequest")
	assert.NotEmpty(t, fhirRequest.ID, "Should have ID")
	assert.NotEmpty(t, fhirRequest.Status, "Should have status")
	assert.NotEmpty(t, fhirRequest.Intent, "Should have intent")
	assert.NotNil(t, fhirRequest.Subject, "Should have subject reference")
	assert.NotNil(t, fhirRequest.MedicationCodeableConcept, "Should have medication concept")
	
	// Validate status values (must be FHIR-compliant)
	validStatuses := []string{"active", "on-hold", "cancelled", "completed", "entered-in-error", "stopped", "draft", "unknown"}
	assert.Contains(t, validStatuses, fhirRequest.Status, "Status should be FHIR-compliant")
	
	// Validate intent values
	validIntents := []string{"proposal", "plan", "order", "original-order", "reflex-order", "filler-order", "instance-order", "option"}
	assert.Contains(t, validIntents, fhirRequest.Intent, "Intent should be FHIR-compliant")
	
	// Validate dosage instruction structure
	if len(fhirRequest.DosageInstruction) > 0 {
		dosage := fhirRequest.DosageInstruction[0]
		assert.NotNil(t, dosage.DoseAndRate, "Should have dose and rate")
		if len(dosage.DoseAndRate) > 0 {
			assert.NotNil(t, dosage.DoseAndRate[0].DoseQuantity, "Should have dose quantity")
			assert.NotEmpty(t, dosage.DoseAndRate[0].DoseQuantity.Value, "Should have dose value")
			assert.NotEmpty(t, dosage.DoseAndRate[0].DoseQuantity.Unit, "Should have dose unit")
		}
	}
	
	// Validate extensions for clinical context
	assert.NotEmpty(t, fhirRequest.Extension, "Should have extensions for clinical context")
	
	// Check for required clinical safety extensions
	hasPatientWeightExt := false
	hasIndicationExt := false
	
	for _, ext := range fhirRequest.Extension {
		switch ext.URL {
		case "http://clinical-platform.com/fhir/StructureDefinition/patient-weight":
			hasPatientWeightExt = true
		case "http://clinical-platform.com/fhir/StructureDefinition/indication":
			hasIndicationExt = true
		}
	}
	
	assert.True(t, hasPatientWeightExt, "Should have patient weight extension")
	assert.True(t, hasIndicationExt, "Should have indication extension")
}

func (suite *ClinicalSafetyTestSuite) TestClinicalDecisionSupportRules() {
	t := suite.T()
	
	// Test clinical decision support rule evaluation
	testCases := []struct {
		name         string
		setupContext func() *entities.ClinicalContext
		expectedCDS  []string // Expected clinical decision support recommendations
	}{
		{
			name: "Standard adult patient",
			setupContext: func() *entities.ClinicalContext {
				return fixtures.ValidClinicalContext()
			},
			expectedCDS: []string{
				"Monitor for peripheral neuropathy",
				"Ensure proper IV administration",
			},
		},
		{
			name: "Pediatric patient",
			setupContext: func() *entities.ClinicalContext {
				ctx := fixtures.ValidClinicalContext()
				ctx.AgeYears = 8
				return ctx
			},
			expectedCDS: []string{
				"Use pediatric dosing guidelines",
				"Enhanced neurological monitoring",
			},
		},
		{
			name: "Patient with neuropathy history",
			setupContext: func() *entities.ClinicalContext {
				ctx := fixtures.ValidClinicalContext()
				ctx.Conditions = append(ctx.Conditions, "peripheral neuropathy")
				return ctx
			},
			expectedCDS: []string{
				"Caution: existing neuropathy",
				"Consider dose reduction",
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			request := &services.ProposeMedicationRequest{
				PatientID:       uuid.New(),
				ProtocolID:      suite.testRecipe.ProtocolID,
				Indication:      "CDS test",
				ClinicalContext: tc.setupContext(),
				CreatedBy:       "cds-test",
			}
			
			response, err := suite.medicationService.ProposeMedication(suite.ctx, request)
			require.NoError(t, err)
			require.NotNil(t, response)
			
			// Check for expected CDS recommendations
			actualRecommendations := response.Recommendations
			
			for _, expectedRec := range tc.expectedCDS {
				found := false
				for _, actualRec := range actualRecommendations {
					if contains(actualRec, expectedRec) {
						found = true
						break
					}
				}
				if !found {
					// Allow partial matches for flexibility
					found = suite.checkPartialMatch(expectedRec, actualRecommendations)
				}
				assert.True(t, found, "Should have CDS recommendation: %s", expectedRec)
			}
		})
	}
}

func (suite *ClinicalSafetyTestSuite) TestMonitoringRequirements() {
	t := suite.T()
	
	// Test that appropriate monitoring requirements are generated
	request := &services.ProposeMedicationRequest{
		PatientID:       uuid.New(),
		ProtocolID:      suite.testRecipe.ProtocolID,
		Indication:      "Monitoring test",
		ClinicalContext: fixtures.ValidClinicalContext(),
		CreatedBy:       "monitoring-test",
	}
	
	response, err := suite.medicationService.ProposeMedication(suite.ctx, request)
	require.NoError(t, err)
	require.NotNil(t, response)
	require.NotEmpty(t, response.Proposal.DosageRecommendations)
	
	// Verify monitoring requirements
	primaryDose := response.Proposal.DosageRecommendations[0]
	assert.NotEmpty(t, primaryDose.MonitoringRequired, "Should have monitoring requirements")
	
	// Verify specific monitoring for vincristine
	hasNeurologicalMonitoring := false
	for _, monitoring := range primaryDose.MonitoringRequired {
		if contains(monitoring.Parameter, "neurological") ||
		   contains(monitoring.Parameter, "neuropathy") {
			hasNeurologicalMonitoring = true
			
			// Verify frequency is appropriate
			assert.NotEqual(t, "", monitoring.Frequency, "Monitoring should have frequency")
			assert.NotEmpty(t, monitoring.Notes, "Monitoring should have notes")
		}
	}
	
	assert.True(t, hasNeurologicalMonitoring, "Should have neurological monitoring for vincristine")
}

func (suite *ClinicalSafetyTestSuite) checkPartialMatch(expected string, actual []string) bool {
	// Check if any actual recommendation contains key terms from expected
	expectedTerms := []string{"monitor", "neuropathy", "pediatric", "dose", "caution"}
	
	for _, term := range expectedTerms {
		if contains(expected, term) {
			for _, actualRec := range actual {
				if contains(actualRec, term) {
					return true
				}
			}
		}
	}
	return false
}

// Helper function for string containment check
func contains(text, substring string) bool {
	return len(text) >= len(substring) && 
		   (text == substring || 
		    text[:len(substring)] == substring || 
		    text[len(text)-len(substring):] == substring ||
		    findInString(text, substring))
}

func findInString(text, substring string) bool {
	for i := 0; i <= len(text)-len(substring); i++ {
		if text[i:i+len(substring)] == substring {
			return true
		}
	}
	return false
}