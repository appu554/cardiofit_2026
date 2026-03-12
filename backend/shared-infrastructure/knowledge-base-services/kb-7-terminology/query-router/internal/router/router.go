package router

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cardiofit/kb7-query-router/internal/cache"
	"github.com/cardiofit/kb7-query-router/internal/elasticsearch"
	"github.com/cardiofit/kb7-query-router/internal/graphdb"
	"github.com/cardiofit/kb7-query-router/internal/postgres"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
	"github.com/sony/gobreaker"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

// QueryIntent defines the type of query being performed
type QueryIntent int

const (
	LookupIntent QueryIntent = iota // Fast exact code lookup
	ReasoningIntent                 // Semantic reasoning/subsumption
	MappingIntent                   // Cross-terminology mapping
	SearchIntent                    // Fuzzy text search
	RelationshipIntent              // Concept relationship traversal
	AdvancedSearchIntent            // Advanced Elasticsearch search
	AutocompleteIntent              // Autocomplete suggestions
)

// QueryDecision defines routing decision for query types
type QueryDecision struct {
	Intent            QueryIntent
	TargetStore       string
	CacheableMinutes  int
	PerformanceTarget time.Duration
}

// Query routing matrix
var QueryRouting = map[string]QueryDecision{
	"exact_code_lookup":    {LookupIntent, "postgresql", 60, 10 * time.Millisecond},
	"subsumption_query":    {ReasoningIntent, "graphdb", 30, 50 * time.Millisecond},
	"cross_terminology":   {MappingIntent, "postgresql", 120, 15 * time.Millisecond},
	"drug_interaction":     {ReasoningIntent, "graphdb", 15, 100 * time.Millisecond},
	"concept_hierarchy":    {RelationshipIntent, "postgresql", 45, 25 * time.Millisecond},
	"fuzzy_text_search":    {SearchIntent, "postgresql", 30, 50 * time.Millisecond},
	"advanced_search":      {AdvancedSearchIntent, "elasticsearch", 15, 25 * time.Millisecond},
	"semantic_search":      {AdvancedSearchIntent, "elasticsearch", 20, 35 * time.Millisecond},
	"hybrid_search":        {AdvancedSearchIntent, "elasticsearch", 10, 40 * time.Millisecond},
	"autocomplete":         {AutocompleteIntent, "elasticsearch", 5, 15 * time.Millisecond},
}

// Prometheus metrics
var (
	queryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "kb7_query_duration_seconds",
			Help: "The duration of queries by intent and store",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"intent", "store", "status"},
	)
	cacheHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kb7_cache_hits_total",
			Help: "The total number of cache hits",
		},
		[]string{"query_type"},
	)
	cacheMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kb7_cache_misses_total",
			Help: "The total number of cache misses",
		},
		[]string{"query_type"},
	)
	queryErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kb7_query_errors_total",
			Help: "The total number of query errors",
		},
		[]string{"intent", "store", "error_type"},
	)
)

// HybridQueryRouter manages intelligent query routing
type HybridQueryRouter struct {
	postgres       *postgres.Client
	graphdb        *graphdb.Client
	elasticsearch  *elasticsearch.Client
	cache          *cache.RedisClient
	logger         *logrus.Logger
	tracer         trace.Tracer
	metrics        *QueryMetrics
	cbPostgres     *gobreaker.CircuitBreaker
	cbGraphDB      *gobreaker.CircuitBreaker
	cbElasticsearch *gobreaker.CircuitBreaker
	mu             sync.RWMutex
}

// QueryMetrics tracks performance metrics
type QueryMetrics struct {
	PostgresQueries     int64                        `json:"postgres_queries"`
	GraphDBQueries      int64                        `json:"graphdb_queries"`
	ElasticsearchQueries int64                        `json:"elasticsearch_queries"`
	CacheHits           int64                        `json:"cache_hits"`
	CacheMisses         int64                        `json:"cache_misses"`
	AverageLatency      map[string]time.Duration     `json:"average_latency"`
	ErrorCounts         map[string]int64             `json:"error_counts"`
	LastUpdated         time.Time                    `json:"last_updated"`
	mu                  sync.RWMutex
}

