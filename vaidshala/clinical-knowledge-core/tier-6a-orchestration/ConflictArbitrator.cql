/*
 * ConflictArbitrator CQL Library
 * ============================================================
 * Tier 6a Orchestration - Multi-Protocol Conflict Resolution
 *
 * PURPOSE: CONFLICT ARBITRATION FOR KB-19 PROTOCOL ORCHESTRATOR
 * This library detects conflicts between multiple clinical protocols
 * and provides arbitration recommendations with conservative/aggressive options.
 *
 * ARCHITECTURE CRITICAL (CTO/CMO DIRECTIVE):
 * - CQL OUTPUT: "VTE vs AKI conflict - conservative: hold anticoagulation"
 * - KB-19 CONSUMES: Arbitration recommendation for protocol execution
 * - ICU DOMINANCE: ICU conflicts are NOT resolved here - ICU ALWAYS wins
 * - This library provides RECOMMENDATIONS, not DECISIONS
 *
 * Arbitration Principles:
 * 1. Safety First: When in doubt, choose the safer option
 * 2. Evidence Priority: Higher evidence grade wins
 * 3. Immediacy: Acute life threats take precedence
 * 4. Reversibility: Prefer reversible actions
 *
 * Governance: Tier-6a review (Clinical + Ethics + Legal)
 * ============================================================
 */
library ConflictArbitrator version '1.0.0'

using FHIR version '4.0.1'

include FHIRHelpers version '4.4.000' called FHIRHelpers
include SafetyCommon version '1.0.0' called Safety
include ClinicalCalculators version '1.0.0' called Calculators
include CDSSOrchestrator version '1.0.0' called CDSS

// ============================================================================
// CONTEXT
// ============================================================================
context Patient

// ============================================================================
// CONFLICT TYPE CODES
// ============================================================================

codesystem "ConflictType": 'http://cardiofit.io/fhir/CodeSystem/conflict-type'
codesystem "ArbitrationRule": 'http://cardiofit.io/fhir/CodeSystem/arbitration-rule'

code "Conflict Medication": 'MEDICATION' from "ConflictType"
code "Conflict Protocol": 'PROTOCOL' from "ConflictType"
code "Conflict Organ": 'ORGAN' from "ConflictType"
code "Conflict Resource": 'RESOURCE' from "ConflictType"

code "Rule Safety First": 'SAFETY_FIRST' from "ArbitrationRule"
code "Rule Evidence Priority": 'EVIDENCE_PRIORITY' from "ArbitrationRule"
code "Rule Immediacy": 'IMMEDIACY' from "ArbitrationRule"
code "Rule Reversibility": 'REVERSIBILITY' from "ArbitrationRule"

// ============================================================================
// SECTION 1: ANTICOAGULATION CONFLICTS
// ============================================================================

/*
 * Anticoagulation vs Bleeding Conflict
 * Common scenario: VTE prophylaxis needed but bleeding risk present.
 */
define "Anticoagulation Bleeding Conflict":
  CDSS."VTE Prophylaxis Protocol Applicable"
    and (Safety."Has Active Bleeding" or "Has High Bleeding Risk")

define "Has High Bleeding Risk":
  Calculators."Platelet Count Value" < 50
  or "Has Recent Major Surgery"
  or "Has Intracranial Pathology"
  or "INR Value" > 2.0

define "Has Recent Major Surgery":
  exists(
    [Procedure] P
      where P.status = 'completed'
        and P.performed.end >= Today() - 48 hours
        and exists(P.category C where C.coding.code contains 'major')
  )

define "Has Intracranial Pathology":
  exists(
    [Condition] C
      where C.clinicalStatus.coding.code contains 'active'
        and (
          C.code.coding.code contains 'I60'   // Subarachnoid hemorrhage
          or C.code.coding.code contains 'I61' // Intracerebral hemorrhage
          or C.code.coding.code contains 'I62' // Other nontraumatic intracranial hemorrhage
        )
  )

