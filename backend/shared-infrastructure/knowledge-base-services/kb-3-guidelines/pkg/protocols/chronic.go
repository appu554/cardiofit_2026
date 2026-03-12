package protocols

import (
	"github.com/cardiofit/kb-3-guidelines/pkg/models"
)

// Chronic Disease Schedule Definitions per README specification

// DiabetesSchedule - ADA Standards 2024
var DiabetesSchedule = models.ChronicSchedule{
	ScheduleID:      "DIABETES-ADA-2024",
	Name:            "Diabetes Management - ADA 2024",
	GuidelineSource: "ADA Standards of Care 2024",
	Description:     "Comprehensive diabetes monitoring and management schedule",
	MonitoringItems: []models.MonitoringItem{
		{
			ItemID: "hba1c",
			Name:   "HbA1c",
			Type:   models.ScheduleLab,
			Recurrence: models.RecurrencePattern{
				Frequency: models.FreqMonthly,
				Interval:  3, // Every 3 months
			},
		},
		{
			ItemID: "hba1c_controlled",
			Name:   "HbA1c (Well-controlled)",
			Type:   models.ScheduleLab,
			Recurrence: models.RecurrencePattern{
				Frequency: models.FreqMonthly,
				Interval:  6, // Every 6 months if well-controlled
			},
			Conditions: []models.Condition{
				{Type: "lab", Field: "hba1c", Operator: "<", Value: 7.0},
			},
		},
		{
			ItemID: "lipid_panel",
			Name:   "Lipid Panel",
			Type:   models.ScheduleLab,
			Recurrence: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  1,
			},
		},
		{
			ItemID: "eye_exam",
			Name:   "Dilated Eye Exam",
			Type:   models.ScheduleScreening,
			Recurrence: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  1,
			},
		},
		{
			ItemID: "foot_exam",
			Name:   "Comprehensive Foot Exam",
			Type:   models.ScheduleAssessment,
			Recurrence: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  1,
			},
		},
		{
			ItemID: "uacr",
			Name:   "Urine Albumin/Creatinine Ratio",
			Type:   models.ScheduleLab,
			Recurrence: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  1,
			},
		},
		{
			ItemID: "egfr",
			Name:   "eGFR",
			Type:   models.ScheduleLab,
			Recurrence: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  1,
			},
		},
		{
			ItemID: "bp_check",
			Name:   "Blood Pressure Check",
			Type:   models.ScheduleAssessment,
			Recurrence: models.RecurrencePattern{
				Frequency: models.FreqMonthly,
				Interval:  3,
			},
		},
	},
	FollowUpRules: []models.FollowUpRule{
		{
			RuleID:  "hba1c_elevated",
			Trigger: models.Condition{Type: "lab", Field: "hba1c", Operator: ">", Value: 9.0},
			Action:  "Schedule follow-up HbA1c in 3 months and medication review",
		},
	},
}

// HeartFailureSchedule - ACC/AHA/HFSA 2022
var HeartFailureSchedule = models.ChronicSchedule{
	ScheduleID:      "HF-ACCAHA-2022",
	Name:            "Heart Failure Management - ACC/AHA/HFSA 2022",
	GuidelineSource: "ACC/AHA/HFSA 2022 Heart Failure Guidelines",
	Description:     "Heart failure monitoring and follow-up schedule",
	MonitoringItems: []models.MonitoringItem{
		{
			ItemID: "followup_7d",
			Name:   "Post-discharge Follow-up",
			Type:   models.ScheduleAppointment,
			Recurrence: models.RecurrencePattern{
				Frequency:      models.FreqDaily,
				Interval:       7,
				MaxOccurrences: 1, // One-time within 7 days of discharge
			},
		},
		{
			ItemID: "bnp",
			Name:   "BNP/NT-proBNP",
			Type:   models.ScheduleLab,
			Recurrence: models.RecurrencePattern{
				Frequency: models.FreqMonthly,
				Interval:  3,
			},
		},
		{
			ItemID: "bmp",
			Name:   "Basic Metabolic Panel",
			Type:   models.ScheduleLab,
			Recurrence: models.RecurrencePattern{
				Frequency: models.FreqMonthly,
				Interval:  3,
			},
		},
		{
			ItemID: "weight_daily",
			Name:   "Daily Weight",
			Type:   models.ScheduleAssessment,
			Recurrence: models.RecurrencePattern{
				Frequency: models.FreqDaily,
				Interval:  1,
			},
		},
	},
	FollowUpRules: []models.FollowUpRule{
		{
			RuleID:  "k_raas",
			Trigger: models.Condition{Type: "medication", Field: "class", Operator: "=", Value: "RAAS"},
			Action:  "Check K+ 3-7 days after initiation/dose change",
		},
		{
			RuleID:  "weight_gain",
			Trigger: models.Condition{Type: "assessment", Field: "weight_gain_3d", Operator: ">", Value: 3},
			Action:  "Schedule urgent clinic visit for diuretic adjustment",
		},
	},
}

