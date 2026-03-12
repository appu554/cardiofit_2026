package testutils

import (
	"time"

	"kb-clinical-context/internal/models"
)

// PatientFixtures provides various patient scenarios for testing
type PatientFixtures struct{}

// NewPatientFixtures creates a new instance of patient fixtures
func NewPatientFixtures() *PatientFixtures {
	return &PatientFixtures{}
}

// CreateCardiovascularPatient creates a patient with cardiovascular conditions
func (pf *PatientFixtures) CreateCardiovascularPatient() models.PatientContext {
	return models.PatientContext{
		PatientID: "CV-001",
		ContextID: "ctx-cv-001",
		Timestamp: time.Now(),
		Demographics: models.Demographics{
			AgeYears:  68,
			Sex:       "M",
			Race:      "White",
			Ethnicity: "Not Hispanic or Latino",
		},
		ActiveConditions: []models.Condition{
			{
				Code:      "I10",
				System:    "ICD-10",
				Name:      "Essential hypertension",
				OnsetDate: time.Now().AddDate(-3, 0, 0),
				Severity:  "moderate",
			},
			{
				Code:      "I25.10",
				System:    "ICD-10",
				Name:      "Atherosclerotic heart disease of native coronary artery without angina pectoris",
				OnsetDate: time.Now().AddDate(-1, 0, 0),
				Severity:  "mild",
			},
		},
		RecentLabs: []models.LabResult{
			{
				LOINCCode:    "2093-3", // Total cholesterol
				Value:        280.0,
				Unit:         "mg/dL",
				ResultDate:   time.Now().AddDate(0, 0, -14),
				AbnormalFlag: "H",
			},
			{
				LOINCCode:    "2089-1", // LDL cholesterol
				Value:        180.0,
				Unit:         "mg/dL",
				ResultDate:   time.Now().AddDate(0, 0, -14),
				AbnormalFlag: "H",
			},
			{
				LOINCCode:    "8480-6", // Systolic BP
				Value:        158.0,
				Unit:         "mmHg",
				ResultDate:   time.Now().AddDate(0, 0, -2),
				AbnormalFlag: "H",
			},
			{
				LOINCCode:    "8462-4", // Diastolic BP
				Value:        95.0,
				Unit:         "mmHg",
				ResultDate:   time.Now().AddDate(0, 0, -2),
				AbnormalFlag: "H",
			},
		},
		CurrentMeds: []models.Medication{
			{
				RxNormCode: "161",
				Name:       "Lisinopril 10mg",
				Dose:       "10mg",
				Frequency:  "daily",
				StartDate:  time.Now().AddDate(0, -8, 0),
			},
			{
				RxNormCode: "36567",
				Name:       "Atorvastatin 40mg",
				Dose:       "40mg",
				Frequency:  "daily",
				StartDate:  time.Now().AddDate(0, -6, 0),
			},
		},
		RiskFactors: map[string]interface{}{
			"cardiovascular_risk": 0.85,
			"ascvd_10yr":         22.5,
		},
		TTL: time.Now().Add(24 * time.Hour),
	}
}

