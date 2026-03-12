// Package database provides PostgreSQL persistence for the KB-0 governance platform.
package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"kb-0-governance-platform/internal/policy"
)

// =============================================================================
// FACT STORE
// =============================================================================
// FactStore provides access to the Canonical Fact Store (clinical_facts table).
// This is the Phase 2 addition that connects KB-0 governance to the Shared DB.
// =============================================================================

// FactStore provides PostgreSQL persistence for clinical facts.
type FactStore struct {
	db *sql.DB
}

// NewFactStore creates a new fact store.
func NewFactStore(db *sql.DB) *FactStore {
	return &FactStore{db: db}
}

// =============================================================================
// QUEUE OPERATIONS
// =============================================================================

// GetGovernanceQueue returns pending facts from the governance queue view.
func (s *FactStore) GetGovernanceQueue(ctx context.Context, limit int) ([]*policy.QueueItem, error) {
	query := `
		SELECT
			fact_id, fact_type, rxcui, drug_name, scope,
			content, source_type, source_id,
			confidence_score, confidence_band, status,
			review_priority, assigned_reviewer, review_due_at,
			has_conflict, conflict_with_fact_ids, authority_priority,
			created_at, priority_rank, days_until_due, sla_status
		FROM v_governance_queue
		ORDER BY priority_rank ASC, created_at ASC
		LIMIT $1
	`

	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get governance queue: %w", err)
	}
	defer rows.Close()

	var items []*policy.QueueItem
	for rows.Next() {
		item := &policy.QueueItem{}
		var contentJSON []byte
		var conflictIDs pq.StringArray
		var reviewPriority sql.NullString
		var assignedReviewer sql.NullString
		var reviewDueAt sql.NullTime
		var confidenceScore sql.NullFloat64
		var daysUntilDue sql.NullFloat64

		err := rows.Scan(
			&item.FactID, &item.FactType, &item.RxCUI, &item.DrugName, &item.Scope,
			&contentJSON, &item.SourceType, &item.SourceID,
			&confidenceScore, &item.ConfidenceBand, &item.Status,
			&reviewPriority, &assignedReviewer, &reviewDueAt,
			&item.HasConflict, &conflictIDs, &item.AuthorityPriority,
			&item.CreatedAt, &item.PriorityRank, &daysUntilDue, &item.SLAStatus,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan queue item: %w", err)
		}

		// Parse JSON content
		if len(contentJSON) > 0 {
			json.Unmarshal(contentJSON, &item.Content)
		}

		// Parse nullable fields
		if confidenceScore.Valid {
			item.ConfidenceScore = &confidenceScore.Float64
		}
		if reviewPriority.Valid {
			rp := policy.ReviewPriority(reviewPriority.String)
			item.ReviewPriority = &rp
		}
		if assignedReviewer.Valid {
			item.AssignedReviewer = &assignedReviewer.String
		}
		if reviewDueAt.Valid {
			item.ReviewDueAt = &reviewDueAt.Time
		}
		if daysUntilDue.Valid {
			item.DaysUntilDue = &daysUntilDue.Float64
		}

		// Parse conflict IDs
		for _, idStr := range conflictIDs {
			if id, err := uuid.Parse(idStr); err == nil {
				item.ConflictWithFactIDs = append(item.ConflictWithFactIDs, id)
			}
		}

		items = append(items, item)
	}

	return items, nil
}

// GetQueueByPriority returns facts filtered by priority level.
func (s *FactStore) GetQueueByPriority(ctx context.Context, priority policy.ReviewPriority, limit int) ([]*policy.QueueItem, error) {
	query := `
		SELECT
			fact_id, fact_type, rxcui, drug_name, scope,
			content, source_type, source_id,
			confidence_score, confidence_band, status,
			review_priority, assigned_reviewer, review_due_at,
			has_conflict, conflict_with_fact_ids, authority_priority,
			created_at, priority_rank, days_until_due, sla_status
		FROM v_governance_queue
		WHERE review_priority = $1
		ORDER BY created_at ASC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, string(priority), limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue by priority: %w", err)
	}
	defer rows.Close()

	return s.scanQueueItems(rows)
}

// GetQueueByReviewer returns facts assigned to a specific reviewer.
func (s *FactStore) GetQueueByReviewer(ctx context.Context, reviewerID string) ([]*policy.QueueItem, error) {
	query := `
		SELECT
			fact_id, fact_type, rxcui, drug_name, scope,
			content, source_type, source_id,
			confidence_score, confidence_band, status,
			review_priority, assigned_reviewer, review_due_at,
			has_conflict, conflict_with_fact_ids, authority_priority,
			created_at, 0 as priority_rank, NULL as days_until_due, 'ASSIGNED' as sla_status
		FROM clinical_facts
		WHERE assigned_reviewer = $1
		  AND status = 'DRAFT'
		  AND governance_decision IS NULL
		ORDER BY review_priority, created_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, reviewerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get reviewer queue: %w", err)
	}
	defer rows.Close()

	return s.scanQueueItems(rows)
}

// =============================================================================
// FACT OPERATIONS
// =============================================================================

