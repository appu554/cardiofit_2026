package learning

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// OverrideAnalyzer analyzes override patterns and outcomes for learning
type OverrideAnalyzer struct {
	logger           *logger.Logger
	eventStore       OverrideEventStore // Interface for storing/retrieving override events
	config          *AnalyzerConfig
	analysisCache   map[string]*AnalysisResult
	cacheMutex      sync.RWMutex
}

// AnalyzerConfig contains configuration for the override analyzer
type AnalyzerConfig struct {
	AnalysisWindowDuration    time.Duration `yaml:"analysis_window_duration"`
	MinEventsForAnalysis     int           `yaml:"min_events_for_analysis"`
	OutcomeCorrelationWindow time.Duration `yaml:"outcome_correlation_window"`
	EnablePatternDetection   bool          `yaml:"enable_pattern_detection"`
	EnableRiskPrediction     bool          `yaml:"enable_risk_prediction"`
	CacheAnalysisResults     bool          `yaml:"cache_analysis_results"`
	CacheTTL                 time.Duration `yaml:"cache_ttl"`
}

// OverrideEventStore interface for storing and retrieving override events
type OverrideEventStore interface {
	StoreOverrideEvent(event *OverrideEvent) error
	StoreOutcomeEvent(event *ClinicalOutcomeEvent) error
	GetOverrideEvents(patientID string, timeWindow time.Duration) ([]OverrideEvent, error)
	GetOutcomeEvents(patientID string, timeWindow time.Duration) ([]ClinicalOutcomeEvent, error)
	GetSystemWideOverrideStats(timeWindow time.Duration) (*SystemOverrideStats, error)
}

// NewOverrideAnalyzer creates a new override analyzer
func NewOverrideAnalyzer(
	eventStore OverrideEventStore,
	config *AnalyzerConfig,
	logger *logger.Logger,
) *OverrideAnalyzer {
	if config == nil {
		config = &AnalyzerConfig{
			AnalysisWindowDuration:    7 * 24 * time.Hour, // 7 days
			MinEventsForAnalysis:     10,
			OutcomeCorrelationWindow: 72 * time.Hour, // 3 days
			EnablePatternDetection:   true,
			EnableRiskPrediction:     true,
			CacheAnalysisResults:     true,
			CacheTTL:                 time.Hour,
		}
	}

	return &OverrideAnalyzer{
		logger:        logger,
		eventStore:    eventStore,
		config:        config,
		analysisCache: make(map[string]*AnalysisResult),
	}
}

// AnalyzeOverridePatterns analyzes override patterns for a patient or system-wide
func (a *OverrideAnalyzer) AnalyzeOverridePatterns(
	ctx context.Context,
	patientID string, // Empty for system-wide analysis
) (*OverrideAnalysisResult, error) {
	startTime := time.Now()
	
	cacheKey := fmt.Sprintf("override_analysis_%s_%s", patientID, a.config.AnalysisWindowDuration.String())
	
	// Check cache first
	if a.config.CacheAnalysisResults {
		if cachedResult := a.getCachedAnalysis(cacheKey); cachedResult != nil {
			a.logger.Debug("Returning cached override analysis",
				zap.String("patient_id", patientID),
				zap.String("cache_key", cacheKey),
			)
			return cachedResult.OverrideAnalysis, nil
		}
	}

	a.logger.Info("Starting override pattern analysis",
		zap.String("patient_id", patientID),
		zap.Duration("time_window", a.config.AnalysisWindowDuration),
	)

	var overrideEvents []OverrideEvent
	var outcomeEvents []ClinicalOutcomeEvent
	var err error

	if patientID != "" {
		// Patient-specific analysis
		overrideEvents, err = a.eventStore.GetOverrideEvents(patientID, a.config.AnalysisWindowDuration)
		if err != nil {
			return nil, fmt.Errorf("failed to get override events: %w", err)
		}

		outcomeEvents, err = a.eventStore.GetOutcomeEvents(patientID, a.config.OutcomeCorrelationWindow)
		if err != nil {
			return nil, fmt.Errorf("failed to get outcome events: %w", err)
		}
	} else {
		// System-wide analysis would require different approach
		return a.analyzeSystemWidePatterns(ctx)
	}

	if len(overrideEvents) < a.config.MinEventsForAnalysis {
		return &OverrideAnalysisResult{
			PatientID:     patientID,
			AnalysisTime:  time.Now(),
			EventCount:    len(overrideEvents),
			SufficientData: false,
			Message:       fmt.Sprintf("Insufficient events for analysis (need %d, have %d)", 
				a.config.MinEventsForAnalysis, len(overrideEvents)),
		}, nil
	}

	// Perform comprehensive analysis
	result := &OverrideAnalysisResult{
		PatientID:      patientID,
		AnalysisTime:   time.Now(),
		AnalysisWindow: a.config.AnalysisWindowDuration,
		EventCount:     len(overrideEvents),
		SufficientData: true,
	}

	// Basic statistics
	result.BasicStats = a.calculateBasicStats(overrideEvents)

	// Pattern analysis
	if a.config.EnablePatternDetection {
		result.Patterns = a.detectOverridePatterns(overrideEvents)
	}

	// Outcome correlation
	result.OutcomeCorrelation = a.correlateWithOutcomes(overrideEvents, outcomeEvents)

	// Risk predictions
	if a.config.EnableRiskPrediction {
		result.RiskPredictions = a.generateRiskPredictions(overrideEvents, outcomeEvents)
	}

	// Performance impact analysis
	result.PerformanceImpact = a.analyzePerformanceImpact(overrideEvents)

	// Cache result
	if a.config.CacheAnalysisResults {
		a.cacheAnalysis(cacheKey, &AnalysisResult{
			CachedAt:         time.Now(),
			OverrideAnalysis: result,
		})
	}

	duration := time.Since(startTime)
	a.logger.Info("Override pattern analysis completed",
		zap.String("patient_id", patientID),
		zap.Int("events_analyzed", len(overrideEvents)),
		zap.Int64("analysis_time_ms", duration.Milliseconds()),
	)

	return result, nil
}

