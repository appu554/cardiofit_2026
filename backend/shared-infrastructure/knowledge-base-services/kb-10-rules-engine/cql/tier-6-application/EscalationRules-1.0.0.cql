/*
 * EscalationRules CQL Library
 * Version: 1.0.0
 * Purpose: Clinical escalation pathway rules for KB-10 Rules Engine
 *
 * This library provides CQL expressions for determining when clinical
 * situations require escalation to higher levels of care or specialist teams.
 */
library EscalationRules version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1'
include ClinicalRulesEngine version '1.0.0' called CRE
include AlertRules version '1.0.0' called AR

// Parameters
parameter "Measurement Period" Interval<DateTime> default Interval[@2024-01-01T00:00:00, @2024-12-31T23:59:59]

// Context
context Patient

// ============================================================================
// RAPID RESPONSE TRIGGERS
// ============================================================================

/*
 * Rapid Response Team Criteria
 * Based on standard early warning score thresholds
 */
define "Rapid Response Criteria Met":
  CRE."Most Recent Heart Rate" > 130 '/min'
    or CRE."Most Recent Heart Rate" < 40 '/min'
    or CRE."Most Recent Respiratory Rate" > 30 '/min'
    or CRE."Most Recent Oxygen Saturation" < 88 '%'
    or CRE."Most Recent Systolic BP" < 80 'mm[Hg]'
    or "Acute Mental Status Change"

define "Acute Mental Status Change":
  exists(
    [Observation] O
      where O.code.coding.code = 'acute-mental-status-change'
        and O.value = true
        and O.effective.value >= Now() - 4 hours
  )

define "Rapid Response Escalation":
  if "Rapid Response Criteria Met" then
    {
      escalation_level: 'RAPID_RESPONSE_TEAM',
      urgency: 'STAT',
      recipients: {'rapid_response_team', 'attending_physician', 'charge_nurse'},
      channel: 'OVERHEAD_PAGE',
      message: 'RAPID RESPONSE: Acute clinical deterioration detected'
    }
  else null

// ============================================================================
// CODE TEAM TRIGGERS
// ============================================================================

/*
 * Code Blue Criteria
 * Cardiac or respiratory arrest
 */
define "Code Blue Criteria Met":
  CRE."Most Recent Heart Rate" = 0 '/min'
    or CRE."Most Recent Respiratory Rate" = 0 '/min'
    or "Patient Unresponsive"

define "Patient Unresponsive":
  exists(
    [Observation] O
      where O.code.coding.code = 'unresponsive'
        and O.value = true
        and O.effective.value >= Now() - 15 minutes
  )

define "Code Blue Escalation":
  if "Code Blue Criteria Met" then
    {
      escalation_level: 'CODE_TEAM',
      urgency: 'STAT',
      recipients: {'code_team', 'anesthesia', 'respiratory_therapy', 'icu_physician'},
      channel: 'OVERHEAD_PAGE',
      message: 'CODE BLUE: Cardiac/Respiratory Arrest'
    }
  else null

// ============================================================================
// SPECIALTY TEAM TRIGGERS
// ============================================================================

/*
 * Stroke Alert Criteria
 * Suspected acute stroke requiring intervention
 */
define "Stroke Alert Criteria":
  "NIHSS Score" >= 4 and "Symptom Onset Within Window"

define "NIHSS Score":
  Last(
    [Observation] O
      where O.code.coding.code = 'nihss-score'
        and O.status = 'final'
      sort by effective.value
  ).value as Integer

define "Symptom Onset Within Window":
  exists(
    [Observation] O
      where O.code.coding.code = 'symptom-onset-time'
        and O.value >= Now() - 24 hours
  )

define "Stroke Alert Escalation":
  if "Stroke Alert Criteria" then
    {
      escalation_level: 'STROKE_TEAM',
      urgency: 'STAT',
      recipients: {'stroke_team', 'neurology', 'radiology', 'er_physician'},
      channel: 'PAGER',
      message: 'STROKE ALERT: Suspected acute stroke, NIHSS ' + ToString("NIHSS Score")
    }
  else null

/*
 * STEMI Alert Criteria
 * ST-elevation myocardial infarction requiring PCI
 */
define "STEMI Alert Criteria":
  "STEMI ECG Criteria Met" or AR."Elevated Troponin"

define "STEMI ECG Criteria Met":
  exists(
    [Observation] O
      where O.code.coding.code = 'stemi-criteria'
        and O.value = true
        and O.effective.value >= Now() - 6 hours
  )

define "STEMI Alert Escalation":
  if "STEMI Alert Criteria" then
    {
      escalation_level: 'CATH_LAB',
      urgency: 'STAT',
      recipients: {'interventional_cardiology', 'cath_lab_team', 'cardiology_fellow'},
      channel: 'PAGER',
      message: 'STEMI ALERT: Activate cardiac catheterization lab'
    }
  else null

// ============================================================================
// SEPSIS BUNDLE ESCALATION
// ============================================================================

/*
 * Sepsis Bundle Trigger
 * Sepsis requiring 3-hour bundle initiation
 */
define "Sepsis Bundle Criteria":
  (CRE."Most Recent Lactate" > 2 'mmol/L' or AR."Severe Hypotension")
    and "Suspected Infection Present"

define "Suspected Infection Present":
  exists(
    [Condition] C
      where C.code in "Infection Conditions"
        and C.clinicalStatus ~ 'active'
  )
    or exists(
      [Observation] O
        where O.code.coding.code = 'suspected-infection'
          and O.value = true
    )

