package com.cardiofit.flink.builders;

import com.cardiofit.flink.models.*;
import java.time.*;
import java.util.*;

/**
 * Test data factory for Module 7 BP Variability Engine tests.
 * Provides patient scenarios with realistic BP reading patterns.
 */
public class Module7TestBuilder {

    // ── Single BPReading builders ──

    public static BPReading reading(String patientId, double sbp, double dbp,
                                     long timestamp, String timeContext, String source) {
        BPReading r = new BPReading();
        r.setPatientId(patientId);
        r.setSystolic(sbp);
        r.setDiastolic(dbp);
        r.setTimestamp(timestamp);
        r.setTimeContext(timeContext);
        r.setSource(source);
        return r;
    }

    public static BPReading morningReading(String patientId, double sbp, double dbp, long timestamp) {
        return reading(patientId, sbp, dbp, timestamp, "MORNING", "HOME_CUFF");
    }

    public static BPReading eveningReading(String patientId, double sbp, double dbp, long timestamp) {
        return reading(patientId, sbp, dbp, timestamp, "EVENING", "HOME_CUFF");
    }

    public static BPReading clinicReading(String patientId, double sbp, double dbp, long timestamp) {
        return reading(patientId, sbp, dbp, timestamp, "MORNING", "CLINIC");
    }

    public static BPReading crisisReading(String patientId) {
        return reading(patientId, 195.0, 125.0,
            System.currentTimeMillis(), "MORNING", "HOME_CUFF");
    }

    // ── PatientBPState builders (pre-populated with multiple days) ──

    /**
     * Well-controlled patient: 7 days of stable readings.
     * Mean SBP ~122, ARV ~3, should classify as CONTROLLED + LOW variability.
     */
    public static PatientBPState controlledPatient(String patientId) {
        PatientBPState state = new PatientBPState(patientId);
        long now = System.currentTimeMillis();
        double[] sbps = {120, 124, 121, 123, 119, 125, 122};
        double[] dbps = {76, 78, 75, 77, 74, 79, 76};
        for (int day = 6; day >= 0; day--) {
            long ts = now - day * 24 * 60 * 60 * 1000L;
            // Morning reading
            state.addReading(morningReading(patientId, sbps[6-day], dbps[6-day], ts));
            // Evening reading (slightly lower — normal dipping)
            state.addReading(eveningReading(patientId, sbps[6-day] - 8, dbps[6-day] - 5,
                ts + 12 * 60 * 60 * 1000L));
        }
        return state;
    }

    /**
     * High variability patient: 7 days of unstable readings.
     * ARV ~18, should classify as HIGH variability.
     */
    public static PatientBPState highVariabilityPatient(String patientId) {
        PatientBPState state = new PatientBPState(patientId);
        long now = System.currentTimeMillis();
        // Oscillating SBP: big swings day to day
        double[] sbps = {125, 155, 118, 160, 122, 152, 128};
        double[] dbps = {78, 95, 74, 98, 76, 92, 80};
        for (int day = 6; day >= 0; day--) {
            long ts = now - day * 24 * 60 * 60 * 1000L;
            state.addReading(morningReading(patientId, sbps[6-day], dbps[6-day], ts));
        }
        return state;
    }

    /**
     * Non-dipper patient: nocturnal SBP does not drop (< 10% reduction).
     * 7 days with daytime ~141, nocturnal ~138 (dip ratio ~3% — NON_DIPPER).
     */
    public static PatientBPState nonDipperPatient(String patientId) {
        PatientBPState state = new PatientBPState(patientId);
        long now = System.currentTimeMillis();
        for (int day = 6; day >= 0; day--) {
            long ts = now - day * 24 * 60 * 60 * 1000L;
            // Daytime: ~142
            state.addReading(morningReading(patientId, 142, 88, ts));
            state.addReading(reading(patientId, 140, 86,
                ts + 6 * 60 * 60 * 1000L, "AFTERNOON", "HOME_CUFF"));
            // Nocturnal: ~138 (only ~3% drop — NON_DIPPER)
            state.addReading(reading(patientId, 138, 85,
                ts + 18 * 60 * 60 * 1000L, "NIGHT", "HOME_CUFF"));
        }
        return state;
    }

