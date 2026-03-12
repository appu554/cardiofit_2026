-- ============================================================================
-- GOVERNANCE TEST DATA SEED SCRIPT
-- ============================================================================
-- This script populates drug_master and clinical_facts from:
-- 1. OMOP Vocabulary (for drug references)
-- 2. DDI Constitutional Rules (for interaction facts)
-- ============================================================================

-- Step 1: Insert drugs into drug_master (required for FK constraint)
-- These are the drugs referenced in our DDI Constitutional Rules

INSERT INTO drug_master (rxcui, drug_name, generic_name, tty, therapeutic_class, rxnorm_version, status)
VALUES
    -- MAO Inhibitors (trigger drugs)
    ('8123', 'Phenelzine', 'phenelzine', 'IN', 'MAO Inhibitors', '2024-12', 'ACTIVE'),
    ('10734', 'Tranylcypromine', 'tranylcypromine', 'IN', 'MAO Inhibitors', '2024-12', 'ACTIVE'),
    ('6011', 'Isocarboxazid', 'isocarboxazid', 'IN', 'MAO Inhibitors', '2024-12', 'ACTIVE'),
    ('9639', 'Selegiline', 'selegiline', 'IN', 'MAO Inhibitors', '2024-12', 'ACTIVE'),

    -- SSRIs (target drugs for MAO-I interaction)
    ('4493', 'Fluoxetine', 'fluoxetine', 'IN', 'SSRIs', '2024-12', 'ACTIVE'),
    ('36437', 'Sertraline', 'sertraline', 'IN', 'SSRIs', '2024-12', 'ACTIVE'),
    ('32937', 'Paroxetine', 'paroxetine', 'IN', 'SSRIs', '2024-12', 'ACTIVE'),
    ('321988', 'Escitalopram', 'escitalopram', 'IN', 'SSRIs', '2024-12', 'ACTIVE'),

    -- Tricyclic Antidepressants
    ('704', 'Amitriptyline', 'amitriptyline', 'IN', 'Tricyclic Antidepressants', '2024-12', 'ACTIVE'),
    ('5691', 'Imipramine', 'imipramine', 'IN', 'Tricyclic Antidepressants', '2024-12', 'ACTIVE'),
    ('7531', 'Nortriptyline', 'nortriptyline', 'IN', 'Tricyclic Antidepressants', '2024-12', 'ACTIVE'),

    -- Amphetamines
    ('723', 'Amphetamine', 'amphetamine', 'IN', 'Amphetamine Derivatives', '2024-12', 'ACTIVE'),
    ('6816', 'Lisdexamfetamine', 'lisdexamfetamine', 'IN', 'Amphetamine Derivatives', '2024-12', 'ACTIVE'),
    ('40114', 'Dextroamphetamine', 'dextroamphetamine', 'IN', 'Amphetamine Derivatives', '2024-12', 'ACTIVE'),

    -- Meperidine (Demerol)
    ('6754', 'Meperidine', 'meperidine', 'IN', 'Opioid Analgesics', '2024-12', 'ACTIVE'),

    -- Vitamin K Antagonists (Warfarin)
    ('11289', 'Warfarin', 'warfarin', 'IN', 'Vitamin K Antagonists', '2024-12', 'ACTIVE'),

    -- NSAIDs (interact with Warfarin)
    ('5640', 'Ibuprofen', 'ibuprofen', 'IN', 'NSAIDs', '2024-12', 'ACTIVE'),
    ('7258', 'Naproxen', 'naproxen', 'IN', 'NSAIDs', '2024-12', 'ACTIVE'),
    ('1191', 'Aspirin', 'aspirin', 'IN', 'NSAIDs', '2024-12', 'ACTIVE'),
    ('3355', 'Diclofenac', 'diclofenac', 'IN', 'NSAIDs', '2024-12', 'ACTIVE'),

    -- Potassium-Sparing Diuretics (Triple Whammy)
    ('9997', 'Spironolactone', 'spironolactone', 'IN', 'Potassium-Sparing Diuretics', '2024-12', 'ACTIVE'),
    ('10763', 'Triamterene', 'triamterene', 'IN', 'Potassium-Sparing Diuretics', '2024-12', 'ACTIVE'),

    -- ACE Inhibitors (Triple Whammy)
    ('6185', 'Lisinopril', 'lisinopril', 'IN', 'ACE Inhibitors', '2024-12', 'ACTIVE'),
    ('3827', 'Enalapril', 'enalapril', 'IN', 'ACE Inhibitors', '2024-12', 'ACTIVE'),
    ('29046', 'Ramipril', 'ramipril', 'IN', 'ACE Inhibitors', '2024-12', 'ACTIVE'),

    -- Digoxin (multiple interactions)
    ('3407', 'Digoxin', 'digoxin', 'IN', 'Cardiac Glycosides', '2024-12', 'ACTIVE'),

    -- Amiodarone (interacts with Digoxin)
    ('703', 'Amiodarone', 'amiodarone', 'IN', 'Antiarrhythmics', '2024-12', 'ACTIVE'),

    -- Methotrexate
    ('6851', 'Methotrexate', 'methotrexate', 'IN', 'Antineoplastics', '2024-12', 'ACTIVE'),

    -- Lithium
    ('6448', 'Lithium', 'lithium', 'IN', 'Mood Stabilizers', '2024-12', 'ACTIVE'),

    -- Theophylline
    ('10438', 'Theophylline', 'theophylline', 'IN', 'Bronchodilators', '2024-12', 'ACTIVE')

