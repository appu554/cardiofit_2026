package protocols

import (
	"github.com/cardiofit/kb-3-guidelines/pkg/models"
)

// Preventive Care Schedule Definitions per README specification

// PrenatalSchedule - ACOG Guidelines
var PrenatalSchedule = models.PreventiveSchedule{
	ScheduleID:  "PRENATAL-ACOG",
	Name:        "Prenatal Care - ACOG Guidelines",
	Description: "Comprehensive prenatal monitoring schedule",
	TargetPopulation: models.PopulationCriteria{
		Sex:        "F",
		Conditions: []string{"pregnancy"},
	},
	ScreeningItems: []models.ScreeningItem{
		{
			ItemID:         "initial_visit",
			Name:           "Initial Prenatal Visit",
			Recommendation: "Complete history, physical, and baseline labs",
			StartAge:       0, // N/A for pregnancy
			EndAge:         0,
			Interval: models.RecurrencePattern{
				Frequency:      models.FreqWeekly,
				Interval:       4,
				MaxOccurrences: 1, // One-time at 8 weeks
			},
			EvidenceGrade: "A",
			Source:        "ACOG",
		},
		{
			ItemID:         "first_trimester",
			Name:           "First Trimester Screening",
			Recommendation: "NT ultrasound, PAPP-A, free beta-hCG",
			StartAge:       0,
			EndAge:         0,
			Interval: models.RecurrencePattern{
				Frequency:      models.FreqWeekly,
				Interval:       11, // 11-14 weeks
				MaxOccurrences: 1,
			},
			EvidenceGrade: "A",
			Source:        "ACOG",
		},
		{
			ItemID:         "anatomy_scan",
			Name:           "Anatomy Ultrasound",
			Recommendation: "Detailed fetal anatomy scan",
			StartAge:       0,
			EndAge:         0,
			Interval: models.RecurrencePattern{
				Frequency:      models.FreqWeekly,
				Interval:       18, // 18-22 weeks
				MaxOccurrences: 1,
			},
			EvidenceGrade: "A",
			Source:        "ACOG",
		},
		{
			ItemID:         "gct",
			Name:           "Glucose Challenge Test",
			Recommendation: "Gestational diabetes screening",
			StartAge:       0,
			EndAge:         0,
			Interval: models.RecurrencePattern{
				Frequency:      models.FreqWeekly,
				Interval:       24, // 24-28 weeks
				MaxOccurrences: 1,
			},
			EvidenceGrade: "A",
			Source:        "ACOG",
		},
		{
			ItemID:         "gbs",
			Name:           "GBS Screening",
			Recommendation: "Group B Streptococcus culture",
			StartAge:       0,
			EndAge:         0,
			Interval: models.RecurrencePattern{
				Frequency:      models.FreqWeekly,
				Interval:       36, // 35-37 weeks
				MaxOccurrences: 1,
			},
			EvidenceGrade: "A",
			Source:        "CDC/ACOG",
		},
		{
			ItemID:         "nst",
			Name:           "Non-stress Test",
			Recommendation: "Fetal heart rate monitoring for high-risk pregnancies",
			StartAge:       0,
			EndAge:         0,
			Interval: models.RecurrencePattern{
				Frequency: models.FreqWeekly,
				Interval:  1, // Weekly from 32 weeks for high-risk
			},
			EvidenceGrade: "B",
			Source:        "ACOG",
		},
		{
			ItemID:         "visits_early",
			Name:           "Prenatal Visits (early)",
			Recommendation: "Monthly visits 8-28 weeks",
			StartAge:       0,
			EndAge:         0,
			Interval: models.RecurrencePattern{
				Frequency: models.FreqWeekly,
				Interval:  4, // Every 4 weeks
			},
			EvidenceGrade: "A",
			Source:        "ACOG",
		},
		{
			ItemID:         "visits_late",
			Name:           "Prenatal Visits (late)",
			Recommendation: "Weekly visits 36-40 weeks",
			StartAge:       0,
			EndAge:         0,
			Interval: models.RecurrencePattern{
				Frequency: models.FreqWeekly,
				Interval:  1, // Weekly
			},
			EvidenceGrade: "A",
			Source:        "ACOG",
		},
	},
}

