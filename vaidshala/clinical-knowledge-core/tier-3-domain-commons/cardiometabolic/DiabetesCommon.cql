/*
 * DiabetesCommon CQL Library
 * ============================================================
 * Tier 3 Domain Commons - Diabetes Clinical Patterns
 *
 * Purpose: Shared diabetes definitions consumed by:
 *   - ICU Intelligence (Tier 7) - Glycemic emergencies
 *   - KB-3 Guidelines (Diabetes)
 *   - KB-4 Patient Safety (Hypoglycemia, DKA)
 *   - CMS Diabetes Measures (CMS122, CMS131, etc.)
 *
 * Clinical Domains:
 *   - Glycemic Control (HbA1c, Glucose)
 *   - Hypoglycemia Detection
 *   - Hyperglycemic Emergencies (DKA, HHS)
 *   - Diabetes Complications
 *   - Insulin Management
 *   - SGLT2i/GLP1 Monitoring
 *
 * Governance: Tier-3 review required (Clinical + Technical + Endocrine)
 * ============================================================
 */
library DiabetesCommon version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.4.000' called FHIRHelpers
include IntervalHelpers version '1.0.0' called Intervals
include ObservationHelpers version '1.0.0' called Observations
include MedicationHelpers version '1.0.0' called Medications

// ============================================================
// VALUE SET REFERENCES (KB-7 Terminology)
// ============================================================

// Diabetes Conditions
valueset "Diabetes Mellitus Type 1": 'http://cardiofit.io/fhir/ValueSet/dm-type1'
valueset "Diabetes Mellitus Type 2": 'http://cardiofit.io/fhir/ValueSet/dm-type2'
valueset "Gestational Diabetes": 'http://cardiofit.io/fhir/ValueSet/dm-gestational'
valueset "Prediabetes Conditions": 'http://cardiofit.io/fhir/ValueSet/prediabetes'
valueset "DKA Conditions": 'http://cardiofit.io/fhir/ValueSet/dka'
valueset "HHS Conditions": 'http://cardiofit.io/fhir/ValueSet/hhs'

// Diabetes Complications
valueset "Diabetic Nephropathy": 'http://cardiofit.io/fhir/ValueSet/diabetic-nephropathy'
valueset "Diabetic Retinopathy": 'http://cardiofit.io/fhir/ValueSet/diabetic-retinopathy'
valueset "Diabetic Neuropathy": 'http://cardiofit.io/fhir/ValueSet/diabetic-neuropathy'
valueset "Diabetic Foot Conditions": 'http://cardiofit.io/fhir/ValueSet/diabetic-foot'

// Diabetes Medications
valueset "Insulins": 'http://cardiofit.io/fhir/ValueSet/insulins'
valueset "Rapid Acting Insulins": 'http://cardiofit.io/fhir/ValueSet/insulins-rapid'
valueset "Long Acting Insulins": 'http://cardiofit.io/fhir/ValueSet/insulins-long'
valueset "Insulin Drip": 'http://cardiofit.io/fhir/ValueSet/insulin-drip'
valueset "Metformin Medications": 'http://cardiofit.io/fhir/ValueSet/metformin'
valueset "Sulfonylureas": 'http://cardiofit.io/fhir/ValueSet/sulfonylureas'
valueset "SGLT2 Inhibitors": 'http://cardiofit.io/fhir/ValueSet/sglt2-inhibitors'
valueset "GLP1 Agonists": 'http://cardiofit.io/fhir/ValueSet/glp1-agonists'
valueset "DPP4 Inhibitors": 'http://cardiofit.io/fhir/ValueSet/dpp4-inhibitors'
valueset "Thiazolidinediones": 'http://cardiofit.io/fhir/ValueSet/tzd'

// ============================================================
// CODESYSTEMS
// ============================================================
codesystem "LOINC": 'http://loinc.org'
codesystem "SNOMED": 'http://snomed.info/sct'

// ============================================================
// OBSERVATION CODES (LOINC)
// ============================================================

