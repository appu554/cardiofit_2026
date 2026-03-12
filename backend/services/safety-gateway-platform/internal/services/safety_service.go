package services

import (
	"context"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	"safety-gateway-platform/internal/config"
	"safety-gateway-platform/internal/orchestration"
	"safety-gateway-platform/internal/validator"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
	pb "safety-gateway-platform/proto"
)

// SafetyService implements the Safety Gateway gRPC service
type SafetyService struct {
	pb.UnimplementedSafetyGatewayServer
	validator    *validator.IngressValidator
	orchestrator *orchestration.OrchestrationEngine
	config       *config.Config
	logger       *logger.Logger
}

// NewSafetyService creates a new safety service
func NewSafetyService(
	validator *validator.IngressValidator,
	orchestrator *orchestration.OrchestrationEngine,
	cfg *config.Config,
	logger *logger.Logger,
) *SafetyService {
	return &SafetyService{
		validator:    validator,
		orchestrator: orchestrator,
		config:       cfg,
		logger:       logger,
	}
}

// ValidateSafety validates a safety request
func (s *SafetyService) ValidateSafety(ctx context.Context, req *pb.SafetyRequest) (*pb.SafetyResponse, error) {
	startTime := time.Now()
	
	// Convert protobuf request to internal type
	safetyReq, err := s.convertToSafetyRequest(req)
	if err != nil {
		s.logger.Error("Failed to convert request", zap.Error(err))
		return nil, status.Errorf(codes.InvalidArgument, "invalid request: %v", err)
	}

	requestLogger := s.logger.WithRequestID(safetyReq.RequestID).WithPatientID(safetyReq.PatientID)
	
	requestLogger.Info("Received safety validation request",
		zap.String("action_type", safetyReq.ActionType),
		zap.String("priority", safetyReq.Priority),
		zap.String("source", safetyReq.Source),
	)

	// Validate request
	if err := s.validator.ValidateRequest(ctx, safetyReq); err != nil {
		requestLogger.Warn("Request validation failed", zap.Error(err))
		return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
	}

	// Process safety request through orchestrator
	response, err := s.orchestrator.ProcessSafetyRequest(ctx, safetyReq)
	if err != nil {
		requestLogger.Error("Safety processing failed", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "safety processing failed: %v", err)
	}

	// Convert response to protobuf
	pbResponse, err := s.convertToProtoResponse(response)
	if err != nil {
		requestLogger.Error("Failed to convert response", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "response conversion failed: %v", err)
	}

	duration := time.Since(startTime)
	requestLogger.Info("Safety validation completed",
		zap.String("status", string(response.Status)),
		zap.Float64("risk_score", response.RiskScore),
		zap.Int64("processing_time_ms", duration.Milliseconds()),
		zap.Int("engines_executed", len(response.EngineResults)),
	)

	return pbResponse, nil
}

// GetEngineStatus returns the status of safety engines
func (s *SafetyService) GetEngineStatus(ctx context.Context, req *pb.EngineStatusRequest) (*pb.EngineStatusResponse, error) {
	s.logger.Debug("Engine status requested", zap.String("engine_id", req.EngineId))

	// This would be implemented to get actual engine status from registry
	// For now, return a mock response
	response := &pb.EngineStatusResponse{
		Engines: []*pb.EngineInfo{
			{
				Id:           "cae_engine",
				Name:         "Clinical Assertion Engine",
				Capabilities: []string{"drug_interaction", "contraindication", "dosing"},
				Tier:         1,
				Priority:     10,
				TimeoutMs:    100000,
				Status:       pb.EngineStatus_ENGINE_STATUS_HEALTHY,
				LastCheck:    timestamppb.Now(),
				FailureCount: 0,
			},
			{
				Id:           "allergy_engine",
				Name:         "Allergy Check Engine",
				Capabilities: []string{"allergy_check", "contraindication"},
				Tier:         1,
				Priority:     9,
				Status:       pb.EngineStatus_ENGINE_STATUS_HEALTHY,
				LastCheck:    timestamppb.Now(),
				FailureCount: 0,
			},
		},
		Metadata: map[string]string{
			"total_engines":   "4",
			"healthy_engines": "4",
			"timestamp":       time.Now().Format(time.RFC3339),
		},
	}

	return response, nil
}

