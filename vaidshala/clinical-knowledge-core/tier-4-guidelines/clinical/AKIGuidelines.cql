// =============================================================================
// AKIGuidelines.cql
// Tier 4b: Clinical Guidelines - AKI per KDIGO 2012
// =============================================================================
//
// PURPOSE:
// Implements Acute Kidney Injury prevention, diagnosis, and management
// based on KDIGO (Kidney Disease: Improving Global Outcomes) 2012 guidelines.
//
// ICU DOMINANCE TRIGGER:
// Per CTO/CMO Directive: "AKI Stage 2-3 activates ICU oversight"
// Severe AKI requires ICU Intelligence coordination for nephrotoxin management.
//
// KEY COMPONENTS:
// 1. AKI Staging (KDIGO criteria)
// 2. Risk Factor Identification
// 3. Nephrotoxin Avoidance
// 4. Fluid Management
// 5. RRT Indication Assessment
//
// SOURCES:
// - KDIGO Clinical Practice Guideline for AKI 2012
// - KDIGO AKI Update 2021
// =============================================================================

library AKIGuidelines version '1.0.0'

using FHIR version '4.0.1'

include "FHIRHelpers" version '4.0.1' called FHIRHelpers

// =============================================================================
// CODESYSTEMS
// =============================================================================

codesystem "LOINC": 'http://loinc.org'
codesystem "SNOMED": 'http://snomed.info/sct'
codesystem "RxNorm": 'http://www.nlm.nih.gov/research/umls/rxnorm'

// Lab codes
code "Serum Creatinine": '2160-0' from "LOINC" display 'Creatinine'
code "eGFR": '33914-3' from "LOINC" display 'eGFR'
code "BUN": '3094-0' from "LOINC" display 'BUN'
code "Potassium": '2823-3' from "LOINC" display 'Potassium'
code "Bicarbonate": '1963-8' from "LOINC" display 'Bicarbonate'
code "Phosphorus": '2777-1' from "LOINC" display 'Phosphorus'
code "Urine Output": '9192-6' from "LOINC" display 'Urine output'
code "Lactate": '2524-7' from "LOINC" display 'Lactate'

// =============================================================================
// VALUESETS
// =============================================================================

// AKI conditions
valueset "AKI Diagnosis Codes": 'http://vaidshala.health/ValueSet/aki'
valueset "CKD Codes": 'http://vaidshala.health/ValueSet/ckd'

// Risk factor conditions
valueset "Diabetes Codes": 'http://vaidshala.health/ValueSet/diabetes'
valueset "Heart Failure Codes": 'http://vaidshala.health/ValueSet/heart-failure'
valueset "Liver Disease Codes": 'http://vaidshala.health/ValueSet/liver-disease'
valueset "Sepsis Codes": 'http://vaidshala.health/ValueSet/sepsis-diagnosis'

// Nephrotoxic medications
valueset "Aminoglycosides": 'http://vaidshala.health/ValueSet/aminoglycosides'
valueset "NSAIDs": 'http://vaidshala.health/ValueSet/nsaids'
valueset "IV Contrast Agents": 'http://vaidshala.health/ValueSet/iv-contrast'
valueset "ACEi ARB Medications": 'http://vaidshala.health/ValueSet/acei-arb'
valueset "Amphotericin B": 'http://vaidshala.health/ValueSet/amphotericin'
valueset "Vancomycin": 'http://vaidshala.health/ValueSet/vancomycin'
valueset "Calcineurin Inhibitors": 'http://vaidshala.health/ValueSet/cni'
valueset "Diuretics": 'http://vaidshala.health/ValueSet/diuretics'
valueset "PPI Medications": 'http://vaidshala.health/ValueSet/ppi'

// Renal protective
valueset "Renal Protective Fluids": 'http://vaidshala.health/ValueSet/renal-fluids'

// =============================================================================
// PARAMETERS
// =============================================================================

parameter "Measurement Period" Interval<DateTime>
  default Interval[@2024-01-01T00:00:00.0, @2024-12-31T23:59:59.999]

// AKI staging thresholds (KDIGO)
parameter "AKI Stage 1 Creatinine Increase" Decimal default 0.3  // mg/dL absolute increase
parameter "AKI Stage 1 Creatinine Ratio" Decimal default 1.5     // 1.5-1.9x baseline
parameter "AKI Stage 2 Creatinine Ratio" Decimal default 2.0     // 2.0-2.9x baseline
parameter "AKI Stage 3 Creatinine Ratio" Decimal default 3.0     // ≥3.0x baseline
parameter "AKI Stage 3 Creatinine Absolute" Decimal default 4.0  // ≥4.0 mg/dL

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

define function GetObservationValue(obs FHIR.Observation):
  (obs.value as Quantity).value

