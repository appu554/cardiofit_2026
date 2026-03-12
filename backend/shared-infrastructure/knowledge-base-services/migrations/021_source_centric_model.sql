-- ============================================================================
-- Phase 3a.2: Source-Centric Data Model for Clinical Truth Arbitration
-- ============================================================================
-- Core Insight: Parse each source ONCE, extract to MULTIPLE KBs
-- Full lineage from production fact back to regulatory label
--
-- INVARIANT: All facts have source_document_id + source_section_id
--            Complete audit trail from production fact to regulatory label
-- ============================================================================

-- ============================================================================
-- SOURCE DOCUMENTS
-- ============================================================================
-- Represents raw documents from authoritative sources (FDA SPL, CPIC, etc.)

CREATE TABLE IF NOT EXISTS source_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Source identification
    source_type VARCHAR(50) NOT NULL,      -- 'FDA_SPL', 'CPIC', 'CREDIBLEMEDS', 'LIVERTOX', 'LACTMED', 'DRUGBANK'
    document_id VARCHAR(255) NOT NULL,     -- SetID for SPL, PMID for literature, etc.
    version_number VARCHAR(50),            -- Document version (e.g., SPL version)

    -- Content tracking
    raw_content_hash VARCHAR(64) NOT NULL, -- SHA-256 for change detection
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    content_updated_at TIMESTAMPTZ,        -- When source content last changed

    -- Drug identification
    drug_name VARCHAR(255),
    generic_name VARCHAR(255),
    rxcui VARCHAR(50),                     -- Primary RxNorm concept
    ndc_codes TEXT[],                      -- Associated NDC codes
    atc_codes TEXT[],                      -- ATC classification codes

    -- Metadata
    effective_date DATE,                   -- When document became effective
    manufacturer VARCHAR(255),
    labeler_code VARCHAR(50),

    -- Processing status
    processing_status VARCHAR(20) DEFAULT 'PENDING',  -- PENDING, PROCESSING, COMPLETED, FAILED
    last_processed_at TIMESTAMPTZ,
    processing_error TEXT,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(source_type, document_id, version_number)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_source_docs_rxcui ON source_documents(rxcui);
CREATE INDEX IF NOT EXISTS idx_source_docs_type ON source_documents(source_type);
CREATE INDEX IF NOT EXISTS idx_source_docs_status ON source_documents(processing_status);
CREATE INDEX IF NOT EXISTS idx_source_docs_drug ON source_documents(drug_name);
CREATE INDEX IF NOT EXISTS idx_source_docs_hash ON source_documents(raw_content_hash);

-- ============================================================================
-- SOURCE SECTIONS
-- ============================================================================
-- Represents parsed sections from source documents (LOINC-coded for SPL)

CREATE TABLE IF NOT EXISTS source_sections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_document_id UUID NOT NULL REFERENCES source_documents(id) ON DELETE CASCADE,

    -- Section identification (LOINC-based for SPL)
    section_code VARCHAR(50) NOT NULL,        -- LOINC code: '34068-7' for Dosage & Administration
    section_name VARCHAR(255),                -- Human-readable: "Dosage and Administration"

    -- Routing configuration
    target_kbs TEXT[] NOT NULL,               -- ['KB-1', 'KB-4', 'KB-6'] - which KBs receive facts from this section

    -- Raw content
    raw_text TEXT,                            -- Original section text
    raw_html TEXT,                            -- Original HTML if available

    -- Parsed content
    parsed_tables JSONB,                      -- Tabular Harvester output (structured JSON)
    extraction_method VARCHAR(50),            -- 'TABLE_PARSE', 'REGEX_PARSE', 'LLM_GAP', 'AUTHORITY'
    extraction_confidence DECIMAL(5,4),       -- 0.0000 to 1.0000

    -- Processing metadata
    has_structured_tables BOOLEAN DEFAULT FALSE,
    table_count INTEGER DEFAULT 0,
    word_count INTEGER,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    UNIQUE(source_document_id, section_code)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_source_sections_doc ON source_sections(source_document_id);
CREATE INDEX IF NOT EXISTS idx_source_sections_code ON source_sections(section_code);
CREATE INDEX IF NOT EXISTS idx_source_sections_target ON source_sections USING GIN(target_kbs);
CREATE INDEX IF NOT EXISTS idx_source_sections_method ON source_sections(extraction_method);

-- ============================================================================
-- DERIVED FACTS
-- ============================================================================
-- Facts extracted from source sections, destined for specific KBs