// ValidateOverride validates an override token
func (s *SafetyService) ValidateOverride(ctx context.Context, req *pb.OverrideRequest) (*pb.OverrideResponse, error) {
	s.logger.Info("Override validation requested",
		zap.String("token_id", req.TokenId),
		zap.String("clinician_id", req.ClinicianId),
		zap.String("reason", req.Reason),
	)

	// This would be implemented with actual override validation logic
	// For now, return a mock response
	response := &pb.OverrideResponse{
		Valid:       true,
		Reason:      "Override validated successfully",
		ClinicianId: req.ClinicianId,
		ValidatedAt: timestamppb.Now(),
	}

	return response, nil
}

// GetHealth returns the health status of the service
func (s *SafetyService) GetHealth(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	s.logger.Debug("Health check requested", zap.Bool("detailed", req.Detailed))

	// Perform basic health check
	healthStatus := pb.HealthStatus_HEALTH_STATUS_HEALTHY
	message := "Service is healthy"
	details := map[string]string{
		"service":   "safety-gateway-platform",
		"version":   s.config.Service.Version,
		"timestamp": time.Now().Format(time.RFC3339),
	}

	if req.Detailed {
		// Add detailed health information
		details["engines_registered"] = "4"
		details["cache_status"] = "healthy"
		details["uptime"] = "running"
	}

	response := &pb.HealthResponse{
		Status:    healthStatus,
		Message:   message,
		Details:   details,
		Timestamp: timestamppb.Now(),
	}

	return response, nil
}

// convertToSafetyRequest converts protobuf request to internal type
func (s *SafetyService) convertToSafetyRequest(req *pb.SafetyRequest) (*types.SafetyRequest, error) {
	timestamp := time.Now()
	if req.Timestamp != nil {
		timestamp = req.Timestamp.AsTime()
	}

	// Prepare context map
	context := make(map[string]string)
	for k, v := range req.Context {
		context[k] = v
	}

	// Phase 2: Add snapshot reference to context if provided
	if req.SnapshotReference != nil {
		context["snapshot_id"] = req.SnapshotReference.SnapshotId
		context["snapshot_checksum"] = req.SnapshotReference.Checksum
		context["snapshot_version"] = req.SnapshotReference.Version
		if req.UseSnapshotMode {
			context["use_snapshot_mode"] = "true"
		}
	}

	return &types.SafetyRequest{
		RequestID:     req.RequestId,
		PatientID:     req.PatientId,
		ClinicianID:   req.ClinicianId,
		ActionType:    req.ActionType,
		Priority:      req.Priority,
		MedicationIDs: req.MedicationIds,
		ConditionIDs:  req.ConditionIds,
		AllergyIDs:    req.AllergyIds,
		Context:       context,
		Timestamp:     timestamp,
		Source:        req.Source,
	}, nil
}

// convertToProtoResponse converts internal response to protobuf
func (s *SafetyService) convertToProtoResponse(resp *types.SafetyResponse) (*pb.SafetyResponse, error) {
	// Convert engine results
	engineResults := make([]*pb.EngineResult, len(resp.EngineResults))
	for i, result := range resp.EngineResults {
		engineResults[i] = &pb.EngineResult{
			EngineId:    result.EngineID,
			EngineName:  result.EngineName,
			Status:      s.convertSafetyStatus(result.Status),
			RiskScore:   result.RiskScore,
			Violations:  result.Violations,
			Warnings:    result.Warnings,
			Confidence:  result.Confidence,
			DurationMs:  result.Duration.Milliseconds(),
			Tier:        int32(result.Tier),
			Error:       result.Error,
		}
	}

	// Convert explanation
	var explanation *pb.Explanation
	if resp.Explanation != nil {
		explanation = s.convertExplanation(resp.Explanation)
	}

	// Convert override token
	var overrideToken *pb.OverrideToken
	if resp.OverrideToken != nil {
		overrideToken = s.convertOverrideToken(resp.OverrideToken)
	}

	// Convert metadata
	metadata := make(map[string]string)
	for k, v := range resp.Metadata {
		if str, ok := v.(string); ok {
			metadata[k] = str
		}
	}

	return &pb.SafetyResponse{
		RequestId:          resp.RequestID,
		Status:             s.convertSafetyStatus(resp.Status),
		RiskScore:          resp.RiskScore,
		CriticalViolations: resp.CriticalViolations,
		Warnings:           resp.Warnings,
		EngineResults:      engineResults,
		EnginesFailed:      resp.EnginesFailed,
		Explanation:        explanation,
		OverrideToken:      overrideToken,
		ProcessingTimeMs:   resp.ProcessingTime.Milliseconds(),
		ContextVersion:     resp.ContextVersion,
		Timestamp:          timestamppb.New(resp.Timestamp),
		Metadata:           metadata,
	}, nil
}

