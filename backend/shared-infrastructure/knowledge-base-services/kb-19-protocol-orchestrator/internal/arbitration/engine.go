// Package arbitration implements the core protocol arbitration engine for KB-19.
//
// The arbitration engine follows an 8-step pipeline:
//   1. Collect candidate protocols based on CQL truth flags
//   2. Filter ineligible protocols (contraindicated, wrong setting)
//   3. Identify conflicts between applicable protocols
//   4. Apply priority hierarchy to resolve conflicts
//   5. Apply safety gatekeepers (ICU safety, pregnancy, etc.)
//   6. Assign recommendation strength (ACC/AHA Class I/IIa/IIb/III)
//   7. Produce human-readable narrative
//   8. Bind execution to KB-3/KB-12/KB-14
package arbitration

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"kb-19-protocol-orchestrator/internal/clients"
	"kb-19-protocol-orchestrator/internal/config"
	"kb-19-protocol-orchestrator/internal/models"
)

// Engine is the core arbitration engine for KB-19.
type Engine struct {
	cfg                *config.Config
	log                *logrus.Entry
	protocols          map[string]*models.ProtocolDescriptor
	conflictDetector   *ConflictDetector
	priorityResolver   *PriorityResolver
	safetyGatekeeper   *SafetyGatekeeper
	narrativeGenerator *NarrativeGenerator

	// KB Service Clients for execution binding (Step 8)
	kb3Client  *clients.KB3TemporalClient   // Temporal binding: deadlines, follow-ups, alerts
	kb12Client *clients.KB12OrderSetClient  // Order set activation: convert decisions to FHIR orders
	kb14Client *clients.KB14GovernanceClient // Governance tasks: escalation, approval workflows

	mu sync.RWMutex
}

// NewEngine creates a new arbitration engine.
func NewEngine(cfg *config.Config, log *logrus.Entry) (*Engine, error) {
	engine := &Engine{
		cfg:       cfg,
		log:       log.WithField("component", "arbitration-engine"),
		protocols: make(map[string]*models.ProtocolDescriptor),
	}

	// Initialize sub-components
	engine.conflictDetector = NewConflictDetector(log)
	engine.priorityResolver = NewPriorityResolver(log)
	engine.safetyGatekeeper = NewSafetyGatekeeper(log)
	engine.narrativeGenerator = NewNarrativeGenerator(log)

	// Initialize KB Service Clients for Step 8 execution binding
	kbTimeout := cfg.KBServices.Timeout
	if kbTimeout == 0 {
		kbTimeout = 30 * time.Second // Default timeout
	}

	// KB-3: Temporal binding (deadlines, follow-ups, alerts)
	engine.kb3Client = clients.NewKB3TemporalClient(cfg.KBServices.KB3URL, kbTimeout, log)
	log.WithField("kb3_url", cfg.KBServices.KB3URL).Debug("KB-3 Temporal client initialized")

	// KB-12: OrderSet activation (convert decisions to FHIR orders)
	engine.kb12Client = clients.NewKB12OrderSetClient(cfg.KBServices.KB12URL, kbTimeout, log)
	log.WithField("kb12_url", cfg.KBServices.KB12URL).Debug("KB-12 OrderSet client initialized")

	// KB-14: Governance tasks (escalation, approval workflows)
	engine.kb14Client = clients.NewKB14GovernanceClient(cfg.KBServices.KB14URL, kbTimeout, log)
	log.WithField("kb14_url", cfg.KBServices.KB14URL).Debug("KB-14 Governance client initialized")

	// Load protocol definitions
	if err := engine.loadProtocols(); err != nil {
		log.WithError(err).Warn("Failed to load protocols, using defaults")
	}

	return engine, nil
}

