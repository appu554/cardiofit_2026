# Phase 6: Dose Calculator - Comprehensive Guide

**Module**: Module 3 Clinical Decision Support
**Phase**: Phase 6 - Comprehensive Medication Database
**Component**: Dose Calculator System
**Version**: 1.0
**Date**: 2025-10-24

---

## Overview

The **DoseCalculator** is the cornerstone of Phase 6's medication safety system. It automatically adjusts medication doses based on patient-specific factors including renal function, hepatic function, age, weight, and obesity status.

### Purpose

**Prevent medication errors** through automatic dose adjustment for:
- Renal impairment (68% of medications require renal dosing)
- Hepatic impairment (25% of medications require hepatic dosing)
- Pediatric patients (different pharmacokinetics than adults)
- Geriatric patients (age-related physiologic changes)
- Obesity (altered volume of distribution)

### When Automatic Adjustment Occurs

```
Patient Order Entered
    ↓
DoseCalculator.calculateDose(medication, patientContext, indication)
    ↓
Checks Applied Automatically:
    1. Renal function (calculate CrCl) → adjust if CrCl < 60
    2. Hepatic function (Child-Pugh score) → adjust if Class B or C
    3. Age (if <18 years) → pediatric dosing
    4. Age (if >65 years AND CrCl declining) → geriatric dosing
    5. Weight (if BMI >30) → obesity dosing
    ↓
CalculatedDose Result
    - Adjusted dose
    - Adjusted frequency
    - Rationale (why adjustment made)
    - Monitoring requirements
    - Warnings (if applicable)
```

### Safety Guarantees

1. **Conservative Dosing**: When in doubt, system recommends lower dose + clinical pharmacist consult
2. **Transparent Rationale**: Every dose adjustment includes clinical justification
3. **Multiple Safety Checks**: Renal + hepatic + age + weight all considered
4. **Override Capability**: Clinicians can override with documented justification
5. **Audit Trail**: All calculations logged for safety review

---

## Renal Dosing Adjustments

### Cockcroft-Gault Formula

**Most widely used** formula for medication dosing (preferred over CKD-EPI for drug dosing):

```
CrCl (mL/min) = [(140 - age) × weight(kg)] / (72 × SCr(mg/dL))

If female: Multiply result by 0.85
```

**Why Cockcroft-Gault over CKD-EPI?**
- Most medication dosing studies used Cockcroft-Gault
- FDA package inserts reference Cockcroft-Gault
- Incorporates weight (important for volume of distribution)
- CKD-EPI is better for CKD staging, but Cockcroft-Gault for dosing

### Formula Components

| Component | Description | Units | Clinical Notes |
|-----------|-------------|-------|----------------|
| **Age** | Patient age | years | Fixed physiologic decline: ~1 mL/min/year after age 40 |
| **Weight** | Body weight | kg | Use actual weight for most patients, special rules for obesity |
| **SCr** | Serum creatinine | mg/dL | Must be stable (not acute kidney injury) |
| **Sex** | Male or female | - | Females: 15% lower CrCl (less muscle mass) |

**Important Notes**:
1. **Requires stable SCr**: If creatinine rising (AKI), Cockcroft-Gault is inaccurate
2. **Use actual weight**: NOT ideal body weight (IBW), except in severe obesity
3. **Geriatric patients**: May have low SCr despite poor renal function (low muscle mass)

### CrCl Ranges and Adjustments

| CrCl Range | Renal Function | Typical Adjustment | Clinical Significance |
|------------|---------------|-------------------|---------------------|
| **>80 mL/min** | Normal | No adjustment | Standard dosing |
| **60-80 mL/min** | Mild impairment | May require adjustment | Some medications need adjustment |
| **30-60 mL/min** | Moderate impairment | Dose reduction typically needed | Most renally-cleared drugs need adjustment |
| **10-30 mL/min** | Severe impairment | Significant dose reduction | Many drugs contraindicated or require 50-75% reduction |
| **<10 mL/min** | End-stage renal disease | Often contraindicated or very low dose | Dialysis considerations |

### Example Calculations

#### Example 1: Male Patient, Normal Renal Function

**Patient**:
- Age: 45 years
- Weight: 80 kg
- SCr: 1.0 mg/dL
- Sex: Male

**Calculation**:
```
CrCl = [(140 - 45) × 80] / (72 × 1.0)
     = [95 × 80] / 72
     = 7600 / 72
     = 105.6 mL/min
```

**Result**: Normal renal function → **No dose adjustment needed**

#### Example 2: Female Patient, Moderate Renal Impairment

**Patient**:
- Age: 72 years
- Weight: 60 kg
- SCr: 1.5 mg/dL
- Sex: Female

**Calculation**:
```
CrCl = [(140 - 72) × 60] / (72 × 1.5) × 0.85
     = [68 × 60] / 108 × 0.85
     = 4080 / 108 × 0.85
     = 37.8 × 0.85
     = 32.1 mL/min
```

**Result**: Moderate renal impairment (CrCl 32 mL/min) → **Dose reduction required**

#### Example 3: Elderly Male, Low SCr but Poor Renal Function

**Patient**:
- Age: 85 years
- Weight: 65 kg
- SCr: 0.8 mg/dL (deceptively low due to low muscle mass)
- Sex: Male

**Calculation**:
```
CrCl = [(140 - 85) × 65] / (72 × 0.8)
     = [55 × 65] / 57.6
     = 3575 / 57.6
     = 62.1 mL/min
```

**Result**: Borderline renal function despite "normal" SCr → **May require dose adjustment**

**Clinical Pearl**: Elderly patients with low muscle mass may have low SCr (0.6-0.9 mg/dL) but still have impaired renal function. Always calculate CrCl.

### Medication-Specific Renal Adjustments

#### Piperacillin-Tazobactam

**Renal Clearance**: 68% renally excreted

| CrCl Range | Standard Dose | Adjusted Dose | Rationale |
|------------|--------------|--------------|-----------|
| >40 mL/min | 4.5 g q6h | 4.5 g q6h | No adjustment |
| 20-40 mL/min | 4.5 g q6h | **3.375 g q6h** | 25% dose reduction |
| <20 mL/min | 4.5 g q6h | **2.25 g q6h** | 50% dose reduction |

**Safety Note**: High doses in renal impairment → accumulation → **seizure risk**

```java
// Code example
if (crCl >= 20 && crCl < 40) {
    adjustedDose = "3.375 g";
    rationale = "Moderate renal impairment (CrCl " + crCl + " mL/min). " +
                "Reduce dose by 25% to prevent accumulation and seizure risk.";
}
```

#### Vancomycin

**Renal Clearance**: 90% renally excreted (highly dependent on renal function)

