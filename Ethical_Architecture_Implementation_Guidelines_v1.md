# Ethical Architecture — Implementation Guidelines v1.0

**Date:** May 2026
**Service scope:** Cross-cutting, applied to all Vaidshala services (`kb-30-authorisation-evaluator`, `kb-31-scope-rules`, `kb-32-recommendation-craft`, `pharmacist-self-visibility`, and all subsequent services)
**Implementation phase:** Phase 1 of Layer 2/3 plan (Weeks 5–10), with extensions for governance committee, ethics-based auditing operations, and incident response
**Builds on:** *Vaidshala v3.0 Product Proposal* §9 (seven principles), *Recommendation Craft Implementation Guidelines v1.0* (clinical safety architecture), *Pharmacist Self-Visibility Implementation Guidelines v1.0* (trust architecture for one module), *Layer 2 & 3 Implementation Plan* (7 May 2026)

**Reading order:** Engineering and product leads read Parts 1–4 (architecture, principles operationalised, three-layer model, cross-cutting metadata). Clinical leads read Parts 5–7 (resident vulnerability, capacity and consent, bias and equity). Legal and ethics leads read Parts 8–11 (governance, ethics-based auditing, incident response, external review). Implementers read Parts 12–17 (cross-cutting metadata, file structure, contracts, tests, sequencing, risks).

---

## Part 0 — Honest framing: what ethical architecture is and isn't

Three failure modes recur in technology projects that produce ethics documents.

**Failure mode 1: Aspirational manifesto.** The document articulates principles eloquently but engineers cannot implement against it. The principles do not constrain product decisions because they are too abstract. The result is principles disconnected from product behaviour, with team members citing the document selectively when convenient.

**Failure mode 2: Checkbox compliance.** The document becomes a checklist that gets implemented narrowly without the underlying intent. Each checkbox is satisfied in form while the spirit is violated in operation. The result is a system that passes ethics review but produces ethically problematic outcomes.

**Failure mode 3: Single-point-in-time governance.** The document is reviewed at deployment but not continuously. Drift over time is undetected because no continuous auditing process exists. The result is a system whose ethical posture degrades as the world changes around it.

This document is operational, intent-traceable, and continuous. Each commitment specifies what gets built, what gets measured, what gets escalated, and how drift is detected. The principles are not aspirational; they are architectural constraints with implementation specifications. The document is not a checklist; it is a structure for ongoing governance.

### What's established in the literature we adopt

The 2024–2026 healthcare AI ethics literature has converged on several established frameworks we adopt:

- **Ethics-by-design framework for medical AI agents (Bisson et al., 2026).** Six interventions: auditable ethical reasoning modules, explicit human override conditions, structured patient preference profiles, AI-specific ethics oversight tools, global benchmarking repositories, regulatory sandboxes. We adopt the framework with adaptations for aged care medication management.

- **Health-system-level AI governance (Liao et al., 2022, University of Wisconsin Health).** Multi-disciplinary steering committee with project-specific sub-committees, multi-stakeholder perspective spanning informatics, data science, clinical operations, ethics, and equity. Tangible parameters for endorsement of both initial deployment and ongoing usage. We adopt this governance structure.

- **Ethics-based auditing as continuous process (Mökander & Floridi, 2021; Mökander et al., 2024).** EBA bridges principles to practice through structured continuous assessment. Best practice: continuous and constructive process, system perspective, aligned with public policies. We adopt EBA as our continuous governance mechanism.

- **AI-specific governance for aging populations (Abbasian, 2026, Harvard Center for Bioethics).** Epistemic opacity of ML models, distributed clinical responsibility, dynamic consent for passive data collection, equitable performance across diverse populations. We adopt these as specific concerns for our aged-care context.

- **Algorithmic management worker-protective frameworks (Bowdler 2026; Vignola 2023).** Already adopted in the pharmacist self-visibility module; extended here to all modules.

### What's distinctive in Vaidshala's ethical architecture

Five elements distinguish this architecture from generic clinical AI ethics frameworks:

**Distinctive 1: Frame-vs-content separation as architectural commitment.** Most clinical AI systems either deploy uniform language or adapt language without auditable separation. We separate clinical content (invariant) from framing (audience-adapted) at the data structure level, with content_hash invariance asserted on every release. This prevents the platform from being accused of varying clinical advice by audience.

**Distinctive 2: Restraint as an ethical commitment, not just a product feature.** Most clinical AI ethics frameworks focus on transparency and accountability for actions taken. We explicitly architect for actions *not* taken — surfacing context arguing for non-intervention, supporting "watchful wait" recommendations, and treating restraint as a first-class ethical answer.

**Distinctive 3: The reviewer pyramid as governance substrate.** Most ethics frameworks position oversight as one-directional (governance reviews the system). We position the platform as participant in a multi-layer review structure where each actor reviews and is reviewed. The platform's audit trail supports every layer's reviewing function.

**Distinctive 4: Self-visibility before aggregation as ethical inversion.** Most workplace performance systems treat workers as data subjects of their employer. We treat the pharmacist as the data subject and the employer as a contractually-permitted observer. This is unusual in workplace performance management generally.

**Distinctive 5: Aged-care vulnerability as primary ethical concern.** Our patient population is one of the most vulnerable in healthcare — frail, often cognitively impaired, often without family advocacy, frequently subject to restrictive practices. Generic clinical AI ethics frameworks under-address vulnerability. We make it primary.

These distinctive elements are not separate from the established frameworks; they extend them for our specific context.

### What this document is not

- It is not a substitute for clinical safety architecture (specified in craft engine guidelines)
- It is not a substitute for trust architecture (specified in self-visibility guidelines)
- It is not a regulatory compliance document (though it informs compliance)
- It is not a marketing or external positioning document
- It is not a one-time review artefact (it is continuously updated)

It is the cross-cutting layer that all modules inherit from and that the governance structure reviews continuously.

---

## Part 1 — The seven principles operationalised

v3.0 §9 established seven principles. This section specifies the operational mechanism for each.

### Principle 1: Frame adapts, content invariant

**Operationalisation:**
- Two-layer EvidenceTrace recording: `clinical_content` field and `framing_adaptation` field, separately queryable
- `content_hash` computed on clinical content; invariant assertion on every recommendation render
- Regulator audit query API: "show me clinical content for recommendation X across all framings" returns content with hash equality
- CI test: `frame_vs_content_invariance_test` blocks any release where content varies across framings

**Detection of violation:** Content hash mismatch across framings of the same recommendation. Audit log query for any recommendation showing content variation.

**Escalation:** Immediate hold of the affected recommendation; clinical informatics review within 48 hours; root-cause analysis; remediation before any further recommendations using the affected pattern.

**Owner:** Clinical informatics lead.

### Principle 2: Acceptance follows appropriateness

