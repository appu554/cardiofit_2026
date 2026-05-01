library ADA2026Diabetes version '1.0.0'

// Authority: ADA Standards of Care 2026
// Effective: 2026-01-01
//
// Defines that bind ADA-2026 thresholds to KB-1 / KB-4 / KB-16 facts.
// Each define has a corresponding registry entry that the
// CQLCompatibilityChecker uses to verify L3-extracted facts agree
// with the CQL author's expected thresholds.

using FHIR version '4.0.1'

context Patient

// ──────────────────────────────────────────────────────────────────────────
// 1. SGLT2i Initiation Eligible (KB-1 / Recommendation 11.7a)
// ──────────────────────────────────────────────────────────────────────────
//
// Source: "SGLT2 inhibitors should be initiated in individuals with
// eGFR ≥20 mL/min/1.73 m2 but can safely continue until kidney failure."
// → contraindication threshold: do not initiate when eGFR < 20.
//
define "SGLT2i Initiation Eligible":
    [Observation: code in 'eGFR (calculated)'] eGFR
        where eGFR.value >= 20 'mL/min/1.73m2'

// ──────────────────────────────────────────────────────────────────────────
// 2. SGLT2i DKA Withhold (KB-16 / Recommendation 10.46)
// ──────────────────────────────────────────────────────────────────────────
//
// Source: ADA flags critical β-hydroxybutyrate elevation as a DKA emergency
// requiring drug withholding.
//
define "SGLT2i DKA Withhold":
    [Observation: code in 'Beta-hydroxybutyrate'] bhb
        where bhb.value >= 3.0 'mmol/L'

// ──────────────────────────────────────────────────────────────────────────
// 3. ACEi/ARB Hyperkalemia Withhold (KB-16 / Section 11)
// ──────────────────────────────────────────────────────────────────────────
//
// Source: serum potassium > 5.5 mmol/L during ACEi or ARB therapy
// triggers reassessment per KDIGO 2024-aligned ADA guidance.
//
define "ACEi or ARB Hyperkalemia Withhold":
    [Observation: code in 'Serum potassium'] k
        where k.value > 5.5 'mmol/L'

// ──────────────────────────────────────────────────────────────────────────
// 4. Perindopril Renal Eligibility (KB-4 / Section 11.6a)
// ──────────────────────────────────────────────────────────────────────────
//
// Source: ADA recommends ACEi initiation when albuminuria + eGFR < 60.
//
define "Perindopril Renal Eligibility":
    [Observation: code in 'eGFR (calculated)'] eGFR
        where eGFR.value < 60 'mL/min/1.73m2'

// ──────────────────────────────────────────────────────────────────────────
// 5. Exenatide Renal Contraindication (KB-4 / Section 9.11)
// ──────────────────────────────────────────────────────────────────────────
//
// Source: exenatide should not be used in dialysis / eGFR < 30.
//
define "Exenatide Renal Contraindication":
    [Observation: code in 'eGFR (calculated)'] eGFR
        where eGFR.value < 30 'mL/min/1.73m2'
