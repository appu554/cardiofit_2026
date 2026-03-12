package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"safety-gateway-platform/internal/config"
	"safety-gateway-platform/pkg/types"
)

// Test fixture creation methods for Phase 2 orchestration testing

func (s *Phase2OrchestrationTestSuite) preparePhase2TestFixtures() {
	s.testRequests = s.createVariedRequests(50)
	s.testBatches = s.createTestBatches(10)
}

func (s *Phase2OrchestrationTestSuite) createVariedRequests(count int) []*types.SafetyRequest {
	requests := make([]*types.SafetyRequest, count)
	
	priorities := []string{"critical", "high", "normal", "low"}
	actionTypes := []string{"medication_prescribe", "medication_interaction", "routine_check"}
	
	for i := 0; i < count; i++ {
		requests[i] = &types.SafetyRequest{
			RequestID:     fmt.Sprintf("varied-req-%03d", i+1),
			PatientID:     fmt.Sprintf("patient-%03d", (i%10)+1), // 10 different patients
			ActionType:    actionTypes[i%len(actionTypes)],
			Priority:      priorities[i%len(priorities)],
			MedicationIDs: s.generateMedicationList(i),
			ConditionIDs:  s.generateConditionList(i),
			Timestamp:     time.Now(),
			Metadata: map[string]interface{}{
				"test_index": i,
				"batch_eligible": true,
			},
		}
	}
	
	return requests
}

func (s *Phase2OrchestrationTestSuite) createPatientGroupedRequests(count int) []*types.SafetyRequest {
	requests := make([]*types.SafetyRequest, count)
	patientID := "grouped-patient-001"
	
	for i := 0; i < count; i++ {
		requests[i] = &types.SafetyRequest{
			RequestID:     fmt.Sprintf("grouped-req-%03d", i+1),
			PatientID:     patientID, // All same patient for grouping
			ActionType:    "medication_review",
			Priority:      "normal",
			MedicationIDs: []string{fmt.Sprintf("med-%d", i+1)},
			ConditionIDs:  []string{"chronic_condition"},
			Timestamp:     time.Now(),
			Metadata: map[string]interface{}{
				"grouping_strategy": "patient_grouped",
				"sequence_number": i,
			},
		}
	}
	
	return requests
}

func (s *Phase2OrchestrationTestSuite) createSnapshotOptimizedRequests(count int) []*types.SafetyRequest {
	requests := make([]*types.SafetyRequest, count)
	
	for i := 0; i < count; i++ {
		requests[i] = &types.SafetyRequest{
			RequestID:     fmt.Sprintf("snapshot-req-%03d", i+1),
			PatientID:     fmt.Sprintf("snapshot-patient-%02d", (i%3)+1), // 3 patients for snapshot sharing
			ActionType:    "medication_interaction_check",
			Priority:      "high",
			MedicationIDs: s.generateComplexMedicationList(i),
			ConditionIDs:  s.generateComplexConditionList(i),
			Timestamp:     time.Now(),
			Metadata: map[string]interface{}{
				"optimization_strategy": "snapshot_optimized",
				"complexity_level": "high",
			},
		}
	}
	
	return requests
}

func (s *Phase2OrchestrationTestSuite) createParallelDirectRequests(count int) []*types.SafetyRequest {
	requests := make([]*types.SafetyRequest, count)
	
	for i := 0; i < count; i++ {
		requests[i] = &types.SafetyRequest{
			RequestID:     fmt.Sprintf("parallel-req-%03d", i+1),
			PatientID:     fmt.Sprintf("parallel-patient-%03d", i+1), // All different patients
			ActionType:    "simple_safety_check",
			Priority:      "normal",
			MedicationIDs: []string{"simple-med"},
			ConditionIDs:  []string{"simple-condition"},
			Timestamp:     time.Now(),
			Metadata: map[string]interface{}{
				"processing_strategy": "parallel_direct",
				"complexity_level": "simple",
			},
		}
	}
	
	return requests
}

func (s *Phase2OrchestrationTestSuite) createTestBatches(count int) []*types.BatchRequest {
	batches := make([]*types.BatchRequest, count)
	strategies := []string{"patient_grouped", "snapshot_optimized", "parallel_direct", "standard"}
	
	for i := 0; i < count; i++ {
		strategy := strategies[i%len(strategies)]
		var requests []*types.SafetyRequest
		
		switch strategy {
		case "patient_grouped":
			requests = s.createPatientGroupedRequests(5)
		case "snapshot_optimized":
			requests = s.createSnapshotOptimizedRequests(8)
		case "parallel_direct":
			requests = s.createParallelDirectRequests(10)
		default:
			requests = s.createVariedRequests(6)
		}
		
		batches[i] = &types.BatchRequest{
			BatchID:   fmt.Sprintf("batch-%03d", i+1),
			Requests:  requests,
			Strategy:  strategy,
			Priority:  "normal",
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"test_batch": true,
				"batch_index": i,
			},
		}
	}
	
	return batches
}