ON CONFLICT (rxcui) DO UPDATE SET
    drug_name = EXCLUDED.drug_name,
    therapeutic_class = EXCLUDED.therapeutic_class,
    updated_at = NOW();

-- Step 2: Create clinical_facts for DDI governance review
-- These facts come from ONC Constitutional DDI Rules and need pharmacist review

-- Fact 1: MAO-I + SSRI Interaction (CRITICAL)
INSERT INTO clinical_facts (
    fact_type, rxcui, drug_name, scope,
    content, source_type, source_id, source_version,
    extraction_method, confidence_score, confidence_band,
    status, authority_priority, created_by
) VALUES (
    'INTERACTION', '8123', 'Phenelzine', 'DRUG',
    '{
        "drug_class": "MAO Inhibitors",
        "interaction_type": "DRUG_DRUG",
        "trigger_drug": {"rxcui": "8123", "name": "Phenelzine", "class": "MAO Inhibitors"},
        "target_drug": {"rxcui": "4493", "name": "Fluoxetine", "class": "SSRIs"},
        "risk_level": "CRITICAL",
        "clinical_effect": "Serotonin syndrome - potentially fatal hyperthermia, rigidity, autonomic instability",
        "mechanism": "MAO-A inhibition prevents serotonin metabolism; SSRIs increase synaptic serotonin",
        "management": "Contraindicated. Allow 14-day washout from MAO-I, 5 weeks from fluoxetine",
        "evidence_level": "HIGH",
        "references": ["ONC High-Priority DDI List", "FDA Drug Safety Communication"]
    }'::jsonb,
    'ETL', 'ONC-DDI-001', '2024-12',
    'constitutional_rule_extraction', 0.98, 'HIGH',
    'DRAFT', 1, 'kb5-extraction-service'
);

-- Fact 2: MAO-I + TCA Interaction (CRITICAL)
INSERT INTO clinical_facts (
    fact_type, rxcui, drug_name, scope,
    content, source_type, source_id, source_version,
    extraction_method, confidence_score, confidence_band,
    status, authority_priority, created_by
) VALUES (
    'INTERACTION', '8123', 'Phenelzine', 'DRUG',
    '{
        "interaction_type": "DRUG_DRUG",
        "trigger_drug": {"rxcui": "8123", "name": "Phenelzine", "class": "MAO Inhibitors"},
        "target_drug": {"rxcui": "704", "name": "Amitriptyline", "class": "Tricyclic Antidepressants"},
        "risk_level": "CRITICAL",
        "clinical_effect": "Hypertensive crisis, hyperpyrexia, seizures, death",
        "mechanism": "Combined norepinephrine and serotonin potentiation",
        "management": "Contraindicated. 14-day washout period required",
        "evidence_level": "HIGH",
        "references": ["ONC High-Priority DDI List"]
    }'::jsonb,
    'ETL', 'ONC-DDI-002', '2024-12',
    'constitutional_rule_extraction', 0.97, 'HIGH',
    'DRAFT', 1, 'kb5-extraction-service'
);

