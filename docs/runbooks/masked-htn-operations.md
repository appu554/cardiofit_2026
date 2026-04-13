# Masked Hypertension Phase 4 — Operations Runbook

_Last updated: 2026-04-13_

## 1. Service Overview

**What ships in Phase 4**
- KB-26 Metabolic Digital Twin now classifies masked / white-coat phenotypes daily using real KB-20 LabEntry readings.
- KB-23 Decision Cards renders eight BP context cards with confidence tiers.
- KB-19 Event Bus receives phenotype change + masked HTN detected events for downstream fan-out.
- KB-20 exposes `/api/v1/patient/:patientId/bp-readings` for per-reading fetches; KB-21 supplies engagement phenotypes.

**Runtime topology**
1. **KB-26** pulls patient profiles from KB-20 + engagement from KB-21, applies phenotype stability engine, persists `bp_context_history`, and emits events via KB-19.
2. **KB-23** ingests events, builds card payloads (YAML fragments in `templates/bp_context`), and responds to clinician UIs.
3. **KB-19** transports `BP_CONTEXT_CLASSIFIED` + `BP_PHENOTYPE_CHANGED` events to subscription queues.
4. **KB-21** supplies engagement phenotype + avoidance scores (best-effort) to bias adjustment.

**Data flow**
Patient LabEntry readings → KB-20 `/bp-readings` → KB-26 orchestrator → stability engine → KB-26 snapshot + KB-19 event → KB-23 card synthesis → clinician UI. Morning surge + medication timing hypotheses depend on per-reading timestamps; fallback to synthetic profile remains available for safety.

## 2. Alerting and Monitoring

Monitor these eight Prometheus metrics emitted by KB-26 (all prefixed `kb26_`). Suggested alert thresholds assume production SLOs; adapt per-region if telemetry differs.

| Metric | Healthy Signal | Warning Threshold | Critical / Action |
| --- | --- | --- | --- |
| `kb26_bp_batch_duration_seconds` (p95) | < 60s | 60–300s → check KB-20/KB-21 latency | >300s or no batch run in 26h → page SWE on-call; consider disabling scheduler via `BP_BATCH_ENABLED=false` |
| `kb26_bp_batch_patients_total{outcome="error"}` (rate) | 0–0.5% | >1% failures in 1h window → inspect KB-20 responses, review logs | >5% or sustained >1h → trigger incident; stop batch percent ramp |
| `kb26_bp_batch_errors_total` | 0 | increments → fatal batch crash; inspect scheduler logs | >1 per 24h or two consecutive days → open Sev2 |
| `kb26_bp_classify_latency_seconds` (p95) | < 500ms | 0.5–2s → KB-20 lag or DB pressure | >2s sustained → consider pausing manual trigger traffic |
| `kb26_bp_classify_errors_total` | 0 | first increment → check KB-20 availability | sustained increase → declare incident |
| `kb26_bp_phenotype_total{phenotype}` | baseline per cohort | sudden drop to zero for all phenotypes → classification stuck | zero classifications in 26h → treat as critical |
| `kb26_kb19_publish_latency_seconds` (p95) | < 250ms | 250–1000ms → KB-19 health degrade | >1s or publish queue backlog → notify KB-19 owner |
| `kb26_kb19_publish_errors_total` | 0 | 1 error → inspect KB-19, check API keys | >3 errors in 1h → halt scheduler and fail open to avoid duplicate traffic |

Alert routing:
- Warnings page masked-HTN feature owner (SWE primary) via Slack + PagerDuty low-urgency.
- Critical alerts page SWE plus Clinical On-Call simultaneously (per HTN program rota).

## 3. Canary & Rollout Plan

1. **Pre-flight (Day -1)**
   - Deploy KB-26 + KB-23 artifacts with `BP_BATCH_ENABLED=false`. Hit `/health` & `/metrics` manually.
   - Verify KB-20 `/bp-readings` returns paired SBP/DBP for a staged patient.
2. **Phase 1 (Day 0)** — Enable API only.
   - Keep scheduler disabled; allow manual `/api/v1/kb26/bp-context/:patientId` triggers for smoke testing.
   - Confirm snapshots and KB-19 events persist.
3. **Phase 2 (Day 2)** — Start 1% batch.
   - Set `BP_BATCH_ENABLED=true`, `BP_BATCH_PERCENT=1`, `BP_BATCH_HOUR_UTC` per market recommendation.
   - Observe metrics for 48h; rollback by flipping `BP_BATCH_ENABLED=false` if anomalies.
4. **Phase 3 (Week 1)** — Ramp to 10% then 25% after 48h stability each.
5. **Phase 4 (Week 2)** — Full rollout (`BP_BATCH_PERCENT=100`). Monitor for two full cycles before removing percent flag.

