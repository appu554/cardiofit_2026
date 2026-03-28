package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.SimpleAlert;
import java.util.Set;

/**
 * Clinical significance and risk level scoring for Module 4 CEP pattern matching.
 * Extracted from Module4_PatternDetection for testability.
 *
 * Maps clinical scores to CEP pattern thresholds:
 * - 0.0-0.3: Low significance (baseline candidate)
 * - 0.3-0.6: Moderate significance (warning candidate)
 * - 0.6-0.8: High significance (early deterioration)
 * - 0.8-1.0: Critical significance (severe deterioration)
 */
class Module4ClinicalScoring {

    private Module4ClinicalScoring() {}

    static double calculateClinicalSignificance(int news2Score, int qsofaScore, double acuityScore) {
        double significance = 0.0;

        // NEWS2 contribution (50% weight)
        if (news2Score >= 10) {
            significance += 0.5;
        } else if (news2Score >= 7) {
            significance += 0.4;
        } else if (news2Score >= 5) {
            significance += 0.35;
        } else if (news2Score > 0) {
            significance += 0.15;
        }

        // qSOFA contribution (30% weight)
        if (qsofaScore >= 2) {
            significance += 0.3;
        } else if (qsofaScore == 1) {
            significance += 0.15;
        }

        // Acuity contribution (20% weight)
        significance += (acuityScore / 10.0) * 0.2;

        return Math.min(1.0, Math.max(0.0, significance));
    }

    static String determineRiskLevel(int news2Score, int qsofaScore, Set<SimpleAlert> alerts) {
        if (news2Score >= 10 || qsofaScore >= 2) {
            return "high";
        }

        if (alerts != null && !alerts.isEmpty()) {
            long criticalAlertCount = alerts.stream()
                .filter(alert -> alert.getSeverity() != null)
                // BUG: getSeverity() returns AlertSeverity enum; .equals("CRITICAL") always false.
                // TODO: Fix to alert.getSeverity() == AlertSeverity.CRITICAL — tracked as tech debt
                .filter(alert -> alert.getSeverity().equals("CRITICAL") ||
                    (alert.getPriorityLevel() != null && alert.getPriorityLevel().equals("CRITICAL")))
                .count();
            if (criticalAlertCount >= 2) {
                return "high";
            }
        }

        if (news2Score >= 5 || qsofaScore >= 1) {
            return "moderate";
        }

        if (news2Score <= 4 && qsofaScore == 0) {
            return "low";
        }

        return "unknown";
    }
}
