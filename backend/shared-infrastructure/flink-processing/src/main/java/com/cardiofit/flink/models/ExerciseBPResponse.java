package com.cardiofit.flink.models;

public enum ExerciseBPResponse {

    NORMAL,
    EXAGGERATED,
    HYPOTENSIVE_RESPONSE,
    POST_EXERCISE_HYPOTENSION,
    INCOMPLETE;

    public static final double EXAGGERATED_RISE_THRESHOLD = 60.0;
    public static final double EXAGGERATED_PEAK_THRESHOLD_MALE = 210.0;   // Miyai et al.
    public static final double EXAGGERATED_PEAK_THRESHOLD_FEMALE = 190.0; // Miyai et al.
    public static final double HYPOTENSIVE_DROP_THRESHOLD = -20.0;
    public static final double PEH_DROP_THRESHOLD = -5.0;

    /**
     * Classify exercise BP response. Uses conservative (female) threshold when sex is unknown.
     *
     * @param preSBP  pre-exercise systolic BP (mmHg)
     * @param peakSBP peak exercise systolic BP (mmHg)
     * @param postSBP post-exercise systolic BP (mmHg)
     * @return classification
     */
    public static ExerciseBPResponse classify(Double preSBP, Double peakSBP, Double postSBP) {
        return classify(preSBP, peakSBP, postSBP, null);
    }

    /**
     * Classify exercise BP response with sex-aware thresholds (Miyai et al.).
     * Peak SBP thresholds: >= 210 mmHg (male), >= 190 mmHg (female).
     * When sex is null/unknown, uses conservative 190 mmHg threshold.
     *
     * @param preSBP     pre-exercise systolic BP (mmHg)
     * @param peakSBP    peak exercise systolic BP (mmHg)
     * @param postSBP    post-exercise systolic BP (mmHg)
     * @param patientSex "M", "F", or null for unknown (uses conservative threshold)
     * @return classification
     */
    public static ExerciseBPResponse classify(Double preSBP, Double peakSBP, Double postSBP,
                                               String patientSex) {
        if (preSBP == null || (peakSBP == null && postSBP == null)) {
            return INCOMPLETE;
        }
        if (peakSBP != null) {
            double rise = peakSBP - preSBP;
            // Use sex-aware peak threshold: conservative (female/190) when sex unknown
            double peakThreshold = "M".equalsIgnoreCase(patientSex)
                    ? EXAGGERATED_PEAK_THRESHOLD_MALE
                    : EXAGGERATED_PEAK_THRESHOLD_FEMALE;
            if (rise > EXAGGERATED_RISE_THRESHOLD || peakSBP >= peakThreshold) {
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
