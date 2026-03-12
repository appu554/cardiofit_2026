-- =============================================================================
-- MIGRATION 001: Drug Master Table (Layer 0 - Drug Universe)
-- Purpose: Canonical RxNorm-anchored drug registry for all Knowledge Base services
-- Reference: KB1 Implementation Plan Section 2.2
-- =============================================================================

BEGIN;

-- Track migration version
CREATE TABLE IF NOT EXISTS schema_migrations (
    version INTEGER PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- =============================================================================
-- DRUG MASTER TABLE
-- The canonical drug registry - ALL KBs reference drugs through this table
-- =============================================================================

CREATE TABLE IF NOT EXISTS drug_master (
    -- Primary Identifier (RxNorm CUI)
    rxcui               VARCHAR(20) PRIMARY KEY,

    -- Names
    drug_name           VARCHAR(500) NOT NULL,
    generic_name        VARCHAR(500),
    brand_names         TEXT[],

    -- Classification (RxNorm Term Type)
    tty                 VARCHAR(10) NOT NULL,  -- IN, MIN, SCDC, SCD, SBD, GPCK, BPCK
    atc_codes           TEXT[],                -- Anatomical Therapeutic Chemical codes
    therapeutic_class   VARCHAR(200),

    -- Hierarchy
    ingredient_rxcui    VARCHAR(20),           -- Points to base ingredient
    drug_class_rxcuis   TEXT[],                -- Drug classes this belongs to

    -- Cross-References (for lookups from external systems)
    ndcs                TEXT[],                -- National Drug Codes
    spl_set_ids         TEXT[],                -- FDA SPL Set IDs
    snomed_codes        TEXT[],                -- SNOMED CT codes
    uniis               TEXT[],                -- FDA UNII codes

    -- Status & Lifecycle
    status              VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',  -- ACTIVE, RETIRED, REMAPPED
    remapped_to         VARCHAR(20),           -- If REMAPPED, points to new RxCUI

    -- Sync Metadata
    rxnorm_version      VARCHAR(20) NOT NULL,
    last_synced_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Self-referential constraint for ingredient hierarchy
    CONSTRAINT fk_ingredient FOREIGN KEY (ingredient_rxcui)
        REFERENCES drug_master(rxcui) ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED,

    -- Ensure remapped drugs point to valid targets
    CONSTRAINT fk_remapped FOREIGN KEY (remapped_to)
        REFERENCES drug_master(rxcui) ON DELETE SET NULL DEFERRABLE INITIALLY DEFERRED,

    -- Status validation
    CONSTRAINT chk_status CHECK (status IN ('ACTIVE', 'RETIRED', 'REMAPPED')),

    -- TTY validation (RxNorm term types)
    CONSTRAINT chk_tty CHECK (tty IN ('IN', 'MIN', 'SCDC', 'SCD', 'SBD', 'GPCK', 'BPCK', 'PIN', 'BN', 'DF', 'DFG'))
);

-- =============================================================================
-- INDEXES FOR COMMON LOOKUPS
-- =============================================================================

-- Full-text search on drug names
CREATE INDEX idx_drug_master_name_fts ON drug_master
    USING gin(to_tsvector('english', drug_name || ' ' || COALESCE(generic_name, '')));

-- Classification lookups
CREATE INDEX idx_drug_master_tty ON drug_master(tty);
CREATE INDEX idx_drug_master_status ON drug_master(status) WHERE status = 'ACTIVE';
CREATE INDEX idx_drug_master_therapeutic ON drug_master(therapeutic_class);

-- Hierarchy traversal
CREATE INDEX idx_drug_master_ingredient ON drug_master(ingredient_rxcui) WHERE ingredient_rxcui IS NOT NULL;

-- Cross-reference lookups (GIN for array containment)
CREATE INDEX idx_drug_master_ndcs ON drug_master USING gin(ndcs);
CREATE INDEX idx_drug_master_spl ON drug_master USING gin(spl_set_ids);
CREATE INDEX idx_drug_master_atc ON drug_master USING gin(atc_codes);
CREATE INDEX idx_drug_master_class ON drug_master USING gin(drug_class_rxcuis);
CREATE INDEX idx_drug_master_snomed ON drug_master USING gin(snomed_codes);

-- =============================================================================
-- DRUG CLASS HIERARCHY TABLE
-- For therapeutic/pharmacological class lookups (RxClass, ATC, EPC, MOA)
-- =============================================================================

CREATE TABLE IF NOT EXISTS drug_class (
    class_id            VARCHAR(50) PRIMARY KEY,
    class_name          VARCHAR(300) NOT NULL,
    class_type          VARCHAR(20) NOT NULL,  -- ATC, EPC, MOA, PE, DISE
    parent_class_id     VARCHAR(50),           -- For hierarchical classes
    member_rxcuis       TEXT[],                -- Drugs in this class
    source              VARCHAR(50) NOT NULL,  -- RxClass, ATC WHO, etc.

    -- Metadata
    last_synced_at      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Self-referential for hierarchy
    CONSTRAINT fk_parent_class FOREIGN KEY (parent_class_id)
        REFERENCES drug_class(class_id) ON DELETE SET NULL,

    -- Class type validation
    CONSTRAINT chk_class_type CHECK (class_type IN ('ATC', 'EPC', 'MOA', 'PE', 'DISE', 'VA', 'MESH'))
);

CREATE INDEX idx_drug_class_type ON drug_class(class_type);
CREATE INDEX idx_drug_class_parent ON drug_class(parent_class_id) WHERE parent_class_id IS NOT NULL;
CREATE INDEX idx_drug_class_members ON drug_class USING gin(member_rxcuis);
CREATE INDEX idx_drug_class_name_fts ON drug_class USING gin(to_tsvector('english', class_name));

-- =============================================================================
-- HELPER FUNCTIONS
-- =============================================================================

-- Function to resolve any drug form to its base ingredient
CREATE OR REPLACE FUNCTION resolve_to_ingredient(p_rxcui VARCHAR(20))
RETURNS VARCHAR(20) AS $$
DECLARE
    v_ingredient VARCHAR(20);
    v_tty VARCHAR(10);
BEGIN
    SELECT tty, ingredient_rxcui INTO v_tty, v_ingredient
    FROM drug_master WHERE rxcui = p_rxcui;

    -- If already an ingredient (IN or MIN), return as-is
    IF v_tty IN ('IN', 'MIN') THEN
        RETURN p_rxcui;
    END IF;

    -- Otherwise return the ingredient RxCUI
    RETURN COALESCE(v_ingredient, p_rxcui);
END;
$$ LANGUAGE plpgsql STABLE;

-- Function to get all drugs containing an ingredient
CREATE OR REPLACE FUNCTION get_drugs_by_ingredient(p_ingredient_rxcui VARCHAR(20))
RETURNS TABLE(rxcui VARCHAR(20), drug_name VARCHAR(500), tty VARCHAR(10)) AS $$
BEGIN
    RETURN QUERY
    SELECT dm.rxcui, dm.drug_name, dm.tty
    FROM drug_master dm
    WHERE dm.ingredient_rxcui = p_ingredient_rxcui
       OR dm.rxcui = p_ingredient_rxcui
    ORDER BY dm.tty;
END;
$$ LANGUAGE plpgsql STABLE;

-- Record migration
INSERT INTO schema_migrations (version, name) VALUES (1, '001_drug_master_table')
ON CONFLICT (version) DO NOTHING;

COMMIT;

-- =============================================================================
-- VERIFICATION
-- =============================================================================
SELECT 'Migration 001: Drug Master Table - COMPLETE' AS status;
