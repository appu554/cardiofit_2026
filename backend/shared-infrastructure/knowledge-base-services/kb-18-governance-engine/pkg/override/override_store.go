// Package override manages override requests, approvals, acknowledgments,
// and escalations. It also provides pattern monitoring to detect suspicious
// override patterns that may indicate policy circumvention.
package override

import (
	"context"
	"errors"
	"sync"
	"time"

	"kb-18-governance-engine/pkg/types"
)

// =============================================================================
// ERRORS
// =============================================================================

var (
	ErrOverrideNotFound   = errors.New("override request not found")
	ErrAckNotFound        = errors.New("acknowledgment not found")
	ErrEscalationNotFound = errors.New("escalation not found")
	ErrAlreadyProcessed   = errors.New("request already processed")
	ErrInvalidStatus      = errors.New("invalid status transition")
	ErrExcessiveOverrides = errors.New("excessive override pattern detected")
)

// =============================================================================
// OVERRIDE STORE
// =============================================================================

// OverrideStore manages overrides, acknowledgments, and escalations
// In production, this would be backed by a database (PostgreSQL)
type OverrideStore struct {
	overrides       map[string]*types.OverrideRequest
	acknowledgments map[string]*types.Acknowledgment
	escalations     map[string]*types.Escalation
	patterns        map[string]*types.OverridePattern // keyed by requestorID:ruleCode

	mu sync.RWMutex

	// Configuration
	overrideExpireHours int
	patternThreshold24h int
	patternThreshold7d  int
}

// NewOverrideStore creates a new override store
func NewOverrideStore() *OverrideStore {
	return &OverrideStore{
		overrides:           make(map[string]*types.OverrideRequest),
		acknowledgments:     make(map[string]*types.Acknowledgment),
		escalations:         make(map[string]*types.Escalation),
		patterns:            make(map[string]*types.OverridePattern),
		overrideExpireHours: 24,
		patternThreshold24h: 5,
		patternThreshold7d:  20,
	}
}

// =============================================================================
// OVERRIDE OPERATIONS
// =============================================================================

// RequestOverride creates a new override request
func (s *OverrideStore) RequestOverride(ctx context.Context, req *types.OverrideRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate ID if not provided
	if req.ID == "" {
		req.ID = types.NewUUID()
	}
	req.Status = types.OverrideStatusPending
	req.RequestedAt = time.Now()
	req.ExpiresAt = time.Now().Add(time.Duration(s.overrideExpireHours) * time.Hour)

	// Check pattern before allowing request
	patternKey := req.RequestorID + ":" + req.RuleCode
	if err := s.checkPatternLocked(patternKey); err != nil {
		return err
	}

	s.overrides[req.ID] = req

	// Update pattern tracking
	s.updatePatternLocked(req.RequestorID, req.RuleCode)

	return nil
}

// ApproveOverride approves a pending override request
func (s *OverrideStore) ApproveOverride(ctx context.Context, overrideID, approverID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	override, ok := s.overrides[overrideID]
	if !ok {
		return ErrOverrideNotFound
	}

	if override.Status != types.OverrideStatusPending {
		return ErrAlreadyProcessed
	}

	override.Status = types.OverrideStatusApproved
	override.ApprovedBy = approverID
	override.ApprovedAt = time.Now()

	return nil
}

// DenyOverride denies a pending override request
func (s *OverrideStore) DenyOverride(ctx context.Context, overrideID, denierID, denialReason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	override, ok := s.overrides[overrideID]
	if !ok {
		return ErrOverrideNotFound
	}

	if override.Status != types.OverrideStatusPending {
		return ErrAlreadyProcessed
	}

	override.Status = types.OverrideStatusDenied
	override.DeniedBy = denierID
	override.DeniedAt = time.Now()
	override.DenialReason = denialReason

	return nil
}

// GetOverride retrieves an override by ID
func (s *OverrideStore) GetOverride(ctx context.Context, overrideID string) (*types.OverrideRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	override, ok := s.overrides[overrideID]
	if !ok {
		return nil, ErrOverrideNotFound
	}
	return override, nil
}

// ListOverrides returns all overrides
func (s *OverrideStore) ListOverrides(ctx context.Context) []*types.OverrideRequest {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*types.OverrideRequest, 0, len(s.overrides))
	for _, override := range s.overrides {
		result = append(result, override)
	}
	return result
}

// =============================================================================
// ACKNOWLEDGMENT OPERATIONS
// =============================================================================

// RecordAcknowledgment records a warning acknowledgment
func (s *OverrideStore) RecordAcknowledgment(ctx context.Context, ack *types.Acknowledgment) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ack.ID == "" {
		ack.ID = types.NewUUID()
	}
	if ack.Timestamp.IsZero() {
		ack.Timestamp = time.Now()
	}

	s.acknowledgments[ack.ID] = ack
	return nil
}

// GetAcknowledgment retrieves an acknowledgment by ID
func (s *OverrideStore) GetAcknowledgment(ctx context.Context, ackID string) (*types.Acknowledgment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ack, ok := s.acknowledgments[ackID]
	if !ok {
		return nil, ErrAckNotFound
	}
	return ack, nil
}

// ListAcknowledgments returns all acknowledgments
func (s *OverrideStore) ListAcknowledgments(ctx context.Context) []*types.Acknowledgment {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*types.Acknowledgment, 0, len(s.acknowledgments))
	for _, ack := range s.acknowledgments {
		result = append(result, ack)
	}
	return result
}

// =============================================================================
// ESCALATION OPERATIONS
// =============================================================================

