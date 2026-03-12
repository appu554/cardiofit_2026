// =============================================================================
// T2DMGuidelines.cql
// Tier 4b: Clinical Guidelines - Type 2 Diabetes per ADA Standards 2024
// =============================================================================
//
// PURPOSE:
// Implements Type 2 Diabetes mellitus management guidelines based on
// ADA (American Diabetes Association) Standards of Medical Care 2024.
//
// KEY COMPONENTS:
// 1. Glycemic Target Individualization
// 2. Metformin as First-Line Therapy
// 3. SGLT2i/GLP1-RA for ASCVD/CKD/HF
// 4. Insulin Initiation Criteria
// 5. Hypoglycemia Risk Assessment
//
// CARDIORENAL PROTECTION:
// - SGLT2i preferred with heart failure
// - GLP1-RA preferred with ASCVD
// - Either for CKD with albuminuria
//
// SOURCES:
// - ADA Standards of Medical Care in Diabetes 2024
// - ADA/EASD Consensus Report on T2DM Management 2022
// =============================================================================

library T2DMGuidelines version '1.0.0'

using FHIR version '4.0.1'

include "FHIRHelpers" version '4.0.1' called FHIRHelpers

// =============================================================================
// CODESYSTEMS
// =============================================================================

codesystem "LOINC": 'http://loinc.org'
codesystem "SNOMED": 'http://snomed.info/sct'
codesystem "RxNorm": 'http://www.nlm.nih.gov/research/umls/rxnorm'

// Lab codes
code "HbA1c": '4548-4' from "LOINC" display 'Hemoglobin A1c'
code "Fasting Glucose": '1558-6' from "LOINC" display 'Fasting glucose'
code "Random Glucose": '2339-0' from "LOINC" display 'Glucose random'
code "eGFR": '33914-3' from "LOINC" display 'eGFR'
code "UACR": '14959-1' from "LOINC" display 'Urine albumin/creatinine ratio'
code "Creatinine": '2160-0' from "LOINC" display 'Creatinine'
code "BMI": '39156-5' from "LOINC" display 'Body mass index'
code "Systolic BP": '8480-6' from "LOINC" display 'Systolic blood pressure'
code "Diastolic BP": '8462-4' from "LOINC" display 'Diastolic blood pressure'
code "LDL": '18262-6' from "LOINC" display 'LDL cholesterol'

// =============================================================================
// VALUESETS
// =============================================================================

// Diabetes conditions
valueset "Type 2 Diabetes Codes": 'http://vaidshala.health/ValueSet/t2dm'
valueset "Type 1 Diabetes Codes": 'http://vaidshala.health/ValueSet/t1dm'
valueset "Prediabetes Codes": 'http://vaidshala.health/ValueSet/prediabetes'

// Cardiovascular conditions
valueset "ASCVD Codes": 'http://vaidshala.health/ValueSet/ascvd'
valueset "Heart Failure Codes": 'http://vaidshala.health/ValueSet/heart-failure'
valueset "PAD Codes": 'http://vaidshala.health/ValueSet/pad'

// CKD
valueset "CKD Codes": 'http://vaidshala.health/ValueSet/ckd'

// Hypoglycemia
valueset "Hypoglycemia Events": 'http://vaidshala.health/ValueSet/hypoglycemia'
valueset "Severe Hypoglycemia Events": 'http://vaidshala.health/ValueSet/severe-hypoglycemia'

// =============================================================================
// VALUESETS - Diabetes Medications
// =============================================================================

