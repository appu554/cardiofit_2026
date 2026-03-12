// Package advisor provides ICU Task Escalation Intelligence.
// This file implements Tier-10 Phase 4: KB-14 ICU Task Integration.
//
// The ICU Task Escalation system provides:
// - ICU-specific mandatory task generation
// - Acuity-based task prioritization
// - Dimension-aware task routing
// - Escalation protocols based on ICU state
// - Integration with KB-14 task management
// - Time-critical task scheduling
package advisor

import (
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// ICU Task Types
// ============================================================================

// ICUTask represents an ICU-specific mandatory task
type ICUTask struct {
	ID               uuid.UUID           `json:"id"`
	TaskType         ICUTaskType         `json:"task_type"`
	Category         ICUTaskCategory     `json:"category"`
	Priority         ICUTaskPriority     `json:"priority"`
	Title            string              `json:"title"`
	Description      string              `json:"description"`
	TriggerDimension string              `json:"trigger_dimension"`
	TriggerEvent     string              `json:"trigger_event"`
	PatientID        uuid.UUID           `json:"patient_id"`
	EncounterID      uuid.UUID           `json:"encounter_id"`
	ICUType          ICUType             `json:"icu_type"`
	AcuityScore      float64             `json:"acuity_score"`

	// Task Timing
	CreatedAt        time.Time           `json:"created_at"`
	DueBy            time.Time           `json:"due_by"`
	EscalatesAt      *time.Time          `json:"escalates_at,omitempty"`
	TimeConstraint   TaskTimeConstraint  `json:"time_constraint"`

	// Assignment
	AssignedTo       []TaskRecipient     `json:"assigned_to"`
	EscalationPath   []EscalationLevel   `json:"escalation_path"`
	CurrentLevel     int                 `json:"current_level"`

	// Clinical Context
	RelatedMedication *ClinicalCode      `json:"related_medication,omitempty"`
	RelatedLabValue   *LabValue          `json:"related_lab_value,omitempty"`
	ClinicalContext   map[string]string  `json:"clinical_context,omitempty"`
	SafetyRationale   string             `json:"safety_rationale"`

	// KB Integration
	KBSource         string              `json:"kb_source"`
	RuleID           string              `json:"rule_id"`
	GovernanceEventID *uuid.UUID         `json:"governance_event_id,omitempty"`

	// Status
	Status           TaskStatus          `json:"status"`
	CompletedAt      *time.Time          `json:"completed_at,omitempty"`
	CompletedBy      *string             `json:"completed_by,omitempty"`
	CompletionNotes  *string             `json:"completion_notes,omitempty"`

	// Audit
	AuditTrail       []TaskAuditEntry    `json:"audit_trail,omitempty"`
}

// ICUTaskType categorizes ICU tasks
type ICUTaskType string

const (
	TaskTypeLabReview        ICUTaskType = "LAB_REVIEW"
	TaskTypeMedReview        ICUTaskType = "MEDICATION_REVIEW"
	TaskTypeDoseAdjustment   ICUTaskType = "DOSE_ADJUSTMENT"
	TaskTypePharmacyConsult  ICUTaskType = "PHARMACY_CONSULT"
	TaskTypeNephrologyConsult ICUTaskType = "NEPHROLOGY_CONSULT"
	TaskTypeICUTeamReview    ICUTaskType = "ICU_TEAM_REVIEW"
	TaskTypeGoalsOfCare      ICUTaskType = "GOALS_OF_CARE"
	TaskTypeMonitoring       ICUTaskType = "ENHANCED_MONITORING"
	TaskTypeIntervention     ICUTaskType = "CLINICAL_INTERVENTION"
	TaskTypeCRRTAdjustment   ICUTaskType = "CRRT_ADJUSTMENT"
	TaskTypeVentAdjustment   ICUTaskType = "VENT_ADJUSTMENT"
	TaskTypeSepsisBundle     ICUTaskType = "SEPSIS_BUNDLE"
	TaskTypeAntibioticReview ICUTaskType = "ANTIBIOTIC_STEWARDSHIP"
	TaskTypeNeurologyConsult ICUTaskType = "NEUROLOGY_CONSULT"
	TaskTypeCardiologyConsult ICUTaskType = "CARDIOLOGY_CONSULT"
)

// ICUTaskCategory groups related task types
type ICUTaskCategory string

const (
	CategoryMedicationSafety  ICUTaskCategory = "MEDICATION_SAFETY"
	CategoryOrganSupport      ICUTaskCategory = "ORGAN_SUPPORT"
	CategoryInfectionControl  ICUTaskCategory = "INFECTION_CONTROL"
	CategoryHemodynamic       ICUTaskCategory = "HEMODYNAMIC"
	CategoryRespiratory       ICUTaskCategory = "RESPIRATORY"
	CategoryRenal             ICUTaskCategory = "RENAL"
	CategoryNeurological      ICUTaskCategory = "NEUROLOGICAL"
	CategoryGovernance        ICUTaskCategory = "GOVERNANCE"
	CategoryEscalation        ICUTaskCategory = "ESCALATION"
)

// ICUTaskPriority determines task urgency
type ICUTaskPriority string

const (
	PriorityStat       ICUTaskPriority = "STAT"      // Immediate action required
	PriorityUrgent     ICUTaskPriority = "URGENT"    // Within 1 hour
	PriorityHigh       ICUTaskPriority = "HIGH"      // Within 4 hours
	PriorityMedium     ICUTaskPriority = "MEDIUM"    // Within 8 hours
	PriorityRoutine    ICUTaskPriority = "ROUTINE"   // Within 24 hours
	PriorityScheduled  ICUTaskPriority = "SCHEDULED" // At specified time
)

// TaskTimeConstraint defines timing requirements
type TaskTimeConstraint struct {
	MaxResponseMinutes   int    `json:"max_response_minutes"`
	MaxCompletionMinutes int    `json:"max_completion_minutes"`
	EscalationMinutes    int    `json:"escalation_minutes"`
	Rationale            string `json:"rationale"`
}

// TaskRecipient represents a task assignee
type TaskRecipient struct {
	RecipientType string `json:"recipient_type"` // provider, team, role
	RecipientID   string `json:"recipient_id"`
	RecipientName string `json:"recipient_name"`
	Role          string `json:"role"`
}

// EscalationLevel defines escalation hierarchy
type EscalationLevel struct {
	Level         int             `json:"level"`
	Recipients    []TaskRecipient `json:"recipients"`
	TimeToEscalate time.Duration  `json:"time_to_escalate"`
	Notification  string          `json:"notification"` // page, secure_message, phone
}

// TaskStatus represents task lifecycle
type TaskStatus string

const (
	StatusPending     TaskStatus = "PENDING"
	StatusAssigned    TaskStatus = "ASSIGNED"
	StatusInProgress  TaskStatus = "IN_PROGRESS"
	StatusCompleted   TaskStatus = "COMPLETED"
	StatusEscalated   TaskStatus = "ESCALATED"
	StatusOverdue     TaskStatus = "OVERDUE"
	StatusCancelled   TaskStatus = "CANCELLED"
)

// TaskAuditEntry tracks task history
type TaskAuditEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
	Actor     string    `json:"actor"`
	Details   string    `json:"details"`
}