| CrCl Range | Standard Dose | Adjusted Approach | Monitoring |
|------------|--------------|------------------|------------|
| >60 mL/min | 15-20 mg/kg q12h | Standard dosing | Trough 10-20 mcg/mL |
| 40-60 mL/min | 15-20 mg/kg q12h | **q18-24h dosing** | Trough before 3rd dose |
| <40 mL/min | 15-20 mg/kg q12h | **Pharmacist consult required** | Individualized TDM |

**Safety Note**: Vancomycin dosing is **complex** - requires therapeutic drug monitoring (TDM) and pharmacokinetic calculations for CrCl <60.

```java
if (crCl < 60) {
    monitoring = "⚠️ PHARMACIST CONSULT REQUIRED for vancomycin dosing. " +
                 "Requires individualized pharmacokinetic dosing based on " +
                 "volume of distribution, target AUC24, and renal function.";
}
```

#### Levofloxacin

**Renal Clearance**: 87% renally excreted

| CrCl Range | Standard Dose (CAP) | Adjusted Dose | Frequency |
|------------|-------------------|--------------|-----------|
| >50 mL/min | 750 mg | 750 mg | q24h |
| 20-50 mL/min | 750 mg | **500 mg** | q24h (day 1: 750mg loading) |
| 10-20 mL/min | 750 mg | **500 mg** | **q48h** |

```java
if (crCl >= 20 && crCl < 50) {
    adjustedDose = "500 mg";
    frequency = "q24h";
    loadingDose = "750 mg × 1 dose on day 1";
    rationale = "Moderate renal impairment. Loading dose 750mg, then 500mg q24h.";
}
```

#### Gentamicin (Aminoglycoside)

**Renal Clearance**: 95% renally excreted
**Special Consideration**: **Nephrotoxic** + **Ototoxic**

| CrCl Range | Standard Dosing | Adjusted Approach | Monitoring |
|------------|----------------|------------------|------------|
| >60 mL/min | Traditional q8h | 1.5-2 mg/kg q8h | Trough <2 mcg/mL |
| <60 mL/min | Traditional q8h | **Extended interval** (5-7 mg/kg q24-48h) | Peak/trough + SCr daily |

**Modern Approach**: Extended-interval dosing (once-daily high dose) is preferred for CrCl <60
- Less nephrotoxic than traditional dosing
- Simpler monitoring
- Equal or better efficacy

```java
if (crCl < 60) {
    adjustedDose = "5-7 mg/kg";
    frequency = "q24h";
    rationale = "Extended-interval dosing for renal impairment. " +
                "Less nephrotoxic than q8h traditional dosing.";
    monitoring = "Monitor trough <1 mcg/mL, peak 20-30 mcg/mL (drawn 30 min post-infusion).";
}
```

#### Enoxaparin (Low Molecular Weight Heparin)

**Renal Clearance**: 40% renally excreted (anti-Xa activity accumulates)

| CrCl Range | Standard DVT Treatment Dose | Adjusted Dose | Monitoring |
|------------|---------------------------|--------------|------------|
| >30 mL/min | 1 mg/kg q12h | 1 mg/kg q12h | No monitoring usually |
| <30 mL/min | 1 mg/kg q12h | **1 mg/kg q24h** or **30 mg q12h** | Anti-Xa levels (target 0.5-1.0 IU/mL) |

**Safety Note**: Severe renal impairment → increased bleeding risk with enoxaparin

```java
if (crCl < 30) {
    adjustedDose = "1 mg/kg";
    frequency = "q24h";
    alternativeDose = "30 mg q12h (if weight-based dosing not feasible)";
    rationale = "Severe renal impairment (CrCl <30). Anti-Xa activity accumulates. " +
                "Reduce frequency to q24h to prevent accumulation and bleeding.";
    monitoring = "Monitor anti-Xa levels 4h post-dose (target 0.5-1.0 IU/mL).";
}
```

### Hemodialysis Adjustments

**Special Considerations**:
1. **Dialyzability**: Is the drug removed by hemodialysis?
   - Dialyzable: Small molecular weight, water-soluble, not protein-bound
   - Not dialyzable: Large molecular weight, lipophilic, highly protein-bound

2. **Timing**: When to give supplemental dose?
   - **Post-dialysis**: For dialyzable drugs
   - **Interdialytic**: For non-dialyzable drugs

#### Example: Vancomycin in Hemodialysis

**Vancomycin** is **not significantly dialyzed** (large molecule, moderate protein binding)

```
Dosing Strategy:
- Loading dose: 15-20 mg/kg × 1 (usually post-dialysis)
- Maintenance: Redose based on trough levels
- Target trough: 10-20 mcg/mL (check before dialysis)
```

```java
if (patientContext.isOnHemodialysis()) {
    dosing = "15-20 mg/kg loading dose post-dialysis. " +
             "Redose when trough <10 mcg/mL (check pre-dialysis). " +
             "No routine dosing schedule - based on levels.";
}
```

#### Example: Piperacillin-Tazobactam in Hemodialysis

**Piperacillin** is **dialyzable** (small molecule, low protein binding)

```
Dosing Strategy:
- 2.25 g q8h (reduced dose + frequency)
- Supplemental dose: 0.75 g post-dialysis
```

```java
if (patientContext.isOnHemodialysis()) {
    adjustedDose = "2.25 g";
    frequency = "q8h";
    supplementalDose = "0.75 g immediately post-dialysis";
    rationale = "Hemodialysis removes ~30-40% of piperacillin. Give supplemental dose post-HD.";
}
```

### Peritoneal Dialysis Adjustments

**Peritoneal dialysis (PD)** removes less drug than hemodialysis (slower, continuous process)

**General Rule**: Use dosing for **CrCl <10 mL/min**, but may need slight increase from HD dosing

#### Example: Vancomycin in Peritoneal Dialysis

```
Dosing Strategy:
- Loading dose: 15-20 mg/kg IV × 1
- Maintenance (IV): 500-1000 mg every 7-10 days based on levels
- Intraperitoneal (preferred): 15-30 mg/L in each exchange
```

### CRRT (Continuous Renal Replacement Therapy) Adjustments

**CRRT** is continuous dialysis for critically ill patients - removes drugs continuously

**Challenge**: Variable clearance depending on:
1. CRRT modality (CVVH, CVVHD, CVVHDF)
2. Effluent rate (1-3 L/hr typically)
3. Filter characteristics

**General Approach**: Assume CrCl of **30-50 mL/min equivalent** for most drugs

#### Example: Piperacillin-Tazobactam in CRRT

```
Standard CRRT dose: 3.375 g q6-8h
(between normal dose and severe renal impairment dose)
```

```java
if (patientContext.isOnCRRT()) {
    adjustedDose = "3.375 g";
    frequency = "q6-8h";
    rationale = "CRRT provides moderate drug clearance (equivalent to CrCl 30-50 mL/min). " +
                "Use intermediate dose between normal and severe renal impairment.";
    monitoring = "Consider therapeutic drug monitoring if available.";
}
```

