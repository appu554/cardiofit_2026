// =============================================================================
// AustraliaThresholdAdapter.cql
// Tier 5: Regional Adapters - Australia Clinical Thresholds
// =============================================================================
//
// PURPOSE:
// Implements Australia-specific clinical thresholds based on:
// - RACGP (Royal Australian College of General Practitioners) guidelines
// - NHMRC (National Health and Medical Research Council) recommendations
// - Kidney Health Australia guidelines
// - Heart Foundation of Australia guidelines
//
// KEY AUSTRALIAN CONSIDERATIONS:
// - Aboriginal and Torres Strait Islander specific thresholds
// - Medicare Benefits Schedule (MBS) item number requirements
// - PBS authority requirements for certain medications
//
// GOVERNANCE:
// - CQL CLASSIFIES patient status using Australian thresholds
// - Does NOT override physician clinical judgment
// - All thresholds sourced from published Australian guidelines
//
// SOURCES:
// - RACGP Red Book (Guidelines for preventive activities in general practice)
// - Diabetes Australia / RACGP Type 2 Diabetes Guidelines
// - Heart Foundation Cardiovascular Guidelines
// - Kidney Health Australia CKD Management Handbook
// =============================================================================

library AustraliaThresholdAdapter version '1.0.0'

using FHIR version '4.0.1'

include "FHIRHelpers" version '4.0.1' called FHIRHelpers

// =============================================================================
// CODESYSTEMS
// =============================================================================

codesystem "LOINC": 'http://loinc.org'
codesystem "SNOMED-AU": 'http://snomed.info/sct/32506021000036107'  // Australian extension

// Observation codes
code "BMI": '39156-5' from "LOINC" display 'Body mass index'
code "Waist Circumference": '8280-0' from "LOINC" display 'Waist circumference'
code "Fasting Glucose": '1558-6' from "LOINC" display 'Fasting glucose'
code "HbA1c": '4548-4' from "LOINC" display 'Hemoglobin A1c'
code "Systolic BP": '8480-6' from "LOINC" display 'Systolic blood pressure'
code "Diastolic BP": '8462-4' from "LOINC" display 'Diastolic blood pressure'
code "Total Cholesterol": '2093-3' from "LOINC" display 'Total cholesterol'
code "LDL Cholesterol": '18262-6' from "LOINC" display 'LDL cholesterol'
code "HDL Cholesterol": '2085-9' from "LOINC" display 'HDL cholesterol'
code "Triglycerides": '2571-8' from "LOINC" display 'Triglycerides'
code "eGFR": '33914-3' from "LOINC" display 'eGFR'
code "UACR": '14959-1' from "LOINC" display 'Urine albumin/creatinine ratio'

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
// PATIENT DEMOGRAPHICS
// =============================================================================

define "Patient Age":
  AgeInYears()

define "Patient Gender":
  Patient.gender.value

define "Is Male":
  "Patient Gender" = 'male'

define "Is Female":
  "Patient Gender" = 'female'

// Aboriginal and Torres Strait Islander status
// In production, this would be determined from Patient.extension or separate resource
define "Is Indigenous Australian":
  exists (
    Patient.extension E
      where E.url = 'http://hl7.org.au/fhir/StructureDefinition/indigenous-status'
        and (E.value as CodeableConcept).coding[0].code in { '1', '2', '3' }
        // 1 = Aboriginal, 2 = Torres Strait Islander, 3 = Both
  )

// =============================================================================
// BMI THRESHOLDS (RACGP GUIDELINES)
// =============================================================================
// Source: RACGP Red Book 9th Edition

define "BMI Obesity Threshold Australia":
  30.0  // Standard WHO threshold

define "BMI Overweight Threshold Australia":
  25.0

define "BMI Normal Lower Australia":
  18.5

define "BMI Normal Upper Australia":
  24.9

// Lower thresholds for Aboriginal and Torres Strait Islander peoples
// (similar to Asian populations due to metabolic risk profile)
define "BMI Obesity Threshold Indigenous":
  25.0

define "Latest BMI Value":
  GetLatestValue("BMI")

define "BMI Category Australia":
  case
    when "Latest BMI Value" is null then 'Unknown'
    when "Is Indigenous Australian" and "Latest BMI Value" >= "BMI Obesity Threshold Indigenous" then 'Obese (Indigenous criteria)'
    when "Latest BMI Value" < 18.5 then 'Underweight'
    when "Latest BMI Value" >= 18.5 and "Latest BMI Value" < 25.0 then 'Normal'
    when "Latest BMI Value" >= 25.0 and "Latest BMI Value" < 30.0 then 'Overweight'
    when "Latest BMI Value" >= 30.0 and "Latest BMI Value" < 35.0 then 'Obese Class I'
    when "Latest BMI Value" >= 35.0 and "Latest BMI Value" < 40.0 then 'Obese Class II'
    when "Latest BMI Value" >= 40.0 then 'Obese Class III'
    else 'Unknown'
  end

