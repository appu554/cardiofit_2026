package fixtures

import (
	"time"

	"github.com/google/uuid"
	"medication-service-v2/internal/domain/entities"
	"medication-service-v2/internal/application/services"
)

// ValidClinicalContext returns a valid clinical context for testing
func ValidClinicalContext() *entities.ClinicalContext {
	weight := 70.0
	height := 175.0
	bsa := 1.8
	creatinine := 1.0
	egfr := 90.0

	return &entities.ClinicalContext{
		PatientID:      uuid.New(),
		WeightKg:       &weight,
		HeightCm:       &height,
		AgeYears:       45,
		Gender:         "female",
		BSAm2:          &bsa,
		CreatinineMgdL: &creatinine,
		eGFR:           &egfr,
		Allergies:      []string{"penicillin"},
		Conditions:     []string{"hypertension"},
		Medications: []entities.CurrentMedication{
			{
				MedicationName: "lisinopril",
				DoseMg:         10.0,
				Frequency:      "daily",
				StartDate:      time.Now().Add(-30 * 24 * time.Hour),
				Route:          "oral",
			},
		},
		LabValues: map[string]entities.LabValue{
			"creatinine": {
				Value:     1.0,
				Unit:      "mg/dL",
				Timestamp: time.Now().Add(-2 * time.Hour),
				Reference: "0.6-1.2",
			},
			"bun": {
				Value:     15.0,
				Unit:      "mg/dL",
				Timestamp: time.Now().Add(-2 * time.Hour),
				Reference: "7-25",
			},
		},
	}
}

// ValidProposal returns a valid medication proposal for testing
func ValidProposal() *entities.MedicationProposal {
	return &entities.MedicationProposal{
		ID:              uuid.New(),
		PatientID:       uuid.New(),
		ProtocolID:      "chemotherapy-protocol-1",
		Indication:      "Acute lymphoblastic leukemia",
		Status:          entities.ProposalStatusProposed,
		ClinicalContext: ValidClinicalContext(),
		MedicationDetails: ValidMedicationDetails(),
		DosageRecommendations: []entities.DosageRecommendation{
			{
				ID:                 uuid.New(),
				RecommendationType: entities.RecommendationStarting,
				DoseMg:            1.4,
				FrequencyPerDay:   1,
				Route:             "intravenous",
				DurationDays:      &[]int{1}[0],
				CalculationMethod: entities.MethodBSABased,
				ConfidenceScore:   0.95,
				ClinicalNotes:     "BSA-based dosing for vincristine",
				MonitoringRequired: []entities.MonitoringRequirement{
					{
						Parameter:   "neurological_assessment",
						Frequency:   entities.FrequencyWeekly,
						Notes:       "Monitor for peripheral neuropathy",
					},
				},
			},
		},
		SafetyConstraints: []entities.SafetyConstraint{
			{
				ID:             uuid.New(),
				ConstraintType: entities.ConstraintDosage,
				Severity:       entities.SeverityWarning,
				Parameter:      "max_single_dose",
				Operator:       "<=",
				ThresholdValue: 2.0,
				Unit:           "mg",
				Message:        "Maximum single dose should not exceed 2mg",
				Action:         "warn",
				Source:         "clinical_guidelines",
			},
		},
		SnapshotID: uuid.New(),
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		CreatedBy:  "dr-smith",
	}
}

