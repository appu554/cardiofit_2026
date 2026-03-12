// Package protocols defines clinical protocol definitions
package protocols

import (
	"time"

	"github.com/cardiofit/kb-3-guidelines/pkg/models"
)

// Acute Protocol Definitions per README specification

// SepsisProtocol - Surviving Sepsis Campaign 2021, CMS SEP-1
var SepsisProtocol = models.Protocol{
	ProtocolID:      "SEPSIS-SEP1-2021",
	Name:            "Sepsis Bundle - CMS SEP-1",
	Type:            models.ProtocolAcute,
	GuidelineSource: "Surviving Sepsis Campaign 2021",
	Version:         "2021.1",
	Description:     "Time-sensitive sepsis recognition and treatment bundle per CMS SEP-1 measure",
	Stages: []models.Stage{
		{
			StageID:     "recognition",
			Name:        "Sepsis Recognition",
			Description: "Initial sepsis screening and recognition",
			Order:       1,
			Actions: []models.Action{
				{ActionID: "screen", Name: "Sepsis Screening", Type: models.ActionAssessment, Required: true, Description: "Perform sepsis screening using validated tool"},
				{ActionID: "lactate_initial", Name: "Initial Lactate", Type: models.ActionLab, Required: true, Deadline: 30 * time.Minute, Description: "Measure serum lactate level"},
			},
		},
		{
			StageID:     "3h_bundle",
			Name:        "3-Hour Bundle",
			Description: "Initial resuscitation bundle - complete within 3 hours",
			Order:       2,
			Actions: []models.Action{
				{ActionID: "blood_cultures", Name: "Blood Cultures", Type: models.ActionLab, Required: true, Description: "Obtain blood cultures before antibiotics"},
				{ActionID: "antibiotics", Name: "Broad-spectrum Antibiotics", Type: models.ActionMedication, Required: true, Deadline: 1 * time.Hour, Description: "Administer broad-spectrum antibiotics"},
				{ActionID: "fluid_bolus", Name: "Crystalloid Fluid Bolus", Type: models.ActionMedication, Required: false, Deadline: 3 * time.Hour, Description: "30 mL/kg crystalloid for hypotension or lactate ≥4 mmol/L"},
			},
		},
		{
			StageID:     "6h_bundle",
			Name:        "6-Hour Bundle",
			Description: "Reassessment bundle - complete within 6 hours",
			Order:       3,
			Actions: []models.Action{
				{ActionID: "vasopressors", Name: "Vasopressors", Type: models.ActionMedication, Required: false, Deadline: 6 * time.Hour, Description: "Apply vasopressors if hypotension persists despite fluid resuscitation"},
				{ActionID: "lactate_repeat", Name: "Repeat Lactate", Type: models.ActionLab, Required: false, Deadline: 6 * time.Hour, Description: "Repeat lactate if initial lactate >2 mmol/L"},
				{ActionID: "reassess_volume", Name: "Reassess Volume Status", Type: models.ActionAssessment, Required: true, Deadline: 6 * time.Hour, Description: "Reassess volume status and tissue perfusion"},
			},
		},
	},
	Constraints: []models.TimeConstraint{
		{ConstraintID: "abx_1h", Action: "Administer antibiotics", Deadline: 1 * time.Hour, AlertThreshold: 45 * time.Minute, Severity: models.SeverityCritical, Reference: "CMS SEP-1"},
		{ConstraintID: "lactate_30m", Action: "Initial lactate measurement", Deadline: 30 * time.Minute, AlertThreshold: 20 * time.Minute, Severity: models.SeverityMajor, Reference: "SSC 2021"},
		{ConstraintID: "fluid_3h", Action: "Crystalloid bolus for hypotension/lactate≥4", Deadline: 3 * time.Hour, AlertThreshold: 2 * time.Hour, Severity: models.SeverityMajor, Reference: "CMS SEP-1"},
		{ConstraintID: "reassess_6h", Action: "Volume and perfusion reassessment", Deadline: 6 * time.Hour, AlertThreshold: 5 * time.Hour, Severity: models.SeverityMajor, Reference: "CMS SEP-1"},
	},
	EntryConditions: []models.Condition{
		{Type: "diagnosis", Field: "sepsis", Operator: "=", Value: true},
	},
	Active: true,
}

