package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/clinical-synthesis-hub/cardiofit/safety-gateway-platform/pkg/types"
	"github.com/clinical-synthesis-hub/cardiofit/safety-gateway-platform/internal/config"
	"github.com/clinical-synthesis-hub/cardiofit/safety-gateway-platform/pkg/logger"
)

// PredictivePreWarmingSystem implements ML-based cache preloading
type PredictivePreWarmingSystem struct {
	cacheManager     *AdvancedCacheManager
	patternAnalyzer  *AccessPatternAnalyzer
	predictor        *SnapshotPredictor
	scheduler        *PreWarmScheduler
	config           *config.PreWarmingConfig
	logger           *logger.Logger
	mu               sync.RWMutex
	isRunning        bool
	stopCh           chan struct{}
	metrics          *PreWarmingMetrics
}

// AccessPattern represents a detected usage pattern
type AccessPattern struct {
	PatientID       string                 `json:"patient_id"`
	TimeOfDay       []int                 `json:"time_of_day"`     // Hours 0-23
	DayOfWeek       []int                 `json:"day_of_week"`     // 0-6 (Sunday-Saturday)
	AccessFrequency float64               `json:"access_frequency"`
	ContextHints    map[string]interface{} `json:"context_hints"`
	Confidence      float64               `json:"confidence"`
	LastAccessed    time.Time             `json:"last_accessed"`
	PredictedNext   time.Time             `json:"predicted_next"`
}

// SnapshotPredictor uses machine learning to predict future snapshot needs
type SnapshotPredictor struct {
	patterns        map[string]*AccessPattern
	timeSeriesData  map[string][]AccessPoint
	modelWeights    map[string]float64
	config          *config.PredictorConfig
	mu              sync.RWMutex
}

// AccessPoint represents a single access event
type AccessPoint struct {
	Timestamp   time.Time             `json:"timestamp"`
	PatientID   string               `json:"patient_id"`
	SnapshotID  string               `json:"snapshot_id"`
	Context     map[string]interface{} `json:"context"`
	CacheHit    bool                 `json:"cache_hit"`
	ResponseTime time.Duration        `json:"response_time"`
}

// PreWarmScheduler manages the timing and execution of prewarming operations
type PreWarmScheduler struct {
	prewarmQueue    chan *PreWarmRequest
	workers         []*PreWarmWorker
	config          *config.SchedulerConfig
	logger          *logger.Logger
	mu              sync.RWMutex
	activeRequests  map[string]*PreWarmRequest
}

// PreWarmRequest represents a request to prewarm cache
type PreWarmRequest struct {
	ID              string                `json:"id"`
	PatientID       string               `json:"patient_id"`
	Priority        int                  `json:"priority"`
	PredictedTime   time.Time            `json:"predicted_time"`
	ScheduledTime   time.Time            `json:"scheduled_time"`
	Context         map[string]interface{} `json:"context"`
	Confidence      float64              `json:"confidence"`
	Status          PreWarmStatus        `json:"status"`
	CreatedAt       time.Time            `json:"created_at"`
	CompletedAt     *time.Time           `json:"completed_at,omitempty"`
	Error           string               `json:"error,omitempty"`
}

type PreWarmStatus int

const (
	PreWarmStatusPending PreWarmStatus = iota
	PreWarmStatusScheduled
	PreWarmStatusExecuting
	PreWarmStatusCompleted
	PreWarmStatusFailed
	PreWarmStatusCancelled
)

// PreWarmWorker executes prewarming operations
type PreWarmWorker struct {
	id           int
	requests     chan *PreWarmRequest
	cacheManager *AdvancedCacheManager
	logger       *logger.Logger
	stopCh       chan struct{}
	isRunning    bool
}

// PreWarmingMetrics tracks prewarming performance
type PreWarmingMetrics struct {
	TotalRequests       int64   `json:"total_requests"`
	SuccessfulPredictions int64 `json:"successful_predictions"`
	FailedPredictions   int64   `json:"failed_predictions"`
	AverageConfidence   float64 `json:"average_confidence"`
	CacheHitImprovement float64 `json:"cache_hit_improvement"`
	PreWarmingLatency   time.Duration `json:"prewarming_latency"`
	
	// Time-based metrics
	HourlyPatterns      map[int]int64     `json:"hourly_patterns"`
	DailyPatterns       map[int]int64     `json:"daily_patterns"`
	PatternAccuracy     map[string]float64 `json:"pattern_accuracy"`
	
	mu sync.RWMutex
}