// calculateBasicStats calculates basic override statistics
func (a *OverrideAnalyzer) calculateBasicStats(events []OverrideEvent) *OverrideBasicStats {
	if len(events) == 0 {
		return &OverrideBasicStats{}
	}

	stats := &OverrideBasicStats{
		TotalOverrides: len(events),
		LevelCounts:    make(map[string]int),
		OutcomeCounts:  make(map[string]int),
	}

	var riskScores []float64
	successfulOverrides := 0
	
	for _, event := range events {
		// Count by required level
		stats.LevelCounts[event.TokenInfo.RequiredLevel]++
		
		// Count by validation outcome
		if event.ValidationInfo.Valid {
			successfulOverrides++
			stats.OutcomeCounts["successful"]++
		} else {
			stats.OutcomeCounts["failed"]++
		}
		
		// Collect risk scores
		riskScores = append(riskScores, event.OriginalDecision.RiskScore)
	}

	stats.SuccessRate = float64(successfulOverrides) / float64(len(events)) * 100
	
	// Calculate risk score statistics
	if len(riskScores) > 0 {
		sort.Float64s(riskScores)
		stats.AverageRiskScore = calculateMean(riskScores)
		stats.MedianRiskScore = calculateMedian(riskScores)
		stats.MaxRiskScore = riskScores[len(riskScores)-1]
		stats.MinRiskScore = riskScores[0]
	}

	return stats
}

// detectOverridePatterns detects patterns in override behavior
func (a *OverrideAnalyzer) detectOverridePatterns(events []OverrideEvent) []*OverridePattern {
	patterns := []*OverridePattern{}

	// Pattern 1: Temporal clustering
	if temporalPattern := a.detectTemporalClustering(events); temporalPattern != nil {
		patterns = append(patterns, temporalPattern)
	}

	// Pattern 2: Risk score escalation
	if escalationPattern := a.detectRiskEscalation(events); escalationPattern != nil {
		patterns = append(patterns, escalationPattern)
	}

	// Pattern 3: Repeated overrides for similar conditions
	if repetitionPattern := a.detectRepetitionPattern(events); repetitionPattern != nil {
		patterns = append(patterns, repetitionPattern)
	}

	// Pattern 4: Clinician-specific patterns
	if clinicianPattern := a.detectClinicianPatterns(events); clinicianPattern != nil {
		patterns = append(patterns, clinicianPattern)
	}

	return patterns
}

