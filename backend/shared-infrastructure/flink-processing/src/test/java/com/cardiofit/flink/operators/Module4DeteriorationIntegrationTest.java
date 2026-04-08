package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.LabResult;
import com.cardiofit.flink.models.PatientContextState;
import org.junit.jupiter.api.Test;

import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Supplementary Task 9b: Deterioration Integration Tests
 *
 * Simulates a patient deteriorating over 3 time points and verifies that
 * the Module 4 conversion pipeline (vitals → scoring → risk level) correctly
 * tracks the escalation through CEP pattern thresholds.
 *
 * Time points:
 *   T0: Baseline — low risk, significance < 0.3
 *   T1: Early deterioration — moderate risk, significance crosses 0.6 (warning)
 *   T2: Severe deterioration — high risk, significance crosses 0.8 (critical)
 *
 * This validates the same code path as Module4_PatternDetection.convertCDSEventToSemanticEvent()
 * using the extracted helpers it delegates to.
 */
public class Module4DeteriorationIntegrationTest {

    // ── T0 → T1 → T2: Full Deterioration Trajectory ──────────────────

    @Test
    void deterioration_T0_baseline_lowRisk() {
        PatientContextState t0 = buildTimepoint_T0("P-DET-001");

        Map<String, Object> vitals = Module4SemanticConverter.normalizeVitalSigns(t0.getLatestVitals());
        assertEquals(6, vitals.size(), "T0: all vitals present");
        assertEquals(120, vitals.get("systolic_bp"), "T0: normal SBP");

        double sig = Module4ClinicalScoring.calculateClinicalSignificance(
            t0.getNews2Score(), t0.getQsofaScore(), t0.getCombinedAcuityScore());
        assertTrue(sig < 0.3,
            "T0 significance should be < 0.3 (low), got " + sig);

        String risk = Module4ClinicalScoring.determineRiskLevel(
            t0.getNews2Score(), t0.getQsofaScore(), null);
        assertEquals("low", risk, "T0 should be low risk");
    }

    @Test
    void deterioration_T1_earlyDeterioration_crossesWarningThreshold() {
        PatientContextState t1 = buildTimepoint_T1("P-DET-001");

        Map<String, Object> vitals = Module4SemanticConverter.normalizeVitalSigns(t1.getLatestVitals());
        assertEquals(105, vitals.get("heart_rate"), "T1: tachycardic");
        assertEquals(98, vitals.get("systolic_bp"), "T1: borderline hypotension");

        // Labs should show early infection markers
        Map<String, Object> labs = Module4SemanticConverter.extractLabValues(t1.getRecentLabs());
        assertEquals(2.5, labs.get("lactate"), "T1: mildly elevated lactate");

        double sig = Module4ClinicalScoring.calculateClinicalSignificance(
            t1.getNews2Score(), t1.getQsofaScore(), t1.getCombinedAcuityScore());
        // NEWS2=6 → 0.35, qSOFA=1 → 0.15, acuity=5.0 → 0.10 = 0.60
        assertTrue(sig >= 0.6,
            "T1 significance must cross WARNING threshold (0.6), got " + sig);
        assertTrue(sig < 0.8,
            "T1 significance should NOT cross CRITICAL threshold, got " + sig);

        String risk = Module4ClinicalScoring.determineRiskLevel(
            t1.getNews2Score(), t1.getQsofaScore(), null);
        assertEquals("moderate", risk, "T1 should be moderate risk");
    }

    @Test
    void deterioration_T2_severeDeterioration_crossesCriticalThreshold() {
        PatientContextState t2 = buildTimepoint_T2("P-DET-001");

        Map<String, Object> vitals = Module4SemanticConverter.normalizeVitalSigns(t2.getLatestVitals());
        assertEquals(130, vitals.get("heart_rate"), "T2: severe tachycardia");
        assertEquals(78, vitals.get("systolic_bp"), "T2: hypotension");
        assertEquals(86, vitals.get("oxygen_saturation"), "T2: hypoxia");

        // Labs should show sepsis progression
        Map<String, Object> labs = Module4SemanticConverter.extractLabValues(t2.getRecentLabs());
        assertEquals(5.2, labs.get("lactate"), "T2: severely elevated lactate");
        assertEquals(22, labs.get("wbc_count"), "T2: leukocytosis (integer)");

        double sig = Module4ClinicalScoring.calculateClinicalSignificance(
            t2.getNews2Score(), t2.getQsofaScore(), t2.getCombinedAcuityScore());
        // NEWS2=10 → 0.50, qSOFA=2 → 0.30, acuity=8.5 → 0.17 = 0.97
        assertTrue(sig >= 0.8,
            "T2 significance must cross CRITICAL threshold (0.8), got " + sig);

        String risk = Module4ClinicalScoring.determineRiskLevel(
            t2.getNews2Score(), t2.getQsofaScore(), null);
        assertEquals("high", risk, "T2 should be high risk");
    }

    // ── Monotonic Escalation: sig(T0) < sig(T1) < sig(T2) ────────────

    @Test
    void deterioration_significanceMonotonicallyIncreases() {
        PatientContextState t0 = buildTimepoint_T0("P-DET-002");
        PatientContextState t1 = buildTimepoint_T1("P-DET-002");
        PatientContextState t2 = buildTimepoint_T2("P-DET-002");

        double sigT0 = Module4ClinicalScoring.calculateClinicalSignificance(
            t0.getNews2Score(), t0.getQsofaScore(), t0.getCombinedAcuityScore());
        double sigT1 = Module4ClinicalScoring.calculateClinicalSignificance(
            t1.getNews2Score(), t1.getQsofaScore(), t1.getCombinedAcuityScore());
        double sigT2 = Module4ClinicalScoring.calculateClinicalSignificance(
            t2.getNews2Score(), t2.getQsofaScore(), t2.getCombinedAcuityScore());

        assertTrue(sigT0 < sigT1,
            "Significance must increase T0→T1: " + sigT0 + " < " + sigT1);
        assertTrue(sigT1 < sigT2,
            "Significance must increase T1→T2: " + sigT1 + " < " + sigT2);

        // Verify threshold crossings
        assertTrue(sigT0 < 0.6, "T0 below warning threshold");
        assertTrue(sigT1 >= 0.6 && sigT1 < 0.8, "T1 in warning band");
        assertTrue(sigT2 >= 0.8, "T2 above critical threshold");
    }

