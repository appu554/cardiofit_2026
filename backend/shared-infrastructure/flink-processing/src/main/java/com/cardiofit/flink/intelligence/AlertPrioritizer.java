package com.cardiofit.flink.intelligence;

import com.cardiofit.flink.models.*;
import com.cardiofit.flink.scoring.NEWS2Calculator.NEWS2Score;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.*;

/**
 * Multi-Dimensional Alert Prioritization Engine
 *
 * Calculates priority scores (0-30 points) across 5 dimensions and assigns P-levels (P0-P4)
 * to enable intelligent alert routing, filtering, and cognitive load reduction.
 *
 * Scoring Dimensions:
 * 1. Clinical Severity (0-10 pts × 2.0 weight) - How serious is the condition?
 * 2. Time Sensitivity (0-5 pts × 1.5 weight) - How quickly must we respond?
 * 3. Patient Vulnerability (0-5 pts × 1.0 weight) - How fragile is this patient?
 * 4. Trending Pattern (0-3 pts × 1.5 weight) - Is this improving or deteriorating?
 * 5. Confidence Score (0-2 pts × 0.5 weight) - How reliable is this alert?
 *
 * Example: Sepsis Alert
 * - Clinical Severity: 9 × 2.0 = 18.0
 * - Time Sensitivity: 4 × 1.5 = 6.0
 * - Patient Vulnerability: 3 × 1.0 = 3.0
 * - Trending Pattern: 2 × 1.5 = 3.0
 * - Confidence Score: 2 × 0.5 = 1.0
 * Total: 31.0 → capped at 30.0 → P0 CRITICAL
 *
 * Reference: ALERT_PRIORITIZATION_DESIGN.md
 */
public class AlertPrioritizer {
    private static final Logger LOG = LoggerFactory.getLogger(AlertPrioritizer.class);

    // Scoring weights
    private static final double CLINICAL_SEVERITY_WEIGHT = 2.0;
    private static final double TIME_SENSITIVITY_WEIGHT = 1.5;
    private static final double PATIENT_VULNERABILITY_WEIGHT = 1.0;
    private static final double TRENDING_PATTERN_WEIGHT = 1.5;
    private static final double CONFIDENCE_SCORE_WEIGHT = 0.5;

    // Maximum priority score
    private static final double MAX_PRIORITY_SCORE = 30.0;

    /**
     * Prioritize all alerts for a patient using multi-dimensional scoring
     *
     * @param alerts Set of alerts to prioritize
     * @param state Patient context state (for vulnerability and trending analysis)
     * @param patientId Patient identifier for logging
     * @return Set of alerts with priority scores and P-levels assigned
     */
    public static Set<SimpleAlert> prioritizeAlerts(Set<SimpleAlert> alerts, PatientContextState state, String patientId) {
        if (alerts == null || alerts.isEmpty()) {
            return alerts;
        }

        LOG.info("Prioritizing {} alerts for patient {}", alerts.size(), patientId);

        for (SimpleAlert alert : alerts) {
            calculatePriority(alert, state);

            LOG.debug("Alert prioritized: {} | Score: {:.1f} | P-Level: {} | Message: {}",
                    alert.getAlertId(),
                    alert.getPriorityScore(),
                    alert.getPriorityLevel(),
                    alert.getMessage());
        }

        LOG.info("Alert prioritization complete for patient {}", patientId);
        return alerts;
    }