define "INR Value":
  Last(
    [Observation] O
      where O.status = 'final'
        and O.code.coding.code contains '5902-2'
      sort by effective.value
  ).value.value

define "Anticoagulation Bleeding Arbitration":
  if "Anticoagulation Bleeding Conflict" then
    {
      conflictId: 'ANTICOAG_BLEEDING',
      conflictType: 'MEDICATION',
      severity: if Safety."Has Active Bleeding" then 'CRITICAL' else 'HIGH',

      // Rule applied
      arbitrationRuleApplied: 'SAFETY_FIRST',
      ruleRationale: 'Active bleeding or high bleeding risk takes precedence over VTE prevention',

      // Conservative recommendation (default in bleeding)
      conservativeRecommendation: {
        action: 'HOLD_PHARMACOLOGIC_PROPHYLAXIS',
        alternatives: { 'Sequential compression devices', 'Early mobilization' },
        duration: 'Until bleeding resolved and hemostasis confirmed',
        monitoring: { 'CBC q6h', 'PT/INR q12h if on warfarin' }
      },

      // Aggressive recommendation (if VTE risk very high)
      aggressiveRecommendation: {
        action: 'REDUCED_DOSE_ANTICOAGULATION',
        dosing: 'Enoxaparin 30mg daily (vs 40mg BID)',
        indications: 'Only if active VTE or very high Padua score (>8)',
        riskAcknowledgment: true
      },

      // Arbiter output (recommended action)
      arbiterOutput: if Safety."Has Active Bleeding" then 'CONSERVATIVE' else 'INDIVIDUALIZED',
      requiresPhysicianDecision: true
    }
  else null

// ============================================================================
// SECTION 2: RENAL PROTECTION CONFLICTS
// ============================================================================

/*
 * Nephrotoxic Medication vs Infection Control Conflict
 * Common scenario: AKI but needs aminoglycosides for infection.
 */
define "Nephrotoxic Infection Conflict":
  CDSS."AKI Protocol Applicable"
    and "Ordering Nephrotoxic Antimicrobial"
    and "Has Serious Infection"

define "Ordering Nephrotoxic Antimicrobial":
  exists(
    [MedicationRequest] MR
      where MR.status in { 'active', 'draft' }
        and (
          MR.medication.coding.code contains 'aminoglycoside'
          or MR.medication.coding.code contains 'vancomycin'
          or MR.medication.coding.code contains 'amphotericin'
        )
  )

define "Has Serious Infection":
  CDSS."Sepsis Protocol Applicable"
  or exists(
    [Condition] C
      where C.clinicalStatus.coding.code contains 'active'
        and C.severity.coding.code contains 'severe'
        and C.code.coding.code contains 'A'  // Infectious disease
  )

define "Nephrotoxic Infection Arbitration":
  if "Nephrotoxic Infection Conflict" then
    {
      conflictId: 'NEPHROTOXIC_INFECTION',
      conflictType: 'ORGAN',
      severity: 'HIGH',

      // Rule applied
      arbitrationRuleApplied: 'IMMEDIACY',
      ruleRationale: 'Life-threatening infection takes precedence; protect kidneys where possible',

      // Conservative recommendation
      conservativeRecommendation: {
        action: 'USE_ALTERNATIVE_ANTIMICROBIAL',
        alternatives: "Alternative Antimicrobials for Renal Impairment",
        monitoring: { 'Creatinine q12h', 'Drug levels if aminoglycoside used' }
      },

      // Aggressive recommendation (if no alternatives)
      aggressiveRecommendation: {
        action: 'USE_NEPHROTOXIC_WITH_PROTECTION',
        dosing: 'Extended interval aminoglycoside dosing',
        renalProtection: { 'IV hydration', 'Avoid other nephrotoxins', 'Consider NAC' },
        monitoring: { 'Trough levels', 'Daily creatinine', 'Urine output q4h' }
      },

      arbiterOutput: if "Has Alternative Antimicrobial Available" then 'CONSERVATIVE' else 'AGGRESSIVE',
      requiresPhysicianDecision: true,
      suggestConsult: 'NEPHROLOGY'
    }
  else null