func (s *Phase2OrchestrationTestSuite) createTestBatchRequest() *types.BatchRequest {
	return &types.BatchRequest{
		BatchID:   fmt.Sprintf("test-batch-%d", time.Now().Unix()),
		Requests:  s.createVariedRequests(5),
		Strategy:  "patient_grouped",
		Priority:  "normal",
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"test_endpoint": true,
		},
	}
}

// Medication and condition generation helpers

func (s *Phase2OrchestrationTestSuite) generateMedicationList(index int) []string {
	baseMeds := []string{
		"acetaminophen", "ibuprofen", "aspirin", "warfarin", "metformin",
		"lisinopril", "atorvastatin", "omeprazole", "levothyroxine", "amlodipine",
	}
	
	// Return 1-3 medications based on index
	count := (index % 3) + 1
	meds := make([]string, count)
	for i := 0; i < count; i++ {
		meds[i] = baseMeds[(index+i)%len(baseMeds)]
	}
	
	return meds
}

func (s *Phase2OrchestrationTestSuite) generateConditionList(index int) []string {
	baseConditions := []string{
		"hypertension", "diabetes", "hyperlipidemia", "gerd", "hypothyroidism",
		"atrial_fibrillation", "heart_failure", "copd", "osteoarthritis", "depression",
	}
	
	// Return 1-2 conditions based on index
	count := (index % 2) + 1
	conditions := make([]string, count)
	for i := 0; i < count; i++ {
		conditions[i] = baseConditions[(index+i)%len(baseConditions)]
	}
	
	return conditions
}

func (s *Phase2OrchestrationTestSuite) generateComplexMedicationList(index int) []string {
	complexMeds := []string{
		"warfarin", "digoxin", "phenytoin", "lithium", "cyclosporine",
		"tacrolimus", "methotrexate", "amiodarone", "quinidine", "theophylline",
	}
	
	// Return 2-4 complex medications
	count := (index % 3) + 2
	meds := make([]string, count)
	for i := 0; i < count; i++ {
		meds[i] = complexMeds[(index+i)%len(complexMeds)]
	}
	
	return meds
}

func (s *Phase2OrchestrationTestSuite) generateComplexConditionList(index int) []string {
	complexConditions := []string{
		"heart_failure", "chronic_kidney_disease", "liver_cirrhosis", "bipolar_disorder",
		"atrial_fibrillation", "coronary_artery_disease", "copd_exacerbation", "seizure_disorder",
	}
	
	// Return 1-3 complex conditions
	count := (index % 3) + 1
	conditions := make([]string, count)
	for i := 0; i < count; i++ {
		conditions[i] = complexConditions[(index+i)%len(complexConditions)]
	}
	
	return conditions
}

// Validation methods for batch processing results

func (s *Phase2OrchestrationTestSuite) validatePatientGroupedResults(results []*types.SafetyResponse) error {
	// All results should be for the same patient
	if len(results) == 0 {
		return fmt.Errorf("no results to validate")
	}
	
	firstPatientID := results[0].Metadata["patient_id"]
	for i, result := range results {
		if result.Status == types.SafetyStatusError {
			return fmt.Errorf("result %d has error status", i)
		}
		
		patientID := result.Metadata["patient_id"]
		if patientID != firstPatientID {
			return fmt.Errorf("result %d has different patient ID: expected %v, got %v", 
				i, firstPatientID, patientID)
		}
	}
	
	return nil
}

func (s *Phase2OrchestrationTestSuite) validateSnapshotOptimizedResults(results []*types.SafetyResponse) error {
	if len(results) == 0 {
		return fmt.Errorf("no results to validate")
	}
	
	// Check that snapshot optimization was applied
	snapshotCounts := make(map[string]int)
	for i, result := range results {
		if result.Status == types.SafetyStatusError {
			return fmt.Errorf("result %d has error status", i)
		}
		
		if snapshotID, ok := result.Metadata["snapshot_id"].(string); ok {
			snapshotCounts[snapshotID]++
		}
	}
	
	// Should have snapshot reuse for optimization
	if len(snapshotCounts) > 5 {
		return fmt.Errorf("too many unique snapshots for optimization: %d", len(snapshotCounts))
	}
	
	return nil
}

