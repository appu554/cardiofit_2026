package grpc

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "kb-drug-interactions/api/pb"
	"kb-drug-interactions/internal/config"
	"kb-drug-interactions/internal/models"
	"kb-drug-interactions/internal/services"
)

// DrugInteractionGRPCServer implements the gRPC service for drug interactions
type DrugInteractionGRPCServer struct {
	pb.UnimplementedDrugInteractionServiceServer
	
	interactionService *services.InteractionService
	enhancedMatrix     *services.EnhancedInteractionMatrixService
	config             *config.Config
}

// NewDrugInteractionGRPCServer creates a new gRPC server instance
func NewDrugInteractionGRPCServer(
	interactionService *services.InteractionService,
	enhancedMatrix *services.EnhancedInteractionMatrixService,
	config *config.Config,
) *DrugInteractionGRPCServer {
	return &DrugInteractionGRPCServer{
		interactionService: interactionService,
		enhancedMatrix:     enhancedMatrix,
		config:             config,
	}
}

// CheckInteractions implements the primary interaction checking endpoint
func (s *DrugInteractionGRPCServer) CheckInteractions(
	ctx context.Context, 
	req *pb.InteractionCheckRequest,
) (*pb.InteractionCheckResponse, error) {
	// Validate request
	if len(req.DrugCodes) < 2 {
		return nil, status.Errorf(codes.InvalidArgument, "minimum 2 drug codes required")
	}

	if len(req.DrugCodes) > s.config.MaxInteractionsPerRequest {
		return nil, status.Errorf(codes.InvalidArgument, 
			"too many drugs: maximum %d allowed", s.config.MaxInteractionsPerRequest)
	}

	// Convert from protobuf to internal request model
	internalRequest := &models.EnhancedInteractionCheckRequest{
		TransactionID:       req.TransactionId,
		DrugCodes:           req.DrugCodes,
		DatasetVersion:      req.DatasetVersion,
		ExpandClasses:       req.ExpandClasses,
		IncludeContextuals:  req.IncludeContextuals,
		IncludeAlternatives: req.IncludeAlternatives,
		IncludeMonitoring:   req.IncludeMonitoring,
		SeverityFilter:      req.SeverityFilter,
	}

	// Convert patient context
	if req.PatientContext != nil {
		internalRequest.PatientContext = &models.PatientContextData{
			PGX:           req.PatientContext.Pgx,
			HepaticStage:  req.PatientContext.HepaticStage,
			RenalStage:    req.PatientContext.RenalStage,
			AgeBand:       req.PatientContext.AgeBand,
			Comorbidities: req.PatientContext.Comorbidities,
			Allergies:     req.PatientContext.Allergies,
		}
	}

	// Convert clinical context
	if len(req.ClinicalContext) > 0 {
		internalRequest.ClinicalContext = make(map[string]interface{})
		for k, v := range req.ClinicalContext {
			internalRequest.ClinicalContext[k] = v
		}
	}

	// Perform interaction checking using enhanced matrix
	response, err := s.enhancedMatrix.CheckInteractionsEnhanced(ctx, internalRequest)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "interaction check failed: %v", err)
	}

	// Convert to protobuf response
	pbResponse := &pb.InteractionCheckResponse{
		TransactionId:  response.TransactionID,
		DatasetVersion: response.DatasetVersion,
		CheckTimestamp: timestamppb.New(response.CheckTimestamp),
		CacheHit:       response.CacheHit,
		RiskScore:      response.RiskScore.InexactFloat64(),
	}

	// Convert interactions
	pbResponse.Interactions = make([]*pb.InteractionDetail, len(response.Interactions))
	for i, interaction := range response.Interactions {
		pbInteraction := &pb.InteractionDetail{
			Drug1Code:              interaction.Drug1.Code,
			Drug2Code:              interaction.Drug2.Code,
			Severity:               string(interaction.Severity),
			Mechanism:              string(interaction.Mechanism),
			ClinicalEffects:        interaction.ClinicalEffects,
			ManagementStrategy:     interaction.ManagementStrategy,
			Evidence:               string(interaction.Evidence),
			PgxApplicable:          interaction.PGXApplicable,
			DoseAdjustmentRequired: interaction.DoseAdjustmentRequired,
			InteractionId:          interaction.InteractionID,
		}

		if interaction.Confidence != nil {
			conf, _ := interaction.Confidence.Float64()
			pbInteraction.Confidence = conf
		}

		if len(interaction.Qualifiers) > 0 {
			pbInteraction.Qualifiers = interaction.Qualifiers
		}

		if len(interaction.Sources) > 0 {
			pbInteraction.Sources = interaction.Sources
		}

		if interaction.TimeToOnset != nil {
			pbInteraction.TimeToOnset = *interaction.TimeToOnset
		}

		if interaction.Duration != nil {
			pbInteraction.Duration = *interaction.Duration
		}

		if len(interaction.MonitoringParameters) > 0 {
			pbInteraction.MonitoringParameters = make(map[string]string)
			for k, v := range interaction.MonitoringParameters {
				if vStr, ok := v.(string); ok {
					pbInteraction.MonitoringParameters[k] = vStr
				}
			}
		}

		if len(interaction.AlternativeDrugs) > 0 {
			pbInteraction.AlternativeDrugs = interaction.AlternativeDrugs
		}

		pbResponse.Interactions[i] = pbInteraction
	}

	// Convert conflict trail
	if response.ConflictTrail != nil {
		pbResponse.ConflictTrail = &pb.ConflictTrail{
			SynthesizedFromVersion: response.ConflictTrail.SynthesizedFromVersion,
			VendorEvidenceIds:      response.ConflictTrail.VendorEvidenceIDs,
			OverridesApplied:       response.ConflictTrail.OverridesApplied,
			HarmonizedAt:           timestamppb.New(response.ConflictTrail.HarmonizedAt),
			HarmonizerVersion:      response.ConflictTrail.HarmonizerVersion,
		}
	}

	// Convert summary
	pbResponse.Summary = &pb.InteractionSummary{
		TotalInteractions:    int32(response.Summary.TotalInteractions),
		HighestSeverity:      response.Summary.HighestSeverity,
		ContraindicatedPairs: int32(response.Summary.ContraindicatedPairs),
		RequiredActions:      response.Summary.RequiredActions,
		PgxInteractions:      int32(response.Summary.PGXInteractions),
		ClassInteractions:    int32(response.Summary.ClassInteractions),
		ModifierInteractions: int32(response.Summary.ModifierInteractions),
	}

	// Convert severity counts
	pbResponse.Summary.SeverityCounts = make(map[string]int32)
	for severity, count := range response.Summary.SeverityCounts {
		pbResponse.Summary.SeverityCounts[severity] = int32(count)
	}

	// Convert recommendations
	pbResponse.Recommendations = response.Recommendations

	// Convert alternative drugs
	if len(response.AlternativeDrugs) > 0 {
		pbResponse.AlternativeDrugs = make(map[string]*pb.DrugAlternatives)
		for drugCode, alternatives := range response.AlternativeDrugs {
			pbAlts := &pb.DrugAlternatives{
				Rationale: alternatives.Rationale,
			}
			
			pbAlts.Alternatives = make([]*pb.AlternativeDrug, len(alternatives.Alternatives))
			for i, alt := range alternatives.Alternatives {
				pbAlt := &pb.AlternativeDrug{
					DrugInfo: &pb.DrugInfo{
						Code:      alt.DrugInfo.Code,
						Name:      alt.DrugInfo.Name,
						BrandName: alt.DrugInfo.Name, // Brand not available, use Name
						Strength:  alt.DrugInfo.Strength,
						Route:     alt.DrugInfo.Route,
						Form:      alt.DrugInfo.Generic, // Map to form field
					},
					Reason:             alt.Reason,
					RequiresMonitoring: alt.RequiresMonitoring,
				}
				
				if alt.SafetyScore != nil {
					score, _ := alt.SafetyScore.Float64()
					pbAlt.SafetyScore = score
				}
				
				pbAlts.Alternatives[i] = pbAlt
			}
			
			pbResponse.AlternativeDrugs[drugCode] = pbAlts
		}
	}

	// Convert monitoring plan
	if len(response.MonitoringPlan) > 0 {
		pbResponse.MonitoringPlan = make([]*pb.MonitoringRecommendation, len(response.MonitoringPlan))
		for i, monitor := range response.MonitoringPlan {
			pbMonitor := &pb.MonitoringRecommendation{
				Parameter:      monitor.Parameter,
				Frequency:      monitor.Frequency,
				Duration:       monitor.Duration,
				TargetRange:    monitor.TargetRange,
				CriticalValues: monitor.CriticalValues,
				Instructions:   monitor.Instructions,
			}
			
			pbResponse.MonitoringPlan[i] = pbMonitor
		}
	}

	return pbResponse, nil
}

