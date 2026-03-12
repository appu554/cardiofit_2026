package services

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"kb-drug-interactions/internal/cache"
	"kb-drug-interactions/internal/config"
	"kb-drug-interactions/internal/database"
	"kb-drug-interactions/internal/metrics"
	"kb-drug-interactions/internal/models"
)

// EnhancedInteractionMatrixService implements high-performance drug interaction lookups
// with hot/warm caching strategy and dataset versioning
type EnhancedInteractionMatrixService struct {
	db                    *database.Database
	cache                 *cache.CacheClient
	config               *config.Config
	metrics              *metrics.Collector
	
	// Hot cache: In-memory map for top 50k interactions
	hotCache             map[string]*models.EnhancedInteractionResult
	hotCacheMutex        sync.RWMutex
	hotCacheStats        *cache.CacheStatistics
	
	// Current dataset version for cache invalidation
	currentDatasetVersion string
	datasetMutex         sync.RWMutex
	
	// Performance tracking
	lookupStats          *metrics.PerformanceTracker
	lastRefresh          time.Time
}

// NewEnhancedInteractionMatrixService creates a new enhanced interaction matrix service
func NewEnhancedInteractionMatrixService(
	db *database.Database,
	cache *cache.CacheClient,
	config *config.Config,
	metrics *metrics.Collector,
) *EnhancedInteractionMatrixService {
	matrix := &EnhancedInteractionMatrixService{
		db:                    db,
		cache:                 cache,
		config:               config,
		metrics:              metrics,
		hotCache:             make(map[string]*models.EnhancedInteractionResult),
		currentDatasetVersion: "",
		lastRefresh:          time.Now(),
	}
	
	// Initialize the matrix asynchronously
	go func() {
		ctx := context.Background()
		if err := matrix.LoadMatrix(ctx); err != nil {
			fmt.Printf("Failed to initialize interaction matrix: %v\n", err)
		}
	}()
	
	return matrix
}

// LoadMatrix loads the interaction matrix from database into hot cache
func (eim *EnhancedInteractionMatrixService) LoadMatrix(ctx context.Context) error {
	startTime := time.Now()
	defer func() {
		loadTime := time.Since(startTime)
		eim.metrics.RecordMatrixLoad("full_reload", loadTime)
	}()

	// Get current dataset version
	currentVersion, err := eim.getCurrentDatasetVersion(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current dataset version: %w", err)
	}

	// Check if we need to reload (version changed)
	eim.datasetMutex.RLock()
	needsReload := eim.currentDatasetVersion != currentVersion
	eim.datasetMutex.RUnlock()

	if !needsReload {
		return nil
	}

	// Load interaction matrix from materialized view
	var matrixData []models.DDIInteractionMatrix
	err = eim.db.DB.WithContext(ctx).
		Table("ddi_interaction_matrix").
		Where("dataset_version = ?", currentVersion).
		Limit(eim.config.MaxMatrixSize).
		Order("confidence DESC").  // Load highest confidence interactions first
		Find(&matrixData).Error
	
	if err != nil {
		return fmt.Errorf("failed to load interaction matrix: %w", err)
	}

	// Build hot cache
	newHotCache := make(map[string]*models.EnhancedInteractionResult)
	for _, interaction := range matrixData {
		key := eim.buildInteractionKey(interaction.Drug1Code, interaction.Drug2Code)
		
		result := &models.EnhancedInteractionResult{
			InteractionID:      fmt.Sprintf("%s_%s_%s", interaction.Drug1Code, interaction.Drug2Code, interaction.DatasetVersion),
			Severity:           interaction.Severity,
			Mechanism:          interaction.Mechanism,
			ClinicalEffects:    interaction.ClinicalEffects,
			ManagementStrategy: interaction.ManagementStrategy,
			Evidence:           interaction.Evidence,
			Confidence:         interaction.Confidence,
			RouteSpecific:      len(interaction.RouteRestriction) > 0,
		}

		// Set drug information
		result.Drug1 = models.DrugInfo{Code: interaction.Drug1Code}
		result.Drug2 = models.DrugInfo{Code: interaction.Drug2Code}

		// Add PGX markers if present
		if interaction.PGXMarkers != nil {
			result.PGXApplicable = true
		}

		newHotCache[key] = result
	}

	// Atomic update of hot cache and dataset version
	eim.hotCacheMutex.Lock()
	eim.hotCache = newHotCache
	eim.hotCacheMutex.Unlock()

	eim.datasetMutex.Lock()
	eim.currentDatasetVersion = currentVersion
	eim.lastRefresh = time.Now()
	eim.datasetMutex.Unlock()

	fmt.Printf("Loaded %d interactions into hot cache for dataset version %s\n",
		len(newHotCache), currentVersion)

	return nil
}

