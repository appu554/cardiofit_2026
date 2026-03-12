package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ExecutionContractService implements the ONC → MED-RT → OHDSI → LOINC pipeline
type ExecutionContractService struct {
	expansionService *OHDSIExpansionService
}

// NewExecutionContractService creates the execution contract service
func NewExecutionContractService(expansion *OHDSIExpansionService) *ExecutionContractService {
	return &ExecutionContractService{
		expansionService: expansion,
	}
}

// DDIEvaluationRequest is the input to the DDI evaluation pipeline
type DDIEvaluationRequest struct {
	PatientID       string             `json:"patient_id"`
	DrugConceptIDs  []int64            `json:"drug_concept_ids"`
	PatientLabs     map[string]float64 `json:"patient_labs"`      // LOINC code → value
	LabTimestamps   map[string]time.Time `json:"lab_timestamps,omitempty"`
}

// DDIEvaluationResponse is the output of the DDI evaluation pipeline
type DDIEvaluationResponse struct {
	RequestID        string        `json:"request_id"`
	PatientID        string        `json:"patient_id"`
	Timestamp        time.Time     `json:"timestamp"`

	// Results
	Alerts           []FinalAlert  `json:"alerts"`
	TotalChecked     int           `json:"total_checked"`
	CriticalCount    int           `json:"critical_count"`
	HighCount        int           `json:"high_count"`
	WarningCount     int           `json:"warning_count"`
	SuppressedCount  int           `json:"suppressed_count"`

	// Performance
	ExecutionTimeMs  int64         `json:"execution_time_ms"`

	// Audit
	EvaluationTrace  []TraceStep   `json:"evaluation_trace"`
}

// FinalAlert represents the output of the execution contract
type FinalAlert struct {
	AlertID          string    `json:"alert_id"`
	RuleID           int       `json:"rule_id"`

	// Drug Information
	Drug1ConceptID   int64     `json:"drug1_concept_id"`
	Drug1Name        string    `json:"drug1_name"`
	Drug2ConceptID   int64     `json:"drug2_concept_id"`
	Drug2Name        string    `json:"drug2_name"`

	// Severity
	FinalSeverity    string    `json:"final_severity"`
	BaseSeverity     string    `json:"base_severity"`
	WasEscalated     bool      `json:"was_escalated"`

	// Clinical Message
	AlertMessage     string    `json:"alert_message"`

	// Context Details
	ContextLabCode   *string   `json:"context_lab_code,omitempty"`
	ContextLabName   *string   `json:"context_lab_name,omitempty"`
	ContextLabValue  *float64  `json:"context_lab_value,omitempty"`
	ContextThreshold *float64  `json:"context_threshold,omitempty"`
	ContextOperator  *string   `json:"context_operator,omitempty"`
	ThresholdMet     bool      `json:"threshold_met"`

	// Governance (CMS-Ready)
	RuleAuthority    string    `json:"rule_authority"`
	RuleVersion      string    `json:"rule_version"`

	// Audit
	EvaluationPath   []string  `json:"evaluation_path"`
}

// TraceStep represents one step in the evaluation trace
type TraceStep struct {
	Layer     string `json:"layer"`     // PROJECTION, EXPANSION, CONTEXT, OUTPUT
	Step      string `json:"step"`
	Source    string `json:"source,omitempty"`
	Timestamp int64  `json:"timestamp_ms"`
}