// NewPredictivePreWarmingSystem creates a new predictive prewarming system
func NewPredictivePreWarmingSystem(
	cacheManager *AdvancedCacheManager,
	config *config.PreWarmingConfig,
	logger *logger.Logger,
) *PredictivePreWarmingSystem {
	system := &PredictivePreWarmingSystem{
		cacheManager:    cacheManager,
		patternAnalyzer: NewAccessPatternAnalyzer(config.PatternAnalyzer, logger),
		predictor:       NewSnapshotPredictor(config.Predictor, logger),
		scheduler:       NewPreWarmScheduler(config.Scheduler, logger),
		config:          config,
		logger:          logger,
		stopCh:          make(chan struct{}),
		metrics:         NewPreWarmingMetrics(),
	}
	
	return system
}

// Start initializes and starts the predictive prewarming system
func (p *PredictivePreWarmingSystem) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.isRunning {
		return fmt.Errorf("predictive prewarming system is already running")
	}
	
	p.logger.Info("Starting predictive prewarming system")
	
	// Start components
	if err := p.patternAnalyzer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start pattern analyzer: %w", err)
	}
	
	if err := p.predictor.Start(ctx); err != nil {
		return fmt.Errorf("failed to start predictor: %w", err)
	}
	
	if err := p.scheduler.Start(ctx); err != nil {
		return fmt.Errorf("failed to start scheduler: %w", err)
	}
	
	// Start main loop
	go p.mainLoop(ctx)
	
	p.isRunning = true
	p.logger.Info("Predictive prewarming system started successfully")
	
	return nil
}

// Stop gracefully shuts down the predictive prewarming system
func (p *PredictivePreWarmingSystem) Stop(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if !p.isRunning {
		return nil
	}
	
	p.logger.Info("Stopping predictive prewarming system")
	
	close(p.stopCh)
	
	// Stop components
	p.scheduler.Stop(ctx)
	p.predictor.Stop(ctx)
	p.patternAnalyzer.Stop(ctx)
	
	p.isRunning = false
	p.logger.Info("Predictive prewarming system stopped")
	
	return nil
}

// RecordAccess records a snapshot access for pattern analysis
func (p *PredictivePreWarmingSystem) RecordAccess(
	patientID string,
	snapshotID string,
	context map[string]interface{},
	cacheHit bool,
	responseTime time.Duration,
) {
	accessPoint := &AccessPoint{
		Timestamp:    time.Now(),
		PatientID:    patientID,
		SnapshotID:   snapshotID,
		Context:      context,
		CacheHit:     cacheHit,
		ResponseTime: responseTime,
	}
	
	p.patternAnalyzer.RecordAccess(accessPoint)
	p.updateMetrics(accessPoint)
}

// mainLoop runs the main predictive prewarming logic
func (p *PredictivePreWarmingSystem) mainLoop(ctx context.Context) {
	ticker := time.NewTicker(p.config.AnalysisInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.analyzeAndPredict()
		}
	}
}

// analyzeAndPredict performs pattern analysis and generates predictions
func (p *PredictivePreWarmingSystem) analyzeAndPredict() {
	p.logger.Debug("Starting pattern analysis and prediction")
	
	// Analyze current patterns
	patterns := p.patternAnalyzer.AnalyzePatterns()
	
	// Generate predictions
	predictions := p.predictor.GeneratePredictions(patterns)
	
	// Schedule prewarming requests
	for _, prediction := range predictions {
		if prediction.Confidence >= p.config.MinimumConfidence {
			request := &PreWarmRequest{
				ID:            fmt.Sprintf("prewarm_%d_%s", time.Now().Unix(), prediction.PatientID),
				PatientID:     prediction.PatientID,
				Priority:      p.calculatePriority(prediction),
				PredictedTime: prediction.PredictedNext,
				ScheduledTime: prediction.PredictedNext.Add(-p.config.PreWarmLeadTime),
				Context:       prediction.ContextHints,
				Confidence:    prediction.Confidence,
				Status:        PreWarmStatusPending,
				CreatedAt:     time.Now(),
			}
			
			p.scheduler.SchedulePreWarm(request)
		}
	}
	
	p.logger.Debug("Pattern analysis and prediction completed", 
		"patterns", len(patterns), 
		"predictions", len(predictions))
}

