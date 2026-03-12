package services

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"

	"kb-2-clinical-context-go/internal/config"
	"kb-2-clinical-context-go/internal/models"
)

// TreatmentPreferenceService provides treatment recommendation functionality
type TreatmentPreferenceService struct {
	config      *config.Config
	mongoClient *mongo.Client
	redisClient *redis.Client
	
	// Treatment knowledge base
	treatmentOptions    map[string][]models.TreatmentOption
	institutionalRules  map[string]InstitutionalRule
	preferenceRules     map[string]PreferenceRule
}

// InstitutionalRule represents institutional treatment rules
type InstitutionalRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Condition   string                 `json:"condition"`
	Rule        string                 `json:"rule"`
	Priority    int                    `json:"priority"`
	Weight      float64                `json:"weight"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// PreferenceRule represents patient preference matching rules
type PreferenceRule struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	PreferenceType  string                 `json:"preference_type"`
	MatchingLogic   string                 `json:"matching_logic"`
	Weight          float64                `json:"weight"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// NewTreatmentPreferenceService creates a new treatment preference service
func NewTreatmentPreferenceService(mongoClient *mongo.Client, redisClient *redis.Client) *TreatmentPreferenceService {
	service := &TreatmentPreferenceService{
		mongoClient:        mongoClient,
		redisClient:        redisClient,
		treatmentOptions:   make(map[string][]models.TreatmentOption),
		institutionalRules: make(map[string]InstitutionalRule),
		preferenceRules:    make(map[string]PreferenceRule),
	}
	
	// Initialize default treatment knowledge
	service.initializeTreatmentKnowledge()
	
	return service
}

