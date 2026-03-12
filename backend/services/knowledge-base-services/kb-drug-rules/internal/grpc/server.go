package grpc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"

	"kb-drug-rules/internal/cache"
	pb "kb-drug-rules/proto"
)

// DosingServer implements the KB-1 gRPC DosingService
// Designed for high-performance communication with Rust Dose Calculation Engine
// SLO Requirement: p95 latency < 60ms
type DosingServer struct {
	pb.UnimplementedDosingServiceServer
	db      *gorm.DB
	cache   cache.KB1CacheInterface
	logger  *logrus.Logger
	metrics MetricsCollector
}

// MetricsCollector interface for gRPC metrics (compatible with metrics.Collector)
type MetricsCollector interface {
	IncrementCounter(name string, labels map[string]string)
	RecordDuration(name string, duration time.Duration, labels map[string]string)
	RecordGauge(name string, value float64, labels map[string]string)
}

// NewDosingServer creates a new gRPC dosing server
func NewDosingServer(db *gorm.DB, cache cache.KB1CacheInterface, logger *logrus.Logger, metrics MetricsCollector) *DosingServer {
	return &DosingServer{
		db:      db,
		cache:   cache,
		logger:  logger,
		metrics: metrics,
	}
}

// GetDosingRuleset retrieves complete dosing ruleset for drug and patient context
// Primary KB-1 endpoint consumed by Rust Dose Calculation Engine
func (s *DosingServer) GetDosingRuleset(ctx context.Context, req *pb.DosingRuleRequest) (*pb.DosingRuleResponse, error) {
	startTime := time.Now()
	
	// Input validation
	if req.DrugCode == "" {
		s.metrics.IncrementCounter("grpc_requests_invalid", map[string]string{"reason": "missing_drug_code"})
		return nil, status.Error(codes.InvalidArgument, "drug_code is required")
	}
	
	if req.PatientContext == nil {
		s.metrics.IncrementCounter("grpc_requests_invalid", map[string]string{"reason": "missing_patient_context"})
		return nil, status.Error(codes.InvalidArgument, "patient_context is required")
	}

	// Generate context hash for cache key (KB-1 specification)
	contextHash := s.generateContextHash(req.PatientContext)
	cacheKey := fmt.Sprintf("dose:v2:%s:%s", req.DrugCode, contextHash)
	
	s.logger.WithFields(logrus.Fields{
		"transaction_id": req.TransactionId,
		"drug_code":      req.DrugCode,
		"version_pin":    req.VersionPin,
		"cache_key":      cacheKey,
	}).Debug("Processing dosing ruleset request")

	// Check Redis cache first (KB-1 write-through cache strategy)
	if cachedResponse, err := s.getCachedResponse(cacheKey); err == nil && cachedResponse != nil {
		s.metrics.IncrementCounter("grpc_cache_hits", map[string]string{"drug_code": req.DrugCode})
		s.recordLatency(startTime, "cache_hit", req.DrugCode)
		
		// Echo transaction_id in response
		cachedResponse.TransactionId = req.TransactionId
		return cachedResponse, nil
	}
	
	s.metrics.IncrementCounter("grpc_cache_misses", map[string]string{"drug_code": req.DrugCode})

	// Query active_dosing_rules materialized view for performance
	response, err := s.queryActiveDosingRules(ctx, req)
	if err != nil {
		s.recordLatency(startTime, "error", req.DrugCode)
		return nil, err
	}

	// Cache the response for future requests
	if err := s.cacheResponse(cacheKey, response); err != nil {
		s.logger.WithError(err).Warn("Failed to cache gRPC response")
	}

	s.recordLatency(startTime, "success", req.DrugCode)
	s.metrics.IncrementCounter("grpc_requests_success", map[string]string{"drug_code": req.DrugCode})
	
	return response, nil
}

// ValidatePatientContext validates patient context parameters
func (s *DosingServer) ValidatePatientContext(ctx context.Context, req *pb.PatientContext) (*pb.ValidationResponse, error) {
	startTime := time.Now()
	
	response := &pb.ValidationResponse{
		Valid: true,
		Errors: []string{},
		Warnings: []string{},
		RequiredFields: []string{},
	}

	// Basic validation rules
	if req.WeightKg <= 0 {
		response.Valid = false
		response.Errors = append(response.Errors, "weight_kg must be positive")
	}
	
	if req.AgeYears < 0 || req.AgeYears > 150 {
		response.Valid = false
		response.Errors = append(response.Errors, "age_years must be between 0 and 150")
	}
	
	if req.Egfr < 0 || req.Egfr > 200 {
		response.Warnings = append(response.Warnings, "eGFR value outside normal range (0-200)")
	}

	// Add required fields for common calculations
	response.RequiredFields = []string{"weight_kg", "age_years", "egfr"}
	
	s.recordLatency(startTime, "validation", "patient_context")
	return response, nil
}

