package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module8TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;

import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

class Module8HALTRulesTest {

    private static final long NOW = System.currentTimeMillis();

    // ── CID-01: Triple Whammy AKI ──

    @Test
    void cid01_tripleWhammy_fires() {
        ComorbidityState state = Module8TestBuilder.tripleWhammyPatient("P-TW");
        List<CIDAlert> alerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertTrue(alerts.stream().anyMatch(a -> "CID_01".equals(a.getRuleId())),
            "Triple Whammy with weight drop should fire CID-01");
    }

    @Test
    void cid01_tripleWhammy_noPrecipitant_doesNotFire() {
        ComorbidityState state = Module8TestBuilder.tripleWhammyNoPrecipitant("P-TW-NP");
        List<CIDAlert> alerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertFalse(alerts.stream().anyMatch(a -> "CID_01".equals(a.getRuleId())),
            "Triple Whammy without precipitant should NOT fire CID-01");
    }

    @Test
    void cid01_missingDiuretic_doesNotFire() {
        ComorbidityState state = new ComorbidityState("P-TW-ND");
        state.addMedication("ramipril", "ACEI", 10.0);
        state.addMedication("empagliflozin", "SGLT2I", 10.0);
        // No diuretic
        state.addToRollingBuffer("weight", 75.0, NOW - 7L * 86_400_000L);
        state.addToRollingBuffer("weight", 72.0, NOW);
        List<CIDAlert> alerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertFalse(alerts.stream().anyMatch(a -> "CID_01".equals(a.getRuleId())),
            "Only 2 of 3 nephrotoxic classes — CID-01 should not fire");
    }

    // ── CID-02: Hyperkalemia Cascade ──

    @Test
    void cid02_hyperkalemia_fires() {
        ComorbidityState state = Module8TestBuilder.hyperkalemiaPatient("P-HK");
        List<CIDAlert> alerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertTrue(alerts.stream().anyMatch(a -> "CID_02".equals(a.getRuleId())),
            "ACEi + finerenone + K+ 5.5 should fire CID-02");
    }

    @Test
    void cid02_normalPotassium_doesNotFire() {
        ComorbidityState state = new ComorbidityState("P-HK-NK");
        state.addMedication("ramipril", "ACEI", 10.0);
        state.addMedication("finerenone", "FINERENONE", 20.0);
        state.updateLab("potassium", 4.5); // normal K+
        List<CIDAlert> alerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertFalse(alerts.stream().anyMatch(a -> "CID_02".equals(a.getRuleId())),
            "Normal K+ should NOT fire CID-02");
    }

    // ── CID-03: Hypoglycemia Masking ──

    @Test
    void cid03_hypoMasking_fires() {
        ComorbidityState state = Module8TestBuilder.hypoMaskingPatient("P-HM");
        List<CIDAlert> alerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertTrue(alerts.stream().anyMatch(a -> "CID_03".equals(a.getRuleId())),
            "Insulin + BB + glucose <60 + no symptoms should fire CID-03");
    }

    @Test
    void cid03_hypoWithSymptoms_doesNotFire() {
        ComorbidityState state = Module8TestBuilder.hypoWithSymptoms("P-HM-S");
        List<CIDAlert> alerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertFalse(alerts.stream().anyMatch(a -> "CID_03".equals(a.getRuleId())),
            "Patient reporting symptoms = not masked = CID-03 should NOT fire");
    }

    // ── CID-04: Euglycemic DKA ──

    @Test
    void cid04_euglycemicDKA_fires() {
        ComorbidityState state = Module8TestBuilder.euglycemicDKAPatient("P-DKA");
        List<CIDAlert> alerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertTrue(alerts.stream().anyMatch(a -> "CID_04".equals(a.getRuleId())),
            "SGLT2i + nausea/vomiting should fire CID-04");
    }

    @Test
    void cid04_sglt2iWithoutSymptoms_doesNotFire() {
        ComorbidityState state = new ComorbidityState("P-DKA-NS");
        state.addMedication("dapagliflozin", "SGLT2I", 10.0);
        state.setSymptomReportedNauseaVomiting(false);
        state.setSymptomReportedKetoDiet(false);
        List<CIDAlert> alerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertFalse(alerts.stream().anyMatch(a -> "CID_04".equals(a.getRuleId())),
            "SGLT2i without DKA triggers should NOT fire CID-04");
    }

    // ── CID-05: Severe Hypotension ──

    @Test
    void cid05_severeHypotension_fires() {
        ComorbidityState state = Module8TestBuilder.severeHypotensionPatient("P-HYPO");
        List<CIDAlert> alerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertTrue(alerts.stream().anyMatch(a -> "CID_05".equals(a.getRuleId())),
            "3+ AH + SGLT2i + SBP <95 should fire CID-05");
    }

    @Test
    void cid05_normalBP_doesNotFire() {
        ComorbidityState state = new ComorbidityState("P-HYPO-N");
        state.addMedication("ramipril", "ACEI", 10.0);
        state.addMedication("amlodipine", "CCB", 10.0);
        state.addMedication("chlorthalidone", "THIAZIDE", 12.5);
        state.addMedication("empagliflozin", "SGLT2I", 10.0);
        state.setLatestSBP(125.0); // normal
        List<CIDAlert> alerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertFalse(alerts.stream().anyMatch(a -> "CID_05".equals(a.getRuleId())),
            "Normal SBP should NOT fire CID-05");
    }

    // ── All HALT rules are HALT severity ──

    @Test
    void allHALTAlerts_haveSeverityHALT() {
        ComorbidityState state = Module8TestBuilder.tripleWhammyPatient("P-SEV");
        List<CIDAlert> alerts = Module8HALTEvaluator.evaluate(state, NOW);
        for (CIDAlert alert : alerts) {
            assertEquals("HALT", alert.getSeverity(),
                "All HALT evaluator alerts must have HALT severity");
        }
    }

    // ── Safe patient fires no HALT rules ──

    @Test
    void safePatient_noHALTAlerts() {
        ComorbidityState state = Module8TestBuilder.safePatient("P-SAFE");
        List<CIDAlert> alerts = Module8HALTEvaluator.evaluate(state, NOW);
        assertTrue(alerts.isEmpty(),
            "Safe patient should trigger zero HALT alerts");
    }
}
