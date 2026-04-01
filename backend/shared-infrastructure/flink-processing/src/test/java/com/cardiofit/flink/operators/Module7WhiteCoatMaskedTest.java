package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module7TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;

class Module7WhiteCoatMaskedTest {

    @Test
    void whiteCoatSuspect_detected() {
        PatientBPState state = Module7TestBuilder.whiteCoatSuspect("P-WC");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(30, System.currentTimeMillis());
        Module7BPControlClassifier.WhiteCoatResult result =
            Module7BPControlClassifier.detectWhiteCoatMasked(summaries);
        assertTrue(result.whiteCoatSuspect(),
            "Clinic ~150 vs home ~123 = delta ~27 should trigger white-coat suspect");
        assertFalse(result.maskedHtnSuspect());
    }

    @Test
    void maskedHtnSuspect_detected() {
        PatientBPState state = Module7TestBuilder.maskedHtnSuspect("P-MH");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(30, System.currentTimeMillis());
        Module7BPControlClassifier.WhiteCoatResult result =
            Module7BPControlClassifier.detectWhiteCoatMasked(summaries);
        assertTrue(result.maskedHtnSuspect(),
            "Home ~146 vs clinic ~127 should trigger masked HTN suspect");
        assertFalse(result.whiteCoatSuspect());
    }

    @Test
    void neitherCondition_whenNoDelta() {
        PatientBPState state = Module7TestBuilder.controlledPatient("P-CTRL");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(30, System.currentTimeMillis());
        Module7BPControlClassifier.WhiteCoatResult result =
            Module7BPControlClassifier.detectWhiteCoatMasked(summaries);
        assertFalse(result.whiteCoatSuspect());
        assertFalse(result.maskedHtnSuspect());
    }
}
