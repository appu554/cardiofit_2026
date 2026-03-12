package reproducibility

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// DecisionReplayService handles exact reproduction of clinical decisions
type DecisionReplayService struct {
	snapshotStore  SnapshotStore
	engineRegistry EngineRegistry
	logger         *logger.Logger
	config         *ReplayConfig
	replayCache    map[string]*ReplayResult
	cacheMutex     sync.RWMutex
}

// ReplayConfig contains configuration for decision replay
type ReplayConfig struct {
	EnableReplay            bool          `yaml:"enable_replay"`
	MaxConcurrentReplays    int           `yaml:"max_concurrent_replays"`
	ReplayTimeout           time.Duration `yaml:"replay_timeout"`
	CacheReplayResults      bool          `yaml:"cache_replay_results"`
	CacheTTL               time.Duration `yaml:"cache_ttl"`
	VerifyReproducibility   bool          `yaml:"verify_reproducibility"`
	AllowPartialReplay      bool          `yaml:"allow_partial_replay"`
	LogReplayDetails        bool          `yaml:"log_replay_details"`
}

// SnapshotStore interface for retrieving historical snapshots
type SnapshotStore interface {
	GetSnapshot(snapshotID string) (*types.ClinicalSnapshot, error)
	ValidateSnapshotIntegrity(snapshot *types.ClinicalSnapshot) error
	GetSnapshotMetadata(snapshotID string) (*SnapshotMetadata, error)
}

// EngineRegistry interface for managing safety engines
type EngineRegistry interface {
	GetEngine(engineID string) (types.SafetyEngine, error)
	GetEngineVersion(engineID string) (string, error)
	CreateEngineWithVersion(engineID, version string) (types.SafetyEngine, error)
	ListAvailableEngines() ([]EngineInfo, error)
}

// NewDecisionReplayService creates a new decision replay service
func NewDecisionReplayService(
	snapshotStore SnapshotStore,
	engineRegistry EngineRegistry,
	config *ReplayConfig,
	logger *logger.Logger,
) *DecisionReplayService {
	if config == nil {
		config = &ReplayConfig{
			EnableReplay:           true,
			MaxConcurrentReplays:   5,
			ReplayTimeout:          30 * time.Second,
			CacheReplayResults:     true,
			CacheTTL:              time.Hour,
			VerifyReproducibility:  true,
			AllowPartialReplay:     false,
			LogReplayDetails:       true,
		}
	}

	return &DecisionReplayService{
		snapshotStore:  snapshotStore,
		engineRegistry: engineRegistry,
		logger:         logger,
		config:         config,
		replayCache:    make(map[string]*ReplayResult),
	}
}

// ReplayDecision replays a decision using the reproducibility package
func (r *DecisionReplayService) ReplayDecision(
	ctx context.Context,
	token *types.EnhancedOverrideToken,
) (*DecisionReplayResult, error) {
	if !r.config.EnableReplay {
		return nil, fmt.Errorf("decision replay is disabled")
	}

	startTime := time.Now()
	replayID := fmt.Sprintf("replay_%s_%d", token.TokenID, startTime.UnixNano())

	r.logger.Info("Starting decision replay",
		zap.String("replay_id", replayID),
		zap.String("token_id", token.TokenID),
		zap.String("proposal_id", token.ReproducibilityPackage.ProposalID),
		zap.String("snapshot_id", token.SnapshotReference.SnapshotID),
	)

	// Check cache first
	cacheKey := r.generateCacheKey(token)
	if r.config.CacheReplayResults {
		if cachedResult := r.getCachedReplay(cacheKey); cachedResult != nil {
			r.logger.Debug("Returning cached replay result",
				zap.String("replay_id", replayID),
				zap.String("cache_key", cacheKey),
			)
			return cachedResult.DecisionReplay, nil
		}
	}

	// Create timeout context
	replayCtx, cancel := context.WithTimeout(ctx, r.config.ReplayTimeout)
	defer cancel()

	// Execute replay
	result, err := r.executeReplay(replayCtx, replayID, token)
	if err != nil {
		r.logger.Error("Decision replay failed",
			zap.String("replay_id", replayID),
			zap.String("token_id", token.TokenID),
			zap.Error(err),
		)
		return nil, err
	}

	// Cache result
	if r.config.CacheReplayResults {
		r.cacheReplay(cacheKey, &ReplayResult{
			CachedAt:      time.Now(),
			DecisionReplay: result,
		})
	}

	duration := time.Since(startTime)
	r.logger.Info("Decision replay completed",
		zap.String("replay_id", replayID),
		zap.String("token_id", token.TokenID),
		zap.Bool("successful", result.Success),
		zap.Float64("reproducibility_score", result.ReproducibilityScore),
		zap.Int64("replay_duration_ms", duration.Milliseconds()),
	)

	return result, nil
}

