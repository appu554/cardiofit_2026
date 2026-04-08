# E2E 14-Day Clinical Pipeline Playbook

## Overview

**Duration:** 14 calendar days of real-time injection  
**Patients:** 3 (Rajesh, Priya, Amit)  
**Total events:** ~761 across all days  
**Modules tested:** M7, M8, M9, M10, M10b, M11, M11b, M12, M12b, M13  

Run the generator first:
```bash
python3 e2e_14day_generator.py --start-date 2026-04-09 --output-dir ./e2e_14day_dataset
```

Inject each day:
```bash
./e2e_14day_dataset/inject_day.sh <day_number> <kafka_bootstrap>
```

---

## Patient Profiles

### Patient A — Rajesh Kumar (The Deteriorator)
- **58M**, T2DM 12yr, HTN Stage 2, CKD 3b (eGFR 42)
- **Meds:** Metformin 1000mg BD, Amlodipine 10mg, Telmisartan 80mg, Dapagliflozin 10mg, HCTZ 12.5mg
- **Clinical story:** Triple whammy medication combo. Nausea starts day 4. Amlodipine uptitrated day 5, Metformin day 7. BP worsens throughout. Engagement collapses days 8-14. Crisis-level readings days 11-12. eGFR falls to 45 by day 14 (critical Metformin threshold).
- **Why this profile:** Exercises M7 full escalation, M8 triple whammy (CID_01) + DKA (CID_04) + Metformin/eGFR (CID_03), M9 engagement collapse to MEASUREMENT_AVOIDANT, M12 concurrent dual interventions, M13 multi-domain DETERIORATING with cross-domain amplification.

### Patient B — Priya Sharma (The Improver)
- **45F**, HTN Stage 1, Pre-diabetes (HbA1c 6.3)
- **Meds:** Amlodipine 5mg OD → Telmisartan 40mg added day 1
- **Clinical story:** Steady BP improvement after ARB addition. Regular meals with CGM. Regular exercise. Sustained high engagement. BP moves from 148/92 to 128/78 over 14 days.
- **Why this profile:** M7 improvement trajectory + good dipper (night readings). M8 negative test (ZERO alerts expected). M10/M10b consistent meal patterns with improving post-prandial glucose. M11/M11b regular fitness patterns. M12/M12b intervention with 12-day window so WINDOW_CLOSED fires day 13, testing M12b delta computation. M13 stable/improving detection.

### Patient C — Amit Patel (Edge Case Specialist)
- **52M**, newly diagnosed HTN, T2DM 2yr
- **Meds:** Enalapril 10mg, Metformin 500mg BD, HCTZ 25mg
- **Clinical story:** Looks normal in clinic (day 5: clinic SBP 126), but high at home (SBP 142). Classic masked HTN. Nocturnal non-dipper pattern. Acute surge event day 7 (148→165 within 2h). Potassium 5.2 on day 5 with ACEi.
- **Why this profile:** M7 masked_htn_suspected, dip_classification (NON_DIPPER), acute_surge_flag, clinic_home_gap_sbp. M8 hyperkalemia + ACEi alert.

---

## Daily Schedule

### Day 1 (Baseline) — 70 events

**Inject:** Morning + Evening BP × 3 patients, medication orders (Rajesh ×5, Priya ×1, Amit ×3), labs (Rajesh eGFR 52 + FBG 142, Amit eGFR 68 + FBG 130 + K 4.4, Priya FBG 118), Priya intervention (Telmisartan 40mg), meals + CGM + activities + engagement for all.

**Check immediately:**
- [ ] M7: 6 BP variability outputs (2 per patient). First readings should show `context_depth: INITIAL`, `variability_classification_7d: INSUFFICIENT_DATA`
- [ ] M8: Rajesh CID_01 (triple whammy) should fire on medication orders (ARB + SGLT2i + Diuretic). **No symptoms yet**, so CID_04 should NOT fire
- [ ] M8: Priya and Amit — ZERO alerts
- [ ] M8: Verify each alert has unique `suppressionKey`
- [ ] M12: Priya WINDOW_OPENED for Telmisartan (12-day observation window)
- [ ] M13: Initial state change event. May show `data_completeness` low on first computation

