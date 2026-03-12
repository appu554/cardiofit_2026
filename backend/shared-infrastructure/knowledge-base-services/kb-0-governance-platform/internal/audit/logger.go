// Package audit provides immutable audit logging for all Knowledge Base governance actions.
package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"kb-0-governance-platform/internal/models"
)

// =============================================================================
// AUDIT LOGGER
// =============================================================================

// Logger provides immutable audit logging for governance actions.
type Logger struct {
	db *sql.DB
}

// NewLogger creates a new audit logger.
func NewLogger(db *sql.DB) *Logger {
	return &Logger{db: db}
}

// Log records an audit entry to the immutable audit log.
func (l *Logger) Log(ctx context.Context, entry *models.AuditEntry) error {
	if entry.ID == "" {
		entry.ID = generateAuditID()
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	checklistJSON, _ := json.Marshal(entry.Checklist)
	attestationsJSON, _ := json.Marshal(entry.Attestations)

	query := `
		INSERT INTO audit_log (
			audit_id, timestamp, action,
			actor_id, actor_external_id, actor_role, actor_name, actor_credentials,
			item_id, kb, item_version,
			previous_state, new_state,
			decision, notes, checklist, attestations,
			ip_address, session_id, user_agent,
			content_hash
		) VALUES (
			$1, $2, $3,
			$4, $5, $6, $7, $8,
			$9, $10, $11,
			$12, $13,
			$14, $15, $16, $17,
			$18, $19, $20,
			$21
		)
	`

	_, err := l.db.ExecContext(ctx, query,
		entry.ID, entry.Timestamp, entry.Action,
		nil, entry.ActorID, entry.ActorRole, entry.ActorName, entry.Credentials,
		entry.ItemID, entry.KB, entry.ItemVersion,
		entry.PreviousState, entry.NewState,
		entry.Decision, entry.Notes, checklistJSON, attestationsJSON,
		entry.IPAddress, entry.SessionID, entry.UserAgent,
		entry.ContentHash,
	)
	if err != nil {
		return fmt.Errorf("failed to insert audit entry: %w", err)
	}

	return nil
}

// GetAuditTrail returns the audit trail for a specific item.
func (l *Logger) GetAuditTrail(ctx context.Context, itemID string) ([]*models.AuditEntry, error) {
	query := `
		SELECT
			audit_id, timestamp, action,
			actor_external_id, actor_role, actor_name, actor_credentials,
			item_id, kb, item_version,
			previous_state, new_state,
			decision, notes, checklist, attestations,
			ip_address, session_id, user_agent,
			content_hash
		FROM audit_log
		WHERE item_id = $1
		ORDER BY timestamp ASC
	`

	rows, err := l.db.QueryContext(ctx, query, itemID)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit trail: %w", err)
	}
	defer rows.Close()

	var entries []*models.AuditEntry
	for rows.Next() {
		entry := &models.AuditEntry{}
		var checklistJSON, attestationsJSON []byte
		var ipAddress, sessionID, userAgent sql.NullString
		var previousState sql.NullString

		err := rows.Scan(
			&entry.ID, &entry.Timestamp, &entry.Action,
			&entry.ActorID, &entry.ActorRole, &entry.ActorName, &entry.Credentials,
			&entry.ItemID, &entry.KB, &entry.ItemVersion,
			&previousState, &entry.NewState,
			&entry.Decision, &entry.Notes, &checklistJSON, &attestationsJSON,
			&ipAddress, &sessionID, &userAgent,
			&entry.ContentHash,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit entry: %w", err)
		}

		if previousState.Valid {
			entry.PreviousState = models.ItemState(previousState.String)
		}
		if ipAddress.Valid {
			entry.IPAddress = ipAddress.String
		}
		if sessionID.Valid {
			entry.SessionID = sessionID.String
		}
		if userAgent.Valid {
			entry.UserAgent = userAgent.String
		}
		if len(checklistJSON) > 0 {
			json.Unmarshal(checklistJSON, &entry.Checklist)
		}
		if len(attestationsJSON) > 0 {
			json.Unmarshal(attestationsJSON, &entry.Attestations)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// GetAuditByKB returns audit entries for a specific KB.
func (l *Logger) GetAuditByKB(ctx context.Context, kb models.KB, since time.Time, limit int) ([]*models.AuditEntry, error) {
	query := `
		SELECT
			audit_id, timestamp, action,
			actor_external_id, actor_role, actor_name,
			item_id, kb, item_version,
			previous_state, new_state,
			decision, notes
		FROM audit_log
		WHERE kb = $1 AND timestamp >= $2
		ORDER BY timestamp DESC
		LIMIT $3
	`

	rows, err := l.db.QueryContext(ctx, query, kb, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query KB audit: %w", err)
	}
	defer rows.Close()

	var entries []*models.AuditEntry
	for rows.Next() {
		entry := &models.AuditEntry{}
		var previousState sql.NullString

		err := rows.Scan(
			&entry.ID, &entry.Timestamp, &entry.Action,
			&entry.ActorID, &entry.ActorRole, &entry.ActorName,
			&entry.ItemID, &entry.KB, &entry.ItemVersion,
			&previousState, &entry.NewState,
			&entry.Decision, &entry.Notes,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit entry: %w", err)
		}

		if previousState.Valid {
			entry.PreviousState = models.ItemState(previousState.String)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

// GetAuditStats returns audit statistics for compliance reporting.
func (l *Logger) GetAuditStats(ctx context.Context, kb models.KB, since time.Time) (*AuditStats, error) {
	query := `
		SELECT
			COUNT(*) FILTER (WHERE action = 'ITEM_CREATED') AS created_count,
			COUNT(*) FILTER (WHERE action = 'ITEM_REVIEWED') AS reviewed_count,
			COUNT(*) FILTER (WHERE action = 'ITEM_APPROVED') AS approved_count,
			COUNT(*) FILTER (WHERE action = 'ITEM_ACTIVATED') AS activated_count,
			COUNT(*) FILTER (WHERE action = 'ITEM_RETIRED') AS retired_count,
			COUNT(*) FILTER (WHERE action = 'ITEM_REJECTED') AS rejected_count,
			COUNT(*) FILTER (WHERE action = 'EMERGENCY_OVERRIDE') AS emergency_count,
			COUNT(DISTINCT actor_external_id) AS unique_actors,
			AVG(EXTRACT(EPOCH FROM (
				(SELECT MIN(al2.timestamp) FROM audit_log al2
				 WHERE al2.item_id = audit_log.item_id AND al2.action = 'ITEM_ACTIVATED')
				- timestamp
			))) / 3600 AS avg_time_to_activation_hours
		FROM audit_log
		WHERE kb = $1 AND timestamp >= $2 AND action = 'ITEM_CREATED'
	`

	stats := &AuditStats{KB: kb, Since: since}
	err := l.db.QueryRowContext(ctx, query, kb, since).Scan(
		&stats.CreatedCount,
		&stats.ReviewedCount,
		&stats.ApprovedCount,
		&stats.ActivatedCount,
		&stats.RetiredCount,
		&stats.RejectedCount,
		&stats.EmergencyCount,
		&stats.UniqueActors,
		&stats.AvgTimeToActivationHours,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get audit stats: %w", err)
	}

	return stats, nil
}

// AuditStats contains audit statistics for compliance reporting.
type AuditStats struct {
	KB                       models.KB `json:"kb"`
	Since                    time.Time `json:"since"`
	CreatedCount             int       `json:"created_count"`
	ReviewedCount            int       `json:"reviewed_count"`
	ApprovedCount            int       `json:"approved_count"`
	ActivatedCount           int       `json:"activated_count"`
	RetiredCount             int       `json:"retired_count"`
	RejectedCount            int       `json:"rejected_count"`
	EmergencyCount           int       `json:"emergency_count"`
	UniqueActors             int       `json:"unique_actors"`
	AvgTimeToActivationHours float64   `json:"avg_time_to_activation_hours"`
}

// =============================================================================
// COMPLIANCE EXPORT
// =============================================================================

// ExportForRegulator exports audit data in regulatory format.
func (l *Logger) ExportForRegulator(ctx context.Context, kb models.KB, since, until time.Time) (*RegulatoryExport, error) {
	entries, err := l.GetAuditByKB(ctx, kb, since, 10000)
	if err != nil {
		return nil, err
	}

	stats, err := l.GetAuditStats(ctx, kb, since)
	if err != nil {
		return nil, err
	}

	export := &RegulatoryExport{
		KB:              kb,
		ExportedAt:      time.Now(),
		ReportingPeriod: ReportingPeriod{Start: since, End: until},
		Summary:         *stats,
		Entries:         entries,
	}

	return export, nil
}

// RegulatoryExport contains data formatted for regulatory audits.
type RegulatoryExport struct {
	KB              models.KB          `json:"kb"`
	ExportedAt      time.Time          `json:"exported_at"`
	ReportingPeriod ReportingPeriod    `json:"reporting_period"`
	Summary         AuditStats         `json:"summary"`
	Entries         []*models.AuditEntry `json:"entries"`
}

// ReportingPeriod defines the time range for the export.
type ReportingPeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// =============================================================================
// HELPERS
// =============================================================================

func generateAuditID() string {
	return fmt.Sprintf("aud_%d_%s", time.Now().UnixNano(), randomHex(4))
}

func randomHex(n int) string {
	// Simple random hex generator
	const hex = "0123456789abcdef"
	b := make([]byte, n)
	for i := range b {
		b[i] = hex[time.Now().UnixNano()%int64(len(hex))]
	}
	return string(b)
}
