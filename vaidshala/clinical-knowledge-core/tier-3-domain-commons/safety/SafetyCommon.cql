/*
 * SafetyCommon CQL Library
 * ============================================================
 * Tier 3 Domain Commons - Clinical Safety Patterns
 *
 * Purpose: Shared safety definitions consumed by:
 *   - ICU Intelligence (Tier 7)
 *   - KB-4 Patient Safety
 *   - KB-5 Drug Interactions
 *
 * Governance: Tier-3 review required (Clinical + Technical + Safety)
 * ============================================================
 */
library SafetyCommon version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.4.000' called FHIRHelpers
include IntervalHelpers version '1.0.0' called Intervals
include ObservationHelpers version '1.0.0' called Observations
include MedicationHelpers version '1.0.0' called Medications

// ============================================================
// VALUE SET REFERENCES (KB-7 Terminology)
// ============================================================

// High-Alert Medications (ISMP categories)
valueset "High Alert Medications": 'http://cardiofit.io/fhir/ValueSet/high-alert-medications'
valueset "Anticoagulants": 'http://cardiofit.io/fhir/ValueSet/anticoagulants'
valueset "Opioids": 'http://cardiofit.io/fhir/ValueSet/opioids'
valueset "Insulins": 'http://cardiofit.io/fhir/ValueSet/insulins'
valueset "Chemotherapy Agents": 'http://cardiofit.io/fhir/ValueSet/chemotherapy-agents'
valueset "Sedatives": 'http://cardiofit.io/fhir/ValueSet/sedatives'
valueset "Neuromuscular Blockers": 'http://cardiofit.io/fhir/ValueSet/neuromuscular-blockers'

// Allergen Classes
valueset "Penicillin Allergens": 'http://cardiofit.io/fhir/ValueSet/penicillin-allergens'
valueset "Sulfa Allergens": 'http://cardiofit.io/fhir/ValueSet/sulfa-allergens'
valueset "NSAID Allergens": 'http://cardiofit.io/fhir/ValueSet/nsaid-allergens'
valueset "Contrast Allergens": 'http://cardiofit.io/fhir/ValueSet/contrast-allergens'

// Critical Conditions
valueset "Anaphylaxis Conditions": 'http://cardiofit.io/fhir/ValueSet/anaphylaxis-conditions'
valueset "Severe Bleeding Conditions": 'http://cardiofit.io/fhir/ValueSet/severe-bleeding'
valueset "Acute Kidney Injury": 'http://cardiofit.io/fhir/ValueSet/aki-conditions'
valueset "Acute Liver Failure": 'http://cardiofit.io/fhir/ValueSet/alf-conditions'

// ============================================================
// CONTEXT
// ============================================================
context Patient

// ============================================================
// HIGH-ALERT MEDICATION DETECTION
// ============================================================

/*
 * Check if patient has any active high-alert medications
 * Returns: List of active high-alert medication requests
 */
define "Active High Alert Medications":
  [MedicationRequest: "High Alert Medications"] MR
    where Medications.IsMedicationActive(MR)

define "Has Active High Alert Medication":
  exists "Active High Alert Medications"

/*
 * Check for specific high-alert categories
 */
define "Active Anticoagulants":
  [MedicationRequest: "Anticoagulants"] MR
    where Medications.IsMedicationActive(MR)

define "Active Opioids":
  [MedicationRequest: "Opioids"] MR
    where Medications.IsMedicationActive(MR)

define "Active Insulins":
  [MedicationRequest: "Insulins"] MR
    where Medications.IsMedicationActive(MR)

define "Active Sedatives":
  [MedicationRequest: "Sedatives"] MR
    where Medications.IsMedicationActive(MR)

define "Active Neuromuscular Blockers":
  [MedicationRequest: "Neuromuscular Blockers"] MR
    where Medications.IsMedicationActive(MR)

define "On Multiple High Alert Categories":
  (if exists "Active Anticoagulants" then 1 else 0) +
  (if exists "Active Opioids" then 1 else 0) +
  (if exists "Active Insulins" then 1 else 0) +
  (if exists "Active Sedatives" then 1 else 0) > 1

// ============================================================
// ALLERGY DETECTION
// ============================================================

/*
 * Active confirmed allergies (not refuted/entered-in-error)
 */
define "Active Allergies":
  [AllergyIntolerance] A
    where A.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { system: FHIR.uri { value: 'http://terminology.hl7.org/CodeSystem/allergyintolerance-clinical' }, code: FHIR.code { value: 'active' } } }
    })
    and A.verificationStatus !~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { system: FHIR.uri { value: 'http://terminology.hl7.org/CodeSystem/allergyintolerance-verification' }, code: FHIR.code { value: 'entered-in-error' } } }
    })

/*
 * Check for specific allergen class allergies
 */
define "Has Penicillin Allergy":
  exists ([AllergyIntolerance: "Penicillin Allergens"] A
    where A.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    }))

define "Has Sulfa Allergy":
  exists ([AllergyIntolerance: "Sulfa Allergens"] A
    where A.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    }))

define "Has NSAID Allergy":
  exists ([AllergyIntolerance: "NSAID Allergens"] A
    where A.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    }))

define "Has Contrast Allergy":
  exists ([AllergyIntolerance: "Contrast Allergens"] A
    where A.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    }))

/*
 * Severe allergy history (anaphylaxis)
 */
define "History of Anaphylaxis":
  exists [Condition: "Anaphylaxis Conditions"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })
    or C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'resolved' } } }
    })

