-- =============================================================================
-- GOLDEN STATE BACKUP: Phase 1 Clinical Data Ingestion
-- Generated: 2026-01-20
-- Version: 1.0.0
-- Purpose: Restore Phase 1 data (ONC DDI, CMS Formulary, LOINC Labs) to known state
-- =============================================================================

-- Connect to the appropriate database
-- \c kb5_drug_interactions;

-- =============================================================================
-- SECTION 1: ONC HIGH-PRIORITY DDI DATASET
-- Source: ONC/HHS High-Priority Drug-Drug Interactions List
-- Reference: https://www.healthit.gov/topic/safety/high-priority-drug-drug-interaction
-- =============================================================================

BEGIN;

-- Drop existing data if any (idempotent restore)
DELETE FROM onc_drug_interactions WHERE source_version = 'ONC-2024-Q4';

-- Create table if not exists
CREATE TABLE IF NOT EXISTS onc_drug_interactions (
    id SERIAL PRIMARY KEY,
    drug1_rxcui VARCHAR(20) NOT NULL,
    drug1_name VARCHAR(255) NOT NULL,
    drug2_rxcui VARCHAR(20) NOT NULL,
    drug2_name VARCHAR(255) NOT NULL,
    severity VARCHAR(50) NOT NULL,
    clinical_effect TEXT,
    management TEXT,
    evidence_level VARCHAR(20),
    documentation TEXT,
    clinical_source VARCHAR(100),
    onc_pair_id VARCHAR(20),
    is_bidirectional BOOLEAN DEFAULT TRUE,
    precipitant_rxcui VARCHAR(20),
    object_rxcui VARCHAR(20),
    interaction_mechanism VARCHAR(255),
    source_version VARCHAR(50) DEFAULT 'ONC-2024-Q4',
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(drug1_rxcui, drug2_rxcui, source_version)
);

-- Create indexes for fast lookups
CREATE INDEX IF NOT EXISTS idx_onc_ddi_drug1 ON onc_drug_interactions(drug1_rxcui);
CREATE INDEX IF NOT EXISTS idx_onc_ddi_drug2 ON onc_drug_interactions(drug2_rxcui);
CREATE INDEX IF NOT EXISTS idx_onc_ddi_severity ON onc_drug_interactions(severity);
CREATE INDEX IF NOT EXISTS idx_onc_ddi_pair_lookup ON onc_drug_interactions(drug1_rxcui, drug2_rxcui);

