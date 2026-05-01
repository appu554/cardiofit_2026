# Gap 21 — Closed-Loop Outcome Learning

**Implementation Guidelines, V5 Sprint 2**

Status: Planning. Inputs ready (Gap 19 T0→T4 lifecycle shipped; Gap 20 Predictive Risk Layer plan delivered, coding pending). Two pilots in flight: HCF CHF (readmission reduction) and Aged Care (admission reduction).

---

## 1. Executive Summary

Gap 19 gave the platform **eyes** — a T0→T4 lifecycle that records when a signal is detected, when it becomes an alert, when a clinician acknowledges it, when an intervention is initiated, and when an outcome is observed or the window closes. Gap 20 gives the platform **foresight** — predictive risk scores that anticipate acute-on-chronic decompensation, admission, and readmission.

Gap 21 is the **learning organ** that turns these into a system that improves over time without drifting, gaming itself, or becoming a victim of its own success.

Concretely, Gap 21 delivers six capabilities:

1. **Causal Outcome Attribution** — the engine Gap 19 explicitly deferred. Not "did the outcome occur" but "was the alert causally linked to a reduction in outcome risk," estimated with target trial emulation, propensity weighting, and doubly-robust estimators against a no-treatment potential outcome.
2. **Feedback-Aware Performance Monitoring** — continuous calibration and discrimination tracking that accounts for the label-selection and treatment-response feedback loops that break naive surveillance. This uses Adherence-Weighted and Sampling-Weighted monitoring so that *successful* alerts don't look like model failure.
3. **Subgroup and Fairness Drift Detection** — first-class monitoring across the subgroups that matter for the two pilots (age bands, comorbidity strata, Indigenous status, postcode-based SES proxies, language, private-vs-public cohort). This is not a compliance add-on; it is a primary safety surface.
4. **Structured Clinician Feedback Capture** — override reasons, true/false-positive marks, counterfactual corrections (*"I would have flagged this case earlier because X"*), and free-text review mined with NLP for pattern discovery. These feed into labelling, active learning, and next-generation retraining.
5. **Retraining with Guardrails** — feedback-aware training sets, shadow deployment, A/B canary, pre-specified acceptance criteria, and automatic rollback. Modelled on FDA's final Dec 2024 PCCP guidance even though the current scope is CDS/population-health rather than SaMD — this future-proofs the Aged Care and HCF contracts and leaves a regulatory door open.
6. **Governance Ledger** — every model version, every training set, every threshold change, every recalibration is a signed entry in an append-only ledger, with impact assessment and rollback playbook attached. Generates the artifacts clinicians and procurement teams will ask for by Q3.

### Scope and non-scope

**In scope.** Readmission outcomes (HCF CHF), admission avoidance outcomes (Aged Care), mortality, transition-to-higher-care, any outcome Gap 19 instruments at T4. Attribution and learning for any pattern produced by Gap 20's Predictive Risk Layer, any legacy V4 pattern, and the compound pattern library.

**Out of scope for Sprint 2.** Reinforcement-learning–based policy optimisation (the AAAI 2026 RLCF work with digital twins); full federated training across HCF and Aged Care cohorts; any generative LLM-based clinical reasoning. These are Gap 22/23 candidates, noted where the architecture supports them later.

### The single most important design stance

**Gap 21 treats the platform as a causal intervention, not a classifier.** Once an alert changes clinician behaviour, the observed outcome is a function of the alert itself. Every design choice in this document — attribution, monitoring, retraining — flows from that stance. The alternative (standard supervised-learning monitoring and retraining) creates a documented failure mode where successful alerts look like model degradation, the model gets "recalibrated" to match the post-intervention world, high-risk patients get relabelled low-risk because they were the ones who got interventions, and the model slowly poisons itself. This failure mode is the single most common reason deployed clinical ML models silently decay, and Gap 21 is designed to prevent it.

---

## 2. Why Gap 21 Matters

Three reasons, in increasing order of urgency.

**Contractual.** Both pilots are outcome-linked. HCF's CHF program is measured on 30-day and 90-day readmission reduction; Aged Care's readiness assessment flagged admission avoidance as the primary metric. Without a defensible attribution engine, the platform cannot claim credit for reductions it caused, cannot distinguish genuine impact from population drift, and cannot answer the inevitable CFO question *"how do we know it's not just regression to the mean?"* Gap 21 is the answer to that question.

**Safety.** Every deployed ML alert system in a clinical environment experiences feedback loops. The literature is now unambiguous: the Stanford group's 2025 paper on monitoring strategies shows that standard unweighted monitoring of a successful classifier drives AUROC estimates from 0.72 down to 0.52 within one retraining cycle, because the outcome labels of high-risk patients are being selectively modified by the treatments the alerts trigger. The models that "work best" decay fastest. Without Gap 21, Gap 20 degrades silently.

**Regulatory optionality.** The platform is currently CDS/population-health and not regulated as a medical device in either market. That is fragile. The FDA's final Dec 2024 PCCP guidance, the EU AI Act's high-risk provisions (relevant for Aged Care in market expansion), India's CDSCO software-as-device draft, and Australia's TGA SaMD refresh all point to a near-future where adaptive clinical algorithms require pre-authorised change control plans. Building the governance now is a fraction of the cost of retrofitting later, and it is the kind of infrastructure that materially differentiates the platform in procurement.

---

## 3. Design Principles

Nine principles that drove the plan below. These are load-bearing; every subsequent file and phase traces back to one of them.

1. **Causal over correlational.** Outcomes are attributed with target trial emulation (TTE) semantics: specify the eligibility window, treatment strategies, time zero, outcome, and causal contrast. Propensity-weighted and doubly-robust estimators are the default. Naive conditional-rate comparisons are prohibited in production attribution outputs.
2. **Feedback loops are expected, not anomalous.** Every monitoring metric has a feedback-aware variant (Adherence-Weighted, Sampling-Weighted, or counterfactual). Dashboards surface both the naive and feedback-aware estimates, with a prominent note when they diverge.
3. **No silent retrain.** Every model update is a named event in the governance ledger, with pre-specified acceptance criteria, shadow results, canary results, and a rollback plan. "Continuous retraining" as an operational mode is explicitly forbidden; all retraining is versioned and gated.
4. **Subgroup performance is a first-class metric.** A model whose aggregate AUROC is 0.85 but whose Indigenous-cohort AUROC is 0.65 fails acceptance. Subgroup monitoring uses the same stringent calibration drift detection (ADWIN over dynamic calibration curves, per subgroup) as aggregate monitoring.
5. **Attribution outputs are ranges, not points.** Every attribution estimate carries an uncertainty interval (bootstrap or doubly-robust influence-function) and a sensitivity analysis against plausible unmeasured confounding. Point estimates without bounds are not returned.
6. **Clinician feedback is labelled data, treated with the same rigour as ground truth.** Override reasons, true/false-positive marks, and counterfactual corrections pass through validation gates identical to T4 outcome ingestion. Noisy clinician input is quantified and weighted, not silently trusted.
7. **Per-patient baselines carry through.** The KB-23 folded-service pattern that drives Gaps 14–19's per-patient baselines also drives attribution: the counterfactual outcome is anchored to the patient's own baseline trajectory, not a cohort mean.
8. **Market-aware, override-driven.** The YAML market-config pattern extends to learning: each market (AU-HCF, AU-AgedCare, IN-pilot) has its own attribution horizons, outcome definitions, fairness cohorts, and retraining cadences, with a base config inherited and overridden.
9. **Rollback is cheap and frequent.** Every deployed change must be rollbackable within 15 minutes using a single operator command. Rollback is rehearsed monthly, not just tested.

---

## 4. State-of-the-Art Grounding

Brief synthesis of the research base this plan draws on. Full reference list at the end. This is the intellectual inheritance; the sections after this are how we operationalise it.

### 4.1 Target Trial Emulation (Hernán & Robins, plus 2025 operational frameworks)

TTE is now the dominant framework for causal inference from observational clinical data, endorsed by ISPOR, NICE's Real-World Evidence Framework, FDA's RWE Program, and the 2025 *npj Digital Medicine* operational framework paper. The method: explicitly specify the hypothetical randomised trial (eligibility, treatment strategies, time zero, assignment procedure, outcomes, causal contrast), then emulate each component with observational data. This prevents the self-inflicted biases (immortal time, selection, prevalent-user bias) that plague naive observational analyses. Gap 21's attribution engine is structured as a TTE protocol that runs continuously.

### 4.2 Feedback-Aware Monitoring (Kim, Corbin, Grolleau et al., Stanford, 2025 JBI)

The single most important recent paper for this gap. They show that standard classifier monitoring collapses when interventions modify the outcome labels (the "label modification" feedback loop), and propose two alternatives:
- **Adherence-Weighted Monitoring**: reweight the evaluation set by the inverse probability that the intervention was actually taken given the alert.
- **Sampling-Weighted Monitoring**: reweight by the probability of observing an un-intervened outcome given observed features.

Both target the **no-treatment potential outcome** — the outcome the patient would have had *without* the intervention — which is what the classifier's performance should be measured against. Without this, a successful model is punished for its success, and retraining turns into slow poisoning.

### 4.3 Calibration Drift Detection (Davis, Greevy, Lasko, Walsh, Matheny, Vanderbilt)

The canonical method. Dynamic calibration curves updated as observations stream in, monitored for drift by an adaptive-windowing (ADWIN) one-sided test. Davis et al. 2020 remains the reference implementation; Haq 2024 and Zhang et al. 2025 extend the idea to CUSUM-style control charts which give tighter drift-detection latency at modest false-alarm cost. Gap 21 uses ADWIN as the production detector and calibration-CUSUM as a second-tier confirmatory detector.

