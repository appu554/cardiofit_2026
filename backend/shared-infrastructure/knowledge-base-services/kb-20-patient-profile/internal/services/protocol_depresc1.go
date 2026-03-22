package services

func (r *ProtocolRegistry) registerDEPRESC1() {
	r.templates["DEPRESC-1"] = &ProtocolTemplate{
		ProtocolID:   "DEPRESC-1",
		ProtocolName: "Deprescribing Protocol",
		Version:      "1.0.0",
		Category:     "medication",
		Subcategory:  "deprescribing",
		IsLifelong:   false,
		SuccessMode:  SuccessModeAll,
		TargetMetric: "hba1c",
		GuidelineRef: "ADA 2026 Standards of Care Section 13 + Deprescribing Guidelines 2024",
		Targets: &TargetRange{
			Metric:      "hba1c",
			Unit:        "%",
			DefaultLow:  0,
			DefaultHigh: 8.0, // relaxed target during deprescribing
			IndividualisedTargets: []IndividualisedTarget{
				{Archetype: "ElderlyFrail", Low: 0, High: 8.5, Rationale: "Further relaxed for frail elderly during medication reduction"},
			},
		},
		Phases: []PhaseDefinition{
			{ID: "ASSESSMENT", Name: "Deprescribing Assessment", DurationDays: 7, AutoAdvance: false, ActiveDrugSteps: []int{}},
			{ID: "STEPDOWN", Name: "Medication Stepdown", DurationDays: 56, ExtendableTo: 84, ActiveDrugSteps: []int{1, 2, 3}},
			{ID: "MONITORING", Name: "Post-Stepdown Monitoring", DurationDays: 0, ActiveDrugSteps: []int{}}, // indefinite
		},
		DrugSequence: []DrugStep{
			{
				StepOrder:           1,
				DrugClass:           "sulfonylurea",
				DrugName:            "glipizide",
				StartingDoseMg:      0, // removal, not addition
				DoseIncrementMg:     0,
				MaxDoseMg:           0,
				FrequencyPerDay:     0,
				EscalationTrigger:   "immediate_removal",
				DeescalationTrigger: "su_discontinued",
				ChannelBGuards:      []string{"B-01", "B-02"},
				ChannelCGuards:      []string{},
				Notes:               "Remove SU first — highest hypoglycaemia risk. ADA 2026 Sec 13.",
			},
			{
				StepOrder:           2,
				DrugClass:           "basal_insulin",
				DrugName:            "insulin_any",
				StartingDoseMg:      0,
				DoseIncrementMg:     0,
				MaxDoseMg:           0,
				FrequencyPerDay:     0,
				EscalationTrigger:   "hba1c_below_6.5_on_insulin",
				DeescalationTrigger: "reduce_20pct_weekly",
				ChannelBGuards:      []string{"B-01", "B-02", "B-11"},
				ChannelCGuards:      []string{},
				Notes:               "Reduce insulin 20% per week while HbA1c < 6.5%. Target: simplest regimen with HbA1c < 8.0%. AD-09 suppression active.",
			},
			{
				StepOrder:           3,
				DrugClass:           "polypharmacy",
				DrugName:            "regimen_simplification",
				StartingDoseMg:      0,
				DoseIncrementMg:     0,
				MaxDoseMg:           0,
				FrequencyPerDay:     0,
				EscalationTrigger:   "pill_count_above_threshold",
				DeescalationTrigger: "consolidate_to_fewer_agents",
				ChannelBGuards:      []string{},
				ChannelCGuards:      []string{},
				Notes:               "Reduce total pill count. Consolidate combination agents where possible.",
			},
		},
		EntryCriteria: []Criterion{
			{Field: "age", Operator: ">=", Value: 75},
			{Field: "hba1c", Operator: "<", Value: 6.5},
			{Field: "on_insulin", Operator: "==", Value: 1},
		},
		ExclusionCriteria: []ExclusionRule{
			{Field: "type1_diabetes", Operator: "==", Value: 1, RuleCode: "T1D-EXCL"},
			{Field: "age", Operator: "<", Value: 75, RuleCode: "AGE-YOUNG"},
		},
		SuccessCriteria: []Criterion{
			{Field: "hba1c", Operator: "<", Value: 8.0},
			{Field: "pill_count_reduced", Operator: "==", Value: 1},
		},
		ConcurrentWith:    []string{"GLYC-1", "HTN-1", "RENAL-1", "V-MCU"},
		EscalationTrigger: "hba1c_rising_above_8.5_during_stepdown",
	}
}