// ValidMedicationDetails returns valid medication details for testing
func ValidMedicationDetails() *entities.MedicationDetails {
	return &entities.MedicationDetails{
		DrugName:    "Vincristine",
		GenericName: "vincristine sulfate",
		BrandName:   "Oncovin",
		DrugClass:   "Vinca alkaloid",
		Mechanism:   "Mitotic inhibitor - disrupts microtubule formation",
		Indication:  "Acute lymphoblastic leukemia",
		Contraindications: []string{
			"Demyelinating form of Charcot-Marie-Tooth syndrome",
			"Intrathecal administration",
		},
		Interactions: []entities.DrugInteraction{
			{
				InteractingDrug: "phenytoin",
				Severity:        entities.SeverityModerate,
				Description:     "May decrease phenytoin levels",
				Management:      "Monitor phenytoin levels",
			},
		},
		FormulationTypes: []entities.FormulationType{
			{
				Form:         "injection",
				Strengths:    []string{"1mg/ml", "2mg/ml"},
				Route:        "intravenous",
				Availability: "generic",
			},
		},
		TherapeuticClass: "Antineoplastic agent",
		PharmacologyProfile: &entities.PharmacologyProfile{
			HalfLifeHours:     &[]float64{24.0}[0],
			OnsetMinutes:      &[]int{30}[0],
			PeakHours:         &[]float64{1.0}[0],
			DurationHours:     &[]float64{168.0}[0], // 1 week
			Bioavailability:   &[]float64{1.0}[0],   // 100% for IV
			ProteinBinding:    &[]float64{75.0}[0],
			Metabolism:        "Hepatic via CYP3A4",
			Excretion:         "Biliary (80%), renal (10-20%)",
			RenalAdjustment:   false,
			HepaticAdjustment: true,
		},
	}
}

// ValidRecipe returns a valid recipe for testing
func ValidRecipe() *entities.Recipe {
	return &entities.Recipe{
		ID:          uuid.New(),
		ProtocolID:  "chemotherapy-protocol-1",
		Name:        "Vincristine Standard Dosing",
		Version:     "1.0",
		Description: "Standard vincristine dosing for pediatric and adult ALL",
		Indication:  "Acute lymphoblastic leukemia",
		ContextRequirements: entities.ContextRequirements{
			CalculationFields: []entities.ContextField{
				{Name: "age", Type: entities.FieldTypeNumber, Required: true, Unit: "years"},
				{Name: "weight", Type: entities.FieldTypeNumber, Required: true, Unit: "kg"},
				{Name: "height", Type: entities.FieldTypeNumber, Required: true, Unit: "cm"},
				{Name: "bsa", Type: entities.FieldTypeNumber, Required: true, Unit: "m2"},
			},
			SafetyFields: []entities.ContextField{
				{Name: "allergies", Type: entities.FieldTypeArray, Required: false},
				{Name: "current_medications", Type: entities.FieldTypeArray, Required: false},
			},
			FreshnessRequirements: map[string]time.Duration{
				"weight": 7 * 24 * time.Hour,  // 1 week
				"height": 30 * 24 * time.Hour, // 1 month
				"lab_values": 24 * time.Hour,  // 1 day
			},
		},
		CalculationRules: []entities.CalculationRule{
			{
				ID:              uuid.New(),
				Name:            "BSA-based vincristine dosing",
				Priority:        1,
				CalculationType: entities.MethodBSABased,
				Formula:         "1.4 * bsa_m2",
				Parameters: map[string]interface{}{
					"dose_per_m2": 1.4,
					"max_dose":    2.0,
				},
				OutputUnit: "mg",
				RoundingRule: entities.RoundingRule{
					Type:      entities.RoundingStandard,
					Precision: 1,
					Direction: entities.RoundingNearest,
				},
			},
		},
		SafetyRules: []entities.SafetyRule{
			{
				ID:       uuid.New(),
				Name:     "Maximum single dose limit",
				Priority: 1,
				Type:     entities.SafetyRuleDoseLimit,
				Condition: entities.RuleCondition{
					Field:    "calculated_dose",
					Operator: ">",
					Value:    2.0,
				},
				Action:   entities.ActionBlock,
				Severity: entities.SeverityCritical,
				Message:  "Calculated dose exceeds maximum safe limit of 2mg",
				Mitigation: "Cap dose at 2mg maximum",
			},
		},
		MonitoringRules: []entities.MonitoringRule{
			{
				ID:        uuid.New(),
				Name:      "Neurological monitoring",
				Parameter: "neurological_assessment",
				Frequency: entities.FrequencyWeekly,
				AlertRules: []entities.AlertRule{
					{
						ID: uuid.New(),
						Condition: entities.RuleCondition{
							Field:    "peripheral_neuropathy_grade",
							Operator: ">=",
							Value:    2,
						},
						AlertLevel: entities.AlertLevelWarning,
						Message:    "Grade 2+ peripheral neuropathy detected",
						Action:     "Consider dose reduction",
					},
				},
				Instructions: "Assess for signs of peripheral neuropathy before each dose",
			},
		},
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now(),
		CreatedBy: "clinical-team",
		Status:    entities.RecipeStatusActive,
		TTL:       30 * 24 * time.Hour, // 30 days
		ClinicalEvidence: &entities.ClinicalEvidence{
			Guidelines: []entities.GuidelineReference{
				{
					Organization: "COG",
					Title:        "Childhood ALL Treatment Guidelines",
					Version:      "2023",
					Relevance:    "Primary dosing reference",
				},
			},
			EvidenceLevel: entities.EvidenceLevelA,
			LastUpdated:   time.Now().Add(-7 * 24 * time.Hour),
		},
	}
}

