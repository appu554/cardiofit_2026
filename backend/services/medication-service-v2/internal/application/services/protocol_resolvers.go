package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"medication-service-v2/internal/domain/entities"
)

// HypertensionProtocolResolver implements protocol-specific resolution for hypertension management
type HypertensionProtocolResolver struct {
	protocolID string
	version    string
}

// NewHypertensionProtocolResolver creates a new hypertension protocol resolver
func NewHypertensionProtocolResolver() *HypertensionProtocolResolver {
	return &HypertensionProtocolResolver{
		protocolID: "hypertension-standard",
		version:    "1.0.0",
	}
}

// ResolveFields resolves fields specific to hypertension management protocols
func (h *HypertensionProtocolResolver) ResolveFields(ctx context.Context, patientContext entities.PatientContext, recipe *entities.Recipe) (*entities.RecipeResolution, error) {
	resolution := &entities.RecipeResolution{
		RecipeID:        recipe.ID,
		ResolutionTime:  time.Now(),
		ContextSnapshot: make(map[string]interface{}),
	}

	// Age-based considerations for hypertension
	if patientContext.Age >= 65 {
		resolution.ContextSnapshot["elderly_considerations"] = true
		resolution.ContextSnapshot["target_bp_systolic"] = 150 // Less aggressive target for elderly
		resolution.ContextSnapshot["target_bp_diastolic"] = 90
	} else {
		resolution.ContextSnapshot["target_bp_systolic"] = 130
		resolution.ContextSnapshot["target_bp_diastolic"] = 80
	}

	// CKD-specific considerations
	if patientContext.RenalFunction != nil {
		eGFR := patientContext.RenalFunction.eGFR
		if eGFR < 60 {
			resolution.ContextSnapshot["ckd_present"] = true
			resolution.ContextSnapshot["ace_inhibitor_preferred"] = true
			resolution.ContextSnapshot["target_bp_systolic"] = 130 // CKD target
			
			if eGFR < 30 {
				resolution.ContextSnapshot["ckd_stage"] = "4-5"
				resolution.ContextSnapshot["potassium_monitoring_required"] = true
				resolution.ContextSnapshot["creatinine_monitoring_frequency"] = "weekly"
			} else {
				resolution.ContextSnapshot["ckd_stage"] = "3"
				resolution.ContextSnapshot["creatinine_monitoring_frequency"] = "monthly"
			}
		}
	}

	// Diabetes considerations
	for _, condition := range patientContext.Conditions {
		if condition.Code == "E10" || condition.Code == "E11" { // Type 1 or Type 2 diabetes
			resolution.ContextSnapshot["diabetes_present"] = true
			resolution.ContextSnapshot["ace_inhibitor_preferred"] = true
			resolution.ContextSnapshot["target_bp_systolic"] = 130 // Diabetes target
			break
		}
	}

	// Pregnancy considerations
	if patientContext.PregnancyStatus {
		resolution.ContextSnapshot["pregnancy_safe_agents"] = []string{"methyldopa", "labetalol", "nifedipine_xl"}
		resolution.ContextSnapshot["contraindicated_agents"] = []string{"ace_inhibitors", "arbs", "atenolol"}
		resolution.ContextSnapshot["target_bp_systolic"] = 140 // Pregnancy targets
		resolution.ContextSnapshot["target_bp_diastolic"] = 90
	}

	// Drug interaction screening
	for _, medication := range patientContext.CurrentMedications {
		if medication.Display == "warfarin" {
			resolution.ContextSnapshot["warfarin_interaction_risk"] = true
		}
		if medication.Display == "digoxin" {
			resolution.ContextSnapshot["digoxin_interaction_risk"] = true
		}
	}

	return resolution, nil
}

