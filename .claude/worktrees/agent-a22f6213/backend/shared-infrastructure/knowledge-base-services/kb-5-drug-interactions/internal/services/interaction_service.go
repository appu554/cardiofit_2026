package services

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"kb-drug-interactions/internal/cache"
	"kb-drug-interactions/internal/config"
	"kb-drug-interactions/internal/database"
	"kb-drug-interactions/internal/metrics"
	"kb-drug-interactions/internal/models"
)

type InteractionService struct {
	db         *database.Database
	cache      *cache.CacheClient
	metrics    *metrics.Collector
	config     *config.Config
	
	interactionRepo *database.InteractionRepository
	synonymRepo     *database.SynonymRepository
	alertRepo       *database.PatientAlertRepository
	ruleRepo        *database.RuleRepository
	cdsRepo         *database.CDSRepository
	analyticsRepo   *database.AnalyticsRepository
	
	// High-performance interaction matrix
	matrix         *InteractionMatrix
}

func NewInteractionService(
	db *database.Database,
	cache *cache.CacheClient,
	metrics *metrics.Collector,
	config *config.Config,
) *InteractionService {
	service := &InteractionService{
		db:              db,
		cache:           cache,
		metrics:         metrics,
		config:          config,
		interactionRepo: database.NewInteractionRepository(db.DB),
		synonymRepo:     database.NewSynonymRepository(db.DB),
		alertRepo:       database.NewPatientAlertRepository(db.DB),
		ruleRepo:        database.NewRuleRepository(db.DB),
		cdsRepo:         database.NewCDSRepository(db.DB),
		analyticsRepo:   database.NewAnalyticsRepository(db.DB),
	}
	
	// Initialize interaction matrix for high-performance lookups
	service.matrix = NewInteractionMatrix(db, cache, config)
	
	return service
}

// CheckInteractions is the main method for checking drug interactions
func (s *InteractionService) CheckInteractions(request models.InteractionCheckRequest) (*models.InteractionCheckResponse, error) {
	timer := metrics.StartTimer()
	defer func() {
		s.metrics.RecordInteractionCheck("comprehensive", timer.Duration())
	}()

	// Validate request
	if len(request.DrugCodes) < 2 {
		return nil, fmt.Errorf("at least 2 drug codes required for interaction checking")
	}

	if len(request.DrugCodes) > s.config.MaxInteractionsPerRequest {
		return nil, fmt.Errorf("too many drugs requested, maximum is %d", s.config.MaxInteractionsPerRequest)
	}

	// Check cache first
	cacheKey := s.buildInteractionCacheKey(request.DrugCodes, request.SeverityFilter)
	if s.config.CacheInteractionResults {
		var cachedResponse models.InteractionCheckResponse
		if err := s.cache.GetInteractionCheck(cacheKey, &cachedResponse); err == nil {
			s.metrics.RecordCacheHit("interaction_check", "comprehensive")
			cachedResponse.CacheHit = true
			return &cachedResponse, nil
		}
		s.metrics.RecordCacheMiss("interaction_check", "comprehensive")
	}

	// Resolve drug synonyms
	resolvedCodes, err := s.resolveDrugCodes(request.DrugCodes)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve drug codes: %w", err)
	}

	// Find interactions
	interactions, err := s.findInteractionsBetweenDrugs(resolvedCodes)
	if err != nil {
		return nil, fmt.Errorf("failed to find interactions: %w", err)
	}

	// Apply severity filtering
	if len(request.SeverityFilter) > 0 {
		interactions = s.filterInteractionsBySeverity(interactions, request.SeverityFilter)
	}

	// Convert to response format
	interactionResults := s.convertToInteractionResults(interactions, resolvedCodes)

	// Build response
	response := &models.InteractionCheckResponse{
		PatientID:       request.PatientID,
		CheckedDrugs:    request.DrugCodes,
		InteractionsFound: interactionResults,
		Summary:         s.buildInteractionSummary(interactionResults),
		Recommendations: s.generateRecommendations(interactionResults),
		CheckTimestamp:  time.Now().UTC(),
		CacheHit:        false,
	}

	// Add alternatives if requested
	if request.IncludeAlternatives {
		response.AlternativeDrugs = s.findAlternativeDrugs(interactionResults)
	}

	// Add monitoring plan if requested
	if request.IncludeMonitoring {
		response.MonitoringPlan = s.buildMonitoringPlan(interactionResults)
	}

	// Calculate risk score
	response.Summary.RiskScore = response.CalculateRiskScore()

	// Cache the response
	if s.config.CacheInteractionResults {
		cacheTTL := s.getCacheTTLForInteractions(interactionResults)
		if err := s.cache.SetInteractionCheckWithTTL(cacheKey, response, cacheTTL); err != nil {
			log.Printf("Failed to cache interaction check: %v", err)
		}
	}

	// Record analytics
	if s.config.EnableInteractionAnalytics {
		s.recordInteractionAnalytics(request, response)
	}

	// Create patient alerts if necessary
	if request.PatientID != "" {
		s.createPatientAlertsIfNeeded(request.PatientID, interactionResults)
	}

	return response, nil
}

