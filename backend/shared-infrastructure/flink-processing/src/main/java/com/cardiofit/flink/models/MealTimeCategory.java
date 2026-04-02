package com.cardiofit.flink.models;

import java.time.Instant;
import java.time.ZoneOffset;
import java.time.ZonedDateTime;

/**
 * Meal time classification based on hour of day (UTC).
 * Used for per-meal-time aggregation in Module 10b.
 */
public enum MealTimeCategory {
    BREAKFAST,  // 05:00–10:59
    LUNCH,      // 11:00–14:59
    DINNER,     // 15:00–21:59
    SNACK;      // 22:00–04:59

    public static MealTimeCategory fromTimestamp(long epochMs) {
        ZonedDateTime zdt = Instant.ofEpochMilli(epochMs).atZone(ZoneOffset.UTC);
        int hour = zdt.getHour();
        if (hour >= 5 && hour <= 10) return BREAKFAST;
        if (hour >= 11 && hour <= 14) return LUNCH;
        if (hour >= 15 && hour <= 21) return DINNER;
        return SNACK;
    }
}
