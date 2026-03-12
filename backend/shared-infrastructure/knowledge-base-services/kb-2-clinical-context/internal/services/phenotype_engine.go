package services

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.uber.org/zap"
	"github.com/redis/go-redis/v9"

	"kb-clinical-context/internal/models"
	"kb-clinical-context/internal/engines"
	"kb-clinical-context/internal/loaders"
)

// PhenotypeEngine handles clinical phenotype detection and analysis
type PhenotypeEngine struct {
	db              *mongo.Database
	cache           *redis.Client
	logger          *zap.Logger
	config          PhenotypeEngineConfig
	multiEngine     *engines.MultiEngineEvaluator
	phenotypeLoader *loaders.PhenotypeLoader
}

// PhenotypeEngineConfig contains configuration for the phenotype engine
type PhenotypeEngineConfig struct {
	ConfidenceThreshold   float64
	MaxConcurrentDetection int
	CacheTTL               time.Duration
	EnableDetailedLogging  bool
}

// NewPhenotypeEngine creates a new phenotype detection engine with CEL support
func NewPhenotypeEngine(db *mongo.Database, cache *redis.Client, logger *zap.Logger, phenotypeDir string) (*PhenotypeEngine, error) {
	config := PhenotypeEngineConfig{
		ConfidenceThreshold:   0.7,
		MaxConcurrentDetection: 10,
		CacheTTL:               15 * time.Minute,
		EnableDetailedLogging:  false,
	}

	// Initialize multi-engine evaluator
	multiEngine, err := engines.NewMultiEngineEvaluator(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create multi-engine evaluator: %w", err)
	}

	// Initialize phenotype loader
	loaderConfig := loaders.LoaderConfig{
		PhenotypeDirectory: phenotypeDir,
		ValidateOnLoad:     true,
		EnableCaching:      true,
	}
	phenotypeLoader := loaders.NewPhenotypeLoader(logger, loaderConfig)

	return &PhenotypeEngine{
		db:              db,
		cache:           cache,
		logger:          logger,
		config:          config,
		multiEngine:     multiEngine,
		phenotypeLoader: phenotypeLoader,
	}, nil
}

// DetectPhenotypes performs comprehensive phenotype detection for a patient
func (p *PhenotypeEngine) DetectPhenotypes(ctx context.Context, patientData models.PatientContext) ([]models.DetectedPhenotype, error) {
	startTime := time.Now()
	
	p.logger.Info("Starting phenotype detection",
		zap.String("patient_id", patientData.PatientID),
		zap.Int("active_conditions", len(patientData.ActiveConditions)),
		zap.Int("recent_labs", len(patientData.RecentLabs)),
		zap.Int("current_meds", len(patientData.CurrentMeds)))

	// 1. Check cache first
	cacheKey := fmt.Sprintf("phenotypes:%s", patientData.PatientID)
	if cachedResult := p.getCachedPhenotypes(ctx, cacheKey); cachedResult != nil {
		p.logger.Info("Returning cached phenotype detection results", zap.String("patient_id", patientData.PatientID))
		return cachedResult, nil
	}

	// 2. Load active phenotype definitions from YAML
	yamlPhenotypes, err := p.phenotypeLoader.LoadAllPhenotypes()
	if err != nil {
		return nil, fmt.Errorf("failed to load phenotype definitions: %w", err)
	}

	// Filter for active phenotypes only
	activePhenotypes := make([]engines.PhenotypeDefinitionYAML, 0)
	for _, phenotype := range yamlPhenotypes {
		if phenotype.Status == "active" {
			activePhenotypes = append(activePhenotypes, phenotype)
		}
	}

	p.logger.Info("Loaded phenotype definitions", 
		zap.Int("total_count", len(yamlPhenotypes)),
		zap.Int("active_count", len(activePhenotypes)))

	// 3. Detect phenotypes using multi-engine evaluation
	detectedPhenotypes, err := p.detectWithMultiEngine(ctx, patientData, activePhenotypes)
	if err != nil {
		return nil, fmt.Errorf("failed to detect phenotypes: %w", err)
	}

	// 4. Post-process and apply confidence scoring
	processedPhenotypes := p.postProcessPhenotypes(detectedPhenotypes, patientData)

	// 5. Cache results
	if err := p.cachePhenotypes(ctx, cacheKey, processedPhenotypes); err != nil {
		p.logger.Warn("Failed to cache phenotype results", zap.Error(err))
	}

	processingTime := time.Since(startTime)
	p.logger.Info("Phenotype detection completed",
		zap.String("patient_id", patientData.PatientID),
		zap.Int("detected_count", len(processedPhenotypes)),
		zap.Duration("processing_time", processingTime))

	return processedPhenotypes, nil
}

