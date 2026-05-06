package scoring

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// ComputeACB calculates the Anticholinergic Cognitive Burden for a Resident
// given their current MedicineUse list and a DrugWeightLookup over the
// acb_drug_weights seed table (Boustani 2008 / Salahudeen). Pure function
// — no DB calls, no logging, no global state.
//
// Behaviour per Layer 2 doc §2.6:
//   - Only MedicineUseStatusActive rows contribute (paused/ceased/completed
//     excluded — they do not contribute to current cognitive burden).
//   - Each contributing drug adds its integer ACBWeight (1/2/3) to the
//     running total.
//   - Unknown drugs are recorded in UnknownDrugs but DO NOT fail the
//     compute, mirroring DBI semantics.
//   - A lookup error aborts the compute.
//
// The returned ACBScore has a fresh ID + ComputedAt from the caller.
func ComputeACB(
	ctx context.Context,
	meds []models.MedicineUse,
	lookup DrugWeightLookup,
	residentRef uuid.UUID,
	computedAt time.Time,
) (models.ACBScore, error) {
	var (
		sum     int
		inputs  []uuid.UUID
		unknown []string
	)
	for _, m := range meds {
		if m.Status != models.MedicineUseStatusActive {
			continue
		}
		w, found, err := lookup.Lookup(ctx, m.DisplayName)
		if err != nil {
			return models.ACBScore{}, err
		}
		if !found {
			unknown = append(unknown, m.DisplayName)
			continue
		}
		sum += w.ACBWeight
		inputs = append(inputs, m.ID)
	}
	return models.ACBScore{
		ID:                uuid.New(),
		ResidentRef:       residentRef,
		ComputedAt:        computedAt,
		Score:             sum,
		ComputationInputs: inputs,
		UnknownDrugs:      unknown,
	}, nil
}