define "Alternative Antimicrobials for Renal Impairment":
  // Alternatives that are less nephrotoxic
  { 'Ceftriaxone', 'Piperacillin-tazobactam (renally adjusted)', 'Fluoroquinolones' }

define "Has Alternative Antimicrobial Available":
  // Check if susceptibility allows alternatives
  // Placeholder - would need microbiology data
  true

// ============================================================================
// SECTION 3: FLUID MANAGEMENT CONFLICTS
// ============================================================================

/*
 * Heart Failure vs AKI Fluid Conflict
 * Common scenario: Need fluids for AKI but patient has HF.
 */
define "HF AKI Fluid Conflict":
  CDSS."HF GDMT Protocol Applicable"
    and CDSS."AKI Protocol Applicable"

define "HF AKI Fluid Arbitration":
  if "HF AKI Fluid Conflict" then
    {
      conflictId: 'HF_AKI_FLUID',
      conflictType: 'ORGAN',
      severity: 'MODERATE',

      // Rule applied
      arbitrationRuleApplied: 'REVERSIBILITY',
      ruleRationale: 'Both dehydration and fluid overload are reversible; target euvolemia',

      // Conservative recommendation (volume restrict)
      conservativeRecommendation: {
        action: 'JUDICIOUS_FLUID_MANAGEMENT',
        strategy: 'Small fluid boluses (250mL) with reassessment',
        targets: {
          JVP: 'normal',
          urinOutput: '> 0.5 mL/kg/hr',
          lungSounds: 'clear'
        },
        avoidance: { 'Aggressive IV fluids', 'High sodium loads' }
      },

      // Aggressive recommendation (prioritize perfusion)
      aggressiveRecommendation: {
        action: 'PERFUSION_PRIORITY',
        strategy: 'Fluids despite mild congestion if MAP < 65 or creatinine rising',
        monitoring: { 'CVP if available', 'Pulmonary congestion assessment q4h' },
        prepareFor: 'Possible diuretics after renal perfusion restored'
      },

      arbiterOutput: "HF AKI Fluid Strategy",
      requiresPhysicianDecision: true,
      suggestConsult: if Calculators."eGFR Value" < 30 then 'NEPHROLOGY_AND_CARDIOLOGY' else 'CARDIOLOGY'
    }
  else null

define "HF AKI Fluid Strategy":
  case
    when Safety."Has Hemodynamic Instability" then 'AGGRESSIVE'  // Perfusion first
    when "Has Pulmonary Edema" then 'CONSERVATIVE'  // Avoid more fluid
    else 'INDIVIDUALIZED'  // Careful titration
  end

define "Has Pulmonary Edema":
  exists(
    [Condition] C
      where C.clinicalStatus.coding.code contains 'active'
        and C.code.coding.code contains 'J81'  // Pulmonary edema
  )
  or exists(
    [Observation] O
      where O.code.coding.code contains 'lung-sounds'
        and O.value.coding.code contains 'crackles'
        and O.effective.value >= Today() - 24 hours
  )

// ============================================================================
// SECTION 4: GLYCEMIC MANAGEMENT CONFLICTS
// ============================================================================

/*
 * Diabetes Treatment vs AKI Conflict
 * Common scenario: Metformin/SGLT2i contraindicated in AKI.
 */
define "Diabetes AKI Medication Conflict":
  CDSS."Diabetes Protocol Applicable"
    and CDSS."AKI Protocol Applicable"
    and "On Metformin or SGLT2i"

define "On Metformin or SGLT2i":
  exists(
    [MedicationRequest] MR
      where MR.status = 'active'
        and (
          MR.medication.coding.code contains 'metformin'
          or MR.medication.coding.code contains 'sglt2'
          or MR.medication.coding.code contains 'empagliflozin'
          or MR.medication.coding.code contains 'dapagliflozin'
          or MR.medication.coding.code contains 'canagliflozin'
        )
  )

