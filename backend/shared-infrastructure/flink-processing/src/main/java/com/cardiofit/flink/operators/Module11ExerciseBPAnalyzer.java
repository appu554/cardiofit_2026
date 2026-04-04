package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.ExerciseBPResponse;

/**
 * Exercise BP response analysis for Module 11.
 *
 * Classifies the blood pressure response during and after exercise using
 * pre-exercise, peak exercise, and post-exercise SBP readings.
 *
 * Classification logic (Miyai et al., Hypertension 2000; Schultz et al., JACC 2013):
 * 1. EXAGGERATED: SBP rise > 60 mmHg or peak SBP ≥ 210 mmHg
 *    → independent predictor of future hypertension and cardiovascular events
 * 2. HYPOTENSIVE_RESPONSE: post-exercise SBP drops > 20 mmHg below pre-exercise
 *    → orthostatic risk, possible autonomic dysfunction
 * 3. POST_EXERCISE_HYPOTENSION: post-exercise SBP drops 5–20 mmHg below pre-exercise
 *    → clinically beneficial, evidence of intact vascular autoregulation (Kenney & Seals 1993)
 * 4. NORMAL: SBP rise ≤ 60 mmHg and peak < 210 mmHg
 * 5. INCOMPLETE: missing pre or post readings
 *
 * Stateless utility class.
 */
public class Module11ExerciseBPAnalyzer {

    private Module11ExerciseBPAnalyzer() {}

    /**
     * Analyze exercise BP response from pre, peak, and post SBP values.
     *
     * @param preSBP   pre-exercise systolic BP (mmHg), null if unavailable
     * @param peakSBP  peak exercise systolic BP (mmHg), null if unavailable
     * @param postSBP  post-exercise systolic BP (mmHg), null if unavailable
     * @return classification result (never null)
     */
    public static Result analyze(Double preSBP, Double peakSBP, Double postSBP) {
        ExerciseBPResponse classification = ExerciseBPResponse.classify(preSBP, peakSBP, postSBP);

        Double sbpRise = null;
        if (preSBP != null && peakSBP != null) {
            sbpRise = peakSBP - preSBP;
        }

        Double postExerciseDelta = null;
        if (preSBP != null && postSBP != null) {
            postExerciseDelta = postSBP - preSBP;
        }

        Result result = new Result();
        result.bpResponse = classification;
        result.preExerciseSBP = preSBP;
        result.peakExerciseSBP = peakSBP;
        result.postExerciseSBP = postSBP;
        result.sbpRise = sbpRise;
        result.postExerciseDelta = postExerciseDelta;
        return result;
    }

    public static class Result {
        public ExerciseBPResponse bpResponse;
        public Double preExerciseSBP;
        public Double peakExerciseSBP;
        public Double postExerciseSBP;
        public Double sbpRise;
        public Double postExerciseDelta;
    }
}
