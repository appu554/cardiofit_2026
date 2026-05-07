-- 007_fact_type_resolution.sql
--
-- Pre-Wave Task 1 of the Layer 3 v2 Rule Encoding Plan.
-- See: claudedocs/audits/2026-05-PreWave-fact-type-decision.md
--
-- DECISION: keep `criterion_set` as the authoritative discriminator for
-- the STOPP-vs-START distinction (and for every future criterion set).
-- DO NOT add a `fact_type` column. The Layer 3 CQL library exposes the
-- discriminator semantics through a helper:
--
--   IsPrescribingOmission(criterionSet) := criterionSet = 'START_V3'
--
-- This migration ships a column-level COMMENT documenting that legend
-- so any reviewer reading the schema knows which criterion sets carry
-- which semantics. NO ROW DATA CHANGES. The migration is idempotent
-- (COMMENT ON COLUMN overwrites any prior comment).
--
-- Author: Layer 3 Pre-Wave dispatch (2026-05-06)
-- Reviewers: pending Layer 3 lead + KB-4 governance signoff (no
--            Ed25519 chain impact since no row data is touched).

BEGIN;

COMMENT ON COLUMN kb4_explicit_criteria.criterion_set IS
$comment$
Authoritative discriminator for the explicit-criteria rule provenance
and semantics. Every value below is a frozen enum understood by the
Layer 3 CQL helper library. Adding a new value REQUIRES a Layer 3
governance review because it changes the helper surface.

  STOPP_V3        -- 80 rows. Drugs to STOP in older adults
                  --          (O'Mahony et al. 2023).
                  --          Layer 3 semantics: potentially
                  --          inappropriate medication (PIM).

  START_V3        -- 40 rows. Drugs to START / prescribing omissions
                  --          in older adults (O'Mahony et al. 2023).
                  --          Layer 3 semantics: PRESCRIBING_OMISSION.
                  --          IsPrescribingOmission() helper returns
                  --          TRUE for these rows and ONLY these rows.

  BEERS_2023      -- 57 rows. AGS Beers Criteria 2023 (US PIM list,
                  --          internationally referenced).
                  --          Layer 3 semantics: PIM.

  BEERS_RENAL     -- Beers renal-adjustment subset (loaded with
                  --          BEERS_2023).
                  --          Layer 3 semantics: PIM-conditional-on-eGFR.

  ACB             -- 56 rows. Anticholinergic Cognitive Burden score
                  --          (Boustani et al. 2008, refreshed 2024).
                  --          Layer 3 semantics: drug-burden contributor.

  PIMS_WANG       -- Wang 2024 Australian PIMs list (deprescribing
                  --          candidates with AU prevalence weighting).
                  --          Layer 3 semantics: PIM (AU-tailored).

  AU_APINCHS      -- Australian APINCHS high-risk medication list
                  --          (ACSQHC). Layer 3 semantics: high-alert
                  --          medication; triggers Authorisation gating.

  AU_TGA_BLACKBOX -- TGA black-box (boxed) warnings. Layer 3 semantics:
                  --          regulatory contraindication / warning.

A Layer 3 CQL define MUST consume this column via the
`IsPrescribingOmission(criterionSet)` and `IsHighRiskFromBeers(...)`
helpers in `shared/cql-libraries/helpers/MedicationHelpers.cql`. Direct
literal comparisons against `criterion_set` are linted out by
`shared/cql-toolchain/rule_specification_validator.py` (Wave 1).
$comment$;

COMMIT;
