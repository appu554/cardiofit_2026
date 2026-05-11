# Clinical Attention Prioritisation Engine (CAPE) — Implementation Guidelines v1.1

**Date:** May 2026
**Supersedes:** *Daily Triage Engine Implementation Guidelines v1.0* (May 2026)
**Service scope:** Service `kb-33-triage-engine` (code name preserved for continuity) — architecturally rebranded as the Clinical Attention Prioritisation Engine (CAPE) to reflect its true purpose. Populates Surface 1 (Pharmacist Worklist) specified in *Decision Packet Rendering Implementation Guidelines v1.0*
**Implementation phase:** Phase 3 of Layer 2/3 plan (Weeks 18–24)
**Builds on:** *Vaidshala v3.0 Product Proposal* §7; *Recommendation Craft Implementation Guidelines v1.0*; *Pharmacist Self-Visibility Implementation Guidelines v1.0*; *Ethical Architecture Implementation Guidelines v1.0*; *Decision Packet Rendering Implementation Guidelines v1.0*; *KB-29 Templates Implementation Guidelines v1.0*; *KB-29 Maturity Roadmap v1.0*

**Why v1.1:** The v1.0 document framed this service as "Daily Triage Engine" — a risk-stratification framing that the senior reviewer correctly identified as conceptually wrong. Two parallel reviews independently surfaced that the engine's true purpose is *attention allocation*, not risk scoring. The reframing is structurally consequential: it changes scoring logic, signal taxonomy, performance metrics, and the engine's relationship to pharmacist clinical reasoning. v1.1 incorporates the reframing and the operationally specific improvements from both reviewers.

**Reading order:** All readers should read Part 0 first — it establishes the conceptual reframing that drives every subsequent design decision. Engineering and product leads then read Parts 1–4. Clinical informatics leads read Parts 3–7. Senior consultant pharmacists read Parts 5–9 and 11. Implementers read Parts 12–16. Risk and ethics leads read Parts 9–10 and 17.

---

## Part 0 — The conceptual reframing: from risk scoring to attention allocation

The single most important architectural decision in this document is naming what the engine actually does.

The v1.0 document framed the engine as "Daily Triage Engine" — producing prioritised lists of "high-acuity residents." This framing is wrong, and the reviewers were right to reject it. Risk scoring optimises for identifying high-baseline-risk residents. Attention allocation optimises for identifying residents where the pharmacist's intervention has the highest probability of meaningfully changing trajectory today. These produce different worklists.

The reviewer's framing question captures the operational reality:

> *"I have 90 minutes — where do I create the most safety?"*

This is what experienced consultant pharmacists actually compute, implicitly, when they triage. They are not asking *who is sickest* or *who has the most medications*. They are asking *where my finite cognitive attention applied today most plausibly changes a trajectory that would otherwise drift toward harm*.

### The structural difference

A CFS-9 resident with goals of care firmly on comfort, family processing the transition, established medication regimen, no recent acute events: high baseline risk, low intervention opportunity. The risk-score worklist surfaces this resident; the attention-allocation worklist deprioritises them, because the pharmacist's intervention here is unlikely to change the trajectory the resident and family have chosen.

A CFS-6 resident with a recent fall, recent benzodiazepine PRN escalation over 5 days, sedation drift documented in nursing notes, antipsychotic review overdue: moderate baseline risk, high intervention opportunity. The attention-allocation worklist surfaces this resident clearly, because the pharmacist's intervention here has high probability of changing a trajectory that is currently drifting toward instability.

The risk-score framing produces lists that experienced pharmacists routinely override. The attention-allocation framing produces lists that match how experienced pharmacists actually think.

### What this means for the document

Every architectural decision in v1.1 is made with the attention-allocation framing in mind. The five-layer scoring architecture (Part 3) explicitly includes intervention opportunity as a first-class layer. The signal taxonomy (Part 4) is reorganised around what changes trajectories rather than what indicates baseline risk. The performance metrics (Part 10) measure intervention opportunity realisation, not just engagement. The boundary specification (Part 2.4) is sharpened to reject specific clinical action recommendations as engine output — those are the pharmacist's irreducible clinical contribution.

### The naming convention

The service code name is preserved as `kb-33-triage-engine` for engineering continuity (changing it mid-development would create confusion). The architectural concept is the **Clinical Attention Prioritisation Engine (CAPE)**. User-facing documentation, training materials, and clinical informatics references should use CAPE; engineering references can use the code name.

### The operational test

Every design decision in this document should pass the operational test:

> *"Does this design choice improve the engine's answer to: 'I have 90 minutes — where do I create the most safety?'"*

If yes, adopt. If no, reject. If unclear, defer to pilot evidence. This test discipline is what protects against scope creep, alert fatigue, and the failure modes the literature documents.

---

## Part 1 — Design philosophy

Eight principles, each grounded in the empirical evidence reviewed and the reframing established in Part 0.

**Principle 1: Attention allocation, not risk scoring.** The engine's output is "where pharmacist intervention is most likely to change trajectory today," not "who is at highest baseline risk." This distinction drives the scoring logic, the signal taxonomy, and the performance metrics. It is the document's primary architectural commitment.

**Principle 2: Recognition over prediction.** The engine surfaces residents whose substrate state contains specific signals warranting attention. It does not predict who will have an adverse event. Recognition is more accurate than prediction because it doesn't require population generalisation. PADR-EC, BADRI, GerontoNet have moderate AUROC (0.62–0.74) — recognition of substrate signals is more reliable.

**Principle 3: Trajectory primary, snapshot secondary.** "eGFR 31" tells you less than "eGFR 48 → 42 → 38 → 31 over 4 months, velocity -4.25 mL/min/month, projected 27 in 30 days." The engine computes and renders trajectories first; absolute values are context, not the headline.

**Principle 4: Multi-dimensional, never collapsed to a single number.** The literature documents single-score collapse (PAST, STORIMAP) as the primary failure mode. The engine retains five-layer scoring through the rendering pipeline. Each worklist entry shows the dimensions that drove its position. The composite is used for ordering only — never as the user-facing primary number. Reviewer 1's framing is adopted: *"Output should NOT be Risk Score = 92. Terrible UX."*

**Principle 5: Surface signals, not clinical actions.** The engine surfaces what's there ("lithium level 117 days old, eGFR declined 35%, new confusion documented"). The integration to specific actions ("therefore order lithium level urgently and consider holding the dose") is the pharmacist's clinical work. The engine does not suggest "Order STAT lithium level," "Cease perindopril," or "Call daughter today." Those are clinical decisions that require the pharmacist's integration of resident-specific context.

**Principle 6: Intervention opportunity is a first-class scoring layer.** Per Reviewer 1, the engine should explicitly weight whether pharmacist intervention is likely to be useful, not just whether risk is present. Recent medication changes (trajectory still modifiable), unresolved pharmacist recommendations, missing monitoring after medication changes, high-risk medications without recent review — these are intervention-opportunity signals that deserve weight equal to risk signals.

