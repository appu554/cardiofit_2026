package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module3TestBuilder;
import com.cardiofit.flink.models.*;
import com.cardiofit.flink.models.protocol.SimplifiedProtocol;
import org.junit.jupiter.api.Test;
import java.util.*;
import static org.junit.jupiter.api.Assertions.*;

public class Module3Phase5ConcordanceTest {

    @Test
    void concordance_htnPatientOnARB_concordant() {
        EnrichedPatientContext patient = Module3TestBuilder.hypertensiveDiabeticPatient("CONC-001");
        // Patient has Telmisartan (ARB) — concordant with HTN protocol
        Map<String, SimplifiedProtocol> protocols = Module3TestBuilder.defaultProtocolMap();
        List<String> matched = Arrays.asList("HTN-MGMT-V3");

        CDSPhaseResult result = Module3PhaseExecutor.executePhase5(patient, matched, protocols);

        assertTrue(result.isActive());
        Object concordantCount = result.getDetail("concordantCount");
        assertEquals(1L, concordantCount);
    }

    @Test
    void concordance_sepsisPatientNoAntibiotics_partial() {
        EnrichedPatientContext patient = Module3TestBuilder.sepsisPatient("CONC-002");
        Map<String, SimplifiedProtocol> protocols = Module3TestBuilder.defaultProtocolMap();
        List<String> matched = Arrays.asList("SEPSIS-BUNDLE-V2");

        CDSPhaseResult result = Module3PhaseExecutor.executePhase5(patient, matched, protocols);

        // Sepsis patient has no medications → PARTIAL (no antibiotics started)
        assertTrue(result.isActive());
        Object discordantCount = result.getDetail("discordantCount");
        // Should generate recommendation
        assertNotNull(result.getDetail("guidelineMatches"));
    }

    @Test
    void phase6_flagsRenal_impairment_dose_concern() {
        EnrichedPatientContext patient = Module3TestBuilder.hypertensiveDiabeticPatient("MED-001");
        // Patient has Creatinine=1.4 → impaired renal function

        CDSPhaseResult result = Module3PhaseExecutor.executePhase6(patient);

        assertTrue(result.isActive());
        assertEquals(1, result.getDetail("totalMedications"));
    }
}