// Execute runs the full 8-step arbitration pipeline.
func (e *Engine) Execute(ctx context.Context, patientID, encounterID uuid.UUID, contextData map[string]interface{}) (*models.RecommendationBundle, error) {
	startTime := time.Now()
	e.log.WithFields(logrus.Fields{
		"patient_id":   patientID,
		"encounter_id": encounterID,
	}).Info("Starting protocol arbitration")

	// Create recommendation bundle
	bundle := models.NewRecommendationBundle(patientID, encounterID)

	// Create patient context (in production, this would fetch from Vaidshala)
	patientCtx := e.buildPatientContext(patientID, encounterID, contextData)

	// Step 1: Collect candidate protocols
	e.log.Debug("Step 1: Collecting candidate protocols")
	candidates := e.collectCandidateProtocols(patientCtx)
	e.log.WithField("candidates", len(candidates)).Debug("Candidates collected")

	// Step 2: Filter ineligible protocols
	e.log.Debug("Step 2: Filtering ineligible protocols")
	eligible := e.filterIneligible(candidates, patientCtx)
	e.log.WithField("eligible", len(eligible)).Debug("Eligible protocols identified")

	// Add protocol evaluations to bundle
	for _, eval := range eligible {
		bundle.AddProtocolEvaluation(eval)
	}

	// Step 3: Identify conflicts
	e.log.Debug("Step 3: Identifying conflicts")
	conflicts := e.conflictDetector.DetectConflicts(eligible)
	e.log.WithField("conflicts", len(conflicts)).Debug("Conflicts detected")

	// Step 4: Apply priority hierarchy
	e.log.Debug("Step 4: Applying priority hierarchy")
	resolvedDecisions := e.priorityResolver.Resolve(eligible, conflicts)
	for _, resolution := range conflicts {
		bundle.AddConflictResolution(resolution)
	}

	// Step 5: Apply safety gatekeepers
	e.log.Debug("Step 5: Applying safety gatekeepers")
	safeDecisions, gates := e.safetyGatekeeper.Apply(resolvedDecisions, patientCtx)
	for _, gate := range gates {
		bundle.AddSafetyGate(gate)
	}

	// Step 6: Assign recommendation strength
	e.log.Debug("Step 6: Grading recommendations")
	gradedDecisions := e.gradeRecommendations(safeDecisions)

	// Add decisions to bundle
	for _, decision := range gradedDecisions {
		bundle.AddDecision(decision)
	}

	// Step 7: Produce narrative
	e.log.Debug("Step 7: Generating narrative")
	bundle.NarrativeSummary = e.narrativeGenerator.Generate(bundle)

	// Step 8: Bind execution to KB-3/KB-12/KB-14
	e.log.Debug("Step 8: Binding execution to downstream KB services")
	e.bindExecution(ctx, bundle)

	// Finalize bundle
	bundle.ProcessingMetrics.TotalDurationMs = time.Since(startTime).Milliseconds()
	bundle.Finalize()

	e.log.WithFields(logrus.Fields{
		"patient_id":      patientID,
		"decisions":       len(bundle.Decisions),
		"conflicts":       len(bundle.ConflictsResolved),
		"duration_ms":     bundle.ProcessingMetrics.TotalDurationMs,
		"highest_urgency": bundle.ExecutiveSummary.HighestUrgency,
	}).Info("Protocol arbitration completed")

	return bundle, nil
}

// EvaluateProtocol evaluates a single protocol against patient context.
func (e *Engine) EvaluateProtocol(ctx context.Context, patientID, encounterID uuid.UUID, protocolID string, contextData map[string]interface{}) (*models.ProtocolEvaluation, error) {
	e.mu.RLock()
	protocol, exists := e.protocols[protocolID]
	e.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("protocol not found: %s", protocolID)
	}

	patientCtx := e.buildPatientContext(patientID, encounterID, contextData)
	evaluation := e.evaluateSingleProtocol(protocol, patientCtx)

	return &evaluation, nil
}

// ListProtocols returns all available protocols, optionally filtered.
func (e *Engine) ListProtocols(category, setting string) []*models.ProtocolDescriptor {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var result []*models.ProtocolDescriptor
	for _, p := range e.protocols {
		// Apply filters if provided
		if category != "" && string(p.Category) != category {
			continue
		}
		if setting != "" && !p.IsApplicableTo(models.ClinicalSetting(setting)) {
			continue
		}
		result = append(result, p)
	}

	return result
}

// GetProtocol returns a specific protocol by ID.
func (e *Engine) GetProtocol(protocolID string) (*models.ProtocolDescriptor, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	protocol, exists := e.protocols[protocolID]
	if !exists {
		return nil, fmt.Errorf("protocol not found: %s", protocolID)
	}

	return protocol, nil
}

// GetDecisionsForPatient returns recent decisions for a patient.
func (e *Engine) GetDecisionsForPatient(ctx context.Context, patientID uuid.UUID) ([]models.ArbitratedDecision, error) {
	// TODO: Implement database query
	return []models.ArbitratedDecision{}, nil
}