---

## Hepatic Dosing Adjustments

### Child-Pugh Scoring

**Gold standard** for assessing hepatic function and guiding medication dosing:

| Component | 1 Point | 2 Points | 3 Points |
|-----------|---------|----------|----------|
| **Bilirubin** (mg/dL) | <2 | 2-3 | >3 |
| **Albumin** (g/dL) | >3.5 | 2.8-3.5 | <2.8 |
| **INR** | <1.7 | 1.7-2.3 | >2.3 |
| **Ascites** | None | Mild (controlled) | Moderate/severe |
| **Encephalopathy** | None | Grade 1-2 | Grade 3-4 |

**Total Score → Child-Pugh Class**:
- **Class A** (5-6 points): **Mild** cirrhosis - well-compensated
- **Class B** (7-9 points): **Moderate** cirrhosis - significant impairment
- **Class C** (10-15 points): **Severe** cirrhosis - decompensated

### Scoring Components Explained

#### 1. Bilirubin (Total)
- **Normal**: 0.3-1.2 mg/dL
- **Significance**: Reflects liver's ability to conjugate and excrete bilirubin
- **Clinical**: Jaundice visible at >3 mg/dL

#### 2. Albumin
- **Normal**: 3.5-5.5 g/dL
- **Significance**: Liver synthesizes albumin - low albumin = poor synthetic function
- **Clinical**: <2.8 g/dL → peripheral edema, ascites

#### 3. INR (International Normalized Ratio)
- **Normal**: 0.9-1.1
- **Significance**: Liver synthesizes clotting factors - elevated INR = poor synthetic function
- **Clinical**: >2.3 → spontaneous bleeding risk

#### 4. Ascites
- **None**: No fluid accumulation
- **Mild**: Controlled with diuretics
- **Moderate/Severe**: Refractory to diuretics, requires paracentesis

#### 5. Hepatic Encephalopathy
- **Grade 0**: No encephalopathy
- **Grade 1-2**: Mild confusion, asterixis, sleep disturbance
- **Grade 3-4**: Severe confusion, stupor, coma

### Child-Pugh Example Calculation

**Patient**:
- Bilirubin: 2.5 mg/dL → 2 points
- Albumin: 3.0 g/dL → 2 points
- INR: 1.9 → 2 points
- Ascites: Mild, controlled with diuretics → 2 points
- Encephalopathy: None → 1 point

**Total**: 9 points → **Child-Pugh Class B** (moderate cirrhosis)

### Medications Requiring Hepatic Dose Adjustment

#### Beta-Blockers (Metoprolol, Propranolol)

**Why**: Extensive first-pass metabolism in liver

| Child-Pugh Class | Metoprolol Standard Dose | Adjusted Dose | Rationale |
|-----------------|-------------------------|--------------|-----------|
| **A** (5-6) | 50 mg PO BID | 50 mg PO BID | No adjustment |
| **B** (7-9) | 50 mg PO BID | **25 mg PO BID** | 50% dose reduction |
| **C** (10-15) | 50 mg PO BID | **Avoid or use with caution** | High risk of bradycardia, hypotension |

```java
if (childPughClass.equals("B")) {
    adjustedDose = "25 mg";
    frequency = "BID";
    rationale = "Moderate cirrhosis (Child-Pugh B). Metoprolol undergoes extensive " +
                "hepatic first-pass metabolism. Reduce dose by 50% to prevent accumulation.";
    monitoring = "Monitor HR, BP. Target HR 60-100 bpm.";
}
```

#### Warfarin

**Why**: Synthesized clotting factors reduced in cirrhosis → baseline elevated INR

| Child-Pugh Class | Standard Dose | Adjusted Dose | INR Target |
|-----------------|--------------|--------------|-----------|
| **A** (5-6) | Typical (5 mg) | Typical | 2.0-3.0 |
| **B** (7-9) | Typical (5 mg) | **Reduce 25-50%** | 2.0-3.0 |
| **C** (10-15) | Typical (5 mg) | **Often contraindicated** | - |

**Safety Note**: Cirrhotic patients have **baseline elevated INR** (reduced clotting factors). Warfarin may cause severe bleeding.

```java
if (childPughClass.equals("B")) {
    adjustedDose = "2.5 mg";  // 50% reduction
    rationale = "Moderate cirrhosis with reduced clotting factor synthesis. " +
                "Baseline INR may be elevated. Start low dose and titrate carefully.";
    monitoring = "⚠️ INCREASED BLEEDING RISK. Monitor INR every 1-2 days initially. " +
                 "Watch for signs of bleeding.";
}

if (childPughClass.equals("C")) {
    contraindicated = true;
    rationale = "Severe cirrhosis (Child-Pugh C). High bleeding risk due to " +
                "thrombocytopenia, reduced clotting factors, and varices. " +
                "Consider alternative anticoagulation strategy (consult hematology).";
}
```

#### NSAIDs (Ibuprofen, Naproxen, Ketorolac)

**Why**: Hepatotoxic + increased bleeding risk (platelet dysfunction) + renal toxicity (cirrhotic patients have poor renal perfusion)

| Child-Pugh Class | NSAID Use | Recommendation |
|-----------------|----------|----------------|
| **A** (5-6) | May use with caution | Monitor liver enzymes |
| **B** (7-9) | **Avoid if possible** | Increased hepatotoxicity + bleeding risk |
| **C** (10-15) | **Contraindicated** | High risk of hepatorenal syndrome, bleeding |

```java
if (childPughClass.equals("B") || childPughClass.equals("C")) {
    contraindicated = true;
    rationale = "Moderate-severe cirrhosis. NSAIDs increase risk of:\n" +
                "1. GI bleeding (varices, platelet dysfunction)\n" +
                "2. Hepatotoxicity (worsening liver function)\n" +
                "3. Hepatorenal syndrome (renal vasoconstriction)\n" +
                "Use acetaminophen ≤2g/day instead.";
    alternatives = Arrays.asList("MED-ACET-084");  // Acetaminophen
}
```

#### Acetaminophen (Paradoxically Safer in Some Cases)

**Standard**: 4g/day maximum
**Cirrhosis**: **2g/day maximum** (reduced hepatic metabolism + glutathione depletion)

```java
if (childPughClass.equals("B") || childPughClass.equals("C")) {
    maxDailyDose = "2 g";  // Reduced from 4g
    rationale = "Cirrhosis reduces glutathione stores (needed for acetaminophen detoxification). " +
                "Limit to 2g/day to prevent hepatotoxicity. Safer than NSAIDs in cirrhosis.";
}
```

### Hepatic Drug Metabolism Pathways

**Phase I** (Oxidation, Reduction, Hydrolysis):
- **CYP450 enzymes**: CYP3A4, CYP2C9, CYP2D6
- **Affected drugs**: Warfarin, beta-blockers, statins, benzodiazepines
- **Cirrhosis impact**: Reduced enzyme activity → slower metabolism → accumulation

