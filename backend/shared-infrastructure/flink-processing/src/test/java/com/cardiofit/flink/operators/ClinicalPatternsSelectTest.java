package com.cardiofit.flink.operators;

import com.cardiofit.flink.builders.Module4TestBuilder;
import com.cardiofit.flink.models.PatternEvent;
import com.cardiofit.flink.models.SemanticEvent;
import com.cardiofit.flink.patterns.ClinicalPatterns;
import org.junit.jupiter.api.Test;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for ClinicalPatterns.java select functions.
 * These convert CEP pattern matches into PatternEvent objects.
 */
public class ClinicalPatternsSelectTest {

    // ── SepsisPatternSelectFunction ─────────────────────────────

    @Test
    void sepsis_withOrganDysfunction_producesCriticalSeverity() throws Exception {
        var fn = new ClinicalPatterns.SepsisPatternSelectFunction();
        Map<String, List<SemanticEvent>> matchMap = Module4TestBuilder.sepsisMatchMap("SEP-001");

        PatternEvent result = fn.select(matchMap);

        assertEquals("SEPSIS_EARLY_WARNING", result.getPatternType());
        assertEquals("SEP-001", result.getPatientId());
        // sepsisDeteriorationEvent has lactate=4.5 (>4.0), creatinine=2.5 (>2.0), platelets=80000 (<100000)
        // → hasOrganDysfunction=true → CRITICAL
        assertEquals("CRITICAL", result.getSeverity());
        assertEquals(0.92, result.getConfidence(), 0.01);
        assertEquals(3, result.getInvolvedEvents().size());
        // Surviving Sepsis Campaign 1-hour bundle
        assertTrue(result.getRecommendedActions().contains("OBTAIN_BLOOD_CULTURES"));
        assertTrue(result.getRecommendedActions().contains("ADMINISTER_BROAD_SPECTRUM_ANTIBIOTICS"));
        assertTrue(result.getRecommendedActions().contains("MEASURE_SERUM_LACTATE"));
        assertTrue(result.getRecommendedActions().contains("ADMINISTER_IV_CRYSTALLOID_30ML_KG"));
        assertNotNull(result.getPatternDetails().get("qsofa_score"));
        assertTrue((boolean) result.getPatternDetails().get("has_organ_dysfunction"));
    }

    @SuppressWarnings("unchecked")
    @Test
    void sepsis_withoutOrganDysfunction_producesHighSeverity() throws Exception {
        var fn = new ClinicalPatterns.SepsisPatternSelectFunction();
        Map<String, List<SemanticEvent>> matchMap = Module4TestBuilder.sepsisMatchMap("SEP-002");

        // Replace deterioration event with one that has NO organ dysfunction
        // but keep warning event with qSOFA >= 2 (RR=24, SBP=95 → score=2 → HIGH)
        SemanticEvent mildDeterioration = Module4TestBuilder.sepsisWarningEvent("SEP-002");
        mildDeterioration.setEventTime(System.currentTimeMillis());
        Map<String, Object> labs = (Map<String, Object>) mildDeterioration.getClinicalData().get("labValues");
        // Set lab values below organ dysfunction thresholds:
        // lactate <= 4.0, creatinine <= 2.0, platelets >= 100000
        labs.put("lactate", 3.0);
        labs.put("creatinine", 1.5);
        labs.put("platelets", 150000);
        matchMap.put("deterioration", List.of(mildDeterioration));

        PatternEvent result = fn.select(matchMap);

        // qSOFA from warning event: RR=24 (>=22→+1), SBP=95 (<=100→+1) → score=2 → HIGH
        assertEquals("HIGH", result.getSeverity());
        assertEquals(0.85, result.getConfidence(), 0.01);
        assertFalse((boolean) result.getPatternDetails().get("has_organ_dysfunction"));
    }

    // ── RapidDeteriorationPatternSelectFunction ─────────────────