// detectWithMultiEngine uses the multi-engine evaluator for phenotype detection
func (p *PhenotypeEngine) detectWithMultiEngine(ctx context.Context, patientData models.PatientContext, definitions []engines.PhenotypeDefinitionYAML) ([]models.DetectedPhenotype, error) {
	var detectedPhenotypes []models.DetectedPhenotype

	for _, definition := range definitions {
		// Evaluate phenotype using multi-engine evaluator
		result, err := p.multiEngine.EvaluatePhenotype(definition, patientData)
		if err != nil {
			p.logger.Error("Failed to evaluate phenotype",
				zap.String("phenotype_id", definition.ID),
				zap.Error(err))
			continue
		}

		// Check if phenotype matched and meets confidence threshold
		if result.Matched && result.Confidence >= p.config.ConfidenceThreshold {
			// Convert evidence from engine format to internal format
			evidence := p.convertEvidence(result.Evidence)
			
			detected := models.DetectedPhenotype{
				PhenotypeID:        definition.ID,
				Confidence:         result.Confidence,
				DetectedAt:         time.Now(),
				SupportingEvidence: evidence,
			}
			
			detectedPhenotypes = append(detectedPhenotypes, detected)
			
			if p.config.EnableDetailedLogging {
				p.logger.Info("Phenotype detected using multi-engine",
					zap.String("phenotype_id", definition.ID),
					zap.String("phenotype_name", definition.Name),
					zap.Float64("confidence", result.Confidence),
					zap.String("engine_used", string(result.EngineUsed)),
					zap.Duration("execution_time", result.ExecutionTime))
			}
		}
	}

	return detectedPhenotypes, nil
}

// detectWithAggregation uses MongoDB aggregation pipeline for efficient phenotype detection (legacy method)
func (p *PhenotypeEngine) detectWithAggregation(ctx context.Context, patientData models.PatientContext, definitions []models.PhenotypeDefinition) ([]models.DetectedPhenotype, error) {
	var detectedPhenotypes []models.DetectedPhenotype

	for _, definition := range definitions {
		confidence := p.evaluatePhenotype(patientData, definition)
		
		if confidence >= p.config.ConfidenceThreshold {
			evidence := p.buildSupportingEvidence(patientData, definition)
			
			detected := models.DetectedPhenotype{
				PhenotypeID:        definition.PhenotypeID,
				Confidence:         confidence,
				DetectedAt:         time.Now(),
				SupportingEvidence: evidence,
			}
			
			detectedPhenotypes = append(detectedPhenotypes, detected)
			
			if p.config.EnableDetailedLogging {
				p.logger.Info("Phenotype detected",
					zap.String("phenotype_id", definition.PhenotypeID),
					zap.String("phenotype_name", definition.Name),
					zap.Float64("confidence", confidence))
			}
		}
	}

	return detectedPhenotypes, nil
}

