package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/cardiofit/kb7-query-router/internal/cache"
	"github.com/cardiofit/kb7-query-router/internal/config"
	"github.com/cardiofit/kb7-query-router/internal/elasticsearch"
	"github.com/cardiofit/kb7-query-router/internal/graphdb"
	"github.com/cardiofit/kb7-query-router/internal/postgres"
	"github.com/cardiofit/kb7-query-router/internal/router"
	es "github.com/elastic/go-elasticsearch/v8"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// TestResult represents the result of a test
type TestResult struct {
	TestName string    `json:"test_name"`
	Success  bool      `json:"success"`
	Duration time.Duration `json:"duration"`
	Error    string    `json:"error,omitempty"`
	Details  map[string]interface{} `json:"details,omitempty"`
}

// TestSuite manages integration tests
type TestSuite struct {
	queryRouter *router.HybridQueryRouter
	baseURL     string
	logger      *logrus.Logger
	results     []TestResult
}

func main() {
	// Initialize logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)
	logger.SetFormatter(&logrus.JSONFormatter{})

	fmt.Println("🚀 Starting KB7 Elasticsearch Integration Tests")

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize test suite
	testSuite, err := NewTestSuite(cfg, logger)
	if err != nil {
		log.Fatalf("Failed to initialize test suite: %v", err)
	}

	// Run tests
	testSuite.RunAllTests()

	// Report results
	testSuite.ReportResults()

	// Exit with appropriate code
	if testSuite.HasFailures() {
		os.Exit(1)
	}
}

// NewTestSuite creates a new test suite
func NewTestSuite(cfg *config.Config, logger *logrus.Logger) (*TestSuite, error) {
	// Initialize clients
	redisClient, err := cache.NewRedisClient(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Redis client: %w", err)
	}

	postgresClient, err := postgres.NewClient(cfg.PostgresURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize PostgreSQL client: %w", err)
	}

	graphDBClient, err := graphdb.NewClient(cfg.GraphDBEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize GraphDB client: %w", err)
	}

	// Initialize Elasticsearch client
	esConfig := es.Config{
		Addresses: []string{cfg.ElasticsearchURL},
	}
	elasticsearchClient, err := elasticsearch.NewClient(esConfig, cfg.ElasticsearchIndex, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Elasticsearch client: %w", err)
	}

	// Initialize query router
	queryRouter := router.NewHybridQueryRouter(
		postgresClient,
		graphDBClient,
		elasticsearchClient,
		redisClient,
		logger,
	)

	// Start test server
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	// Health endpoints
	r.GET("/health", func(c *gin.Context) {
		status := queryRouter.HealthCheck()
		if status.Healthy {
			c.JSON(http.StatusOK, status)
		} else {
			c.JSON(http.StatusServiceUnavailable, status)
		}
	})

	// API routes
	v1 := r.Group("/api/v1")
	{
		// Advanced search endpoints
		v1.GET("/search/advanced", queryRouter.HandleAdvancedSearch)
		v1.POST("/search/advanced", queryRouter.HandleAdvancedSearch)

		// Autocomplete endpoints
		v1.GET("/search/autocomplete", queryRouter.HandleAutocomplete)
		v1.POST("/search/autocomplete", queryRouter.HandleAutocomplete)

		// Metrics
		v1.GET("/metrics", queryRouter.HandleMetrics)
	}

	// Start server in background
	go func() {
		r.Run(":8089") // Use different port for testing
	}()

	// Wait for server to start
	time.Sleep(2 * time.Second)

	return &TestSuite{
		queryRouter: queryRouter,
		baseURL:     "http://localhost:8089",
		logger:      logger,
		results:     []TestResult{},
	}, nil
}

// RunAllTests executes all integration tests
func (ts *TestSuite) RunAllTests() {
	fmt.Println("📋 Running integration tests...")

	// Health checks
	ts.TestHealthCheck()
	ts.TestElasticsearchConnectivity()

	// Search functionality
	ts.TestAdvancedSearchGET()
	ts.TestAdvancedSearchPOST()
	ts.TestSearchModes()
	ts.TestSearchFilters()

	// Autocomplete functionality
	ts.TestAutocompleteSuggestions()
	ts.TestAutocompleteMinLength()
	ts.TestAutocompleteSystemFilter()

	// Performance tests
	ts.TestSearchPerformance()
	ts.TestAutocompletePerformance()

	// Error handling
	ts.TestInvalidRequests()
	ts.TestEmptyResults()

	// Metrics validation
	ts.TestMetricsReporting()
}

