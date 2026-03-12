package com.cardiofit.flink.alerts;

import com.cardiofit.flink.indicators.EnhancedRiskIndicators;
import com.cardiofit.flink.scoring.ClinicalScoreCalculator.AcuityScores;
import org.apache.flink.api.common.state.MapState;
import org.apache.flink.api.common.state.MapStateDescriptor;
import org.apache.flink.api.common.typeinfo.TypeInformation;
import org.apache.flink.api.common.typeinfo.Types;
import org.apache.flink.configuration.Configuration;
import org.apache.flink.streaming.api.functions.ProcessFunction;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.util.ArrayList;
import java.util.List;
import java.util.Map;

/**
 * Alert Generator with Intelligent Suppression.
 *
 * Features:
 * - Severity-based alert generation
 * - Time-based suppression to prevent alert fatigue
 * - Combination alerts for multiple conditions
 * - Priority override for critical alerts
 * - Stateful tracking of alert history
 */
public class AlertGenerator implements Serializable {

    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(AlertGenerator.class);

    // Suppression windows by severity
    private static final long SUPPRESSION_WINDOW_CRITICAL_MS = 1800000;  // 30 minutes for critical
    private static final long SUPPRESSION_WINDOW_HIGH_MS = 3600000;      // 1 hour for high
    private static final long SUPPRESSION_WINDOW_MEDIUM_MS = 7200000;    // 2 hours for medium
    private static final long SUPPRESSION_WINDOW_LOW_MS = 14400000;      // 4 hours for low

    // State for alert suppression tracking
    private transient MapState<String, Long> lastAlertTimestamps;
    private transient MapState<String, Integer> alertCounts;

    /**
     * Alert object containing type, severity, message, and metadata.
     */
    public static class Alert implements Serializable {
        private String type;
        private String severity;
        private String message;
        private long timestamp;
        private Map<String, Object> metadata;
        private String actionRequired;
        private boolean suppressed;
        private int occurrenceCount;

        public Alert() {}

        public Alert(String type, String severity, String message, long timestamp) {
            this.type = type;
            this.severity = severity;
            this.message = message;
            this.timestamp = timestamp;
            this.suppressed = false;
            this.occurrenceCount = 1;
        }

        // Getters and setters
        public String getType() { return type; }
        public void setType(String type) { this.type = type; }

        public String getSeverity() { return severity; }
        public void setSeverity(String severity) { this.severity = severity; }

        public String getMessage() { return message; }
        public void setMessage(String message) { this.message = message; }

        public long getTimestamp() { return timestamp; }
        public void setTimestamp(long timestamp) { this.timestamp = timestamp; }

        public Map<String, Object> getMetadata() { return metadata; }
        public void setMetadata(Map<String, Object> metadata) { this.metadata = metadata; }

        public String getActionRequired() { return actionRequired; }
        public void setActionRequired(String actionRequired) { this.actionRequired = actionRequired; }

        public boolean isSuppressed() { return suppressed; }
        public void setSuppressed(boolean suppressed) { this.suppressed = suppressed; }

        public int getOccurrenceCount() { return occurrenceCount; }
        public void setOccurrenceCount(int occurrenceCount) { this.occurrenceCount = occurrenceCount; }
    }

    /**
     * Initialize state for alert tracking.
     */
    public void open(Configuration parameters) throws Exception {
        MapStateDescriptor<String, Long> timestampDescriptor =
            new MapStateDescriptor<>("alert-timestamps",
                TypeInformation.of(String.class),
                TypeInformation.of(Long.class));
        lastAlertTimestamps = getRuntimeContext().getMapState(timestampDescriptor);

        MapStateDescriptor<String, Integer> countDescriptor =
            new MapStateDescriptor<>("alert-counts",
                TypeInformation.of(String.class),
                TypeInformation.of(Integer.class));
        alertCounts = getRuntimeContext().getMapState(countDescriptor);
    }

