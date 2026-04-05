package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.ClinicalStateSummary;

import java.util.*;

public final class Module13DataCompletenessMonitor {

    private static final long FRESH_WINDOW_MS = 7 * 86_400_000L;
    private static final long DECAY_WINDOW_MS = 30 * 86_400_000L;
    private static final long CRITICAL_ABSENCE_MS = 14 * 86_400_000L;
    private static final long WARNING_ABSENCE_MS = 7 * 86_400_000L;

    private static final String[] TRACKED_MODULES = {
            "module7", "module9", "module10b", "module11b", "module12", "module12b", "enriched"
    };

    private static final Map<String, Map<String, Double>> TIER_WEIGHTS = new HashMap<>();
    static {
        Map<String, Double> cgm = new HashMap<>();
        cgm.put("module7", 0.15);
        cgm.put("module9", 0.15);
        cgm.put("module10b", 0.20);
        cgm.put("module11b", 0.15);
        cgm.put("module12", 0.10);
        cgm.put("module12b", 0.10);
        cgm.put("enriched", 0.15);
        TIER_WEIGHTS.put("TIER_1_CGM", cgm);

        Map<String, Double> smbg = new HashMap<>();
        smbg.put("module7", 0.15);
        smbg.put("module9", 0.15);
        smbg.put("module10b", 0.10);
        smbg.put("module11b", 0.10);
        smbg.put("module12", 0.10);
        smbg.put("module12b", 0.10);
        smbg.put("enriched", 0.30);
        TIER_WEIGHTS.put("TIER_2_SMBG", smbg);
        TIER_WEIGHTS.put("TIER_3_SMBG", smbg);
    }

    private static final Map<String, Double> DEFAULT_WEIGHTS;
    static {
        DEFAULT_WEIGHTS = new HashMap<>();
        double equal = 1.0 / TRACKED_MODULES.length;
        for (String m : TRACKED_MODULES) DEFAULT_WEIGHTS.put(m, equal);
    }

    private Module13DataCompletenessMonitor() {}

    public static Result evaluate(ClinicalStateSummary state, long currentTimestamp) {
        Map<String, Double> weights = TIER_WEIGHTS.getOrDefault(
                state.getDataTier() != null ? state.getDataTier() : "", DEFAULT_WEIGHTS);

        Map<String, String> gapFlags = new LinkedHashMap<>();
        double weightedScore = 0.0;
        int allStaleCount = 0;
        long oldestLastSeen = Long.MAX_VALUE;

        for (String module : TRACKED_MODULES) {
            Long lastSeen = state.getModuleLastSeenMs().get(module);
            double weight = weights.getOrDefault(module, 1.0 / TRACKED_MODULES.length);

            if (lastSeen == null) {
                gapFlags.put(module, "NEVER_SEEN");
                allStaleCount++;
                continue;
            }

            long age = currentTimestamp - lastSeen;
            oldestLastSeen = Math.min(oldestLastSeen, lastSeen);

            if (age <= FRESH_WINDOW_MS) {
                weightedScore += weight * 1.0;
            } else if (age <= DECAY_WINDOW_MS) {
                double freshness = 1.0 - ((double)(age - FRESH_WINDOW_MS)
                        / (DECAY_WINDOW_MS - FRESH_WINDOW_MS));
                weightedScore += weight * Math.max(0, freshness);
                if (age > WARNING_ABSENCE_MS) {
                    gapFlags.put(module, age > CRITICAL_ABSENCE_MS ? "CRITICAL" : "WARNING");
                }
                allStaleCount++;
            } else {
                gapFlags.put(module, "EXPIRED");
                allStaleCount++;
            }
        }

        double totalWeight = 0;
        for (String m : TRACKED_MODULES) totalWeight += weights.getOrDefault(m, 1.0 / TRACKED_MODULES.length);
        double compositeScore = totalWeight > 0 ? weightedScore / totalWeight : 0.0;

        boolean absenceCritical = allStaleCount == TRACKED_MODULES.length
                || (oldestLastSeen != Long.MAX_VALUE
                    && (currentTimestamp - oldestLastSeen) > CRITICAL_ABSENCE_MS
                    && allStaleCount >= TRACKED_MODULES.length - 1);

        return new Result(Math.min(1.0, compositeScore), gapFlags, absenceCritical);
    }

    public static class Result {
        private final double compositeScore;
        private final Map<String, String> dataGapFlags;
        private final boolean dataAbsenceCritical;

        public Result(double compositeScore, Map<String, String> dataGapFlags, boolean dataAbsenceCritical) {
            this.compositeScore = compositeScore;
            this.dataGapFlags = dataGapFlags;
            this.dataAbsenceCritical = dataAbsenceCritical;
        }

        public double getCompositeScore() { return compositeScore; }
        public Map<String, String> getDataGapFlags() { return dataGapFlags; }
        public boolean isDataAbsenceCritical() { return dataAbsenceCritical; }
    }
}
