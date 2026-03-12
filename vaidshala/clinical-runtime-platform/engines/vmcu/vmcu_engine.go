package vmcu

import (
	"context"
	"fmt"
	"time"

	"vaidshala/clinical-runtime-platform/engines/vmcu/arbiter"
	"vaidshala/clinical-runtime-platform/engines/vmcu/autonomy"
	"vaidshala/clinical-runtime-platform/engines/vmcu/cache"
	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_b"
	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_c"
	"vaidshala/clinical-runtime-platform/engines/vmcu/events"
	"vaidshala/clinical-runtime-platform/engines/vmcu/metabolic"
	"vaidshala/clinical-runtime-platform/engines/vmcu/metrics"
	"vaidshala/clinical-runtime-platform/engines/vmcu/titration"
	"vaidshala/clinical-runtime-platform/engines/vmcu/trace"
	vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"
)

// VMCUEngine orchestrates the three-channel safety architecture.
//
// For each titration cycle:
//  1. Resolve patient data from cache (or use caller-provided input)
//  2. Evaluate Channel A (MCU_GATE from KB-23 cache)
//  3. Evaluate Channel B (PhysiologySafetyMonitor on raw labs)
//  4. Evaluate Channel C (ProtocolGuard on compiled rules)
//  5. Arbitrate (1oo3 veto — most restrictive wins)
//  6. Check integrator state (freeze/resume)
//  7. Check cooldown tracker (minimum dose interval)
//  8. Apply re-entry protocol constraints
//  9. Apply rate limiter (post-resume reduction)
// 10. Optionally enrich via MetabolicEngine (KB-24)
// 11. Compute dose via TitrationEngine
// 12. Record SafetyTrace + async persistence
// 13. Publish events + handle HOLD_DATA response
// 14. Record metrics
type VMCUEngine struct {
	// Core pipeline (always present)
	channelB  *channel_b.PhysiologySafetyMonitor
	channelC  *channel_c.ProtocolGuard
	titration *titration.TitrationEngine
	tracer    *trace.TraceWriter

	// Phase 8 — Titration safety commitments (pure in-process, always wired)
	integrators map[string]*titration.Integrator   // patientID → integrator
	rateLimiter *titration.RateLimiter
	reentry     map[string]*titration.ReentryProtocol // patientID → protocol
	cooldown    *titration.CooldownTracker

	// KB-24 Metabolic enrichment (pure in-process, always wired)
	metabolicEngine *metabolic.Engine

	// Autonomy limits (always wired — enforces dose change constraints)
	autonomyLimits *autonomy.AutonomyLimits

	// Deprescribing manager (always wired — tracks active deprescribing plans)
	deprescribing *titration.DeprescribingManager

	// Infrastructure-dependent (optional, injected by runtime layer)
	safetyCache     *cache.SafetyCache          // nil = caller provides input directly
	asyncTracer     *trace.AsyncTraceWriter      // nil = traces stay in-memory only
	holdResponder   *events.HoldDataResponder    // nil = no KB-20 anomaly flagging
	eventPublisher  events.EventPublisher        // nil = no outbound KB-19 events
	recorder        metrics.Recorder             // never nil (defaults to NoopRecorder)
}

// VMCUConfig holds V-MCU engine configuration.
type VMCUConfig struct {
	ProtocolRulesPath string
	MaxDoseDeltaPct   float64
	PhysioConfig      channel_b.PhysioConfig
	CooldownConfig    titration.CooldownConfig
	AutonomyLimits    *autonomy.AutonomyLimits // nil = use defaults

	// Optional infrastructure dependencies (nil = disabled)
	SafetyCache    *cache.SafetyCache
	TraceStore     trace.TraceStore         // if set, enables async persistence
	HoldResponder  *events.HoldDataResponder
	EventPublisher events.EventPublisher
	Recorder       metrics.Recorder
}

// DefaultVMCUConfig returns production-safe defaults.
func DefaultVMCUConfig() VMCUConfig {
	return VMCUConfig{
		ProtocolRulesPath: "./engines/vmcu/protocol_rules.yaml",
		MaxDoseDeltaPct:   20.0,
		PhysioConfig:      channel_b.DefaultPhysioConfig(),
		CooldownConfig:    titration.DefaultCooldownConfig(),
	}
}