### 4.4 Feedback Loop Taxonomy (Pagan et al., ACM EAAMO 2023; van Amsterdam et al., *Lancet Digital Health* 2025)

Clinical prognostic models exhibit several distinguishable feedback loops. Van Amsterdam's 2025 paper distinguishes positive feedback (self-fulfilling prophecy — predicting high mortality triggers comfort care, which produces high mortality, which reinforces the prediction) from negative feedback (self-defeating prophecy — the sepsis alert prevents the sepsis, so the alert looks like a false positive). Gap 21 has explicit handling for both: attribution distinguishes *outcome prevented* from *outcome correctly predicted but not preventable*, and monitoring distinguishes *calibration drift* from *treatment-effect–induced label modification*.

### 4.5 PCCP Framework (FDA final guidance Dec 2024; FDA/HC/MHRA joint principles Aug 2025)

Three required components:
1. **Description of Modifications** — the specific, planned changes (retraining, recalibration, feature additions, threshold changes, scope expansions).
2. **Modification Protocol** — methodology, verification/validation activities, pre-defined acceptance criteria, implementation steps.
3. **Impact Assessment** — how the modification affects safety and effectiveness, for whom, with what magnitude.

Plus five guiding principles (Aug 2025 joint statement): **focused, risk-based, evidence-based, transparent, and consistent with the device's intended use**. Gap 21's Governance Ledger is structured as a rolling PCCP — every change lands as a triplet (modifications, protocol, impact assessment) with lineage to the monitoring signals that triggered it.

### 4.6 Active Learning and Uncertainty-Aware Labelling (Sener & Savarese k-center; Lakshminarayanan ensembles; Jayaraman et al. 2024)

For any non-trivial clinical cohort, clinician labelling budget is the scarcest resource. The efficient strategy is **k-center selection on high-uncertainty cases**: pick the case with highest model uncertainty whose features are most dissimilar from already-labelled cases, iterate. Uncertainty quantified by ensemble coefficient of variation (Lakshminarayanan-style deep ensembles with tanh-squashed output) is simple and reliable. Gap 21 uses ensemble CoV as the scoring function and k-center for diversity; the labelling queue is sized to clinician capacity.

### 4.7 Federated Methods (Teo et al. *Cell Reports Medicine* 2024; FL-TTE, *npj Digital Medicine* 2025)

Federated TTE — federated propensity score estimation plus federated Cox regression — has been shown to outperform naive meta-analysis on sepsis and Alzheimer's cohorts across dozens of hospitals. Not in Sprint 2 scope but the architecture should not preclude it: attribution outputs should be aggregatable across sites without patient-level data leaving site boundaries.

### 4.8 Alert Fatigue and Clinician Override Mining

90–96% of generic CDS alerts are overridden. The determinants are well-characterised: irrelevance to current context, known-patient-tolerance, duplicate firings, and timing within workflow. The Premier CDS platform and published work (Poly et al. 2020) shows that NLP on free-text override comments surfaces patterns that can retire or refactor alerts. Gap 21 ingests override reasons as first-class feedback.

---

## 5. Architecture Overview

### 5.1 Closed-loop at a glance

```
                          ┌─────────────────────┐
                          │  Gap 20 Predictions │
                          │  (Predictive Risk)  │
                          └──────────┬──────────┘
                                     │ risk_score, features, model_version
                                     ▼
┌────────────────┐         ┌─────────────────────┐         ┌────────────────┐
│  Gap 19 Events │────────▶│   Alert Firing      │────────▶│ Clinician View │
│  T0 detect     │         │   (thresholded)     │         │ (Worklist)     │
└────────┬───────┘         └──────────┬──────────┘         └───────┬────────┘
         │                            │ alert_id                   │ ack/override/action
         │                            ▼                            ▼
         │                 ┌─────────────────────┐     ┌───────────────────┐
         │                 │ T1 alert_raised     │     │ T2 acknowledged   │
         │                 └─────────────────────┘     │ T3 intervention   │
         │                                             └─────────┬─────────┘
         │                                                       │
         ▼                                                       ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                    Event Ledger (Gap 19 extended)                        │
│      T0 → T1 → T2 → T3 → T4 with full feature/prediction snapshot       │
└─────────────────────────────────────────────────────────────────────────┘
         │                               │                                │
         │ structured                    │ T4 outcome                     │ clinician
         │ clinician feedback            │ ingestion                      │ override reasons
         ▼                               ▼                                ▼
┌────────────────────────────────────────────────────────────────────────────┐
│                       GAP 21 CORE ENGINES                                   │
├────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  1. Attribution      2. Monitoring     3. Feedback      4. Active Learning │
│     Engine              Engine             Ingestion        Queue           │
│     (TTE, IPW, DR)      (ADWIN,            (NLP on          (ensemble CoV, │
│                          calib-CUSUM,       override         k-center)      │
│                          subgroup,          reasons;                        │
│                          adherence-         structured                      │
│                          weighted)          feedback)                       │
│                                                                             │
│  5. Retraining        6. Governance                                         │
│     Pipeline              Ledger                                            │
│     (shadow →             (PCCP-style                                       │
│      canary →              change control,                                  │
│      full;                 impact                                           │
│      guardrailed)          assessment)                                      │
│                                                                             │
└────────────────────────────────────────────────────────────────────────────┘
         │                     │                     │
         │ attribution         │ retraining          │ governance
         │ reports             │ triggers            │ artifacts
         ▼                     ▼                     ▼
  Clinician/CFO          Next-gen models        Regulatory /
  dashboards             → back to Gap 20       procurement
```

### 5.2 Data flow, in plain English

A patient's signal is detected (T0) and the Gap 20 Predictive Risk Layer scores them. If the score crosses a market-configured threshold, an alert fires (T1). A clinician sees it on the worklist (Gap 18), acknowledges it (T2), and either initiates an intervention or overrides with a structured reason (T3). At T4, the outcome window closes — either the monitored outcome occurred, or it didn't.

Gap 21 takes this event stream and does six things in parallel:

1. **Attribution engine** estimates *what the patient's outcome would have been without the alert* (the counterfactual no-treatment potential outcome) and computes the causal effect attributable to the alert.
2. **Monitoring engine** updates feedback-aware calibration and discrimination metrics for the model, including per-subgroup breakdowns.
3. **Feedback ingestion** structures the clinician's override reason (or intervention choice) and routes it to the learning queue.
4. **Active learning queue** picks the cases whose labels are highest-value for the next retrain (high uncertainty, diverse).
5. **Retraining pipeline** runs on schedule (or when monitoring fires a drift alarm), produces a candidate model, runs it in shadow, then canary, then full, each gated by pre-specified acceptance criteria.
6. **Governance ledger** records everything.

The outputs of Gap 21 are consumed by: the clinician dashboard (attribution = "your intervention helped"), the CFO dashboard (outcome reduction attributable to program), the engineering team (drift alarms), the next-generation Gap 20 models (new training sets), and the procurement/regulatory files (PCCP-style change control artifacts).

### 5.3 Where Gap 21 sits in the service topology

Gap 21 is three new services and extensions to three existing ones, following the KB-23 folded service pattern established in Gaps 14–19.

**New services**
- `services/attribution/` — the causal attribution engine and counterfactual estimator.
- `services/learning/` — the retraining pipeline, active learning queue, feedback ingestion.
- `services/governance/` — the PCCP-style change control ledger and artifact generator.

**Existing services extended**
- `services/monitoring/` (exists from Gap 18/19 worklist & time-to-response) — add feedback-aware metric computation, subgroup dashboards, drift detection.
- `services/events/` (Gap 19 event ledger) — add outcome ingestion, ground-truth resolution, event-to-attribution bridging.
- `services/prediction/` (Gap 20 when shipped) — add model-version metadata, prediction snapshotting for post-hoc attribution.

Everything below is elaborated against these six surfaces.

---

## 6. The Six Core Engines

Each engine is a folded service with a single public contract, a thin YAML market config, and a testable core. This section is the specification; Section 7 translates it to phases and steps.

### 6.1 Attribution Engine (`services/attribution/`)

**Public contract.** Given a closed T0→T4 lifecycle (alert raised, clinician response, outcome observed), return a structured attribution verdict: the estimated counterfactual outcome probability, the observed outcome, the attributable risk difference with confidence interval, the sensitivity bound against unmeasured confounding (E-value), and a clinician_label (e.g., "prevented readmission", "no effect detected", "outcome despite intervention").

**Internal structure.**
- A **Target Trial Emulator** that takes the cohort and period of interest and runs the TTE protocol: eligibility specification, time zero identification, treatment-strategy assignment from observed clinician actions, outcome ascertainment, and causal contrast computation.
- A **Propensity Score Estimator** (gradient-boosted trees, calibrated with isotonic regression) that estimates `P(intervention | pre-alert features)` per patient. Overlap/positivity diagnostics run automatically; if overlap fails, attribution is returned as "inconclusive" rather than a forced estimate.
- A **Doubly-Robust Estimator** combining IPW and outcome regression (AIPW / targeted maximum likelihood estimation — TMLE preferred where sample permits). Produces a consistent estimate if either the propensity model or the outcome model is correct, which buys resilience against any single-model misspecification.
- A **Counterfactual Outcome Head** — a model that predicts the no-treatment potential outcome trained on historical untreated analogues. This is the engine Gap 19 deferred. It is trained on patients who received alerts but no intervention (override cohort), with per-patient-baseline anchoring to control for severity confounding.
- A **Sensitivity Analysis Module** computing the E-value (the minimum strength of unmeasured confounding needed to explain away the observed effect) and a tipping-point analysis. Every attribution output carries its E-value; below a configured threshold (e.g., E-value < 1.5 for readmission outcomes), the attribution is labelled "fragile."