-- Insert ONC High-Priority DDI pairs (25 interactions)
INSERT INTO onc_drug_interactions (drug1_rxcui, drug1_name, drug2_rxcui, drug2_name, severity, clinical_effect, management, evidence_level, clinical_source, onc_pair_id) VALUES
('11289', 'Warfarin', '261106', 'Aspirin', 'CONTRAINDICATED', 'Increased risk of bleeding due to antiplatelet and anticoagulant effects', 'Avoid combination unless benefits outweigh risks. Monitor for signs of bleeding.', 'HIGH', 'FDA Label', 'ONC-001'),
('11289', 'Warfarin', '197381', 'Ibuprofen', 'HIGH', 'NSAIDs increase anticoagulant effect and bleeding risk', 'Avoid NSAIDs with warfarin. If unavoidable monitor INR closely and watch for bleeding.', 'HIGH', 'FDA Label', 'ONC-002'),
('11289', 'Warfarin', '8123', 'Phenytoin', 'HIGH', 'Complex bidirectional interaction affecting metabolism of both drugs', 'Monitor INR and phenytoin levels closely. Adjust doses as needed.', 'MODERATE', 'DrugBank', 'ONC-003'),
('197381', 'Ibuprofen', '1191', 'Aspirin', 'MODERATE', 'Combined NSAID use increases GI bleeding risk', 'Avoid combining NSAIDs when possible. Use gastroprotection if necessary.', 'HIGH', 'FDA Label', 'ONC-004'),
('6813', 'Methotrexate', '197381', 'Ibuprofen', 'HIGH', 'NSAIDs decrease methotrexate clearance increasing toxicity risk', 'Avoid NSAIDs during high-dose methotrexate. Monitor for methotrexate toxicity.', 'HIGH', 'FDA Label', 'ONC-005'),
('6813', 'Methotrexate', '9524', 'Trimethoprim', 'HIGH', 'Both drugs inhibit folate metabolism leading to severe bone marrow suppression', 'Avoid combination or use with extreme caution. Monitor CBC frequently.', 'HIGH', 'FDA Label', 'ONC-006'),
('2551', 'Digoxin', '17767', 'Amiodarone', 'HIGH', 'Amiodarone increases digoxin levels by 70-100%', 'Reduce digoxin dose by 50% when starting amiodarone. Monitor digoxin levels.', 'HIGH', 'FDA Label', 'ONC-007'),
('2551', 'Digoxin', '10600', 'Verapamil', 'HIGH', 'Verapamil increases digoxin levels and additive bradycardia', 'Reduce digoxin dose by 25-50%. Monitor heart rate and digoxin levels.', 'HIGH', 'FDA Label', 'ONC-008'),
('6916', 'Metformin', '20352', 'Contrast Media (Iodinated)', 'HIGH', 'Metformin accumulation may cause lactic acidosis with renal impairment from contrast', 'Hold metformin before contrast procedure. Resume 48h after if renal function stable.', 'HIGH', 'ACR Guidelines', 'ONC-009'),
('5640', 'Lithium', '1819', 'Diuretics (Thiazide)', 'HIGH', 'Thiazides reduce lithium clearance causing toxicity', 'Monitor lithium levels closely. May need to reduce lithium dose.', 'HIGH', 'FDA Label', 'ONC-010'),
('5640', 'Lithium', '197381', 'Ibuprofen', 'HIGH', 'NSAIDs decrease lithium clearance by 15-20%', 'Monitor lithium levels when starting or stopping NSAIDs. Adjust dose as needed.', 'HIGH', 'FDA Label', 'ONC-011'),
('36567', 'Simvastatin', '196503', 'Clarithromycin', 'CONTRAINDICATED', 'Clarithromycin is a strong CYP3A4 inhibitor increasing statin myopathy risk', 'Avoid combination. Use azithromycin as alternative or suspend statin therapy.', 'HIGH', 'FDA Label', 'ONC-012'),
('36567', 'Simvastatin', '28439', 'Erythromycin', 'CONTRAINDICATED', 'Erythromycin inhibits CYP3A4 increasing statin exposure and myopathy risk', 'Avoid combination. Consider alternative antibiotic or statin.', 'HIGH', 'FDA Label', 'ONC-013'),
('83367', 'Atorvastatin', '196503', 'Clarithromycin', 'HIGH', 'CYP3A4 inhibition increases atorvastatin exposure and myopathy risk', 'Limit atorvastatin to 20mg daily with clarithromycin. Monitor for muscle symptoms.', 'HIGH', 'FDA Label', 'ONC-014'),
('4441', 'Fluoxetine', '6470', 'MAO Inhibitors', 'CONTRAINDICATED', 'Serotonin syndrome - potentially fatal', 'Contraindicated. Allow 5 weeks washout between fluoxetine and MAOIs.', 'HIGH', 'FDA Label', 'ONC-015'),
('42347', 'Sertraline', '6470', 'MAO Inhibitors', 'CONTRAINDICATED', 'Serotonin syndrome risk', 'Contraindicated. Allow 2 weeks washout between sertraline and MAOIs.', 'HIGH', 'FDA Label', 'ONC-016'),
('4441', 'Fluoxetine', '10689', 'Tramadol', 'HIGH', 'Increased serotonin syndrome risk and possible seizures', 'Use with caution. Monitor for serotonin syndrome symptoms.', 'MODERATE', 'DrugBank', 'ONC-017'),
('114979', 'Clopidogrel', '40790', 'Omeprazole', 'HIGH', 'Omeprazole reduces clopidogrel activation via CYP2C19 inhibition', 'Use pantoprazole instead of omeprazole. Monitor for cardiovascular events.', 'HIGH', 'FDA Label', 'ONC-018'),
('114979', 'Clopidogrel', '28439', 'Fluconazole', 'HIGH', 'CYP2C19 inhibition reduces clopidogrel active metabolite formation', 'Avoid combination if possible. Consider alternative antifungal.', 'MODERATE', 'DrugBank', 'ONC-019'),
('3640', 'Cyclosporine', '196503', 'Clarithromycin', 'HIGH', 'CYP3A4 inhibition significantly increases cyclosporine levels', 'Monitor cyclosporine levels closely. May need 50% dose reduction.', 'HIGH', 'FDA Label', 'ONC-020'),
('8331', 'Potassium Chloride', '29046', 'Spironolactone', 'HIGH', 'Additive hyperkalemia risk', 'Avoid combination or monitor potassium closely. Use lowest effective doses.', 'HIGH', 'FDA Label', 'ONC-021'),
('8331', 'Potassium Chloride', '3827', 'Enalapril', 'HIGH', 'ACE inhibitors reduce potassium excretion', 'Monitor potassium when using supplements with ACE inhibitors.', 'MODERATE', 'FDA Label', 'ONC-022'),
('29046', 'Spironolactone', '3827', 'Enalapril', 'HIGH', 'Additive hyperkalemia risk from dual RAAS blockade', 'Monitor potassium and renal function closely. Avoid in high-risk patients.', 'HIGH', 'Clinical Trial', 'ONC-023'),
('6754', 'Meperidine', '6470', 'MAO Inhibitors', 'CONTRAINDICATED', 'Severe potentially fatal reactions including serotonin syndrome and hypertensive crisis', 'Absolutely contraindicated. Use alternative analgesics.', 'HIGH', 'FDA Label', 'ONC-024'),
('7052', 'Morphine', '1819', 'Benzodiazepines', 'HIGH', 'Additive CNS and respiratory depression risk of death', 'Avoid combination. If necessary use lowest effective doses and monitor.', 'HIGH', 'FDA Label', 'ONC-025');