// HealthStatus represents the health of the service
type HealthStatus struct {
	Healthy   bool              `json:"healthy"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
	Version   string            `json:"version"`
}

// ReadinessStatus represents the readiness of the service
type ReadinessStatus struct {
	Ready     bool              `json:"ready"`
	Timestamp time.Time         `json:"timestamp"`
	Checks    map[string]string `json:"checks"`
}

// NewHybridQueryRouter creates a new hybrid query router
func NewHybridQueryRouter(
	postgres *postgres.Client,
	graphdb *graphdb.Client,
	elasticsearch *elasticsearch.Client,
	cache *cache.RedisClient,
	logger *logrus.Logger,
) *HybridQueryRouter {
	// Circuit breaker settings
	cbSettings := gobreaker.Settings{
		Name:        "query-router",
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     60 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures > 5
		},
	}

	esSettings := cbSettings
	esSettings.Name = "elasticsearch"

	return &HybridQueryRouter{
		postgres:       postgres,
		graphdb:        graphdb,
		elasticsearch:  elasticsearch,
		cache:          cache,
		logger:         logger,
		tracer:         otel.Tracer("kb7-query-router"),
		metrics:        newQueryMetrics(),
		cbPostgres:     gobreaker.NewCircuitBreaker(cbSettings),
		cbGraphDB:      gobreaker.NewCircuitBreaker(cbSettings),
		cbElasticsearch: gobreaker.NewCircuitBreaker(esSettings),
	}
}

func newQueryMetrics() *QueryMetrics {
	return &QueryMetrics{
		AverageLatency: make(map[string]time.Duration),
		ErrorCounts:    make(map[string]int64),
		LastUpdated:    time.Now(),
	}
}

// HandleConceptLookup handles exact concept lookups (PostgreSQL route)
func (h *HybridQueryRouter) HandleConceptLookup(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "concept-lookup")
	defer span.End()

	system := c.Param("system")
	code := c.Param("code")

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		queryDuration.WithLabelValues("lookup", "postgresql", "success").Observe(duration.Seconds())
		h.updateMetrics(LookupIntent, duration)
	}()

	// Check cache first
	cacheKey := fmt.Sprintf("concept:%s:%s", system, code)
	if cached, err := h.cache.Get(ctx, cacheKey); err == nil {
		cacheHits.WithLabelValues("concept_lookup").Inc()
		h.metrics.incrementCacheHits()
		c.JSON(http.StatusOK, cached)
		return
	}

	cacheMisses.WithLabelValues("concept_lookup").Inc()
	h.metrics.incrementCacheMisses()

	// Query PostgreSQL
	concept, err := h.cbPostgres.Execute(func() (interface{}, error) {
		return h.postgres.GetConcept(ctx, system, code)
	})

	if err != nil {
		queryErrors.WithLabelValues("lookup", "postgresql", "query_error").Inc()
		h.metrics.incrementError("postgresql_error")
		h.logger.WithError(err).Error("Failed to lookup concept")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if concept == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Concept not found"})
		return
	}

	// Cache the result
	h.cache.Set(ctx, cacheKey, concept, 1*time.Hour)
	h.metrics.incrementPostgresQueries()

	c.JSON(http.StatusOK, concept)
}

// HandleSubconceptQuery handles subsumption queries (GraphDB route)
func (h *HybridQueryRouter) HandleSubconceptQuery(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "subconcept-query")
	defer span.End()

	system := c.Param("system")
	code := c.Param("code")
	limitStr := c.DefaultQuery("limit", "50")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter"})
		return
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		queryDuration.WithLabelValues("reasoning", "graphdb", "success").Observe(duration.Seconds())
		h.updateMetrics(ReasoningIntent, duration)
	}()

	// Check cache for subconcepts
	cacheKey := fmt.Sprintf("subconcepts:%s:%s:%d", system, code, limit)
	if cached, err := h.cache.Get(ctx, cacheKey); err == nil {
		cacheHits.WithLabelValues("subconcept_query").Inc()
		h.metrics.incrementCacheHits()
		c.JSON(http.StatusOK, cached)
		return
	}

	cacheMisses.WithLabelValues("subconcept_query").Inc()
	h.metrics.incrementCacheMisses()

	// Query GraphDB for subconcepts
	subconcepts, err := h.cbGraphDB.Execute(func() (interface{}, error) {
		return h.graphdb.FindSubconcepts(ctx, system, code, limit)
	})

	if err != nil {
		queryErrors.WithLabelValues("reasoning", "graphdb", "query_error").Inc()
		h.metrics.incrementError("graphdb_error")
		h.logger.WithError(err).Error("Failed to find subconcepts")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Cache the result for 30 minutes
	h.cache.Set(ctx, cacheKey, subconcepts, 30*time.Minute)
	h.metrics.incrementGraphDBQueries()

	c.JSON(http.StatusOK, subconcepts)
}

// HandleMappingQuery handles cross-terminology mapping (PostgreSQL route)
func (h *HybridQueryRouter) HandleMappingQuery(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "mapping-query")
	defer span.End()

	fromSystem := c.Param("fromSystem")
	fromCode := c.Param("fromCode")
	toSystem := c.Param("toSystem")

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		queryDuration.WithLabelValues("mapping", "postgresql", "success").Observe(duration.Seconds())
		h.updateMetrics(MappingIntent, duration)
	}()

	// Check cache for mapping
	cacheKey := fmt.Sprintf("mapping:%s:%s:%s", fromSystem, fromCode, toSystem)
	if cached, err := h.cache.Get(ctx, cacheKey); err == nil {
		cacheHits.WithLabelValues("mapping_query").Inc()
		h.metrics.incrementCacheHits()
		c.JSON(http.StatusOK, cached)
		return
	}

	cacheMisses.WithLabelValues("mapping_query").Inc()
	h.metrics.incrementCacheMisses()

	// Query PostgreSQL for mapping
	mapping, err := h.cbPostgres.Execute(func() (interface{}, error) {
		return h.postgres.GetMapping(ctx, fromSystem, fromCode, toSystem)
	})

	if err != nil {
		queryErrors.WithLabelValues("mapping", "postgresql", "query_error").Inc()
		h.metrics.incrementError("postgresql_error")
		h.logger.WithError(err).Error("Failed to get mapping")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	if mapping == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Mapping not found"})
		return
	}

	// Cache the result for 2 hours
	h.cache.Set(ctx, cacheKey, mapping, 2*time.Hour)
	h.metrics.incrementPostgresQueries()

	c.JSON(http.StatusOK, mapping)
}

// HandleDrugInteractions handles drug interaction queries (GraphDB route)
func (h *HybridQueryRouter) HandleDrugInteractions(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "drug-interactions")
	defer span.End()

	var request struct {
		MedicationCodes []string `json:"medication_codes" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		queryDuration.WithLabelValues("reasoning", "graphdb", "success").Observe(duration.Seconds())
		h.updateMetrics(ReasoningIntent, duration)
	}()

	// Check cache for drug interactions
	cacheKey := fmt.Sprintf("interactions:%s", strings.Join(request.MedicationCodes, ","))
	if cached, err := h.cache.Get(ctx, cacheKey); err == nil {
		cacheHits.WithLabelValues("drug_interactions").Inc()
		h.metrics.incrementCacheHits()
		c.JSON(http.StatusOK, cached)
		return
	}

	cacheMisses.WithLabelValues("drug_interactions").Inc()
	h.metrics.incrementCacheMisses()

	// Query GraphDB for drug interactions
	interactions, err := h.cbGraphDB.Execute(func() (interface{}, error) {
		return h.graphdb.CheckDrugInteractions(ctx, request.MedicationCodes)
	})

	if err != nil {
		queryErrors.WithLabelValues("reasoning", "graphdb", "query_error").Inc()
		h.metrics.incrementError("graphdb_error")
		h.logger.WithError(err).Error("Failed to check drug interactions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Cache the result for 15 minutes
	h.cache.Set(ctx, cacheKey, interactions, 15*time.Minute)
	h.metrics.incrementGraphDBQueries()

	c.JSON(http.StatusOK, interactions)
}

// HandleRelationshipQuery handles concept relationship queries (Hybrid route)
func (h *HybridQueryRouter) HandleRelationshipQuery(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "relationship-query")
	defer span.End()

	system := c.Param("system")
	code := c.Param("code")
	relType := c.DefaultQuery("type", "all")

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		queryDuration.WithLabelValues("relationship", "hybrid", "success").Observe(duration.Seconds())
		h.updateMetrics(RelationshipIntent, duration)
	}()

	// Check cache for relationships
	cacheKey := fmt.Sprintf("relationships:%s:%s:%s", system, code, relType)
	if cached, err := h.cache.Get(ctx, cacheKey); err == nil {
		cacheHits.WithLabelValues("relationship_query").Inc()
		h.metrics.incrementCacheHits()
		c.JSON(http.StatusOK, cached)
		return
	}

	cacheMisses.WithLabelValues("relationship_query").Inc()
	h.metrics.incrementCacheMisses()

	// Query both stores and combine results
	pgRelationships, err := h.cbPostgres.Execute(func() (interface{}, error) {
		return h.postgres.GetRelationships(ctx, system, code, relType)
	})

	if err != nil {
		h.logger.WithError(err).Warn("PostgreSQL relationship query failed")
	}

	graphRelationships, err := h.cbGraphDB.Execute(func() (interface{}, error) {
		return h.graphdb.GetRelationships(ctx, system, code, relType)
	})

	if err != nil {
		h.logger.WithError(err).Warn("GraphDB relationship query failed")
	}

	// Combine results
	result := map[string]interface{}{
		"postgres_relationships": pgRelationships,
		"graph_relationships":    graphRelationships,
		"combined":               true,
	}

	// Cache the result for 45 minutes
	h.cache.Set(ctx, cacheKey, result, 45*time.Minute)
	h.metrics.incrementPostgresQueries()
	h.metrics.incrementGraphDBQueries()

	c.JSON(http.StatusOK, result)
}