// executeReplay performs the actual decision replay
func (r *DecisionReplayService) executeReplay(
	ctx context.Context,
	replayID string,
	token *types.EnhancedOverrideToken,
) (*DecisionReplayResult, error) {
	result := &DecisionReplayResult{
		ReplayID:           replayID,
		TokenID:            token.TokenID,
		ProposalID:         token.ReproducibilityPackage.ProposalID,
		ReplayTime:         time.Now(),
		Success:            false,
		OriginalDecision:   token.DecisionSummary,
		ReproducedDecision: &types.DecisionSummary{},
		EngineComparisons:  []EngineComparison{},
		Issues:             []ReproducibilityIssue{},
	}

	// Step 1: Retrieve and validate snapshot
	snapshot, err := r.retrieveAndValidateSnapshot(token.SnapshotReference)
	if err != nil {
		result.Issues = append(result.Issues, ReproducibilityIssue{
			Type:        "snapshot_error",
			Description: fmt.Sprintf("Failed to retrieve/validate snapshot: %v", err),
			Severity:    "critical",
		})
		return result, nil
	}

	result.SnapshotValid = true

	// Step 2: Reconstruct safety request
	safetyRequest, err := r.reconstructSafetyRequest(token, snapshot)
	if err != nil {
		result.Issues = append(result.Issues, ReproducibilityIssue{
			Type:        "request_reconstruction_error",
			Description: fmt.Sprintf("Failed to reconstruct safety request: %v", err),
			Severity:    "critical",
		})
		return result, nil
	}

	// Step 3: Replay decision with original engines/versions
	replayedResponse, engineComparisons, err := r.replayWithOriginalEngines(
		ctx, 
		safetyRequest, 
		snapshot, 
		token.ReproducibilityPackage,
	)
	if err != nil {
		result.Issues = append(result.Issues, ReproducibilityIssue{
			Type:        "engine_replay_error",
			Description: fmt.Sprintf("Failed to replay with original engines: %v", err),
			Severity:    "high",
		})
		
		// Try partial replay if allowed
		if r.config.AllowPartialReplay {
			replayedResponse, engineComparisons, _ = r.attemptPartialReplay(
				ctx, 
				safetyRequest, 
				snapshot, 
				token.ReproducibilityPackage,
			)
		}
	}

	if replayedResponse != nil {
		result.ReproducedDecision = &types.DecisionSummary{
			Status:             replayedResponse.Status,
			CriticalViolations: replayedResponse.CriticalViolations,
			EnginesFailed:      replayedResponse.EnginesFailed,
			RiskScore:          replayedResponse.RiskScore,
			Explanation:        "Reproduced decision from replay",
		}
	}

	result.EngineComparisons = engineComparisons

	// Step 4: Compare results and calculate reproducibility score
	result.ReproducibilityScore = r.calculateReproducibilityScore(result)

	// Step 5: Determine success
	result.Success = result.ReproducibilityScore >= 0.9 && len(result.Issues) == 0

	// Step 6: Generate detailed comparison
	result.DetailedComparison = r.generateDetailedComparison(result)

	if r.config.LogReplayDetails {
		r.logReplayDetails(result)
	}

	return result, nil
}

// retrieveAndValidateSnapshot retrieves and validates the historical snapshot
func (r *DecisionReplayService) retrieveAndValidateSnapshot(
	snapshotRef *types.SnapshotReference,
) (*types.ClinicalSnapshot, error) {
	// Get snapshot from storage
	snapshot, err := r.snapshotStore.GetSnapshot(snapshotRef.SnapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve snapshot: %w", err)
	}

	// Validate integrity
	if err := r.snapshotStore.ValidateSnapshotIntegrity(snapshot); err != nil {
		return nil, fmt.Errorf("snapshot integrity validation failed: %w", err)
	}

	// Verify checksum matches
	if snapshot.Checksum != snapshotRef.Checksum {
		return nil, fmt.Errorf("snapshot checksum mismatch: expected %s, got %s",
			snapshotRef.Checksum, snapshot.Checksum)
	}

	return snapshot, nil
}

