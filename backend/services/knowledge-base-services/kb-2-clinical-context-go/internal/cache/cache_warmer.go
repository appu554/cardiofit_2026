package cache

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"kb-2-clinical-context-go/internal/config"
	"kb-2-clinical-context-go/internal/models"
)

// CacheWarmer implements intelligent cache warming strategies
type CacheWarmer struct {
	cache   *MultiTierCache
	config  *config.Config
	
	// Warming strategies
	strategies map[string]WarmingStrategy
	
	// Statistics
	warmingRuns    int64
	itemsWarmed    int64
	warmingErrors  int64
	lastWarmingRun time.Time
	
	// Coordination
	mu sync.RWMutex
}

// WarmingStrategy interface for different warming approaches
type WarmingStrategy interface {
	Name() string
	ShouldWarm(stats map[string]*CacheStats) bool
	WarmKeys(ctx context.Context, cache *MultiTierCache) ([]string, error)
	Priority() int
}

// NewCacheWarmer creates a new cache warmer
func NewCacheWarmer(cache *MultiTierCache, config *config.Config) *CacheWarmer {
	warmer := &CacheWarmer{
		cache:      cache,
		config:     config,
		strategies: make(map[string]WarmingStrategy),
	}
	
	// Initialize warming strategies
	warmer.initializeStrategies()
	
	return warmer
}

// WarmCache executes cache warming strategies
func (cw *CacheWarmer) WarmCache(ctx context.Context) error {
	cw.mu.Lock()
	defer cw.mu.Unlock()
	
	cw.warmingRuns++
	cw.lastWarmingRun = time.Now()
	
	// Get current cache statistics
	stats := cw.cache.GetStats()
	
	// Execute warming strategies based on priority and conditions
	strategies := cw.getSortedStrategies(stats)
	
	var errors []error
	totalWarmed := 0
	
	for _, strategy := range strategies {
		if !strategy.ShouldWarm(stats) {
			continue
		}
		
		keys, err := strategy.WarmKeys(ctx, cw.cache)
		if err != nil {
			cw.warmingErrors++
			errors = append(errors, fmt.Errorf("strategy %s failed: %w", strategy.Name(), err))
			continue
		}
		
		totalWarmed += len(keys)
		log.Printf("Cache warming strategy %s warmed %d keys", strategy.Name(), len(keys))
	}
	
	cw.itemsWarmed += int64(totalWarmed)
	
	if len(errors) > 0 {
		return fmt.Errorf("cache warming completed with errors: %v", errors)
	}
	
	log.Printf("Cache warming completed: %d items warmed across %d strategies", totalWarmed, len(strategies))
	return nil
}

// OptimizeWarmingStrategy optimizes warming based on cache performance
func (cw *CacheWarmer) OptimizeWarmingStrategy(ctx context.Context, stats map[string]*CacheStats) error {
	// Adjust warming frequency based on hit rates
	l1HitRate := stats["l1"].HitRate
	l2HitRate := stats["l2"].HitRate
	
	// If hit rates are below target, increase warming frequency
	if l1HitRate < 0.85 || l2HitRate < 0.95 {
		// Trigger immediate warming
		return cw.WarmCache(ctx)
	}
	
	return nil
}

// initializeStrategies initializes all warming strategies
func (cw *CacheWarmer) initializeStrategies() {
	// Strategy 1: Phenotype Definitions (Static, High Priority)
	cw.strategies["phenotype_definitions"] = &PhenotypeDefinitionsStrategy{}
	
	// Strategy 2: Frequent Patient Contexts (Dynamic, Medium Priority)
	cw.strategies["frequent_patients"] = &FrequentPatientsStrategy{}
	
	// Strategy 3: Risk Models (Static, High Priority)
	cw.strategies["risk_models"] = &RiskModelsStrategy{}
	
	// Strategy 4: Treatment Preferences (Static, Medium Priority)
	cw.strategies["treatment_preferences"] = &TreatmentPreferencesStrategy{}
	
	// Strategy 5: Hot Keys (Dynamic, Low Priority)
	cw.strategies["hot_keys"] = &HotKeysStrategy{}
}

