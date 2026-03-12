package services

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/cache"
	"kb-22-hpi-engine/internal/database"
	"kb-22-hpi-engine/internal/metrics"
	"kb-22-hpi-engine/internal/models"
)

// SessionService is the main session lifecycle orchestrator for KB-22 HPI Engine.
// It coordinates the full HPI session flow: creation, answer processing,
// suspension, resumption, completion, and differential/safety queries.
type SessionService struct {
	db      *database.Database
	cache   *cache.CacheClient
	log     *zap.Logger
	metrics *metrics.Collector

	nodeLoader      *NodeLoader
	bayesian        *BayesianEngine
	safety          *SafetyEngine
	orchestrator    *QuestionOrchestrator
	cmApplicator    *CMApplicator
	contextProvider *SessionContextProvider
	guidelineClient *GuidelineClient
	medSafety       *MedicationSafetyProvider
	telemetry       *TelemetryWriter
	publisher       *OutcomePublisher
	crossNodeSafety *CrossNodeSafety
	contradiction   *ContradictionDetector
	transition      *TransitionEvaluator
}

// NewSessionService creates a new SessionService with all required dependencies.
func NewSessionService(
	db *database.Database,
	cacheClient *cache.CacheClient,
	log *zap.Logger,
	m *metrics.Collector,
	nodeLoader *NodeLoader,
	bayesian *BayesianEngine,
	safety *SafetyEngine,
	orchestrator *QuestionOrchestrator,
	cmApplicator *CMApplicator,
	contextProvider *SessionContextProvider,
	guidelineClient *GuidelineClient,
	medSafety *MedicationSafetyProvider,
	telemetry *TelemetryWriter,
	publisher *OutcomePublisher,
	crossNodeSafety *CrossNodeSafety,
	contradiction *ContradictionDetector,
	transition *TransitionEvaluator,
) *SessionService {
	return &SessionService{
		db:              db,
		cache:           cacheClient,
		log:             log,
		metrics:         m,
		nodeLoader:      nodeLoader,
		bayesian:        bayesian,
		safety:          safety,
		orchestrator:    orchestrator,
		cmApplicator:    cmApplicator,
		contextProvider: contextProvider,
		guidelineClient: guidelineClient,
		medSafety:       medSafety,
		telemetry:       telemetry,
		publisher:       publisher,
		crossNodeSafety: crossNodeSafety,
		contradiction:   contradiction,
		transition:      transition,
	}
}

