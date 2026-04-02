package com.cardiofit.flink.models;

public enum FitnessLevel {

    EXCELLENT,
    GOOD,
    AVERAGE,
    BELOW_AVERAGE,
    POOR,
    INSUFFICIENT_DATA;

    public static final int MIN_SESSIONS_FOR_ESTIMATION = 3;

    public static FitnessLevel fromVO2max(double vo2max) {
        if (vo2max >= 45.0) return EXCELLENT;
        if (vo2max >= 35.0) return GOOD;
        if (vo2max >= 25.0) return AVERAGE;
        if (vo2max >= 18.0) return BELOW_AVERAGE;
        return POOR;
    }

    public static FitnessLevel fromMETs(double mets) {
        return fromVO2max(mets * 3.5);
    }
}
