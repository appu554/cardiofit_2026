package com.cardiofit.flink.models;

/**
 * Eight engagement signal channels (DD#8 authoritative + V4 additions).
 * S1-S6 from DD#8 (0.80 total weight), S7-S8 from V4 architecture (0.20).
 */
public enum SignalType {
    GLUCOSE_MONITORING(0.20, "Glucose reading recorded"),
    BP_MEASUREMENT(0.20, "BP reading recorded"),
    MEDICATION_ADHERENCE(0.15, "Medication taken"),
    MEAL_LOGGING(0.15, "Meal logged"),
    APP_SESSION(0.05, "App session"),
    WEIGHT_MEASUREMENT(0.05, "Weight recorded"),
    GOAL_COMPLETION(0.10, "Goal completed"),
    APPOINTMENT_ATTENDANCE(0.10, "Appointment attended");

    private final double weight;
    private final String displayName;

    SignalType(double weight, String displayName) {
        this.weight = weight;
        this.displayName = displayName;
    }

    public double getWeight() { return weight; }
    public String getDisplayName() { return displayName; }

    /** Validate weights sum to 1.0 at class load time */
    static {
        double sum = 0.0;
        for (SignalType s : values()) sum += s.weight;
        if (Math.abs(sum - 1.0) > 1e-9) {
            throw new ExceptionInInitializerError(
                "SignalType weights must sum to 1.0, got " + sum);
        }
    }
}