// reconstructSafetyRequest reconstructs the original safety request
func (r *DecisionReplayService) reconstructSafetyRequest(
	token *types.EnhancedOverrideToken,
	snapshot *types.ClinicalSnapshot,
) (*types.SafetyRequest, error) {
	// Extract request details from token and snapshot metadata
	metadata, ok := token.ReproducibilityPackage.Metadata["request_type"]
	if !ok {
		return nil, fmt.Errorf("request_type not found in reproducibility metadata")
	}

	requestType, ok := metadata.(string)
	if !ok {
		return nil, fmt.Errorf("invalid request_type format in metadata")
	}

	// Reconstruct the safety request
	request := &types.SafetyRequest{
		RequestID:   token.RequestID,
		PatientID:   token.PatientID,
		ActionType:  requestType,
		Priority:    extractPriority(token.ReproducibilityPackage.Metadata),
		Timestamp:   token.CreatedAt,
		Source:      "replay",
	}

	// Extract medication IDs from snapshot if available
	if snapshot.Data != nil {
		for _, med := range snapshot.Data.ActiveMedications {
			request.MedicationIDs = append(request.MedicationIDs, med.ID)
		}
		
		for _, condition := range snapshot.Data.Conditions {
			request.ConditionIDs = append(request.ConditionIDs, condition.ID)
		}
	}

	return request, nil
}

// replayWithOriginalEngines replays the decision using original engines and versions
func (r *DecisionReplayService) replayWithOriginalEngines(
	ctx context.Context,
	request *types.SafetyRequest,
	snapshot *types.ClinicalSnapshot,
	reproPackage *types.ReproducibilityPackage,
) (*types.SafetyResponse, []EngineComparison, error) {
	engineComparisons := []EngineComparison{}
	engineResults := []types.EngineResult{}

	// Replay each engine with its original version
	for engineID, originalVersion := range reproPackage.EngineVersions {
		comparison := EngineComparison{
			EngineID:        engineID,
			OriginalVersion: originalVersion,
		}

		// Get current engine version
		currentVersion, err := r.engineRegistry.GetEngineVersion(engineID)
		if err != nil {
			comparison.Issues = append(comparison.Issues, fmt.Sprintf("Failed to get current version: %v", err))
			engineComparisons = append(engineComparisons, comparison)
			continue
		}

		comparison.CurrentVersion = currentVersion
		comparison.VersionMatch = originalVersion == currentVersion

		// Create engine with original version if possible
		var engine types.SafetyEngine
		if comparison.VersionMatch {
			engine, err = r.engineRegistry.GetEngine(engineID)
		} else {
			engine, err = r.engineRegistry.CreateEngineWithVersion(engineID, originalVersion)
			if err != nil {
				comparison.Issues = append(comparison.Issues, fmt.Sprintf("Failed to create engine with version %s: %v", originalVersion, err))
				engineComparisons = append(engineComparisons, comparison)
				continue
			}
		}

		// Execute engine evaluation
		if snapshotAwareEngine, ok := engine.(types.SnapshotAwareEngine); ok {
			// Use snapshot-aware evaluation
			result, err := snapshotAwareEngine.EvaluateWithSnapshot(ctx, request, snapshot)
			if err != nil {
				comparison.Issues = append(comparison.Issues, fmt.Sprintf("Engine evaluation failed: %v", err))
			} else {
				comparison.Successful = true
				engineResults = append(engineResults, *result)
			}
		} else {
			// Fall back to legacy evaluation
			result, err := engine.Evaluate(ctx, request, snapshot.Data)
			if err != nil {
				comparison.Issues = append(comparison.Issues, fmt.Sprintf("Legacy engine evaluation failed: %v", err))
			} else {
				comparison.Successful = true
				engineResults = append(engineResults, *result)
			}
		}

		engineComparisons = append(engineComparisons, comparison)
	}

	// Aggregate engine results into safety response
	response := r.aggregateEngineResults(request, engineResults)

	return response, engineComparisons, nil
}

