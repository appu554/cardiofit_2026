package services

import (
	"strings"
	"time"
)

// drugClassDomainMap maps 22 drug classes to their clinical inertia domain.
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

// MapDrugClassToDomain returns the clinical inertia domain for a drug class.
// Returns "OTHER" for unmapped classes.
func MapDrugClassToDomain(drugClass string) string {
	key := strings.ToUpper(strings.TrimSpace(drugClass))
	if domain, ok := drugClassDomainMap[key]; ok {
		return domain
	}
	// Try original case (handles mixed-case keys like DPP4i, SGLT2i).
	if domain, ok := drugClassDomainMap[drugClass]; ok {
		return domain
	}
	return "OTHER"
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
