#!/bin/bash

echo "════════════════════════════════════════════════════════════════════"
echo "  MIMIC-IV Training Pipeline with Personal Credentials"
echo "════════════════════════════════════════════════════════════════════"
echo ""
echo "This will use YOUR Google account (onkarshahi@vaidshala.com)"
echo "instead of the service account."
echo ""
echo "Training will:"
echo "  1. Extract 33,000+ ICU patients from MIMIC-IV"
echo "  2. Build 70-dimensional clinical features"
echo "  3. Train 4 XGBoost models"
echo "  4. Export to ONNX format"
echo "  5. Replace mock models"
echo ""
echo "Estimated time: 60-90 minutes"
echo ""
echo "Press Ctrl+C to cancel, or Enter to start..."
read

# Unset service account credentials to force personal auth
unset GOOGLE_APPLICATION_CREDENTIALS

echo ""
echo "✅ Using personal credentials"
echo ""
echo "Step 1/5: Extracting Patient Cohorts from MIMIC-IV..."
echo "────────────────────────────────────────────────────────────"

python3 scripts/extract_mimic_cohorts.py

if [ $? -ne 0 ]; then
    echo ""
    echo "❌ Cohort extraction failed!"
    exit 1
fi

echo ""
echo "✅ Cohorts extracted successfully!"
echo ""
echo "Step 2/5: Extracting Clinical Features..."
echo "────────────────────────────────────────────────────────────"

python3 scripts/extract_mimic_features.py

if [ $? -ne 0 ]; then
    echo ""
    echo "❌ Feature extraction failed!"
    exit 1
fi

echo ""
echo "✅ Features extracted successfully!"
echo ""
echo "Step 3/5: Training XGBoost Models..."
echo "────────────────────────────────────────────────────────────"

python3 scripts/train_mimic_models.py

if [ $? -ne 0 ]; then
    echo ""
    echo "❌ Model training failed!"
    exit 1
fi

echo ""
echo "════════════════════════════════════════════════════════════════════"
echo "  🎉 TRAINING COMPLETE!"
echo "════════════════════════════════════════════════════════════════════"
echo ""
echo "✅ Real clinical models trained on MIMIC-IV data"
echo "✅ Mock models replaced with production models"
echo ""
echo "📁 New models location:"
echo "   models/sepsis_risk_v2.0.0_mimic.onnx"
echo "   models/deterioration_risk_v2.0.0_mimic.onnx"
echo "   models/mortality_risk_v2.0.0_mimic.onnx"
echo "   models/readmission_risk_v2.0.0_mimic.onnx"
echo ""
echo "📊 Training reports:"
echo "   results/mimic_iv/"
echo ""
echo "Next: Test the models!"
echo "   mvn test -Dtest=CustomPatientMLTest"
echo ""

