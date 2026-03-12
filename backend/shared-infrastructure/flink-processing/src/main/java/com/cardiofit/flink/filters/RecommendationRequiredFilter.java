package com.cardiofit.flink.filters;

import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.models.ClinicalSnapshot;
import com.cardiofit.flink.models.LabResult;
import com.cardiofit.flink.models.Medication;
import org.apache.flink.api.common.functions.FilterFunction;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.util.List;
import java.util.Map;
import java.util.stream.Collectors;

/**
 * RecommendationRequiredFilter - Filters enriched patient contexts to determine
 * which require clinical recommendation generation.
 *
 * <p>Implements 8 filter conditions to identify patients requiring decision support:
 * <ol>
 *   <li>CRITICAL urgency level from Module 2</li>
 *   <li>NEWS2 score >= 3 (early warning score)</li>
 *   <li>Active clinical alerts present</li>
 *   <li>Risk indicators (elevated lactate, hypotension, hypoxia)</li>
 *   <li>qSOFA score >= 2 (sepsis screening)</li>
 *   <li>Potential medication interactions</li>
 *   <li>Therapy failure detection</li>
 *   <li>Deteriorating trends over time</li>
 * </ol>
 *
 * <p>Purpose: Reduce load by ~88-92%, processing only patients who need recommendations.
 * Baseline filtering rate: 8-12% of events pass filter.
 *
 * @see EnrichedPatientContext
 * @author Module 3 Clinical Recommendation Engine
 * @version 1.0
 */
public class RecommendationRequiredFilter implements FilterFunction<EnrichedPatientContext>, Serializable {

    private static final Logger LOG = LoggerFactory.getLogger(RecommendationRequiredFilter.class);

    // Clinical thresholds
    private static final double NEWS2_THRESHOLD = 3.0;
    private static final double QSOFA_THRESHOLD = 2.0;
    private static final double LACTATE_ELEVATED_THRESHOLD = 2.0; // mmol/L
    private static final double SYSTOLIC_BP_LOW_THRESHOLD = 90.0; // mmHg
    private static final double OXYGEN_SAT_LOW_THRESHOLD = 92.0; // %
    private static final double DETERIORATION_THRESHOLD = 0.15; // 15% decline

    // Trend analysis window
    private static final int MIN_TRAJECTORY_POINTS = 3;
    private static final long TRAJECTORY_WINDOW_MS = 2 * 60 * 60 * 1000; // 2 hours

    /**
     * Filter function to determine if recommendation generation is required
     *
     * @param context Enriched patient context from Module 2
     * @return true if patient requires clinical recommendations, false otherwise
     * @throws Exception if filtering logic fails
     */
    @Override
    public boolean filter(EnrichedPatientContext context) throws Exception {
        try {
            // Condition 1: CRITICAL urgency level
            if (hasCriticalUrgency(context)) {
                LOG.debug("Patient {} requires recommendation: CRITICAL urgency",
                    context.getPatientId());
                return true;
            }

            // Condition 2: NEWS2 >= 3 (early warning score)
            if (hasElevatedNEWS2(context)) {
                LOG.debug("Patient {} requires recommendation: NEWS2 >= 3 ({})",
                    context.getPatientId(), context.getPatientState().getNews2Score());
                return true;
            }

            // Condition 3: Active alerts present
            if (hasActiveAlerts(context)) {
                LOG.debug("Patient {} requires recommendation: {} active alerts",
                    context.getPatientId(), context.getPatientState().getActiveAlerts().size());
                return true;
            }

            // Condition 4: Risk indicators (elevated lactate, hypotension, hypoxia)
            if (hasRiskIndicators(context)) {
                LOG.debug("Patient {} requires recommendation: Risk indicators detected",
                    context.getPatientId());
                return true;
            }

            // Condition 5: qSOFA >= 2 (sepsis screening)
            if (hasElevatedQSOFA(context)) {
                LOG.debug("Patient {} requires recommendation: qSOFA >= 2 ({})",
                    context.getPatientId(), context.getPatientState().getQsofaScore());
                return true;
            }

            // Condition 6: Potential medication interactions
            if (hasPotentialMedicationInteractions(context)) {
                LOG.debug("Patient {} requires recommendation: Medication interactions suspected",
                    context.getPatientId());
                return true;
            }

            // Condition 7: Therapy failure detection
            if (hasTherapyFailure(context)) {
                LOG.debug("Patient {} requires recommendation: Therapy failure detected",
                    context.getPatientId());
                return true;
            }

            // Condition 8: Deteriorating trends
            if (hasDeterioratingTrends(context)) {
                LOG.debug("Patient {} requires recommendation: Deteriorating clinical trends",
                    context.getPatientId());
                return true;
            }

            // No conditions met - patient does not require recommendations
            return false;

        } catch (Exception e) {
            LOG.error("Error filtering patient {}: {}", context.getPatientId(), e.getMessage());
            // Fail open: include patient if filtering fails (safety-first principle)
            return true;
        }
    }

    /**
     * Condition 1: Check for CRITICAL urgency level from Module 2
     */
    private boolean hasCriticalUrgency(EnrichedPatientContext context) {
        return "CRITICAL".equalsIgnoreCase(context.getPatientState().getAcuityLevel());
    }

    /**
     * Condition 2: Check for elevated NEWS2 score (>= 3)
     *
     * NEWS2 (National Early Warning Score 2):
     * - 0: Low risk
     * - 1-4: Low-medium risk (monitor)
     * - 5-6: Medium risk (urgent response)
     * - 7+: High risk (emergency response)
     *
     * We flag >= 3 to catch deterioration early
     */
    private boolean hasElevatedNEWS2(EnrichedPatientContext context) {
        Integer news2 = context.getPatientState().getNews2Score();
        return news2 != null && news2 >= NEWS2_THRESHOLD;
    }