// calculatePriority calculates the priority for a prewarming request
func (p *PredictivePreWarmingSystem) calculatePriority(pattern *AccessPattern) int {
	// Higher confidence and frequency = higher priority
	priority := int(pattern.Confidence * 100 * pattern.AccessFrequency)
	
	// Time-based adjustments
	now := time.Now()
	timeUntilPredicted := pattern.PredictedNext.Sub(now)
	
	if timeUntilPredicted < 5*time.Minute {
		priority += 50 // Urgent
	} else if timeUntilPredicted < 15*time.Minute {
		priority += 25 // High
	}
	
	return priority
}

// updateMetrics updates prewarming metrics
func (p *PredictivePreWarmingSystem) updateMetrics(access *AccessPoint) {
	p.metrics.mu.Lock()
	defer p.metrics.mu.Unlock()
	
	p.metrics.TotalRequests++
	
	hour := access.Timestamp.Hour()
	day := int(access.Timestamp.Weekday())
	
	if p.metrics.HourlyPatterns == nil {
		p.metrics.HourlyPatterns = make(map[int]int64)
	}
	if p.metrics.DailyPatterns == nil {
		p.metrics.DailyPatterns = make(map[int]int64)
	}
	
	p.metrics.HourlyPatterns[hour]++
	p.metrics.DailyPatterns[day]++
}

// GetMetrics returns current prewarming metrics
func (p *PredictivePreWarmingSystem) GetMetrics() *PreWarmingMetrics {
	p.metrics.mu.RLock()
	defer p.metrics.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	metrics := &PreWarmingMetrics{
		TotalRequests:       p.metrics.TotalRequests,
		SuccessfulPredictions: p.metrics.SuccessfulPredictions,
		FailedPredictions:   p.metrics.FailedPredictions,
		AverageConfidence:   p.metrics.AverageConfidence,
		CacheHitImprovement: p.metrics.CacheHitImprovement,
		PreWarmingLatency:   p.metrics.PreWarmingLatency,
		HourlyPatterns:      make(map[int]int64),
		DailyPatterns:       make(map[int]int64),
		PatternAccuracy:     make(map[string]float64),
	}
	
	for k, v := range p.metrics.HourlyPatterns {
		metrics.HourlyPatterns[k] = v
	}
	for k, v := range p.metrics.DailyPatterns {
		metrics.DailyPatterns[k] = v
	}
	for k, v := range p.metrics.PatternAccuracy {
		metrics.PatternAccuracy[k] = v
	}
	
	return metrics
}

// NewAccessPatternAnalyzer creates a new access pattern analyzer
func NewAccessPatternAnalyzer(config *config.PatternAnalyzerConfig, logger *logger.Logger) *AccessPatternAnalyzer {
	return &AccessPatternAnalyzer{
		accessHistory:   make(map[string][]AccessPoint),
		patterns:        make(map[string]*AccessPattern),
		config:          config,
		logger:          logger,
	}
}

// AccessPatternAnalyzer analyzes access patterns to detect trends
type AccessPatternAnalyzer struct {
	accessHistory map[string][]AccessPoint
	patterns      map[string]*AccessPattern
	config        *config.PatternAnalyzerConfig
	logger        *logger.Logger
	mu            sync.RWMutex
}

// Start initializes the pattern analyzer
func (a *AccessPatternAnalyzer) Start(ctx context.Context) error {
	a.logger.Info("Access pattern analyzer started")
	return nil
}

// Stop shuts down the pattern analyzer
func (a *AccessPatternAnalyzer) Stop(ctx context.Context) error {
	a.logger.Info("Access pattern analyzer stopped")
	return nil
}

