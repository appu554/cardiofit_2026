#!/bin/bash

# Interactive Manual Patient Data Testing for MIMIC-IV Models
# This script allows you to input patient data and see ML predictions

set -e

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║     Manual Patient Data Testing - MIMIC-IV Models             ║"
echo "║     Enter patient data to see risk predictions                ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

# Collect patient data interactively
echo "📋 PATIENT DEMOGRAPHICS"
echo "----------------------"
read -p "Patient ID (e.g., PT-12345): " PATIENT_ID
read -p "Age (years): " AGE
read -p "Gender (M/F): " GENDER
read -p "Weight (kg): " WEIGHT

echo ""
echo "❤️ VITAL SIGNS"
echo "-------------"
read -p "Heart Rate (bpm): " HEART_RATE
read -p "Systolic Blood Pressure (mmHg): " SBP
read -p "Diastolic Blood Pressure (mmHg): " DBP
read -p "Respiratory Rate (breaths/min): " RESP_RATE
read -p "Temperature (°C): " TEMP
read -p "Oxygen Saturation (%, e.g., 95): " SPO2

echo ""
echo "🧪 LABORATORY VALUES"
echo "-------------------"
read -p "White Blood Cell Count (10^3/uL): " WBC
read -p "Hemoglobin (g/dL): " HGB
read -p "Platelets (10^3/uL): " PLATELETS
read -p "Creatinine (mg/dL): " CREATININE
read -p "BUN (mg/dL): " BUN
read -p "Glucose (mg/dL): " GLUCOSE
read -p "Sodium (mmol/L): " SODIUM
read -p "Potassium (mmol/L): " POTASSIUM
read -p "Lactate (mmol/L): " LACTATE

echo ""
echo "📊 CLINICAL SCORES"
echo "-----------------"
read -p "NEWS2 Score (0-20): " NEWS2
read -p "qSOFA Score (0-3): " QSOFA

echo ""
echo "⏳ Processing patient data..."
echo ""

# Create temporary Java test file with user's data
cat > /tmp/ManualPatientTest.java << 'JAVACODE'
import com.cardiofit.flink.adapters.PatientContextAdapter;
import com.cardiofit.flink.ml.ONNXModelContainer;
import com.cardiofit.flink.ml.ModelConfig;
import com.cardiofit.flink.ml.PatientContextSnapshot;
import com.cardiofit.flink.ml.features.MIMICFeatureExtractor;
import com.cardiofit.flink.models.*;
import com.cardiofit.flink.models.MLPrediction;

import java.util.*;