// Glycemic Markers
code "HbA1c Code": '4548-4' from "LOINC" display 'Hemoglobin A1c'
code "HbA1c IFCC Code": '59261-8' from "LOINC" display 'Hemoglobin A1c IFCC'
code "Fasting Glucose Code": '1558-6' from "LOINC" display 'Fasting glucose'
code "Random Glucose Code": '2345-7' from "LOINC" display 'Glucose'
code "2h OGTT Glucose Code": '20438-8' from "LOINC" display 'Glucose 2h post 75g glucose PO'

// POC / Bedside Glucose
code "POC Glucose Code": '41653-7' from "LOINC" display 'Glucose [Mass/volume] Capillary'
code "Blood Glucose POC Code": '14749-6' from "LOINC" display 'Glucose POC'

// Ketones and Acidosis Markers
code "Serum Ketones Code": '53061-8' from "LOINC" display 'Ketones [Mass/volume] in Blood'
code "Beta-Hydroxybutyrate Code": '53061-8' from "LOINC" display 'Beta-hydroxybutyrate'
code "Urine Ketones Code": '2514-8' from "LOINC" display 'Ketones [Presence] in Urine'
code "Anion Gap Code": '33037-3' from "LOINC" display 'Anion gap'
code "Serum Bicarbonate Code": '1963-8' from "LOINC" display 'Bicarbonate [Moles/volume] in Serum'
code "Blood pH Code": '2744-1' from "LOINC" display 'pH of Blood'
code "Serum Osmolality Code": '2692-2' from "LOINC" display 'Osmolality of Serum'

// Renal (for diabetes complications)
code "Urine Albumin/Creatinine Code": '9318-7' from "LOINC" display 'Albumin/Creatinine [Mass ratio] in Urine'

// C-Peptide (for insulin production assessment)
code "C-Peptide Code": '1986-9' from "LOINC" display 'C-peptide [Mass/volume] in Serum'

// ============================================================
// CONTEXT
// ============================================================
context Patient

// ============================================================
// DIABETES DIAGNOSIS FLAGS
// ============================================================

