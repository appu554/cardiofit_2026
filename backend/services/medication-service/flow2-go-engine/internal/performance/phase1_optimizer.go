package performance

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"flow2-go-engine/internal/clients"
	"flow2-go-engine/internal/models"
)

// Phase1Optimizer implements performance optimizations to meet the 25ms SLA
// for Phase 1 ORB evaluation and recipe resolution
type Phase1Optimizer struct {
	// In-memory caches for sub-millisecond access
	orbRuleCache     *ORBRuleCache
	recipeCache      *FastRecipeCache
	protocolCache    *ProtocolCache
	
	// Parallel processing pools
	workerPool       *WorkerPool
	
	// Performance monitoring
	slaMonitor       *SLAMonitor
	logger           *logrus.Logger
}

// ORBRuleCache provides ultra-fast ORB rule access
type ORBRuleCache struct {
	mu                sync.RWMutex
	rulesByCondition  map[string][]*CachedRule
	rulesByMedication map[string][]*CachedRule
	allRules         []*CachedRule
	lastReload       time.Time
}

// CachedRule is an optimized rule structure for fast evaluation
type CachedRule struct {
	ID               string
	Priority         int
	ConditionHash    uint64  // Pre-computed hash for fast matching
	MedicationCode   string
	CompiledConditions []CompiledCondition
	IntentTemplate   *models.IntentManifest
}

// CompiledCondition represents a pre-compiled condition for fast evaluation
type CompiledCondition struct {
	Type      string // AGE, COMORBIDITY, MEDICATION, etc.
	Operator  string // EQ, GT, LT, CONTAINS, etc.
	Value     interface{}
	ValueHash uint64 // Pre-computed hash for string values
}

// FastRecipeCache provides millisecond recipe access
type FastRecipeCache struct {
	mu              sync.RWMutex
	contextRecipes  map[string]*CachedContextRecipe
	clinicalRecipes map[string]*CachedClinicalRecipe
	lastUpdate      time.Time
}

// CachedContextRecipe is an optimized context recipe
type CachedContextRecipe struct {
	ID                string
	ProtocolID        string
	PrecomputedFields []models.FieldRequirement
	ConditionalRules  []FastConditionalRule
	FreshnessProfile  *FreshnessProfile
}

// CachedClinicalRecipe is an optimized clinical recipe
type CachedClinicalRecipe struct {
	ID                string
	ProtocolID        string
	TherapyOptions    []models.TherapyCandidate
	MonitoringFields  []models.FieldRequirement
	SafetyProfile     *SafetyProfile
}

// FastConditionalRule is a pre-compiled conditional rule
type FastConditionalRule struct {
	ConditionHash     uint64
	Evaluator        func(*models.MedicationRequest) bool
	AdditionalFields []models.FieldRequirement
}

// FreshnessProfile contains pre-computed freshness requirements
type FreshnessProfile struct {
	MaxAgeSeconds    int
	CriticalFields   []string
	SnapshotTTL      int
}

// SafetyProfile contains pre-computed safety requirements
type SafetyProfile struct {
	RequiredChecks   []string
	MonitoringParams []string
	Contraindications []string
}

// ProtocolCache provides instant protocol lookup
type ProtocolCache struct {
	mu            sync.RWMutex
	protocols     map[string]*CachedProtocol
	lastUpdate    time.Time
}

// CachedProtocol is an optimized protocol structure
type CachedProtocol struct {
	ID            string
	Version       string
	EvidenceGrade string
	Category      string
}

// WorkerPool manages parallel processing for Phase 1 operations
type WorkerPool struct {
	workers    int
	taskQueue  chan Task
	resultPool sync.Pool
	wg         sync.WaitGroup
}

// Task represents a parallel processing task
type Task struct {
	Type    string
	Data    interface{}
	Result  chan TaskResult
}

// TaskResult represents the result of a parallel task
type TaskResult struct {
	Data  interface{}
	Error error
}

// SLAMonitor tracks Phase 1 performance against the 25ms SLA
type SLAMonitor struct {
	mu                 sync.RWMutex
	totalRequests      int64
	slaViolations      int64
	averageLatencyMs   float64
	p95LatencyMs       float64
	p99LatencyMs       float64
	latencyHistory     []float64
	maxHistorySize     int
}

// NewPhase1Optimizer creates a new performance optimizer
func NewPhase1Optimizer(logger *logrus.Logger) *Phase1Optimizer {
	return &Phase1Optimizer{
		orbRuleCache:  NewORBRuleCache(),
		recipeCache:   NewFastRecipeCache(),
		protocolCache: NewProtocolCache(),
		workerPool:    NewWorkerPool(4), // 4 parallel workers
		slaMonitor:    NewSLAMonitor(),
		logger:        logger,
	}
}