// Get baseline creatinine (lowest in past 7-90 days, or estimate)
define function GetBaselineCreatinine():
  Coalesce(
    Min(
      [Observation: "Serum Creatinine"] O
        where O.status in { 'final', 'amended', 'corrected' }
          and FHIRHelpers.ToDateTime(O.effective as dateTime) >= Now() - 90 days
          and FHIRHelpers.ToDateTime(O.effective as dateTime) <= Now() - 7 days
        return GetObservationValue(O)
    ),
    // If no prior creatinine, estimate from eGFR = 75 (MDRD back-calculation)
    1.0  // Simplified default
  )

// =============================================================================
// CREATININE VALUES
// =============================================================================

define "Latest Creatinine":
  GetLatestValue("Serum Creatinine")

define "Baseline Creatinine":
  GetBaselineCreatinine()

define "Creatinine Change Absolute":
  if "Latest Creatinine" is not null and "Baseline Creatinine" is not null then
    "Latest Creatinine" - "Baseline Creatinine"
  else
    null

define "Creatinine Ratio":
  if "Latest Creatinine" is not null and "Baseline Creatinine" is not null and "Baseline Creatinine" > 0 then
    "Latest Creatinine" / "Baseline Creatinine"
  else
    null

// =============================================================================
// URINE OUTPUT
// =============================================================================

define "Latest Urine Output":
  GetLatestValue("Urine Output")

// Urine output criteria (mL/kg/hour)
define "Oliguria Present":
  "Latest Urine Output" is not null and "Latest Urine Output" < 0.5

define "Severe Oliguria":
  "Latest Urine Output" is not null and "Latest Urine Output" < 0.3

define "Anuria":
  "Latest Urine Output" is not null and "Latest Urine Output" < 0.1

// =============================================================================
// AKI STAGING (KDIGO)
// =============================================================================

define "AKI Stage":
  case
    // Stage 3
    when "Creatinine Ratio" >= "AKI Stage 3 Creatinine Ratio" then 3
    when "Latest Creatinine" >= "AKI Stage 3 Creatinine Absolute" then 3
    when "Anuria" then 3  // Anuria ≥12 hours
    // Stage 2
    when "Creatinine Ratio" >= "AKI Stage 2 Creatinine Ratio" then 2
    when "Severe Oliguria" then 2  // UO <0.5 mL/kg/h for ≥12 hours
    // Stage 1
    when "Creatinine Change Absolute" >= "AKI Stage 1 Creatinine Increase" then 1
    when "Creatinine Ratio" >= "AKI Stage 1 Creatinine Ratio" then 1
    when "Oliguria Present" then 1  // UO <0.5 mL/kg/h for 6-12 hours
    // No AKI
    else 0
  end

define "Has AKI":
  "AKI Stage" >= 1

define "Has Severe AKI":
  "AKI Stage" >= 2

define "AKI Stage Description":
  case
    when "AKI Stage" = 3 then 'Stage 3 - Severe'
    when "AKI Stage" = 2 then 'Stage 2 - Moderate'
    when "AKI Stage" = 1 then 'Stage 1 - Mild'
    else 'No AKI'
  end

// =============================================================================
// AKI RISK FACTORS
// =============================================================================

define "Has CKD":
  exists ([Condition] C where C.code in "CKD Codes"
    and C.clinicalStatus.coding[0].code = 'active')

define "Latest eGFR":
  GetLatestValue("eGFR")

define "Has Low Baseline GFR":
  "Latest eGFR" < 60

define "Has Diabetes":
  exists ([Condition] C where C.code in "Diabetes Codes"
    and C.clinicalStatus.coding[0].code = 'active')

define "Has Heart Failure":
  exists ([Condition] C where C.code in "Heart Failure Codes"
    and C.clinicalStatus.coding[0].code = 'active')

define "Has Liver Disease":
  exists ([Condition] C where C.code in "Liver Disease Codes"
    and C.clinicalStatus.coding[0].code = 'active')

define "Has Sepsis":
  exists ([Condition] C where C.code in "Sepsis Codes"
    and C.clinicalStatus.coding[0].code = 'active')

define "Is Elderly":
  AgeInYears() >= 65

define "AKI Risk Factors":
  {
    ckd: "Has CKD",
    lowBaselineGFR: "Has Low Baseline GFR",
    diabetes: "Has Diabetes",
    heartFailure: "Has Heart Failure",
    liverDisease: "Has Liver Disease",
    sepsis: "Has Sepsis",
    elderly: "Is Elderly"
  }