// RecordAccess records an access event for analysis
func (a *AccessPatternAnalyzer) RecordAccess(access *AccessPoint) {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	if a.accessHistory[access.PatientID] == nil {
		a.accessHistory[access.PatientID] = make([]AccessPoint, 0)
	}
	
	a.accessHistory[access.PatientID] = append(a.accessHistory[access.PatientID], *access)
	
	// Keep only recent history
	if len(a.accessHistory[access.PatientID]) > a.config.MaxHistorySize {
		start := len(a.accessHistory[access.PatientID]) - a.config.MaxHistorySize
		a.accessHistory[access.PatientID] = a.accessHistory[access.PatientID][start:]
	}
}

// AnalyzePatterns analyzes access history to detect patterns
func (a *AccessPatternAnalyzer) AnalyzePatterns() map[string]*AccessPattern {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	patterns := make(map[string]*AccessPattern)
	
	for patientID, history := range a.accessHistory {
		if len(history) < a.config.MinHistorySize {
			continue
		}
		
		pattern := a.analyzePatientPattern(patientID, history)
		if pattern != nil && pattern.Confidence >= a.config.MinConfidence {
			patterns[patientID] = pattern
		}
	}
	
	a.patterns = patterns
	return patterns
}

// analyzePatientPattern analyzes patterns for a specific patient
func (a *AccessPatternAnalyzer) analyzePatientPattern(patientID string, history []AccessPoint) *AccessPattern {
	if len(history) == 0 {
		return nil
	}
	
	// Analyze temporal patterns
	hourFreq := make(map[int]int)
	dayFreq := make(map[int]int)
	
	var totalAccesses int
	var lastAccess time.Time
	
	for _, access := range history {
		hour := access.Timestamp.Hour()
		day := int(access.Timestamp.Weekday())
		
		hourFreq[hour]++
		dayFreq[day]++
		totalAccesses++
		
		if access.Timestamp.After(lastAccess) {
			lastAccess = access.Timestamp
		}
	}
	
	// Find most common hours and days
	commonHours := a.findTopFrequent(hourFreq, 3)
	commonDays := a.findTopFrequent(dayFreq, 3)
	
	// Calculate access frequency (accesses per day)
	timeSpan := history[len(history)-1].Timestamp.Sub(history[0].Timestamp)
	accessFrequency := float64(totalAccesses) / timeSpan.Hours() * 24
	
	// Calculate confidence based on pattern consistency
	confidence := a.calculatePatternConfidence(hourFreq, dayFreq, totalAccesses)
	
	// Predict next access time
	nextAccess := a.predictNextAccess(commonHours, commonDays, lastAccess)
	
	return &AccessPattern{
		PatientID:       patientID,
		TimeOfDay:       commonHours,
		DayOfWeek:       commonDays,
		AccessFrequency: accessFrequency,
		ContextHints:    a.extractContextHints(history),
		Confidence:      confidence,
		LastAccessed:    lastAccess,
		PredictedNext:   nextAccess,
	}
}

// findTopFrequent finds the most frequent items
func (a *AccessPatternAnalyzer) findTopFrequent(freq map[int]int, top int) []int {
	type kv struct {
		Key   int
		Value int
	}
	
	var items []kv
	for k, v := range freq {
		items = append(items, kv{k, v})
	}
	
	sort.Slice(items, func(i, j int) bool {
		return items[i].Value > items[j].Value
	})
	
	var result []int
	for i := 0; i < len(items) && i < top; i++ {
		result = append(result, items[i].Key)
	}
	
	return result
}

// calculatePatternConfidence calculates confidence based on pattern consistency
func (a *AccessPatternAnalyzer) calculatePatternConfidence(hourFreq, dayFreq map[int]int, totalAccesses int) float64 {
	// Calculate entropy for hours and days
	hourEntropy := a.calculateEntropy(hourFreq, totalAccesses)
	dayEntropy := a.calculateEntropy(dayFreq, totalAccesses)
	
	// Lower entropy = higher confidence (more predictable)
	maxHourEntropy := math.Log2(24) // Maximum entropy for 24 hours
	maxDayEntropy := math.Log2(7)   // Maximum entropy for 7 days
	
	hourConfidence := 1.0 - (hourEntropy / maxHourEntropy)
	dayConfidence := 1.0 - (dayEntropy / maxDayEntropy)
	
	// Combined confidence
	return (hourConfidence + dayConfidence) / 2.0
}

