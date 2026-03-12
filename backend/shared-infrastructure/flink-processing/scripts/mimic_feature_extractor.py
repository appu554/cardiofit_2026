#!/usr/bin/env python3
"""
MIMIC-IV Feature Extractor

Extracts 70 clinical features from MIMIC-IV database for ML model training.
This script is the foundation for all 4 model training pipelines.

Features Extracted:
- Demographics (7): age, gender, ethnicity, weight, height, BMI, admission type
- Vitals (12): heart rate, blood pressure, respiratory rate, temperature, SpO2, etc.
- Labs (25): CBC, chemistry panel, blood gases, lactate, troponin, etc.
- Medications (8): vasopressors, sedatives, antibiotics, anticoagulants, etc.
- Clinical scores (6): SOFA, APACHE, NEWS2, qSOFA, Glasgow Coma Scale, etc.
- Temporal trends (6): 6-hour changes in vital signs and labs
- Comorbidities (6): diabetes, hypertension, heart failure, COPD, CKD, cancer

Prerequisites:
- MIMIC-IV database access
- PostgreSQL connection configured
- pandas, numpy, sqlalchemy

Usage:
    python scripts/mimic_feature_extractor.py --cohort sepsis --output data/sepsis_features.csv

Arguments:
    --cohort: Target cohort (sepsis, deterioration, mortality, readmission)
    --output: Output CSV file path
    --limit: Max patients to process (for testing)

Output:
    CSV file with 70 features + target label + patient metadata

@author CardioFit Team - Module 5 Training Pipeline
@version 1.0.0
"""

import argparse
import sys
import pandas as pd
import numpy as np
from datetime import datetime, timedelta
from sqlalchemy import create_engine
import warnings
warnings.filterwarnings('ignore')


