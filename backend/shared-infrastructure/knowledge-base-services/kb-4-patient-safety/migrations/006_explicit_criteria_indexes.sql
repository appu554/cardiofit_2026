-- 006_explicit_criteria_indexes.sql
--
-- Indexes for kb4_explicit_criteria. Apply AFTER bulk load.

-- Filter by criterion set
CREATE INDEX IF NOT EXISTS idx_kb4_explicit_set
    ON kb4_explicit_criteria (criterion_set);

-- Drug class lookup
CREATE INDEX IF NOT EXISTS idx_kb4_explicit_drug_class
    ON kb4_explicit_criteria (drug_class)
    WHERE drug_class IS NOT NULL;

-- Drug name (Beers — case-insensitive)
CREATE INDEX IF NOT EXISTS idx_kb4_explicit_drug_name_lower
    ON kb4_explicit_criteria (lower(drug_name))
    WHERE drug_name IS NOT NULL;

-- Beers single rxnorm
CREATE INDEX IF NOT EXISTS idx_kb4_explicit_rxnorm_primary
    ON kb4_explicit_criteria (rxnorm_code_primary)
    WHERE rxnorm_code_primary IS NOT NULL;

-- STOPP/START rxnorm array — GIN for "any criterion that mentions this rxcui"
CREATE INDEX IF NOT EXISTS idx_kb4_explicit_rxnorm_codes_gin
    ON kb4_explicit_criteria USING gin (rxnorm_codes);

-- ATC code (Beers)
CREATE INDEX IF NOT EXISTS idx_kb4_explicit_atc_code
    ON kb4_explicit_criteria (atc_code)
    WHERE atc_code IS NOT NULL;

-- ICD-10 condition codes (START) — GIN for "any criterion targeting this condition"
CREATE INDEX IF NOT EXISTS idx_kb4_explicit_icd10_gin
    ON kb4_explicit_criteria USING gin (condition_icd10);

-- Trigram for substring search across criteria text (if pg_trgm available)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_available_extensions WHERE name = 'pg_trgm') THEN
        CREATE EXTENSION IF NOT EXISTS pg_trgm;
        CREATE INDEX IF NOT EXISTS idx_kb4_explicit_criteria_text_trgm
            ON kb4_explicit_criteria USING gin (criteria_text gin_trgm_ops);
    END IF;
END $$;

-- ACB score lookup (Beers — drugs with high anticholinergic burden)
CREATE INDEX IF NOT EXISTS idx_kb4_explicit_acb
    ON kb4_explicit_criteria (acb_score)
    WHERE acb_score IS NOT NULL;
