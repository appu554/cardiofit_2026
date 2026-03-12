-- ============================================================================
-- KB-3 Guidelines Data Import Script
-- Generated: 2025-09-18T12:35:17.756781
-- Total Guidelines: 10
-- ============================================================================

-- Set schema
SET search_path TO guideline_evidence;

-- Begin transaction for atomic import
BEGIN;

-- Guideline 1/10: ADA-DM-2025-001
INSERT INTO guideline_evidence.guidelines (
    guideline_id,
    organization,
    region,
    condition_primary,
    icd10_codes,
    version,
    effective_date,
    status,
    approval_status,
    evidence_summary,
    quality_metrics,
    created_by,
    approved_by
) VALUES (
    'ADA-DM-2025-001',
    'ADA',
    'Global',
    'Type 2 Diabetes Mellitus - Glycemic Targets',
    '{"E11.9"}',
    '2025',
    '2025-01-01',
    'active',
    'approved',
    '{"recommendation": "For most nonpregnant adults with type 2 diabetes, a reasonable A1C goal is <7% (53 mmol/mol).", "evidence_grade": "A", "strength_of_recommendation": "Strong"}'::jsonb,
    '{"methodology_score": 9.5, "bias_risk": "low", "consistency": "high"}'::jsonb,
    'system_import',
    'clinical_team'
) ON CONFLICT (guideline_id) DO UPDATE SET
    organization = EXCLUDED.organization,
    region = EXCLUDED.region,
    condition_primary = EXCLUDED.condition_primary,
    icd10_codes = EXCLUDED.icd10_codes,
    version = EXCLUDED.version,
    effective_date = EXCLUDED.effective_date,
    status = EXCLUDED.status,
    approval_status = EXCLUDED.approval_status,
    evidence_summary = EXCLUDED.evidence_summary,
    quality_metrics = EXCLUDED.quality_metrics,
    updated_at = NOW();

-- Guideline 2/10: ADA-DM-2025-002
INSERT INTO guideline_evidence.guidelines (
    guideline_id,
    organization,
    region,
    condition_primary,
    icd10_codes,
    version,
    effective_date,
    status,
    approval_status,
    evidence_summary,
    quality_metrics,
    created_by,
    approved_by
) VALUES (
    'ADA-DM-2025-002',
    'ADA',
    'Global',
    'Type 2 Diabetes Mellitus - First-Line Therapy',
    '{"E11.9"}',
    '2025',
    '2025-01-01',
    'active',
    'approved',
    '{"recommendation": "Metformin is the preferred initial pharmacologic agent for the treatment of type 2 diabetes.", "evidence_grade": "A", "strength_of_recommendation": "Strong"}'::jsonb,
    '{"methodology_score": 9.8, "bias_risk": "low", "consistency": "high"}'::jsonb,
    'system_import',
    'clinical_team'
) ON CONFLICT (guideline_id) DO UPDATE SET
    organization = EXCLUDED.organization,
    region = EXCLUDED.region,
    condition_primary = EXCLUDED.condition_primary,
    icd10_codes = EXCLUDED.icd10_codes,
    version = EXCLUDED.version,
    effective_date = EXCLUDED.effective_date,
    status = EXCLUDED.status,
    approval_status = EXCLUDED.approval_status,
    evidence_summary = EXCLUDED.evidence_summary,
    quality_metrics = EXCLUDED.quality_metrics,
    updated_at = NOW();