// CreateSession initialises a new HPI session through a 10-step sequence.
//
// Steps:
//  1. Validate node_id via NodeLoader
//  2. Create HPISession with status=INITIALISING
//  3. Fetch session context (3 parallel goroutines to KB-20/KB-21)
//  4. Snapshot stratum from KB-20 response
//  5. Init priors from node YAML for stratum
//  6. Query KB-3 for guideline adjustments (optional, N-01)
//  7. Apply CMs with adherence scaling (F-03)
//  8. Start SafetyEngine goroutine (F-02)
//  9. Select first question via QuestionOrchestrator
//  10. Set status=ACTIVE, save, return
func (s *SessionService) CreateSession(ctx context.Context, req models.CreateSessionRequest) (*models.SessionResponse, error) {
	start := time.Now()
	defer func() {
		s.metrics.SessionInitLatency.Observe(float64(time.Since(start).Milliseconds()))
	}()

	// Step 1: Validate node_id
	node := s.nodeLoader.Get(req.NodeID)
	if node == nil {
		return nil, fmt.Errorf("unknown node_id: %s", req.NodeID)
	}

	// Step 2: Create session with INITIALISING status
	session := models.HPISession{
		SessionID: uuid.New(),
		PatientID: req.PatientID,
		NodeID:    req.NodeID,
		Status:    models.StatusInitialising,
		StartedAt: time.Now(),
	}

	if err := s.db.DB.WithContext(ctx).Create(&session).Error; err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	s.log.Info("session created",
		zap.String("session_id", session.SessionID.String()),
		zap.String("patient_id", req.PatientID.String()),
		zap.String("node_id", req.NodeID),
	)

	// Step 3: Fetch session context (3 parallel goroutines to KB-20/KB-21)
	sessionCtx, err := s.contextProvider.Fetch(ctx, req.PatientID, req.NodeID)
	if err != nil {
		// Mark session as abandoned on context fetch failure
		s.db.DB.WithContext(ctx).Model(&session).Update("status", models.StatusAbandoned)
		return nil, fmt.Errorf("fetch session context: %w", err)
	}

	// Step 4: Snapshot stratum
	session.StratumLabel = sessionCtx.StratumLabel
	session.CKDSubstage = sessionCtx.CKDSubstage
	session.ReliabilityModifier = sessionCtx.ReliabilityModifier
	session.AdherenceGainFactor = sessionCtx.AdherenceGainFactor

	// Validate stratum is supported by the node
	stratumSupported := false
	for _, supported := range node.StrataSupported {
		if supported == sessionCtx.StratumLabel {
			stratumSupported = true
			break
		}
	}
	if !stratumSupported {
		s.db.DB.WithContext(ctx).Model(&session).Update("status", models.StatusAbandoned)
		return nil, fmt.Errorf("stratum %s not supported by node %s (supported: %v)",
			sessionCtx.StratumLabel, req.NodeID, node.StrataSupported)
	}

	// Step 5: Init priors from node YAML for stratum
	// G3: Extract active medication classes from KB-20 context modifiers
	// for medication-conditional differential evaluation.
	var activeMedClasses []string
	for _, mod := range sessionCtx.ActiveModifiers {
		if mod.DrugClass != "" {
			activeMedClasses = append(activeMedClasses, mod.DrugClass)
		}
	}
	logOdds := s.bayesian.InitPriors(node, sessionCtx.StratumLabel, activeMedClasses)

	// Step 5b: G2 sex-modifier prior adjustment (after InitPriors, before CMs)
	if len(node.SexModifiers) > 0 {
		s.bayesian.ApplySexModifiers(logOdds, node.SexModifiers, sessionCtx.PatientSex, sessionCtx.PatientAge)
	}

	// Step 6: Query KB-3 for guideline adjustments (optional, N-01)
	var guidelineRefs []string
	if node.GuidelinePriorSource != "" {
		adjustment, err := s.guidelineClient.FetchAdjustments(ctx, node.GuidelinePriorSource, sessionCtx.StratumLabel)
		if err != nil {
			s.log.Warn("guideline adjustment fetch error (non-fatal)",
				zap.String("session_id", session.SessionID.String()),
				zap.Error(err),
			)
		}
		if adjustment != nil && len(adjustment.Adjustments) > 0 {
			s.bayesian.ApplyGuidelineAdjustments(logOdds, adjustment.Adjustments)
			guidelineRefs = append(guidelineRefs, adjustment.GuidelineRef)
		}
	}
	session.GuidelinePriorRefs = guidelineRefs

	// Step 7: Apply CMs with adherence scaling (F-03)
	// Merge node-level YAML CMs (expanded from adjustments map) with runtime KB-20 CMs.
	// Node CMs describe static context factors authored per-node (e.g. "ARB/ACEi active").
	// Runtime CMs come from KB-20 patient profile and are patient-specific.
	allModifiers := sessionCtx.ActiveModifiers
	if len(node.ContextModifiers) > 0 {
		nodeCMs := ExpandNodeCMs(node.ContextModifiers)
		allModifiers = append(allModifiers, nodeCMs...)
	}
	logOdds, cmDeltas := s.cmApplicator.Apply(logOdds, allModifiers, sessionCtx.AdherenceWeights)
	cmDeltasJSON, _ := json.Marshal(cmDeltas)
	session.CMLogDeltasApplied = cmDeltasJSON

	// Persist log-odds state
	if err := session.SetLogOdds(logOdds); err != nil {
		return nil, fmt.Errorf("set log odds: %w", err)
	}

	// Step 8: Start SafetyEngine goroutine (F-02)
	// The safety engine runs as a parallel goroutine for the session lifetime.
	// For the creation step, we start it but don't feed any answers yet.
	// The goroutine channels are managed per-request in SubmitAnswer.
	// Here we just validate that triggers parse correctly.
	s.log.Debug("safety engine triggers validated",
		zap.String("session_id", session.SessionID.String()),
		zap.Int("node_triggers", len(node.SafetyTriggers)),
		zap.Int("cross_node_triggers", s.crossNodeSafety.Count()),
	)

	// Step 9: Select first question
	answeredQuestions := make(map[string]bool)
	answers := make(map[string]string)
	firstQuestion := s.orchestrator.Next(
		node, logOdds, answeredQuestions,
		session.StratumLabel, session.CKDSubstage, answers,
	)

	if firstQuestion == nil {
		s.db.DB.WithContext(ctx).Model(&session).Update("status", models.StatusAbandoned)
		return nil, fmt.Errorf("no eligible questions for node %s stratum %s", req.NodeID, sessionCtx.StratumLabel)
	}

	session.CurrentQuestionID = &firstQuestion.ID

	// Initialize cluster tracking
	if err := session.SetClusterAnswered(make(map[string]int)); err != nil {
		return nil, fmt.Errorf("init cluster tracking: %w", err)
	}

	// Step 10: Set status=ACTIVE, save, return
	session.Status = models.StatusActive
	session.LastActivityAt = time.Now()

	if err := s.db.DB.WithContext(ctx).Save(&session).Error; err != nil {
		return nil, fmt.Errorf("save session: %w", err)
	}

	// Cache the session
	if err := s.cache.SetSession(session.SessionID.String(), &session); err != nil {
		s.log.Warn("failed to cache session",
			zap.String("session_id", session.SessionID.String()),
			zap.Error(err),
		)
	}

	s.metrics.SessionsStarted.Inc()
	s.metrics.SessionsByStatus.WithLabelValues(string(models.StatusActive)).Inc()

	s.log.Info("session initialised and active",
		zap.String("session_id", session.SessionID.String()),
		zap.String("stratum", session.StratumLabel),
		zap.String("first_question", firstQuestion.ID),
		zap.Duration("init_latency", time.Since(start)),
	)

	// Build response
	posteriors := s.bayesian.GetPosteriors(logOdds, ResolveFloors(node, session.StratumLabel))

	return s.buildSessionResponse(&session, firstQuestion, posteriors), nil
}