// FastLookup performs ultra-fast pairwise interaction lookup
func (eim *EnhancedInteractionMatrixService) FastLookup(
	drugACode, drugBCode string,
) (*models.EnhancedInteractionResult, bool, time.Duration) {
	startTime := time.Now()

	// Normalize drug pair order for consistent lookup
	key := eim.buildInteractionKey(drugACode, drugBCode)

	// Check hot cache first
	eim.hotCacheMutex.RLock()
	if interaction, exists := eim.hotCache[key]; exists {
		eim.hotCacheMutex.RUnlock()
		eim.metrics.RecordCacheHit("hot_cache", "interaction_lookup")
		return interaction, true, time.Since(startTime)
	}
	eim.hotCacheMutex.RUnlock()
	eim.metrics.RecordCacheMiss("hot_cache", "interaction_lookup")

	// Check warm cache (Redis) - commented out until cache interface is implemented
	/*
	warmCacheKey := fmt.Sprintf("ddi:%s:%s", eim.currentDatasetVersion, key)
	var cachedResult models.EnhancedInteractionResult
	if err := eim.cache.Get(warmCacheKey, &cachedResult); err == nil {
		eim.metrics.RecordCacheHit("warm_cache", "interaction_lookup")

		// Promote to hot cache if space available
		eim.promoteToHotCache(key, &cachedResult)

		return &cachedResult, true, time.Since(startTime)
	}
	eim.metrics.RecordCacheMiss("warm_cache", "interaction_lookup")
	*/

	// Database lookup (materialized view)
	ctx := context.Background()
	var matrixRow models.DDIInteractionMatrix
	drug1, drug2 := eim.normalizeOrder(drugACode, drugBCode)
	err := eim.db.DB.WithContext(ctx).
		Table("ddi_interaction_matrix").
		Where("dataset_version = ? AND drug1_code = ? AND drug2_code = ?",
			eim.currentDatasetVersion, drug1, drug2).
		First(&matrixRow).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, false, time.Since(startTime)
		}
		return nil, false, time.Since(startTime)
	}

	// Convert to result format
	result := eim.convertMatrixRowToResult(&matrixRow)

	// Cache in warm cache - commented out until cache interface is implemented
	// eim.cacheInteractionResult(warmCacheKey, result)

	return result, true, time.Since(startTime)
}

