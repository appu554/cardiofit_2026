package ingestion

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/evidence_trace"
	"github.com/cardiofit/shared/v2_substrate/identity"
	"github.com/cardiofit/shared/v2_substrate/models"
	"github.com/cardiofit/shared/v2_substrate/validation"
)

// vaidshalaIngestionNamespace is a fixed UUID v5 namespace used for
// deterministic MedicineUse and EvidenceTrace IDs derived from CSV row
// content. Stable IDs make re-ingestion idempotent: replaying the same
// row from the same source overwrites the prior write rather than
// duplicating it.
//
// Trade-off: source-of-truth conflicts on repeated reads (an updated
// row arriving from the same source) overwrite the prior write. Cross-
// source reconciliation is the IdentityMapping resolution path's job,
// not the deterministic-ID path.
//
// This UUID is generated once and frozen; do NOT regenerate it without
// migrating prior writes.
var vaidshalaIngestionNamespace = uuid.MustParse("4f8e7c10-1d2a-5b3c-9e6f-7a8b0c1d2e3f")

// KB20Writer is the subset of client.KB20Client surface the runner needs.
// Defining it as an interface here keeps the runner testable with
// in-memory fakes and avoids a hard dependency on the HTTP client.
//
// GetMedicineUse is the idempotency probe: the runner queries by
// deterministic ID before upsert; a non-nil result means the row was
// already ingested and we treat the row as a duplicate-skip outcome.
type KB20Writer interface {
	GetMedicineUse(ctx context.Context, id uuid.UUID) (*models.MedicineUse, error)
	UpsertMedicineUse(ctx context.Context, m models.MedicineUse) (*models.MedicineUse, error)
	UpsertEvidenceTraceNode(ctx context.Context, n models.EvidenceTraceNode) (*models.EvidenceTraceNode, error)
	InsertEvidenceTraceEdge(ctx context.Context, e evidence_trace.Edge) error
}

// RunnerConfig wires the runner's collaborators. Matcher and Normaliser
// are required; Client is required unless DryRun is true.
type RunnerConfig struct {
	FacilityID  uuid.UUID
	SourceLabel string
	Client      KB20Writer
	Matcher     identity.IdentityMatcher
	Normaliser  *Normaliser
	DryRun      bool
	// Now is an optional time source; defaults to time.Now. Tests inject a
	// fixed clock to assert deterministic run-id derivation.
	Now func() time.Time
}

// RunResult is the per-run summary returned to the caller and to the CLI
// JSON stdout. Counters partition every parsed row into exactly one
// outcome bucket.
type RunResult struct {
	RowsParsed           int            `json:"rows_parsed"`
	RowsIngested         int            `json:"rows_ingested"`
	RowsSkippedDup       int            `json:"rows_skipped_dup"`
	RowsSkippedNoMatch   int            `json:"rows_skipped_no_match"`
	RowsErrored          int            `json:"rows_errored"`
	ParseErrors          []ParseError   `json:"parse_errors,omitempty"`
	PerRowErrors         []RunRowError  `json:"per_row_errors,omitempty"`
	EvidenceTraceNodeRef uuid.UUID      `json:"evidence_trace_node_ref"`
	IngestedRefs         []uuid.UUID    `json:"ingested_refs,omitempty"`
	ReviewQueueRefs      []ReviewIntent `json:"review_queue_refs,omitempty"`
}

// RunRowError is a per-line failure description (validation rejected,
// normaliser transport error, etc.).
type RunRowError struct {
	LineNumber int    `json:"line_number"`
	Reason     string `json:"reason"`
}

// ReviewIntent records a row whose identity match landed in NONE
// confidence (no resident found). The kb-20 service-layer matcher
// wrapper enqueues these onto the manual-review queue; the runner
// records the rows for surfaceability in the run report.
type ReviewIntent struct {
	LineNumber int    `json:"line_number"`
	Reason     string `json:"reason"`
}