    /**
     * Generate alerts based on risk indicators and acuity scores.
     */
    public List<Alert> generateAlerts(
            EnhancedRiskIndicators.RiskAssessment indicators,
            AcuityScores scores,
            long eventTime,
            String patientId) throws Exception {

        List<Alert> alerts = new ArrayList<>();
        long now = System.currentTimeMillis();

        // 1. Vital sign alerts
        generateVitalSignAlerts(indicators, alerts, now, patientId);

        // 2. Acuity-based alerts
        generateAcuityAlerts(scores, alerts, now, patientId);

        // 3. Combination alerts (multiple conditions)
        generateCombinationAlerts(indicators, scores, alerts, now, patientId);

        // 4. Trend-based alerts
        generateTrendAlerts(indicators, alerts, now, patientId);

        // 5. Missing data alerts
        generateDataQualityAlerts(indicators, alerts, now, patientId);

        LOG.info("Generated {} alerts for patient {} (before suppression)",
            alerts.size(), patientId);

        // Apply suppression logic
        List<Alert> finalAlerts = applySuppression(alerts, patientId);

        LOG.info("Final alert count for patient {}: {} (after suppression)",
            patientId, finalAlerts.size());

        return finalAlerts;
    }

    /**
     * Generate alerts for vital sign abnormalities.
     */
    private void generateVitalSignAlerts(
            EnhancedRiskIndicators.RiskAssessment indicators,
            List<Alert> alerts,
            long now,
            String patientId) throws Exception {

        // Tachycardia alert with severity
        if (indicators.isTachycardia()) {
            String severity = mapTachycardiaSeverity(indicators.getTachycardiaSeverity());
            Alert alert = new Alert(
                "TACHYCARDIA",
                severity,
                String.format("Heart rate elevated (%s severity). Current HR: %d bpm",
                    indicators.getTachycardiaSeverity(),
                    indicators.getCurrentHeartRate()),
                now
            );

            // Add action based on severity
            if ("CRITICAL".equals(severity)) {
                alert.setActionRequired("IMMEDIATE: ECG and cardiac monitoring required");
            } else if ("HIGH".equals(severity)) {
                alert.setActionRequired("Order ECG within 1 hour, consider cardiology consult");
            } else {
                alert.setActionRequired("Monitor closely, check for underlying causes");
            }

            alerts.add(alert);
        }

        // Bradycardia alert
        if (indicators.isBradycardia()) {
            String severity = mapBradycardiaSeverity(indicators.getBradycardiaSeverity());
            Alert alert = new Alert(
                "BRADYCARDIA",
                severity,
                String.format("Heart rate low (%s severity). Current HR: %d bpm",
                    indicators.getBradycardiaSeverity(),
                    indicators.getCurrentHeartRate()),
                now
            );
            alert.setActionRequired("Review medications, check for heart block");
            alerts.add(alert);
        }

        // Hypertension alerts with staging
        if (indicators.isHypertension()) {
            String alertType;
            String severity;
            String message;
            String action;

            if (indicators.isHypertensionCrisis()) {
                alertType = "HYPERTENSIVE_CRISIS";
                severity = "CRITICAL";
                message = String.format("Hypertensive crisis! BP: %s",
                    indicators.getCurrentBloodPressure());
                action = "IMMEDIATE: Check for end-organ damage, consider IV antihypertensives";
            } else if (indicators.isHypertensionStage2()) {
                alertType = "HTN_STAGE2";
                severity = "HIGH";
                message = String.format("Stage 2 hypertension. BP: %s",
                    indicators.getCurrentBloodPressure());
                action = "Review medications, assess adherence, consider intensification";
            } else {
                alertType = "HTN_STAGE1";
                severity = "MEDIUM";
                message = String.format("Stage 1 hypertension. BP: %s",
                    indicators.getCurrentBloodPressure());
                action = "Lifestyle modification counseling, consider medication initiation";
            }

            Alert alert = new Alert(alertType, severity, message, now);
            alert.setActionRequired(action);
            alerts.add(alert);
        }

        // Hypotension alert
        if (indicators.isHypotension()) {
            Alert alert = new Alert(
                "HYPOTENSION",
                "HIGH",
                String.format("Low blood pressure detected. BP: %s",
                    indicators.getCurrentBloodPressure()),
                now
            );
            alert.setActionRequired("Assess for shock, check volume status, review medications");
            alerts.add(alert);
        }

        // Fever alert
        if (indicators.isFever()) {
            String severity = indicators.getCurrentTemperature() >= 39.0 ? "HIGH" : "MEDIUM";
            Alert alert = new Alert(
                "FEVER",
                severity,
                String.format("Elevated temperature: %.1f°C",
                    indicators.getCurrentTemperature()),
                now
            );
            alert.setActionRequired("Check for infection source, consider blood cultures if >38.5°C");
            alerts.add(alert);
        }

        // Hypoxia alert
        if (indicators.isHypoxia()) {
            String severity = indicators.getCurrentSpO2() < 88 ? "CRITICAL" : "HIGH";
            Alert alert = new Alert(
                "HYPOXIA",
                severity,
                String.format("Low oxygen saturation: %d%%",
                    indicators.getCurrentSpO2()),
                now
            );
            alert.setActionRequired(
                severity.equals("CRITICAL") ?
                "IMMEDIATE: Apply oxygen, check ABG, assess for respiratory failure" :
                "Apply oxygen, monitor closely, check for underlying cause"
            );
            alerts.add(alert);
        }
    }

