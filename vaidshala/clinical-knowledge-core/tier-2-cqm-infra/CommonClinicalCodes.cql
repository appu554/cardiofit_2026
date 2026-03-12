/*
 * ==============================================================================
 * Common Clinical Codes Library
 * ==============================================================================
 *
 * Purpose: Defines commonly used clinical codes (LOINC, SNOMED) for quality
 *          measures and clinical decision support.
 *
 * Tier: 2 (CQM Infrastructure)
 *
 * GOVERNANCE: This library contains CLINICAL MEANING (named codes).
 *             Changes require Clinical + Technical review.
 *
 * RATIONALE: Named clinical codes belong in Tier 2+, not Tier 0-1.
 *            Tier 0-1 is grammar-only (accepts codes as parameters).
 *            This separation preserves the Grammar Lock principle.
 *
 * ==============================================================================
 */

library CommonClinicalCodes version '1.0.0'

using FHIR version '4.0.1'

// ============================================================
// CODESYSTEMS
// ============================================================

codesystem "LOINC": 'http://loinc.org'
codesystem "SNOMED": 'http://snomed.info/sct'
codesystem "RxNorm": 'http://www.nlm.nih.gov/research/umls/rxnorm'
codesystem "ICD10CM": 'http://hl7.org/fhir/sid/icd-10-cm'
codesystem "UCUM": 'http://unitsofmeasure.org'

// ============================================================
// BLOOD PRESSURE CODES
// ============================================================

/*
 * Blood pressure measurement codes per LOINC
 * Reference: https://loinc.org/85354-9/
 */
code "Systolic Blood Pressure": '8480-6' from "LOINC" display 'Systolic blood pressure'
code "Diastolic Blood Pressure": '8462-4' from "LOINC" display 'Diastolic blood pressure'
code "Blood Pressure Panel": '85354-9' from "LOINC" display 'Blood pressure panel with all children optional'

// ============================================================
// VITAL SIGNS CODES
// ============================================================

/*
 * Core vital signs per FHIR Vital Signs Profile
 * Reference: http://hl7.org/fhir/vitalsigns.html
 */
code "Heart Rate": '8867-4' from "LOINC" display 'Heart rate'
code "Respiratory Rate": '9279-1' from "LOINC" display 'Respiratory rate'
code "Body Temperature": '8310-5' from "LOINC" display 'Body temperature'
code "Oxygen Saturation": '2708-6' from "LOINC" display 'Oxygen saturation in Arterial blood'
code "BMI": '39156-5' from "LOINC" display 'Body mass index (BMI) [Ratio]'
code "Body Weight": '29463-7' from "LOINC" display 'Body weight'
code "Body Height": '8302-2' from "LOINC" display 'Body height'

// ============================================================
// DIABETES / CARDIOMETABOLIC CODES
// ============================================================

/*
 * HbA1c and glucose codes commonly used in diabetes measures
 * Reference: CMS122 (Diabetes: HbA1c Poor Control)
 */
code "HbA1c": '4548-4' from "LOINC" display 'Hemoglobin A1c/Hemoglobin.total in Blood'
code "Fasting Glucose": '1558-6' from "LOINC" display 'Fasting glucose [Mass/volume] in Serum or Plasma'
code "Random Glucose": '2339-0' from "LOINC" display 'Glucose [Mass/volume] in Blood'

// ============================================================
// RENAL FUNCTION CODES
// ============================================================

/*
 * Renal function markers for CKD staging and AKI detection
 * Reference: KDIGO CKD Guidelines
 */
code "Creatinine": '2160-0' from "LOINC" display 'Creatinine [Mass/volume] in Serum or Plasma'
code "eGFR": '33914-3' from "LOINC" display 'Glomerular filtration rate/1.73 sq M.predicted [Volume Rate/Area] in Serum, Plasma or Blood'
code "eGFR CKD-EPI": '98979-8' from "LOINC" display 'eGFR by CKD-EPI 2021'
code "BUN": '3094-0' from "LOINC" display 'Urea nitrogen [Mass/volume] in Serum or Plasma'
code "Urine Albumin": '14957-5' from "LOINC" display 'Albumin [Mass/volume] in Urine'
code "Urine Creatinine": '2161-8' from "LOINC" display 'Creatinine [Mass/volume] in Urine'
code "UACR": '9318-7' from "LOINC" display 'Albumin/Creatinine [Mass Ratio] in Urine'