**Check after 23:59 UTC (05:29 IST next morning):**
- [ ] M9: Engagement signals for all 3 patients. Rajesh composite ~0.65-0.72, Priya ~0.80+

**Check ~3h05m after lunch injection:**
- [ ] M10: Meal response for Rajesh (post-prandial CGM: 155→185→210→190) and Priya (110→135→155→120)

**Check ~2h05m after activity injection:**
- [ ] M11: Activity response for Priya (brisk walk 30min) and Rajesh day 2 activity (not today for Rajesh)

---

### Day 2 — 62 events

**Inject:** BPs, labs, meals + CGM, Rajesh activity (brisk walk 25min) + post-activity HR recovery, engagement.

**Check immediately:**
- [ ] M7: Cumulative 12 outputs (6 new). Rajesh `context_depth` transitioning to `BUILDING`
- [ ] M7: ARV should start appearing (verify computed across ALL readings, not time-context silos)

**Check after timers:**
- [ ] M9: Day 2 engagement for all patients
- [ ] M10: Rajesh + Priya meal responses
- [ ] M11: Rajesh activity response — HR recovery curve (105→98→88→80)

---

### Day 3 — 60 events (includes NIGHT readings)

**Inject:** BPs including NIGHT for Rajesh (148/90), Priya (128/78), Amit (134/84). Labs, meals, engagement.

**Check immediately:**
- [ ] M7: **CRITICAL — Dip ratio computation.** Night readings now available for all 3 patients
  - Priya night 128 vs morning 144 → dip ratio ~0.89 → DIPPER (normal)
  - Amit night 134 vs morning 136 → dip ratio ~0.99 → **NON_DIPPER** (should flag)
  - Rajesh night 148 vs morning 158 → dip ratio ~0.94 → borderline/non-dipper
- [ ] M7: `variability_classification_7d` should exit `INSUFFICIENT_DATA` for patients with 4+ readings
- [ ] M7: Morning surge computable (morning SBP − preceding evening SBP)
  - Rajesh day 3: 158 − 140 (day 2 evening) = 18 → NORMAL
  - Priya day 3: 144 − 136 (day 2 evening) = 8 → NORMAL
- [ ] M7: `context_depth` should be `BUILDING` (3 days of data)

---

### Day 4 — 63 events (nausea onset)

**Inject:** BPs, **Rajesh nausea symptom (mild)**, meals + CGM, Rajesh activity (brisk walk 20min), engagement.

**Check immediately:**
- [ ] M7: `context_depth` approaching `ESTABLISHED` (4 readings ≥ 4 is typical threshold)
- [ ] M7: Rajesh SBP 7d avg should be ~155-160 range
- [ ] M8: **Rajesh CID_04 should NOW fire** — SGLT2i (Dapagliflozin) + nausea = euglycemic DKA risk. Severity HALT
- [ ] M8: CID_01 may re-fire (nausea = illness signal, re-triggering triple whammy). **Check suppressionKey dedup**
- [ ] M8: Priya still ZERO, Amit still ZERO

**Check after timers:**
- [ ] M10: Rajesh meal response — post-meal glucose peak 225 (worsening)
- [ ] M11: Rajesh activity response — HR recovery 108→100→90→82 (slower than day 2)

---

### Day 5 — 71 events (clinic visit + intervention + edge cases)

**THE BIG DAY — most assertions of any day.**

**Inject:** Rajesh CLINIC + HOME_CUFF (170 clinic, 155 home), Amit CLINIC + HOME_CUFF (126 clinic, 142 home → masked HTN), labs (Rajesh eGFR 50 + HbA1c 8.2 + K 4.6 + ACR 45, Amit eGFR 65 + K 5.2), Rajesh nausea worsening (moderate), **Amlodipine uptitration 10→15mg for Rajesh**.