**Operationalisation:**
- Five-dimension appropriateness rubric scored on every recommendation (per craft engine §9)
- Paired tracking: every recommendation acceptance/rejection is recorded with the prior appropriateness score
- Pattern detection: recommendations with high acceptance + low appropriateness flagged
- Pattern detection: rules with rising acceptance trajectories must show concomitant appropriateness trajectories

**Detection of violation:** Acceptance-appropriateness divergence pattern. Any rule where 30-day rolling acceptance rises ≥10pp without appropriateness rising in parallel.

**Escalation:** Rule held for clinical review; pharmacist using the rule notified; clinical informatics review within 7 days.

**Owner:** Clinical informatics lead.

### Principle 3: Restraint is a clinical answer

**Operationalisation:**
- Restraint signal detector queries substrate for nine signal types (per craft engine §10)
- Restraint signals surfaced alongside action recommendations in UI
- Pharmacist override of restraint signal captured with reasoning
- Aggregate analysis of restraint override patterns

**Detection of failure:** Restraint signals firing but consistently overridden without documented reasoning; or restraint signals not firing in cases where retrospective review suggests they should have.

**Escalation:** Pattern review by clinical informatics; restraint signal definitions reviewed quarterly.

**Owner:** Clinical informatics lead.

### Principle 4: Pharmacist autonomy preserved

**Operationalisation:**
- Algorithmic-vs-human distinction in EvidenceTrace (4-class: substrate-fact / platform-suggestion / pharmacist-reflection / hybrid)
- Pharmacist override pathway available on every algorithmic suggestion
- Override reasons captured in structured taxonomy
- AHPRA accountability preserved: pharmacist's clinical judgment is the ultimate authority on each recommendation

**Detection of violation:** Algorithmic suggestions implemented as recommendations without pharmacist confirmation; override pathways unavailable or hidden; suggestions surfaced as directives rather than observations.

**Escalation:** UI review; pharmacist feedback collection; design correction.

**Owner:** Product lead.

### Principle 5: Self-visibility before aggregation

**Operationalisation (per pharmacist self-visibility guidelines):**
- Five visibility classes: POA, PDP, PFA, WO, AD
- Query-layer enforcement of visibility classes
- Temporal-order commitment: pharmacist sees own data 30 days before employer aggregation
- Consent model with purpose-bounded, time-bounded, revocable consent

**Detection of violation:** Aggregation occurring before pharmacist view; consent boundary violations; re-identification through small-subset queries.

**Escalation:** Immediate suspension of the violating query path; pharmacist notification; remediation; affected pharmacists offered contestation pathway.

**Owner:** Privacy/ethics lead.

### Principle 6: Reviewability of platform itself

**Operationalisation:**
- Every algorithmic decision logged in EvidenceTrace with full reasoning trace
- Layer 4 deep audit available on every recommendation
- External review mechanisms: clinical informatics consultants, ethics committee, regulator data-sharing agreements
- Annual external ethics audit (per Part 10)

**Detection of failure:** Algorithmic decisions without audit trail; reasoning traces incomplete; external review prevented or limited.

**Escalation:** Engineering review of audit completeness; correction within 30 days.

**Owner:** Engineering lead.

### Principle 7: GP authority strengthened, not routed around

**Operationalisation:**
- Recommendation packets framed as collaborative clinical input, not directives
- GP communication channels respected (no routing around the GP for prescribing decisions)
- Time-saving language used (not bottleneck or delay implications)
- Audit trail strengthens medico-legal protection for GPs

**Detection of violation:** Communication patterns implying GP bypass; recommendations skipping GP authorisation; audit trail used against GPs rather than supporting them.

**Escalation:** Communication review; pattern correction; RACGP relationship preserved per v3.0 Move 3.

**Owner:** Clinical informatics lead + commercial lead.

---

## Part 2 — The three-layer ethical architecture

The architecture operates at three layers, each with distinct mechanisms and timing.

### Layer A: Preventive

Mechanisms that prevent ethical violations from occurring.

- **Architectural constraints:** visibility class enforcement at query layer; content hash invariance at render; appropriateness threshold blocking at draft transition
- **Design principles in UI/UX:** observation language, not directive language; restraint signals surfaced alongside action; pharmacist confirmation required for algorithmic suggestions
- **Data structure constraints:** five-class visibility metadata on every data element; algorithmic-vs-human distinction on every observation
- **Code review checklists:** explicit visibility class annotation required; algorithmic decision logging required
- **CI/CD gates:** frame-vs-content invariance tests; privacy boundary tests; algorithmic determination tests must pass before release

### Layer B: Detective

Mechanisms that detect ethical violations after they occur.

- **Pattern detection:** acceptance-appropriateness divergence; restraint override patterns; suppression patterns; surveillance patterns; re-identification risks
- **Audit trail queries:** continuous monitoring for visibility class violations, override pattern anomalies, content variation across framings
- **External signal collection:** pharmacist contestations, GP feedback, RACH operator concerns, regulator inquiries, family complaints
- **Anonymous reporting mechanism:** internal team can flag concerns via anonymous channel
- **Quarterly ethics review:** structured review of patterns, contestations, escalations

### Layer C: Corrective

Mechanisms that respond to detected violations.

- **Immediate response protocols:** hold affected component; notify affected parties; preserve evidence; engage incident response
- **Remediation pathways:** technical correction; methodology revision; design change; governance review
- **Affected-party support:** contestation pathway active; independent review available; transparency about resolution
- **Learning loop:** every incident produces structured learning entry; quarterly review of patterns informs preventive layer
- **External communication:** when incidents are material, transparent communication with affected pharmacists, employers, regulators

The three layers form a continuous cycle: preventive constraints reduce incidence; detective mechanisms surface what gets through; corrective mechanisms remediate and feed back into preventive design.

---

## Part 3 — Cross-cutting ethical commitments by module

Each module inherits ethical commitments from this architecture. This section specifies module-specific operationalisation.

### kb-30 Authorisation Evaluator

**Inherited principles:** All seven, with particular emphasis on Principle 6 (reviewability) and Principle 4 (pharmacist autonomy).

**Module-specific commitments:**
- Authorisation decisions are auditable end-to-end
- Override pathway available for clinical urgency cases (with documented reasoning)
- Authorisation rules transparent to affected parties (not black-box)
- Performance: latency budget respects clinical workflow

### kb-31 ScopeRules Engine

**Inherited principles:** All seven, with particular emphasis on Principle 7 (GP authority) and Principle 6 (reviewability).

**Module-specific commitments:**
- Scope determinations transparent and auditable
- Jurisdictional variations explicit (no hidden state)
- Updates to scope rules logged with effective dates
- Pharmacist visibility into scope-related decisions

### kb-32 Recommendation Craft Engine

**Inherited principles:** All seven, with particular emphasis on Principles 1, 2, 3 (already specified in craft engine guidelines).

