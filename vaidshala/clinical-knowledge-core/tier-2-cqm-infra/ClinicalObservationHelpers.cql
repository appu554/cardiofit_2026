/*
 * ==============================================================================
 * Clinical Observation Helpers Library
 * ==============================================================================
 *
 * Purpose: Provides convenience functions for accessing specific clinical
 *          observations by their LOINC codes. These are domain-specific
 *          wrappers around the generic Tier-1 ObservationHelpers functions.
 *
 * Tier: 2 (CQM Infrastructure)
 *
 * GOVERNANCE: This library contains CLINICAL MEANING (specific lab/vitals).
 *             Changes require Clinical + Technical review.
 *
 * RATIONALE: Functions referencing specific clinical tests (HbA1c, BP, eGFR)
 *            belong in Tier 2+, not Tier 0-1. This preserves the Grammar Lock
 *            principle where Tier 0-1 is pure grammar (accepts codes as params).
 *
 * DEPENDENCIES:
 *   - FHIRHelpers (Tier 0)
 *   - ObservationHelpers (Tier 1) - generic observation functions
 *   - CommonClinicalCodes (Tier 2) - LOINC code definitions
 *
 * ==============================================================================
 */

library ClinicalObservationHelpers version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.4.000' called FHIRHelpers
include ObservationHelpers version '1.0.0' called ObsHelpers
include CommonClinicalCodes version '1.0.0' called Codes

context Patient

// ============================================================
// BLOOD PRESSURE HELPERS
// ============================================================

/*
 * Get most recent systolic blood pressure value (mmHg)
 */
define function MostRecentSystolicBP(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."Systolic Blood Pressure")

/*
 * Get most recent diastolic blood pressure value (mmHg)
 */
define function MostRecentDiastolicBP(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."Diastolic Blood Pressure")

/*
 * Get blood pressure as a tuple (systolic, diastolic) from panel observation
 */
define function GetBloodPressure(observation Observation):
  Tuple {
    systolic: ObsHelpers.ComponentNumericValue(observation, Codes."Systolic Blood Pressure"),
    diastolic: ObsHelpers.ComponentNumericValue(observation, Codes."Diastolic Blood Pressure")
  }

/*
 * Get all blood pressure observations within a period
 */
define function BloodPressureObservations(observations List<Observation>, period Interval<DateTime>):
  ObsHelpers.ObservationsByCodeAndPeriod(observations, Codes."Blood Pressure Panel", period)

// ============================================================
// VITAL SIGNS HELPERS
// ============================================================

/*
 * Get most recent heart rate (bpm)
 */
define function MostRecentHeartRate(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."Heart Rate")

/*
 * Get most recent respiratory rate (breaths/min)
 */
define function MostRecentRespiratoryRate(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."Respiratory Rate")

/*
 * Get most recent body temperature (Celsius)
 */
define function MostRecentTemperature(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."Body Temperature")

/*
 * Get most recent oxygen saturation (%)
 */
define function MostRecentSpO2(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."Oxygen Saturation")

/*
 * Get most recent BMI (kg/m²)
 */
define function MostRecentBMI(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."BMI")

/*
 * Get most recent body weight (kg)
 */
define function MostRecentWeight(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."Body Weight")

/*
 * Get most recent body height (cm)
 */
define function MostRecentHeight(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."Body Height")

// ============================================================
// DIABETES / CARDIOMETABOLIC HELPERS
// ============================================================

/*
 * Get most recent HbA1c value (%)
 */
define function MostRecentHbA1c(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."HbA1c")

/*
 * Get most recent fasting glucose (mg/dL)
 */
define function MostRecentFastingGlucose(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."Fasting Glucose")

/*
 * Get most recent random glucose (mg/dL)
 */
define function MostRecentRandomGlucose(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."Random Glucose")

// ============================================================
// RENAL FUNCTION HELPERS
// ============================================================

/*
 * Get most recent eGFR (mL/min/1.73m²)
 */
define function MostRecentEGFR(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."eGFR")

/*
 * Get most recent creatinine (mg/dL)
 */
define function MostRecentCreatinine(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."Creatinine")

/*
 * Get most recent BUN (mg/dL)
 */
define function MostRecentBUN(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."BUN")

/*
 * Get most recent UACR (mg/g)
 */
define function MostRecentUACR(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."UACR")

// ============================================================
// LIPID PANEL HELPERS
// ============================================================

/*
 * Get most recent total cholesterol (mg/dL)
 */
define function MostRecentTotalCholesterol(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."Total Cholesterol")

/*
 * Get most recent LDL cholesterol (mg/dL)
 */
define function MostRecentLDL(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."LDL Cholesterol")

/*
 * Get most recent HDL cholesterol (mg/dL)
 */
define function MostRecentHDL(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."HDL Cholesterol")

/*
 * Get most recent triglycerides (mg/dL)
 */
define function MostRecentTriglycerides(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."Triglycerides")

// ============================================================
// HEMATOLOGY HELPERS
// ============================================================

/*
 * Get most recent hemoglobin (g/dL)
 */
define function MostRecentHemoglobin(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."Hemoglobin")

/*
 * Get most recent platelet count (K/uL)
 */
define function MostRecentPlatelets(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."Platelets")

/*
 * Get most recent WBC count (K/uL)
 */
define function MostRecentWBC(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."WBC")

// ============================================================
// ELECTROLYTE HELPERS
// ============================================================

/*
 * Get most recent potassium (mEq/L)
 */
define function MostRecentPotassium(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."Potassium")

/*
 * Get most recent sodium (mEq/L)
 */
define function MostRecentSodium(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."Sodium")

// ============================================================
// LIVER FUNCTION HELPERS
// ============================================================

/*
 * Get most recent ALT (U/L)
 */
define function MostRecentALT(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."ALT")

/*
 * Get most recent AST (U/L)
 */
define function MostRecentAST(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."AST")

/*
 * Get most recent total bilirubin (mg/dL)
 */
define function MostRecentBilirubin(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."Total Bilirubin")

/*
 * Get most recent albumin (g/dL)
 */
define function MostRecentAlbumin(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."Albumin")

/*
 * Get most recent INR
 */
define function MostRecentINR(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."INR")

// ============================================================
// SEPSIS / CRITICAL CARE HELPERS
// ============================================================

/*
 * Get most recent lactate (mmol/L)
 */
define function MostRecentLactate(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."Lactate")

/*
 * Get most recent procalcitonin (ng/mL)
 */
define function MostRecentProcalcitonin(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."Procalcitonin")

/*
 * Get most recent CRP (mg/L)
 */
define function MostRecentCRP(observations List<Observation>):
  ObsHelpers.MostRecentNumericValueByCode(observations, Codes."CRP")