**Check immediately:**
- [ ] M7 Rajesh: `white_coat_suspected` — clinic SBP 170 vs home 155. Gap = +15. Evaluate threshold
- [ ] M7 Amit: **`masked_htn_suspected` should be TRUE** — clinic SBP 126 vs home 142. Gap = −16. Home >> Clinic
- [ ] M7 Amit: `clinic_home_gap_sbp` should be approximately −16
- [ ] M7: Both patients should have 5 days of data, `context_depth: ESTABLISHED`
- [ ] M8 Rajesh: CID_04 should fire/re-fire with worsening nausea (moderate)
- [ ] M8 Rajesh: **CID_03 should fire** — Metformin + eGFR 50 (approaching dose-reduction threshold)
- [ ] M8 Rajesh: ACR 45 > 30 threshold — microalbuminuria alert if rule exists
- [ ] M8 Amit: **Hyperkalemia alert must fire** — K=5.2 with ACEi (Enalapril). This is a safety-critical drug interaction
- [ ] M8: Priya still ZERO
- [ ] M8: **DEDUPLICATION CHECK** — count unique suppressionKeys vs total alerts. Must be equal
- [ ] M12: Rajesh WINDOW_OPENED for Amlodipine uptitration (14-day observation window, ends ~day 19)
- [ ] M12: Check `concurrent_intervention_count` — should be 0 (no other active interventions yet)
- [ ] M13: With eGFR declining (52→50) and BP worsening, RENAL velocity should be > 0

---

### Day 6 — 59 events (engagement starts declining)

**Inject:** BPs including Amit NIGHT (142/90 — non-dipper), Rajesh engagement declining (app session 90s, 2/5 goals), meals but Rajesh portion larger (sodium 880mg).

**Check immediately:**
- [ ] M7 Amit: **Non-dipper confirmed** — night SBP 142 vs morning 145. Dip ratio ~0.98 → NON_DIPPER
- [ ] M7 Rajesh: `variability_classification_7d` trending toward ELEVATED or HIGH (5+ days, increasing spread)
- [ ] M7 Rajesh: Morning surge today = 165 − 148 (day 5 evening) = 17 → NORMAL

**Check after 23:59 UTC:**
- [ ] M9 Rajesh: Engagement score should be visibly lower than days 1-5. App session only 90s, 2/5 goals

---

### Day 7 — 61 events (second intervention + weekly boundary)

**IMPORTANT — Weekly boundary. M10b/M11b may fire on next Monday.**

**Inject:** BPs including Rajesh NIGHT (152/94), **Rajesh Metformin uptitration 1000→1500mg**, labs (Rajesh eGFR 48 + FBG 158), engagement declining further (60s session).

**Check immediately:**
- [ ] M7: Full 7 days of data. `days_with_data_in_7d: 7`. Morning surge 7d average now computable
- [ ] M7 Rajesh: `morning_surge_7d_avg` should be calculable. Check if NORMAL or ELEVATED
- [ ] M7 Rajesh: ARV should show ELEVATED pattern across 7 days
- [ ] M7 Priya: BP consistently improving — `sbp_7d_avg` should show downward trend from ~145 to ~135
- [ ] M8 Rajesh: eGFR 48 — CID_03 should update/re-fire with increased severity (approaching 45 threshold)
- [ ] M12: Rajesh WINDOW_OPENED for Metformin uptitration (21-day observation window)
- [ ] M12: **CONCURRENT CHECK** — Metformin window should show `concurrent_intervention_count: 1` (Amlodipine)
- [ ] M12: **BIDIRECTIONAL CHECK** — Does Amlodipine window retroactively update to show Metformin as concurrent?
- [ ] M13: 7-day snapshot rotation should fire (if using event-time watermarks). Domain velocities should populate

**Check Monday 00:00 UTC (if day 7 falls before Monday):**
- [ ] M10b: Weekly meal pattern aggregation should fire if M10 output exists from days 1-7
- [ ] M11b: Weekly fitness pattern aggregation should fire if M11 output exists

---

### Day 8 — 56 events (deterioration accelerates)

**Inject:** Rajesh morning BP 172 + evening 150, meal log but NO CGM (stopped wearing device), no activity. Priya + Amit continue normally.