// initializeTreatmentKnowledge initializes default treatment options and rules
func (tps *TreatmentPreferenceService) initializeTreatmentKnowledge() {
	// Initialize diabetes treatment options
	tps.treatmentOptions["diabetes"] = []models.TreatmentOption{
		{
			ID:       "metformin",
			Name:     "Metformin",
			Category: "first_line",
			Suitability: 0.9,
			Contraindications: []string{"severe_kidney_disease", "liver_disease"},
			Preferences: []models.PreferenceMatch{
				{Preference: "oral_medication", Match: true, Weight: 0.8},
				{Preference: "low_cost", Match: true, Weight: 0.9},
				{Preference: "weight_neutral", Match: true, Weight: 0.7},
			},
			Evidence: models.EvidenceLevel{
				Grade:       "A",
				Level:       1,
				Description: "Strong evidence from multiple RCTs",
				References:  []string{"ADA 2023", "EASD 2023"},
			},
			Cost: models.CostProfile{
				Category:      "low",
				EstimatedCost: 15.0,
				Currency:      "USD",
			},
		},
		{
			ID:       "insulin_glargine",
			Name:     "Insulin Glargine",
			Category: "insulin",
			Suitability: 0.8,
			Contraindications: []string{},
			Preferences: []models.PreferenceMatch{
				{Preference: "once_daily", Match: true, Weight: 0.7},
				{Preference: "injectable", Match: false, Weight: 0.3},
				{Preference: "high_efficacy", Match: true, Weight: 0.9},
			},
			Evidence: models.EvidenceLevel{
				Grade:       "A",
				Level:       1,
				Description: "Proven efficacy in T2DM",
				References:  []string{"Cochrane 2023"},
			},
			Cost: models.CostProfile{
				Category:      "high",
				EstimatedCost: 200.0,
				Currency:      "USD",
			},
		},
		{
			ID:       "semaglutide",
			Name:     "Semaglutide",
			Category: "glp1_agonist",
			Suitability: 0.85,
			Contraindications: []string{"pancreatitis_history", "medullary_thyroid_cancer"},
			Preferences: []models.PreferenceMatch{
				{Preference: "weight_loss", Match: true, Weight: 0.95},
				{Preference: "cv_protection", Match: true, Weight: 0.9},
				{Preference: "weekly_injection", Match: true, Weight: 0.8},
			},
			Evidence: models.EvidenceLevel{
				Grade:       "A",
				Level:       1,
				Description: "Cardiovascular and weight benefits",
				References:  []string{"SUSTAIN trials", "STEP trials"},
			},
			Cost: models.CostProfile{
				Category:      "very_high",
				EstimatedCost: 800.0,
				Currency:      "USD",
			},
		},
	}
	
	// Initialize hypertension treatment options
	tps.treatmentOptions["hypertension"] = []models.TreatmentOption{
		{
			ID:       "lisinopril",
			Name:     "Lisinopril",
			Category: "ace_inhibitor",
			Suitability: 0.9,
			Contraindications: []string{"pregnancy", "angioedema_history", "bilateral_renal_stenosis"},
			Preferences: []models.PreferenceMatch{
				{Preference: "once_daily", Match: true, Weight: 0.8},
				{Preference: "cv_protection", Match: true, Weight: 0.9},
				{Preference: "kidney_protection", Match: true, Weight: 0.85},
			},
			Evidence: models.EvidenceLevel{
				Grade:       "A",
				Level:       1,
				Description: "First-line therapy for hypertension",
				References:  []string{"ACC/AHA 2017", "ESC/ESH 2018"},
			},
			Cost: models.CostProfile{
				Category:      "low",
				EstimatedCost: 10.0,
				Currency:      "USD",
			},
		},
		{
			ID:       "amlodipine",
			Name:     "Amlodipine",
			Category: "calcium_channel_blocker",
			Suitability: 0.85,
			Contraindications: []string{"severe_aortic_stenosis"},
			Preferences: []models.PreferenceMatch{
				{Preference: "once_daily", Match: true, Weight: 0.8},
				{Preference: "elderly_friendly", Match: true, Weight: 0.9},
				{Preference: "minimal_side_effects", Match: false, Weight: 0.6}, // ankle edema
			},
			Evidence: models.EvidenceLevel{
				Grade:       "A",
				Level:       1,
				Description: "Effective antihypertensive with CV outcomes",
				References:  []string{"ASCOT", "VALUE"},
			},
			Cost: models.CostProfile{
				Category:      "low",
				EstimatedCost: 12.0,
				Currency:      "USD",
			},
		},
	}
	
	// Initialize institutional rules
	tps.institutionalRules["diabetes_first_line"] = InstitutionalRule{
		ID:        "diabetes_first_line",
		Name:      "Diabetes First-Line Treatment",
		Condition: "diabetes",
		Rule:      "metformin_unless_contraindicated",
		Priority:  1,
		Weight:    0.9,
		Metadata:  map[string]interface{}{"guideline": "ADA 2023"},
	}
	
	tps.institutionalRules["cost_effectiveness"] = InstitutionalRule{
		ID:        "cost_effectiveness",
		Name:      "Cost-Effectiveness Priority",
		Condition: "*",
		Rule:      "prefer_generic_when_equivalent",
		Priority:  3,
		Weight:    0.6,
		Metadata:  map[string]interface{}{"policy": "formulary_preference"},
	}
	
	// Initialize preference rules
	tps.preferenceRules["dosing_frequency"] = PreferenceRule{
		ID:             "dosing_frequency",
		Name:           "Dosing Frequency Preference",
		PreferenceType: "convenience",
		MatchingLogic:  "prefer_once_daily",
		Weight:         0.7,
		Metadata:       map[string]interface{}{"adherence_impact": "high"},
	}
	
	tps.preferenceRules["injection_aversion"] = PreferenceRule{
		ID:             "injection_aversion",
		Name:           "Injection Aversion",
		PreferenceType: "delivery_method",
		MatchingLogic:  "avoid_injections_if_possible",
		Weight:         0.8,
		Metadata:       map[string]interface{}{"patient_factor": "needle_phobia"},
	}
}

