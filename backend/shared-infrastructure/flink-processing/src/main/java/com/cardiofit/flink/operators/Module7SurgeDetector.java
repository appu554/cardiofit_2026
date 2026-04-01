package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.DailyBPSummary;
import java.time.Instant;
import java.time.ZoneOffset;
import java.util.List;

/**
 * Morning BP surge detection using the sleep-trough method.
 * Surge = morning_SBP_today - evening_SBP_yesterday
 * No Flink dependencies — fully unit-testable.
 */
public final class Module7SurgeDetector {

    private Module7SurgeDetector() {}

    public static Double computeTodaySurge(List<DailyBPSummary> summaries, long referenceTime) {
        if (summaries == null || summaries.size() < 2) return null;

        String todayKey = Instant.ofEpochMilli(referenceTime)
            .atZone(ZoneOffset.UTC).toLocalDate().toString();
        String yesterdayKey = Instant.ofEpochMilli(referenceTime)
            .atZone(ZoneOffset.UTC).toLocalDate().minusDays(1).toString();

        DailyBPSummary today = null;
        DailyBPSummary yesterday = null;

        for (DailyBPSummary s : summaries) {
            if (todayKey.equals(s.getDateKey())) today = s;
            if (yesterdayKey.equals(s.getDateKey())) yesterday = s;
        }

        if (today == null || today.getMorningAvgSBP() == null) return null;
        if (yesterday == null || yesterday.getEveningAvgSBP() == null) return null;

        return today.getMorningAvgSBP() - yesterday.getEveningAvgSBP();
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