// attemptPartialReplay attempts partial replay with available engines
func (r *DecisionReplayService) attemptPartialReplay(
	ctx context.Context,
	request *types.SafetyRequest,
	snapshot *types.ClinicalSnapshot,
	reproPackage *types.ReproducibilityPackage,
) (*types.SafetyResponse, []EngineComparison, error) {
	r.logger.Info("Attempting partial replay with available engines")

	availableEngines, err := r.engineRegistry.ListAvailableEngines()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list available engines: %w", err)
	}

	engineResults := []types.EngineResult{}
	engineComparisons := []EngineComparison{}

	for _, engineInfo := range availableEngines {
		// Skip if this engine wasn't in the original decision
		if _, exists := reproPackage.EngineVersions[engineInfo.ID]; !exists {
			continue
		}

		engine, err := r.engineRegistry.GetEngine(engineInfo.ID)
		if err != nil {
			continue
		}

		comparison := EngineComparison{
			EngineID:       engineInfo.ID,
			CurrentVersion: engineInfo.Version,
			PartialReplay:  true,
		}

		// Execute with current engine
		var result *types.EngineResult
		if snapshotAwareEngine, ok := engine.(types.SnapshotAwareEngine); ok {
			result, err = snapshotAwareEngine.EvaluateWithSnapshot(ctx, request, snapshot)
		} else {
			result, err = engine.Evaluate(ctx, request, snapshot.Data)
		}

		if err != nil {
			comparison.Issues = append(comparison.Issues, fmt.Sprintf("Partial replay failed: %v", err))
		} else {
			comparison.Successful = true
			engineResults = append(engineResults, *result)
		}

		engineComparisons = append(engineComparisons, comparison)
	}

	response := r.aggregateEngineResults(request, engineResults)
	return response, engineComparisons, nil
}

// aggregateEngineResults aggregates individual engine results into a safety response
func (r *DecisionReplayService) aggregateEngineResults(
	request *types.SafetyRequest,
	engineResults []types.EngineResult,
) *types.SafetyResponse {
	if len(engineResults) == 0 {
		return &types.SafetyResponse{
			RequestID:      request.RequestID,
			Status:         types.SafetyStatusError,
			RiskScore:      0.0,
			EngineResults:  []types.EngineResult{},
			ProcessingTime: 0,
			Timestamp:      time.Now(),
		}
	}

	response := &types.SafetyResponse{
		RequestID:          request.RequestID,
		Status:             types.SafetyStatusSafe,
		RiskScore:          0.0,
		CriticalViolations: []string{},
		Warnings:           []string{},
		EngineResults:      engineResults,
		EnginesFailed:      []string{},
		ProcessingTime:     0,
		Timestamp:          time.Now(),
	}

	// Aggregate results
	maxRiskScore := 0.0
	hasUnsafeResult := false
	hasWarning := false

	for _, result := range engineResults {
		if result.RiskScore > maxRiskScore {
			maxRiskScore = result.RiskScore
		}

		if result.Status == types.SafetyStatusUnsafe {
			hasUnsafeResult = true
			response.CriticalViolations = append(response.CriticalViolations, result.Violations...)
		}

		if result.Status == types.SafetyStatusWarning {
			hasWarning = true
			response.Warnings = append(response.Warnings, result.Warnings...)
		}

		if result.Error != "" {
			response.EnginesFailed = append(response.EnginesFailed, result.EngineID)
		}
	}

	response.RiskScore = maxRiskScore

	// Determine overall status
	if hasUnsafeResult {
		response.Status = types.SafetyStatusUnsafe
	} else if hasWarning || len(response.EnginesFailed) > 0 {
		response.Status = types.SafetyStatusWarning
	} else {
		response.Status = types.SafetyStatusSafe
	}

	return response
}