// PreloadKnowledge preloads and optimizes all knowledge for ultra-fast access
func (p *Phase1Optimizer) PreloadKnowledge(ctx context.Context, apolloClient *clients.ApolloFederationClient) error {
	start := time.Now()
	
	p.logger.Info("Preloading and optimizing knowledge for Phase 1 performance")
	
	// Use parallel loading for maximum speed
	errChan := make(chan error, 3)
	
	go func() {
		errChan <- p.preloadORBRules(ctx, apolloClient)
	}()
	
	go func() {
		errChan <- p.preloadCommonRecipes(ctx, apolloClient)
	}()
	
	go func() {
		errChan <- p.preloadProtocols(ctx, apolloClient)
	}()
	
	// Wait for all preloading to complete
	for i := 0; i < 3; i++ {
		if err := <-errChan; err != nil {
			return fmt.Errorf("knowledge preloading failed: %w", err)
		}
	}
	
	loadTime := time.Since(start)
	p.logger.WithField("load_time_ms", loadTime.Milliseconds()).Info("Knowledge preloading completed")
	
	return nil
}

// OptimizedORBEvaluation performs ORB evaluation optimized for <15ms execution
func (p *Phase1Optimizer) OptimizedORBEvaluation(ctx context.Context, request *models.MedicationRequest) (*models.IntentManifest, error) {
	start := time.Now()
	
	// Fast path: check medication-based rules first
	if candidates := p.orbRuleCache.GetRulesByMedication(request.Indication); len(candidates) > 0 {
		for _, rule := range candidates {
			if p.fastRuleEvaluation(rule, request) {
				manifest := p.generateOptimizedManifest(rule, request)
				
				// Track performance
				elapsed := time.Since(start)
				p.slaMonitor.RecordLatency(elapsed)
				
				return manifest, nil
			}
		}
	}
	
	// Fallback: evaluate all rules in parallel
	manifest, err := p.parallelRuleEvaluation(ctx, request)
	if err != nil {
		return nil, err
	}
	
	// Track performance
	elapsed := time.Since(start)
	p.slaMonitor.RecordLatency(elapsed)
	
	if elapsed.Milliseconds() > 15 {
		p.logger.WithFields(logrus.Fields{
			"elapsed_ms": elapsed.Milliseconds(),
			"request_id": request.RequestID,
		}).Warn("ORB evaluation exceeded 15ms target")
	}
	
	return manifest, nil
}

// OptimizedRecipeResolution performs recipe resolution optimized for <10ms execution
func (p *Phase1Optimizer) OptimizedRecipeResolution(ctx context.Context, manifest *models.IntentManifest, request *models.MedicationRequest) error {
	start := time.Now()
	
	// Fast path: use cached recipes
	contextRecipe, exists := p.recipeCache.GetContextRecipe(manifest.ProtocolID)
	if !exists {
		return fmt.Errorf("context recipe not found in cache: %s", manifest.ProtocolID)
	}
	
	clinicalRecipe, exists := p.recipeCache.GetClinicalRecipe(manifest.ProtocolID)
	if !exists {
		return fmt.Errorf("clinical recipe not found in cache: %s", manifest.ProtocolID)
	}
	
	// Parallel field resolution
	requiredFields := p.parallelFieldResolution(contextRecipe, request)
	
	// Update manifest with pre-computed values
	manifest.ContextRecipeID = contextRecipe.ID
	manifest.ClinicalRecipeID = clinicalRecipe.ID
	manifest.RequiredFields = requiredFields
	manifest.OptionalFields = []models.FieldRequirement{} // Computed separately if needed
	manifest.DataFreshness = models.FreshnessRequirements{
		MaxAge:         time.Duration(contextRecipe.FreshnessProfile.MaxAgeSeconds) * time.Second,
		CriticalFields: contextRecipe.FreshnessProfile.CriticalFields,
	}
	manifest.SnapshotTTL = contextRecipe.FreshnessProfile.SnapshotTTL
	manifest.TherapyOptions = clinicalRecipe.TherapyOptions
	
	// Track performance
	elapsed := time.Since(start)
	if elapsed.Milliseconds() > 10 {
		p.logger.WithFields(logrus.Fields{
			"elapsed_ms": elapsed.Milliseconds(),
			"protocol_id": manifest.ProtocolID,
		}).Warn("Recipe resolution exceeded 10ms target")
	}
	
	return nil
}