// GetRequiredFields returns the fields required for hypertension protocol
func (h *HypertensionProtocolResolver) GetRequiredFields(ctx context.Context, protocolID string) ([]entities.FieldRequirement, error) {
	return []entities.FieldRequirement{
		{
			Name:         "systolic_bp",
			Type:         entities.FieldTypeNumber,
			Required:     true,
			Source:       "vital_signs",
			FreshnessReq: 24 * time.Hour,
			Priority:     1,
		},
		{
			Name:         "diastolic_bp",
			Type:         entities.FieldTypeNumber,
			Required:     true,
			Source:       "vital_signs",
			FreshnessReq: 24 * time.Hour,
			Priority:     1,
		},
		{
			Name:         "serum_creatinine",
			Type:         entities.FieldTypeNumber,
			Required:     false,
			Source:       "lab_results",
			FreshnessReq: 7 * 24 * time.Hour,
			Priority:     2,
		},
		{
			Name:         "serum_potassium",
			Type:         entities.FieldTypeNumber,
			Required:     false,
			Source:       "lab_results",
			FreshnessReq: 7 * 24 * time.Hour,
			Priority:     2,
		},
	}, nil
}

// ValidateConditions validates hypertension-specific conditions
func (h *HypertensionProtocolResolver) ValidateConditions(ctx context.Context, conditions []entities.RuleCondition, patientContext entities.PatientContext) (bool, error) {
	for _, condition := range conditions {
		result, err := h.evaluateHypertensionCondition(condition, patientContext)
		if err != nil {
			return false, err
		}
		if !result {
			return false, nil
		}
	}
	return true, nil
}

// GetCacheKey generates cache key for hypertension protocol
func (h *HypertensionProtocolResolver) GetCacheKey(patientContext entities.PatientContext, recipe *entities.Recipe) string {
	return fmt.Sprintf("hypertension:%s:%s:%d", recipe.ID.String(), patientContext.PatientID, patientContext.Age)
}

func (h *HypertensionProtocolResolver) evaluateHypertensionCondition(condition entities.RuleCondition, patientContext entities.PatientContext) (bool, error) {
	switch condition.Field {
	case "bp_stage":
		// Evaluate blood pressure stage from lab results
		if systolicBP, exists := patientContext.LabResults["systolic_bp"]; exists {
			stage := h.calculateBPStage(systolicBP.Value, patientContext.LabResults["diastolic_bp"].Value)
			return h.evaluateStringCondition(stage, condition.Value.(string), condition.Operator)
		}
		return false, nil
	case "ckd_stage":
		if patientContext.RenalFunction != nil {
			stage := h.calculateCKDStage(patientContext.RenalFunction.eGFR)
			return h.evaluateStringCondition(stage, condition.Value.(string), condition.Operator)
		}
		return false, nil
	default:
		// Delegate to standard condition evaluation
		return evaluateRuleCondition(condition, patientContext)
	}
}

func (h *HypertensionProtocolResolver) calculateBPStage(systolic, diastolic float64) string {
	if systolic < 120 && diastolic < 80 {
		return "normal"
	} else if systolic <= 129 && diastolic < 80 {
		return "elevated"
	} else if (systolic >= 130 && systolic <= 139) || (diastolic >= 80 && diastolic <= 89) {
		return "stage1"
	} else if systolic >= 140 || diastolic >= 90 {
		return "stage2"
	} else if systolic > 180 || diastolic > 120 {
		return "crisis"
	}
	return "unknown"
}

func (h *HypertensionProtocolResolver) calculateCKDStage(eGFR float64) string {
	if eGFR >= 90 {
		return "1-2"
	} else if eGFR >= 60 {
		return "3a"
	} else if eGFR >= 45 {
		return "3b"
	} else if eGFR >= 30 {
		return "4"
	} else if eGFR >= 15 {
		return "5"
	} else {
		return "5-dialysis"
	}
}

func (h *HypertensionProtocolResolver) evaluateStringCondition(value, expected, operator string) (bool, error) {
	switch operator {
	case "==":
		return value == expected, nil
	case "!=":
		return value != expected, nil
	default:
		return false, fmt.Errorf("unsupported string operator: %s", operator)
	}
}

// DiabetesProtocolResolver implements protocol-specific resolution for diabetes management
type DiabetesProtocolResolver struct {
	protocolID string
	version    string
}

