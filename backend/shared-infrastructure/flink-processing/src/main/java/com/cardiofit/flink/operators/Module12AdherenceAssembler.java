package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.InterventionType;
import java.util.Map;

public final class Module12AdherenceAssembler {

    private static final double DEFAULT_ADHERENCE = 0.5;

    private Module12AdherenceAssembler() {}

    public static Result assemble(InterventionType type, Map<String, Object> signals) {
        if (signals == null || signals.isEmpty()) {
            return new Result(DEFAULT_ADHERENCE, "LOW");
        }

        String dataSource = getStringOrDefault(signals, "data_source", "");
        String quality = classifyDataQuality(dataSource, signals);

        double score;

        if (type.isMedication()) {
            score = computeMedicationAdherence(signals);
        } else if (type == InterventionType.LIFESTYLE_ACTIVITY
                || type == InterventionType.LIFESTYLE_SLEEP) {
            score = computeRatioAdherence(signals,
                    "activity_sessions_per_week", "target_sessions_per_week",
                    "mean_sleep_hours", "target_sleep_hours");
        } else if (type.isNutrition()) {
            score = computeNutritionAdherence(type, signals);
        } else {
            score = DEFAULT_ADHERENCE;
        }

        return new Result(Math.min(score, 1.0), quality);
    }

    private static double computeMedicationAdherence(Map<String, Object> signals) {
        double reminderRate = getDoubleOrDefault(signals, "reminder_ack_rate", -1);
        double refillRate = getDoubleOrDefault(signals, "refill_compliance", -1);
        if (reminderRate < 0 && refillRate < 0) return DEFAULT_ADHERENCE;
        return Math.max(
                reminderRate >= 0 ? reminderRate : 0,
                refillRate >= 0 ? refillRate : 0
        );
    }

    private static double computeRatioAdherence(Map<String, Object> signals,
                                                  String actualKey1, String targetKey1,
                                                  String actualKey2, String targetKey2) {
        double actual = getDoubleOrDefault(signals, actualKey1, -1);
        double target = getDoubleOrDefault(signals, targetKey1, -1);
        if (actual < 0 || target <= 0) {
            actual = getDoubleOrDefault(signals, actualKey2, -1);
            target = getDoubleOrDefault(signals, targetKey2, -1);
        }
        if (actual < 0 || target <= 0) return DEFAULT_ADHERENCE;
        return actual / target;
    }

    private static double computeNutritionAdherence(InterventionType type,
                                                      Map<String, Object> signals) {
        switch (type) {
            case NUTRITION_FOOD_CHANGE:
            case NUTRITION_TIMING_CHANGE: {
                double changed = getDoubleOrDefault(signals, "meals_with_prescribed_change", -1);
                double total = getDoubleOrDefault(signals, "total_meals_logged", -1);
                if (changed < 0 || total <= 0) return DEFAULT_ADHERENCE;
                return changed / total;
            }
            case NUTRITION_SODIUM_REDUCTION: {
                double achieved = getDoubleOrDefault(signals, "sodium_reduction_achieved", -1);
                double target = getDoubleOrDefault(signals, "target_reduction", -1);
                if (achieved < 0 || target <= 0) return DEFAULT_ADHERENCE;
                return achieved / target;
            }
            case NUTRITION_PORTION_CHANGE: {
                double achieved = getDoubleOrDefault(signals, "portion_reduction_achieved", -1);
                double target = getDoubleOrDefault(signals, "target_reduction", -1);
                if (achieved < 0 || target <= 0) return DEFAULT_ADHERENCE;
                return achieved / target;
            }
            default:
                return DEFAULT_ADHERENCE;
        }
    }

    private static String classifyDataQuality(String source, Map<String, Object> signals) {
        if ("WEARABLE".equalsIgnoreCase(source) || "PHARMACY".equalsIgnoreCase(source)) {
            return "HIGH";
        }
        if ("APP_SELF_REPORT".equalsIgnoreCase(source)) {
            return "MODERATE";
        }
        if (signals.containsKey("refill_compliance")
                || signals.containsKey("reminder_ack_rate")) {
            return "HIGH";
        }
        if (signals.containsKey("meals_with_prescribed_change")
                || signals.containsKey("activity_sessions_per_week")) {
            return "MODERATE";
        }
        return "LOW";
    }

    private static double getDoubleOrDefault(Map<String, Object> m, String key, double def) {
        Object v = m.get(key);
        if (v instanceof Number) return ((Number) v).doubleValue();
        return def;
    }

    private static String getStringOrDefault(Map<String, Object> m, String key, String def) {
        Object v = m.get(key);
        return v != null ? v.toString() : def;
    }

    public static class Result {
        private final double adherenceScore;
        private final String dataQuality;

        public Result(double adherenceScore, String dataQuality) {
            this.adherenceScore = adherenceScore;
            this.dataQuality = dataQuality;
        }

        public double getAdherenceScore() { return adherenceScore; }
        public String getDataQuality() { return dataQuality; }
    }
}