// NewVMCUEngine creates and initializes the V-MCU engine.
// Loads Channel C rules from YAML at construction time.
// All pure-logic safety components are wired automatically.
// Infrastructure-dependent components are wired only if provided in config.
func NewVMCUEngine(cfg VMCUConfig) (*VMCUEngine, error) {
	chB := channel_b.NewPhysiologySafetyMonitor(cfg.PhysioConfig)

	chC, err := channel_c.LoadRules(cfg.ProtocolRulesPath)
	if err != nil {
		return nil, fmt.Errorf("channel C init failed: %w", err)
	}

	rec := cfg.Recorder
	if rec == nil {
		rec = metrics.NoopRecorder{}
	}

	autoLimits := cfg.AutonomyLimits
	if autoLimits == nil {
		autoLimits = autonomy.DefaultAutonomyLimits()
	}

	eng := &VMCUEngine{
		// Core pipeline
		channelB:  chB,
		channelC:  chC,
		titration: titration.NewTitrationEngine(cfg.MaxDoseDeltaPct),
		tracer:    trace.NewTraceWriter(),

		// Phase 8 pure-logic (always wired)
		integrators:     make(map[string]*titration.Integrator),
		rateLimiter:     titration.NewRateLimiter(),
		reentry:         make(map[string]*titration.ReentryProtocol),
		cooldown:        titration.NewCooldownTracker(cfg.CooldownConfig),
		metabolicEngine: metabolic.NewEngine(),

		// Autonomy and deprescribing (always wired)
		autonomyLimits: autoLimits,
		deprescribing:  titration.NewDeprescribingManager(),

		// Infrastructure (optional)
		safetyCache:    cfg.SafetyCache,
		holdResponder:  cfg.HoldResponder,
		eventPublisher: cfg.EventPublisher,
		recorder:       rec,
	}

	// Wire async trace persistence if TraceStore provided
	if cfg.TraceStore != nil {
		eng.asyncTracer = trace.NewAsyncTraceWriter(cfg.TraceStore, 1000)
	}

	return eng, nil
}

// ProtocolRulesHash returns the SHA-256 of the loaded protocol_rules.yaml.
func (e *VMCUEngine) ProtocolRulesHash() string {
	return e.channelC.RulesHash()
}

// TitrationCycleInput contains everything needed for one titration cycle.
type TitrationCycleInput struct {
	PatientID string

	// Channel A: pre-fetched from KB-23 cache
	ChannelAResult vt.ChannelAResult

	// Channel B: raw lab data from KB-20 cache
	// If nil AND SafetyCache is wired, engine reads from cache.
	RawLabs *channel_b.RawPatientData

	// Channel C: titration context assembled by caller
	// If nil AND SafetyCache is wired, engine assembles from cache.
	TitrationContext *channel_c.TitrationContext

	// Proposed dose change
	CurrentDose   float64
	ProposedDelta float64

	// Medication class for cooldown tracking (optional)
	MedClass titration.MedicationClass

	// Metabolic enrichment input (optional — if nil, metabolic engine uses defaults)
	MetabolicInput *metabolic.MetabolicInput
}

