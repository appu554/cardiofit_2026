package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"kb-clinical-context/internal/cache"
	"kb-clinical-context/internal/config"
	"kb-clinical-context/internal/database"
	"kb-clinical-context/internal/metrics"
	"kb-clinical-context/internal/models"
	"kb-clinical-context/internal/loaders"
)

type ContextService struct {
	db              *database.Database
	cache           *cache.MultiTierCache
	metrics         *metrics.Collector
	config          *config.Config
	phenotypeEngine *PhenotypeEngine
}

func NewContextService(db *database.Database, cache *cache.MultiTierCache, metrics *metrics.Collector, cfg *config.Config, phenotypeDir string) (*ContextService, error) {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}
	
	// Initialize phenotype engine with CEL support
	// For now, we'll create a basic phenotype engine without Redis dependency
	phenotypeEngine, err := NewPhenotypeEngine(db.DB, nil, logger, phenotypeDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize phenotype engine: %w", err)
	}
	
	return &ContextService{
		db:              db,
		cache:           cache,
		metrics:         metrics,
		config:          cfg,
		phenotypeEngine: phenotypeEngine,
	}, nil
}

// BuildContext creates a comprehensive clinical context for a patient
func (s *ContextService) BuildContext(request models.BuildContextRequest) (*models.BuildContextResponse, error) {
	start := time.Now()

	// Check cache first
	cachedData, err := s.cache.GetPatientContext(request.PatientID)
	if err == nil && cachedData != nil {
		var cachedContext models.PatientContext
		if json.Unmarshal(cachedData, &cachedContext) == nil {
			// Check if cached context is still fresh (< 30 minutes)
			if time.Since(cachedContext.Timestamp) < 30*time.Minute {
				s.metrics.RecordCacheHit("patient_context")
				s.metrics.RecordContextBuild(true)
				
				return &models.BuildContextResponse{
					Context:     cachedContext,
					Phenotypes:  s.extractPhenotypeIDs(cachedContext.DetectedPhenotypes),
					RiskScores:  s.convertToFloat64Map(cachedContext.RiskFactors),
					CacheHit:    true,
					ProcessedAt: time.Now(),
				}, nil
			}
		}
	}
	s.metrics.RecordCacheMiss("patient_context")

	// Build new context
	context, err := s.buildPatientContext(request)
	if err != nil {
		s.metrics.RecordContextBuild(false)
		return nil, fmt.Errorf("failed to build patient context: %w", err)
	}

	// Detect phenotypes
	phenotypes, err := s.detectPhenotypes(context)
	if err != nil {
		log.Printf("Warning: phenotype detection failed: %v", err)
	} else {
		context.DetectedPhenotypes = phenotypes
	}

	// ARCHITECTURE NOTE (CTO/CMO Directive):
	// Risk scores are NOT calculated here. KB-2A is data-only assembly.
	// Risk calculations are delegated to KB-8 via KnowledgeSnapshotBuilder.
	// The RiskFactors field remains empty - it will be populated by:
	// 1. KnowledgeSnapshotBuilder → KB-8 (Calculators)
	// 2. KB-2B Intelligence Adapter (enrichment layer)
	context.RiskFactors = make(map[string]interface{})

	// Store in database
	if err := s.storePatientContext(context); err != nil {
		log.Printf("Warning: failed to store context in database: %v", err)
	}

	// Cache the result
	if err := s.cache.CachePatientContext(request.PatientID, context); err != nil {
		log.Printf("Warning: failed to cache context: %v", err)
	}

	s.metrics.RecordContextBuild(true)
	s.metrics.RecordContextBuildDuration(len(phenotypes), time.Since(start))

	return &models.BuildContextResponse{
		Context:     *context,
		Phenotypes:  s.extractPhenotypeIDs(phenotypes),
		RiskScores:  make(map[string]float64), // Empty - KB-2A is data-only, KB-8 calculates risks
		CacheHit:    false,
		ProcessedAt: time.Now(),
	}, nil
}

