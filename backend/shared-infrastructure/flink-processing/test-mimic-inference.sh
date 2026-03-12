#!/bin/bash

# Test MIMIC-IV ML Inference with Real Patient Data
# This script runs inference tests to validate model outputs

set -e  # Exit on error

echo "========================================="
echo "MIMIC-IV ML Inference Testing"
echo "========================================="
echo ""

# Check if models directory exists
if [ ! -d "models" ]; then
    echo "⚠️  WARNING: models/ directory not found"
    echo "   ONNX models must be present for inference testing"
    echo "   Expected files:"
    echo "   - models/sepsis_risk_v2.0.0_mimic.onnx"
    echo "   - models/deterioration_risk_v2.0.0_mimic.onnx"
    echo "   - models/mortality_risk_v2.0.0_mimic.onnx"
    echo ""
    echo "   Without models, tests will validate data preparation only"
    echo ""
fi

echo "Step 1: Compile test classes"
echo "----------------------------"
mvn test-compile -q
if [ $? -eq 0 ]; then
    echo "✅ Test compilation successful"
else
    echo "❌ Test compilation failed"
    exit 1
fi
echo ""

echo "Step 2: Run adapter and feature extraction tests"
echo "------------------------------------------------"
mvn test -Dtest=RealPatientDataMLTest -q
TEST_RESULT=$?
echo ""

if [ $TEST_RESULT -eq 0 ]; then
    echo "========================================="
    echo "✅ ALL TESTS PASSED"
    echo "========================================="
    echo ""
    echo "Test Results Summary:"
    echo "-------------------"
    echo "✅ Adapter conversion working"
    echo "✅ Feature extraction producing 37 MIMIC-IV features"
    echo "✅ Low-risk patient data prepared"
    echo "✅ High-risk patient data prepared"
    echo "✅ Moderate-risk patient data prepared"
    echo ""
    echo "Next Steps:"
    echo "----------"
    echo "1. If ONNX models are available:"
    echo "   - Models will automatically be loaded during inference"
    echo "   - Check logs for actual risk predictions"
    echo ""
    echo "2. To test with ONNX models:"
    echo "   - Ensure models are in models/ directory"
    echo "   - Run: mvn test -Dtest=MIMICModelTest"
    echo ""
    echo "3. To test full Module 5 pipeline:"
    echo "   - Start Module 2 (produces EnrichedPatientContext)"
    echo "   - Start Module 5 with MIMIC-IV integration"
    echo "   - Verify predictions in Kafka topics"
else
    echo "========================================="
    echo "❌ TESTS FAILED"
    echo "========================================="
    echo ""
    echo "Check test output above for error details"
    exit 1
fi