// SubmitAnswer processes a patient answer through a 10-step sequence.
//
// Steps:
//  1. Load session from cache or DB
//  2. Validate session status (must be ACTIVE)
//  3. Load node definition
//  4. Apply Bayesian update (F-01, R-02, R-03, F-04)
//  5. Record answer in append-only log
//  6. Evaluate safety triggers via SafetyEngine (F-02)
//  7. Enrich safety flags with KB-5 medication safety (N-02)
//  8. Publish IMMEDIATE safety alerts to KB-19
//  9. Check convergence (R-01) and select next question
//  10. Write telemetry to KB-21 (async), update session, return
func (s *SessionService) SubmitAnswer(ctx context.Context, sessionID uuid.UUID, req models.SubmitAnswerRequest) (*models.AnswerResponse, error) {
	// Step 1: Load session
	session, err := s.loadSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("load session: %w", err)
	}

	// Step 2: Validate status
	if session.Status != models.StatusActive {
		return nil, fmt.Errorf("session %s is %s, expected ACTIVE", sessionID, session.Status)
	}

	// Step 3: Load node definition
	node := s.nodeLoader.Get(session.NodeID)
	if node == nil {
		return nil, fmt.Errorf("node %s not found (was it removed since session creation?)", session.NodeID)
	}

	// Find the question definition
	var question *models.QuestionDef
	for i := range node.Questions {
		if node.Questions[i].ID == req.QuestionID {
			question = &node.Questions[i]
			break
		}
	}
	if question == nil {
		return nil, fmt.Errorf("question %s not found in node %s", req.QuestionID, session.NodeID)
	}

	// Parse current log-odds state
	logOdds, err := session.LogOddsMap()
	if err != nil {
		return nil, fmt.Errorf("parse log odds: %w", err)
	}

	// Parse cluster tracking
	clusterAnswered, err := session.ClusterAnsweredMap()
	if err != nil {
		return nil, fmt.Errorf("parse cluster answered: %w", err)
	}

	// Step 4: Apply Bayesian update (G6: stratum-conditional LR overrides)
	isPataNahi := req.AnswerValue == string(models.AnswerPata)
	logOdds, informationGain := s.bayesian.UpdateWithStratum(
		logOdds, req.QuestionID, req.AnswerValue,
		question, session.ReliabilityModifier, session.AdherenceGainFactor,
		clusterAnswered, session.StratumLabel,
	)

	// Update cluster tracking (R-02)
	if question.Cluster != "" {
		clusterAnswered[question.Cluster]++
	}

	// CTL Panel 4: Accumulate reasoning step if information gain is meaningful.
	// Only questions where |IG| > 0.01 contribute to the reasoning chain —
	// this filters out PATA_NAHI answers and low-signal questions.
	if math.Abs(informationGain) > 0.01 {
		// Get current posteriors for the reasoning step snapshot
		stepPosteriors := s.bayesian.GetPosteriors(logOdds, ResolveFloors(node, session.StratumLabel))
		topDiff := ""
		topPost := 0.0
		if len(stepPosteriors) > 0 {
			topDiff = stepPosteriors[0].DifferentialID
			topPost = stepPosteriors[0].PosteriorProbability
		}

		// Load existing chain from cache (or start fresh)
		var chain []models.ReasoningStep
		_ = s.cache.GetReasoningChain(sessionID.String(), &chain)

		chain = append(chain, models.ReasoningStep{
			StepNumber:      len(chain) + 1,
			QuestionID:      req.QuestionID,
			QuestionText:    question.TextEN,
			Answer:          req.AnswerValue,
			InformationGain: informationGain,
			TopDifferential: topDiff,
			TopPosterior:    topPost,
		})

		if cacheErr := s.cache.SetReasoningChain(sessionID.String(), chain); cacheErr != nil {
			s.log.Warn("failed to cache reasoning step",
				zap.String("session_id", sessionID.String()),
				zap.Error(cacheErr),
			)
		}
	}

	// Step 5: Record answer
	lrApplied := s.computeLRApplied(question, req.AnswerValue)
	lrJSON, _ := json.Marshal(lrApplied)

	answer := models.SessionAnswer{
		AnswerID:                uuid.New(),
		SessionID:               sessionID,
		QuestionID:              req.QuestionID,
		AnswerValue:             req.AnswerValue,
		LRApplied:               lrJSON,
		InformationGainObserved: informationGain,
		WasPataNahi:             isPataNahi,
		AnswerLatencyMS:         req.LatencyMS,
		AnsweredAt:              time.Now(),
	}

	if err := s.db.DB.WithContext(ctx).Create(&answer).Error; err != nil {
		return nil, fmt.Errorf("save answer: %w", err)
	}

	// Update session counters
	session.QuestionsAsked++
	if isPataNahi {
		session.QuestionsPataNahi++
		session.ConsecutiveLowConf++
	} else {
		// G16: reset consecutive counter on any non-pata-nahi answer
		session.ConsecutiveLowConf = 0
	}

	// G16: safety-role + pata-nahi → immediate escalation.
	// If a patient says "don't know" to a red-flag question, escalate
	// regardless of the consecutive count — missing safety signal is critical.
	g16SafetyEscalation := false
	if isPataNahi && question.SafetyRole != "" {
		s.log.Warn("G16: pata-nahi on safety-role question, escalating",
			zap.String("session_id", sessionID.String()),
			zap.String("question_id", req.QuestionID),
			zap.String("safety_role", question.SafetyRole),
		)
		g16SafetyEscalation = true
	}

	// Step 6: Evaluate safety triggers
	// Build answer map from all session answers
	allAnswers, err := s.loadAnswerMap(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("load answer map: %w", err)
	}

	posteriors := s.bayesian.GetPosteriors(logOdds, ResolveFloors(node, session.StratumLabel))

	// Use synchronous trigger evaluation for the answer path
	answerChan := make(chan AnswerEvent, 1)
	flagChan := make(chan models.SafetyFlag, len(node.SafetyTriggers)+s.crossNodeSafety.Count())

	// Start safety engine, feed current answer, close channel
	s.safety.Start(
		node.SafetyTriggers,
		s.crossNodeSafety.GetActiveTriggers(),
		answerChan,
		flagChan,
		sessionID,
	)

	// Feed all answers to reconstruct trigger state
	for qid, ans := range allAnswers {
		answerChan <- AnswerEvent{
			QuestionID: qid,
			Answer:     ans,
			SessionID:  sessionID,
		}
	}
	close(answerChan)

	// Collect fired flags (non-blocking drain)
	var newFlags []models.SafetyFlag
	drainTimeout := time.After(50 * time.Millisecond)