define "Diabetes AKI Arbitration":
  if "Diabetes AKI Medication Conflict" then
    {
      conflictId: 'DIABETES_AKI',
      conflictType: 'MEDICATION',
      severity: 'HIGH',

      // Rule applied
      arbitrationRuleApplied: 'SAFETY_FIRST',
      ruleRationale: 'Lactic acidosis risk (metformin) and dehydration risk (SGLT2i) in AKI',

      // Conservative recommendation (always appropriate in AKI)
      conservativeRecommendation: {
        action: 'HOLD_CONTRAINDICATED_AGENTS',
        agentsToHold: { 'Metformin', 'SGLT2 inhibitors' },
        alternatives: { 'Insulin sliding scale', 'DPP-4 inhibitors (renal-adjusted)' },
        resumption: 'When creatinine returns to baseline and stable x48h'
      },

      // Aggressive recommendation (not recommended in AKI)
      aggressiveRecommendation: {
        action: 'NOT_RECOMMENDED',
        rationale: 'No benefit to continuing these agents during acute kidney injury'
      },

      arbiterOutput: 'CONSERVATIVE',  // Always conservative in this conflict
      requiresPhysicianDecision: false,  // Clear guidance
      suggestConsult: if Calculators."HbA1c Value" > 10.0 then 'ENDOCRINOLOGY' else null
    }
  else null

// ============================================================================
// SECTION 5: SEDATION VS RESPIRATORY CONFLICT
// ============================================================================

/*
 * Pain/Sedation vs Respiratory Depression Conflict
 * Common scenario: Patient needs pain control but has respiratory compromise.
 */
define "Sedation Respiratory Conflict":
  "Needs Opioid Analgesia"
    and (Safety."Has Respiratory Compromise" or "Has Opioid Sensitivity")

define "Needs Opioid Analgesia":
  exists(
    [MedicationRequest] MR
      where MR.status in { 'active', 'draft' }
        and MR.medication.coding.code contains 'opioid'
  )

define "Has Opioid Sensitivity":
  exists(
    [AllergyIntolerance] AI
      where AI.category contains 'medication'
        and AI.code.coding.code contains 'opioid'
        and AI.type = 'intolerance'
  )
  or AgeInYears() >= 65
  or Calculators."eGFR Value" < 30

define "Sedation Respiratory Arbitration":
  if "Sedation Respiratory Conflict" then
    {
      conflictId: 'SEDATION_RESPIRATORY',
      conflictType: 'MEDICATION',
      severity: if Safety."Has Respiratory Compromise" then 'HIGH' else 'MODERATE',

      // Rule applied
      arbitrationRuleApplied: 'SAFETY_FIRST',
      ruleRationale: 'Respiratory depression risk; use multimodal analgesia',

      // Conservative recommendation
      conservativeRecommendation: {
        action: 'MULTIMODAL_ANALGESIA',
        strategy: 'Non-opioid first (acetaminophen, NSAIDs if renal function ok, regional blocks)',
        opioidApproach: 'Low-dose, short-acting opioids if needed',
        monitoring: { 'Continuous pulse oximetry', 'Capnography if high risk' }
      },

      // Aggressive recommendation (if pain severe)
      aggressiveRecommendation: {
        action: 'OPIOIDS_WITH_REVERSAL_READY',
        dosing: 'Reduce initial dose by 50%',
        monitoring: { 'Naloxone at bedside', 'Nursing checks q1h', 'SpO2 continuous' },
        contraindicated: Safety."Has Respiratory Compromise" and "On High FiO2"
      },

      arbiterOutput: if Safety."Has Respiratory Compromise" then 'CONSERVATIVE' else 'INDIVIDUALIZED',
      requiresPhysicianDecision: true
    }
  else null

define "On High FiO2":
  exists(
    [DeviceUseStatement] D
      where D.status = 'active'
        and D.device.display ~ 'oxygen'
  )

// ============================================================================
// SECTION 6: CONFLICTING RECOMMENDATIONS AGGREGATION
// ============================================================================

