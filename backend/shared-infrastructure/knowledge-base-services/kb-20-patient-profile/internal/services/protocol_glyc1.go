package services

func (r *ProtocolRegistry) registerGLYC1() {
	r.templates["GLYC-1"] = &ProtocolTemplate{
		ProtocolID:   "GLYC-1",
		ProtocolName: "Glycaemic Management Protocol",
		Version:      "1.0.0",
		Category:     "medication",
		Subcategory:  "glycaemic",
		IsLifelong:   true,
		SuccessMode:  SuccessModeNever,
		TargetMetric: "hba1c",
		GuidelineRef: "ADA 2026 Standards of Care, Figure 9.4",
		Targets: &TargetRange{
			Metric:      "hba1c",
			Unit:        "%",
			DefaultLow:  0,
			DefaultHigh: 7.0,
			IndividualisedTargets: []IndividualisedTarget{
				{Archetype: "ElderlyFrail", Low: 0, High: 8.0, Rationale: "ADA 2026 Sec 6: relaxed target for frail elderly, avoiding hypoglycaemia"},
				{Archetype: "CKDProgressor", Low: 0, High: 7.0, Rationale: "Standard target; renal monitoring via RENAL-1"},
				{Archetype: "VisceralObese", Low: 0, High: 7.0, Rationale: "Standard target; weight-neutral agents preferred"},
				{Archetype: "YoungHealthy", Low: 0, High: 6.5, Rationale: "ADA 2026: tighter target if achievable without hypoglycaemia"},
			},
		},
		Phases: []PhaseDefinition{
			{ID: "BASELINE", Name: "Assessment", DurationDays: 7, AutoAdvance: false, ActiveDrugSteps: []int{}},
			{ID: "MONOTHERAPY", Name: "Metformin Titration", DurationDays: 84, ExtendableTo: 112, ActiveDrugSteps: []int{1}},
			{ID: "COMBINATION", Name: "Multi-Agent Titration", DurationDays: 168, ExtendableTo: 252, ActiveDrugSteps: []int{1, 2, 3}},
			{ID: "OPTIMIZATION", Name: "Target Maintenance", DurationDays: 0, ActiveDrugSteps: []int{1, 2, 3, 4, 5}}, // indefinite
		},
		DrugSequence: []DrugStep{
			{
				StepOrder:         1,
				DrugClass:         "biguanide",
				DrugName:          "metformin",
				StartingDoseMg:    500,
				DoseIncrementMg:   500,
				MaxDoseMg:         2000,
				FrequencyPerDay:   2,
				EscalationTrigger: "hba1c_above_target_12wk",
				ChannelBGuards:    []string{"B-01", "B-02", "B-11"},
				ChannelCGuards:    []string{"PG-04", "PG-07"},
				Notes:             "First-line if eGFR >= 30. Reduce dose if eGFR 30-45. ADA 2026 Rec 9.6.",
			},
			{
				StepOrder:         2,
				DrugClass:         "sglt2i",
				DrugName:          "dapagliflozin",
				StartingDoseMg:    10,
				DoseIncrementMg:   0, // single dose, no titration
				MaxDoseMg:         10,
				FrequencyPerDay:   1,
				EscalationTrigger: "added_for_cardiorenal_benefit",
				ChannelBGuards:    []string{"B-01", "B-04"},
				ChannelCGuards:    []string{"PG-04"},
				OwningProtocol:    "GLYC-1", // RENAL-1 shares but GLYC-1 owns dose
				Notes:             "Add independently of HbA1c for cardiorenal benefit. Continue even if eGFR drops below 20 if already started. ADA 2026 Rec 9.8.",
			},
			{
				StepOrder:           3,
				DrugClass:           "glp1ra",
				DrugName:            "semaglutide",
				StartingDoseMg:      0.25, // weekly
				DoseIncrementMg:     0.25,
				MaxDoseMg:           2.0,
				FrequencyPerDay:     0, // weekly dosing — 0 means non-daily
				EscalationTrigger:   "hba1c_above_target_12wk_on_step2",
				DeescalationTrigger: "hba1c_below_target_24wk",
				ChannelBGuards:      []string{"B-01", "B-02"},
				ChannelCGuards:      []string{"PG-04", "PG-07"},
				Notes:               "Preferred over insulin for initial injectable. Titrate monthly: 0.25 -> 0.5 -> 1.0 -> 2.0 mg/wk. ADA 2026 Rec 9.10.",
			},
			{
				StepOrder:         4,
				DrugClass:         "basal_insulin",
				DrugName:          "insulin_glargine",
				StartingDoseMg:    10, // 10 units/day (using Mg field for units)
				DoseIncrementMg:   2,  // 2 units every 3 days
				MaxDoseMg:         80, // units
				FrequencyPerDay:   1,
				EscalationTrigger: "hba1c_above_target_12wk_on_step3",
				ChannelBGuards:    []string{"B-01", "B-02", "B-11"},
				ChannelCGuards:    []string{"PG-04", "PG-07"},
				Notes:             "Start 10U/day bedtime. Titrate 2U every 3 days to FBG target 80-130 mg/dL. ADA 2026 Rec 9.12.",
			},
			{
				StepOrder:         5,
				DrugClass:         "intensification",
				DrugName:          "tirzepatide_or_prandial_insulin",
				StartingDoseMg:    0,
				DoseIncrementMg:   0,
				MaxDoseMg:         0, // physician-directed
				FrequencyPerDay:   0,
				EscalationTrigger: "hba1c_above_target_on_basal_insulin",
				ChannelBGuards:    []string{"B-01", "B-02"},
				ChannelCGuards:    []string{"PG-04"},
				Notes:             "Consider tirzepatide or prandial insulin. Physician-directed dosing. ADA 2026 Rec 9.14.",
			},
		},
		EntryCriteria: []Criterion{
			{Field: "hba1c", Operator: ">=", Value: 7.0},
			{Field: "has_diabetes", Operator: "==", Value: 1},
		},
		ExclusionCriteria: []ExclusionRule{
			{Field: "egfr", Operator: "<", Value: 30, RuleCode: "MET-RENAL"},
			{Field: "type1_diabetes", Operator: "==", Value: 1, RuleCode: "T1D-EXCL"},
			{Field: "dka_history_acute", Operator: "==", Value: 1, RuleCode: "DKA-EXCL"},
		},
		ConcurrentWith:    []string{"HTN-1", "RENAL-1", "LIPID-1", "M3-PRP", "M3-VFRP", "DEPRESC-1", "V-MCU"},
		EscalationTrigger: "hba1c_rising_despite_max_step",
	}
}