**Module-specific commitments per craft engine guidelines:**
- Frame-vs-content separation in EvidenceTrace
- Appropriateness paired with acceptance
- Restraint signals as first-class
- Override-reason taxonomy with appropriateness pairing
- Citation versioning with effective-date semantics
- Negative-evidence citation patterns with substrate-backing

### Pharmacist Self-Visibility Module

**Inherited principles:** All seven, with particular emphasis on Principles 5, 6, and the algorithmic management protections (per self-visibility guidelines).

**Module-specific commitments per self-visibility guidelines:**
- Five visibility classes
- Temporal-order commitment
- Contestation pathway
- Cross-employer portability
- Development-not-evaluation framing

### Future modules

Every new module's design specification must include an "Ethical commitments" section that:
- Identifies which principles apply with particular emphasis
- Specifies module-specific operationalisation
- Specifies detection mechanisms for violations
- Specifies escalation paths
- Names the responsible owner

This section is a required gate for any new service entering the platform.

---

## Part 4 — The Ethical Reasoning Module (ERM)

Adapted from Bisson et al. (2026) "Six Interventions for Responsible and Ethical Implementation of Medical AI Agents." The ERM is a structural component of the platform that reviews other modules' decisions from an ethical viewpoint.

### 4.1 ERM scope

The ERM reviews:
- Recommendation drafts before transition to `drafted` state (acts in concert with the craft engine appropriateness check)
- Visibility class assignments and aggregation queries
- Authorisation decisions for cases involving consent gating
- ScopeRules determinations for jurisdictionally-novel cases
- Any algorithmic suggestion to a pharmacist that could feed performance evaluation

The ERM does not replace the clinical appropriateness check or the visibility class enforcer — it operates alongside them, providing structured ethical review at decision points.

### 4.2 ERM architecture

```go
// /shared/v2_substrate/erm/module.go
type EthicalReasoningModule struct {
    decisionPoints   []DecisionPoint
    reasoners        map[DecisionType]Reasoner
    auditLog         AuditLogger
    escalationPaths  map[ConcernLevel]EscalationPath
}

type DecisionPoint struct {
    Component       string  // which service is making the decision
    DecisionType    DecisionType
    Inputs          interface{}
    ProposedOutput  interface{}
    EthicalConcerns []EthicalConcern
}

type EthicalConcern struct {
    Principle       string  // which of the 7 principles is implicated
    ConcernLevel    int     // 1-5, with 5 most severe
    Reasoning       string
    Recommendation  string  // approve / hold / escalate / reject
}
```

### 4.3 ERM decision modes

For each decision point, the ERM produces one of four outcomes:

- **Approve:** No ethical concerns; decision proceeds
- **Approve with monitoring:** Approved but flagged for pattern detection; logged for quarterly review
- **Hold for review:** Decision held; clinical informatics or ethics review required before proceeding
- **Reject:** Decision blocked; reasoning logged; affected parties notified

### 4.4 ERM auditability

Every ERM decision is logged in EvidenceTrace with:
- Inputs reviewed
- Reasoning trace
- Outcome
- Reviewer identity (algorithm version + human if escalated)
- Timestamp
- Subsequent action by the calling component

This makes the ERM itself reviewable — its decisions can be audited, contested, and refined.

### 4.5 ERM evolution

The ERM is not static. Quarterly review of ERM patterns:
- Were holds appropriate or over-cautious?
- Were approvals appropriate or under-cautious?
- Are there patterns suggesting ERM rule refinement?
- Are there novel ethical concerns the ERM doesn't yet recognise?

ERM rule changes follow the same governance as other rule changes (per Layer 3 v2 spec).

### 4.6 ERM not as full automation

Critical commitment: the ERM does not make ethical judgments autonomously. It identifies decisions that require ethical attention, applies established review patterns, and escalates when patterns are unclear or stakes are high. Human judgment remains the final ethical authority for any non-routine case.

---

## Part 5 — Resident vulnerability and capacity-related considerations

Aged care residents are among the most vulnerable patient populations in healthcare. Generic clinical AI ethics frameworks under-address vulnerability. This section makes vulnerability primary.

### 5.1 The vulnerability dimensions

Aged care residents commonly experience:
- Cognitive impairment (dementia, delirium, depression)
- Frailty (physical and cognitive)
- Polypharmacy (often >5 active medications)
- Limited family advocacy (geographically distant or unavailable)
- Communication barriers (hearing, vision, language, cognition)
- Power imbalance with care providers
- Restrictive practices in some cases (chemical, physical, seclusion)
- Approaching end of life

Each dimension intensifies the ethical stakes of platform decisions affecting that resident.

### 5.2 Vulnerability-aware design commitments

**Commitment 1: Goals-of-care primacy.** Every recommendation considers the resident's documented goals-of-care. Where care intensity is "comfort" or "palliative," the recommendation patterns shift dramatically (per care_intensity tag in Clinical state machine). The platform respects these transitions.

**Commitment 2: Family/SDM involvement.** For residents lacking capacity, the SubstituteDecisionMaker (SDM) is involved per consent state machine. The platform supports SDM communication patterns specifically.

**Commitment 3: Recently-deteriorating residents handled with restraint.** When the substrate detects rapid clinical deterioration, the platform's recommendation pattern shifts toward stability rather than optimisation. Substrate signals: care_intensity transition, frailty score deterioration, recent hospitalisation, family-distress markers.

**Commitment 4: Restrictive practice consent gating.** Recommendations involving psychotropic medications, physical restraints, or seclusion are gated on Consent state. The platform supports the regulatory consent process; it does not bypass it.

**Commitment 5: End-of-life pattern recognition.** When clinical signals suggest the resident is approaching end of life, the platform's recommendation patterns shift toward comfort, dignity, and family presence rather than optimisation. Specific recommendation classes (e.g., new statins, stringent blood pressure targets) are suppressed by default in late-stage frailty.

### 5.3 The "vulnerable subject" data flag

Every Resident entity carries a vulnerability assessment computed from substrate signals:

```go
type VulnerabilityAssessment struct {
    CognitiveCapacity      string  // "intact" / "mild_impairment" / "moderate_impairment" / "severe_impairment" / "uncertain"
    FrailtyTier            string  // CFS-based
    CareIntensity          string  // "active" / "comfort" / "palliative" / "end_of_life"
    SDMRequired            bool
    FamilyAdvocacyPresent  bool
    RestrictivePractice    bool
    RecentDeterioration    bool
    AssessedAt             time.Time
}
```

This assessment is consumed by the ERM and by the craft engine to adapt recommendation patterns appropriately. It is not a label — it is a structured context that shifts what the platform considers appropriate.

### 5.4 Avoiding ageist or ableist patterns

The platform must not encode patterns that:
- Assume diminished value of older lives in optimisation
- Suppress all proactive care for older residents (under-treatment is also harm)
- Treat cognitive impairment as licence to bypass autonomy
- Treat frailty as licence to disregard quality of life

The balance is restraint plus respect for the resident's continued personhood and remaining autonomy.

