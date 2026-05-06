package ingestion

import (
	"context"
	"errors"
	"strings"
)

// AMTLookup resolves a free-text medication name + strength + form into an
// Australian Medicines Terminology (AMT) code. Implementations may consult
// kb-7-terminology, an external SNOMED-CT-AU service, or an in-memory
// fixture (for tests). A confidence score in [0.0, 1.0] is returned: 1.0 =
// exact match; <1.0 = fuzzy / partial match; 0.0 = not found.
//
// Implementations MUST NOT panic on unknown inputs; "not found" is a
// normal, expected outcome (eNRMC exports routinely contain
// brand/specialised names that have no AMT equivalent yet).
type AMTLookup interface {
	LookupByName(ctx context.Context, medName, strength, form string) (code string, confidence float64, err error)
}

// SNOMEDLookup resolves a free-text indication into a SNOMED-CT-AU code.
// Same confidence semantics as AMTLookup. Empty indication text yields
// ("", 0.0, nil) — not an error.
type SNOMEDLookup interface {
	LookupIndication(ctx context.Context, indicationText string) (code string, confidence float64, err error)
}

// Normaliser turns a CSVRow into a NormalisedMedicineUse by consulting the
// AMTLookup and SNOMEDLookup interfaces. The Normaliser itself is pure
// (no IO); all IO happens behind the two interfaces.
type Normaliser struct {
	AMT    AMTLookup
	SNOMED SNOMEDLookup
}

// NormalisedMedicineUse is the runner-friendly intermediate produced by
// Normalise. The runner combines this with the matched ResidentRef to
// build the final models.MedicineUse.
//
// AMTConfidence and IndicationConfidence are surfaced so the runner can
// flag low-confidence rows in the EvidenceTrace audit node and (in the
// future) gate auto-acceptance behind a configurable threshold.
type NormalisedMedicineUse struct {
	AMTCode              string
	AMTConfidence        float64
	PrimaryIndication    string
	IndicationConfidence float64
	Original             CSVRow
}

// ErrNoAMTLookup signals a misconfigured Normaliser. The runner converts
// this into a per-row error so a single misconfiguration does not crash
// the entire run.
var ErrNoAMTLookup = errors.New("ingestion: no AMTLookup configured")

// Normalise runs both lookups against row. Returns a NormalisedMedicineUse
// with whatever could be resolved; AMT-not-found and SNOMED-not-found are
// reported as zero-confidence rather than errors. A non-nil error is
// returned only for transport-level failures from the lookup
// implementations or for misconfiguration.
func (n *Normaliser) Normalise(ctx context.Context, row CSVRow) (NormalisedMedicineUse, error) {
	if n == nil || n.AMT == nil {
		return NormalisedMedicineUse{Original: row}, ErrNoAMTLookup
	}
	out := NormalisedMedicineUse{Original: row}

	medName := strings.TrimSpace(row.MedicationName)
	strength := strings.TrimSpace(row.Strength)
	form := strings.TrimSpace(row.Form)
	if medName != "" {
		code, conf, err := n.AMT.LookupByName(ctx, medName, strength, form)
		if err != nil {
			return out, err
		}
		out.AMTCode = code
		out.AMTConfidence = conf
	}

	if n.SNOMED != nil {
		ind := strings.TrimSpace(row.IndicationText)
		if ind != "" {
			code, conf, err := n.SNOMED.LookupIndication(ctx, ind)
			if err != nil {
				return out, err
			}
			out.PrimaryIndication = code
			out.IndicationConfidence = conf
		}
	}
	return out, nil
}

// ----- Stub lookups (production-binary default until kb-7 wired) -----

// StubAMTLookup is a placeholder that always returns "not found". The
// production binary uses this until kb-7-terminology is wired in. Tests
// use richer in-memory fakes defined alongside the test files.
type StubAMTLookup struct{}

// LookupByName always returns ("", 0.0, nil).
func (StubAMTLookup) LookupByName(_ context.Context, _, _, _ string) (string, float64, error) {
	return "", 0.0, nil
}

// StubSNOMEDLookup is a placeholder that always returns "not found".
type StubSNOMEDLookup struct{}

// LookupIndication always returns ("", 0.0, nil).
func (StubSNOMEDLookup) LookupIndication(_ context.Context, _ string) (string, float64, error) {
	return "", 0.0, nil
}
