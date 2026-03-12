-- ============================================================================
-- KB-6 Formulary Service - Prior Authorization Requirements Seed Data
-- ============================================================================
-- Clinical PA criteria based on evidence-based guidelines
-- ============================================================================

-- =============================================================================
-- GLP-1 AGONIST PA REQUIREMENTS (Specialty Diabetes Medications)
-- =============================================================================

-- Semaglutide (Ozempic) PA Requirements - Universal
INSERT INTO pa_requirements (
    drug_rxnorm, drug_name, payer_id, plan_id,
    criteria,
    approval_duration_days, renewal_allowed, max_renewals,
    required_documents, urgency_levels,
    standard_review_hours, urgent_review_hours, expedited_review_hours,
    effective_date, version
) VALUES
(
    '1991302', 'Semaglutide (Ozempic) 0.25mg/0.5mL', NULL, NULL,
    '[
        {
            "type": "DIAGNOSIS",
            "codes": ["E11", "E11.9", "E11.65", "E11.21", "E11.22"],
            "code_system": "ICD10",
            "description": "Type 2 Diabetes Mellitus",
            "required": true
        },
        {
            "type": "LAB",
            "test": "HbA1c",
            "loinc": "4548-4",
            "operator": ">",
            "value": 7.0,
            "unit": "%",
            "max_age_days": 90,
            "description": "HbA1c above 7.0% within last 90 days"
        },
        {
            "type": "PRIOR_THERAPY",
            "drug_class": "Biguanides",
            "rxnorm_codes": ["6809", "860975"],
            "min_duration_days": 90,
            "description": "Metformin trial for at least 90 days",
            "or_contraindication": true
        },
        {
            "type": "PRIOR_THERAPY",
            "drug_class": "SGLT2 Inhibitor",
            "rxnorm_codes": ["1545653", "1545658", "1373458", "1373463", "1486436"],
            "min_duration_days": 60,
            "description": "SGLT2 inhibitor trial for at least 60 days",
            "or_contraindication": true
        },
        {
            "type": "AGE",
            "operator": ">=",
            "value": 18,
            "description": "Adult patients (18 years or older)"
        }
    ]'::jsonb,
    365, true, 3,
    ARRAY['Prior medication history', 'HbA1c lab result within 90 days', 'Diagnosis documentation', 'Clinical notes'],
    ARRAY['STANDARD', 'URGENT', 'EXPEDITED'],
    72, 24, 4,
    '2025-01-01', 1
),

-- Liraglutide (Victoza) PA Requirements
(
    '897122', 'Liraglutide (Victoza) 18mg/3mL', NULL, NULL,
    '[
        {
            "type": "DIAGNOSIS",
            "codes": ["E11", "E11.9", "E11.65"],
            "code_system": "ICD10",
            "description": "Type 2 Diabetes Mellitus"
        },
        {
            "type": "LAB",
            "test": "HbA1c",
            "loinc": "4548-4",
            "operator": ">",
            "value": 7.0,
            "unit": "%",
            "max_age_days": 90
        },
        {
            "type": "PRIOR_THERAPY",
            "drug_class": "Biguanides",
            "rxnorm_codes": ["6809"],
            "min_duration_days": 90,
            "or_contraindication": true
        },
        {
            "type": "CONTRAINDICATION",
            "conditions": ["personal_history_medullary_thyroid_ca", "MEN2_syndrome"],
            "action": "deny",
            "description": "History of medullary thyroid carcinoma or MEN 2 syndrome"
        }
    ]'::jsonb,
    365, true, 3,
    ARRAY['Prior medication history', 'HbA1c lab result', 'Diagnosis documentation'],
    ARRAY['STANDARD', 'URGENT', 'EXPEDITED'],
    72, 24, 4,
    '2025-01-01', 1
)
ON CONFLICT (drug_rxnorm, payer_id) DO UPDATE SET
    criteria = EXCLUDED.criteria,
    approval_duration_days = EXCLUDED.approval_duration_days,
    required_documents = EXCLUDED.required_documents,
    updated_at = NOW();

-- =============================================================================
-- SGLT2 INHIBITOR PA REQUIREMENTS (Non-Preferred)
-- =============================================================================

-- Canagliflozin (Invokana) PA Requirements
INSERT INTO pa_requirements (
    drug_rxnorm, drug_name, payer_id, plan_id,
    criteria,
    approval_duration_days, renewal_allowed,
    required_documents,
    effective_date, version
) VALUES
(
    '1373458', 'Canagliflozin (Invokana) 100mg', NULL, NULL,
    '[
        {
            "type": "DIAGNOSIS",
            "codes": ["E11"],
            "code_system": "ICD10",
            "description": "Type 2 Diabetes Mellitus"
        },
        {
            "type": "PRIOR_THERAPY",
            "drug_class": "Preferred SGLT2i",
            "rxnorm_codes": ["1545653", "1545658"],
            "min_duration_days": 30,
            "description": "Jardiance trial or documented intolerance/contraindication",
            "or_contraindication": true
        },
        {
            "type": "LAB",
            "test": "eGFR",
            "loinc": "48642-3",
            "operator": ">=",
            "value": 30,
            "unit": "mL/min/1.73m2",
            "description": "eGFR >= 30 for Invokana use"
        }
    ]'::jsonb,
    365, true,
    ARRAY['Prior SGLT2i history or contraindication documentation', 'eGFR lab result', 'Clinical rationale for non-preferred drug'],
    '2025-01-01', 1
)
ON CONFLICT (drug_rxnorm, payer_id) DO UPDATE SET
    criteria = EXCLUDED.criteria,
    updated_at = NOW();

