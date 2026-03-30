package com.cardiofit.flink.operators;

import java.util.Map;

/**
 * Clinical scoring utilities for Module 5: cooldown logic, calibration, risk classification.
 * All methods are static and testable without Flink runtime.
 */
public class Module5ClinicalScoring {

    private static final long STABLE_COOLDOWN_MS = 30_000;
    private static final long MODERATE_COOLDOWN_MS = 10_000;

    // ── Platt scaling parameters per category (trained offline) ──
    // Format: { A, B } where P(y=1) = 1 / (1 + exp(A * rawScore + B))
    private static final Map<String, double[]> PLATT_PARAMS = Map.of(
        "sepsis",         new double[]{ -2.5, 1.2 },
        "deterioration",  new double[]{ -2.0, 0.8 },
        "readmission",    new double[]{ -1.8, 0.5 },
        "fall",           new double[]{ -2.2, 1.0 },
        "mortality",      new double[]{ -2.0, 0.9 }
    );

    // ═══════════════════════════════════════════
    // Gap 3: Lab-Aware Inference Cooldown
    // ═══════════════════════════════════════════

    /**
     * Determines whether ONNX inference should run for this event.
     *
     * The cooldown tiers account for lab-only emergencies (AKI, anticoagulation,
     * hematologic) that produce NEWS2=0 with a critically ill patient.
     *
     * @param news2            Current NEWS2 score
     * @param qsofa            Current qSOFA score
     * @param riskIndicators   Risk indicator map from CDS event
     * @param lastInferenceMs  Timestamp of last inference (0 if never run)
     * @param currentTimeMs    Current processing time
     * @return true if inference should run
     */
    public static boolean shouldRunInference(
            int news2, int qsofa,
            Map<String, Object> riskIndicators,
            long lastInferenceMs, long currentTimeMs) {

        // First event for this patient — always run
        if (lastInferenceMs == 0) return true;

        long elapsed = currentTimeMs - lastInferenceMs;

        // HIGH RISK — no cooldown (every event)
        // Vitals-based
        if (news2 >= 7 || qsofa >= 2) return true;
        // Lab-based (Gap 3: invisible to NEWS2/qSOFA)
        if (hasLabCritical(riskIndicators)) return true;

        // MODERATE — 10s cooldown
        if (news2 >= 5) return elapsed >= MODERATE_COOLDOWN_MS;
        if (hasLabElevated(riskIndicators)) return elapsed >= MODERATE_COOLDOWN_MS;

        // STABLE — 30s cooldown
        return elapsed >= STABLE_COOLDOWN_MS;
    }

    /**
     * Lab-only emergencies that don't elevate NEWS2/qSOFA.
     * These patients need immediate ML prediction on every event.
     */
    public static boolean hasLabCritical(Map<String, Object> risk) {
        if (risk == null) return false;
        return Boolean.TRUE.equals(risk.get("hyperkalemia"))
            || Boolean.TRUE.equals(risk.get("severelyElevatedLactate"))
            || Boolean.TRUE.equals(risk.get("elevatedCreatinine"))
            || Boolean.TRUE.equals(risk.get("thrombocytopenia"));
    }

    /**
     * Elevated lab markers — reduce cooldown to moderate tier.
     */
    public static boolean hasLabElevated(Map<String, Object> risk) {
        if (risk == null) return false;
        return Boolean.TRUE.equals(risk.get("elevatedLactate"))
            || Boolean.TRUE.equals(risk.get("leukocytosis"))
            || Boolean.TRUE.equals(risk.get("leukopenia"));
    }

    // ═══════════════════════════════════════════
    // Platt Scaling Calibration
    // ═══════════════════════════════════════════

    /**
     * Calibrate raw ONNX output (sigmoid) to clinical probability.
     * Uses Platt scaling: P(y=1) = 1 / (1 + exp(A * rawScore + B))
     */
    public static double calibrate(double rawScore, String category) {
        double[] params = PLATT_PARAMS.get(category);
        if (params == null) return rawScore; // uncalibrated fallback
        return 1.0 / (1.0 + Math.exp(params[0] * rawScore + params[1]));
    }

    // ═══════════════════════════════════════════
    // Category-Specific Risk Classification
    // ═══════════════════════════════════════════

    /**
     * Classify calibrated score into risk level.
     * Sepsis thresholds are intentionally lower (false negatives are fatal).
     * Uses epsilon tolerance for floating-point boundary comparison (Lesson 6).
     */
    public static String classifyRiskLevel(double calibratedScore, String category) {
        double eps = 1e-9;
        return switch (category) {
            case "sepsis" -> {
                if (calibratedScore >= 0.60 - eps) yield "CRITICAL";
                if (calibratedScore >= 0.35 - eps) yield "HIGH";
                if (calibratedScore >= 0.15 - eps) yield "MODERATE";
                yield "LOW";
            }
            case "deterioration" -> {
                if (calibratedScore >= 0.70 - eps) yield "CRITICAL";
                if (calibratedScore >= 0.45 - eps) yield "HIGH";
                if (calibratedScore >= 0.20 - eps) yield "MODERATE";
                yield "LOW";
            }
            case "readmission" -> {
                if (calibratedScore >= 0.80 - eps) yield "CRITICAL";
                if (calibratedScore >= 0.55 - eps) yield "HIGH";
                if (calibratedScore >= 0.30 - eps) yield "MODERATE";
                yield "LOW";
            }
            default -> {
                if (calibratedScore >= 0.75 - eps) yield "CRITICAL";
                if (calibratedScore >= 0.50 - eps) yield "HIGH";
                if (calibratedScore >= 0.25 - eps) yield "MODERATE";
                yield "LOW";
            }
        };
    }
}
