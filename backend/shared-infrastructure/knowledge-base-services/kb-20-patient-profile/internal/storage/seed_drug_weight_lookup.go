package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/cardiofit/shared/v2_substrate/scoring"
)

// SeedDrugWeightLookup is a scoring.DrugWeightLookup backed by the
// dbi_drug_weights + acb_drug_weights seed tables (migration 018). One
// query joins both tables on amt_code_pattern so a single round-trip
// returns the full DrugWeight bundle.
//
// Match strategy: case-insensitive prefix match. We compute the
// lower-cased display name in Go (cheap) and use a parameterised LIKE
// query that prefixes the seed pattern with the candidate.
type SeedDrugWeightLookup struct {
	db *sql.DB
}

// NewSeedDrugWeightLookup constructs a lookup over db. Pass the same
// *sql.DB that backs the rest of the kb-20 storage layer.
func NewSeedDrugWeightLookup(db *sql.DB) *SeedDrugWeightLookup {
	return &SeedDrugWeightLookup{db: db}
}

// Lookup implements scoring.DrugWeightLookup. Returns the LONGEST
// matching pattern from the union of dbi_drug_weights + acb_drug_weights
// (so a more-specific seed entry wins over a shorter-prefix one).
//
// (DrugWeight{}, false, nil) means no row in either seed table starts
// the supplied display name — calculator records the drug in
// UnknownDrugs and continues.
func (l *SeedDrugWeightLookup) Lookup(ctx context.Context, displayName string) (scoring.DrugWeight, bool, error) {
	if l == nil || l.db == nil {
		return scoring.DrugWeight{}, false, nil
	}
	dn := strings.ToLower(displayName)
	if dn == "" {
		return scoring.DrugWeight{}, false, nil
	}
	// LEFT JOIN both tables on amt_code_pattern, filter rows whose
	// pattern is a prefix of dn ($1 LIKE pattern || '%'), longest
	// pattern wins. COALESCE so rows present in only one table still
	// return defaulted weights for the other.
	const q = `
		SELECT
			COALESCE(d.drug_name, a.drug_name) AS drug_name,
			COALESCE(d.anticholinergic_weight, 0) AS ach,
			COALESCE(d.sedative_weight, 0) AS sed,
			COALESCE(a.weight, 0) AS acb_weight,
			COALESCE(d.amt_code_pattern, a.amt_code_pattern) AS pattern
		  FROM dbi_drug_weights d
		  FULL OUTER JOIN acb_drug_weights a USING (amt_code_pattern)
		 WHERE $1 LIKE COALESCE(d.amt_code_pattern, a.amt_code_pattern) || '%'
		 ORDER BY length(COALESCE(d.amt_code_pattern, a.amt_code_pattern)) DESC
		 LIMIT 1`

	var (
		drugName string
		ach, sed float64
		acb      int
		_pattern string
	)
	err := l.db.QueryRowContext(ctx, q, dn).Scan(&drugName, &ach, &sed, &acb, &_pattern)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return scoring.DrugWeight{}, false, nil
		}
		return scoring.DrugWeight{}, false, fmt.Errorf("seed drug weight lookup: %w", err)
	}
	return scoring.DrugWeight{
		DrugName:              drugName,
		AnticholinergicWeight: ach,
		SedativeWeight:        sed,
		ACBWeight:             acb,
	}, true, nil
}

// Compile-time assertion.
var _ scoring.DrugWeightLookup = (*SeedDrugWeightLookup)(nil)
