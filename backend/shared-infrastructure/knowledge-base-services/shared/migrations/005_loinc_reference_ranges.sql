-- =============================================================================
-- MIGRATION 005: LOINC Reference Ranges for Context Router
-- Purpose: Load LAB_REFERENCE facts into canonical_facts for DDI context evaluation
-- Reference: Context Router Execution Contract (ONC → OHDSI → LOINC pipeline)
-- =============================================================================
--
-- Golden Rule: "Class Expansion NEVER checks LOINC. Context Router ALWAYS does."
--
-- This migration populates the shared database with LOINC reference ranges that
-- the Context Router uses to evaluate DDI context thresholds at runtime.
-- =============================================================================

BEGIN;

-- =============================================================================
-- LOINC REFERENCE RANGES TABLE
-- Dedicated table for high-performance LOINC lookups by Context Router
-- =============================================================================

CREATE TABLE IF NOT EXISTS loinc_reference_ranges (
    -- Identity
    range_id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- LOINC Identification
    loinc_code          VARCHAR(20) NOT NULL,
    component           VARCHAR(200) NOT NULL,
    long_name           VARCHAR(500) NOT NULL,
    short_name          VARCHAR(100),

    -- Classification
    loinc_class         VARCHAR(50),
    clinical_category   VARCHAR(50) NOT NULL,
    property            VARCHAR(50),
    time_aspect         VARCHAR(20),
    system              VARCHAR(50),
    scale_type          VARCHAR(20),
    method_type         VARCHAR(50),

    -- Reference Ranges
    unit                VARCHAR(50) NOT NULL,
    low_normal          DECIMAL(15,5),
    high_normal         DECIMAL(15,5),

    -- Critical/Panic Values
    critical_low        DECIMAL(15,5),
    critical_high       DECIMAL(15,5),

    -- Population Specificity
    age_group           VARCHAR(20) NOT NULL DEFAULT 'adult',
    sex                 VARCHAR(10) NOT NULL DEFAULT 'all',

    -- Delta Check Rules
    delta_check_percent DECIMAL(5,2),
    delta_check_hours   INTEGER,

    -- Interpretation
    interpretation_guidance TEXT,

    -- Metadata
    source              VARCHAR(100) NOT NULL DEFAULT 'loinc_labs_expanded',
    version             VARCHAR(20) NOT NULL DEFAULT '1.0.0',
    deprecated          BOOLEAN NOT NULL DEFAULT FALSE,

    -- Audit
    created_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- Unique constraint: One range per LOINC code + age + sex combination
    CONSTRAINT uq_loinc_range UNIQUE (loinc_code, age_group, sex)
);

-- =============================================================================
-- INDEXES FOR CONTEXT ROUTER PERFORMANCE
-- =============================================================================

-- Primary lookup: LOINC code (most common query from Context Router)
CREATE INDEX IF NOT EXISTS idx_loinc_ref_code ON loinc_reference_ranges(loinc_code);

-- Population-specific lookup: LOINC + age + sex
CREATE INDEX IF NOT EXISTS idx_loinc_ref_population ON loinc_reference_ranges(loinc_code, age_group, sex);

-- Clinical category queries
CREATE INDEX IF NOT EXISTS idx_loinc_ref_category ON loinc_reference_ranges(clinical_category);

-- Critical values (for alert generation)
CREATE INDEX IF NOT EXISTS idx_loinc_ref_critical ON loinc_reference_ranges(loinc_code)
    WHERE critical_low IS NOT NULL OR critical_high IS NOT NULL;

-- Non-deprecated active ranges
CREATE INDEX IF NOT EXISTS idx_loinc_ref_active ON loinc_reference_ranges(loinc_code, age_group, sex)
    WHERE deprecated = FALSE;

-- =============================================================================
-- LOAD LOINC REFERENCE RANGES DATA
-- Clinical lab reference ranges with age/sex-specific values
-- Source: loinc_labs_expanded.csv (352 records across 5 age groups)
-- =============================================================================

