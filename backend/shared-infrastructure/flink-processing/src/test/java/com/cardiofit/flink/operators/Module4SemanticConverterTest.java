package com.cardiofit.flink.operators;

import com.cardiofit.flink.models.LabResult;
import com.cardiofit.flink.models.RiskIndicators;
import org.junit.jupiter.api.Test;

import java.util.HashMap;
import java.util.Map;

import static org.junit.jupiter.api.Assertions.*;

/**
 * Tests for Module4SemanticConverter helper methods.
 * Covers vital sign normalization, lab value extraction, and risk indicator extraction.
 */
public class Module4SemanticConverterTest {

    // ── Vital Sign Normalization ─────────────────────────────────

    @Test
    void normalizeVitalSigns_allKeys_mapped() {
        Map<String, Object> vitals = new HashMap<>();
        vitals.put("heartrate", 88);
        vitals.put("systolicbp", 120);
        vitals.put("diastolicbp", 80);
        vitals.put("respiratoryrate", 18);
        vitals.put("temperature", 37.2);
        vitals.put("oxygensaturation", 97);

        Map<String, Object> result = Module4SemanticConverter.normalizeVitalSigns(vitals);

        assertEquals(6, result.size());
        assertEquals(88, result.get("heart_rate"));
        assertEquals(120, result.get("systolic_bp"));
        assertEquals(80, result.get("diastolic_bp"));
        assertEquals(18, result.get("respiratory_rate"));
        assertEquals(37.2, result.get("temperature"));
        assertEquals(97, result.get("oxygen_saturation"));
    }

    @Test
    void normalizeVitalSigns_partialKeys_onlyPresent() {
        Map<String, Object> vitals = new HashMap<>();
        vitals.put("heartrate", 72);
        vitals.put("temperature", 36.8);

        Map<String, Object> result = Module4SemanticConverter.normalizeVitalSigns(vitals);

        assertEquals(2, result.size());
        assertEquals(72, result.get("heart_rate"));
        assertEquals(36.8, result.get("temperature"));
        assertNull(result.get("systolic_bp"));
    }

    @Test
    void normalizeVitalSigns_null_returnsEmpty() {
        Map<String, Object> result = Module4SemanticConverter.normalizeVitalSigns(null);
        assertTrue(result.isEmpty());
    }

    @Test
    void normalizeVitalSigns_empty_returnsEmpty() {
        Map<String, Object> result = Module4SemanticConverter.normalizeVitalSigns(new HashMap<>());
        assertTrue(result.isEmpty());
    }

    // ── Vital Sign Normalization — Production Key Format ─────────

    @Test
    void normalizeVitalSigns_productionKeys_systolicbloodpressure() {
        // Production PatientContextAggregator stores "systolicbloodpressure" not "systolicbp"
        Map<String, Object> vitals = new HashMap<>();
        vitals.put("heartrate", 68);
        vitals.put("systolicbloodpressure", 120);
        vitals.put("diastolicbloodpressure", 78);
        vitals.put("respiratoryrate", 16);
        vitals.put("temperature", 37.0);
        vitals.put("oxygensaturation", 96);

        Map<String, Object> result = Module4SemanticConverter.normalizeVitalSigns(vitals);

        assertEquals(6, result.size(), "All 6 vitals should be mapped from production keys");
        assertEquals(120, result.get("systolic_bp"),
            "systolicbloodpressure must map to systolic_bp");
        assertEquals(78, result.get("diastolic_bp"),
            "diastolicbloodpressure must map to diastolic_bp");
        assertEquals(68, result.get("heart_rate"));
    }

    @Test
    void normalizeVitalSigns_shortKeysPreferredOverLong_whenBothPresent() {
        // Edge case: if both key formats exist, systolicbloodpressure takes precedence
        // (checked first) since it's the production format
        Map<String, Object> vitals = new HashMap<>();
        vitals.put("systolicbloodpressure", 142);
        vitals.put("systolicbp", 130); // should be ignored

        Map<String, Object> result = Module4SemanticConverter.normalizeVitalSigns(vitals);

        assertEquals(142, result.get("systolic_bp"),
            "Production key (systolicbloodpressure) should take precedence");
    }

    @Test
    void normalizeVitalSigns_demographicsExcluded() {
        // Production data includes age and gender inside latestVitals
        // These are demographics, NOT vital signs — must be excluded
        Map<String, Object> vitals = new HashMap<>();
        vitals.put("heartrate", 82);
        vitals.put("systolicbloodpressure", 142);
        vitals.put("diastolicbloodpressure", 92);
        vitals.put("age", 50);
        vitals.put("gender", "female");

        Map<String, Object> result = Module4SemanticConverter.normalizeVitalSigns(vitals);

        assertNull(result.get("age"), "age must not be in normalized vitals");
        assertNull(result.get("gender"), "gender must not be in normalized vitals");
        assertEquals(3, result.size(), "Only HR, SBP, DBP should be mapped (no demographics)");
    }

