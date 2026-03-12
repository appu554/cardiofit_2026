// Package models provides domain models for KB-19 Protocol Orchestrator.
package models

// ConflictMatrixEntry represents a single conflict rule between two protocols.
// These are loaded from YAML configuration files.
type ConflictMatrixEntry struct {
	// Unique ID for this conflict rule
	ID string `json:"id" yaml:"id"`

	// First protocol in the conflict
	ProtocolA string `json:"protocol_a" yaml:"protocol_a"`

	// Second protocol in the conflict
	ProtocolB string `json:"protocol_b" yaml:"protocol_b"`

	// Type of conflict
	ConflictType ConflictType `json:"conflict_type" yaml:"conflict_type"`

	// Description of the conflict
	Description string `json:"description" yaml:"description"`

	// Resolution rule
	Resolution ConflictResolutionRule `json:"resolution" yaml:"resolution"`

	// Severity of this conflict
	Severity string `json:"severity" yaml:"severity"` // LOW, MEDIUM, HIGH, CRITICAL

	// Clinical rationale for the resolution
	ClinicalRationale string `json:"clinical_rationale" yaml:"clinical_rationale"`

	// Citation for the resolution
	Citation string `json:"citation" yaml:"citation"`
}

// ConflictResolutionRule defines how to resolve a specific conflict.
type ConflictResolutionRule struct {
	// Winner of the conflict
	Winner string `json:"winner" yaml:"winner"`

	// Condition that must be true for this resolution to apply
	// This is a CQL expression or simple condition
	Condition string `json:"condition" yaml:"condition"`

	// What happens to the loser
	LoserOutcome string `json:"loser_outcome" yaml:"loser_outcome"` // DELAY, AVOID, MODIFY

	// Delay duration if applicable
	DelayDuration string `json:"delay_duration" yaml:"delay_duration"`

	// Alternative action for the loser
	AlternativeAction string `json:"alternative_action" yaml:"alternative_action"`

	// Confidence in this resolution (0.0 - 1.0)
	Confidence float64 `json:"confidence" yaml:"confidence"`
}

// ConflictMatrixConfig represents the full conflict matrix configuration file.
type ConflictMatrixConfig struct {
	// Version of the conflict matrix
	Version string `json:"version" yaml:"version"`

	// Last updated timestamp
	LastUpdated string `json:"last_updated" yaml:"last_updated"`

	// Description of this conflict matrix
	Description string `json:"description" yaml:"description"`

	// Conflict entries
	Conflicts []ConflictMatrixEntry `json:"conflicts" yaml:"conflicts"`
}