    /**
     * Calculate priority score and assign P-level for a single alert
     *
     * @param alert Alert to prioritize
     * @param state Patient context state
     */
    private static void calculatePriority(SimpleAlert alert, PatientContextState state) {
        // Calculate raw dimension scores
        int clinicalSeverity = calculateClinicalSeverity(alert, state);
        int timeSensitivity = calculateTimeSensitivity(alert, state);
        int patientVulnerability = calculatePatientVulnerability(state);
        int trendingPattern = calculateTrendingPattern(alert, state);
        int confidenceScore = calculateConfidenceScore(alert, state);

        // Apply weights
        double clinicalWeighted = clinicalSeverity * CLINICAL_SEVERITY_WEIGHT;
        double timeWeighted = timeSensitivity * TIME_SENSITIVITY_WEIGHT;
        double vulnerabilityWeighted = patientVulnerability * PATIENT_VULNERABILITY_WEIGHT;
        double trendingWeighted = trendingPattern * TRENDING_PATTERN_WEIGHT;
        double confidenceWeighted = confidenceScore * CONFIDENCE_SCORE_WEIGHT;

        // Calculate total priority score (capped at 30.0)
        double totalScore = clinicalWeighted + timeWeighted + vulnerabilityWeighted +
                           trendingWeighted + confidenceWeighted;
        totalScore = Math.min(totalScore, MAX_PRIORITY_SCORE);

        // Round to 1 decimal place
        totalScore = Math.round(totalScore * 10.0) / 10.0;

        // Assign P-level based on score
        AlertPriority priorityLevel = AlertPriority.fromScore(totalScore);

        // Store priority data in alert
        alert.setPriorityScore(totalScore);
        alert.setPriorityLevel(priorityLevel);

        // Store breakdown for transparency
        Map<String, Object> breakdown = new HashMap<>();
        breakdown.put("clinical_severity", clinicalSeverity);
        breakdown.put("clinical_severity_weighted", clinicalWeighted);
        breakdown.put("time_sensitivity", timeSensitivity);
        breakdown.put("time_sensitivity_weighted", timeWeighted);
        breakdown.put("patient_vulnerability", patientVulnerability);
        breakdown.put("patient_vulnerability_weighted", vulnerabilityWeighted);
        breakdown.put("trending_pattern", trendingPattern);
        breakdown.put("trending_pattern_weighted", trendingWeighted);
        breakdown.put("confidence_score", confidenceScore);
        breakdown.put("confidence_score_weighted", confidenceWeighted);
        breakdown.put("total_score", totalScore);
        breakdown.put("capped_at", MAX_PRIORITY_SCORE);

        alert.setPriorityBreakdown(breakdown);

        LOG.debug("Priority calculation for alert {}: severity={} time={} vuln={} trend={} conf={} → total={:.1f} → {}",
                alert.getAlertId(), clinicalSeverity, timeSensitivity, patientVulnerability,
                trendingPattern, confidenceScore, totalScore, priorityLevel);
    }

    /**
     * Dimension 1: Clinical Severity (0-10 points)
     *
     * Intrinsic severity of the clinical condition based on medical evidence
     * and outcome risk.
     */
    private static int calculateClinicalSeverity(SimpleAlert alert, PatientContextState state) {
        AlertType type = alert.getAlertType();
        AlertSeverity severity = alert.getSeverity();
        String message = alert.getMessage();

        if (message == null) {
            message = "";
        }

        // Life-threatening conditions (10 points)
        if (message.contains("CARDIAC ARREST") ||
            message.contains("RESPIRATORY FAILURE") ||
            message.contains("SEVERE SEPTIC SHOCK")) {
            return 10;
        }

        // Critical conditions (9 points)
        if (message.contains("SEPSIS LIKELY") &&
            state.getRiskIndicators() != null &&
            state.getRiskIndicators().isElevatedLactate()) {
            return 9;
        }

        // High acuity (8-9 points)
        // Respiratory distress gets 9 points due to rapid deterioration risk
        if (type == AlertType.RESPIRATORY_DISTRESS) {
            return 9;
        }

        // Other high acuity conditions (8 points)
        if (message.contains("SpO2 critically low") ||
            message.contains("acute renal failure") ||
            message.contains("severe hypotension")) {
            return 8;
        }

        // Serious conditions (7 points)
        if (severity == AlertSeverity.HIGH &&
            (type == AlertType.VITAL_THRESHOLD_BREACH ||
             type == AlertType.LAB_CRITICAL_VALUE)) {
            return 7;
        }

        // Moderate concern (6 points)
        if (message.contains("SIRS criteria met") ||
            message.contains("SIRS CRITERIA MET") ||
            type == AlertType.SEPSIS_PATTERN) {
            return 6;
        }

        // Borderline/threshold breach (5 points)
        if (severity == AlertSeverity.WARNING &&
            type == AlertType.VITAL_THRESHOLD_BREACH) {
            return 5;
        }

        // Deterioration pattern scoring - context-dependent
        if (type == AlertType.DETERIORATION_PATTERN) {
            // High NEWS2 (≥7) requires urgent critical care assessment
            if (severity == AlertSeverity.HIGH && message.contains("NEWS2")) {
                return 7; // Same severity as serious sepsis conditions
            }
            // Medium NEWS2 (5-6) or other deterioration patterns
            return 4;
        }

        // Informational (3 points)
        if (type == AlertType.MEDICATION || type == AlertType.MEDICATION_MISSED) {
            return 3;
        }

        // Default based on severity
        switch (severity) {
            case CRITICAL: return 9;
            case HIGH: return 7;
            case WARNING: return 5;
            case INFO: return 3;
            default: return 2;
        }
    }

