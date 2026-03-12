# Page 74 Audit — PP 3.2.2 (Avoid Sedentary), PP 3.2.3 (Falls/Exercise Type), PP 3.2.4 (Obesity/Weight Loss), Figure 22

| Field | Value |
|-------|-------|
| **Page** | 74 (PDF page S73) |
| **Content Type** | PP 3.2.1 closing (dialysis exercise: home-based, intradialytic), PP 3.2.2 (avoid sedentary behavior, short bouts of exercise, spread activity over week), PP 3.2.3 (falls risk patients: exercise intensity/type guidance, sarcopenia, multicomponent activities, individualized recommendations), PP 3.2.4 (obesity+diabetes+CKD weight loss for eGFR ≥30; BMI >30 kg/m² risk factor; BMI >27.5 for Asians; eGFR <30 malnutrition/muscle-wasting concern; higher BMI paradox in dialysis), Figure 22 (suggested approach to physical inactivity in CKD — decision algorithm) |
| **Extracted Spans** | 7 total (4 T1, 3 T2) |
| **Channels** | C, F |
| **Disagreements** | 1 |
| **Review Status** | PENDING: 7 |
| **Risk** | Disagreement |
| **Cross-Check** | Count verified against pipeline DB |
| **Audit Date** | 2026-02-25 |

---

## Source PDF Content

**PP 3.2.1 Closing (from p73):**
- Home-based exercise programs feasible and beneficial for dialysis patients
- Intradialytic exercise → improved hemodialysis adequacy, exercise capacity, depression, quality of life

**Practice Point 3.2.2:**
- "Patients should be advised to avoid sedentary behavior"
- CKD patients often sedentary → increased mortality risk
- Limited exercise tolerance → short bouts of exercise still offer health benefits
- **Accumulated weekly activity is critical** — shorter bouts yield similar benefits to intense physical activity
- Activity spread throughout the week to maximize benefits

**Practice Point 3.2.3:**
- "For patients at higher risk of falls, healthcare providers should provide advice on intensity (low/moderate/vigorous) and type (aerobic vs resistance or both)"
- Sarcopenia common in CKD → adverse outcomes
- Multicomponent activities: aerobic + muscle-strengthening + balance-training
- Muscle-strengthening promotes weight/lean body mass maintenance
- Recommendations individualized by age, comorbidities, baseline activity
- Referral to physical activity specialist if resources available

**Practice Point 3.2.4:**
- "Physicians should consider advising patients with obesity, diabetes, and CKD to lose weight, particularly patients with **eGFR ≥30 ml/min/1.73 m²**"
- **Obesity: BMI >30 kg/m²** = independent risk factor for CKD progression and CVD
- **Asian populations: BMI >27.5 kg/m²** increases adverse outcome risk
- Pooled data from 40 countries (~5.5 million adults): higher BMI, waist circumference, waist-to-height ratio → independent risk factors for kidney decline and death
- Intentional weight loss → reduced albuminuria, improved BP, potential kidney benefits in mild-moderate CKD
- **eGFR <30 or dialysis**: patients may spontaneously reduce intake → **malnutrition and muscle-wasting concerns**
- Differentiating unintentional from intentional weight loss challenging with declining kidney function
- **Higher BMI associated with better outcomes in dialysis** (obesity paradox)

**Figure 22 — Suggested Approach to Physical Inactivity in CKD:**
Decision algorithm:
1. Assess baseline physical activity level
2. If active >150 min/wk → assess muscle-strengthening
3. If active <150 min/wk → recommend increase to >150 min/wk
4. If sedentary → assess fall risk and comorbidity burden
   - Low risk → recommend low-intensity, increase as tolerated
   - High risk → referral to exercise specialists

---

## Key Spans Assessment

### Tier 1 Spans (4)

