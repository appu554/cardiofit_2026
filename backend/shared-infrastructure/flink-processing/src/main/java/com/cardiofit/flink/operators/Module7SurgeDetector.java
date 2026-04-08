package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.DailyBPSummary;
import java.time.Instant;
import java.time.ZoneOffset;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

/**
 * Morning BP surge detection using the sleep-trough method.
 * Surge = morning_SBP_today - evening_SBP_preceding_day
 *
 * <p>The preceding evening is found by looking back exactly 1 calendar day
 * (the evening immediately before this morning). A 48h+ gap makes the surge
 * clinically meaningless per Kario (2010) and ESH 2023.</p>
 *
 * No Flink dependencies — fully unit-testable.
 */
public final class Module7SurgeDetector {

    /** Maximum days to look back for an evening reading to pair with today's morning.
     *  Set to 1: only pair with immediately preceding evening (within ~18h). */
    static final int MAX_SURGE_LOOKBACK_DAYS = 1;

    private Module7SurgeDetector() {}

    public static Double computeTodaySurge(List<DailyBPSummary> summaries, long referenceTime) {
        return computeTodaySurge(summaries, referenceTime, MAX_SURGE_LOOKBACK_DAYS);
    }

    static Double computeTodaySurge(List<DailyBPSummary> summaries, long referenceTime,
                                     int maxLookbackDays) {
        if (summaries == null || summaries.size() < 2) return null;

        java.time.LocalDate todayDate = Instant.ofEpochMilli(referenceTime)
            .atZone(ZoneOffset.UTC).toLocalDate();
        String todayKey = todayDate.toString();

        // Build lookup map for O(1) access by date key
        Map<String, DailyBPSummary> byDate = new HashMap<>();
        for (DailyBPSummary s : summaries) {
            byDate.put(s.getDateKey(), s);
        }

        DailyBPSummary today = byDate.get(todayKey);
        if (today == null || today.getMorningAvgSBP() == null) return null;

        // Walk backwards up to maxLookbackDays to find the most recent evening reading
        for (int d = 1; d <= maxLookbackDays; d++) {
            String lookbackKey = todayDate.minusDays(d).toString();
            DailyBPSummary candidate = byDate.get(lookbackKey);
            if (candidate != null && candidate.getEveningAvgSBP() != null) {
                return today.getMorningAvgSBP() - candidate.getEveningAvgSBP();
            }
        }

        return null; // No evening reading found within lookback window
    }

    public static Double compute7DayAvgSurge(List<DailyBPSummary> summaries) {
        if (summaries == null || summaries.size() < 2) return null;

        double sumSurge = 0;
        int validPairs = 0;

        for (int i = 1; i < summaries.size(); i++) {
            DailyBPSummary today = summaries.get(i);
            DailyBPSummary prevDay = summaries.get(i - 1);

            if (today.getMorningAvgSBP() != null && prevDay.getEveningAvgSBP() != null) {
                sumSurge += today.getMorningAvgSBP() - prevDay.getEveningAvgSBP();
                validPairs++;
            }
        }

        if (validPairs < 3) return null;
        return sumSurge / validPairs;
    }
}
