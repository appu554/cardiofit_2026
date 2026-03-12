// Package database provides PostgreSQL database operations for KB-19.
// This layer handles decision audit storage for regulatory compliance (FDA 21 CFR Part 11).
package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"

	"kb-19-protocol-orchestrator/internal/models"
)

// PostgresDB is the PostgreSQL database client for KB-19.
type PostgresDB struct {
	db  *sql.DB
	log *logrus.Entry
}

// NewPostgresDB creates a new PostgreSQL database connection.
func NewPostgresDB(connString string, log *logrus.Entry) (*PostgresDB, error) {
	db, err := sql.Open("postgres", connString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresDB{
		db:  db,
		log: log.WithField("component", "postgres"),
	}, nil
}

// Close closes the database connection.
func (p *PostgresDB) Close() error {
	return p.db.Close()
}

// SaveRecommendationBundle saves a complete recommendation bundle for audit.
func (p *PostgresDB) SaveRecommendationBundle(ctx context.Context, bundle *models.RecommendationBundle) error {
	tx, err := p.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Marshal complex fields to JSON
	serviceVersionsJSON, _ := json.Marshal(bundle.ServiceVersions)
	executiveSummaryJSON, _ := json.Marshal(bundle.ExecutiveSummary)
	riskAssessmentJSON, _ := json.Marshal(bundle.RiskAssessment)

	// Insert recommendation bundle
	_, err = tx.ExecContext(ctx, `
		INSERT INTO recommendation_bundles (
			id, patient_id, encounter_id, timestamp, status, narrative_summary,
			processing_time_ms, protocols_evaluated, protocols_applicable,
			conflicts_detected, safety_blocks, highest_urgency,
			service_versions, executive_summary, risk_assessment
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`,
		bundle.ID, bundle.PatientID, bundle.EncounterID, bundle.Timestamp,
		bundle.Status, bundle.NarrativeSummary, bundle.ProcessingMetrics.TotalDurationMs,
		bundle.ExecutiveSummary.ProtocolsEvaluated, bundle.ExecutiveSummary.ProtocolsApplicable,
		bundle.ExecutiveSummary.ConflictsDetected, bundle.ExecutiveSummary.SafetyBlocks, bundle.ExecutiveSummary.HighestUrgency,
		serviceVersionsJSON, executiveSummaryJSON, riskAssessmentJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to insert recommendation bundle: %w", err)
	}

	// Insert each arbitrated decision
	for _, decision := range bundle.Decisions {
		if err := p.saveDecision(ctx, tx, bundle.ID, &decision); err != nil {
			return fmt.Errorf("failed to save decision: %w", err)
		}
	}

	// Insert each protocol evaluation
	for _, eval := range bundle.ProtocolEvaluations {
		if err := p.saveProtocolEvaluation(ctx, tx, bundle.ID, &eval); err != nil {
			return fmt.Errorf("failed to save protocol evaluation: %w", err)
		}
	}

	// Insert each conflict resolution
	for _, conflict := range bundle.ConflictsResolved {
		if err := p.saveConflictResolution(ctx, tx, bundle.ID, &conflict); err != nil {
			return fmt.Errorf("failed to save conflict resolution: %w", err)
		}
	}

	// Insert each safety gate
	for _, gate := range bundle.SafetyGatesApplied {
		if err := p.saveSafetyGate(ctx, tx, bundle.ID, &gate); err != nil {
			return fmt.Errorf("failed to save safety gate: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	p.log.WithFields(logrus.Fields{
		"bundle_id":  bundle.ID,
		"patient_id": bundle.PatientID,
		"decisions":  len(bundle.Decisions),
	}).Debug("Recommendation bundle saved")

	return nil
}

// saveDecision saves a single arbitrated decision.
func (p *PostgresDB) saveDecision(ctx context.Context, tx *sql.Tx, bundleID uuid.UUID, decision *models.ArbitratedDecision) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO arbitrated_decisions (
			id, bundle_id, decision_type, target, target_rxnorm, target_snomed,
			rationale, urgency, source_protocol, source_protocol_id,
			arbitration_reason, conflicted_with, conflict_type,
			recommendation_class, evidence_level
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`,
		decision.ID, bundleID, decision.DecisionType, decision.Target,
		decision.TargetRxNorm, decision.TargetSNOMED, decision.Rationale,
		decision.Urgency, decision.SourceProtocol, decision.SourceProtocolID,
		decision.ArbitrationReason, decision.ConflictedWith, decision.ConflictType,
		decision.Evidence.RecommendationClass, decision.Evidence.EvidenceLevel,
	)
	if err != nil {
		return err
	}

	// Insert safety flags for this decision
	for _, flag := range decision.SafetyFlags {
		_, err := tx.ExecContext(ctx, `
			INSERT INTO safety_flags (
				decision_id, flag_type, severity, reason, source
			) VALUES ($1, $2, $3, $4, $5)
		`, decision.ID, flag.Type, flag.Severity, flag.Reason, flag.Source)
		if err != nil {
			return fmt.Errorf("failed to insert safety flag: %w", err)
		}
	}

	// Insert evidence envelope
	if err := p.saveEvidenceEnvelope(ctx, tx, decision.ID, &decision.Evidence); err != nil {
		return fmt.Errorf("failed to save evidence envelope: %w", err)
	}

	return nil
}

// saveEvidenceEnvelope saves an evidence envelope.
func (p *PostgresDB) saveEvidenceEnvelope(ctx context.Context, tx *sql.Tx, decisionID uuid.UUID, evidence *models.EvidenceEnvelope) error {
	inferenceChainJSON, _ := json.Marshal(evidence.InferenceChain)
	kbVersionsJSON, _ := json.Marshal(evidence.KBVersions)

	_, err := tx.ExecContext(ctx, `
		INSERT INTO evidence_envelopes (
			decision_id, recommendation_class, evidence_level,
			guideline_source, guideline_version, guideline_year,
			citation_anchor, citation_text, inference_chain,
			kb_versions, cql_engine_version, patient_context_id,
			checksum, finalized
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	`,
		decisionID, evidence.RecommendationClass, evidence.EvidenceLevel,
		evidence.GuidelineSource, evidence.GuidelineVersion, evidence.GuidelineYear,
		evidence.CitationAnchor, evidence.CitationText, inferenceChainJSON,
		kbVersionsJSON, evidence.CQLEngineVersion, evidence.PatientContextID,
		evidence.Checksum, evidence.Finalized,
	)
	return err
}

// saveProtocolEvaluation saves a protocol evaluation.
func (p *PostgresDB) saveProtocolEvaluation(ctx context.Context, tx *sql.Tx, bundleID uuid.UUID, eval *models.ProtocolEvaluation) error {
	calculatorsJSON, _ := json.Marshal(eval.CalculatorsUsed)

	_, err := tx.ExecContext(ctx, `
		INSERT INTO protocol_evaluations (
			bundle_id, protocol_id, protocol_name, is_applicable,
			applicability_reason, contraindicated, contraindication_reasons,
			priority_class, risk_score_impact, cql_facts_used,
			calculators_used, confidence
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`,
		bundleID, eval.ProtocolID, eval.ProtocolName, eval.IsApplicable,
		eval.ApplicabilityReason, eval.Contraindicated, eval.ContraindicationReasons,
		eval.PriorityClass, eval.RiskScoreImpact, eval.CQLFactsUsed,
		calculatorsJSON, eval.Confidence,
	)
	return err
}

// saveConflictResolution saves a conflict resolution.
func (p *PostgresDB) saveConflictResolution(ctx context.Context, tx *sql.Tx, bundleID uuid.UUID, conflict *models.ConflictResolution) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO conflict_resolutions (
			bundle_id, protocol_a, protocol_b, conflict_type,
			winner, loser, resolution_rule, explanation,
			loser_outcome, confidence
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`,
		bundleID, conflict.ProtocolA, conflict.ProtocolB, conflict.ConflictType,
		conflict.Winner, conflict.Loser, conflict.ResolutionRule,
		conflict.Explanation, conflict.LoserOutcome, conflict.Confidence,
	)
	return err
}

// saveSafetyGate saves a safety gate.
func (p *PostgresDB) saveSafetyGate(ctx context.Context, tx *sql.Tx, bundleID uuid.UUID, gate *models.SafetyGate) error {
	_, err := tx.ExecContext(ctx, `
		INSERT INTO safety_gates_applied (
			bundle_id, gate_name, source, triggered, result,
			details, affected_decisions
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`,
		bundleID, gate.Name, gate.Source, gate.Triggered,
		gate.Result, gate.Details, gate.AffectedDecisions,
	)
	return err
}

// GetRecommendationBundle retrieves a recommendation bundle by ID.
func (p *PostgresDB) GetRecommendationBundle(ctx context.Context, bundleID uuid.UUID) (*models.RecommendationBundle, error) {
	var bundle models.RecommendationBundle
	var serviceVersionsJSON, executiveSummaryJSON, riskAssessmentJSON []byte
	var createdAt time.Time // Database metadata field, not used in model

	err := p.db.QueryRowContext(ctx, `
		SELECT id, patient_id, encounter_id, timestamp, status, narrative_summary,
			   processing_time_ms, protocols_evaluated, protocols_applicable,
			   conflicts_detected, safety_blocks, highest_urgency,
			   service_versions, executive_summary, risk_assessment, created_at
		FROM recommendation_bundles WHERE id = $1
	`, bundleID).Scan(
		&bundle.ID, &bundle.PatientID, &bundle.EncounterID, &bundle.Timestamp,
		&bundle.Status, &bundle.NarrativeSummary, &bundle.ProcessingMetrics.TotalDurationMs,
		&bundle.ExecutiveSummary.ProtocolsEvaluated, &bundle.ExecutiveSummary.ProtocolsApplicable,
		&bundle.ExecutiveSummary.ConflictsDetected, &bundle.ExecutiveSummary.SafetyBlocks, &bundle.ExecutiveSummary.HighestUrgency,
		&serviceVersionsJSON, &executiveSummaryJSON, &riskAssessmentJSON,
		&createdAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get bundle: %w", err)
	}

	// Unmarshal JSON fields
	json.Unmarshal(serviceVersionsJSON, &bundle.ServiceVersions)
	json.Unmarshal(executiveSummaryJSON, &bundle.ExecutiveSummary)
	json.Unmarshal(riskAssessmentJSON, &bundle.RiskAssessment)

	// Load decisions
	decisions, err := p.getDecisionsForBundle(ctx, bundleID)
	if err != nil {
		return nil, err
	}
	bundle.Decisions = decisions

	return &bundle, nil
}

// getDecisionsForBundle retrieves all decisions for a bundle.
func (p *PostgresDB) getDecisionsForBundle(ctx context.Context, bundleID uuid.UUID) ([]models.ArbitratedDecision, error) {
	rows, err := p.db.QueryContext(ctx, `
		SELECT id, decision_type, target, target_rxnorm, target_snomed,
			   rationale, urgency, source_protocol, source_protocol_id,
			   arbitration_reason, conflicted_with, conflict_type,
			   recommendation_class, evidence_level
		FROM arbitrated_decisions WHERE bundle_id = $1
	`, bundleID)
	if err != nil {
		return nil, fmt.Errorf("failed to query decisions: %w", err)
	}
	defer rows.Close()

	var decisions []models.ArbitratedDecision
	for rows.Next() {
		var d models.ArbitratedDecision
		var targetRxNorm, targetSNOMED, conflictedWith, conflictType sql.NullString

		err := rows.Scan(
			&d.ID, &d.DecisionType, &d.Target, &targetRxNorm, &targetSNOMED,
			&d.Rationale, &d.Urgency, &d.SourceProtocol, &d.SourceProtocolID,
			&d.ArbitrationReason, &conflictedWith, &conflictType,
			&d.Evidence.RecommendationClass, &d.Evidence.EvidenceLevel,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan decision: %w", err)
		}

		if targetRxNorm.Valid {
			d.TargetRxNorm = targetRxNorm.String
		}
		if targetSNOMED.Valid {
			d.TargetSNOMED = targetSNOMED.String
		}
		if conflictedWith.Valid {
			d.ConflictedWith = conflictedWith.String
		}
		if conflictType.Valid {
			d.ConflictType = conflictType.String
		}

		decisions = append(decisions, d)
	}

	return decisions, nil
}

// GetDecisionsForPatient retrieves recent decisions for a patient.
func (p *PostgresDB) GetDecisionsForPatient(ctx context.Context, patientID uuid.UUID, limit int) ([]models.ArbitratedDecision, error) {
	rows, err := p.db.QueryContext(ctx, `
		SELECT d.id, d.decision_type, d.target, d.rationale, d.urgency,
			   d.source_protocol, d.recommendation_class, d.evidence_level,
			   b.timestamp
		FROM arbitrated_decisions d
		JOIN recommendation_bundles b ON d.bundle_id = b.id
		WHERE b.patient_id = $1
		ORDER BY b.timestamp DESC
		LIMIT $2
	`, patientID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query decisions: %w", err)
	}
	defer rows.Close()

	var decisions []models.ArbitratedDecision
	for rows.Next() {
		var d models.ArbitratedDecision
		var timestamp time.Time

		err := rows.Scan(
			&d.ID, &d.DecisionType, &d.Target, &d.Rationale, &d.Urgency,
			&d.SourceProtocol, &d.Evidence.RecommendationClass,
			&d.Evidence.EvidenceLevel, &timestamp,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan decision: %w", err)
		}

		decisions = append(decisions, d)
	}

	return decisions, nil
}

// WriteAuditLog writes an audit log entry.
func (p *PostgresDB) WriteAuditLog(ctx context.Context, operation, entityType string, entityID, patientID uuid.UUID, userID string, details map[string]interface{}) error {
	detailsJSON, _ := json.Marshal(details)

	_, err := p.db.ExecContext(ctx, `
		INSERT INTO audit_log (operation, entity_type, entity_id, patient_id, user_id, details)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, operation, entityType, entityID, patientID, userID, detailsJSON)
	if err != nil {
		return fmt.Errorf("failed to write audit log: %w", err)
	}

	return nil
}

// Health checks if the database is healthy.
func (p *PostgresDB) Health(ctx context.Context) error {
	return p.db.PingContext(ctx)
}
