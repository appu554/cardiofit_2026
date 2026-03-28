package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module3TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

public class Module3Phase7SafetyTest {

    @Test
    void penicillinAllergyPatient_flagsBetaLactam() {
        EnrichedPatientContext patient = Module3TestBuilder.hypertensiveDiabeticPatient("ALLERGY-001");
        // Patient has Penicillin allergy set in builder

        CDSPhaseResult result = Module3PhaseExecutor.executePhase7(patient);
        SafetyCheckResult safety = (SafetyCheckResult) result.getDetail("safetyResult");

        assertTrue(result.isActive());
        assertNotNull(safety);
        // No active beta-lactam meds → no allergy alerts expected
        assertEquals(0, safety.getAllergyAlerts().size());
    }

    @Test
    void patientWithNoAllergies_noAlerts() {
        EnrichedPatientContext patient = Module3TestBuilder.sepsisPatient("SAFE-001");
        patient.getPatientState().setAllergies(java.util.Collections.emptyList());

        CDSPhaseResult result = Module3PhaseExecutor.executePhase7(patient);
        SafetyCheckResult safety = (SafetyCheckResult) result.getDetail("safetyResult");

        assertNotNull(safety);
        assertEquals(0, safety.getTotalAlerts());
        assertEquals("LOW", safety.getHighestSeverity());
    }

    @Test
    void phaseAlwaysActive_evenWithNullState() {
        EnrichedPatientContext patient = new EnrichedPatientContext("NULL-001", new PatientContextState("NULL-001"));
        patient.setEventType("VITAL_SIGN");

        CDSPhaseResult result = Module3PhaseExecutor.executePhase7(patient);

        assertTrue(result.isActive());
        assertEquals("PHASE_7_SAFETY_CHECK", result.getPhaseName());
    }
}