// calculateEntropy calculates Shannon entropy
func (a *AccessPatternAnalyzer) calculateEntropy(freq map[int]int, total int) float64 {
	var entropy float64
	for _, count := range freq {
		if count > 0 {
			p := float64(count) / float64(total)
			entropy -= p * math.Log2(p)
		}
	}
	return entropy
}

// extractContextHints extracts common context patterns
func (a *AccessPatternAnalyzer) extractContextHints(history []AccessPoint) map[string]interface{} {
	hints := make(map[string]interface{})
	
	// This would extract common context patterns from the access history
	// For now, we'll return basic hints
	hints["avg_response_time"] = a.calculateAverageResponseTime(history)
	hints["cache_hit_rate"] = a.calculateCacheHitRate(history)
	
	return hints
}

// calculateAverageResponseTime calculates average response time
func (a *AccessPatternAnalyzer) calculateAverageResponseTime(history []AccessPoint) time.Duration {
	var total time.Duration
	for _, access := range history {
		total += access.ResponseTime
	}
	return total / time.Duration(len(history))
}

// calculateCacheHitRate calculates cache hit rate
func (a *AccessPatternAnalyzer) calculateCacheHitRate(history []AccessPoint) float64 {
	var hits int
	for _, access := range history {
		if access.CacheHit {
			hits++
		}
	}
	return float64(hits) / float64(len(history))
}

// predictNextAccess predicts the next access time based on patterns
func (a *AccessPatternAnalyzer) predictNextAccess(commonHours, commonDays []int, lastAccess time.Time) time.Time {
	now := time.Now()
	
	// Find next occurrence of common patterns
	var candidates []time.Time
	
	for _, hour := range commonHours {
		for _, day := range commonDays {
			// Find next occurrence of this day/hour combination
			next := a.findNextOccurrence(now, day, hour)
			candidates = append(candidates, next)
		}
	}
	
	// Return earliest candidate
	if len(candidates) == 0 {
		return now.Add(24 * time.Hour) // Default to 24 hours from now
	}
	
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Before(candidates[j])
	})
	
	return candidates[0]
}

// findNextOccurrence finds the next occurrence of a specific day and hour
func (a *AccessPatternAnalyzer) findNextOccurrence(from time.Time, targetDay, targetHour int) time.Time {
	// Start from the next hour to avoid immediate scheduling
	next := from.Add(time.Hour).Truncate(time.Hour)
	
	for {
		if int(next.Weekday()) == targetDay && next.Hour() == targetHour {
			return next
		}
		next = next.Add(time.Hour)
		
		// Prevent infinite loops - max 7 days ahead
		if next.Sub(from) > 7*24*time.Hour {
			break
		}
	}
	
	return from.Add(24 * time.Hour)
}

// NewSnapshotPredictor creates a new snapshot predictor
func NewSnapshotPredictor(config *config.PredictorConfig, logger *logger.Logger) *SnapshotPredictor {
	return &SnapshotPredictor{
		patterns:       make(map[string]*AccessPattern),
		timeSeriesData: make(map[string][]AccessPoint),
		modelWeights:   map[string]float64{
			"temporal":    0.4,
			"frequency":   0.3,
			"recency":     0.2,
			"context":     0.1,
		},
		config: config,
	}
}

// Start initializes the predictor
func (p *SnapshotPredictor) Start(ctx context.Context) error {
	return nil
}

// Stop shuts down the predictor
func (p *SnapshotPredictor) Stop(ctx context.Context) error {
	return nil
}

// GeneratePredictions generates predictions based on analyzed patterns
func (p *SnapshotPredictor) GeneratePredictions(patterns map[string]*AccessPattern) []*AccessPattern {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	var predictions []*AccessPattern
	
	for patientID, pattern := range patterns {
		// Apply machine learning weights to refine predictions
		refinedPattern := p.refinePattern(pattern)
		
		if refinedPattern.Confidence >= p.config.MinConfidence {
			predictions = append(predictions, refinedPattern)
			p.patterns[patientID] = refinedPattern
		}
	}
	
	return predictions
}