// BatchCheckInteractions handles batch interaction checking
func (s *DrugInteractionGRPCServer) BatchCheckInteractions(
	ctx context.Context, 
	req *pb.BatchInteractionCheckRequest,
) (*pb.BatchInteractionCheckResponse, error) {
	if len(req.Requests) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "no requests provided")
	}

	if len(req.Requests) > s.config.MaxBatchSize {
		return nil, status.Errorf(codes.InvalidArgument, 
			"batch size %d exceeds maximum %d", len(req.Requests), s.config.MaxBatchSize)
	}

	startTime := time.Now()
	
	// Convert requests
	internalRequests := make([]models.EnhancedInteractionCheckRequest, len(req.Requests))
	for i, pbReq := range req.Requests {
		internalReq := models.EnhancedInteractionCheckRequest{
			TransactionID:       pbReq.TransactionId,
			DrugCodes:           pbReq.DrugCodes,
			DatasetVersion:      pbReq.DatasetVersion,
			ExpandClasses:       pbReq.ExpandClasses,
			IncludeContextuals:  pbReq.IncludeContextuals,
			IncludeAlternatives: pbReq.IncludeAlternatives,
			IncludeMonitoring:   pbReq.IncludeMonitoring,
			SeverityFilter:      pbReq.SeverityFilter,
		}

		if pbReq.PatientContext != nil {
			internalReq.PatientContext = &models.PatientContextData{
				PGX:           pbReq.PatientContext.Pgx,
				HepaticStage:  pbReq.PatientContext.HepaticStage,
				RenalStage:    pbReq.PatientContext.RenalStage,
				AgeBand:       pbReq.PatientContext.AgeBand,
				Comorbidities: pbReq.PatientContext.Comorbidities,
				Allergies:     pbReq.PatientContext.Allergies,
			}
		}

		internalRequests[i] = internalReq
	}

	// Process batch (with parallel processing if enabled)
	var responses []models.EnhancedInteractionCheckResponse
	var successfulChecks, failedChecks int32

	if req.Options != nil && req.Options.Parallel {
		responses, successfulChecks, failedChecks = s.processBatchParallel(ctx, internalRequests, req.Options)
	} else {
		responses, successfulChecks, failedChecks = s.processBatchSequential(ctx, internalRequests)
	}

	// Build statistics
	processingTime := time.Since(startTime)
	stats := &pb.BatchProcessingStatistics{
		StartedAt:               timestamppb.New(startTime),
		CompletedAt:             timestamppb.New(time.Now()),
		TotalRequests:           int32(len(req.Requests)),
		SuccessfulRequests:      successfulChecks,
		FailedRequests:          failedChecks,
		AverageProcessingTimeMs: float64(processingTime.Nanoseconds()) / 1e6 / float64(len(req.Requests)),
	}

	// Convert responses to protobuf
	pbResponses := make([]*pb.InteractionCheckResponse, len(responses))
	totalInteractions := int32(0)
	
	for i, response := range responses {
		pbResp, err := s.convertToProtobufResponse(&response)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert response %d: %v", i, err)
		}
		pbResponses[i] = pbResp
		totalInteractions += int32(len(response.Interactions))
	}

	stats.TotalInteractionsFound = totalInteractions

	return &pb.BatchInteractionCheckResponse{
		Responses:       pbResponses,
		Statistics:      stats,
		SuccessfulChecks: successfulChecks,
		FailedChecks:    failedChecks,
		ProcessedAt:     timestamppb.New(time.Now()),
	}, nil
}

