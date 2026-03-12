#!/usr/bin/env python3
"""
Extract BALANCED Clinical Cohorts from MIMIC-IV BigQuery

This script extracts patient cohorts with BOTH positive and negative cases:
1. Sepsis (positive) vs Non-sepsis ICU patients (negative)
2. Clinical deterioration (positive) vs Stable ICU patients (negative)
3. In-hospital mortality (died) vs Survivors (negative)

Uses MIMIC-IV v3.1 in GCP BigQuery.
"""

import os
import sys
from pathlib import Path
from google.cloud import bigquery
import pandas as pd

# Add script directory to path
sys.path.append(str(Path(__file__).parent))
from mimic_iv_config import (
    GCP_PROJECT_ID,
    MIMIC_TABLES,
    OUTPUT_DIRS,
    validate_config,
    create_bigquery_client,
)


class BalancedMIMICExtractor:
    """Extract balanced clinical cohorts from MIMIC-IV BigQuery."""

    def __init__(self):
        """Initialize BigQuery client."""
        validate_config()
        self.client = create_bigquery_client()
        print("✅ BigQuery client initialized")
        print(f"   Project: {GCP_PROJECT_ID}")
        print()

    def extract_sepsis_cohort_balanced(self) -> pd.DataFrame:
        """
        Extract BALANCED sepsis cohort (positive + negative cases).

        Positive cases: SOFA ≥2 + infection ICD codes
        Negative cases: ICU patients WITHOUT sepsis criteria
        Target: 50/50 split
        """
        print("🔬 Extracting BALANCED Sepsis Cohort...")

        query = f"""
        WITH infection_diagnoses AS (
            -- Patients with sepsis/infection ICD-10 codes
            SELECT DISTINCT subject_id, hadm_id
            FROM `{MIMIC_TABLES['diagnoses_icd']}`
            WHERE icd_version = 10
                AND (
                    icd_code LIKE 'A40%' OR icd_code LIKE 'A41%' OR icd_code LIKE 'R65.2%'
                    OR icd_code = 'A02.1' OR icd_code = 'A22.7' OR icd_code = 'A26.7'
                    OR icd_code = 'A32.7' OR icd_code = 'A42.7' OR icd_code = 'B37.7'
                )
        ),
        all_icu_patients AS (
            SELECT
                i.subject_id,
                i.hadm_id,
                i.stay_id,
                sofa.sofa as sofa_score,
                i.intime as icu_intime,
                i.outtime as icu_outtime,
                a.admittime,
                a.dischtime,
                a.hospital_expire_flag,
                p.anchor_age as age,
                p.gender,
                TIMESTAMP_DIFF(i.outtime, i.intime, HOUR) as icu_los_hours,
                -- Sepsis label
                CASE
                    WHEN inf.subject_id IS NOT NULL AND sofa.sofa >= 2 THEN 1
                    ELSE 0
                END as sepsis_label
            FROM `{MIMIC_TABLES['icustays']}` i
            INNER JOIN `{MIMIC_TABLES['admissions']}` a ON i.hadm_id = a.hadm_id
            INNER JOIN `{MIMIC_TABLES['patients']}` p ON i.subject_id = p.subject_id
            INNER JOIN `sincere-hybrid-477206-h2.mimiciv_3_1_derived.first_day_sofa` sofa ON i.stay_id = sofa.stay_id
            LEFT JOIN infection_diagnoses inf ON i.subject_id = inf.subject_id AND i.hadm_id = inf.hadm_id
            WHERE p.anchor_age >= 18
                AND TIMESTAMP_DIFF(i.outtime, i.intime, HOUR) >= 6
        ),
        positive_cases AS (
            SELECT * FROM all_icu_patients WHERE sepsis_label = 1 LIMIT 5000
        ),
        negative_cases AS (
            SELECT * FROM all_icu_patients WHERE sepsis_label = 0 LIMIT 5000
        )
        SELECT * FROM positive_cases
        UNION ALL
        SELECT * FROM negative_cases
        """

        print("   Running BigQuery query...")
        df = self.client.query(query).to_dataframe()

        print(f"✅ Balanced sepsis cohort extracted: {len(df):,} cases")
        print(f"   Positive (sepsis): {df['sepsis_label'].sum():,} ({100*df['sepsis_label'].mean():.1f}%)")
        print(f"   Negative (non-sepsis): {(df['sepsis_label']==0).sum():,} ({100*(1-df['sepsis_label'].mean()):.1f}%)")
        print(f"   Mean age: {df['age'].mean():.1f} years")
        print()

        return df

    def extract_deterioration_cohort_balanced(self) -> pd.DataFrame:
        """
        Extract BALANCED deterioration cohort.

        Positive: Died OR high SOFA (≥4) OR prolonged ICU (>3 days)
        Negative: Survived + low SOFA + short ICU stay
        """
        print("📉 Extracting BALANCED Deterioration Cohort...")

        query = f"""
        WITH all_icu_patients AS (
            SELECT
                i.subject_id,
                i.hadm_id,
                i.stay_id,
                i.intime,
                i.outtime,
                a.admittime,
                a.hospital_expire_flag,
                p.anchor_age as age,
                p.gender,
                sofa.sofa as sofa_score,
                TIMESTAMP_DIFF(i.outtime, i.intime, HOUR) as icu_los_hours,
                CASE
                    WHEN a.hospital_expire_flag = 1 THEN 1
                    WHEN sofa.sofa >= 4 THEN 1
                    WHEN TIMESTAMP_DIFF(i.outtime, i.intime, DAY) > 3 THEN 1
                    ELSE 0
                END as deterioration_label
            FROM `{MIMIC_TABLES['icustays']}` i
            INNER JOIN `{MIMIC_TABLES['admissions']}` a ON i.hadm_id = a.hadm_id
            INNER JOIN `{MIMIC_TABLES['patients']}` p ON i.subject_id = p.subject_id
            LEFT JOIN `sincere-hybrid-477206-h2.mimiciv_3_1_derived.first_day_sofa` sofa ON i.stay_id = sofa.stay_id
            WHERE p.anchor_age >= 18
                AND TIMESTAMP_DIFF(i.outtime, i.intime, HOUR) >= 24
        ),
        positive_cases AS (
            SELECT * FROM all_icu_patients WHERE deterioration_label = 1 LIMIT 4000
        ),
        negative_cases AS (
            SELECT * FROM all_icu_patients WHERE deterioration_label = 0 LIMIT 4000
        )
        SELECT * FROM positive_cases
        UNION ALL
        SELECT * FROM negative_cases
        """

        print("   Running BigQuery query...")
        df = self.client.query(query).to_dataframe()

        print(f"✅ Balanced deterioration cohort extracted: {len(df):,} cases")
        print(f"   Positive (deteriorated): {df['deterioration_label'].sum():,} ({100*df['deterioration_label'].mean():.1f}%)")
        print(f"   Negative (stable): {(df['deterioration_label']==0).sum():,} ({100*(1-df['deterioration_label'].mean()):.1f}%)")
        print()

        return df

    def extract_mortality_cohort_balanced(self) -> pd.DataFrame:
        """
        Extract BALANCED mortality cohort.

        Positive: Died in hospital
        Negative: Survived discharge
        """
        print("💀 Extracting BALANCED Mortality Cohort...")

        query = f"""
        WITH all_icu_patients AS (
            SELECT
                i.subject_id,
                i.hadm_id,
                i.stay_id,
                i.intime,
                i.outtime,
                a.admittime,
                a.dischtime,
                a.deathtime,
                a.hospital_expire_flag as mortality_label,
                p.anchor_age as age,
                p.gender,
                TIMESTAMP_DIFF(a.dischtime, a.admittime, DAY) as los_days,
                TIMESTAMP_DIFF(i.outtime, i.intime, DAY) as icu_los_days
            FROM `{MIMIC_TABLES['icustays']}` i
            INNER JOIN `{MIMIC_TABLES['admissions']}` a ON i.hadm_id = a.hadm_id
            INNER JOIN `{MIMIC_TABLES['patients']}` p ON i.subject_id = p.subject_id
            WHERE p.anchor_age >= 18
                AND TIMESTAMP_DIFF(i.outtime, i.intime, HOUR) >= 24
        ),
        positive_cases AS (
            SELECT * FROM all_icu_patients WHERE mortality_label = 1 LIMIT 2500
        ),
        negative_cases AS (
            SELECT * FROM all_icu_patients WHERE mortality_label = 0 LIMIT 2500
        )
        SELECT * FROM positive_cases
        UNION ALL
        SELECT * FROM negative_cases
        """

        print("   Running BigQuery query...")
        df = self.client.query(query).to_dataframe()

        print(f"✅ Balanced mortality cohort extracted: {len(df):,} cases")
        print(f"   Positive (died): {df['mortality_label'].sum():,} ({100*df['mortality_label'].mean():.1f}%)")
        print(f"   Negative (survived): {(df['mortality_label']==0).sum():,} ({100*(1-df['mortality_label'].mean()):.1f}%)")
        print()

        return df

    def save_cohorts(
        self,
        sepsis_df: pd.DataFrame,
        deterioration_df: pd.DataFrame,
        mortality_df: pd.DataFrame,
    ) -> None:
        """Save extracted cohorts to CSV files."""
        print("💾 Saving balanced cohorts to disk...")

        output_dir = Path(OUTPUT_DIRS["data"])
        output_dir.mkdir(parents=True, exist_ok=True)

        cohorts = {
            "sepsis": sepsis_df,
            "deterioration": deterioration_df,
            "mortality": mortality_df,
        }

        for name, df in cohorts.items():
            filepath = output_dir / f"{name}_cohort.csv"
            df.to_csv(filepath, index=False)
            print(f"   ✅ {filepath} ({len(df):,} rows)")

        print()
        print("✅ All balanced cohorts saved successfully!")


