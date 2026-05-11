// Package postgres provides Postgres-backed implementations of the kb-32
// recommendation-craft engine's substrate interfaces.
//
// PostgresSubstrateClient is the production replacement for the Phase 2a
// in-memory placeholder. It reads from the kb-20-patient-profile schema and
// assembles a ClinicalSnapshot in the shape consumed by Stage 1 of the
// rendering pipeline (internal/context.Assembler).
//
// # Schema mapping (kb-20 → kb-32 ClinicalSnapshot)
//
// The Phase 2-completion plan referenced a hypothetical unified
// `patient_strata` table. That table does not exist in kb-20; instead the
// scoring instruments and care-intensity data are spread across several
// append-only tables, each with its own resident_ref keying and timestamp
// column. The actual mapping used here is:
//
//   EGFR                        → lab_entries (lab_type='egfr', most recent)
//   DBI                         → dbi_scores.score (most recent computed_at)
//   ACB                         → acb_scores.score (most recent computed_at)
//   CFS                         → cfs_scores.score (most recent assessed_at)
//   CareIntensity               → care_intensity_history.tag (most recent
//                                 effective_date), translated from kb-20's
//                                 vocabulary into kb-32's vocabulary:
//                                   active_treatment → active
//                                   rehabilitation   → active
//                                   comfort_focused  → comfort
//                                   palliative       → palliative
//                                 kb-32 also recognises "end_of_life" but
//                                 kb-20 has no equivalent enum value today.
//   RecentFall72h               → active_concerns row with
//                                 concern_type='post_fall_72h',
//                                 resolution_status='open'
//   RecentAdmission72h          → active_concerns row with
//                                 concern_type='post_hospital_discharge_72h',
//                                 resolution_status='open'
//   FamilyDistress              → NOT YET WIRED — kb-20 has no SDM/family
//                                 distress event source as of migration 022.
//   CapacityLapse               → capacity_assessments.outcome IN
//                                 ('impaired','unable_to_assess'), most
//                                 recent assessed_at
//   FrailtyStepIncrease30d      → derived from cfs_scores: latest score minus
//                                 most-recent score ≥ 30 days old, ≥ 2
//   RestrictivePracticeActive   → NOT YET WIRED — restrictive-practice
//                                 consent state machine (Plan 0.2) is not
//                                 yet present in kb-20 migrations.
//
// Both FamilyDistress and RestrictivePracticeActive are returned as the
// zero-value (false). Downstream Phase 2-completion tasks (Task 9 restraint
// substrate) will extend this client when those data sources land.
//
// # Error handling
//
// sql.ErrNoRows is treated as "no data available" and yields a zero-value
// field in ClinicalSnapshot rather than an error. Any other error from the
// database is returned to the caller after annotation with the field name.
package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	kb32ctx "github.com/cardiofit/kb32/internal/context"
)

// PostgresSubstrateClient reads kb-20-patient-profile data and returns it
// shaped as a kb-32 ClinicalSnapshot.
type PostgresSubstrateClient struct {
	db *sql.DB
}

// NewPostgresSubstrateClient constructs a PostgresSubstrateClient backed by
// the supplied *sql.DB handle. The handle is not validated at construction
// time; per-query errors will surface at SnapshotFor call sites instead.
func NewPostgresSubstrateClient(db *sql.DB) *PostgresSubstrateClient {
	return &PostgresSubstrateClient{db: db}
}

