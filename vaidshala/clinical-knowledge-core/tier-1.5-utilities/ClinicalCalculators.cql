/*
 * ClinicalCalculators CQL Library
 * ============================================================
 * Tier 1.5 Clinical Utilities - Calculator Integration
 *
 * Purpose: Provides CQL interfaces to KB-8 Calculator outputs.
 *          These are WRAPPERS around pre-computed values, NOT calculations.
 *
 * ARCHITECTURE NOTE (CTO/CMO Directive):
 * KB-8 Calculator Service performs actual calculations at snapshot build time.
 * This library provides CQL definitions that consume those pre-computed values
 * from the ClinicalExecutionContext.CalculatorSnapshot.
 *
 * Why Wrappers?
 * - CQL cannot perform natural logarithms (ln) - required for MELD, eGFR
 * - CQL cannot perform exponentials (e^x) - required for ASCVD
 * - CQL cannot perform square roots - required for BSA, some risk scores
 * - Pre-computation ensures consistent results across all engines
 *
 * Governance: Tier-1.5 review (Clinical + Technical)
 * ============================================================
 */
library ClinicalCalculators version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.4.000' called FHIRHelpers
include IntervalHelpers version '1.0.0' called Intervals
include ObservationHelpers version '1.0.0' called Observations

// ============================================================================
// VALUE SET REFERENCES
// ============================================================================

valueset "Serum Creatinine": 'http://cardiofit.io/fhir/ValueSet/serum-creatinine'
valueset "Body Weight": 'http://cardiofit.io/fhir/ValueSet/body-weight'
valueset "Body Height": 'http://cardiofit.io/fhir/ValueSet/body-height'
valueset "Bilirubin Total": 'http://cardiofit.io/fhir/ValueSet/bilirubin-total'
valueset "INR Coagulation": 'http://cardiofit.io/fhir/ValueSet/inr'
valueset "Albumin Serum": 'http://cardiofit.io/fhir/ValueSet/albumin'
valueset "Platelet Count": 'http://cardiofit.io/fhir/ValueSet/platelet-count'
valueset "Total Cholesterol": 'http://cardiofit.io/fhir/ValueSet/total-cholesterol'
valueset "HDL Cholesterol": 'http://cardiofit.io/fhir/ValueSet/hdl-cholesterol'
valueset "LDL Cholesterol": 'http://cardiofit.io/fhir/ValueSet/ldl-cholesterol'
valueset "Triglycerides": 'http://cardiofit.io/fhir/ValueSet/triglycerides'
valueset "Blood Glucose": 'http://cardiofit.io/fhir/ValueSet/blood-glucose'
valueset "HbA1c Observations": 'http://cardiofit.io/fhir/ValueSet/hba1c'
valueset "Respiratory Rate": 'http://cardiofit.io/fhir/ValueSet/respiratory-rate'
valueset "Glasgow Coma Scale": 'http://cardiofit.io/fhir/ValueSet/gcs'
valueset "Systolic Blood Pressure": 'http://cardiofit.io/fhir/ValueSet/systolic-bp'
valueset "PaO2 FiO2 Ratio": 'http://cardiofit.io/fhir/ValueSet/pf-ratio'
valueset "Mean Arterial Pressure": 'http://cardiofit.io/fhir/ValueSet/map'

// ============================================================================
// CONTEXT
// ============================================================================
context Patient

// ============================================================================
// SECTION 1: RAW OBSERVATION VALUES (Input to Calculators)
// ============================================================================

/*
 * These definitions retrieve raw lab values from FHIR Observations.
 * They serve as inputs for simple CQL calculations (BMI)
 * and as documentation for what KB-8 uses for complex calculations.
 */

