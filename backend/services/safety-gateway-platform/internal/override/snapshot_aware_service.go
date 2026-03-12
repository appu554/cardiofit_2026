package override

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// SnapshotAwareOverrideService manages override operations with snapshot integration
type SnapshotAwareOverrideService struct {
	generator    *EnhancedTokenGenerator
	snapshotCache map[string]*types.ClinicalSnapshot // Simple in-memory cache for demo
	cacheMutex   sync.RWMutex
	logger       *logger.Logger
	config       *OverrideServiceConfig
}

// OverrideServiceConfig contains configuration for the override service
type OverrideServiceConfig struct {
	TokenExpirationDuration time.Duration `yaml:"token_expiration"`
	EnableSnapshotValidation bool         `yaml:"enable_snapshot_validation"`
	MaxCachedSnapshots      int          `yaml:"max_cached_snapshots"`
	RequireSnapshotForOverrides bool     `yaml:"require_snapshot_for_overrides"`
}

// NewSnapshotAwareOverrideService creates a new snapshot-aware override service
func NewSnapshotAwareOverrideService(
	generator *EnhancedTokenGenerator,
	config *OverrideServiceConfig,
	logger *logger.Logger,
) *SnapshotAwareOverrideService {
	if config == nil {
		config = &OverrideServiceConfig{
			TokenExpirationDuration:     24 * time.Hour,
			EnableSnapshotValidation:    true,
			MaxCachedSnapshots:         100,
			RequireSnapshotForOverrides: true,
		}
	}

	return &SnapshotAwareOverrideService{
		generator:     generator,
		snapshotCache: make(map[string]*types.ClinicalSnapshot),
		logger:        logger,
		config:        config,
	}
}

// ProcessOverrideRequest processes an override request with snapshot awareness
func (s *SnapshotAwareOverrideService) ProcessOverrideRequest(
	ctx context.Context,
	req *types.SafetyRequest,
	response *types.SafetyResponse,
	snapshot *types.ClinicalSnapshot,
) (*types.EnhancedOverrideToken, error) {
	startTime := time.Now()

	s.logger.Info("Processing snapshot-aware override request",
		zap.String("request_id", req.RequestID),
		zap.String("patient_id", req.PatientID),
		zap.String("snapshot_id", snapshot.SnapshotID),
		zap.String("status", string(response.Status)),
	)

	// Validate that override is required
	if !s.isOverrideRequired(response) {
		return nil, fmt.Errorf("override not required for status: %s", response.Status)
	}

	// Validate snapshot if required
	if s.config.RequireSnapshotForOverrides {
		if err := s.validateSnapshotForOverride(snapshot); err != nil {
			return nil, fmt.Errorf("snapshot validation failed: %w", err)
		}
	}

	// Cache snapshot for future reference
	s.cacheSnapshot(snapshot)

	// Generate enhanced override token
	token, err := s.generator.GenerateEnhancedToken(req, response, snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to generate enhanced token: %w", err)
	}

	// Log override token creation
	s.logOverrideTokenCreation(token, req, response, snapshot)

	duration := time.Since(startTime)
	s.logger.Info("Override request processed successfully",
		zap.String("token_id", token.TokenID),
		zap.String("request_id", req.RequestID),
		zap.String("required_level", string(token.RequiredLevel)),
		zap.Int64("processing_time_ms", duration.Milliseconds()),
	)

	return token, nil
}

// ValidateOverrideToken validates an enhanced override token
func (s *SnapshotAwareOverrideService) ValidateOverrideToken(
	ctx context.Context,
	token *types.EnhancedOverrideToken,
	clinicianID string,
	clinicianLevel types.OverrideLevel,
) (*types.OverrideValidation, error) {
	startTime := time.Now()

	s.logger.Debug("Validating enhanced override token",
		zap.String("token_id", token.TokenID),
		zap.String("clinician_id", clinicianID),
		zap.String("clinician_level", string(clinicianLevel)),
	)

	// Validate token structure and signature
	if err := s.generator.ValidateEnhancedToken(token); err != nil {
		validation := &types.OverrideValidation{
			Valid:       false,
			Reason:      fmt.Sprintf("Token validation failed: %v", err),
			ValidatedAt: time.Now(),
		}
		return validation, nil
	}

	// Check clinician authorization level
	if !s.hasRequiredAuthorizationLevel(clinicianLevel, token.RequiredLevel) {
		validation := &types.OverrideValidation{
			Valid:       false,
			Reason:      fmt.Sprintf("Insufficient authorization level: required %s, clinician has %s", 
				token.RequiredLevel, clinicianLevel),
			ValidatedAt: time.Now(),
		}
		return validation, nil
	}

	// Validate snapshot reference
	if s.config.EnableSnapshotValidation {
		if err := s.validateSnapshotReference(token.SnapshotReference); err != nil {
			validation := &types.OverrideValidation{
				Valid:       false,
				Reason:      fmt.Sprintf("Snapshot reference validation failed: %v", err),
				ValidatedAt: time.Now(),
			}
			return validation, nil
		}
	}

	// Create successful validation result
	validation := &types.OverrideValidation{
		Valid:       true,
		Token:       token,
		ClinicianID: clinicianID,
		ValidatedAt: time.Now(),
	}

	duration := time.Since(startTime)
	s.logger.Info("Override token validated successfully",
		zap.String("token_id", token.TokenID),
		zap.String("clinician_id", clinicianID),
		zap.String("snapshot_id", token.SnapshotReference.SnapshotID),
		zap.Int64("validation_time_ms", duration.Milliseconds()),
	)

	return validation, nil
}