    /**
     * Generate alerts based on acuity scores.
     */
    private void generateAcuityAlerts(
            AcuityScores scores,
            List<Alert> alerts,
            long now,
            String patientId) {

        // NEWS2 score alerts
        if (scores.getNews2Score() >= 7) {
            Alert alert = new Alert(
                "HIGH_NEWS2",
                "CRITICAL",
                String.format("HIGH NEWS2 score: %d - Immediate clinical evaluation required",
                    scores.getNews2Score()),
                now
            );
            alert.setActionRequired("IMMEDIATE: Senior clinical review, consider ICU evaluation");
            alerts.add(alert);
        } else if (scores.getNews2Score() >= 5) {
            Alert alert = new Alert(
                "ELEVATED_NEWS2",
                "HIGH",
                String.format("Elevated NEWS2 score: %d - Urgent clinical review needed",
                    scores.getNews2Score()),
                now
            );
            alert.setActionRequired("Urgent clinical review within 1 hour, increase monitoring");
            alerts.add(alert);
        }

        // Combined acuity alerts
        if (scores.getCombinedAcuityScore() >= 7) {
            Alert alert = new Alert(
                "CRITICAL_ACUITY",
                "CRITICAL",
                String.format("Critical combined acuity: %.1f - Multiple risk factors present",
                    scores.getCombinedAcuityScore()),
                now
            );
            alert.setActionRequired("Initiate continuous monitoring, notify rapid response team");
            alerts.add(alert);
        } else if (scores.getCombinedAcuityScore() >= 5) {
            Alert alert = new Alert(
                "HIGH_ACUITY",
                "HIGH",
                String.format("High combined acuity: %.1f - Close monitoring required",
                    scores.getCombinedAcuityScore()),
                now
            );
            alert.setActionRequired("Increase monitoring to hourly, senior review within 2 hours");
            alerts.add(alert);
        }
    }