// FastLookup performs ultra-fast pairwise interaction lookup
func (s *DrugInteractionGRPCServer) FastLookup(
	ctx context.Context, 
	req *pb.FastLookupRequest,
) (*pb.FastLookupResponse, error) {
	if req.DrugACode == "" || req.DrugBCode == "" {
		return nil, status.Errorf(codes.InvalidArgument, "both drug codes are required")
	}

	result, found, lookupTime := s.enhancedMatrix.FastLookup(req.DrugACode, req.DrugBCode)
	
	response := &pb.FastLookupResponse{
		InteractionFound: found,
		LookupTimeMs:     float64(lookupTime.Nanoseconds()) / 1e6,
		CacheHit:         lookupTime < time.Millisecond, // Heuristic for cache hit
	}

	if found && result != nil {
		pbInteraction, err := s.convertInteractionToProtobuf(result)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert interaction: %v", err)
		}
		response.Interaction = pbInteraction
	}

	return response, nil
}

// HealthCheck provides service health information
func (s *DrugInteractionGRPCServer) HealthCheck(
	ctx context.Context, 
	req *pb.HealthCheckRequest,
) (*pb.HealthCheckResponse, error) {
	// Check component health
	componentHealth := make(map[string]string)
	overallStatus := "healthy"

	// Check database
	if err := s.interactionService.GetDatabase().HealthCheck(); err != nil {
		componentHealth["database"] = "unhealthy"
		overallStatus = "degraded"
	} else {
		componentHealth["database"] = "healthy"
	}

	// Check cache
	if err := s.interactionService.GetCache().HealthCheck(); err != nil {
		componentHealth["cache"] = "unhealthy"
		if overallStatus == "healthy" {
			overallStatus = "degraded"
		}
	} else {
		componentHealth["cache"] = "healthy"
	}

	// Check interaction matrix
	stats := s.enhancedMatrix.GetMatrixStatistics()
	if stats.TotalInteractions == 0 {
		componentHealth["matrix"] = "empty"
		overallStatus = "degraded"
	} else {
		componentHealth["matrix"] = "healthy"
	}

	response := &pb.HealthCheckResponse{
		Status:           overallStatus,
		ComponentHealth:  componentHealth,
		Version:          "1.0.0",
		Timestamp:        timestamppb.New(time.Now()),
		DatasetVersion:   stats.CurrentDatasetVersion,
		TotalInteractions: int32(stats.TotalInteractions),
	}

	return response, nil
}

