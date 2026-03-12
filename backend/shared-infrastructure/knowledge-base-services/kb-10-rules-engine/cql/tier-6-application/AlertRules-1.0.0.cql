/*
 * AlertRules CQL Library
 * Version: 1.0.0
 * Purpose: Critical alert detection rules for KB-10 Rules Engine
 *
 * This library provides CQL expressions for detecting critical clinical
 * conditions that require immediate attention and alerting.
 */
library AlertRules version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.0.1'
include ClinicalRulesEngine version '1.0.0' called CRE

// Parameters
parameter "Measurement Period" Interval<DateTime> default Interval[@2024-01-01T00:00:00, @2024-12-31T23:59:59]

// Context
context Patient

// ============================================================================
// CRITICAL LABORATORY ALERTS
// ============================================================================

/*
 * Critical Hyperkalemia Detection
 * Triggers when potassium level is >= 6.5 mEq/L
 * Risk: Cardiac arrhythmia, sudden death
 */
define "Critical Hyperkalemia":
  CRE."Most Recent Potassium" >= 6.5 'mEq/L'

define "Critical Hyperkalemia Alert":
  if "Critical Hyperkalemia" then
    {
      severity: 'CRITICAL',
      message: 'CRITICAL: Potassium ' + ToString(CRE."Most Recent Potassium") + ' - Risk of cardiac arrhythmia',
      action: 'STAT ECG, IV calcium gluconate, insulin/glucose, kayexalate'
    }
  else null

/*
 * Critical Hypokalemia Detection
 * Triggers when potassium level is < 2.5 mEq/L
 * Risk: Cardiac arrhythmia, muscle weakness
 */
define "Critical Hypokalemia":
  CRE."Most Recent Potassium" < 2.5 'mEq/L'

define "Critical Hypokalemia Alert":
  if "Critical Hypokalemia" then
    {
      severity: 'CRITICAL',
      message: 'CRITICAL: Potassium ' + ToString(CRE."Most Recent Potassium") + ' - Severe hypokalemia',
      action: 'IV potassium replacement, continuous cardiac monitoring'
    }
  else null

/*
 * Critical Hypoglycemia Detection
 * Triggers when glucose level is < 50 mg/dL
 * Risk: Altered consciousness, seizure, death
 */
define "Critical Hypoglycemia":
  CRE."Most Recent Glucose" < 50 'mg/dL'

define "Critical Hypoglycemia Alert":
  if "Critical Hypoglycemia" then
    {
      severity: 'CRITICAL',
      message: 'CRITICAL: Glucose ' + ToString(CRE."Most Recent Glucose") + ' - Severe hypoglycemia',
      action: 'D50W IV push, recheck glucose in 15 minutes'
    }
  else null

/*
 * Critical Hyperglycemia Detection
 * Triggers when glucose level is > 500 mg/dL
 * Risk: DKA, HHS, altered mental status
 */
define "Critical Hyperglycemia":
  CRE."Most Recent Glucose" > 500 'mg/dL'

define "Critical Hyperglycemia Alert":
  if "Critical Hyperglycemia" then
    {
      severity: 'HIGH',
      message: 'HIGH ALERT: Glucose ' + ToString(CRE."Most Recent Glucose") + ' - Evaluate for DKA/HHS',
      action: 'Check ketones, BMP, start IV fluids, consider insulin drip'
    }
  else null

/*
 * Elevated Troponin Detection
 * Triggers when troponin is above normal threshold
 * Risk: Acute MI, myocardial injury
 */
define "Elevated Troponin":
  CRE."Most Recent Troponin" > 0.04 'ng/mL'

define "Elevated Troponin Alert":
  if "Elevated Troponin" then
    {
      severity: 'HIGH',
      message: 'Elevated Troponin: ' + ToString(CRE."Most Recent Troponin") + ' - Evaluate for ACS',
      action: 'Serial troponins, ECG, cardiology consult'
    }
  else null

/*
 * Elevated Lactate Detection
 * Triggers when lactate is >= 4 mmol/L
 * Risk: Severe sepsis, shock, tissue hypoperfusion
 */
define "Critical Lactate":
  CRE."Most Recent Lactate" >= 4 'mmol/L'

define "Critical Lactate Alert":
  if "Critical Lactate" then
    {
      severity: 'CRITICAL',
      message: 'CRITICAL: Lactate ' + ToString(CRE."Most Recent Lactate") + ' - Severe tissue hypoperfusion',
      action: 'Sepsis workup, aggressive fluid resuscitation, consider vasopressors'
    }
  else null

/*
 * Critical Anemia Detection
 * Triggers when hemoglobin is < 7 g/dL
 * Risk: Symptomatic anemia, tissue hypoxia
 */
define "Critical Anemia":
  CRE."Most Recent Hemoglobin" < 7 'g/dL'

define "Critical Anemia Alert":
  if "Critical Anemia" then
    {
      severity: 'HIGH',
      message: 'Severe Anemia: Hemoglobin ' + ToString(CRE."Most Recent Hemoglobin"),
      action: 'Type and screen, consider transfusion, evaluate for bleeding source'
    }
  else null