// GetRuleMetadata retrieves metadata about available rules
func (s *DosingServer) GetRuleMetadata(ctx context.Context, req *pb.RuleMetadataRequest) (*pb.RuleMetadataResponse, error) {
	startTime := time.Now()
	
	if req.DrugCode == "" {
		return nil, status.Error(codes.InvalidArgument, "drug_code is required")
	}

	// Query from active_dosing_rules materialized view
	var rules []struct {
		SemanticVersion string `json:"semantic_version"`
		Provenance      string `json:"provenance"`
		CreatedAt       time.Time `json:"created_at"`
	}
	
	query := `
		SELECT semantic_version, provenance::text, created_at 
		FROM active_dosing_rules 
		WHERE drug_code = ? 
		ORDER BY semantic_version DESC
	`
	
	if err := s.db.Raw(query, req.DrugCode).Scan(&rules).Error; err != nil {
		s.logger.WithError(err).Error("Failed to query rule metadata")
		return nil, status.Error(codes.Internal, "failed to query rule metadata")
	}
	
	if len(rules) == 0 {
		return nil, status.Error(codes.NotFound, "no rules found for drug_code")
	}

	// Build response
	versions := make([]string, len(rules))
	for i, rule := range rules {
		versions[i] = rule.SemanticVersion
	}
	
	// Parse provenance for metadata
	var provenance map[string]interface{}
	if err := json.Unmarshal([]byte(rules[0].Provenance), &provenance); err != nil {
		s.logger.WithError(err).Warn("Failed to parse provenance")
	}

	metadata := &pb.RuleMetadata{
		SourceFile: fmt.Sprintf("%s_%s.toml", req.DrugCode, rules[0].SemanticVersion),
		CreatedAt:  rules[0].CreatedAt.Format(time.RFC3339),
	}
	
	// Extract authors from provenance
	if authors, ok := provenance["authors"].([]interface{}); ok {
		for _, author := range authors {
			if authorStr, ok := author.(string); ok {
				metadata.Authors = append(metadata.Authors, authorStr)
			}
		}
	}

	response := &pb.RuleMetadataResponse{
		DrugCode:          req.DrugCode,
		AvailableVersions: versions,
		LatestVersion:     versions[0],
		Metadata:          metadata,
		SupportedRegions:  []string{"US", "EU", "CA", "AU"},
	}

	s.recordLatency(startTime, "metadata", req.DrugCode)
	return response, nil
}

// CheckRuleAvailability checks if rules are available for a drug code
func (s *DosingServer) CheckRuleAvailability(ctx context.Context, req *pb.AvailabilityRequest) (*pb.AvailabilityResponse, error) {
	startTime := time.Now()
	
	if req.DrugCode == "" {
		return nil, status.Error(codes.InvalidArgument, "drug_code is required")
	}

	// Quick check in materialized view
	var count int64
	var latestVersion string
	
	query := `
		SELECT COUNT(*), MAX(semantic_version) 
		FROM active_dosing_rules 
		WHERE drug_code = ?
	`
	
	if err := s.db.Raw(query, req.DrugCode).Row().Scan(&count, &latestVersion); err != nil {
		s.logger.WithError(err).Error("Failed to check rule availability")
		return nil, status.Error(codes.Internal, "failed to check availability")
	}

	response := &pb.AvailabilityResponse{
		Available:        count > 0,
		LatestVersion:    latestVersion,
		SupportedRegions: []string{"US", "EU", "CA", "AU"},
	}
	
	if count == 0 {
		response.Message = "No active rules found for this drug code"
	} else {
		response.Message = fmt.Sprintf("Found %d active rule versions", count)
	}

	s.recordLatency(startTime, "availability", req.DrugCode)
	return response, nil
}

// Private helper methods

// generateContextHash creates hash from patient context for cache key
func (s *DosingServer) generateContextHash(ctx *pb.PatientContext) string {
	// Create normalized context for hashing (only fields that affect dosing)
	normalized := map[string]interface{}{
		"weight_kg": ctx.WeightKg,
		"egfr":      ctx.Egfr,
		"age_years": ctx.AgeYears,
		"sex":       ctx.Sex,
		"pregnant":  ctx.Pregnant,
	}
	
	// Add relevant extra parameters
	for key, value := range ctx.ExtraNumeric {
		normalized[key] = value
	}
	
	// Generate deterministic hash
	jsonBytes, _ := json.Marshal(normalized)
	hash := sha256.Sum256(jsonBytes)
	return hex.EncodeToString(hash[:8]) // Use first 8 bytes for shorter cache keys
}