| Span | Channel | Conf | Assessment |
|------|---------|------|------------|
| "Practice Point 3.2.2" (C) | C | 98% | **→ T3** — PP label only; PP 3.2.2 text about sedentary behavior NOT captured |
| "Practice Point 3.2.3" (C) | C | 98% | **→ T3** — PP label only; PP 3.2.3 text about falls/exercise type NOT captured |
| "Practice Point 3.2.4" (C) | C | 98% | **→ T3** — PP label only; PP 3.2.4 text about obesity/weight loss NOT captured |
| "With an eGFR <30 mL/min/1.73m2, and kidney failure treated with dialysis, patients may spontaneously reduce dietary inta..." (C+F) | C+F | 95% | **✅ T1 CORRECT — EXCELLENT** — C+F multi-channel with disagreement. Captures the critical safety caveat: eGFR <30 + dialysis → malnutrition/muscle-wasting risk. Specific threshold (eGFR <30) with patient safety implication. This is genuine T1 content. |

**Summary: 1/4 T1 genuine (C+F eGFR <30 malnutrition safety caveat — excellent capture). 3 are PP labels → T3.**

### Tier 2 Spans (3)

| Category | Count | Assessment |
|----------|-------|------------|
| **`<!-- PAGE 74 -->`** (F) | 1 | **→ NOISE** — Pipeline artifact (7th occurrence) |
| **"at baseline"** (C) | 1 | **→ T3** — Two-word temporal phrase without context |
| **"Obesity (defined by body mass index [BMI] >30 kg/m2) is an independent risk factor for kidney disease progression and CV..."** (F) | 1 | **⚠️ SHOULD BE T1** — Obesity definition with specific BMI threshold (>30 kg/m²) + clinical significance (CKD progression + CVD risk). This is a clinically actionable threshold. |

**Summary: 1/3 T2 should be T1 (obesity BMI threshold). 1 pipeline artifact. 1 "at baseline" noise.**

---

## Critical Findings

### ✅ C+F Multi-Channel eGFR <30 Safety Caveat — EXCELLENT

The C+F span capturing "With an eGFR <30 mL/min/1.73m2, and kidney failure treated with dialysis, patients may spontaneously reduce dietary intake, and malnutrition and muscle-wasting are potential concerns" is a genuine T1 patient safety extraction:
- **Specific threshold**: eGFR <30 ml/min/1.73m²
- **Population**: dialysis patients
- **Safety concern**: malnutrition and muscle-wasting
- **Clinical action**: weight loss advice should NOT be given below this threshold

The C channel fires on "eGFR <30 mL/min/1.73m2" and F extracts the surrounding sentence. The disagreement flag likely reflects slightly different text boundaries between the two channels.

### ✅ Obesity BMI Threshold Captured — F Channel

The F channel captures the obesity definition sentence with BMI >30 kg/m² as an independent risk factor. This should be T1 (specific clinical threshold with actionable implication).

### ❌ Asian BMI Threshold NOT EXTRACTED

"Among Asian populations, having a BMI >27.5 kg/m² increases the risk for adverse outcomes" — a population-specific threshold important for clinical practice in Asian countries. Not captured by any channel.

### ❌ Obesity Paradox NOT EXTRACTED

"Higher BMI has been associated with better outcomes among patients treated with dialysis" — the obesity paradox is a critical clinical nuance that modifies the weight loss recommendation for dialysis patients. Not captured.

### ❌ Figure 22 Decision Algorithm NOT EXTRACTED

Figure 22 provides a clinical decision algorithm for addressing physical inactivity in CKD, with branching based on baseline activity level, fall risk, and comorbidity burden. This is actionable implementation content. The D channel did not fire on this figure.

### ❌ Missing Critical Content

