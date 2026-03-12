// =============================================================================
// IndiaThresholdAdapter.cql
// Tier 5: Regional Adapters - India Clinical Thresholds
// =============================================================================
//
// PURPOSE:
// Implements India-specific clinical thresholds based on ICMR (Indian Council
// of Medical Research) guidelines. These thresholds differ from Western
// guidelines due to population-specific risk profiles.
//
// KEY DIFFERENCES FROM WESTERN GUIDELINES:
// - BMI obesity threshold: ≥23 (vs ≥30 Western) - higher metabolic risk at lower BMI
// - Waist circumference: ≥90cm men, ≥80cm women (vs ≥102/88 Western)
// - Diabetes screening thresholds aligned with Indian epidemiology
//
// GOVERNANCE:
// - CQL CLASSIFIES patient status using regional thresholds
// - Does NOT override physician clinical judgment
// - All thresholds sourced from published ICMR/API guidelines
//
// SOURCES:
// - ICMR Revised Guidelines for Diagnosis and Management of Diabetes 2018
// - API Clinical Practice Guidelines for Metabolic Syndrome 2020
// - Indian Hypertension Guidelines III (IHG-III) 2013/2020
// =============================================================================

library IndiaThresholdAdapter version '1.0.0'

using FHIR version '4.0.1'

include "FHIRHelpers" version '4.0.1' called FHIRHelpers

// =============================================================================
// CODESYSTEMS
// =============================================================================

codesystem "LOINC": 'http://loinc.org'
codesystem "SNOMED": 'http://snomed.info/sct'

// Observation codes
code "BMI": '39156-5' from "LOINC" display 'Body mass index'
code "Waist Circumference": '8280-0' from "LOINC" display 'Waist circumference'
code "Fasting Glucose": '1558-6' from "LOINC" display 'Fasting glucose'
code "Random Glucose": '2339-0' from "LOINC" display 'Glucose random'
code "HbA1c": '4548-4' from "LOINC" display 'Hemoglobin A1c'
code "2hr OGTT": '1521-4' from "LOINC" display '2 hour glucose tolerance test'
code "Systolic BP": '8480-6' from "LOINC" display 'Systolic blood pressure'
code "Diastolic BP": '8462-4' from "LOINC" display 'Diastolic blood pressure'
code "Total Cholesterol": '2093-3' from "LOINC" display 'Total cholesterol'
code "LDL Cholesterol": '18262-6' from "LOINC" display 'LDL cholesterol'
code "HDL Cholesterol": '2085-9' from "LOINC" display 'HDL cholesterol'
code "Triglycerides": '2571-8' from "LOINC" display 'Triglycerides'
code "Serum Creatinine": '2160-0' from "LOINC" display 'Creatinine'
code "eGFR": '33914-3' from "LOINC" display 'eGFR'

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
// HELPER: GET LATEST OBSERVATION
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

// =============================================================================
// BMI THRESHOLDS (ICMR/API ASIA-PACIFIC CRITERIA)
// =============================================================================
// Source: WHO Expert Consultation on BMI in Asian populations (2004)
// Endorsed by ICMR and Indian Academy of Pediatrics

define "BMI Obesity Threshold India":
  23.0  // Obesity cut-off for South Asians (vs 30 for Caucasians)

define "BMI Overweight Threshold India":
  18.5  // Overweight starts at normal Western range

define "BMI Normal Lower India":
  18.5

define "BMI Normal Upper India":
  22.9

define "Latest BMI Value":
  GetLatestValue("BMI")

define "BMI Category India":
  case
    when "Latest BMI Value" is null then 'Unknown'
    when "Latest BMI Value" < 18.5 then 'Underweight'
    when "Latest BMI Value" >= 18.5 and "Latest BMI Value" < 23.0 then 'Normal'
    when "Latest BMI Value" >= 23.0 and "Latest BMI Value" < 25.0 then 'Overweight'
    when "Latest BMI Value" >= 25.0 and "Latest BMI Value" < 30.0 then 'Obese Class I'
    when "Latest BMI Value" >= 30.0 then 'Obese Class II'
    else 'Unknown'
  end

define "Is Obese India Criteria":
  "Latest BMI Value" >= "BMI Obesity Threshold India"