CREATE TABLE IF NOT EXISTS derived_facts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Full Lineage (Source-Centric) - REQUIRED
    source_document_id UUID NOT NULL REFERENCES source_documents(id),
    source_section_id UUID REFERENCES source_sections(id),

    -- Fact identification
    target_kb VARCHAR(20) NOT NULL,            -- 'KB-1', 'KB-4', 'KB-5', etc.
    fact_type VARCHAR(100) NOT NULL,           -- 'RENAL_DOSE_ADJUST', 'HEPATIC_DOSE', 'QT_RISK', etc.
    fact_key VARCHAR(255),                     -- Unique identifier within KB (e.g., "metformin:gfr_band:30-60")

    -- Fact content
    fact_data JSONB NOT NULL,                  -- The actual extracted fact (structured JSON)

    -- Extraction metadata
    extraction_method VARCHAR(50) NOT NULL,    -- 'AUTHORITY', 'TABLE_PARSE', 'REGEX_PARSE', 'LLM_CONSENSUS'
    extraction_confidence DECIMAL(5,4),        -- 0.0000 to 1.0000
    evidence_spans JSONB,                      -- Quoted source text with offsets

    -- LLM extraction details (if applicable)
    llm_provider VARCHAR(50),                  -- 'claude', 'gpt4', 'gemini'
    llm_model VARCHAR(100),
    consensus_achieved BOOLEAN,
    consensus_providers TEXT[],                -- Which providers agreed

    -- Governance
    governance_status VARCHAR(20) DEFAULT 'DRAFT',  -- DRAFT, PENDING_REVIEW, APPROVED, REJECTED, SUPERSEDED
    reviewed_by VARCHAR(100),
    reviewed_at TIMESTAMPTZ,
    review_notes TEXT,

    -- Lifecycle
    is_active BOOLEAN DEFAULT TRUE,
    superseded_by UUID REFERENCES derived_facts(id),
    supersedes UUID REFERENCES derived_facts(id),

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Ensure facts always have source lineage
    CONSTRAINT fact_must_have_source CHECK (source_document_id IS NOT NULL)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_derived_facts_source ON derived_facts(source_document_id);
CREATE INDEX IF NOT EXISTS idx_derived_facts_section ON derived_facts(source_section_id);
CREATE INDEX IF NOT EXISTS idx_derived_facts_kb ON derived_facts(target_kb);
CREATE INDEX IF NOT EXISTS idx_derived_facts_type ON derived_facts(fact_type);
CREATE INDEX IF NOT EXISTS idx_derived_facts_key ON derived_facts(fact_key);
CREATE INDEX IF NOT EXISTS idx_derived_facts_status ON derived_facts(governance_status);
CREATE INDEX IF NOT EXISTS idx_derived_facts_active ON derived_facts(is_active) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_derived_facts_data ON derived_facts USING GIN(fact_data);

-- ============================================================================
-- LOINC SECTION ROUTING TABLE
-- ============================================================================
-- Maps LOINC section codes to target KBs and extraction methods

CREATE TABLE IF NOT EXISTS loinc_section_routing (
    section_code VARCHAR(50) PRIMARY KEY,     -- LOINC code
    section_name VARCHAR(255) NOT NULL,
    target_kbs TEXT[] NOT NULL,               -- Which KBs receive facts from this section
    extraction_method VARCHAR(50) NOT NULL,   -- Primary extraction method
    fallback_method VARCHAR(50),              -- Fallback if primary fails
    priority INTEGER DEFAULT 0,               -- Processing priority (higher = first)
    is_active BOOLEAN DEFAULT TRUE,
    notes TEXT
);

-- Seed with Phase 3 LOINC routing table
INSERT INTO loinc_section_routing (section_code, section_name, target_kbs, extraction_method, fallback_method, priority) VALUES
    ('34066-1', 'Boxed Warning', ARRAY['KB-4'], 'TABLE_PARSE', 'LLM_GAP', 100),
    ('34068-7', 'Dosage and Administration', ARRAY['KB-1', 'KB-6', 'KB-16'], 'TABLE_PARSE', 'LLM_GAP', 90),
    ('34070-3', 'Contraindications', ARRAY['KB-4', 'KB-5'], 'TABLE_PARSE', 'LLM_GAP', 85),
    ('34073-7', 'Drug Interactions', ARRAY['KB-5'], 'TABLE_PARSE', 'LLM_GAP', 80),
    ('34077-8', 'Pregnancy', ARRAY['KB-4'], 'AUTHORITY', 'LLM_GAP', 70),
    ('34080-2', 'Nursing Mothers', ARRAY['KB-4'], 'AUTHORITY', 'LLM_GAP', 70),
    ('34081-0', 'Pediatric Use', ARRAY['KB-4', 'KB-1'], 'TABLE_PARSE', 'LLM_GAP', 65),
    ('34082-8', 'Geriatric Use', ARRAY['KB-4', 'KB-1'], 'TABLE_PARSE', 'LLM_GAP', 65),
    ('34090-1', 'Clinical Pharmacology', ARRAY['KB-1'], 'TABLE_PARSE', 'LLM_GAP', 60),
    ('43685-7', 'Warnings and Precautions', ARRAY['KB-4'], 'TABLE_PARSE', 'LLM_GAP', 75)
