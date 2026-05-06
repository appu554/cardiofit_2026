// Package reconciliation provides the pure-Go substrate engine for
// hospital-discharge medication reconciliation (Layer 2 doc §3.2,
// Wave 4 of the substrate plan).
//
// Three pieces compose the engine:
//
//   - diff.go       — ComputeDiff matches pre-admission MedicineUses against
//                     discharge medication lines and classifies each into
//                     new_medication | ceased_medication | dose_change |
//                     unchanged.
//   - classifier.go — ClassifyIntent inspects the discharge text near a
//                     diff entry and produces an intent class
//                     (acute_illness_temporary | new_chronic |
//                     reconciled_change | unclear).
//   - worklist.go   — BuildWorklistInputs turns the diff + classifications
//                     into the row inputs needed for a reconciliation
//                     worklist + decision rows.
//   - writeback.go  — ApplyDecision converts an ACOP-resolved decision into
//                     concrete MedicineUse mutations (insert / end / update).
//
// The whole package is IO-free. Storage layers wrap it; tests run without
// a database.
package reconciliation

import (
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// DiffClass classifies the relationship between a pre-admission
// MedicineUse list and a discharge medication line list (Layer 2 §3.2
// step 4).
type DiffClass string

const (
	DiffNewMedication    DiffClass = "new_medication"
	DiffCeasedMedication DiffClass = "ceased_medication"
	DiffDoseChange       DiffClass = "dose_change"
	DiffUnchanged        DiffClass = "unchanged"
)

// IsValidDiffClass reports whether s is a recognised DiffClass value.
func IsValidDiffClass(s string) bool {
	switch DiffClass(s) {
	case DiffNewMedication, DiffCeasedMedication, DiffDoseChange, DiffUnchanged:
		return true
	}
	return false
}

// DischargeLineSummary is the diff-engine input shape for a single line
// from a parsed discharge document. It carries enough context to (a)
// match against a pre-admission MedicineUse and (b) feed the classifier.
type DischargeLineSummary struct {
	LineRef     uuid.UUID
	AMTCode     string
	DisplayName string
	Dose        string
	Frequency   string
	Route       string
	// IndicationText / Notes are passed to the classifier verbatim; they
	// hold the per-line free text that the classifier scans for acute /
	// chronic / reconciled markers.
	IndicationText string
	Notes          string
}

// DiffEntry is one row of the reconciliation diff. Exactly one of
// PreAdmissionMedUseRef / DischargeLineRef is nil for new_medication
// (pre is nil) and ceased_medication (discharge is nil); both are set
// for dose_change and unchanged.
type DiffEntry struct {
	Class                 DiffClass
	PreAdmissionMedUseRef *uuid.UUID            // nil for new_medication
	DischargeLineRef      *uuid.UUID            // nil for ceased_medication
	PreAdmissionMedicine  *models.MedicineUse   // optional payload for downstream
	DischargeLineMedicine *DischargeLineSummary // optional payload for downstream
	// DoseChangeSummary is a human-readable description of the dose-line
	// delta when Class == DiffDoseChange. Empty for the other classes.
	DoseChangeSummary string
}

// ComputeDiff classifies pre-admission MedicineUses against discharge
// medication lines.
//
// Matching strategy (per Layer 2 §3.2 step 4):
//
//  1. Filter pre-admission to active rows only (MedicineUseStatusActive).
//  2. Primary match: AMTCode equality (both sides non-empty).
//  3. Fallback match: case-insensitive DisplayName prefix match (the
//     shorter of the two normalised names is the prefix). Brand vs
//     generic names commonly share a leading token; this catches them
//     without requiring an AMT-mapping service in the pure engine.
//  4. Matched pair: dose / frequency / route comparison decides
//     dose_change vs unchanged.
//  5. Unmatched pre-admission rows → ceased_medication.
//  6. Unmatched discharge lines → new_medication.
//
// The function is deterministic: the order of returned DiffEntries is
// (matched-pairs in pre-admission order, then ceased, then new) — tests
// rely on this. Inputs are not mutated.
func ComputeDiff(pre []models.MedicineUse, discharge []DischargeLineSummary) []DiffEntry {
	// Index discharge lines by AMTCode and normalised name for matching.
	byAMT := map[string]*dischargeRef{}
	byName := map[string]*dischargeRef{}
	dischargeRefs := make([]dischargeRef, len(discharge))
	for i := range discharge {
		dischargeRefs[i] = dischargeRef{idx: i}
	}
	for i := range discharge {
		ref := &dischargeRefs[i]
		if c := strings.TrimSpace(discharge[i].AMTCode); c != "" {
			byAMT[c] = ref
		}
		if n := normalisedName(discharge[i].DisplayName); n != "" {
			// First-write wins so tests have stable ordering when names collide.
			if _, exists := byName[n]; !exists {
				byName[n] = ref
			}
		}
	}

	out := make([]DiffEntry, 0, len(pre)+len(discharge))

	// Pass 1 — matched + ceased, in pre-admission order.
	for i := range pre {
		p := pre[i]
		if p.Status != models.MedicineUseStatusActive {
			// Inactive pre-admission rows are out of scope per step 4.
			continue
		}
		match := matchDischarge(p, byAMT, byName, discharge)
		if match == nil {
			pid := p.ID
			pcopy := pre[i]
			out = append(out, DiffEntry{
				Class:                 DiffCeasedMedication,
				PreAdmissionMedUseRef: &pid,
				PreAdmissionMedicine:  &pcopy,
			})
			continue
		}
		match.used = true
		d := discharge[match.idx]
		entry := DiffEntry{
			PreAdmissionMedUseRef: ptrUUID(p.ID),
			DischargeLineRef:      ptrUUID(d.LineRef),
			PreAdmissionMedicine:  &pre[i],
			DischargeLineMedicine: &discharge[match.idx],
		}
		summary, changed := compareDose(p, d)
		if changed {
			entry.Class = DiffDoseChange
			entry.DoseChangeSummary = summary
		} else {
			entry.Class = DiffUnchanged
		}
		out = append(out, entry)
	}

	// Pass 2 — unmatched discharge lines become new_medication.
	for i := range discharge {
		if dischargeRefs[i].used {
			continue
		}
		out = append(out, DiffEntry{
			Class:                 DiffNewMedication,
			DischargeLineRef:      ptrUUID(discharge[i].LineRef),
			DischargeLineMedicine: &discharge[i],
		})
	}

	return out
}

// matchDischarge picks the discharge line that corresponds to a
// pre-admission MedicineUse, or nil if none match. Strategy: AMT-first,
// name-prefix fallback. Mutation of the returned ref's `used` flag is
// the caller's responsibility (kept side-effect-free here so tests can
// assert on intermediate state cleanly).
func matchDischarge(p models.MedicineUse, byAMT, byName map[string]*dischargeRef, discharge []DischargeLineSummary) *dischargeRef {
	if c := strings.TrimSpace(p.AMTCode); c != "" {
		if ref, ok := byAMT[c]; ok && !ref.used {
			return ref
		}
	}
	pn := normalisedName(p.DisplayName)
	if pn == "" {
		return nil
	}
	// Direct name hit first.
	if ref, ok := byName[pn]; ok && !ref.used {
		return ref
	}
	// Prefix fallback — find a discharge line whose normalised name shares
	// a leading token with the pre-admission name. Catches "metformin" vs
	// "metformin XR" and "ramipril 5mg" vs "ramipril".
	pHead := firstToken(pn)
	if pHead == "" {
		return nil
	}
	for i := range discharge {
		dn := normalisedName(discharge[i].DisplayName)
		if dn == "" {
			continue
		}
		ref := byName[dn]
		if ref == nil || ref.used {
			continue
		}
		if firstToken(dn) == pHead {
			return ref
		}
	}
	return nil
}

// compareDose returns a human-readable dose/frequency/route summary and
// reports whether any of the three differ between the pre-admission row
// and the discharge line. The comparison is whitespace-insensitive and
// case-insensitive.
func compareDose(p models.MedicineUse, d DischargeLineSummary) (string, bool) {
	preDose := strings.ToLower(strings.TrimSpace(p.Dose))
	preFreq := strings.ToLower(strings.TrimSpace(p.Frequency))
	preRoute := strings.ToLower(strings.TrimSpace(p.Route))
	postDose := strings.ToLower(strings.TrimSpace(d.Dose))
	postFreq := strings.ToLower(strings.TrimSpace(d.Frequency))
	postRoute := strings.ToLower(strings.TrimSpace(d.Route))

	parts := []string{}
	changed := false
	if preDose != postDose {
		parts = append(parts, fmt.Sprintf("dose %q→%q", p.Dose, d.Dose))
		changed = true
	}
	if preFreq != postFreq {
		parts = append(parts, fmt.Sprintf("frequency %q→%q", p.Frequency, d.Frequency))
		changed = true
	}
	if preRoute != postRoute {
		parts = append(parts, fmt.Sprintf("route %q→%q", p.Route, d.Route))
		changed = true
	}
	if !changed {
		return "", false
	}
	return strings.Join(parts, "; "), true
}

// normalisedName lowercases and collapses whitespace; empty input returns
// empty string. Used as a stable map key for name-based matching.
func normalisedName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	return strings.Join(strings.Fields(s), " ")
}

// firstToken returns the first whitespace-separated token of a normalised
// name; empty if the input is empty.
func firstToken(s string) string {
	if s == "" {
		return ""
	}
	idx := strings.IndexByte(s, ' ')
	if idx < 0 {
		return s
	}
	return s[:idx]
}

func ptrUUID(u uuid.UUID) *uuid.UUID { return &u }

// dischargeRef is the index-shared mutable cell used by matchDischarge;
// declared as a package-level type to keep the maps simple.
type dischargeRef struct {
	idx  int
	used bool
}