// WellChildSchedule - AAP/EPSDT Guidelines
var WellChildSchedule = models.PreventiveSchedule{
	ScheduleID:  "WELLCHILD-AAP",
	Name:        "Well Child Care - AAP Bright Futures",
	Description: "Pediatric preventive care from birth to 21 years",
	TargetPopulation: models.PopulationCriteria{
		AgeMin: models.IntPtr(0),
		AgeMax: models.IntPtr(21),
	},
	ScreeningItems: []models.ScreeningItem{
		{
			ItemID:         "newborn",
			Name:           "Newborn Visit",
			Recommendation: "Initial newborn assessment, hearing screen, metabolic screen",
			StartAge:       0,
			EndAge:         0,
			Interval: models.RecurrencePattern{
				Frequency:      models.FreqDaily,
				Interval:       3, // First few days
				MaxOccurrences: 1,
			},
			Sex:           "any",
			EvidenceGrade: "A",
			Source:        "AAP",
		},
		{
			ItemID:         "well_infant",
			Name:           "Well Infant Visits",
			Recommendation: "Growth, development, immunizations at 1,2,4,6,9,12 months",
			StartAge:       0,
			EndAge:         1,
			Interval: models.RecurrencePattern{
				Frequency: models.FreqMonthly,
				Interval:  2, // Approximately every 2 months first year
			},
			Sex:           "any",
			EvidenceGrade: "A",
			Source:        "AAP",
		},
		{
			ItemID:         "well_toddler",
			Name:           "Well Toddler Visits",
			Recommendation: "Development, behavior, immunizations at 15,18,24,30 months",
			StartAge:       1,
			EndAge:         3,
			Interval: models.RecurrencePattern{
				Frequency: models.FreqMonthly,
				Interval:  6, // Every 6 months
			},
			Sex:           "any",
			EvidenceGrade: "A",
			Source:        "AAP",
		},
		{
			ItemID:         "well_child_annual",
			Name:           "Annual Well Child Visit",
			Recommendation: "Comprehensive annual assessment 3-21 years",
			StartAge:       3,
			EndAge:         21,
			Interval: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  1,
			},
			Sex:           "any",
			EvidenceGrade: "A",
			Source:        "AAP/EPSDT",
		},
		{
			ItemID:         "developmental",
			Name:           "Developmental Screening",
			Recommendation: "ASQ-3 or PEDS at 9, 18, and 30 months",
			StartAge:       0,
			EndAge:         3,
			Interval: models.RecurrencePattern{
				Frequency:      models.FreqMonthly,
				Interval:       9,
				MaxOccurrences: 3,
			},
			Sex:           "any",
			EvidenceGrade: "B",
			Source:        "AAP",
		},
		{
			ItemID:         "autism",
			Name:           "Autism Screening",
			Recommendation: "M-CHAT at 18 and 24 months",
			StartAge:       1,
			EndAge:         2,
			Interval: models.RecurrencePattern{
				Frequency:      models.FreqMonthly,
				Interval:       6,
				MaxOccurrences: 2,
			},
			Sex:           "any",
			EvidenceGrade: "B",
			Source:        "AAP",
		},
		{
			ItemID:         "vision",
			Name:           "Vision Screening",
			Recommendation: "Photo screening 1-5 years, visual acuity 5+ years",
			StartAge:       1,
			EndAge:         21,
			Interval: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  1,
			},
			Sex:           "any",
			EvidenceGrade: "B",
			Source:        "AAP/USPSTF",
		},
		{
			ItemID:         "hearing",
			Name:           "Hearing Screening",
			Recommendation: "Objective hearing screen at birth, 4, 5, 6, 8, 10 years",
			StartAge:       0,
			EndAge:         10,
			Interval: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  2,
			},
			Sex:           "any",
			EvidenceGrade: "B",
			Source:        "AAP",
		},
		{
			ItemID:         "lead",
			Name:           "Lead Screening",
			Recommendation: "Blood lead level at 12 and 24 months if at risk",
			StartAge:       1,
			EndAge:         2,
			Interval: models.RecurrencePattern{
				Frequency:      models.FreqMonthly,
				Interval:       12,
				MaxOccurrences: 2,
			},
			Sex:           "any",
			EvidenceGrade: "B",
			Source:        "AAP/CDC",
		},
		{
			ItemID:         "lipid",
			Name:           "Lipid Screening",
			Recommendation: "Universal lipid screening at 9-11 years and 17-21 years",
			StartAge:       9,
			EndAge:         21,
			Interval: models.RecurrencePattern{
				Frequency:      models.FreqYearly,
				Interval:       8,
				MaxOccurrences: 2,
			},
			Sex:           "any",
			EvidenceGrade: "B",
			Source:        "NHLBI/AAP",
		},
	},
}

