-- =============================================================================
-- MIGRATION 004: TRUTH ARBITRATION ENGINE
-- Phase 3d: Conflict Resolution Between Rules, Authorities, and Interpretations
-- =============================================================================
-- "When truths collide, precedence decides."
-- The final arbiter before clinical action.
--
-- Precedence Hierarchy:
--   1. REGULATORY (FDA Black Box, REMS) - Trust: 1.00
--   2. AUTHORITY (CPIC, CredibleMeds, LactMed) - Trust: 1.00
--   3. LAB (KB-16 interpretations) - Trust: 0.95
--   4. RULE (Phase 3b.5 canonical rules) - Trust: 0.90
--   5. LOCAL (Hospital policies) - Trust: 0.80
-- =============================================================================

-- =============================================================================
-- 1. ENUMS FOR TRUTH ARBITRATION
-- =============================================================================

-- Source types in precedence order
CREATE TYPE source_type AS ENUM (
    'REGULATORY',    -- FDA Black Box, REMS, Contraindications
    'AUTHORITY',     -- CPIC, CredibleMeds, LactMed
    'LAB',           -- KB-16 lab interpretations
    'RULE',          -- Phase 3b.5 canonical rules
    'LOCAL'          -- Hospital formulary policies
);

-- Decision outcomes
CREATE TYPE decision_type AS ENUM (
    'ACCEPT',        -- All sources agree or no conflicts. Proceed.
    'BLOCK',         -- Hard constraint violated. Cannot proceed.
    'OVERRIDE',      -- Soft conflict. Can proceed with acknowledgment.
    'DEFER',         -- Insufficient data. Need more information.
    'ESCALATE'       -- Complex conflict. Requires human review.
);

-- Conflict types
CREATE TYPE conflict_type AS ENUM (
    'RULE_VS_AUTHORITY',       -- SPL says "avoid", CPIC says "contraindicated"
    'RULE_VS_LAB',             -- Rule: CrCl < 30, Lab: eGFR = 28
    'AUTHORITY_VS_LAB',        -- CPIC: eGFR < 30, Lab: eGFR normal for pregnancy
    'AUTHORITY_VS_AUTHORITY',  -- CPIC vs CredibleMeds on same drug
    'RULE_VS_RULE',            -- Two SPLs have different thresholds
    'LOCAL_VS_ANY'             -- Hospital policy overrides guideline
);

-- Authority levels
CREATE TYPE authority_level AS ENUM (
    'DEFINITIVE',    -- Highest evidence (CPIC 1A, FDA contraindication)
    'PRIMARY',       -- Strong evidence (CPIC 1B, major guidelines)
    'SECONDARY',     -- Moderate evidence (expert consensus)
    'TERTIARY'       -- Limited evidence (case reports, local practice)
);

-- Clinical effect types
CREATE TYPE clinical_effect AS ENUM (
    'CONTRAINDICATED',   -- Must not use
    'AVOID',             -- Should not use unless no alternatives
    'CAUTION',           -- Use with monitoring
    'REDUCE_DOSE',       -- Dose adjustment required
    'MONITOR',           -- Enhanced monitoring required
    'ALLOW',             -- Safe to proceed
    'NO_EFFECT'          -- No clinical impact
);

-- =============================================================================
-- 2. PRECEDENCE RULES TABLE
-- =============================================================================
-- Configurable P1-P7 rules for conflict resolution

CREATE TABLE precedence_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_code VARCHAR(10) NOT NULL UNIQUE,  -- P1, P2, P3...
    rule_name VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    priority INTEGER NOT NULL,              -- Lower = higher priority
    source_a source_type,                   -- First source in comparison
    source_b source_type,                   -- Second source in comparison
    winner source_type,                     -- Which source wins
    special_condition TEXT,                 -- Additional logic conditions
    rationale TEXT NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_precedence_rules_priority ON precedence_rules(priority);
CREATE INDEX idx_precedence_rules_code ON precedence_rules(rule_code);

-- =============================================================================
-- 3. REGULATORY BLOCKS TABLE
-- =============================================================================
-- FDA Black Box Warnings, REMS, Hard Contraindications