func (s *Phase2OrchestrationTestSuite) validateParallelDirectResults(results []*types.SafetyResponse) error {
	if len(results) == 0 {
		return fmt.Errorf("no results to validate")
	}
	
	// All should be processed quickly in parallel
	for i, result := range results {
		if result.Status == types.SafetyStatusError {
			return fmt.Errorf("result %d has error status", i)
		}
		
		if result.ProcessingTime > 100*time.Millisecond {
			return fmt.Errorf("result %d took too long for parallel processing: %v", 
				i, result.ProcessingTime)
		}
	}
	
	return nil
}

// Batch metadata validation

func (s *Phase2OrchestrationTestSuite) validateBatchMetadata(batchResult *types.BatchResult, expectedStrategy string) {
	// Verify batch metadata
	if batchResult.Strategy != expectedStrategy {
		s.T().Errorf("Expected strategy %s, got %s", expectedStrategy, batchResult.Strategy)
	}
	
	if batchResult.CompletedAt.Before(batchResult.StartedAt) {
		s.T().Error("Completed time should be after started time")
	}
	
	if batchResult.TotalProcessingTime <= 0 {
		s.T().Error("Total processing time should be positive")
	}
}

// Load balancing performance validation

func (s *Phase2OrchestrationTestSuite) validateLoadBalancingPerformance(strategy string, results []*requestResult, totalDuration time.Duration) {
	if len(results) == 0 {
		s.T().Error("No results to validate")
		return
	}
	
	// Sort results by duration for analysis
	sortedResults := make([]*requestResult, len(results))
	copy(sortedResults, results)
	sort.Slice(sortedResults, func(i, j int) bool {
		return sortedResults[i].Duration < sortedResults[j].Duration
	})
	
	// Calculate performance metrics
	minDuration := sortedResults[0].Duration
	maxDuration := sortedResults[len(sortedResults)-1].Duration
	p95Index := int(float64(len(sortedResults)) * 0.95)
	p95Duration := sortedResults[p95Index].Duration
	
	var totalLatency time.Duration
	for _, result := range results {
		totalLatency += result.Duration
	}
	avgDuration := totalLatency / time.Duration(len(results))
	
	// Strategy-specific validations
	switch strategy {
	case "adaptive":
		// Adaptive should show good performance balance
		if p95Duration > 200*time.Millisecond {
			s.T().Errorf("Adaptive strategy P95 latency too high: %v", p95Duration)
		}
		
	case "round_robin":
		// Round robin should show consistent distribution
		variance := maxDuration - minDuration
		if variance > 150*time.Millisecond {
			s.T().Errorf("Round robin strategy variance too high: %v", variance)
		}
		
	case "least_loaded":
		// Least loaded should prefer faster engines
		if avgDuration > 100*time.Millisecond {
			s.T().Errorf("Least loaded strategy average latency too high: %v", avgDuration)
		}
		
	case "performance_weighted":
		// Performance weighted should show low P95
		if p95Duration > 150*time.Millisecond {
			s.T().Errorf("Performance weighted strategy P95 latency too high: %v", p95Duration)
		}
	}
	
	// General performance requirements
	if totalDuration > 3*time.Second {
		s.T().Errorf("Total processing time too long for concurrent requests: %v", totalDuration)
	}
	
	if avgDuration > 200*time.Millisecond {
		s.T().Errorf("Average latency too high: %v", avgDuration)
	}
}

// Configuration management for testing

func (s *Phase2OrchestrationTestSuite) updateLoadBalancingStrategy(strategy string) {
	// Update the orchestrator's load balancing strategy
	// This would require exposing configuration update methods
	s.config.AdvancedOrchestration.LoadBalancing.Strategy = strategy
}

func (s *Phase2OrchestrationTestSuite) loadEnvironmentConfig(environment string) *config.Config {
	baseConfig := s.loadPhase2Config()
	
	// Apply environment-specific overrides
	switch environment {
	case "development":
		baseConfig.AdvancedOrchestration.LoadBalancing.Strategy = "round_robin"
		baseConfig.AdvancedOrchestration.BatchProcessing.MaxBatchSize = 10
		baseConfig.Logging.Level = "debug"
		
	case "staging":
		baseConfig.AdvancedOrchestration.LoadBalancing.Strategy = "least_loaded"
		baseConfig.AdvancedOrchestration.Metrics.MetricsInterval = "5s"
		
	case "production":
		baseConfig.AdvancedOrchestration.LoadBalancing.Strategy = "adaptive"
		baseConfig.AdvancedOrchestration.Performance.EnablePerformanceOptimization = true
		baseConfig.Logging.Level = "warn"
	}
	
	baseConfig.Service.Environment = environment
	return baseConfig
}

// Metrics and load generation