// BatchCheckInteractions handles multiple interaction checks
func (s *InteractionService) BatchCheckInteractions(request models.BatchInteractionCheckRequest) ([]models.InteractionCheckResponse, error) {
	if len(request.Requests) == 0 {
		return nil, fmt.Errorf("no interaction check requests provided")
	}

	responses := make([]models.InteractionCheckResponse, len(request.Requests))
	
	// Process each request
	for i, req := range request.Requests {
		response, err := s.CheckInteractions(req)
		if err != nil {
			// Log error but continue processing other requests
			log.Printf("Error processing batch request %d: %v", i, err)
			responses[i] = models.InteractionCheckResponse{
				CheckedDrugs:    req.DrugCodes,
				InteractionsFound: []models.InteractionResult{},
				Summary: models.InteractionSummary{
					TotalInteractions: 0,
					SeverityCounts:    make(map[string]int),
				},
				CheckTimestamp: time.Now().UTC(),
			}
		} else {
			responses[i] = *response
		}
	}

	return responses, nil
}

// BatchCheckInteractionsMatrix performs optimized batch interaction checking using the interaction matrix
func (s *InteractionService) BatchCheckInteractionsMatrix(request models.BatchInteractionCheckRequest) ([]models.BatchInteractionResult, error) {
	timer := metrics.StartTimer()
	defer func() {
		s.metrics.RecordInteractionCheck("batch_matrix", timer.Duration())
	}()

	if len(request.Requests) == 0 {
		return nil, fmt.Errorf("no interaction check requests provided")
	}

	if len(request.Requests) > s.config.MaxBatchSize {
		return nil, fmt.Errorf("batch size %d exceeds maximum %d", len(request.Requests), s.config.MaxBatchSize)
	}

	// Convert to matrix batch format
	matrixRequests := make([]models.BatchInteractionRequest, len(request.Requests))
	for i, req := range request.Requests {
		matrixRequests[i] = models.BatchInteractionRequest{
			RequestID:      fmt.Sprintf("batch_%d", i),
			DrugCodes:      req.DrugCodes,
			SeverityFilter: req.SeverityFilter,
			PatientContext: map[string]interface{}{
				"patient_id": req.PatientID,
				"check_type": req.CheckType,
			},
		}
	}

	// Use matrix for high-performance batch processing
	ctx := context.Background()
	results, err := s.matrix.BatchCheckInteractions(ctx, matrixRequests)
	if err != nil {
		return nil, fmt.Errorf("matrix batch check failed: %w", err)
	}

	// Convert back to response format
	responses := make([]models.BatchInteractionResult, len(results))
	for i, result := range results {
		responses[i] = models.BatchInteractionResult{
			RequestID:    result.RequestID,
			Interactions: result.Interactions,
			Summary:      result.Summary,
			ProcessedAt:  result.ProcessedAt,
			Error:        result.Error,
		}
	}

	return responses, nil
}

// FastInteractionLookup performs ultra-fast pairwise interaction lookup
func (s *InteractionService) FastInteractionLookup(drugACode, drugBCode string) (*models.DrugInteraction, bool) {
	return s.matrix.FastLookup(drugACode, drugBCode)
}

// GetInteractionMatrix returns a subset of the interaction matrix for visualization
func (s *InteractionService) GetInteractionMatrix(drugCodes []string) (map[string]map[string]*models.DrugInteraction, error) {
	return s.matrix.GetInteractionMatrix(drugCodes)
}

// GetMatrixStatistics returns performance statistics for the interaction matrix
func (s *InteractionService) GetMatrixStatistics() models.MatrixStatistics {
	return s.matrix.GetMatrixStatistics()
}

// RefreshInteractionMatrix manually refreshes the interaction matrix
func (s *InteractionService) RefreshInteractionMatrix() error {
	ctx := context.Background()
	return s.matrix.LoadMatrix(ctx)
}