// CheckInteractionsEnhanced performs comprehensive interaction checking with PGx and class rules
func (eim *EnhancedInteractionMatrixService) CheckInteractionsEnhanced(
	ctx context.Context,
	request *models.EnhancedInteractionCheckRequest,
) (*models.EnhancedInteractionCheckResponse, error) {
	startTime := time.Now()
	defer func() {
		checkTime := time.Since(startTime)
		eim.metrics.RecordInteractionCheck("enhanced", checkTime)
	}()

	// Validate request
	if len(request.DrugCodes) < 2 {
		return nil, fmt.Errorf("minimum 2 drug codes required")
	}

	// Use current dataset version if not specified
	datasetVersion := request.DatasetVersion
	if datasetVersion == "" {
		datasetVersion = eim.currentDatasetVersion
	}

	var allInteractions []models.EnhancedInteractionResult

	// 1. Check pairwise drug-drug interactions
	pairwiseInteractions, err := eim.checkPairwiseInteractions(ctx, request.DrugCodes, datasetVersion, request.PatientContext)
	if err != nil {
		return nil, fmt.Errorf("pairwise interaction check failed: %w", err)
	}
	allInteractions = append(allInteractions, pairwiseInteractions...)

	// 2. Check pharmacogenomic interactions if patient context provided
	if request.PatientContext != nil && len(request.PatientContext.PGX) > 0 {
		pgxInteractions, err := eim.checkPGXInteractions(ctx, request.DrugCodes, datasetVersion, request.PatientContext)
		if err != nil {
			return nil, fmt.Errorf("PGx interaction check failed: %w", err)
		}
		allInteractions = append(allInteractions, pgxInteractions...)
	}

	// 3. Check drug class interactions if enabled
	if request.ExpandClasses {
		classInteractions, err := eim.checkClassInteractions(ctx, request.DrugCodes, datasetVersion)
		if err != nil {
			return nil, fmt.Errorf("class interaction check failed: %w", err)
		}
		allInteractions = append(allInteractions, classInteractions...)
	}

	// 4. Check food/alcohol/herbal modifiers if enabled
	if request.IncludeContextuals {
		modifierInteractions, err := eim.checkModifierInteractions(ctx, request.DrugCodes, datasetVersion)
		if err != nil {
			return nil, fmt.Errorf("modifier interaction check failed: %w", err)
		}
		allInteractions = append(allInteractions, modifierInteractions...)
	}

	// Apply severity filtering
	if len(request.SeverityFilter) > 0 {
		allInteractions = eim.filterBySeverity(allInteractions, request.SeverityFilter)
	}

	// Sort by clinical priority
	sort.Slice(allInteractions, func(i, j int) bool {
		return allInteractions[i].GetClinicalPriority() > allInteractions[j].GetClinicalPriority()
	})

	// Build response
	response := &models.EnhancedInteractionCheckResponse{
		TransactionID:   request.TransactionID,
		DatasetVersion:  datasetVersion,
		Interactions:    allInteractions,
		CheckTimestamp:  time.Now().UTC(),
		Summary:         eim.buildEnhancedSummary(allInteractions),
		Recommendations: eim.generateClinicalRecommendations(allInteractions),
	}

	// Add conflict trail for audit purposes
	response.ConflictTrail = &models.ConflictTrail{
		SynthesizedFromVersion: datasetVersion,
		HarmonizedAt:           eim.lastRefresh,
		HarmonizerVersion:      "2.1.0",
	}

	// Calculate risk score
	response.RiskScore = response.CalculateRiskScore()

	// Add alternatives if requested
	if request.IncludeAlternatives {
		response.AlternativeDrugs = eim.findAlternativeDrugs(ctx, allInteractions)
	}

	// Add monitoring plan if requested
	if request.IncludeMonitoring {
		enhancedPlan := eim.buildMonitoringPlan(allInteractions)
		// Convert to base MonitoringRecommendation type
		var basePlan []models.MonitoringRecommendation
		for _, rec := range enhancedPlan {
			basePlan = append(basePlan, models.MonitoringRecommendation{
				Parameter:    rec.Parameter,
				Instructions: rec.Instructions,
				Frequency:    rec.Frequency,
				Duration:     rec.Duration,
				TargetRange:  rec.TargetRange,
			})
		}
		response.MonitoringPlan = basePlan
	}

	return response, nil
}

// Private helper methods

func (eim *EnhancedInteractionMatrixService) checkPairwiseInteractions(
	ctx context.Context, 
	drugCodes []string, 
	datasetVersion string,
	patientContext *models.PatientContextData,
) ([]models.EnhancedInteractionResult, error) {
	var interactions []models.EnhancedInteractionResult
	
	// Check all pairs
	for i := 0; i < len(drugCodes); i++ {
		for j := i + 1; j < len(drugCodes); j++ {
			result, found, _ := eim.FastLookup(drugCodes[i], drugCodes[j])
			if found {
				// Apply patient context filtering
				if eim.isApplicableToPatient(result, patientContext) {
					interactions = append(interactions, *result)
				}
			}
		}
	}
	
	return interactions, nil
}