// getSortedStrategies returns strategies sorted by priority
func (cw *CacheWarmer) getSortedStrategies(stats map[string]*CacheStats) []WarmingStrategy {
	strategies := make([]WarmingStrategy, 0, len(cw.strategies))
	
	for _, strategy := range cw.strategies {
		strategies = append(strategies, strategy)
	}
	
	// Sort by priority (higher priority first)
	sort.Slice(strategies, func(i, j int) bool {
		return strategies[i].Priority() > strategies[j].Priority()
	})
	
	return strategies
}

// GetWarmingStats returns cache warming statistics
func (cw *CacheWarmer) GetWarmingStats() map[string]interface{} {
	cw.mu.RLock()
	defer cw.mu.RUnlock()
	
	return map[string]interface{}{
		"warming_runs":     cw.warmingRuns,
		"items_warmed":     cw.itemsWarmed,
		"warming_errors":   cw.warmingErrors,
		"last_warming_run": cw.lastWarmingRun,
		"strategies_count": len(cw.strategies),
	}
}

// Warming Strategy Implementations

// PhenotypeDefinitionsStrategy warms phenotype definitions (static content)
type PhenotypeDefinitionsStrategy struct{}

func (pds *PhenotypeDefinitionsStrategy) Name() string {
	return "phenotype_definitions"
}

func (pds *PhenotypeDefinitionsStrategy) ShouldWarm(stats map[string]*CacheStats) bool {
	// Always warm static phenotype definitions if L3 hit rate is low
	if l3Stats, exists := stats["l3"]; exists {
		return l3Stats.HitRate < 0.9
	}
	return true
}

func (pds *PhenotypeDefinitionsStrategy) WarmKeys(ctx context.Context, cache *MultiTierCache) ([]string, error) {
	// Warm common phenotype definitions
	phenotypeKeys := []string{
		"phenotype_definition:diabetes_type2",
		"phenotype_definition:hypertension_essential",
		"phenotype_definition:cardiovascular_risk_high",
		"phenotype_definition:medication_adherence_low",
		"phenotype_definition:frailty_syndrome",
		"phenotype_definition:polypharmacy_risk",
		"phenotype_definition:fall_risk_high",
		"phenotype_definition:bleeding_risk_high",
	}
	
	warmedKeys := []string{}
	
	for _, key := range phenotypeKeys {
		// Check if already cached
		if _, found := cache.l1Cache.Get(key); found {
			continue
		}
		
		// Load phenotype definition (mock loader)
		_, err := cache.Get(ctx, key, func() (interface{}, error) {
			// In production, this would load from knowledge base
			return &models.PhenotypeDefinition{
				Name:        extractNameFromKey(key),
				Description: "Cached phenotype definition",
				Category:    "clinical",
				CELRule:     "true", // Placeholder
				Priority:    1,
				Version:     "1.0",
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}, nil
		})
		
		if err != nil {
			return warmedKeys, fmt.Errorf("failed to warm phenotype %s: %w", key, err)
		}
		
		warmedKeys = append(warmedKeys, key)
	}
	
	return warmedKeys, nil
}

func (pds *PhenotypeDefinitionsStrategy) Priority() int {
	return 100 // Highest priority
}

// FrequentPatientsStrategy warms frequently accessed patient contexts
type FrequentPatientsStrategy struct{}

func (fps *FrequentPatientsStrategy) Name() string {
	return "frequent_patients"
}

func (fps *FrequentPatientsStrategy) ShouldWarm(stats map[string]*CacheStats) bool {
	// Warm if L1 hit rate is below target
	if l1Stats, exists := stats["l1"]; exists {
		return l1Stats.HitRate < 0.85
	}
	return true
}

func (fps *FrequentPatientsStrategy) WarmKeys(ctx context.Context, cache *MultiTierCache) ([]string, error) {
	// Get hot keys from L1 cache (most frequently accessed)
	hotKeys := cache.l1Cache.GetHotKeys(20) // Top 20 hot keys
	
	// Filter for patient context keys
	patientKeys := []string{}
	for _, key := range hotKeys {
		if isPatientContextKey(key) {
			patientKeys = append(patientKeys, key)
		}
	}
	
	// Warm patient contexts that might have expired
	warmedKeys := []string{}
	for _, key := range patientKeys {
		if _, found := cache.l1Cache.Get(key); !found {
			// Load patient context (mock loader)
			_, err := cache.Get(ctx, key, func() (interface{}, error) {
				return &models.ClinicalContext{
					PatientID:   extractPatientIDFromKey(key),
					GeneratedAt: time.Now(),
					ContextSummary: models.ContextSummary{
						KeyFindings: []string{"Cached context"},
					},
				}, nil
			})
			
			if err == nil {
				warmedKeys = append(warmedKeys, key)
			}
		}
	}
	
	return warmedKeys, nil
}

