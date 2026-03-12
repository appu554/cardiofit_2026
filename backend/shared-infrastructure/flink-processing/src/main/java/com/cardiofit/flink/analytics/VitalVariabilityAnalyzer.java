package com.cardiofit.flink.analytics;

import com.cardiofit.flink.models.EventType;
import com.cardiofit.flink.models.SemanticEvent;
import com.cardiofit.flink.models.VitalVariabilityAlert;
import org.apache.flink.streaming.api.datastream.DataStream;
import org.apache.flink.streaming.api.functions.windowing.WindowFunction;
import org.apache.flink.streaming.api.windowing.assigners.SlidingEventTimeWindows;
import org.apache.flink.streaming.api.windowing.windows.TimeWindow;
import org.apache.flink.util.Collector;

import java.io.Serializable;
import java.time.Duration;
import java.util.*;

/**
 * Vital Sign Variability Analyzer for Apache Flink.
 *
 * Analyzes vital sign variability using Coefficient of Variation (CV).
 * High variability indicates physiological instability and increased mortality risk.
 *
 * Clinical Thresholds:
 * - Heart Rate CV >15%: Cardiac instability, autonomic dysfunction
 * - Blood Pressure CV >15%: Hemodynamic instability
 * - Respiratory Rate CV >20%: Respiratory distress
 * - Temperature CV >5%: Infection/inflammatory process
 * - SpO2 CV >5%: Oxygen instability
 *
 * Coefficient of Variation: CV = (Standard Deviation / Mean) × 100%
 *
 * Clinical Applications:
 * - Early sepsis detection: Increased HR variability
 * - Hemodynamic instability: High BP variability
 * - Autonomic dysfunction: Abnormal HR variability
 * - Respiratory distress: High RR variability
 *
 * Architecture:
 * - 4-hour sliding window (sufficient for variability assessment)
 * - 30-minute slide interval for timely alerts
 * - Per-vital-sign analysis with specific thresholds
 * - Statistical significance validation (minimum 5 readings)
 */
public class VitalVariabilityAnalyzer implements Serializable {
    private static final long serialVersionUID = 1L;

    // Clinical thresholds for coefficient of variation (%)
    private static final double HEART_RATE_CV_THRESHOLD = 15.0;
    private static final double BLOOD_PRESSURE_CV_THRESHOLD = 15.0;
    private static final double RESPIRATORY_RATE_CV_THRESHOLD = 20.0;
    private static final double TEMPERATURE_CV_THRESHOLD = 5.0;
    private static final double SPO2_CV_THRESHOLD = 5.0;

    // Minimum readings required for reliable variability assessment
    private static final int MIN_READINGS = 5;

    /**
     * Analyze heart rate variability.
     * 4-hour sliding window with 30-minute slide.
     */
    public static DataStream<VitalVariabilityAlert> analyzeHeartRateVariability(
            DataStream<SemanticEvent> vitalStream) {
        return vitalStream
            .filter(event -> isHeartRateVital(event))
            .keyBy(SemanticEvent::getPatientId)
            .window(SlidingEventTimeWindows.of(Duration.ofHours(4), Duration.ofMinutes(30)))
            .apply(new VitalVariabilityWindowFunction("Heart Rate", HEART_RATE_CV_THRESHOLD))
            .uid("Heart_Rate_Variability_Analysis");
    }

    /**
     * Analyze systolic blood pressure variability.
     * 4-hour sliding window with 30-minute slide.
     */
    public static DataStream<VitalVariabilityAlert> analyzeSystolicBPVariability(
            DataStream<SemanticEvent> vitalStream) {
        return vitalStream
            .filter(event -> isSystolicBPVital(event))
            .keyBy(SemanticEvent::getPatientId)
            .window(SlidingEventTimeWindows.of(Duration.ofHours(4), Duration.ofMinutes(30)))
            .apply(new VitalVariabilityWindowFunction("Systolic BP", BLOOD_PRESSURE_CV_THRESHOLD))
            .uid("Systolic_BP_Variability_Analysis");
    }

