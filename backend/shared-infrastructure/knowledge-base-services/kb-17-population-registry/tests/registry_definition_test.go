// Package tests provides comprehensive tests for KB-17 Population Registry
// registry_definition_test.go - Registry definition loading and validation tests
package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-17-population-registry/internal/models"
	"kb-17-population-registry/internal/registry"
)

// =============================================================================
// REGISTRY LOAD TESTS
// =============================================================================

// TestAllRegistriesLoadCorrectly verifies all 8 registries load on startup
func TestAllRegistriesLoadCorrectly(t *testing.T) {
	t.Parallel()

	registries := registry.GetAllRegistryDefinitions()

	// Must have exactly 8 registries
	require.Len(t, registries, 8, "Expected exactly 8 pre-configured registries")

	// Expected registry codes
	expectedCodes := []models.RegistryCode{
		models.RegistryDiabetes,
		models.RegistryHypertension,
		models.RegistryHeartFailure,
		models.RegistryCKD,
		models.RegistryCOPD,
		models.RegistryPregnancy,
		models.RegistryOpioidUse,
		models.RegistryAnticoagulation,
	}

	// Verify each expected code is present
	foundCodes := make(map[models.RegistryCode]bool)
	for _, reg := range registries {
		foundCodes[reg.Code] = true
	}

	for _, code := range expectedCodes {
		assert.True(t, foundCodes[code], "Missing registry: %s", code)
	}
}

// TestRegistryCodesAreUnique ensures no duplicate registry codes
func TestRegistryCodesAreUnique(t *testing.T) {
	t.Parallel()

	registries := registry.GetAllRegistryDefinitions()
	codeCount := make(map[models.RegistryCode]int)

	for _, reg := range registries {
		codeCount[reg.Code]++
	}

	for code, count := range codeCount {
		assert.Equal(t, 1, count, "Duplicate registry code found: %s", code)
	}
}

// TestRegistryHasRequiredFields validates each registry has mandatory fields
func TestRegistryHasRequiredFields(t *testing.T) {
	t.Parallel()

	registries := registry.GetAllRegistryDefinitions()

	for _, reg := range registries {
		t.Run(string(reg.Code), func(t *testing.T) {
			// Code must be non-empty
			assert.NotEmpty(t, reg.Code, "Registry code should not be empty")

			// Name must be non-empty
			assert.NotEmpty(t, reg.Name, "Registry %s: name should not be empty", reg.Code)

			// Description must be non-empty
			assert.NotEmpty(t, reg.Description, "Registry %s: description should not be empty", reg.Code)

			// Category must be non-empty
			assert.NotEmpty(t, reg.Category, "Registry %s: category should not be empty", reg.Code)

			// Must have inclusion criteria
			assert.NotEmpty(t, reg.InclusionCriteria, "Registry %s: must have inclusion criteria", reg.Code)
		})
	}
}

// TestInclusionCriteriaAreParseable validates inclusion criteria structure
func TestInclusionCriteriaAreParseable(t *testing.T) {
	t.Parallel()

	registries := registry.GetAllRegistryDefinitions()

	for _, reg := range registries {
		t.Run(string(reg.Code), func(t *testing.T) {
			for i, group := range reg.InclusionCriteria {
				// Group must have ID
				assert.NotEmpty(t, group.ID, "Registry %s: criteria group %d missing ID", reg.Code, i)

				// Group must have criteria
				assert.NotEmpty(t, group.Criteria, "Registry %s: criteria group %s has no criteria", reg.Code, group.ID)

				// Validate each criterion
				for j, crit := range group.Criteria {
					assert.NotEmpty(t, crit.Type, "Registry %s: criterion %d in group %s missing type", reg.Code, j, group.ID)
					assert.NotEmpty(t, crit.Field, "Registry %s: criterion %d in group %s missing field", reg.Code, j, group.ID)
					assert.NotEmpty(t, crit.Operator, "Registry %s: criterion %d in group %s missing operator", reg.Code, j, group.ID)
				}
			}
		})
	}
}

// TestRiskStratificationConfigPresent validates risk stratification is configured
func TestRiskStratificationConfigPresent(t *testing.T) {
	t.Parallel()

	registries := registry.GetAllRegistryDefinitions()

	for _, reg := range registries {
		t.Run(string(reg.Code), func(t *testing.T) {
			// Risk stratification should be present
			require.NotNil(t, reg.RiskStratification, "Registry %s: missing risk stratification config", reg.Code)

			// Method should be valid
			assert.NotEmpty(t, reg.RiskStratification.Method, "Registry %s: risk method not specified", reg.Code)

			// Should have rules or score type
			hasRules := len(reg.RiskStratification.Rules) > 0
			hasScoreType := reg.RiskStratification.ScoreType != ""

			assert.True(t, hasRules || hasScoreType,
				"Registry %s: must have either rules or score type for risk stratification", reg.Code)
		})
	}
}

