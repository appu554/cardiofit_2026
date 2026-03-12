#!/usr/bin/env python3
"""
Download and analyze NidaanKosha-100k dataset from HuggingFace.

This script downloads the Indian clinical dataset and analyzes its structure
to identify fields that can map to the 37 MIMIC-IV features.

NidaanKosha (निदान कोष) = "Diagnosis Repository" in Hindi
Dataset: https://huggingface.co/datasets/ekacare/NidaanKosha-100k-V1.0
"""

import json
import requests
import pandas as pd
from typing import Dict, List, Any
from pathlib import Path

# HuggingFace API endpoint
DATASET_API = "https://datasets-server.huggingface.co/rows"
DATASET_NAME = "ekacare/NidaanKosha-100k-V1.0"
CONFIG = "default"
SPLIT = "train"

# Output directory
OUTPUT_DIR = Path(__file__).parent / "indian_datasets"
OUTPUT_DIR.mkdir(exist_ok=True)


def fetch_dataset_sample(offset: int = 0, length: int = 100) -> Dict[str, Any]:
    """
    Fetch a sample of the NidaanKosha dataset from HuggingFace.

    Args:
        offset: Starting index
        length: Number of records to fetch

    Returns:
        JSON response from HuggingFace API
    """
    url = f"{DATASET_API}?dataset={DATASET_NAME}&config={CONFIG}&split={SPLIT}&offset={offset}&length={length}"

    print(f"📥 Fetching records {offset} to {offset + length}...")
    response = requests.get(url)
    response.raise_for_status()

    return response.json()


def analyze_dataset_structure(data: Dict[str, Any]) -> None:
    """Analyze the structure of the NidaanKosha dataset."""

    print("\n" + "="*80)
    print("📊 NIDAANKOSHA-100K DATASET ANALYSIS")
    print("="*80)

    # Extract rows
    rows = data.get("rows", [])
    print(f"\n✅ Fetched {len(rows)} records")

    if not rows:
        print("⚠️ No data returned from API")
        return

    # Get column names from first row
    first_row = rows[0].get("row", {})
    columns = list(first_row.keys())

    print(f"\n📋 Dataset has {len(columns)} columns:")
    for i, col in enumerate(columns, 1):
        print(f"   {i}. {col}")

    # Show sample data
    print("\n📝 Sample Record (first row):")
    print(json.dumps(first_row, indent=2, ensure_ascii=False))

    # Convert to DataFrame for analysis
    df = pd.DataFrame([row["row"] for row in rows])

    print("\n📊 Dataset Statistics:")
    print(df.describe(include='all'))

    print("\n🔍 Column Data Types:")
    print(df.dtypes)

    print("\n📈 Missing Value Analysis:")
    missing = df.isnull().sum()
    missing_pct = (missing / len(df) * 100).round(2)
    missing_df = pd.DataFrame({
        'Missing Count': missing,
        'Missing %': missing_pct
    })
    print(missing_df[missing_df['Missing Count'] > 0])

    return df