// evaluatePhenotype evaluates if a patient meets the criteria for a specific phenotype
func (p *PhenotypeEngine) evaluatePhenotype(patientData models.PatientContext, definition models.PhenotypeDefinition) float64 {
	var totalScore float64
	var maxScore float64

	// 1. Evaluate condition criteria
	conditionScore, conditionMax := p.evaluateConditionCriteria(patientData.ActiveConditions, definition.Criteria.RequiredConditions)
	totalScore += conditionScore
	maxScore += conditionMax

	// 2. Evaluate lab criteria
	labScore, labMax := p.evaluateLabCriteria(patientData.RecentLabs, definition.Criteria.RequiredLabs)
	totalScore += labScore
	maxScore += labMax

	// 3. Evaluate medication criteria
	medScore, medMax := p.evaluateMedicationCriteria(patientData.CurrentMeds, definition.Criteria.RequiredMeds)
	totalScore += medScore
	maxScore += medMax

	// 4. Apply exclusion criteria
	if p.hasExclusionCriteria(patientData, definition.Criteria.ExclusionCriteria) {
		return 0.0 // Excluded
	}

	// 5. Apply demographic modifiers
	demographicModifier := p.getDemographicModifier(patientData.Demographics, definition.PhenotypeID)
	totalScore *= demographicModifier

	if maxScore == 0 {
		return 0.0
	}

	confidence := totalScore / maxScore
	return math.Min(confidence, 1.0)
}

// evaluateConditionCriteria evaluates condition-based criteria
func (p *PhenotypeEngine) evaluateConditionCriteria(conditions []models.Condition, criteria []models.ConditionCriteria) (float64, float64) {
	if len(criteria) == 0 {
		return 0, 0
	}

	var score, maxScore float64

	for _, criterion := range criteria {
		maxScore += 1.0
		
		matchCount := 0
		for _, condition := range conditions {
			if p.matchesConditionCriteria(condition, criterion) {
				matchCount++
			}
		}

		if matchCount >= criterion.MinOccurrences {
			score += 1.0
		} else if matchCount > 0 {
			// Partial credit based on how many conditions matched
			score += float64(matchCount) / float64(criterion.MinOccurrences)
		}
	}

	return score, maxScore
}

// matchesConditionCriteria checks if a condition matches specific criteria
func (p *PhenotypeEngine) matchesConditionCriteria(condition models.Condition, criteria models.ConditionCriteria) bool {
	// Check if condition code matches any of the required codes
	for _, code := range criteria.Codes {
		if condition.Code == code {
			// Check time window if specified
			if criteria.TimeWindow != "" {
				if !p.isWithinTimeWindow(condition.OnsetDate, criteria.TimeWindow) {
					continue
				}
			}
			return true
		}
	}
	return false
}

// evaluateLabCriteria evaluates laboratory-based criteria
func (p *PhenotypeEngine) evaluateLabCriteria(labs []models.LabResult, criteria []models.LabCriteria) (float64, float64) {
	if len(criteria) == 0 {
		return 0, 0
	}

	var score, maxScore float64

	for _, criterion := range criteria {
		maxScore += 1.0
		
		for _, lab := range labs {
			if lab.LOINCCode == criterion.LOINCCode {
				// Check time window
				if criterion.TimeWindow != "" && !p.isWithinTimeWindow(lab.ResultDate, criterion.TimeWindow) {
					continue
				}

				// Evaluate the lab value against criteria
				if p.evaluateLabValue(lab.Value, criterion.Operator, criterion.Value) {
					score += 1.0
					break // Only count once per criterion
				}
			}
		}
	}

	return score, maxScore
}

// evaluateLabValue evaluates a lab value against an operator and threshold
func (p *PhenotypeEngine) evaluateLabValue(labValue float64, operator string, threshold float64) bool {
	switch strings.ToLower(operator) {
	case ">", "gt":
		return labValue > threshold
	case ">=", "gte":
		return labValue >= threshold
	case "<", "lt":
		return labValue < threshold
	case "<=", "lte":
		return labValue <= threshold
	case "=", "eq", "==":
		return math.Abs(labValue-threshold) < 0.01 // Allow for floating point precision
	case "!=", "ne":
		return math.Abs(labValue-threshold) >= 0.01
	default:
		return false
	}
}

