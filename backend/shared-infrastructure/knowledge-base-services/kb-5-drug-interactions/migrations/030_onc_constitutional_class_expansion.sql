-- =============================================================================
-- Migration 030: ONC Constitutional Rules & OHDSI Class Expansion
-- =============================================================================
-- This migration implements the "Class Expansion Logic" that transforms
-- 25 class-based ONC rules into executable drug-pair projections using
-- 73,842 OHDSI Drug → Class relationships.
--
-- Architecture: ONC → MED-RT → OHDSI → LOINC
-- =============================================================================

-- Drop existing if exists (for idempotency)
DROP VIEW IF EXISTS v_active_ddi_definitions CASCADE;
DROP VIEW IF EXISTS v_ddi_expansion_stats CASCADE;
DROP TABLE IF EXISTS ddi_projections CASCADE;
DROP TABLE IF EXISTS ddi_constitutional_rules CASCADE;
DROP TABLE IF EXISTS ohdsi_concept_relationship CASCADE;
DROP TABLE IF EXISTS ohdsi_concept CASCADE;

-- =============================================================================
-- 1. OHDSI Vocabulary Tables (Minimal Schema for Class Expansion)
-- =============================================================================

-- OHDSI Concept table (drugs and classes)
CREATE TABLE ohdsi_concept (
    concept_id BIGINT PRIMARY KEY,
    concept_name VARCHAR(500) NOT NULL,
    domain_id VARCHAR(50),           -- 'Drug', 'Drug/Ingredient'
    vocabulary_id VARCHAR(50),       -- 'RxNorm', 'ATC', 'MED-RT'
    concept_class_id VARCHAR(50),    -- 'Ingredient', 'Drug Class', 'ATC 4th'
    standard_concept VARCHAR(1),     -- 'S' = standard, 'C' = classification
    concept_code VARCHAR(100),
    valid_start_date DATE,
    valid_end_date DATE,
    invalid_reason VARCHAR(1),

    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- OHDSI Concept Relationship table (drug → class mappings)
CREATE TABLE ohdsi_concept_relationship (
    concept_id_1 BIGINT NOT NULL,    -- Drug concept
    concept_id_2 BIGINT NOT NULL,    -- Class concept
    relationship_id VARCHAR(50) NOT NULL,  -- 'Drug has drug class'
    valid_start_date DATE,
    valid_end_date DATE,
    invalid_reason VARCHAR(1),

    created_at TIMESTAMPTZ DEFAULT NOW(),

    PRIMARY KEY (concept_id_1, concept_id_2, relationship_id),
    FOREIGN KEY (concept_id_1) REFERENCES ohdsi_concept(concept_id),
    FOREIGN KEY (concept_id_2) REFERENCES ohdsi_concept(concept_id)
);

-- Performance indexes for class expansion
CREATE INDEX idx_ocr_relationship ON ohdsi_concept_relationship(relationship_id);
CREATE INDEX idx_ocr_class ON ohdsi_concept_relationship(concept_id_2)
    WHERE relationship_id = 'Drug has drug class';
CREATE INDEX idx_ocr_drug ON ohdsi_concept_relationship(concept_id_1)
    WHERE relationship_id = 'Drug has drug class';
CREATE INDEX idx_oc_standard ON ohdsi_concept(standard_concept) WHERE standard_concept = 'S';
CREATE INDEX idx_oc_class ON ohdsi_concept(concept_class_id);
CREATE INDEX idx_oc_vocabulary ON ohdsi_concept(vocabulary_id);

-- =============================================================================
-- 2. ONC Constitutional Rules Table (The "Brain")
-- =============================================================================

CREATE TABLE ddi_constitutional_rules (
    rule_id SERIAL PRIMARY KEY,

    -- Class identifiers (OHDSI execution anchors)
    trigger_class_name VARCHAR(255) NOT NULL,
    trigger_concept_id BIGINT NOT NULL,
    target_class_name VARCHAR(255) NOT NULL,
    target_concept_id BIGINT NOT NULL,

    -- Severity and clinical context
    risk_level VARCHAR(20) NOT NULL CHECK (risk_level IN ('CRITICAL', 'HIGH', 'WARNING', 'MODERATE')),
    description TEXT NOT NULL,

    -- LOINC context for smart alerting (KB-16 reference)
    context_loinc_id VARCHAR(20),
    context_loinc_name VARCHAR(255),
    context_threshold_val DECIMAL(10,2),
    context_logic_operator VARCHAR(5) CHECK (context_logic_operator IN ('<', '>', '<=', '>=', '=')),
    context_required BOOLEAN DEFAULT FALSE,

    -- Governance columns (critical for audit)
    rule_authority VARCHAR(100) NOT NULL,
    rule_version VARCHAR(20) NOT NULL DEFAULT 'v1.0',

    -- Metadata
    frozen BOOLEAN DEFAULT TRUE,  -- Constitutional rules are immutable
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for rule lookup
CREATE INDEX idx_dcr_trigger ON ddi_constitutional_rules(trigger_concept_id);
CREATE INDEX idx_dcr_target ON ddi_constitutional_rules(target_concept_id);
CREATE INDEX idx_dcr_authority ON ddi_constitutional_rules(rule_authority);
CREATE INDEX idx_dcr_active ON ddi_constitutional_rules(active) WHERE active = TRUE;

-- =============================================================================
-- 3. DDI Projections Table (Expanded Drug Pairs)
-- =============================================================================

CREATE TABLE ddi_projections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Link to constitutional rule
    rule_id INTEGER NOT NULL REFERENCES ddi_constitutional_rules(rule_id),

    -- Canonical drug pair (always drug_a < drug_b for deduplication)
    drug_a_concept_id BIGINT NOT NULL,
    drug_a_name VARCHAR(500),
    drug_b_concept_id BIGINT NOT NULL,
    drug_b_name VARCHAR(500),

    -- Inherited from rule
    risk_level VARCHAR(20) NOT NULL,
    rule_authority VARCHAR(100) NOT NULL,
    rule_version VARCHAR(20) NOT NULL,

    -- Precedence for override logic
    precedence_rank INTEGER DEFAULT 3,  -- 1=Ingredient-Ingredient, 2=Ingredient-Class, 3=Class-Class

    -- Context reference
    requires_context BOOLEAN DEFAULT FALSE,
    context_loinc_id VARCHAR(20),

    -- Metadata
    expansion_timestamp TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE (drug_a_concept_id, drug_b_concept_id, rule_id)
);

-- Performance indexes for DDI checking
CREATE INDEX idx_dp_drug_a ON ddi_projections(drug_a_concept_id);
CREATE INDEX idx_dp_drug_b ON ddi_projections(drug_b_concept_id);
CREATE INDEX idx_dp_rule ON ddi_projections(rule_id);
CREATE INDEX idx_dp_lookup ON ddi_projections(drug_a_concept_id, drug_b_concept_id);
CREATE INDEX idx_dp_severity ON ddi_projections(risk_level);

-- =============================================================================
-- 4. Class Expansion View (Virtual Expansion - Runtime)
-- =============================================================================

CREATE OR REPLACE VIEW v_active_ddi_definitions AS
SELECT
    r.rule_id,
    r.description AS alert_message,
    r.risk_level,

    -- TRIGGER SIDE (e.g., Warfarin)
    r.trigger_class_name,
    c1.concept_name AS trigger_drug_name,
    c1.concept_id AS trigger_drug_id,

    -- TARGET SIDE (e.g., Ibuprofen)
    r.target_class_name,
    c2.concept_name AS target_drug_name,
    c2.concept_id AS target_drug_id,

    -- CONTEXT (LOINC reference)
    r.context_loinc_id,
    r.context_loinc_name,
    r.context_threshold_val,
    r.context_logic_operator,
    r.context_required,

    -- GOVERNANCE
    r.rule_authority,
    r.rule_version

FROM ddi_constitutional_rules r

-- Expand Trigger Class to Drugs
JOIN ohdsi_concept_relationship cr1
    ON r.trigger_concept_id = cr1.concept_id_2
    AND cr1.relationship_id = 'Drug has drug class'
    AND cr1.invalid_reason IS NULL
JOIN ohdsi_concept c1
    ON cr1.concept_id_1 = c1.concept_id
    AND c1.standard_concept = 'S'

-- Expand Target Class to Drugs
JOIN ohdsi_concept_relationship cr2
    ON r.target_concept_id = cr2.concept_id_2
    AND cr2.relationship_id = 'Drug has drug class'
    AND cr2.invalid_reason IS NULL
JOIN ohdsi_concept c2
    ON cr2.concept_id_1 = c2.concept_id
    AND c2.standard_concept = 'S'

WHERE r.active = TRUE
    AND r.frozen = TRUE  -- Only constitutional rules
    -- Exclude self-pairs (unless same drug in different classes)
    AND c1.concept_id != c2.concept_id;

-- =============================================================================
-- 5. Expansion Statistics View (Validation)
-- =============================================================================

CREATE OR REPLACE VIEW v_ddi_expansion_stats AS
SELECT
    r.rule_id,
    r.trigger_class_name,
    r.target_class_name,
    r.risk_level,
    r.rule_authority,
    COUNT(DISTINCT cr1.concept_id_1) AS trigger_drug_count,
    COUNT(DISTINCT cr2.concept_id_1) AS target_drug_count,
    COUNT(*) AS total_pairs_covered
FROM ddi_constitutional_rules r
LEFT JOIN ohdsi_concept_relationship cr1
    ON r.trigger_concept_id = cr1.concept_id_2
    AND cr1.relationship_id = 'Drug has drug class'
LEFT JOIN ohdsi_concept_relationship cr2
    ON r.target_concept_id = cr2.concept_id_2
    AND cr2.relationship_id = 'Drug has drug class'
WHERE r.active = TRUE
GROUP BY r.rule_id, r.trigger_class_name, r.target_class_name, r.risk_level, r.rule_authority
ORDER BY total_pairs_covered DESC;

-- =============================================================================
-- 6. DDI Check Function (Runtime Execution)
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
    rule_authority VARCHAR(100)
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
        v.rule_authority
    FROM v_active_ddi_definitions v
    WHERE v.trigger_drug_id = ANY(drug_concept_ids)
      AND v.target_drug_id = ANY(drug_concept_ids)
      AND v.trigger_drug_id != v.target_drug_id
    ORDER BY
        CASE v.risk_level
            WHEN 'CRITICAL' THEN 1
            WHEN 'HIGH' THEN 2
            WHEN 'WARNING' THEN 3
            ELSE 4
        END,
        v.rule_id;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- 7. Populate ONC Constitutional Rules (From kb5_canonical_ddi_rules.csv)
-- =============================================================================

INSERT INTO ddi_constitutional_rules (
    rule_id, trigger_class_name, trigger_concept_id, target_class_name, target_concept_id,
    risk_level, description, context_loinc_id, context_loinc_name, context_threshold_val,
    context_logic_operator, context_required, rule_authority, rule_version
) VALUES
-- MAOI Cluster (Rules 1-5) - Absolute Contraindications
(1, 'MAO Inhibitors', 21603933, 'Amphetamine Derivatives', 21604321, 'CRITICAL', 'Hypertensive crisis risk - absolute contraindication', NULL, NULL, NULL, NULL, FALSE, 'ONC-Phansalkar-2012', 'v1.0'),
(2, 'MAO Inhibitors', 21603933, 'SSRIs', 21604044, 'CRITICAL', 'Serotonin syndrome - potentially fatal. Allow 2-5 week washout', NULL, NULL, NULL, NULL, FALSE, 'ONC-Phansalkar-2012', 'v1.0'),
(3, 'MAO Inhibitors', 21603933, 'Tricyclic Antidepressants', 21602472, 'CRITICAL', 'Hypertensive crisis and serotonin syndrome risk', NULL, NULL, NULL, NULL, FALSE, 'ONC-Phansalkar-2012', 'v1.0'),
(4, 'MAO Inhibitors', 21603933, 'Meperidine', 21604271, 'CRITICAL', 'Severe potentially fatal reactions including hyperpyrexia', NULL, NULL, NULL, NULL, FALSE, 'ONC-Phansalkar-2012', 'v1.0'),
(5, 'MAO Inhibitors', 21603933, 'Serotonergic Agents', 21604189, 'CRITICAL', 'Serotonin syndrome - monitor and avoid combination', NULL, NULL, NULL, NULL, FALSE, 'ONC-Phansalkar-2012', 'v1.0'),

-- Vitamin K Antagonist Cluster (Rules 6-8) - INR Context
(6, 'Vitamin K Antagonists', 21602722, 'NSAIDs', 21603931, 'HIGH', 'Increased bleeding risk (GI and Platelet) - Monitor INR closely', '5902-2', 'INR', 3.5, '>', FALSE, 'ONC-Phansalkar-2012', 'v1.1'),
(7, 'Vitamin K Antagonists', 21602722, 'Sulfonamide Antibiotics', 21602523, 'CRITICAL', 'Significant INR elevation risk - Consider alternative', '5902-2', 'INR', 3.5, '>', FALSE, 'ONC-Phansalkar-2012', 'v1.1'),
(8, 'Vitamin K Antagonists', 21602722, 'Azole Antifungals', 21601668, 'CRITICAL', 'Significant INR elevation risk - Consider alternative', '5902-2', 'INR', 3.5, '>', FALSE, 'ONC-Phansalkar-2012', 'v1.1'),

-- QT Prolongation (Rule 9) - Self-Class Rule
(9, 'QT Prolonging Agents', 21604254, 'QT Prolonging Agents', 21604254, 'WARNING', 'Additive QT prolongation - torsades de pointes risk', '8636-3', 'QTc interval', 500, '>', FALSE, 'ONC-Phansalkar-2012', 'v1.0'),

-- Diuretic + Cardiac/Lithium (Rules 10-11) - Context Required
(10, 'Loop Diuretics', 21604258, 'Cardiac Glycosides', 21602854, 'WARNING', 'Digitalis toxicity from hypokalemia', '2823-3', 'Potassium', 3.5, '<', TRUE, 'ONC-Phansalkar-2012', 'v1.0'),
(11, 'Thiazide Diuretics', 21604259, 'Lithium Salts', 21603684, 'HIGH', 'Reduced lithium clearance causing toxicity', '14879-1', 'Lithium level', 1.2, '>', TRUE, 'ONC-Phansalkar-2012', 'v1.0'),

-- RAAS + Potassium (Rules 12-14) - Context Required
(12, 'ACE Inhibitors', 21600046, 'Potassium-Sparing Diuretics', 21604225, 'HIGH', 'Additive hyperkalemia risk from dual RAAS effect', '2823-3', 'Potassium', 5.5, '>', TRUE, 'ONC-Phansalkar-2012', 'v1.0'),
(13, 'ACE Inhibitors', 21600046, 'Potassium Supplements', 21604226, 'HIGH', 'Hyperkalemia risk - monitor potassium closely', '2823-3', 'Potassium', 5.5, '>', TRUE, 'ONC-Phansalkar-2012', 'v1.0'),
(14, 'ARBs', 21601664, 'Potassium-Sparing Diuretics', 21604225, 'HIGH', 'Hyperkalemia from dual RAAS blockade', '2823-3', 'Potassium', 5.5, '>', TRUE, 'ONC-Phansalkar-2012', 'v1.0'),

-- Immunosuppression (Rule 15)
(15, 'Anti-TNF Agents', 21601639, 'Abatacept', 21600016, 'HIGH', 'Severe immunosuppression and infection risk', NULL, NULL, NULL, NULL, FALSE, 'ONC-Phansalkar-2012', 'v1.0'),

-- Extended Rules (16-25) - Non-ONC Core
(16, 'Non-Selective Beta Blockers', 21602170, 'Beta-2 Agonists', 21601785, 'WARNING', 'Pharmacodynamic antagonism reducing bronchodilation', NULL, NULL, NULL, NULL, FALSE, 'ONC-Derived', 'v1.0'),
(17, 'Carbapenems', 21601918, 'Valproic Acid', 21604422, 'CRITICAL', 'Carbapenems reduce valproate levels by 60-90%', '3184-9', 'Valproic acid level', 50, '<', TRUE, 'Post-ONC-Critical', 'v1.0'),
(18, 'PDE-5 Inhibitors', 21604207, 'Nitrates', 21603934, 'CRITICAL', 'Severe hypotension - contraindicated', '8480-6', 'Systolic BP', 90, '<', FALSE, 'ONC-Derived', 'v1.0'),
(19, 'Strong CYP3A4 Inhibitors', 21602722, 'Statins CYP3A4 Metabolized', 21601855, 'HIGH', 'Myopathy and rhabdomyolysis risk from increased statin exposure', '2157-6', 'Creatine kinase', 1000, '>', FALSE, 'Post-ONC-Critical', 'v1.0'),
(20, 'SSRIs', 21604044, 'Pimozide', 21604211, 'CRITICAL', 'QT prolongation from CYP2D6 inhibition - fatal arrhythmia risk', '8636-3', 'QTc interval', 500, '>', FALSE, 'FDA-Boxed-Warning', 'v1.0'),
(21, 'SSRIs', 21604044, 'Thioridazine', 21604349, 'CRITICAL', 'QT prolongation - torsades de pointes risk', '8636-3', 'QTc interval', 500, '>', FALSE, 'FDA-Boxed-Warning', 'v1.0'),
(22, 'Opioids', 21604182, 'Benzodiazepines', 21601779, 'CRITICAL', 'Additive CNS/respiratory depression - FDA boxed warning', NULL, NULL, NULL, NULL, FALSE, 'FDA-Boxed-Warning', 'v1.0'),
(23, 'Opioids', 21604182, 'Gabapentinoids', 955006, 'HIGH', 'Enhanced respiratory depression risk', NULL, NULL, NULL, NULL, FALSE, 'FDA-Boxed-Warning', 'v1.0'),
(24, 'Macrolide Antibiotics', 21603698, 'Colchicine', 21604141, 'CRITICAL', 'CYP3A4 inhibition dramatically increases colchicine levels - fatal cases', NULL, NULL, NULL, NULL, FALSE, 'Post-ONC-Critical', 'v1.0'),
(25, 'CYP2C19 Inhibitors', 21602723, 'Clopidogrel', 21602465, 'HIGH', 'Reduced clopidogrel activation - decreased antiplatelet effect', NULL, NULL, NULL, NULL, FALSE, 'Post-ONC-Critical', 'v1.0');

-- Reset sequence
SELECT setval('ddi_constitutional_rules_rule_id_seq', 25, true);

-- =============================================================================
-- 8. Validation Queries (Run After OHDSI Data Import)
-- =============================================================================

-- Cardinality check (run after importing OHDSI data)
-- SELECT * FROM v_ddi_expansion_stats ORDER BY total_pairs_covered DESC;

-- Spot check: Warfarin + Ibuprofen
-- SELECT * FROM v_active_ddi_definitions
-- WHERE trigger_drug_name ILIKE '%warfarin%'
--   AND target_drug_name ILIKE '%ibuprofen%';

-- Spot check: QT + QT rule expansion
-- SELECT * FROM v_ddi_expansion_stats WHERE rule_id = 9;

COMMENT ON TABLE ddi_constitutional_rules IS 'ONC Constitutional DDI Rules - The Clinical Safety Constitution. Class-based rules expanded via OHDSI at runtime.';
COMMENT ON TABLE ohdsi_concept IS 'OHDSI Vocabulary concepts (drugs and drug classes) for class expansion.';
COMMENT ON TABLE ohdsi_concept_relationship IS 'OHDSI Drug → Class relationships (73,842 mappings) for constitutional rule expansion.';
COMMENT ON VIEW v_active_ddi_definitions IS 'Runtime expansion of constitutional rules to drug pairs via OHDSI class membership.';
COMMENT ON FUNCTION check_constitutional_ddi IS 'Runtime DDI check using constitutional rules with class expansion.';