def map_to_mimic_features(df: pd.DataFrame) -> Dict[str, str]:
    """
    Map NidaanKosha fields to the 37 MIMIC-IV features.

    MIMIC-IV Features:
    - Demographics (2): age, gender_male
    - Vital Signs (15): HR, RR, Temp, BP (mean/min/max/std), SpO2
    - Lab Values (12): WBC, Hgb, Platelets, Creatinine, BUN, Glucose, Na, K, Lactate, etc.
    - Clinical Scores (8): SOFA total/components, GCS, NEWS2, qSOFA

    Returns:
        Dictionary mapping MIMIC-IV features to NidaanKosha columns
    """

    print("\n" + "="*80)
    print("🗺️  FEATURE MAPPING: NIDAANKOSHA → MIMIC-IV")
    print("="*80)

    # Get all column names
    nidaan_columns = df.columns.tolist()

    print("\n📋 Available NidaanKosha Columns:")
    for col in nidaan_columns:
        print(f"   - {col}")

    # Feature mapping dictionary
    feature_map = {}

    # Demographics mapping
    print("\n👤 DEMOGRAPHICS MAPPING:")
    if 'age' in nidaan_columns:
        feature_map['age'] = 'age'
        print(f"   ✅ age → age")

    if 'gender' in nidaan_columns or 'sex' in nidaan_columns:
        gender_col = 'gender' if 'gender' in nidaan_columns else 'sex'
        feature_map['gender_male'] = gender_col
        print(f"   ✅ gender_male → {gender_col}")

    # Vital Signs mapping
    print("\n🩺 VITAL SIGNS MAPPING:")
    vital_mappings = {
        'heart_rate': ['heart_rate', 'hr', 'pulse', 'pulse_rate'],
        'respiratory_rate': ['respiratory_rate', 'rr', 'respiration_rate'],
        'temperature': ['temperature', 'temp', 'body_temperature'],
        'systolic_bp': ['sbp', 'systolic_bp', 'systolic_blood_pressure'],
        'diastolic_bp': ['dbp', 'diastolic_bp', 'diastolic_blood_pressure'],
        'spo2': ['spo2', 'oxygen_saturation', 'o2_saturation']
    }

    for mimic_feature, possible_names in vital_mappings.items():
        for possible_name in possible_names:
            if possible_name in nidaan_columns:
                feature_map[mimic_feature] = possible_name
                print(f"   ✅ {mimic_feature} → {possible_name}")
                break

    # Lab Values mapping
    print("\n🧪 LAB VALUES MAPPING:")
    lab_mappings = {
        'wbc': ['wbc', 'white_blood_cells', 'total_leucocyte_count', 'tlc'],
        'hemoglobin': ['hemoglobin', 'hgb', 'hb'],
        'platelets': ['platelets', 'platelet_count'],
        'creatinine': ['creatinine', 'serum_creatinine'],
        'bun': ['bun', 'blood_urea_nitrogen', 'urea'],
        'glucose': ['glucose', 'blood_glucose', 'blood_sugar', 'fbs', 'rbs'],
        'sodium': ['sodium', 'na', 'serum_sodium'],
        'potassium': ['potassium', 'k', 'serum_potassium'],
        'lactate': ['lactate', 'serum_lactate']
    }

    for mimic_feature, possible_names in lab_mappings.items():
        for possible_name in possible_names:
            if possible_name in nidaan_columns:
                feature_map[mimic_feature] = possible_name
                print(f"   ✅ {mimic_feature} → {possible_name}")
                break

    # Clinical Scores mapping
    print("\n📊 CLINICAL SCORES MAPPING:")
    score_mappings = {
        'sofa_score': ['sofa', 'sofa_score'],
        'gcs': ['gcs', 'glasgow_coma_scale'],
        'news2': ['news2', 'news_score'],
        'qsofa': ['qsofa', 'quick_sofa']
    }

    for mimic_feature, possible_names in score_mappings.items():
        for possible_name in possible_names:
            if possible_name in nidaan_columns:
                feature_map[mimic_feature] = possible_name
                print(f"   ✅ {mimic_feature} → {possible_name}")
                break

    # Summary
    print(f"\n📈 MAPPING SUMMARY:")
    print(f"   Total MIMIC-IV features: 37")
    print(f"   Mapped features: {len(feature_map)}")
    print(f"   Missing features: {37 - len(feature_map)}")
    print(f"   Coverage: {len(feature_map) / 37 * 100:.1f}%")

    # Missing features
    all_mimic_features = [
        'age', 'gender_male',
        'heart_rate_mean', 'heart_rate_min', 'heart_rate_max', 'heart_rate_std',
        'respiratory_rate_mean', 'respiratory_rate_max',
        'temperature_mean', 'temperature_max',
        'sbp_mean', 'sbp_min', 'dbp_mean',
        'map_mean', 'map_min',
        'spo2_mean', 'spo2_min',
        'wbc', 'hemoglobin', 'platelets', 'creatinine', 'bun', 'glucose',
        'sodium', 'potassium', 'lactate',
        'sofa_total', 'sofa_cardiovascular', 'sofa_respiratory',
        'gcs_total', 'gcs_eye', 'gcs_verbal', 'gcs_motor',
        'news2_score', 'qsofa_score'
    ]

    missing_features = [f for f in all_mimic_features if f not in feature_map]
    if missing_features:
        print(f"\n⚠️  MISSING FEATURES (will use imputation):")
        for feature in missing_features:
            print(f"   - {feature}")

    return feature_map


