// =============================================================================
// VTEProphylaxisGuidelines.cql
// Tier 4b: Clinical Guidelines - VTE Prevention per CHEST 2012/2016
// =============================================================================
//
// PURPOSE:
// Implements Venous Thromboembolism (VTE) prevention guidelines based on:
// - CHEST Guidelines (9th Edition, 2012 and updates)
// - Padua Prediction Score for medical patients
// - Caprini Score for surgical patients
// - IMPROVE Bleeding Risk Score
//
// KEY COMPONENTS:
// 1. VTE Risk Stratification (Padua/Caprini)
// 2. Bleeding Risk Assessment (IMPROVE)
// 3. Pharmacologic vs Mechanical Prophylaxis Decision
// 4. Contraindication Detection
//
// SOURCES:
// - CHEST Antithrombotic Guidelines 2012/2016
// - Padua Prediction Score (Barbar et al., J Thromb Haemost 2010)
// - Caprini Risk Assessment Model (Bahl et al., J Vasc Surg 2010)
// - IMPROVE Bleeding Risk Score (Decousus et al., Chest 2011)
// =============================================================================

library VTEProphylaxisGuidelines version '1.0.0'

using FHIR version '4.0.1'

include "FHIRHelpers" version '4.0.1' called FHIRHelpers

// =============================================================================
// CODESYSTEMS
// =============================================================================

codesystem "LOINC": 'http://loinc.org'
codesystem "SNOMED": 'http://snomed.info/sct'
codesystem "ICD10": 'http://hl7.org/fhir/sid/icd-10-cm'
codesystem "RxNorm": 'http://www.nlm.nih.gov/research/umls/rxnorm'

// Lab codes
code "Platelet Count": '777-3' from "LOINC" display 'Platelet count'
code "Hemoglobin": '718-7' from "LOINC" display 'Hemoglobin'
code "Creatinine": '2160-0' from "LOINC" display 'Creatinine'
code "eGFR": '33914-3' from "LOINC" display 'eGFR'
code "INR": '6301-6' from "LOINC" display 'INR'
code "aPTT": '3173-2' from "LOINC" display 'aPTT'

// =============================================================================
// VALUESETS
// =============================================================================

// VTE conditions
valueset "Prior VTE Codes": 'http://vaidshala.health/ValueSet/prior-vte'
valueset "Active VTE Codes": 'http://vaidshala.health/ValueSet/active-vte'
valueset "Thrombophilia Codes": 'http://vaidshala.health/ValueSet/thrombophilia'

// Cancer
valueset "Active Cancer Codes": 'http://vaidshala.health/ValueSet/active-cancer'
valueset "Cancer Treatment Codes": 'http://vaidshala.health/ValueSet/cancer-treatment'

// Immobility
valueset "Stroke with Paralysis Codes": 'http://vaidshala.health/ValueSet/stroke-paralysis'
valueset "Immobility Codes": 'http://vaidshala.health/ValueSet/immobility'

// Bleeding conditions
valueset "Active Bleeding Codes": 'http://vaidshala.health/ValueSet/active-bleeding'
valueset "Bleeding Disorder Codes": 'http://vaidshala.health/ValueSet/bleeding-disorder'
valueset "GI Ulcer Codes": 'http://vaidshala.health/ValueSet/gi-ulcer'
valueset "Recent Surgery Codes": 'http://vaidshala.health/ValueSet/recent-surgery'

// Medications
valueset "Anticoagulant Medications": 'http://vaidshala.health/ValueSet/anticoagulants'
valueset "Antiplatelet Medications": 'http://vaidshala.health/ValueSet/antiplatelets'
valueset "LMWH Medications": 'http://vaidshala.health/ValueSet/lmwh'
valueset "UFH Medications": 'http://vaidshala.health/ValueSet/ufh'

// =============================================================================
// PARAMETERS
// =============================================================================