func (eim *EnhancedInteractionMatrixService) checkPGXInteractions(
	ctx context.Context,
	drugCodes []string,
	datasetVersion string,
	patientContext *models.PatientContextData,
) ([]models.EnhancedInteractionResult, error) {
	if patientContext == nil || len(patientContext.PGX) == 0 {
		return []models.EnhancedInteractionResult{}, nil
	}

	var pgxRules []models.DDIPharmacogenomicRule
	// Use IN clause instead of ANY() since GORM doesn't properly convert slices to PostgreSQL arrays.
	err := eim.db.DB.WithContext(ctx).
		Where("dataset_version = ? AND active = TRUE AND drug_code IN ?",
			datasetVersion, drugCodes).
		Find(&pgxRules).Error
	
	if err != nil {
		return nil, err
	}

	var interactions []models.EnhancedInteractionResult
	for _, rule := range pgxRules {
		// Check if patient has the relevant genetic marker
		if patientPhenotype, exists := patientContext.PGX[rule.Gene]; exists && patientPhenotype == rule.Phenotype {
			interaction := eim.convertPGXRuleToResult(&rule, drugCodes)
			interactions = append(interactions, interaction)
		}
	}

	return interactions, nil
}

func (eim *EnhancedInteractionMatrixService) checkClassInteractions(
	ctx context.Context,
	drugCodes []string,
	datasetVersion string,
) ([]models.EnhancedInteractionResult, error) {
	// TODO: Implement class-based interaction checking
	// This would involve resolving drug codes to their therapeutic classes
	// and checking class-to-class and class-to-drug rules
	return []models.EnhancedInteractionResult{}, nil
}

func (eim *EnhancedInteractionMatrixService) checkModifierInteractions(
	ctx context.Context,
	drugCodes []string,
	datasetVersion string,
) ([]models.EnhancedInteractionResult, error) {
	var modifiers []models.DDIModifier
	// Use IN clause instead of ANY() since GORM doesn't properly convert slices to PostgreSQL arrays.
	err := eim.db.DB.WithContext(ctx).
		Where("dataset_version = ? AND active = TRUE AND drug_code IN ?",
			datasetVersion, drugCodes).
		Find(&modifiers).Error
	
	if err != nil {
		return nil, err
	}

	var interactions []models.EnhancedInteractionResult
	for _, modifier := range modifiers {
		interaction := eim.convertModifierToResult(&modifier)
		interactions = append(interactions, interaction)
	}

	return interactions, nil
}

func (eim *EnhancedInteractionMatrixService) isApplicableToPatient(
	interaction *models.EnhancedInteractionResult,
	patientContext *models.PatientContextData,
) bool {
	// If no patient context, assume applicable
	if patientContext == nil {
		return true
	}

	// Check PGx applicability (if needed)
	// if interaction.PGXApplicable && len(patientContext.PGX) == 0 {
	// 	return false
	// }

	// TODO: Add more sophisticated patient context filtering
	// - Age band restrictions
	// - Hepatic/renal stage considerations
	// - Comorbidity factors

	return true
}

