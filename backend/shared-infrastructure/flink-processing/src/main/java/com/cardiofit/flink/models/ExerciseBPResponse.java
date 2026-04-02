package com.cardiofit.flink.models;

public enum ExerciseBPResponse {

    NORMAL,
    EXAGGERATED,
    HYPOTENSIVE_RESPONSE,
    POST_EXERCISE_HYPOTENSION,
    INCOMPLETE;

    public static final double EXAGGERATED_RISE_THRESHOLD = 60.0;
    public static final double EXAGGERATED_PEAK_THRESHOLD = 210.0;
    public static final double HYPOTENSIVE_DROP_THRESHOLD = -20.0;
    public static final double PEH_DROP_THRESHOLD = -5.0;

    public static ExerciseBPResponse classify(Double preSBP, Double peakSBP, Double postSBP) {
        if (preSBP == null || (peakSBP == null && postSBP == null)) {
            return INCOMPLETE;
        }
        if (peakSBP != null) {
            double rise = peakSBP - preSBP;
            if (rise > EXAGGERATED_RISE_THRESHOLD || peakSBP >= EXAGGERATED_PEAK_THRESHOLD) {
                return EXAGGERATED;
            }
        }
        if (postSBP != null) {
            double postDelta = postSBP - preSBP;
            if (postDelta < HYPOTENSIVE_DROP_THRESHOLD) {
                return HYPOTENSIVE_RESPONSE;
            }
            if (postDelta < PEH_DROP_THRESHOLD) {
                return POST_EXERCISE_HYPOTENSION;
            }
        }
        return NORMAL;
    }

    public boolean isPrognosticFlag() {
        return this == EXAGGERATED || this == HYPOTENSIVE_RESPONSE;
    }
}
