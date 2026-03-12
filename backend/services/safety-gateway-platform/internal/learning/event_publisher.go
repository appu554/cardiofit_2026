package learning

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// LearningEventPublisher publishes learning events for clinical decision analysis
type LearningEventPublisher struct {
	kafkaProducer KafkaProducer // Interface for Kafka producer
	logger        *logger.Logger
	config        *PublisherConfig
}

// PublisherConfig contains configuration for the learning event publisher
type PublisherConfig struct {
	TopicPrefix              string        `yaml:"topic_prefix"`
	EnableEventPublishing    bool          `yaml:"enable_event_publishing"`
	BatchSize               int           `yaml:"batch_size"`
	FlushInterval           time.Duration `yaml:"flush_interval"`
	RetryAttempts           int           `yaml:"retry_attempts"`
	EnableOutcomeCorrelation bool         `yaml:"enable_outcome_correlation"`
}

// KafkaProducer interface for Kafka operations (to be implemented by kafka_integration.go)
type KafkaProducer interface {
	Produce(topic string, message []byte) error
	ProduceBatch(topic string, messages [][]byte) error
	Close() error
}

// NewLearningEventPublisher creates a new learning event publisher
func NewLearningEventPublisher(
	kafkaProducer KafkaProducer,
	config *PublisherConfig,
	logger *logger.Logger,
) *LearningEventPublisher {
	if config == nil {
		config = &PublisherConfig{
			TopicPrefix:              "clinical-learning",
			EnableEventPublishing:    true,
			BatchSize:               100,
			FlushInterval:           5 * time.Second,
			RetryAttempts:           3,
			EnableOutcomeCorrelation: true,
		}
	}

	return &LearningEventPublisher{
		kafkaProducer: kafkaProducer,
		logger:        logger,
		config:        config,
	}
}

// PublishSafetyDecisionEvent publishes a safety decision event for learning analysis
func (p *LearningEventPublisher) PublishSafetyDecisionEvent(
	ctx context.Context,
	req *types.SafetyRequest,
	response *types.SafetyResponse,
	snapshot *types.ClinicalSnapshot,
) error {
	if !p.config.EnableEventPublishing {
		p.logger.Debug("Event publishing is disabled, skipping safety decision event")
		return nil
	}

	event := &SafetyDecisionEvent{
		EventID:     p.generateEventID(),
		EventType:   "safety_decision",
		Timestamp:   time.Now(),
		RequestID:   req.RequestID,
		PatientID:   req.PatientID,
		ClinicianID: req.ClinicianID,
		ActionType:  req.ActionType,
		Priority:    req.Priority,
		Decision: DecisionInfo{
			Status:             string(response.Status),
			RiskScore:          response.RiskScore,
			CriticalViolations: response.CriticalViolations,
			Warnings:           response.Warnings,
			ProcessingTime:     response.ProcessingTime,
			EngineResults:      p.convertEngineResults(response.EngineResults),
			EnginesFailed:      response.EnginesFailed,
		},
		ClinicalContext: ClinicalContextInfo{
			SnapshotID:       snapshot.SnapshotID,
			DataCompleteness: snapshot.DataCompleteness,
			PatientAge:       p.extractPatientAge(snapshot),
			ActiveMedications: len(snapshot.Data.ActiveMedications),
			ActiveConditions: len(snapshot.Data.Conditions),
			RecentVitals:     len(snapshot.Data.RecentVitals),
		},
		Metadata: map[string]interface{}{
			"context_version": response.ContextVersion,
			"data_sources":    snapshot.Data.DataSources,
			"snapshot_version": snapshot.Version,
			"request_source": req.Source,
		},
	}

	return p.publishEvent("safety-decisions", event)
}

// PublishOverrideEvent publishes an override event for learning analysis
func (p *LearningEventPublisher) PublishOverrideEvent(
	ctx context.Context,
	token *types.EnhancedOverrideToken,
	validation *types.OverrideValidation,
) error {
	if !p.config.EnableEventPublishing {
		return nil
	}

	event := &OverrideEvent{
		EventID:   p.generateEventID(),
		EventType: "clinical_override",
		Timestamp: time.Now(),
		TokenInfo: OverrideTokenInfo{
			TokenID:       token.TokenID,
			RequestID:     token.RequestID,
			PatientID:     token.PatientID,
			RequiredLevel: string(token.RequiredLevel),
			CreatedAt:     token.CreatedAt,
			ExpiresAt:     token.ExpiresAt,
		},
		ValidationInfo: OverrideValidationInfo{
			Valid:       validation.Valid,
			ClinicianID: validation.ClinicianID,
			ValidatedAt: validation.ValidatedAt,
			Reason:      validation.Reason,
		},
		OriginalDecision: DecisionInfo{
			Status:             string(token.DecisionSummary.Status),
			RiskScore:          token.DecisionSummary.RiskScore,
			CriticalViolations: token.DecisionSummary.CriticalViolations,
			Explanation:        token.DecisionSummary.Explanation,
		},
		ReproducibilityInfo: ReproducibilityInfo{
			ProposalID:     token.ReproducibilityPackage.ProposalID,
			EngineVersions: token.ReproducibilityPackage.EngineVersions,
			RuleVersions:   token.ReproducibilityPackage.RuleVersions,
			DataSources:    token.ReproducibilityPackage.DataSources,
		},
		SnapshotInfo: SnapshotInfo{
			SnapshotID:       token.SnapshotReference.SnapshotID,
			Checksum:         token.SnapshotReference.Checksum,
			CreatedAt:        token.SnapshotReference.CreatedAt,
			DataCompleteness: token.SnapshotReference.DataCompleteness,
		},
	}

	return p.publishEvent("clinical-overrides", event)
}

