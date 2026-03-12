#!/bin/bash

# Comprehensive test error fix script
# Fixes all 100 test compilation errors

cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing

echo "🔧 Fixing test compilation errors..."
echo ""

# Fix 1: Replace DoseRecommendation with CalculatedDose
echo "1️⃣ Replacing DoseRecommendation → CalculatedDose..."
find src/test/java -name "*.java" -exec sed -i '' 's/DoseRecommendation/CalculatedDose/g' {} \;
echo "   ✓ Done"

# Fix 2: Replace PediatricDosing with Medication.PediatricDosing
echo "2️⃣ Replacing PediatricDosing → Medication.PediatricDosing..."
find src/test/java -name "*.java" -exec sed -i '' 's/\([^.]\)PediatricDosing/\1Medication.PediatricDosing/g' {} \;
echo "   ✓ Done"

# Fix 3: Replace NeonatalDosing with Medication.NeonatalDosing
echo "3️⃣ Replacing NeonatalDosing → Medication.NeonatalDosing..."
find src/test/java -name "*.java" -exec sed -i '' 's/\([^.]\)NeonatalDosing/\1Medication.NeonatalDosing/g' {} \;
echo "   ✓ Done"

# Fix 4: Add missing indication parameter to calculateDose calls
echo "4️⃣ Fixing calculateDose method calls (2 params → 3 params)..."
# This requires more complex replacement - find calculateDose(med, patient) and add third param
find src/test/java -name "*Test.java" -exec sed -i '' \
  's/calculateDose(\([^,]*\), \([^)]*\))/calculateDose(\1, \2, "test-indication")/g' {} \;
echo "   ✓ Done"

echo ""
echo "✅ Test fixes applied!"
echo ""
echo "📊 Compiling to check remaining errors..."
mvn test-compile 2>&1 | grep -E "(errors|BUILD)" | tail -5