// refinePattern applies ML weights to refine pattern predictions
func (p *SnapshotPredictor) refinePattern(pattern *AccessPattern) *AccessPattern {
	refined := *pattern // Copy
	
	// Apply temporal weight
	temporalScore := p.calculateTemporalScore(pattern)
	
	// Apply frequency weight
	frequencyScore := p.calculateFrequencyScore(pattern)
	
	// Apply recency weight
	recencyScore := p.calculateRecencyScore(pattern)
	
	// Apply context weight
	contextScore := p.calculateContextScore(pattern)
	
	// Weighted confidence
	refined.Confidence = 
		temporalScore * p.modelWeights["temporal"] +
		frequencyScore * p.modelWeights["frequency"] +
		recencyScore * p.modelWeights["recency"] +
		contextScore * p.modelWeights["context"]
	
	// Ensure confidence is between 0 and 1
	if refined.Confidence > 1.0 {
		refined.Confidence = 1.0
	}
	if refined.Confidence < 0.0 {
		refined.Confidence = 0.0
	}
	
	return &refined
}

// calculateTemporalScore calculates temporal consistency score
func (p *SnapshotPredictor) calculateTemporalScore(pattern *AccessPattern) float64 {
	// Higher score for consistent temporal patterns
	if len(pattern.TimeOfDay) > 0 && len(pattern.DayOfWeek) > 0 {
		return pattern.Confidence * 1.2 // Boost for temporal patterns
	}
	return pattern.Confidence
}

// calculateFrequencyScore calculates frequency-based score
func (p *SnapshotPredictor) calculateFrequencyScore(pattern *AccessPattern) float64 {
	// Higher frequency = higher score
	normalized := math.Min(pattern.AccessFrequency / 10.0, 1.0) // Normalize to 0-1
	return normalized
}

// calculateRecencyScore calculates recency-based score
func (p *SnapshotPredictor) calculateRecencyScore(pattern *AccessPattern) float64 {
	// More recent accesses = higher score
	hoursSinceLastAccess := time.Since(pattern.LastAccessed).Hours()
	
	if hoursSinceLastAccess < 1 {
		return 1.0
	} else if hoursSinceLastAccess < 24 {
		return 0.8
	} else if hoursSinceLastAccess < 168 { // 1 week
		return 0.6
	}
	
	return 0.3
}

// calculateContextScore calculates context-based score
func (p *SnapshotPredictor) calculateContextScore(pattern *AccessPattern) float64 {
	// This would analyze context hints for additional signals
	// For now, return base confidence
	return pattern.Confidence
}

// NewPreWarmScheduler creates a new prewarming scheduler
func NewPreWarmScheduler(config *config.SchedulerConfig, logger *logger.Logger) *PreWarmScheduler {
	scheduler := &PreWarmScheduler{
		prewarmQueue:   make(chan *PreWarmRequest, config.QueueSize),
		config:         config,
		logger:         logger,
		activeRequests: make(map[string]*PreWarmRequest),
	}
	
	// Create worker pool
	for i := 0; i < config.WorkerCount; i++ {
		worker := &PreWarmWorker{
			id:       i,
			requests: make(chan *PreWarmRequest, 10),
			logger:   logger,
			stopCh:   make(chan struct{}),
		}
		scheduler.workers = append(scheduler.workers, worker)
	}
	
	return scheduler
}

// Start initializes the scheduler
func (s *PreWarmScheduler) Start(ctx context.Context) error {
	s.logger.Info("Starting prewarming scheduler")
	
	// Start workers
	for _, worker := range s.workers {
		go worker.start()
	}
	
	// Start request dispatcher
	go s.dispatcher()
	
	return nil
}

// Stop shuts down the scheduler
func (s *PreWarmScheduler) Stop(ctx context.Context) error {
	s.logger.Info("Stopping prewarming scheduler")
	
	// Stop workers
	for _, worker := range s.workers {
		worker.stop()
	}
	
	close(s.prewarmQueue)
	return nil
}

