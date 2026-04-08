package com.cardiofit.flink.models;

/**
 * Time-of-day context for BP readings.
 * Critical for morning surge computation (morning vs evening differential)
 * and dipping classification (daytime vs nocturnal).
 *
 * Resolution logic:
 *   1. Explicit time_context field from device/app (preferred)
 *   2. Fallback: derive from measurement_timestamp in patient's local timezone
 *      MORNING = 06:00-10:00, AFTERNOON = 10:00-17:00,
 *      EVENING = 17:00-22:00, NIGHT = 22:00-06:00
 */
public enum TimeContext {
    MORNING,      // 06:00-10:00 - used for surge computation (numerator)
    AFTERNOON,    // 10:00-17:00
    EVENING,      // 17:00-22:00 - used for surge computation (denominator)
    NIGHT,        // 22:00-06:00 - used for dipping classification (nocturnal)
    UNKNOWN;      // insufficient metadata - excluded from surge/dip analysis

    public boolean isDaytime() {
        return this == MORNING || this == AFTERNOON || this == EVENING;
    }

    public boolean isNocturnal() {
        return this == NIGHT;
    }

    /**
     * Derive TimeContext from hour of day (0-23).
     * Used as fallback when explicit time_context is not provided.
     */
    public static TimeContext fromHour(int hour) {
        if (hour >= 6 && hour < 10) return MORNING;
        if (hour >= 10 && hour < 17) return AFTERNOON;
        if (hour >= 17 && hour < 22) return EVENING;
        return NIGHT; // 22:00-05:59
    }
}
