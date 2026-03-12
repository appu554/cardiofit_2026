-- ============================================================================
-- KB-6 Formulary Service - Step Therapy Rules Seed Data
-- ============================================================================
-- Step therapy requirements based on clinical guidelines and formulary design
-- ============================================================================

-- =============================================================================
-- GLP-1 AGONIST STEP THERAPY (Diabetes)
-- =============================================================================

-- Semaglutide (Ozempic) - Requires Metformin + SGLT2i before approval
INSERT INTO step_therapy_rules (
    target_drug_rxnorm, target_drug_name, payer_id, plan_id,
    steps,
    override_criteria,
    exception_diagnosis_codes,
    protocol_name, protocol_version, evidence_level,
    effective_date, version
) VALUES
(
    '1991302', 'Semaglutide (Ozempic)', NULL, NULL,
    '[
        {
            "step_number": 1,
            "drug_class": "Biguanides",
            "description": "Metformin - First-line therapy",
            "rxnorm_codes": ["6809", "860975"],
            "min_duration_days": 90,
            "max_duration_days": null,
            "required_dose": null,
            "allow_any_in_class": true,
            "rationale": "ADA/EASD Guidelines recommend metformin as first-line therapy"
        },
        {
            "step_number": 2,
            "drug_class": "SGLT2 Inhibitors",
            "description": "SGLT2i - Second-line with cardiovascular benefit",
            "rxnorm_codes": ["1545653", "1545658", "1373458", "1373463", "1486436"],
            "min_duration_days": 60,
            "max_duration_days": null,
            "allow_any_in_class": true,
            "rationale": "SGLT2i preferred for patients with heart failure or CKD"
        }
    ]'::jsonb,
    ARRAY['metformin_contraindication', 'renal_impairment_egfr_lt_30', 'lactic_acidosis_history', 'sglt2_contraindication', 'treatment_failure', 'adverse_reaction', 'cardiovascular_disease'],
    ARRAY['I50', 'I50.9', 'N18.4', 'N18.5'],
    'ADA-EASD T2DM Treatment Algorithm', '2024.1', 'Level A',
    '2025-01-01', 1
),

-- Liraglutide (Victoza) - Similar step therapy
(
    '897122', 'Liraglutide (Victoza)', NULL, NULL,
    '[
        {
            "step_number": 1,
            "drug_class": "Biguanides",
            "description": "Metformin - First-line therapy",
            "rxnorm_codes": ["6809", "860975"],
            "min_duration_days": 90,
            "allow_any_in_class": true
        }
    ]'::jsonb,
    ARRAY['metformin_contraindication', 'renal_impairment', 'treatment_failure', 'adverse_reaction'],
    ARRAY['I50', 'N18'],
    'ADA T2DM Treatment Algorithm', '2024.1', 'Level A',
    '2025-01-01', 1
)
ON CONFLICT (target_drug_rxnorm, payer_id) DO UPDATE SET
    steps = EXCLUDED.steps,
    override_criteria = EXCLUDED.override_criteria,
    updated_at = NOW();

-- =============================================================================
-- SGLT2 INHIBITOR STEP THERAPY (Preferred before Non-Preferred)
-- =============================================================================

-- Jardiance (Empagliflozin) - Preferred, requires Metformin first
INSERT INTO step_therapy_rules (
    target_drug_rxnorm, target_drug_name, payer_id, plan_id,
    steps,
    override_criteria,
    protocol_name, evidence_level,
    effective_date, version
) VALUES
(
    '1545653', 'Empagliflozin (Jardiance) 10mg', NULL, NULL,
    '[
        {
            "step_number": 1,
            "drug_class": "Biguanides",
            "description": "Metformin",
            "rxnorm_codes": ["6809", "860975"],
            "min_duration_days": 90,
            "allow_any_in_class": true,
            "rationale": "First-line diabetes therapy"
        }
    ]'::jsonb,
    ARRAY['metformin_contraindication', 'renal_impairment_egfr_lt_30', 'lactic_acidosis_history', 'adverse_reaction'],
    'Formulary Step Therapy Protocol', 'Formulary-based',
    '2025-01-01', 1
),

-- Invokana (Canagliflozin) - Non-Preferred, requires Metformin + Jardiance first
(
    '1373458', 'Canagliflozin (Invokana) 100mg', NULL, NULL,
    '[
        {
            "step_number": 1,
            "drug_class": "Biguanides",
            "description": "Metformin",
            "rxnorm_codes": ["6809", "860975"],
            "min_duration_days": 90,
            "allow_any_in_class": true
        },
        {
            "step_number": 2,
            "drug_class": "Preferred SGLT2i",
            "description": "Jardiance (preferred SGLT2i)",
            "rxnorm_codes": ["1545653", "1545658"],
            "min_duration_days": 30,
            "allow_any_in_class": false,
            "rationale": "Must try preferred SGLT2i before non-preferred"
        }
    ]'::jsonb,
    ARRAY['jardiance_contraindication', 'jardiance_adverse_reaction', 'jardiance_treatment_failure'],
    'Formulary Step Therapy Protocol', 'Formulary-based',
    '2025-01-01', 1
)
ON CONFLICT (target_drug_rxnorm, payer_id) DO UPDATE SET
    steps = EXCLUDED.steps,
    updated_at = NOW();

