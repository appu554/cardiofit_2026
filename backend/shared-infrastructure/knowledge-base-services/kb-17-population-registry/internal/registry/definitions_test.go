package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"kb-17-population-registry/internal/models"
)

func TestGetAllRegistryDefinitions(t *testing.T) {
	registries := GetAllRegistryDefinitions()

	// Should have 8 predefined registries
	assert.Len(t, registries, 8)

	// Check each registry type exists
	codes := make(map[models.RegistryCode]bool)
	for _, reg := range registries {
		codes[reg.Code] = true
	}

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

	for _, code := range expectedCodes {
		assert.True(t, codes[code], "Missing registry: %s", code)
	}
}

func TestGetDiabetesRegistry(t *testing.T) {
	registry := GetDiabetesRegistry()

	assert.Equal(t, models.RegistryDiabetes, registry.Code)
	assert.Equal(t, "Diabetes Mellitus Registry", registry.Name)
	assert.True(t, registry.Active)

	// Should have inclusion criteria
	assert.NotEmpty(t, registry.InclusionCriteria)

	// Check for ICD-10 diabetes codes
	found := false
	for _, group := range registry.InclusionCriteria {
		for _, criterion := range group.Criteria {
			if criterion.Type == models.CriteriaTypeDiagnosis {
				if val, ok := criterion.Value.(string); ok {
					if val == "E10" || val == "E11" || val == "E13" {
						found = true
						break
					}
				}
			}
		}
	}
	assert.True(t, found, "Diabetes registry should include E10/E11/E13 ICD-10 codes")

	// Should have risk stratification rules
	assert.NotNil(t, registry.RiskStratification)
}

func TestGetHypertensionRegistry(t *testing.T) {
	registry := GetHypertensionRegistry()

	assert.Equal(t, models.RegistryHypertension, registry.Code)
	assert.Equal(t, "Hypertension Registry", registry.Name)
	assert.True(t, registry.Active)

	// Check for ICD-10 hypertension codes
	found := false
	for _, group := range registry.InclusionCriteria {
		for _, criterion := range group.Criteria {
			if criterion.Type == models.CriteriaTypeDiagnosis {
				if val, ok := criterion.Value.(string); ok {
					if val == "I10" || val == "I11" || val == "I12" || val == "I13" {
						found = true
						break
					}
				}
			}
		}
	}
	assert.True(t, found, "Hypertension registry should include I10-I13 ICD-10 codes")
}

func TestGetHeartFailureRegistry(t *testing.T) {
	registry := GetHeartFailureRegistry()

	assert.Equal(t, models.RegistryHeartFailure, registry.Code)
	assert.Equal(t, "Heart Failure Registry", registry.Name)
	assert.True(t, registry.Active)

	// Check for heart failure ICD-10 codes
	found := false
	for _, group := range registry.InclusionCriteria {
		for _, criterion := range group.Criteria {
			if criterion.Type == models.CriteriaTypeDiagnosis {
				if val, ok := criterion.Value.(string); ok {
					if val == "I50" {
						found = true
						break
					}
				}
			}
		}
	}
	assert.True(t, found, "Heart Failure registry should include I50 ICD-10 codes")
}

func TestGetCKDRegistry(t *testing.T) {
	registry := GetCKDRegistry()

	assert.Equal(t, models.RegistryCKD, registry.Code)
	assert.Equal(t, "Chronic Kidney Disease Registry", registry.Name)
	assert.True(t, registry.Active)

	// Check for CKD ICD-10 codes
	found := false
	for _, group := range registry.InclusionCriteria {
		for _, criterion := range group.Criteria {
			if criterion.Type == models.CriteriaTypeDiagnosis {
				if val, ok := criterion.Value.(string); ok {
					if val == "N18" {
						found = true
						break
					}
				}
			}
		}
	}
	assert.True(t, found, "CKD registry should include N18 ICD-10 codes")
}

func TestGetCOPDRegistry(t *testing.T) {
	registry := GetCOPDRegistry()

	assert.Equal(t, models.RegistryCOPD, registry.Code)
	assert.Equal(t, "COPD Registry", registry.Name)
	assert.True(t, registry.Active)
}

func TestGetPregnancyRegistry(t *testing.T) {
	registry := GetPregnancyRegistry()

	assert.Equal(t, models.RegistryPregnancy, registry.Code)
	assert.Equal(t, "Pregnancy Registry", registry.Name)
	assert.True(t, registry.Active)
}

func TestGetOpioidUseRegistry(t *testing.T) {
	registry := GetOpioidUseRegistry()

	assert.Equal(t, models.RegistryOpioidUse, registry.Code)
	assert.Equal(t, "Opioid Use Disorder Registry", registry.Name)
	assert.True(t, registry.Active)
}

func TestGetAnticoagulationRegistry(t *testing.T) {
	registry := GetAnticoagulationRegistry()

	assert.Equal(t, models.RegistryAnticoagulation, registry.Code)
	assert.Equal(t, "Anticoagulation Management Registry", registry.Name)
	assert.True(t, registry.Active)
}

func TestRegistryDefinitions_HaveRequiredFields(t *testing.T) {
	registries := GetAllRegistryDefinitions()

	for _, reg := range registries {
		t.Run(string(reg.Code), func(t *testing.T) {
			// Required fields
			assert.NotEmpty(t, reg.Code)
			assert.NotEmpty(t, reg.Name)
			assert.NotEmpty(t, reg.Description)
			assert.True(t, reg.Active, "Default registries should be active")

			// Should have inclusion criteria
			assert.NotEmpty(t, reg.InclusionCriteria, "Registry should have inclusion criteria")

			// Each criteria group should have criteria
			for _, group := range reg.InclusionCriteria {
				assert.NotEmpty(t, group.ID)
				assert.NotEmpty(t, group.Operator)
				assert.NotEmpty(t, group.Criteria)
			}
		})
	}
}

func TestRegistryDefinitions_ValidCodeSystems(t *testing.T) {
	registries := GetAllRegistryDefinitions()

	validCodeSystems := map[models.CodeSystem]bool{
		models.CodeSystemICD10:   true,
		models.CodeSystemICD10CM: true,
		models.CodeSystemSNOMED:  true,
		models.CodeSystemLOINC:   true,
		models.CodeSystemRxNorm:  true,
		models.CodeSystemCPT:     true,
		models.CodeSystemHCPCS:   true,
	}

	for _, reg := range registries {
		for _, group := range reg.InclusionCriteria {
			for _, criterion := range group.Criteria {
				if criterion.CodeSystem != "" {
					_, valid := validCodeSystems[criterion.CodeSystem]
					assert.True(t, valid, "Invalid code system %s in registry %s",
						criterion.CodeSystem, reg.Code)
				}
			}
		}
	}
}
