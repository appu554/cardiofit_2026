package services

import (
	"fmt"

	"go.uber.org/zap"

	"kb-patient-profile/internal/database"
	"kb-patient-profile/internal/models"
)

// ADRService manages adverse drug reaction profile storage and retrieval.
type ADRService struct {
	db     *database.Database
	logger *zap.Logger
}

// NewADRService creates the ADR profile service.
func NewADRService(db *database.Database, logger *zap.Logger) *ADRService {
	return &ADRService{db: db, logger: logger}
}

// GetByDrugClass retrieves all active ADR profiles for a drug class.
// STUB records are excluded from clinical consumption but visible in dashboards.
func (s *ADRService) GetByDrugClass(drugClass string, includeStubs bool) ([]models.AdverseReactionProfile, error) {
	var profiles []models.AdverseReactionProfile
	query := s.db.DB.Where("drug_class = ? AND active = true", drugClass)
	if !includeStubs {
		query = query.Where("completeness_grade != 'STUB'")
	}
	if err := query.Find(&profiles).Error; err != nil {
		return nil, fmt.Errorf("failed to get ADR profiles: %w", err)
	}
	return profiles, nil
}

// Upsert creates or updates an ADR profile. On conflict (drug_class + reaction),
// applies the merge strategy: SPL mechanism/onset retained; pipeline context_modifier_rule retained.
func (s *ADRService) Upsert(profile *models.AdverseReactionProfile) error {
	// Check for existing record with same drug_class + reaction
	var existing models.AdverseReactionProfile
	err := s.db.DB.Where("drug_class = ? AND reaction = ? AND active = true",
		profile.DrugClass, profile.Reaction).First(&existing).Error

	if err != nil {
		// No existing record — create new
		return s.db.DB.Create(profile).Error
	}

	// Merge strategy: combine best data from both sources
	merged := s.mergeProfiles(&existing, profile)
	return s.db.DB.Model(&existing).Updates(merged).Error
}

// mergeProfiles applies the dual-path merge strategy with MANUAL_CURATED priority.
//
// Source priority hierarchy (highest to lowest):
//  1. MANUAL_CURATED — clinician-verified data always wins all fields
//  2. SPL — wins mechanism and onset fields (structured labeling data)
//  3. PIPELINE — wins context_modifier_rule (computed from clinical rules)
//
// When MANUAL_CURATED is the incoming source, ALL non-empty fields overwrite.
// When MANUAL_CURATED is the existing source, only MANUAL_CURATED incoming can overwrite.
func (s *ADRService) mergeProfiles(existing, incoming *models.AdverseReactionProfile) map[string]interface{} {
	updates := map[string]interface{}{}

	// MANUAL_CURATED is the highest priority — always wins
	if incoming.Source == "MANUAL_CURATED" {
		if incoming.Mechanism != "" {
			updates["mechanism"] = incoming.Mechanism
		}
		if incoming.OnsetWindow != "" {
			updates["onset_window"] = incoming.OnsetWindow
			updates["onset_category"] = incoming.OnsetCategory
		}
		if incoming.ContextModifierRule != "" {
			updates["context_modifier_rule"] = incoming.ContextModifierRule
		}
		if incoming.Confidence.GreaterThan(existing.Confidence) {
			updates["confidence"] = incoming.Confidence
		}
		updates["source"] = "MANUAL_CURATED"
		if len(updates) > 0 {
			updates["completeness_grade"] = computeMergedGrade(existing, updates)
		}
		return updates
	}

	// If existing is MANUAL_CURATED, do not overwrite with lower-priority sources
	if existing.Source == "MANUAL_CURATED" {
		s.logger.Debug("skipping merge: existing is MANUAL_CURATED",
			zap.String("drug_class", existing.DrugClass),
			zap.String("incoming_source", incoming.Source),
		)
		return updates
	}

	// SPL provides better mechanism and onset data
	if existing.Source == "SPL" && incoming.Source == "PIPELINE" {
		if incoming.ContextModifierRule != "" {
			updates["context_modifier_rule"] = incoming.ContextModifierRule
		}
		if incoming.Confidence.GreaterThan(existing.Confidence) {
			updates["confidence"] = incoming.Confidence
		}
	} else if existing.Source == "PIPELINE" && incoming.Source == "SPL" {
		if incoming.Mechanism != "" {
			updates["mechanism"] = incoming.Mechanism
		}
		if incoming.OnsetWindow != "" {
			updates["onset_window"] = incoming.OnsetWindow
			updates["onset_category"] = incoming.OnsetCategory
		}
	}

	// Always upgrade if incoming has more data
	if incoming.Mechanism != "" && existing.Mechanism == "" {
		updates["mechanism"] = incoming.Mechanism
	}
	if incoming.OnsetWindow != "" && existing.OnsetWindow == "" {
		updates["onset_window"] = incoming.OnsetWindow
		updates["onset_category"] = incoming.OnsetCategory
	}
	if incoming.ContextModifierRule != "" && existing.ContextModifierRule == "" {
		updates["context_modifier_rule"] = incoming.ContextModifierRule
	}

	// Recompute completeness grade on merged result
	if len(updates) > 0 {
		updates["completeness_grade"] = computeMergedGrade(existing, updates)
	}

	return updates
}

func computeMergedGrade(existing *models.AdverseReactionProfile, updates map[string]interface{}) string {
	hasMech := existing.Mechanism != ""
	if m, ok := updates["mechanism"]; ok {
		hasMech = m.(string) != ""
	}
	hasOnset := existing.OnsetWindow != ""
	if o, ok := updates["onset_window"]; ok {
		hasOnset = o.(string) != ""
	}
	hasCMR := existing.ContextModifierRule != "" && existing.ContextModifierRule != "{}"
	if c, ok := updates["context_modifier_rule"]; ok {
		hasCMR = c.(string) != "" && c.(string) != "{}"
	}

	if existing.DrugName != "" && existing.Reaction != "" && hasOnset && hasMech && hasCMR {
		return "FULL"
	}
	if existing.DrugName != "" && existing.Reaction != "" && (hasOnset || hasMech) {
		return "PARTIAL"
	}
	return "STUB"
}
