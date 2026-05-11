# CAPE v1.1 Architectural Commitment: Multi-Surface Consumption

**Date:** May 2026
**Document scope:** Supplements *Clinical Attention Prioritisation Engine (CAPE) Implementation Guidelines v1.1*; does not replace it
**Status:** Architectural commitment — load-bearing for Phase 3 engineering decisions and beyond
**Builds on:** CAPE v1.1; v3.0 §5 (Buyer 2: RACH operator, "fewer surprises" KPI); Decision Packet Rendering Implementation Guidelines v1.0 (S1, S2, S4, S5 surfaces); Ethical Architecture Implementation Guidelines v1.0 (visibility classes, algorithmic management protections)

**Why this document exists:** CAPE v1.1 specifies the engine that populates the pharmacist worklist (Surface 1). The platform's commercial strategy targets a second buyer class — RACH operators — with an operational view of resident instability that consumes from the same substrate but renders differently. The architectural relationship between these two consumers is load-bearing: get it right and the platform supports both surfaces without cannibalisation or divergent substrate; get it wrong and the platform builds two engines that produce contradictory signals about the same residents.

This addendum specifies the architectural commitment in writing, before Phase 3 engineering decisions accumulate that would constrain or contradict it.

**Reading order:** Engineering leads read all sections (commitment shapes API design, schema, event subscriptions). Product leads read Sections 1 and 4 (architectural commitment + relationship to commercial sequencing). Clinical informatics leads read Sections 2 and 3 (boundary specification + chronology primitive). Risk and ethics leads read Section 5 (anti-patterns and governance).

---

## Section 1 — The architectural commitment

The commitment is straightforward to state and load-bearing to implement.

### 1.1 The commitment

**The Vaidshala platform observes resident instability through a single longitudinal substrate. The substrate is canonical. Operational surfaces are interpretive renderings of substrate observations, calibrated to specific audiences and workflows. There are not two engines; there is one engine with multiple consumers.**

This commitment has four operational consequences:

- The substrate (Layer 2 state machines + EvidenceTrace + trajectory primitives) is a single source of truth for resident instability observations
- The CAPE engine (Layer 3 + scoring + signal detection + trajectory composition) is a single source of truth for prioritisation and pattern recognition
- The observation layer (substrate + CAPE outputs) is consumed by multiple surfaces, each rendering for its audience
- The action layer (rendering, workflow, escalation pathways, KPI computation) is surface-specific; this is where pharmacist and RACH views legitimately differ

The commitment is not stylistic. It changes engineering decisions about API design, schema scope, event subscription patterns, calibration parameter ownership, and audit trail structure. Sections 2 and 3 specify what's affected.

### 1.2 The two intervention horizons

The most precise framing for the relationship between the pharmacist worklist and the RACH operational view is not "two products sharing data" or "primary and secondary surfaces." It is:

> **Two intervention horizons operating on the same longitudinal instability graph.**

This framing makes architectural sense because both surfaces observe the same evolving resident instability; they differ in *what action layer they trigger*.

**Pharmacist intervention horizon:**
Operational question: *Where can pharmacist intervention alter trajectory?*
Action surface: Medication recommendations, monitoring obligations, deprescribing decisions, GP coordination on clinical reasoning.
Temporal window: Modifiable intervention window — typically 24 hours to 14 days for a specific intervention; longitudinal medication governance over months.

**RACH operational horizon:**
Operational question: *Where is instability emerging that requires operational response?*
Action surface: Staffing adjustments, fall prevention escalation, family communication, GP coordination on care plan, MAC escalation, transfer decision support.
Temporal window: Operational escalation window — typically 4 hours to 72 hours for immediate response; facility-level pattern recognition over weeks.

These horizons overlap because medication-related instability is a major subset of RACF deterioration patterns. They do not compete because they trigger different workflows for different decision-makers with different funding pools and ROI narratives (per v3.0 §5).

### 1.3 What this is not

The commitment is explicit about what it rejects:

- **Not "primary surface and secondary surface."** Both surfaces are first-class consumers of the substrate. Neither is built as a derivative of the other.
- **Not "pharmacist data shared with RACH operators."** The substrate captures resident instability broadly; both audiences observe the same canonical observations.
- **Not "one engine that produces a list, rendered two ways."** The engine produces observation-layer outputs (trajectory primitives, signal detections, pattern recognitions, instability chronologies); each surface composes those outputs into its own rendering.
- **Not "build the RACH surface later as an extension of the pharmacist surface."** The RACH surface is specified later in time per commercial sequencing, but architecturally it is a peer consumer of the substrate, not a downstream extension of the pharmacist surface.

### 1.4 The strategic implication