def save_analysis_results(df: pd.DataFrame, feature_map: Dict[str, str]) -> None:
    """Save the analysis results to files."""

    # Save sample data
    sample_file = OUTPUT_DIR / "nidaan_kosha_sample.csv"
    df.to_csv(sample_file, index=False)
    print(f"\n💾 Saved sample data to: {sample_file}")

    # Save feature mapping
    mapping_file = OUTPUT_DIR / "feature_mapping.json"
    with open(mapping_file, 'w') as f:
        json.dump(feature_map, f, indent=2)
    print(f"💾 Saved feature mapping to: {mapping_file}")

    # Save analysis report
    report_file = OUTPUT_DIR / "nidaan_kosha_analysis.md"
    with open(report_file, 'w') as f:
        f.write("# NidaanKosha-100k Dataset Analysis\n\n")
        f.write("**Dataset**: ekacare/NidaanKosha-100k-V1.0 (HuggingFace)\n")
        f.write("**Purpose**: Indian clinical cases for fine-tuning MIMIC-IV models\n\n")

        f.write("## Dataset Overview\n\n")
        f.write(f"- Total records analyzed: {len(df)}\n")
        f.write(f"- Total columns: {len(df.columns)}\n")
        f.write(f"- Mapped to MIMIC-IV: {len(feature_map)}/37 features ({len(feature_map)/37*100:.1f}%)\n\n")

        f.write("## Available Columns\n\n")
        for col in df.columns:
            f.write(f"- `{col}`\n")

        f.write("\n## Feature Mapping to MIMIC-IV\n\n")
        f.write("| MIMIC-IV Feature | NidaanKosha Column |\n")
        f.write("|-----------------|--------------------|\n")
        for mimic_feat, nidaan_col in sorted(feature_map.items()):
            f.write(f"| {mimic_feat} | {nidaan_col} |\n")

        f.write("\n## Next Steps\n\n")
        f.write("1. Download full dataset (100k records)\n")
        f.write("2. Implement feature extraction pipeline\n")
        f.write("3. Handle missing features with imputation\n")
        f.write("4. Generate training/validation/test splits\n")
        f.write("5. Fine-tune MIMIC-IV models on Indian data\n")

    print(f"💾 Saved analysis report to: {report_file}")


def main():
    """Main execution function."""

    print("🚀 Starting NidaanKosha-100k Dataset Analysis...")

    try:
        # Fetch sample data
        data = fetch_dataset_sample(offset=0, length=100)

        # Analyze structure
        df = analyze_dataset_structure(data)

        if df is not None:
            # Map to MIMIC-IV features
            feature_map = map_to_mimic_features(df)

            # Save results
            save_analysis_results(df, feature_map)

            print("\n✅ Analysis complete!")
            print(f"\n📂 Output directory: {OUTPUT_DIR}")
            print(f"   - nidaan_kosha_sample.csv (sample data)")
            print(f"   - feature_mapping.json (MIMIC-IV mapping)")
            print(f"   - nidaan_kosha_analysis.md (full report)")

    except Exception as e:
        print(f"\n❌ Error: {e}")
        import traceback
        traceback.print_exc()


if __name__ == "__main__":
    main()