// GetMatrixStatistics provides detailed matrix performance statistics
func (s *DrugInteractionGRPCServer) GetMatrixStatistics(
	ctx context.Context, 
	req *pb.MatrixStatisticsRequest,
) (*pb.MatrixStatisticsResponse, error) {
	stats := s.enhancedMatrix.GetMatrixStatistics()
	
	response := &pb.MatrixStatisticsResponse{
		TotalDrugs:            int32(stats.TotalDrugs),
		TotalInteractions:     int32(stats.TotalInteractions),
		MatrixDensity:         stats.MatrixDensity,
		LastUpdated:           timestamppb.New(stats.LastUpdated),
		MemoryUsageMb:         stats.MemoryUsageMB,
		CurrentDatasetVersion: stats.CurrentDatasetVersion,
	}

	// Convert performance metrics
	response.LookupPerformance = &pb.PerformanceMetrics{
		AverageLookupTimeNs: stats.LookupPerformance.AverageLookupTimeNs,
		CacheHitRate:        stats.LookupPerformance.CacheHitRate,
		TotalLookups:        stats.LookupPerformance.TotalLookups,
		BatchProcessingRate: stats.LookupPerformance.BatchProcessingRate,
	}

	// Convert cache statistics
	response.CacheStatistics = &pb.CacheStatistics{
		HotCacheHitRate:    stats.CacheStatistics.HotCacheHitRate,
		WarmCacheHitRate:   stats.CacheStatistics.WarmCacheHitRate,
		HotCacheEntries:    stats.CacheStatistics.HotCacheEntries,
		WarmCacheEntries:   stats.CacheStatistics.WarmCacheEntries,
		HotCacheMemoryMb:   stats.CacheStatistics.HotCacheMemoryMB,
		EvictionRate:       stats.CacheStatistics.EvictionRate,
		LastRefresh:        timestamppb.New(stats.CacheStatistics.LastRefresh),
	}

	return response, nil
}