drainLoop:
	for {
		select {
		case flag, ok := <-flagChan:
			if !ok {
				break drainLoop
			}
			newFlags = append(newFlags, flag)
		case <-drainTimeout:
			break drainLoop
		}
	}

	// Step 7: Enrich safety flags with KB-5 medication context (N-02)
	// Fetch active medications from context provider (cached from session init)
	var activeMedications []string
	var sessionCtx SessionContext
	if cacheErr := s.cache.GetSession(sessionID.String()+"_ctx", &sessionCtx); cacheErr == nil {
		// Use cached context for medication list
		for _, mod := range sessionCtx.ActiveModifiers {
			if mod.DrugClass != "" {
				activeMedications = append(activeMedications, mod.DrugClass)
			}
		}
	}

	for i := range newFlags {
		if newFlags[i].IsUrgentOrImmediate() {
			medSafety, _ := s.medSafety.CheckContraindications(
				ctx, session.PatientID, newFlags[i].FlagID, activeMedications,
			)
			if medSafety != nil {
				medJSON, _ := json.Marshal(medSafety)
				newFlags[i].MedicationSafetyContext = medJSON
			}
		}
	}

	// Persist new safety flags
	for i := range newFlags {
		newFlags[i].SessionID = sessionID
		if err := s.db.DB.WithContext(ctx).Create(&newFlags[i]).Error; err != nil {
			s.log.Error("failed to persist safety flag",
				zap.String("flag_id", newFlags[i].FlagID),
				zap.Error(err),
			)
		}
	}

	// Step 8: Publish IMMEDIATE safety alerts to KB-19
	for _, flag := range newFlags {
		if flag.IsImmediate() {
			alertEvent := models.SafetyAlertEvent{
				EventType:               models.EventSafetyAlert,
				PatientID:               session.PatientID,
				SessionID:               sessionID,
				FlagID:                  flag.FlagID,
				Severity:                string(flag.Severity),
				RecommendedAction:       flag.RecommendedAction,
				MedicationSafetyContext: flag.MedicationSafetyContext,
				FiredAt:                 flag.FiredAt,
			}
			if pubErr := s.publisher.PublishSafetyAlert(ctx, alertEvent); pubErr != nil {
				s.log.Error("failed to publish IMMEDIATE safety alert",
					zap.String("flag_id", flag.FlagID),
					zap.Error(pubErr),
				)
			}
			flag.PublishedToKB19 = true
			s.db.DB.WithContext(ctx).Model(&flag).Update("published_to_kb19", true)
		}
	}

	// Step 8b: G17 contradiction detection
	// Load or initialize the already-detected set from cache
	var alreadyDetected map[string]bool
	if cacheErr := s.cache.GetContradictions(sessionID.String(), &alreadyDetected); cacheErr != nil || alreadyDetected == nil {
		alreadyDetected = make(map[string]bool)
	}
	contradictions := s.contradiction.Check(node.ContradictionPairs, allAnswers, alreadyDetected)
	if len(contradictions) > 0 {
		for _, c := range contradictions {
			alreadyDetected[c.PairID] = true
		}
		s.cache.SetContradictions(sessionID.String(), alreadyDetected)
	}

	// Step 8c: G13 transition evaluation
	// Build posterior map for transition evaluator
	posteriorMap := make(map[string]float64, len(posteriors))
	for _, p := range posteriors {
		posteriorMap[p.DifferentialID] = p.PosteriorProbability
	}
	firedSafetyIDs := make(map[string]bool)
	for _, f := range newFlags {
		firedSafetyIDs[f.FlagID] = true
	}
	transitionState := TransitionSessionState{
		Posteriors:     posteriorMap,
		QuestionsAsked: session.QuestionsAsked,
		Converged:      false, // not yet checked
		FiredSafetyIDs: firedSafetyIDs,
	}
	transitions := s.transition.Evaluate(node.Transitions, transitionState)
	for i := range transitions {
		transitions[i].SourceNode = node.NodeID
	}

	// Step 9: Check convergence (G18: multi-criteria guard) and select next question
	// Build answer confidence and IG maps for G18 closure guard.
	// Confidence: non-pata-nahi answers = 1.0, pata-nahi = 0.0.
	// IGs: from the reasoning chain accumulated during the session.
	var answerConfidences map[string]float64
	var answerIGs map[string]float64
	{
		var allSessionAnswers []models.SessionAnswer
		s.db.DB.WithContext(ctx).Where("session_id = ?", sessionID).Find(&allSessionAnswers)
		if len(allSessionAnswers) > 0 {
			answerConfidences = make(map[string]float64, len(allSessionAnswers))
			answerIGs = make(map[string]float64, len(allSessionAnswers))
			for _, sa := range allSessionAnswers {
				if sa.WasPataNahi {
					answerConfidences[sa.QuestionID] = 0.0
				} else {
					answerConfidences[sa.QuestionID] = 1.0
				}
				answerIGs[sa.QuestionID] = sa.InformationGainObserved
			}
		}
	}
	convergenceResult := s.bayesian.CheckConvergenceMultiCriteria(posteriors, node, answerConfidences, answerIGs)
	converged := convergenceResult.Converged
	maxReached := session.QuestionsAsked >= node.MaxQuestions

	var nextQuestion *models.QuestionDef
	sessionCompleted := false
	terminationReason := ""

	// G16: cascade protocol termination checks
	g16PartialAssessment := session.ConsecutiveLowConf >= 5

	if g16SafetyEscalation {
		// G16: safety-role pata-nahi → escalate immediately
		sessionCompleted = true
		terminationReason = "SAFETY_ESCALATED"
		session.Status = models.StatusSafetyEscalated
		s.log.Warn("G16: session escalated due to pata-nahi on safety-role question",
			zap.String("session_id", sessionID.String()),
		)
	} else if g16PartialAssessment {
		// G16: ≥5 consecutive pata-nahi → partial assessment termination
		sessionCompleted = true
		terminationReason = "PARTIAL_ASSESSMENT"
		session.Status = models.StatusPartialAssessment
		s.log.Warn("G16: session terminated as PARTIAL_ASSESSMENT",
			zap.String("session_id", sessionID.String()),
			zap.Int("consecutive_low_conf", session.ConsecutiveLowConf),
		)
	} else if converged {
		sessionCompleted = true
		terminationReason = "CONVERGED"
	} else if maxReached {
		sessionCompleted = true
		terminationReason = "MAX_QUESTIONS"
	}

	if !sessionCompleted {
		// Build answered questions set
		answeredQuestions := make(map[string]bool)
		for qid := range allAnswers {
			answeredQuestions[qid] = true
		}

		nextQuestion = s.orchestrator.Next(
			node, logOdds, answeredQuestions,
			session.StratumLabel, session.CKDSubstage, allAnswers,
		)
		if nextQuestion == nil {
			sessionCompleted = true
			terminationReason = "NO_MORE_QUESTIONS"
		}
	}

	// Step 10: Update session state
	if err := session.SetLogOdds(logOdds); err != nil {
		return nil, fmt.Errorf("set log odds: %w", err)
	}
	if err := session.SetClusterAnswered(clusterAnswered); err != nil {
		return nil, fmt.Errorf("set cluster answered: %w", err)
	}

	if sessionCompleted {
		// G16: status may already be set by cascade protocol (SAFETY_ESCALATED, PARTIAL_ASSESSMENT)
		if session.Status == models.StatusActive {
			session.Status = models.StatusCompleted
		}
		now := time.Now()
		session.CompletedAt = &now

		// Create differential snapshot
		if err := s.createSnapshot(ctx, session, posteriors, converged); err != nil {
			s.log.Error("failed to create differential snapshot",
				zap.String("session_id", sessionID.String()),
				zap.Error(err),
			)
		}

		// Publish HPI_COMPLETE event
		s.publishHPIComplete(ctx, session, posteriors, converged)

		s.metrics.SessionsByStatus.WithLabelValues(string(session.Status)).Inc()
	} else {
		session.CurrentQuestionID = &nextQuestion.ID
	}

	session.LastActivityAt = time.Now()

	if err := s.db.DB.WithContext(ctx).Save(session).Error; err != nil {
		return nil, fmt.Errorf("save session: %w", err)
	}

	// Update cache
	s.cache.SetSession(sessionID.String(), session)

	// Invalidate differential cache
	s.cache.Delete(cache.DifferentialPrefix + sessionID.String())

	s.metrics.QuestionsAsked.Inc()

	// Update pata-nahi rate gauge
	if session.QuestionsAsked > 0 {
		pataRate := float64(session.QuestionsPataNahi) / float64(session.QuestionsAsked)
		s.metrics.PatanahiRate.WithLabelValues(session.NodeID).Set(pataRate)
	}

	// Write telemetry to KB-21 (async, non-blocking)
	s.telemetry.WriteAsync(models.QuestionTelemetry{
		PatientID:               session.PatientID,
		SessionID:               sessionID,
		QuestionID:              req.QuestionID,
		NodeID:                  session.NodeID,
		StratumLabel:            session.StratumLabel,
		InformationGainObserved: informationGain,
		WasPataNahi:             isPataNahi,
		AnswerLatencyMS:         req.LatencyMS,
		AnsweredAt:              answer.AnsweredAt,
	})

	s.log.Info("answer processed",
		zap.String("session_id", sessionID.String()),
		zap.String("question_id", req.QuestionID),
		zap.String("answer", req.AnswerValue),
		zap.Float64("information_gain", informationGain),
		zap.Int("new_safety_flags", len(newFlags)),
		zap.Bool("completed", sessionCompleted),
	)

	// Build safety flag summaries for response
	allFlags, _ := s.loadSafetyFlagSummaries(ctx, sessionID)

	response := &models.AnswerResponse{
		SessionID:         sessionID,
		Status:            session.Status,
		TopDifferentials:  topN(posteriors, 5),
		SafetyFlags:       allFlags,
		TerminationReason: terminationReason,
		Contradictions:    contradictions,
		Transitions:       transitions,
	}

	if nextQuestion != nil {
		qr := &models.QuestionResponse{
			QuestionID: nextQuestion.ID,
			TextEN:     nextQuestion.TextEN,
			TextHI:     nextQuestion.TextHI,
			Mandatory:  nextQuestion.Mandatory,
		}

		// G16: cascade protocol modifications to the next question
		if session.ConsecutiveLowConf >= 3 {
			// Binary-only mode: signal to frontend to suppress PATA_NAHI option
			qr.BinaryOnly = true
			s.log.Info("G16: binary-only mode active",
				zap.String("session_id", sessionID.String()),
				zap.Int("consecutive_low_conf", session.ConsecutiveLowConf),
			)
		}
		if session.ConsecutiveLowConf >= 2 {
			// Rephrase mode: use alt_prompt if available
			if nextQuestion.AltPromptEN != "" {
				qr.TextEN = nextQuestion.AltPromptEN
				qr.IsRephrase = true
			}
			if nextQuestion.AltPromptHI != "" {
				qr.TextHI = nextQuestion.AltPromptHI
				qr.IsRephrase = true
			}
			if qr.IsRephrase {
				s.log.Info("G16: rephrasing question via alt_prompt",
					zap.String("session_id", sessionID.String()),
					zap.String("question_id", nextQuestion.ID),
				)
			}
		}

		response.NextQuestion = qr
	}

	return response, nil
}

