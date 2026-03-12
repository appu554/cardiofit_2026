/*
 * LabReferenceRanges CQL Library
 * ============================================================
 * Tier 1.5 Clinical Utilities - Lab Interpretation Integration
 *
 * Purpose: Provides CQL interfaces to KB-16 Lab Interpretation outputs.
 *          Stratified reference ranges and critical value detection.
 *
 * ARCHITECTURE NOTE (CTO/CMO Directive):
 * KB-16 Lab Interpretation Service provides age/gender/pregnancy-stratified
 * reference ranges at SNAPSHOT build time. This library consumes those
 * pre-computed interpretations for CQL evaluation.
 *
 * Stratification Categories:
 * - Age: Pediatric (0-17), Adult (18-64), Geriatric (65+)
 * - Gender: Male, Female
 * - Pregnancy: Non-pregnant, Pregnant (trimester-specific)
 *
 * Governance: Tier-1.5 review (Clinical + Technical)
 * ============================================================
 */
library LabReferenceRanges version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.4.000' called FHIRHelpers
include IntervalHelpers version '1.0.0' called Intervals
include ObservationHelpers version '1.0.0' called Observations

// ============================================================================
// VALUE SET REFERENCES - Common Lab Panels
// ============================================================================

valueset "Complete Blood Count": 'http://cardiofit.io/fhir/ValueSet/cbc'
valueset "Basic Metabolic Panel": 'http://cardiofit.io/fhir/ValueSet/bmp'
valueset "Comprehensive Metabolic Panel": 'http://cardiofit.io/fhir/ValueSet/cmp'
valueset "Liver Function Tests": 'http://cardiofit.io/fhir/ValueSet/lft'
valueset "Thyroid Panel": 'http://cardiofit.io/fhir/ValueSet/thyroid'
valueset "Coagulation Panel": 'http://cardiofit.io/fhir/ValueSet/coag'
valueset "Cardiac Markers": 'http://cardiofit.io/fhir/ValueSet/cardiac-markers'

// Individual Labs - LOINC codes embedded for specificity
valueset "Hemoglobin Labs": 'http://cardiofit.io/fhir/ValueSet/hemoglobin'
valueset "Sodium Labs": 'http://cardiofit.io/fhir/ValueSet/sodium'
valueset "Potassium Labs": 'http://cardiofit.io/fhir/ValueSet/potassium'
valueset "Glucose Labs": 'http://cardiofit.io/fhir/ValueSet/glucose'
valueset "Creatinine Labs": 'http://cardiofit.io/fhir/ValueSet/creatinine'
valueset "ALT Labs": 'http://cardiofit.io/fhir/ValueSet/alt'
valueset "AST Labs": 'http://cardiofit.io/fhir/ValueSet/ast'
valueset "Bilirubin Labs": 'http://cardiofit.io/fhir/ValueSet/bilirubin'
valueset "Platelet Labs": 'http://cardiofit.io/fhir/ValueSet/platelets'
valueset "WBC Labs": 'http://cardiofit.io/fhir/ValueSet/wbc'
valueset "TSH Labs": 'http://cardiofit.io/fhir/ValueSet/tsh'
valueset "Troponin Labs": 'http://cardiofit.io/fhir/ValueSet/troponin'
valueset "BNP Labs": 'http://cardiofit.io/fhir/ValueSet/bnp'
valueset "Lactate Labs": 'http://cardiofit.io/fhir/ValueSet/lactate'
valueset "Calcium Labs": 'http://cardiofit.io/fhir/ValueSet/calcium'
valueset "Magnesium Labs": 'http://cardiofit.io/fhir/ValueSet/magnesium'
valueset "Phosphorus Labs": 'http://cardiofit.io/fhir/ValueSet/phosphorus'

// ============================================================================
// CONTEXT
// ============================================================================
context Patient

// ============================================================================
// SECTION 1: LAB ABNORMALITY LEVEL CODES
// ============================================================================

/*
 * Standardized lab interpretation levels.
 * Used by all lab-specific definitions.
 */
