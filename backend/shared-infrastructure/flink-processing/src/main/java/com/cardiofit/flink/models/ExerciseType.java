package com.cardiofit.flink.models;

public enum ExerciseType {

    AEROBIC,
    RESISTANCE,
    HIIT,
    FLEXIBILITY,
    MIXED;

    public double getDefaultMETs() {
        switch (this) {
            case AEROBIC:     return 6.0;
            case RESISTANCE:  return 5.0;
            case HIIT:        return 8.0;
            case FLEXIBILITY: return 2.5;
            case MIXED:       return 5.5;
            default:          return 4.0;
        }
    }

    public boolean expectsGlucoseSpike() {
        return this == RESISTANCE || this == HIIT;
    }

    public static ExerciseType fromString(String type) {
        if (type == null) return MIXED;
        switch (type.toUpperCase().trim().replace(" ", "_")) {
            case "AEROBIC": case "CARDIO": case "RUNNING": case "CYCLING": case "SWIMMING": case "WALKING":
                return AEROBIC;
            case "RESISTANCE": case "WEIGHTS": case "STRENGTH": case "WEIGHT_TRAINING":
                return RESISTANCE;
            case "HIIT": case "INTERVAL": case "INTERVALS": case "TABATA": case "CROSSFIT":
                return HIIT;
            case "FLEXIBILITY": case "YOGA": case "STRETCHING": case "PILATES": case "TAI_CHI":
                return FLEXIBILITY;
            default:
                return MIXED;
        }
    }
}