// DetectPhenotypes performs phenotype detection for a patient
func (s *ContextService) DetectPhenotypes(request models.PhenotypeDetectionRequest) (*models.PhenotypeDetectionResponse, error) {
	start := time.Now()

	// Get phenotype definitions
	phenotypeDefinitions, err := s.getActivePhenotypeDefinitions(request.PhenotypeIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get phenotype definitions: %w", err)
	}

	// Detect phenotypes
	var detectedPhenotypes []models.DetectedPhenotype

	for _, definition := range phenotypeDefinitions {
		confidence, evidence := s.evaluatePhenotype(definition, request.PatientData)
		if confidence > 0.5 { // Threshold for detection
			detectedPhenotype := models.DetectedPhenotype{
				PhenotypeID:        definition.PhenotypeID,
				Confidence:         confidence,
				DetectedAt:         time.Now(),
				SupportingEvidence: evidence,
			}
			detectedPhenotypes = append(detectedPhenotypes, detectedPhenotype)
			s.metrics.RecordPhenotypeDetection(definition.PhenotypeID)
		}
	}

	s.metrics.RecordPhenotypeDetectionDuration(len(phenotypeDefinitions), time.Since(start))

	return &models.PhenotypeDetectionResponse{
		PatientID:          request.PatientID,
		DetectedPhenotypes: detectedPhenotypes,
		TotalPhenotypes:    len(detectedPhenotypes),
		ProcessingTime:     time.Since(start).Milliseconds(),
		Timestamp:          time.Now(),
	}, nil
}

// AssessRisk performs risk assessment for a patient.
//
// ARCHITECTURE NOTE (CTO/CMO Directive):
// Risk assessment has been DELEGATED to KB-8 Calculator Service.
// This endpoint now returns empty scores to indicate that callers should:
// 1. Use KnowledgeSnapshotBuilder → KB-8 for clinical calculators
// 2. Access pre-computed scores from KnowledgeSnapshot.Calculators
//
// KB-8 provides validated calculators:
// - ASCVD 10-year risk (Pooled Cohort Equations)
// - eGFR (CKD-EPI 2021)
// - CHA2DS2-VASc (stroke risk)
// - HAS-BLED (bleeding risk)
// - SOFA/qSOFA (sepsis)
// - Child-Pugh/MELD (liver function)
func (s *ContextService) AssessRisk(request models.RiskAssessmentRequest) (*models.RiskAssessmentResponse, error) {
	// DEPRECATED: Risk calculations delegated to KB-8 via KnowledgeSnapshotBuilder.
	// Return empty response with delegation notice.
	log.Printf("AssessRisk called for patient %s - NOTE: Risk calculations delegated to KB-8", request.PatientID)

	return &models.RiskAssessmentResponse{
		PatientID:           request.PatientID,
		RiskScores:          make(map[string]float64),   // Empty - use KB-8
		RiskFactors:         make(map[string]interface{}), // Empty - use KB-8
		Recommendations:     []string{"Risk assessment delegated to KB-8 Calculator Service via KnowledgeSnapshotBuilder"},
		ConfidenceScore:     0.0, // Not computed
		AssessmentTimestamp: time.Now(),
	}, nil
}

// IdentifyCareGaps identifies care gaps for a patient
func (s *ContextService) IdentifyCareGaps(request models.CareGapsRequest) (*models.CareGapsResponse, error) {
	// This would implement care gap identification logic
	// For now, return a placeholder response
	var careGaps []models.CareGap

	// Example care gaps based on common scenarios
	if !request.IncludeResolved {
		careGaps = append(careGaps, models.CareGap{
			ID:          uuid.New(),
			Type:        "preventive_care",
			Description: "Annual wellness visit due",
			Priority:    "medium",
			DueDays:     30,
			Actions:     []string{"Schedule annual physical", "Update vaccinations"},
		})
	}

	for _, gap := range careGaps {
		s.metrics.RecordCareGap(gap.Type)
	}

	return &models.CareGapsResponse{
		PatientID:  request.PatientID,
		CareGaps:   careGaps,
		TotalGaps:  len(careGaps),
		Priority:   s.determinePriority(careGaps),
		NextReview: time.Now().Add(30 * 24 * time.Hour),
	}, nil
}

// Helper methods