// Run is the end-to-end ingest entry point. It parses the CSV, identity-
// matches each row, normalises drug + indication codes, builds and
// validates the MedicineUse, and (unless DryRun) writes through Client.
// Per Layer 2 doc §1.6 every run produces a run-level
// extraction_pipeline EvidenceTrace node bracketing per-row nodes via
// derived_from edges, so the audit graph is structurally complete
// regardless of how many rows succeed.
func Run(ctx context.Context, r io.Reader, cfg RunnerConfig) (RunResult, error) {
	if cfg.Matcher == nil {
		return RunResult{}, errors.New("ingestion: RunnerConfig.Matcher is required")
	}
	if cfg.Normaliser == nil {
		return RunResult{}, errors.New("ingestion: RunnerConfig.Normaliser is required")
	}
	if !cfg.DryRun && cfg.Client == nil {
		return RunResult{}, errors.New("ingestion: RunnerConfig.Client is required unless DryRun")
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}

	rows, parseErrs, err := ParseCSV(r)
	if err != nil {
		return RunResult{ParseErrors: parseErrs}, fmt.Errorf("parse: %w", err)
	}

	result := RunResult{
		RowsParsed:  len(rows),
		ParseErrors: parseErrs,
	}

	// Run-level node id is derived from source label + facility + minute-
	// truncated run time. Re-runs in close succession dedupe; runs across
	// different minutes produce distinct nodes.
	runStart := cfg.Now().UTC().Truncate(time.Minute)
	runStartID := deterministicID("run-start", cfg.SourceLabel, cfg.FacilityID.String(), runStart.Format(time.RFC3339))
	runEndID := deterministicID("run-end", cfg.SourceLabel, cfg.FacilityID.String(), runStart.Format(time.RFC3339))
	result.EvidenceTraceNodeRef = runStartID

	if !cfg.DryRun {
		startNode := models.EvidenceTraceNode{
			ID:              runStartID,
			StateMachine:    models.EvidenceTraceStateMachineClinicalState,
			StateChangeType: "extraction_pipeline_started",
			RecordedAt:      cfg.Now().UTC(),
			OccurredAt:      cfg.Now().UTC(),
			Inputs: []models.TraceInput{{
				InputType:      models.TraceInputTypeOther,
				InputRef:       deterministicID("source", cfg.SourceLabel),
				RoleInDecision: models.TraceRoleInDecisionPrimaryEvidence,
			}},
			ReasoningSummary: &models.ReasoningSummary{
				Text: fmt.Sprintf("ingest source=%s facility=%s rows_parsed=%d",
					cfg.SourceLabel, cfg.FacilityID, len(rows)),
			},
		}
		if _, err := cfg.Client.UpsertEvidenceTraceNode(ctx, startNode); err != nil {
			return result, fmt.Errorf("evidence_trace start node: %w", err)
		}
	}

	// Track which line numbers had blocking parse errors so we don't
	// double-count them downstream.
	blockedLines := map[int]bool{}
	for _, pe := range parseErrs {
		// Only "required field empty" parse errors are blocking; others
		// (currently none) would be advisory.
		if strings.Contains(pe.Reason, "required") || strings.Contains(pe.Reason, "name field") {
			blockedLines[pe.LineNumber] = true
		}
	}

	for _, row := range rows {
		if blockedLines[row.LineNumber] {
			result.RowsErrored++
			result.PerRowErrors = append(result.PerRowErrors, RunRowError{
				LineNumber: row.LineNumber,
				Reason:     "row blocked by parse error (see ParseErrors)",
			})
			continue
		}

		outcome := processRow(ctx, row, cfg, runStartID)
		switch outcome.kind {
		case rowOutcomeIngested:
			result.RowsIngested++
			result.IngestedRefs = append(result.IngestedRefs, outcome.medicineUseID)
		case rowOutcomeDuplicate:
			result.RowsSkippedDup++
		case rowOutcomeNoMatch:
			result.RowsSkippedNoMatch++
			result.ReviewQueueRefs = append(result.ReviewQueueRefs, ReviewIntent{
				LineNumber: row.LineNumber,
				Reason:     outcome.reason,
			})
		case rowOutcomeError:
			result.RowsErrored++
			result.PerRowErrors = append(result.PerRowErrors, RunRowError{
				LineNumber: row.LineNumber, Reason: outcome.reason,
			})
		}
	}

	if !cfg.DryRun {
		summary, _ := json.Marshal(map[string]int{
			"rows_parsed":           result.RowsParsed,
			"rows_ingested":         result.RowsIngested,
			"rows_skipped_dup":      result.RowsSkippedDup,
			"rows_skipped_no_match": result.RowsSkippedNoMatch,
			"rows_errored":          result.RowsErrored,
		})
		outs := make([]models.TraceOutput, 0, len(result.IngestedRefs))
		for _, ref := range result.IngestedRefs {
			outs = append(outs, models.TraceOutput{
				OutputType: "MedicineUse",
				OutputRef:  ref,
			})
		}
		endNode := models.EvidenceTraceNode{
			ID:              runEndID,
			StateMachine:    models.EvidenceTraceStateMachineClinicalState,
			StateChangeType: "extraction_pipeline_completed",
			RecordedAt:      cfg.Now().UTC(),
			OccurredAt:      cfg.Now().UTC(),
			Inputs: []models.TraceInput{{
				InputType:      models.TraceInputTypeOther,
				InputRef:       runStartID,
				RoleInDecision: models.TraceRoleInDecisionPrimaryEvidence,
			}},
			ReasoningSummary: &models.ReasoningSummary{Text: string(summary)},
			Outputs:          outs,
		}
		if _, err := cfg.Client.UpsertEvidenceTraceNode(ctx, endNode); err != nil {
			return result, fmt.Errorf("evidence_trace end node: %w", err)
		}
		// derived_from edge: end node ← start node (end node "derived from" start)
		if err := cfg.Client.InsertEvidenceTraceEdge(ctx, evidence_trace.Edge{
			From: runEndID,
			To:   runStartID,
			Kind: evidence_trace.EdgeKindDerivedFrom,
		}); err != nil {
			return result, fmt.Errorf("evidence_trace edge: %w", err)
		}
	}

	return result, nil
}