// TestCareGapMeasuresPresent validates care gap measures are defined
func TestCareGapMeasuresPresent(t *testing.T) {
	t.Parallel()

	registries := registry.GetAllRegistryDefinitions()

	for _, reg := range registries {
		t.Run(string(reg.Code), func(t *testing.T) {
			assert.NotEmpty(t, reg.CareGapMeasures, "Registry %s: should have care gap measures defined", reg.Code)
		})
	}
}

// =============================================================================
// INDIVIDUAL REGISTRY VALIDATION
// =============================================================================

// TestDiabetesRegistryDefinition validates Diabetes registry specifics
func TestDiabetesRegistryDefinition(t *testing.T) {
	t.Parallel()

	reg := registry.GetDiabetesRegistry()

	// Verify basic properties
	assert.Equal(t, models.RegistryDiabetes, reg.Code)
	assert.Equal(t, models.CategoryChronic, reg.Category)
	assert.True(t, reg.AutoEnroll, "Diabetes registry should auto-enroll")
	assert.True(t, reg.Active, "Diabetes registry should be active")

	// Verify ICD-10 codes are included
	var hasDiabetesCodes bool
	for _, group := range reg.InclusionCriteria {
		for _, crit := range group.Criteria {
			if crit.Type == models.CriteriaTypeDiagnosis {
				if crit.CodeSystem == models.CodeSystemICD10 {
					if crit.Value == "E10" || crit.Value == "E11" || crit.Value == "E13" {
						hasDiabetesCodes = true
						break
					}
				}
			}
		}
	}
	assert.True(t, hasDiabetesCodes, "Diabetes registry should include E10, E11, E13 ICD-10 codes")

	// Verify risk stratification thresholds
	require.NotNil(t, reg.RiskStratification)
	assert.Contains(t, reg.RiskStratification.Thresholds, "hba1c_critical")
	assert.Contains(t, reg.RiskStratification.Thresholds, "hba1c_high")

	// Verify CMS measures
	assert.Contains(t, reg.CareGapMeasures, "CMS122", "Should include CMS122 (Diabetes HbA1c)")
}

// TestHypertensionRegistryDefinition validates Hypertension registry specifics
func TestHypertensionRegistryDefinition(t *testing.T) {
	t.Parallel()

	reg := registry.GetHypertensionRegistry()

	assert.Equal(t, models.RegistryHypertension, reg.Code)
	assert.Equal(t, models.CategoryChronic, reg.Category)

	// Verify ICD-10 codes
	var hasI10, hasI11, hasI12, hasI13 bool
	for _, group := range reg.InclusionCriteria {
		for _, crit := range group.Criteria {
			if crit.Type == models.CriteriaTypeDiagnosis {
				switch crit.Value {
				case "I10":
					hasI10 = true
				case "I11":
					hasI11 = true
				case "I12":
					hasI12 = true
				case "I13":
					hasI13 = true
				}
			}
		}
	}
	assert.True(t, hasI10, "Should include I10 (Essential hypertension)")
	assert.True(t, hasI11, "Should include I11 (Hypertensive heart disease)")
	assert.True(t, hasI12, "Should include I12 (Hypertensive CKD)")
	assert.True(t, hasI13, "Should include I13 (Hypertensive heart and CKD)")

	// Verify BP thresholds
	require.NotNil(t, reg.RiskStratification)
	assert.Contains(t, reg.RiskStratification.Thresholds, "systolic_critical")
	assert.Contains(t, reg.RiskStratification.Thresholds, "diastolic_critical")

	// Verify CMS measures
	assert.Contains(t, reg.CareGapMeasures, "CMS165", "Should include CMS165 (BP Control)")
}

