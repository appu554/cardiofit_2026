// Package storage — ScoringStore is the kb-20 implementation of the
// CFS / AKPS / DBI / ACB persistence + service layer (Wave 2.6 of the
// Layer 2 substrate plan; Layer 2 doc §2.4 / §2.6). It owns the four
// scoring tables + four current views created by migration 018, plus
// the dbi_drug_weights / acb_drug_weights seed tables.
//
// Service-layer behaviour (per plan):
//
//   - CFS / AKPS are clinician-entered. Each Create* call writes one
//     EvidenceTrace node tagged with state_machine=ClinicalState. When
//     the score crosses the review threshold (CFS>=7 or AKPS<=40) the
//     state_change_type is care_intensity_review_suggested and the
//     CareIntensityReviewHint is returned in the ScoringResult; the
//     substrate NEVER writes a care_intensity_history row from a score
//     (Layer 2 doc §2.4 line 540-547).
//   - DBI / ACB are computed. RecomputeDrugBurden pulls the resident's
//     active MedicineUse list, runs the pure scoring.ComputeDBI /
//     scoring.ComputeACB calculators against the seed-table-backed
//     DrugWeightLookup, and writes new dbi_scores + acb_scores rows.
//     Recompute is best-effort: failure MUST NOT fail any caller.
//
// The history is append-only — never UPDATE rows.
package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/cardiofit/shared/v2_substrate/interfaces"
	"github.com/cardiofit/shared/v2_substrate/models"
	"github.com/cardiofit/shared/v2_substrate/scoring"
	"github.com/cardiofit/shared/v2_substrate/validation"
)

// ScoringStore implements interfaces.ScoringStore. Depends on a *sql.DB
// for the four scoring tables + the seed weight tables, and a
// *V2SubstrateStore handle for EvidenceTrace writes + the active
// MedicineUse list used by RecomputeDrugBurden. Both must be backed by
// the same *sql.DB.
type ScoringStore struct {
	db     *sql.DB
	v2     *V2SubstrateStore
	lookup scoring.DrugWeightLookup
	now    func() time.Time
}

// NewScoringStore wires a *sql.DB + *V2SubstrateStore into the scoring
// persistence contract. The DrugWeightLookup is a DB-backed lookup over
// the dbi_drug_weights + acb_drug_weights seed tables (constructed
// internally from db).
func NewScoringStore(db *sql.DB, v2 *V2SubstrateStore) *ScoringStore {
	return &ScoringStore{
		db:     db,
		v2:     v2,
		lookup: NewSeedDrugWeightLookup(db),
		now:    func() time.Time { return time.Now().UTC() },
	}
}

// WithClock overrides the store's clock so tests can drive deterministic
// timestamps.
func (s *ScoringStore) WithClock(now func() time.Time) *ScoringStore {
	if now != nil {
		s.now = now
	}
	return s
}

// WithDrugWeightLookup overrides the DrugWeightLookup. Tests that don't
// want to depend on the seed-table contents inject a static lookup.
func (s *ScoringStore) WithDrugWeightLookup(l scoring.DrugWeightLookup) *ScoringStore {
	if l != nil {
		s.lookup = l
	}
	return s
}

// ---------------------------------------------------------------------------
// CFS
// ---------------------------------------------------------------------------

const cfsScoreColumns = `id, resident_ref, assessed_at, assessor_role_ref,
       instrument_version, score, rationale, created_at`

func scanCFSScore(sc rowScanner) (models.CFSScore, error) {
	var (
		c         models.CFSScore
		rationale sql.NullString
	)
	if err := sc.Scan(
		&c.ID, &c.ResidentRef, &c.AssessedAt, &c.AssessorRoleRef,
		&c.InstrumentVersion, &c.Score, &rationale, &c.CreatedAt,
	); err != nil {
		return models.CFSScore{}, err
	}
	if rationale.Valid {
		c.Rationale = rationale.String
	}
	return c, nil
}

