package com.cardiofit.flink.indicators;

import com.cardiofit.flink.models.PatientSnapshot;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.time.Instant;
import java.util.*;

/**
 * Enhanced Risk Indicators with Severity Levels and Freshness Tracking
 *
 * Provides comprehensive risk assessment across multiple clinical domains:
 * - Cardiac risk (tachycardia, bradycardia with severity levels)
 * - Hypertension staging (Stage 1, Stage 2, Crisis)
 * - Vitals freshness (how recent are measurements)
 * - Trend analysis (improving, stable, deteriorating)
 *
 * Based on MODULE2_ADVANCED_ENHANCEMENTS.md Phase 1, Item 1
 */
public class EnhancedRiskIndicators implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(EnhancedRiskIndicators.class);

    // Thresholds for cardiac risk
    private static final int TACHYCARDIA_MILD = 100;
    private static final int TACHYCARDIA_MODERATE = 110;
    private static final int TACHYCARDIA_SEVERE = 120;
    private static final int BRADYCARDIA_MILD = 60;
    private static final int BRADYCARDIA_MODERATE = 50;
    private static final int BRADYCARDIA_SEVERE = 40;

    // Thresholds for blood pressure
    private static final int HTN_STAGE1_SYSTOLIC = 130;
    private static final int HTN_STAGE1_DIASTOLIC = 80;
    private static final int HTN_STAGE2_SYSTOLIC = 140;
    private static final int HTN_STAGE2_DIASTOLIC = 90;
    private static final int HTN_CRISIS_SYSTOLIC = 180;
    private static final int HTN_CRISIS_DIASTOLIC = 120;

    // Freshness thresholds (in milliseconds)
    private static final long FRESH_THRESHOLD = 4 * 60 * 60 * 1000; // 4 hours
    private static final long STALE_THRESHOLD = 24 * 60 * 60 * 1000; // 24 hours

    /**
     * Calculate comprehensive risk indicators for a patient
     */
    public static RiskAssessment assessRisk(PatientSnapshot snapshot, Map<String, Object> currentVitals) {
        RiskAssessment assessment = new RiskAssessment();
        assessment.setPatientId(snapshot.getPatientId());
        assessment.setTimestamp(System.currentTimeMillis());

        // Assess cardiac risks
        assessCardiacRisk(assessment, currentVitals);

        // Assess blood pressure risks
        assessBloodPressureRisk(assessment, currentVitals);

        // Assess vital signs freshness
        assessVitalsFreshness(assessment, currentVitals);

        // Assess trends if historical data available
        assessTrends(assessment, snapshot, currentVitals);

        // Calculate overall risk score
        calculateOverallRisk(assessment);

        LOG.debug("Risk assessment for patient {}: overall={}, cardiac={}, bp={}",
            snapshot.getPatientId(), assessment.getOverallRiskLevel(),
            assessment.getCardiacRisk(), assessment.getBloodPressureRisk());

        return assessment;
    }

    /**
     * Assess cardiac risk (tachycardia and bradycardia with severity)
     */
    private static void assessCardiacRisk(RiskAssessment assessment, Map<String, Object> vitals) {
        Integer heartRate = extractInteger(vitals, "heartRate");

        if (heartRate == null) {
            assessment.setCardiacRisk(RiskLevel.UNKNOWN);
            return;
        }

        assessment.setCurrentHeartRate(heartRate);

        // Tachycardia assessment
        if (heartRate >= TACHYCARDIA_SEVERE) {
            assessment.setCardiacRisk(RiskLevel.SEVERE);
            assessment.setTachycardiaSeverity(Severity.SEVERE);
            assessment.addFinding("Severe tachycardia detected (HR: " + heartRate + " bpm)");
        } else if (heartRate >= TACHYCARDIA_MODERATE) {
            assessment.setCardiacRisk(RiskLevel.HIGH);
            assessment.setTachycardiaSeverity(Severity.MODERATE);
            assessment.addFinding("Moderate tachycardia detected (HR: " + heartRate + " bpm)");
        } else if (heartRate >= TACHYCARDIA_MILD) {
            assessment.setCardiacRisk(RiskLevel.MODERATE);
            assessment.setTachycardiaSeverity(Severity.MILD);
            assessment.addFinding("Mild tachycardia detected (HR: " + heartRate + " bpm)");
        }
        // Bradycardia assessment
        else if (heartRate <= BRADYCARDIA_SEVERE) {
            assessment.setCardiacRisk(RiskLevel.SEVERE);
            assessment.setBradycardiaSeverity(Severity.SEVERE);
            assessment.addFinding("Severe bradycardia detected (HR: " + heartRate + " bpm)");
        } else if (heartRate <= BRADYCARDIA_MODERATE) {
            assessment.setCardiacRisk(RiskLevel.HIGH);
            assessment.setBradycardiaSeverity(Severity.MODERATE);
            assessment.addFinding("Moderate bradycardia detected (HR: " + heartRate + " bpm)");
        } else if (heartRate <= BRADYCARDIA_MILD) {
            assessment.setCardiacRisk(RiskLevel.MODERATE);
            assessment.setBradycardiaSeverity(Severity.MILD);
            assessment.addFinding("Mild bradycardia detected (HR: " + heartRate + " bpm)");
        } else {
            assessment.setCardiacRisk(RiskLevel.LOW);
            assessment.setTachycardiaSeverity(Severity.NONE);
            assessment.setBradycardiaSeverity(Severity.NONE);
        }
    }

    /**
     * Assess blood pressure risk with hypertension staging
     */
    private static void assessBloodPressureRisk(RiskAssessment assessment, Map<String, Object> vitals) {
        Integer systolic = extractInteger(vitals, "systolicBP");
        Integer diastolic = extractInteger(vitals, "diastolicBP");

        if (systolic == null || diastolic == null) {
            assessment.setBloodPressureRisk(RiskLevel.UNKNOWN);
            return;
        }

        assessment.setCurrentBloodPressure(systolic + "/" + diastolic);

        // Hypertensive Crisis
        if (systolic >= HTN_CRISIS_SYSTOLIC || diastolic >= HTN_CRISIS_DIASTOLIC) {
            assessment.setBloodPressureRisk(RiskLevel.SEVERE);
            assessment.setHypertensionStage(HypertensionStage.CRISIS);
            assessment.addFinding("HYPERTENSIVE CRISIS - Immediate intervention required (BP: " +
                systolic + "/" + diastolic + ")");
        }
        // Stage 2 Hypertension
        else if (systolic >= HTN_STAGE2_SYSTOLIC || diastolic >= HTN_STAGE2_DIASTOLIC) {
            assessment.setBloodPressureRisk(RiskLevel.HIGH);
            assessment.setHypertensionStage(HypertensionStage.STAGE_2);
            assessment.addFinding("Stage 2 Hypertension (BP: " + systolic + "/" + diastolic + ")");
        }
        // Stage 1 Hypertension
        else if (systolic >= HTN_STAGE1_SYSTOLIC || diastolic >= HTN_STAGE1_DIASTOLIC) {
            assessment.setBloodPressureRisk(RiskLevel.MODERATE);
            assessment.setHypertensionStage(HypertensionStage.STAGE_1);
            assessment.addFinding("Stage 1 Hypertension (BP: " + systolic + "/" + diastolic + ")");
        }
        // Elevated
        else if (systolic >= 120) {
            assessment.setBloodPressureRisk(RiskLevel.LOW);
            assessment.setHypertensionStage(HypertensionStage.ELEVATED);
            assessment.addFinding("Elevated blood pressure (BP: " + systolic + "/" + diastolic + ")");
        }
        // Normal
        else {
            assessment.setBloodPressureRisk(RiskLevel.LOW);
            assessment.setHypertensionStage(HypertensionStage.NORMAL);
        }
    }

    /**
     * Assess freshness of vital signs
     */
    private static void assessVitalsFreshness(RiskAssessment assessment, Map<String, Object> vitals) {
        Long timestamp = extractLong(vitals, "timestamp");

        if (timestamp == null) {
            assessment.setVitalsFreshness(Freshness.UNKNOWN);
            return;
        }

        long age = System.currentTimeMillis() - timestamp;

        if (age < FRESH_THRESHOLD) {
            assessment.setVitalsFreshness(Freshness.FRESH);
            assessment.setVitalsAgeMinutes((int) (age / (60 * 1000)));
        } else if (age < STALE_THRESHOLD) {
            assessment.setVitalsFreshness(Freshness.MODERATE);
            assessment.setVitalsAgeMinutes((int) (age / (60 * 1000)));
            assessment.addFinding("Vitals are " + (age / (60 * 60 * 1000)) + " hours old");
        } else {
            assessment.setVitalsFreshness(Freshness.STALE);
            assessment.setVitalsAgeMinutes((int) (age / (60 * 1000)));
            assessment.addFinding("Vitals are STALE (>" + (age / (60 * 60 * 1000)) + " hours old)");
        }
    }

    /**
     * Assess trends in vital signs
     */
    private static void assessTrends(RiskAssessment assessment, PatientSnapshot snapshot,
                                     Map<String, Object> currentVitals) {
        // Extract historical vitals from snapshot
        Map<String, Object> previousVitals = extractPreviousVitals(snapshot);

        if (previousVitals == null || previousVitals.isEmpty()) {
            assessment.setTrendDirection(TrendDirection.UNKNOWN);
            return;
        }

        // Compare current vs previous heart rate
        Integer currentHR = extractInteger(currentVitals, "heartRate");
        Integer previousHR = extractInteger(previousVitals, "heartRate");

        if (currentHR != null && previousHR != null) {
            int hrChange = currentHR - previousHR;

            if (Math.abs(hrChange) > 10) {
                if (hrChange > 0) {
                    assessment.setTrendDirection(TrendDirection.DETERIORATING);
                    assessment.addFinding("Heart rate increasing trend: " + previousHR + " → " + currentHR);
                } else {
                    assessment.setTrendDirection(TrendDirection.IMPROVING);
                    assessment.addFinding("Heart rate decreasing trend: " + previousHR + " → " + currentHR);
                }
            } else {
                assessment.setTrendDirection(TrendDirection.STABLE);
            }
        }

        // Compare blood pressure trends
        Integer currentSBP = extractInteger(currentVitals, "systolicBP");
        Integer previousSBP = extractInteger(previousVitals, "systolicBP");

        if (currentSBP != null && previousSBP != null) {
            int bpChange = currentSBP - previousSBP;

            if (Math.abs(bpChange) > 15) {
                if (bpChange > 0) {
                    assessment.addFinding("Blood pressure increasing trend: " +
                        previousSBP + " → " + currentSBP);
                } else {
                    assessment.addFinding("Blood pressure decreasing trend: " +
                        previousSBP + " → " + currentSBP);
                }
            }
        }
    }

    /**
     * Calculate overall risk level based on all assessments
     */
    private static void calculateOverallRisk(RiskAssessment assessment) {
        // Take the highest risk level from all categories
        RiskLevel maxRisk = RiskLevel.LOW;

        if (assessment.getCardiacRisk() != null &&
            assessment.getCardiacRisk().ordinal() > maxRisk.ordinal()) {
            maxRisk = assessment.getCardiacRisk();
        }

        if (assessment.getBloodPressureRisk() != null &&
            assessment.getBloodPressureRisk().ordinal() > maxRisk.ordinal()) {
            maxRisk = assessment.getBloodPressureRisk();
        }

        assessment.setOverallRiskLevel(maxRisk);

        // Add summary finding
        if (maxRisk == RiskLevel.SEVERE) {
            assessment.addFinding("CRITICAL: Immediate clinical intervention required");
        } else if (maxRisk == RiskLevel.HIGH) {
            assessment.addFinding("HIGH RISK: Prompt medical attention recommended");
        } else if (maxRisk == RiskLevel.MODERATE) {
            assessment.addFinding("MODERATE RISK: Close monitoring advised");
        }
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

    private static Long extractLong(Map<String, Object> map, String key) {
        Object value = map.get(key);
        if (value == null) return null;
        if (value instanceof Long) return (Long) value;
        if (value instanceof Number) return ((Number) value).longValue();
        try {
            return Long.parseLong(value.toString());
        } catch (NumberFormatException e) {
            return null;
        }
    }

    private static Map<String, Object> extractPreviousVitals(PatientSnapshot snapshot) {
        // Extract from snapshot - implementation depends on PatientSnapshot structure
        // For now, return empty map
        return new HashMap<>();
    }

    // Enums

    public enum RiskLevel {
        UNKNOWN, LOW, MODERATE, HIGH, SEVERE
    }

    public enum Severity {
        NONE, MILD, MODERATE, SEVERE
    }

    public enum HypertensionStage {
        NORMAL, ELEVATED, STAGE_1, STAGE_2, CRISIS
    }

    public enum Freshness {
        UNKNOWN, FRESH, MODERATE, STALE
    }

    public enum TrendDirection {
        UNKNOWN, IMPROVING, STABLE, DETERIORATING
    }

    // Result class

    public static class RiskAssessment implements Serializable {
        private static final long serialVersionUID = 1L;

        private String patientId;
        private long timestamp;

        // Overall assessment
        private RiskLevel overallRiskLevel;

        // Cardiac risk
        private RiskLevel cardiacRisk;
        private Integer currentHeartRate;
        private Severity tachycardiaSeverity;
        private Severity bradycardiaSeverity;

        // Blood pressure risk
        private RiskLevel bloodPressureRisk;
        private String currentBloodPressure;
        private HypertensionStage hypertensionStage;

        // Vitals freshness
        private Freshness vitalsFreshness;
        private Integer vitalsAgeMinutes;

        // Trends
        private TrendDirection trendDirection;

        // Clinical findings
        private List<String> findings = new ArrayList<>();

        // Getters and setters
        public String getPatientId() { return patientId; }
        public void setPatientId(String patientId) { this.patientId = patientId; }

        public long getTimestamp() { return timestamp; }
        public void setTimestamp(long timestamp) { this.timestamp = timestamp; }

        public RiskLevel getOverallRiskLevel() { return overallRiskLevel; }
        public void setOverallRiskLevel(RiskLevel overallRiskLevel) {
            this.overallRiskLevel = overallRiskLevel;
        }

        public RiskLevel getCardiacRisk() { return cardiacRisk; }
        public void setCardiacRisk(RiskLevel cardiacRisk) { this.cardiacRisk = cardiacRisk; }

        public Integer getCurrentHeartRate() { return currentHeartRate; }
        public void setCurrentHeartRate(Integer currentHeartRate) {
            this.currentHeartRate = currentHeartRate;
        }

        public Severity getTachycardiaSeverity() { return tachycardiaSeverity; }
        public void setTachycardiaSeverity(Severity tachycardiaSeverity) {
            this.tachycardiaSeverity = tachycardiaSeverity;
        }

        public Severity getBradycardiaSeverity() { return bradycardiaSeverity; }
        public void setBradycardiaSeverity(Severity bradycardiaSeverity) {
            this.bradycardiaSeverity = bradycardiaSeverity;
        }

        public RiskLevel getBloodPressureRisk() { return bloodPressureRisk; }
        public void setBloodPressureRisk(RiskLevel bloodPressureRisk) {
            this.bloodPressureRisk = bloodPressureRisk;
        }

        public String getCurrentBloodPressure() { return currentBloodPressure; }
        public void setCurrentBloodPressure(String currentBloodPressure) {
            this.currentBloodPressure = currentBloodPressure;
        }

        public HypertensionStage getHypertensionStage() { return hypertensionStage; }
        public void setHypertensionStage(HypertensionStage hypertensionStage) {
            this.hypertensionStage = hypertensionStage;
        }

        public Freshness getVitalsFreshness() { return vitalsFreshness; }
        public void setVitalsFreshness(Freshness vitalsFreshness) {
            this.vitalsFreshness = vitalsFreshness;
        }

        public Integer getVitalsAgeMinutes() { return vitalsAgeMinutes; }
        public void setVitalsAgeMinutes(Integer vitalsAgeMinutes) {
            this.vitalsAgeMinutes = vitalsAgeMinutes;
        }

        public TrendDirection getTrendDirection() { return trendDirection; }
        public void setTrendDirection(TrendDirection trendDirection) {
            this.trendDirection = trendDirection;
        }

        public List<String> getFindings() { return findings; }
        public void setFindings(List<String> findings) { this.findings = findings; }
        public void addFinding(String finding) { this.findings.add(finding); }

        // Convenience methods for checking specific conditions
        public boolean isTachycardia() {
            return tachycardiaSeverity != null;
        }

        public boolean isHypertensionStage1() {
            return hypertensionStage == HypertensionStage.STAGE_1;
        }

        public boolean isHypertensionStage2() {
            return hypertensionStage == HypertensionStage.STAGE_2;
        }

        public boolean isHypertensionCrisis() {
            return hypertensionStage == HypertensionStage.CRISIS;
        }

        public boolean isHypoxia() {
            return findings != null && findings.stream().anyMatch(f -> f.toLowerCase().contains("hypoxia"));
        }

        @Override
        public String toString() {
            return "RiskAssessment{" +
                    "patientId='" + patientId + '\'' +
                    ", overallRisk=" + overallRiskLevel +
                    ", cardiacRisk=" + cardiacRisk +
                    ", bpRisk=" + bloodPressureRisk +
                    ", findings=" + findings.size() +
                    '}';
        }
    }
}