| Missing Content | Tier | Clinical Importance |
|-----------------|------|---------------------|
| PP 3.2.2 full text: avoid sedentary behavior, short activity bouts | **T2** | Activity guidance |
| PP 3.2.3 full text: falls risk, sarcopenia, multicomponent exercises | **T2** | Falls prevention guidance |
| PP 3.2.4 full text: weight loss for eGFR ≥30, not for eGFR <30 | **T2** | Weight management guidance |
| Asian BMI threshold: >27.5 kg/m² | **T1** | Population-specific obesity cutoff |
| Obesity paradox in dialysis (higher BMI → better outcomes) | **T1** | Modifies weight loss advice for dialysis |
| "Accumulated weekly activity is critical" — frequency guidance | **T2** | Exercise prescription |
| Figure 22 physical inactivity decision algorithm | **T2** | Implementation workflow |
| Intradialytic exercise → improved hemodialysis adequacy | **T2** | Dialysis-specific exercise evidence |

---

## Reviewer Recommendation

| Action | Details |
|--------|---------|
| **Decision** | **Conditional ACCEPT** — C+F eGFR <30 safety caveat is excellent genuine T1; F obesity BMI threshold useful; but 3 PP labels without text, Asian BMI threshold missing, obesity paradox missing |
| **Tier corrections** | PP labels ×3: T1 → T3; Obesity BMI sentence: T2 → T1; "at baseline": T2 → T3; Pipeline artifact: T2 → NOISE |
| **Missing T1** | Asian BMI >27.5, obesity paradox in dialysis |
| **Missing T2** | PP 3.2.2-3.2.4 text, Figure 22 algorithm, intradialytic exercise evidence |

---

## Completeness Score

| Metric | Value |
|--------|-------|
| **Extraction completeness** | ~20% — 2 genuine clinical sentences (eGFR <30 safety + obesity BMI) from page with 3 PPs, Asian BMI, obesity paradox, and decision algorithm |
| **Tier accuracy** | ~29% (1/4 T1 correct + 1/3 T2 useful = 2/7) |
| **Noise ratio** | ~43% — PP labels x3, pipeline artifact, "at baseline" |
| **Genuine T1 content** | 1 extracted correctly (C+F eGFR <30) + 1 T2 should be T1 (BMI >30) |
| **Prior review** | 0/7 reviewed |
| **Overall quality** | **MODERATE** — C+F multi-channel captures key safety caveat; obesity threshold useful; but PP text and important clinical nuances (Asian BMI, obesity paradox) missing |

---

## Raw PDF Gap Analysis (Pipeline 2 L3-L5 Perspective)

### KB-1 (Dosing/Dietary) Gaps
| Gap | PDF Text | Priority |
|-----|----------|----------|
| PP 3.2.4 weight loss threshold | "advise patients with obesity, diabetes, and CKD to lose weight, particularly patients with eGFR >=30" | HIGH — Specific eGFR threshold for weight loss |
| Activity frequency/distribution | "accumulated weekly activity is more important than intensity; spread throughout the week" | MEDIUM — Exercise dosing guidance |
| Weight loss benefits | "reduced albuminuria, improved blood pressure" in mild-to-moderate CKD | MEDIUM — Treatment response |

### KB-4 (Patient Safety) Gaps
| Gap | PDF Text | Priority |
|-----|----------|----------|
| Obesity paradox in dialysis | "Higher BMI associated with better outcomes among patients treated with dialysis" | CRITICAL — Modifies weight loss advice |
| Falls risk exercise guidance | "aerobic, resistance, and balance training" for high-risk patients | HIGH — Falls prevention |
| eGFR <30 malnutrition concern | Already captured by C+F span | Confirmed |
| Intradialytic exercise safety | "improve hemodialysis adequacy, exercise capacity, depression, quality of life" | MEDIUM — Dialysis exercise evidence |
| Sarcopenia in CKD | "Sarcopenia common in CKD leading to adverse outcomes" | MEDIUM — Muscle-wasting risk |

### KB-16 (Lab Monitoring) Gaps
| Gap | PDF Text | Priority |
|-----|----------|----------|
| Asian BMI threshold | "BMI >27.5 kg/m2 increases risk for adverse outcomes" in Asian populations | HIGH — Population-specific cutoff |
| Obesity BMI threshold | "BMI >30 kg/m2 is an independent risk factor" | Captured by F channel span |
| Weight loss monitoring markers | "albuminuria, blood pressure" as response indicators | MEDIUM |