// queryActiveDosingRules queries the materialized view for active rules
func (s *DosingServer) queryActiveDosingRules(ctx context.Context, req *pb.DosingRuleRequest) (*pb.DosingRuleResponse, error) {
	// Query the materialized view for optimal performance
	var result struct {
		RuleID          string          `json:"rule_id"`
		DrugCode        string          `json:"drug_code"`
		DrugName        string          `json:"drug_name"`
		SemanticVersion string          `json:"semantic_version"`
		CompiledJSON    json.RawMessage `json:"compiled_json"`
		Checksum        string          `json:"checksum"`
		Adjustments     json.RawMessage `json:"adjustments"`
		TitrationSchedule json.RawMessage `json:"titration_schedule"`
		PopulationRules json.RawMessage `json:"population_rules"`
		Provenance      json.RawMessage `json:"provenance"`
	}

	query := `
		SELECT rule_id, drug_code, drug_name, semantic_version, compiled_json, 
		       checksum, adjustments, titration_schedule, population_rules, provenance
		FROM active_dosing_rules 
		WHERE drug_code = ?
	`
	
	args := []interface{}{req.DrugCode}
	
	// Add version filter if specified
	if req.VersionPin != "" {
		query += " AND semantic_version = ?"
		args = append(args, req.VersionPin)
	}
	
	query += " ORDER BY semantic_version DESC LIMIT 1"

	if err := s.db.Raw(query, args...).Scan(&result).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, status.Error(codes.NotFound, "no active rules found for drug_code")
		}
		s.logger.WithError(err).Error("Failed to query active dosing rules")
		return nil, status.Error(codes.Internal, "database query failed")
	}

	// Build gRPC response
	response := &pb.DosingRuleResponse{
		TransactionId:   req.TransactionId,
		DrugCode:        result.DrugCode,
		SemanticVersion: result.SemanticVersion,
		Checksum:        result.Checksum,
		Warnings:        []string{}, // TODO: Implement clinical warnings based on patient context
	}

	// Include compiled JSON bundle if requested
	if req.IncludeCompiledBundle {
		response.CompiledJson = result.CompiledJSON
	}

	// Parse and include dose adjustments
	if len(result.Adjustments) > 0 {
		var adjustments []map[string]interface{}
		if err := json.Unmarshal(result.Adjustments, &adjustments); err == nil {
			response.Adjustments = s.convertToDoseAdjustments(adjustments, req.PatientContext)
		}
	}

	// Parse and include titration schedule
	if len(result.TitrationSchedule) > 0 {
		response.Schedule = &pb.TitrationSchedule{
			ScheduleJson: result.TitrationSchedule,
		}
		
		var titrationSteps []map[string]interface{}
		if err := json.Unmarshal(result.TitrationSchedule, &titrationSteps); err == nil {
			response.Schedule.Steps = s.convertToTitrationSteps(titrationSteps)
		}
	}

	// Parse and include population dosing
	if len(result.PopulationRules) > 0 {
		response.Population = &pb.PopulationDosing{
			PopulationJson: result.PopulationRules,
		}
		
		var populationRules []map[string]interface{}
		if err := json.Unmarshal(result.PopulationRules, &populationRules); err == nil {
			response.Population.Rules = s.convertToPopulationRules(populationRules, req.PatientContext)
		}
	}

	// Add provenance information
	response.ProvenanceJson = string(result.Provenance)

	// Add rule metadata
	response.Metadata = &pb.RuleMetadata{
		SourceFile: fmt.Sprintf("%s_%s.toml", result.DrugCode, result.SemanticVersion),
		CreatedAt:  time.Now().Format(time.RFC3339), // TODO: Get from actual creation time
	}

	return response, nil
}

// convertToDoseAdjustments converts database adjustments to protobuf format
func (s *DosingServer) convertToDoseAdjustments(adjustments []map[string]interface{}, patientCtx *pb.PatientContext) []*pb.DoseAdjustment {
	var result []*pb.DoseAdjustment
	
	for _, adj := range adjustments {
		adjustment := &pb.DoseAdjustment{
			AdjId:       getString(adj, "adj_id"),
			AdjustType:  getString(adj, "adjust_type"),
			Description: fmt.Sprintf("%s adjustment", getString(adj, "adjust_type")),
		}
		
		// Add formula and condition JSON
		if conditionJSON := getJSON(adj, "condition_json"); conditionJSON != nil {
			if bytes, err := json.Marshal(conditionJSON); err == nil {
				adjustment.ConditionJson = string(bytes)
			}
		}
		
		if formulaJSON := getJSON(adj, "formula_json"); formulaJSON != nil {
			if bytes, err := json.Marshal(formulaJSON); err == nil {
				adjustment.FormulaJson = string(bytes)
			}
		}

		// TODO: Evaluate condition against patient context to determine applicability
		// TODO: Calculate actual adjustment values based on patient parameters
		
		result = append(result, adjustment)
	}
	
	return result
}

