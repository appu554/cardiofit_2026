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
        long now = System.currentTimeMillis();
        long yesterday = now - 24 * 60 * 60 * 1000L;
        state.addReading(Module7TestBuilder.eveningReading("P-TODAY", 115, 72, yesterday + 12 * 3600000L));
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
}