// rowOutcomeKind partitions per-row results.
type rowOutcomeKind int

const (
	rowOutcomeIngested rowOutcomeKind = iota
	rowOutcomeDuplicate
	rowOutcomeNoMatch
	rowOutcomeError
)

type rowOutcome struct {
	kind          rowOutcomeKind
	medicineUseID uuid.UUID
	reason        string
}

func processRow(ctx context.Context, row CSVRow, cfg RunnerConfig, runStartID uuid.UUID) rowOutcome {
	dob, dobErr := parseDate(row.DOB)
	if dobErr != nil {
		return rowOutcome{kind: rowOutcomeError, reason: "dob: " + dobErr.Error()}
	}
	startedAt, sErr := parseDate(row.StartDate)
	if sErr != nil {
		return rowOutcome{kind: rowOutcomeError, reason: "start_date: " + sErr.Error()}
	}
	var endedAt *time.Time
	if row.EndDate != "" {
		t, eErr := parseDate(row.EndDate)
		if eErr != nil {
			return rowOutcome{kind: rowOutcomeError, reason: "end_date: " + eErr.Error()}
		}
		endedAt = &t
	}

	facility := cfg.FacilityID
	incoming := identity.IncomingIdentifier{
		IHI:        row.IHI,
		Medicare:   row.Medicare,
		GivenName:  row.GivenName,
		FamilyName: row.FamilyName,
		DOB:        dob,
		FacilityID: &facility,
		Source:     cfg.SourceLabel,
	}
	matchRes, matchErr := cfg.Matcher.Match(ctx, incoming)
	if matchErr != nil {
		return rowOutcome{kind: rowOutcomeError, reason: "identity: " + matchErr.Error()}
	}
	if matchRes.ResidentRef == nil {
		return rowOutcome{kind: rowOutcomeNoMatch, reason: "no resident match (" + string(matchRes.Path) + ")"}
	}
	residentID := *matchRes.ResidentRef

	norm, nErr := cfg.Normaliser.Normalise(ctx, row)
	if nErr != nil {
		return rowOutcome{kind: rowOutcomeError, reason: "normaliser: " + nErr.Error()}
	}

	mu, buildErr := buildMedicineUse(row, norm, residentID, startedAt, endedAt)
	if buildErr != nil {
		return rowOutcome{kind: rowOutcomeError, reason: "build: " + buildErr.Error()}
	}

	if err := validation.ValidateMedicineUse(mu); err != nil {
		return rowOutcome{kind: rowOutcomeError, reason: "validate: " + err.Error()}
	}

	if cfg.DryRun {
		return rowOutcome{kind: rowOutcomeIngested, medicineUseID: mu.ID}
	}

	// Idempotency: deterministic ID + Get-then-Upsert. We probe by ID
	// first; if the row already exists it's a duplicate-skip outcome and
	// we still record a per-row EvidenceTrace node so the audit graph
	// captures the dedupe decision. Otherwise we Upsert as a fresh row.
	// We deliberately ignore the GetMedicineUse error — any error other
	// than not-found will resurface from the Upsert call below, with
	// richer context.
	priorRef := mu.ID
	if existing, _ := cfg.Client.GetMedicineUse(ctx, mu.ID); existing != nil {
		_ = writePerRowEvidence(ctx, cfg, row, runStartID, residentID, mu.ID, matchRes, norm, true)
		return rowOutcome{kind: rowOutcomeDuplicate, medicineUseID: priorRef}
	}
	if _, wErr := cfg.Client.UpsertMedicineUse(ctx, mu); wErr != nil {
		return rowOutcome{kind: rowOutcomeError, reason: "upsert: " + wErr.Error()}
	}

	if err := writePerRowEvidence(ctx, cfg, row, runStartID, residentID, mu.ID, matchRes, norm, false); err != nil {
		return rowOutcome{kind: rowOutcomeError, reason: "evidence_trace: " + err.Error()}
	}
	return rowOutcome{kind: rowOutcomeIngested, medicineUseID: mu.ID}
}

