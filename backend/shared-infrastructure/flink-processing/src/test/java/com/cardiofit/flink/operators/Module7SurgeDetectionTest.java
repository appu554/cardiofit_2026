package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module7TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;

class Module7SurgeDetectionTest {

    @Test
    void morningSurgePatient_classifiesHigh() {
        PatientBPState state = Module7TestBuilder.morningSurgePatient("P-SURGE");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, System.currentTimeMillis());
        Double surge7dAvg = Module7SurgeDetector.compute7DayAvgSurge(summaries);
        assertNotNull(surge7dAvg);
        assertTrue(surge7dAvg >= 35.0 - 1e-9,
            "Morning surge ~37 should be >= 35, got: " + surge7dAvg);
        assertEquals(SurgeClassification.HIGH,
            SurgeClassification.fromSurge(surge7dAvg));
    }

    @Test
    void controlledPatient_surgeNormal() {
        PatientBPState state = Module7TestBuilder.controlledPatient("P-CTRL");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, System.currentTimeMillis());
        Double surge7dAvg = Module7SurgeDetector.compute7DayAvgSurge(summaries);
        if (surge7dAvg != null) {
            assertTrue(surge7dAvg < 20.0,
                "Controlled patient surge should be NORMAL (< 20), got: " + surge7dAvg);
            assertEquals(SurgeClassification.NORMAL,
                SurgeClassification.fromSurge(surge7dAvg));
        }
    }

    @Test
    void todaySurge_computedFromMorningMinusPreviousEvening() {
        PatientBPState state = new PatientBPState("P-TODAY");
        // Use fixed noon-UTC timestamps to avoid date-boundary flakiness.
        // yesterday+12h can land on today's calendar day when test runs after noon UTC.
        java.time.LocalDate todayDate = java.time.Instant.ofEpochMilli(System.currentTimeMillis())
            .atZone(java.time.ZoneOffset.UTC).toLocalDate();
        long now = todayDate.atTime(10, 0).toInstant(java.time.ZoneOffset.UTC).toEpochMilli();
        long yesterdayEvening = todayDate.minusDays(1).atTime(20, 0)
            .toInstant(java.time.ZoneOffset.UTC).toEpochMilli();
        state.addReading(Module7TestBuilder.eveningReading("P-TODAY", 115, 72, yesterdayEvening));
        state.addReading(Module7TestBuilder.morningReading("P-TODAY", 148, 88, now));

        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, now);
        Double todaySurge = Module7SurgeDetector.computeTodaySurge(summaries, now);
        assertNotNull(todaySurge);
        assertEquals(33.0, todaySurge, 1.0, "Surge should be ~148 - 115 = 33");
    }

    @Test
    void noEveningReading_surgeIsNull() {
        PatientBPState state = new PatientBPState("P-NO-EVE");
        long now = System.currentTimeMillis();
        state.addReading(Module7TestBuilder.morningReading("P-NO-EVE", 140, 85, now));
        state.addReading(Module7TestBuilder.morningReading("P-NO-EVE", 138, 84, now - 86400000L));

        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, now);
        Double todaySurge = Module7SurgeDetector.computeTodaySurge(summaries, now);
        assertNull(todaySurge, "No evening reading means surge cannot be computed");
    }

    @Test
    void surgeClassification_boundaries() {
        assertEquals(SurgeClassification.NORMAL, SurgeClassification.fromSurge(15.0));
        assertEquals(SurgeClassification.ELEVATED, SurgeClassification.fromSurge(25.0));
        assertEquals(SurgeClassification.HIGH, SurgeClassification.fromSurge(40.0));
        assertEquals(SurgeClassification.INSUFFICIENT_DATA, SurgeClassification.fromSurge(null));
    }

    // ---- Surge lookback window tests (Fix 2: 72-hour configurable lookback) ----

    @Test
    void threeDayGap_surgeIsNull_lookbackOneDayOnly() {
        // Lookback is now 1 day only — a 3-day gap produces no valid surge pair.
        // Clinically, a 48h+ gap makes the surge meaningless per Kario (2010).
        PatientBPState state = new PatientBPState("P-GAP-3D");
        java.time.LocalDate today = java.time.Instant.ofEpochMilli(System.currentTimeMillis())
            .atZone(java.time.ZoneOffset.UTC).toLocalDate();

        // Evening reading 3 days ago (outside 1-day lookback)
        long threeDaysAgoEvening = today.minusDays(3).atTime(20, 0)
            .toInstant(java.time.ZoneOffset.UTC).toEpochMilli();
        state.addReading(Module7TestBuilder.eveningReading("P-GAP-3D", 138, 82, threeDaysAgoEvening));

        // Morning reading today
        long todayMorning = today.atTime(8, 0)
            .toInstant(java.time.ZoneOffset.UTC).toEpochMilli();
        state.addReading(Module7TestBuilder.morningReading("P-GAP-3D", 172, 95, todayMorning));

        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, todayMorning);
        Double surge = Module7SurgeDetector.computeTodaySurge(summaries, todayMorning);

        assertNull(surge, "3-day gap should NOT compute surge — lookback is 1 day only");
    }

    @Test
    void fourDayGap_surgeIsNull() {
        // Beyond the 3-day lookback window
        PatientBPState state = new PatientBPState("P-GAP-4D");
        java.time.LocalDate today = java.time.Instant.ofEpochMilli(System.currentTimeMillis())
            .atZone(java.time.ZoneOffset.UTC).toLocalDate();

        long fourDaysAgoEvening = today.minusDays(4).atTime(20, 0)
            .toInstant(java.time.ZoneOffset.UTC).toEpochMilli();
        state.addReading(Module7TestBuilder.eveningReading("P-GAP-4D", 120, 75, fourDaysAgoEvening));

        long todayMorning = today.atTime(8, 0)
            .toInstant(java.time.ZoneOffset.UTC).toEpochMilli();
        state.addReading(Module7TestBuilder.morningReading("P-GAP-4D", 150, 90, todayMorning));

        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, todayMorning);
        Double surge = Module7SurgeDetector.computeTodaySurge(summaries, todayMorning);

        assertNull(surge, "4-day gap exceeds 1-day lookback — surge should be null");
    }

    @Test
    void twoDayGap_pairsWithMostRecent() {
        // Day-2 evening and day-1 evening both exist; should pair with day-1 (most recent)
        PatientBPState state = new PatientBPState("P-GAP-PREFER");
        java.time.LocalDate today = java.time.Instant.ofEpochMilli(System.currentTimeMillis())
            .atZone(java.time.ZoneOffset.UTC).toLocalDate();

        long twoDaysAgoEvening = today.minusDays(2).atTime(20, 0)
            .toInstant(java.time.ZoneOffset.UTC).toEpochMilli();
        state.addReading(Module7TestBuilder.eveningReading("P-GAP-PREFER", 110, 70, twoDaysAgoEvening));

        long oneDayAgoEvening = today.minusDays(1).atTime(20, 0)
            .toInstant(java.time.ZoneOffset.UTC).toEpochMilli();
        state.addReading(Module7TestBuilder.eveningReading("P-GAP-PREFER", 125, 78, oneDayAgoEvening));

        long todayMorning = today.atTime(8, 0)
            .toInstant(java.time.ZoneOffset.UTC).toEpochMilli();
        state.addReading(Module7TestBuilder.morningReading("P-GAP-PREFER", 155, 92, todayMorning));

        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, todayMorning);
        Double surge = Module7SurgeDetector.computeTodaySurge(summaries, todayMorning);

        assertNotNull(surge);
        // Should pair with day-1 (125), not day-2 (110): 155 - 125 = 30
        assertEquals(30.0, surge, 1.0, "Should pair with most recent evening (day-1=125), not older (day-2=110)");
    }
}