// GetStatistics provides overall service statistics
func (s *DrugInteractionGRPCServer) GetStatistics(
	ctx context.Context,
	req *pb.GetStatisticsRequest,
) (*pb.GetStatisticsResponse, error) {
	// Get matrix statistics as the base
	matrixStats := s.enhancedMatrix.GetMatrixStatistics()

	response := &pb.GetStatisticsResponse{
		TotalInteractionsInDb: int64(matrixStats.TotalInteractions),
		UniqueInteractions:    int64(matrixStats.TotalInteractions), // Same as total for now
		ActiveRules:           int64(matrixStats.TotalDrugs),        // Approximate
		LastDatasetUpdate:     timestamppb.New(matrixStats.LastUpdated),
		MemoryUsageMb:         matrixStats.MemoryUsageMB,
		CurrentDatasetVersion: matrixStats.CurrentDatasetVersion,
	}

	// Convert performance metrics
	response.LookupPerformance = &pb.PerformanceMetrics{
		AverageLookupTimeNs: matrixStats.LookupPerformance.AverageLookupTimeNs,
		CacheHitRate:        matrixStats.LookupPerformance.CacheHitRate,
		TotalLookups:        matrixStats.LookupPerformance.TotalLookups,
		BatchProcessingRate: matrixStats.LookupPerformance.BatchProcessingRate,
	}

	// Convert cache statistics
	response.CacheStatistics = &pb.CacheStatistics{
		HotCacheHitRate:  matrixStats.CacheStatistics.HotCacheHitRate,
		WarmCacheHitRate: matrixStats.CacheStatistics.WarmCacheHitRate,
		HotCacheEntries:  matrixStats.CacheStatistics.HotCacheEntries,
		WarmCacheEntries: matrixStats.CacheStatistics.WarmCacheEntries,
		HotCacheMemoryMb: matrixStats.CacheStatistics.HotCacheMemoryMB,
		EvictionRate:     matrixStats.CacheStatistics.EvictionRate,
		LastRefresh:      timestamppb.New(matrixStats.CacheStatistics.LastRefresh),
	}

	return response, nil
}

// Helper methods for batch processing

