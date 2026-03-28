package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module3TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

public class Module3Phase4DiagnosticTest {

    @Test
    void phase4_returnsActiveForPatientWithLabs() {
        EnrichedPatientContext patient = Module3TestBuilder.hypertensiveDiabeticPatient("DIAG-001");

        CDSPhaseResult result = Module3PhaseExecutor.executePhase4(patient);

        assertEquals("PHASE_4_DIAGNOSTIC_ASSESSMENT", result.getPhaseName());
        assertTrue(result.isActive());
        assertEquals(2, result.getDetail("labCount")); // HbA1c + Creatinine
    }

    @Test
    void phase4_inactiveForPatientWithoutLabs() {
        PatientContextState state = new PatientContextState("NO-LABS");
        state.getLatestVitals().put("heartrate", 80);
        EnrichedPatientContext patient = new EnrichedPatientContext("NO-LABS", state);
        patient.setEventType("VITAL_SIGN");
        patient.setEventTime(System.currentTimeMillis());

        CDSPhaseResult result = Module3PhaseExecutor.executePhase4(patient);

        assertFalse(result.isActive());
        assertEquals(0, result.getDetail("labCount"));
    }

    @Test
    void phase4_identifiesAbnormalLabs() {
        EnrichedPatientContext patient = Module3TestBuilder.hypertensiveDiabeticPatient("DIAG-002");

        CDSPhaseResult result = Module3PhaseExecutor.executePhase4(patient);

        // HbA1c=8.2 is abnormal (>6.5%), Creatinine=1.4 is elevated (>1.2 for male)
        Object abnormalCount = result.getDetail("abnormalLabCount");
        assertNotNull(abnormalCount);
        assertTrue(((Number) abnormalCount).intValue() >= 1);
    }
}