define "Is Overweight India Criteria":
  "Latest BMI Value" >= "BMI Overweight Threshold India" and "Latest BMI Value" < "BMI Obesity Threshold India"

// =============================================================================
// WAIST CIRCUMFERENCE THRESHOLDS (INDIAN CRITERIA)
// =============================================================================
// Source: API Metabolic Syndrome Guidelines 2020
// Abdominal obesity cut-offs for Indians

define "Waist Circumference Threshold Male India":
  90.0  // cm (vs 102 cm Western)

define "Waist Circumference Threshold Female India":
  80.0  // cm (vs 88 cm Western)

define "Latest Waist Circumference":
  GetLatestValue("Waist Circumference")

define "Has Abdominal Obesity India":
  case
    when "Latest Waist Circumference" is null then false
    when "Is Male" and "Latest Waist Circumference" >= "Waist Circumference Threshold Male India" then true
    when "Is Female" and "Latest Waist Circumference" >= "Waist Circumference Threshold Female India" then true
    else false
  end

// =============================================================================
// DIABETES THRESHOLDS (ICMR GUIDELINES 2018)
// =============================================================================
// Source: ICMR Guidelines for Management of Type 2 Diabetes 2018

define "Diabetes Fasting Glucose Threshold India":
  126.0  // mg/dL (same as ADA)

define "Prediabetes Fasting Glucose Lower India":
  100.0  // mg/dL (IFG starts)

define "Prediabetes Fasting Glucose Upper India":
  125.0  // mg/dL (IFG ends)

define "Diabetes Random Glucose Threshold India":
  200.0  // mg/dL with symptoms

define "Diabetes 2hr OGTT Threshold India":
  200.0  // mg/dL

define "Prediabetes 2hr OGTT Lower India":
  140.0  // mg/dL (IGT starts)

define "Prediabetes 2hr OGTT Upper India":
  199.0  // mg/dL (IGT ends)

define "Diabetes HbA1c Threshold India":
  6.5  // % (same as ADA)

define "Prediabetes HbA1c Lower India":
  5.7  // %

define "Prediabetes HbA1c Upper India":
  6.4  // %

define "Latest Fasting Glucose":
  GetLatestValue("Fasting Glucose")

define "Latest HbA1c":
  GetLatestValue("HbA1c")

define "Glycemic Status India":
  case
    when "Latest HbA1c" >= "Diabetes HbA1c Threshold India" then 'Diabetes'
    when "Latest Fasting Glucose" >= "Diabetes Fasting Glucose Threshold India" then 'Diabetes'
    when "Latest HbA1c" >= "Prediabetes HbA1c Lower India" and "Latest HbA1c" <= "Prediabetes HbA1c Upper India" then 'Prediabetes'
    when "Latest Fasting Glucose" >= "Prediabetes Fasting Glucose Lower India" and "Latest Fasting Glucose" <= "Prediabetes Fasting Glucose Upper India" then 'Prediabetes'
    when "Latest Fasting Glucose" < "Prediabetes Fasting Glucose Lower India" then 'Normal'
    else 'Unknown'
  end

define "Has Diabetes India Criteria":
  "Glycemic Status India" = 'Diabetes'

define "Has Prediabetes India Criteria":
  "Glycemic Status India" = 'Prediabetes'

// =============================================================================
// HYPERTENSION THRESHOLDS (IHG-III GUIDELINES)
// =============================================================================
// Source: Indian Hypertension Guidelines III (IHG-III) 2020

define "Hypertension Stage 1 Systolic Threshold India":
  140.0  // mmHg

define "Hypertension Stage 1 Diastolic Threshold India":
  90.0  // mmHg

define "Hypertension Stage 2 Systolic Threshold India":
  160.0  // mmHg

define "Hypertension Stage 2 Diastolic Threshold India":
  100.0  // mmHg

define "Elevated BP Systolic Lower India":
  120.0  // mmHg

define "Elevated BP Systolic Upper India":
  139.0  // mmHg

define "Elevated BP Diastolic Lower India":
  80.0  // mmHg

define "Elevated BP Diastolic Upper India":
  89.0  // mmHg

define "Latest Systolic BP":
  GetLatestValue("Systolic BP")