    /**
     * Reverse dipper: nocturnal SBP > daytime SBP.
     * Highest CV risk pattern. Daytime ~134, nocturnal ~145.
     */
    public static PatientBPState reverseDipperPatient(String patientId) {
        PatientBPState state = new PatientBPState(patientId);
        long now = System.currentTimeMillis();
        for (int day = 6; day >= 0; day--) {
            long ts = now - day * 24 * 60 * 60 * 1000L;
            // Daytime: ~135
            state.addReading(morningReading(patientId, 135, 82, ts));
            state.addReading(reading(patientId, 133, 80,
                ts + 6 * 60 * 60 * 1000L, "AFTERNOON", "HOME_CUFF"));
            // Nocturnal: ~145 (reverse dip — nocturnal > daytime)
            state.addReading(reading(patientId, 145, 90,
                ts + 18 * 60 * 60 * 1000L, "NIGHT", "HOME_CUFF"));
        }
        return state;
    }

    /**
     * Morning surge patient: large morning-evening differential.
     * Morning SBP ~155, previous evening ~118 → surge ~37 (HIGH).
     */
    public static PatientBPState morningSurgePatient(String patientId) {
        PatientBPState state = new PatientBPState(patientId);
        long now = System.currentTimeMillis();
        for (int day = 6; day >= 0; day--) {
            long ts = now - day * 24 * 60 * 60 * 1000L;
            // Evening: low
            state.addReading(eveningReading(patientId, 118, 72,
                ts - 6 * 60 * 60 * 1000L)); // previous evening
            // Morning: high surge
            state.addReading(morningReading(patientId, 155, 92, ts));
        }
        return state;
    }

    /**
     * White-coat hypertension suspect: clinic BP > home BP by > 15 mmHg.
     * Home ~123, clinic ~150 → delta ~27.
     */
    public static PatientBPState whiteCoatSuspect(String patientId) {
        PatientBPState state = new PatientBPState(patientId);
        long now = System.currentTimeMillis();
        for (int day = 6; day >= 0; day--) {
            long ts = now - day * 24 * 60 * 60 * 1000L;
            // Home readings: normal
            state.addReading(morningReading(patientId, 125, 78, ts));
            state.addReading(eveningReading(patientId, 122, 76, ts + 12 * 60 * 60 * 1000L));
        }
        // Add clinic readings: elevated (> 15 mmHg above home)
        long clinicTs = now - 2 * 24 * 60 * 60 * 1000L;
        state.addReading(clinicReading(patientId, 148, 92, clinicTs));
        state.addReading(clinicReading(patientId, 152, 94, clinicTs + 300_000));
        return state;
    }

    /**
     * Masked hypertension suspect: home BP > clinic BP by > 15 mmHg.
     * Home ~146, clinic ~127 → delta ~-19.
     */
    public static PatientBPState maskedHtnSuspect(String patientId) {
        PatientBPState state = new PatientBPState(patientId);
        long now = System.currentTimeMillis();
        for (int day = 6; day >= 0; day--) {
            long ts = now - day * 24 * 60 * 60 * 1000L;
            // Home readings: elevated
            state.addReading(morningReading(patientId, 148, 92, ts));
            state.addReading(eveningReading(patientId, 145, 90, ts + 12 * 60 * 60 * 1000L));
        }
        // Clinic readings: normal (patient relaxed in clinic)
        long clinicTs = now - 2 * 24 * 60 * 60 * 1000L;
        state.addReading(clinicReading(patientId, 128, 78, clinicTs));
        state.addReading(clinicReading(patientId, 126, 76, clinicTs + 300_000));
        return state;
    }

    /**
     * Stage 2 uncontrolled: 7-day average SBP ~158.
     */
    public static PatientBPState stage2Uncontrolled(String patientId) {
        PatientBPState state = new PatientBPState(patientId);
        long now = System.currentTimeMillis();
        double[] sbps = {155, 162, 158, 160, 154, 163, 156};
        double[] dbps = {95, 98, 96, 97, 94, 99, 95};
        for (int day = 6; day >= 0; day--) {
            long ts = now - day * 24 * 60 * 60 * 1000L;
            state.addReading(morningReading(patientId, sbps[6-day], dbps[6-day], ts));
        }
        return state;
    }

    /**
     * Insufficient data: only 2 days of readings.
     * ARV and dipping should return INSUFFICIENT_DATA.
     */
    public static PatientBPState insufficientData(String patientId) {
        PatientBPState state = new PatientBPState(patientId);
        long now = System.currentTimeMillis();
        state.addReading(morningReading(patientId, 130, 82, now));
        state.addReading(morningReading(patientId, 128, 80, now - 24 * 60 * 60 * 1000L));
        return state;
    }
}