CREATE TABLE regulatory_blocks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    drug_rxcui VARCHAR(20) NOT NULL,
    drug_name VARCHAR(255) NOT NULL,
    block_type VARCHAR(50) NOT NULL,        -- BLACK_BOX, REMS, CONTRAINDICATION
    condition_description TEXT NOT NULL,
    affected_population TEXT,               -- Who is affected
    clinical_effect clinical_effect NOT NULL DEFAULT 'CONTRAINDICATED',
    severity VARCHAR(20) NOT NULL DEFAULT 'CRITICAL',
    source_url TEXT,
    fda_label_date DATE,
    effective_date DATE NOT NULL,
    expiration_date DATE,
    trust_level DECIMAL(3,2) DEFAULT 1.00,  -- Always 1.00 for regulatory
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT chk_trust_level CHECK (trust_level = 1.00)
);

CREATE INDEX idx_regulatory_blocks_rxcui ON regulatory_blocks(drug_rxcui);
CREATE INDEX idx_regulatory_blocks_type ON regulatory_blocks(block_type);
CREATE INDEX idx_regulatory_blocks_effective ON regulatory_blocks(effective_date);

-- =============================================================================
-- 4. AUTHORITY FACTS TABLE
-- =============================================================================
-- CPIC, CredibleMeds, LactMed assertions

CREATE TABLE authority_facts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    authority VARCHAR(50) NOT NULL,         -- CPIC, CREDIBLEMEDS, LACTMED, ATA, KDIGO
    authority_level authority_level NOT NULL,
    drug_rxcui VARCHAR(20),
    drug_name VARCHAR(255),
    gene_symbol VARCHAR(50),                -- For pharmacogenomics (CYP2C19, CYP2D6)
    phenotype VARCHAR(100),                 -- Poor metabolizer, intermediate, etc.
    condition_code VARCHAR(50),             -- ICD-10 or clinical condition
    condition_name VARCHAR(255),
    assertion TEXT NOT NULL,                -- What the authority says
    clinical_effect clinical_effect NOT NULL,
    evidence_level VARCHAR(10),             -- 1A, 1B, 2A, 2B, etc.
    recommendation TEXT,
    dosing_guidance TEXT,
    monitoring_requirements TEXT,
    source_url TEXT,
    source_pmid VARCHAR(20),                -- PubMed ID
    guideline_version VARCHAR(20),
    last_reviewed DATE,
    effective_date DATE NOT NULL,
    expiration_date DATE,
    trust_level DECIMAL(3,2) DEFAULT 1.00,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_authority_facts_authority ON authority_facts(authority);
CREATE INDEX idx_authority_facts_rxcui ON authority_facts(drug_rxcui);
CREATE INDEX idx_authority_facts_gene ON authority_facts(gene_symbol);
CREATE INDEX idx_authority_facts_level ON authority_facts(authority_level);
CREATE INDEX idx_authority_facts_effect ON authority_facts(clinical_effect);

-- =============================================================================
-- 5. LOCAL POLICIES TABLE
-- =============================================================================
-- Hospital/institution-specific overrides

CREATE TABLE local_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    institution_id VARCHAR(50) NOT NULL,    -- Hospital/clinic identifier
    institution_name VARCHAR(255) NOT NULL,
    policy_code VARCHAR(50) NOT NULL,
    policy_name VARCHAR(255) NOT NULL,
    drug_rxcui VARCHAR(20),
    drug_class VARCHAR(100),                -- Can apply to drug classes
    condition_code VARCHAR(50),
    condition_description TEXT,
    override_target source_type NOT NULL,   -- What this policy overrides
    clinical_effect clinical_effect NOT NULL,
    justification TEXT NOT NULL,            -- Why this override exists
    restrictions TEXT,                      -- Under what conditions
    approval_required BOOLEAN DEFAULT TRUE,
    approved_by VARCHAR(255),
    approval_date DATE,
    effective_date DATE NOT NULL,
    expiration_date DATE,
    trust_level DECIMAL(3,2) DEFAULT 0.80,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    CONSTRAINT chk_local_trust CHECK (trust_level <= 0.80),
    CONSTRAINT chk_override_target CHECK (override_target IN ('RULE', 'LOCAL'))
);

CREATE INDEX idx_local_policies_institution ON local_policies(institution_id);
CREATE INDEX idx_local_policies_rxcui ON local_policies(drug_rxcui);
CREATE INDEX idx_local_policies_active ON local_policies(is_active);

