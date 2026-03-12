#!/bin/bash

# Test with Rohan's patient data - CORRECTED to realistic values

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║     Manual Patient Test - Rohan's Patient (Corrected)         ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

echo "📋 PATIENT: PAT-ROHAN-001 (Age 42, Male, 80kg)"
echo ""
echo "❤️ VITAL SIGNS (Concerning):"
echo "   • Heart Rate: 108 bpm (tachycardia)"
echo "   • Blood Pressure: 100/60 mmHg (hypotension)"
echo "   • Respiratory Rate: 23 breaths/min (tachypnea)"
echo "   • Temperature: 38.8°C (fever)"
echo "   • Oxygen Saturation: 92% (hypoxia)"
echo ""
echo "🧪 LABORATORY VALUES (Abnormal):"
echo "   • WBC: 15 × 10³/μL (leukocytosis)"
echo "   • Hemoglobin: 10 g/dL (anemia)"
echo "   • Platelets: 100 × 10³/μL (thrombocytopenia)"
echo "   • Creatinine: 2.5 mg/dL (renal dysfunction)"
echo "   • BUN: 35 mg/dL (elevated)"
echo "   • Glucose: 150 mg/dL (hyperglycemia)"
echo "   • Sodium: 148 mmol/L (hypernatremia)"
echo "   • Potassium: 5.5 mmol/L (hyperkalemia)"
echo "   • Lactate: 4.5 mmol/L (⚠️ SEPSIS INDICATOR)"
echo ""
echo "📊 CLINICAL SCORES:"
echo "   • NEWS2: 8 (HIGH RISK)"
echo "   • qSOFA: 2 (SEPSIS SUSPECTED)"
echo ""
echo "🔬 CLINICAL INTERPRETATION:"
echo "   This profile suggests SEPTIC SHOCK with multi-organ involvement:"
echo "   - Cardiovascular: Hypotension + Tachycardia"
echo "   - Respiratory: Tachypnea + Hypoxia"
echo "   - Renal: Elevated creatinine + BUN"
echo "   - Hematologic: Anemia + Thrombocytopenia"
echo "   - Metabolic: Elevated lactate (tissue hypoperfusion)"
echo ""
echo "⏳ Running MIMIC-IV ML models..."
echo ""

# Create temporary Java test file with Rohan's corrected data
cat > /tmp/RohanPatientTest.java << 'JAVACODE'
import com.cardiofit.flink.adapters.PatientContextAdapter;
import com.cardiofit.flink.ml.ONNXModelContainer;
import com.cardiofit.flink.ml.ModelConfig;
import com.cardiofit.flink.ml.PatientContextSnapshot;
import com.cardiofit.flink.ml.features.MIMICFeatureExtractor;
import com.cardiofit.flink.models.*;
import com.cardiofit.flink.models.MLPrediction;

import java.util.*;

