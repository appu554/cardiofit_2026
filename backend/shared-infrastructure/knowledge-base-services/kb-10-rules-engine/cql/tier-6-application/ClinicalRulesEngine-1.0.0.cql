/*
 * ClinicalRulesEngine CQL Library
 * Version: 1.0.0
 * Purpose: Core clinical rules evaluation functions for KB-10 Rules Engine
 *
 * This library provides reusable CQL expressions for clinical decision support
 * integrated with the KB-10 YAML-based rules engine via Vaidshala CQL Engine.
 */
library ClinicalRulesEngine version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1'

// Code Systems
codesystem "LOINC": 'http://loinc.org'
codesystem "SNOMED": 'http://snomed.info/sct'
codesystem "RxNorm": 'http://www.nlm.nih.gov/research/umls/rxnorm'
codesystem "ICD10CM": 'http://hl7.org/fhir/sid/icd-10-cm'

// Value Sets - Laboratory
valueset "Potassium Lab": 'http://cardiofit.org/fhir/ValueSet/potassium-labs'
valueset "Glucose Lab": 'http://cardiofit.org/fhir/ValueSet/glucose-labs'
valueset "Creatinine Lab": 'http://cardiofit.org/fhir/ValueSet/creatinine-labs'
valueset "Hemoglobin Lab": 'http://cardiofit.org/fhir/ValueSet/hemoglobin-labs'
valueset "Troponin Lab": 'http://cardiofit.org/fhir/ValueSet/troponin-labs'
valueset "Lactate Lab": 'http://cardiofit.org/fhir/ValueSet/lactate-labs'
valueset "WBC Lab": 'http://cardiofit.org/fhir/ValueSet/wbc-labs'
valueset "BNP Lab": 'http://cardiofit.org/fhir/ValueSet/bnp-labs'
valueset "HbA1c Lab": 'http://cardiofit.org/fhir/ValueSet/hba1c-labs'
valueset "INR Lab": 'http://cardiofit.org/fhir/ValueSet/inr-labs'

// Value Sets - Vital Signs
valueset "Heart Rate": 'http://cardiofit.org/fhir/ValueSet/heart-rate'
valueset "Blood Pressure Systolic": 'http://cardiofit.org/fhir/ValueSet/bp-systolic'
valueset "Blood Pressure Diastolic": 'http://cardiofit.org/fhir/ValueSet/bp-diastolic'
valueset "Respiratory Rate": 'http://cardiofit.org/fhir/ValueSet/respiratory-rate'
valueset "Oxygen Saturation": 'http://cardiofit.org/fhir/ValueSet/oxygen-saturation'
valueset "Temperature": 'http://cardiofit.org/fhir/ValueSet/temperature'

// Value Sets - Conditions
valueset "Sepsis Conditions": 'http://cardiofit.org/fhir/ValueSet/sepsis-conditions'
valueset "Heart Failure Conditions": 'http://cardiofit.org/fhir/ValueSet/heart-failure-conditions'
valueset "Diabetes Conditions": 'http://cardiofit.org/fhir/ValueSet/diabetes-conditions'
valueset "CKD Conditions": 'http://cardiofit.org/fhir/ValueSet/ckd-conditions'

// Value Sets - Medications
valueset "High Alert Medications": 'http://cardiofit.org/fhir/ValueSet/high-alert-meds'
valueset "Anticoagulant Medications": 'http://cardiofit.org/fhir/ValueSet/anticoagulant-meds'
valueset "Opioid Medications": 'http://cardiofit.org/fhir/ValueSet/opioid-meds'

// Parameters
parameter "Measurement Period" Interval<DateTime> default Interval[@2024-01-01T00:00:00, @2024-12-31T23:59:59]

// Context
context Patient

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

/*
 * Get most recent observation value
 */
define function MostRecentValue(observations List<Observation>):
  Last(
    observations O
      where O.status = 'final'
      sort by effective.value
  ).value as Quantity

/*
 * Get observations within time window
 */
define function ObservationsWithinDays(observations List<Observation>, days Integer):
  observations O
    where O.effective.value >= Now() - (days * 1 day)
      and O.status = 'final'

/*
 * Check if patient has active condition
 */
define function HasActiveCondition(conditionValueSet ValueSet):
  exists(
    [Condition] C
      where C.code in conditionValueSet
        and C.clinicalStatus ~ 'active'
  )

/*
 * Check if patient is on medication class
 */
define function IsOnMedication(medicationValueSet ValueSet):
  exists(
    [MedicationRequest] MR
      where MR.medication in medicationValueSet
        and MR.status in {'active', 'completed'}
  )

/*
 * Calculate age in years
 */
define "Age in Years":
  AgeInYears()