**Principle 7: Memory of failed interventions.** A resident may have legitimate clinical signals warranting attention, but if the relevant intervention was attempted and failed in the past 12 months, surfacing them again creates noise without value. The engine maintains a failed intervention history and uses it as veto-pattern signals. This connects to the veto primitives concept from the prior complex resident workspace synthesis.

**Principle 8: Capacity-aware, pharmacist-controllable, audit-defensible.** The worklist size matches pharmacist reviewing capacity (typically 5–12 items). Every triage decision is contestable. Every adjustment is logged in EvidenceTrace per algorithmic management protections in ethical architecture §8.

---

## Part 2 — Architecture: where CAPE sits

The engine is a service consuming from existing substrate and rule infrastructure, producing prioritisation that populates S1.

### 2.1 Service positioning

```
                    ┌─────────────────────────────┐
                    │ Layer 2 Substrate           │
                    │ (Clinical, Operational,     │
                    │  Consent, Care Intensity,   │
                    │  Goals-of-Care, Trajectories)│
                    └──────────────┬──────────────┘
                                   │
                                   │ subscribes to:
                                   │ - state machine events
                                   │ - observation deltas
                                   │ - trajectory updates
                                   │ - PRN administration patterns
                                   ▼
┌─────────────────┐    ┌─────────────────────────────┐    ┌─────────────────┐
│ Layer 3 Rules   │───▶│ Clinical Attention           │◀───│ kb-32 Craft     │
│ (CQL firings,   │    │ Prioritisation Engine (CAPE) │    │ Engine          │
│ restraint       │    │ (kb-33-triage-engine)        │    │ (recommendation │
│ signals,        │    │                              │    │  lifecycles)    │
│ pattern         │    │ • 5-layer scoring            │    │                 │
│ detectors)      │    │ • Trajectory awareness       │    └─────────────────┘
└─────────────────┘    │ • Capacity calibration       │             ▲
                       │ • Failed intervention memory │             │
┌─────────────────┐    │ • Operational signals        │             │
│ Failed          │───▶│ • Pharmacist controls        │             │
│ Intervention    │    │ • Audit trail                │             │
│ History         │    └──────────────┬──────────────┘             │
└─────────────────┘                   │                            │
                                      │ produces ordered            │
                                      │ worklist entries (not       │
                                      │ specific clinical actions)  │
                                      ▼                            │
                       ┌─────────────────────────────┐             │
                       │ Surface 1 (Pharmacist        │             │
                       │ Worklist) — rendering layer  │─────────────┘
                       │ per Decision Packet          │  surfaces
                       │ Rendering Guidelines          │  pending
                       └─────────────────────────────┘  recommendations
                                      │
                                      │ pharmacist actions
                                      │ (open, defer, mark
                                      │  considered, etc.)
                                      ▼
                       ┌─────────────────────────────┐
                       │ EvidenceTrace audit trail    │
                       │ (algorithmic management      │
                       │ protections per ethical      │
                       │ architecture §8)             │
                       └─────────────────────────────┘
```

### 2.2 What the engine consumes

