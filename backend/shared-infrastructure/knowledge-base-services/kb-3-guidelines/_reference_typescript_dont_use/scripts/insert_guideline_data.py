#!/usr/bin/env python3
"""
Script to convert YAML guideline data to JSON and generate SQL INSERT statements
for KB-3 Guidelines database.
"""

import json
from datetime import datetime
from typing import Dict, List, Any

# Define the guideline data in Python dict format (converted from YAML)
guidelines_data = [
    # Diabetes Guidelines
    {
        "guideline_id": "ADA-DM-2025-001",
        "organization": "ADA",
        "title": "Initial Glycemic Target for T2DM",
        "region": "Global",
        "condition_primary": "Type 2 Diabetes Mellitus - Glycemic Targets",
        "icd10_codes": ["E11.9"],
        "version": "2025",
        "effective_date": "2025-01-01",
        "status": "active",
        "approval_status": "approved",
        "evidence_summary": {
            "recommendation": "For most nonpregnant adults with type 2 diabetes, a reasonable A1C goal is <7% (53 mmol/mol).",
            "evidence_grade": "A",
            "strength_of_recommendation": "Strong"
        },
        "quality_metrics": {
            "methodology_score": 9.5,
            "bias_risk": "low",
            "consistency": "high"
        }
    },
    {
        "guideline_id": "ADA-DM-2025-002",
        "organization": "ADA",
        "title": "First-Line T2DM Pharmacotherapy",
        "region": "Global",
        "condition_primary": "Type 2 Diabetes Mellitus - First-Line Therapy",
        "icd10_codes": ["E11.9"],
        "version": "2025",
        "effective_date": "2025-01-01",
        "status": "active",
        "approval_status": "approved",
        "evidence_summary": {
            "recommendation": "Metformin is the preferred initial pharmacologic agent for the treatment of type 2 diabetes.",
            "evidence_grade": "A",
            "strength_of_recommendation": "Strong"
        },
        "quality_metrics": {
            "methodology_score": 9.8,
            "bias_risk": "low",
            "consistency": "high"
        }
    },
    {
        "guideline_id": "ADA-DM-2025-003",
        "organization": "ADA",
        "title": "T2DM Therapy in Patients with ASCVD",
        "region": "Global",
        "condition_primary": "Type 2 Diabetes Mellitus - High-Risk ASCVD",
        "icd10_codes": ["E11.9", "I25.10"],
        "version": "2025",
        "effective_date": "2025-01-01",
        "status": "active",
        "approval_status": "approved",
        "evidence_summary": {
            "recommendation": "In patients with type 2 diabetes and established ASCVD, a GLP-1 receptor agonist or SGLT2 inhibitor with demonstrated cardiovascular benefit is recommended.",
            "evidence_grade": "A",
            "strength_of_recommendation": "Strong"
        },
        "quality_metrics": {
            "methodology_score": 9.7,
            "bias_risk": "low",
            "consistency": "high"
        }
    },
    {
        "guideline_id": "KDIGO-DM-CKD-2024-001",
        "organization": "KDIGO",
        "title": "T2DM Therapy in Patients with CKD",
        "region": "Global",
        "condition_primary": "Type 2 Diabetes Mellitus with CKD",
        "icd10_codes": ["E11.22", "N18.3"],
        "version": "2024",
        "effective_date": "2024-10-01",
        "status": "active",
        "approval_status": "approved",
        "evidence_summary": {
            "recommendation": "For patients with T2DM and CKD (eGFR >20), treatment with an SGLT2 inhibitor is recommended to reduce risk of CKD progression and cardiovascular events.",
            "evidence_grade": "A",
            "strength_of_recommendation": "Strong"
        },
        "quality_metrics": {
            "methodology_score": 9.6,
            "bias_risk": "low",
            "consistency": "high"
        }
    },
    {
        "guideline_id": "ADA-DM-HTN-2025-001",
        "organization": "ADA",
        "title": "Blood Pressure Control in Diabetes",
        "region": "Global",
        "condition_primary": "Hypertension in Patients with Diabetes",
        "icd10_codes": ["E11.9", "I10"],
        "version": "2025",
        "effective_date": "2025-01-01",
        "status": "active",
        "approval_status": "approved",
        "evidence_summary": {
            "recommendation": "For individuals with diabetes and hypertension, a blood pressure target of <130/80 mmHg is recommended. An ACE inhibitor or ARB is recommended as first-line therapy if albuminuria is present.",
            "evidence_grade": "A",
            "strength_of_recommendation": "Strong"
        },
        "quality_metrics": {
            "methodology_score": 9.5,
            "bias_risk": "low",
            "consistency": "high"
        }
    },
    # Hypertension Guidelines
    {
        "guideline_id": "ACC-AHA-HTN-2017-001",
        "organization": "ACC/AHA",
        "title": "Definition of Stage 1 Hypertension",
        "region": "US",
        "condition_primary": "Hypertension - Stage 1 Definition",
        "icd10_codes": ["I10"],
        "version": "2017",
        "effective_date": "2017-11-13",
        "status": "active",
        "approval_status": "approved",
        "evidence_summary": {
            "recommendation": "Stage 1 Hypertension is defined as a Systolic BP of 130-139 mmHg or a Diastolic BP of 80-89 mmHg.",
            "evidence_grade": "B-R",
            "strength_of_recommendation": "Moderate"
        },
        "quality_metrics": {
            "methodology_score": 9.2,
            "bias_risk": "low",
            "consistency": "high"
        }
    },
    {
        "guideline_id": "ACC-AHA-HTN-2017-002",
        "organization": "ACC/AHA",
        "title": "Initial Pharmacotherapy for Stage 1 HTN",
        "region": "US",
        "condition_primary": "Hypertension - Stage 1 Treatment Initiation",
        "icd10_codes": ["I10"],
        "version": "2017",
        "effective_date": "2017-11-13",
        "status": "active",
        "approval_status": "approved",
        "evidence_summary": {
            "recommendation": "For Stage 1 Hypertension with an estimated 10-year ASCVD risk of ≥10%, initiate BP-lowering medication. First-line agents include thiazide diuretics, CCBs, and ACE inhibitors or ARBs.",
            "evidence_grade": "A",
            "strength_of_recommendation": "Strong"
        },
        "quality_metrics": {
            "methodology_score": 9.8,
            "bias_risk": "low",
            "consistency": "high"
        }
    },
    {
        "guideline_id": "ACC-AHA-HTN-2017-003",
        "organization": "ACC/AHA",
        "title": "Treatment of HTN in Black Adults",
        "region": "US",
        "condition_primary": "Hypertension - Treatment in Black Adults",
        "icd10_codes": ["I10"],
        "version": "2017",
        "effective_date": "2017-11-13",
        "status": "active",
        "approval_status": "approved",
        "evidence_summary": {
            "recommendation": "In Black adults with hypertension but without HF or CKD, initial antihypertensive treatment should include a thiazide-type diuretic or calcium channel blocker.",
            "evidence_grade": "B-R",
            "strength_of_recommendation": "Moderate"
        },
        "quality_metrics": {
            "methodology_score": 9.0,
            "bias_risk": "low-moderate",
            "consistency": "high"
        }
    },
    {
        "guideline_id": "ACC-AHA-HTN-2017-004",
        "organization": "ACC/AHA",
        "title": "BP Target in Older Adults",
        "region": "US",
        "condition_primary": "Hypertension - Target in Older Adults",
        "icd10_codes": ["I10"],
        "version": "2017",
        "effective_date": "2017-11-13",
        "status": "active",
        "approval_status": "approved",
        "evidence_summary": {
            "recommendation": "For community-dwelling adults ≥65 years, a Systolic BP target of <130 mmHg is recommended. Treatment decisions should be based on a shared decision-making process incorporating patient preferences and risk.",
            "evidence_grade": "A",
            "strength_of_recommendation": "Strong"
        },
        "quality_metrics": {
            "methodology_score": 9.6,
            "bias_risk": "low",
            "consistency": "moderate"
        }
    },
    {
        "guideline_id": "ACC-AHA-HTN-2017-005",
        "organization": "ACC/AHA",
        "title": "Treatment of Resistant Hypertension",
        "region": "US",
        "condition_primary": "Resistant Hypertension - Treatment",
        "icd10_codes": ["I15.0"],
        "version": "2017",
        "effective_date": "2017-11-13",
        "status": "active",
        "approval_status": "approved",
        "evidence_summary": {
            "recommendation": "For patients with resistant hypertension, addition of a mineralocorticoid receptor antagonist (spironolactone or eplerenone) is recommended, pending appropriate renal function and potassium levels.",
            "evidence_grade": "B-R",
            "strength_of_recommendation": "Moderate"
        },
        "quality_metrics": {
            "methodology_score": 9.1,
            "bias_risk": "low",
            "consistency": "high"
        }
    }
]