define "Is Obese Australia Criteria":
  if "Is Indigenous Australian" then
    "Latest BMI Value" >= "BMI Obesity Threshold Indigenous"
  else
    "Latest BMI Value" >= "BMI Obesity Threshold Australia"

// =============================================================================
// WAIST CIRCUMFERENCE (HEART FOUNDATION)
// =============================================================================
// Source: Heart Foundation of Australia

define "Waist Circumference Threshold Male Australia":
  94.0  // cm - increased risk

define "Waist Circumference High Risk Male Australia":
  102.0  // cm - high risk

define "Waist Circumference Threshold Female Australia":
  80.0  // cm - increased risk

define "Waist Circumference High Risk Female Australia":
  88.0  // cm - high risk

define "Latest Waist Circumference":
  GetLatestValue("Waist Circumference")

define "Waist Circumference Risk Australia":
  case
    when "Latest Waist Circumference" is null then 'Unknown'
    when "Is Male" and "Latest Waist Circumference" >= "Waist Circumference High Risk Male Australia" then 'High Risk'
    when "Is Male" and "Latest Waist Circumference" >= "Waist Circumference Threshold Male Australia" then 'Increased Risk'
    when "Is Female" and "Latest Waist Circumference" >= "Waist Circumference High Risk Female Australia" then 'High Risk'
    when "Is Female" and "Latest Waist Circumference" >= "Waist Circumference Threshold Female Australia" then 'Increased Risk'
    else 'Normal'
  end

// =============================================================================
// DIABETES THRESHOLDS (DIABETES AUSTRALIA/RACGP)
// =============================================================================
// Source: RACGP/Diabetes Australia Type 2 Diabetes Guidelines

define "Diabetes Fasting Glucose Threshold Australia":
  7.0  // mmol/L (= 126 mg/dL)

define "Prediabetes Fasting Glucose Lower Australia":
  6.1  // mmol/L (IFG starts)

define "Prediabetes Fasting Glucose Upper Australia":
  6.9  // mmol/L (IFG ends)

define "Diabetes HbA1c Threshold Australia":
  6.5  // % (48 mmol/mol)

define "Prediabetes HbA1c Lower Australia":
  6.0  // %

define "Prediabetes HbA1c Upper Australia":
  6.4  // %

define "Latest Fasting Glucose":
  GetLatestValue("Fasting Glucose")

define "Latest HbA1c":
  GetLatestValue("HbA1c")

// Convert mg/dL to mmol/L for glucose if needed
define "Fasting Glucose mmol":
  if "Latest Fasting Glucose" > 30 then  // Likely mg/dL
    "Latest Fasting Glucose" / 18.0
  else
    "Latest Fasting Glucose"

define "Glycemic Status Australia":
  case
    when "Latest HbA1c" >= "Diabetes HbA1c Threshold Australia" then 'Diabetes'
    when "Fasting Glucose mmol" >= "Diabetes Fasting Glucose Threshold Australia" then 'Diabetes'
    when "Latest HbA1c" >= "Prediabetes HbA1c Lower Australia" and "Latest HbA1c" < "Diabetes HbA1c Threshold Australia" then 'Prediabetes'
    when "Fasting Glucose mmol" >= "Prediabetes Fasting Glucose Lower Australia" and "Fasting Glucose mmol" < "Diabetes Fasting Glucose Threshold Australia" then 'Prediabetes'
    when "Fasting Glucose mmol" < "Prediabetes Fasting Glucose Lower Australia" then 'Normal'
    else 'Unknown'
  end

define "Has Diabetes Australia Criteria":
  "Glycemic Status Australia" = 'Diabetes'

// =============================================================================
// HBA1C TARGETS (RACGP INDIVIDUALIZED)
// =============================================================================
// Source: RACGP Type 2 Diabetes Guidelines - individualized targets

define "HbA1c Target General Australia":
  7.0  // % for most adults

define "HbA1c Target Tight Australia":
  6.5  // % for those at low hypoglycemia risk

define "HbA1c Target Relaxed Australia":
  8.0  // % for frail elderly, limited life expectancy

define "Recommended HbA1c Target":
  case
    when "Patient Age" >= 80 then "HbA1c Target Relaxed Australia"
    when "Patient Age" >= 70 then "HbA1c Target General Australia"
    else "HbA1c Target General Australia"
  end

