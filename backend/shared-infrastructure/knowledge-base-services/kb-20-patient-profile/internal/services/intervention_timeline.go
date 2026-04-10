package services

import (
	"strings"
	"time"
)

// drugClassDomainMap maps drug classes to their PRIMARY clinical inertia domain.
var drugClassDomainMap = map[string]string{
	"METFORMIN":      "GLYCAEMIC",
	"SULFONYLUREA":   "GLYCAEMIC",
	"DPP4i":          "GLYCAEMIC",
	"SGLT2i":         "GLYCAEMIC",
	"GLP1_RA":        "GLYCAEMIC",
	"INSULIN":        "GLYCAEMIC",
	"BASAL_INSULIN":  "GLYCAEMIC",
	"PIOGLITAZONE":   "GLYCAEMIC",
	"EXENATIDE":      "GLYCAEMIC",
	"ACEi":           "HEMODYNAMIC",
	"ARB":            "HEMODYNAMIC",
	"CCB":            "HEMODYNAMIC",
	"AMLODIPINE":     "HEMODYNAMIC",
	"THIAZIDE":       "HEMODYNAMIC",
	"LOOP_DIURETIC":  "HEMODYNAMIC",
	"BETA_BLOCKER":   "HEMODYNAMIC",
	"MRA":            "HEMODYNAMIC",
	"SPIRONOLACTONE": "HEMODYNAMIC",
	"STATIN":         "LIPID",
	"EZETIMIBE":      "LIPID",
	"FINERENONE":     "RENAL",
}

// drugClassSecondaryDomains maps dual-benefit drugs to their secondary domains.
// SGLT2i: primary GLYCAEMIC, also renoprotective (DAPA-CKD/CREDENCE) and
// modest BP lowering (5-6 mmHg SBP reduction per EMPA-REG).
// GLP1_RA: primary GLYCAEMIC, also modest hemodynamic benefit (LEADER/SUSTAIN-6).
// ACEi/ARB: primary HEMODYNAMIC, also renoprotective (KDIGO 2024 first-line).
var drugClassSecondaryDomains = map[string][]string{
	"SGLT2i":    {"RENAL", "HEMODYNAMIC"},
	"GLP1_RA":   {"HEMODYNAMIC"},
	"ACEi":      {"RENAL"},
	"ARB":       {"RENAL"},
	"FINERENONE": {"HEMODYNAMIC"}, // FIDELIO-DKD: CV benefit alongside renal
}

// MapDrugClassToDomain returns the PRIMARY clinical inertia domain for a drug class.
// Returns "OTHER" for unmapped classes.
func MapDrugClassToDomain(drugClass string) string {
	key := strings.ToUpper(strings.TrimSpace(drugClass))
	if domain, ok := drugClassDomainMap[key]; ok {
		return domain
	}
	if domain, ok := drugClassDomainMap[drugClass]; ok {
		return domain
	}
	return "OTHER"
}

// MapDrugClassToAllDomains returns ALL clinical domains a drug class affects.
// This is used by the inertia detector to reset inertia clocks across all
// relevant domains when a dual-benefit drug is started or changed.
// Example: SGLT2i start resets GLYCAEMIC, RENAL, and HEMODYNAMIC inertia clocks.
func MapDrugClassToAllDomains(drugClass string) []string {
	primary := MapDrugClassToDomain(drugClass)
	if primary == "OTHER" {
		return nil
	}
	domains := []string{primary}

	key := strings.ToUpper(strings.TrimSpace(drugClass))
	if secondary, ok := drugClassSecondaryDomains[key]; ok {
		domains = append(domains, secondary...)
	} else if secondary, ok := drugClassSecondaryDomains[drugClass]; ok {
		domains = append(domains, secondary...)
	}
	return domains
}

// InterventionTimelineResult summarises the most recent intervention per
// clinical domain for a patient. Used by the therapeutic-inertia engine
// to cross-reference action recency with target status.
type InterventionTimelineResult struct {
	PatientID                string
	ByDomain                 map[string]LatestDomainAction
	AnyChangeInLast12Weeks   bool
	TotalActiveInterventions int
}

// LatestDomainAction captures the most recent intervention within one domain.
type LatestDomainAction struct {
	InterventionID   string
	InterventionType string
	DrugClass        string
	DrugName         string
	DoseMg           float64
	ActionDate       time.Time
	DaysSince        int
}
