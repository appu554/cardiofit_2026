// =============================================================================
// SepsisGuidelines.cql
// Tier 4b: Clinical Guidelines - Surviving Sepsis Campaign 2021
// =============================================================================
//
// PURPOSE:
// Implements Surviving Sepsis Campaign (SSC) 2021 international guidelines
// for management of sepsis and septic shock in adults.
//
// ICU DOMINANCE TRIGGER:
// Per CTO/CMO Directive: "Sepsis criteria AUTOMATICALLY activate ICU veto authority"
// When "Meets Sepsis Criteria" or "Meets Septic Shock Criteria" is true,
// ICU Intelligence has override authority over all other recommendations.
//
// KEY COMPONENTS:
// 1. Sepsis-3 Definition: Infection + organ dysfunction (SOFA ≥ 2)
// 2. qSOFA: Quick bedside screening (RR, mentation, BP)
// 3. Hour-1 Bundle: Immediate interventions for suspected sepsis
// 4. Fluid Resuscitation: 30 mL/kg crystalloid target
// 5. Vasopressor Support: MAP target ≥ 65 mmHg
//
// SOURCES:
// - Surviving Sepsis Campaign 2021 Guidelines (Intensive Care Med 2021)
// - Sepsis-3 Definitions (JAMA 2016)
// - SSC Bundle Updates 2018
// =============================================================================

library SepsisGuidelines version '1.0.0'

using FHIR version '4.0.1'

include "FHIRHelpers" version '4.0.1' called FHIRHelpers
include "ClinicalCalculators" version '1.0.0' called Calc

// =============================================================================
// CODESYSTEMS
// =============================================================================

codesystem "LOINC": 'http://loinc.org'
codesystem "SNOMED": 'http://snomed.info/sct'
codesystem "RxNorm": 'http://www.nlm.nih.gov/research/umls/rxnorm'
codesystem "ICD10": 'http://hl7.org/fhir/sid/icd-10-cm'

// Vital signs
code "Respiratory Rate": '9279-1' from "LOINC" display 'Respiratory rate'
code "Systolic BP": '8480-6' from "LOINC" display 'Systolic blood pressure'
code "Diastolic BP": '8462-4' from "LOINC" display 'Diastolic blood pressure'
code "MAP": '8478-0' from "LOINC" display 'Mean arterial pressure'
code "Heart Rate": '8867-4' from "LOINC" display 'Heart rate'
code "Temperature": '8310-5' from "LOINC" display 'Body temperature'
code "SpO2": '2708-6' from "LOINC" display 'Oxygen saturation'
code "GCS Total": '9269-2' from "LOINC" display 'Glasgow coma score total'

// Lab values
code "Lactate": '2524-7' from "LOINC" display 'Lactate'
code "Serum Creatinine": '2160-0' from "LOINC" display 'Creatinine'
code "Bilirubin Total": '1975-2' from "LOINC" display 'Bilirubin total'
code "Platelet Count": '777-3' from "LOINC" display 'Platelet count'
code "WBC": '6690-2' from "LOINC" display 'WBC count'
code "Procalcitonin": '75241-0' from "LOINC" display 'Procalcitonin'
code "CRP": '1988-5' from "LOINC" display 'C-reactive protein'
code "INR": '6301-6' from "LOINC" display 'INR'
code "PaO2": '2703-7' from "LOINC" display 'PaO2'
code "FiO2": '3150-0' from "LOINC" display 'FiO2'

// Urine output
code "Urine Output": '9192-6' from "LOINC" display 'Urine output'

// =============================================================================
// VALUESETS
// =============================================================================