// EvaluateTreatmentPreferences evaluates treatment options based on patient and institutional preferences
func (tps *TreatmentPreferenceService) EvaluateTreatmentPreferences(ctx context.Context, request *models.TreatmentPreferencesRequest) (*models.TreatmentPreferencesResult, error) {
	startTime := time.Now()
	
	result := &models.TreatmentPreferencesResult{
		PatientID:          request.PatientID,
		Condition:          request.Condition,
		TreatmentOptions:   []models.TreatmentOption{},
		PreferredTreatments: []models.PreferredTreatment{},
		ConflictResolution: []models.ConflictResolution{},
		GeneratedAt:        time.Now(),
	}
	
	// Get available treatment options for the condition
	options, exists := tps.treatmentOptions[request.Condition]
	if !exists {
		return nil, fmt.Errorf("no treatment options available for condition: %s", request.Condition)
	}
	
	// Filter options based on contraindications
	filteredOptions := tps.filterByContraindications(options, request.PatientData)
	
	// Score each treatment option
	scoredOptions := []ScoredTreatmentOption{}
	for _, option := range filteredOptions {
		score := tps.calculateTreatmentScore(option, request)
		scoredOptions = append(scoredOptions, ScoredTreatmentOption{
			Option: option,
			Score:  score,
		})
	}
	
	// Sort by score (descending)
	sort.Slice(scoredOptions, func(i, j int) bool {
		return scoredOptions[i].Score > scoredOptions[j].Score
	})
	
	// Apply conflict resolution
	conflicts, resolvedOptions := tps.resolveConflicts(scoredOptions, request)
	result.ConflictResolution = conflicts
	
	// Generate final recommendations
	result.TreatmentOptions = tps.extractTreatmentOptions(resolvedOptions)
	result.PreferredTreatments = tps.generatePreferredTreatments(resolvedOptions)
	
	result.ProcessingTime = time.Since(startTime)
	
	// Check SLA compliance (50ms target)
	slaThreshold := time.Duration(50) * time.Millisecond
	if result.ProcessingTime > slaThreshold {
		return result, fmt.Errorf("SLA violation: treatment preference evaluation took %v, threshold is %v", result.ProcessingTime, slaThreshold)
	}
	
	return result, nil
}

// ScoredTreatmentOption represents a treatment option with calculated score
type ScoredTreatmentOption struct {
	Option models.TreatmentOption
	Score  float64
}

// filterByContraindications filters treatments based on patient contraindications
func (tps *TreatmentPreferenceService) filterByContraindications(options []models.TreatmentOption, patient models.Patient) []models.TreatmentOption {
	filtered := []models.TreatmentOption{}
	
	for _, option := range options {
		contraindicated := false
		
		// Check each contraindication
		for _, contraindication := range option.Contraindications {
			if tps.hasContraindication(patient, contraindication) {
				contraindicated = true
				break
			}
		}
		
		if !contraindicated {
			filtered = append(filtered, option)
		}
	}
	
	return filtered
}

// hasContraindication checks if patient has a specific contraindication
func (tps *TreatmentPreferenceService) hasContraindication(patient models.Patient, contraindication string) bool {
	// Check conditions
	for _, condition := range patient.Conditions {
		if condition == contraindication {
			return true
		}
	}
	
	// Check allergies
	for _, allergy := range patient.Allergies {
		if allergy == contraindication {
			return true
		}
	}
	
	// Check special cases
	switch contraindication {
	case "pregnancy":
		if patient.Gender == "female" {
			// In real implementation, would check pregnancy status
			if pregnant, exists := patient.Metadata["pregnant"]; exists {
				if pregnantBool, ok := pregnant.(bool); ok && pregnantBool {
					return true
				}
			}
		}
	case "severe_kidney_disease":
		if creatinine, exists := patient.Labs["creatinine"]; exists {
			if creatinine.Value > 2.0 { // mg/dL
				return true
			}
		}
	case "liver_disease":
		if alt, exists := patient.Labs["alt"]; exists {
			if alt.Value > 120 { // U/L, 3x upper normal limit
				return true
			}
		}
	}
	
	return false
}

// calculateTreatmentScore calculates a composite score for a treatment option
func (tps *TreatmentPreferenceService) calculateTreatmentScore(option models.TreatmentOption, request *models.TreatmentPreferencesRequest) float64 {
	score := option.Suitability // Base suitability score
	
	// Apply preference matching
	preferenceScore := tps.calculatePreferenceScore(option, request.PreferenceProfile)
	score += preferenceScore * 0.3 // 30% weight for preferences
	
	// Apply institutional rules
	institutionalScore := tps.calculateInstitutionalScore(option, request)
	score += institutionalScore * 0.4 // 40% weight for institutional rules
	
	// Apply evidence weight
	evidenceScore := tps.calculateEvidenceScore(option.Evidence)
	score += evidenceScore * 0.2 // 20% weight for evidence
	
	// Apply cost considerations
	costScore := tps.calculateCostScore(option.Cost)
	score += costScore * 0.1 // 10% weight for cost
	
	return math.Min(score, 1.0) // Cap at 1.0
}

