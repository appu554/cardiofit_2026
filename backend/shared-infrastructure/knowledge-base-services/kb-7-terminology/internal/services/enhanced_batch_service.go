package services

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"kb-7-terminology/internal/cache"
	"kb-7-terminology/internal/database"
	"kb-7-terminology/internal/metrics"
	"kb-7-terminology/internal/models"

	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// BatchOperationType represents the type of batch operation
type BatchOperationType string

const (
	BatchTypeLookup      BatchOperationType = "lookup"
	BatchTypeValidation  BatchOperationType = "validation"
	BatchTypeTranslation BatchOperationType = "translation"
	BatchTypeSearch      BatchOperationType = "search"
)

// EnhancedBatchRequest represents a high-performance batch request
type EnhancedBatchRequest struct {
	Operation     BatchOperationType `json:"operation"`
	Items         []BatchItem        `json:"items"`
	Options       BatchOptions       `json:"options"`
	CacheStrategy string             `json:"cache_strategy,omitempty"` // "aggressive", "normal", "none"
}

// BatchItem represents an item in a batch request
type BatchItem struct {
	ID          string                 `json:"id,omitempty"`
	System      string                 `json:"system,omitempty"`
	Code        string                 `json:"code"`
	Display     string                 `json:"display,omitempty"`
	Version     string                 `json:"version,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// BatchOptions configures batch operation behavior
type BatchOptions struct {
	MaxConcurrency       int     `json:"max_concurrency"`
	TimeoutSeconds       int     `json:"timeout_seconds"`
	UseConnectionPool    bool    `json:"use_connection_pool"`
	EnableCaching        bool    `json:"enable_caching"`
	CacheExpiryMinutes   int     `json:"cache_expiry_minutes"`
	RetryFailedItems     bool    `json:"retry_failed_items"`
	FailFast             bool    `json:"fail_fast"`
	MinConfidenceScore   float64 `json:"min_confidence_score,omitempty"`
	IncludeInactive      bool    `json:"include_inactive"`
}

// BatchResponse represents the response from a batch operation
type BatchResponse struct {
	RequestID            string                 `json:"request_id"`
	Operation            BatchOperationType     `json:"operation"`
	Results              []BatchResult          `json:"results"`
	Summary              BatchSummary           `json:"summary"`
	ProcessingTimeMs     float64                `json:"processing_time_ms"`
	CacheHitRate         float64                `json:"cache_hit_rate"`
	OptimizationApplied  bool                   `json:"optimization_applied"`
	Errors               []string               `json:"errors,omitempty"`
}

// BatchResult represents a single result in a batch operation
type BatchResult struct {
	ID       string      `json:"id,omitempty"`
	Success  bool        `json:"success"`
	Data     interface{} `json:"data,omitempty"`
	Error    string      `json:"error,omitempty"`
	Cached   bool        `json:"cached"`
	Duration float64     `json:"duration_ms"`
}

// BatchSummary provides statistics about the batch operation
type BatchSummary struct {
	TotalItems       int     `json:"total_items"`
	SuccessfulItems  int     `json:"successful_items"`
	FailedItems      int     `json:"failed_items"`
	CachedItems      int     `json:"cached_items"`
	AverageDuration  float64 `json:"average_duration_ms"`
	ThroughputPerSec float64 `json:"throughput_per_sec"`
}

// EnhancedBatchService provides high-performance batch operations
type EnhancedBatchService struct {
	db           *sql.DB
	cache        cache.EnhancedCache
	logger       *logrus.Logger
	metrics      *metrics.Collector
	poolManager  *database.ConnectionPoolManager
	
	// Services for specific operations
	terminologyService *TerminologyService
	validationService  *ValidationService
	conceptMapService  *ConceptMapService
	searchService      *EnhancedSearchService
}

// NewEnhancedBatchService creates a new enhanced batch service
func NewEnhancedBatchService(
	db *sql.DB,
	cache cache.EnhancedCache,
	logger *logrus.Logger,
	metrics *metrics.Collector,
	poolManager *database.ConnectionPoolManager,
	terminologyService *TerminologyService,
	validationService *ValidationService,
	conceptMapService *ConceptMapService,
	searchService *EnhancedSearchService,
) *EnhancedBatchService {
	return &EnhancedBatchService{
		db:                 db,
		cache:              cache,
		logger:             logger,
		metrics:            metrics,
		poolManager:        poolManager,
		terminologyService: terminologyService,
		validationService:  validationService,
		conceptMapService:  conceptMapService,
		searchService:      searchService,
	}
}

// ProcessBatchRequest processes a batch request with optimization
func (s *EnhancedBatchService) ProcessBatchRequest(request EnhancedBatchRequest) (*BatchResponse, error) {
	start := time.Now()
	requestID := s.generateRequestID(request)
	
	s.logger.WithFields(logrus.Fields{
		"request_id":  requestID,
		"operation":   request.Operation,
		"item_count":  len(request.Items),
	}).Info("Processing enhanced batch request")
	
	// Validate request
	if err := s.validateBatchRequest(request); err != nil {
		return nil, fmt.Errorf("invalid batch request: %w", err)
	}
	
	// Set default options
	request.Options = s.setDefaultOptions(request.Options)
	
	// Check for cached batch result
	if request.Options.EnableCaching {
		if cached, err := s.getCachedBatchResult(requestID); err == nil {
			s.metrics.RecordCacheHit("batch_service", string(request.Operation))
			cached.ProcessingTimeMs = float64(time.Since(start).Nanoseconds()) / 1e6
			return cached, nil
		}
		s.metrics.RecordCacheMiss("batch_service", string(request.Operation))
	}
	
	// Execute batch operation with connection pool optimization
	var response *BatchResponse
	var err error
	
	if request.Options.UseConnectionPool && len(request.Items) > 50 {
		err = s.poolManager.ExecuteWithOptimization(len(request.Items), func() error {
			response, err = s.executeBatchOperation(request, requestID)
			return err
		})
		if response != nil {
			response.OptimizationApplied = true
		}
	} else {
		response, err = s.executeBatchOperation(request, requestID)
	}
	
	if err != nil {
		return nil, err
	}
	
	// Calculate final metrics
	response.ProcessingTimeMs = float64(time.Since(start).Nanoseconds()) / 1e6
	response.Summary.ThroughputPerSec = float64(len(request.Items)) / (response.ProcessingTimeMs / 1000)
	
	// Cache the result
	if request.Options.EnableCaching {
		s.cacheBatchResult(requestID, response, request.Options.CacheExpiryMinutes)
	}
	
	// Record metrics
	s.recordBatchMetrics(request.Operation, response)
	
	return response, nil
}

// executeBatchOperation executes the specific batch operation
func (s *EnhancedBatchService) executeBatchOperation(request EnhancedBatchRequest, requestID string) (*BatchResponse, error) {
	switch request.Operation {
	case BatchTypeLookup:
		return s.processBatchLookup(request, requestID)
	case BatchTypeValidation:
		return s.processBatchValidation(request, requestID)
	case BatchTypeTranslation:
		return s.processBatchTranslation(request, requestID)
	case BatchTypeSearch:
		return s.processBatchSearch(request, requestID)
	default:
		return nil, fmt.Errorf("unsupported batch operation: %s", request.Operation)
	}
}

// processBatchLookup processes batch concept lookup operations
func (s *EnhancedBatchService) processBatchLookup(request EnhancedBatchRequest, requestID string) (*BatchResponse, error) {
	results := make([]BatchResult, len(request.Items))
	
	// Group items by system for optimized queries
	systemGroups := s.groupItemsBySystem(request.Items)
	
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, request.Options.MaxConcurrency)
	
	for system, items := range systemGroups {
		wg.Add(1)
		go func(sys string, itemList []BatchItem) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			s.processSystemLookup(sys, itemList, results, request.Options)
		}(system, items)
	}
	
	wg.Wait()
	
	return s.buildBatchResponse(requestID, BatchTypeLookup, results), nil
}

// processSystemLookup processes lookup for a specific system using optimized queries
func (s *EnhancedBatchService) processSystemLookup(system string, items []BatchItem, results []BatchResult, options BatchOptions) {
	codes := make([]string, len(items))
	indexMap := make(map[string]int) // Map code to original index
	
	for i, item := range items {
		codes[i] = item.Code
		indexMap[item.Code] = i // Assuming items have original indices
	}
	
	// Use optimized batch lookup function
	query := `SELECT * FROM optimize_batch_lookup($1, $2)`
	
	rows, err := s.db.Query(query, pq.Array(codes), system)
	if err != nil {
		s.logger.WithError(err).Error("Batch lookup query failed")
		// Mark all items as failed
		for i := range items {
			results[i] = BatchResult{
				Success: false,
				Error:   fmt.Sprintf("Query failed: %s", err.Error()),
			}
		}
		return
	}
	defer rows.Close()
	
	foundCodes := make(map[string]*models.Concept)
	
	for rows.Next() {
		var concept models.Concept
		var properties sql.NullString

		err := rows.Scan(
			&concept.Code,
			&concept.System,
			&concept.PreferredTerm,
			&concept.Active,
			&properties,
		)
		if err != nil {
			continue
		}
		
		// Parse properties
		if properties.Valid {
			if err := concept.Properties.UnmarshalJSON([]byte(properties.String)); err != nil {
				s.logger.Warn("Failed to parse concept properties")
			}
		}
		
		foundCodes[concept.Code] = &concept
	}
	
	// Build results
	for i, item := range items {
		if concept, found := foundCodes[item.Code]; found {
			results[i] = BatchResult{
				ID:      item.ID,
				Success: true,
				Data:    concept,
			}
		} else {
			results[i] = BatchResult{
				ID:      item.ID,
				Success: false,
				Error:   "Concept not found",
			}
		}
	}
}

// processBatchValidation processes batch validation operations
func (s *EnhancedBatchService) processBatchValidation(request EnhancedBatchRequest, requestID string) (*BatchResponse, error) {
	results := make([]BatchResult, len(request.Items))
	
	// Convert to validation request format
	validationRequests := make([]models.ValidationRequest, len(request.Items))
	for i, item := range request.Items {
		validationRequests[i] = models.ValidationRequest{
			Code:    item.Code,
			System:  item.System,
			Version: item.Version,
			Display: item.Display,
		}
	}
	
	batchRequest := models.BatchValidationRequest{
		Requests: validationRequests,
		Options: models.BatchOptions{
			ParallelProcessing: true,
			MaxConcurrency:     request.Options.MaxConcurrency,
		},
	}
	
	// Use existing batch validation service
	response, err := s.validationService.BatchValidate(batchRequest)
	if err != nil {
		return nil, err
	}
	
	// Convert results
	for i, result := range response.Results {
		results[i] = BatchResult{
			ID:      request.Items[i].ID,
			Success: result.Valid,
			Data:    result,
		}
		if !result.Valid {
			results[i].Error = result.Message
		}
	}
	
	return s.buildBatchResponse(requestID, BatchTypeValidation, results), nil
}

// processBatchTranslation processes batch translation operations
func (s *EnhancedBatchService) processBatchTranslation(request EnhancedBatchRequest, requestID string) (*BatchResponse, error) {
	results := make([]BatchResult, len(request.Items))
	
	// Group by source/target system pairs for efficiency
	translationGroups := make(map[string][]BatchItem)
	
	for _, item := range request.Items {
		targetSystem, ok := item.Context["target_system"].(string)
		if !ok {
			continue
		}
		
		key := fmt.Sprintf("%s->%s", item.System, targetSystem)
		translationGroups[key] = append(translationGroups[key], item)
	}
	
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, request.Options.MaxConcurrency)
	
	for systemPair, items := range translationGroups {
		wg.Add(1)
		go func(pair string, itemList []BatchItem) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			s.processTranslationGroup(pair, itemList, results, request.Options)
		}(systemPair, items)
	}
	
	wg.Wait()
	
	return s.buildBatchResponse(requestID, BatchTypeTranslation, results), nil
}

// processTranslationGroup processes translation for a specific system pair
func (s *EnhancedBatchService) processTranslationGroup(systemPair string, items []BatchItem, results []BatchResult, options BatchOptions) {
	// Extract systems from pair string
	parts := strings.Split(systemPair, "->")
	if len(parts) != 2 {
		return
	}
	
	sourceSystem, targetSystem := parts[0], parts[1]
	
	// Convert to translation request format
	concepts := make([]TranslationConcept, len(items))
	for i, item := range items {
		concepts[i] = TranslationConcept{
			Code:    item.Code,
			Display: item.Display,
		}
	}
	
	batchRequest := BatchTranslationRequest{
		SourceSystem: sourceSystem,
		TargetSystem: targetSystem,
		Concepts:     concepts,
		Options: TranslationOptions{
			MinConfidence: options.MinConfidenceScore,
			MaxResults:    10,
		},
	}
	
	// Use concept map service for batch translation
	response, err := s.conceptMapService.BatchTranslateConcepts(batchRequest)
	if err != nil {
		// Mark all items as failed
		for i := range items {
			results[i] = BatchResult{
				Success: false,
				Error:   fmt.Sprintf("Translation failed: %s", err.Error()),
			}
		}
		return
	}
	
	// Convert results
	for i, result := range response.Results {
		results[i] = BatchResult{
			ID:      items[i].ID,
			Success: result.Match,
			Data:    result,
		}
		if !result.Match {
			results[i].Error = "No translation found"
		}
	}
}

// processBatchSearch processes batch search operations
func (s *EnhancedBatchService) processBatchSearch(request EnhancedBatchRequest, requestID string) (*BatchResponse, error) {
	results := make([]BatchResult, len(request.Items))
	
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, request.Options.MaxConcurrency)
	
	for i, item := range request.Items {
		wg.Add(1)
		go func(index int, searchItem BatchItem) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			start := time.Now()
			
			searchOptions := SearchOptions{
				TargetSystem:   searchItem.System,
				MaxResults:     10,
				ExpandSynonyms: true,
			}
			
			searchResponse, err := s.searchService.Search(searchItem.Code, searchOptions)
			duration := float64(time.Since(start).Nanoseconds()) / 1e6
			
			if err != nil {
				results[index] = BatchResult{
					ID:       searchItem.ID,
					Success:  false,
					Error:    err.Error(),
					Duration: duration,
				}
			} else {
				results[index] = BatchResult{
					ID:       searchItem.ID,
					Success:  searchResponse.TotalCount > 0,
					Data:     searchResponse,
					Duration: duration,
				}
			}
		}(i, item)
	}
	
	wg.Wait()
	
	return s.buildBatchResponse(requestID, BatchTypeSearch, results), nil
}

// Helper methods

func (s *EnhancedBatchService) validateBatchRequest(request EnhancedBatchRequest) error {
	if len(request.Items) == 0 {
		return fmt.Errorf("no items in batch request")
	}
	
	if len(request.Items) > 10000 {
		return fmt.Errorf("batch size exceeds maximum limit of 10000 items")
	}
	
	return nil
}

func (s *EnhancedBatchService) setDefaultOptions(options BatchOptions) BatchOptions {
	if options.MaxConcurrency == 0 {
		options.MaxConcurrency = 10
	}
	if options.TimeoutSeconds == 0 {
		options.TimeoutSeconds = 300 // 5 minutes
	}
	if options.CacheExpiryMinutes == 0 {
		options.CacheExpiryMinutes = 60
	}
	
	return options
}

func (s *EnhancedBatchService) groupItemsBySystem(items []BatchItem) map[string][]BatchItem {
	groups := make(map[string][]BatchItem)
	
	for _, item := range items {
		system := item.System
		if system == "" {
			system = "unknown"
		}
		groups[system] = append(groups[system], item)
	}
	
	return groups
}

func (s *EnhancedBatchService) buildBatchResponse(requestID string, operation BatchOperationType, results []BatchResult) *BatchResponse {
	summary := BatchSummary{
		TotalItems: len(results),
	}
	
	var totalDuration float64
	
	for _, result := range results {
		if result.Success {
			summary.SuccessfulItems++
		} else {
			summary.FailedItems++
		}
		
		if result.Cached {
			summary.CachedItems++
		}
		
		totalDuration += result.Duration
	}
	
	if len(results) > 0 {
		summary.AverageDuration = totalDuration / float64(len(results))
	}
	
	return &BatchResponse{
		RequestID: requestID,
		Operation: operation,
		Results:   results,
		Summary:   summary,
		CacheHitRate: float64(summary.CachedItems) / float64(summary.TotalItems),
	}
}

func (s *EnhancedBatchService) generateRequestID(request EnhancedBatchRequest) string {
	data, _ := json.Marshal(request)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:8]) // Use first 8 bytes for shorter ID
}

func (s *EnhancedBatchService) getCachedBatchResult(requestID string) (*BatchResponse, error) {
	cacheKey := fmt.Sprintf("batch_result:%s", requestID)
	cached, err := s.cache.Get(cacheKey)
	if err != nil {
		return nil, err
	}
	
	if response, ok := cached.(*BatchResponse); ok {
		return response, nil
	}
	
	return nil, fmt.Errorf("invalid cached batch result")
}

func (s *EnhancedBatchService) cacheBatchResult(requestID string, response *BatchResponse, expiryMinutes int) {
	cacheKey := fmt.Sprintf("batch_result:%s", requestID)
	expiry := time.Duration(expiryMinutes) * time.Minute
	
	if err := s.cache.Set(cacheKey, response, expiry); err != nil {
		s.logger.WithError(err).Warn("Failed to cache batch result")
	}
}

func (s *EnhancedBatchService) recordBatchMetrics(operation BatchOperationType, response *BatchResponse) {
	s.metrics.RecordBatchOperation(
		string(operation),
		"success",
		time.Duration(response.ProcessingTimeMs)*time.Millisecond,
		response.Summary.TotalItems,
	)
}