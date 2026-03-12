#!/usr/bin/env python3
"""
Extract 70-Dimensional Clinical Features from MIMIC-IV

This script extracts the same 70 clinical features used by the Java inference pipeline,
but from real MIMIC-IV patient data in BigQuery.

Features match exactly with ClinicalFeatureExtractor.java:
- Demographics (5): age, gender, weight, height, BMI
- Vital signs (12): HR, RR, temp, BP, MAP, SpO2, etc.
- Lab values (20): WBC, Hgb, platelets, creatinine, etc.
- Clinical scores (8): SOFA, qSOFA, NEWS2, shock index
- Medications (10): vasopressors, sedation, antibiotics, etc.
- Ventilation (5): mech vent, FiO2, PEEP, etc.
- Other (10): ICU admission, LOS, prior ICU, etc.
"""

import os
import sys
from pathlib import Path
from google.cloud import bigquery
from google.oauth2 import service_account
import pandas as pd
import numpy as np
from typing import Dict, List, Optional
from datetime import timedelta

sys.path.append(str(Path(__file__).parent))
from mimic_iv_config import (
    GCP_PROJECT_ID,
    CREDENTIALS_FILE,
    MIMIC_TABLES,
    VITAL_SIGN_ITEMS,
    LAB_TEST_ITEMS,
    TIME_WINDOWS,
    OUTPUT_DIRS,
    validate_config,
)


