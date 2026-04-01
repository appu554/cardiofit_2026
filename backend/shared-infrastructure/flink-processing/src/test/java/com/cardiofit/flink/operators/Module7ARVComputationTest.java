package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module7TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;

class Module7ARVComputationTest {

    @Test
    void controlledPatient_arvBelow8() {
        PatientBPState state = Module7TestBuilder.controlledPatient("P-CTRL");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, System.currentTimeMillis());
        Double arv = Module7ARVComputer.computeARV(summaries);
        assertNotNull(arv, "7 days of data should produce non-null ARV");
        assertTrue(arv < 8.0, "Controlled patient ARV should be LOW (< 8), got: " + arv);
    }

    @Test
    void highVariabilityPatient_arvAbove16() {
        PatientBPState state = Module7TestBuilder.highVariabilityPatient("P-HIGH-VAR");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, System.currentTimeMillis());
        Double arv = Module7ARVComputer.computeARV(summaries);
        assertNotNull(arv);
        assertTrue(arv >= 16.0, "High variability patient ARV should be HIGH (>= 16), got: " + arv);
    }

    @Test
    void insufficientData_returnsNull() {
        PatientBPState state = Module7TestBuilder.insufficientData("P-INSUFF");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, System.currentTimeMillis());
        Double arv = Module7ARVComputer.computeARV(summaries);
        assertNull(arv, "Fewer than 3 days should produce null ARV");
    }

    @Test
    void exactlyThreeDays_computesARV() {
        PatientBPState state = new PatientBPState("P-3DAY");
        long now = System.currentTimeMillis();
        state.addReading(Module7TestBuilder.morningReading("P-3DAY", 120, 78, now - 2 * 86400000L));
        state.addReading(Module7TestBuilder.morningReading("P-3DAY", 130, 84, now - 1 * 86400000L));
        state.addReading(Module7TestBuilder.morningReading("P-3DAY", 125, 80, now));
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, now);
        Double arv = Module7ARVComputer.computeARV(summaries);
        assertNotNull(arv);
        assertEquals(7.5, arv, 0.01, "ARV for [120, 130, 125] should be 7.5");
    }

    @Test
    void sdAndCV_computeCorrectly() {
        PatientBPState state = new PatientBPState("P-SD");
        long now = System.currentTimeMillis();
        double[] sbps = {120, 130, 125, 135, 120};
        for (int i = 0; i < 5; i++) {
            state.addReading(Module7TestBuilder.morningReading("P-SD", sbps[i], 80, now - (4-i) * 86400000L));
        }
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, now);
        double mean = Module7ARVComputer.computeMeanSBP(summaries);
        double sd = Module7ARVComputer.computeSD(summaries);
        double cv = Module7ARVComputer.computeCV(summaries);
        assertEquals(126.0, mean, 0.1);
        assertTrue(sd > 0);
        assertEquals(sd / mean, cv, 0.001);
    }

    @Test
    void thirtyDayWindow_usesAllDays() {
        PatientBPState state = new PatientBPState("P-30D");
        long now = System.currentTimeMillis();
        for (int day = 19; day >= 0; day--) {
            double sbp = 130 + (day % 2 == 0 ? 5 : -5);
            state.addReading(Module7TestBuilder.morningReading("P-30D", sbp, 80, now - day * 86400000L));
        }
        List<DailyBPSummary> summaries7 = state.getSummariesInWindow(7, now);
        List<DailyBPSummary> summaries30 = state.getSummariesInWindow(30, now);
        assertTrue(summaries30.size() > summaries7.size());
        Double arv30 = Module7ARVComputer.computeARV(summaries30);
        assertNotNull(arv30);
        assertEquals(10.0, arv30, 0.1);
    }

    @Test
    void variabilityClassification_matchesARV() {
        assertEquals(VariabilityClassification.LOW, VariabilityClassification.fromARV(5.0));
        assertEquals(VariabilityClassification.MODERATE, VariabilityClassification.fromARV(10.0));
        assertEquals(VariabilityClassification.ELEVATED, VariabilityClassification.fromARV(14.0));
        assertEquals(VariabilityClassification.HIGH, VariabilityClassification.fromARV(18.0));
        assertEquals(VariabilityClassification.INSUFFICIENT_DATA, VariabilityClassification.fromARV(null));
    }
}