// Infection-related conditions
valueset "Sepsis Diagnosis Codes": 'http://vaidshala.health/ValueSet/sepsis-diagnosis'
valueset "Suspected Infection Codes": 'http://vaidshala.health/ValueSet/suspected-infection'
valueset "Pneumonia Codes": 'http://vaidshala.health/ValueSet/pneumonia'
valueset "UTI Codes": 'http://vaidshala.health/ValueSet/uti'
valueset "Skin Soft Tissue Infection Codes": 'http://vaidshala.health/ValueSet/ssti'
valueset "Intra Abdominal Infection Codes": 'http://vaidshala.health/ValueSet/iab-infection'
valueset "CNS Infection Codes": 'http://vaidshala.health/ValueSet/cns-infection'
valueset "Bacteremia Codes": 'http://vaidshala.health/ValueSet/bacteremia'

// Broad spectrum antibiotics for sepsis
valueset "Empiric Sepsis Antibiotics": 'http://vaidshala.health/ValueSet/sepsis-antibiotics'

// Vasopressors
valueset "Vasopressor Medications": 'http://vaidshala.health/ValueSet/vasopressors'

// Crystalloid fluids
valueset "Crystalloid Fluids": 'http://vaidshala.health/ValueSet/crystalloids'

// =============================================================================
// PARAMETERS
// =============================================================================

parameter "Measurement Period" Interval<DateTime>
  default Interval[@2024-01-01T00:00:00.0, @2024-12-31T23:59:59.999]

// Time windows for sepsis assessment
parameter "qSOFA Window Hours" Integer default 6
parameter "SOFA Window Hours" Integer default 24
parameter "Lactate Window Hours" Integer default 6
parameter "Bundle Compliance Window Hours" Integer default 1

// =============================================================================
// CONTEXT
// =============================================================================

context Patient

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

define function GetLatestObservation(obsCode Code):
  Last(
    [Observation: obsCode] O
      where O.status in { 'final', 'amended', 'corrected' }
      sort by FHIRHelpers.ToDateTime(effective as dateTime)
  )

define function GetLatestValue(obsCode Code):
  (GetLatestObservation(obsCode).value as Quantity).value

define function GetRecentObservation(obsCode Code, hoursBack Integer):
  Last(
    [Observation: obsCode] O
      where O.status in { 'final', 'amended', 'corrected' }
        and FHIRHelpers.ToDateTime(O.effective as dateTime) >= Now() - (hoursBack hours)
      sort by FHIRHelpers.ToDateTime(effective as dateTime)
  )

define function GetRecentValue(obsCode Code, hoursBack Integer):
  (GetRecentObservation(obsCode, hoursBack).value as Quantity).value

// =============================================================================
// SUSPECTED INFECTION DETECTION
// =============================================================================

define "Active Infection Conditions":
  [Condition] C
    where C.clinicalStatus.coding[0].code = 'active'
      and (
        C.code in "Suspected Infection Codes"
        or C.code in "Pneumonia Codes"
        or C.code in "UTI Codes"
        or C.code in "Skin Soft Tissue Infection Codes"
        or C.code in "Intra Abdominal Infection Codes"
        or C.code in "CNS Infection Codes"
        or C.code in "Bacteremia Codes"
        or C.code in "Sepsis Diagnosis Codes"
      )

define "Has Suspected Infection":
  exists("Active Infection Conditions")

define "Infection Source":
  case
    when exists([Condition] C where C.code in "Pneumonia Codes" and C.clinicalStatus.coding[0].code = 'active') then 'Pneumonia'
    when exists([Condition] C where C.code in "UTI Codes" and C.clinicalStatus.coding[0].code = 'active') then 'Urinary Tract'
    when exists([Condition] C where C.code in "Skin Soft Tissue Infection Codes" and C.clinicalStatus.coding[0].code = 'active') then 'Skin/Soft Tissue'
    when exists([Condition] C where C.code in "Intra Abdominal Infection Codes" and C.clinicalStatus.coding[0].code = 'active') then 'Intra-abdominal'
    when exists([Condition] C where C.code in "CNS Infection Codes" and C.clinicalStatus.coding[0].code = 'active') then 'CNS'
    when exists([Condition] C where C.code in "Bacteremia Codes" and C.clinicalStatus.coding[0].code = 'active') then 'Bloodstream'
    when "Has Suspected Infection" then 'Unknown Source'
    else null
  end

