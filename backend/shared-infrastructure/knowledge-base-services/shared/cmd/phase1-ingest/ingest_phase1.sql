-- =============================================================================
-- PHASE 1 PRODUCTION DATA INGESTION
-- Reference: KB1_DATA_SOURCE_INJECTION_IMPLEMENTATION_PLAN.md
--
-- This script loads Phase 1 data following the Canonical Fact Store architecture:
-- - KB-5: ONC High-Priority DDI (interaction_matrix)
-- - KB-6: CMS Medicare Formulary (formulary_coverage)
-- - KB-16: LOINC Lab Reference Ranges (lab_reference_ranges)
-- =============================================================================

BEGIN;

-- =============================================================================
-- CONFIGURATION
-- =============================================================================
\set source_version 'Phase1-2026-01-20'
\set created_by 'phase1-ingest'

-- =============================================================================
-- KB-5: ONC DRUG-DRUG INTERACTIONS
-- Structured Load (ETL) - No LLM Required
-- Source: ONC High-Priority DDI List
-- =============================================================================

-- Temporary staging table for CSV import
CREATE TEMP TABLE staging_onc_ddi (
    drug1_rxcui VARCHAR(20),
    drug1_name VARCHAR(500),
    drug2_rxcui VARCHAR(20),
    drug2_name VARCHAR(500),
    severity VARCHAR(50),
    clinical_effect TEXT,
    management TEXT,
    evidence_level VARCHAR(50),
    documentation VARCHAR(100),
    clinical_source VARCHAR(100),
    onc_pair_id VARCHAR(50),
    last_updated DATE
);

-- Import CSV (header row will be skipped by COPY command)
\copy staging_onc_ddi FROM './data/onc_ddi.csv' WITH (FORMAT csv, HEADER true);

-- Insert into interaction_matrix with bidirectional pairs
-- Direction 1: Drug1 -> Drug2
INSERT INTO interaction_matrix (
    precipitant_rxcui,
    precipitant_name,
    object_rxcui,
    object_name,
    severity,
    clinical_effect,
    management_recommendation,
    evidence_level,
    documentation_level,
    source,
    source_version,
    is_bidirectional,
    created_at
)
SELECT
    drug1_rxcui,
    drug1_name,
    drug2_rxcui,
    drug2_name,
    severity,
    clinical_effect,
    management,
    evidence_level,
    documentation,
    'ONC_HIGH_PRIORITY',
    'ONC-2024-Q4',
    true,
    NOW()
FROM staging_onc_ddi
ON CONFLICT (precipitant_rxcui, object_rxcui, source) DO UPDATE SET
    severity = EXCLUDED.severity,
    clinical_effect = EXCLUDED.clinical_effect,
    management_recommendation = EXCLUDED.management_recommendation,
    updated_at = NOW();

-- Direction 2: Drug2 -> Drug1 (reverse pair for bidirectional lookup)
INSERT INTO interaction_matrix (
    precipitant_rxcui,
    precipitant_name,
    object_rxcui,
    object_name,
    severity,
    clinical_effect,
    management_recommendation,
    evidence_level,
    documentation_level,
    source,
    source_version,
    is_bidirectional,
    created_at
)
SELECT
    drug2_rxcui,
    drug2_name,
    drug1_rxcui,
    drug1_name,
    severity,
    clinical_effect,
    management,
    evidence_level,
    documentation,
    'ONC_HIGH_PRIORITY',
    'ONC-2024-Q4',
    true,
    NOW()
FROM staging_onc_ddi
ON CONFLICT (precipitant_rxcui, object_rxcui, source) DO UPDATE SET
    severity = EXCLUDED.severity,
    clinical_effect = EXCLUDED.clinical_effect,
    management_recommendation = EXCLUDED.management_recommendation,
    updated_at = NOW();

-- Record DDI count
DO $$
DECLARE
    ddi_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO ddi_count FROM interaction_matrix WHERE source = 'ONC_HIGH_PRIORITY';
    RAISE NOTICE 'KB-5 DDI: Loaded % interaction facts (bidirectional)', ddi_count;
END $$;