// RunCycle executes one complete titration cycle through all three channels.
func (e *VMCUEngine) RunCycle(input TitrationCycleInput) (*vt.TitrationCycleResult, *trace.SafetyTrace) {
	cycleStart := time.Now()

	// ── Resolve from cache if available ──
	e.resolveFromCache(&input)

	// ── Channel A: already resolved by caller ──
	chAStart := time.Now()
	chAResult := input.ChannelAResult
	e.recorder.RecordChannelLatency("A", time.Since(chAStart))

	// ── Channel B: evaluate raw physiology ──
	// If deprescribing is active for this patient, widen glucose thresholds.
	chBStart := time.Now()
	chBOpts := channel_b.EvaluateOptions{
		DeprescribingActive: e.deprescribing.IsDeprescribing(input.PatientID),
	}
	physioResult := e.channelB.EvaluateWithOptions(input.RawLabs, chBOpts)
	chBResult := vt.ChannelBResult{
		Gate:       mapPhysioGate(physioResult.Gate),
		RuleFired:  physioResult.RuleFired,
		RawValues:  physioResult.RawValues,
		IsAnomaly:  physioResult.IsAnomaly,
		AnomalyLab: physioResult.AnomalyLab,
	}
	e.recorder.RecordChannelLatency("B", time.Since(chBStart))

	// ── KB22 Trigger: async route to KB-22 HPI via KB-19 ──
	// When Channel B sentinels (e.g., B-16 irregular HR) fire a KB22_TRIGGER,
	// emit the event asynchronously. This does NOT affect the current V-MCU
	// cycle's gate signal — it is fire-and-forget.
	if len(physioResult.KB22Triggers) > 0 && e.eventPublisher != nil {
		for _, trigger := range physioResult.KB22Triggers {
			triggerCopy := trigger // capture for goroutine
			go func() {
				_ = events.PublishKB22Trigger(
					context.Background(),
					e.eventPublisher,
					events.KB22TriggerEvent{
						SentinelID:  triggerCopy.SentinelID,
						PatientID:   input.PatientID,
						HPINodeID:   triggerCopy.HPINodeID,
						TriggerData: triggerCopy.Data,
					},
				)
			}()
		}
	}

	// If Channel B fires HALT during active deprescribing, pause the plan
	// at the current reduced dose — do NOT revert to original higher dose.
	if chBOpts.DeprescribingActive && chBResult.Gate.IsBlocking() {
		if plan := e.deprescribing.GetPlan(input.PatientID, string(input.MedClass)); plan != nil {
			_ = e.deprescribing.PausePlan(input.PatientID, string(input.MedClass),
				fmt.Sprintf("Channel B %s fired rule %s during deprescribing", chBResult.Gate, chBResult.RuleFired))
		}
	}

	// Pass Channel B findings into Channel C context
	if input.TitrationContext != nil {
		// B-03 fires as "B-03" (true AKI) or "B-03-RAAS-SUPPRESSED" (expected RAAS rise)
		input.TitrationContext.AKIDetected = physioResult.RuleFired == "B-03"
		input.TitrationContext.ActiveHypoglycaemia = physioResult.RuleFired == "B-01"

		// If B-03 was RAAS-suppressed, inform Channel C for PG-14 audit trail
		if physioResult.RuleFired == "B-03-RAAS-SUPPRESSED" {
			input.TitrationContext.RAASCreatinineTolerant = true
		}
	}

	// ── Channel C: evaluate protocol rules ──
	chCStart := time.Now()
	var chCResult vt.ChannelCResult
	if input.TitrationContext != nil {
		protoResult := e.channelC.Evaluate(input.TitrationContext)
		chCResult = vt.ChannelCResult{
			Gate:         mapProtocolGate(protoResult.Gate),
			RuleID:       protoResult.RuleID,
			RuleVersion:  protoResult.RuleVersion,
			GuidelineRef: protoResult.GuidelineRef,
		}
	} else {
		chCResult = vt.ChannelCResult{
			Gate:        vt.GateClear,
			RuleVersion: e.channelC.RulesHash(),
		}
	}
	e.recorder.RecordChannelLatency("C", time.Since(chCStart))

	// ── Arbiter: 1oo3 veto ──
	arbiterStart := time.Now()
	arbiterInput := vt.ArbiterInput{
		MCUGate:      chAResult.Gate,
		PhysioGate:   chBResult.Gate,
		ProtocolGate: chCResult.Gate,
	}
	arbiterOutput := arbiter.Arbitrate(arbiterInput)
	e.recorder.RecordArbiterLatency(time.Since(arbiterStart))

	// Record blocked gates as metrics
	if arbiterOutput.FinalGate.IsBlocking() {
		e.recorder.RecordGateBlocked(arbiterOutput.DominantChannel, string(arbiterOutput.FinalGate))
	}
	if chBResult.IsAnomaly {
		e.recorder.RecordHoldDataTriggered(chBResult.RuleFired)
	}

	// ── BP-status-dependent titration velocity adjustment (Proposal §8.1) ──
	// BP conditions modify insulin titration aggressiveness. Applied after
	// arbiter but before dose computation. URGENCY and HYPOTENSIVE override
	// the arbiter gate to halt/pause respectively.
	bpVelocityMultiplier := 1.0
	if bpStatus := classifyBPStatus(input.RawLabs); bpStatus != "" {
		switch bpStatus {
		case "URGENCY":
			// Clinical emergency — halt all titration.
			// Override arbiter gate to HALT regardless of other channels.
			arbiterOutput.FinalGate = vt.GateHalt
			arbiterOutput.DominantChannel = "BP"
			arbiterOutput.RationaleCode = "BP_URGENCY:SBP>=180"
		case "SEVERE":
			// Cardiovascular stress — reduce titration velocity by 30%
			bpVelocityMultiplier = 0.70
		case "HYPOTENSIVE":
			// Over-treated or haemodynamic instability — pause dose increases.
			// Only override if arbiter hasn't already set a more restrictive gate.
			if !arbiterOutput.FinalGate.IsBlocking() {
				arbiterOutput.FinalGate = vt.GatePause
				arbiterOutput.DominantChannel = "BP"
				arbiterOutput.RationaleCode = "BP_HYPOTENSIVE:SBP<90"
			}
		case "ABOVE_TARGET":
			// Suboptimal BP control — reduce velocity by 20%
			bpVelocityMultiplier = 0.80
		}
	}

	// ── Integrator: check freeze/resume state ──
	ig := e.getIntegrator(input.PatientID, input.CurrentDose)

	if arbiterOutput.FinalGate.IsBlocking() {
		// Freeze the integrator — dose locked at current value
		if !ig.IsFrozen() {
			ig.Freeze(input.CurrentDose, fmt.Sprintf("CH_%s:%s", arbiterOutput.DominantChannel, arbiterOutput.FinalGate))
		}
	} else if ig.IsFrozen() {
		// Arbiter cleared — resume from frozen dose
		pauseHours := ig.PauseHours()
		ig.Resume()

		// Activate post-resume rate limiter
		e.rateLimiter.ActivatePostResume(pauseHours)

		// Activate re-entry protocol for this patient
		rp := e.getReentryProtocol(input.PatientID)
		rp.Activate()

		// Override current dose with frozen dose (no drift)
		input.CurrentDose = ig.FrozenDose()
	}

	// ── Cooldown: check minimum dose interval ──
	cooldownBlocked := false
	var cooldownRemaining time.Duration
	if input.MedClass != "" && !arbiterOutput.FinalGate.IsBlocking() {
		if e.cooldown.IsOnCooldown(input.PatientID, input.MedClass) {
			cooldownBlocked = true
			cooldownRemaining = e.cooldown.RemainingCooldown(input.PatientID, input.MedClass)
		}
	}

	// ── Re-entry protocol: check if dose changes are allowed ──
	rp := e.getReentryProtocol(input.PatientID)
	reentryBlocked := rp.IsActive() && !rp.AllowsDoseChange()
	reentryMultiplier := 1.0
	if rp.IsActive() && rp.AllowsDoseChange() {
		reentryMultiplier = rp.MaxDeltaMultiplier()
	}

	// ── Rate limiter: post-resume reduction ──
	effectiveGainFactor := chAResult.GainFactor
	if effectiveGainFactor <= 0 {
		effectiveGainFactor = 1.0
	}

	// ── Metabolic Engine (KB-24): enrich gain factor ──
	if input.MetabolicInput != nil {
		metaOutput := e.metabolicEngine.Classify(*input.MetabolicInput)
		effectiveGainFactor *= metaOutput.SuggestedGainAdj
	}

	// Apply rate limiter to effective gain
	if e.rateLimiter.IsLimited() {
		effectiveGainFactor = e.rateLimiter.ApplyLimit(effectiveGainFactor)
	}

	// Apply re-entry multiplier (only in CONSERVATIVE phase, not MONITORING)
	effectiveGainFactor *= reentryMultiplier

	// Beta-blocker gain factor floor (Proposal §8.1): patients on beta-blockers
	// cannot self-detect early hypoglycaemia. Conservative titration is mandatory
	// regardless of adherence score. Gain factor must not drop below 0.50 —
	// ensures minimum titration conservatism.
	if input.RawLabs != nil && input.RawLabs.BetaBlockerActive && effectiveGainFactor < 0.50 {
		effectiveGainFactor = 0.50
	}

	// Apply BP-status velocity multiplier (computed above, after arbiter)
	effectiveGainFactor *= bpVelocityMultiplier

	// ── Deprescribing escalation suppression ──
	// If a drug class is being deprescribed, suppress any proposed escalation
	// for that class. This prevents the titration engine from fighting the
	// clinician's deliberate step-down by re-escalating the same medication.
	deprescribingSuppressed := false
	if input.ProposedDelta > 0 && input.MedClass != "" {
		depCtx := e.buildDeprescribingContext(input.PatientID, string(input.MedClass))
		if depCtx.Active {
			proposed := titration.DoseChange{
				DrugClass: depCtx.DrugClass,
				Direction: "UP",
				DeltaMg:   input.ProposedDelta,
			}
			if titration.ShouldSuppressEscalation(proposed, depCtx) {
				deprescribingSuppressed = true
			}
		}
	}

	// ── Titration: compute dose (or block) ──
	var doseResult *titration.DoseResult
	if deprescribingSuppressed {
		// Suppress escalation: return CLEAR with no dose change
		doseResult = &titration.DoseResult{
			Blocked:   true,
			BlockedBy: fmt.Sprintf("DEPRESCRIBING_SUPPRESSED:%s", input.MedClass),
		}
	} else if reentryBlocked {
		// Re-entry MONITORING phase: observe only, no dose changes
		doseResult = &titration.DoseResult{
			Blocked:   true,
			BlockedBy: fmt.Sprintf("REENTRY:%s", rp.Phase()),
		}
	} else if cooldownBlocked {
		doseResult = &titration.DoseResult{
			Blocked:   true,
			BlockedBy: fmt.Sprintf("COOLDOWN:%s:%.0fh_remaining", input.MedClass, cooldownRemaining.Hours()),
		}
	} else {
		doseResult = e.titration.ComputeDose(
			arbiterOutput,
			input.CurrentDose,
			input.ProposedDelta,
			effectiveGainFactor,
		)
	}

	// ── Autonomy limit check ──
	// Prevents V-MCU from exceeding single-step, cumulative, or absolute
	// dose ceilings without explicit physician confirmation.
	if !doseResult.Blocked && doseResult.NewDose != input.CurrentDose {
		limitResult := e.autonomyLimits.CheckLimit(
			input.PatientID, input.CurrentDose, doseResult.NewDose, string(input.MedClass),
		)
		if !limitResult.Allowed {
			doseResult = &titration.DoseResult{
				Blocked:   true,
				BlockedBy: fmt.Sprintf("AUTONOMY:%s", limitResult.LimitID),
			}
		}
	}

	// ── Post-dose bookkeeping ──
	if !doseResult.Blocked && input.MedClass != "" {
		// Record dose change for cooldown tracking
		e.cooldown.RecordDoseChange(titration.DoseEvent{
			PatientID: input.PatientID,
			MedClass:  input.MedClass,
			AppliedAt: time.Now(),
			DoseDelta: doseResult.DoseDelta,
		})

		// Advance re-entry protocol cycle
		if rp.IsActive() {
			rp.AdvanceCycle()
		}
	}

	// ── Assemble result ──
	cycleResult := &vt.TitrationCycleResult{
		PatientID: input.PatientID,
		Timestamp: cycleStart,
		ChannelA:  chAResult,
		ChannelB:  chBResult,
		ChannelC:  chCResult,
		Arbiter:   arbiterOutput,
	}

	if doseResult.Blocked {
		cycleResult.BlockedBy = doseResult.BlockedBy
	} else {
		cycleResult.DoseApplied = &doseResult.NewDose
		cycleResult.DoseDelta = &doseResult.DoseDelta
	}

	// ── SafetyTrace: append-only audit record ──
	traceStart := time.Now()
	traceRecord := e.tracer.Record(cycleResult)
	e.recorder.RecordTraceWriteLatency(time.Since(traceStart))

	// Async persistence (if wired)
	if e.asyncTracer != nil {
		e.asyncTracer.Enqueue(traceRecord)
	}

	// ── HOLD_DATA response flow (if wired) ──
	if chBResult.IsAnomaly && e.holdResponder != nil {
		// Fire-and-forget in goroutine to stay within latency budget
		go func() {
			_ = e.holdResponder.Respond(
				context.Background(),
				input.PatientID,
				"", // labID resolved by runtime layer
				chBResult.RuleFired,
				chBResult.AnomalyLab,
			)
		}()
	}

	// ── Publish TITRATION_COMPLETED event (if wired) ──
	if e.eventPublisher != nil && !doseResult.Blocked {
		go func() {
			_ = e.eventPublisher.Publish(context.Background(), events.Event{
				Type:      events.EventTitrationCompleted,
				PatientID: input.PatientID,
				Source:    "V-MCU",
				Payload: map[string]interface{}{
					"new_dose":   doseResult.NewDose,
					"dose_delta": doseResult.DoseDelta,
					"final_gate": string(arbiterOutput.FinalGate),
				},
			})
		}()
	}

	// ── Metrics ──
	e.recorder.RecordCycleCompleted(string(arbiterOutput.FinalGate))

	return cycleResult, &traceRecord
}

