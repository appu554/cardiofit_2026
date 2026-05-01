# Gap 22 — Prescriptive AI: Personalised Intervention Recommendation

**Implementation Guidelines, V5 Sprint 3**

Status: Planning. Prerequisites in flight (Gap 20 Predictive Risk Layer, Gap 21 Closed-Loop Outcome Learning). Two pilots continuing: HCF CHF (readmission reduction) and Aged Care (admission reduction). This gap shifts the platform from *predictive* to *prescriptive*.

---

## 1. Executive Summary

Gap 20 told the platform **what is likely to happen**. Gap 21 told the platform **what did happen and why**. Gap 22 tells the platform **what it should recommend happen next**.

The move from prediction to prescription is more than a new feature — it is a different kind of claim. A predictive system that says "this patient has a 42% 30-day readmission risk" passes responsibility back to the clinician. A prescriptive system that says "for this patient, a nurse phone follow-up at 48 hours is the intervention most likely to reduce readmission risk by 18 ± 6 percentage points" is a substantive clinical recommendation that the clinician can act on or reject. The regulatory, methodological, and clinical bars are all higher.

Concretely, Gap 22 delivers six capabilities:

1. **Conditional Average Treatment Effect (CATE) Engine** — per-patient, per-intervention estimates of expected outcome change. Built on meta-learners (S, T, X, DR, R), causal forests, and an ensemble-of-learners approach with uncertainty quantification. The engine is the CATE-centric sibling of Gap 21's attribution engine: attribution looks backward and asks *did this intervention work*; CATE looks forward and asks *will this intervention work here*.
2. **Intervention Recommender** — given CATE estimates across candidate interventions, combined with clinical constraints (contraindications, care plan, patient preferences), rank interventions, produce a single or small-N recommendation with transparent basis, and surface the counterfactual comparison ("versus no intervention, versus standard of care").
3. **Safe Exploration Layer** — structured contextual-bandit-style learning for novel recommendations, with conservative Q-learning guards on action selection and Offline-Guarded Safe RL (OGSRL-style) constraints on state trajectories. Explicit rule-based safety gates over the algorithmic output.
4. **Digital Twin Policy Evaluator** — a patient digital twin, causally grounded, used exclusively for **pre-deployment** counterfactual policy evaluation. The digital twin is not customer-facing and is not used for live recommendations; it is the simulation environment against which candidate recommendation policies are evaluated before they ever touch a real clinician's worklist.
5. **Recommendation Explanation Layer** — per-recommendation, clinician-facing basis that makes Gap 22 compliant with the FDA January 2026 Clinical Decision Support guidance's Criterion 4 (HCP must be able to independently review the basis). Non-negotiable from a regulatory standpoint.
6. **Policy Governance Extensions** — extends Gap 21's governance ledger to cover recommendation policies as first-class artefacts, with additional impact-assessment requirements specific to prescriptive outputs, specifically including clinician-override rate thresholds, patient-level harm bounds, and subgroup-differential recommendation fairness.

### Scope and non-scope

**In scope.** Recommendations over a pre-specified, clinically-bounded intervention set for HCF CHF post-discharge pathway and Aged Care admission-avoidance pathway. Both pilots have well-characterised intervention menus (HCF: nurse phone follow-up, GP visit, specialist referral, medication review, device monitoring enrolment, pharmacist review; Aged Care: geriatrician review, pharmacist medication review, allied health intervention, care plan revision, family conference, GP home visit). Sequential recommendations over the alert lifecycle. Integration with Gaps 17, 18, 20, 21.

**Out of scope for Sprint 3.** Open-ended generative recommendations (e.g., "recommend any intervention from the full pharmacopeia"). Direct patient-facing recommendations (the "intended user" is always the HCP, which is also a hard FDA 2026 CDS Criterion 1 requirement). Online learning that modifies production policy without governance authorisation. Critical time-sensitive recommendations (ED triage decisions, code-blue-adjacent interventions — these fail FDA 2026 Criterion 4 and become SaMD-regulated; explicitly out of scope). Drug dosing recommendations (deliberately excluded to stay clear of bright-line SaMD triggers).

### The single most important design stance

**Gap 22 is advisory and intentionally scoped to stay non-device CDS under FDA's January 2026 guidance.** Every design choice — the HCP-only intended user, the transparent basis requirement, the explicit avoidance of critical-time-sensitive decisions, the clinician-override-preserving workflow — flows from this stance. The platform could pursue SaMD classification later; the architecture is built to allow that upgrade path without rewriting. But Sprint 3 ships the non-device variant because that is the fastest route to clinical value in both pilots, and because the 2026 guidance explicitly permits single-recommendation outputs with enforcement discretion when the four criteria are met.

This is not a hedge. A non-device CDS recommender that reduces HCF 30-day readmissions by a defensible 10% is more valuable, faster to market, and easier to trust than a mid-flight-reclassified SaMD recommender that hasn't yet cleared regulatory review.

---

## 2. Why Gap 22 Matters

Three reasons, in increasing order of urgency.

**Clinical.** The platform currently tells clinicians *what to look at*. Clinicians then decide *what to do*. This is exactly how it should work in a predictive system, and it's what Gap 18's worklist was built for. But a senior clinician in HCF CHF has eight reasonable interventions they might choose from; a busy GP in the Aged Care pilot has similar optionality. Decision quality varies enormously with clinician experience, time pressure, and cognitive load. A well-calibrated CATE engine and a transparent recommender can level this up without replacing the clinician's authority. This is the single biggest uncaptured value in the platform as of end of Sprint 2.

**Economic.** Both pilots have a budget constraint on nurse-care-coordinator time, pharmacist-review capacity, and specialist-referral availability. Gap 20 identifies the high-risk patients; Gap 22 answers the next question, which is *which intervention, from our constrained capacity, maximises expected outcome improvement given this patient's features*. This is precisely what uplift modelling does in consumer retention; the stakes and methods are different here but the shape of the problem is identical. A naive "highest-risk patients get the most resources" policy is typically wrong — the highest-risk patients are often the ones whose outcomes respond least to any given intervention, while the medium-risk "persuadables" are where the interventions produce the largest effect. Identifying the persuadables is the CATE/uplift problem.

**Strategic.** Competitors in the care-management space are ramping up recommendation features. The regulatory landscape just got materially more permissive with the January 2026 FDA CDS guidance. Being second-to-market with a recommender behind a well-resourced competitor costs substantially more than being first with a conservative but defensible one. The window is now.

---

## 3. Design Principles

Ten principles that drove the plan below. These are load-bearing.

1. **Prescription requires causation, not prediction.** Every recommendation is grounded in a CATE estimate, never in a raw outcome-risk ranking. A patient with very high outcome risk but very low CATE is *not* a recommendation target; a patient with moderate risk and high CATE is. This is the opposite of the default instinct, and it's the most important epistemic shift in Gap 22.
2. **The clinician is always in the loop.** Recommendations are advisory. The recommender never auto-dispatches interventions. The workflow preserves the clinician's authority to accept, reject, or modify any recommendation. This is both a safety stance and a regulatory one (FDA 2026 CDS Criterion 2).
3. **Every recommendation carries its basis.** The transparent basis is a first-class deliverable, not an afterthought. It includes: what the recommendation is, what the CATE estimate is with uncertainty bounds, what features drove the estimate, what the alternatives were and why they ranked lower, what the overlap-positivity diagnostic says about whether the estimate is trustworthy for this patient, and when the system was last validated. This is FDA 2026 CDS Criterion 4 made operational.
4. **Safe exploration has hard limits.** The safety layer is rule-based on top of any algorithmic output. Medication contraindications, care-plan conflicts, recent-similar-intervention cool-down periods, and blacklist rules are enforced by code that the CATE engine cannot override. Offline-Guarded Safe RL principles apply: state-trajectory constraints, not just action constraints.
5. **Digital twins are for policy evaluation, never for patient recommendation.** The digital twin evaluates candidate policies in simulation before any policy change reaches production. It never produces a recommendation that reaches a clinician directly. This separation prevents a well-documented failure mode (simulator-trained policies performing well in silico, poorly in vivo) from becoming a patient-safety event.
6. **CATE is reported with uncertainty, not as a point estimate.** Following Gap 21's precedent on attribution. An intervention with CATE = 0.15 ± 0.02 and overlap-diagnostic = pass is recommendable; CATE = 0.15 ± 0.12 is not, regardless of whether 0.15 is the highest point estimate in the candidate set.
7. **Subgroup heterogeneity is a deliverable, not a diagnostic.** Per-subgroup CATE distributions are visible to clinicians and governance, not buried in an engineering dashboard. If an intervention's CATE is 0.20 for metro patients and 0.05 for regional patients, the clinician sees that and the governance ledger records it. Fairness is a property of the recommendation system, not a property of the training set alone.
8. **Recommendation policies are versioned, gated, and rollbackable — at the policy level, not just the model level.** This is a Gap 21 inheritance, extended. A policy change (e.g., moving from 1-intervention recommendation to 3-intervention recommendation; changing the CATE threshold at which recommendations trigger) is its own governance-ledger event, with shadow/canary/promotion gates.
9. **Market-config overrides carry through from Gap 21.** Every market has its own intervention set, clinical constraints, acceptance thresholds, and explanation templates, all in YAML. Base config + market overrides + pilot-specific overrides. No fork.
10. **Keep the bright line visible.** Explicitly avoid drifting into SaMD territory. If a planned feature starts to look like critical-time-sensitive recommendation, drug dosing, diagnostic-substitution, or non-HCP use, it gets carved out into a separate "SaMD-track" roadmap item that ships under a different regulatory regime. The default is non-device CDS.

---

## 4. State-of-the-Art Grounding

Brief synthesis of the research base. Full reference list at the end.

### 4.1 CATE estimation (Künzel, Athey, Wager, Nie, Kennedy)