// SchedulePreWarm schedules a prewarming request
func (s *PreWarmScheduler) SchedulePreWarm(request *PreWarmRequest) {
	s.mu.Lock()
	s.activeRequests[request.ID] = request
	s.mu.Unlock()
	
	select {
	case s.prewarmQueue <- request:
		s.logger.Debug("Prewarming request scheduled", "request_id", request.ID)
	default:
		s.logger.Warn("Prewarming queue is full, dropping request", "request_id", request.ID)
	}
}

// dispatcher distributes requests to workers
func (s *PreWarmScheduler) dispatcher() {
	for request := range s.prewarmQueue {
		// Find the least busy worker
		worker := s.findBestWorker()
		if worker != nil {
			select {
			case worker.requests <- request:
				// Request dispatched
			default:
				// Worker queue is full, try another
				s.logger.Warn("Worker queue full, retrying", "worker_id", worker.id)
				// Put request back in main queue
				go func(r *PreWarmRequest) {
					time.Sleep(100 * time.Millisecond)
					s.prewarmQueue <- r
				}(request)
			}
		}
	}
}

// findBestWorker finds the worker with the smallest queue
func (s *PreWarmScheduler) findBestWorker() *PreWarmWorker {
	if len(s.workers) == 0 {
		return nil
	}
	
	// Simple round-robin for now
	// In production, this would track worker load
	return s.workers[time.Now().UnixNano()%int64(len(s.workers))]
}

// start starts the worker
func (w *PreWarmWorker) start() {
	w.isRunning = true
	for {
		select {
		case request := <-w.requests:
			w.executePreWarm(request)
		case <-w.stopCh:
			return
		}
	}
}

// stop stops the worker
func (w *PreWarmWorker) stop() {
	if !w.isRunning {
		return
	}
	close(w.stopCh)
	w.isRunning = false
}

// executePreWarm executes a prewarming request
func (w *PreWarmWorker) executePreWarm(request *PreWarmRequest) {
	startTime := time.Now()
	request.Status = PreWarmStatusExecuting
	
	w.logger.Debug("Executing prewarming request", 
		"request_id", request.ID, 
		"patient_id", request.PatientID)
	
	// Execute the actual prewarming
	err := w.performPreWarm(request)
	
	completedAt := time.Now()
	request.CompletedAt = &completedAt
	
	if err != nil {
		request.Status = PreWarmStatusFailed
		request.Error = err.Error()
		w.logger.Error("Prewarming request failed", 
			"request_id", request.ID, 
			"error", err.Error())
	} else {
		request.Status = PreWarmStatusCompleted
		w.logger.Debug("Prewarming request completed", 
			"request_id", request.ID, 
			"duration", time.Since(startTime))
	}
}

// performPreWarm performs the actual cache prewarming
func (w *PreWarmWorker) performPreWarm(request *PreWarmRequest) error {
	if w.cacheManager == nil {
		return fmt.Errorf("cache manager not available")
	}
	
	// This would trigger actual snapshot fetching and caching
	// For now, we'll simulate the operation
	ctx := context.Background()
	
	// Simulate cache prewarming by attempting to fetch the snapshot
	// In real implementation, this would call the Context Service
	snapshotKey := fmt.Sprintf("snapshot:%s", request.PatientID)
	
	// Check if already cached
	if w.cacheManager.Has(ctx, snapshotKey) {
		w.logger.Debug("Snapshot already cached", "patient_id", request.PatientID)
		return nil
	}
	
	// Simulate fetching and caching
	// In real implementation, this would call context service and cache the result
	w.logger.Debug("Prewarming cache for patient", "patient_id", request.PatientID)
	
	// Simulate some processing time
	time.Sleep(time.Duration(50+time.Now().UnixNano()%100) * time.Millisecond)
	
	return nil
}

// NewPreWarmingMetrics creates new prewarming metrics
func NewPreWarmingMetrics() *PreWarmingMetrics {
	return &PreWarmingMetrics{
		HourlyPatterns:  make(map[int]int64),
		DailyPatterns:   make(map[int]int64),
		PatternAccuracy: make(map[string]float64),
	}
}