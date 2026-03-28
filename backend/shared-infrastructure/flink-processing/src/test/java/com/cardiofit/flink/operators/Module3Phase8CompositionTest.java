package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module3TestBuilder;
import com.cardiofit.flink.models.*;
import com.cardiofit.flink.models.protocol.SimplifiedProtocol;
import org.junit.jupiter.api.Test;
import java.util.*;
import static org.junit.jupiter.api.Assertions.*;

public class Module3Phase8CompositionTest {

    @Test
    void phase8_aggregatesAllPhases_singlePass() {
        EnrichedPatientContext patient = Module3TestBuilder.hypertensiveDiabeticPatient("COMP-001");
        Map<String, SimplifiedProtocol> protocols = Module3TestBuilder.defaultProtocolMap();

        CDSPhaseResult phase1 = Module3PhaseExecutor.executePhase1(patient, protocols);
        CDSPhaseResult phase2 = Module3PhaseExecutor.executePhase2(patient);
        CDSPhaseResult phase4 = Module3PhaseExecutor.executePhase4(patient);

        @SuppressWarnings("unchecked")
        List<String> matchedIds = phase1.isActive()
                ? (List<String>) phase1.getDetail("matchedProtocolIds")
                : Collections.emptyList();
        CDSPhaseResult phase5 = Module3PhaseExecutor.executePhase5(patient, matchedIds, protocols);
        CDSPhaseResult phase6 = Module3PhaseExecutor.executePhase6(patient);
        CDSPhaseResult phase7 = Module3PhaseExecutor.executePhase7(patient);

        List<CDSPhaseResult> allResults = Arrays.asList(phase1, phase2, phase4, phase5, phase6, phase7);
        CDSEvent cdsEvent = new CDSEvent(patient);

        Module3PhaseExecutor.executePhase8(cdsEvent, allResults);

        // MHRI extracted from Phase 2
        assertNotNull(cdsEvent.getMhriScore());
        // Protocol count from Phase 1
        assertTrue(cdsEvent.getProtocolsMatched() >= 0);
        // Phase 8 itself was added
        assertTrue(cdsEvent.getPhaseResults().stream()
                .anyMatch(pr -> "PHASE_8_OUTPUT_COMPOSITION".equals(pr.getPhaseName())));
    }

    @Test
    void phase8_handlesEmptyPhaseResults() {
        EnrichedPatientContext patient = Module3TestBuilder.hypertensiveDiabeticPatient("COMP-002");
        CDSEvent cdsEvent = new CDSEvent(patient);
        List<CDSPhaseResult> empty = Collections.emptyList();

        Module3PhaseExecutor.executePhase8(cdsEvent, empty);

        assertEquals(0, cdsEvent.getRecommendations().size());
        assertEquals(0, cdsEvent.getSafetyAlerts().size());
    }
}