The last six years have established a stable, well-benchmarked methodology:
- **S-learner** — one model trained jointly on treatment and outcome. Strong when the CATE is small or uniform; sometimes under-estimates heterogeneity because the treatment indicator can get swamped. The 2026 UpliftBench benchmark (Criteo, ~14M records) showed S-learner with LightGBM winning on Qini and capture rate.
- **T-learner** — separate models for treated and untreated. Strong when response functions differ substantially. Weaker when data is imbalanced because one arm trains on less data.
- **X-learner** (Künzel 2019) — two-stage imputation of individual treatment effects, weighted by propensity. Designed for imbalanced data; very often the best choice in clinical settings.
- **DR-learner / R-learner** (Kennedy 2023, Nie & Wager 2021) — Neyman-orthogonal quasi-oracle estimators. Robust to nuisance-function misspecification. Heavier machinery but the theoretical properties are the strongest.
- **Causal forest** (Wager & Athey 2018) — honest random forests for CATE. Natural uncertainty quantification (bootstrap-based CI) which is a major operational advantage.

The empirical lesson from 2025–2026 is that **no single learner dominates**; performance depends on data balance, effect-size heterogeneity, outcome distribution, and nuisance-function complexity. Gap 22's engine runs a committee of learners and selects per cohort based on a held-out CATE-ranking benchmark (Qini coefficient, uplift-by-decile, calibration of CATE).

### 4.2 Uplift modelling

Uplift modelling and CATE estimation are the same problem formulated by different communities (Gutierrez & Gérardy 2017). The uplift view emphasises *ranking* rather than *absolute-effect estimation*, which is often what matters for a recommender under capacity constraints. Key metrics: Qini coefficient (area between cumulative gain curve and random targeting), uplift-by-decile, capture rate of the top-k predicted uplift. Gap 22 uses both absolute-CATE (for recommendations where CATE is reported numerically) and uplift-ranking (for capacity-constrained targeting).

### 4.3 Off-policy evaluation (OPE)

Every candidate recommendation policy must be evaluated *without being deployed* before it reaches production. This is OPE, and it's hard because the logging policy (the current clinical standard of care, as reflected in observed data) doesn't necessarily provide coverage for the candidate policy.

- **Doubly-robust OPE** — inverse propensity scoring + direct method, robust to one-sided misspecification.
- **Adaptive weighting** (Dimakopoulou et al.) — variance reduction for OPE under adaptive data collection.
- **DataCOPE** (Sun et al. 2024) — a data-centric OPE evaluator that answers the prior question of *whether* OPE is meaningful for a given dataset × policy combination. Crucial for Gap 22 because the platform's cohorts are heterogeneous and some target policies will simply not be evaluable on the available data. DataCOPE tells you that instead of returning a misleading estimate.

### 4.4 Dynamic treatment regimes and offline RL

The sequential aspect (multiple alerts and interventions for the same patient over a care episode) is a dynamic treatment regime (DTR). Single-timestep CATE doesn't fully handle it. The 2025 systematic review (Frommeyer et al., *Healthcare*) surveys recent work:

- **Conservative Q-learning (CQL)** — Kumar et al. NeurIPS 2020. Penalises out-of-distribution actions. Standard baseline.
- **Batch-constrained Q-learning (BCQ)** — Fujimoto et al. ICML 2019. Restricts to actions observed in the batch data.
- **OGSRL** (Yan et al. arXiv 2025) — Offline Guarded Safe RL. Extends beyond action-constraint to state-trajectory constraint, which is necessary in clinical settings where a sequence of individually-safe actions can produce an unsafe patient state.
- **PROP-RL** (Jayaraman et al. JMIR Med Inform 2025) — practical pipeline for offline RL in healthcare. State representation, policy selection, OPE. The operational reference.
- **EpiCare benchmark** (Lorber et al. NeurIPS 2024) — a sobering benchmark showing that offline RL in the low-to-moderate data regime typical of clinical settings often fails to beat standard-of-care, and that common OPE methods fail on these benchmarks. **The lesson for Gap 22 is that we ship CATE-based single-step recommendation first, and treat sequential DTR as an extension**, not the foundation.

### 4.5 Digital twins for clinical policy evaluation

The DT4H paper (Qin et al. AAAI 2026) is the operational reference: RL policy + patient digital twin as environment + treatment effect as reward. Causal digital twins (Vallée 2026, *J Translational Medicine*) formalise the integration of structural causal models, potential outcomes, and RL. The Digital Twin Counterfactual Framework (Dawid et al. arXiv 2026) gives a five-level validation architecture for claims made from digital-twin simulation.