func (fps *FrequentPatientsStrategy) Priority() int {
	return 80 // High priority
}

// RiskModelsStrategy warms risk assessment models
type RiskModelsStrategy struct{}

func (rms *RiskModelsStrategy) Name() string {
	return "risk_models"
}

func (rms *RiskModelsStrategy) ShouldWarm(stats map[string]*CacheStats) bool {
	// Warm risk models if operations count is high but hit rate is low
	if combined, exists := stats["combined"]; exists {
		return combined.Operations > 100 && combined.HitRate < 0.9
	}
	return true
}

func (rms *RiskModelsStrategy) WarmKeys(ctx context.Context, cache *MultiTierCache) ([]string, error) {
	riskModelKeys := []string{
		"risk_model:cardiovascular_framingham",
		"risk_model:diabetes_complications",
		"risk_model:medication_interactions", 
		"risk_model:fall_risk_assessment",
		"risk_model:bleeding_risk_hasbled",
	}
	
	warmedKeys := []string{}
	
	for _, key := range riskModelKeys {
		_, err := cache.Get(ctx, key, func() (interface{}, error) {
			// Mock risk model
			return map[string]interface{}{
				"model_name":    extractNameFromKey(key),
				"version":       "1.0",
				"coefficients":  []float64{0.1, 0.2, 0.3},
				"intercept":     0.05,
				"last_updated":  time.Now(),
			}, nil
		})
		
		if err == nil {
			warmedKeys = append(warmedKeys, key)
		}
	}
	
	return warmedKeys, nil
}

func (rms *RiskModelsStrategy) Priority() int {
	return 90 // High priority
}

// TreatmentPreferencesStrategy warms treatment preference templates
type TreatmentPreferencesStrategy struct{}

func (tps *TreatmentPreferencesStrategy) Name() string {
	return "treatment_preferences"
}

func (tps *TreatmentPreferencesStrategy) ShouldWarm(stats map[string]*CacheStats) bool {
	// Warm treatment preferences periodically
	return true
}

func (tps *TreatmentPreferencesStrategy) WarmKeys(ctx context.Context, cache *MultiTierCache) ([]string, error) {
	treatmentKeys := []string{
		"treatment_preference_template:diabetes_type2",
		"treatment_preference_template:hypertension_essential",
		"treatment_preference_template:cardiovascular_prevention",
		"treatment_preference_template:anticoagulation_therapy",
	}
	
	warmedKeys := []string{}
	
	for _, key := range treatmentKeys {
		_, err := cache.Get(ctx, key, func() (interface{}, error) {
			// Mock treatment preference template
			return map[string]interface{}{
				"condition":      extractNameFromKey(key),
				"options":        []string{"option1", "option2", "option3"},
				"default_rules":  []string{"first_line", "cost_effective"},
				"last_updated":   time.Now(),
			}, nil
		})
		
		if err == nil {
			warmedKeys = append(warmedKeys, key)
		}
	}
	
	return warmedKeys, nil
}

func (tps *TreatmentPreferencesStrategy) Priority() int {
	return 70 // Medium priority
}

// HotKeysStrategy warms the most frequently accessed keys
type HotKeysStrategy struct{}

func (hks *HotKeysStrategy) Name() string {
	return "hot_keys"
}

func (hks *HotKeysStrategy) ShouldWarm(stats map[string]*CacheStats) bool {
	// Only warm hot keys if there's evidence of frequent access
	if l1Stats, exists := stats["l1"]; exists {
		return l1Stats.Operations > 1000 // High activity
	}
	return false
}

func (hks *HotKeysStrategy) WarmKeys(ctx context.Context, cache *MultiTierCache) ([]string, error) {
	// This strategy relies on actual hot key tracking
	// For now, return empty as it requires runtime data
	return []string{}, nil
}

func (hks *HotKeysStrategy) Priority() int {
	return 50 // Lower priority
}

// Predictive warming based on usage patterns

// PredictiveWarmer predicts which keys will be accessed soon
type PredictiveWarmer struct {
	accessPatterns map[string][]time.Time
	mu            sync.RWMutex
}

