package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module4TestBuilder;
import com.cardiofit.flink.models.LabResult;
import com.cardiofit.flink.models.PatientContextState;
import com.cardiofit.flink.models.RiskIndicators;
import org.junit.jupiter.api.Test;

import java.util.HashMap;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Supplementary Task 0: Real CDS Mapping Tests
 *
 * Validates that production-shaped PatientContextState data flows correctly
 * through the Module4SemanticConverter extraction helpers and Module4ClinicalScoring.
 *
 * These tests mirror the conversion path inside Module4_PatternDetection.convertCDSEventToSemanticEvent()
 * (which is private) by calling the same extracted helpers it delegates to:
 *   - Module4SemanticConverter.normalizeVitalSigns()
 *   - Module4SemanticConverter.extractLabValues()
 *   - Module4SemanticConverter.extractRiskIndicators()
 *   - Module4ClinicalScoring.calculateClinicalSignificance()
 *   - Module4ClinicalScoring.determineRiskLevel()
 */
public class Module4RealCDSMappingTest {

    // ── Full Conversion Path: Production-Format Patient State ─────────

    @Test
    void fullPath_moderateRiskPatient_allFieldsMapped() {
        PatientContextState state = Module4TestBuilder.moderateRiskPatientState("P-MOD-001");

        // Step 1: Vital signs — uses production keys (systolicbloodpressure)
        Map<String, Object> vitals = Module4SemanticConverter.normalizeVitalSigns(state.getLatestVitals());
        assertEquals(6, vitals.size(), "All 6 vital signs should be extracted");
        assertEquals(102, vitals.get("heart_rate"));
        assertEquals(95, vitals.get("systolic_bp"), "systolicbloodpressure → systolic_bp");
        assertEquals(62, vitals.get("diastolic_bp"), "diastolicbloodpressure → diastolic_bp");
        assertEquals(22, vitals.get("respiratory_rate"));
        assertEquals(38.2, vitals.get("temperature"));
        assertEquals(94, vitals.get("oxygen_saturation"));

        // Step 2: Lab values
        Map<String, Object> labs = Module4SemanticConverter.extractLabValues(state.getRecentLabs());
        assertEquals(2.8, labs.get("lactate"), "Lactate by standardized name");
        assertEquals(2.8, labs.get("2524-7"), "Lactate by LOINC code");
        assertEquals(1.6, labs.get("creatinine"), "Creatinine by standardized name");
        assertEquals(1.6, labs.get("2160-0"), "Creatinine by LOINC code");

        // Step 3: Clinical significance
        double sig = Module4ClinicalScoring.calculateClinicalSignificance(
            state.getNews2Score(), state.getQsofaScore(), state.getCombinedAcuityScore());
        // NEWS2=5 → 0.35, qSOFA=1 → 0.15, acuity=5.5 → 0.11 = 0.61
        assertTrue(sig >= 0.55 && sig <= 0.70,
            "Moderate patient significance should be ~0.61, got " + sig);

        // Step 4: Risk level
        String risk = Module4ClinicalScoring.determineRiskLevel(
            state.getNews2Score(), state.getQsofaScore(), null);
        assertEquals("moderate", risk, "NEWS2=5 should yield moderate risk");
    }

    @Test
    void fullPath_highRiskSepticPatient_allFieldsMapped() {
        PatientContextState state = Module4TestBuilder.highRiskSepticPatientState("P-SEP-001");

        // Vitals
        Map<String, Object> vitals = Module4SemanticConverter.normalizeVitalSigns(state.getLatestVitals());
        assertEquals(6, vitals.size());
        assertEquals(125, vitals.get("heart_rate"));
        assertEquals(82, vitals.get("systolic_bp"), "Septic patient SBP=82 must be mapped");
        assertEquals(50, vitals.get("diastolic_bp"));
        assertEquals(88, vitals.get("oxygen_saturation"));

        // Labs — septic patient has lactate, WBC, procalcitonin
        Map<String, Object> labs = Module4SemanticConverter.extractLabValues(state.getRecentLabs());
        assertEquals(4.5, labs.get("lactate"), "Elevated lactate for sepsis");
        assertEquals(18, labs.get("wbc_count"), "WBC count as integer");
        assertEquals(8.2, labs.get("procalcitonin"), "Procalcitonin for sepsis");

        // Significance should be critical range
        double sig = Module4ClinicalScoring.calculateClinicalSignificance(
            state.getNews2Score(), state.getQsofaScore(), state.getCombinedAcuityScore());
        // NEWS2=10 → 0.50, qSOFA=2 → 0.30, acuity=9.0 → 0.18 = 0.98
        assertTrue(sig >= 0.90,
            "Septic patient significance should be >=0.90, got " + sig);

        String risk = Module4ClinicalScoring.determineRiskLevel(
            state.getNews2Score(), state.getQsofaScore(), null);
        assertEquals("high", risk, "NEWS2=10 should yield high risk");
    }