// evaluateMedicationCriteria evaluates medication-based criteria
func (p *PhenotypeEngine) evaluateMedicationCriteria(medications []models.Medication, criteria []models.MedicationCriteria) (float64, float64) {
	if len(criteria) == 0 {
		return 0, 0
	}

	var score, maxScore float64

	for _, criterion := range criteria {
		maxScore += 1.0
		
		for _, medication := range medications {
			if p.medicationMatches(medication, criterion) {
				score += 1.0
				break // Only count once per criterion
			}
		}
	}

	return score, maxScore
}

// medicationMatches checks if a medication matches the criteria
func (p *PhenotypeEngine) medicationMatches(medication models.Medication, criteria models.MedicationCriteria) bool {
	// Check if medication RxNorm code matches
	for _, rxnormCode := range criteria.RxNormCodes {
		if medication.RxNormCode == rxnormCode {
			// Check duration if specified
			if criteria.DurationDays > 0 {
				daysSinceStart := int(time.Since(medication.StartDate).Hours() / 24)
				if daysSinceStart >= criteria.DurationDays {
					return true
				}
			} else {
				return true
			}
		}
	}
	return false
}

// hasExclusionCriteria checks if patient meets any exclusion criteria
func (p *PhenotypeEngine) hasExclusionCriteria(patientData models.PatientContext, exclusions []string) bool {
	for _, exclusion := range exclusions {
		// Check conditions for exclusion criteria
		for _, condition := range patientData.ActiveConditions {
			if condition.Code == exclusion {
				return true
			}
		}
		
		// Check medications for exclusion criteria
		for _, medication := range patientData.CurrentMeds {
			if medication.RxNormCode == exclusion {
				return true
			}
		}
	}
	return false
}

// getDemographicModifier applies demographic-based confidence modifiers
func (p *PhenotypeEngine) getDemographicModifier(demographics models.Demographics, phenotypeID string) float64 {
	modifier := 1.0

	// Age-based modifiers for specific phenotypes
	switch phenotypeID {
	case "diabetes_t2_elderly":
		if demographics.AgeYears >= 65 {
			modifier *= 1.2 // Higher confidence for elderly
		}
	case "hypertension_young_adult":
		if demographics.AgeYears < 40 {
			modifier *= 1.1 // Higher confidence for young adults
		}
	case "ckd_progressive":
		if demographics.AgeYears >= 70 {
			modifier *= 1.15 // Higher risk in elderly
		}
	}

	// Sex-based modifiers
	switch phenotypeID {
	case "osteoporosis_postmenopausal":
		if demographics.Sex == "female" && demographics.AgeYears >= 50 {
			modifier *= 1.3
		}
	case "cad_male_pattern":
		if demographics.Sex == "male" {
			modifier *= 1.1
		}
	}

	return modifier
}

// isWithinTimeWindow checks if a date is within the specified time window
func (p *PhenotypeEngine) isWithinTimeWindow(date time.Time, timeWindow string) bool {
	now := time.Now()
	
	// Parse time window (e.g., "90d", "6m", "1y")
	duration, err := p.parseTimeWindow(timeWindow)
	if err != nil {
		p.logger.Warn("Invalid time window format", zap.String("window", timeWindow), zap.Error(err))
		return true // Default to true if can't parse
	}

	return now.Sub(date) <= duration
}

// parseTimeWindow parses a time window string into a duration
func (p *PhenotypeEngine) parseTimeWindow(window string) (time.Duration, error) {
	if len(window) < 2 {
		return 0, fmt.Errorf("invalid time window format: %s", window)
	}

	numStr := window[:len(window)-1]
	unit := strings.ToLower(window[len(window)-1:])

	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("invalid number in time window: %s", numStr)
	}

	switch unit {
	case "d":
		return time.Duration(num) * 24 * time.Hour, nil
	case "w":
		return time.Duration(num) * 7 * 24 * time.Hour, nil
	case "m":
		return time.Duration(num) * 30 * 24 * time.Hour, nil // Approximate
	case "y":
		return time.Duration(num) * 365 * 24 * time.Hour, nil // Approximate
	default:
		return 0, fmt.Errorf("unknown time unit: %s", unit)
	}
}