-- Guideline 3/10: ADA-DM-2025-003
INSERT INTO guideline_evidence.guidelines (
    guideline_id,
    organization,
    region,
    condition_primary,
    icd10_codes,
    version,
    effective_date,
    status,
    approval_status,
    evidence_summary,
    quality_metrics,
    created_by,
    approved_by
) VALUES (
    'ADA-DM-2025-003',
    'ADA',
    'Global',
    'Type 2 Diabetes Mellitus - High-Risk ASCVD',
    '{"E11.9","I25.10"}',
    '2025',
    '2025-01-01',
    'active',
    'approved',
    '{"recommendation": "In patients with type 2 diabetes and established ASCVD, a GLP-1 receptor agonist or SGLT2 inhibitor with demonstrated cardiovascular benefit is recommended.", "evidence_grade": "A", "strength_of_recommendation": "Strong"}'::jsonb,
    '{"methodology_score": 9.7, "bias_risk": "low", "consistency": "high"}'::jsonb,
    'system_import',
    'clinical_team'
) ON CONFLICT (guideline_id) DO UPDATE SET
    organization = EXCLUDED.organization,
    region = EXCLUDED.region,
    condition_primary = EXCLUDED.condition_primary,
    icd10_codes = EXCLUDED.icd10_codes,
    version = EXCLUDED.version,
    effective_date = EXCLUDED.effective_date,
    status = EXCLUDED.status,
    approval_status = EXCLUDED.approval_status,
    evidence_summary = EXCLUDED.evidence_summary,
    quality_metrics = EXCLUDED.quality_metrics,
    updated_at = NOW();

-- Guideline 4/10: KDIGO-DM-CKD-2024-001
INSERT INTO guideline_evidence.guidelines (
    guideline_id,
    organization,
    region,
    condition_primary,
    icd10_codes,
    version,
    effective_date,
    status,
    approval_status,
    evidence_summary,
    quality_metrics,
    created_by,
    approved_by
) VALUES (
    'KDIGO-DM-CKD-2024-001',
    'KDIGO',
    'Global',
    'Type 2 Diabetes Mellitus with CKD',
    '{"E11.22","N18.3"}',
    '2024',
    '2024-10-01',
    'active',
    'approved',
    '{"recommendation": "For patients with T2DM and CKD (eGFR >20), treatment with an SGLT2 inhibitor is recommended to reduce risk of CKD progression and cardiovascular events.", "evidence_grade": "A", "strength_of_recommendation": "Strong"}'::jsonb,
    '{"methodology_score": 9.6, "bias_risk": "low", "consistency": "high"}'::jsonb,
    'system_import',
    'clinical_team'
) ON CONFLICT (guideline_id) DO UPDATE SET
    organization = EXCLUDED.organization,
    region = EXCLUDED.region,
    condition_primary = EXCLUDED.condition_primary,
    icd10_codes = EXCLUDED.icd10_codes,
    version = EXCLUDED.version,
    effective_date = EXCLUDED.effective_date,
    status = EXCLUDED.status,
    approval_status = EXCLUDED.approval_status,
    evidence_summary = EXCLUDED.evidence_summary,
    quality_metrics = EXCLUDED.quality_metrics,
    updated_at = NOW();

-- Guideline 5/10: ADA-DM-HTN-2025-001
INSERT INTO guideline_evidence.guidelines (
    guideline_id,
    organization,
    region,
    condition_primary,
    icd10_codes,
    version,
    effective_date,
    status,
    approval_status,
    evidence_summary,
    quality_metrics,
    created_by,
    approved_by
) VALUES (
    'ADA-DM-HTN-2025-001',
    'ADA',
    'Global',
    'Hypertension in Patients with Diabetes',
    '{"E11.9","I10"}',
    '2025',
    '2025-01-01',
    'active',
    'approved',
    '{"recommendation": "For individuals with diabetes and hypertension, a blood pressure target of <130/80 mmHg is recommended. An ACE inhibitor or ARB is recommended as first-line therapy if albuminuria is present.", "evidence_grade": "A", "strength_of_recommendation": "Strong"}'::jsonb,
    '{"methodology_score": 9.5, "bias_risk": "low", "consistency": "high"}'::jsonb,
    'system_import',
    'clinical_team'
) ON CONFLICT (guideline_id) DO UPDATE SET
    organization = EXCLUDED.organization,
    region = EXCLUDED.region,
    condition_primary = EXCLUDED.condition_primary,
    icd10_codes = EXCLUDED.icd10_codes,
    version = EXCLUDED.version,
    effective_date = EXCLUDED.effective_date,
    status = EXCLUDED.status,
    approval_status = EXCLUDED.approval_status,
    evidence_summary = EXCLUDED.evidence_summary,
    quality_metrics = EXCLUDED.quality_metrics,
    updated_at = NOW();