// CKDSchedule - KDIGO 2024
var CKDSchedule = models.ChronicSchedule{
	ScheduleID:      "CKD-KDIGO-2024",
	Name:            "CKD Management - KDIGO 2024",
	GuidelineSource: "KDIGO 2024 CKD Guidelines",
	Description:     "Chronic kidney disease monitoring based on stage",
	MonitoringItems: []models.MonitoringItem{
		{
			ItemID: "egfr_g1g2",
			Name:   "eGFR (Stage 1-2)",
			Type:   models.ScheduleLab,
			Recurrence: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  1,
			},
			Conditions: []models.Condition{
				{Type: "ckd_stage", Operator: "<=", Value: 2},
			},
		},
		{
			ItemID: "egfr_g3a",
			Name:   "eGFR (Stage 3a)",
			Type:   models.ScheduleLab,
			Recurrence: models.RecurrencePattern{
				Frequency: models.FreqMonthly,
				Interval:  6,
			},
			Conditions: []models.Condition{
				{Type: "ckd_stage", Operator: "=", Value: "3a"},
			},
		},
		{
			ItemID: "egfr_g3b_g4",
			Name:   "eGFR (Stage 3b-4)",
			Type:   models.ScheduleLab,
			Recurrence: models.RecurrencePattern{
				Frequency: models.FreqMonthly,
				Interval:  3,
			},
			Conditions: []models.Condition{
				{Type: "ckd_stage", Operator: ">=", Value: "3b"},
			},
		},
		{
			ItemID: "uacr_annual",
			Name:   "UACR",
			Type:   models.ScheduleLab,
			Recurrence: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  1,
			},
		},
	},
	FollowUpRules: []models.FollowUpRule{
		{
			RuleID:  "nephrology_referral",
			Trigger: models.Condition{Type: "lab", Field: "egfr", Operator: "<", Value: 30},
			Action:  "Refer to nephrology",
		},
	},
}

// AnticoagSchedule - CHEST Guidelines
var AnticoagSchedule = models.ChronicSchedule{
	ScheduleID:      "ANTICOAG-CHEST",
	Name:            "Anticoagulation Management - CHEST Guidelines",
	GuidelineSource: "CHEST Guidelines",
	Description:     "Warfarin INR monitoring schedule",
	MonitoringItems: []models.MonitoringItem{
		{
			ItemID: "inr_routine",
			Name:   "INR (routine)",
			Type:   models.ScheduleLab,
			Recurrence: models.RecurrencePattern{
				Frequency: models.FreqWeekly,
				Interval:  4, // Every 4 weeks when stable
			},
		},
		{
			ItemID: "inr_initiation",
			Name:   "INR (initiation phase)",
			Type:   models.ScheduleLab,
			Recurrence: models.RecurrencePattern{
				Frequency: models.FreqWeekly,
				Interval:  1,
			},
			Conditions: []models.Condition{
				{Type: "phase", Field: "anticoag_phase", Operator: "=", Value: "initiation"},
			},
		},
	},
	FollowUpRules: []models.FollowUpRule{
		{
			RuleID:  "inr_dose_change",
			Trigger: models.Condition{Type: "medication", Field: "dose_change", Operator: "=", Value: true},
			Action:  "Recheck INR 3-7 days after dose change",
		},
		{
			RuleID:  "inr_supratherapeutic",
			Trigger: models.Condition{Type: "lab", Field: "inr", Operator: ">", Value: 4.0},
			Action:  "Hold warfarin, recheck INR in 24-48 hours",
		},
	},
}