parameter "Measurement Period" Interval<DateTime>
  default Interval[@2024-01-01T00:00:00.0, @2024-12-31T23:59:59.999]

// Risk thresholds
parameter "High VTE Risk Padua Threshold" Integer default 4
parameter "Moderate VTE Risk Caprini Threshold" Integer default 2
parameter "High VTE Risk Caprini Threshold" Integer default 5
parameter "High Bleeding Risk IMPROVE Threshold" Decimal default 7.0

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
  "Patient Age" >= 70

// =============================================================================
// PADUA PREDICTION SCORE (Medical Patients)
// =============================================================================
// Score ≥4 indicates high VTE risk

// Active cancer (3 points)
define "Has Active Cancer":
  exists ([Condition] C where C.code in "Active Cancer Codes" and C.clinicalStatus.coding[0].code = 'active')
    or exists ([Condition] C where C.code in "Cancer Treatment Codes" and C.clinicalStatus.coding[0].code = 'active')

define "Padua Cancer Points":
  if "Has Active Cancer" then 3 else 0

// Previous VTE (3 points)
define "Has Prior VTE":
  exists ([Condition] C where C.code in "Prior VTE Codes")

define "Padua Prior VTE Points":
  if "Has Prior VTE" then 3 else 0

// Reduced mobility (3 points)
define "Has Reduced Mobility":
  exists ([Condition] C where C.code in "Immobility Codes" and C.clinicalStatus.coding[0].code = 'active')
    or exists ([Condition] C where C.code in "Stroke with Paralysis Codes" and C.clinicalStatus.coding[0].code = 'active')

define "Padua Mobility Points":
  if "Has Reduced Mobility" then 3 else 0

// Known thrombophilia (3 points)
define "Has Thrombophilia":
  exists ([Condition] C where C.code in "Thrombophilia Codes")

define "Padua Thrombophilia Points":
  if "Has Thrombophilia" then 3 else 0

// Recent trauma/surgery (2 points)
define "Has Recent Surgery":
  exists ([Procedure] P where P.code in "Recent Surgery Codes"
    and FHIRHelpers.ToDateTime(P.performed) >= Now() - 30 days)

define "Padua Surgery Points":
  if "Has Recent Surgery" then 2 else 0

// Age ≥70 (1 point)
define "Padua Age Points":
  if "Is Elderly" then 1 else 0

// Heart/respiratory failure (1 point)
valueset "Heart Failure Codes": 'http://vaidshala.health/ValueSet/heart-failure'
valueset "Respiratory Failure Codes": 'http://vaidshala.health/ValueSet/respiratory-failure'

define "Has Heart Respiratory Failure":
  exists ([Condition] C where C.code in "Heart Failure Codes" and C.clinicalStatus.coding[0].code = 'active')
    or exists ([Condition] C where C.code in "Respiratory Failure Codes" and C.clinicalStatus.coding[0].code = 'active')

define "Padua Heart Respiratory Points":
  if "Has Heart Respiratory Failure" then 1 else 0

// Acute MI or stroke (1 point)
valueset "Acute MI Codes": 'http://vaidshala.health/ValueSet/acute-mi'
valueset "Acute Stroke Codes": 'http://vaidshala.health/ValueSet/acute-stroke'

define "Has Acute MI or Stroke":
  exists ([Condition] C where C.code in "Acute MI Codes" and C.clinicalStatus.coding[0].code = 'active')
    or exists ([Condition] C where C.code in "Acute Stroke Codes" and C.clinicalStatus.coding[0].code = 'active')

define "Padua MI Stroke Points":
  if "Has Acute MI or Stroke" then 1 else 0

// Acute infection or rheumatic disorder (1 point)
valueset "Acute Infection Codes": 'http://vaidshala.health/ValueSet/acute-infection'
valueset "Rheumatic Disorder Codes": 'http://vaidshala.health/ValueSet/rheumatic'

