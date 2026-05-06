-- ============================================================================
-- Migration 018 — CFS / AKPS / DBI / ACB scoring instrument tables + seeds
-- Layer 2 substrate plan, Wave 2.6: clinical scoring instruments.
-- See Layer2_Implementation_Guidelines.md §2.4 / §2.6 and
-- docs/superpowers/plans/2026-05-04-layer2-substrate-plan.md (lines 415-431).
--
-- Four instruments enter the substrate as separate append-only history
-- tables. CFS / AKPS are clinician-entered (capture only). DBI / ACB are
-- computed from the resident's active MedicineUse list using weighted
-- seed tables. All four inform care intensity decisions WITHOUT
-- automating transitions (Layer 2 doc §2.4 line 540-547 explicitly
-- preserves clinician judgement).
--
-- The history tables are append-only: never UPDATE rows. The latest row
-- by assessed_at (CFS/AKPS) or computed_at (DBI/ACB) per resident_ref is
-- the current score (queried via the four *_current views).
--
-- Service-layer hooks (kb-20 internal/storage/scoring_store.go):
--   - CFS>=7 or AKPS<=40 surfaces an EvidenceTrace hint with
--     state_machine=ClinicalState and
--     state_change_type=care_intensity_review_suggested. Layer 4 worklist
--     consumes the hint. The substrate NEVER writes a
--     care_intensity_history row from a score.
--   - Any MedicineUse insert/update/end triggers a service-layer
--     recompute that writes new dbi_scores + acb_scores rows. Recompute
--     is best-effort: failure MUST NOT fail the underlying MedicineUse
--     write.
--
-- MVP coverage: 20 high-frequency aged-care medications per the Hilmer
-- 2007 (DBI) and Boustani 2008 (ACB) seed lists. TODO: expand to
-- top-100 aged-care medications in a follow-up wave once the formulary
-- working group ratifies the full list.
-- ============================================================================

BEGIN;

-- ----------------------------------------------------------------------------
-- CFS — Clinical Frailty Scale (Rockwood 2020 revision)
-- ----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS cfs_scores (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resident_ref          UUID NOT NULL,
    assessed_at           TIMESTAMPTZ NOT NULL,
    assessor_role_ref     UUID NOT NULL,
    instrument_version    TEXT NOT NULL,
    score                 INTEGER NOT NULL CHECK (score BETWEEN 1 AND 9),
    rationale             TEXT,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- One row per (resident, assessed_at) — prevents two distinct
    -- assessments from claiming the same effective timestamp, which
    -- would make cfs_current ambiguous.
    UNIQUE (resident_ref, assessed_at)
);

COMMENT ON TABLE cfs_scores IS
    'Append-only Clinical Frailty Scale (Rockwood 2020) score history. Wave 2.6 (Layer 2 §2.4 / §2.6). Latest assessed_at per resident_ref is current. Score in [1,9]. CFS>=7 surfaces a care-intensity review hint via EvidenceTrace; substrate never auto-transitions.';

CREATE INDEX IF NOT EXISTS idx_cfs_scores_resident_assessed
    ON cfs_scores(resident_ref, assessed_at DESC);

CREATE OR REPLACE VIEW cfs_current AS
SELECT DISTINCT ON (resident_ref)
    id, resident_ref, assessed_at, assessor_role_ref,
    instrument_version, score, rationale, created_at
FROM cfs_scores
ORDER BY resident_ref, assessed_at DESC;

COMMENT ON VIEW cfs_current IS
    'Latest cfs_scores row per resident_ref by assessed_at DESC. Backed by idx_cfs_scores_resident_assessed.';

-- ----------------------------------------------------------------------------
-- AKPS — Australia-modified Karnofsky Performance Status (Abernethy 2005)
-- ----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS akps_scores (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resident_ref          UUID NOT NULL,
    assessed_at           TIMESTAMPTZ NOT NULL,
    assessor_role_ref     UUID NOT NULL,
    instrument_version    TEXT NOT NULL,
    score                 INTEGER NOT NULL CHECK (score BETWEEN 0 AND 100 AND score % 10 = 0),
    rationale             TEXT,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (resident_ref, assessed_at)
);

COMMENT ON TABLE akps_scores IS
    'Append-only Australia-modified Karnofsky Performance Status (Abernethy 2005) score history. Wave 2.6 (Layer 2 §2.4 / §2.6). Score in [0,100], multiples of 10. AKPS<=40 surfaces a care-intensity review hint via EvidenceTrace; substrate never auto-transitions.';

CREATE INDEX IF NOT EXISTS idx_akps_scores_resident_assessed
    ON akps_scores(resident_ref, assessed_at DESC);

