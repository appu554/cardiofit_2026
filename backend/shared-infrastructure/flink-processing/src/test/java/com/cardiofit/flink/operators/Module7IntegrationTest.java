package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module7TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

/**
 * Integration tests for Module 7.
 * Validates full flow: reading → state → all 5 analyzers → classifications.
 */
class Module7IntegrationTest {

    @Test
    void controlledPatient_producesExpectedClassifications() {
        PatientBPState state = Module7TestBuilder.controlledPatient("P-CTRL");
        long now = System.currentTimeMillis();

        java.util.List<DailyBPSummary> window7 = state.getSummariesInWindow(7, now);
        assertTrue(window7.size() >= 3, "Should have at least 3 days of data");

        Double arv = Module7ARVComputer.computeARV(window7);
        assertNotNull(arv);
        assertEquals(VariabilityClassification.LOW, VariabilityClassification.fromARV(arv));

        BPControlStatus status = Module7BPControlClassifier.classifyControl(window7);
        assertEquals(BPControlStatus.CONTROLLED, status);
    }

    @Test
    void stage2WithHighVariability_producesExpectedOutput() {
        PatientBPState state = Module7TestBuilder.highVariabilityPatient("P-HV");
        long now = System.currentTimeMillis();

        java.util.List<DailyBPSummary> window7 = state.getSummariesInWindow(7, now);
        Double arv = Module7ARVComputer.computeARV(window7);
        assertNotNull(arv);
        assertEquals(VariabilityClassification.HIGH, VariabilityClassification.fromARV(arv),
            "ARV ~18 should be HIGH, got: " + arv);
    }

    @Test
    void crisisReading_detectedBeforeWindowed() {
        BPReading crisis = Module7TestBuilder.crisisReading("P-CRISIS");
        assertTrue(Module7CrisisDetector.isCrisis(crisis));
        assertTrue(crisis.isValid(), "Crisis reading should still be valid");
    }

    @Test
    void reverseDipper_fullPipeline() {
        PatientBPState state = Module7TestBuilder.reverseDipperPatient("P-REVDIP");
        long now = System.currentTimeMillis();

        java.util.List<DailyBPSummary> window7 = state.getSummariesInWindow(7, now);
        Module7DipClassifier.DipResult dip = Module7DipClassifier.classify(window7);
        assertEquals(DipClassification.REVERSE_DIPPER, dip.classification());

        BPControlStatus status = Module7BPControlClassifier.classifyControl(window7);
        assertNotEquals(BPControlStatus.CONTROLLED, status,
            "Reverse dipper with mean SBP ~138 should not be CONTROLLED");
    }

    @Test
    void whiteCoatAndMasked_cannotBothBeTrue() {
        PatientBPState wcState = Module7TestBuilder.whiteCoatSuspect("P-WC");
        java.util.List<DailyBPSummary> wcSummaries = wcState.getSummariesInWindow(30, System.currentTimeMillis());
        Module7BPControlClassifier.WhiteCoatResult wcResult =
            Module7BPControlClassifier.detectWhiteCoatMasked(wcSummaries);
        assertFalse(wcResult.whiteCoatSuspect() && wcResult.maskedHtnSuspect(),
            "Cannot be both white-coat AND masked");

        PatientBPState mhState = Module7TestBuilder.maskedHtnSuspect("P-MH");
        java.util.List<DailyBPSummary> mhSummaries = mhState.getSummariesInWindow(30, System.currentTimeMillis());
        Module7BPControlClassifier.WhiteCoatResult mhResult =
            Module7BPControlClassifier.detectWhiteCoatMasked(mhSummaries);
        assertFalse(mhResult.whiteCoatSuspect() && mhResult.maskedHtnSuspect(),
            "Cannot be both white-coat AND masked");
    }

    @Test
    void insufficientData_gracefulDegradation() {
        PatientBPState state = Module7TestBuilder.insufficientData("P-INSUFF");
        long now = System.currentTimeMillis();

        java.util.List<DailyBPSummary> window7 = state.getSummariesInWindow(7, now);
        Double arv = Module7ARVComputer.computeARV(window7);
        assertNull(arv, "Insufficient data should produce null ARV");
        assertEquals(VariabilityClassification.INSUFFICIENT_DATA,
            VariabilityClassification.fromARV(arv));

        Module7DipClassifier.DipResult dip = Module7DipClassifier.classify(window7);
        assertEquals(DipClassification.INSUFFICIENT_DATA, dip.classification());
    }
}