// --- Serum Creatinine ---
define "Most Recent Serum Creatinine":
  Observations.MostRecentObservation(
    [Observation: "Serum Creatinine"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent Serum Creatinine Value":
  Observations.ObservationQuantityValue("Most Recent Serum Creatinine")

// --- Body Weight ---
define "Most Recent Weight":
  Observations.MostRecentObservation(
    [Observation: "Body Weight"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent Weight Kg":
  Observations.ObservationQuantityValue("Most Recent Weight")

// --- Body Height ---
define "Most Recent Height":
  Observations.MostRecentObservation(
    [Observation: "Body Height"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent Height Cm":
  Observations.ObservationQuantityValue("Most Recent Height")

// --- Bilirubin ---
define "Most Recent Bilirubin":
  Observations.MostRecentObservation(
    [Observation: "Bilirubin Total"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent Bilirubin Value":
  Observations.ObservationQuantityValue("Most Recent Bilirubin")

// --- INR ---
define "Most Recent INR":
  Observations.MostRecentObservation(
    [Observation: "INR Coagulation"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent INR Value":
  Observations.ObservationQuantityValue("Most Recent INR")

// --- Albumin ---
define "Most Recent Albumin":
  Observations.MostRecentObservation(
    [Observation: "Albumin Serum"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent Albumin Value":
  Observations.ObservationQuantityValue("Most Recent Albumin")

// --- Platelets ---
define "Most Recent Platelet Count":
  Observations.MostRecentObservation(
    [Observation: "Platelet Count"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent Platelet Value":
  Observations.ObservationQuantityValue("Most Recent Platelet Count")

// ============================================================================
// SECTION 2: ESTIMATED GFR (eGFR) - From KB-8 Snapshot
// ============================================================================

/*
 * eGFR Calculation: CKD-EPI 2021 Formula (Race-Free)
 *
 * Formula uses natural logarithm - CANNOT be computed in CQL.
 * Pre-computed by KB-8 Calculator Service at snapshot time.
 *
 * Inputs (from snapshot):
 * - Serum creatinine (mg/dL)
 * - Age (years)
 * - Sex (male/female - no longer race-adjusted per 2021 guidelines)
 *
 * CLINICAL NOTE: 2021 CKD-EPI removed race coefficient.
 */

/*
 * eGFR Value from Snapshot (pre-computed by KB-8)
 * Returns: Decimal representing mL/min/1.73m²
 *
 * USAGE: Access via ClinicalExecutionContext.CalculatorSnapshot.EGFR
 * Runtime injects actual value; this placeholder returns null.
 */
define "eGFR Value":
  null as Decimal

/*
 * CKD Stage Classification based on eGFR (KDIGO 2012)
 * This CAN be computed in CQL once we have the eGFR value.
 */
define "CKD Stage from eGFR":
  case
    when "eGFR Value" is null then null
    when "eGFR Value" >= 90 then 'G1 - Normal or High'
    when "eGFR Value" >= 60 then 'G2 - Mildly Decreased'
    when "eGFR Value" >= 45 then 'G3a - Mildly to Moderately Decreased'
    when "eGFR Value" >= 30 then 'G3b - Moderately to Severely Decreased'
    when "eGFR Value" >= 15 then 'G4 - Severely Decreased'
    else 'G5 - Kidney Failure'
  end

define "Has Severe Renal Impairment":
  "eGFR Value" is not null and "eGFR Value" < 30

define "Has Moderate Renal Impairment":
  "eGFR Value" is not null and "eGFR Value" >= 30 and "eGFR Value" < 60

define "Has Mild Renal Impairment":
  "eGFR Value" is not null and "eGFR Value" >= 60 and "eGFR Value" < 90

define "Requires Dose Adjustment for Renal Function":
  "Has Severe Renal Impairment" or "Has Moderate Renal Impairment"

// ============================================================================
// SECTION 3: BODY MASS INDEX (BMI) - Can Be Computed in CQL
// ============================================================================

/*
 * BMI Calculation: weight(kg) / height(m)²
 * This simple division CAN be performed in CQL.
 */
define "BMI Value":
  if "Most Recent Weight Kg" is null or "Most Recent Height Cm" is null then null
  else "Most Recent Weight Kg" / (("Most Recent Height Cm" / 100) * ("Most Recent Height Cm" / 100))

/*
 * BMI Category (WHO Classification)
 */
define "BMI Category":
  case
    when "BMI Value" is null then 'Unknown'
    when "BMI Value" < 18.5 then 'Underweight'
    when "BMI Value" < 25 then 'Normal Weight'
    when "BMI Value" < 30 then 'Overweight'
    when "BMI Value" < 35 then 'Obese Class I'
    when "BMI Value" < 40 then 'Obese Class II'
    else 'Obese Class III (Severe)'
  end

/*
 * Asian BMI Thresholds (WHO Asia-Pacific Classification)
 * Used for Tier 5 regional adapters (India, ICMR guidelines)
 *
 * CLINICAL NOTE: Asian populations have increased cardiometabolic
 * risk at lower BMI compared to European populations.
 */
define "BMI Category Asian":
  case
    when "BMI Value" is null then 'Unknown'
    when "BMI Value" < 18.5 then 'Underweight'
    when "BMI Value" < 23 then 'Normal Weight'
    when "BMI Value" < 25 then 'Overweight (At Risk)'
    when "BMI Value" < 30 then 'Obese Class I'
    else 'Obese Class II+'
  end

define "Obesity Present":
  "BMI Value" >= 30

define "Obesity Present Asian Criteria":
  "BMI Value" >= 25

// ============================================================================
// SECTION 4: BODY SURFACE AREA (BSA) - From KB-8 Snapshot
// ============================================================================

/*
 * BSA Calculation: Mosteller Formula
 * Formula: sqrt((height_cm × weight_kg) / 3600)
 *
 * Requires square root - CANNOT be computed in CQL.
 * Pre-computed by KB-8 Calculator Service.
 *
 * CLINICAL USE: Chemotherapy dosing, cardiac output indexing
 */
define "BSA Value":
  null as Decimal

define "BSA Category":
  case
    when "BSA Value" is null then 'Unknown'
    when "BSA Value" < 1.5 then 'Small (<1.5 m²)'
    when "BSA Value" < 2.0 then 'Normal (1.5-2.0 m²)'
    else 'Large (>2.0 m²)'
  end

// ============================================================================
// SECTION 5: CHILD-PUGH SCORE - From KB-8 Snapshot
// ============================================================================

/*
 * Child-Pugh Score (Hepatic Function Assessment)
 *
 * Components (each scored 1-3):
 * - Total Bilirubin
 * - Serum Albumin
 * - INR/PT
 * - Ascites (clinical assessment)
 * - Hepatic Encephalopathy (clinical assessment)
 *
 * Total: 5-15 points
 * Class A: 5-6 (well-compensated)
 * Class B: 7-9 (significant compromise)
 * Class C: 10-15 (decompensated)
 *
 * Pre-computed by KB-8 using lab values + clinical inputs.
 */
define "Child-Pugh Score":
  null as Integer

define "Child-Pugh Class":
  case
    when "Child-Pugh Score" is null then null
    when "Child-Pugh Score" <= 6 then 'A'
    when "Child-Pugh Score" <= 9 then 'B'
    else 'C'
  end

define "Has Severe Hepatic Impairment":
  "Child-Pugh Class" = 'C'

define "Has Moderate Hepatic Impairment":
  "Child-Pugh Class" = 'B'

define "Has Mild Hepatic Impairment":
  "Child-Pugh Class" = 'A'

define "Requires Dose Adjustment for Hepatic Function":
  "Has Severe Hepatic Impairment" or "Has Moderate Hepatic Impairment"

// ============================================================================
// SECTION 6: MELD SCORE - From KB-8 Snapshot
// ============================================================================

/*
 * MELD Score (Model for End-Stage Liver Disease)
 *
 * Formula: 3.78×ln(bilirubin) + 11.2×ln(INR) + 9.57×ln(creatinine) + 6.43
 *
 * Requires natural logarithm - CANNOT be computed in CQL.
 * Pre-computed by KB-8 Calculator Service.
 *
 * CLINICAL USE: Liver transplant prioritization, mortality prediction
 */
define "MELD Score":
  null as Integer

define "MELD Mortality Risk":
  case
    when "MELD Score" is null then 'Unknown'
    when "MELD Score" < 10 then 'Low (<2% 90-day mortality)'
    when "MELD Score" < 20 then 'Moderate (~6% 90-day mortality)'
    when "MELD Score" < 30 then 'High (~20% 90-day mortality)'
    when "MELD Score" < 40 then 'Very High (~53% 90-day mortality)'
    else 'Severe (>71% 90-day mortality)'
  end

define "MELD Na Score":
  // MELD-Na includes sodium correction - from KB-8
  null as Integer

// ============================================================================
// SECTION 7: SOFA SCORE (ICU) - From KB-8 Snapshot
// ============================================================================

/*
 * SOFA Score (Sequential Organ Failure Assessment)
 *
 * Components (each scored 0-4):
 * - Respiration (PaO2/FiO2 ratio)
 * - Coagulation (Platelet count)
 * - Liver (Bilirubin)
 * - Cardiovascular (MAP + vasopressors)
 * - CNS (Glasgow Coma Scale)
 * - Renal (Creatinine + urine output)
 *
 * Total: 0-24 points
 *
 * CLINICAL USE: ICU mortality prediction, sepsis diagnosis (Sepsis-3)
 */
define "SOFA Score":
  null as Integer

define "Has Organ Dysfunction":
  "SOFA Score" is not null and "SOFA Score" >= 2

define "SOFA Mortality Risk":
  case
    when "SOFA Score" is null then 'Unknown'
    when "SOFA Score" < 2 then 'Very Low (<10%)'
    when "SOFA Score" < 4 then 'Low (<10%)'
    when "SOFA Score" < 6 then 'Moderate (15-20%)'
    when "SOFA Score" < 8 then 'High (40-50%)'
    when "SOFA Score" < 10 then 'Very High (50-60%)'
    else 'Critical (>80%)'
  end

define "SOFA Score Change":
  // Delta SOFA from baseline - from KB-8
  null as Integer

define "SOFA Trending Worse":
  "SOFA Score Change" is not null and "SOFA Score Change" >= 2

// ============================================================================
// SECTION 8: qSOFA SCORE - Can Be Computed in CQL
// ============================================================================

/*
 * qSOFA Score (Quick SOFA) - Bedside Sepsis Screen
 *
 * Components (each 0 or 1):
 * - Respiratory rate ≥ 22
 * - Altered mentation (GCS < 15)
 * - Systolic BP ≤ 100
 *
 * This is a simple criteria count - CAN be computed in CQL.
 */

define "Has Respiratory Rate High for qSOFA":
  exists(
    [Observation: "Respiratory Rate"] O
      where O.status in { 'final', 'amended', 'corrected' }
        and FHIRHelpers.ToQuantity(O.value as FHIR.Quantity).value >= 22
        and O.effective in day of Today()
  )

define "Has Systolic BP Low for qSOFA":
  exists(
    [Observation: "Systolic Blood Pressure"] O
      where O.status in { 'final', 'amended', 'corrected' }
        and FHIRHelpers.ToQuantity(O.value as FHIR.Quantity).value <= 100
        and O.effective in day of Today()
  )

define "Has Altered Mentation for qSOFA":
  exists(
    [Observation: "Glasgow Coma Scale"] O
      where O.status in { 'final', 'amended', 'corrected' }
        and FHIRHelpers.ToQuantity(O.value as FHIR.Quantity).value < 15
        and O.effective in day of Today()
  )

define "qSOFA Score":
  (if "Has Respiratory Rate High for qSOFA" then 1 else 0) +
  (if "Has Systolic BP Low for qSOFA" then 1 else 0) +
  (if "Has Altered Mentation for qSOFA" then 1 else 0)

define "qSOFA Positive":
  "qSOFA Score" >= 2

// ============================================================================
// SECTION 9: ASCVD RISK SCORE - From KB-8 Snapshot
// ============================================================================

/*
 * ASCVD 10-Year Risk Score (Pooled Cohort Equations)
 *
 * Requires exponential calculations - CANNOT be computed in CQL.
 * Pre-computed by KB-8 Calculator Service.
 *
 * Inputs:
 * - Age, Sex, Race
 * - Total Cholesterol, HDL Cholesterol
 * - Systolic BP, Treatment status
 * - Diabetes, Smoking status
 *
 * CLINICAL USE: Statin therapy initiation (ACC/AHA guidelines)
 */
define "ASCVD 10 Year Risk":
  null as Decimal

define "ASCVD Risk Category":
  case
    when "ASCVD 10 Year Risk" is null then 'Unknown'
    when "ASCVD 10 Year Risk" < 5.0 then 'Low (<5%)'
    when "ASCVD 10 Year Risk" < 7.5 then 'Borderline (5-7.5%)'
    when "ASCVD 10 Year Risk" < 20.0 then 'Intermediate (7.5-20%)'
    else 'High (≥20%)'
  end

define "Statin Recommended by ASCVD Risk":
  "ASCVD 10 Year Risk" is not null and "ASCVD 10 Year Risk" >= 7.5

define "High Intensity Statin Indicated":
  "ASCVD 10 Year Risk" is not null and "ASCVD 10 Year Risk" >= 20.0

// ============================================================================
// SECTION 10: CHA₂DS₂-VASc SCORE - From KB-8 Snapshot
// ============================================================================

/*
 * CHA₂DS₂-VASc Score (AFib Stroke Risk)
 *
 * Components:
 * - C: Congestive heart failure (1)
 * - H: Hypertension (1)
 * - A₂: Age ≥75 (2)
 * - D: Diabetes mellitus (1)
 * - S₂: Prior Stroke/TIA/TE (2)
 * - V: Vascular disease (1)
 * - A: Age 65-74 (1)
 * - Sc: Sex category female (1)
 *
 * Some components require clinical assessment beyond labs.
 * Pre-computed by KB-8 with full patient context.
 */
define "CHA2DS2VASc Score":
  null as Integer

define "Anticoagulation Recommended by CHA2DS2VASc":
  "CHA2DS2VASc Score" is not null and "CHA2DS2VASc Score" >= 2

define "Anticoagulation Consider by CHA2DS2VASc":
  "CHA2DS2VASc Score" = 1

define "Annual Stroke Risk CHA2DS2VASc":
  case
    when "CHA2DS2VASc Score" is null then null
    when "CHA2DS2VASc Score" = 0 then 0.0
    when "CHA2DS2VASc Score" = 1 then 1.3
    when "CHA2DS2VASc Score" = 2 then 2.2
    when "CHA2DS2VASc Score" = 3 then 3.2
    when "CHA2DS2VASc Score" = 4 then 4.0
    when "CHA2DS2VASc Score" = 5 then 6.7
    when "CHA2DS2VASc Score" = 6 then 9.8
    when "CHA2DS2VASc Score" = 7 then 9.6
    when "CHA2DS2VASc Score" = 8 then 6.7
    when "CHA2DS2VASc Score" = 9 then 15.2
    else null
  end

// ============================================================================
// SECTION 11: HAS-BLED SCORE - From KB-8 Snapshot
// ============================================================================

/*
 * HAS-BLED Score (Bleeding Risk Assessment)
 *
 * Components:
 * - H: Hypertension (1)
 * - A: Abnormal renal/liver function (1-2)
 * - S: Stroke (1)
 * - B: Bleeding history (1)
 * - L: Labile INR (1)
 * - E: Elderly >65 (1)
 * - D: Drugs or alcohol (1-2)
 *
 * Pre-computed by KB-8.
 */
define "HAS-BLED Score":
  null as Integer

define "High Bleeding Risk":
  "HAS-BLED Score" is not null and "HAS-BLED Score" >= 3

define "Moderate Bleeding Risk":
  "HAS-BLED Score" is not null and "HAS-BLED Score" >= 1 and "HAS-BLED Score" < 3

// ============================================================================
// SECTION 12: WELLS SCORE (DVT/PE) - From KB-8 Snapshot
// ============================================================================

/*
 * Wells Score for DVT Risk
 *
 * Requires clinical criteria (paralysis, bed rest, malignancy, etc.)
 * Pre-computed by KB-8 with clinical context.
 */
define "Wells DVT Score":
  null as Integer

define "Wells DVT Risk Category":
  case
    when "Wells DVT Score" is null then 'Unknown'
    when "Wells DVT Score" <= 0 then 'Low (5%)'
    when "Wells DVT Score" <= 2 then 'Moderate (17%)'
    else 'High (53%)'
  end

/*
 * Wells Score for PE Risk
 */
define "Wells PE Score":
  null as Integer

define "Wells PE Risk Category":
  case
    when "Wells PE Score" is null then 'Unknown'
    when "Wells PE Score" <= 4 then 'PE Unlikely'
    else 'PE Likely'
  end

// ============================================================================
// SECTION 13: FIB-4 SCORE - From KB-8 Snapshot
// ============================================================================

/*
 * FIB-4 Score (Fibrosis Index)
 *
 * Formula: (Age × AST) / (Platelets × √ALT)
 *
 * Requires square root - CANNOT be computed in CQL.
 * Pre-computed by KB-8.
 *
 * CLINICAL USE: NAFLD/NASH fibrosis risk stratification
 */
define "FIB-4 Score":
  null as Decimal

define "FIB-4 Risk Category":
  case
    when "FIB-4 Score" is null then 'Unknown'
    when "FIB-4 Score" < 1.30 then 'Low Risk (F0-F2)'
    when "FIB-4 Score" < 2.67 then 'Intermediate Risk'
    else 'High Risk (F3-F4)'
  end

define "Advanced Fibrosis Likely":
  "FIB-4 Score" is not null and "FIB-4 Score" >= 2.67

// ============================================================================
// SECTION 14: CREATININE CLEARANCE (Cockcroft-Gault) - From KB-8 Snapshot
// ============================================================================

/*
 * Creatinine Clearance (Cockcroft-Gault Formula)
 *
 * Formula: ((140-age) × weight) / (72 × creatinine) × [0.85 if female]
 *
 * CLINICAL NOTE: Different from eGFR. Still used for drug dosing
 * especially for drugs studied with Cockcroft-Gault (e.g., DOACs).
 *
 * Pre-computed by KB-8.
 */
define "Creatinine Clearance CG":
  null as Decimal

define "CrCl Dosing Category":
  case
    when "Creatinine Clearance CG" is null then 'Unknown'
    when "Creatinine Clearance CG" >= 50 then 'Normal (≥50 mL/min)'
    when "Creatinine Clearance CG" >= 30 then 'Moderate Impairment (30-49 mL/min)'
    when "Creatinine Clearance CG" >= 15 then 'Severe Impairment (15-29 mL/min)'
    else 'End-Stage (<15 mL/min)'
  end

// ============================================================================
// SECTION 15: IDEAL BODY WEIGHT - Can Be Computed in CQL
// ============================================================================

/*
 * Ideal Body Weight (Devine Formula)
 *
 * Male: 50 + 2.3 × (height_inches - 60)
 * Female: 45.5 + 2.3 × (height_inches - 60)
 *
 * Simple multiplication - CAN be computed in CQL if sex is known.
 * Note: Placeholder for sex extraction - typically from Demographics
 */
define "Height in Inches":
  if "Most Recent Height Cm" is null then null
  else "Most Recent Height Cm" / 2.54

define "Ideal Body Weight Male":
  if "Height in Inches" is null then null
  else 50 + 2.3 * ("Height in Inches" - 60)

define "Ideal Body Weight Female":
  if "Height in Inches" is null then null
  else 45.5 + 2.3 * ("Height in Inches" - 60)

/*
 * Adjusted Body Weight (for obese patients)
 * ABW = IBW + 0.4 × (Actual Weight - IBW)
 */
define "Adjusted Body Weight Male":
  if "Ideal Body Weight Male" is null or "Most Recent Weight Kg" is null then null
  else "Ideal Body Weight Male" + 0.4 * ("Most Recent Weight Kg" - "Ideal Body Weight Male")

define "Adjusted Body Weight Female":
  if "Ideal Body Weight Female" is null or "Most Recent Weight Kg" is null then null
  else "Ideal Body Weight Female" + 0.4 * ("Most Recent Weight Kg" - "Ideal Body Weight Female")

// ============================================================================
// SECTION 16: FRAMINGHAM RISK SCORE - From KB-8 Snapshot
// ============================================================================

/*
 * Framingham 10-Year CVD Risk
 *
 * Similar to ASCVD but different population and formula.
 * Pre-computed by KB-8.
 */
define "Framingham 10 Year Risk":
  null as Decimal

define "Framingham Risk Category":
  case
    when "Framingham 10 Year Risk" is null then 'Unknown'
    when "Framingham 10 Year Risk" < 10.0 then 'Low (<10%)'
    when "Framingham 10 Year Risk" < 20.0 then 'Intermediate (10-20%)'
    else 'High (≥20%)'
  end

// ============================================================================
// ICU CALCULATOR FACTS OUTPUT (For Go Consumption)
// ============================================================================

/*
 * Aggregated calculator outputs for ICU Intelligence consumption.
 * This tuple provides all pre-computed scores from KB-8 snapshot.
 */
define "ICU Calculator Facts":
  {
    // Renal Function
    eGFRValue: "eGFR Value",
    ckdStage: "CKD Stage from eGFR",
    severeRenalImpairment: "Has Severe Renal Impairment",
    creatinineClearance: "Creatinine Clearance CG",

    // Hepatic Function
    childPughScore: "Child-Pugh Score",
    childPughClass: "Child-Pugh Class",
    meldScore: "MELD Score",
    severeHepaticImpairment: "Has Severe Hepatic Impairment",

    // ICU Scores
    sofaScore: "SOFA Score",
    sofaTrendingWorse: "SOFA Trending Worse",
    qsofaScore: "qSOFA Score",
    qsofaPositive: "qSOFA Positive",
    organDysfunction: "Has Organ Dysfunction",

    // Body Composition
    bmiValue: "BMI Value",
    bmiCategory: "BMI Category",
    bsaValue: "BSA Value",

    // Cardiovascular Risk
    ascvdRisk: "ASCVD 10 Year Risk",
    statinRecommended: "Statin Recommended by ASCVD Risk",
    cha2ds2vascScore: "CHA2DS2VASc Score",
    anticoagulationRecommended: "Anticoagulation Recommended by CHA2DS2VASc",
    hasbledScore: "HAS-BLED Score",
    highBleedingRisk: "High Bleeding Risk",

    // VTE Risk
    wellsDVTScore: "Wells DVT Score",
    wellsDVTRisk: "Wells DVT Risk Category",
    wellsPEScore: "Wells PE Score",
    wellsPERisk: "Wells PE Risk Category",

    // Liver Fibrosis
    fib4Score: "FIB-4 Score",
    advancedFibrosisLikely: "Advanced Fibrosis Likely"
  }