func (s *ContextService) buildPatientContext(request models.BuildContextRequest) (*models.PatientContext, error) {
	contextID := uuid.New().String()

	// Extract patient data
	patient := request.Patient
	
	// Build demographics
	demographics := models.Demographics{}
	if demo, ok := patient["demographics"].(map[string]interface{}); ok {
		if age, ok := demo["age_years"].(float64); ok {
			demographics.AgeYears = int(age)
		}
		if sex, ok := demo["sex"].(string); ok {
			demographics.Sex = sex
		}
		if race, ok := demo["race"].(string); ok {
			demographics.Race = race
		}
		if ethnicity, ok := demo["ethnicity"].(string); ok {
			demographics.Ethnicity = ethnicity
		}
	}

	// Build conditions
	var conditions []models.Condition
	if conds, ok := patient["active_conditions"].([]interface{}); ok {
		for _, c := range conds {
			if condMap, ok := c.(map[string]interface{}); ok {
				condition := models.Condition{}
				if code, ok := condMap["code"].(string); ok {
					condition.Code = code
				}
				if system, ok := condMap["system"].(string); ok {
					condition.System = system
				}
				if name, ok := condMap["name"].(string); ok {
					condition.Name = name
				}
				if severity, ok := condMap["severity"].(string); ok {
					condition.Severity = severity
				}
				condition.OnsetDate = time.Now() // Default to now
				conditions = append(conditions, condition)
			}
		}
	}

	// Build lab results
	var labResults []models.LabResult
	if labs, ok := patient["recent_labs"].([]interface{}); ok {
		for _, l := range labs {
			if labMap, ok := l.(map[string]interface{}); ok {
				lab := models.LabResult{}
				if loinc, ok := labMap["loinc_code"].(string); ok {
					lab.LOINCCode = loinc
				}
				if value, ok := labMap["value"].(float64); ok {
					lab.Value = value
				}
				if unit, ok := labMap["unit"].(string); ok {
					lab.Unit = unit
				}
				if flag, ok := labMap["abnormal_flag"].(string); ok {
					lab.AbnormalFlag = flag
				}
				lab.ResultDate = time.Now() // Default to now
				labResults = append(labResults, lab)
			}
		}
	}

	// Build medications
	var medications []models.Medication
	if meds, ok := patient["current_medications"].([]interface{}); ok {
		for _, m := range meds {
			if medMap, ok := m.(map[string]interface{}); ok {
				med := models.Medication{}
				if rxnorm, ok := medMap["rxnorm_code"].(string); ok {
					med.RxNormCode = rxnorm
				}
				if name, ok := medMap["name"].(string); ok {
					med.Name = name
				}
				if dose, ok := medMap["dose"].(string); ok {
					med.Dose = dose
				}
				if freq, ok := medMap["frequency"].(string); ok {
					med.Frequency = freq
				}
				med.StartDate = time.Now() // Default to now
				medications = append(medications, med)
			}
		}
	}

	context := &models.PatientContext{
		PatientID:          request.PatientID,
		ContextID:          contextID,
		Timestamp:          time.Now(),
		Demographics:       demographics,
		ActiveConditions:   conditions,
		RecentLabs:         labResults,
		CurrentMeds:        medications,
		DetectedPhenotypes: []models.DetectedPhenotype{},
		RiskFactors:        make(map[string]interface{}),
		CareGaps:          []string{},
		TTL:               time.Now().Add(24 * time.Hour), // Expire after 24 hours
	}

	return context, nil
}

func (s *ContextService) detectPhenotypes(patientCtx *models.PatientContext) ([]models.DetectedPhenotype, error) {
	start := time.Now()

	// Use the enhanced phenotype engine for detection
	detectedPhenotypes, err := s.phenotypeEngine.DetectPhenotypes(context.Background(), *patientCtx)
	if err != nil {
		return nil, fmt.Errorf("phenotype engine detection failed: %w", err)
	}

	// Record metrics for detected phenotypes
	for _, detected := range detectedPhenotypes {
		s.metrics.RecordPhenotypeDetection(detected.PhenotypeID)
	}

	s.metrics.RecordPhenotypeDetectionDuration(len(detectedPhenotypes), time.Since(start))

	return detectedPhenotypes, nil
}