// CreateDiabeticPatient creates a patient with diabetes
func (pf *PatientFixtures) CreateDiabeticPatient() models.PatientContext {
	return models.PatientContext{
		PatientID: "DM-001",
		ContextID: "ctx-dm-001",
		Timestamp: time.Now(),
		Demographics: models.Demographics{
			AgeYears:  62,
			Sex:       "F",
			Race:      "Hispanic",
			Ethnicity: "Hispanic or Latino",
		},
		ActiveConditions: []models.Condition{
			{
				Code:      "E11.9",
				System:    "ICD-10",
				Name:      "Type 2 diabetes mellitus without complications",
				OnsetDate: time.Now().AddDate(-7, 0, 0),
				Severity:  "moderate",
			},
			{
				Code:      "I10",
				System:    "ICD-10",
				Name:      "Essential hypertension",
				OnsetDate: time.Now().AddDate(-4, 0, 0),
				Severity:  "mild",
			},
		},
		RecentLabs: []models.LabResult{
			{
				LOINCCode:    "4548-4", // HbA1c
				Value:        8.2,
				Unit:         "%",
				ResultDate:   time.Now().AddDate(0, 0, -21),
				AbnormalFlag: "H",
			},
			{
				LOINCCode:    "33747-0", // Glucose random
				Value:        185.0,
				Unit:         "mg/dL",
				ResultDate:   time.Now().AddDate(0, 0, -7),
				AbnormalFlag: "H",
			},
			{
				LOINCCode:    "2160-0", // Creatinine
				Value:        1.3,
				Unit:         "mg/dL",
				ResultDate:   time.Now().AddDate(0, 0, -14),
				AbnormalFlag: "H",
			},
			{
				LOINCCode:    "33914-3", // eGFR
				Value:        58.0,
				Unit:         "mL/min/1.73m2",
				ResultDate:   time.Now().AddDate(0, 0, -14),
				AbnormalFlag: "L",
			},
		},
		CurrentMeds: []models.Medication{
			{
				RxNormCode: "6918",
				Name:       "Metformin 1000mg",
				Dose:       "1000mg",
				Frequency:  "twice daily",
				StartDate:  time.Now().AddDate(-2, 0, 0),
			},
			{
				RxNormCode: "274783",
				Name:       "Glipizide 5mg",
				Dose:       "5mg",
				Frequency:  "daily",
				StartDate:  time.Now().AddDate(-1, 0, 0),
			},
			{
				RxNormCode: "161",
				Name:       "Lisinopril 5mg",
				Dose:       "5mg",
				Frequency:  "daily",
				StartDate:  time.Now().AddDate(0, -3, 0),
			},
		},
		RiskFactors: map[string]interface{}{
			"cardiovascular_risk": 0.75,
			"ade_risk":           0.6,
		},
		TTL: time.Now().Add(24 * time.Hour),
	}
}

// CreateCKDPatient creates a patient with chronic kidney disease
func (pf *PatientFixtures) CreateCKDPatient() models.PatientContext {
	return models.PatientContext{
		PatientID: "CKD-001",
		ContextID: "ctx-ckd-001",
		Timestamp: time.Now(),
		Demographics: models.Demographics{
			AgeYears:  75,
			Sex:       "M",
			Race:      "Black or African American",
			Ethnicity: "Not Hispanic or Latino",
		},
		ActiveConditions: []models.Condition{
			{
				Code:      "N18.3",
				System:    "ICD-10",
				Name:      "Chronic kidney disease, stage 3 (moderate)",
				OnsetDate: time.Now().AddDate(-2, 0, 0),
				Severity:  "moderate",
			},
			{
				Code:      "I10",
				System:    "ICD-10",
				Name:      "Essential hypertension",
				OnsetDate: time.Now().AddDate(-5, 0, 0),
				Severity:  "moderate",
			},
			{
				Code:      "E11.22",
				System:    "ICD-10",
				Name:      "Type 2 diabetes mellitus with diabetic chronic kidney disease",
				OnsetDate: time.Now().AddDate(-8, 0, 0),
				Severity:  "moderate",
			},
		},
		RecentLabs: []models.LabResult{
			{
				LOINCCode:    "2160-0", // Creatinine
				Value:        2.1,
				Unit:         "mg/dL",
				ResultDate:   time.Now().AddDate(0, 0, -10),
				AbnormalFlag: "H",
			},
			{
				LOINCCode:    "33914-3", // eGFR
				Value:        35.0,
				Unit:         "mL/min/1.73m2",
				ResultDate:   time.Now().AddDate(0, 0, -10),
				AbnormalFlag: "L",
			},
			{
				LOINCCode:    "14956-7", // Microalbumin
				Value:        125.0,
				Unit:         "mg/g",
				ResultDate:   time.Now().AddDate(0, 0, -21),
				AbnormalFlag: "H",
			},
			{
				LOINCCode:    "4548-4", // HbA1c
				Value:        7.1,
				Unit:         "%",
				ResultDate:   time.Now().AddDate(0, 0, -30),
				AbnormalFlag: "H",
			},
		},
		CurrentMeds: []models.Medication{
			{
				RxNormCode: "161",
				Name:       "Lisinopril 10mg",
				Dose:       "10mg",
				Frequency:  "daily",
				StartDate:  time.Now().AddDate(-1, 0, 0),
			},
			{
				RxNormCode: "6918",
				Name:       "Metformin 500mg",
				Dose:       "500mg",
				Frequency:  "twice daily",
				StartDate:  time.Now().AddDate(-2, 0, 0),
			},
		},
		RiskFactors: map[string]interface{}{
			"cardiovascular_risk": 0.9,
			"ade_risk":           0.8,
			"readmission_risk":   0.7,
		},
		TTL: time.Now().Add(24 * time.Hour),
	}
}

