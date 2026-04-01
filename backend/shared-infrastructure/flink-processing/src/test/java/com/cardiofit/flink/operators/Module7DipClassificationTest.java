package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module7TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;

class Module7DipClassificationTest {

    @Test
    void controlledPatient_withoutNightReadings_returnsInsufficientData() {
        // controlledPatient() has MORNING + EVENING but no NIGHT readings.
        // EVENING (17-22h) is isDaytime()=true, isNocturnal()=false in TimeContext,
        // so the dip classifier cannot compute a nocturnal mean → INSUFFICIENT_DATA.
        PatientBPState state = Module7TestBuilder.controlledPatient("P-DIP");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, System.currentTimeMillis());
        Module7DipClassifier.DipResult result = Module7DipClassifier.classify(summaries);
        assertNotNull(result);
        assertEquals(DipClassification.INSUFFICIENT_DATA, result.classification(),
            "No NIGHT readings → classifier should return INSUFFICIENT_DATA");
    }

    @Test
    void nonDipperPatient_classifiesAsNonDipper() {
        PatientBPState state = Module7TestBuilder.nonDipperPatient("P-NONDIP");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, System.currentTimeMillis());
        Module7DipClassifier.DipResult result = Module7DipClassifier.classify(summaries);
        assertNotNull(result);
        assertNotNull(result.dipRatio());
        assertEquals(DipClassification.NON_DIPPER, result.classification(),
            "Dip ratio ~3% should be NON_DIPPER, got ratio: " + result.dipRatio());
    }

    @Test
    void reverseDipperPatient_classifiesAsReverseDipper() {
        PatientBPState state = Module7TestBuilder.reverseDipperPatient("P-REVDIP");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, System.currentTimeMillis());
        Module7DipClassifier.DipResult result = Module7DipClassifier.classify(summaries);
        assertNotNull(result);
        assertNotNull(result.dipRatio());
        assertTrue(result.dipRatio() < 0, "Reverse dipper should have negative dip ratio");
        assertEquals(DipClassification.REVERSE_DIPPER, result.classification());
    }

    @Test
    void insufficientData_returnsInsufficientClassification() {
        PatientBPState state = Module7TestBuilder.insufficientData("P-INSUFF");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, System.currentTimeMillis());
        Module7DipClassifier.DipResult result = Module7DipClassifier.classify(summaries);
        assertEquals(DipClassification.INSUFFICIENT_DATA, result.classification());
    }

    @Test
    void dipRatio_boundaries() {
        assertEquals(DipClassification.REVERSE_DIPPER, DipClassification.fromDipRatio(-0.05));
        assertEquals(DipClassification.NON_DIPPER, DipClassification.fromDipRatio(0.05));
        assertEquals(DipClassification.DIPPER, DipClassification.fromDipRatio(0.15));
        assertEquals(DipClassification.EXTREME_DIPPER, DipClassification.fromDipRatio(0.25));
    }
}