def main():
    """Extract all balanced cohorts from MIMIC-IV."""
    print("=" * 70)
    print("MIMIC-IV BALANCED COHORT EXTRACTION")
    print("=" * 70)
    print()

    # Initialize extractor
    extractor = BalancedMIMICExtractor()

    # Extract cohorts
    print("Starting balanced cohort extraction...")
    print()

    try:
        sepsis_df = extractor.extract_sepsis_cohort_balanced()
        deterioration_df = extractor.extract_deterioration_cohort_balanced()
        mortality_df = extractor.extract_mortality_cohort_balanced()

        # Save results
        extractor.save_cohorts(sepsis_df, deterioration_df, mortality_df)

        print("=" * 70)
        print("✅ BALANCED COHORT EXTRACTION COMPLETE")
        print("=" * 70)
        print()
        print("Summary:")
        print(f"  - Sepsis: {len(sepsis_df):,} cases ({sepsis_df['sepsis_label'].mean()*100:.1f}% positive)")
        print(f"  - Deterioration: {len(deterioration_df):,} cases ({deterioration_df['deterioration_label'].mean()*100:.1f}% positive)")
        print(f"  - Mortality: {len(mortality_df):,} cases ({mortality_df['mortality_label'].mean()*100:.1f}% positive)")
        print()
        print("Next Steps:")
        print("  1. Run feature extraction:")
        print("     python scripts/extract_mimic_features.py")
        print()

        return 0

    except Exception as e:
        print(f"❌ Error during extraction: {e}")
        import traceback
        traceback.print_exc()
        return 1


if __name__ == "__main__":
    sys.exit(main())