// ============================================================================
// ICU Task Generator Engine
// ============================================================================

// ICUTaskGenerator creates ICU-specific tasks based on clinical state
type ICUTaskGenerator struct {
	acuityWeights     map[string]float64
	priorityThresholds map[ICUTaskPriority]float64
	escalationPolicies map[ICUTaskType]EscalationPolicy
}

// EscalationPolicy defines how tasks escalate
type EscalationPolicy struct {
	InitialResponseMinutes int
	EscalationIntervals    []int // Minutes between escalations
	MaxEscalationLevel     int
	AutoComplete           bool
}

// NewICUTaskGenerator creates a new task generator
func NewICUTaskGenerator() *ICUTaskGenerator {
	return &ICUTaskGenerator{
		acuityWeights: map[string]float64{
			"hemodynamic":   0.25,
			"respiratory":   0.20,
			"renal":         0.15,
			"neurological":  0.15,
			"infection":     0.10,
			"coagulation":   0.10,
			"hepatic":       0.05,
		},
		priorityThresholds: map[ICUTaskPriority]float64{
			PriorityStat:     90.0, // Acuity >= 90
			PriorityUrgent:   75.0, // Acuity >= 75
			PriorityHigh:     60.0, // Acuity >= 60
			PriorityMedium:   40.0, // Acuity >= 40
			PriorityRoutine:  0.0,  // Default
		},
		escalationPolicies: loadEscalationPolicies(),
	}
}

// GenerateTasksFromViolations creates tasks from ICU rule violations
func (g *ICUTaskGenerator) GenerateTasksFromViolations(
	violations []ICURuleViolation,
	icuState *ICUClinicalState,
	governanceEvents []GovernanceEvent,
) []ICUTask {
	tasks := []ICUTask{}

	for _, v := range violations {
		// Determine task type based on violation
		taskType := g.violationToTaskType(v)
		category := g.ruleCategoriesToTaskCategory(v.Category)
		priority := g.calculatePriority(icuState.ICUAcuityScore, v.Severity)

		// Create time constraint based on priority
		timeConstraint := g.getTimeConstraint(priority, v.Category)

		// Build escalation path
		escalationPath := g.buildEscalationPath(taskType, icuState.ICUType)

		// Find related governance event
		var govEventID *uuid.UUID
		for _, ge := range governanceEvents {
			if ge.RuleID == v.RuleID {
				govEventID = &ge.ID
				break
			}
		}

		task := ICUTask{
			ID:                uuid.New(),
			TaskType:          taskType,
			Category:          category,
			Priority:          priority,
			Title:             g.generateTaskTitle(v, taskType),
			Description:       g.generateTaskDescription(v, icuState),
			TriggerDimension:  v.Dimension,
			TriggerEvent:      v.RuleName,
			PatientID:         icuState.PatientID,
			EncounterID:       icuState.EncounterID,
			ICUType:           icuState.ICUType,
			AcuityScore:       icuState.ICUAcuityScore,
			CreatedAt:         time.Now(),
			DueBy:             time.Now().Add(time.Duration(timeConstraint.MaxCompletionMinutes) * time.Minute),
			TimeConstraint:    timeConstraint,
			AssignedTo:        g.getInitialAssignees(taskType, icuState.ICUType),
			EscalationPath:    escalationPath,
			CurrentLevel:      0,
			RelatedMedication: &v.Medication,
			ClinicalContext: map[string]string{
				"trigger_value": v.TriggerValue,
				"threshold":     v.Threshold,
				"dimension":     v.Dimension,
			},
			SafetyRationale:   v.Recommendation,
			KBSource:          v.KBSource,
			RuleID:            v.RuleID,
			GovernanceEventID: govEventID,
			Status:            StatusPending,
			AuditTrail: []TaskAuditEntry{
				{
					Timestamp: time.Now(),
					Action:    "CREATED",
					Actor:     "SYSTEM",
					Details:   fmt.Sprintf("Task generated from rule violation: %s", v.RuleName),
				},
			},
		}

		// Set escalation time
		escalationTime := time.Now().Add(time.Duration(timeConstraint.EscalationMinutes) * time.Minute)
		task.EscalatesAt = &escalationTime

		tasks = append(tasks, task)
	}

	// Sort by priority
	sort.Slice(tasks, func(i, j int) bool {
		return g.priorityToInt(tasks[i].Priority) > g.priorityToInt(tasks[j].Priority)
	})

	return tasks
}

