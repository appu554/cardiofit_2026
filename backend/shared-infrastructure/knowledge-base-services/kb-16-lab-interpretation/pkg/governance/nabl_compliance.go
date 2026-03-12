// Package governance provides compliance and regulatory functionality
// nabl_compliance.go implements NABL 112:2022 critical value notification compliance
package governance

import (
	"time"
)

// =============================================================================
// NABL 112:2022 CRITICAL VALUE COMPLIANCE
// National Accreditation Board for Testing and Calibration Laboratories
// Specific Requirements for Medical Laboratories (ISO 15189:2022 aligned)
// =============================================================================

// NABLCriticalValuePolicy defines critical value notification requirements
type NABLCriticalValuePolicy struct {
	TestCode                string   `json:"testCode"`
	TestName                string   `json:"testName"`
	Unit                    string   `json:"unit"`
	CriticalLow             *float64 `json:"criticalLow,omitempty"`
	CriticalHigh            *float64 `json:"criticalHigh,omitempty"`
	NotificationTimeMinutes int      `json:"notificationTimeMinutes"` // Max time to notify
	NotificationType        string   `json:"notificationType"`        // IMMEDIATE, 30_MIN, 60_MIN
	RepeatCritical          bool     `json:"repeatCritical"`          // Report repeat critical?
	RequiresReadBack        bool     `json:"requiresReadBack"`        // Verbal read-back required
	NABLCompliant           bool     `json:"nablCompliant"`
	DocumentationRequired   []string `json:"documentationRequired"`
	Governance              NABLGovernance `json:"governance"`
}

// NABLGovernance tracks NABL compliance metadata
type NABLGovernance struct {
	Standard         string `json:"standard"`          // NABL 112:2022
	Section          string `json:"section"`           // Section reference
	Clause           string `json:"clause"`            // Specific clause
	EffectiveDate    string `json:"effectiveDate"`
	AuditRequirement string `json:"auditRequirement"`
	ReviewFrequency  string `json:"reviewFrequency"`   // Annual review required
}

// NABLCriticalValueNotification records a critical value notification event
type NABLCriticalValueNotification struct {
	// Event Identification
	NotificationID    string    `json:"notificationId"`
	LabAccessionNumber string   `json:"labAccessionNumber"`
	TestCode          string    `json:"testCode"`
	TestName          string    `json:"testName"`

	// Critical Value Details
	Value             float64   `json:"value"`
	Unit              string    `json:"unit"`
	CriticalType      string    `json:"criticalType"` // HIGH, LOW
	PreviousCritical  bool      `json:"previousCritical"` // Was previous also critical?

	// Timing Compliance
	ResultTime        time.Time `json:"resultTime"`
	NotificationTime  time.Time `json:"notificationTime"`
	TimeToNotifyMin   int       `json:"timeToNotifyMin"`
	WithinTimeLimit   bool      `json:"withinTimeLimit"`
	RequiredTimeLimit int       `json:"requiredTimeLimit"` // in minutes

	// Notification Details
	NotifiedTo        string    `json:"notifiedTo"`        // Name of person notified
	NotifierName      string    `json:"notifierName"`      // Lab staff who notified
	NotificationMethod string   `json:"notificationMethod"` // PHONE, FAX, DIRECT, EMR
	ReadBackCompleted bool      `json:"readBackCompleted"`
	ReadBackValue     string    `json:"readBackValue,omitempty"` // Value read back

	// Documentation
	AttemptsMade      int       `json:"attemptsMade"`
	FailedAttempts    []NABLFailedAttempt `json:"failedAttempts,omitempty"`
	EscalatedTo       string    `json:"escalatedTo,omitempty"`
	Notes             string    `json:"notes,omitempty"`

	// Compliance Status
	NABLCompliant     bool      `json:"nablCompliant"`
	ComplianceIssues  []string  `json:"complianceIssues,omitempty"`
}