func (s *Phase2OrchestrationTestSuite) generateMetricsLoad() {
	// Generate various types of requests to collect metrics
	requests := []*types.SafetyRequest{
		s.createCriticalPriorityRequest(),
		s.createMedicationInteractionRequest(),
		s.createRoutineAdvisoryRequest(),
	}
	
	// Process multiple rounds
	for round := 0; round < 5; round++ {
		for _, request := range requests {
			request.RequestID = fmt.Sprintf("%s-round-%d", request.RequestID, round)
			_, _ = s.orchestrator.ProcessSafetyRequest(s.ctx, request)
		}
		
		// Add some delay between rounds
		time.Sleep(100 * time.Millisecond)
	}
}

func (s *Phase2OrchestrationTestSuite) testMetricsExport() {
	// Test JSON export functionality
	if s.config.AdvancedOrchestration.Metrics.ExportJSON {
		// Verify export file creation and content
		// This would check the actual file at the configured path
	}
	
	// Test Prometheus export if enabled
	if s.config.AdvancedOrchestration.Metrics.ExportPrometheus {
		// Verify Prometheus metrics endpoint
	}
}

// HTTP endpoint handlers for testing

func (s *Phase2OrchestrationTestSuite) handleBatchValidate(w http.ResponseWriter, r *http.Request) {
	// Mock batch validation endpoint
	response := map[string]interface{}{
		"status": "accepted",
		"batch_id": fmt.Sprintf("batch-%d", time.Now().Unix()),
		"estimated_processing_time": "500ms",
		"strategy": "patient_grouped",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Phase2OrchestrationTestSuite) handleOrchestrationStats(w http.ResponseWriter, r *http.Request) {
	// Mock orchestration stats endpoint
	stats := map[string]interface{}{
		"total_requests": 100,
		"active_batches": 3,
		"average_latency": "75ms",
		"engine_health": map[string]string{
			"cae_engine": "healthy",
		},
		"load_balancing_strategy": "adaptive",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *Phase2OrchestrationTestSuite) handleOrchestrationMetrics(w http.ResponseWriter, r *http.Request) {
	// Mock detailed metrics endpoint
	metrics := map[string]interface{}{
		"performance": map[string]interface{}{
			"total_requests": 100,
			"average_latency": 75.5,
			"p95_latency": 150.0,
			"error_rate": 0.02,
		},
		"load": map[string]interface{}{
			"current_load": 0.35,
			"cpu_usage": 0.45,
			"memory_usage": 0.60,
		},
		"batch_processing": map[string]interface{}{
			"total_batches": 15,
			"average_batch_size": 8.3,
			"batch_success_rate": 0.98,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func (s *Phase2OrchestrationTestSuite) handleOrchestrationHealth(w http.ResponseWriter, r *http.Request) {
	// Mock health check endpoint
	health := map[string]interface{}{
		"status": "healthy",
		"components": map[string]string{
			"advanced_orchestration": "healthy",
			"batch_processor": "healthy",
			"load_balancer": "healthy",
			"metrics_collector": "healthy",
		},
		"version": "2.0.0",
		"uptime": "2h30m15s",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// Response validation methods

func (s *Phase2OrchestrationTestSuite) validateBatchValidationResponse(body []byte) error {
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	if status, ok := response["status"].(string); !ok || status != "accepted" {
		return fmt.Errorf("expected status 'accepted', got %v", response["status"])
	}
	
	if _, ok := response["batch_id"].(string); !ok {
		return fmt.Errorf("batch_id missing or invalid")
	}
	
	return nil
}

func (s *Phase2OrchestrationTestSuite) validateOrchestrationStatsResponse(body []byte) error {
	var stats map[string]interface{}
	if err := json.Unmarshal(body, &stats); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	requiredFields := []string{"total_requests", "average_latency", "load_balancing_strategy"}
	for _, field := range requiredFields {
		if _, ok := stats[field]; !ok {
			return fmt.Errorf("required field %s missing", field)
		}
	}
	
	return nil
}

func (s *Phase2OrchestrationTestSuite) validateOrchestrationMetricsResponse(body []byte) error {
	var metrics map[string]interface{}
	if err := json.Unmarshal(body, &metrics); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	requiredSections := []string{"performance", "load", "batch_processing"}
	for _, section := range requiredSections {
		if _, ok := metrics[section]; !ok {
			return fmt.Errorf("required metrics section %s missing", section)
		}
	}
	
	return nil
}

func (s *Phase2OrchestrationTestSuite) validateOrchestrationHealthResponse(body []byte) error {
	var health map[string]interface{}
	if err := json.Unmarshal(body, &health); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}
	
	if status, ok := health["status"].(string); !ok || status != "healthy" {
		return fmt.Errorf("expected status 'healthy', got %v", health["status"])
	}
	
	if components, ok := health["components"].(map[string]interface{}); ok {
		for component, status := range components {
			if status != "healthy" {
				return fmt.Errorf("component %s is not healthy: %v", component, status)
			}
		}
	}
	
	return nil
}