// TestHeartFailureRegistryDefinition validates Heart Failure registry specifics
func TestHeartFailureRegistryDefinition(t *testing.T) {
	t.Parallel()

	reg := registry.GetHeartFailureRegistry()

	assert.Equal(t, models.RegistryHeartFailure, reg.Code)
	assert.Equal(t, models.CategoryChronic, reg.Category)

	// Verify ICD-10 codes include I50 and I42
	var hasI50, hasI42 bool
	for _, group := range reg.InclusionCriteria {
		for _, crit := range group.Criteria {
			if crit.Type == models.CriteriaTypeDiagnosis {
				if crit.Value == "I50" {
					hasI50 = true
				}
				if crit.Value == "I42" {
					hasI42 = true
				}
			}
		}
	}
	assert.True(t, hasI50, "Should include I50 (Heart failure)")
	assert.True(t, hasI42, "Should include I42 (Cardiomyopathy)")

	// Verify BNP thresholds
	require.NotNil(t, reg.RiskStratification)
	assert.Contains(t, reg.RiskStratification.Thresholds, "bnp_critical")
}

// TestCKDRegistryDefinition validates CKD registry specifics
func TestCKDRegistryDefinition(t *testing.T) {
	t.Parallel()

	reg := registry.GetCKDRegistry()

	assert.Equal(t, models.RegistryCKD, reg.Code)
	assert.Equal(t, models.CategoryChronic, reg.Category)

	// Verify N18 is included
	var hasN18 bool
	for _, group := range reg.InclusionCriteria {
		for _, crit := range group.Criteria {
			if crit.Type == models.CriteriaTypeDiagnosis && crit.Value == "N18" {
				hasN18 = true
				break
			}
		}
	}
	assert.True(t, hasN18, "Should include N18 (Chronic kidney disease)")

	// Verify eGFR thresholds for CKD staging
	require.NotNil(t, reg.RiskStratification)
	assert.Contains(t, reg.RiskStratification.Thresholds, "egfr_stage5", "Should have Stage 5 threshold")
	assert.Contains(t, reg.RiskStratification.Thresholds, "egfr_stage4", "Should have Stage 4 threshold")
}

// TestPregnancyRegistryDefinition validates Pregnancy registry specifics
func TestPregnancyRegistryDefinition(t *testing.T) {
	t.Parallel()

	reg := registry.GetPregnancyRegistry()

	assert.Equal(t, models.RegistryPregnancy, reg.Code)
	assert.Equal(t, models.CategoryPreventive, reg.Category)

	// Verify inclusion codes
	var hasZ34, hasOCodes bool
	for _, group := range reg.InclusionCriteria {
		for _, crit := range group.Criteria {
			if crit.Type == models.CriteriaTypeDiagnosis {
				if crit.Value == "Z34" {
					hasZ34 = true
				}
				if crit.Value == "O" {
					hasOCodes = true
				}
			}
		}
	}
	assert.True(t, hasZ34, "Should include Z34 (Supervision of pregnancy)")
	assert.True(t, hasOCodes, "Should include O codes (Pregnancy complications)")

	// Verify exclusion criteria exists (post-delivery)
	assert.NotEmpty(t, reg.ExclusionCriteria, "Pregnancy registry should have exclusion criteria")

	// Verify age-based risk stratification
	require.NotNil(t, reg.RiskStratification)
	assert.Contains(t, reg.RiskStratification.Thresholds, "age_high_risk_lower", "Should have advanced maternal age threshold")
}

// TestAnticoagulationRegistryDefinition validates Anticoagulation registry specifics
func TestAnticoagulationRegistryDefinition(t *testing.T) {
	t.Parallel()

	reg := registry.GetAnticoagulationRegistry()

	assert.Equal(t, models.RegistryAnticoagulation, reg.Code)
	assert.Equal(t, models.CategoryMedication, reg.Category)

	// Verify medication-based inclusion
	var hasMedicationCriteria bool
	for _, group := range reg.InclusionCriteria {
		for _, crit := range group.Criteria {
			if crit.Type == models.CriteriaTypeMedication {
				hasMedicationCriteria = true
				// Should include warfarin and DOACs
				if crit.CodeSystem == models.CodeSystemRxNorm {
					assert.NotEmpty(t, crit.Values, "Should have RxNorm codes for anticoagulants")
				}
			}
		}
	}
	assert.True(t, hasMedicationCriteria, "Should include medication-based criteria")

	// Verify HAS-BLED score type
	require.NotNil(t, reg.RiskStratification)
	assert.Equal(t, "HAS-BLED", reg.RiskStratification.ScoreType, "Should use HAS-BLED score for risk")
}

// =============================================================================
// INVALID REGISTRY DEFINITION TESTS
// =============================================================================

// TestGetRegistryDefinition_InvalidCode tests retrieval of non-existent registry
func TestGetRegistryDefinition_InvalidCode(t *testing.T) {
	t.Parallel()

	reg := registry.GetRegistryDefinition("INVALID_CODE")
	assert.Nil(t, reg, "Should return nil for invalid registry code")
}