// RecordAccess records key access for pattern analysis
func (pw *PredictiveWarmer) RecordAccess(key string) {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	
	if pw.accessPatterns == nil {
		pw.accessPatterns = make(map[string][]time.Time)
	}
	
	// Record access time
	accesses := pw.accessPatterns[key]
	accesses = append(accesses, time.Now())
	
	// Keep only recent accesses (last 24 hours)
	cutoff := time.Now().Add(-24 * time.Hour)
	recentAccesses := []time.Time{}
	for _, access := range accesses {
		if access.After(cutoff) {
			recentAccesses = append(recentAccesses, access)
		}
	}
	
	pw.accessPatterns[key] = recentAccesses
}

// PredictNextAccess predicts when a key will be accessed next
func (pw *PredictiveWarmer) PredictNextAccess(key string) time.Time {
	pw.mu.RLock()
	defer pw.mu.RUnlock()
	
	accesses, exists := pw.accessPatterns[key]
	if !exists || len(accesses) < 2 {
		return time.Time{} // Not enough data
	}
	
	// Calculate average interval
	var totalInterval time.Duration
	for i := 1; i < len(accesses); i++ {
		interval := accesses[i].Sub(accesses[i-1])
		totalInterval += interval
	}
	
	avgInterval := totalInterval / time.Duration(len(accesses)-1)
	lastAccess := accesses[len(accesses)-1]
	
	return lastAccess.Add(avgInterval)
}

// GetKeysToWarmSoon returns keys that should be warmed based on prediction
func (pw *PredictiveWarmer) GetKeysToWarmSoon(within time.Duration) []string {
	pw.mu.RLock()
	defer pw.mu.RUnlock()
	
	now := time.Now()
	threshold := now.Add(within)
	
	keysToWarm := []string{}
	
	for key := range pw.accessPatterns {
		nextAccess := pw.PredictNextAccess(key)
		if !nextAccess.IsZero() && nextAccess.Before(threshold) {
			keysToWarm = append(keysToWarm, key)
		}
	}
	
	return keysToWarm
}

// Batch warming operations

// WarmBatch warms multiple keys efficiently
func (cw *CacheWarmer) WarmBatch(ctx context.Context, keys []string) error {
	if len(keys) == 0 {
		return nil
	}
	
	// Group keys by cache tier preference
	l1Keys, l2Keys, l3Keys := cw.groupKeysByTier(keys)
	
	var wg sync.WaitGroup
	errors := make(chan error, 3)
	
	// Warm L1 cache
	if len(l1Keys) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := cw.warmL1Keys(ctx, l1Keys); err != nil {
				errors <- fmt.Errorf("L1 warming failed: %w", err)
			}
		}()
	}
	
	// Warm L2 cache
	if len(l2Keys) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := cw.warmL2Keys(ctx, l2Keys); err != nil {
				errors <- fmt.Errorf("L2 warming failed: %w", err)
			}
		}()
	}
	
	// Warm L3 cache
	if len(l3Keys) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := cw.warmL3Keys(ctx, l3Keys); err != nil {
				errors <- fmt.Errorf("L3 warming failed: %w", err)
			}
		}()
	}
	
	wg.Wait()
	close(errors)
	
	// Check for errors
	var warmingErrors []error
	for err := range errors {
		warmingErrors = append(warmingErrors, err)
	}
	
	if len(warmingErrors) > 0 {
		return fmt.Errorf("batch warming had errors: %v", warmingErrors)
	}
	
	return nil
}

// groupKeysByTier groups keys by their optimal cache tier
func (cw *CacheWarmer) groupKeysByTier(keys []string) (l1Keys, l2Keys, l3Keys []string) {
	for _, key := range keys {
		if cw.isHighFrequencyKey(key) {
			l1Keys = append(l1Keys, key)
		} else if cw.cache.isStaticContent(key) {
			l3Keys = append(l3Keys, key)
		} else {
			l2Keys = append(l2Keys, key)
		}
	}
	return
}

// warmL1Keys warms keys in L1 cache
func (cw *CacheWarmer) warmL1Keys(ctx context.Context, keys []string) error {
	for _, key := range keys {
		_, err := cw.cache.Get(ctx, key, func() (interface{}, error) {
			return cw.mockLoader(key)
		})
		if err != nil {
			return fmt.Errorf("failed to warm L1 key %s: %w", key, err)
		}
	}
	return nil
}