// writePerRowEvidence creates the per-row EvidenceTrace node and links it
// to the run-level start node via derived_from.
func writePerRowEvidence(ctx context.Context, cfg RunnerConfig, row CSVRow,
	runStartID, residentID, medicineUseID uuid.UUID,
	matchRes identity.MatchResult, norm NormalisedMedicineUse, dup bool) error {

	flags := []string{}
	if matchRes.Confidence == identity.ConfidenceLow {
		flags = append(flags, "low-confidence-identity-match")
	}
	if matchRes.RequiresReview {
		flags = append(flags, "identity-review-required")
	}
	if norm.AMTConfidence > 0 && norm.AMTConfidence < 1.0 {
		flags = append(flags, "low-confidence-normalisation")
	}
	if norm.AMTConfidence == 0 {
		flags = append(flags, "amt-not-found")
	}
	if dup {
		flags = append(flags, "duplicate-of-prior-run")
	}
	rid := residentID

	node := models.EvidenceTraceNode{
		ID:              deterministicID("row-node", cfg.SourceLabel, fmt.Sprintf("%d", row.LineNumber), runStartID.String()),
		StateMachine:    models.EvidenceTraceStateMachineClinicalState,
		StateChangeType: "ingestion_row",
		RecordedAt:      cfg.Now().UTC(),
		OccurredAt:      cfg.Now().UTC(),
		ResidentRef:     &rid,
		Inputs: []models.TraceInput{{
			InputType:      models.TraceInputTypeOther,
			InputRef:       deterministicID("csv-row", cfg.SourceLabel, fmt.Sprintf("%d", row.LineNumber)),
			RoleInDecision: models.TraceRoleInDecisionPrimaryEvidence,
		}},
		ReasoningSummary: &models.ReasoningSummary{
			Text: fmt.Sprintf("row line=%d match_path=%s match_confidence=%s flags=%s",
				row.LineNumber, matchRes.Path, matchRes.Confidence, strings.Join(flags, ",")),
		},
		Outputs: []models.TraceOutput{{
			OutputType: "MedicineUse",
			OutputRef:  medicineUseID,
		}},
	}
	if _, err := cfg.Client.UpsertEvidenceTraceNode(ctx, node); err != nil {
		return err
	}
	return cfg.Client.InsertEvidenceTraceEdge(ctx, evidence_trace.Edge{
		From: node.ID,
		To:   runStartID,
		Kind: evidence_trace.EdgeKindDerivedFrom,
	})
}