-- =============================================================================
-- CONTROLLED SUBSTANCE PA REQUIREMENTS
-- =============================================================================

-- Oxycodone PA Requirements
INSERT INTO pa_requirements (
    drug_rxnorm, drug_name, payer_id, plan_id,
    criteria,
    approval_duration_days, renewal_allowed, max_renewals,
    required_documents,
    standard_review_hours, urgent_review_hours,
    effective_date, version
) VALUES
(
    '7804', 'Oxycodone', NULL, NULL,
    '[
        {
            "type": "DIAGNOSIS",
            "codes": ["G89", "G89.29", "G89.4", "M54", "C00-C96"],
            "code_system": "ICD10",
            "description": "Chronic pain syndrome, cancer, or documented pain condition"
        },
        {
            "type": "PRIOR_THERAPY",
            "drug_class": "Non-opioid analgesics",
            "description": "Trial of acetaminophen, NSAIDs, or other non-opioid therapy",
            "min_duration_days": 7,
            "or_contraindication": true
        },
        {
            "type": "AGE",
            "operator": ">=",
            "value": 18,
            "description": "Adult patients only"
        },
        {
            "type": "CUSTOM",
            "check": "PDMP_verification",
            "description": "Prescription Drug Monitoring Program check completed"
        },
        {
            "type": "CUSTOM",
            "check": "opioid_agreement",
            "description": "Signed opioid treatment agreement on file"
        }
    ]'::jsonb,
    30, true, 11,
    ARRAY['Pain assessment documentation', 'PDMP check results', 'Opioid treatment agreement', 'Non-opioid therapy history'],
    24, 4,
    '2025-01-01', 1
)
ON CONFLICT (drug_rxnorm, payer_id) DO UPDATE SET
    criteria = EXCLUDED.criteria,
    approval_duration_days = EXCLUDED.approval_duration_days,
    updated_at = NOW();

-- =============================================================================
-- SPECIALTY MEDICATION PA REQUIREMENTS
-- =============================================================================

-- Adalimumab (Humira) PA Requirements - Example Biologic
INSERT INTO pa_requirements (
    drug_rxnorm, drug_name, payer_id, plan_id,
    criteria,
    approval_duration_days, renewal_allowed,
    required_documents,
    effective_date, version
) VALUES
(
    '327361', 'Adalimumab (Humira)', NULL, NULL,
    '[
        {
            "type": "DIAGNOSIS",
            "codes": ["M05", "M06", "L40.5", "K50", "K51"],
            "code_system": "ICD10",
            "description": "Rheumatoid arthritis, psoriatic arthritis, Crohns, or ulcerative colitis"
        },
        {
            "type": "PRIOR_THERAPY",
            "drug_class": "DMARDs",
            "description": "Methotrexate or other conventional DMARD trial",
            "min_duration_days": 90,
            "or_contraindication": true
        },
        {
            "type": "LAB",
            "test": "TB_screening",
            "description": "Negative TB screening within 12 months"
        },
        {
            "type": "LAB",
            "test": "Hepatitis_B",
            "description": "Hepatitis B screening completed"
        }
    ]'::jsonb,
    180, true,
    ARRAY['Disease activity documentation', 'DMARD history', 'TB screening results', 'Hepatitis B screening'],
    '2025-01-01', 1
)
ON CONFLICT (drug_rxnorm, payer_id) DO UPDATE SET
    criteria = EXCLUDED.criteria,
    updated_at = NOW();

-- Additional high-dose Ozempic strengths use same PA requirements
INSERT INTO pa_requirements (drug_rxnorm, drug_name, payer_id, criteria, approval_duration_days, renewal_allowed, required_documents, effective_date, version)
SELECT '1991306', 'Semaglutide (Ozempic) 0.5mg/0.5mL', payer_id, criteria, approval_duration_days, renewal_allowed, required_documents, effective_date, version
FROM pa_requirements WHERE drug_rxnorm = '1991302'
ON CONFLICT (drug_rxnorm, payer_id) DO NOTHING;

INSERT INTO pa_requirements (drug_rxnorm, drug_name, payer_id, criteria, approval_duration_days, renewal_allowed, required_documents, effective_date, version)
SELECT '1991310', 'Semaglutide (Ozempic) 1mg/0.5mL', payer_id, criteria, approval_duration_days, renewal_allowed, required_documents, effective_date, version
FROM pa_requirements WHERE drug_rxnorm = '1991302'
ON CONFLICT (drug_rxnorm, payer_id) DO NOTHING;

-- Invokana 300mg uses same PA as 100mg
INSERT INTO pa_requirements (drug_rxnorm, drug_name, payer_id, criteria, approval_duration_days, renewal_allowed, required_documents, effective_date, version)
SELECT '1373463', 'Canagliflozin (Invokana) 300mg', payer_id, criteria, approval_duration_days, renewal_allowed, required_documents, effective_date, version
FROM pa_requirements WHERE drug_rxnorm = '1373458'
ON CONFLICT (drug_rxnorm, payer_id) DO NOTHING;