// warmL2Keys warms keys in L2 cache
func (cw *CacheWarmer) warmL2Keys(ctx context.Context, keys []string) error {
	// Use batch loading for efficiency
	keyMap := make(map[string]interface{})
	
	for _, key := range keys {
		data, err := cw.mockLoader(key)
		if err != nil {
			continue // Skip failed loads
		}
		keyMap[key] = data
	}
	
	return cw.cache.l2Cache.SetBatch(ctx, keyMap, time.Hour)
}

// warmL3Keys warms keys in L3 cache
func (cw *CacheWarmer) warmL3Keys(ctx context.Context, keys []string) error {
	for _, key := range keys {
		if cw.cache.isStaticContent(key) {
			_, err := cw.cache.l3Cache.Get(ctx, key)
			if err != nil {
				log.Printf("L3 warming failed for key %s: %v", key, err)
			}
		}
	}
	return nil
}

// isHighFrequencyKey determines if a key is accessed frequently
func (cw *CacheWarmer) isHighFrequencyKey(key string) bool {
	// Keys that are typically accessed frequently
	highFreqPrefixes := []string{
		"patient_context:",
		"phenotype_definition:diabetes",
		"phenotype_definition:hypertension",
		"risk_assessment:",
	}
	
	for _, prefix := range highFreqPrefixes {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// mockLoader provides mock data for cache warming
func (cw *CacheWarmer) mockLoader(key string) (interface{}, error) {
	// In production, this would load from actual data sources
	// For warming purposes, we create appropriate mock objects
	
	if isPatientContextKey(key) {
		return &models.ClinicalContext{
			PatientID:   extractPatientIDFromKey(key),
			GeneratedAt: time.Now(),
			ContextSummary: models.ContextSummary{
				KeyFindings: []string{"Warmed context"},
			},
		}, nil
	}
	
	if isPhenotypeDefinitionKey(key) {
		return &models.PhenotypeDefinition{
			Name:        extractNameFromKey(key),
			Description: "Warmed phenotype definition",
			Category:    "clinical",
			CELRule:     "true",
			Priority:    1,
			Version:     "1.0",
			CreatedAt:   time.Now(),
		}, nil
	}
	
	// Generic data for other keys
	return map[string]interface{}{
		"key":        key,
		"warmed_at":  time.Now(),
		"data_type":  "generic",
	}, nil
}

// Utility functions

// extractNameFromKey extracts name from cache key
func extractNameFromKey(key string) string {
	parts := strings.SplitN(key, ":", 2)
	if len(parts) == 2 {
		return strings.ReplaceAll(parts[1], "_", " ")
	}
	return key
}

// extractPatientIDFromKey extracts patient ID from context key
func extractPatientIDFromKey(key string) string {
	// Expected format: "patient_context:patient_id:context_type"
	parts := strings.Split(key, ":")
	if len(parts) >= 2 {
		return parts[1]
	}
	return "unknown"
}

// isPatientContextKey checks if key is for patient context
func isPatientContextKey(key string) bool {
	return strings.HasPrefix(key, "patient_context:")
}

// isPhenotypeDefinitionKey checks if key is for phenotype definition
func isPhenotypeDefinitionKey(key string) bool {
	return strings.HasPrefix(key, "phenotype_definition:")
}

// Performance monitoring

// GetWarmingEffectiveness calculates warming effectiveness
func (cw *CacheWarmer) GetWarmingEffectiveness() float64 {
	cw.mu.RLock()
	defer cw.mu.RUnlock()
	
	if cw.warmingRuns == 0 {
		return 0.0
	}
	
	// Calculate success rate (runs without errors)
	successfulRuns := cw.warmingRuns - cw.warmingErrors
	return float64(successfulRuns) / float64(cw.warmingRuns)
}

// GetAverageItemsPerRun returns average items warmed per run
func (cw *CacheWarmer) GetAverageItemsPerRun() float64 {
	cw.mu.RLock()
	defer cw.mu.RUnlock()
	
	if cw.warmingRuns == 0 {
		return 0.0
	}
	
	return float64(cw.itemsWarmed) / float64(cw.warmingRuns)
}

// ScheduledWarming manages scheduled cache warming
func (cw *CacheWarmer) ScheduledWarming(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := cw.WarmCache(ctx); err != nil {
				log.Printf("Scheduled cache warming failed: %v", err)
			}
		}
	}
}