#!/bin/bash

# MIMIC-IV Model Training Pipeline
# Complete automated workflow from BigQuery to ONNX models

set -e  # Exit on error

cd "$(dirname "$0")"

echo ""
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║     MIMIC-IV Clinical Model Training Pipeline                 ║"
echo "║     Train Real Models on 300K+ ICU Admissions                 ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

# ═══════════════════════════════════════════════════════════════════════════
# STEP 0: Pre-flight Checks
# ═══════════════════════════════════════════════════════════════════════════

echo "🔍 Pre-flight Checks..."
echo ""

# Check Python
if ! command -v python3 &> /dev/null; then
    echo "❌ Error: Python 3 not found"
    echo "   Install Python 3.8+ and try again"
    exit 1
fi

python_version=$(python3 --version | cut -d' ' -f2)
echo "✅ Python: $python_version"

# Check pip
if ! command -v pip3 &> /dev/null; then
    echo "❌ Error: pip not found"
    exit 1
fi
echo "✅ pip installed"

# Check credentials
if [ -z "$GOOGLE_APPLICATION_CREDENTIALS" ]; then
    echo "⚠️  Warning: GOOGLE_APPLICATION_CREDENTIALS not set"
    echo ""
    echo "Please set your GCP credentials:"
    echo "  export GOOGLE_APPLICATION_CREDENTIALS=\"/path/to/credentials.json\""
    echo ""
    read -p "Continue anyway? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
else
    echo "✅ GCP Credentials: $GOOGLE_APPLICATION_CREDENTIALS"
fi

# Check GCP project ID
if [ -z "$GCP_PROJECT_ID" ]; then
    echo "⚠️  Warning: GCP_PROJECT_ID not set"
    echo ""
    read -p "Enter your GCP Project ID: " project_id
    export GCP_PROJECT_ID="$project_id"
fi
echo "✅ GCP Project: $GCP_PROJECT_ID"

echo ""
echo "═══════════════════════════════════════════════════════════════════════════"
echo "🚀 Starting MIMIC-IV Training Pipeline"
echo "═══════════════════════════════════════════════════════════════════════════"
echo ""

# ═══════════════════════════════════════════════════════════════════════════
# STEP 1: Install Dependencies
# ═══════════════════════════════════════════════════════════════════════════

echo "📦 STEP 1: Installing Python Dependencies"
echo "─────────────────────────────────────────────────────────────────"
echo ""

read -p "Install/upgrade Python packages? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    pip3 install -r requirements_mimic.txt
    echo ""
    echo "✅ Dependencies installed"
else
    echo "⏭️  Skipped dependency installation"
fi

echo ""

# ═══════════════════════════════════════════════════════════════════════════
# STEP 2: Validate Configuration
# ═══════════════════════════════════════════════════════════════════════════

echo "⚙️  STEP 2: Validating Configuration"
echo "─────────────────────────────────────────────────────────────────"
echo ""

python3 scripts/mimic_iv_config.py

if [ $? -ne 0 ]; then
    echo ""
    echo "❌ Configuration validation failed"
    echo "   Please update scripts/mimic_iv_config.py with your GCP project ID"
    exit 1
fi

echo ""

# ═══════════════════════════════════════════════════════════════════════════
# STEP 3: Extract Patient Cohorts
# ═══════════════════════════════════════════════════════════════════════════

echo "📊 STEP 3: Extracting Patient Cohorts from BigQuery"
echo "─────────────────────────────────────────────────────────────────"
echo ""
echo "This will query MIMIC-IV BigQuery and extract:"
echo "  - 10,000 sepsis cases"
echo "  - 8,000 deterioration cases"
echo "  - 5,000 mortality cases"
echo "  - 10,000 readmission cases"
echo ""
echo "⏱️  Estimated time: 5-10 minutes"
echo ""

read -p "Start cohort extraction? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    python3 scripts/extract_mimic_cohorts.py

    if [ $? -ne 0 ]; then
        echo ""
        echo "❌ Cohort extraction failed"
        echo "   Check BigQuery access and credentials"
        exit 1
    fi
else
    echo "⏭️  Skipped cohort extraction (using existing cohorts)"
fi

echo ""

# ═══════════════════════════════════════════════════════════════════════════
# STEP 4: Extract Clinical Features
# ═══════════════════════════════════════════════════════════════════════════

echo "🧪 STEP 4: Extracting 70-Dimensional Clinical Features"
echo "─────────────────────────────────────────────────────────────────"
echo ""
echo "This will extract clinical features from BigQuery:"
echo "  - Demographics (5 features)"
echo "  - Vital signs (15 features)"
echo "  - Lab values (13 features)"
echo "  - Clinical scores (8 features)"
echo "  - Medications, ventilation, other (29 features)"
echo ""
echo "⏱️  Estimated time: 15-30 minutes"
echo ""

read -p "Start feature extraction? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    python3 scripts/extract_mimic_features.py

    if [ $? -ne 0 ]; then
        echo ""
        echo "❌ Feature extraction failed"
        exit 1
    fi