// NABLFailedAttempt records a failed notification attempt
type NABLFailedAttempt struct {
	AttemptTime   time.Time `json:"attemptTime"`
	Method        string    `json:"method"`
	Reason        string    `json:"reason"`
	ContactTried  string    `json:"contactTried"`
}

// =============================================================================
// NABL CRITICAL VALUE LIST (NABL 112:2022 Mandatory Critical Values)
// =============================================================================

// NABLCriticalValues contains the mandatory critical value list per NABL 112:2022
var NABLCriticalValues = map[string]NABLCriticalValuePolicy{
	// Hematology
	"4544-3": { // Hematocrit
		TestCode:                "4544-3",
		TestName:                "Hematocrit",
		Unit:                    "%",
		CriticalLow:             floatPtr(15.0),
		CriticalHigh:            floatPtr(60.0),
		NotificationTimeMinutes: 30,
		NotificationType:        "30_MIN",
		RequiresReadBack:        true,
		NABLCompliant:           true,
		DocumentationRequired:   []string{"time_result", "time_notification", "person_notified", "read_back"},
		Governance: NABLGovernance{
			Standard:         "NABL 112:2022",
			Section:          "5.9.1",
			Clause:           "Critical value notification",
			EffectiveDate:    "2022-07-01",
			AuditRequirement: "Monthly audit of notification times",
			ReviewFrequency:  "Annual",
		},
	},
	"718-7": { // Hemoglobin
		TestCode:                "718-7",
		TestName:                "Hemoglobin",
		Unit:                    "g/dL",
		CriticalLow:             floatPtr(6.0),
		CriticalHigh:            floatPtr(20.0),
		NotificationTimeMinutes: 30,
		NotificationType:        "30_MIN",
		RequiresReadBack:        true,
		NABLCompliant:           true,
		DocumentationRequired:   []string{"time_result", "time_notification", "person_notified", "read_back"},
		Governance: NABLGovernance{
			Standard:         "NABL 112:2022",
			Section:          "5.9.1",
			Clause:           "Critical value notification",
			EffectiveDate:    "2022-07-01",
			AuditRequirement: "Monthly audit of notification times",
			ReviewFrequency:  "Annual",
		},
	},
	"777-3": { // Platelets
		TestCode:                "777-3",
		TestName:                "Platelets",
		Unit:                    "x10^9/L",
		CriticalLow:             floatPtr(20.0),
		CriticalHigh:            floatPtr(1000.0),
		NotificationTimeMinutes: 30,
		NotificationType:        "30_MIN",
		RequiresReadBack:        true,
		NABLCompliant:           true,
		DocumentationRequired:   []string{"time_result", "time_notification", "person_notified", "read_back"},
		Governance: NABLGovernance{
			Standard:         "NABL 112:2022",
			Section:          "5.9.1",
			Clause:           "Critical value notification",
			EffectiveDate:    "2022-07-01",
			AuditRequirement: "Monthly audit of notification times",
			ReviewFrequency:  "Annual",
		},
	},
	"26464-8": { // WBC
		TestCode:                "26464-8",
		TestName:                "WBC",
		Unit:                    "x10^9/L",
		CriticalLow:             floatPtr(2.0),
		CriticalHigh:            floatPtr(30.0),
		NotificationTimeMinutes: 30,
		NotificationType:        "30_MIN",
		RequiresReadBack:        true,
		NABLCompliant:           true,
		DocumentationRequired:   []string{"time_result", "time_notification", "person_notified", "read_back"},
		Governance: NABLGovernance{
			Standard:         "NABL 112:2022",
			Section:          "5.9.1",
			Clause:           "Critical value notification",
			EffectiveDate:    "2022-07-01",
			AuditRequirement: "Monthly audit of notification times",
			ReviewFrequency:  "Annual",
		},
	},

	// Chemistry
	"2345-7": { // Glucose
		TestCode:                "2345-7",
		TestName:                "Glucose",
		Unit:                    "mg/dL",
		CriticalLow:             floatPtr(40.0),
		CriticalHigh:            floatPtr(500.0),
		NotificationTimeMinutes: 30,
		NotificationType:        "30_MIN",
		RequiresReadBack:        true,
		NABLCompliant:           true,
		DocumentationRequired:   []string{"time_result", "time_notification", "person_notified", "read_back"},
		Governance: NABLGovernance{
			Standard:         "NABL 112:2022",
			Section:          "5.9.1",
			Clause:           "Critical value notification",
			EffectiveDate:    "2022-07-01",
			AuditRequirement: "Monthly audit of notification times",
			ReviewFrequency:  "Annual",
		},
	},
	"2823-3": { // Potassium
		TestCode:                "2823-3",
		TestName:                "Potassium",
		Unit:                    "mEq/L",
		CriticalLow:             floatPtr(2.5),
		CriticalHigh:            floatPtr(6.5),
		NotificationTimeMinutes: 30,
		NotificationType:        "IMMEDIATE",
		RequiresReadBack:        true,
		NABLCompliant:           true,
		DocumentationRequired:   []string{"time_result", "time_notification", "person_notified", "read_back"},
		Governance: NABLGovernance{
			Standard:         "NABL 112:2022",
			Section:          "5.9.1",
			Clause:           "Critical value notification",
			EffectiveDate:    "2022-07-01",
			AuditRequirement: "Monthly audit of notification times",
			ReviewFrequency:  "Annual",
		},
	},
	"2951-2": { // Sodium
		TestCode:                "2951-2",
		TestName:                "Sodium",
		Unit:                    "mEq/L",
		CriticalLow:             floatPtr(120.0),
		CriticalHigh:            floatPtr(160.0),
		NotificationTimeMinutes: 30,
		NotificationType:        "30_MIN",
		RequiresReadBack:        true,
		NABLCompliant:           true,
		DocumentationRequired:   []string{"time_result", "time_notification", "person_notified", "read_back"},
		Governance: NABLGovernance{
			Standard:         "NABL 112:2022",
			Section:          "5.9.1",
			Clause:           "Critical value notification",
			EffectiveDate:    "2022-07-01",
			AuditRequirement: "Monthly audit of notification times",
			ReviewFrequency:  "Annual",
		},
	},
	"2000-8": { // Calcium
		TestCode:                "2000-8",
		TestName:                "Calcium",
		Unit:                    "mg/dL",
		CriticalLow:             floatPtr(6.0),
		CriticalHigh:            floatPtr(13.0),
		NotificationTimeMinutes: 30,
		NotificationType:        "30_MIN",
		RequiresReadBack:        true,
		NABLCompliant:           true,
		DocumentationRequired:   []string{"time_result", "time_notification", "person_notified", "read_back"},
		Governance: NABLGovernance{
			Standard:         "NABL 112:2022",
			Section:          "5.9.1",
			Clause:           "Critical value notification",
			EffectiveDate:    "2022-07-01",
			AuditRequirement: "Monthly audit of notification times",
			ReviewFrequency:  "Annual",
		},
	},
	"2160-0": { // Creatinine
		TestCode:                "2160-0",
		TestName:                "Creatinine",
		Unit:                    "mg/dL",
		CriticalLow:             nil, // No critical low
		CriticalHigh:            floatPtr(10.0),
		NotificationTimeMinutes: 30,
		NotificationType:        "30_MIN",
		RequiresReadBack:        true,
		NABLCompliant:           true,
		DocumentationRequired:   []string{"time_result", "time_notification", "person_notified", "read_back"},
		Governance: NABLGovernance{
			Standard:         "NABL 112:2022",
			Section:          "5.9.1",
			Clause:           "Critical value notification",
			EffectiveDate:    "2022-07-01",
			AuditRequirement: "Monthly audit of notification times",
			ReviewFrequency:  "Annual",
		},
	},
	"1975-2": { // Bilirubin Total
		TestCode:                "1975-2",
		TestName:                "Bilirubin, Total",
		Unit:                    "mg/dL",
		CriticalLow:             nil,
		CriticalHigh:            floatPtr(15.0),
		NotificationTimeMinutes: 30,
		NotificationType:        "30_MIN",
		RequiresReadBack:        true,
		NABLCompliant:           true,
		DocumentationRequired:   []string{"time_result", "time_notification", "person_notified", "read_back"},
		Governance: NABLGovernance{
			Standard:         "NABL 112:2022",
			Section:          "5.9.1",
			Clause:           "Critical value notification",
			EffectiveDate:    "2022-07-01",
			AuditRequirement: "Monthly audit of notification times",
			ReviewFrequency:  "Annual",
		},
	},

	// Blood Gases
	"2744-1": { // pH
		TestCode:                "2744-1",
		TestName:                "pH (Blood)",
		Unit:                    "",
		CriticalLow:             floatPtr(7.20),
		CriticalHigh:            floatPtr(7.60),
		NotificationTimeMinutes: 15,
		NotificationType:        "IMMEDIATE",
		RequiresReadBack:        true,
		NABLCompliant:           true,
		DocumentationRequired:   []string{"time_result", "time_notification", "person_notified", "read_back"},
		Governance: NABLGovernance{
			Standard:         "NABL 112:2022",
			Section:          "5.9.1",
			Clause:           "Critical value notification",
			EffectiveDate:    "2022-07-01",
			AuditRequirement: "Monthly audit of notification times",
			ReviewFrequency:  "Annual",
		},
	},
	"2019-8": { // pCO2
		TestCode:                "2019-8",
		TestName:                "pCO2",
		Unit:                    "mmHg",
		CriticalLow:             floatPtr(20.0),
		CriticalHigh:            floatPtr(70.0),
		NotificationTimeMinutes: 15,
		NotificationType:        "IMMEDIATE",
		RequiresReadBack:        true,
		NABLCompliant:           true,
		DocumentationRequired:   []string{"time_result", "time_notification", "person_notified", "read_back"},
		Governance: NABLGovernance{
			Standard:         "NABL 112:2022",
			Section:          "5.9.1",
			Clause:           "Critical value notification",
			EffectiveDate:    "2022-07-01",
			AuditRequirement: "Monthly audit of notification times",
			ReviewFrequency:  "Annual",
		},
	},
	"2703-7": { // pO2
		TestCode:                "2703-7",
		TestName:                "pO2",
		Unit:                    "mmHg",
		CriticalLow:             floatPtr(40.0),
		CriticalHigh:            nil, // No critical high
		NotificationTimeMinutes: 15,
		NotificationType:        "IMMEDIATE",
		RequiresReadBack:        true,
		NABLCompliant:           true,
		DocumentationRequired:   []string{"time_result", "time_notification", "person_notified", "read_back"},
		Governance: NABLGovernance{
			Standard:         "NABL 112:2022",
			Section:          "5.9.1",
			Clause:           "Critical value notification",
			EffectiveDate:    "2022-07-01",
			AuditRequirement: "Monthly audit of notification times",
			ReviewFrequency:  "Annual",
		},
	},

	// Cardiac Markers
	"10839-9": { // Troponin T
		TestCode:                "10839-9",
		TestName:                "Troponin T",
		Unit:                    "ng/mL",
		CriticalLow:             nil,
		CriticalHigh:            floatPtr(0.1), // Varies by assay
		NotificationTimeMinutes: 30,
		NotificationType:        "30_MIN",
		RequiresReadBack:        true,
		NABLCompliant:           true,
		DocumentationRequired:   []string{"time_result", "time_notification", "person_notified", "read_back"},
		Governance: NABLGovernance{
			Standard:         "NABL 112:2022",
			Section:          "5.9.1",
			Clause:           "Critical value notification",
			EffectiveDate:    "2022-07-01",
			AuditRequirement: "Monthly audit of notification times",
			ReviewFrequency:  "Annual",
		},
	},

	// Coagulation
	"5902-2": { // PT/INR
		TestCode:                "5902-2",
		TestName:                "PT/INR",
		Unit:                    "",
		CriticalLow:             nil,
		CriticalHigh:            floatPtr(5.0),
		NotificationTimeMinutes: 30,
		NotificationType:        "30_MIN",
		RequiresReadBack:        true,
		NABLCompliant:           true,
		DocumentationRequired:   []string{"time_result", "time_notification", "person_notified", "read_back"},
		Governance: NABLGovernance{
			Standard:         "NABL 112:2022",
			Section:          "5.9.1",
			Clause:           "Critical value notification",
			EffectiveDate:    "2022-07-01",
			AuditRequirement: "Monthly audit of notification times",
			ReviewFrequency:  "Annual",
		},
	},
	"3173-2": { // aPTT
		TestCode:                "3173-2",
		TestName:                "aPTT",
		Unit:                    "seconds",
		CriticalLow:             nil,
		CriticalHigh:            floatPtr(100.0),
		NotificationTimeMinutes: 30,
		NotificationType:        "30_MIN",
		RequiresReadBack:        true,
		NABLCompliant:           true,
		DocumentationRequired:   []string{"time_result", "time_notification", "person_notified", "read_back"},
		Governance: NABLGovernance{
			Standard:         "NABL 112:2022",
			Section:          "5.9.1",
			Clause:           "Critical value notification",
			EffectiveDate:    "2022-07-01",
			AuditRequirement: "Monthly audit of notification times",
			ReviewFrequency:  "Annual",
		},
	},

	// Microbiology
	"BLOOD-CULTURE": { // Positive Blood Culture
		TestCode:                "BLOOD-CULTURE",
		TestName:                "Blood Culture (Positive)",
		Unit:                    "",
		CriticalLow:             nil,
		CriticalHigh:            nil, // Qualitative - any positive is critical
		NotificationTimeMinutes: 30,
		NotificationType:        "IMMEDIATE",
		RequiresReadBack:        true,
		RepeatCritical:          true, // Always report positive cultures
		NABLCompliant:           true,
		DocumentationRequired:   []string{"time_result", "time_notification", "person_notified", "read_back", "organism"},
		Governance: NABLGovernance{
			Standard:         "NABL 112:2022",
			Section:          "5.9.1",
			Clause:           "Critical value notification",
			EffectiveDate:    "2022-07-01",
			AuditRequirement: "Monthly audit of notification times",
			ReviewFrequency:  "Annual",
		},
	},
	"CSF-CULTURE": { // Positive CSF Culture
		TestCode:                "CSF-CULTURE",
		TestName:                "CSF Culture (Positive)",
		Unit:                    "",
		CriticalLow:             nil,
		CriticalHigh:            nil,
		NotificationTimeMinutes: 15,
		NotificationType:        "IMMEDIATE",
		RequiresReadBack:        true,
		RepeatCritical:          true,
		NABLCompliant:           true,
		DocumentationRequired:   []string{"time_result", "time_notification", "person_notified", "read_back", "organism"},
		Governance: NABLGovernance{
			Standard:         "NABL 112:2022",
			Section:          "5.9.1",
			Clause:           "Critical value notification",
			EffectiveDate:    "2022-07-01",
			AuditRequirement: "Monthly audit of notification times",
			ReviewFrequency:  "Annual",
		},
	},
}

