package orchestration

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/internal/cache"
	"safety-gateway-platform/internal/clients"
	"safety-gateway-platform/internal/config"
	"safety-gateway-platform/internal/registry"
	"safety-gateway-platform/internal/snapshot"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// SnapshotOrchestrationEngine extends the base orchestration engine with snapshot capabilities
type SnapshotOrchestrationEngine struct {
	*OrchestrationEngine
	snapshotValidator  *snapshot.Validator
	contextClient      *clients.ContextGatewayClient
	snapshotCache      *cache.SnapshotCache
	config             *config.SnapshotConfig
	logger             *logger.Logger
}

// NewSnapshotOrchestrationEngine creates a new snapshot-aware orchestration engine
func NewSnapshotOrchestrationEngine(
	baseEngine *OrchestrationEngine,
	snapshotValidator *snapshot.Validator,
	contextClient *clients.ContextGatewayClient,
	snapshotCache *cache.SnapshotCache,
	cfg *config.SnapshotConfig,
	logger *logger.Logger,
) *SnapshotOrchestrationEngine {
	return &SnapshotOrchestrationEngine{
		OrchestrationEngine: baseEngine,
		snapshotValidator:   snapshotValidator,
		contextClient:       contextClient,
		snapshotCache:       snapshotCache,
		config:              cfg,
		logger:              logger,
	}
}

// ProcessSafetyRequestWithSnapshot processes a safety request using snapshot-based data
func (o *SnapshotOrchestrationEngine) ProcessSafetyRequestWithSnapshot(
	ctx context.Context,
	req *types.SafetyRequest,
) (*types.SafetyResponse, error) {
	startTime := time.Now()
	requestLogger := o.logger.WithRequestID(req.RequestID).WithPatientID(req.PatientID)

	requestLogger.Info("Processing safety request with snapshot",
		zap.String("action_type", req.ActionType),
		zap.String("priority", req.Priority),
		zap.Int("medication_count", len(req.MedicationIDs)),
	)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, o.config.RequestTimeout)
	defer cancel()

	// 1. Extract snapshot reference from request
	snapshotRef := o.extractSnapshotReference(req)
	if snapshotRef == nil {
		return o.createErrorResponse(req, fmt.Errorf("snapshot reference required for snapshot-based processing"), startTime), nil
	}

	// 2. Validate snapshot reference
	if err := o.snapshotValidator.ValidateReference(snapshotRef); err != nil {
		requestLogger.Error("Snapshot reference validation failed", zap.Error(err))
		return o.createErrorResponse(req, fmt.Errorf("snapshot reference validation failed: %w", err), startTime), nil
	}

	// 3. Retrieve and validate snapshot
	snapshot, err := o.getValidatedSnapshot(ctx, snapshotRef.SnapshotID, requestLogger)
	if err != nil {
		requestLogger.Error("Snapshot retrieval failed", zap.Error(err))
		return o.createErrorResponse(req, fmt.Errorf("snapshot retrieval failed: %w", err), startTime), nil
	}

	// 4. Get applicable engines for the request
	engines := o.registry.GetEnginesForRequest(req)
	if len(engines) == 0 {
		requestLogger.Warn("No engines available for request")
		return o.createErrorResponse(req, fmt.Errorf("no engines available"), startTime), nil
	}

	requestLogger.Debug("Selected engines for snapshot-based execution",
		zap.Int("engine_count", len(engines)),
		zap.Strings("engines", o.getEngineIDs(engines)),
		zap.String("snapshot_id", snapshot.SnapshotID),
		zap.Float64("data_completeness", snapshot.DataCompleteness),
	)

	// 5. Execute engines with snapshot data
	engineCtx, engineCancel := context.WithTimeout(ctx, o.config.EngineExecutionTimeout)
	defer engineCancel()

	results := o.executeEnginesWithSnapshot(engineCtx, engines, req, snapshot, requestLogger)

	// 6. Aggregate results with snapshot reference
	response := o.aggregateWithSnapshot(req, results, snapshot)
	response.ProcessingTime = time.Since(startTime)

	// Log final decision
	requestLogger.Info("Snapshot-based safety request processed",
		zap.String("status", string(response.Status)),
		zap.Float64("risk_score", response.RiskScore),
		zap.Int64("processing_time_ms", response.ProcessingTime.Milliseconds()),
		zap.Int("engines_executed", len(results)),
		zap.String("snapshot_id", snapshot.SnapshotID),
	)

	return response, nil
}