// StrokeProtocol - AHA/ASA 2019
var StrokeProtocol = models.Protocol{
	ProtocolID:      "STROKE-AHA-2019",
	Name:            "Acute Ischemic Stroke - AHA/ASA 2019",
	Type:            models.ProtocolAcute,
	GuidelineSource: "AHA/ASA 2019 Guidelines",
	Version:         "2019.1",
	Description:     "Time-critical acute ischemic stroke treatment pathway",
	Stages: []models.Stage{
		{
			StageID:     "door",
			Name:        "Door to Imaging",
			Order:       1,
			Actions: []models.Action{
				{ActionID: "triage", Name: "Stroke Team Activation", Type: models.ActionNotification, Required: true, Deadline: 5 * time.Minute},
				{ActionID: "ct_scan", Name: "Non-contrast CT", Type: models.ActionProcedure, Required: true, Deadline: 25 * time.Minute},
				{ActionID: "ct_interpret", Name: "CT Interpretation", Type: models.ActionAssessment, Required: true, Deadline: 45 * time.Minute},
			},
		},
		{
			StageID:     "treatment",
			Name:        "Treatment Decision",
			Order:       2,
			Actions: []models.Action{
				{ActionID: "tpa_decision", Name: "tPA Eligibility Assessment", Type: models.ActionAssessment, Required: true, Deadline: 50 * time.Minute},
				{ActionID: "tpa_admin", Name: "tPA Administration", Type: models.ActionMedication, Required: false, Deadline: 60 * time.Minute},
			},
		},
	},
	Constraints: []models.TimeConstraint{
		{ConstraintID: "ct_25min", Action: "Door-to-CT", Deadline: 25 * time.Minute, AlertThreshold: 20 * time.Minute, Severity: models.SeverityCritical, Reference: "AHA/ASA 2019"},
		{ConstraintID: "tpa_60min", Action: "Door-to-needle (tPA)", Deadline: 60 * time.Minute, AlertThreshold: 50 * time.Minute, Severity: models.SeverityCritical, Reference: "AHA/ASA 2019"},
		{ConstraintID: "tpa_window", Action: "tPA within treatment window", Deadline: 270 * time.Minute, AlertThreshold: 240 * time.Minute, Severity: models.SeverityCritical, Reference: "AHA/ASA 2019", Description: "4.5 hours from symptom onset"},
	},
	Active: true,
}

// STEMIProtocol - ACC/AHA 2013
var STEMIProtocol = models.Protocol{
	ProtocolID:      "STEMI-ACC-2013",
	Name:            "STEMI - ACC/AHA 2013",
	Type:            models.ProtocolAcute,
	GuidelineSource: "ACC/AHA 2013 STEMI Guidelines",
	Version:         "2013.1",
	Description:     "ST-elevation myocardial infarction treatment pathway",
	Stages: []models.Stage{
		{
			StageID:     "diagnosis",
			Name:        "Rapid Diagnosis",
			Order:       1,
			Actions: []models.Action{
				{ActionID: "ecg", Name: "12-lead ECG", Type: models.ActionProcedure, Required: true, Deadline: 10 * time.Minute},
				{ActionID: "stemi_activation", Name: "STEMI Team Activation", Type: models.ActionNotification, Required: true, Deadline: 15 * time.Minute},
			},
		},
		{
			StageID:     "reperfusion",
			Name:        "Reperfusion",
			Order:       2,
			Actions: []models.Action{
				{ActionID: "cath_lab", Name: "Transfer to Cath Lab", Type: models.ActionProcedure, Required: true, Deadline: 60 * time.Minute},
				{ActionID: "pci", Name: "Primary PCI", Type: models.ActionProcedure, Required: true, Deadline: 90 * time.Minute},
			},
		},
	},
	Constraints: []models.TimeConstraint{
		{ConstraintID: "ecg_10min", Action: "12-lead ECG acquisition", Deadline: 10 * time.Minute, AlertThreshold: 7 * time.Minute, Severity: models.SeverityCritical, Reference: "ACC/AHA 2013"},
		{ConstraintID: "d2b_90min", Action: "Door-to-balloon", Deadline: 90 * time.Minute, AlertThreshold: 75 * time.Minute, Severity: models.SeverityCritical, Reference: "ACC/AHA 2013"},
	},
	Active: true,
}

// DKAProtocol - ADA 2024
var DKAProtocol = models.Protocol{
	ProtocolID:      "DKA-ADA-2024",
	Name:            "Diabetic Ketoacidosis - ADA 2024",
	Type:            models.ProtocolAcute,
	GuidelineSource: "ADA Standards of Care 2024",
	Version:         "2024.1",
	Description:     "Diabetic ketoacidosis treatment protocol",
	Stages: []models.Stage{
		{
			StageID:     "initial",
			Name:        "Initial Management",
			Order:       1,
			Actions: []models.Action{
				{ActionID: "k_check", Name: "Potassium Check", Type: models.ActionLab, Required: true, Description: "Check K+ before starting insulin"},
				{ActionID: "fluid_resus", Name: "IV Fluid Resuscitation", Type: models.ActionMedication, Required: true, Deadline: 1 * time.Hour},
				{ActionID: "insulin_start", Name: "Insulin Infusion", Type: models.ActionMedication, Required: true, Description: "Start after K+ confirmed ≥3.3"},
			},
		},
		{
			StageID:     "monitoring",
			Name:        "Ongoing Monitoring",
			Order:       2,
			Actions: []models.Action{
				{ActionID: "glucose_hourly", Name: "Hourly Glucose", Type: models.ActionLab, Required: true},
				{ActionID: "bmp_q2h", Name: "BMP Every 2 Hours", Type: models.ActionLab, Required: true},
				{ActionID: "anion_gap", Name: "Anion Gap Monitoring", Type: models.ActionLab, Required: true},
			},
		},
		{
			StageID:     "transition",
			Name:        "Transition to SC Insulin",
			Order:       3,
			Actions: []models.Action{
				{ActionID: "sc_insulin", Name: "Subcutaneous Insulin", Type: models.ActionMedication, Required: true, Description: "Start SC insulin with 2h overlap"},
				{ActionID: "drip_overlap", Name: "IV-SC Overlap", Type: models.ActionMedication, Required: true, Deadline: 2 * time.Hour, Description: "Maintain IV insulin for 2 hours after SC dose"},
			},
		},
	},
	Constraints: []models.TimeConstraint{
		{ConstraintID: "k_before_insulin", Action: "K+ check before insulin", Deadline: 0, Severity: models.SeverityCritical, Reference: "ADA 2024", Description: "Must confirm K+ ≥3.3 before insulin"},
		{ConstraintID: "overlap_2h", Action: "2h IV-SC overlap on transition", Deadline: 2 * time.Hour, Severity: models.SeverityMajor, Reference: "ADA 2024"},
	},
	Active: true,
}

