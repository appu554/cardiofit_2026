// Package scoring contains the pure DBI/ACB calculators and the
// CFS/AKPS clinician-entered capture validators (Wave 2.6 of the Layer 2
// substrate plan; Layer 2 doc §2.6).
//
// "Pure" means: no DB calls, no logging, no global state. Calculators
// receive a slice of MedicineUse + a DrugWeightLookup interface and return
// a fully-formed DBIScore / ACBScore. The storage layer wires a concrete
// lookup (backed by the dbi_drug_weights / acb_drug_weights seed tables
// from migration 018) and persists the returned score; the calculator
// itself is unaware of the persistence layer.
package scoring

import "context"

// DrugWeight is the per-drug weight bundle returned by DrugWeightLookup.
// One lookup row services both DBI and ACB calculators — DBI consumes
// AnticholinergicWeight + SedativeWeight (Hilmer 2007 0.0/0.5 split);
// ACB consumes ACBWeight (Boustani 2008 integer 1/2/3 scale).
//
// DrugName is the canonical name from the seed table — useful for the
// EvidenceTrace text and for surfacing in admin/coverage tooling.
type DrugWeight struct {
	DrugName              string
	AnticholinergicWeight float64
	SedativeWeight        float64
	ACBWeight             int
}

// DrugWeightLookup resolves a MedicineUse display name to its weight
// bundle. The lookup is responsible for case-folding / pattern matching;
// the calculator simply asks "do you have weights for this drug?". A
// (DrugWeight{}, false, nil) return means "no match in the seed table" —
// the calculator records the drug in UnknownDrugs and continues.
//
// An error return means "lookup failed" (e.g. DB unreachable) and aborts
// the compute — the storage-layer recompute path is best-effort and
// swallows the error so the underlying MedicineUse write still commits.
type DrugWeightLookup interface {
	Lookup(ctx context.Context, displayName string) (DrugWeight, bool, error)
}

// StaticDrugWeightLookup is an in-memory DrugWeightLookup useful for tests
// and for the calculator's own self-contained smoke fixtures. The map key
// is the case-folded display-name prefix that should match
// MedicineUse.DisplayName (case-insensitive prefix match).
//
// Production callers use the storage-backed lookup that queries
// dbi_drug_weights / acb_drug_weights via a single LIKE query.
type StaticDrugWeightLookup struct {
	// PrefixWeights maps a lowercased prefix → weight bundle. The first
	// matching prefix (longest-match wins on equal-length collisions
	// resolved by map iteration order, which is non-deterministic — so
	// callers should avoid overlapping prefixes that don't share a
	// common longer prefix).
	PrefixWeights map[string]DrugWeight
}

// NewStaticDrugWeightLookup returns a lookup keyed by lowercased prefix.
func NewStaticDrugWeightLookup(weights map[string]DrugWeight) *StaticDrugWeightLookup {
	return &StaticDrugWeightLookup{PrefixWeights: weights}
}

// Lookup implements DrugWeightLookup with a longest-prefix match against
// the lower-cased display name.
func (s *StaticDrugWeightLookup) Lookup(_ context.Context, displayName string) (DrugWeight, bool, error) {
	if s == nil || len(s.PrefixWeights) == 0 {
		return DrugWeight{}, false, nil
	}
	dn := toLowerASCII(displayName)
	var (
		best     DrugWeight
		bestLen  int
		bestHit  bool
	)
	for prefix, w := range s.PrefixWeights {
		if len(prefix) == 0 {
			continue
		}
		if len(dn) < len(prefix) {
			continue
		}
		if dn[:len(prefix)] != prefix {
			continue
		}
		if len(prefix) > bestLen {
			best = w
			bestLen = len(prefix)
			bestHit = true
		}
	}
	return best, bestHit, nil
}

// toLowerASCII lowercases A-Z without allocating an unicode-aware
// converter. The seed-table display names are ASCII (clinical drug names
// in the Hilmer/Boustani seed lists) so this is sufficient and fast.
func toLowerASCII(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}