define "Has Type 1 Diabetes":
  exists [Condition: "Diabetes Mellitus Type 1"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

define "Has Type 2 Diabetes":
  exists [Condition: "Diabetes Mellitus Type 2"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

define "Has Gestational Diabetes":
  exists [Condition: "Gestational Diabetes"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

define "Has Any Diabetes":
  "Has Type 1 Diabetes" or "Has Type 2 Diabetes" or "Has Gestational Diabetes"

define "Has Prediabetes":
  exists [Condition: "Prediabetes Conditions"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

// ============================================================
// HbA1c RETRIEVAL AND CLASSIFICATION
// ============================================================

/*
 * Most Recent HbA1c (considers both DCCT and IFCC codes)
 */
define "Most Recent HbA1c":
  Observations.MostRecentObservation(
    ([Observation: "HbA1c Code"] union [Observation: "HbA1c IFCC Code"]) O
      where Observations.IsValidObservation(O)
  )

define "Most Recent HbA1c Value":
  Observations.ObservationNumericValue("Most Recent HbA1c")

define "Most Recent HbA1c Date":
  Observations.ObservationDateTime("Most Recent HbA1c")

/*
 * HbA1c Classifications (ADA Guidelines)
 */
define "HbA1c At Target":
  "Most Recent HbA1c Value" < 7.0

define "HbA1c Near Target":
  "Most Recent HbA1c Value" >= 7.0 and "Most Recent HbA1c Value" < 8.0

define "HbA1c Above Target":
  "Most Recent HbA1c Value" >= 8.0 and "Most Recent HbA1c Value" < 9.0

define "HbA1c Poorly Controlled":
  "Most Recent HbA1c Value" >= 9.0

define "HbA1c Very Poorly Controlled":
  "Most Recent HbA1c Value" >= 10.0

define "HbA1c Indicates Prediabetes":
  "Most Recent HbA1c Value" >= 5.7 and "Most Recent HbA1c Value" < 6.5

define "HbA1c Indicates Diabetes":
  "Most Recent HbA1c Value" >= 6.5

// ============================================================
// GLUCOSE RETRIEVAL
// ============================================================

/*
 * Most Recent Blood Glucose (any type)
 */
define "Most Recent Glucose":
  Observations.MostRecentObservation(
    ([Observation: "Random Glucose Code"]
     union [Observation: "Fasting Glucose Code"]
     union [Observation: "POC Glucose Code"]
     union [Observation: "Blood Glucose POC Code"]) O
      where Observations.IsValidObservation(O)
  )

define "Most Recent Glucose Value":
  Observations.ObservationNumericValue("Most Recent Glucose")

define "Most Recent Glucose DateTime":
  Observations.ObservationDateTime("Most Recent Glucose")

/*
 * Most Recent Fasting Glucose
 */
define "Most Recent Fasting Glucose":
  Observations.MostRecentObservation(
    [Observation: "Fasting Glucose Code"] O
      where Observations.IsValidObservation(O)
  )

define "Most Recent Fasting Glucose Value":
  Observations.ObservationNumericValue("Most Recent Fasting Glucose")

// ============================================================
// HYPOGLYCEMIA DETECTION (CRITICAL FOR ICU INTELLIGENCE)
// ============================================================

/*
 * Hypoglycemia: Glucose < 70 mg/dL
 * ADA Level 1 Hypoglycemia
 */
define "Has Hypoglycemia":
  "Most Recent Glucose Value" < 70

/*
 * Clinically Significant Hypoglycemia: Glucose < 54 mg/dL
 * ADA Level 2 Hypoglycemia - CRITICAL
 */
define "Has Clinically Significant Hypoglycemia":
  "Most Recent Glucose Value" < 54

/*
 * Severe Hypoglycemia: Glucose < 40 mg/dL
 * ADA Level 3 / Near-fatal - CRITICAL for ICU Intelligence
 */
define "Has Severe Hypoglycemia":
  "Most Recent Glucose Value" < 40

/*
 * Nocturnal Hypoglycemia Risk Flag
 * (on insulin or sulfonylurea with recent hypoglycemia)
 */
define "At Risk for Nocturnal Hypoglycemia":
  ("On Insulin" or "On Sulfonylurea") and "Has Hypoglycemia"

// ============================================================
// HYPERGLYCEMIA AND EMERGENCY DETECTION
// ============================================================

/*
 * Hyperglycemia: Glucose > 180 mg/dL (inpatient threshold)
 */
define "Has Hyperglycemia":
  "Most Recent Glucose Value" > 180

/*
 * Significant Hyperglycemia: Glucose > 250 mg/dL
 */
define "Has Significant Hyperglycemia":
  "Most Recent Glucose Value" > 250

/*
 * Severe Hyperglycemia: Glucose > 400 mg/dL
 * DKA/HHS consideration - CRITICAL for ICU Intelligence
 */
define "Has Severe Hyperglycemia":
  "Most Recent Glucose Value" > 400

/*
 * Critical Hyperglycemia: Glucose > 600 mg/dL
 * HHS likely - CRITICAL
 */
define "Has Critical Hyperglycemia":
  "Most Recent Glucose Value" > 600

// ============================================================
// DKA/HHS BIOCHEMICAL MARKERS
// ============================================================

define "Most Recent Anion Gap":
  Observations.MostRecentObservation(
    [Observation: "Anion Gap Code"] O
      where Observations.IsValidObservation(O)
  )

define "Most Recent Anion Gap Value":
  Observations.ObservationNumericValue("Most Recent Anion Gap")

define "Most Recent Bicarbonate":
  Observations.MostRecentObservation(
    [Observation: "Serum Bicarbonate Code"] O
      where Observations.IsValidObservation(O)
  )

define "Most Recent Bicarbonate Value":
  Observations.ObservationNumericValue("Most Recent Bicarbonate")

define "Most Recent Blood pH":
  Observations.MostRecentObservation(
    [Observation: "Blood pH Code"] O
      where Observations.IsValidObservation(O)
  )

define "Most Recent Blood pH Value":
  Observations.ObservationNumericValue("Most Recent Blood pH")

define "Most Recent Serum Osmolality":
  Observations.MostRecentObservation(
    [Observation: "Serum Osmolality Code"] O
      where Observations.IsValidObservation(O)
  )

define "Most Recent Serum Osmolality Value":
  Observations.ObservationNumericValue("Most Recent Serum Osmolality")

define "Most Recent Beta-Hydroxybutyrate":
  Observations.MostRecentObservation(
    [Observation: "Beta-Hydroxybutyrate Code"] O
      where Observations.IsValidObservation(O)
  )

define "Most Recent Beta-Hydroxybutyrate Value":
  Observations.ObservationNumericValue("Most Recent Beta-Hydroxybutyrate")

/*
 * Elevated Anion Gap: > 12 mEq/L
 */
define "Has Elevated Anion Gap":
  "Most Recent Anion Gap Value" > 12

/*
 * Significantly Elevated Anion Gap: > 16 mEq/L (DKA consideration)
 */
define "Has Significantly Elevated Anion Gap":
  "Most Recent Anion Gap Value" > 16

/*
 * Metabolic Acidosis: pH < 7.35 OR Bicarbonate < 22
 */
define "Has Metabolic Acidosis":
  "Most Recent Blood pH Value" < 7.35
  or "Most Recent Bicarbonate Value" < 22

/*
 * Severe Metabolic Acidosis: pH < 7.25 OR Bicarbonate < 15
 * CRITICAL for ICU Intelligence
 */
define "Has Severe Metabolic Acidosis":
  "Most Recent Blood pH Value" < 7.25
  or "Most Recent Bicarbonate Value" < 15

/*
 * Hyperosmolar State: Osmolality > 320 mOsm/kg
 * HHS indicator - CRITICAL
 */
define "Has Hyperosmolar State":
  "Most Recent Serum Osmolality Value" > 320

/*
 * Elevated Ketones: Beta-hydroxybutyrate > 3.0 mmol/L
 */
define "Has Elevated Ketones":
  "Most Recent Beta-Hydroxybutyrate Value" > 3.0

// ============================================================
// DKA/HHS SYNDROME FLAGS (CRITICAL FOR ICU)
// ============================================================

/*
 * DKA Biochemical Criteria (Simplified)
 * Glucose >250 + Acidosis + Elevated Anion Gap + Ketones
 * NOTE: Full DKA diagnosis requires clinical assessment
 */
define "Meets DKA Biochemical Criteria":
  "Most Recent Glucose Value" > 250
  and "Has Metabolic Acidosis"
  and "Has Elevated Anion Gap"

/*
 * Severe DKA Biochemical Criteria
 * CRITICAL for ICU Intelligence
 */
define "Meets Severe DKA Criteria":
  "Most Recent Glucose Value" > 250
  and "Has Severe Metabolic Acidosis"
  and "Has Significantly Elevated Anion Gap"

/*
 * HHS Biochemical Criteria (Simplified)
 * Glucose >600 + Hyperosmolar + Minimal/no ketosis
 */
define "Meets HHS Biochemical Criteria":
  "Most Recent Glucose Value" > 600
  and "Has Hyperosmolar State"
  and "Most Recent Blood pH Value" >= 7.30

/*
 * Active DKA Diagnosis
 */
define "Has Active DKA":
  exists [Condition: "DKA Conditions"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

/*
 * Active HHS Diagnosis
 */
define "Has Active HHS":
  exists [Condition: "HHS Conditions"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

// ============================================================
// DIABETES COMPLICATIONS
// ============================================================

define "Has Diabetic Nephropathy":
  exists [Condition: "Diabetic Nephropathy"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

define "Has Diabetic Retinopathy":
  exists [Condition: "Diabetic Retinopathy"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

define "Has Diabetic Neuropathy":
  exists [Condition: "Diabetic Neuropathy"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

define "Has Diabetic Foot":
  exists [Condition: "Diabetic Foot Conditions"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

define "Has Any Microvascular Complication":
  "Has Diabetic Nephropathy"
  or "Has Diabetic Retinopathy"
  or "Has Diabetic Neuropathy"

// ============================================================
// ALBUMINURIA (Diabetic Kidney Disease Staging)
// ============================================================

define "Most Recent UACR":
  Observations.MostRecentObservation(
    [Observation: "Urine Albumin/Creatinine Code"] O
      where Observations.IsValidObservation(O)
  )

define "Most Recent UACR Value":
  Observations.ObservationNumericValue("Most Recent UACR")

/*
 * UACR Classifications (mg/g Cr)
 */
define "Has Normal Albuminuria":
  "Most Recent UACR Value" < 30

define "Has Moderately Increased Albuminuria":
  "Most Recent UACR Value" >= 30 and "Most Recent UACR Value" < 300

define "Has Severely Increased Albuminuria":
  "Most Recent UACR Value" >= 300

// ============================================================
// DIABETES MEDICATION STATUS
// ============================================================

define "On Insulin":
  exists [MedicationRequest: "Insulins"] MR
    where Medications.IsMedicationActive(MR)

define "On Rapid Acting Insulin":
  exists [MedicationRequest: "Rapid Acting Insulins"] MR
    where Medications.IsMedicationActive(MR)

define "On Long Acting Insulin":
  exists [MedicationRequest: "Long Acting Insulins"] MR
    where Medications.IsMedicationActive(MR)

define "On Insulin Drip":
  exists [MedicationRequest: "Insulin Drip"] MR
    where Medications.IsMedicationActive(MR)

define "On Metformin":
  exists [MedicationRequest: "Metformin Medications"] MR
    where Medications.IsMedicationActive(MR)

define "On Sulfonylurea":
  exists [MedicationRequest: "Sulfonylureas"] MR
    where Medications.IsMedicationActive(MR)

define "On SGLT2 Inhibitor":
  exists [MedicationRequest: "SGLT2 Inhibitors"] MR
    where Medications.IsMedicationActive(MR)

define "On GLP1 Agonist":
  exists [MedicationRequest: "GLP1 Agonists"] MR
    where Medications.IsMedicationActive(MR)

define "On DPP4 Inhibitor":
  exists [MedicationRequest: "DPP4 Inhibitors"] MR
    where Medications.IsMedicationActive(MR)

define "On TZD":
  exists [MedicationRequest: "Thiazolidinediones"] MR
    where Medications.IsMedicationActive(MR)

/*
 * High Hypoglycemia Risk Medications
 */
define "On High Hypoglycemia Risk Medications":
  "On Insulin" or "On Sulfonylurea"

/*
 * Diabetes Medication Count (polypharmacy tracking)
 */
define "Diabetes Medication Count":
  (if "On Insulin" then 1 else 0) +
  (if "On Metformin" then 1 else 0) +
  (if "On Sulfonylurea" then 1 else 0) +
  (if "On SGLT2 Inhibitor" then 1 else 0) +
  (if "On GLP1 Agonist" then 1 else 0) +
  (if "On DPP4 Inhibitor" then 1 else 0) +
  (if "On TZD" then 1 else 0)

// ============================================================
// SGLT2i SAFETY MONITORING (KB-4 Input)
// ============================================================

/*
 * SGLT2i Euglycemic DKA Risk
 * DKA biochemical markers with near-normal glucose on SGLT2i
 */
define "At Risk for Euglycemic DKA":
  "On SGLT2 Inhibitor"
  and "Has Metabolic Acidosis"
  and "Most Recent Glucose Value" < 250

// ============================================================
// GLYCEMIC VARIABILITY FLAGS
// ============================================================

/*
 * Glucose Observations in Last 24 Hours
 */
define "Recent Glucose Observations":
  ([Observation: "Random Glucose Code"]
   union [Observation: "POC Glucose Code"]
   union [Observation: "Blood Glucose POC Code"]) O
    where Observations.IsValidObservation(O)
    and Observations.ObservationDateTime(O) >= Now() - 24 hours

/*
 * Glucose Range in Last 24h
 */
define "Recent Glucose Max":
  Observations.MaxObservationValue("Recent Glucose Observations")

define "Recent Glucose Min":
  Observations.MinObservationValue("Recent Glucose Observations")

define "Recent Glucose Range":
  "Recent Glucose Max" - "Recent Glucose Min"

/*
 * High Glycemic Variability: Range > 150 mg/dL in 24h
 */
define "Has High Glycemic Variability":
  "Recent Glucose Range" > 150

// ============================================================
// GLYCEMIC RISK SCORING (For ICU Intelligence)
// ============================================================

/*
 * Glycemic Emergency Risk Score
 * OUTPUT: Integer score for classification
 * NOTE: This CLASSIFIES risk. ICU DECIDES actions.
 */
define "Glycemic Emergency Risk Score":
  (if "Has Severe Hypoglycemia" then 5 else
   if "Has Clinically Significant Hypoglycemia" then 3 else
   if "Has Hypoglycemia" then 1 else 0) +
  (if "Meets Severe DKA Criteria" then 5 else
   if "Meets DKA Biochemical Criteria" then 3 else 0) +
  (if "Meets HHS Biochemical Criteria" then 5 else
   if "Has Critical Hyperglycemia" then 3 else
   if "Has Severe Hyperglycemia" then 2 else 0) +
  (if "Has Severe Metabolic Acidosis" then 3 else
   if "Has Metabolic Acidosis" then 1 else 0) +
  (if "On Insulin Drip" then 1 else 0) +
  (if "At Risk for Euglycemic DKA" then 2 else 0)

define "Glycemic Emergency Risk Level":
  case
    when "Glycemic Emergency Risk Score" >= 8 then 'CRITICAL'
    when "Glycemic Emergency Risk Score" >= 5 then 'HIGH'
    when "Glycemic Emergency Risk Score" >= 2 then 'MODERATE'
    else 'LOW'
  end

// ============================================================
// CMS MEASURE SUPPORT
// ============================================================

/*
 * CMS122: Diabetes HbA1c Poor Control (>9%)
 */
define "CMS122 Numerator":
  "Most Recent HbA1c Value" > 9.0
  or "Most Recent HbA1c" is null

/*
 * Has HbA1c in Measurement Period (typically 2 years)
 */
define "Has Recent HbA1c":
  "Most Recent HbA1c Date" >= Now() - 2 years

// ============================================================
// SUMMARY OUTPUTS FOR ICU INTELLIGENCE
// ============================================================

/*
 * DiabetesFacts - structured output for Go consumption
 * This is what the KnowledgeSnapshotBuilder will capture
 */
define "ICU Diabetes Facts":
  {
    // Diagnosis
    hasType1Diabetes: "Has Type 1 Diabetes",
    hasType2Diabetes: "Has Type 2 Diabetes",
    hasGestationalDiabetes: "Has Gestational Diabetes",
    hasAnyDiabetes: "Has Any Diabetes",

    // Glycemic Control
    hba1cValue: "Most Recent HbA1c Value",
    hba1cAtTarget: "HbA1c At Target",
    hba1cPoorlyControlled: "HbA1c Poorly Controlled",

    // Acute Glucose
    glucoseValue: "Most Recent Glucose Value",
    hasSevereHypoglycemia: "Has Severe Hypoglycemia",
    hasClinicallySignificantHypoglycemia: "Has Clinically Significant Hypoglycemia",
    hasHypoglycemia: "Has Hypoglycemia",
    hasSevereHyperglycemia: "Has Severe Hyperglycemia",
    hasCriticalHyperglycemia: "Has Critical Hyperglycemia",

    // DKA/HHS
    meetsDKACriteria: "Meets DKA Biochemical Criteria",
    meetsSevereDKACriteria: "Meets Severe DKA Criteria",
    meetsHHSCriteria: "Meets HHS Biochemical Criteria",
    hasActiveDKA: "Has Active DKA",
    hasActiveHHS: "Has Active HHS",
    atRiskForEuglycemicDKA: "At Risk for Euglycemic DKA",

    // Biochemical Markers
    anionGapValue: "Most Recent Anion Gap Value",
    bicarbonateValue: "Most Recent Bicarbonate Value",
    bloodpHValue: "Most Recent Blood pH Value",
    serumOsmolalityValue: "Most Recent Serum Osmolality Value",
    hasMetabolicAcidosis: "Has Metabolic Acidosis",
    hasSevereMetabolicAcidosis: "Has Severe Metabolic Acidosis",

    // Medications
    onInsulin: "On Insulin",
    onInsulinDrip: "On Insulin Drip",
    onHighHypoglycemiaRiskMeds: "On High Hypoglycemia Risk Medications",
    onSGLT2Inhibitor: "On SGLT2 Inhibitor",

    // Risk
    glycemicEmergencyRiskLevel: "Glycemic Emergency Risk Level",
    glycemicEmergencyRiskScore: "Glycemic Emergency Risk Score",

    // Complications
    hasAnyMicrovascularComplication: "Has Any Microvascular Complication"
  }