// ValidRecipeWithRules returns a recipe with comprehensive rules for testing
func ValidRecipeWithRules() *entities.Recipe {
	recipe := ValidRecipe()
	
	// Add more comprehensive rules for testing
	recipe.CalculationRules = append(recipe.CalculationRules, entities.CalculationRule{
		ID:              uuid.New(),
		Name:            "Pediatric weight-based adjustment",
		Priority:        2,
		Condition: &entities.RuleCondition{
			Field:    "age",
			Operator: "<",
			Value:    18,
		},
		CalculationType: entities.MethodWeightBased,
		Formula:         "0.05 * weight_kg",
		Parameters: map[string]interface{}{
			"dose_per_kg": 0.05,
			"max_dose":    2.0,
		},
		OutputUnit: "mg",
		RoundingRule: entities.RoundingRule{
			Type:      entities.RoundingPractical,
			Precision: 1,
			Direction: entities.RoundingNearest,
		},
		Adjustments: []entities.DoseAdjustment{
			{
				ID:   uuid.New(),
				Name: "Renal impairment adjustment",
				Condition: entities.RuleCondition{
					Field:    "egfr",
					Operator: "<",
					Value:    60.0,
				},
				Type:     entities.AdjustmentMultiplier,
				Value:    0.75,
				Reason:   "Reduce dose for moderate renal impairment",
				Evidence: "Clinical pharmacology studies",
			},
		},
	})

	return recipe
}

// ValidSnapshot returns a valid snapshot for testing
func ValidSnapshot() *entities.Snapshot {
	return &entities.Snapshot{
		ID:        uuid.New(),
		PatientID: uuid.New(),
		RecipeID:  uuid.New(),
		Type:      entities.SnapshotTypeCalculation,
		ClinicalContext: ValidClinicalContext(),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(1 * time.Hour),
		Status:    entities.SnapshotStatusActive,
		Metadata: map[string]interface{}{
			"source": "medication_service",
			"version": "1.0",
		},
	}
}

// ValidPatientContext returns a valid patient context for testing
func ValidPatientContext() entities.PatientContext {
	return entities.PatientContext{
		PatientID:       uuid.New().String(),
		Age:             45,
		Weight:          70.0,
		Height:          175.0,
		Gender:          "female",
		PregnancyStatus: false,
		RenalFunction: &entities.RenalFunction{
			eGFR:       90.0,
			Creatinine: 1.0,
			BUN:        15.0,
		},
		HepaticFunction: &entities.HepaticFunction{
			ChildPugh: "A",
			ALT:       25.0,
			AST:       20.0,
			Bilirubin: 0.8,
		},
		LabResults: map[string]entities.LabValue{
			"creatinine": {
				Value:     1.0,
				Unit:      "mg/dL",
				Timestamp: time.Now().Add(-2 * time.Hour),
				Reference: "0.6-1.2",
			},
		},
		EncounterContext: entities.EncounterContext{
			EncounterID: uuid.New().String(),
			ProviderID:  "dr-smith-id",
			FacilityID:  "hospital-main",
			Department:  "oncology",
		},
	}
}