// AdultPreventiveSchedule - USPSTF Guidelines
var AdultPreventiveSchedule = models.PreventiveSchedule{
	ScheduleID:  "ADULT-USPSTF",
	Name:        "Adult Preventive Care - USPSTF",
	Description: "Evidence-based preventive services for adults",
	TargetPopulation: models.PopulationCriteria{
		AgeMin: models.IntPtr(18),
	},
	ScreeningItems: []models.ScreeningItem{
		{
			ItemID:         "bp",
			Name:           "Blood Pressure Screening",
			Recommendation: "Screen for hypertension in adults 18+",
			StartAge:       18,
			EndAge:         100,
			Interval: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  1,
			},
			Sex:           "any",
			EvidenceGrade: "A",
			Source:        "USPSTF",
		},
		{
			ItemID:         "diabetes",
			Name:           "Diabetes Screening",
			Recommendation: "Screen for prediabetes/diabetes in adults 35-70 with overweight/obesity",
			StartAge:       35,
			EndAge:         70,
			Interval: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  3, // Every 3 years if normal
			},
			Sex:           "any",
			EvidenceGrade: "B",
			Source:        "USPSTF",
		},
		{
			ItemID:         "lipid",
			Name:           "Lipid Screening",
			Recommendation: "Lipid panel for cardiovascular risk assessment",
			StartAge:       40,
			EndAge:         75,
			Interval: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  5,
			},
			Sex:           "any",
			EvidenceGrade: "B",
			Source:        "USPSTF",
		},
		{
			ItemID:         "hiv",
			Name:           "HIV Screening",
			Recommendation: "One-time HIV screening for all adults 15-65",
			StartAge:       15,
			EndAge:         65,
			Interval: models.RecurrencePattern{
				Frequency:      models.FreqYearly,
				Interval:       1,
				MaxOccurrences: 1, // One-time unless high-risk
			},
			Sex:           "any",
			EvidenceGrade: "A",
			Source:        "USPSTF",
		},
		{
			ItemID:         "hep_c",
			Name:           "Hepatitis C Screening",
			Recommendation: "One-time HCV screening for adults 18-79",
			StartAge:       18,
			EndAge:         79,
			Interval: models.RecurrencePattern{
				Frequency:      models.FreqYearly,
				Interval:       1,
				MaxOccurrences: 1, // One-time
			},
			Sex:           "any",
			EvidenceGrade: "B",
			Source:        "USPSTF",
		},
		{
			ItemID:         "hep_b",
			Name:           "Hepatitis B Screening",
			Recommendation: "HBV screening for adults at increased risk",
			StartAge:       18,
			EndAge:         100,
			Interval: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  1,
			},
			Sex:           "any",
			EvidenceGrade: "B",
			Source:        "USPSTF",
		},
		{
			ItemID:         "depression",
			Name:           "Depression Screening",
			Recommendation: "Screen for depression in general adult population",
			StartAge:       18,
			EndAge:         100,
			Interval: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  1,
			},
			Sex:           "any",
			EvidenceGrade: "B",
			Source:        "USPSTF",
		},
		{
			ItemID:         "anxiety",
			Name:           "Anxiety Screening",
			Recommendation: "Screen for anxiety disorders in adults",
			StartAge:       18,
			EndAge:         64,
			Interval: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  1,
			},
			Sex:           "any",
			EvidenceGrade: "B",
			Source:        "USPSTF",
		},
		{
			ItemID:         "sti",
			Name:           "STI Screening",
			Recommendation: "Screen for chlamydia and gonorrhea in sexually active women under 25",
			StartAge:       15,
			EndAge:         24,
			Interval: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  1,
			},
			Sex:           "F",
			EvidenceGrade: "B",
			Source:        "USPSTF",
		},
		{
			ItemID:         "aaa",
			Name:           "AAA Screening",
			Recommendation: "One-time abdominal aortic aneurysm ultrasound for men 65-75 who have ever smoked",
			StartAge:       65,
			EndAge:         75,
			Interval: models.RecurrencePattern{
				Frequency:      models.FreqYearly,
				Interval:       1,
				MaxOccurrences: 1,
			},
			Sex:           "M",
			EvidenceGrade: "B",
			Source:        "USPSTF",
		},
		{
			ItemID:         "osteoporosis",
			Name:           "Osteoporosis Screening",
			Recommendation: "DXA screening for women 65+ or postmenopausal with risk factors",
			StartAge:       65,
			EndAge:         100,
			Interval: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  2, // Every 2 years if normal
			},
			Sex:           "F",
			EvidenceGrade: "B",
			Source:        "USPSTF",
		},
	},
}

