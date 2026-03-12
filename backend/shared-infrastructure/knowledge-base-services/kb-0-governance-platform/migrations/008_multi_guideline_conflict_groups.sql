-- =============================================================================
-- Migration 008: Multi-Guideline Conflict Resolution Schema
-- =============================================================================
-- Extends Pipeline 1 review tables to support multi-guideline extraction
-- (ADA 2026, RSSDI 2022, KDIGO 2024) with conflict group tracking,
-- override types, and 3-tier resolution (universal → country → patient).
--
-- Prerequisite: 002_pipeline1_schema.sql (l2_extraction_jobs, l2_merged_spans)
-- =============================================================================

-- =============================================================================
-- 1. ALTER l2_extraction_jobs — add guideline tier metadata
-- =============================================================================

ALTER TABLE l2_extraction_jobs
    ADD COLUMN IF NOT EXISTS guideline_tier SMALLINT NOT NULL DEFAULT 1;

COMMENT ON COLUMN l2_extraction_jobs.guideline_tier IS
    'Tier level: 1=universal (ADA/KDIGO), 2=country override (RSSDI/RACGP), 3=patient-level (KB-20)';

-- =============================================================================
-- 2. ALTER l2_merged_spans — add conflict resolution columns
-- =============================================================================

ALTER TABLE l2_merged_spans
    ADD COLUMN IF NOT EXISTS conflict_group_id VARCHAR(30),
    ADD COLUMN IF NOT EXISTS override_type VARCHAR(20),
    ADD COLUMN IF NOT EXISTS tier_level SMALLINT DEFAULT 1,
    ADD COLUMN IF NOT EXISTS country_code VARCHAR(2);

-- Override type constraint: 7 valid override types
-- NULL          = no conflict (standalone span)
-- ADDITIVE      = adds new information not present in higher tier
-- MODIFIED      = same target, different threshold/dose/cadence
-- CONDITIONAL   = applies only when specific antecedents met
-- SUBSTITUTION  = replaces higher-tier drug with local alternative
-- RESTRICTED    = blocks entire drug class pathway (no substitute available)
-- UNRESOLVED    = conflict detected, awaiting manual review
ALTER TABLE l2_merged_spans
    ADD CONSTRAINT chk_l2_spans_override_type
    CHECK (override_type IS NULL OR override_type IN (
        'ADDITIVE', 'MODIFIED', 'CONDITIONAL',
        'SUBSTITUTION', 'RESTRICTED', 'UNRESOLVED'
    ));

COMMENT ON COLUMN l2_merged_spans.conflict_group_id IS
    'Links spans across guidelines that address the same clinical decision (e.g., CG-CKD-001)';
COMMENT ON COLUMN l2_merged_spans.override_type IS
    'How this span relates to the higher-tier span in the same conflict group';
COMMENT ON COLUMN l2_merged_spans.tier_level IS
    'Tier: 1=universal (ADA/KDIGO), 2=country (RSSDI/RACGP), 3=patient (KB-20)';
COMMENT ON COLUMN l2_merged_spans.country_code IS
    'ISO 3166-1 alpha-2 country code for tier-2 spans (IN=India, AU=Australia)';

-- Indexes for conflict resolution queries
CREATE INDEX IF NOT EXISTS idx_l2_spans_conflict_group
    ON l2_merged_spans(conflict_group_id)
    WHERE conflict_group_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_l2_spans_tier_country
    ON l2_merged_spans(tier_level, country_code);

CREATE INDEX IF NOT EXISTS idx_l2_spans_override_type
    ON l2_merged_spans(override_type)
    WHERE override_type IS NOT NULL;

-- =============================================================================
-- 3. CREATE l2_conflict_groups — conflict resolution tracking
-- =============================================================================
-- Each row represents a clinical decision point where multiple guidelines
-- may disagree (e.g., "SGLT2i initiation threshold in CKD").
-- Populated after both ADA and RSSDI extractions complete.
-- =============================================================================