valueset "Metformin Medications": 'http://vaidshala.health/ValueSet/metformin'
valueset "SGLT2i Medications": 'http://vaidshala.health/ValueSet/sglt2i'
valueset "GLP1-RA Medications": 'http://vaidshala.health/ValueSet/glp1-ra'
valueset "DPP4i Medications": 'http://vaidshala.health/ValueSet/dpp4i'
valueset "Sulfonylurea Medications": 'http://vaidshala.health/ValueSet/sulfonylurea'
valueset "TZD Medications": 'http://vaidshala.health/ValueSet/tzd'
valueset "Basal Insulin": 'http://vaidshala.health/ValueSet/basal-insulin'
valueset "Bolus Insulin": 'http://vaidshala.health/ValueSet/bolus-insulin'
valueset "Premixed Insulin": 'http://vaidshala.health/ValueSet/premixed-insulin'

// =============================================================================
// PARAMETERS
// =============================================================================

parameter "Measurement Period" Interval<DateTime>
  default Interval[@2024-01-01T00:00:00.0, @2024-12-31T23:59:59.999]

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

// =============================================================================
// PATIENT CHARACTERISTICS
// =============================================================================

define "Patient Age":
  AgeInYears()

define "Is Elderly":
  "Patient Age" >= 65

define "Is Very Elderly":
  "Patient Age" >= 80

// =============================================================================
// DIABETES DIAGNOSIS
// =============================================================================

define "Has Type 2 Diabetes":
  exists ([Condition] C where C.code in "Type 2 Diabetes Codes"
    and C.clinicalStatus.coding[0].code = 'active')

define "Has Type 1 Diabetes":
  exists ([Condition] C where C.code in "Type 1 Diabetes Codes"
    and C.clinicalStatus.coding[0].code = 'active')

define "Has Prediabetes":
  exists ([Condition] C where C.code in "Prediabetes Codes"
    and C.clinicalStatus.coding[0].code = 'active')

// =============================================================================
// GLYCEMIC STATUS
// =============================================================================

define "Latest HbA1c":
  GetLatestValue("HbA1c")

define "Latest Fasting Glucose":
  GetLatestValue("Fasting Glucose")

define "Latest BMI":
  GetLatestValue("BMI")

define "Glycemic Control Status":
  case
    when "Latest HbA1c" is null then 'Unknown'
    when "Latest HbA1c" < 7.0 then 'At Target'
    when "Latest HbA1c" >= 7.0 and "Latest HbA1c" < 8.0 then 'Above Target - Mild'
    when "Latest HbA1c" >= 8.0 and "Latest HbA1c" < 9.0 then 'Above Target - Moderate'
    when "Latest HbA1c" >= 9.0 and "Latest HbA1c" < 10.0 then 'Above Target - Poor'
    when "Latest HbA1c" >= 10.0 then 'Above Target - Very Poor'
    else 'Unknown'
  end

// =============================================================================
// INDIVIDUALIZED GLYCEMIC TARGET (ADA 2024)
// =============================================================================

define "Has Limited Life Expectancy":
  // Simplified - would require more complex assessment
  "Is Very Elderly"

define "Has Extensive Comorbidities":
  // Presence of multiple serious conditions
  "Has ASCVD"
    and "Has Heart Failure"
    and "Has CKD"

define "Has History of Severe Hypoglycemia":
  exists ([Condition] C where C.code in "Severe Hypoglycemia Events")

define "Glycemic Target":
  case
    // Less stringent target (≤8%)
    when "Has Limited Life Expectancy" or "Has Extensive Comorbidities"
         or "Has History of Severe Hypoglycemia" then 8.0
    // Standard target (≤7%)
    when "Is Elderly" then 7.5
    // More stringent for younger patients with short duration
    when "Patient Age" < 50 and not "Has ASCVD" then 6.5
    // Standard target
    else 7.0
  end

define "HbA1c At Target":
  "Latest HbA1c" is not null and "Latest HbA1c" <= "Glycemic Target"

define "HbA1c Gap":
  if "Latest HbA1c" is not null then
    "Latest HbA1c" - "Glycemic Target"
  else
    null

// =============================================================================
// CARDIOVASCULAR AND RENAL RISK
// =============================================================================