valueset "Infection Conditions": 'http://cardiofit.org/fhir/ValueSet/infection-conditions'

define "Sepsis Bundle Escalation":
  if "Sepsis Bundle Criteria" then
    {
      escalation_level: 'PHYSICIAN',
      urgency: 'URGENT',
      recipients: {'attending_physician', 'rapid_response_team'},
      channel: 'PAGER',
      message: 'SEPSIS ALERT: Initiate sepsis bundle, lactate ' + ToString(CRE."Most Recent Lactate"),
      bundle_requirements: {
        'Blood cultures x2 before antibiotics',
        'Broad-spectrum antibiotics within 1 hour',
        '30mL/kg crystalloid if hypotensive or lactate > 4'
      }
    }
  else null

// ============================================================================
// ICU TRANSFER CRITERIA
// ============================================================================

/*
 * ICU Transfer Recommendation
 * Patient requires higher level of care
 */
define "ICU Transfer Criteria":
  "High FiO2 Requirement"
    or "Vasopressor Required"
    or "Invasive Monitoring Needed"
    or "Severe Acidosis"

define "High FiO2 Requirement":
  exists(
    [Observation] O
      where O.code.coding.code = 'fio2'
        and O.value > 0.5
        and O.effective.value >= Now() - 4 hours
  )

define "Vasopressor Required":
  exists(
    [MedicationRequest] MR
      where MR.medication in "Vasopressor Medications"
        and MR.status = 'active'
  )

valueset "Vasopressor Medications": 'http://cardiofit.org/fhir/ValueSet/vasopressor-meds'

define "Invasive Monitoring Needed":
  exists(
    [Observation] O
      where O.code.coding.code = 'invasive-monitoring-needed'
        and O.value = true
  )

define "Severe Acidosis":
  exists(
    [Observation] O
      where O.code.coding.code in {'ph', 'arterial-ph'}
        and O.value < 7.25
        and O.effective.value >= Now() - 4 hours
  )

define "ICU Transfer Escalation":
  if "ICU Transfer Criteria" then
    {
      escalation_level: 'ICU_ATTENDING',
      urgency: 'URGENT',
      recipients: {'icu_attending', 'hospitalist', 'charge_nurse'},
      channel: 'PHONE',
      message: 'ICU TRANSFER RECOMMENDED: Patient requires higher level of care'
    }
  else null

// ============================================================================
// SPECIALTY CONSULT TRIGGERS
// ============================================================================

/*
 * Nephrology Consult Trigger
 * Acute kidney injury or severe CKD
 */
define "Nephrology Consult Criteria":
  "Severe AKI" or "ESRD Criteria"

define "Severe AKI":
  CRE."Most Recent Creatinine" >= (3 * "Baseline Creatinine")

define "Baseline Creatinine":
  First(
    [Observation: CRE."Creatinine Lab"] O
      where O.effective.value <= Now() - 90 days
        and O.status = 'final'
      sort by effective.value descending
  ).value as Quantity

define "ESRD Criteria":
  CRE."eGFR Calculated" < 15

define "Nephrology Consult Escalation":
  if "Nephrology Consult Criteria" then
    {
      escalation_level: 'NEPHROLOGY',
      urgency: 'ROUTINE',
      recipients: {'nephrology'},
      channel: 'EHR_MESSAGE',
      message: 'Nephrology consult recommended for acute kidney injury or ESRD'
    }
  else null

/*
 * Pharmacy Consult Trigger
 * Complex medication management
 */
define "Pharmacy Consult Criteria":
  "Polypharmacy Present" or "Major Drug Interaction" or "Anticoagulation Complexity"

define "Polypharmacy Present":
  Count([MedicationRequest] MR where MR.status = 'active') >= 10

define "Major Drug Interaction":
  exists(
    [DetectedIssue] DI
      where DI.severity = 'high'
        and DI.status = 'final'
  )

define "Anticoagulation Complexity":
  CRE."On Anticoagulation"
    and (CRE."Has CKD" or CRE."Most Recent INR" > 4)

define "Pharmacy Consult Escalation":
  if "Pharmacy Consult Criteria" then
    {
      escalation_level: 'PHARMACIST',
      urgency: 'ROUTINE',
      recipients: {'clinical_pharmacist'},
      channel: 'WORKLIST',
      message: 'Pharmacy consult recommended for complex medication management'
    }
  else null

// ============================================================================
// ESCALATION SUMMARY
// ============================================================================

/*
 * Highest Urgency Escalation Level
 */
define "Highest Escalation Urgency":
  case
    when "Code Blue Criteria Met" then 'CODE'
    when "Rapid Response Criteria Met" then 'STAT'
    when "Stroke Alert Criteria" or "STEMI Alert Criteria" then 'STAT'
    when "Sepsis Bundle Criteria" or "ICU Transfer Criteria" then 'URGENT'
    when "Nephrology Consult Criteria" or "Pharmacy Consult Criteria" then 'ROUTINE'
    else 'NONE'
  end

/*
 * Total Escalation Count
 */
define "Active Escalation Count":
  (if "Rapid Response Criteria Met" then 1 else 0)
  + (if "Code Blue Criteria Met" then 1 else 0)
  + (if "Stroke Alert Criteria" then 1 else 0)
  + (if "STEMI Alert Criteria" then 1 else 0)
  + (if "Sepsis Bundle Criteria" then 1 else 0)
  + (if "ICU Transfer Criteria" then 1 else 0)
  + (if "Nephrology Consult Criteria" then 1 else 0)
  + (if "Pharmacy Consult Criteria" then 1 else 0)
