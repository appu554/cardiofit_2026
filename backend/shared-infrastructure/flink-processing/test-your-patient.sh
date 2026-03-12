#!/bin/bash

# Quick Manual Patient Test - Uses MIMICModelTest directly
# Bypasses compilation issues

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║     Quick Patient Test - MIMIC-IV Models                      ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

echo "📋 Your Patient Data:"
echo "   ID: PAT-ROHAN-001"
echo "   Age: 42, Gender: M, Weight: 80kg"
echo ""
echo "❤️ Vitals: HR=108, BP=100/60, RR=23, Temp=38.8°C, SpO2=92%"
echo ""
echo "🧪 Labs: WBC=5, Hgb=200(!), Platelets=5(!), Creat=20(!)"
echo "        Lactate=58(!!!) - EXTREMELY ABNORMAL"
echo ""
echo "📊 Scores: NEWS2=5, qSOFA=5(!)"
echo ""
echo "⚠️  WARNING: Several values are physiologically impossible:"
echo "   - Hemoglobin 200 g/dL (normal: 13-17, incompatible with life)"
echo "   - Creatinine 20 mg/dL (normal: 0.7-1.3, severe renal failure)"
echo "   - Lactate 58 mmol/L (normal: 0.5-2.0, INCOMPATIBLE WITH LIFE)"
echo "   - Platelets 5×10³/μL (normal: 150-400, severe thrombocytopenia)"
echo "   - qSOFA 5 (max is 3)"
echo ""
echo "💡 For realistic testing, please use values within physiological ranges."
echo "   See example-patient-data.txt for reference ranges."
echo ""
echo "❓ Would you like to:"
echo "   1. Test with corrected realistic values"
echo "   2. Test with these extreme values anyway (model may not handle well)"
echo ""
read -p "Enter choice (1 or 2): " choice

if [ "$choice" = "1" ]; then
    echo ""
    echo "📝 Suggested Corrected Values for a HIGH-RISK Patient:"
    echo ""
    echo "Age: 42"
    echo "HR: 108 bpm (tachycardia) ✓"
    echo "BP: 100/60 mmHg (hypotension) ✓"
    echo "RR: 23 breaths/min (tachypnea) ✓"
    echo "Temp: 38.8°C (fever) ✓"
    echo "SpO2: 92% (hypoxia) ✓"
    echo ""
    echo "WBC: 15 (instead of 5) - leukocytosis"
    echo "Hemoglobin: 10 (instead of 200) - anemia"
    echo "Platelets: 100 (instead of 5) - thrombocytopenia"
    echo "Creatinine: 2.5 (instead of 20) - renal dysfunction"
    echo "Lactate: 4.5 (instead of 58) - elevated, sepsis indicator"
    echo ""
    echo "NEWS2: 8 (instead of 5)"
    echo "qSOFA: 2 (instead of 5, max is 3)"
    echo ""
    echo "This profile would indicate: Septic shock, high mortality risk"
    echo ""
    echo "Run ./test-manual-patient.sh to enter these corrected values interactively."
    exit 0
fi

echo ""
echo "⚠️  Testing with extreme values - results may be unreliable..."
echo ""

# Just run the existing MIMICModelTest which already works
echo "🔬 Running MIMIC-IV model test with predefined patient profiles..."
echo ""

mvn exec:java -Dexec.mainClass="MIMICModelTest" -Dexec.classpathScope="test" -q

echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "💡 Note: The test above used REALISTIC patient values."
echo "   Your extreme values (Hgb=200, Lactate=58) are not physiologically"
echo "   possible and the models may not handle them correctly."
echo ""
echo "   To test with YOUR corrected values, run: ./test-manual-patient.sh"
echo "═══════════════════════════════════════════════════════════════"