The commitment shapes Vaidshala's category positioning. A platform that is "a pharmacist tool with reporting features" has a $10M ARR ceiling and competes with hospital pharmacy software extensions. A platform that is "longitudinal resident instability infrastructure with medication governance as privileged lens and multiple operational surfaces" has a different ceiling and competes in a different category.

The architectural commitment is what makes the second positioning honest. Without the commitment, the platform builds pharmacist software and grafts RACH reporting onto it — the positioning would be aspirational, not architectural. With the commitment, the substrate is built once and serves multiple stakeholders; the positioning matches the architecture.

Year 1–2 commercial messaging continues to lead with medication intelligence (the strongest wedge, the clearest ROI, the funded buyer). The architectural truth is broader, and protecting it through engineering discipline is what enables future positioning evolution.

---

## Section 2 — The observation/action layer boundary

The architectural commitment requires a clear boundary between what is shared and what is surface-specific. This section specifies the boundary.

### 2.1 What lives in the observation layer (shared, canonical)

The observation layer is the single source of truth that all surfaces consume from.

**Substrate (Layer 2):**
- All five state machines: Clinical, Operational, Consent, Care Intensity, Goals-of-Care
- EvidenceTrace bidirectional graph
- Trajectory primitives (per-parameter velocity, acceleration, baselines)
- MedicineUse intent/target/stop_criteria entities
- Acute event records
- Family meeting documentation
- Restraint signal evaluations
- Failed intervention history
- Specialist consultation records

**Layer 3 outputs:**
- CQL rule firings (per kb-31 ScopeRules)
- Pattern detector outputs (single-resident patterns)
- Facility-level pattern detection (cross-resident clusters)
- Negative-evidence pattern recognitions

**CAPE engine outputs:**
- Five-layer scoring per resident (Stability, Medication Contribution, Complexity, Intervention Opportunity, Governance)
- Signal detections with substrate references
- Trajectory composition primitives
- Instability chronology primitives (per Section 3)
- Failed intervention history annotations
- Restraint countermand annotations

**Cross-cutting metadata:**
- Observation timestamps (immutable across all surfaces)
- EvidenceTrace edges (single graph; queried by surface)
- Audit trail entries
- Visibility class tags (per pharmacist self-visibility module)

The observation layer is what gets written to canonical storage. No surface writes its own observations; all surfaces read from the same observation store.

### 2.2 What lives in the action layer (surface-specific)

The action layer is where surfaces legitimately differ.