    @SuppressWarnings("unchecked")
    @Test
    void rapidDeterioration_producesCriticalWithClinicalConcerns() throws Exception {
        var fn = new ClinicalPatterns.RapidDeteriorationPatternSelectFunction();
        Map<String, List<SemanticEvent>> matchMap = Module4TestBuilder.rapidDeteriorationMatchMap("RD-001");

        PatternEvent result = fn.select(matchMap);

        assertEquals("RAPID_CLINICAL_DETERIORATION", result.getPatternType());
        assertEquals("RD-001", result.getPatientId());
        assertEquals("CRITICAL", result.getSeverity());
        assertEquals(0.90, result.getConfidence(), 0.01);
        assertEquals(4, result.getInvolvedEvents().size());

        // HR baseline=80.0, HR elevated=125.0 → increase=45.0
        double hrIncrease = (double) result.getPatternDetails().get("hr_increase_bpm");
        assertTrue(hrIncrease > 40, "HR increase should be >40 bpm, got " + hrIncrease);
        assertEquals(45.0, hrIncrease, 0.1);

        // O2sat from o2satDecreased event = 85.0
        double o2sat = (double) result.getPatternDetails().get("oxygen_saturation");
        assertEquals(85.0, o2sat, 0.1);

        // RR from rrElevated event = 30.0
        double rr = (double) result.getPatternDetails().get("respiratory_rate");
        assertEquals(30.0, rr, 0.1);

        // Clinical concerns: HR increase > 30 → SEVERE_TACHYCARDIA, RR > 28 → SEVERE_TACHYPNEA, O2 < 88 → SEVERE_HYPOXEMIA
        List<String> concerns = (List<String>) result.getPatternDetails().get("clinical_concerns");
        assertTrue(concerns.contains("SEVERE_TACHYCARDIA"));
        assertTrue(concerns.contains("SEVERE_TACHYPNEA"));
        assertTrue(concerns.contains("SEVERE_HYPOXEMIA"));

        assertTrue(result.getRecommendedActions().contains("IMMEDIATE_PHYSICIAN_NOTIFICATION"));
        assertTrue(result.getRecommendedActions().contains("PREPARE_FOR_POTENTIAL_ICU_TRANSFER"));
        assertTrue(result.getRecommendedActions().contains("OBTAIN_ABG_ARTERIAL_BLOOD_GAS"));
        assertTrue(result.getRecommendedActions().contains("CONTINUOUS_CARDIAC_MONITORING"));
    }

    // ── DrugLabMonitoringPatternSelectFunction ──────────────────

    @Test
    void drugLabMonitoring_warfarin_producesAnticoagulantMonitoring() throws Exception {
        var fn = new ClinicalPatterns.DrugLabMonitoringPatternSelectFunction();
        // The builder stores medication under "medications" key, but getMedicationName()
        // reads from "medicationData". Build the match map with corrected key.
        Map<String, List<SemanticEvent>> matchMap = buildDrugLabMatchMap("DL-001", "Warfarin 5mg");

        PatternEvent result = fn.select(matchMap);

        assertEquals("DRUG_LAB_MONITORING_COMPLIANCE", result.getPatternType());
        assertEquals("DL-001", result.getPatientId());
        assertEquals("MODERATE", result.getSeverity());
        assertEquals(0.88, result.getConfidence(), 0.01);
        assertEquals("ANTICOAGULANT", result.getPatternDetails().get("drug_class"));
        assertEquals("MISSING_LABS", result.getPatternDetails().get("compliance_status"));
        assertEquals("Monitor for bleeding risk and therapeutic range",
            result.getPatternDetails().get("monitoring_rationale"));
        // Anticoagulant branch adds CHECK_INR_PT_PTT
        assertTrue(result.getRecommendedActions().contains("CHECK_INR_PT_PTT"));
        assertTrue(result.getRecommendedActions().contains("ORDER_REQUIRED_LABS"));
        assertTrue(result.getRecommendedActions().contains("PHARMACIST_REVIEW"));
    }

    @Test
    void drugLabMonitoring_gentamicin_producesNephrotoxicMonitoring() throws Exception {
        var fn = new ClinicalPatterns.DrugLabMonitoringPatternSelectFunction();
        Map<String, List<SemanticEvent>> matchMap = buildDrugLabMatchMap("DL-002", "Gentamicin 80mg IV");

        PatternEvent result = fn.select(matchMap);

        assertEquals("NEPHROTOXIC_ANTIBIOTIC", result.getPatternDetails().get("drug_class"));
        assertEquals("Monitor for nephrotoxicity", result.getPatternDetails().get("monitoring_rationale"));
        // Nephrotoxic antibiotic does not match ACE_INHIBITOR/ANTICOAGULANT/CARDIAC_GLYCOSIDE/MOOD_STABILIZER
        // so no specific lab-check action beyond the defaults
        assertTrue(result.getRecommendedActions().contains("ORDER_REQUIRED_LABS"));
        assertTrue(result.getRecommendedActions().contains("REVIEW_MEDICATION_INDICATION"));
    }