define "Latest Diastolic BP":
  GetLatestValue("Diastolic BP")

define "Blood Pressure Category India":
  case
    when "Latest Systolic BP" is null or "Latest Diastolic BP" is null then 'Unknown'
    when "Latest Systolic BP" >= "Hypertension Stage 2 Systolic Threshold India"
         or "Latest Diastolic BP" >= "Hypertension Stage 2 Diastolic Threshold India" then 'Hypertension Stage 2'
    when "Latest Systolic BP" >= "Hypertension Stage 1 Systolic Threshold India"
         or "Latest Diastolic BP" >= "Hypertension Stage 1 Diastolic Threshold India" then 'Hypertension Stage 1'
    when ("Latest Systolic BP" >= "Elevated BP Systolic Lower India" and "Latest Systolic BP" <= "Elevated BP Systolic Upper India")
         or ("Latest Diastolic BP" >= "Elevated BP Diastolic Lower India" and "Latest Diastolic BP" <= "Elevated BP Diastolic Upper India") then 'Elevated'
    when "Latest Systolic BP" < 120 and "Latest Diastolic BP" < 80 then 'Normal'
    else 'Unknown'
  end

define "Has Hypertension India Criteria":
  "Blood Pressure Category India" in { 'Hypertension Stage 1', 'Hypertension Stage 2' }

// =============================================================================
// LIPID THRESHOLDS (API GUIDELINES)
// =============================================================================
// Source: API Dyslipidemia Guidelines 2016

define "Total Cholesterol Desirable India":
  200.0  // mg/dL

define "Total Cholesterol Borderline High India":
  240.0  // mg/dL

define "LDL Optimal India":
  100.0  // mg/dL

define "LDL Near Optimal India":
  130.0  // mg/dL

define "LDL Borderline High India":
  160.0  // mg/dL

define "LDL High India":
  190.0  // mg/dL

define "HDL Low Male India":
  40.0  // mg/dL

define "HDL Low Female India":
  50.0  // mg/dL

define "Triglycerides Normal India":
  150.0  // mg/dL

define "Triglycerides Borderline High India":
  200.0  // mg/dL

define "Triglycerides High India":
  500.0  // mg/dL

define "Latest Total Cholesterol":
  GetLatestValue("Total Cholesterol")

define "Latest LDL":
  GetLatestValue("LDL Cholesterol")

define "Latest HDL":
  GetLatestValue("HDL Cholesterol")

define "Latest Triglycerides":
  GetLatestValue("Triglycerides")

define "Has Dyslipidemia India":
  "Latest Total Cholesterol" >= "Total Cholesterol Desirable India"
    or "Latest LDL" >= "LDL Near Optimal India"
    or ("Is Male" and "Latest HDL" < "HDL Low Male India")
    or ("Is Female" and "Latest HDL" < "HDL Low Female India")
    or "Latest Triglycerides" >= "Triglycerides Normal India"

// =============================================================================
// METABOLIC SYNDROME (API MODIFIED NCEP-ATP III FOR INDIANS)
// =============================================================================
// Source: API Metabolic Syndrome Guidelines 2020
// Requires 3 of 5 criteria

define "Metabolic Syndrome Criteria Count India":
  (if "Has Abdominal Obesity India" then 1 else 0)
  + (if "Latest Triglycerides" >= 150 then 1 else 0)
  + (if ("Is Male" and "Latest HDL" < 40) or ("Is Female" and "Latest HDL" < 50) then 1 else 0)
  + (if "Latest Systolic BP" >= 130 or "Latest Diastolic BP" >= 85 then 1 else 0)
  + (if "Latest Fasting Glucose" >= 100 then 1 else 0)

define "Has Metabolic Syndrome India":
  "Metabolic Syndrome Criteria Count India" >= 3

// =============================================================================
// PEDIATRIC AGE THRESHOLDS
// =============================================================================

define "Pediatric Age Threshold India":
  18  // years (for pediatric protocols)

define "Adolescent Age Lower India":
  10  // years

define "Adolescent Age Upper India":
  19  // years

define "Is Pediatric India":
  "Patient Age" < "Pediatric Age Threshold India"

define "Is Adolescent India":
  "Patient Age" >= "Adolescent Age Lower India" and "Patient Age" <= "Adolescent Age Upper India"

