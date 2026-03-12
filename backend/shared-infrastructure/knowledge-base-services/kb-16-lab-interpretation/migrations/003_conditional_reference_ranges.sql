-- =============================================================================
-- Migration 003: Conditional Reference Ranges for Context-Aware Lab Interpretation
-- =============================================================================
-- Phase 3b.6: Transform KB-16 into Context-Aware Clinical Reference Engine
--
-- PURPOSE:
--   Traditional reference ranges (e.g., Hgb 12-16 g/dL) apply to HEALTHY adults.
--   This migration adds CONDITIONAL ranges based on patient context:
--     - Pregnancy (by trimester) - ACOG, ATA guidelines
--     - CKD Stage (1-5 + dialysis) - KDIGO guidelines
--     - Age (neonate, pediatric, adult, geriatric) - CLSI C28
--     - Neonatal bilirubin nomogram - AAP 2022 guidelines
--
-- CLINICAL IMPORTANCE:
--   ❌ Hemoglobin 11 g/dL with standard range (12-16): ABNORMAL
--   ✅ Hemoglobin 11 g/dL with pregnancy T3 range (10.5-14): NORMAL
--   Using wrong range = dangerous clinical decisions
--
-- AUTHORITY SOURCES:
--   - CLSI C28-A3c (2024): Reference interval methodology
--   - ACOG Practice Bulletins: Pregnancy laboratory values
--   - ATA 2017 Guidelines: Thyroid function in pregnancy
--   - KDIGO 2024: CKD-specific laboratory targets
--   - AAP 2022: Neonatal hyperbilirubinemia guidelines (Bhutani nomogram)
-- =============================================================================

BEGIN;

-- =============================================================================
-- 1. LAB TESTS TABLE (Centralized LOINC Definitions)
-- =============================================================================
-- Purpose: Master list of laboratory tests with LOINC codes
-- This allows multiple conditional ranges to reference a single test definition
-- =============================================================================