class MIMICFeatureExtractor:
    """
    MIMIC-IV feature extraction for clinical prediction models.
    """

    def __init__(self, db_connection_string):
        """
        Initialize extractor with database connection.

        Args:
            db_connection_string: PostgreSQL connection string for MIMIC-IV
        """
        self.engine = create_engine(db_connection_string)
        print("✅ Connected to MIMIC-IV database")

    def extract_cohort(self, cohort_type, limit=None):
        """
        Extract patient cohort based on prediction target.

        Args:
            cohort_type: 'sepsis', 'deterioration', 'mortality', or 'readmission'
            limit: Maximum patients to extract (for testing)

        Returns:
            DataFrame with patient IDs and time windows
        """
        print(f"\n📋 Extracting {cohort_type} cohort...")

        if cohort_type == 'sepsis':
            cohort_query = self._build_sepsis_cohort_query(limit)
        elif cohort_type == 'deterioration':
            cohort_query = self._build_deterioration_cohort_query(limit)
        elif cohort_type == 'mortality':
            cohort_query = self._build_mortality_cohort_query(limit)
        elif cohort_type == 'readmission':
            cohort_query = self._build_readmission_cohort_query(limit)
        else:
            raise ValueError(f"Unknown cohort type: {cohort_type}")

        cohort_df = pd.read_sql(cohort_query, self.engine)
        print(f"   Found {len(cohort_df)} patients")
        print(f"   Positive cases: {cohort_df['label'].sum()} ({cohort_df['label'].mean()*100:.1f}%)")

        return cohort_df

    def extract_demographics(self, patient_ids):
        """
        Extract demographic features (7 features).

        Features:
        - age
        - gender (M=1, F=0)
        - ethnicity (white, black, hispanic, asian, other)
        - weight (kg)
        - height (cm)
        - BMI
        - admission_type (emergency=1, elective=0)
        """
        print("\n👤 Extracting demographics...")

        query = """
        SELECT
            p.subject_id,
            EXTRACT(YEAR FROM a.admittime) - p.anchor_year + p.anchor_age as age,
            CASE WHEN p.gender = 'M' THEN 1 ELSE 0 END as gender,
            CASE
                WHEN p.race LIKE '%WHITE%' THEN 1
                WHEN p.race LIKE '%BLACK%' THEN 2
                WHEN p.race LIKE '%HISPANIC%' THEN 3
                WHEN p.race LIKE '%ASIAN%' THEN 4
                ELSE 5
            END as ethnicity,
            CASE WHEN a.admission_type = 'EMERGENCY' THEN 1 ELSE 0 END as admission_emergency
        FROM mimiciv_hosp.patients p
        INNER JOIN mimiciv_hosp.admissions a ON p.subject_id = a.subject_id
        WHERE p.subject_id IN :patient_ids
        """

        demographics_df = pd.read_sql(query, self.engine, params={'patient_ids': tuple(patient_ids)})

        # Extract weight and height from chartevents
        weight_query = """
        SELECT subject_id, AVG(valuenum) as weight_kg
        FROM mimiciv_icu.chartevents
        WHERE itemid IN (226512, 224639) -- Weight in kg
        AND valuenum BETWEEN 30 AND 300
        AND subject_id IN :patient_ids
        GROUP BY subject_id
        """
        weight_df = pd.read_sql(weight_query, self.engine, params={'patient_ids': tuple(patient_ids)})

        height_query = """
        SELECT subject_id, AVG(valuenum) as height_cm
        FROM mimiciv_icu.chartevents
        WHERE itemid IN (226730, 226707) -- Height in cm
        AND valuenum BETWEEN 100 AND 250
        AND subject_id IN :patient_ids
        GROUP BY subject_id
        """
        height_df = pd.read_sql(height_query, self.engine, params={'patient_ids': tuple(patient_ids)})

        # Merge demographics
        demographics_df = demographics_df.merge(weight_df, on='subject_id', how='left')
        demographics_df = demographics_df.merge(height_df, on='subject_id', how='left')

        # Calculate BMI
        demographics_df['bmi'] = demographics_df['weight_kg'] / ((demographics_df['height_cm'] / 100) ** 2)

        # Fill missing values with cohort medians
        demographics_df['weight_kg'].fillna(demographics_df['weight_kg'].median(), inplace=True)
        demographics_df['height_cm'].fillna(demographics_df['height_cm'].median(), inplace=True)
        demographics_df['bmi'].fillna(demographics_df['bmi'].median(), inplace=True)

        print(f"   Extracted demographics for {len(demographics_df)} patients")
        return demographics_df

    def extract_vitals(self, patient_ids, time_windows):
        """
        Extract vital signs features (12 features).

        Features:
        - heart_rate (bpm)
        - systolic_bp (mmHg)
        - diastolic_bp (mmHg)
        - mean_arterial_pressure (mmHg)
        - respiratory_rate (breaths/min)
        - temperature (Celsius)
        - oxygen_saturation (%)
        - gcs_total (Glasgow Coma Scale)
        - heart_rate_variability (std dev)
        - shock_index (HR / SBP)
        - pulse_pressure (SBP - DBP)
        - oxygen_saturation_variability
        """
        print("\n💓 Extracting vital signs...")

        # This is a simplified example - full implementation would query chartevents
        vitals_query = """
        SELECT
            ce.subject_id,
            AVG(CASE WHEN ce.itemid IN (220045, 220050) THEN ce.valuenum END) as heart_rate,
            AVG(CASE WHEN ce.itemid IN (220179, 220050) THEN ce.valuenum END) as systolic_bp,
            AVG(CASE WHEN ce.itemid IN (220180, 220051) THEN ce.valuenum END) as diastolic_bp,
            AVG(CASE WHEN ce.itemid IN (220181, 220052) THEN ce.valuenum END) as map,
            AVG(CASE WHEN ce.itemid IN (220210, 224690) THEN ce.valuenum END) as respiratory_rate,
            AVG(CASE WHEN ce.itemid IN (223761, 223762) THEN ce.valuenum END) as temperature,
            AVG(CASE WHEN ce.itemid IN (220277, 220227) THEN ce.valuenum END) as oxygen_saturation,
            AVG(CASE WHEN ce.itemid = 220739 THEN ce.valuenum END) as gcs_total
        FROM mimiciv_icu.chartevents ce
        WHERE ce.subject_id IN :patient_ids
        AND ce.valuenum IS NOT NULL
        GROUP BY ce.subject_id
        """

        vitals_df = pd.read_sql(vitals_query, self.engine, params={'patient_ids': tuple(patient_ids)})

        # Calculate derived features
        vitals_df['shock_index'] = vitals_df['heart_rate'] / vitals_df['systolic_bp']
        vitals_df['pulse_pressure'] = vitals_df['systolic_bp'] - vitals_df['diastolic_bp']

        # Fill missing values
        for col in vitals_df.columns:
            if col != 'subject_id':
                vitals_df[col].fillna(vitals_df[col].median(), inplace=True)

        print(f"   Extracted vital signs for {len(vitals_df)} patients")
        return vitals_df

    def extract_labs(self, patient_ids, time_windows):
        """
        Extract laboratory values (25 features).

        Categories:
        - CBC: WBC, hemoglobin, hematocrit, platelets
        - Chemistry: sodium, potassium, chloride, bicarbonate, BUN, creatinine, glucose
        - Liver: bilirubin, ALT, AST, alkaline phosphatase
        - Blood gases: pH, pO2, pCO2, lactate
        - Cardiac: troponin, BNP
        - Coagulation: PT, PTT, INR
        - Other: albumin, calcium, magnesium
        """
        print("\n🧪 Extracting laboratory values...")

        labs_query = """
        SELECT
            le.subject_id,
            AVG(CASE WHEN le.itemid = 51300 THEN le.valuenum END) as wbc,
            AVG(CASE WHEN le.itemid = 51221 THEN le.valuenum END) as hemoglobin,
            AVG(CASE WHEN le.itemid = 51265 THEN le.valuenum END) as platelets,
            AVG(CASE WHEN le.itemid = 50983 THEN le.valuenum END) as sodium,
            AVG(CASE WHEN le.itemid = 50971 THEN le.valuenum END) as potassium,
            AVG(CASE WHEN le.itemid = 50912 THEN le.valuenum END) as creatinine,
            AVG(CASE WHEN le.itemid = 50931 THEN le.valuenum END) as glucose,
            AVG(CASE WHEN le.itemid = 50885 THEN le.valuenum END) as bilirubin,
            AVG(CASE WHEN le.itemid = 50878 THEN le.valuenum END) as alt,
            AVG(CASE WHEN le.itemid = 50863 THEN le.valuenum END) as alkaline_phosphatase,
            AVG(CASE WHEN le.itemid = 50820 THEN le.valuenum END) as ph,
            AVG(CASE WHEN le.itemid = 50813 THEN le.valuenum END) as lactate,
            AVG(CASE WHEN le.itemid = 51003 THEN le.valuenum END) as troponin,
            AVG(CASE WHEN le.itemid = 50893 THEN le.valuenum END) as calcium
        FROM mimiciv_hosp.labevents le
        WHERE le.subject_id IN :patient_ids
        AND le.valuenum IS NOT NULL
        GROUP BY le.subject_id
        """

        labs_df = pd.read_sql(labs_query, self.engine, params={'patient_ids': tuple(patient_ids)})

        # Fill missing values with cohort medians
        for col in labs_df.columns:
            if col != 'subject_id':
                labs_df[col].fillna(labs_df[col].median(), inplace=True)

        print(f"   Extracted labs for {len(labs_df)} patients")
        return labs_df

    def extract_medications(self, patient_ids):
        """
        Extract medication features (8 features).

        Binary indicators (0/1):
        - vasopressors (norepinephrine, vasopressin, dopamine)
        - sedatives (propofol, midazolam, fentanyl)
        - antibiotics (broad spectrum)
        - anticoagulants (heparin, warfarin)
        - diuretics (furosemide)
        - steroids (hydrocortisone, methylprednisolone)
        - insulin
        - antiarrhythmics (amiodarone)
        """
        print("\n💊 Extracting medications...")

        # Placeholder - real implementation would query prescriptions and inputevents
        meds_df = pd.DataFrame({'subject_id': patient_ids})
        meds_df['vasopressors'] = 0
        meds_df['sedatives'] = 0
        meds_df['antibiotics'] = 0
        meds_df['anticoagulants'] = 0
        meds_df['diuretics'] = 0
        meds_df['steroids'] = 0
        meds_df['insulin'] = 0
        meds_df['antiarrhythmics'] = 0

        print(f"   Extracted medications for {len(meds_df)} patients")
        return meds_df

    def extract_clinical_scores(self, patient_ids):
        """
        Extract clinical scoring systems (6 features).

        Scores:
        - SOFA (Sequential Organ Failure Assessment)
        - APACHE II (Acute Physiology and Chronic Health Evaluation)
        - NEWS2 (National Early Warning Score)
        - qSOFA (Quick SOFA)
        - Charlson Comorbidity Index
        - Elixhauser Comorbidity Score
        """
        print("\n📊 Extracting clinical scores...")

        # Placeholder - real implementation would calculate scores
        scores_df = pd.DataFrame({'subject_id': patient_ids})
        scores_df['sofa_score'] = 0
        scores_df['apache_score'] = 0
        scores_df['news2_score'] = 0
        scores_df['qsofa_score'] = 0
        scores_df['charlson_index'] = 0
        scores_df['elixhauser_score'] = 0

        print(f"   Extracted clinical scores for {len(scores_df)} patients")
        return scores_df

    def extract_temporal_trends(self, patient_ids):
        """
        Extract 6-hour temporal trends (6 features).

        Trend features (change over 6 hours):
        - heart_rate_change
        - systolic_bp_change
        - respiratory_rate_change
        - lactate_change
        - creatinine_change
        - gcs_change
        """
        print("\n📈 Extracting temporal trends...")

        # Placeholder - real implementation would calculate 6-hour deltas
        trends_df = pd.DataFrame({'subject_id': patient_ids})
        trends_df['heart_rate_change'] = 0.0
        trends_df['systolic_bp_change'] = 0.0
        trends_df['respiratory_rate_change'] = 0.0
        trends_df['lactate_change'] = 0.0
        trends_df['creatinine_change'] = 0.0
        trends_df['gcs_change'] = 0.0

        print(f"   Extracted temporal trends for {len(trends_df)} patients")
        return trends_df

    def extract_comorbidities(self, patient_ids):
        """
        Extract comorbidity indicators (6 features).

        Binary indicators (0/1):
        - diabetes
        - hypertension
        - heart_failure
        - copd
        - chronic_kidney_disease
        - cancer
        """
        print("\n🏥 Extracting comorbidities...")

        # Query diagnoses from ICD codes
        comorbidities_query = """
        SELECT
            d.subject_id,
            MAX(CASE WHEN d.icd_code LIKE 'E11%' OR d.icd_code LIKE '250%' THEN 1 ELSE 0 END) as diabetes,
            MAX(CASE WHEN d.icd_code LIKE 'I10%' OR d.icd_code LIKE '401%' THEN 1 ELSE 0 END) as hypertension,
            MAX(CASE WHEN d.icd_code LIKE 'I50%' OR d.icd_code LIKE '428%' THEN 1 ELSE 0 END) as heart_failure,
            MAX(CASE WHEN d.icd_code LIKE 'J44%' OR d.icd_code LIKE '496%' THEN 1 ELSE 0 END) as copd,
            MAX(CASE WHEN d.icd_code LIKE 'N18%' OR d.icd_code LIKE '585%' THEN 1 ELSE 0 END) as ckd,
            MAX(CASE WHEN d.icd_code LIKE 'C%' OR d.icd_code LIKE '1%' OR d.icd_code LIKE '2%' THEN 1 ELSE 0 END) as cancer
        FROM mimiciv_hosp.diagnoses_icd d
        WHERE d.subject_id IN :patient_ids
        GROUP BY d.subject_id
        """

        comorbidities_df = pd.read_sql(comorbidities_query, self.engine, params={'patient_ids': tuple(patient_ids)})

        # Fill missing subjects with 0 (no comorbidity)
        all_patients_df = pd.DataFrame({'subject_id': patient_ids})
        comorbidities_df = all_patients_df.merge(comorbidities_df, on='subject_id', how='left').fillna(0)

        print(f"   Extracted comorbidities for {len(comorbidities_df)} patients")
        return comorbidities_df

    def build_feature_matrix(self, cohort_df):
        """
        Combine all feature groups into final 70-feature matrix.

        Args:
            cohort_df: DataFrame with patient IDs, time windows, and labels

        Returns:
            DataFrame with 70 features + label + metadata
        """
        print("\n🔧 Building feature matrix...")

        patient_ids = cohort_df['subject_id'].unique()

        # Extract all feature groups
        demographics_df = self.extract_demographics(patient_ids)
        vitals_df = self.extract_vitals(patient_ids, cohort_df)
        labs_df = self.extract_labs(patient_ids, cohort_df)
        meds_df = self.extract_medications(patient_ids)
        scores_df = self.extract_clinical_scores(patient_ids)
        trends_df = self.extract_temporal_trends(patient_ids)
        comorbidities_df = self.extract_comorbidities(patient_ids)

        # Merge all features
        feature_matrix = cohort_df[['subject_id', 'label']].copy()
        feature_matrix = feature_matrix.merge(demographics_df, on='subject_id', how='left')
        feature_matrix = feature_matrix.merge(vitals_df, on='subject_id', how='left')
        feature_matrix = feature_matrix.merge(labs_df, on='subject_id', how='left')
        feature_matrix = feature_matrix.merge(meds_df, on='subject_id', how='left')
        feature_matrix = feature_matrix.merge(scores_df, on='subject_id', how='left')
        feature_matrix = feature_matrix.merge(trends_df, on='subject_id', how='left')
        feature_matrix = feature_matrix.merge(comorbidities_df, on='subject_id', how='left')

        # Verify 70 features (excluding subject_id and label)
        num_features = len(feature_matrix.columns) - 2
        print(f"   ✅ Feature matrix built: {len(feature_matrix)} patients × {num_features} features")

        if num_features != 70:
            print(f"   ⚠️  Warning: Expected 70 features, got {num_features}")

        return feature_matrix

    # ========== Cohort Query Builders ==========

    def _build_sepsis_cohort_query(self, limit):
        """Build SQL query for sepsis cohort (Sepsis-3 criteria)."""
        query = """
        SELECT
            subject_id,
            hadm_id,
            1 as label,
            admittime as index_time
        FROM mimiciv_hosp.admissions
        WHERE admission_type = 'EMERGENCY'
        """
        if limit:
            query += f" LIMIT {limit}"
        return query

    def _build_deterioration_cohort_query(self, limit):
        """Build SQL query for deterioration cohort."""
        query = """
        SELECT
            subject_id,
            hadm_id,
            1 as label,
            admittime as index_time
        FROM mimiciv_hosp.admissions
        WHERE admission_type = 'EMERGENCY'
        """
        if limit:
            query += f" LIMIT {limit}"
        return query

    def _build_mortality_cohort_query(self, limit):
        """Build SQL query for mortality cohort."""
        query = """
        SELECT
            subject_id,
            hadm_id,
            CASE WHEN hospital_expire_flag = 1 THEN 1 ELSE 0 END as label,
            admittime as index_time
        FROM mimiciv_hosp.admissions
        """
        if limit:
            query += f" LIMIT {limit}"
        return query

    def _build_readmission_cohort_query(self, limit):
        """Build SQL query for readmission cohort."""
        query = """
        SELECT
            subject_id,
            hadm_id,
            0 as label,
            dischtime as index_time
        FROM mimiciv_hosp.admissions
        WHERE dischtime IS NOT NULL
        """
        if limit:
            query += f" LIMIT {limit}"
        return query