// calculatePreferenceScore calculates score based on patient preferences
func (tps *TreatmentPreferenceService) calculatePreferenceScore(option models.TreatmentOption, preferenceProfile map[string]interface{}) float64 {
	if preferenceProfile == nil {
		return 0.5 // Neutral score when no preferences specified
	}
	
	totalScore := 0.0
	totalWeight := 0.0
	
	for _, prefMatch := range option.Preferences {
		// Check if patient has this preference
		if prefValue, exists := preferenceProfile[prefMatch.Preference]; exists {
			if prefBool, ok := prefValue.(bool); ok {
				if prefBool == prefMatch.Match {
					// Preference matches
					totalScore += prefMatch.Weight
				} else {
					// Preference conflicts
					totalScore -= prefMatch.Weight * 0.5
				}
				totalWeight += prefMatch.Weight
			}
		}
	}
	
	if totalWeight > 0 {
		return totalScore / totalWeight
	}
	return 0.5 // Neutral when no applicable preferences
}

// calculateInstitutionalScore calculates score based on institutional rules
func (tps *TreatmentPreferenceService) calculateInstitutionalScore(option models.TreatmentOption, request *models.TreatmentPreferencesRequest) float64 {
	score := 0.0
	
	// Apply condition-specific institutional rules
	for _, rule := range tps.institutionalRules {
		if rule.Condition == request.Condition || rule.Condition == "*" {
			ruleScore := tps.applyInstitutionalRule(rule, option, request)
			score += ruleScore * rule.Weight
		}
	}
	
	return score
}

// applyInstitutionalRule applies a specific institutional rule
func (tps *TreatmentPreferenceService) applyInstitutionalRule(rule InstitutionalRule, option models.TreatmentOption, request *models.TreatmentPreferencesRequest) float64 {
	switch rule.Rule {
	case "metformin_unless_contraindicated":
		if option.ID == "metformin" {
			return 1.0 // Strongly prefer metformin
		}
		return 0.3 // Lower score for non-metformin
		
	case "prefer_generic_when_equivalent":
		if option.Cost.Category == "low" {
			return 1.0 // Prefer low-cost options
		}
		return 0.5 // Neutral for higher-cost options
		
	default:
		return 0.5 // Neutral when rule doesn't apply
	}
}

// calculateEvidenceScore calculates score based on evidence quality
func (tps *TreatmentPreferenceService) calculateEvidenceScore(evidence models.EvidenceLevel) float64 {
	switch evidence.Grade {
	case "A":
		return 1.0
	case "B":
		return 0.8
	case "C":
		return 0.6
	default:
		return 0.4
	}
}

// calculateCostScore calculates score based on cost considerations
func (tps *TreatmentPreferenceService) calculateCostScore(cost models.CostProfile) float64 {
	switch cost.Category {
	case "low":
		return 1.0
	case "moderate":
		return 0.8
	case "high":
		return 0.6
	case "very_high":
		return 0.4
	default:
		return 0.5
	}
}