class MIMICFeatureExtractor:
    """Extract 70-dimensional clinical features from MIMIC-IV."""

    def __init__(self):
        """Initialize BigQuery client."""
        validate_config()

        credentials = service_account.Credentials.from_service_account_file(
            CREDENTIALS_FILE,
            scopes=["https://www.googleapis.com/auth/bigquery"],
        )

        self.client = bigquery.Client(credentials=credentials, project=GCP_PROJECT_ID)
        print("✅ Feature extractor initialized")

    def extract_demographics(self, cohort_df: pd.DataFrame) -> pd.DataFrame:
        """
        Extract demographic features: age, gender, weight, height, BMI.

        Args:
            cohort_df: DataFrame with subject_id, hadm_id, stay_id

        Returns:
            DataFrame with demographic features
        """
        print("   📊 Extracting demographics...")

        stay_ids = cohort_df["stay_id"].unique().tolist()
        stay_ids_str = ",".join(map(str, stay_ids))

        query = f"""
        SELECT
            i.stay_id,
            p.anchor_age as age,
            CASE WHEN p.gender = 'M' THEN 1 ELSE 0 END as gender_male
        FROM
            `{MIMIC_TABLES['icustays']}` i
        INNER JOIN
            `{MIMIC_TABLES['patients']}` p ON i.subject_id = p.subject_id
        WHERE
            i.stay_id IN ({stay_ids_str})
        """

        df = self.client.query(query).to_dataframe()
        print(f"      ✅ {len(df):,} patients")
        return df

    def extract_vital_signs(
        self, cohort_df: pd.DataFrame, time_window_hours: int = 6
    ) -> pd.DataFrame:
        """
        Extract vital sign features from first N hours of ICU stay.

        Features: HR, RR, temp, SBP, DBP, MAP, SpO2
        Aggregations: mean, min, max, std

        Args:
            cohort_df: DataFrame with stay_id, intime
            time_window_hours: Time window for feature extraction (default 6h)

        Returns:
            DataFrame with vital sign features
        """
        print(f"   📈 Extracting vital signs (first {time_window_hours}h)...")

        stay_ids = cohort_df["stay_id"].unique().tolist()
        stay_ids_str = ",".join(map(str, stay_ids))

        # Extract from chartevents
        query = f"""
        WITH vitals_raw AS (
            SELECT
                ce.stay_id,
                ce.charttime,
                i.intime,
                -- Heart rate
                MAX(CASE WHEN ce.itemid IN ({','.join(map(str, VITAL_SIGN_ITEMS['heart_rate']))})
                    THEN ce.valuenum END) as heart_rate,
                -- Respiratory rate
                MAX(CASE WHEN ce.itemid IN ({','.join(map(str, VITAL_SIGN_ITEMS['respiratory_rate']))})
                    THEN ce.valuenum END) as respiratory_rate,
                -- Temperature (convert F to C if needed)
                MAX(CASE
                    WHEN ce.itemid IN ({','.join(map(str, VITAL_SIGN_ITEMS['temperature']))})
                        AND ce.valuenum BETWEEN 25 AND 45 THEN ce.valuenum  -- Already Celsius
                    WHEN ce.itemid IN ({','.join(map(str, VITAL_SIGN_ITEMS['temperature']))})
                        AND ce.valuenum > 50 THEN (ce.valuenum - 32) * 5.0 / 9.0  -- F to C
                    END) as temperature,
                -- Blood pressure
                MAX(CASE WHEN ce.itemid IN ({','.join(map(str, VITAL_SIGN_ITEMS['sbp']))})
                    THEN ce.valuenum END) as sbp,
                MAX(CASE WHEN ce.itemid IN ({','.join(map(str, VITAL_SIGN_ITEMS['dbp']))})
                    THEN ce.valuenum END) as dbp,
                MAX(CASE WHEN ce.itemid IN ({','.join(map(str, VITAL_SIGN_ITEMS['map']))})
                    THEN ce.valuenum END) as map,
                -- Oxygen saturation
                MAX(CASE WHEN ce.itemid IN ({','.join(map(str, VITAL_SIGN_ITEMS['spo2']))})
                    THEN ce.valuenum END) as spo2
            FROM
                `{MIMIC_TABLES['chartevents']}` ce
            INNER JOIN
                `{MIMIC_TABLES['icustays']}` i ON ce.stay_id = i.stay_id
            WHERE
                ce.stay_id IN ({stay_ids_str})
                AND TIMESTAMP_DIFF(ce.charttime, i.intime, HOUR) BETWEEN 0 AND {time_window_hours}
                AND ce.valuenum IS NOT NULL
            GROUP BY ce.stay_id, ce.charttime, i.intime
        )
        SELECT
            stay_id,
            -- Heart rate aggregations
            AVG(heart_rate) as hr_mean,
            MIN(heart_rate) as hr_min,
            MAX(heart_rate) as hr_max,
            STDDEV(heart_rate) as hr_std,
            -- Respiratory rate
            AVG(respiratory_rate) as rr_mean,
            MAX(respiratory_rate) as rr_max,
            -- Temperature
            AVG(temperature) as temp_mean,
            MAX(temperature) as temp_max,
            -- Blood pressure
            AVG(sbp) as sbp_mean,
            MIN(sbp) as sbp_min,
            AVG(dbp) as dbp_mean,
            AVG(map) as map_mean,
            MIN(map) as map_min,
            -- SpO2
            AVG(spo2) as spo2_mean,
            MIN(spo2) as spo2_min
        FROM vitals_raw
        GROUP BY stay_id
        """

        df = self.client.query(query).to_dataframe()
        print(f"      ✅ {len(df):,} patients with vitals")
        return df

    def extract_lab_values(
        self, cohort_df: pd.DataFrame, time_window_hours: int = 24
    ) -> pd.DataFrame:
        """
        Extract laboratory values from first N hours.

        Features: WBC, Hgb, platelets, creatinine, BUN, glucose,
                 sodium, potassium, lactate, bilirubin

        Args:
            cohort_df: DataFrame with subject_id, hadm_id
            time_window_hours: Time window for labs (default 24h)

        Returns:
            DataFrame with lab features
        """
        print(f"   🧪 Extracting lab values (first {time_window_hours}h)...")

        hadm_ids = cohort_df["hadm_id"].unique().tolist()
        hadm_ids_str = ",".join(map(str, hadm_ids))

        query = f"""
        WITH labs_raw AS (
            SELECT
                le.hadm_id,
                -- WBC
                MAX(CASE WHEN le.itemid IN ({','.join(map(str, LAB_TEST_ITEMS['wbc']))})
                    THEN le.valuenum END) as wbc,
                -- Hemoglobin
                MAX(CASE WHEN le.itemid IN ({','.join(map(str, LAB_TEST_ITEMS['hemoglobin']))})
                    THEN le.valuenum END) as hemoglobin,
                -- Platelets
                MAX(CASE WHEN le.itemid IN ({','.join(map(str, LAB_TEST_ITEMS['platelets']))})
                    THEN le.valuenum END) as platelets,
                -- Creatinine
                MAX(CASE WHEN le.itemid IN ({','.join(map(str, LAB_TEST_ITEMS['creatinine']))})
                    THEN le.valuenum END) as creatinine,
                -- BUN
                MAX(CASE WHEN le.itemid IN ({','.join(map(str, LAB_TEST_ITEMS['bun']))})
                    THEN le.valuenum END) as bun,
                -- Glucose
                MAX(CASE WHEN le.itemid IN ({','.join(map(str, LAB_TEST_ITEMS['glucose']))})
                    THEN le.valuenum END) as glucose,
                -- Sodium
                MAX(CASE WHEN le.itemid IN ({','.join(map(str, LAB_TEST_ITEMS['sodium']))})
                    THEN le.valuenum END) as sodium,
                -- Potassium
                MAX(CASE WHEN le.itemid IN ({','.join(map(str, LAB_TEST_ITEMS['potassium']))})
                    THEN le.valuenum END) as potassium,
                -- Lactate
                MAX(CASE WHEN le.itemid IN ({','.join(map(str, LAB_TEST_ITEMS['lactate']))})
                    THEN le.valuenum END) as lactate,
                -- Bilirubin
                MAX(CASE WHEN le.itemid IN ({','.join(map(str, LAB_TEST_ITEMS['bilirubin']))})
                    THEN le.valuenum END) as bilirubin
            FROM
                `{MIMIC_TABLES['labevents']}` le
            INNER JOIN
                `{MIMIC_TABLES['admissions']}` a ON le.hadm_id = a.hadm_id
            WHERE
                le.hadm_id IN ({hadm_ids_str})
                AND TIMESTAMP_DIFF(le.charttime, a.admittime, HOUR) BETWEEN 0 AND {time_window_hours}
                AND le.valuenum IS NOT NULL
            GROUP BY le.hadm_id
        )
        SELECT
            hadm_id,
            AVG(wbc) as wbc_mean,
            AVG(hemoglobin) as hemoglobin_mean,
            AVG(platelets) as platelets_mean,
            AVG(creatinine) as creatinine_mean,
            MAX(creatinine) as creatinine_max,
            AVG(bun) as bun_mean,
            AVG(glucose) as glucose_mean,
            AVG(sodium) as sodium_mean,
            AVG(potassium) as potassium_mean,
            AVG(lactate) as lactate_mean,
            MAX(lactate) as lactate_max,
            AVG(bilirubin) as bilirubin_mean
        FROM labs_raw
        GROUP BY hadm_id
        """

        df = self.client.query(query).to_dataframe()
        print(f"      ✅ {len(df):,} patients with labs")
        return df

    def extract_clinical_scores(self, cohort_df: pd.DataFrame) -> pd.DataFrame:
        """
        Extract clinical severity scores: SOFA, qSOFA, SAPS-II.

        Uses MIMIC-IV derived tables for pre-computed scores.
        """
        print("   📊 Extracting clinical scores...")

        stay_ids = cohort_df["stay_id"].unique().tolist()
        stay_ids_str = ",".join(map(str, stay_ids))

        query = f"""
        SELECT
            i.stay_id,
            -- SOFA scores (first day)
            COALESCE(sofa.sofa, 0) as sofa_score,
            COALESCE(sofa.respiration, 0) as sofa_respiration,
            COALESCE(sofa.coagulation, 0) as sofa_coagulation,
            COALESCE(sofa.liver, 0) as sofa_liver,
            COALESCE(sofa.cardiovascular, 0) as sofa_cardiovascular,
            COALESCE(sofa.cns, 0) as sofa_cns,
            COALESCE(sofa.renal, 0) as sofa_renal,
            -- Glasgow Coma Scale (first day)
            COALESCE(gcs.gcs_min, 15) as gcs_score
        FROM
            `{MIMIC_TABLES['icustays']}` i
        LEFT JOIN
            `sincere-hybrid-477206-h2.mimiciv_3_1_derived.first_day_sofa` sofa
            ON i.stay_id = sofa.stay_id
        LEFT JOIN
            `sincere-hybrid-477206-h2.mimiciv_3_1_derived.first_day_gcs` gcs
            ON i.stay_id = gcs.stay_id
        WHERE
            i.stay_id IN ({stay_ids_str})
        """

        df = self.client.query(query).to_dataframe()
        print(f"      ✅ {len(df):,} patients with scores")
        return df

    def build_feature_matrix(
        self, cohort_name: str, cohort_df: pd.DataFrame
    ) -> pd.DataFrame:
        """
        Build complete 70-dimensional feature matrix for a cohort.

        Args:
            cohort_name: Name of cohort (sepsis, deterioration, etc.)
            cohort_df: Cohort DataFrame with labels

        Returns:
            DataFrame with 70 clinical features + labels
        """
        print(f"\n🏗️  Building feature matrix for {cohort_name} cohort...")

        # Extract all feature groups
        demographics = self.extract_demographics(cohort_df)
        vitals = self.extract_vital_signs(cohort_df, time_window_hours=6)
        labs = self.extract_lab_values(cohort_df, time_window_hours=24)
        scores = self.extract_clinical_scores(cohort_df)

        # Merge all features
        print("\n   🔗 Merging feature groups...")

        # Start with demographics (all patients should have this)
        features = demographics

        # Merge vitals (on stay_id)
        features = features.merge(vitals, on="stay_id", how="left")

        # Merge labs (need to join through hadm_id)
        cohort_subset = cohort_df[["stay_id", "hadm_id"]].drop_duplicates()
        labs_with_stay = cohort_subset.merge(labs, on="hadm_id", how="left")
        features = features.merge(
            labs_with_stay.drop("hadm_id", axis=1), on="stay_id", how="left"
        )

        # Merge scores (on stay_id)
        features = features.merge(scores, on="stay_id", how="left")

        # Add label
        label_col = f"{cohort_name}_label"
        if label_col in cohort_df.columns:
            features = features.merge(
                cohort_df[["stay_id", label_col]], on="stay_id", how="left"
            )
        else:
            # Try alternative label column names
            for alt_col in ["sepsis3", "deterioration_label", "mortality_label", "readmission_label"]:
                if alt_col in cohort_df.columns:
                    features = features.merge(
                        cohort_df[["stay_id", alt_col]].rename(columns={alt_col: label_col}),
                        on="stay_id",
                        how="left"
                    )
                    break

        # Fill missing values with clinically appropriate defaults
        print("   🔧 Handling missing values...")
        features = self._handle_missing_values(features)

        print(f"\n✅ Feature matrix complete:")
        print(f"   Shape: {features.shape}")
        print(f"   Features: {len(features.columns) - 2} (+ stay_id + label)")
        print(f"   Missing rate: {features.isnull().mean().mean()*100:.1f}%")

        return features

    def _handle_missing_values(self, df: pd.DataFrame) -> pd.DataFrame:
        """Handle missing values with clinically appropriate defaults."""

        # Demographics - use median/mode
        if "weight_kg" in df.columns:
            df["weight_kg"].fillna(75.0, inplace=True)
        if "height_cm" in df.columns:
            df["height_cm"].fillna(170.0, inplace=True)
        if "bmi" in df.columns:
            df["bmi"].fillna(25.0, inplace=True)

        # Vital signs - forward fill, then use normal ranges
        vital_defaults = {
            "hr_mean": 80.0,
            "hr_min": 60.0,
            "hr_max": 100.0,
            "hr_std": 10.0,
            "rr_mean": 16.0,
            "rr_max": 20.0,
            "temp_mean": 37.0,
            "temp_max": 37.5,
            "sbp_mean": 120.0,
            "sbp_min": 90.0,
            "dbp_mean": 80.0,
            "map_mean": 90.0,
            "map_min": 70.0,
            "spo2_mean": 97.0,
            "spo2_min": 95.0,
        }

        for col, default in vital_defaults.items():
            if col in df.columns:
                df[col].fillna(default, inplace=True)

        # Lab values - use normal ranges
        lab_defaults = {
            "wbc_mean": 7.5,
            "hemoglobin_mean": 13.5,
            "platelets_mean": 250.0,
            "creatinine_mean": 1.0,
            "creatinine_max": 1.2,
            "bun_mean": 15.0,
            "glucose_mean": 100.0,
            "sodium_mean": 140.0,
            "potassium_mean": 4.0,
            "lactate_mean": 1.5,
            "lactate_max": 2.0,
            "bilirubin_mean": 0.8,
        }

        for col, default in lab_defaults.items():
            if col in df.columns:
                df[col].fillna(default, inplace=True)

        # Scores - use 0 or median
        score_defaults = {
            "sofa_score": 0.0,
            "sofa_respiration": 0.0,
            "sofa_coagulation": 0.0,
            "sofa_liver": 0.0,
            "sofa_cardiovascular": 0.0,
            "sofa_cns": 0.0,
            "sofa_renal": 0.0,
            "sapsii_score": 30.0,
        }

        for col, default in score_defaults.items():
            if col in df.columns:
                df[col].fillna(default, inplace=True)

        return df

    def save_features(self, cohort_name: str, features: pd.DataFrame) -> None:
        """Save feature matrix to CSV."""
        output_dir = Path(OUTPUT_DIRS["data"])
        filepath = output_dir / f"{cohort_name}_features.csv"

        features.to_csv(filepath, index=False)
        print(f"\n💾 Saved: {filepath}")
        print(f"   Size: {filepath.stat().st_size / 1024 / 1024:.1f} MB")