// COPDSchedule - GOLD 2024
var COPDSchedule = models.ChronicSchedule{
	ScheduleID:      "COPD-GOLD-2024",
	Name:            "COPD Management - GOLD 2024",
	GuidelineSource: "GOLD 2024 COPD Guidelines",
	Description:     "COPD monitoring and assessment schedule",
	MonitoringItems: []models.MonitoringItem{
		{
			ItemID: "cat_score",
			Name:   "CAT Score Assessment",
			Type:   models.ScheduleAssessment,
			Recurrence: models.RecurrencePattern{
				Frequency: models.FreqMonthly,
				Interval:  3, // Quarterly
			},
		},
		{
			ItemID: "spirometry",
			Name:   "Spirometry",
			Type:   models.ScheduleProcedure,
			Recurrence: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  1,
			},
		},
		{
			ItemID: "exacerbation_review",
			Name:   "Exacerbation History Review",
			Type:   models.ScheduleAssessment,
			Recurrence: models.RecurrencePattern{
				Frequency: models.FreqMonthly,
				Interval:  3,
			},
		},
	},
	FollowUpRules: []models.FollowUpRule{
		{
			RuleID:  "exacerbation",
			Trigger: models.Condition{Type: "event", Field: "exacerbation", Operator: "=", Value: true},
			Action:  "Schedule follow-up within 2-4 weeks post-exacerbation",
		},
	},
}

// HTNSchedule - ACC/AHA 2017
var HTNSchedule = models.ChronicSchedule{
	ScheduleID:      "HTN-ACCAHA-2017",
	Name:            "Hypertension Management - ACC/AHA 2017",
	GuidelineSource: "ACC/AHA 2017 Hypertension Guidelines",
	Description:     "Blood pressure monitoring schedule",
	MonitoringItems: []models.MonitoringItem{
		{
			ItemID: "bp_monthly",
			Name:   "BP Check (until at goal)",
			Type:   models.ScheduleAssessment,
			Recurrence: models.RecurrencePattern{
				Frequency: models.FreqMonthly,
				Interval:  1,
			},
			Conditions: []models.Condition{
				{Type: "status", Field: "bp_at_goal", Operator: "=", Value: false},
			},
		},
		{
			ItemID: "bp_maintenance",
			Name:   "BP Check (at goal)",
			Type:   models.ScheduleAssessment,
			Recurrence: models.RecurrencePattern{
				Frequency: models.FreqMonthly,
				Interval:  3, // Every 3-6 months when stable
			},
			Conditions: []models.Condition{
				{Type: "status", Field: "bp_at_goal", Operator: "=", Value: true},
			},
		},
		{
			ItemID: "bmp_annual",
			Name:   "Annual BMP",
			Type:   models.ScheduleLab,
			Recurrence: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  1,
			},
		},
	},
	FollowUpRules: []models.FollowUpRule{
		{
			RuleID:  "bp_uncontrolled",
			Trigger: models.Condition{Type: "assessment", Field: "bp_systolic", Operator: ">", Value: 180},
			Action:  "Schedule urgent follow-up within 1 week",
		},
	},
}

// GetAllChronicSchedules returns all chronic disease schedules
func GetAllChronicSchedules() []models.ChronicSchedule {
	return []models.ChronicSchedule{
		DiabetesSchedule,
		HeartFailureSchedule,
		CKDSchedule,
		AnticoagSchedule,
		COPDSchedule,
		HTNSchedule,
	}
}