// buildSupportingEvidence builds evidence structure for detected phenotype
func (p *PhenotypeEngine) buildSupportingEvidence(patientData models.PatientContext, definition models.PhenotypeDefinition) []map[string]interface{} {
	var evidence []map[string]interface{}

	// Add condition evidence
	for _, condition := range patientData.ActiveConditions {
		for _, criterion := range definition.Criteria.RequiredConditions {
			if p.matchesConditionCriteria(condition, criterion) {
				evidence = append(evidence, map[string]interface{}{
					"type":        "condition",
					"code":        condition.Code,
					"name":        condition.Name,
					"onset_date":  condition.OnsetDate,
					"criterion":   criterion.Type,
				})
			}
		}
	}

	// Add lab evidence
	for _, lab := range patientData.RecentLabs {
		for _, criterion := range definition.Criteria.RequiredLabs {
			if lab.LOINCCode == criterion.LOINCCode && p.evaluateLabValue(lab.Value, criterion.Operator, criterion.Value) {
				evidence = append(evidence, map[string]interface{}{
					"type":       "laboratory",
					"loinc_code": lab.LOINCCode,
					"value":      lab.Value,
					"unit":       lab.Unit,
					"date":       lab.ResultDate,
					"criterion":  fmt.Sprintf("%s %s %.2f", criterion.LOINCCode, criterion.Operator, criterion.Value),
				})
			}
		}
	}

	// Add medication evidence
	for _, medication := range patientData.CurrentMeds {
		for _, criterion := range definition.Criteria.RequiredMeds {
			if p.medicationMatches(medication, criterion) {
				evidence = append(evidence, map[string]interface{}{
					"type":         "medication",
					"rxnorm_code":  medication.RxNormCode,
					"name":         medication.Name,
					"start_date":   medication.StartDate,
					"criterion":    "prescribed_medication",
				})
			}
		}
	}

	return evidence
}

// postProcessPhenotypes applies post-processing logic to detected phenotypes
func (p *PhenotypeEngine) postProcessPhenotypes(detected []models.DetectedPhenotype, patientData models.PatientContext) []models.DetectedPhenotype {
	// Sort by confidence score (highest first)
	for i := 0; i < len(detected); i++ {
		for j := i + 1; j < len(detected); j++ {
			if detected[j].Confidence > detected[i].Confidence {
				detected[i], detected[j] = detected[j], detected[i]
			}
		}
	}

	// Apply phenotype interaction rules
	processed := p.applyPhenotypeInteractionRules(detected)

	// Limit to top phenotypes if too many detected
	maxPhenotypes := 10
	if len(processed) > maxPhenotypes {
		processed = processed[:maxPhenotypes]
	}

	return processed
}