// correlateWithOutcomes correlates overrides with clinical outcomes
func (a *OverrideAnalyzer) correlateWithOutcomes(
	overrideEvents []OverrideEvent,
	outcomeEvents []ClinicalOutcomeEvent,
) *OutcomeCorrelation {
	correlation := &OutcomeCorrelation{
		TotalOverrides:      len(overrideEvents),
		TotalOutcomes:       len(outcomeEvents),
		CorrelatedEvents:    0,
		AdverseOutcomes:     0,
		PositiveOutcomes:    0,
		NeutralOutcomes:     0,
		CorrelationStrength: 0.0,
	}

	if len(overrideEvents) == 0 || len(outcomeEvents) == 0 {
		return correlation
	}

	correlatedPairs := []CorrelatedPair{}
	
	for _, overrideEvent := range overrideEvents {
		for _, outcomeEvent := range outcomeEvents {
			// Check if outcome is within correlation window
			timeDiff := outcomeEvent.Timestamp.Sub(overrideEvent.Timestamp)
			if timeDiff > 0 && timeDiff <= a.config.OutcomeCorrelationWindow {
				correlation.CorrelatedEvents++
				
				pair := CorrelatedPair{
					OverrideTokenID: overrideEvent.TokenInfo.TokenID,
					OutcomeType:     outcomeEvent.OutcomeType,
					OutcomeSeverity: outcomeEvent.OutcomeSeverity,
					TimeToOutcome:   timeDiff,
					RiskScore:       overrideEvent.OriginalDecision.RiskScore,
				}
				
				// Categorize outcomes
				switch outcomeEvent.OutcomeSeverity {
				case "severe", "critical":
					correlation.AdverseOutcomes++
					pair.IsAdverse = true
				case "mild", "moderate":
					correlation.NeutralOutcomes++
				default:
					correlation.PositiveOutcomes++
				}
				
				correlatedPairs = append(correlatedPairs, pair)
				break // Only correlate with first matching outcome
			}
		}
	}

	correlation.CorrelatedPairs = correlatedPairs
	
	// Calculate correlation strength (simplified)
	if len(overrideEvents) > 0 {
		correlation.CorrelationStrength = float64(correlation.CorrelatedEvents) / float64(len(overrideEvents))
	}

	return correlation
}

// generateRiskPredictions generates risk predictions based on patterns
func (a *OverrideAnalyzer) generateRiskPredictions(
	overrideEvents []OverrideEvent,
	outcomeEvents []ClinicalOutcomeEvent,
) []*RiskPrediction {
	predictions := []*RiskPrediction{}

	// Prediction 1: Future override likelihood
	if overrideLikelihood := a.predictFutureOverrideLikelihood(overrideEvents); overrideLikelihood != nil {
		predictions = append(predictions, overrideLikelihood)
	}

	// Prediction 2: Adverse outcome risk
	if adverseRisk := a.predictAdverseOutcomeRisk(overrideEvents, outcomeEvents); adverseRisk != nil {
		predictions = append(predictions, adverseRisk)
	}

	return predictions
}

// analyzePerformanceImpact analyzes the performance impact of overrides
func (a *OverrideAnalyzer) analyzePerformanceImpact(events []OverrideEvent) *PerformanceImpactAnalysis {
	if len(events) == 0 {
		return &PerformanceImpactAnalysis{}
	}

	impact := &PerformanceImpactAnalysis{
		TotalEvents: len(events),
	}

	var processingTimes []float64
	overrideCounts := make(map[string]int)
	
	for _, event := range events {
		// Collect processing times
		if event.OriginalDecision.ProcessingTime > 0 {
			processingTimes = append(processingTimes, float64(event.OriginalDecision.ProcessingTime.Milliseconds()))
		}
		
		// Count overrides by clinician
		overrideCounts[event.ValidationInfo.ClinicianID]++
	}

	if len(processingTimes) > 0 {
		impact.AverageProcessingTime = time.Duration(calculateMean(processingTimes)) * time.Millisecond
		impact.MedianProcessingTime = time.Duration(calculateMedian(processingTimes)) * time.Millisecond
	}

	// Find most active clinicians
	type ClinicianCount struct {
		ClinicianID string
		Count       int
	}
	
	var clinicianCounts []ClinicianCount
	for clinicianID, count := range overrideCounts {
		clinicianCounts = append(clinicianCounts, ClinicianCount{
			ClinicianID: clinicianID,
			Count:       count,
		})
	}
	
	sort.Slice(clinicianCounts, func(i, j int) bool {
		return clinicianCounts[i].Count > clinicianCounts[j].Count
	})
	
	for i, cc := range clinicianCounts {
		if i >= 5 { // Top 5 clinicians
			break
		}
		impact.TopClinicians = append(impact.TopClinicians, ClinicianOverrideInfo{
			ClinicianID:    cc.ClinicianID,
			OverrideCount:  cc.Count,
			OverrideRate:   float64(cc.Count) / float64(len(events)) * 100,
		})
	}

	return impact
}

