package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module8TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;

class Module8PAUSERulesTest {

    private static final long NOW = System.currentTimeMillis();

    @Test
    void cid06_thiazideGlucose_fires() {
        ComorbidityState state = Module8TestBuilder.thiazideGlucosePatient("P-TG");
        List<CIDAlert> alerts = Module8PAUSEEvaluator.evaluate(state, NOW);
        assertTrue(alerts.stream().anyMatch(a -> "CID_06".equals(a.getRuleId())),
            "Thiazide + FBG increase >15 mg/dL should fire CID-06");
    }

    @Test
    void cid06_thiazideSmallGlucoseChange_doesNotFire() {
        ComorbidityState state = new ComorbidityState("P-TG-S");
        state.addMedication("hydrochlorothiazide", "THIAZIDE", 25.0);
        // FBG: 112 → 118 = delta +6 — below 15 threshold
        state.addToRollingBuffer("fbg", 112.0, NOW - 14L * 86_400_000L);
        state.addToRollingBuffer("fbg", 118.0, NOW);
        List<CIDAlert> alerts = Module8PAUSEEvaluator.evaluate(state, NOW);
        assertFalse(alerts.stream().anyMatch(a -> "CID_06".equals(a.getRuleId())));
    }

    @Test
    void cid07_eGFRDecline_fires() {
        ComorbidityState state = Module8TestBuilder.eGFRDeclinePatient("P-EGD");
        List<CIDAlert> alerts = Module8PAUSEEvaluator.evaluate(state, NOW);
        assertTrue(alerts.stream().anyMatch(a -> "CID_07".equals(a.getRuleId())),
            "ARB + eGFR drop >25% + >6 weeks should fire CID-07");
    }

    @Test
    void cid07_expectedDip_doesNotFire() {
        ComorbidityState state = new ComorbidityState("P-EGD-ED");
        state.addMedication("telmisartan", "ARB", 80.0);
        state.setEGFRBaseline(65.0);
        state.setEGFRCurrent(55.0); // 15.4% decline — within expected 10-25% dip
        state.setEGFRBaselineTimestamp(System.currentTimeMillis() - 21 * 86400000L); // 3 weeks — within dip window
        state.setEGFRCurrentTimestamp(System.currentTimeMillis());
        List<CIDAlert> alerts = Module8PAUSEEvaluator.evaluate(state, NOW);
        assertFalse(alerts.stream().anyMatch(a -> "CID_07".equals(a.getRuleId())),
            "Expected eGFR dip within 6 weeks should NOT fire CID-07");
    }

    @Test
    void cid08_statinMyopathy_fires() {
        ComorbidityState state = Module8TestBuilder.statinMyopathyPatient("P-SM");
        List<CIDAlert> alerts = Module8PAUSEEvaluator.evaluate(state, NOW);
        assertTrue(alerts.stream().anyMatch(a -> "CID_08".equals(a.getRuleId())),
            "Statin + muscle pain should fire CID-08");
    }

    @Test
    void cid09_glp1raGI_fires() {
        ComorbidityState state = Module8TestBuilder.glp1raGIPatient("P-GI");
        List<CIDAlert> alerts = Module8PAUSEEvaluator.evaluate(state, NOW);
        assertTrue(alerts.stream().anyMatch(a -> "CID_09".equals(a.getRuleId())),
            "GLP-1RA + nausea + weight drop >1.5kg should fire CID-09");
    }

    @Test
    void cid10_concurrentDeterioration_fires() {
        ComorbidityState state = Module8TestBuilder.concurrentDeteriorationPatient("P-CD");
        List<CIDAlert> alerts = Module8PAUSEEvaluator.evaluate(state, NOW);
        assertTrue(alerts.stream().anyMatch(a -> "CID_10".equals(a.getRuleId())),
            "Both glucose AND BP worsening should fire CID-10");
    }

    @Test
    void cid10_onlyGlucoseWorsening_doesNotFire() {
        ComorbidityState state = new ComorbidityState("P-CD-G");
        // FBG worsening: 135 → 165
        state.addToRollingBuffer("fbg", 135.0, NOW - 14L * 86_400_000L);
        state.addToRollingBuffer("fbg", 165.0, NOW);
        // SBP stable/improving: 134 → 132 (no worsening)
        state.addToRollingBuffer("sbp", 134.0, NOW - 14L * 86_400_000L);
        state.addToRollingBuffer("sbp", 132.0, NOW);
        state.setLastMedicationChangeTimestamp(NOW - 30L * 86_400_000L);
        List<CIDAlert> alerts = Module8PAUSEEvaluator.evaluate(state, NOW);
        assertFalse(alerts.stream().anyMatch(a -> "CID_10".equals(a.getRuleId())),
            "Only glucose worsening (no BP worsening) should NOT fire CID-10");
    }

    @Test
    void allPAUSEAlerts_haveSeverityPAUSE() {
        ComorbidityState state = Module8TestBuilder.thiazideGlucosePatient("P-SEV");
        List<CIDAlert> alerts = Module8PAUSEEvaluator.evaluate(state, NOW);
        for (CIDAlert alert : alerts) {
            assertEquals("PAUSE", alert.getSeverity());
        }
    }

    @Test
    void safePatient_noPAUSEAlerts() {
        ComorbidityState state = Module8TestBuilder.safePatient("P-SAFE");
        List<CIDAlert> alerts = Module8PAUSEEvaluator.evaluate(state, NOW);
        assertTrue(alerts.isEmpty());
    }
}