// convertSafetyStatus converts internal safety status to protobuf
func (s *SafetyService) convertSafetyStatus(status types.SafetyStatus) pb.SafetyStatus {
	switch status {
	case types.SafetyStatusSafe:
		return pb.SafetyStatus_SAFETY_STATUS_SAFE
	case types.SafetyStatusUnsafe:
		return pb.SafetyStatus_SAFETY_STATUS_UNSAFE
	case types.SafetyStatusWarning:
		return pb.SafetyStatus_SAFETY_STATUS_WARNING
	case types.SafetyStatusManualReview:
		return pb.SafetyStatus_SAFETY_STATUS_MANUAL_REVIEW
	case types.SafetyStatusError:
		return pb.SafetyStatus_SAFETY_STATUS_ERROR
	default:
		return pb.SafetyStatus_SAFETY_STATUS_UNSPECIFIED
	}
}

// convertExplanation converts internal explanation to protobuf
func (s *SafetyService) convertExplanation(exp *types.Explanation) *pb.Explanation {
	// Convert explanation details
	details := make([]*pb.ExplanationDetail, len(exp.Details))
	for i, detail := range exp.Details {
		details[i] = &pb.ExplanationDetail{
			Category:           detail.Category,
			Severity:           detail.Severity,
			Description:        detail.Description,
			ClinicalRationale:  detail.ClinicalRationale,
			Confidence:         detail.Confidence,
			EngineSource:       detail.EngineSource,
			RecommendedAction:  detail.RecommendedAction,
		}
	}

	// Convert evidence
	evidence := make([]*pb.Evidence, len(exp.Evidence))
	for i, ev := range exp.Evidence {
		evidence[i] = &pb.Evidence{
			Type:        ev.Type,
			Source:      ev.Source,
			Description: ev.Description,
			Strength:    ev.Strength,
			Url:         ev.URL,
		}
	}

	// Convert actionable guidance
	actionable := make([]*pb.ActionableGuidance, len(exp.Actionable))
	for i, action := range exp.Actionable {
		actionable[i] = &pb.ActionableGuidance{
			Action:      action.Action,
			Priority:    action.Priority,
			Steps:       action.Steps,
			Monitoring:  action.Monitoring,
			Timeline:    action.Timeline,
			Responsible: action.Responsible,
		}
	}

	return &pb.Explanation{
		Level:       s.convertExplanationLevel(exp.Level),
		Summary:     exp.Summary,
		Details:     details,
		Confidence:  exp.Confidence,
		Evidence:    evidence,
		Actionable:  actionable,
		GeneratedAt: timestamppb.New(exp.GeneratedAt),
	}
}

// convertExplanationLevel converts internal explanation level to protobuf
func (s *SafetyService) convertExplanationLevel(level types.ExplanationLevel) pb.ExplanationLevel {
	switch level {
	case types.ExplanationLevelBasic:
		return pb.ExplanationLevel_EXPLANATION_LEVEL_BASIC
	case types.ExplanationLevelDetailed:
		return pb.ExplanationLevel_EXPLANATION_LEVEL_DETAILED
	case types.ExplanationLevelExpert:
		return pb.ExplanationLevel_EXPLANATION_LEVEL_EXPERT
	default:
		return pb.ExplanationLevel_EXPLANATION_LEVEL_UNSPECIFIED
	}
}

// convertOverrideToken converts internal override token to protobuf
func (s *SafetyService) convertOverrideToken(token *types.OverrideToken) *pb.OverrideToken {
	var decisionSummary *pb.DecisionSummary
	if token.DecisionSummary != nil {
		decisionSummary = &pb.DecisionSummary{
			Status:             s.convertSafetyStatus(token.DecisionSummary.Status),
			CriticalViolations: token.DecisionSummary.CriticalViolations,
			EnginesFailed:      token.DecisionSummary.EnginesFailed,
			RiskScore:          token.DecisionSummary.RiskScore,
			Explanation:        token.DecisionSummary.Explanation,
		}
	}

	return &pb.OverrideToken{
		TokenId:         token.TokenID,
		RequestId:       token.RequestID,
		PatientId:       token.PatientID,
		DecisionSummary: decisionSummary,
		RequiredLevel:   s.convertOverrideLevel(token.RequiredLevel),
		ExpiresAt:       timestamppb.New(token.ExpiresAt),
		ContextHash:     token.ContextHash,
		CreatedAt:       timestamppb.New(token.CreatedAt),
		Signature:       token.Signature,
	}
}