// CancerScreeningSchedule - USPSTF/ACS Guidelines
var CancerScreeningSchedule = models.PreventiveSchedule{
	ScheduleID:  "CANCER-SCREENING",
	Name:        "Cancer Screening - USPSTF/ACS",
	Description: "Evidence-based cancer screening recommendations",
	TargetPopulation: models.PopulationCriteria{
		AgeMin: models.IntPtr(21),
	},
	ScreeningItems: []models.ScreeningItem{
		{
			ItemID:         "mammography",
			Name:           "Mammography",
			Recommendation: "Biennial screening mammography for women 50-74; consider starting at 40",
			StartAge:       50,
			EndAge:         74,
			Interval: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  2, // Every 2 years
			},
			Sex:           "F",
			EvidenceGrade: "B",
			Source:        "USPSTF",
		},
		{
			ItemID:         "colonoscopy",
			Name:           "Colonoscopy",
			Recommendation: "Colorectal cancer screening starting at age 45",
			StartAge:       45,
			EndAge:         75,
			Interval: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  10, // Every 10 years for colonoscopy
			},
			Sex:           "any",
			EvidenceGrade: "A",
			Source:        "USPSTF",
		},
		{
			ItemID:         "fit",
			Name:           "Fecal Immunochemical Test (FIT)",
			Recommendation: "Annual FIT as alternative to colonoscopy",
			StartAge:       45,
			EndAge:         75,
			Interval: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  1,
			},
			Sex:           "any",
			EvidenceGrade: "A",
			Source:        "USPSTF",
		},
		{
			ItemID:         "cervical",
			Name:           "Cervical Cancer Screening",
			Recommendation: "Pap smear every 3 years (21-29) or co-testing every 5 years (30-65)",
			StartAge:       21,
			EndAge:         65,
			Interval: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  3, // Every 3 years for Pap alone
			},
			Sex:           "F",
			EvidenceGrade: "A",
			Source:        "USPSTF",
		},
		{
			ItemID:         "lung_ldct",
			Name:           "Lung Cancer Screening (LDCT)",
			Recommendation: "Annual low-dose CT for adults 50-80 with 20+ pack-year smoking history",
			StartAge:       50,
			EndAge:         80,
			Interval: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  1,
			},
			Sex:           "any",
			EvidenceGrade: "B",
			Source:        "USPSTF",
		},
		{
			ItemID:         "prostate",
			Name:           "Prostate Cancer Screening",
			Recommendation: "Shared decision-making for PSA screening in men 55-69",
			StartAge:       55,
			EndAge:         69,
			Interval: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  2, // If PSA <2.5, every 2 years
			},
			Sex:           "M",
			EvidenceGrade: "C",
			Source:        "USPSTF",
		},
		{
			ItemID:         "skin",
			Name:           "Skin Cancer Screening",
			Recommendation: "Total body skin examination for high-risk individuals",
			StartAge:       18,
			EndAge:         100,
			Interval: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  1,
			},
			Sex:           "any",
			EvidenceGrade: "I", // Insufficient evidence for general population
			Source:        "AAD",
		},
	},
}