**Phase II** (Conjugation):
- **Glucuronidation, sulfation, acetylation**
- **Affected drugs**: Morphine (glucuronidation), acetaminophen (glucuronidation + sulfation)
- **Cirrhosis impact**: Variable - glucuronidation often preserved until advanced cirrhosis

**Clinical Pearl**: Drugs requiring Phase II only (e.g., lorazepam) may be safer in cirrhosis than drugs requiring Phase I (e.g., diazepam).

---

## Pediatric Dosing

### Weight-Based Calculations

**Fundamental Principle**: Children are NOT "small adults"
- Different pharmacokinetics (absorption, distribution, metabolism, excretion)
- Organ immaturity (especially in neonates)
- Rapid growth and changing body composition

**Standard Format**: **mg/kg/day** or **mg/kg/dose**

#### Example: Piperacillin-Tazobactam Pediatric Dosing

**Infant (2-9 months)**: 80 mg/kg/dose q8h
**Child (9 months - 12 years)**: 100 mg/kg/dose q8h
**Maximum**: Never exceed 4.5 g/dose (adult maximum)

**Example Calculation**:
```
Child: 5 years old, 20 kg
Dose: 100 mg/kg/dose q8h
     = 100 mg/kg × 20 kg
     = 2000 mg/dose
     = 2 g/dose q8h

Total daily dose: 2 g × 3 doses = 6 g/day
(Well below 16 g/day pediatric maximum)
```

```java
public CalculatedDose calculatePediatricDose(Medication med, PatientContext patient) {
    double weightKg = patient.getWeight();
    int ageMonths = patient.getAgeMonths();

    // Get appropriate age group dosing
    AgeGroupDose ageGroupDose = med.getPediatricDosing().getAgeGroupDose(ageMonths);

    // Calculate dose in mg
    double dosePerKg = ageGroupDose.getDosePerKg();  // e.g., 100 mg/kg/dose
    double calculatedDose = dosePerKg * weightKg;

    // Apply maximum dose cap
    double maxDose = ageGroupDose.getMaxDoseInMg();  // e.g., 4500 mg
    if (calculatedDose > maxDose) {
        calculatedDose = maxDose;
        warnings.add("⚠️ Calculated dose exceeds maximum. Using adult maximum: " + maxDose + " mg");
    }

    return new CalculatedDose(calculatedDose, ageGroupDose.getFrequency(), rationale);
}
```

### Age Groups and Physiologic Differences

#### Premature Neonate (<37 weeks gestation)

**Pharmacokinetic Differences**:
- **Immature renal function**: GFR 10-20 mL/min/1.73m² (vs 120 adult)
- **Reduced liver metabolism**: CYP450 enzymes not fully developed
- **Altered protein binding**: Lower albumin → more free drug
- **Increased total body water**: 80-85% (vs 60% adult) → larger volume of distribution for hydrophilic drugs

**Dosing Impact**: **Lower doses, longer intervals**

**Example: Gentamicin in Premature Neonate**
```
Standard pediatric dose: 2.5 mg/kg q8h
Premature neonate (<30 weeks): 2.5 mg/kg q24-48h
(Extended interval due to immature renal function)
```

#### Term Neonate (0-1 month, ≥37 weeks gestation)

**Pharmacokinetic Differences**:
- **Improving renal function**: GFR 20-40 mL/min/1.73m²
- **Developing liver enzymes**: Some CYP450 enzymes still immature
- **High total body water**: 75-80%

**Dosing Impact**: **Reduced doses or extended intervals compared to infants**

**Example: Piperacillin-Tazobactam**
```
Contraindicated <2 months: Immature renal function increases accumulation risk
```

#### Infant (1-12 months)

**Pharmacokinetic Differences**:
- **Rapidly improving renal function**: GFR approaching adult values by 6-12 months
- **Maturing liver enzymes**: CYP3A4 reaches adult levels by 6 months
- **Total body water declining**: 70-75%

**Dosing Impact**: **Weight-based dosing typically appropriate**

**Example: Piperacillin-Tazobactam**
```
Infant (2-9 months): 80 mg/kg/dose q8h
(Lower dose than child due to immature renal function)
```

#### Child (1-12 years)

**Pharmacokinetic Differences**:
- **Adult-like renal function**: GFR 90-120 mL/min/1.73m²
- **Mature liver enzymes**: Some children have HIGHER metabolism than adults
- **Total body water**: 60-65%

**Dosing Impact**: **Standard weight-based dosing, may require higher mg/kg doses than adults**

**Example: Piperacillin-Tazobactam**
```
Child: 100 mg/kg/dose q8h
(Higher mg/kg than infant due to higher metabolic rate)
```

#### Adolescent (12-18 years)

**Pharmacokinetic Differences**:
- **Adult-like pharmacokinetics**
- **Variable body composition** (puberty, growth spurts)

**Dosing Impact**: **Transition to adult dosing, but cap at adult maximum**

**Example: Many medications**
```
Use adult dosing if weight >40 kg and Tanner stage ≥4
Otherwise use pediatric weight-based dosing
```

### Maximum Dose Limits

**Cardinal Rule**: **NEVER exceed adult dose**, even if weight-based calculation suggests higher

**Example**:
```
Large adolescent: 16 years old, 90 kg
Piperacillin-Tazobactam: 100 mg/kg/dose q8h
Calculated: 100 mg/kg × 90 kg = 9000 mg = 9 g/dose

❌ WRONG: Give 9 g/dose (exceeds adult maximum)
✅ CORRECT: Give 4.5 g/dose (adult maximum)
```

```java
if (calculatedDose > adultMaximumDose) {
    finalDose = adultMaximumDose;
    rationale = "Weight-based calculation (" + calculatedDose + " mg) exceeds adult maximum. " +
                "Capping at adult dose: " + adultMaximumDose + " mg.";
}
```

### Special Considerations

#### Obesity in Pediatrics

**Use Total Body Weight (TBW)** for most medications
- Children have less adipose tissue than obese adults
- Pharmacokinetic data limited in obese children

**Exceptions** (use IBW or adjusted weight):
- Chemotherapy (toxicity concerns)
- Some antibiotics (consult pharmacist)

#### Neonatal Seizures with Beta-Lactams

**Risk**: High doses of beta-lactams (especially imipenem, but also piperacillin) can cause seizures in neonates
**Mechanism**: GABA antagonism in immature brain

**Mitigation**: Use lower end of dosing range, avoid in neonates if alternative available

---

## Geriatric Dosing

### Age-Related Physiological Changes