-- =============================================================================
-- 6. CONFLICTS DETECTED TABLE
-- =============================================================================
-- Record of all conflicts found during arbitration

CREATE TABLE conflicts_detected (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    arbitration_id UUID NOT NULL,           -- Links to parent arbitration
    conflict_type conflict_type NOT NULL,
    source_a_type source_type NOT NULL,
    source_a_id UUID,                       -- Reference to specific assertion
    source_a_assertion TEXT NOT NULL,
    source_a_effect clinical_effect,
    source_b_type source_type NOT NULL,
    source_b_id UUID,
    source_b_assertion TEXT NOT NULL,
    source_b_effect clinical_effect,
    resolution_winner source_type,
    resolution_rule VARCHAR(10),            -- P1, P2, P3...
    resolution_rationale TEXT,
    severity VARCHAR(20) DEFAULT 'MEDIUM',  -- LOW, MEDIUM, HIGH, CRITICAL
    detected_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_conflicts_arbitration ON conflicts_detected(arbitration_id);
CREATE INDEX idx_conflicts_type ON conflicts_detected(conflict_type);
CREATE INDEX idx_conflicts_severity ON conflicts_detected(severity);

-- =============================================================================
-- 7. ARBITRATION DECISIONS TABLE
-- =============================================================================
-- Complete audit log of all arbitration decisions

CREATE TABLE arbitration_decisions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Clinical context
    drug_rxcui VARCHAR(20) NOT NULL,
    drug_name VARCHAR(255),
    patient_id VARCHAR(100),                -- Anonymized or reference
    clinical_intent VARCHAR(50) NOT NULL,   -- PRESCRIBE, CONTINUE, MODIFY, DISCONTINUE

    -- Patient context snapshot
    patient_age INTEGER,
    patient_gender VARCHAR(1),
    patient_pregnant BOOLEAN,
    patient_trimester INTEGER,
    patient_egfr DECIMAL(6,2),
    patient_crcl DECIMAL(6,2),
    patient_ckd_stage INTEGER,
    patient_child_pugh VARCHAR(1),
    patient_genotype JSONB,                 -- Pharmacogenomic data

    -- Decision outcome
    decision decision_type NOT NULL,
    confidence DECIMAL(3,2) NOT NULL,       -- 0.00-1.00
    winning_source source_type,
    winning_assertion_id UUID,
    precedence_rule_applied VARCHAR(10),    -- P1, P2, P3...

    -- Conflict summary
    conflict_count INTEGER DEFAULT 0,
    conflicts_summary JSONB,                -- Summary of all conflicts

    -- Recommendations
    recommended_action TEXT NOT NULL,
    clinical_rationale TEXT NOT NULL,
    alternative_actions JSONB,              -- Other options if applicable

    -- Governance
    input_hash VARCHAR(64) NOT NULL,        -- SHA256 of all inputs
    arbitrated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    arbitrated_by VARCHAR(100) DEFAULT 'SYSTEM',

    -- Override tracking
    was_overridden BOOLEAN DEFAULT FALSE,
    overridden_by VARCHAR(255),
    override_reason TEXT,
    override_at TIMESTAMP WITH TIME ZONE,

    -- Audit
    audit_trail JSONB NOT NULL,             -- Complete decision trace
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_arbitration_drug ON arbitration_decisions(drug_rxcui);
CREATE INDEX idx_arbitration_patient ON arbitration_decisions(patient_id);
CREATE INDEX idx_arbitration_decision ON arbitration_decisions(decision);
CREATE INDEX idx_arbitration_timestamp ON arbitration_decisions(arbitrated_at);
CREATE INDEX idx_arbitration_overridden ON arbitration_decisions(was_overridden);

-- =============================================================================
-- 8. AUDIT ENTRIES TABLE
-- =============================================================================
-- Detailed step-by-step audit trail

CREATE TABLE arbitration_audit_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    arbitration_id UUID NOT NULL REFERENCES arbitration_decisions(id),
    step_number INTEGER NOT NULL,
    step_name VARCHAR(100) NOT NULL,
    step_description TEXT,
    inputs JSONB,
    outputs JSONB,
    duration_ms INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_audit_entries_arbitration ON arbitration_audit_entries(arbitration_id);
CREATE INDEX idx_audit_entries_step ON arbitration_audit_entries(step_number);

-- =============================================================================
-- 9. SEED DATA: PRECEDENCE RULES (P1-P7)
-- =============================================================================

INSERT INTO precedence_rules (rule_code, rule_name, description, priority, source_a, source_b, winner, special_condition, rationale)
VALUES
    ('P1', 'Regulatory Always Wins',
     'REGULATORY_BLOCK always wins over any other source',
     1, 'REGULATORY', NULL, 'REGULATORY',
     NULL,
     'Legal requirement - FDA Black Box warnings and REMS programs cannot be overridden'),

    ('P2', 'Authority Hierarchy',
     'DEFINITIVE authority level beats PRIMARY authority level',
     2, 'AUTHORITY', 'AUTHORITY', NULL,
     'Compare authority_level: DEFINITIVE > PRIMARY > SECONDARY > TERTIARY',
     'Evidence hierarchy - higher evidence grades take precedence'),

    ('P3', 'Authority Over Rule',
     'AUTHORITY_FACT beats CANONICAL_RULE for the same drug',
     3, 'AUTHORITY', 'RULE', 'AUTHORITY',
     'Same drug_rxcui',
     'Curated expert consensus (CPIC) is more authoritative than extracted SPL rules'),

    ('P4', 'Lab Critical Escalation',
     'LAB critical interpretation + RULE triggered = ESCALATE',
     4, 'LAB', 'RULE', NULL,
     'LAB.interpretation = CRITICAL AND RULE.triggered = TRUE',
     'Real-time lab validation of rule triggers requires human review'),

    ('P5', 'Provenance Consensus',
     'Source with more provenance agreement wins ties',
     5, NULL, NULL, NULL,
     'Compare provenance_count',
     'Consensus strength - more sources agreeing indicates higher reliability'),

    ('P6', 'Local Policy Limits',
     'LOCAL_POLICY can override RULE but NOT AUTHORITY',
     6, 'LOCAL', 'RULE', 'LOCAL',
     'LOCAL cannot override AUTHORITY or REGULATORY',
     'Site autonomy for formulary decisions while maintaining safety'),

    ('P7', 'Restrictive Wins Ties',
     'More restrictive clinical effect wins in ties',
     7, NULL, NULL, NULL,
     'Compare clinical_effect: CONTRAINDICATED > AVOID > CAUTION > REDUCE_DOSE > MONITOR',
     'Fail-safe default - when uncertain, choose the safer option');

-- =============================================================================
-- 10. SEED DATA: SAMPLE REGULATORY BLOCKS
-- =============================================================================

INSERT INTO regulatory_blocks (drug_rxcui, drug_name, block_type, condition_description, affected_population, clinical_effect, severity, fda_label_date, effective_date)
VALUES
    -- Metformin Black Box
    ('6809', 'Metformin', 'BLACK_BOX',
     'Lactic acidosis risk with renal impairment. Contraindicated in patients with eGFR < 30 mL/min/1.73m².',
     'Patients with severe renal impairment (eGFR < 30)',
     'CONTRAINDICATED', 'CRITICAL',
     '2024-01-15', '2024-01-15'),

    -- Warfarin Black Box
    ('11289', 'Warfarin', 'BLACK_BOX',
     'Major or fatal bleeding risk. Regular INR monitoring required.',
     'All patients',
     'MONITOR', 'HIGH',
     '2023-06-01', '2023-06-01'),

    -- Isotretinoin REMS
    ('6064', 'Isotretinoin', 'REMS',
     'iPLEDGE REMS program required due to teratogenicity.',
     'All patients, especially females of childbearing potential',
     'CONTRAINDICATED', 'CRITICAL',
     '2024-03-01', '2024-03-01'),

    -- Clozapine REMS
    ('2626', 'Clozapine', 'REMS',
     'Clozapine REMS program required due to severe neutropenia risk.',
     'All patients',
     'MONITOR', 'HIGH',
     '2024-01-01', '2024-01-01'),

    -- Thalidomide REMS
    ('10219', 'Thalidomide', 'REMS',
     'THALOMID REMS program required due to teratogenicity and VTE risk.',
     'All patients, especially females of childbearing potential',
     'CONTRAINDICATED', 'CRITICAL',
     '2023-09-01', '2023-09-01');

-- =============================================================================
-- 11. SEED DATA: SAMPLE AUTHORITY FACTS (CPIC)
-- =============================================================================

INSERT INTO authority_facts (authority, authority_level, drug_rxcui, drug_name, gene_symbol, phenotype, assertion, clinical_effect, evidence_level, recommendation, dosing_guidance, effective_date)
VALUES
    -- Clopidogrel + CYP2C19
    ('CPIC', 'DEFINITIVE', '32968', 'Clopidogrel', 'CYP2C19', 'Poor Metabolizer',
     'CYP2C19 poor metabolizers have reduced clopidogrel activation and increased cardiovascular risk.',
     'AVOID', '1A',
     'Use alternative antiplatelet agent (prasugrel, ticagrelor) if not contraindicated.',
     'If clopidogrel must be used, consider higher loading dose (600mg) and maintenance dose (150mg).',
     '2024-01-01'),

    ('CPIC', 'DEFINITIVE', '32968', 'Clopidogrel', 'CYP2C19', 'Intermediate Metabolizer',
     'CYP2C19 intermediate metabolizers have reduced clopidogrel activation.',
     'CAUTION', '1A',
     'Consider alternative antiplatelet agent or monitor closely.',
     'Standard dosing may be used with enhanced monitoring.',
     '2024-01-01'),

    -- Codeine + CYP2D6
    ('CPIC', 'DEFINITIVE', '2670', 'Codeine', 'CYP2D6', 'Ultra-rapid Metabolizer',
     'CYP2D6 ultra-rapid metabolizers convert codeine to morphine rapidly, increasing toxicity risk.',
     'AVOID', '1A',
     'Avoid codeine. Use alternative analgesic not metabolized by CYP2D6.',
     NULL,
     '2024-01-01'),

    ('CPIC', 'DEFINITIVE', '2670', 'Codeine', 'CYP2D6', 'Poor Metabolizer',
     'CYP2D6 poor metabolizers have minimal codeine-to-morphine conversion with reduced efficacy.',
     'AVOID', '1A',
     'Avoid codeine due to lack of efficacy. Use alternative analgesic.',
     NULL,
     '2024-01-01'),

    -- Simvastatin + SLCO1B1
    ('CPIC', 'DEFINITIVE', '36567', 'Simvastatin', 'SLCO1B1', '521TC or 521CC',
     'SLCO1B1 521T>C carriers have increased simvastatin exposure and myopathy risk.',
     'REDUCE_DOSE', '1A',
     'Use lower simvastatin dose or alternative statin.',
     '521TC: Prescribe ≤20mg/day. 521CC: Prescribe ≤10mg/day or use alternative statin.',
     '2024-01-01'),

    -- Metformin + Renal Function
    ('CPIC', 'DEFINITIVE', '6809', 'Metformin', NULL, NULL,
     'Metformin contraindicated with eGFR < 30 mL/min/1.73m² due to lactic acidosis risk.',
     'CONTRAINDICATED', '1A',
     'Do not initiate if eGFR < 30. Discontinue if eGFR falls below 30.',
     'eGFR 30-45: Reduce dose to 50%. eGFR 45-60: Monitor renal function closely. eGFR < 30: Contraindicated.',
     '2024-01-01');

-- =============================================================================
-- 12. SEED DATA: CREDIBLEMEDS FACTS
-- =============================================================================

INSERT INTO authority_facts (authority, authority_level, drug_rxcui, drug_name, condition_code, condition_name, assertion, clinical_effect, evidence_level, recommendation, monitoring_requirements, effective_date)
VALUES
    -- QT Prolongation Known Risk
    ('CREDIBLEMEDS', 'DEFINITIVE', '2551', 'Cisapride', 'I45.81', 'Long QT syndrome',
     'Cisapride has known risk of Torsades de Pointes.',
     'AVOID', 'KNOWN_RISK',
     'Avoid in patients with long QT syndrome or concomitant QT-prolonging drugs.',
     'ECG monitoring required if use cannot be avoided.',
     '2024-01-01'),

    ('CREDIBLEMEDS', 'PRIMARY', '4441', 'Erythromycin', 'I45.81', 'Long QT syndrome',
     'Erythromycin has possible risk of Torsades de Pointes.',
     'CAUTION', 'POSSIBLE_RISK',
     'Use with caution in patients with QT prolongation risk factors.',
     'Consider ECG monitoring in high-risk patients.',
     '2024-01-01'),

    ('CREDIBLEMEDS', 'DEFINITIVE', '5093', 'Haloperidol', 'I45.81', 'Long QT syndrome',
     'IV Haloperidol has known risk of Torsades de Pointes.',
     'AVOID', 'KNOWN_RISK',
     'Avoid IV administration. If essential, continuous ECG monitoring required.',
     'Continuous ECG monitoring with IV use. Check QTc before and during therapy.',
     '2024-01-01');

-- =============================================================================
-- 13. SEED DATA: LACTMED FACTS
-- =============================================================================

INSERT INTO authority_facts (authority, authority_level, drug_rxcui, drug_name, condition_name, assertion, clinical_effect, recommendation, effective_date)
VALUES
    -- Methotrexate - Lactation
    ('LACTMED', 'DEFINITIVE', '6851', 'Methotrexate', 'Breastfeeding',
     'Methotrexate is present in breast milk. Contraindicated during breastfeeding.',
     'CONTRAINDICATED',
     'Do not use during breastfeeding. Withhold breastfeeding for at least 1 week after last dose.',
     '2024-01-01'),

    -- Ibuprofen - Lactation (Safe)
    ('LACTMED', 'PRIMARY', '5640', 'Ibuprofen', 'Breastfeeding',
     'Ibuprofen is excreted in breast milk in small amounts. Compatible with breastfeeding.',
     'ALLOW',
     'Short-term use acceptable. Monitor infant for unusual irritability or drowsiness.',
     '2024-01-01'),

    -- Amiodarone - Lactation
    ('LACTMED', 'DEFINITIVE', '703', 'Amiodarone', 'Breastfeeding',
     'Amiodarone and its metabolite are present in breast milk. High iodine content may affect infant thyroid.',
     'AVOID',
     'Avoid during breastfeeding. If essential, monitor infant thyroid function.',
     '2024-01-01');

-- =============================================================================
-- 14. VIEWS FOR COMMON QUERIES
-- =============================================================================

-- Active regulatory blocks by drug
CREATE VIEW v_active_regulatory_blocks AS
SELECT
    rb.*,
    CASE
        WHEN rb.block_type = 'BLACK_BOX' THEN 1
        WHEN rb.block_type = 'REMS' THEN 2
        ELSE 3
    END AS block_priority
FROM regulatory_blocks rb
WHERE rb.effective_date <= CURRENT_DATE
  AND (rb.expiration_date IS NULL OR rb.expiration_date > CURRENT_DATE);

-- Active authority facts by drug
CREATE VIEW v_active_authority_facts AS
SELECT
    af.*,
    CASE af.authority_level
        WHEN 'DEFINITIVE' THEN 1
        WHEN 'PRIMARY' THEN 2
        WHEN 'SECONDARY' THEN 3
        WHEN 'TERTIARY' THEN 4
    END AS level_priority
FROM authority_facts af
WHERE af.effective_date <= CURRENT_DATE
  AND (af.expiration_date IS NULL OR af.expiration_date > CURRENT_DATE);

-- Recent arbitration decisions summary
CREATE VIEW v_recent_arbitrations AS
SELECT
    ad.id,
    ad.drug_name,
    ad.clinical_intent,
    ad.decision,
    ad.confidence,
    ad.winning_source,
    ad.precedence_rule_applied,
    ad.conflict_count,
    ad.was_overridden,
    ad.arbitrated_at
FROM arbitration_decisions ad
WHERE ad.arbitrated_at > NOW() - INTERVAL '30 days'
ORDER BY ad.arbitrated_at DESC;

-- Conflict resolution statistics
CREATE VIEW v_conflict_statistics AS
SELECT
    conflict_type,
    resolution_rule,
    COUNT(*) AS occurrence_count,
    AVG(CASE severity
        WHEN 'LOW' THEN 1
        WHEN 'MEDIUM' THEN 2
        WHEN 'HIGH' THEN 3
        WHEN 'CRITICAL' THEN 4
    END) AS avg_severity_score
FROM conflicts_detected
WHERE detected_at > NOW() - INTERVAL '90 days'
GROUP BY conflict_type, resolution_rule
ORDER BY occurrence_count DESC;

-- =============================================================================
-- 15. FUNCTIONS FOR ARBITRATION
-- =============================================================================

-- Function to get effective clinical effect priority (lower = more restrictive)
CREATE OR REPLACE FUNCTION get_effect_priority(effect clinical_effect)
RETURNS INTEGER AS $$
BEGIN
    RETURN CASE effect
        WHEN 'CONTRAINDICATED' THEN 1
        WHEN 'AVOID' THEN 2
        WHEN 'CAUTION' THEN 3
        WHEN 'REDUCE_DOSE' THEN 4
        WHEN 'MONITOR' THEN 5
        WHEN 'ALLOW' THEN 6
        WHEN 'NO_EFFECT' THEN 7
        ELSE 10
    END;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Function to determine winner between two clinical effects (P7: more restrictive wins)
CREATE OR REPLACE FUNCTION get_more_restrictive_effect(
    effect_a clinical_effect,
    effect_b clinical_effect
) RETURNS clinical_effect AS $$
BEGIN
    IF get_effect_priority(effect_a) <= get_effect_priority(effect_b) THEN
        RETURN effect_a;
    ELSE
        RETURN effect_b;
    END IF;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Function to check if a drug has regulatory blocks
CREATE OR REPLACE FUNCTION has_regulatory_block(p_drug_rxcui VARCHAR)
RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM v_active_regulatory_blocks
        WHERE drug_rxcui = p_drug_rxcui
    );
