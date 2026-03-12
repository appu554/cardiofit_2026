#!/usr/bin/env python3
"""
Extract Clinical Cohorts from MIMIC-IV BigQuery

This script extracts patient cohorts for 4 prediction tasks:
1. Sepsis onset (48-hour prediction window)
2. Clinical deterioration (6-24 hour prediction)
3. In-hospital mortality
4. 30-day readmission

Uses MIMIC-IV v3.1 in GCP BigQuery via PhysioNet access.
"""

import os
import sys
from pathlib import Path
from google.cloud import bigquery
from google.oauth2 import service_account
import pandas as pd
import numpy as np
from datetime import datetime, timedelta
from typing import Dict, List, Tuple

# Add script directory to path
sys.path.append(str(Path(__file__).parent))
from mimic_iv_config import (
    GCP_PROJECT_ID,
    MIMIC_TABLES,
    SEPSIS_ICD10_CODES,
    SEPSIS_CLINICAL_CRITERIA,
    OUTPUT_DIRS,
    validate_config,
    create_bigquery_client,
)


class MIMICCohortExtractor:
    """Extract clinical cohorts from MIMIC-IV BigQuery."""

    def __init__(self):
        """Initialize BigQuery client."""
        validate_config()

        # Create BigQuery client (supports both service account and personal auth)
        self.client = create_bigquery_client()

        print("✅ BigQuery client initialized")
        print(f"   Project: {GCP_PROJECT_ID}")
        print()

    def test_connection(self) -> bool:
        """Test BigQuery connection and MIMIC-IV access."""
        print("🔍 Testing BigQuery connection...")

        try:
            # Simple query to count patients
            query = f"""
                SELECT COUNT(*) as patient_count
                FROM `{MIMIC_TABLES['patients']}`
            """

            result = self.client.query(query).to_dataframe()
            patient_count = result['patient_count'].iloc[0]

            print(f"✅ Connection successful!")
            print(f"   MIMIC-IV Patients: {patient_count:,}")
            return True

        except Exception as e:
            print(f"❌ Connection failed: {e}")
            return False

    def extract_sepsis_cohort(self) -> pd.DataFrame:
        """
        Extract sepsis cohort using Sepsis-3 criteria.

        Sepsis-3 Definition (derived from available tables):
        - SOFA score ≥2 (organ dysfunction indicator)
        - Infection diagnosis (ICD-10 codes: A40%, A41%, R65.2%)
        - Patients with BOTH criteria = sepsis

        Returns:
            DataFrame with sepsis cases
        """
        print("🔬 Extracting Sepsis Cohort (Sepsis-3 criteria)...")
        print("   Deriving sepsis from SOFA scores + infection diagnoses...")

        # Derive sepsis from first_day_sofa + diagnoses_icd
        query = f"""
        WITH infection_diagnoses AS (
            -- Patients with sepsis/infection ICD-10 codes
            SELECT DISTINCT
                subject_id,
                hadm_id
            FROM
                `{MIMIC_TABLES['diagnoses_icd']}`
            WHERE
                icd_version = 10
                AND (
                    icd_code LIKE 'A40%'  -- Streptococcal sepsis
                    OR icd_code LIKE 'A41%'  -- Other sepsis
                    OR icd_code LIKE 'R65.2%'  -- Severe sepsis
                    OR icd_code = 'A02.1'  -- Salmonella sepsis
                    OR icd_code = 'A22.7'  -- Anthrax sepsis
                    OR icd_code = 'A26.7'  -- Erysipelothrix sepsis
                    OR icd_code = 'A32.7'  -- Listerial sepsis
                    OR icd_code = 'A42.7'  -- Actinomycotic sepsis
                    OR icd_code = 'B37.7'  -- Candidal sepsis
                )
        ),
        sepsis_cohort AS (
            SELECT
                i.subject_id,
                i.hadm_id,
                i.stay_id,
                1 as sepsis_label,  -- All cases in this cohort are sepsis
                sofa.sofa as sofa_score,
                i.intime as icu_intime,
                i.outtime as icu_outtime,
                a.admittime,
                a.dischtime,
                a.hospital_expire_flag,
                p.anchor_age as age,
                p.gender,
                -- ICU length of stay in hours
                TIMESTAMP_DIFF(i.outtime, i.intime, HOUR) as icu_los_hours
            FROM
                `{MIMIC_TABLES['icustays']}` i
            INNER JOIN
                `{MIMIC_TABLES['admissions']}` a
                ON i.hadm_id = a.hadm_id
            INNER JOIN
                `{MIMIC_TABLES['patients']}` p
                ON i.subject_id = p.subject_id
            INNER JOIN
                `sincere-hybrid-477206-h2.mimiciv_3_1_derived.first_day_sofa` sofa
                ON i.stay_id = sofa.stay_id
            INNER JOIN
                infection_diagnoses inf
                ON i.subject_id = inf.subject_id AND i.hadm_id = inf.hadm_id
            WHERE
                -- Adult patients only
                p.anchor_age >= 18
                -- ICU stay at least 6 hours (for feature extraction)
                AND TIMESTAMP_DIFF(i.outtime, i.intime, HOUR) >= 6
                -- SOFA score ≥2 (organ dysfunction)
                AND sofa.sofa >= 2
        )
        SELECT * FROM sepsis_cohort
        LIMIT 10000  -- Start with 10K cases for testing
        """

        print("   Running BigQuery query...")
        df = self.client.query(query).to_dataframe()

        print(f"✅ Sepsis cohort extracted: {len(df):,} cases")
        print(f"   Mean age: {df['age'].mean():.1f} years")
        print(f"   Gender distribution: {df['gender'].value_counts().to_dict()}")
        print(f"   Mean SOFA score: {df['sofa_score'].mean():.1f}")
        print(f"   Mortality rate: {df['hospital_expire_flag'].mean()*100:.1f}%")
        print()

        return df

    def extract_deterioration_cohort(self) -> pd.DataFrame:
        """
        Extract clinical deterioration cohort.

        Deterioration indicators (simplified for available tables):
        - In-hospital mortality (definite deterioration)
        - High SOFA score (≥4) indicating organ dysfunction
        - ICU length of stay >3 days (prolonged critical illness)
        """
        print("📉 Extracting Clinical Deterioration Cohort...")
        print("   Using hospital mortality + high SOFA scores as deterioration markers...")

        query = f"""
        WITH deterioration_events AS (
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
                -- Deterioration label (death OR high SOFA indicating severe organ dysfunction)
                CASE
                    WHEN a.hospital_expire_flag = 1 THEN 1  -- Died in hospital
                    WHEN sofa.sofa >= 4 THEN 1  -- High SOFA score (severe organ dysfunction)
                    WHEN TIMESTAMP_DIFF(i.outtime, i.intime, DAY) > 3 THEN 1  -- Prolonged ICU stay
                    ELSE 0
                END as deterioration_label
            FROM
                `{MIMIC_TABLES['icustays']}` i
            INNER JOIN
                `{MIMIC_TABLES['admissions']}` a
                ON i.hadm_id = a.hadm_id
            INNER JOIN
                `{MIMIC_TABLES['patients']}` p
                ON i.subject_id = p.subject_id
            LEFT JOIN
                `sincere-hybrid-477206-h2.mimiciv_3_1_derived.first_day_sofa` sofa
                ON i.stay_id = sofa.stay_id
            WHERE
                p.anchor_age >= 18
                AND TIMESTAMP_DIFF(i.outtime, i.intime, HOUR) >= 24
        )
        SELECT * FROM deterioration_events
        WHERE deterioration_label = 1
        LIMIT 8000  -- 8K deterioration cases
        """

        print("   Running BigQuery query...")
        df = self.client.query(query).to_dataframe()

        print(f"✅ Deterioration cohort extracted: {len(df):,} cases")
        print(f"   Mean age: {df['age'].mean():.1f} years")
        print(f"   Mean SOFA score: {df['sofa_score'].mean():.1f}")
        print(f"   Mean ICU LOS: {df['icu_los_hours'].mean()/24:.1f} days")
        print(f"   Mortality rate: {df['hospital_expire_flag'].mean()*100:.1f}%")
        print()

        return df

    def extract_mortality_cohort(self) -> pd.DataFrame:
        """
        Extract in-hospital mortality cohort.

        Outcome: Hospital death (not just ICU death)
        Prediction window: First 24 hours of ICU admission
        """
        print("💀 Extracting Mortality Cohort...")

        query = f"""
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
            -- Length of stay
            TIMESTAMP_DIFF(a.dischtime, a.admittime, DAY) as los_days,
            TIMESTAMP_DIFF(i.outtime, i.intime, DAY) as icu_los_days
        FROM
            `{MIMIC_TABLES['icustays']}` i
        INNER JOIN
            `{MIMIC_TABLES['admissions']}` a
            ON i.hadm_id = a.hadm_id
        INNER JOIN
            `{MIMIC_TABLES['patients']}` p
            ON i.subject_id = p.subject_id
        WHERE
            p.anchor_age >= 18
            AND TIMESTAMP_DIFF(i.outtime, i.intime, HOUR) >= 24
            AND a.hospital_expire_flag = 1  -- Died in hospital
        LIMIT 5000  -- 5K mortality cases
        """

        print("   Running BigQuery query...")
        df = self.client.query(query).to_dataframe()

        print(f"✅ Mortality cohort extracted: {len(df):,} cases")
        print(f"   Mean age: {df['age'].mean():.1f} years")
        print(f"   Mean LOS: {df['los_days'].mean():.1f} days")
        print(f"   Mean ICU LOS: {df['icu_los_days'].mean():.1f} days")
        print()

        return df

    def extract_readmission_cohort(self) -> pd.DataFrame:
        """
        Extract 30-day readmission cohort.

        Outcome: Unplanned hospital readmission within 30 days of discharge
        Excludes: Planned readmissions, transfers, deaths
        """
        print("🔄 Extracting Readmission Cohort...")

        query = f"""
        WITH admissions_with_readmit AS (
            SELECT
                a1.subject_id,
                a1.hadm_id,
                a1.admittime,
                a1.dischtime,
                a1.admission_type,
                a1.hospital_expire_flag,
                p.anchor_age as age,
                p.gender,
                -- Next admission
                LEAD(a1.admittime) OVER (
                    PARTITION BY a1.subject_id
                    ORDER BY a1.admittime
                ) as next_admittime,
                LEAD(a1.admission_type) OVER (
                    PARTITION BY a1.subject_id
                    ORDER BY a1.admittime
                ) as next_admission_type,
                -- Days to readmission
                DATE_DIFF(
                    DATE(LEAD(a1.admittime) OVER (
                        PARTITION BY a1.subject_id ORDER BY a1.admittime
                    )),
                    DATE(a1.dischtime),
                    DAY
                ) as days_to_readmit
            FROM
                `{MIMIC_TABLES['admissions']}` a1
            INNER JOIN
                `{MIMIC_TABLES['patients']}` p
                ON a1.subject_id = p.subject_id
            WHERE
                p.anchor_age >= 18
                AND a1.hospital_expire_flag = 0  -- Survived discharge
        ),
        readmissions_with_icu AS (
            SELECT
                r.subject_id,
                r.hadm_id,
                i.stay_id,
                r.admittime,
                r.dischtime,
                r.age,
                r.gender,
                r.days_to_readmit,
                CASE
                    WHEN r.days_to_readmit <= 30
                        AND r.next_admission_type != 'ELECTIVE'
                        THEN 1
                    ELSE 0
                END as readmission_label
            FROM admissions_with_readmit r
            INNER JOIN
                `{MIMIC_TABLES['icustays']}` i
                ON r.hadm_id = i.hadm_id
            WHERE r.days_to_readmit IS NOT NULL
                AND r.days_to_readmit <= 30
                AND r.next_admission_type != 'ELECTIVE'
        )
        SELECT * FROM readmissions_with_icu
        LIMIT 10000  -- 10K readmission cases
        """

        print("   Running BigQuery query...")
        df = self.client.query(query).to_dataframe()

        print(f"✅ Readmission cohort extracted: {len(df):,} cases")
        print(f"   Mean days to readmit: {df['days_to_readmit'].mean():.1f}")
        print(f"   Readmission rate: {df['readmission_label'].mean()*100:.1f}%")
        print()

        return df

    def save_cohorts(
        self,
        sepsis_df: pd.DataFrame,
        deterioration_df: pd.DataFrame,
        mortality_df: pd.DataFrame,
        readmission_df: pd.DataFrame,
    ) -> None:
        """Save extracted cohorts to CSV files."""
        print("💾 Saving cohorts to disk...")

        output_dir = Path(OUTPUT_DIRS["data"])
        output_dir.mkdir(parents=True, exist_ok=True)

        cohorts = {
            "sepsis": sepsis_df,
            "deterioration": deterioration_df,
            "mortality": mortality_df,
            "readmission": readmission_df,
        }

        for name, df in cohorts.items():
            filepath = output_dir / f"{name}_cohort.csv"
            df.to_csv(filepath, index=False)
            print(f"   ✅ {filepath} ({len(df):,} rows)")

        print()
        print("✅ All cohorts saved successfully!")


def main():
    """Extract all cohorts from MIMIC-IV."""
    print("=" * 70)
    print("MIMIC-IV COHORT EXTRACTION")
    print("=" * 70)
    print()

    # Initialize extractor
    extractor = MIMICCohortExtractor()

    # Test connection
    if not extractor.test_connection():
        print("❌ Failed to connect to BigQuery. Check credentials.")
        return 1

    # Extract cohorts
    print("Starting cohort extraction...")
    print()

    try:
        sepsis_df = extractor.extract_sepsis_cohort()
        deterioration_df = extractor.extract_deterioration_cohort()
        mortality_df = extractor.extract_mortality_cohort()
        readmission_df = extractor.extract_readmission_cohort()

        # Save results
        extractor.save_cohorts(
            sepsis_df, deterioration_df, mortality_df, readmission_df
        )

        print("=" * 70)
        print("✅ COHORT EXTRACTION COMPLETE")
        print("=" * 70)
        print()
        print("Summary:")
        print(f"  - Sepsis: {len(sepsis_df):,} cases")
        print(f"  - Deterioration: {len(deterioration_df):,} cases")
        print(f"  - Mortality: {len(mortality_df):,} cases")
        print(f"  - Readmission: {len(readmission_df):,} cases")
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