// GetPatientInteractionHistory retrieves a patient's interaction alert history
func (s *InteractionService) GetPatientInteractionHistory(request models.PatientInteractionHistoryRequest) ([]models.PatientInteractionAlert, int64, error) {
	limit := request.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	offset := request.Offset
	if offset < 0 {
		offset = 0
	}

	alerts, total, err := s.alertRepo.FindPatientHistory(request.PatientID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get patient interaction history: %w", err)
	}

	// Apply filters
	if !request.StartDate.IsZero() || !request.EndDate.IsZero() || len(request.SeverityFilter) > 0 {
		alerts = s.filterPatientAlerts(alerts, request)
	}

	return alerts, total, nil
}

// OverrideInteractionAlert allows authorized users to override interaction alerts
func (s *InteractionService) OverrideInteractionAlert(request models.AlertOverrideRequest) error {
	if request.OverrideReason == "" {
		return fmt.Errorf("override reason is required")
	}

	err := s.alertRepo.OverrideAlert(
		request.AlertID.String(),
		request.OverrideReason,
		request.OverriddenBy,
	)
	if err != nil {
		return fmt.Errorf("failed to override alert: %w", err)
	}

	// Record metrics
	s.metrics.RecordAlertOverride("interaction", "manual_override")

	// Log for audit trail
	log.Printf("Interaction alert %s overridden by %s: %s", 
		request.AlertID, request.OverriddenBy, request.OverrideReason)

	return nil
}

// GetInteractionDetails retrieves detailed information about a specific interaction
func (s *InteractionService) GetInteractionDetails(interactionID string) (*models.DrugInteraction, error) {
	// Check cache first
	cacheKey := cache.InteractionDetailCacheKey(interactionID)
	var cachedInteraction models.DrugInteraction
	if err := s.cache.GetInteractionDetail(cacheKey, &cachedInteraction); err == nil {
		s.metrics.RecordCacheHit("interaction_detail", "get")
		return &cachedInteraction, nil
	}
	s.metrics.RecordCacheMiss("interaction_detail", "get")

	// Query database
	interaction, err := s.interactionRepo.FindInteractionByID(interactionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get interaction details: %w", err)
	}

	// Cache the result
	if err := s.cache.SetInteractionDetail(cacheKey, interaction); err != nil {
		log.Printf("Failed to cache interaction detail: %v", err)
	}

	return interaction, nil
}

// Private helper methods

func (s *InteractionService) resolveDrugCodes(inputCodes []string) ([]string, error) {
	if !s.config.EnableSynonymResolution {
		return inputCodes, nil
	}

	codeMap, err := s.synonymRepo.ResolveDrugCodes(inputCodes)
	if err != nil {
		return nil, err
	}

	resolvedCodes := make([]string, 0, len(inputCodes))
	for _, code := range inputCodes {
		if resolvedCode, exists := codeMap[code]; exists {
			resolvedCodes = append(resolvedCodes, resolvedCode)
		} else {
			resolvedCodes = append(resolvedCodes, code)
		}
	}

	return resolvedCodes, nil
}

func (s *InteractionService) findInteractionsBetweenDrugs(drugCodes []string) ([]models.DrugInteraction, error) {
	return s.interactionRepo.FindInteractionsBetweenDrugs(drugCodes)
}

func (s *InteractionService) filterInteractionsBySeverity(interactions []models.DrugInteraction, severityFilter []string) []models.DrugInteraction {
	if len(severityFilter) == 0 {
		return interactions
	}

	severitySet := make(map[string]bool)
	for _, severity := range severityFilter {
		severitySet[severity] = true
	}

	filtered := make([]models.DrugInteraction, 0)
	for _, interaction := range interactions {
		if severitySet[interaction.Severity] {
			filtered = append(filtered, interaction)
		}
	}

	return filtered
}

