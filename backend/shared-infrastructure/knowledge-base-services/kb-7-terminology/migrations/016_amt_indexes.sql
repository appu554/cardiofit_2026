-- 016_amt_indexes.sql
--
-- Indexes for kb7_amt_pack. Apply AFTER bulk COPY load.

-- "Find all packs of substance X" — the ACOP MP-level rule pattern
CREATE INDEX IF NOT EXISTS idx_kb7_amt_pack_mp
    ON kb7_amt_pack (mp_sctid);

-- "Find all packs of generic UoU X"
CREATE INDEX IF NOT EXISTS idx_kb7_amt_pack_mpuu
    ON kb7_amt_pack (mpuu_sctid);

-- "Find all packs of brand X"
CREATE INDEX IF NOT EXISTS idx_kb7_amt_pack_tp
    ON kb7_amt_pack (tpp_tp_sctid);

-- "Find packs at TPP level"
CREATE INDEX IF NOT EXISTS idx_kb7_amt_pack_tpp
    ON kb7_amt_pack (tpp_sctid);

-- "Find by ARTG_ID" — for cross-reference to TGA registration data
CREATE INDEX IF NOT EXISTS idx_kb7_amt_pack_artg
    ON kb7_amt_pack (artg_id)
    WHERE artg_id IS NOT NULL;

-- Substance-name lookups (case-insensitive); useful for "find all amlodipine packs"
CREATE INDEX IF NOT EXISTS idx_kb7_amt_pack_mp_pt_lower
    ON kb7_amt_pack (lower(mp_pt));