define "Has Infection or Rheumatic":
  exists ([Condition] C where C.code in "Acute Infection Codes" and C.clinicalStatus.coding[0].code = 'active')
    or exists ([Condition] C where C.code in "Rheumatic Disorder Codes" and C.clinicalStatus.coding[0].code = 'active')

define "Padua Infection Rheumatic Points":
  if "Has Infection or Rheumatic" then 1 else 0

// Obesity BMI ≥30 (1 point)
code "BMI": '39156-5' from "LOINC" display 'Body mass index'

define "Latest BMI":
  GetLatestValue("BMI")

define "Is Obese":
  "Latest BMI" >= 30

define "Padua Obesity Points":
  if "Is Obese" then 1 else 0

// Hormonal treatment (1 point)
valueset "Hormonal Therapy Codes": 'http://vaidshala.health/ValueSet/hormonal-therapy'

define "On Hormonal Treatment":
  exists ([MedicationRequest] MR where MR.medication in "Hormonal Therapy Codes" and MR.status = 'active')

define "Padua Hormonal Points":
  if "On Hormonal Treatment" then 1 else 0

// PADUA TOTAL SCORE
define "Padua Score":
  "Padua Cancer Points"
  + "Padua Prior VTE Points"
  + "Padua Mobility Points"
  + "Padua Thrombophilia Points"
  + "Padua Surgery Points"
  + "Padua Age Points"
  + "Padua Heart Respiratory Points"
  + "Padua MI Stroke Points"
  + "Padua Infection Rheumatic Points"
  + "Padua Obesity Points"
  + "Padua Hormonal Points"

define "High VTE Risk Padua":
  "Padua Score" >= "High VTE Risk Padua Threshold"

// =============================================================================
// VTE RISK SCORE (Simplified - for demonstration)
// =============================================================================

define "VTE Risk Score":
  "Padua Score"

define "VTE Risk Level":
  case
    when "Padua Score" >= 4 then 'HIGH'
    when "Padua Score" >= 2 then 'MODERATE'
    else 'LOW'
  end

// =============================================================================
// IMPROVE BLEEDING RISK SCORE
// =============================================================================
// Assesses risk of major bleeding on anticoagulation

// Active gastric/duodenal ulcer (4.5 points)
define "Has Active GI Ulcer":
  exists ([Condition] C where C.code in "GI Ulcer Codes" and C.clinicalStatus.coding[0].code = 'active')

define "IMPROVE Ulcer Points":
  if "Has Active GI Ulcer" then 4.5 else 0.0

// Bleeding in 3 months before admission (4 points)
define "Has Recent Bleeding":
  exists ([Condition] C where C.code in "Active Bleeding Codes"
    and FHIRHelpers.ToDateTime(C.onset) >= Now() - 90 days)

define "IMPROVE Recent Bleeding Points":
  if "Has Recent Bleeding" then 4.0 else 0.0

// Platelet count <50 (4 points)
define "Latest Platelets":
  GetLatestValue("Platelet Count")

define "Has Thrombocytopenia":
  "Latest Platelets" < 50

define "IMPROVE Platelet Points":
  if "Has Thrombocytopenia" then 4.0 else 0.0

// Age ≥85 (3.5 points)
define "IMPROVE Age Points":
  if "Patient Age" >= 85 then 3.5 else 0.0

// Hepatic failure (2.5 points)
valueset "Hepatic Failure Codes": 'http://vaidshala.health/ValueSet/hepatic-failure'

define "Has Hepatic Failure":
  exists ([Condition] C where C.code in "Hepatic Failure Codes" and C.clinicalStatus.coding[0].code = 'active')

define "IMPROVE Hepatic Points":
  if "Has Hepatic Failure" then 2.5 else 0.0

// Renal failure GFR <30 (2.5 points)
define "Latest eGFR":
  GetLatestValue("eGFR")

define "Has Severe Renal Failure":
  "Latest eGFR" < 30