define "HbA1c At Target":
  "Latest HbA1c" is not null and "Latest HbA1c" <= "Recommended HbA1c Target"

// =============================================================================
// BLOOD PRESSURE THRESHOLDS (HEART FOUNDATION)
// =============================================================================
// Source: Heart Foundation / Hypertension Guidelines 2016

define "Hypertension Threshold Systolic Australia":
  140.0  // mmHg (office measurement)

define "Hypertension Threshold Diastolic Australia":
  90.0  // mmHg

// Lower threshold for diabetes/CKD
define "Hypertension Threshold Systolic DM Australia":
  130.0  // mmHg

define "Hypertension Threshold Diastolic DM Australia":
  80.0  // mmHg

define "Latest Systolic BP":
  GetLatestValue("Systolic BP")

define "Latest Diastolic BP":
  GetLatestValue("Diastolic BP")

define "Blood Pressure Category Australia":
  case
    when "Latest Systolic BP" is null or "Latest Diastolic BP" is null then 'Unknown'
    when "Latest Systolic BP" >= 180 or "Latest Diastolic BP" >= 110 then 'Severe Hypertension'
    when "Latest Systolic BP" >= 160 or "Latest Diastolic BP" >= 100 then 'Moderate Hypertension'
    when "Latest Systolic BP" >= 140 or "Latest Diastolic BP" >= 90 then 'Mild Hypertension'
    when "Latest Systolic BP" >= 120 or "Latest Diastolic BP" >= 80 then 'High Normal'
    else 'Normal'
  end

define "Has Hypertension Australia Criteria":
  "Latest Systolic BP" >= "Hypertension Threshold Systolic Australia"
    or "Latest Diastolic BP" >= "Hypertension Threshold Diastolic Australia"

// =============================================================================
// LIPID TARGETS (HEART FOUNDATION)
// =============================================================================
// Source: Heart Foundation Lipid Guidelines

define "LDL Target General Australia":
  2.0  // mmol/L for moderate risk

define "LDL Target High Risk Australia":
  1.8  // mmol/L for high risk

define "LDL Target Very High Risk Australia":
  1.4  // mmol/L for very high risk (ACS, recurrent CVD)

define "Total Cholesterol Target Australia":
  4.0  // mmol/L

define "HDL Target Male Australia":
  1.0  // mmol/L

define "HDL Target Female Australia":
  1.3  // mmol/L

define "Triglycerides Target Australia":
  2.0  // mmol/L

define "Latest Total Cholesterol":
  GetLatestValue("Total Cholesterol")

define "Latest LDL":
  GetLatestValue("LDL Cholesterol")

define "Latest HDL":
  GetLatestValue("HDL Cholesterol")

define "Latest Triglycerides":
  GetLatestValue("Triglycerides")

// Convert mg/dL to mmol/L if needed
define "LDL mmol":
  if "Latest LDL" > 10 then "Latest LDL" / 38.67 else "Latest LDL"

define "Lipid Profile Assessment Australia":
  {
    ldlAtTarget: "LDL mmol" <= "LDL Target General Australia",
    totalCholAtTarget: "Latest Total Cholesterol" <= "Total Cholesterol Target Australia",
    hdlAdequate: ("Is Male" and "Latest HDL" >= "HDL Target Male Australia")
                  or ("Is Female" and "Latest HDL" >= "HDL Target Female Australia"),
    triglyceridesNormal: "Latest Triglycerides" <= "Triglycerides Target Australia"
  }

// =============================================================================
// CKD STAGING (KIDNEY HEALTH AUSTRALIA)
// =============================================================================
// Source: Kidney Health Australia CKD Management Handbook

define "eGFR CKD Stage 1":
  90.0  // mL/min/1.73m²

define "eGFR CKD Stage 2":
  60.0

define "eGFR CKD Stage 3a":
  45.0

define "eGFR CKD Stage 3b":
  30.0

define "eGFR CKD Stage 4":
  15.0

define "Latest eGFR":
  GetLatestValue("eGFR")

define "Latest UACR":
  GetLatestValue("UACR")

define "CKD Stage Australia":
  case
    when "Latest eGFR" is null then 'Unknown'
    when "Latest eGFR" >= "eGFR CKD Stage 1" then 'G1'
    when "Latest eGFR" >= "eGFR CKD Stage 2" then 'G2'
    when "Latest eGFR" >= "eGFR CKD Stage 3a" then 'G3a'
    when "Latest eGFR" >= "eGFR CKD Stage 3b" then 'G3b'
    when "Latest eGFR" >= "eGFR CKD Stage 4" then 'G4'
    else 'G5'
  end

