package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.ActivityResponseRecord;
import com.cardiofit.flink.models.ExerciseType;

import java.util.*;

/**
 * Exercise dose calculator for Module 11b.
 *
 * Computes weekly exercise dose metrics:
 * - Total MET-minutes (sum of METs x duration for all activities)
 * - Total active duration
 * - Per-exercise-type breakdown
 * - WHO guideline adherence check
 *
 * WHO Physical Activity Guidelines (2020):
 * - Moderate: 150-300 MET-minutes/week
 * - Vigorous: 75-150 MET-minutes/week
 * - Combined: >= 600 MET-minutes/week for substantial benefit
 *
 * Simplified benchmark: >= 150 MET-minutes/week = meets minimum.
 *
 * Stateless utility class.
 */
public class Module11bExerciseDoseCalculator {

    private static final double WHO_MODERATE_MINIMUM = 150.0; // MET-minutes/week

    private Module11bExerciseDoseCalculator() {}

    public static Result calculate(List<ActivityResponseRecord> records) {
        Result result = new Result();
        if (records == null || records.isEmpty()) {
            result.perTypeMetMinutes = Collections.emptyMap();
            return result;
        }

        double totalMetMin = 0;
        double totalDuration = 0;
        Map<ExerciseType, Double> perType = new LinkedHashMap<>();

        for (ActivityResponseRecord r : records) {
            double mm = r.getMetMinutes();
            totalMetMin += mm;
            totalDuration += r.getActivityDurationMin();
            perType.merge(r.getExerciseType(), mm, Double::sum);
        }

        result.totalMetMinutes = totalMetMin;
        result.totalDurationMin = totalDuration;
        result.activityCount = records.size();
        result.perTypeMetMinutes = perType;
        result.meetsWHOModerate = totalMetMin >= WHO_MODERATE_MINIMUM;

        return result;
    }

    public static class Result {
        public double totalMetMinutes;
        public double totalDurationMin;
        public int activityCount;
        public Map<ExerciseType, Double> perTypeMetMinutes;
        public boolean meetsWHOModerate;
    }
}