**Check immediately:**
- [ ] M7 Rajesh: `sbp_7d_avg` should be rising noticeably (>155)
- [ ] M7 Rajesh: With 7-day sliding window, older stable readings drop out, newer high readings dominate
- [ ] M9 Rajesh: Engagement very low — 30s app session, 1/5 goals, 1500 steps. Approaching RED

**Check after 23:59 UTC:**
- [ ] M9 Rajesh: Composite should be < 0.40 (YELLOW→RED transition territory)

---

### Day 9 — 49 events (engagement collapsing)

**Inject:** Rajesh morning BP 176 + evening 155. FBG 168. NO meal log, NO activity, NO app session. Only medication taken. Priya + Amit continue.

**Check immediately:**
- [ ] M7 Rajesh: SBP trending high. `variability_classification_7d` should be HIGH
- [ ] M10: No Rajesh meal response today (no meal log)
- [ ] M11: No Rajesh activity response today (no activity)

**Check after 23:59 UTC:**
- [ ] M9 Rajesh: Engagement score should be very low (~0.25-0.30). Only signal = medication taken
- [ ] M9 Rajesh: If MEASUREMENT_AVOIDANT phenotype detection exists, should be flagging risk

---

### Day 10 — 38 events (missed readings)

**Inject:** Rajesh morning BP 178 ONLY — **NO evening BP** (first missed reading). Priya + Amit continue with morning + evening.

**Check immediately:**
- [ ] M7 Rajesh: Only 1 reading today. `days_with_data_in_7d` still 7 (earlier days still in window)
- [ ] M7 Rajesh: No morning surge computable today (no preceding evening to pair with... unless pairing with day 9 evening)
- [ ] M9 Rajesh: Nearly zero engagement. No app, no meal, no goal, no med taken, minimal steps

---

### Day 11 — 45 events (crisis territory)

**Inject:** Rajesh morning BP **182/110** (CRISIS) + evening 160/100. FBG 175. Dizziness + ongoing nausea. Priya + Amit continue.

**Check immediately:**
- [ ] **M7 Rajesh: `crisis_flag: TRUE`** — SBP ≥ 180. This is the most critical clinical flag
- [ ] M7 Rajesh: `variability_classification_7d: HIGH` — SBP range now 155-182
- [ ] M7 Rajesh: `sbp_7d_avg` should be > 165
- [ ] M7 Rajesh: `morning_surge_today` = 182 − 155 (day 9 evening) = 27 → ELEVATED
- [ ] M8 Rajesh: New symptom (dizziness) + ongoing nausea may re-trigger CID_04 or new rules
- [ ] M13: **CRITICAL CHECKPOINT** — By day 11, M13 should show:
  - `composite_classification: DETERIORATING`
  - RENAL velocity > 0 (eGFR 52→50→48)
  - CARDIOVASCULAR velocity > 0 (if ARV bug is fixed)
  - `cross_domain_amplification: true`
  - `domains_deteriorating >= 2`
  - `confidence_score > 0.5`

---

### Day 12 — 45 events (Rajesh near-absent, M12 midpoint)

**Inject:** Rajesh morning BP **180/108** (CRISIS again) — **NO evening BP**. Priya + Amit with NIGHT readings.

**Check immediately:**
- [ ] M7 Rajesh: `crisis_flag: TRUE` again. Two consecutive crisis-level mornings
- [ ] M7 Priya: Night 114/72 vs morning 128/78 → dip ratio ~0.89 → good DIPPER. BP well controlled now
- [ ] M12: **Amlodipine MIDPOINT timer should fire** (day 5 + 7 = day 12). Check `clinical.intervention-window-signals` for MIDPOINT signal
- [ ] M12: Midpoint signal should contain trajectory assessment and any adherence signals

---

### Day 13 — 39 events (Rajesh disengaged, Priya window closes)

**Inject:** Rajesh **NO morning BP**, single reluctant evening 164/100. Priya morning 130 + evening 120.