---

## Part 6 — Substituted decision-making and consent for aged care

Consent in aged care is structurally different from consent in general clinical contexts. This section specifies the platform's consent architecture.

### 6.1 Three consent contexts

**Context 1: Resident with capacity.** The resident makes their own decisions. Platform respects resident autonomy directly.

**Context 2: Resident with fluctuating capacity.** Capacity may vary day-to-day or with delirium episodes. Platform supports capacity assessment integration; consent is captured during periods of capacity for ongoing situations.

**Context 3: Resident lacking capacity, SDM involved.** Substitute Decision Maker (typically family member or appointed guardian) makes decisions on the resident's behalf. Platform supports SDM communication and decision capture.

### 6.2 Consent state machine

Per v3.0 §3 and the Phase 0.2 implementation, the Consent entity carries:

```
requested → discussed → granted / refused / conditions → active → under_review → withdrawn / expired
```

States and their ethical implications:

- `requested`: consent initiated; not yet effective
- `discussed`: consent process underway; SDM/resident considering
- `granted`: consent in place; recommendation can proceed
- `refused`: consent declined; recommendation cannot proceed; alternative considered
- `conditions`: consent conditional (e.g., "trial for 6 weeks then re-evaluate"); platform tracks conditions
- `active`: consent currently effective
- `under_review`: consent due for re-evaluation
- `withdrawn`: consent revoked; ongoing recommendation must stop
- `expired`: consent timed out; renewal required

### 6.3 Restrictive practice consent

Australian aged care regulation requires specific consent processes for restrictive practices (chemical restraint, physical restraint, environmental restraint, seclusion). Platform supports:
- Assessment of less-restrictive alternatives (documented before restrictive practice)
- Behaviour Support Plan integration
- SDM consent process with required information
- Time-bounded consent (typically 12 weeks for chemical restraint)
- Mandatory review by designated practitioner
- ACQSC reportable incident handling for non-compliant cases

Recommendations involving these practices are gated on Consent state in `active` for the specific practice.

### 6.4 Dynamic consent for passive data collection

Per Abbasian (2026) on aged care AI ethics: dynamic consent is essential for passive longitudinal data collection. The platform captures observations continuously; the consent for this collection must be:
- Initial: at deployment, with SDM where applicable
- Renewable: annual at minimum, more frequently if circumstances change
- Granular: specific consent for cognitive monitoring, behavioural observation, biometric data
- Revocable: SDM/resident can withdraw consent for specific data types

### 6.5 SDM communication patterns

When the SDM is the decision-maker, platform communication adapts:
- Recommendation packets framed for SDM audience (not clinical-jargon-heavy)
- Time given for SDM consideration (no urgency framing unless clinically warranted)
- Family meeting integration (recommendations may wait for scheduled family meetings)
- Multiple-SDM scenarios handled (family disagreements documented; not platform-resolved)

### 6.6 Capacity assessment integration

Per Phase 1.2 implementation (Layer 2 capacity_assessment.go), the substrate carries capacity assessment outcomes. The ERM consumes this data:
- Recommendations involving consent are gated on current capacity assessment
- Capacity transitions (e.g., new diagnosis of dementia) trigger consent re-evaluation
- Capacity uncertainty triggers conservative defaults (assume SDM required)

---

## Part 7 — Bias mitigation and equity

Algorithmic bias in healthcare AI is well-documented. Per the JAMIA 2025 scoping review and the Tribulsi (Dartmouth, 2024) guidance: algorithmic bias can worsen healthcare inequities. This section specifies the platform's bias mitigation architecture.

### 7.1 Sources of potential bias in the platform

**Source 1: Substrate data bias.** The substrate reflects what's documented in source systems (eNRMC, care management, hospital discharge). If those systems documented some populations less consistently, the platform inherits the bias.

**Source 2: Rule authoring bias.** CQL rules are authored by clinical informatics specialists. Their training, experience, and perspective shape what rules fire. Systematic blind spots are possible.

**Source 3: Evidence base bias.** The Australian Deprescribing Guideline 2025, Beers Criteria, STOPP/START — all are developed predominantly with reference to particular populations. Application to populations under-represented in the evidence base may produce inappropriate recommendations.

**Source 4: Framing learning bias.** Per-GP framing learning (per craft engine §8) could learn patterns that amplify implicit biases in GP decision-making.

**Source 5: Outcome measurement bias.** RIR aggregation could systematically disadvantage pharmacists working with more complex cohorts.

### 7.2 Bias detection mechanisms

**Mechanism 1: Demographic stratification of metrics.** RIR, appropriateness scores, override patterns, and outcomes are stratified by:
- Resident age band (65–74, 75–84, 85+)
- Resident sex
- Resident frailty tier
- Cultural and linguistic background (where documented)
- Socioeconomic indicator (where available)
- Facility type and geography

Material disparities flagged for review.

**Mechanism 2: Rule outcome equity audit.** Quarterly review of which rules fire most frequently for which populations. Disparities investigated for clinical justification or bias.

**Mechanism 3: Pharmacist demographic diversity tracking.** The pharmacist user base demographics tracked (where consented). If the user base demographics don't match the broader pharmacist workforce, this is flagged for adoption-strategy review.

**Mechanism 4: External equity review.** Annual review by external clinical equity expert with aged-care specialisation.

### 7.3 Bias mitigation responses

When bias is detected:

- **Documentation:** the pattern is documented in detail
- **Investigation:** clinical informatics + ethics committee investigates root cause
- **Remediation:** if rule authoring is the cause, rules are revised; if evidence base is the cause, additional evidence is sought; if framing learning is the cause, learning patterns are revised
- **Monitoring:** the pattern is monitored after remediation to confirm resolution
- **Communication:** affected pharmacists, patients (via SDM), and employer pharmacies are informed

### 7.4 Equity in deployment

Equity considerations also apply to deployment patterns:
- Are some RACH operators (e.g., regional, smaller, more diverse-population-serving) being under-served?
- Are pricing structures effectively excluding equity-essential operators?
- Is the freemium tier accessible to pharmacists from underrepresented backgrounds?

These questions are reviewed quarterly by the commercial team in conjunction with ethics committee.

### 7.5 Indigenous health considerations

For Aboriginal and Torres Strait Islander residents in aged care:
- Cultural safety protocols embedded in care plans must be respected by recommendations
- Family/community decision-making structures (often broader than nuclear-family SDM model) supported
- Aboriginal Community Controlled Health Organisations engaged where applicable
- Specific consultation on platform design with Aboriginal health workers and community

This is a specific commitment beyond generic equity.

---

## Part 8 — Algorithmic management and worker protection

Extending the algorithmic management protections from the pharmacist self-visibility module to all platform contexts where workers are affected.

### 8.1 Scope of worker protection

