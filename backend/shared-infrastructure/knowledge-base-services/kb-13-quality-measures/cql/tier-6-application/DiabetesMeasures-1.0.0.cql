/*
 * DiabetesMeasures Library v1.0.0
 *
 * CQL definitions for diabetes quality measures including:
 *   - HBD: Hemoglobin A1c Control (<8%)
 *   - CDC: Comprehensive Diabetes Care
 *   - Kidney Health Evaluation
 *
 * Implements HEDIS/CMS measure specifications
 * 🔴 CRITICAL: All evaluations are BATCH - never per-patient calls
 */

library DiabetesMeasures version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1' called FHIRHelpers
include QualityMeasures version '1.0.0' called QM

// Additional Value Sets for Diabetes Measures
valueset "Eye Exam": 'urn:oid:2.16.840.1.113883.3.464.1003.115.12.1088'
valueset "Diabetic Retinopathy": 'urn:oid:2.16.840.1.113883.3.464.1003.115.12.1090'
valueset "Nephropathy Screening": 'urn:oid:2.16.840.1.113883.3.464.1003.109.12.1024'
valueset "ACE/ARB Therapy": 'urn:oid:2.16.840.1.113883.3.526.3.1139'
valueset "Statin Therapy": 'urn:oid:2.16.840.1.113883.3.526.3.1003'
valueset "Pregnancy": 'urn:oid:2.16.840.1.113883.3.526.3.378'

// LOINC Codes
code "Urine Albumin Creatinine Ratio LOINC": '9318-7' from QM."LOINC" display 'Urine albumin/creatinine ratio'
code "eGFR LOINC": '48642-3' from QM."LOINC" display 'Glomerular filtration rate/1.73 sq M.predicted'

// Parameters
parameter "Measurement Period" Interval<DateTime>

context Patient

// ============================================================================
// HBD - HEMOGLOBIN A1C CONTROL FOR PATIENTS WITH DIABETES
// HEDIS Measure / CMS122
// ============================================================================

/*
 * Initial Population: Ages 18-75 with diabetes
 */
define "HBD Initial Population":
  QM."AgeInYearsAtStart" >= 18
  and QM."AgeInYearsAtStart" <= 75
  and QM."Has Diabetes"
  and QM."Has Qualifying Encounter"

/*
 * Denominator: Equals initial population
 */
define "HBD Denominator":
  "HBD Initial Population"

/*
 * Denominator Exclusions: Hospice, ESRD, Pregnancy
 */
define "HBD Denominator Exclusion":
  QM."Standard Denominator Exclusions"
  or "Has Pregnancy During Period"

/*
 * Numerator: HbA1c < 8%
 */
define "HBD Numerator":
  QM."HbA1c Less Than 8"

/*
 * Poor Control (inverse): HbA1c > 9% or no test
 */
define "HBD Poor Control":
  QM."HbA1c Greater Than 9"
  or QM."Most Recent HbA1c" is null

// ============================================================================
// CDC - COMPREHENSIVE DIABETES CARE (EYE EXAM)
// HEDIS Measure
// ============================================================================

/*
 * Initial Population: Ages 18-75 with diabetes
 */
define "CDC Eye Exam Initial Population":
  "HBD Initial Population"

/*
 * Eye exam in measurement period or prior year with no retinopathy
 */
define "Has Diabetic Retinal Exam":
  exists ([Procedure: "Eye Exam"] E
    where E.performed during "Measurement Period"
    and E.status = 'completed')
  or (
    exists ([Procedure: "Eye Exam"] E
      where E.performed during Interval[start of "Measurement Period" - 1 year, start of "Measurement Period"]
      and E.status = 'completed')
    and not exists ([Condition: "Diabetic Retinopathy"] DR
      where DR.clinicalStatus ~ 'active')
  )

define "CDC Eye Exam Numerator":
  "Has Diabetic Retinal Exam"

// ============================================================================
// KIDNEY HEALTH EVALUATION FOR DIABETES
// HEDIS Measure / CMS134
// ============================================================================

/*
 * Initial Population: Ages 18-85 with diabetes
 */
define "KED Initial Population":
  QM."AgeInYearsAtStart" >= 18
  and QM."AgeInYearsAtStart" <= 85
  and QM."Has Diabetes"
  and QM."Has Qualifying Encounter"

/*
 * Has kidney health evaluation (uACR + eGFR)
 */
define "Has UACR Test":
  exists ([Observation: "Urine Albumin Creatinine Ratio LOINC"] U
    where U.effective during "Measurement Period"
    and U.status in {'final', 'amended'})

define "Has eGFR Test":
  exists ([Observation: "eGFR LOINC"] G
    where G.effective during "Measurement Period"
    and G.status in {'final', 'amended'})

define "KED Numerator":
  "Has UACR Test" and "Has eGFR Test"

// ============================================================================
// STATIN THERAPY FOR DIABETES
// CMS Measure
// ============================================================================

/*
 * Initial Population: Ages 40-75 with diabetes
 */
define "Statin Therapy Initial Population":
  QM."AgeInYearsAtStart" >= 40
  and QM."AgeInYearsAtStart" <= 75
  and QM."Has Diabetes"
  and QM."Has Qualifying Encounter"

/*
 * Currently on statin therapy
 */
define "On Statin Therapy":
  exists ([MedicationRequest: "Statin Therapy"] M
    where M.authoredOn during "Measurement Period"
    and M.status in {'active', 'completed'})

define "Statin Therapy Numerator":
  "On Statin Therapy"

// ============================================================================
// HELPER DEFINITIONS
// ============================================================================

/*
 * Pregnancy exclusion
 */
define "Has Pregnancy During Period":
  exists ([Condition: "Pregnancy"] P
    where P.onset during "Measurement Period"
    or (P.onset before start of "Measurement Period"
        and (P.abatement is null or P.abatement during "Measurement Period")))

/*
 * Most recent eGFR value
 */
define "Most Recent eGFR Value":
  (Last([Observation: "eGFR LOINC"] G
    where G.effective during "Measurement Period"
    and G.status in {'final', 'amended'}
    sort by effective).value as Quantity).value

/*
 * CKD staging based on eGFR
 */
define "Has CKD Stage 3 or Higher":
  "Most Recent eGFR Value" < 60

/*
 * On ACE/ARB therapy (for nephropathy)
 */
define "On ACE ARB Therapy":
  exists ([MedicationRequest: "ACE/ARB Therapy"] M
    where M.authoredOn during "Measurement Period"
    and M.status in {'active', 'completed'})