// ============================================================================
// LABORATORY VALUE RETRIEVALS
// ============================================================================

define "Most Recent Potassium":
  MostRecentValue([Observation: "Potassium Lab"])

define "Most Recent Glucose":
  MostRecentValue([Observation: "Glucose Lab"])

define "Most Recent Creatinine":
  MostRecentValue([Observation: "Creatinine Lab"])

define "Most Recent Hemoglobin":
  MostRecentValue([Observation: "Hemoglobin Lab"])

define "Most Recent Troponin":
  MostRecentValue([Observation: "Troponin Lab"])

define "Most Recent Lactate":
  MostRecentValue([Observation: "Lactate Lab"])

define "Most Recent WBC":
  MostRecentValue([Observation: "WBC Lab"])

define "Most Recent BNP":
  MostRecentValue([Observation: "BNP Lab"])

define "Most Recent HbA1c":
  MostRecentValue([Observation: "HbA1c Lab"])

define "Most Recent INR":
  MostRecentValue([Observation: "INR Lab"])

// ============================================================================
// VITAL SIGN RETRIEVALS
// ============================================================================

define "Most Recent Heart Rate":
  MostRecentValue([Observation: "Heart Rate"])

define "Most Recent Systolic BP":
  MostRecentValue([Observation: "Blood Pressure Systolic"])

define "Most Recent Diastolic BP":
  MostRecentValue([Observation: "Blood Pressure Diastolic"])

define "Most Recent Respiratory Rate":
  MostRecentValue([Observation: "Respiratory Rate"])

define "Most Recent Oxygen Saturation":
  MostRecentValue([Observation: "Oxygen Saturation"])

define "Most Recent Temperature":
  MostRecentValue([Observation: "Temperature"])

// ============================================================================
// CLINICAL STATE ASSESSMENTS
// ============================================================================

/*
 * Renal Function Assessment
 */
define "eGFR Calculated":
  // CKD-EPI equation would be calculated here
  // For now, return creatinine-based estimate
  if "Most Recent Creatinine" is not null then
    141 * (
      if "Most Recent Creatinine" <= 0.9 'mg/dL' then
        Power(("Most Recent Creatinine" / 0.9 'mg/dL'), -0.411)
      else
        Power(("Most Recent Creatinine" / 0.9 'mg/dL'), -1.209)
    ) * Power(0.993, "Age in Years")
  else null

define "CKD Stage":
  case
    when "eGFR Calculated" >= 90 then 'Stage 1'
    when "eGFR Calculated" >= 60 then 'Stage 2'
    when "eGFR Calculated" >= 45 then 'Stage 3a'
    when "eGFR Calculated" >= 30 then 'Stage 3b'
    when "eGFR Calculated" >= 15 then 'Stage 4'
    else 'Stage 5'
  end

/*
 * SIRS Criteria Assessment
 */
define "SIRS Criteria Count":
  (if "Most Recent Temperature" > 38 'Cel' or "Most Recent Temperature" < 36 'Cel' then 1 else 0)
  + (if "Most Recent Heart Rate" > 90 '/min' then 1 else 0)
  + (if "Most Recent Respiratory Rate" > 20 '/min' then 1 else 0)
  + (if "Most Recent WBC" > 12000 '10*3/uL' or "Most Recent WBC" < 4000 '10*3/uL' then 1 else 0)

define "SIRS Criteria Met":
  "SIRS Criteria Count" >= 2

/*
 * Sepsis Assessment
 */
define "Suspected Sepsis":
  "SIRS Criteria Met"
    and (HasActiveCondition("Sepsis Conditions") or "Most Recent Lactate" > 2 'mmol/L')

/*
 * Heart Failure Assessment
 */
define "Heart Failure Exacerbation":
  "Most Recent BNP" > 400 'pg/mL'
    and "Most Recent Oxygen Saturation" < 92 '%'

// ============================================================================
// PATIENT RISK FLAGS
// ============================================================================

define "Elderly Patient":
  "Age in Years" >= 65

define "Pediatric Patient":
  "Age in Years" < 18

define "High Fall Risk":
  "Elderly Patient"
    and (IsOnMedication("Opioid Medications") or HasActiveCondition("CKD Conditions"))

define "On Anticoagulation":
  IsOnMedication("Anticoagulant Medications")

define "On High Alert Medication":
  IsOnMedication("High Alert Medications")

define "Has Diabetes":
  HasActiveCondition("Diabetes Conditions")
    or "Most Recent HbA1c" >= 6.5 '%'

define "Has Heart Failure":
  HasActiveCondition("Heart Failure Conditions")

define "Has CKD":
  HasActiveCondition("CKD Conditions")
    or "eGFR Calculated" < 60
