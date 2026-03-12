// Package services implements the core business logic for the Context Gateway
package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"context-gateway-go/internal/models"
	"context-gateway-go/internal/storage"
	pb "context-gateway-go/proto"
)

// ContextGatewayService implements the gRPC Context Gateway service
type ContextGatewayService struct {
	pb.UnimplementedContextGatewayServer
	snapshotStore     *storage.SnapshotStore
	recipeService     *RecipeService
	dataSourceRegistry *DataSourceRegistry
	auditLogger       *AuditLogger
	metricsCollector  *MetricsCollector
}

// NewContextGatewayService creates a new Context Gateway service instance
func NewContextGatewayService(
	snapshotStore *storage.SnapshotStore,
	recipeService *RecipeService,
	dataSourceRegistry *DataSourceRegistry,
) *ContextGatewayService {
	return &ContextGatewayService{
		snapshotStore:       snapshotStore,
		recipeService:       recipeService,
		dataSourceRegistry:  dataSourceRegistry,
		auditLogger:         NewAuditLogger(),
		metricsCollector:    NewMetricsCollector(),
	}
}

// CreateSnapshot creates a new clinical snapshot using a workflow recipe
func (s *ContextGatewayService) CreateSnapshot(ctx context.Context, req *pb.CreateSnapshotRequest) (*pb.ClinicalSnapshot, error) {
	startTime := time.Now()
	log.Printf("Creating snapshot for patient %s using recipe %s", req.PatientId, req.Recipe.RecipeId)

	// Validate request
	if req.PatientId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "patient_id is required")
	}
	if req.Recipe == nil {
		return nil, status.Errorf(codes.InvalidArgument, "recipe is required")
	}

	// Convert protobuf recipe to internal model
	recipe, err := s.convertProtoToRecipe(req.Recipe)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid recipe: %v", err)
	}

	// Validate recipe
	valid, errors, warnings := recipe.Validate()
	if !valid {
		log.Printf("Recipe validation failed: %v", errors)
		return nil, status.Errorf(codes.InvalidArgument, "recipe validation failed: %v", errors)
	}
	if len(warnings) > 0 {
		log.Printf("Recipe validation warnings: %v", warnings)
	}

	// Start distributed transaction context
	transactionCtx, cancel := context.WithTimeout(ctx, time.Duration(recipe.SLAMs)*time.Millisecond)
	defer cancel()

	// Fetch clinical data using the recipe
	assembledData, err := s.assembleContextData(transactionCtx, recipe, req.PatientId, req.ProviderId, req.EncounterId, req.ForceRefresh)
	if err != nil {
		log.Printf("Failed to assemble context data: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to assemble clinical context: %v", err)
	}

	// Calculate expiration time
	expiresAt := startTime.Add(time.Duration(req.TtlHours) * time.Hour)

	// Create the clinical snapshot
	snapshot := models.NewClinicalSnapshot(req.PatientId, recipe.RecipeID, assembledData.Data)
	snapshot.RecipeID = recipe.RecipeID
	snapshot.CompletenessScore = assembledData.CompletenessScore
	snapshot.ExpiresAt = expiresAt
	snapshot.SignatureMethod = models.SignatureMethodMock // For development
	
	if req.ProviderId != nil {
		snapshot.ProviderID = req.ProviderId
	}
	if req.EncounterId != nil {
		snapshot.EncounterID = req.EncounterId
	}

	// Create assembly metadata
	snapshot.AssemblyMetadata = map[string]interface{}{
		"recipe_version":       recipe.Version,
		"assembly_duration_ms": time.Since(startTime).Milliseconds(),
		"sources_used":         len(assembledData.SourceMetadata),
		"force_refresh":        req.ForceRefresh,
		"ttl_hours":           req.TtlHours,
		"requesting_service":   req.RequestingService,
	}

	// Create evidence envelope for clinical safety
	snapshot.EvidenceEnvelope = map[string]interface{}{
		"recipe_used": map[string]interface{}{
			"recipe_id":         recipe.RecipeID,
			"version":           recipe.Version,
			"clinical_scenario": recipe.ClinicalScenario,
		},
		"assembly_evidence": map[string]interface{}{
			"sources_used":         len(assembledData.SourceMetadata),
			"assembly_duration_ms": time.Since(startTime).Milliseconds(),
			"completeness_score":   assembledData.CompletenessScore,
		},
		"integrity_evidence": map[string]interface{}{
			"checksum":         snapshot.Checksum,
			"signature_method": snapshot.SignatureMethod,
			"created_at":       startTime.Format(time.RFC3339),
		},
	}

	// Store the snapshot in dual-layer storage
	if err := s.snapshotStore.Save(transactionCtx, snapshot); err != nil {
		log.Printf("Failed to save snapshot: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to save snapshot: %v", err)
	}

	// Log audit event
	s.auditLogger.LogSnapshotCreated(snapshot.ID, req.PatientId, recipe.RecipeID, req.RequestingService)

	// Update metrics
	s.metricsCollector.RecordSnapshotCreated(recipe.RecipeID, time.Since(startTime))

	duration := time.Since(startTime).Milliseconds()
	log.Printf("Successfully created snapshot %s in %dms", snapshot.ID, duration)

	// Convert to protobuf and return
	return s.convertSnapshotToProto(snapshot), nil
}