def generate_sql_insert(guideline: Dict[str, Any]) -> str:
    """
    Generate SQL INSERT statement for a single guideline.
    """
    # Convert lists to PostgreSQL array format
    icd10_codes = '{' + ','.join(f'"{code}"' for code in guideline['icd10_codes']) + '}'

    # Convert dicts to JSON strings with proper escaping
    evidence_summary = json.dumps(guideline['evidence_summary']).replace("'", "''")
    quality_metrics = json.dumps(guideline['quality_metrics']).replace("'", "''")

    sql = f"""
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
    '{guideline['guideline_id']}',
    '{guideline['organization']}',
    '{guideline['region']}',
    '{guideline['condition_primary']}',
    '{icd10_codes}',
    '{guideline['version']}',
    '{guideline['effective_date']}',
    '{guideline['status']}',
    '{guideline['approval_status']}',
    '{evidence_summary}'::jsonb,
    '{quality_metrics}'::jsonb,
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
"""
    return sql.strip()

def main():
    """
    Main function to generate all SQL INSERT statements.
    """
    print("-- ============================================================================")
    print("-- KB-3 Guidelines Data Import Script")
    print(f"-- Generated: {datetime.now().isoformat()}")
    print("-- Total Guidelines: {}".format(len(guidelines_data)))
    print("-- ============================================================================")
    print()
    print("-- Set schema")
    print("SET search_path TO guideline_evidence;")
    print()
    print("-- Begin transaction for atomic import")
    print("BEGIN;")
    print()

    # Generate INSERT statements for each guideline
    for i, guideline in enumerate(guidelines_data, 1):
        print(f"-- Guideline {i}/{len(guidelines_data)}: {guideline['guideline_id']}")
        print(generate_sql_insert(guideline))
        print()

    print("-- Verify import")
    print("SELECT COUNT(*) as total_guidelines FROM guideline_evidence.guidelines;")
    print("SELECT guideline_id, organization, condition_primary, status FROM guideline_evidence.guidelines ORDER BY guideline_id;")
    print()
    print("-- Commit transaction")
    print("COMMIT;")
    print()
    print("-- Display import summary")
    print("SELECT 'Import completed successfully' as status;")

    # Also save as JSON for reference
    json_output = {
        "import_metadata": {
            "timestamp": datetime.now().isoformat(),
            "total_guidelines": len(guidelines_data),
            "categories": {
                "diabetes": 5,
                "hypertension": 5
            }
        },
        "guidelines": guidelines_data
    }

    with open('/Users/apoorvabk/Downloads/cardiofit/backend/services/medication-service/knowledge-bases/kb-3-guidelines/scripts/guidelines_data.json', 'w') as f:
        json.dump(json_output, f, indent=2)

    print()
    print("-- JSON data also saved to: guidelines_data.json")

if __name__ == "__main__":
    main()