// CreateCFSScore validates, persists, writes the EvidenceTrace node, and
// (when Score>=7) attaches a CareIntensityReviewHint to the result.
func (s *ScoringStore) CreateCFSScore(ctx context.Context, in models.CFSScore) (*interfaces.ScoringResult, error) {
	if in.ID == uuid.Nil {
		in.ID = uuid.New()
	}
	if in.AssessedAt.IsZero() {
		in.AssessedAt = s.now()
	}
	if err := validation.ValidateCFSScore(in); err != nil {
		return nil, fmt.Errorf("validate cfs: %w", err)
	}
	const q = `
		INSERT INTO cfs_scores
			(id, resident_ref, assessed_at, assessor_role_ref,
			 instrument_version, score, rationale, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`
	if _, err := s.db.ExecContext(ctx, q,
		in.ID, in.ResidentRef, in.AssessedAt, in.AssessorRoleRef,
		in.InstrumentVersion, in.Score, nilIfEmpty(in.Rationale),
	); err != nil {
		return nil, fmt.Errorf("insert cfs_score: %w", err)
	}
	persisted, err := s.GetCFSScore(ctx, in.ID)
	if err != nil {
		return nil, fmt.Errorf("reload cfs: %w", err)
	}
	hint := s.maybeBuildCFSHint(persisted)
	traceNodeID, err := s.writeScoreEvidenceTrace(
		ctx, persisted.ResidentRef, persisted.AssessorRoleRef, persisted.AssessedAt,
		"CFSScore", persisted.ID,
		fmt.Sprintf("CFS score=%d (%s)", persisted.Score, scoring.CFSScoreLabel(persisted.Score)),
		hint != nil,
	)
	if err != nil {
		return nil, fmt.Errorf("write evidence trace: %w", err)
	}
	return &interfaces.ScoringResult{
		CFSScore:             persisted,
		CareIntensityHint:    hint,
		EvidenceTraceNodeRef: traceNodeID,
	}, nil
}

func (s *ScoringStore) maybeBuildCFSHint(c *models.CFSScore) *interfaces.CareIntensityReviewHint {
	if c == nil || !models.CFSScoreShouldHintCareIntensityReview(c.Score) {
		return nil
	}
	return &interfaces.CareIntensityReviewHint{
		Instrument: "CFS",
		Score:      c.Score,
		ScoreRef:   c.ID,
		Reason:     fmt.Sprintf("CFS>=%d — consider care intensity review (Layer 2 §2.4)", models.CFSCareIntensityReviewThreshold),
	}
}

// GetCFSScore reads a single cfs_scores row by id.
func (s *ScoringStore) GetCFSScore(ctx context.Context, id uuid.UUID) (*models.CFSScore, error) {
	q := `SELECT ` + cfsScoreColumns + ` FROM cfs_scores WHERE id = $1`
	c, err := scanCFSScore(s.db.QueryRowContext(ctx, q, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get cfs_score %s: %w", id, interfaces.ErrNotFound)
		}
		return nil, fmt.Errorf("get cfs_score %s: %w", id, err)
	}
	return &c, nil
}

// GetCurrentCFSScore returns the latest CFS row for residentRef via cfs_current.
func (s *ScoringStore) GetCurrentCFSScore(ctx context.Context, residentRef uuid.UUID) (*models.CFSScore, error) {
	q := `SELECT ` + cfsScoreColumns + ` FROM cfs_current WHERE resident_ref = $1`
	c, err := scanCFSScore(s.db.QueryRowContext(ctx, q, residentRef))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get current cfs for %s: %w", residentRef, interfaces.ErrNotFound)
		}
		return nil, fmt.Errorf("get current cfs for %s: %w", residentRef, err)
	}
	return &c, nil
}