Gap 22 uses a digital twin for one purpose only: offline policy evaluation before production deployment. The twin is validated by observable-arm backtesting (the twin's predictions for the factual-treatment arm should match observed outcomes) and sensitivity analysis bounding the unobservable-arm claims.

### 4.6 Contextual bandits for clinical interventions

For interventions with low individual stakes (e.g., nudge-level nurse phone-call timing), contextual bandits are appropriate. Greenewald et al. and Tomkins et al.'s work on action-centred contextual bandits with a "do-nothing" action translates cleanly. DML-TS-NNR (Brown et al. NeurIPS 2024) handles the heterogeneity-nonstationarity-nonlinearity triple that clinical settings always exhibit.

For higher-stakes interventions (specialist referral, medication change), Gap 22 does not use online-learning contextual bandits. It uses offline-computed CATE with an explicit authorisation-required update cycle (Gap 21 governance).

### 4.7 FDA January 2026 CDS guidance

Signed by Commissioner Makary, January 6 2026. Relaxes the prior 2022 interpretation of Criterion 3 to explicitly allow single-recommendation outputs. Preserves the four statutory criteria:

- **Criterion 1** — intended user is HCP, not patient/caregiver.
- **Criterion 2** — purpose is to support clinical judgment.
- **Criterion 3** — purpose is to provide recommendations to HCP about prevention/diagnosis/treatment (single-recommendation outputs now OK under enforcement discretion).
- **Criterion 4** — HCP can independently review the basis for the recommendation.

Critical exclusions:
- Critical time-sensitive decisions fail Criterion 4 (clinician lacks time to review basis).
- Patient-facing or caregiver-facing tools fail Criterion 1.
- Black-box outputs where the basis cannot be reviewed fail Criterion 4.

Gap 22 is architected to meet all four criteria. The explanation layer is the Criterion 4 vehicle; the HCP-only audience is Criterion 1; the advisory-not-directive workflow is Criterion 2; the explicit avoidance of critical-time-sensitive scope is how Criterion 4 stays satisfied.

### 4.8 EU AI Act Article 14 (human oversight) and Article 15 (post-market monitoring)

The EU AI Act is relevant for the Aged Care pilot's future expansion. Article 14 requires human oversight of high-risk AI systems; Gap 22's clinician-in-the-loop design satisfies this. Article 15 requires post-market performance monitoring; Gap 21's governance infrastructure satisfies this. No Gap 22-specific design changes, but explicit mapping artefacts are produced (§13.3).

---

## 5. Architecture Overview

### 5.1 Gap 22 at a glance

```
   ┌───────────────────────┐     ┌───────────────────────┐
   │  Gap 20 Predictions   │     │  Gap 21 Attribution   │
   │  P(outcome | features)│     │  CATE from historical │
   └───────────┬───────────┘     │  outcomes + feedback  │
               │                 └──────────┬────────────┘
               │ risk_score                 │ historical CATE
               │                            │ per pattern/cohort
               ▼                            ▼
   ┌───────────────────────────────────────────────────┐
   │       GAP 22 CATE ENGINE (multi-learner)          │
   │   S, T, X, DR, R learners + causal forest         │
   │   Ensemble w/ uncertainty + overlap diagnostics   │
   └───────────────────────┬───────────────────────────┘
                           │ per-patient, per-intervention
                           │ CATE estimates with CI
                           ▼
   ┌───────────────────────────────────────────────────┐
   │       INTERVENTION RECOMMENDER                    │
   │   ┌─────────────────┐  ┌──────────────────┐      │
   │   │ Constraint      │  │ Capacity         │      │
   │   │ filter (contra- │→ │ optimiser (uplift│      │
   │   │ indications,    │  │  ranking under   │      │
   │   │ care plan, etc.)│  │  resource limits)│      │
   │   └─────────────────┘  └─────────┬────────┘      │
   │                                  │                │
   │                                  ▼                │
   │                      ┌──────────────────┐         │
   │                      │ Safe Exploration │         │
   │                      │ Layer (CQL+OGSRL │         │
   │                      │ guards, safety   │         │
   │                      │ rules)           │         │
   │                      └────────┬─────────┘         │
   └───────────────────────────────┼───────────────────┘
                                   │ ranked rec(s) +
                                   │ basis artefacts
                                   ▼
   ┌───────────────────────────────────────────────────┐
   │       EXPLANATION LAYER                           │
   │   CATE with CI, feature contributions, overlap    │
   │   status, counterfactual alternatives,            │
   │   cohort-CATE context, validation provenance      │
   └───────────────────────┬───────────────────────────┘
                           │
                           ▼
               ┌─────────────────────────┐
               │ Clinician worklist      │
               │ (Gap 18, extended with  │
               │  rec panel + basis      │
               │  drawer)                │
               └────────┬────────────────┘
                        │ accept / modify / reject
                        │ + reason
                        ▼
               ┌─────────────────────────┐
               │ Gap 19 lifecycle +      │
               │ Gap 21 feedback         │
               │ ingestion               │
               └─────────────────────────┘

                           ┆
                    (separate, offline)
                           ┆
                           ▼
   ┌───────────────────────────────────────────────────┐
   │       DIGITAL TWIN POLICY EVALUATOR               │
   │   Validated patient digital twin, used only for   │
   │   pre-deployment policy-change evaluation. Never  │
   │   produces a live recommendation.                 │
   └───────────────────────┬───────────────────────────┘
                           │ policy evaluation
                           │ results
                           ▼
               ┌─────────────────────────┐
               │ Policy Governance       │
               │ (Gap 21 ledger extended │
               │  with recommender       │
               │  policy entries)        │
               └─────────────────────────┘
```

### 5.2 Data flow, in plain English

A Gap 20 alert fires. The patient's Gap 19 consolidated record (features, predicted risk, contextual state) is handed to the Gap 22 CATE engine. The engine estimates, for each candidate intervention in the market's YAML-configured intervention set, the conditional average treatment effect with an uncertainty interval and an overlap-positivity diagnostic indicating whether this patient is in the support of the training data.

The intervention recommender takes the CATE estimates, applies the constraint filter (active contraindications, current care plan, intervention-cool-down periods), applies the capacity optimiser (if capacity is constrained that day, prefer the intervention with highest expected outcome improvement *per unit resource*), and runs the safety layer (state-trajectory constraints, OGSRL-style). The output is a ranked list — typically one primary recommendation plus one or two alternatives.

The explanation layer wraps the recommendation with its basis: the CATE estimate, uncertainty, top feature contributions, why alternatives ranked lower, overlap status, and provenance pointer to the current validated model version. This package lands on the clinician's Gap 18 worklist as a structured recommendation panel.

The clinician accepts (proceeds with the recommendation), modifies (proceeds with a variation), or rejects (overrides with a reason). The action flows back through Gap 19's T2/T3 lifecycle and Gap 21's feedback ingestion, and eventually Gap 21 computes post-hoc attribution on the actual outcome at T4.

Separately, and never in the live recommendation path, the digital twin policy evaluator runs in an offline environment. When a policy change is proposed (new learner, new threshold, new intervention added to the set), the candidate policy is evaluated in the digital twin with the DTCF-style validation architecture. Results feed into Gap 21's governance ledger as the "Modification Protocol" and "Impact Assessment" components.

### 5.3 Where Gap 22 sits in the service topology

Two new services and extensions to four existing ones.

**New services**
- `services/recommendation/` — the CATE engine, recommender, and safe exploration layer.
- `services/digital_twin/` — the patient digital twin and policy evaluator.

**Existing services extended**
- `services/monitoring/` (Gap 18/19/21) — add CATE-calibration monitoring, recommendation-acceptance monitoring, subgroup recommendation fairness monitoring.
- `services/events/` (Gap 19) — add recommendation events (presented, accepted, modified, rejected) as T1.5 events in the lifecycle.
- `services/learning/` (Gap 21) — add policy-training pipeline distinct from the Gap 21 prediction-retraining pipeline.
- `services/governance/` (Gap 21) — add policy-entry type to the PCCP triplet ledger, with recommender-specific impact-assessment fields.

Everything below is specified against these six surfaces.

---

## 6. The Six Core Engines

Each engine is a folded service with a single public contract, a thin YAML market config, and a testable core. Identical pattern to Gaps 14–21.

### 6.1 CATE Engine (`services/recommendation/cate/`)

**Public contract.** Given a patient's consolidated pre-alert record (from Gap 19) and a candidate intervention set (from market YAML), return per-intervention CATE estimates with uncertainty intervals, overlap-positivity diagnostics, and feature-contribution attributions.

**Internal structure.**

- **Learner committee.** S-learner, T-learner, X-learner, DR-learner, R-learner, and a causal forest, all trained on the Gap 21 outcome-labelled cohort. Each learner is instantiated with LightGBM (or per-market-config GBM base), except the causal forest which uses the `grf` reference implementation. The committee's output for each patient×intervention is the set of six CATE estimates; the "consensus" estimate is the median; uncertainty is computed from both within-learner CI (bootstrap for causal forest; residual-based for DR/R; cross-validation for S/T/X) and between-learner dispersion.
- **Per-cohort learner selection.** Not every learner is best for every cohort. On a pre-registered held-out CATE-ranking benchmark (Qini coefficient from a pseudo-outcome validation), select the top-performing learner per cohort × horizon × intervention as the primary; other learners remain in the committee as sensitivity checks. This selection is a governance-ledger event.
- **Overlap-positivity diagnostic.** For each patient × intervention, compute the propensity (probability of receiving this intervention given features) and check that it lies in the permitted overlap range (typical default: 0.05–0.95). Patients outside this range get a `CATE_INCONCLUSIVE_NO_OVERLAP` flag instead of a point estimate. This is a hard guard and cannot be disabled.
- **Feature-contribution attribution.** SHAP values computed on the primary learner for the top feature contributions. For causal-forest primary, use causal-forest variable importance. The output is the top-5 features driving the CATE estimate for this patient × intervention, signed (positive contribution means feature pushes CATE up).
- **CATE calibration monitoring integration.** Post-hoc (at T4), observed outcomes feed back to Gap 21's attribution engine, which computes actual causal effect. Gap 22 consumes this to calibrate the CATE estimates — a CATE prediction of +0.15 should, on average, be followed by observed outcome improvements averaging +0.15 for patients with similar features. Miscalibration fires a governance review.

**What it does NOT do.** It does not produce a recommendation. It produces an input to the recommender. The separation matters for testability and for regulatory clarity (the CATE engine is pre-recommendation scoring; the recommender is the decision surface).

### 6.2 Intervention Recommender (`services/recommendation/recommender/`)

**Public contract.** Given CATE estimates, produce a ranked intervention list with structured rationale.

**Internal structure.**

- **Constraint filter.** Market-configured rules that disqualify interventions based on contraindications (from the patient's active medication list, allergies, comorbidities), care plan conflicts (if the patient just had a specialist visit, down-rank "specialist referral" for a cool-down period), intervention-cool-down periods (don't recommend a nurse follow-up within 48 hours of the last one), and patient-preference flags (if the patient has opted out of phone contact, remove phone-based interventions).
- **Capacity optimiser.** If daily capacity for a given intervention type is constrained (pharmacist-review slots, geriatrician appointments), rank by *expected outcome improvement per unit resource* rather than raw CATE. This is the uplift-under-capacity problem; the reference solution is a fractional-knapsack-style greedy allocation with per-patient CATE as the value and per-intervention resource weight as the cost. Capacity is a market YAML config, updated daily by operator input or integration with scheduling.
- **Ranking.** After constraints and capacity, rank interventions by CATE point estimate, breaking ties by lower-bound of the uncertainty interval (prefer an intervention with CATE = 0.15 ± 0.03 over one with 0.16 ± 0.10).
- **Recommendation cardinality.** Per market config, the recommender returns N = 1 primary or N = 3 ranked (primary, secondary, tertiary). HCF CHF defaults to N = 1 with two alternatives visible on demand; Aged Care defaults to N = 3 because multi-modal care is the norm there.
- **Structured rationale.** The output includes: the recommendation itself, its CATE with CI, why each alternative ranked lower (with their CATE and CI), the constraint-filter outcomes (which candidate interventions were filtered and why), the capacity-optimiser state at time of recommendation, and a "confidence" label derived from CATE CI width and overlap status.

**What it does NOT do.** It does not execute the recommendation. It does not dispatch the intervention. It produces a recommendation package that the clinician, on the Gap 18 worklist, sees and decides on.

### 6.3 Safe Exploration Layer (`services/recommendation/safety/`)

**Public contract.** Between the recommender and the worklist, apply a deterministic safety layer that cannot be bypassed.

**Internal structure.**

- **Rule-based safety gates.** Hard rules that disqualify recommendations regardless of CATE magnitude:
  - Dosing-related (explicit blacklist; we do not recommend dose changes).
  - Recent-similar-action suppression (already handled in the constraint filter, but re-checked here with tighter thresholds).
  - Blacklist-intervention-for-cohort rules (e.g., "do not recommend aggressive mobilisation for RAC residents with falls in last 30 days").
  - Missing-critical-feature suppression (if key features — current weight trend, current medication list — are missing or stale, recommendations are suppressed with a "recommendation unavailable, missing data" status rather than returned as-if-normal).
- **Conservative Q-learning guards (algorithmic).** For any recommendation whose CATE estimate relies on features with low coverage in the training data (OOD-adjacent), the CQL penalty pushes the effective CATE toward zero. Implementation detail: the primary learner's outputs pass through a CQL-style regulariser that is trained jointly with the learner.
- **OGSRL state-trajectory guards.** Beyond individual-action safety, check that the recommendation doesn't move the patient into an OOD state trajectory. Concretely: look at the patient's recent state history and the expected state after the recommendation; if the pair is rare or unobserved in training, flag and degrade the recommendation's confidence label.
- **Clinician override is always permitted.** The safety layer never forces a recommendation. It can suppress, downgrade, or flag — it cannot override the clinician's right to choose any action from their legal scope.

**What it does NOT do.** It does not make the recommendation *safe* — it makes the recommender's *output* safe. Clinical safety ultimately rests with the clinician. The safety layer is belt-and-braces defence against algorithmic edge cases, not a substitute for clinical judgment.

### 6.4 Digital Twin Policy Evaluator (`services/digital_twin/`)

**Public contract.** Given a candidate recommendation policy and a validated digital twin, run the policy in simulation, produce counterfactual outcomes, and produce a policy-evaluation report with DTCF-style validation architecture.

**Internal structure.**

- **Patient digital twin.** A generative model of patient state-trajectory dynamics. Inputs: patient features at time t, intervention at time t. Output: patient state at time t+1, with uncertainty. Trained on Gap 19 consolidated records across the full pilot cohort. Implementation: bounded-residual update rule with feature-specific dynamics models (vital-sign smoothing, medication pharmacokinetic approximations for relevant drug classes, event probabilities for transitions). The "bounded-residual" design is critical: the twin's state update is anchored to the observed patient's state plus a learned residual that is bounded in magnitude to prevent runaway simulation.
- **DTCF validation architecture (Dawid et al. 2026).** Five levels of validation:
  1. **Marginal outcome validation** — does the twin produce marginal outcome distributions that match observed? Test: backtest on held-out Gap 19 data.
  2. **Conditional outcome validation** — conditional on features, does the twin predict accurately? Test: calibration of conditional predictions.
  3. **Observable-arm validation** — for any treatment arm actually observed, does the twin produce outcomes consistent with observation?
  4. **Counterfactual bounding** — for the unobserved counterfactual arm, produce Fréchet-Hoeffding bounds and sensitivity intervals.
  5. **Structural validation** — does the twin respect known clinical causal structure (e.g., stopping a diuretic does not reduce fluid overload)?
- **Policy evaluator.** Given a candidate policy, simulate N patients through the twin under the candidate policy vs the current production policy, estimate the difference in average outcome, and report with the full DTCF uncertainty decomposition.
- **Fail-safe on validation failure.** If any of the five validation levels fails (by pre-specified thresholds), the digital twin refuses to produce policy evaluation results for the affected cohort. It returns `TWIN_INVALID_FOR_COHORT` instead of a suspect estimate.

**What it does NOT do.** It does not generate patient-specific recommendations. It does not run in real-time as part of the production recommendation pipeline. It is exclusively an offline policy-evaluation tool that feeds Gap 21's governance ledger.

### 6.5 Explanation Layer (`services/recommendation/explanation/`)

**Public contract.** Given a recommendation package from the recommender, produce a clinician-facing basis artefact that satisfies FDA 2026 CDS Criterion 4.

**Internal structure.**

- **Basis template.** Market-YAML-configured, populated per recommendation. Content:
  - What: the recommendation and CATE with CI.
  - Why (short): the top 3 feature contributions driving the CATE, phrased in clinician language.
  - Why (long): a drawer/expander with the full SHAP-style feature breakdown, the alternatives considered and their CATE, the constraint and capacity context, and the overlap status.
  - Trust: confidence label (high / medium / low based on CI width + overlap), the model version, the date of last validation, the subgroup-specific CATE context ("for patients similar to this one, the CATE ranges from X to Y").
  - Limits: what this recommendation *is not* — "this is not a prescription", "this is not a substitute for clinical judgment", "the basis can be reviewed in full below". These are templated but deliberately visible.
- **Clinician language mapping.** Technical features map to clinician-language phrasings via a lookup table maintained jointly with clinical leads. "elevated_nt_pro_bnp_trend" maps to "rising NT-proBNP over last 7 days"; "polypharmacy_score_band_3" maps to "polypharmacy burden (9+ active medications)". This lookup is versioned in governance.
- **Counterfactual alternatives.** "If you prefer intervention B instead, here's what the CATE looks like for that." Supports the clinician's right to choose a different action with full information.
- **Validation-provenance footer.** Every basis artefact carries: the model version, the last validation date, the cohort on which it was validated, and a link to the governance ledger entry that authorised the current model version. This is the audit trail that regulators will ask for.

**What it does NOT do.** It does not attempt to be "interpretable AI" in a deep sense — no natural-language generative explanations (which risk hallucination), no visualisation beyond the basis template. It is a structured, templated, reliable basis, not a dazzling explanation.

### 6.6 Policy Governance Extensions (`services/governance/policy/`)

**Public contract.** Extend Gap 21's PCCP-triplet governance ledger to cover recommendation policies as first-class entries, with recommender-specific impact-assessment fields.

**Internal structure.**

- **Policy as artefact.** A recommendation policy is the tuple (CATE-learner-committee version, primary learner selection rule, constraint-filter rules version, capacity-optimiser rules version, safety-layer rules version, explanation-template version, recommendation cardinality, CATE-threshold-for-recommendation). Each element is independently versioned; a "policy" is a specific combination of element versions.
- **Additional impact-assessment fields (beyond Gap 21).** Recommendation-specific:
  - Expected change in recommendation volume (total and per intervention type).
  - Expected change in clinician-acceptance rate.
  - Per-subgroup recommendation-frequency shift.
  - Worst-case patient-level CATE-change bound (if the new policy changes a recommendation's CATE for an individual patient, how much could it change?).
  - Capacity-utilisation impact.
- **Rollback playbook, extended.** Policy rollback is atomic (single operator command) like Gap 21, plus a recommendation-retraction communication plan if any recommendations already landed on clinician worklists under the to-be-rolled-back policy in the active care window. Retraction does not delete the recommendation from the clinician's history but adds a "policy superseded" flag visible in the audit trail.
- **Shadow/canary/promotion gates, extended.** Same three stages as Gap 21. Canary runs for longer (default 8 weeks for HCF CHF, 12 weeks for Aged Care) because recommendations have longer downstream effect chains than raw predictions. The acceptance criteria are richer: not just AUROC-equivalent metrics but also clinician-acceptance rate, recommendation-fairness across subgroups, and attribution-weighted outcome improvement relative to the current policy.

**What it does NOT do.** It does not introduce a new governance framework. It extends Gap 21's PCCP-triplet ledger with recommender-specific fields. Same RACI, same cryptographic signing, same append-only semantics.

---

## 7. Phase Breakdown

Four phases, fourteen steps. Identical pattern to Gap 20/21.

### Phase 1 — CATE Foundation (4 steps)

**Step 1.1 — Intervention taxonomy and eligibility codification.**
Codify the HCF CHF and Aged Care intervention sets as structured YAML with: intervention ID, clinician-language name, technical categorisation, eligibility criteria (structured), contraindication list (structured), cool-down period, typical resource cost (for the capacity optimiser), and the feature signature (which patient features are required to score this intervention's CATE). This is foundational; every downstream step depends on the taxonomy being right.

*Files*: `services/recommendation/taxonomy/builder.py`, `services/recommendation/taxonomy/validation.py`, `contracts/interventions/schema_v1.yaml`, `configs/interventions/hcf-chf.yaml`, `configs/interventions/aged-care-au.yaml`, `configs/interventions/base.yaml`, `tests/recommendation/test_taxonomy.py`.

**Step 1.2 — CATE learner committee.**
Implement the six-learner committee (S, T, X, DR, R, causal forest). Each learner is a class with a consistent interface (`fit`, `predict_cate`, `predict_interval`, `feature_contribution`). LightGBM base for meta-learners; `grf`-style causal forest. Training pipeline ingests Gap 21's labelled cohort. Cross-validation folds configured per cohort size. Hyperparameter selection via pre-specified grids, logged in governance.

*Files*: `services/recommendation/cate/learners/s_learner.py`, `services/recommendation/cate/learners/t_learner.py`, `services/recommendation/cate/learners/x_learner.py`, `services/recommendation/cate/learners/dr_learner.py`, `services/recommendation/cate/learners/r_learner.py`, `services/recommendation/cate/learners/causal_forest.py`, `services/recommendation/cate/committee.py`, `services/recommendation/cate/training.py`, `configs/cate/base.yaml`, `configs/cate/hcf-chf.yaml`, `configs/cate/aged-care-au.yaml`, `tests/cate/test_s_learner.py`, `tests/cate/test_t_learner.py`, `tests/cate/test_x_learner.py`, `tests/cate/test_dr_learner.py`, `tests/cate/test_r_learner.py`, `tests/cate/test_causal_forest.py`, `tests/cate/test_committee.py`.

**Step 1.3 — Overlap-positivity diagnostics and per-cohort learner selection.**
Propensity estimation (gradient-boosted trees, calibrated with isotonic regression — mirrors Gap 21 Step 1.3 for consistency). Overlap range configuration per market; default (0.05, 0.95) with pilot-specific overrides. Per-cohort learner selection: held-out Qini coefficient, uplift-by-decile, calibration-on-pseudo-outcomes. Selected learner per cohort × horizon × intervention logged to governance.

*Files*: `services/recommendation/cate/propensity.py`, `services/recommendation/cate/overlap.py`, `services/recommendation/cate/selection.py`, `services/recommendation/cate/evaluation.py`, `configs/cate/selection.yaml`, `tests/cate/test_overlap.py`, `tests/cate/test_selection.py`.

**Step 1.4 — Feature contribution and CATE calibration monitoring.**
SHAP values for the primary learner (per cohort). Causal-forest variable importance for cf-primary cohorts. Integration with Gap 21's attribution outputs: for each T4-closed alert with an intervention, compare the attributed effect to the CATE estimate that was produced when the alert fired. Miscalibration alarm: if the rolling mean |attributed - predicted CATE| exceeds a threshold for a specific cohort × intervention, fire a governance review. This integrates Gap 22's CATE engine with Gap 21's monitoring infrastructure cleanly.

*Files*: `services/recommendation/cate/explanation.py`, `services/recommendation/cate/calibration_monitor.py`, `services/monitoring/cate_calibration.py`, `tests/cate/test_explanation.py`, `tests/cate/test_calibration_monitor.py`.

**Phase 1 acceptance criteria.**
- End-to-end CATE estimate for a single HCF CHF patient × intervention in under 500 ms (p95), given the patient's consolidated record.
- Overlap-positivity diagnostics automatically flag inconclusive cases; synthetic-data tests confirm the engine refuses to estimate when overlap fails.
- Per-cohort learner selection produces different primary learners for at least two cohorts (validates the selection machinery).
- CATE calibration monitoring fires a synthetic miscalibration alarm within expected latency.

### Phase 2 — Recommendation and Explanation (4 steps)

**Step 2.1 — Constraint filter and capacity optimiser.**
The constraint filter: YAML-configured rules that disqualify interventions based on patient state. Rules support: contraindication matching against active medication list, care-plan-conflict detection, cool-down period enforcement, patient-preference flag respect. Rules are deterministic, testable, and order-independent within each category. The capacity optimiser: fractional-knapsack-style greedy allocation given per-intervention daily capacity and per-patient CATE. Capacity input is a daily YAML/operator-maintained file or integration with an external scheduling system.

*Files*: `services/recommendation/recommender/constraints.py`, `services/recommendation/recommender/capacity.py`, `services/recommendation/recommender/rules_engine.py`, `configs/constraints/hcf-chf.yaml`, `configs/constraints/aged-care-au.yaml`, `configs/capacity/hcf-chf.yaml`, `configs/capacity/aged-care-au.yaml`, `tests/recommender/test_constraints.py`, `tests/recommender/test_capacity.py`.

**Step 2.2 — Ranking and recommendation cardinality.**
Rank candidate interventions by CATE point estimate with lower-bound-of-CI tie-breaking. Per-market recommendation cardinality configuration. Structured rationale output: recommendation, CATE with CI, alternatives with CATE, filtered candidates with reasons, capacity context, confidence label. This is the final "Recommender" public contract output.

*Files*: `services/recommendation/recommender/ranking.py`, `services/recommendation/recommender/rationale.py`, `services/recommendation/recommender/public_api.py`, `contracts/recommendation/schema_v1.yaml`, `tests/recommender/test_ranking.py`, `tests/recommender/test_rationale.py`.

**Step 2.3 — Safe exploration layer.**
Rule-based safety gates (hard guards, cannot be bypassed). CQL-style regulariser integration into the primary learner output. OGSRL-style state-trajectory checks (uses Gap 21's per-patient baseline trajectory to detect OOD-adjacent transitions). The three layers compose: constraints → capacity → safety. Each layer's pass/fail decision is logged with a structured reason code for audit.

*Files*: `services/recommendation/safety/rules.py`, `services/recommendation/safety/cql_guard.py`, `services/recommendation/safety/ogsrl_guard.py`, `services/recommendation/safety/pipeline.py`, `configs/safety/base.yaml`, `configs/safety/hcf-chf.yaml`, `configs/safety/aged-care-au.yaml`, `tests/safety/test_rules.py`, `tests/safety/test_cql_guard.py`, `tests/safety/test_ogsrl_guard.py`, `tests/safety/test_pipeline.py`.

**Step 2.4 — Explanation layer and Gap 18 worklist integration.**
Explanation layer implementation: template population, clinician-language mapping, counterfactual alternatives rendering, validation-provenance footer. Gap 18 worklist integration: a new "Recommendation Panel" surface with primary, alternatives visible on expand, full basis drawer. UX approved by clinical leads before rollout. Recommendation events (presented, viewed, expanded, accepted, modified, rejected) flow to Gap 19's lifecycle as T1.5 events. This is where the human factors work lives; rushing it is a mistake.

*Files*: `services/recommendation/explanation/templates.py`, `services/recommendation/explanation/language_mapper.py`, `services/recommendation/explanation/provenance.py`, `services/recommendation/explanation/renderer.py`, `services/events/recommendation_events.py`, `configs/explanation/templates/hcf-chf/`, `configs/explanation/templates/aged-care-au/`, `configs/explanation/language_map.yaml`, `tests/explanation/test_templates.py`, `tests/explanation/test_language_mapper.py`, `tests/events/test_recommendation_events.py`, `docs/design/worklist_rec_panel.md`.

**Phase 2 acceptance criteria.**
- End-to-end recommendation (CATE → constraints → capacity → safety → explanation) in under 2 seconds (p95) for a single patient.
- Synthetic contraindication cases are correctly filtered; synthetic OOD states are correctly flagged by OGSRL.
- Clinical-lead sign-off on the Recommendation Panel UX and explanation templates before any live deployment.
- Property-based tests confirm the safety layer cannot be bypassed by adversarial CATE inputs.

### Phase 3 — Safe Exploration and Digital Twin (3 steps)

**Step 3.1 — Patient digital twin (bounded-residual model).**
State-trajectory model per cohort. Bounded-residual update rule: `state_{t+1} = state_t + f_θ(state_t, intervention, Δt)` with per-feature residual magnitude bounds. Feature-specific dynamics: vital-sign smoothing for HR/BP, bounded medication-effect approximations for diuretics and beta-blockers (not dose recommendations — just effect approximations on state), event-probability head for transitions. Trained on Gap 19 consolidated records. Designed to be interpretable in each component, even if the full model is not.

*Files*: `services/digital_twin/model/state_model.py`, `services/digital_twin/model/residual_bounds.py`, `services/digital_twin/model/dynamics/vital_signs.py`, `services/digital_twin/model/dynamics/medication_effects.py`, `services/digital_twin/model/dynamics/events.py`, `services/digital_twin/training.py`, `configs/digital_twin/base.yaml`, `configs/digital_twin/hcf-chf.yaml`, `configs/digital_twin/aged-care-au.yaml`, `tests/digital_twin/test_state_model.py`, `tests/digital_twin/test_residual_bounds.py`, `tests/digital_twin/test_vital_signs_dynamics.py`.

**Step 3.2 — DTCF validation architecture.**
Five-level validation: marginal, conditional, observable-arm, counterfactual bounding, structural. Each level as a test suite that runs on the trained digital twin before it is authorised for policy evaluation. Counterfactual bounding uses Fréchet-Hoeffding bounds (hard theoretical bounds on the counterfactual given observed marginals). Structural validation runs canonical clinical causal checks (e.g., diuretic start → fluid decrease → weight decrease; stopping diuretic does *not* produce fluid decrease). The `TWIN_INVALID_FOR_COHORT` status is produced whenever any level fails beyond configured thresholds.

*Files*: `services/digital_twin/validation/marginal.py`, `services/digital_twin/validation/conditional.py`, `services/digital_twin/validation/observable_arm.py`, `services/digital_twin/validation/counterfactual_bounds.py`, `services/digital_twin/validation/structural.py`, `services/digital_twin/validation/pipeline.py`, `configs/digital_twin/validation.yaml`, `tests/digital_twin/test_validation_levels.py`, `tests/digital_twin/test_validation_pipeline.py`, `docs/digital_twin/dtcf_mapping.md`.

**Step 3.3 — Policy evaluator and OPE integration.**
Policy evaluator: given a candidate policy, run it in simulation against the twin for N patients, produce outcome-difference estimates with DTCF-style uncertainty decomposition. OPE integration: compute doubly-robust OPE estimates against the observational data as a cross-check on the twin-based evaluation. DataCOPE-style pre-check: confirm that OPE is meaningful for the candidate policy given the observational data coverage; if not, flag and rely on twin-based evaluation only, with clearer uncertainty bounds. All results go to Gap 21's governance ledger as part of the Modification Protocol + Impact Assessment for the candidate policy.

*Files*: `services/digital_twin/evaluator/simulator.py`, `services/digital_twin/evaluator/ope.py`, `services/digital_twin/evaluator/datacope.py`, `services/digital_twin/evaluator/report_generator.py`, `configs/policy_evaluation/base.yaml`, `tests/digital_twin/test_simulator.py`, `tests/digital_twin/test_ope.py`, `tests/digital_twin/test_datacope.py`, `docs/runbooks/policy_evaluation.md`.

**Phase 3 acceptance criteria.**
- Digital twin passes all five DTCF validation levels on the HCF CHF cohort; flagged cohorts for any failure are explicitly enumerated.
- Twin-based policy evaluation agrees with OPE-based evaluation within configured tolerance for synthetic policies where ground-truth effect is known.
- A synthetic "clearly-worse" candidate policy is correctly identified as worse by the evaluator; a synthetic "clearly-better" policy is correctly identified as better. False positives and false negatives are quantified.
- DataCOPE correctly flags at least one policy as "OPE-inconclusive" on a synthetic cohort-coverage edge case, demonstrating the pre-check is functional.

### Phase 4 — Governance and Deployment (3 steps)

**Step 4.1 — Policy governance extension.**
Extend Gap 21's PCCP-triplet ledger with recommender-specific impact-assessment fields. Policy-as-artefact schema. Rollback playbook extension (policy-level rollback + recommendation-retraction communication). RACI unchanged from Gap 21 (engineer + clinical lead + governance officer). Ledger query API extension for policy-change queries.

*Files*: `services/governance/policy/policy_entry.py`, `services/governance/policy/impact_fields.py`, `services/governance/policy/rollback.py`, `services/governance/policy/query.py`, `contracts/governance/policy_pccp_schema.yaml`, `tests/governance/test_policy_entry.py`, `tests/governance/test_policy_rollback.py`, `docs/runbooks/policy_rollback.md`.

**Step 4.2 — Shadow → canary → full promotion for policies.**
Three-stage deployment identical in structure to Gap 21 but with recommender-specific acceptance criteria. Shadow: candidate policy runs against logged alerts but produces no clinician surface. Canary: candidate serves 10% of alerts for 8-12 weeks (market-dependent). Full: candidate becomes production; previous policy held as hot-standby for rollback. Acceptance criteria: aggregate attribution-weighted outcome improvement; per-subgroup fairness; clinician-acceptance rate ≥ production; CATE-calibration monitoring all-green; recommendation-volume change within configured bounds.

*Files*: `services/learning/policy_retrain/shadow.py`, `services/learning/policy_retrain/canary.py`, `services/learning/policy_retrain/promotion.py`, `services/learning/policy_retrain/acceptance.py`, `configs/policy_retrain/hcf-chf.yaml`, `configs/policy_retrain/aged-care-au.yaml`, `tests/policy_retrain/test_shadow.py`, `tests/policy_retrain/test_canary.py`, `tests/policy_retrain/test_promotion.py`, `tests/policy_retrain/test_acceptance.py`, `docs/runbooks/policy_deployment.md`.

**Step 4.3 — FDA 2026 CDS compliance artefact pack.**
Structured pack demonstrating Gap 22's satisfaction of the four FDA 2026 CDS criteria. Per Criterion 1 (HCP intended user): user-access controls, labelling, training materials. Per Criterion 2 (supports clinical judgment): workflow design, advisory-not-directive language, clinician-acceptance rate data. Per Criterion 3 (about prevention/diagnosis/treatment): intervention-set scope, pathway design. Per Criterion 4 (independent basis review): explanation layer, language mapping, validation provenance, feature contributions, alternatives. Pack generated on demand from the governance ledger. EU AI Act Article 14/15 mapping as a parallel artefact.

*Files*: `services/governance/compliance/fda_2026_cds.py`, `services/governance/compliance/eu_ai_act.py`, `services/governance/compliance/artefact_pack.py`, `configs/compliance/fda_2026_cds_template/`, `configs/compliance/eu_ai_act_template/`, `tests/governance/test_fda_2026_pack.py`, `tests/governance/test_eu_ai_act_pack.py`, `docs/governance/fda_2026_mapping.md`, `docs/governance/eu_ai_act_mapping.md`.

**Phase 4 acceptance criteria.**
- Policy change end-to-end: proposed → shadow → canary → full promotion, for a synthetic policy, completes without unplanned incidents.
- Rollback drill at the policy level (vs. the model level from Gap 21) completed monthly.
- FDA 2026 CDS compliance pack generated and reviewed by external regulatory counsel.
- At least one policy change lands in production with full governance ledger record, ready to cite in the Q-end stakeholder report.

---

## 8. Integration with Gaps 17, 18, 19, 20, 21

Gap 22 depends on five prior gaps. Clean interfaces matter.

### 8.1 Gap 20 Predictive Risk Layer interface

Gap 22 consumes: risk score, feature vector, model version (prediction-snapshot contract).

Gap 22 adds: nothing to Gap 20. The CATE engine is a separate model family from the risk-prediction model family; it is trained independently on the Gap 21 outcome-labelled cohort.

### 8.2 Gap 21 Closed-Loop Outcome Learning interface

Gap 22 consumes: Gap 21's attribution outputs (to calibrate CATE estimates post-hoc); Gap 21's feedback ingestion (override reasons from T2/T3 feed into both the CATE learner's retraining set and the recommender's constraint-filter refinement); Gap 21's governance ledger (Gap 22 policies land as new entry types in the same ledger).

Gap 22 adds: `recommendation_calibration` metric to the Gap 21 monitoring dashboard; recommender-specific PCCP-triplet impact-assessment fields to the governance ledger.

### 8.3 Gap 19 event lifecycle interface

Gap 22 consumes: consolidated pre-alert record (features, contextual state, patient baseline trajectory).

Gap 22 adds: T1.5 "recommendation lifecycle" events — recommendation_presented, recommendation_viewed, recommendation_expanded, recommendation_accepted, recommendation_modified, recommendation_rejected. These are additive schema extensions, not breaking changes.

### 8.4 Gap 18 Clinician Worklist interface

Gap 22 consumes: clinician identity, session state, current worklist view.

Gap 22 adds: the Recommendation Panel surface (primary, alternatives, expand-for-basis), and the action buttons that emit recommendation lifecycle events. The panel is a new UI component; the rest of Gap 18 is unchanged.

### 8.5 Gap 17 Care Transition Bridge interface

Gap 22 consumes: currently-active care-plan state (for constraint filter), active intervention cool-down state.

Gap 22 adds: nothing to Gap 17. The constraint filter reads Gap 17's state; it does not write to it.

### 8.6 Dependency direction and release sequencing

Gap 22 is dependent on Gap 21 shipping (feedback ingestion, governance ledger, attribution monitoring). Phase 1 of Gap 22 can begin once Gap 21 Phase 1 (outcome ingestion and consolidation) is stable — the CATE learner committee needs labelled outcomes.

Phase 2 can begin in parallel with Phase 1 because it doesn't depend on CATE outputs; the constraint filter, capacity optimiser, and UX design can all proceed.

Phase 3 can begin once Phase 1 is done (the digital twin trains on the same consolidated records; its DTCF validation uses the same outcome labels).

Phase 4 is blocked on Phase 3 completion for the compliance artefacts; policy governance extension (Step 4.1) can start earlier.

---

## 9. Pilot-Specific Considerations

### 9.1 HCF CHF

**Intervention set.** From market YAML: nurse phone follow-up (within 48 hours of discharge); GP visit (within 7 days); specialist referral (cardiologist); medication review by pharmacist; device monitoring enrolment; nutritionist consultation. Roughly 6–8 candidate interventions per alert.

**Primary CATE horizon.** 30 days (matches Gap 21 primary).

**Learner selection expectation (pre-data).** HCF CHF has moderate imbalance (the "no intervention" arm is historically smaller than "some intervention"), which suggests X-learner may dominate. To be confirmed empirically in Step 1.3.

**Capacity constraints.** Nurse call capacity is configured daily. Specialist-referral capacity is market-configured (depends on HCF's network). Pharmacist review slots are the tightest resource; the capacity optimiser will frequently push alternatives when pharmacist capacity is saturated.

**Recommendation cardinality.** N = 1 primary + 2 visible alternatives. Clinical-lead rationale: a single clear recommendation reduces cognitive load; alternatives are one click away.

**Specific risks.**
- **Seasonal volume.** Winter CHF spikes mean recommendation volume will too. The capacity optimiser must handle load spikes gracefully — when capacity is oversubscribed, recommendations are explicitly "no intervention recommended due to capacity" rather than falling back to a naive default.
- **Device monitoring enrolment recommendation.** A device-based intervention has different acceptance dynamics (patient consent, device availability, onboarding time). The CATE horizon for device-monitoring outcomes is longer than 30 days; Gap 22 should produce device-monitoring recommendations with a longer attribution horizon flagged in the explanation panel.

### 9.2 Aged Care

**Intervention set.** Geriatrician review, pharmacist medication review (highly relevant given polypharmacy), allied health intervention (physio/OT), care plan revision, family conference, GP home visit. Roughly 5–7 candidate interventions.

**Primary CATE horizon.** 90 days (matches Gap 21 primary for Aged Care).

**Learner selection expectation.** Aged Care has higher heterogeneity in response functions (a pharmacist review does different things for a dementia-patient vs. a cognitively-intact resident). T-learner or R-learner may dominate here. Empirical in Step 1.3.

**Capacity constraints.** Geriatrician capacity is extremely tight; pharmacist review slots are also tight. The capacity optimiser will do most of its work here.

**Recommendation cardinality.** N = 3 (primary, secondary, tertiary). Clinical-lead rationale: multi-modal care is the Aged Care norm; a single recommendation understates the clinical reality.

**Specific risks.**
- **Outcome ascertainment lag.** 90-day outcomes close slowly; CATE calibration monitoring will have sparse signal in the first 6 months. Solution: supplement with 30-day interim outcome proxies (structured in the market YAML) during early pilot operation, explicitly flagged as provisional.
- **CALD and Indigenous subgroups.** Consent constraints (per Gap 21 §9.2) mean CATE estimates for these subgroups may be underpowered. The overlap-positivity diagnostic will correctly flag inconclusive cases, but the consequence is that the recommender will more often return "no strong recommendation — consider standard of care" for these subgroups. This is the *correct* behaviour (never recommend on insufficient evidence), but it creates an apparent equity issue (minority-subgroup patients get fewer recommendations). The governance response: explicitly track subgroup recommendation rates, and when they drop below configured thresholds, route those patients to a human-reviewer queue (integrating with Gap 18) rather than producing a degraded recommendation.
- **Family conference as an intervention.** This is not a clinical action in the narrow sense; it's a care-planning discussion. Its CATE is different in structure — it's about discovering the patient's preferences and updating the plan. Gap 22 should treat it as a first-class intervention but with a distinct attribution horizon and explanation template.

### 9.3 India pilot

Placeholder configs ship with SCOPE_TBD flags. The CATE engine refuses to produce recommendations against SCOPE_TBD configs until the pilot is scoped. Per Gap 21's pattern.

---

## 10. Validation, Safety, and Rollback

### 10.1 Pre-deployment validation

Same three-tier structure as Gap 21, with Gap 22-specific additions:

- **Unit tests.** Every public function. 90% line coverage, 100% branch coverage on safety-critical paths (safety-layer rules, explanation-layer provenance footer, governance-ledger policy-entry signing).
- **Integration tests.** Every service boundary. Particular attention to recommender → safety-layer composition and the Gap 18 worklist integration.
- **Property-based tests.**
  - CATE engine: synthetic ground-truth treatment-effect scenarios; verify that the committee recovers the true CATE within acceptable error for each learner's strength regime.
  - Recommender: synthetic constraint scenarios; verify that contraindicated interventions never make it past the constraint filter regardless of CATE magnitude.
  - Safety layer: adversarial CATE inputs (astronomically-high CATE for a contraindicated intervention); verify suppression.
  - Digital twin: synthetic scenarios where ground-truth policy value is known; verify twin-based evaluation agrees with ground truth within tolerance.
- **Adversarial tests.** Attempt to bypass the safety layer by crafting CATE estimates that look like edge cases (should be suppressed); attempt to promote a policy without governance authorisation (should fail); attempt to retract a recommendation via the policy-rollback path (should fail cleanly — policy rollback doesn't erase history).

### 10.2 Shadow and canary

Every policy change goes through shadow (Phase 4 Step 4.2) and canary. Canary is longer than Gap 21's retraining canary because recommendation effects propagate further through the care pathway and take longer to observe. Acceptance criteria are richer: attribution-weighted outcome improvement; per-subgroup fairness; clinician-acceptance rate ≥ production; CATE-calibration all-green; recommendation-volume change within configured bounds.

### 10.3 Rollback

Policy-level rollback. Single-command atomic switch. Adds a retraction-communication component for recommendations already landed on live worklists under the to-be-rolled-back policy. Monthly rehearsal, same as Gap 21.

### 10.4 Safety-critical guardrails (cannot be disabled)

- **No recommendation without overlap.** If overlap-positivity fails, CATE is `INCONCLUSIVE` and the recommender does not surface a recommendation for that intervention.
- **No recommendation bypassing the safety layer.** The safety layer is always invoked; a CATE > 0 is not sufficient to recommend. Configuration cannot skip the safety layer.
- **No policy change without authorisation.** Governance ledger authorisation is required; no "auto-retrain" mode for policies.
- **No subgroup-blind recommendation.** Every recommendation carries its per-subgroup CATE context in the explanation. A subgroup-blind mode cannot be enabled.
- **No critical-time-sensitive extension.** The market config for a pilot cannot add intervention types that map to critical-time-sensitive clinical decisions. The config validator rejects such additions with a regulatory-risk error. Expanding into critical-time-sensitive scope requires a formal scope change and SaMD-track governance.
- **No patient-facing access.** User access controls enforce HCP-only audience. Patient-facing portals cannot consume the recommendation API.

---

## 11. File Map

Roughly 70 implementation files + ~60 test files + configs + docs, consistent with Gap 20/21 scale.

**Taxonomy and eligibility**
1. `services/recommendation/taxonomy/builder.py`
2. `services/recommendation/taxonomy/validation.py`
3. `contracts/interventions/schema_v1.yaml`
4. `configs/interventions/base.yaml`
5. `configs/interventions/hcf-chf.yaml`
6. `configs/interventions/aged-care-au.yaml`
7. `configs/interventions/india-pilot.yaml` (SCOPE_TBD)

**CATE engine**
8. `services/recommendation/cate/learners/s_learner.py`
9. `services/recommendation/cate/learners/t_learner.py`
10. `services/recommendation/cate/learners/x_learner.py`
11. `services/recommendation/cate/learners/dr_learner.py`
12. `services/recommendation/cate/learners/r_learner.py`
13. `services/recommendation/cate/learners/causal_forest.py`
14. `services/recommendation/cate/committee.py`
15. `services/recommendation/cate/training.py`
16. `services/recommendation/cate/propensity.py`
17. `services/recommendation/cate/overlap.py`
18. `services/recommendation/cate/selection.py`
19. `services/recommendation/cate/evaluation.py`
20. `services/recommendation/cate/explanation.py`
21. `services/recommendation/cate/calibration_monitor.py`
22. `services/monitoring/cate_calibration.py`
23. `configs/cate/base.yaml`
24. `configs/cate/hcf-chf.yaml`
25. `configs/cate/aged-care-au.yaml`
26. `configs/cate/selection.yaml`

**Recommender**
27. `services/recommendation/recommender/constraints.py`
28. `services/recommendation/recommender/capacity.py`
29. `services/recommendation/recommender/rules_engine.py`
30. `services/recommendation/recommender/ranking.py`
31. `services/recommendation/recommender/rationale.py`
32. `services/recommendation/recommender/public_api.py`
33. `contracts/recommendation/schema_v1.yaml`
34. `configs/constraints/hcf-chf.yaml`
35. `configs/constraints/aged-care-au.yaml`
36. `configs/capacity/hcf-chf.yaml`
37. `configs/capacity/aged-care-au.yaml`

**Safety layer**
38. `services/recommendation/safety/rules.py`
39. `services/recommendation/safety/cql_guard.py`
40. `services/recommendation/safety/ogsrl_guard.py`
41. `services/recommendation/safety/pipeline.py`
42. `configs/safety/base.yaml`
43. `configs/safety/hcf-chf.yaml`
44. `configs/safety/aged-care-au.yaml`

**Explanation layer**
45. `services/recommendation/explanation/templates.py`
46. `services/recommendation/explanation/language_mapper.py`
47. `services/recommendation/explanation/provenance.py`
48. `services/recommendation/explanation/renderer.py`
49. `configs/explanation/templates/hcf-chf/`
50. `configs/explanation/templates/aged-care-au/`
51. `configs/explanation/language_map.yaml`

**Event lifecycle**
52. `services/events/recommendation_events.py`

**Digital twin**
53. `services/digital_twin/model/state_model.py`
54. `services/digital_twin/model/residual_bounds.py`
55. `services/digital_twin/model/dynamics/vital_signs.py`
56. `services/digital_twin/model/dynamics/medication_effects.py`
57. `services/digital_twin/model/dynamics/events.py`
58. `services/digital_twin/training.py`
59. `services/digital_twin/validation/marginal.py`
60. `services/digital_twin/validation/conditional.py`
61. `services/digital_twin/validation/observable_arm.py`
62. `services/digital_twin/validation/counterfactual_bounds.py`
63. `services/digital_twin/validation/structural.py`
64. `services/digital_twin/validation/pipeline.py`
65. `services/digital_twin/evaluator/simulator.py`
66. `services/digital_twin/evaluator/ope.py`
67. `services/digital_twin/evaluator/datacope.py`
68. `services/digital_twin/evaluator/report_generator.py`
69. `configs/digital_twin/base.yaml`
70. `configs/digital_twin/hcf-chf.yaml`
71. `configs/digital_twin/aged-care-au.yaml`
72. `configs/digital_twin/validation.yaml`
73. `configs/policy_evaluation/base.yaml`

**Governance extensions**
74. `services/governance/policy/policy_entry.py`
75. `services/governance/policy/impact_fields.py`
76. `services/governance/policy/rollback.py`
77. `services/governance/policy/query.py`
78. `contracts/governance/policy_pccp_schema.yaml`

**Policy retraining pipeline**
79. `services/learning/policy_retrain/shadow.py`
80. `services/learning/policy_retrain/canary.py`
81. `services/learning/policy_retrain/promotion.py`
82. `services/learning/policy_retrain/acceptance.py`
83. `configs/policy_retrain/hcf-chf.yaml`
84. `configs/policy_retrain/aged-care-au.yaml`

**Compliance packs**
85. `services/governance/compliance/fda_2026_cds.py`
86. `services/governance/compliance/eu_ai_act.py`
87. `services/governance/compliance/artefact_pack.py`
88. `configs/compliance/fda_2026_cds_template/`
89. `configs/compliance/eu_ai_act_template/`

**Docs and runbooks**
90. `docs/design/worklist_rec_panel.md`
91. `docs/runbooks/policy_evaluation.md`
92. `docs/runbooks/policy_deployment.md`
93. `docs/runbooks/policy_rollback.md`
94. `docs/digital_twin/dtcf_mapping.md`
95. `docs/governance/fda_2026_mapping.md`
96. `docs/governance/eu_ai_act_mapping.md`
97. `docs/architecture/gap22_overview.md`

Tests mirror the implementation structure (~60 test files).

---

## 12. Regulatory Compliance Detail (FDA 2026 CDS)

Gap 22 is deliberately architected to meet the four FDA 2026 CDS criteria and stay outside device regulation under enforcement discretion.

### 12.1 Criterion 1 — HCP intended user

- Access controls: recommendation API requires HCP-level authentication. Patient portals cannot reach it.
- Labelling: "This tool is intended for use by healthcare professionals as an aid to clinical decision-making. It is not intended for patient use." On every panel, every export, every artefact.
- Training materials: all user-facing documentation is written for HCPs, uses clinical terminology, and assumes clinical context.

### 12.2 Criterion 2 — Supports clinical judgment

- Workflow design: recommendation is advisory. The clinician's accept/modify/reject action is the decision, not the recommendation.
- Language: "Recommended" rather than "Prescribed"; "Consider" rather than "Do"; the basis is visible to support the clinician's evaluation, not replace it.
- Data: clinician-acceptance rate is monitored. If acceptance is near 100%, that is itself a warning sign (clinician reliance concern); if it's too low, the recommender is probably not helpful. Target range is 40–70% acceptance.

### 12.3 Criterion 3 — Prevention/diagnosis/treatment recommendations

- Intervention set scope: the pilot intervention sets are all within prevention (e.g., falls prevention), diagnosis (e.g., specialist referral for workup), or treatment (e.g., medication review) — the natural FDA CDS scope.
- Single-recommendation output is allowed under the 2026 enforcement-discretion expansion; the recommender's N=1 or N=3 configurations both qualify.

### 12.4 Criterion 4 — Independent review of basis

- The explanation layer is the Criterion 4 vehicle. For every recommendation, the HCP can see: what is recommended, why (top features), what alternatives existed, the CATE estimate with uncertainty, the validation status of the underlying model, and the basis in full detail on expand.
- Not critical-time-sensitive: the market configs reject interventions that would fall into critical-time-sensitive scope. The config validator is the enforcement point.

### 12.5 Compliance pack (automatic, on demand)

Generated from the governance ledger at any time. Contents:
- Current recommender policy version, with full lineage to genesis.
- Per-Criterion evidence (structured mapping to the four criteria).
- Per-cohort validation evidence, including digital-twin validation levels and CATE calibration history.
- Change history with PCCP triplets for each policy change.
- Clinician-acceptance rate history with subgroup breakdowns.
- User-access audit trail.
- EU AI Act Article 14/15 evidence as a parallel artefact.

---

## 13. Open Questions and Deferrals

**Deferred to Sprint 4 (Gap 23 candidates).**

1. **Dynamic treatment regime (sequential DTR) with offline RL.** The EpiCare 2024 benchmark suggests offline RL in moderate-data regimes is not yet clinically reliable. Gap 22 ships single-step CATE-based recommendations first; sequential DTR with CQL/BCQ/OGSRL is Sprint 4 candidate.
2. **Online contextual bandit for low-stakes interventions.** The mHealth-style "just-in-time adaptive intervention" design with Thompson sampling over a "do-nothing" action is technically natural for nurse-call-timing optimisation. Deferred because the regulatory posture is cleaner if the recommender is fully offline-trained in Sprint 3.
3. **Federated CATE estimation across sites.** Federated causal forests (Teo et al. 2024 extension) would let HCF and Aged Care pool CATE learning without patient-data movement. Gap 22's learner committee is architected to be federatable, but the federation infrastructure is Sprint 4 candidate.
4. **Natural-language explanation generation.** An LLM-based narrative layer on top of the structured basis would be more clinician-friendly. Deferred because FDA 2026 Criterion 4 requires a basis the HCP can review; a hallucinating narrative layer could undermine that. When LLM-judge reliability is better established, revisit.
5. **Patient-facing variants (SaMD-track).** A patient-facing version of the recommender (e.g., "suggested questions to ask your doctor") would be SaMD-regulated. Not on the Sprint 3 roadmap; explicitly out of scope.

**Open questions requiring clinical/business input before Phase 1 ships.**

6. **Intervention-set final vocabulary.** Each pilot's clinical lead must sign off on the intervention taxonomy (Step 1.1). The ~6–8 interventions per pilot listed in §9 are illustrative; final vocabulary requires a workshop.
7. **Capacity-constraint data source.** HCF's and Aged Care's day-to-day capacity is not currently exposed via an API. The short-term workaround is an operator-maintained YAML updated daily; long-term requires scheduling-system integration. Operational scope for Sprint 3.
8. **Recommendation cardinality per pilot.** N=1 vs N=3 default is stated above but requires clinical-lead confirmation.
9. **CALD/Indigenous subgroup routing.** When a patient falls in a subgroup with insufficient CATE overlap, the recommender returns "no strong recommendation." Whether these patients should be routed to a human-reviewer queue, and what that queue's SLA is, requires discussion with Aged Care's clinical and equity leads.
10. **Clinician language mapping.** The feature-to-clinician-language lookup (§6.5) is load-bearing for the explanation layer. Requires joint development with clinical leads.
11. **Digital twin validation thresholds.** The DTCF thresholds (what counts as a marginal-validation pass, how much conditional-calibration error is acceptable) must be set per pilot and signed off in governance before Step 3.2 is considered complete.

**Known limitations that will not be closed in Sprint 3.**

12. **Single-step horizon.** Gap 22's CATE is single-step. A patient may receive multiple recommendations across their care episode; each is CATE-scored independently. Sequential correlations (the benefit of *this* intervention depends on *the previous* intervention) are only captured through features in the state representation, not through a proper DTR. For the HCF CHF 30-day window and the Aged Care 90-day window, this is often acceptable; for longer-horizon sequential optimisation, it isn't. Sprint 4.
13. **Digital twin fidelity ceiling.** The bounded-residual twin is interpretable and safe but has a fidelity ceiling compared to state-of-the-art generative-sequence models. This is a deliberate trade: DTCF validation demands interpretable per-component dynamics, and the policy evaluator's safety story rests on that.
14. **Observational confounding is always possible.** Same caveat as Gap 21. CATE estimated from observational data carries unmeasured-confounding risk; the overlap diagnostic and sensitivity bounds quantify but do not eliminate this.
15. **No fairness guarantees at the individual level.** Subgroup fairness monitoring catches population-level patterns; it cannot guarantee individual-level fairness in every recommendation. This is a fundamental limitation of population-trained statistical models, honestly reported.

---

## 14. Success Metrics

**For HCF CHF pilot.**
- Recommender accepted on ≥40% of presented recommendations within 3 months.
- Attribution-weighted 30-day readmission reduction under the recommender policy not worse than the Gap 20 prediction-only policy, with target: equal or better.
- Per-subgroup fairness: recommendation acceptance rate within ±15% across age bands, sex, comorbidity strata, metro/regional, Indigenous status.
- CATE calibration: |observed attributed effect − predicted CATE| < 0.05 on a rolling 90-day average.
- Zero recommendations ever landing on a worklist without a visible Criterion 4 basis artefact.

**For Aged Care pilot.**
- Recommender accepted on ≥35% of presented recommendations within 6 months (lower target than HCF CHF given multi-recommendation cardinality and more complex clinical context).
- Attribution-weighted 90-day admission reduction under recommender policy not worse than Gap 20 policy.
- CALD and Indigenous subgroup routing: any patient with `CATE_INCONCLUSIVE_NO_OVERLAP` successfully routed to human-reviewer queue within SLA.
- Digital twin passes all five DTCF validation levels for the RAC-level-1 cohort; levels that fail for other cohorts are explicitly enumerated and recommendations for those cohorts are gated on human review until remediated.

**For the platform.**
- FDA 2026 CDS compliance pack generated successfully on demand within 24 hours.
- Zero unauthorised policy changes (every policy change has a governance-ledger entry).
- Monthly policy-level rollback drill completed without unplanned incidents.
- At least one policy change in HCF CHF promoted to production through full shadow → canary → promotion cycle with governance sign-off.

---

## 15. References and Further Reading

### CATE estimation and meta-learners
- Künzel SR, Sekhon JS, Bickel PJ, Yu B. *Metalearners for estimating heterogeneous treatment effects using machine learning.* PNAS 2019;116(10):4156–4165.
- Nie X, Wager S. *Quasi-oracle estimation of heterogeneous treatment effects.* Biometrika 2021;108(2):299–319.
- Kennedy EH. *Towards optimal doubly robust estimation of heterogeneous causal effects.* Electronic Journal of Statistics 2023.
- Wager S, Athey S. *Estimation and inference of heterogeneous treatment effects using random forests.* JASA 2018;113(523):1228–1242.
- Hahn PR, Murray JS, Carvalho CM. *Bayesian regression tree models for causal inference.* Bayesian Analysis 2020.
- Curth A, van der Schaar M. *Nonparametric estimation of heterogeneous treatment effects: From theory to learning algorithms.* AISTATS 2021.
- Thelen H, Hennessy S. *Characterizing treatment effect heterogeneity using real-world data.* Clinical Pharmacology & Therapeutics 2025;117(5):1209–1216.
- *Estimating Heterogeneous Treatment Effects With Real-World Health Data: A Scoping Review of Machine Learning Methods.* ScienceDirect 2026.
- *Estimating Heterogeneous Treatment Effects With Target Trial Emulation: A Checklist of Causal Machine Learning for Observational Data.* American College of Chest Physicians 2025.
- *A Large-Scale Empirical Comparison of Meta-Learners and Causal Forests for Heterogeneous Treatment Effect Estimation.* UpliftBench, arXiv 2026.

### Uplift modelling
- Gutierrez P, Gérardy JY. *Causal inference and uplift modelling: A review of the literature.* ICPR 2017.
- Radcliffe NJ, Surry PD. *Real-world uplift modelling with significance-based uplift trees.* White paper 2011.
- Zhao Z, Harinen T. *Uplift modeling for multiple treatments with cost optimization.* DSAA 2019.

### Off-policy evaluation
- Dudík M, Langford J, Li L. *Doubly robust policy evaluation and learning.* ICML 2011.
- Su Y, Dimakopoulou M, Krishnamurthy A, Dudík M. *Doubly robust off-policy evaluation with shrinkage.* ICML 2020.
- Sun H, Hüyük A, van der Schaar M. *When is Off-Policy Evaluation (Reward Modeling) Useful in Contextual Bandits? A Data-Centric Perspective.* arXiv 2024 (DataCOPE).

### Offline reinforcement learning for healthcare
- Fujimoto S, Meger D, Precup D. *Off-policy deep reinforcement learning without exploration.* ICML 2019 (BCQ).
- Kumar A, Zhou A, Tucker G, Levine S. *Conservative Q-learning for offline reinforcement learning.* NeurIPS 2020 (CQL).
- Kidambi R, Rajeswaran A, Netrapalli P, Joachims T. *MOReL: Model-based offline reinforcement learning.* NeurIPS 2020.
- Yan R, Shen X, Wachi A, Gros S, Zhao A, Hu X. *Offline Guarded Safe Reinforcement Learning for Medical Treatment Optimization Strategies.* arXiv 2505.16242, 2025 (OGSRL).
- Jayaraman P, et al. *Optimizing Loop Diuretic Treatment for Mortality Reduction in Patients With Acute Dyspnea Using a Practical Offline Reinforcement Learning Pipeline.* JMIR Medical Informatics 2025 (PROP-RL).
- Lorber, et al. *EpiCare: A Reinforcement Learning Benchmark for Dynamic Treatment Regimes.* NeurIPS 2024.
- Frommeyer TC, et al. *Reinforcement Learning and Its Clinical Applications Within Healthcare: A Systematic Review of Precision Medicine and Dynamic Treatment Regimes.* Healthcare 2025;13(14):1752.

### Digital twins for clinical policy evaluation
- Qin X, et al. *Reinforcement Learning enhanced Online Adaptive Clinical Decision Support via Digital Twin powered Policy and Treatment Effect optimized Reward.* AAAI 2026 (DT4H).
- Vallée A. *From prediction to intervention: causal digital twins for personalized clinical decision support.* Journal of Translational Medicine 2026.
- *The Digital Twin Counterfactual Framework: A Validation Architecture for Simulated Potential Outcomes.* arXiv 2026 (DTCF).
- Riahi V, Diouf I, Khanna S, Boyle J, Hassanzadeh H. *Digital Twins for Clinical and Operational Decision-Making: Scoping Review.* JMIR 2025;27:e55015.
- Vallée A. *Multi-scale digital twins for personalized medicine.* Frontiers in Digital Health 2026.
- *Harnessing the power of virtual (digital) twins: Graphical causal tools for understanding patient and hospital differences.* 2025.

### Contextual bandits and mHealth interventions
- Greenewald K, Tewari A, Murphy S, Klasnja P. *Action-centered contextual bandits.* NeurIPS 2017.
- Tomkins S, et al. *Contextual bandits with debiased machine learning.* NeurIPS 2021.
- Brown K, et al. *A Robust Mixed-Effects Bandit Algorithm for Assessing Mobile Health Interventions.* NeurIPS 2024 (DML-TS-NNR).
- *Designing digital health interventions with causal inference and multi-armed bandits: a review.* Frontiers in Digital Health 2025.

### Regulatory
- US FDA. *Clinical Decision Support Software: Guidance for Industry and FDA Staff.* Final guidance, January 6, 2026.
- US FDA. *Marketing Submission Recommendations for a Predetermined Change Control Plan for Artificial Intelligence-Enabled Device Software Functions.* Final guidance, December 2024.
- FDA, Health Canada, MHRA. *Guiding Principles for Predetermined Change Control Plans.* August 2025.
- European Union. *Artificial Intelligence Act.* Regulation (EU) 2024/1689, Articles 14 and 15.

---

## 16. Suggested Sequencing and Handoff

**Weeks 1–3.** Phase 1 Step 1.1 (intervention taxonomy) + Step 1.2 start (learner scaffolding). Intervention-taxonomy workshop with both clinical leads happens in week 1. Step 1.1 is deceptively important — the taxonomy shape ripples through every downstream step.

**Weeks 3–7.** Phase 1 Steps 1.2–1.4 (CATE committee, overlap, calibration monitoring). Parallel: Phase 2 Step 2.1 (constraint filter, capacity optimiser) — doesn't depend on CATE outputs.

**Weeks 7–11.** Phase 2 Steps 2.2–2.4 (ranking, safety layer, explanation + worklist integration). UX design and clinical-lead review on the Recommendation Panel happen in week 8–9; no live deployment before UX sign-off.

**Weeks 11–15.** Phase 3 (digital twin + DTCF validation + policy evaluator). Parallel with late Phase 2 where possible.

**Weeks 15–18.** Phase 4 (policy governance extensions, shadow→canary→promotion, compliance pack). First policy promotion (HCF CHF) targeted for week 18.

**Handoff priorities.**

1. First 30 minutes: §5 (architecture), §6 (six engines), §7 (phase breakdown), §12 (FDA 2026 compliance detail).
2. First hour: map §11 file layout to existing codebase conventions (KB-23 folded service pattern applies throughout).
3. Day 1: schedule the intervention-taxonomy workshops with both clinical leads — these are the highest-urgency open question (§13 item 6) that blocks everything.
4. Day 2: spike Step 1.2 (learner committee) against a toy synthetic dataset to validate the committee infrastructure independent of the production data pipeline.
5. Week 1: wire up the Policy Governance extension (Step 4.1) as a service skeleton so every subsequent change can land in the ledger from the start — same pattern recommended for Gap 21's Step 4.1.
6. Week 1: review the FDA 2026 CDS compliance design with external regulatory counsel before any UX or workflow decisions are locked. The Criterion 4 design is the load-bearing regulatory piece.

**What "done" looks like for Sprint 3.**
- The platform produces per-patient CATE-based recommendations for HCF CHF and Aged Care pilots.
- Every recommendation carries a Criterion 4-compliant basis that a clinician can independently review.
- The digital twin is validated against DTCF's five-level architecture and is used in offline policy evaluation for every production policy change.
- The FDA 2026 CDS compliance pack is generated and approved by external regulatory counsel.
- At least one full policy cycle (shadow → canary → promotion) has completed in HCF CHF.
- Clinician acceptance rates are in the 40–70% target band for HCF CHF; lower-bound data emerging for Aged Care.
- Aggregate outcome effect is equal or better than the Gap 20 prediction-only baseline, with defensible attribution and per-subgroup fairness evidence.

---

*End of plan. The move from prediction to prescription is the move from "here is what might happen" to "here is what we recommend happen next," made with humility, overlap checks, uncertainty bounds, and a clinician at every step.*
