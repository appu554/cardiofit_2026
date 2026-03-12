package services

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// MockCAEEngine implements a mock Clinical Assertion Engine
type MockCAEEngine struct {
	id     string
	name   string
	logger *logger.Logger
}

// NewMockCAEEngine creates a new mock CAE engine
func NewMockCAEEngine(logger *logger.Logger) *MockCAEEngine {
	return &MockCAEEngine{
		id:     "cae_engine",
		name:   "Clinical Assertion Engine",
		logger: logger,
	}
}

func (m *MockCAEEngine) ID() string                { return m.id }
func (m *MockCAEEngine) Name() string              { return m.name }
func (m *MockCAEEngine) Capabilities() []string   { return []string{"drug_interaction", "contraindication", "dosing"} }
func (m *MockCAEEngine) HealthCheck() error       { return nil }
func (m *MockCAEEngine) Initialize(config types.EngineConfig) error { return nil }
func (m *MockCAEEngine) Shutdown() error          { return nil }

func (m *MockCAEEngine) Evaluate(ctx context.Context, req *types.SafetyRequest, clinicalContext *types.ClinicalContext) (*types.EngineResult, error) {
	// Simulate processing time
	time.Sleep(time.Duration(rand.Intn(50)) * time.Millisecond)

	result := &types.EngineResult{
		EngineID:   m.id,
		EngineName: m.name,
		Confidence: 0.85 + rand.Float64()*0.15, // 85-100% confidence
	}

	// Check for drug interactions based on medication count
	if len(req.MedicationIDs) > 3 {
		result.Status = types.SafetyStatusWarning
		result.RiskScore = 0.6 + rand.Float64()*0.3 // 60-90% risk
		result.Warnings = []string{"Multiple medications detected - monitor for interactions"}
	} else if len(req.MedicationIDs) > 5 {
		result.Status = types.SafetyStatusUnsafe
		result.RiskScore = 0.8 + rand.Float64()*0.2 // 80-100% risk
		result.Violations = []string{"High polypharmacy risk - review medication regimen"}
	} else {
		result.Status = types.SafetyStatusSafe
		result.RiskScore = rand.Float64() * 0.3 // 0-30% risk
	}

	// Simulate specific drug interaction checks
	for _, medID := range req.MedicationIDs {
		if strings.Contains(medID, "warfarin") && len(req.MedicationIDs) > 1 {
			result.Status = types.SafetyStatusUnsafe
			result.RiskScore = 0.9
			result.Violations = append(result.Violations, "Warfarin interaction risk detected")
		}
	}

	return result, nil
}

// MockAllergyEngine implements a mock allergy checking engine
type MockAllergyEngine struct {
	id     string
	name   string
	logger *logger.Logger
}

func NewMockAllergyEngine(logger *logger.Logger) *MockAllergyEngine {
	return &MockAllergyEngine{
		id:     "allergy_engine",
		name:   "Allergy Check Engine",
		logger: logger,
	}
}

func (m *MockAllergyEngine) ID() string                { return m.id }
func (m *MockAllergyEngine) Name() string              { return m.name }
func (m *MockAllergyEngine) Capabilities() []string   { return []string{"allergy_check", "contraindication"} }
func (m *MockAllergyEngine) HealthCheck() error       { return nil }
func (m *MockAllergyEngine) Initialize(config types.EngineConfig) error { return nil }
func (m *MockAllergyEngine) Shutdown() error          { return nil }

func (m *MockAllergyEngine) Evaluate(ctx context.Context, req *types.SafetyRequest, clinicalContext *types.ClinicalContext) (*types.EngineResult, error) {
	time.Sleep(time.Duration(rand.Intn(30)) * time.Millisecond)

	result := &types.EngineResult{
		EngineID:   m.id,
		EngineName: m.name,
		Status:     types.SafetyStatusSafe,
		RiskScore:  rand.Float64() * 0.2, // 0-20% risk for safe cases
		Confidence: 0.9 + rand.Float64()*0.1, // 90-100% confidence
	}

	// Check for allergy conflicts
	if len(req.AllergyIDs) > 0 && len(req.MedicationIDs) > 0 {
		// Simulate allergy checking
		for _, allergyID := range req.AllergyIDs {
			if strings.Contains(allergyID, "penicillin") {
				for _, medID := range req.MedicationIDs {
					if strings.Contains(medID, "amoxicillin") || strings.Contains(medID, "penicillin") {
						result.Status = types.SafetyStatusUnsafe
						result.RiskScore = 0.95
						result.Violations = []string{"Penicillin allergy contraindication detected"}
						return result, nil
					}
				}
			}
		}
	}

	// Check clinical context allergies
	if clinicalContext != nil {
		for _, allergy := range clinicalContext.Allergies {
			if allergy.Severity == "severe" {
				result.Status = types.SafetyStatusWarning
				result.RiskScore = 0.4
				result.Warnings = []string{fmt.Sprintf("Severe allergy to %s on record", allergy.Allergen)}
			}
		}
	}

	return result, nil
}