**What it does NOT do.** It does not compute population-level ATE in one shot and call it a day. Attribution runs at the individual level, aggregated per pattern, per cohort, per subgroup, per horizon. The same infrastructure produces both the per-patient attribution for the clinician dashboard ("your intervention reduced this patient's 30-day readmission probability from 42% to 18%") and the portfolio-level attribution for the CFO ("the CHF program prevented an estimated 47 ± 12 readmissions this quarter, E-value 1.8").

### 6.2 Monitoring Engine (`services/monitoring/` — extended)

**Public contract.** For each model×cohort×subgroup×horizon combination, produce the current calibration curve, discrimination metrics, and drift status (stable, warning, alarm). Feedback-aware variants are computed in parallel with naive variants; both are returned so discrepancies are visible.

**Three detector tiers, deliberately layered.**

- **Tier 1 — ADWIN over dynamic calibration curves (Davis et al. 2020).** Production detector. One-sided test for increasing miscalibration. Runs continuously. Default alpha = 0.001 to keep false-alarm rate low under the expected monitoring volume.
- **Tier 2 — Calibration CUSUM (Zhang et al. 2025).** Confirmatory detector. Tighter drift-detection latency. Runs on ADWIN warnings; if both agree, escalate to alarm.
- **Tier 3 — Subgroup drift.** Tier 1 + Tier 2 run per subgroup independently. A subgroup-only alarm (aggregate stable, subgroup drifting) is treated as equal priority to an aggregate alarm.

**Feedback-aware variants.** Every drift detector runs twice: once on standard labels (may be biased by intervention effects) and once on Adherence-Weighted or Sampling-Weighted labels (unbiased estimate of no-treatment potential outcome performance). Divergence between the two is itself an alarm signal — it indicates the model is working (successfully modifying outcomes) rather than failing.

**Subgroup definitions.** Market-configured via YAML. For HCF CHF pilot: age bands (<65, 65–74, 75–84, 85+), sex, NYHA class at enrolment, diabetes comorbidity, chronic kidney disease comorbidity, hospital vs community setting, metro vs regional postcode, Aboriginal and Torres Strait Islander status where consented. For Aged Care pilot: RAC-level-at-admission, dementia diagnosis presence, polypharmacy band, frailty score band, state/territory. For India cohort: language, urban/rural, insurance type.

**What it does NOT do.** It does not silently retrain. An alarm triggers a retraining *candidate event* in the governance ledger which must be reviewed and authorised before the retraining pipeline runs.

### 6.3 Feedback Ingestion (`services/learning/feedback/`)

**Public contract.** Accept structured clinician feedback (override reasons from T2/T3, true/false-positive marks, counterfactual corrections, free-text commentary) and produce validated, confidence-weighted labels and covariates that enter the training and evaluation streams.

**Structured inputs.**
- **Override reason taxonomy.** Clinician_label enum driven by Gap 18 worklist: `not_clinically_relevant`, `already_addressed`, `patient_declined`, `alert_too_late`, `alert_too_early`, `features_inaccurate`, `disagree_with_risk`, `defer_to_specialist`, `other_with_free_text`. Each has a technical_label carrying structured implications for the learning pipeline.
- **True/false-positive marks.** Binary with optional rationale. Reviewed against the eventual T4 outcome for adjudication.
- **Counterfactual corrections.** Structured: "I would have wanted this alert at signal X" or "I would not have wanted this alert because Y." These become proxy labels for timing/threshold learning.
- **Free-text commentary.** Passed through a domain-tuned NLP pipeline (pattern-matching first, small LM second, with validation gate) that categorises into the override-reason taxonomy and surfaces new themes.

**Confidence weighting.** Clinician feedback is not infallible. Each feedback item is weighted by:
- Reviewer experience (resident vs senior clinician vs specialist).
- Temporal proximity (feedback given within 15 minutes of alert vs retrospectively days later).
- Consistency across a reviewer's history (a reviewer whose false-positive marks disagree with eventual T4 outcomes at 40% is weighted lower than one at 10%).
- Inter-rater agreement on a rotating 5% sample of dually-reviewed cases.

**Validation gates.** Feedback fails validation if: the reviewer didn't see the alert (timestamp mismatch with Gap 18 worklist view); the T4 outcome arrived before the feedback and contradicts it in a way that suggests the feedback is post-hoc rationalisation; the free-text is detectably copy-paste or adversarial.

### 6.4 Active Learning Queue (`services/learning/active/`)

**Public contract.** Maintain a prioritised queue of cases for clinician labelling, sized to clinician capacity, that maximises information gain per label.

**Scoring.**
- **Uncertainty component** — ensemble coefficient of variation on the Gap 20 risk score. The model ensemble is a natural byproduct of Gap 20's cross-validation folds; running inference through the ensemble costs marginal compute and produces a free uncertainty signal.
- **Diversity component** — k-center selection: given the already-labelled pool, pick the case furthest from any labelled case in feature space. Prevents labelling budget being spent on minor variations of the same case.
- **Rarity component** — minor-subgroup cases are up-weighted. If the Indigenous cohort has 3% of data but is a fairness monitoring target, rarity weighting ensures it gets proportional labelling attention.
- **Cost component** — labelling cost is not uniform. Cases requiring chart review are weighted differently from cases needing only alert acknowledgement feedback.

**Queue management.** Sized per clinician per day based on observed labelling throughput. Over-queueing produces rejection, which itself is a signal (if a reviewer consistently skips certain case types, those case types are down-weighted for that reviewer). Under-queueing produces idle capacity; the queue refills on demand.

### 6.5 Retraining Pipeline (`services/learning/retrain/`)

**Public contract.** On a governance-authorised retraining event, produce a candidate model version with feedback-aware training, shadow-evaluate it, canary-deploy it, and promote or rollback based on pre-specified acceptance criteria.

**Training set construction.**
- **Feedback-aware weights.** Adherence-weighted sample weights by default. Where adherence can't be estimated (e.g., early in a new pilot), fall back to Sampling-weighted or, as a last resort, a pre-intervention-only subset.
- **Temporal stratification.** Holdouts are temporally blocked, not randomly split. The most recent 10% of data is the acceptance-criterion holdout; it is never touched until final evaluation.
- **Subgroup stratification.** Training set preserves or up-samples fairness-target subgroups to prevent aggregate optimisation at the expense of minority cohorts.
- **Label sources.** T4 outcomes (primary); clinician true/false-positive marks (secondary, weighted); counterfactual corrections (tertiary, used only for timing/threshold sub-models, not for primary risk model).

**Evaluation stages.**
- **Stage 0 — Offline holdout.** Standard metrics (AUROC, AUPRC, calibration-in-the-large, calibration slope, Brier score) plus feedback-aware variants, plus subgroup breakdowns. All must meet pre-specified acceptance criteria.
- **Stage 1 — Shadow mode.** Candidate model runs in parallel with production, making predictions that are logged but not surfaced to clinicians. Minimum 2 weeks or 500 alerts' worth of events, whichever is later. Prediction-level agreement with production is tracked; large disagreements are flagged for clinician review.
- **Stage 2 — Canary.** Candidate is promoted for a random 10% of new alerts, production serves the other 90%. Runs for a pre-specified window (default 4 weeks for HCF CHF, 8 weeks for Aged Care due to longer outcome horizons). A/B comparison on attribution-weighted outcome reduction is the primary acceptance metric.
- **Stage 3 — Full promotion.** Candidate becomes production. Previous production moves to shadow for 4 more weeks as a rollback hot standby.
- **Rollback.** At any stage, if a monitoring alarm fires or a safety threshold is breached, single-command rollback to the immediately previous version. Rollback is rehearsed monthly.

**Acceptance criteria (pre-specified before candidate is built).** Example for HCF CHF:
- Aggregate AUROC ≥ production AUROC − 0.01 (non-inferiority).
- Calibration-in-the-large |observed − predicted| < 0.02.
- Subgroup AUROC ≥ 0.7 in every fairness-target subgroup.
- Attribution-weighted 30-day readmission reduction point estimate not worse than production.
- E-value of attribution estimate ≥ 1.5.

Criteria are stored in the governance ledger alongside the candidate version. Meeting them authorises promotion; missing any requires either rollback or an explicit governance-level override with written rationale.

### 6.6 Governance Ledger (`services/governance/`)

**Public contract.** An append-only, cryptographically-signed ledger of every change that affects model behaviour, structured as a rolling PCCP.

**Entry structure (each entry is a PCCP triplet).**
- **Description of Modifications.** What changed: version bump, retraining event, threshold change, feature addition, scope expansion, cohort change.
- **Modification Protocol.** How it was produced: training set definition (with content-hash), evaluation protocol, acceptance criteria, shadow/canary results.
- **Impact Assessment.** Who is affected: which cohorts, which alerts, what magnitude of prediction change, what subgroup-differential effects, what safety implications.