// =============================================================================
// RENAL FUNCTION THRESHOLDS (ICMR/ISRN GUIDELINES)
// =============================================================================
// Source: Indian Society of Nephrology guidelines

define "eGFR CKD Stage 1 India":
  90.0  // mL/min/1.73m²

define "eGFR CKD Stage 2 India":
  60.0  // mL/min/1.73m²

define "eGFR CKD Stage 3a India":
  45.0  // mL/min/1.73m²

define "eGFR CKD Stage 3b India":
  30.0  // mL/min/1.73m²

define "eGFR CKD Stage 4 India":
  15.0  // mL/min/1.73m²

define "Latest eGFR":
  GetLatestValue("eGFR")

define "CKD Stage India":
  case
    when "Latest eGFR" is null then 'Unknown'
    when "Latest eGFR" >= "eGFR CKD Stage 1 India" then 'Stage 1'
    when "Latest eGFR" >= "eGFR CKD Stage 2 India" then 'Stage 2'
    when "Latest eGFR" >= "eGFR CKD Stage 3a India" then 'Stage 3a'
    when "Latest eGFR" >= "eGFR CKD Stage 3b India" then 'Stage 3b'
    when "Latest eGFR" >= "eGFR CKD Stage 4 India" then 'Stage 4'
    when "Latest eGFR" < "eGFR CKD Stage 4 India" then 'Stage 5'
    else 'Unknown'
  end

// =============================================================================
// CARDIOVASCULAR RISK (ICMR CVD RISK SCORE)
// =============================================================================
// Note: ICMR has developed India-specific CVD risk charts
// This is a simplified version - full implementation requires age/gender charts

define "High CVD Risk Age Male India":
  45  // years (earlier than Western guidelines)

define "High CVD Risk Age Female India":
  55  // years

define "Is High CVD Risk Age India":
  ("Is Male" and "Patient Age" >= "High CVD Risk Age Male India")
    or ("Is Female" and "Patient Age" >= "High CVD Risk Age Female India")

define "Has Multiple CVD Risk Factors India":
  (if "Has Hypertension India Criteria" then 1 else 0)
  + (if "Has Diabetes India Criteria" then 1 else 0)
  + (if "Has Dyslipidemia India" then 1 else 0)
  + (if "Is Obese India Criteria" then 1 else 0)
  + (if "Has Abdominal Obesity India" then 1 else 0) >= 2

// =============================================================================
// COMPOSITE OUTPUT
// =============================================================================

define "India Threshold Assessment Output":
  {
    region: 'IN',
    guidelineSource: 'ICMR/API',

    // BMI Assessment
    bmiValue: "Latest BMI Value",
    bmiCategoryIndia: "BMI Category India",
    isObeseIndia: "Is Obese India Criteria",
    obesityThresholdUsed: "BMI Obesity Threshold India",

    // Abdominal Obesity
    waistCircumference: "Latest Waist Circumference",
    hasAbdominalObesity: "Has Abdominal Obesity India",

    // Glycemic Status
    fastingGlucose: "Latest Fasting Glucose",
    hba1c: "Latest HbA1c",
    glycemicStatus: "Glycemic Status India",
    hasDiabetes: "Has Diabetes India Criteria",
    hasPrediabetes: "Has Prediabetes India Criteria",

    // Blood Pressure
    systolicBP: "Latest Systolic BP",
    diastolicBP: "Latest Diastolic BP",
    bpCategory: "Blood Pressure Category India",
    hasHypertension: "Has Hypertension India Criteria",

    // Lipids
    hasDyslipidemia: "Has Dyslipidemia India",

    // Metabolic Syndrome
    metabolicSyndromeCriteriaMet: "Metabolic Syndrome Criteria Count India",
    hasMetabolicSyndrome: "Has Metabolic Syndrome India",

    // Renal Function
    egfr: "Latest eGFR",
    ckdStage: "CKD Stage India",

    // Age Classification
    isPediatric: "Is Pediatric India",
    isAdolescent: "Is Adolescent India",

    // CVD Risk
    isHighCVDRiskAge: "Is High CVD Risk Age India",
    hasMultipleCVDRiskFactors: "Has Multiple CVD Risk Factors India"
  }