// FlushTraces returns all pending SafetyTrace records and clears the buffer.
func (e *VMCUEngine) FlushTraces() []trace.SafetyTrace {
	return e.tracer.Flush()
}

// Stop gracefully shuts down the engine (flushes async traces).
func (e *VMCUEngine) Stop() {
	if e.asyncTracer != nil {
		e.asyncTracer.Stop()
	}
}

// GetIntegratorState returns the integrator state for a patient (for external inspection).
func (e *VMCUEngine) GetIntegratorState(patientID string) (titration.IntegratorState, float64) {
	ig, ok := e.integrators[patientID]
	if !ok {
		return titration.IntegratorActive, 0
	}
	return ig.State(), ig.FrozenDose()
}

// GetReentryPhase returns the current re-entry phase for a patient.
func (e *VMCUEngine) GetReentryPhase(patientID string) titration.ReentryPhase {
	rp, ok := e.reentry[patientID]
	if !ok {
		return titration.ReentryNone
	}
	return rp.Phase()
}

// IsOnCooldown checks if a patient's medication class is within cooldown.
func (e *VMCUEngine) IsOnCooldown(patientID string, medClass titration.MedicationClass) bool {
	return e.cooldown.IsOnCooldown(patientID, medClass)
}

// ── Internal helpers ──