    /**
     * Generate combination alerts when multiple conditions are present.
     */
    private void generateCombinationAlerts(
            EnhancedRiskIndicators.RiskAssessment indicators,
            AcuityScores scores,
            List<Alert> alerts,
            long now,
            String patientId) {

        // Tachycardia + Hypertension (cardiovascular stress)
        if (indicators.isTachycardia() && indicators.isHypertension()) {
            Alert alert = new Alert(
                "CARDIOVASCULAR_STRESS",
                "HIGH",
                "Combined tachycardia and hypertension - significant cardiovascular stress",
                now
            );
            alert.setActionRequired("ECG stat, check troponin, assess for ACS");
            alerts.add(alert);
        }

        // Tachycardia + Fever (possible sepsis)
        if (indicators.isTachycardia() && indicators.isFever()) {
            Alert alert = new Alert(
                "POSSIBLE_SEPSIS",
                "HIGH",
                "Tachycardia with fever - evaluate for sepsis",
                now
            );
            alert.setActionRequired("Check lactate, blood cultures x2, consider sepsis bundle");
            alerts.add(alert);
        }

        // Hypotension + Tachycardia (shock)
        if (indicators.isHypotension() && indicators.isTachycardia()) {
            Alert alert = new Alert(
                "POSSIBLE_SHOCK",
                "CRITICAL",
                "Hypotension with tachycardia - possible shock state",
                now
            );
            alert.setActionRequired("IMMEDIATE: IV access, fluids, assess shock type");
            alerts.add(alert);
        }

        // Multiple metabolic risks
        if (indicators.isHypertension() && indicators.hasDiabetes() &&
            scores.getMetabolicAcuityScore() > 3) {
            Alert alert = new Alert(
                "HIGH_METABOLIC_RISK",
                "MEDIUM",
                String.format("Multiple metabolic risk factors (score: %.1f)",
                    scores.getMetabolicAcuityScore()),
                now
            );
            alert.setActionRequired("Comprehensive metabolic panel, HbA1c if due, lifestyle counseling");
            alerts.add(alert);
        }
    }

    /**
     * Generate alerts based on vital sign trends.
     */
    private void generateTrendAlerts(
            EnhancedRiskIndicators.RiskAssessment indicators,
            List<Alert> alerts,
            long now,
            String patientId) {

        // Deteriorating trend alerts
        if ("DETERIORATING".equals(indicators.getHeartRateTrend())) {
            Alert alert = new Alert(
                "HR_DETERIORATING",
                "HIGH",
                "Heart rate trend deteriorating over last 4 hours",
                now
            );
            alert.setActionRequired("Review trend data, identify underlying cause");
            alerts.add(alert);
        }

        if ("DETERIORATING".equals(indicators.getBloodPressureTrend())) {
            Alert alert = new Alert(
                "BP_DETERIORATING",
                "HIGH",
                "Blood pressure trend deteriorating",
                now
            );
            alert.setActionRequired("Assess for shock, bleeding, medication effects");
            alerts.add(alert);
        }

        if ("DETERIORATING".equals(indicators.getOxygenSaturationTrend())) {
            Alert alert = new Alert(
                "SPO2_DETERIORATING",
                "HIGH",
                "Oxygen saturation trend deteriorating",
                now
            );
            alert.setActionRequired("Increase O2, check ABG, CXR if indicated");
            alerts.add(alert);
        }
    }

    /**
     * Generate data quality alerts for missing critical data.
     */
    private void generateDataQualityAlerts(
            EnhancedRiskIndicators.RiskAssessment indicators,
            List<Alert> alerts,
            long now,
            String patientId) {

        // Check vital sign freshness
        if (indicators.getVitalsLastObservedTimestamp() != null) {
            long ageMinutes = (now - indicators.getVitalsLastObservedTimestamp()) / 60000;

            if (ageMinutes > 240) { // >4 hours old
                Alert alert = new Alert(
                    "STALE_VITALS",
                    "MEDIUM",
                    String.format("Vital signs are %d hours old", ageMinutes / 60),
                    now
                );
                alert.setActionRequired("Obtain current vital signs");
                alerts.add(alert);
            }
        }

        // Missing critical labs for high-risk patients
        if (indicators.hasDiabetes() && indicators.getMissingCriticalLabs() != null &&
            indicators.getMissingCriticalLabs().contains("HbA1c")) {
            Alert alert = new Alert(
                "MISSING_DIABETIC_LABS",
                "LOW",
                "HbA1c not checked in >3 months for diabetic patient",
                now
            );
            alert.setActionRequired("Order HbA1c, fasting glucose");
            alerts.add(alert);
        }
    }