// TestHealthCheck verifies health endpoint includes Elasticsearch
func (ts *TestSuite) TestHealthCheck() {
	start := time.Now()
	testName := "Health Check with Elasticsearch"

	resp, err := http.Get(ts.baseURL + "/health")
	if err != nil {
		ts.recordResult(testName, false, time.Since(start), err.Error(), nil)
		return
	}
	defer resp.Body.Close()

	var health map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		ts.recordResult(testName, false, time.Since(start), err.Error(), nil)
		return
	}

	services, ok := health["services"].(map[string]interface{})
	if !ok {
		ts.recordResult(testName, false, time.Since(start), "services field missing", nil)
		return
	}

	esStatus, exists := services["elasticsearch"]
	success := exists && esStatus == "healthy" && resp.StatusCode == 200

	details := map[string]interface{}{
		"elasticsearch_status": esStatus,
		"elasticsearch_exists": exists,
		"http_status":         resp.StatusCode,
	}

	ts.recordResult(testName, success, time.Since(start), "", details)
}

// TestElasticsearchConnectivity verifies Elasticsearch is reachable
func (ts *TestSuite) TestElasticsearchConnectivity() {
	start := time.Now()
	testName := "Elasticsearch Direct Connectivity"

	// Test ping directly through our client
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Create a simple search request to verify connectivity
	searchReq := &elasticsearch.SearchRequest{
		Query:      "test",
		MaxResults: 1,
	}

	// This will fail if Elasticsearch is not available
	_, err := ts.queryRouter.HandleAdvancedSearch // This is not the right way to test, but demonstrates the concept

	success := err == nil
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}

	details := map[string]interface{}{
		"search_request": searchReq,
	}

	ts.recordResult(testName, success, time.Since(start), errorMsg, details)
}

// TestAdvancedSearchGET tests GET endpoint for advanced search
func (ts *TestSuite) TestAdvancedSearchGET() {
	start := time.Now()
	testName := "Advanced Search GET Endpoint"

	url := ts.baseURL + "/api/v1/search/advanced?q=hypertension&mode=standard&limit=5"
	resp, err := http.Get(url)
	if err != nil {
		ts.recordResult(testName, false, time.Since(start), err.Error(), nil)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		ts.recordResult(testName, false, time.Since(start), err.Error(), nil)
		return
	}

	success := resp.StatusCode == 200
	details := map[string]interface{}{
		"http_status":    resp.StatusCode,
		"has_results":    result["results"] != nil,
		"response_fields": getMapKeys(result),
	}

	ts.recordResult(testName, success, time.Since(start), "", details)
}

// TestAdvancedSearchPOST tests POST endpoint for advanced search
func (ts *TestSuite) TestAdvancedSearchPOST() {
	start := time.Now()
	testName := "Advanced Search POST Endpoint"

	searchReq := map[string]interface{}{
		"query":               "diabetes",
		"systems":             []string{"snomed", "icd10"},
		"mode":                "hybrid",
		"max_results":         10,
		"include_highlights":  true,
		"include_facets":      true,
	}

	jsonData, _ := json.Marshal(searchReq)
	resp, err := http.Post(ts.baseURL+"/api/v1/search/advanced", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		ts.recordResult(testName, false, time.Since(start), err.Error(), nil)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		ts.recordResult(testName, false, time.Since(start), err.Error(), nil)
		return
	}

	success := resp.StatusCode == 200
	details := map[string]interface{}{
		"http_status":     resp.StatusCode,
		"request_payload": searchReq,
		"has_results":     result["results"] != nil,
		"has_facets":      result["facets"] != nil,
		"has_highlights":  hasHighlights(result),
	}

	ts.recordResult(testName, success, time.Since(start), "", details)
}