func (s *ContextService) getActivePhenotypeDefinitions(phenotypeIDs []string) ([]models.PhenotypeDefinition, error) {
	start := time.Now()

	ctx := context.Background()
	collection := s.db.PhenotypeDefinitions()

	// Build query filter
	filter := bson.M{"status": "active"}
	if len(phenotypeIDs) > 0 {
		filter["phenotype_id"] = bson.M{"$in": phenotypeIDs}
	}

	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		s.metrics.RecordMongoOperation("find", "phenotype_definitions", false, time.Since(start))
		return nil, err
	}
	defer cursor.Close(ctx)

	var definitions []models.PhenotypeDefinition
	if err = cursor.All(ctx, &definitions); err != nil {
		s.metrics.RecordMongoOperation("find", "phenotype_definitions", false, time.Since(start))
		return nil, err
	}

	s.metrics.RecordMongoOperation("find", "phenotype_definitions", true, time.Since(start))
	return definitions, nil
}

func (s *ContextService) evaluatePhenotype(definition models.PhenotypeDefinition, patientData map[string]interface{}) (float64, []map[string]interface{}) {
	var confidence float64 = 0.0
	var evidence []map[string]interface{}

	// Evaluate required conditions
	conditionScore := s.evaluateConditions(definition.Criteria.RequiredConditions, patientData)
	confidence += conditionScore * 0.4

	// Evaluate required labs
	labScore := s.evaluateLabs(definition.Criteria.RequiredLabs, patientData)
	confidence += labScore * 0.3

	// Evaluate required medications
	medScore := s.evaluateMedications(definition.Criteria.RequiredMeds, patientData)
	confidence += medScore * 0.3

	// Check exclusion criteria
	if s.hasExclusionCriteria(definition.Criteria.ExclusionCriteria, patientData) {
		confidence = 0.0
	}

	// Build evidence
	if confidence > 0.5 {
		evidence = append(evidence, map[string]interface{}{
			"type": "phenotype_detected",
			"phenotype_id": definition.PhenotypeID,
			"confidence": confidence,
			"timestamp": time.Now(),
		})
	}

	return confidence, evidence
}

func (s *ContextService) evaluateConditions(conditions []models.ConditionCriteria, patientData map[string]interface{}) float64 {
	if len(conditions) == 0 {
		return 1.0
	}

	// Get patient conditions
	activeConditions, ok := patientData["active_conditions"].([]models.Condition)
	if !ok {
		return 0.0
	}

	// Build condition code set
	conditionCodes := make(map[string]bool)
	for _, condition := range activeConditions {
		conditionCodes[condition.Code] = true
	}

	// Evaluate each condition criteria
	var score float64
	for _, criteria := range conditions {
		// Check if any required codes are present
		hasCode := false
		for _, code := range criteria.Codes {
			if conditionCodes[code] {
				hasCode = true
				break
			}
		}
		if hasCode {
			score += 1.0
		}
	}

	return score / float64(len(conditions))
}

func (s *ContextService) evaluateLabs(labs []models.LabCriteria, patientData map[string]interface{}) float64 {
	if len(labs) == 0 {
		return 1.0
	}

	// Get patient labs
	recentLabs, ok := patientData["recent_labs"].([]models.LabResult)
	if !ok {
		return 0.0
	}

	// Build lab results map
	labResults := make(map[string]float64)
	for _, lab := range recentLabs {
		labResults[lab.LOINCCode] = lab.Value
	}

	// Evaluate each lab criteria
	var score float64
	for _, criteria := range labs {
		if value, exists := labResults[criteria.LOINCCode]; exists {
			if s.evaluateLabCondition(value, criteria.Operator, criteria.Value) {
				score += 1.0
			}
		}
	}

	return score / float64(len(labs))
}

func (s *ContextService) evaluateLabCondition(value float64, operator string, threshold float64) bool {
	switch operator {
	case ">", "gt":
		return value > threshold
	case ">=", "gte":
		return value >= threshold
	case "<", "lt":
		return value < threshold
	case "<=", "lte":
		return value <= threshold
	case "=", "eq":
		return value == threshold
	case "!=", "ne":
		return value != threshold
	default:
		return false
	}
}

