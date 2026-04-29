-- 018_icd10am_indexes.sql
--
-- Indexes for ICD-10-AM and ACHI tables. Apply AFTER bulk load.

-- ICD-10-AM: code prefix lookups (e.g., find all E11.* diabetes-related codes)
CREATE INDEX IF NOT EXISTS idx_kb7_icd10am_code_parent
    ON kb7_icd10am_code (parent_code);

CREATE INDEX IF NOT EXISTS idx_kb7_icd10am_code_chapter
    ON kb7_icd10am_code (chapter_number, block_id);

-- ICD-10-AM: case-insensitive description search
CREATE INDEX IF NOT EXISTS idx_kb7_icd10am_code_desc_lower
    ON kb7_icd10am_code (lower(description));

-- ICD-10-AM: trigram index for substring search ("contains 'diabetes'")
-- Requires pg_trgm extension; create CONDITIONALLY so migration succeeds
-- on databases that don't have it (will fall back to LIKE without index).
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_available_extensions WHERE name = 'pg_trgm') THEN
        CREATE EXTENSION IF NOT EXISTS pg_trgm;
        CREATE INDEX IF NOT EXISTS idx_kb7_icd10am_code_desc_trgm
            ON kb7_icd10am_code USING gin (description gin_trgm_ops);
    END IF;
END $$;

-- ICD-10-AM index: lookup by lead term (case-insensitive)
CREATE INDEX IF NOT EXISTS idx_kb7_icd10am_index_term_lower
    ON kb7_icd10am_index (lower(lead_term));

CREATE INDEX IF NOT EXISTS idx_kb7_icd10am_index_code
    ON kb7_icd10am_index (code);

-- ACHI: same patterns
CREATE INDEX IF NOT EXISTS idx_kb7_achi_code_block
    ON kb7_achi_code (block_id);

CREATE INDEX IF NOT EXISTS idx_kb7_achi_code_desc_lower
    ON kb7_achi_code (lower(description));

CREATE INDEX IF NOT EXISTS idx_kb7_achi_index_term_lower
    ON kb7_achi_index (lower(lead_term));

CREATE INDEX IF NOT EXISTS idx_kb7_achi_index_code
    ON kb7_achi_index (code);