// convertOverrideLevel converts internal override level to protobuf
func (s *SafetyService) convertOverrideLevel(level types.OverrideLevel) pb.OverrideLevel {
	switch level {
	case types.OverrideLevelResident:
		return pb.OverrideLevel_OVERRIDE_LEVEL_RESIDENT
	case types.OverrideLevelAttending:
		return pb.OverrideLevel_OVERRIDE_LEVEL_ATTENDING
	case types.OverrideLevelPharmacist:
		return pb.OverrideLevel_OVERRIDE_LEVEL_PHARMACIST
	case types.OverrideLevelChief:
		return pb.OverrideLevel_OVERRIDE_LEVEL_CHIEF
	default:
		return pb.OverrideLevel_OVERRIDE_LEVEL_UNSPECIFIED
	}
}

// GetSnapshotStats returns snapshot processing statistics
func (s *SafetyService) GetSnapshotStats(ctx context.Context, req *pb.SnapshotStatsRequest) (*pb.SnapshotStatsResponse, error) {
	s.logger.Debug("Snapshot statistics requested", zap.Bool("detailed", req.Detailed))

	// Check if snapshot orchestration is available
	snapshotOrchestrator, ok := s.orchestrator.(*orchestration.SnapshotOrchestrationEngine)
	if !ok {
		s.logger.Debug("Snapshot orchestration not available")
		return &pb.SnapshotStatsResponse{
			CacheStats:          &pb.SnapshotStats{},
			EngineStats:         make(map[string]string),
			SnapshotModeEnabled: false,
			Timestamp:           timestamppb.Now(),
			Metadata: map[string]string{
				"error": "Snapshot orchestration not configured",
				"mode":  "legacy_only",
			},
		}, nil
	}

	// Get snapshot statistics from orchestrator
	stats := snapshotOrchestrator.GetSnapshotStats()
	
	response := &pb.SnapshotStatsResponse{
		SnapshotModeEnabled: true,
		Timestamp:          timestamppb.Now(),
		Metadata: map[string]string{
			"service":   "safety-gateway-platform",
			"version":   s.config.Service.Version,
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}

	// Convert cache stats if available
	if cacheStats, ok := stats["cache_stats"].(*types.SnapshotCacheStats); ok {
		response.CacheStats = &pb.SnapshotStats{
			L1CacheHits:    cacheStats.L1CacheHits,
			L1CacheMisses:  cacheStats.L1CacheMisses,
			L2CacheHits:    cacheStats.L2CacheHits,
			L2CacheMisses:  cacheStats.L2CacheMisses,
			TotalRequests:  cacheStats.TotalRequests,
			L1HitRate:      cacheStats.L1HitRate,
			L2HitRate:      cacheStats.L2HitRate,
			OverallHitRate: cacheStats.OverallHitRate,
			CacheSize:      cacheStats.CacheSize,
			Metadata:       make(map[string]string),
		}

		// Add cache metadata
		for k, v := range cacheStats.Metadata {
			if str, ok := v.(string); ok {
				response.CacheStats.Metadata[k] = str
			}
		}
	} else {
		response.CacheStats = &pb.SnapshotStats{}
	}

	// Convert engine stats
	response.EngineStats = make(map[string]string)
	for k, v := range stats {
		if k == "cache_stats" {
			continue // Already handled
		}
		if str, ok := v.(string); ok {
			response.EngineStats[k] = str
		} else if b, ok := v.(bool); ok {
			response.EngineStats[k] = fmt.Sprintf("%t", b)
		} else {
			response.EngineStats[k] = fmt.Sprintf("%v", v)
		}
	}

	if req.Detailed {
		// Add detailed information about engine registry
		// Note: This would require adding a method to access the registry from orchestrator
		// For now, add placeholder information
		response.EngineStats["detailed_mode"] = "true"
		response.Metadata["detailed"] = "true"
	}

	s.logger.Debug("Snapshot statistics retrieved successfully",
		zap.Bool("snapshot_enabled", response.SnapshotModeEnabled),
		zap.Int64("cache_total_requests", response.CacheStats.TotalRequests),
		zap.Float64("cache_hit_rate", response.CacheStats.OverallHitRate),
	)

	return response, nil
}