// Antibiotic administration suggests suspected infection
define "Recent Antibiotic Orders":
  [MedicationRequest] MR
    where MR.status = 'active'
      and MR.medication in "Empiric Sepsis Antibiotics"
      and FHIRHelpers.ToDateTime(MR.authoredOn) >= Now() - 24 hours

define "Antibiotics Started Recently":
  exists("Recent Antibiotic Orders")

// =============================================================================
// qSOFA SCORE (QUICK SOFA)
// =============================================================================
// Bedside screening tool: RR ≥22, Altered mentation, SBP ≤100

define "Latest Respiratory Rate":
  GetRecentValue("Respiratory Rate", "qSOFA Window Hours")

define "Latest Systolic BP":
  GetRecentValue("Systolic BP", "qSOFA Window Hours")

define "Latest GCS":
  GetRecentValue("GCS Total", "qSOFA Window Hours")

define "qSOFA Respiratory Rate Point":
  if "Latest Respiratory Rate" >= 22 then 1 else 0

define "qSOFA Systolic BP Point":
  if "Latest Systolic BP" is not null and "Latest Systolic BP" <= 100 then 1 else 0

define "qSOFA Altered Mentation Point":
  if "Latest GCS" is not null and "Latest GCS" < 15 then 1 else 0

define "qSOFA Score":
  "qSOFA Respiratory Rate Point"
  + "qSOFA Systolic BP Point"
  + "qSOFA Altered Mentation Point"

define "qSOFA Positive":
  "qSOFA Score" >= 2

// =============================================================================
// SOFA SCORE (SEQUENTIAL ORGAN FAILURE ASSESSMENT)
// =============================================================================
// Full organ dysfunction assessment

// Respiration: PaO2/FiO2 ratio
define "Latest PaO2":
  GetRecentValue("PaO2", "SOFA Window Hours")

define "Latest FiO2":
  GetRecentValue("FiO2", "SOFA Window Hours")

define "PF Ratio":
  if "Latest PaO2" is not null and "Latest FiO2" is not null and "Latest FiO2" > 0 then
    "Latest PaO2" / "Latest FiO2" * 100
  else
    null

define "SOFA Respiration Score":
  case
    when "PF Ratio" is null then 0
    when "PF Ratio" >= 400 then 0
    when "PF Ratio" >= 300 then 1
    when "PF Ratio" >= 200 then 2
    when "PF Ratio" >= 100 then 3
    else 4
  end

// Coagulation: Platelets
define "Latest Platelets":
  GetRecentValue("Platelet Count", "SOFA Window Hours")

define "SOFA Coagulation Score":
  case
    when "Latest Platelets" is null then 0
    when "Latest Platelets" >= 150 then 0
    when "Latest Platelets" >= 100 then 1
    when "Latest Platelets" >= 50 then 2
    when "Latest Platelets" >= 20 then 3
    else 4
  end

// Liver: Bilirubin
define "Latest Bilirubin":
  GetRecentValue("Bilirubin Total", "SOFA Window Hours")

define "SOFA Liver Score":
  case
    when "Latest Bilirubin" is null then 0
    when "Latest Bilirubin" < 1.2 then 0
    when "Latest Bilirubin" < 2.0 then 1
    when "Latest Bilirubin" < 6.0 then 2
    when "Latest Bilirubin" < 12.0 then 3
    else 4
  end

// Cardiovascular: MAP or vasopressor requirement
define "Latest MAP":
  GetRecentValue("MAP", "SOFA Window Hours")

define "On Vasopressors":
  exists (
    [MedicationRequest] MR
      where MR.status = 'active'
        and MR.medication in "Vasopressor Medications"
  )

define "SOFA Cardiovascular Score":
  case
    when "On Vasopressors" then 3  // Simplified - full scoring requires dose
    when "Latest MAP" is not null and "Latest MAP" < 70 then 1
    else 0
  end