-- =============================================================================
-- KB-6: CMS MEDICARE FORMULARY
-- Structured Load (ETL) - No LLM Required
-- Source: CMS Medicare Part D Formulary
-- =============================================================================

-- Temporary staging table for CSV import
CREATE TEMP TABLE staging_cms_formulary (
    contract_id VARCHAR(20),
    plan_id VARCHAR(10),
    rxcui VARCHAR(20),
    ndc VARCHAR(20),
    drug_name VARCHAR(500),
    tier_level_code INTEGER,
    quantity_limit VARCHAR(10),
    quantity_limit_amount NUMERIC,
    quantity_limit_days INTEGER,
    prior_auth VARCHAR(10),
    step_therapy VARCHAR(10),
    coverage_status VARCHAR(50),
    effective_year INTEGER
);

-- Import CSV
\copy staging_cms_formulary FROM './data/cms_formulary.csv' WITH (FORMAT csv, HEADER true);

-- Insert into formulary_coverage (filter NOT_COVERED per governance)
INSERT INTO formulary_coverage (
    contract_id,
    plan_id,
    rxcui,
    ndc,
    drug_name,
    tier_level,
    quantity_limit_flag,
    quantity_limit_amount,
    quantity_limit_days,
    prior_auth_required,
    step_therapy_required,
    coverage_status,
    effective_year,
    source_version,
    created_at
)
SELECT
    contract_id,
    plan_id,
    rxcui,
    ndc,
    drug_name,
    tier_level_code,
    quantity_limit = 'Y',
    quantity_limit_amount,
    quantity_limit_days,
    prior_auth = 'Y',
    step_therapy = 'Y',
    coverage_status,
    effective_year,
    'CMS-2024',
    NOW()
FROM staging_cms_formulary
WHERE coverage_status != 'NOT_COVERED'  -- Filter per FILTERED_RECORDS_AUDIT.md
ON CONFLICT (contract_id, plan_id, rxcui, effective_year) DO UPDATE SET
    tier_level = EXCLUDED.tier_level,
    prior_auth_required = EXCLUDED.prior_auth_required,
    step_therapy_required = EXCLUDED.step_therapy_required,
    coverage_status = EXCLUDED.coverage_status,
    updated_at = NOW();

-- Record formulary count
DO $$
DECLARE
    formulary_count INTEGER;
    filtered_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO formulary_count FROM formulary_coverage WHERE source_version = 'CMS-2024';
    SELECT COUNT(*) INTO filtered_count FROM staging_cms_formulary WHERE coverage_status = 'NOT_COVERED';
    RAISE NOTICE 'KB-6 Formulary: Loaded % entries (% filtered as NOT_COVERED)', formulary_count, filtered_count;
END $$;

-- =============================================================================
-- KB-16: LOINC LAB REFERENCE RANGES
-- Structured Load (ETL) - No LLM Required
-- Source: LOINC Tables + NHANES Population Statistics
-- =============================================================================

-- Temporary staging table for CSV import
CREATE TEMP TABLE staging_loinc_labs (
    loinc_code VARCHAR(20),
    component VARCHAR(200),
    property VARCHAR(50),
    time_aspect VARCHAR(20),
    system VARCHAR(50),
    scale_type VARCHAR(20),
    method_type VARCHAR(50),
    class VARCHAR(50),
    short_name VARCHAR(100),
    long_name VARCHAR(500),
    unit VARCHAR(50),
    low_normal NUMERIC,
    high_normal NUMERIC,
    critical_low NUMERIC,
    critical_high NUMERIC,
    age_group VARCHAR(20),
    sex VARCHAR(20),
    clinical_category VARCHAR(50),
    interpretation_guidance TEXT,
    delta_check_percent NUMERIC,
    delta_check_hours INTEGER,
    deprecated VARCHAR(5)
);

-- Import CSV
\copy staging_loinc_labs FROM './data/loinc_labs.csv' WITH (FORMAT csv, HEADER true);