**Rendering rules:**
- Audience adaptation (pharmacist clinical detail vs. RACH operational summary)
- Visual hierarchy and emphasis
- Brevity budgets (different audiences absorb different information density)
- Drill-through patterns (what's one click away from a worklist entry)

**Aggregation patterns:**
- Pharmacist worklist: per-resident detail
- RACH operational dashboard: cross-resident aggregation across facility
- Standard 5 evidence panel: governance-oriented composition
- Audit/regulator interface: investigation-oriented composition

**Workflow integration:**
- Pharmacist surface: opens kb-32 craft engine, generates recommendations, populates monitoring obligations
- RACH surface: triggers operational workflows — staffing coordination, family communication, escalation pathways
- Governance surface: feeds quality improvement processes, MAC committee preparation

**KPI computation:**
- Pharmacist surface: RIR, appropriateness, intervention opportunity realisation, prevented harms
- RACH surface: "fewer surprises" KPI (preventable hospital transfers, SIRS-reportable events, family complaints), Standard 5 evidence quality, star rating support
- Governance surface: closed-loop intervention rate, attribution-validated cost savings

**Controls and affordances:**
- Pharmacist controls: open, defer, mark considered, promote, override (per CAPE v1.1 Part 7)
- RACH controls: acknowledge facility pattern, deploy operational response, escalate to MAC, generate family communication scaffolding
- Governance controls: open audit query, generate evidence pack, export for external review

Action-layer differences are expected, designed, and protected. The surfaces serve different decision-makers with different operational questions; the rendering legitimately diverges.

### 2.3 The API contract between layers

The observation layer exposes outputs through a stable API consumed by action layers. The API contract is the explicit interface that protects against divergent substrate.

```go
// Observation Layer API — single source of truth
service ObservationLayer {
    // CAPE engine outputs
    rpc GetResidentScoring(ResidentID) returns (FiveLayerScoring);
    rpc GetSignalDetections(ResidentID, TimeWindow) returns ([]Signal);
    rpc GetInstabilityChronology(ResidentID, TimeWindow) returns (Chronology);
    rpc GetTrajectoryPrimitives(ResidentID) returns (TrajectoryComposite);
    
    // Substrate queries
    rpc GetSubstrateState(ResidentID) returns (SubstrateSnapshot);
    rpc GetEvidenceTrace(ResidentID, EvidenceQuery) returns (EvidenceGraph);
    rpc GetFailedInterventionHistory(ResidentID) returns ([]FailedInterventionRecord);
    
    // Facility-level outputs
    rpc GetFacilityPatterns(FacilityID) returns ([]FacilityPattern);
    rpc GetResidentList(FacilityID, FilterCriteria) returns ([]ResidentSummary);
    
    // Cross-resident aggregation (for RACH surface)
    rpc GetFacilityInstabilityOverview(FacilityID, TimeWindow) returns (FacilityInstability);
    rpc GetResidentCluster(FacilityID, ClusterCriteria) returns ([]ResidentCluster);
}

// Action Layer Interfaces — surface-specific
service PharmacistWorklistService {
    rpc GeneratePharmacistWorklist(PharmacistID, FacilityID) returns (PharmacistWorklist);
    // Consumes from ObservationLayer; renders for pharmacist audience
}

service RACHOperationalView {
    rpc GenerateFacilityOverview(FacilityID, DONUserID) returns (FacilityOverview);
    // Consumes from ObservationLayer; renders for RACH operational audience
}

service GovernanceWorkspace {
    rpc GenerateAuditView(FacilityID, AuditQuery) returns (AuditView);
    // Consumes from ObservationLayer; renders for governance audience
}
```

**The discipline:** Action-layer services do not bypass the observation layer to write their own observations. If a RACH operational concern requires detecting something the substrate doesn't currently capture, the response is to extend the substrate (making the new observation available to all surfaces), not to build a RACH-private detection.

Section 5 specifies the governance for substrate extensions that protects this discipline.

### 2.4 What the action layer can legitimately compute on top of observations

The action layer is allowed substantial composition on top of observation-layer outputs:

- **Audience-adapted prose:** Same substrate observation rendered with different framing (pharmacist clinical vs. operational management language)
- **Aggregation across residents:** Cross-resident composition for facility dashboards
- **Workflow state:** Action-specific state (escalation acknowledged, response deployed, family contacted)
- **Surface-specific metrics:** KPIs computed from observations but specific to surface success measurement
- **Visual composition:** Layout, emphasis, drill-through patterns

The discipline is: action layer composes from observations; it does not generate observations. A new observation type requires substrate extension; a new presentation of existing observations is action-layer work.

### 2.5 Cross-surface coherence requirement

When pharmacist, DON, and governance review the same resident on the same day, the underlying observations must be identical. Different rendering is expected; different chronology is not.

If Mrs Chen has:
- `confusion_acute` observation timestamped 2026-05-09 06:00
- `fall_no_injury_recent` observation timestamped 2026-05-09 08:00
- `PRN_benzodiazepine_escalation_velocity` signal active

Then:
- Pharmacist sees these three observations rendered as priority drivers with intervention opportunity assessment
- DON sees the same three observations rendered as instability escalation requiring operational response
- Governance sees the same three observations rendered as part of an intervention lifecycle audit

The timestamps match. The substrate references match. The severity assessment is shared (though contextual interpretation differs). The chronology is preserved.

This is what enables trust across stakeholders. A pharmacist who tells the DON "I saw the same fall data you saw, my response was X" depends on the observations being identical. A governance review that examines an intervention lifecycle depends on the chronology being preserved across all surfaces that interacted with the case.

CI tests (Section 5) enforce this property at code level.

---

## Section 3 — Instability chronology as first-class primitive

The most powerful shared rendering primitive across all surfaces is the cross-parameter event chronology. CAPE v1.1 specifies per-parameter trajectories (eGFR over time, MMSE over time); the instability chronology composes events across parameters into a temporal narrative.

### 3.1 The primitive

```go
type InstabilityChronology struct {
    ResidentID    ResidentID
    TimeWindow    TimeWindow
    Events        []ChronologyEvent
    Patterns      []TemporalPattern  // patterns recognised across events
    Severity      Severity
    AudienceAdaptations map[AudienceClass]ChronologyRendering
}

type ChronologyEvent struct {
    Timestamp        time.Time
    EventType        string  // e.g., "medication_change", "intake_decline", "fall", "confusion_onset"
    PrimitiveType    InstabilityPrimitive  // canonical vocabulary
    Severity         Severity
    Description      string  // factual, audience-neutral
    SubstrateRefs    []SubstrateReference  // verifiable to underlying data
    SuspectedCauses  []string  // when temporal pattern suggests causation
    RelatedEvents    []EventID  // co-occurring or causally adjacent
}

type TemporalPattern struct {
    PatternType   string  // e.g., "cascade", "escalation", "drift", "destabilisation"
    EventSequence []EventID
    Reasoning     string  // why this constitutes a pattern
    Confidence    string  // "established", "suggestive", "preliminary"
}
```

### 3.2 The rendering example

For Mrs Chen, the chronology composes events across parameters:

```
Mrs Chen — Instability chronology (last 14 days)
─────────────────────────────────────────────────────────────────

Day -8 (2026-05-01): Frusemide dose increased 40mg → 80mg daily
                     Substrate refs: MedicineUse change record; GP letter 2026-05-01

Day -6 (2026-05-03): Reduced PO intake first noted
                     Substrate refs: Nursing progress notes 2026-05-03; intake chart

Day -4 (2026-05-05): Orthostatic BP drop documented (sit 138/82, stand 108/68)
                     Substrate refs: Vital signs observation 2026-05-05

Day -2 (2026-05-07): Daytime somnolence first documented; mobility decline
                     Substrate refs: Nursing notes 2026-05-07; ADL chart

Day 0 (2026-05-09):  Near-fall in bathroom; confusion onset (4AT 4)
                     Substrate refs: Incident report 2026-05-09 08:00;
                                     cognitive assessment 2026-05-09 09:30

Temporal pattern recognised: Volume-contraction cascade
  • Diuretic escalation → reduced PO intake → volume contraction
  • Volume contraction → orthostatic instability → near-fall
  • Possible contributors to confusion: dehydration, possible lithium toxicity
    (lithium level overdue 117 days; reduced clearance with volume contraction)
  
Confidence: suggestive (chronology consistent with cascade pattern; 
            lithium level required for confirmation)
```

The chronology is rendered with audience adaptations:

**Pharmacist audience adaptation:**
Emphasises medication contribution (frusemide dose increase as cascade origin), surfaces intervention opportunity (lithium level, frusemide dose review, hydration support), connects to recommendation craft engine for action.

**RACH operational audience adaptation:**
Emphasises operational signals (near-fall, mobility decline, confusion onset requiring monitoring escalation), surfaces operational response opportunity (1:1 supervision consideration, family communication, GP coordination), connects to facility instability tracking.

**Governance audience adaptation:**
Emphasises temporal completeness for retrospective review (was the cascade detectable earlier?), surfaces audit trail for quality improvement, connects to intervention lifecycle tracking and outcome attribution.

### 3.3 What makes chronology distinctive

Three properties distinguish chronology from per-parameter trajectory rendering:

**Property 1: Cross-parameter composition.** Per-parameter trajectory shows eGFR over time. Chronology shows medication change at Day -8, intake decline at Day -6, orthostatic instability at Day -4, sedation at Day -2, near-fall and confusion at Day 0 — across different parameters, composed into a temporal narrative.

**Property 2: Temporal pattern recognition.** Chronology surfaces patterns like cascades, escalations, drifts, and destabilisations that are only visible when events are composed across parameters. A volume-contraction cascade is invisible in any single parameter; it requires the cross-parameter narrative.

**Property 3: Audience-neutral substrate plus audience-adapted rendering.** The events and patterns themselves are observations (audience-neutral); the rendering composes audience-specific framing on top. This is precisely the observation/action layer boundary specified in Section 2.

### 3.4 Why both surfaces consume chronology

The chronology is genuinely useful to all surfaces:

- **Pharmacist:** Sees the temporal pattern suggesting medication contribution; can integrate to specific clinical reasoning ("the cascade origin is the frusemide dose change; the intervention sequence is hydration support, lithium level, possibly frusemide dose review")
- **RACH operator:** Sees the operational pattern warranting escalation ("the cascade has produced near-fall and confusion; operational response includes monitoring escalation and family communication")
- **Governance:** Sees the chronology for retrospective review ("was the cascade detectable at Day -6 when intake decline was first noted? could earlier intervention have prevented the Day 0 events?")
- **Family communication:** The chronology provides the narrative structure families understand when receiving updates ("over the past two weeks, your mother's medication change was followed by these changes; we are investigating whether they are connected")
- **Audit defensibility:** ACQSC investigators and regulators reason in chronologies; the platform's audit responses align with this reasoning structure

The chronology is the single most powerful shared rendering primitive because it serves multiple audiences without requiring substrate divergence.

### 3.5 Computation responsibility

The chronology is computed by the CAPE engine as part of observation-layer outputs. Each surface consumes the chronology and applies audience-adapted rendering; surfaces do not compute their own chronologies.

The discipline matters: if pharmacist surface and RACH surface computed chronologies independently, they would inevitably diverge. The CAPE engine computes one chronology per resident per time window; surfaces render it differently.

### 3.6 What requires senior consultant pharmacist authoring

The chronology computation is structural — events with timestamps, severity, primitive types. The temporal pattern recognition is partly structural and partly clinical:

- **Structural pattern recognition** (engine work): Temporal proximity of events, severity escalation across events, primitive type combinations frequently associated with cascades
- **Clinical pattern recognition** (senior consultant pharmacist authoring): What constitutes a "volume-contraction cascade" vs. "anticholinergic cascade" vs. "infection-driven destabilisation" — the named pattern templates that the engine recognises

The clinical pattern library is authored by senior consultant pharmacists with clinical informatics support. The engine recognises matches to authored patterns; it does not invent pattern types from substrate alone.

This connects to the discipline established in the calibration discussion: clinical content is human-authored; structural support is engine work.

---

## Section 4 — Relationship to commercial sequencing and positioning

The architectural commitment shapes commercial decisions about sequencing and positioning. This section specifies the relationship.

### 4.1 Sequencing per v3.0 commercial strategy

Per v3.0 §5 and the Pilot Design document:

**Pilot Phase 1 (Months 1–8):** Pharmacist worklist surface. Anchor pilots with pharmacy chain enterprise tier buyers. ACOP funding flow established. CAPE engine substrate accumulates 6+ months of longitudinal data. RIR baseline established and target of +12 percentage points pursued.

**Pilot Phase 2 (Months 6–12, overlapping):** RACH operational view surface. Co-pilot engagement with RACH operators at sites where Phase 1 pharmacy chain is also deployed. "Fewer surprises" KPI baseline established (preventable hospital transfers, SIRS-reportable events, family complaints) and target reduction pursued. Standard 5 evidence panel integration.

**Phase 3 onwards (Post-pilot):** Additional surfaces as substrate maturity supports. GP coordination workspace, family portal, governance/regulator interface.

The sequencing is informed by both commercial reality (pharmacy chain ACOP funding is the strongest immediate wedge) and substrate maturity (Phase 1 pharmacist work accumulates the substrate that makes Phase 2 RACH detection clinically meaningful).

### 4.2 Why Phase 2 RACH is materially better than Phase 1 RACH would be

The reviewer's observation that "the pharmacist queue becomes your data acquisition wedge" is operationally important.

Phase 1 pharmacist deployment produces:
- Intervention attribution data (which interventions actually changed trajectories)
- Outcome closure data (which recommendations resulted in implemented changes with documented outcomes)
- Failed intervention history (which interventions were attempted and reversed, with reasoning)
- Trajectory validation (which substrate signals correlated with clinically meaningful events)
- Override patterns (which engine outputs the pharmacist overrode, why, with what alternative reasoning)

This data improves the substrate that the RACH operational view consumes in Phase 2. A Phase 2 RACH deployment without Phase 1 substrate accumulation would face the same cold-start problem that competitor deterioration AI platforms face — detection signals without intervention attribution, alerts without outcome data, patterns without validation.

The architectural commitment to one substrate makes this compounding possible. Two separate engines would not compound; one engine with two surfaces does.

### 4.3 The positioning architecture

The architectural commitment shapes commercial messaging in specific ways.

**For pharmacy chain commercial conversations:**
Lead with medication intelligence (CAPE pharmacist worklist; recommendation craft engine; RIR target; ACOP funding utilisation). The architectural truth is broader but the wedge is medication-clinical.

The commitment to RACH-future-compatibility is not a marketing point for pharmacy chains; it's an engineering commitment that protects them. Their concern is that their pharmacist data eventually serves a RACH operator's reporting, which could create awkward employer-of-record dynamics. The architectural commitment provides the answer: pharmacist clinical work is visibility-class-controlled (per pharmacist self-visibility module); the RACH surface consumes from substrate that respects visibility classes; aggregations to employer-of-record (the pharmacy chain) follow the algorithmic management protections in ethical architecture §8.

This means the architectural commitment to multi-surface consumption is also a trust commitment to pharmacy chains: their pharmacists are not being surveilled for RACH operator benefit; the surfaces serve different decision-makers with different observation scopes.

**For RACH operator commercial conversations:**
Lead with operational visibility and "fewer surprises" KPI. The architectural truth — that the RACH view consumes from the same substrate as the pharmacist worklist — is a commercial advantage to articulate, not a complexity to hide.

The articulation: "This is not a separate deterioration AI product grafted onto a pharmacist tool. The same substrate that supports the pharmacist's clinical work supports your operational visibility. When your pharmacist observes that Mrs Chen's medication change preceded the cascade that produced the near-fall, you see the same chronology, contextualised for your operational response. The pharmacist's intervention and your operational response are coordinated through shared observation, not through reporting that crosses organisational boundaries."

This positioning resists commoditisation because competitor RPM platforms don't have the medication-governance substrate; their detection is downstream of clinical reasoning that Vaidshala captures upstream.

### 4.4 Long-term positioning evolution

Year 1–2 commercial messaging leads with medication intelligence — "Medication intelligence platform for ACOP pharmacists and RACHs." The architectural truth is broader.

Year 3+ commercial messaging can evolve toward broader positioning — "Longitudinal resident stability orchestration" or "Clinical execution continuity infrastructure" — as substrate accumulation supports it.

The discipline: the architecture supports both positioning options; the commercial messaging follows substrate maturity, not architectural aspiration. The team should not market Year 3 positioning at Year 1; the architecture protects the option without committing to its premature deployment.

### 4.5 Positioning anti-patterns

Specific commercial messaging the team should avoid:

- **"Early deterioration AI"** — generic, RPM-like, commoditised. Doesn't reflect the medication-governance substrate.
- **"Pharmacist tool with reporting features"** — undersells the architecture, constrains the Year 3+ evolution.
- **"AI-driven medication management"** — generic, doesn't differentiate from competitor LLM deprescribing tools that the JMIR Aging 2025 research documents as having poor performance on complex cases.
- **"Predictive risk modelling for aged care"** — risks the failure modes of PADR-EC, BADRI, GerontoNet predictive overreach.

The differentiated positioning is some variant of: medication-governance-linked longitudinal instability infrastructure, with clinical reasoning continuity as the defensible moat. The specific wording is commercial team work; the architectural truth that underlies the wording is the commitment in this addendum.

---

## Section 5 — Anti-patterns and governance against divergent substrate

The architectural commitment is load-bearing only if it's defended against the pressures that would erode it. This section specifies the anti-patterns to reject and the governance that protects the commitment.

### 5.1 The seven anti-patterns

**Anti-pattern 1: Surface-specific observation generation.**
A surface (typically RACH operational view) needs an observation type that the substrate doesn't currently capture. Local solution: detect it within the surface itself, write to surface-private storage. Architectural correct response: extend the substrate so the observation is canonical and available to all surfaces.

Example: RACH operational view wants to detect "family escalation risk" based on family contact patterns. Wrong: build family-contact-monitoring inside the RACH surface. Right: extend the Operational state machine to capture family contact events and frequencies as substrate observations; both surfaces consume from there.

**Anti-pattern 2: Surface-specific scoring overlays.**
A surface wants a different priority ordering than what CAPE produces. Local solution: re-score CAPE outputs within the surface. Architectural correct response: surface the underlying signals to the user with audience-appropriate emphasis; the user's interpretation legitimately differs by surface.

The CAPE engine produces five-layer scoring; the pharmacist worklist emphasises Intervention Opportunity at 25%; the RACH operational view might legitimately emphasise Resident Stability + Complexity differently for facility-level pattern detection. This is acceptable as audience adaptation in the action layer. What is not acceptable is the RACH surface computing its own scoring that contradicts CAPE's scoring on the same resident — different audience emphasis is fine; contradictory underlying scores are not.

**Anti-pattern 3: Surface-specific severity calibrations.**
A surface wants different severity thresholds for the same signal. Local solution: re-classify severity within the surface. Architectural correct response: the severity calibration is observation-layer; if a surface needs different sensitivity, it's an audience adaptation (which signals to surface preferentially) not a severity re-classification.

If `fall_no_injury_recent` is severity 3 in the substrate, both surfaces see severity 3. The pharmacist surface might deprioritise it relative to other Layer 1 signals; the RACH surface might prioritise it for operational response. The underlying severity is identical; the surface emphasis legitimately differs.

**Anti-pattern 4: Bypass of EvidenceTrace for surface-specific audit trails.**
A surface wants its own audit trail with surface-specific structure. Local solution: write surface-private audit records. Architectural correct response: extend EvidenceTrace with surface-specific edge types if needed; the audit trail is canonical.

This protects against the failure mode where pharmacist clinical decisions are audit-recorded one way and RACH operational decisions audit-recorded differently, making cross-stakeholder review impossible. Both surfaces write to the same audit substrate with surface-specific edge classifications.

**Anti-pattern 5: Visibility class violations through surface-specific aggregation.**
A surface wants to aggregate data in ways that bypass pharmacist self-visibility controls. Local solution: aggregate at the surface layer where visibility checks are less rigorous. Architectural correct response: visibility class enforcement is observation-layer; aggregations respect visibility constraints regardless of which surface requests them.

This protects against the failure mode where the RACH operational view inadvertently exposes pharmacist-private clinical work to facility operators in ways that violate algorithmic management protections per ethical architecture §8.

**Anti-pattern 6: Independent ML/AI overlays per surface.**
A surface wants its own predictive model trained on surface-specific outcomes. Local solution: build an ML model that overlays surface-specific predictions on top of CAPE outputs. Architectural correct response: substrate accumulation produces the data that informs maturity roadmap Phase 4 capabilities; the engine evolves, not the surfaces independently.

The temptation here is real because each surface has its own success metric (pharmacist surface tracks intervention opportunity realisation; RACH surface tracks "fewer surprises"; governance surface tracks closed-loop intervention rate). Independent ML overlays would optimise each surface's metric in isolation. The architectural discipline keeps the substrate canonical and evolves the engine to serve multiple optimisation targets coherently.

**Anti-pattern 7: Documentation drift between surfaces.**
A surface's user-facing documentation describes how the surface "computes" or "predicts" things in ways that disagree with how other surfaces describe the same observations. Local solution: each surface team owns its own documentation. Architectural correct response: documentation of observation-layer outputs is canonical; surfaces document their rendering and workflow integration on top of that.

This is the documentation equivalent of divergent substrate. It happens easily because surface teams write user-facing materials independently. The discipline is that observation-layer language (what signals exist, what they mean, what substrate references support them) is shared across surface documentation.

### 5.2 Governance for substrate extensions

When a surface needs an observation type the substrate doesn't capture, the architectural commitment requires substrate extension rather than surface-private detection. This needs governance to prevent the substrate from accumulating poorly-scoped observations.

**Substrate extension request process:**

1. Surface team identifies the missing observation type. Documents:
   - What observation is needed
   - Which surface needs it (initial requester)
   - What substrate state machine it would extend (Clinical, Operational, Consent, Care Intensity, Goals-of-Care)
   - What computation produces it
   - What audience adaptations are anticipated

2. Cross-surface validation: would other surfaces find this observation useful? At minimum, the pharmacist surface team and the RACH operational view team review the proposal. Substrate observations that serve only one surface should still be added (canonical observations don't have to serve all surfaces equally) but the cross-surface review checks for naming alignment, severity calibration consistency, and audience adaptation patterns.

3. Clinical informatics review: does this observation type align with the canonical instability primitive vocabulary? Is the severity calibration consistent with existing primitives? Does the substrate extension introduce risks (privacy, consent, visibility class)?

4. Ethics review: substrate extensions that touch new data types (e.g., family communication, behavioural monitoring) require Ethics Steering Committee review per ethical architecture §9.

5. Implementation: substrate extension lands as a regular substrate change; both surfaces (and any future surfaces) consume from the canonical observation.

This governance is operationally heavier than letting surfaces build their own detection. The cost is intentional. Substrate divergence is more expensive to recover from than substrate extension governance is to operate.

### 5.3 CI tests that enforce the commitment

The architectural commitment is enforced at code level through CI tests.

**Test category 1: Cross-surface observation consistency.**

```go
func TestCrossSurfaceObservationConsistency(t *testing.T) {
    resident := generateTestResident()
    addInstabilityEvents(resident, []Event{...})
    
    pharmacistView := pharmacistService.GenerateWorklist(pharmacistID, residentID)
    rachView := rachService.GenerateFacilityOverview(facilityID, donID)
    governanceView := governanceService.GenerateAuditView(facilityID, auditQuery)
    
    // The same resident's observations must appear identically in all surfaces
    pharmacistObs := pharmacistView.GetResidentObservations(residentID)
    rachObs := rachView.GetResidentObservations(residentID)
    govObs := governanceView.GetResidentObservations(residentID)
    
    require.Equal(t, pharmacistObs.Timestamps, rachObs.Timestamps)
    require.Equal(t, rachObs.Timestamps, govObs.Timestamps)
    require.Equal(t, pharmacistObs.SubstrateRefs, rachObs.SubstrateRefs)
    require.Equal(t, pharmacistObs.Severities, rachObs.Severities)
    
    // Different rendering is expected
    require.NotEqual(t, pharmacistView.Rendering, rachView.Rendering)
}
```

**Test category 2: No surface-private observation writes.**

```go
func TestSurfacesDoNotWriteObservations(t *testing.T) {
    // Surface services must not have write access to observation storage
    pharmacistRoles := pharmacistService.GetRolePermissions()
    require.NotContains(t, pharmacistRoles, "observation_write")
    
    rachRoles := rachService.GetRolePermissions()
    require.NotContains(t, rachRoles, "observation_write")
    
    // Only the substrate and CAPE engine write observations
    substrateRoles := substrateService.GetRolePermissions()
    require.Contains(t, substrateRoles, "observation_write")
    
    capeRoles := capeService.GetRolePermissions()
    require.Contains(t, capeRoles, "observation_write")
}
```

**Test category 3: Chronology rendering coherence.**

```go
func TestChronologyCoherenceAcrossSurfaces(t *testing.T) {
    resident := generateTestResidentWithCascade()
    chronology := capeService.GetInstabilityChronology(resident.ID, last14Days)
    
    pharmacistRender := pharmacistService.RenderChronology(chronology, pharmacistAudience)
    rachRender := rachService.RenderChronology(chronology, rachAudience)
    
    // Events and timestamps must match
    require.Equal(t, pharmacistRender.Events, rachRender.Events)
    
    // Audience-adapted framing legitimately differs
    require.NotEqual(t, pharmacistRender.Framing, rachRender.Framing)
    
    // But the substrate references must be identical
    for i := range pharmacistRender.Events {
        require.Equal(t, 
            pharmacistRender.Events[i].SubstrateRefs,
            rachRender.Events[i].SubstrateRefs)
    }
}
```

**Test category 4: Visibility class enforcement across surfaces.**

```go
func TestVisibilityClassEnforcedAcrossSurfaces(t *testing.T) {
    // PDP-classified observations must not leak through any surface aggregation
    observation := generatePDPClassifiedObservation()
    
    employerView := rachService.GenerateEmployerAggregateView(employerID)
    require.NotContains(t, employerView.Observations, observation.ID)
    
    facilityView := rachService.GenerateFacilityOverview(facilityID, donID)
    // DON may see facility-level patterns but not pharmacist-private clinical decisions
    require.NotExposesIndividualPharmacistDecisions(t, facilityView)
}
```

These tests are load-bearing. They are not optional, they are not advisory, and they are not relaxable under product pressure. The architectural commitment is what they enforce; relaxing them is relaxing the commitment.

### 5.4 Governance checkpoints

Three governance checkpoints protect the commitment over time.

**Quarterly architecture review:** Engineering leads + clinical informatics + Ethics Steering Committee review substrate extensions, CI test results, and any architectural pressure points. Surface-private observation requests that were rejected are reviewed to confirm rejection was appropriate. Substrate extensions that landed are reviewed for cross-surface utility.

**Annual external architecture audit:** Per ethical architecture §12, external clinical informatics review includes architectural commitment review. The auditor specifically examines whether divergent substrate has accumulated, whether surfaces have built private detection, whether documentation drift has occurred.

**Pre-Phase 4 architecture audit:** Before maturity roadmap Phase 4 capabilities (predictive modelling, adaptive composition) are implemented, a focused architectural audit confirms the substrate is canonical and the engine evolution serves all surfaces coherently. Phase 4 work is gated on this audit.

---

## Closing

This addendum specifies the architectural commitment that the platform observes resident instability through a single longitudinal substrate, with multiple operational surfaces rendering observations for different audiences. The commitment is not stylistic; it shapes engineering decisions, commercial messaging, and long-term moat formation.

Three observations to close.

**One:** The two intervention horizons framing — pharmacist intervention horizon and RACH operational horizon, operating on the same instability graph — is the precise architectural relationship between the surfaces. They are not two products; they are two action layers consuming from one observation layer. The commitment in writing protects against the engineering pressure to build them as separate engines, which would produce divergent substrate and erode the platform's defensible moat.

**Two:** The instability chronology as first-class shared primitive is the most powerful rendering artefact across all surfaces. Cross-parameter event composition with temporal pattern recognition serves pharmacist clinical reasoning, RACH operational response, governance retrospective review, family communication, and audit defensibility. Computing it once in the engine and rendering it differently per surface is the canonical example of the observation/action layer boundary working correctly.

**Three:** The architectural commitment makes the platform's category positioning honest. Without the commitment, "longitudinal resident stability infrastructure" is aspirational marketing on top of pharmacist software. With the commitment, the architecture matches the positioning — substrate is built once and serves multiple stakeholders coherently. Year 1–2 commercial messaging leads with medication intelligence (the strongest wedge); Year 3+ commercial messaging can evolve as substrate maturity supports it. The architectural commitment protects the option.

The architecture stack now stands at ten documents, with this addendum supplementing CAPE v1.1:

1. v3.0 strategic positioning
2. Pilot design
3. Recommendation craft engine
4. Pharmacist self-visibility
5. Ethical architecture
6. Decision packet rendering
7. KB-29 templates
8. KB-29 maturity roadmap
9. Clinical Attention Prioritisation Engine (CAPE) v1.1
10. **CAPE v1.1 Architectural Commitment to Multi-Surface Consumption** ← this addendum

What this addendum does not yet specify, and what should be subsequent work:

- The RACH Operational View Implementation Guidelines (subsequent specification after Phase 1 pilot evidence accumulates)
- The Governance Workspace Implementation Guidelines (subsequent specification when intervention lifecycle data accumulates)
- The Family Communication Surface Implementation Guidelines (subsequent specification after RACH surface stabilises)
- The clinical pattern library for instability chronology recognition (senior consultant pharmacist authoring work)
- The substrate extension governance operational runbook (governance work alongside Ethics Steering Committee charter)

These are subsequent deliverables. This addendum's scope is the architectural commitment that makes them possible. The substrate decision (one engine, multiple surfaces) is documented in writing; the surfaces' specifications follow as evidence and commercial sequencing inform them.

— Claude