// GenerateTasksFromAlerts creates tasks from temporal alerts
func (g *ICUTaskGenerator) GenerateTasksFromAlerts(
	alerts []TemporalAlert,
	icuState *ICUClinicalState,
) []ICUTask {
	tasks := []ICUTask{}

	for _, alert := range alerts {
		// Only generate tasks for urgent/critical alerts
		if alert.Severity != AlertUrgent && alert.Severity != AlertCritical {
			continue
		}

		taskType := g.alertToTaskType(alert)
		priority := g.alertSeverityToPriority(alert.Severity)
		category := g.alertDimensionToCategory(alert.Dimension)
		timeConstraint := g.getTimeConstraint(priority, ICURuleCategory(category))

		task := ICUTask{
			ID:               uuid.New(),
			TaskType:         taskType,
			Category:         category,
			Priority:         priority,
			Title:            alert.Title,
			Description:      fmt.Sprintf("%s\n\nRecommendation: %s", alert.Description, alert.Recommendation),
			TriggerDimension: alert.Dimension,
			TriggerEvent:     string(alert.AlertType),
			PatientID:        icuState.PatientID,
			EncounterID:      icuState.EncounterID,
			ICUType:          icuState.ICUType,
			AcuityScore:      icuState.ICUAcuityScore,
			CreatedAt:        time.Now(),
			DueBy:            time.Now().Add(time.Duration(timeConstraint.MaxCompletionMinutes) * time.Minute),
			TimeConstraint:   timeConstraint,
			AssignedTo:       g.getInitialAssignees(taskType, icuState.ICUType),
			EscalationPath:   g.buildEscalationPath(taskType, icuState.ICUType),
			CurrentLevel:     0,
			ClinicalContext: map[string]string{
				"current_value": fmt.Sprintf("%.2f", alert.CurrentValue),
				"trend":         string(alert.TriggerTrend),
			},
			SafetyRationale: alert.Recommendation,
			KBSource:        "KB-TEMPORAL",
			RuleID:          string(alert.AlertType),
			Status:          StatusPending,
			AuditTrail: []TaskAuditEntry{
				{
					Timestamp: time.Now(),
					Action:    "CREATED",
					Actor:     "SYSTEM",
					Details:   fmt.Sprintf("Task generated from temporal alert: %s", alert.AlertType),
				},
			},
		}

		tasks = append(tasks, task)
	}

	return tasks
}

// GenerateTasksFromPredictions creates proactive tasks from deterioration predictions
func (g *ICUTaskGenerator) GenerateTasksFromPredictions(
	predictions []DeteriorationPrediction,
	icuState *ICUClinicalState,
) []ICUTask {
	tasks := []ICUTask{}

	for _, pred := range predictions {
		// Only generate tasks for high-probability predictions
		if pred.Probability < 0.5 {
			continue
		}

		// Priority based on probability and time horizon
		priority := PriorityMedium
		if pred.Probability >= 0.75 && pred.TimeHorizon < 4*time.Hour {
			priority = PriorityUrgent
		} else if pred.Probability >= 0.6 {
			priority = PriorityHigh
		}

		taskType := g.predictionToTaskType(pred)
		category := g.alertDimensionToCategory(pred.Dimension)

		task := ICUTask{
			ID:               uuid.New(),
			TaskType:         taskType,
			Category:         category,
			Priority:         priority,
			Title:            fmt.Sprintf("PROACTIVE: Predicted %s", pred.PredictedEvent),
			Description:      g.buildPredictionDescription(pred),
			TriggerDimension: pred.Dimension,
			TriggerEvent:     "DETERIORATION_PREDICTION",
			PatientID:        icuState.PatientID,
			EncounterID:      icuState.EncounterID,
			ICUType:          icuState.ICUType,
			AcuityScore:      icuState.ICUAcuityScore,
			CreatedAt:        time.Now(),
			DueBy:            time.Now().Add(pred.TimeHorizon / 2), // Due before predicted event
			TimeConstraint: TaskTimeConstraint{
				MaxResponseMinutes:   30,
				MaxCompletionMinutes: int(pred.TimeHorizon.Minutes() / 2),
				EscalationMinutes:    int(pred.TimeHorizon.Minutes() / 4),
				Rationale:            "Proactive intervention before predicted deterioration",
			},
			AssignedTo:     g.getInitialAssignees(taskType, icuState.ICUType),
			EscalationPath: g.buildEscalationPath(taskType, icuState.ICUType),
			CurrentLevel:   0,
			ClinicalContext: map[string]string{
				"probability":    fmt.Sprintf("%.0f%%", pred.Probability*100),
				"time_horizon":   pred.TimeHorizon.String(),
				"trajectory":     pred.CurrentTrajectory,
			},
			SafetyRationale: fmt.Sprintf("Predicted event: %s. Preventive action recommended.", pred.PredictedEvent),
			KBSource:        "KB-PREDICTIVE",
			RuleID:          fmt.Sprintf("PREDICT-%s", pred.Dimension),
			Status:          StatusPending,
			AuditTrail: []TaskAuditEntry{
				{
					Timestamp: time.Now(),
					Action:    "CREATED",
					Actor:     "SYSTEM",
					Details:   fmt.Sprintf("Proactive task from prediction: %.0f%% probability", pred.Probability*100),
				},
			},
		}

		tasks = append(tasks, task)
	}

	return tasks
}

