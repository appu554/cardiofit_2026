package com.cardiofit.flink.processors;

import com.cardiofit.flink.models.ClinicalAction;
import com.cardiofit.flink.models.EnrichedPatientContext;
import com.cardiofit.flink.models.PatientContextState;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.Serializable;
import java.util.*;

/**
 * Priority Assigner - Calculate priority scores and assign urgency levels
 *
 * Prioritizes clinical actions based on:
 * - Patient acuity (NEWS2, qSOFA, combined acuity score)
 * - Action urgency (STAT, URGENT, ROUTINE)
 * - Protocol priority (from protocol definition)
 * - Clinical context (active alerts, vital signs, lab values)
 * - Evidence strength (STRONG > MODERATE > WEAK)
 *
 * Priority Score Range: 0-100
 * - 90-100: CRITICAL (life-threatening, immediate intervention)
 * - 70-89: HIGH (urgent, < 1 hour)
 * - 50-69: MEDIUM (important, < 4 hours)
 * - 30-49: LOW (routine, < 24 hours)
 * - 0-29: ROUTINE (standard care)
 *
 * Urgency Levels:
 * - CRITICAL: Immediate life-saving intervention required
 * - HIGH: Urgent intervention to prevent morbidity/mortality
 * - MEDIUM: Important for optimal outcomes
 * - LOW: Routine care improvement
 * - ROUTINE: Standard monitoring/maintenance
 *
 * @author CardioFit Platform - Module 3
 * @version 1.0
 * @since 2025-10-20
 */
public class PriorityAssigner implements Serializable {
    private static final long serialVersionUID = 1L;
    private static final Logger LOG = LoggerFactory.getLogger(PriorityAssigner.class);

    /**
     * Prioritize actions by calculating priority scores and sorting
     *
     * @param actions List of clinical actions to prioritize
     * @return List of actions sorted by priority (highest first)
     */
    public List<ClinicalAction> prioritize(List<ClinicalAction> actions) {
        if (actions == null || actions.isEmpty()) {
            return new ArrayList<>();
        }

        // Actions are already built but need priority scoring
        // In production, this would use context passed to buildActions
        // For now, sort by urgency and action type

        List<ClinicalAction> prioritizedActions = new ArrayList<>(actions);

        // Sort by urgency first, then by action type
        prioritizedActions.sort((a1, a2) -> {
            int urgencyCompare = compareUrgency(a1.getUrgency(), a2.getUrgency());
            if (urgencyCompare != 0) {
                return urgencyCompare;
            }

            // If urgency equal, therapeutic actions first
            if (a1.isTherapeutic() && !a2.isTherapeutic()) {
                return -1;
            } else if (!a1.isTherapeutic() && a2.isTherapeutic()) {
                return 1;
            }

            // Finally, by sequence order
            return Integer.compare(a1.getSequenceOrder(), a2.getSequenceOrder());
        });

        return prioritizedActions;
    }

    /**
     * Calculate priority score for a clinical action
     *
     * Priority score components:
     * 1. Base score from protocol priority (0-40 points)
     * 2. Patient acuity bonus (0-30 points)
     * 3. Action urgency multiplier (0-20 points)
     * 4. Evidence strength bonus (0-10 points)
     *
     * @param protocol Protocol definition
     * @param context Patient context
     * @param confidence Protocol matching confidence
     * @return Priority score 0-100
     */
    public int calculatePriority(
            Map<String, Object> protocol,
            EnrichedPatientContext context,
            double confidence) {

        int priorityScore = 0;

        // Component 1: Protocol base priority (0-40 points)
        String protocolPriority = (String) protocol.get("priority");
        priorityScore += getProtocolPriorityPoints(protocolPriority);

        // Component 2: Patient acuity (0-30 points)
        if (context != null && context.getPatientState() != null) {
            priorityScore += getAcuityPoints(context.getPatientState());
        }

        // Component 3: Confidence multiplier (0-20 points)
        priorityScore += (int) (confidence * 20);

        // Component 4: Active alerts bonus (0-10 points)
        if (context != null && context.getPatientState() != null) {
            priorityScore += getAlertPoints(context.getPatientState());
        }

        // Cap at 100
        return Math.min(priorityScore, 100);
    }

    /**
     * Determine urgency level from priority score
     *
     * @param priority Priority score 0-100
     * @return Urgency string
     */
    public String determineUrgency(int priority) {
        if (priority >= 90) {
            return "CRITICAL";
        } else if (priority >= 70) {
            return "HIGH";
        } else if (priority >= 50) {
            return "MEDIUM";
        } else if (priority >= 30) {
            return "LOW";
        } else {
            return "ROUTINE";
        }
    }

    /**
     * Get priority points from protocol priority level
     *
     * @param protocolPriority Protocol priority string
     * @return Points 0-40
     */
    private int getProtocolPriorityPoints(String protocolPriority) {
        if (protocolPriority == null) {
            return 20; // Default to medium
        }

        String upper = protocolPriority.toUpperCase();
        if (upper.contains("CRITICAL") || upper.contains("EMERGENCY")) {
            return 40;
        } else if (upper.contains("HIGH") || upper.contains("URGENT")) {
            return 30;
        } else if (upper.contains("MEDIUM") || upper.contains("MODERATE")) {
            return 20;
        } else if (upper.contains("LOW")) {
            return 10;
        } else {
            return 20; // Default
        }
    }