// NewDiabetesProtocolResolver creates a new diabetes protocol resolver
func NewDiabetesProtocolResolver() *DiabetesProtocolResolver {
	return &DiabetesProtocolResolver{
		protocolID: "diabetes-management",
		version:    "1.0.0",
	}
}

// ResolveFields resolves fields specific to diabetes management protocols
func (d *DiabetesProtocolResolver) ResolveFields(ctx context.Context, patientContext entities.PatientContext, recipe *entities.Recipe) (*entities.RecipeResolution, error) {
	resolution := &entities.RecipeResolution{
		RecipeID:        recipe.ID,
		ResolutionTime:  time.Now(),
		ContextSnapshot: make(map[string]interface{}),
	}

	// HbA1c-based treatment intensity
	if hba1c, exists := patientContext.LabResults["hba1c"]; exists {
		resolution.ContextSnapshot["current_hba1c"] = hba1c.Value
		
		if hba1c.Value > 9.0 {
			resolution.ContextSnapshot["treatment_intensity"] = "aggressive"
			resolution.ContextSnapshot["insulin_required"] = true
		} else if hba1c.Value > 8.0 {
			resolution.ContextSnapshot["treatment_intensity"] = "intensive"
			resolution.ContextSnapshot["combination_therapy"] = true
		} else if hba1c.Value > 7.0 {
			resolution.ContextSnapshot["treatment_intensity"] = "moderate"
		} else {
			resolution.ContextSnapshot["treatment_intensity"] = "maintenance"
		}
	}

	// Age-based considerations
	if patientContext.Age >= 75 {
		resolution.ContextSnapshot["elderly_diabetes"] = true
		resolution.ContextSnapshot["hba1c_target"] = 8.0 // Less stringent for elderly
		resolution.ContextSnapshot["hypoglycemia_risk"] = "high"
	} else if patientContext.Age >= 65 {
		resolution.ContextSnapshot["hba1c_target"] = 7.5
		resolution.ContextSnapshot["hypoglycemia_risk"] = "moderate"
	} else {
		resolution.ContextSnapshot["hba1c_target"] = 7.0
		resolution.ContextSnapshot["hypoglycemia_risk"] = "low"
	}

	// CKD considerations for diabetes
	if patientContext.RenalFunction != nil {
		eGFR := patientContext.RenalFunction.eGFR
		if eGFR < 60 {
			resolution.ContextSnapshot["diabetic_nephropathy"] = true
			resolution.ContextSnapshot["metformin_contraindicated"] = eGFR < 30
			resolution.ContextSnapshot["sglt2_inhibitor_preferred"] = eGFR >= 25
		}
	}

	// Heart failure considerations
	for _, condition := range patientContext.Conditions {
		if condition.Code == "I50" { // Heart failure
			resolution.ContextSnapshot["heart_failure_present"] = true
			resolution.ContextSnapshot["sglt2_inhibitor_preferred"] = true
			resolution.ContextSnapshot["glp1_agonist_preferred"] = true
			break
		}
	}

	// Pregnancy considerations
	if patientContext.PregnancyStatus {
		resolution.ContextSnapshot["gestational_diabetes"] = true
		resolution.ContextSnapshot["insulin_only"] = true
		resolution.ContextSnapshot["hba1c_target"] = 6.0
		resolution.ContextSnapshot["contraindicated_agents"] = []string{"metformin", "sglt2_inhibitors", "glp1_agonists"}
	}

	return resolution, nil
}