// EvaluateDDI executes the full ONC → MED-RT → OHDSI → LOINC pipeline
func (s *ExecutionContractService) EvaluateDDI(ctx context.Context, req DDIEvaluationRequest) (*DDIEvaluationResponse, error) {
	startTime := time.Now()
	requestID := uuid.New().String()

	response := &DDIEvaluationResponse{
		RequestID:       requestID,
		PatientID:       req.PatientID,
		Timestamp:       startTime,
		Alerts:          make([]FinalAlert, 0),
		EvaluationTrace: make([]TraceStep, 0),
	}

	// ═══════════════════════════════════════════════════════════════════
	// LAYER 1: PROJECTION (ONC Rules)
	// ═══════════════════════════════════════════════════════════════════
	response.addTrace("PROJECTION", "Starting DDI evaluation", "ONC Constitutional Rules")

	// ═══════════════════════════════════════════════════════════════════
	// LAYER 2: EXPANSION (OHDSI Class Membership)
	// ═══════════════════════════════════════════════════════════════════
	response.addTrace("EXPANSION", fmt.Sprintf("Checking %d drugs for interactions", len(req.DrugConceptIDs)), "OHDSI Vocabulary")

	checkResult, err := s.expansionService.CheckDDI(ctx, req.DrugConceptIDs)
	if err != nil {
		return nil, fmt.Errorf("expansion layer failed: %w", err)
	}

	response.TotalChecked = checkResult.TotalChecked
	response.addTrace("EXPANSION", fmt.Sprintf("Found %d potential interactions", len(checkResult.Interactions)), "v_active_ddi_definitions")

	// ═══════════════════════════════════════════════════════════════════
	// LAYER 3: CONTEXT (LOINC Lab Evaluation)
	// ═══════════════════════════════════════════════════════════════════
	for _, projection := range checkResult.Interactions {
		alert := s.evaluateContext(projection, req.PatientLabs, response)

		if alert != nil {
			response.Alerts = append(response.Alerts, *alert)

			// Count by severity
			switch alert.FinalSeverity {
			case "CRITICAL":
				response.CriticalCount++
			case "HIGH":
				response.HighCount++
			case "WARNING":
				response.WarningCount++
			}
		} else {
			response.SuppressedCount++
		}
	}

	// ═══════════════════════════════════════════════════════════════════
	// LAYER 4: OUTPUT (Final Alert Generation)
	// ═══════════════════════════════════════════════════════════════════
	response.addTrace("OUTPUT", fmt.Sprintf("Generated %d alerts, suppressed %d", len(response.Alerts), response.SuppressedCount), "Execution Contract")

	response.ExecutionTimeMs = time.Since(startTime).Milliseconds()

	return response, nil
}