// MockProtocolEngine implements a mock clinical protocol engine
type MockProtocolEngine struct {
	id     string
	name   string
	logger *logger.Logger
}

func NewMockProtocolEngine(logger *logger.Logger) *MockProtocolEngine {
	return &MockProtocolEngine{
		id:     "protocol_engine",
		name:   "Clinical Protocol Engine",
		logger: logger,
	}
}

func (m *MockProtocolEngine) ID() string                { return m.id }
func (m *MockProtocolEngine) Name() string              { return m.name }
func (m *MockProtocolEngine) Capabilities() []string   { return []string{"clinical_protocol", "guideline_compliance"} }
func (m *MockProtocolEngine) HealthCheck() error       { return nil }
func (m *MockProtocolEngine) Initialize(config types.EngineConfig) error { return nil }
func (m *MockProtocolEngine) Shutdown() error          { return nil }

func (m *MockProtocolEngine) Evaluate(ctx context.Context, req *types.SafetyRequest, clinicalContext *types.ClinicalContext) (*types.EngineResult, error) {
	time.Sleep(time.Duration(rand.Intn(40)) * time.Millisecond)

	result := &types.EngineResult{
		EngineID:   m.id,
		EngineName: m.name,
		Status:     types.SafetyStatusSafe,
		RiskScore:  rand.Float64() * 0.3, // 0-30% risk
		Confidence: 0.8 + rand.Float64()*0.2, // 80-100% confidence
	}

	// Check for protocol compliance based on action type
	switch req.ActionType {
	case "medication_order":
		if req.Priority == "emergency" && len(req.MedicationIDs) > 2 {
			result.Status = types.SafetyStatusWarning
			result.RiskScore = 0.5
			result.Warnings = []string{"Emergency medication order with multiple drugs - verify protocol compliance"}
		}
	case "procedure_order":
		result.Status = types.SafetyStatusWarning
		result.RiskScore = 0.4
		result.Warnings = []string{"Procedure order requires protocol verification"}
	}

	// Check patient demographics for age-specific protocols
	if clinicalContext != nil && clinicalContext.Demographics != nil {
		if clinicalContext.Demographics.Age >= 65 {
			result.Warnings = append(result.Warnings, "Elderly patient - consider geriatric protocols")
			result.RiskScore += 0.1
		}
		if clinicalContext.Demographics.Age < 18 {
			result.Warnings = append(result.Warnings, "Pediatric patient - verify pediatric protocols")
			result.RiskScore += 0.15
		}
	}

	return result, nil
}

// MockConstraintEngine implements a mock constraint validation engine
type MockConstraintEngine struct {
	id     string
	name   string
	logger *logger.Logger
}

func NewMockConstraintEngine(logger *logger.Logger) *MockConstraintEngine {
	return &MockConstraintEngine{
		id:     "constraint_engine",
		name:   "Constraint Validation Engine",
		logger: logger,
	}
}

func (m *MockConstraintEngine) ID() string                { return m.id }
func (m *MockConstraintEngine) Name() string              { return m.name }
func (m *MockConstraintEngine) Capabilities() []string   { return []string{"hard_constraints", "safety_limits"} }
func (m *MockConstraintEngine) HealthCheck() error       { return nil }
func (m *MockConstraintEngine) Initialize(config types.EngineConfig) error { return nil }
func (m *MockConstraintEngine) Shutdown() error          { return nil }

func (m *MockConstraintEngine) Evaluate(ctx context.Context, req *types.SafetyRequest, clinicalContext *types.ClinicalContext) (*types.EngineResult, error) {
	time.Sleep(time.Duration(rand.Intn(20)) * time.Millisecond)

	result := &types.EngineResult{
		EngineID:   m.id,
		EngineName: m.name,
		Status:     types.SafetyStatusSafe,
		RiskScore:  0.0,
		Confidence: 1.0, // Constraints are binary - 100% confidence
	}

	// Hard constraint checks
	if len(req.MedicationIDs) > 10 {
		result.Status = types.SafetyStatusUnsafe
		result.RiskScore = 1.0
		result.Violations = []string{"Medication count exceeds safety limit (max 10)"}
		return result, nil
	}

	if len(req.ConditionIDs) > 20 {
		result.Status = types.SafetyStatusUnsafe
		result.RiskScore = 1.0
		result.Violations = []string{"Condition count exceeds system limit (max 20)"}
		return result, nil
	}

	// Check for emergency priority constraints
	if req.Priority == "emergency" {
		if req.ActionType != "medication_order" && req.ActionType != "procedure_order" {
			result.Status = types.SafetyStatusUnsafe
			result.RiskScore = 0.8
			result.Violations = []string{"Emergency priority not allowed for this action type"}
		}
	}

	// Validate patient context constraints
	if clinicalContext != nil && clinicalContext.Demographics != nil {
		age := clinicalContext.Demographics.Age
		if age < 0 || age > 150 {
			result.Status = types.SafetyStatusUnsafe
			result.RiskScore = 1.0
			result.Violations = []string{"Invalid patient age detected"}
		}
	}

	return result, nil
}