// fastRuleEvaluation performs ultra-fast rule evaluation using pre-compiled conditions
func (p *Phase1Optimizer) fastRuleEvaluation(rule *CachedRule, request *models.MedicationRequest) bool {
	// Fast medication code check
	if rule.MedicationCode != "" && rule.MedicationCode != request.Indication {
		return false
	}
	
	// Fast condition evaluation using pre-compiled conditions
	for _, condition := range rule.CompiledConditions {
		if !p.evaluateCompiledCondition(condition, request) {
			return false
		}
	}
	
	return true
}

// evaluateCompiledCondition evaluates a pre-compiled condition
func (p *Phase1Optimizer) evaluateCompiledCondition(condition CompiledCondition, request *models.MedicationRequest) bool {
	switch condition.Type {
	case "AGE":
		if request.ClinicalContext.Age == 0 {
			return false
		}
		return p.compareNumeric(request.ClinicalContext.Age, condition.Operator, condition.Value.(float64))
		
	case "COMORBIDITY":
		targetHash := condition.ValueHash
		for _, comorbidity := range request.ClinicalContext.Comorbidities {
			if p.fastStringHash(comorbidity) == targetHash {
				return condition.Operator == "CONTAINS"
			}
		}
		return condition.Operator == "NOT_CONTAINS"
		
	case "WEIGHT":
		if request.ClinicalContext.Weight == 0 {
			return false
		}
		return p.compareNumeric(request.ClinicalContext.Weight, condition.Operator, condition.Value.(float64))
		
	default:
		return false
	}
}

// compareNumeric performs fast numeric comparison
func (p *Phase1Optimizer) compareNumeric(value float64, operator string, target float64) bool {
	switch operator {
	case "GT":
		return value > target
	case "GTE":
		return value >= target
	case "LT":
		return value < target
	case "LTE":
		return value <= target
	case "EQ":
		return value == target
	default:
		return false
	}
}

// fastStringHash computes a fast hash for string comparison
func (p *Phase1Optimizer) fastStringHash(s string) uint64 {
	// Simple FNV-1a hash for fast string comparison
	hash := uint64(14695981039346656037)
	for i := 0; i < len(s); i++ {
		hash ^= uint64(s[i])
		hash *= 1099511628211
	}
	return hash
}

