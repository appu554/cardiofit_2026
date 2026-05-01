-- 019_concepts_icd10_who.sql
--
-- WHO ICD-10 reference table (separate from concepts_icd10 which holds
-- ICD-10-CM, the dotless billable US Clinical Modification codes).
--
-- Why a separate table:
--   KB-7's existing concepts_icd10 holds ICD-10-CM codes like 'E1010'
--   (no dots, billable 5-char). WHO ICD-10 uses dotted codes like
--   'E10.10' AND has codes that don't exist in CM (e.g., F00 dementia
--   series, which CM moved to G30/F01-F03). Mixing them in one table
--   would conflate two distinct coding systems.
--
-- Validator behaviour: kb-7-terminology/scripts/validate_kb_codes.py
-- resolves a consumer ICD-10 code if it matches EITHER concepts_icd10
-- (after dot-strip + prefix-rollup) OR concepts_icd10_who (direct).
--
-- Reference: claudedocs/audits/Layer1_AU_AgedCare_Codebase_Gap_Audit_v2_2026-04-30.md
-- §"Quality / Integrity findings"

CREATE TABLE IF NOT EXISTS concepts_icd10_who (
    id              BIGSERIAL PRIMARY KEY,
    code            TEXT NOT NULL UNIQUE,
    preferred_term  TEXT NOT NULL,
    parent_code     TEXT,
    chapter         TEXT,
    notes           TEXT,
    loaded_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_concepts_icd10_who_code ON concepts_icd10_who (code);
CREATE INDEX IF NOT EXISTS idx_concepts_icd10_who_parent ON concepts_icd10_who (parent_code);

COMMENT ON TABLE concepts_icd10_who IS
    'WHO ICD-10 reference codes (dotted format) for KB-4 START_V3 condition '
    'codes that do not resolve in concepts_icd10 (ICD-10-CM, dotless billable).';

-- Seed: minimal set covering the 7 residuals + immediate context for aged-care
-- Sourced from WHO ICD-10 (10th revision) public tabular list. These are
-- factual codes — the WHO ICD-10 codes themselves are not copyrightable.

INSERT INTO concepts_icd10_who (code, preferred_term, parent_code, chapter) VALUES

-- F00-F03 dementia series (WHO has F00; ICD-10-CM moved to G30/F01-F03)
('F00',     'Dementia in Alzheimer disease',                              NULL,  'F'),
('F00.0',   'Dementia in Alzheimer disease with early onset',             'F00', 'F'),
('F00.1',   'Dementia in Alzheimer disease with late onset',              'F00', 'F'),
('F00.2',   'Dementia in Alzheimer disease, atypical or mixed type',      'F00', 'F'),
('F00.9',   'Dementia in Alzheimer disease, unspecified',                 'F00', 'F'),
('F01',     'Vascular dementia',                                          NULL,  'F'),
('F02',     'Dementia in other diseases classified elsewhere',            NULL,  'F'),
('F03',     'Unspecified dementia',                                       NULL,  'F'),

-- H40 glaucoma series (WHO uses .10/.11/.12/.13 subcodes; CM uses different shape)
('H40',     'Glaucoma',                                                    NULL,  'H'),
('H40.0',   'Glaucoma suspect',                                            'H40', 'H'),
('H40.1',   'Primary open-angle glaucoma',                                 'H40', 'H'),
('H40.10',  'Primary open-angle glaucoma',                                 'H40', 'H'),
('H40.11',  'Low-tension glaucoma',                                        'H40', 'H'),
('H40.12',  'Pigmentary glaucoma',                                         'H40', 'H'),
('H40.13',  'Capsular glaucoma with pseudoexfoliation of lens',            'H40', 'H'),
('H40.2',   'Primary angle-closure glaucoma',                              'H40', 'H'),

-- J46 status asthmaticus (WHO has dedicated code; CM merged into J45.x)
('J45',     'Asthma',                                                      NULL,  'J'),
('J46',     'Status asthmaticus',                                          NULL,  'J'),

-- M82 osteoporosis in diseases (WHO standalone; CM in M80.x)
('M80',     'Osteoporosis with current pathological fracture',             NULL,  'M'),
('M81',     'Osteoporosis without current pathological fracture',          NULL,  'M'),
('M82',     'Osteoporosis in diseases classified elsewhere',               NULL,  'M'),
('M82.0',   'Osteoporosis in multiple myelomatosis',                       'M82', 'M'),
('M82.1',   'Osteoporosis in endocrine disorders',                         'M82', 'M'),
('M82.8',   'Osteoporosis in other diseases classified elsewhere',         'M82', 'M')

ON CONFLICT (code) DO NOTHING;
