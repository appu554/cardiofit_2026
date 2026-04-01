package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module8TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;

class Module8IntegrationTest {

    private static final long NOW = System.currentTimeMillis();

    @Test
    void tripleWhammy_producesHALTAlert() {
        ComorbidityState state = Module8TestBuilder.tripleWhammyPatient("P-INT-TW");
        List<CIDAlert> alerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertFalse(alerts.isEmpty());
        assertEquals("HALT", alerts.get(0).getSeverity());
        assertEquals("CID_01", alerts.get(0).getRuleId());
        assertNotNull(alerts.get(0).getAlertId());
        assertNotNull(alerts.get(0).getSuppressionKey());
    }

    @Test
    void safePatient_producesNoAlerts() {
        ComorbidityState state = Module8TestBuilder.safePatient("P-INT-SAFE");
        List<CIDAlert> halt = Module8HALTEvaluator.evaluate(state, NOW);
        List<CIDAlert> pause = Module8PAUSEEvaluator.evaluate(state, NOW);
        List<CIDAlert> soft = Module8SOFTFLAGEvaluator.evaluate(state, null);
        assertTrue(halt.isEmpty(), "Safe patient: no HALT");
        assertTrue(pause.isEmpty(), "Safe patient: no PAUSE");
        assertTrue(soft.isEmpty(), "Safe patient: no SOFT_FLAG");
    }

    @Test
    void multipleRulesCanFireSimultaneously() {
        ComorbidityState state = Module8TestBuilder.tripleWhammyPatient("P-MULTI");
        state.addMedication("ibuprofen", "NSAID", 400.0);

        List<CIDAlert> halt = Module8HALTEvaluator.evaluate(state, NOW);
        List<CIDAlert> soft = Module8SOFTFLAGEvaluator.evaluate(state, null);

        assertTrue(halt.stream().anyMatch(a -> "CID_01".equals(a.getRuleId())),
            "CID-01 should fire");
        assertTrue(soft.stream().anyMatch(a -> "CID_15".equals(a.getRuleId())),
            "CID-15 should also fire (SGLT2i + NSAID)");
    }

    @Test
    void suppressionPreventsRepeatedPAUSEAlerts() {
        ComorbidityState state = Module8TestBuilder.statinMyopathyPatient("P-SUP");

        List<CIDAlert> alerts1 = Module8PAUSEEvaluator.evaluate(state, NOW);
        assertFalse(alerts1.isEmpty());
        CIDAlert first = alerts1.get(0);

        Module8SuppressionManager.recordEmission(first, state, NOW);

        List<CIDAlert> alerts2 = Module8PAUSEEvaluator.evaluate(state, NOW + 12 * 3600_000L);
        for (CIDAlert a : alerts2) {
            if ("CID_08".equals(a.getRuleId())) {
                assertTrue(Module8SuppressionManager.shouldSuppress(a, state, NOW + 12 * 3600_000L),
                    "CID-08 should be suppressed within 72h");
            }
        }
    }

    @Test
    void haltAndPauseCanCoexist() {
        long nauseaOnset = NOW - 50L * 3600_000L;
        ComorbidityState state = new ComorbidityState("P-COEXIST");
        state.addMedication("dapagliflozin", "SGLT2I", 10.0);
        state.addMedication("semaglutide", "GLP1RA", 1.0);
        state.setSymptomReportedNauseaVomiting(true);
        state.setSymptomNauseaOnsetTimestamp(nauseaOnset);
        state.addToRollingBuffer("weight", 80.0, NOW - 7L * 86_400_000L);
        state.addToRollingBuffer("weight", 78.0, NOW);

        List<CIDAlert> halt = Module8HALTEvaluator.evaluate(state, NOW);
        List<CIDAlert> pause = Module8PAUSEEvaluator.evaluate(state, NOW);

        assertTrue(halt.stream().anyMatch(a -> "CID_04".equals(a.getRuleId())),
            "CID-04 HALT should fire (SGLT2i + nausea)");
        assertTrue(pause.stream().anyMatch(a -> "CID_09".equals(a.getRuleId())),
            "CID-09 PAUSE should also fire (GLP-1RA + GI ≥48h + weight drop)");
    }

    @Test
    void alertContainsMedicationList() {
        ComorbidityState state = Module8TestBuilder.sglt2iNsaidPatient("P-MEDS");
        List<CIDAlert> alerts = Module8SOFTFLAGEvaluator.evaluate(state, null);
        CIDAlert cid15 = alerts.stream()
            .filter(a -> "CID_15".equals(a.getRuleId()))
            .findFirst().orElse(null);
        assertNotNull(cid15);
        assertNotNull(cid15.getMedicationsInvolved());
        assertFalse(cid15.getMedicationsInvolved().isEmpty(),
            "Alert should list involved medications");
    }
}