// ReproduceDecision reproduces a decision using the reproducibility package
func (s *SnapshotAwareOverrideService) ReproduceDecision(
	ctx context.Context,
	token *types.EnhancedOverrideToken,
) (*ReproductionResult, error) {
	s.logger.Info("Reproducing decision from token",
		zap.String("token_id", token.TokenID),
		zap.String("proposal_id", token.ReproducibilityPackage.ProposalID),
	)

	// Get snapshot from cache or storage
	snapshot, err := s.getSnapshot(token.SnapshotReference.SnapshotID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve snapshot: %w", err)
	}

	// Validate snapshot integrity
	if snapshot.Checksum != token.SnapshotReference.Checksum {
		return nil, fmt.Errorf("snapshot checksum mismatch: expected %s, got %s",
			token.SnapshotReference.Checksum, snapshot.Checksum)
	}

	// Create reproduction result
	result := &ReproductionResult{
		TokenID:              token.TokenID,
		ProposalID:           token.ReproducibilityPackage.ProposalID,
		OriginalDecision:     token.DecisionSummary,
		ReproducedAt:         time.Now(),
		SnapshotValid:        true,
		EngineVersions:       token.ReproducibilityPackage.EngineVersions,
		RuleVersions:         token.ReproducibilityPackage.RuleVersions,
		DataSources:          token.ReproducibilityPackage.DataSources,
		ReproductionContext: map[string]interface{}{
			"snapshot_id":         snapshot.SnapshotID,
			"data_completeness":   snapshot.DataCompleteness,
			"original_created_at": token.CreatedAt,
			"reproduction_time":   time.Now(),
		},
	}

	s.logger.Info("Decision reproduction completed",
		zap.String("token_id", token.TokenID),
		zap.String("proposal_id", token.ReproducibilityPackage.ProposalID),
		zap.Bool("snapshot_valid", result.SnapshotValid),
	)

	return result, nil
}

// isOverrideRequired checks if an override is required for the given response
func (s *SnapshotAwareOverrideService) isOverrideRequired(response *types.SafetyResponse) bool {
	return response.Status == types.SafetyStatusUnsafe ||
		response.Status == types.SafetyStatusManualReview ||
		len(response.CriticalViolations) > 0
}

// validateSnapshotForOverride validates that a snapshot is suitable for override generation
func (s *SnapshotAwareOverrideService) validateSnapshotForOverride(snapshot *types.ClinicalSnapshot) error {
	if snapshot == nil {
		return fmt.Errorf("snapshot is required for override generation")
	}

	if snapshot.SnapshotID == "" {
		return fmt.Errorf("snapshot ID is required")
	}

	if snapshot.Checksum == "" {
		return fmt.Errorf("snapshot checksum is required")
	}

	// Check if snapshot is expired
	if time.Now().After(snapshot.ExpiresAt) {
		return fmt.Errorf("snapshot expired at %v", snapshot.ExpiresAt)
	}

	// Check data completeness threshold
	if snapshot.DataCompleteness < 50.0 {
		return fmt.Errorf("insufficient data completeness: %.1f%% (minimum 50%% required)", 
			snapshot.DataCompleteness)
	}

	return nil
}