func (s *DrugInteractionGRPCServer) processBatchParallel(
	ctx context.Context,
	requests []models.EnhancedInteractionCheckRequest,
	options *pb.BatchCheckOptions,
) ([]models.EnhancedInteractionCheckResponse, int32, int32) {
	maxConcurrency := int(options.MaxConcurrency)
	if maxConcurrency <= 0 || maxConcurrency > 20 {
		maxConcurrency = 10 // Default safe concurrency
	}

	responses := make([]models.EnhancedInteractionCheckResponse, len(requests))
	var successfulChecks, failedChecks int32

	// Use worker pool for controlled concurrency
	type job struct {
		index   int
		request models.EnhancedInteractionCheckRequest
	}

	type result struct {
		index    int
		response models.EnhancedInteractionCheckResponse
		error    error
	}

	jobs := make(chan job, len(requests))
	results := make(chan result, len(requests))

	// Start workers
	for w := 0; w < maxConcurrency; w++ {
		go func() {
			for j := range jobs {
				response, err := s.enhancedMatrix.CheckInteractionsEnhanced(ctx, &j.request)
				if err != nil {
					results <- result{index: j.index, error: err}
				} else {
					results <- result{index: j.index, response: *response}
				}
			}
		}()
	}

	// Send jobs
	for i, req := range requests {
		jobs <- job{index: i, request: req}
	}
	close(jobs)

	// Collect results
	for i := 0; i < len(requests); i++ {
		res := <-results
		if res.error != nil {
			failedChecks++
			// Create empty response for failed request
			responses[res.index] = models.EnhancedInteractionCheckResponse{
				TransactionID:  requests[res.index].TransactionID,
				Interactions:   []models.EnhancedInteractionResult{},
				CheckTimestamp: time.Now(),
			}
		} else {
			successfulChecks++
			responses[res.index] = res.response
		}

		// Fail fast if enabled
		if options.FailFast && res.error != nil {
			break
		}
	}

	return responses, successfulChecks, failedChecks
}

func (s *DrugInteractionGRPCServer) processBatchSequential(
	ctx context.Context,
	requests []models.EnhancedInteractionCheckRequest,
) ([]models.EnhancedInteractionCheckResponse, int32, int32) {
	responses := make([]models.EnhancedInteractionCheckResponse, len(requests))
	var successfulChecks, failedChecks int32

	for i, req := range requests {
		response, err := s.enhancedMatrix.CheckInteractionsEnhanced(ctx, &req)
		if err != nil {
			failedChecks++
			responses[i] = models.EnhancedInteractionCheckResponse{
				TransactionID:  req.TransactionID,
				Interactions:   []models.EnhancedInteractionResult{},
				CheckTimestamp: time.Now(),
			}
		} else {
			successfulChecks++
			responses[i] = *response
		}
	}

	return responses, successfulChecks, failedChecks
}

// Helper conversion methods

func (s *DrugInteractionGRPCServer) convertToProtobufResponse(
	response *models.EnhancedInteractionCheckResponse,
) (*pb.InteractionCheckResponse, error) {
	pbResponse := &pb.InteractionCheckResponse{
		TransactionId:  response.TransactionID,
		DatasetVersion: response.DatasetVersion,
		CheckTimestamp: timestamppb.New(response.CheckTimestamp),
		CacheHit:       response.CacheHit,
		RiskScore:      response.RiskScore.InexactFloat64(),
		Recommendations: response.Recommendations,
	}

	// Convert interactions
	pbResponse.Interactions = make([]*pb.InteractionDetail, len(response.Interactions))
	for i, interaction := range response.Interactions {
		pbInteraction, err := s.convertInteractionToProtobuf(&interaction)
		if err != nil {
			return nil, err
		}
		pbResponse.Interactions[i] = pbInteraction
	}

	// Convert summary
	pbResponse.Summary = &pb.InteractionSummary{
		TotalInteractions:    int32(response.Summary.TotalInteractions),
		HighestSeverity:      response.Summary.HighestSeverity,
		ContraindicatedPairs: int32(response.Summary.ContraindicatedPairs),
		RequiredActions:      response.Summary.RequiredActions,
		PgxInteractions:      int32(response.Summary.PGXInteractions),
		ClassInteractions:    int32(response.Summary.ClassInteractions),
		ModifierInteractions: int32(response.Summary.ModifierInteractions),
		SeverityCounts:       make(map[string]int32),
	}

	for severity, count := range response.Summary.SeverityCounts {
		pbResponse.Summary.SeverityCounts[severity] = int32(count)
	}

	return pbResponse, nil
}