// CNS: Glasgow Coma Scale
define "SOFA CNS Score":
  case
    when "Latest GCS" is null then 0
    when "Latest GCS" = 15 then 0
    when "Latest GCS" >= 13 then 1
    when "Latest GCS" >= 10 then 2
    when "Latest GCS" >= 6 then 3
    else 4
  end

// Renal: Creatinine or urine output
define "Latest Creatinine":
  GetRecentValue("Serum Creatinine", "SOFA Window Hours")

define "Latest Urine Output":
  GetRecentValue("Urine Output", "SOFA Window Hours")

define "SOFA Renal Score":
  case
    when "Latest Creatinine" is null then 0
    when "Latest Creatinine" < 1.2 then 0
    when "Latest Creatinine" < 2.0 then 1
    when "Latest Creatinine" < 3.5 then 2
    when "Latest Creatinine" < 5.0 then 3
    else 4
  end

define "SOFA Total Score":
  "SOFA Respiration Score"
  + "SOFA Coagulation Score"
  + "SOFA Liver Score"
  + "SOFA Cardiovascular Score"
  + "SOFA CNS Score"
  + "SOFA Renal Score"

// =============================================================================
// SEPSIS CRITERIA (SEPSIS-3 DEFINITION)
// =============================================================================
// Sepsis = Suspected Infection + SOFA ≥ 2 (or increase of ≥ 2)

define "Meets Sepsis Criteria":
  "Has Suspected Infection" and "SOFA Total Score" >= 2

define "Meets qSOFA Screening":
  "Has Suspected Infection" and "qSOFA Positive"

// =============================================================================
// SEPTIC SHOCK CRITERIA
// =============================================================================
// Septic Shock = Sepsis + Vasopressors to maintain MAP ≥ 65 + Lactate > 2

define "Latest Lactate":
  GetRecentValue("Lactate", "Lactate Window Hours")

define "Lactate Elevated":
  "Latest Lactate" is not null and "Latest Lactate" > 2.0

define "MAP Below Target":
  "Latest MAP" is not null and "Latest MAP" < 65

define "Meets Septic Shock Criteria":
  "Meets Sepsis Criteria"
    and "On Vasopressors"
    and "Lactate Elevated"

// =============================================================================
// HOUR-1 BUNDLE COMPONENTS
// =============================================================================
// SSC 2018/2021: Complete within 1 hour of sepsis recognition

// 1. Measure Lactate
define "Lactate Measured Recently":
  GetRecentObservation("Lactate", "Bundle Compliance Window Hours") is not null

// 2. Obtain Blood Cultures
valueset "Blood Culture Orders": 'http://vaidshala.health/ValueSet/blood-culture'

define "Blood Cultures Ordered":
  exists (
    [ServiceRequest] SR
      where SR.code in "Blood Culture Orders"
        and SR.status in { 'active', 'completed' }
        and FHIRHelpers.ToDateTime(SR.authoredOn) >= Now() - 1 hours
  )

// 3. Administer Broad Spectrum Antibiotics
define "Antibiotics Administered":
  exists (
    [MedicationAdministration] MA
      where MA.medication in "Empiric Sepsis Antibiotics"
        and MA.status = 'completed'
        and FHIRHelpers.ToDateTime(MA.effective as dateTime) >= Now() - 1 hours
  )

// 4. Begin Rapid Fluid Resuscitation (if hypotension or lactate ≥ 4)
define "Fluid Resuscitation Indicated":
  "MAP Below Target" or ("Latest Lactate" is not null and "Latest Lactate" >= 4)

define "Crystalloid Fluids Administered":
  exists (
    [MedicationAdministration] MA
      where MA.medication in "Crystalloid Fluids"
        and MA.status = 'completed'
        and FHIRHelpers.ToDateTime(MA.effective as dateTime) >= Now() - 1 hours
  )

// 5. Apply Vasopressors (if hypotensive during/after fluid resuscitation)
define "Vasopressor Indication":
  "MAP Below Target" and "Fluid Resuscitation Indicated"

