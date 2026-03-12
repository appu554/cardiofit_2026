-- KB-1 Drug Rules Database Schema
-- Supports full formulary (~40,000+ drugs) with multi-jurisdiction governance
-- Risk Level: CRITICAL - KB-1 computes doses that get administered

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";  -- For fast text search

-- =============================================================================
-- MAIN DRUG RULES TABLE
-- =============================================================================
-- Stores governed drug dosing rules with full provenance tracking
-- Each rule is uniquely identified by (rxnorm_code, jurisdiction)
CREATE TABLE drug_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    rxnorm_code VARCHAR(20) NOT NULL,
    jurisdiction VARCHAR(10) NOT NULL,  -- US, AU, IN, GLOBAL

    -- Drug identification
    drug_name VARCHAR(500) NOT NULL,
    generic_name VARCHAR(500),
    drug_class VARCHAR(200),
    atc_code VARCHAR(20),
    snomed_code VARCHAR(50),

    -- Full rule as JSONB (matches GovernedDrugRule struct)
    rule_data JSONB NOT NULL,

    -- Governance metadata (denormalized for fast queries)
    authority VARCHAR(50) NOT NULL,      -- FDA, TGA, CDSCO, WHO
    document_name VARCHAR(500),
    document_section VARCHAR(200),
    document_url TEXT,
    evidence_level VARCHAR(20),
    effective_date DATE,

    -- Ingestion tracking
    source_set_id VARCHAR(100),          -- FDA SetID or equivalent
    source_hash VARCHAR(64) NOT NULL,    -- SHA-256 of source document
    ingested_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    ingestion_run_id UUID,

    -- Approval tracking (YAML metadata based)
    approved_by VARCHAR(200),
    approved_at TIMESTAMP WITH TIME ZONE,
    version VARCHAR(20) NOT NULL,

    -- Safety flags (denormalized for fast filtering)
    is_high_alert BOOLEAN DEFAULT FALSE,
    is_narrow_ti BOOLEAN DEFAULT FALSE,
    has_black_box BOOLEAN DEFAULT FALSE,
    is_beers_list BOOLEAN DEFAULT FALSE,

    -- Audit
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Composite unique constraint
    CONSTRAINT uk_drug_rules_rxnorm_jurisdiction UNIQUE (rxnorm_code, jurisdiction)
);

-- =============================================================================
-- INDEXES FOR PERFORMANCE
-- =============================================================================
-- Primary lookup index
CREATE INDEX idx_drug_rules_rxnorm ON drug_rules(rxnorm_code);

-- Jurisdiction filtering
CREATE INDEX idx_drug_rules_jurisdiction ON drug_rules(jurisdiction);

-- Full-text search on drug names using trigram matching
CREATE INDEX idx_drug_rules_drug_name ON drug_rules USING gin(drug_name gin_trgm_ops);
CREATE INDEX idx_drug_rules_generic_name ON drug_rules USING gin(generic_name gin_trgm_ops);

-- Authority filtering (FDA, TGA, CDSCO)
CREATE INDEX idx_drug_rules_authority ON drug_rules(authority);

-- Safety flag filtering (partial index for efficiency)
CREATE INDEX idx_drug_rules_high_alert ON drug_rules(is_high_alert) WHERE is_high_alert = TRUE;
CREATE INDEX idx_drug_rules_black_box ON drug_rules(has_black_box) WHERE has_black_box = TRUE;
CREATE INDEX idx_drug_rules_narrow_ti ON drug_rules(is_narrow_ti) WHERE is_narrow_ti = TRUE;

-- JSONB index for rule_data queries
CREATE INDEX idx_drug_rules_rule_data ON drug_rules USING gin(rule_data);

-- Drug class filtering
CREATE INDEX idx_drug_rules_drug_class ON drug_rules(drug_class);

