package services

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"kb-drug-interactions/internal/cache"
	"kb-drug-interactions/internal/config"
	"kb-drug-interactions/internal/database"
	"kb-drug-interactions/internal/models"
)

// InteractionMatrix provides optimized batch interaction checking
type InteractionMatrix struct {
	db      *database.Database
	cache   *cache.CacheClient
	config  *config.Config
	
	// Pre-computed interaction matrix for common drug combinations
	matrix       map[string]map[string]*models.DrugInteraction
	matrixMutex  sync.RWMutex
	lastUpdated  time.Time
	updateTicker *time.Ticker
	
	// High-performance lookup structures
	drugIndex    map[string]int
	indexToDrug  map[int]string
	adjacencyMat [][]bool
	interactionMap map[[2]int]*models.DrugInteraction
	
	// Batch processing pools
	batchPool    sync.Pool
	resultPool   sync.Pool
}

// NewInteractionMatrix creates a new optimized interaction matrix service
func NewInteractionMatrix(db *database.Database, cache *cache.CacheClient, config *config.Config) *InteractionMatrix {
	matrix := &InteractionMatrix{
		db:             db,
		cache:          cache,
		config:         config,
		matrix:         make(map[string]map[string]*models.DrugInteraction),
		drugIndex:      make(map[string]int),
		indexToDrug:    make(map[int]string),
		interactionMap: make(map[[2]int]*models.DrugInteraction),
		updateTicker:   time.NewTicker(24 * time.Hour), // Update daily
	}
	
	// Initialize object pools
	matrix.batchPool = sync.Pool{
		New: func() interface{} {
			return make([]models.BatchInteractionRequest, 0, 100)
		},
	}
	
	matrix.resultPool = sync.Pool{
		New: func() interface{} {
			return make([]models.InteractionResult, 0, 50)
		},
	}
	
	// Load initial matrix
	if err := matrix.LoadMatrix(context.Background()); err != nil {
		log.Printf("Failed to load initial interaction matrix: %v", err)
	}
	
	// Start background matrix updates
	go matrix.backgroundMatrixUpdate()
	
	return matrix
}

// LoadMatrix loads the interaction matrix from the database with optimizations
func (m *InteractionMatrix) LoadMatrix(ctx context.Context) error {
	timer := time.Now()
	defer func() {
		log.Printf("Matrix load completed in %v", time.Since(timer))
	}()
	
	m.matrixMutex.Lock()
	defer m.matrixMutex.Unlock()
	
	// Clear existing data
	m.matrix = make(map[string]map[string]*models.DrugInteraction)
	m.drugIndex = make(map[string]int)
	m.indexToDrug = make(map[int]string)
	m.interactionMap = make(map[[2]int]*models.DrugInteraction)
	
	// Load all interactions from database
	interactions, err := m.loadAllInteractions(ctx)
	if err != nil {
		return fmt.Errorf("failed to load interactions: %w", err)
	}
	
	log.Printf("Loaded %d interactions from database", len(interactions))
	
	// Build drug index
	drugSet := make(map[string]bool)
	for _, interaction := range interactions {
		drugSet[interaction.DrugACode] = true
		drugSet[interaction.DrugBCode] = true
	}
	
	// Create indexed mappings
	drugIndex := 0
	for drugCode := range drugSet {
		m.drugIndex[drugCode] = drugIndex
		m.indexToDrug[drugIndex] = drugCode
		drugIndex++
	}
	
	// Initialize adjacency matrix
	matrixSize := len(drugSet)
	m.adjacencyMat = make([][]bool, matrixSize)
	for i := range m.adjacencyMat {
		m.adjacencyMat[i] = make([]bool, matrixSize)
	}
	
	// Populate matrix structures
	for _, interaction := range interactions {
		drugACode := interaction.DrugACode
		drugBCode := interaction.DrugBCode
		
		// Ensure consistent ordering (A < B alphabetically)
		if drugACode > drugBCode {
			drugACode, drugBCode = drugBCode, drugACode
		}
		
		// Add to nested map structure
		if m.matrix[drugACode] == nil {
			m.matrix[drugACode] = make(map[string]*models.DrugInteraction)
		}
		m.matrix[drugACode][drugBCode] = &interaction
		
		// Add to indexed structures
		indexA := m.drugIndex[drugACode]
		indexB := m.drugIndex[drugBCode]
		
		if indexA < matrixSize && indexB < matrixSize {
			m.adjacencyMat[indexA][indexB] = true
			m.adjacencyMat[indexB][indexA] = true
			
			// Store interaction with consistent ordering
			key := [2]int{indexA, indexB}
			if indexA > indexB {
				key = [2]int{indexB, indexA}
			}
			m.interactionMap[key] = &interaction
		}
	}
	
	m.lastUpdated = time.Now()
	log.Printf("Matrix built with %d drugs and %d interactions", len(drugSet), len(interactions))
	
	return nil
}