The platform affects workers in multiple categories:
- **ACOP-credentialed pharmacists** (primary user; self-visibility commitments apply)
- **Pharmacy practice support staff** (intern pharmacists, technicians, dispensary staff)
- **RACH staff** (RNs, ENs, PCWs, care managers) — workflow data flows through the platform
- **GPs and prescribers** — their decision patterns are observed (per craft engine §8)
- **Pharmacy employer management** — their decisions about pharmacist deployment and contract retention

### 8.2 The four worker-protective commitments

**Commitment 1: No algorithmic determination as sole basis for adverse employment decision.** Per v3.0 Risk 12 and Bowdler 2026 OSH guidance. Operationalised through enterprise tier contractual clauses and contestation pathway.

**Commitment 2: Transparency about algorithmic observation.** Workers are informed when their patterns are observed. GPs are informed about the per-GP framing learning module and may opt out (per craft engine §8). Pharmacy support staff observed only for workflow function, not performance.

**Commitment 3: Worker access to their own data.** Per self-visibility module for pharmacists. Extended commitment for other worker categories: any worker whose patterns are observed has the right to view those observations, with appropriate visibility class controls.

**Commitment 4: No surveillance creep.** New observation patterns require ethics review before deployment. The platform's observation scope is bounded by current ethical architecture; expansion requires governance approval.

### 8.3 The "algorithmic management impact assessment"

Before any new feature that affects worker observation or evaluation:

- Document what's observed
- Document who can see the observation
- Document what decisions could be affected
- Document the contestation pathway
- Document the worker's right to opt out (where applicable)
- Document the OSH risk assessment

Required for any feature touching pharmacist KPIs, GP observation patterns, RACH staff workflow data, or pharmacy management dashboards. Ethics committee review required before implementation.

### 8.4 Burnout and psychological safety

Per the 2026 healthcare workplace research and the Wong 2024 well-being data: burnout in healthcare is rising. The platform's design must not contribute to burnout.

