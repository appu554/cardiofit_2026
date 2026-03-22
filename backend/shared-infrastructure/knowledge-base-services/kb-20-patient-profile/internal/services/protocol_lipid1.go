package services

func (r *ProtocolRegistry) registerLIPID1() {
	r.templates["LIPID-1"] = &ProtocolTemplate{
		ProtocolID:   "LIPID-1",
		ProtocolName: "Lipid Management Protocol",
		Version:      "1.0.0",
		Category:     "medication",
		Subcategory:  "cv_risk",
		IsLifelong:   true,
		SuccessMode:  SuccessModeCardOnly,
		TargetMetric: "ldl",
		GuidelineRef: "ADA 2026 Section 10 + AHA PREVENT 2024",
		Targets: &TargetRange{
			Metric:      "ldl",
			Unit:        "mg/dL",
			DefaultLow:  0,
			DefaultHigh: 70,
			IndividualisedTargets: []IndividualisedTarget{
				{Archetype: "VeryHighRisk", Low: 0, High: 55, Rationale: "ASCVD event history or PREVENT ≥20%: LDL <55 per ESC/EAS"},
				{Archetype: "HighRisk", Low: 0, High: 70, Rationale: "Diabetes + risk factors: LDL <70 per ADA 2026 Rec 10.34"},
				{Archetype: "ModerateRisk", Low: 0, High: 100, Rationale: "Diabetes age 40-75 without additional risk factors"},
			},
		},
		Phases: []PhaseDefinition{
			{ID: "ASSESSMENT", Name: "Statin Assessment", DurationDays: 0, ActiveDrugSteps: []int{1}}, // single phase, card-only
		},
		DrugSequence: []DrugStep{
			{
				StepOrder:         1,
				DrugClass:         "statin",
				DrugName:          "atorvastatin",
				StartingDoseMg:    40,
				DoseIncrementMg:   40,
				MaxDoseMg:         80,
				FrequencyPerDay:   1,
				EscalationTrigger: "ldl_above_target_12wk",
				ChannelBGuards:    []string{},
				ChannelCGuards:    []string{"PG-22"},
				Notes:             "High-intensity statin for all T2DM age 40-75. Card-only: physician reviews STATIN_REVIEW card. ADA 2026 Rec 10.32.",
			},
		},
		EntryCriteria: []Criterion{
			{Field: "age", Operator: ">=", Value: 40},
			{Field: "has_diabetes", Operator: "==", Value: 1},
		},
		ExclusionCriteria: []ExclusionRule{
			{Field: "age", Operator: ">", Value: 75, RuleCode: "AGE-EXCL"},
			{Field: "statin_intolerance", Operator: "==", Value: 1, RuleCode: "STATIN-INTOL"},
		},
		ConcurrentWith:    []string{"GLYC-1", "HTN-1", "RENAL-1", "M3-PRP", "M3-VFRP", "V-MCU"},
		EscalationTrigger: "ldl_persistently_above_target",
	}
}