// GenerateCriticalTasks creates STAT tasks for critical ICU situations
func (g *ICUTaskGenerator) GenerateCriticalTasks(icuState *ICUClinicalState) []ICUTask {
	tasks := []ICUTask{}

	// STAT: Septic shock + no vasopressor increase
	if icuState.Infection.SepticShock && icuState.Hemodynamic.VasopressorReq != VasopressorMaximal {
		tasks = append(tasks, ICUTask{
			ID:               uuid.New(),
			TaskType:         TaskTypeSepsisBundle,
			Category:         CategoryInfectionControl,
			Priority:         PriorityStat,
			Title:            "STAT: Septic Shock - SSC Bundle Required",
			Description:      "Patient in septic shock. Ensure Hour-1 and Hour-3 Surviving Sepsis bundles complete.",
			TriggerDimension: "infection",
			TriggerEvent:     "SEPTIC_SHOCK",
			PatientID:        icuState.PatientID,
			EncounterID:      icuState.EncounterID,
			ICUType:          icuState.ICUType,
			AcuityScore:      icuState.ICUAcuityScore,
			CreatedAt:        time.Now(),
			DueBy:            time.Now().Add(60 * time.Minute),
			TimeConstraint: TaskTimeConstraint{
				MaxResponseMinutes:   5,
				MaxCompletionMinutes: 60,
				EscalationMinutes:    15,
				Rationale:            "Surviving Sepsis Campaign Hour-1 Bundle",
			},
			AssignedTo: []TaskRecipient{
				{RecipientType: "team", RecipientID: "ICU_TEAM", RecipientName: "ICU Team", Role: "Primary"},
			},
			EscalationPath: []EscalationLevel{
				{Level: 1, Recipients: []TaskRecipient{{RecipientType: "role", RecipientID: "INTENSIVIST", RecipientName: "On-Call Intensivist", Role: "Attending"}}, TimeToEscalate: 15 * time.Minute, Notification: "page"},
				{Level: 2, Recipients: []TaskRecipient{{RecipientType: "role", RecipientID: "ICU_DIRECTOR", RecipientName: "ICU Director", Role: "Director"}}, TimeToEscalate: 30 * time.Minute, Notification: "page"},
			},
			ClinicalContext: map[string]string{
				"sepsis_status":    string(icuState.Infection.SepsisStatus),
				"vasopressor_req":  string(icuState.Hemodynamic.VasopressorReq),
				"map":              fmt.Sprintf("%.0f mmHg", icuState.Hemodynamic.MAP),
			},
			SafetyRationale: "Septic shock is a medical emergency. SSC bundles reduce mortality.",
			KBSource:        "KB-3",
			RuleID:          "SSC-BUNDLE-STAT",
			Status:          StatusPending,
		})
	}

	// STAT: GCS drop to ≤ 8 without secured airway
	if icuState.Neurological.GCS <= 8 && icuState.Respiratory.AirwayType == AirwayNatural {
		tasks = append(tasks, ICUTask{
			ID:               uuid.New(),
			TaskType:         TaskTypeIntervention,
			Category:         CategoryNeurological,
			Priority:         PriorityStat,
			Title:            "STAT: GCS ≤ 8 - Airway Assessment Required",
			Description:      "GCS has fallen to ≤ 8 without secured airway. Immediate airway assessment and protection required.",
			TriggerDimension: "neurological",
			TriggerEvent:     "GCS_CRITICAL",
			PatientID:        icuState.PatientID,
			EncounterID:      icuState.EncounterID,
			ICUType:          icuState.ICUType,
			AcuityScore:      icuState.ICUAcuityScore,
			CreatedAt:        time.Now(),
			DueBy:            time.Now().Add(15 * time.Minute),
			TimeConstraint: TaskTimeConstraint{
				MaxResponseMinutes:   2,
				MaxCompletionMinutes: 15,
				EscalationMinutes:    5,
				Rationale:            "Unprotected airway with GCS ≤ 8 - aspiration risk",
			},
			AssignedTo: []TaskRecipient{
				{RecipientType: "team", RecipientID: "RAPID_RESPONSE", RecipientName: "Rapid Response Team", Role: "Primary"},
			},
			ClinicalContext: map[string]string{
				"gcs":         fmt.Sprintf("%d", icuState.Neurological.GCS),
				"airway_type": string(icuState.Respiratory.AirwayType),
			},
			SafetyRationale: "Patients with GCS ≤ 8 cannot protect their airway. Intubation likely required.",
			KBSource:        "KB-4",
			RuleID:          "GCS-AIRWAY-STAT",
			Status:          StatusPending,
		})
	}

	// STAT: Critical hypotension (MAP < 55) on maximal support
	if icuState.Hemodynamic.MAP < 55 && icuState.Hemodynamic.VasopressorReq == VasopressorMaximal {
		tasks = append(tasks, ICUTask{
			ID:               uuid.New(),
			TaskType:         TaskTypeGoalsOfCare,
			Category:         CategoryHemodynamic,
			Priority:         PriorityStat,
			Title:            "STAT: Refractory Hypotension - Goals Discussion",
			Description:      "MAP < 55 mmHg on maximal vasopressor support. Consider mechanical support or goals of care discussion.",
			TriggerDimension: "hemodynamic",
			TriggerEvent:     "REFRACTORY_SHOCK",
			PatientID:        icuState.PatientID,
			EncounterID:      icuState.EncounterID,
			ICUType:          icuState.ICUType,
			AcuityScore:      100, // Maximum acuity
			CreatedAt:        time.Now(),
			DueBy:            time.Now().Add(30 * time.Minute),
			TimeConstraint: TaskTimeConstraint{
				MaxResponseMinutes:   5,
				MaxCompletionMinutes: 30,
				EscalationMinutes:    10,
				Rationale:            "Refractory shock with poor prognosis",
			},
			AssignedTo: []TaskRecipient{
				{RecipientType: "role", RecipientID: "INTENSIVIST", RecipientName: "Attending Intensivist", Role: "Attending"},
			},
			ClinicalContext: map[string]string{
				"map":             fmt.Sprintf("%.0f mmHg", icuState.Hemodynamic.MAP),
				"vasopressor_req": string(icuState.Hemodynamic.VasopressorReq),
				"shock_state":     string(icuState.Hemodynamic.ShockState),
			},
			SafetyRationale: "Refractory shock despite maximal vasopressor therapy. Consider ECMO if appropriate or palliative care.",
			KBSource:        "KB-4",
			RuleID:          "REFRACTORY-SHOCK-GOC",
			Status:          StatusPending,
		})
	}

	return tasks
}

