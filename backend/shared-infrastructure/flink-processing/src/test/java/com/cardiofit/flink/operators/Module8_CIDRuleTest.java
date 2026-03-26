package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.ComorbidityAlert;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.DisplayName;
import static org.junit.jupiter.api.Assertions.*;

import java.util.*;

public class Module8_CIDRuleTest {

    // ==================== HALT RULES ====================

    @Test
    @DisplayName("CID-01: Triple Whammy AKI fires when RASi + SGLT2i + diuretic + eGFR drop >20%")
    void cid01_tripleWhammy_firesOnEGFRDrop() {
        Map<String, Object> state = new HashMap<>();
        state.put("activeMeds", List.of("ACEI", "SGLT2I", "THIAZIDE"));
        state.put("currentEGFR", 45.0);
        state.put("previousEGFR", 62.0); // 27% drop

        List<ComorbidityAlert> alerts = CIDRuleEvaluatorTestHelper.evaluateCID01(state);

        assertEquals(1, alerts.size());
        assertEquals("CID-01", alerts.get(0).getRuleId());
        assertEquals(ComorbidityAlert.AlertSeverity.HALT, alerts.get(0).getSeverity());
    }

    @Test
    @DisplayName("CID-01: No alert when only 2 of 3 drug classes present")
    void cid01_noAlert_whenMissingDrugClass() {
        Map<String, Object> state = new HashMap<>();
        state.put("activeMeds", List.of("ACEI", "SGLT2I")); // no diuretic
        state.put("currentEGFR", 45.0);
        state.put("previousEGFR", 62.0);

        List<ComorbidityAlert> alerts = CIDRuleEvaluatorTestHelper.evaluateCID01(state);
        assertTrue(alerts.isEmpty());
    }

    @Test
    @DisplayName("CID-02: Hyperkalemia fires when RASi + finerenone + K+ >5.3 rising")
    void cid02_hyperkalemia_fires() {
        Map<String, Object> state = new HashMap<>();
        state.put("activeMeds", List.of("ARB", "FINERENONE"));
        state.put("currentK", 5.5);
        state.put("previousK", 5.1);

        List<ComorbidityAlert> alerts = CIDRuleEvaluatorTestHelper.evaluateCID02(state);
        assertEquals(1, alerts.size());
        assertEquals("CID-02", alerts.get(0).getRuleId());
    }

    @Test
    @DisplayName("CID-03: Hypoglycemia masking fires when insulin/SU + beta-blocker + glucose <60")
    void cid03_hypoMasking_fires() {
        Map<String, Object> state = new HashMap<>();
        state.put("activeMeds", List.of("INSULIN", "BETA_BLOCKER"));
        state.put("currentGlucose", 55.0);
        state.put("symptomReportPresent", false);

        List<ComorbidityAlert> alerts = CIDRuleEvaluatorTestHelper.evaluateCID03(state);
        assertEquals(1, alerts.size());
        assertEquals("CID-03", alerts.get(0).getRuleId());
        assertEquals(ComorbidityAlert.AlertSeverity.HALT, alerts.get(0).getSeverity());
    }

    @Test
    @DisplayName("CID-03: No alert when symptoms ARE reported (patient aware of hypo)")
    void cid03_noAlert_whenSymptomsReported() {
        Map<String, Object> state = new HashMap<>();
        state.put("activeMeds", List.of("SULFONYLUREA", "BETA_BLOCKER"));
        state.put("currentGlucose", 55.0);
        state.put("symptomReportPresent", true);
        assertTrue(CIDRuleEvaluatorTestHelper.evaluateCID03(state).isEmpty());
    }

    @Test
    @DisplayName("CID-04: Euglycemic DKA fires when SGLT2i + nausea/vomiting context")
    void cid04_euglycemicDKA_fires() {
        Map<String, Object> state = new HashMap<>();
        state.put("activeMeds", List.of("SGLT2I"));
        state.put("nauseaVomitingSignal", true);

        List<ComorbidityAlert> alerts = CIDRuleEvaluatorTestHelper.evaluateCID04(state);
        assertEquals(1, alerts.size());
        assertEquals("CID-04", alerts.get(0).getRuleId());
    }

    @Test
    @DisplayName("CID-05: Severe hypotension fires when >=3 antihypertensives + SGLT2i + SBP <95")
    void cid05_severeHypotension_fires() {
        Map<String, Object> state = new HashMap<>();
        state.put("activeMeds", List.of("ACEI", "CCB", "THIAZIDE", "SGLT2I"));
        state.put("currentSBP", 92.0);

        List<ComorbidityAlert> alerts = CIDRuleEvaluatorTestHelper.evaluateCID05(state);
        assertEquals(1, alerts.size());
        assertEquals("CID-05", alerts.get(0).getRuleId());
    }

    // ==================== PAUSE RULES ====================

    @Test
    @DisplayName("CID-06: Hyponatremia fires when thiazide + loop + Na+ <130 falling")
    void cid06_hyponatremia_fires() {
        Map<String, Object> state = new HashMap<>();
        state.put("activeMeds", List.of("THIAZIDE", "LOOP_DIURETIC"));
        state.put("currentNa", 127.0);
        state.put("previousNa", 133.0);
        List<ComorbidityAlert> alerts = CIDRuleEvaluatorTestHelper.evaluateCID06(state);
        assertEquals(1, alerts.size());
        assertEquals(ComorbidityAlert.AlertSeverity.PAUSE, alerts.get(0).getSeverity());
    }