// CreateEscalation creates a new escalation
func (s *OverrideStore) CreateEscalation(ctx context.Context, esc *types.Escalation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if esc.ID == "" {
		esc.ID = types.NewUUID()
	}
	esc.Status = types.EscalationStatusOpen
	esc.CurrentLevel = 1
	esc.CreatedAt = time.Now()

	s.escalations[esc.ID] = esc
	return nil
}

// AcknowledgeEscalation marks an escalation as acknowledged
func (s *OverrideStore) AcknowledgeEscalation(ctx context.Context, escalationID, acknowledgedBy string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	esc, ok := s.escalations[escalationID]
	if !ok {
		return ErrEscalationNotFound
	}

	if esc.Status != types.EscalationStatusOpen {
		return ErrInvalidStatus
	}

	esc.Status = types.EscalationStatusAcknowledged
	esc.AcknowledgedBy = acknowledgedBy
	esc.AcknowledgedAt = time.Now()

	return nil
}

// ResolveEscalation resolves an escalation
func (s *OverrideStore) ResolveEscalation(ctx context.Context, escalationID, resolvedBy, resolution string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	esc, ok := s.escalations[escalationID]
	if !ok {
		return ErrEscalationNotFound
	}

	if esc.Status == types.EscalationStatusClosed || esc.Status == types.EscalationStatusResolved {
		return ErrAlreadyProcessed
	}

	esc.Status = types.EscalationStatusResolved
	esc.ResolvedBy = resolvedBy
	esc.ResolvedAt = time.Now()
	esc.Resolution = resolution

	return nil
}

// GetEscalation retrieves an escalation by ID
func (s *OverrideStore) GetEscalation(ctx context.Context, escalationID string) (*types.Escalation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	esc, ok := s.escalations[escalationID]
	if !ok {
		return nil, ErrEscalationNotFound
	}
	return esc, nil
}

// ListEscalations returns all escalations
func (s *OverrideStore) ListEscalations(ctx context.Context) []*types.Escalation {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*types.Escalation, 0, len(s.escalations))
	for _, esc := range s.escalations {
		result = append(result, esc)
	}
	return result
}

// =============================================================================
// PATTERN MONITORING
// =============================================================================

// checkPatternLocked checks if user has excessive overrides (must hold lock)
func (s *OverrideStore) checkPatternLocked(patternKey string) error {
	pattern, ok := s.patterns[patternKey]
	if !ok {
		return nil // No pattern yet - OK
	}

	if pattern.Flagged {
		return ErrExcessiveOverrides
	}

	return nil
}

// updatePatternLocked updates pattern tracking for a requestor/rule combination (must hold lock)
func (s *OverrideStore) updatePatternLocked(requestorID, ruleCode string) {
	patternKey := requestorID + ":" + ruleCode

	pattern, ok := s.patterns[patternKey]
	if !ok {
		pattern = &types.OverridePattern{
			RequestorID: requestorID,
			RuleCode:    ruleCode,
			LastRequest: time.Now(),
		}
		s.patterns[patternKey] = pattern
	}

	// Count overrides in last 24 hours and 7 days
	count24h := 0
	count7d := 0
	cutoff24h := time.Now().Add(-24 * time.Hour)
	cutoff7d := time.Now().Add(-7 * 24 * time.Hour)

	for _, override := range s.overrides {
		if override.RequestorID != requestorID || override.RuleCode != ruleCode {
			continue
		}
		if override.RequestedAt.After(cutoff24h) {
			count24h++
		}
		if override.RequestedAt.After(cutoff7d) {
			count7d++
		}
	}

	pattern.Count24h = count24h
	pattern.Count7d = count7d
	pattern.LastRequest = time.Now()

	// Check thresholds
	if count24h >= s.patternThreshold24h {
		pattern.Flagged = true
		pattern.FlagReason = "Excessive overrides in 24 hours"
	} else if count7d >= s.patternThreshold7d {
		pattern.Flagged = true
		pattern.FlagReason = "Excessive overrides in 7 days"
	}
}

// GetPatternAnalysis returns all override patterns for analysis
func (s *OverrideStore) GetPatternAnalysis(ctx context.Context) map[string]*types.OverridePattern {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[string]*types.OverridePattern)
	for key, pattern := range s.patterns {
		result[key] = pattern
	}
	return result
}

// =============================================================================
// CLEANUP & MAINTENANCE
// =============================================================================

// CleanupExpired removes expired override requests
func (s *OverrideStore) CleanupExpired(ctx context.Context) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	count := 0

	for _, override := range s.overrides {
		if override.Status == types.OverrideStatusPending && now.After(override.ExpiresAt) {
			override.Status = types.OverrideStatusExpired
			count++
		}
	}

	return count, nil
}

// GetStats returns override store statistics
func (s *OverrideStore) GetStats(ctx context.Context) map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pendingOverrides := 0
	approvedOverrides := 0
	deniedOverrides := 0

	for _, o := range s.overrides {
		switch o.Status {
		case types.OverrideStatusPending:
			pendingOverrides++
		case types.OverrideStatusApproved:
			approvedOverrides++
		case types.OverrideStatusDenied:
			deniedOverrides++
		}
	}

	openEscalations := 0
	for _, e := range s.escalations {
		if e.Status == types.EscalationStatusOpen {
			openEscalations++
		}
	}

	flaggedPatterns := 0
	for _, p := range s.patterns {
		if p.Flagged {
			flaggedPatterns++
		}
	}

	return map[string]interface{}{
		"total_overrides":    len(s.overrides),
		"pending_overrides":  pendingOverrides,
		"approved_overrides": approvedOverrides,
		"denied_overrides":   deniedOverrides,
		"total_acks":         len(s.acknowledgments),
		"total_escalations":  len(s.escalations),
		"open_escalations":   openEscalations,
		"flagged_patterns":   flaggedPatterns,
	}
}