define "AKI Risk Factor Count":
  (if "Has CKD" then 1 else 0)
  + (if "Has Low Baseline GFR" then 1 else 0)
  + (if "Has Diabetes" then 1 else 0)
  + (if "Has Heart Failure" then 1 else 0)
  + (if "Has Liver Disease" then 1 else 0)
  + (if "Has Sepsis" then 1 else 0)
  + (if "Is Elderly" then 1 else 0)

define "High AKI Risk":
  "AKI Risk Factor Count" >= 3 or "Has CKD" or "Has Sepsis"

// =============================================================================
// NEPHROTOXIN EXPOSURE
// =============================================================================

define "On Aminoglycosides":
  exists ([MedicationRequest] MR where MR.medication in "Aminoglycosides" and MR.status = 'active')

define "On NSAIDs":
  exists ([MedicationRequest] MR where MR.medication in "NSAIDs" and MR.status = 'active')

define "On ACEi ARB":
  exists ([MedicationRequest] MR where MR.medication in "ACEi ARB Medications" and MR.status = 'active')

define "On Vancomycin":
  exists ([MedicationRequest] MR where MR.medication in "Vancomycin" and MR.status = 'active')

define "On Amphotericin":
  exists ([MedicationRequest] MR where MR.medication in "Amphotericin B" and MR.status = 'active')

define "On Calcineurin Inhibitors":
  exists ([MedicationRequest] MR where MR.medication in "Calcineurin Inhibitors" and MR.status = 'active')

define "On Diuretics":
  exists ([MedicationRequest] MR where MR.medication in "Diuretics" and MR.status = 'active')

define "On PPI":
  exists ([MedicationRequest] MR where MR.medication in "PPI Medications" and MR.status = 'active')

define "Recent IV Contrast":
  exists (
    [Procedure] P
      where P.code in "IV Contrast Agents"
        and FHIRHelpers.ToDateTime(P.performed) >= Now() - 72 hours
  )

define "Nephrotoxin Avoidance List":
  {
    aminoglycosides: "On Aminoglycosides",
    nsaids: "On NSAIDs",
    aceiArb: "On ACEi ARB",
    vancomycin: "On Vancomycin",
    amphotericin: "On Amphotericin",
    cni: "On Calcineurin Inhibitors",
    recentContrast: "Recent IV Contrast"
  }

define "Nephrotoxin Count":
  (if "On Aminoglycosides" then 1 else 0)
  + (if "On NSAIDs" then 1 else 0)
  + (if "On ACEi ARB" then 1 else 0)
  + (if "On Vancomycin" then 1 else 0)
  + (if "On Amphotericin" then 1 else 0)
  + (if "On Calcineurin Inhibitors" then 1 else 0)

define "Triple Whammy Present":
  // ACEi/ARB + Diuretic + NSAID - high AKI risk combination
  "On ACEi ARB" and "On Diuretics" and "On NSAIDs"

define "Nephrotoxin Alert":
  case
    when "Triple Whammy Present" then
      {
        severity: 'CRITICAL',
        alert: 'Triple whammy combination detected',
        medications: 'ACEi/ARB + Diuretic + NSAID',
        action: 'Discontinue NSAID immediately, consider holding ACEi/ARB and diuretic'
      }
    when "Has AKI" and "Nephrotoxin Count" > 0 then
      {
        severity: 'HIGH',
        alert: 'Active AKI with nephrotoxin exposure',
        action: 'Review and discontinue nephrotoxins if possible'
      }
    when "High AKI Risk" and "Nephrotoxin Count" > 0 then
      {
        severity: 'MODERATE',
        alert: 'High AKI risk with nephrotoxin exposure',
        action: 'Monitor renal function closely, consider alternatives'
      }
    else null
  end

// =============================================================================
// FLUID MANAGEMENT
// =============================================================================

define "Latest Lactate":
  GetLatestValue("Lactate")

define "Volume Depletion Suspected":
  "Has AKI"
    and ("Latest Lactate" > 2.0 or "Oliguria Present")
    and not "Has Heart Failure"

define "Fluid Management Target":
  case
    when "Volume Depletion Suspected" then
      {
        strategy: 'VOLUME_RESUSCITATION',
        target: 'Euvolemia - assess fluid responsiveness',
        fluid: 'Balanced crystalloid (LR preferred over NS)',
        caution: 'Avoid fluid overload - use dynamic assessment'
      }
    when "Has Heart Failure" and "Has AKI" then
      {
        strategy: 'CARDIORENAL',
        target: 'Optimal filling pressure without congestion',
        fluid: 'Careful diuresis vs. volume management',
        caution: 'Consider nephrology/cardiology consultation'
      }
    when "Has AKI" then
      {
        strategy: 'MONITOR',
        target: 'Maintain euvolemia',
        fluid: 'Avoid nephrotoxins, ensure adequate perfusion',
        caution: 'Daily fluid balance assessment'
      }
    else null
  end