END;
$$ LANGUAGE plpgsql STABLE;

-- Function to get highest authority level for a drug
CREATE OR REPLACE FUNCTION get_highest_authority_level(p_drug_rxcui VARCHAR)
RETURNS authority_level AS $$
DECLARE
    v_level authority_level;
BEGIN
    SELECT authority_level INTO v_level
    FROM v_active_authority_facts
    WHERE drug_rxcui = p_drug_rxcui
    ORDER BY level_priority
    LIMIT 1;

    RETURN v_level;
END;
$$ LANGUAGE plpgsql STABLE;

-- =============================================================================
-- 16. TRIGGERS FOR AUDIT
-- =============================================================================

-- Trigger to update timestamps
CREATE OR REPLACE FUNCTION update_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_regulatory_blocks_timestamp
    BEFORE UPDATE ON regulatory_blocks
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER trg_authority_facts_timestamp
    BEFORE UPDATE ON authority_facts
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER trg_local_policies_timestamp
    BEFORE UPDATE ON local_policies
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();

CREATE TRIGGER trg_precedence_rules_timestamp
    BEFORE UPDATE ON precedence_rules
    FOR EACH ROW EXECUTE FUNCTION update_timestamp();

-- =============================================================================
-- 17. COMMENTS FOR DOCUMENTATION
-- =============================================================================