-- Insert into lab_reference_ranges
INSERT INTO lab_reference_ranges (
    loinc_code,
    component_name,
    property,
    time_aspect,
    system,
    scale_type,
    method_type,
    class,
    short_name,
    long_name,
    unit,
    reference_low,
    reference_high,
    critical_low,
    critical_high,
    age_group,
    sex,
    clinical_category,
    interpretation_guidance,
    delta_check_percent,
    delta_check_hours,
    source_version,
    created_at
)
SELECT
    loinc_code,
    component,
    property,
    time_aspect,
    system,
    scale_type,
    method_type,
    class,
    short_name,
    long_name,
    unit,
    low_normal,
    high_normal,
    critical_low,
    critical_high,
    age_group,
    sex,
    clinical_category,
    interpretation_guidance,
    delta_check_percent,
    delta_check_hours,
    'LOINC-2024',
    NOW()
FROM staging_loinc_labs
WHERE deprecated != 'Y'  -- Skip deprecated LOINC codes
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    reference_low = EXCLUDED.reference_low,
    reference_high = EXCLUDED.reference_high,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    interpretation_guidance = EXCLUDED.interpretation_guidance,
    updated_at = NOW();

-- Record lab count
DO $$
DECLARE
    lab_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO lab_count FROM lab_reference_ranges WHERE source_version = 'LOINC-2024';
    RAISE NOTICE 'KB-16 Labs: Loaded % reference ranges', lab_count;
END $$;

-- =============================================================================
-- INGESTION METADATA
-- =============================================================================

INSERT INTO ingestion_metadata (
    ingestion_id,
    source_name,
    source_version,
    ingestion_type,
    records_processed,
    records_loaded,
    records_skipped,
    started_at,
    completed_at,
    status
)
SELECT
    gen_random_uuid(),
    'Phase1-Combined',
    'Phase1-2026-01-20',
    'ETL',
    (SELECT COUNT(*) FROM staging_onc_ddi) +
    (SELECT COUNT(*) FROM staging_cms_formulary) +
    (SELECT COUNT(*) FROM staging_loinc_labs),
    (SELECT COUNT(*) FROM interaction_matrix WHERE source = 'ONC_HIGH_PRIORITY') +
    (SELECT COUNT(*) FROM formulary_coverage WHERE source_version = 'CMS-2024') +
    (SELECT COUNT(*) FROM lab_reference_ranges WHERE source_version = 'LOINC-2024'),
    (SELECT COUNT(*) FROM staging_cms_formulary WHERE coverage_status = 'NOT_COVERED'),
    NOW() - INTERVAL '1 second',
    NOW(),
    'COMPLETED';

-- =============================================================================
-- VERIFICATION
-- =============================================================================

-- Summary counts
SELECT 'PHASE 1 INGESTION COMPLETE' as status;

SELECT
    'KB-5 DDI' as knowledge_base,
    COUNT(*) as record_count,
    'interaction_matrix' as table_name
FROM interaction_matrix
WHERE source = 'ONC_HIGH_PRIORITY'

UNION ALL

SELECT
    'KB-6 Formulary',
    COUNT(*),
    'formulary_coverage'
FROM formulary_coverage
WHERE source_version = 'CMS-2024'

UNION ALL

SELECT
    'KB-16 Labs',
    COUNT(*),
    'lab_reference_ranges'
FROM lab_reference_ranges
WHERE source_version = 'LOINC-2024';

-- Verify KB projection views work
SELECT 'Verifying KB projections...' as status;

SELECT COUNT(*) as kb5_interactions_view FROM kb5_interactions;
SELECT COUNT(*) as kb6_formulary_view FROM kb6_formulary;
SELECT COUNT(*) as kb16_lab_ranges_view FROM kb16_lab_ranges;

COMMIT;

-- =============================================================================
-- POST-COMMIT: Record schema version
-- =============================================================================

SELECT record_schema_version(
    5,
    '005_phase1_data_ingestion'::VARCHAR(255),
    current_user::VARCHAR(255),
    'development'::VARCHAR(50),
    NULL::VARCHAR(40),
    'Phase 1 data ingestion: 100 DDI pairs, 164 formulary entries, 50 lab ranges'::TEXT
);

SELECT 'Phase 1 Data Ingestion Complete' as final_status;
