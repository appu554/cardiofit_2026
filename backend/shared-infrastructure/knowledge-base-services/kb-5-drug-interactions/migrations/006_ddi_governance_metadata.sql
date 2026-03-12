-- =============================================================================
-- KB-5 Phase 5: DDI Governance Metadata - Three-Layer Authority Model
-- Migration: 006_ddi_governance_metadata.sql
-- Purpose: Add per-interaction provenance tracking for medico-legal defensibility
--
-- Three-Layer Authority Model:
--   Layer 1: REGULATORY AUTHORITY - "What is LEGALLY prohibited?"
--   Layer 2: PHARMACOLOGY AUTHORITY - "WHY does this interaction happen?"
--   Layer 3: CLINICAL PRACTICE AUTHORITY - "HOW do clinicians manage this?"
-- =============================================================================

-- =============================================================================
-- STEP 1: Extend clinical_source enum with additional authorities
-- =============================================================================

-- Add new clinical source values for Three-Layer Authority Model
-- Note: ALTER TYPE ADD VALUE cannot be inside a transaction block
ALTER TYPE clinical_source ADD VALUE IF NOT EXISTS 'TGA';            -- Therapeutic Goods Administration (AU)
ALTER TYPE clinical_source ADD VALUE IF NOT EXISTS 'CDSCO';          -- Central Drugs Standard Control (IN)
ALTER TYPE clinical_source ADD VALUE IF NOT EXISTS 'EMA';            -- European Medicines Agency
ALTER TYPE clinical_source ADD VALUE IF NOT EXISTS 'CREDIBLEMEDS';   -- CredibleMeds QT Drug Lists
ALTER TYPE clinical_source ADD VALUE IF NOT EXISTS 'UW_DDI';         -- University of Washington DDI Database
ALTER TYPE clinical_source ADD VALUE IF NOT EXISTS 'FLOCKHART_CYP';  -- Indiana University Flockhart CYP Table
ALTER TYPE clinical_source ADD VALUE IF NOT EXISTS 'STOCKLEY';       -- Stockley's Drug Interactions
ALTER TYPE clinical_source ADD VALUE IF NOT EXISTS 'AMH';            -- Australian Medicines Handbook
ALTER TYPE clinical_source ADD VALUE IF NOT EXISTS 'BNF';            -- British National Formulary
ALTER TYPE clinical_source ADD VALUE IF NOT EXISTS 'ACCP';           -- American College of Clinical Pharmacy

-- =============================================================================
-- STEP 2: Add governance columns to drug_interactions table
-- These columns implement the Three-Layer Authority Model
-- =============================================================================

-- Layer 1: REGULATORY AUTHORITY - "What is LEGALLY prohibited?"
ALTER TABLE drug_interactions
ADD COLUMN IF NOT EXISTS gov_regulatory_authority VARCHAR(50),
ADD COLUMN IF NOT EXISTS gov_regulatory_document VARCHAR(500),
ADD COLUMN IF NOT EXISTS gov_regulatory_url TEXT,
ADD COLUMN IF NOT EXISTS gov_regulatory_jurisdiction VARCHAR(20);

-- Layer 2: PHARMACOLOGY AUTHORITY - "WHY does this interaction happen?"
ALTER TABLE drug_interactions
ADD COLUMN IF NOT EXISTS gov_pharmacology_authority VARCHAR(50),
ADD COLUMN IF NOT EXISTS gov_mechanism_evidence VARCHAR(500),
ADD COLUMN IF NOT EXISTS gov_transporter_data VARCHAR(200),
ADD COLUMN IF NOT EXISTS gov_cyp_pathway VARCHAR(100),
ADD COLUMN IF NOT EXISTS gov_qt_risk_category VARCHAR(50);

-- Layer 3: CLINICAL PRACTICE AUTHORITY - "HOW do clinicians manage this?"
ALTER TABLE drug_interactions
ADD COLUMN IF NOT EXISTS gov_clinical_authority VARCHAR(50),
ADD COLUMN IF NOT EXISTS gov_severity_source VARCHAR(200),
ADD COLUMN IF NOT EXISTS gov_management_source VARCHAR(200);

-- Quality & Review Metadata
ALTER TABLE drug_interactions
ADD COLUMN IF NOT EXISTS gov_evidence_grade VARCHAR(10),
ADD COLUMN IF NOT EXISTS gov_last_reviewed_date DATE,
ADD COLUMN IF NOT EXISTS gov_next_review_due DATE,
ADD COLUMN IF NOT EXISTS gov_reviewed_by VARCHAR(100);

-- =============================================================================
-- STEP 3: Create indexes for efficient governance queries
-- =============================================================================

-- Index on regulatory authority for jurisdiction-based filtering
CREATE INDEX IF NOT EXISTS idx_ddi_gov_regulatory_authority
ON drug_interactions(gov_regulatory_authority);

-- Index on regulatory jurisdiction for regional compliance queries
CREATE INDEX IF NOT EXISTS idx_ddi_gov_regulatory_jurisdiction
ON drug_interactions(gov_regulatory_jurisdiction);

-- Index on pharmacology authority for mechanism-based queries
CREATE INDEX IF NOT EXISTS idx_ddi_gov_pharmacology_authority
ON drug_interactions(gov_pharmacology_authority);