// convertToTitrationSteps converts database titration to protobuf format
func (s *DosingServer) convertToTitrationSteps(steps []map[string]interface{}) []*pb.TitrationStep {
	var result []*pb.TitrationStep
	
	for _, step := range steps {
		titrationStep := &pb.TitrationStep{
			StepNumber:  int32(getInt(step, "step_number")),
			AfterDays:   int32(getInt(step, "after_days")),
			ActionType:  getString(step, "action_type"),
			ActionValue: getFloat64(step, "action_value"),
			MaxStep:     int32(getInt(step, "max_step")),
		}
		
		if monitoring := getJSON(step, "monitoring_requirements"); monitoring != nil {
			if bytes, err := json.Marshal(monitoring); err == nil {
				titrationStep.MonitoringRequired = string(bytes)
			}
		}
		
		result = append(result, titrationStep)
	}
	
	return result
}

// convertToPopulationRules converts database population rules to protobuf format
func (s *DosingServer) convertToPopulationRules(rules []map[string]interface{}, patientCtx *pb.PatientContext) []*pb.PopulationRule {
	var result []*pb.PopulationRule
	
	for _, rule := range rules {
		populationRule := &pb.PopulationRule{
			PopId:          getString(rule, "pop_id"),
			PopulationType: getString(rule, "population_type"),
			AgeMin:         int32(getInt(rule, "age_min")),
			AgeMax:         int32(getInt(rule, "age_max")),
			WeightMin:      getFloat64(rule, "weight_min"),
			WeightMax:      getFloat64(rule, "weight_max"),
		}
		
		if formulaJSON := getJSON(rule, "formula_json"); formulaJSON != nil {
			if bytes, err := json.Marshal(formulaJSON); err == nil {
				populationRule.FormulaJson = string(bytes)
			}
		}
		
		if safetyJSON := getJSON(rule, "safety_limits"); safetyJSON != nil {
			if bytes, err := json.Marshal(safetyJSON); err == nil {
				populationRule.SafetyLimitsJson = string(bytes)
			}
		}

		// TODO: Check if this population rule applies to the patient context
		
		result = append(result, populationRule)
	}
	
	return result
}

// Cache management methods

func (s *DosingServer) getCachedResponse(key string) (*pb.DosingRuleResponse, error) {
	ctx := context.Background()
	data, err := s.cache.Get(ctx, key)
	if err != nil || data == nil {
		return nil, err
	}

	var response pb.DosingRuleResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (s *DosingServer) cacheResponse(key string, response *pb.DosingRuleResponse) error {
	ctx := context.Background()
	data, err := json.Marshal(response)
	if err != nil {
		return err
	}

	// KB-1 specification: 1 hour TTL (3600 seconds) with immediate invalidation on rule updates
	return s.cache.Set(ctx, key, data, 3600)
}

// recordLatency records request latency for SLO monitoring
func (s *DosingServer) recordLatency(startTime time.Time, result, drugCode string) {
	duration := time.Since(startTime)
	s.metrics.RecordDuration("grpc_request_duration_seconds", duration, map[string]string{
		"result":    result,
		"drug_code": drugCode,
	})
	
	// Alert if approaching SLO violation (p95 < 60ms)
	if duration > 50*time.Millisecond {
		s.logger.WithFields(logrus.Fields{
			"duration_ms": duration.Milliseconds(),
			"drug_code":   drugCode,
			"result":      result,
		}).Warn("Request approaching SLO threshold")
	}
}

// Helper functions for JSON parsing

func getString(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getInt(data map[string]interface{}, key string) int {
	if val, ok := data[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case int64:
			return int(v)
		case float64:
			return int(v)
		}
	}
	return 0
}

func getFloat64(data map[string]interface{}, key string) float64 {
	if val, ok := data[key]; ok {
		if f, ok := val.(float64); ok {
			return f
		}
	}
	return 0.0
}

func getJSON(data map[string]interface{}, key string) interface{} {
	if val, ok := data[key]; ok {
		return val
	}
	return nil
}

// StartGRPCServer starts the gRPC server on the specified port
func StartGRPCServer(server *DosingServer, port int, logger *logrus.Logger) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", port, err)
	}

	// Create gRPC server with performance optimizations
	grpcServer := grpc.NewServer(
		grpc.MaxRecvMsgSize(1024*1024),    // 1MB max message size
		grpc.MaxSendMsgSize(1024*1024),    // 1MB max message size
		grpc.MaxConcurrentStreams(1000),   // Support high concurrency
	)

	// Register the dosing service
	pb.RegisterDosingServiceServer(grpcServer, server)

	logger.WithField("port", port).Info("Starting KB-1 gRPC server")
	
	// Start serving
	if err := grpcServer.Serve(listener); err != nil {
		return fmt.Errorf("gRPC server failed: %w", err)
	}
	
	return nil
}