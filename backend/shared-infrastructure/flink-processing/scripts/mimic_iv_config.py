#!/usr/bin/env python3
"""
MIMIC-IV BigQuery Configuration
Credentials and connection settings for accessing MIMIC-IV v3.1 in GCP BigQuery.
"""

import os
from pathlib import Path
from google.cloud import bigquery
from google.oauth2 import service_account

# ═══════════════════════════════════════════════════════════════════════════
# GCP CONFIGURATION
# ═══════════════════════════════════════════════════════════════════════════

# GCP Project ID (replace with your project)
GCP_PROJECT_ID = os.getenv("GCP_PROJECT_ID", "sincere-hybrid-477206-h2")

# BigQuery Dataset containing MIMIC-IV
# Using YOUR local copy of MIMIC-IV (copied to your project)
MIMIC_DATASET = "sincere-hybrid-477206-h2.mimiciv_3_1_hosp"  # Hospital data
MIMIC_ICU_DATASET = "sincere-hybrid-477206-h2.mimiciv_3_1_icu"  # ICU data
MIMIC_DERIVED_DATASET = "sincere-hybrid-477206-h2.mimiciv_3_1_derived"  # Derived tables

# Credentials file path
# Download your service account JSON from GCP Console
CREDENTIALS_FILE = os.getenv(
    "GOOGLE_APPLICATION_CREDENTIALS",
    str(Path.home() / ".gcp" / "mimic-iv-credentials.json")
)

# ═══════════════════════════════════════════════════════════════════════════
# MIMIC-IV TABLE REFERENCES
# ═══════════════════════════════════════════════════════════════════════════

MIMIC_TABLES = {
    # Core tables
    "patients": f"{MIMIC_DATASET}.patients",
    "admissions": f"{MIMIC_DATASET}.admissions",
    "icustays": f"{MIMIC_ICU_DATASET}.icustays",

    # Clinical data
    "chartevents": f"{MIMIC_ICU_DATASET}.chartevents",  # Vital signs
    "labevents": f"{MIMIC_DATASET}.labevents",  # Lab results
    "prescriptions": f"{MIMIC_DATASET}.prescriptions",  # Medications
    "procedures_icd": f"{MIMIC_DATASET}.procedures_icd",  # Procedures
    "diagnoses_icd": f"{MIMIC_DATASET}.diagnoses_icd",  # Diagnoses

    # Derived tables (pre-computed features)
    "sepsis3": f"{MIMIC_DERIVED_DATASET}.sepsis3",  # Sepsis-3 criteria
    "sofa": f"{MIMIC_DERIVED_DATASET}.sofa",  # SOFA scores
    "sapsii": f"{MIMIC_DERIVED_DATASET}.sapsii",  # SAPS-II scores
    "first_day_vitalsign": f"{MIMIC_DERIVED_DATASET}.first_day_vitalsign",
    "first_day_lab": f"{MIMIC_DERIVED_DATASET}.first_day_lab",
}

# ═══════════════════════════════════════════════════════════════════════════
# COHORT DEFINITIONS
# ═══════════════════════════════════════════════════════════════════════════

# Sepsis ICD-10 codes (Sepsis-3 definition)
SEPSIS_ICD10_CODES = [
    "A40%",  # Streptococcal sepsis
    "A41%",  # Other sepsis
    "A02.1",  # Salmonella sepsis
    "A22.7",  # Anthrax sepsis
    "A26.7",  # Erysipelothrix sepsis
    "A32.7",  # Listerial sepsis
    "A42.7",  # Actinomycotic sepsis
    "B37.7",  # Candidal sepsis
    "R65.2%",  # Severe sepsis
]

# Clinical criteria for sepsis (Sepsis-3)
SEPSIS_CLINICAL_CRITERIA = {
    "sofa_threshold": 2,  # SOFA score increase ≥2
    "lactate_threshold": 2.0,  # Lactate >2 mmol/L
    "qsofa_threshold": 2,  # qSOFA ≥2
}

# Mortality outcomes
MORTALITY_TYPES = {
    "hospital_death": True,  # Died during hospital stay
    "icu_death": True,  # Died in ICU
    "30day_death": True,  # Died within 30 days
}

# ═══════════════════════════════════════════════════════════════════════════
# FEATURE EXTRACTION CONFIGURATION
# ═══════════════════════════════════════════════════════════════════════════

# Time windows for feature extraction
TIME_WINDOWS = {
    "first_6h": 6,  # First 6 hours of ICU stay
    "first_12h": 12,  # First 12 hours
    "first_24h": 24,  # First 24 hours
}

# Vital signs item IDs from chartevents
VITAL_SIGN_ITEMS = {
    "heart_rate": [220045],  # Heart rate (bpm)
    "respiratory_rate": [220210, 224690],  # Respiratory rate
    "temperature": [223761, 223762],  # Temperature (C/F)
    "sbp": [220050, 220179],  # Systolic BP
    "dbp": [220051, 220180],  # Diastolic BP
    "map": [220052, 220181, 225312],  # Mean arterial pressure
    "spo2": [220277],  # Oxygen saturation
}