-- Index on clinical authority for source attribution queries
CREATE INDEX IF NOT EXISTS idx_ddi_gov_clinical_authority
ON drug_interactions(gov_clinical_authority);

-- Index on evidence grade for quality-filtered queries
CREATE INDEX IF NOT EXISTS idx_ddi_gov_evidence_grade
ON drug_interactions(gov_evidence_grade);

-- Index on QT risk category for cardiac safety queries
CREATE INDEX IF NOT EXISTS idx_ddi_gov_qt_risk_category
ON drug_interactions(gov_qt_risk_category)
WHERE gov_qt_risk_category IS NOT NULL;

-- Index on review dates for compliance tracking
CREATE INDEX IF NOT EXISTS idx_ddi_gov_next_review_due
ON drug_interactions(gov_next_review_due)
WHERE gov_next_review_due IS NOT NULL;

-- =============================================================================
-- STEP 4: Add comments for documentation
-- =============================================================================

COMMENT ON COLUMN drug_interactions.gov_regulatory_authority IS
'Layer 1: Regulatory authority source (FDA_LABEL, TGA, CDSCO, EMA)';

COMMENT ON COLUMN drug_interactions.gov_regulatory_document IS
'Layer 1: Specific regulatory document reference (e.g., "Warfarin FDA Label Section 7")';

COMMENT ON COLUMN drug_interactions.gov_regulatory_url IS
'Layer 1: URL link to authoritative regulatory source';

COMMENT ON COLUMN drug_interactions.gov_regulatory_jurisdiction IS
'Layer 1: Geographic jurisdiction (US, AU, IN, EU, UK)';

COMMENT ON COLUMN drug_interactions.gov_pharmacology_authority IS
'Layer 2: Pharmacology authority source (DRUGBANK, FLOCKHART_CYP, PHARMGKB, CREDIBLEMEDS, UW_DDI)';

COMMENT ON COLUMN drug_interactions.gov_mechanism_evidence IS
'Layer 2: Mechanistic evidence (e.g., "CYP2C9 inhibition, Ki = 7.1 μM")';

COMMENT ON COLUMN drug_interactions.gov_transporter_data IS
'Layer 2: Drug transporter information (e.g., "P-gp substrate", "OATP1B1 inhibitor")';

COMMENT ON COLUMN drug_interactions.gov_cyp_pathway IS
'Layer 2: CYP enzyme pathways involved (e.g., "CYP3A4, CYP2C9")';

COMMENT ON COLUMN drug_interactions.gov_qt_risk_category IS
'Layer 2: CredibleMeds QT risk category (Known, Possible, Conditional)';

COMMENT ON COLUMN drug_interactions.gov_clinical_authority IS
'Layer 3: Clinical practice authority source (LEXICOMP, MICROMEDEX, AMH, BNF, STOCKLEY)';

COMMENT ON COLUMN drug_interactions.gov_severity_source IS
'Layer 3: Source for severity classification (e.g., "Lexicomp Drug Interactions 2024")';

COMMENT ON COLUMN drug_interactions.gov_management_source IS
'Layer 3: Source for management recommendations (e.g., "ACCP Antithrombotic Guidelines 2024")';

COMMENT ON COLUMN drug_interactions.gov_evidence_grade IS
'Quality: Evidence grade per ACCP grading (A, B, C, D)';

COMMENT ON COLUMN drug_interactions.gov_last_reviewed_date IS
'Quality: Date of last clinical review';

COMMENT ON COLUMN drug_interactions.gov_next_review_due IS
'Quality: Date when next review is due';

COMMENT ON COLUMN drug_interactions.gov_reviewed_by IS
'Quality: Name/ID of reviewer who last validated this interaction';

-- =============================================================================
-- STEP 5: Create view for governance compliance reporting
-- =============================================================================

CREATE OR REPLACE VIEW v_ddi_governance_status AS
SELECT
    interaction_id,
    drug_a_code,
    drug_a_name,
    drug_b_code,
    drug_b_name,
    severity,
    evidence_level,
    -- Governance status
    gov_regulatory_authority,
    gov_pharmacology_authority,
    gov_clinical_authority,
    gov_evidence_grade,
    -- Attribution completeness check
    CASE
        WHEN gov_regulatory_authority IS NOT NULL
         AND gov_pharmacology_authority IS NOT NULL
         AND gov_clinical_authority IS NOT NULL
        THEN TRUE
        ELSE FALSE
    END AS is_fully_attributed,
    -- Gaps identification
    ARRAY_REMOVE(ARRAY[
        CASE WHEN gov_regulatory_authority IS NULL THEN 'REGULATORY' END,
        CASE WHEN gov_pharmacology_authority IS NULL THEN 'PHARMACOLOGY' END,
        CASE WHEN gov_clinical_authority IS NULL THEN 'CLINICAL' END
    ], NULL) AS provenance_gaps,
    -- Review status
    gov_last_reviewed_date,
    gov_next_review_due,
    CASE
        WHEN gov_next_review_due IS NOT NULL AND gov_next_review_due < CURRENT_DATE
        THEN TRUE
        ELSE FALSE
    END AS review_overdue
FROM drug_interactions
WHERE active = TRUE;

COMMENT ON VIEW v_ddi_governance_status IS
'View for tracking DDI governance compliance and attribution completeness';

-- =============================================================================
-- Migration Complete
-- =============================================================================