// GetSnapshot retrieves a clinical snapshot by ID
func (s *ContextGatewayService) GetSnapshot(ctx context.Context, req *pb.GetSnapshotRequest) (*pb.ClinicalSnapshot, error) {
	if req.SnapshotId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "snapshot_id is required")
	}

	log.Printf("Retrieving snapshot %s for service %s", req.SnapshotId, req.RequestingService)

	snapshot, err := s.snapshotStore.Get(ctx, req.SnapshotId)
	if err != nil {
		log.Printf("Failed to retrieve snapshot: %v", err)
		return nil, status.Errorf(codes.Internal, "failed to retrieve snapshot: %v", err)
	}

	if snapshot == nil {
		return nil, status.Errorf(codes.NotFound, "snapshot not found")
	}

	// Mark as accessed and update
	snapshot.MarkAccessed()
	if err := s.snapshotStore.Update(ctx, snapshot); err != nil {
		log.Printf("Warning: Failed to update access count: %v", err)
	}

	// Log audit event
	s.auditLogger.LogSnapshotAccessed(snapshot.ID, snapshot.PatientID, req.RequestingService)

	return s.convertSnapshotToProto(snapshot), nil
}

// ValidateSnapshot validates the integrity and status of a snapshot
func (s *ContextGatewayService) ValidateSnapshot(ctx context.Context, req *pb.ValidateSnapshotRequest) (*pb.ValidateSnapshotResponse, error) {
	startTime := time.Now()

	if req.SnapshotId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "snapshot_id is required")
	}

	log.Printf("Validating snapshot %s", req.SnapshotId)

	snapshot, err := s.snapshotStore.Get(ctx, req.SnapshotId)
	if err != nil {
		return &pb.ValidateSnapshotResponse{
			Valid:                false,
			ChecksumValid:        false,
			SignatureValid:       false,
			NotExpired:           false,
			Errors:               []string{fmt.Sprintf("Failed to retrieve snapshot: %v", err)},
			ValidationDurationMs: float64(time.Since(startTime).Milliseconds()),
		}, nil
	}

	if snapshot == nil {
		return &pb.ValidateSnapshotResponse{
			Valid:                false,
			ChecksumValid:        false,
			SignatureValid:       false,
			NotExpired:           false,
			Errors:               []string{"Snapshot not found"},
			ValidationDurationMs: float64(time.Since(startTime).Milliseconds()),
		}, nil
	}

	// Perform validation
	isValid, validationErrors := snapshot.IsValid()
	
	response := &pb.ValidateSnapshotResponse{
		Valid:                isValid,
		ChecksumValid:        snapshot.VerifyChecksum(),
		SignatureValid:       true, // Mock signature always valid for development
		NotExpired:           !snapshot.IsExpired(),
		Errors:               validationErrors,
		ValidationDurationMs: float64(time.Since(startTime).Milliseconds()),
	}

	// Add warnings for low completeness or high access count
	var warnings []string
	if snapshot.CompletenessScore < 0.8 {
		warnings = append(warnings, fmt.Sprintf("Low completeness score: %.2f%%", snapshot.CompletenessScore*100))
	}
	if snapshot.AccessedCount > 100 {
		warnings = append(warnings, fmt.Sprintf("High access count: %d", snapshot.AccessedCount))
	}
	response.Warnings = warnings

	duration := time.Since(startTime).Milliseconds()
	log.Printf("Validated snapshot %s: valid=%v in %dms", req.SnapshotId, isValid, duration)

	return response, nil
}