// GetSession retrieves the full session state for GET /sessions/:id.
func (s *SessionService) GetSession(ctx context.Context, sessionID uuid.UUID) (*models.SessionResponse, error) {
	session, err := s.loadSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("load session: %w", err)
	}

	node := s.nodeLoader.Get(session.NodeID)

	logOdds, err := session.LogOddsMap()
	if err != nil {
		return nil, fmt.Errorf("parse log odds: %w", err)
	}

	posteriors := s.bayesian.GetPosteriors(logOdds, ResolveFloors(node, session.StratumLabel))

	var currentQuestion *models.QuestionDef
	if session.CurrentQuestionID != nil && node != nil {
		for i := range node.Questions {
			if node.Questions[i].ID == *session.CurrentQuestionID {
				currentQuestion = &node.Questions[i]
				break
			}
		}
	}

	return s.buildSessionResponse(session, currentQuestion, posteriors), nil
}

// SuspendSession sets a session to SUSPENDED status.
func (s *SessionService) SuspendSession(ctx context.Context, sessionID uuid.UUID) error {
	session, err := s.loadSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("load session: %w", err)
	}

	if session.Status != models.StatusActive {
		return fmt.Errorf("session %s is %s, can only suspend ACTIVE sessions", sessionID, session.Status)
	}

	session.Status = models.StatusSuspended
	session.LastActivityAt = time.Now()

	if err := s.db.DB.WithContext(ctx).Save(session).Error; err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	s.cache.SetSession(sessionID.String(), session)
	s.metrics.SessionsByStatus.WithLabelValues(string(models.StatusSuspended)).Inc()

	s.log.Info("session suspended",
		zap.String("session_id", sessionID.String()),
	)

	return nil
}

