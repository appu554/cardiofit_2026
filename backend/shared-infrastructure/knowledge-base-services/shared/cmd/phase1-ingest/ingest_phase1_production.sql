-- =============================================================================
-- PHASE 1 PRODUCTION DATA INGESTION
-- Reference: KB1_DATA_SOURCE_INJECTION_IMPLEMENTATION_PLAN.md
--
-- This script loads Phase 1 data following the Canonical Fact Store architecture:
-- - KB-5: ONC High-Priority DDI (interaction_matrix)
-- - KB-6: CMS Medicare Formulary (formulary_coverage)
-- - KB-16: LOINC Lab Reference Ranges (lab_reference_ranges)
--
-- Prerequisites:
--   1. CSV files must be copied to container at /tmp/data/
--   2. Run with: psql -U kb_admin -d canonical_facts -f /tmp/ingest_phase1_production.sql
-- =============================================================================

BEGIN;

-- =============================================================================
-- CONFIGURATION
-- =============================================================================
\set source_version_ddi 'ONC-2024-Q4'
\set source_version_formulary 'CMS-2024'
\set source_version_labs 'LOINC-2024'

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
\copy staging_onc_ddi FROM '/tmp/data/onc_ddi.csv' WITH (FORMAT csv, HEADER true);

-- Insert into interaction_matrix (bidirectional: insert both directions)
-- Direction 1: Drug1 -> Drug2
INSERT INTO interaction_matrix (
    drug1_rxcui,
    drug1_name,
    drug2_rxcui,
    drug2_name,
    severity,
    clinical_effect,
    management,
    documentation,
    evidence_level,
    clinical_source,
    source_pair_id,
    source_dataset,
    source_version,
    is_bidirectional,
    precipitant_rxcui,
    object_rxcui,
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
    LEFT(documentation, 50),  -- Truncate to fit schema
    evidence_level,
    clinical_source,
    onc_pair_id,
    'ONC_HIGH_PRIORITY',
    'ONC-2024-Q4',
    true,
    drug1_rxcui,
    drug2_rxcui,
    NOW()
FROM staging_onc_ddi
ON CONFLICT (drug1_rxcui, drug2_rxcui, source_dataset) DO UPDATE SET
    severity = EXCLUDED.severity,
    clinical_effect = EXCLUDED.clinical_effect,
    management = EXCLUDED.management,
    evidence_level = EXCLUDED.evidence_level,
    last_updated = NOW();

-- Direction 2: Drug2 -> Drug1 (reverse pair for bidirectional lookup)
INSERT INTO interaction_matrix (
    drug1_rxcui,
    drug1_name,
    drug2_rxcui,
    drug2_name,
    severity,
    clinical_effect,
    management,
    documentation,
    evidence_level,
    clinical_source,
    source_pair_id,
    source_dataset,
    source_version,
    is_bidirectional,
    precipitant_rxcui,
    object_rxcui,
    created_at
)
SELECT
    drug2_rxcui,      -- Reversed
    drug2_name,
    drug1_rxcui,      -- Reversed
    drug1_name,
    severity,
    clinical_effect,
    management,
    LEFT(documentation, 50),  -- Truncate to fit schema
    evidence_level,
    clinical_source,
    onc_pair_id || '-REV',
    'ONC_HIGH_PRIORITY',
    'ONC-2024-Q4',
    true,
    drug2_rxcui,
    drug1_rxcui,
    NOW()
FROM staging_onc_ddi
ON CONFLICT (drug1_rxcui, drug2_rxcui, source_dataset) DO UPDATE SET
    severity = EXCLUDED.severity,
    clinical_effect = EXCLUDED.clinical_effect,
    management = EXCLUDED.management,
    evidence_level = EXCLUDED.evidence_level,
    last_updated = NOW();

-- Record DDI count
DO $$
DECLARE
    ddi_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO ddi_count FROM interaction_matrix WHERE source_dataset = 'ONC_HIGH_PRIORITY';
    RAISE NOTICE 'KB-5 DDI: Loaded % interaction facts (bidirectional pairs)', ddi_count;
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
    tier_level_code VARCHAR(50),
    quantity_limit VARCHAR(10),
    quantity_limit_amount VARCHAR(20),
    quantity_limit_days VARCHAR(20),
    prior_auth VARCHAR(10),
    step_therapy VARCHAR(10),
    coverage_status VARCHAR(50),
    effective_year INTEGER
);