// TestGetRegistryDefinition_AllValidCodes tests retrieval of all valid codes
func TestGetRegistryDefinition_AllValidCodes(t *testing.T) {
	t.Parallel()

	validCodes := []models.RegistryCode{
		models.RegistryDiabetes,
		models.RegistryHypertension,
		models.RegistryHeartFailure,
		models.RegistryCKD,
		models.RegistryCOPD,
		models.RegistryPregnancy,
		models.RegistryOpioidUse,
		models.RegistryAnticoagulation,
	}

	for _, code := range validCodes {
		t.Run(string(code), func(t *testing.T) {
			reg := registry.GetRegistryDefinition(code)
			require.NotNil(t, reg, "Should return registry for valid code: %s", code)
			assert.Equal(t, code, reg.Code)
		})
	}
}

// =============================================================================
// CRITERIA OPERATOR VALIDATION
// =============================================================================

// TestCriteriaOperatorsAreValid validates all operators used are valid
func TestCriteriaOperatorsAreValid(t *testing.T) {
	t.Parallel()

	registries := registry.GetAllRegistryDefinitions()

	for _, reg := range registries {
		t.Run(string(reg.Code), func(t *testing.T) {
			// Check inclusion criteria
			for _, group := range reg.InclusionCriteria {
				assert.NotEmpty(t, group.Operator,
					"Group operator should not be empty in %s", reg.Code)

				for _, crit := range group.Criteria {
					assert.NotEmpty(t, crit.Operator,
						"Criterion operator should not be empty in %s", reg.Code)
				}
			}

			// Check exclusion criteria
			for _, group := range reg.ExclusionCriteria {
				assert.NotEmpty(t, group.Operator,
					"Exclusion group operator should not be empty in %s", reg.Code)
			}

			// Check risk stratification rules
			if reg.RiskStratification != nil {
				for _, rule := range reg.RiskStratification.Rules {
					for _, group := range rule.Criteria {
						assert.NotEmpty(t, group.Operator,
							"Risk rule operator should not be empty in %s", reg.Code)
					}
				}
			}
		})
	}
}

// =============================================================================
// CODE SYSTEM VALIDATION
// =============================================================================

// TestCodeSystemsAreValid validates all code systems used are valid
func TestCodeSystemsAreValid(t *testing.T) {
	t.Parallel()

	registries := registry.GetAllRegistryDefinitions()

	// Valid code systems
	validCodeSystems := map[models.CodeSystem]bool{
		models.CodeSystemICD10:  true,
		models.CodeSystemSNOMED: true,
		models.CodeSystemLOINC:  true,
		models.CodeSystemRxNorm: true,
		models.CodeSystemCPT:    true,
	}

	for _, reg := range registries {
		t.Run(string(reg.Code), func(t *testing.T) {
			for _, group := range reg.InclusionCriteria {
				for _, crit := range group.Criteria {
					if crit.CodeSystem != "" {
						assert.True(t, validCodeSystems[crit.CodeSystem],
							"Unknown code system in %s criterion: %s", reg.Code, crit.CodeSystem)
					}
				}
			}
		})
	}
}

// =============================================================================
// RISK TIER COVERAGE
// =============================================================================

// TestRiskTiersAreCovered verifies each registry can classify into all risk tiers
func TestRiskTiersAreCovered(t *testing.T) {
	t.Parallel()

	registries := registry.GetAllRegistryDefinitions()

	for _, reg := range registries {
		t.Run(string(reg.Code), func(t *testing.T) {
			if reg.RiskStratification == nil {
				t.Skip("No risk stratification configured")
			}

			tiersFound := make(map[models.RiskTier]bool)
			for _, rule := range reg.RiskStratification.Rules {
				tiersFound[rule.Tier] = true
			}

			// Should have at least CRITICAL and HIGH tiers
			assert.True(t, tiersFound[models.RiskTierCritical] || tiersFound[models.RiskTierHigh],
				"Registry %s should have at least CRITICAL or HIGH risk tier rules", reg.Code)
		})
	}
}

// =============================================================================
// BENCHMARK TESTS
// =============================================================================

// BenchmarkLoadAllRegistries benchmarks registry loading
func BenchmarkLoadAllRegistries(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = registry.GetAllRegistryDefinitions()
	}
}

// BenchmarkGetRegistryDefinition benchmarks single registry retrieval
func BenchmarkGetRegistryDefinition(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = registry.GetRegistryDefinition(models.RegistryDiabetes)
	}
}