// Helper methods for pattern detection

func (a *OverrideAnalyzer) detectTemporalClustering(events []OverrideEvent) *OverridePattern {
	if len(events) < 3 {
		return nil
	}

	// Sort events by timestamp
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	clusters := 0
	clusterThreshold := 4 * time.Hour // Events within 4 hours are considered clustered
	
	for i := 1; i < len(events); i++ {
		if events[i].Timestamp.Sub(events[i-1].Timestamp) <= clusterThreshold {
			clusters++
		}
	}

	if clusters >= 2 {
		return &OverridePattern{
			Type:        "temporal_clustering",
			Description: fmt.Sprintf("Detected %d temporal clusters of overrides within %v", clusters, clusterThreshold),
			Confidence:  float64(clusters) / float64(len(events)),
			Severity:    "medium",
			Recommendation: "Review clustering patterns for potential workflow issues",
		}
	}

	return nil
}

func (a *OverrideAnalyzer) detectRiskEscalation(events []OverrideEvent) *OverridePattern {
	if len(events) < 3 {
		return nil
	}

	// Sort by timestamp
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp.Before(events[j].Timestamp)
	})

	escalatingCount := 0
	for i := 1; i < len(events); i++ {
		if events[i].OriginalDecision.RiskScore > events[i-1].OriginalDecision.RiskScore {
			escalatingCount++
		}
	}

	escalationRate := float64(escalatingCount) / float64(len(events)-1)
	
	if escalationRate > 0.6 { // 60% escalation rate
		return &OverridePattern{
			Type:        "risk_escalation",
			Description: fmt.Sprintf("Risk scores escalating in %.1f%% of consecutive overrides", escalationRate*100),
			Confidence:  escalationRate,
			Severity:    "high",
			Recommendation: "Investigate underlying causes of increasing risk patterns",
		}
	}

	return nil
}

func (a *OverrideAnalyzer) detectRepetitionPattern(events []OverrideEvent) *OverridePattern {
	// Implementation for detecting repeated overrides would go here
	// This is a placeholder for the pattern detection logic
	return nil
}

func (a *OverrideAnalyzer) detectClinicianPatterns(events []OverrideEvent) *OverridePattern {
	// Implementation for detecting clinician-specific patterns would go here
	// This is a placeholder for the pattern detection logic
	return nil
}

func (a *OverrideAnalyzer) predictFutureOverrideLikelihood(events []OverrideEvent) *RiskPrediction {
	// Implementation for predicting future override likelihood would go here
	// This is a placeholder for the prediction logic
	return nil
}

func (a *OverrideAnalyzer) predictAdverseOutcomeRisk(overrideEvents []OverrideEvent, outcomeEvents []ClinicalOutcomeEvent) *RiskPrediction {
	// Implementation for predicting adverse outcome risk would go here
	// This is a placeholder for the prediction logic
	return nil
}

// Cache management methods
func (a *OverrideAnalyzer) getCachedAnalysis(cacheKey string) *AnalysisResult {
	a.cacheMutex.RLock()
	defer a.cacheMutex.RUnlock()

	if result, exists := a.analysisCache[cacheKey]; exists {
		if time.Since(result.CachedAt) <= a.config.CacheTTL {
			return result
		}
		// Remove expired cache entry
		delete(a.analysisCache, cacheKey)
	}
	return nil
}

func (a *OverrideAnalyzer) cacheAnalysis(cacheKey string, result *AnalysisResult) {
	a.cacheMutex.Lock()
	defer a.cacheMutex.Unlock()
	a.analysisCache[cacheKey] = result
}

// analyzeSystemWidePatterns analyzes system-wide override patterns
func (a *OverrideAnalyzer) analyzeSystemWidePatterns(ctx context.Context) (*OverrideAnalysisResult, error) {
	// Get system-wide stats
	systemStats, err := a.eventStore.GetSystemWideOverrideStats(a.config.AnalysisWindowDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to get system-wide stats: %w", err)
	}

	result := &OverrideAnalysisResult{
		PatientID:      "", // System-wide
		AnalysisTime:   time.Now(),
		AnalysisWindow: a.config.AnalysisWindowDuration,
		EventCount:     systemStats.TotalOverrides,
		SufficientData: systemStats.TotalOverrides >= a.config.MinEventsForAnalysis,
		SystemStats:    systemStats,
	}

	return result, nil
}