    // ── Vitals Consistency: BP Mapped at Every Timepoint ──────────────

    @Test
    void deterioration_bpMappedAtAllTimepoints() {
        // Regression guard: systolicbloodpressure must resolve to systolic_bp at every stage
        PatientContextState t0 = buildTimepoint_T0("P-DET-003");
        PatientContextState t1 = buildTimepoint_T1("P-DET-003");
        PatientContextState t2 = buildTimepoint_T2("P-DET-003");

        Map<String, Object> v0 = Module4SemanticConverter.normalizeVitalSigns(t0.getLatestVitals());
        Map<String, Object> v1 = Module4SemanticConverter.normalizeVitalSigns(t1.getLatestVitals());
        Map<String, Object> v2 = Module4SemanticConverter.normalizeVitalSigns(t2.getLatestVitals());

        assertNotNull(v0.get("systolic_bp"), "T0 must have systolic_bp");
        assertNotNull(v1.get("systolic_bp"), "T1 must have systolic_bp");
        assertNotNull(v2.get("systolic_bp"), "T2 must have systolic_bp");

        assertNotNull(v0.get("diastolic_bp"), "T0 must have diastolic_bp");
        assertNotNull(v1.get("diastolic_bp"), "T1 must have diastolic_bp");
        assertNotNull(v2.get("diastolic_bp"), "T2 must have diastolic_bp");

        // BP should be dropping across timepoints
        int sbp0 = (int) v0.get("systolic_bp");
        int sbp1 = (int) v1.get("systolic_bp");
        int sbp2 = (int) v2.get("systolic_bp");
        assertTrue(sbp0 > sbp1 && sbp1 > sbp2,
            "SBP should drop: " + sbp0 + " > " + sbp1 + " > " + sbp2);
    }

    // ── Timepoint Builders ────────────────────────────────────────────

    /**
     * T0: Baseline — normal vitals, NEWS2=2, qSOFA=0, acuity=1.5
     */
    private static PatientContextState buildTimepoint_T0(String patientId) {
        PatientContextState state = new PatientContextState(patientId);
        state.getLatestVitals().put("heartrate", 76);
        state.getLatestVitals().put("systolicbloodpressure", 120);
        state.getLatestVitals().put("diastolicbloodpressure", 78);
        state.getLatestVitals().put("respiratoryrate", 15);
        state.getLatestVitals().put("temperature", 37.0);
        state.getLatestVitals().put("oxygensaturation", 97);
        state.setNews2Score(2);
        state.setQsofaScore(0);
        state.setCombinedAcuityScore(1.5);
        return state;
    }

    /**
     * T1: Early deterioration — tachycardic, borderline hypotension, mild lactate elevation
     * NEWS2=6, qSOFA=1, acuity=5.0 → significance ~0.60 (crosses warning)
     */
    private static PatientContextState buildTimepoint_T1(String patientId) {
        PatientContextState state = new PatientContextState(patientId);
        state.getLatestVitals().put("heartrate", 105);
        state.getLatestVitals().put("systolicbloodpressure", 98);
        state.getLatestVitals().put("diastolicbloodpressure", 60);
        state.getLatestVitals().put("respiratoryrate", 22);
        state.getLatestVitals().put("temperature", 38.4);
        state.getLatestVitals().put("oxygensaturation", 93);

        LabResult lactate = new LabResult();
        lactate.setLabCode("2524-7");
        lactate.setValue(2.5);
        lactate.setLabType("Lactate");
        lactate.setUnit("mmol/L");
        state.getRecentLabs().put("2524-7", lactate);

        state.setNews2Score(6);
        state.setQsofaScore(1);
        state.setCombinedAcuityScore(5.0);
        return state;
    }

    /**
     * T2: Severe deterioration — septic shock profile, severe lactate, leukocytosis
     * NEWS2=10, qSOFA=2, acuity=8.5 → significance ~0.97 (crosses critical)
     */
    private static PatientContextState buildTimepoint_T2(String patientId) {
        PatientContextState state = new PatientContextState(patientId);
        state.getLatestVitals().put("heartrate", 130);
        state.getLatestVitals().put("systolicbloodpressure", 78);
        state.getLatestVitals().put("diastolicbloodpressure", 45);
        state.getLatestVitals().put("respiratoryrate", 30);
        state.getLatestVitals().put("temperature", 39.8);
        state.getLatestVitals().put("oxygensaturation", 86);

        LabResult lactate = new LabResult();
        lactate.setLabCode("2524-7");
        lactate.setValue(5.2);
        lactate.setLabType("Lactate");
        lactate.setUnit("mmol/L");
        state.getRecentLabs().put("2524-7", lactate);

        LabResult wbc = new LabResult();
        wbc.setLabCode("6690-2");
        wbc.setValue(22.0);
        wbc.setLabType("WBC");
        wbc.setUnit("10^3/uL");
        state.getRecentLabs().put("6690-2", wbc);

        state.setNews2Score(10);
        state.setQsofaScore(2);
        state.setCombinedAcuityScore(8.5);
        return state;
    }
}