ON CONFLICT (section_code) DO NOTHING;

-- ============================================================================
-- AUTHORITY SOURCES REGISTRY
-- ============================================================================
-- Tracks authoritative sources and their fact type coverage

CREATE TABLE IF NOT EXISTS authority_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_name VARCHAR(100) NOT NULL UNIQUE,   -- 'CPIC', 'CREDIBLEMEDS', 'LIVERTOX', 'LACTMED'
    source_type VARCHAR(50) NOT NULL,           -- 'API', 'XML_DUMP', 'CSV', 'DATABASE'
    base_url VARCHAR(500),
    api_version VARCHAR(50),

    -- Coverage
    fact_types_covered TEXT[] NOT NULL,         -- ['PHARMACOGENOMICS', 'QT_RISK', 'HEPATOTOXICITY']
    authority_level VARCHAR(20) NOT NULL,       -- 'DEFINITIVE', 'PRIMARY', 'SECONDARY'

    -- LLM policy
    llm_allowed BOOLEAN DEFAULT FALSE,          -- Can LLM be used for this source's facts?
    llm_policy_notes TEXT,

    -- Status
    is_active BOOLEAN DEFAULT TRUE,
    last_sync_at TIMESTAMPTZ,
    next_sync_at TIMESTAMPTZ,
    sync_frequency_hours INTEGER DEFAULT 24,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Seed authority sources from Phase 3 plan
INSERT INTO authority_sources (source_name, source_type, fact_types_covered, authority_level, llm_allowed, llm_policy_notes) VALUES
    ('CPIC', 'API', ARRAY['PHARMACOGENOMICS', 'GENE_DRUG_INTERACTION'], 'DEFINITIVE', FALSE, 'LLM NEVER allowed - use CPIC API directly'),
    ('CredibleMeds', 'API', ARRAY['QT_RISK', 'QT_PROLONGATION', 'TORSADES_RISK'], 'DEFINITIVE', FALSE, 'LLM NEVER allowed - use CredibleMeds data directly'),
    ('LiverTox', 'XML_DUMP', ARRAY['HEPATOTOXICITY', 'LIVER_INJURY_RISK'], 'DEFINITIVE', FALSE, 'LLM NEVER allowed - use LiverTox database'),
    ('LactMed', 'XML_DUMP', ARRAY['LACTATION_SAFETY', 'BREASTFEEDING_RISK'], 'DEFINITIVE', FALSE, 'LLM NEVER allowed - use LactMed database'),
    ('FDA_DailyMed', 'API', ARRAY['RENAL_DOSE', 'HEPATIC_DOSE', 'CONTRAINDICATION', 'BLACK_BOX_WARNING'], 'PRIMARY', TRUE, 'LLM allowed for gap-filling with 2-of-3 consensus'),
    ('DrugBank', 'DATABASE', ARRAY['PK_PARAMETERS', 'DRUG_METABOLISM', 'HALF_LIFE'], 'PRIMARY', FALSE, 'Use structured DrugBank data'),
    ('OHDSI_Athena', 'DATABASE', ARRAY['TERMINOLOGY', 'DRUG_CLASS', 'CONCEPT_MAPPING'], 'PRIMARY', FALSE, 'Use OHDSI vocabulary tables')
ON CONFLICT (source_name) DO NOTHING;

-- ============================================================================
-- EXTRACTION AUDIT LOG
-- ============================================================================
-- Tracks all extraction attempts for debugging and governance