public class ManualPatientTest {
    public static void main(String[] args) {
        // Parse command line arguments
        Map<String, String> data = new HashMap<>();
        for (int i = 0; i < args.length; i += 2) {
            if (i + 1 < args.length) {
                data.put(args[i], args[i + 1]);
            }
        }

        System.out.println("╔════════════════════════════════════════════════════════════════╗");
        System.out.println("║     MIMIC-IV Model Predictions                                 ║");
        System.out.println("╚════════════════════════════════════════════════════════════════╝");
        System.out.println();

        try {
            // Create patient context from user input
            EnrichedPatientContext context = createPatientFromInput(data);

            // Convert to snapshot
            PatientContextAdapter adapter = new PatientContextAdapter();
            PatientContextSnapshot snapshot = adapter.adapt(context);

            System.out.println("📋 PATIENT PROFILE:");
            System.out.println("   ID: " + data.get("patientId"));
            System.out.println("   Age: " + data.get("age") + " years");
            System.out.println("   Gender: " + data.get("gender"));
            System.out.println("   Weight: " + data.get("weight") + " kg");
            System.out.println();

            System.out.println("❤️ VITAL SIGNS:");
            System.out.println("   Heart Rate: " + data.get("heartRate") + " bpm");
            System.out.println("   Blood Pressure: " + data.get("sbp") + "/" + data.get("dbp") + " mmHg");
            System.out.println("   Respiratory Rate: " + data.get("respRate") + " breaths/min");
            System.out.println("   Temperature: " + data.get("temp") + " °C");
            System.out.println("   SpO2: " + data.get("spo2") + "%");
            System.out.println();

            System.out.println("🧪 KEY LAB VALUES:");
            System.out.println("   WBC: " + data.get("wbc") + " 10^3/uL");
            System.out.println("   Hemoglobin: " + data.get("hgb") + " g/dL");
            System.out.println("   Creatinine: " + data.get("creatinine") + " mg/dL");
            System.out.println("   Lactate: " + data.get("lactate") + " mmol/L");
            System.out.println();

            System.out.println("📊 CLINICAL SCORES:");
            System.out.println("   NEWS2: " + data.get("news2"));
            System.out.println("   qSOFA: " + data.get("qsofa"));
            System.out.println();

            // Extract features
            MIMICFeatureExtractor extractor = new MIMICFeatureExtractor();
            float[] features = extractor.extractFeatures(snapshot);

            System.out.println("🔬 FEATURE EXTRACTION:");
            System.out.println("   ✅ Extracted " + features.length + " MIMIC-IV features");
            System.out.println();

            // Load models and run inference
            System.out.println("🤖 LOADING ML MODELS...");
            System.out.println("═══════════════════════════════════════════════════════════════");

            // Sepsis Model
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

            // Deterioration Model
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

            // Mortality Model
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

            System.out.println("✅ All 3 MIMIC-IV models loaded successfully!");
            System.out.println();

            // Run predictions
            System.out.println("🔮 RISK PREDICTIONS:");
            System.out.println("═══════════════════════════════════════════════════════════════");

            MLPrediction sepsisPred = sepsisModel.predict(features);
            double sepsisRisk = sepsisPred.getPredictionScores().get("confidence_score");
            System.out.printf("   Sepsis Risk         : %.4f (%.2f%%) - %s%n",
                sepsisRisk, sepsisRisk * 100, getRiskLevel((float)sepsisRisk));

            MLPrediction detPred = detModel.predict(features);
            double detRisk = detPred.getPredictionScores().get("confidence_score");
            System.out.printf("   Deterioration Risk  : %.4f (%.2f%%) - %s%n",
                detRisk, detRisk * 100, getRiskLevel((float)detRisk));

            MLPrediction mortalityPred = mortalityModel.predict(features);
            double mortalityRisk = mortalityPred.getPredictionScores().get("confidence_score");
            System.out.printf("   Mortality Risk      : %.4f (%.2f%%) - %s%n",
                mortalityRisk, mortalityRisk * 100, getRiskLevel((float)mortalityRisk));

            System.out.println();
            System.out.println("═══════════════════════════════════════════════════════════════");

            // Clinical recommendations
            System.out.println();
            System.out.println("💡 CLINICAL RECOMMENDATIONS:");
            if (sepsisRisk > 0.8 || detRisk > 0.8 || mortalityRisk > 0.8) {
                System.out.println("   🚨 HIGH RISK - Immediate clinical intervention recommended");
                System.out.println("   • Consider ICU consultation");
                System.out.println("   • Increase monitoring frequency");
                System.out.println("   • Review treatment plan urgently");
            } else if (sepsisRisk > 0.5 || detRisk > 0.5 || mortalityRisk > 0.5) {
                System.out.println("   ⚠️ MODERATE RISK - Enhanced monitoring recommended");
                System.out.println("   • Increase vital sign frequency");
                System.out.println("   • Consider additional lab tests");
                System.out.println("   • Review clinical status");
            } else {
                System.out.println("   ✅ LOW RISK - Continue standard care");
                System.out.println("   • Maintain routine monitoring");
                System.out.println("   • Follow standard protocols");
            }

            System.out.println();
            System.out.println("✅ Analysis Complete!");

        } catch (Exception e) {
            System.err.println("❌ Error: " + e.getMessage());
            e.printStackTrace();
            System.exit(1);
        }
    }