// ProcessSafetyRequest enhanced to support both legacy and snapshot modes
func (o *SnapshotOrchestrationEngine) ProcessSafetyRequest(ctx context.Context, req *types.SafetyRequest) (*types.SafetyResponse, error) {
	// Check if snapshot mode is enabled and snapshot reference is provided
	if o.config.Enabled && o.hasSnapshotReference(req) {
		o.logger.Debug("Using snapshot-based processing", zap.String("request_id", req.RequestID))
		return o.ProcessSafetyRequestWithSnapshot(ctx, req)
	}

	// Fallback to legacy mode
	o.logger.Debug("Using legacy processing mode", 
		zap.String("request_id", req.RequestID),
		zap.Bool("snapshot_enabled", o.config.Enabled),
		zap.Bool("has_snapshot_ref", o.hasSnapshotReference(req)),
	)
	return o.OrchestrationEngine.ProcessSafetyRequest(ctx, req)
}

// getValidatedSnapshot retrieves and validates a snapshot
func (o *SnapshotOrchestrationEngine) getValidatedSnapshot(
	ctx context.Context,
	snapshotID string,
	logger *logger.Logger,
) (*types.ClinicalSnapshot, error) {
	// 1. Try cache first
	if snapshot, exists := o.snapshotCache.Get(snapshotID); exists {
		logger.Debug("Snapshot cache hit", zap.String("snapshot_id", snapshotID))

		// Validate cached snapshot
		validationResult := o.snapshotValidator.ValidateIntegrity(snapshot)
		if !validationResult.Valid {
			logger.Warn("Cached snapshot failed validation",
				zap.String("snapshot_id", snapshotID),
				zap.Strings("errors", validationResult.Errors),
			)
			// Remove invalid snapshot from cache
			o.snapshotCache.Delete(snapshotID)
		} else {
			return snapshot, nil
		}
	}

	// 2. Fetch from Context Gateway
	logger.Debug("Fetching snapshot from Context Gateway", zap.String("snapshot_id", snapshotID))
	snapshot, err := o.contextClient.GetSnapshot(ctx, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch snapshot from Context Gateway: %w", err)
	}

	// 3. Validate retrieved snapshot
	validationResult := o.snapshotValidator.ValidateIntegrity(snapshot)
	if !validationResult.Valid {
		logger.Error("Retrieved snapshot failed validation",
			zap.String("snapshot_id", snapshotID),
			zap.Strings("errors", validationResult.Errors),
		)
		return nil, fmt.Errorf("snapshot validation failed: %v", validationResult.Errors)
	}

	// 4. Cache valid snapshot
	cacheTTL := o.calculateCacheTTL(snapshot)
	if err := o.snapshotCache.Set(snapshotID, snapshot, cacheTTL); err != nil {
		logger.Warn("Failed to cache snapshot", zap.Error(err))
		// Don't fail the operation if caching fails
	}

	logger.Debug("Snapshot retrieved and validated successfully",
		zap.String("snapshot_id", snapshotID),
		zap.String("patient_id", snapshot.PatientID),
		zap.Duration("cache_ttl", cacheTTL),
	)

	return snapshot, nil
}