define "Has ASCVD":
  exists ([Condition] C where C.code in "ASCVD Codes"
    and C.clinicalStatus.coding[0].code = 'active')
    or exists ([Condition] C where C.code in "PAD Codes"
    and C.clinicalStatus.coding[0].code = 'active')

define "Has Heart Failure":
  exists ([Condition] C where C.code in "Heart Failure Codes"
    and C.clinicalStatus.coding[0].code = 'active')

define "Has CKD":
  exists ([Condition] C where C.code in "CKD Codes"
    and C.clinicalStatus.coding[0].code = 'active')

define "Latest eGFR":
  GetLatestValue("eGFR")

define "Latest UACR":
  GetLatestValue("UACR")

define "Has Albuminuria":
  "Latest UACR" is not null and "Latest UACR" >= 30

define "Has CKD with Albuminuria":
  "Has CKD" or ("Latest eGFR" < 60) or "Has Albuminuria"

define "Cardiorenal Risk Category":
  case
    when "Has ASCVD" and "Has Heart Failure" and "Has CKD with Albuminuria" then 'Very High'
    when "Has ASCVD" or "Has Heart Failure" or "Has CKD with Albuminuria" then 'High'
    when "Is Elderly" or "Latest eGFR" < 90 then 'Moderate'
    else 'Low'
  end

// =============================================================================
// CURRENT DIABETES THERAPY
// =============================================================================

define "On Metformin":
  exists ([MedicationRequest] MR where MR.medication in "Metformin Medications" and MR.status = 'active')

define "On SGLT2i":
  exists ([MedicationRequest] MR where MR.medication in "SGLT2i Medications" and MR.status = 'active')

define "On GLP1-RA":
  exists ([MedicationRequest] MR where MR.medication in "GLP1-RA Medications" and MR.status = 'active')

define "On DPP4i":
  exists ([MedicationRequest] MR where MR.medication in "DPP4i Medications" and MR.status = 'active')

define "On Sulfonylurea":
  exists ([MedicationRequest] MR where MR.medication in "Sulfonylurea Medications" and MR.status = 'active')

define "On TZD":
  exists ([MedicationRequest] MR where MR.medication in "TZD Medications" and MR.status = 'active')

define "On Basal Insulin":
  exists ([MedicationRequest] MR where MR.medication in "Basal Insulin" and MR.status = 'active')

define "On Bolus Insulin":
  exists ([MedicationRequest] MR where MR.medication in "Bolus Insulin" and MR.status = 'active')

define "On Any Insulin":
  "On Basal Insulin" or "On Bolus Insulin"
    or exists ([MedicationRequest] MR where MR.medication in "Premixed Insulin" and MR.status = 'active')

define "Diabetes Medication Count":
  (if "On Metformin" then 1 else 0)
  + (if "On SGLT2i" then 1 else 0)
  + (if "On GLP1-RA" then 1 else 0)
  + (if "On DPP4i" then 1 else 0)
  + (if "On Sulfonylurea" then 1 else 0)
  + (if "On TZD" then 1 else 0)
  + (if "On Any Insulin" then 1 else 0)

// =============================================================================
// METFORMIN FIRST-LINE ASSESSMENT
// =============================================================================

define "Metformin Contraindicated":
  "Latest eGFR" < 30

define "Metformin Dose Adjustment Needed":
  "Latest eGFR" >= 30 and "Latest eGFR" < 45

define "Metformin First Line Applicable":
  "Has Type 2 Diabetes"
    and not "Metformin Contraindicated"
    and not "On Metformin"

define "Metformin Recommendation":
  case
    when "Metformin Contraindicated" then
      {
        status: 'CONTRAINDICATED',
        reason: 'eGFR < 30 mL/min/1.73m²'
      }
    when "Metformin Dose Adjustment Needed" and "On Metformin" then
      {
        status: 'DOSE_ADJUSTMENT',
        reason: 'eGFR 30-44 - reduce dose to 1000mg/day max'
      }
    when "On Metformin" then
      {
        status: 'CONTINUE',
        reason: 'First-line therapy - continue unless contraindicated'
      }
    when "Metformin First Line Applicable" then
      {
        status: 'INITIATE',
        reason: 'Start metformin as first-line therapy',
        action: 'Metformin 500mg BID, titrate to 1000mg BID'
      }
    else null
  end

