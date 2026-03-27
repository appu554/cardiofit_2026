package com.cardiofit.flink.models;

import org.junit.jupiter.api.Test;
import static org.junit.jupiter.api.Assertions.*;

import java.util.List;

public class CDSEventTest {

    @Test
    void constructFromEnrichedPatientContext() {
        PatientContextState state = new PatientContextState("P-001");
        state.setNews2Score(4);
        state.setQsofaScore(1);
        state.setCombinedAcuityScore(3.5);

        EnrichedPatientContext epc = new EnrichedPatientContext("P-001", state);
        epc.setEventType("VITAL_SIGN");
        epc.setEventTime(1000L);
        epc.setDataTier("TIER_1_CGM");

        CDSEvent cds = new CDSEvent(epc);

        assertEquals("P-001", cds.getPatientId());
        assertEquals("VITAL_SIGN", cds.getEventType());
        assertEquals("TIER_1_CGM", cds.getDataTier());
        assertNotNull(cds.getPatientState());
        assertNotNull(cds.getPhaseResults());
        assertTrue(cds.getPhaseResults().isEmpty());
        assertNotNull(cds.getRecommendations());
    }

    @Test
    void addPhaseResultTyped() {
        CDSEvent cds = new CDSEvent();
        cds.setPatientId("P-002");

        CDSPhaseResult phase1 = new CDSPhaseResult("PHASE_1_PROTOCOL_MATCH");
        phase1.setActive(true);
        phase1.addDetail("matchedCount", 3);
        phase1.addDetail("topProtocol", "SEPSIS-BUNDLE-V2");

        cds.addPhaseResult(phase1);

        assertEquals(1, cds.getPhaseResults().size());
        CDSPhaseResult retrieved = cds.getPhaseResults().get(0);
        assertEquals("PHASE_1_PROTOCOL_MATCH", retrieved.getPhaseName());
        assertTrue(retrieved.isActive());
        assertEquals(3, retrieved.getDetail("matchedCount"));
    }

    @Test
    void serialization_excludesEmptyCollections() throws Exception {
        CDSEvent cds = new CDSEvent();
        cds.setPatientId("P-003");

        com.fasterxml.jackson.databind.ObjectMapper mapper = new com.fasterxml.jackson.databind.ObjectMapper();
        String json = mapper.writeValueAsString(cds);

        assertTrue(json.contains("\"patientId\""));
        // Empty lists excluded by @JsonInclude(NON_EMPTY)
        assertFalse(json.contains("\"recommendations\""));
    }
}