// getIntegrator returns or creates an integrator for a patient.
func (e *VMCUEngine) getIntegrator(patientID string, currentDose float64) *titration.Integrator {
	ig, ok := e.integrators[patientID]
	if !ok {
		ig = titration.NewIntegrator(currentDose)
		e.integrators[patientID] = ig
	}
	return ig
}

// getReentryProtocol returns or creates a re-entry protocol for a patient.
func (e *VMCUEngine) getReentryProtocol(patientID string) *titration.ReentryProtocol {
	rp, ok := e.reentry[patientID]
	if !ok {
		rp = titration.NewReentryProtocol()
		e.reentry[patientID] = rp
	}
	return rp
}

// buildDeprescribingContext constructs a DeprescribingContext for the given
// patient and drug class by querying the DeprescribingManager.
func (e *VMCUEngine) buildDeprescribingContext(patientID, drugClass string) titration.DeprescribingContext {
	plan := e.deprescribing.GetPlan(patientID, drugClass)
	if plan == nil || plan.State != titration.DeprescribingActive {
		return titration.DeprescribingContext{Active: false}
	}
	phase := "DOSE_REDUCTION"
	if plan.CurrentDose <= plan.TargetDose {
		phase = "MONITORING"
	}
	return titration.DeprescribingContext{
		Active:            true,
		DrugClass:         plan.DrugClass,
		Phase:             phase,
		MonitoringCadence: "WEEKLY",
	}
}