CREATE TABLE IF NOT EXISTS lab_tests (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    loinc_code          VARCHAR(20) NOT NULL UNIQUE,
    test_name           VARCHAR(200) NOT NULL,
    short_name          VARCHAR(50),
    unit                VARCHAR(50) NOT NULL,
    specimen_type       VARCHAR(50),           -- blood, urine, csf, plasma, serum
    method              VARCHAR(100),          -- enzymatic, colorimetric, immunoassay
    category            VARCHAR(50),           -- Chemistry, Hematology, Coagulation, Endocrine
    decimal_places      INTEGER DEFAULT 2,
    trending_enabled    BOOLEAN DEFAULT TRUE,
    is_active           BOOLEAN DEFAULT TRUE,
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    updated_at          TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for lab_tests
CREATE INDEX IF NOT EXISTS idx_lab_tests_loinc ON lab_tests(loinc_code);
CREATE INDEX IF NOT EXISTS idx_lab_tests_category ON lab_tests(category) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_lab_tests_active ON lab_tests(is_active) WHERE is_active = TRUE;

-- =============================================================================
-- 2. CONDITIONAL REFERENCE RANGES TABLE
-- =============================================================================
-- Purpose: Context-aware reference ranges with patient condition matching
-- Key design: NULL condition = matches any patient for that condition
--             Non-NULL condition = MUST match for range to apply
-- =============================================================================

CREATE TABLE IF NOT EXISTS conditional_reference_ranges (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lab_test_id         UUID NOT NULL REFERENCES lab_tests(id) ON DELETE CASCADE,

    -- ==========================================================================
    -- CONDITIONS (null = any, non-null = must match)
    -- ==========================================================================

    -- Demographics
    gender              VARCHAR(1),             -- M, F, null=any
    age_min_years       DECIMAL(5,2),           -- Minimum age in years
    age_max_years       DECIMAL(5,2),           -- Maximum age in years (exclusive)
    age_min_days        INTEGER,                -- For neonates (days of life)
    age_max_days        INTEGER,                -- For neonates (days of life)

    -- Pregnancy & Lactation (ACOG, ATA guidelines)
    is_pregnant         BOOLEAN,                -- TRUE for pregnancy ranges
    trimester           INTEGER CHECK (trimester BETWEEN 1 AND 3),  -- 1, 2, 3
    is_postpartum       BOOLEAN,                -- Postpartum period
    postpartum_weeks    INTEGER,                -- Weeks postpartum
    is_lactating        BOOLEAN,                -- Breastfeeding status

    -- Neonatal (AAP 2022 bilirubin guidelines)
    gestational_age_weeks_min INTEGER,          -- GA in weeks (for preterm)
    gestational_age_weeks_max INTEGER,          -- GA in weeks (upper bound)
    hours_of_life_min   INTEGER,                -- Hours since birth (for bili nomogram)
    hours_of_life_max   INTEGER,                -- Hours since birth (upper bound)

    -- Renal Status (KDIGO guidelines)
    ckd_stage           INTEGER CHECK (ckd_stage BETWEEN 1 AND 5),  -- CKD Stage 1-5
    is_on_dialysis      BOOLEAN,                -- Dialysis status
    egfr_min            DECIMAL(6,2),           -- eGFR minimum (mL/min/1.73m²)
    egfr_max            DECIMAL(6,2),           -- eGFR maximum (mL/min/1.73m²)

    -- ==========================================================================
    -- REFERENCE VALUES
    -- ==========================================================================
    low_normal          DECIMAL(10,4),          -- Lower limit of normal
    high_normal         DECIMAL(10,4),          -- Upper limit of normal
    critical_low        DECIMAL(10,4),          -- Critical low (requires notification)
    critical_high       DECIMAL(10,4),          -- Critical high (requires notification)
    panic_low           DECIMAL(10,4),          -- Panic low (immediate action)
    panic_high          DECIMAL(10,4),          -- Panic high (immediate action)

    -- ==========================================================================
    -- INTERPRETATION GUIDANCE
    -- ==========================================================================
    interpretation_note TEXT,                   -- Context-specific interpretation
    clinical_action     TEXT,                   -- Recommended action if abnormal

    -- ==========================================================================
    -- GOVERNANCE (Authority & Version Tracking)
    -- ==========================================================================
    authority           VARCHAR(50) NOT NULL,   -- CLSI, ACOG, ATA, KDIGO, AAP
    authority_reference TEXT NOT NULL,          -- Specific document/table
    authority_version   VARCHAR(50),            -- Version/year of guideline
    effective_date      DATE NOT NULL,          -- When this range became effective
    expiration_date     DATE,                   -- When this range expires (null = current)

    -- ==========================================================================
    -- SPECIFICITY SCORING
    -- ==========================================================================
    -- Higher score = more specific condition = wins when multiple ranges match
    -- Example: Pregnancy T3 Female (score=3) beats generic Female (score=1)
    specificity_score   INTEGER DEFAULT 0,

    -- ==========================================================================
    -- METADATA
    -- ==========================================================================
    is_active           BOOLEAN DEFAULT TRUE,
    created_at          TIMESTAMPTZ DEFAULT NOW(),
    updated_at          TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- 3. NEONATAL BILIRUBIN THRESHOLDS (AAP 2022 Bhutani Nomogram)
-- =============================================================================
-- Purpose: Hour-of-life based phototherapy/exchange thresholds
-- This is separate because it requires interpolation, not just comparison
-- =============================================================================

CREATE TABLE IF NOT EXISTS neonatal_bilirubin_thresholds (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Risk Stratification
    gestational_age_weeks_min INTEGER NOT NULL, -- GA lower bound (weeks)
    gestational_age_weeks_max INTEGER NOT NULL, -- GA upper bound (weeks)
    risk_category       VARCHAR(20) NOT NULL CHECK (risk_category IN ('LOW', 'MEDIUM', 'HIGH')),

    -- Hour-of-Life Threshold Points (for interpolation)
    hour_of_life        INTEGER NOT NULL,       -- Specific hour point

    -- Treatment Thresholds (mg/dL)
    photo_threshold     DECIMAL(5,2) NOT NULL,  -- Start phototherapy above this
    exchange_threshold  DECIMAL(5,2),           -- Consider exchange transfusion

    -- Governance
    authority           VARCHAR(50) DEFAULT 'AAP',
    authority_reference TEXT DEFAULT 'AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022',

    -- Metadata
    created_at          TIMESTAMPTZ DEFAULT NOW(),

    -- Unique: one threshold per GA range/risk/hour combination
    UNIQUE(gestational_age_weeks_min, gestational_age_weeks_max, risk_category, hour_of_life)
);

-- =============================================================================
-- 4. INDEXES
-- =============================================================================

-- Conditional reference ranges indexes
CREATE INDEX IF NOT EXISTS idx_crr_lab_test ON conditional_reference_ranges(lab_test_id) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_crr_pregnancy ON conditional_reference_ranges(is_pregnant, trimester) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_crr_ckd ON conditional_reference_ranges(ckd_stage) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_crr_dialysis ON conditional_reference_ranges(is_on_dialysis) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_crr_age_years ON conditional_reference_ranges(age_min_years, age_max_years) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_crr_age_days ON conditional_reference_ranges(age_min_days, age_max_days) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_crr_gender ON conditional_reference_ranges(gender) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_crr_neonatal ON conditional_reference_ranges(hours_of_life_min, gestational_age_weeks_min) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_crr_authority ON conditional_reference_ranges(authority) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_crr_specificity ON conditional_reference_ranges(specificity_score DESC) WHERE is_active = TRUE;
CREATE INDEX IF NOT EXISTS idx_crr_effective ON conditional_reference_ranges(effective_date, expiration_date) WHERE is_active = TRUE;

-- Neonatal bilirubin indexes
CREATE INDEX IF NOT EXISTS idx_bili_ga ON neonatal_bilirubin_thresholds(gestational_age_weeks_min, gestational_age_weeks_max);
CREATE INDEX IF NOT EXISTS idx_bili_hour ON neonatal_bilirubin_thresholds(hour_of_life);
CREATE INDEX IF NOT EXISTS idx_bili_risk ON neonatal_bilirubin_thresholds(risk_category);
CREATE INDEX IF NOT EXISTS idx_bili_ga_risk_hour ON neonatal_bilirubin_thresholds(gestational_age_weeks_min, risk_category, hour_of_life);

-- =============================================================================
-- 5. TRIGGERS
-- =============================================================================

-- Updated_at trigger for lab_tests
DROP TRIGGER IF EXISTS update_lab_tests_updated_at ON lab_tests;
CREATE TRIGGER update_lab_tests_updated_at
    BEFORE UPDATE ON lab_tests
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Updated_at trigger for conditional_reference_ranges
DROP TRIGGER IF EXISTS update_crr_updated_at ON conditional_reference_ranges;
CREATE TRIGGER update_crr_updated_at
    BEFORE UPDATE ON conditional_reference_ranges
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- =============================================================================
-- 6. SEED DATA: LAB TESTS (LOINC Codes)
-- =============================================================================

INSERT INTO lab_tests (loinc_code, test_name, short_name, unit, specimen_type, category, decimal_places)
VALUES
    -- Hematology
    ('718-7', 'Hemoglobin [Mass/volume] in Blood', 'Hgb', 'g/dL', 'blood', 'Hematology', 1),
    ('777-3', 'Platelets [#/volume] in Blood', 'Plt', 'k/µL', 'blood', 'Hematology', 0),
    ('4544-3', 'Hematocrit [Volume Fraction] of Blood', 'Hct', '%', 'blood', 'Hematology', 1),
    ('789-8', 'Red blood cells [#/volume] in Blood', 'RBC', 'M/µL', 'blood', 'Hematology', 2),
    ('6690-2', 'White blood cells [#/volume] in Blood', 'WBC', 'k/µL', 'blood', 'Hematology', 2),

    -- Chemistry - Renal
    ('2160-0', 'Creatinine [Mass/volume] in Serum or Plasma', 'Cr', 'mg/dL', 'serum', 'Chemistry', 2),
    ('3094-0', 'Urea nitrogen [Mass/volume] in Serum or Plasma', 'BUN', 'mg/dL', 'serum', 'Chemistry', 1),
    ('33914-3', 'eGFR CKD-EPI [Volume Rate] in Serum/Plasma', 'eGFR', 'mL/min/1.73m²', 'serum', 'Chemistry', 0),

    -- Chemistry - Electrolytes
    ('2823-3', 'Potassium [Moles/volume] in Serum or Plasma', 'K', 'mEq/L', 'serum', 'Chemistry', 1),
    ('2951-2', 'Sodium [Moles/volume] in Serum or Plasma', 'Na', 'mEq/L', 'serum', 'Chemistry', 0),
    ('17861-6', 'Calcium [Mass/volume] in Serum or Plasma', 'Ca', 'mg/dL', 'serum', 'Chemistry', 1),
    ('2777-1', 'Phosphate [Mass/volume] in Serum or Plasma', 'Phos', 'mg/dL', 'serum', 'Chemistry', 1),
    ('2601-3', 'Magnesium [Mass/volume] in Serum or Plasma', 'Mg', 'mg/dL', 'serum', 'Chemistry', 1),

    -- Endocrine - Thyroid
    ('3016-3', 'TSH [Units/volume] in Serum or Plasma', 'TSH', 'mIU/L', 'serum', 'Endocrine', 2),
    ('3024-7', 'Free T4 [Mass/volume] in Serum or Plasma', 'fT4', 'ng/dL', 'serum', 'Endocrine', 2),
    ('3053-6', 'Free T3 [Mass/volume] in Serum or Plasma', 'fT3', 'pg/mL', 'serum', 'Endocrine', 1),

    -- Endocrine - Other
    ('2132-9', 'Parathyroid hormone [Mass/volume] in Serum or Plasma', 'PTH', 'pg/mL', 'serum', 'Endocrine', 0),

    -- Coagulation
    ('3173-2', 'Fibrinogen [Mass/volume] in Platelet poor plasma', 'Fibrinogen', 'mg/dL', 'plasma', 'Coagulation', 0),
    ('6301-6', 'INR', 'INR', 'ratio', 'plasma', 'Coagulation', 1),

    -- Chemistry - Liver/Metabolic
    ('3084-1', 'Uric acid [Mass/volume] in Serum or Plasma', 'Uric Acid', 'mg/dL', 'serum', 'Chemistry', 1),
    ('1920-8', 'AST [Enzymatic activity/volume] in Serum or Plasma', 'AST', 'U/L', 'serum', 'Chemistry', 0),
    ('1742-6', 'ALT [Enzymatic activity/volume] in Serum or Plasma', 'ALT', 'U/L', 'serum', 'Chemistry', 0),

    -- Neonatal
    ('1975-2', 'Bilirubin.total [Mass/volume] in Serum or Plasma', 'T.Bili', 'mg/dL', 'serum', 'Chemistry', 1),
    ('1971-1', 'Bilirubin.direct [Mass/volume] in Serum or Plasma', 'D.Bili', 'mg/dL', 'serum', 'Chemistry', 1),

    -- Iron Studies
    ('2498-4', 'Ferritin [Mass/volume] in Serum or Plasma', 'Ferritin', 'ng/mL', 'serum', 'Chemistry', 0),
    ('14800-7', 'Iron [Mass/volume] in Serum or Plasma', 'Iron', 'µg/dL', 'serum', 'Chemistry', 0),
    ('2500-7', 'Transferrin saturation', 'TSAT', '%', 'serum', 'Chemistry', 0)

ON CONFLICT (loinc_code) DO UPDATE SET
    test_name = EXCLUDED.test_name,
    unit = EXCLUDED.unit,
    updated_at = NOW();

-- =============================================================================
-- 7. SEED DATA: STANDARD ADULT REFERENCE RANGES (CLSI)
-- =============================================================================

-- Helper function to get lab_test_id by LOINC
CREATE OR REPLACE FUNCTION get_lab_test_id(p_loinc VARCHAR(20))
RETURNS UUID AS $$
    SELECT id FROM lab_tests WHERE loinc_code = p_loinc;
$$ LANGUAGE SQL STABLE;

-- Standard Adult Ranges (Male)
INSERT INTO conditional_reference_ranges
(lab_test_id, gender, age_min_years, age_max_years, low_normal, high_normal, critical_low, critical_high, panic_low, panic_high, authority, authority_reference, authority_version, effective_date, specificity_score, interpretation_note)
VALUES
    -- Hemoglobin - Adult Male
    (get_lab_test_id('718-7'), 'M', 18, 120, 14.0, 18.0, 7.0, 20.0, 5.0, 22.0,
     'CLSI', 'CLSI C28-A3c Reference Intervals', '2024', '2024-01-01', 2,
     'Standard adult male hemoglobin reference range'),

    -- Creatinine - Adult Male
    (get_lab_test_id('2160-0'), 'M', 18, 120, 0.74, 1.35, 0.4, 10.0, NULL, 12.0,
     'CLSI', 'CLSI C28-A3c Reference Intervals', '2024', '2024-01-01', 2,
     'Standard adult male creatinine reference range'),

    -- Ferritin - Adult Male
    (get_lab_test_id('2498-4'), 'M', 18, 120, 30, 400, 10, 1000, NULL, NULL,
     'CLSI', 'CLSI C28-A3c Reference Intervals', '2024', '2024-01-01', 2,
     'Standard adult male ferritin reference range');

-- Standard Adult Ranges (Female)
INSERT INTO conditional_reference_ranges
(lab_test_id, gender, age_min_years, age_max_years, low_normal, high_normal, critical_low, critical_high, panic_low, panic_high, authority, authority_reference, authority_version, effective_date, specificity_score, interpretation_note)
VALUES
    -- Hemoglobin - Adult Female (non-pregnant)
    (get_lab_test_id('718-7'), 'F', 18, 120, 12.0, 16.0, 7.0, 18.0, 5.0, 20.0,
     'CLSI', 'CLSI C28-A3c Reference Intervals', '2024', '2024-01-01', 2,
     'Standard adult female hemoglobin reference range (non-pregnant)'),

    -- Creatinine - Adult Female
    (get_lab_test_id('2160-0'), 'F', 18, 120, 0.59, 1.04, 0.3, 10.0, NULL, 12.0,
     'CLSI', 'CLSI C28-A3c Reference Intervals', '2024', '2024-01-01', 2,
     'Standard adult female creatinine reference range'),

    -- Ferritin - Adult Female
    (get_lab_test_id('2498-4'), 'F', 18, 120, 13, 150, 5, 1000, NULL, NULL,
     'CLSI', 'CLSI C28-A3c Reference Intervals', '2024', '2024-01-01', 2,
     'Standard adult female ferritin reference range');

-- Standard Adult Ranges (Any Gender)
INSERT INTO conditional_reference_ranges
(lab_test_id, gender, age_min_years, age_max_years, low_normal, high_normal, critical_low, critical_high, panic_low, panic_high, authority, authority_reference, authority_version, effective_date, specificity_score, interpretation_note)
VALUES
    -- Potassium - Adult
    (get_lab_test_id('2823-3'), NULL, 18, 120, 3.5, 5.1, 2.5, 6.0, 2.0, 7.0,
     'CLSI', 'CLSI C28-A3c Reference Intervals', '2024', '2024-01-01', 1,
     'Standard adult potassium reference range'),

    -- Sodium - Adult
    (get_lab_test_id('2951-2'), NULL, 18, 120, 136, 145, 120, 160, 115, 165,
     'CLSI', 'CLSI C28-A3c Reference Intervals', '2024', '2024-01-01', 1,
     'Standard adult sodium reference range'),

    -- Phosphate - Adult
    (get_lab_test_id('2777-1'), NULL, 18, 120, 2.5, 4.5, 1.0, 7.0, 0.5, 9.0,
     'CLSI', 'CLSI C28-A3c Reference Intervals', '2024', '2024-01-01', 1,
     'Standard adult phosphate reference range'),

    -- Calcium - Adult
    (get_lab_test_id('17861-6'), NULL, 18, 120, 8.6, 10.2, 6.0, 13.0, 5.0, 14.0,
     'CLSI', 'CLSI C28-A3c Reference Intervals', '2024', '2024-01-01', 1,
     'Standard adult calcium reference range'),

    -- TSH - Adult (non-pregnant)
    (get_lab_test_id('3016-3'), NULL, 18, 120, 0.4, 4.0, 0.01, 10.0, NULL, 100.0,
     'CLSI', 'CLSI C28-A3c Reference Intervals', '2024', '2024-01-01', 1,
     'Standard adult TSH reference range (non-pregnant)'),

    -- PTH - Adult
    (get_lab_test_id('2132-9'), NULL, 18, 120, 15, 65, NULL, 300, NULL, NULL,
     'CLSI', 'CLSI C28-A3c Reference Intervals', '2024', '2024-01-01', 1,
     'Standard adult PTH reference range'),

    -- Platelets - Adult
    (get_lab_test_id('777-3'), NULL, 18, 120, 150, 400, 50, 1000, 20, 1500,
     'CLSI', 'CLSI C28-A3c Reference Intervals', '2024', '2024-01-01', 1,
     'Standard adult platelet count reference range'),

    -- Fibrinogen - Adult
    (get_lab_test_id('3173-2'), NULL, 18, 120, 200, 400, 100, 800, NULL, NULL,
     'CLSI', 'CLSI C28-A3c Reference Intervals', '2024', '2024-01-01', 1,
     'Standard adult fibrinogen reference range'),

    -- Uric Acid - Adult
    (get_lab_test_id('3084-1'), NULL, 18, 120, 3.5, 7.2, 1.0, 12.0, NULL, 15.0,
     'CLSI', 'CLSI C28-A3c Reference Intervals', '2024', '2024-01-01', 1,
     'Standard adult uric acid reference range'),

    -- AST - Adult
    (get_lab_test_id('1920-8'), NULL, 18, 120, 10, 40, NULL, 500, NULL, 1000,
     'CLSI', 'CLSI C28-A3c Reference Intervals', '2024', '2024-01-01', 1,
     'Standard adult AST reference range'),

    -- ALT - Adult
    (get_lab_test_id('1742-6'), NULL, 18, 120, 7, 56, NULL, 500, NULL, 1000,
     'CLSI', 'CLSI C28-A3c Reference Intervals', '2024', '2024-01-01', 1,
     'Standard adult ALT reference range'),

    -- Total Bilirubin - Adult
    (get_lab_test_id('1975-2'), NULL, 18, 120, 0.1, 1.2, NULL, 10.0, NULL, 15.0,
     'CLSI', 'CLSI C28-A3c Reference Intervals', '2024', '2024-01-01', 1,
     'Standard adult total bilirubin reference range');

-- =============================================================================
-- 8. SEED DATA: PREGNANCY-SPECIFIC RANGES (ACOG, ATA)
-- =============================================================================

-- Pregnancy Trimester 1 Ranges
INSERT INTO conditional_reference_ranges
(lab_test_id, gender, is_pregnant, trimester, low_normal, high_normal, critical_low, critical_high, panic_low, panic_high, authority, authority_reference, authority_version, effective_date, specificity_score, interpretation_note, clinical_action)
VALUES
    -- Hemoglobin - Pregnancy T1
    (get_lab_test_id('718-7'), 'F', TRUE, 1, 11.0, 14.0, 7.0, 16.0, 5.0, 18.0,
     'ACOG', 'ACOG Practice Bulletin: Anemia in Pregnancy', '2021', '2021-08-01', 5,
     'First trimester: Hgb ≥11 g/dL recommended. Early pregnancy physiologic changes begin.',
     'If Hgb <11: Consider iron supplementation and dietary counseling'),

    -- Creatinine - Pregnancy T1
    (get_lab_test_id('2160-0'), 'F', TRUE, 1, 0.4, 0.7, 0.2, 1.5, NULL, 2.0,
     'ACOG', 'ACOG Practice Bulletin: Chronic Kidney Disease in Pregnancy', '2019', '2019-01-01', 5,
     'First trimester: Creatinine lower due to increased GFR. Normal non-pregnant values may indicate renal dysfunction.',
     'If Cr >0.8: Evaluate for underlying renal disease'),

    -- TSH - Pregnancy T1 (ATA 2017 Guidelines)
    (get_lab_test_id('3016-3'), 'F', TRUE, 1, 0.1, 2.5, NULL, 10.0, NULL, 50.0,
     'ATA', 'ATA Guidelines for Thyroid Disease in Pregnancy', '2017', '2017-01-01', 5,
     'First trimester: Lower TSH due to hCG thyroid stimulation. Upper limit 2.5 mIU/L.',
     'If TSH >2.5: Consider levothyroxine; monitor closely'),

    -- Platelets - Pregnancy T1
    (get_lab_test_id('777-3'), 'F', TRUE, 1, 150, 400, 50, 600, 20, 800,
     'ACOG', 'ACOG Practice Bulletin: Thrombocytopenia in Pregnancy', '2019', '2019-03-01', 5,
     'First trimester: Platelet count typically stable. Gestational thrombocytopenia develops later.',
     NULL),

    -- Fibrinogen - Pregnancy T1
    (get_lab_test_id('3173-2'), 'F', TRUE, 1, 300, 500, 150, 700, 100, NULL,
     'ACOG', 'ACOG Practice Bulletin: Postpartum Hemorrhage', '2017', '2017-10-01', 5,
     'First trimester: Fibrinogen begins to rise. Important baseline for monitoring.',
     NULL);

-- Pregnancy Trimester 2 Ranges
INSERT INTO conditional_reference_ranges
(lab_test_id, gender, is_pregnant, trimester, low_normal, high_normal, critical_low, critical_high, panic_low, panic_high, authority, authority_reference, authority_version, effective_date, specificity_score, interpretation_note, clinical_action)
VALUES
    -- Hemoglobin - Pregnancy T2
    (get_lab_test_id('718-7'), 'F', TRUE, 2, 10.5, 14.0, 7.0, 16.0, 5.0, 18.0,
     'ACOG', 'ACOG Practice Bulletin: Anemia in Pregnancy', '2021', '2021-08-01', 5,
     'Second trimester: Physiologic hemodilution - plasma volume expands faster than red cell mass. Hgb ≥10.5 acceptable.',
     'If Hgb <10.5: Evaluate for iron deficiency; consider parenteral iron if oral fails'),

    -- Creatinine - Pregnancy T2
    (get_lab_test_id('2160-0'), 'F', TRUE, 2, 0.4, 0.8, 0.2, 1.5, NULL, 2.0,
     'ACOG', 'ACOG Practice Bulletin: Chronic Kidney Disease in Pregnancy', '2019', '2019-01-01', 5,
     'Second trimester: GFR peaks, creatinine at nadir. Cr >0.8 warrants evaluation.',
     'If Cr >0.8: Monitor for preeclampsia risk; nephrology consultation if rising'),

    -- TSH - Pregnancy T2
    (get_lab_test_id('3016-3'), 'F', TRUE, 2, 0.2, 3.0, NULL, 10.0, NULL, 50.0,
     'ATA', 'ATA Guidelines for Thyroid Disease in Pregnancy', '2017', '2017-01-01', 5,
     'Second trimester: TSH normalizes slightly as hCG declines. Target 0.2-3.0 mIU/L.',
     'Adjust levothyroxine dose if outside range'),

    -- Platelets - Pregnancy T2
    (get_lab_test_id('777-3'), 'F', TRUE, 2, 100, 400, 50, 600, 20, 800,
     'ACOG', 'ACOG Practice Bulletin: Thrombocytopenia in Pregnancy', '2019', '2019-03-01', 5,
     'Second trimester: Gestational thrombocytopenia may begin (mild decrease normal).',
     'If Plt <100: Rule out HELLP, ITP, preeclampsia'),

    -- Fibrinogen - Pregnancy T2
    (get_lab_test_id('3173-2'), 'F', TRUE, 2, 350, 550, 150, 700, 100, NULL,
     'ACOG', 'ACOG Practice Bulletin: Postpartum Hemorrhage', '2017', '2017-10-01', 5,
     'Second trimester: Fibrinogen continues to rise. Levels <200 mg/dL concerning for DIC.',
     'If Fibrinogen <200: Urgent evaluation for DIC/placental abruption');

-- Pregnancy Trimester 3 Ranges
INSERT INTO conditional_reference_ranges
(lab_test_id, gender, is_pregnant, trimester, low_normal, high_normal, critical_low, critical_high, panic_low, panic_high, authority, authority_reference, authority_version, effective_date, specificity_score, interpretation_note, clinical_action)
VALUES
    -- Hemoglobin - Pregnancy T3
    (get_lab_test_id('718-7'), 'F', TRUE, 3, 10.5, 14.0, 7.0, 16.0, 5.0, 18.0,
     'ACOG', 'ACOG Practice Bulletin: Anemia in Pregnancy', '2021', '2021-08-01', 5,
     'Third trimester: Maximum hemodilution. Hgb 10.5-14 g/dL normal. Critical to ensure adequate iron stores for delivery.',
     'If Hgb <10: Parenteral iron recommended; prepare for potential transfusion at delivery'),

    -- Creatinine - Pregnancy T3
    (get_lab_test_id('2160-0'), 'F', TRUE, 3, 0.4, 0.9, 0.2, 1.5, NULL, 2.0,
     'ACOG', 'ACOG Practice Bulletin: Chronic Kidney Disease in Pregnancy', '2019', '2019-01-01', 5,
     'Third trimester: GFR begins to normalize. Cr >0.9 warrants close monitoring.',
     'If Cr >0.9 or rising: Rule out preeclampsia; monitor BP and proteinuria'),

    -- TSH - Pregnancy T3
    (get_lab_test_id('3016-3'), 'F', TRUE, 3, 0.3, 3.0, NULL, 10.0, NULL, 50.0,
     'ATA', 'ATA Guidelines for Thyroid Disease in Pregnancy', '2017', '2017-01-01', 5,
     'Third trimester: TSH approaches non-pregnant levels. Maintain target 0.3-3.0 mIU/L.',
     'Continue current levothyroxine dose if stable'),

    -- Platelets - Pregnancy T3
    (get_lab_test_id('777-3'), 'F', TRUE, 3, 100, 400, 50, 600, 20, 800,
     'ACOG', 'ACOG Practice Bulletin: Thrombocytopenia in Pregnancy', '2019', '2019-03-01', 5,
     'Third trimester: Gestational thrombocytopenia common (5-8%). Plt >100k safe for epidural.',
     'If Plt <70: No epidural; monitor for HELLP syndrome; consider delivery if declining'),

    -- Fibrinogen - Pregnancy T3
    (get_lab_test_id('3173-2'), 'F', TRUE, 3, 400, 600, 150, 800, 100, NULL,
     'ACOG', 'ACOG Practice Bulletin: Postpartum Hemorrhage', '2017', '2017-10-01', 5,
     'Third trimester: Peak fibrinogen (400-600 mg/dL). Protects against postpartum hemorrhage.',
     'If Fibrinogen <300: Increased hemorrhage risk; have blood products available'),

    -- Uric Acid - Pregnancy T3
    (get_lab_test_id('3084-1'), 'F', TRUE, 3, 2.5, 5.5, 1.0, 8.0, NULL, 10.0,
     'ACOG', 'ACOG Practice Bulletin: Gestational Hypertension and Preeclampsia', '2020', '2020-06-01', 5,
     'Third trimester: Uric acid >5.5-6 mg/dL associated with preeclampsia risk.',
     'If Uric Acid >6.0: Screen for preeclampsia; check BP and proteinuria');

-- =============================================================================
-- 8.5 SEED DATA: PREGNANCY AST/ALT RANGES (AASLD) - Gap Fix
-- =============================================================================
-- Critical for HELLP syndrome detection - AST/ALT ≥2× ULN is concerning

-- AST Pregnancy Ranges (LOINC 1920-8)
INSERT INTO conditional_reference_ranges
(lab_test_id, gender, is_pregnant, trimester, low_normal, high_normal, critical_high, authority, authority_reference, authority_version, effective_date, specificity_score, interpretation_note, clinical_action)
VALUES
    -- AST - Pregnancy T1
    (get_lab_test_id('1920-8'), 'F', TRUE, 1, 10, 35, 70,
     'AASLD', 'AASLD Practice Guidance on Liver Disease in Pregnancy', '2020', '2020-01-01', 5,
     'First trimester: AST range similar to non-pregnant. AST ≥2× ULN (>70 U/L) warrants HELLP evaluation.',
     'If AST >70: Check platelets, LDH, blood smear; evaluate for HELLP syndrome'),

    -- AST - Pregnancy T2
    (get_lab_test_id('1920-8'), 'F', TRUE, 2, 10, 35, 70,
     'AASLD', 'AASLD Practice Guidance on Liver Disease in Pregnancy', '2020', '2020-01-01', 5,
     'Second trimester: AST should remain normal. Elevation warrants investigation.',
     'If AST >70: Evaluate for HELLP syndrome, acute fatty liver, or viral hepatitis'),

    -- AST - Pregnancy T3
    (get_lab_test_id('1920-8'), 'F', TRUE, 3, 10, 35, 70,
     'AASLD', 'AASLD Practice Guidance on Liver Disease in Pregnancy', '2020', '2020-01-01', 5,
     'Third trimester: AST ≥2× ULN critical marker for HELLP syndrome. Most HELLP occurs after 34 weeks.',
     'If AST >70: URGENT - evaluate for HELLP syndrome; consider delivery timing');

-- ALT Pregnancy Ranges (LOINC 1742-6)
INSERT INTO conditional_reference_ranges
(lab_test_id, gender, is_pregnant, trimester, low_normal, high_normal, critical_high, authority, authority_reference, authority_version, effective_date, specificity_score, interpretation_note, clinical_action)
VALUES
    -- ALT - Pregnancy T1
    (get_lab_test_id('1742-6'), 'F', TRUE, 1, 7, 35, 70,
     'AASLD', 'AASLD Practice Guidance on Liver Disease in Pregnancy', '2020', '2020-01-01', 5,
     'First trimester: ALT range similar to non-pregnant. Hyperemesis gravidarum may cause mild elevation.',
     NULL),

    -- ALT - Pregnancy T2
    (get_lab_test_id('1742-6'), 'F', TRUE, 2, 7, 35, 70,
     'AASLD', 'AASLD Practice Guidance on Liver Disease in Pregnancy', '2020', '2020-01-01', 5,
     'Second trimester: ALT should remain normal. Intrahepatic cholestasis typically presents T2-T3.',
     'If ALT elevated: Check bile acids to rule out intrahepatic cholestasis of pregnancy'),

    -- ALT - Pregnancy T3
    (get_lab_test_id('1742-6'), 'F', TRUE, 3, 7, 35, 70,
     'AASLD', 'AASLD Practice Guidance on Liver Disease in Pregnancy', '2020', '2020-01-01', 5,
     'Third trimester: ALT ≥2× ULN concerning for HELLP or acute fatty liver of pregnancy.',
     'If ALT >70: URGENT - evaluate for HELLP/AFLP; check platelets, bilirubin, glucose');

-- Uric Acid Pregnancy T1/T2 Ranges (Gap Fix)
INSERT INTO conditional_reference_ranges
(lab_test_id, gender, is_pregnant, trimester, low_normal, high_normal, critical_high, authority, authority_reference, authority_version, effective_date, specificity_score, interpretation_note, clinical_action)
VALUES
    -- Uric Acid - Pregnancy T1
    (get_lab_test_id('3084-1'), 'F', TRUE, 1, 2.0, 4.5, 6.0,
     'ACOG', 'ACOG Practice Bulletin: Gestational Hypertension and Preeclampsia', '2020', '2020-06-01', 5,
     'First trimester: Uric acid typically decreases early pregnancy. >4.5 mg/dL warrants monitoring.',
     'If Uric Acid >5.0: Consider baseline renal function; monitor trend'),

    -- Uric Acid - Pregnancy T2
    (get_lab_test_id('3084-1'), 'F', TRUE, 2, 2.5, 5.0, 6.5,
     'ACOG', 'ACOG Practice Bulletin: Gestational Hypertension and Preeclampsia', '2020', '2020-06-01', 5,
     'Second trimester: Rising uric acid from T1 nadir. Elevated levels may predict preeclampsia.',
     'If Uric Acid >5.5: Monitor BP closely; increasing values concerning for developing preeclampsia');

-- =============================================================================
-- 9. SEED DATA: CKD-STAGE SPECIFIC RANGES (KDIGO 2024)
-- =============================================================================

-- CKD Stage 3 (eGFR 30-59)
INSERT INTO conditional_reference_ranges
(lab_test_id, ckd_stage, low_normal, high_normal, critical_low, critical_high, panic_low, panic_high, authority, authority_reference, authority_version, effective_date, specificity_score, interpretation_note, clinical_action)
VALUES
    -- Potassium - CKD Stage 3
    (get_lab_test_id('2823-3'), 3, 3.5, 5.0, 2.5, 5.5, 2.0, 6.5,
     'KDIGO', 'KDIGO Clinical Practice Guideline for CKD', '2024', '2024-01-01', 4,
     'CKD Stage 3: Potassium retention begins. Target 3.5-5.0 mEq/L. Avoid K-sparing diuretics.',
     'If K >5.0: Review medications (ACEi/ARB, K-sparing diuretics); dietary K restriction'),

    -- Phosphate - CKD Stage 3
    (get_lab_test_id('2777-1'), 3, 2.5, 4.5, 1.0, 5.5, 0.5, 7.0,
     'KDIGO', 'KDIGO CKD-MBD Guidelines', '2017', '2017-01-01', 4,
     'CKD Stage 3: Maintain phosphate in normal range. Elevated phosphate accelerates vascular calcification.',
     'If Phos >4.5: Start phosphate binder; dietary phosphate restriction'),

    -- PTH - CKD Stage 3
    (get_lab_test_id('2132-9'), 3, 35, 70, NULL, 200, NULL, NULL,
     'KDIGO', 'KDIGO CKD-MBD Guidelines', '2017', '2017-01-01', 4,
     'CKD Stage 3: PTH begins to rise. Upper limit 2x normal acceptable.',
     'If PTH >70: Check vitamin D; consider calcitriol/analogs');

-- CKD Stage 4 (eGFR 15-29)
INSERT INTO conditional_reference_ranges
(lab_test_id, ckd_stage, low_normal, high_normal, critical_low, critical_high, panic_low, panic_high, authority, authority_reference, authority_version, effective_date, specificity_score, interpretation_note, clinical_action)
VALUES
    -- Potassium - CKD Stage 4
    (get_lab_test_id('2823-3'), 4, 3.5, 5.5, 2.5, 6.0, 2.0, 7.0,
     'KDIGO', 'KDIGO Clinical Practice Guideline for CKD', '2024', '2024-01-01', 4,
     'CKD Stage 4: Higher K tolerance but monitor closely. Upper limit 5.5 mEq/L.',
     'If K >5.5: Sodium polystyrene; consider ACEi/ARB dose reduction; prepare for dialysis'),

    -- Phosphate - CKD Stage 4
    (get_lab_test_id('2777-1'), 4, 2.5, 4.5, 1.0, 6.0, 0.5, 8.0,
     'KDIGO', 'KDIGO CKD-MBD Guidelines', '2017', '2017-01-01', 4,
     'CKD Stage 4: Phosphate control critical. Hyperphosphatemia increases mortality.',
     'If Phos >4.5: Intensify phosphate binder; dietary counseling; consider lanthanum/sevelamer'),

    -- PTH - CKD Stage 4
    (get_lab_test_id('2132-9'), 4, 70, 110, NULL, 300, NULL, NULL,
     'KDIGO', 'KDIGO CKD-MBD Guidelines', '2017', '2017-01-01', 4,
     'CKD Stage 4: PTH 2-9x upper limit normal acceptable. Avoid oversuppression.',
     'If PTH >150: Active vitamin D (calcitriol/alfacalcidol); consider cinacalcet'),

    -- Hemoglobin - CKD Stage 4 (Target Range)
    (get_lab_test_id('718-7'), 4, 10.0, 11.5, 7.0, 13.0, 5.0, 15.0,
     'KDIGO', 'KDIGO Anemia in CKD Guidelines', '2012', '2012-01-01', 4,
     'CKD Stage 4: Target Hgb 10-11.5 g/dL. Higher targets increase thrombosis risk.',
     'If Hgb <10: Initiate/increase ESA; ensure iron replete (TSAT >20%, Ferritin >100)');

-- CKD Stage 5 / Dialysis
INSERT INTO conditional_reference_ranges
(lab_test_id, ckd_stage, is_on_dialysis, low_normal, high_normal, critical_low, critical_high, panic_low, panic_high, authority, authority_reference, authority_version, effective_date, specificity_score, interpretation_note, clinical_action)
VALUES
    -- Potassium - CKD Stage 5 Dialysis
    (get_lab_test_id('2823-3'), 5, TRUE, 3.5, 6.0, 2.5, 6.5, 2.0, 7.5,
     'KDIGO', 'KDIGO Clinical Practice Guideline for CKD', '2024', '2024-01-01', 5,
     'Dialysis: Higher K tolerance. Monitor pre-dialysis levels. Target <6.0 mEq/L.',
     'If K >6.0 pre-dialysis: Urgent dialysis; check for dietary indiscretion; kayexalate'),

    -- Phosphate - CKD Stage 5 Dialysis
    (get_lab_test_id('2777-1'), 5, TRUE, 3.5, 5.5, 1.0, 7.0, 0.5, 9.0,
     'KDIGO', 'KDIGO CKD-MBD Guidelines', '2017', '2017-01-01', 5,
     'Dialysis: Phosphate target 3.5-5.5 mg/dL. Control reduces vascular calcification.',
     'If Phos >5.5: Increase dialysis frequency; adjust phosphate binder; dietary counseling'),

    -- PTH - CKD Stage 5 Dialysis
    (get_lab_test_id('2132-9'), 5, TRUE, 150, 600, NULL, 1000, NULL, NULL,
     'KDIGO', 'KDIGO CKD-MBD Guidelines', '2017', '2017-01-01', 5,
     'Dialysis: Target PTH 2-9x upper normal (150-600 pg/mL). Avoid both over/undersuppression.',
     'If PTH >600: Cinacalcet + vitamin D analog; consider parathyroidectomy if refractory'),

    -- Hemoglobin - CKD Stage 5 Dialysis (Target Range)
    (get_lab_test_id('718-7'), 5, TRUE, 10.0, 11.5, 7.0, 13.0, 5.0, 15.0,
     'KDIGO', 'KDIGO Anemia in CKD Guidelines', '2012', '2012-01-01', 5,
     'Dialysis: Target Hgb 10-11.5 g/dL. Higher targets linked to stroke/CV events in TREAT trial.',
     'If Hgb <10: Optimize ESA; ensure iron replete (TSAT >20%, Ferritin >200). Avoid Hgb >13.');

-- =============================================================================
-- 10. SEED DATA: NEONATAL BILIRUBIN THRESHOLDS (AAP 2022 Bhutani Nomogram)
-- =============================================================================

-- Low Risk (≥38 weeks, no risk factors)
INSERT INTO neonatal_bilirubin_thresholds
(gestational_age_weeks_min, gestational_age_weeks_max, risk_category, hour_of_life, photo_threshold, exchange_threshold, authority, authority_reference)
VALUES
    (38, 45, 'LOW', 12, 6.0, NULL, 'AAP', 'AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022'),
    (38, 45, 'LOW', 24, 12.0, 20.0, 'AAP', 'AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022'),
    (38, 45, 'LOW', 36, 14.0, 22.0, 'AAP', 'AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022'),
    (38, 45, 'LOW', 48, 15.0, 23.0, 'AAP', 'AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022'),
    (38, 45, 'LOW', 60, 17.0, 24.0, 'AAP', 'AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022'),
    (38, 45, 'LOW', 72, 18.0, 25.0, 'AAP', 'AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022'),
    (38, 45, 'LOW', 84, 19.0, 26.0, 'AAP', 'AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022'),
    (38, 45, 'LOW', 96, 20.0, 27.0, 'AAP', 'AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022'),
    (38, 45, 'LOW', 120, 21.0, 28.0, 'AAP', 'AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022');

-- Medium Risk (35-37 weeks OR risk factors)
INSERT INTO neonatal_bilirubin_thresholds
(gestational_age_weeks_min, gestational_age_weeks_max, risk_category, hour_of_life, photo_threshold, exchange_threshold, authority, authority_reference)
VALUES
    (35, 37, 'MEDIUM', 12, 5.0, NULL, 'AAP', 'AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022'),
    (35, 37, 'MEDIUM', 24, 10.0, 18.0, 'AAP', 'AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022'),
    (35, 37, 'MEDIUM', 36, 12.0, 19.0, 'AAP', 'AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022'),
    (35, 37, 'MEDIUM', 48, 13.0, 20.0, 'AAP', 'AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022'),
    (35, 37, 'MEDIUM', 60, 14.5, 21.0, 'AAP', 'AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022'),
    (35, 37, 'MEDIUM', 72, 16.0, 22.0, 'AAP', 'AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022'),
    (35, 37, 'MEDIUM', 84, 17.0, 23.0, 'AAP', 'AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022'),
    (35, 37, 'MEDIUM', 96, 18.0, 24.0, 'AAP', 'AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022'),
    (35, 37, 'MEDIUM', 120, 19.0, 25.0, 'AAP', 'AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022');

-- High Risk (<35 weeks)
INSERT INTO neonatal_bilirubin_thresholds
(gestational_age_weeks_min, gestational_age_weeks_max, risk_category, hour_of_life, photo_threshold, exchange_threshold, authority, authority_reference)
VALUES
    (28, 34, 'HIGH', 12, 4.0, NULL, 'AAP', 'AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022'),
    (28, 34, 'HIGH', 24, 8.0, 15.0, 'AAP', 'AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022'),
    (28, 34, 'HIGH', 36, 10.0, 16.0, 'AAP', 'AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022'),
    (28, 34, 'HIGH', 48, 11.0, 17.0, 'AAP', 'AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022'),
    (28, 34, 'HIGH', 60, 12.5, 18.0, 'AAP', 'AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022'),
    (28, 34, 'HIGH', 72, 14.0, 19.0, 'AAP', 'AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022'),
    (28, 34, 'HIGH', 84, 14.5, 20.0, 'AAP', 'AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022'),
    (28, 34, 'HIGH', 96, 15.0, 21.0, 'AAP', 'AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022'),
    (28, 34, 'HIGH', 120, 16.0, 22.0, 'AAP', 'AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022');

-- =============================================================================
-- 11. HELPER FUNCTIONS
-- =============================================================================

-- Get bilirubin threshold with interpolation
CREATE OR REPLACE FUNCTION get_bilirubin_threshold(
    p_ga_weeks INTEGER,
    p_risk_category VARCHAR(20),
    p_hours_of_life INTEGER
)
RETURNS TABLE (
    photo_threshold DECIMAL(5,2),
    exchange_threshold DECIMAL(5,2),
    interpolated BOOLEAN
) AS $$
DECLARE
    v_lower RECORD;
    v_upper RECORD;
    v_interpolation_factor DECIMAL;
BEGIN
    -- Try exact match first
    SELECT nbt.photo_threshold, nbt.exchange_threshold
    INTO v_lower
    FROM neonatal_bilirubin_thresholds nbt
    WHERE p_ga_weeks BETWEEN nbt.gestational_age_weeks_min AND nbt.gestational_age_weeks_max
      AND nbt.risk_category = p_risk_category
      AND nbt.hour_of_life = p_hours_of_life;

    IF FOUND THEN
        RETURN QUERY SELECT v_lower.photo_threshold, v_lower.exchange_threshold, FALSE;
        RETURN;
    END IF;

    -- Find bracketing thresholds for interpolation
    SELECT * INTO v_lower
    FROM neonatal_bilirubin_thresholds nbt
    WHERE p_ga_weeks BETWEEN nbt.gestational_age_weeks_min AND nbt.gestational_age_weeks_max
      AND nbt.risk_category = p_risk_category
      AND nbt.hour_of_life < p_hours_of_life
    ORDER BY nbt.hour_of_life DESC
    LIMIT 1;

    SELECT * INTO v_upper
    FROM neonatal_bilirubin_thresholds nbt
    WHERE p_ga_weeks BETWEEN nbt.gestational_age_weeks_min AND nbt.gestational_age_weeks_max
      AND nbt.risk_category = p_risk_category
      AND nbt.hour_of_life > p_hours_of_life
    ORDER BY nbt.hour_of_life ASC
    LIMIT 1;

    -- If no bracketing values, return NULL
    IF v_lower IS NULL OR v_upper IS NULL THEN
        RETURN QUERY SELECT NULL::DECIMAL(5,2), NULL::DECIMAL(5,2), FALSE;
        RETURN;
    END IF;

    -- Linear interpolation
    v_interpolation_factor := (p_hours_of_life - v_lower.hour_of_life)::DECIMAL /
                              (v_upper.hour_of_life - v_lower.hour_of_life)::DECIMAL;

    RETURN QUERY SELECT
        (v_lower.photo_threshold + v_interpolation_factor * (v_upper.photo_threshold - v_lower.photo_threshold))::DECIMAL(5,2),
        (v_lower.exchange_threshold + v_interpolation_factor * (v_upper.exchange_threshold - v_lower.exchange_threshold))::DECIMAL(5,2),
        TRUE;
END;
$$ LANGUAGE plpgsql STABLE;

-- =============================================================================
-- 12. COMMENTS
-- =============================================================================

COMMENT ON TABLE lab_tests IS 'Centralized LOINC-based laboratory test definitions with units and metadata';
COMMENT ON TABLE conditional_reference_ranges IS 'Context-aware reference ranges with patient condition matching (pregnancy, CKD, age, etc.)';
COMMENT ON TABLE neonatal_bilirubin_thresholds IS 'AAP 2022 Bhutani nomogram phototherapy and exchange transfusion thresholds';

COMMENT ON COLUMN conditional_reference_ranges.specificity_score IS 'Higher score = more specific condition = wins selection when multiple ranges match';
COMMENT ON COLUMN conditional_reference_ranges.trimester IS 'Pregnancy trimester: 1 (weeks 1-13), 2 (weeks 14-27), 3 (weeks 28-40)';
COMMENT ON COLUMN conditional_reference_ranges.ckd_stage IS 'CKD stage 1-5 per KDIGO classification based on eGFR';
COMMENT ON COLUMN conditional_reference_ranges.is_on_dialysis IS 'TRUE for patients on hemodialysis or peritoneal dialysis';

COMMENT ON FUNCTION get_lab_test_id(VARCHAR) IS 'Helper function to retrieve lab_test UUID by LOINC code';
COMMENT ON FUNCTION get_bilirubin_threshold(INTEGER, VARCHAR, INTEGER) IS
'Returns phototherapy/exchange thresholds for neonatal bilirubin with linear interpolation between hour points.
Parameters: gestational age (weeks), risk category (LOW/MEDIUM/HIGH), hours of life';

COMMIT;

-- =============================================================================
-- MIGRATION METADATA
-- =============================================================================
-- Version: 003
-- Date: 2026-01-26
-- Author: Claude Code
-- Purpose: Phase 3b.6 - Conditional Reference Ranges for Context-Aware Lab Interpretation
-- Phase: Phase 3b.6 - KB-16 Lab Reference Ranges Ingestion
-- Dependencies: 001_initial_schema.sql, 002_clinical_decision_limits.sql
-- Authority Sources: CLSI C28-A3c, ACOG, ATA 2017, KDIGO 2024, AAP 2022
-- =============================================================================