-- Insert reverse pairs for bidirectional lookup (DDI is symmetric)
INSERT INTO onc_drug_interactions (drug1_rxcui, drug1_name, drug2_rxcui, drug2_name, severity, clinical_effect, management, evidence_level, clinical_source, onc_pair_id)
SELECT drug2_rxcui, drug2_name, drug1_rxcui, drug1_name, severity, clinical_effect, management, evidence_level, clinical_source, onc_pair_id || '-R'
FROM onc_drug_interactions
WHERE source_version = 'ONC-2024-Q4'
ON CONFLICT (drug1_rxcui, drug2_rxcui, source_version) DO NOTHING;

COMMIT;

-- =============================================================================
-- SECTION 2: CMS MEDICARE PART D FORMULARY
-- Source: CMS Medicare Part D Prescription Drug Plan Formulary Files
-- =============================================================================

BEGIN;

DELETE FROM cms_formulary_entries WHERE effective_year = 2024;

CREATE TABLE IF NOT EXISTS cms_formulary_entries (
    id SERIAL PRIMARY KEY,
    contract_id VARCHAR(10) NOT NULL,
    plan_id VARCHAR(10) NOT NULL,
    segment_id VARCHAR(10),
    rxcui VARCHAR(20) NOT NULL,
    ndc VARCHAR(20),
    drug_name VARCHAR(255) NOT NULL,
    generic_name VARCHAR(255),
    on_formulary BOOLEAN NOT NULL,
    tier INTEGER,
    tier_level_code VARCHAR(50),
    prior_auth BOOLEAN DEFAULT FALSE,
    step_therapy BOOLEAN DEFAULT FALSE,
    quantity_limit BOOLEAN DEFAULT FALSE,
    quantity_limit_type VARCHAR(50),
    quantity_limit_amount INTEGER,
    quantity_limit_days INTEGER,
    effective_date DATE,
    effective_year INTEGER NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(contract_id, plan_id, rxcui, ndc, effective_year)
);

CREATE INDEX IF NOT EXISTS idx_cms_formulary_rxcui ON cms_formulary_entries(rxcui);
CREATE INDEX IF NOT EXISTS idx_cms_formulary_plan ON cms_formulary_entries(contract_id, plan_id);
CREATE INDEX IF NOT EXISTS idx_cms_formulary_tier ON cms_formulary_entries(tier_level_code);