func (s *DrugInteractionGRPCServer) convertInteractionToProtobuf(
	interaction *models.EnhancedInteractionResult,
) (*pb.InteractionDetail, error) {
	pbInteraction := &pb.InteractionDetail{
		Drug1Code:              interaction.Drug1.Code,
		Drug2Code:              interaction.Drug2.Code,
		Severity:               string(interaction.Severity),
		Mechanism:              string(interaction.Mechanism),
		ClinicalEffects:        interaction.ClinicalEffects,
		ManagementStrategy:     interaction.ManagementStrategy,
		Evidence:               string(interaction.Evidence),
		PgxApplicable:          interaction.PGXApplicable,
		DoseAdjustmentRequired: interaction.DoseAdjustmentRequired,
		InteractionId:          interaction.InteractionID,
	}

	if interaction.Confidence != nil {
		conf, _ := interaction.Confidence.Float64()
		pbInteraction.Confidence = conf
	}

	if len(interaction.Qualifiers) > 0 {
		pbInteraction.Qualifiers = interaction.Qualifiers
	}

	if len(interaction.Sources) > 0 {
		pbInteraction.Sources = interaction.Sources
	}

	if interaction.TimeToOnset != nil {
		pbInteraction.TimeToOnset = *interaction.TimeToOnset
	}

	if interaction.Duration != nil {
		pbInteraction.Duration = *interaction.Duration
	}

	if len(interaction.MonitoringParameters) > 0 {
		pbInteraction.MonitoringParameters = make(map[string]string)
		for k, v := range interaction.MonitoringParameters {
			if vStr, ok := v.(string); ok {
				pbInteraction.MonitoringParameters[k] = vStr
			}
		}
	}

	if len(interaction.AlternativeDrugs) > 0 {
		pbInteraction.AlternativeDrugs = interaction.AlternativeDrugs
	}

	return pbInteraction, nil
}

// StartGRPCServer starts the gRPC server on specified port
func StartGRPCServer(
	cfg *config.Config,
	interactionService *services.InteractionService,
	enhancedMatrix *services.EnhancedInteractionMatrixService,
) error {
	grpcPort := cfg.Server.GRPCPort
	if grpcPort == "" {
		grpcPort = "8086" // Default gRPC port (HTTP on 8085)
	}

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", grpcPort, err)
	}

	// Create gRPC server instance first
	server := NewDrugInteractionGRPCServer(interactionService, enhancedMatrix, cfg)

	// Create gRPC server with interceptors
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(server.unaryInterceptor),
	)

	// Register service
	pb.RegisterDrugInteractionServiceServer(grpcServer, server)

	fmt.Printf("gRPC server starting on port %s\n", grpcPort)

	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve gRPC: %w", err)
	}

	return nil
}

// unaryInterceptor provides logging and metrics for gRPC calls
func (s *DrugInteractionGRPCServer) unaryInterceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	startTime := time.Now()
	
	// Call the handler
	resp, err := handler(ctx, req)
	
	// Record metrics
	duration := time.Since(startTime)
	method := extractMethodName(info.FullMethod)
	
	if err != nil {
		s.interactionService.GetMetrics().RecordGRPCRequest(method, "error", duration)
	} else {
		s.interactionService.GetMetrics().RecordGRPCRequest(method, "success", duration)
	}

	return resp, err
}

func extractMethodName(fullMethod string) string {
	// Extract method name from full gRPC method path
	// e.g., "/kb5.v1.DrugInteractionService/CheckInteractions" -> "CheckInteractions"
	parts := strings.Split(fullMethod, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "unknown"
}