// calculateReproducibilityScore calculates how well the replay matched the original
func (r *DecisionReplayService) calculateReproducibilityScore(result *DecisionReplayResult) float64 {
	if result.OriginalDecision == nil || result.ReproducedDecision == nil {
		return 0.0
	}

	score := 0.0
	factors := 0.0

	// Status match (40% weight)
	if result.OriginalDecision.Status == result.ReproducedDecision.Status {
		score += 0.4
	}
	factors += 0.4

	// Risk score similarity (30% weight)
	riskScoreDiff := abs(result.OriginalDecision.RiskScore - result.ReproducedDecision.RiskScore)
	riskScoreSimilarity := max(0, 1.0-riskScoreDiff)
	score += riskScoreSimilarity * 0.3
	factors += 0.3

	// Critical violations match (20% weight)
	violationSimilarity := r.calculateStringSliceSimilarity(
		result.OriginalDecision.CriticalViolations,
		result.ReproducedDecision.CriticalViolations,
	)
	score += violationSimilarity * 0.2
	factors += 0.2

	// Engine success rate (10% weight)
	successfulEngines := 0
	totalEngines := len(result.EngineComparisons)
	for _, comp := range result.EngineComparisons {
		if comp.Successful {
			successfulEngines++
		}
	}

	if totalEngines > 0 {
		engineSuccessRate := float64(successfulEngines) / float64(totalEngines)
		score += engineSuccessRate * 0.1
		factors += 0.1
	}

	if factors > 0 {
		return score / factors
	}
	return 0.0
}

// calculateStringSliceSimilarity calculates similarity between two string slices
func (r *DecisionReplayService) calculateStringSliceSimilarity(slice1, slice2 []string) float64 {
	if len(slice1) == 0 && len(slice2) == 0 {
		return 1.0
	}
	
	if len(slice1) == 0 || len(slice2) == 0 {
		return 0.0
	}

	// Convert to sets
	set1 := make(map[string]bool)
	set2 := make(map[string]bool)
	
	for _, item := range slice1 {
		set1[item] = true
	}
	
	for _, item := range slice2 {
		set2[item] = true
	}

	// Calculate intersection
	intersection := 0
	for item := range set1 {
		if set2[item] {
			intersection++
		}
	}

	// Calculate union
	union := len(set1) + len(set2) - intersection

	if union == 0 {
		return 1.0
	}

	return float64(intersection) / float64(union)
}

// generateDetailedComparison generates detailed comparison results
func (r *DecisionReplayService) generateDetailedComparison(result *DecisionReplayResult) *DetailedComparison {
	comparison := &DetailedComparison{
		StatusMatch:        result.OriginalDecision.Status == result.ReproducedDecision.Status,
		RiskScoreDelta:     result.ReproducedDecision.RiskScore - result.OriginalDecision.RiskScore,
		ViolationsAdded:    []string{},
		ViolationsRemoved:  []string{},
		EnginesSuccessful:  0,
		EnginesFailed:      0,
		VersionMismatches:  []string{},
	}

	// Calculate violations differences
	origViolations := make(map[string]bool)
	for _, v := range result.OriginalDecision.CriticalViolations {
		origViolations[v] = true
	}

	reproViolations := make(map[string]bool)
	for _, v := range result.ReproducedDecision.CriticalViolations {
		reproViolations[v] = true
	}

	// Find added violations
	for v := range reproViolations {
		if !origViolations[v] {
			comparison.ViolationsAdded = append(comparison.ViolationsAdded, v)
		}
	}

	// Find removed violations
	for v := range origViolations {
		if !reproViolations[v] {
			comparison.ViolationsRemoved = append(comparison.ViolationsRemoved, v)
		}
	}

	// Calculate engine success/failure counts
	for _, comp := range result.EngineComparisons {
		if comp.Successful {
			comparison.EnginesSuccessful++
		} else {
			comparison.EnginesFailed++
		}

		if !comp.VersionMatch {
			comparison.VersionMismatches = append(comparison.VersionMismatches, 
				fmt.Sprintf("%s: %s -> %s", comp.EngineID, comp.OriginalVersion, comp.CurrentVersion))
		}
	}

	return comparison
}

// Cache management methods
func (r *DecisionReplayService) generateCacheKey(token *types.EnhancedOverrideToken) string {
	return fmt.Sprintf("replay_%s_%s", token.TokenID, token.ReproducibilityPackage.ProposalID)
}

func (r *DecisionReplayService) getCachedReplay(cacheKey string) *ReplayResult {
	r.cacheMutex.RLock()
	defer r.cacheMutex.RUnlock()

	if result, exists := r.replayCache[cacheKey]; exists {
		if time.Since(result.CachedAt) <= r.config.CacheTTL {
			return result
		}
		delete(r.replayCache, cacheKey)
	}
	return nil
}

func (r *DecisionReplayService) cacheReplay(cacheKey string, result *ReplayResult) {
	r.cacheMutex.Lock()
	defer r.cacheMutex.Unlock()
	r.replayCache[cacheKey] = result
}