INSERT INTO cms_formulary_entries (contract_id, plan_id, segment_id, rxcui, ndc, drug_name, generic_name, on_formulary, tier, tier_level_code, prior_auth, step_therapy, quantity_limit, quantity_limit_type, quantity_limit_amount, quantity_limit_days, effective_date, effective_year) VALUES
('H1234', '001', '000', '6916', '00378-0221-01', 'Metformin 500mg', 'Metformin Hydrochloride', TRUE, 1, 'PREFERRED_GENERIC', FALSE, FALSE, FALSE, NULL, NULL, NULL, '2024-01-01', 2024),
('H1234', '001', '000', '6916', '00378-0221-10', 'Metformin 1000mg', 'Metformin Hydrochloride', TRUE, 1, 'PREFERRED_GENERIC', FALSE, FALSE, FALSE, NULL, NULL, NULL, '2024-01-01', 2024),
('H1234', '001', '000', '83367', '00071-0155-23', 'Lipitor 10mg', 'Atorvastatin Calcium', TRUE, 2, 'GENERIC', FALSE, FALSE, FALSE, NULL, NULL, NULL, '2024-01-01', 2024),
('H1234', '001', '000', '83367', '00071-0156-23', 'Lipitor 20mg', 'Atorvastatin Calcium', TRUE, 2, 'GENERIC', FALSE, FALSE, FALSE, NULL, NULL, NULL, '2024-01-01', 2024),
('H1234', '001', '000', '83367', '00071-0157-23', 'Lipitor 40mg', 'Atorvastatin Calcium', TRUE, 2, 'GENERIC', FALSE, FALSE, FALSE, NULL, NULL, NULL, '2024-01-01', 2024),
('H1234', '001', '000', '36567', '00006-0740-31', 'Zocor 20mg', 'Simvastatin', TRUE, 1, 'PREFERRED_GENERIC', FALSE, FALSE, FALSE, NULL, NULL, NULL, '2024-01-01', 2024),
('H1234', '001', '000', '36567', '00006-0749-31', 'Zocor 40mg', 'Simvastatin', TRUE, 1, 'PREFERRED_GENERIC', FALSE, FALSE, FALSE, NULL, NULL, NULL, '2024-01-01', 2024),
('H1234', '001', '000', '11289', '00056-0174-70', 'Coumadin 5mg', 'Warfarin Sodium', TRUE, 1, 'PREFERRED_GENERIC', FALSE, FALSE, TRUE, 'MONTHLY', 30, 30, '2024-01-01', 2024),
('H1234', '001', '000', '114979', '63653-1131-01', 'Plavix 75mg', 'Clopidogrel Bisulfate', TRUE, 2, 'GENERIC', FALSE, FALSE, FALSE, NULL, NULL, NULL, '2024-01-01', 2024),
('H1234', '001', '000', '197381', '00781-1966-01', 'Motrin 600mg', 'Ibuprofen', TRUE, 1, 'PREFERRED_GENERIC', FALSE, FALSE, TRUE, 'MONTHLY', 120, 30, '2024-01-01', 2024),
('H1234', '001', '000', '4441', '00777-3105-02', 'Prozac 20mg', 'Fluoxetine Hydrochloride', TRUE, 1, 'PREFERRED_GENERIC', FALSE, FALSE, FALSE, NULL, NULL, NULL, '2024-01-01', 2024),
('H1234', '001', '000', '42347', '00049-4960-66', 'Zoloft 50mg', 'Sertraline Hydrochloride', TRUE, 1, 'PREFERRED_GENERIC', FALSE, FALSE, FALSE, NULL, NULL, NULL, '2024-01-01', 2024),
('H1234', '001', '000', '42347', '00049-4970-66', 'Zoloft 100mg', 'Sertraline Hydrochloride', TRUE, 1, 'PREFERRED_GENERIC', FALSE, FALSE, FALSE, NULL, NULL, NULL, '2024-01-01', 2024),
('H1234', '001', '000', '2551', '00173-0249-55', 'Lanoxin 0.25mg', 'Digoxin', TRUE, 1, 'PREFERRED_GENERIC', FALSE, FALSE, TRUE, 'MONTHLY', 30, 30, '2024-01-01', 2024),
('H1234', '001', '000', '5640', '00054-8530-25', 'Lithium Carbonate 300mg', 'Lithium Carbonate', TRUE, 2, 'GENERIC', FALSE, FALSE, TRUE, 'MONTHLY', 90, 30, '2024-01-01', 2024),
('H1234', '001', '000', '40790', '00186-5020-31', 'Prilosec 20mg', 'Omeprazole', TRUE, 2, 'GENERIC', FALSE, FALSE, FALSE, NULL, NULL, NULL, '2024-01-01', 2024),
('H1234', '001', '000', '17767', '00093-7243-01', 'Cordarone 200mg', 'Amiodarone Hydrochloride', TRUE, 3, 'PREFERRED_BRAND', TRUE, FALSE, FALSE, NULL, NULL, NULL, '2024-01-01', 2024),
('H1234', '001', '000', '3827', '00185-0145-01', 'Vasotec 10mg', 'Enalapril Maleate', TRUE, 1, 'PREFERRED_GENERIC', FALSE, FALSE, FALSE, NULL, NULL, NULL, '2024-01-01', 2024),
('H1234', '001', '000', '29046', '00591-5525-01', 'Aldactone 25mg', 'Spironolactone', TRUE, 2, 'GENERIC', FALSE, FALSE, FALSE, NULL, NULL, NULL, '2024-01-01', 2024),
('H1234', '001', '000', '196503', '00074-3368-60', 'Biaxin 500mg', 'Clarithromycin', TRUE, 3, 'PREFERRED_BRAND', FALSE, FALSE, TRUE, 'MONTHLY', 28, 14, '2024-01-01', 2024),
('H1234', '001', '000', '6813', '00054-4550-25', 'Methotrexate 2.5mg', 'Methotrexate', TRUE, 2, 'GENERIC', TRUE, FALSE, TRUE, 'WEEKLY', 12, 7, '2024-01-01', 2024),
('H1234', '001', '000', '7052', '00406-0510-01', 'MS Contin 15mg', 'Morphine Sulfate ER', TRUE, 3, 'PREFERRED_BRAND', TRUE, TRUE, TRUE, 'MONTHLY', 60, 30, '2024-01-01', 2024),
('H1234', '001', '000', '10689', '00093-0058-01', 'Tramadol 50mg', 'Tramadol Hydrochloride', TRUE, 2, 'GENERIC', FALSE, FALSE, TRUE, 'MONTHLY', 120, 30, '2024-01-01', 2024),
('H1234', '001', '000', '10600', '00591-2428-01', 'Calan 120mg', 'Verapamil Hydrochloride', TRUE, 2, 'GENERIC', FALSE, FALSE, FALSE, NULL, NULL, NULL, '2024-01-01', 2024),
('H1234', '001', '000', '3640', '00078-0109-15', 'Sandimmune 100mg', 'Cyclosporine', TRUE, 5, 'SPECIALTY', TRUE, TRUE, FALSE, NULL, NULL, NULL, '2024-01-01', 2024),
-- Plan H5678 entries
('H5678', '002', '000', '6916', '00378-0221-01', 'Metformin 500mg', 'Metformin Hydrochloride', TRUE, 1, 'PREFERRED_GENERIC', FALSE, FALSE, FALSE, NULL, NULL, NULL, '2024-01-01', 2024),
('H5678', '002', '000', '83367', '00071-0155-23', 'Lipitor 10mg', 'Atorvastatin Calcium', TRUE, 1, 'PREFERRED_GENERIC', FALSE, FALSE, FALSE, NULL, NULL, NULL, '2024-01-01', 2024),
('H5678', '002', '000', '11289', '00056-0174-70', 'Coumadin 5mg', 'Warfarin Sodium', TRUE, 2, 'GENERIC', FALSE, FALSE, TRUE, 'MONTHLY', 30, 30, '2024-01-01', 2024),
('H5678', '002', '000', '4441', '00777-3105-02', 'Prozac 20mg', 'Fluoxetine Hydrochloride', TRUE, 2, 'GENERIC', FALSE, FALSE, FALSE, NULL, NULL, NULL, '2024-01-01', 2024);