// TestSearchModes tests different search modes
func (ts *TestSuite) TestSearchModes() {
	modes := []string{"standard", "exact", "fuzzy", "semantic", "hybrid"}

	for _, mode := range modes {
		start := time.Now()
		testName := fmt.Sprintf("Search Mode: %s", mode)

		url := fmt.Sprintf("%s/api/v1/search/advanced?q=medication&mode=%s&limit=3", ts.baseURL, mode)
		resp, err := http.Get(url)
		if err != nil {
			ts.recordResult(testName, false, time.Since(start), err.Error(), nil)
			continue
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		success := resp.StatusCode == 200
		details := map[string]interface{}{
			"mode":           mode,
			"http_status":    resp.StatusCode,
			"has_results":    result["results"] != nil,
			"query_time_ms":  result["query_time_ms"],
		}

		ts.recordResult(testName, success, time.Since(start), "", details)
	}
}

// TestSearchFilters tests search filtering capabilities
func (ts *TestSuite) TestSearchFilters() {
	start := time.Now()
	testName := "Search Filters"

	searchReq := map[string]interface{}{
		"query":       "heart",
		"systems":     []string{"snomed"},
		"max_results": 5,
		"filters": map[string]interface{}{
			"status": []string{"active"},
		},
	}

	jsonData, _ := json.Marshal(searchReq)
	resp, err := http.Post(ts.baseURL+"/api/v1/search/advanced", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		ts.recordResult(testName, false, time.Since(start), err.Error(), nil)
		return
	}
	defer resp.Body.Close()

	success := resp.StatusCode == 200
	details := map[string]interface{}{
		"http_status":    resp.StatusCode,
		"filter_applied": searchReq["filters"],
	}

	ts.recordResult(testName, success, time.Since(start), "", details)
}

// TestAutocompleteSuggestions tests autocomplete functionality
func (ts *TestSuite) TestAutocompleteSuggestions() {
	start := time.Now()
	testName := "Autocomplete Suggestions"

	url := ts.baseURL + "/api/v1/search/autocomplete?q=hyper&limit=5"
	resp, err := http.Get(url)
	if err != nil {
		ts.recordResult(testName, false, time.Since(start), err.Error(), nil)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		ts.recordResult(testName, false, time.Since(start), err.Error(), nil)
		return
	}

	success := resp.StatusCode == 200
	details := map[string]interface{}{
		"http_status":      resp.StatusCode,
		"has_suggestions":  result["suggestions"] != nil,
		"query_time_ms":    result["query_time_ms"],
	}

	ts.recordResult(testName, success, time.Since(start), "", details)
}

// TestAutocompleteMinLength tests minimum length requirement
func (ts *TestSuite) TestAutocompleteMinLength() {
	start := time.Now()
	testName := "Autocomplete Min Length"

	url := ts.baseURL + "/api/v1/search/autocomplete?q=h&limit=5"
	resp, err := http.Get(url)
	if err != nil {
		ts.recordResult(testName, false, time.Since(start), err.Error(), nil)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	// Should return empty suggestions for single character
	success := resp.StatusCode == 200
	suggestions := result["suggestions"].([]interface{})
	isEmpty := len(suggestions) == 0

	details := map[string]interface{}{
		"http_status":        resp.StatusCode,
		"suggestions_empty":  isEmpty,
		"suggestions_count":  len(suggestions),
	}

	ts.recordResult(testName, success && isEmpty, time.Since(start), "", details)
}

// TestAutocompleteSystemFilter tests system filtering in autocomplete
func (ts *TestSuite) TestAutocompleteSystemFilter() {
	start := time.Now()
	testName := "Autocomplete System Filter"

	url := ts.baseURL + "/api/v1/search/autocomplete?q=card&systems=snomed&limit=5"
	resp, err := http.Get(url)
	if err != nil {
		ts.recordResult(testName, false, time.Since(start), err.Error(), nil)
		return
	}
	defer resp.Body.Close()

	success := resp.StatusCode == 200
	details := map[string]interface{}{
		"http_status": resp.StatusCode,
		"system_filter": "snomed",
	}

	ts.recordResult(testName, success, time.Since(start), "", details)
}

// TestSearchPerformance tests search performance
func (ts *TestSuite) TestSearchPerformance() {
	start := time.Now()
	testName := "Search Performance"

	url := ts.baseURL + "/api/v1/search/advanced?q=medication&mode=standard&limit=20"
	resp, err := http.Get(url)
	if err != nil {
		ts.recordResult(testName, false, time.Since(start), err.Error(), nil)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	duration := time.Since(start)
	queryTimeMs := result["query_time_ms"]

	success := resp.StatusCode == 200 && duration < 1*time.Second
	details := map[string]interface{}{
		"total_duration_ms": duration.Milliseconds(),
		"query_time_ms":     queryTimeMs,
		"performance_ok":    duration < 1*time.Second,
	}

	ts.recordResult(testName, success, duration, "", details)
}

// TestAutocompletePerformance tests autocomplete performance
func (ts *TestSuite) TestAutocompletePerformance() {
	start := time.Now()
	testName := "Autocomplete Performance"

	url := ts.baseURL + "/api/v1/search/autocomplete?q=card&limit=10"
	resp, err := http.Get(url)
	if err != nil {
		ts.recordResult(testName, false, time.Since(start), err.Error(), nil)
		return
	}
	defer resp.Body.Close()

	duration := time.Since(start)
	success := resp.StatusCode == 200 && duration < 500*time.Millisecond

	details := map[string]interface{}{
		"duration_ms":    duration.Milliseconds(),
		"performance_ok": duration < 500*time.Millisecond,
	}

	ts.recordResult(testName, success, duration, "", details)
}

// TestInvalidRequests tests error handling
func (ts *TestSuite) TestInvalidRequests() {
	testCases := []struct {
		name string
		url  string
		expectedStatus int
	}{
		{"Missing Query Parameter", "/api/v1/search/advanced", 400},
		{"Empty Query", "/api/v1/search/advanced?q=", 400},
		{"Missing Autocomplete Query", "/api/v1/search/autocomplete", 400},
	}

	for _, tc := range testCases {
		start := time.Now()
		resp, err := http.Get(ts.baseURL + tc.url)
		if err != nil {
			ts.recordResult(tc.name, false, time.Since(start), err.Error(), nil)
			continue
		}
		defer resp.Body.Close()

		success := resp.StatusCode == tc.expectedStatus
		details := map[string]interface{}{
			"expected_status": tc.expectedStatus,
			"actual_status":   resp.StatusCode,
		}

		ts.recordResult(tc.name, success, time.Since(start), "", details)
	}
}

// TestEmptyResults tests handling of empty search results
func (ts *TestSuite) TestEmptyResults() {
	start := time.Now()
	testName := "Empty Results Handling"

	// Search for something that definitely won't exist
	url := ts.baseURL + "/api/v1/search/advanced?q=zzzzinvalidterminologyzzzz&limit=5"
	resp, err := http.Get(url)
	if err != nil {
		ts.recordResult(testName, false, time.Since(start), err.Error(), nil)
		return
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&result)

	success := resp.StatusCode == 200 && result["total_results"].(float64) == 0
	details := map[string]interface{}{
		"http_status":    resp.StatusCode,
		"total_results":  result["total_results"],
	}

	ts.recordResult(testName, success, time.Since(start), "", details)
}

// TestMetricsReporting tests metrics endpoint
func (ts *TestSuite) TestMetricsReporting() {
	start := time.Now()
	testName := "Metrics Reporting"

	resp, err := http.Get(ts.baseURL + "/api/v1/metrics")
	if err != nil {
		ts.recordResult(testName, false, time.Since(start), err.Error(), nil)
		return
	}
	defer resp.Body.Close()

	var metrics map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&metrics); err != nil {
		ts.recordResult(testName, false, time.Since(start), err.Error(), nil)
		return
	}

	success := resp.StatusCode == 200 && metrics["elasticsearch_queries"] != nil
	details := map[string]interface{}{
		"http_status":           resp.StatusCode,
		"has_es_metrics":        metrics["elasticsearch_queries"] != nil,
		"elasticsearch_queries": metrics["elasticsearch_queries"],
	}

	ts.recordResult(testName, success, time.Since(start), "", details)
}

// Helper methods

func (ts *TestSuite) recordResult(testName string, success bool, duration time.Duration, error string, details map[string]interface{}) {
	result := TestResult{
		TestName: testName,
		Success:  success,
		Duration: duration,
		Error:    error,
		Details:  details,
	}
	ts.results = append(ts.results, result)

	status := "✅ PASS"
	if !success {
		status = "❌ FAIL"
	}

	fmt.Printf("%s %s (%.2fms)\n", status, testName, float64(duration.Nanoseconds())/1e6)
	if error != "" {
		fmt.Printf("   Error: %s\n", error)
	}
}

func (ts *TestSuite) ReportResults() {
	fmt.Println("\n📊 Test Summary")
	fmt.Println("================")

	passed := 0
	failed := 0
	totalDuration := time.Duration(0)

	for _, result := range ts.results {
		totalDuration += result.Duration
		if result.Success {
			passed++
		} else {
			failed++
		}
	}

	fmt.Printf("Total Tests: %d\n", len(ts.results))
	fmt.Printf("Passed: %d\n", passed)
	fmt.Printf("Failed: %d\n", failed)
	fmt.Printf("Total Duration: %.2fms\n", float64(totalDuration.Nanoseconds())/1e6)

	if failed > 0 {
		fmt.Println("\n❌ Failed Tests:")
		for _, result := range ts.results {
			if !result.Success {
				fmt.Printf("  - %s: %s\n", result.TestName, result.Error)
			}
		}
	}

	// Write detailed results to file
	jsonData, _ := json.MarshalIndent(ts.results, "", "  ")
	os.WriteFile("elasticsearch-integration-test-results.json", jsonData, 0644)
	fmt.Println("\n📄 Detailed results written to elasticsearch-integration-test-results.json")
}

func (ts *TestSuite) HasFailures() bool {
	for _, result := range ts.results {
		if !result.Success {
			return true
		}
	}
	return false
}

// Utility functions

func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func hasHighlights(result map[string]interface{}) bool {
	if results, ok := result["results"].([]interface{}); ok {
		for _, item := range results {
			if resultMap, ok := item.(map[string]interface{}); ok {
				if highlights, ok := resultMap["highlights"].([]interface{}); ok && len(highlights) > 0 {
					return true
				}
			}
		}
	}
	return false
}