**From Layer 2 substrate:**
- Clinical state machine: trajectory deltas, vital sign updates, lab result deltas, cognitive assessment changes
- Operational state machine: care plan updates, family meetings, room changes, transfers
- Consent state machine: consent state transitions, restrictive practice consent expiries
- Care intensity state machine: transition events
- Goals-of-care entity: documentation freshness
- **PRN administration patterns:** velocity of PRN benzodiazepine, antipsychotic, analgesic use (per Reviewer 1's emphasis)

**From Layer 3 rules:**
- CQL rule firings (per kb-31 ScopeRules)
- Restraint signal evaluations
- Negative-evidence pattern detections
- Override taxonomy patterns (recurring rejections)

**From kb-32 Craft Engine:**
- Recommendation lifecycle states
- Pending recommendations age (overdue follow-up)
- Outcome observation status

**From scheduling:**
- Scheduled review dates (RMMR, RMMR follow-ups)
- MAC committee dates
- Family meeting dates
- Specialist appointments

**From operational signals:**
- Recent acute events (falls, hospital transfers, rapid response calls)
- Recent medication changes
- Recent diagnostic results
- Family engagement signals (visit frequency, communication patterns)

**From Failed Intervention History (new in v1.1):**
- Previous deprescribing attempts and outcomes (per resident, per medication class)
- Previous dose change attempts and outcomes
- Recommendations that were attempted and reversed
- Pattern detection: which intervention types fail repeatedly for which residents

### 2.3 What the engine produces

For each pharmacist's worklist request, the engine produces:

```go
type AttentionAllocationResult struct {
    Pharmacist        PharmacistID
    Facility          FacilityID
    GeneratedAt       time.Time
    
    PriorityEntries   []AttentionEntry  // typically 5-12 entries
    DeferredEntries   []AttentionEntry  // monitor-only; not in today's list
    NotEvaluable      []AttentionEntry  // substrate gaps prevent evaluation
    SuppressedByVeto  []AttentionEntry  // suppressed due to failed intervention history
    
    FacilityPattern   *FacilityPattern  // "everything looks off" signal if active
    
    CapacityCalibration AttentionCalibration
    EstimatedTotalReviewTime time.Duration  // sum of estimated per-entry times
    AuditTraceRef     uuid.UUID
}

type AttentionEntry struct {
    ResidentID         uuid.UUID
    PriorityRank       int
    LayerScores        map[Layer]float64  // 5 layers, retained
    PrimaryReasons     []AttentionReason   // top 1-3 specific signals
    SecondaryReasons   []AttentionReason
    InterventionOpportunity *OpportunityAssessment  // explicit intervention-opportunity assessment
    LinkedRecommendations []uuid.UUID  // pending recommendations
    ComplexWorkspaceActive bool        // resident in complex workspace mode
    EstimatedReviewTime time.Duration // operational planning support
    PharmacistOverride *Override      // if pharmacist has adjusted
    LastReviewed       *time.Time     // when pharmacist last opened this resident
    DeferralHistory    []DeferralEntry
    FailedInterventionContext *FailedInterventionContext  // if relevant
}

type AttentionReason struct {
    SignalType    string  // e.g., "PRN_benzodiazepine_escalation_velocity"
    SubstrateRefs []SubstrateReference  // verifiable to underlying data
    Severity      int     // 1-5
    Trajectory    *TrajectoryContext   // if trajectory-based
    Description   string  // human-readable, not template-stuffed
    Layer         Layer   // which of 5 layers this contributes to
}

type OpportunityAssessment struct {
    OpportunityScore  int  // 1-5
    Reasoning         string  // why intervention here is/isn't likely to change trajectory
    ReversibilityIndicators []string  // recent changes, modifiable factors
    VetoFactors       []VetoFactor  // failed history, restraint signals, etc.
}
```

### 2.4 Engine boundary (sharpened in v1.1)

The engine does NOT:
- Generate recommendations (that's kb-32 craft engine's job)
- Make clinical decisions (that's the pharmacist's job)
- Modify recommendations (that's craft engine + rendering layer)
- Aggregate to employer view (visibility class enforcement per self-visibility module)
- **Suggest specific clinical actions** (new in v1.1) — *the engine surfaces signals and dimensions; it does not say "Order STAT lithium level," "Cease perindopril," or "Call daughter Lisa." Those are clinical decisions that require the pharmacist's integration of resident-specific context. The engine is bounded at surfacing what's there; the integration to specific actions is the pharmacist's irreducible clinical contribution.*

The engine DOES:
- Rank residents by attention-allocation priority
- Surface specific signals driving the priority
- Surface intervention opportunity assessment (high/moderate/low) with reasoning
- Maintain pharmacist-specific deferral and adjustment state
- Maintain failed intervention history and apply as veto patterns
- Detect facility-level patterns
- Provide audit trail of attention allocation decisions
- Estimate review time per entry to support pharmacist session planning

This boundary is consequential. The engine is an attention-allocation service, not a clinical reasoning service. The clinical reasoning happens in the pharmacist's mind; the engine surfaces the residents and the signals that warrant that reasoning today.

---

## Part 3 — Five-layer scoring architecture

The v1.0 document used six dimensions; v1.1 adopts Reviewer 1's cleaner five-layer architecture, with intervention opportunity as a first-class layer.

### 3.1 The five layers

| Layer | What it captures | Substrate sources |
|---|---|---|
| **Layer 1: Resident Stability Signals** | Trajectory changes and acute events affecting clinical stability | Clinical state machine trajectories, acute events, observation deltas |
| **Layer 2: Medication Contribution Signals** | Medications plausibly contributing to current instability | MedicineUse entities, anticholinergic burden, sedative load, FRIDs, narrow-therapeutic-index drugs |
| **Layer 3: Complexity Signals** | Multi-domain coupling requiring cognitive escalation | Cross-domain pattern detection, goals-of-care drift, transition instability, monitoring fragility |
| **Layer 4: Intervention Opportunity** | Probability that pharmacist intervention will change trajectory | Recent med changes, pending recommendations, reversible trajectories, review overdue, failed intervention history (veto) |
| **Layer 5: Governance Signals** | Accountability and regulatory exposure | Restrictive practice review overdue, psychotropic monitoring gaps, missing rationale, audit findings |

The five layers score independently. The composite for ordering is computed from the layer scores; the layer scores remain visible in the worklist entry rendering. The pharmacist sees what is contributing, not just a single number.

### 3.2 Layer 1: Resident Stability Signals

This layer captures what is changing in the resident's clinical state that warrants attention.

**High-value signal classes:**

**PRN escalation velocity** (per Reviewer 1's emphasis):
- PRN benzodiazepine use: rate of administrations per week, with velocity computation
- PRN antipsychotic use: frequency change over rolling 30-day window vs 90-day baseline
- PRN analgesic escalation: frequency increase suggesting unrecognised pain or symptom progression
- PRN antiemetic, laxative, or sedative escalation patterns

```cql
define "PRN Escalation Velocity":
  let recent_30d = count of PRN administrations of class X in last 30 days
  let baseline_90d = average of PRN administrations of class X per 30 days, prior 90 days
  let velocity_ratio = recent_30d / baseline_90d
  in
    case
      when velocity_ratio > 4.0 then 5  // 400%+ increase
      when velocity_ratio > 2.5 then 4  // 250%+ increase
      when velocity_ratio > 1.5 then 3  // 150%+ increase
      when velocity_ratio > 1.0 then 2  // any increase
      else 1
    end
```

PRN escalation is one of the strongest early instability markers in aged care, and the v1.0 document under-emphasised it. v1.1 elevates it to a primary signal class.

**Sedation drift** (per Reviewer 1):
- Daytime somnolence documented in nursing notes
- Mobility decline (ADL or care plan observations)
- Reduced engagement (social withdrawal, reduced dining room presence)
- Increased sleeping (sleep duration changes)
- Pattern composition: ≥2 of these within 14 days fires sedation drift signal

**Falls and near-falls clustering:**
- Fall events in past 14 days
- Near-fall events in past 14 days
- Clustering with psychotropic medications, sedation, orthostasis

**Delirium/confusion evolution:**
- Fluctuating cognition documented
- Acute confusion within 14 days
- New agitation
- Sleep reversal pattern

**Hospital transition events:**
- Recent hospital discharge (≤14 days)
- Medication changes documented
- Reconciliation incomplete

**Renal trajectory:**
- eGFR declining at velocity above threshold
- Particularly relevant when narrow-therapeutic-index drugs active

**Cognitive trajectory:**
- MMSE/MoCA decline at velocity above threshold
- 4AT score elevation suggesting delirium

**Functional trajectory:**
- CFS progression
- ADL decline
- Weight loss velocity

### 3.3 Layer 2: Medication Contribution Signals

This layer captures medications plausibly contributing to current instability — the question is not "what is on the list" but "what is plausibly contributing to what's changing."

**Anticholinergic burden:**
- ACB score with trajectory
- ACB ≥5 contributing to cognitive decline or constipation
- ACB increase recent (new anticholinergic added)

**Sedative load:**
- Drug Burden Index (DBI) with trajectory
- DBI ≥3 contributing to falls or sedation drift
- DBI increase recent

**Fall-Risk-Increasing Drugs (FRIDs):**
- Active FRIDs in resident with recent fall
- FRID combinations (psychotropic + antihypertensive + diuretic)

**Narrow-therapeutic-index drugs:**
- Lithium (with renal trajectory awareness)
- Digoxin
- Warfarin
- Phenytoin
- Each with monitoring obligation status

**Psychotropic exposure with stability changes:**
- Antipsychotic + behavioural destabilisation
- Benzodiazepine + falls
- Antidepressant + serotonin syndrome risk factors

**Critical discipline:** This layer is *not* about generic potentially inappropriate medications (Beers/STOPP). Static PIM flags belong as background substrate; they do not drive priority on their own. A medication only contributes to Layer 2 score when it is plausibly contributing to current instability evidenced in Layer 1 signals.

### 3.4 Layer 3: Complexity Signals

This layer captures situations where multi-domain coupling makes pharmacist cognitive engagement disproportionately valuable.

**Multi-domain coupling patterns:**
- Lithium + declining renal function + potential delirium + ACE-i (4-way coupling)
- CHF + diuretic + low BP + falls + fatigue (no safe single intervention)
- Diabetes + insulin + poor appetite + weight loss + frequent hypos (competing priorities)
- Anticoagulant + recent fall + thrombocytopenia + NSAID (bleed risk cascade)
- Antipsychotic + anticholinergic burden + delirium + parkinsonism (CNS toxicity cascade)

**Goals-of-care drift:**
- Goals of care documented but medication regimen not aligned
- Recent care intensity transition without medication review
- Curative-intent medications in comfort-care residents

**Transition instability:**
- Hospital discharge with incomplete reconciliation
- Recent admission still in high-error window
- GP practice change

**Monitoring fragility:**
- Multiple monitoring obligations active
- Monitoring overdue with declining clinical state

When Layer 3 signals fire, the engine often activates the Complex Resident Workspace mode (per the prior synthesis discussion). The worklist entry indicates this and offers direct entry to the complex workspace.

### 3.5 Layer 4: Intervention Opportunity (first-class in v1.1)

This is the layer most strongly elevated in v1.1 per Reviewer 1's framing. A resident may have genuine clinical signals but be unlikely to benefit from pharmacist intervention today; the engine surfaces those of higher opportunity preferentially.

**High-opportunity signals:**

**Recent medication changes:**
- Medications changed within 14 days create modifiable trajectory
- Pharmacist intervention while trajectory is settling has high impact
- After 30+ days of stability, the same medications are less modifiable without strong rationale

**Unresolved pharmacist recommendations:**
- Pending recommendations >7 days without GP response
- Pending recommendations >30 days especially urgent (governance + clinical)
- Recommendations awaiting outcome observation

**Missing monitoring after medication changes:**
- Opioid increased without bowel monitoring
- Antipsychotic added without metabolic monitoring
- Lithium change without level recheck
- Anticoagulant started without INR or renal function check

**High-risk medication without recent review:**
- Lithium level overdue
- Warfarin INR overdue (if applicable)
- Insulin without recent glycaemic review
- Anticonvulsant without level if indicated

**Reversibility indicators:**
- New symptom within 14 days of medication change (high reversibility)
- Pattern suggesting medication-induced syndrome (delirium with high ACB, falls with new BP medication)

**Veto factors (suppress intervention opportunity):**
- Failed intervention history within 12 months
- Active restraint signals (per craft engine §10)
- Care intensity transition recent (stabilisation period)
- Goals-of-care explicitly aligned with current regimen

**The opportunity assessment:**

```go
type OpportunityAssessment struct {
    OpportunityScore  int  // 1-5
    Reasoning         string
    ReversibilityIndicators []string
    VetoFactors       []VetoFactor
}

func ComputeOpportunity(resident Resident) OpportunityAssessment {
    base := 0
    
    // Positive contributors
    if hasRecentMedicationChange(resident, 14*Days) {
        base += 2
    }
    if hasPendingRecommendations(resident, 7*Days) {
        base += 2
    }
    if hasMissingMonitoring(resident) {
        base += 2
    }
    if hasReversibilityIndicators(resident) {
        base += 1
    }
    
    // Veto factors
    veto := []VetoFactor{}
    if hasFailedInterventionHistory(resident, 12*Months) {
        veto = append(veto, FailedHistoryVeto)
        base = max(0, base - 3)
    }
    if hasActiveRestraintSignal(resident) {
        veto = append(veto, RestraintVeto)
        base = max(0, base - 2)
    }
    if hasRecentCareIntensityTransition(resident, 14*Days) {
        veto = append(veto, StabilisationVeto)
        base = max(0, base - 2)
    }
    
    return OpportunityAssessment{
        OpportunityScore: clamp(base, 0, 5),
        Reasoning: composeReasoning(base, veto),
        ReversibilityIndicators: extractIndicators(resident),
        VetoFactors: veto,
    }
}
```

The opportunity assessment is rendered alongside the layer scores in worklist entries — the pharmacist sees not just "this resident has signals" but "intervention opportunity is high because of recent medication change and missing monitoring; no veto factors active."

### 3.6 Layer 5: Governance Signals

This layer captures regulatory and accountability dimensions that warrant pharmacist attention regardless of clinical change.

**Restrictive practice review obligations:**
- Antipsychotic 12-week review overdue
- Chemical restraint consent renewal due
- Behaviour Support Plan integration overdue

**Psychotropic monitoring gaps:**
- Metabolic monitoring overdue (antipsychotic)
- Dystonia/akathisia screening overdue
- Sedation level documentation gaps

**Missing rationale:**
- Long-term medications without documented current indication
- PIMs in long-term use without review

**Audit findings:**
- ACQSC findings related to this resident
- MAC committee items
- Quality improvement indicators

Layer 5 typically contributes 0–10% to the composite ordering, but ensures governance-critical items don't get systematically deprioritised by clinical signal absence.

### 3.7 Composite ordering with retained dimensionality

```go
func ComposeOrdering(layers map[Layer]float64) (composite float64, primary []Layer) {
    // Composite for ordering only — never the user-facing primary number
    composite = layers[ResidentStability] * 0.30 +
                layers[MedicationContribution] * 0.20 +
                layers[Complexity] * 0.15 +
                layers[InterventionOpportunity] * 0.25 +
                layers[Governance] * 0.10
    
    primary = topLayersContributing(layers)
    return composite, primary
}
```

The weights (30%, 20%, 15%, 25%, 10%) are calibration starting points. v1.1 elevates Intervention Opportunity to 25% (vs ~10% in v1.0's six-dimension model) reflecting Reviewer 1's framing that intervention opportunity is co-primary with stability signals, not a secondary consideration.

**Critical:** The composite is used for ordering only. The worklist entry shows layer scores, signals, and opportunity assessment — never just the composite number. Reviewer 1's framing is adopted explicitly: *"Output should NOT be Risk Score = 92. Terrible UX."*

### 3.8 What the worklist entry shows (rendering the multi-dimensional state)

```
Mr Patel — Priority 1
─────────────────────────────────────────────────────────────────
Stability 5 │ Med Contribution 4 │ Complexity 4 │ Opportunity 4 │ Governance 1

Why surfaced today (top signals):
• PRN benzodiazepine use 2/week → 8/week over 5 days (escalation velocity 5)
• Sedation drift documented: nursing notes 5/9, 5/11; mobility decline 5/12
• Near-fall yesterday (Layer 1 cluster: PRN escalation + sedation + near-fall)
• eGFR trajectory 48 → 38 over 90 days (velocity -3.3/month, accelerating)

Intervention opportunity: HIGH
• Recent medication change 11 days ago (frusemide dose increase)
• Missing monitoring: lithium level 117 days old
• No active veto factors (no failed deprescribing history; no restraint active)

Estimated review time: 35 minutes (multi-domain; complex workspace recommended)

[Open complex resident workspace] [Mark as considered] [Defer to next session]
```

The worklist entry is what the pharmacist sees. The composite ordering decides Mr Patel is Priority 1. The dimensional breakdown, signals, opportunity assessment, and time estimate are what the pharmacist actually reads to decide what to do.

---

## Part 4 — Signal taxonomy

Approximately 35–45 signal classes across the five layers. Final taxonomy emerges from clinical informatics review and pilot evidence.

### 4.1 Signal taxonomy structure

Each signal class carries:
- Layer membership (1–5)
- Severity (1–5)
- Substrate origins
- Computation logic
- Veto interactions (which signals suppress this one)
- Rendering pattern (how it appears in worklist entries)

### 4.2 High-value signal classes by layer

**Layer 1 (Resident Stability):**
- `PRN_benzodiazepine_escalation_velocity` ★
- `PRN_antipsychotic_escalation_velocity` ★
- `PRN_analgesic_escalation_velocity` ★
- `sedation_drift_pattern` ★
- `fall_with_injury_recent`
- `fall_no_injury_recent`
- `near_fall_clustering`
- `hospital_transfer_recent`
- `rapid_response_call_recent`
- `delirium_confusion_evolution`
- `behavioural_destabilisation`
- `weight_loss_significant`
- `intake_decline_significant`
- `renal_function_declining`
- `cognitive_decline_accelerating`
- `functional_decline_significant`
- `frailty_progression`

★ = Reviewer 1's emphasis incorporated in v1.1

**Layer 2 (Medication Contribution):**
- `anticholinergic_burden_high_with_delirium`
- `anticholinergic_burden_increased_recent`
- `sedative_load_high_with_falls`
- `sedative_load_increased_recent`
- `FRID_combination_with_recent_fall`
- `narrow_therapeutic_index_with_renal_decline`
- `psychotropic_with_behavioural_destabilisation`
- `polypharmacy_with_recent_acute_event`

**Layer 3 (Complexity):**
- `lithium_renal_decline_ACE_inhibitor_cascade`
- `CHF_diuretic_falls_competing_priorities`
- `diabetes_insulin_intake_decline_competing`
- `anticoagulant_fall_risk_cascade`
- `CNS_toxicity_cascade`
- `goals_of_care_medication_misalignment`
- `transition_instability_complex`
- `monitoring_fragility_multiple_obligations`

**Layer 4 (Intervention Opportunity):**
- `recent_medication_change_modifiable`
- `pending_recommendation_overdue_amber`
- `pending_recommendation_overdue_red`
- `missing_monitoring_after_med_change`
- `high_risk_medication_review_overdue`
- `reversibility_indicators_present`
- `failed_intervention_history_veto` (suppressor)
- `restraint_signal_active_veto` (suppressor)
- `stabilisation_period_veto` (suppressor)

**Layer 5 (Governance):**
- `restrictive_practice_review_overdue`
- `psychotropic_metabolic_monitoring_overdue`
- `medication_indication_review_overdue_24mo`
- `MAC_committee_item_pending`
- `ACQSC_finding_resident_level`

### 4.3 Failed Intervention History as veto pattern (new in v1.1)

The failed intervention history is a first-class veto pattern, addressing both Reviewer 2's emphasis and the prior complex resident workspace synthesis on veto primitives.

```go
type FailedInterventionRecord struct {
    ResidentID         uuid.UUID
    InterventionType   string  // e.g., "antipsychotic_deprescribing"
    AttemptDate        time.Time
    Outcome            string  // "reversed_due_to_BPSD_recurrence", "reversed_due_to_family_request", etc.
    DocumentedReason   string
    RetryEligibleDate  time.Time  // typically attempt date + 12 months
    DocumentedBy       PharmacistID
}

func IsVetoActive(resident Resident, proposedIntervention string) (bool, *FailedInterventionRecord) {
    history := GetFailedInterventionHistory(resident.ID, proposedIntervention)
    for _, record := range history {
        if record.RetryEligibleDate.After(time.Now()) {
            return true, &record
        }
    }
    return false, nil
}
```

When the engine considers surfacing a resident for a particular intervention type, it checks the failed intervention history. If a relevant intervention was attempted and reversed within 12 months, the engine:

1. Suppresses the intervention from the worklist's primary reasons
2. Adds a context note: "Previous attempt: 2025-08-15, reversed due to BPSD recurrence; not retry-eligible until 2026-08-15"
3. May still surface the resident if other signals are strong, but with the context

The failed intervention history is part of EvidenceTrace — pharmacists' override capture (per craft engine §5) populates it automatically. No separate entry workflow is required.

### 4.4 Signal severity calibration

Signal severity (1–5) feeds the layer score. Severity is calibrated by clinical informatics with senior consultant pharmacist validation.

**Severity 5 (highest):** Signals warranting same-day attention regardless of context.
- Fall with injury within 72 hours
- Hospital transfer within 14 days
- PRN benzodiazepine escalation velocity ratio >4.0
- Lithium level overdue + renal decline + new confusion (Layer 3 cascade)
- New high-risk medication started without monitoring plan

**Severity 4:** Signals warranting attention this week.
- Falls without injury within 7 days
- Sedation drift pattern documented
- Rapid response call within 7 days
- PRN escalation velocity ratio 2.5–4.0
- Trajectory velocity acceleration in any single parameter

**Severity 3:** Signals warranting attention this month.
- Behavioural change documented
- PRN escalation velocity ratio 1.5–2.5
- Frailty progression
- Pending recommendation overdue 30+ days

**Severity 2:** Signals warranting eventual attention.
- Routine pending recommendations 60+ days
- Periodic review due

**Severity 1:** Surveillance only.
- Trajectory monitoring without recent acceleration
- Stable on long-term regimen

The calibration of which signals warrant which severity is clinical informatics work. Senior consultant pharmacist validation during pilot will tune.

---

## Part 5 — Capacity-aware prioritisation

A 30-item worklist is operationally useless. The engine produces a list calibrated to the pharmacist's reviewing capacity, with explicit time estimates per entry.

### 5.1 The capacity model

```yaml
default_capacity:
  reviewing_minutes_per_session: 240  # 4 hours
  
adjustment_factors:
  high_complexity_resident_multiplier: 1.5  # complex residents take longer
  fragmented_session_multiplier: 0.8  # interrupted sessions reduce throughput
  
pharmacist_calibration:
  initial: default
  adjusted_quarterly_based_on: observed_completion_rates
```

### 5.2 Estimated review time per entry (new in v1.1)

Per Reviewer 2's emphasis, each worklist entry includes an estimated review time:

```go
func EstimateReviewTime(entry AttentionEntry) time.Duration {
    base := 20 * time.Minute  // baseline review
    
    // Complexity adjustments
    if entry.LayerScores[Complexity] >= 4 {
        base += 25 * time.Minute  // multi-domain integration
    }
    if entry.ComplexWorkspaceActive {
        base += 15 * time.Minute  // cognitive support session
    }
    if entry.LayerScores[InterventionOpportunity] >= 4 {
        base += 5 * time.Minute  // action conversation likely
    }
    
    // Operational adjustments
    if hasFamilyCommunicationIndicated(entry) {
        base += 15 * time.Minute  // proactive family contact
    }
    if hasGPCommunicationLikely(entry) {
        base += 10 * time.Minute  // recommendation drafting
    }
    
    return base
}
```

The estimated review time appears on each entry. The aggregate estimated review time appears at the worklist level: "Estimated total review time: 4 hours 15 minutes." If aggregate exceeds the pharmacist's session capacity, lower-priority entries are deferred.

This is operationally useful: it answers Reviewer 1's question — *"I have 90 minutes — where do I create the most safety?"* — by letting the pharmacist see at a glance what their session can realistically accomplish.

### 5.3 Worklist sizing logic

```go
func SizeWorklist(entries []AttentionEntry, capacity time.Duration) WorklistResult {
    sorted := sortByComposite(entries)
    
    priorityList := []AttentionEntry{}
    deferred := []AttentionEntry{}
    
    cumulativeTime := 0
    for _, entry := range sorted {
        estimated := EstimateReviewTime(entry)
        if cumulativeTime + estimated <= capacity {
            priorityList = append(priorityList, entry)
            cumulativeTime += estimated
        } else {
            deferred = append(deferred, entry)
        }
    }
    
    // Severity-5 floor: always surface severity-5 signals regardless of capacity
    for _, entry := range deferred {
        if hasSeverityFiveSignal(entry) {
            priorityList = append(priorityList, entry)
            // Pharmacist sees "exceeded capacity" indicator
        }
    }
    
    return WorklistResult{
        PriorityEntries: priorityList,
        DeferredEntries: deferred,
        EstimatedTotalReviewTime: cumulativeTime,
    }
}
```

### 5.4 The deferred list and not-evaluable list

(Same as v1.0 — preserved for continuity.)

---

## Part 6 — Trajectory and velocity computation

(Same fundamentals as v1.0, with one v1.1 addition.)

### 6.1 Multi-window velocity

The engine computes velocity over multiple time windows:
- 30-day velocity (recent)
- 90-day velocity (mid-term)
- 180-day velocity (long-term context)

Acceleration = recent velocity − long-term velocity. Accelerating decline is more concerning than stable decline.

### 6.2 Z-score outlier detection (new in v1.1)

Per Reviewer 2's emphasis, the engine uses z-score outlier detection for trajectory parameters:

```go
func IsTrajectoryOutlier(parameter string, currentValue float64, history TrajectoryHistory) bool {
    mean := history.RollingMean(180 * Days)
    stddev := history.RollingStdDev(180 * Days)
    zscore := (currentValue - mean) / stddev
    return abs(zscore) > 2.0
}
```

Z-score >2 SD flags as outlier. This is more robust than absolute thresholds because it adapts to the resident's individual trajectory.

### 6.3 Trajectory rendering in worklist entries

(Same as v1.0.)

### 6.4 Multi-parameter trajectory composition

(Same as v1.0.)

---

## Part 7 — Pharmacist controls

(Same controls as v1.0, with v1.1 additions.)

### 7.1 The five primary controls

(Same as v1.0: Open, Mark as considered, Defer, Promote, Override.)

### 7.2 The "I disagree with this priority" affordance

(Same as v1.0.)

### 7.3 Failed intervention documentation (new in v1.1)

When a pharmacist's recommendation is reversed (per craft engine override taxonomy), the failed intervention record is automatically populated. The pharmacist can:

- Confirm the failure documentation
- Adjust the retry-eligibility date (default 12 months; can be shortened or lengthened)
- Add documented reason
- Mark as "do not retry without senior review"

This documentation flows into the failed intervention history and informs subsequent attention allocation. The discipline matters: well-documented failures are a feature, not a deficiency. The platform's memory of what didn't work is part of its growing implementation intelligence.

### 7.4 Calibration learning (Phase 4 capability)

(Same as v1.0 — calibration adapts per pharmacist within bounded parameters.)

### 7.5 Audit trail

(Same as v1.0.)

---

## Part 8 — Integration with existing surfaces and modules

(Same integrations as v1.0.)

### 8.1 Integration with S1 Pharmacist Worklist
### 8.2 Integration with S2 Resident Workspace
### 8.3 Integration with Complex Resident Workspace
### 8.4 Integration with kb-32 Craft Engine
### 8.5 Integration with KB-29 Templates
### 8.6 Integration with pharmacist self-visibility
### 8.7 Integration with ethical architecture

---

## Part 9 — "Everything looks off" mode: facility-level pattern detection

The user's explicit use case — *"chaos situation where everything looks off"* — is operationally important. Per the v1.0 specification, when facility-level patterns activate, the engine surfaces the pattern alongside individual priorities.

(Substantively same as v1.0; key elements preserved.)

### 9.1 Facility-level pattern classes

- Cluster of acute events
- Cluster of trajectory acceleration
- Cluster of restraint signal activations
- Cluster of monitoring overdue
- Cluster of pending recommendation aging
- **Cluster of PRN escalation across multiple residents** (new in v1.1, per Reviewer 1's emphasis)

### 9.2 Coordination support

(Same as v1.0.)

### 9.3 Discipline against false alarms

(Same as v1.0.)

---

## Part 10 — Performance monitoring (sharpened in v1.1)

The performance metrics are reframed in v1.1 to reflect the attention-allocation framing rather than the risk-scoring framing.

### 10.1 Performance metrics

**Metric 1: Priority engagement rate.** Of priority entries surfaced, what fraction does the pharmacist actually open and review? Target: ≥80%.

**Metric 2: Priority deferral rate.** Of priority entries surfaced, what fraction does the pharmacist defer or mark as considered? Target: ≤20%.

**Metric 3: Promoted-from-deferred rate.** Of deferred entries, what fraction does the pharmacist promote? Target: ≤10%.

**Metric 4: Override rate.** Of priority orderings, what fraction does the pharmacist override? Target: ≤25%.

**Metric 5: Intervention opportunity realisation rate (reframed in v1.1).** Of priority items the pharmacist engaged with, what fraction resulted in clinically meaningful action — recommendation generated, monitoring scheduled, family communication initiated, GP discussion held? Target: ≥60%. *This is the attention-allocation metric. A list that pharmacists engage with but produces no action is a list that allocated attention to the wrong residents.*

**Metric 6: Time-to-priority-engagement.** When a priority item appears, how quickly does the pharmacist engage with it? Target: same session for severity 4–5 items.

**Metric 7: Pharmacist-reported satisfaction (new in v1.1, per Reviewer 2).** Quarterly Likert-scale survey: "The engine surfaces residents I would have otherwise missed" / "The engine surfaces residents I judge are not the right priority." Target: 70%+ "agree" on first; <20% "agree" on second.

**Metric 8: Prioritisation accuracy (new in v1.1, per Reviewer 2).** Quarterly blind review: senior consultant pharmacist independently triages a sample of facility days, compared with engine output. Target: >75% agreement on top-tier residents.

**Metric 9: Outcome correlation — prevented harms (new in v1.1).** Six-monthly review: of residents engaged through CAPE-surfaced priorities, what fraction had subsequent intervention that plausibly prevented an adverse event (avoided hospitalisation, prevented fall, addressed toxicity precursor)? This is hard to attribute causally but tracks the engine's clinical impact.

**Metric 10: GP acceptance rate impact (new in v1.1).** Does engagement with CAPE-surfaced residents produce recommendations with higher GP acceptance than baseline? If CAPE is correctly identifying high-opportunity residents, the recommendations should have higher acceptance because they are more clinically warranted. Baseline 51.5% from Ramsey 2025; CAPE-engagement target should aim higher.

### 10.2 Calibration tuning thresholds

When metrics deviate from targets:

- Priority engagement <70% sustained: priority threshold too aggressive
- Priority deferral >30% sustained: same
- Promoted-from-deferred >15% sustained: priority threshold not aggressive enough
- Override rate >35% sustained: engine logic mismatch with clinical reasoning; clinical informatics review required
- Intervention opportunity realisation <50% sustained: engine surfacing items that don't produce meaningful action; signal taxonomy and Layer 4 weighting review
- Pharmacist satisfaction <60% sustained: structural review required

### 10.3 Pharmacist-specific and facility-specific calibration

(Same as v1.0.)

### 10.4 Quarterly calibration review

(Same as v1.0.)

### 10.5 Anti-pattern discipline (sharpened in v1.1)

The engine must resist scope creep. Specific anti-patterns to reject:

- **Anti-pattern 1: Adding signals because they're available.** Every additional signal class needs to demonstrate it improves intervention opportunity realisation, not just adds information.
- **Anti-pattern 2: Increasing priority list size to "be safe."** Larger lists reduce engagement; smaller lists with calibration fit beat larger lists with low threshold.
- **Anti-pattern 3: Predictive modelling overlay.** The engine recognises substrate signals; it does not predict baseline risk. Adding ML-based prediction layers risks the failure modes documented in PADR-EC, BADRI, GerontoNet research.
- **Anti-pattern 4: Removing pharmacist override pathways.** Override pathways remain regardless of engine maturity. They are the operational substrate of pharmacist autonomy.
- **Anti-pattern 5 (new in v1.1): Surfacing specific clinical actions as engine output.** "Order STAT lithium level" / "Cease perindopril" / "Call daughter today" — these are clinical decisions, not engine output. The engine surfaces signals; the pharmacist integrates to action.
- **Anti-pattern 6 (new in v1.1): Single composite score as primary user-facing number.** "URGENCY: 87/100" loses the dimensionality that drives clinical reasoning. The composite is for ordering only.
- **Anti-pattern 7 (new in v1.1): Premature integration of wearable sensor data, ambient monitoring, or NLP on nursing notes.** These are Phase 4 capabilities per the maturity roadmap, not Phase 1. Adding them now creates engineering scope and clinical safety implications that the Phase 1 architecture cannot validate.

---

## Part 11 — Junior pharmacist scaffolding without de-skilling

(Same fundamentals as v1.0; v1.1 preserves.)

### 11.1 The reasoning surfacing principle
### 11.2 The reasoning library
### 11.3 The progressive disclosure of complexity
### 11.4 The mentorship integration

---

## Part 12 — File and code organisation

(Same as v1.0, with naming clarification.)

The service code lives in `kb-33-triage-engine/`. The architectural concept is referenced in code comments and documentation as CAPE. User-facing strings (worklist headers, help text, training material) use "Clinical Attention Prioritisation" or contextual abbreviations; "Daily Triage Engine" is retired from user-facing language.

(File structure same as v1.0.)

---

## Part 13 — API contracts

(Same as v1.0, with one rename.)

```protobuf
service ClinicalAttentionPrioritisationService {
    // gRPC service — code namespace remains kb_33_triage_engine for continuity
    
    rpc GenerateWorklist(WorklistRequest) returns (AttentionAllocationResult);
    rpc RefreshWorklist(RefreshRequest) returns (AttentionAllocationResult);
    
    rpc OpenWorklistEntry(EntryAction) returns (Acknowledgment);
    rpc MarkAsConsidered(EntryAction) returns (Acknowledgment);
    rpc DeferEntry(DeferRequest) returns (Acknowledgment);
    rpc PromoteFromDeferred(EntryAction) returns (AttentionAllocationResult);
    rpc OverrideOrdering(OverrideRequest) returns (AttentionAllocationResult);
    
    rpc FlagPriorityDisagreement(DisagreementRequest) returns (Acknowledgment);
    
    rpc GetPharmacistCalibration(PharmacistID) returns (Calibration);
    rpc UpdatePharmacistCalibration(CalibrationUpdate) returns (Calibration);
    
    rpc GetFacilityPatterns(FacilityID) returns (FacilityPatterns);
    
    rpc GetPerformanceMetrics(MetricsRequest) returns (Metrics);
    
    // New in v1.1:
    rpc DocumentFailedIntervention(FailedInterventionRecord) returns (Acknowledgment);
    rpc GetFailedInterventionHistory(ResidentID) returns (FailedInterventionList);
    rpc GetEstimatedReviewTime(ResidentID) returns (TimeEstimate);
}
```

---

## Part 14 — Testing approach

(Same six categories as v1.0, with v1.1 additions.)

### Category 1–6: as v1.0

### Category 7 (new in v1.1): Intervention opportunity realisation tests

- Layer 4 score correctly elevated when reversibility indicators present
- Veto factors correctly suppress opportunity score
- Failed intervention history correctly applied as veto

### Category 8 (new in v1.1): Single-score collapse rejection tests

```go
func TestWorklistEntryNeverShowsCompositeAsHeadline(t *testing.T) {
    entry := generateWorklistEntry()
    rendered := renderWorklistEntry(entry)
    
    require.NotContains(t, rendered, "URGENCY: ")
    require.NotContains(t, rendered, "Score:")
    require.NotContains(t, rendered, "Composite:")
    
    // Layer scores must be visible
    require.Contains(t, rendered, "Stability")
    require.Contains(t, rendered, "Med Contribution")
    require.Contains(t, rendered, "Complexity")
    require.Contains(t, rendered, "Opportunity")
    require.Contains(t, rendered, "Governance")
}
```

### Category 9 (new in v1.1): Engine boundary tests

```go
func TestEngineDoesNotSuggestSpecificClinicalActions(t *testing.T) {
    entry := generateWorklistEntry()
    rendered := renderWorklistEntry(entry)
    
    // Engine output must not contain specific clinical action recommendations
    forbidden := []string{
        "Order STAT", "Cease ", "Reduce dose", "Increase to",
        "Call daughter", "Discontinue", "Hold the dose",
    }
    for _, phrase := range forbidden {
        require.NotContains(t, rendered, phrase,
            "Engine output must not contain specific clinical action: %s", phrase)
    }
}
```

These tests enforce the v1.1 architectural commitments at CI level.

---

## Part 15 — Performance budgets

(Same as v1.0.)

---

## Part 16 — Implementation sequencing

Phase 3 of the implementation plan, Weeks 18–24.

(Substantively same as v1.0; minor scope adjustments for v1.1 changes.)

### Week 18: Foundation
- Service scaffold
- Storage and event subscriptions
- Five-layer scoring (basic implementation)
- Failed intervention history substrate

### Week 19: Signal taxonomy + detection
- Signal class definitions across five layers
- PRN escalation velocity detection (priority signal)
- Sedation drift pattern detection
- Detection logic per signal class
- Substrate reference integration

### Week 20: Trajectory computation + opportunity assessment
- Velocity and acceleration primitives
- Z-score outlier detection
- Multi-parameter composition
- Layer 4 (Intervention Opportunity) scoring
- Veto factor logic

### Week 21: Capacity-aware prioritisation + controls
- Worklist sizing logic with time estimation
- Pharmacist control handlers
- EvidenceTrace integration
- Failed intervention documentation flow
- Calibration learning scaffold

### Week 22: Facility pattern detection
- Pattern detector implementations
- Coordination support
- "Everything looks off" mode

### Week 23: Junior pharmacist scaffolding + performance monitoring
- Reasoning library content integration
- Progressive disclosure
- Performance metrics (10 metrics per v1.1 Part 10)
- ERM quarterly review integration

### Week 24: Buffer for testing and external review
- Cross-component integration testing
- Single-score collapse rejection tests
- Engine boundary tests
- External clinical informatics UX review
- Pilot pharmacist user testing (3 pharmacists, 1 week)
- Calibration baseline tuning

### Estimated team

(Same as v1.0; senior consultant pharmacist time emphasised for signal severity calibration and reasoning library content authoring.)

---

## Part 17 — Risks and mitigations

(v1.0 risks preserved; v1.1 additions.)

**Risk 13 (new in v1.1): Composite score leakage into user-facing rendering.** Despite architectural commitment, a UI implementation accidentally surfaces composite as primary number. Mitigation: CI test (Category 8); code review checklist; user-facing string review.

**Risk 14 (new in v1.1): Engine boundary violation through specific clinical action suggestions.** Engine output drifts toward "Order STAT" / "Cease perindopril" specificity. Mitigation: CI test (Category 9); template review for any phrase library; clinical informatics oversight.

**Risk 15 (new in v1.1): Failed intervention history misused as performance evaluation.** Pharmacy chain employer interprets failed intervention records as pharmacist failures. Mitigation: visibility class enforcement (PDP — pharmacist's own clinical work record); algorithmic management protections per ethical architecture §8; failed intervention records never aggregate to employer view at individual pharmacist resolution.

**Risk 16 (new in v1.1): Wearable sensor pressure to ship in Phase 1.** Commercial or research pressure to integrate wearables prematurely. Mitigation: maturity roadmap discipline; Phase 4 placement explicit; commercial team training on capability staging.

---

## Part 18 — Closing

Three observations as we close v1.1.

**One:** The reframing from "Daily Triage Engine" to "Clinical Attention Prioritisation Engine (CAPE)" is structurally consequential, not stylistic. It changes the scoring logic (intervention opportunity becomes co-primary with stability), the signal taxonomy (PRN escalation, sedation drift, failed intervention history elevated), the performance metrics (intervention opportunity realisation, prioritisation accuracy, GP acceptance impact), and the engine boundary (specific clinical actions explicitly excluded). The two reviewers independently surfaced this reframing. The team should adopt the language change in user-facing materials while preserving the service code name for engineering continuity.

**Two:** The operational test for every design decision in CAPE is: *"Does this design choice improve the engine's answer to: 'I have 90 minutes — where do I create the most safety?'"* This test is the discipline against scope creep, alert fatigue, and the failure modes the literature documents. When a feature, signal class, or capability is proposed, run it against this test. If it fails, reject it. If it passes, prioritise it according to the maturity roadmap.

**Three:** The literature on prioritisation tools (PAST, STORIMAP, Wirral, PADR-EC, BADRI, GerontoNet) documents specific failure modes — single-score collapse, alert fatigue, de-skilling, predictive overreach, complex case underperformance. v1.1 incorporates explicit architectural commitments against each, with CI tests enforcing the commitments. The team should treat these tests as load-bearing — they are what protect the platform from drifting into the failure modes that have produced the alert-fatigue tools the field is currently trying to recover from.

What this document does not yet specify, and what should be subsequent work:

- **The reasoning library content** for each signal class (clinical informatics + senior consultant pharmacist authoring)
- **The signal severity calibration** for each signal class (clinical informatics validation)
- **The visual design system** for worklist rendering (UI design work)
- **The integration with the Complex Resident Workspace** at the rendering layer (subsequent specification)
- **The pharmacy chain dashboard** view of CAPE patterns across pharmacists (subsequent specification)
- **Phase 4 capabilities** — wearable sensor integration, ambient monitoring, NLP on nursing notes, predictive risk modelling — explicitly deferred to maturity roadmap Phase 4

The architecture stack now stands at nine implementation guideline documents, with this one updated to v1.1:

1. v3.0 strategic positioning
2. Pilot design
3. Recommendation craft engine
4. Pharmacist self-visibility
5. Ethical architecture
6. Decision packet rendering
7. KB-29 templates
8. KB-29 maturity roadmap
9. **Clinical Attention Prioritisation Engine (CAPE) v1.1** ← this document

The eight months of pilot operation will test whether CAPE genuinely augments pharmacist efficiency in chaos days. Performance metrics in Part 10 are the discipline that determines which outcome we get. The team's willingness to tighten thresholds, retire signal classes, or revise calibration based on metric evidence is what separates the high-leverage outcome from the high-risk outcome.

The two reviewers' contributions to this v1.1 are substantial. Reviewer 1's structural reframing — attention allocation, not risk scoring — is the architectural insight. Reviewer 2's operational specifics — review time estimates, validation metrics, percentage-change baselines, failed intervention memory — are the implementation refinements. v1.1 incorporates both while holding the architectural discipline against the failure modes (single-score collapse, specific clinical action overreach, premature wearable integration) that either reviewer alone would have introduced.

— Claude