    /**
     * Analyze respiratory rate variability.
     * 4-hour sliding window with 30-minute slide.
     */
    public static DataStream<VitalVariabilityAlert> analyzeRespiratoryRateVariability(
            DataStream<SemanticEvent> vitalStream) {
        return vitalStream
            .filter(event -> isRespiratoryRateVital(event))
            .keyBy(SemanticEvent::getPatientId)
            .window(SlidingEventTimeWindows.of(Duration.ofHours(4), Duration.ofMinutes(30)))
            .apply(new VitalVariabilityWindowFunction("Respiratory Rate", RESPIRATORY_RATE_CV_THRESHOLD))
            .uid("Respiratory_Rate_Variability_Analysis");
    }

    /**
     * Analyze temperature variability.
     * 4-hour sliding window with 30-minute slide.
     */
    public static DataStream<VitalVariabilityAlert> analyzeTemperatureVariability(
            DataStream<SemanticEvent> vitalStream) {
        return vitalStream
            .filter(event -> isTemperatureVital(event))
            .keyBy(SemanticEvent::getPatientId)
            .window(SlidingEventTimeWindows.of(Duration.ofHours(4), Duration.ofMinutes(30)))
            .apply(new VitalVariabilityWindowFunction("Temperature", TEMPERATURE_CV_THRESHOLD))
            .uid("Temperature_Variability_Analysis");
    }

    /**
     * Analyze SpO2 variability.
     * 4-hour sliding window with 30-minute slide.
     */
    public static DataStream<VitalVariabilityAlert> analyzeSpO2Variability(
            DataStream<SemanticEvent> vitalStream) {
        return vitalStream
            .filter(event -> isSpO2Vital(event))
            .keyBy(SemanticEvent::getPatientId)
            .window(SlidingEventTimeWindows.of(Duration.ofHours(4), Duration.ofMinutes(30)))
            .apply(new VitalVariabilityWindowFunction("SpO2", SPO2_CV_THRESHOLD))
            .uid("SpO2_Variability_Analysis");
    }

    // Vital sign type filters
    private static boolean isHeartRateVital(SemanticEvent event) {
        if (event.getEventType() != EventType.VITAL_SIGN &&
            event.getEventType() != EventType.VITAL_SIGNS) return false;
        String vitalType = getVitalType(event);
        return vitalType != null &&
               (vitalType.toLowerCase().contains("heart_rate") ||
                vitalType.toLowerCase().contains("pulse") ||
                vitalType.toLowerCase().contains("hr"));
    }

    private static boolean isSystolicBPVital(SemanticEvent event) {
        if (event.getEventType() != EventType.VITAL_SIGN &&
            event.getEventType() != EventType.VITAL_SIGNS) return false;
        String vitalType = getVitalType(event);
        return vitalType != null &&
               (vitalType.toLowerCase().contains("systolic") ||
                vitalType.toLowerCase().contains("sbp") ||
                vitalType.toLowerCase().contains("blood_pressure_systolic"));
    }

    private static boolean isRespiratoryRateVital(SemanticEvent event) {
        if (event.getEventType() != EventType.VITAL_SIGN &&
            event.getEventType() != EventType.VITAL_SIGNS) return false;
        String vitalType = getVitalType(event);
        return vitalType != null &&
               (vitalType.toLowerCase().contains("respiratory_rate") ||
                vitalType.toLowerCase().contains("rr") ||
                vitalType.toLowerCase().contains("respiration"));
    }

    private static boolean isTemperatureVital(SemanticEvent event) {
        if (event.getEventType() != EventType.VITAL_SIGN &&
            event.getEventType() != EventType.VITAL_SIGNS) return false;
        String vitalType = getVitalType(event);
        return vitalType != null &&
               (vitalType.toLowerCase().contains("temperature") ||
                vitalType.toLowerCase().contains("temp"));
    }

    private static boolean isSpO2Vital(SemanticEvent event) {
        if (event.getEventType() != EventType.VITAL_SIGN &&
            event.getEventType() != EventType.VITAL_SIGNS) return false;
        String vitalType = getVitalType(event);
        return vitalType != null &&
               (vitalType.toLowerCase().contains("spo2") ||
                vitalType.toLowerCase().contains("oxygen_saturation") ||
                vitalType.toLowerCase().contains("o2_sat"));
    }