// resolveFromCache populates input fields from SafetyCache if available.
func (e *VMCUEngine) resolveFromCache(input *TitrationCycleInput) {
	if e.safetyCache == nil {
		return
	}
	cached := e.safetyCache.Get(input.PatientID)
	if cached == nil {
		return
	}

	// Fill in RawLabs from cache if caller didn't provide
	if input.RawLabs == nil && cached.RawLabs != nil {
		input.RawLabs = cached.RawLabs
		e.recorder.RecordCacheRefresh("labs_from_cache")
	}

	// Fill in Channel A from cache if caller sent CLEAR (default/unset)
	if input.ChannelAResult.Gate == vt.GateClear && cached.MCUGate.Gate != vt.GateClear {
		input.ChannelAResult = cached.MCUGate
		e.recorder.RecordCacheRefresh("mcu_gate_from_cache")
	}

	// Fill in TitrationContext medications from cache
	if input.TitrationContext != nil && len(input.TitrationContext.ActiveMedications) == 0 {
		input.TitrationContext.ActiveMedications = cached.ActiveMedications
	}

	// Fill in HypoWithin7d from cache
	if input.TitrationContext != nil && cached.HypoWithin7d {
		input.TitrationContext.HypoglycaemiaWithin7d = true
	}

	// ── HTN co-management fields from cache ──

	// Propagate RAAS tolerance context to Channel B inputs
	if input.RawLabs != nil && cached.RAASChangeRecency != nil {
		days := cached.RAASChangeRecency.DaysSinceChange
		// RAAS creatinine tolerance applies within 14 days of ACEi/ARB change
		if days > 0 && days <= 14 {
			input.RawLabs.CreatinineRiseExplained = true
		}
		// Propagate CKD stage for J-curve (B-12) stratification
		if cached.CKDStage != "" {
			input.RawLabs.CKDStage = cached.CKDStage
		}
		// Season for seasonal BP adjustment (future use)
		if cached.Season != "" {
			input.RawLabs.Season = cached.Season
		}
	}

	// Fill sodium into RawLabs from cache
	if input.RawLabs != nil && cached.SodiumCurrent != nil {
		input.RawLabs.SodiumCurrent = cached.SodiumCurrent
	}

	// Propagate HTN composites to Channel C context
	if input.TitrationContext != nil {
		if cached.PotassiumCurrent != nil {
			input.TitrationContext.PotassiumCurrent = *cached.PotassiumCurrent
		}
		if cached.SodiumCurrent != nil {
			input.TitrationContext.SodiumCurrent = *cached.SodiumCurrent
		}
		if cached.SBPCurrent != nil {
			input.TitrationContext.SBPCurrent = *cached.SBPCurrent
		}
	}
}

