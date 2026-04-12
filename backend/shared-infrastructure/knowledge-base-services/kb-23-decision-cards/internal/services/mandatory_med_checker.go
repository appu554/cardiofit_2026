package services

// MandatoryMedGap represents a medication that should be present but isn't.
type MandatoryMedGap struct {
	MissingClass      string `json:"missing_class"`
	Rationale         string `json:"rationale"`
	Urgency           string `json:"urgency"`
	Alternative       string `json:"alternative,omitempty"`
	SourceTrial       string `json:"source_trial"`
	SafetyPrecautions string `json:"safety_precautions,omitempty"`
	Suppressed        bool   `json:"suppressed,omitempty"` // true if renal gate blocks this class
	SuppressionReason string `json:"suppression_reason,omitempty"`
}

// ClinicalContext holds optional patient state that refines mandatory medication checks.
// All fields are optional — CheckMandatory falls back to conservative defaults.
type ClinicalContext struct {
	// ASCVDEventTypes lists the patient's ASCVD events ("MI", "STROKE", "PAD", "PCI",
	// "CABG", "TIA", "SIGNIFICANT_CAD"). Used to gate post-MI beta-blocker recommendation.
	ASCVDEventTypes []string

	// HFUnknownSubtype is true when the patient is classified 4c but no EF is known.
	// Triggers IMMEDIATE echocardiogram gap instead of full GDMT gap list.
	HFUnknownSubtype bool

	// SBPmmHg is the most recent systolic BP. Used for ARNI initiation safety guard.
	SBPmmHg float64

	// BlockedByRenal contains drug classes already flagged by the renal gate.
	// CheckMandatory will mark those gaps as Suppressed rather than elevating urgency.
	BlockedByRenal []string
}

type MandatoryMedChecker struct{}

func NewMandatoryMedChecker() *MandatoryMedChecker {
	return &MandatoryMedChecker{}
}

// CheckMandatory returns medication gaps for the given CKM substage.
// The optional ClinicalContext refines results — when omitted, conservative
// defaults are used (beta-blocker flagged for all 4b, full GDMT for 4c).
func (c *MandatoryMedChecker) CheckMandatory(
	ckmStage string,
	hfType string,
	activeMedClasses []string,
	ctx ...ClinicalContext,
) []MandatoryMedGap {
	activeSet := make(map[string]bool)
	for _, mc := range activeMedClasses {
		activeSet[mc] = true
	}

	// Normalize: ACEi or ARB counts as RAS blockade
	hasRAS := activeSet["ACEi"] || activeSet["ARB"] || activeSet["ARNI"] || activeSet["SACUBITRIL_VALSARTAN"]
	hasARNI := activeSet["ARNI"] || activeSet["SACUBITRIL_VALSARTAN"]
	hasACEiARB := activeSet["ACEi"] || activeSet["ARB"]
	hasAntiplatelet := activeSet["ASPIRIN"] || activeSet["CLOPIDOGREL"] || activeSet["TICAGRELOR"] || activeSet["PRASUGREL"]
	hasBB := activeSet["BETA_BLOCKER"] || activeSet["BETA_BLOCKER_HF"] || activeSet["CARVEDILOL"] || activeSet["BISOPROLOL"]

	var clinCtx ClinicalContext
	if len(ctx) > 0 {
		clinCtx = ctx[0]
	}

	var gaps []MandatoryMedGap

	switch ckmStage {
	case "4a":
		if !activeSet["STATIN"] {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "STATIN",
				Rationale:    "All Stage 4a require high-intensity statin for subclinical CVD",
				Urgency:      "URGENT",
				SourceTrial:  "AHA_ACC_2018",
			})
		}

	case "4b":
		if !activeSet["STATIN"] {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "STATIN",
				Rationale:    "Secondary prevention requires high-intensity statin",
				Urgency:      "URGENT",
				SourceTrial:  "AHA_ACC_2018",
			})
		}
		if !hasAntiplatelet {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "ANTIPLATELET",
				Rationale:    "Post-ASCVD requires antiplatelet therapy",
				Urgency:      "URGENT",
				SourceTrial:  "AHA_ACC_2018",
			})
		}
		if !hasRAS {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "ACEi_OR_ARB",
				Rationale:    "Post-MI cardioprotection + renoprotection",
				Urgency:      "URGENT",
				SourceTrial:  "HOPE, EUROPA",
			})
		}
		// Beta-blocker only mandatory post-MI (CAPRICORN). Not recommended
		// post-stroke or isolated PAD — no outcome data in those contexts.
		if !hasBB && hasEventType(clinCtx.ASCVDEventTypes, "MI") {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "BETA_BLOCKER",
				Rationale:    "Post-MI mortality reduction — indicated only when MI in ASCVD history",
				Urgency:      "URGENT",
				SourceTrial:  "CAPRICORN",
			})
		}

	case "4c":
		// Unknown HF subtype (no EF) — require echo BEFORE prescribing GDMT.
		if clinCtx.HFUnknownSubtype || hfType == "" {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass:      "ECHOCARDIOGRAM",
				Rationale:         "Stage 4c requires EF measurement before GDMT can be targeted (HFrEF/HFmrEF/HFpEF)",
				Urgency:           "IMMEDIATE",
				SourceTrial:       "AHA_ACC_HFSA_2022",
				SafetyPrecautions: "Do not initiate HF-specific therapy without EF — four-pillar GDMT is EF-dependent",
			})
			// Also flag SGLT2i — beneficial across EF spectrum, safe without EF knowledge
			if !activeSet["SGLT2i"] {
				gaps = append(gaps, MandatoryMedGap{
					MissingClass: "SGLT2i",
					Rationale:    "SGLT2i beneficial across HF spectrum — safe to initiate before EF known",
					Urgency:      "URGENT",
					SourceTrial:  "DAPA-HF, EMPEROR-Preserved",
				})
			}
			break
		}
		gaps = append(gaps, c.checkHFMandatory(hfType, activeSet, hasRAS, hasBB, hasARNI, hasACEiARB, clinCtx)...)
	}

	// Suppress gaps for classes that the renal gate has already blocked.
	// Generates a compound "refer for shared decision-making" note via Suppressed flag.
	if len(clinCtx.BlockedByRenal) > 0 {
		gaps = suppressRenalBlocked(gaps, clinCtx.BlockedByRenal)
	}

	return gaps
}

