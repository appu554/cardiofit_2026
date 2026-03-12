package services

import (
	"fmt"
	"math"

	"go.uber.org/zap"
)

// OnsetDerivationService implements D06: PK-derived onset windows.
// When SPL data lacks explicit onset timing, pharmacokinetic parameters
// (Tmax, T½) are used to auto-compute expected ADR onset windows.
//
// Clinical rationale:
//   - Type A (dose-dependent) ADRs typically manifest within 1-3 T½ after reaching Cmax
//   - Type B (idiosyncratic) ADRs have unpredictable onset (not PK-derivable)
//   - This service only generates onset windows for Type A reactions
//
// Reference: Rawlins & Thompson classification (1977), FDA guidance on
// pharmacokinetic-based safety monitoring windows.
type OnsetDerivationService struct {
	log      *zap.Logger
	profiles map[string]PKProfile
}

// PKProfile holds pharmacokinetic parameters for a drug class.
// These are population-median values from FDA-approved labeling.
type PKProfile struct {
	DrugClass      string  `json:"drug_class"`
	RepresentDrug  string  `json:"representative_drug"`
	TmaxHours      float64 `json:"tmax_hours"`       // Time to peak concentration (median)
	HalfLifeHours  float64 `json:"half_life_hours"`   // Elimination half-life (median)
	BioavailPct    float64 `json:"bioavail_pct"`      // Oral bioavailability %
	SteadyStateDays float64 `json:"steady_state_days"` // Days to reach steady-state (approx 5 * T½)
}

// OnsetWindow is the computed ADR onset window for a drug class.
type OnsetWindow struct {
	DrugClass     string  `json:"drug_class"`
	EarlyOnsetH   float64 `json:"early_onset_hours"`   // Earliest expected onset
	PeakOnsetH    float64 `json:"peak_onset_hours"`    // Most likely onset time
	LateOnsetH    float64 `json:"late_onset_hours"`    // Latest expected onset
	OnsetCategory string  `json:"onset_category"`      // ACUTE, SUBACUTE, CHRONIC
	Derivation    string  `json:"derivation"`          // Explanation of computation
}

// NewOnsetDerivationService creates the PK onset derivation service with
// built-in profiles for 8 common CKD/cardiovascular drug classes.
func NewOnsetDerivationService(log *zap.Logger) *OnsetDerivationService {
	svc := &OnsetDerivationService{
		log:      log,
		profiles: make(map[string]PKProfile),
	}
	svc.loadBuiltinProfiles()
	return svc
}

// loadBuiltinProfiles populates PK profiles for 8 drug classes commonly
// encountered in CKD/HTN/DM patients. Values from FDA labeling (median).
func (s *OnsetDerivationService) loadBuiltinProfiles() {
	builtins := []PKProfile{
		{
			DrugClass:      "ACE_INHIBITOR",
			RepresentDrug:  "Lisinopril",
			TmaxHours:      7.0,
			HalfLifeHours:  12.0,
			BioavailPct:    25.0,
			SteadyStateDays: 2.5,
		},
		{
			DrugClass:      "ARB",
			RepresentDrug:  "Losartan",
			TmaxHours:      1.0,
			HalfLifeHours:  6.0,
			BioavailPct:    33.0,
			SteadyStateDays: 1.3,
		},
		{
			DrugClass:      "SGLT2_INHIBITOR",
			RepresentDrug:  "Empagliflozin",
			TmaxHours:      1.5,
			HalfLifeHours:  12.4,
			BioavailPct:    78.0,
			SteadyStateDays: 2.6,
		},
		{
			DrugClass:      "STATIN",
			RepresentDrug:  "Atorvastatin",
			TmaxHours:      1.0,
			HalfLifeHours:  14.0,
			BioavailPct:    14.0,
			SteadyStateDays: 2.9,
		},
		{
			DrugClass:      "BETA_BLOCKER",
			RepresentDrug:  "Metoprolol",
			TmaxHours:      1.5,
			HalfLifeHours:  3.5,
			BioavailPct:    50.0,
			SteadyStateDays: 0.7,
		},
		{
			DrugClass:      "CCB",
			RepresentDrug:  "Amlodipine",
			TmaxHours:      8.0,
			HalfLifeHours:  40.0,
			BioavailPct:    64.0,
			SteadyStateDays: 8.3,
		},
		{
			DrugClass:      "DIURETIC_LOOP",
			RepresentDrug:  "Furosemide",
			TmaxHours:      1.0,
			HalfLifeHours:  2.0,
			BioavailPct:    50.0,
			SteadyStateDays: 0.4,
		},
		{
			DrugClass:      "DIURETIC_THIAZIDE",
			RepresentDrug:  "Hydrochlorothiazide",
			TmaxHours:      2.0,
			HalfLifeHours:  10.0,
			BioavailPct:    70.0,
			SteadyStateDays: 2.1,
		},
	}

	for _, p := range builtins {
		s.profiles[p.DrugClass] = p
	}

	s.log.Info("D06: loaded PK profiles",
		zap.Int("count", len(s.profiles)),
	)
}

