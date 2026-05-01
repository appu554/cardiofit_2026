-- 002_au_indicators_seed.sql
--
-- Tier 3 / Layer 1 / Wave 6 — Australian quality indicator seed.
-- Two sources:
--
--   (1) PHARMA-Care National Quality Framework (UniSA, $1.5M MRFF, in active
--       national pilot 2025-2026 evaluating $350M ACOP program). Five-domain
--       framework. Indicators are STILL IN PILOT — definitions seeded here
--       are placeholders pending public framework publication. See README
--       for procurement path (EOI: ALH-PHARMA-Care@unisa.edu.au).
--
--   (2) National Aged Care Mandatory Quality Indicator Program (QI Program) —
--       11 indicators that every Australian RACF must report quarterly to
--       the Department of Health, Disability and Ageing. Mandatory since
--       1 April 2023 with expansion 1 April 2024 to include the medication-
--       safety indicators. These ARE publicly defined and seeded with full
--       structural metadata; numerator/denominator definitions in
--       definition_yaml are abridged and should be enriched against the
--       official ACSQHC QI Program manual when available locally.
--
-- Reference: claudedocs/audits/Layer1_AU_AgedCare_Codebase_Gap_Audit_v2_2026-04-30.md
--            §"v2 sources to add to Layer 1 plan"

-- ============================================================
-- (1) PHARMA-Care 5-domain placeholder seeds
-- ============================================================
INSERT INTO measures (id, version, name, title, type, scoring, domain, program,
                      improvement_notation, active, definition_yaml) VALUES

('PHARMA-CARE-D1', '0.1-pilot', 'pharma_care_quality_of_use',
 'PHARMA-Care Domain 1: Quality of Medication Use',
 'PROCESS', 'proportion', 'MEDICATION_USE', 'PHARMA_CARE',
 'increase', false,
 '{"status": "PILOT_PLACEHOLDER", "framework": "PHARMA-Care National Quality Framework",
   "publisher": "UniSA / Sluggett (lead) + 14 partners + PSA-endorsed",
   "indicator_definition_pending": true,
   "procurement": "EOI: ALH-PHARMA-Care@unisa.edu.au"}'::jsonb),

('PHARMA-CARE-D2', '0.1-pilot', 'pharma_care_safety',
 'PHARMA-Care Domain 2: Medication Safety',
 'OUTCOME', 'proportion', 'PATIENT_SAFETY', 'PHARMA_CARE',
 'decrease', false,
 '{"status": "PILOT_PLACEHOLDER", "indicator_definition_pending": true}'::jsonb),

('PHARMA-CARE-D3', '0.1-pilot', 'pharma_care_continuity',
 'PHARMA-Care Domain 3: Continuity of Care across Transitions',
 'PROCESS', 'proportion', 'CARE_COORDINATION', 'PHARMA_CARE',
 'increase', false,
 '{"status": "PILOT_PLACEHOLDER", "indicator_definition_pending": true}'::jsonb),

('PHARMA-CARE-D4', '0.1-pilot', 'pharma_care_person_centred',
 'PHARMA-Care Domain 4: Person-Centred Medication Care',
 'PROCESS', 'proportion', 'PERSON_CENTRED_CARE', 'PHARMA_CARE',
 'increase', false,
 '{"status": "PILOT_PLACEHOLDER", "indicator_definition_pending": true}'::jsonb),

('PHARMA-CARE-D5', '0.1-pilot', 'pharma_care_system_integration',
 'PHARMA-Care Domain 5: System Integration & Workflow',
 'STRUCTURE', 'proportion', 'SYSTEM_INTEGRATION', 'PHARMA_CARE',
 'increase', false,
 '{"status": "PILOT_PLACEHOLDER", "indicator_definition_pending": true}'::jsonb)

ON CONFLICT (id) DO UPDATE SET
    version = EXCLUDED.version,
    title = EXCLUDED.title,
    definition_yaml = EXCLUDED.definition_yaml,
    updated_at = NOW();

-- ============================================================
-- (2) National Aged Care Mandatory Quality Indicator Program (QI Program)
--     11 indicators. Publicly defined by ACSQHC + DoHDA. Mandatory quarterly
--     reporting for every Australian RACF.
-- ============================================================
INSERT INTO measures (id, version, name, title, type, scoring, domain, program,
                      improvement_notation, active, definition_yaml) VALUES

('AU-QI-01-PRESSURE-INJURY', '2024.04', 'qi_pressure_injury',
 'QI Program: Pressure Injuries',
 'OUTCOME', 'proportion', 'PATIENT_SAFETY', 'AU_QI_PROGRAM',
 'decrease', true,
 '{"category": "Stage 1-4 + unstageable + suspected deep tissue injury",
   "numerator": "Care recipients with one or more pressure injuries during the quarter",
   "denominator": "All care recipients in the service during the quarter",
   "stratifications": ["stage", "presentation_status (present_on_admission|developed_in_service)"],
   "regulator": "ACSQHC + DoHDA",
   "mandatory_since": "2019-07-01",
   "reporting_cadence": "quarterly"}'::jsonb),

('AU-QI-02-PHYSICAL-RESTRAINT', '2024.04', 'qi_physical_restraint',
 'QI Program: Physical Restraint',
 'OUTCOME', 'proportion', 'PATIENT_SAFETY', 'AU_QI_PROGRAM',
 'decrease', true,
 '{"category": "Use of physical restraint",
   "numerator": "Care recipients subject to one or more episodes of physical restraint during the quarter",
   "denominator": "All care recipients in the service during the quarter",
   "exclusions": ["consensual mechanical aids that do not restrict movement"],
   "regulator": "ACSQHC + DoHDA",
   "mandatory_since": "2019-07-01"}'::jsonb),