**Check immediately:**
- [ ] M7 Rajesh: Only 1 evening reading. Gap from day 12 morning. Missing data pattern
- [ ] M7 Priya: SBP steadily improved. 7d avg should be ~125-130 now. Classification LOW or NORMAL variability
- [ ] M12: **Priya Telmisartan WINDOW_CLOSED should fire** (day 1 + 12 days + 1 day grace ≈ day 14). May fire today or tomorrow depending on exact timer registration

**Check when M12 emits WINDOW_CLOSED for Priya:**
- [ ] **M12b: Intervention delta should compute** — Priya pre-window BP avg ~148/92, post-window avg ~128/78. Delta should show clear improvement
- [ ] M12b: This is the FIRST M12b output in the entire test. Verify the schema and content

---

### Day 14 — 43 events (final snapshot)

**Inject:** Rajesh morning BP 174, labs (eGFR **45** — CRITICAL, FBG 180, K 4.9). Priya morning 128 + evening 118, labs (eGFR 87 stable, FBG 105, HbA1c 6.1).

**Check immediately:**
- [ ] M7: Final variability snapshots for all patients
  - Rajesh: HIGH variability, crisis_flag history, STAGE_2_UNCONTROLLED throughout
  - Priya: LOW/NORMAL variability, SBP ~128, CONTROLLED or approaching controlled
  - Amit: ELEVATED variability, masked_htn pattern, non-dipper pattern
- [ ] M8 Rajesh: **eGFR 45 = critical Metformin dose-reduction threshold.** CID_03 should fire at highest severity
- [ ] M8 Rajesh: K 4.9 — approaching 5.0 warning, no ACEi so may not trigger drug-K interaction
- [ ] M8 Priya: Still ZERO alerts for entire 14-day run
- [ ] M13 Rajesh: Final state assessment:
  - `composite_classification: DETERIORATING`
  - RENAL velocity high (eGFR 52→50→48→45 — 13% decline in 14 days)
  - CARDIOVASCULAR velocity high (SBP avg increased ~15 mmHg, crisis flags, HIGH variability)
  - METABOLIC velocity should be > 0 (FBG 142→180, if M13 maps FBG to METABOLIC)
  - `confidence_score` should now be > 0.5 with 14 days of data
- [ ] M13 Priya: Should show STABLE or IMPROVING. No domain deteriorating

---

## Module-Specific Feature Coverage Matrix