// =============================================================================
// VALIDATION AND COMPLIANCE FUNCTIONS
// =============================================================================

// NABLComplianceResult contains compliance validation results
type NABLComplianceResult struct {
	TestCode             string   `json:"testCode"`
	TestName             string   `json:"testName"`
	IsCriticalValue      bool     `json:"isCriticalValue"`
	CriticalType         string   `json:"criticalType,omitempty"` // HIGH, LOW
	RequiredNotifyTime   int      `json:"requiredNotifyTime"`
	ActualNotifyTime     int      `json:"actualNotifyTime"`
	IsCompliant          bool     `json:"isCompliant"`
	ComplianceIssues     []string `json:"complianceIssues"`
	RequiredDocumentation []string `json:"requiredDocumentation"`
	Recommendations      []string `json:"recommendations"`
}

// ValidateNABLCompliance checks if a critical value notification is NABL compliant
func ValidateNABLCompliance(testCode string, value float64, notificationTimeMin int, readBackCompleted bool) *NABLComplianceResult {
	result := &NABLComplianceResult{
		TestCode:          testCode,
		ActualNotifyTime:  notificationTimeMin,
		ComplianceIssues:  []string{},
		Recommendations:   []string{},
	}

	policy, exists := NABLCriticalValues[testCode]
	if !exists {
		result.IsCriticalValue = false
		result.IsCompliant = true
		result.Recommendations = []string{"Test not on NABL critical value list - standard reporting applies"}
		return result
	}

	result.TestName = policy.TestName
	result.RequiredNotifyTime = policy.NotificationTimeMinutes
	result.RequiredDocumentation = policy.DocumentationRequired

	// Check if value is critical
	isCriticalLow := policy.CriticalLow != nil && value < *policy.CriticalLow
	isCriticalHigh := policy.CriticalHigh != nil && value > *policy.CriticalHigh

	if !isCriticalLow && !isCriticalHigh {
		result.IsCriticalValue = false
		result.IsCompliant = true
		return result
	}

	result.IsCriticalValue = true
	if isCriticalLow {
		result.CriticalType = "LOW"
	} else {
		result.CriticalType = "HIGH"
	}

	// Check compliance
	result.IsCompliant = true

	// Time compliance
	if notificationTimeMin > policy.NotificationTimeMinutes {
		result.IsCompliant = false
		result.ComplianceIssues = append(result.ComplianceIssues,
			"Notification time exceeded NABL requirement",
		)
	}

	// Read-back compliance
	if policy.RequiresReadBack && !readBackCompleted {
		result.IsCompliant = false
		result.ComplianceIssues = append(result.ComplianceIssues,
			"Read-back verification not completed",
		)
	}

	// Generate recommendations
	if !result.IsCompliant {
		result.Recommendations = []string{
			"Review critical value notification workflow",
			"Implement escalation protocol for delayed notifications",
			"Train staff on read-back verification requirements",
			"Document root cause analysis for compliance gaps",
		}
	} else {
		result.Recommendations = []string{
			"Notification compliant with NABL 112:2022",
			"Ensure all documentation is complete",
		}
	}

	return result
}

