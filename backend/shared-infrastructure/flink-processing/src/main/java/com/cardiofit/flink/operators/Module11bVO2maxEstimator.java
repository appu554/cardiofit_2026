package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.FitnessLevel;

/**
 * Submaximal VO2max estimation for Module 11b.
 *
 * Uses a hybrid Astrand-Ryhming approach for streaming context:
 *   Base: VO2max = 15 × (HR_max / HR_rest)  [Uth et al., Eur J Appl Physiol 2004]
 *   Adjusted by submaximal exercise HR fraction using HR reserve.
 *
 * Validation constraints:
 * - Peak HR must be >= 60% of HR_max (submaximal effort threshold)
 * - Peak HR capped at HR_max (physiological ceiling)
 * - Resting HR defaults to 72 bpm if unavailable
 *
 * Accuracy: +/-10-15% vs direct VO2max measurement. Improves with
 * multiple sessions (averaging reduces noise from day-to-day HR variability).
 *
 * Stateless utility class.
 */
public class Module11bVO2maxEstimator {

    private static final double DEFAULT_RESTING_HR = 72.0;
    private static final double MIN_EFFORT_FRACTION = 0.60; // 60% of HR_max minimum

    private Module11bVO2maxEstimator() {}

    /**
     * Estimate VO2max from submaximal exercise HR data.
     *
     * @param peakExerciseHR peak HR observed during exercise
     * @param restingHR      resting HR (or null for default)
     * @param hrMax          age-predicted HR_max
     * @return result with VO2max estimate and fitness level, or null if insufficient effort
     */
    public static Result estimate(Double peakExerciseHR, Double restingHR, double hrMax) {
        if (peakExerciseHR == null || hrMax <= 0) return null;

        double rhr = (restingHR != null && restingHR > 30) ? restingHR : DEFAULT_RESTING_HR;

        // Validate minimum effort
        if (peakExerciseHR / hrMax < MIN_EFFORT_FRACTION) return null;
        if (peakExerciseHR > hrMax) peakExerciseHR = hrMax; // cap at HR_max

        // Astrand-Ryhming estimation: VO2max = 15 * (HR_max / HR_rest)
        double astrand = 15.0 * (hrMax / rhr);

        // Submaximal adjustment: scale by effort fraction using HR reserve
        double hrReserve = hrMax - rhr;
        if (hrReserve <= 0) hrReserve = 1.0; // guard
        double effortFraction = (peakExerciseHR - rhr) / hrReserve;

        // Hybrid estimate: Astrand base adjusted by observed effort
        // Higher effort fraction -> more reliable estimate -> less correction needed
        double vo2max = astrand * (0.8 + 0.2 * effortFraction);

        // Clamp to physiological range
        vo2max = Math.max(8.0, Math.min(80.0, vo2max));

        Result result = new Result();
        result.vo2max = vo2max;
        result.fitnessLevel = FitnessLevel.fromVO2max(vo2max);
        result.effortFraction = effortFraction;
        return result;
    }

    public static class Result {
        public double vo2max;
        public FitnessLevel fitnessLevel;
        public double effortFraction;
    }
}