// ============================================================================
// Helper Methods
// ============================================================================

func (g *ICUTaskGenerator) violationToTaskType(v ICURuleViolation) ICUTaskType {
	switch v.Category {
	case RuleCategoryRenal:
		if v.Severity == SeverityCritical || v.Severity == SeverityBlock {
			return TaskTypeNephrologyConsult
		}
		return TaskTypeDoseAdjustment
	case RuleCategoryCRRT:
		return TaskTypeCRRTAdjustment
	case RuleCategoryRespiratory:
		return TaskTypeVentAdjustment
	case RuleCategoryInfection:
		return TaskTypeAntibioticReview
	case RuleCategoryNeurological:
		if v.Severity == SeverityCritical {
			return TaskTypeNeurologyConsult
		}
		return TaskTypeMedReview
	case RuleCategoryHemodynamic:
		if v.Severity == SeverityCritical {
			return TaskTypeCardiologyConsult
		}
		return TaskTypeMedReview
	case RuleCategoryCoagulation:
		return TaskTypeLabReview
	default:
		if v.Severity == SeverityCritical || v.Severity == SeverityBlock {
			return TaskTypePharmacyConsult
		}
		return TaskTypeMedReview
	}
}

func (g *ICUTaskGenerator) ruleCategoriesToTaskCategory(rc ICURuleCategory) ICUTaskCategory {
	switch rc {
	case RuleCategoryHemodynamic:
		return CategoryHemodynamic
	case RuleCategoryRespiratory:
		return CategoryRespiratory
	case RuleCategoryRenal, RuleCategoryCRRT:
		return CategoryRenal
	case RuleCategoryNeurological:
		return CategoryNeurological
	case RuleCategoryInfection:
		return CategoryInfectionControl
	case RuleCategoryCoagulation:
		return CategoryMedicationSafety
	default:
		return CategoryMedicationSafety
	}
}

func (g *ICUTaskGenerator) calculatePriority(acuityScore float64, severity RuleSeverity) ICUTaskPriority {
	// Severity-based baseline
	var basePriority ICUTaskPriority
	switch severity {
	case SeverityCritical:
		basePriority = PriorityStat
	case SeverityBlock:
		basePriority = PriorityUrgent
	case SeverityWarning:
		basePriority = PriorityHigh
	case SeverityCaution:
		basePriority = PriorityMedium
	default:
		basePriority = PriorityRoutine
	}

	// Upgrade based on acuity
	if acuityScore >= 80 && basePriority != PriorityStat {
		return g.upgradePriority(basePriority)
	}
	if acuityScore >= 60 && (basePriority == PriorityMedium || basePriority == PriorityRoutine) {
		return g.upgradePriority(basePriority)
	}

	return basePriority
}