// resolveConflicts resolves conflicts between different scoring criteria
func (tps *TreatmentPreferenceService) resolveConflicts(options []ScoredTreatmentOption, request *models.TreatmentPreferencesRequest) ([]models.ConflictResolution, []ScoredTreatmentOption) {
	conflicts := []models.ConflictResolution{}
	resolved := options
	
	// Detect high-cost but high-efficacy conflicts
	for i, option := range options {
		if option.Option.Cost.Category == "very_high" && option.Score > 0.8 {
			// High-cost but high-scoring treatment
			conflict := models.ConflictResolution{
				ConflictType: "cost_efficacy",
				Resolution:   "Consider step therapy or prior authorization",
				Priority:     "medium",
				Rationale:    fmt.Sprintf("High-efficacy treatment %s has very high cost", option.Option.Name),
				AppliedRules: []string{"cost_effectiveness", "clinical_efficacy"},
			}
			conflicts = append(conflicts, conflict)
			
			// Apply penalty to cost score
			resolved[i].Score *= 0.9
		}
	}
	
	// Detect preference vs institutional rule conflicts
	for i, option := range options {
		// Check if institutional rule conflicts with patient preference
		if tps.hasPreferenceInstitutionalConflict(option.Option, request) {
			conflict := models.ConflictResolution{
				ConflictType: "preference_institutional",
				Resolution:   "Institutional guideline takes precedence with patient education",
				Priority:     "high",
				Rationale:    fmt.Sprintf("Patient preference conflicts with institutional guideline for %s", option.Option.Name),
				AppliedRules: []string{"institutional_priority"},
			}
			conflicts = append(conflicts, conflict)
			
			// Boost institutional rule score
			resolved[i].Score *= 1.1
		}
	}
	
	// Re-sort after conflict resolution
	sort.Slice(resolved, func(i, j int) bool {
		return resolved[i].Score > resolved[j].Score
	})
	
	return conflicts, resolved
}

// hasPreferenceInstitutionalConflict checks for conflicts between preferences and institutional rules
func (tps *TreatmentPreferenceService) hasPreferenceInstitutionalConflict(option models.TreatmentOption, request *models.TreatmentPreferencesRequest) bool {
	// Example: Patient prefers non-injectable but institutional rule prefers insulin for T2DM with poor control
	if request.Condition == "diabetes" && option.Category == "insulin" {
		if preferenceProfile := request.PreferenceProfile; preferenceProfile != nil {
			if injectable, exists := preferenceProfile["injectable"]; exists {
				if injectableBool, ok := injectable.(bool); ok && !injectableBool {
					// Patient doesn't want injections but institutional rule may require insulin
					if hba1c, exists := request.PatientData.Labs["hba1c"]; exists && hba1c.Value > 9.0 {
						return true // Conflict: patient avoids injections but needs insulin
					}
				}
			}
		}
	}
	return false
}

// extractTreatmentOptions extracts treatment options from scored options
func (tps *TreatmentPreferenceService) extractTreatmentOptions(scored []ScoredTreatmentOption) []models.TreatmentOption {
	options := make([]models.TreatmentOption, len(scored))
	for i, so := range scored {
		options[i] = so.Option
	}
	return options
}

// generatePreferredTreatments generates ranked preferred treatments
func (tps *TreatmentPreferenceService) generatePreferredTreatments(scored []ScoredTreatmentOption) []models.PreferredTreatment {
	preferred := make([]models.PreferredTreatment, 0, len(scored))
	
	for i, so := range scored {
		// Only include options with reasonable scores
		if so.Score >= 0.4 {
			rationale := tps.generateRationale(so.Option, so.Score, i+1)
			
			preferred = append(preferred, models.PreferredTreatment{
				TreatmentID:  so.Option.ID,
				Rank:         i + 1,
				OverallScore: so.Score,
				Rationale:    rationale,
			})
		}
	}
	
	return preferred
}

// generateRationale generates explanation for treatment ranking
func (tps *TreatmentPreferenceService) generateRationale(option models.TreatmentOption, score float64, rank int) string {
	reasons := []string{}
	
	// Evidence-based rationale
	if option.Evidence.Grade == "A" {
		reasons = append(reasons, "strong clinical evidence")
	}
	
	// Cost rationale
	if option.Cost.Category == "low" {
		reasons = append(reasons, "cost-effective")
	}
	
	// Preference rationale
	matchedPrefs := 0
	for _, pref := range option.Preferences {
		if pref.Match {
			matchedPrefs++
		}
	}
	if matchedPrefs > 0 {
		reasons = append(reasons, fmt.Sprintf("matches %d patient preferences", matchedPrefs))
	}
	
	// Category rationale
	if option.Category == "first_line" {
		reasons = append(reasons, "first-line therapy")
	}
	
	baseRationale := fmt.Sprintf("Ranked #%d with score %.2f", rank, score)
	if len(reasons) > 0 {
		baseRationale += " due to: " + joinStrings(reasons, ", ")
	}
	
	return baseRationale
}

// Utility function to join strings
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}
	
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}