| Feature | Patient | Day(s) | Expected Result |
|---------|---------|--------|-----------------|
| **M7: ARV cross-context** | Rajesh | 3+ | ARV computed across ALL readings (morning+evening), not within silos |
| **M7: crisis_flag** | Rajesh | 11,12 | TRUE for SBP ≥ 180 |
| **M7: acute_surge_flag** | Amit | 7 | Two morning readings 148→165 within 2h |
| **M7: morning_surge** | All | 3+ | Morning SBP − preceding evening SBP |
| **M7: dip_ratio** | All | 3,6,9,12 | Night vs day. Priya=DIPPER, Amit=NON_DIPPER |
| **M7: masked_htn** | Amit | 5 | Clinic 126 vs Home 142 → suspected |
| **M7: white_coat** | Rajesh | 5 | Clinic 170 vs Home 155 → evaluate |
| **M7: clinic_home_gap** | Both | 5 | Non-null, correct direction |
| **M7: context_depth** | All | 1-7 | INITIAL → BUILDING → ESTABLISHED |
| **M7: bp_control_status** | All | all | Rajesh=STAGE_2, Priya=improving, Amit=masked |
| **M8: CID_01 triple whammy** | Rajesh | 1 | ARB+SGLT2i+Diuretic |
| **M8: CID_04 DKA** | Rajesh | 4+ | SGLT2i+nausea |
| **M8: CID_03 Metformin/eGFR** | Rajesh | 5,7,14 | Escalating severity as eGFR falls |
| **M8: hyperkalemia+ACEi** | Amit | 5 | K=5.2+Enalapril |
| **M8: negative test** | Priya | all | ZERO alerts entire 14 days |
| **M8: deduplication** | All | all | suppressionKey uniqueness |
| **M9: daily composite** | All | all | Fires at 23:59 UTC |
| **M9: engagement collapse** | Rajesh | 7→14 | 0.72 → 0.18 decline |
| **M9: MEASUREMENT_AVOIDANT** | Rajesh | 10+ | Phenotype detection |
| **M9: sustained engagement** | Priya | all | Consistently > 0.70 |
| **M10: meal session** | Rajesh,Priya | 1-8 | Post-prandial CGM curve captured |
| **M10: no meal = no output** | Rajesh | 9-14 | No meal log → no response |
| **M10b: weekly pattern** | Both | Mon | Aggregation of week's meals |
| **M11: activity session** | Rajesh,Priya | active days | HR recovery curve |
| **M11: declining recovery** | Rajesh | 2→6 | Slower HR recovery over time |
| **M11b: weekly pattern** | Both | Mon | Aggregation of week's activities |
| **M12: WINDOW_OPENED** | Rajesh,Priya | 1,5,7 | 3 intervention windows |
| **M12: concurrent detection** | Rajesh | 7 | Metformin window sees Amlodipine |
| **M12: bidirectional** | Rajesh | 7 | Amlodipine retroactively updated? |
| **M12: MIDPOINT** | Rajesh | 12 | Amlodipine midpoint (day 5+7) |
| **M12: WINDOW_CLOSED** | Priya | 13-14 | Telmisartan 12d window closes |
| **M12b: intervention delta** | Priya | 13-14 | BP improvement computed |
| **M13: multi-domain deterioration** | Rajesh | 7+ | RENAL+CARDIO+METABOLIC |
| **M13: cross-domain amplification** | Rajesh | 10+ | amplification_factor > 1 |
| **M13: improving detection** | Priya | 14 | STABLE or IMPROVING |
| **M13: data_completeness** | All | all | Should increase over time |

---

## Known Bug Tracking

Check these against actual output each day. If any fires incorrectly, log the bug:

| Bug ID | Module | Description | Expected Fix Day | Pass? |
|--------|--------|-------------|------------------|-------|
| BUG-01 | M7 | ARV computed within time-context silos, not across all readings | Pre-test | |
| BUG-02 | M7 | Morning surge skips most recent evening reading | Pre-test | |
| BUG-03 | M8 | Suppression key deduplication not enforced (5x duplicate alerts) | Pre-test | |
| BUG-04 | M13 | CARDIOVASCULAR velocity = 0.0 (uses bp_control_status only) | Pre-test | |
| BUG-05 | M13 | Low confidence (0.375) triggers HIGH-priority urgent actions | Day 7+ | |
| BUG-06 | M12 | Concurrent intervention detection is one-directional | Day 7 | |
| BUG-07 | M8 | CID_03 format-fragile (fired in one run, not another) | Day 5 | |
| BUG-08 | M13 | METABOLIC velocity = 0.0 despite FBG 142→180 | Day 14 | |
| NEW-01 | M7 | Dip ratio — verify Amit shows NON_DIPPER | Day 3 | |
| NEW-02 | M7 | Masked HTN — verify Amit flags on day 5 | Day 5 | |
| NEW-03 | M7 | Acute surge — verify Amit flags on day 7 | Day 7 | |
| NEW-04 | M8 | Hyperkalemia + ACEi — verify Amit alert day 5 | Day 5 | |
| NEW-05 | M8 | eGFR 45 critical threshold — verify Rajesh day 14 | Day 14 | |
| NEW-06 | M12b | First-ever M12b output — verify schema and content | Day 13-14 | |

---

## Daily Kafka Topic Monitoring Commands