// InvalidateSnapshot invalidates a clinical snapshot
func (s *ContextGatewayService) InvalidateSnapshot(ctx context.Context, req *pb.InvalidateRequest) (*pb.InvalidateResponse, error) {
	if req.SnapshotId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "snapshot_id is required")
	}

	log.Printf("Invalidating snapshot %s, reason: %s", req.SnapshotId, req.Reason)

	// Get the snapshot first
	snapshot, err := s.snapshotStore.Get(ctx, req.SnapshotId)
	if err != nil {
		return &pb.InvalidateResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to retrieve snapshot: %v", err),
		}, nil
	}

	if snapshot == nil {
		return &pb.InvalidateResponse{
			Success: false,
			Message: "Snapshot not found",
		}, nil
	}

	// Update snapshot status
	snapshot.Status = models.SnapshotStatusInvalidated
	
	if err := s.snapshotStore.Update(ctx, snapshot); err != nil {
		log.Printf("Failed to invalidate snapshot: %v", err)
		return &pb.InvalidateResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to invalidate snapshot: %v", err),
		}, nil
	}

	// Log audit event
	s.auditLogger.LogSnapshotInvalidated(snapshot.ID, snapshot.PatientID, req.Reason, req.RequestingService)

	now := timestamppb.New(time.Now().UTC())
	return &pb.InvalidateResponse{
		Success:       true,
		Message:       fmt.Sprintf("Snapshot %s invalidated successfully", req.SnapshotId),
		InvalidatedAt: now,
	}, nil
}

// FetchLiveFields performs live data fetching with governance controls
func (s *ContextGatewayService) FetchLiveFields(ctx context.Context, req *pb.LiveFetchRequest) (*pb.LiveFetchResponse, error) {
	if req.SnapshotId == "" {
		return nil, status.Errorf(codes.InvalidArgument, "snapshot_id is required")
	}
	if len(req.MissingFields) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "missing_fields is required")
	}

	log.Printf("Live fetching fields %v for snapshot %s", req.MissingFields, req.SnapshotId)

	// Get the snapshot to check permissions
	snapshot, err := s.snapshotStore.Get(ctx, req.SnapshotId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to retrieve snapshot: %v", err)
	}

	if snapshot == nil {
		return nil, status.Errorf(codes.NotFound, "snapshot not found")
	}

	// Validate live fetch permissions
	if !snapshot.AllowLiveFetch {
		return &pb.LiveFetchResponse{
			Errors: []string{"Live fetch not allowed for this snapshot"},
		}, nil
	}

	// Check field-level permissions
	var unauthorizedFields []string
	for _, field := range req.MissingFields {
		allowed := false
		for _, allowedField := range snapshot.AllowedLiveFields {
			if field == allowedField {
				allowed = true
				break
			}
		}
		if !allowed {
			unauthorizedFields = append(unauthorizedFields, field)
		}
	}

	if len(unauthorizedFields) > 0 {
		return &pb.LiveFetchResponse{
			Errors: []string{fmt.Sprintf("Unauthorized fields: %v", unauthorizedFields)},
		}, nil
	}

	// Fetch live data
	fetchedData, err := s.dataSourceRegistry.FetchFields(ctx, req.PatientId, req.MissingFields)
	if err != nil {
		log.Printf("Failed to fetch live fields: %v", err)
		return &pb.LiveFetchResponse{
			Errors: []string{fmt.Sprintf("Failed to fetch live data: %v", err)},
		}, nil
	}

	// Convert to protobuf struct
	protoData, err := structpb.NewStruct(fetchedData.Data)
	if err != nil {
		return &pb.LiveFetchResponse{
			Errors: []string{fmt.Sprintf("Failed to convert data: %v", err)},
		}, nil
	}

	// Create audit entry
	auditEntry := &pb.AuditEntry{
		Event:            "LIVE_FETCH",
		SnapshotId:       req.SnapshotId,
		PatientId:        req.PatientId,
		RequestingService: req.RequestingService,
		FieldsFetched:    req.MissingFields,
		Reason:           "missing_required_fields",
		Timestamp:        timestamppb.New(time.Now().UTC()),
	}

	// Log the live fetch event
	s.auditLogger.LogLiveFetch(req.SnapshotId, req.PatientId, req.MissingFields, req.RequestingService)

	// Update metrics
	s.metricsCollector.RecordLiveFetch(req.RequestingService, len(req.MissingFields))

	return &pb.LiveFetchResponse{
		FetchedData:  protoData,
		Completeness: fetchedData.CompletenessScore,
		AuditEntry:   auditEntry,
	}, nil
}