// GetBundle returns a recommendation bundle by ID.
func (e *Engine) GetBundle(ctx context.Context, bundleID uuid.UUID) (*models.RecommendationBundle, error) {
	// TODO: Implement database query
	return nil, fmt.Errorf("bundle not found")
}

// buildPatientContext creates a PatientContext from input data.
func (e *Engine) buildPatientContext(patientID, encounterID uuid.UUID, contextData map[string]interface{}) *models.PatientContext {
	ctx := models.NewPatientContext(patientID, encounterID)

	// In production, this would:
	// 1. Call Vaidshala CQL Engine for truth flags
	// 2. Call KB-8 for calculator scores
	// 3. Aggregate ICU state from ICU Intelligence

	// For now, extract from contextData if provided
	if contextData != nil {
		if flags, ok := contextData["cql_truth_flags"].(map[string]interface{}); ok {
			for k, v := range flags {
				if boolVal, ok := v.(bool); ok {
					ctx.CQLTruthFlags[k] = boolVal
				}
			}
		}
		if scores, ok := contextData["calculator_scores"].(map[string]interface{}); ok {
			for k, v := range scores {
				if floatVal, ok := v.(float64); ok {
					ctx.CalculatorScores[k] = floatVal
				}
			}
		}
	}

	return ctx
}

// collectCandidateProtocols identifies protocols that might apply based on CQL flags.
func (e *Engine) collectCandidateProtocols(patientCtx *models.PatientContext) []*models.ProtocolDescriptor {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var candidates []*models.ProtocolDescriptor

	for _, protocol := range e.protocols {
		if !protocol.IsActive {
			continue
		}

		// Check if any trigger criteria are met
		for _, trigger := range protocol.TriggerCriteria {
			if patientCtx.GetCQLFlag(trigger) {
				candidates = append(candidates, protocol)
				break
			}
		}
	}

	return candidates
}

// filterIneligible removes protocols that are contraindicated or not applicable.
func (e *Engine) filterIneligible(candidates []*models.ProtocolDescriptor, patientCtx *models.PatientContext) []models.ProtocolEvaluation {
	var evaluations []models.ProtocolEvaluation

	for _, protocol := range candidates {
		eval := e.evaluateSingleProtocol(protocol, patientCtx)
		if eval.IsApplicable || eval.Contraindicated {
			// Include both applicable and contraindicated for audit
			evaluations = append(evaluations, eval)
		}
	}

	return evaluations
}

// evaluateSingleProtocol evaluates one protocol against patient context.
func (e *Engine) evaluateSingleProtocol(protocol *models.ProtocolDescriptor, patientCtx *models.PatientContext) models.ProtocolEvaluation {
	eval := models.NewProtocolEvaluation(protocol.ID, protocol.Name)
	eval.PriorityClass = protocol.PriorityClass

	// Check contraindications
	for _, rule := range protocol.ContraindicationRules {
		if patientCtx.GetCQLFlag(rule) {
			eval.AddContraindication(rule)
			eval.RecordCQLFact(rule)
		}
	}

	// Check trigger criteria
	triggersFound := 0
	for _, trigger := range protocol.TriggerCriteria {
		if patientCtx.GetCQLFlag(trigger) {
			triggersFound++
			eval.RecordCQLFact(trigger)
		}
	}

	// Determine applicability
	if eval.Contraindicated {
		eval.MarkNotApplicable("Contraindicated: " + eval.ContraindicationReasons[0])
	} else if triggersFound > 0 {
		eval.MarkApplicable(fmt.Sprintf("%d trigger criteria met", triggersFound))
	} else {
		eval.MarkNotApplicable("No trigger criteria met")
	}

	// Record any calculator scores used
	for _, calcID := range protocol.RequiredCalculators {
		if score := patientCtx.GetCalculatorScore(calcID); score > 0 {
			eval.RecordCalculator(calcID, score)
		}
	}

	return *eval
}

