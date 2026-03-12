// Package analytics provides population-level analytics for KB-11.
//
// North Star: "KB-11 answers population-level questions, NOT patient-level decisions."
//
// This engine aggregates data to provide:
// - Population risk distribution
// - Care gap metrics (consumed from KB-13)
// - Attribution analytics (PCP/Practice panels)
// - Trend analysis over time
package analytics

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/cardiofit/kb-11-population-health/internal/clients"
	"github.com/cardiofit/kb-11-population-health/internal/database"
	"github.com/cardiofit/kb-11-population-health/internal/models"
)

// Engine provides population-level analytics capabilities.
type Engine struct {
	repo       *database.ProjectionRepository
	kb13Client *clients.KB13Client
	cache      *AnalyticsCache
	logger     *logrus.Entry
}

// NewEngine creates a new analytics engine.
func NewEngine(
	repo *database.ProjectionRepository,
	kb13Client *clients.KB13Client,
	logger *logrus.Entry,
) *Engine {
	return &Engine{
		repo:       repo,
		kb13Client: kb13Client,
		cache:      NewAnalyticsCache(15 * time.Minute),
		logger:     logger.WithField("component", "analytics-engine"),
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Population Analytics
// ──────────────────────────────────────────────────────────────────────────────

// PopulationSnapshot represents a point-in-time view of population health.
type PopulationSnapshot struct {
	TotalPatients     int                        `json:"total_patients"`
	ActivePatients    int                        `json:"active_patients"`
	RiskDistribution  map[models.RiskTier]int    `json:"risk_distribution"`
	RiskPercentages   map[models.RiskTier]float64 `json:"risk_percentages"`
	AverageRiskScore  float64                    `json:"average_risk_score"`
	HighRiskCount     int                        `json:"high_risk_count"`
	RisingRiskCount   int                        `json:"rising_risk_count"`
	CareGapMetrics    *CareGapSnapshot           `json:"care_gap_metrics,omitempty"`
	AttributionStats  *AttributionSnapshot       `json:"attribution_stats"`
	CalculatedAt      time.Time                  `json:"calculated_at"`
}

// CareGapSnapshot represents care gap metrics for the population.
type CareGapSnapshot struct {
	TotalOpenGaps         int                `json:"total_open_gaps"`
	PatientsWithGaps      int                `json:"patients_with_gaps"`
	AverageGapsPerPatient float64            `json:"average_gaps_per_patient"`
	GapsByCategory        map[string]int     `json:"gaps_by_category,omitempty"`
	GapsByPriority        map[string]int     `json:"gaps_by_priority,omitempty"`
	TopGapTypes           []clients.GapTypeSummary `json:"top_gap_types,omitempty"`
}

// CareGapMetrics is an alias for CareGapSnapshot for backward compatibility.
// Tests and external consumers may use either name interchangeably.
type CareGapMetrics = CareGapSnapshot

// AttributionSnapshot represents attribution statistics.
type AttributionSnapshot struct {
	TotalPCPs           int            `json:"total_pcps"`
	TotalPractices      int            `json:"total_practices"`
	PatientsByPCP       map[string]int `json:"patients_by_pcp,omitempty"`
	PatientsByPractice  map[string]int `json:"patients_by_practice,omitempty"`
	UnattributedCount   int            `json:"unattributed_count"`
}

// AttributionStats is an alias for AttributionSnapshot for backward compatibility.
type AttributionStats = AttributionSnapshot

// GetPopulationSnapshot generates a comprehensive population health snapshot.
func (e *Engine) GetPopulationSnapshot(ctx context.Context, filter *PopulationFilter) (*PopulationSnapshot, error) {
	// Check cache first
	cacheKey := "population_snapshot:" + filter.CacheKey()
	if cached := e.cache.Get(cacheKey); cached != nil {
		if snapshot, ok := cached.(*PopulationSnapshot); ok {
			return snapshot, nil
		}
	}

	snapshot := &PopulationSnapshot{
		RiskDistribution: make(map[models.RiskTier]int),
		RiskPercentages:  make(map[models.RiskTier]float64),
		CalculatedAt:     time.Now(),
	}

	// Get risk distribution from database
	riskDist, err := e.repo.GetRiskDistribution(ctx)
	if err != nil {
		e.logger.WithError(err).Error("Failed to get risk distribution")
		return nil, err
	}

	// Calculate totals and percentages
	var totalScore float64
	var scoredPatients int

	for tier, count := range riskDist {
		snapshot.RiskDistribution[tier] = count
		snapshot.TotalPatients += count

		if tier == models.RiskTierHigh || tier == models.RiskTierVeryHigh {
			snapshot.HighRiskCount += count
		}
		if tier == models.RiskTierRising {
			snapshot.RisingRiskCount += count
		}
	}

	snapshot.ActivePatients = snapshot.TotalPatients

	// Calculate percentages
	if snapshot.TotalPatients > 0 {
		for tier, count := range snapshot.RiskDistribution {
			snapshot.RiskPercentages[tier] = float64(count) / float64(snapshot.TotalPatients) * 100
		}
	}

	// Get average risk score
	avgScore, err := e.repo.GetAverageRiskScore(ctx)
	if err == nil && scoredPatients > 0 {
		snapshot.AverageRiskScore = totalScore / float64(scoredPatients)
	} else {
		snapshot.AverageRiskScore = avgScore
	}

	// Get attribution stats
	attrStats, err := e.getAttributionStats(ctx)
	if err == nil {
		snapshot.AttributionStats = attrStats
	}

	// Get care gap metrics from KB-13 (if available)
	if e.kb13Client != nil {
		careGaps, err := e.getCareGapSnapshot(ctx)
		if err != nil {
			e.logger.WithError(err).Warn("Failed to get care gap metrics from KB-13")
		} else {
			snapshot.CareGapMetrics = careGaps
		}
	}

	// Cache the result
	e.cache.Set(cacheKey, snapshot)

	return snapshot, nil
}

// getAttributionStats retrieves attribution statistics.
func (e *Engine) getAttributionStats(ctx context.Context) (*AttributionSnapshot, error) {
	stats := &AttributionSnapshot{
		PatientsByPCP:      make(map[string]int),
		PatientsByPractice: make(map[string]int),
	}

	// Get PCP distribution
	pcpDist, err := e.repo.GetPatientsByPCP(ctx)
	if err != nil {
		return nil, err
	}
	stats.PatientsByPCP = pcpDist
	stats.TotalPCPs = len(pcpDist)

	// Get practice distribution
	practiceDist, err := e.repo.GetPatientsByPractice(ctx)
	if err != nil {
		return nil, err
	}
	stats.PatientsByPractice = practiceDist
	stats.TotalPractices = len(practiceDist)

	// Get unattributed count
	unattributed, err := e.repo.GetUnattributedCount(ctx)
	if err == nil {
		stats.UnattributedCount = unattributed
	}

	return stats, nil
}

// getCareGapSnapshot retrieves care gap metrics from KB-13.
func (e *Engine) getCareGapSnapshot(ctx context.Context) (*CareGapSnapshot, error) {
	if e.kb13Client == nil {
		return nil, nil
	}

	metrics, err := e.kb13Client.GetPopulationCareGapMetrics(ctx, nil)
	if err != nil {
		return nil, err
	}

	snapshot := &CareGapSnapshot{
		TotalOpenGaps:         metrics.TotalOpenGaps,
		PatientsWithGaps:      metrics.PatientsWithGaps,
		AverageGapsPerPatient: metrics.AverageGapsPerPatient,
		GapsByCategory:        metrics.GapsByCategory,
		GapsByPriority:        metrics.GapsByPriority,
		TopGapTypes:           metrics.TopGapTypes,
	}

	return snapshot, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Risk Stratification Analytics
// ──────────────────────────────────────────────────────────────────────────────

// RiskStratificationReport provides detailed risk stratification analysis.
type RiskStratificationReport struct {
	Distribution       map[models.RiskTier]*TierDetails `json:"distribution"`
	Trends             []RiskTrendPoint                 `json:"trends"`
	RisingRiskPatients []RisingRiskSummary              `json:"rising_risk_patients"`
	HighRiskBreakdown  *HighRiskBreakdown               `json:"high_risk_breakdown"`
	ReportDate         time.Time                        `json:"report_date"`
}

// TierDetails provides details about a risk tier.
type TierDetails struct {
	Tier            models.RiskTier `json:"tier"`
	Count           int             `json:"count"`
	Percentage      float64         `json:"percentage"`
	AverageScore    float64         `json:"average_score"`
	AverageAge      float64         `json:"average_age"`
	TopConditions   []string        `json:"top_conditions,omitempty"`
}

// RiskTrendPoint represents a point in risk trend analysis.
type RiskTrendPoint struct {
	Date          time.Time              `json:"date"`
	Distribution  map[models.RiskTier]int `json:"distribution"`
	HighRiskCount int                    `json:"high_risk_count"`
	ChangeFromPrev float64               `json:"change_from_prev"`
}

// RisingRiskSummary summarizes a rising-risk patient.
type RisingRiskSummary struct {
	PatientFHIRID   string    `json:"patient_fhir_id"`
	CurrentScore    float64   `json:"current_score"`
	PreviousScore   float64   `json:"previous_score"`
	RisingRate      float64   `json:"rising_rate"`
	DaysRising      int       `json:"days_rising"`
	AttributedPCP   string    `json:"attributed_pcp,omitempty"`
}

// HighRiskBreakdown provides breakdown of high-risk population.
type HighRiskBreakdown struct {
	TotalHighRisk      int               `json:"total_high_risk"`
	ByAge              map[string]int    `json:"by_age"`
	ByConditionCount   map[string]int    `json:"by_condition_count"`
	WithRecentAdmit    int               `json:"with_recent_admit"`
	WithCareGaps       int               `json:"with_care_gaps"`
	AverageConditions  float64           `json:"average_conditions"`
}

// GetRiskStratificationReport generates a detailed risk stratification report.
func (e *Engine) GetRiskStratificationReport(ctx context.Context) (*RiskStratificationReport, error) {
	report := &RiskStratificationReport{
		Distribution: make(map[models.RiskTier]*TierDetails),
		ReportDate:   time.Now(),
	}

	// Get distribution with details
	riskDist, err := e.repo.GetRiskDistribution(ctx)
	if err != nil {
		return nil, err
	}

	totalPatients := 0
	for _, count := range riskDist {
		totalPatients += count
	}

	for tier, count := range riskDist {
		percentage := 0.0
		if totalPatients > 0 {
			percentage = float64(count) / float64(totalPatients) * 100
		}
		report.Distribution[tier] = &TierDetails{
			Tier:       tier,
			Count:      count,
			Percentage: percentage,
		}
	}

	// Get rising risk patients
	risingPatients, err := e.repo.GetRisingRiskPatients(ctx, 50) // Top 50
	if err == nil {
		for _, p := range risingPatients {
			var score float64
			if p.LatestRiskScore != nil {
				score = *p.LatestRiskScore
			}
			report.RisingRiskPatients = append(report.RisingRiskPatients, RisingRiskSummary{
				PatientFHIRID: p.FHIRID,
				CurrentScore:  score,
				AttributedPCP: ptrToString(p.AttributedPCP),
			})
		}
	}

	return report, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Provider Analytics (PCP/Practice)
// ──────────────────────────────────────────────────────────────────────────────

// ProviderPanelAnalytics provides analytics for a provider's panel.
type ProviderPanelAnalytics struct {
	ProviderID        string                     `json:"provider_id"`
	ProviderName      string                     `json:"provider_name"`
	PanelSize         int                        `json:"panel_size"`
	RiskDistribution  map[models.RiskTier]int    `json:"risk_distribution"`
	HighRiskCount     int                        `json:"high_risk_count"`
	RisingRiskCount   int                        `json:"rising_risk_count"`
	AverageRiskScore  float64                    `json:"average_risk_score"`
	CareGapCount      int                        `json:"care_gap_count,omitempty"`
	ComparedToAverage *ComparisonMetrics         `json:"compared_to_average,omitempty"`
	CalculatedAt      time.Time                  `json:"calculated_at"`
}

// ComparisonMetrics compares a provider to population averages.
type ComparisonMetrics struct {
	HighRiskPercentDiff  float64 `json:"high_risk_percent_diff"`
	RiskScoreDiff        float64 `json:"risk_score_diff"`
	CareGapDiff          float64 `json:"care_gap_diff,omitempty"`
	Percentile           int     `json:"percentile"`
}

// PracticeAnalytics provides analytics for a practice.
type PracticeAnalytics struct {
	PracticeID        string                     `json:"practice_id"`
	PracticeName      string                     `json:"practice_name"`
	TotalPatients     int                        `json:"total_patients"`
	ProviderCount     int                        `json:"provider_count"`
	RiskDistribution  map[models.RiskTier]int    `json:"risk_distribution"`
	HighRiskCount     int                        `json:"high_risk_count"`
	AverageRiskScore  float64                    `json:"average_risk_score"`
	TopProviders      []ProviderSummary          `json:"top_providers,omitempty"`
	CalculatedAt      time.Time                  `json:"calculated_at"`
}

// ProviderSummary provides a summary for a provider.
type ProviderSummary struct {
	ProviderID       string  `json:"provider_id"`
	ProviderName     string  `json:"provider_name"`
	PanelSize        int     `json:"panel_size"`
	HighRiskPercent  float64 `json:"high_risk_percent"`
}

// GetProviderPanelAnalytics generates analytics for a specific provider.
func (e *Engine) GetProviderPanelAnalytics(ctx context.Context, providerID string) (*ProviderPanelAnalytics, error) {
	analytics := &ProviderPanelAnalytics{
		ProviderID:       providerID,
		RiskDistribution: make(map[models.RiskTier]int),
		CalculatedAt:     time.Now(),
	}

	// Get patients for this PCP
	patients, err := e.repo.GetPatientsByAttributedPCP(ctx, providerID)
	if err != nil {
		return nil, err
	}

	analytics.PanelSize = len(patients)

	// Calculate risk distribution
	var totalScore float64
	for _, p := range patients {
		analytics.RiskDistribution[p.CurrentRiskTier]++
		if p.LatestRiskScore != nil {
			totalScore += *p.LatestRiskScore
		}

		if p.CurrentRiskTier == models.RiskTierHigh || p.CurrentRiskTier == models.RiskTierVeryHigh {
			analytics.HighRiskCount++
		}
		if p.CurrentRiskTier == models.RiskTierRising {
			analytics.RisingRiskCount++
		}
	}

	if analytics.PanelSize > 0 {
		analytics.AverageRiskScore = totalScore / float64(analytics.PanelSize)
	}

	// Get population averages for comparison
	popAvg, err := e.repo.GetAverageRiskScore(ctx)
	if err == nil && analytics.PanelSize > 0 {
		popHighRiskPct, _ := e.repo.GetHighRiskPercentage(ctx)
		providerHighRiskPct := float64(analytics.HighRiskCount) / float64(analytics.PanelSize) * 100

		analytics.ComparedToAverage = &ComparisonMetrics{
			HighRiskPercentDiff: providerHighRiskPct - popHighRiskPct,
			RiskScoreDiff:       analytics.AverageRiskScore - popAvg,
		}
	}

	return analytics, nil
}

// GetPracticeAnalytics generates analytics for a specific practice.
func (e *Engine) GetPracticeAnalytics(ctx context.Context, practiceID string) (*PracticeAnalytics, error) {
	analytics := &PracticeAnalytics{
		PracticeID:       practiceID,
		PracticeName:     practiceID, // Would be fetched from a provider registry
		RiskDistribution: make(map[models.RiskTier]int),
		CalculatedAt:     time.Now(),
	}

	// Get patients for this practice
	patients, err := e.repo.GetPatientsByAttributedPractice(ctx, practiceID)
	if err != nil {
		return nil, err
	}

	analytics.TotalPatients = len(patients)

	// Calculate risk distribution and track providers
	providerCounts := make(map[string]int)
	var totalScore float64

	for _, p := range patients {
		analytics.RiskDistribution[p.CurrentRiskTier]++
		if p.LatestRiskScore != nil {
			totalScore += *p.LatestRiskScore
		}

		if p.CurrentRiskTier == models.RiskTierHigh || p.CurrentRiskTier == models.RiskTierVeryHigh {
			analytics.HighRiskCount++
		}

		if p.AttributedPCP != nil && *p.AttributedPCP != "" {
			providerCounts[*p.AttributedPCP]++
		}
	}

	analytics.ProviderCount = len(providerCounts)

	if analytics.TotalPatients > 0 {
		analytics.AverageRiskScore = totalScore / float64(analytics.TotalPatients)
	}

	return analytics, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Population Filter
// ──────────────────────────────────────────────────────────────────────────────

// PopulationFilter provides filtering options for population analytics.
type PopulationFilter struct {
	Practice   string
	PCP        string
	RiskTiers  []models.RiskTier
	MinAge     int
	MaxAge     int
	WithCareGaps bool
}

// CacheKey generates a cache key for this filter.
func (f *PopulationFilter) CacheKey() string {
	if f == nil {
		return "all"
	}
	return f.Practice + ":" + f.PCP
}

// ──────────────────────────────────────────────────────────────────────────────
// Analytics Cache
// ──────────────────────────────────────────────────────────────────────────────

// AnalyticsCache provides in-memory caching for analytics results.
type AnalyticsCache struct {
	data map[string]*cacheEntry
	ttl  time.Duration
	mu   sync.RWMutex
}

type cacheEntry struct {
	value      interface{}
	expiration time.Time
}

// NewAnalyticsCache creates a new analytics cache.
func NewAnalyticsCache(ttl time.Duration) *AnalyticsCache {
	cache := &AnalyticsCache{
		data: make(map[string]*cacheEntry),
		ttl:  ttl,
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves a value from the cache.
func (c *AnalyticsCache) Get(key string) interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.data[key]
	if !ok {
		return nil
	}

	if time.Now().After(entry.expiration) {
		return nil
	}

	return entry.value
}

// Set stores a value in the cache.
func (c *AnalyticsCache) Set(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[key] = &cacheEntry{
		value:      value,
		expiration: time.Now().Add(c.ttl),
	}
}

// cleanup periodically removes expired entries.
func (c *AnalyticsCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.data {
			if now.After(entry.expiration) {
				delete(c.data, key)
			}
		}
		c.mu.Unlock()
	}
}

// Invalidate removes all cached entries.
func (c *AnalyticsCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data = make(map[string]*cacheEntry)
}

// ──────────────────────────────────────────────────────────────────────────────
// Custom Query Engine (Phase D - Custom Analytics Query)
// ──────────────────────────────────────────────────────────────────────────────

// CustomQuery represents a flexible analytics query.
type CustomQuery struct {
	// Filtering
	Filters     QueryFilters     `json:"filters"`
	// Aggregation
	Aggregations []Aggregation   `json:"aggregations"`
	// Grouping
	GroupBy      []string        `json:"group_by,omitempty"`
	// Sorting
	SortBy       string          `json:"sort_by,omitempty"`
	SortOrder    string          `json:"sort_order,omitempty"` // "asc" or "desc"
	// Pagination
	Limit        int             `json:"limit,omitempty"`
	Offset       int             `json:"offset,omitempty"`
}

// QueryFilters contains filter criteria for custom queries.
type QueryFilters struct {
	RiskTiers       []models.RiskTier `json:"risk_tiers,omitempty"`
	MinRiskScore    *float64          `json:"min_risk_score,omitempty"`
	MaxRiskScore    *float64          `json:"max_risk_score,omitempty"`
	Practices       []string          `json:"practices,omitempty"`
	PCPs            []string          `json:"pcps,omitempty"`
	AgeRange        *AgeRange         `json:"age_range,omitempty"`
	Gender          *string           `json:"gender,omitempty"`
	HasCareGaps     *bool             `json:"has_care_gaps,omitempty"`
	HasAttribution  *bool             `json:"has_attribution,omitempty"`
	DateRange       *DateRange        `json:"date_range,omitempty"`
}

// AgeRange specifies an age range filter.
type AgeRange struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

// DateRange specifies a date range filter.
type DateRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// Aggregation specifies an aggregation operation.
type Aggregation struct {
	Field     string `json:"field"`     // "risk_score", "age", "care_gap_count"
	Operation string `json:"operation"` // "count", "sum", "avg", "min", "max", "percentile"
	Percentile float64 `json:"percentile,omitempty"` // For percentile operation
	Alias     string `json:"alias,omitempty"`
}

// CustomQueryResult contains the result of a custom query.
type CustomQueryResult struct {
	Data         []map[string]interface{} `json:"data"`
	Aggregations map[string]interface{}   `json:"aggregations,omitempty"`
	TotalCount   int                      `json:"total_count"`
	Limit        int                      `json:"limit"`
	Offset       int                      `json:"offset"`
	ExecutedAt   time.Time                `json:"executed_at"`
}

// ExecuteCustomQuery executes a flexible analytics query.
func (e *Engine) ExecuteCustomQuery(ctx context.Context, query *CustomQuery) (*CustomQueryResult, error) {
	result := &CustomQueryResult{
		Data:         make([]map[string]interface{}, 0),
		Aggregations: make(map[string]interface{}),
		ExecutedAt:   time.Now(),
		Limit:        query.Limit,
		Offset:       query.Offset,
	}

	// Set defaults
	if query.Limit == 0 {
		query.Limit = 100
	}
	if query.Limit > 1000 {
		query.Limit = 1000 // Cap at 1000 for performance
	}

	// Build and execute the query using existing Query method
	queryReq := e.buildQueryRequest(query)
	patients, total, err := e.repo.Query(ctx, queryReq)
	if err != nil {
		return nil, err
	}

	result.TotalCount = total

	// Process results based on grouping
	if len(query.GroupBy) > 0 {
		result.Data = e.groupResults(patients, query.GroupBy)
	} else {
		for _, p := range patients {
			result.Data = append(result.Data, map[string]interface{}{
				"fhir_id":             p.FHIRID,
				"risk_score":          p.LatestRiskScore,
				"risk_tier":           p.CurrentRiskTier,
				"attributed_pcp":      ptrToString(p.AttributedPCP),
				"attributed_practice": ptrToString(p.AttributedPractice),
			})
		}
	}

	// Execute aggregations
	for _, agg := range query.Aggregations {
		aggResult, err := e.executeAggregation(ctx, agg, patients)
		if err != nil {
			e.logger.WithError(err).WithField("aggregation", agg.Field).Warn("Failed to execute aggregation")
			continue
		}
		alias := agg.Alias
		if alias == "" {
			alias = agg.Operation + "_" + agg.Field
		}
		result.Aggregations[alias] = aggResult
	}

	return result, nil
}

// ptrToString safely converts a *string to string.
func ptrToString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// buildQueryRequest converts CustomQuery filters to a PatientQueryRequest.
func (e *Engine) buildQueryRequest(query *CustomQuery) *models.PatientQueryRequest {
	req := &models.PatientQueryRequest{
		Limit:     query.Limit,
		Offset:    query.Offset,
		SortBy:    query.SortBy,
		SortOrder: query.SortOrder,
	}

	// Map risk tier filter (first one if multiple)
	if len(query.Filters.RiskTiers) > 0 {
		tier := query.Filters.RiskTiers[0]
		req.RiskTier = &tier
	}

	// Map practice filter (first one if multiple)
	if len(query.Filters.Practices) > 0 {
		req.AttributedPractice = &query.Filters.Practices[0]
	}

	// Map PCP filter (first one if multiple)
	if len(query.Filters.PCPs) > 0 {
		req.AttributedPCP = &query.Filters.PCPs[0]
	}

	return req
}

// groupResults groups patient data by specified fields.
func (e *Engine) groupResults(patients []*models.PatientProjection, groupBy []string) []map[string]interface{} {
	groups := make(map[string][]map[string]interface{})

	for _, p := range patients {
		key := ""
		for _, field := range groupBy {
			switch field {
			case "risk_tier":
				key += string(p.CurrentRiskTier) + "|"
			case "attributed_pcp":
				key += ptrToString(p.AttributedPCP) + "|"
			case "attributed_practice":
				key += ptrToString(p.AttributedPractice) + "|"
			}
		}

		entry := map[string]interface{}{
			"fhir_id":     p.FHIRID,
			"risk_score":  p.LatestRiskScore,
			"risk_tier":   p.CurrentRiskTier,
		}
		groups[key] = append(groups[key], entry)
	}

	// Convert to result format with counts
	result := make([]map[string]interface{}, 0, len(groups))
	for key, members := range groups {
		entry := map[string]interface{}{
			"group_key":    key,
			"count":        len(members),
			"members":      members,
		}
		result = append(result, entry)
	}

	return result
}

// executeAggregation executes a single aggregation operation.
func (e *Engine) executeAggregation(ctx context.Context, agg Aggregation, patients []*models.PatientProjection) (interface{}, error) {
	var values []float64

	for _, p := range patients {
		switch agg.Field {
		case "risk_score":
			if p.LatestRiskScore != nil {
				values = append(values, *p.LatestRiskScore)
			}
		}
	}

	if len(values) == 0 {
		return nil, nil
	}

	switch agg.Operation {
	case "count":
		return len(values), nil
	case "sum":
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		return sum, nil
	case "avg":
		sum := 0.0
		for _, v := range values {
			sum += v
		}
		return sum / float64(len(values)), nil
	case "min":
		min := values[0]
		for _, v := range values {
			if v < min {
				min = v
			}
		}
		return min, nil
	case "max":
		max := values[0]
		for _, v := range values {
			if v > max {
				max = v
			}
		}
		return max, nil
	default:
		return nil, nil
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Trend Analysis (Phase D - Time-Series Analytics)
// ──────────────────────────────────────────────────────────────────────────────

// TrendAnalysisRequest specifies parameters for trend analysis.
type TrendAnalysisRequest struct {
	StartDate    time.Time         `json:"start_date"`
	EndDate      time.Time         `json:"end_date"`
	Interval     string            `json:"interval"` // "daily", "weekly", "monthly"
	Metrics      []string          `json:"metrics"`  // "risk_score", "high_risk_count", "care_gaps"
	GroupBy      string            `json:"group_by,omitempty"` // "practice", "pcp", "risk_tier"
	Filters      *QueryFilters     `json:"filters,omitempty"`
}

// TrendAnalysisResult contains time-series trend data.
type TrendAnalysisResult struct {
	StartDate    time.Time           `json:"start_date"`
	EndDate      time.Time           `json:"end_date"`
	Interval     string              `json:"interval"`
	DataPoints   []TrendDataPoint    `json:"data_points"`
	Summary      *TrendSummary       `json:"summary"`
	GeneratedAt  time.Time           `json:"generated_at"`
}

// TrendDataPoint represents a single point in the trend.
type TrendDataPoint struct {
	Date       time.Time              `json:"date"`
	Metrics    map[string]float64     `json:"metrics"`
	GroupData  map[string]interface{} `json:"group_data,omitempty"`
}

// TrendSummary provides summary statistics for the trend.
type TrendSummary struct {
	OverallChange      map[string]float64 `json:"overall_change"`      // Percentage change
	AverageValues      map[string]float64 `json:"average_values"`
	MinValues          map[string]float64 `json:"min_values"`
	MaxValues          map[string]float64 `json:"max_values"`
	TrendDirection     map[string]string  `json:"trend_direction"`     // "increasing", "decreasing", "stable"
}

// GetTrendAnalysis generates time-series trend analysis.
func (e *Engine) GetTrendAnalysis(ctx context.Context, req *TrendAnalysisRequest) (*TrendAnalysisResult, error) {
	result := &TrendAnalysisResult{
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Interval:    req.Interval,
		DataPoints:  make([]TrendDataPoint, 0),
		GeneratedAt: time.Now(),
	}

	// Default interval to weekly
	if req.Interval == "" {
		req.Interval = "weekly"
	}

	// Generate date intervals
	dates := e.generateDateIntervals(req.StartDate, req.EndDate, req.Interval)

	// For each date interval, get metrics
	metricSums := make(map[string][]float64)

	for _, date := range dates {
		dataPoint := TrendDataPoint{
			Date:    date,
			Metrics: make(map[string]float64),
		}

		// Get historical snapshot for this date
		dbSnapshot, err := e.repo.GetHistoricalSnapshot(ctx, date)
		if err != nil {
			e.logger.WithError(err).WithField("date", date).Warn("Failed to get historical snapshot")
			continue
		}

		// Convert to local type
		snapshot := &HistoricalSnapshotData{
			SnapshotDate:     dbSnapshot.SnapshotDate,
			TotalPatients:    dbSnapshot.TotalPatients,
			HighRiskCount:    dbSnapshot.HighRiskCount,
			RisingRiskCount:  dbSnapshot.RisingRiskCount,
			AverageRiskScore: dbSnapshot.AverageRiskScore,
			CareGapCount:     dbSnapshot.CareGapCount,
		}

		// Extract requested metrics
		for _, metric := range req.Metrics {
			value := e.extractMetricFromSnapshot(snapshot, metric)
			dataPoint.Metrics[metric] = value
			metricSums[metric] = append(metricSums[metric], value)
		}

		result.DataPoints = append(result.DataPoints, dataPoint)
	}

	// Calculate summary statistics
	result.Summary = e.calculateTrendSummary(metricSums, req.Metrics)

	return result, nil
}

// generateDateIntervals creates date intervals for trend analysis.
func (e *Engine) generateDateIntervals(start, end time.Time, interval string) []time.Time {
	var dates []time.Time
	current := start

	var step time.Duration
	switch interval {
	case "daily":
		step = 24 * time.Hour
	case "weekly":
		step = 7 * 24 * time.Hour
	case "monthly":
		step = 30 * 24 * time.Hour // Approximate
	default:
		step = 7 * 24 * time.Hour
	}

	for current.Before(end) || current.Equal(end) {
		dates = append(dates, current)
		current = current.Add(step)
	}

	return dates
}

// extractMetricFromSnapshot extracts a specific metric from a snapshot.
func (e *Engine) extractMetricFromSnapshot(snapshot *HistoricalSnapshotData, metric string) float64 {
	if snapshot == nil {
		return 0
	}

	switch metric {
	case "risk_score", "average_risk_score":
		return snapshot.AverageRiskScore
	case "high_risk_count":
		return float64(snapshot.HighRiskCount)
	case "total_patients":
		return float64(snapshot.TotalPatients)
	case "rising_risk_count":
		return float64(snapshot.RisingRiskCount)
	case "care_gap_count":
		return float64(snapshot.CareGapCount)
	default:
		return 0
	}
}

// HistoricalSnapshotData is a local type for historical snapshot data.
type HistoricalSnapshotData struct {
	SnapshotDate     time.Time
	TotalPatients    int
	HighRiskCount    int
	RisingRiskCount  int
	AverageRiskScore float64
	CareGapCount     int
}

// calculateTrendSummary calculates summary statistics for trends.
func (e *Engine) calculateTrendSummary(metricSums map[string][]float64, metrics []string) *TrendSummary {
	summary := &TrendSummary{
		OverallChange:  make(map[string]float64),
		AverageValues:  make(map[string]float64),
		MinValues:      make(map[string]float64),
		MaxValues:      make(map[string]float64),
		TrendDirection: make(map[string]string),
	}

	for _, metric := range metrics {
		values := metricSums[metric]
		if len(values) == 0 {
			continue
		}

		// Calculate statistics
		sum := 0.0
		min := values[0]
		max := values[0]
		for _, v := range values {
			sum += v
			if v < min {
				min = v
			}
			if v > max {
				max = v
			}
		}

		summary.AverageValues[metric] = sum / float64(len(values))
		summary.MinValues[metric] = min
		summary.MaxValues[metric] = max

		// Calculate overall change
		if len(values) >= 2 && values[0] != 0 {
			first := values[0]
			last := values[len(values)-1]
			summary.OverallChange[metric] = ((last - first) / first) * 100
		}

		// Determine trend direction
		if len(values) >= 2 {
			first := values[0]
			last := values[len(values)-1]
			if last > first*1.05 {
				summary.TrendDirection[metric] = "increasing"
			} else if last < first*0.95 {
				summary.TrendDirection[metric] = "decreasing"
			} else {
				summary.TrendDirection[metric] = "stable"
			}
		}
	}

	return summary
}

// ──────────────────────────────────────────────────────────────────────────────
// Utilization Reporting (Phase D - Resource Utilization)
// ──────────────────────────────────────────────────────────────────────────────

// UtilizationReportRequest specifies parameters for utilization reporting.
type UtilizationReportRequest struct {
	ReportType   string        `json:"report_type"` // "inpatient", "ed", "outpatient", "all"
	StartDate    time.Time     `json:"start_date"`
	EndDate      time.Time     `json:"end_date"`
	GroupBy      string        `json:"group_by,omitempty"` // "practice", "pcp", "risk_tier", "month"
	Filters      *QueryFilters `json:"filters,omitempty"`
}

// UtilizationReport contains utilization metrics.
type UtilizationReport struct {
	ReportType      string                 `json:"report_type"`
	DateRange       DateRange              `json:"date_range"`
	Summary         *UtilizationSummary    `json:"summary"`
	ByRiskTier      map[string]*UtilizationMetrics `json:"by_risk_tier,omitempty"`
	ByProvider      map[string]*UtilizationMetrics `json:"by_provider,omitempty"`
	ByPractice      map[string]*UtilizationMetrics `json:"by_practice,omitempty"`
	MonthlyTrend    []MonthlyUtilization   `json:"monthly_trend,omitempty"`
	GeneratedAt     time.Time              `json:"generated_at"`
}

// UtilizationSummary provides summary utilization metrics.
type UtilizationSummary struct {
	TotalPatients       int     `json:"total_patients"`
	TotalEncounters     int     `json:"total_encounters"`
	InpatientAdmissions int     `json:"inpatient_admissions"`
	EDVisits            int     `json:"ed_visits"`
	OutpatientVisits    int     `json:"outpatient_visits"`
	Readmissions30Day   int     `json:"readmissions_30day"`
	AvgLengthOfStay     float64 `json:"avg_length_of_stay"`
	CostPerPatient      float64 `json:"cost_per_patient,omitempty"`
}

// UtilizationMetrics contains detailed utilization metrics.
type UtilizationMetrics struct {
	PatientCount        int     `json:"patient_count"`
	EncounterCount      int     `json:"encounter_count"`
	InpatientRate       float64 `json:"inpatient_rate"`       // Per 1000 patients
	EDRate              float64 `json:"ed_rate"`              // Per 1000 patients
	ReadmissionRate     float64 `json:"readmission_rate"`     // Percentage
	AvgEncountersPerPt  float64 `json:"avg_encounters_per_pt"`
}

// MonthlyUtilization represents utilization for a specific month.
type MonthlyUtilization struct {
	Month             string  `json:"month"` // "2025-01"
	InpatientAdmissions int   `json:"inpatient_admissions"`
	EDVisits          int     `json:"ed_visits"`
	OutpatientVisits  int     `json:"outpatient_visits"`
	Readmissions      int     `json:"readmissions"`
}

// GetUtilizationReport generates a utilization report.
func (e *Engine) GetUtilizationReport(ctx context.Context, req *UtilizationReportRequest) (*UtilizationReport, error) {
	report := &UtilizationReport{
		ReportType:  req.ReportType,
		DateRange:   DateRange{Start: req.StartDate, End: req.EndDate},
		GeneratedAt: time.Now(),
	}

	// Get utilization data from repository
	utilData, err := e.repo.GetUtilizationData(ctx, req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}

	// Build summary
	report.Summary = &UtilizationSummary{
		TotalPatients:       utilData.TotalPatients,
		TotalEncounters:     utilData.TotalEncounters,
		InpatientAdmissions: utilData.InpatientAdmissions,
		EDVisits:            utilData.EDVisits,
		OutpatientVisits:    utilData.OutpatientVisits,
		Readmissions30Day:   utilData.Readmissions30Day,
		AvgLengthOfStay:     utilData.AvgLengthOfStay,
	}

	// Group by risk tier
	report.ByRiskTier = make(map[string]*UtilizationMetrics)
	for tier, data := range utilData.ByRiskTier {
		report.ByRiskTier[tier] = &UtilizationMetrics{
			PatientCount:       data.PatientCount,
			EncounterCount:     data.EncounterCount,
			InpatientRate:      data.InpatientRate,
			EDRate:             data.EDRate,
			ReadmissionRate:    data.ReadmissionRate,
			AvgEncountersPerPt: data.AvgEncountersPerPt,
		}
	}

	// Group by provider if requested
	if req.GroupBy == "pcp" || req.GroupBy == "provider" {
		report.ByProvider = make(map[string]*UtilizationMetrics)
		for provider, data := range utilData.ByProvider {
			report.ByProvider[provider] = &UtilizationMetrics{
				PatientCount:       data.PatientCount,
				EncounterCount:     data.EncounterCount,
				InpatientRate:      data.InpatientRate,
				EDRate:             data.EDRate,
				ReadmissionRate:    data.ReadmissionRate,
				AvgEncountersPerPt: data.AvgEncountersPerPt,
			}
		}
	}

	// Group by practice if requested
	if req.GroupBy == "practice" {
		report.ByPractice = make(map[string]*UtilizationMetrics)
		for practice, data := range utilData.ByPractice {
			report.ByPractice[practice] = &UtilizationMetrics{
				PatientCount:       data.PatientCount,
				EncounterCount:     data.EncounterCount,
				InpatientRate:      data.InpatientRate,
				EDRate:             data.EDRate,
				ReadmissionRate:    data.ReadmissionRate,
				AvgEncountersPerPt: data.AvgEncountersPerPt,
			}
		}
	}

	// Convert monthly trend
	report.MonthlyTrend = make([]MonthlyUtilization, len(utilData.MonthlyTrend))
	for i, m := range utilData.MonthlyTrend {
		report.MonthlyTrend[i] = MonthlyUtilization{
			Month:               m.Month,
			InpatientAdmissions: m.InpatientAdmissions,
			EDVisits:            m.EDVisits,
			OutpatientVisits:    m.OutpatientVisits,
			Readmissions:        m.Readmissions,
		}
	}

	return report, nil
}
