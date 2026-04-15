package services

import (
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"kb-patient-profile/internal/models"
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

// InterventionTimelineService queries medication_states and aggregates the
// latest action per clinical inertia domain. Phase 7 P7-D: the KB-23
// InertiaInputAssembler fetches this via GET /patient/:id/intervention-timeline
// to determine the LastIntervention timestamp for each DomainInertiaInput
// fed into DetectInertia.
type InterventionTimelineService struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewInterventionTimelineService wires the dependencies.
func NewInterventionTimelineService(db *gorm.DB, logger *zap.Logger) *InterventionTimelineService {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &InterventionTimelineService{db: db, logger: logger}
}

// BuildTimeline assembles an InterventionTimelineResult for a patient by
// scanning medication_state rows updated in the last 90 days, mapping each
// row's drug class to its PRIMARY clinical domain, and keeping the latest
// action per domain. Returns a non-nil result even when the patient has
// no recent interventions — ByDomain will be empty and
// AnyChangeInLast12Weeks will be false.
func (s *InterventionTimelineService) BuildTimeline(patientID string) (*InterventionTimelineResult, error) {
	if s.db == nil {
		return &InterventionTimelineResult{PatientID: patientID, ByDomain: map[string]LatestDomainAction{}}, nil
	}

	cutoff := time.Now().AddDate(0, 0, -90)

	// Query every medication_state row started, updated, or still active
	// for this patient within the last 90 days. Using UpdatedAt rather
	// than StartDate so a dose change on an existing prescription
	// resets the inertia clock — that's the clinical intent, not the
	// row's original creation date.
	var rows []models.MedicationState
	err := s.db.
		Where("patient_id = ? AND (updated_at >= ? OR start_date >= ?)", patientID, cutoff, cutoff).
		Order("updated_at DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := &InterventionTimelineResult{
		PatientID: patientID,
		ByDomain:  map[string]LatestDomainAction{},
	}

	now := time.Now().UTC()
	for _, row := range rows {
		domain := MapDrugClassToDomain(row.DrugClass)
		if domain == "OTHER" {
			continue
		}

		// The action date is the more recent of StartDate or UpdatedAt —
		// a mid-course dose change has the UpdatedAt stamp, an initial
		// prescription has StartDate, and the later one reflects the
		// most recent clinical decision touching the regimen.
		actionDate := row.StartDate
		if row.UpdatedAt.After(actionDate) {
			actionDate = row.UpdatedAt
		}

		existing, found := result.ByDomain[domain]
		if found && !actionDate.After(existing.ActionDate) {
			continue
		}

		doseMg, _ := row.DoseMg.Float64()
		result.ByDomain[domain] = LatestDomainAction{
			InterventionID:   row.ID.String(),
			InterventionType: "MEDICATION_CHANGE",
			DrugClass:        row.DrugClass,
			DrugName:         row.DrugName,
			DoseMg:           doseMg,
			ActionDate:       actionDate,
			DaysSince:        int(now.Sub(actionDate).Hours() / 24),
		}

		if row.IsActive {
			result.TotalActiveInterventions++
		}
	}

	// AnyChangeInLast12Weeks is true if at least one domain's latest action
	// is ≤84 days old (the threshold the inertia detector uses as the
	// grace window for HbA1c-driven glycaemic inertia).
	for _, action := range result.ByDomain {
		if action.DaysSince <= 84 {
			result.AnyChangeInLast12Weeks = true
			break
		}
	}
	return result, nil
}