-- =============================================================================
-- INGESTION RUN LOGS
-- =============================================================================
-- Tracks each ingestion run with statistics and error tracking
CREATE TABLE ingestion_runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    authority VARCHAR(50) NOT NULL,
    jurisdiction VARCHAR(10) NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) DEFAULT 'RUNNING',  -- RUNNING, COMPLETED, FAILED

    -- Statistics
    total_drugs_processed INT DEFAULT 0,
    drugs_added INT DEFAULT 0,
    drugs_updated INT DEFAULT 0,
    drugs_unchanged INT DEFAULT 0,
    drugs_failed INT DEFAULT 0,

    -- Error tracking
    error_message TEXT,
    error_details JSONB,

    -- Audit
    triggered_by VARCHAR(200),
    trigger_type VARCHAR(50)  -- MANUAL, SCHEDULED
);

CREATE INDEX idx_ingestion_runs_authority ON ingestion_runs(authority);
CREATE INDEX idx_ingestion_runs_status ON ingestion_runs(status);
CREATE INDEX idx_ingestion_runs_started ON ingestion_runs(started_at DESC);

-- =============================================================================
-- INGESTION ITEM LOGS
-- =============================================================================
-- Per-drug tracking within an ingestion run
CREATE TABLE ingestion_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ingestion_run_id UUID REFERENCES ingestion_runs(id) ON DELETE CASCADE,
    rxnorm_code VARCHAR(20),
    drug_name VARCHAR(500),
    status VARCHAR(20),  -- SUCCESS, FAILED, SKIPPED
    action VARCHAR(20),  -- INSERT, UPDATE, UNCHANGED
    error_message TEXT,
    processing_time_ms INT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_ingestion_items_run ON ingestion_items(ingestion_run_id);
CREATE INDEX idx_ingestion_items_status ON ingestion_items(status);
CREATE INDEX idx_ingestion_items_rxnorm ON ingestion_items(rxnorm_code);

-- =============================================================================
-- CHANGE HISTORY FOR AUDIT TRAIL
-- =============================================================================
-- Complete audit trail of all drug rule changes
CREATE TABLE drug_rule_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    drug_rule_id UUID,  -- Can be NULL if rule was deleted
    rxnorm_code VARCHAR(20) NOT NULL,
    jurisdiction VARCHAR(10) NOT NULL,
    change_type VARCHAR(20) NOT NULL,  -- INSERT, UPDATE, DELETE
    old_rule_data JSONB,
    new_rule_data JSONB,
    changed_fields TEXT[],
    changed_by VARCHAR(200),
    change_reason TEXT,
    changed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_drug_rule_history_drug ON drug_rule_history(drug_rule_id);
CREATE INDEX idx_drug_rule_history_rxnorm ON drug_rule_history(rxnorm_code);
CREATE INDEX idx_drug_rule_history_change_type ON drug_rule_history(change_type);
CREATE INDEX idx_drug_rule_history_changed_at ON drug_rule_history(changed_at DESC);

-- =============================================================================
-- VIEW FOR QUICK LOOKUPS WITH GOVERNANCE
-- =============================================================================
CREATE VIEW v_governed_drug_rules AS
SELECT
    dr.id,
    dr.rxnorm_code,
    dr.jurisdiction,
    dr.drug_name,
    dr.generic_name,
    dr.drug_class,
    dr.rule_data,
    dr.authority,
    dr.document_name,
    dr.document_section,
    dr.document_url,
    dr.version,
    dr.approved_by,
    dr.approved_at,
    dr.is_high_alert,
    dr.is_narrow_ti,
    dr.has_black_box,
    dr.is_beers_list,
    dr.source_set_id,
    dr.source_hash,
    dr.ingested_at,
    dr.created_at,
    dr.updated_at,
    ir.status as ingestion_status,
    ir.triggered_by as ingestion_triggered_by
FROM drug_rules dr
LEFT JOIN ingestion_runs ir ON dr.ingestion_run_id = ir.id;