// GetRequiredFields returns the fields required for diabetes protocol
func (d *DiabetesProtocolResolver) GetRequiredFields(ctx context.Context, protocolID string) ([]entities.FieldRequirement, error) {
	return []entities.FieldRequirement{
		{
			Name:         "hba1c",
			Type:         entities.FieldTypeNumber,
			Required:     true,
			Source:       "lab_results",
			FreshnessReq: 90 * 24 * time.Hour,
			Priority:     1,
		},
		{
			Name:         "glucose_fasting",
			Type:         entities.FieldTypeNumber,
			Required:     false,
			Source:       "lab_results",
			FreshnessReq: 24 * time.Hour,
			Priority:     2,
		},
		{
			Name:         "microalbumin",
			Type:         entities.FieldTypeNumber,
			Required:     false,
			Source:       "lab_results",
			FreshnessReq: 365 * 24 * time.Hour,
			Priority:     3,
		},
	}, nil
}

// ValidateConditions validates diabetes-specific conditions
func (d *DiabetesProtocolResolver) ValidateConditions(ctx context.Context, conditions []entities.RuleCondition, patientContext entities.PatientContext) (bool, error) {
	for _, condition := range conditions {
		result, err := evaluateRuleCondition(condition, patientContext)
		if err != nil {
			return false, err
		}
		if !result {
			return false, nil
		}
	}
	return true, nil
}

// GetCacheKey generates cache key for diabetes protocol
func (d *DiabetesProtocolResolver) GetCacheKey(patientContext entities.PatientContext, recipe *entities.Recipe) string {
	hba1c := "unknown"
	if hba1cLab, exists := patientContext.LabResults["hba1c"]; exists {
		hba1c = fmt.Sprintf("%.1f", hba1cLab.Value)
	}
	return fmt.Sprintf("diabetes:%s:%s:%s", recipe.ID.String(), patientContext.PatientID, hba1c)
}

// PediatricProtocolResolver implements protocol-specific resolution for pediatric patients
type PediatricProtocolResolver struct {
	protocolID string
	version    string
}

// NewPediatricProtocolResolver creates a new pediatric protocol resolver
func NewPediatricProtocolResolver() *PediatricProtocolResolver {
	return &PediatricProtocolResolver{
		protocolID: "pediatric-standard",
		version:    "1.0.0",
	}
}

// ResolveFields resolves fields specific to pediatric protocols
func (p *PediatricProtocolResolver) ResolveFields(ctx context.Context, patientContext entities.PatientContext, recipe *entities.Recipe) (*entities.RecipeResolution, error) {
	resolution := &entities.RecipeResolution{
		RecipeID:        recipe.ID,
		ResolutionTime:  time.Now(),
		ContextSnapshot: make(map[string]interface{}),
	}

	// Age-based dosing categories
	age := patientContext.Age
	if age < 1 {
		resolution.ContextSnapshot["age_category"] = "neonate"
		resolution.ContextSnapshot["dosing_basis"] = "weight_only"
		resolution.ContextSnapshot["max_dose_adjustment"] = 0.5
	} else if age < 2 {
		resolution.ContextSnapshot["age_category"] = "infant"
		resolution.ContextSnapshot["dosing_basis"] = "weight_primary"
		resolution.ContextSnapshot["max_dose_adjustment"] = 0.7
	} else if age < 12 {
		resolution.ContextSnapshot["age_category"] = "child"
		resolution.ContextSnapshot["dosing_basis"] = "weight_and_age"
		resolution.ContextSnapshot["max_dose_adjustment"] = 0.8
	} else {
		resolution.ContextSnapshot["age_category"] = "adolescent"
		resolution.ContextSnapshot["dosing_basis"] = "weight_or_adult"
		resolution.ContextSnapshot["max_dose_adjustment"] = 0.9
	}

	// Weight-based dosing parameters
	weight := patientContext.Weight
	if weight > 0 {
		resolution.ContextSnapshot["weight_kg"] = weight
		if weight < 10 {
			resolution.ContextSnapshot["dosing_precision"] = "0.1mg"
		} else if weight < 40 {
			resolution.ContextSnapshot["dosing_precision"] = "1mg"
		} else {
			resolution.ContextSnapshot["dosing_precision"] = "5mg"
		}
	}

	// Organ maturity considerations
	if age < 2 {
		resolution.ContextSnapshot["hepatic_maturity"] = "immature"
		resolution.ContextSnapshot["renal_maturity"] = "immature"
		resolution.ContextSnapshot["dose_reduction_required"] = true
	} else if age < 12 {
		resolution.ContextSnapshot["hepatic_maturity"] = "developing"
		resolution.ContextSnapshot["renal_maturity"] = "developing"
	} else {
		resolution.ContextSnapshot["hepatic_maturity"] = "mature"
		resolution.ContextSnapshot["renal_maturity"] = "mature"
	}

	// Formulation considerations
	resolution.ContextSnapshot["preferred_formulations"] = []string{"liquid", "chewable", "orally_disintegrating"}
	if age < 5 {
		resolution.ContextSnapshot["avoid_capsules"] = true
		resolution.ContextSnapshot["avoid_large_tablets"] = true
	}

	return resolution, nil
}