CREATE TABLE l2_conflict_groups (
    conflict_group_id   VARCHAR(30) PRIMARY KEY,
    domain              VARCHAR(30) NOT NULL,
    description         TEXT,

    -- Resolution state machine
    resolution_state    VARCHAR(20) NOT NULL DEFAULT 'PENDING'
        CHECK (resolution_state IN (
            'PENDING',       -- conflict detected, not yet reviewed
            'HARMONIZED',    -- guidelines agree (no real conflict)
            'OVERRIDDEN',    -- tier-2 overrides tier-1 for target country
            'CONDITIONAL',   -- override applies only under specific antecedents
            'RESTRICTED',    -- drug class pathway blocked in target country
            'UNRESOLVED'     -- requires escalation to clinical governance
        )),
    resolution_justification TEXT,

    -- Reviewer tracking
    resolved_by         VARCHAR(100),
    resolved_at         TIMESTAMP WITH TIME ZONE,

    -- Winning span (the span that takes precedence after resolution)
    winning_span_id     UUID REFERENCES l2_merged_spans(id),

    -- Source-lag metadata: flags when tier-2 source predates tier-1
    -- significantly (e.g., RSSDI 2022 vs ADA 2026 = 4-year gap)
    source_lag          BOOLEAN NOT NULL DEFAULT FALSE,
    source_lag_note     TEXT,

    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE l2_conflict_groups IS
    'Tracks clinical decision points where multiple guidelines may conflict. '
    'Each group links spans across guidelines via l2_merged_spans.conflict_group_id.';

CREATE INDEX idx_l2_conflict_groups_domain
    ON l2_conflict_groups(domain);
CREATE INDEX idx_l2_conflict_groups_state
    ON l2_conflict_groups(resolution_state);
CREATE INDEX idx_l2_conflict_groups_lag
    ON l2_conflict_groups(source_lag)
    WHERE source_lag = TRUE;

-- Auto-update updated_at trigger
CREATE OR REPLACE FUNCTION l2_update_conflict_group_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tr_l2_conflict_groups_updated
    BEFORE UPDATE ON l2_conflict_groups
    FOR EACH ROW
    EXECUTE FUNCTION l2_update_conflict_group_timestamp();

-- =============================================================================
-- 4. VIEW: v_l2_conflict_summary — conflict resolution dashboard
-- =============================================================================

CREATE OR REPLACE VIEW v_l2_conflict_summary AS
SELECT
    cg.conflict_group_id,
    cg.domain,
    cg.description,
    cg.resolution_state,
    cg.source_lag,
    cg.source_lag_note,
    cg.resolved_by,
    cg.resolved_at,
    COUNT(ms.id) AS span_count,
    COUNT(DISTINCT ms.job_id) AS guideline_count,
    ARRAY_AGG(DISTINCT ms.tier_level ORDER BY ms.tier_level) AS tier_levels,
    ARRAY_AGG(DISTINCT ms.country_code) FILTER (WHERE ms.country_code IS NOT NULL) AS countries
FROM l2_conflict_groups cg
LEFT JOIN l2_merged_spans ms ON ms.conflict_group_id = cg.conflict_group_id
GROUP BY cg.conflict_group_id, cg.domain, cg.description,
         cg.resolution_state, cg.source_lag, cg.source_lag_note,
         cg.resolved_by, cg.resolved_at;

-- =============================================================================
-- 5. SEED: Pre-populate conflict group IDs for the 5 overlap zones
-- =============================================================================
-- These are the ~40 conflict group IDs from the RSSDI Extraction Map.
-- Resolution state starts as PENDING — populated after extraction.
-- =============================================================================

INSERT INTO l2_conflict_groups (conflict_group_id, domain, description, source_lag, source_lag_note) VALUES
    -- Zone 1: CKD Drug Rules (HIGH density)
    ('CG-CKD-001', 'ckd_drug_rules', 'SGLT2i initiation eGFR threshold', TRUE, 'ADA 2026 adopts eGFR≥20 (EMPA-KIDNEY); RSSDI 2022 uses eGFR≥30'),
    ('CG-CKD-002', 'ckd_drug_rules', 'SGLT2i continuation below initiation threshold', TRUE, 'ADA 2026: continue to dialysis; RSSDI 2022 silent'),
    ('CG-CKD-003', 'ckd_drug_rules', 'Finerenone (nsMRA) for CKD+DM', TRUE, 'ADA 2026 recommends finerenone; RSSDI 2022 does not list (not available in India)'),
    ('CG-CKD-004', 'ckd_drug_rules', 'Metformin eGFR threshold', FALSE, NULL),
    ('CG-CKD-005', 'ckd_drug_rules', 'DPP-4i dose adjustment in CKD stages', FALSE, NULL),
    ('CG-CKD-006', 'ckd_drug_rules', 'Sulfonylurea CKD safety', FALSE, NULL),
    ('CG-CKD-007', 'ckd_drug_rules', 'GLP-1 RA CKD dosing', TRUE, 'ADA 2026 includes tirzepatide data; RSSDI 2022 covers semaglutide/liraglutide only'),
    ('CG-CKD-008', 'ckd_drug_rules', 'Insulin dosing in advanced CKD', FALSE, NULL),
    ('CG-CKD-009', 'ckd_drug_rules', 'ACEi/ARB max tolerated dose in CKD', FALSE, NULL),
    ('CG-CKD-010', 'ckd_drug_rules', 'Dual SGLT2i+finerenone simultaneous start', TRUE, 'ADA 2026 endorses (CONFIDENCE trial); RSSDI 2022 silent on combination'),

    -- Zone 2: BP Targets (MEDIUM density)
    ('CG-BP-001', 'bp_targets', 'SBP target for DM+CKD', TRUE, 'ADA 2026: SBP<120 for high-risk (BPROAD trial); RSSDI 2022: <130/80'),
    ('CG-BP-002', 'bp_targets', 'First-line antihypertensive class', FALSE, NULL),
    ('CG-BP-003', 'bp_targets', 'ACEi/ARB preference for albuminuria', FALSE, NULL),
    ('CG-BP-004', 'bp_targets', 'BP target in elderly (>65y) with DM', FALSE, NULL),
    ('CG-BP-005', 'bp_targets', 'Resistant hypertension add-on therapy', FALSE, NULL),

    -- Zone 3: Glycemic Targets in CKD (MEDIUM density)
    ('CG-GLYC-001', 'glycemic_targets', 'HbA1c target for CKD G3-G5', FALSE, NULL),
    ('CG-GLYC-002', 'glycemic_targets', 'HbA1c target in elderly with CKD', FALSE, NULL),
    ('CG-GLYC-003', 'glycemic_targets', 'CGM TIR target in CKD', FALSE, NULL),
    ('CG-GLYC-004', 'glycemic_targets', 'Hypoglycemia avoidance threshold', FALSE, NULL),
    ('CG-GLYC-005', 'glycemic_targets', 'First-line glucose-lowering in T2DM', TRUE, 'ADA 2026: tirzepatide first-line option; RSSDI 2022: metformin first-line'),
    ('CG-GLYC-006', 'glycemic_targets', 'HbA1c reliability in advanced CKD (GA/fructosamine)', FALSE, NULL),

    -- Zone 4: Drug Sequencing Multi-Comorbidity (VERY HIGH density)
    ('CG-SEQ-001', 'drug_sequencing', 'Second-line after metformin with ASCVD', TRUE, 'ADA 2026: GLP-1 RA or SGLT2i; RSSDI 2022: similar but different drug availability'),
    ('CG-SEQ-002', 'drug_sequencing', 'Second-line after metformin with HF', FALSE, NULL),
    ('CG-SEQ-003', 'drug_sequencing', 'Second-line after metformin with CKD', TRUE, 'ADA 2026 adds tirzepatide path'),
    ('CG-SEQ-004', 'drug_sequencing', 'Triple therapy combinations', FALSE, NULL),
    ('CG-SEQ-005', 'drug_sequencing', 'Insulin initiation criteria', FALSE, NULL),
    ('CG-SEQ-006', 'drug_sequencing', 'GLP-1 RA vs insulin preference', TRUE, 'ADA 2026: stronger GLP-1 RA preference including tirzepatide'),
    ('CG-SEQ-007', 'drug_sequencing', 'Statin intensity by risk category', FALSE, NULL),
    ('CG-SEQ-008', 'drug_sequencing', 'Add-on lipid therapy (ezetimibe/PCSK9i)', FALSE, NULL),
    ('CG-SEQ-009', 'drug_sequencing', 'India-specific drug substitutions', FALSE, NULL),
    ('CG-SEQ-010', 'drug_sequencing', 'Alpha-glucosidase inhibitor positioning', FALSE, NULL),

    -- Zone 5: Monitoring Cadences (LOW-MEDIUM density)
    ('CG-MON-001', 'monitoring', 'HbA1c monitoring frequency', FALSE, NULL),
    ('CG-MON-002', 'monitoring', 'eGFR monitoring frequency in CKD', FALSE, NULL),
    ('CG-MON-003', 'monitoring', 'UACR monitoring frequency', FALSE, NULL),
    ('CG-MON-004', 'monitoring', 'Potassium monitoring with RAAS blockade', FALSE, NULL),
    ('CG-MON-005', 'monitoring', 'Lipid panel monitoring frequency', FALSE, NULL),
    ('CG-MON-006', 'monitoring', 'BP home monitoring recommendation', FALSE, NULL),
    ('CG-MON-007', 'monitoring', 'CGM usage recommendation for insulin users', FALSE, NULL),
    ('CG-MON-008', 'monitoring', 'Renal function monitoring on SGLT2i', FALSE, NULL)
ON CONFLICT (conflict_group_id) DO NOTHING;

-- =============================================================================
-- COMPLETION MESSAGE
-- =============================================================================
DO $$
BEGIN
    RAISE NOTICE '===================================================';
    RAISE NOTICE 'Migration 008: Multi-Guideline Conflict Groups';
    RAISE NOTICE '===================================================';
    RAISE NOTICE 'ALTER: l2_extraction_jobs + guideline_tier';
    RAISE NOTICE 'ALTER: l2_merged_spans + conflict_group_id,';
    RAISE NOTICE '       override_type, tier_level, country_code';
    RAISE NOTICE 'TABLE: l2_conflict_groups (40 groups seeded)';
    RAISE NOTICE 'VIEW:  v_l2_conflict_summary';
    RAISE NOTICE '===================================================';
END $$;