    /**
     * Get priority points from patient acuity scores
     *
     * @param state Patient context state
     * @return Points 0-30
     */
    private int getAcuityPoints(PatientContextState state) {
        int points = 0;

        // NEWS2 score contribution (0-15 points)
        Integer news2 = state.getNews2Score();
        if (news2 != null) {
            if (news2 >= 7) {
                points += 15; // Critical
            } else if (news2 >= 5) {
                points += 10; // High
            } else if (news2 >= 3) {
                points += 5; // Medium
            }
        }

        // qSOFA score contribution (0-10 points)
        Integer qsofa = state.getQsofaScore();
        if (qsofa != null) {
            if (qsofa >= 2) {
                points += 10; // Sepsis risk
            } else if (qsofa >= 1) {
                points += 5; // Elevated risk
            }
        }

        // Combined acuity score contribution (0-5 points)
        Double combinedAcuity = state.getCombinedAcuityScore();
        if (combinedAcuity != null) {
            if (combinedAcuity > 7.0) {
                points += 5;
            } else if (combinedAcuity > 5.0) {
                points += 3;
            }
        }

        return Math.min(points, 30); // Cap at 30
    }

    /**
     * Get priority points from active alerts
     *
     * @param state Patient context state
     * @return Points 0-10
     */
    private int getAlertPoints(PatientContextState state) {
        if (state.getActiveAlerts() == null || state.getActiveAlerts().isEmpty()) {
            return 0;
        }

        int alertCount = state.getActiveAlerts().size();

        if (alertCount >= 5) {
            return 10; // Multiple critical alerts
        } else if (alertCount >= 3) {
            return 7; // Several alerts
        } else if (alertCount >= 1) {
            return 5; // Some alerts
        }

        return 0;
    }

    /**
     * Compare urgency levels for sorting
     *
     * @param urgency1 First urgency
     * @param urgency2 Second urgency
     * @return Negative if urgency1 > urgency2 (higher urgency first)
     */
    private int compareUrgency(String urgency1, String urgency2) {
        int priority1 = getUrgencyValue(urgency1);
        int priority2 = getUrgencyValue(urgency2);
        return Integer.compare(priority2, priority1); // Reverse for descending
    }

    /**
     * Get numeric value for urgency level
     *
     * @param urgency Urgency string
     * @return Numeric priority (higher = more urgent)
     */
    private int getUrgencyValue(String urgency) {
        if (urgency == null) {
            return 0;
        }

        String upper = urgency.toUpperCase();
        if (upper.contains("STAT") || upper.contains("CRITICAL")) {
            return 5;
        } else if (upper.contains("URGENT") || upper.contains("HIGH")) {
            return 4;
        } else if (upper.contains("MEDIUM") || upper.contains("MODERATE")) {
            return 3;
        } else if (upper.contains("LOW")) {
            return 2;
        } else if (upper.contains("ROUTINE")) {
            return 1;
        }
        return 0;
    }

    /**
     * Get recommended timeframe based on priority score
     *
     * @param priority Priority score 0-100
     * @return Timeframe string
     */
    public String getRecommendedTimeframe(int priority) {
        if (priority >= 90) {
            return "IMMEDIATE";
        } else if (priority >= 70) {
            return "<1 hour";
        } else if (priority >= 50) {
            return "<4 hours";
        } else if (priority >= 30) {
            return "<24 hours";
        } else {
            return "ROUTINE";
        }
    }

    /**
     * Generate urgency rationale explanation
     *
     * @param priority Priority score
     * @param urgency Urgency level
     * @param context Patient context
     * @return Human-readable rationale
     */
    public String generateUrgencyRationale(
            int priority,
            String urgency,
            EnrichedPatientContext context) {

        StringBuilder rationale = new StringBuilder();

        if ("CRITICAL".equals(urgency)) {
            rationale.append("Life-threatening condition requiring immediate intervention. ");
        } else if ("HIGH".equals(urgency)) {
            rationale.append("Time-sensitive condition - delay increases morbidity/mortality risk. ");
        } else if ("MEDIUM".equals(urgency)) {
            rationale.append("Important intervention for optimal patient outcomes. ");
        } else {
            rationale.append("Routine intervention for standard of care. ");
        }

        // Add context-specific details
        if (context != null && context.getPatientState() != null) {
            PatientContextState state = context.getPatientState();

            if (state.getNews2Score() != null && state.getNews2Score() >= 7) {
                rationale.append("Patient has critical NEWS2 score (").append(state.getNews2Score()).append("). ");
            }

            if (state.getQsofaScore() != null && state.getQsofaScore() >= 2) {
                rationale.append("Sepsis risk identified (qSOFA ").append(state.getQsofaScore()).append("). ");
            }

            int alertCount = state.getActiveAlerts() != null ? state.getActiveAlerts().size() : 0;
            if (alertCount >= 3) {
                rationale.append("Multiple active clinical alerts (").append(alertCount).append("). ");
            }
        }

        return rationale.toString().trim();
    }
}