// cacheSnapshot caches a snapshot for future reference
func (s *SnapshotAwareOverrideService) cacheSnapshot(snapshot *types.ClinicalSnapshot) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()

	// Simple LRU eviction if cache is full
	if len(s.snapshotCache) >= s.config.MaxCachedSnapshots {
		// Find oldest snapshot to evict
		oldestID := ""
		oldestTime := time.Now()
		for id, cached := range s.snapshotCache {
			if cached.CreatedAt.Before(oldestTime) {
				oldestTime = cached.CreatedAt
				oldestID = id
			}
		}
		if oldestID != "" {
			delete(s.snapshotCache, oldestID)
		}
	}

	s.snapshotCache[snapshot.SnapshotID] = snapshot
}

// getSnapshot retrieves a snapshot from cache or storage
func (s *SnapshotAwareOverrideService) getSnapshot(snapshotID string) (*types.ClinicalSnapshot, error) {
	s.cacheMutex.RLock()
	snapshot, exists := s.snapshotCache[snapshotID]
	s.cacheMutex.RUnlock()

	if exists {
		return snapshot, nil
	}

	// In a real implementation, this would fetch from persistent storage
	return nil, fmt.Errorf("snapshot %s not found in cache", snapshotID)
}

// hasRequiredAuthorizationLevel checks if clinician has required authorization level
func (s *SnapshotAwareOverrideService) hasRequiredAuthorizationLevel(
	clinicianLevel, requiredLevel types.OverrideLevel,
) bool {
	levelHierarchy := map[types.OverrideLevel]int{
		types.OverrideLevelResident:   1,
		types.OverrideLevelAttending:  2,
		types.OverrideLevelPharmacist: 3,
		types.OverrideLevelChief:      4,
	}

	clinicianLevelValue, exists1 := levelHierarchy[clinicianLevel]
	requiredLevelValue, exists2 := levelHierarchy[requiredLevel]

	if !exists1 || !exists2 {
		return false
	}

	return clinicianLevelValue >= requiredLevelValue
}

// validateSnapshotReference validates a snapshot reference
func (s *SnapshotAwareOverrideService) validateSnapshotReference(ref *types.SnapshotReference) error {
	if ref == nil {
		return fmt.Errorf("snapshot reference is required")
	}

	if ref.SnapshotID == "" {
		return fmt.Errorf("snapshot ID is required")
	}

	if ref.Checksum == "" {
		return fmt.Errorf("snapshot checksum is required")
	}

	if ref.CreatedAt.IsZero() {
		return fmt.Errorf("snapshot creation time is required")
	}

	return nil
}

// logOverrideTokenCreation logs override token creation for audit purposes
func (s *SnapshotAwareOverrideService) logOverrideTokenCreation(
	token *types.EnhancedOverrideToken,
	req *types.SafetyRequest,
	response *types.SafetyResponse,
	snapshot *types.ClinicalSnapshot,
) {
	s.logger.Info("Override token created",
		zap.String("event", "override_token_created"),
		zap.String("token_id", token.TokenID),
		zap.String("request_id", req.RequestID),
		zap.String("patient_id", req.PatientID),
		zap.String("snapshot_id", snapshot.SnapshotID),
		zap.String("required_level", string(token.RequiredLevel)),
		zap.Float64("risk_score", response.RiskScore),
		zap.Int("critical_violations", len(response.CriticalViolations)),
		zap.String("proposal_id", token.ReproducibilityPackage.ProposalID),
		zap.Time("expires_at", token.ExpiresAt),
	)
}

// GetServiceMetrics returns service metrics
func (s *SnapshotAwareOverrideService) GetServiceMetrics() map[string]interface{} {
	s.cacheMutex.RLock()
	cacheSize := len(s.snapshotCache)
	s.cacheMutex.RUnlock()

	return map[string]interface{}{
		"service_version":          "1.0.0",
		"snapshot_aware":          true,
		"cached_snapshots":        cacheSize,
		"max_cached_snapshots":    s.config.MaxCachedSnapshots,
		"snapshot_validation_enabled": s.config.EnableSnapshotValidation,
		"require_snapshot":        s.config.RequireSnapshotForOverrides,
		"token_expiration":        s.config.TokenExpirationDuration.String(),
	}
}

// ReproductionResult represents the result of decision reproduction
type ReproductionResult struct {
	TokenID              string                 `json:"token_id"`
	ProposalID           string                 `json:"proposal_id"`
	OriginalDecision     *types.DecisionSummary `json:"original_decision"`
	ReproducedAt         time.Time              `json:"reproduced_at"`
	SnapshotValid        bool                   `json:"snapshot_valid"`
	EngineVersions       map[string]string      `json:"engine_versions"`
	RuleVersions         map[string]string      `json:"rule_versions"`
	DataSources          []string               `json:"data_sources"`
	ReproductionContext  map[string]interface{} `json:"reproduction_context"`
}