// parallelRuleEvaluation evaluates multiple rules in parallel
func (p *Phase1Optimizer) parallelRuleEvaluation(ctx context.Context, request *models.MedicationRequest) (*models.IntentManifest, error) {
	rules := p.orbRuleCache.GetAllRules()
	
	// Split rules across workers
	ruleChunks := p.splitRules(rules, p.workerPool.workers)
	
	resultChan := make(chan *models.IntentManifest, len(ruleChunks))
	errorChan := make(chan error, len(ruleChunks))
	
	// Evaluate rule chunks in parallel
	for _, chunk := range ruleChunks {
		go func(ruleChunk []*CachedRule) {
			for _, rule := range ruleChunk {
				if p.fastRuleEvaluation(rule, request) {
					manifest := p.generateOptimizedManifest(rule, request)
					resultChan <- manifest
					return
				}
			}
			resultChan <- nil
		}(chunk)
	}
	
	// Get the first successful result
	for i := 0; i < len(ruleChunks); i++ {
		select {
		case manifest := <-resultChan:
			if manifest != nil {
				return manifest, nil
			}
		case err := <-errorChan:
			return nil, err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	
	return nil, fmt.Errorf("no matching ORB rule found")
}

// generateOptimizedManifest generates an intent manifest using pre-computed data
func (p *Phase1Optimizer) generateOptimizedManifest(rule *CachedRule, request *models.MedicationRequest) *models.IntentManifest {
	manifest := &models.IntentManifest{
		ManifestID:   p.generateULID(),
		RequestID:    request.RequestID,
		GeneratedAt:  time.Now(),
		ProtocolID:   rule.ID, // Simplified for performance
		ORBVersion:   "2.1.0",
		RulesApplied: []models.AppliedRule{{
			RuleID:     rule.ID,
			Confidence: 0.95, // Pre-computed confidence
			AppliedAt:  time.Now(),
		}},
	}
	
	// Copy pre-computed values from rule template
	if rule.IntentTemplate != nil {
		manifest.PrimaryIntent = rule.IntentTemplate.PrimaryIntent
		manifest.TherapyOptions = rule.IntentTemplate.TherapyOptions
	}
	
	return manifest
}

// parallelFieldResolution resolves fields in parallel for maximum speed
func (p *Phase1Optimizer) parallelFieldResolution(recipe *CachedContextRecipe, request *models.MedicationRequest) []models.FieldRequirement {
	// Start with pre-computed fields
	result := make([]models.FieldRequirement, len(recipe.PrecomputedFields))
	copy(result, recipe.PrecomputedFields)
	
	// Evaluate conditional rules in parallel
	if len(recipe.ConditionalRules) > 0 {
		additionalFields := make(chan []models.FieldRequirement, len(recipe.ConditionalRules))
		
		for _, rule := range recipe.ConditionalRules {
			go func(r FastConditionalRule) {
				if r.Evaluator(request) {
					additionalFields <- r.AdditionalFields
				} else {
					additionalFields <- nil
				}
			}(rule)
		}
		
		// Collect results
		for i := 0; i < len(recipe.ConditionalRules); i++ {
			if fields := <-additionalFields; fields != nil {
				result = append(result, fields...)
			}
		}
	}
	
	return result
}

// Utility methods

func (p *Phase1Optimizer) splitRules(rules []*CachedRule, chunks int) [][]*CachedRule {
	if len(rules) < chunks {
		chunks = len(rules)
	}
	
	chunkSize := len(rules) / chunks
	result := make([][]*CachedRule, chunks)
	
	for i := 0; i < chunks; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if i == chunks-1 {
			end = len(rules)
		}
		result[i] = rules[start:end]
	}
	
	return result
}

func (p *Phase1Optimizer) generateULID() string {
	// Fast ULID generation for performance
	return fmt.Sprintf("01%d", time.Now().UnixNano())
}

// Initialization methods

func NewORBRuleCache() *ORBRuleCache {
	return &ORBRuleCache{
		rulesByCondition:  make(map[string][]*CachedRule),
		rulesByMedication: make(map[string][]*CachedRule),
		allRules:         make([]*CachedRule, 0),
	}
}

func NewFastRecipeCache() *FastRecipeCache {
	return &FastRecipeCache{
		contextRecipes:  make(map[string]*CachedContextRecipe),
		clinicalRecipes: make(map[string]*CachedClinicalRecipe),
	}
}

func NewProtocolCache() *ProtocolCache {
	return &ProtocolCache{
		protocols: make(map[string]*CachedProtocol),
	}
}

func NewWorkerPool(workers int) *WorkerPool {
	return &WorkerPool{
		workers:   workers,
		taskQueue: make(chan Task, workers*2),
	}
}

func NewSLAMonitor() *SLAMonitor {
	return &SLAMonitor{
		maxHistorySize: 1000,
		latencyHistory: make([]float64, 0, 1000),
	}
}

// Cache getter methods

func (c *ORBRuleCache) GetRulesByMedication(medicationCode string) []*CachedRule {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.rulesByMedication[medicationCode]
}

func (c *ORBRuleCache) GetAllRules() []*CachedRule {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.allRules
}

func (c *FastRecipeCache) GetContextRecipe(protocolID string) (*CachedContextRecipe, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	recipe, exists := c.contextRecipes[protocolID]
	return recipe, exists
}

func (c *FastRecipeCache) GetClinicalRecipe(protocolID string) (*CachedClinicalRecipe, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	recipe, exists := c.clinicalRecipes[protocolID]
	return recipe, exists
}

// Performance monitoring methods

func (s *SLAMonitor) RecordLatency(duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	latencyMs := float64(duration.Milliseconds())
	s.totalRequests++
	
	// Track SLA violations (>25ms)
	if latencyMs > 25 {
		s.slaViolations++
	}
	
	// Update average latency
	totalLatency := s.averageLatencyMs * float64(s.totalRequests-1)
	s.averageLatencyMs = (totalLatency + latencyMs) / float64(s.totalRequests)
	
	// Update latency history for percentile calculation
	s.latencyHistory = append(s.latencyHistory, latencyMs)
	if len(s.latencyHistory) > s.maxHistorySize {
		s.latencyHistory = s.latencyHistory[1:]
	}
}

func (s *SLAMonitor) GetSLACompliance() float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if s.totalRequests == 0 {
		return 100.0
	}
	
	compliance := float64(s.totalRequests-s.slaViolations) / float64(s.totalRequests) * 100
	return compliance
}

// Preloading stub methods (would be implemented with actual Apollo queries)

func (p *Phase1Optimizer) preloadORBRules(ctx context.Context, apolloClient *clients.ApolloFederationClient) error {
	// Implementation would load and optimize ORB rules
	return nil
}

func (p *Phase1Optimizer) preloadCommonRecipes(ctx context.Context, apolloClient *clients.ApolloFederationClient) error {
	// Implementation would load and optimize common recipes
	return nil
}

func (p *Phase1Optimizer) preloadProtocols(ctx context.Context, apolloClient *clients.ApolloFederationClient) error {
	// Implementation would load and optimize protocols
	return nil
}