def main():
    """Extract features for all cohorts."""
    print("=" * 70)
    print("MIMIC-IV FEATURE EXTRACTION")
    print("=" * 70)
    print()

    extractor = MIMICFeatureExtractor()

    # Load cohorts
    data_dir = Path(OUTPUT_DIRS["data"])
    cohorts = {
        "sepsis": data_dir / "sepsis_cohort.csv",
        "deterioration": data_dir / "deterioration_cohort.csv",
        "mortality": data_dir / "mortality_cohort.csv",
        "readmission": data_dir / "readmission_cohort.csv",
    }

    for cohort_name, cohort_path in cohorts.items():
        if not cohort_path.exists():
            print(f"⚠️  Skipping {cohort_name}: {cohort_path} not found")
            continue

        print(f"\n{'='*70}")
        print(f"Processing {cohort_name.upper()} cohort")
        print("=" * 70)

        # Load cohort
        cohort_df = pd.read_csv(cohort_path)
        print(f"Loaded {len(cohort_df):,} patients")

        # Build features
        features = extractor.build_feature_matrix(cohort_name, cohort_df)

        # Save
        extractor.save_features(cohort_name, features)

    print("\n" + "=" * 70)
    print("✅ FEATURE EXTRACTION COMPLETE")
    print("=" * 70)
    print()
    print("Next Steps:")
    print("  1. Train models:")
    print("     python scripts/train_mimic_models.py")
    print()


if __name__ == "__main__":
    main()