COMMIT;

-- =============================================================================
-- SECTION 3: LOINC LAB REFERENCE RANGES
-- Source: LOINC + NHANES population statistics
-- =============================================================================

BEGIN;

DELETE FROM loinc_lab_ranges WHERE source_version = 'LOINC-2024';

CREATE TABLE IF NOT EXISTS loinc_lab_ranges (
    id SERIAL PRIMARY KEY,
    loinc_code VARCHAR(20) NOT NULL,
    component VARCHAR(255) NOT NULL,
    property VARCHAR(50),
    time_aspect VARCHAR(20),
    system VARCHAR(100),
    scale_type VARCHAR(20),
    method_type VARCHAR(100),
    class VARCHAR(100),
    short_name VARCHAR(100),
    long_name TEXT,
    unit VARCHAR(50),
    low_normal NUMERIC(10,4),
    high_normal NUMERIC(10,4),
    critical_low NUMERIC(10,4),
    critical_high NUMERIC(10,4),
    age_group VARCHAR(50),
    sex VARCHAR(20),
    clinical_category VARCHAR(50),
    interpretation_guidance TEXT,
    delta_check_percent NUMERIC(5,2),
    delta_check_hours INTEGER,
    deprecated BOOLEAN DEFAULT FALSE,
    source_version VARCHAR(50) DEFAULT 'LOINC-2024',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(loinc_code, age_group, sex, source_version)
);

CREATE INDEX IF NOT EXISTS idx_loinc_code ON loinc_lab_ranges(loinc_code);
CREATE INDEX IF NOT EXISTS idx_loinc_category ON loinc_lab_ranges(clinical_category);
CREATE INDEX IF NOT EXISTS idx_loinc_component ON loinc_lab_ranges(component);

-- Core Electrolytes
INSERT INTO loinc_lab_ranges (loinc_code, component, property, system, scale_type, short_name, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance, delta_check_percent, delta_check_hours) VALUES
('2951-2', 'Sodium', 'MCnc', 'Ser/Plas', 'Qn', 'Sodium SerPl-mCnc', 'Sodium [Moles/volume] in Serum or Plasma', 'mmol/L', 136, 145, 120, 160, 'adult', 'all', 'electrolyte', 'Low sodium may indicate SIADH or diuretic use. High sodium indicates dehydration or diabetes insipidus.', 10, 24),
('2823-3', 'Potassium', 'MCnc', 'Ser/Plas', 'Qn', 'Potassium SerPl-mCnc', 'Potassium [Moles/volume] in Serum or Plasma', 'mmol/L', 3.5, 5.0, 2.5, 6.5, 'adult', 'all', 'electrolyte', 'Monitor for cardiac arrhythmias at extremes. Consider hemolysis artifact if elevated.', 20, 24),
('2075-0', 'Chloride', 'MCnc', 'Ser/Plas', 'Qn', 'Chloride SerPl-mCnc', 'Chloride [Moles/volume] in Serum or Plasma', 'mmol/L', 98, 106, 80, 120, 'adult', 'all', 'electrolyte', 'Interpret with sodium for acid-base status.', NULL, 24),
('1963-8', 'Bicarbonate', 'MCnc', 'Ser/Plas', 'Qn', 'HCO3 SerPl-mCnc', 'Bicarbonate [Moles/volume] in Serum or Plasma', 'mmol/L', 22, 29, 10, 40, 'adult', 'all', 'electrolyte', 'Low indicates metabolic acidosis. High indicates metabolic alkalosis.', NULL, 24);

