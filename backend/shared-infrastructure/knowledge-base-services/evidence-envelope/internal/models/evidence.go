package models

import (
	"time"
	"encoding/json"
	"github.com/google/uuid"
)

// EvidenceTransaction represents a complete audit trail for a system transaction
type EvidenceTransaction struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	TransactionID   string          `json:"transaction_id" db:"transaction_id"`
	UserID          *string         `json:"user_id,omitempty" db:"user_id"`
	SessionID       *string         `json:"session_id,omitempty" db:"session_id"`
	SourceService   string          `json:"source_service" db:"source_service"`
	TargetService   *string         `json:"target_service,omitempty" db:"target_service"`
	OperationType   string          `json:"operation_type" db:"operation_type"`
	GraphQLOperation *string        `json:"graphql_operation,omitempty" db:"graphql_operation"`
	RequestPayload  json.RawMessage `json:"request_payload,omitempty" db:"request_payload"`
	ResponsePayload json.RawMessage `json:"response_payload,omitempty" db:"response_payload"`
	HTTPStatus      *int            `json:"http_status,omitempty" db:"http_status"`
	ProcessingTimeMS *int           `json:"processing_time_ms,omitempty" db:"processing_time_ms"`
	Timestamp       time.Time       `json:"timestamp" db:"timestamp"`
	CorrelationID   *string         `json:"correlation_id,omitempty" db:"correlation_id"`
	TraceID         *string         `json:"trace_id,omitempty" db:"trace_id"`
	SpanID          *string         `json:"span_id,omitempty" db:"span_id"`
}

// DataLineage tracks data transformations and origins
type DataLineage struct {
	ID                  uuid.UUID       `json:"id" db:"id"`
	TransactionID       string          `json:"transaction_id" db:"transaction_id"`
	SourceSystem        string          `json:"source_system" db:"source_system"`
	SourceEntity        string          `json:"source_entity" db:"source_entity"`
	SourceID            string          `json:"source_id" db:"source_id"`
	TargetSystem        string          `json:"target_system" db:"target_system"`
	TargetEntity        string          `json:"target_entity" db:"target_entity"`
	TargetID            *string         `json:"target_id,omitempty" db:"target_id"`
	TransformationType  *string         `json:"transformation_type,omitempty" db:"transformation_type"`
	TransformationRules json.RawMessage `json:"transformation_rules,omitempty" db:"transformation_rules"`
	DataQualityScore    *float64        `json:"data_quality_score,omitempty" db:"data_quality_score"`
	CreatedAt           time.Time       `json:"created_at" db:"created_at"`
}

// ClinicalDecision represents audited clinical reasoning decisions
type ClinicalDecision struct {
	ID               uuid.UUID       `json:"id" db:"id"`
	TransactionID    string          `json:"transaction_id" db:"transaction_id"`
	DecisionID       string          `json:"decision_id" db:"decision_id"`
	PatientID        *string         `json:"patient_id,omitempty" db:"patient_id"`
	DecisionType     string          `json:"decision_type" db:"decision_type"`
	KnowledgeSource  string          `json:"knowledge_source" db:"knowledge_source"`
	InputData        json.RawMessage `json:"input_data" db:"input_data"`
	DecisionOutcome  json.RawMessage `json:"decision_outcome" db:"decision_outcome"`
	ConfidenceScore  *float64        `json:"confidence_score,omitempty" db:"confidence_score"`
	EvidenceSources  json.RawMessage `json:"evidence_sources,omitempty" db:"evidence_sources"`
	OverriddenBy     *string         `json:"overridden_by,omitempty" db:"overridden_by"`
	OverrideReason   *string         `json:"override_reason,omitempty" db:"override_reason"`
	CreatedAt        time.Time       `json:"created_at" db:"created_at"`
	ExpiresAt        *time.Time      `json:"expires_at,omitempty" db:"expires_at"`
}

// KBVersion tracks knowledge base versions and deployments
type KBVersion struct {
	ID                    uuid.UUID       `json:"id" db:"id"`
	KBService             string          `json:"kb_service" db:"kb_service"`
	Version               string          `json:"version" db:"version"`
	SchemaVersion         string          `json:"schema_version" db:"schema_version"`
	DataSources           json.RawMessage `json:"data_sources" db:"data_sources"`
	DeploymentTimestamp   time.Time       `json:"deployment_timestamp" db:"deployment_timestamp"`
	ValidationStatus      string          `json:"validation_status" db:"validation_status"`
	ValidationResults     json.RawMessage `json:"validation_results,omitempty" db:"validation_results"`
	IsActive              bool            `json:"is_active" db:"is_active"`
	DeactivatedAt         *time.Time      `json:"deactivated_at,omitempty" db:"deactivated_at"`
}

// DataProvenance tracks the origin and quality of clinical data
type DataProvenance struct {
	ID                   uuid.UUID       `json:"id" db:"id"`
	EntityType           string          `json:"entity_type" db:"entity_type"`
	EntityID             string          `json:"entity_id" db:"entity_id"`
	SourceSystem         string          `json:"source_system" db:"source_system"`
	SourceTimestamp      *time.Time      `json:"source_timestamp,omitempty" db:"source_timestamp"`
	IngestionTimestamp   time.Time       `json:"ingestion_timestamp" db:"ingestion_timestamp"`
	DataQualityFlags     json.RawMessage `json:"data_quality_flags,omitempty" db:"data_quality_flags"`
	ValidationStatus     string          `json:"validation_status" db:"validation_status"`
	RetentionPolicy      *string         `json:"retention_policy,omitempty" db:"retention_policy"`
	GDPRConsentStatus    *string         `json:"gdpr_consent_status,omitempty" db:"gdpr_consent_status"`
}

// SystemMetric represents performance and operational metrics
type SystemMetric struct {
	ID          uuid.UUID       `json:"id" db:"id"`
	ServiceName string          `json:"service_name" db:"service_name"`
	MetricName  string          `json:"metric_name" db:"metric_name"`
	MetricValue float64         `json:"metric_value" db:"metric_value"`
	MetricUnit  *string         `json:"metric_unit,omitempty" db:"metric_unit"`
	Tags        json.RawMessage `json:"tags,omitempty" db:"tags"`
	Timestamp   time.Time       `json:"timestamp" db:"timestamp"`
}

// TransactionRequest represents the request to create a new transaction
type TransactionRequest struct {
	UserID           *string         `json:"user_id,omitempty"`
	SessionID        *string         `json:"session_id,omitempty"`
	SourceService    string          `json:"source_service"`
	TargetService    *string         `json:"target_service,omitempty"`
	OperationType    string          `json:"operation_type"`
	GraphQLOperation *string         `json:"graphql_operation,omitempty"`
	RequestPayload   json.RawMessage `json:"request_payload,omitempty"`
	CorrelationID    *string         `json:"correlation_id,omitempty"`
}

// TransactionResponse represents the response when creating/updating a transaction
type TransactionResponse struct {
	TransactionID string    `json:"transaction_id"`
	CreatedAt     time.Time `json:"created_at"`
}

// AuditQuery represents query parameters for audit trail searches
type AuditQuery struct {
	UserID        *string    `json:"user_id,omitempty"`
	Service       *string    `json:"service,omitempty"`
	OperationType *string    `json:"operation_type,omitempty"`
	PatientID     *string    `json:"patient_id,omitempty"`
	StartTime     *time.Time `json:"start_time,omitempty"`
	EndTime       *time.Time `json:"end_time,omitempty"`
	Limit         int        `json:"limit"`
	Offset        int        `json:"offset"`
}