// ============================================================================
// CRITICAL VITAL SIGN ALERTS
// ============================================================================

/*
 * Severe Hypotension Detection
 * Triggers when systolic BP < 80 mmHg
 * Risk: Shock, end-organ damage
 */
define "Severe Hypotension":
  CRE."Most Recent Systolic BP" < 80 'mm[Hg]'

define "Severe Hypotension Alert":
  if "Severe Hypotension" then
    {
      severity: 'CRITICAL',
      message: 'CRITICAL: SBP ' + ToString(CRE."Most Recent Systolic BP") + ' - Severe hypotension',
      action: 'IV fluid bolus, assess for shock etiology, consider vasopressors'
    }
  else null

/*
 * Hypertensive Crisis Detection
 * Triggers when systolic BP > 180 or diastolic > 120
 * Risk: Stroke, MI, aortic dissection
 */
define "Hypertensive Crisis":
  CRE."Most Recent Systolic BP" > 180 'mm[Hg]'
    or CRE."Most Recent Diastolic BP" > 120 'mm[Hg]'

define "Hypertensive Crisis Alert":
  if "Hypertensive Crisis" then
    {
      severity: 'HIGH',
      message: 'Hypertensive Crisis: BP ' + ToString(CRE."Most Recent Systolic BP") + '/' + ToString(CRE."Most Recent Diastolic BP"),
      action: 'Assess for end-organ damage, consider IV antihypertensive'
    }
  else null

/*
 * Severe Bradycardia Detection
 * Triggers when heart rate < 40 bpm
 * Risk: Syncope, cardiac arrest
 */
define "Severe Bradycardia":
  CRE."Most Recent Heart Rate" < 40 '/min'

define "Severe Bradycardia Alert":
  if "Severe Bradycardia" then
    {
      severity: 'HIGH',
      message: 'Severe Bradycardia: HR ' + ToString(CRE."Most Recent Heart Rate"),
      action: 'ECG, assess hemodynamic stability, atropine at bedside'
    }
  else null

/*
 * Severe Tachycardia Detection
 * Triggers when heart rate > 150 bpm
 * Risk: Hemodynamic instability, arrhythmia
 */
define "Severe Tachycardia":
  CRE."Most Recent Heart Rate" > 150 '/min'

define "Severe Tachycardia Alert":
  if "Severe Tachycardia" then
    {
      severity: 'HIGH',
      message: 'Severe Tachycardia: HR ' + ToString(CRE."Most Recent Heart Rate"),
      action: '12-lead ECG, assess for underlying cause, consider rate control'
    }
  else null

/*
 * Severe Hypoxemia Detection
 * Triggers when SpO2 < 88%
 * Risk: Respiratory failure, tissue hypoxia
 */
define "Severe Hypoxemia":
  CRE."Most Recent Oxygen Saturation" < 88 '%'

define "Severe Hypoxemia Alert":
  if "Severe Hypoxemia" then
    {
      severity: 'CRITICAL',
      message: 'CRITICAL: SpO2 ' + ToString(CRE."Most Recent Oxygen Saturation") + ' - Severe hypoxemia',
      action: 'Increase supplemental O2, assess airway, consider respiratory consult'
    }
  else null

/*
 * High Fever Detection
 * Triggers when temperature > 39.5°C
 * Risk: Severe infection, sepsis
 */
define "High Fever":
  CRE."Most Recent Temperature" > 39.5 'Cel'

define "High Fever Alert":
  if "High Fever" then
    {
      severity: 'MODERATE',
      message: 'High Fever: Temperature ' + ToString(CRE."Most Recent Temperature"),
      action: 'Blood cultures if not done, evaluate infection source, antipyretics'
    }
  else null

// ============================================================================
// COMPOSITE ALERT DEFINITIONS
// ============================================================================

/*
 * Any Critical Alert Present
 */
define "Has Critical Alert":
  "Critical Hyperkalemia"
    or "Critical Hypokalemia"
    or "Critical Hypoglycemia"
    or "Critical Lactate"
    or "Severe Hypotension"
    or "Severe Hypoxemia"

/*
 * Any High Priority Alert Present
 */
define "Has High Priority Alert":
  "Critical Hyperglycemia"
    or "Elevated Troponin"
    or "Critical Anemia"
    or "Hypertensive Crisis"
    or "Severe Bradycardia"
    or "Severe Tachycardia"

/*
 * Total Alert Count
 */
define "Active Alert Count":
  (if "Critical Hyperkalemia" then 1 else 0)
  + (if "Critical Hypokalemia" then 1 else 0)
  + (if "Critical Hypoglycemia" then 1 else 0)
  + (if "Critical Hyperglycemia" then 1 else 0)
  + (if "Elevated Troponin" then 1 else 0)
  + (if "Critical Lactate" then 1 else 0)
  + (if "Critical Anemia" then 1 else 0)
  + (if "Severe Hypotension" then 1 else 0)
  + (if "Hypertensive Crisis" then 1 else 0)
  + (if "Severe Bradycardia" then 1 else 0)
  + (if "Severe Tachycardia" then 1 else 0)
  + (if "Severe Hypoxemia" then 1 else 0)
  + (if "High Fever" then 1 else 0)