// HandleTextSearch handles fuzzy text search (PostgreSQL route)
func (h *HybridQueryRouter) HandleTextSearch(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "text-search")
	defer span.End()

	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter 'q' is required"})
		return
	}

	system := c.DefaultQuery("system", "all")
	limitStr := c.DefaultQuery("limit", "20")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid limit parameter"})
		return
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		queryDuration.WithLabelValues("search", "postgresql", "success").Observe(duration.Seconds())
		h.updateMetrics(SearchIntent, duration)
	}()

	// Check cache for search results
	cacheKey := fmt.Sprintf("search:%s:%s:%d", query, system, limit)
	if cached, err := h.cache.Get(ctx, cacheKey); err == nil {
		cacheHits.WithLabelValues("text_search").Inc()
		h.metrics.incrementCacheHits()
		c.JSON(http.StatusOK, cached)
		return
	}

	cacheMisses.WithLabelValues("text_search").Inc()
	h.metrics.incrementCacheMisses()

	// Query PostgreSQL for search results
	results, err := h.cbPostgres.Execute(func() (interface{}, error) {
		return h.postgres.SearchConcepts(ctx, query, system, limit)
	})

	if err != nil {
		queryErrors.WithLabelValues("search", "postgresql", "query_error").Inc()
		h.metrics.incrementError("postgresql_error")
		h.logger.WithError(err).Error("Failed to search concepts")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Cache the result for 30 minutes
	h.cache.Set(ctx, cacheKey, results, 30*time.Minute)
	h.metrics.incrementPostgresQueries()

	c.JSON(http.StatusOK, results)
}