// DeriveOnset computes the expected ADR onset window for a drug class
// using PK parameters. Returns nil if no profile is available.
//
// Algorithm:
//   - Early onset: Tmax (drug reaches Cmax → first exposure to peak levels)
//   - Peak onset: Tmax + 1*T½ (one elimination half-life after peak)
//   - Late onset: Tmax + 3*T½ (three half-lives → covers >87.5% of cases)
//   - Category: <24h=ACUTE, 24h-14d=SUBACUTE, >14d=CHRONIC
func (s *OnsetDerivationService) DeriveOnset(drugClass string) *OnsetWindow {
	pk, ok := s.profiles[drugClass]
	if !ok {
		s.log.Debug("D06: no PK profile for drug class",
			zap.String("drug_class", drugClass),
		)
		return nil
	}

	early := pk.TmaxHours
	peak := pk.TmaxHours + pk.HalfLifeHours
	late := pk.TmaxHours + 3.0*pk.HalfLifeHours

	category := categorizeOnset(late)

	window := &OnsetWindow{
		DrugClass:     drugClass,
		EarlyOnsetH:   math.Round(early*10) / 10,
		PeakOnsetH:    math.Round(peak*10) / 10,
		LateOnsetH:    math.Round(late*10) / 10,
		OnsetCategory: category,
		Derivation: fmt.Sprintf(
			"PK-derived from %s: Tmax=%.1fh, T½=%.1fh → onset window %.1f-%.1fh (%s)",
			pk.RepresentDrug, pk.TmaxHours, pk.HalfLifeHours, early, late, category,
		),
	}

	s.log.Info("D06: derived onset window",
		zap.String("drug_class", drugClass),
		zap.String("category", category),
		zap.Float64("early_h", window.EarlyOnsetH),
		zap.Float64("late_h", window.LateOnsetH),
	)

	return window
}

// FormatOnsetString formats an onset window as a human-readable string
// suitable for the ADR profile's onset_window field.
func (s *OnsetDerivationService) FormatOnsetString(w *OnsetWindow) string {
	if w.LateOnsetH < 24 {
		return fmt.Sprintf("%.0f-%.0f hours (PK-derived)", w.EarlyOnsetH, w.LateOnsetH)
	}
	earlyDays := w.EarlyOnsetH / 24.0
	lateDays := w.LateOnsetH / 24.0
	if earlyDays < 1 {
		return fmt.Sprintf("%.0f hours - %.0f days (PK-derived)", w.EarlyOnsetH, lateDays)
	}
	return fmt.Sprintf("%.0f-%.0f days (PK-derived)", earlyDays, lateDays)
}

// GetProfile returns the PK profile for a drug class, or nil if not found.
func (s *OnsetDerivationService) GetProfile(drugClass string) *PKProfile {
	pk, ok := s.profiles[drugClass]
	if !ok {
		return nil
	}
	return &pk
}

// ListProfiles returns all available PK profiles.
func (s *OnsetDerivationService) ListProfiles() []PKProfile {
	result := make([]PKProfile, 0, len(s.profiles))
	for _, p := range s.profiles {
		result = append(result, p)
	}
	return result
}

// categorizeOnset maps the late onset time to an onset category.
func categorizeOnset(lateOnsetHours float64) string {
	switch {
	case lateOnsetHours < 24:
		return "ACUTE"
	case lateOnsetHours < 336: // 14 days
		return "SUBACUTE"
	default:
		return "CHRONIC"
	}
}
