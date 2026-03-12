/*
 * QualityMeasures Library v1.0.0
 *
 * Common CQL definitions for quality measure calculations.
 * This library provides reusable expressions for:
 *   - Population criteria (initial, denominator, numerator)
 *   - Common clinical conditions
 *   - Measurement period utilities
 *
 * 🔴 CRITICAL: All evaluations are BATCH - never per-patient calls
 */

library QualityMeasures version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1' called FHIRHelpers

// Value Sets (OIDs from VSAC/NLM)
valueset "Diabetes Diagnoses": 'urn:oid:2.16.840.1.113883.3.464.1003.103.12.1001'
valueset "HbA1c Lab Test": 'urn:oid:2.16.840.1.113883.3.464.1003.198.12.1013'
valueset "Annual Wellness Visit": 'urn:oid:2.16.840.1.113883.3.526.3.1240'
valueset "Office Visit": 'urn:oid:2.16.840.1.113883.3.464.1003.101.12.1001'
valueset "Preventive Care Services": 'urn:oid:2.16.840.1.113883.3.464.1003.101.12.1027'
valueset "Hospice Care": 'urn:oid:2.16.840.1.113883.3.464.1003.1048'
valueset "Palliative Care": 'urn:oid:2.16.840.1.113883.3.464.1003.1090'
valueset "ESRD Diagnosis": 'urn:oid:2.16.840.1.113883.3.464.1003.109.12.1028'
valueset "Hypertension Diagnoses": 'urn:oid:2.16.840.1.113883.3.464.1003.104.12.1011'
valueset "Blood Pressure Measurement": 'urn:oid:2.16.840.1.113883.3.526.3.1032'

// LOINC codes
code "HbA1c LOINC": '4548-4' from "LOINC" display 'Hemoglobin A1c/Hemoglobin.total in Blood'
code "Systolic BP LOINC": '8480-6' from "LOINC" display 'Systolic blood pressure'
code "Diastolic BP LOINC": '8462-4' from "LOINC" display 'Diastolic blood pressure'

// Code systems
codesystem "LOINC": 'http://loinc.org'
codesystem "SNOMEDCT": 'http://snomed.info/sct'
codesystem "ICD10CM": 'http://hl7.org/fhir/sid/icd-10-cm'

// Parameters
parameter "Measurement Period" Interval<DateTime>

// Context
context Patient

// ============================================================================
// COMMON FUNCTIONS
// ============================================================================

/*
 * Get patient age at start of measurement period
 */
define "AgeInYearsAtStart":
  AgeInYearsAt(start of "Measurement Period")

/*
 * Check if patient had qualifying encounter during measurement period
 */
define "Qualifying Encounters":
  [Encounter: "Office Visit"] union
  [Encounter: "Annual Wellness Visit"] union
  [Encounter: "Preventive Care Services"]

define "Has Qualifying Encounter":
  exists ("Qualifying Encounters" E
    where E.period during "Measurement Period"
    and E.status = 'finished')

// ============================================================================
// EXCLUSION CRITERIA
// ============================================================================

/*
 * Hospice exclusion - patients in hospice care
 */
define "In Hospice":
  exists ([Encounter: "Hospice Care"] H
    where H.period overlaps "Measurement Period")
  or exists ([ServiceRequest: "Hospice Care"] SR
    where SR.authoredOn during "Measurement Period"
    and SR.status in {'active', 'completed'})

/*
 * Palliative care exclusion
 */
define "Receiving Palliative Care":
  exists ([Encounter: "Palliative Care"] P
    where P.period overlaps "Measurement Period")

/*
 * ESRD exclusion
 */
define "Has ESRD":
  exists ([Condition: "ESRD Diagnosis"] C
    where C.clinicalStatus ~ 'active'
    and C.onset before end of "Measurement Period")

/*
 * Standard denominator exclusions (HEDIS/CMS common)
 */
define "Standard Denominator Exclusions":
  "In Hospice" or "Receiving Palliative Care" or "Has ESRD"

// ============================================================================
// DIABETES MEASURES COMMON
// ============================================================================

/*
 * Diabetes diagnosis
 */
define "Has Diabetes":
  exists ([Condition: "Diabetes Diagnoses"] D
    where D.clinicalStatus ~ 'active'
    and D.onset before end of "Measurement Period")

/*
 * Most recent HbA1c result
 */
define "Most Recent HbA1c":
  Last([Observation: "HbA1c Lab Test"] O
    where O.effective during "Measurement Period"
    and O.status in {'final', 'amended'}
    sort by effective)

define "Most Recent HbA1c Value":
  ("Most Recent HbA1c".value as Quantity).value

/*
 * HbA1c control thresholds
 */
define "HbA1c Less Than 8":
  "Most Recent HbA1c Value" < 8.0

define "HbA1c Less Than 7":
  "Most Recent HbA1c Value" < 7.0

define "HbA1c Greater Than 9":
  "Most Recent HbA1c Value" > 9.0

// ============================================================================
// HYPERTENSION/CARDIOVASCULAR COMMON
// ============================================================================

/*
 * Hypertension diagnosis
 */
define "Has Hypertension":
  exists ([Condition: "Hypertension Diagnoses"] H
    where H.clinicalStatus ~ 'active'
    and H.onset before end of "Measurement Period")

/*
 * Most recent blood pressure
 */
define "Most Recent BP Observation":
  Last([Observation: "Blood Pressure Measurement"] BP
    where BP.effective during "Measurement Period"
    and BP.status in {'final', 'amended'}
    sort by effective)

define "Most Recent Systolic BP":
  (singleton from (
    "Most Recent BP Observation".component C
    where C.code ~ "Systolic BP LOINC"
  ).value as Quantity).value

define "Most Recent Diastolic BP":
  (singleton from (
    "Most Recent BP Observation".component C
    where C.code ~ "Diastolic BP LOINC"
  ).value as Quantity).value

/*
 * Blood pressure control
 */
define "BP Controlled Less Than 140/90":
  "Most Recent Systolic BP" < 140 and "Most Recent Diastolic BP" < 90

define "BP Controlled Less Than 130/80":
  "Most Recent Systolic BP" < 130 and "Most Recent Diastolic BP" < 80

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

/*
 * Check if observation exists in measurement period
 */
define function HasObservationDuringPeriod(observations List<Observation>):
  exists (observations O
    where O.effective during "Measurement Period"
    and O.status in {'final', 'amended'})

/*
 * Check if condition is active
 */
define function IsActiveCondition(condition Condition):
  condition.clinicalStatus ~ 'active'
  and (condition.abatement is null or condition.abatement after end of "Measurement Period")

/*
 * Get most recent value from observations
 */
define function MostRecentValue(observations List<Observation>):
  (Last(observations O
    where O.effective during "Measurement Period"
    sort by effective).value as Quantity).value