```bash
# Check M7 output count
kafka-console-consumer.sh --bootstrap-server $KAFKA --topic flink.bp-variability-metrics \
  --from-beginning --timeout-ms 5000 2>/dev/null | wc -l

# Check M8 alerts
kafka-console-consumer.sh --bootstrap-server $KAFKA --topic alerts.comorbidity-interactions \
  --from-beginning --timeout-ms 5000 2>/dev/null | jq '.ruleId, .severity'

# Check M8 deduplication
kafka-console-consumer.sh --bootstrap-server $KAFKA --topic alerts.comorbidity-interactions \
  --from-beginning --timeout-ms 5000 2>/dev/null | jq -r '.suppressionKey' | sort | uniq -c | sort -rn

# Check M9 engagement (after 23:59 UTC)
kafka-console-consumer.sh --bootstrap-server $KAFKA --topic flink.engagement-signals \
  --from-beginning --timeout-ms 5000 2>/dev/null | jq '.'

# Check M10 meal response (~3h after lunch)
kafka-console-consumer.sh --bootstrap-server $KAFKA --topic flink.meal-response \
  --from-beginning --timeout-ms 5000 2>/dev/null | jq '.'

# Check M11 activity response (~2h after activity)
kafka-console-consumer.sh --bootstrap-server $KAFKA --topic flink.activity-response \
  --from-beginning --timeout-ms 5000 2>/dev/null | jq '.'

# Check M12 intervention windows
kafka-console-consumer.sh --bootstrap-server $KAFKA --topic clinical.intervention-window-signals \
  --from-beginning --timeout-ms 5000 2>/dev/null | jq '.signal_type, .intervention_id'

# Check M12b intervention deltas
kafka-console-consumer.sh --bootstrap-server $KAFKA --topic flink.intervention-deltas \
  --from-beginning --timeout-ms 5000 2>/dev/null | jq '.'

# Check M13 state changes
kafka-console-consumer.sh --bootstrap-server $KAFKA --topic clinical.state-change-events \
  --from-beginning --timeout-ms 5000 2>/dev/null | jq '.change_type, .ckm_velocity_at_change.domain_velocities'

# Check M10b/M11b (Monday after first week)
kafka-console-consumer.sh --bootstrap-server $KAFKA --topic flink.meal-patterns \
  --from-beginning --timeout-ms 5000 2>/dev/null | jq '.'
kafka-console-consumer.sh --bootstrap-server $KAFKA --topic flink.fitness-patterns \
  --from-beginning --timeout-ms 5000 2>/dev/null | jq '.'
```

---

## Pass/Fail Criteria at Day 14

The pipeline passes this E2E test when ALL of the following are true:

### Hard Requirements (must pass)
1. M7 produces 1:1 output for every BP input (93 readings → 93 outputs)
2. M7 ARV for Rajesh at day 14 ≥ 10 (true cross-context ARV)
3. M7 crisis_flag = TRUE for at least 2 Rajesh readings (SBP 180, 182)
4. M7 Amit masked_htn_suspected = TRUE on day 5
5. M8 Rajesh CID_01 fires at least once (triple whammy)
6. M8 Rajesh CID_04 fires (DKA risk)
7. M8 Priya has ZERO alerts across all 14 days
8. M8 unique suppressionKeys = total alert count (deduplication works)
9. M9 produces daily engagement signal for each patient on each day
10. M10 produces meal response for at least 5 distinct days
11. M12 produces 3 WINDOW_OPENED signals (Rajesh ×2, Priya ×1)
12. M13 Rajesh composite_classification = DETERIORATING by day 14
13. M13 Rajesh confidence_score > 0.5 by day 14

### Soft Requirements (should pass, flag if not)
14. M7 dip_classification shows Amit as NON_DIPPER
15. M7 acute_surge_flag for Amit on day 7
16. M8 Amit hyperkalemia + ACEi alert fires
17. M8 Rajesh CID_03 fires for Metformin + eGFR
18. M10b produces weekly meal pattern after first Monday
19. M11b produces weekly fitness pattern after first Monday
20. M12 midpoint fires for Rajesh Amlodipine at day 12
21. M12 WINDOW_CLOSED fires for Priya at day 13-14
22. M12b produces intervention delta for Priya
23. M13 RENAL velocity > 0 for Rajesh
24. M13 CARDIOVASCULAR velocity > 0 for Rajesh
25. M13 cross_domain_amplification = true for Rajesh
