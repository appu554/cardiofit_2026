package services

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/models"
)

// SCEService wraps SafetyEngine for independent deployment (CC-1).
// In the current deployment, it runs in-process with KB-22.
// Future: separate process on port 8201 with independent health checks.
//
// Architecture (CC-1 decision):
//   - SCE runs as sidecar on same host to minimize network latency (~1-2ms)
//   - KB-19 routes answers to BOTH M2 (BayesianEngine) and SCE in parallel
//   - If SCE returns ESCALATE, KB-19 overrides M2's response
//   - Independent health-check circuit breaker in KB-19
type SCEService struct {
	safetyEngine *SafetyEngine
	nodeLoader   *NodeLoader
	publisher    KafkaPublisher
	log          *zap.Logger

	// Per-session answer accumulation for stateful evaluation.
	// Key: session_id string, Value: map[question_id]answer
	sessions   map[string]map[string]string
	sessionsMu sync.RWMutex
}

// SCEResult is the safety evaluation outcome.
type SCEResult struct {
	Clear              bool               `json:"clear"`
	Flags              []models.SafetyFlag `json:"flags,omitempty"`
	EscalationRequired bool               `json:"escalation_required"`
	ReasonCode         string             `json:"reason_code,omitempty"`
}

// NewSCEService creates the Safety Constraint Engine service.
func NewSCEService(
	safetyEngine *SafetyEngine,
	nodeLoader *NodeLoader,
	publisher KafkaPublisher,
	log *zap.Logger,
) *SCEService {
	return &SCEService{
		safetyEngine: safetyEngine,
		nodeLoader:   nodeLoader,
		publisher:    publisher,
		log:          log,
		sessions:     make(map[string]map[string]string),
	}
}

// EvaluateAnswer processes a single answer through the safety engine,
// independently of the Bayesian inference loop. This is the core SCE
// function — it runs in parallel with M2 and can veto M2's output.
func (s *SCEService) EvaluateAnswer(
	ctx context.Context,
	sessionID uuid.UUID,
	nodeID string,
	questionID string,
	answer string,
	firedCMs map[string]bool,
) (*SCEResult, error) {
	node := s.nodeLoader.Get(nodeID)
	if node == nil {
		return &SCEResult{Clear: true}, nil
	}

	// Accumulate answers for this session
	sid := sessionID.String()
	s.sessionsMu.Lock()
	if s.sessions[sid] == nil {
		s.sessions[sid] = make(map[string]string)
	}
	s.sessions[sid][questionID] = answer
	answers := make(map[string]string, len(s.sessions[sid]))
	for k, v := range s.sessions[sid] {
		answers[k] = v
	}
	s.sessionsMu.Unlock()

	// Evaluate all safety triggers (CM-aware via G8)
	var flags []models.SafetyFlag
	if firedCMs != nil {
		flags = s.safetyEngine.EvaluateTriggersWithCMs(node.SafetyTriggers, answers, firedCMs)
	} else {
		flags = s.safetyEngine.EvaluateTriggers(node.SafetyTriggers, answers)
	}

	result := &SCEResult{
		Clear: len(flags) == 0,
		Flags: flags,
	}

	// Check for IMMEDIATE severity — requires escalation
	for _, flag := range flags {
		if flag.Severity == models.SafetyImmediate {
			result.EscalationRequired = true
			result.ReasonCode = flag.FlagID
			break
		}
	}

	// Publish escalation event if needed (BAY-11)
	if result.EscalationRequired && s.publisher != nil {
		event := KafkaEscalationEvent{
			EventType: KafkaEventRedFlag,
			SessionID: sessionID,
			FlagID:    result.ReasonCode,
			Severity:  string(models.SafetyImmediate),
		}
		if err := s.publisher.Publish(ctx, TopicEscalationEvents, sessionID.String(), event); err != nil {
			s.log.Error("CC-1: failed to publish SCE escalation",
				zap.String("session_id", sid),
				zap.Error(err),
			)
		}
	}

	if result.EscalationRequired {
		s.log.Warn("CC-1: SCE escalation triggered",
			zap.String("session_id", sid),
			zap.String("reason_code", result.ReasonCode),
			zap.Int("flag_count", len(flags)),
		)
	}

	return result, nil
}

// ClearSession removes accumulated answer state for a completed/abandoned session.
func (s *SCEService) ClearSession(sessionID uuid.UUID) {
	s.sessionsMu.Lock()
	delete(s.sessions, sessionID.String())
	s.sessionsMu.Unlock()
}

// Health returns nil if the SCE is operational.
func (s *SCEService) Health() error {
	return nil
}