// ============================================================
// LIPID PANEL CODES
// ============================================================

/*
 * Lipid markers for cardiovascular risk assessment
 * Reference: ASCVD Risk Calculator requirements
 */
code "Total Cholesterol": '2093-3' from "LOINC" display 'Cholesterol [Mass/volume] in Serum or Plasma'
code "LDL Cholesterol": '2089-1' from "LOINC" display 'Cholesterol in LDL [Mass/volume] in Serum or Plasma'
code "HDL Cholesterol": '2085-9' from "LOINC" display 'Cholesterol in HDL [Mass/volume] in Serum or Plasma'
code "Triglycerides": '2571-8' from "LOINC" display 'Triglyceride [Mass/volume] in Serum or Plasma'

// ============================================================
// HEMATOLOGY CODES
// ============================================================

/*
 * Common hematology markers
 */
code "Hemoglobin": '718-7' from "LOINC" display 'Hemoglobin [Mass/volume] in Blood'
code "Hematocrit": '4544-3' from "LOINC" display 'Hematocrit [Volume Fraction] of Blood'
code "WBC": '6690-2' from "LOINC" display 'Leukocytes [#/volume] in Blood'
code "Platelets": '26515-7' from "LOINC" display 'Platelets [#/volume] in Blood'

// ============================================================
// ELECTROLYTE CODES
// ============================================================

/*
 * Electrolyte markers for fluid/electrolyte management
 */
code "Potassium": '2823-3' from "LOINC" display 'Potassium [Moles/volume] in Serum or Plasma'
code "Sodium": '2951-2' from "LOINC" display 'Sodium [Moles/volume] in Serum or Plasma'
code "Chloride": '2075-0' from "LOINC" display 'Chloride [Moles/volume] in Serum or Plasma'
code "Bicarbonate": '1963-8' from "LOINC" display 'Bicarbonate [Moles/volume] in Serum or Plasma'
code "Calcium": '17861-6' from "LOINC" display 'Calcium [Mass/volume] in Serum or Plasma'
code "Magnesium": '19123-9' from "LOINC" display 'Magnesium [Mass/volume] in Serum or Plasma'

// ============================================================
// LIVER FUNCTION CODES
// ============================================================

/*
 * Hepatic function markers for hepatic dose adjustment
 */
code "ALT": '1742-6' from "LOINC" display 'Alanine aminotransferase [Enzymatic activity/volume] in Serum or Plasma'
code "AST": '1920-8' from "LOINC" display 'Aspartate aminotransferase [Enzymatic activity/volume] in Serum or Plasma'
code "Total Bilirubin": '1975-2' from "LOINC" display 'Bilirubin.total [Mass/volume] in Serum or Plasma'
code "Direct Bilirubin": '1968-7' from "LOINC" display 'Bilirubin.direct [Mass/volume] in Serum or Plasma'
code "Albumin": '1751-7' from "LOINC" display 'Albumin [Mass/volume] in Serum or Plasma'
code "INR": '6301-6' from "LOINC" display 'INR in Platelet poor plasma by Coagulation assay'

// ============================================================
// COAGULATION CODES
// ============================================================

/*
 * Coagulation markers for anticoagulation management
 */
code "PT": '5902-2' from "LOINC" display 'Prothrombin time (PT)'
code "aPTT": '3173-2' from "LOINC" display 'Activated partial thromboplastin time (aPTT)'

// ============================================================
// SEPSIS / CRITICAL CARE CODES
// ============================================================

/*
 * Markers for sepsis screening and SOFA scoring
 */
code "Lactate": '2524-7' from "LOINC" display 'Lactate [Moles/volume] in Serum or Plasma'
code "Procalcitonin": '33959-8' from "LOINC" display 'Procalcitonin [Mass/volume] in Serum or Plasma'
code "CRP": '1988-5' from "LOINC" display 'C reactive protein [Mass/volume] in Serum or Plasma'

// ============================================================
// MATERNAL HEALTH CODES
// ============================================================

/*
 * Codes for antenatal care and pregnancy monitoring
 */
code "Urine Protein": '2888-6' from "LOINC" display 'Protein [Mass/volume] in Urine'
code "Urine Glucose": '25428-4' from "LOINC" display 'Glucose [Presence] in Urine'
code "OGTT 2hr": '20438-8' from "LOINC" display 'Glucose [Mass/volume] in Serum or Plasma --2 hours post dose glucose'