-- Guideline 6/10: ACC-AHA-HTN-2017-001
INSERT INTO guideline_evidence.guidelines (
    guideline_id,
    organization,
    region,
    condition_primary,
    icd10_codes,
    version,
    effective_date,
    status,
    approval_status,
    evidence_summary,
    quality_metrics,
    created_by,
    approved_by
) VALUES (
    'ACC-AHA-HTN-2017-001',
    'ACC/AHA',
    'US',
    'Hypertension - Stage 1 Definition',
    '{"I10"}',
    '2017',
    '2017-11-13',
    'active',
    'approved',
    '{"recommendation": "Stage 1 Hypertension is defined as a Systolic BP of 130-139 mmHg or a Diastolic BP of 80-89 mmHg.", "evidence_grade": "B-R", "strength_of_recommendation": "Moderate"}'::jsonb,
    '{"methodology_score": 9.2, "bias_risk": "low", "consistency": "high"}'::jsonb,
    'system_import',
    'clinical_team'
) ON CONFLICT (guideline_id) DO UPDATE SET
    organization = EXCLUDED.organization,
    region = EXCLUDED.region,
    condition_primary = EXCLUDED.condition_primary,
    icd10_codes = EXCLUDED.icd10_codes,
    version = EXCLUDED.version,
    effective_date = EXCLUDED.effective_date,
    status = EXCLUDED.status,
    approval_status = EXCLUDED.approval_status,
    evidence_summary = EXCLUDED.evidence_summary,
    quality_metrics = EXCLUDED.quality_metrics,
    updated_at = NOW();

-- Guideline 7/10: ACC-AHA-HTN-2017-002
INSERT INTO guideline_evidence.guidelines (
    guideline_id,
    organization,
    region,
    condition_primary,
    icd10_codes,
    version,
    effective_date,
    status,
    approval_status,
    evidence_summary,
    quality_metrics,
    created_by,
    approved_by
) VALUES (
    'ACC-AHA-HTN-2017-002',
    'ACC/AHA',
    'US',
    'Hypertension - Stage 1 Treatment Initiation',
    '{"I10"}',
    '2017',
    '2017-11-13',
    'active',
    'approved',
    '{"recommendation": "For Stage 1 Hypertension with an estimated 10-year ASCVD risk of \u226510%, initiate BP-lowering medication. First-line agents include thiazide diuretics, CCBs, and ACE inhibitors or ARBs.", "evidence_grade": "A", "strength_of_recommendation": "Strong"}'::jsonb,
    '{"methodology_score": 9.8, "bias_risk": "low", "consistency": "high"}'::jsonb,
    'system_import',
    'clinical_team'
) ON CONFLICT (guideline_id) DO UPDATE SET
    organization = EXCLUDED.organization,
    region = EXCLUDED.region,
    condition_primary = EXCLUDED.condition_primary,
    icd10_codes = EXCLUDED.icd10_codes,
    version = EXCLUDED.version,
    effective_date = EXCLUDED.effective_date,
    status = EXCLUDED.status,
    approval_status = EXCLUDED.approval_status,
    evidence_summary = EXCLUDED.evidence_summary,
    quality_metrics = EXCLUDED.quality_metrics,
    updated_at = NOW();

-- Guideline 8/10: ACC-AHA-HTN-2017-003
INSERT INTO guideline_evidence.guidelines (
    guideline_id,
    organization,
    region,
    condition_primary,
    icd10_codes,
    version,
    effective_date,
    status,
    approval_status,
    evidence_summary,
    quality_metrics,
    created_by,
    approved_by
) VALUES (
    'ACC-AHA-HTN-2017-003',
    'ACC/AHA',
    'US',
    'Hypertension - Treatment in Black Adults',
    '{"I10"}',
    '2017',
    '2017-11-13',
    'active',
    'approved',
    '{"recommendation": "In Black adults with hypertension but without HF or CKD, initial antihypertensive treatment should include a thiazide-type diuretic or calcium channel blocker.", "evidence_grade": "B-R", "strength_of_recommendation": "Moderate"}'::jsonb,
    '{"methodology_score": 9.0, "bias_risk": "low-moderate", "consistency": "high"}'::jsonb,
    'system_import',
    'clinical_team'
) ON CONFLICT (guideline_id) DO UPDATE SET
    organization = EXCLUDED.organization,
    region = EXCLUDED.region,
    condition_primary = EXCLUDED.condition_primary,
    icd10_codes = EXCLUDED.icd10_codes,
    version = EXCLUDED.version,
    effective_date = EXCLUDED.effective_date,
    status = EXCLUDED.status,
    approval_status = EXCLUDED.approval_status,
    evidence_summary = EXCLUDED.evidence_summary,
    quality_metrics = EXCLUDED.quality_metrics,
    updated_at = NOW();