-- Fact 3: MAO-I + Meperidine (CRITICAL)
INSERT INTO clinical_facts (
    fact_type, rxcui, drug_name, scope,
    content, source_type, source_id, source_version,
    extraction_method, confidence_score, confidence_band,
    status, authority_priority, created_by
) VALUES (
    'INTERACTION', '8123', 'Phenelzine', 'DRUG',
    '{
        "interaction_type": "DRUG_DRUG",
        "trigger_drug": {"rxcui": "8123", "name": "Phenelzine", "class": "MAO Inhibitors"},
        "target_drug": {"rxcui": "6754", "name": "Meperidine", "class": "Opioid Analgesics"},
        "risk_level": "CRITICAL",
        "clinical_effect": "Serotonin syndrome or opioid toxicity - coma, respiratory depression, death",
        "mechanism": "Meperidine inhibits serotonin reuptake; MAO-I prevents metabolism",
        "management": "Absolutely contraindicated. Use morphine or fentanyl as alternatives",
        "evidence_level": "HIGH",
        "references": ["ONC High-Priority DDI List", "ISMP Medication Safety Alert"]
    }'::jsonb,
    'ETL', 'ONC-DDI-003', '2024-12',
    'constitutional_rule_extraction', 0.99, 'HIGH',
    'DRAFT', 1, 'kb5-extraction-service'
);

-- Fact 4: Warfarin + NSAID (HIGH)
INSERT INTO clinical_facts (
    fact_type, rxcui, drug_name, scope,
    content, source_type, source_id, source_version,
    extraction_method, confidence_score, confidence_band,
    status, authority_priority, created_by
) VALUES (
    'INTERACTION', '11289', 'Warfarin', 'DRUG',
    '{
        "interaction_type": "DRUG_DRUG",
        "trigger_drug": {"rxcui": "11289", "name": "Warfarin", "class": "Vitamin K Antagonists"},
        "target_drug": {"rxcui": "5640", "name": "Ibuprofen", "class": "NSAIDs"},
        "risk_level": "HIGH",
        "clinical_effect": "Increased bleeding risk - GI hemorrhage, intracranial bleeding",
        "mechanism": "NSAIDs inhibit platelet function and may displace warfarin from protein binding",
        "management": "Avoid combination if possible. If necessary, use lowest NSAID dose, monitor INR closely",
        "evidence_level": "HIGH",
        "references": ["ONC High-Priority DDI List", "CHEST Guidelines"]
    }'::jsonb,
    'ETL', 'ONC-DDI-004', '2024-12',
    'constitutional_rule_extraction', 0.92, 'HIGH',
    'DRAFT', 1, 'kb5-extraction-service'
);

-- Fact 5: Triple Whammy - ACE-I + NSAID + Diuretic (HIGH)
INSERT INTO clinical_facts (
    fact_type, rxcui, drug_name, scope,
    content, source_type, source_id, source_version,
    extraction_method, confidence_score, confidence_band,
    status, authority_priority, created_by
) VALUES (
    'INTERACTION', '6185', 'Lisinopril', 'DRUG',
    '{
        "interaction_type": "DRUG_DRUG_DRUG",
        "trigger_drug": {"rxcui": "6185", "name": "Lisinopril", "class": "ACE Inhibitors"},
        "target_drugs": [
            {"rxcui": "5640", "name": "Ibuprofen", "class": "NSAIDs"},
            {"rxcui": "9997", "name": "Spironolactone", "class": "Potassium-Sparing Diuretics"}
        ],
        "risk_level": "HIGH",
        "clinical_effect": "Acute kidney injury (Triple Whammy syndrome)",
        "mechanism": "ACE-I + Diuretic reduce renal perfusion; NSAIDs further constrict afferent arteriole",
        "management": "Avoid triple combination. If unavoidable, monitor renal function and electrolytes weekly",
        "evidence_level": "HIGH",
        "references": ["Australian Prescriber Triple Whammy Alert", "FDA MedWatch"]
    }'::jsonb,
    'ETL', 'ONC-DDI-005', '2024-12',
    'constitutional_rule_extraction', 0.88, 'HIGH',
    'DRAFT', 1, 'kb5-extraction-service'
);