    // ── SepsisPathwayCompliancePatternSelectFunction ─────────────

    @Test
    void sepsisPathway_fullCompliance_producesLowSeverity() throws Exception {
        var fn = new ClinicalPatterns.SepsisPathwayCompliancePatternSelectFunction();
        // 30 min to cultures, 45 min to antibiotics → both <= 60 → fullCompliance
        Map<String, List<SemanticEvent>> matchMap =
            Module4TestBuilder.sepsisPathwayMatchMap("SP-001", 30.0, 45.0);

        PatternEvent result = fn.select(matchMap);

        assertEquals("SEPSIS_PATHWAY_COMPLIANCE", result.getPatternType());
        assertEquals("LOW", result.getSeverity());
        assertEquals(0.95, result.getConfidence(), 0.01);
        assertEquals("COMPLIANT", result.getPatternDetails().get("compliance_status"));
        assertEquals(3, result.getInvolvedEvents().size());
        assertTrue(result.getRecommendedActions().contains("CONTINUE_SEPSIS_PROTOCOL"));
        assertTrue(result.getRecommendedActions().contains("MEASURE_LACTATE_CLEARANCE"));
        assertTrue(result.getRecommendedActions().contains("ASSESS_SOURCE_CONTROL"));
    }

    @Test
    void sepsisPathway_nonCompliant_producesHighSeverity() throws Exception {
        var fn = new ClinicalPatterns.SepsisPathwayCompliancePatternSelectFunction();
        // 120 min to cultures, 150 min to antibiotics → both > 90 → neither fullCompliance nor partialCompliance → HIGH
        Map<String, List<SemanticEvent>> matchMap =
            Module4TestBuilder.sepsisPathwayMatchMap("SP-002", 120.0, 150.0);

        PatternEvent result = fn.select(matchMap);

        assertEquals("HIGH", result.getSeverity());
        assertEquals(0.92, result.getConfidence(), 0.01);
        assertEquals("NON_COMPLIANT", result.getPatternDetails().get("compliance_status"));
        assertTrue(result.getRecommendedActions().contains("REVIEW_BUNDLE_COMPLIANCE_BARRIERS"));
        assertTrue(result.getRecommendedActions().contains("IMPLEMENT_PROCESS_IMPROVEMENTS"));
        assertTrue(result.getRecommendedActions().contains("STAFF_EDUCATION_SEPSIS_BUNDLE"));
    }

    @Test
    void sepsisPathway_partialCompliance_producesModerate() throws Exception {
        var fn = new ClinicalPatterns.SepsisPathwayCompliancePatternSelectFunction();
        // 45 min to cultures (<=60→ok), 85 min to antibiotics (>60→miss, but <=90→partial)
        // fullCompliance = (45<=60 && 85<=60) = false
        // partialCompliance = (45<=90 || 85<=90) = true → MODERATE
        Map<String, List<SemanticEvent>> matchMap =
            Module4TestBuilder.sepsisPathwayMatchMap("SP-003", 45.0, 85.0);

        PatternEvent result = fn.select(matchMap);

        assertEquals("MODERATE", result.getSeverity());
        assertEquals(0.88, result.getConfidence(), 0.01);
        // Not full compliance → NON_COMPLIANT status
        assertEquals("NON_COMPLIANT", result.getPatternDetails().get("compliance_status"));
    }

    // ── Helper: build drug-lab match map with correct "medicationData" key ──

    /**
     * Build a drug-lab monitoring match map that uses the "medicationData" key
     * expected by ClinicalPatterns.getMedicationName().
     */
    private static Map<String, List<SemanticEvent>> buildDrugLabMatchMap(String patientId, String medicationName) {
        Map<String, List<SemanticEvent>> map = new HashMap<>();

        SemanticEvent medEvent = Module4TestBuilder.baselineVitalEvent(patientId);
        medEvent.setEventType(com.cardiofit.flink.models.EventType.MEDICATION_ORDERED);
        Map<String, Object> medData = new HashMap<>();
        medData.put("medication_name", medicationName);
        medEvent.getClinicalData().put("medicationData", medData);

        map.put("high_risk_med_started", List.of(medEvent));
        return map;
    }
}