**Rollback**
- Immediate: `BP_BATCH_ENABLED=false` disables scheduler loop; API/manual trigger still available for critical patients.
- Full rollback: redeploy previous KB-26 image + revert KB-23 templates if YAML migration introduces regressions (pending P7).

## 4. Common Failure Modes

1. **No batch run in >26h**
   - Check scheduler goroutine logs for panic or context cancellation.
   - Confirm host still receiving SIGTERM events (K8s restarts) and ensure `batchScheduler.Drain()` not stuck (look for `Drain waiting` log).
   - If scheduler dead, manually trigger `RunOnce` via `kubectl exec` + `curl -XPOST /api/v1/internal/bp-batch/run` (internal handler), or redeploy pod.

2. **KB-20 endpoint failing (synthetic fallback engaged)**
   - Orchestrator logs `real BP reading fetch failed`. Query KB-20 logs for `/bp-readings` 5xx.
   - If outage >6h, inform clinical ops that medication timing hypothesis is paused (cards still fire using synthetic data but chronotherapy signal degrades).

3. **Phenotype flapping despite stability engine**
   - Check `bp_context_history` entries; verify dwell/resolution. If override needed (e.g., medication change), manually edit next snapshot via SQL or run a forced classification after verifying clinical event.

4. **KB-19 publishes failing**
   - Inspect `kb26_kb19_publish_errors_total` and KB-19 health endpoint.
   - If KB-19 outage >30m, disable scheduler to avoid backlog; once healthy, re-run impacted patients manually.

5. **Selection bias demoting urgency unexpectedly**
   - KB-21 may be returning stale engagement scores. Validate via KB-21 `/api/v1/engagement/:patientId`. If data missing, note in ticket; classification will log `selection bias risk` details.

## 5. Manual Operations

- **Trigger single patient classification**
  ```bash
  curl -XPOST \
    -H "Content-Type: application/json" \
    http://kb26:8137/api/v1/kb26/bp-context/PATIENT_ID
  ```
  Returns latest `BPContextClassification`. Use when clinicians request urgent re-eval or after medication change.

- **Inspect BP context history**
  ```sql
  SELECT snapshot_date, phenotype, clinic_sbp_mean, home_sbp_mean, confidence
  FROM bp_context_history
  WHERE patient_id = 'PATIENT_ID'
  ORDER BY snapshot_date DESC
  LIMIT 30;
  ```

- **Disable batch scheduler immediately**
  - Set env `BP_BATCH_ENABLED=false` and restart deployment (ConfigMap/secret update). Takes effect on next pod start; manual `curl /shutdown` (SIGTERM) ensures quick rollout.

- **Adjust ramp percent**
  - Update `BP_BATCH_PERCENT` (1–100). Scheduler reads during startup; restart required. Document change in release log.

- **Manual event replay**
  - If KB-19 missed events, re-fetch latest snapshot and POST to KB-19 `/api/v1/events` using stored payload template (see KB-19 runbook §4.2).

## 6. Glossary

- **Phenotypes**
  - `MASKED_HTN`, `MASKED_UNCONTROLLED`, `WHITE_COAT_HTN`, `WHITE_COAT_UNCONTROLLED`, `SUSTAINED_HTN`, `SELECTION_BIAS_WARNING` (auxiliary), `MEDICATION_TIMING`.
- **Amplification Flags** — `DiabetesAmplification`, `CKDAmplification`, `MorningSurgeCompound` boost urgency.
- **Stability Engine Terms**
  - **MinDwell**: 14 days the phenotype must remain before accepting transitions.
  - **FlapWindow**: 30-day lookback for counting transitions; >3 flips locks state.
  - **Override**: Future hook (Phase 5) allowing medication-change events to bypass dwell.
- **Scheduler Flags**
  - `BP_BATCH_ENABLED`, `BP_BATCH_PERCENT`, `BP_BATCH_HOUR_UTC`, `BP_ACTIVE_WINDOW_DAYS`.
- **Selection Bias Categories** — measurement-avoidant vs crisis-only (from KB-21). Drives urgency demotion + SELECTION_BIAS_WARNING card.

## 7. Contacts & Escalation

1. **Primary SWE (Masked HTN)** — `#kb-htn-phase4` Slack channel / PagerDuty schedule `Masked HTN SWE`.
2. **Clinical Reviewer** — reach via `#clinical-htn` for card text or guideline escalations.
3. **Infra Support** — `#kb-platform` for K8s / Postgres / Redis incidents.

Escalate Sev1 when:
- BP batch halts for >26h across markets.
- KB-19 publishes fail for >1h without viable workaround.
- Clinical safety risk identified (incorrect phenotype with HIGH confidence).
