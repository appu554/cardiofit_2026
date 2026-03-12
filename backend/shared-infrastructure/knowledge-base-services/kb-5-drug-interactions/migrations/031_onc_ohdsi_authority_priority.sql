-- =============================================================================
-- Migration 031: ONC > OHDSI Authority Priority in Query Layer
-- =============================================================================
-- This migration updates the check_constitutional_ddi() function to enforce
-- authority-based ordering, ensuring ONC Constitutional rules are returned
-- before OHDSI-derived rules when checking drug interactions.
--
-- Semantic Contract: Authority > Severity > Deterministic ID
--   1. Authority precedence (federal/regulatory first)
--   2. Clinical severity (within same authority)
--   3. Rule ID (deterministic tie-break)
--
-- This aligns the query layer with:
--   - GAP 2 Conflict Resolution (shared/conflicts/resolver.go)
--   - Context Router v2.0 Execution Contract
--
-- Reference: KB1_DATA_SOURCE_INJECTION_IMPLEMENTATION_PLAN.md - Phase 1 Item
-- =============================================================================

BEGIN;

-- =============================================================================
-- 1. Source Authority Reference Table (for documentation and future use)
-- =============================================================================

CREATE TABLE IF NOT EXISTS source_authority_ranking (
    authority_code VARCHAR(100) PRIMARY KEY,
    authority_rank INTEGER NOT NULL,           -- Lower = higher priority (1 = highest)
    authority_tier VARCHAR(50) NOT NULL,       -- REGULATORY, GUIDELINE, CURATED, DERIVED
    description TEXT,
    jurisdiction VARCHAR(50),                  -- US, EU, AU, IN, GLOBAL
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert authority rankings (aligned with shared/conflicts/resolver.go)
INSERT INTO source_authority_ranking (authority_code, authority_rank, authority_tier, description, jurisdiction)
VALUES
    -- Tier 1: Constitutional / Regulatory (1-10)
    ('ONC-Phansalkar-2012', 1, 'REGULATORY', 'ONC High-Priority Drug Interactions - 25 Constitutional Rules', 'US'),
    ('FDA_SPL', 2, 'REGULATORY', 'FDA Structured Product Labeling', 'US'),
    ('FDA_FAERS', 3, 'REGULATORY', 'FDA Adverse Event Reporting System', 'US'),
    ('EMA_EPAR', 4, 'REGULATORY', 'European Medicines Agency EPAR', 'EU'),
    ('Post-ONC-Critical', 5, 'REGULATORY', 'Post-ONC Critical Additions (curated extensions)', 'US'),

    -- Tier 2: Clinical Guidelines (11-20)
    ('ONC-Derived', 11, 'GUIDELINE', 'ONC-derived rules with lower confidence', 'US'),
    ('KDIGO', 12, 'GUIDELINE', 'Kidney Disease: Improving Global Outcomes', 'GLOBAL'),
    ('AHA', 13, 'GUIDELINE', 'American Heart Association Guidelines', 'US'),
    ('ACC', 14, 'GUIDELINE', 'American College of Cardiology Guidelines', 'US'),

    -- Tier 3: Curated Medical Databases (21-30)
    ('OHDSI_ATHENA', 21, 'CURATED', 'OHDSI Athena Vocabulary (multi-source)', 'GLOBAL'),
    ('DRUGBANK', 22, 'CURATED', 'DrugBank Database', 'GLOBAL'),
    ('MEDRT', 23, 'CURATED', 'MED-RT Medication Reference Terminology', 'US'),

    -- Tier 4: Commercial / Research (31+)
    ('CLINICAL', 31, 'DERIVED', 'Institution-specific clinical rules', 'LOCAL'),
    ('RESEARCH', 41, 'DERIVED', 'Research/literature-derived rules', 'GLOBAL')
ON CONFLICT (authority_code) DO UPDATE
SET authority_rank = EXCLUDED.authority_rank,
    authority_tier = EXCLUDED.authority_tier,
    description = EXCLUDED.description;

-- =============================================================================
-- 2. Updated DDI Check Function with Authority Priority
-- =============================================================================
-- CRITICAL: This function now orders results by:
--   1. Authority (ONC > FDA > OHDSI)
--   2. Severity (CRITICAL > HIGH > WARNING > MODERATE)
--   3. Rule ID (deterministic)
-- =============================================================================

CREATE OR REPLACE FUNCTION check_constitutional_ddi(
    drug_concept_ids BIGINT[]
)
RETURNS TABLE (
    rule_id INTEGER,
    risk_level VARCHAR(20),
    alert_message TEXT,
    drug_a_name VARCHAR(500),
    drug_b_name VARCHAR(500),
    context_loinc_id VARCHAR(20),
    context_threshold DECIMAL(10,2),
    context_operator VARCHAR(5),
    context_required BOOLEAN,
    rule_authority VARCHAR(100),
    authority_rank INTEGER  -- NEW: Exposed for audit trail
) AS $$
BEGIN
    RETURN QUERY
    SELECT DISTINCT
        v.rule_id,
        v.risk_level,
        v.alert_message::TEXT,
        v.trigger_drug_name,
        v.target_drug_name,
        v.context_loinc_id,
        v.context_threshold_val,
        v.context_logic_operator,
        v.context_required,
        v.rule_authority,
        COALESCE(sar.authority_rank, 99)::INTEGER AS authority_rank
    FROM v_active_ddi_definitions v
    LEFT JOIN source_authority_ranking sar
        ON v.rule_authority = sar.authority_code
    WHERE v.trigger_drug_id = ANY(drug_concept_ids)
      AND v.target_drug_id = ANY(drug_concept_ids)
      AND v.trigger_drug_id != v.target_drug_id
    ORDER BY
        -- 1️⃣ Authority precedence (semantic law - ONC > OHDSI)
        COALESCE(sar.authority_rank, 99),

        -- 2️⃣ Clinical severity (within same authority)
        CASE v.risk_level
            WHEN 'CRITICAL' THEN 1
            WHEN 'HIGH' THEN 2
            WHEN 'WARNING' THEN 3
            ELSE 4
        END,

        -- 3️⃣ Deterministic tie-break
        v.rule_id;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- 3. Helper Function: Get Authority Rank for a Rule
-- =============================================================================

CREATE OR REPLACE FUNCTION get_authority_rank(p_authority VARCHAR(100))
RETURNS INTEGER AS $$
DECLARE
    v_rank INTEGER;
BEGIN
    SELECT authority_rank INTO v_rank
    FROM source_authority_ranking
    WHERE authority_code = p_authority;

    RETURN COALESCE(v_rank, 99);  -- Unknown authorities get lowest priority
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- =============================================================================
-- 4. Add comment documenting the semantic contract
-- =============================================================================

COMMENT ON FUNCTION check_constitutional_ddi(BIGINT[]) IS
'DDI check function with ONC > OHDSI authority priority.

SEMANTIC CONTRACT (Phase 1 v1.0.0):
  Results are ordered by:
    1. Authority rank (ONC=1 > FDA=2 > OHDSI=21 > others)
    2. Risk level (CRITICAL > HIGH > WARNING > MODERATE)
    3. Rule ID (deterministic tie-break)

This ensures federally normative sources (ONC Constitutional Rules)
always appear before vocabulary-derived sources (OHDSI Athena).

Aligned with:
  - GAP 2 Conflict Resolution (shared/conflicts/resolver.go)
  - Context Router v2.0 Execution Contract
  - KB1 Data Source Injection Plan - Phase 1 completion

Version: 2.0 (2026-01-22)
';

COMMENT ON TABLE source_authority_ranking IS
'Source authority hierarchy for DDI rule prioritization.
Lower authority_rank = higher priority (1 = highest).
Used by check_constitutional_ddi() for result ordering.';

-- =============================================================================
-- 5. Validation Query (Run manually to verify)
-- =============================================================================

-- Uncomment to test after migration:
-- SELECT
--     authority_code,
--     authority_rank,
--     authority_tier,
--     description
-- FROM source_authority_ranking
-- ORDER BY authority_rank;

COMMIT;

-- =============================================================================
-- MIGRATION METADATA
-- =============================================================================
-- Version: 031
-- Date: 2026-01-22
-- Author: Claude Code
-- Purpose: Implement ONC > OHDSI authority priority in query layer
-- Phase: Phase 1 completion item
-- Dependencies: 030_onc_constitutional_class_expansion.sql
-- =============================================================================