Plus: cryptographic signature chain (each entry hash includes prior entry's hash); authorising party (engineer + clinical lead + governance officer, per RACI); rollback playbook (exact operator command and expected recovery time).

**Artifact generation.** Automatic production of:
- **Stakeholder reports** (quarterly): aggregate attribution, outcome trends, fairness metrics, known issues.
- **Procurement artifacts** (on demand): full model lineage, training data provenance, evaluation history, for contracts that require evidence.
- **Regulatory artifacts** (future-facing): structured as FDA PCCP submission components, ready to lift if/when the platform becomes SaMD-classified.

**Retention.** Ledger entries are retained indefinitely. Training set content-hashes retained indefinitely; training set bodies retained per market data-retention policy (HCF: 7 years; Aged Care: as per Aged Care Quality and Safety Commission; India: as per DPDP Act requirements).

---

## 7. Phase Breakdown

Four phases, fourteen steps, matching the Gap 20 shape. Each step is an independently shippable increment with its own tests, configs, and governance-ledger entry.

### Phase 1 — Attribution Foundation (4 steps)

The foundation that the rest of Gap 21 is built on. Ships the engine Gap 19 explicitly deferred.

**Step 1.1 — Outcome ingestion and ground-truth ledger.**
Extend the Gap 19 event ledger with structured outcome ingestion for both pilots. Outcomes arrive from multiple sources (hospital discharge feeds, claims data, mortality data, clinician confirmation) and must be deduplicated, reconciled, and timestamped. Conflicting outcome reports are surfaced for human adjudication rather than silently resolved. T4 closure is triggered either by outcome occurrence or by horizon expiry; both cases produce a ledger entry with the same schema so downstream consumers don't branch on it.

*Files*: `services/events/outcomes/ingestion.py`, `services/events/outcomes/reconciliation.py`, `services/events/outcomes/adjudication_queue.py`, `contracts/outcomes/schema_v1.yaml`, `configs/markets/hcf-chf/outcomes.yaml`, `configs/markets/aged-care-au/outcomes.yaml`, `tests/events/test_outcome_ingestion.py`, `tests/events/test_reconciliation.py`.

**Step 1.2 — T0→T4 lifecycle consolidation and causal annotation.**
Produce a consolidated per-alert record combining all lifecycle events with pre-alert features, clinician response (structured from Gap 18), intervention indicator (structured from the Care Transition Bridge — Gap 17), outcome, and horizon metadata. Add causal annotation: for each record, a "treatment strategy" label (intervention taken / override with reason / no response) and a "time-zero" timestamp aligned with the TTE protocol. This is the input format for the attribution engine.

*Files*: `services/events/consolidation/builder.py`, `services/events/consolidation/causal_annotator.py`, `contracts/consolidated_alert_record/schema_v1.yaml`, `tests/events/test_consolidation.py`, `tests/events/test_causal_annotation.py`.

**Step 1.3 — Causal attribution engine.**
The TTE protocol runner: eligibility, time zero, treatment strategy, outcome, causal contrast. Propensity score estimation with overlap diagnostics. Doubly-robust estimation (AIPW; TMLE where sample permits). Sensitivity analysis (E-value, tipping point). Per-patient attribution output with CI and E-value; aggregates per cohort, subgroup, and horizon produced by a second pass. Clinician_label mapping: point-estimate + E-value + CI combine into a discrete label (`prevented`, `no_effect_detected`, `outcome_despite_intervention`, `fragile_estimate`, `inconclusive`) that is safe to show a clinician.

*Files*: `services/attribution/tte_runner.py`, `services/attribution/propensity.py`, `services/attribution/doubly_robust.py`, `services/attribution/sensitivity.py`, `services/attribution/labels.py`, `configs/attribution/hcf-chf.yaml`, `configs/attribution/aged-care-au.yaml`, `configs/attribution/base.yaml`, `tests/attribution/test_tte_runner.py`, `tests/attribution/test_propensity_overlap.py`, `tests/attribution/test_doubly_robust.py`, `tests/attribution/test_sensitivity.py`, `tests/attribution/test_label_mapping.py`.

**Step 1.4 — Counterfactual outcome estimator.**
The no-treatment potential outcome head. Trained on the override cohort (patients who received alerts but no intervention) with per-patient-baseline anchoring. Produces the Ŷ(0) estimate for the Adherence-Weighted and Sampling-Weighted monitoring variants in Phase 2, and for the DR estimator's outcome regression in Step 1.3. Rigorous training-set construction: excludes patients whose overrides were themselves effective interventions (e.g., "already addressed" reason), uses IPW to adjust for selection into the override cohort, and includes a calibration check against the overall pre-intervention historical baseline.

*Files*: `services/attribution/counterfactual/model.py`, `services/attribution/counterfactual/training.py`, `services/attribution/counterfactual/evaluation.py`, `configs/counterfactual/base.yaml`, `tests/attribution/counterfactual/test_training.py`, `tests/attribution/counterfactual/test_override_cohort.py`.

**Phase 1 acceptance criteria.**
- End-to-end attribution runs for a single HCF CHF alert from T0 through attribution output in under 5 seconds (p95).
- E-values produced for ≥95% of attribution outputs.
- Overlap diagnostics automatically flag inconclusive cases; at least one synthetic-data test passes showing the engine correctly refuses to attribute when overlap fails.
- Clinical lead signs off on the five discrete clinician_labels as the right vocabulary.

### Phase 2 — Performance Monitoring (3 steps)

Builds the feedback-aware monitoring stack on top of the attribution foundation.

**Step 2.1 — Feedback-aware calibration and discrimination.**
ADWIN-based dynamic calibration drift detector, per market, per cohort, per model version. Calibration-CUSUM as second-tier confirmatory detector. Adherence-Weighted and Sampling-Weighted variants using Ŷ(0) from Step 1.4. Discrimination (AUROC, AUPRC) tracked in parallel but with explicit note that these are less reliable than calibration under feedback loops. Dashboard integration with Gap 18 worklist surfaces.

*Files*: `services/monitoring/drift/adwin.py`, `services/monitoring/drift/calibration_cusum.py`, `services/monitoring/metrics/feedback_aware.py`, `services/monitoring/metrics/discrimination.py`, `configs/monitoring/base.yaml`, `configs/monitoring/hcf-chf.yaml`, `configs/monitoring/aged-care-au.yaml`, `tests/monitoring/test_adwin.py`, `tests/monitoring/test_calibration_cusum.py`, `tests/monitoring/test_feedback_aware_metrics.py`.

**Step 2.2 — Subgroup and fairness monitoring.**
Per-subgroup running of all Tier 1 and Tier 2 detectors. Subgroup definitions from market YAML. Fairness-specific metrics: equalised odds on threshold-dependent outputs, demographic-conditional calibration, and a cross-subgroup calibration-gap metric. Alerts that appear disproportionately in a specific subgroup trigger a subgroup-differential impact review even without calibration drift (prevents the "works on average, fails for minorities" failure mode). Dashboards surface subgroup drift as a peer to aggregate drift.

*Files*: `services/monitoring/subgroups/definitions.py`, `services/monitoring/subgroups/fairness_metrics.py`, `services/monitoring/subgroups/differential_impact.py`, `configs/subgroups/hcf-chf.yaml`, `configs/subgroups/aged-care-au.yaml`, `tests/monitoring/test_subgroup_drift.py`, `tests/monitoring/test_fairness_metrics.py`.

**Step 2.3 — Drift-to-retrain incident workflow.**
When a Tier 2-confirmed drift fires, generate a structured retraining candidate event in the governance ledger, notify the on-call engineer plus clinical lead plus governance officer, and open an incident record with: root-cause hypotheses (feature drift vs concept drift vs feedback-loop artifact), recommended response (recalibrate vs retrain vs hold), and the feedback-aware vs naive metric comparison (to help distinguish "model failing" from "model succeeding so well standard monitoring thinks it's failing"). Incident resolution is ledger-recorded.

*Files*: `services/monitoring/incidents/detector.py`, `services/monitoring/incidents/workflow.py`, `services/monitoring/incidents/root_cause.py`, `tests/monitoring/incidents/test_workflow.py`, `docs/runbooks/drift_response.md`.

**Phase 2 acceptance criteria.**
- Synthetic drift injected at known timepoints is detected by ADWIN within configured latency bound.
- Synthetic feedback-loop effect (successful classifier label modification) produces divergence between naive and feedback-aware metrics that correctly fires the "succeeding-looks-like-failing" warning rather than a retraining alarm.
- Subgroup drift detection catches injected minority-cohort degradation that aggregate detection misses.
- End-to-end runbook exercise: drift injection → detection → incident → ledger entry → clinical lead sign-off, completed in under 2 hours.

### Phase 3 — Learning and Retraining (4 steps)

Closes the loop from observed outcomes and clinician feedback back into updated models.

**Step 3.1 — Structured clinician feedback ingestion.**
Override reason taxonomy wired into Gap 18 worklist. Structured UI for true/false-positive marks and counterfactual corrections. Free-text NLP pipeline: pattern-matching first pass, small domain-tuned LM second pass with validation gate, confidence-scored labels. Feedback records validated against Gap 19 lifecycle timestamps. Confidence weighting per reviewer. Inter-rater agreement sampling on 5% of cases.

*Files*: `services/learning/feedback/taxonomy.py`, `services/learning/feedback/structured_ingestion.py`, `services/learning/feedback/nlp_pipeline.py`, `services/learning/feedback/validation.py`, `services/learning/feedback/confidence_weighting.py`, `configs/feedback/taxonomy.yaml`, `configs/feedback/hcf-chf.yaml`, `configs/feedback/aged-care-au.yaml`, `tests/learning/feedback/test_taxonomy.py`, `tests/learning/feedback/test_nlp_pipeline.py`, `tests/learning/feedback/test_validation.py`.

**Step 3.2 — Active learning queue.**
Ensemble CoV uncertainty scoring on Gap 20 predictions. k-center diversity selection. Rarity and cost weighting. Queue sizing per clinician per day. Integration with Gap 18 worklist (queue items appear as a distinct "labelling requests" lane, separate from live alerts). Per-reviewer throughput tracking feeds back into queue sizing.

*Files*: `services/learning/active/uncertainty.py`, `services/learning/active/kcenter.py`, `services/learning/active/queue.py`, `services/learning/active/throughput.py`, `configs/active_learning/base.yaml`, `tests/learning/active/test_uncertainty.py`, `tests/learning/active/test_kcenter.py`, `tests/learning/active/test_queue.py`.

**Step 3.3 — Feedback-aware retraining pipeline.**
Training set construction with Adherence-Weighted sample weighting, temporal stratification, subgroup preservation. Model training with Gap 20's base algorithm (inherits Gap 20's hyperparameter choices). Offline evaluation against pre-specified acceptance criteria from governance ledger. Handoff to shadow deployment.

