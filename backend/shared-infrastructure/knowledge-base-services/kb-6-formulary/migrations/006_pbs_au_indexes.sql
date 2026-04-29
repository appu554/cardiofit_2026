-- 006_pbs_au_indexes.sql
--
-- Indexes for kb6_pbs_* tables. Apply AFTER bulk load.

-- Items: text search by drug name (case-insensitive)
CREATE INDEX IF NOT EXISTS idx_kb6_pbs_items_drug_name_lower
    ON kb6_pbs_items (lower(drug_name));

-- Items: trigram for substring search on drug name (if pg_trgm available)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_available_extensions WHERE name = 'pg_trgm') THEN
        CREATE EXTENSION IF NOT EXISTS pg_trgm;
        CREATE INDEX IF NOT EXISTS idx_kb6_pbs_items_drug_name_trgm
            ON kb6_pbs_items USING gin (drug_name gin_trgm_ops);
    END IF;
END $$;

-- Items: schedule classification (most common filter for ACOP queries)
CREATE INDEX IF NOT EXISTS idx_kb6_pbs_items_schedule
    ON kb6_pbs_items (schedule_section, is_active) WHERE is_active = TRUE;

CREATE INDEX IF NOT EXISTS idx_kb6_pbs_items_authority
    ON kb6_pbs_items (is_authority_required, is_streamlined, is_active) WHERE is_active = TRUE;

CREATE INDEX IF NOT EXISTS idx_kb6_pbs_items_section_100
    ON kb6_pbs_items (is_section_100) WHERE is_section_100 = TRUE;

-- Items: AMT cross-reference (look up PBS items for a given AMT MP / MPUU)
CREATE INDEX IF NOT EXISTS idx_kb6_pbs_items_amt_mp
    ON kb6_pbs_items (amt_mp_sctid) WHERE amt_mp_sctid IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_kb6_pbs_items_amt_mpuu
    ON kb6_pbs_items (amt_mpuu_sctid) WHERE amt_mpuu_sctid IS NOT NULL;

-- Items: RxNorm cross-reference
CREATE INDEX IF NOT EXISTS idx_kb6_pbs_items_rxnorm
    ON kb6_pbs_items (rxnorm_code) WHERE rxnorm_code IS NOT NULL;

-- Authorities: lookup by item
CREATE INDEX IF NOT EXISTS idx_kb6_pbs_authorities_pbs
    ON kb6_pbs_authorities (pbs_code, authority_type);

-- Restrictions: lookup by item
CREATE INDEX IF NOT EXISTS idx_kb6_pbs_restrictions_pbs
    ON kb6_pbs_restrictions (pbs_code);

-- Prescriber types: lookup by item
CREATE INDEX IF NOT EXISTS idx_kb6_pbs_prescriber_types_pbs
    ON kb6_pbs_prescriber_types (pbs_code);

-- Section 100: type lookup (find all HSD / RAAHS items)
CREATE INDEX IF NOT EXISTS idx_kb6_pbs_section_100_type
    ON kb6_pbs_section_100 (section_100_type);

-- Indications: lookup by item
CREATE INDEX IF NOT EXISTS idx_kb6_pbs_indications_pbs
    ON kb6_pbs_indications (pbs_code);