// CreateElderlyMultiMorbidPatient creates an elderly patient with multiple conditions
func (pf *PatientFixtures) CreateElderlyMultiMorbidPatient() models.PatientContext {
	return models.PatientContext{
		PatientID: "ELDER-001",
		ContextID: "ctx-elder-001",
		Timestamp: time.Now(),
		Demographics: models.Demographics{
			AgeYears:  82,
			Sex:       "F",
			Race:      "White",
			Ethnicity: "Not Hispanic or Latino",
		},
		ActiveConditions: []models.Condition{
			{
				Code:      "I50.9",
				System:    "ICD-10",
				Name:      "Heart failure, unspecified",
				OnsetDate: time.Now().AddDate(-1, -6, 0),
				Severity:  "moderate",
			},
			{
				Code:      "I48.91",
				System:    "ICD-10",
				Name:      "Unspecified atrial fibrillation",
				OnsetDate: time.Now().AddDate(-2, 0, 0),
				Severity:  "moderate",
			},
			{
				Code:      "E11.9",
				System:    "ICD-10",
				Name:      "Type 2 diabetes mellitus without complications",
				OnsetDate: time.Now().AddDate(-10, 0, 0),
				Severity:  "mild",
			},
			{
				Code:      "M79.3",
				System:    "ICD-10",
				Name:      "Panniculitis, unspecified",
				OnsetDate: time.Now().AddDate(0, -3, 0),
				Severity:  "mild",
			},
		},
		RecentLabs: []models.LabResult{
			{
				LOINCCode:    "42719-5", // NT-proBNP
				Value:        1850.0,
				Unit:         "pg/mL",
				ResultDate:   time.Now().AddDate(0, 0, -5),
				AbnormalFlag: "H",
			},
			{
				LOINCCode:    "4548-4", // HbA1c
				Value:        6.8,
				Unit:         "%",
				ResultDate:   time.Now().AddDate(0, 0, -45),
				AbnormalFlag: "H",
			},
			{
				LOINCCode:    "6301-6", // INR
				Value:        2.8,
				Unit:         "ratio",
				ResultDate:   time.Now().AddDate(0, 0, -3),
				AbnormalFlag: "H",
			},
			{
				LOINCCode:    "2160-0", // Creatinine
				Value:        1.4,
				Unit:         "mg/dL",
				ResultDate:   time.Now().AddDate(0, 0, -7),
				AbnormalFlag: "H",
			},
		},
		CurrentMeds: []models.Medication{
			{
				RxNormCode: "855332",
				Name:       "Warfarin 5mg",
				Dose:       "5mg",
				Frequency:  "daily",
				StartDate:  time.Now().AddDate(0, -8, 0),
			},
			{
				RxNormCode: "18631",
				Name:       "Furosemide 40mg",
				Dose:       "40mg",
				Frequency:  "daily",
				StartDate:  time.Now().AddDate(0, -4, 0),
			},
			{
				RxNormCode: "6918",
				Name:       "Metformin 500mg",
				Dose:       "500mg",
				Frequency:  "twice daily",
				StartDate:  time.Now().AddDate(-3, 0, 0),
			},
			{
				RxNormCode: "1998",
				Name:       "Digoxin 0.25mg",
				Dose:       "0.25mg",
				Frequency:  "daily",
				StartDate:  time.Now().AddDate(0, -2, 0),
			},
		},
		RiskFactors: map[string]interface{}{
			"cardiovascular_risk": 0.95,
			"fall_risk":          0.8,
			"ade_risk":           0.9,
			"readmission_risk":   0.85,
		},
		TTL: time.Now().Add(24 * time.Hour),
	}
}

