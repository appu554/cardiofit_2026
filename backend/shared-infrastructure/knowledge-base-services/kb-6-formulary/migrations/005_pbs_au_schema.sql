-- 005_pbs_au_schema.sql
--
-- Australian PBS (Pharmaceutical Benefits Scheme) tables for KB-6.
--
-- Coexists with the existing US-insurance schema (formulary_entries,
-- pa_submissions, etc.). All AU tables are namespaced kb6_pbs_*.
--
-- Source: PBS Schedule monthly XML extract from https://www.pbs.gov.au
-- License: Public domain (Australian Government)
--
-- Cross-references
-- - amt_mp_sctid / amt_mpuu_sctid bridge to kb7_amt_pack in KB-7
--   (loaded by SNOMED CT-AU + AMT pipeline). Soft FKs since KB-7 lives
--   in a different DB and we don't enforce cross-DB FKs.
-- - rxnorm_code bridge to kb1_drug_rules.rxnorm_code (resolved via RxNav).
--
-- Indexes are in 006_pbs_au_indexes.sql, applied AFTER bulk load.

-- ---------- Core item table ----------

-- One row per PBS item code. PBS item codes are 4-5 characters
-- (e.g., "1574H" = metformin 500mg tablet, "8362K" = empagliflozin
-- 10mg tablet General Schedule). Item code is the primary key.

