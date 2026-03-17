package services

import (
	"math"
	"sort"

	"kb-25-lifestyle-knowledge-graph/internal/models"

	"go.uber.org/zap"
)

type ComparatorEngine struct {
	chainService *ChainTraversalService
	logger       *zap.Logger
}

func NewComparatorEngine(chainSvc *ChainTraversalService, logger *zap.Logger) *ComparatorEngine {
	return &ComparatorEngine{chainService: chainSvc, logger: logger}
}

func ApplyDecisionRule(hba1c, sbp float64) string {
	if sbp > 0 {
		if sbp > 160 {
			return "MEDICATION_PRIMARY"
		}
		if sbp >= 130 && sbp <= 150 {
			return "LIFESTYLE_FIRST"
		}
	}

	switch {
	case hba1c >= 5.7 && hba1c < 6.5:
		return "LIFESTYLE_ONLY"
	case hba1c >= 6.5 && hba1c <= 7.5:
		return "LIFESTYLE_FIRST"
	case hba1c > 7.5 && hba1c <= 9.0:
		return "COMBINED"
	case hba1c > 9.0:
		return "MEDICATION_PRIMARY"
	default:
		return "LIFESTYLE_FIRST"
	}
}

func RankOptions(options []models.ComparedOption) []models.ComparedOption {
	sort.Slice(options, func(i, j int) bool {
		absI := math.Abs(options[i].ProjectedEffect)
		absJ := math.Abs(options[j].ProjectedEffect)
		if absI != absJ {
			return absI > absJ
		}
		gradeOrder := map[string]int{"A": 4, "B": 3, "C": 2, "D": 1}
		if gradeOrder[options[i].EvidenceGrade] != gradeOrder[options[j].EvidenceGrade] {
			return gradeOrder[options[i].EvidenceGrade] > gradeOrder[options[j].EvidenceGrade]
		}
		return options[i].SafetyScore > options[j].SafetyScore
	})

	for i := range options {
		options[i].Rank = i + 1
	}
	return options
}