func (s *ContextService) evaluateMedications(medications []models.MedicationCriteria, patientData map[string]interface{}) float64 {
	if len(medications) == 0 {
		return 1.0
	}

	// Get patient medications
	currentMeds, ok := patientData["current_medications"].([]models.Medication)
	if !ok {
		return 0.0
	}

	// Build medication code set
	medCodes := make(map[string]bool)
	for _, med := range currentMeds {
		medCodes[med.RxNormCode] = true
	}

	// Evaluate each medication criteria
	var score float64
	for _, criteria := range medications {
		// Check if any required codes are present
		hasCode := false
		for _, code := range criteria.RxNormCodes {
			if medCodes[code] {
				hasCode = true
				break
			}
		}
		if hasCode {
			score += 1.0
		}
	}

	return score / float64(len(medications))
}

func (s *ContextService) hasExclusionCriteria(exclusions []string, patientData map[string]interface{}) bool {
	// Simple exclusion check - would be more sophisticated in practice
	activeConditions, ok := patientData["active_conditions"].([]models.Condition)
	if !ok {
		return false
	}

	for _, condition := range activeConditions {
		for _, exclusion := range exclusions {
			if strings.Contains(strings.ToLower(condition.Name), strings.ToLower(exclusion)) {
				return true
			}
		}
	}
	return false
}

// ============================================================================
// REMOVED: INLINE RISK CALCULATORS (CTO/CMO Directive)
// ============================================================================
//
// The following functions have been REMOVED as tier-boundary violations:
//
//   - calculateRiskScores()
//   - calculateRiskScore()
//   - calculateCardiovascularRisk()   ❌ Used strings.Contains("diabetes")
//   - calculateFallRisk()             ❌ Hardcoded age/medication weights
//   - calculateReadmissionRisk()      ❌ Hardcoded condition count weights
//   - calculateADEPisk()              ❌ Hardcoded renal/medication checks
//
// CORRECT ARCHITECTURE:
//
//   KB-2A (this service): Data assembly ONLY
//     └── Parses FHIR input
//     └── Extracts demographics, conditions, labs, medications
//     └── Returns PatientContext with EMPTY RiskFactors
//
//   KnowledgeSnapshotBuilder → KB-8: Clinical calculators
//     └── ASCVD 10-year risk (Pooled Cohort Equations)
//     └── eGFR (CKD-EPI 2021)
//     └── CHA2DS2-VASc, HAS-BLED (anticoagulation)
//     └── SOFA, qSOFA (sepsis)
//     └── Child-Pugh, MELD (liver function)
//
//   KnowledgeSnapshotBuilder → KB-7: Terminology
//     └── ValueSetMemberships["is_diabetic"] → true/false
//     └── Proper SNOMED/ICD-10 code membership checks
//
//   KB-2B Intelligence Adapter: Enriches PatientContext
//     └── Uses pre-computed KB-8 calculator results
//     └── Uses pre-computed KB-7 terminology results
//     └── Builds CQLExportBundle for CQL engine
//
// ============================================================================

func (s *ContextService) storePatientContext(patientCtx *models.PatientContext) error {
	start := time.Now()

	ctx := context.Background()
	collection := s.db.PatientContexts()

	_, err := collection.InsertOne(ctx, patientCtx)
	
	s.metrics.RecordMongoOperation("insert", "patient_contexts", err == nil, time.Since(start))
	
	return err
}

// NOTE: contextToMap() was removed - was only used by deleted risk calculators

func (s *ContextService) extractPhenotypeIDs(phenotypes []models.DetectedPhenotype) []string {
	var ids []string
	for _, phenotype := range phenotypes {
		ids = append(ids, phenotype.PhenotypeID)
	}
	return ids
}

// NOTE: calculateConfidenceScore() was removed - was only used by deleted risk calculators

func (s *ContextService) determinePriority(careGaps []models.CareGap) string {
	if len(careGaps) == 0 {
		return "low"
	}

	// Check for high priority gaps
	for _, gap := range careGaps {
		if gap.Priority == "high" || gap.DueDays < 7 {
			return "high"
		}
	}

	// Check for medium priority gaps
	for _, gap := range careGaps {
		if gap.Priority == "medium" || gap.DueDays < 30 {
			return "medium"
		}
	}

	return "low"
}

// ValidateAllPhenotypes delegates to the phenotype engine
func (s *ContextService) ValidateAllPhenotypes() ([]loaders.ValidationResult, error) {
	return s.phenotypeEngine.ValidateAllPhenotypes()
}

// GetEngineStats delegates to the phenotype engine
func (s *ContextService) GetEngineStats() map[string]interface{} {
	return s.phenotypeEngine.GetEngineStats()
}

