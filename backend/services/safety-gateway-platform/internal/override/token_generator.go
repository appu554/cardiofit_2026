package override

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// EnhancedTokenGenerator handles snapshot-aware override token generation
type EnhancedTokenGenerator struct {
	signingKey []byte
	logger     *logger.Logger
}

// NewEnhancedTokenGenerator creates a new enhanced token generator
func NewEnhancedTokenGenerator(signingKey []byte, logger *logger.Logger) *EnhancedTokenGenerator {
	return &EnhancedTokenGenerator{
		signingKey: signingKey,
		logger:     logger,
	}
}

// GenerateEnhancedToken creates an enhanced override token with snapshot integration
func (g *EnhancedTokenGenerator) GenerateEnhancedToken(
	req *types.SafetyRequest,
	response *types.SafetyResponse,
	snapshot *types.ClinicalSnapshot,
) (*types.EnhancedOverrideToken, error) {
	startTime := time.Now()

	g.logger.Debug("Generating enhanced override token",
		zap.String("request_id", req.RequestID),
		zap.String("patient_id", req.PatientID),
		zap.String("snapshot_id", snapshot.SnapshotID),
	)

	// Generate unique token ID
	tokenID, err := g.generateTokenID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token ID: %w", err)
	}

	// Create snapshot reference
	snapshotRef := &types.SnapshotReference{
		SnapshotID:       snapshot.SnapshotID,
		Checksum:         snapshot.Checksum,
		CreatedAt:        snapshot.CreatedAt,
		DataCompleteness: snapshot.DataCompleteness,
	}

	// Create reproducibility package
	reproducibilityPkg, err := g.createReproducibilityPackage(req, response, snapshot)
	if err != nil {
		return nil, fmt.Errorf("failed to create reproducibility package: %w", err)
	}

	// Create decision summary
	decisionSummary := &types.DecisionSummary{
		Status:             response.Status,
		CriticalViolations: response.CriticalViolations,
		EnginesFailed:      response.EnginesFailed,
		RiskScore:          response.RiskScore,
		Explanation:        g.createSummaryExplanation(response),
	}

	// Determine required override level
	requiredLevel := g.determineRequiredOverrideLevel(response.RiskScore, len(response.CriticalViolations))

	// Create context hash
	contextHash, err := g.createContextHash(snapshot, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create context hash: %w", err)
	}

	// Create enhanced token
	token := &types.EnhancedOverrideToken{
		TokenID:         tokenID,
		RequestID:       req.RequestID,
		PatientID:       req.PatientID,
		DecisionSummary: decisionSummary,
		RequiredLevel:   requiredLevel,
		ExpiresAt:       time.Now().Add(24 * time.Hour), // 24-hour expiration
		ContextHash:     contextHash,
		CreatedAt:       time.Now(),
		SnapshotReference:      snapshotRef,
		ReproducibilityPackage: reproducibilityPkg,
	}

	// Generate cryptographic signature
	signature, err := g.signToken(token)
	if err != nil {
		return nil, fmt.Errorf("failed to sign token: %w", err)
	}
	token.Signature = signature

	duration := time.Since(startTime)
	g.logger.Info("Enhanced override token generated",
		zap.String("token_id", tokenID),
		zap.String("request_id", req.RequestID),
		zap.String("patient_id", req.PatientID),
		zap.String("snapshot_id", snapshot.SnapshotID),
		zap.String("required_level", string(requiredLevel)),
		zap.Int64("generation_time_ms", duration.Milliseconds()),
	)

	return token, nil
}

// generateTokenID creates a cryptographically secure token ID
func (g *EnhancedTokenGenerator) generateTokenID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// createReproducibilityPackage creates a package for exact decision reproduction
func (g *EnhancedTokenGenerator) createReproducibilityPackage(
	req *types.SafetyRequest,
	response *types.SafetyResponse,
	snapshot *types.ClinicalSnapshot,
) (*types.ReproducibilityPackage, error) {
	// Generate unique proposal ID for this decision
	proposalBytes := make([]byte, 8)
	if _, err := rand.Read(proposalBytes); err != nil {
		return nil, err
	}
	proposalID := hex.EncodeToString(proposalBytes)

	// Collect engine versions from response metadata
	engineVersions := make(map[string]string)
	for _, engineResult := range response.EngineResults {
		if version, ok := engineResult.Metadata["version"].(string); ok {
			engineVersions[engineResult.EngineID] = version
		} else {
			engineVersions[engineResult.EngineID] = "unknown"
		}
	}

	// Collect rule versions (from snapshot metadata if available)
	ruleVersions := make(map[string]string)
	if snapshot.Metadata != nil {
		if rules, ok := snapshot.Metadata["rule_versions"].(map[string]interface{}); ok {
			for ruleName, version := range rules {
				if versionStr, ok := version.(string); ok {
					ruleVersions[ruleName] = versionStr
				}
			}
		}
	}

	// Collect data sources
	dataSources := snapshot.Data.DataSources
	if dataSources == nil {
		dataSources = []string{"snapshot"}
	}

	return &types.ReproducibilityPackage{
		ProposalID:           proposalID,
		EngineVersions:       engineVersions,
		RuleVersions:         ruleVersions,
		DataSources:          dataSources,
		SnapshotCreationTime: snapshot.CreatedAt,
		ValidationTime:       time.Now(),
		Metadata: map[string]interface{}{
			"request_type":    req.ActionType,
			"priority":        req.Priority,
			"context_version": response.ContextVersion,
			"processing_time": response.ProcessingTime.String(),
		},
	}, nil
}