// ============================================================
// CRITICAL CONDITION DETECTION (ICU Intelligence Input)
// ============================================================

/*
 * Active severe bleeding - CRITICAL for anticoagulant safety
 */
define "Has Active Severe Bleeding":
  exists [Condition: "Severe Bleeding Conditions"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

/*
 * Acute Kidney Injury - CRITICAL for nephrotoxic drug safety
 */
define "Has Acute Kidney Injury":
  exists [Condition: "Acute Kidney Injury"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

/*
 * Acute Liver Failure - CRITICAL for hepatotoxic drug safety
 */
define "Has Acute Liver Failure":
  exists [Condition: "Acute Liver Failure"] C
    where C.clinicalStatus ~ ToConcept(FHIR.CodeableConcept {
      coding: { FHIR.Coding { code: FHIR.code { value: 'active' } } }
    })

// ============================================================
// SAFETY RISK SCORING (Classification - NOT enforcement)
// ============================================================

/*
 * Overall Safety Risk Level
 * OUTPUT: Integer score for classification
 * NOTE: This CLASSIFIES risk. KB-4/ICU ENFORCES actions.
 */
define "Safety Risk Score":
  (if "Has Active High Alert Medication" then 2 else 0) +
  (if "On Multiple High Alert Categories" then 3 else 0) +
  (if "History of Anaphylaxis" then 5 else 0) +
  (if "Has Active Severe Bleeding" then 5 else 0) +
  (if "Has Acute Kidney Injury" then 4 else 0) +
  (if "Has Acute Liver Failure" then 4 else 0)

define "Safety Risk Level":
  case
    when "Safety Risk Score" >= 10 then 'CRITICAL'
    when "Safety Risk Score" >= 5 then 'HIGH'
    when "Safety Risk Score" >= 2 then 'MODERATE'
    else 'LOW'
  end

// ============================================================
// DRUG-CONDITION CONTRAINDICATION FLAGS
// ============================================================

/*
 * Anticoagulant Contraindication Flags
 * OUTPUT: Boolean flags for ICU Intelligence consumption
 */
define "Anticoagulant Contraindicated Due To Bleeding":
  "Has Active Severe Bleeding" and exists "Active Anticoagulants"

define "NSAID Contraindicated Due To AKI":
  "Has Acute Kidney Injury"

define "Hepatotoxic Drug Contraindicated":
  "Has Acute Liver Failure"

// ============================================================
// POLYPHARMACY DETECTION
// ============================================================

/*
 * Count of active medications
 */
define "Active Medication Count":
  Count([MedicationRequest] MR where Medications.IsMedicationActive(MR))

define "Has Polypharmacy":
  "Active Medication Count" >= 5

define "Has Extreme Polypharmacy":
  "Active Medication Count" >= 10

define "Polypharmacy Risk Level":
  case
    when "Active Medication Count" >= 15 then 'CRITICAL'
    when "Active Medication Count" >= 10 then 'HIGH'
    when "Active Medication Count" >= 5 then 'MODERATE'
    else 'LOW'
  end

// ============================================================
// FALL RISK ASSESSMENT
// ============================================================

/*
 * Fall Risk Indicators
 * Used by KB-4 Patient Safety for fall prevention protocols
 */
define "Fall Risk Medications":
  exists "Active Sedatives" or
  exists "Active Opioids" or
  exists ([MedicationRequest: "Anticoagulants"] MR where Medications.IsMedicationActive(MR))

define "Fall Risk Score Component - Medications":
  (if exists "Active Sedatives" then 2 else 0) +
  (if exists "Active Opioids" then 2 else 0) +
  (if "Has Polypharmacy" then 1 else 0)

// ============================================================
// INTERACTION RISK FLAGS (For KB-5 Drug Interactions)
// ============================================================

/*
 * High-risk combination indicators
 * These flags enable KB-5 to prioritize interaction checking
 */
define "Has Anticoagulant NSAID Risk":
  exists "Active Anticoagulants" and "Has NSAID Allergy" = false

define "Has Opioid Sedative Risk":
  exists "Active Opioids" and exists "Active Sedatives"

define "Has Multiple CNS Depressants":
  (if exists "Active Opioids" then 1 else 0) +
  (if exists "Active Sedatives" then 1 else 0) +
  (if exists "Active Neuromuscular Blockers" then 1 else 0) >= 2

// ============================================================
// SUMMARY OUTPUTS FOR ICU INTELLIGENCE
// ============================================================

/*
 * SafetyFactsOutput - structured output for Go consumption
 * This is what the KnowledgeSnapshotBuilder will capture
 */
define "ICU Safety Facts":
  {
    hasActiveHighAlertMedication: "Has Active High Alert Medication",
    onMultipleHighAlertCategories: "On Multiple High Alert Categories",
    hasAnaphylaxisHistory: "History of Anaphylaxis",
    hasActiveSevereBleed: "Has Active Severe Bleeding",
    hasAKI: "Has Acute Kidney Injury",
    hasALF: "Has Acute Liver Failure",
    safetyRiskLevel: "Safety Risk Level",
    safetyRiskScore: "Safety Risk Score",
    polypharmacyRiskLevel: "Polypharmacy Risk Level",
    activeMedicationCount: "Active Medication Count",
    hasPenicillinAllergy: "Has Penicillin Allergy",
    hasSulfaAllergy: "Has Sulfa Allergy",
    hasNSAIDAllergy: "Has NSAID Allergy",
    hasContrastAllergy: "Has Contrast Allergy"
  }