CREATE TABLE IF NOT EXISTS kb6_pbs_items (
    pbs_code               VARCHAR(10)  PRIMARY KEY,
    drug_name              VARCHAR(500) NOT NULL,
    drug_class             TEXT,            -- some PBS combination products have very long class descriptors
    form                   TEXT,            -- e.g., "Tablet containing 12.5 mg X with 850 mg Y"
    strength               TEXT,
    manner_of_administration VARCHAR(100),
    max_quantity           INTEGER,
    max_repeats            INTEGER,
    pack_size              INTEGER,
    pack_quantity          INTEGER,
    -- Schedule classification
    schedule_section       VARCHAR(50),     -- 'GENERAL', 'AUTHORITY', 'STREAMLINED',
                                            -- 'RESTRICTED', 'S100_HSD', 'S100_RAAHS',
                                            -- 'CHEMO', 'PALLIATIVE'
    is_authority_required  BOOLEAN DEFAULT FALSE,
    is_streamlined         BOOLEAN DEFAULT FALSE,
    is_restricted          BOOLEAN DEFAULT FALSE,
    is_section_100         BOOLEAN DEFAULT FALSE,
    is_palliative_care     BOOLEAN DEFAULT FALSE,
    is_chemotherapy        BOOLEAN DEFAULT FALSE,
    -- AMT / KB-7 cross-reference (soft FK; kb7_amt_pack is in KB-7 DB)
    amt_mp_sctid           BIGINT,          -- Medicinal Product
    amt_mpuu_sctid         BIGINT,          -- Medicinal Product Unit of Use
    amt_tpp_sctid          BIGINT,          -- Trade Product Pack
    amt_ctpp_sctid         BIGINT,          -- Containered Trade Product Pack
    -- RxNorm cross-reference (soft FK; resolved via RxNav-in-a-Box)
    rxnorm_code            VARCHAR(50),
    -- Lifecycle
    effective_date         DATE,
    end_date               DATE,
    is_active              BOOLEAN DEFAULT TRUE,
    -- Provenance
    schedule_publish_date  DATE,
    raw_xml                JSONB,           -- full source row for audit
    loaded_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------- Authority requirements (1:N per item) ----------

-- An item can have multiple authority requirements (one per indication).

CREATE TABLE IF NOT EXISTS kb6_pbs_authorities (
    id                     BIGSERIAL    PRIMARY KEY,
    pbs_code               VARCHAR(10)  NOT NULL REFERENCES kb6_pbs_items(pbs_code) ON DELETE CASCADE,
    authority_type         VARCHAR(50)  NOT NULL,   -- 'AUTHORITY_REQUIRED', 'STREAMLINED',
                                                    -- 'TELEPHONE', 'WRITTEN', 'NONE'
    authority_code         VARCHAR(50),             -- e.g., "1234" (PBS streamlined code)
    description            TEXT,
    requires_specialist    BOOLEAN DEFAULT FALSE,
    requires_consultant    BOOLEAN DEFAULT FALSE
);

-- ---------- Clinical restrictions (1:N per item) ----------

-- Free-text restriction criteria (e.g., "Must demonstrate inadequate
-- glycaemic control on dual oral therapy"). PBS doesn't structure
-- these formally — extracted as free text plus optional indication code.

CREATE TABLE IF NOT EXISTS kb6_pbs_restrictions (
    id                     BIGSERIAL    PRIMARY KEY,
    pbs_code               VARCHAR(10)  NOT NULL REFERENCES kb6_pbs_items(pbs_code) ON DELETE CASCADE,
    restriction_text       TEXT         NOT NULL,
    indication_code        VARCHAR(50),             -- PBS indication code if present
    is_initial             BOOLEAN DEFAULT FALSE,   -- initial-treatment criterion
    is_continuing          BOOLEAN DEFAULT FALSE    -- continuing-treatment criterion
);

-- ---------- Prescriber types (1:N per item) ----------

-- Who is permitted to prescribe each item under PBS rules.

CREATE TABLE IF NOT EXISTS kb6_pbs_prescriber_types (
    id                     BIGSERIAL    PRIMARY KEY,
    pbs_code               VARCHAR(10)  NOT NULL REFERENCES kb6_pbs_items(pbs_code) ON DELETE CASCADE,
    prescriber_type        VARCHAR(50)  NOT NULL    -- 'GP', 'SPECIALIST', 'NURSE_PRACTITIONER',
                                                    -- 'MIDWIFE', 'OPTOMETRIST', 'DENTIST'
);

-- ---------- Section 100 (HSD / RAAHS) details ----------

-- Section 100 covers Highly Specialised Drugs (HSD), Remote Area
-- Aboriginal Health Services (RAAHS), and a few other special programs.
-- Critical for aged care because some HSD drugs (e.g., subcutaneous
-- biologics, IV infusion drugs) require special supply pathways.

CREATE TABLE IF NOT EXISTS kb6_pbs_section_100 (
    pbs_code               VARCHAR(10)  PRIMARY KEY REFERENCES kb6_pbs_items(pbs_code) ON DELETE CASCADE,
    section_100_type       VARCHAR(30)  NOT NULL,   -- 'HSD', 'RAAHS', 'METHADONE', 'GROWTH_HORMONE',
                                                    -- 'IVF', 'BOTULINUM_TOXIN', 'CHEMOTHERAPY'
    supply_pathway         VARCHAR(100),            -- e.g., 'public_hospital', 'private_hospital'
    notes                  TEXT
);

-- ---------- PBS-approved indications (1:N per item) ----------

-- Many PBS items list specific approved indications, sometimes with
-- ICD-10/SNOMED codes attached.

CREATE TABLE IF NOT EXISTS kb6_pbs_indications (
    id                     BIGSERIAL    PRIMARY KEY,
    pbs_code               VARCHAR(10)  NOT NULL REFERENCES kb6_pbs_items(pbs_code) ON DELETE CASCADE,
    indication_text        TEXT         NOT NULL,
    icd10am_codes          TEXT[],
    snomed_codes           TEXT[]
);

-- ---------- Audit log ----------

CREATE TABLE IF NOT EXISTS kb6_pbs_load_log (
    load_id                SERIAL      PRIMARY KEY,
    schedule_date          DATE        NOT NULL,    -- e.g., '2026-04-01' for April 2026 schedule
    source_file            TEXT        NOT NULL,
    table_name             TEXT        NOT NULL,
    rows_loaded            BIGINT      NOT NULL,
    sha256                 TEXT,
    loaded_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);