// ReloadPhenotypes delegates to the phenotype engine
func (s *ContextService) ReloadPhenotypes() error {
	return s.phenotypeEngine.ReloadPhenotypes()
}

// GetPhenotypeDefinitions retrieves phenotype definitions from MongoDB with pagination and filtering
func (s *ContextService) GetPhenotypeDefinitions(domain, status string, limit, offset int) ([]map[string]interface{}, int, error) {
	start := time.Now()

	ctx := context.Background()
	collection := s.db.PhenotypeDefinitions()

	// Build query filter
	filter := bson.M{}
	if domain != "" {
		filter["category"] = domain
	}
	if status != "" {
		filter["status"] = status
	}

	// Get total count for pagination
	totalCount, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		s.metrics.RecordMongoOperation("count", "phenotype_definitions", false, time.Since(start))
		return nil, 0, fmt.Errorf("failed to count phenotype definitions: %w", err)
	}

	// Execute query with pagination
	cursor, err := collection.Find(ctx, filter, &options.FindOptions{
		Limit: int64Ptr(int64(limit)),
		Skip:  int64Ptr(int64(offset)),
		Sort:  bson.M{"phenotype_id": 1}, // Sort by ID for consistent pagination
	})
	if err != nil {
		s.metrics.RecordMongoOperation("find", "phenotype_definitions", false, time.Since(start))
		return nil, 0, fmt.Errorf("failed to query phenotype definitions: %w", err)
	}
	defer cursor.Close(ctx)

	// Convert results to API-friendly format
	var results []map[string]interface{}
	for cursor.Next(ctx) {
		var doc bson.M
		if err := cursor.Decode(&doc); err != nil {
			continue // Skip invalid documents
		}

		// Format document for API response
		result := map[string]interface{}{
			"phenotype_id": doc["phenotype_id"],
			"name":         doc["name"],
			"description":  doc["description"],
			"category":     doc["category"],
			"severity":     doc["severity"],
			"status":       doc["status"],
			"version":      doc["version"],
			"created_at":   doc["created_at"],
			"updated_at":   doc["updated_at"],
		}

		// Include ICD-10 and SNOMED codes
		if icd10Codes, ok := doc["icd10_codes"]; ok {
			result["icd10_codes"] = icd10Codes
		}
		if snomedCodes, ok := doc["snomed_codes"]; ok {
			result["snomed_codes"] = snomedCodes
		}

		// Include algorithm information
		if algorithm, ok := doc["algorithm"].(bson.M); ok {
			result["algorithm_type"] = algorithm["type"]
			if thresholds, ok := algorithm["thresholds"].(bson.M); ok {
				result["match_threshold"] = thresholds["match_threshold"]
				result["confidence_threshold"] = thresholds["confidence_threshold"]
			}
		}

		// Include validation data
		if validation, ok := doc["validation_data"].(bson.M); ok {
			result["validation"] = map[string]interface{}{
				"sensitivity":   validation["sensitivity"],
				"specificity":   validation["specificity"],
				"ppv":          validation["ppv"],
				"npv":          validation["npv"],
				"f1_score":     validation["f1_score"],
				"auc":          validation["auc"],
				"validated_at": validation["validated_at"],
			}
		}

		results = append(results, result)
	}

	if err := cursor.Err(); err != nil {
		s.metrics.RecordMongoOperation("find", "phenotype_definitions", false, time.Since(start))
		return nil, 0, fmt.Errorf("error iterating phenotype definitions: %w", err)
	}

	s.metrics.RecordMongoOperation("find", "phenotype_definitions", true, time.Since(start))
	return results, int(totalCount), nil
}

// convertToFloat64Map converts map[string]interface{} to map[string]float64
func (s *ContextService) convertToFloat64Map(input map[string]interface{}) map[string]float64 {
	result := make(map[string]float64)
	for key, value := range input {
		if floatVal, ok := value.(float64); ok {
			result[key] = floatVal
		} else if intVal, ok := value.(int); ok {
			result[key] = float64(intVal)
		} else {
			// Skip non-numeric values
			continue
		}
	}
	return result
}

// Helper function for int64 pointer
func int64Ptr(i int64) *int64 {
	return &i
}