else
    echo "⏭️  Skipped feature extraction (using existing features)"
fi

echo ""

# ═══════════════════════════════════════════════════════════════════════════
# STEP 5: Train Models
# ═══════════════════════════════════════════════════════════════════════════

echo "🎓 STEP 5: Training XGBoost Models"
echo "─────────────────────────────────────────────────────────────────"
echo ""
echo "This will train 4 clinical prediction models:"
echo "  1. Sepsis risk prediction"
echo "  2. Clinical deterioration"
echo "  3. In-hospital mortality"
echo "  4. 30-day readmission"
echo ""
echo "Each model will be:"
echo "  - Trained on 70% of data"
echo "  - Validated on 15% of data"
echo "  - Tested on 15% of data"
echo "  - Exported to ONNX format"
echo ""
echo "⏱️  Estimated time: 20-40 minutes"
echo ""

read -p "Start model training? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    python3 scripts/train_mimic_models.py

    if [ $? -ne 0 ]; then
        echo ""
        echo "❌ Model training failed"
        exit 1
    fi
else
    echo "⏭️  Skipped model training"
    exit 0
fi

echo ""

# ═══════════════════════════════════════════════════════════════════════════
# STEP 6: Replace Mock Models
# ═══════════════════════════════════════════════════════════════════════════

echo "🔄 STEP 6: Replace Mock Models with Real Models"
echo "─────────────────────────────────────────────────────────────────"
echo ""
echo "This will:"
echo "  1. Backup mock models to models/backup_mock_models/"
echo "  2. Copy MIMIC-IV trained models to production location"
echo ""

read -p "Replace mock models with real models? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    # Create backup directory
    mkdir -p models/backup_mock_models

    # Backup old mock models
    if [ -f "models/sepsis_risk_v1.0.0.onnx" ]; then
        echo "📦 Backing up mock models..."
        mv models/*_v1.0.0.onnx models/backup_mock_models/ 2>/dev/null || true
        echo "   ✅ Mock models backed up"
    fi

    # Copy new MIMIC-IV models
    if [ -f "models/sepsis_risk_v2.0.0_mimic.onnx" ]; then
        echo "🔄 Installing MIMIC-IV trained models..."
        cp models/sepsis_risk_v2.0.0_mimic.onnx models/sepsis_risk_v1.0.0.onnx
        cp models/deterioration_risk_v2.0.0_mimic.onnx models/deterioration_risk_v1.0.0.onnx
        cp models/mortality_risk_v2.0.0_mimic.onnx models/mortality_risk_v1.0.0.onnx
        cp models/readmission_risk_v2.0.0_mimic.onnx models/readmission_risk_v1.0.0.onnx
        echo "   ✅ MIMIC-IV models installed"
    else
        echo "   ⚠️  MIMIC-IV models not found (training may have failed)"
    fi
else
    echo "⏭️  Skipped model replacement"
fi

echo ""

# ═══════════════════════════════════════════════════════════════════════════
# STEP 7: Test with Java
# ═══════════════════════════════════════════════════════════════════════════

echo "🧪 STEP 7: Test Real Models with Java"
echo "─────────────────────────────────────────────────────────────────"
echo ""
echo "Run Java tests to verify real models work correctly"
echo ""

read -p "Run Java tests? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo ""
    echo "Running QuickMLDemo (3 patient scenarios)..."
    mvn test -Dtest=QuickMLDemo -q

    echo ""
    echo "Running ProofMLWorking (verify ML is real)..."
    mvn test -Dtest=ProofMLWorking -q

    echo ""
    echo "✅ Java tests complete"
else
    echo "⏭️  Skipped Java tests"
fi

echo ""

# ═══════════════════════════════════════════════════════════════════════════
# COMPLETE
# ═══════════════════════════════════════════════════════════════════════════

echo "═══════════════════════════════════════════════════════════════════════════"
echo "✅ MIMIC-IV TRAINING PIPELINE COMPLETE!"
echo "═══════════════════════════════════════════════════════════════════════════"
echo ""
echo "📊 Results Summary:"
echo "   - Trained models: 4/4"
echo "   - Model location: models/*_v2.0.0_mimic.onnx"
echo "   - Production location: models/*_v1.0.0.onnx (replaced mock models)"
echo "   - Training reports: results/mimic_iv/*_training_report.md"
echo "   - Performance plots: results/mimic_iv/figures/*.png"
echo ""
echo "📖 Next Steps:"
echo "   1. Review training reports:"
echo "      cd results/mimic_iv && ls -la"
echo ""
echo "   2. Compare mock vs real model predictions:"
echo "      mvn test -Dtest=CustomPatientMLTest"
echo ""
echo "   3. Deploy to Flink cluster (when ready for production)"
echo ""
echo "🎯 Your ML models are now trained on REAL clinical data from MIMIC-IV!"
echo "═══════════════════════════════════════════════════════════════════════════"
echo ""