    /**
     * Dimension 2: Time Sensitivity (0-5 points)
     *
     * How quickly clinical response is needed based on rate of deterioration
     * and intervention window.
     */
    private static int calculateTimeSensitivity(SimpleAlert alert, PatientContextState state) {
        String message = alert.getMessage();
        AlertType type = alert.getAlertType();

        if (message == null) {
            message = "";
        }

        // Immediate response needed (5 points)
        if (message.contains("CARDIAC ARREST") ||
            message.contains("SEVERE RESPIRATORY DISTRESS") ||
            alert.getSeverity() == AlertSeverity.CRITICAL) {
            return 5;
        }

        // Urgent response - Sepsis-3 bundle (4 points)
        if (message.contains("SEPSIS LIKELY")) {
            return 4; // SEP-1 requires intervention within 1 hour
        }

        // Prompt response needed (3 points)
        // NEWS2 ≥7 requires urgent response within 30 minutes (Royal College of Physicians guideline)
        if (type == AlertType.DETERIORATION_PATTERN && message.contains("NEWS2") && alert.getSeverity() == AlertSeverity.HIGH) {
            return 3; // Urgent ward-based response
        }

        if (type == AlertType.SEPSIS_PATTERN ||
            type == AlertType.RESPIRATORY_DISTRESS ||
            message.contains("worsening") ||
            message.contains("deteriorating") ||
            message.contains("hypotension")) {
            return 3;
        }

        // Routine response (2 points)
        if (alert.getSeverity() == AlertSeverity.WARNING ||
            type == AlertType.LAB_ABNORMALITY) {
            return 2;
        }

        // Scheduled or no urgency (1-0 points)
        if (type == AlertType.MEDICATION || type == AlertType.MEDICATION_MISSED) {
            return 1;
        }

        return 0;
    }

    /**
     * Dimension 3: Patient Vulnerability (0-5 points)
     *
     * Patient's physiological reserve and ability to compensate for stressors
     * based on age, comorbidities, and baseline acuity.
     */
    private static int calculatePatientVulnerability(PatientContextState state) {
        int vulnerabilityScore = 0;

        // Age factor from demographics
        PatientDemographics demographics = state.getDemographics();
        if (demographics != null && demographics.getAge() != null) {
            int age = demographics.getAge();
            if (age >= 75) {
                vulnerabilityScore += 2;
            } else if (age >= 65) {
                vulnerabilityScore += 1;
            }
        }

        // Chronic conditions from state
        List<Condition> conditions = state.getChronicConditions();
        if (conditions != null) {
            int chronicCount = conditions.size();
            if (chronicCount >= 3) {
                vulnerabilityScore += 2;
            } else if (chronicCount >= 1) {
                vulnerabilityScore += 1;
            }
        }

        // Risk indicators suggesting chronic conditions
        RiskIndicators risks = state.getRiskIndicators();
        if (risks != null) {
            if (risks.isHasDiabetes() || risks.isHasHeartFailure() || risks.isHasChronicKidneyDisease()) {
                vulnerabilityScore += 1;
            }
        }

        // Baseline acuity (NEWS2 reflects current physiological reserve)
        Integer news2Score = state.getNews2Score();
        if (news2Score != null && news2Score >= 5) {
            vulnerabilityScore += 1; // Already compromised baseline
        }

        return Math.min(vulnerabilityScore, 5); // Cap at 5
    }