    private static EnrichedPatientContext createPatientFromInput(Map<String, String> data) {
        EnrichedPatientContext context = new EnrichedPatientContext();
        context.setPatientId(data.get("patientId"));
        context.setEncounterId("ENC-" + data.get("patientId"));
        context.setEventTime(System.currentTimeMillis());
        context.setEventType("MANUAL_TEST");

        PatientContextState state = new PatientContextState();

        // Demographics
        PatientDemographics demographics = new PatientDemographics();
        demographics.setAge(Integer.parseInt(data.get("age")));
        demographics.setGender(data.get("gender"));
        demographics.setWeight(Double.parseDouble(data.get("weight")));
        state.setDemographics(demographics);

        // Vital Signs
        Map<String, Object> vitals = new HashMap<>();
        vitals.put("heartrate", Double.parseDouble(data.get("heartRate")));
        vitals.put("systolicbloodpressure", Double.parseDouble(data.get("sbp")));
        vitals.put("diastolicbloodpressure", Double.parseDouble(data.get("dbp")));
        vitals.put("respiratoryrate", Double.parseDouble(data.get("respRate")));
        vitals.put("temperature", Double.parseDouble(data.get("temp")));
        vitals.put("oxygensaturation", Double.parseDouble(data.get("spo2")));
        state.setLatestVitals(vitals);

        // Lab Values
        Map<String, LabResult> labs = new HashMap<>();
        labs.put("WBC", createLabResult(Double.parseDouble(data.get("wbc")), "10^3/uL"));
        labs.put("hemoglobin", createLabResult(Double.parseDouble(data.get("hgb")), "g/dL"));
        labs.put("platelets", createLabResult(Double.parseDouble(data.get("platelets")), "10^3/uL"));
        labs.put("creatinine", createLabResult(Double.parseDouble(data.get("creatinine")), "mg/dL"));
        labs.put("bun", createLabResult(Double.parseDouble(data.get("bun")), "mg/dL"));
        labs.put("glucose", createLabResult(Double.parseDouble(data.get("glucose")), "mg/dL"));
        labs.put("sodium", createLabResult(Double.parseDouble(data.get("sodium")), "mmol/L"));
        labs.put("potassium", createLabResult(Double.parseDouble(data.get("potassium")), "mmol/L"));
        labs.put("lactate", createLabResult(Double.parseDouble(data.get("lactate")), "mmol/L"));
        state.setRecentLabs(labs);

        // Clinical Scores
        state.setNews2Score(Integer.parseInt(data.get("news2")));
        state.setQsofaScore(Integer.parseInt(data.get("qsofa")));
        state.setCombinedAcuityScore(Double.parseDouble(data.get("news2")) * 5.0);

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
        if (risk >= 0.8) return "🚨 HIGH RISK";
        if (risk >= 0.5) return "⚠️ MODERATE RISK";
        if (risk >= 0.3) return "⚡ LOW-MODERATE RISK";
        return "✅ LOW RISK";
    }
}
JAVACODE

# Compile the temporary test class
echo "📦 Compiling test..."
mvn compile -q
cp /tmp/ManualPatientTest.java src/test/java/
cd src/test/java && javac -cp "../../../target/classes:../../../target/test-classes:$HOME/.m2/repository/org/apache/flink/flink-core/2.0.0/flink-core-2.0.0.jar:$HOME/.m2/repository/org/slf4j/slf4j-api/1.7.36/slf4j-api-1.7.36.jar" ManualPatientTest.java 2>/dev/null || true
cd ../../..

# Run the test with user's data
echo "🚀 Running MIMIC-IV inference..."
echo ""

mvn exec:java \
  -Dexec.mainClass="ManualPatientTest" \
  -Dexec.classpathScope="test" \
  -Dexec.args="patientId $PATIENT_ID age $AGE gender $GENDER weight $WEIGHT heartRate $HEART_RATE sbp $SBP dbp $DBP respRate $RESP_RATE temp $TEMP spo2 $SPO2 wbc $WBC hgb $HGB platelets $PLATELETS creatinine $CREATININE bun $BUN glucose $GLUCOSE sodium $SODIUM potassium $POTASSIUM lactate $LACTATE news2 $NEWS2 qsofa $QSOFA" \
  -q

# Cleanup
rm -f /tmp/ManualPatientTest.java src/test/java/ManualPatientTest.java

echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "Test completed! You can run this script again to test another patient."
echo "═══════════════════════════════════════════════════════════════"