// ImmunizationSchedule - ACIP Guidelines
var ImmunizationSchedule = models.PreventiveSchedule{
	ScheduleID:  "IMMUNIZATIONS-ACIP",
	Name:        "Immunization Schedule - ACIP",
	Description: "CDC/ACIP recommended immunization schedule",
	TargetPopulation: models.PopulationCriteria{
		AgeMin: models.IntPtr(0),
	},
	ScreeningItems: []models.ScreeningItem{
		{
			ItemID:         "influenza",
			Name:           "Influenza Vaccine",
			Recommendation: "Annual influenza vaccination for everyone 6 months and older",
			StartAge:       0,
			EndAge:         100,
			Interval: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  1,
			},
			Sex:           "any",
			EvidenceGrade: "A",
			Source:        "ACIP",
		},
		{
			ItemID:         "covid",
			Name:           "COVID-19 Vaccine",
			Recommendation: "COVID-19 vaccination per current ACIP guidelines",
			StartAge:       0,
			EndAge:         100,
			Interval: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  1, // Annual boosters recommended
			},
			Sex:           "any",
			EvidenceGrade: "A",
			Source:        "ACIP",
		},
		{
			ItemID:         "tdap",
			Name:           "Tdap/Td Vaccine",
			Recommendation: "Tdap once, then Td booster every 10 years",
			StartAge:       11,
			EndAge:         100,
			Interval: models.RecurrencePattern{
				Frequency: models.FreqYearly,
				Interval:  10,
			},
			Sex:           "any",
			EvidenceGrade: "A",
			Source:        "ACIP",
		},
		{
			ItemID:         "mmr",
			Name:           "MMR Vaccine",
			Recommendation: "2 doses for adults born after 1957 without evidence of immunity",
			StartAge:       18,
			EndAge:         100,
			Interval: models.RecurrencePattern{
				Frequency:      models.FreqMonthly,
				Interval:       1,
				MaxOccurrences: 2,
			},
			Sex:           "any",
			EvidenceGrade: "A",
			Source:        "ACIP",
		},
		{
			ItemID:         "varicella",
			Name:           "Varicella Vaccine",
			Recommendation: "2 doses for adults without evidence of immunity",
			StartAge:       18,
			EndAge:         100,
			Interval: models.RecurrencePattern{
				Frequency:      models.FreqMonthly,
				Interval:       1,
				MaxOccurrences: 2,
			},
			Sex:           "any",
			EvidenceGrade: "A",
			Source:        "ACIP",
		},
		{
			ItemID:         "zoster",
			Name:           "Shingles Vaccine (Shingrix)",
			Recommendation: "2 doses of Shingrix for adults 50+",
			StartAge:       50,
			EndAge:         100,
			Interval: models.RecurrencePattern{
				Frequency:      models.FreqMonthly,
				Interval:       2, // 2-6 months between doses
				MaxOccurrences: 2,
			},
			Sex:           "any",
			EvidenceGrade: "A",
			Source:        "ACIP",
		},
		{
			ItemID:         "pneumo_ppsv23",
			Name:           "Pneumococcal PPSV23",
			Recommendation: "PPSV23 for adults 65+ or with risk factors",
			StartAge:       65,
			EndAge:         100,
			Interval: models.RecurrencePattern{
				Frequency:      models.FreqYearly,
				Interval:       5,
				MaxOccurrences: 2,
			},
			Sex:           "any",
			EvidenceGrade: "A",
			Source:        "ACIP",
		},
		{
			ItemID:         "pneumo_pcv20",
			Name:           "Pneumococcal PCV20",
			Recommendation: "PCV20 for adults 65+ (single dose preferred)",
			StartAge:       65,
			EndAge:         100,
			Interval: models.RecurrencePattern{
				Frequency:      models.FreqYearly,
				Interval:       1,
				MaxOccurrences: 1,
			},
			Sex:           "any",
			EvidenceGrade: "A",
			Source:        "ACIP",
		},
		{
			ItemID:         "hpv",
			Name:           "HPV Vaccine",
			Recommendation: "HPV vaccination through age 26; shared decision-making 27-45",
			StartAge:       11,
			EndAge:         26,
			Interval: models.RecurrencePattern{
				Frequency:      models.FreqMonthly,
				Interval:       6, // 0, 6-12 months (2 doses if started <15)
				MaxOccurrences: 3,
			},
			Sex:           "any",
			EvidenceGrade: "A",
			Source:        "ACIP",
		},
		{
			ItemID:         "hepatitis_a",
			Name:           "Hepatitis A Vaccine",
			Recommendation: "2-dose series for at-risk adults or all adults who desire protection",
			StartAge:       18,
			EndAge:         100,
			Interval: models.RecurrencePattern{
				Frequency:      models.FreqMonthly,
				Interval:       6, // 6-12 months apart
				MaxOccurrences: 2,
			},
			Sex:           "any",
			EvidenceGrade: "A",
			Source:        "ACIP",
		},
		{
			ItemID:         "hepatitis_b",
			Name:           "Hepatitis B Vaccine",
			Recommendation: "3-dose series for at-risk adults; now recommended for all adults 19-59",
			StartAge:       19,
			EndAge:         59,
			Interval: models.RecurrencePattern{
				Frequency:      models.FreqMonthly,
				Interval:       1, // 0, 1, 6 months
				MaxOccurrences: 3,
			},
			Sex:           "any",
			EvidenceGrade: "A",
			Source:        "ACIP",
		},
		{
			ItemID:         "rsv",
			Name:           "RSV Vaccine",
			Recommendation: "RSV vaccination for adults 60+ using shared clinical decision-making",
			StartAge:       60,
			EndAge:         100,
			Interval: models.RecurrencePattern{
				Frequency:      models.FreqYearly,
				Interval:       1,
				MaxOccurrences: 1, // Single dose currently
			},
			Sex:           "any",
			EvidenceGrade: "B",
			Source:        "ACIP",
		},
	},
}

// GetAllPreventiveSchedules returns all preventive care schedules
func GetAllPreventiveSchedules() []models.PreventiveSchedule {
	return []models.PreventiveSchedule{
		PrenatalSchedule,
		WellChildSchedule,
		AdultPreventiveSchedule,
		CancerScreeningSchedule,
		ImmunizationSchedule,
	}
}
