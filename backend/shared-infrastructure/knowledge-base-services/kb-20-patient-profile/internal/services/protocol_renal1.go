package services

func (r *ProtocolRegistry) registerRENAL1() {
	r.templates["RENAL-1"] = &ProtocolTemplate{
		ProtocolID:   "RENAL-1",
		ProtocolName: "Renal Protection Protocol",
		Version:      "1.0.0",
		Category:     "medication",
		Subcategory:  "renal",
		IsLifelong:   true,
		SuccessMode:  SuccessModeNever,
		TargetMetric: "egfr_slope",
		GuidelineRef: "KDIGO 2024 CKD Guideline + FIDELITY/CREDENCE evidence",
		Targets: &TargetRange{
			Metric:      "egfr_slope",
			Unit:        "mL/min/1.73m2/yr",
			DefaultLow:  -3, // acceptable decline rate
			DefaultHigh: 0,  // stable or improving
			IndividualisedTargets: []IndividualisedTarget{
				{Archetype: "CKDProgressor", Low: -5, High: 0, Rationale: "KDIGO: rapid progressors tolerate steeper decline if ACR improving"},
				{Archetype: "ElderlyFrail", Low: -4, High: 0, Rationale: "Age-related decline accepted; focus on avoiding AKI"},
			},
		},
		Phases: []PhaseDefinition{
			{ID: "BASELINE", Name: "Assessment", DurationDays: 7, AutoAdvance: false, ActiveDrugSteps: []int{}},
			{ID: "RAAS_OPTIMISATION", Name: "ACEi/ARB Max Tolerated", DurationDays: 28, ExtendableTo: 56, ActiveDrugSteps: []int{1}},
			{ID: "SGLT2I_ADDITION", Name: "Add SGLT2i", DurationDays: 28, ExtendableTo: 42, ActiveDrugSteps: []int{1, 2}},
			{ID: "FINERENONE_ADDITION", Name: "Add Finerenone", DurationDays: 56, ExtendableTo: 84, ActiveDrugSteps: []int{1, 2, 3}},
			{ID: "MONITORING", Name: "Maintenance Monitoring", DurationDays: 0, ActiveDrugSteps: []int{1, 2, 3}}, // indefinite
		},
		DrugSequence: []DrugStep{
			{
				StepOrder:         1,
				DrugClass:         "acei_arb",
				DrugName:          "ramipril",
				StartingDoseMg:    2.5,
				DoseIncrementMg:   2.5,
				MaxDoseMg:         10,
				FrequencyPerDay:   1,
				EscalationTrigger: "acr_not_improving_4wk",
				ChannelBGuards:    []string{"B-03", "B-04"},
				ChannelCGuards:    []string{"PG-14"},
				OwningProtocol:    "HTN-1", // shared with HTN-1 which owns dose
				Notes:             "Titrate to max tolerated for ACR ≥ 300. Shared with HTN-1. KDIGO 2024 Rec 3.1.",
			},
			{
				StepOrder:         2,
				DrugClass:         "sglt2i",
				DrugName:          "dapagliflozin",
				StartingDoseMg:    10,
				DoseIncrementMg:   0,
				MaxDoseMg:         10,
				FrequencyPerDay:   1,
				EscalationTrigger: "renal_benefit_independent_of_hba1c",
				ChannelBGuards:    []string{"B-01", "B-04"},
				ChannelCGuards:    []string{"PG-04"},
				OwningProtocol:    "GLYC-1", // shared with GLYC-1 which owns dose
				Notes:             "Continue for renal protection even if HbA1c at target. Continue if already started even if eGFR <20. DAPA-CKD evidence.",
			},
			{
				StepOrder:         3,
				DrugClass:         "nsMRA",
				DrugName:          "finerenone",
				StartingDoseMg:    10,
				DoseIncrementMg:   10,
				MaxDoseMg:         20,
				FrequencyPerDay:   1,
				EscalationTrigger: "acr_persistently_elevated_on_raas_sglt2i",
				ChannelBGuards:    []string{"B-03"},
				ChannelCGuards:    []string{"PG-18", "PG-19"},
				Notes:             "Start 10mg if eGFR ≥ 60, target 20mg. Start 10mg if eGFR 25-59, stay at 10mg. FIDELITY/CREDENCE evidence. KDIGO 2024 Rec 3.3.",
			},
		},
		EntryCriteria: []Criterion{
			{Field: "egfr", Operator: "<", Value: 60},
			{Field: "acr", Operator: ">=", Value: 30},
			{Field: "has_ckd", Operator: "==", Value: 1},
		},
		ExclusionCriteria: []ExclusionRule{
			{Field: "potassium", Operator: ">=", Value: 5.0, RuleCode: "K-HIGH"},
			{Field: "egfr", Operator: "<", Value: 25, RuleCode: "EGFR-LOW"},
		},
		ConcurrentWith:    []string{"GLYC-1", "HTN-1", "LIPID-1", "M3-PRP", "M3-VFRP", "DEPRESC-1", "V-MCU"},
		EscalationTrigger: "egfr_rapid_decline_despite_triple_therapy",
	}
}