// BatchCheckInteractions performs optimized batch interaction checking
func (m *InteractionMatrix) BatchCheckInteractions(ctx context.Context, requests []models.BatchInteractionRequest) ([]models.BatchInteractionResult, error) {
	if len(requests) == 0 {
		return nil, fmt.Errorf("no requests provided")
	}
	
	if len(requests) > m.config.MaxBatchSize {
		return nil, fmt.Errorf("batch size %d exceeds maximum %d", len(requests), m.config.MaxBatchSize)
	}
	
	timer := time.Now()
	defer func() {
		log.Printf("Batch check completed for %d requests in %v", len(requests), time.Since(timer))
	}()
	
	results := make([]models.BatchInteractionResult, len(requests))
	
	// Process in parallel using goroutine pool
	semaphore := make(chan struct{}, m.config.BatchConcurrency)
	var wg sync.WaitGroup
	
	for i, request := range requests {
		wg.Add(1)
		go func(idx int, req models.BatchInteractionRequest) {
			defer wg.Done()
			semaphore <- struct{}{} // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore
			
			result, err := m.checkSingleBatchRequest(ctx, req)
			if err != nil {
				log.Printf("Error processing batch request %d: %v", idx, err)
				results[idx] = models.BatchInteractionResult{
					RequestID:    req.RequestID,
					Interactions: []models.InteractionResult{},
					ProcessedAt:  time.Now().UTC(),
					Error:        err.Error(),
				}
			} else {
				results[idx] = *result
			}
		}(i, request)
	}
	
	wg.Wait()
	return results, nil
}

// FastLookup performs ultra-fast interaction lookup using the adjacency matrix
func (m *InteractionMatrix) FastLookup(drugACode, drugBCode string) (*models.DrugInteraction, bool) {
	m.matrixMutex.RLock()
	defer m.matrixMutex.RUnlock()
	
	// Ensure consistent ordering
	if drugACode > drugBCode {
		drugACode, drugBCode = drugBCode, drugACode
	}
	
	// Check nested map first (fastest for sparse lookups)
	if drugAMap, exists := m.matrix[drugACode]; exists {
		if interaction, found := drugAMap[drugBCode]; found {
			return interaction, true
		}
	}
	
	// Fallback to indexed lookup
	indexA, existsA := m.drugIndex[drugACode]
	indexB, existsB := m.drugIndex[drugBCode]
	
	if !existsA || !existsB {
		return nil, false
	}
	
	// Check adjacency matrix
	if indexA < len(m.adjacencyMat) && indexB < len(m.adjacencyMat[0]) {
		if m.adjacencyMat[indexA][indexB] {
			key := [2]int{indexA, indexB}
			if indexA > indexB {
				key = [2]int{indexB, indexA}
			}
			
			if interaction, exists := m.interactionMap[key]; exists {
				return interaction, true
			}
		}
	}
	
	return nil, false
}

// GetInteractionMatrix returns a subset of the interaction matrix for given drugs
func (m *InteractionMatrix) GetInteractionMatrix(drugCodes []string) (map[string]map[string]*models.DrugInteraction, error) {
	if len(drugCodes) > m.config.MaxMatrixSize {
		return nil, fmt.Errorf("requested matrix size %d exceeds maximum %d", len(drugCodes), m.config.MaxMatrixSize)
	}
	
	m.matrixMutex.RLock()
	defer m.matrixMutex.RUnlock()
	
	subMatrix := make(map[string]map[string]*models.DrugInteraction)
	
	for _, drugA := range drugCodes {
		for _, drugB := range drugCodes {
			if drugA != drugB {
				if interaction, found := m.FastLookupUnsafe(drugA, drugB); found {
					if subMatrix[drugA] == nil {
						subMatrix[drugA] = make(map[string]*models.DrugInteraction)
					}
					subMatrix[drugA][drugB] = interaction
				}
			}
		}
	}
	
	return subMatrix, nil
}

// GetMatrixStatistics returns performance and usage statistics
func (m *InteractionMatrix) GetMatrixStatistics() models.MatrixStatistics {
	m.matrixMutex.RLock()
	defer m.matrixMutex.RUnlock()
	
	return models.MatrixStatistics{
		TotalDrugs:        len(m.drugIndex),
		TotalInteractions: len(m.interactionMap),
		MatrixDensity:     float64(len(m.interactionMap)) / float64(len(m.drugIndex)*len(m.drugIndex)),
		LastUpdated:       m.lastUpdated,
		MemoryUsageMB:     m.estimateMemoryUsage(),
	}
}

// Private helper methods

func (m *InteractionMatrix) loadAllInteractions(ctx context.Context) ([]models.DrugInteraction, error) {
	// Check if cached version exists
	cacheKey := cache.AllInteractionsCacheKey()
	var cachedInteractions []models.DrugInteraction
	
	if m.config.EnableMatrixCaching {
		if err := m.cache.GetAllInteractions(cacheKey, &cachedInteractions); err == nil {
			log.Printf("Loaded %d interactions from cache", len(cachedInteractions))
			return cachedInteractions, nil
		}
	}
	
	// Load from database
	repo := database.NewInteractionRepository(m.db.DB)
	interactions, err := repo.GetAllActiveInteractions()
	if err != nil {
		return nil, err
	}
	
	// Cache the result
	if m.config.EnableMatrixCaching {
		cacheTTL := 6 * time.Hour
		if err := m.cache.SetAllInteractionsWithTTL(cacheKey, interactions, cacheTTL); err != nil {
			log.Printf("Failed to cache interactions: %v", err)
		}
	}
	
	return interactions, nil
}