-- Import CSV
\copy staging_cms_formulary FROM '/tmp/data/cms_formulary.csv' WITH (FORMAT csv, HEADER true);

-- Insert into formulary_coverage (filter NOT_COVERED per governance)
INSERT INTO formulary_coverage (
    contract_id,
    plan_id,
    rxcui,
    ndc,
    drug_name,
    tier_level_code,
    tier,
    quantity_limit,
    quantity_limit_amt,
    quantity_limit_days,
    prior_auth,
    step_therapy,
    on_formulary,
    effective_year,
    source_version,
    created_at
)
SELECT
    contract_id,
    plan_id,
    rxcui,
    NULLIF(ndc, ''),
    drug_name,
    tier_level_code,
    CASE
        WHEN tier_level_code ~ '^\d+$' THEN tier_level_code::INTEGER
        ELSE NULL
    END,
    COALESCE(quantity_limit, '') IN ('Y', 'TRUE', 'true', '1'),
    CASE
        WHEN quantity_limit_amount ~ '^\d+$' THEN quantity_limit_amount::INTEGER
        ELSE NULL
    END,
    CASE
        WHEN quantity_limit_days ~ '^\d+$' THEN quantity_limit_days::INTEGER
        ELSE NULL
    END,
    COALESCE(prior_auth, '') IN ('Y', 'TRUE', 'true', '1'),
    COALESCE(step_therapy, '') IN ('Y', 'TRUE', 'true', '1'),
    coverage_status = 'COVERED',
    effective_year,
    'CMS-2024',
    NOW()
FROM staging_cms_formulary
WHERE coverage_status != 'NOT_COVERED'  -- Filter per FILTERED_RECORDS_AUDIT.md
ON CONFLICT (contract_id, plan_id, rxcui, ndc, effective_year) DO UPDATE SET
    tier_level_code = EXCLUDED.tier_level_code,
    tier = EXCLUDED.tier,
    prior_auth = EXCLUDED.prior_auth,
    step_therapy = EXCLUDED.step_therapy,
    on_formulary = EXCLUDED.on_formulary,
    created_at = NOW();

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
    component VARCHAR(255),
    property VARCHAR(50),
    time_aspect VARCHAR(20),
    system VARCHAR(100),
    scale_type VARCHAR(20),
    method_type VARCHAR(100),
    class VARCHAR(100),
    short_name VARCHAR(100),
    long_name TEXT,
    unit VARCHAR(50),
    low_normal NUMERIC,
    high_normal NUMERIC,
    critical_low NUMERIC,
    critical_high NUMERIC,
    age_group VARCHAR(50),
    sex VARCHAR(20),
    clinical_category VARCHAR(50),
    interpretation_guidance TEXT,
    delta_check_percent NUMERIC,
    delta_check_hours INTEGER,
    deprecated VARCHAR(5)
);

-- Import CSV
\copy staging_loinc_labs FROM '/tmp/data/loinc_labs.csv' WITH (FORMAT csv, HEADER true);