// HandleAdvancedSearch handles advanced Elasticsearch-powered search
func (h *HybridQueryRouter) HandleAdvancedSearch(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "advanced-search")
	defer span.End()

	// Parse request - can be GET with query params or POST with JSON body
	var searchReq *elasticsearch.SearchRequest
	var err error

	if c.Request.Method == "POST" {
		// Parse JSON request body
		if err := c.ShouldBindJSON(&searchReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else {
		// Parse GET parameters
		searchReq = &elasticsearch.SearchRequest{
			Query:               c.Query("q"),
			Systems:             parseCommaSeparated(c.Query("systems")),
			Mode:                c.DefaultQuery("mode", "standard"),
			MaxResults:          parseIntDefault(c.Query("limit"), 10),
			Offset:              parseIntDefault(c.Query("offset"), 0),
			IncludeHighlights:   c.DefaultQuery("highlights", "true") == "true",
			IncludeFacets:       c.DefaultQuery("facets", "false") == "true",
		}

		if searchReq.Query == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter 'q' is required"})
			return
		}
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		queryDuration.WithLabelValues("advanced_search", "elasticsearch", "success").Observe(duration.Seconds())
		h.updateMetrics(AdvancedSearchIntent, duration)
	}()

	// Check cache for advanced search results
	cacheKey := fmt.Sprintf("advanced_search:%s:%v:%s:%d:%d",
		searchReq.Query, searchReq.Systems, searchReq.Mode, searchReq.MaxResults, searchReq.Offset)
	if cached, err := h.cache.Get(ctx, cacheKey); err == nil {
		cacheHits.WithLabelValues("advanced_search").Inc()
		h.metrics.incrementCacheHits()
		c.JSON(http.StatusOK, cached)
		return
	}

	cacheMisses.WithLabelValues("advanced_search").Inc()
	h.metrics.incrementCacheMisses()

	// Execute advanced search via Elasticsearch
	results, err := h.cbElasticsearch.Execute(func() (interface{}, error) {
		return h.elasticsearch.Search(ctx, searchReq)
	})

	if err != nil {
		queryErrors.WithLabelValues("advanced_search", "elasticsearch", "query_error").Inc()
		h.metrics.incrementError("elasticsearch_error")
		h.logger.WithError(err).Error("Failed to execute advanced search")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Determine cache duration based on query complexity
	cacheDuration := 15 * time.Minute
	if searchReq.Mode == "semantic" || searchReq.Mode == "hybrid" {
		cacheDuration = 10 * time.Minute
	}

	// Cache the result
	h.cache.Set(ctx, cacheKey, results, cacheDuration)
	h.metrics.incrementElasticsearchQueries()

	c.JSON(http.StatusOK, results)
}

// HandleAutocomplete handles autocomplete suggestions
func (h *HybridQueryRouter) HandleAutocomplete(c *gin.Context) {
	ctx, span := h.tracer.Start(c.Request.Context(), "autocomplete")
	defer span.End()

	// Parse request - can be GET with query params or POST with JSON body
	var autocompleteReq *elasticsearch.AutocompleteRequest
	var err error

	if c.Request.Method == "POST" {
		// Parse JSON request body
		if err := c.ShouldBindJSON(&autocompleteReq); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else {
		// Parse GET parameters
		autocompleteReq = &elasticsearch.AutocompleteRequest{
			Query:      c.Query("q"),
			Systems:    parseCommaSeparated(c.Query("systems")),
			MaxResults: parseIntDefault(c.Query("limit"), 10),
		}

		if autocompleteReq.Query == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Query parameter 'q' is required"})
			return
		}

		// Minimum query length for autocomplete
		if len(autocompleteReq.Query) < 2 {
			c.JSON(http.StatusOK, &elasticsearch.AutocompleteResponse{
				Suggestions: []elasticsearch.AutocompleteSuggestion{},
				QueryTimeMs: 0,
			})
			return
		}
	}

	start := time.Now()
	defer func() {
		duration := time.Since(start)
		queryDuration.WithLabelValues("autocomplete", "elasticsearch", "success").Observe(duration.Seconds())
		h.updateMetrics(AutocompleteIntent, duration)
	}()

	// Check cache for autocomplete suggestions
	cacheKey := fmt.Sprintf("autocomplete:%s:%v:%d",
		autocompleteReq.Query, autocompleteReq.Systems, autocompleteReq.MaxResults)
	if cached, err := h.cache.Get(ctx, cacheKey); err == nil {
		cacheHits.WithLabelValues("autocomplete").Inc()
		h.metrics.incrementCacheHits()
		c.JSON(http.StatusOK, cached)
		return
	}

	cacheMisses.WithLabelValues("autocomplete").Inc()
	h.metrics.incrementCacheMisses()

	// Execute autocomplete via Elasticsearch
	suggestions, err := h.cbElasticsearch.Execute(func() (interface{}, error) {
		return h.elasticsearch.GetAutocompleteSuggestions(ctx, autocompleteReq)
	})

	if err != nil {
		queryErrors.WithLabelValues("autocomplete", "elasticsearch", "query_error").Inc()
		h.metrics.incrementError("elasticsearch_error")
		h.logger.WithError(err).Error("Failed to get autocomplete suggestions")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// Cache autocomplete results for 5 minutes
	h.cache.Set(ctx, cacheKey, suggestions, 5*time.Minute)
	h.metrics.incrementElasticsearchQueries()

	c.JSON(http.StatusOK, suggestions)
}

// HandleMetrics returns query performance metrics
func (h *HybridQueryRouter) HandleMetrics(c *gin.Context) {
	h.metrics.mu.RLock()
	defer h.metrics.mu.RUnlock()

	c.JSON(http.StatusOK, h.metrics)
}

// HealthCheck performs health check on all dependencies
func (h *HybridQueryRouter) HealthCheck() HealthStatus {
	status := HealthStatus{
		Healthy:   true,
		Timestamp: time.Now(),
		Services:  make(map[string]string),
		Version:   "1.0.0",
	}

	// Check PostgreSQL
	if err := h.postgres.Ping(); err != nil {
		status.Healthy = false
		status.Services["postgresql"] = fmt.Sprintf("unhealthy: %v", err)
	} else {
		status.Services["postgresql"] = "healthy"
	}

	// Check GraphDB
	if err := h.graphdb.Ping(); err != nil {
		status.Healthy = false
		status.Services["graphdb"] = fmt.Sprintf("unhealthy: %v", err)
	} else {
		status.Services["graphdb"] = "healthy"
	}

	// Check Redis
	if err := h.cache.Ping(); err != nil {
		status.Healthy = false
		status.Services["redis"] = fmt.Sprintf("unhealthy: %v", err)
	} else {
		status.Services["redis"] = "healthy"
	}

	// Check Elasticsearch
	if err := h.elasticsearch.Ping(); err != nil {
		status.Healthy = false
		status.Services["elasticsearch"] = fmt.Sprintf("unhealthy: %v", err)
	} else {
		status.Services["elasticsearch"] = "healthy"
	}

	return status
}

// ReadinessCheck performs readiness check
func (h *HybridQueryRouter) ReadinessCheck() ReadinessStatus {
	status := ReadinessStatus{
		Ready:     true,
		Timestamp: time.Now(),
		Checks:    make(map[string]string),
	}

	// Check if we can perform a simple query
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test PostgreSQL readiness
	if _, err := h.postgres.GetConcept(ctx, "test", "test"); err != nil {
		status.Ready = false
		status.Checks["postgresql_query"] = fmt.Sprintf("failed: %v", err)
	} else {
		status.Checks["postgresql_query"] = "ready"
	}

	// Test cache readiness
	if err := h.cache.Set(ctx, "readiness_test", "ok", 1*time.Second); err != nil {
		status.Ready = false
		status.Checks["cache_write"] = fmt.Sprintf("failed: %v", err)
	} else {
		status.Checks["cache_write"] = "ready"
	}

	return status
}

func (h *HybridQueryRouter) updateMetrics(intent QueryIntent, duration time.Duration) {
	h.metrics.mu.Lock()
	defer h.metrics.mu.Unlock()

	intentStr := intentToString(intent)
	h.metrics.AverageLatency[intentStr] = duration
	h.metrics.LastUpdated = time.Now()
}

func (m *QueryMetrics) incrementCacheHits() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CacheHits++
}

func (m *QueryMetrics) incrementCacheMisses() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CacheMisses++
}

func (m *QueryMetrics) incrementPostgresQueries() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.PostgresQueries++
}

func (m *QueryMetrics) incrementGraphDBQueries() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.GraphDBQueries++
}

func (m *QueryMetrics) incrementElasticsearchQueries() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ElasticsearchQueries++
}

func (m *QueryMetrics) incrementError(errorType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ErrorCounts[errorType]++
}

func intentToString(intent QueryIntent) string {
	switch intent {
	case LookupIntent:
		return "lookup"
	case ReasoningIntent:
		return "reasoning"
	case MappingIntent:
		return "mapping"
	case SearchIntent:
		return "search"
	case RelationshipIntent:
		return "relationship"
	case AdvancedSearchIntent:
		return "advanced_search"
	case AutocompleteIntent:
		return "autocomplete"
	default:
		return "unknown"
	}
}

// Helper functions for parameter parsing
func parseCommaSeparated(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

func parseIntDefault(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	if val, err := strconv.Atoi(s); err == nil {
		return val
	}
	return defaultVal
}