// GetFact retrieves a clinical fact by ID.
// First tries clinical_facts, then falls back to derived_facts for pending governance items.
func (s *FactStore) GetFact(ctx context.Context, factID uuid.UUID) (*policy.ClinicalFact, error) {
	// First try clinical_facts
	fact, err := s.getFactFromClinicalFacts(ctx, factID)
	if err == nil {
		return fact, nil
	}

	// If not found in clinical_facts, try derived_facts (for pending governance items)
	fact, err = s.getFactFromDerivedFacts(ctx, factID)
	if err == nil {
		return fact, nil
	}

	return nil, fmt.Errorf("fact not found: %s", factID)
}

// getFactFromClinicalFacts retrieves a fact from the clinical_facts table.
func (s *FactStore) getFactFromClinicalFacts(ctx context.Context, factID uuid.UUID) (*policy.ClinicalFact, error) {
	query := `
		SELECT
			fact_id, fact_type, rxcui, drug_name, scope, class_rxcui, class_name,
			content, source_type, source_id, source_version, extraction_method,
			confidence_score, confidence_band, confidence_signals,
			status, effective_from, effective_to, superseded_by, version,
			review_priority, assigned_reviewer, assigned_at, review_due_at,
			governance_decision, decision_reason, decision_at, decision_by,
			has_conflict, conflict_with_fact_ids, conflict_resolution_notes,
			authority_priority, created_at, created_by, updated_at
		FROM clinical_facts
		WHERE fact_id = $1
	`

	fact := &policy.ClinicalFact{}
	var contentJSON, confidenceSignalsJSON []byte
	var conflictIDs pq.StringArray

	// Nullable fields
	var classRxCUI, className, sourceVersion sql.NullString
	var confidenceScore sql.NullFloat64
	var effectiveTo sql.NullTime
	var supersededBy sql.NullString
	var reviewPriority, assignedReviewer, govDecision, decisionReason, decisionBy sql.NullString
	var assignedAt, reviewDueAt, decisionAt sql.NullTime
	var conflictNotes sql.NullString

	err := s.db.QueryRowContext(ctx, query, factID).Scan(
		&fact.FactID, &fact.FactType, &fact.RxCUI, &fact.DrugName, &fact.Scope, &classRxCUI, &className,
		&contentJSON, &fact.SourceType, &fact.SourceID, &sourceVersion, &fact.ExtractionMethod,
		&confidenceScore, &fact.ConfidenceBand, &confidenceSignalsJSON,
		&fact.Status, &fact.EffectiveFrom, &effectiveTo, &supersededBy, &fact.Version,
		&reviewPriority, &assignedReviewer, &assignedAt, &reviewDueAt,
		&govDecision, &decisionReason, &decisionAt, &decisionBy,
		&fact.HasConflict, &conflictIDs, &conflictNotes,
		&fact.AuthorityPriority, &fact.CreatedAt, &fact.CreatedBy, &fact.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("fact not found in clinical_facts: %s", factID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get fact from clinical_facts: %w", err)
	}

	// Parse JSON
	if len(contentJSON) > 0 {
		json.Unmarshal(contentJSON, &fact.Content)
	}
	if len(confidenceSignalsJSON) > 0 {
		json.Unmarshal(confidenceSignalsJSON, &fact.ConfidenceSignals)
	}

	// Parse nullable strings
	if classRxCUI.Valid {
		fact.ClassRxCUI = &classRxCUI.String
	}
	if className.Valid {
		fact.ClassName = &className.String
	}
	if sourceVersion.Valid {
		fact.SourceVersion = &sourceVersion.String
	}
	if confidenceScore.Valid {
		fact.ConfidenceScore = &confidenceScore.Float64
	}
	if effectiveTo.Valid {
		fact.EffectiveTo = &effectiveTo.Time
	}
	if supersededBy.Valid {
		if id, err := uuid.Parse(supersededBy.String); err == nil {
			fact.SupersededBy = &id
		}
	}
	if reviewPriority.Valid {
		rp := policy.ReviewPriority(reviewPriority.String)
		fact.ReviewPriority = &rp
	}
	if assignedReviewer.Valid {
		fact.AssignedReviewer = &assignedReviewer.String
	}
	if assignedAt.Valid {
		fact.AssignedAt = &assignedAt.Time
	}
	if reviewDueAt.Valid {
		fact.ReviewDueAt = &reviewDueAt.Time
	}
	if govDecision.Valid {
		gd := policy.GovernanceDecision(govDecision.String)
		fact.GovernanceDecision = &gd
	}
	if decisionReason.Valid {
		fact.DecisionReason = &decisionReason.String
	}
	if decisionAt.Valid {
		fact.DecisionAt = &decisionAt.Time
	}
	if decisionBy.Valid {
		fact.DecisionBy = &decisionBy.String
	}
	if conflictNotes.Valid {
		fact.ConflictResolutionNotes = &conflictNotes.String
	}

	// Parse conflict IDs
	for _, idStr := range conflictIDs {
		if id, err := uuid.Parse(idStr); err == nil {
			fact.ConflictWithFactIDs = append(fact.ConflictWithFactIDs, id)
		}
	}

	return fact, nil
}

// getFactFromDerivedFacts retrieves a fact from the derived_facts table.
// This is used for pending governance items that haven't been projected to clinical_facts yet.
func (s *FactStore) getFactFromDerivedFacts(ctx context.Context, factID uuid.UUID) (*policy.ClinicalFact, error) {
	query := `
		SELECT
			df.id,
			df.fact_type,
			sd.rxcui,
			sd.drug_name,
			COALESCE(dm.generic_name, sd.generic_name),
			sd.manufacturer,
			COALESCE(dm.ndcs, sd.ndc_codes),
			COALESCE(dm.atc_codes, sd.atc_codes),
			'DRUG' AS scope,
			df.fact_data,
			sd.source_type,
			sd.document_id,
			df.extraction_method,
			df.extraction_confidence,
			df.governance_status,
			df.reviewed_by,
			df.reviewed_at,
			df.review_notes,
			df.is_active,
			df.created_at,
			df.updated_at,
			df.evidence_spans,
			df.source_section_id
		FROM derived_facts df
		JOIN source_documents sd ON df.source_document_id = sd.id
		LEFT JOIN drug_master dm ON sd.rxcui = dm.rxcui
		WHERE df.id = $1
	`

	fact := &policy.ClinicalFact{}
	var contentJSON []byte
	var confidenceScore sql.NullFloat64
	var governanceStatus sql.NullString
	var reviewedBy sql.NullString
	var reviewedAt sql.NullTime
	var reviewNotes sql.NullString
	var isActive bool
	var evidenceSpansJSON []byte
	var sourceSectionID *uuid.UUID
	var genericName sql.NullString
	var manufacturer sql.NullString
	var ndcCodes pq.StringArray
	var atcCodes pq.StringArray

	err := s.db.QueryRowContext(ctx, query, factID).Scan(
		&fact.FactID,
		&fact.FactType,
		&fact.RxCUI,
		&fact.DrugName,
		&genericName,
		&manufacturer,
		&ndcCodes,
		&atcCodes,
		&fact.Scope,
		&contentJSON,
		&fact.SourceType,
		&fact.SourceID,
		&fact.ExtractionMethod,
		&confidenceScore,
		&governanceStatus,
		&reviewedBy,
		&reviewedAt,
		&reviewNotes,
		&isActive,
		&fact.CreatedAt,
		&fact.UpdatedAt,
		&evidenceSpansJSON,
		&sourceSectionID,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("fact not found in derived_facts: %s", factID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get fact from derived_facts: %w", err)
	}

	// Parse JSON content
	if len(contentJSON) > 0 {
		json.Unmarshal(contentJSON, &fact.Content)
	}

	// Parse evidence spans
	if len(evidenceSpansJSON) > 0 {
		json.Unmarshal(evidenceSpansJSON, &fact.EvidenceSpans)
	}
	fact.SourceSectionID = sourceSectionID

	// Drug composition from drug_master (preferred) or source_documents (fallback)
	if genericName.Valid {
		fact.GenericName = &genericName.String
	}
	if manufacturer.Valid {
		fact.Manufacturer = &manufacturer.String
	}
	if len(ndcCodes) > 0 {
		fact.NDCCodes = []string(ndcCodes)
	}
	if len(atcCodes) > 0 {
		fact.ATCCodes = []string(atcCodes)
	}

	// Map governance_status to ClinicalFact status
	if governanceStatus.Valid {
		switch governanceStatus.String {
		case "PENDING_REVIEW":
			fact.Status = policy.FactStatusDraft
		case "APPROVED":
			fact.Status = policy.FactStatusApproved
		case "REJECTED":
			fact.Status = policy.FactStatus("REJECTED")
		default:
			fact.Status = policy.FactStatus(governanceStatus.String)
		}
	} else {
		fact.Status = policy.FactStatusDraft
	}

	// Map confidence score
	if confidenceScore.Valid {
		fact.ConfidenceScore = &confidenceScore.Float64
		// Determine confidence band
		score := confidenceScore.Float64
		if score >= 0.85 {
			fact.ConfidenceBand = "HIGH"
		} else if score >= 0.65 {
			fact.ConfidenceBand = "MEDIUM"
		} else {
			fact.ConfidenceBand = "LOW"
		}
	}

	// Map reviewer info
	if reviewedBy.Valid {
		fact.DecisionBy = &reviewedBy.String
	}
	if reviewedAt.Valid {
		fact.DecisionAt = &reviewedAt.Time
	}
	if reviewNotes.Valid {
		fact.DecisionReason = &reviewNotes.String
	}

	// Set defaults
	fact.Version = 1
	fact.HasConflict = false
	fact.AuthorityPriority = 0
	fact.CreatedBy = "SPL_PIPELINE"
	now := time.Now()
	fact.EffectiveFrom = now

	return fact, nil
}

// GetAllFacts returns all facts with optional filtering and pagination.
func (s *FactStore) GetAllFacts(ctx context.Context, status, factType, search string, page, pageSize int) ([]*policy.ClinicalFact, int, error) {
	// Build WHERE clauses for both tables
	cfClauses := []string{"1=1"}
	dfClauses := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if status != "" {
		cfClauses = append(cfClauses, fmt.Sprintf("status = $%d", argIdx))
		// Map clinical_facts status to derived_facts governance_status
		dfStatus := status
		switch status {
		case "ACTIVE", "APPROVED":
			dfStatus = "APPROVED"
		case "DRAFT":
			dfStatus = "PENDING_REVIEW"
		}
		dfClauses = append(dfClauses, fmt.Sprintf("governance_status = $%d", argIdx))
		args = append(args, dfStatus)
		// We need the clinical_facts filter to use the original status value
		// but both use the same $argIdx. Use a separate approach:
		// Actually, since the arg value differs, we need separate args.
		// Reset and use separate arg indices.
		args = args[:0]
		cfClauses = cfClauses[:1]
		dfClauses = dfClauses[:1]
		argIdx = 1

		cfClauses = append(cfClauses, fmt.Sprintf("cf.status = $%d", argIdx))
		args = append(args, status)
		argIdx++

		dfClauses = append(dfClauses, fmt.Sprintf("df.governance_status = $%d", argIdx))
		args = append(args, dfStatus)
		argIdx++
	}
	if factType != "" {
		cfClauses = append(cfClauses, fmt.Sprintf("cf.fact_type = $%d", argIdx))
		dfClauses = append(dfClauses, fmt.Sprintf("df.fact_type = $%d", argIdx))
		args = append(args, factType)
		argIdx++
	}
	if search != "" {
		cfClauses = append(cfClauses, fmt.Sprintf("(cf.drug_name ILIKE $%d OR cf.rxcui ILIKE $%d OR cf.content::text ILIKE $%d)", argIdx, argIdx, argIdx))
		dfClauses = append(dfClauses, fmt.Sprintf("(sd.drug_name ILIKE $%d OR sd.rxcui ILIKE $%d OR df.fact_data::text ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+search+"%")
		argIdx++
	}

	cfWhere := strings.Join(cfClauses, " AND ")
	dfWhere := strings.Join(dfClauses, " AND ")

	// Count total from both tables (deduplicated by fact_id)
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM (
			SELECT cf.fact_id::text AS id FROM clinical_facts cf WHERE %s
			UNION
			SELECT df.id AS id FROM derived_facts df
			JOIN source_documents sd ON df.source_document_id = sd.id
			WHERE %s AND df.governance_status IN ('APPROVED', 'ACTIVE')
			  AND df.id NOT IN (SELECT fact_id::text FROM clinical_facts)
		) combined
	`, cfWhere, dfWhere)

	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count facts: %w", err)
	}

	// Get paginated results from both tables
	offset := (page - 1) * pageSize
	paginatedArgs := append(append([]interface{}{}, args...), pageSize, offset)

	query := fmt.Sprintf(`
		SELECT id FROM (
			SELECT cf.fact_id::text AS id, cf.created_at FROM clinical_facts cf WHERE %s
			UNION
			SELECT df.id AS id, df.created_at FROM derived_facts df
			JOIN source_documents sd ON df.source_document_id = sd.id
			WHERE %s AND df.governance_status IN ('APPROVED', 'ACTIVE')
			  AND df.id NOT IN (SELECT fact_id::text FROM clinical_facts)
		) combined
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, cfWhere, dfWhere, argIdx, argIdx+1)

	rows, err := s.db.QueryContext(ctx, query, paginatedArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query facts: %w", err)
	}
	defer rows.Close()

	var facts []*policy.ClinicalFact
	for rows.Next() {
		var factID uuid.UUID
		if err := rows.Scan(&factID); err != nil {
			return nil, 0, fmt.Errorf("failed to scan fact ID: %w", err)
		}
		fact, err := s.GetFact(ctx, factID)
		if err != nil {
			continue
		}
		facts = append(facts, fact)
	}

	return facts, total, nil
}

// GetAllConflictGroups returns all facts that have conflicts, grouped by drug.
func (s *FactStore) GetAllConflictGroups(ctx context.Context) ([]*policy.ConflictGroup, error) {
	query := `
		SELECT DISTINCT rxcui, drug_name, fact_type
		FROM clinical_facts
		WHERE has_conflict = TRUE
		  AND status IN ('DRAFT', 'APPROVED', 'ACTIVE')
		ORDER BY drug_name
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query conflict groups: %w", err)
	}
	defer rows.Close()

	var groups []*policy.ConflictGroup
	for rows.Next() {
		var rxcui, drugName, factType string
		if err := rows.Scan(&rxcui, &drugName, &factType); err != nil {
			return nil, fmt.Errorf("failed to scan conflict group: %w", err)
		}

		// Get all conflicting facts for this drug/type
		factsQuery := `
			SELECT fact_id FROM clinical_facts
			WHERE rxcui = $1 AND fact_type = $2 AND has_conflict = TRUE
			  AND status IN ('DRAFT', 'APPROVED', 'ACTIVE')
		`
		factRows, err := s.db.QueryContext(ctx, factsQuery, rxcui, factType)
		if err != nil {
			continue
		}

		var facts []*policy.ClinicalFact
		for factRows.Next() {
			var factID uuid.UUID
			if err := factRows.Scan(&factID); err != nil {
				continue
			}
			fact, err := s.GetFact(ctx, factID)
			if err == nil {
				facts = append(facts, fact)
			}
		}
		factRows.Close()

		if len(facts) > 0 {
			group := &policy.ConflictGroup{
				GroupID:            fmt.Sprintf("%s_%s", rxcui, factType),
				DrugRxCUI:          rxcui,
				DrugName:           drugName,
				FactType:           factType,
				Facts:              facts,
				ResolutionStrategy: "MANUAL", // Default to manual
			}

			// Try to suggest a winner based on authority priority
			var highestPriority int = -1
			for _, f := range facts {
				if f.AuthorityPriority > highestPriority {
					highestPriority = f.AuthorityPriority
					group.SuggestedWinner = &f.FactID
					group.ResolutionStrategy = "AUTHORITY_PRIORITY"
					group.ResolutionReason = ptrString(fmt.Sprintf("Highest authority priority: %d", highestPriority))
				}
			}

			groups = append(groups, group)
		}
	}

	return groups, nil
}

// GetFactHistory returns audit events for a specific fact (21 CFR Part 11 compliant audit trail).
func (s *FactStore) GetFactHistory(ctx context.Context, factID uuid.UUID) ([]*policy.AuditLogEntry, error) {
	query := `
		SELECT
			audit_id, event_type, fact_id, previous_state, new_state,
			actor_type, actor_id, actor_name, event_details, ip_address,
			session_id, signature_hash, event_timestamp
		FROM governance_audit_log
		WHERE fact_id = $1
		ORDER BY event_timestamp DESC
	`

	rows, err := s.db.QueryContext(ctx, query, factID)
	if err != nil {
		// Table might not exist yet, return empty
		return []*policy.AuditLogEntry{}, nil
	}
	defer rows.Close()

	var entries []*policy.AuditLogEntry
	for rows.Next() {
		entry := &policy.AuditLogEntry{}
		var detailsJSON []byte
		var auditID uuid.UUID
		var factIDScanned sql.NullString
		var previousState, newState, ipAddress, sessionID, signature, actorName sql.NullString

		err := rows.Scan(
			&auditID, &entry.EventType, &factIDScanned, &previousState, &newState,
			&entry.ActorType, &entry.ActorID, &actorName, &detailsJSON, &ipAddress,
			&sessionID, &signature, &entry.CreatedAt,
		)
		if err != nil {
			log.Printf("[FactStore] Warning: failed to scan fact history entry: %v", err)
			continue
		}

		entry.ID = auditID.String()
		if factIDScanned.Valid {
			entry.FactID = factIDScanned.String
		}
		if previousState.Valid {
			entry.PreviousState = previousState.String
		}
		if newState.Valid {
			entry.NewState = newState.String
		}
		if actorName.Valid {
			entry.ActorName = actorName.String
		}
		if ipAddress.Valid {
			entry.IPAddress = ipAddress.String
		}
		if sessionID.Valid {
			entry.SessionID = sessionID.String
		}
		if signature.Valid {
			entry.Signature = signature.String
		}
		if len(detailsJSON) > 0 {
			json.Unmarshal(detailsJSON, &entry.Details)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// GetAuditLog returns system-wide audit events with filtering and pagination.
func (s *FactStore) GetAuditLog(ctx context.Context, eventType, actorID, fromDate, toDate string, page, pageSize int) ([]*policy.AuditLogEntry, int, error) {
	// Build WHERE clause
	whereClauses := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if eventType != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("event_type = $%d", argIdx))
		args = append(args, eventType)
		argIdx++
	}
	if actorID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("actor_id = $%d", argIdx))
		args = append(args, actorID)
		argIdx++
	}
	if fromDate != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("event_timestamp >= $%d::timestamp", argIdx))
		args = append(args, fromDate)
		argIdx++
	}
	if toDate != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("event_timestamp <= $%d::timestamp", argIdx))
		args = append(args, toDate)
		argIdx++
	}

	whereStr := ""
	for i, clause := range whereClauses {
		if i == 0 {
			whereStr = clause
		} else {
			whereStr += " AND " + clause
		}
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM governance_audit_log WHERE %s", whereStr)
	var total int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		// Table might not exist yet, return empty
		return []*policy.AuditLogEntry{}, 0, nil
	}

	// Get paginated results
	offset := (page - 1) * pageSize
	args = append(args, pageSize, offset)

	query := fmt.Sprintf(`
		SELECT
			audit_id, event_type, fact_id, previous_state, new_state,
			actor_type, actor_id, actor_name, event_details, ip_address,
			session_id, signature_hash, event_timestamp
		FROM governance_audit_log
		WHERE %s
		ORDER BY event_timestamp DESC
		LIMIT $%d OFFSET $%d
	`, whereStr, argIdx, argIdx+1)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		// Table might not exist yet, return empty
		return []*policy.AuditLogEntry{}, 0, nil
	}
	defer rows.Close()

	var entries []*policy.AuditLogEntry
	for rows.Next() {
		entry := &policy.AuditLogEntry{}
		var detailsJSON []byte
		var auditID uuid.UUID
		var factID sql.NullString
		var previousState, newState, ipAddress, sessionID, signature, actorName sql.NullString

		err := rows.Scan(
			&auditID, &entry.EventType, &factID, &previousState, &newState,
			&entry.ActorType, &entry.ActorID, &actorName, &detailsJSON, &ipAddress,
			&sessionID, &signature, &entry.CreatedAt,
		)
		if err != nil {
			log.Printf("[FactStore] Warning: failed to scan audit entry: %v", err)
			continue
		}

		entry.ID = auditID.String()
		if factID.Valid {
			entry.FactID = factID.String
		}
		if previousState.Valid {
			entry.PreviousState = previousState.String
		}
		if newState.Valid {
			entry.NewState = newState.String
		}
		if actorName.Valid {
			entry.ActorName = actorName.String
		}
		if ipAddress.Valid {
			entry.IPAddress = ipAddress.String
		}
		if sessionID.Valid {
			entry.SessionID = sessionID.String
		}
		if signature.Valid {
			entry.Signature = signature.String
		}
		if len(detailsJSON) > 0 {
			json.Unmarshal(detailsJSON, &entry.Details)
		}

		entries = append(entries, entry)
	}

	return entries, total, nil
}

func ptrString(s string) *string {
	return &s
}

// GetFactsByDrug returns all facts for a given RxCUI.
func (s *FactStore) GetFactsByDrug(ctx context.Context, rxcui string) ([]*policy.ClinicalFact, error) {
	query := `
		SELECT fact_id FROM clinical_facts
		WHERE rxcui = $1
		  AND status IN ('DRAFT', 'APPROVED', 'ACTIVE')
	`

	rows, err := s.db.QueryContext(ctx, query, rxcui)
	if err != nil {
		return nil, fmt.Errorf("failed to query facts by drug: %w", err)
	}
	defer rows.Close()

	var facts []*policy.ClinicalFact
	for rows.Next() {
		var factID uuid.UUID
		if err := rows.Scan(&factID); err != nil {
			return nil, fmt.Errorf("failed to scan fact ID: %w", err)
		}
		fact, err := s.GetFact(ctx, factID)
		if err != nil {
			continue // Skip if fetch fails
		}
		facts = append(facts, fact)
	}

	return facts, nil
}

// =============================================================================
// GOVERNANCE OPERATIONS
// =============================================================================

// UpdateGovernanceDecision updates the governance decision for a fact.
// Tries clinical_facts first, then falls back to derived_facts.
func (s *FactStore) UpdateGovernanceDecision(
	ctx context.Context,
	factID uuid.UUID,
	decision policy.GovernanceDecision,
	reason string,
	decisionBy string,
) error {
	now := time.Now()

	// Try clinical_facts first
	query := `
		UPDATE clinical_facts SET
			governance_decision = $2,
			decision_reason = $3,
			decision_at = $4,
			decision_by = $5,
			updated_at = $4
		WHERE fact_id = $1
	`

	result, err := s.db.ExecContext(ctx, query, factID, string(decision), reason, now, decisionBy)
	if err != nil {
		return fmt.Errorf("failed to update governance decision in clinical_facts: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		return nil
	}

	// Fallback to derived_facts
	governanceStatus := "PENDING_REVIEW"
	switch decision {
	case policy.DecisionApproved:
		governanceStatus = "APPROVED"
	case policy.DecisionRejected:
		governanceStatus = "REJECTED"
	}

	derivedQuery := `
		UPDATE derived_facts SET
			governance_status = $2,
			reviewed_by = $3,
			reviewed_at = $4,
			review_notes = $5,
			updated_at = $4
		WHERE id = $1
	`

	_, err = s.db.ExecContext(ctx, derivedQuery, factID, governanceStatus, decisionBy, now, reason)
	if err != nil {
		return fmt.Errorf("failed to update governance decision in derived_facts: %w", err)
	}

	return nil
}

// ActivateFact transitions a fact to ACTIVE status.
// Tries clinical_facts first, then falls back to derived_facts (marking as APPROVED).
func (s *FactStore) ActivateFact(ctx context.Context, factID uuid.UUID, activatedBy string) error {
	now := time.Now()

	// Try clinical_facts first
	query := `
		UPDATE clinical_facts SET
			status = 'ACTIVE',
			governance_decision = 'APPROVED',
			decision_at = $2,
			decision_by = $3,
			effective_from = $2,
			updated_at = $2
		WHERE fact_id = $1
		  AND status IN ('DRAFT', 'APPROVED')
	`

	result, err := s.db.ExecContext(ctx, query, factID, now, activatedBy)
	if err != nil {
		return fmt.Errorf("failed to activate fact in clinical_facts: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		return nil
	}

	// Fallback to derived_facts - mark as APPROVED and promote to clinical_facts
	derivedQuery := `
		UPDATE derived_facts SET
			governance_status = 'APPROVED',
			reviewed_by = $2,
			reviewed_at = $3,
			updated_at = $3
		WHERE id = $1
		  AND governance_status = 'PENDING_REVIEW'
	`

	result, err = s.db.ExecContext(ctx, derivedQuery, factID, activatedBy, now)
	if err != nil {
		return fmt.Errorf("failed to activate fact in derived_facts: %w", err)
	}

	rowsAffected, _ = result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("fact not found or not in activatable state: %s", factID)
	}

	// Promote: insert approved derived_fact into clinical_facts as ACTIVE
	promoteQuery := `
		INSERT INTO clinical_facts (
			fact_id, fact_type, rxcui, drug_name, scope,
			content, source_type, source_id, extraction_method,
			confidence_score, confidence_band,
			status, effective_from, version,
			governance_decision, decision_at, decision_by,
			created_at, created_by, updated_at
		)
		SELECT
			df.id,
			df.fact_type::fact_type,
			sd.rxcui,
			sd.drug_name,
			'DRUG'::fact_scope,
			df.fact_data,
			CASE sd.source_type
				WHEN 'FDA_SPL' THEN 'ETL'::source_type
				WHEN 'LLM' THEN 'LLM'::source_type
				WHEN 'API_SYNC' THEN 'API_SYNC'::source_type
				WHEN 'MANUAL' THEN 'MANUAL'::source_type
				ELSE 'ETL'::source_type
			END,
			sd.document_id,
			df.extraction_method,
			df.extraction_confidence,
			CASE
				WHEN df.extraction_confidence >= 0.85 THEN 'HIGH'::confidence_band
				WHEN df.extraction_confidence >= 0.65 THEN 'MEDIUM'::confidence_band
				ELSE 'LOW'::confidence_band
			END,
			'ACTIVE'::fact_status,
			$2,
			1,
			'APPROVED'::governance_decision,
			$2,
			$3,
			df.created_at,
			'SPL_PIPELINE',
			$2
		FROM derived_facts df
		JOIN source_documents sd ON df.source_document_id = sd.id
		WHERE df.id = $1
		ON CONFLICT (fact_id) DO UPDATE SET
			status = 'ACTIVE'::fact_status,
			governance_decision = 'APPROVED'::governance_decision,
			decision_at = $2,
			decision_by = $3,
			updated_at = $2
	`

	_, err = s.db.ExecContext(ctx, promoteQuery, factID, now, activatedBy)
	if err != nil {
		log.Printf("[Governance] Warning: failed to promote derived_fact to clinical_facts: %v", err)
		// Non-fatal: the fact is still APPROVED in derived_facts
	}

	return nil
}

// SupersedeFact marks a fact as superseded by another.
func (s *FactStore) SupersedeFact(ctx context.Context, oldFactID, newFactID uuid.UUID) error {
	now := time.Now()

	query := `
		UPDATE clinical_facts SET
			status = 'SUPERSEDED',
			superseded_by = $2,
			effective_to = $3,
			updated_at = $3
		WHERE fact_id = $1
		  AND status = 'ACTIVE'
	`

	_, err := s.db.ExecContext(ctx, query, oldFactID, newFactID, now)
	if err != nil {
		return fmt.Errorf("failed to supersede fact: %w", err)
	}

	return nil
}

// AssignReviewer assigns a reviewer to a fact.
func (s *FactStore) AssignReviewer(ctx context.Context, factID uuid.UUID, reviewerID string, priority policy.ReviewPriority) error {
	now := time.Now()
	dueAt := policy.CalculateSLADueDate(priority)

	query := `
		UPDATE clinical_facts SET
			assigned_reviewer = $2,
			assigned_at = $3,
			review_priority = $4,
			review_due_at = $5,
			updated_at = $3
		WHERE fact_id = $1
	`

	_, err := s.db.ExecContext(ctx, query, factID, reviewerID, now, string(priority), dueAt)
	if err != nil {
		return fmt.Errorf("failed to assign reviewer: %w", err)
	}

	return nil
}

// MarkConflict marks a fact as having a conflict.
func (s *FactStore) MarkConflict(ctx context.Context, factID uuid.UUID, conflictingIDs []uuid.UUID, notes string) error {
	now := time.Now()

	// Convert UUIDs to string array
	idStrings := make([]string, len(conflictingIDs))
	for i, id := range conflictingIDs {
		idStrings[i] = id.String()
	}

	query := `
		UPDATE clinical_facts SET
			has_conflict = TRUE,
			conflict_with_fact_ids = $2,
			conflict_resolution_notes = $3,
			updated_at = $4
		WHERE fact_id = $1
	`

	_, err := s.db.ExecContext(ctx, query, factID, pq.Array(idStrings), notes, now)
	if err != nil {
		return fmt.Errorf("failed to mark conflict: %w", err)
	}

	return nil
}

// =============================================================================
// AUDIT OPERATIONS
// =============================================================================

// LogGovernanceEvent logs an event to the governance audit log.
func (s *FactStore) LogGovernanceEvent(ctx context.Context, event *policy.AuditEvent) error {
	detailsJSON, _ := json.Marshal(event.Details)

	query := `
		SELECT log_governance_event($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := s.db.ExecContext(ctx, query,
		event.EventType,
		event.FactID,
		event.PreviousState,
		event.NewState,
		event.ActorType,
		event.ActorID,
		event.ActorName,
		detailsJSON,
		event.IPAddress,
		event.SessionID,
	)
	if err != nil {
		return fmt.Errorf("failed to log governance event: %w", err)
	}

	return nil
}

// RecordDecision records a governance decision in the decisions table.
func (s *FactStore) RecordDecision(
	ctx context.Context,
	factID uuid.UUID,
	decision policy.GovernanceDecision,
	policyName string,
	evaluationResult interface{},
	actorType, actorID string,
	credentials string,
) error {
	evalJSON, _ := json.Marshal(evaluationResult)

	query := `
		INSERT INTO governance_decisions (
			fact_id, decision, policy_name, evaluation_result,
			actor_type, actor_id, actor_credentials
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := s.db.ExecContext(ctx, query,
		factID, string(decision), policyName, evalJSON,
		actorType, actorID, credentials,
	)
	if err != nil {
		return fmt.Errorf("failed to record decision: %w", err)
	}

	return nil
}

// =============================================================================
// METRICS
// =============================================================================

// FactMetrics contains governance metrics for clinical facts.
type FactMetrics struct {
	TotalDraft       int       `json:"totalDraft"`
	TotalApproved    int       `json:"totalApproved"`
	TotalActive      int       `json:"totalActive"`
	TotalSuperseded  int       `json:"totalSuperseded"`
	PendingReview    int       `json:"pendingReview"`
	CriticalPending  int       `json:"criticalPending"`
	BreachedSLA      int       `json:"breachedSLA"`
	AtRiskSLA        int       `json:"atRiskSLA"`
	WithConflicts    int       `json:"withConflicts"`
	GeneratedAt      time.Time `json:"generatedAt"`
}

// GetFactMetrics returns governance metrics for clinical facts.
func (s *FactStore) GetFactMetrics(ctx context.Context) (*FactMetrics, error) {
	query := `
		SELECT
			COALESCE(cf.total_draft,0) + COALESCE(df.total_draft,0),
			COALESCE(cf.total_approved,0) + COALESCE(df.total_approved,0),
			COALESCE(cf.total_active,0) + COALESCE(df.total_active,0),
			COALESCE(cf.total_superseded,0) + COALESCE(df.total_superseded,0),
			COALESCE(cf.pending_review,0) + COALESCE(df.pending_review,0),
			COALESCE(cf.critical_pending,0) + COALESCE(df.critical_pending,0),
			COALESCE(cf.breached_sla,0) + COALESCE(df.breached_sla,0),
			COALESCE(cf.at_risk_sla,0) + COALESCE(df.at_risk_sla,0),
			COALESCE(cf.with_conflicts,0) + COALESCE(df.with_conflicts,0)
		FROM
		(SELECT
			COUNT(*) FILTER (WHERE status = 'DRAFT') AS total_draft,
			COUNT(*) FILTER (WHERE status = 'APPROVED') AS total_approved,
			COUNT(*) FILTER (WHERE status = 'ACTIVE') AS total_active,
			COUNT(*) FILTER (WHERE status = 'SUPERSEDED') AS total_superseded,
			COUNT(*) FILTER (WHERE status = 'DRAFT' AND governance_decision IS NULL) AS pending_review,
			COUNT(*) FILTER (WHERE status = 'DRAFT' AND review_priority = 'CRITICAL') AS critical_pending,
			COUNT(*) FILTER (WHERE review_due_at < NOW() AND status = 'DRAFT') AS breached_sla,
			COUNT(*) FILTER (WHERE review_due_at < NOW() + INTERVAL '24 hours' AND review_due_at >= NOW() AND status = 'DRAFT') AS at_risk_sla,
			COUNT(*) FILTER (WHERE has_conflict = TRUE AND status = 'DRAFT') AS with_conflicts
		FROM clinical_facts) cf,
		(SELECT
			COUNT(*) FILTER (WHERE governance_status = 'DRAFT') AS total_draft,
			COUNT(*) FILTER (WHERE governance_status = 'APPROVED') AS total_approved,
			COUNT(*) FILTER (WHERE governance_status = 'ACTIVE') AS total_active,
			COUNT(*) FILTER (WHERE governance_status IN ('SUPERSEDED','REJECTED')) AS total_superseded,
			COUNT(*) FILTER (WHERE governance_status = 'PENDING_REVIEW') AS pending_review,
			0 AS critical_pending,
			0 AS breached_sla,
			0 AS at_risk_sla,
			0 AS with_conflicts
		FROM derived_facts) df
	`

	metrics := &FactMetrics{GeneratedAt: time.Now()}
	err := s.db.QueryRowContext(ctx, query).Scan(
		&metrics.TotalDraft,
		&metrics.TotalApproved,
		&metrics.TotalActive,
		&metrics.TotalSuperseded,
		&metrics.PendingReview,
		&metrics.CriticalPending,
		&metrics.BreachedSLA,
		&metrics.AtRiskSLA,
		&metrics.WithConflicts,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get fact metrics: %w", err)
	}

	return metrics, nil
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func (s *FactStore) scanQueueItems(rows *sql.Rows) ([]*policy.QueueItem, error) {
	var items []*policy.QueueItem

	for rows.Next() {
		item := &policy.QueueItem{}
		var contentJSON []byte
		var conflictIDs pq.StringArray
		var reviewPriority sql.NullString
		var assignedReviewer sql.NullString
		var reviewDueAt sql.NullTime
		var confidenceScore sql.NullFloat64
		var daysUntilDue sql.NullFloat64

		err := rows.Scan(
			&item.FactID, &item.FactType, &item.RxCUI, &item.DrugName, &item.Scope,
			&contentJSON, &item.SourceType, &item.SourceID,
			&confidenceScore, &item.ConfidenceBand, &item.Status,
			&reviewPriority, &assignedReviewer, &reviewDueAt,
			&item.HasConflict, &conflictIDs, &item.AuthorityPriority,
			&item.CreatedAt, &item.PriorityRank, &daysUntilDue, &item.SLAStatus,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan queue item: %w", err)
		}

		if len(contentJSON) > 0 {
			json.Unmarshal(contentJSON, &item.Content)
		}
		if confidenceScore.Valid {
			item.ConfidenceScore = &confidenceScore.Float64
		}
		if reviewPriority.Valid {
			rp := policy.ReviewPriority(reviewPriority.String)
			item.ReviewPriority = &rp
		}
		if assignedReviewer.Valid {
			item.AssignedReviewer = &assignedReviewer.String
		}
		if reviewDueAt.Valid {
			item.ReviewDueAt = &reviewDueAt.Time
		}
		if daysUntilDue.Valid {
			item.DaysUntilDue = &daysUntilDue.Float64
		}
		for _, idStr := range conflictIDs {
			if id, err := uuid.Parse(idStr); err == nil {
				item.ConflictWithFactIDs = append(item.ConflictWithFactIDs, id)
			}
		}

		items = append(items, item)
	}

	return items, nil
}