// gradeRecommendations assigns ACC/AHA recommendation classes.
func (e *Engine) gradeRecommendations(decisions []models.ArbitratedDecision) []models.ArbitratedDecision {
	for i := range decisions {
		// Default grading logic
		switch decisions[i].DecisionType {
		case models.DecisionDo:
			if decisions[i].Urgency == models.UrgencySTAT {
				decisions[i].Evidence.RecommendationClass = models.ClassI
			} else {
				decisions[i].Evidence.RecommendationClass = models.ClassIIa
			}
		case models.DecisionConsider:
			decisions[i].Evidence.RecommendationClass = models.ClassIIb
		case models.DecisionAvoid:
			decisions[i].Evidence.RecommendationClass = models.ClassIII
		case models.DecisionDelay:
			decisions[i].Evidence.RecommendationClass = models.ClassIIb
		}

		// Finalize evidence envelope
		decisions[i].Evidence.Finalize()
	}

	// Sort by urgency
	sort.Slice(decisions, func(i, j int) bool {
		urgencyOrder := map[models.ActionUrgency]int{
			models.UrgencySTAT:      1,
			models.UrgencyUrgent:    2,
			models.UrgencyRoutine:   3,
			models.UrgencyScheduled: 4,
		}
		return urgencyOrder[decisions[i].Urgency] < urgencyOrder[decisions[j].Urgency]
	})

	return decisions
}

// bindExecution binds decisions to KB-3/KB-12/KB-14 for execution (Step 8).
// This converts abstract clinical decisions into concrete actions:
// - KB-3: Temporal bindings (scheduling, deadlines, alerts)
// - KB-12: Order set activations (FHIR orders)
// - KB-14: Governance tasks (escalations, approvals)
func (e *Engine) bindExecution(ctx context.Context, bundle *models.RecommendationBundle) {
	e.log.WithField("decisions", len(bundle.Decisions)).Debug("Starting execution binding")

	for _, decision := range bundle.Decisions {
		switch decision.DecisionType {
		case models.DecisionDo:
			// For DO decisions: activate order sets and create temporal bindings
			e.bindDoDecision(ctx, bundle, decision)

		case models.DecisionDelay:
			// For DELAY decisions: create follow-up temporal bindings
			e.bindDelayDecision(ctx, bundle, decision)

		case models.DecisionAvoid:
			// For AVOID decisions: create governance alerts/escalations
			e.bindAvoidDecision(ctx, bundle, decision)

		case models.DecisionConsider:
			// For CONSIDER decisions: create review tasks
			e.bindConsiderDecision(ctx, bundle, decision)
		}
	}

	e.log.WithFields(logrus.Fields{
		"temporal_bindings":    len(bundle.ExecutionPlan.TemporalBindings),
		"orderset_activations": len(bundle.ExecutionPlan.OrderSetActivations),
		"governance_tasks":     len(bundle.ExecutionPlan.GovernanceTasks),
	}).Debug("Execution binding completed")
}

// bindDoDecision handles execution binding for DO decisions.
func (e *Engine) bindDoDecision(ctx context.Context, bundle *models.RecommendationBundle, decision models.ArbitratedDecision) {
	// 1. Try to find and activate an order set via KB-12
	// Note: We pass empty category to let KB-12 auto-detect based on target
	orderSet, err := e.kb12Client.FindOrderSetForTarget(ctx, decision.Target, "")
	if err != nil {
		e.log.WithError(err).WithField("target", decision.Target).Warn("Failed to find order set")
	} else if orderSet != nil {
		// Activate the order set
		activation, err := e.kb12Client.ActivateOrderSet(ctx, clients.OrderSetActivationRequest{
			PatientID:      bundle.PatientID,
			EncounterID:    bundle.EncounterID,
			OrderSetID:     orderSet.ID,
			DecisionID:     decision.ID,
			Target:         decision.Target,
			Urgency:        string(decision.Urgency),
			SourceProtocol: decision.SourceProtocol,
			Rationale:      decision.Rationale,
		})
		if err != nil {
			e.log.WithError(err).WithField("orderset_id", orderSet.ID).Warn("Failed to activate order set")
		} else {
			// Record activation in bundle
			var individualOrders []string
			for _, order := range activation.GeneratedOrders {
				individualOrders = append(individualOrders, order.Display)
			}
			bundle.ExecutionPlan.OrderSetActivations = append(bundle.ExecutionPlan.OrderSetActivations, models.OrderSetActivation{
				DecisionID:       decision.ID,
				OrderSetID:       orderSet.ID,
				OrderSetName:     orderSet.Name,
				Parameters:       make(map[string]interface{}),
				ActivatedAt:      activation.ActivatedAt,
				IndividualOrders: individualOrders,
			})
		}
	}

	// 2. Create temporal binding via KB-3 based on urgency
	timing := e.determineTimingFromUrgency(decision.Urgency)
	temporalBinding, err := e.kb3Client.BindTiming(ctx, clients.TemporalBindingRequest{
		PatientID:      bundle.PatientID,
		EncounterID:    bundle.EncounterID,
		DecisionID:     decision.ID,
		DecisionType:   string(decision.DecisionType),
		Target:         decision.Target,
		SourceProtocol: decision.SourceProtocol,
		Urgency:        string(decision.Urgency),
		Timing:         timing,
	})
	if err != nil {
		e.log.WithError(err).WithField("decision_id", decision.ID).Warn("Failed to create temporal binding")
	} else {
		bundle.ExecutionPlan.TemporalBindings = append(bundle.ExecutionPlan.TemporalBindings, models.TemporalBinding{
			DecisionID:  decision.ID,
			ActionID:    temporalBinding.ID.String(),
			ScheduledAt: temporalBinding.DueAt,
			Deadline:    &temporalBinding.DueAt,
		})
	}
}