/*
 * Conflicting Recommendations
 * All detected conflicts requiring arbitration.
 */
define "Conflicting Recommendations":
  {
    anticoagBleedingConflict: "Anticoagulation Bleeding Arbitration",
    nephrotoxicInfectionConflict: "Nephrotoxic Infection Arbitration",
    hfAkiFluidConflict: "HF AKI Fluid Arbitration",
    diabetesAkiConflict: "Diabetes AKI Arbitration",
    sedationRespiratoryConflict: "Sedation Respiratory Arbitration"
  }

define "Has Conflicting Recommendations":
  "Anticoagulation Bleeding Conflict"
  or "Nephrotoxic Infection Conflict"
  or "HF AKI Fluid Conflict"
  or "Diabetes AKI Medication Conflict"
  or "Sedation Respiratory Conflict"

define "Conflict Count":
  Count({
    if "Anticoagulation Bleeding Conflict" then 1 else null,
    if "Nephrotoxic Infection Conflict" then 1 else null,
    if "HF AKI Fluid Conflict" then 1 else null,
    if "Diabetes AKI Medication Conflict" then 1 else null,
    if "Sedation Respiratory Conflict" then 1 else null
  })

// ============================================================================
// SECTION 7: ARBITER OUTPUT
// ============================================================================

/*
 * Arbiter Output
 * Final recommendation with rationale for KB-19 consumption.
 *
 * NOTE: ICU conflicts are handled by ICU Intelligence, not this arbiter.
 * This arbiter handles protocol-level conflicts only.
 */
define "Arbiter Output":
  {
    // Summary
    hasConflicts: "Has Conflicting Recommendations",
    totalConflicts: "Conflict Count",

    // Individual arbitrations
    arbitrations: "Conflicting Recommendations",

    // Highest severity conflict
    highestSeverity: "Highest Conflict Severity",

    // Overall recommendation strategy
    overallStrategy: case
      when "Highest Conflict Severity" = 'CRITICAL' then 'CONSERVATIVE_PENDING_REVIEW'
      when "Highest Conflict Severity" = 'HIGH' then 'PHYSICIAN_DECISION_REQUIRED'
      else 'GUIDELINE_DIRECTED'
    end,

    // Consult recommendations
    suggestedConsults: "Aggregated Consult Recommendations",

    // ICU deferral
    deferToICU: Safety."Safety Risk Level" in { 'HIGH', 'CRITICAL' },
    icuHasVetoAuthority: true,  // Always true per CTO/CMO directive

    // Audit trail
    arbitrationRulesApplied: "Applied Arbitration Rules"
  }

define "Highest Conflict Severity":
  case
    when "Anticoagulation Bleeding Conflict" and Safety."Has Active Bleeding" then 'CRITICAL'
    when "Nephrotoxic Infection Conflict" then 'HIGH'
    when "Anticoagulation Bleeding Conflict" then 'HIGH'
    when "Diabetes AKI Medication Conflict" then 'HIGH'
    when "Sedation Respiratory Conflict" and Safety."Has Respiratory Compromise" then 'HIGH'
    when "HF AKI Fluid Conflict" then 'MODERATE'
    when "Sedation Respiratory Conflict" then 'MODERATE'
    else null
  end

define "Aggregated Consult Recommendations":
  {
    nephrology: "Nephrotoxic Infection Conflict" or ("HF AKI Fluid Conflict" and Calculators."eGFR Value" < 30),
    cardiology: "HF AKI Fluid Conflict",
    hematology: "Anticoagulation Bleeding Conflict",
    endocrinology: "Diabetes AKI Medication Conflict" and Calculators."HbA1c Value" > 10.0,
    painService: "Sedation Respiratory Conflict"
  }

define "Applied Arbitration Rules":
  {
    safetyFirst: "Anticoagulation Bleeding Conflict" or "Diabetes AKI Medication Conflict" or "Sedation Respiratory Conflict",
    immediacy: "Nephrotoxic Infection Conflict",
    reversibility: "HF AKI Fluid Conflict"
  }