func (g *ICUTaskGenerator) upgradePriority(p ICUTaskPriority) ICUTaskPriority {
	switch p {
	case PriorityRoutine:
		return PriorityMedium
	case PriorityMedium:
		return PriorityHigh
	case PriorityHigh:
		return PriorityUrgent
	case PriorityUrgent:
		return PriorityStat
	default:
		return p
	}
}

func (g *ICUTaskGenerator) priorityToInt(p ICUTaskPriority) int {
	switch p {
	case PriorityStat:
		return 5
	case PriorityUrgent:
		return 4
	case PriorityHigh:
		return 3
	case PriorityMedium:
		return 2
	case PriorityRoutine:
		return 1
	default:
		return 0
	}
}

func (g *ICUTaskGenerator) getTimeConstraint(priority ICUTaskPriority, category ICURuleCategory) TaskTimeConstraint {
	switch priority {
	case PriorityStat:
		return TaskTimeConstraint{
			MaxResponseMinutes:   5,
			MaxCompletionMinutes: 30,
			EscalationMinutes:    10,
			Rationale:            "STAT priority requires immediate response",
		}
	case PriorityUrgent:
		return TaskTimeConstraint{
			MaxResponseMinutes:   15,
			MaxCompletionMinutes: 60,
			EscalationMinutes:    30,
			Rationale:            "Urgent clinical safety issue",
		}
	case PriorityHigh:
		return TaskTimeConstraint{
			MaxResponseMinutes:   30,
			MaxCompletionMinutes: 240, // 4 hours
			EscalationMinutes:    60,
			Rationale:            "High priority medication safety concern",
		}
	case PriorityMedium:
		return TaskTimeConstraint{
			MaxResponseMinutes:   60,
			MaxCompletionMinutes: 480, // 8 hours
			EscalationMinutes:    120,
			Rationale:            "Moderate priority - complete within shift",
		}
	default:
		return TaskTimeConstraint{
			MaxResponseMinutes:   120,
			MaxCompletionMinutes: 1440, // 24 hours
			EscalationMinutes:    480,
			Rationale:            "Routine priority - complete within 24 hours",
		}
	}
}

func (g *ICUTaskGenerator) getInitialAssignees(taskType ICUTaskType, icuType ICUType) []TaskRecipient {
	switch taskType {
	case TaskTypePharmacyConsult:
		return []TaskRecipient{
			{RecipientType: "role", RecipientID: "ICU_PHARMACIST", RecipientName: "ICU Clinical Pharmacist", Role: "Primary"},
		}
	case TaskTypeNephrologyConsult:
		return []TaskRecipient{
			{RecipientType: "role", RecipientID: "NEPHROLOGY_CONSULT", RecipientName: "Nephrology Consult Service", Role: "Consulting"},
		}
	case TaskTypeNeurologyConsult:
		return []TaskRecipient{
			{RecipientType: "role", RecipientID: "NEUROLOGY_CONSULT", RecipientName: "Neurology Consult Service", Role: "Consulting"},
		}
	case TaskTypeCardiologyConsult:
		return []TaskRecipient{
			{RecipientType: "role", RecipientID: "CARDIOLOGY_CONSULT", RecipientName: "Cardiology Consult Service", Role: "Consulting"},
		}
	case TaskTypeSepsisBundle:
		return []TaskRecipient{
			{RecipientType: "team", RecipientID: "ICU_TEAM", RecipientName: "ICU Team", Role: "Primary"},
			{RecipientType: "role", RecipientID: "ICU_RN", RecipientName: "Bedside RN", Role: "Executing"},
		}
	case TaskTypeGoalsOfCare:
		return []TaskRecipient{
			{RecipientType: "role", RecipientID: "INTENSIVIST", RecipientName: "Attending Intensivist", Role: "Primary"},
		}
	default:
		// Default to ICU team based on ICU type
		return []TaskRecipient{
			{RecipientType: "team", RecipientID: fmt.Sprintf("%s_TEAM", icuType), RecipientName: fmt.Sprintf("%s Team", icuType), Role: "Primary"},
		}
	}
}

func (g *ICUTaskGenerator) buildEscalationPath(taskType ICUTaskType, icuType ICUType) []EscalationLevel {
	// Standard escalation path
	return []EscalationLevel{
		{
			Level: 1,
			Recipients: []TaskRecipient{
				{RecipientType: "role", RecipientID: "ICU_FELLOW", RecipientName: "ICU Fellow", Role: "Fellow"},
			},
			TimeToEscalate: 30 * time.Minute,
			Notification:   "secure_message",
		},
		{
			Level: 2,
			Recipients: []TaskRecipient{
				{RecipientType: "role", RecipientID: "INTENSIVIST", RecipientName: "Attending Intensivist", Role: "Attending"},
			},
			TimeToEscalate: 60 * time.Minute,
			Notification:   "page",
		},
		{
			Level: 3,
			Recipients: []TaskRecipient{
				{RecipientType: "role", RecipientID: "ICU_DIRECTOR", RecipientName: "ICU Medical Director", Role: "Director"},
			},
			TimeToEscalate: 120 * time.Minute,
			Notification:   "page",
		},
	}
}