// GetRequiredFields returns the fields required for pediatric protocol
func (p *PediatricProtocolResolver) GetRequiredFields(ctx context.Context, protocolID string) ([]entities.FieldRequirement, error) {
	return []entities.FieldRequirement{
		{
			Name:         "weight",
			Type:         entities.FieldTypeNumber,
			Required:     true,
			Source:       "vital_signs",
			FreshnessReq: 7 * 24 * time.Hour,
			Priority:     1,
		},
		{
			Name:         "height",
			Type:         entities.FieldTypeNumber,
			Required:     true,
			Source:       "vital_signs",
			FreshnessReq: 30 * 24 * time.Hour,
			Priority:     1,
		},
		{
			Name:         "bsa",
			Type:         entities.FieldTypeNumber,
			Required:     false,
			Source:       "calculated",
			FreshnessReq: 30 * 24 * time.Hour,
			Priority:     2,
		},
	}, nil
}

// ValidateConditions validates pediatric-specific conditions
func (p *PediatricProtocolResolver) ValidateConditions(ctx context.Context, conditions []entities.RuleCondition, patientContext entities.PatientContext) (bool, error) {
	for _, condition := range conditions {
		result, err := evaluateRuleCondition(condition, patientContext)
		if err != nil {
			return false, err
		}
		if !result {
			return false, nil
		}
	}
	return true, nil
}

// GetCacheKey generates cache key for pediatric protocol
func (p *PediatricProtocolResolver) GetCacheKey(patientContext entities.PatientContext, recipe *entities.Recipe) string {
	return fmt.Sprintf("pediatric:%s:%s:%d:%.1f", recipe.ID.String(), patientContext.PatientID, patientContext.Age, patientContext.Weight)
}

// ProtocolResolverRegistry manages protocol-specific resolvers
type ProtocolResolverRegistry struct {
	resolvers map[string]entities.ProtocolResolver
}

// NewProtocolResolverRegistry creates a new protocol resolver registry
func NewProtocolResolverRegistry() *ProtocolResolverRegistry {
	registry := &ProtocolResolverRegistry{
		resolvers: make(map[string]entities.ProtocolResolver),
	}

	// Register built-in protocol resolvers
	registry.Register("hypertension-standard", NewHypertensionProtocolResolver())
	registry.Register("diabetes-management", NewDiabetesProtocolResolver())
	registry.Register("pediatric-standard", NewPediatricProtocolResolver())

	return registry
}

// Register registers a protocol resolver
func (r *ProtocolResolverRegistry) Register(protocolID string, resolver entities.ProtocolResolver) {
	r.resolvers[protocolID] = resolver
}

// Get retrieves a protocol resolver
func (r *ProtocolResolverRegistry) Get(protocolID string) (entities.ProtocolResolver, error) {
	resolver, exists := r.resolvers[protocolID]
	if !exists {
		return nil, fmt.Errorf("protocol resolver not found for protocol: %s", protocolID)
	}
	return resolver, nil
}

// List returns all available protocol IDs
func (r *ProtocolResolverRegistry) List() []string {
	protocols := make([]string, 0, len(r.resolvers))
	for protocolID := range r.resolvers {
		protocols = append(protocols, protocolID)
	}
	return protocols
}