func (eim *EnhancedInteractionMatrixService) buildEnhancedSummary(
	interactions []models.EnhancedInteractionResult,
) models.EnhancedInteractionSummary {
	summary := models.EnhancedInteractionSummary{
		TotalInteractions: len(interactions),
		SeverityCounts:    make(map[string]int),
		RequiredActions:   []string{},
	}

	if len(interactions) == 0 {
		return summary
	}

	// Count by severity and interaction type
	for _, interaction := range interactions {
		summary.SeverityCounts[string(interaction.Severity)]++
		
		if interaction.Severity == models.SeverityContraindicated {
			summary.ContraindicatedPairs++
		}
		
		if interaction.PGXApplicable {
			summary.PGXInteractions++
		}
	}

	// Determine highest severity
	severityOrder := []string{"contraindicated", "major", "moderate", "minor"}
	for _, severity := range severityOrder {
		if summary.SeverityCounts[severity] > 0 {
			summary.HighestSeverity = severity
			break
		}
	}

	// Generate required actions
	if summary.ContraindicatedPairs > 0 {
		summary.RequiredActions = append(summary.RequiredActions, 
			"IMMEDIATE: Review contraindicated combinations - alternative therapy required")
	}
	if summary.SeverityCounts["major"] > 0 {
		summary.RequiredActions = append(summary.RequiredActions, 
			"HIGH PRIORITY: Major interactions require dose adjustment or enhanced monitoring")
	}
	if summary.PGXInteractions > 0 {
		summary.RequiredActions = append(summary.RequiredActions, 
			"GENETIC: Pharmacogenomic factors affect drug metabolism - verify patient genotype")
	}

	return summary
}

func (eim *EnhancedInteractionMatrixService) generateClinicalRecommendations(
	interactions []models.EnhancedInteractionResult,
) []string {
	if len(interactions) == 0 {
		return []string{"No significant drug interactions detected"}
	}

	var recommendations []string

	// Count by severity
	severityCounts := make(map[models.DDISeverity]int)
	for _, interaction := range interactions {
		severityCounts[interaction.Severity]++
	}

	// Generate severity-based recommendations
	if count := severityCounts[models.SeverityContraindicated]; count > 0 {
		recommendations = append(recommendations, 
			fmt.Sprintf("🚨 CONTRAINDICATED: %d drug combinations are contraindicated. Immediate intervention required.", count))
	}

	if count := severityCounts[models.SeverityMajor]; count > 0 {
		recommendations = append(recommendations, 
			fmt.Sprintf("⚠️ MAJOR: %d major interactions require dose adjustment or enhanced monitoring.", count))
	}

	// Add specific management for top 3 critical interactions
	criticalCount := 0
	for _, interaction := range interactions {
		if criticalCount >= 3 {
			break
		}
		
		if interaction.Severity == models.SeverityContraindicated || interaction.Severity == models.SeverityMajor {
			recommendations = append(recommendations, 
				fmt.Sprintf("• %s + %s: %s", 
					interaction.Drug1.Code, 
					interaction.Drug2.Code, 
					interaction.ManagementStrategy))
			criticalCount++
		}
	}

	return recommendations
}

func (eim *EnhancedInteractionMatrixService) buildMonitoringPlan(
	interactions []models.EnhancedInteractionResult,
) []models.EnhancedMonitoringRecommendation {
	var monitoringPlan []models.EnhancedMonitoringRecommendation

	for _, interaction := range interactions {
		if len(interaction.MonitoringParameters) > 0 {
			for param, details := range interaction.MonitoringParameters {
				if detailsMap, ok := details.(map[string]interface{}); ok {
					recommendation := models.EnhancedMonitoringRecommendation{
						Parameter:    param,
						Instructions: fmt.Sprintf("Monitor for %s interaction: %s + %s", 
							string(interaction.Severity), 
							interaction.Drug1.Code, 
							interaction.Drug2.Code),
						Rationale: interaction.ManagementStrategy,
						Priority:  int(interaction.Severity.GetPriority()),
					}

					if frequency, ok := detailsMap["frequency"].(string); ok {
						recommendation.Frequency = frequency
					}
					if duration, ok := detailsMap["duration"].(string); ok {
						recommendation.Duration = duration
					}
					if targetRange, ok := detailsMap["target_range"].(string); ok {
						recommendation.TargetRange = targetRange
					}
					
					monitoringPlan = append(monitoringPlan, recommendation)
				}
			}
		}
	}

	// Sort by priority (highest first)
	sort.Slice(monitoringPlan, func(i, j int) bool {
		return monitoringPlan[i].Priority > monitoringPlan[j].Priority
	})

	return monitoringPlan
}