-- Renal Panel (with sex-specific ranges)
INSERT INTO loinc_lab_ranges (loinc_code, component, property, system, scale_type, short_name, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance, delta_check_percent, delta_check_hours) VALUES
('2160-0', 'Creatinine', 'MCnc', 'Ser/Plas', 'Qn', 'Creat SerPl-mCnc', 'Creatinine [Mass/volume] in Serum or Plasma', 'mg/dL', 0.7, 1.3, 0.4, 10.0, 'adult', 'male', 'renal', 'Baseline for eGFR calculation. 50% rise in 48h suggests AKI per KDIGO.', 50, 48),
('2160-0', 'Creatinine', 'MCnc', 'Ser/Plas', 'Qn', 'Creat SerPl-mCnc', 'Creatinine [Mass/volume] in Serum or Plasma', 'mg/dL', 0.6, 1.1, 0.4, 10.0, 'adult', 'female', 'renal', 'Baseline for eGFR calculation. 50% rise in 48h suggests AKI per KDIGO.', 50, 48),
('3094-0', 'BUN', 'MCnc', 'Ser/Plas', 'Qn', 'BUN SerPl-mCnc', 'Urea nitrogen [Mass/volume] in Serum or Plasma', 'mg/dL', 7, 20, 2, 100, 'adult', 'all', 'renal', 'BUN:Cr ratio >20 suggests prerenal azotemia.', NULL, 24),
('33914-3', 'eGFR', 'ArVRat', 'Ser/Plas', 'Qn', 'eGFR CKD-EPI', 'Glomerular filtration rate/1.73 sq M.predicted by CKD-EPI', 'mL/min/1.73m2', 90, 999, 15, 999, 'adult', 'all', 'renal', 'Stage CKD: >90=G1 60-89=G2 45-59=G3a 30-44=G3b 15-29=G4 <15=G5.', NULL, 0);

-- Cardiac Markers
INSERT INTO loinc_lab_ranges (loinc_code, component, property, system, scale_type, short_name, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance, delta_check_percent, delta_check_hours) VALUES
('33762-6', 'NT-proBNP', 'MCnc', 'Ser/Plas', 'Qn', 'NT-proBNP SerPl-mCnc', 'Natriuretic peptide.B prohormone N-Terminal [Mass/volume] in Serum or Plasma', 'pg/mL', 0, 125, 0, 30000, 'adult', 'all', 'cardiac', 'Age-adjusted cutoffs: <50yo <450. 50-75yo <900. >75yo <1800. Rules out HF if normal.', 50, 24),
('10839-9', 'Troponin I', 'MCnc', 'Ser/Plas', 'Qn', 'Troponin I SerPl-mCnc', 'Troponin I.cardiac [Mass/volume] in Serum or Plasma', 'ng/mL', 0.00, 0.04, 0.00, 50.00, 'adult', 'all', 'cardiac', '99th percentile is cutoff for MI. Serial measurements q3-6h. Rise and fall pattern.', 100, 6),
('6598-7', 'Troponin T', 'MCnc', 'Ser/Plas', 'Qn', 'Troponin T SerPl-mCnc', 'Troponin T.cardiac [Mass/volume] in Serum or Plasma', 'ng/mL', 0.00, 0.01, 0.00, 10.00, 'adult', 'all', 'cardiac', 'High-sensitivity assay improves early detection. Elevated in CKD non-MI causes.', 100, 6);