// =============================================================================
// RRT (RENAL REPLACEMENT THERAPY) INDICATIONS
// =============================================================================

define "Latest Potassium":
  GetLatestValue("Potassium")

define "Latest Bicarbonate":
  GetLatestValue("Bicarbonate")

define "Hyperkalemia":
  "Latest Potassium" >= 6.5

define "Severe Metabolic Acidosis":
  "Latest Bicarbonate" < 15

define "Uremic Symptoms":
  // Would check for pericarditis, encephalopathy - simplified
  false

define "Volume Overload Refractory":
  // Would check for pulmonary edema unresponsive to diuretics
  false

define "RRT Indication":
  "Hyperkalemia"
    or "Severe Metabolic Acidosis"
    or "Uremic Symptoms"
    or "Volume Overload Refractory"
    or ("AKI Stage" = 3 and "Anuria")

define "RRT Urgency":
  case
    when "Hyperkalemia" then 'URGENT'
    when "Severe Metabolic Acidosis" then 'URGENT'
    when "Uremic Symptoms" then 'URGENT'
    when "Volume Overload Refractory" then 'URGENT'
    when "AKI Stage" = 3 then 'CONSIDER'
    else 'NOT_INDICATED'
  end

define "RRT Recommendation":
  if "RRT Indication" then
    {
      indication: true,
      urgency: "RRT Urgency",
      reasons: {
        hyperkalemia: "Hyperkalemia",
        acidosis: "Severe Metabolic Acidosis",
        uremia: "Uremic Symptoms",
        volumeOverload: "Volume Overload Refractory"
      },
      action: 'Urgent nephrology consultation for RRT',
      modality: 'CRRT if hemodynamically unstable, otherwise IHD'
    }
  else
    null

// =============================================================================
// ICU DOMINANCE OUTPUT
// =============================================================================
// Per CTO/CMO Directive: Severe AKI triggers ICU oversight

define "ICU Dominance Activated":
  "Has Severe AKI" or "RRT Indication"

define "ICU Dominance Reason":
  case
    when "RRT Indication" then 'RRT indicated - ICU oversight required'
    when "AKI Stage" = 3 then 'Stage 3 AKI - Critical'
    when "AKI Stage" = 2 then 'Stage 2 AKI - Moderate severity requires monitoring'
    else null
  end

// =============================================================================
// PREVENTION RECOMMENDATIONS
// =============================================================================

define "AKI Prevention Recommendations":
  if "High AKI Risk" and not "Has AKI" then
    {
      applicable: true,
      recommendations: [
        'Avoid or minimize nephrotoxin exposure',
        'Optimize volume status before procedures',
        'Hold ACEi/ARB during acute illness if hypovolemic',
        'Monitor creatinine after nephrotoxin exposure',
        'Use lowest effective dose of IV contrast',
        'Pre-hydrate with isotonic crystalloid before contrast'
      ]
    }
  else
    null

// =============================================================================
// COMPOSITE OUTPUT
// =============================================================================

define "AKI Guidelines Output":
  {
    // AKI Status
    hasAKI: "Has AKI",
    akiStage: "AKI Stage",
    akiStageDescription: "AKI Stage Description",
    hasSevereAKI: "Has Severe AKI",

    // Creatinine
    latestCreatinine: "Latest Creatinine",
    baselineCreatinine: "Baseline Creatinine",
    creatinineChange: "Creatinine Change Absolute",
    creatinineRatio: "Creatinine Ratio",

    // Urine Output
    urineOutput: "Latest Urine Output",
    oliguria: "Oliguria Present",
    anuria: "Anuria",

    // Risk Factors
    akiRiskFactors: "AKI Risk Factors",
    riskFactorCount: "AKI Risk Factor Count",
    highAKIRisk: "High AKI Risk",

    // Nephrotoxins
    nephrotoxinExposure: "Nephrotoxin Avoidance List",
    nephrotoxinCount: "Nephrotoxin Count",
    tripleWhammy: "Triple Whammy Present",
    nephrotoxinAlert: "Nephrotoxin Alert",

    // Management
    fluidManagement: "Fluid Management Target",
    rrtIndication: "RRT Indication",
    rrtUrgency: "RRT Urgency",
    rrtRecommendation: "RRT Recommendation",

    // Prevention
    preventionRecommendations: "AKI Prevention Recommendations",

    // ICU Dominance (CTO/CMO Directive)
    icuDominanceActivated: "ICU Dominance Activated",
    icuDominanceReason: "ICU Dominance Reason",

    // Labs
    labs: {
      egfr: "Latest eGFR",
      potassium: "Latest Potassium",
      bicarbonate: "Latest Bicarbonate",
      lactate: "Latest Lactate"
    }
  }