func (eim *EnhancedInteractionMatrixService) findAlternativeDrugs(
	ctx context.Context,
	interactions []models.EnhancedInteractionResult,
) map[string]models.DrugAlternatives {
	alternatives := make(map[string]models.DrugAlternatives)
	
	// For each drug involved in high-severity interactions, suggest alternatives
	for _, interaction := range interactions {
		if interaction.Severity == models.SeverityContraindicated || interaction.Severity == models.SeverityMajor {
			// TODO: Implement sophisticated alternative drug discovery
			// This would involve therapeutic equivalence databases and contraindication checking
			
			if len(interaction.AlternativeDrugs) > 0 {
				drugAlts := models.DrugAlternatives{
					Rationale: fmt.Sprintf("Alternatives for %s due to %s interaction", 
						interaction.Drug1.Code, string(interaction.Severity)),
				}
				
				for _, altCode := range interaction.AlternativeDrugs {
					alt := models.AlternativeDrug{
						DrugInfo: models.DrugInfo{Code: altCode},
						Reason:   "Lower interaction risk",
						SafetyScore: func() *decimal.Decimal {
							score := decimal.NewFromFloat(0.85)
							return &score
						}(),
					}
					drugAlts.Alternatives = append(drugAlts.Alternatives, alt)
				}
				
				alternatives[interaction.Drug1.Code] = drugAlts
				alternatives[interaction.Drug2.Code] = drugAlts
			}
		}
	}
	
	return alternatives
}

// Cache management methods

func (eim *EnhancedInteractionMatrixService) promoteToHotCache(key string, result *models.EnhancedInteractionResult) {
	eim.hotCacheMutex.Lock()
	defer eim.hotCacheMutex.Unlock()

	// Check if hot cache has space
	if len(eim.hotCache) < eim.config.MaxMatrixSize {
		eim.hotCache[key] = result
	}
}

func (eim *EnhancedInteractionMatrixService) cacheInteractionResult(key string, result *models.EnhancedInteractionResult) {
	// Cache in Redis with appropriate TTL - commented out until cache interface is implemented
	// ttl := eim.getCacheTTLForSeverity(result.Severity)
	// if err := eim.cache.SetWithTTL(key, result, ttl); err != nil {
	// 	fmt.Printf("Failed to cache interaction result: %v\n", err)
	// }
}

func (eim *EnhancedInteractionMatrixService) getCacheTTLForSeverity(severity models.DDISeverity) time.Duration {
	// Higher severity interactions cached longer (more likely to be needed again)
	switch severity {
	case models.SeverityContraindicated:
		return 24 * time.Hour
	case models.SeverityMajor:
		return 12 * time.Hour
	case models.SeverityModerate:
		return 6 * time.Hour
	case models.SeverityMinor:
		return 2 * time.Hour
	default:
		return 1 * time.Hour
	}
}

// Utility methods

func (eim *EnhancedInteractionMatrixService) buildInteractionKey(drugA, drugB string) string {
	// Ensure consistent ordering for cache keys
	if drugA > drugB {
		return fmt.Sprintf("%s_%s", drugB, drugA)
	}
	return fmt.Sprintf("%s_%s", drugA, drugB)
}

func (eim *EnhancedInteractionMatrixService) normalizeOrder(drugA, drugB string) (string, string) {
	if drugA > drugB {
		return drugB, drugA
	}
	return drugA, drugB
}

func (eim *EnhancedInteractionMatrixService) getCurrentDatasetVersion(ctx context.Context) (string, error) {
	var version string
	err := eim.db.DB.WithContext(ctx).
		Table("ddi_dataset_versions").
		Select("version_name").
		Where("is_current = TRUE").
		Scan(&version).Error
	
	if err != nil {
		return "", err
	}
	
	return version, nil
}