func (g *ICUTaskGenerator) generateTaskTitle(v ICURuleViolation, taskType ICUTaskType) string {
	severityPrefix := ""
	switch v.Severity {
	case SeverityCritical:
		severityPrefix = "CRITICAL: "
	case SeverityBlock:
		severityPrefix = "BLOCKED: "
	case SeverityWarning:
		severityPrefix = "WARNING: "
	}

	return fmt.Sprintf("%s%s - %s", severityPrefix, v.Medication.Display, v.RuleName)
}

func (g *ICUTaskGenerator) generateTaskDescription(v ICURuleViolation, state *ICUClinicalState) string {
	return fmt.Sprintf(`Medication: %s (%s)
Rule Triggered: %s
Dimension: %s
Current Value: %s | Threshold: %s

Clinical Context:
- ICU Acuity Score: %.0f/100
- Trend: %s

Safety Concern:
%s

Required Action:
%s`,
		v.Medication.Display, v.Medication.Code,
		v.RuleName,
		v.Dimension,
		v.TriggerValue, v.Threshold,
		state.ICUAcuityScore,
		state.TrendDirection,
		v.Recommendation,
		g.getActionDescription(v.Action))
}

func (g *ICUTaskGenerator) getActionDescription(action ICURuleAction) string {
	switch action.ActionType {
	case "block":
		if action.BlockLevel == "hard" {
			return "Hard block - medication cannot be administered without override"
		}
		return "Soft block - requires acknowledgment before administration"
	case "warn":
		return "Review and acknowledge warning before proceeding"
	case "adjust":
		if action.DoseAdjustment != nil {
			return fmt.Sprintf("Adjust dose to %.0f%% of standard", *action.DoseAdjustment)
		}
		return "Dose adjustment required"
	case "monitor":
		if len(action.MonitoringReqs) > 0 {
			return fmt.Sprintf("Enhanced monitoring required: %v", action.MonitoringReqs)
		}
		return "Enhanced monitoring required"
	default:
		return "Review and take appropriate action"
	}
}

func (g *ICUTaskGenerator) alertToTaskType(alert TemporalAlert) ICUTaskType {
	switch alert.Dimension {
	case "hemodynamic":
		return TaskTypeIntervention
	case "respiratory":
		return TaskTypeVentAdjustment
	case "renal":
		return TaskTypeNephrologyConsult
	case "neurological":
		return TaskTypeNeurologyConsult
	case "infection":
		return TaskTypeAntibioticReview
	default:
		return TaskTypeICUTeamReview
	}
}

func (g *ICUTaskGenerator) alertSeverityToPriority(severity AlertSeverity) ICUTaskPriority {
	switch severity {
	case AlertCritical:
		return PriorityStat
	case AlertUrgent:
		return PriorityUrgent
	case AlertWarning:
		return PriorityHigh
	default:
		return PriorityMedium
	}
}

func (g *ICUTaskGenerator) alertDimensionToCategory(dimension string) ICUTaskCategory {
	switch dimension {
	case "hemodynamic":
		return CategoryHemodynamic
	case "respiratory":
		return CategoryRespiratory
	case "renal":
		return CategoryRenal
	case "neurological":
		return CategoryNeurological
	case "infection":
		return CategoryInfectionControl
	case "composite":
		return CategoryEscalation
	default:
		return CategoryMedicationSafety
	}
}

func (g *ICUTaskGenerator) predictionToTaskType(pred DeteriorationPrediction) ICUTaskType {
	switch pred.Dimension {
	case "hemodynamic":
		return TaskTypeIntervention
	case "respiratory":
		return TaskTypeVentAdjustment
	case "renal":
		return TaskTypeCRRTAdjustment
	default:
		return TaskTypeICUTeamReview
	}
}

func (g *ICUTaskGenerator) buildPredictionDescription(pred DeteriorationPrediction) string {
	desc := fmt.Sprintf(`Predicted Event: %s
Probability: %.0f%%
Time Horizon: %s
Current Trajectory: %s

Risk Factors:
`,
		pred.PredictedEvent,
		pred.Probability*100,
		pred.TimeHorizon.String(),
		pred.CurrentTrajectory)

	for _, rf := range pred.RiskFactors {
		desc += fmt.Sprintf("• %s\n", rf)
	}

	if len(pred.PreventiveActions) > 0 {
		desc += "\nPreventive Actions:\n"
		for _, pa := range pred.PreventiveActions {
			desc += fmt.Sprintf("• %s\n", pa)
		}
	}

	return desc
}