public class RohanPatientTest {
    public static void main(String[] args) {
        System.out.println("════════════════════════════════════════════════════════════════");
        System.out.println("🔮 MIMIC-IV ML RISK PREDICTIONS");
        System.out.println("════════════════════════════════════════════════════════════════");
        System.out.println();

        try {
            // Create Rohan's patient data (corrected realistic values)
            EnrichedPatientContext context = createRohanPatient();

            // Convert to snapshot
            PatientContextAdapter adapter = new PatientContextAdapter();
            PatientContextSnapshot snapshot = adapter.adapt(context);

            // Extract features
            MIMICFeatureExtractor extractor = new MIMICFeatureExtractor();
            float[] features = extractor.extractFeatures(snapshot);

            System.out.println("✅ Feature Extraction Complete: " + features.length + " MIMIC-IV features");
            System.out.println();

            // Load models
            System.out.println("📦 Loading ONNX Models...");

            ModelConfig sepsisConfig = ModelConfig.builder()
                .modelPath("models/sepsis_risk_v2.0.0_mimic.onnx")
                .inputDimension(37)
                .outputDimension(2)
                .predictionThreshold(0.5f)
                .build();

            ONNXModelContainer sepsisModel = ONNXModelContainer.builder()
                .modelId("sepsis_risk_v2")
                .modelName("Sepsis Risk")
                .modelType(ONNXModelContainer.ModelType.SEPSIS_ONSET)
                .modelVersion("2.0.0")
                .inputFeatureNames(MIMICFeatureExtractor.getFeatureNames())
                .outputNames(Arrays.asList("label", "probabilities"))
                .config(sepsisConfig)
                .build();
            sepsisModel.initialize();

            ModelConfig detConfig = ModelConfig.builder()
                .modelPath("models/deterioration_risk_v2.0.0_mimic.onnx")
                .inputDimension(37)
                .outputDimension(2)
                .predictionThreshold(0.5f)
                .build();

            ONNXModelContainer detModel = ONNXModelContainer.builder()
                .modelId("deterioration_risk_v2")
                .modelName("Clinical Deterioration")
                .modelType(ONNXModelContainer.ModelType.CLINICAL_DETERIORATION)
                .modelVersion("2.0.0")
                .inputFeatureNames(MIMICFeatureExtractor.getFeatureNames())
                .outputNames(Arrays.asList("label", "probabilities"))
                .config(detConfig)
                .build();
            detModel.initialize();

            ModelConfig mortalityConfig = ModelConfig.builder()
                .modelPath("models/mortality_risk_v2.0.0_mimic.onnx")
                .inputDimension(37)
                .outputDimension(2)
                .predictionThreshold(0.5f)
                .build();

            ONNXModelContainer mortalityModel = ONNXModelContainer.builder()
                .modelId("mortality_risk_v2")
                .modelName("Mortality Risk")
                .modelType(ONNXModelContainer.ModelType.MORTALITY_PREDICTION)
                .modelVersion("2.0.0")
                .inputFeatureNames(MIMICFeatureExtractor.getFeatureNames())
                .outputNames(Arrays.asList("label", "probabilities"))
                .config(mortalityConfig)
                .build();
            mortalityModel.initialize();

            System.out.println("✅ All 3 models loaded");
            System.out.println();

            // Run predictions
            System.out.println("════════════════════════════════════════════════════════════════");
            System.out.println("🎯 RISK PREDICTIONS FOR PAT-ROHAN-001");
            System.out.println("════════════════════════════════════════════════════════════════");

            MLPrediction sepsisPred = sepsisModel.predict(features);
            double sepsisRisk = sepsisPred.getPredictionScores().get("confidence_score");
            System.out.printf("   🦠 Sepsis Risk          : %.4f (%.2f%%) - %s%n",
                sepsisRisk, sepsisRisk * 100, getRiskLevel((float)sepsisRisk));

            MLPrediction detPred = detModel.predict(features);
            double detRisk = detPred.getPredictionScores().get("confidence_score");
            System.out.printf("   📉 Deterioration Risk   : %.4f (%.2f%%) - %s%n",
                detRisk, detRisk * 100, getRiskLevel((float)detRisk));

            MLPrediction mortalityPred = mortalityModel.predict(features);
            double mortalityRisk = mortalityPred.getPredictionScores().get("confidence_score");
            System.out.printf("   💀 Mortality Risk       : %.4f (%.2f%%) - %s%n",
                mortalityRisk, mortalityRisk * 100, getRiskLevel((float)mortalityRisk));

            System.out.println("════════════════════════════════════════════════════════════════");
            System.out.println();

            // Clinical recommendations
            System.out.println("💡 CLINICAL RECOMMENDATIONS:");
            System.out.println("────────────────────────────────────────────────────────────────");

            double avgRisk = (sepsisRisk + detRisk + mortalityRisk) / 3.0;

            if (avgRisk > 0.8) {
                System.out.println("   🚨 CRITICAL - IMMEDIATE ACTION REQUIRED");
                System.out.println("   • Transfer to ICU immediately");
                System.out.println("   • Initiate sepsis protocol (fluid resuscitation, broad-spectrum antibiotics)");
                System.out.println("   • Continuous hemodynamic monitoring");
                System.out.println("   • Consider vasopressor support");
                System.out.println("   • Urgent nephrology consultation for renal dysfunction");
                System.out.println("   • Serial lactate measurements (target <2.0 mmol/L)");
            } else if (avgRisk > 0.5) {
                System.out.println("   ⚠️ HIGH RISK - Enhanced Monitoring");
                System.out.println("   • Increase vital sign frequency to q1h");
                System.out.println("   • Consider ICU consultation");
                System.out.println("   • Serial lactate and blood cultures");
                System.out.println("   • Review and optimize treatment plan");
            } else {
                System.out.println("   ✅ MODERATE RISK - Standard Enhanced Care");
                System.out.println("   • Continue current monitoring");
                System.out.println("   • Regular vital signs");
                System.out.println("   • Review labs in 4-6 hours");
            }

            System.out.println();
            System.out.println("✅ Analysis Complete!");

        } catch (Exception e) {
            System.err.println("❌ Error: " + e.getMessage());
            e.printStackTrace();
            System.exit(1);
        }
    }

