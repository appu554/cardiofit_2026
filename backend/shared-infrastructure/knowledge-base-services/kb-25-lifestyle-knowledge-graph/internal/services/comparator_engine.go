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

// MRIDomainTargets maps intervention codes to the MRI domains they affect.
var MRIDomainTargets = map[string]string{
	"post_meal_walking":   "Glucose Control",
	"carb_quality":        "Glucose Control",
	"metformin_titration": "Glucose Control",
	"exercise_rx":         "Body Composition",
	"protein_increase":    "Body Composition",
	"weight_management":   "Body Composition",
	"bp_medication":       "Cardiovascular Regulation",
	"sodium_reduction":    "Cardiovascular Regulation",
	"sleep_hygiene":       "Behavioral Metabolism",
	"step_increase":       "Behavioral Metabolism",
}

// PrioritizeByMRIDomain reorders ranked options to prioritize interventions
// targeting the highest-scoring MRI domain.
func PrioritizeByMRIDomain(options []models.ComparedOption, topDriver string) []models.ComparedOption {
	if topDriver == "" || len(options) == 0 {
		return options
	}

	for i := range options {
		targetDomain, exists := MRIDomainTargets[options[i].Option.Code]
		if exists && targetDomain == topDriver {
			options[i].MRIBoost = true
		}
	}

	sort.SliceStable(options, func(i, j int) bool {
		if options[i].MRIBoost != options[j].MRIBoost {
			return options[i].MRIBoost
		}
		return options[i].Rank < options[j].Rank
	})

	for i := range options {
		options[i].Rank = i + 1
	}

	return options
}