// evaluateContext applies LOINC context to a projection
func (s *ExecutionContractService) evaluateContext(projection DDIProjection, labs map[string]float64, response *DDIEvaluationResponse) *FinalAlert {
	alert := &FinalAlert{
		AlertID:        fmt.Sprintf("DDI-%d-%d-%d", projection.RuleID, projection.DrugAConceptID, projection.DrugBConceptID),
		RuleID:         projection.RuleID,
		Drug1ConceptID: projection.DrugAConceptID,
		Drug1Name:      projection.DrugAName,
		Drug2ConceptID: projection.DrugBConceptID,
		Drug2Name:      projection.DrugBName,
		BaseSeverity:   projection.RiskLevel,
		FinalSeverity:  projection.RiskLevel,
		AlertMessage:   projection.AlertMessage,
		RuleAuthority:  projection.RuleAuthority,
		RuleVersion:    projection.RuleVersion,
		EvaluationPath: make([]string, 0),
	}

	alert.EvaluationPath = append(alert.EvaluationPath,
		fmt.Sprintf("Rule %d: %s + %s", projection.RuleID, projection.DrugAName, projection.DrugBName))

	// ─────────────────────────────────────────────────────────────────
	// Case 1: No context defined → absolute rule, always alert
	// ─────────────────────────────────────────────────────────────────
	if projection.ContextLOINCID == nil {
		alert.EvaluationPath = append(alert.EvaluationPath, "No context defined - absolute contraindication")
		response.addTrace("CONTEXT", fmt.Sprintf("Rule %d: No context, absolute alert at %s", projection.RuleID, alert.FinalSeverity), "")
		return alert
	}

	alert.ContextLabCode = projection.ContextLOINCID
	alert.ContextThreshold = projection.ContextThreshold
	alert.ContextOperator = projection.ContextOperator

	// ─────────────────────────────────────────────────────────────────
	// Case 2: Context defined, check lab availability
	// ─────────────────────────────────────────────────────────────────
	labValue, labExists := labs[*projection.ContextLOINCID]

	if !labExists {
		// Lab missing
		if projection.ContextRequired {
			// context_required=true → FAIL-SAFE: alert anyway
			alert.EvaluationPath = append(alert.EvaluationPath,
				fmt.Sprintf("Lab %s unavailable, context_required=true, fail-safe alert", *projection.ContextLOINCID))
			response.addTrace("CONTEXT", fmt.Sprintf("Rule %d: Lab missing, fail-safe alert", projection.RuleID), "LOINC")
			return alert
		} else {
			// context_required=false → alert at base severity
			alert.EvaluationPath = append(alert.EvaluationPath,
				fmt.Sprintf("Lab %s unavailable, context_required=false, base severity", *projection.ContextLOINCID))
			response.addTrace("CONTEXT", fmt.Sprintf("Rule %d: Lab missing, base severity alert", projection.RuleID), "LOINC")
			return alert
		}
	}

	// ─────────────────────────────────────────────────────────────────
	// Case 3: Lab available, evaluate threshold
	// ─────────────────────────────────────────────────────────────────
	alert.ContextLabValue = &labValue
	thresholdMet := s.evaluateThreshold(labValue, projection.ContextOperator, projection.ContextThreshold)
	alert.ThresholdMet = thresholdMet

	if thresholdMet {
		// Threshold exceeded → ESCALATE severity
		alert.FinalSeverity = s.escalateSeverity(projection.RiskLevel)
		alert.WasEscalated = alert.FinalSeverity != projection.RiskLevel
		alert.EvaluationPath = append(alert.EvaluationPath,
			fmt.Sprintf("Lab %s = %.2f (threshold %s %.2f) - EXCEEDED",
				*projection.ContextLOINCID, labValue, *projection.ContextOperator, *projection.ContextThreshold))
		if alert.WasEscalated {
			alert.EvaluationPath = append(alert.EvaluationPath,
				fmt.Sprintf("Severity escalated: %s → %s", projection.RiskLevel, alert.FinalSeverity))
		}
		response.addTrace("CONTEXT", fmt.Sprintf("Rule %d: Threshold exceeded, severity=%s", projection.RuleID, alert.FinalSeverity), "LOINC")
		return alert
	} else {
		// Threshold not exceeded
		if !projection.ContextRequired {
			// context_required=false AND threshold not met → can suppress
			alert.EvaluationPath = append(alert.EvaluationPath,
				fmt.Sprintf("Lab %s = %.2f within safe range, context_required=false, SUPPRESSED",
					*projection.ContextLOINCID, labValue))
			response.addTrace("CONTEXT", fmt.Sprintf("Rule %d: Within safe range, suppressed", projection.RuleID), "LOINC")
			return nil // Suppress this alert
		} else {
			// context_required=true → still alert at base severity
			alert.EvaluationPath = append(alert.EvaluationPath,
				fmt.Sprintf("Lab %s = %.2f within safe range, context_required=true, base severity alert",
					*projection.ContextLOINCID, labValue))
			response.addTrace("CONTEXT", fmt.Sprintf("Rule %d: Within range but required context, alert at base", projection.RuleID), "LOINC")
			return alert
		}
	}
}

// evaluateThreshold checks if a lab value meets the threshold
func (s *ExecutionContractService) evaluateThreshold(value float64, operator *string, threshold *float64) bool {
	if operator == nil || threshold == nil {
		return false
	}

	switch *operator {
	case "<":
		return value < *threshold
	case ">":
		return value > *threshold
	case "<=":
		return value <= *threshold
	case ">=":
		return value >= *threshold
	case "=":
		return value == *threshold
	default:
		return false
	}
}

// escalateSeverity escalates severity when threshold is exceeded
func (s *ExecutionContractService) escalateSeverity(base string) string {
	switch base {
	case "WARNING":
		return "HIGH"
	case "HIGH":
		return "CRITICAL"
	case "MODERATE":
		return "WARNING"
	default:
		return base // CRITICAL stays CRITICAL
	}
}

// addTrace adds a step to the evaluation trace
func (r *DDIEvaluationResponse) addTrace(layer, step, source string) {
	r.EvaluationTrace = append(r.EvaluationTrace, TraceStep{
		Layer:     layer,
		Step:      step,
		Source:    source,
		Timestamp: time.Now().UnixMilli(),
	})
}