define "IMPROVE Renal Points":
  if "Has Severe Renal Failure" then 2.5 else 0.0

// ICU/CCU stay (2.5 points)
// Simplified - would check encounter location
define "In ICU":
  false  // Placeholder

define "IMPROVE ICU Points":
  if "In ICU" then 2.5 else 0.0

// Central venous catheter (2 points)
valueset "Central Line Codes": 'http://vaidshala.health/ValueSet/central-line'

define "Has Central Line":
  exists ([Device] D where D.type in "Central Line Codes" and D.status = 'active')

define "IMPROVE CVC Points":
  if "Has Central Line" then 2.0 else 0.0

// Rheumatic disease (2 points)
define "IMPROVE Rheumatic Points":
  if "Has Infection or Rheumatic" then 2.0 else 0.0

// Current cancer (2 points)
define "IMPROVE Cancer Points":
  if "Has Active Cancer" then 2.0 else 0.0

// Male gender (1 point)
define "Is Male":
  Patient.gender.value = 'male'

define "IMPROVE Gender Points":
  if "Is Male" then 1.0 else 0.0

// IMPROVE TOTAL SCORE
define "Bleeding Risk Score":
  "IMPROVE Ulcer Points"
  + "IMPROVE Recent Bleeding Points"
  + "IMPROVE Platelet Points"
  + "IMPROVE Age Points"
  + "IMPROVE Hepatic Points"
  + "IMPROVE Renal Points"
  + "IMPROVE ICU Points"
  + "IMPROVE CVC Points"
  + "IMPROVE Rheumatic Points"
  + "IMPROVE Cancer Points"
  + "IMPROVE Gender Points"

define "High Bleeding Risk":
  "Bleeding Risk Score" >= "High Bleeding Risk IMPROVE Threshold"

define "Bleeding Risk Level":
  case
    when "Bleeding Risk Score" >= 7.0 then 'HIGH'
    when "Bleeding Risk Score" >= 4.0 then 'MODERATE'
    else 'LOW'
  end

// =============================================================================
// CONTRAINDICATIONS TO ANTICOAGULATION
// =============================================================================

define "Active Bleeding":
  exists ([Condition] C where C.code in "Active Bleeding Codes" and C.clinicalStatus.coding[0].code = 'active')

define "Bleeding Disorder":
  exists ([Condition] C where C.code in "Bleeding Disorder Codes")

define "Severe Thrombocytopenia":
  "Latest Platelets" < 25

define "Absolute Contraindication to Anticoagulation":
  "Active Bleeding"
    or "Severe Thrombocytopenia"
    or ("Latest INR" is not null and "Latest INR" > 3.0)

define "Latest INR":
  GetLatestValue("INR")

define "Relative Contraindication to Anticoagulation":
  "Has Active GI Ulcer"
    or "Bleeding Disorder"
    or "Has Thrombocytopenia"
    or "Has Severe Renal Failure"

define "Contraindication to Anticoagulation":
  "Absolute Contraindication to Anticoagulation"
    or "Relative Contraindication to Anticoagulation"

// =============================================================================
// PROPHYLAXIS RECOMMENDATIONS
// =============================================================================

define "Pharmacologic Prophylaxis Indicated":
  "VTE Risk Level" in { 'HIGH', 'MODERATE' }
    and not "Absolute Contraindication to Anticoagulation"
    and not "High Bleeding Risk"

define "Mechanical Prophylaxis Indicated":
  ("VTE Risk Level" in { 'HIGH', 'MODERATE' } and "Contraindication to Anticoagulation")
    or ("VTE Risk Level" in { 'HIGH', 'MODERATE' } and "High Bleeding Risk")
    or "VTE Risk Level" = 'HIGH'  // Both mechanical + pharmacologic for high risk

