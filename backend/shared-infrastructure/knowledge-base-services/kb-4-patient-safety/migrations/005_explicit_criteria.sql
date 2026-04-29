-- 005_explicit_criteria.sql
--
-- Explicit-criteria geriatric prescribing rules for KB-4 (Wave 3 of the
-- AU Aged Care Layer 1 plan). Single unified table covering:
--   - STOPP v3 (80 entries — drugs to STOP in older adults)
--   - START v3 (40 entries — drugs to START / prescribing omissions)
--   - AGS Beers 2023 (57 entries — US PIM list, internationally referenced)
--
-- Source YAMLs already structured at:
--   knowledge/global/stopp_start/stopp_v3.yaml
--   knowledge/global/stopp_start/start_v3.yaml
--   knowledge/beers/beers_criteria_2023.yaml
--
-- Loaded by scripts/load_explicit_criteria.py.
--
-- Coexists with kb_l3_staging (KDIGO L3 facts) and safety_alerts (runtime
-- US-PA alerts). All Wave 3 columns namespaced kb4_explicit_*.

CREATE TABLE IF NOT EXISTS kb4_explicit_criteria (
    id                          BIGSERIAL    PRIMARY KEY,
    -- Discriminator
    criterion_set               VARCHAR(20)  NOT NULL,    -- STOPP_V3 | START_V3 | BEERS_2023
    criterion_id                VARCHAR(50)  NOT NULL,    -- e.g., "A1" (STOPP/START) or RxCUI (Beers)
    -- Section / classification
    section                     VARCHAR(200),
    section_name                VARCHAR(500),
    drug_class                  TEXT,
    drug_name                   VARCHAR(500),             -- single drug (Beers); plural goes via rxnorm_codes
    -- Drug codes
    rxnorm_codes                TEXT[],                   -- STOPP/START — array
    rxnorm_code_primary         VARCHAR(50),              -- Beers — single
    atc_code                    VARCHAR(20),              -- Beers
    -- Condition / context
    condition_text              TEXT,                     -- START + Beers
    condition_icd10             TEXT[],                   -- START
    conditions_to_avoid         JSONB,                    -- Beers structured list
    -- Recommendation
    recommended_drugs           TEXT[],                   -- START — what to start
    recommendation              VARCHAR(50),              -- Beers (AVOID, USE_WITH_CAUTION, AVOID_IN_PRESENCE_OF, etc.)
    criteria_text               TEXT NOT NULL,            -- the main rule sentence
    rationale                   TEXT,
    exceptions                  TEXT,                     -- STOPP/START
    -- Evidence quality
    evidence_level              VARCHAR(20),
    quality_of_evidence         VARCHAR(20),              -- Beers
    strength_of_recommendation  VARCHAR(20),              -- Beers
    -- Beers-specific
    acb_score                   INTEGER,                  -- anticholinergic burden score 0-3
    alternatives                TEXT[],                   -- safer alternative drugs
    -- Provenance
    source_authority            VARCHAR(50),
    source_document             VARCHAR(200),
    source_url                  TEXT,
    source_section              VARCHAR(200),
    jurisdiction                VARCHAR(20),              -- 'global', 'US', 'AU', etc.
    knowledge_version           VARCHAR(30),
    effective_date              DATE,
    review_date                 DATE,
    approval_status             VARCHAR(20),
    governance                  JSONB,                    -- full governance block
    -- Audit
    raw_yaml                    JSONB,                    -- entire source row
    loaded_at                   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (criterion_set, criterion_id)
);

CREATE TABLE IF NOT EXISTS kb4_explicit_criteria_load_log (
    load_id        SERIAL      PRIMARY KEY,
    criterion_set  VARCHAR(20) NOT NULL,
    source_file    TEXT        NOT NULL,
    rows_loaded    BIGINT      NOT NULL,
    sha256         TEXT,
    loaded_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