// SnapshotFor satisfies kb32ctx.SubstrateClient.
//
// Missing kb-20 data (sql.ErrNoRows on any individual query) is treated as
// "field unknown" and produces a zero-value entry in the returned snapshot.
// Any other database error short-circuits the assembly and is returned to
// the caller.
func (p *PostgresSubstrateClient) SnapshotFor(
	ctx context.Context, residentID uuid.UUID,
) (kb32ctx.ClinicalSnapshot, error) {
	snap := kb32ctx.ClinicalSnapshot{
		ResidentID: residentID,
		AssessedAt: time.Now().UTC(),
	}

	// --- Most recent CFS score + assessed_at -------------------------------
	// assessed_at doubles as the snapshot's AssessedAt anchor when available;
	// CFS is the most clinically-meaningful "when was this resident last
	// looked at" timestamp the substrate carries.
	var cfsAssessedAt sql.NullTime
	var cfsScore sql.NullInt64
	if err := p.db.QueryRowContext(ctx, `
		SELECT score, assessed_at
		FROM cfs_scores
		WHERE resident_ref = $1
		ORDER BY assessed_at DESC
		LIMIT 1
	`, residentID).Scan(&cfsScore, &cfsAssessedAt); err != nil && err != sql.ErrNoRows {
		return snap, fmt.Errorf("substrate: cfs_scores: %w", err)
	}
	if cfsScore.Valid {
		snap.CFS = int(cfsScore.Int64)
	}
	if cfsAssessedAt.Valid {
		snap.AssessedAt = cfsAssessedAt.Time
	}

	// --- Most recent DBI score --------------------------------------------
	var dbiScore sql.NullFloat64
	if err := p.db.QueryRowContext(ctx, `
		SELECT score
		FROM dbi_scores
		WHERE resident_ref = $1
		ORDER BY computed_at DESC
		LIMIT 1
	`, residentID).Scan(&dbiScore); err != nil && err != sql.ErrNoRows {
		return snap, fmt.Errorf("substrate: dbi_scores: %w", err)
	}
	if dbiScore.Valid {
		snap.DBI = dbiScore.Float64
	}

	// --- Most recent ACB score --------------------------------------------
	var acbScore sql.NullInt64
	if err := p.db.QueryRowContext(ctx, `
		SELECT score
		FROM acb_scores
		WHERE resident_ref = $1
		ORDER BY computed_at DESC
		LIMIT 1
	`, residentID).Scan(&acbScore); err != nil && err != sql.ErrNoRows {
		return snap, fmt.Errorf("substrate: acb_scores: %w", err)
	}
	if acbScore.Valid {
		snap.ACB = int(acbScore.Int64)
	}

	// --- Most recent eGFR (lab_entries.lab_type='egfr') -------------------
	// lab_entries.patient_id is VARCHAR — keyed by the resident UUID's
	// canonical string form rather than a uuid column.
	var egfrValue sql.NullFloat64
	if err := p.db.QueryRowContext(ctx, `
		SELECT value
		FROM lab_entries
		WHERE patient_id = $1
		  AND lab_type = 'egfr'
		  AND validation_status <> 'REJECTED'
		ORDER BY measured_at DESC
		LIMIT 1
	`, residentID.String()).Scan(&egfrValue); err != nil && err != sql.ErrNoRows {
		return snap, fmt.Errorf("substrate: lab_entries(egfr): %w", err)
	}
	if egfrValue.Valid {
		snap.EGFR = egfrValue.Float64
	}

	// --- Care intensity (most recent care_intensity_history.tag) ----------
	var careTag sql.NullString
	if err := p.db.QueryRowContext(ctx, `
		SELECT tag
		FROM care_intensity_history
		WHERE resident_ref = $1
		ORDER BY effective_date DESC
		LIMIT 1
	`, residentID).Scan(&careTag); err != nil && err != sql.ErrNoRows {
		return snap, fmt.Errorf("substrate: care_intensity_history: %w", err)
	}
	if careTag.Valid {
		snap.CareIntensity = translateCareIntensity(careTag.String)
	}

	// --- RecentFall72h (open post_fall_72h concern) -----------------------
	if err := p.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM active_concerns
			WHERE resident_id = $1
			  AND concern_type = 'post_fall_72h'
			  AND resolution_status = 'open'
			  AND started_at >= NOW() - INTERVAL '72 hours'
		)
	`, residentID).Scan(&snap.RecentFall72h); err != nil {
		return snap, fmt.Errorf("substrate: active_concerns(post_fall_72h): %w", err)
	}

	// --- RecentAdmission72h (open post_hospital_discharge_72h concern) ----
	if err := p.db.QueryRowContext(ctx, `
		SELECT EXISTS(
			SELECT 1 FROM active_concerns
			WHERE resident_id = $1
			  AND concern_type = 'post_hospital_discharge_72h'
			  AND resolution_status = 'open'
			  AND started_at >= NOW() - INTERVAL '72 hours'
		)
	`, residentID).Scan(&snap.RecentAdmission72h); err != nil {
		return snap, fmt.Errorf("substrate: active_concerns(post_hospital_discharge_72h): %w", err)
	}

	// --- CapacityLapse (most recent capacity_assessment outcome) ----------
	var capacityOutcome sql.NullString
	if err := p.db.QueryRowContext(ctx, `
		SELECT outcome
		FROM capacity_assessments
		WHERE resident_ref = $1
		ORDER BY assessed_at DESC
		LIMIT 1
	`, residentID).Scan(&capacityOutcome); err != nil && err != sql.ErrNoRows {
		return snap, fmt.Errorf("substrate: capacity_assessments: %w", err)
	}
	if capacityOutcome.Valid &&
		(capacityOutcome.String == "impaired" || capacityOutcome.String == "unable_to_assess") {
		snap.CapacityLapse = true
	}

	// --- FrailtyStepIncrease30d (CFS delta over 30d ≥ 2) ------------------
	// Compares the most-recent CFS score against the most-recent CFS score
	// recorded ≥ 30 days ago. A monotone increase of ≥ 2 steps is the
	// trigger condition documented in the ClinicalSnapshot godoc.
	if cfsScore.Valid {
		var prior sql.NullInt64
		if err := p.db.QueryRowContext(ctx, `
			SELECT score
			FROM cfs_scores
			WHERE resident_ref = $1
			  AND assessed_at <= NOW() - INTERVAL '30 days'
			ORDER BY assessed_at DESC
			LIMIT 1
		`, residentID).Scan(&prior); err != nil && err != sql.ErrNoRows {
			return snap, fmt.Errorf("substrate: cfs_scores(prior 30d): %w", err)
		}
		if prior.Valid && (cfsScore.Int64-prior.Int64) >= 2 {
			snap.FrailtyStepIncrease30d = true
		}
	}

	// FamilyDistress and RestrictivePracticeActive: see package doc — kb-20
	// has no substrate for these signals at migration 022. Left at zero
	// values until Task 9 lands the restraint-substrate joins.

	return snap, nil
}

// translateCareIntensity maps kb-20 care-intensity tag vocabulary onto the
// vocabulary kb-32's ClinicalSnapshot.CareIntensity expects. Unknown tags
// pass through unchanged so a later vocabulary expansion does not silently
// erase data.
func translateCareIntensity(kb20Tag string) string {
	switch kb20Tag {
	case "active_treatment", "rehabilitation":
		return "active"
	case "comfort_focused":
		return "comfort"
	case "palliative":
		return "palliative"
	default:
		return kb20Tag
	}
}

// Compile-time guarantee that PostgresSubstrateClient satisfies the
// SubstrateClient port consumed by internal/context.Assembler.
var _ kb32ctx.SubstrateClient = (*PostgresSubstrateClient)(nil)