// =============================================================================
// SGLT2i/GLP1-RA FOR CARDIORENAL PROTECTION
// =============================================================================

define "SGLT2i Indicated for ASCVD":
  "Has ASCVD" and not "On SGLT2i"

define "SGLT2i Indicated for HF":
  "Has Heart Failure" and not "On SGLT2i"

define "SGLT2i Indicated for CKD":
  "Has CKD with Albuminuria" and not "On SGLT2i"

define "SGLT2i Contraindicated":
  "Latest eGFR" < 20

define "GLP1RA Indicated for ASCVD":
  "Has ASCVD" and not "On GLP1-RA"

define "Cardiorenal Protection Needed":
  ("Has ASCVD" or "Has Heart Failure" or "Has CKD with Albuminuria")
    and not "On SGLT2i"
    and not "On GLP1-RA"

define "Cardiorenal Medication Recommendation":
  case
    when "Has Heart Failure" and not "SGLT2i Contraindicated" and not "On SGLT2i" then
      {
        medication: 'SGLT2i',
        status: 'STRONGLY_INDICATED',
        reason: 'Heart failure - SGLT2i reduces hospitalization and mortality',
        action: 'Start dapagliflozin 10mg or empagliflozin 10mg daily'
      }
    when "Has ASCVD" and not "On GLP1-RA" then
      {
        medication: 'GLP1-RA',
        status: 'STRONGLY_INDICATED',
        reason: 'ASCVD - GLP1-RA reduces CV events',
        action: 'Start semaglutide, dulaglutide, or liraglutide'
      }
    when "Has CKD with Albuminuria" and not "SGLT2i Contraindicated" and not "On SGLT2i" then
      {
        medication: 'SGLT2i',
        status: 'STRONGLY_INDICATED',
        reason: 'CKD with albuminuria - SGLT2i slows progression',
        action: 'Start dapagliflozin or empagliflozin'
      }
    when "On SGLT2i" or "On GLP1-RA" then
      {
        medication: if "On SGLT2i" then 'SGLT2i' else 'GLP1-RA',
        status: 'ON_THERAPY',
        reason: 'Continue cardiorenal protective agent'
      }
    else null
  end

// =============================================================================
// INSULIN INITIATION CRITERIA
// =============================================================================

define "Insulin Initiation Criteria":
  // Consider insulin if:
  // 1. A1C ≥10% or
  // 2. Symptoms of hyperglycemia or
  // 3. Blood glucose ≥300 mg/dL
  ("Latest HbA1c" >= 10.0)
    or ("Latest Fasting Glucose" >= 250)

define "Intensification Needed":
  not "HbA1c At Target" and "Diabetes Medication Count" >= 2

define "Insulin Recommendation":
  case
    when "Insulin Initiation Criteria" and not "On Any Insulin" then
      {
        status: 'INITIATE_INSULIN',
        reason: 'Significant hyperglycemia (A1C ≥10% or glucose ≥250)',
        action: 'Start basal insulin 10 units or 0.1-0.2 units/kg',
        type: 'Basal insulin (glargine, detemir, or degludec)'
      }
    when "Intensification Needed" and not "On Basal Insulin" then
      {
        status: 'CONSIDER_BASAL',
        reason: 'Oral agents insufficient - consider adding basal insulin',
        action: 'Start basal insulin if A1C remains above target'
      }
    when "On Basal Insulin" and not "On Bolus Insulin" and not "HbA1c At Target" then
      {
        status: 'CONSIDER_INTENSIFICATION',
        reason: 'Basal insufficient - consider basal-bolus or GLP1-RA',
        action: 'Add prandial insulin or GLP1-RA to basal'
      }
    else null
  end