| System | Change with Aging | Medication Impact |
|--------|------------------|-------------------|
| **Renal** | ↓ GFR ~1 mL/min/year after age 40 | Reduced clearance of renally-excreted drugs |
| **Hepatic** | ↓ Liver mass 20-40%, ↓ blood flow | Reduced first-pass metabolism |
| **Cardiac** | ↓ Cardiac output 1%/year | Reduced drug delivery to tissues |
| **Body Composition** | ↑ Fat 20-40%, ↓ Lean mass, ↓ Total body water | Altered volume of distribution |
| **Gastric** | ↓ Acid secretion, ↓ motility | Altered absorption (usually minor) |
| **Protein Binding** | ↓ Albumin in frail elderly | ↑ Free drug (active) concentration |

### Reduced Renal Function

**Key Point**: **Creatinine may be "normal" (0.8-1.2 mg/dL) despite poor renal function**

**Why**: Elderly have less muscle mass → produce less creatinine → lower serum creatinine despite reduced GFR

**Example**:
```
Patient A: 30 years old, 80 kg, SCr 1.0 mg/dL
CrCl = [(140-30) × 80] / (72 × 1.0) = 122 mL/min (excellent)

Patient B: 80 years old, 65 kg, SCr 1.0 mg/dL (same as Patient A!)
CrCl = [(140-80) × 65] / (72 × 1.0) = 54 mL/min (moderate impairment)
```

**Clinical Rule**: **ALWAYS calculate CrCl in elderly**, even if SCr appears "normal"

### Reduced Hepatic Metabolism

**Hepatic Blood Flow**: Declines ~1% per year after age 40
- Reduced first-pass metabolism for high-extraction drugs (beta-blockers, opioids)

**CYP450 Activity**: Variable decline
- CYP2D6: Minimal change
- CYP3A4: 20-30% decline in frail elderly

**Clinical Impact**: Drugs with extensive hepatic metabolism accumulate
- Beta-blockers (metoprolol, propranolol)
- Opioids (morphine, fentanyl)
- Benzodiazepines (diazepam - Phase I metabolism)

### Altered Volume of Distribution

**Increased Fat**: 20-40% increase in adipose tissue
- **Lipophilic drugs** (diazepam, fentanyl) have larger volume of distribution → longer half-life

**Decreased Lean Mass**: 20-30% decline
- **Hydrophilic drugs** (digoxin, gentamicin) have smaller volume of distribution → higher peak levels

**Decreased Total Body Water**: 10-15% decline
- **Water-soluble drugs** have higher concentrations

**Example: Digoxin**
```
Young adult: Volume of distribution ~7 L/kg
Elderly: Volume of distribution ~4-5 L/kg
→ Same dose produces 40-50% higher peak levels in elderly
→ Increased toxicity risk (arrhythmias)
```

### "Start Low, Go Slow" Principle

**Philosophy**: Begin with lowest effective dose, titrate slowly based on response

**Why**:
1. Reduced clearance → slower elimination → accumulation risk
2. Increased sensitivity to adverse effects (CNS effects, hypotension, bleeding)
3. Polypharmacy increases interaction risk
4. Frailty reduces physiologic reserve

**Example: Metoprolol in Elderly**
```
Young adult: Start 50 mg PO BID
Elderly: Start 25 mg PO BID (50% reduction)
Titrate: Increase by 25 mg every 1-2 weeks based on HR, BP, tolerance
```

```java
if (patient.getAge() >= 65 && patient.isFrail()) {
    startingDose = standardDose * 0.5;  // 50% reduction
    rationale = "Geriatric patient (age " + age + "). Start low dose and titrate slowly. " +
                "Reduced renal clearance, altered volume of distribution, increased sensitivity.";
    titrationGuidance = "Increase by 50% increments every 1-2 weeks based on clinical response.";
}
```

### Beers Criteria for Potentially Inappropriate Medications

**Beers Criteria**: Evidence-based list of medications to **avoid or use with caution in elderly** (≥65 years)

**Categories**:
1. **Avoid in all elderly**: High risk, safer alternatives available
2. **Avoid in certain conditions**: Disease-specific risks
3. **Use with caution**: Increased monitoring needed

#### Avoid in All Elderly

| Medication Class | Examples | Risk | Alternative |
|-----------------|----------|------|-------------|
| **First-generation antihistamines** | Diphenhydramine, hydroxyzine | Anticholinergic (confusion, falls, urinary retention) | Cetirizine, loratadine |
| **Benzodiazepines (long-acting)** | Diazepam, flurazepam | Prolonged sedation, falls, cognitive impairment | Lorazepam (short-acting), non-benzo alternatives |
| **Tricyclic antidepressants** | Amitriptyline, doxepin | Anticholinergic, orthostatic hypotension | SSRIs (sertraline, citalopram) |
| **NSAIDs (chronic use)** | Ibuprofen, naproxen | GI bleeding, renal toxicity, CV events | Acetaminophen, topical NSAIDs |

#### Avoid in Certain Conditions

| Condition | Medication to Avoid | Risk |
|-----------|-------------------|------|
| **Heart Failure** | NSAIDs, thiazolidinediones | Fluid retention, worsening HF |
| **Dementia** | Anticholinergics | Worsening cognitive impairment |
| **Falls/Fractures** | Benzodiazepines, opioids | Sedation, increased fall risk |
| **Chronic Kidney Disease** | NSAIDs | Acute kidney injury |

```java
if (patient.getAge() >= 65) {
    BeersListChecker checker = new BeersListChecker();
    BeersResult result = checker.checkMedication(medication, patient);

    if (result.isBeersListMedication()) {
        warnings.add("⚠️ BEERS CRITERIA: " + result.getWarning());
        warnings.add("Consider alternative: " + result.getSuggestedAlternatives());
    }
}
```

---

## Obesity Dosing

### Body Weight Types

#### Total Body Weight (TBW)
**Definition**: Actual measured body weight
**Use Cases**:
- Hydrophilic drugs (distribute to total body water)
- Most antibiotics
- Heparin (unfractionated)

#### Ideal Body Weight (IBW)
**Calculation**:
```
Male: IBW = 50 kg + 2.3 kg per inch over 5 feet
Female: IBW = 45.5 kg + 2.3 kg per inch over 5 feet
```

**Example**:
```
Female, 5'6" (66 inches)
Height over 5 feet: 66 - 60 = 6 inches
IBW = 45.5 + (2.3 × 6) = 45.5 + 13.8 = 59.3 kg
```

**Use Cases**:
- Aminoglycosides (gentamicin, tobramycin) - lipophobic, don't distribute to fat
- Some chemotherapy agents

#### Adjusted Body Weight (AdjBW)
**Calculation**:
```
AdjBW = IBW + 0.4 × (TBW - IBW)
```

**Example**:
```
Female, 5'6", 100 kg
IBW = 59.3 kg (calculated above)
AdjBW = 59.3 + 0.4 × (100 - 59.3)
      = 59.3 + 0.4 × 40.7
      = 59.3 + 16.3
      = 75.6 kg
```