// bindDelayDecision handles execution binding for DELAY decisions.
func (e *Engine) bindDelayDecision(ctx context.Context, bundle *models.RecommendationBundle, decision models.ArbitratedDecision) {
	// Create a follow-up task via KB-3
	followUp, err := e.kb3Client.ScheduleFollowUp(ctx, clients.FollowUpRequest{
		PatientID:        bundle.PatientID,
		EncounterID:      bundle.EncounterID,
		FollowUpType:     "REASSESS",
		Reason:           fmt.Sprintf("Re-evaluate: %s - %s", decision.Target, decision.Rationale),
		ScheduleWithin:   "24h", // Default delay reassessment window
		Priority:         string(decision.Urgency),
		SourceDecisionID: decision.ID,
	})
	if err != nil {
		e.log.WithError(err).WithField("decision_id", decision.ID).Warn("Failed to schedule follow-up")
	} else {
		bundle.ExecutionPlan.TemporalBindings = append(bundle.ExecutionPlan.TemporalBindings, models.TemporalBinding{
			DecisionID:  decision.ID,
			ActionID:    followUp.ID.String(),
			ScheduledAt: followUp.ScheduledDate,
			Recurring:   false,
		})
	}

	// Also create a governance review task
	e.createGovernanceTask(ctx, bundle, decision, "REVIEW", "Review delayed action and reassess clinical appropriateness")
}

// bindAvoidDecision handles execution binding for AVOID decisions.
func (e *Engine) bindAvoidDecision(ctx context.Context, bundle *models.RecommendationBundle, decision models.ArbitratedDecision) {
	// Create an escalation/alert governance task via KB-14
	priority := "HIGH"
	if decision.Urgency == models.UrgencySTAT {
		priority = "CRITICAL"
	}

	e.createGovernanceTask(ctx, bundle, decision, "ESCALATION", fmt.Sprintf(
		"AVOID: %s - %s. Safety flags: %d",
		decision.Target,
		decision.Rationale,
		len(decision.SafetyFlags),
	))

	// If there are safety flags, create additional monitoring
	if len(decision.SafetyFlags) > 0 {
		_, _ = e.kb3Client.SetDeadline(ctx, clients.DeadlineRequest{
			PatientID:        bundle.PatientID,
			EncounterID:      bundle.EncounterID,
			DeadlineType:     "ACTION_REQUIRED",
			Description:      fmt.Sprintf("Acknowledge AVOID decision for %s", decision.Target),
			DueAt:            time.Now().Add(1 * time.Hour),
			EscalateAfter:    "30m",
			SourceDecisionID: decision.ID,
		})
	}

	_ = priority // Used in governance task
}

// bindConsiderDecision handles execution binding for CONSIDER decisions.
func (e *Engine) bindConsiderDecision(ctx context.Context, bundle *models.RecommendationBundle, decision models.ArbitratedDecision) {
	// Create a lower-priority review task
	e.createGovernanceTask(ctx, bundle, decision, "REVIEW", fmt.Sprintf(
		"Consider: %s - %s",
		decision.Target,
		decision.Rationale,
	))
}