// ValidDosageCalculations returns valid dosage calculations for testing
func ValidDosageCalculations() *services.CalculateDosagesResponse {
	return &services.CalculateDosagesResponse{
		DosageRecommendations: []entities.DosageRecommendation{
			{
				ID:                 uuid.New(),
				RecommendationType: entities.RecommendationStarting,
				DoseMg:            2.52, // 1.4 * 1.8 BSA
				FrequencyPerDay:   1,
				Route:             "intravenous",
				DurationDays:      &[]int{1}[0],
				MaxDoseMg:         &[]float64{2.0}[0], // Capped at max
				CalculationMethod: entities.MethodBSABased,
				ConfidenceScore:   0.95,
				ClinicalNotes:     "Dose calculated based on BSA, capped at 2mg maximum",
			},
		},
		SafetyConstraints: []entities.SafetyConstraint{
			{
				ID:             uuid.New(),
				ConstraintType: entities.ConstraintDosage,
				Severity:       entities.SeverityWarning,
				Parameter:      "max_single_dose",
				Operator:       "<=",
				ThresholdValue: 2.0,
				Unit:           "mg",
				Message:        "Dose capped at maximum safe limit",
				Action:         "adjust",
				Source:         "safety_rules",
			},
		},
		Warnings: []string{},
		ClinicalRecommendations: []string{
			"Monitor for peripheral neuropathy",
			"Ensure proper IV administration technique",
		},
		SafetyAlerts: []services.SafetyAlert{},
	}
}

// ValidDosageCalculationsWithAlerts returns dosage calculations with safety alerts
func ValidDosageCalculationsWithAlerts() *services.CalculateDosagesResponse {
	calc := ValidDosageCalculations()
	calc.SafetyAlerts = []services.SafetyAlert{
		{
			Level:   "critical",
			Message: "Dose exceeds maximum safe limit",
			Code:    "DOSE_LIMIT_EXCEEDED",
			Details: map[string]interface{}{
				"calculated_dose": 2.52,
				"max_safe_dose":   2.0,
			},
		},
	}
	return calc
}

// PediatricPatientContext returns a pediatric patient context for testing
func PediatricPatientContext() entities.PatientContext {
	context := ValidPatientContext()
	context.Age = 8
	context.Weight = 25.0
	context.Height = 120.0
	return context
}

// PregnantPatientContext returns a pregnant patient context for testing
func PregnantPatientContext() entities.PatientContext {
	context := ValidPatientContext()
	context.Gender = "female"
	context.PregnancyStatus = true
	context.Age = 28
	return context
}

// RenalImpairedPatientContext returns a patient with renal impairment for testing
func RenalImpairedPatientContext() entities.PatientContext {
	context := ValidPatientContext()
	context.RenalFunction.eGFR = 45.0
	context.RenalFunction.Creatinine = 2.1
	context.LabResults["creatinine"] = entities.LabValue{
		Value:     2.1,
		Unit:      "mg/dL",
		Timestamp: time.Now().Add(-1 * time.Hour),
		Reference: "0.6-1.2",
	}
	return context
}

// InvalidClinicalContext returns an invalid clinical context for testing error cases
func InvalidClinicalContext() *entities.ClinicalContext {
	return &entities.ClinicalContext{
		PatientID: uuid.Nil, // Invalid UUID
		AgeYears:  -5,       // Invalid age
		Gender:    "",       // Missing gender
	}
}

// ExpiredRecipe returns an expired recipe for testing
func ExpiredRecipe() *entities.Recipe {
	recipe := ValidRecipe()
	recipe.UpdatedAt = time.Now().Add(-48 * time.Hour) // 2 days ago
	recipe.TTL = 24 * time.Hour                        // 1 day TTL - expired
	return recipe
}

// InactiveRecipe returns an inactive recipe for testing
func InactiveRecipe() *entities.Recipe {
	recipe := ValidRecipe()
	recipe.Status = entities.RecipeStatusDeprecated
	return recipe
}