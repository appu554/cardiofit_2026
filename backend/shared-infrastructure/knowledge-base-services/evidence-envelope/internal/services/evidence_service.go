package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
	"context"
	"crypto/rand"
	"encoding/hex"

	"evidence-envelope/internal/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type EvidenceService struct {
	db     *sql.DB
	logger *zap.Logger
}

func NewEvidenceService(db *sql.DB, logger *zap.Logger) *EvidenceService {
	return &EvidenceService{
		db:     db,
		logger: logger,
	}
}

// GenerateTransactionID creates a unique transaction identifier
func (s *EvidenceService) GenerateTransactionID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("txn_%s_%d", hex.EncodeToString(bytes)[:16], time.Now().UnixNano())
}

// CreateTransaction creates a new evidence transaction
func (s *EvidenceService) CreateTransaction(ctx context.Context, req models.TransactionRequest) (*models.TransactionResponse, error) {
	transactionID := s.GenerateTransactionID()
	
	query := `
		INSERT INTO evidence_transactions (
			transaction_id, user_id, session_id, source_service, target_service,
			operation_type, graphql_operation, request_payload, correlation_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING timestamp`
	
	var timestamp time.Time
	err := s.db.QueryRowContext(ctx, query,
		transactionID, req.UserID, req.SessionID, req.SourceService, req.TargetService,
		req.OperationType, req.GraphQLOperation, req.RequestPayload, req.CorrelationID,
	).Scan(&timestamp)
	
	if err != nil {
		s.logger.Error("Failed to create transaction", zap.Error(err))
		return nil, fmt.Errorf("failed to create transaction: %w", err)
	}

	return &models.TransactionResponse{
		TransactionID: transactionID,
		CreatedAt:     timestamp,
	}, nil
}

// CompleteTransaction updates a transaction with response data and performance metrics
func (s *EvidenceService) CompleteTransaction(ctx context.Context, transactionID string, responsePayload json.RawMessage, httpStatus int, processingTimeMS int) error {
	query := `
		UPDATE evidence_transactions 
		SET response_payload = $2, http_status = $3, processing_time_ms = $4
		WHERE transaction_id = $1`
	
	_, err := s.db.ExecContext(ctx, query, transactionID, responsePayload, httpStatus, processingTimeMS)
	if err != nil {
		s.logger.Error("Failed to complete transaction", zap.String("transaction_id", transactionID), zap.Error(err))
		return fmt.Errorf("failed to complete transaction: %w", err)
	}
	
	return nil
}

// RecordDataLineage tracks data transformation and lineage
func (s *EvidenceService) RecordDataLineage(ctx context.Context, lineage models.DataLineage) error {
	query := `
		INSERT INTO data_lineage (
			transaction_id, source_system, source_entity, source_id,
			target_system, target_entity, target_id, transformation_type,
			transformation_rules, data_quality_score
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	
	_, err := s.db.ExecContext(ctx, query,
		lineage.TransactionID, lineage.SourceSystem, lineage.SourceEntity, lineage.SourceID,
		lineage.TargetSystem, lineage.TargetEntity, lineage.TargetID, lineage.TransformationType,
		lineage.TransformationRules, lineage.DataQualityScore,
	)
	
	if err != nil {
		s.logger.Error("Failed to record data lineage", zap.Error(err))
		return fmt.Errorf("failed to record data lineage: %w", err)
	}
	
	return nil
}

// RecordClinicalDecision logs clinical reasoning decisions with full audit trail
func (s *EvidenceService) RecordClinicalDecision(ctx context.Context, decision models.ClinicalDecision) error {
	query := `
		INSERT INTO clinical_decisions (
			transaction_id, decision_id, patient_id, decision_type, knowledge_source,
			input_data, decision_outcome, confidence_score, evidence_sources, expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	
	_, err := s.db.ExecContext(ctx, query,
		decision.TransactionID, decision.DecisionID, decision.PatientID, decision.DecisionType,
		decision.KnowledgeSource, decision.InputData, decision.DecisionOutcome,
		decision.ConfidenceScore, decision.EvidenceSources, decision.ExpiresAt,
	)
	
	if err != nil {
		s.logger.Error("Failed to record clinical decision", zap.String("decision_id", decision.DecisionID), zap.Error(err))
		return fmt.Errorf("failed to record clinical decision: %w", err)
	}
	
	return nil
}

// GetTransactionAuditTrail retrieves complete audit trail for a transaction
func (s *EvidenceService) GetTransactionAuditTrail(ctx context.Context, transactionID string) (*models.EvidenceTransaction, error) {
	query := `
		SELECT id, transaction_id, user_id, session_id, source_service, target_service,
			   operation_type, graphql_operation, request_payload, response_payload,
			   http_status, processing_time_ms, timestamp, correlation_id, trace_id, span_id
		FROM evidence_transactions 
		WHERE transaction_id = $1`
	
	var transaction models.EvidenceTransaction
	err := s.db.QueryRowContext(ctx, query, transactionID).Scan(
		&transaction.ID, &transaction.TransactionID, &transaction.UserID, &transaction.SessionID,
		&transaction.SourceService, &transaction.TargetService, &transaction.OperationType,
		&transaction.GraphQLOperation, &transaction.RequestPayload, &transaction.ResponsePayload,
		&transaction.HTTPStatus, &transaction.ProcessingTimeMS, &transaction.Timestamp,
		&transaction.CorrelationID, &transaction.TraceID, &transaction.SpanID,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("transaction not found: %s", transactionID)
		}
		s.logger.Error("Failed to get transaction audit trail", zap.String("transaction_id", transactionID), zap.Error(err))
		return nil, fmt.Errorf("failed to get transaction audit trail: %w", err)
	}
	
	return &transaction, nil
}