-- Electrolytes - Sodium (5 age groups)
INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, short_name, loinc_class, clinical_category, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, delta_check_percent, delta_check_hours, interpretation_guidance)
VALUES
    ('2951-2', 'Sodium', 'Sodium [Moles/volume] in Serum or Plasma', 'Sodium SerPl-mCnc', 'CHEM', 'electrolyte', 'mmol/L', 136.0, 145.0, 120.0, 160.0, 'adult', 'all', 10, 24, 'Low sodium may indicate SIADH or diuretic use. High sodium indicates dehydration or diabetes insipidus.'),
    ('2951-2', 'Sodium', 'Sodium [Moles/volume] in Serum or Plasma', 'Sodium SerPl-mCnc', 'CHEM', 'electrolyte', 'mmol/L', 129.2, 152.25, 114.0, 168.0, 'pediatric', 'all', 10, 24, 'Pediatric-specific: Low sodium may indicate SIADH or diuretic use.'),
    ('2951-2', 'Sodium', 'Sodium [Moles/volume] in Serum or Plasma', 'Sodium SerPl-mCnc', 'CHEM', 'electrolyte', 'mmol/L', 122.4, 159.5, 108.0, 176.0, 'neonate', 'all', 10, 24, 'Neonate-specific: Monitor closely for fluid balance.'),
    ('2951-2', 'Sodium', 'Sodium [Moles/volume] in Serum or Plasma', 'Sodium SerPl-mCnc', 'CHEM', 'electrolyte', 'mmol/L', 129.2, 152.25, 114.0, 168.0, 'geriatric', 'all', 10, 24, 'Geriatric-specific: More susceptible to hyponatremia.'),
    ('2951-2', 'Sodium', 'Sodium [Moles/volume] in Serum or Plasma', 'Sodium SerPl-mCnc', 'CHEM', 'electrolyte', 'mmol/L', 125.12, 156.6, 110.4, 172.8, 'infant', 'all', 10, 24, 'Infant-specific: Monitor for dehydration.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

-- Electrolytes - Potassium (5 age groups) - Critical for DDI context (digoxin, K-sparing diuretics)
INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, short_name, loinc_class, clinical_category, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, delta_check_percent, delta_check_hours, interpretation_guidance)
VALUES
    ('2823-3', 'Potassium', 'Potassium [Moles/volume] in Serum or Plasma', 'Potassium SerPl-mCnc', 'CHEM', 'electrolyte', 'mmol/L', 3.5, 5.0, 2.5, 6.5, 'adult', 'all', 20, 24, 'Monitor for cardiac arrhythmias at extremes. Consider hemolysis artifact if elevated.'),
    ('2823-3', 'Potassium', 'Potassium [Moles/volume] in Serum or Plasma', 'Potassium SerPl-mCnc', 'CHEM', 'electrolyte', 'mmol/L', 3.325, 5.25, 2.375, 6.825, 'pediatric', 'all', 20, 24, 'Pediatric-specific: Monitor for cardiac arrhythmias.'),
    ('2823-3', 'Potassium', 'Potassium [Moles/volume] in Serum or Plasma', 'Potassium SerPl-mCnc', 'CHEM', 'electrolyte', 'mmol/L', 3.15, 5.5, 2.25, 7.15, 'neonate', 'all', 20, 24, 'Neonate-specific: Higher normal range expected.'),
    ('2823-3', 'Potassium', 'Potassium [Moles/volume] in Serum or Plasma', 'Potassium SerPl-mCnc', 'CHEM', 'electrolyte', 'mmol/L', 3.325, 5.25, 2.375, 6.825, 'geriatric', 'all', 20, 24, 'Geriatric-specific: Higher risk with renal impairment and ACE/ARB/K-sparing diuretics.'),
    ('2823-3', 'Potassium', 'Potassium [Moles/volume] in Serum or Plasma', 'Potassium SerPl-mCnc', 'CHEM', 'electrolyte', 'mmol/L', 3.22, 5.4, 2.3, 7.02, 'infant', 'all', 20, 24, 'Infant-specific: Monitor for cardiac arrhythmias.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

-- Renal Function - Creatinine (adult with sex-specific ranges)
INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, short_name, loinc_class, clinical_category, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, delta_check_percent, delta_check_hours, interpretation_guidance)
VALUES
    ('2160-0', 'Creatinine', 'Creatinine [Mass/volume] in Serum or Plasma', 'Creat SerPl-mCnc', 'CHEM', 'renal', 'mg/dL', 0.7, 1.3, 0.4, 10.0, 'adult', 'all', 50, 48, 'Baseline for eGFR calculation. 50% rise in 48h suggests AKI per KDIGO.'),
    ('2160-0', 'Creatinine', 'Creatinine [Mass/volume] in Serum or Plasma', 'Creat SerPl-mCnc', 'CHEM', 'renal', 'mg/dL', 0.7, 1.3, 0.4, 10.0, 'adult', 'male', 50, 48, 'Male adult: Baseline for eGFR calculation.'),
    ('2160-0', 'Creatinine', 'Creatinine [Mass/volume] in Serum or Plasma', 'Creat SerPl-mCnc', 'CHEM', 'renal', 'mg/dL', 0.6, 1.1, 0.4, 10.0, 'adult', 'female', 50, 48, 'Female adult: Lower range due to lower muscle mass.'),
    ('2160-0', 'Creatinine', 'Creatinine [Mass/volume] in Serum or Plasma', 'Creat SerPl-mCnc', 'CHEM', 'renal', 'mg/dL', 0.5, 1.0, 0.3, 10.0, 'pediatric', 'all', 50, 48, 'Pediatric-specific: Age-dependent ranges.'),
    ('2160-0', 'Creatinine', 'Creatinine [Mass/volume] in Serum or Plasma', 'Creat SerPl-mCnc', 'CHEM', 'renal', 'mg/dL', 0.3, 1.0, 0.2, 10.0, 'neonate', 'all', 50, 48, 'Neonate-specific: Reflects maternal creatinine initially.'),
    ('2160-0', 'Creatinine', 'Creatinine [Mass/volume] in Serum or Plasma', 'Creat SerPl-mCnc', 'CHEM', 'renal', 'mg/dL', 0.7, 1.5, 0.4, 10.0, 'geriatric', 'all', 50, 48, 'Geriatric-specific: May underestimate renal impairment due to reduced muscle mass.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    updated_at = NOW();

-- Renal Function - eGFR (Critical for drug dosing adjustments)
INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, short_name, loinc_class, clinical_category, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, delta_check_percent, delta_check_hours, interpretation_guidance)
VALUES
    ('33914-3', 'eGFR', 'Glomerular filtration rate/1.73 sq M.predicted by CKD-EPI', 'eGFR CKD-EPI', 'CHEM', 'renal', 'mL/min/1.73m2', 90.0, 999.0, 15.0, 999.0, 'adult', 'all', NULL, NULL, 'Stage CKD: >90=G1 60-89=G2 45-59=G3a 30-44=G3b 15-29=G4 <15=G5. Drug dosing adjustments required <60.'),
    ('33914-3', 'eGFR', 'Glomerular filtration rate/1.73 sq M.predicted by CKD-EPI', 'eGFR CKD-EPI', 'CHEM', 'renal', 'mL/min/1.73m2', 90.0, 999.0, 15.0, 999.0, 'adult', 'male', NULL, NULL, 'Male adult CKD staging. Drug dosing adjustments required <60.'),
    ('33914-3', 'eGFR', 'Glomerular filtration rate/1.73 sq M.predicted by CKD-EPI', 'eGFR CKD-EPI', 'CHEM', 'renal', 'mL/min/1.73m2', 90.0, 999.0, 15.0, 999.0, 'adult', 'female', NULL, NULL, 'Female adult CKD staging. Drug dosing adjustments required <60.'),
    ('33914-3', 'eGFR', 'Glomerular filtration rate/1.73 sq M.predicted by CKD-EPI', 'eGFR CKD-EPI', 'CHEM', 'renal', 'mL/min/1.73m2', 90.0, 999.0, 15.0, 999.0, 'geriatric', 'all', NULL, NULL, 'Geriatric-specific: Expect age-related decline. Drug dosing adjustments required <60.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    interpretation_guidance = EXCLUDED.interpretation_guidance,
    updated_at = NOW();

-- Coagulation - INR (Critical for warfarin interactions)
INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, short_name, loinc_class, clinical_category, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, delta_check_percent, delta_check_hours, interpretation_guidance)
VALUES
    ('6301-6', 'INR', 'Coagulation tissue factor induced INR', 'INR', 'COAG', 'coagulation', '', 0.9, 1.1, NULL, 5.0, 'adult', 'all', NULL, NULL, 'Therapeutic range for warfarin typically 2.0-3.0. >5.0 indicates significant bleeding risk.'),
    ('34714-6', 'INR', 'INR in Platelet poor plasma by Coagulation assay', 'INR', 'COAG', 'coagulation', '', 0.9, 1.1, NULL, 5.0, 'adult', 'all', NULL, NULL, 'Therapeutic range for warfarin typically 2.0-3.0. >5.0 indicates significant bleeding risk.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

-- Liver Function - ALT (Critical for hepatotoxic drug monitoring)
INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, short_name, loinc_class, clinical_category, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, delta_check_percent, delta_check_hours, interpretation_guidance)
VALUES
    ('1742-6', 'ALT', 'Alanine aminotransferase [Enzymatic activity/volume] in Serum or Plasma', 'ALT SerPl-cCnc', 'CHEM', 'hepatic', 'U/L', 7, 56, NULL, 1000, 'adult', 'all', NULL, NULL, 'Elevation >3x ULN may indicate drug-induced hepatotoxicity. Monitor with hepatotoxic medications.'),
    ('1742-6', 'ALT', 'Alanine aminotransferase [Enzymatic activity/volume] in Serum or Plasma', 'ALT SerPl-cCnc', 'CHEM', 'hepatic', 'U/L', 7, 55, NULL, 1000, 'adult', 'male', NULL, NULL, 'Male adult: Elevation >3x ULN requires drug review.'),
    ('1742-6', 'ALT', 'Alanine aminotransferase [Enzymatic activity/volume] in Serum or Plasma', 'ALT SerPl-cCnc', 'CHEM', 'hepatic', 'U/L', 7, 45, NULL, 1000, 'adult', 'female', NULL, NULL, 'Female adult: Lower ULN. Elevation >3x ULN requires drug review.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    high_normal = EXCLUDED.high_normal,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

-- Liver Function - AST
INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, short_name, loinc_class, clinical_category, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, delta_check_percent, delta_check_hours, interpretation_guidance)
VALUES
    ('1920-8', 'AST', 'Aspartate aminotransferase [Enzymatic activity/volume] in Serum or Plasma', 'AST SerPl-cCnc', 'CHEM', 'hepatic', 'U/L', 10, 40, NULL, 1000, 'adult', 'all', NULL, NULL, 'Non-specific. Elevated with liver, cardiac, or muscle injury.'),
    ('1920-8', 'AST', 'Aspartate aminotransferase [Enzymatic activity/volume] in Serum or Plasma', 'AST SerPl-cCnc', 'CHEM', 'hepatic', 'U/L', 10, 40, NULL, 1000, 'adult', 'male', NULL, NULL, 'Male adult AST reference range.'),
    ('1920-8', 'AST', 'Aspartate aminotransferase [Enzymatic activity/volume] in Serum or Plasma', 'AST SerPl-cCnc', 'CHEM', 'hepatic', 'U/L', 10, 35, NULL, 1000, 'adult', 'female', NULL, NULL, 'Female adult: Slightly lower ULN.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    high_normal = EXCLUDED.high_normal,
    updated_at = NOW();

-- Cardiac - Digoxin Level (Critical for DDI context)
INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, short_name, loinc_class, clinical_category, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, delta_check_percent, delta_check_hours, interpretation_guidance)
VALUES
    ('10535-3', 'Digoxin', 'Digoxin [Mass/volume] in Serum or Plasma', 'Digoxin SerPl-mCnc', 'DRUG', 'therapeutic_drug', 'ng/mL', 0.8, 2.0, NULL, 2.5, 'adult', 'all', NULL, NULL, 'Therapeutic range 0.8-2.0 ng/mL. Toxicity risk increases >2.0. Check K+, Mg2+, Ca2+ with elevated levels.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

-- Cardiac - QTc Interval (Critical for QT-prolonging drug interactions)
INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, short_name, loinc_class, clinical_category, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, delta_check_percent, delta_check_hours, interpretation_guidance)
VALUES
    ('8634-8', 'QTc', 'QTc interval', 'QTc', 'CARDIAC', 'cardiac', 'ms', NULL, 450, NULL, 500, 'adult', 'male', NULL, NULL, 'Male: QTc >450ms prolonged, >500ms significant arrhythmia risk. Review QT-prolonging medications.'),
    ('8634-8', 'QTc', 'QTc interval', 'QTc', 'CARDIAC', 'cardiac', 'ms', NULL, 460, NULL, 500, 'adult', 'female', NULL, NULL, 'Female: QTc >460ms prolonged, >500ms significant arrhythmia risk. Review QT-prolonging medications.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    high_normal = EXCLUDED.high_normal,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

-- Metabolic - Glucose
INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, short_name, loinc_class, clinical_category, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, delta_check_percent, delta_check_hours, interpretation_guidance)
VALUES
    ('2345-7', 'Glucose', 'Glucose [Mass/volume] in Serum or Plasma', 'Glucose SerPl-mCnc', 'CHEM', 'metabolic', 'mg/dL', 70, 100, 40, 500, 'adult', 'all', 25, 4, 'Fasting <100 normal. 100-125 prediabetes. >=126 diabetes. Critical values require immediate intervention.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

-- Hematology - Platelets (Critical for bleeding risk assessment)
INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, short_name, loinc_class, clinical_category, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, delta_check_percent, delta_check_hours, interpretation_guidance)
VALUES
    ('777-3', 'Platelets', 'Platelets [#/volume] in Blood by Automated count', 'Platelets Bld Auto', 'HEM', 'hematology', 'x10^3/uL', 150, 400, 50, 1000, 'adult', 'all', 50, 24, '<50 significant bleeding risk. <20 spontaneous bleeding risk. >1000 thrombotic risk.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

-- Hematology - Hemoglobin
INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, short_name, loinc_class, clinical_category, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, delta_check_percent, delta_check_hours, interpretation_guidance)
VALUES
    ('718-7', 'Hemoglobin', 'Hemoglobin [Mass/volume] in Blood', 'Hgb Bld-mCnc', 'HEM', 'hematology', 'g/dL', 12.0, 16.0, 7.0, 20.0, 'adult', 'all', 20, 24, 'Assess for anemia or polycythemia. <7 may require transfusion.'),
    ('718-7', 'Hemoglobin', 'Hemoglobin [Mass/volume] in Blood', 'Hgb Bld-mCnc', 'HEM', 'hematology', 'g/dL', 13.5, 17.5, 7.0, 20.0, 'adult', 'male', 20, 24, 'Male adult: Higher normal range.'),
    ('718-7', 'Hemoglobin', 'Hemoglobin [Mass/volume] in Blood', 'Hgb Bld-mCnc', 'HEM', 'hematology', 'g/dL', 12.0, 16.0, 7.0, 20.0, 'adult', 'female', 20, 24, 'Female adult: Lower normal range.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    updated_at = NOW();

-- Drug Levels - Lithium (Critical for toxicity monitoring)
INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, short_name, loinc_class, clinical_category, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, delta_check_percent, delta_check_hours, interpretation_guidance)
VALUES
    ('14334-7', 'Lithium', 'Lithium [Moles/volume] in Serum or Plasma', 'Lithium SerPl-sCnc', 'DRUG', 'therapeutic_drug', 'mmol/L', 0.6, 1.2, NULL, 1.5, 'adult', 'all', NULL, NULL, 'Therapeutic 0.6-1.2 mmol/L. >1.5 toxicity risk. Check renal function and hydration status.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

-- Inflammatory Markers - Lactate (Sepsis marker)
INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, short_name, loinc_class, clinical_category, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, delta_check_percent, delta_check_hours, interpretation_guidance)
VALUES
    ('2524-7', 'Lactate', 'Lactate [Moles/volume] in Blood', 'Lactate Bld-sCnc', 'CHEM', 'inflammatory', 'mmol/L', 0.5, 2.0, NULL, 4.0, 'adult', 'all', NULL, NULL, '>2 mmol/L may indicate tissue hypoperfusion. >4 mmol/L associated with poor outcomes in sepsis.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

-- Cardiac Markers - Troponin I
INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, short_name, loinc_class, clinical_category, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, delta_check_percent, delta_check_hours, interpretation_guidance)
VALUES
    ('10839-9', 'Troponin I', 'Troponin I.cardiac [Mass/volume] in Serum or Plasma', 'TnI SerPl-mCnc', 'CARDIAC', 'cardiac', 'ng/mL', NULL, 0.04, NULL, 0.5, 'adult', 'all', NULL, NULL, 'Elevation indicates myocardial injury. Serial measurements for trend assessment.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    high_normal = EXCLUDED.high_normal,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

-- Thyroid - TSH
INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, short_name, loinc_class, clinical_category, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, delta_check_percent, delta_check_hours, interpretation_guidance)
VALUES
    ('3016-3', 'TSH', 'Thyrotropin [Units/volume] in Serum or Plasma', 'TSH SerPl-aCnc', 'CHEM', 'thyroid', 'mIU/L', 0.4, 4.0, 0.01, 100, 'adult', 'all', NULL, NULL, 'Low TSH may indicate hyperthyroidism (or suppressive therapy). High TSH indicates hypothyroidism.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

-- =============================================================================
-- VIEW FOR CONTEXT ROUTER LOOKUP
-- Simplified view for quick reference range retrieval
-- =============================================================================

CREATE OR REPLACE VIEW v_loinc_ranges_for_context AS
SELECT
    loinc_code,
    component AS loinc_name,
    unit,
    low_normal,
    high_normal,
    critical_low,
    critical_high,
    age_group,
    sex,
    clinical_category,
    interpretation_guidance
FROM loinc_reference_ranges
WHERE deprecated = FALSE
ORDER BY loinc_code, age_group, sex;

-- =============================================================================
-- FUNCTION: Get Reference Range for LOINC Code with Population Match
-- Used by Context Router for threshold evaluation
-- =============================================================================

CREATE OR REPLACE FUNCTION get_loinc_reference_range(
    p_loinc_code VARCHAR(20),
    p_age_group VARCHAR(20) DEFAULT 'adult',
    p_sex VARCHAR(10) DEFAULT 'all'
) RETURNS TABLE (
    loinc_code VARCHAR(20),
    loinc_name VARCHAR(200),
    unit VARCHAR(50),
    low_normal DECIMAL(15,5),
    high_normal DECIMAL(15,5),
    critical_low DECIMAL(15,5),
    critical_high DECIMAL(15,5),
    interpretation_guidance TEXT
) AS $$
BEGIN
    -- First try exact match (code + age + sex)
    RETURN QUERY
    SELECT
        r.loinc_code,
        r.component,
        r.unit,
        r.low_normal,
        r.high_normal,
        r.critical_low,
        r.critical_high,
        r.interpretation_guidance
    FROM loinc_reference_ranges r
    WHERE r.loinc_code = p_loinc_code
      AND r.age_group = p_age_group
      AND r.sex = p_sex
      AND r.deprecated = FALSE
    LIMIT 1;

    IF NOT FOUND THEN
        -- Try sex='all' fallback
        RETURN QUERY
        SELECT
            r.loinc_code,
            r.component,
            r.unit,
            r.low_normal,
            r.high_normal,
            r.critical_low,
            r.critical_high,
            r.interpretation_guidance
        FROM loinc_reference_ranges r
        WHERE r.loinc_code = p_loinc_code
          AND r.age_group = p_age_group
          AND r.sex = 'all'
          AND r.deprecated = FALSE
        LIMIT 1;
    END IF;

    IF NOT FOUND THEN
        -- Try adult/all fallback
        RETURN QUERY
        SELECT
            r.loinc_code,
            r.component,
            r.unit,
            r.low_normal,
            r.high_normal,
            r.critical_low,
            r.critical_high,
            r.interpretation_guidance
        FROM loinc_reference_ranges r
        WHERE r.loinc_code = p_loinc_code
          AND r.age_group = 'adult'
          AND r.sex = 'all'
          AND r.deprecated = FALSE
        LIMIT 1;
    END IF;
END;
$$ LANGUAGE plpgsql STABLE;

-- =============================================================================
-- COMMENTS
-- =============================================================================

COMMENT ON TABLE loinc_reference_ranges IS 'LOINC laboratory reference ranges for Context Router DDI evaluation. Source: loinc_labs_expanded.csv';
COMMENT ON FUNCTION get_loinc_reference_range IS 'Retrieves LOINC reference range with population-specific fallback for Context Router';
COMMENT ON VIEW v_loinc_ranges_for_context IS 'Simplified view of active LOINC reference ranges for Context Router';

-- =============================================================================
-- RECORD MIGRATION
-- =============================================================================

INSERT INTO schema_migrations (version, description)
VALUES ('005', 'LOINC reference ranges for Context Router')
ON CONFLICT (version) DO NOTHING;

COMMIT;

-- =============================================================================
-- STATISTICS
-- Report loaded LOINC codes
-- =============================================================================

SELECT
    clinical_category,
    COUNT(DISTINCT loinc_code) AS unique_loinc_codes,
    COUNT(*) AS total_ranges_with_population_variants
FROM loinc_reference_ranges
GROUP BY clinical_category
ORDER BY unique_loinc_codes DESC;
