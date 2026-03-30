package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.PatientMLState;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.*;

/**
 * Static feature extraction for Module 5 ONNX inference.
 * Extracts a 55-element float array from PatientMLState.
 *
 * Feature layout (indices):
 * [0-5]   Vital signs (normalized 0-1)
 * [6-8]   Clinical scores (normalized 0-1)
 * [9-10]  Event count (log-scaled) + hours since admission
 * [11-20] NEWS2 history ring buffer
 * [21-30] Acuity history ring buffer
 * [31-34] Pattern features
 * [35-44] Risk indicator flags (0/1)
 * [45-49] Lab-derived features (normalized, -1 = missing)
 * [50-54] Active alert features (binary + severity)
 *
 * All methods are static and testable without Flink runtime.
 */
public class Module5FeatureExtractor {
    private static final Logger LOG = LoggerFactory.getLogger(Module5FeatureExtractor.class);

    public static final int FEATURE_COUNT = 55;

    // ── Vital key aliases (Gap 1: snake_case → production format) ──
    private static final Map<String, String> VITAL_KEY_ALIASES = Map.ofEntries(
        Map.entry("heart_rate", "heartrate"),
        Map.entry("systolic_bp", "systolicbloodpressure"),
        Map.entry("diastolic_bp", "diastolicbloodpressure"),
        Map.entry("respiratory_rate", "respiratoryrate"),
        Map.entry("oxygen_saturation", "oxygensaturation"),
        Map.entry("HeartRate", "heartrate"),
        Map.entry("SystolicBP", "systolicbloodpressure")
    );

    // ── Non-vital fields that appear in latestVitals (must skip) ──
    private static final Set<String> NON_VITAL_KEYS = Set.of(
        "age", "gender", "bloodpressure", "weight", "height",
        "restingheartrate", "leftventricularejectionfraction", "data_tier"
    );

    /**
     * Extract full 55-element feature vector from patient ML state.
     */
    public static float[] extractFeatures(PatientMLState state) {
        float[] features = new float[FEATURE_COUNT];

        if (state == null) {
            LOG.warn("Null PatientMLState — returning zero feature vector");
            return features;
        }

        // [0-5] Vital signs
        Map<String, Double> vitals = normalizeVitalKeys(state.getLatestVitals());
        features[0] = normalize(vitals.getOrDefault("heartrate", 0.0), 30, 200);
        features[1] = normalize(vitals.getOrDefault("systolicbloodpressure", 0.0), 60, 250);
        features[2] = normalize(vitals.getOrDefault("diastolicbloodpressure", 0.0), 30, 150);
        features[3] = normalize(vitals.getOrDefault("respiratoryrate", 0.0), 5, 50);
        features[4] = normalize(vitals.getOrDefault("oxygensaturation", 0.0), 70, 100);
        features[5] = normalize(vitals.getOrDefault("temperature", 0.0), 34, 42);

        // [6-8] Clinical scores
        features[6] = normalize(state.getNews2Score(), 0, 20);
        features[7] = normalize(state.getQsofaScore(), 0, 3);
        features[8] = normalize(state.getAcuityScore(), 0, 10);

        // [9-10] Event context
        features[9] = (float) Math.log1p(state.getTotalEventCount());
        long hoursSinceAdmission = state.getFirstEventTime() > 0
            ? (System.currentTimeMillis() - state.getFirstEventTime()) / 3600000L
            : 0;
        features[10] = normalize(Math.min(hoursSinceAdmission, 720), 0, 720);

        // [11-20] NEWS2 history
        double[] news2Hist = state.getNews2History();
        int news2Count = (int) Math.min(state.getNews2HistoryIndex(), 10);
        for (int i = 0; i < 10; i++) {
            features[11 + i] = i < news2Count ? normalize(news2Hist[i], 0, 20) : 0.0f;
        }

        // [21-30] Acuity history
        double[] acuityHist = state.getAcuityHistory();
        int acuityCount = (int) Math.min(state.getAcuityHistoryIndex(), 10);
        for (int i = 0; i < 10; i++) {
            features[21 + i] = i < acuityCount ? normalize(acuityHist[i], 0, 10) : 0.0f;
        }

        // [31-34] Pattern features
        features[31] = (float) state.getRecentPatterns().size();
        features[32] = (float) state.getDeteriorationPatternCount();
        features[33] = (float) PatientMLState.severityIndex(state.getMaxSeveritySeen());
        features[34] = state.isSeverityEscalationDetected() ? 1.0f : 0.0f;

        // [35-44] Risk indicator flags (Gap 2: safe extraction with missingness)
        Map<String, Object> risk = state.getRiskIndicators();
        features[35] = safeRiskFlag(risk, "tachycardia");
        features[36] = safeRiskFlag(risk, "hypotension");
        features[37] = safeRiskFlag(risk, "fever");
        features[38] = safeRiskFlag(risk, "hypoxia");
        features[39] = safeRiskFlag(risk, "elevatedLactate");
        features[40] = safeRiskFlag(risk, "elevatedCreatinine");
        features[41] = safeRiskFlag(risk, "hyperkalemia");
        features[42] = safeRiskFlag(risk, "thrombocytopenia");
        features[43] = safeRiskFlag(risk, "onAnticoagulation");
        features[44] = safeRiskFlag(risk, "sepsisRisk");

        // [45-49] Lab-derived features (Gap 1: -1 = missing)
        Map<String, Double> labs = state.getLatestLabs();
        features[45] = safeLabFeature(labs, "lactate", 0, 20);
        features[46] = safeLabFeature(labs, "creatinine", 0, 15);
        features[47] = safeLabFeature(labs, "potassium", 2, 8);
        features[48] = safeLabFeature(labs, "wbc", 0, 40);
        features[49] = safeLabFeature(labs, "platelets", 0, 500);

        // [50-54] Active alert features (Gap 4)
        Map<String, Object> alerts = state.getActiveAlerts();
        features[50] = alertPresent(alerts, "SEPSIS_PATTERN") ? 1.0f : 0.0f;
        features[51] = alertPresent(alerts, "AKI_RISK") ? 1.0f : 0.0f;
        features[52] = alertPresent(alerts, "ANTICOAGULATION_RISK") ? 1.0f : 0.0f;
        features[53] = alertPresent(alerts, "BLEEDING_RISK") ? 1.0f : 0.0f;
        features[54] = alertMaxSeverity(alerts);

        return features;
    }