// createGovernanceTask creates a governance task via KB-14.
func (e *Engine) createGovernanceTask(ctx context.Context, bundle *models.RecommendationBundle, decision models.ArbitratedDecision, taskType, description string) {
	priority := e.mapUrgencyToPriority(decision.Urgency)
	assignee := e.determineAssignee(decision)
	dueAt := e.calculateDueTime(decision.Urgency)

	task, err := e.kb14Client.CreateTask(ctx, clients.GovernanceTaskRequest{
		PatientID:        bundle.PatientID,
		EncounterID:      bundle.EncounterID,
		TaskType:         taskType,
		Priority:         priority,
		AssignedRole:     assignee,
		Title:            fmt.Sprintf("[%s] %s: %s", decision.DecisionType, taskType, decision.Target),
		Description:      description,
		DueAt:            dueAt,
		SourceDecisionID: decision.ID,
		RequiresSignoff:  taskType == "ESCALATION", // Escalations require acknowledgment
	})
	if err != nil {
		e.log.WithError(err).WithField("decision_id", decision.ID).Warn("Failed to create governance task")
		return
	}

	bundle.ExecutionPlan.GovernanceTasks = append(bundle.ExecutionPlan.GovernanceTasks, models.GovernanceTask{
		DecisionID:  decision.ID,
		TaskType:    taskType,
		AssignedTo:  assignee,
		Priority:    priority,
		DueAt:       dueAt,
		Description: description,
	})

	e.log.WithFields(logrus.Fields{
		"task_id":     task.ID,
		"task_type":   taskType,
		"decision_id": decision.ID,
	}).Debug("Governance task created")
}

// determineTimingFromUrgency converts urgency to temporal binding timing.
func (e *Engine) determineTimingFromUrgency(urgency models.ActionUrgency) *clients.Timing {
	switch urgency {
	case models.UrgencySTAT:
		return &clients.Timing{DueWithin: "15m"}
	case models.UrgencyUrgent:
		return &clients.Timing{DueWithin: "1h"}
	case models.UrgencyRoutine:
		return &clients.Timing{DueWithin: "24h"}
	case models.UrgencyScheduled:
		return &clients.Timing{DueWithin: "7d"}
	default:
		return &clients.Timing{DueWithin: "24h"}
	}
}

// mapUrgencyToPriority maps action urgency to governance priority.
func (e *Engine) mapUrgencyToPriority(urgency models.ActionUrgency) string {
	switch urgency {
	case models.UrgencySTAT:
		return "CRITICAL"
	case models.UrgencyUrgent:
		return "HIGH"
	case models.UrgencyRoutine:
		return "MEDIUM"
	default:
		return "LOW"
	}
}

// determineAssignee determines who should receive the governance task.
func (e *Engine) determineAssignee(decision models.ArbitratedDecision) string {
	// Default role-based assignment
	switch decision.Urgency {
	case models.UrgencySTAT:
		return "ATTENDING_PHYSICIAN"
	case models.UrgencyUrgent:
		return "CARE_TEAM"
	default:
		return "PRIMARY_PROVIDER"
	}
}

// calculateDueTime calculates due time based on urgency.
func (e *Engine) calculateDueTime(urgency models.ActionUrgency) time.Time {
	switch urgency {
	case models.UrgencySTAT:
		return time.Now().Add(15 * time.Minute)
	case models.UrgencyUrgent:
		return time.Now().Add(1 * time.Hour)
	case models.UrgencyRoutine:
		return time.Now().Add(24 * time.Hour)
	default:
		return time.Now().Add(7 * 24 * time.Hour)
	}
}

// loadProtocols loads protocol definitions from YAML files.
func (e *Engine) loadProtocols() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Load default protocols
	defaultProtocols := getDefaultProtocols()
	for _, p := range defaultProtocols {
		e.protocols[p.ID] = p
	}

	e.log.WithField("protocols_loaded", len(e.protocols)).Info("Protocols loaded")

	return nil
}