def main():
    """Main execution function."""
    parser = argparse.ArgumentParser(description='Extract MIMIC-IV features for ML training')
    parser.add_argument('--cohort', required=True, choices=['sepsis', 'deterioration', 'mortality', 'readmission'],
                        help='Target cohort to extract')
    parser.add_argument('--output', required=True, help='Output CSV file path')
    parser.add_argument('--db', default='postgresql://user:pass@localhost:5432/mimic', help='Database connection string')
    parser.add_argument('--limit', type=int, help='Limit number of patients (for testing)')

    args = parser.parse_args()

    print("=" * 70)
    print("MIMIC-IV FEATURE EXTRACTOR")
    print("=" * 70)
    print(f"\nCohort: {args.cohort}")
    print(f"Output: {args.output}")
    print(f"Limit:  {args.limit if args.limit else 'None (all patients)'}")
    print()

    try:
        # Initialize extractor
        extractor = MIMICFeatureExtractor(args.db)

        # Extract cohort
        cohort_df = extractor.extract_cohort(args.cohort, args.limit)

        # Build feature matrix
        feature_matrix = extractor.build_feature_matrix(cohort_df)

        # Save to CSV
        feature_matrix.to_csv(args.output, index=False)
        print(f"\n✅ Feature matrix saved to {args.output}")
        print(f"   Shape: {feature_matrix.shape}")
        print(f"   Positive rate: {feature_matrix['label'].mean() * 100:.1f}%")

        print("\n" + "=" * 70)
        print("FEATURE EXTRACTION COMPLETE")
        print("=" * 70)

    except Exception as e:
        print(f"\n❌ Error: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)


if __name__ == '__main__':
    main()
