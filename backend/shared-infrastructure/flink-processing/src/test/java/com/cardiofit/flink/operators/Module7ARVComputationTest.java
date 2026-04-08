package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module7TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;

class Module7ARVComputationTest {

    // ── Day-to-day ARV (legacy, secondary metric) ──

    @Test
    void controlledPatient_dayToDayArvBelow8() {
        PatientBPState state = Module7TestBuilder.controlledPatient("P-CTRL");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, System.currentTimeMillis());
        Double arv = Module7ARVComputer.computeDayToDayARV(summaries);
        assertNotNull(arv, "7 days of data should produce non-null day-to-day ARV");
        assertTrue(arv < 8.0, "Controlled patient day-to-day ARV should be LOW (< 8), got: " + arv);
    }

    @Test
    void highVariabilityPatient_dayToDayArvAbove16() {
        PatientBPState state = Module7TestBuilder.highVariabilityPatient("P-HIGH-VAR");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, System.currentTimeMillis());
        Double arv = Module7ARVComputer.computeDayToDayARV(summaries);
        assertNotNull(arv);
        assertTrue(arv >= 16.0, "High variability patient day-to-day ARV should be HIGH (>= 16), got: " + arv);
    }

    // ── Reading-level ARV (Mena et al. 2005, ESH 2023 — PRIMARY metric) ──

    @Test
    void readingLevelARV_capturesMorningEveningOscillation() {
        // Rajesh scenario: morning/evening oscillation of 4-20 mmHg
        // Daily-average ARV would report ~0.5, reading-level ARV should be ~11.6
        List<Double> readings = List.of(172.0, 168.0, 175.0, 164.0, 180.0, 160.0);
        Double arv = Module7ARVComputer.computeReadingLevelARV(readings);
        assertNotNull(arv);
        // |168-172|+|175-168|+|164-175|+|180-164|+|160-180| = 4+7+11+16+20 = 58, /5 = 11.6
        assertEquals(11.6, arv, 0.01, "Reading-level ARV should capture morning-evening oscillation");
    }

    @Test
    void readingLevelARV_controlledPatient_stillReasonable() {
        // Controlled patient with morning+evening readings — small oscillations
        PatientBPState state = Module7TestBuilder.controlledPatient("P-CTRL-RL");
        long now = System.currentTimeMillis();
        List<Double> readings = state.getReadingSBPsInWindow(7, now);
        Double arv = Module7ARVComputer.computeReadingLevelARV(readings);
        assertNotNull(arv);
        // Controlled patient: small morning-evening dip (~8 mmHg), so ARV ~8
        assertTrue(arv < 12.0, "Controlled patient reading-level ARV should be moderate, got: " + arv);
    }

    @Test
    void readingLevelARV_insufficientReadings_returnsNull() {
        List<Double> twoReadings = List.of(120.0, 130.0);
        assertNull(Module7ARVComputer.computeReadingLevelARV(twoReadings));
        assertNull(Module7ARVComputer.computeReadingLevelARV(null));
    }

    @Test
    void insufficientData_returnsNull() {
        PatientBPState state = Module7TestBuilder.insufficientData("P-INSUFF");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, System.currentTimeMillis());
        Double arv = Module7ARVComputer.computeDayToDayARV(summaries);
        assertNull(arv, "Fewer than 3 days should produce null day-to-day ARV");
    }

    @Test
    void exactlyThreeDays_computesARV() {
        PatientBPState state = new PatientBPState("P-3DAY");
        long now = System.currentTimeMillis();
        state.addReading(Module7TestBuilder.morningReading("P-3DAY", 120, 78, now - 2 * 86400000L));
        state.addReading(Module7TestBuilder.morningReading("P-3DAY", 130, 84, now - 1 * 86400000L));
        state.addReading(Module7TestBuilder.morningReading("P-3DAY", 125, 80, now));
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, now);
        Double arv = Module7ARVComputer.computeDayToDayARV(summaries);
        assertNotNull(arv);
        assertEquals(7.5, arv, 0.01, "Day-to-day ARV for [120, 130, 125] should be 7.5");
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
        Double arv30 = Module7ARVComputer.computeDayToDayARV(summaries30);
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