*Files*: `services/learning/retrain/training_set.py`, `services/learning/retrain/weighting.py`, `services/learning/retrain/trainer.py`, `services/learning/retrain/offline_eval.py`, `services/learning/retrain/acceptance_gate.py`, `configs/retrain/base.yaml`, `configs/retrain/hcf-chf.yaml`, `configs/retrain/aged-care-au.yaml`, `tests/learning/retrain/test_training_set.py`, `tests/learning/retrain/test_weighting.py`, `tests/learning/retrain/test_acceptance_gate.py`.

**Step 3.4 — Shadow → canary → full promotion with rollback.**
Shadow mode: candidate runs in parallel with production; predictions logged; agreement tracked; no clinician surface. Canary: random 10% traffic; attribution-weighted outcome comparison. Full promotion: candidate becomes production; previous production held in shadow for 4 weeks as hot-standby. Rollback: single-operator command; pre-rehearsed monthly. All stage transitions gated on governance-ledger authorisation.

*Files*: `services/learning/retrain/shadow.py`, `services/learning/retrain/canary.py`, `services/learning/retrain/promotion.py`, `services/learning/retrain/rollback.py`, `tests/learning/retrain/test_shadow.py`, `tests/learning/retrain/test_canary.py`, `tests/learning/retrain/test_rollback.py`, `docs/runbooks/retraining_deployment.md`, `docs/runbooks/emergency_rollback.md`.

**Phase 3 acceptance criteria.**
- Structured clinician feedback captured on ≥80% of overrides within 2 weeks of launch.
- NLP pipeline categorises free-text overrides at ≥75% agreement with human coders on a held-out sample.
- Active learning queue demonstrably shifts uncertainty distribution of labelled cases toward the high-uncertainty tail (measurable metric: median uncertainty of labelled cases increases by ≥50% vs random sampling).
- End-to-end dry run: drift alarm → governance authorisation → retraining → shadow → canary → full promotion → rollback exercise, completed without unplanned incidents.

### Phase 4 — Governance and Closed Loop (3 steps)

The governance ledger, dashboards, and external-facing artifacts that convert learning infrastructure into contractual and regulatory evidence.

**Step 4.1 — PCCP-style governance ledger.**
Append-only, cryptographically-signed (HMAC-SHA256 chain as baseline; Ed25519 signatures for entries authorising a production change). Every entry is the PCCP triplet: Description of Modifications, Modification Protocol, Impact Assessment. Authorisation RACI: engineer proposes, clinical lead reviews, governance officer signs. Rollback playbook attached to every entry. API for querying the ledger by model version, date range, change type, affected cohort.

*Files*: `services/governance/ledger/store.py`, `services/governance/ledger/entry.py`, `services/governance/ledger/signing.py`, `services/governance/ledger/raci.py`, `services/governance/ledger/query.py`, `contracts/governance/pccp_triplet_schema.yaml`, `tests/governance/test_ledger_append_only.py`, `tests/governance/test_signing_chain.py`, `tests/governance/test_raci.py`.