// createSummaryExplanation creates a brief explanation for the decision summary
func (g *EnhancedTokenGenerator) createSummaryExplanation(response *types.SafetyResponse) string {
	if response.Explanation != nil {
		return response.Explanation.Summary
	}
	
	// Create default explanation based on status
	switch response.Status {
	case types.SafetyStatusUnsafe:
		return fmt.Sprintf("Safety validation failed with %d critical violations (risk score: %.2f)", 
			len(response.CriticalViolations), response.RiskScore)
	case types.SafetyStatusWarning:
		return fmt.Sprintf("Safety warnings detected (risk score: %.2f)", response.RiskScore)
	case types.SafetyStatusManualReview:
		return "Manual review required due to safety engine failures or inconclusive results"
	default:
		return "Safety evaluation completed"
	}
}

// determineRequiredOverrideLevel determines the required authorization level
func (g *EnhancedTokenGenerator) determineRequiredOverrideLevel(riskScore float64, criticalViolations int) types.OverrideLevel {
	// High-risk scenarios require chief authorization
	if riskScore >= 0.9 || criticalViolations >= 3 {
		return types.OverrideLevelChief
	}
	
	// Medium-high risk requires pharmacist authorization
	if riskScore >= 0.7 || criticalViolations >= 2 {
		return types.OverrideLevelPharmacist
	}
	
	// Medium risk requires attending authorization
	if riskScore >= 0.4 || criticalViolations >= 1 {
		return types.OverrideLevelAttending
	}
	
	// Low risk requires resident authorization
	return types.OverrideLevelResident
}

// createContextHash creates a hash of the clinical context for validation
func (g *EnhancedTokenGenerator) createContextHash(snapshot *types.ClinicalSnapshot, req *types.SafetyRequest) (string, error) {
	// Create hash payload including key elements
	hashPayload := fmt.Sprintf("%s|%s|%s|%s|%s|%v|%s",
		snapshot.SnapshotID,
		snapshot.Checksum,
		req.RequestID,
		req.ActionType,
		req.PatientID,
		req.MedicationIDs,
		snapshot.CreatedAt.Format(time.RFC3339),
	)
	
	hasher := sha256.New()
	hasher.Write([]byte(hashPayload))
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// signToken creates a cryptographic signature for the token
func (g *EnhancedTokenGenerator) signToken(token *types.EnhancedOverrideToken) (string, error) {
	if len(g.signingKey) == 0 {
		return "", fmt.Errorf("signing key not configured")
	}

	// Create signature payload (exclude signature field)
	signaturePayload := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s",
		token.TokenID,
		token.RequestID,
		token.PatientID,
		token.ContextHash,
		token.CreatedAt.Format(time.RFC3339),
		token.ExpiresAt.Format(time.RFC3339),
		string(token.RequiredLevel),
	)
	
	// Add snapshot reference to signature
	if token.SnapshotReference != nil {
		signaturePayload += fmt.Sprintf("|%s|%s", 
			token.SnapshotReference.SnapshotID, 
			token.SnapshotReference.Checksum,
		)
	}

	// Generate HMAC signature
	h := hmac.New(sha256.New, g.signingKey)
	h.Write([]byte(signaturePayload))
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ValidateEnhancedToken validates an enhanced override token
func (g *EnhancedTokenGenerator) ValidateEnhancedToken(token *types.EnhancedOverrideToken) error {
	// Check expiration
	if time.Now().After(token.ExpiresAt) {
		return fmt.Errorf("token expired at %v", token.ExpiresAt)
	}

	// Verify signature
	expectedSignature, err := g.signToken(token)
	if err != nil {
		return fmt.Errorf("failed to generate expected signature: %w", err)
	}

	if token.Signature != expectedSignature {
		return fmt.Errorf("token signature verification failed")
	}

	// Validate snapshot reference
	if token.SnapshotReference == nil {
		return fmt.Errorf("snapshot reference is required")
	}

	if token.SnapshotReference.SnapshotID == "" {
		return fmt.Errorf("snapshot ID is required")
	}

	if token.SnapshotReference.Checksum == "" {
		return fmt.Errorf("snapshot checksum is required")
	}

	return nil
}

// GetTokenMetrics returns token generation metrics
func (g *EnhancedTokenGenerator) GetTokenMetrics() map[string]interface{} {
	return map[string]interface{}{
		"generator_version":    "1.0.0",
		"signing_key_configured": len(g.signingKey) > 0,
		"supported_algorithms": []string{"HMAC-SHA256"},
		"default_expiration":   "24h",
		"snapshot_aware":      true,
		"reproducibility_enabled": true,
	}
}