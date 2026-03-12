/*
 * CardiovascularMeasures Library v1.0.0
 *
 * CQL definitions for cardiovascular quality measures including:
 *   - CBP: Controlling High Blood Pressure
 *   - Statin Use in CVD
 *   - Aspirin Use
 *
 * Implements HEDIS/CMS measure specifications
 * 🔴 CRITICAL: All evaluations are BATCH - never per-patient calls
 */

library CardiovascularMeasures version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1' called FHIRHelpers
include QualityMeasures version '1.0.0' called QM

// Additional Value Sets for Cardiovascular Measures
valueset "Essential Hypertension": 'urn:oid:2.16.840.1.113883.3.464.1003.104.12.1011'
valueset "Ischemic Heart Disease": 'urn:oid:2.16.840.1.113883.3.464.1003.104.12.1003'
valueset "Atherosclerotic CVD": 'urn:oid:2.16.840.1.113762.1.4.1047.21'
valueset "Hyperlipidemia": 'urn:oid:2.16.840.1.113883.3.464.1003.198.12.1001'
valueset "LDL Cholesterol": 'urn:oid:2.16.840.1.113883.3.464.1003.198.12.1016'
valueset "Statin Therapy": 'urn:oid:2.16.840.1.113883.3.526.3.1003'
valueset "Aspirin or Antiplatelet": 'urn:oid:2.16.840.1.113883.3.464.1003.196.12.1211'
valueset "Pregnancy": 'urn:oid:2.16.840.1.113883.3.526.3.378'
valueset "Kidney Failure": 'urn:oid:2.16.840.1.113883.3.464.1003.109.12.1028'
valueset "Heart Failure": 'urn:oid:2.16.840.1.113883.3.464.1003.104.12.1002'
valueset "Myocardial Infarction": 'urn:oid:2.16.840.1.113883.3.526.3.403'

// LOINC Codes
code "LDL-C LOINC": '2089-1' from QM."LOINC" display 'Low Density Lipoprotein Cholesterol'
code "Total Cholesterol LOINC": '2093-3' from QM."LOINC" display 'Cholesterol [Mass/volume] in Serum or Plasma'

// Parameters
parameter "Measurement Period" Interval<DateTime>

context Patient

// ============================================================================
// CBP - CONTROLLING HIGH BLOOD PRESSURE
// HEDIS Measure / CMS165
// ============================================================================

/*
 * Initial Population: Ages 18-85 with hypertension diagnosis
 */
define "CBP Initial Population":
  QM."AgeInYearsAtStart" >= 18
  and QM."AgeInYearsAtStart" <= 85
  and "Has Essential Hypertension"
  and QM."Has Qualifying Encounter"

/*
 * Has essential hypertension diagnosis before end of measurement period
 */
define "Has Essential Hypertension":
  exists ([Condition: "Essential Hypertension"] H
    where H.clinicalStatus ~ 'active'
    and H.onset during Interval[start of "Measurement Period" - 1 year, end of "Measurement Period"])

/*
 * Denominator: Equals initial population
 */
define "CBP Denominator":
  "CBP Initial Population"

/*
 * Denominator Exclusions
 */
define "CBP Denominator Exclusion":
  QM."Standard Denominator Exclusions"
  or "Has Pregnancy During Period"
  or "Has Advanced CKD"

/*
 * Numerator: BP < 140/90 mmHg
 */
define "CBP Numerator":
  QM."BP Controlled Less Than 140/90"

/*
 * Alternative tighter control numerator: BP < 130/80
 */
define "CBP Numerator Tight Control":
  QM."BP Controlled Less Than 130/80"

// ============================================================================
// STATIN USE IN PERSONS WITH CVD
// CMS Measure / MIPS
// ============================================================================

/*
 * Initial Population: Ages 21-75 with clinical ASCVD
 */
define "Statin CVD Initial Population":
  QM."AgeInYearsAtStart" >= 21
  and QM."AgeInYearsAtStart" <= 75
  and "Has Clinical ASCVD"
  and QM."Has Qualifying Encounter"

/*
 * Clinical ASCVD includes IHD, prior MI, atherosclerotic disease
 */
define "Has Clinical ASCVD":
  exists ([Condition: "Ischemic Heart Disease"] IHD
    where IHD.clinicalStatus ~ 'active')
  or exists ([Condition: "Atherosclerotic CVD"] A
    where A.clinicalStatus ~ 'active')
  or exists ([Condition: "Myocardial Infarction"] MI
    where MI.onset before end of "Measurement Period")

/*
 * Denominator Exclusions
 */
define "Statin CVD Denominator Exclusion":
  QM."Standard Denominator Exclusions"
  or "Has Pregnancy During Period"
  or "Has Statin Allergy or Intolerance"

/*
 * On high-intensity statin therapy
 */
define "On Statin Therapy":
  exists ([MedicationRequest: "Statin Therapy"] M
    where M.authoredOn during "Measurement Period"
    and M.status in {'active', 'completed'})

define "Statin CVD Numerator":
  "On Statin Therapy"

// ============================================================================
// ASPIRIN USE FOR PRIMARY PREVENTION
// ACC/AHA Guideline Based
// ============================================================================

/*
 * Initial Population: Ages 40-70 with elevated CVD risk
 */
define "Aspirin Prevention Initial Population":
  QM."AgeInYearsAtStart" >= 40
  and QM."AgeInYearsAtStart" <= 70
  and "Has Elevated CVD Risk"
  and not "Has Bleeding Risk"
  and QM."Has Qualifying Encounter"

/*
 * Elevated CVD risk: diabetes, hypertension, hyperlipidemia
 */
define "Has Elevated CVD Risk":
  QM."Has Diabetes"
  or QM."Has Hypertension"
  or exists ([Condition: "Hyperlipidemia"] H
    where H.clinicalStatus ~ 'active')

/*
 * On aspirin or antiplatelet therapy
 */
define "On Aspirin Therapy":
  exists ([MedicationRequest: "Aspirin or Antiplatelet"] A
    where A.authoredOn during "Measurement Period"
    and A.status in {'active', 'completed'})

define "Aspirin Prevention Numerator":
  "On Aspirin Therapy"

// ============================================================================
// LDL CHOLESTEROL MANAGEMENT
// ============================================================================

/*
 * Most recent LDL-C value
 */
define "Most Recent LDL":
  Last([Observation: "LDL Cholesterol"] L
    where L.effective during "Measurement Period"
    and L.status in {'final', 'amended'}
    sort by effective)

define "Most Recent LDL Value":
  ("Most Recent LDL".value as Quantity).value

/*
 * LDL-C control thresholds
 */
define "LDL Less Than 100":
  "Most Recent LDL Value" < 100

define "LDL Less Than 70":
  "Most Recent LDL Value" < 70

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
 * Advanced CKD exclusion (Stage 5/ESRD)
 */
define "Has Advanced CKD":
  QM."Has ESRD"
  or exists ([Condition: "Kidney Failure"] KF
    where KF.clinicalStatus ~ 'active')

/*
 * Statin allergy/intolerance (placeholder - would need actual allergy valueset)
 */
define "Has Statin Allergy or Intolerance":
  false // Would check AllergyIntolerance resources

/*
 * Bleeding risk contraindication
 */
define "Has Bleeding Risk":
  false // Would check for bleeding disorders, anticoagulant use, etc.

/*
 * Heart failure diagnosis
 */
define "Has Heart Failure":
  exists ([Condition: "Heart Failure"] HF
    where HF.clinicalStatus ~ 'active')