// QueryAuditTrail searches audit trails with flexible filters
func (s *EvidenceService) QueryAuditTrail(ctx context.Context, query models.AuditQuery) ([]models.EvidenceTransaction, error) {
	baseQuery := `
		SELECT id, transaction_id, user_id, session_id, source_service, target_service,
			   operation_type, graphql_operation, request_payload, response_payload,
			   http_status, processing_time_ms, timestamp, correlation_id, trace_id, span_id
		FROM evidence_transactions WHERE 1=1`
	
	var args []interface{}
	argCount := 0
	
	if query.UserID != nil {
		argCount++
		baseQuery += fmt.Sprintf(" AND user_id = $%d", argCount)
		args = append(args, *query.UserID)
	}
	
	if query.Service != nil {
		argCount++
		baseQuery += fmt.Sprintf(" AND (source_service = $%d OR target_service = $%d)", argCount, argCount)
		args = append(args, *query.Service)
	}
	
	if query.OperationType != nil {
		argCount++
		baseQuery += fmt.Sprintf(" AND operation_type = $%d", argCount)
		args = append(args, *query.OperationType)
	}
	
	if query.StartTime != nil {
		argCount++
		baseQuery += fmt.Sprintf(" AND timestamp >= $%d", argCount)
		args = append(args, *query.StartTime)
	}
	
	if query.EndTime != nil {
		argCount++
		baseQuery += fmt.Sprintf(" AND timestamp <= $%d", argCount)
		args = append(args, *query.EndTime)
	}
	
	baseQuery += " ORDER BY timestamp DESC"
	
	if query.Limit > 0 {
		argCount++
		baseQuery += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, query.Limit)
	}
	
	if query.Offset > 0 {
		argCount++
		baseQuery += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, query.Offset)
	}
	
	rows, err := s.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		s.logger.Error("Failed to query audit trail", zap.Error(err))
		return nil, fmt.Errorf("failed to query audit trail: %w", err)
	}
	defer rows.Close()
	
	var transactions []models.EvidenceTransaction
	for rows.Next() {
		var transaction models.EvidenceTransaction
		err := rows.Scan(
			&transaction.ID, &transaction.TransactionID, &transaction.UserID, &transaction.SessionID,
			&transaction.SourceService, &transaction.TargetService, &transaction.OperationType,
			&transaction.GraphQLOperation, &transaction.RequestPayload, &transaction.ResponsePayload,
			&transaction.HTTPStatus, &transaction.ProcessingTimeMS, &transaction.Timestamp,
			&transaction.CorrelationID, &transaction.TraceID, &transaction.SpanID,
		)
		if err != nil {
			s.logger.Error("Failed to scan transaction row", zap.Error(err))
			return nil, fmt.Errorf("failed to scan transaction row: %w", err)
		}
		transactions = append(transactions, transaction)
	}
	
	return transactions, nil
}