    private static String getVitalType(SemanticEvent event) {
        Map<String, Object> data = event.getClinicalData();
        if (data == null) return null;

        Object vitalType = data.get("vital_type");
        if (vitalType != null) return vitalType.toString();

        Object vitalName = data.get("vital_name");
        if (vitalName != null) return vitalName.toString();

        Object code = data.get("code");
        if (code != null) return code.toString();

        return null;
    }

    private static Double getVitalValue(SemanticEvent event) {
        Map<String, Object> data = event.getClinicalData();
        if (data == null) return null;

        Object value = data.get("value");
        if (value == null) value = data.get("vital_value");
        if (value == null) value = data.get("numeric_value");

        if (value instanceof Number) {
            return ((Number) value).doubleValue();
        }

        // Try parsing string values
        if (value instanceof String) {
            try {
                return Double.parseDouble((String) value);
            } catch (NumberFormatException e) {
                return null;
            }
        }

        return null;
    }

    /**
     * Generic vital sign variability window function.
     * Calculates CV and generates alerts based on threshold.
     */
    public static class VitalVariabilityWindowFunction
            implements WindowFunction<SemanticEvent, VitalVariabilityAlert, String, TimeWindow> {

        private final String vitalSignName;
        private final double cvThreshold;

        public VitalVariabilityWindowFunction(String vitalSignName, double cvThreshold) {
            this.vitalSignName = vitalSignName;
            this.cvThreshold = cvThreshold;
        }

        @Override
        public void apply(String patientId, TimeWindow window,
                         Iterable<SemanticEvent> events,
                         Collector<VitalVariabilityAlert> out) throws Exception {

            List<Double> values = extractVitalValues(events);
            if (values.size() < MIN_READINGS) return;  // Need sufficient readings

            // Calculate statistics
            double mean = calculateMean(values);
            double stdDev = calculateStdDev(values, mean);
            double cv = (stdDev / mean) * 100;  // Coefficient of variation

            // Only alert if CV exceeds threshold
            if (cv > cvThreshold) {
                String variabilityLevel = categorizeVariability(cv, cvThreshold);
                String clinicalSignificance = interpretVariability(
                    vitalSignName, cv, mean, stdDev, values);

                VitalVariabilityAlert alert = new VitalVariabilityAlert();
                alert.setPatientId(patientId);
                alert.setVitalSignName(vitalSignName);
                alert.setMeanValue(mean);
                alert.setStandardDeviation(stdDev);
                alert.setCoefficientOfVariation(cv);
                alert.setVariabilityLevel(variabilityLevel);
                alert.setClinicalSignificance(clinicalSignificance);
                alert.setTimestamp(System.currentTimeMillis());
                alert.setWindowStart(window.getStart());
                alert.setWindowEnd(window.getEnd());

                out.collect(alert);
            }
        }

        private List<Double> extractVitalValues(Iterable<SemanticEvent> events) {
            List<Double> values = new ArrayList<>();
            for (SemanticEvent event : events) {
                Double value = getVitalValue(event);
                if (value != null && value > 0) {  // Filter invalid values
                    values.add(value);
                }
            }
            return values;
        }

        private String categorizeVariability(double cv, double threshold) {
            if (cv > threshold * 2) {
                return "CRITICAL";
            } else if (cv > threshold * 1.5) {
                return "HIGH";
            } else if (cv > threshold) {
                return "MODERATE";
            } else {
                return "LOW";
            }
        }

        private String interpretVariability(String vitalName, double cv,
                                           double mean, double stdDev,
                                           List<Double> values) {
            StringBuilder interpretation = new StringBuilder();

            double min = values.stream().mapToDouble(v -> v).min().orElse(0);
            double max = values.stream().mapToDouble(v -> v).max().orElse(0);

            interpretation.append(String.format("⚠️ HIGH %s VARIABILITY DETECTED\n", vitalName.toUpperCase()));
            interpretation.append(String.format("Coefficient of Variation: %.1f%% (Threshold: %.1f%%)\n",
                cv, cvThreshold));

            interpretation.append(String.format("\nStatistics:\n"));
            interpretation.append(String.format("  Mean: %.1f\n", mean));
            interpretation.append(String.format("  Range: %.1f - %.1f\n", min, max));
            interpretation.append(String.format("  Standard Deviation: %.1f\n", stdDev));
            interpretation.append(String.format("  Measurements: %d\n", values.size()));

            // Vital-specific clinical interpretation
            switch (vitalName) {
                case "Heart Rate":
                    interpretation.append("\nClinical Significance:\n");
                    interpretation.append("- May indicate autonomic dysfunction or cardiac instability\n");
                    interpretation.append("- Consider: Sepsis, arrhythmia, pain, anxiety\n");
                    interpretation.append("Actions: ECG monitoring, assess pain/agitation, check hemodynamics\n");
                    break;

                case "Systolic BP":
                    interpretation.append("\nClinical Significance:\n");
                    interpretation.append("- Suggests hemodynamic instability\n");
                    interpretation.append("- Consider: Volume status, cardiac output, vasopressor needs\n");
                    interpretation.append("Actions: Fluid assessment, MAP target review, cardiac output monitoring\n");
                    break;

                case "Respiratory Rate":
                    interpretation.append("\nClinical Significance:\n");
                    interpretation.append("- May indicate respiratory distress or metabolic derangement\n");
                    interpretation.append("- Consider: Pain, anxiety, metabolic acidosis, respiratory failure\n");
                    interpretation.append("Actions: ABG analysis, assess work of breathing, consider ventilatory support\n");
                    break;

                case "Temperature":
                    interpretation.append("\nClinical Significance:\n");
                    interpretation.append("- May indicate infection or inflammatory process\n");
                    interpretation.append("- Consider: Sepsis, drug fever, malignancy\n");
                    interpretation.append("Actions: Blood cultures, infectious workup, review medications\n");
                    break;

                case "SpO2":
                    interpretation.append("\nClinical Significance:\n");
                    interpretation.append("- Suggests oxygen instability\n");
                    interpretation.append("- Consider: Pulmonary disease, cardiac issues, positioning\n");
                    interpretation.append("Actions: Assess respiratory status, optimize oxygen delivery, consider ABG\n");
                    break;

                default:
                    interpretation.append("\nClinical Significance:\n");
                    interpretation.append("- High variability indicates physiological instability\n");
                    interpretation.append("Actions: Close monitoring, assess underlying causes\n");
            }

            // Severity-based recommendations
            if (cv > cvThreshold * 2) {
                interpretation.append("\n🔴 CRITICAL VARIABILITY: Immediate clinical assessment required\n");
                interpretation.append("Consider ICU-level monitoring and specialist consultation\n");
            } else if (cv > cvThreshold * 1.5) {
                interpretation.append("\n⚠️ HIGH VARIABILITY: Close monitoring and intervention needed\n");
            }

            return interpretation.toString();
        }

        private double calculateMean(List<Double> values) {
            return values.stream()
                .mapToDouble(v -> v)
                .average()
                .orElse(0.0);
        }

        private double calculateStdDev(List<Double> values, double mean) {
            if (values.size() < 2) return 0.0;

            double variance = values.stream()
                .mapToDouble(v -> Math.pow(v - mean, 2))
                .sum() / (values.size() - 1);  // Sample standard deviation

            return Math.sqrt(variance);
        }
    }

    /**
     * Analyze all vital signs variability in a unified stream.
     * Useful for comprehensive patient monitoring.
     */
    public static DataStream<VitalVariabilityAlert> analyzeAllVitalVariability(
            DataStream<SemanticEvent> vitalStream) {

        DataStream<VitalVariabilityAlert> hrAlerts = analyzeHeartRateVariability(vitalStream);
        DataStream<VitalVariabilityAlert> bpAlerts = analyzeSystolicBPVariability(vitalStream);
        DataStream<VitalVariabilityAlert> rrAlerts = analyzeRespiratoryRateVariability(vitalStream);
        DataStream<VitalVariabilityAlert> tempAlerts = analyzeTemperatureVariability(vitalStream);
        DataStream<VitalVariabilityAlert> spo2Alerts = analyzeSpO2Variability(vitalStream);

        return hrAlerts
            .union(bpAlerts)
            .union(rrAlerts)
            .union(tempAlerts)
            .union(spo2Alerts);
    }
}