Specific commitments:
- Notification volume managed (no alert fatigue)
- Restraint signals respected (the platform doesn't push toward more action when restraint is appropriate)
- Reflective writing supported as restorative practice (per self-visibility guidelines)
- Performance pressure contextualised (trajectory framing, not peer ranking)
- Time-allocation surfacing supports work-life balance, not extension

Pilot evaluation includes specific qualitative measurement of platform impact on pharmacist burnout and psychological safety.

---

## Part 9 — Governance structure

Adapted from Liao et al. (2022, UWH) multi-disciplinary governance model. The structure is sized to Vaidshala's scale; it grows as the platform grows.

### 9.1 The Ethics Steering Committee

Composition (initial deployment):
- Clinical informatics lead (chair)
- Engineering lead
- Privacy/ethics lead
- Senior ACOP-credentialed pharmacist (external, advisory)
- Aged care clinical ethicist (external, advisory)
- Patient/family representative (external, advisory)
- Legal advisor (external, advisory)

Meeting cadence: monthly during pilot; quarterly thereafter.

Responsibilities:
- Review ethics-based audit findings
- Endorse new module ethical architecture sections
- Review bias detection findings and remediation
- Review contested algorithmic determinations escalated for committee
- Review external regulator inquiries and platform responses
- Commission annual external ethics audit

### 9.2 Project-specific sub-committees

For specific consequential decisions:
- **Bias remediation sub-committee:** convened when material bias detected
- **Restrictive practice review sub-committee:** for psychotropic and restraint recommendation patterns
- **Vulnerable population review sub-committee:** for end-of-life and severe-frailty recommendation patterns
- **Inter-organisational data sharing sub-committee:** for proposed regulator data sharing or cross-organisational aggregation

Sub-committees report to the Steering Committee.

### 9.3 The Pharmacist Advisory Group

Per pharmacist self-visibility guidelines: ongoing pharmacist advisory input is essential. The Pharmacist Advisory Group:
- 5–10 ACOP-credentialed pharmacists from diverse practice settings
- Quarterly meetings during pilot
- Reviews user experience, surveillance perception, contestation patterns
- Provides input on dashboard design, reflective prompts, framing language
- Independent of employer representation

### 9.4 Decision authority

The Steering Committee has decision authority for:
- Approval of new modules' ethical architecture sections (required gate)
- Approval of bias remediation plans
- Endorsement of external review findings
- Recommendations on policy changes affecting ethical architecture

For implementation-level decisions, individual leads have authority within their domain. For cross-cutting decisions, Steering Committee approval required.

### 9.5 Disclosure and conflict-of-interest

Committee members disclose:
- Financial interests in Vaidshala or competitors
- Clinical or professional relationships affecting independence
- Prior involvement in regulatory bodies relevant to the platform

Material conflicts trigger recusal from specific decisions.

### 9.6 Documentation and transparency

Steering Committee minutes are kept (with appropriate confidentiality for commercially sensitive items). Ethics-related decisions are documented and traceable. Annual ethics report published (with material findings) to internal team and selected external stakeholders.

---

## Part 10 — Continuous ethics-based auditing

Drawing from Mökander & Floridi: ethics-based auditing is most effective as a continuous process, not a one-time review. This section specifies the operational EBA practice.

### 10.1 EBA cadence

- **Daily:** automated pattern detection across substrate (acceptance-appropriateness divergence, suppression patterns, surveillance patterns, content variation across framings)
- **Weekly:** triage of pattern detection alerts by ethics-team-on-call; clear or escalate
- **Monthly:** structured review of patterns, contestations, escalations by the Ethics Steering Committee
- **Quarterly:** comprehensive review of the seven principles' operational health; rule tuning based on patterns
- **Annually:** external ethics audit by independent reviewer (clinical ethicist + technical ethicist); findings published to Steering Committee and pharmacy chains

### 10.2 EBA scope

Each level of EBA covers:

**Daily automated detection scope:**
- All algorithmic decisions
- All visibility class transitions
- All consent state changes
- All recommendation drafts
- All metric aggregations

**Weekly triage scope:**
- All flags from daily detection
- Pharmacist contestations raised
- External feedback received
- Anomalies reported by team members

**Monthly committee scope:**
- Patterns from weekly triage
- Bias detection findings
- New module proposals (if any)
- Regulatory or external inquiries

**Quarterly scope:**
- All seven principles' operational metrics
- Cross-cutting commitment health (e.g., visibility class enforcement integrity)
- Trends over time
- ERM rule refinement

**Annual scope:**
- Independent ethics audit
- External clinical ethicist review of recommendation patterns
- External technical ethicist review of bias and privacy
- Report to internal team and key partners

### 10.3 EBA outputs

Each EBA cycle produces:
- Findings (patterns identified)
- Concerns (potential ethical issues warranting attention)
- Actions (specific remediation)
- Owners (who will act)
- Timelines (when action will complete)
- Verification (how completion will be confirmed)

These are tracked in a structured EBA register.

### 10.4 Continuous improvement loop

Findings feed back into:
- Preventive layer architecture (architectural changes)
- Detective layer (new pattern detectors)
- ERM rules (new ethical concerns recognised)
- UI/UX changes (new framing or interaction patterns)
- Governance structure (new sub-committees or processes)

The loop is the mechanism by which the ethical architecture remains responsive to operational reality.

---

## Part 11 — Incident response

When ethical violations are detected, the response must be structured and prompt.

### 11.1 Incident classification

**Severity 1 — Clinical safety affected.** Recommendation pattern producing clinical harm or near-miss.
- Immediate hold of affected component
- Notification to pharmacy, RACH, affected residents/SDMs as applicable
- Clinical informatics review within 24 hours
- ACQSC and AHPRA reporting if required

**Severity 2 — Trust architecture violated.** Visibility class breach, surveillance pattern, surprise outreach to pharmacist's employer.
- Immediate hold of violating query path
- Pharmacist notification within 24 hours
- Remediation within 7 days
- Contestation pathway active for affected pharmacists

**Severity 3 — Bias or equity concern.** Material disparity in recommendation patterns or outcomes.
- Investigation within 14 days
- Remediation plan within 30 days
- Affected populations communicated with within 60 days

**Severity 4 — Procedural concern.** ERM rule misapplication, audit trail gap, governance process miss.
- Investigation within 30 days
- Process improvement within 60 days

### 11.2 Incident handling protocol

1. Detection: incident detected via monitoring, contestation, external feedback, or internal report
2. Triage: severity classified by ethics-team-on-call
3. Notification: affected parties informed per severity protocol
4. Response: immediate response per severity (hold, investigate, remediate)
5. Investigation: root-cause analysis with structured documentation
6. Remediation: technical, procedural, or design change
7. Verification: confirm remediation effective
8. Communication: transparent communication with affected parties
9. Learning: structured learning entry for governance review

### 11.3 No-blame culture for incident reporting

Internal team members reporting concerns face no adverse consequences. Anonymous reporting available. The platform's culture treats incident reports as essential for learning, not as failures attributable to individuals.

This is operationalised through:
- Anonymous reporting channel available to all team members
- Steering Committee attention to reporting volume (low volume may indicate concerns are not being reported)
- Annual review of reporting culture

### 11.4 External communication during incidents

For Severity 1 and Severity 2 incidents, transparent communication with affected parties is required. This includes:
- Acknowledgment of the incident
- Description of what occurred
- Description of immediate response
- Description of root cause (where determined)
- Description of remediation
- Timeline for resolution

External communication is reviewed by Steering Committee before release where stakes are high.

---

## Part 12 — External review mechanisms

The platform's ethical architecture must withstand external review. This section specifies how external review is structured.

### 12.1 Annual external ethics audit

Independent reviewer conducts annual audit covering:
- Operational health of the seven principles
- Bias detection and remediation effectiveness
- Pharmacist trust (via pharmacist advisory group survey)
- Resident outcomes (via aggregated substrate data with appropriate consent)
- Compliance with stated commitments

The reviewer is contractually independent: no financial interest in Vaidshala, no employer relationship with the platform team, professional standing in clinical AI ethics. Reviewer findings are published to the Steering Committee with no platform-team editorial control over content.

### 12.2 Clinical informatics consultation

For specific decisions (e.g., new rule classes, new framing patterns, restrictive practice handling), independent clinical informatics consultation is sought. This is distinct from the annual ethics audit:
- Project-specific
- Brief (5–20 hours typical)
- Output: specific recommendations on the proposed feature
- Authority: advisory; the team makes final decisions but the consultation is documented

### 12.3 Regulator engagement

The platform engages with regulators proactively, not reactively. Per v3.0 §13 commercial moves:
- ACQSC: evidence-quality recognition
- Inspector-General of Aged Care: systemic-issue support
- APC: RPL pathway alignment
- AHPRA: professional standards engagement

Each engagement involves transparent sharing of platform behaviour, willingness to receive feedback, and adaptation to regulatory guidance.

### 12.4 Academic partnership review

Through the PHARMA-Care framework consortium engagement (per v3.0 Move 2), academic researchers have access to anonymised platform data for evaluation. This is dual-purpose:
- Generates publication-grade evidence for regulator review
- Provides external scrutiny of platform behaviour by academic researchers with no commercial interest

### 12.5 Patient/family input mechanisms

Direct patient/family input is more difficult in aged care due to capacity issues. Mechanisms:
- SDM input via DON-mediated focus groups during pilot
- Family complaint channel monitored
- Independent patient/family representative on Ethics Steering Committee
- Aboriginal community engagement where Indigenous residents are affected

---

## Part 13 — Transparency and explainability

Transparency without explainability is theatre; explainability without transparency is opacity. Both are required.

### 13.1 What's transparent

- The seven principles (this document and v3.0 §9)
- The visibility class architecture (per self-visibility guidelines)
- The frame-vs-content separation (per craft engine guidelines)
- The override-reason taxonomy (per craft engine guidelines)
- The contestation pathway (per self-visibility guidelines)
- The governance structure (this document)
- Annual ethics audit findings (per Part 12)
- Incident reports for Severity 1 and 2 events (per Part 11)

These are documented, accessible to affected parties, and updated as the architecture evolves.

### 13.2 What's explainable

For any specific decision the platform makes affecting a person:
- Algorithmic suggestions to pharmacists: full reasoning trace via Layer 4 deep audit (per craft engine §2)
- Visibility class enforcement decisions: queryable
- Authorisation determinations: auditable end-to-end
- Override of pharmacist actions: never (the platform does not override; only suggests)
- Aggregation decisions affecting employer view: traceable to consent records

Explainability is at the level of "why did this specific thing happen for this specific person at this specific time," not just at the level of "how does the system work in general."

### 13.3 What's appropriately opaque

Some platform internals are not exposed:
- Specific implementation algorithms (where exposing them would enable gaming)
- Commercially sensitive configuration (e.g., specific pricing models)
- Personal information of individuals (per visibility class architecture)
- Audit trail details for incidents under active investigation (released after resolution)

The boundary between appropriate transparency and appropriate opacity is itself transparent: this Part 13 documents what's exposed and why what's not exposed is held back.

### 13.4 Plain-language summaries

For non-technical audiences (residents, families, RACH staff, GPs):
- Plain-language summary of platform behaviour
- Plain-language privacy notice
- Plain-language description of pharmacist autonomy preservation
- Plain-language description of how to contest an outcome

These summaries are reviewed by the patient/family representative on Ethics Steering Committee before publication.

---

## Part 14 — Cross-cutting ethical metadata

The ethical architecture lives in the data structures, not just in policies. This section specifies the cross-cutting metadata.

### 14.1 Required fields on every algorithmic decision

```go
type EthicalDecisionMetadata struct {
    DecisionID           uuid.UUID
    Component            string        // which service made the decision
    DecisionType         string        // e.g., "recommendation_draft", "visibility_aggregation"
    AffectedSubjectID    string        // resident / pharmacist / GP / etc.
    AffectedSubjectClass string        // "resident" / "pharmacist" / "gp" / etc.
    PrinciplesImplicated []string      // which of 7 principles apply
    ERMReviewed          bool
    ERMOutcome           *string       // approve / approve_with_monitoring / hold / reject
    ContestationEnabled  bool
    AuditTraceRef        uuid.UUID
    Timestamp            time.Time
}
```

This metadata is attached to every algorithmic decision in the platform. Queries against this metadata are how detection mechanisms work.

### 14.2 EthicsLog as parallel substrate

The platform maintains an EthicsLog alongside the EvidenceTrace:

```go
type EthicsLogEntry struct {
    ID                     uuid.UUID
    DecisionID             uuid.UUID  // references EthicalDecisionMetadata
    EntryType              string     // "decision" / "concern_flagged" / "review_requested" / "pattern_detected" / "incident"
    Severity               int        // 1-5
    Description            string
    Reviewer               *string
    ReviewOutcome          *string
    RemediationActions     []string
    Status                 string     // "open" / "investigating" / "remediated" / "verified" / "closed"
    CreatedAt              time.Time
    UpdatedAt              time.Time
}
```

The EthicsLog is queryable for:
- All decisions affecting a specific subject
- All concerns flagged in a time window
- Pattern aggregation across decisions
- Incident tracking and resolution

### 14.3 Storage

```sql
CREATE TABLE ethical_decision_metadata (
    decision_id UUID PRIMARY KEY,
    component VARCHAR(64) NOT NULL,
    decision_type VARCHAR(64) NOT NULL,
    affected_subject_id VARCHAR(64) NOT NULL,
    affected_subject_class VARCHAR(32) NOT NULL,
    principles_implicated TEXT[],
    erm_reviewed BOOLEAN NOT NULL,
    erm_outcome VARCHAR(32),
    contestation_enabled BOOLEAN NOT NULL,
    audit_trace_ref UUID NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    
    INDEX idx_subject (affected_subject_id),
    INDEX idx_component (component),
    INDEX idx_timestamp (timestamp),
    INDEX idx_principles USING GIN (principles_implicated)
);

CREATE TABLE ethics_log (
    id UUID PRIMARY KEY,
    decision_id UUID REFERENCES ethical_decision_metadata(decision_id),
    entry_type VARCHAR(32) NOT NULL,
    severity INTEGER NOT NULL,
    description TEXT NOT NULL,
    reviewer VARCHAR(64),
    review_outcome VARCHAR(64),
    remediation_actions TEXT[],
    status VARCHAR(16) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    INDEX idx_decision (decision_id),
    INDEX idx_severity (severity),
    INDEX idx_status (status),
    INDEX idx_created (created_at)
);
```

These tables underpin all ethics-related queries, dashboards, and audits.

---

## Part 15 — File and code organisation

```
shared/v2_substrate/
├── ethics/
│   ├── erm/                          # Ethical Reasoning Module
│   │   ├── module.go                 # Main ERM
│   │   ├── reasoners/                # Per-decision-type reasoners
│   │   │   ├── recommendation.go
│   │   │   ├── visibility.go
│   │   │   ├── authorisation.go
│   │   │   └── tests/
│   │   ├── escalation.go             # Escalation paths
│   │   └── tests/
│   ├── decision_metadata/            # Cross-cutting decision metadata
│   │   ├── recorder.go
│   │   ├── store.go
│   │   └── tests/
│   ├── ethics_log/                   # EthicsLog substrate
│   │   ├── logger.go
│   │   ├── querier.go
│   │   └── tests/
│   ├── pattern_detection/            # Detective layer
│   │   ├── acceptance_appropriateness.go
│   │   ├── suppression.go
│   │   ├── surveillance.go
│   │   ├── content_variation.go
│   │   ├── bias.go
│   │   └── tests/
│   ├── incident_response/            # Corrective layer
│   │   ├── classifier.go
│   │   ├── notifier.go
│   │   ├── remediation.go
│   │   └── tests/
│   ├── vulnerability/                # Resident vulnerability handling
│   │   ├── assessment.go
│   │   ├── adapter.go
│   │   └── tests/
│   ├── consent/                      # Consent state machine (cross-cutting)
│   │   ├── state_machine.go
│   │   ├── restrictive_practice.go
│   │   ├── dynamic_consent.go
│   │   ├── sdm_integration.go
│   │   └── tests/
│   └── governance/
│       ├── steering_committee.go     # Committee data structures
│       ├── eba_register.go           # EBA findings register
│       ├── audit_scheduling.go       # Annual audit scheduling
│       └── tests/

backend/services/ethics-monitoring/   # Standalone monitoring service
├── cmd/server/main.go
├── internal/
│   ├── api/
│   ├── monitors/                     # Pattern detection workers
│   │   ├── daily/
│   │   ├── weekly/
│   │   └── tests/
│   ├── alerting/
│   ├── reports/
│   └── tests/
└── README.md
```

The substrate-level ethics modules are consumed by every service. The standalone ethics-monitoring service runs the continuous detection layer.

---

## Part 16 — Testing approach

Five test categories specific to ethical architecture, in addition to the privacy boundary tests, contestation pathway tests, anti-surveillance tests, development-not-evaluation tests, and portability tests already specified in self-visibility guidelines.

### Category 1: Principle-level tests

Each of the seven principles has dedicated tests asserting its operational integrity:

- Frame-vs-content invariance (already specified in craft engine)
- Acceptance-appropriateness pairing (already specified)
- Restraint signal completeness
- Pharmacist autonomy preservation (override pathways available everywhere)
- Self-visibility temporal-order (already specified)
- Reviewability completeness (audit trails complete on every algorithmic decision)
- GP authority strengthening (no bypass patterns)

### Category 2: ERM tests

- ERM correctly identifies decisions requiring review
- ERM correctly classifies severity
- ERM escalation paths function
- ERM decisions auditable
- ERM rule changes follow governance

### Category 3: Vulnerability and consent tests

- Care intensity transitions correctly shift recommendation patterns
- SDM consent gating functions for restrictive practice
- Capacity assessment integration drives correct defaults
- Dynamic consent expiration triggers re-evaluation
- End-of-life pattern recognition correctly suppresses inappropriate recommendations

### Category 4: Bias detection tests

- Demographic stratification of metrics computes correctly
- Disparity flagging triggers at appropriate thresholds
- Rule outcome equity audit produces complete coverage
- Bias remediation tracking functions

### Category 5: Incident response tests

- Severity classification correct
- Notification protocols trigger appropriately
- Hold mechanisms function (affected components actually held)
- Remediation tracking complete

### Annual external testing

In addition to internal CI/CD test coverage, the annual external ethics audit includes hands-on testing of:
- Pharmacist contestation pathway (auditor submits test contestation)
- Visibility class boundaries (auditor attempts boundary violations)
- Override pathways (auditor confirms availability)
- Audit trail completeness (auditor traces specific decisions end-to-end)

---

## Part 17 — Implementation sequencing

Aligned with Phase 1 of the implementation plan (Weeks 5–10) plus extensions for governance and ethics-based auditing operations.

### Week 5–6: Foundation

- Cross-cutting decision metadata structure
- EthicsLog substrate
- ERM scaffold (decision points identified, reasoner stubs)
- Vulnerability assessment integration into Resident entity

### Week 7–8: ERM and consent

- ERM reasoners for recommendation, visibility, authorisation decisions
- Consent state machine (extends Phase 0.2)
- Restrictive practice consent gating
- Dynamic consent for passive data
- Capacity assessment integration

### Week 9: Detection layer

- Pattern detectors for acceptance-appropriateness, suppression, surveillance
- Bias detection (demographic stratification of metrics)
- Daily automated detection job

### Week 10: Governance operationalisation

- EBA register
- Steering Committee data structures
- Annual audit scheduling
- Incident response protocols

### Week 11: Buffer for external review

- External clinical ethicist review of architecture
- External technical ethicist review of bias and privacy
- Address findings before pilot deployment
- Document findings and responses

### Estimated team

- 1 backend engineer specifically for ethics substrate (full-time, 7 weeks)
- 1 backend engineer for cross-service integration of ethics metadata (full-time, 5 weeks; parallel)
- Privacy/ethics lead (part-time, 7 weeks)
- Clinical informatics lead (part-time, 7 weeks for principle operationalisation)
- External clinical ethicist (3-day intensive review at week 11)
- External technical ethicist (3-day intensive review at week 11)

---

## Part 18 — Risks and mitigations

**Risk 1: Aspirational drift.** The architecture is documented but not implemented operationally. Engineers cite the document selectively. Mitigation: every module's design specification has a required "Ethical commitments" section reviewed by Steering Committee. Code review checklist requires explicit ethics metadata on algorithmic decisions. CI tests block releases with missing metadata.

**Risk 2: Checkbox compliance.** The architecture is implemented narrowly without intent. Mitigation: continuous ethics-based auditing detects spirit-vs-form gaps. Pharmacist Advisory Group provides ground-truth on user experience. Annual external audit catches drift.

**Risk 3: Governance overload.** The Steering Committee becomes a bottleneck for ordinary decisions. Mitigation: clear delegation of implementation-level decisions to individual leads; Committee reserved for cross-cutting and consequential decisions. Sub-committees handle project-specific work.

**Risk 4: Detection false positives overwhelm response capacity.** Pattern detection produces too many flags for available review. Mitigation: severity classification triages; weekly triage absorbs minor patterns; Steering Committee handles material patterns only. Pattern thresholds tuned based on volume.

**Risk 5: External reviewer access creates security risk.** Auditors with platform access could exfiltrate sensitive data. Mitigation: contractual confidentiality; limited access scope; no PHI without specific data-sharing agreement; reviewer security background checks.

**Risk 6: Incidents handled slower than commitments require.** Severity 1 24-hour response isn't met during off-hours. Mitigation: ethics-team-on-call rotation; automated paging; documented escalation tree.

**Risk 7: Community concerns about platform legitimacy.** Aboriginal community, family advocacy groups, or aged care advocacy organisations raise concerns about platform governance. Mitigation: proactive engagement during pilot; representation on Ethics Steering Committee where applicable; willingness to adapt based on feedback.

**Risk 8: Regulatory inquiry exceeds expected scope.** ACQSC, AHPRA, or Inspector-General investigation goes deeper than anticipated. Mitigation: audit trail completeness ensures defensibility; legal advisor engaged early; transparency commitment maintained throughout.

**Risk 9: Bias detection finds patterns the team cannot remediate.** Detected bias is structural to the evidence base or substrate, not amenable to internal action. Mitigation: bias documented openly; communicated to affected parties; advocacy for evidence base improvement; mitigations applied where possible.

**Risk 10: Worker protection commitments conflict with employer expectations.** Pharmacy chain expectations of platform's surveillance capability exceed the architecture's commitments. Mitigation: clear contractual specifications; Steering Committee endorsement of any expansion; pharmacist advocacy in commercial conversations.

**Risk 11: ERM rule misalignment with clinical reality.** ERM flags too many or too few decisions. Mitigation: quarterly ERM review; clinical informatics oversight; pharmacist feedback on ERM quality; iterative refinement.

**Risk 12: Substituted decision-maker conflicts.** Multiple SDMs disagree on consent for restrictive practice. Mitigation: platform documents disagreement; does not algorithmically resolve; surfaces for human resolution; respects governance pathways (formal SDM determinations, tribunal processes where applicable).

---

## Part 19 — Closing

Three observations as we close v1.0 of the ethical architecture guidelines.

**One:** The ethical architecture is not separate from the platform; it is the platform's foundation. Every module inherits from this architecture. The seven principles are not aspirational; they are architectural constraints with implementation specifications, detection mechanisms, and escalation paths. The team should treat any new module's design as incomplete until its "Ethical commitments" section has been written and reviewed.

**Two:** The architecture is responsive to the literature's recent shift from high-level principles to operationalisation. Bisson et al. (2026) on the six interventions for medical AI agents, Liao et al. (2022) on multi-disciplinary governance, Mökander & Floridi on ethics-based auditing, Abbasian (2026) on AI ethics for aging populations — these frameworks have informed every part of this document. The team should continue tracking the literature; this document will evolve.

**Three:** The single most important commitment in the architecture is the continuous ethics-based auditing operations (Part 10). One-time review at deployment is insufficient because the world changes around the platform. Continuous EBA is what keeps the architecture honest as the substrate grows, the pharmacist user base diversifies, the regulatory environment shifts, and the clinical evidence base updates. The Steering Committee's monthly cadence, the quarterly comprehensive review, and the annual external audit are the discipline that prevents drift.

What this document does not yet specify, and what should be subsequent work:

- **The Ethics Steering Committee charter** with detailed terms of reference, member responsibilities, decision-making protocols, and conflict resolution procedures
- **The annual external ethics audit specification** detailing scope, methodology, reviewer qualifications, and reporting requirements
- **The incident response runbook** with detailed protocols for each severity tier, including specific external communication templates
- **The bias detection specification** detailing metric stratification, threshold setting, and remediation playbooks
- **The Pharmacist Advisory Group operating model** including selection, compensation, and engagement patterns

Each of these merits its own implementation guideline of comparable rigour. They are the operational substrate for what this document specifies architecturally.

The ethical architecture is the platform's most distinctive commitment. Every other module inherits from it. The trust pharmacists, RACHs, regulators, and patients place in the platform depends on this architecture holding operationally — not just being documented well. The continuous ethics-based auditing is what proves it holds. The next eight months of pilot will test whether the commitments are real.

— Claude