// logReplayDetails logs detailed replay information
func (r *DecisionReplayService) logReplayDetails(result *DecisionReplayResult) {
	r.logger.Info("Detailed replay results",
		zap.String("replay_id", result.ReplayID),
		zap.String("token_id", result.TokenID),
		zap.Bool("successful", result.Success),
		zap.Float64("reproducibility_score", result.ReproducibilityScore),
		zap.Bool("snapshot_valid", result.SnapshotValid),
		zap.Int("engine_comparisons", len(result.EngineComparisons)),
		zap.Int("issues_count", len(result.Issues)),
	)

	for _, issue := range result.Issues {
		r.logger.Warn("Reproducibility issue",
			zap.String("replay_id", result.ReplayID),
			zap.String("issue_type", issue.Type),
			zap.String("severity", issue.Severity),
			zap.String("description", issue.Description),
		)
	}
}

// GetReplayMetrics returns replay service metrics
func (r *DecisionReplayService) GetReplayMetrics() map[string]interface{} {
	r.cacheMutex.RLock()
	cacheSize := len(r.replayCache)
	r.cacheMutex.RUnlock()

	return map[string]interface{}{
		"service_version":          "1.0.0",
		"replay_enabled":          r.config.EnableReplay,
		"max_concurrent_replays":  r.config.MaxConcurrentReplays,
		"replay_timeout":          r.config.ReplayTimeout.String(),
		"cache_size":              cacheSize,
		"verify_reproducibility":  r.config.VerifyReproducibility,
		"allow_partial_replay":    r.config.AllowPartialReplay,
	}
}

// Utility functions
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func extractPriority(metadata map[string]interface{}) string {
	if priority, exists := metadata["priority"]; exists {
		if priorityStr, ok := priority.(string); ok {
			return priorityStr
		}
	}
	return "normal"
}

// Data structures

// DecisionReplayResult contains the results of decision replay
type DecisionReplayResult struct {
	ReplayID             string                    `json:"replay_id"`
	TokenID              string                    `json:"token_id"`
	ProposalID           string                    `json:"proposal_id"`
	ReplayTime           time.Time                 `json:"replay_time"`
	Success              bool                      `json:"success"`
	ReproducibilityScore float64                   `json:"reproducibility_score"`
	SnapshotValid        bool                      `json:"snapshot_valid"`
	OriginalDecision     *types.DecisionSummary    `json:"original_decision"`
	ReproducedDecision   *types.DecisionSummary    `json:"reproduced_decision"`
	EngineComparisons    []EngineComparison        `json:"engine_comparisons"`
	Issues               []ReproducibilityIssue    `json:"issues"`
	DetailedComparison   *DetailedComparison       `json:"detailed_comparison"`
}

// EngineComparison represents comparison results for a specific engine
type EngineComparison struct {
	EngineID        string   `json:"engine_id"`
	OriginalVersion string   `json:"original_version"`
	CurrentVersion  string   `json:"current_version"`
	VersionMatch    bool     `json:"version_match"`
	Successful      bool     `json:"successful"`
	PartialReplay   bool     `json:"partial_replay"`
	Issues          []string `json:"issues"`
}

// ReproducibilityIssue represents an issue found during replay
type ReproducibilityIssue struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

// DetailedComparison provides detailed comparison between original and reproduced decisions
type DetailedComparison struct {
	StatusMatch        bool     `json:"status_match"`
	RiskScoreDelta     float64  `json:"risk_score_delta"`
	ViolationsAdded    []string `json:"violations_added"`
	ViolationsRemoved  []string `json:"violations_removed"`
	EnginesSuccessful  int      `json:"engines_successful"`
	EnginesFailed      int      `json:"engines_failed"`
	VersionMismatches  []string `json:"version_mismatches"`
}

// ReplayResult represents cached replay results
type ReplayResult struct {
	CachedAt      time.Time
	DecisionReplay *DecisionReplayResult
}

// SnapshotMetadata contains metadata about a snapshot
type SnapshotMetadata struct {
	SnapshotID      string                 `json:"snapshot_id"`
	CreatedAt       time.Time              `json:"created_at"`
	Size            int64                  `json:"size"`
	CompressionType string                 `json:"compression_type"`
	StorageLocation string                 `json:"storage_location"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// EngineInfo contains information about available engines
type EngineInfo struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Capabilities []string `json:"capabilities"`
}