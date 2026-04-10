-- CKM Stage 4 subcategorization migration.
-- Changes ckm_stage from integer 0-4 to ckm_stage_v2 VARCHAR(5) with 4a/4b/4c.
-- Existing Stage 4 patients default to "4a" (safest: triggers preventive pathway).

-- Step 1: Add new columns
ALTER TABLE patient_profiles
    ADD COLUMN IF NOT EXISTS ckm_stage_v2 VARCHAR(5),
    ADD COLUMN IF NOT EXISTS ckm_substage_metadata JSONB,
    ADD COLUMN IF NOT EXISTS ckm_substage_review_needed BOOLEAN DEFAULT FALSE;

-- Step 2: Migrate existing integer ckm_stage to string
UPDATE patient_profiles SET ckm_stage_v2 = CASE
    WHEN ckm_stage = 0 THEN '0'
    WHEN ckm_stage = 1 THEN '1'
    WHEN ckm_stage = 2 THEN '2'
    WHEN ckm_stage = 3 THEN '3'
    WHEN ckm_stage = 4 THEN '4a'
    ELSE '0'
END
WHERE ckm_stage_v2 IS NULL;

-- Step 3: Flag existing Stage 4 patients for clinician review
UPDATE patient_profiles
    SET ckm_substage_review_needed = TRUE,
        ckm_substage_metadata = jsonb_build_object(
            'review_needed', true,
            'staging_source', 'MIGRATION',
            'staging_date', NOW()
        )
WHERE ckm_stage = 4;

-- Step 4: Create indexes
CREATE INDEX IF NOT EXISTS idx_pp_ckm_v2 ON patient_profiles(ckm_stage_v2);
CREATE INDEX IF NOT EXISTS idx_pp_ckm_review ON patient_profiles(ckm_substage_review_needed)
    WHERE ckm_substage_review_needed = TRUE;

-- Step 5: Add check constraint
ALTER TABLE patient_profiles
    ADD CONSTRAINT chk_ckm_stage_v2
    CHECK (ckm_stage_v2 IN ('0', '1', '2', '3', '4a', '4b', '4c'));