---

## Post-Review State (2026-02-27)

### Actions Taken
| Action | Count | Details |
|--------|-------|---------|
| **REJECTED** | 5 | Pipeline artifact `<!-- PAGE 74 -->` (out_of_scope); PP labels "Practice Point 3.2.2", "3.2.3", "3.2.4" (out_of_scope x3); "at baseline" fragment (out_of_scope) |
| **CONFIRMED** | 2 | Obesity BMI >30 threshold (genuine clinical threshold for KB-4/KB-16); eGFR <30 malnutrition/muscle-wasting safety caveat (genuine T1 for KB-4) |
| **ADDED** | 7 | PP 3.2.2 full text (sedentary behavior); PP 3.2.3 full text (falls/exercise type); PP 3.2.4 full text (obesity/weight loss eGFR>=30); Asian BMI >27.5; Obesity paradox in dialysis; Intradialytic exercise evidence; Weight loss benefits in mild CKD |

### Final Span Inventory
| Span | Status | KB Targets |
|------|--------|------------|
| Obesity BMI >30 kg/m2 as independent CKD/CVD risk factor | CONFIRMED | KB-4, KB-16 |
| eGFR <30 + dialysis: malnutrition/muscle-wasting concern | CONFIRMED | KB-4, KB-1 |
| PP 3.2.2: avoid sedentary behavior, short bouts, weekly accumulation | ADDED | KB-1, KB-4 |
| PP 3.2.3: falls risk, aerobic + resistance + balance training | ADDED | KB-4 |
| PP 3.2.4: weight loss for obesity+diabetes+CKD with eGFR >=30 | ADDED | KB-1, KB-4 |
| Asian BMI >27.5 kg/m2 as population-specific threshold | ADDED | KB-16 |
| Obesity paradox: higher BMI better outcomes in dialysis | ADDED | KB-4 |
| Intradialytic exercise: hemodialysis adequacy, exercise capacity, QoL | ADDED | KB-4 |
| Weight loss: reduced albuminuria, improved BP in mild-moderate CKD | ADDED | KB-1, KB-16 |
| `<!-- PAGE 74 -->` | REJECTED | N/A |
| "Practice Point 3.2.2" (label only) | REJECTED | N/A |
| "Practice Point 3.2.3" (label only) | REJECTED | N/A |
| "at baseline" (fragment) | REJECTED | N/A |
| "Practice Point 3.2.4" (label only) | REJECTED | N/A |

### Updated Completeness Score
| Metric | Before | After |
|--------|--------|-------|
| **Total spans** | 7 | 14 |
| **Genuine clinical content** | 2 (29%) | 9 (64%) |
| **Noise** | 5 (71%) | 5 rejected (36%) |
| **Review coverage** | 0/7 (0%) | 14/14 (100%) |
| **KB-1 coverage** | None | PP 3.2.2 exercise dosing, PP 3.2.4 weight loss threshold, weight loss benefits |
| **KB-4 coverage** | eGFR <30 safety only | + Falls guidance, obesity paradox, intradialytic exercise, all 3 PP texts |
| **KB-16 coverage** | BMI >30 only | + Asian BMI >27.5, albuminuria/BP monitoring markers |
| **Overall quality** | MODERATE | **GOOD** — All practice points with full text, critical safety nuances (Asian BMI, obesity paradox) captured |

---

## Raw PDF Cross-Check Gap Analysis (2026-02-28)