func hasEventType(events []string, target string) bool {
	for _, e := range events {
		if e == target {
			return true
		}
	}
	return false
}

// suppressRenalBlocked marks gaps as Suppressed when the renal gate has contraindicated
// the same drug class. The gap remains in the output (for audit/display) but is flagged
// so downstream card rendering generates a compound "renal-cardio shared decision" card
// rather than a straightforward "add medication" card.
func suppressRenalBlocked(gaps []MandatoryMedGap, blockedClasses []string) []MandatoryMedGap {
	blockedSet := make(map[string]bool)
	for _, c := range blockedClasses {
		blockedSet[c] = true
	}
	out := make([]MandatoryMedGap, len(gaps))
	for i, g := range gaps {
		out[i] = g
		// Match both exact class and common aliases
		if blockedSet[g.MissingClass] ||
			(g.MissingClass == "MRA" && (blockedSet["SPIRONOLACTONE"] || blockedSet["EPLERENONE"])) ||
			(g.MissingClass == "ARNI_OR_ACEi_ARB" && (blockedSet["ACEi"] || blockedSet["ARB"] || blockedSet["ARNI"] || blockedSet["SACUBITRIL_VALSARTAN"])) ||
			(g.MissingClass == "ACEi_OR_ARB" && (blockedSet["ACEi"] || blockedSet["ARB"])) {
			out[i].Suppressed = true
			out[i].SuppressionReason = "Renal gate has contraindicated this class — refer for renal-cardio shared decision-making"
		}
	}
	return out
}