// CreateHealthyPatient creates a healthy patient for baseline testing
func (pf *PatientFixtures) CreateHealthyPatient() models.PatientContext {
	return models.PatientContext{
		PatientID: "HEALTHY-001",
		ContextID: "ctx-healthy-001",
		Timestamp: time.Now(),
		Demographics: models.Demographics{
			AgeYears:  35,
			Sex:       "M",
			Race:      "Asian",
			Ethnicity: "Not Hispanic or Latino",
		},
		ActiveConditions: []models.Condition{},
		RecentLabs: []models.LabResult{
			{
				LOINCCode:    "2093-3", // Total cholesterol
				Value:        180.0,
				Unit:         "mg/dL",
				ResultDate:   time.Now().AddDate(0, 0, -30),
				AbnormalFlag: "",
			},
			{
				LOINCCode:    "4548-4", // HbA1c
				Value:        5.4,
				Unit:         "%",
				ResultDate:   time.Now().AddDate(0, 0, -90),
				AbnormalFlag: "",
			},
			{
				LOINCCode:    "8480-6", // Systolic BP
				Value:        118.0,
				Unit:         "mmHg",
				ResultDate:   time.Now().AddDate(0, 0, -1),
				AbnormalFlag: "",
			},
			{
				LOINCCode:    "8462-4", // Diastolic BP
				Value:        75.0,
				Unit:         "mmHg",
				ResultDate:   time.Now().AddDate(0, 0, -1),
				AbnormalFlag: "",
			},
		},
		CurrentMeds: []models.Medication{},
		RiskFactors: map[string]interface{}{
			"cardiovascular_risk": 0.1,
			"fall_risk":          0.05,
		},
		TTL: time.Now().Add(24 * time.Hour),
	}
}

// CreatePatientWithMissingData creates a patient with incomplete data for edge case testing
func (pf *PatientFixtures) CreatePatientWithMissingData() models.PatientContext {
	return models.PatientContext{
		PatientID: "INCOMPLETE-001",
		ContextID: "ctx-incomplete-001",
		Timestamp: time.Now(),
		Demographics: models.Demographics{
			AgeYears: 45,
			Sex:      "F",
			// Missing race and ethnicity
		},
		ActiveConditions: []models.Condition{
			{
				Code:   "I10",
				System: "ICD-10",
				Name:   "Essential hypertension",
				// Missing onset date and severity
			},
		},
		RecentLabs: []models.LabResult{
			{
				LOINCCode:  "2093-3",
				Value:      250.0,
				Unit:       "mg/dL",
				ResultDate: time.Now().AddDate(0, 0, -60), // Older lab
				// Missing abnormal flag
			},
		},
		CurrentMeds: []models.Medication{
			{
				RxNormCode: "161",
				Name:       "Lisinopril",
				// Missing dose, frequency, start date
			},
		},
		RiskFactors: map[string]interface{}{},
		TTL:         time.Now().Add(24 * time.Hour),
	}
}

// GetAllTestPatients returns all test patient scenarios
func (pf *PatientFixtures) GetAllTestPatients() []models.PatientContext {
	return []models.PatientContext{
		pf.CreateCardiovascularPatient(),
		pf.CreateDiabeticPatient(),
		pf.CreateCKDPatient(),
		pf.CreateElderlyMultiMorbidPatient(),
		pf.CreateHealthyPatient(),
		pf.CreatePatientWithMissingData(),
	}
}