// IsCriticalValue checks if a value is critical per NABL standards
func IsCriticalValue(testCode string, value float64) (bool, string) {
	policy, exists := NABLCriticalValues[testCode]
	if !exists {
		return false, ""
	}

	if policy.CriticalLow != nil && value < *policy.CriticalLow {
		return true, "LOW"
	}
	if policy.CriticalHigh != nil && value > *policy.CriticalHigh {
		return true, "HIGH"
	}

	return false, ""
}

// GetCriticalValuePolicy returns the NABL policy for a test
func GetCriticalValuePolicy(testCode string) *NABLCriticalValuePolicy {
	policy, exists := NABLCriticalValues[testCode]
	if !exists {
		return nil
	}
	return &policy
}

// =============================================================================
// AUDIT AND REPORTING FUNCTIONS
// =============================================================================

// NABLAuditReport contains audit metrics for critical value notifications
type NABLAuditReport struct {
	ReportPeriodStart   time.Time `json:"reportPeriodStart"`
	ReportPeriodEnd     time.Time `json:"reportPeriodEnd"`
	TotalCriticalValues int       `json:"totalCriticalValues"`
	NotifiedOnTime      int       `json:"notifiedOnTime"`
	NotifiedLate        int       `json:"notifiedLate"`
	ComplianceRate      float64   `json:"complianceRate"`      // Percentage
	AverageNotifyTime   float64   `json:"averageNotifyTime"`   // Minutes
	ReadBackCompliance  float64   `json:"readBackCompliance"`  // Percentage
	ByTest              map[string]NABLTestAudit `json:"byTest"`
	NonCompliantEvents  []string  `json:"nonCompliantEvents"`  // IDs of non-compliant notifications
	Recommendations     []string  `json:"recommendations"`
}

