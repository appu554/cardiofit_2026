package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module12TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import static com.cardiofit.flink.builders.Module12TestBuilder.*;
import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for Module12ConfounderAccumulator: event-driven confounder
 * flag accumulation during observation windows.
 */
class Module12ConfounderAccumulatorTest {

    @Test
    void externalMedication_flagsConfounder() {
        InterventionWindowState.InterventionWindow window = new InterventionWindowState.InterventionWindow();
        window.status = "OBSERVING";
        window.observationStartMs = BASE_TIME;
        window.observationEndMs = daysAfter(BASE_TIME, 28);

        CanonicalEvent event = externalMedicationEvent("P1", daysAfter(BASE_TIME, 5));

        Module12ConfounderAccumulator.accumulate(window, event);

        assertTrue(window.confoundersDetected.contains("EXTERNAL_MEDICATION_CHANGE"));
    }

    @Test
    void hospitalisation_flagsConfounder() {
        InterventionWindowState.InterventionWindow window = new InterventionWindowState.InterventionWindow();
        window.status = "OBSERVING";
        window.observationStartMs = BASE_TIME;
        window.observationEndMs = daysAfter(BASE_TIME, 28);

        CanonicalEvent event = hospitalisationEvent("P1", daysAfter(BASE_TIME, 10));

        Module12ConfounderAccumulator.accumulate(window, event);

        assertTrue(window.confoundersDetected.contains("HOSPITALISATION"));
    }

    @Test
    void festivalPeriod_flagsConfounder() {
        InterventionWindowState.InterventionWindow window = new InterventionWindowState.InterventionWindow();
        window.status = "OBSERVING";
        window.observationStartMs = BASE_TIME;
        window.observationEndMs = daysAfter(BASE_TIME, 14);

        Module12ConfounderAccumulator.addFestivalConfounder(window, "DIWALI");

        assertTrue(window.confoundersDetected.contains("FESTIVAL_PERIOD:DIWALI"));
    }

    @Test
    void labResult_accumulatedToLabChanges() {
        InterventionWindowState.InterventionWindow window = new InterventionWindowState.InterventionWindow();
        window.status = "OBSERVING";
        window.observationStartMs = BASE_TIME;
        window.observationEndMs = daysAfter(BASE_TIME, 28);

        CanonicalEvent event = fbgReading("P1", daysAfter(BASE_TIME, 7), 145.0);

        Module12ConfounderAccumulator.accumulate(window, event);

        assertEquals(1, window.labChanges.size());
        // Lab results are accumulated, not flagged as confounders
        assertFalse(window.confoundersDetected.contains("LAB_RESULT"));
    }

    @Test
    void multipleConfounders_allAccumulated() {
        InterventionWindowState.InterventionWindow window = new InterventionWindowState.InterventionWindow();
        window.status = "OBSERVING";
        window.observationStartMs = BASE_TIME;
        window.observationEndMs = daysAfter(BASE_TIME, 28);

        Module12ConfounderAccumulator.accumulate(window,
                externalMedicationEvent("P1", daysAfter(BASE_TIME, 3)));
        Module12ConfounderAccumulator.accumulate(window,
                hospitalisationEvent("P1", daysAfter(BASE_TIME, 10)));
        Module12ConfounderAccumulator.accumulate(window,
                patientReportedIllness("P1", daysAfter(BASE_TIME, 15)));

        assertEquals(3, window.confoundersDetected.size());
        assertTrue(window.confoundersDetected.contains("EXTERNAL_MEDICATION_CHANGE"));
        assertTrue(window.confoundersDetected.contains("HOSPITALISATION"));
        assertTrue(window.confoundersDetected.contains("INTERCURRENT_ILLNESS"));
    }
}