// GetServiceHealth returns the health status of the Context Gateway service
func (s *ContextGatewayService) GetServiceHealth(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	response := &pb.HealthResponse{
		Status:    pb.ServiceStatus_SERVICE_STATUS_HEALTHY,
		Version:   "1.0.0",
		Timestamp: timestamppb.New(time.Now().UTC()),
	}

	// Check dependencies if requested
	if req.IncludeDependencies {
		dependencies := s.checkDependencyHealth(ctx)
		response.Dependencies = dependencies
		
		// Update overall status based on dependencies
		for _, dep := range dependencies {
			if dep.Status == pb.ServiceStatus_SERVICE_STATUS_UNHEALTHY {
				response.Status = pb.ServiceStatus_SERVICE_STATUS_DEGRADED
			}
		}
	}

	// Get cache statistics
	stats, _ := s.snapshotStore.GetStats(ctx)
	if cacheStats := s.convertToCacheStats(stats); cacheStats != nil {
		response.CacheStats = cacheStats
	}

	return response, nil
}

// GetMetrics returns service metrics
func (s *ContextGatewayService) GetMetrics(ctx context.Context, req *pb.MetricsRequest) (*pb.MetricsResponse, error) {
	metrics := s.metricsCollector.GetMetrics()
	
	protoMetrics, err := structpb.NewStruct(metrics)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to convert metrics: %v", err)
	}

	return &pb.MetricsResponse{
		Metrics:   protoMetrics,
		Timestamp: timestamppb.New(time.Now().UTC()),
	}, nil
}

// Helper methods

// assembleContextData fetches and assembles clinical data using the recipe
func (s *ContextGatewayService) assembleContextData(
	ctx context.Context,
	recipe *models.WorkflowRecipe,
	patientID string,
	providerID, encounterID *string,
	forceRefresh bool,
) (*AssembledContext, error) {
	
	// This is a simplified implementation - in production, you'd implement:
	// 1. Parallel data fetching from multiple sources
	// 2. Circuit breaker patterns
	// 3. Retry logic
	// 4. Data quality validation
	// 5. Safety flag generation

	assembledData := make(map[string]interface{})
	sourceMetadata := make(map[string]interface{})
	
	// Mock assembled data for demonstration
	assembledData["patient_demographics"] = map[string]interface{}{
		"patient_id": patientID,
		"age":        35,
		"gender":     "M",
		"weight_kg":  75.0,
	}
	
	if providerID != nil {
		assembledData["provider_context"] = map[string]interface{}{
			"provider_id": *providerID,
		}
	}
	
	if encounterID != nil {
		assembledData["encounter_context"] = map[string]interface{}{
			"encounter_id": *encounterID,
		}
	}

	sourceMetadata["patient_service"] = map[string]interface{}{
		"source":           "patient_service",
		"response_time_ms": 45,
		"completeness":     0.95,
		"cache_hit":        !forceRefresh,
	}

	return &AssembledContext{
		Data:              assembledData,
		CompletenessScore: 0.95,
		SourceMetadata:    sourceMetadata,
		SafetyFlags:       []interface{}{}, // No safety flags in mock data
	}, nil
}

// convertProtoToRecipe converts protobuf recipe to internal model
func (s *ContextGatewayService) convertProtoToRecipe(protoRecipe *pb.WorkflowRecipe) (*models.WorkflowRecipe, error) {
	// This is a simplified conversion - in production, you'd need complete mapping
	recipe := &models.WorkflowRecipe{
		RecipeID:         protoRecipe.RecipeId,
		RecipeName:       protoRecipe.RecipeName,
		Version:          protoRecipe.Version,
		ClinicalScenario: protoRecipe.ClinicalScenario,
		WorkflowCategory: protoRecipe.WorkflowCategory,
		ExecutionPattern: protoRecipe.ExecutionPattern,
		SLAMs:           protoRecipe.SlaMs,
	}

	// Convert required fields
	for _, protoField := range protoRecipe.RequiredFields {
		field := models.DataPoint{
			Name:                 protoField.Name,
			SourceType:           models.DataSourceType(protoField.SourceType),
			Fields:               protoField.Fields,
			Required:             protoField.Required,
			MaxAgeHours:          protoField.MaxAgeHours,
			QualityThreshold:     protoField.QualityThreshold,
			TimeoutMs:            protoField.TimeoutMs,
			RetryCount:           protoField.RetryCount,
			FreshnessRequirement: protoField.FreshnessRequirement,
		}
		
		for _, fallback := range protoField.FallbackSources {
			field.FallbackSources = append(field.FallbackSources, models.DataSourceType(fallback))
		}
		
		recipe.RequiredFields = append(recipe.RequiredFields, field)
	}

	// Set governance metadata
	if protoRecipe.GovernanceMetadata != nil {
		recipe.GovernanceMetadata = models.GovernanceMetadata{
			ApprovedBy:              protoRecipe.GovernanceMetadata.ApprovedBy,
			ApprovalDate:            protoRecipe.GovernanceMetadata.ApprovalDate.AsTime(),
			Version:                 protoRecipe.GovernanceMetadata.Version,
			EffectiveDate:           protoRecipe.GovernanceMetadata.EffectiveDate.AsTime(),
			ClinicalBoardApprovalID: protoRecipe.GovernanceMetadata.ClinicalBoardApprovalId,
			Tags:                    protoRecipe.GovernanceMetadata.Tags,
			ChangeLog:               protoRecipe.GovernanceMetadata.ChangeLog,
		}
		
		if protoRecipe.GovernanceMetadata.ExpiryDate != nil {
			expiryDate := protoRecipe.GovernanceMetadata.ExpiryDate.AsTime()
			recipe.GovernanceMetadata.ExpiryDate = &expiryDate
		}
	}

	return recipe, nil
}

