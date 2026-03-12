package com.cardiofit.flink.alerts;

import com.cardiofit.flink.indicators.EnhancedRiskIndicators;
import com.cardiofit.flink.scoring.NEWS2Calculator;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.util.*;
import java.util.concurrent.ConcurrentHashMap;

/**
 * Smart Alert Generation with Suppression Logic
 *
 * Generates clinical alerts while preventing alert fatigue through:
 * - Time-based suppression (don't re-alert for same condition within window)
 * - Alert escalation (increase frequency for deteriorating patients)
 * - Combined alerts (group related alerts)
 * - Priority-based routing
 *
 * Based on MODULE2_ADVANCED_ENHANCEMENTS.md Phase 1, Item 3
 */
public class SmartAlertGenerator implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(SmartAlertGenerator.class);

    // ALERT SUPPRESSION DISABLED FOR PATIENT SAFETY
    //
    // Previous implementation used static shared alertHistory causing cross-patient suppression:
    // - Patient A triggers "HYPERTENSIVE_CRISIS" alert at 10:00
    // - Patient B with same condition at 10:02 gets NO alert (suppressed)
    // - Result: CRITICAL patients missed alerts
    //
    // Proper implementation requires per-patient state management in Flink:
    // - Use KeyedState with patient ID as key
    // - Maintain separate alert history per patient
    // - Prevents cross-patient interference
    //
    // For now: SUPPRESSION DISABLED to ensure all CRITICAL alerts fire

    // Suppression windows (preserved for future use with proper state management)
    private static final long CRITICAL_ALERT_WINDOW = 5 * 60 * 1000; // 5 minutes
    private static final long HIGH_ALERT_WINDOW = 15 * 60 * 1000; // 15 minutes
    private static final long MEDIUM_ALERT_WINDOW = 30 * 60 * 1000; // 30 minutes
    private static final long LOW_ALERT_WINDOW = 60 * 60 * 1000; // 1 hour

    /**
     * Generate smart alerts from risk assessment and NEWS2 score
     */
    public static List<ClinicalAlert> generateAlerts(
            String patientId,
            EnhancedRiskIndicators.RiskAssessment riskAssessment,
            NEWS2Calculator.NEWS2Score news2Score,
            Map<String, Object> currentVitals) {

        List<ClinicalAlert> alerts = new ArrayList<>();

        // Generate alerts from risk assessment
        alerts.addAll(generateRiskAlerts(patientId, riskAssessment));

        // Generate alerts from NEWS2 score with detailed vital sign breakdown
        alerts.addAll(generateNEWS2Alerts(patientId, news2Score, currentVitals));

        // Generate alerts from specific vital sign thresholds
        alerts.addAll(generateVitalSignAlerts(patientId, currentVitals));

        // SUPPRESSION DISABLED - Patient Safety Priority
        // Previous suppression caused cross-patient alert blocking
        // TODO: Implement per-patient suppression using Flink KeyedState

        // Combine related alerts (still useful to reduce noise)
        List<ClinicalAlert> combinedAlerts = combineRelatedAlerts(alerts);

        // Sort by priority (CRITICAL first)
        combinedAlerts.sort(Comparator.comparing(ClinicalAlert::getPriority));

        LOG.info("Generated {} alerts for patient {} (combined: {})",
            alerts.size(), patientId, combinedAlerts.size());

        return combinedAlerts;
    }

    /**
     * Generate alerts from risk assessment
     */
    private static List<ClinicalAlert> generateRiskAlerts(
            String patientId,
            EnhancedRiskIndicators.RiskAssessment assessment) {

        List<ClinicalAlert> alerts = new ArrayList<>();

        if (assessment == null) return alerts;

        // Critical cardiac alerts - Phase 3.2 Enhancement: Add normal ranges and clinical context
        if (assessment.getTachycardiaSeverity() == EnhancedRiskIndicators.Severity.SEVERE) {
            int hr = assessment.getCurrentHeartRate();
            int deviation = hr - 100; // Normal upper limit
            alerts.add(createAlert(
                patientId,
                "CARDIAC_TACHYCARDIA_SEVERE",
                AlertPriority.CRITICAL,
                String.format("SEVERE Tachycardia: HR %d bpm (Normal: 60-100 bpm, Deviation: +%d bpm)", hr, deviation),
                "Immediate evaluation required. Consider: ECG, cardiac monitoring, assess for arrhythmias, " +
                "check hemodynamic stability, review medications (beta-agonists, anticholinergics)",
                AlertCategory.CARDIAC
            ));
        } else if (assessment.getTachycardiaSeverity() == EnhancedRiskIndicators.Severity.MODERATE) {
            int hr = assessment.getCurrentHeartRate();
            int deviation = hr - 100;
            alerts.add(createAlert(
                patientId,
                "CARDIAC_TACHYCARDIA_MODERATE",
                AlertPriority.HIGH,
                String.format("MODERATE Tachycardia: HR %d bpm (Normal: 60-100 bpm, Deviation: +%d bpm)", hr, deviation),
                "Clinical assessment needed. Consider: Vital sign trends, patient symptoms, underlying causes " +
                "(fever, pain, anxiety, hypovolemia), need for intervention",
                AlertCategory.CARDIAC
            ));
        }

        if (assessment.getBradycardiaSeverity() == EnhancedRiskIndicators.Severity.SEVERE) {
            int hr = assessment.getCurrentHeartRate();
            int deviation = 60 - hr; // Normal lower limit
            alerts.add(createAlert(
                patientId,
                "CARDIAC_BRADYCARDIA_SEVERE",
                AlertPriority.CRITICAL,
                String.format("SEVERE Bradycardia: HR %d bpm (Normal: 60-100 bpm, Deviation: -%d bpm)", hr, deviation),
                "Immediate evaluation required. Consider: ECG, assess for heart blocks, check medications " +
                "(beta-blockers, calcium channel blockers, digoxin), evaluate for hemodynamic compromise, " +
                "prepare for potential pacing",
                AlertCategory.CARDIAC
            ));
        } else if (assessment.getBradycardiaSeverity() == EnhancedRiskIndicators.Severity.MODERATE) {
            int hr = assessment.getCurrentHeartRate();
            int deviation = 60 - hr;
            alerts.add(createAlert(
                patientId,
                "CARDIAC_BRADYCARDIA_MODERATE",
                AlertPriority.HIGH,
                String.format("MODERATE Bradycardia: HR %d bpm (Normal: 60-100 bpm, Deviation: -%d bpm)", hr, deviation),
                "Clinical assessment needed. Consider: Patient symptoms, baseline heart rate (athletic patients), " +
                "medication review, assess perfusion and hemodynamic status",
                AlertCategory.CARDIAC
            ));
        }

        // Critical blood pressure alerts - Phase 3.2 Enhancement: Add normal ranges and clinical context
        if (assessment.getHypertensionStage() == EnhancedRiskIndicators.HypertensionStage.CRISIS) {
            String bp = assessment.getCurrentBloodPressure();
            alerts.add(createAlert(
                patientId,
                "BP_HYPERTENSIVE_CRISIS",
                AlertPriority.CRITICAL,
                String.format("HYPERTENSIVE CRISIS: BP %s (Normal: <120/80 mmHg, Crisis: ≥180/120 mmHg)", bp),
                "EMERGENCY intervention required. Immediate actions: Assess for end-organ damage " +
                "(chest pain, shortness of breath, vision changes, severe headache), IV antihypertensive therapy, " +
                "continuous BP monitoring, evaluate for hypertensive encephalopathy, stroke, MI, aortic dissection",
                AlertCategory.BLOOD_PRESSURE
            ));
        } else if (assessment.getHypertensionStage() == EnhancedRiskIndicators.HypertensionStage.STAGE_2) {
            String bp = assessment.getCurrentBloodPressure();
            alerts.add(createAlert(
                patientId,
                "BP_STAGE2_HYPERTENSION",
                AlertPriority.HIGH,
                String.format("Stage 2 Hypertension: BP %s (Normal: <120/80 mmHg, Stage 2: ≥140/90 mmHg)", bp),
                "Medical intervention recommended. Consider: Repeat measurement to confirm, assess for symptoms, " +
                "review medication compliance, evaluate cardiovascular risk factors, initiate or adjust " +
                "antihypertensive therapy per guidelines",
                AlertCategory.BLOOD_PRESSURE
            ));
        }

        // Vitals freshness alerts
        if (assessment.getVitalsFreshness() == EnhancedRiskIndicators.Freshness.STALE) {
            alerts.add(createAlert(
                patientId,
                "VITALS_STALE",
                AlertPriority.MEDIUM,
                "Stale vital signs",
                "Last vitals recorded " + assessment.getVitalsAgeMinutes() + " minutes ago - Update needed",
                AlertCategory.DATA_QUALITY
            ));
        }

        // Deterioration trend alerts
        if (assessment.getTrendDirection() == EnhancedRiskIndicators.TrendDirection.DETERIORATING) {
            alerts.add(createAlert(
                patientId,
                "TREND_DETERIORATING",
                AlertPriority.HIGH,
                "Patient condition deteriorating",
                "Clinical parameters showing worsening trend - Increased monitoring recommended",
                AlertCategory.TRENDING
            ));
        }

        return alerts;
    }

    /**
     * Generate alerts from NEWS2 score with detailed vital sign breakdown
     * Phase 3 Enhancement: Adds specific vital values and contribution scores
     */
    private static List<ClinicalAlert> generateNEWS2Alerts(
            String patientId,
            NEWS2Calculator.NEWS2Score news2Score,
            Map<String, Object> currentVitals) {

        List<ClinicalAlert> alerts = new ArrayList<>();

        if (news2Score == null) return alerts;

        int totalScore = news2Score.getTotalScore();

        // High risk (score ≥7)
        if (totalScore >= 7) {
            String detailedMessage = buildDetailedNEWS2Message(
                news2Score,
                currentVitals,
                "CRITICAL: NEWS2 Score " + totalScore + " - Multiple Physiological Abnormalities"
            );

            alerts.add(createAlert(
                patientId,
                "NEWS2_HIGH_RISK",
                AlertPriority.CRITICAL,
                detailedMessage,
                news2Score.getRecommendedResponse() + " - " + news2Score.getResponseFrequency(),
                AlertCategory.ACUITY
            ));
        }
        // Medium risk (score 5-6)
        else if (totalScore >= 5) {
            String detailedMessage = buildDetailedNEWS2Message(
                news2Score,
                currentVitals,
                "URGENT: NEWS2 Score " + totalScore + " - Physiological Abnormalities Detected"
            );

            alerts.add(createAlert(
                patientId,
                "NEWS2_MEDIUM_RISK",
                AlertPriority.HIGH,
                detailedMessage,
                news2Score.getRecommendedResponse() + " - " + news2Score.getResponseFrequency(),
                AlertCategory.ACUITY
            ));
        }
        // Low-medium risk (red score present)
        else if ("LOW-MEDIUM".equals(news2Score.getRiskLevel())) {
            String detailedMessage = buildRedScoreMessage(news2Score, currentVitals);

            alerts.add(createAlert(
                patientId,
                "NEWS2_RED_SCORE",
                AlertPriority.HIGH,
                detailedMessage,
                "One parameter critically abnormal - " + news2Score.getRecommendedResponse(),
                AlertCategory.ACUITY
            ));
        }

        return alerts;
    }

    /**
     * Build detailed NEWS2 message with vital sign breakdown
     * Shows which vitals contribute ≥2 points to the score
     */
    private static String buildDetailedNEWS2Message(
            NEWS2Calculator.NEWS2Score news2Score,
            Map<String, Object> vitals,
            String prefix) {

        StringBuilder message = new StringBuilder(prefix);
        message.append(". Contributing factors: ");

        List<String> contributors = new ArrayList<>();

        // Respiratory Rate
        if (news2Score.getRespiratoryRateScore() >= 2) {
            Integer rr = extractValue(vitals, "respiratoryRate");
            contributors.add(String.format("RR=%s (+%d)",
                rr != null ? rr : "N/A",
                news2Score.getRespiratoryRateScore()));
        }

        // Heart Rate
        if (news2Score.getHeartRateScore() >= 2) {
            Integer hr = extractValue(vitals, "heartRate");
            contributors.add(String.format("HR=%s (+%d)",
                hr != null ? hr : "N/A",
                news2Score.getHeartRateScore()));
        }

        // Oxygen Saturation
        if (news2Score.getOxygenSaturationScore() >= 2) {
            Integer spo2 = extractValue(vitals, "oxygenSaturation");
            contributors.add(String.format("SpO2=%s%% (+%d)",
                spo2 != null ? spo2 : "N/A",
                news2Score.getOxygenSaturationScore()));
        }

        // Systolic BP
        if (news2Score.getSystolicBPScore() >= 2) {
            Integer sbp = extractValue(vitals, "systolicBP");
            contributors.add(String.format("SBP=%s (+%d)",
                sbp != null ? sbp : "N/A",
                news2Score.getSystolicBPScore()));
        }

        // Consciousness
        if (news2Score.getConsciousnessScore() >= 2) {
            String consciousness = extractStringValue(vitals, "consciousness");
            contributors.add(String.format("Consciousness=%s (+%d)",
                consciousness != null ? consciousness : "Altered",
                news2Score.getConsciousnessScore()));
        }

        // Temperature
        if (news2Score.getTemperatureScore() >= 2) {
            Double temp = extractDoubleValue(vitals, "temperature");
            contributors.add(String.format("Temp=%.1f°C (+%d)",
                temp != null ? temp : 0.0,
                news2Score.getTemperatureScore()));
        }

        // Supplemental Oxygen
        if (news2Score.getSupplementalOxygenScore() >= 2) {
            contributors.add(String.format("O2 Therapy (+%d)",
                news2Score.getSupplementalOxygenScore()));
        }

        if (contributors.isEmpty()) {
            message.append("Multiple minor abnormalities");
        } else {
            message.append(String.join(", ", contributors));
        }

        return message.toString();
    }

    /**
     * Build message for red score alerts (single parameter = 3 points)
     */
    private static String buildRedScoreMessage(
            NEWS2Calculator.NEWS2Score news2Score,
            Map<String, Object> vitals) {

        StringBuilder message = new StringBuilder("NEWS2 Red Score: Critical single parameter - ");

        // Identify which parameter has red score
        if (news2Score.getRespiratoryRateScore() == 3) {
            Integer rr = extractValue(vitals, "respiratoryRate");
            message.append(String.format("Respiratory Rate %s (Critically abnormal)",
                rr != null ? rr : "N/A"));
        } else if (news2Score.getOxygenSaturationScore() == 3) {
            Integer spo2 = extractValue(vitals, "oxygenSaturation");
            message.append(String.format("Oxygen Saturation %s%% (Critically low)",
                spo2 != null ? spo2 : "N/A"));
        } else if (news2Score.getSystolicBPScore() == 3) {
            Integer sbp = extractValue(vitals, "systolicBP");
            message.append(String.format("Systolic BP %s mmHg (Critically abnormal)",
                sbp != null ? sbp : "N/A"));
        } else if (news2Score.getHeartRateScore() == 3) {
            Integer hr = extractValue(vitals, "heartRate");
            message.append(String.format("Heart Rate %s bpm (Critically abnormal)",
                hr != null ? hr : "N/A"));
        } else if (news2Score.getConsciousnessScore() == 3) {
            String consciousness = extractStringValue(vitals, "consciousness");
            message.append(String.format("Consciousness %s (Unresponsive)",
                consciousness != null ? consciousness : "Altered"));
        } else if (news2Score.getTemperatureScore() == 3) {
            Double temp = extractDoubleValue(vitals, "temperature");
            message.append(String.format("Temperature %.1f°C (Critically abnormal)",
                temp != null ? temp : 0.0));
        }

        return message.toString();
    }

    /**
     * Helper method to extract integer vital sign values
     */
    private static Integer extractValue(Map<String, Object> vitals, String key) {
        if (vitals == null) return null;
        Object value = vitals.get(key);
        if (value == null) return null;
        if (value instanceof Integer) return (Integer) value;
        if (value instanceof Number) return ((Number) value).intValue();
        try {
            return Integer.parseInt(value.toString());
        } catch (NumberFormatException e) {
            return null;
        }
    }

    /**
     * Helper method to extract double vital sign values
     */
    private static Double extractDoubleValue(Map<String, Object> vitals, String key) {
        if (vitals == null) return null;
        Object value = vitals.get(key);
        if (value == null) return null;
        if (value instanceof Double) return (Double) value;
        if (value instanceof Number) return ((Number) value).doubleValue();
        try {
            return Double.parseDouble(value.toString());
        } catch (NumberFormatException e) {
            return null;
        }
    }

    /**
     * Helper method to extract string vital sign values
     */
    private static String extractStringValue(Map<String, Object> vitals, String key) {
        if (vitals == null) return null;
        Object value = vitals.get(key);
        return value != null ? value.toString() : null;
    }

    /**
     * Generate alerts from specific vital sign thresholds
     */
    private static List<ClinicalAlert> generateVitalSignAlerts(
            String patientId,
            Map<String, Object> vitals) {

        List<ClinicalAlert> alerts = new ArrayList<>();

        // Critical oxygen saturation
        Integer spO2 = extractInteger(vitals, "oxygenSaturation");
        if (spO2 != null && spO2 < 90) {
            alerts.add(createAlert(
                patientId,
                "OXYGEN_CRITICAL",
                AlertPriority.CRITICAL,
                "Critical oxygen saturation",
                "SpO2 " + spO2 + "% - Urgent intervention required",
                AlertCategory.RESPIRATORY
            ));
        } else if (spO2 != null && spO2 < 92) {
            alerts.add(createAlert(
                patientId,
                "OXYGEN_LOW",
                AlertPriority.HIGH,
                "Low oxygen saturation",
                "SpO2 " + spO2 + "% - Assessment needed",
                AlertCategory.RESPIRATORY
            ));
        }

        // Abnormal respiratory rate
        Integer rr = extractInteger(vitals, "respiratoryRate");
        if (rr != null && (rr < 8 || rr > 30)) {
            alerts.add(createAlert(
                patientId,
                "RESPIRATORY_ABNORMAL",
                AlertPriority.CRITICAL,
                "Abnormal respiratory rate",
                "RR " + rr + " - Immediate assessment required",
                AlertCategory.RESPIRATORY
            ));
        }

        // Temperature extremes
        Double temp = extractDouble(vitals, "temperature");
        if (temp != null) {
            if (temp < 35.0) {
                alerts.add(createAlert(
                    patientId,
                    "HYPOTHERMIA",
                    AlertPriority.CRITICAL,
                    "Hypothermia detected",
                    "Temperature " + temp + "°C - Warming measures required",
                    AlertCategory.TEMPERATURE
                ));
            } else if (temp >= 39.5) {
                alerts.add(createAlert(
                    patientId,
                    "HIGH_FEVER",
                    AlertPriority.HIGH,
                    "High fever",
                    "Temperature " + temp + "°C - Antipyretic intervention",
                    AlertCategory.TEMPERATURE
                ));
            }
        }

        return alerts;
    }

    /**
     * DEPRECATED: Alert suppression disabled for patient safety
     *
     * Previous implementation caused cross-patient alert blocking.
     *
     * Proper implementation requires:
     * 1. Flink KeyedState partitioned by patient ID
     * 2. Per-patient alert history tracking
     * 3. Event-time based suppression (not processing time)
     *
     * @deprecated Use per-patient state management instead
     */
    @Deprecated
    private static List<ClinicalAlert> applySuppression(List<ClinicalAlert> alerts) {
        LOG.warn("applySuppression() is deprecated and disabled - returning all alerts");
        return alerts;
    }

    /**
     * DEPRECATED: Get suppression window based on alert priority
     * @deprecated Suppression disabled for patient safety
     */
    @Deprecated
    private static long getSuppressionWindow(AlertPriority priority) {
        switch (priority) {
            case CRITICAL:
                return CRITICAL_ALERT_WINDOW;
            case HIGH:
                return HIGH_ALERT_WINDOW;
            case MEDIUM:
                return MEDIUM_ALERT_WINDOW;
            case LOW:
            case INFO:
            default:
                return LOW_ALERT_WINDOW;
        }
    }

    /**
     * Combine related alerts to reduce alert volume
     */
    private static List<ClinicalAlert> combineRelatedAlerts(List<ClinicalAlert> alerts) {
        Map<String, List<ClinicalAlert>> alertsByCategory = new HashMap<>();

        // Group alerts by category
        for (ClinicalAlert alert : alerts) {
            alertsByCategory.computeIfAbsent(
                alert.getCategory().name(),
                k -> new ArrayList<>()
            ).add(alert);
        }

        List<ClinicalAlert> combinedAlerts = new ArrayList<>();

        // Combine alerts within same category if multiple present
        for (Map.Entry<String, List<ClinicalAlert>> entry : alertsByCategory.entrySet()) {
            List<ClinicalAlert> categoryAlerts = entry.getValue();

            if (categoryAlerts.size() > 2) {
                // Combine into single alert if 3+ alerts in same category
                ClinicalAlert combined = createCombinedAlert(categoryAlerts);
                combinedAlerts.add(combined);
            } else {
                // Keep individual alerts
                combinedAlerts.addAll(categoryAlerts);
            }
        }

        return combinedAlerts;
    }

    /**
     * Create combined alert from multiple related alerts
     */
    private static ClinicalAlert createCombinedAlert(List<ClinicalAlert> alerts) {
        // Use highest priority
        AlertPriority maxPriority = alerts.stream()
            .map(ClinicalAlert::getPriority)
            .min(Comparator.naturalOrder())
            .orElse(AlertPriority.INFO);

        String patientId = alerts.get(0).getPatientId();
        AlertCategory category = alerts.get(0).getCategory();

        StringBuilder details = new StringBuilder();
        for (ClinicalAlert alert : alerts) {
            details.append("• ").append(alert.getMessage()).append("\n");
        }

        return createAlert(
            patientId,
            "COMBINED_" + category.name(),
            maxPriority,
            "Multiple " + category.name() + " alerts",
            details.toString(),
            category
        );
    }

    /**
     * Create a clinical alert
     */
    private static ClinicalAlert createAlert(
            String patientId,
            String alertType,
            AlertPriority priority,
            String message,
            String details,
            AlertCategory category) {

        ClinicalAlert alert = new ClinicalAlert();
        alert.setAlertId(UUID.randomUUID().toString());
        alert.setPatientId(patientId);
        alert.setAlertType(alertType);
        alert.setPriority(priority);
        alert.setMessage(message);
        alert.setDetails(details);
        alert.setCategory(category);
        alert.setTimestamp(System.currentTimeMillis());
        alert.setStatus(AlertStatus.ACTIVE);

        return alert;
    }

    // Helper methods

    private static Integer extractInteger(Map<String, Object> map, String key) {
        Object value = map.get(key);
        if (value == null) return null;
        if (value instanceof Integer) return (Integer) value;
        if (value instanceof Number) return ((Number) value).intValue();
        try {
            return Integer.parseInt(value.toString());
        } catch (NumberFormatException e) {
            return null;
        }
    }

    private static Double extractDouble(Map<String, Object> map, String key) {
        Object value = map.get(key);
        if (value == null) return null;
        if (value instanceof Double) return (Double) value;
        if (value instanceof Number) return ((Number) value).doubleValue();
        try {
            return Double.parseDouble(value.toString());
        } catch (NumberFormatException e) {
            return null;
        }
    }

    // Enums

    public enum AlertPriority {
        CRITICAL(1),
        HIGH(2),
        MEDIUM(3),
        LOW(4),
        INFO(5);

        private final int level;

        AlertPriority(int level) {
            this.level = level;
        }

        public int getLevel() {
            return level;
        }
    }

    public enum AlertCategory {
        CARDIAC,
        BLOOD_PRESSURE,
        RESPIRATORY,
        TEMPERATURE,
        ACUITY,
        TRENDING,
        DATA_QUALITY,
        GENERAL
    }

    public enum AlertStatus {
        ACTIVE,
        ACKNOWLEDGED,
        RESOLVED,
        SUPPRESSED
    }

    /**
     * Clinical Alert class
     */
    public static class ClinicalAlert implements Serializable {
        private static final long serialVersionUID = 1L;

        private String alertId;
        private String patientId;
        private String alertType;
        private AlertPriority priority;
        private String message;
        private String details;
        private AlertCategory category;
        private long timestamp;
        private AlertStatus status;

        // Getters and setters
        public String getAlertId() { return alertId; }
        public void setAlertId(String alertId) { this.alertId = alertId; }

        public String getPatientId() { return patientId; }
        public void setPatientId(String patientId) { this.patientId = patientId; }

        public String getAlertType() { return alertType; }
        public void setAlertType(String alertType) { this.alertType = alertType; }

        public AlertPriority getPriority() { return priority; }
        public void setPriority(AlertPriority priority) { this.priority = priority; }

        public String getMessage() { return message; }
        public void setMessage(String message) { this.message = message; }

        public String getDetails() { return details; }
        public void setDetails(String details) { this.details = details; }

        public AlertCategory getCategory() { return category; }
        public void setCategory(AlertCategory category) { this.category = category; }

        public long getTimestamp() { return timestamp; }
        public void setTimestamp(long timestamp) { this.timestamp = timestamp; }

        public AlertStatus getStatus() { return status; }
        public void setStatus(AlertStatus status) { this.status = status; }

        @Override
        public String toString() {
            return "ClinicalAlert{" +
                    "type='" + alertType + '\'' +
                    ", priority=" + priority +
                    ", message='" + message + '\'' +
                    '}';
        }
    }
}