func loadEscalationPolicies() map[ICUTaskType]EscalationPolicy {
	return map[ICUTaskType]EscalationPolicy{
		TaskTypeSepsisBundle: {
			InitialResponseMinutes: 5,
			EscalationIntervals:    []int{15, 30, 60},
			MaxEscalationLevel:     3,
			AutoComplete:           false,
		},
		TaskTypeGoalsOfCare: {
			InitialResponseMinutes: 10,
			EscalationIntervals:    []int{30, 60},
			MaxEscalationLevel:     2,
			AutoComplete:           false,
		},
		TaskTypePharmacyConsult: {
			InitialResponseMinutes: 30,
			EscalationIntervals:    []int{60, 120},
			MaxEscalationLevel:     2,
			AutoComplete:           false,
		},
		TaskTypeMedReview: {
			InitialResponseMinutes: 60,
			EscalationIntervals:    []int{120, 240},
			MaxEscalationLevel:     2,
			AutoComplete:           true, // Auto-complete after review
		},
	}
}

// ============================================================================
// ICU Task Coordinator - Orchestrates all task generation
// ============================================================================

// ICUTaskCoordinator orchestrates task generation from all sources
type ICUTaskCoordinator struct {
	taskGenerator    *ICUTaskGenerator
	rulesEngine      *ICUSafetyRulesEngine
	temporalEngine   *ICUTemporalEngine
}

// NewICUTaskCoordinator creates a fully-wired task coordinator
func NewICUTaskCoordinator() *ICUTaskCoordinator {
	return &ICUTaskCoordinator{
		taskGenerator:   NewICUTaskGenerator(),
		rulesEngine:     NewICUSafetyRulesEngine(),
		temporalEngine:  NewICUTemporalEngine(),
	}
}

// GenerateAllTasks creates comprehensive task list for an ICU patient
func (c *ICUTaskCoordinator) GenerateAllTasks(
	icuState *ICUClinicalState,
	temporal *ICUTemporalState,
	proposedMeds []ClinicalCode,
	governanceEvents []GovernanceEvent,
) *ICUTaskBundle {
	bundle := &ICUTaskBundle{
		ID:          uuid.New(),
		PatientID:   icuState.PatientID,
		EncounterID: icuState.EncounterID,
		GeneratedAt: time.Now(),
		Tasks:       []ICUTask{},
	}

	// 1. Generate critical tasks first (STAT)
	criticalTasks := c.taskGenerator.GenerateCriticalTasks(icuState)
	bundle.Tasks = append(bundle.Tasks, criticalTasks...)
	bundle.StatTaskCount = len(criticalTasks)

	// 2. Evaluate medications against ICU rules
	if len(proposedMeds) > 0 {
		evaluation := c.rulesEngine.EvaluateMultipleMedications(proposedMeds, icuState)
		violationTasks := c.taskGenerator.GenerateTasksFromViolations(
			evaluation.Violations,
			icuState,
			governanceEvents,
		)
		bundle.Tasks = append(bundle.Tasks, violationTasks...)
		bundle.ViolationTaskCount = len(violationTasks)
	}

	// 3. Generate tasks from temporal alerts
	if temporal != nil {
		temporalResult := c.temporalEngine.AnalyzeTemporalState(temporal)
		if temporalResult.Sufficient {
			alertTasks := c.taskGenerator.GenerateTasksFromAlerts(temporalResult.Alerts, icuState)
			bundle.Tasks = append(bundle.Tasks, alertTasks...)
			bundle.AlertTaskCount = len(alertTasks)

			// 4. Generate proactive tasks from predictions
			predictionTasks := c.taskGenerator.GenerateTasksFromPredictions(temporalResult.Predictions, icuState)
			bundle.Tasks = append(bundle.Tasks, predictionTasks...)
			bundle.PredictiveTaskCount = len(predictionTasks)
		}
	}

	// Sort all tasks by priority
	sort.Slice(bundle.Tasks, func(i, j int) bool {
		pi := c.taskGenerator.priorityToInt(bundle.Tasks[i].Priority)
		pj := c.taskGenerator.priorityToInt(bundle.Tasks[j].Priority)
		if pi != pj {
			return pi > pj
		}
		return bundle.Tasks[i].DueBy.Before(bundle.Tasks[j].DueBy)
	})

	// Calculate bundle metrics
	bundle.TotalTasks = len(bundle.Tasks)
	bundle.CriticalCount = c.countByPriority(bundle.Tasks, PriorityStat)
	bundle.UrgentCount = c.countByPriority(bundle.Tasks, PriorityUrgent)

	return bundle
}

func (c *ICUTaskCoordinator) countByPriority(tasks []ICUTask, priority ICUTaskPriority) int {
	count := 0
	for _, t := range tasks {
		if t.Priority == priority {
			count++
		}
	}
	return count
}

// ICUTaskBundle contains all generated tasks for a patient
type ICUTaskBundle struct {
	ID                  uuid.UUID  `json:"id"`
	PatientID           uuid.UUID  `json:"patient_id"`
	EncounterID         uuid.UUID  `json:"encounter_id"`
	GeneratedAt         time.Time  `json:"generated_at"`
	Tasks               []ICUTask  `json:"tasks"`
	TotalTasks          int        `json:"total_tasks"`
	CriticalCount       int        `json:"critical_count"`
	UrgentCount         int        `json:"urgent_count"`
	StatTaskCount       int        `json:"stat_task_count"`
	ViolationTaskCount  int        `json:"violation_task_count"`
	AlertTaskCount      int        `json:"alert_task_count"`
	PredictiveTaskCount int        `json:"predictive_task_count"`
}