-- Hematology
INSERT INTO loinc_lab_ranges (loinc_code, component, property, system, scale_type, short_name, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance, delta_check_percent, delta_check_hours) VALUES
('718-7', 'Hemoglobin', 'MCnc', 'Bld', 'Qn', 'Hgb Bld-mCnc', 'Hemoglobin [Mass/volume] in Blood', 'g/dL', 13.5, 17.5, 7.0, 20.0, 'adult', 'male', 'hematology', 'Anemia <13 men <12 women. Consider transfusion threshold 7-8 g/dL.', 25, 24),
('718-7', 'Hemoglobin', 'MCnc', 'Bld', 'Qn', 'Hgb Bld-mCnc', 'Hemoglobin [Mass/volume] in Blood', 'g/dL', 12.0, 16.0, 7.0, 20.0, 'adult', 'female', 'hematology', 'Anemia <13 men <12 women. Consider transfusion threshold 7-8 g/dL.', 25, 24),
('777-3', 'Platelet count', 'NCnc', 'Bld', 'Qn', 'Platelet # Bld', 'Platelets [#/volume] in Blood', '10*3/uL', 150, 400, 50, 1000, 'adult', 'all', 'hematology', '<100 thrombocytopenia. >50% drop in 5-10 days consider HIT if on heparin.', 50, 120),
('6690-2', 'WBC', 'NCnc', 'Bld', 'Qn', 'WBC # Bld', 'Leukocytes [#/volume] in Blood', '10*3/uL', 4.5, 11.0, 1.0, 30.0, 'adult', 'all', 'hematology', 'Leukocytosis >11. Leukopenia <4.5. Neutropenia ANC <1500 needs evaluation.', NULL, 24);

-- Coagulation
INSERT INTO loinc_lab_ranges (loinc_code, component, property, system, scale_type, short_name, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance, delta_check_percent, delta_check_hours) VALUES
('6301-6', 'INR', 'Rto', 'Bld', 'Qn', 'INR Bld', 'INR in Blood by Coagulation assay', NULL, 0.9, 1.1, 0.8, 6.0, 'adult', 'all', 'coagulation', 'Warfarin target 2-3 for most indications. 2.5-3.5 for mechanical valves.', 30, 24),
('5902-2', 'PT', 'Time', 'PPP', 'Qn', 'PT PPP', 'Prothrombin time (PT)', 'seconds', 11.0, 13.5, 9.0, 50.0, 'adult', 'all', 'coagulation', 'Monitors warfarin therapy. Elevated with liver disease or vitamin K deficiency.', NULL, 8),
('3173-2', 'aPTT', 'Time', 'PPP', 'Qn', 'aPTT PPP', 'aPTT in Platelet poor plasma by Coagulation assay', 'seconds', 25, 35, 20, 100, 'adult', 'all', 'coagulation', 'Monitors heparin. Prolonged with lupus anticoagulant or factor deficiencies.', NULL, 8);

-- Liver Panel
INSERT INTO loinc_lab_ranges (loinc_code, component, property, system, scale_type, short_name, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance, delta_check_percent, delta_check_hours) VALUES
('1742-6', 'ALT', 'CCnc', 'Ser/Plas', 'Qn', 'ALT SerPl-cCnc', 'Alanine aminotransferase [Enzymatic activity/volume] in Serum or Plasma', 'U/L', 7, 56, 5, 1000, 'adult', 'all', 'liver', 'More liver-specific than AST. >3x ULN significant. >10x ULN acute hepatitis.', NULL, 24),
('1920-8', 'AST', 'CCnc', 'Ser/Plas', 'Qn', 'AST SerPl-cCnc', 'Aspartate aminotransferase [Enzymatic activity/volume] in Serum or Plasma', 'U/L', 10, 40, 5, 1000, 'adult', 'all', 'liver', 'Also elevated in muscle injury MI. AST:ALT >2 suggests alcoholic liver disease.', NULL, 24),
('1975-2', 'Total Bilirubin', 'MCnc', 'Ser/Plas', 'Qn', 'Bilirub SerPl-mCnc', 'Bilirubin.total [Mass/volume] in Serum or Plasma', 'mg/dL', 0.1, 1.2, 0.1, 15.0, 'adult', 'all', 'liver', 'Jaundice visible >2.5. Conjugated vs unconjugated guides differential.', NULL, 24),
('1751-7', 'Albumin', 'MCnc', 'Ser/Plas', 'Qn', 'Albumin SerPl-mCnc', 'Albumin [Mass/volume] in Serum or Plasma', 'g/dL', 3.5, 5.0, 1.5, 6.0, 'adult', 'all', 'liver', 'Low in malnutrition liver disease nephrotic syndrome. Half-life ~21 days.', NULL, 24);

-- Metabolic
INSERT INTO loinc_lab_ranges (loinc_code, component, property, system, scale_type, short_name, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance, delta_check_percent, delta_check_hours) VALUES
('2345-7', 'Glucose', 'MCnc', 'Ser/Plas', 'Qn', 'Glucose SerPl-mCnc', 'Glucose [Mass/volume] in Serum or Plasma', 'mg/dL', 70, 100, 40, 500, 'adult', 'all', 'metabolic', 'Fasting <100 normal. 100-125 prediabetes. >=126 diabetes (confirm with repeat).', 25, 4),
('4548-4', 'HbA1c', 'MFr', 'Bld', 'Qn', 'HgbA1c MFr Bld', 'Hemoglobin A1c/Hemoglobin.total in Blood', '%', 4.0, 5.6, 3.0, 15.0, 'adult', 'all', 'metabolic', '<5.7% normal. 5.7-6.4% prediabetes. >=6.5% diabetes. Target <7% for most diabetics.', NULL, 0);