// buildMedicineUse assembles the MedicineUse model from a parsed CSV row,
// the normalised codes, and the matched ResidentRef. Intent.Category is
// always IntentUnspecified at this layer because the CSV does not carry
// intent data; downstream rule-engine workflows enrich intent later
// (Layer 2 doc §3.2 — indication NLP fallback is deferred).
func buildMedicineUse(row CSVRow, norm NormalisedMedicineUse, residentID uuid.UUID,
	startedAt time.Time, endedAt *time.Time) (models.MedicineUse, error) {

	indication := strings.TrimSpace(row.IndicationText)
	if indication == "" {
		// Validator requires Intent.Indication non-empty. Use a structured
		// sentinel so downstream queries can find rows that need NLP
		// enrichment.
		indication = "unspecified"
	}

	openSpec, _ := json.Marshal(models.TargetOpenSpec{Rationale: "ingested from " + row.MedicationName})

	status := models.MedicineUseStatusActive
	if endedAt != nil {
		status = models.MedicineUseStatusCeased
	}

	id := deterministicID(
		"medicine-use",
		residentID.String(),
		preferAMTOrName(norm),
		startedAt.UTC().Format(time.RFC3339),
	)

	mu := models.MedicineUse{
		ID:          id,
		ResidentID:  residentID,
		AMTCode:     norm.AMTCode,
		DisplayName: row.MedicationName,
		Intent: models.Intent{
			Category:   models.IntentUnspecified,
			Indication: combineIndication(indication, norm.PrimaryIndication),
			Notes:      "ingested from CSV; intent not captured in source",
		},
		Target: models.Target{
			Kind: models.TargetKindOpen,
			Spec: openSpec,
		},
		StopCriteria: models.StopCriteria{Triggers: []string{}},
		Dose:         strings.TrimSpace(row.Strength),
		Route:        strings.ToUpper(strings.TrimSpace(row.Route)),
		Frequency:    strings.TrimSpace(row.Frequency),
		StartedAt:    startedAt,
		EndedAt:      endedAt,
		Status:       status,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}
	return mu, nil
}

func combineIndication(text, snomed string) string {
	if snomed == "" {
		return text
	}
	if text == "" || text == "unspecified" {
		return snomed
	}
	return text + " [" + snomed + "]"
}

func preferAMTOrName(n NormalisedMedicineUse) string {
	if n.AMTCode != "" {
		return n.AMTCode
	}
	return strings.ToLower(strings.TrimSpace(n.Original.MedicationName))
}

// parseDate accepts ISO-8601 (YYYY-MM-DD) and a small set of common
// Australian eNRMC export formats.
func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, errors.New("empty date")
	}
	formats := []string{
		"2006-01-02",
		"02/01/2006",
		"2006-01-02T15:04:05Z07:00",
		time.RFC3339,
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognised date format %q", s)
}

// deterministicID returns a uuid v5 derived from the Vaidshala ingestion
// namespace and the SHA-256 of the joined component strings. Using
// SHA-256 of the components rather than the components themselves
// keeps the derivation stable across encoding quirks (whitespace,
// character escapes) and avoids accidental collisions with the
// uuid package's internal hashing.
func deterministicID(component ...string) uuid.UUID {
	h := sha256.Sum256([]byte(strings.Join(component, "|")))
	return uuid.NewSHA1(vaidshalaIngestionNamespace, []byte(hex.EncodeToString(h[:])))
}