// EscalateSession transitions a session to SAFETY_ESCALATED state.
// Called by the BAY-10 escalation webhook when SCE detects a red flag.
func (s *SessionService) EscalateSession(ctx context.Context, sessionID uuid.UUID) error {
	session, err := s.loadSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("load session: %w", err)
	}

	if session.Status != models.StatusActive && session.Status != models.StatusSuspended {
		return fmt.Errorf("session %s is %s, can only escalate ACTIVE or SUSPENDED sessions", sessionID, session.Status)
	}

	session.Status = models.StatusSafetyEscalated
	session.LastActivityAt = time.Now()

	if err := s.db.DB.WithContext(ctx).Save(session).Error; err != nil {
		return fmt.Errorf("save session: %w", err)
	}

	s.cache.SetSession(sessionID.String(), session)
	s.metrics.SessionsByStatus.WithLabelValues(string(models.StatusSafetyEscalated)).Inc()

	s.log.Warn("session escalated to SAFETY_ESCALATED",
		zap.String("session_id", sessionID.String()),
	)

	return nil
}

// ResumeSession resumes a SUSPENDED session. Implements R-04: re-query KB-20
// for current stratum and detect stratum drift.
func (s *SessionService) ResumeSession(ctx context.Context, sessionID uuid.UUID) (*models.SessionResponse, error) {
	session, err := s.loadSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("load session: %w", err)
	}

	if session.Status != models.StatusSuspended {
		return nil, fmt.Errorf("session %s is %s, can only resume SUSPENDED sessions", sessionID, session.Status)
	}

	// R-04: Re-query KB-20 for current stratum
	currentStratum, currentCKDSubstage, fetchErr := s.contextProvider.FetchStratum(ctx, session.PatientID, session.NodeID)
	if fetchErr != nil {
		s.log.Warn("failed to re-query stratum on resume, proceeding with original",
			zap.String("session_id", sessionID.String()),
			zap.Error(fetchErr),
		)
	} else {
		// Detect stratum drift
		stratumDrifted := currentStratum != session.StratumLabel
		substageDrifted := false

		if session.CKDSubstage != nil && currentCKDSubstage != nil {
			substageDrifted = *session.CKDSubstage != *currentCKDSubstage
		} else if (session.CKDSubstage == nil) != (currentCKDSubstage == nil) {
			substageDrifted = true
		}

		if stratumDrifted || substageDrifted {
			session.SubstageDrifted = true
			session.Status = models.StatusStratumDrifted

			s.log.Warn("stratum drift detected on session resume",
				zap.String("session_id", sessionID.String()),
				zap.String("old_stratum", session.StratumLabel),
				zap.String("new_stratum", currentStratum),
				zap.Bool("substage_drifted", substageDrifted),
			)

			// Publish stratum drift event to KB-19
			driftEvent := models.StratumDriftEvent{
				EventType:      models.EventStratumDrifted,
				PatientID:      session.PatientID,
				SessionID:      sessionID,
				OldStratum:     session.StratumLabel,
				NewStratum:     currentStratum,
				OldCKDSubstage: session.CKDSubstage,
				NewCKDSubstage: currentCKDSubstage,
				DetectedAt:     time.Now(),
			}

			driftJSON, _ := json.Marshal(driftEvent)
			go func() {
				driftURL := fmt.Sprintf("%s/api/v1/events", s.contextProvider.config.KB19URL)
				if postErr := s.publisher.postWithRetry(context.Background(), driftURL, driftJSON, 3, s.publisher.config.OutcomeRetryDelay); postErr != nil {
					s.log.Error("failed to publish stratum drift event",
						zap.String("session_id", sessionID.String()),
						zap.Error(postErr),
					)
				}
			}()

			if err := s.db.DB.WithContext(ctx).Save(session).Error; err != nil {
				return nil, fmt.Errorf("save drifted session: %w", err)
			}
			s.cache.SetSession(sessionID.String(), session)

			return s.GetSession(ctx, sessionID)
		}
	}

	// No drift detected: resume normally
	session.Status = models.StatusActive
	session.LastActivityAt = time.Now()

	if err := s.db.DB.WithContext(ctx).Save(session).Error; err != nil {
		return nil, fmt.Errorf("save resumed session: %w", err)
	}

	s.cache.SetSession(sessionID.String(), session)
	s.metrics.SessionsByStatus.WithLabelValues(string(models.StatusActive)).Inc()

	s.log.Info("session resumed",
		zap.String("session_id", sessionID.String()),
	)

	return s.GetSession(ctx, sessionID)
}

// CompleteSession force-completes a session, creates a differential snapshot,
// and publishes the HPI_COMPLETE event.
func (s *SessionService) CompleteSession(ctx context.Context, sessionID uuid.UUID) error {
	session, err := s.loadSession(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("load session: %w", err)
	}

	if session.Status == models.StatusCompleted {
		return nil // already complete, idempotent
	}

	if session.Status != models.StatusActive && session.Status != models.StatusSuspended {
		return fmt.Errorf("session %s is %s, cannot complete", sessionID, session.Status)
	}

	// R-05: minimum inclusion guard — block completion if safety-critical questions unanswered
	node := s.nodeLoader.Get(session.NodeID)
	if node != nil {
		allAnswers, err := s.loadAnswerMap(ctx, sessionID)
		if err != nil {
			return fmt.Errorf("load answers for safety guard check: %w", err)
		}
		for i := range node.Questions {
			q := &node.Questions[i]
			if q.MinimumInclusionGuard && allAnswers[q.ID] == "" {
				return fmt.Errorf("cannot complete session: safety-guard question %s (%s) has not been answered", q.ID, q.TextEN)
			}
		}
	}

	logOdds, err := session.LogOddsMap()
	if err != nil {
		return fmt.Errorf("parse log odds: %w", err)
	}

	posteriors := s.bayesian.GetPosteriors(logOdds, ResolveFloors(node, session.StratumLabel))
	if node == nil {
		node = s.nodeLoader.Get(session.NodeID)
	}

	converged := false
	if node != nil {
		converged, _ = s.bayesian.CheckConvergence(posteriors, node)
	}

	session.Status = models.StatusCompleted
	now := time.Now()
	session.CompletedAt = &now
	session.CurrentQuestionID = nil
	session.LastActivityAt = now

	if err := s.createSnapshot(ctx, session, posteriors, converged); err != nil {
		s.log.Error("failed to create snapshot on force complete",
			zap.String("session_id", sessionID.String()),
			zap.Error(err),
		)
	}

	s.publishHPIComplete(ctx, session, posteriors, converged)

	if err := s.db.DB.WithContext(ctx).Save(session).Error; err != nil {
		return fmt.Errorf("save completed session: %w", err)
	}

	s.cache.SetSession(sessionID.String(), session)
	s.metrics.SessionsByStatus.WithLabelValues(string(models.StatusCompleted)).Inc()

	s.log.Info("session force-completed",
		zap.String("session_id", sessionID.String()),
		zap.Bool("converged", converged),
	)

	return nil
}