// RecordKBVersion tracks knowledge base deployments and versions
func (s *EvidenceService) RecordKBVersion(ctx context.Context, version models.KBVersion) error {
	// First deactivate any existing active version for this KB service
	_, err := s.db.ExecContext(ctx,
		"UPDATE kb_versions SET is_active = FALSE, deactivated_at = NOW() WHERE kb_service = $1 AND is_active = TRUE",
		version.KBService)
	if err != nil {
		s.logger.Error("Failed to deactivate existing KB version", zap.String("kb_service", version.KBService), zap.Error(err))
		return fmt.Errorf("failed to deactivate existing KB version: %w", err)
	}
	
	// Insert new version
	query := `
		INSERT INTO kb_versions (
			kb_service, version, schema_version, data_sources, validation_status,
			validation_results, is_active
		) VALUES ($1, $2, $3, $4, $5, $6, $7)`
	
	_, err = s.db.ExecContext(ctx, query,
		version.KBService, version.Version, version.SchemaVersion, version.DataSources,
		version.ValidationStatus, version.ValidationResults, version.IsActive,
	)
	
	if err != nil {
		s.logger.Error("Failed to record KB version", zap.String("kb_service", version.KBService), zap.Error(err))
		return fmt.Errorf("failed to record KB version: %w", err)
	}
	
	return nil
}

// RecordSystemMetric logs performance and operational metrics
func (s *EvidenceService) RecordSystemMetric(ctx context.Context, metric models.SystemMetric) error {
	query := `
		INSERT INTO system_metrics (service_name, metric_name, metric_value, metric_unit, tags)
		VALUES ($1, $2, $3, $4, $5)`
	
	_, err := s.db.ExecContext(ctx, query,
		metric.ServiceName, metric.MetricName, metric.MetricValue, metric.MetricUnit, metric.Tags,
	)
	
	if err != nil {
		s.logger.Error("Failed to record system metric", 
			zap.String("service", metric.ServiceName),
			zap.String("metric", metric.MetricName), 
			zap.Error(err))
		return fmt.Errorf("failed to record system metric: %w", err)
	}
	
	return nil
}

// GetClinicalDecisionHistory retrieves decision audit trail for a patient
func (s *EvidenceService) GetClinicalDecisionHistory(ctx context.Context, patientID string, limit int) ([]models.ClinicalDecision, error) {
	query := `
		SELECT id, transaction_id, decision_id, patient_id, decision_type, knowledge_source,
			   input_data, decision_outcome, confidence_score, evidence_sources,
			   overridden_by, override_reason, created_at, expires_at
		FROM clinical_decisions 
		WHERE patient_id = $1 
		ORDER BY created_at DESC 
		LIMIT $2`
	
	rows, err := s.db.QueryContext(ctx, query, patientID, limit)
	if err != nil {
		s.logger.Error("Failed to get clinical decision history", zap.String("patient_id", patientID), zap.Error(err))
		return nil, fmt.Errorf("failed to get clinical decision history: %w", err)
	}
	defer rows.Close()
	
	var decisions []models.ClinicalDecision
	for rows.Next() {
		var decision models.ClinicalDecision
		err := rows.Scan(
			&decision.ID, &decision.TransactionID, &decision.DecisionID, &decision.PatientID,
			&decision.DecisionType, &decision.KnowledgeSource, &decision.InputData,
			&decision.DecisionOutcome, &decision.ConfidenceScore, &decision.EvidenceSources,
			&decision.OverriddenBy, &decision.OverrideReason, &decision.CreatedAt, &decision.ExpiresAt,
		)
		if err != nil {
			s.logger.Error("Failed to scan clinical decision row", zap.Error(err))
			return nil, fmt.Errorf("failed to scan clinical decision row: %w", err)
		}
		decisions = append(decisions, decision)
	}
	
	return decisions, nil
}