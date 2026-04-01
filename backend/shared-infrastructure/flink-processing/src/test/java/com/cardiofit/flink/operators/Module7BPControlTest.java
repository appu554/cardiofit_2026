package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module7TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;

class Module7BPControlTest {

    @Test
    void controlledPatient_classifiesControlled() {
        PatientBPState state = Module7TestBuilder.controlledPatient("P-CTRL");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, System.currentTimeMillis());
        BPControlStatus status = Module7BPControlClassifier.classifyControl(summaries);
        assertEquals(BPControlStatus.CONTROLLED, status,
            "7-day avg SBP ~122 should be CONTROLLED");
    }

    @Test
    void stage2Patient_classifiesStage2() {
        PatientBPState state = Module7TestBuilder.stage2Uncontrolled("P-STG2");
        List<DailyBPSummary> summaries = state.getSummariesInWindow(7, System.currentTimeMillis());
        BPControlStatus status = Module7BPControlClassifier.classifyControl(summaries);
        assertEquals(BPControlStatus.STAGE_2_UNCONTROLLED, status,
            "7-day avg SBP ~158 should be STAGE_2");
    }

    @Test
    void controlStatusBoundaries() {
        assertEquals(BPControlStatus.CONTROLLED, BPControlStatus.fromAverages(125, 75));
        assertEquals(BPControlStatus.ELEVATED, BPControlStatus.fromAverages(132, 78));
        assertEquals(BPControlStatus.STAGE_1_UNCONTROLLED, BPControlStatus.fromAverages(138, 82));
        assertEquals(BPControlStatus.STAGE_2_UNCONTROLLED, BPControlStatus.fromAverages(150, 95));
    }

    @Test
    void dbpAlone_canTriggerStage() {
        assertEquals(BPControlStatus.STAGE_1_UNCONTROLLED,
            BPControlStatus.fromAverages(128, 87),
            "DBP 87 alone should trigger STAGE_1 even with controlled SBP");
    }
}