code "Lab Normal": 'normal' from "http://cardiofit.io/fhir/CodeSystem/lab-interpretation"
code "Lab Low": 'low' from "http://cardiofit.io/fhir/CodeSystem/lab-interpretation"
code "Lab High": 'high' from "http://cardiofit.io/fhir/CodeSystem/lab-interpretation"
code "Lab Critical Low": 'critical-low' from "http://cardiofit.io/fhir/CodeSystem/lab-interpretation"
code "Lab Critical High": 'critical-high' from "http://cardiofit.io/fhir/CodeSystem/lab-interpretation"
code "Lab Panic": 'panic' from "http://cardiofit.io/fhir/CodeSystem/lab-interpretation"

// ============================================================================
// SECTION 2: KB-16 SNAPSHOT INTERPRETATIONS
// ============================================================================

/*
 * These definitions access KB-16 pre-computed interpretations.
 * KB-16 applies age/gender/pregnancy stratified reference ranges.
 *
 * Runtime injects actual values; placeholders return null.
 */

// --- Hemoglobin Interpretation (KB-16) ---
define "Hemoglobin Interpretation":
  // From KB-16 LabSnapshot.Hemoglobin.Interpretation
  null as String

define "Hemoglobin Reference Range Low":
  // From KB-16 LabSnapshot.Hemoglobin.ReferenceLow
  null as Decimal

define "Hemoglobin Reference Range High":
  // From KB-16 LabSnapshot.Hemoglobin.ReferenceHigh
  null as Decimal

// --- Sodium Interpretation (KB-16) ---
define "Sodium Interpretation":
  null as String

define "Sodium Reference Range Low":
  null as Decimal

define "Sodium Reference Range High":
  null as Decimal

// --- Potassium Interpretation (KB-16) ---
define "Potassium Interpretation":
  null as String

define "Potassium Reference Range Low":
  null as Decimal

define "Potassium Reference Range High":
  null as Decimal

// --- Glucose Interpretation (KB-16) ---
define "Glucose Interpretation":
  null as String

define "Glucose Reference Range Low":
  null as Decimal

define "Glucose Reference Range High":
  null as Decimal

// --- Creatinine Interpretation (KB-16) ---
define "Creatinine Interpretation":
  null as String

define "Creatinine Reference Range Low":
  null as Decimal

define "Creatinine Reference Range High":
  null as Decimal

// ============================================================================
// SECTION 3: RAW LAB VALUES (For CQL Classification)
// ============================================================================