    @Test
    void fullPath_lowRiskPatient_baselineValues() {
        PatientContextState state = Module4TestBuilder.lowRiskPatientState("P-LOW-001");

        Map<String, Object> vitals = Module4SemanticConverter.normalizeVitalSigns(state.getLatestVitals());
        assertEquals(6, vitals.size());
        assertEquals(72, vitals.get("heart_rate"));
        assertEquals(122, vitals.get("systolic_bp"));
        assertEquals(78, vitals.get("diastolic_bp"));

        // No labs on low risk patient
        Map<String, Object> labs = Module4SemanticConverter.extractLabValues(state.getRecentLabs());
        assertTrue(labs.isEmpty(), "Low-risk patient has no labs");

        double sig = Module4ClinicalScoring.calculateClinicalSignificance(
            state.getNews2Score(), state.getQsofaScore(), state.getCombinedAcuityScore());
        // NEWS2=1 → 0.15, qSOFA=0, acuity=1.5 → 0.03 = 0.18
        assertTrue(sig < 0.3,
            "Low-risk patient significance should be <0.3, got " + sig);

        String risk = Module4ClinicalScoring.determineRiskLevel(
            state.getNews2Score(), state.getQsofaScore(), null);
        assertEquals("low", risk);
    }

    // ── Production Data Shape: Demographics in Vitals ─────────────────

    @Test
    void productionShape_demographicsInVitals_excludedFromMapping() {
        // Real production PatientContextAggregator stores age/gender inside latestVitals
        PatientContextState state = new PatientContextState("P-PROD-001");
        state.getLatestVitals().put("heartrate", 75);
        state.getLatestVitals().put("systolicbloodpressure", 130);
        state.getLatestVitals().put("diastolicbloodpressure", 85);
        state.getLatestVitals().put("respiratoryrate", 16);
        state.getLatestVitals().put("temperature", 37.0);
        state.getLatestVitals().put("oxygensaturation", 97);
        // Demographics — must NOT appear in normalized vitals
        state.getLatestVitals().put("age", 65);
        state.getLatestVitals().put("gender", "male");

        Map<String, Object> vitals = Module4SemanticConverter.normalizeVitalSigns(state.getLatestVitals());

        assertEquals(6, vitals.size(), "Only 6 vital signs, no demographics");
        assertNull(vitals.get("age"), "age must be excluded");
        assertNull(vitals.get("gender"), "gender must be excluded");
        assertEquals(130, vitals.get("systolic_bp"));
    }

    // ── Score Availability: Zero/Null Scores ──────────────────────────

    @Test
    void zeroScores_productionBaseline_significanceIsZero() {
        // Both production patients in sample data had NEWS2=0, qSOFA=0, acuity=0.0
        double sig = Module4ClinicalScoring.calculateClinicalSignificance(0, 0, 0.0);
        assertEquals(0.0, sig, 0.001,
            "Zero scores must produce 0.0 significance (real production baseline)");

        String risk = Module4ClinicalScoring.determineRiskLevel(0, 0, null);
        assertEquals("low", risk, "Zero scores must be low risk");
    }

    // ── Risk Indicators: Full Extraction Path ─────────────────────────

    @Test
    void riskIndicators_sepsisProfile_allFlagsExtracted() {
        RiskIndicators ri = new RiskIndicators();
        ri.setTachycardia(true);
        ri.setHypotension(true);
        ri.setFever(true);
        ri.setHypoxia(false);
        ri.setTachypnea(true);
        ri.setElevatedLactate(true);
        ri.setSeverelyElevatedLactate(false);
        ri.setLeukocytosis(true);

        Map<String, Object> riskData = Module4SemanticConverter.extractRiskIndicators(ri);

        // 8 boolean flags + 1 computed sepsisRisk = 9
        assertEquals(9, riskData.size());
        assertEquals(true, riskData.get("tachycardia"));
        assertEquals(true, riskData.get("hypotension"));
        assertEquals(true, riskData.get("fever"));
        assertEquals(false, riskData.get("hypoxia"));
        assertEquals(true, riskData.get("tachypnea"));
        assertEquals(true, riskData.get("elevatedLactate"));
        assertNotNull(riskData.get("sepsisRisk"), "sepsisRisk must be computed");
    }

    // ── Lab Mapping: LOINC + Standardized Name + labType ──────────────

    @Test
    void labMapping_allLoincCodes_tripleKeyed() {
        // Verify that each lab produces entries under: LOINC code, standardized name, AND labType
        Map<String, LabResult> labs = new HashMap<>();

        LabResult lactate = new LabResult();
        lactate.setValue(3.2);
        lactate.setLabType("Lactate");
        labs.put("2524-7", lactate);

        LabResult wbc = new LabResult();
        wbc.setValue(15.0);
        wbc.setLabType("WBC");
        labs.put("6690-2", wbc);

        Map<String, Object> result = Module4SemanticConverter.extractLabValues(labs);

        // Lactate: LOINC + standardized + labType
        assertEquals(3.2, result.get("2524-7"), "by LOINC code");
        assertEquals(3.2, result.get("lactate"), "by standardized name");

        // WBC: LOINC + standardized (integer) + labType
        assertEquals(15.0, result.get("6690-2"), "by LOINC code");
        assertEquals(15, result.get("wbc_count"), "by standardized name (intValue)");
        assertEquals(15.0, result.get("wbc"), "by labType (lowercase)");
    }
}
