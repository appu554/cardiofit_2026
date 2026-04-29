-- 017_icd10am_schema.sql
--
-- ICD-10-AM (Australian Modification of ICD-10) + ACHI (Australian
-- Classification of Health Interventions) schema.
--
-- Source: IHACPA / ACCD distribution (commercially licensed, not free).
-- Distribution typically includes:
--   - ICD-10-AM Tabular List (XML, ~14 chapters)
--   - ICD-10-AM Alphabetic Index
--   - ACHI Tabular List (XML, ~20 chapters of procedures)
--   - ACHI Alphabetic Index
--   - Australian Coding Standards (PDF — not loaded)
--
-- Coexists with kb7_snomed_* (concepts) and kb7_amt_pack (medicines).
-- Cross-references between SNOMED CT-AU and ICD-10-AM live in a separate
-- mapping table (kb7_snomed_to_icd10am_map) — added when IHACPA mapping
-- distributions are loaded.
--
-- Indexes are in 018_icd10am_indexes.sql, applied AFTER bulk load.

-- ---------- ICD-10-AM (diseases) ----------

CREATE TABLE IF NOT EXISTS kb7_icd10am_chapter (
    chapter_number  INTEGER     PRIMARY KEY,
    title           TEXT        NOT NULL,
    code_range      TEXT        NOT NULL,        -- e.g., "A00-B99"
    description     TEXT
);

CREATE TABLE IF NOT EXISTS kb7_icd10am_block (
    block_id        TEXT        PRIMARY KEY,     -- e.g., "A00-A09"
    chapter_number  INTEGER     NOT NULL REFERENCES kb7_icd10am_chapter(chapter_number),
    title           TEXT        NOT NULL,
    code_range      TEXT        NOT NULL,
    description     TEXT
);

CREATE TABLE IF NOT EXISTS kb7_icd10am_code (
    code            TEXT        PRIMARY KEY,     -- e.g., "E11.65" or "Z51.5"
    parent_code     TEXT,                        -- e.g., "E11" for "E11.65"
    chapter_number  INTEGER,
    block_id        TEXT        REFERENCES kb7_icd10am_block(block_id),
    description     TEXT        NOT NULL,
    is_billable     BOOLEAN,                     -- terminal codes (4-/5-char) are billable for hospital coding
    asterisk_dagger CHAR(1),                     -- AM dual-classification marker ('*' or '†' or NULL)
    inclusions      TEXT[],                      -- inclusion notes from the tabular list
    exclusions      TEXT[],                      -- exclusion notes
    notes           TEXT,                        -- general notes
    edition         TEXT                         -- e.g., "12th edition" — supports cross-edition coexistence
);

CREATE TABLE IF NOT EXISTS kb7_icd10am_index (
    -- Alphabetic Index — lookup terms that map to ICD-10-AM codes.
    -- Multi-row per term (different lead terms map to same code, vice versa).
    id              BIGSERIAL   PRIMARY KEY,
    lead_term       TEXT        NOT NULL,        -- the index entry term
    modifiers       TEXT,                        -- modifier qualifiers (e.g., "with complications")
    code            TEXT        NOT NULL,        -- target ICD-10-AM code (FK soft, may not always resolve)
    edition         TEXT
);

-- ---------- ACHI (procedures) ----------

CREATE TABLE IF NOT EXISTS kb7_achi_block (
    block_id        TEXT        PRIMARY KEY,     -- e.g., "1820" or "1821-1834"
    chapter_number  INTEGER,
    title           TEXT        NOT NULL,
    description     TEXT
);

CREATE TABLE IF NOT EXISTS kb7_achi_code (
    code            TEXT        PRIMARY KEY,     -- e.g., "30445-00"
    block_id        TEXT        REFERENCES kb7_achi_block(block_id),
    description     TEXT        NOT NULL,
    procedure_type  TEXT,                        -- if categorised in distribution
    notes           TEXT,
    edition         TEXT
);

CREATE TABLE IF NOT EXISTS kb7_achi_index (
    id              BIGSERIAL   PRIMARY KEY,
    lead_term       TEXT        NOT NULL,
    modifiers       TEXT,
    code            TEXT        NOT NULL,
    edition         TEXT
);

-- ---------- Audit log ----------

CREATE TABLE IF NOT EXISTS kb7_icd10am_load_log (
    load_id         SERIAL      PRIMARY KEY,
    edition         TEXT        NOT NULL,        -- e.g., "12th edition"
    release_date    DATE,
    source_file     TEXT        NOT NULL,
    table_name      TEXT        NOT NULL,
    rows_loaded     BIGINT      NOT NULL,
    sha256          TEXT,
    loaded_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