func (c *MandatoryMedChecker) checkHFMandatory(
	hfType string,
	activeSet map[string]bool,
	hasRAS bool,
	hasBB bool,
	hasARNI bool,
	hasACEiARB bool,
	clinCtx ClinicalContext,
) []MandatoryMedGap {
	var gaps []MandatoryMedGap

	switch hfType {
	case "HFrEF":
		// Four pillars: ARNI/ACEi/ARB, BB, MRA, SGLT2i
		if !hasRAS {
			gap := MandatoryMedGap{
				MissingClass: "ARNI_OR_ACEi_ARB",
				Rationale:    "PARADIGM-HF: ARNI preferred; ACEi/ARB if not tolerated",
				Urgency:      "IMMEDIATE",
				SourceTrial:  "PARADIGM-HF",
			}
			// SBP-based safety precaution: ARNI requires SBP ≥100 mmHg
			if clinCtx.SBPmmHg > 0 && clinCtx.SBPmmHg < 100 {
				gap.Alternative = "ACEi or ARB (ARNI contraindicated at SBP <100 mmHg)"
				gap.SafetyPrecautions = "ARNI requires SBP ≥100 mmHg"
			} else {
				gap.SafetyPrecautions = "ARNI initiation: requires SBP ≥100 mmHg"
			}
			gaps = append(gaps, gap)
		} else if hasACEiARB && !hasARNI {
			// Patient on ACEi/ARB — flag ARNI upgrade with washout safety guard
			gap := MandatoryMedGap{
				MissingClass: "ARNI_UPGRADE",
				Rationale:    "PARADIGM-HF: sacubitril/valsartan superior to enalapril for HFrEF",
				Urgency:      "URGENT",
				SourceTrial:  "PARADIGM-HF",
				Alternative:  "Continue current ACEi/ARB if upgrade not feasible",
				SafetyPrecautions: "ARNI switch requires 36-hour ACEi washout (angioedema risk if overlapped); SBP ≥100 mmHg required",
			}
			if clinCtx.SBPmmHg > 0 && clinCtx.SBPmmHg < 100 {
				gap.Urgency = "ROUTINE"
				gap.SafetyPrecautions = "ARNI upgrade deferred: SBP <100 mmHg — optimize current ACEi/ARB dose first"
			}
			gaps = append(gaps, gap)
		}
		if !hasBB {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "BETA_BLOCKER_HF",
				Rationale:    "HFrEF mortality reduction — carvedilol/bisoprolol/metoprolol succinate",
				Urgency:      "IMMEDIATE",
				SourceTrial:  "CIBIS-II, MERIT-HF, COPERNICUS",
			})
		}
		if !activeSet["MRA"] && !activeSet["SPIRONOLACTONE"] && !activeSet["EPLERENONE"] {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "MRA",
				Rationale:    "HFrEF mortality + hospitalization reduction",
				Urgency:      "IMMEDIATE",
				SourceTrial:  "RALES, EMPHASIS-HF",
			})
		}
		if !activeSet["SGLT2i"] {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "SGLT2i",
				Rationale:    "HFrEF mortality + hospitalization reduction",
				Urgency:      "IMMEDIATE",
				SourceTrial:  "DAPA-HF, EMPEROR-Reduced",
			})
		}

	case "HFmrEF":
		if !activeSet["SGLT2i"] {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "SGLT2i",
				Rationale:    "Benefit across EF spectrum (DELIVER subgroup)",
				Urgency:      "URGENT",
				SourceTrial:  "DELIVER, EMPEROR-Preserved",
			})
		}
		if !hasRAS {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "ACEi_OR_ARB",
				Rationale:    "Reasonable from HFrEF extrapolation",
				Urgency:      "URGENT",
				SourceTrial:  "AHA_ACC_HFSA_2022",
			})
		}

	case "HFpEF":
		// ONLY mandatory: SGLT2i
		if !activeSet["SGLT2i"] {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "SGLT2i",
				Rationale:    "ONLY proven disease-modifying therapy for HFpEF (EMPEROR-Preserved, DELIVER)",
				Urgency:      "URGENT",
				SourceTrial:  "EMPEROR-Preserved, DELIVER",
			})
		}

	default:
		// Unknown HF type — at minimum, SGLT2i
		if !activeSet["SGLT2i"] {
			gaps = append(gaps, MandatoryMedGap{
				MissingClass: "SGLT2i",
				Rationale:    "SGLT2i beneficial across HF spectrum",
				Urgency:      "URGENT",
				SourceTrial:  "DAPA-HF, EMPEROR-Preserved",
			})
		}
	}

	return gaps
}