    private static EnrichedPatientContext createRohanPatient() {
        EnrichedPatientContext context = new EnrichedPatientContext();
        context.setPatientId("PAT-ROHAN-001");
        context.setEncounterId("ENC-ROHAN-001");
        context.setEventTime(System.currentTimeMillis());
        context.setEventType("SEPSIS_SCREENING");

        PatientContextState state = new PatientContextState();

        // Demographics
        PatientDemographics demographics = new PatientDemographics();
        demographics.setAge(42);
        demographics.setGender("M");
        demographics.setWeight(80.0);
        state.setDemographics(demographics);

        // Vital Signs (CORRECTED realistic values)
        Map<String, Object> vitals = new HashMap<>();
        vitals.put("heartrate", 108.0);
        vitals.put("systolicbloodpressure", 100.0);
        vitals.put("diastolicbloodpressure", 60.0);
        vitals.put("respiratoryrate", 23.0);
        vitals.put("temperature", 38.8);
        vitals.put("oxygensaturation", 92.0);
        state.setLatestVitals(vitals);

        // Lab Values (CORRECTED realistic values)
        Map<String, LabResult> labs = new HashMap<>();
        labs.put("WBC", createLabResult(15.0, "10^3/uL"));
        labs.put("hemoglobin", createLabResult(10.0, "g/dL"));
        labs.put("platelets", createLabResult(100.0, "10^3/uL"));
        labs.put("creatinine", createLabResult(2.5, "mg/dL"));
        labs.put("bun", createLabResult(35.0, "mg/dL"));
        labs.put("glucose", createLabResult(150.0, "mg/dL"));
        labs.put("sodium", createLabResult(148.0, "mmol/L"));
        labs.put("potassium", createLabResult(5.5, "mmol/L"));
        labs.put("lactate", createLabResult(4.5, "mmol/L"));
        state.setRecentLabs(labs);

        // Clinical Scores
        state.setNews2Score(8);
        state.setQsofaScore(2);
        state.setCombinedAcuityScore(80.0);

        context.setPatientState(state);
        return context;
    }

    private static LabResult createLabResult(double value, String unit) {
        LabResult result = new LabResult();
        result.setValue(value);
        result.setUnit(unit);
        result.setTimestamp(System.currentTimeMillis());
        return result;
    }

    private static String getRiskLevel(float risk) {
        if (risk >= 0.8) return "🚨 CRITICAL RISK";
        if (risk >= 0.5) return "⚠️ HIGH RISK";
        if (risk >= 0.3) return "⚡ MODERATE RISK";
        return "✅ LOW RISK";
    }
}
JAVACODE

# Copy to test directory
cp /tmp/RohanPatientTest.java src/test/java/

# Compile the test class
echo "🔧 Compiling test class..."
mvn test-compile -q

# Run the test
mvn exec:java -Dexec.mainClass="RohanPatientTest" -Dexec.classpathScope="test" -q 2>&1 | grep -v "WARNING" | grep -v "SLF4J"

# Cleanup
rm -f /tmp/RohanPatientTest.java src/test/java/RohanPatientTest.java src/test/java/RohanPatientTest.class

echo ""
echo "════════════════════════════════════════════════════════════════"
echo "📊 TEST COMPLETE"
echo "════════════════════════════════════════════════════════════════"