| # | Gap Text (Exact PDF) | Priority | Rationale |
|---|---------------------|----------|-----------|
| 1 | "In those with CKD, sarcopenia is common and is related to adverse outcomes." | **MODERATE** | Sarcopenia prevalence in CKD — context for PP 3.2.3 exercise guidance. KB-4. |
| 2 | "Benefits of muscle strengthening are often underappreciated. They promote weight maintenance and maintenance of lean body mass while a person is attempting to lose weight." | **MODERATE** | Muscle-strengthening rationale: weight/lean mass maintenance during weight loss. KB-1. |
| 3 | "Hence, recommendations for intensity and type of activity should be individualized based on their age, comorbid conditions, and activity status at baseline also. Depending on the availability of resources, referral to a physical activity specialist to provide guidance about the type and amount of exercise can be considered." | **MODERATE** | Exercise individualization + specialist referral guidance. KB-1 implementation. |
| 4 | "Pooled data from 40 countries (including approximately 5.5 million adults) suggest that higher BMI, waist circumference, and waist-to-height ratio are independent risk factors for kidney function decline and death in individuals who have normal or reduced levels of eGFR." | **MODERATE** | 40-country pooled evidence (5.5M adults): BMI, waist circumference, waist-to-height ratio as CKD risk factors. KB-16. |
| 5 | "Physicians should assess the patients' interest in losing weight and recommend increasing physical activity and appropriate dietary modifications in those who are obese, particularly when the eGFR is ≥30 ml/min per 1.73 m2." | **MODERATE** | Weight loss implementation: assess patient interest, recommend activity + dietary changes. KB-1. |
| 6 | "Often, differentiating unintentional from intentional weight loss can be challenging in those with decline in kidney function." | **MODERATE** | Clinical challenge: unintentional vs intentional weight loss in declining CKD. KB-4. |
| 7 | "CKD patients are often sedentary, which is associated with an increased risk of mortality. In addition, they have limited exercise tolerance and may not be able to do longer periods of exercise." | **MODERATE** | Sedentary → mortality risk; limited exercise tolerance in CKD. KB-4. |
| 8 | "Figure 22. Suggested approach to address physical inactivity and sedentary behavior in CKD: Assess baseline physical activity level. If sedentary, assess fall risk and comorbidity burden — low risk: recommend low-intensity activity and increase intensity as tolerated; high risk: referral to exercise specialists. If physically active for <150 min/wk, recommend to increase to achieve >150 min/wk. If physically active for >150 min/wk, assess and recommend muscle-strengthening activities. If unable to increase activity level due to comorbid conditions, continue current level." | **MODERATE** | Figure 22 decision algorithm: structured approach to CKD physical inactivity. KB-1 implementation. |

**All 8 gaps added via API (all 201).**

---

## Post-Review State (Final — with raw PDF gap fills)

| Metric | Value |
|--------|-------|
| **Total spans (post-review)** | 22 (7 original + 7 agent-added + 8 gap-fill) |
| **Reviewed** | 22/22 (100%) |
| **REJECTED** | 5 |
| **CONFIRMED** | 2 |
| **ADDED (agent)** | 7 |
| **ADDED (gap fill)** | 8 |
| **Total ADDED** | 15 |
| **Pipeline 2 ready** | 17 (2 confirmed + 15 added) |
| **Completeness (post-review)** | ~93% — PP 3.2.2 full text (sedentary behavior, short bouts, weekly accumulation); PP 3.2.3 full text (falls risk, multicomponent exercise); PP 3.2.4 full text (weight loss eGFR ≥30); obesity BMI >30 CKD/CVD risk; Asian BMI >27.5; eGFR <30 malnutrition/muscle-wasting safety; obesity paradox in dialysis; intradialytic exercise evidence; weight loss benefits (albuminuria, BP); sarcopenia in CKD; muscle-strengthening rationale (lean mass maintenance); exercise individualization + specialist referral; 40-country pooled BMI/waist evidence (5.5M adults); physician implementation (assess interest, recommend activity/diet); unintentional vs intentional weight loss challenge; sedentary → mortality + limited exercise tolerance; Figure 22 decision algorithm |
| **Remaining gaps** | Figure 22 citation details (T3); reference numbers (T3) |
| **Review Status** | COMPLETE |