func (s *InteractionService) convertToInteractionResults(interactions []models.DrugInteraction, drugCodes []string) []models.InteractionResult {
	results := make([]models.InteractionResult, len(interactions))

	for i, interaction := range interactions {
		result := models.InteractionResult{
			InteractionID:       interaction.InteractionID,
			Severity:            interaction.Severity,
			InteractionType:     interaction.InteractionType,
			EvidenceLevel:       interaction.EvidenceLevel,
			Mechanism:           interaction.Mechanism,
			ClinicalEffect:      interaction.ClinicalEffect,
			ManagementStrategy:  interaction.ManagementStrategy,
			DoseAdjustmentRequired: interaction.DoseAdjustmentRequired,
			TimeToOnset:         interaction.TimeToOnset,
			Duration:            interaction.Duration,
		}

		// Set drug information
		result.DrugA = models.DrugInfo{
			Code: interaction.DrugACode,
			Name: interaction.DrugAName,
		}
		result.DrugB = models.DrugInfo{
			Code: interaction.DrugBCode,
			Name: interaction.DrugBName,
		}

		// Add monitoring parameters if available
		if interaction.MonitoringParameters != nil {
			result.MonitoringParameters = *interaction.MonitoringParameters
		}

		// Add alternative drugs if available
		if interaction.AlternativeDrugs != nil {
			if alternatives, ok := (*interaction.AlternativeDrugs)["alternatives"].([]interface{}); ok {
				result.AlternativeDrugs = make([]string, len(alternatives))
				for j, alt := range alternatives {
					if altStr, ok := alt.(string); ok {
						result.AlternativeDrugs[j] = altStr
					}
				}
			}
		}

		// Add scores if available
		if interaction.FrequencyScore != nil {
			freq := interaction.FrequencyScore.InexactFloat64()
			result.FrequencyScore = &freq
		}
		if interaction.ClinicalSignificance != nil {
			sig := interaction.ClinicalSignificance.InexactFloat64()
			result.ClinicalSignificance = &sig
		}

		results[i] = result
	}

	// Sort by severity priority
	sort.Slice(results, func(i, j int) bool {
		return s.config.GetSeverityPriority(results[i].Severity) > s.config.GetSeverityPriority(results[j].Severity)
	})

	return results
}