func (m *InteractionMatrix) checkSingleBatchRequest(ctx context.Context, request models.BatchInteractionRequest) (*models.BatchInteractionResult, error) {
	if len(request.DrugCodes) < 2 {
		return nil, fmt.Errorf("at least 2 drug codes required")
	}
	
	// Get pooled result slice
	interactions := m.resultPool.Get().([]models.InteractionResult)
	interactions = interactions[:0] // Reset length but keep capacity
	defer m.resultPool.Put(interactions)
	
	// Check all pairwise combinations
	for i, drugA := range request.DrugCodes {
		for j := i + 1; j < len(request.DrugCodes); j++ {
			drugB := request.DrugCodes[j]
			
			if interaction, found := m.FastLookup(drugA, drugB); found {
				// Apply severity filtering if specified
				if len(request.SeverityFilter) > 0 && !m.matchesSeverityFilter(interaction.Severity, request.SeverityFilter) {
					continue
				}
				
				result := m.convertInteractionToResult(*interaction)
				interactions = append(interactions, result)
			}
		}
	}
	
	// Sort by severity priority
	sort.Slice(interactions, func(i, j int) bool {
		return m.config.GetSeverityPriority(interactions[i].Severity) > m.config.GetSeverityPriority(interactions[j].Severity)
	})
	
	// Create result copy (since we're returning the pooled slice)
	resultCopy := make([]models.InteractionResult, len(interactions))
	copy(resultCopy, interactions)
	
	result := &models.BatchInteractionResult{
		RequestID:    request.RequestID,
		Interactions: resultCopy,
		Summary:      m.buildBatchSummary(resultCopy),
		ProcessedAt:  time.Now().UTC(),
	}
	
	return result, nil
}

func (m *InteractionMatrix) FastLookupUnsafe(drugACode, drugBCode string) (*models.DrugInteraction, bool) {
	// Unsafe version that assumes caller holds the read lock
	if drugACode > drugBCode {
		drugACode, drugBCode = drugBCode, drugACode
	}
	
	if drugAMap, exists := m.matrix[drugACode]; exists {
		if interaction, found := drugAMap[drugBCode]; found {
			return interaction, true
		}
	}
	
	return nil, false
}

func (m *InteractionMatrix) matchesSeverityFilter(severity string, filter []string) bool {
	for _, filterSeverity := range filter {
		if severity == filterSeverity {
			return true
		}
	}
	return false
}

func (m *InteractionMatrix) convertInteractionToResult(interaction models.DrugInteraction) models.InteractionResult {
	result := models.InteractionResult{
		InteractionID:          interaction.InteractionID,
		Severity:               interaction.Severity,
		InteractionType:        interaction.InteractionType,
		EvidenceLevel:          interaction.EvidenceLevel,
		Mechanism:              interaction.Mechanism,
		ClinicalEffect:         interaction.ClinicalEffect,
		ManagementStrategy:     interaction.ManagementStrategy,
		DoseAdjustmentRequired: interaction.DoseAdjustmentRequired,
		TimeToOnset:            interaction.TimeToOnset,
		Duration:               interaction.Duration,
		DrugA: models.DrugInfo{
			Code: interaction.DrugACode,
			Name: interaction.DrugAName,
		},
		DrugB: models.DrugInfo{
			Code: interaction.DrugBCode,
			Name: interaction.DrugBName,
		},
	}
	
	// Add monitoring parameters if available
	if interaction.MonitoringParameters != nil {
		result.MonitoringParameters = *interaction.MonitoringParameters
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
	
	return result
}

func (m *InteractionMatrix) buildBatchSummary(interactions []models.InteractionResult) models.InteractionSummary {
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
	
	return summary
}

func (m *InteractionMatrix) estimateMemoryUsage() float64 {
	// Rough estimation of memory usage in MB
	drugsSize := len(m.drugIndex) * 32        // Drug index maps
	matrixSize := len(m.matrix) * 64          // Nested maps
	interactionsSize := len(m.interactionMap) * 256 // Interaction objects
	adjacencySize := len(m.adjacencyMat) * len(m.adjacencyMat[0]) / 8 // Boolean matrix
	
	totalBytes := drugsSize + matrixSize + interactionsSize + adjacencySize
	return float64(totalBytes) / (1024 * 1024)
}

func (m *InteractionMatrix) backgroundMatrixUpdate() {
	for range m.updateTicker.C {
		log.Printf("Starting scheduled matrix update")
		ctx := context.Background()
		
		if err := m.LoadMatrix(ctx); err != nil {
			log.Printf("Failed to update interaction matrix: %v", err)
		} else {
			log.Printf("Interaction matrix updated successfully")
		}
	}
}

// Close gracefully shuts down the matrix service
func (m *InteractionMatrix) Close() {
	if m.updateTicker != nil {
		m.updateTicker.Stop()
	}
}