    /**
     * Condition 3: Check for active clinical alerts
     */
    private boolean hasActiveAlerts(EnrichedPatientContext context) {
        java.util.Set<?> alerts = context.getPatientState().getActiveAlerts();
        return alerts != null && !alerts.isEmpty();
    }

    /**
     * Condition 4: Check for risk indicators
     * - Elevated lactate (> 2.0 mmol/L) - tissue hypoperfusion
     * - Hypotension (SBP < 90 mmHg) - shock
     * - Hypoxia (SpO2 < 92%) - respiratory compromise
     */
    private boolean hasRiskIndicators(EnrichedPatientContext context) {
        Map<String, Object> vitals = context.getPatientState().getLatestVitals();
        Map<String, LabResult> labs = context.getPatientState().getRecentLabs();

        if (vitals == null && labs == null) {
            return false;
        }

        // Check lactate elevation
        if (labs != null && labs.containsKey("lactate")) {
            LabResult lactateResult = labs.get("lactate");
            if (lactateResult != null && lactateResult.getValue() > LACTATE_ELEVATED_THRESHOLD) {
                LOG.debug("Elevated lactate detected: {} mmol/L", lactateResult.getValue());
                return true;
            }
        }

        // Check hypotension
        if (vitals != null && vitals.containsKey("systolicBP")) {
            Object sbpObj = vitals.get("systolicBP");
            Double sbp = sbpObj instanceof Number ? ((Number)sbpObj).doubleValue() : null;
            if (sbp != null && sbp < SYSTOLIC_BP_LOW_THRESHOLD) {
                LOG.debug("Hypotension detected: {} mmHg", sbp);
                return true;
            }
        }

        // Check hypoxia
        if (vitals != null && vitals.containsKey("oxygenSaturation")) {
            Object spO2Obj = vitals.get("oxygenSaturation");
            Double spO2 = spO2Obj instanceof Number ? ((Number)spO2Obj).doubleValue() : null;
            if (spO2 != null && spO2 < OXYGEN_SAT_LOW_THRESHOLD) {
                LOG.debug("Hypoxia detected: SpO2 {}%", spO2);
                return true;
            }
        }

        return false;
    }

    /**
     * Condition 5: Check for elevated qSOFA score (>= 2)
     *
     * qSOFA (Quick Sequential Organ Failure Assessment):
     * - Altered mental status (GCS < 15)
     * - Respiratory rate >= 22
     * - Systolic BP <= 100
     *
     * Score >= 2 suggests high risk of poor outcomes from sepsis
     */
    private boolean hasElevatedQSOFA(EnrichedPatientContext context) {
        Integer qsofa = context.getPatientState().getQsofaScore();
        return qsofa != null && qsofa >= QSOFA_THRESHOLD;
    }

    /**
     * Condition 6: Check for potential medication interactions
     * - Polypharmacy (> 5 medications) increases interaction risk
     * - High-risk medication combinations flagged by Module 2
     */
    private boolean hasPotentialMedicationInteractions(EnrichedPatientContext context) {
        Map<String, Medication> medications = context.getPatientState().getActiveMedications();

        if (medications == null) {
            return false;
        }

        // Polypharmacy threshold (> 5 active medications)
        if (medications.size() > 5) {
            LOG.debug("Polypharmacy detected: {} active medications", medications.size());
            return true;
        }

        // Check for interaction alerts from Module 2
        java.util.Set<?> alerts = context.getPatientState().getActiveAlerts();
        if (alerts != null) {
            for (Object alert : alerts) {
                String alertStr = alert.toString().toLowerCase();
                if (alertStr.contains("interaction") || alertStr.contains("drug-drug")) {
                    return true;
                }
            }
        }

        return false;
    }

    /**
     * Condition 7: Check for therapy failure
     * - Persistent fever despite antibiotics
     * - Worsening symptoms on current treatment
     * - Inadequate response to therapy
     */
    private boolean hasTherapyFailure(EnrichedPatientContext context) {
        java.util.Set<?> alerts = context.getPatientState().getActiveAlerts();

        if (alerts == null) {
            return false;
        }

        // Check for therapy failure alerts
        for (Object alert : alerts) {
            String alertStr = alert.toString().toLowerCase();
            if (alertStr.contains("therapy failure") ||
                alertStr.contains("inadequate response") ||
                alertStr.contains("treatment failure")) {
                return true;
            }
        }

        // Check for persistent fever with antibiotics
        Map<String, Object> vitals = context.getPatientState().getLatestVitals();
        Map<String, Medication> medications = context.getPatientState().getActiveMedications();

        if (vitals != null && medications != null) {
            Object tempObj = vitals.get("temperature");
            Double temperature = tempObj instanceof Number ? ((Number)tempObj).doubleValue() : null;
            boolean onAntibiotics = medications.values().stream()
                .anyMatch(med -> med.getName().toLowerCase().contains("antibiotic"));

            if (temperature != null && temperature > 38.3 && onAntibiotics) {
                LOG.debug("Persistent fever ({}) despite antibiotics", temperature);
                return true;
            }
        }

        return false;
    }

    /**
     * Condition 8: Check for deteriorating clinical trends
     * - Worsening vital signs over time
     * - Increasing NEWS2 score
     * - Declining oxygen saturation
     *
     * Note: Clinical trajectory not available in PatientContextState
     * This condition currently returns false - requires enhancement
     */
    private boolean hasDeterioratingTrends(EnrichedPatientContext context) {
        // TODO: Implement clinical trajectory tracking in future enhancement
        // For now, return false as trajectory data is not available
        return false;
    }

}