// PublishOutcomeEvent publishes a clinical outcome event for correlation analysis
func (p *LearningEventPublisher) PublishOutcomeEvent(
	ctx context.Context,
	outcome *ClinicalOutcomeEvent,
) error {
	if !p.config.EnableEventPublishing || !p.config.EnableOutcomeCorrelation {
		return nil
	}

	outcome.EventID = p.generateEventID()
	outcome.EventType = "clinical_outcome"
	outcome.Timestamp = time.Now()

	return p.publishEvent("clinical-outcomes", outcome)
}

// PublishPerformanceEvent publishes a performance analysis event
func (p *LearningEventPublisher) PublishPerformanceEvent(
	ctx context.Context,
	performance *PerformanceAnalysisEvent,
) error {
	if !p.config.EnableEventPublishing {
		return nil
	}

	performance.EventID = p.generateEventID()
	performance.EventType = "performance_analysis"
	performance.Timestamp = time.Now()

	return p.publishEvent("performance-analysis", performance)
}

// publishEvent publishes an event to the specified Kafka topic
func (p *LearningEventPublisher) publishEvent(topicSuffix string, event interface{}) error {
	topicName := fmt.Sprintf("%s-%s", p.config.TopicPrefix, topicSuffix)
	
	eventBytes, err := json.Marshal(event)
	if err != nil {
		p.logger.Error("Failed to marshal event",
			zap.String("topic", topicName),
			zap.Error(err),
		)
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Retry logic
	var lastErr error
	for attempt := 0; attempt < p.config.RetryAttempts; attempt++ {
		err = p.kafkaProducer.Produce(topicName, eventBytes)
		if err == nil {
			p.logger.Debug("Event published successfully",
				zap.String("topic", topicName),
				zap.Int("attempt", attempt+1),
			)
			return nil
		}
		
		lastErr = err
		p.logger.Warn("Failed to publish event, retrying",
			zap.String("topic", topicName),
			zap.Int("attempt", attempt+1),
			zap.Int("max_attempts", p.config.RetryAttempts),
			zap.Error(err),
		)
		
		// Exponential backoff
		time.Sleep(time.Duration(attempt+1) * time.Second)
	}

	p.logger.Error("Failed to publish event after all retries",
		zap.String("topic", topicName),
		zap.Int("attempts", p.config.RetryAttempts),
		zap.Error(lastErr),
	)
	
	return fmt.Errorf("failed to publish event after %d attempts: %w", p.config.RetryAttempts, lastErr)
}

// convertEngineResults converts engine results to learning event format
func (p *LearningEventPublisher) convertEngineResults(results []types.EngineResult) []EngineResultInfo {
	converted := make([]EngineResultInfo, len(results))
	for i, result := range results {
		converted[i] = EngineResultInfo{
			EngineID:   result.EngineID,
			EngineName: result.EngineName,
			Status:     string(result.Status),
			RiskScore:  result.RiskScore,
			Violations: result.Violations,
			Warnings:   result.Warnings,
			Confidence: result.Confidence,
			Duration:   result.Duration,
			Tier:       int(result.Tier),
			Error:      result.Error,
		}
	}
	return converted
}

// extractPatientAge extracts patient age from snapshot data
func (p *LearningEventPublisher) extractPatientAge(snapshot *types.ClinicalSnapshot) int {
	if snapshot.Data != nil && snapshot.Data.Demographics != nil {
		return snapshot.Data.Demographics.Age
	}
	return 0
}

// generateEventID generates a unique event ID
func (p *LearningEventPublisher) generateEventID() string {
	return fmt.Sprintf("event_%d_%d", time.Now().UnixNano(), time.Now().Unix())
}

// GetPublisherMetrics returns publisher metrics
func (p *LearningEventPublisher) GetPublisherMetrics() map[string]interface{} {
	return map[string]interface{}{
		"publisher_version":         "1.0.0",
		"event_publishing_enabled":  p.config.EnableEventPublishing,
		"outcome_correlation_enabled": p.config.EnableOutcomeCorrelation,
		"topic_prefix":              p.config.TopicPrefix,
		"batch_size":               p.config.BatchSize,
		"flush_interval":           p.config.FlushInterval.String(),
		"retry_attempts":           p.config.RetryAttempts,
	}
}

// Close closes the event publisher
func (p *LearningEventPublisher) Close() error {
	if p.kafkaProducer != nil {
		return p.kafkaProducer.Close()
	}
	return nil
}

// Event type definitions for learning analysis

// SafetyDecisionEvent represents a safety decision event
type SafetyDecisionEvent struct {
	EventID         string              `json:"event_id"`
	EventType       string              `json:"event_type"`
	Timestamp       time.Time           `json:"timestamp"`
	RequestID       string              `json:"request_id"`
	PatientID       string              `json:"patient_id"`
	ClinicianID     string              `json:"clinician_id"`
	ActionType      string              `json:"action_type"`
	Priority        string              `json:"priority"`
	Decision        DecisionInfo        `json:"decision"`
	ClinicalContext ClinicalContextInfo `json:"clinical_context"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// OverrideEvent represents a clinical override event
type OverrideEvent struct {
	EventID             string                   `json:"event_id"`
	EventType           string                   `json:"event_type"`
	Timestamp           time.Time                `json:"timestamp"`
	TokenInfo           OverrideTokenInfo        `json:"token_info"`
	ValidationInfo      OverrideValidationInfo   `json:"validation_info"`
	OriginalDecision    DecisionInfo            `json:"original_decision"`
	ReproducibilityInfo ReproducibilityInfo     `json:"reproducibility_info"`
	SnapshotInfo        SnapshotInfo            `json:"snapshot_info"`
}

// ClinicalOutcomeEvent represents a clinical outcome for correlation analysis
type ClinicalOutcomeEvent struct {
	EventID           string                 `json:"event_id"`
	EventType         string                 `json:"event_type"`
	Timestamp         time.Time              `json:"timestamp"`
	PatientID         string                 `json:"patient_id"`
	OutcomeType       string                 `json:"outcome_type"`
	OutcomeValue      string                 `json:"outcome_value"`
	OutcomeSeverity   string                 `json:"outcome_severity"`
	RelatedRequestID  string                 `json:"related_request_id,omitempty"`
	RelatedTokenID    string                 `json:"related_token_id,omitempty"`
	TimeToOutcome     time.Duration          `json:"time_to_outcome"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

// PerformanceAnalysisEvent represents a performance analysis event
type PerformanceAnalysisEvent struct {
	EventID         string                 `json:"event_id"`
	EventType       string                 `json:"event_type"`
	Timestamp       time.Time              `json:"timestamp"`
	AnalysisType    string                 `json:"analysis_type"`
	TimeWindow      string                 `json:"time_window"`
	Metrics         map[string]float64     `json:"metrics"`
	Recommendations []string               `json:"recommendations"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// Supporting info structures
type DecisionInfo struct {
	Status             string               `json:"status"`
	RiskScore          float64              `json:"risk_score"`
	CriticalViolations []string             `json:"critical_violations"`
	Warnings           []string             `json:"warnings"`
	ProcessingTime     time.Duration        `json:"processing_time"`
	EngineResults      []EngineResultInfo   `json:"engine_results"`
	EnginesFailed      []string             `json:"engines_failed"`
	Explanation        string               `json:"explanation,omitempty"`
}

type ClinicalContextInfo struct {
	SnapshotID        string  `json:"snapshot_id"`
	DataCompleteness  float64 `json:"data_completeness"`
	PatientAge        int     `json:"patient_age"`
	ActiveMedications int     `json:"active_medications"`
	ActiveConditions  int     `json:"active_conditions"`
	RecentVitals      int     `json:"recent_vitals"`
}

type OverrideTokenInfo struct {
	TokenID       string    `json:"token_id"`
	RequestID     string    `json:"request_id"`
	PatientID     string    `json:"patient_id"`
	RequiredLevel string    `json:"required_level"`
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"`
}

type OverrideValidationInfo struct {
	Valid       bool      `json:"valid"`
	ClinicianID string    `json:"clinician_id"`
	ValidatedAt time.Time `json:"validated_at"`
	Reason      string    `json:"reason,omitempty"`
}

type ReproducibilityInfo struct {
	ProposalID     string            `json:"proposal_id"`
	EngineVersions map[string]string `json:"engine_versions"`
	RuleVersions   map[string]string `json:"rule_versions"`
	DataSources    []string          `json:"data_sources"`
}

type SnapshotInfo struct {
	SnapshotID       string    `json:"snapshot_id"`
	Checksum         string    `json:"checksum"`
	CreatedAt        time.Time `json:"created_at"`
	DataCompleteness float64   `json:"data_completeness"`
}

type EngineResultInfo struct {
	EngineID   string        `json:"engine_id"`
	EngineName string        `json:"engine_name"`
	Status     string        `json:"status"`
	RiskScore  float64       `json:"risk_score"`
	Violations []string      `json:"violations"`
	Warnings   []string      `json:"warnings"`
	Confidence float64       `json:"confidence"`
	Duration   time.Duration `json:"duration"`
	Tier       int           `json:"tier"`
	Error      string        `json:"error,omitempty"`
}