define "Vasopressors Applied":
  "On Vasopressors"

// =============================================================================
// HOUR-1 BUNDLE COMPLIANCE
// =============================================================================

define "Hour 1 Bundle Applicable":
  "Meets Sepsis Criteria" or "Meets qSOFA Screening"

define "Hour 1 Bundle Components":
  {
    lactate: if "Lactate Measured Recently" then 'COMPLETE' else 'PENDING',
    bloodCultures: if "Blood Cultures Ordered" then 'COMPLETE' else 'PENDING',
    antibiotics: if "Antibiotics Administered" then 'COMPLETE' else 'PENDING',
    fluids: if not "Fluid Resuscitation Indicated" then 'NOT_INDICATED'
            else if "Crystalloid Fluids Administered" then 'COMPLETE'
            else 'PENDING',
    vasopressors: if not "Vasopressor Indication" then 'NOT_INDICATED'
                  else if "Vasopressors Applied" then 'COMPLETE'
                  else 'PENDING'
  }

define "Hour 1 Bundle Complete":
  "Lactate Measured Recently"
    and "Blood Cultures Ordered"
    and "Antibiotics Administered"
    and (not "Fluid Resuscitation Indicated" or "Crystalloid Fluids Administered")
    and (not "Vasopressor Indication" or "Vasopressors Applied")

// =============================================================================
// FLUID RESUSCITATION TARGET
// =============================================================================

define "Patient Weight Kg":
  GetLatestValue(Code { code: '29463-7', system: 'http://loinc.org', display: 'Body weight' })

define "Fluid Resuscitation Target mL":
  if "Patient Weight Kg" is not null then
    "Patient Weight Kg" * 30  // 30 mL/kg crystalloid
  else
    null

define "Fluid Resuscitation Guidance":
  if "Fluid Resuscitation Indicated" then
    {
      targetVolume: "Fluid Resuscitation Target mL",
      targetUnit: 'mL',
      fluidType: 'Balanced crystalloid preferred (Lactated Ringers)',
      rate: 'Bolus over 30-60 minutes',
      monitoring: 'Reassess after each bolus - signs of fluid responsiveness',
      caution: 'Avoid fluid overload - dynamic assessment recommended'
    }
  else
    null

// =============================================================================
// VASOPRESSOR GUIDANCE
// =============================================================================

define "Vasopressor Guidance":
  if "Vasopressor Indication" then
    {
      firstLine: 'Norepinephrine',
      mapTarget: 65,  // mmHg
      escalation: 'Add vasopressin if norepinephrine > 0.25-0.5 mcg/kg/min',
      alternatives: 'Epinephrine if inadequate response',
      monitoring: 'Central line preferred for infusion',
      caution: 'Ensure adequate volume resuscitation before/during vasopressors'
    }
  else
    null

// =============================================================================
// ANTIMICROBIAL DELAY RISK
// =============================================================================

define "Time Since Sepsis Onset":
  // In production: calculate from first documented sepsis criteria
  null  // Requires encounter timestamp tracking

define "Antimicrobial Delay Risk":
  case
    when "Meets Septic Shock Criteria" and not "Antibiotics Administered" then 'CRITICAL'
    when "Meets Sepsis Criteria" and not "Antibiotics Administered" then 'HIGH'
    when "Meets qSOFA Screening" and not "Antibiotics Administered" then 'MODERATE'
    else 'LOW'
  end

define "Antimicrobial Delay Alert":
  if "Antimicrobial Delay Risk" in { 'CRITICAL', 'HIGH' } then
    {
      alert: 'ANTIMICROBIAL_DELAY',
      severity: "Antimicrobial Delay Risk",
      message: 'Delay in antibiotic administration increases mortality',
      guidance: 'Administer empiric broad-spectrum antibiotics immediately',
      evidence: 'Each hour delay associated with ~4% increase in mortality (Kumar 2006)'
    }
  else
    null

// =============================================================================
// LACTATE CLEARANCE MONITORING
// =============================================================================