    // ── Lab Value Extraction ─────────────────────────────────────

    @Test
    void extractLabValues_loincCodes_mappedToStandardNames() {
        Map<String, LabResult> labs = new HashMap<>();

        LabResult lactate = new LabResult();
        lactate.setValue(2.5);
        lactate.setLabType("Lactate");
        labs.put("2524-7", lactate);

        LabResult creatinine = new LabResult();
        creatinine.setValue(1.2);
        creatinine.setLabType("Creatinine");
        labs.put("2160-0", creatinine);

        LabResult wbc = new LabResult();
        wbc.setValue(12.5);
        wbc.setLabType("WBC");
        labs.put("6690-2", wbc);

        Map<String, Object> result = Module4SemanticConverter.extractLabValues(labs);

        // LOINC code keys
        assertEquals(2.5, result.get("2524-7"));
        assertEquals(1.2, result.get("2160-0"));
        assertEquals(12.5, result.get("6690-2"));

        // Standardized name keys
        assertEquals(2.5, result.get("lactate"));
        assertEquals(1.2, result.get("creatinine"));
        assertEquals(12, result.get("wbc_count")); // intValue()

        // labType keys (lowercase)
        assertEquals(12.5, result.get("wbc"));
    }

    @Test
    void extractLabValues_procalcitoninAndPlatelets_mapped() {
        Map<String, LabResult> labs = new HashMap<>();

        LabResult pct = new LabResult();
        pct.setValue(0.8);
        labs.put("33959-8", pct);

        LabResult platelets = new LabResult();
        platelets.setValue(150000.0);
        labs.put("777-3", platelets);

        Map<String, Object> result = Module4SemanticConverter.extractLabValues(labs);

        assertEquals(0.8, result.get("procalcitonin"));
        assertEquals(0.8, result.get("33959-8"));
        assertEquals(150000, result.get("platelet_count")); // intValue()
        assertEquals(150000.0, result.get("777-3"));
    }

    @Test
    void extractLabValues_null_returnsEmpty() {
        Map<String, Object> result = Module4SemanticConverter.extractLabValues(null);
        assertTrue(result.isEmpty());
    }

    @Test
    void extractLabValues_nullValue_skipped() {
        Map<String, LabResult> labs = new HashMap<>();

        LabResult nullValueLab = new LabResult();
        nullValueLab.setValue(null);
        nullValueLab.setLabType("Glucose");
        labs.put("2345-7", nullValueLab);

        LabResult validLab = new LabResult();
        validLab.setValue(5.0);
        validLab.setLabType("Lactate");
        labs.put("2524-7", validLab);

        Map<String, Object> result = Module4SemanticConverter.extractLabValues(labs);

        // Null-value lab skipped entirely
        assertNull(result.get("2345-7"));
        assertNull(result.get("glucose"));

        // Valid lab present
        assertEquals(5.0, result.get("lactate"));
        assertEquals(5.0, result.get("2524-7"));
    }

    // ── Risk Indicator Extraction ────────────────────────────────

    @Test
    void extractRiskIndicators_allFlags_mapped() {
        RiskIndicators ri = new RiskIndicators();
        ri.setTachycardia(true);
        ri.setHypotension(true);
        ri.setFever(false);
        ri.setHypoxia(true);
        ri.setTachypnea(false);
        ri.setElevatedLactate(true);
        ri.setSeverelyElevatedLactate(false);
        ri.setLeukocytosis(true);

        Map<String, Object> result = Module4SemanticConverter.extractRiskIndicators(ri);

        assertEquals(9, result.size());
        assertEquals(true, result.get("tachycardia"));
        assertEquals(true, result.get("hypotension"));
        assertEquals(false, result.get("fever"));
        assertEquals(true, result.get("hypoxia"));
        assertEquals(false, result.get("tachypnea"));
        assertEquals(true, result.get("elevatedLactate"));
        assertEquals(false, result.get("severelyElevatedLactate"));
        assertEquals(true, result.get("leukocytosis"));
        // sepsisRisk is computed from other flags
        assertNotNull(result.get("sepsisRisk"));
    }

    @Test
    void extractRiskIndicators_null_returnsEmpty() {
        Map<String, Object> result = Module4SemanticConverter.extractRiskIndicators(null);
        assertTrue(result.isEmpty());
    }
}