-- Insert into lab_reference_ranges
INSERT INTO lab_reference_ranges (
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
    deprecated,
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
    COALESCE(deprecated, 'N') = 'Y',
    'LOINC-2024',
    NOW()
FROM staging_loinc_labs
WHERE COALESCE(deprecated, 'N') != 'Y'  -- Skip deprecated LOINC codes
ON CONFLICT (loinc_code, age_group, sex, source_version) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    interpretation_guidance = EXCLUDED.interpretation_guidance,
    created_at = NOW();

-- Record lab count
DO $$
DECLARE
    lab_count INTEGER;
    deprecated_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO lab_count FROM lab_reference_ranges WHERE source_version = 'LOINC-2024';
    SELECT COUNT(*) INTO deprecated_count FROM staging_loinc_labs WHERE COALESCE(deprecated, 'N') = 'Y';
    RAISE NOTICE 'KB-16 Labs: Loaded % reference ranges (% deprecated skipped)', lab_count, deprecated_count;
END $$;

-- =============================================================================
-- INGESTION METADATA
-- =============================================================================

-- Record ingestion for KB-5 DDI
INSERT INTO ingestion_metadata (
    source_name,
    source_version,
    records_loaded,
    records_skipped,
    records_failed,
    load_timestamp,
    notes
)
VALUES (
    'KB-5-ONC-DDI',
    'ONC-2024-Q4',
    (SELECT COUNT(*) FROM interaction_matrix WHERE source_dataset = 'ONC_HIGH_PRIORITY'),
    0,
    0,
    NOW(),
    'Phase 1 ETL: ONC High-Priority Drug-Drug Interactions (bidirectional pairs)'
);

-- Record ingestion for KB-6 Formulary
INSERT INTO ingestion_metadata (
    source_name,
    source_version,
    records_loaded,
    records_skipped,
    records_failed,
    load_timestamp,
    notes
)
VALUES (
    'KB-6-CMS-Formulary',
    'CMS-2024',
    (SELECT COUNT(*) FROM formulary_coverage WHERE source_version = 'CMS-2024'),
    (SELECT COUNT(*) FROM staging_cms_formulary WHERE coverage_status = 'NOT_COVERED'),
    0,
    NOW(),
    'Phase 1 ETL: CMS Medicare Part D Formulary (NOT_COVERED filtered per governance)'
);

-- Record ingestion for KB-16 Labs
INSERT INTO ingestion_metadata (
    source_name,
    source_version,
    records_loaded,
    records_skipped,
    records_failed,
    load_timestamp,
    notes
)
VALUES (
    'KB-16-LOINC-Labs',
    'LOINC-2024',
    (SELECT COUNT(*) FROM lab_reference_ranges WHERE source_version = 'LOINC-2024'),
    (SELECT COUNT(*) FROM staging_loinc_labs WHERE COALESCE(deprecated, 'N') = 'Y'),
    0,
    NOW(),
    'Phase 1 ETL: LOINC Lab Reference Ranges (deprecated codes skipped)'
);

COMMIT;

-- =============================================================================
-- VERIFICATION (outside transaction for immediate visibility)
-- =============================================================================

-- Summary counts
SELECT 'PHASE 1 INGESTION COMPLETE' as status;

SELECT
    'KB-5 DDI' as knowledge_base,
    COUNT(*) as record_count,
    'interaction_matrix' as table_name
FROM interaction_matrix
WHERE source_dataset = 'ONC_HIGH_PRIORITY'

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

-- Check views exist and are queryable
SELECT COUNT(*) as kb5_interactions_view FROM kb5_interactions;
SELECT COUNT(*) as kb6_formulary_view FROM kb6_formulary;
SELECT COUNT(*) as kb16_lab_ranges_view FROM kb16_lab_ranges;

-- =============================================================================
-- POST-INGESTION: Record schema version
-- =============================================================================

SELECT record_schema_version(
    5,
    '005_phase1_data_ingestion'::VARCHAR(255),
    current_user::VARCHAR(255),
    'development'::VARCHAR(50),
    NULL::VARCHAR(40),
    'Phase 1 data ingestion: DDI pairs, formulary entries, lab ranges'::TEXT
);

SELECT 'Phase 1 Production Data Ingestion Complete' as final_status;
