package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.*;

/**
 * Static clinical action classifier. Determines the urgency tier for a clinical event.
 *
 * Evaluation order matters: HALT conditions checked first (most dangerous),
 * then PAUSE, then SOFT_FLAG. First match wins.
 *
 * This class has NO Flink dependencies — fully unit-testable.
 */
public final class Module6ActionClassifier {

    private Module6ActionClassifier() {} // static utility

    public static ActionTier classify(ClinicalEvent event) {

        // ══ HALT conditions (immediate danger) ══

        // Vitals-based
        if (event.getNews2Score() >= 10) return ActionTier.HALT;
        if (event.getQsofaScore() >= 2 && event.hasSepsisIndicators()) return ActionTier.HALT;

        // Lab emergencies (from active alerts — invisible to vitals scoring)
        if (event.hasActiveAlert("HYPERKALEMIA", "CRITICAL")) return ActionTier.HALT;
        if (event.hasActiveAlert("ANTICOAGULATION_RISK", "CRITICAL")) return ActionTier.HALT;
        if (event.hasActiveAlert("AKI_RISK")
                && "STAGE_3".equals(event.getAlertDetail("AKI_RISK", "stage"))) return ActionTier.HALT;

        // ML predictions at critical threshold (epsilon for IEEE 754)
        if (event.hasPrediction("sepsis")
                && event.getPrediction("sepsis").getCalibratedScore() != null
                && event.getPrediction("sepsis").getCalibratedScore() >= 0.60 - 1e-9) return ActionTier.HALT;

        // Pattern escalation
        if (event.hasPattern("CLINICAL_DETERIORATION")
                && "CRITICAL".equalsIgnoreCase(event.getPattern("CLINICAL_DETERIORATION").getSeverity()))
            return ActionTier.HALT;

        // ══ PAUSE conditions (needs physician review) ══

        if (event.getNews2Score() >= 7) return ActionTier.PAUSE;
        if (event.getQsofaScore() >= 1) return ActionTier.PAUSE;

        if (event.hasActiveAlert("AKI_RISK", "HIGH")) return ActionTier.PAUSE;
        if (event.hasActiveAlert("ANTICOAGULATION_RISK", "HIGH")) return ActionTier.PAUSE;
        if (event.hasActiveAlert("BLEEDING_RISK", "HIGH")) return ActionTier.PAUSE;

        if (event.hasPrediction("deterioration")
                && event.getPrediction("deterioration").getCalibratedScore() != null
                && event.getPrediction("deterioration").getCalibratedScore() >= 0.45 - 1e-9) return ActionTier.PAUSE;
        if (event.hasPrediction("sepsis")
                && event.getPrediction("sepsis").getCalibratedScore() != null
                && event.getPrediction("sepsis").getCalibratedScore() >= 0.35 - 1e-9) return ActionTier.PAUSE;

        if (event.hasPatternWithSeverity("HIGH")) return ActionTier.PAUSE;

        // ══ SOFT_FLAG conditions (advisory) ══

        if (event.getNews2Score() >= 5) return ActionTier.SOFT_FLAG;
        if (event.hasActiveAlert("AKI_RISK", "MODERATE")) return ActionTier.SOFT_FLAG;
        if (event.hasAnyPredictionAbove(0.25)) return ActionTier.SOFT_FLAG;
        if (event.hasPatternWithSeverity("MODERATE")) return ActionTier.SOFT_FLAG;

        return ActionTier.ROUTINE;
    }
}