// Utility functions
func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateMedian(sortedValues []float64) float64 {
	n := len(sortedValues)
	if n == 0 {
		return 0
	}
	if n%2 == 0 {
		return (sortedValues[n/2-1] + sortedValues[n/2]) / 2
	}
	return sortedValues[n/2]
}

// Data structures for analysis results

type AnalysisResult struct {
	CachedAt         time.Time
	OverrideAnalysis *OverrideAnalysisResult
}

type OverrideAnalysisResult struct {
	PatientID           string                    `json:"patient_id"`
	AnalysisTime        time.Time                 `json:"analysis_time"`
	AnalysisWindow      time.Duration             `json:"analysis_window"`
	EventCount          int                       `json:"event_count"`
	SufficientData      bool                      `json:"sufficient_data"`
	Message             string                    `json:"message,omitempty"`
	BasicStats          *OverrideBasicStats       `json:"basic_stats,omitempty"`
	Patterns            []*OverridePattern        `json:"patterns,omitempty"`
	OutcomeCorrelation  *OutcomeCorrelation       `json:"outcome_correlation,omitempty"`
	RiskPredictions     []*RiskPrediction         `json:"risk_predictions,omitempty"`
	PerformanceImpact   *PerformanceImpactAnalysis `json:"performance_impact,omitempty"`
	SystemStats         *SystemOverrideStats      `json:"system_stats,omitempty"`
}

type OverrideBasicStats struct {
	TotalOverrides     int                `json:"total_overrides"`
	SuccessRate        float64            `json:"success_rate"`
	AverageRiskScore   float64            `json:"average_risk_score"`
	MedianRiskScore    float64            `json:"median_risk_score"`
	MaxRiskScore       float64            `json:"max_risk_score"`
	MinRiskScore       float64            `json:"min_risk_score"`
	LevelCounts        map[string]int     `json:"level_counts"`
	OutcomeCounts      map[string]int     `json:"outcome_counts"`
}

type OverridePattern struct {
	Type           string  `json:"type"`
	Description    string  `json:"description"`
	Confidence     float64 `json:"confidence"`
	Severity       string  `json:"severity"`
	Recommendation string  `json:"recommendation"`
}

type OutcomeCorrelation struct {
	TotalOverrides      int               `json:"total_overrides"`
	TotalOutcomes       int               `json:"total_outcomes"`
	CorrelatedEvents    int               `json:"correlated_events"`
	AdverseOutcomes     int               `json:"adverse_outcomes"`
	PositiveOutcomes    int               `json:"positive_outcomes"`
	NeutralOutcomes     int               `json:"neutral_outcomes"`
	CorrelationStrength float64           `json:"correlation_strength"`
	CorrelatedPairs     []CorrelatedPair  `json:"correlated_pairs"`
}

type CorrelatedPair struct {
	OverrideTokenID string        `json:"override_token_id"`
	OutcomeType     string        `json:"outcome_type"`
	OutcomeSeverity string        `json:"outcome_severity"`
	TimeToOutcome   time.Duration `json:"time_to_outcome"`
	RiskScore       float64       `json:"risk_score"`
	IsAdverse       bool          `json:"is_adverse"`
}

type RiskPrediction struct {
	PredictionType string                 `json:"prediction_type"`
	RiskScore      float64                `json:"risk_score"`
	Confidence     float64                `json:"confidence"`
	TimeHorizon    time.Duration          `json:"time_horizon"`
	Factors        []string               `json:"factors"`
	Recommendations []string              `json:"recommendations"`
	Metadata       map[string]interface{} `json:"metadata"`
}

type PerformanceImpactAnalysis struct {
	TotalEvents           int                      `json:"total_events"`
	AverageProcessingTime time.Duration            `json:"average_processing_time"`
	MedianProcessingTime  time.Duration            `json:"median_processing_time"`
	TopClinicians         []ClinicianOverrideInfo  `json:"top_clinicians"`
}

type ClinicianOverrideInfo struct {
	ClinicianID   string  `json:"clinician_id"`
	OverrideCount int     `json:"override_count"`
	OverrideRate  float64 `json:"override_rate"`
}

type SystemOverrideStats struct {
	TotalOverrides      int                `json:"total_overrides"`
	UniquePatients      int                `json:"unique_patients"`
	UniqueClinicians    int                `json:"unique_clinicians"`
	OverrideRate        float64            `json:"override_rate"`
	AverageRiskScore    float64            `json:"average_risk_score"`
	LevelDistribution   map[string]int     `json:"level_distribution"`
	OutcomeDistribution map[string]int     `json:"outcome_distribution"`
}