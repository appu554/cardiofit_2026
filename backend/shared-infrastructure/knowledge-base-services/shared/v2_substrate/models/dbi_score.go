// Package models — DBIScore is the Drug Burden Index computed-score entity
// introduced by Wave 2.6 of the Layer 2 substrate plan (Layer 2 doc §2.6).
//
// DBI quantifies the cumulative anticholinergic + sedative burden carried
// by a Resident's active medication list, per Hilmer et al. 2007:
//
//	DBI = Σ (anticholinergic_weight_i + sedative_weight_i)
//
// where each active MedicineUse contributes its weighted load looked up
// from the dbi_drug_weights seed table. Higher DBI is associated with
// increased risk of falls and cognitive impairment in older adults.
//
// DBI is computed (not clinician-entered): any MedicineUse insert/update/end
// for the Resident triggers a recompute that writes a new dbi_scores row.
// The history is append-only — never UPDATE rows.
//
// Canonical storage: kb-20-patient-profile (dbi_scores table, migration 018).
// The latest row by ComputedAt per ResidentRef is the current burden
// (queried via the dbi_current view).
//
// FHIR boundary: not mapped in MVP — DBI is a Vaidshala-internal informational
// score per the plan.
package models

import (
	"time"

	"github.com/google/uuid"
)

// DBIScore captures one Drug Burden Index recomputation for a Resident.
//
// AnticholinergicComponent + SedativeComponent MUST sum to Score (sanity
// invariant enforced by ValidateDBIScore). ComputationInputs records the
// MedicineUse refs that contributed to the score so the EvidenceTrace
// chain can walk back from the score to the underlying medications.
// UnknownDrugs records DisplayNames that were skipped because no row in
// dbi_drug_weights matched — surfaces the seed-table coverage gap to
// downstream consumers without failing the compute.
type DBIScore struct {
	ID                       uuid.UUID   `json:"id"`
	ResidentRef              uuid.UUID   `json:"resident_ref"`
	ComputedAt               time.Time   `json:"computed_at"`
	Score                    float64     `json:"score"`                     // anticholinergic + sedative load (0..N)
	AnticholinergicComponent float64     `json:"anticholinergic_component"` // ach component of Score
	SedativeComponent        float64     `json:"sedative_component"`        // sed component of Score
	ComputationInputs        []uuid.UUID `json:"computation_inputs"`        // MedicineUse refs that contributed
	UnknownDrugs             []string    `json:"unknown_drugs,omitempty"`   // DisplayNames not in seed table
	CreatedAt                time.Time   `json:"created_at"`
}
