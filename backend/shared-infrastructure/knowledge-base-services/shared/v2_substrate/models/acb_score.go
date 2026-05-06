// Package models — ACBScore is the Anticholinergic Cognitive Burden
// computed-score entity introduced by Wave 2.6 of the Layer 2 substrate
// plan (Layer 2 doc §2.6).
//
// ACB quantifies the cumulative anticholinergic cognitive burden carried
// by a Resident's active medication list, per the Boustani 2008 / Salahudeen
// scoring system:
//
//	ACB = Σ acb_weight_i  where acb_weight_i ∈ {1, 2, 3}
//
// Each active MedicineUse contributes its integer ACB weight (1=possible,
// 2=definite mild, 3=definite strong) looked up from the acb_drug_weights
// seed table. Higher ACB is associated with increased risk of cognitive
// decline and dementia in older adults.
//
// ACB is computed (not clinician-entered): any MedicineUse insert/update/end
// for the Resident triggers a recompute that writes a new acb_scores row.
// The history is append-only — never UPDATE rows.
//
// Canonical storage: kb-20-patient-profile (acb_scores table, migration 018).
// The latest row by ComputedAt per ResidentRef is the current burden
// (queried via the acb_current view).
//
// FHIR boundary: not mapped in MVP — ACB is a Vaidshala-internal informational
// score per the plan.
package models

import (
	"time"

	"github.com/google/uuid"
)

// ACBScore captures one Anticholinergic Cognitive Burden recomputation
// for a Resident.
//
// ComputationInputs records the MedicineUse refs that contributed to the
// score so the EvidenceTrace chain can walk back from the score to the
// underlying medications. UnknownDrugs records DisplayNames that were
// skipped because no row in acb_drug_weights matched — surfaces the
// seed-table coverage gap to downstream consumers without failing the
// compute.
type ACBScore struct {
	ID                uuid.UUID   `json:"id"`
	ResidentRef       uuid.UUID   `json:"resident_ref"`
	ComputedAt        time.Time   `json:"computed_at"`
	Score             int         `json:"score"`                   // weighted sum (0..N)
	ComputationInputs []uuid.UUID `json:"computation_inputs"`      // MedicineUse refs that contributed
	UnknownDrugs      []string    `json:"unknown_drugs,omitempty"` // DisplayNames not in seed table
	CreatedAt         time.Time   `json:"created_at"`
}