// executeEnginesWithSnapshot executes engines with snapshot data
func (o *SnapshotOrchestrationEngine) executeEnginesWithSnapshot(
	ctx context.Context,
	engines []*registry.EngineInfo,
	req *types.SafetyRequest,
	snapshot *types.ClinicalSnapshot,
	logger *logger.Logger,
) []types.EngineResult {
	// Check if engines support snapshot-based evaluation
	compatibleEngines := o.filterSnapshotCompatibleEngines(engines)
	if len(compatibleEngines) < len(engines) {
		logger.Warn("Some engines don't support snapshot-based evaluation",
			zap.Int("total_engines", len(engines)),
			zap.Int("compatible_engines", len(compatibleEngines)),
		)
	}

	// Use the existing parallel execution but with snapshot data
	return o.executeEnginesParallel(ctx, compatibleEngines, req, snapshot.Data, logger)
}

// aggregateWithSnapshot aggregates results with snapshot reference
func (o *SnapshotOrchestrationEngine) aggregateWithSnapshot(
	req *types.SafetyRequest,
	results []types.EngineResult,
	snapshot *types.ClinicalSnapshot,
) *types.SafetyResponse {
	// Use existing aggregation logic
	response := o.responseBuilder.AggregateResults(req, results, snapshot.Data)

	// Add snapshot reference to response
	response.Metadata["snapshot_id"] = snapshot.SnapshotID
	response.Metadata["snapshot_checksum"] = snapshot.Checksum
	response.Metadata["data_completeness"] = snapshot.DataCompleteness
	response.Metadata["snapshot_created_at"] = snapshot.CreatedAt.Format(time.RFC3339)
	response.Metadata["snapshot_expires_at"] = snapshot.ExpiresAt.Format(time.RFC3339)
	response.Metadata["processing_mode"] = "snapshot_based"

	// Enhanced context version using snapshot ID
	response.ContextVersion = fmt.Sprintf("snapshot_%s_%s", 
		snapshot.SnapshotID[:8], snapshot.Version)

	return response
}

// extractSnapshotReference extracts snapshot reference from request
func (o *SnapshotOrchestrationEngine) extractSnapshotReference(req *types.SafetyRequest) *types.SnapshotReference {
	// Check if snapshot reference is in request context
	if snapshotID, exists := req.Context["snapshot_id"]; exists && snapshotID != "" {
		return &types.SnapshotReference{
			SnapshotID: snapshotID,
			// Other fields would be populated from additional context
		}
	}

	// In a real implementation, this might be extracted from a more structured field
	// For now, we check the context map
	return nil
}

// hasSnapshotReference checks if request contains snapshot reference
func (o *SnapshotOrchestrationEngine) hasSnapshotReference(req *types.SafetyRequest) bool {
	return o.extractSnapshotReference(req) != nil
}

// filterSnapshotCompatibleEngines filters engines that support snapshot-based evaluation
func (o *SnapshotOrchestrationEngine) filterSnapshotCompatibleEngines(engines []*registry.EngineInfo) []*registry.EngineInfo {
	var compatible []*registry.EngineInfo
	
	for _, engine := range engines {
		// Check if engine supports snapshot-based evaluation
		// This would typically be determined by engine capabilities or version
		if o.isEngineSnapshotCompatible(engine) {
			compatible = append(compatible, engine)
		}
	}
	
	return compatible
}

// isEngineSnapshotCompatible checks if an engine supports snapshot-based evaluation
func (o *SnapshotOrchestrationEngine) isEngineSnapshotCompatible(engine *registry.EngineInfo) bool {
	// Check engine capabilities for snapshot support
	for _, capability := range engine.Capabilities {
		if capability == "snapshot_evaluation" {
			return true
		}
	}
	
	// For now, assume all engines are compatible (during transition period)
	// In production, this would be more selective
	return true
}