// getDefaultProtocols returns built-in protocol definitions.
// Protocol IDs MUST match KB-3's registry for temporal binding to work.
// KB-3 acute protocols: SEPSIS-SEP1-2021, STROKE-AHA-2019, STEMI-ACC-2013, DKA-ADA-2024, TRAUMA-ATLS-10, PE-ESC-2019
// KB-3 chronic protocols: COPD-GOLD-2024, HTN-ACCAHA-2017, DIABETES-ADA-2024, HF-ACCAHA-2022, CKD-KDIGO-2024, ANTICOAG-CHEST
func getDefaultProtocols() []*models.ProtocolDescriptor {
	return []*models.ProtocolDescriptor{
		{
			ID:            "SEPSIS-SEP1-2021", // Matches KB-3 registry
			Name:          "Sepsis 1-Hour Bundle",
			Description:   "SSC Sepsis Bundle for early sepsis management",
			Category:      models.CategoryEmergency,
			PriorityClass: models.PriorityEmergency,
			TriggerCriteria: []string{
				"HasSepsis",
				"HasSepsisAlert",
				"qSOFA >= 2",
			},
			ContraindicationRules: []string{
				"IsDNR",
				"IsComfortCareOnly",
			},
			RequiredCalculators: []string{"SOFA", "qSOFA"},
			GuidelineSource:     "Surviving Sepsis Campaign",
			GuidelineVersion:    "2021",
			ApplicableSettings: []models.ClinicalSetting{
				models.SettingED,
				models.SettingICU,
				models.SettingInpatient,
			},
			IsActive: true,
			Version:  "1.0.0",
		},
		{
			ID:            "HF-ACCAHA-2022", // Matches KB-3 registry
			Name:          "Heart Failure GDMT Optimization",
			Description:   "Guideline-directed medical therapy for HFrEF",
			Category:      models.CategoryChronic,
			PriorityClass: models.PriorityChronic,
			TriggerCriteria: []string{
				"HasHFrEF",
				"EF <= 40",
			},
			ContraindicationRules: []string{
				"HasCardiogenicShock",
				"SystolicBP < 90",
			},
			RequiredCalculators: []string{},
			GuidelineSource:     "ACC/AHA",
			GuidelineVersion:    "2022",
			ApplicableSettings: []models.ClinicalSetting{
				models.SettingOutpatient,
				models.SettingInpatient,
			},
			IsActive: true,
			Version:  "1.0.0",
		},
		{
			ID:            "ANTICOAG-CHEST", // Matches KB-3 registry
			Name:          "AFib Anticoagulation",
			Description:   "Stroke prevention in atrial fibrillation",
			Category:      models.CategoryChronic,
			PriorityClass: models.PriorityChronic,
			TriggerCriteria: []string{
				"HasAFib",
				"CHA2DS2VASc >= 2",
			},
			ContraindicationRules: []string{
				"HasActiveICH",
				"PlateletsLow",
				"HasActiveBleeding",
			},
			RequiredCalculators: []string{"CHA2DS2VASc", "HASBLED"},
			GuidelineSource:     "ACC/AHA/HRS",
			GuidelineVersion:    "2023",
			ApplicableSettings: []models.ClinicalSetting{
				models.SettingOutpatient,
				models.SettingInpatient,
			},
			IsActive: true,
			Version:  "1.0.0",
		},
		{
			ID:            "CKD-KDIGO-2024", // Matches KB-3 registry
			Name:          "AKI Prevention and Management",
			Description:   "KDIGO AKI prevention and management bundle",
			Category:      models.CategoryAcute,
			PriorityClass: models.PriorityAcute,
			TriggerCriteria: []string{
				"HasAKI",
				"HasAKIRisk",
				"CreatinineRising",
			},
			ContraindicationRules: []string{},
			RequiredCalculators:   []string{"eGFR", "CrCl"},
			GuidelineSource:       "KDIGO",
			GuidelineVersion:      "2024",
			ApplicableSettings: []models.ClinicalSetting{
				models.SettingICU,
				models.SettingInpatient,
			},
			IsActive: true,
			Version:  "1.0.0",
		},
		{
			ID:            "DIABETES-ADA-2024", // Matches KB-3 registry
			Name:          "Inpatient Glucose Management",
			Description:   "ADA inpatient glucose management protocol",
			Category:      models.CategoryChronic,
			PriorityClass: models.PriorityChronic,
			TriggerCriteria: []string{
				"HasDiabetes",
				"Hyperglycemia",
			},
			ContraindicationRules: []string{},
			RequiredCalculators:   []string{},
			GuidelineSource:       "ADA",
			GuidelineVersion:      "2024",
			ApplicableSettings: []models.ClinicalSetting{
				models.SettingICU,
				models.SettingInpatient,
			},
			IsActive: true,
			Version:  "1.0.0",
		},
	}
}