-- =============================================================================
-- VIEW FOR STATISTICS DASHBOARD
-- =============================================================================
CREATE VIEW v_drug_rule_stats AS
SELECT
    jurisdiction,
    authority,
    COUNT(*) as total_drugs,
    COUNT(*) FILTER (WHERE is_high_alert = TRUE) as high_alert_count,
    COUNT(*) FILTER (WHERE has_black_box = TRUE) as black_box_count,
    COUNT(*) FILTER (WHERE is_narrow_ti = TRUE) as narrow_ti_count,
    COUNT(*) FILTER (WHERE is_beers_list = TRUE) as beers_list_count,
    MAX(ingested_at) as last_ingestion,
    MIN(ingested_at) as first_ingestion
FROM drug_rules
GROUP BY jurisdiction, authority;

-- =============================================================================
-- TRIGGER: AUTO-UPDATE updated_at
-- =============================================================================
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tr_drug_rules_updated
    BEFORE UPDATE ON drug_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- =============================================================================
-- TRIGGER: LOG CHANGES TO HISTORY
-- =============================================================================
CREATE OR REPLACE FUNCTION log_drug_rule_change()
RETURNS TRIGGER AS $$
DECLARE
    v_changed_fields TEXT[];
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO drug_rule_history (
            drug_rule_id, rxnorm_code, jurisdiction, change_type,
            new_rule_data, changed_by, change_reason
        )
        VALUES (
            NEW.id, NEW.rxnorm_code, NEW.jurisdiction, 'INSERT',
            NEW.rule_data, NEW.approved_by, 'Initial ingestion'
        );
        RETURN NEW;
    ELSIF TG_OP = 'UPDATE' THEN
        -- Detect changed fields
        v_changed_fields := ARRAY[]::TEXT[];
        IF OLD.drug_name IS DISTINCT FROM NEW.drug_name THEN
            v_changed_fields := array_append(v_changed_fields, 'drug_name');
        END IF;
        IF OLD.generic_name IS DISTINCT FROM NEW.generic_name THEN
            v_changed_fields := array_append(v_changed_fields, 'generic_name');
        END IF;
        IF OLD.rule_data IS DISTINCT FROM NEW.rule_data THEN
            v_changed_fields := array_append(v_changed_fields, 'rule_data');
        END IF;
        IF OLD.is_high_alert IS DISTINCT FROM NEW.is_high_alert THEN
            v_changed_fields := array_append(v_changed_fields, 'is_high_alert');
        END IF;
        IF OLD.has_black_box IS DISTINCT FROM NEW.has_black_box THEN
            v_changed_fields := array_append(v_changed_fields, 'has_black_box');
        END IF;
        IF OLD.version IS DISTINCT FROM NEW.version THEN
            v_changed_fields := array_append(v_changed_fields, 'version');
        END IF;

        INSERT INTO drug_rule_history (
            drug_rule_id, rxnorm_code, jurisdiction, change_type,
            old_rule_data, new_rule_data, changed_fields,
            changed_by, change_reason
        )
        VALUES (
            NEW.id, NEW.rxnorm_code, NEW.jurisdiction, 'UPDATE',
            OLD.rule_data, NEW.rule_data, v_changed_fields,
            NEW.approved_by, 'Rule update via ingestion'
        );
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        INSERT INTO drug_rule_history (
            drug_rule_id, rxnorm_code, jurisdiction, change_type,
            old_rule_data, change_reason
        )
        VALUES (
            OLD.id, OLD.rxnorm_code, OLD.jurisdiction, 'DELETE',
            OLD.rule_data, 'Rule deleted'
        );
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tr_drug_rules_history
    AFTER INSERT OR UPDATE OR DELETE ON drug_rules
    FOR EACH ROW
    EXECUTE FUNCTION log_drug_rule_change();

