-- ============================================================================
-- OHDSI Athena Vocabulary Schema for CardioFit Phase 3
-- ============================================================================
-- This migration creates the OMOP CDM vocabulary tables for:
--   - RxNorm (313K+ drug concepts)
--   - SNOMED (1M+ clinical terms)
--   - LOINC (277K+ lab/observation codes)
--   - NDC (1.2M+ national drug codes)
--   - ATC (7K+ drug classifications)
--   - UCUM (1K+ units of measure)
--
-- Source: OHDSI Athena (https://athena.ohdsi.org/)
-- ============================================================================

-- Create schema for OHDSI vocabularies
CREATE SCHEMA IF NOT EXISTS ohdsi;

-- ============================================================================
-- CORE VOCABULARY TABLES
-- ============================================================================

-- VOCABULARY: Defines each vocabulary (RxNorm, SNOMED, LOINC, etc.)
CREATE TABLE IF NOT EXISTS ohdsi.vocabulary (
    vocabulary_id VARCHAR(20) PRIMARY KEY,
    vocabulary_name VARCHAR(255) NOT NULL,
    vocabulary_reference VARCHAR(255),
    vocabulary_version VARCHAR(255),
    vocabulary_concept_id INTEGER
);

-- DOMAIN: Categorizes concepts (Drug, Condition, Observation, etc.)
CREATE TABLE IF NOT EXISTS ohdsi.domain (
    domain_id VARCHAR(20) PRIMARY KEY,
    domain_name VARCHAR(255) NOT NULL,
    domain_concept_id INTEGER
);

-- CONCEPT_CLASS: Sub-classification within domains
CREATE TABLE IF NOT EXISTS ohdsi.concept_class (
    concept_class_id VARCHAR(20) PRIMARY KEY,
    concept_class_name VARCHAR(255) NOT NULL,
    concept_class_concept_id INTEGER
);

-- RELATIONSHIP: Defines relationship types between concepts
CREATE TABLE IF NOT EXISTS ohdsi.relationship (
    relationship_id VARCHAR(20) PRIMARY KEY,
    relationship_name VARCHAR(255) NOT NULL,
    is_hierarchical VARCHAR(1),
    defines_ancestry VARCHAR(1),
    reverse_relationship_id VARCHAR(20),
    relationship_concept_id INTEGER
);

-- ============================================================================
-- MAIN CONCEPT TABLE
-- ============================================================================

-- CONCEPT: The core table containing all vocabulary concepts
CREATE TABLE IF NOT EXISTS ohdsi.concept (
    concept_id INTEGER PRIMARY KEY,
    concept_name VARCHAR(512) NOT NULL,
    domain_id VARCHAR(20) NOT NULL,
    vocabulary_id VARCHAR(20) NOT NULL,
    concept_class_id VARCHAR(50) NOT NULL,
    standard_concept VARCHAR(1),
    concept_code VARCHAR(50) NOT NULL,
    valid_start_date DATE NOT NULL,
    valid_end_date DATE NOT NULL,
    invalid_reason VARCHAR(1)
);

-- Indexes for common lookups
CREATE INDEX IF NOT EXISTS idx_concept_vocab ON ohdsi.concept(vocabulary_id);
CREATE INDEX IF NOT EXISTS idx_concept_domain ON ohdsi.concept(domain_id);
CREATE INDEX IF NOT EXISTS idx_concept_code ON ohdsi.concept(vocabulary_id, concept_code);
CREATE INDEX IF NOT EXISTS idx_concept_name ON ohdsi.concept(concept_name);
CREATE INDEX IF NOT EXISTS idx_concept_standard ON ohdsi.concept(standard_concept) WHERE standard_concept = 'S';

-- ============================================================================
-- CONCEPT RELATIONSHIPS
-- ============================================================================

-- CONCEPT_RELATIONSHIP: Links between concepts (Maps to, Has ingredient, etc.)
CREATE TABLE IF NOT EXISTS ohdsi.concept_relationship (
    concept_id_1 INTEGER NOT NULL,
    concept_id_2 INTEGER NOT NULL,
    relationship_id VARCHAR(20) NOT NULL,
    valid_start_date DATE NOT NULL,
    valid_end_date DATE NOT NULL,
    invalid_reason VARCHAR(1),
    PRIMARY KEY (concept_id_1, concept_id_2, relationship_id)
);

-- Indexes for relationship lookups
CREATE INDEX IF NOT EXISTS idx_concept_rel_1 ON ohdsi.concept_relationship(concept_id_1);
CREATE INDEX IF NOT EXISTS idx_concept_rel_2 ON ohdsi.concept_relationship(concept_id_2);
CREATE INDEX IF NOT EXISTS idx_concept_rel_type ON ohdsi.concept_relationship(relationship_id);

-- CONCEPT_ANCESTOR: Hierarchical relationships for efficient traversal
CREATE TABLE IF NOT EXISTS ohdsi.concept_ancestor (
    ancestor_concept_id INTEGER NOT NULL,
    descendant_concept_id INTEGER NOT NULL,
    min_levels_of_separation INTEGER NOT NULL,
    max_levels_of_separation INTEGER NOT NULL,
    PRIMARY KEY (ancestor_concept_id, descendant_concept_id)
);

-- Indexes for hierarchy navigation
CREATE INDEX IF NOT EXISTS idx_ancestor_desc ON ohdsi.concept_ancestor(descendant_concept_id);

-- ============================================================================
-- CONCEPT SYNONYMS
-- ============================================================================

-- CONCEPT_SYNONYM: Alternative names for concepts
CREATE TABLE IF NOT EXISTS ohdsi.concept_synonym (
    concept_id INTEGER NOT NULL,
    concept_synonym_name VARCHAR(1000) NOT NULL,
    language_concept_id INTEGER NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_synonym_concept ON ohdsi.concept_synonym(concept_id);
CREATE INDEX IF NOT EXISTS idx_synonym_name ON ohdsi.concept_synonym(concept_synonym_name);

-- ============================================================================
-- DRUG-SPECIFIC TABLES
-- ============================================================================

-- DRUG_STRENGTH: Dosage information for drug products
CREATE TABLE IF NOT EXISTS ohdsi.drug_strength (
    drug_concept_id INTEGER NOT NULL,
    ingredient_concept_id INTEGER NOT NULL,
    amount_value NUMERIC,
    amount_unit_concept_id INTEGER,
    numerator_value NUMERIC,
    numerator_unit_concept_id INTEGER,
    denominator_value NUMERIC,
    denominator_unit_concept_id INTEGER,
    box_size INTEGER,
    valid_start_date DATE NOT NULL,
    valid_end_date DATE NOT NULL,
    invalid_reason VARCHAR(1),
    PRIMARY KEY (drug_concept_id, ingredient_concept_id)
);

-- Indexes for drug strength lookups
CREATE INDEX IF NOT EXISTS idx_drug_strength_drug ON ohdsi.drug_strength(drug_concept_id);
CREATE INDEX IF NOT EXISTS idx_drug_strength_ingredient ON ohdsi.drug_strength(ingredient_concept_id);

-- ============================================================================
-- CONVENIENCE VIEWS
-- ============================================================================

-- View: Active RxNorm drugs with standard concepts
CREATE OR REPLACE VIEW ohdsi.v_rxnorm_drugs AS
SELECT
    c.concept_id,
    c.concept_code AS rxcui,
    c.concept_name AS drug_name,
    c.concept_class_id AS drug_type,
    c.valid_start_date,
    c.valid_end_date
FROM ohdsi.concept c
WHERE c.vocabulary_id = 'RxNorm'
  AND c.standard_concept = 'S'
  AND c.invalid_reason IS NULL;

-- View: Active LOINC codes
CREATE OR REPLACE VIEW ohdsi.v_loinc_codes AS
SELECT
    c.concept_id,
    c.concept_code AS loinc_code,
    c.concept_name AS loinc_name,
    c.concept_class_id AS loinc_class,
    c.domain_id
FROM ohdsi.concept c
WHERE c.vocabulary_id = 'LOINC'
  AND c.invalid_reason IS NULL;

-- View: Active SNOMED conditions
CREATE OR REPLACE VIEW ohdsi.v_snomed_conditions AS
SELECT
    c.concept_id,
    c.concept_code AS snomed_code,
    c.concept_name AS condition_name,
    c.concept_class_id
FROM ohdsi.concept c
WHERE c.vocabulary_id = 'SNOMED'
  AND c.domain_id = 'Condition'
  AND c.invalid_reason IS NULL;

-- View: ATC drug classifications
CREATE OR REPLACE VIEW ohdsi.v_atc_classes AS
SELECT
    c.concept_id,
    c.concept_code AS atc_code,
    c.concept_name AS atc_name,
    c.concept_class_id AS atc_level
FROM ohdsi.concept c
WHERE c.vocabulary_id = 'ATC'
  AND c.invalid_reason IS NULL
ORDER BY c.concept_code;

-- View: Drug ingredients with strength
CREATE OR REPLACE VIEW ohdsi.v_drug_ingredients AS
SELECT
    d.concept_id AS drug_id,
    d.concept_name AS drug_name,
    d.concept_code AS rxcui,
    i.concept_id AS ingredient_id,
    i.concept_name AS ingredient_name,
    ds.amount_value,
    u.concept_name AS amount_unit,
    ds.numerator_value,
    ds.denominator_value
FROM ohdsi.concept d
JOIN ohdsi.drug_strength ds ON d.concept_id = ds.drug_concept_id
JOIN ohdsi.concept i ON ds.ingredient_concept_id = i.concept_id
LEFT JOIN ohdsi.concept u ON ds.amount_unit_concept_id = u.concept_id
WHERE d.vocabulary_id IN ('RxNorm', 'RxNorm Extension')
  AND d.invalid_reason IS NULL;

-- ============================================================================
-- LOOKUP FUNCTIONS
-- ============================================================================

-- Function: Get RxCUI by NDC code
CREATE OR REPLACE FUNCTION ohdsi.get_rxcui_by_ndc(p_ndc VARCHAR)
RETURNS TABLE(rxcui VARCHAR, drug_name VARCHAR) AS $$
BEGIN
    RETURN QUERY
    SELECT
        rxnorm.concept_code,
        rxnorm.concept_name
    FROM ohdsi.concept ndc
    JOIN ohdsi.concept_relationship cr ON ndc.concept_id = cr.concept_id_1
    JOIN ohdsi.concept rxnorm ON cr.concept_id_2 = rxnorm.concept_id
    WHERE ndc.vocabulary_id = 'NDC'
      AND ndc.concept_code = p_ndc
      AND cr.relationship_id = 'Maps to'
      AND rxnorm.vocabulary_id = 'RxNorm'
      AND rxnorm.standard_concept = 'S';
END;
$$ LANGUAGE plpgsql;

-- Function: Get drug interactions by RxCUI
CREATE OR REPLACE FUNCTION ohdsi.get_drug_class(p_rxcui VARCHAR)
RETURNS TABLE(atc_code VARCHAR, atc_name VARCHAR, atc_level VARCHAR) AS $$
BEGIN
    RETURN QUERY
    SELECT
        atc.concept_code,
        atc.concept_name,
        atc.concept_class_id
    FROM ohdsi.concept drug
    JOIN ohdsi.concept_ancestor ca ON drug.concept_id = ca.descendant_concept_id
    JOIN ohdsi.concept atc ON ca.ancestor_concept_id = atc.concept_id
    WHERE drug.vocabulary_id = 'RxNorm'
      AND drug.concept_code = p_rxcui
      AND atc.vocabulary_id = 'ATC'
    ORDER BY atc.concept_code;
END;
$$ LANGUAGE plpgsql;

-- Function: Get drug ingredients
CREATE OR REPLACE FUNCTION ohdsi.get_drug_ingredients(p_rxcui VARCHAR)
RETURNS TABLE(
    ingredient_rxcui VARCHAR,
    ingredient_name VARCHAR,
    amount_value NUMERIC,
    amount_unit VARCHAR
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        ing.concept_code,
        ing.concept_name,
        ds.amount_value,
        unit.concept_name
    FROM ohdsi.concept drug
    JOIN ohdsi.drug_strength ds ON drug.concept_id = ds.drug_concept_id
    JOIN ohdsi.concept ing ON ds.ingredient_concept_id = ing.concept_id
    LEFT JOIN ohdsi.concept unit ON ds.amount_unit_concept_id = unit.concept_id
    WHERE drug.vocabulary_id = 'RxNorm'
      AND drug.concept_code = p_rxcui;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON SCHEMA ohdsi IS 'OHDSI Athena Vocabulary Schema for CardioFit Phase 3';
COMMENT ON TABLE ohdsi.concept IS 'Core vocabulary concepts from RxNorm, SNOMED, LOINC, NDC, ATC';
COMMENT ON TABLE ohdsi.drug_strength IS 'Drug dosage/strength information for dosing calculations';
COMMENT ON TABLE ohdsi.concept_relationship IS 'Relationships between concepts (Maps to, Has ingredient, etc.)';
COMMENT ON TABLE ohdsi.concept_ancestor IS 'Hierarchical ancestry for efficient traversal';