**Use Cases**:
- Medications with intermediate lipophilicity
- Enoxaparin (LMWH) in severe obesity (BMI >40)

### When to Use Which Weight

| Medication | Weight Type | Rationale |
|-----------|------------|-----------|
| **Piperacillin-Tazobactam** | TBW | Hydrophilic, distributes to total body water |
| **Vancomycin** | TBW | Hydrophilic |
| **Gentamicin** | IBW (or AdjBW if BMI >40) | Lipophobic, doesn't distribute to fat tissue |
| **Enoxaparin** | TBW (cap at 150 kg) | Primarily TBW, but cap dose for bleeding risk |
| **Heparin (unfraction)** | TBW | Distributes to total body water |
| **Propofol** | TBW | Lipophilic, but use TBW for practical dosing |

### Lipophilic vs Hydrophilic Drugs

**Hydrophilic** (water-soluble):
- Do NOT distribute into fat tissue
- Volume of distribution based on lean body mass + water
- **Use TBW or IBW** depending on obesity severity
- Examples: Aminoglycosides, beta-lactams, vancomycin

**Lipophilic** (fat-soluble):
- DO distribute into fat tissue
- Larger volume of distribution in obese patients
- **Use TBW**
- Examples: Benzodiazepines, fentanyl, propofol

```java
public double getAppropriateWeight(Medication med, PatientContext patient) {
    double tbw = patient.getWeight();
    double ibw = calculateIBW(patient.getHeight(), patient.getSex());
    double bmi = calculateBMI(tbw, patient.getHeight());

    String weightType = med.getObesityDosing().getWeightType();

    if (weightType.equals("TBW")) {
        return tbw;
    } else if (weightType.equals("IBW")) {
        return ibw;
    } else if (weightType.equals("AdjBW")) {
        return ibw + 0.4 * (tbw - ibw);
    }

    // Default: use TBW
    return tbw;
}
```

### Examples

#### Example 1: Piperacillin-Tazobactam in Obesity

**Patient**:
- Sex: Female
- Height: 5'4" (64 inches)
- Weight: 120 kg
- BMI: 45.6 (Class III obesity)

**Calculation**:
```
IBW = 45.5 + 2.3 × (64-60) = 45.5 + 9.2 = 54.7 kg
TBW = 120 kg

Piperacillin-Tazobactam: Use TBW (hydrophilic)
Dose: 4.5 g q6h (no weight-based calculation, fixed dose)

Max daily dose: 18 g/day (not exceeded)
```

**Rationale**: Piperacillin is hydrophilic and distributes to total body water (which is increased in obesity). Use standard dose.

#### Example 2: Gentamicin in Obesity

**Patient**: Same as above (120 kg, IBW 54.7 kg, BMI 45.6)

**Calculation**:
```
Traditional dosing: 1.5 mg/kg (for q8h) or 5-7 mg/kg (extended interval)

❌ WRONG: Use TBW
Dose = 7 mg/kg × 120 kg = 840 mg (toxic!)

✅ CORRECT: Use AdjBW (severe obesity BMI >40)
AdjBW = 54.7 + 0.4 × (120 - 54.7) = 54.7 + 26.1 = 80.8 kg
Dose = 7 mg/kg × 80.8 kg = 566 mg q24h

Monitor peak/trough levels to ensure appropriate dosing.
```