-- =============================================================================
-- FUNCTION: GET RULE BY RXNORM WITH JURISDICTION FALLBACK
-- =============================================================================
-- Returns rule for specific jurisdiction, falling back to GLOBAL if not found
CREATE OR REPLACE FUNCTION get_drug_rule_with_fallback(
    p_rxnorm_code VARCHAR(20),
    p_jurisdiction VARCHAR(10)
)
RETURNS TABLE (
    id UUID,
    rxnorm_code VARCHAR(20),
    jurisdiction VARCHAR(10),
    drug_name VARCHAR(500),
    rule_data JSONB,
    authority VARCHAR(50),
    version VARCHAR(20)
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        dr.id,
        dr.rxnorm_code,
        dr.jurisdiction,
        dr.drug_name,
        dr.rule_data,
        dr.authority,
        dr.version
    FROM drug_rules dr
    WHERE dr.rxnorm_code = p_rxnorm_code
      AND dr.jurisdiction IN (p_jurisdiction, 'GLOBAL')
    ORDER BY CASE dr.jurisdiction
        WHEN p_jurisdiction THEN 0
        ELSE 1
    END
    LIMIT 1;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- FUNCTION: SEARCH DRUGS BY NAME
-- =============================================================================
CREATE OR REPLACE FUNCTION search_drugs(
    p_query VARCHAR(500),
    p_jurisdiction VARCHAR(10) DEFAULT 'US',
    p_limit INT DEFAULT 100
)
RETURNS TABLE (
    rxnorm_code VARCHAR(20),
    drug_name VARCHAR(500),
    generic_name VARCHAR(500),
    drug_class VARCHAR(200),
    jurisdiction VARCHAR(10),
    is_high_alert BOOLEAN,
    authority VARCHAR(50)
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        dr.rxnorm_code,
        dr.drug_name,
        dr.generic_name,
        dr.drug_class,
        dr.jurisdiction,
        dr.is_high_alert,
        dr.authority
    FROM drug_rules dr
    WHERE dr.jurisdiction IN (p_jurisdiction, 'GLOBAL')
      AND (
          dr.drug_name ILIKE '%' || p_query || '%'
          OR dr.generic_name ILIKE '%' || p_query || '%'
          OR dr.drug_class ILIKE '%' || p_query || '%'
      )
    ORDER BY
        -- Prioritize exact matches
        CASE WHEN dr.drug_name ILIKE p_query THEN 0
             WHEN dr.generic_name ILIKE p_query THEN 1
             WHEN dr.drug_name ILIKE p_query || '%' THEN 2
             ELSE 3
        END,
        dr.drug_name
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- COMMENTS FOR DOCUMENTATION
-- =============================================================================
COMMENT ON TABLE drug_rules IS 'Governed drug dosing rules from regulatory authorities (FDA, TGA, CDSCO)';
COMMENT ON TABLE ingestion_runs IS 'Audit log of drug rule ingestion runs from regulatory sources';
COMMENT ON TABLE ingestion_items IS 'Per-drug status tracking within an ingestion run';
COMMENT ON TABLE drug_rule_history IS 'Complete audit trail of all drug rule changes';

COMMENT ON COLUMN drug_rules.rxnorm_code IS 'RxNorm concept code from KB-7 terminology service';
COMMENT ON COLUMN drug_rules.jurisdiction IS 'Geographic jurisdiction: US, AU, IN, or GLOBAL';
COMMENT ON COLUMN drug_rules.rule_data IS 'Complete GovernedDrugRule as JSONB with dosing, safety, governance';
COMMENT ON COLUMN drug_rules.source_hash IS 'SHA-256 hash of source document for change detection';
COMMENT ON COLUMN drug_rules.is_high_alert IS 'ISMP High-Alert Medication flag';
COMMENT ON COLUMN drug_rules.is_narrow_ti IS 'Narrow Therapeutic Index drug flag';
COMMENT ON COLUMN drug_rules.has_black_box IS 'FDA Black Box Warning present';
COMMENT ON COLUMN drug_rules.is_beers_list IS 'AGS Beers Criteria for inappropriate use in elderly';