-- Thyroid
INSERT INTO loinc_lab_ranges (loinc_code, component, property, system, scale_type, short_name, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance, delta_check_percent, delta_check_hours) VALUES
('3016-3', 'TSH', 'ACnc', 'Ser/Plas', 'Qn', 'TSH SerPl-aCnc', 'Thyrotropin [Units/volume] in Serum or Plasma', 'mIU/L', 0.4, 4.0, 0.01, 100.0, 'adult', 'all', 'endocrine', 'Low TSH + high T4 = hyperthyroid. High TSH + low T4 = hypothyroid. Screen first.', NULL, 0),
('3026-2', 'Free T4', 'MCnc', 'Ser/Plas', 'Qn', 'T4 Free SerPl-mCnc', 'Thyroxine (T4) free [Mass/volume] in Serum or Plasma', 'ng/dL', 0.8, 1.8, 0.4, 7.0, 'adult', 'all', 'endocrine', 'Free T4 preferred over total T4. Interpret with TSH.', NULL, 0);

COMMIT;

-- =============================================================================
-- SECTION 4: METADATA AND VERSIONING
-- =============================================================================

BEGIN;

CREATE TABLE IF NOT EXISTS phase1_ingestion_metadata (
    id SERIAL PRIMARY KEY,
    source_name VARCHAR(50) NOT NULL,
    source_version VARCHAR(50) NOT NULL,
    records_loaded INTEGER,
    load_timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    sha256_checksum VARCHAR(64),
    source_url TEXT,
    notes TEXT,
    UNIQUE(source_name, source_version)
);

INSERT INTO phase1_ingestion_metadata (source_name, source_version, records_loaded, source_url, notes) VALUES
('ONC_DDI', 'ONC-2024-Q4', 50, 'https://www.healthit.gov/topic/safety/high-priority-drug-drug-interaction', 'High-priority DDI pairs with bidirectional entries'),
('CMS_FORMULARY', 'CMS-2024-PD', 29, 'https://data.cms.gov/provider-data/topics/prescription-drug-plan', 'Medicare Part D formulary entries for sample plans'),
('LOINC_LABS', 'LOINC-2024', 25, 'https://loinc.org/', 'Lab reference ranges with KDIGO delta checks for AKI/HIT detection')
ON CONFLICT (source_name, source_version) DO UPDATE SET
    records_loaded = EXCLUDED.records_loaded,
    load_timestamp = CURRENT_TIMESTAMP;

COMMIT;

-- =============================================================================
-- VERIFICATION QUERIES
-- =============================================================================

-- Verify ONC DDI counts
SELECT 'ONC DDI Total' AS metric, COUNT(*) AS count FROM onc_drug_interactions WHERE source_version = 'ONC-2024-Q4';
SELECT 'ONC DDI Contraindicated' AS metric, COUNT(*) AS count FROM onc_drug_interactions WHERE severity = 'CONTRAINDICATED' AND source_version = 'ONC-2024-Q4';
SELECT 'ONC DDI High Severity' AS metric, COUNT(*) AS count FROM onc_drug_interactions WHERE severity = 'HIGH' AND source_version = 'ONC-2024-Q4';

-- Verify CMS Formulary counts
SELECT 'CMS Formulary Total' AS metric, COUNT(*) AS count FROM cms_formulary_entries WHERE effective_year = 2024;
SELECT 'CMS Unique RxCUIs' AS metric, COUNT(DISTINCT rxcui) AS count FROM cms_formulary_entries WHERE effective_year = 2024;
SELECT 'CMS Plans' AS metric, COUNT(DISTINCT contract_id || plan_id) AS count FROM cms_formulary_entries WHERE effective_year = 2024;

-- Verify LOINC Labs counts
SELECT 'LOINC Labs Total' AS metric, COUNT(*) AS count FROM loinc_lab_ranges WHERE source_version = 'LOINC-2024';
SELECT 'LOINC Unique Codes' AS metric, COUNT(DISTINCT loinc_code) AS count FROM loinc_lab_ranges WHERE source_version = 'LOINC-2024';
SELECT 'LOINC By Category' AS metric, clinical_category, COUNT(*) AS count FROM loinc_lab_ranges WHERE source_version = 'LOINC-2024' GROUP BY clinical_category ORDER BY count DESC;

-- Print summary
SELECT '=== GOLDEN STATE PHASE 1 VERIFICATION COMPLETE ===' AS status;