// mapPhysioGate converts Channel B's local gate type to the unified GateSignal.
func mapPhysioGate(g channel_b.PhysioGate) vt.GateSignal {
	switch g {
	case channel_b.PhysioHalt:
		return vt.GateHalt
	case channel_b.PhysioHoldData:
		return vt.GateHoldData
	case channel_b.PhysioPause:
		return vt.GatePause
	case channel_b.PhysioModify:
		return vt.GateModify
	default:
		return vt.GateClear
	}
}

// mapProtocolGate converts Channel C's local gate type to the unified GateSignal.
func mapProtocolGate(g channel_c.ProtocolGate) vt.GateSignal {
	switch g {
	case channel_c.ProtoHalt:
		return vt.GateHalt
	case channel_c.ProtoPause:
		return vt.GatePause
	case channel_c.ProtoModify:
		return vt.GateModify
	default:
		return vt.GateClear
	}
}

// classifyBPStatus derives a BP status category from the patient's current SBP.
// Used by the BP-velocity modulation logic (Proposal §8.1) to adjust titration
// aggressiveness based on blood pressure conditions.
//
// Returns one of: "URGENCY", "SEVERE", "HYPOTENSIVE", "ABOVE_TARGET", or ""
// (empty string = normal BP, no velocity adjustment needed).
//
// Thresholds follow standard clinical BP classification:
//   - URGENCY:      SBP >= 180 mmHg (hypertensive emergency)
//   - SEVERE:       SBP >= 160 mmHg (stage 2 hypertension)
//   - HYPOTENSIVE:  SBP < 90 mmHg  (haemodynamic instability)
//   - ABOVE_TARGET: SBP >= 140 mmHg (stage 1 hypertension)
func classifyBPStatus(labs *channel_b.RawPatientData) string {
	if labs == nil || labs.SBPCurrent == nil {
		return ""
	}
	sbp := *labs.SBPCurrent
	switch {
	case sbp >= 180:
		return "URGENCY"
	case sbp >= 160:
		return "SEVERE"
	case sbp < 90:
		return "HYPOTENSIVE"
	case sbp >= 140:
		return "ABOVE_TARGET"
	default:
		return ""
	}
}