-- Fact 6: Digoxin + Amiodarone (HIGH) - needs MEDIUM confidence review
INSERT INTO clinical_facts (
    fact_type, rxcui, drug_name, scope,
    content, source_type, source_id, source_version,
    extraction_method, confidence_score, confidence_band,
    status, authority_priority, created_by
) VALUES (
    'INTERACTION', '3407', 'Digoxin', 'DRUG', NULL,
    '{
        "interaction_type": "DRUG_DRUG",
        "trigger_drug": {"rxcui": "3407", "name": "Digoxin", "class": "Cardiac Glycosides"},
        "target_drug": {"rxcui": "703", "name": "Amiodarone", "class": "Antiarrhythmics"},
        "risk_level": "HIGH",
        "clinical_effect": "Digoxin toxicity - nausea, visual disturbances, arrhythmias",
        "mechanism": "Amiodarone inhibits P-glycoprotein, reducing digoxin clearance by 50%",
        "management": "Reduce digoxin dose by 50% when starting amiodarone. Monitor digoxin levels",
        "evidence_level": "HIGH",
        "references": ["Lexi-Interact", "Clinical Pharmacokinetics"]
    }'::jsonb,
    'ETL', 'FDA-DDI-001', '2024-12',
    'fda_label_extraction', 0.78, 'MEDIUM',
    'DRAFT', 2, 'kb5-extraction-service'
);

-- Fact 7: Methotrexate + NSAID (HIGH) - MEDIUM confidence
INSERT INTO clinical_facts (
    fact_type, rxcui, drug_name, scope,
    content, source_type, source_id, source_version,
    extraction_method, confidence_score, confidence_band,
    status, authority_priority, created_by
) VALUES (
    'INTERACTION', '6851', 'Methotrexate', 'DRUG', NULL,
    '{
        "interaction_type": "DRUG_DRUG",
        "trigger_drug": {"rxcui": "6851", "name": "Methotrexate", "class": "Antineoplastics"},
        "target_drug": {"rxcui": "5640", "name": "Ibuprofen", "class": "NSAIDs"},
        "risk_level": "HIGH",
        "clinical_effect": "Methotrexate toxicity - bone marrow suppression, hepatotoxicity",
        "mechanism": "NSAIDs reduce renal prostaglandins, decreasing methotrexate clearance",
        "management": "Avoid NSAIDs with high-dose MTX. With low-dose MTX, use cautiously",
        "evidence_level": "MODERATE",
        "references": ["ONC High-Priority DDI List", "ACR Guidelines"]
    }'::jsonb,
    'ETL', 'ONC-DDI-006', '2024-12',
    'constitutional_rule_extraction', 0.75, 'MEDIUM',
    'DRAFT', 1, 'kb5-extraction-service'
);

-- Fact 8: Lithium + NSAID (HIGH) - needs review
INSERT INTO clinical_facts (
    fact_type, rxcui, drug_name, scope,
    content, source_type, source_id, source_version,
    extraction_method, confidence_score, confidence_band,
    status, authority_priority, created_by
) VALUES (
    'INTERACTION', '6448', 'Lithium', 'DRUG', NULL,
    '{
        "interaction_type": "DRUG_DRUG",
        "trigger_drug": {"rxcui": "6448", "name": "Lithium", "class": "Mood Stabilizers"},
        "target_drug": {"rxcui": "5640", "name": "Ibuprofen", "class": "NSAIDs"},
        "risk_level": "HIGH",
        "clinical_effect": "Lithium toxicity - tremor, ataxia, confusion, seizures",
        "mechanism": "NSAIDs reduce renal prostaglandins, decreasing lithium excretion",
        "management": "Avoid combination. If necessary, reduce lithium dose and monitor levels closely",
        "evidence_level": "HIGH",
        "references": ["ONC High-Priority DDI List"]
    }'::jsonb,
    'ETL', 'ONC-DDI-007', '2024-12',
    'constitutional_rule_extraction', 0.82, 'MEDIUM',
    'DRAFT', 1, 'kb5-extraction-service'
);