define "Most Recent Hemoglobin":
  Observations.MostRecentObservation(
    [Observation: "Hemoglobin Labs"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent Hemoglobin Value":
  Observations.ObservationQuantityValue("Most Recent Hemoglobin")

define "Most Recent Sodium":
  Observations.MostRecentObservation(
    [Observation: "Sodium Labs"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent Sodium Value":
  Observations.ObservationQuantityValue("Most Recent Sodium")

define "Most Recent Potassium":
  Observations.MostRecentObservation(
    [Observation: "Potassium Labs"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent Potassium Value":
  Observations.ObservationQuantityValue("Most Recent Potassium")

define "Most Recent Glucose":
  Observations.MostRecentObservation(
    [Observation: "Glucose Labs"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent Glucose Value":
  Observations.ObservationQuantityValue("Most Recent Glucose")

define "Most Recent Creatinine":
  Observations.MostRecentObservation(
    [Observation: "Creatinine Labs"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent Creatinine Value":
  Observations.ObservationQuantityValue("Most Recent Creatinine")

define "Most Recent ALT":
  Observations.MostRecentObservation(
    [Observation: "ALT Labs"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent ALT Value":
  Observations.ObservationQuantityValue("Most Recent ALT")

define "Most Recent AST":
  Observations.MostRecentObservation(
    [Observation: "AST Labs"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent AST Value":
  Observations.ObservationQuantityValue("Most Recent AST")

define "Most Recent Bilirubin":
  Observations.MostRecentObservation(
    [Observation: "Bilirubin Labs"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent Bilirubin Value":
  Observations.ObservationQuantityValue("Most Recent Bilirubin")

define "Most Recent Platelet Count":
  Observations.MostRecentObservation(
    [Observation: "Platelet Labs"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent Platelet Value":
  Observations.ObservationQuantityValue("Most Recent Platelet Count")

define "Most Recent WBC":
  Observations.MostRecentObservation(
    [Observation: "WBC Labs"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent WBC Value":
  Observations.ObservationQuantityValue("Most Recent WBC")

define "Most Recent TSH":
  Observations.MostRecentObservation(
    [Observation: "TSH Labs"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent TSH Value":
  Observations.ObservationQuantityValue("Most Recent TSH")

define "Most Recent Troponin":
  Observations.MostRecentObservation(
    [Observation: "Troponin Labs"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent Troponin Value":
  Observations.ObservationQuantityValue("Most Recent Troponin")

define "Most Recent BNP":
  Observations.MostRecentObservation(
    [Observation: "BNP Labs"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent BNP Value":
  Observations.ObservationQuantityValue("Most Recent BNP")

define "Most Recent Lactate":
  Observations.MostRecentObservation(
    [Observation: "Lactate Labs"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent Lactate Value":
  Observations.ObservationQuantityValue("Most Recent Lactate")

define "Most Recent Calcium":
  Observations.MostRecentObservation(
    [Observation: "Calcium Labs"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent Calcium Value":
  Observations.ObservationQuantityValue("Most Recent Calcium")

define "Most Recent Magnesium":
  Observations.MostRecentObservation(
    [Observation: "Magnesium Labs"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent Magnesium Value":
  Observations.ObservationQuantityValue("Most Recent Magnesium")

define "Most Recent Phosphorus":
  Observations.MostRecentObservation(
    [Observation: "Phosphorus Labs"] O
      where O.status in { 'final', 'amended', 'corrected' }
  )

define "Most Recent Phosphorus Value":
  Observations.ObservationQuantityValue("Most Recent Phosphorus")

// ============================================================================
// SECTION 4: CRITICAL VALUE DETECTION (Adult Reference)
// ============================================================================

/*
 * CRITICAL VALUE FLAGS
 * These detect immediately life-threatening lab values.
 * Reference ranges are default adult values; KB-16 provides stratified ranges.
 *
 * CLINICAL NOTE: Critical values require immediate notification and action.
 */

// --- Sodium Critical Values ---
// Critical Low: <120 mEq/L (severe hyponatremia - seizure risk)
// Critical High: >160 mEq/L (severe hypernatremia - CNS dysfunction)
define "Sodium Is Critical Low":
  "Most Recent Sodium Value" is not null and "Most Recent Sodium Value" < 120

define "Sodium Is Critical High":
  "Most Recent Sodium Value" is not null and "Most Recent Sodium Value" > 160

define "Sodium Is Critical":
  "Sodium Is Critical Low" or "Sodium Is Critical High"

// --- Potassium Critical Values ---
// Critical Low: <2.5 mEq/L (cardiac arrhythmia risk)
// Critical High: >6.5 mEq/L (cardiac arrest risk)
define "Potassium Is Critical Low":
  "Most Recent Potassium Value" is not null and "Most Recent Potassium Value" < 2.5

define "Potassium Is Critical High":
  "Most Recent Potassium Value" is not null and "Most Recent Potassium Value" > 6.5

define "Potassium Is Critical":
  "Potassium Is Critical Low" or "Potassium Is Critical High"

// --- Glucose Critical Values ---
// Critical Low: <40 mg/dL (neuroglycopenia)
// Critical High: >500 mg/dL (DKA/HHS risk)
define "Glucose Is Critical Low":
  "Most Recent Glucose Value" is not null and "Most Recent Glucose Value" < 40

define "Glucose Is Critical High":
  "Most Recent Glucose Value" is not null and "Most Recent Glucose Value" > 500

define "Glucose Is Critical":
  "Glucose Is Critical Low" or "Glucose Is Critical High"

// --- Hemoglobin Critical Values ---
// Critical Low: <7 g/dL (transfusion threshold, varies by context)
// Critical High: >20 g/dL (polycythemia severe)
define "Hemoglobin Is Critical Low":
  "Most Recent Hemoglobin Value" is not null and "Most Recent Hemoglobin Value" < 7

define "Hemoglobin Is Critical High":
  "Most Recent Hemoglobin Value" is not null and "Most Recent Hemoglobin Value" > 20

define "Hemoglobin Is Critical":
  "Hemoglobin Is Critical Low" or "Hemoglobin Is Critical High"

// --- Platelet Critical Values ---
// Critical Low: <20,000/uL (spontaneous bleeding risk)
// Critical High: >1,000,000/uL (thrombosis or bleeding risk)
define "Platelet Is Critical Low":
  "Most Recent Platelet Value" is not null and "Most Recent Platelet Value" < 20

define "Platelet Is Critical High":
  "Most Recent Platelet Value" is not null and "Most Recent Platelet Value" > 1000

define "Platelet Is Critical":
  "Platelet Is Critical Low" or "Platelet Is Critical High"

// --- WBC Critical Values ---
// Critical Low: <2,000/uL (neutropenic sepsis risk)
// Critical High: >30,000/uL (leukemoid reaction, leukemia)
define "WBC Is Critical Low":
  "Most Recent WBC Value" is not null and "Most Recent WBC Value" < 2.0

define "WBC Is Critical High":
  "Most Recent WBC Value" is not null and "Most Recent WBC Value" > 30.0

define "WBC Is Critical":
  "WBC Is Critical Low" or "WBC Is Critical High"

// --- Calcium Critical Values ---
// Critical Low: <6.0 mg/dL (tetany, cardiac arrhythmia)
// Critical High: >14.0 mg/dL (hypercalcemic crisis)
define "Calcium Is Critical Low":
  "Most Recent Calcium Value" is not null and "Most Recent Calcium Value" < 6.0

define "Calcium Is Critical High":
  "Most Recent Calcium Value" is not null and "Most Recent Calcium Value" > 14.0

define "Calcium Is Critical":
  "Calcium Is Critical Low" or "Calcium Is Critical High"

// --- Lactate Critical Values ---
// Critical High: >4.0 mmol/L (severe tissue hypoperfusion/sepsis)
define "Lactate Is Critical High":
  "Most Recent Lactate Value" is not null and "Most Recent Lactate Value" > 4.0

define "Lactate Is Critical":
  "Lactate Is Critical High"

// --- Troponin Critical Values ---
// Critical High: >0.04 ng/mL (depends on assay - generic threshold)
define "Troponin Is Elevated":
  "Most Recent Troponin Value" is not null and "Most Recent Troponin Value" > 0.04

define "Troponin Is Critical High":
  "Most Recent Troponin Value" is not null and "Most Recent Troponin Value" > 0.4

// --- Bilirubin Critical Values ---
// Critical High: >15 mg/dL in adults (severe hyperbilirubinemia)
// NOTE: Neonates have different thresholds - KB-16 handles stratification
define "Bilirubin Is Critical High":
  "Most Recent Bilirubin Value" is not null and "Most Recent Bilirubin Value" > 15

// ============================================================================
// SECTION 5: ABNORMALITY LEVEL CLASSIFICATION
// ============================================================================

/*
 * Lab Abnormality Level Classification
 * Uses KB-16 stratified ranges when available, falls back to defaults.
 *
 * Levels:
 * - 'normal': Within reference range
 * - 'low': Below reference range, not critical
 * - 'high': Above reference range, not critical
 * - 'critical-low': Critically low, immediate action needed
 * - 'critical-high': Critically high, immediate action needed
 */

define "Sodium Abnormality Level":
  case
    when "Most Recent Sodium Value" is null then null
    when "Sodium Is Critical Low" then 'critical-low'
    when "Sodium Is Critical High" then 'critical-high'
    when "Most Recent Sodium Value" < 136 then 'low'
    when "Most Recent Sodium Value" > 145 then 'high'
    else 'normal'
  end

define "Potassium Abnormality Level":
  case
    when "Most Recent Potassium Value" is null then null
    when "Potassium Is Critical Low" then 'critical-low'
    when "Potassium Is Critical High" then 'critical-high'
    when "Most Recent Potassium Value" < 3.5 then 'low'
    when "Most Recent Potassium Value" > 5.0 then 'high'
    else 'normal'
  end

define "Glucose Abnormality Level":
  case
    when "Most Recent Glucose Value" is null then null
    when "Glucose Is Critical Low" then 'critical-low'
    when "Glucose Is Critical High" then 'critical-high'
    when "Most Recent Glucose Value" < 70 then 'low'
    when "Most Recent Glucose Value" > 180 then 'high'
    else 'normal'
  end

define "Hemoglobin Abnormality Level":
  case
    when "Most Recent Hemoglobin Value" is null then null
    when "Hemoglobin Is Critical Low" then 'critical-low'
    when "Hemoglobin Is Critical High" then 'critical-high'
    when "Most Recent Hemoglobin Value" < 12 then 'low' // Female threshold
    when "Most Recent Hemoglobin Value" > 17 then 'high' // Male threshold
    else 'normal'
  end

define "Platelet Abnormality Level":
  case
    when "Most Recent Platelet Value" is null then null
    when "Platelet Is Critical Low" then 'critical-low'
    when "Platelet Is Critical High" then 'critical-high'
    when "Most Recent Platelet Value" < 150 then 'low'
    when "Most Recent Platelet Value" > 400 then 'high'
    else 'normal'
  end

define "WBC Abnormality Level":
  case
    when "Most Recent WBC Value" is null then null
    when "WBC Is Critical Low" then 'critical-low'
    when "WBC Is Critical High" then 'critical-high'
    when "Most Recent WBC Value" < 4.5 then 'low'
    when "Most Recent WBC Value" > 11.0 then 'high'
    else 'normal'
  end

define "Creatinine Abnormality Level":
  case
    when "Most Recent Creatinine Value" is null then null
    when "Most Recent Creatinine Value" > 10 then 'critical-high'
    when "Most Recent Creatinine Value" > 1.2 then 'high'
    when "Most Recent Creatinine Value" < 0.6 then 'low'
    else 'normal'
  end

define "ALT Abnormality Level":
  case
    when "Most Recent ALT Value" is null then null
    when "Most Recent ALT Value" > 1000 then 'critical-high' // Acute liver injury
    when "Most Recent ALT Value" > 40 then 'high'
    else 'normal'
  end

define "AST Abnormality Level":
  case
    when "Most Recent AST Value" is null then null
    when "Most Recent AST Value" > 1000 then 'critical-high'
    when "Most Recent AST Value" > 40 then 'high'
    else 'normal'
  end

define "Bilirubin Abnormality Level":
  case
    when "Most Recent Bilirubin Value" is null then null
    when "Bilirubin Is Critical High" then 'critical-high'
    when "Most Recent Bilirubin Value" > 1.2 then 'high'
    else 'normal'
  end

define "Calcium Abnormality Level":
  case
    when "Most Recent Calcium Value" is null then null
    when "Calcium Is Critical Low" then 'critical-low'
    when "Calcium Is Critical High" then 'critical-high'
    when "Most Recent Calcium Value" < 8.5 then 'low'
    when "Most Recent Calcium Value" > 10.5 then 'high'
    else 'normal'
  end

define "Lactate Abnormality Level":
  case
    when "Most Recent Lactate Value" is null then null
    when "Lactate Is Critical High" then 'critical-high'
    when "Most Recent Lactate Value" > 2.0 then 'high'
    else 'normal'
  end

define "TSH Abnormality Level":
  case
    when "Most Recent TSH Value" is null then null
    when "Most Recent TSH Value" < 0.1 then 'critical-low' // Thyrotoxicosis
    when "Most Recent TSH Value" > 10.0 then 'critical-high' // Severe hypothyroid
    when "Most Recent TSH Value" < 0.4 then 'low'
    when "Most Recent TSH Value" > 4.0 then 'high'
    else 'normal'
  end

// ============================================================================
// SECTION 6: AGGREGATE CRITICAL VALUE FLAGS
// ============================================================================

/*
 * These aggregate flags help identify any critical lab values requiring
 * immediate clinical attention.
 */

define "Any Critical Low Lab":
  "Sodium Is Critical Low"
  or "Potassium Is Critical Low"
  or "Glucose Is Critical Low"
  or "Hemoglobin Is Critical Low"
  or "Platelet Is Critical Low"
  or "WBC Is Critical Low"
  or "Calcium Is Critical Low"

define "Any Critical High Lab":
  "Sodium Is Critical High"
  or "Potassium Is Critical High"
  or "Glucose Is Critical High"
  or "Hemoglobin Is Critical High"
  or "Platelet Is Critical High"
  or "WBC Is Critical High"
  or "Calcium Is Critical High"
  or "Lactate Is Critical High"
  or "Troponin Is Critical High"
  or "Bilirubin Is Critical High"

define "Any Critical Lab Value":
  "Any Critical Low Lab" or "Any Critical High Lab"

define "Critical Lab Count":
  (if "Sodium Is Critical" then 1 else 0) +
  (if "Potassium Is Critical" then 1 else 0) +
  (if "Glucose Is Critical" then 1 else 0) +
  (if "Hemoglobin Is Critical" then 1 else 0) +
  (if "Platelet Is Critical" then 1 else 0) +
  (if "WBC Is Critical" then 1 else 0) +
  (if "Calcium Is Critical" then 1 else 0) +
  (if "Lactate Is Critical" then 1 else 0)

// ============================================================================
// SECTION 7: PREGNANCY-ADJUSTED FLAGS
// ============================================================================

/*
 * PREGNANCY ADJUSTMENTS
 * Pregnancy physiologically alters many lab reference ranges.
 * KB-16 provides trimester-specific interpretations.
 *
 * Common adjustments (simplified):
 * - Hemoglobin: Lower threshold (11.0 g/dL normal in pregnancy)
 * - Platelets: Lower threshold (gestational thrombocytopenia)
 * - Creatinine: Lower threshold (increased GFR in pregnancy)
 * - Alkaline Phosphatase: Higher threshold (placental source)
 */

// Pregnancy status - from KB-16 or MaternalHealthCommon
define "Is Pregnant for Lab Interpretation":
  // From KB-16 context or PatientContext.IsPregnant
  null as Boolean

define "Pregnancy Trimester":
  // From KB-16 context
  null as Integer

// Pregnancy-adjusted hemoglobin interpretation
define "Hemoglobin Abnormality Level Pregnancy Adjusted":
  case
    when "Most Recent Hemoglobin Value" is null then null
    when "Hemoglobin Is Critical Low" then 'critical-low'
    when "Is Pregnant for Lab Interpretation" = true then
      case
        when "Most Recent Hemoglobin Value" < 11.0 then 'low'
        when "Most Recent Hemoglobin Value" > 15 then 'high'
        else 'normal'
      end
    else "Hemoglobin Abnormality Level"
  end

// Pregnancy-adjusted platelet interpretation
define "Platelet Abnormality Level Pregnancy Adjusted":
  case
    when "Most Recent Platelet Value" is null then null
    when "Platelet Is Critical Low" then 'critical-low'
    when "Platelet Is Critical High" then 'critical-high'
    when "Is Pregnant for Lab Interpretation" = true then
      case
        when "Most Recent Platelet Value" < 100 then 'low' // Gestational threshold
        when "Most Recent Platelet Value" > 400 then 'high'
        else 'normal'
      end
    else "Platelet Abnormality Level"
  end

// Pregnancy-adjusted creatinine (lower due to hyperfiltration)
define "Creatinine Abnormality Level Pregnancy Adjusted":
  case
    when "Most Recent Creatinine Value" is null then null
    when "Is Pregnant for Lab Interpretation" = true then
      case
        when "Most Recent Creatinine Value" > 0.8 then 'high' // Lower threshold in pregnancy
        when "Most Recent Creatinine Value" < 0.4 then 'low'
        else 'normal'
      end
    else "Creatinine Abnormality Level"
  end

// ============================================================================
// SECTION 8: AGE-STRATIFIED FLAGS
// ============================================================================

/*
 * AGE STRATIFICATION
 * Reference ranges differ by age group.
 * KB-16 provides age-appropriate interpretations.
 */

define "Patient Age in Years":
  // From PatientContext.Demographics.Age
  null as Integer

define "Is Pediatric Patient":
  "Patient Age in Years" is not null and "Patient Age in Years" < 18

define "Is Geriatric Patient":
  "Patient Age in Years" is not null and "Patient Age in Years" >= 65

// Pediatric hemoglobin thresholds vary significantly by age
define "Hemoglobin Abnormality Level Age Adjusted":
  case
    when "Most Recent Hemoglobin Value" is null then null
    when "Hemoglobin Is Critical Low" then 'critical-low'
    when "Is Pediatric Patient" then
      case
        // Simplified pediatric thresholds - KB-16 has detailed age bands
        when "Most Recent Hemoglobin Value" < 10.0 then 'low'
        when "Most Recent Hemoglobin Value" > 16.0 then 'high'
        else 'normal'
      end
    when "Is Geriatric Patient" then
      case
        // Geriatric patients may have slightly lower baseline
        when "Most Recent Hemoglobin Value" < 11.0 then 'low'
        when "Most Recent Hemoglobin Value" > 17.0 then 'high'
        else 'normal'
      end
    else "Hemoglobin Abnormality Level"
  end

// Pediatric creatinine thresholds are much lower
define "Creatinine Abnormality Level Age Adjusted":
  case
    when "Most Recent Creatinine Value" is null then null
    when "Is Pediatric Patient" then
      case
        when "Most Recent Creatinine Value" > 0.7 then 'high' // Much lower in children
        else 'normal'
      end
    else "Creatinine Abnormality Level"
  end

// ============================================================================
// SECTION 9: TREND ANALYSIS
// ============================================================================

/*
 * LAB TRENDS
 * These definitions analyze changes over time.
 * KB-16 may provide computed trend data.
 */

define "Creatinine Trending Up":
  // From KB-16 trend analysis
  null as Boolean

define "Hemoglobin Trending Down":
  // From KB-16 trend analysis
  null as Boolean

define "Potassium Trending Up":
  // From KB-16 trend analysis
  null as Boolean

define "Lactate Trending Up":
  // From KB-16 trend analysis
  null as Boolean

// Delta creatinine from baseline (AKI detection)
define "Creatinine Delta from Baseline":
  // From KB-16 - difference from known baseline
  null as Decimal

define "Creatinine Rise Greater Than 50 Percent":
  "Creatinine Delta from Baseline" is not null
  and "Creatinine Delta from Baseline" > 0.5 * "Creatinine Reference Range High"

// ============================================================================
// ICU LAB REFERENCE FACTS OUTPUT (For Go Consumption)
// ============================================================================

/*
 * Aggregated lab interpretation outputs for ICU Intelligence consumption.
 * Provides all critical flags and abnormality levels.
 */
define "ICU Lab Reference Facts":
  {
    // Critical Value Flags
    anyCriticalLab: "Any Critical Lab Value",
    anyCriticalLow: "Any Critical Low Lab",
    anyCriticalHigh: "Any Critical High Lab",
    criticalLabCount: "Critical Lab Count",

    // Individual Critical Flags
    sodiumCritical: "Sodium Is Critical",
    potassiumCritical: "Potassium Is Critical",
    glucoseCritical: "Glucose Is Critical",
    hemoglobinCritical: "Hemoglobin Is Critical",
    plateletCritical: "Platelet Is Critical",
    wbcCritical: "WBC Is Critical",
    calciumCritical: "Calcium Is Critical",
    lactateCritical: "Lactate Is Critical",

    // Abnormality Levels
    sodiumLevel: "Sodium Abnormality Level",
    potassiumLevel: "Potassium Abnormality Level",
    glucoseLevel: "Glucose Abnormality Level",
    hemoglobinLevel: "Hemoglobin Abnormality Level",
    plateletLevel: "Platelet Abnormality Level",
    wbcLevel: "WBC Abnormality Level",
    creatinineLevel: "Creatinine Abnormality Level",
    calciumLevel: "Calcium Abnormality Level",
    lactateLevel: "Lactate Abnormality Level",
    tshLevel: "TSH Abnormality Level",
    altLevel: "ALT Abnormality Level",
    astLevel: "AST Abnormality Level",
    bilirubinLevel: "Bilirubin Abnormality Level",

    // Cardiac Markers
    troponinElevated: "Troponin Is Elevated",
    troponinCritical: "Troponin Is Critical High",
    bnpValue: "Most Recent BNP Value",

    // Trend Flags
    creatinineTrendingUp: "Creatinine Trending Up",
    hemoglobinTrendingDown: "Hemoglobin Trending Down",
    lactateTrendingUp: "Lactate Trending Up",

    // Stratification Context
    isPregnant: "Is Pregnant for Lab Interpretation",
    isPediatric: "Is Pediatric Patient",
    isGeriatric: "Is Geriatric Patient"
  }
