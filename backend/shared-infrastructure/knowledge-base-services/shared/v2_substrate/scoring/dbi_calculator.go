package scoring

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// ComputeDBI calculates the Drug Burden Index for a Resident given their
// current MedicineUse list and a DrugWeightLookup over the dbi_drug_weights
// seed table (Hilmer et al. 2007). Pure function — no DB calls, no logging,
// no global state.
//
// Behaviour per Layer 2 doc §2.6:
//   - Only MedicineUseStatusActive rows contribute. Paused/ceased/completed
//     rows are intentionally excluded — they do not contribute to current
//     burden.
//   - Each contributing drug adds its AnticholinergicWeight + SedativeWeight
//     to the running totals. Score = AnticholinergicComponent +
//     SedativeComponent.
//   - Unknown drugs (no row in the seed table) are recorded in UnknownDrugs
//     but DO NOT fail the compute. Surfacing the gap is more useful than
//     stalling the recompute when a novel agent appears in the formulary.
//   - A lookup error aborts the compute and returns the error — the
//     caller (storage layer) decides whether to swallow it or surface it.
//
// The returned DBIScore has a fresh ID + ComputedAt set from the caller's
// computedAt argument so the storage layer controls the wall-clock for
// audit purposes.
func ComputeDBI(
	ctx context.Context,
	meds []models.MedicineUse,
	lookup DrugWeightLookup,
	residentRef uuid.UUID,
	computedAt time.Time,
) (models.DBIScore, error) {
	var (
		ach, sed float64
		inputs   []uuid.UUID
		unknown  []string
	)
	for _, m := range meds {
		if m.Status != models.MedicineUseStatusActive {
			continue
		}
		w, found, err := lookup.Lookup(ctx, m.DisplayName)
		if err != nil {
			return models.DBIScore{}, err
		}
		if !found {
			unknown = append(unknown, m.DisplayName)
			continue
		}
		ach += w.AnticholinergicWeight
		sed += w.SedativeWeight
		inputs = append(inputs, m.ID)
	}
	return models.DBIScore{
		ID:                       uuid.New(),
		ResidentRef:              residentRef,
		ComputedAt:               computedAt,
		Score:                    ach + sed,
		AnticholinergicComponent: ach,
		SedativeComponent:        sed,
		ComputationInputs:        inputs,
		UnknownDrugs:             unknown,
	}, nil
}
