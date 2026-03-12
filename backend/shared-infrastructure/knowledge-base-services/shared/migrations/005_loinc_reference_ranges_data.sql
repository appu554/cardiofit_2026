-- =============================================================================
-- MIGRATION 005 DATA: LOINC Reference Ranges (Auto-generated)
-- Generated: 203 entries, 105 unique LOINC codes
-- Source: Standard clinical reference ranges (Tietz, Mayo, UpToDate, Guidelines)
-- =============================================================================

BEGIN;

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2951-2', 'Sodium', 'Sodium reference range', 'mmol/L', 136, 145, 120, 160, 'adult', 'all', 'electrolyte', 'Low sodium may indicate SIADH, diuretic use, or heart failure. High sodium indicates dehydration or diabetes insipidus.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2951-2', 'Sodium', 'Sodium reference range', 'mmol/L', 136, 145, 125, 155, 'pediatric', 'all', 'electrolyte', 'Low sodium may indicate SIADH, diuretic use, or heart failure. High sodium indicates dehydration or diabetes insipidus.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2951-2', 'Sodium', 'Sodium reference range', 'mmol/L', 133, 146, 120, 160, 'neonate', 'all', 'electrolyte', 'Low sodium may indicate SIADH, diuretic use, or heart failure. High sodium indicates dehydration or diabetes insipidus.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2951-2', 'Sodium', 'Sodium reference range', 'mmol/L', 136, 145, 125, 155, 'geriatric', 'all', 'electrolyte', 'Low sodium may indicate SIADH, diuretic use, or heart failure. High sodium indicates dehydration or diabetes insipidus.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2823-3', 'Potassium', 'Potassium reference range', 'mmol/L', 3.5, 5.0, 2.5, 6.5, 'adult', 'all', 'electrolyte', 'Monitor for cardiac arrhythmias at extremes. Critical for digoxin and K-sparing diuretic interactions.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2823-3', 'Potassium', 'Potassium reference range', 'mmol/L', 3.4, 4.7, 2.5, 6.5, 'pediatric', 'all', 'electrolyte', 'Monitor for cardiac arrhythmias at extremes. Critical for digoxin and K-sparing diuretic interactions.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2823-3', 'Potassium', 'Potassium reference range', 'mmol/L', 3.7, 5.9, 2.5, 7.0, 'neonate', 'all', 'electrolyte', 'Monitor for cardiac arrhythmias at extremes. Critical for digoxin and K-sparing diuretic interactions.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2823-3', 'Potassium', 'Potassium reference range', 'mmol/L', 3.5, 5.3, 2.8, 6.2, 'geriatric', 'all', 'electrolyte', 'Monitor for cardiac arrhythmias at extremes. Critical for digoxin and K-sparing diuretic interactions.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2075-0', 'Chloride', 'Chloride reference range', 'mmol/L', 98, 106, 80, 120, 'adult', 'all', 'electrolyte', 'Interpret with sodium for acid-base status. Metabolic acidosis often shows elevated chloride.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2075-0', 'Chloride', 'Chloride reference range', 'mmol/L', 98, 106, 85, 115, 'pediatric', 'all', 'electrolyte', 'Interpret with sodium for acid-base status. Metabolic acidosis often shows elevated chloride.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2075-0', 'Chloride', 'Chloride reference range', 'mmol/L', 96, 106, 85, 115, 'neonate', 'all', 'electrolyte', 'Interpret with sodium for acid-base status. Metabolic acidosis often shows elevated chloride.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('1963-8', 'Bicarbonate', 'Bicarbonate reference range', 'mmol/L', 22, 29, 10, 40, 'adult', 'all', 'electrolyte', 'Low indicates metabolic acidosis. High indicates metabolic alkalosis or compensation.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('1963-8', 'Bicarbonate', 'Bicarbonate reference range', 'mmol/L', 20, 28, 12, 35, 'pediatric', 'all', 'electrolyte', 'Low indicates metabolic acidosis. High indicates metabolic alkalosis or compensation.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('1963-8', 'Bicarbonate', 'Bicarbonate reference range', 'mmol/L', 17, 24, 10, 30, 'neonate', 'all', 'electrolyte', 'Low indicates metabolic acidosis. High indicates metabolic alkalosis or compensation.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('17861-6', 'Calcium', 'Calcium reference range', 'mg/dL', 8.6, 10.2, 6.0, 14.0, 'adult', 'all', 'electrolyte', 'Correct for albumin: Corrected Ca = measured Ca + 0.8*(4.0 - albumin).')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('17861-6', 'Calcium', 'Calcium reference range', 'mg/dL', 8.8, 10.8, 6.5, 13.0, 'pediatric', 'all', 'electrolyte', 'Correct for albumin: Corrected Ca = measured Ca + 0.8*(4.0 - albumin).')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('17861-6', 'Calcium', 'Calcium reference range', 'mg/dL', 7.6, 10.4, 6.0, 13.0, 'neonate', 'all', 'electrolyte', 'Correct for albumin: Corrected Ca = measured Ca + 0.8*(4.0 - albumin).')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2000-8', 'Calcium ionized', 'Calcium ionized reference range', 'mmol/L', 1.12, 1.32, 0.8, 1.6, 'adult', 'all', 'electrolyte', 'True measure of metabolically active calcium. Critical for cardiac and neuromuscular function.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2777-1', 'Phosphorus', 'Phosphorus reference range', 'mg/dL', 2.5, 4.5, 1.0, 9.0, 'adult', 'all', 'electrolyte', 'Inverse relationship with calcium. Monitor in CKD and refeeding syndrome.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2777-1', 'Phosphorus', 'Phosphorus reference range', 'mg/dL', 4.0, 7.0, 2.0, 10.0, 'pediatric', 'all', 'electrolyte', 'Inverse relationship with calcium. Monitor in CKD and refeeding syndrome.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2777-1', 'Phosphorus', 'Phosphorus reference range', 'mg/dL', 4.8, 8.2, 2.5, 10.0, 'neonate', 'all', 'electrolyte', 'Inverse relationship with calcium. Monitor in CKD and refeeding syndrome.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('19123-9', 'Magnesium', 'Magnesium reference range', 'mg/dL', 1.7, 2.2, 1.0, 4.0, 'adult', 'all', 'electrolyte', 'Low Mg potentiates digoxin toxicity and causes refractory hypokalemia.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2160-0', 'Creatinine', 'Creatinine reference range', 'mg/dL', 0.7, 1.3, 0.4, 10.0, 'adult', 'all', 'renal', 'Baseline for eGFR calculation. 50% rise in 48h suggests AKI per KDIGO criteria.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2160-0', 'Creatinine', 'Creatinine reference range', 'mg/dL', 0.7, 1.3, 0.4, 10.0, 'adult', 'male', 'renal', 'Baseline for eGFR calculation. 50% rise in 48h suggests AKI per KDIGO criteria.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2160-0', 'Creatinine', 'Creatinine reference range', 'mg/dL', 0.6, 1.1, 0.4, 10.0, 'adult', 'female', 'renal', 'Baseline for eGFR calculation. 50% rise in 48h suggests AKI per KDIGO criteria.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2160-0', 'Creatinine', 'Creatinine reference range', 'mg/dL', 0.3, 0.7, 0.2, 5.0, 'pediatric', 'all', 'renal', 'Baseline for eGFR calculation. 50% rise in 48h suggests AKI per KDIGO criteria.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2160-0', 'Creatinine', 'Creatinine reference range', 'mg/dL', 0.3, 1.0, 0.2, 5.0, 'neonate', 'all', 'renal', 'Baseline for eGFR calculation. 50% rise in 48h suggests AKI per KDIGO criteria.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2160-0', 'Creatinine', 'Creatinine reference range', 'mg/dL', 0.7, 1.5, 0.4, 10.0, 'geriatric', 'all', 'renal', 'Baseline for eGFR calculation. 50% rise in 48h suggests AKI per KDIGO criteria.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('3094-0', 'BUN', 'BUN reference range', 'mg/dL', 7, 20, 2, 100, 'adult', 'all', 'renal', 'BUN:Cr ratio >20 suggests prerenal azotemia. Affected by protein intake and catabolic states.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('3094-0', 'BUN', 'BUN reference range', 'mg/dL', 5, 18, 2, 80, 'pediatric', 'all', 'renal', 'BUN:Cr ratio >20 suggests prerenal azotemia. Affected by protein intake and catabolic states.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('3094-0', 'BUN', 'BUN reference range', 'mg/dL', 3, 12, 2, 50, 'neonate', 'all', 'renal', 'BUN:Cr ratio >20 suggests prerenal azotemia. Affected by protein intake and catabolic states.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('3094-0', 'BUN', 'BUN reference range', 'mg/dL', 8, 23, 3, 100, 'geriatric', 'all', 'renal', 'BUN:Cr ratio >20 suggests prerenal azotemia. Affected by protein intake and catabolic states.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('33914-3', 'eGFR', 'eGFR reference range', 'mL/min/1.73m2', 90, 999, 15, NULL, 'adult', 'all', 'renal', 'CKD staging: >90=G1, 60-89=G2, 45-59=G3a, 30-44=G3b, 15-29=G4, <15=G5. Drug dosing adjustments required <60.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('48642-3', 'eGFR MDRD', 'eGFR MDRD reference range', 'mL/min/1.73m2', 90, 999, 15, NULL, 'adult', 'all', 'renal', 'Legacy formula. CKD-EPI preferred for most populations.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('62238-1', 'eGFR CKD-EPI 2021', 'eGFR CKD-EPI 2021 reference range', 'mL/min/1.73m2', 90, 999, 15, NULL, 'adult', 'all', 'renal', 'Race-free CKD-EPI 2021 equation. Preferred formula per KDIGO 2024.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('1742-6', 'ALT', 'ALT reference range', 'U/L', 7, 56, NULL, 1000, 'adult', 'all', 'hepatic', 'Elevation >3x ULN may indicate drug-induced hepatotoxicity. Monitor with statins, acetaminophen.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('1742-6', 'ALT', 'ALT reference range', 'U/L', 7, 55, NULL, 1000, 'adult', 'male', 'hepatic', 'Elevation >3x ULN may indicate drug-induced hepatotoxicity. Monitor with statins, acetaminophen.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('1742-6', 'ALT', 'ALT reference range', 'U/L', 7, 45, NULL, 1000, 'adult', 'female', 'hepatic', 'Elevation >3x ULN may indicate drug-induced hepatotoxicity. Monitor with statins, acetaminophen.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('1742-6', 'ALT', 'ALT reference range', 'U/L', 10, 35, NULL, 500, 'pediatric', 'all', 'hepatic', 'Elevation >3x ULN may indicate drug-induced hepatotoxicity. Monitor with statins, acetaminophen.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('1920-8', 'AST', 'AST reference range', 'U/L', 10, 40, NULL, 1000, 'adult', 'all', 'hepatic', 'Non-specific. Elevated with liver, cardiac, or muscle injury. AST:ALT >2 suggests alcoholic liver disease.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('1920-8', 'AST', 'AST reference range', 'U/L', 10, 40, NULL, 1000, 'adult', 'male', 'hepatic', 'Non-specific. Elevated with liver, cardiac, or muscle injury. AST:ALT >2 suggests alcoholic liver disease.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('1920-8', 'AST', 'AST reference range', 'U/L', 10, 35, NULL, 1000, 'adult', 'female', 'hepatic', 'Non-specific. Elevated with liver, cardiac, or muscle injury. AST:ALT >2 suggests alcoholic liver disease.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('1920-8', 'AST', 'AST reference range', 'U/L', 15, 60, NULL, 500, 'pediatric', 'all', 'hepatic', 'Non-specific. Elevated with liver, cardiac, or muscle injury. AST:ALT >2 suggests alcoholic liver disease.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('6768-6', 'Alkaline Phosphatase', 'Alkaline Phosphatase reference range', 'U/L', 44, 147, NULL, 1000, 'adult', 'all', 'hepatic', 'Elevated in cholestatic liver disease, bone disorders, and growth (pediatric).')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('6768-6', 'Alkaline Phosphatase', 'Alkaline Phosphatase reference range', 'U/L', 100, 390, NULL, 1500, 'pediatric', 'all', 'hepatic', 'Elevated in cholestatic liver disease, bone disorders, and growth (pediatric).')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('1975-2', 'Bilirubin Total', 'Bilirubin Total reference range', 'mg/dL', 0.1, 1.2, NULL, 15.0, 'adult', 'all', 'hepatic', 'Jaundice typically visible >2.5 mg/dL. Direct bilirubin helps differentiate hepatocellular vs cholestatic.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('1975-2', 'Bilirubin Total', 'Bilirubin Total reference range', 'mg/dL', 0.0, 12.0, NULL, 20.0, 'neonate', 'all', 'hepatic', 'Jaundice typically visible >2.5 mg/dL. Direct bilirubin helps differentiate hepatocellular vs cholestatic.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('1968-7', 'Bilirubin Direct', 'Bilirubin Direct reference range', 'mg/dL', 0.0, 0.3, NULL, 10.0, 'adult', 'all', 'hepatic', 'Elevated direct bilirubin indicates hepatocellular injury or biliary obstruction.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('1751-7', 'Albumin', 'Albumin reference range', 'g/dL', 3.5, 5.0, 1.5, NULL, 'adult', 'all', 'hepatic', 'Low albumin indicates liver synthetic dysfunction, malnutrition, or nephrotic syndrome.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('1751-7', 'Albumin', 'Albumin reference range', 'g/dL', 3.2, 4.8, 1.5, NULL, 'geriatric', 'all', 'hepatic', 'Low albumin indicates liver synthetic dysfunction, malnutrition, or nephrotic syndrome.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2324-2', 'GGT', 'GGT reference range', 'U/L', 0, 65, NULL, 1000, 'adult', 'all', 'hepatic', 'Most sensitive marker for biliary disease. Elevated with alcohol use.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2324-2', 'GGT', 'GGT reference range', 'U/L', 8, 61, NULL, 1000, 'adult', 'male', 'hepatic', 'Most sensitive marker for biliary disease. Elevated with alcohol use.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2324-2', 'GGT', 'GGT reference range', 'U/L', 5, 36, NULL, 1000, 'adult', 'female', 'hepatic', 'Most sensitive marker for biliary disease. Elevated with alcohol use.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('6301-6', 'INR', 'INR reference range', '', 0.9, 1.1, NULL, 5.0, 'adult', 'all', 'coagulation', 'Therapeutic range for warfarin typically 2.0-3.0 (2.5-3.5 for mechanical valves). >5.0 indicates significant bleeding risk.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('34714-6', 'INR', 'INR reference range', '', 0.9, 1.1, NULL, 5.0, 'adult', 'all', 'coagulation', 'Critical for warfarin and DOAC bridging decisions.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('5902-2', 'Prothrombin Time', 'Prothrombin Time reference range', 'seconds', 11.0, 13.5, NULL, 30.0, 'adult', 'all', 'coagulation', 'Extrinsic pathway. Prolonged with warfarin, vitamin K deficiency, liver disease.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('3173-2', 'PTT', 'PTT reference range', 'seconds', 25, 35, NULL, 100, 'adult', 'all', 'coagulation', 'Intrinsic pathway. Prolonged with heparin therapy, factor deficiencies.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('3255-7', 'Fibrinogen', 'Fibrinogen reference range', 'mg/dL', 200, 400, 100, NULL, 'adult', 'all', 'coagulation', 'Low in DIC, liver disease. Elevated as acute phase reactant.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('48065-7', 'D-dimer', 'D-dimer reference range', 'ng/mL FEU', 0, 500, NULL, NULL, 'adult', 'all', 'coagulation', 'Elevated in VTE, DIC, sepsis, malignancy, pregnancy. High NPV for PE/DVT.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('718-7', 'Hemoglobin', 'Hemoglobin reference range', 'g/dL', 12.0, 16.0, 7.0, 20.0, 'adult', 'all', 'hematology', '<7 g/dL may require transfusion. Assess for anemia etiology: iron, B12, folate, chronic disease.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('718-7', 'Hemoglobin', 'Hemoglobin reference range', 'g/dL', 13.5, 17.5, 7.0, 20.0, 'adult', 'male', 'hematology', '<7 g/dL may require transfusion. Assess for anemia etiology: iron, B12, folate, chronic disease.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('718-7', 'Hemoglobin', 'Hemoglobin reference range', 'g/dL', 12.0, 16.0, 7.0, 20.0, 'adult', 'female', 'hematology', '<7 g/dL may require transfusion. Assess for anemia etiology: iron, B12, folate, chronic disease.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('718-7', 'Hemoglobin', 'Hemoglobin reference range', 'g/dL', 11.0, 14.0, 7.0, 18.0, 'pediatric', 'all', 'hematology', '<7 g/dL may require transfusion. Assess for anemia etiology: iron, B12, folate, chronic disease.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('718-7', 'Hemoglobin', 'Hemoglobin reference range', 'g/dL', 14.0, 24.0, 10.0, 26.0, 'neonate', 'all', 'hematology', '<7 g/dL may require transfusion. Assess for anemia etiology: iron, B12, folate, chronic disease.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('4544-3', 'Hematocrit', 'Hematocrit reference range', '%', 36, 48, 20, 60, 'adult', 'all', 'hematology', 'Low in anemia, fluid overload. High in polycythemia, dehydration.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('4544-3', 'Hematocrit', 'Hematocrit reference range', '%', 40, 52, 20, 60, 'adult', 'male', 'hematology', 'Low in anemia, fluid overload. High in polycythemia, dehydration.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('4544-3', 'Hematocrit', 'Hematocrit reference range', '%', 36, 46, 20, 60, 'adult', 'female', 'hematology', 'Low in anemia, fluid overload. High in polycythemia, dehydration.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('6690-2', 'WBC', 'WBC reference range', 'x10^3/uL', 4.5, 11.0, 2.0, 30.0, 'adult', 'all', 'hematology', 'Leukocytosis: infection, stress, steroids, leukemia. Leukopenia: bone marrow suppression, overwhelming sepsis.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('6690-2', 'WBC', 'WBC reference range', 'x10^3/uL', 5.0, 15.0, 2.5, 30.0, 'pediatric', 'all', 'hematology', 'Leukocytosis: infection, stress, steroids, leukemia. Leukopenia: bone marrow suppression, overwhelming sepsis.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('6690-2', 'WBC', 'WBC reference range', 'x10^3/uL', 9.0, 30.0, 5.0, 40.0, 'neonate', 'all', 'hematology', 'Leukocytosis: infection, stress, steroids, leukemia. Leukopenia: bone marrow suppression, overwhelming sepsis.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('777-3', 'Platelets', 'Platelets reference range', 'x10^3/uL', 150, 400, 50, 1000, 'adult', 'all', 'hematology', '<50 significant bleeding risk. <20 spontaneous bleeding risk. >1000 thrombotic risk (essential thrombocythemia).')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('789-8', 'RBC', 'RBC reference range', 'x10^6/uL', 4.0, 5.5, 2.5, 7.0, 'adult', 'all', 'hematology', 'Interpret with hemoglobin and MCV for anemia classification.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('789-8', 'RBC', 'RBC reference range', 'x10^6/uL', 4.5, 5.9, 2.5, 7.0, 'adult', 'male', 'hematology', 'Interpret with hemoglobin and MCV for anemia classification.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('789-8', 'RBC', 'RBC reference range', 'x10^6/uL', 4.0, 5.2, 2.5, 7.0, 'adult', 'female', 'hematology', 'Interpret with hemoglobin and MCV for anemia classification.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('787-2', 'MCV', 'MCV reference range', 'fL', 80, 100, 50, 130, 'adult', 'all', 'hematology', 'Microcytic (<80): iron deficiency, thalassemia. Macrocytic (>100): B12/folate deficiency, liver disease.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('786-4', 'MCH', 'MCH reference range', 'pg', 27, 33, NULL, NULL, 'adult', 'all', 'hematology', 'Mean cell hemoglobin. Correlates with MCV for anemia classification.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('785-6', 'MCHC', 'MCHC reference range', 'g/dL', 32, 36, NULL, NULL, 'adult', 'all', 'hematology', 'Low in hypochromic anemias. Elevated may indicate spherocytosis or cold agglutinins.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('788-0', 'RDW', 'RDW reference range', '%', 11.5, 14.5, NULL, NULL, 'adult', 'all', 'hematology', 'Elevated RDW (anisocytosis) suggests iron deficiency, mixed deficiencies, or myelodysplasia.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('10839-9', 'Troponin I', 'Troponin I reference range', 'ng/mL', NULL, 0.04, NULL, 0.5, 'adult', 'all', 'cardiac', 'Elevation indicates myocardial injury. Serial measurements for trend. >99th percentile with rise/fall pattern = MI.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('6598-7', 'Troponin T', 'Troponin T reference range', 'ng/mL', NULL, 0.01, NULL, 0.1, 'adult', 'all', 'cardiac', 'High-sensitivity assay. Interpret with clinical context and serial values.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('33762-6', 'NT-proBNP', 'NT-proBNP reference range', 'pg/mL', NULL, 125, NULL, 5000, 'adult', 'all', 'cardiac', 'Heart failure marker. Age-dependent cutoffs. Rule-out HF if <300 pg/mL.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('33762-6', 'NT-proBNP', 'NT-proBNP reference range', 'pg/mL', NULL, 450, NULL, 5000, 'geriatric', 'all', 'cardiac', 'Heart failure marker. Age-dependent cutoffs. Rule-out HF if <300 pg/mL.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('30341-2', 'BNP', 'BNP reference range', 'pg/mL', NULL, 100, NULL, 2000, 'adult', 'all', 'cardiac', 'Heart failure marker. <100 pg/mL makes HF unlikely. Elevated in renal dysfunction.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('8634-8', 'QTc', 'QTc reference range', 'ms', NULL, 450, NULL, 500, 'adult', 'male', 'cardiac', '>500ms significant arrhythmia risk. Review QT-prolonging medications (antipsychotics, antibiotics, antiarrhythmics).')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('8634-8', 'QTc', 'QTc reference range', 'ms', NULL, 460, NULL, 500, 'adult', 'female', 'cardiac', '>500ms significant arrhythmia risk. Review QT-prolonging medications (antipsychotics, antibiotics, antiarrhythmics).')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2345-7', 'Glucose', 'Glucose reference range', 'mg/dL', 70, 100, 40, 500, 'adult', 'all', 'metabolic', 'Fasting <100 normal. 100-125 prediabetes. >=126 diabetes (confirm with repeat). Critical values require immediate intervention.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2345-7', 'Glucose', 'Glucose reference range', 'mg/dL', 60, 100, 40, 400, 'pediatric', 'all', 'metabolic', 'Fasting <100 normal. 100-125 prediabetes. >=126 diabetes (confirm with repeat). Critical values require immediate intervention.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2345-7', 'Glucose', 'Glucose reference range', 'mg/dL', 40, 60, 30, 250, 'neonate', 'all', 'metabolic', 'Fasting <100 normal. 100-125 prediabetes. >=126 diabetes (confirm with repeat). Critical values require immediate intervention.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('4548-4', 'HbA1c', 'HbA1c reference range', '%', 4.0, 5.6, 3.0, 15.0, 'adult', 'all', 'metabolic', '<5.7% normal. 5.7-6.4% prediabetes. >=6.5% diabetes. Target <7% for most diabetics (individualize).')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('3016-3', 'TSH', 'TSH reference range', 'mIU/L', 0.4, 4.0, 0.01, 100, 'adult', 'all', 'thyroid', 'Low TSH: hyperthyroidism or suppressive therapy. High TSH: hypothyroidism.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('3016-3', 'TSH', 'TSH reference range', 'mIU/L', 0.4, 7.0, 0.01, 100, 'geriatric', 'all', 'thyroid', 'Low TSH: hyperthyroidism or suppressive therapy. High TSH: hypothyroidism.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('3026-2', 'Free T4', 'Free T4 reference range', 'ng/dL', 0.8, 1.8, 0.3, 5.0, 'adult', 'all', 'thyroid', 'Interpret with TSH. Elevated in hyperthyroidism. Low in hypothyroidism.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('3051-0', 'Free T3', 'Free T3 reference range', 'pg/mL', 2.3, 4.2, 1.0, 10.0, 'adult', 'all', 'thyroid', 'Active thyroid hormone. Useful in T3 toxicosis and euthyroid sick syndrome.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2093-3', 'Cholesterol Total', 'Cholesterol Total reference range', 'mg/dL', 0, 200, NULL, 400, 'adult', 'all', 'lipid', 'Desirable <200. Borderline 200-239. High >=240.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2571-8', 'Triglycerides', 'Triglycerides reference range', 'mg/dL', 0, 150, NULL, 1000, 'adult', 'all', 'lipid', 'Normal <150. High 200-499. Very high >=500 (pancreatitis risk).')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2085-9', 'HDL Cholesterol', 'HDL Cholesterol reference range', 'mg/dL', 40, 999, 20, NULL, 'adult', 'all', 'lipid', 'Protective factor. Low HDL is CV risk factor.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2085-9', 'HDL Cholesterol', 'HDL Cholesterol reference range', 'mg/dL', 40, 999, 20, NULL, 'adult', 'male', 'lipid', 'Protective factor. Low HDL is CV risk factor.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2085-9', 'HDL Cholesterol', 'HDL Cholesterol reference range', 'mg/dL', 50, 999, 20, NULL, 'adult', 'female', 'lipid', 'Protective factor. Low HDL is CV risk factor.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2089-1', 'LDL Cholesterol', 'LDL Cholesterol reference range', 'mg/dL', 0, 100, NULL, 300, 'adult', 'all', 'lipid', 'Primary target for therapy. <70 for very high risk. <100 for high risk. <130 for moderate risk.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('13457-7', 'LDL Cholesterol Direct', 'LDL Cholesterol Direct reference range', 'mg/dL', 0, 100, NULL, 300, 'adult', 'all', 'lipid', 'Direct measurement preferred when TG >400 mg/dL.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('10535-3', 'Digoxin', 'Digoxin reference range', 'ng/mL', 0.8, 2.0, NULL, 2.5, 'adult', 'all', 'therapeutic_drug', 'Therapeutic 0.8-2.0 ng/mL (0.5-1.0 for HF). Toxicity >2.0. Check K+, Mg2+, Ca2+, renal function.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('14334-7', 'Lithium', 'Lithium reference range', 'mmol/L', 0.6, 1.2, NULL, 1.5, 'adult', 'all', 'therapeutic_drug', 'Therapeutic 0.6-1.2 mmol/L. >1.5 toxicity risk. Monitor renal function and hydration.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('3968-5', 'Phenytoin', 'Phenytoin reference range', 'mcg/mL', 10, 20, NULL, 25, 'adult', 'all', 'therapeutic_drug', 'Therapeutic 10-20 mcg/mL. Adjust for albumin: Corrected = measured/(0.2*albumin + 0.1).')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('4049-3', 'Theophylline', 'Theophylline reference range', 'mcg/mL', 10, 20, NULL, 25, 'adult', 'all', 'therapeutic_drug', 'Therapeutic 10-20 mcg/mL. Narrow therapeutic index. Monitor for toxicity signs.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('4090-7', 'Vancomycin', 'Vancomycin reference range', 'mcg/mL', 10, 20, NULL, 30, 'adult', 'all', 'therapeutic_drug', 'Trough 10-20 mcg/mL (higher for serious infections). AUC/MIC monitoring preferred.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('20578-1', 'Vancomycin Peak', 'Vancomycin Peak reference range', 'mcg/mL', 25, 40, NULL, 50, 'adult', 'all', 'therapeutic_drug', 'Peak levels less commonly monitored. AUC-guided dosing preferred.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('35669-1', 'Tacrolimus', 'Tacrolimus reference range', 'ng/mL', 5, 15, NULL, 25, 'adult', 'all', 'therapeutic_drug', 'Transplant: 10-15 ng/mL early, 5-10 maintenance. Monitor renal function for nephrotoxicity.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('4092-3', 'Valproic Acid', 'Valproic Acid reference range', 'mcg/mL', 50, 100, NULL, 150, 'adult', 'all', 'therapeutic_drug', 'Therapeutic 50-100 mcg/mL. Monitor LFTs, ammonia, and platelet count.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('14979-9', 'Carbamazepine', 'Carbamazepine reference range', 'mcg/mL', 4, 12, NULL, 15, 'adult', 'all', 'therapeutic_drug', 'Therapeutic 4-12 mcg/mL. Auto-induction may require dose adjustment.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('1988-5', 'CRP', 'CRP reference range', 'mg/L', 0, 10, NULL, NULL, 'adult', 'all', 'inflammatory', 'Non-specific inflammation marker. <10 mg/L generally considered low risk.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('30522-7', 'hs-CRP', 'hs-CRP reference range', 'mg/L', 0, 3.0, NULL, NULL, 'adult', 'all', 'inflammatory', 'CV risk: <1.0 low risk, 1-3 average, >3.0 high risk. Not useful during acute illness.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('4537-7', 'ESR', 'ESR reference range', 'mm/hr', 0, 20, NULL, 100, 'adult', 'all', 'inflammatory', 'Non-specific inflammation. Rule of thumb: age/2 (male) or (age+10)/2 (female) for upper limit.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('4537-7', 'ESR', 'ESR reference range', 'mm/hr', 0, 15, NULL, 100, 'adult', 'male', 'inflammatory', 'Non-specific inflammation. Rule of thumb: age/2 (male) or (age+10)/2 (female) for upper limit.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('4537-7', 'ESR', 'ESR reference range', 'mm/hr', 0, 20, NULL, 100, 'adult', 'female', 'inflammatory', 'Non-specific inflammation. Rule of thumb: age/2 (male) or (age+10)/2 (female) for upper limit.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('4537-7', 'ESR', 'ESR reference range', 'mm/hr', 0, 30, NULL, 100, 'geriatric', 'all', 'inflammatory', 'Non-specific inflammation. Rule of thumb: age/2 (male) or (age+10)/2 (female) for upper limit.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2524-7', 'Lactate', 'Lactate reference range', 'mmol/L', 0.5, 2.0, NULL, 4.0, 'adult', 'all', 'inflammatory', '>2 mmol/L may indicate tissue hypoperfusion. >4 mmol/L associated with poor outcomes in sepsis.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('33959-8', 'Procalcitonin', 'Procalcitonin reference range', 'ng/mL', 0, 0.5, NULL, 10.0, 'adult', 'all', 'inflammatory', 'Bacterial infection marker. <0.25 unlikely bacterial. >0.5 likely bacterial. Guide antibiotic duration.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2498-4', 'Iron', 'Iron reference range', 'mcg/dL', 60, 170, 20, 300, 'adult', 'all', 'iron_studies', 'Low in iron deficiency. Elevated in hemochromatosis, transfusion overload.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2498-4', 'Iron', 'Iron reference range', 'mcg/dL', 65, 175, 20, 300, 'adult', 'male', 'iron_studies', 'Low in iron deficiency. Elevated in hemochromatosis, transfusion overload.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2498-4', 'Iron', 'Iron reference range', 'mcg/dL', 50, 170, 20, 300, 'adult', 'female', 'iron_studies', 'Low in iron deficiency. Elevated in hemochromatosis, transfusion overload.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2502-3', 'TIBC', 'TIBC reference range', 'mcg/dL', 250, 370, NULL, NULL, 'adult', 'all', 'iron_studies', 'High TIBC in iron deficiency. Low in anemia of chronic disease.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2505-6', 'Transferrin Saturation', 'Transferrin Saturation reference range', '%', 20, 50, 10, 80, 'adult', 'all', 'iron_studies', '<20% suggests iron deficiency. >45% in hemochromatosis workup.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2276-4', 'Ferritin', 'Ferritin reference range', 'ng/mL', 12, 300, 5, 1000, 'adult', 'all', 'iron_studies', '<12 diagnostic of iron deficiency. Elevated as acute phase reactant and in iron overload.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2276-4', 'Ferritin', 'Ferritin reference range', 'ng/mL', 24, 336, 10, 1000, 'adult', 'male', 'iron_studies', '<12 diagnostic of iron deficiency. Elevated as acute phase reactant and in iron overload.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2276-4', 'Ferritin', 'Ferritin reference range', 'ng/mL', 12, 150, 5, 1000, 'adult', 'female', 'iron_studies', '<12 diagnostic of iron deficiency. Elevated as acute phase reactant and in iron overload.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2132-9', 'Vitamin B12', 'Vitamin B12 reference range', 'pg/mL', 200, 900, 100, NULL, 'adult', 'all', 'vitamin', '<200 deficiency. 200-300 borderline. Check methylmalonic acid if borderline.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2284-8', 'Folate', 'Folate reference range', 'ng/mL', 2.7, 17.0, 1.0, NULL, 'adult', 'all', 'vitamin', '<2.7 deficiency. Causes macrocytic anemia. Important in pregnancy.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('1989-3', 'Vitamin D, 25-OH', 'Vitamin D, 25-OH reference range', 'ng/mL', 30, 100, 10, 150, 'adult', 'all', 'vitamin', '<20 deficiency. 20-29 insufficiency. >30 sufficient. >100 potential toxicity.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2965-2', 'Urine Specific Gravity', 'Urine Specific Gravity reference range', '', 1.005, 1.03, 1.001, 1.04, 'adult', 'all', 'urinalysis', 'Reflects concentrating ability. Low in diabetes insipidus. High in dehydration.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2756-5', 'Urine pH', 'Urine pH reference range', '', 4.5, 8.0, NULL, NULL, 'adult', 'all', 'urinalysis', 'Acidic urine in metabolic acidosis, protein-rich diet. Alkaline in UTI, vegetarian diet.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2888-6', 'Urine Protein', 'Urine Protein reference range', 'mg/dL', 0, 14, NULL, NULL, 'adult', 'all', 'urinalysis', 'Proteinuria indicates glomerular or tubular disease. Quantify with 24h or spot protein/creatinine ratio.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2339-0', 'Urine Glucose', 'Urine Glucose reference range', 'mg/dL', 0, 0, NULL, NULL, 'adult', 'all', 'urinalysis', 'Glucosuria when serum glucose exceeds renal threshold (~180 mg/dL). Present in SGLT2 inhibitor use.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2744-1', 'pH Arterial', 'pH Arterial reference range', '', 7.35, 7.45, 7.2, 7.6, 'adult', 'all', 'blood_gas', '<7.35 acidemia. >7.45 alkalemia. Interpret with pCO2 and HCO3 for primary disorder.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2019-8', 'pCO2 Arterial', 'pCO2 Arterial reference range', 'mmHg', 35, 45, 20, 70, 'adult', 'all', 'blood_gas', 'Respiratory component. Low in respiratory alkalosis. High in respiratory acidosis.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2703-7', 'pO2 Arterial', 'pO2 Arterial reference range', 'mmHg', 80, 100, 50, NULL, 'adult', 'all', 'blood_gas', '<60 mmHg indicates respiratory failure. Interpret with FiO2 (P/F ratio).')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2713-6', 'O2 Saturation Arterial', 'O2 Saturation Arterial reference range', '%', 95, 100, 88, NULL, 'adult', 'all', 'blood_gas', '<88% requires supplemental oxygen in most patients. Target may be lower in COPD.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2039-6', 'CEA', 'CEA reference range', 'ng/mL', 0, 3.0, NULL, NULL, 'adult', 'all', 'tumor_marker', 'Colorectal cancer surveillance. May be elevated in smokers. Not for screening.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('2857-1', 'PSA', 'PSA reference range', 'ng/mL', 0, 4.0, NULL, NULL, 'adult', 'male', 'tumor_marker', 'Age-adjusted cutoffs available. Used for prostate cancer screening and monitoring.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('19177-5', 'AFP', 'AFP reference range', 'ng/mL', 0, 10, NULL, NULL, 'adult', 'all', 'tumor_marker', 'Hepatocellular carcinoma surveillance in cirrhosis. Also elevated in germ cell tumors.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('10334-1', 'CA 125', 'CA 125 reference range', 'U/mL', 0, 35, NULL, NULL, 'adult', 'female', 'tumor_marker', 'Ovarian cancer monitoring. May be elevated in endometriosis, pregnancy, menstruation.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('17842-6', 'CA 19-9', 'CA 19-9 reference range', 'U/mL', 0, 37, NULL, NULL, 'adult', 'all', 'tumor_marker', 'Pancreatic cancer monitoring. May be elevated in biliary obstruction.')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('39791-9', 'Potassium', 'Potassium [Moles/volume] in Venous blood', 'mmol/L', 3.5, 5.0, 2.5, 6.5, 'adult', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('39791-9', 'Potassium', 'Potassium [Moles/volume] in Venous blood', 'mmol/L', 3.4, 4.7, 2.5, 6.5, 'pediatric', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('39791-9', 'Potassium', 'Potassium [Moles/volume] in Venous blood', 'mmol/L', 3.7, 5.9, 2.5, 7.0, 'neonate', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('39791-9', 'Potassium', 'Potassium [Moles/volume] in Venous blood', 'mmol/L', 3.5, 5.3, 2.8, 6.2, 'geriatric', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('6298-4', 'Potassium', 'Potassium [Moles/volume] in Blood', 'mmol/L', 3.5, 5.0, 2.5, 6.5, 'adult', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('6298-4', 'Potassium', 'Potassium [Moles/volume] in Blood', 'mmol/L', 3.4, 4.7, 2.5, 6.5, 'pediatric', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('6298-4', 'Potassium', 'Potassium [Moles/volume] in Blood', 'mmol/L', 3.7, 5.9, 2.5, 7.0, 'neonate', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('6298-4', 'Potassium', 'Potassium [Moles/volume] in Blood', 'mmol/L', 3.5, 5.3, 2.8, 6.2, 'geriatric', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('77140-2', 'Sodium', 'Sodium [Moles/volume] in Serum, Plasma or Blood', 'mmol/L', 136, 145, 120, 160, 'adult', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('77140-2', 'Sodium', 'Sodium [Moles/volume] in Serum, Plasma or Blood', 'mmol/L', 136, 145, 125, 155, 'pediatric', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('77140-2', 'Sodium', 'Sodium [Moles/volume] in Serum, Plasma or Blood', 'mmol/L', 133, 146, 120, 160, 'neonate', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('77140-2', 'Sodium', 'Sodium [Moles/volume] in Serum, Plasma or Blood', 'mmol/L', 136, 145, 125, 155, 'geriatric', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('38483-4', 'Creatinine', 'Creatinine [Mass/volume] in Blood', 'mmol/L', 0.7, 1.3, 0.4, 10.0, 'adult', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('38483-4', 'Creatinine', 'Creatinine [Mass/volume] in Blood', 'mmol/L', 0.7, 1.3, 0.4, 10.0, 'adult', 'male', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('38483-4', 'Creatinine', 'Creatinine [Mass/volume] in Blood', 'mmol/L', 0.6, 1.1, 0.4, 10.0, 'adult', 'female', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('38483-4', 'Creatinine', 'Creatinine [Mass/volume] in Blood', 'mmol/L', 0.3, 0.7, 0.2, 5.0, 'pediatric', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('38483-4', 'Creatinine', 'Creatinine [Mass/volume] in Blood', 'mmol/L', 0.3, 1.0, 0.2, 5.0, 'neonate', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('38483-4', 'Creatinine', 'Creatinine [Mass/volume] in Blood', 'mmol/L', 0.7, 1.5, 0.4, 10.0, 'geriatric', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('77139-4', 'Creatinine', 'Creatinine [Mass/volume] in Serum, Plasma or Blood', 'mmol/L', 0.7, 1.3, 0.4, 10.0, 'adult', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('77139-4', 'Creatinine', 'Creatinine [Mass/volume] in Serum, Plasma or Blood', 'mmol/L', 0.7, 1.3, 0.4, 10.0, 'adult', 'male', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('77139-4', 'Creatinine', 'Creatinine [Mass/volume] in Serum, Plasma or Blood', 'mmol/L', 0.6, 1.1, 0.4, 10.0, 'adult', 'female', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('77139-4', 'Creatinine', 'Creatinine [Mass/volume] in Serum, Plasma or Blood', 'mmol/L', 0.3, 0.7, 0.2, 5.0, 'pediatric', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('77139-4', 'Creatinine', 'Creatinine [Mass/volume] in Serum, Plasma or Blood', 'mmol/L', 0.3, 1.0, 0.2, 5.0, 'neonate', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('77139-4', 'Creatinine', 'Creatinine [Mass/volume] in Serum, Plasma or Blood', 'mmol/L', 0.7, 1.5, 0.4, 10.0, 'geriatric', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('88293-6', 'eGFR', 'Glomerular filtration rate/1.73 sq M.predicted by CKD-EPI 2021', 'mmol/L', 90, 999, 15, NULL, 'adult', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('98979-8', 'eGFR', 'Glomerular filtration rate/1.73 sq M.predicted [Volume Rate/Area] in Serum, Plasma or Blood by CKD-EPI 2021', 'mmol/L', 90, 999, 15, NULL, 'adult', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('1743-4', 'ALT', 'Alanine aminotransferase [Enzymatic activity/volume] in Serum or Plasma with P-5-P', 'mmol/L', 7, 56, NULL, 1000, 'adult', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('1743-4', 'ALT', 'Alanine aminotransferase [Enzymatic activity/volume] in Serum or Plasma with P-5-P', 'mmol/L', 7, 55, NULL, 1000, 'adult', 'male', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('1743-4', 'ALT', 'Alanine aminotransferase [Enzymatic activity/volume] in Serum or Plasma with P-5-P', 'mmol/L', 7, 45, NULL, 1000, 'adult', 'female', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('1743-4', 'ALT', 'Alanine aminotransferase [Enzymatic activity/volume] in Serum or Plasma with P-5-P', 'mmol/L', 10, 35, NULL, 500, 'pediatric', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('1921-6', 'AST', 'Aspartate aminotransferase [Enzymatic activity/volume] in Serum or Plasma with P-5-P', 'mmol/L', 10, 40, NULL, 1000, 'adult', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('1921-6', 'AST', 'Aspartate aminotransferase [Enzymatic activity/volume] in Serum or Plasma with P-5-P', 'mmol/L', 10, 40, NULL, 1000, 'adult', 'male', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('1921-6', 'AST', 'Aspartate aminotransferase [Enzymatic activity/volume] in Serum or Plasma with P-5-P', 'mmol/L', 10, 35, NULL, 1000, 'adult', 'female', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('1921-6', 'AST', 'Aspartate aminotransferase [Enzymatic activity/volume] in Serum or Plasma with P-5-P', 'mmol/L', 15, 60, NULL, 500, 'pediatric', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('46418-0', 'INR', 'INR in Platelet poor plasma by Coagulation assay', 'mmol/L', 0.9, 1.1, NULL, 5.0, 'adult', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('49563-0', 'Troponin I', 'Troponin I.cardiac [Mass/volume] in Serum or Plasma by High sensitivity method', 'mmol/L', NULL, 0.034, NULL, 0.5, 'adult', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('89579-7', 'Troponin I', 'Troponin I.cardiac [Mass/volume] in Serum or Plasma by High sensitivity immunoassay', 'mmol/L', NULL, 0.034, NULL, 0.5, 'adult', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('67151-1', 'Troponin T', 'Troponin T.cardiac [Mass/volume] in Serum or Plasma by High sensitivity method', 'mmol/L', NULL, 0.014, NULL, 0.1, 'adult', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('41653-7', 'Glucose', 'Glucose [Mass/volume] in Capillary blood by Glucometer', 'mmol/L', 70, 100, 40, 500, 'adult', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('41653-7', 'Glucose', 'Glucose [Mass/volume] in Capillary blood by Glucometer', 'mmol/L', 60, 100, 40, 400, 'pediatric', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('41653-7', 'Glucose', 'Glucose [Mass/volume] in Capillary blood by Glucometer', 'mmol/L', 40, 60, 30, 250, 'neonate', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('74774-1', 'Glucose', 'Glucose [Mass/volume] in Serum, Plasma or Blood', 'mmol/L', 70, 100, 40, 500, 'adult', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('74774-1', 'Glucose', 'Glucose [Mass/volume] in Serum, Plasma or Blood', 'mmol/L', 60, 100, 40, 400, 'pediatric', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('74774-1', 'Glucose', 'Glucose [Mass/volume] in Serum, Plasma or Blood', 'mmol/L', 40, 60, 30, 250, 'neonate', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('17856-6', 'HbA1c', 'Hemoglobin A1c/Hemoglobin.total in Blood by HPLC', 'mmol/L', 4.0, 5.6, 3.0, 15.0, 'adult', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('20570-8', 'Hematocrit', 'Hematocrit [Volume Fraction] of Blood by Calculation', 'mmol/L', 36, 48, 20, 60, 'adult', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('20570-8', 'Hematocrit', 'Hematocrit [Volume Fraction] of Blood by Calculation', 'mmol/L', 40, 52, 20, 60, 'adult', 'male', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('20570-8', 'Hematocrit', 'Hematocrit [Volume Fraction] of Blood by Calculation', 'mmol/L', 36, 46, 20, 60, 'adult', 'female', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('26515-7', 'Platelets', 'Platelets [#/volume] in Blood', 'mmol/L', 150, 400, 50, 1000, 'adult', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('26464-8', 'WBC', 'Leukocytes [#/volume] in Blood', 'mmol/L', 4.5, 11.0, 2.0, 30.0, 'adult', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('26464-8', 'WBC', 'Leukocytes [#/volume] in Blood', 'mmol/L', 5.0, 15.0, 2.5, 30.0, 'pediatric', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('26464-8', 'WBC', 'Leukocytes [#/volume] in Blood', 'mmol/L', 9.0, 30.0, 5.0, 40.0, 'neonate', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('30313-1', 'Hemoglobin', 'Hemoglobin [Mass/volume] in Arterial blood', 'mmol/L', 12.0, 16.0, 7.0, 20.0, 'adult', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('30313-1', 'Hemoglobin', 'Hemoglobin [Mass/volume] in Arterial blood', 'mmol/L', 13.5, 17.5, 7.0, 20.0, 'adult', 'male', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('30313-1', 'Hemoglobin', 'Hemoglobin [Mass/volume] in Arterial blood', 'mmol/L', 12.0, 16.0, 7.0, 20.0, 'adult', 'female', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('30313-1', 'Hemoglobin', 'Hemoglobin [Mass/volume] in Arterial blood', 'mmol/L', 11.0, 14.0, 7.0, 18.0, 'pediatric', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('30313-1', 'Hemoglobin', 'Hemoglobin [Mass/volume] in Arterial blood', 'mmol/L', 14.0, 24.0, 10.0, 26.0, 'neonate', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('59260-0', 'Hemoglobin', 'Hemoglobin [Moles/volume] in Blood', 'mmol/L', 12.0, 16.0, 7.0, 20.0, 'adult', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('59260-0', 'Hemoglobin', 'Hemoglobin [Moles/volume] in Blood', 'mmol/L', 13.5, 17.5, 7.0, 20.0, 'adult', 'male', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('59260-0', 'Hemoglobin', 'Hemoglobin [Moles/volume] in Blood', 'mmol/L', 12.0, 16.0, 7.0, 20.0, 'adult', 'female', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('59260-0', 'Hemoglobin', 'Hemoglobin [Moles/volume] in Blood', 'mmol/L', 11.0, 14.0, 7.0, 18.0, 'pediatric', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

INSERT INTO loinc_reference_ranges (loinc_code, component, long_name, unit, low_normal, high_normal, critical_low, critical_high, age_group, sex, clinical_category, interpretation_guidance)
VALUES ('59260-0', 'Hemoglobin', 'Hemoglobin [Moles/volume] in Blood', 'mmol/L', 14.0, 24.0, 10.0, 26.0, 'neonate', 'all', 'chemistry', '')
ON CONFLICT (loinc_code, age_group, sex) DO UPDATE SET
    low_normal = EXCLUDED.low_normal,
    high_normal = EXCLUDED.high_normal,
    critical_low = EXCLUDED.critical_low,
    critical_high = EXCLUDED.critical_high,
    updated_at = NOW();

COMMIT;