// applyPhenotypeInteractionRules applies rules for phenotype interactions
func (p *PhenotypeEngine) applyPhenotypeInteractionRules(phenotypes []models.DetectedPhenotype) []models.DetectedPhenotype {
	// Example: If both "diabetes_t1" and "diabetes_t2" are detected, keep the higher confidence one
	phenotypeMap := make(map[string]models.DetectedPhenotype)
	
	for _, phenotype := range phenotypes {
		existing, exists := phenotypeMap[phenotype.PhenotypeID]
		if !exists || phenotype.Confidence > existing.Confidence {
			phenotypeMap[phenotype.PhenotypeID] = phenotype
		}
	}

	// Handle mutually exclusive phenotypes
	mutuallyExclusive := map[string][]string{
		"diabetes_t1": {"diabetes_t2"},
		"diabetes_t2": {"diabetes_t1"},
		"ckd_stage_3": {"ckd_stage_4", "ckd_stage_5"},
		"ckd_stage_4": {"ckd_stage_3", "ckd_stage_5"},
		"ckd_stage_5": {"ckd_stage_3", "ckd_stage_4"},
	}

	result := make([]models.DetectedPhenotype, 0)
	processed := make(map[string]bool)

	for phenotypeID, phenotype := range phenotypeMap {
		if processed[phenotypeID] {
			continue
		}

		// Check for mutually exclusive phenotypes
		if exclusions, hasExclusions := mutuallyExclusive[phenotypeID]; hasExclusions {
			// Find the highest confidence among mutually exclusive phenotypes
			highestConfidence := phenotype.Confidence
			selectedPhenotype := phenotype
			
			for _, exclusiveID := range exclusions {
				if exclusive, exists := phenotypeMap[exclusiveID]; exists && !processed[exclusiveID] {
					if exclusive.Confidence > highestConfidence {
						highestConfidence = exclusive.Confidence
						selectedPhenotype = exclusive
					}
					processed[exclusiveID] = true
				}
			}
			
			result = append(result, selectedPhenotype)
			processed[selectedPhenotype.PhenotypeID] = true
		} else {
			result = append(result, phenotype)
			processed[phenotypeID] = true
		}
	}

	return result
}

// Helper methods for caching

func (p *PhenotypeEngine) getCachedPhenotypes(ctx context.Context, key string) []models.DetectedPhenotype {
	// Implementation would retrieve from Redis cache
	return nil // Simplified for now
}

func (p *PhenotypeEngine) cachePhenotypes(ctx context.Context, key string, phenotypes []models.DetectedPhenotype) error {
	// Implementation would store to Redis cache
	return nil // Simplified for now
}

func (p *PhenotypeEngine) loadActivePhenotypeDefinitions(ctx context.Context) ([]models.PhenotypeDefinition, error) {
	var definitions []models.PhenotypeDefinition
	
	cursor, err := p.db.Collection("phenotype_definitions").Find(ctx, bson.M{"status": "active"})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	if err = cursor.All(ctx, &definitions); err != nil {
		return nil, err
	}

	return definitions, nil
}

// convertEvidence converts engine evidence format to internal evidence format
func (p *PhenotypeEngine) convertEvidence(engineEvidence []engines.EvidenceItem) []map[string]interface{} {
	evidence := make([]map[string]interface{}, len(engineEvidence))
	
	for i, item := range engineEvidence {
		evidence[i] = map[string]interface{}{
			"type":        item.Type,
			"description": item.Description,
			"value":       item.Value,
			"source":      item.Source,
			"timestamp":   item.Timestamp,
			"confidence":  item.Confidence,
		}
	}
	
	return evidence
}

// ValidateAllPhenotypes validates all loaded phenotypes
func (p *PhenotypeEngine) ValidateAllPhenotypes() ([]loaders.ValidationResult, error) {
	return p.phenotypeLoader.ValidateAllPhenotypes(p.multiEngine), nil
}

// GetEngineStats returns statistics about the phenotype engines
func (p *PhenotypeEngine) GetEngineStats() map[string]interface{} {
	stats := p.multiEngine.GetEngineStats()
	phenotypeStats := p.phenotypeLoader.GetPhenotypeStats()
	
	return map[string]interface{}{
		"engine_stats":    stats,
		"phenotype_stats": phenotypeStats,
		"confidence_threshold": p.config.ConfidenceThreshold,
	}
}

// ReloadPhenotypes reloads all phenotype definitions from files
func (p *PhenotypeEngine) ReloadPhenotypes() error {
	_, err := p.phenotypeLoader.ReloadPhenotypes()
	if err != nil {
		return fmt.Errorf("failed to reload phenotypes: %w", err)
	}
	
	p.logger.Info("Phenotype definitions reloaded successfully")
	return nil
}