package services

import (
	"go.uber.org/zap"

	"kb-23-decision-cards/internal/models"
)

// TemplateSelector maps a DifferentialSnapshot to the best-matching
// CardTemplate. It supports V-09 secondary differential auto-inclusion
// and most-restrictive MCU gate computation across secondaries.
type TemplateSelector struct {
	loader *TemplateLoader
	log    *zap.Logger
}

// NewTemplateSelector creates a TemplateSelector backed by the given
// TemplateLoader.
func NewTemplateSelector(loader *TemplateLoader, log *zap.Logger) *TemplateSelector {
	return &TemplateSelector{loader: loader, log: log}
}

// SelectBest finds the best matching template for the top differential and
// node. It first tries an exact differential+node match, then falls back to
// differential-only matching.
func (s *TemplateSelector) SelectBest(differentialID, nodeID string) *models.CardTemplate {
	// First try exact differential match
	templates := s.loader.GetByDifferential(differentialID)
	for _, t := range templates {
		if t.NodeID == nodeID {
			s.log.Debug("template matched by differential+node",
				zap.String("template_id", t.TemplateID),
				zap.String("differential_id", differentialID),
				zap.String("node_id", nodeID),
			)
			return t
		}
	}

	// Fallback: match by differential only
	if len(templates) > 0 {
		s.log.Debug("template matched by differential only",
			zap.String("template_id", templates[0].TemplateID),
			zap.String("differential_id", differentialID),
		)
		return templates[0]
	}

	s.log.Debug("no template match",
		zap.String("differential_id", differentialID),
		zap.String("node_id", nodeID),
	)
	return nil
}

// SelectSecondaryTemplates returns templates for secondary differentials
// (V-09). Only INVESTIGATION and MONITORING recommendations are
// auto-included from secondaries. The excludeTop parameter is the primary
// differential ID to skip.
func (s *TemplateSelector) SelectSecondaryTemplates(differentials []models.DifferentialEntry, nodeID string, excludeTop string) []*models.CardTemplate {
	var secondaries []*models.CardTemplate
	for _, diff := range differentials {
		if diff.DifferentialID == excludeTop {
			continue
		}
		tmpl := s.SelectBest(diff.DifferentialID, nodeID)
		if tmpl != nil {
			secondaries = append(secondaries, tmpl)
		}
	}
	return secondaries
}

// MostRestrictiveGateFromSecondaries returns the most restrictive MCU_GATE
// across secondary differential templates (V-09). Starts from GateSafe and
// escalates to whichever secondary template carries the highest gate level.
func (s *TemplateSelector) MostRestrictiveGateFromSecondaries(secondaries []*models.CardTemplate) models.MCUGate {
	gate := models.GateSafe
	for _, tmpl := range secondaries {
		gate = models.MostRestrictive(gate, tmpl.MCUGateDefault)
	}
	return gate
}