// NABLTestAudit contains audit data for a specific test
type NABLTestAudit struct {
	TestCode         string  `json:"testCode"`
	TestName         string  `json:"testName"`
	TotalCritical    int     `json:"totalCritical"`
	OnTimeCount      int     `json:"onTimeCount"`
	ComplianceRate   float64 `json:"complianceRate"`
	AvgNotifyTime    float64 `json:"avgNotifyTime"`
}

// GenerateAuditReport creates a NABL compliance audit report
func GenerateAuditReport(notifications []NABLCriticalValueNotification, startDate, endDate time.Time) *NABLAuditReport {
	report := &NABLAuditReport{
		ReportPeriodStart: startDate,
		ReportPeriodEnd:   endDate,
		ByTest:            make(map[string]NABLTestAudit),
		NonCompliantEvents: []string{},
		Recommendations:   []string{},
	}

	if len(notifications) == 0 {
		report.ComplianceRate = 100.0
		report.ReadBackCompliance = 100.0
		return report
	}

	totalNotifyTime := 0
	readBackCount := 0

	for _, n := range notifications {
		// Filter by date range
		if n.ResultTime.Before(startDate) || n.ResultTime.After(endDate) {
			continue
		}

		report.TotalCriticalValues++
		totalNotifyTime += n.TimeToNotifyMin

		if n.WithinTimeLimit {
			report.NotifiedOnTime++
		} else {
			report.NotifiedLate++
			report.NonCompliantEvents = append(report.NonCompliantEvents, n.NotificationID)
		}

		if n.ReadBackCompleted {
			readBackCount++
		}

		// Update per-test stats
		testAudit, exists := report.ByTest[n.TestCode]
		if !exists {
			testAudit = NABLTestAudit{
				TestCode: n.TestCode,
				TestName: n.TestName,
			}
		}
		testAudit.TotalCritical++
		if n.WithinTimeLimit {
			testAudit.OnTimeCount++
		}
		testAudit.AvgNotifyTime = (testAudit.AvgNotifyTime*float64(testAudit.TotalCritical-1) + float64(n.TimeToNotifyMin)) / float64(testAudit.TotalCritical)
		report.ByTest[n.TestCode] = testAudit
	}

	// Calculate overall metrics
	if report.TotalCriticalValues > 0 {
		report.ComplianceRate = float64(report.NotifiedOnTime) / float64(report.TotalCriticalValues) * 100
		report.AverageNotifyTime = float64(totalNotifyTime) / float64(report.TotalCriticalValues)
		report.ReadBackCompliance = float64(readBackCount) / float64(report.TotalCriticalValues) * 100
	}

	// Calculate per-test compliance rates
	for code, audit := range report.ByTest {
		audit.ComplianceRate = float64(audit.OnTimeCount) / float64(audit.TotalCritical) * 100
		report.ByTest[code] = audit
	}

	// Generate recommendations
	if report.ComplianceRate < 95 {
		report.Recommendations = append(report.Recommendations,
			"CRITICAL: Compliance rate below 95% - immediate action required",
			"Review notification workflow and staffing",
			"Implement automated alerting systems",
		)
	} else if report.ComplianceRate < 100 {
		report.Recommendations = append(report.Recommendations,
			"Minor compliance gaps identified - review non-compliant events",
		)
	}

	if report.ReadBackCompliance < 100 {
		report.Recommendations = append(report.Recommendations,
			"Read-back verification gaps identified - reinforce training",
		)
	}

	return report
}

// GetAllNABLCriticalTests returns list of all tests with NABL critical values
func GetAllNABLCriticalTests() []string {
	tests := make([]string, 0, len(NABLCriticalValues))
	for code := range NABLCriticalValues {
		tests = append(tests, code)
	}
	return tests
}

// Helper function
func floatPtr(f float64) *float64 {
	return &f
}