define "Albuminuria Category Australia":
  case
    when "Latest UACR" is null then 'Unknown'
    when "Latest UACR" < 3 then 'A1 (Normal)'
    when "Latest UACR" >= 3 and "Latest UACR" < 30 then 'A2 (Microalbuminuria)'
    when "Latest UACR" >= 30 then 'A3 (Macroalbuminuria)'
    else 'Unknown'
  end

// =============================================================================
// ABORIGINAL AND TORRES STRAIT ISLANDER SCREENING
// =============================================================================
// Source: RACGP National Guide to a Preventive Health Assessment for Aboriginal and Torres Strait Islander People

define "Indigenous Health Check Due":
  "Is Indigenous Australian"
    and (
      "Patient Age" >= 15  // Annual health assessment MBS 715
    )

define "Indigenous CVD Screening Age":
  18  // Earlier screening for Indigenous Australians (vs 45 general population)

define "Indigenous Diabetes Screening Age":
  18  // Earlier screening

define "CVD Screening Due Indigenous":
  "Is Indigenous Australian"
    and "Patient Age" >= "Indigenous CVD Screening Age"

define "Diabetes Screening Due Indigenous":
  "Is Indigenous Australian"
    and "Patient Age" >= "Indigenous Diabetes Screening Age"

// =============================================================================
// AGE THRESHOLDS
// =============================================================================

define "Pediatric Age Threshold Australia":
  18  // years

define "Elderly Age Threshold Australia":
  65  // years

define "Frail Elderly Age Threshold Australia":
  80  // years

define "Is Pediatric Australia":
  "Patient Age" < "Pediatric Age Threshold Australia"

define "Is Elderly Australia":
  "Patient Age" >= "Elderly Age Threshold Australia"

define "Is Frail Elderly Australia":
  "Patient Age" >= "Frail Elderly Age Threshold Australia"

// =============================================================================
// ABSOLUTE CVD RISK (FRAMINGHAM ADAPTATION)
// =============================================================================
// Source: Australian absolute CVD risk guidelines
// Note: Full calculation requires dedicated calculator (KB-8)

define "CVD Risk Assessment Age Australia":
  45  // Start at 45 for non-Indigenous, 35 for Indigenous

define "CVD Risk Assessment Required":
  if "Is Indigenous Australian" then
    "Patient Age" >= 35
  else
    "Patient Age" >= 45

// =============================================================================
// COMPOSITE OUTPUT
// =============================================================================

define "Australia Threshold Assessment Output":
  {
    region: 'AU',
    guidelineSource: 'RACGP/Heart Foundation/Kidney Health Australia',

    // Demographics
    isIndigenous: "Is Indigenous Australian",
    isPediatric: "Is Pediatric Australia",
    isElderly: "Is Elderly Australia",
    isFrailElderly: "Is Frail Elderly Australia",

    // BMI Assessment
    bmiValue: "Latest BMI Value",
    bmiCategoryAustralia: "BMI Category Australia",
    isObeseAustralia: "Is Obese Australia Criteria",
    obesityThresholdUsed: if "Is Indigenous Australian" then "BMI Obesity Threshold Indigenous" else "BMI Obesity Threshold Australia",

    // Waist Circumference
    waistCircumference: "Latest Waist Circumference",
    waistCircumferenceRisk: "Waist Circumference Risk Australia",

    // Glycemic Status
    fastingGlucose: "Fasting Glucose mmol",
    hba1c: "Latest HbA1c",
    glycemicStatus: "Glycemic Status Australia",
    hasDiabetes: "Has Diabetes Australia Criteria",
    hba1cTarget: "Recommended HbA1c Target",
    hba1cAtTarget: "HbA1c At Target",

    // Blood Pressure
    systolicBP: "Latest Systolic BP",
    diastolicBP: "Latest Diastolic BP",
    bpCategory: "Blood Pressure Category Australia",
    hasHypertension: "Has Hypertension Australia Criteria",

    // Lipids
    lipidAssessment: "Lipid Profile Assessment Australia",

    // Renal Function
    egfr: "Latest eGFR",
    ckdStage: "CKD Stage Australia",
    uacr: "Latest UACR",
    albuminuriaCategory: "Albuminuria Category Australia",

    // Screening Requirements
    indigenousHealthCheckDue: "Indigenous Health Check Due",
    cvdScreeningDueIndigenous: "CVD Screening Due Indigenous",
    diabetesScreeningDueIndigenous: "Diabetes Screening Due Indigenous",
    cvdRiskAssessmentRequired: "CVD Risk Assessment Required"
  }
