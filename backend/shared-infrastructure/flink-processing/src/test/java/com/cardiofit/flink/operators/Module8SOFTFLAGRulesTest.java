package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module8TestBuilder;
import com.cardiofit.flink.models.*;
import org.junit.jupiter.api.Test;
import java.util.List;
import static org.junit.jupiter.api.Assertions.*;

class Module8SOFTFLAGRulesTest {

    @Test
    void cid12_polypharmacy_fires() {
        ComorbidityState state = Module8TestBuilder.polypharmacyPatient("P-POLY");
        List<CIDAlert> alerts = Module8SOFTFLAGEvaluator.evaluate(state, null);
        assertTrue(alerts.stream().anyMatch(a -> "CID_12".equals(a.getRuleId())),
            "9 medications should fire CID-12 polypharmacy warning");
    }

    @Test
    void cid12_fewMeds_doesNotFire() {
        ComorbidityState state = Module8TestBuilder.safePatient("P-FEW");
        List<CIDAlert> alerts = Module8SOFTFLAGEvaluator.evaluate(state, null);
        assertFalse(alerts.stream().anyMatch(a -> "CID_12".equals(a.getRuleId())),
            "2 medications should NOT fire CID-12");
    }

    @Test
    void cid13_elderlyIntensiveBP_fires() {
        ComorbidityState state = Module8TestBuilder.elderlyIntensiveBPPatient("P-ELD");
        // SBP target would be provided as parameter (from KB-20 patient profile)
        List<CIDAlert> alerts = Module8SOFTFLAGEvaluator.evaluate(state, 125.0);
        assertTrue(alerts.stream().anyMatch(a -> "CID_13".equals(a.getRuleId())),
            "Age 78 + SBP target 125 + eGFR <45 + falls should fire CID-13");
    }

    @Test
    void cid13_youngPatient_doesNotFire() {
        ComorbidityState state = new ComorbidityState("P-YOUNG");
        state.setAge(55);
        state.setEGFRCurrent(42.0);
        List<CIDAlert> alerts = Module8SOFTFLAGEvaluator.evaluate(state, 125.0);
        assertFalse(alerts.stream().anyMatch(a -> "CID_13".equals(a.getRuleId())),
            "Age 55 should NOT fire CID-13 (threshold is >= 75)");
    }

    @Test
    void cid14_metforminEGFR_fires() {
        ComorbidityState state = Module8TestBuilder.metforminEGFRPatient("P-MET");
        List<CIDAlert> alerts = Module8SOFTFLAGEvaluator.evaluate(state, null);
        assertTrue(alerts.stream().anyMatch(a -> "CID_14".equals(a.getRuleId())),
            "Metformin + eGFR 32 + declining should fire CID-14");
    }

    @Test
    void cid15_sglt2iNsaid_fires() {
        ComorbidityState state = Module8TestBuilder.sglt2iNsaidPatient("P-NSAID");
        List<CIDAlert> alerts = Module8SOFTFLAGEvaluator.evaluate(state, null);
        assertTrue(alerts.stream().anyMatch(a -> "CID_15".equals(a.getRuleId())),
            "SGLT2i + NSAID should fire CID-15");
    }

    @Test
    void cid11_genitalInfection_fires() {
        ComorbidityState state = new ComorbidityState("P-GI");
        state.addMedication("empagliflozin", "SGLT2I", 10.0);
        state.setGenitalInfectionHistory(true);
        List<CIDAlert> alerts = Module8SOFTFLAGEvaluator.evaluate(state, null);
        assertTrue(alerts.stream().anyMatch(a -> "CID_11".equals(a.getRuleId())),
            "SGLT2i + genital infection history should fire CID-11");
    }

    @Test
    void cid17_sglt2iFasting_fires() {
        ComorbidityState state = new ComorbidityState("P-FAST");
        state.addMedication("dapagliflozin", "SGLT2I", 10.0);
        state.setActiveFastingPeriod(true);
        state.setActiveFastingDurationHours(18); // >16h
        List<CIDAlert> alerts = Module8SOFTFLAGEvaluator.evaluate(state, null);
        assertTrue(alerts.stream().anyMatch(a -> "CID_17".equals(a.getRuleId())),
            "SGLT2i + fasting >16h should fire CID-17");
    }

    @Test
    void cid17_shortFast_doesNotFire() {
        ComorbidityState state = new ComorbidityState("P-FAST-S");
        state.addMedication("dapagliflozin", "SGLT2I", 10.0);
        state.setActiveFastingPeriod(true);
        state.setActiveFastingDurationHours(12); // <16h
        List<CIDAlert> alerts = Module8SOFTFLAGEvaluator.evaluate(state, null);
        assertFalse(alerts.stream().anyMatch(a -> "CID_17".equals(a.getRuleId())),
            "Fasting <16h should NOT fire CID-17");
    }

    @Test
    void allSOFTFLAGs_haveSeveritySOFTFLAG() {
        ComorbidityState state = Module8TestBuilder.polypharmacyPatient("P-SEV");
        List<CIDAlert> alerts = Module8SOFTFLAGEvaluator.evaluate(state, null);
        for (CIDAlert alert : alerts) {
            assertEquals("SOFT_FLAG", alert.getSeverity());
        }
    }

    @Test
    void safePatient_noSOFTFLAGs() {
        ComorbidityState state = Module8TestBuilder.safePatient("P-SAFE");
        List<CIDAlert> alerts = Module8SOFTFLAGEvaluator.evaluate(state, null);
        assertTrue(alerts.isEmpty());
    }
}