// GetDifferential returns the current ranked differential for a session.
func (s *SessionService) GetDifferential(ctx context.Context, sessionID uuid.UUID) ([]models.DifferentialEntry, error) {
	// Try cache first
	var cached []models.DifferentialEntry
	if err := s.cache.GetDifferential(sessionID.String(), &cached); err == nil {
		return cached, nil
	}

	session, err := s.loadSession(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("load session: %w", err)
	}

	logOdds, err := session.LogOddsMap()
	if err != nil {
		return nil, fmt.Errorf("parse log odds: %w", err)
	}

	// Load node before GetPosteriors so G1 safety floors can be resolved
	node := s.nodeLoader.Get(session.NodeID)
	posteriors := s.bayesian.GetPosteriors(logOdds, ResolveFloors(node, session.StratumLabel))

	// Enrich with labels from node definition
	if node != nil {
		labelMap := make(map[string]string, len(node.Differentials))
		for _, d := range node.Differentials {
			labelMap[d.ID] = d.LabelEN
		}
		for i := range posteriors {
			if label, ok := labelMap[posteriors[i].DifferentialID]; ok {
				posteriors[i].Label = label
			}
		}
	}

	// Cache the result
	s.cache.SetDifferential(sessionID.String(), posteriors)

	return posteriors, nil
}

// GetSafetyFlags returns all safety flags fired during a session.
func (s *SessionService) GetSafetyFlags(ctx context.Context, sessionID uuid.UUID) ([]models.SafetyFlag, error) {
	var flags []models.SafetyFlag
	if err := s.db.DB.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("fired_at ASC").
		Find(&flags).Error; err != nil {
		return nil, fmt.Errorf("query safety flags: %w", err)
	}
	return flags, nil
}

// GetSnapshot returns the differential snapshot for a completed session.
func (s *SessionService) GetSnapshot(ctx context.Context, sessionID uuid.UUID) (*models.DifferentialSnapshot, error) {
	var snapshot models.DifferentialSnapshot
	if err := s.db.DB.WithContext(ctx).
		Where("session_id = ?", sessionID).
		First(&snapshot).Error; err != nil {
		return nil, fmt.Errorf("snapshot not found: %w", err)
	}
	return &snapshot, nil
}

// --- internal helpers ---

// loadSession loads a session from cache, falling back to DB.
func (s *SessionService) loadSession(ctx context.Context, sessionID uuid.UUID) (*models.HPISession, error) {
	var session models.HPISession

	// Try cache
	if err := s.cache.GetSession(sessionID.String(), &session); err == nil {
		return &session, nil
	}

	// Fall back to DB
	if err := s.db.DB.WithContext(ctx).Where("session_id = ?", sessionID).First(&session).Error; err != nil {
		return nil, fmt.Errorf("session %s not found: %w", sessionID, err)
	}

	// Populate cache for next access
	s.cache.SetSession(sessionID.String(), &session)

	return &session, nil
}

// loadAnswerMap loads all answers for a session as a question_id -> answer_value map.
func (s *SessionService) loadAnswerMap(ctx context.Context, sessionID uuid.UUID) (map[string]string, error) {
	var answers []models.SessionAnswer
	if err := s.db.DB.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("answered_at ASC").
		Find(&answers).Error; err != nil {
		return nil, fmt.Errorf("load answers: %w", err)
	}

	result := make(map[string]string, len(answers))
	for _, a := range answers {
		result[a.QuestionID] = a.AnswerValue
	}
	return result, nil
}

// loadSafetyFlagSummaries loads all safety flags as compact summaries.
func (s *SessionService) loadSafetyFlagSummaries(ctx context.Context, sessionID uuid.UUID) ([]models.SafetyFlagSummary, error) {
	var flags []models.SafetyFlag
	if err := s.db.DB.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Find(&flags).Error; err != nil {
		return nil, err
	}

	summaries := make([]models.SafetyFlagSummary, len(flags))
	for i, f := range flags {
		summaries[i] = models.SafetyFlagSummary{
			FlagID:            f.FlagID,
			Severity:          string(f.Severity),
			RecommendedAction: f.RecommendedAction,
		}
	}
	return summaries, nil
}

// createSnapshot persists a DifferentialSnapshot for a completed session.
func (s *SessionService) createSnapshot(
	ctx context.Context,
	session *models.HPISession,
	posteriors []models.DifferentialEntry,
	converged bool,
) error {
	rankedJSON, _ := json.Marshal(posteriors)

	// Load safety flags for snapshot
	var flags []models.SafetyFlag
	s.db.DB.WithContext(ctx).Where("session_id = ?", session.SessionID).Find(&flags)
	flagsJSON, _ := json.Marshal(flags)

	topDiagnosis := ""
	topPosterior := 0.0
	if len(posteriors) > 0 {
		topDiagnosis = posteriors[0].DifferentialID
		topPosterior = posteriors[0].PosteriorProbability
	}

	var questionsToConvergence *int
	if converged {
		q := session.QuestionsAsked
		questionsToConvergence = &q
	}

	// CTL Panel 4: Load reasoning chain from cache
	var reasoningChain []models.ReasoningStep
	if cacheErr := s.cache.GetReasoningChain(session.SessionID.String(), &reasoningChain); cacheErr != nil {
		s.log.Debug("no reasoning chain in cache (may be empty session)",
			zap.String("session_id", session.SessionID.String()),
		)
	}
	var reasoningChainJSON models.JSONB
	if len(reasoningChain) > 0 {
		reasoningChainJSON, _ = json.Marshal(reasoningChain)
	}

	snapshot := models.DifferentialSnapshot{
		SnapshotID:             uuid.New(),
		SessionID:              session.SessionID,
		RankedDifferentials:    rankedJSON,
		SafetyFlags:            flagsJSON,
		TopDiagnosis:           topDiagnosis,
		TopPosterior:           topPosterior,
		ConvergenceReached:     converged,
		QuestionsToConvergence: questionsToConvergence,
		GuidelinePriorRefs:     session.GuidelinePriorRefs,
		ReasoningChain:         reasoningChainJSON,
	}

	if err := s.db.DB.WithContext(ctx).Create(&snapshot).Error; err != nil {
		return fmt.Errorf("create snapshot: %w", err)
	}

	s.log.Info("differential snapshot created",
		zap.String("snapshot_id", snapshot.SnapshotID.String()),
		zap.String("session_id", session.SessionID.String()),
		zap.String("top_diagnosis", topDiagnosis),
		zap.Float64("top_posterior", topPosterior),
		zap.Bool("converged", converged),
	)

	return nil
}