-- =============================================================================
-- ANTICOAGULANT STEP THERAPY (DOACs)
-- =============================================================================

-- Rivaroxaban (Xarelto) - Non-Preferred DOAC, requires Eliquis trial first
INSERT INTO step_therapy_rules (
    target_drug_rxnorm, target_drug_name, payer_id, plan_id,
    steps,
    override_criteria,
    exception_diagnosis_codes,
    protocol_name, evidence_level,
    effective_date, version
) VALUES
(
    '1232082', 'Rivaroxaban (Xarelto) 20mg', NULL, NULL,
    '[
        {
            "step_number": 1,
            "drug_class": "Preferred DOAC",
            "description": "Apixaban (Eliquis) - Preferred direct oral anticoagulant",
            "rxnorm_codes": ["1364430", "1364435"],
            "min_duration_days": 30,
            "allow_any_in_class": false,
            "rationale": "Eliquis is preferred DOAC on formulary"
        }
    ]'::jsonb,
    ARRAY['apixaban_contraindication', 'apixaban_adverse_reaction', 'drug_interaction', 'indication_specific', 'renal_dose_adjustment'],
    ARRAY['I48', 'I26', 'I82'],
    'DOAC Formulary Protocol', 'Formulary-based',
    '2025-01-01', 1
),

-- Xarelto 15mg same step therapy
(
    '1232086', 'Rivaroxaban (Xarelto) 15mg', NULL, NULL,
    '[
        {
            "step_number": 1,
            "drug_class": "Preferred DOAC",
            "description": "Apixaban (Eliquis)",
            "rxnorm_codes": ["1364430", "1364435"],
            "min_duration_days": 30,
            "allow_any_in_class": false
        }
    ]'::jsonb,
    ARRAY['apixaban_contraindication', 'apixaban_adverse_reaction', 'drug_interaction'],
    ARRAY['I48', 'I26', 'I82'],
    'DOAC Formulary Protocol', 'Formulary-based',
    '2025-01-01', 1
)
ON CONFLICT (target_drug_rxnorm, payer_id) DO UPDATE SET
    steps = EXCLUDED.steps,
    updated_at = NOW();

-- =============================================================================
-- COPY STEP THERAPY RULES FOR RELATED STRENGTHS
-- =============================================================================

-- Jardiance 25mg uses same ST as 10mg
INSERT INTO step_therapy_rules (target_drug_rxnorm, target_drug_name, payer_id, steps, override_criteria, protocol_name, evidence_level, effective_date, version)
SELECT '1545658', 'Empagliflozin (Jardiance) 25mg', payer_id, steps, override_criteria, protocol_name, evidence_level, effective_date, version
FROM step_therapy_rules WHERE target_drug_rxnorm = '1545653'
ON CONFLICT (target_drug_rxnorm, payer_id) DO NOTHING;

-- Invokana 300mg uses same ST as 100mg
INSERT INTO step_therapy_rules (target_drug_rxnorm, target_drug_name, payer_id, steps, override_criteria, protocol_name, evidence_level, effective_date, version)
SELECT '1373463', 'Canagliflozin (Invokana) 300mg', payer_id, steps, override_criteria, protocol_name, evidence_level, effective_date, version
FROM step_therapy_rules WHERE target_drug_rxnorm = '1373458'
ON CONFLICT (target_drug_rxnorm, payer_id) DO NOTHING;

-- Ozempic higher strengths use same ST
INSERT INTO step_therapy_rules (target_drug_rxnorm, target_drug_name, payer_id, steps, override_criteria, exception_diagnosis_codes, protocol_name, evidence_level, effective_date, version)
SELECT '1991306', 'Semaglutide (Ozempic) 0.5mg/0.5mL', payer_id, steps, override_criteria, exception_diagnosis_codes, protocol_name, evidence_level, effective_date, version
FROM step_therapy_rules WHERE target_drug_rxnorm = '1991302'
ON CONFLICT (target_drug_rxnorm, payer_id) DO NOTHING;

INSERT INTO step_therapy_rules (target_drug_rxnorm, target_drug_name, payer_id, steps, override_criteria, exception_diagnosis_codes, protocol_name, evidence_level, effective_date, version)
SELECT '1991310', 'Semaglutide (Ozempic) 1mg/0.5mL', payer_id, steps, override_criteria, exception_diagnosis_codes, protocol_name, evidence_level, effective_date, version
FROM step_therapy_rules WHERE target_drug_rxnorm = '1991302'
ON CONFLICT (target_drug_rxnorm, payer_id) DO NOTHING;