// PredefinedConflicts provides built-in conflict rules for common clinical scenarios.
var PredefinedConflicts = []ConflictMatrixEntry{
	// Hemodynamic conflicts
	{
		ID:           "CONFLICT_SEPSIS_VS_HF",
		ProtocolA:    "SEPSIS-FLUIDS",
		ProtocolB:    "HF-DIURESIS",
		ConflictType: ConflictHemodynamic,
		Description:  "Sepsis fluid resuscitation conflicts with heart failure diuresis",
		Resolution: ConflictResolutionRule{
			Winner:            "SEPSIS-FLUIDS",
			Condition:         "ShockState != NONE",
			LoserOutcome:      "DELAY",
			DelayDuration:     "until_hemodynamically_stable",
			AlternativeAction: "Monitor fluid status closely",
			Confidence:        0.95,
		},
		Severity:          "CRITICAL",
		ClinicalRationale: "Septic shock requires volume resuscitation; HF management delayed until stabilized",
		Citation:          "Surviving Sepsis Campaign Guidelines 2021",
	},
	{
		ID:           "CONFLICT_SEPSIS_VS_HF_NO_SHOCK",
		ProtocolA:    "SEPSIS-FLUIDS",
		ProtocolB:    "HF-DIURESIS",
		ConflictType: ConflictHemodynamic,
		Description:  "Sepsis without shock in patient with heart failure",
		Resolution: ConflictResolutionRule{
			Winner:            "COMBINED",
			Condition:         "ShockState == NONE AND HasHF",
			LoserOutcome:      "MODIFY",
			AlternativeAction: "Judicious fluid resuscitation with close monitoring",
			Confidence:        0.8,
		},
		Severity:          "HIGH",
		ClinicalRationale: "Balance infection treatment with volume status management",
		Citation:          "ACC/AHA Heart Failure Guidelines 2022",
	},

	// Anticoagulation conflicts
	{
		ID:           "CONFLICT_AFIB_VS_THROMBOCYTOPENIA",
		ProtocolA:    "AFIB-ANTICOAG",
		ProtocolB:    "THROMBOCYTOPENIA-MANAGEMENT",
		ConflictType: ConflictAnticoagulation,
		Description:  "AFib anticoagulation vs bleeding risk from thrombocytopenia",
		Resolution: ConflictResolutionRule{
			Winner:            "THROMBOCYTOPENIA-MANAGEMENT",
			Condition:         "Platelets < 50000",
			LoserOutcome:      "AVOID",
			AlternativeAction: "Hold anticoagulation until platelets recover",
			Confidence:        0.9,
		},
		Severity:          "CRITICAL",
		ClinicalRationale: "Severe thrombocytopenia creates unacceptable bleeding risk",
		Citation:          "ASH Thrombocytopenia Guidelines 2020",
	},
	{
		ID:           "CONFLICT_AFIB_VS_ICH",
		ProtocolA:    "AFIB-ANTICOAG",
		ProtocolB:    "INTRACRANIAL-HEMORRHAGE",
		ConflictType: ConflictNeurological,
		Description:  "AFib anticoagulation contraindicated with active ICH",
		Resolution: ConflictResolutionRule{
			Winner:            "INTRACRANIAL-HEMORRHAGE",
			Condition:         "HasActiveICH",
			LoserOutcome:      "AVOID",
			AlternativeAction: "Reversal agents if anticoagulated; LAA closure consideration later",
			Confidence:        0.99,
		},
		Severity:          "CRITICAL",
		ClinicalRationale: "Active intracranial hemorrhage is absolute contraindication to anticoagulation",
		Citation:          "AHA/ASA Stroke Guidelines 2019",
	},

	// Nephrotoxicity conflicts
	{
		ID:           "CONFLICT_NSAID_VS_AKI",
		ProtocolA:    "PAIN-NSAID",
		ProtocolB:    "AKI-PROTECTION",
		ConflictType: ConflictNephrotoxic,
		Description:  "NSAID use conflicts with AKI or renal protection",
		Resolution: ConflictResolutionRule{
			Winner:            "AKI-PROTECTION",
			Condition:         "AKIStage >= 1 OR eGFR < 30",
			LoserOutcome:      "AVOID",
			AlternativeAction: "Acetaminophen for pain; consider opioids if severe",
			Confidence:        0.95,
		},
		Severity:          "HIGH",
		ClinicalRationale: "NSAIDs reduce renal blood flow and worsen AKI",
		Citation:          "KDIGO AKI Guidelines 2012",
	},
	{
		ID:           "CONFLICT_AMINOGLYCOSIDE_VS_AKI",
		ProtocolA:    "SEPSIS-AMINOGLYCOSIDE",
		ProtocolB:    "AKI-PROTECTION",
		ConflictType: ConflictNephrotoxic,
		Description:  "Aminoglycoside use in patient with AKI",
		Resolution: ConflictResolutionRule{
			Winner:            "SEPSIS-AMINOGLYCOSIDE",
			Condition:         "HasSevereInfection AND NoAlternativeAntibiotic",
			LoserOutcome:      "MODIFY",
			AlternativeAction: "Use once-daily dosing, adjust for renal function, monitor levels",
			Confidence:        0.75,
		},
		Severity:          "HIGH",
		ClinicalRationale: "Severe infection may justify nephrotoxic antibiotics with precautions",
		Citation:          "IDSA Guidelines 2021",
	},

	// Pregnancy conflicts
	{
		ID:           "CONFLICT_ACE_VS_PREGNANCY",
		ProtocolA:    "HYPERTENSION-ACE",
		ProtocolB:    "PREGNANCY-SAFETY",
		ConflictType: ConflictPregnancy,
		Description:  "ACE inhibitors are teratogenic in pregnancy",
		Resolution: ConflictResolutionRule{
			Winner:            "PREGNANCY-SAFETY",
			Condition:         "IsPregnant",
			LoserOutcome:      "AVOID",
			AlternativeAction: "Switch to labetalol, nifedipine, or methyldopa",
			Confidence:        0.99,
		},
		Severity:          "CRITICAL",
		ClinicalRationale: "ACE inhibitors cause fetal renal dysgenesis and other anomalies",
		Citation:          "ACOG Hypertension in Pregnancy Guidelines 2020",
	},
	{
		ID:           "CONFLICT_WARFARIN_VS_PREGNANCY",
		ProtocolA:    "AFIB-ANTICOAG-WARFARIN",
		ProtocolB:    "PREGNANCY-SAFETY",
		ConflictType: ConflictPregnancy,
		Description:  "Warfarin is teratogenic in pregnancy",
		Resolution: ConflictResolutionRule{
			Winner:            "PREGNANCY-SAFETY",
			Condition:         "IsPregnant AND GestationalAge < 13",
			LoserOutcome:      "AVOID",
			AlternativeAction: "Use LMWH during first trimester and peripartum",
			Confidence:        0.99,
		},
		Severity:          "CRITICAL",
		ClinicalRationale: "Warfarin causes warfarin embryopathy in first trimester",
		Citation:          "ESC Guidelines on CVD in Pregnancy 2018",
	},

	// Metabolic conflicts
	{
		ID:           "CONFLICT_INSULIN_VS_HYPOGLYCEMIA",
		ProtocolA:    "DIABETES-INSULIN",
		ProtocolB:    "HYPOGLYCEMIA-MANAGEMENT",
		ConflictType: ConflictMetabolic,
		Description:  "Insulin therapy when patient is hypoglycemic",
		Resolution: ConflictResolutionRule{
			Winner:            "HYPOGLYCEMIA-MANAGEMENT",
			Condition:         "BloodGlucose < 70",
			LoserOutcome:      "DELAY",
			DelayDuration:     "until_glucose_normalized",
			AlternativeAction: "Treat hypoglycemia first, then resume modified insulin regimen",
			Confidence:        0.99,
		},
		Severity:          "CRITICAL",
		ClinicalRationale: "Hypoglycemia is immediately life-threatening",
		Citation:          "ADA Standards of Care 2024",
	},
}

// GetConflictsForProtocol returns all conflicts involving a specific protocol.
func GetConflictsForProtocol(protocolID string) []ConflictMatrixEntry {
	var result []ConflictMatrixEntry
	for _, conflict := range PredefinedConflicts {
		if conflict.ProtocolA == protocolID || conflict.ProtocolB == protocolID {
			result = append(result, conflict)
		}
	}
	return result
}

// FindConflict looks for a conflict between two specific protocols.
func FindConflict(protocolA, protocolB string) *ConflictMatrixEntry {
	for _, conflict := range PredefinedConflicts {
		if (conflict.ProtocolA == protocolA && conflict.ProtocolB == protocolB) ||
			(conflict.ProtocolA == protocolB && conflict.ProtocolB == protocolA) {
			return &conflict
		}
	}
	return nil
}