**Rationale**: Gentamicin is lipophobic (doesn't distribute to fat). Using TBW would overdose. Use AdjBW for severe obesity.

```java
if (medication.getName().contains("gentamicin") && patient.getBMI() >= 40) {
    double ibw = calculateIBW(patient);
    double tbw = patient.getWeight();
    double adjBW = ibw + 0.4 * (tbw - ibw);

    adjustedDose = dosePerKg * adjBW;
    rationale = "Gentamicin dosing in severe obesity (BMI " + bmi + "). " +
                "Use adjusted body weight (" + adjBW + " kg) instead of TBW (" + tbw + " kg). " +
                "Gentamicin is lipophobic and does not distribute to adipose tissue.";
    monitoring = "⚠️ MONITOR PEAK AND TROUGH LEVELS. Target peak 20-30 mcg/mL, trough <1 mcg/mL.";
}
```

#### Example 3: Enoxaparin in Obesity

**Patient**: Same as above (120 kg, IBW 54.7 kg)

**Calculation**:
```
Standard DVT treatment: 1 mg/kg q12h

Option 1: Use TBW (capped at 150 kg for safety)
Dose = 1 mg/kg × 120 kg = 120 mg q12h

Option 2: Cap total daily dose at 300 mg/day
Dose = 150 mg q12h (total 300 mg/day)

Recommendation: Use Option 1 (120 mg q12h) with anti-Xa monitoring
```

**Rationale**: Enoxaparin distributes to total body water, but very high doses increase bleeding risk. Use TBW but monitor anti-Xa levels.

```java
if (medication.getName().contains("enoxaparin") && patient.getWeight() > 150) {
    warnings.add("⚠️ OBESITY DOSING CONCERN: Weight exceeds 150 kg. " +
                 "Enoxaparin dosing data limited above 150 kg. " +
                 "Consider using weight of 150 kg or monitoring anti-Xa levels.");
    monitoring = "Monitor anti-Xa levels 4h post-dose (target 0.6-1.0 IU/mL for q12h dosing).";
}
```

---

## Code Examples

### Example 1: Complete Dose Calculation Workflow

```java
public class DoseCalculator {

    /**
     * Main entry point: Calculate dose for medication + patient + indication
     */
    public CalculatedDose calculateDose(
            Medication medication,
            PatientContext patient,
            String indication
    ) {
        // Start with standard dose for indication
        DoseSpecification standardDose = getStandardDose(medication, indication);

        // Apply adjustments in order of precedence
        CalculatedDose result = new CalculatedDose(standardDose);

        // 1. Pediatric dosing (if age <18)
        if (patient.getAge() < 18) {
            result = applyPediatricDosing(medication, patient, result);
        }

        // 2. Renal adjustment (if CrCl <60)
        double crCl = calculateCrCl(patient);
        if (crCl < 60) {
            result = applyRenalAdjustment(medication, patient, crCl, result);
        }

        // 3. Hepatic adjustment (if Child-Pugh B or C)
        String childPugh = calculateChildPugh(patient);
        if (childPugh.equals("B") || childPugh.equals("C")) {
            result = applyHepaticAdjustment(medication, patient, childPugh, result);
        }

        // 4. Geriatric considerations (if age ≥65)
        if (patient.getAge() >= 65) {
            result = applyGeriatricConsiderations(medication, patient, result);
        }

        // 5. Obesity dosing (if BMI ≥30)
        if (patient.getBMI() >= 30) {
            result = applyObesityDosing(medication, patient, result);
        }

        // Add monitoring requirements
        result.setMonitoring(medication.getMonitoring());

        return result;
    }

    /**
     * Calculate creatinine clearance using Cockcroft-Gault
     */
    private double calculateCrCl(PatientContext patient) {
        int age = patient.getAge();
        double weight = patient.getWeight();  // kg
        double scr = patient.getCreatinine();  // mg/dL
        String sex = patient.getSex();

        if (scr <= 0 || weight <= 0) {
            logger.warn("Invalid patient data for CrCl calculation. Returning default 60.0");
            return 60.0;  // Default assumption
        }

        // Cockcroft-Gault formula
        double crCl = ((140.0 - age) * weight) / (72.0 * scr);

        // Female adjustment
        if (sex.equalsIgnoreCase("F") || sex.equalsIgnoreCase("Female")) {
            crCl *= 0.85;
        }

        logger.info("Calculated CrCl: {} mL/min (age={}, weight={}, SCr={}, sex={})",
                crCl, age, weight, scr, sex);

        return crCl;
    }

    /**
     * Apply renal dose adjustment based on CrCl
     */
    private CalculatedDose applyRenalAdjustment(
            Medication med,
            PatientContext patient,
            double crCl,
            CalculatedDose current
    ) {
        // Get renal adjustments from medication
        List<RenalAdjustment> adjustments = med.getAdultDosing().getRenalAdjustment();

        // Find applicable adjustment based on CrCl
        for (RenalAdjustment adj : adjustments) {
            if (isInCrClRange(crCl, adj.getCrClRange())) {
                current.setAdjustedDose(adj.getDose());
                current.setAdjustedFrequency(adj.getFrequency());
                current.addRationale("Renal adjustment for CrCl " + String.format("%.1f", crCl) + " mL/min: " + adj.getRationale());
                current.addMonitoring(adj.getMonitoring());

                // Check for contraindication
                if (adj.getContraindicatedIfBelow() && crCl < extractLowerBound(adj.getCrClRange())) {
                    current.setContraindicated(true);
                    current.addWarning("⚠️ CONTRAINDICATED: CrCl too low for safe use.");
                }

                return current;
            }
        }

        return current;  // No adjustment needed
    }

    /**
     * Check if CrCl falls within range (e.g., "30-60", "<10")
     */
    private boolean isInCrClRange(double crCl, String range) {
        if (range.startsWith("<")) {
            double upperBound = Double.parseDouble(range.substring(1));
            return crCl < upperBound;
        } else if (range.contains("-")) {
            String[] parts = range.split("-");
            double lower = Double.parseDouble(parts[0]);
            double upper = Double.parseDouble(parts[1]);
            return crCl >= lower && crCl < upper;
        }
        return false;
    }
}
```

### Example 2: Pediatric Dose Calculator

```java
public class PediatricDoseCalculator {

    public CalculatedDose calculatePediatricDose(
            Medication med,
            PatientContext patient
    ) {
        double weightKg = patient.getWeight();
        int ageMonths = patient.getAgeMonths();

        PediatricDosing pedDosing = med.getPediatricDosing();
        if (pedDosing == null) {
            return new CalculatedDose(null, "No pediatric dosing available. Consult pharmacist.");
        }

        // Get age-appropriate dosing
        AgeGroupDose ageGroupDose = getAgeGroupDose(pedDosing, ageMonths);
        if (ageGroupDose == null) {
            return new CalculatedDose(null, "Age " + ageMonths + " months: No established pediatric dose. Consult pediatric pharmacist.");
        }

        // Calculate weight-based dose
        double dosePerKg = ageGroupDose.getDosePerKg();  // mg/kg/dose
        double calculatedDoseMg = dosePerKg * weightKg;

        // Check maximum dose cap
        double maxDoseMg = ageGroupDose.getMaxDoseInMg();
        if (calculatedDoseMg > maxDoseMg) {
            logger.warn("Calculated dose {} mg exceeds max {} mg. Using max.", calculatedDoseMg, maxDoseMg);
            calculatedDoseMg = maxDoseMg;
        }

        // Build result
        CalculatedDose result = new CalculatedDose();
        result.setDose(String.format("%.0f mg", calculatedDoseMg));
        result.setFrequency(ageGroupDose.getFrequency());
        result.setRationale(String.format(
                "Pediatric dosing for %s (%d months, %.1f kg): %.0f mg/kg/dose = %.0f mg %s",
                ageGroupDose.getAgeGroup(),
                ageMonths,
                weightKg,
                dosePerKg,
                calculatedDoseMg,
                ageGroupDose.getFrequency()
        ));

        // Add contraindications
        if (ageGroupDose.getContraindications() != null) {
            result.addWarning("⚠️ " + ageGroupDose.getContraindications());
        }

        return result;
    }

    private AgeGroupDose getAgeGroupDose(PediatricDosing pedDosing, int ageMonths) {
        for (AgeGroupDose agd : pedDosing.getByAgeGroup()) {
            if (isInAgeRange(ageMonths, agd.getAgeRange())) {
                return agd;
            }
        }
        return null;
    }

    private boolean isInAgeRange(int ageMonths, String ageRange) {
        // Parse age ranges like "0-1 month", "1-12 months", "1-12 years"
        // Implementation depends on format...
        return true;  // Simplified
    }
}
```

---

## Edge Cases

### 1. Extremes of Weight

#### Very Low Weight (<3 kg)

**Scenarios**:
- Premature neonates
- Severely malnourished patients

**Challenges**:
- Volume of distribution calculations unreliable
- Rounding errors significant (0.1 mL difference = large dose change)
- Limited pharmacokinetic data

**Strategy**:
```java
if (patient.getWeight() < 3.0) {
    warnings.add("⚠️ VERY LOW WEIGHT (<3 kg): Limited dosing data. " +
                 "Consult neonatal pharmacist for precise dosing.");
    requirePharmacistApproval = true;
}
```

#### Very High Weight (>200 kg)

**Scenarios**:
- Severe obesity (Class III, BMI >40)

**Challenges**:
- Limited dosing data above 150-200 kg
- Risk of toxicity vs underdosing unclear
- Volume of distribution assumptions may not hold

**Strategy**:
```java
if (patient.getWeight() > 200) {
    warnings.add("⚠️ VERY HIGH WEIGHT (>200 kg): Limited dosing data. " +
                 "Consider capping weight at 150-200 kg for dosing calculation. " +
                 "Monitor drug levels if available.");

    // Cap weight for safety
    if (medication.getObesityDosing().getWeightType().equals("TBW")) {
        effectiveWeight = Math.min(patient.getWeight(), 200);
    }
}
```

### 2. Extremes of Age

#### Very Young (<1 week, especially premature)

**Challenges**:
- Immature renal function (GFR 10-20 mL/min/1.73m²)
- Immature hepatic enzymes
- Different protein binding
- Increased permeability of blood-brain barrier

**Strategy**:
```java
if (patient.getAgeWeeks() < 1) {
    warnings.add("⚠️ NEONATE <1 WEEK OLD: Organ immaturity significantly affects pharmacokinetics. " +
                 "Use lowest effective dose with extended dosing intervals. " +
                 "Neonatal pharmacist consultation required.");
}
```

#### Very Old (>100 years)

**Challenges**:
- Extreme renal decline (CrCl may be <30 despite "normal" SCr)
- Frailty, polypharmacy
- Limited data in centenarians

**Strategy**:
```java
if (patient.getAge() > 100) {
    warnings.add("⚠️ EXTREME AGE (>100 years): Assume significantly reduced renal and hepatic function. " +
                 "Start with 25-50% of standard dose. Titrate slowly based on response.");
    startingDose = standardDose * 0.25;  // Very conservative
}
```

### 3. Anuric Patients (CrCl <5 mL/min)

**Scenarios**:
- End-stage renal disease not yet on dialysis
- Acute kidney injury with anuria

**Challenges**:
- Essentially no renal clearance
- Accumulation risk for renally-cleared drugs
- Many medications contraindicated

**Strategy**:
```java
if (crCl < 5) {
    // Check if medication is renally cleared
    if (medication.getRenalClearancePercent() > 50) {
        warnings.add("❌ CONTRAINDICATED or USE WITH EXTREME CAUTION: " +
                     "Anuric patient (CrCl <5 mL/min). Medication is " +
                     medication.getRenalClearancePercent() + "% renally cleared. " +
                     "High accumulation and toxicity risk. Consider alternative.");
    }
}
```

### 4. Combined Renal + Hepatic Impairment

**Challenges**:
- Additive reduction in clearance
- Very limited dosing data
- High toxicity risk

**Strategy**:
```java
if (crCl < 30 && (childPugh.equals("B") || childPugh.equals("C"))) {
    warnings.add("⚠️ COMBINED RENAL AND HEPATIC IMPAIRMENT: " +
                 "Severely reduced drug clearance. Use lowest possible dose. " +
                 "Pharmacist consultation and therapeutic drug monitoring (TDM) required.");

    // Apply both adjustments conservatively
    doseAfterRenalAdj = applyRenalAdjustment(...);
    doseAfterHepaticAdj = applyHepaticAdjustment(doseAfterRenalAdj);

    // Further reduce by 25-50% for safety
    finalDose = doseAfterHepaticAdj * 0.75;
}
```

### 5. Pregnancy Dose Adjustments

**Physiologic Changes in Pregnancy**:
- Increased blood volume (+40-50%)
- Increased renal blood flow (+50-80%) → increased GFR → faster clearance
- Increased hepatic metabolism (some CYP enzymes)
- Altered protein binding (lower albumin)

**Dosing Impact**:
- Some medications require **higher doses** in pregnancy (e.g., enoxaparin, some antibiotics)
- Others require **lower doses** (teratogenic risk)

**Strategy**:
```java
if (patient.isPregnant()) {
    // Check pregnancy category
    String pregCategory = medication.getSafety().getPregnancyCategory();

    if (pregCategory.equals("X")) {
        return new CalculatedDose(null, "❌ CONTRAINDICATED IN PREGNANCY (Category X)");
    }

    // Some medications need dose increase due to increased clearance
    if (medication.getName().equals("Enoxaparin")) {
        warnings.add("ℹ️ PREGNANCY: May require higher dose due to increased volume of distribution. " +
                     "Monitor anti-Xa levels.");
    }
}
```

---

## Clinical Decision Support

### When to Override Calculated Dose

**Calculated dose is a RECOMMENDATION, not absolute**. Clinicians may override when:

1. **Clinical Indication Requires Higher Dose**
   - Severe infection (meningitis, endocarditis) may require maximum doses
   - Resistant organisms may require higher doses + prolonged infusion

2. **Patient-Specific Factors**
   - Augmented renal clearance (trauma, burns) may require HIGHER doses than calculated
   - Therapeutic drug monitoring (TDM) shows subtherapeutic levels

3. **Alternative Evidence**
   - Newer literature suggests different dosing
   - Hospital-specific protocols differ from database

**System Requirements**:
```java
if (clinician.overridesDose()) {
    auditLog.record(
        clinicianId,
        patientId,
        medicationId,
        calculatedDose,
        overriddenDose,
        overrideReason,  // Required field
        timestamp
    );

    alerts.add("⚠️ DOSE OVERRIDE: Clinician overrode calculated dose. Reason: " + overrideReason);
}
```

### Mandatory Monitoring Requirements

**High-Risk Medications** require specific monitoring:

#### Warfarin
```
Monitoring:
- INR: Every 1-2 days until stable, then weekly, then monthly
- Target INR: 2.0-3.0 (most indications), 2.5-3.5 (mechanical valves)
- Signs of bleeding: Daily assessment
```

#### Vancomycin
```
Monitoring:
- Trough level: Before 4th dose (steady state)
- Target trough: 10-20 mcg/mL (depends on indication)
- Renal function (SCr): Daily
```

#### Gentamicin
```
Monitoring:
- Peak: 30 min after 3rd dose (target 20-30 mcg/mL for q24h dosing)
- Trough: Before 3rd dose (target <1 mcg/mL)
- Renal function (SCr): Daily
```

#### Digoxin
```
Monitoring:
- Digoxin level: 6-8 hours post-dose (target 0.5-2.0 ng/mL)
- Potassium: Daily (hypokalemia increases toxicity)
- Renal function: Every 2-3 days
- ECG: Monitor for arrhythmias
```

### Therapeutic Drug Monitoring (TDM) Recommendations

**When to Use TDM**:
1. **Narrow therapeutic index** (small difference between effective and toxic dose)
2. **Variable pharmacokinetics** (wide inter-patient variability)
3. **Uncertain clinical response** (hard to assess efficacy clinically)

**Medications Requiring TDM**:
- Vancomycin
- Aminoglycosides (gentamicin, tobramycin)
- Digoxin
- Phenytoin
- Theophylline
- Immunosuppressants (tacrolimus, cyclosporine)

```java
if (medication.getMonitoring().getTherapeuticRange() != null) {
    TherapeuticRange tr = medication.getMonitoring().getTherapeuticRange();

    alerts.add("📊 THERAPEUTIC DRUG MONITORING REQUIRED:");
    alerts.add("   Drug: " + tr.getDrug());
    alerts.add("   Target range: " + tr.getTarget() + " " + tr.getUnit());
    alerts.add("   Timing: " + tr.getTimingInstructions());
    alerts.add("   Frequency: " + medication.getMonitoring().getFrequency());
}
```

---

**Version**: 1.0
**Last Updated**: 2025-10-24
**Next Review**: After clinical validation
**Maintained By**: CardioFit Platform - Module 3 CDS Team

---

*Generated with Claude Code - CardioFit Technical Documentation*