// TraumaProtocol - ATLS 10th Edition
var TraumaProtocol = models.Protocol{
	ProtocolID:      "TRAUMA-ATLS-10",
	Name:            "Trauma - ATLS 10th Edition",
	Type:            models.ProtocolAcute,
	GuidelineSource: "ATLS 10th Edition",
	Version:         "10.0",
	Description:     "Advanced Trauma Life Support protocol",
	Stages: []models.Stage{
		{
			StageID:     "primary",
			Name:        "Primary Survey",
			Order:       1,
			Actions: []models.Action{
				{ActionID: "airway", Name: "Airway Assessment", Type: models.ActionAssessment, Required: true},
				{ActionID: "breathing", Name: "Breathing Assessment", Type: models.ActionAssessment, Required: true},
				{ActionID: "circulation", Name: "Circulation Assessment", Type: models.ActionAssessment, Required: true},
				{ActionID: "disability", Name: "Disability Assessment", Type: models.ActionAssessment, Required: true},
				{ActionID: "exposure", Name: "Exposure/Environment", Type: models.ActionAssessment, Required: true},
			},
		},
		{
			StageID:     "resuscitation",
			Name:        "Resuscitation",
			Order:       2,
			Actions: []models.Action{
				{ActionID: "blood_products", Name: "Blood Products", Type: models.ActionMedication, Required: false},
				{ActionID: "txa", Name: "Tranexamic Acid", Type: models.ActionMedication, Required: false, Deadline: 3 * time.Hour, Description: "TXA within 3h of injury for significant hemorrhage"},
			},
		},
	},
	Constraints: []models.TimeConstraint{
		{ConstraintID: "txa_3h", Action: "TXA administration", Deadline: 3 * time.Hour, AlertThreshold: 2 * time.Hour, Severity: models.SeverityCritical, Reference: "CRASH-2 Trial", Description: "TXA must be given within 3h of injury"},
	},
	Active: true,
}

// PEProtocol - ESC 2019
var PEProtocol = models.Protocol{
	ProtocolID:      "PE-ESC-2019",
	Name:            "Pulmonary Embolism - ESC 2019",
	Type:            models.ProtocolAcute,
	GuidelineSource: "ESC 2019 PE Guidelines",
	Version:         "2019.1",
	Description:     "Pulmonary embolism diagnosis and treatment pathway",
	Stages: []models.Stage{
		{
			StageID:     "risk_stratification",
			Name:        "Risk Stratification",
			Order:       1,
			Actions: []models.Action{
				{ActionID: "pesi", Name: "PESI/sPESI Score", Type: models.ActionAssessment, Required: true},
				{ActionID: "echo", Name: "Echocardiography", Type: models.ActionProcedure, Required: false, Description: "For hemodynamically unstable patients"},
				{ActionID: "troponin", Name: "Troponin", Type: models.ActionLab, Required: true},
			},
		},
		{
			StageID:     "treatment",
			Name:        "Anticoagulation",
			Order:       2,
			Actions: []models.Action{
				{ActionID: "anticoag", Name: "Anticoagulation Initiation", Type: models.ActionMedication, Required: true, Deadline: 1 * time.Hour},
				{ActionID: "thrombolysis", Name: "Thrombolysis", Type: models.ActionMedication, Required: false, Description: "For high-risk/hemodynamically unstable"},
			},
		},
	},
	Constraints: []models.TimeConstraint{
		{ConstraintID: "anticoag_1h", Action: "Anticoagulation initiation", Deadline: 1 * time.Hour, AlertThreshold: 45 * time.Minute, Severity: models.SeverityCritical, Reference: "ESC 2019"},
	},
	Active: true,
}

// GetAllAcuteProtocols returns all acute protocol definitions
func GetAllAcuteProtocols() []models.Protocol {
	return []models.Protocol{
		SepsisProtocol,
		StrokeProtocol,
		STEMIProtocol,
		DKAProtocol,
		TraumaProtocol,
		PEProtocol,
	}
}