**Step 4.2 — Learning analytics dashboards.**
Per-market dashboard with: current model version, aggregate attribution (outcome reduction attributable to the program, with CI and E-value), subgroup attribution, calibration state (stable/warning/alarm) per cohort, feedback-aware vs naive metric comparison, active learning queue state, retraining pipeline state. Clinician view emphasises per-patient attribution (affirming the clinician's role in the causal chain); CFO view emphasises aggregate attribution and trend.

*Files*: `services/governance/dashboards/clinician_view.py`, `services/governance/dashboards/cfo_view.py`, `services/governance/dashboards/engineering_view.py`, `services/governance/dashboards/aggregation.py`, `configs/dashboards/hcf-chf.yaml`, `configs/dashboards/aged-care-au.yaml`, `tests/governance/test_dashboard_aggregation.py`.

**Step 4.3 — Stakeholder, procurement, and regulatory artifact generation.**
Quarterly stakeholder reports: auto-generated narrative + figures from dashboard data, with clinical-lead review before release. Procurement artifacts: full model lineage, training data provenance, evaluation history, on-demand. Regulatory-ready artifacts: structured as FDA PCCP submission components (Description of Modifications, Modification Protocol, Impact Assessment for each change), structured also as EU AI Act Article 15 post-market monitoring evidence, lift-ready if and when the platform crosses into SaMD classification.

*Files*: `services/governance/artifacts/stakeholder_report.py`, `services/governance/artifacts/procurement_pack.py`, `services/governance/artifacts/regulatory_pack.py`, `services/governance/artifacts/narrative_generator.py`, `configs/artifacts/templates/`, `tests/governance/test_stakeholder_report.py`, `tests/governance/test_regulatory_pack.py`, `docs/governance/pccp_mapping.md`, `docs/governance/eu_ai_act_mapping.md`.

**Phase 4 acceptance criteria.**
- Governance ledger is verifiably append-only and signing chain validates from genesis.
- Three RACI actors can independently authorise / reject changes; unauthorised changes cannot reach production.
- Stakeholder report generated automatically for Q1 pilot period; clinical lead approves with <20% edit rate on the auto-generated narrative.
- Procurement pack lifted successfully for a synthetic procurement request within 24 hours.

---

## 8. Integration with Gap 19 (T0→T4 Lifecycle) and Gap 20 (Predictive Risk Layer)

Gap 21 depends on both. Clean interface boundaries matter; this section defines them.

### 8.1 Gap 19 interface

**What Gap 21 consumes.** The T0→T4 event ledger. Specifically: per-alert records with pre-alert feature snapshot, prediction snapshot (Gap 20 output), alert firing timestamp (T1), clinician acknowledgement (T2), intervention initiation or override (T3), outcome observation or horizon expiry (T4).

**What Gap 21 adds to Gap 19.** Step 1.1 extends the ledger with structured outcome ingestion. Step 1.2 adds causal annotation to consolidated records. These are additive (new fields on existing records, new tables for new data); they do not modify the Gap 19 contract.

**What changes in Gap 19.** Nothing in Gap 19's public contract. Internal storage may need to support the new fields; this is a Sprint 2 task but purely mechanical.

### 8.2 Gap 20 interface

**What Gap 21 consumes.** Per-alert risk score, model version, prediction feature vector, and ensemble member predictions (for uncertainty estimation). Gap 20's plan should include a "prediction snapshot" contract; Gap 21 reads from it.

**What Gap 21 adds to Gap 20.** The retraining pipeline produces new model versions that Gap 20 serves. The interface is model-registry-based: Gap 21 publishes a candidate version with metadata (training set hash, acceptance-criterion results, canary results); Gap 20 reads the registry and serves the currently-promoted version. Promotion is a governance-ledger action that atomically switches the served version.

**What changes in Gap 20.** Gap 20 must support serving multiple model versions simultaneously (production, shadow, canary) and routing traffic between them. This should be factored into Gap 20's implementation plan. The actual routing logic lives in Gap 20; Gap 21 only tells it which version is in which slot.

### 8.3 Dependency direction and release sequencing

Gap 21 is dependent on Gap 20 in production but not in planning. Planning proceeds in parallel; implementation of Gap 21 Phase 1 can begin once Gap 19's outcome-ingestion extension (Step 1.1) is stable — which is a Gap 21 deliverable, so effectively Phase 1 is unblocked now. Gap 21 Phase 2 depends on Gap 20 serving predictions; if Gap 20 slips, Phase 2 can be partially implemented against a Gap 20 stub (a legacy V4 pattern model with prediction snapshot interface). Gap 21 Phases 3 and 4 depend on Gap 20 shipping.

---

## 9. Pilot-Specific Considerations

The two pilots are different enough that the architecture must accommodate them via market-config overrides, not fork.

### 9.1 HCF CHF (Australia)

**Outcome definitions.** 30-day all-cause readmission (primary), 30-day CHF-specific readmission (secondary), 90-day all-cause readmission (tertiary), 30-day all-cause mortality (safety), emergency department presentation within 30 days (secondary), functional decline at 30 days (self-reported; tertiary). Outcomes ingested from HCF claims feed and hospital discharge feeds, reconciled per Step 1.1.

**Time zero.** Hospital discharge timestamp when the pathway is post-discharge readmission prevention; CHF diagnosis confirmation when the pathway is acute-on-chronic trajectory (Gap 16). Both are well-structured in the HCF data model.

**Treatment strategies.** The TTE "treatment" is the clinician's response to the alert, structured as {`nurse_phone_followup`, `gp_visit_scheduled`, `specialist_referral`, `medication_review`, `device_monitoring_enrolled`, `none`/override, ...}. These come from Gap 17 Care Transition Bridge and Gap 18 Worklist action records.

**Subgroups for fairness monitoring.** Age bands (<65, 65–74, 75–84, 85+), sex, NYHA class, diabetes comorbidity, CKD comorbidity, LVEF band where available, metro/regional postcode, Aboriginal and Torres Strait Islander status where consented, private/public insurance crossover where relevant.

**Attribution horizons.** 30-day (primary evaluation), 90-day (secondary), 1-year (exploratory, not for primary attribution claims).

**Retraining cadence.** Target quarterly with event-triggered additions; the canary window is 4 weeks because 30-day outcomes close quickly relative to the window.

**Specific risks.**
- **Readmission is highly preventable-vs-not heterogeneous.** Some readmissions (e.g., planned procedures, deterioration despite optimal care) are not responsive to any outpatient intervention. Attribution must distinguish *preventable* readmissions (where attribution makes sense) from *non-preventable* (where attribution is meaningless). This requires a preventability flag, either structured in the discharge data or estimated by a sub-model; Gap 21 Step 1.3 has a preventability-flag input that, if present, restricts attribution to the preventable subset.
- **Seasonality.** CHF readmissions spike in winter. Attribution estimates computed quarterly will see this; the temporally-stratified training set handles it for retraining, but quarterly comparisons need year-over-year framing, not quarter-over-quarter.

### 9.2 Aged Care (Australia)

**Outcome definitions.** Hospital admission within 90 days (primary), hospital admission within 180 days (secondary), transition to higher level of care (secondary), pressure injury incidence (safety), fall with injury (safety), mortality within 90 days (safety). Outcomes ingested from Aged Care Quality and Safety Commission data where available, facility data otherwise.

**Time zero.** Varies by pathway: admission to residential aged care; post-hospital-discharge return to RAC; acute event detection (Gap 16 acute-on-chronic triggers).

**Treatment strategies.** Geriatrician review, pharmacist medication review, allied health intervention, care plan revision, family conference, GP home visit, override. The structured taxonomy is smaller than HCF CHF because RAC has fewer intervention pathways.

**Subgroups for fairness monitoring.** RAC level at enrolment, dementia diagnosis presence, polypharmacy bands, frailty score bands, state/territory, residential vs home-care package, CALD (Culturally And Linguistically Diverse) status where consented.

**Attribution horizons.** 90-day (primary), 180-day (secondary), 1-year (exploratory for mortality; primary for transition-to-higher-care).

**Retraining cadence.** Target semi-annual because outcome horizons are longer; canary window is 8 weeks.

**Specific risks.**
- **Outcome ascertainment is harder.** Hospital admissions for aged care residents are frequently coded under the hospital's record, not the RAC's, and reconciliation is imperfect. Step 1.1's adjudication queue will see more volume here than in HCF CHF; staff it accordingly.
- **Resident consent for subgroup data is more complex.** CALD and Indigenous status must be opt-in; a meaningful fraction of residents cannot provide informed consent themselves. Fairness monitoring on unconsented subgroups is prohibited; the config supports a "coverage warning" when a subgroup's labelled sample fraction drops below a threshold, which should surface on dashboards rather than produce a silent blind spot.
- **Functional decline is the real outcome, admission is the proxy.** The program's clinical intent is to preserve function. Admissions are an imperfect proxy. Gap 21 should flag this in attribution outputs — "admission avoidance estimate: X ± Y; functional preservation not directly estimated" — rather than claiming the admission number is the clinical win.

### 9.3 India pilot (early design, not yet scoped for Sprint 2)

The India market configs exist for platform-wide consistency but the India pilot has not specified outcome definitions or data feeds at Gap 21's level of detail. Placeholder configs ship with "SCOPE_TBD" flags that the attribution engine refuses to run against. When the India pilot is scoped, Gap 21 accepts the config additions via standard market-override pattern.

---

## 10. Validation, Safety, and Rollback

Three concerns that cut across every phase.

### 10.1 Pre-deployment validation

- **Unit tests** for every public function (target: 90% line coverage, 100% branch coverage on safety-critical paths — attribution output labels, governance ledger signing, rollback command).
- **Integration tests** for every service boundary (attribution ↔ events, monitoring ↔ predictions, retraining ↔ model registry, governance ↔ everything).
- **Property-based tests** on the attribution engine: synthetic ground-truth causal-effect scenarios, verify that IPW/DR estimators recover the true effect within acceptable bounds; synthetic feedback-loop scenarios, verify that feedback-aware metrics recover the no-treatment potential outcome AUROC.
- **Adversarial tests**: attempt to corrupt the governance ledger (should fail); attempt to promote a model without authorisation (should fail); attempt to roll back to a non-existent version (should fail cleanly).

### 10.2 Shadow and canary

Every model promotion goes through shadow (Stage 1) and canary (Stage 2) as described in Step 3.4. The point is not just to catch bugs but to validate against attribution-weighted outcome metrics, which are the actual contract. A candidate that improves AUROC but worsens attribution-weighted outcome reduction fails acceptance — the classifier got "better" at the labelled task but worse at the underlying clinical goal, which is exactly the feedback-loop failure mode Gap 21 is designed to prevent.

### 10.3 Rollback

**Scope.** Rollback covers: the active model version, the active calibration recipe, the active threshold set, the active active-learning queue policy, and the active attribution configuration. It does not cover the governance ledger itself (which is append-only; you append a rollback entry, you don't rewind).

**Mechanics.** A single operator command `gap21-rollback --target-version <version>` atomically switches the model registry pointer, invalidates in-flight shadow/canary runs, writes a rollback entry to the governance ledger, and notifies RACI parties. Expected recovery time: ≤15 minutes from command to fully-rolled-back production state.

**Rehearsal.** Monthly rollback drill: pick a recent promotion, rollback to its predecessor, run the test-alert suite, rollback back. This is a scheduled calendar event, not an on-demand exercise. The metric is "time from drill start to normal operations resumed"; target ≤30 minutes end-to-end.

**Rollback-preventing modes.** Two situations prevent rollback: an attribution claim has already been made in an external-facing artifact (a CFO quarterly report citing X prevented readmissions under version V) — rollback must still be possible but requires an explicit amendment to the external artifact, and the governance ledger records both the rollback and the artifact-retraction; clinician-affecting changes that are not binary-reversible (e.g., a threshold change that was announced to clinicians and then trained into their workflow) — rollback is possible but requires a communications plan.

### 10.4 Safety-critical guardrails (cannot be disabled)

- **No attribution without overlap.** If propensity overlap diagnostics fail, attribution returns `inconclusive`. This is not a flag that can be overridden by a configuration; it is a hard guard.
- **No retrain without authorisation.** The retraining pipeline refuses to run without a signed governance-ledger authorisation entry. No "auto-retrain" mode exists.
- **No subgroup monitoring opt-out.** Market configs can add subgroups but cannot remove fairness-target subgroups that have been configured by the governance process. Removing a subgroup is itself a governance-ledger action requiring RACI sign-off.
- **No silent change to outcome definitions.** Changing an outcome definition is a governance-ledger action with a 30-day freeze on attribution claims under the new definition to prevent inadvertently conflating pre- and post-change estimates.

---

## 11. File Map

Full map, ~50 files, grouped by service. Configuration files, tests, and docs are listed. Inherited patterns from Gaps 14–19 (KB-23 folded service, YAML market configs, India/Australia overrides, clinician_label vs technical_label conventions, temporal state machine integration, validation gates, per-patient baselines) apply throughout and are called out where non-obvious.

**Attribution service (`services/attribution/`)**
1. `services/attribution/tte_runner.py` — target trial emulation protocol runner.
2. `services/attribution/propensity.py` — propensity score estimator with overlap diagnostics.
3. `services/attribution/doubly_robust.py` — AIPW/TMLE estimator.
4. `services/attribution/sensitivity.py` — E-value, tipping-point analysis.
5. `services/attribution/labels.py` — clinician_label mapping (follows KB-23 label-separation convention).
6. `services/attribution/counterfactual/model.py` — no-treatment potential outcome head.
7. `services/attribution/counterfactual/training.py` — override-cohort training with IPW adjustment.
8. `services/attribution/counterfactual/evaluation.py` — calibration check against historical baseline.
9. `configs/attribution/base.yaml` — base configuration.
10. `configs/attribution/hcf-chf.yaml` — HCF CHF market override (horizons, outcome defs).
11. `configs/attribution/aged-care-au.yaml` — Aged Care market override.
12. `configs/attribution/india-pilot.yaml` — placeholder with SCOPE_TBD flags.

**Events service extension (`services/events/`)**
13. `services/events/outcomes/ingestion.py` — outcome ingestion pipeline.
14. `services/events/outcomes/reconciliation.py` — cross-source outcome reconciliation.
15. `services/events/outcomes/adjudication_queue.py` — human-adjudication queue for conflicts.
16. `services/events/consolidation/builder.py` — per-alert consolidated record builder.
17. `services/events/consolidation/causal_annotator.py` — TTE-semantic annotation.
18. `contracts/outcomes/schema_v1.yaml` — outcome record contract.
19. `contracts/consolidated_alert_record/schema_v1.yaml` — consolidated record contract.

**Monitoring service extension (`services/monitoring/`)**
20. `services/monitoring/drift/adwin.py` — ADWIN calibration drift detector.
21. `services/monitoring/drift/calibration_cusum.py` — calibration-CUSUM confirmatory detector.
22. `services/monitoring/metrics/feedback_aware.py` — Adherence-Weighted / Sampling-Weighted metrics.
23. `services/monitoring/metrics/discrimination.py` — AUROC/AUPRC with feedback-aware notes.
24. `services/monitoring/subgroups/definitions.py` — subgroup definition resolver.
25. `services/monitoring/subgroups/fairness_metrics.py` — equalised odds, calibration-gap, etc.
26. `services/monitoring/subgroups/differential_impact.py` — subgroup differential impact review.
27. `services/monitoring/incidents/detector.py` — drift-to-incident generator.
28. `services/monitoring/incidents/workflow.py` — incident workflow.
29. `services/monitoring/incidents/root_cause.py` — structured root-cause hypothesis generator.

**Learning service (`services/learning/`)**
30. `services/learning/feedback/taxonomy.py` — override reason taxonomy + clinician_label/technical_label mapping.
31. `services/learning/feedback/structured_ingestion.py` — structured feedback records.
32. `services/learning/feedback/nlp_pipeline.py` — free-text categorisation.
33. `services/learning/feedback/validation.py` — feedback validation gates.
34. `services/learning/feedback/confidence_weighting.py` — per-reviewer confidence weighting.
35. `services/learning/active/uncertainty.py` — ensemble CoV scoring.
36. `services/learning/active/kcenter.py` — k-center diversity selection.
37. `services/learning/active/queue.py` — queue sizing and management.
38. `services/learning/active/throughput.py` — per-reviewer throughput tracking.
39. `services/learning/retrain/training_set.py` — training set construction with feedback-aware weights.
40. `services/learning/retrain/weighting.py` — Adherence/Sampling weights.
41. `services/learning/retrain/trainer.py` — model training (wraps Gap 20 base algorithm).
42. `services/learning/retrain/offline_eval.py` — acceptance-criterion evaluation.
43. `services/learning/retrain/acceptance_gate.py` — accept/reject against pre-specified criteria.
44. `services/learning/retrain/shadow.py` — shadow deployment manager.
45. `services/learning/retrain/canary.py` — canary deployment with A/B comparison.
46. `services/learning/retrain/promotion.py` — full promotion with hot-standby.
47. `services/learning/retrain/rollback.py` — single-command rollback.

**Governance service (`services/governance/`)**
48. `services/governance/ledger/store.py` — append-only ledger storage.
49. `services/governance/ledger/entry.py` — PCCP triplet entry structure.
50. `services/governance/ledger/signing.py` — HMAC chain + Ed25519 signatures.
51. `services/governance/ledger/raci.py` — RACI authorisation workflow.
52. `services/governance/ledger/query.py` — ledger query API.
53. `services/governance/dashboards/clinician_view.py`
54. `services/governance/dashboards/cfo_view.py`
55. `services/governance/dashboards/engineering_view.py`
56. `services/governance/dashboards/aggregation.py` — shared aggregation layer.
57. `services/governance/artifacts/stakeholder_report.py` — quarterly report generator.
58. `services/governance/artifacts/procurement_pack.py` — on-demand procurement artifacts.
59. `services/governance/artifacts/regulatory_pack.py` — PCCP-structured regulatory artifacts.
60. `services/governance/artifacts/narrative_generator.py` — auto-narrative for reports.

**Configs (beyond attribution, already listed)**
61. `configs/monitoring/base.yaml`, `configs/monitoring/hcf-chf.yaml`, `configs/monitoring/aged-care-au.yaml`
62. `configs/subgroups/hcf-chf.yaml`, `configs/subgroups/aged-care-au.yaml`
63. `configs/feedback/taxonomy.yaml`, `configs/feedback/hcf-chf.yaml`, `configs/feedback/aged-care-au.yaml`
64. `configs/active_learning/base.yaml`
65. `configs/retrain/base.yaml`, `configs/retrain/hcf-chf.yaml`, `configs/retrain/aged-care-au.yaml`
66. `configs/dashboards/hcf-chf.yaml`, `configs/dashboards/aged-care-au.yaml`
67. `configs/artifacts/templates/`
68. `configs/markets/hcf-chf/outcomes.yaml`, `configs/markets/aged-care-au/outcomes.yaml`

**Contracts**
69. `contracts/governance/pccp_triplet_schema.yaml`

**Docs and runbooks**
70. `docs/runbooks/drift_response.md`
71. `docs/runbooks/retraining_deployment.md`
72. `docs/runbooks/emergency_rollback.md`
73. `docs/governance/pccp_mapping.md` — how each ledger entry maps to FDA PCCP components.
74. `docs/governance/eu_ai_act_mapping.md` — how each ledger entry maps to EU AI Act Article 15 evidence.
75. `docs/architecture/gap21_overview.md` — architecture overview (this document, trimmed).

**Tests (one per non-trivial implementation file; not enumerated exhaustively to avoid bloat, but present for every file above)** — conservatively ~50 test files, giving Gap 21 a total of roughly 50 implementation + 50 test files. Given the prior thread's scale reference, this is consistent.

---

## 12. Governance Artifacts — Detail

This section details the concrete structure of each governance output. These are not placeholders; they are the shapes that downstream consumers (clinical leads, CFO, procurement, future regulatory) will need.

### 12.1 PCCP triplet structure per ledger entry

```yaml
entry_id: <uuid>
timestamp: <iso8601>
prior_entry_hash: <sha256>
signatures:
  engineer: <ed25519>
  clinical_lead: <ed25519>
  governance_officer: <ed25519>

description_of_modifications:
  change_type: <retrain | recalibrate | threshold_change | feature_add | scope_change | rollback>
  summary: <free text, max 500 chars>
  affected_models: [<model_id>]
  affected_cohorts: [<cohort_id>]
  affected_markets: [<market_id>]

modification_protocol:
  training_set:
    hash: <sha256>
    size: <int>
    temporal_window: [<iso8601>, <iso8601>]
    weighting_scheme: <adherence | sampling | unweighted>
    subgroup_stratification: <dict>
  acceptance_criteria:
    aggregate_auroc_min: <float>
    calibration_in_large_max: <float>
    subgroup_auroc_min: <dict>
    attribution_weighted_outcome_change: <none_worse_than | improvement>
    e_value_min: <float>
  shadow_results: <nested dict, populated after shadow stage>
  canary_results: <nested dict, populated after canary stage>

impact_assessment:
  prediction_change_magnitude: <quantiles of Δ risk score across cohort>
  subgroup_differential_impact: <per subgroup, magnitude and direction>
  expected_alert_volume_change: <absolute and relative>
  safety_considerations: <free text>
  rollback_playbook:
    command: <string>
    expected_recovery_time_minutes: <int>
    last_rehearsed: <iso8601>
```

### 12.2 Stakeholder quarterly report

Sections:
1. Executive summary — headline attribution numbers with CI and E-value.
2. Outcome trends — line charts, YoY framing for seasonal outcomes.
3. Subgroup breakdown — attribution and fairness metrics per subgroup.
4. Changes this quarter — summary of governance-ledger entries: versions promoted, thresholds changed, any rollbacks.
5. Monitoring state — current drift status per cohort, any open incidents.
6. Known issues and forward work.

Auto-generated narrative, clinical-lead reviewed, CFO + clinical-lead co-signed.

### 12.3 Procurement pack

Sections:
1. Full model lineage from genesis to current version.
2. Training data provenance for each version — data sources, temporal windows, content hashes.
3. Evaluation history — acceptance-criterion results for each version.
4. Subgroup performance history.
5. Governance RACI evidence.
6. Monitoring infrastructure description.
7. Rollback capability evidence (rehearsal history).

Generated on demand within 24 hours of request; versioned to capture the state at request time.

### 12.4 Regulatory pack (future-facing)

Structured as lift-ready FDA PCCP submission components:
- **Description of Modifications** — aggregated from all ledger entries in the submission window.
- **Modification Protocol** — the methodology documented across the codebase, packaged as a single document.
- **Impact Assessment** — aggregated across ledger entries, showing cumulative impact.

Plus EU AI Act Article 15 post-market monitoring evidence:
- Logs of performance metrics over time.
- Incident records and resolutions.
- Subgroup fairness evidence.
- Change control evidence.

Plus Australian TGA SaMD evidence package (if/when platform crosses into SaMD classification):
- Clinical evaluation report, essential principles checklist, risk management file, verification and validation records — all traceable to governance ledger entries.

---

## 13. Open Questions and Deferrals

Explicit. These are not omissions; they are choices to defer with reasons.

**Deferred.**

1. **Reinforcement learning from clinical feedback (RLCF) with digital twins.** The AAAI 2026 work (batch-constrained Q-learning ensemble, digital twin–powered reward from counterfactual treatment effect, uncertainty-driven expert querying) is a natural extension of Gap 21's active learning and attribution engines, but the implementation complexity and validation burden is substantial. Gap 22 candidate.

2. **Full federated learning across HCF and Aged Care cohorts.** The federated TTE framework is well-characterised in the 2025 *npj Digital Medicine* paper and would let the platform pool learning across pilots without patient-level data leaving site boundaries. Gap 21's attribution outputs are shaped to be federated-aggregatable, but the federation infrastructure itself is Gap 23 candidate.

3. **Generative LLM-based clinical reasoning for attribution explanation.** Promising (an LLM-as-judge that reviews a case and generates a narrative attribution explanation) but the hallucination risk in a causal inference context is not yet acceptable. The structured attribution labels (from Step 1.3) are safer for Sprint 2. LLM-narrative is a Sprint 3+ candidate.

4. **Time-varying treatment effects and dynamic treatment regimes.** Gap 21 attributes the first alert → first intervention pair. Dynamic treatment regimes (sequences of alerts and responses) are richer and more realistic but require g-methods (g-computation, marginal structural models, g-estimation of structural nested models) that are substantially more complex. Gap 22/23 candidate.

5. **Causal discovery from observational patterns.** The current architecture assumes a causal structure specified by clinicians (what intervenes on what). Automated causal discovery (from data, surface candidate DAGs) is possible but not yet reliable enough for clinical use. Not on roadmap.

**Open questions requiring clinical/business input.**

6. **Override-reason taxonomy vocabulary.** The eight-entry proposal in §6.3 is my best guess based on published CDS literature; the clinical leads for HCF and Aged Care may have strong opinions about what's actually actionable in their workflow. Requires a 1-hour clinical-lead workshop before Step 3.1 ships.

7. **Preventability flag for HCF readmissions.** Whether HCF's discharge data reliably carries a preventability assessment, or whether Gap 21 needs to model preventability itself, affects Phase 1 scope. Requires a data-availability review with HCF's analytics team.

8. **Consent model for fairness-subgroup data in Aged Care.** CALD and Indigenous status are sensitive and consent varies. The "coverage warning" mechanism (§9.2) is the current proposed treatment; whether this is sufficient for the Aged Care Quality and Safety Commission's expectations requires legal/compliance review.

9. **External outcome-data latency for Aged Care admissions.** If hospital-admission data for RAC residents lands with a 30–90 day lag, the 90-day primary attribution horizon is materially delayed. May need an interim claims-based estimate, flagged as provisional, for quarterly reporting. Requires a data-feed characterisation with Aged Care pilot partners.

10. **Acceptance criterion for attribution-weighted outcome change.** The acceptance gate in Step 3.3 says "not worse than production." This is deliberately conservative to avoid inadvertently promoting an apparently-better model that has a feedback-loop artifact. Whether to move to a stronger criterion ("materially better with statistical significance") once the platform has a stable baseline is a question for mid-2026.

**Known limitations that will not be closed in Sprint 2.**

11. **Unmeasured confounding is always possible.** The E-value is a sensitivity bound, not a resolution. An attribution with E-value 1.8 can still be explained away by an unmeasured confounder of strength 1.8+. The clinician_label `fragile_estimate` exists precisely to flag this; stakeholders must understand that attribution is never proof of causality, only structured evidence.

12. **The counterfactual model is trained on the override cohort, which is selected.** Even with IPW adjustment, patients who get overridden are systematically different from patients who don't. The counterfactual estimates will have residual bias. Benchmarking against historical pre-intervention periods (when there was no alert to override) is a partial mitigation; fully addressing it requires either randomised-alert-withholding trials (ethically fraught) or federated pooling of long enough historical periods.

---

## 14. Success Metrics

How we know Gap 21 worked. Listed as pilot-specific leading metrics, not "code shipped" metrics.

**For HCF CHF pilot.**
- Attribution-weighted 30-day all-cause readmission reduction, point estimate with 95% CI and E-value, by end of Q2.
- Subgroup attribution within ±20% of aggregate attribution, i.e., no subgroup sees materially worse attribution than the cohort average.
- Clinician override rate stable or declining quarter-on-quarter, with overrides showing concentration in the `already_addressed` and `not_clinically_relevant` categories (healthy) rather than `disagree_with_risk` or `features_inaccurate` (concerning).
- Calibration stable across the pilot period (no Tier-2 drift alarms fired, or fired and resolved through authorised recalibration within 2 weeks).

**For Aged Care pilot.**
- Attribution-weighted 90-day admission reduction, point estimate with 95% CI and E-value, by end of Q3 (later than HCF because of longer outcome horizons).
- Transition-to-higher-care attribution flagged but not used as primary metric until 12-month outcomes available.
- CALD-cohort and Indigenous-cohort coverage ≥50% where consent permits, and attribution reported separately where coverage reaches the configured threshold.
- Rollback drill completion monthly without unplanned incidents.

**For the platform.**
- Governance ledger append-only integrity verifiable from genesis at every audit.
- Retraining cadence achieved: at least one governance-authorised retraining event per quarter for HCF CHF, one per half-year for Aged Care.
- Clinician labelling throughput: ≥80% of active-learning queue items labelled within their SLA.
- Zero silent model changes (every production-affecting change has a governance-ledger entry).

---

## 15. References and Further Reading

Primary methodological references cited in this plan. Grouped by topic; dates are most-recent versions I could locate.

**Target Trial Emulation.**
- Hernán MA, Robins JM. *Using big data to emulate a target trial when a randomized trial is not available.* American Journal of Epidemiology 2016; 183(8):758–764.
- Hernán MA, Wang W, Leaf DE. *Target trial emulation: a framework for causal inference from observational data.* JAMA 2022;328(24):2446–2447.
- *An operational target trial emulation framework for causal inference using electronic health record data.* npj Digital Medicine 2026.
- *Target Trial Emulation for Regulatory and Clinical Decision Making in Cancer.* Journal of Clinical Oncology 2026.
- *Federated target trial emulation using distributed observational data for treatment effect estimation.* npj Digital Medicine 2025.

**Feedback loops and monitoring.**
- Kim GYE, Corbin CK, Grolleau F, et al. *Monitoring strategies for continuous evaluation of deployed clinical prediction models.* Journal of Biomedical Informatics 2025 — **foundational for Gap 21 monitoring design**.
- Corbin CK, et al. *Avoiding biased clinical machine learning model performance estimates in the presence of label selection.* 2022.
- van Amsterdam WAC, et al. *Feedback loops in intensive care unit prognostic models: an under-recognised threat to clinical validity.* Lancet Digital Health 2025.
- Joshi S, Urteaga I, van Amsterdam WAC, et al. *AI as an intervention: improving clinical outcomes relies on a causal approach to AI development and validation.* JAMIA 2025;32:589–594.
- Davis SE, Greevy RA, Lasko TA, Walsh CG, Matheny ME. *Detection of calibration drift in clinical prediction models to inform model updating.* Journal of Biomedical Informatics 2020;112:103611.
- Pagan N, et al. *A Classification of Feedback Loops and Their Relation to Biases in Automated Decision-Making Systems.* ACM EAAMO 2023.

**Causal inference methods.**
- Hernán MA, Robins JM. *Causal Inference: What If.* 2020 (standard reference).
- Rosenbaum PR, Rubin DB. *The central role of the propensity score in observational studies for causal effects.* Biometrika 1983;70:41–55.
- Van der Laan MJ, Rose S. *Targeted Learning: Causal Inference for Observational and Experimental Data.* Springer 2011 (TMLE reference).
- Feuerriegel S, Frauen D, Melnychuk V, et al. *Causal machine learning for predicting treatment outcomes.* Nature Medicine 2024;30:958–968.
- *Learning Counterfactual Outcomes Under Rank Preservation.* arXiv 2025.
- *Propensity score weighting across counterfactual worlds: longitudinal effects under positivity violations.* arXiv 2025.

**Regulatory frameworks.**
- US FDA. *Marketing Submission Recommendations for a Predetermined Change Control Plan for Artificial Intelligence-Enabled Device Software Functions.* Final guidance, December 2024.
- FDA, Health Canada, MHRA. *Guiding Principles for Predetermined Change Control Plans.* August 2025.
- *Predetermined Change Control Plans: Guiding Principles for Advancing Safe, Effective, and High-Quality AI-ML Technologies.* JMIR AI 2025.
- European Union. *Artificial Intelligence Act.* Regulation (EU) 2024/1689, Article 15 (post-market monitoring for high-risk AI).
- UK MHRA. *Software and AI as a Medical Device Change Programme — Roadmap.* 2024.

**Active learning and uncertainty.**
- Sener O, Savarese S. *Active Learning for Convolutional Neural Networks: A Core-Set Approach.* ICLR 2018 (k-center reference).
- Lakshminarayanan B, Pritzel A, Blundell C. *Simple and scalable predictive uncertainty estimation using deep ensembles.* NeurIPS 2017.
- *Reinforcement Learning enhanced Online Adaptive Clinical Decision Support via Digital Twin powered Policy and Treatment Effect optimized Reward.* AAAI 2026.

**Calibration and drift detection.**
- Bifet A, Gavaldà R. *Learning from time-changing data with adaptive windowing.* SDM 2007 (ADWIN reference).
- Haq A. *Dynamic probability control limits for the adaptive MEWMA chart.* 2024.
- *Monitoring the calibration of probability forecasts with an application to concept drift detection involving image classification.* arXiv 2025 (calibration-CUSUM).

**Alert fatigue and clinical feedback.**
- Poly TN, Islam MM, Muhtar MS, et al. *Machine Learning Approach to Reduce Alert Fatigue Using a Disease Medication–Related Clinical Decision Support System.* JMIR 2020.
- *Clinical Decision Support Alert Appropriateness: A Review and Proposal for Improvement.* 2014.
- AHRQ PSNet. *Alert Fatigue.* Primer (ongoing).

---

## 16. Suggested Sequencing and Handoff

The natural order:

- **Weeks 1–3.** Phase 1 Steps 1.1 and 1.2 (outcome ingestion, consolidation). These are mechanical extensions to Gap 19 and unblock everything else.
- **Weeks 3–6.** Phase 1 Steps 1.3 and 1.4 (attribution engine, counterfactual estimator). Most methodologically dense portion of the work; worth the time.
- **Weeks 6–9.** Phase 2 (feedback-aware monitoring, subgroup monitoring, incident workflow). Can run partly in parallel with Phase 1 completion.
- **Weeks 9–13.** Phase 3 (feedback ingestion, active learning, retraining with shadow/canary/rollback). Phase 3 benefits from having real alert traffic to act on; if HCF pilot is still in ramp-up when Phase 3 begins, schedule the first actual retrain for a synthetic or shadow-only rehearsal.
- **Weeks 13–16.** Phase 4 (governance ledger, dashboards, artifact generation). Some Phase 4 pieces (the ledger in particular) could shift earlier; if a stakeholder artifact is needed before end of sprint, pull Phase 4 Step 4.1 forward.

**Handoff priorities for the team picking this up.**

1. First 30 minutes: read §5 (architecture), §6 (six engines), §7 (phase breakdown).
2. Next hour: decide on the §13 open questions that need clinical/business input and schedule the workshops.
3. Day 1: map the file layout in §11 to the existing codebase conventions; the service paths I proposed follow the KB-23 folded-service pattern but actual package names should align with the existing structure.
4. Day 2: spike Step 1.1 (outcome ingestion) to establish the data contract with HCF's claims feed and Aged Care's data sources. This is the single biggest unknown that could invalidate downstream scope.
5. Week 1: wire up the governance ledger (Step 4.1) as the first "service skeleton" even though it's Phase 4 — it's where every subsequent change should land, and having it operational from the start makes the rest of the sprint's changes visible.

**What "done" looks like for Sprint 2.**
- Gap 19's outcome attribution engine is no longer deferred.
- The HCF CHF pilot can make defensible attribution claims at quarterly-report time.
- The Aged Care pilot has the instrumentation to report in its longer cadence.
- At least one governance-authorised retraining event has completed end-to-end (shadow → canary → full) in at least one market, even if initially on synthetic drift.
- The platform has a complete, lift-ready regulatory artifact set for each pilot, filed against any future SaMD classification query.

---

*End of plan. The closed loop is the loop that learns without drifting; this is the shape of one.*
