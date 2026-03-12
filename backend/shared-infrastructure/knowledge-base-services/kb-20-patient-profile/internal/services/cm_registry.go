package services

import (
	"go.uber.org/zap"

	"kb-patient-profile/internal/database"
	"kb-patient-profile/internal/models"
)

// CMRegistry manages the context modifier registry and evaluates which modifiers
// are active for a patient/node combination.
type CMRegistry struct {
	db     *database.Database
	logger *zap.Logger
}

// NewCMRegistry creates the context modifier registry.
func NewCMRegistry(db *database.Database, logger *zap.Logger) *CMRegistry {
	return &CMRegistry{db: db, logger: logger}
}

// GetActiveModifiers returns all context modifiers active for a patient at a given HPI node.
// Completeness-grade-aware: FULL at 1.0x, PARTIAL at 0.7x, STUB excluded.
func (r *CMRegistry) GetActiveModifiers(nodeID string, medications []models.MedicationState, patientID string) []models.ActiveModifier {
	// Get all drug classes the patient is on (including FDC decomposition)
	drugClasses := make(map[string]bool)
	for _, med := range medications {
		for _, dc := range med.EffectiveDrugClasses() {
			drugClasses[dc] = true
		}
	}

	if len(drugClasses) == 0 {
		return nil
	}

	// Build list of drug class strings
	var classList []string
	for dc := range drugClasses {
		classList = append(classList, dc)
	}

	// Query matching modifiers
	var modifiers []models.ContextModifier
	r.db.DB.Where(
		"target_node_id = ? AND drug_class_trigger IN ? AND active = true AND completeness_grade != 'STUB'",
		nodeID, classList,
	).Find(&modifiers)

	// Filter by confidence threshold
	var active []models.ActiveModifier
	for _, cm := range modifiers {
		conf, _ := cm.Confidence.Float64()
		if conf < models.ConfidenceCalibrate {
			continue // Below 0.70 → ignored
		}

		mag := cm.EffectiveMagnitude()
		if mag == 0 {
			continue
		}

		active = append(active, models.ActiveModifier{
			ModifierID:         cm.ID.String(),
			ModifierType:       cm.ModifierType,
			DrugClassTrigger:   cm.DrugClassTrigger,
			Effect:             cm.Effect,
			TargetDifferential: cm.TargetDifferential,
			Magnitude:          mag,
			CompletenessGrade:  cm.CompletenessGrade,
			EffectiveMagnitude: mag,
		})
	}

	return active
}

// GetRegistryForNode returns the full CM registry for a node (including STUBs for dashboard).
func (r *CMRegistry) GetRegistryForNode(nodeID string) ([]models.ContextModifier, error) {
	var modifiers []models.ContextModifier
	if err := r.db.DB.Where("target_node_id = ? AND active = true", nodeID).Find(&modifiers).Error; err != nil {
		return nil, err
	}
	return modifiers, nil
}