// =============================================================================
// HYPOGLYCEMIA RISK ASSESSMENT
// =============================================================================

define "Hypoglycemia Risk Level":
  case
    when "Has History of Severe Hypoglycemia" then 'HIGH'
    when "On Sulfonylurea" and "On Any Insulin" then 'HIGH'
    when "On Any Insulin" or "On Sulfonylurea" then 'MODERATE'
    when "Is Elderly" and ("On Sulfonylurea" or "On Any Insulin") then 'MODERATE'
    when "Latest eGFR" < 45 and ("On Sulfonylurea" or "On Any Insulin") then 'MODERATE'
    else 'LOW'
  end

define "Hypoglycemia Risk Mitigation":
  case
    when "Hypoglycemia Risk Level" = 'HIGH' then
      {
        recommendation: 'Consider de-intensification',
        actions: [
          'Reduce or discontinue sulfonylurea',
          'Consider switch to GLP1-RA or SGLT2i',
          'Liberalize glycemic target',
          'Ensure CGM or frequent monitoring'
        ]
      }
    when "Hypoglycemia Risk Level" = 'MODERATE' then
      {
        recommendation: 'Monitor closely',
        actions: [
          'Educate on hypoglycemia recognition and treatment',
          'Consider dose reduction if hypoglycemia occurs'
        ]
      }
    else null
  end

// =============================================================================
// COMPREHENSIVE CARE GAPS
// =============================================================================

define "Latest LDL":
  GetLatestValue("LDL")

define "Latest Systolic BP":
  GetLatestValue("Systolic BP")

define "Care Gaps":
  {
    glycemicControl: not "HbA1c At Target",
    metforminGap: "Metformin First Line Applicable",
    cardiorenalProtectionGap: "Cardiorenal Protection Needed",
    bpControl: "Latest Systolic BP" > 130,
    ldlControl: "Latest LDL" > 100 and "Has ASCVD",
    ckdScreening: "Latest UACR" is null,
    retinopathyScreening: true  // Would check procedure dates
  }

// =============================================================================
// COMPOSITE OUTPUT
// =============================================================================

define "T2DM Guidelines Output":
  {
    // Diagnosis
    hasType2Diabetes: "Has Type 2 Diabetes",
    hasPrediabetes: "Has Prediabetes",

    // Glycemic Status
    latestHbA1c: "Latest HbA1c",
    glycemicControlStatus: "Glycemic Control Status",
    individualizedTarget: "Glycemic Target",
    atTarget: "HbA1c At Target",
    hba1cGap: "HbA1c Gap",

    // Cardiorenal Risk
    hasASCVD: "Has ASCVD",
    hasHeartFailure: "Has Heart Failure",
    hasCKD: "Has CKD",
    hasAlbuminuria: "Has Albuminuria",
    cardiorenalRiskCategory: "Cardiorenal Risk Category",
    egfr: "Latest eGFR",
    uacr: "Latest UACR",

    // Current Therapy
    currentMedications: {
      metformin: "On Metformin",
      sglt2i: "On SGLT2i",
      glp1ra: "On GLP1-RA",
      dpp4i: "On DPP4i",
      sulfonylurea: "On Sulfonylurea",
      tzd: "On TZD",
      insulin: "On Any Insulin"
    },
    medicationCount: "Diabetes Medication Count",

    // Recommendations
    metforminRecommendation: "Metformin Recommendation",
    cardiorenalRecommendation: "Cardiorenal Medication Recommendation",
    insulinRecommendation: "Insulin Recommendation",

    // Hypoglycemia
    hypoglycemiaRisk: "Hypoglycemia Risk Level",
    hypoglycemiaMitigation: "Hypoglycemia Risk Mitigation",

    // Care Gaps
    careGaps: "Care Gaps"
  }