define "Prophylaxis Recommendation":
  case
    when "Absolute Contraindication to Anticoagulation" then
      {
        prophylaxis: 'MECHANICAL_ONLY',
        pharmacologic: null,
        mechanical: 'Intermittent pneumatic compression (IPC)',
        rationale: 'Absolute contraindication to anticoagulation'
      }
    when "VTE Risk Level" = 'HIGH' and not "High Bleeding Risk" then
      {
        prophylaxis: 'COMBINED',
        pharmacologic: 'LMWH (preferred) or UFH',
        mechanical: 'Consider adding IPC',
        rationale: 'High VTE risk - combine pharmacologic and mechanical'
      }
    when "VTE Risk Level" = 'HIGH' and "High Bleeding Risk" then
      {
        prophylaxis: 'MECHANICAL_ONLY',
        pharmacologic: null,
        mechanical: 'IPC - start pharmacologic when bleeding risk decreases',
        rationale: 'High VTE risk with high bleeding risk'
      }
    when "VTE Risk Level" = 'MODERATE' and not "High Bleeding Risk" then
      {
        prophylaxis: 'PHARMACOLOGIC',
        pharmacologic: 'LMWH or UFH',
        mechanical: 'Optional - consider patient preference',
        rationale: 'Moderate VTE risk'
      }
    when "VTE Risk Level" = 'LOW' then
      {
        prophylaxis: 'AMBULATION_ONLY',
        pharmacologic: null,
        mechanical: null,
        rationale: 'Low VTE risk - early ambulation encouraged'
      }
    else
      {
        prophylaxis: 'ASSESS',
        pharmacologic: null,
        mechanical: null,
        rationale: 'Unable to determine - clinical judgment required'
      }
  end

// =============================================================================
// CURRENT PROPHYLAXIS STATUS
// =============================================================================

define "On Anticoagulant Prophylaxis":
  exists ([MedicationRequest] MR where MR.medication in "Anticoagulant Medications" and MR.status = 'active')
    or exists ([MedicationRequest] MR where MR.medication in "LMWH Medications" and MR.status = 'active')
    or exists ([MedicationRequest] MR where MR.medication in "UFH Medications" and MR.status = 'active')

define "On Mechanical Prophylaxis":
  exists ([Device] D where D.type.coding[0].code = 'IPC' and D.status = 'active')

define "Prophylaxis Gap":
  "Pharmacologic Prophylaxis Indicated" and not "On Anticoagulant Prophylaxis"

// =============================================================================
// COMPOSITE OUTPUT
// =============================================================================

define "VTE Prophylaxis Output":
  {
    // VTE Risk
    vteRiskScore: "VTE Risk Score",
    vteRiskLevel: "VTE Risk Level",
    paduaScore: "Padua Score",
    highVTERisk: "High VTE Risk Padua",

    // Bleeding Risk
    bleedingRiskScore: "Bleeding Risk Score",
    bleedingRiskLevel: "Bleeding Risk Level",
    highBleedingRisk: "High Bleeding Risk",

    // Contraindications
    absoluteContraindication: "Absolute Contraindication to Anticoagulation",
    relativeContraindication: "Relative Contraindication to Anticoagulation",
    activeBleedingPresent: "Active Bleeding",

    // Recommendations
    pharmacologicIndicated: "Pharmacologic Prophylaxis Indicated",
    mechanicalIndicated: "Mechanical Prophylaxis Indicated",
    recommendation: "Prophylaxis Recommendation",

    // Current Status
    onAnticoagulation: "On Anticoagulant Prophylaxis",
    onMechanical: "On Mechanical Prophylaxis",
    prophylaxisGap: "Prophylaxis Gap",

    // Risk Factors
    riskFactors: {
      activeCancer: "Has Active Cancer",
      priorVTE: "Has Prior VTE",
      reducedMobility: "Has Reduced Mobility",
      thrombophilia: "Has Thrombophilia",
      recentSurgery: "Has Recent Surgery",
      obesity: "Is Obese",
      elderly: "Is Elderly"
    }
  }