-- Guideline 9/10: ACC-AHA-HTN-2017-004
INSERT INTO guideline_evidence.guidelines (
    guideline_id,
    organization,
    region,
    condition_primary,
    icd10_codes,
    version,
    effective_date,
    status,
    approval_status,
    evidence_summary,
    quality_metrics,
    created_by,
    approved_by
) VALUES (
    'ACC-AHA-HTN-2017-004',
    'ACC/AHA',
    'US',
    'Hypertension - Target in Older Adults',
    '{"I10"}',
    '2017',
    '2017-11-13',
    'active',
    'approved',
    '{"recommendation": "For community-dwelling adults \u226565 years, a Systolic BP target of <130 mmHg is recommended. Treatment decisions should be based on a shared decision-making process incorporating patient preferences and risk.", "evidence_grade": "A", "strength_of_recommendation": "Strong"}'::jsonb,
    '{"methodology_score": 9.6, "bias_risk": "low", "consistency": "moderate"}'::jsonb,
    'system_import',
    'clinical_team'
) ON CONFLICT (guideline_id) DO UPDATE SET
    organization = EXCLUDED.organization,
    region = EXCLUDED.region,
    condition_primary = EXCLUDED.condition_primary,
    icd10_codes = EXCLUDED.icd10_codes,
    version = EXCLUDED.version,
    effective_date = EXCLUDED.effective_date,
    status = EXCLUDED.status,
    approval_status = EXCLUDED.approval_status,
    evidence_summary = EXCLUDED.evidence_summary,
    quality_metrics = EXCLUDED.quality_metrics,
    updated_at = NOW();

-- Guideline 10/10: ACC-AHA-HTN-2017-005
INSERT INTO guideline_evidence.guidelines (
    guideline_id,
    organization,
    region,
    condition_primary,
    icd10_codes,
    version,
    effective_date,
    status,
    approval_status,
    evidence_summary,
    quality_metrics,
    created_by,
    approved_by
) VALUES (
    'ACC-AHA-HTN-2017-005',
    'ACC/AHA',
    'US',
    'Resistant Hypertension - Treatment',
    '{"I15.0"}',
    '2017',
    '2017-11-13',
    'active',
    'approved',
    '{"recommendation": "For patients with resistant hypertension, addition of a mineralocorticoid receptor antagonist (spironolactone or eplerenone) is recommended, pending appropriate renal function and potassium levels.", "evidence_grade": "B-R", "strength_of_recommendation": "Moderate"}'::jsonb,
    '{"methodology_score": 9.1, "bias_risk": "low", "consistency": "high"}'::jsonb,
    'system_import',
    'clinical_team'
) ON CONFLICT (guideline_id) DO UPDATE SET
    organization = EXCLUDED.organization,
    region = EXCLUDED.region,
    condition_primary = EXCLUDED.condition_primary,
    icd10_codes = EXCLUDED.icd10_codes,
    version = EXCLUDED.version,
    effective_date = EXCLUDED.effective_date,
    status = EXCLUDED.status,
    approval_status = EXCLUDED.approval_status,
    evidence_summary = EXCLUDED.evidence_summary,
    quality_metrics = EXCLUDED.quality_metrics,
    updated_at = NOW();

-- Verify import
SELECT COUNT(*) as total_guidelines FROM guideline_evidence.guidelines;
SELECT guideline_id, organization, condition_primary, status FROM guideline_evidence.guidelines ORDER BY guideline_id;

-- Commit transaction
COMMIT;

-- Display import summary
SELECT 'Import completed successfully' as status;

-- JSON data also saved to: guidelines_data.json