define "Lactate Clearance Target":
  // Target: ≥10% reduction per 2 hours or normalization
  if "Latest Lactate" > 2.0 then
    {
      target: 'Lactate clearance ≥10% every 2 hours',
      repeatMeasurement: '2-4 hours',
      goal: 'Normalization (< 2 mmol/L)',
      implication: 'Inadequate clearance suggests ongoing hypoperfusion'
    }
  else
    null

// =============================================================================
// SOURCE CONTROL
// =============================================================================

define "Source Control Required":
  "Infection Source" in { 'Intra-abdominal', 'Skin/Soft Tissue' }
    and "Meets Sepsis Criteria"

define "Source Control Guidance":
  if "Source Control Required" then
    {
      urgency: 'Within 6-12 hours if feasible',
      interventions: case "Infection Source"
        when 'Intra-abdominal' then 'Surgical drainage, laparotomy, or interventional radiology'
        when 'Skin/Soft Tissue' then 'Surgical debridement of necrotizing infection'
        else 'Assess for drainable source'
      end,
      caution: 'Balance urgency against hemodynamic stability'
    }
  else
    null

// =============================================================================
// ICU DOMINANCE OUTPUT
// =============================================================================
// Per CTO/CMO Directive: Sepsis triggers ICU veto authority

define "ICU Dominance Activated":
  "Meets Sepsis Criteria" or "Meets Septic Shock Criteria"

define "ICU Dominance Reason":
  case
    when "Meets Septic Shock Criteria" then 'Septic Shock - Immediate ICU oversight required'
    when "Meets Sepsis Criteria" then 'Sepsis - ICU Intelligence has veto authority'
    else null
  end

// =============================================================================
// COMPOSITE OUTPUT
// =============================================================================

define "Sepsis Guidelines Output":
  {
    // Sepsis Detection
    hasSuspectedInfection: "Has Suspected Infection",
    infectionSource: "Infection Source",

    // Scoring
    qsofaScore: "qSOFA Score",
    qsofaPositive: "qSOFA Positive",
    sofaScore: "SOFA Total Score",
    sofaComponents: {
      respiration: "SOFA Respiration Score",
      coagulation: "SOFA Coagulation Score",
      liver: "SOFA Liver Score",
      cardiovascular: "SOFA Cardiovascular Score",
      cns: "SOFA CNS Score",
      renal: "SOFA Renal Score"
    },

    // Sepsis Status
    meetsSepsisCriteria: "Meets Sepsis Criteria",
    meetsSepticShockCriteria: "Meets Septic Shock Criteria",

    // Lactate
    latestLactate: "Latest Lactate",
    lactateElevated: "Lactate Elevated",
    lactateClearanceTarget: "Lactate Clearance Target",

    // Bundle Compliance
    hour1BundleApplicable: "Hour 1 Bundle Applicable",
    hour1BundleComponents: "Hour 1 Bundle Components",
    hour1BundleComplete: "Hour 1 Bundle Complete",

    // Resuscitation
    fluidResuscitationIndicated: "Fluid Resuscitation Indicated",
    fluidTarget: "Fluid Resuscitation Target mL",
    fluidGuidance: "Fluid Resuscitation Guidance",

    // Vasopressors
    vasopressorIndication: "Vasopressor Indication",
    onVasopressors: "On Vasopressors",
    vasopressorGuidance: "Vasopressor Guidance",

    // Antimicrobials
    antimicrobialDelayRisk: "Antimicrobial Delay Risk",
    antimicrobialDelayAlert: "Antimicrobial Delay Alert",

    // Source Control
    sourceControlRequired: "Source Control Required",
    sourceControlGuidance: "Source Control Guidance",

    // ICU Dominance (CTO/CMO Directive)
    icuDominanceActivated: "ICU Dominance Activated",
    icuDominanceReason: "ICU Dominance Reason",
    icuHasVetoAuthority: "ICU Dominance Activated"  // Always true when sepsis present
  }