    // ── Vital key normalization ──

    public static Map<String, Double> normalizeVitalKeys(Map<String, Double> rawVitals) {
        if (rawVitals == null) return Collections.emptyMap();
        Map<String, Double> normalized = new HashMap<>();
        for (Map.Entry<String, Double> entry : rawVitals.entrySet()) {
            String key = VITAL_KEY_ALIASES.getOrDefault(entry.getKey(), entry.getKey());
            if (NON_VITAL_KEYS.contains(key)) continue;
            if (entry.getValue() != null) {
                normalized.put(key, entry.getValue());
            }
        }
        return normalized;
    }

    // ── Safe risk flag extraction (Gap 2) ──

    public static float safeRiskFlag(Map<String, Object> risk, String key) {
        if (risk == null) return 0.0f;
        Object val = risk.get(key);
        if (val instanceof Boolean) return ((Boolean) val) ? 1.0f : 0.0f;
        return 0.0f;
    }

    // ── Safe lab feature extraction (Gap 1) ──

    public static float safeLabFeature(Map<String, Double> labs, String key, double min, double max) {
        if (labs == null) return -1.0f;
        Double val = labs.get(key);
        if (val == null) return -1.0f;
        return normalize(val, min, max);
    }

    // ── Alert feature extraction (Gap 4) ──

    public static boolean alertPresent(Map<String, Object> alerts, String alertType) {
        if (alerts == null) return false;
        return alerts.containsKey(alertType);
    }

    @SuppressWarnings("unchecked")
    public static float alertMaxSeverity(Map<String, Object> alerts) {
        if (alerts == null || alerts.isEmpty()) return 0.0f;
        int maxSev = 0;
        for (Object alertObj : alerts.values()) {
            if (alertObj instanceof Map) {
                Map<String, Object> alert = (Map<String, Object>) alertObj;
                Object sev = alert.get("severity");
                if (sev instanceof String) {
                    maxSev = Math.max(maxSev, PatientMLState.severityIndex((String) sev));
                }
            }
        }
        return normalize(maxSev, 0, 4);
    }

    // ── Normalization ──

    public static float normalize(double value, double min, double max) {
        if (Math.abs(max - min) < 1e-9) return 0.0f;
        return (float) Math.max(0.0, Math.min(1.0, (value - min) / (max - min)));
    }
}