CREATE OR REPLACE VIEW akps_current AS
SELECT DISTINCT ON (resident_ref)
    id, resident_ref, assessed_at, assessor_role_ref,
    instrument_version, score, rationale, created_at
FROM akps_scores
ORDER BY resident_ref, assessed_at DESC;

COMMENT ON VIEW akps_current IS
    'Latest akps_scores row per resident_ref by assessed_at DESC.';

-- ----------------------------------------------------------------------------
-- DBI — Drug Burden Index (Hilmer 2007), computed
-- ----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS dbi_scores (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resident_ref                UUID NOT NULL,
    computed_at                 TIMESTAMPTZ NOT NULL,
    score                       DOUBLE PRECISION NOT NULL CHECK (score >= 0),
    anticholinergic_component   DOUBLE PRECISION NOT NULL CHECK (anticholinergic_component >= 0),
    sedative_component          DOUBLE PRECISION NOT NULL CHECK (sedative_component >= 0),
    computation_inputs          UUID[] NOT NULL DEFAULT '{}',
    unknown_drugs               TEXT[] NOT NULL DEFAULT '{}',
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE dbi_scores IS
    'Append-only Drug Burden Index (Hilmer 2007) recompute history. Wave 2.6 (Layer 2 §2.6). Score = anticholinergic_component + sedative_component (calculator invariant). Recomputed on every MedicineUse insert/update/end.';

CREATE INDEX IF NOT EXISTS idx_dbi_scores_resident_computed
    ON dbi_scores(resident_ref, computed_at DESC);

CREATE OR REPLACE VIEW dbi_current AS
SELECT DISTINCT ON (resident_ref)
    id, resident_ref, computed_at, score,
    anticholinergic_component, sedative_component,
    computation_inputs, unknown_drugs, created_at
FROM dbi_scores
ORDER BY resident_ref, computed_at DESC;

COMMENT ON VIEW dbi_current IS
    'Latest dbi_scores row per resident_ref by computed_at DESC.';

-- ----------------------------------------------------------------------------
-- ACB — Anticholinergic Cognitive Burden (Boustani 2008), computed
-- ----------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS acb_scores (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resident_ref        UUID NOT NULL,
    computed_at         TIMESTAMPTZ NOT NULL,
    score               INTEGER NOT NULL CHECK (score >= 0),
    computation_inputs  UUID[] NOT NULL DEFAULT '{}',
    unknown_drugs       TEXT[] NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE acb_scores IS
    'Append-only Anticholinergic Cognitive Burden (Boustani 2008) recompute history. Wave 2.6 (Layer 2 §2.6). Integer 1/2/3 weights summed. Recomputed on every MedicineUse insert/update/end.';

CREATE INDEX IF NOT EXISTS idx_acb_scores_resident_computed
    ON acb_scores(resident_ref, computed_at DESC);

CREATE OR REPLACE VIEW acb_current AS
SELECT DISTINCT ON (resident_ref)
    id, resident_ref, computed_at, score,
    computation_inputs, unknown_drugs, created_at
FROM acb_scores
ORDER BY resident_ref, computed_at DESC;

COMMENT ON VIEW acb_current IS
    'Latest acb_scores row per resident_ref by computed_at DESC.';

-- ----------------------------------------------------------------------------
-- Seed weight tables — DBI (Hilmer 2007) + ACB (Boustani 2008)
-- ----------------------------------------------------------------------------
--
-- Match strategy is case-insensitive prefix LIKE on the lowercased
-- MedicineUse.display_name vs amt_code_pattern. The pattern column is
-- named amt_code_pattern for forward compatibility (when we move from
-- display-name match to AMT code match in a later wave the column
-- semantics shift but the column itself stays).
--
-- MVP coverage: 20 high-frequency aged-care medications. TODO(layer2-2.6):
-- expand to top-100 aged-care medications once the formulary working
-- group ratifies the full list.

CREATE TABLE IF NOT EXISTS dbi_drug_weights (
    amt_code_pattern        TEXT PRIMARY KEY,
    drug_name               TEXT NOT NULL,
    anticholinergic_weight  DOUBLE PRECISION NOT NULL CHECK (anticholinergic_weight >= 0),
    sedative_weight         DOUBLE PRECISION NOT NULL CHECK (sedative_weight >= 0),
    source                  TEXT NOT NULL,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE dbi_drug_weights IS
    'DBI weight seed table per Hilmer 2007. amt_code_pattern matches LOWER(MedicineUse.display_name) via prefix LIKE. MVP=20 rows; TODO expand to top-100 aged-care drugs.';

CREATE TABLE IF NOT EXISTS acb_drug_weights (
    amt_code_pattern  TEXT PRIMARY KEY,
    drug_name         TEXT NOT NULL,
    weight            INTEGER NOT NULL CHECK (weight BETWEEN 1 AND 3),
    source            TEXT NOT NULL,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE acb_drug_weights IS
    'ACB weight seed table per Boustani 2008. Integer 1/2/3 (1=possible, 2=definite mild, 3=definite strong). MVP=20 rows; TODO expand to top-100.';

-- DBI seed rows (20 high-frequency aged-care medications).
INSERT INTO dbi_drug_weights (amt_code_pattern, drug_name, anticholinergic_weight, sedative_weight, source) VALUES
    ('amitriptyline',     'amitriptyline',     0.5, 0.5, 'Hilmer 2007'),
    ('oxybutynin',        'oxybutynin',        0.5, 0.0, 'Hilmer 2007'),
    ('diphenhydramine',   'diphenhydramine',   0.5, 0.5, 'Hilmer 2007'),
    ('temazepam',         'temazepam',         0.0, 0.5, 'Hilmer 2007'),
    ('diazepam',          'diazepam',          0.0, 0.5, 'Hilmer 2007'),
    ('zopiclone',         'zopiclone',         0.0, 0.5, 'Hilmer 2007'),
    ('zolpidem',          'zolpidem',          0.0, 0.5, 'Hilmer 2007'),
    ('codeine',           'codeine',           0.0, 0.5, 'Hilmer 2007'),
    ('oxycodone',         'oxycodone',         0.0, 0.5, 'Hilmer 2007'),
    ('tramadol',          'tramadol',          0.0, 0.5, 'Hilmer 2007'),
    ('morphine',          'morphine',          0.0, 0.5, 'Hilmer 2007'),
    ('quetiapine',        'quetiapine',        0.5, 0.5, 'Hilmer 2007'),
    ('risperidone',       'risperidone',       0.5, 0.0, 'Hilmer 2007'),
    ('olanzapine',        'olanzapine',        0.5, 0.5, 'Hilmer 2007'),
    ('haloperidol',       'haloperidol',       0.5, 0.0, 'Hilmer 2007'),
    ('citalopram',        'citalopram',        0.5, 0.0, 'Hilmer 2007'),
    ('paroxetine',        'paroxetine',        0.5, 0.0, 'Hilmer 2007'),
    ('mirtazapine',       'mirtazapine',       0.5, 0.5, 'Hilmer 2007'),
    ('chlorpromazine',    'chlorpromazine',    0.5, 0.5, 'Hilmer 2007'),
    ('hyoscine',          'hyoscine',          0.5, 0.0, 'Hilmer 2007')
ON CONFLICT (amt_code_pattern) DO NOTHING;

-- ACB seed rows (same 20 drugs; Boustani 2008 integer weights).
INSERT INTO acb_drug_weights (amt_code_pattern, drug_name, weight, source) VALUES
    ('amitriptyline',     'amitriptyline',     3, 'Boustani 2008'),
    ('oxybutynin',        'oxybutynin',        3, 'Boustani 2008'),
    ('diphenhydramine',   'diphenhydramine',   3, 'Boustani 2008'),
    ('temazepam',         'temazepam',         1, 'Boustani 2008'),
    ('diazepam',          'diazepam',          1, 'Boustani 2008'),
    ('zopiclone',         'zopiclone',         1, 'Boustani 2008'),
    ('zolpidem',          'zolpidem',          1, 'Boustani 2008'),
    ('codeine',           'codeine',           1, 'Boustani 2008'),
    ('oxycodone',         'oxycodone',         1, 'Boustani 2008'),
    ('tramadol',          'tramadol',          1, 'Boustani 2008'),
    ('morphine',          'morphine',          1, 'Boustani 2008'),
    ('quetiapine',        'quetiapine',        3, 'Boustani 2008'),
    ('risperidone',       'risperidone',       1, 'Boustani 2008'),
    ('olanzapine',        'olanzapine',        3, 'Boustani 2008'),
    ('haloperidol',       'haloperidol',       1, 'Boustani 2008'),
    ('citalopram',        'citalopram',        1, 'Boustani 2008'),
    ('paroxetine',        'paroxetine',        3, 'Boustani 2008'),
    ('mirtazapine',       'mirtazapine',       1, 'Boustani 2008'),
    ('chlorpromazine',    'chlorpromazine',    3, 'Boustani 2008'),
    ('hyoscine',          'hyoscine',          3, 'Boustani 2008')
ON CONFLICT (amt_code_pattern) DO NOTHING;

COMMIT;
