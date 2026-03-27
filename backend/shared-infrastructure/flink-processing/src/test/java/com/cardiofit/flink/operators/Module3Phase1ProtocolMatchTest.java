package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module3TestBuilder;
import com.cardiofit.flink.models.*;
import com.cardiofit.flink.models.protocol.SimplifiedProtocol;
import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

import java.util.List;
import java.util.Map;

public class Module3Phase1ProtocolMatchTest {

    @Test
    void sepsisPatient_matchesSepsisProtocol() {
        EnrichedPatientContext patient = Module3TestBuilder.sepsisPatient("SEP-001");
        Map<String, SimplifiedProtocol> protocols = Module3TestBuilder.defaultProtocolMap();

        CDSPhaseResult result = Module3PhaseExecutor.executePhase1(patient, protocols);

        assertTrue(result.isActive());
        assertEquals("PHASE_1_PROTOCOL_MATCH", result.getPhaseName());
        @SuppressWarnings("unchecked")
        List<String> matched = (List<String>) result.getDetail("matchedProtocolIds");
        assertNotNull(matched);
        assertTrue(matched.contains("SEPSIS-BUNDLE-V2"));
    }

    @Test
    void hypertensivePatient_matchesHTNProtocol() {
        EnrichedPatientContext patient = Module3TestBuilder.hypertensiveDiabeticPatient("HTN-001");
        Map<String, SimplifiedProtocol> protocols = Module3TestBuilder.defaultProtocolMap();

        CDSPhaseResult result = Module3PhaseExecutor.executePhase1(patient, protocols);

        assertTrue(result.isActive());
        @SuppressWarnings("unchecked")
        List<String> matched = (List<String>) result.getDetail("matchedProtocolIds");
        assertTrue(matched.contains("HTN-MGMT-V3"));
    }

    @Test
    void emptyProtocols_returnsInactivePhase() {
        EnrichedPatientContext patient = Module3TestBuilder.sepsisPatient("SEP-002");
        Map<String, SimplifiedProtocol> empty = Map.of();

        CDSPhaseResult result = Module3PhaseExecutor.executePhase1(patient, empty);

        assertFalse(result.isActive());
        assertEquals(0, result.getDetail("matchedCount"));
    }
}