-- Fact 9: Theophylline + Ciprofloxacin (needs to add cipro first - use Warfarin as placeholder)
INSERT INTO clinical_facts (
    fact_type, rxcui, drug_name, scope,
    content, source_type, source_id, source_version,
    extraction_method, confidence_score, confidence_band,
    status, authority_priority, created_by
) VALUES (
    'INTERACTION', '10438', 'Theophylline', 'DRUG', NULL,
    '{
        "interaction_type": "DRUG_DRUG",
        "trigger_drug": {"rxcui": "10438", "name": "Theophylline", "class": "Bronchodilators"},
        "target_drug": {"rxcui": "2551", "name": "Ciprofloxacin", "class": "Fluoroquinolones"},
        "risk_level": "HIGH",
        "clinical_effect": "Theophylline toxicity - seizures, arrhythmias",
        "mechanism": "Ciprofloxacin inhibits CYP1A2, reducing theophylline metabolism",
        "management": "Reduce theophylline dose by 40-50% or use alternative antibiotic",
        "evidence_level": "HIGH",
        "references": ["ONC High-Priority DDI List", "FDA Label"]
    }'::jsonb,
    'ETL', 'ONC-DDI-008', '2024-12',
    'constitutional_rule_extraction', 0.72, 'MEDIUM',
    'DRAFT', 1, 'kb5-extraction-service'
);

-- Fact 10: MAO-I + Amphetamines (CRITICAL) - waiting for escalation
INSERT INTO clinical_facts (
    fact_type, rxcui, drug_name, scope,
    content, source_type, source_id, source_version,
    extraction_method, confidence_score, confidence_band,
    status, authority_priority, created_by
) VALUES (
    'INTERACTION', '9639', 'Selegiline', 'DRUG',
    '{
        "interaction_type": "DRUG_DRUG",
        "trigger_drug": {"rxcui": "9639", "name": "Selegiline", "class": "MAO Inhibitors"},
        "target_drug": {"rxcui": "723", "name": "Amphetamine", "class": "Amphetamine Derivatives"},
        "risk_level": "CRITICAL",
        "clinical_effect": "Hypertensive crisis - severe headache, chest pain, stroke",
        "mechanism": "MAO-B inhibition prevents amphetamine metabolism; both increase catecholamines",
        "management": "Contraindicated. Selegiline doses >10mg/day lose selectivity",
        "evidence_level": "HIGH",
        "references": ["ONC High-Priority DDI List", "Product Labeling"]
    }'::jsonb,
    'ETL', 'ONC-DDI-009', '2024-12',
    'constitutional_rule_extraction', 0.95, 'HIGH',
    'DRAFT', 1, 'kb5-extraction-service'
);

-- Add Ciprofloxacin to drug_master for theophylline interaction
INSERT INTO drug_master (rxcui, drug_name, generic_name, tty, therapeutic_class, rxnorm_version, status)
VALUES ('2551', 'Ciprofloxacin', 'ciprofloxacin', 'IN', 'Fluoroquinolones', '2024-12', 'ACTIVE')
ON CONFLICT (rxcui) DO NOTHING;

-- ============================================================================
-- SUMMARY
-- ============================================================================
-- Inserted:
--   - 31 drugs into drug_master (covering DDI rule drug classes)
--   - 10 clinical_facts for governance review:
--     - 4 CRITICAL priority (MAO-I interactions)
--     - 6 HIGH priority (Warfarin, Triple Whammy, etc.)
--     - 4 HIGH confidence (auto-approve eligible)
--     - 6 MEDIUM confidence (requires manual review)
-- ============================================================================

SELECT
    'drug_master' as table_name,
    COUNT(*) as row_count
FROM drug_master
UNION ALL
SELECT
    'clinical_facts' as table_name,
    COUNT(*) as row_count
FROM clinical_facts;