// convertSnapshotToProto converts internal snapshot to protobuf
func (s *ContextGatewayService) convertSnapshotToProto(snapshot *models.ClinicalSnapshot) *pb.ClinicalSnapshot {
	protoData, _ := structpb.NewStruct(snapshot.Data)
	assemblyMetadata, _ := structpb.NewStruct(snapshot.AssemblyMetadata)
	evidenceEnvelope, _ := structpb.NewStruct(snapshot.EvidenceEnvelope)

	protoSnapshot := &pb.ClinicalSnapshot{
		Id:                snapshot.ID,
		PatientId:         snapshot.PatientID,
		RecipeId:          snapshot.RecipeID,
		ContextId:         snapshot.ContextID,
		Data:              protoData,
		CompletenessScore: snapshot.CompletenessScore,
		Checksum:          snapshot.Checksum,
		Signature:         snapshot.Signature,
		SignatureMethod:   pb.SignatureMethod(snapshot.SignatureMethod),
		CreatedAt:         timestamppb.New(snapshot.CreatedAt),
		ExpiresAt:         timestamppb.New(snapshot.ExpiresAt),
		Status:            pb.SnapshotStatus(snapshot.Status),
		AssemblyMetadata:  assemblyMetadata,
		EvidenceEnvelope:  evidenceEnvelope,
		AccessedCount:     snapshot.AccessedCount,
		AllowLiveFetch:    snapshot.AllowLiveFetch,
		AllowedLiveFields: snapshot.AllowedLiveFields,
	}

	if snapshot.ProviderID != nil {
		protoSnapshot.ProviderId = snapshot.ProviderID
	}
	if snapshot.EncounterID != nil {
		protoSnapshot.EncounterId = snapshot.EncounterID
	}
	if snapshot.LastAccessedAt != nil {
		protoSnapshot.LastAccessedAt = timestamppb.New(*snapshot.LastAccessedAt)
	}

	return protoSnapshot
}

// checkDependencyHealth checks the health of service dependencies
func (s *ContextGatewayService) checkDependencyHealth(ctx context.Context) []*pb.DependencyHealth {
	// Mock dependency health checks
	return []*pb.DependencyHealth{
		{
			ServiceName:    "redis",
			Status:         pb.ServiceStatus_SERVICE_STATUS_HEALTHY,
			Endpoint:       "localhost:6379",
			ResponseTimeMs: 2,
			LastCheck:      timestamppb.New(time.Now().UTC()),
		},
		{
			ServiceName:    "mongodb",
			Status:         pb.ServiceStatus_SERVICE_STATUS_HEALTHY,
			Endpoint:       "localhost:27017",
			ResponseTimeMs: 5,
			LastCheck:      timestamppb.New(time.Now().UTC()),
		},
	}
}

// convertToCacheStats converts storage stats to protobuf format
func (s *ContextGatewayService) convertToCacheStats(stats map[string]interface{}) *pb.CacheStats {
	if stats == nil {
		return nil
	}

	return &pb.CacheStats{
		TotalEntries: 1000, // Mock values - implement actual stats
		HitRatio:     0.85,
		L1Entries:    250,
		L2Entries:    750,
		L3Entries:    0,
		LastUpdated:  timestamppb.New(time.Now().UTC()),
	}
}

// AssembledContext represents the result of clinical data assembly
type AssembledContext struct {
	Data              map[string]interface{}
	CompletenessScore float64
	SourceMetadata    map[string]interface{}
	SafetyFlags       []interface{}
}