-- 015_amt_schema.sql
--
-- Australian Medicines Terminology (AMT) flat-pack table.
--
-- AMT distributes as a denormalized TSV with one row per CTPP
-- (Containered Trade Product Pack — the most specific level).
-- Each row carries all 6 hierarchy levels inline:
--
--   CTPP -> TPP -> TPUU
--                    \--> TP (brand)
--           TPP -> MPP -> MPUU -> MP (substance)
--
-- ACOP rule queries typically operate at MP (substance-level checks
-- like "is amlodipine prescribed?") and CTPP (claim-level "what was
-- actually dispensed?"). The flat shape lets these be simple WHERE
-- clauses without joins.
--
-- Module ID for AMT in SNOMED namespace: 900062011000036103
--
-- Indexes are in 016_amt_indexes.sql, applied AFTER bulk load.

-- Note on primary key: the AMT TSV is at row-granularity (pack, mpuu, mp,
-- tpuu, tp_for_tpp, tp_for_tpuu) — multi-chamber/multi-substance packs
-- emit multiple rows per CTPP, and even (ctpp, mp) and (ctpp, mpuu) are
-- not unique in the source data. A surrogate BIGSERIAL PK is the clean
-- choice; uniqueness of the (ctpp, mpuu, mp) triple is enforced as a
-- separate UNIQUE constraint instead.
CREATE TABLE IF NOT EXISTS kb7_amt_pack (
    id             BIGSERIAL PRIMARY KEY,
    ctpp_sctid     BIGINT  NOT NULL,
    ctpp_pt        TEXT    NOT NULL,
    artg_id        BIGINT,                  -- AU Register of Therapeutic Goods ID; nullable for non-marketed packs
    tpp_sctid      BIGINT  NOT NULL,
    tpp_pt         TEXT    NOT NULL,
    tpuu_sctid     BIGINT  NOT NULL,
    tpuu_pt        TEXT    NOT NULL,
    tpp_tp_sctid   BIGINT  NOT NULL,        -- Trade Product (brand) reachable from TPP
    tpp_tp_pt      TEXT    NOT NULL,
    tpuu_tp_sctid  BIGINT  NOT NULL,        -- Trade Product reachable from TPUU
    tpuu_tp_pt     TEXT    NOT NULL,
    mpp_sctid      BIGINT  NOT NULL,
    mpp_pt         TEXT    NOT NULL,
    mpuu_sctid     BIGINT  NOT NULL,
    mpuu_pt        TEXT    NOT NULL,
    mp_sctid       BIGINT  NOT NULL,        -- Medicinal Product (substance) — the ACOP "what drug?" anchor
    mp_pt          TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS kb7_amt_load_log (
    load_id       SERIAL    PRIMARY KEY,
    release_date  DATE      NOT NULL,
    source_file   TEXT      NOT NULL,
    rows_loaded   BIGINT    NOT NULL,
    sha256        TEXT,
    loaded_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