// calculateCacheTTL calculates appropriate cache TTL based on snapshot properties
func (o *SnapshotOrchestrationEngine) calculateCacheTTL(snapshot *types.ClinicalSnapshot) time.Duration {
	// Use remaining time until expiration, but with minimum and maximum bounds
	timeToExpiry := time.Until(snapshot.ExpiresAt)
	
	// Ensure we don't cache beyond snapshot expiration
	if timeToExpiry <= 0 {
		return 0
	}
	
	// Use a percentage of remaining time to ensure cache expiry before snapshot expiry
	cacheTTL := time.Duration(float64(timeToExpiry) * 0.8) // 80% of remaining time
	
	// Apply bounds
	minTTL := o.config.CacheMinTTL
	maxTTL := o.config.CacheMaxTTL
	
	if cacheTTL < minTTL {
		cacheTTL = minTTL
	}
	if cacheTTL > maxTTL {
		cacheTTL = maxTTL
	}
	
	return cacheTTL
}

// GetSnapshotStats returns snapshot processing statistics
func (o *SnapshotOrchestrationEngine) GetSnapshotStats() map[string]interface{} {
	stats := make(map[string]interface{})
	
	// Cache statistics
	if o.snapshotCache != nil {
		cacheStats := o.snapshotCache.GetStats()
		stats["cache_stats"] = cacheStats
	}
	
	// Configuration
	stats["snapshot_mode_enabled"] = o.config.Enabled
	stats["request_timeout"] = o.config.RequestTimeout.String()
	stats["engine_execution_timeout"] = o.config.EngineExecutionTimeout.String()
	
	// Context Gateway client status
	stats["context_gateway_configured"] = o.contextClient != nil
	
	return stats
}

// HandleMissingFields handles cases where snapshot is missing required fields
func (o *SnapshotOrchestrationEngine) HandleMissingFields(
	ctx context.Context,
	snapshot *types.ClinicalSnapshot,
	requiredFields []string,
	logger *logger.Logger,
) (*types.ClinicalSnapshot, error) {
	// Check if live fetch is allowed for this snapshot
	if !snapshot.AllowLiveFetch {
		return nil, fmt.Errorf("required fields missing and live fetch not permitted")
	}

	// Identify fetchable fields
	fetchableFields := []string{}
	for _, field := range requiredFields {
		for _, allowedField := range snapshot.AllowedLiveFields {
			if field == allowedField {
				fetchableFields = append(fetchableFields, field)
				break
			}
		}
	}

	if len(fetchableFields) == 0 {
		return nil, fmt.Errorf("required fields not authorized for live fetch")
	}

	logger.Warn("Snapshot missing required fields, attempting live fetch",
		zap.String("snapshot_id", snapshot.SnapshotID),
		zap.Strings("required_fields", requiredFields),
		zap.Strings("fetchable_fields", fetchableFields),
	)

	// TODO: Implement live field fetching through Context Gateway
	// This would be a separate method in the Context Gateway client
	// For now, return error as live fetch is not yet implemented
	return nil, fmt.Errorf("live field fetching not yet implemented")
}

// ValidateSnapshotForRequest validates that a snapshot contains required data for a request
func (o *SnapshotOrchestrationEngine) ValidateSnapshotForRequest(
	snapshot *types.ClinicalSnapshot,
	req *types.SafetyRequest,
) error {
	// Check if snapshot matches request patient
	if snapshot.PatientID != req.PatientID {
		return fmt.Errorf("snapshot patient ID (%s) doesn't match request patient ID (%s)",
			snapshot.PatientID, req.PatientID)
	}

	// Check data completeness requirements
	if snapshot.DataCompleteness < o.config.MinDataCompleteness {
		return fmt.Errorf("snapshot data completeness (%.1f%%) below minimum required (%.1f%%)",
			snapshot.DataCompleteness, o.config.MinDataCompleteness)
	}

	// Validate that snapshot contains essential data for safety evaluation
	if snapshot.Data == nil {
		return fmt.Errorf("snapshot contains no clinical data")
	}

	// Check for demographics (usually required for safety evaluation)
	if snapshot.Data.Demographics == nil {
		return fmt.Errorf("snapshot missing demographics data")
	}

	return nil
}