CREATE TABLE IF NOT EXISTS extraction_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_document_id UUID REFERENCES source_documents(id),
    source_section_id UUID REFERENCES source_sections(id),
    derived_fact_id UUID REFERENCES derived_facts(id),

    -- Extraction details
    extraction_method VARCHAR(50) NOT NULL,
    extraction_started_at TIMESTAMPTZ NOT NULL,
    extraction_completed_at TIMESTAMPTZ,
    extraction_duration_ms INTEGER,

    -- LLM details (if applicable)
    llm_provider VARCHAR(50),
    llm_model VARCHAR(100),
    llm_prompt_tokens INTEGER,
    llm_completion_tokens INTEGER,
    llm_raw_response TEXT,

    -- Consensus details (if applicable)
    consensus_required BOOLEAN,
    consensus_achieved BOOLEAN,
    providers_agreed TEXT[],
    providers_disagreed TEXT[],
    disagreement_details JSONB,

    -- Outcome
    success BOOLEAN NOT NULL,
    error_message TEXT,
    confidence_score DECIMAL(5,4),

    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_audit_log_doc ON extraction_audit_log(source_document_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_fact ON extraction_audit_log(derived_fact_id);
CREATE INDEX IF NOT EXISTS idx_audit_log_time ON extraction_audit_log(extraction_started_at);

-- ============================================================================
-- HUMAN ESCALATION QUEUE
-- ============================================================================
-- Queue for facts requiring human review (consensus not achieved, low confidence)

CREATE TABLE IF NOT EXISTS human_escalation_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    derived_fact_id UUID REFERENCES derived_facts(id),
    source_document_id UUID REFERENCES source_documents(id),

    -- Escalation reason
    escalation_reason VARCHAR(100) NOT NULL,   -- 'CONSENSUS_NOT_ACHIEVED', 'LOW_CONFIDENCE', 'CRITICAL_SAFETY', 'MANUAL_REQUEST'
    escalation_details JSONB,

    -- Priority and assignment
    priority VARCHAR(20) DEFAULT 'NORMAL',     -- LOW, NORMAL, HIGH, CRITICAL
    assigned_to VARCHAR(100),
    assigned_at TIMESTAMPTZ,

    -- Resolution
    status VARCHAR(20) DEFAULT 'PENDING',      -- PENDING, IN_REVIEW, RESOLVED, DEFERRED
    resolution VARCHAR(100),                   -- 'APPROVED', 'REJECTED', 'MODIFIED', 'DEFERRED'
    resolution_notes TEXT,
    resolved_by VARCHAR(100),
    resolved_at TIMESTAMPTZ,

    -- SLA tracking
    sla_deadline TIMESTAMPTZ,
    sla_breached BOOLEAN DEFAULT FALSE,

    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_escalation_status ON human_escalation_queue(status);
CREATE INDEX IF NOT EXISTS idx_escalation_priority ON human_escalation_queue(priority);
CREATE INDEX IF NOT EXISTS idx_escalation_assigned ON human_escalation_queue(assigned_to);

-- ============================================================================
-- HELPER VIEWS
-- ============================================================================

-- View: Facts with full lineage
CREATE OR REPLACE VIEW v_facts_with_lineage AS
SELECT
    df.id AS fact_id,
    df.target_kb,
    df.fact_type,
    df.fact_key,
    df.fact_data,
    df.extraction_method,
    df.extraction_confidence,
    df.governance_status,
    sd.source_type,
    sd.document_id,
    sd.drug_name,
    sd.rxcui,
    ss.section_code,
    ss.section_name,
    df.created_at AS extracted_at
FROM derived_facts df
JOIN source_documents sd ON df.source_document_id = sd.id
LEFT JOIN source_sections ss ON df.source_section_id = ss.id
WHERE df.is_active = TRUE;

-- View: Pending escalations by priority
CREATE OR REPLACE VIEW v_pending_escalations AS
SELECT
    heq.*,
    sd.drug_name,
    sd.source_type,
    df.fact_type,
    df.extraction_confidence
FROM human_escalation_queue heq
LEFT JOIN derived_facts df ON heq.derived_fact_id = df.id
LEFT JOIN source_documents sd ON heq.source_document_id = sd.id
WHERE heq.status = 'PENDING'
ORDER BY
    CASE heq.priority
        WHEN 'CRITICAL' THEN 1
        WHEN 'HIGH' THEN 2
        WHEN 'NORMAL' THEN 3
        WHEN 'LOW' THEN 4
    END,
    heq.created_at;

-- ============================================================================
-- COMMENTS
-- ============================================================================

COMMENT ON TABLE source_documents IS 'Raw documents from authoritative sources with change tracking';
COMMENT ON TABLE source_sections IS 'Parsed sections from source documents with LOINC coding';
COMMENT ON TABLE derived_facts IS 'Facts extracted from sources with full lineage to regulatory labels';
COMMENT ON TABLE loinc_section_routing IS 'Maps LOINC sections to target KBs and extraction methods';
COMMENT ON TABLE authority_sources IS 'Registry of authoritative sources and their LLM policies';
COMMENT ON TABLE extraction_audit_log IS 'Complete audit trail of all extraction attempts';
COMMENT ON TABLE human_escalation_queue IS 'Queue for facts requiring human review';