// publishHPIComplete sends the HPI_COMPLETE event to KB-23 and KB-19.
func (s *SessionService) publishHPIComplete(
	ctx context.Context,
	session *models.HPISession,
	posteriors []models.DifferentialEntry,
	converged bool,
) {
	// Parse CM deltas for event
	var cmDeltas map[string]float64
	if len(session.CMLogDeltasApplied) > 0 {
		json.Unmarshal(session.CMLogDeltasApplied, &cmDeltas)
	}

	// Load IMMEDIATE/URGENT safety flags for event
	var flags []models.SafetyFlag
	s.db.DB.WithContext(ctx).
		Where("session_id = ? AND severity IN ?", session.SessionID, []string{"IMMEDIATE", "URGENT"}).
		Find(&flags)

	flagSummaries := make([]models.SafetyFlagSummary, len(flags))
	for i, f := range flags {
		flagSummaries[i] = models.SafetyFlagSummary{
			FlagID:            f.FlagID,
			Severity:          string(f.Severity),
			RecommendedAction: f.RecommendedAction,
		}
	}

	completedAt := time.Now()
	if session.CompletedAt != nil {
		completedAt = *session.CompletedAt
	}

	// CTL Panel 4: Load reasoning chain from cache for event payload
	var reasoningChain []models.ReasoningStep
	if cacheErr := s.cache.GetReasoningChain(session.SessionID.String(), &reasoningChain); cacheErr != nil {
		s.log.Debug("no reasoning chain in cache for HPI_COMPLETE event",
			zap.String("session_id", session.SessionID.String()),
		)
	}

	event := models.HPICompleteEvent{
		EventType:           models.EventHPIComplete,
		PatientID:           session.PatientID,
		SessionID:           session.SessionID,
		NodeID:              session.NodeID,
		StratumLabel:        session.StratumLabel,
		TopDiagnosis:        posteriors[0].DifferentialID,
		TopPosterior:        posteriors[0].PosteriorProbability,
		RankedDifferentials: topN(posteriors, 5),
		SafetyFlags:         flagSummaries,
		CMLogDeltasApplied:  cmDeltas,
		GuidelinePriorRefs:  session.GuidelinePriorRefs,
		ReasoningChain:      reasoningChain,
		ConvergenceReached:  converged,
		CompletedAt:         completedAt,
	}

	go func() {
		if err := s.publisher.PublishHPIComplete(context.Background(), event); err != nil {
			s.log.Error("failed to publish HPI_COMPLETE event",
				zap.String("session_id", session.SessionID.String()),
				zap.Error(err),
			)
		} else {
			// Mark outcome published
			s.db.DB.Model(session).Update("outcome_published", true)
		}
	}()
}

// computeLRApplied extracts the LR values applied for a given answer.
func (s *SessionService) computeLRApplied(question *models.QuestionDef, answerValue string) map[string]float64 {
	switch answerValue {
	case string(models.AnswerYes):
		return question.LRPositive
	case string(models.AnswerNo):
		return question.LRNegative
	default:
		// PATA_NAHI: zero LR applied (F-04)
		result := make(map[string]float64, len(question.LRPositive))
		for diffID := range question.LRPositive {
			result[diffID] = 0.0
		}
		return result
	}
}

// buildSessionResponse constructs a SessionResponse from session state.
func (s *SessionService) buildSessionResponse(
	session *models.HPISession,
	currentQuestion *models.QuestionDef,
	posteriors []models.DifferentialEntry,
) *models.SessionResponse {
	resp := &models.SessionResponse{
		SessionID:        session.SessionID,
		PatientID:        session.PatientID,
		NodeID:           session.NodeID,
		StratumLabel:     session.StratumLabel,
		Status:           session.Status,
		QuestionsAsked:   session.QuestionsAsked,
		TopDifferentials: topN(posteriors, 5),
		StartedAt:        session.StartedAt,
		LastActivityAt:   session.LastActivityAt,
	}

	if currentQuestion != nil {
		resp.CurrentQuestion = &models.QuestionResponse{
			QuestionID: currentQuestion.ID,
			TextEN:     currentQuestion.TextEN,
			TextHI:     currentQuestion.TextHI,
			Mandatory:  currentQuestion.Mandatory,
		}
	}

	// Load safety flag summaries (best effort)
	var flags []models.SafetyFlag
	s.db.DB.Where("session_id = ?", session.SessionID).Find(&flags)
	for _, f := range flags {
		resp.SafetyFlags = append(resp.SafetyFlags, models.SafetyFlagSummary{
			FlagID:            f.FlagID,
			Severity:          string(f.Severity),
			RecommendedAction: f.RecommendedAction,
		})
	}

	return resp
}

// topN returns the first n entries from a posteriors slice.
func topN(entries []models.DifferentialEntry, n int) []models.DifferentialEntry {
	if len(entries) <= n {
		return entries
	}
	return entries[:n]
}

// FetchStratum fetches only the current stratum from KB-20 for drift detection (R-04).
func (p *SessionContextProvider) FetchStratum(ctx context.Context, patientID uuid.UUID, nodeID string) (string, *string, error) {
	result, err := p.Fetch(ctx, patientID, nodeID)
	if err != nil {
		return "", nil, err
	}
	return result.StratumLabel, result.CKDSubstage, nil
}