func (s *InteractionService) buildInteractionSummary(interactions []models.InteractionResult) models.InteractionSummary {
	summary := models.InteractionSummary{
		TotalInteractions:    len(interactions),
		SeverityCounts:       make(map[string]int),
		RequiredActions:      []string{},
		ContraindicatedPairs: 0,
	}

	if len(interactions) == 0 {
		return summary
	}

	// Count by severity
	for _, interaction := range interactions {
		summary.SeverityCounts[interaction.Severity]++
		
		if interaction.Severity == "contraindicated" {
			summary.ContraindicatedPairs++
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
		summary.RequiredActions = append(summary.RequiredActions, "Review contraindicated combinations")
	}
	if summary.SeverityCounts["major"] > 0 {
		summary.RequiredActions = append(summary.RequiredActions, "Consider dose adjustments or monitoring")
	}
	if summary.SeverityCounts["moderate"] > 0 {
		summary.RequiredActions = append(summary.RequiredActions, "Monitor for adverse effects")
	}

	return summary
}

func (s *InteractionService) generateRecommendations(interactions []models.InteractionResult) []string {
	recommendations := []string{}
	
	if len(interactions) == 0 {
		return []string{"No significant drug interactions detected"}
	}

	// Add severity-specific recommendations
	contraindicated := 0
	major := 0
	for _, interaction := range interactions {
		switch interaction.Severity {
		case "contraindicated":
			contraindicated++
		case "major":
			major++
		}
	}

	if contraindicated > 0 {
		recommendations = append(recommendations, 
			fmt.Sprintf("URGENT: %d contraindicated drug combinations found. Consider alternative medications immediately.", contraindicated))
	}

	if major > 0 {
		recommendations = append(recommendations, 
			fmt.Sprintf("CAUTION: %d major interactions detected. Review dosing and implement enhanced monitoring.", major))
	}

	// Add specific management strategies for top interactions
	for i, interaction := range interactions {
		if i >= 3 { // Limit to top 3 interactions
			break
		}
		if interaction.Severity == "contraindicated" || interaction.Severity == "major" {
			recommendations = append(recommendations, 
				fmt.Sprintf("%s + %s: %s", 
					interaction.DrugA.Name, 
					interaction.DrugB.Name, 
					interaction.ManagementStrategy))
		}
	}

	return recommendations
}

func (s *InteractionService) findAlternativeDrugs(interactions []models.InteractionResult) map[string][]string {
	alternatives := make(map[string][]string)
	
	for _, interaction := range interactions {
		if len(interaction.AlternativeDrugs) > 0 {
			// Add alternatives for both drugs
			alternatives[interaction.DrugA.Code] = interaction.AlternativeDrugs
			alternatives[interaction.DrugB.Code] = interaction.AlternativeDrugs
		}
	}
	
	return alternatives
}

func (s *InteractionService) buildMonitoringPlan(interactions []models.InteractionResult) []models.MonitoringRecommendation {
	monitoringPlan := []models.MonitoringRecommendation{}
	
	for _, interaction := range interactions {
		if len(interaction.MonitoringParameters) > 0 {
			for param, details := range interaction.MonitoringParameters {
				if detailsMap, ok := details.(map[string]interface{}); ok {
					recommendation := models.MonitoringRecommendation{
						Parameter:    param,
						Instructions: fmt.Sprintf("Monitor for %s interaction between %s and %s", 
							interaction.Severity, interaction.DrugA.Name, interaction.DrugB.Name),
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
	
	return monitoringPlan
}

func (s *InteractionService) createPatientAlertsIfNeeded(patientID string, interactions []models.InteractionResult) {
	for _, interaction := range interactions {
		// Only create alerts for major and contraindicated interactions
		if interaction.Severity == "major" || interaction.Severity == "contraindicated" {
			alert := &models.PatientInteractionAlert{
				ID:               uuid.New(),
				PatientID:        patientID,
				AlertTriggeredAt: time.Now().UTC(),
				AlertSeverity:    interaction.Severity,
				AlertStatus:      "active",
				DrugRegimen: models.JSONB{
					"drug_a": interaction.DrugA,
					"drug_b": interaction.DrugB,
					"interaction_id": interaction.InteractionID,
				},
			}

			if err := s.alertRepo.Create(alert); err != nil {
				log.Printf("Failed to create patient alert: %v", err)
			} else {
				s.metrics.RecordAlertCreated("interaction", interaction.Severity, "automatic")
			}
		}
	}
}

// Utility methods

func (s *InteractionService) buildInteractionCacheKey(drugCodes []string, severityFilter []string) string {
	// Sort drug codes for consistent cache keys
	sortedCodes := make([]string, len(drugCodes))
	copy(sortedCodes, drugCodes)
	sort.Strings(sortedCodes)
	
	key := strings.Join(sortedCodes, ",")
	if len(severityFilter) > 0 {
		sortedSeverities := make([]string, len(severityFilter))
		copy(sortedSeverities, severityFilter)
		sort.Strings(sortedSeverities)
		key += ":severity:" + strings.Join(sortedSeverities, ",")
	}
	
	return key
}

func (s *InteractionService) getCacheTTLForInteractions(interactions []models.InteractionResult) time.Duration {
	if len(interactions) == 0 {
		return s.config.InteractionCacheTTL
	}
	
	// Use the highest severity to determine TTL
	highestSeverity := "minor"
	for _, interaction := range interactions {
		if s.config.GetSeverityPriority(interaction.Severity) > s.config.GetSeverityPriority(highestSeverity) {
			highestSeverity = interaction.Severity
		}
	}
	
	return s.config.GetInteractionCacheTTL(highestSeverity)
}

func (s *InteractionService) filterPatientAlerts(alerts []models.PatientInteractionAlert, request models.PatientInteractionHistoryRequest) []models.PatientInteractionAlert {
	filtered := make([]models.PatientInteractionAlert, 0)
	
	for _, alert := range alerts {
		// Date filtering
		if !request.StartDate.IsZero() && alert.AlertTriggeredAt.Before(request.StartDate) {
			continue
		}
		if !request.EndDate.IsZero() && alert.AlertTriggeredAt.After(request.EndDate) {
			continue
		}
		
		// Severity filtering
		if len(request.SeverityFilter) > 0 {
			found := false
			for _, severity := range request.SeverityFilter {
				if alert.AlertSeverity == severity {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		
		// Resolved status filtering
		if !request.IncludeResolved && alert.AlertStatus != "active" {
			continue
		}
		
		filtered = append(filtered, alert)
	}
	
	return filtered
}

func (s *InteractionService) recordInteractionAnalytics(request models.InteractionCheckRequest, response *models.InteractionCheckResponse) {
	// This would typically be done in a background goroutine
	go func() {
		analytics := &models.InteractionAnalytics{
			AnalysisDate:    time.Now().UTC(),
			AlertFrequency:  len(response.InteractionsFound),
			ClinicalEvents:  0, // Would be updated based on actual outcomes
		}

		if err := s.analyticsRepo.CreateAnalytics(analytics); err != nil {
			log.Printf("Failed to record interaction analytics: %v", err)
		}
	}()
}

// GetDatabase returns the database for health checks
func (s *InteractionService) GetDatabase() *database.Database {
	return s.db
}

// GetCache returns the cache client for health checks
func (s *InteractionService) GetCache() *cache.CacheClient {
	return s.cache
}

// GetMetrics returns the metrics collector
func (s *InteractionService) GetMetrics() *metrics.Collector {
	return s.metrics
}