// ListCFSHistory returns the full CFS history for residentRef, newest-first.
func (s *ScoringStore) ListCFSHistory(ctx context.Context, residentRef uuid.UUID) ([]models.CFSScore, error) {
	q := `SELECT ` + cfsScoreColumns + ` FROM cfs_scores WHERE resident_ref = $1 ORDER BY assessed_at DESC`
	rows, err := s.db.QueryContext(ctx, q, residentRef)
	if err != nil {
		return nil, fmt.Errorf("list cfs history: %w", err)
	}
	defer rows.Close()
	var out []models.CFSScore
	for rows.Next() {
		c, err := scanCFSScore(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// AKPS
// ---------------------------------------------------------------------------

const akpsScoreColumns = `id, resident_ref, assessed_at, assessor_role_ref,
       instrument_version, score, rationale, created_at`

func scanAKPSScore(sc rowScanner) (models.AKPSScore, error) {
	var (
		a         models.AKPSScore
		rationale sql.NullString
	)
	if err := sc.Scan(
		&a.ID, &a.ResidentRef, &a.AssessedAt, &a.AssessorRoleRef,
		&a.InstrumentVersion, &a.Score, &rationale, &a.CreatedAt,
	); err != nil {
		return models.AKPSScore{}, err
	}
	if rationale.Valid {
		a.Rationale = rationale.String
	}
	return a, nil
}

// CreateAKPSScore validates, persists, writes the EvidenceTrace node, and
// (when Score<=40) attaches a CareIntensityReviewHint to the result.
func (s *ScoringStore) CreateAKPSScore(ctx context.Context, in models.AKPSScore) (*interfaces.ScoringResult, error) {
	if in.ID == uuid.Nil {
		in.ID = uuid.New()
	}
	if in.AssessedAt.IsZero() {
		in.AssessedAt = s.now()
	}
	if err := validation.ValidateAKPSScore(in); err != nil {
		return nil, fmt.Errorf("validate akps: %w", err)
	}
	const q = `
		INSERT INTO akps_scores
			(id, resident_ref, assessed_at, assessor_role_ref,
			 instrument_version, score, rationale, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`
	if _, err := s.db.ExecContext(ctx, q,
		in.ID, in.ResidentRef, in.AssessedAt, in.AssessorRoleRef,
		in.InstrumentVersion, in.Score, nilIfEmpty(in.Rationale),
	); err != nil {
		return nil, fmt.Errorf("insert akps_score: %w", err)
	}
	persisted, err := s.GetAKPSScore(ctx, in.ID)
	if err != nil {
		return nil, fmt.Errorf("reload akps: %w", err)
	}
	hint := s.maybeBuildAKPSHint(persisted)
	traceNodeID, err := s.writeScoreEvidenceTrace(
		ctx, persisted.ResidentRef, persisted.AssessorRoleRef, persisted.AssessedAt,
		"AKPSScore", persisted.ID,
		fmt.Sprintf("AKPS score=%d (%s)", persisted.Score, scoring.AKPSScoreLabel(persisted.Score)),
		hint != nil,
	)
	if err != nil {
		return nil, fmt.Errorf("write evidence trace: %w", err)
	}
	return &interfaces.ScoringResult{
		AKPSScore:            persisted,
		CareIntensityHint:    hint,
		EvidenceTraceNodeRef: traceNodeID,
	}, nil
}

func (s *ScoringStore) maybeBuildAKPSHint(a *models.AKPSScore) *interfaces.CareIntensityReviewHint {
	if a == nil || !models.AKPSScoreShouldHintCareIntensityReview(a.Score) {
		return nil
	}
	return &interfaces.CareIntensityReviewHint{
		Instrument: "AKPS",
		Score:      a.Score,
		ScoreRef:   a.ID,
		Reason:     fmt.Sprintf("AKPS<=%d — consider care intensity review (Layer 2 §2.4)", models.AKPSCareIntensityReviewThreshold),
	}
}

// GetAKPSScore reads a single akps_scores row by id.
func (s *ScoringStore) GetAKPSScore(ctx context.Context, id uuid.UUID) (*models.AKPSScore, error) {
	q := `SELECT ` + akpsScoreColumns + ` FROM akps_scores WHERE id = $1`
	a, err := scanAKPSScore(s.db.QueryRowContext(ctx, q, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get akps_score %s: %w", id, interfaces.ErrNotFound)
		}
		return nil, fmt.Errorf("get akps_score %s: %w", id, err)
	}
	return &a, nil
}

// GetCurrentAKPSScore returns the latest AKPS row for residentRef via akps_current.
func (s *ScoringStore) GetCurrentAKPSScore(ctx context.Context, residentRef uuid.UUID) (*models.AKPSScore, error) {
	q := `SELECT ` + akpsScoreColumns + ` FROM akps_current WHERE resident_ref = $1`
	a, err := scanAKPSScore(s.db.QueryRowContext(ctx, q, residentRef))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get current akps for %s: %w", residentRef, interfaces.ErrNotFound)
		}
		return nil, fmt.Errorf("get current akps for %s: %w", residentRef, err)
	}
	return &a, nil
}

// ListAKPSHistory returns the full AKPS history for residentRef, newest-first.
func (s *ScoringStore) ListAKPSHistory(ctx context.Context, residentRef uuid.UUID) ([]models.AKPSScore, error) {
	q := `SELECT ` + akpsScoreColumns + ` FROM akps_scores WHERE resident_ref = $1 ORDER BY assessed_at DESC`
	rows, err := s.db.QueryContext(ctx, q, residentRef)
	if err != nil {
		return nil, fmt.Errorf("list akps history: %w", err)
	}
	defer rows.Close()
	var out []models.AKPSScore
	for rows.Next() {
		a, err := scanAKPSScore(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// DBI
// ---------------------------------------------------------------------------

const dbiScoreColumns = `id, resident_ref, computed_at, score,
       anticholinergic_component, sedative_component,
       computation_inputs, unknown_drugs, created_at`

func scanDBIScore(sc rowScanner) (models.DBIScore, error) {
	var (
		d        models.DBIScore
		inputs   pq.StringArray
		unknown  pq.StringArray
	)
	if err := sc.Scan(
		&d.ID, &d.ResidentRef, &d.ComputedAt, &d.Score,
		&d.AnticholinergicComponent, &d.SedativeComponent,
		&inputs, &unknown, &d.CreatedAt,
	); err != nil {
		return models.DBIScore{}, err
	}
	d.ComputationInputs = parseStringUUIDs(inputs)
	d.UnknownDrugs = []string(unknown)
	return d, nil
}

func (s *ScoringStore) insertDBIScore(ctx context.Context, d models.DBIScore) error {
	const q = `
		INSERT INTO dbi_scores
			(id, resident_ref, computed_at, score,
			 anticholinergic_component, sedative_component,
			 computation_inputs, unknown_drugs, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())`
	inputs := make([]string, 0, len(d.ComputationInputs))
	for _, id := range d.ComputationInputs {
		inputs = append(inputs, id.String())
	}
	unknown := pq.StringArray(d.UnknownDrugs)
	if d.UnknownDrugs == nil {
		unknown = pq.StringArray{}
	}
	if _, err := s.db.ExecContext(ctx, q,
		d.ID, d.ResidentRef, d.ComputedAt, d.Score,
		d.AnticholinergicComponent, d.SedativeComponent,
		pq.Array(inputs), unknown,
	); err != nil {
		return fmt.Errorf("insert dbi_score: %w", err)
	}
	return nil
}

// GetCurrentDBIScore returns the latest DBI row for residentRef via dbi_current.
func (s *ScoringStore) GetCurrentDBIScore(ctx context.Context, residentRef uuid.UUID) (*models.DBIScore, error) {
	q := `SELECT ` + dbiScoreColumns + ` FROM dbi_current WHERE resident_ref = $1`
	d, err := scanDBIScore(s.db.QueryRowContext(ctx, q, residentRef))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get current dbi for %s: %w", residentRef, interfaces.ErrNotFound)
		}
		return nil, fmt.Errorf("get current dbi for %s: %w", residentRef, err)
	}
	return &d, nil
}

// ListDBIHistory returns the full DBI history for residentRef, newest-first.
func (s *ScoringStore) ListDBIHistory(ctx context.Context, residentRef uuid.UUID) ([]models.DBIScore, error) {
	q := `SELECT ` + dbiScoreColumns + ` FROM dbi_scores WHERE resident_ref = $1 ORDER BY computed_at DESC`
	rows, err := s.db.QueryContext(ctx, q, residentRef)
	if err != nil {
		return nil, fmt.Errorf("list dbi history: %w", err)
	}
	defer rows.Close()
	var out []models.DBIScore
	for rows.Next() {
		d, err := scanDBIScore(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// ACB
// ---------------------------------------------------------------------------

const acbScoreColumns = `id, resident_ref, computed_at, score,
       computation_inputs, unknown_drugs, created_at`

func scanACBScore(sc rowScanner) (models.ACBScore, error) {
	var (
		a       models.ACBScore
		inputs  pq.StringArray
		unknown pq.StringArray
	)
	if err := sc.Scan(
		&a.ID, &a.ResidentRef, &a.ComputedAt, &a.Score,
		&inputs, &unknown, &a.CreatedAt,
	); err != nil {
		return models.ACBScore{}, err
	}
	a.ComputationInputs = parseStringUUIDs(inputs)
	a.UnknownDrugs = []string(unknown)
	return a, nil
}

func (s *ScoringStore) insertACBScore(ctx context.Context, a models.ACBScore) error {
	const q = `
		INSERT INTO acb_scores
			(id, resident_ref, computed_at, score,
			 computation_inputs, unknown_drugs, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW())`
	inputs := make([]string, 0, len(a.ComputationInputs))
	for _, id := range a.ComputationInputs {
		inputs = append(inputs, id.String())
	}
	unknown := pq.StringArray(a.UnknownDrugs)
	if a.UnknownDrugs == nil {
		unknown = pq.StringArray{}
	}
	if _, err := s.db.ExecContext(ctx, q,
		a.ID, a.ResidentRef, a.ComputedAt, a.Score,
		pq.Array(inputs), unknown,
	); err != nil {
		return fmt.Errorf("insert acb_score: %w", err)
	}
	return nil
}

// GetCurrentACBScore returns the latest ACB row for residentRef via acb_current.
func (s *ScoringStore) GetCurrentACBScore(ctx context.Context, residentRef uuid.UUID) (*models.ACBScore, error) {
	q := `SELECT ` + acbScoreColumns + ` FROM acb_current WHERE resident_ref = $1`
	a, err := scanACBScore(s.db.QueryRowContext(ctx, q, residentRef))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get current acb for %s: %w", residentRef, interfaces.ErrNotFound)
		}
		return nil, fmt.Errorf("get current acb for %s: %w", residentRef, err)
	}
	return &a, nil
}

// ListACBHistory returns the full ACB history for residentRef, newest-first.
func (s *ScoringStore) ListACBHistory(ctx context.Context, residentRef uuid.UUID) ([]models.ACBScore, error) {
	q := `SELECT ` + acbScoreColumns + ` FROM acb_scores WHERE resident_ref = $1 ORDER BY computed_at DESC`
	rows, err := s.db.QueryContext(ctx, q, residentRef)
	if err != nil {
		return nil, fmt.Errorf("list acb history: %w", err)
	}
	defer rows.Close()
	var out []models.ACBScore
	for rows.Next() {
		a, err := scanACBScore(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// Recompute orchestration
// ---------------------------------------------------------------------------

// recomputeMedicineUseListLimit caps the number of MedicineUse rows we
// pull for the recompute. Aged-care residents typically carry 5-15
// active medications; 200 is a generous safety cap that prevents a
// pathological row from running away.
const recomputeMedicineUseListLimit = 200

// RecomputeDrugBurden pulls the resident's current MedicineUse list,
// runs the pure DBI + ACB calculators, and persists fresh dbi_scores +
// acb_scores rows. Best-effort — caller decides whether to surface an
// error or swallow it.
func (s *ScoringStore) RecomputeDrugBurden(ctx context.Context, residentRef uuid.UUID) (*interfaces.DrugBurdenRecomputeResult, error) {
	meds, err := s.v2.ListMedicineUsesByResident(ctx, residentRef, recomputeMedicineUseListLimit, 0)
	if err != nil {
		return nil, fmt.Errorf("load medicine_uses: %w", err)
	}
	now := s.now()

	dbi, err := scoring.ComputeDBI(ctx, meds, s.lookup, residentRef, now)
	if err != nil {
		return nil, fmt.Errorf("compute dbi: %w", err)
	}
	if err := s.insertDBIScore(ctx, dbi); err != nil {
		return nil, err
	}

	acb, err := scoring.ComputeACB(ctx, meds, s.lookup, residentRef, now)
	if err != nil {
		return nil, fmt.Errorf("compute acb: %w", err)
	}
	if err := s.insertACBScore(ctx, acb); err != nil {
		return nil, err
	}

	persistedDBI, err := s.GetCurrentDBIScore(ctx, residentRef)
	if err != nil {
		return nil, fmt.Errorf("reload dbi: %w", err)
	}
	persistedACB, err := s.GetCurrentACBScore(ctx, residentRef)
	if err != nil {
		return nil, fmt.Errorf("reload acb: %w", err)
	}
	return &interfaces.DrugBurdenRecomputeResult{
		DBIScore: persistedDBI,
		ACBScore: persistedACB,
	}, nil
}

// CurrentScoresByResident reads the latest of each instrument from the
// four current views. ErrNotFound is squashed to nil so the response
// payload can show "no row yet" cleanly.
func (s *ScoringStore) CurrentScoresByResident(ctx context.Context, residentRef uuid.UUID) (*interfaces.CurrentScores, error) {
	out := &interfaces.CurrentScores{}
	if cfs, err := s.GetCurrentCFSScore(ctx, residentRef); err == nil {
		out.CFS = cfs
	} else if !errors.Is(err, interfaces.ErrNotFound) {
		return nil, err
	}
	if akps, err := s.GetCurrentAKPSScore(ctx, residentRef); err == nil {
		out.AKPS = akps
	} else if !errors.Is(err, interfaces.ErrNotFound) {
		return nil, err
	}
	if dbi, err := s.GetCurrentDBIScore(ctx, residentRef); err == nil {
		out.DBI = dbi
	} else if !errors.Is(err, interfaces.ErrNotFound) {
		return nil, err
	}
	if acb, err := s.GetCurrentACBScore(ctx, residentRef); err == nil {
		out.ACB = acb
	} else if !errors.Is(err, interfaces.ErrNotFound) {
		return nil, err
	}
	return out, nil
}

// ---------------------------------------------------------------------------
// EvidenceTrace helper
// ---------------------------------------------------------------------------

// writeScoreEvidenceTrace records one EvidenceTrace node per scoring
// instrument write. State machine is always ClinicalState. State change
// type is care_intensity_review_suggested when a hint was emitted (CFS>=7
// or AKPS<=40), otherwise scoring_recorded.
func (s *ScoringStore) writeScoreEvidenceTrace(
	ctx context.Context,
	residentRef uuid.UUID,
	roleRef uuid.UUID,
	occurredAt time.Time,
	scoreType string,
	scoreID uuid.UUID,
	summaryText string,
	hintEmitted bool,
) (uuid.UUID, error) {
	now := s.now()
	stateChangeType := "scoring_recorded"
	ruleFire := scoreType + ":recorded"
	if hintEmitted {
		stateChangeType = "care_intensity_review_suggested"
		ruleFire = scoreType + ":care_intensity_review_suggested"
	}
	rid := residentRef
	role := roleRef
	node := models.EvidenceTraceNode{
		ID:              uuid.New(),
		StateMachine:    models.EvidenceTraceStateMachineClinicalState,
		StateChangeType: stateChangeType,
		RecordedAt:      now,
		OccurredAt:      occurredAt,
		Actor:           models.TraceActor{RoleRef: &role},
		Inputs: []models.TraceInput{
			{
				InputType:      models.TraceInputTypeOther,
				InputRef:       scoreID,
				RoleInDecision: models.TraceRoleInDecisionPrimaryEvidence,
			},
		},
		ReasoningSummary: &models.ReasoningSummary{
			Text:      summaryText,
			RuleFires: []string{ruleFire},
		},
		Outputs: []models.TraceOutput{
			{OutputType: "Resident", OutputRef: residentRef},
			{OutputType: scoreType, OutputRef: scoreID},
		},
		ResidentRef: &rid,
		CreatedAt:   now,
	}
	persisted, err := s.v2.UpsertEvidenceTraceNode(ctx, node)
	if err != nil {
		return uuid.Nil, err
	}
	return persisted.ID, nil
}

// Compile-time assertion.
var _ interfaces.ScoringStore = (*ScoringStore)(nil)