func (eim *EnhancedInteractionMatrixService) convertMatrixRowToResult(row *models.DDIInteractionMatrix) *models.EnhancedInteractionResult {
	return &models.EnhancedInteractionResult{
		InteractionID:      fmt.Sprintf("%s_%s_%s", row.Drug1Code, row.Drug2Code, row.DatasetVersion),
		Severity:           row.Severity,
		Mechanism:          row.Mechanism,
		ClinicalEffects:    row.ClinicalEffects,
		ManagementStrategy: row.ManagementStrategy,
		Evidence:           row.Evidence,
		Confidence:         row.Confidence,
		RouteSpecific:      len(row.RouteRestriction) > 0,
		Drug1:              models.DrugInfo{Code: row.Drug1Code},
		Drug2:              models.DrugInfo{Code: row.Drug2Code},
	}
}

func (eim *EnhancedInteractionMatrixService) convertPGXRuleToResult(rule *models.DDIPharmacogenomicRule, drugCodes []string) models.EnhancedInteractionResult {
	return models.EnhancedInteractionResult{
		InteractionID:      fmt.Sprintf("PGX_%s_%s_%s", rule.DrugCode, rule.Gene, rule.Phenotype),
		Severity:           rule.Severity,
		Mechanism:          models.MechanismPK, // PGx is typically PK
		ClinicalEffects:    getStringOrEmpty(rule.ClinicalEffects),
		ManagementStrategy: rule.ManagementStrategy,
		Evidence:           rule.Evidence,
		PGXApplicable:      true,
		Drug1:              models.DrugInfo{Code: rule.DrugCode},
		Drug2:              models.DrugInfo{Code: fmt.Sprintf("PGX_%s_%s", rule.Gene, rule.Phenotype)},
	}
}

func (eim *EnhancedInteractionMatrixService) convertModifierToResult(modifier *models.DDIModifier) models.EnhancedInteractionResult {
	return models.EnhancedInteractionResult{
		InteractionID:      fmt.Sprintf("MODIFIER_%s_%s", modifier.ModifierType, modifier.DrugCode),
		Severity:           modifier.Severity,
		Mechanism:          models.MechanismPD, // Modifiers typically PD
		ClinicalEffects:    modifier.Effect,
		ManagementStrategy: modifier.ManagementStrategy,
		Evidence:           modifier.Evidence,
		Drug1:              models.DrugInfo{Code: modifier.DrugCode},
		Drug2:              models.DrugInfo{Code: fmt.Sprintf("%s_%s", modifier.ModifierType, getStringOrEmpty(modifier.ModifierCode))},
	}
}

func (eim *EnhancedInteractionMatrixService) filterBySeverity(
	interactions []models.EnhancedInteractionResult,
	severityFilter []string,
) []models.EnhancedInteractionResult {
	if len(severityFilter) == 0 {
		return interactions
	}

	severitySet := make(map[string]bool)
	for _, severity := range severityFilter {
		severitySet[severity] = true
	}

	var filtered []models.EnhancedInteractionResult
	for _, interaction := range interactions {
		if severitySet[string(interaction.Severity)] {
			filtered = append(filtered, interaction)
		}
	}

	return filtered
}

// GetMatrixStatistics returns comprehensive statistics about the interaction matrix
func (eim *EnhancedInteractionMatrixService) GetMatrixStatistics() *models.EnhancedMatrixStatistics {
	eim.hotCacheMutex.RLock()
	hotCacheSize := len(eim.hotCache)
	eim.hotCacheMutex.RUnlock()

	stats := &models.EnhancedMatrixStatistics{
		TotalInteractions:     hotCacheSize,
		LastUpdated:           eim.lastRefresh,
		CurrentDatasetVersion: eim.currentDatasetVersion,
	}

	return stats
}

// RefreshMatrix manually refreshes the interaction matrix (for admin operations)
func (eim *EnhancedInteractionMatrixService) RefreshMatrix(ctx context.Context) error {
	return eim.LoadMatrix(ctx)
}

// Helper function
func getStringOrEmpty(s *string) string {
	if s != nil {
		return *s
	}
	return ""
}