('AU-QI-03-UNPLANNED-WEIGHT-LOSS', '2024.04', 'qi_unplanned_weight_loss',
 'QI Program: Unplanned Weight Loss',
 'OUTCOME', 'proportion', 'NUTRITION', 'AU_QI_PROGRAM',
 'decrease', true,
 '{"thresholds": {"significant": ">=5% in 3 months OR >=10% in 6 months", "consecutive": "any unplanned loss across 3 consecutive months"},
   "regulator": "ACSQHC + DoHDA"}'::jsonb),

('AU-QI-04-FALLS-MAJOR-INJURY', '2024.04', 'qi_falls_major_injury',
 'QI Program: Falls and Major Injury',
 'OUTCOME', 'proportion', 'PATIENT_SAFETY', 'AU_QI_PROGRAM',
 'decrease', true,
 '{"sub_indicators": ["any_fall", "fall_with_major_injury"],
   "major_injury_definition": "fracture, head injury requiring imaging, or any fall requiring hospital transfer",
   "regulator": "ACSQHC + DoHDA"}'::jsonb),

('AU-QI-05-MEDICATION-POLYPHARMACY', '2024.04', 'qi_polypharmacy',
 'QI Program: Medication Management — Polypharmacy',
 'PROCESS', 'proportion', 'MEDICATION_SAFETY', 'AU_QI_PROGRAM',
 'decrease', true,
 '{"definition": "Care recipients prescribed 9 or more regular medications",
   "expansion_date": "2024-04-01",
   "regulator": "ACSQHC + DoHDA",
   "kb_dependencies": ["KB-4 PIM rules", "KB-1 active medication list", "KB-6 PBS items"]}'::jsonb),

('AU-QI-06-MEDICATION-ANTIPSYCHOTIC', '2024.04', 'qi_antipsychotic_use',
 'QI Program: Medication Management — Antipsychotic Use',
 'OUTCOME', 'proportion', 'MEDICATION_SAFETY', 'AU_QI_PROGRAM',
 'decrease', true,
 '{"definition": "Care recipients prescribed an antipsychotic during the quarter",
   "exclusions": ["diagnosis_of_schizophrenia", "diagnosis_of_bipolar", "psychotic_disorder_documented"],
   "expansion_date": "2024-04-01",
   "rationale": "Royal Commission into Aged Care found systemic over-prescribing of antipsychotics for BPSD",
   "regulator": "ACSQHC + DoHDA",
   "kb_dependencies": ["KB-4 antipsychotic CONTRAINDICATION rules", "KB-20 ADR profiles for psychotropics"]}'::jsonb),

('AU-QI-07-ACTIVITIES-DAILY-LIVING', '2024.04', 'qi_adl',
 'QI Program: Activities of Daily Living',
 'OUTCOME', 'continuous', 'FUNCTIONAL_STATUS', 'AU_QI_PROGRAM',
 'increase', true,
 '{"instrument": "Modified Barthel Index or equivalent",
   "expansion_date": "2024-04-01",
   "regulator": "ACSQHC + DoHDA"}'::jsonb),

('AU-QI-08-INCONTINENCE', '2024.04', 'qi_incontinence',
 'QI Program: Incontinence Care',
 'OUTCOME', 'proportion', 'CONTINENCE', 'AU_QI_PROGRAM',
 'decrease', true,
 '{"definition": "Care recipients experiencing incontinence-associated dermatitis during the quarter",
   "expansion_date": "2024-04-01",
   "regulator": "ACSQHC + DoHDA"}'::jsonb),

('AU-QI-09-HOSPITALISATION', '2024.04', 'qi_hospitalisation',
 'QI Program: Hospitalisation',
 'OUTCOME', 'proportion', 'CARE_COORDINATION', 'AU_QI_PROGRAM',
 'decrease', true,
 '{"sub_indicators": ["any_hospitalisation", "ed_presentation", "potentially_preventable_hospitalisation"],
   "expansion_date": "2024-04-01",
   "regulator": "ACSQHC + DoHDA"}'::jsonb),

('AU-QI-10-WORKFORCE', '2024.04', 'qi_workforce',
 'QI Program: Workforce',
 'STRUCTURE', 'continuous', 'WORKFORCE', 'AU_QI_PROGRAM',
 'increase', true,
 '{"sub_indicators": ["RN_minutes_per_resident_per_day", "total_care_minutes_per_resident_per_day"],
   "minimum_targets": {"RN_minutes": 40, "total_care_minutes": 215},
   "expansion_date": "2024-04-01",
   "regulator": "ACSQHC + DoHDA + Aged Care Quality and Safety Commission"}'::jsonb),

('AU-QI-11-CONSUMER-EXPERIENCE', '2024.04', 'qi_consumer_experience',
 'QI Program: Consumer Experience',
 'OUTCOME', 'proportion', 'PERSON_CENTRED_CARE', 'AU_QI_PROGRAM',
 'increase', true,
 '{"instrument": "Consumer Experience Reports interview",
   "expansion_date": "2024-04-01",
   "regulator": "ACSQHC + DoHDA"}'::jsonb)

ON CONFLICT (id) DO UPDATE SET
    version = EXCLUDED.version,
    title = EXCLUDED.title,
    definition_yaml = EXCLUDED.definition_yaml,
    updated_at = NOW();
