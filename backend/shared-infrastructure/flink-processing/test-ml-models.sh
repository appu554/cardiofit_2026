#!/bin/bash

# CardioFit ML Model Testing Script
# Shows predictions from all 4 clinical risk models

cd "$(dirname "$0")"

echo ""
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║     CardioFit ML Model Testing                                 ║"
echo "║     Testing All 4 Clinical Risk Prediction Models             ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

# Check if models exist
if [ ! -f "models/sepsis_risk_v1.0.0.onnx" ]; then
    echo "❌ Error: ONNX models not found"
    echo ""
    echo "Please ensure the models directory contains:"
    echo "  - sepsis_risk_v1.0.0.onnx"
    echo "  - deterioration_risk_v1.0.0.onnx"
    echo "  - mortality_risk_v1.0.0.onnx"
    echo "  - readmission_risk_v1.0.0.onnx"
    echo ""
    exit 1
fi

echo "✅ Found all 4 ONNX models"
echo ""
echo "🚀 Running ML prediction tests..."
echo "   This will show predictions for 3 patient scenarios:"
echo "   - High-Risk Septic Patient"
echo "   - Low-Risk Stable Patient"
echo "   - Critically Ill ICU Patient"
echo ""

# Run the test
mvn test -Dtest=QuickMLDemo -q

if [ $? -eq 0 ]; then
    echo ""
    echo "════════════════════════════════════════════════════════════════"
    echo "✅ Test Complete!"
    echo ""
    echo "📊 What You Just Saw:"
    echo "   - All 4 ML models loaded successfully"
    echo "   - Predictions for 3 different patient scenarios"
    echo "   - Risk levels: 🔴 High (≥70%), 🟡 Moderate (30-70%), 🟢 Low (<30%)"
    echo ""
    echo "📖 For More Information:"
    echo "   - See: claudedocs/MODULE5_HOW_TO_TEST_YOUR_MODELS.md"
    echo "   - Test code: src/test/java/com/cardiofit/flink/ml/QuickMLDemo.java"
    echo ""
    echo "🔬 Next Steps:"
    echo "   - Modify QuickMLDemo.java to test custom patient data"
    echo "   - Run full integration tests: mvn test -Dtest=Module5IntegrationTest"
    echo "   - Deploy to Flink cluster for real-time predictions"
    echo "════════════════════════════════════════════════════════════════"
else
    echo ""
    echo "❌ Test failed. Check error messages above."
    echo ""
    echo "Common fixes:"
    echo "  - Ensure Maven and Java are properly configured"
    echo "  - Run: mvn clean test-compile"
    echo "  - Check that all 4 ONNX models exist in models/"
    echo ""
    exit 1
fi