    /**
     * Dimension 4: Trending Pattern (0-3 points)
     *
     * Direction and rate of clinical change based on recent measurements.
     */
    private static int calculateTrendingPattern(SimpleAlert alert, PatientContextState state) {
        // Check if we have historical trend data in alert context
        Map<String, Object> context = alert.getContext();
        if (context == null) {
            return 1; // Default: stable
        }

        // Check for deterioration indicators in alert context
        Object trendObj = context.get("trend");
        if (trendObj != null) {
            String trend = trendObj.toString().toUpperCase();
            if (trend.contains("RAPID_DETERIORATION") ||
                trend.contains("WORSENING")) {
                return 3;
            }
            if (trend.contains("GRADUAL_DECLINE")) {
                return 2;
            }
            if (trend.contains("IMPROVING")) {
                return 0;
            }
        }

        // Check vitals trend from state (use latest vitals)
        Map<String, Object> vitals = state.getLatestVitals();
        if (vitals != null && vitals.containsKey("trend_direction")) {
            String trendDirection = vitals.get("trend_direction").toString().toUpperCase();
            if (trendDirection.equals("DETERIORATING")) {
                return 3;
            }
            if (trendDirection.equals("WORSENING")) {
                return 2;
            }
            if (trendDirection.equals("IMPROVING")) {
                return 0;
            }
        }

        // Check message for deterioration keywords
        String message = alert.getMessage();
        if (message != null) {
            message = message.toLowerCase();
            if (message.contains("worsening") || message.contains("deteriorating")) {
                return 2;
            }
            if (message.contains("improving") || message.contains("resolving")) {
                return 0;
            }
        }

        return 1; // Default: stable
    }

    /**
     * Dimension 5: Confidence Score (0-2 points)
     *
     * Reliability of the alert based on data quality, validation, and corroboration.
     */
    private static int calculateConfidenceScore(SimpleAlert alert, PatientContextState state) {
        String sourceModule = alert.getSourceModule();
        Map<String, Object> context = alert.getContext();

        // High confidence: Multiple corroborating sources
        if (sourceModule != null && sourceModule.contains("CEP")) {
            // Complex Event Processing alerts have multiple data points
            return 2;
        }

        // Check for corroboration in context
        if (context != null && context.containsKey("corroborated_by")) {
            return 2;
        }

        // High confidence for parent alerts (consolidated from multiple sources)
        if (alert.getAlertHierarchy() != null &&
            alert.getAlertHierarchy().equals("parent") &&
            alert.getConsolidatedFrom() != null &&
            !alert.getConsolidatedFrom().isEmpty()) {
            return 2;
        }

        // NEWS2 has high confidence (composite score from multiple vital signs)
        if (alert.getAlertType() == AlertType.DETERIORATION_PATTERN &&
            alert.getMessage() != null && alert.getMessage().contains("NEWS2")) {
            return 2; // Multiple corroborating vital signs
        }

        // Standard threshold-based alerts
        if (alert.getAlertType() == AlertType.VITAL_THRESHOLD_BREACH ||
            alert.getAlertType() == AlertType.LAB_CRITICAL_VALUE) {
            return 1;
        }

        // Default moderate confidence
        return 1;
    }
}