    @Test
    @DisplayName("CID-07: Recurrent hypo fires when GLP-1RA + SU + FBG <70")
    void cid07_recurrentHypo_fires() {
        Map<String, Object> state = new HashMap<>();
        state.put("activeMeds", List.of("GLP1RA", "SULFONYLUREA"));
        state.put("currentFBG", 65.0);
        List<ComorbidityAlert> alerts = CIDRuleEvaluatorTestHelper.evaluateCID07(state);
        assertEquals(1, alerts.size());
        assertEquals(ComorbidityAlert.AlertSeverity.PAUSE, alerts.get(0).getSeverity());
    }

    @Test
    @DisplayName("CID-08: Volume depletion fires when thiazide + SGLT2i + rising Na >145")
    void cid08_volumeDepletion_fires() {
        Map<String, Object> state = new HashMap<>();
        state.put("activeMeds", List.of("THIAZIDE", "SGLT2I"));
        state.put("currentNa", 148.0);
        state.put("previousNa", 143.0);
        List<ComorbidityAlert> alerts = CIDRuleEvaluatorTestHelper.evaluateCID08(state);
        assertEquals(1, alerts.size());
        assertEquals(ComorbidityAlert.AlertSeverity.PAUSE, alerts.get(0).getSeverity());
    }

    @Test
    @DisplayName("CID-09: Heart rate masking fires when beta-blocker + GLP-1RA")
    void cid09_heartRateMasking_fires() {
        Map<String, Object> state = new HashMap<>();
        state.put("activeMeds", List.of("BETA_BLOCKER", "GLP1RA"));
        List<ComorbidityAlert> alerts = CIDRuleEvaluatorTestHelper.evaluateCID09(state);
        assertEquals(1, alerts.size());
        assertEquals(ComorbidityAlert.AlertSeverity.PAUSE, alerts.get(0).getSeverity());
    }

    // ==================== SOFT_FLAG RULES ====================

    @Test
    @DisplayName("CID-11: Metformin dose cap fires when eGFR 30-45")
    void cid11_metforminCap_fires() {
        Map<String, Object> state = new HashMap<>();
        state.put("activeMeds", List.of("METFORMIN"));
        state.put("currentEGFR", 38.0);
        List<ComorbidityAlert> alerts = CIDRuleEvaluatorTestHelper.evaluateCID11(state);
        assertEquals(1, alerts.size());
        assertEquals(ComorbidityAlert.AlertSeverity.SOFT_FLAG, alerts.get(0).getSeverity());
    }

    @Test
    @DisplayName("CID-12: Statin-fibrate myopathy risk fires")
    void cid12_statinFibrate_fires() {
        Map<String, Object> state = new HashMap<>();
        state.put("activeMeds", List.of("STATIN", "FIBRATE"));
        List<ComorbidityAlert> alerts = CIDRuleEvaluatorTestHelper.evaluateCID12(state);
        assertEquals(1, alerts.size());
        assertEquals(ComorbidityAlert.AlertSeverity.SOFT_FLAG, alerts.get(0).getSeverity());
    }

    @Test
    @DisplayName("CID-13: Expected eGFR dip fires when 10-20% drop on RASi + SGLT2i")
    void cid13_expectedDip_fires() {
        Map<String, Object> state = new HashMap<>();
        state.put("activeMeds", List.of("ACEI", "SGLT2I"));
        state.put("currentEGFR", 54.0);
        state.put("previousEGFR", 62.0); // 12.9% drop
        List<ComorbidityAlert> alerts = CIDRuleEvaluatorTestHelper.evaluateCID13(state);
        assertEquals(1, alerts.size());
        assertEquals(ComorbidityAlert.AlertSeverity.SOFT_FLAG, alerts.get(0).getSeverity());
    }

    @Test
    @DisplayName("CID-14: Triple antithrombotic risk fires")
    void cid14_tripleAntithrombotic_fires() {
        Map<String, Object> state = new HashMap<>();
        state.put("activeMeds", List.of("ASPIRIN", "WARFARIN"));
        List<ComorbidityAlert> alerts = CIDRuleEvaluatorTestHelper.evaluateCID14(state);
        assertEquals(1, alerts.size());
        assertEquals(ComorbidityAlert.AlertSeverity.SOFT_FLAG, alerts.get(0).getSeverity());
    }

    @Test
    @DisplayName("CID-15: NSAID-RASi renal risk fires")
    void cid15_nsaidRasi_fires() {
        Map<String, Object> state = new HashMap<>();
        state.put("activeMeds", List.of("NSAID", "ARB"));
        List<ComorbidityAlert> alerts = CIDRuleEvaluatorTestHelper.evaluateCID15(state);
        assertEquals(1, alerts.size());
        assertEquals(ComorbidityAlert.AlertSeverity.SOFT_FLAG, alerts.get(0).getSeverity());
    }

    @Test
    @DisplayName("CID-16: Bradycardia risk fires when beta-blocker + CCB")
    void cid16_bradycardia_fires() {
        Map<String, Object> state = new HashMap<>();
        state.put("activeMeds", List.of("BETA_BLOCKER", "CCB"));
        List<ComorbidityAlert> alerts = CIDRuleEvaluatorTestHelper.evaluateCID16(state);
        assertEquals(1, alerts.size());
        assertEquals(ComorbidityAlert.AlertSeverity.SOFT_FLAG, alerts.get(0).getSeverity());
    }

    @Test
    @DisplayName("CID-17: Fasting period risk fires when SGLT2i + >=3 meal skips")
    void cid17_fastingRisk_fires() {
        Map<String, Object> state = new HashMap<>();
        state.put("activeMeds", List.of("SGLT2I", "INSULIN"));
        state.put("mealSkips24h", 3);
        List<ComorbidityAlert> alerts = CIDRuleEvaluatorTestHelper.evaluateCID17(state);
        assertEquals(1, alerts.size());
        assertEquals(ComorbidityAlert.AlertSeverity.SOFT_FLAG, alerts.get(0).getSeverity());
    }
}