COMMENT ON TABLE precedence_rules IS 'P1-P7 precedence rules for conflict resolution in truth arbitration';
COMMENT ON TABLE regulatory_blocks IS 'FDA Black Box warnings, REMS programs, and hard contraindications';
COMMENT ON TABLE authority_facts IS 'CPIC, CredibleMeds, LactMed curated clinical assertions';
COMMENT ON TABLE local_policies IS 'Hospital/institution-specific policy overrides';
COMMENT ON TABLE conflicts_detected IS 'Record of conflicts found during arbitration';
COMMENT ON TABLE arbitration_decisions IS 'Complete audit log of all arbitration decisions';
COMMENT ON TABLE arbitration_audit_entries IS 'Step-by-step audit trail for each arbitration';

COMMENT ON TYPE source_type IS 'Precedence hierarchy: REGULATORY > AUTHORITY > LAB > RULE > LOCAL';
COMMENT ON TYPE decision_type IS 'Arbitration outcomes: ACCEPT, BLOCK, OVERRIDE, DEFER, ESCALATE';
COMMENT ON TYPE conflict_type IS 'Types of conflicts between different truth sources';
COMMENT ON TYPE authority_level IS 'Evidence hierarchy: DEFINITIVE > PRIMARY > SECONDARY > TERTIARY';
COMMENT ON TYPE clinical_effect IS 'Clinical action spectrum from CONTRAINDICATED to ALLOW';

-- =============================================================================
-- END OF MIGRATION 004
-- =============================================================================