# Lab test item IDs from labevents
LAB_TEST_ITEMS = {
    "wbc": [51300, 51301],  # White blood cells
    "hemoglobin": [51222],  # Hemoglobin
    "platelets": [51265],  # Platelets
    "creatinine": [50912],  # Creatinine
    "bun": [51006],  # Blood urea nitrogen
    "glucose": [50809, 50931],  # Glucose
    "sodium": [50824, 50983],  # Sodium
    "potassium": [50822, 50971],  # Potassium
    "lactate": [50813],  # Lactate
    "bilirubin": [50885],  # Bilirubin
}

# ═══════════════════════════════════════════════════════════════════════════
# MODEL TRAINING CONFIGURATION
# ═══════════════════════════════════════════════════════════════════════════

TRAINING_CONFIG = {
    # Data splits
    "train_ratio": 0.70,
    "val_ratio": 0.15,
    "test_ratio": 0.15,

    # Temporal split (for temporal validation)
    "temporal_split_date": "2017-01-01",  # Train before, test after

    # Minimum sample sizes
    "min_positive_samples": {
        "sepsis": 5000,
        "deterioration": 3000,
        "mortality": 2000,
        "readmission": 5000,
    },

    # XGBoost hyperparameters (starting point)
    "xgboost": {
        "n_estimators": 100,
        "max_depth": 6,
        "learning_rate": 0.1,
        "subsample": 0.8,
        "colsample_bytree": 0.8,
        "min_child_weight": 1,
        "gamma": 0,
        "reg_alpha": 0,
        "reg_lambda": 1,
        "random_state": 42,
        "eval_metric": "logloss",
        "n_jobs": -1,
    },

    # Performance thresholds
    "min_auroc": 0.85,
    "min_sensitivity": 0.80,
    "min_specificity": 0.75,
}

# ═══════════════════════════════════════════════════════════════════════════
# OUTPUT CONFIGURATION
# ═══════════════════════════════════════════════════════════════════════════

OUTPUT_DIRS = {
    "data": "data/mimic_iv",
    "models": "models",
    "results": "results/mimic_iv",
    "figures": "results/mimic_iv/figures",
}

# Ensure output directories exist
for dir_path in OUTPUT_DIRS.values():
    Path(dir_path).mkdir(parents=True, exist_ok=True)

# Model output paths
MODEL_OUTPUTS = {
    "sepsis": "models/sepsis_risk_v2.0.0_mimic.onnx",
    "deterioration": "models/deterioration_risk_v2.0.0_mimic.onnx",
    "mortality": "models/mortality_risk_v2.0.0_mimic.onnx",
    "readmission": "models/readmission_risk_v2.0.0_mimic.onnx",
}

# ═══════════════════════════════════════════════════════════════════════════
# VALIDATION
# ═══════════════════════════════════════════════════════════════════════════

def create_bigquery_client():
    """
    Create BigQuery client with flexible authentication.

    Tries (in order):
    1. Service account credentials file (if GOOGLE_APPLICATION_CREDENTIALS set)
    2. Application Default Credentials (personal Google account)

    Returns:
        bigquery.Client: Authenticated BigQuery client
    """
    # Try service account first (production)
    if "GOOGLE_APPLICATION_CREDENTIALS" in os.environ and os.path.exists(CREDENTIALS_FILE):
        print(f"🔐 Using service account: {CREDENTIALS_FILE}")
        credentials = service_account.Credentials.from_service_account_file(
            CREDENTIALS_FILE,
            scopes=["https://www.googleapis.com/auth/bigquery"],
        )
        return bigquery.Client(credentials=credentials, project=GCP_PROJECT_ID)

    # Fall back to personal credentials (development)
    print("🔐 Using Application Default Credentials (your personal Google account)")
    print("   This uses your logged-in Google account, not a service account.")
    return bigquery.Client(project=GCP_PROJECT_ID)


def validate_config():
    """Validate configuration before running pipeline."""
    errors = []

    # Check GCP project ID
    if GCP_PROJECT_ID == "your-gcp-project-id":
        errors.append("GCP_PROJECT_ID not set. Update mimic_iv_config.py")

    # Check credentials (warn if missing, but don't fail - might use personal auth)
    if "GOOGLE_APPLICATION_CREDENTIALS" in os.environ:
        if not os.path.exists(CREDENTIALS_FILE):
            print(f"⚠️  Service account credentials file not found: {CREDENTIALS_FILE}")
            print("   Falling back to Application Default Credentials (personal account)")
    else:
        print("⚠️  GOOGLE_APPLICATION_CREDENTIALS not set")
        print("   Using Application Default Credentials (personal account)")

    if errors:
        raise ValueError("Configuration errors:\n" + "\n".join(f"  - {e}" for e in errors))

    print("✅ Configuration validated successfully")
    return True


if __name__ == "__main__":
    # Test configuration
    print("MIMIC-IV Configuration")
    print("=" * 70)
    print(f"GCP Project: {GCP_PROJECT_ID}")
    print(f"Credentials: {CREDENTIALS_FILE}")
    print(f"MIMIC Dataset: {MIMIC_DATASET}")
    print(f"Tables: {len(MIMIC_TABLES)} configured")
    print()

    try:
        validate_config()
    except ValueError as e:
        print(f"❌ {e}")
        exit(1)