    /**
     * Apply suppression logic to prevent alert fatigue.
     */
    private List<Alert> applySuppression(List<Alert> alerts, String patientId) throws Exception {
        List<Alert> finalAlerts = new ArrayList<>();

        for (Alert alert : alerts) {
            String alertKey = alert.getType();
            Long lastAlertTime = lastAlertTimestamps.get(alertKey);
            Integer count = alertCounts.get(alertKey);

            if (count == null) count = 0;

            // Determine suppression window based on severity
            long suppressionWindow = getSuppressionWindow(alert.getSeverity());

            // Check if alert should be suppressed
            boolean shouldSuppress = false;
            if (lastAlertTime != null) {
                long timeSinceLastAlert = alert.getTimestamp() - lastAlertTime;
                shouldSuppress = timeSinceLastAlert < suppressionWindow;

                // Override suppression for critical alerts that are escalating
                if (shouldSuppress && "CRITICAL".equals(alert.getSeverity())) {
                    // Allow critical alert if it's been at least 30 minutes
                    shouldSuppress = timeSinceLastAlert < SUPPRESSION_WINDOW_CRITICAL_MS;
                }
            }

            if (!shouldSuppress) {
                // Update tracking state
                lastAlertTimestamps.put(alertKey, alert.getTimestamp());
                alertCounts.put(alertKey, count + 1);
                alert.setOccurrenceCount(count + 1);

                finalAlerts.add(alert);

                LOG.debug("Alert generated for patient {}: {} ({})",
                    patientId, alert.getType(), alert.getSeverity());
            } else {
                // Log suppressed alert
                LOG.debug("Alert suppressed for patient {}: {} (within {} ms window)",
                    patientId, alert.getType(), suppressionWindow);
            }
        }

        return finalAlerts;
    }

    /**
     * Get suppression window based on alert severity.
     */
    private long getSuppressionWindow(String severity) {
        switch (severity) {
            case "CRITICAL":
                return SUPPRESSION_WINDOW_CRITICAL_MS;
            case "HIGH":
                return SUPPRESSION_WINDOW_HIGH_MS;
            case "MEDIUM":
                return SUPPRESSION_WINDOW_MEDIUM_MS;
            case "LOW":
                return SUPPRESSION_WINDOW_LOW_MS;
            default:
                return SUPPRESSION_WINDOW_MEDIUM_MS;
        }
    }

    /**
     * Map tachycardia severity to alert severity.
     */
    private String mapTachycardiaSeverity(String clinicalSeverity) {
        if ("SEVERE".equals(clinicalSeverity)) return "CRITICAL";
        if ("MODERATE".equals(clinicalSeverity)) return "HIGH";
        if ("MILD".equals(clinicalSeverity)) return "MEDIUM";
        return "LOW";
    }

    /**
     * Map bradycardia severity to alert severity.
     */
    private String mapBradycardiaSeverity(String clinicalSeverity) {
        if ("SEVERE".equals(clinicalSeverity)) return "HIGH";
        if ("MILD".equals(clinicalSeverity)) return "MEDIUM";
        return "LOW";
    }

    /**
     * Get runtime context for state access.
     * Note: This needs to be injected by the ProcessFunction using this generator.
     */
    private RuntimeContext getRuntimeContext() {
        // This would be injected by the containing ProcessFunction
        // For now, returning null - actual implementation would have this injected
        return null;
    }

    // Placeholder for RuntimeContext interface
    private interface RuntimeContext {
        <K, V> MapState<K, V> getMapState(MapStateDescriptor<K, V> descriptor);
    }
}